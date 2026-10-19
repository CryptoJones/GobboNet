/**
 * Thinking on/off per character — roadmap 1.7.7 item #9.
 *
 * A checkbox on each character card. Off sends
 *     chat_template_kwargs: { enable_thinking: false }
 * with that character's requests; llama-server (pinned b10456) hands it to the
 * model's chat template, and a template that supports it answers without its
 * reasoning step.
 *
 * What this holds:
 *   A. Off sends exactly that, and enable_thinking is a real boolean — the
 *      engine rejects the request if it arrives as the string "false".
 *   B. On sends NOTHING: a thinking character's request is byte-identical to
 *      what it was before this setting existed.
 *   C. A card with no field — older, imported, from another app — thinks.
 *   D. Both chat requests (send and regenerate) carry it; the editor shows and
 *      saves it; new cards and the default card start with it on.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const read = (p) => fs.readFileSync(ROOT + '/' + p, 'utf8');
const SYNC = read('js/06-state-sync.js');
const CHAT = read('js/10-chat.js');
const CARDS = read('js/15-cards.js');
const STATE = read('js/04-state.js');
const HTML = read('chat.html');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};
const eq = (a, b, l) => ok(a === b, l,
  a === b ? '' : `got:  ${JSON.stringify(a)}\n      want: ${JSON.stringify(b)}`);

const fnBody = (src, sig, file) => {
  const a = src.indexOf(sig);
  if (a < 0) { console.error(`could not locate ${sig} in ${file}`); process.exit(1); }
  return src.slice(a, src.indexOf('\n}\n', a) + 2);
};

const ctx = { getActiveCard: () => ({ name: 'active' }), JSON };
vm.createContext(ctx);
vm.runInContext(fnBody(SYNC, 'function cardThinkingEnabled(card)', 'js/06-state-sync.js') + '\n'
              + fnBody(SYNC, 'function getCardThinkingParams(card)', 'js/06-state-sync.js'), ctx);
const params = (card) => JSON.parse(JSON.stringify(ctx.getCardThinkingParams(card)));

/* ================================================================
   A. OFF SENDS enable_thinking: false, AS A BOOLEAN
================================================================ */
console.log('\n=== A. off asks the template not to think ===');
{
  const p = params({ thinkingEnabled: false });
  eq(JSON.stringify(p), '{"chat_template_kwargs":{"enable_thinking":false}}',
     'off → chat_template_kwargs.enable_thinking = false');
  eq(typeof p.chat_template_kwargs.enable_thinking, 'boolean',
     'a real boolean — the engine rejects the string "false"');
}

/* ================================================================
   B. ON SENDS NOTHING
================================================================ */
console.log('\n=== B. on leaves the request exactly as it was ===');
{
  eq(JSON.stringify(params({ thinkingEnabled: true })), '{}', 'on → no fields at all');
  const base = { model: 'm', messages: [], stream: true, max_tokens: -1, temperature: 0.7 };
  eq(JSON.stringify({ ...base, ...ctx.getCardThinkingParams({ thinkingEnabled: true }) }),
     JSON.stringify(base), 'a thinking character’s request is byte-identical to before');
}

/* ================================================================
   C. NO FIELD MEANS THINKING, AS BEFORE
================================================================ */
console.log('\n=== C. a card without the field thinks ===');
{
  eq(JSON.stringify(params({})), '{}', 'a card from before 1.7.7 → thinks');
  eq(JSON.stringify(params({ thinkingEnabled: undefined })), '{}', 'undefined → thinks');
  eq(JSON.stringify(params({ thinkingEnabled: 0 })), '{}', 'only an explicit false turns it off');
  eq(JSON.stringify(params(null)), '{}', 'no card given → the active card, which thinks');
}

/* ================================================================
   D. WIRING
================================================================ */
console.log('\n=== D. both requests, the editor and the defaults ===');
{
  const bodies = CHAT.match(/const requestBody = \{[^\n]*\};/g) || [];
  eq(bodies.length, 2, 'there are two chat request bodies (send, regenerate)');
  ok(bodies.every(b => b.includes('...getCardThinkingParams(card)')),
     'both carry the thinking switch', bodies.join('\n      '));

  const box = HTML.slice(HTML.indexOf('Context &amp; Reply Length'), HTML.indexOf('Sampler Parameters'));
  ok(/<input type="checkbox" id="card-thinking-enabled">/.test(box),
     'the checkbox is in the card’s Context & Reply Length box');
  const capRow = /<label class="code-enable-row" style="margin-top:4px;">\s*<input type="checkbox" id="card-smart-limit-enabled">/.test(box);
  const thinkRow = /<label class="code-enable-row" style="margin-top:4px;">\s*<input type="checkbox" id="card-thinking-enabled">/.test(box);
  ok(capRow && thinkRow, 'and its row is built exactly like the reply-cap row');
  ok(box.indexOf('card-thinking-enabled') > box.indexOf('card-smart-limit-enabled'), 'placed after the reply cap');

  ok(/getElementById\('card-thinking-enabled'\)\.checked = \(card\.thinkingEnabled !== false\)/.test(CARDS),
     'opening a card shows a missing field as on');
  ok(/card\.thinkingEnabled = document\.getElementById\('card-thinking-enabled'\)\.checked/.test(CARDS),
     'saving a card stores the checkbox');
  ok(/smartLimitTokens: 300,\s*thinkingEnabled: true/.test(CARDS), 'a new card starts with thinking on');
  const defStart = STATE.indexOf('const DEFAULT_CARD = {');
  const defaultCard = defStart < 0 ? '' : STATE.slice(defStart, STATE.indexOf('\n};', defStart));
  ok(/\n  thinkingEnabled: true,/.test(defaultCard), 'the default card has thinking on');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
