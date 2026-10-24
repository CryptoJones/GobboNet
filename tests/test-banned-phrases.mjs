/**
 * Banned phrases through logit_bias -- issue #51, the stand-in for CFG.
 *
 * Real classifier-free guidance needs logits llama-server does not expose, so
 * a card's Banned Phrases list is the "stop writing X" part of it, sent as
 * logit_bias. The first version banned every token of every case variant of
 * every phrase: a phrase took its ordinary words with it (" her", " down"),
 * and a word that splits took fragments with it ("Chuckle" -> [Ch][uckle]
 * banned [Ch] everywhere). It also re-tokenized on every message.
 *
 * Runs the real code out of js/06-state-sync.js in Node's vm, against a
 * stand-in /tokenize that splits greedily over a small vocabulary:
 *   A. picking the word: anchors, variants, de-duplication, the cap
 *   B. picking the token: own token, a safe leading piece, never a fragment
 *      or a common word, space-only pieces skipped
 *   C. buildLogitBias: what is sent, the strength and "ban outright"
 *   D. engines: an older /tokenize without pieces, one that is down
 *   E. the cache: one round of /tokenize per model, not per message
 *   F. CHECK in the editor: the readout, as text, never HTML
 *   G. the page, the request sites, and the docs that called it broken
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const read = (p) => fs.readFileSync(ROOT + '/' + p, 'utf8');
// BANNED_SYNC_SRC lets the same checks run against another copy of the file
// (the code before this change, to see them fail).
const SYNC = process.env.BANNED_SYNC_SRC ? fs.readFileSync(process.env.BANNED_SYNC_SRC, 'utf8') : read('js/06-state-sync.js');
const HTML = read('chat.html');
const CARDS = read('js/15-cards.js');
const CHAT = read('js/10-chat.js');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};
const eq = (a, b, l) => ok(JSON.stringify(a) === JSON.stringify(b), l,
  `got:  ${JSON.stringify(a)}\n      want: ${JSON.stringify(b)}`);
const between = (src, from, to) => {
  const a = src.indexOf(from), b = src.indexOf(to, a + 1);
  return (a < 0 || b < 0) ? null : src.slice(a, b);
};

/* ---- A stand-in tokenizer: greedy longest match over a small vocabulary,
   one token per character for anything else. Shaped like a real BPE vocab
   where it matters: common mid-sentence words are one token with their
   leading space, capitalised and upper-case forms often are not. */
const VOCAB = [
  ' suddenly', 'suddenly', ' Suddenly', 'Suddenly',
  ' chuckle', ' chuck', 'ch', 'Ch', ' Ch', 'uckle',
  ' shivers', 'shivers', ' Shivers', 'Shivers',
  ' her', 'her', ' Her', 'Her', ' down', ' spine', ' Spine', 'spine',
  ' mix', 'mix', ' Mix', 'Mix',
  ' what', 'ever', ' El', 'El', 'ara',
  ' the', ' of', ' arms', 'arms', ' Arms', 'Arms', ' tape',
];
const ID = new Map(VOCAB.map((t, i) => [t, 100 + i]));
const PIECE = new Map([...ID].map(([t, i]) => [i, t]));
function tokenize(text) {
  const out = [];
  let i = 0;
  while (i < text.length) {
    let best = null;
    for (const t of VOCAB) if (text.startsWith(t, i) && (!best || t.length > best.length)) best = t;
    if (best) { out.push({ id: ID.get(best), piece: best }); i += best.length; }
    else { out.push({ id: 10000 + text.codePointAt(i), piece: text[i] }); i += 1; }
  }
  return out;
}
const idOf = (t) => ID.get(t);

/* ---- Load the block, with the globals it reads. ---- */
function load(opts = {}) {
  const calls = [];
  const logs = { log: [], warn: [] };
  const els = {};
  const ctx = {
    LLAMA_URL: 'http://127.0.0.1:9/llm',
    _currentModelFile: 'fake.gguf',
    activeModel: { name: 'Fake' },
    console: { log: (...a) => logs.log.push(a), warn: (...a) => logs.warn.push(a), error: () => {} },
    document: { getElementById: (id) => els[id] || null },
    JSON, Promise, Map, Set, Number, String, Array, Math, Error, isFinite,
  };
  ctx.privacyFetch = async (url, init) => {
    const body = JSON.parse(init.body);
    calls.push({ url, body });
    if (opts.onFetch) opts.onFetch(body);
    if (opts.mode === 'throw') throw new Error('connection refused');
    if (opts.mode === 'down') return { ok: false, status: 503, json: async () => ({}) };
    let toks = tokenize(body.content);
    if (opts.mode === 'spm' && body.content.startsWith(' ')) toks = [{ id: 7, piece: ' ' }, ...toks];
    const legacy = opts.mode === 'legacy' || !body.with_pieces;
    return { ok: true, status: 200, json: async () => ({ tokens: legacy ? toks.map(t => t.id) : toks }) };
  };
  vm.createContext(ctx);
  const block = between(SYNC, '/* ================================================================\n   BANNED PHRASES', 'function resetSamplerDefaults()')
    || between(SYNC, '/**\n * Tokenize banned phrases', 'function resetSamplerDefaults()');
  if (!block) { console.error('could not locate the banned-phrases block in js/06-state-sync.js'); process.exit(1); }
  vm.runInContext(block + '\n;globalThis.__cache = (typeof _bannedTokenCache !== "undefined") ? _bannedTokenCache : null;', ctx);
  return { ctx, calls, logs, els };
}
const has = (ctx, name) => typeof ctx[name] === 'function';
const biasOf = async (env, phrases, strength) => {
  const card = { bannedPhrases: phrases };
  if (strength !== undefined) card.logitBiasStrength = strength;
  const r = await env.ctx.buildLogitBias(card);
  return (r && r.logit_bias) || {};
};
const keys = (bias) => Object.keys(bias).map(Number).sort((a, b) => a - b);
const pieces = (bias) => keys(bias).map(i => PIECE.get(i) ?? ('#' + i));

console.log('\nA. Picking the word');
{
  const { ctx } = load();
  ok(has(ctx, 'bannedAnchorWord') && has(ctx, 'bannedVariants') && has(ctx, 'bannedPhraseList'),
    'the block has bannedAnchorWord, bannedVariants and bannedPhraseList');
  if (has(ctx, 'bannedAnchorWord')) {
    eq(ctx.bannedAnchorWord('suddenly'), 'suddenly', 'a single word is itself');
    eq(ctx.bannedAnchorWord('  Chuckle. '), 'Chuckle', 'punctuation around a single word is dropped');
    eq(ctx.bannedAnchorWord('her'), 'her', 'a single common word is still banned: it was asked for by name');
    eq(ctx.bannedAnchorWord('shivers down her spine'), 'shivers', 'a phrase is stopped at its longest uncommon word');
    eq(ctx.bannedAnchorWord('a mix of'), 'mix', '"a mix of" is stopped at "mix"');
    eq(ctx.bannedAnchorWord('she said with a smirk'), 'smirk', 'common words never win, however long ("with")');
    eq(ctx.bannedAnchorWord('of the'), '', 'a phrase of only common words has no anchor');
    eq(ctx.bannedAnchorWord('— … —'), '', 'punctuation alone has no anchor');
    eq(ctx.bannedAnchorWord("can't help but"), "can't", 'an apostrophe stays inside its word');
    eq(ctx.bannedAnchorWord('<|im_end|>'), 'end', 'special-token syntax is read as its words, never as a token');
  }
  if (has(ctx, 'bannedAnchorCandidates')) {
    eq(ctx.bannedAnchorCandidates('shivers down her spine'), ['shivers', 'spine'], 'a phrase offers its uncommon words, longest first');
    eq(ctx.bannedAnchorCandidates('Spine, spine and SPINE'), ['Spine'], 'each word once, in any case');
    eq(ctx.bannedAnchorCandidates('alpha bravo delta gamma kilo lima'), ['alpha', 'bravo', 'delta', 'gamma'],
      'equal lengths keep the phrase\'s order; at most four are tried');
  } else ok(false, 'the block has bannedAnchorCandidates');
  if (has(ctx, 'bannedVariants')) {
    eq(ctx.bannedVariants('suddenly'), [' suddenly', 'suddenly', ' Suddenly', 'Suddenly', ' SUDDENLY', 'SUDDENLY'],
      'variants: as typed / lower, Title, UPPER, each with and without the leading space');
    eq(ctx.bannedVariants('McCoy').length, 8, 'a mixed-case word keeps its own spelling as a fourth form');
  }
  if (has(ctx, 'bannedPhraseList')) {
    eq(ctx.bannedPhraseList(' suddenly \n\nSuddenly\nchuckle\n  \n'), ['suddenly', 'chuckle'],
      'one per line, trimmed, blank lines skipped, duplicates in any case dropped');
    eq(ctx.bannedPhraseList(Array.from({ length: 450 }, (_, i) => 'word' + i).join('\n')).length, 200, 'capped at 200 phrases');
    eq(ctx.bannedPhraseList(['not', 'a', 'string']), [], 'a list that is not text (a malformed import) is empty, not an error');
  }
}

console.log('\nB. Picking the token');
{
  const { ctx } = load();
  ok(has(ctx, 'bannedTokenFor'), 'the block has bannedTokenFor');
  if (has(ctx, 'bannedTokenFor')) {
    const f = (v) => ctx.bannedTokenFor(v, tokenize(v));
    eq(f(' suddenly'), { id: idOf(' suddenly'), piece: ' suddenly', exact: true }, 'a word that is one token is banned at that token');
    eq(f('Chuckle'), null, '"Chuckle" -> [Ch][uckle]: "Ch" is a fragment other words share, so nothing is banned');
    eq(f(' Chuckle'), null, '" Chuckle" -> [ Ch][uckle]: likewise');
    eq(f('SUDDENLY'), null, 'a word spelled out a letter at a time is left alone');
    eq(f(' chuckling'), { id: idOf(' chuck'), piece: ' chuck', exact: false },
      '" chuckling" -> [ chuck]...: a leading piece of 4+ letters and half the word is banned');
    eq(f(' whatever'), null, '" whatever" -> [ what][ever]: a leading piece that is a common word is never banned');
    eq(f(' Elara'), null, '" Elara" -> [ El][ara]: too short a piece');
    eq(f(' tapestry'), null, '" tapestry" -> [ tape]...: a piece of only half the word is a word of its own, so not banned');
    eq(ctx.bannedTokenFor(' suddenly', [{ id: 7, piece: ' ' }, { id: idOf(' suddenly'), piece: ' suddenly' }]),
      { id: idOf(' suddenly'), piece: ' suddenly', exact: true }, 'a space-only piece in front (SentencePiece) is skipped');
    eq(ctx.bannedTokenFor(' ', [{ id: 7, piece: ' ' }]), null, 'a lone space token is never banned');
    eq(ctx.bannedTokenFor(' suddenly', [{ id: 5, piece: null }]), { id: 5, piece: null, exact: true },
      'no pieces (an older engine): a single token is still the word');
    eq(ctx.bannedTokenFor(' Chuckle', [{ id: 5, piece: null }, { id: 6, piece: null }]), null,
      'no pieces and several tokens: nothing is known to be safe, so nothing is banned');
    eq(ctx.bannedTokenFor(' x', []), null, 'no tokens: nothing');
  }
}

console.log('\nC. buildLogitBias');
{
  const env = load();
  const sudden = await biasOf(env, 'suddenly');
  eq(pieces(sudden), [' suddenly', 'suddenly', ' Suddenly', 'Suddenly'].sort((a, b) => idOf(a) - idOf(b)),
    '"suddenly" bans its four one-token forms');
  ok(Object.values(sudden).every(v => v === -20), 'at the default strength, -20', JSON.stringify(sudden));

  const chuck = await biasOf(env, 'Chuckle');
  eq(pieces(chuck), [' chuckle'], '"Chuckle" bans " chuckle", the one form that is a whole token');
  ok(!(idOf('Ch') in chuck) && !(idOf(' Ch') in chuck) && !(idOf('ch') in chuck) && !(idOf('uckle') in chuck),
    'and none of [Ch], [ Ch], [ch], [uckle]: lines starting "Ch" are untouched', JSON.stringify(pieces(chuck)));

  const spine = await biasOf(env, 'shivers down her spine');
  eq(pieces(spine), [' shivers', 'shivers', ' Shivers', 'Shivers'],
    '"shivers down her spine" is stopped at "shivers"');
  ok(!(idOf(' her') in spine) && !(idOf(' down') in spine) && !(idOf(' spine') in spine),
    'and " her", " down", " spine" stay free in ordinary text');

  const arms = await biasOf(env, 'goosebumps along her arms');
  eq(pieces(arms), [' arms', 'arms', ' Arms', 'Arms'],
    'when the rarest words split too finely ("goosebumps", "along"), the phrase is stopped at the next one, "arms"');
  eq(await env.ctx.buildLogitBias({ bannedPhrases: 'ministrations galore' }), {}, 'and when none of its words can be banned, nothing is');

  eq(await env.ctx.buildLogitBias({ bannedPhrases: 'of the\n— …' }), {}, 'a list with nothing safe to ban sends no logit_bias at all');
  eq(await env.ctx.buildLogitBias({ bannedPhrases: '   ' }), {}, 'an empty list sends nothing');
  eq(await env.ctx.buildLogitBias({}), {}, 'a card without the field sends nothing');
  let threw = null;
  try { eq(await env.ctx.buildLogitBias({ bannedPhrases: ['suddenly'] }), {}, 'a list that is not text sends nothing'); }
  catch (e) { threw = e; }
  ok(!threw, 'and does not throw (it would fail the whole turn)', threw && threw.message);

  const outright = await biasOf(env, 'suddenly', -100);
  ok(Object.keys(outright).length === 4 && Object.values(outright).every(v => v === false),
    'at -100, every banned token is false: llama-server never produces it', JSON.stringify(outright));
  ok(Object.values(await biasOf(env, 'suddenly', -35)).every(v => v === -35), 'in between, the slider value as it is');
  ok(Object.values(await biasOf(env, 'suddenly', 50)).every(v => v === -1),
    'a positive strength (an imported card) is pulled back to -1: a ban never promotes its words');
  ok(Object.values(await biasOf(env, 'suddenly', -500)).every(v => v === false), 'below -100 is "ban outright" too');
  ok(Object.values(await biasOf(env, 'suddenly', 'abc')).every(v => v === -20), 'an unreadable strength is the default, not a hard ban');

  const env2 = load();
  await biasOf(env2, 'suddenly\nshivers down her spine');
  ok(env2.calls.length > 0 && env2.calls.every(c => c.url === 'http://127.0.0.1:9/llm/tokenize'),
    '/tokenize on the local engine, nowhere else');
  ok(env2.calls.every(c => c.body.with_pieces === true && c.body.parse_special === false),
    'asks for pieces, and never parses special-token text', JSON.stringify(env2.calls[0] && env2.calls[0].body));
  ok(env2.calls.every(c => /^ ?[\p{L}\p{N}'’-]+$/u.test(c.body.content)),
    'only single words are tokenized, never the whole phrase', JSON.stringify(env2.calls.map(c => c.body.content)));
  ok(!env2.calls.some(c => /\b(her|down)\b/i.test(c.body.content)), 'the common words of a phrase are never even tokenized');
}

console.log('\nD. Engines');
{
  const spm = load({ mode: 'spm' });
  eq(pieces(await biasOf(spm, 'suddenly')), [' suddenly', 'suddenly', ' Suddenly', 'Suddenly'].sort((a, b) => idOf(a) - idOf(b)),
    'a SentencePiece-style engine that puts a space token in front: the same four bans, the space token never');

  const legacy = load({ mode: 'legacy' });
  const lc = await biasOf(legacy, 'Chuckle\nsuddenly');
  ok(idOf(' chuckle') in lc && idOf(' suddenly') in lc, 'an older engine without pieces: words that are one token are still banned');
  ok(!(idOf('Ch') in lc) && !(idOf('uckle') in lc), 'and a word that splits bans nothing, since nothing is known to be safe');

  for (const mode of ['down', 'throw']) {
    const env = load({ mode });
    eq(await env.ctx.buildLogitBias({ bannedPhrases: 'suddenly' }), {}, `/tokenize ${mode === 'down' ? 'answering 503' : 'unreachable'}: no logit_bias, the turn goes on`);
    ok(env.logs.warn.length === 1 && env.calls.length <= 6, 'one warning in the console, and it stops asking', `warns ${env.logs.warn.length}, calls ${env.calls.length}`);
  }
}

console.log('\nE. The cache');
{
  const env = load();
  await biasOf(env, 'suddenly\nshivers down her spine');
  const first = env.calls.length;
  ok(first === 12, 'first message: one /tokenize per form of each word (6 + 6)', `calls ${first}`);
  await biasOf(env, 'suddenly\nshivers down her spine');
  await biasOf(env, 'suddenly\nshivers down her spine');
  ok(env.calls.length === first, 'later messages: no /tokenize at all', `calls ${env.calls.length}`);
  await biasOf(env, 'suddenly\nshivers down her spine\na mix of');
  ok(env.calls.length === first + 6, 'a word added to the list costs only its own forms', `calls ${env.calls.length - first}`);
  env.ctx._currentModelFile = 'other-model.gguf';
  await biasOf(env, 'suddenly');
  ok(env.calls.length === first + 12, 'another model is asked again: token ids belong to a model', `calls ${env.calls.length - first}`);

  const down = load({ mode: 'down' });
  await biasOf(down, 'suddenly');
  const n = down.calls.length;
  await biasOf(down, 'suddenly');
  ok(down.calls.length > n, 'a failure is not cached: the next message tries again');
}

console.log('\nF. CHECK in the editor');
{
  function editor(env, text, strength) {
    const readout = { _t: '', html: false };
    Object.defineProperty(readout, 'textContent', { get() { return this._t; }, set(v) { this._t = String(v); } });
    Object.defineProperty(readout, 'innerHTML', { get() { return this._t; }, set(v) { this.html = true; this._t = String(v); } });
    env.els['banned-phrases-readout'] = readout;
    env.els['card-banned-phrases'] = { value: text };
    env.els['card-logit-strength'] = { value: String(strength ?? -20) };
    return readout;
  }
  const env = load();
  ok(has(env.ctx, 'checkBannedPhrases'), 'the block has checkBannedPhrases');
  if (has(env.ctx, 'checkBannedPhrases')) {
    let r = editor(env, '  ');
    await env.ctx.checkBannedPhrases();
    ok(/Nothing to check/.test(r.textContent), 'an empty list says so', r.textContent);

    r = editor(env, 'suddenly\nChuckle\nshivers down her spine\nof the\nElara\n<img src=x onerror=alert(1)>\ngoosebumps along her arms\nministrations galore');
    await env.ctx.checkBannedPhrases();
    const lines = r.textContent.split('\n');
    ok(/^\d+ token\(s\) pushed down by -20 with fake\.gguf:$/.test(lines[0]), 'the first line gives the count, the strength and the model', lines[0]);
    ok(lines[0].startsWith('13 token(s)'), 'counted once each: 4 + 1 + 4 + 4', lines[0]);
    ok(lines.some(l => l.startsWith('suddenly — banned: " suddenly"')), 'a word lists the pieces it bans, quoted', r.textContent);
    ok(lines.some(l => /^Chuckle — banned: " chuckle"\. Left alone in 5 form\(s\) that split too finely, e\.g\. " Chuckle"\.$/.test(l)),
      'a word that splits says which forms it is left alone in', lines.find(l => l.startsWith('Chuckle')));
    ok(lines.some(l => l.startsWith('shivers down her spine — stopped at "shivers", everywhere it appears, not only in this phrase:')),
      'a phrase says which word it is stopped at, and that the word goes everywhere');
    ok(lines.some(l => l === 'of the — not banned: every word in it is too common to ban without changing ordinary text.'),
      'a phrase of common words says it is not banned, and why');
    ok(lines.some(l => l.startsWith('Elara — not banned: "Elara" splits into pieces too small')), 'a word with nothing safe to ban says so');
    ok(lines.some(l => l.startsWith('goosebumps along her arms — stopped at "arms" ("goosebumps", "along" split too finely), everywhere it appears')),
      'a phrase stopped at a later word names the ones it passed over', lines.find(l => l.startsWith('goosebumps')));
    ok(lines.some(l => l === 'ministrations galore — not banned: none of "ministrations", "galore" can be banned safely with this model; each splits into pieces too small or too common.'),
      'a phrase with no word to stop at says which words were tried', lines.find(l => l.startsWith('ministrations')));
    ok(!r.html && r.textContent.includes('<img src=x onerror=alert(1)>'), 'the readout is text: a phrase with markup is shown, never parsed');

    r = editor(env, 'suddenly', -100);
    await env.ctx.checkBannedPhrases();
    ok(r.textContent.startsWith('4 token(s) banned outright with fake.gguf:'), 'at -100 it says "banned outright"', r.textContent.split('\n')[0]);
  }
  const down = load({ mode: 'down' });
  if (has(down.ctx, 'checkBannedPhrases')) {
    const r = editor(down, 'suddenly');
    await down.ctx.checkBannedPhrases();
    ok(/^Can’t check: no model is answering \(\/tokenize returned 503\)\. Load a model and try again\.$/.test(r.textContent),
      'no engine: says it cannot check, and why', r.textContent);
  }
  let ta;
  const moving = load({ onFetch: () => { ta.value = 'something else'; } });
  if (has(moving.ctx, 'checkBannedPhrases')) {
    const r = editor(moving, 'suddenly');
    ta = moving.els['card-banned-phrases'];
    await moving.ctx.checkBannedPhrases();
    eq(r.textContent, 'The list changed while checking. Press CHECK again.', 'a result for a list no longer in the editor is not shown');
  }
}

console.log('\nG. The page, the request sites, the docs');
{
  const group = (HTML.match(/<label for="card-banned-phrases">[\s\S]*?<\/div>\s*<\/div>/) || [''])[0];
  ok(/<button[^>]*onclick="checkBannedPhrases\(\)"[^>]*>CHECK<\/button>/.test(group), 'the editor has a CHECK button beside the list');
  ok(/id="banned-phrases-readout"/.test(group), 'and a readout under it');
  ok(/single words work best/.test(group) && /rarest word/.test(group), 'the hint says it works per token: single words best, a phrase at its rarest word');
  ok(!/8 variants|multi-token decomposition|streaming filter/.test(HTML), 'the old hint (8 variants, "use a streaming filter") is gone');
  ok(/<span>-100 Ban outright<\/span>/.test(HTML) && /banned outright: the model never produces those tokens/.test(HTML),
    'the slider\'s end says "Ban outright", and means it');
  ok(!/Near-ban|near-ban/.test(HTML), '"near-ban" is gone');

  const edit = between(CARDS, 'function editCard(', '\nfunction ') || '';
  ok(/getElementById\('banned-phrases-readout'\)[\s\S]{0,120}textContent = ''/.test(edit), 'opening a card clears the last CHECK');

  const sites = CHAT.match(/const logitBiasParams = await buildLogitBias\(card\);[\s\S]{0,1200}?\.\.\.logitBiasParams \}/g) || [];
  ok(sites.length === 2, 'both chat request sites (send, and regenerate) still send it', `sites ${sites.length}`);

  const informant = JSON.parse(read('default-characters.json')).find(c => /Informant/.test(c.name || '')) || {};
  const style = String(informant.writingStyle || '');
  ok(/Banned Phrases:/.test(style) && !/Banned Phrases: currently bugged/.test(style), 'the built-in guide character no longer calls it bugged');
  const readme = read('README.md');
  ok(!/\*\*Logit bias is broken\*\*/.test(readme), 'README: no longer listed as a known bug');
  ok(!/appears to be bugged at this time/.test(readme), 'README: the feature list no longer calls it bugged');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
