/**
 * The setup wizard's oversized-model warning -- roadmap 1.7.7 item #8a.
 *
 * Windows has warned for a long time: launch.bat's "Heads up ... Download it
 * anyway?" and the installer's dialog. The Linux wizard only printed "suits N
 * GB VRAM or more". Now it asks first, through the browser's own confirm(),
 * when the engine reports less GPU memory than the model wants.
 *
 * The figure comes from /api/gpu (internal/setup, tested in Go). This covers
 * the page:
 *
 *   A. It asks only when there is a known figure below the model's minimum.
 *      Every other case -- no figure, a failed request, a model with no
 *      minimum -- asks nothing: launch.bat's rule, a failed probe never blocks.
 *   B. Declining sends nothing; accepting downloads.
 *   C. Both wizards carry it: the one the Linux packages serve
 *      (installer-linux/, copied into linux-amd64/) and the one built into
 *      the program (internal/setup/).
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const read = (p) => fs.readFileSync(ROOT + '/' + p, 'utf8');
const WIZARDS = ['installer-linux/wizard.html', 'linux-amd64/wizard.html', 'internal/setup/wizard.html'];

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

const BIG = { index: 5, display: 'Gemma 4 26B-A4B MoE', min_vram: 18 };
const FITS = { index: 1, display: 'Gemma 4 E4B IT', min_vram: 8 };

/* The wizard's own confirmFit and b3 handler, run against a stand-in DOM and
   server. */
function harness(file, { gpu = { known: true, vram_gb: 8, device: 'NVIDIA GeForce RTX 4060' },
                         gpuStatus = 200, answer = false, picked = BIG } = {}) {
  const src = read(file);
  const fit = between(src, 'function confirmFit(m)', '$("b3").onclick', file);
  const b3 = between(src, '$("b3").onclick', '\n  };', file) + '\n  };';
  const els = {};
  const el = (id) => (els[id] = els[id] || { id, disabled: false, textContent: '', style: {}, classList: { toggle() {} } });
  const calls = { confirm: [], gpu: 0, posts: [], shown: [], failed: [] };
  const ctx = {
    console, Promise, JSON,
    window: { confirm: (msg) => { calls.confirm.push(msg); return answer; } },
    $: el,
    state: { models: [BIG, FITS] },
    pick: picked ? picked.index : null,
    poll: null, setInterval: () => 1, tick() {},
    fail: (m) => calls.failed.push(m),
    show: (n) => calls.shown.push(n),
    fetch: async (path) => {
      if (path !== '/api/gpu') throw new Error('unexpected GET ' + path);
      calls.gpu++;
      if (gpu === 'throw') throw new Error('offline');
      return { ok: gpuStatus < 300, json: async () => gpu };
    },
    post: async (path, body) => { calls.posts.push([path, body]); return { ok: true }; },
  };
  vm.createContext(ctx);
  vm.runInContext(fit + '\n' + b3, ctx);
  const click = async () => { ctx.$('b3').onclick(); await new Promise(r => setTimeout(r, 20)); };
  return { ctx, calls, click, els };
}

for (const file of WIZARDS) {
  console.log(`\n=== ${file} ===`);

  /* A. when it asks */
  {
    let h = harness(file);
    await h.click();
    eq(h.calls.confirm.length, 1, 'A model wanting 18 GB on an 8 GB card: asks');
    eq(h.calls.confirm[0],
       'Gemma 4 26B-A4B MoE wants about 18 GB of GPU memory, but only 8 GB was detected (NVIDIA GeForce RTX 4060).\n\n' +
       'It may run far more slowly than a model that fits: whatever does not fit on the GPU runs from system memory instead.\n\n' +
       'Download it anyway?',
       'in launch.bat’s words, with the card named');

    h = harness(file, { picked: FITS });
    await h.click();
    eq(h.calls.confirm.length, 0, 'A model that fits (8 GB on 8 GB): asks nothing');
    eq(h.calls.posts.length, 1, 'and downloads');

    for (const [label, opts] of [
      ['no figure (no GPU, integrated only, failed probe)', { gpu: { known: false, vram_gb: 0, note: 'the engine lists no GPU' } }],
      ['the GPU question failing outright', { gpu: 'throw' }],
      ['an older setup server without /api/gpu', { gpuStatus: 404, gpu: { error: 'Not found' } }],
    ]) {
      h = harness(file, opts);
      await h.click();
      ok(h.calls.confirm.length === 0 && h.calls.posts.length === 1, label + ': asks nothing, downloads',
         JSON.stringify(h.calls));
    }

    h = harness(file, { picked: { index: 9, display: 'No minimum', min_vram: 0 } });
    h.ctx.state.models.push({ index: 9, display: 'No minimum', min_vram: 0 });
    await h.click();
    ok(h.calls.gpu === 0 && h.calls.posts.length === 1, 'A model with no minimum: does not even ask the server');
  }

  /* B. the answer */
  {
    let h = harness(file, { answer: false });
    await h.click();
    eq(h.calls.posts.length, 0, 'Declined: nothing is downloaded');
    eq(h.calls.shown.length, 0, 'and the wizard stays on the model step');
    eq(h.els.b3.disabled, false, 'with Download clickable again');

    h = harness(file, { answer: true });
    await h.click();
    eq(JSON.stringify(h.calls.posts), '[["/api/download",{"index":5}]]', 'Accepted: that model downloads');
    eq(h.calls.shown[0], 4, 'and the wizard moves to the download step');

    h = harness(file, { picked: null });
    h.ctx.pick = null;
    await h.click();
    ok(h.calls.failed.some(m => /Choose a model first/.test(m)) && h.calls.gpu === 0, 'No model picked: says so, asks nothing');
  }
}

/* C. the copies */
console.log('\n=== the copies ===');
eq(read('installer-linux/wizard.html'), read('linux-amd64/wizard.html'), 'linux-amd64/wizard.html is still an exact copy of installer-linux/');
for (const file of WIZARDS) {
  const src = read(file);
  ok(/window\.confirm\(/.test(between(src, 'function confirmFit(m)', '$("b3").onclick', file)) &&
     !/<dialog|class="modal/.test(src), `${file}: the browser’s own confirm(), no new dialog design`);
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
