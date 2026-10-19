/**
 * The context budget is the character's — roadmap 1.7.7 item #6d.
 *
 * Four numbers decide how a character spends its context and when older
 * messages are compressed into lore. They were constants in
 * buildContextMessages (js/08-rag.js), and a copy of two of them lived in the
 * context meter (js/13-dashboard.js) with a "keep these in sync" comment:
 *
 *   input budget   90% of the context the prompt may fill
 *   reply room     20% of that kept free; less left → compression starts
 *   land at        65% of that, where compression brings the chat back to
 *   keep verbatim  22% of that, the most recent turns, never summarised
 *
 * Now each is per character, falling back to a setting (presettable from
 * gobbonet.toml), falling back to the old constant. Held here:
 *
 *   A. Nothing moves for anyone who leaves them alone.
 *   B. Card beats setting beats constant; blank inherits; 0 is a real value
 *      where it is allowed.
 *   C. Out-of-range values are clamped AND reported.
 *   D. The two combinations that stall compression are adjusted and reported:
 *      landing at/above the trigger (fires on every message), and keeping more
 *      verbatim than the landing point (can never get there).
 *   E. The budgeter and the meter both read the resolver; the editor stores
 *      what was typed (blank = inherit) and explains any adjustment.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const read = (p) => fs.readFileSync(ROOT + '/' + p, 'utf8');
const STATE = read('js/04-state.js');
const RAG = read('js/08-rag.js');
const DASH = read('js/13-dashboard.js');
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
  const a = src.indexOf(from), b = src.indexOf(to, a + 1);
  if (a < 0 || b < 0) { console.error(`could not locate ${JSON.stringify(from)} in ${file}`); process.exit(1); }
  return src.slice(a, b);
};
// The resolver and its ranges, evaluated for real.
const RESOLVER = between(STATE, 'const CONTEXT_BUDGET_RANGES', 'const DEFAULT_SETTINGS', 'js/04-state.js');
const DEFAULTS_SRC = between(STATE, 'const DEFAULT_SETTINGS = {', '\n};', 'js/04-state.js') + '\n};';
function resolver(settings) {
  const ctx = { state: { settings }, Math, Number, Object, isFinite };
  vm.createContext(ctx);
  vm.runInContext(RESOLVER + DEFAULTS_SRC.replace('const DEFAULT_SETTINGS', 'globalThis.DEFAULT_SETTINGS'), ctx);
  return ctx;
}
const r = (card, settings = {}) => resolver(settings).resolveContextBudget(card);
const pct = (b) => [b.percent.contextInputPercent, b.percent.contextReplyPercent,
                    b.percent.contextLandPercent, b.percent.contextKeepPercent].join('/');

/* ================================================================
   A. THE DEFAULTS ARE THE OLD CONSTANTS
================================================================ */
console.log('\n=== A. nothing moves for anyone who leaves them alone ===');
{
  const b = r(null);
  eq([b.input, b.reply, b.land, b.keep].join(), '0.9,0.2,0.65,0.22', 'no card, no settings → 0.9 / 0.20 / 0.65 / 0.22');
  eq(b.notes.length, 0, 'and nothing is reported as adjusted');
  const d = resolver({}).DEFAULT_SETTINGS;
  eq([d.contextInputPercent, d.contextReplyPercent, d.contextLandPercent, d.contextKeepPercent].join('/'),
     '90/20/65/22', 'DEFAULT_SETTINGS carries the same four, so gobbonet.toml [ui] can preset them');
  eq(pct(r({ contextInputPercent: null, contextReplyPercent: null, contextLandPercent: null, contextKeepPercent: null })),
     '90/20/65/22', 'a new card (all null) inherits all four');
}

/* ================================================================
   B. CARD → SETTING → CONSTANT
================================================================ */
console.log('\n=== B. the card wins, then the setting, then the old value ===');
{
  eq(pct(r({ contextInputPercent: 80 })), '80/20/65/22', 'a card value is used');
  eq(pct(r({}, { contextLandPercent: 50 })), '90/20/50/22', 'a setting is used when the card is blank');
  eq(pct(r({ contextLandPercent: 40 }, { contextLandPercent: 50 })), '90/20/40/22', 'the card beats the setting');
  eq(pct(r({ contextLandPercent: '' }, { contextLandPercent: 50 })), '90/20/50/22', 'an empty string inherits');
  eq(pct(r({ contextReplyPercent: 0, contextKeepPercent: 0 })), '90/0/65/0', '0 is honoured where allowed (reply, keep)');
  eq(pct(r({ contextInputPercent: '75' })), '75/20/65/22', 'a numeric string from an import still counts');
}

/* ================================================================
   C. OUT OF RANGE IS CLAMPED AND SAID
================================================================ */
console.log('\n=== C. out-of-range values are clamped, and reported ===');
{
  const b = r({ contextInputPercent: 120, contextReplyPercent: 80 });
  eq(b.percent.contextInputPercent, 99, 'input 120% → 99% (100% would leave no room to reply at all)');
  eq(b.percent.contextReplyPercent, 60, 'reply 80% → 60%');
  ok(b.notes.some(n => /Input budget 120%/.test(n)) && b.notes.some(n => /Reply room 80%/.test(n)),
     'both are reported', JSON.stringify(b.notes));
  eq(r({ contextInputPercent: 3 }).percent.contextInputPercent, 10, 'input 3% → 10%');
}

/* ================================================================
   D. THE COMBINATIONS THAT STALL COMPRESSION
================================================================ */
console.log('\n=== D. combinations that would stall compression are adjusted ===');
{
  const land = r({ contextReplyPercent: 20, contextLandPercent: 85 });
  eq(land.percent.contextLandPercent, 75, 'land 85% with 20% reply room → 75% (compression starts at 80%)');
  ok(land.notes.some(n => /fire on every message/.test(n)), 'reported, with the reason', JSON.stringify(land.notes));

  const keep = r({ contextLandPercent: 50, contextKeepPercent: 60 });
  eq(keep.percent.contextKeepPercent, 45, 'keep 60% under a 50% landing point → 45%');
  ok(keep.notes.some(n => /could never reach it/.test(n)), 'reported, with the reason', JSON.stringify(keep.notes));

  const chain = r({ contextReplyPercent: 60, contextLandPercent: 90, contextKeepPercent: 90 });
  eq(pct(chain), '90/60/35/30', 'they chain: reply 60 → land ≤35 → keep ≤30');
  eq(chain.notes.length, 2, 'each adjustment reported once');

  // Every value a field can hold resolves to an ordering that works.
  let bad = 0;
  for (const rep of [0, 20, 40, 60]) for (const land of [10, 40, 70, 95]) for (const kp of [0, 30, 60, 90]) {
    const b = r({ contextReplyPercent: rep, contextLandPercent: land, contextKeepPercent: kp });
    if (!(b.keep < b.land && b.land < 1 - b.reply)) bad++;
  }
  eq(bad, 0, 'across the whole range: keep < land < where compression starts, always');
}

/* ================================================================
   E. WIRING
================================================================ */
console.log('\n=== E. one resolver for the budgeter, the meter and the editor ===');
{
  const build = between(RAG, 'async function buildContextMessages', 'const writingStyleTokens', 'js/08-rag.js');
  ok(/const ctxBudget = resolveContextBudget\(card\);/.test(build), 'buildContextMessages resolves the card’s budget');
  ok(/Math\.floor\(tokenLimit \* ctxBudget\.input\)/.test(build), 'input budget from it');
  ok(/TARGET_FACTOR = ctxBudget\.land/.test(build) && /RECENT_RESERVE_FACTOR = ctxBudget\.keep/.test(build)
     && /RESPONSE_RESERVE_FACTOR = ctxBudget\.reply/.test(build), 'land, keep and reply room from it');
  ok(!/tokenLimit \* 0\.9|= 0\.65;|= 0\.22;|= 0\.20;/.test(build), 'no constant left behind');
  const meter = between(DASH, 'const projected = systemTokens', 'const pct ', 'js/13-dashboard.js');
  ok(/resolveContextBudget\(card\)/.test(meter) && !/\* 0\.9|\* 0\.20/.test(meter),
     'the context meter reads the same resolver — no second copy to drift');

  for (const id of ['card-ctx-input-pct', 'card-ctx-reply-pct', 'card-ctx-land-pct', 'card-ctx-keep-pct']) {
    ok(new RegExp(`<input type="number" id="${id}"[^>]*oninput="updateCardBudgetHint\\(\\)"`).test(HTML),
       `the editor has #${id}, and the note updates as it is typed`);
  }
  ok(/id="card-budget-hint"/.test(HTML), 'there is a note that explains the effect and any adjustment');
  ok(!/90% of this is the input budget/.test(CARDS), 'the context hint no longer says a fixed 90%');

  // Saving stores what was typed (range-held), blank as null — never the
  // adjusted value, which would change the user's number behind their back.
  const reader = between(CARDS, 'function readCardBudgetFields()', '/* The card being edited', 'js/15-cards.js');
  const els = { 'card-ctx-input-pct': { value: '' }, 'card-ctx-reply-pct': { value: '20' },
                'card-ctx-land-pct': { value: '85' }, 'card-ctx-keep-pct': { value: '300' } };
  const ctx = resolver({});
  ctx.document = { getElementById: (id) => els[id] || null };
  vm.runInContext('const CARD_BUDGET_FIELDS = { "card-ctx-input-pct": "contextInputPercent", "card-ctx-reply-pct": "contextReplyPercent", "card-ctx-land-pct": "contextLandPercent", "card-ctx-keep-pct": "contextKeepPercent" };\n'
                  + reader, ctx);
  const saved = ctx.readCardBudgetFields();
  eq(saved.contextInputPercent, null, 'a blank field saves as null — inherit');
  eq(saved.contextLandPercent, 85, 'land 85 is stored as typed (it is adjusted at use, and the note says so)');
  eq(saved.contextKeepPercent, 90, 'keep 300 is held to its own field range, 90');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
