/**
 * The model-load wait: how long the page waits for a swap.
 *
 * Until 1.7.8 the server gave a load three minutes (swapTimeout in
 * internal/supervisor) and the page waited exactly three minutes too
 * (pollSwapStatus(180000), three times in js/02-model.js). A big model off a
 * slow disk could not finish, and when the two clocks ran out together the page
 * often won and showed "Timed out" instead of the server's reason.
 *
 * Now the server's limit is model_load_timeout_seconds in config.toml, reported
 * in every /swap-status as load_timeout_seconds, and the page waits that long
 * plus a margin for stopping the old engine. The page's own clock is only a
 * backstop for a server that stops answering.
 *
 * Runs the real pollSwapStatus on a fake clock.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const MODEL = fs.readFileSync(ROOT + '/js/02-model.js', 'utf8');
const GO = fs.readFileSync(ROOT + '/internal/supervisor/supervisor.go', 'utf8');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra !== undefined ? '\n      ' + String(extra).slice(0, 500) : '')); }
};
const between = (src, from, to) => {
  const a = src.indexOf(from), b = src.indexOf(to, a + 1);
  if (a < 0 || b < 0) { console.error('could not locate ' + JSON.stringify(from)); process.exit(1); }
  return src.slice(a, b);
};
const POLL = between(MODEL, '/* How long the page waits for a swap.', '/**\n * Show the header toast');

/* The page's poll against a server that answers `reply(n, now)` to its n-th
   /swap-status. Time is fake: every setTimeout jumps the clock. */
function run(reply, timeoutMs) {
  let now = 0, polls = 0;
  const c = {
    Math, Number, Promise,
    Date: { now: () => now },
    setTimeout: (fn, ms) => { now += ms; Promise.resolve().then(fn); return 0; },
    fetch: async (url) => {
      polls++;
      const st = reply(polls, now);
      if (st === 'down') throw new TypeError('Failed to fetch');
      return { ok: true, json: async () => st };
    },
  };
  vm.createContext(c);
  vm.runInContext(POLL, c);
  return c.pollSwapStatus(timeoutMs).then(st => ({ st, at: now, polls }));
}

console.log('\n=== the server says how long a load may take ===');
{
  const starting = (load) => () => ({ phase: 'starting', load_timeout_seconds: load });

  let r = await run(starting(600));
  ok(r.st.phase === 'error' && r.at >= 630000 && r.at < 633000, 'model_load_timeout_seconds = 600: the page waits 630 s (600 + 30 for stopping the old engine)', r.at);
  ok(/Timed out after 630 s/.test(r.st.message), 'and says how long it waited', r.st.message);

  r = await run(starting(60));
  ok(r.at >= 90000 && r.at < 93000, 'a shorter limit is honoured too: 60 s → 90 s', r.at);

  r = await run(() => ({ phase: 'starting' }));
  ok(r.at >= 210000 && r.at < 213000, 'a server that does not say (legacy, or before 1.7.8): 180 + 30 s', r.at);

  // The point of the margin: the server's verdict always arrives first.
  r = await run((n, now) => now >= 600000 + 20000
    ? { phase: 'error', message: 'model did not finish loading within 10m0s. … raise model_load_timeout_seconds', load_timeout_seconds: 600 }
    : { phase: 'starting', load_timeout_seconds: 600 });
  ok(r.st.phase === 'error' && /model_load_timeout_seconds/.test(r.st.message), "a server that times out at 600 s (after stopping the old engine) is heard: its reason, not the page's", r.st.message);

  r = await run((n) => n < 4 ? { phase: 'starting', load_timeout_seconds: 600 } : { phase: 'ready', file: 'big.gguf' });
  ok(r.st.phase === 'ready' && r.polls === 4, 'ready ends the wait at once');

  r = await run((n) => n === 1 ? { phase: 'starting', load_timeout_seconds: 600 } : 'down');
  ok(r.at >= 630000 && r.at < 633000 && /Timed out/.test(r.st.message), 'a server that stops answering mid-load: the page keeps the budget it was told, then gives up', r.at);

  r = await run(starting(600), 5000);
  ok(r.at >= 5000 && r.at < 8000, 'an explicit budget from a caller still wins');
}

console.log('\n=== no copy of the number left behind ===');
{
  ok(!/pollSwapStatus\(\d/.test(MODEL), 'no caller passes its own number any more (three used to pass 180000)');
  ok((MODEL.match(/await pollSwapStatus\(\)/g) || []).length === 3, 'the header swap, the lore model borrow and the Performance restart all use the server\'s limit');
  ok(/LoadTimeoutSeconds int `json:"load_timeout_seconds,omitempty"`/.test(GO) && /st\.LoadTimeoutSeconds = int\(s\.loadTimeout\(\) \/ time\.Second\)/.test(GO),
     '/swap-status reports the limit the supervisor is using');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
