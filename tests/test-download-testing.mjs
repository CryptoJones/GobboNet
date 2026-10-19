/**
 * In-app downloads, live for testing -- roadmap 1.7.7 item #8b.
 *
 * Held off from 1.7 (button: NOT WORKING YET, disabled). For 1.7.7 the
 * maintainer chose to make it live so it can be tested on real installs,
 * still not listed as working. A first attempt hid it behind ?test-downloads
 * in the address; the maintainer clicked the button, nothing happened, and
 * there was nothing to test. So:
 *
 *   A. The button works, from the PC or a phone, with no switch to find.
 *   B. While it is not confirmed working, the box says so, and says where to
 *      look when a download fails.
 *   C. The hold still works as one flag, so the path can be taken out of use
 *      again -- and no hidden switch is left behind.
 *   D. A failed download says which step it stopped at.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const MODEL_JS = fs.readFileSync(ROOT + '/js/02-model.js', 'utf8');
const block = MODEL_JS.slice(MODEL_JS.indexOf('/* ================================================================\n   MODEL CATALOGUE'));

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};
const eq = (a, b, l) => ok(a === b, l, a === b ? '' : `got:  ${JSON.stringify(a)}\n      want: ${JSON.stringify(b)}`);

function page({ served = true, poll = { state: 'idle' }, search = '' } = {}) {
  const els = {};
  const el = (id) => (els[id] = els[id] || {
    id, style: {}, textContent: '', disabled: false, title: '', classList: { add() {}, remove() {} },
  });
  ['model-catalog-modal', 'model-catalog-list', 'model-catalog-free', 'model-catalog-progress',
   'model-catalog-dl-title', 'model-catalog-bar', 'model-catalog-log', 'model-catalog-note',
   'model-catalog-swap-row', 'add-model-unavailable'].forEach(el);
  const btn = el('add-model-button');
  els['add-model-block'] = { id: 'add-model-block', querySelector: () => btn };
  let opened = 0;
  els['model-catalog-modal'].classList = { add: () => { opened++; }, remove() {} };
  const ctx = {
    console: { error() {}, log() {}, warn() {} },
    window: { location: { search } },
    document: { getElementById: (id) => els[id] || null, createElement: () => el('x' + Math.random()) },
    IS_SERVED: served,
    fetch: async (url) => ({ ok: true, status: 200, json: async () =>
      (String(url).startsWith('/catalog.json') ? { models: [] } : poll) }),
    setInterval: () => 1, clearInterval() {},
  };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(block + '\nglobalThis.__hold = () => { MODEL_CATALOG_ENABLED = false; };' +
                  '\nglobalThis.__confirm = () => { MODEL_CATALOG_CONFIRMED = true; };', ctx);
  return { ctx, els, btn, opened: () => opened };
}

console.log('\n=== A. the button works, with nothing to find first ===');
{
  ok(/^let MODEL_CATALOG_ENABLED = true;$/m.test(MODEL_JS), 'the hold is lifted in the source');
  const p = page();
  p.ctx.applyModelCatalogAvailability();
  eq(p.btn.textContent, 'BROWSE CATALOGUE', 'the button reads BROWSE CATALOGUE');
  eq(p.btn.disabled, false, 'and is enabled');
  p.ctx.openModelCatalog();
  eq(p.opened(), 1, 'clicking it opens the catalogue');
}

console.log('\n=== B. not listed as working, and it says so ===');
{
  ok(/^let MODEL_CATALOG_CONFIRMED = false;$/m.test(MODEL_JS), 'it is marked unconfirmed in the source');
  const p = page();
  p.ctx.applyModelCatalogAvailability();
  const note = p.els['add-model-unavailable'];
  eq(note.style.display, '', 'the note under the button is shown');
  ok(/^Being tested — not confirmed working yet\./.test(note.textContent), 'and says it is not confirmed working', note.textContent);
  ok(/names the step it stopped at/.test(note.textContent) && /\[download\] lines/.test(note.textContent),
     'and where to look when a download fails');

  const done = page();
  done.ctx.__confirm();
  done.ctx.applyModelCatalogAvailability();
  ok(done.els['add-model-unavailable'].style.display === 'none', 'once confirmed, the note goes');

  const file = page({ served: false });
  file.ctx.applyModelCatalogAvailability();
  eq(file.btn.style.display, 'none', 'opened from a file there is still nothing to download with');
  ok(/needs the GobboNet server/.test(file.els['add-model-unavailable'].textContent), 'and it says so');
}

console.log('\n=== C. one flag still takes it out of use, and no hidden switch remains ===');
{
  const p = page({ search: '?test-downloads' });
  p.ctx.__hold();
  p.ctx.applyModelCatalogAvailability();
  eq(p.btn.textContent, 'NOT WORKING YET', 'with the hold back on, the button says NOT WORKING YET');
  eq(p.btn.disabled, true, 'and is disabled');
  p.ctx.openModelCatalog();
  eq(p.opened(), 0, 'and nothing opens -- an old ?test-downloads address does not lift it');
  ok(!/test-downloads/.test(MODEL_JS), 'the address switch is gone from the code');
}

console.log('\n=== D. a failure names its step ===');
{
  for (const [stage, words] of [['folder', 'checking the models folder'], ['connect', 'contacting Hugging Face'],
                                ['download', 'downloading'], ['move', 'moving it into place']]) {
    const p = page({ poll: { state: 'error', stage, message: 'why' } });
    await p.ctx.pollModelDownload();
    eq(p.els['model-catalog-dl-title'].textContent, 'Download failed while ' + words, `stage "${stage}"`);
    eq(p.els['model-catalog-log'].textContent, 'why', 'with the server’s explanation underneath');
  }
  const old = page({ poll: { state: 'error', message: 'why' } });
  await old.ctx.pollModelDownload();
  eq(old.els['model-catalog-dl-title'].textContent, 'Download failed', 'an older server with no stage: as before');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
