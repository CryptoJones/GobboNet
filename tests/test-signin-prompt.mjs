/**
 * The sign-in prompt asks only when the tab really is signed out, and goes
 * away by itself once the browser is signed in again.
 *
 * Reported against 1.7.8: "sign in, tab out, immediately get asked to sign
 * back in, including on dev tool use ... a little aggressive". One 401 from
 * any state or reply request locked the tab and opened SIGNED OUT, and the
 * prompt then stayed up even after the browser had signed in again elsewhere
 * -- the session cookie belongs to the browser, not the tab.
 *
 * Runs the real code out of js/06-state-sync.js in Node's vm:
 *   A. a 401 is checked once more before the prompt opens
 *   B. a locked tab looks again on return and while on screen, and carries on
 *   C. the wiring: wake handler, reply polls, the probe route, the prompt text
 * The server half (the session binding, and why a session was refused) is in
 * internal/auth/session_test.go and internal/server/authlog_test.go.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const read = (p) => fs.readFileSync(ROOT + '/' + p, 'utf8');
const SYNC = read('js/06-state-sync.js');
const BOOT = read('js/24-boot.js');
const GEN = read('js/03-generation.js');
const HTML = read('chat.html');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};
const between = (src, from, to) => {
  const a = src.indexOf(from), b = src.indexOf(to, a + 1);
  if (a < 0 || b < 0) { console.error('could not locate ' + JSON.stringify(from)); process.exit(1); }
  return src.slice(a, b);
};
const tick = () => new Promise(r => setTimeout(r, 0));
const settle = async () => { for (let i = 0; i < 6; i++) await tick(); };

function load({ probe = 200, sync = true } = {}) {
  const calls = [], logs = [];
  const modal = { open: false, classList: { add() { modal.open = true; }, remove() { modal.open = false; } } };
  const els = { 'unlock-modal': modal, 'unlock-error': { style: {} }, 'unlock-password': { value: '', focus() {} } };
  const timers = [];
  const ctx = {
    console: { log: (...a) => logs.push(['log', a.join(' ')]), warn: (...a) => logs.push(['warn', a.join(' ')]), error() {} },
    location: { href: 'http://127.0.0.1:9066/', origin: 'http://127.0.0.1:9066' },
    URL, Date, Promise, JSON, Error,
    document: { visibilityState: 'visible', getElementById: (id) => els[id] || null },
    setTimeout: (f) => { timers.push(f); return timers.length; }, clearTimeout() {},
    setInterval: (f) => { ctx._interval = f; return 99; }, clearInterval: () => { ctx._interval = null; },
    STATE_SYNC_AVAILABLE: sync,
    stateSyncUrl: (suffix) => 'http://127.0.0.1:9066/state' + (suffix || ''),
    stateSync: { status: 'idle', lastError: null, pushTimer: 7 },
    updateSyncIndicator() {},
    resumed: 0,
    resumeAfterUnlock() { ctx.resumed++; ctx.stateSync.status = 'idle'; },
  };
  ctx.fetch = async (url) => {
    calls.push(url);
    const status = typeof ctx.probe === 'function' ? ctx.probe(url) : ctx.probe;
    if (status === 'down') throw new TypeError('Failed to fetch');
    return { status, ok: status >= 200 && status < 300 };
  };
  ctx.probe = probe;
  vm.createContext(ctx);
  const block = between(SYNC, 'let _lockCheck = null;', '/**\n * Sign back in without leaving the page.');
  vm.runInContext(block + '\n;globalThis.__lockCheck = () => _lockCheck;', ctx);
  return { ctx, calls, logs, modal };
}
const r401 = (url) => ({ status: 401, url });

console.log('\nA. One 401 is checked once more before asking');
{
  const { ctx, calls, logs, modal } = load({ probe: 200 });
  ok(ctx.noteLocked(r401('http://127.0.0.1:9066/state/index')) === true, 'the caller is told to stop this attempt');
  await settle();
  ok(calls.length === 1 && calls[0] === 'http://127.0.0.1:9066/state/info', 'one fresh check, on a state route (session and, encrypted, the key)', calls);
  ok(!modal.open && ctx.stateSync.status !== 'locked', 'still signed in: no prompt, not locked');
  ok(ctx.stateSync.pushTimer === 7, 'and the pending push timer is left to retry');
  ok(logs.some(([k, m]) => k === 'warn' && /\/state\/index was refused \(401\), but this tab is still signed in/.test(m)),
    'the console says which request was refused and that it did not ask', JSON.stringify(logs));
}
{
  const { ctx, logs, modal } = load({ probe: 401 });
  ctx.noteLocked(r401('http://127.0.0.1:9066/state/abc'));
  await settle();
  ok(modal.open && ctx.stateSync.status === 'locked', 'refused again: the tab locks and asks');
  ok(ctx.stateSync.pushTimer === null, 'and the push retry is cancelled until sign-in, as before');
  ok(logs.some(([k, m]) => k === 'warn' && /signed out: \/state\/abc was refused \(401\), and so was a fresh check/.test(m)),
    'the console names the request and the second refusal', JSON.stringify(logs));
  ok(ctx.noteLocked(r401('/x')) === true && ctx.__lockCheck() === null, 'already locked: no further checks');
}
{
  const { ctx, modal } = load({ probe: 'down' });
  ctx.noteLocked(r401('http://127.0.0.1:9066/state/abc'));
  await settle();
  ok(modal.open && ctx.stateSync.status === 'locked', 'the check cannot reach the server: asks, as it always did');
}
{
  const { ctx, calls } = load({ probe: 401 });
  ctx.noteLocked(r401('/state/a')); ctx.noteLocked(r401('/state/b')); ctx.noteLocked(r401('/llm/jobs/1'));
  await settle();
  ok(calls.length === 1, 'a burst of 401s makes one check, not one each', calls.length);
}
{
  const { ctx, calls } = load({ probe: 200, sync: false });
  ctx.noteLocked(r401('/llm/jobs/1')); await settle();
  ok(calls[0] === '/health-fileserver', 'with no state routes (file server only), the check is /health-fileserver', calls);
  ok(ctx.noteLocked({ status: 500 }) === false && ctx.noteLocked(null) === false, 'anything but a 401 is not "signed out"');
}

console.log('\nB. A locked tab carries on once the browser is signed in again');
{
  const { ctx, calls, logs, modal } = load({ probe: 401 });
  ctx.noteLocked(r401('/state/abc')); await settle();
  ok(modal.open && typeof ctx._interval === 'function', 'while the prompt is up, a periodic look is scheduled');
  ok(await ctx.recheckLockedSession('focus') === false && modal.open, 'still signed out: the prompt stays');
  ctx.probe = 200;
  ok(await ctx.recheckLockedSession('focus') === false, 'a second look within 1.5 s is skipped (the wake signals arrive together)');
  ctx._lockRecheckAt = 0; vm.runInContext('_lockRecheckAt = 0', ctx);
  ctx.document.visibilityState = 'hidden';
  ok(await ctx.recheckLockedSession('poll') === false && calls.length === 2, 'a hidden tab does not look', calls.length);
  ctx.document.visibilityState = 'visible';
  ok(await ctx.recheckLockedSession('visibility') === true, 'on screen again and the browser is signed in: carries on');
  ok(!modal.open && ctx.resumed === 1, 'the prompt closes and sync resumes (resumeAfterUnlock)');
  ok(ctx._interval == null, 'and the periodic look stops');
  ok(logs.some(([k, m]) => k === 'log' && /signed in again \(visibility\)/.test(m)), 'the console says so');
  ok(await ctx.recheckLockedSession('focus') === false, 'not locked: nothing to do');
}
{
  const { ctx, modal } = load({ probe: 401 });
  ctx.noteLocked(r401('/state/abc')); await settle();
  ctx.probe = 200;
  await ctx._interval(); await settle();
  ok(!modal.open && ctx.resumed === 1, 'the periodic look alone is enough (a tab beside the one that signed in)');
}

console.log('\nC. The wiring');
{
  const wake = between(BOOT, 'function handleAppWake(reason) {', '\n}\n');
  const recheckAt = wake.indexOf('recheckLockedSession(reason)'), gateAt = wake.indexOf('if (!_appBooted) return;');
  ok(recheckAt > 0 && gateAt > recheckAt, 'every return to the tab (focus, visibility, pageshow, online) looks again, even before boot finishes');
  ok(/noteLocked\(\{ status: 401, url: JOBS_URL \+ '\/' \+ jobId \}\)/.test(GEN), 'a refused reply poll names its request');
  ok(/Signed in again in another tab\? This one notices and carries on by itself\./.test(HTML), 'the prompt says it can carry on by itself');
  ok(/function serverError\(resp, message\) \{\n  noteLocked\(resp\);/.test(SYNC), 'every state-route error still goes through noteLocked');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
