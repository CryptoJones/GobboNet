/**
 * The lore prompt: what the model is told when Lore System writes older
 * messages down, and the card editor's box for changing it.
 *
 * The bug: the old prompt showed its format with a worked example, a vampire
 * story about "Lucia" and "Vasch". Small models copied the example, so
 * "Lucia" turned up in people's lore and, because lore is fed back as fact,
 * in their chats. The default now names nobody, and a card can replace it.
 *
 * Runs the real js/07-prompt.js (whole), the editor functions from
 * js/15-cards.js and the import/export from js/16-card-io.js.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const read = (p) => fs.readFileSync(ROOT + '/' + p, 'utf8');
const PROMPT = read('js/07-prompt.js');
const CARDS = read('js/15-cards.js');
const CARDIO = read('js/16-card-io.js');
const STATE = read('js/04-state.js');
const HTML = read('chat.html');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra !== undefined ? '\n      ' + String(extra).slice(0, 600) : '')); }
};
const between = (src, from, to, file) => {
  const a = src.indexOf(from), b = src.indexOf(to, a + 1);
  if (a < 0 || b < 0) { console.error(`could not locate ${JSON.stringify(from)} in ${file}`); process.exit(1); }
  return src.slice(a, b);
};

/* ---- a harness around the whole of 07-prompt.js ------------------------ */
function makeSSE(text) {
  const body = 'data: ' + JSON.stringify({ choices: [{ delta: { content: text } }] }) + '\n' + 'data: [DONE]\n';
  const bytes = new TextEncoder().encode(body);
  let done = false;
  return { getReader: () => ({ read: async () => done ? { done: true } : (done = true, { done: false, value: bytes }) }) };
}

function promptContext({ reply = 'Mara opened the vault.', persona = {} } = {}) {
  const requests = [];
  const ctx = {
    console: { log() {}, warn() {}, error() {} },
    Date, Math, JSON, String, Number, Array, Object, RegExp, parseInt,
    TextDecoder, TextEncoder, setTimeout, clearTimeout, AbortController, Promise,
    IS_SERVED: true, LLAMA_URL: '/llm', _currentModelFile: 'chat.gguf',
    getRequestModelName: () => 'local',
    activeModel: { name: 'T', family: 'llama', thinkingFormat: 'none' },
    state: { settings: {} },
    renderLoreIndicator() {}, showModelSwitchToast() {},
    normalizeThinkingFormat: () => 'none',
    getActivePersona: () => persona, escapeHtml: (s) => String(s ?? ''),
    extractTokenFromLine: (line) => {
      const t = line.trim();
      if (!t.startsWith('data:') || t.includes('[DONE]')) return null;
      try { const d = JSON.parse(t.slice(5)).choices?.[0]?.delta?.content; return d ? { text: d, field: 'content' } : null; }
      catch { return null; }
    },
    processStreamDelta: (m, t, f) => { m[f] += t; },
    finalizeStreamMessage: (m) => { m.content = m.content.trim(); },
    swapToModelFile: async () => {},
    privacyFetch: async (url, opts) => {
      if (String(url).endsWith('/props')) return { ok: true, json: async () => ({ default_generation_settings: { n_ctx: 32768 } }) };
      requests.push(JSON.parse(opts.body));
      return { ok: true, body: makeSSE(ctx.__reply) };
    },
  };
  ctx.__reply = reply;
  ctx.globalThis = ctx; ctx.window = ctx;
  vm.createContext(ctx);
  vm.runInContext(PROMPT + `
    globalThis.__compress = compressWithCardModel;
    globalThis.__sys = loreSystemPrompt;
    globalThis.__DEFAULT = LORE_DEFAULT_PROMPT;
    globalThis.__MAX = LORE_PROMPT_MAX_CHARS;
    globalThis.__echo = _loreEchoesTemplate;
    globalThis.__probe = () => ({ kind: _loreLastKind, outcome: _loreLastOutcome });
  `, ctx, { filename: '07-prompt.js' });
  ctx.requests = requests;
  return ctx;
}
const msgs = [
  { role: 'user', content: 'Mara, can you open the vault before the guards come back?' },
  { role: 'assistant', content: 'Mara turned the last dial and the vault door swung open.' },
];
const sysOf = (req) => (req.messages.find(m => m.role === 'system') || {}).content || '';
const userOf = (req) => req.messages.filter(m => m.role !== 'system').map(m => m.content).join('\n');

/* ================================================================ A */
console.log('\n=== A. the default prompt names nobody ===');
{
  const c = promptContext();
  const P = c.__DEFAULT;
  ok(!/lucia|vasch/i.test(P), 'no Lucia, no Vasch', P);
  // Every capitalised word in it is a sentence start, a rule word or SKIP,
  // so there is no name for a model to copy.
  const caps = (P.match(/\b[A-Z][a-z]+\b/g) || []).filter(w => !['You', 'Write', 'Rules', 'One', 'Only', 'Never', 'Concrete', 'No', 'Use', 'If', 'An'].includes(w));
  ok(caps.length === 0, 'no other capitalised word that could be read as a name', caps.join(', '));
  ok(!/vampire|tavern|castle|sword|dragon|\bking\b|\bqueen\b/i.test(P), 'and no story of its own: no setting, no genre');
  const words = P.split(/\s+/).filter(Boolean).length;
  ok(words <= 130 && P.length <= 800, `short: ${words} words, ${P.length} characters`);
  ok(/Use only names that appear in the messages\. Never invent a name\./.test(P), 'it says to use only names from the messages');
  ok(/write exactly: SKIP/.test(P), 'it keeps SKIP for a pass with nothing new');
  ok(/One sentence, under 20 words, past tense\./.test(P), 'one sentence, under 20 words, past tense');
  ok(/<who> <did what> <to whom or where>\./.test(P), 'the shape is a template, its parts in <...>');
  const rules = (P.match(/^\d\. /gm) || []).length;
  ok(rules === 6, `six numbered rules (${rules})`);
}

/* ================================================================ B */
console.log('\n=== B. what reaches the model by default ===');
{
  const c = promptContext();
  const out = await c.__compress('', msgs, '', 32768, { name: 'Mara', loreEnabled: true });
  ok(c.requests.length === 1, 'one request');
  const req = c.requests[0] || { messages: [] };
  ok(sysOf(req) === c.__DEFAULT, 'the system message is the default prompt, unchanged');
  ok(!/lucia|vasch/i.test(JSON.stringify(req)), 'nothing in the whole request says Lucia or Vasch');
  ok(/Write the one new entry now\. One sentence\. Nothing else\.$/.test(userOf(req)), 'the default closes on its own rules');
  ok(out.trim() === '- Mara opened the vault.', 'the entry is stored as written, as one line', out);
  ok(c.__probe().kind === 'ok', 'outcome ok', c.__probe().kind);

  for (const [label, card] of [['a card with no prompt fields', { name: 'X' }],
                               ['ticked, but the box is empty', { name: 'X', loreCustomPrompt: true, lorePrompt: '   ' }],
                               ['written, but not ticked', { name: 'X', loreCustomPrompt: false, lorePrompt: 'Write haiku.' }],
                               ['no card at all', null]]) {
    ok(c.__sys(card) === c.__DEFAULT, label + ': the default');
  }
}

/* ================================================================ C */
console.log('\n=== C. a card\'s own prompt ===');
{
  const c = promptContext({ persona: { name: 'Sam' } });
  const card = { name: 'Mara', loreEnabled: true, loreCustomPrompt: true,
                 lorePrompt: '  Log what {{char}} and {{USER}} did, in one line. Reply SKIP if nothing.  ' };
  await c.__compress('', msgs, '', 32768, card);
  const req = c.requests[0] || { messages: [] };
  ok(sysOf(req) === 'Log what Mara and Sam did, in one line. Reply SKIP if nothing.',
     'ticked: the card\'s prompt is the system message, trimmed, with {{char}} and {{user}} filled in', sysOf(req));
  ok(/=== END ===\nWrite the new entry now\.$/.test(userOf(req)), 'and it closes with a neutral line, not the default\'s rules', userOf(req).slice(-80));
  ok(!/One sentence\. Nothing else/.test(userOf(req)), 'the default\'s rules are not added behind it');

  const anon = promptContext();
  ok(anon.__sys({ loreCustomPrompt: true, lorePrompt: '{{char}} / {{user}}' }) === 'the character / the user',
     'with no name and no persona, the macros read "the character" and "the user"');
  const long = anon.__sys({ loreCustomPrompt: true, lorePrompt: 'x'.repeat(9000) });
  ok(long.length === anon.__MAX && anon.__MAX === 4000, 'a prompt longer than 4,000 characters is cut there', long.length);

  // Ticked with the default's own words is the default, closing line and all.
  const same = promptContext();
  await same.__compress('', msgs, '', 32768, { name: 'Mara', loreCustomPrompt: true, lorePrompt: same.__DEFAULT });
  ok(/One sentence\. Nothing else\.$/.test(userOf(same.requests[0])), 'ticked with the built-in words is treated as the built-in prompt');
}

/* ================================================================ D */
console.log('\n=== D. a reply that is the template is not stored ===');
{
  const c = promptContext({ reply: '<who> <did what> <to whom or where>.' });
  const out = await c.__compress('- Earlier.', msgs, '', 32768, { name: 'Mara' });
  ok(!/<who>|did what/.test(out), 'the template is not written into the lore', out);
  ok(/2 earlier messages dropped/.test(out) && out.startsWith('- Earlier.'), 'the lore keeps what it had and notes the gap instead', out);
  ok(c.__probe().kind === 'failed' && /template/.test(c.__probe().outcome), 'and the pass is reported as failed, saying why', c.__probe().outcome);
  ok(c.__echo('Mara <did what> the vault.'), 'a half-filled template counts too');
  ok(!c.__echo('Mara opened the vault.') && !c.__echo('Mara wrote <3 on the door.') && !c.__echo(''), 'ordinary entries, angle brackets and all, do not');

  const skip = promptContext({ reply: 'SKIP' });
  const kept = await skip.__compress('- Earlier.', msgs, '', 32768, { name: 'Mara' });
  ok(kept.trim() === '- Earlier.', 'SKIP still adds nothing', kept);
}

/* ================================================================ E */
console.log('\n=== E. the source no longer carries the example ===');
{
  // Comments may mention it as history; no string may.
  const code = PROMPT.replace(/\/\*[\s\S]*?\*\//g, '').split('\n').map(l => l.replace(/(^|\s)\/\/.*$/, '')).join('\n');
  ok(!/lucia|vasch/i.test(code), 'no Lucia or Vasch anywhere in 07-prompt.js outside comments');
  ok(!/\bthe shape\b.*example/i.test(code), 'and no worked example of a beat');
  for (const f of fs.readdirSync(ROOT + '/js')) {
    const src = read('js/' + f).replace(/\/\*[\s\S]*?\*\//g, '');
    if (/['"`][^'"`\n]*\b(Lucia|Vasch)\b/.test(src)) ok(false, `js/${f} has Lucia or Vasch in a string`);
  }
  ok(!/\bLucia\b|\bVasch\b/.test(HTML), 'nor chat.html');
  ok(/const prompt = loreSystemPrompt\(card\);/.test(PROMPT)
     && (PROMPT.match(/summarizeForLore\(existingLore, toArchive, authored, tokenLimit, prompt\)/g) || []).length === 2,
     'both compression paths (with and without a borrowed model) pass the card\'s prompt');
}

/* ================================================================ F */
console.log('\n=== F. the editor box ===');
{
  const editor = between(HTML, 'id="card-starting-lore"', 'id="card-rag-storybook"', 'chat.html');
  ok(/id="card-lore-prompt-group"/.test(editor), 'the box sits between Starting Lore and RAG Storybook');
  ok(/<input type="checkbox" id="card-lore-prompt-custom" onchange="toggleLorePromptEditor\(\)">/.test(editor)
     && /Edit the lore summary prompt/.test(editor), 'a checkbox, "Edit the lore summary prompt"');
  ok(/<div id="card-lore-prompt-body" style="display:none;">/.test(editor), 'the text box is hidden until it is ticked');
  ok(/<textarea id="card-lore-prompt"[^>]*maxlength="4000"/.test(editor), 'a textarea capped at 4,000 characters');
  ok(/onclick="resetLorePrompt\(\)"/.test(editor), 'and a reset to the built-in prompt');
  ok(/SKIP/.test(editor) && /\{\{char\}\}/.test(editor) && /\{\{user\}\}/.test(editor) && /example names/.test(editor),
     'the hint covers SKIP, the macros, and why not to use example names');
  ok(/class="code-enable-row"/.test(editor), 'styled like the other enable rows');

  // The functions, on a stub page.
  const els = {
    'card-lore-prompt-custom': { checked: false },
    'card-lore-prompt-body': { style: { display: 'none' } },
    'card-lore-prompt': { value: '' },
  };
  const c = { document: { getElementById: (id) => els[id] || null }, Object, String };
  vm.createContext(c);
  vm.runInContext(between(PROMPT, 'const LORE_DEFAULT_PROMPT', 'function loreSystemPrompt', 'js/07-prompt.js')
    + between(CARDS, 'function loadLorePromptEditor', 'function saveCard() {', 'js/15-cards.js')
    + ';globalThis.__DEFAULT = LORE_DEFAULT_PROMPT;', c);
  const D = c.__DEFAULT;

  c.loadLorePromptEditor({ name: 'A' });
  ok(!els['card-lore-prompt-custom'].checked && els['card-lore-prompt-body'].style.display === 'none',
     'a card without its own prompt opens unticked, box hidden');
  ok(els['card-lore-prompt'].value === D, 'the box is filled with the built-in prompt, ready to edit');
  ok(JSON.stringify(c.readLorePromptEditor({})) === JSON.stringify({ loreCustomPrompt: false, lorePrompt: '' }),
     'saved untouched: nothing stored, the card follows the built-in prompt');

  els['card-lore-prompt-custom'].checked = true; c.toggleLorePromptEditor();
  ok(els['card-lore-prompt-body'].style.display === '', 'ticking shows the box');
  ok(c.readLorePromptEditor({}).loreCustomPrompt === false, 'ticked but unchanged is still the built-in prompt (stored as nothing)');

  els['card-lore-prompt'].value = 'Write what {{char}} did.';
  ok(JSON.stringify(c.readLorePromptEditor({})) === JSON.stringify({ loreCustomPrompt: true, lorePrompt: 'Write what {{char}} did.' }),
     'ticked and edited: saved as the card\'s own');

  els['card-lore-prompt-custom'].checked = false; c.toggleLorePromptEditor();
  ok(els['card-lore-prompt-body'].style.display === 'none' && els['card-lore-prompt'].value === 'Write what {{char}} did.',
     'unticking hides the box but keeps what was typed');
  const off = c.readLorePromptEditor({});
  ok(off.loreCustomPrompt === false && off.lorePrompt === 'Write what {{char}} did.', 'saved unticked: built-in prompt used, the text kept for later');

  c.resetLorePrompt();
  ok(els['card-lore-prompt'].value === D, 'RESET TO DEFAULT puts the built-in prompt back');

  els['card-lore-prompt-custom'].checked = true; els['card-lore-prompt'].value = '';
  c.toggleLorePromptEditor();
  ok(els['card-lore-prompt'].value === D, 'ticking an empty box fills it with the built-in prompt');

  c.loadLorePromptEditor({ loreCustomPrompt: true, lorePrompt: 'Mine.' });
  ok(els['card-lore-prompt-custom'].checked && els['card-lore-prompt'].value === 'Mine.' && els['card-lore-prompt-body'].style.display === '',
     'a card with its own prompt opens ticked, showing it');
  els['card-lore-prompt'].value = 'y'.repeat(5000);
  ok(c.readLorePromptEditor({}).lorePrompt.length === 4000, 'over 4,000 characters (pasted past maxlength) is cut on save');

  const bare = { document: { getElementById: () => null }, Object, String };
  vm.createContext(bare);
  vm.runInContext(between(PROMPT, 'const LORE_DEFAULT_PROMPT', 'function loreSystemPrompt', 'js/07-prompt.js')
    + between(CARDS, 'function loadLorePromptEditor', 'function saveCard() {', 'js/15-cards.js'), bare);
  ok(JSON.stringify(bare.readLorePromptEditor({ loreCustomPrompt: true, lorePrompt: 'Kept.' })) === JSON.stringify({ loreCustomPrompt: true, lorePrompt: 'Kept.' }),
     'a page without the box leaves the card\'s prompt as it was');

  const save = between(CARDS, 'function saveCard() {', '\nfunction ', 'js/15-cards.js');
  ok(/Object\.assign\(card, readLorePromptEditor\(card\)\);/.test(save), 'SAVE stores the box');
  const edit = between(CARDS, 'function editCard(', 'function ', 'js/15-cards.js');
  ok(/loadLorePromptEditor\(card\);/.test(edit), 'opening a card fills the box');
  ok(/loreCustomPrompt: false,\s*lorePrompt: '',/.test(between(CARDS, 'function createCard() {', '\n}', 'js/15-cards.js')), 'a new card starts on the built-in prompt');
  ok(/loreCustomPrompt: false,\s*lorePrompt: '',/.test(between(STATE, 'const DEFAULT_CARD = {', '\n};', 'js/04-state.js')), 'so does the default card');
  ok(/querySelectorAll\('input, textarea, select'\)/.test(read('js/22-scheduler.js')) && /\.checked/.test(read('js/22-scheduler.js')),
     'the unsaved-changes guard sees the checkbox and the text');
}

/* ================================================================ G */
console.log('\n=== G. export and import ===');
{
  const c = {
    console: { log() {}, warn() {}, error() {} }, Object, Array, String, Number, JSON, Math, Date, RegExp,
    DEFAULT_CARD: { name: '', memoryFiles: [], loreCustomPrompt: false, lorePrompt: '' }, generateId: () => 'new-id',
    applyImportedCardCode: (card) => card, parseAltGreetings: () => [],
    memoryPutFile: () => ({ ok: false }), cardMemoryFiles: (card) => (card && card.memoryFiles) || [],
    FALLBACK_CARD_TEXT_COLOR: '#fff', FALLBACK_CARD_DIALOG_COLOR: '#f0f',
  };
  vm.createContext(c);
  vm.runInContext('const LORE_PROMPT_MAX_CHARS = 4000;'
    + between(CARDIO, 'function _stripOriginalMacro', 'function _uniqueCardName', 'js/16-card-io.js')
    + between(CARDIO, 'function _cardToV3', 'function _v3ToV2', 'js/16-card-io.js'), c);

  const v3 = c._cardToV3({ name: 'Aria', loreCustomPrompt: true, lorePrompt: 'Log {{char}}\'s deeds.' });
  ok(v3.data.extensions.gobbonet && v3.data.extensions.gobbonet.lorePrompt === 'Log {{char}}\'s deeds.',
     'a card using its own prompt exports it in extensions.gobbonet.lorePrompt');
  ok(!('gobbonet' in c._cardToV3({ name: 'Plain' }).data.extensions), 'a card on the built-in prompt adds nothing');
  ok(!('gobbonet' in c._cardToV3({ name: 'Off', loreCustomPrompt: false, lorePrompt: 'Unused.' }).data.extensions),
     'nor does one whose prompt is written but unticked');

  const back = c._cardDataToInternal(v3.data, 'chara_card_v3', '');
  ok(back.loreCustomPrompt === true && back.lorePrompt === 'Log {{char}}\'s deeds.', 'import reads it back, ticked');
  const none = c._cardDataToInternal({ name: 'Other app' }, 'chara_card_v2', '');
  ok(none.loreCustomPrompt === false && none.lorePrompt === '', 'a card from another app arrives on the built-in prompt');
  for (const [label, lp] of [['a number', 7], ['an object', { a: 1 }], ['blank', '   ']]) {
    const r = c._cardDataToInternal({ name: 'X', extensions: { gobbonet: { lorePrompt: lp } } }, 'chara_card_v3', '');
    ok(r.loreCustomPrompt === false && r.lorePrompt === '', `a lorePrompt that is ${label} is ignored`);
  }
  const big = c._cardDataToInternal({ name: 'X', extensions: { gobbonet: { lorePrompt: 'a\u0000b' + 'z'.repeat(9000) } } }, 'chara_card_v3', '');
  ok(big.lorePrompt.length === 4000 && !big.lorePrompt.includes('\u0000'), 'an imported prompt is capped at 4,000 characters, NULs removed');

  const snap = between(CARDIO, 'function _liveCardSnapshot', 'function _cardFileStem', 'js/16-card-io.js');
  ok(/readLorePromptEditor\(card\)/.test(snap), 'exporting from the open editor includes its unsaved prompt');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
