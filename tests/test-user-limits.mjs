/**
 * Limits the user sets, not the code — roadmap 1.7.7 items #6a, #6b, #6c, #4.
 *
 * Four numbers that were constants are now the user's:
 *
 *   A. #6a  the per-character reply cap had a silent 8,192 ceiling; it is gone
 *           (the floor of 25 stays — below it the sentence-end cut has nothing
 *           to work with).
 *   B. #6b  the pause between auto-continued posts (20 s) and the most posts
 *           one {{auto_continue_N}} may run (50) are CONFIG settings.
 *   C. #6c  how long one lore compression pass may take (45 s) is a setting.
 *   D. #4   the server's fixed 30-minute stop is gone; the CONFIG runtime
 *           limit is sent with every job so the server enforces the user's
 *           limit instead, and off means no limit.
 *
 * The rule each reader keeps: the old value is the default, so nothing changes
 * for anyone who leaves it alone, and a missing or nonsense stored value falls
 * back to that default — never to 0, which for these would mean "instantly"
 * or "never".
 *
 * Every section drives the real code out of js/, not a copy.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const read = (p) => fs.readFileSync(ROOT + '/' + p, 'utf8');
const STATE = read('js/04-state.js');
const PROMPT = read('js/07-prompt.js');
const THREADS = read('js/09-threads.js');
const CHAT = read('js/10-chat.js');
const GEN = read('js/03-generation.js');
const CARDS = read('js/15-cards.js');
const HTML = read('chat.html');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};
const eq = (a, b, l) => ok(a === b, l,
  a === b ? '' : `got:  ${JSON.stringify(a)}\n      want: ${JSON.stringify(b)}`);

const between = (src, from, to, file) => {
  const a = src.indexOf(from);
  const b = to ? src.indexOf(to, a + 1) : src.length;
  if (a < 0 || b < 0) { console.error(`could not locate ${JSON.stringify(from)} in ${file}`); process.exit(1); }
  return src.slice(a, b);
};
const fnBody = (src, sig, file) => {
  const a = src.indexOf(sig);
  if (a < 0) { console.error(`could not locate ${sig} in ${file}`); process.exit(1); }
  return src.slice(a, src.indexOf('\n}\n', a) + 2);
};

// DEFAULT_SETTINGS, evaluated for real.
const DEFAULTS = (() => {
  const block = between(STATE, 'const DEFAULT_SETTINGS = {', '\n};', 'js/04-state.js') + '\n};';
  const ctx = {}; vm.createContext(ctx);
  vm.runInContext(block.replace('const DEFAULT_SETTINGS', 'globalThis.DEFAULT_SETTINGS'), ctx);
  return ctx.DEFAULT_SETTINGS;
})();

/* ================================================================
   A. #6a — THE REPLY CAP HAS NO CEILING
================================================================ */
console.log('\n=== A. the reply cap is the number the user typed ===');
{
  const ctx = { state: { settings: {} } }; vm.createContext(ctx);
  vm.runInContext(fnBody(STATE, 'function resolveSmartLimit(card)', 'js/04-state.js'), ctx);
  const cap = (tokens) => ctx.resolveSmartLimit({ smartLimitEnabled: true, smartLimitTokens: tokens }).tokens;
  eq(cap(20000), 20000, '20,000 stays 20,000 (was silently 8,192)');
  eq(cap(8193), 8193, 'one past the old ceiling is kept');
  eq(cap(300), 300, 'an ordinary value is untouched');
  eq(cap(10), 25, 'the floor of 25 still holds');
  ok(!/Math\.min\(\s*8192/.test(CARDS), 'saving a card no longer clamps to 8192 (js/15-cards.js)');
  const input = /<input[^>]*id="card-smart-limit-tokens"[^>]*>/.exec(HTML);
  ok(!!input && !/\bmax=/.test(input[0]) && /\bmin="25"/.test(input[0]),
     'the card editor field keeps min="25" and has no max', input && input[0]);
}

/* ================================================================
   B. #6b — AUTO-CONTINUE PAUSE AND CHAIN CAP
================================================================ */
console.log('\n=== B. auto-continue: the pause and the chain cap are settings ===');
eq(DEFAULTS.autoContinueDelaySeconds, 20, 'default pause is the old 20 s');
eq(DEFAULTS.autoContinueMaxChain, 50, 'default chain cap is the old 50');
{
  // The whole AUTO-CONTINUE section up to the indicator renderer: constants,
  // readers, the macro parser and the chain stepper.
  const SECTION = between(THREADS, 'const AUTO_CONTINUE_TEXT', 'function renderAutoContinueIndicator', 'js/09-threads.js');
  function build(settings) {
    const timers = [];
    const sent = [];
    const ctx = {
      console: { log() {}, warn() {}, error() {} },
      state: { settings },
      getActiveThread: () => ({ id: 't1' }),
      renderAutoContinueIndicator: () => {},
      sendMessage: (m) => sent.push(m),
      setTimeout: (fn, ms) => { timers.push({ fn, ms }); return timers.length; },
      clearTimeout: () => {}, setInterval: () => 0, clearInterval: () => {},
      Math, isFinite, parseInt,
    };
    vm.createContext(ctx);
    vm.runInContext(SECTION + '\n;globalThis.__ac = () => autoContinue;', ctx);
    return { ctx, timers, sent };
  }
  const delay = (s) => build(s).ctx.autoContinueDelayMs();
  const chain = (s) => build(s).ctx.autoContinueMaxChain();
  eq(delay({}), 20000, 'no stored value → 20 s');
  eq(delay({ autoContinueDelaySeconds: 5 }), 5000, '5 s → 5000 ms');
  eq(delay({ autoContinueDelaySeconds: 0 }), 0, '0 s is honoured — no pause is a real choice');
  eq(delay({ autoContinueDelaySeconds: '' }), 20000, 'a cleared field falls back to 20 s, not 0');
  eq(delay({ autoContinueDelaySeconds: -3 }), 20000, 'a negative value falls back to 20 s');
  eq(chain({}), 50, 'no stored value → 50 posts');
  eq(chain({ autoContinueMaxChain: 200 }), 200, '200 is allowed — the old ceiling was the constant');
  eq(chain({ autoContinueMaxChain: 0 }), 50, '0 falls back to 50 — the post being sent is always the first');

  // The macro parser obeys the setting.
  eq(build({}).ctx.detectAutoContinueMacro('go {{auto_continue_120}}').count, 50,
     '{{auto_continue_120}} is capped at the default 50');
  eq(build({ autoContinueMaxChain: 200 }).ctx.detectAutoContinueMacro('go {{auto_continue_120}}').count, 120,
     'and runs all 120 when the cap is raised to 200');

  // The chain stepper waits the configured time, and the countdown agrees.
  const t = build({ autoContinueDelaySeconds: 7 });
  t.ctx.startAutoContinueChain('t1', 3);
  t.ctx.maybeContinueChain();
  eq(t.timers.length, 1, 'one send is scheduled');
  eq(t.timers[0] && t.timers[0].ms, 7000, 'it is scheduled 7 s out — the setting, not 20 s');
  eq(t.ctx.__ac().secsLeft, 7, 'and the countdown starts at 7');
  t.timers[0].fn();
  eq(t.sent.join(), '{{auto_continue}}', 'when it fires, the next post is sent');
}

/* ================================================================
   C. #6c — LORE COMPRESSION TIMEOUT
================================================================ */
console.log('\n=== C. the lore compression timeout is a setting ===');
eq(DEFAULTS.loreTimeoutSeconds, 45, 'default is the old 45 s');
{
  const reader = between(PROMPT, 'const LORE_TIMEOUT_MS', 'async function getServerContextSize', 'js/07-prompt.js');
  const lore = (settings) => {
    const ctx = { state: { settings }, Math, isFinite, parseInt }; vm.createContext(ctx);
    vm.runInContext(reader.replace(/\nlet _loreServerCtx[\s\S]*$/, ''), ctx);
    return ctx.loreTimeoutMs();
  };
  eq(lore({}), 45000, 'no stored value → 45 s');
  eq(lore({ loreTimeoutSeconds: 180 }), 180000, '180 s for a slow CPU-only machine');
  eq(lore({ loreTimeoutSeconds: 2 }), 5000, 'below the 5 s floor is raised to it');
  eq(lore({ loreTimeoutSeconds: 'abc' }), 45000, 'nonsense falls back to 45 s');
  const loreSrc = fnBody(PROMPT, 'async function summarizeForLore(', 'js/07-prompt.js');
  ok(/setTimeout\(\(\) => _loreAbort\.abort\(\), _loreTimeoutMs\)/.test(loreSrc)
     && /const _loreTimeoutMs = loreTimeoutMs\(\)/.test(loreSrc),
     'each compression pass is cancelled at the configured time');
  ok(/timed out after ' \+ Math\.round\(_loreTimeoutMs \/ 1000\)/.test(loreSrc),
     'and the timeout message reports the limit that applied');
  ok(!/LORE_TIMEOUT_MS\)/.test(loreSrc), 'the pass no longer reads the constant directly');
}

/* ================================================================
   D. #4 — THE USER'S RUNTIME LIMIT GOES TO THE SERVER
================================================================ */
console.log('\n=== D. the runtime limit is sent with every job; off sends none ===');
{
  const reader = fnBody(CHAT, 'function cotTimeoutLimitMinutes()', 'js/10-chat.js');
  const start = between(GEN, 'async function startGenerationJob(', '\n/**', 'js/03-generation.js');
  async function urlFor(settings) {
    const urls = [];
    const ctx = {
      console: { log() {}, warn() {}, error() {} },
      state: { settings }, Math, parseInt, JSON, encodeURIComponent,
      privacyFetch: async (url) => { urls.push(url); return { status: 202, ok: true, json: async () => ({ id: 'j1' }) }; },
    };
    vm.createContext(ctx);
    vm.runInContext('var jobsAvailable = null; var JOBS_URL = "/llm/jobs";\n' + reader + '\n' + start, ctx);
    const id = await ctx.startGenerationJob({ id: 'th1' }, { messages: [] });
    return { url: urls[0], id, minutes: ctx.cotTimeoutLimitMinutes() };
  }
  const off = await urlFor({ cotTimeoutEnabled: false, cotTimeoutMinutes: 5 });
  eq(off.minutes, 0, 'auto-cull off → no limit');
  eq(off.url, '/llm/jobs?thread=th1', 'and no limit is sent — the job runs until it ends or is stopped');
  const on = await urlFor({ cotTimeoutEnabled: true, cotTimeoutMinutes: 5 });
  eq(on.url, '/llm/jobs?thread=th1&max_runtime_s=300', 'auto-cull at 5 min → max_runtime_s=300');
  eq(on.id, 'j1', 'the job still starts normally');
  const long = await urlFor({ cotTimeoutEnabled: true, cotTimeoutMinutes: 90 });
  eq(long.url, '/llm/jobs?thread=th1&max_runtime_s=5400', '90 min is sent as is — nothing caps the user’s own limit');
  const blank = await urlFor({ cotTimeoutEnabled: true, cotTimeoutMinutes: '' });
  eq(blank.url, '/llm/jobs?thread=th1&max_runtime_s=120', 'a cleared minutes field means the CONFIG default of 2, as before');

  ok(/function startCotWatchdog\(\) \{\s*clearCotWatchdog\(\);\s*const mins = cotTimeoutLimitMinutes\(\);/.test(CHAT),
     'the page’s own timer reads the same limit — the two cannot disagree');
  ok(/only time limit on a reply/.test(HTML), 'CONFIG says this is now the only time limit, and that it holds with the page closed');
}

/* ================================================================
   E. CONFIG WIRING — the fields exist and round-trip through the readers
================================================================ */
console.log('\n=== E. CONFIG has the fields and saves them ===');
{
  for (const id of ['set-auto-continue-delay', 'set-auto-continue-max', 'set-lore-timeout']) {
    ok(new RegExp('<input[^>]*id="' + id + '"').test(HTML), `CONFIG has #${id}`);
    ok(CARDS.includes(`getElementById('${id}')`), `openSettings/saveSettings read and write #${id}`);
  }
  const save = fnBody(CARDS, 'function saveSettings()', 'js/15-cards.js');
  ok(/autoContinueDelaySeconds = \(isFinite\(acDelay\) && acDelay >= 0\) \? acDelay : DEFAULT_SETTINGS\.autoContinueDelaySeconds/.test(save),
     'a cleared pause field saves the default, not NaN');
  ok(/loreTimeoutSeconds = isFinite\(loreT\) \? Math\.max\(5, loreT\) : DEFAULT_SETTINGS\.loreTimeoutSeconds/.test(save),
     'a lore timeout under 5 s saves as 5');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
