/**
 * Imported threads whose folder is missing -- issue #61.
 *
 * THE BUG
 * -------
 * A thread names its folder by id. The threads export wrote the threads and
 * not the folders, so on another install (or after the folder was deleted) an
 * imported thread pointed at an id that did not exist. renderSidebar drew a
 * thread either inside a folder section or in the unfiled list, and the unfiled
 * list only took threads with NO folderId -- so this one went in neither. The
 * import reported success, re-importing reported it as already there, and the
 * sidebar never showed it.
 *
 * THE FIX, IN TWO LAYERS
 * ----------------------
 *   A. renderSidebar treats a folderId that names no folder here as unfiled.
 *      That rescues anyone already affected, and it is decided at paint time,
 *      so the stored folderId is not rewritten and device sync sees nothing.
 *   B. Thread exports carry the folders their threads sit in, and import
 *      gives each new thread a folder that exists here: a same-named local
 *      folder, else the folder from the file (recreated with its id), else --
 *      for files written before this fix -- the unfiled list.
 *
 * Both run the REAL functions out of js/12-render.js and js/21-data.js.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const DATA = fs.readFileSync(ROOT + '/js/21-data.js', 'utf8');
const RENDER = fs.readFileSync(ROOT + '/js/12-render.js', 'utf8');

const slice = (src, file, from, to) => {
  const a = src.indexOf(from);
  const b = to ? src.indexOf(to, a + 1) : src.length;
  if (a < 0 || b < 0 || b <= a) {
    console.error('could not locate ' + JSON.stringify(from) + ' in ' + file);
    process.exit(1);
  }
  return src.slice(a, b);
};

// Everything from the bulk exporter through the end of importData.
const DATA_BLOCK = slice(DATA, 'js/21-data.js', 'function exportData(',
  '/* ================================================================\n   DATA MANAGER MODAL');
const SIDEBAR = slice(RENDER, 'js/12-render.js', 'function renderSidebar(', 'function filterThreads(');
const PICKER = slice(RENDER, 'js/12-render.js', 'function openFolderPicker(', 'function moveThread(');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};
const eq = (a, b, l) => ok(a === b, l,
  a === b ? '' : `got:  ${JSON.stringify(a)}\n      want: ${JSON.stringify(b)}`);

function freshState() {
  return {
    activeThreadId: 't-camp',
    settings: {}, schedules: [], extensions: [], macros: [], searchEnabled: false,
    characterCards: [], personaCards: [],
    folders: [
      // Open, so its threads are drawn; collapsed: true is checked by the
      // export assertion below, which must not carry it.
      { id: 'f-camp', name: 'Campaigns', collapsed: false },
      { id: 'f-empty', name: 'Empty one', collapsed: false },
    ],
    threads: [
      { id: 't-camp', name: 'Dragon heist', folderId: 'f-camp', pinned: false, messages: [] },
      { id: 't-loose', name: 'Loose chat', folderId: null, pinned: false, messages: [] },
    ],
  };
}

function build(stateObj) {
  const downloads = [];
  const status = [];
  const listEl = { innerHTML: '' };
  const searchEl = { value: '' };
  let popoverHtml = '';
  const ctx = {
    console: { log() {}, warn() {}, error() {} },
    state: stateObj || freshState(),
    Date, JSON, Math, Object, Array, Set, Map, String, Number, Boolean,
    DEFAULT_PERSONA: { id: '', name: 'Anonymous' },
    DEFAULT_CARD: { id: 'default', name: 'Assistant' },
    DEFAULT_SETTINGS: {}, DEFAULT_MACROS: [],
    saveState: () => {}, render: () => {}, applyExtensions: () => {},
    migrateExtensions: (e) => e || [],
    neutralizeUntrustedCode: () => ({ cards: 0, extensions: false }),
    downloadJSON: (data, filename) => { downloads.push({ data: JSON.parse(JSON.stringify(data)), filename }); },
    confirm: () => true,
    setTimeout: () => 0,
    // renderSidebar's collaborators, reduced to what the assertions read.
    sortThreadsByOrder: () => {},
    renderThreadItem: (t) => `<item:${t.id}>`,
    escapeHtml: (s) => String(s),
    escapeJsAttr: (s) => String(s),
    openPopover: (_el, html) => { popoverHtml = html; },
  };
  ctx.document = {
    getElementById: (id) => {
      if (id === 'thread-list') return listEl;
      if (id === 'thread-search') return searchEl;
      const el = { style: {}, className: '' };
      Object.defineProperty(el, 'textContent', { get: () => '', set: (v) => { status.push(v); } });
      return el;
    },
  };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  const stateSource = fs.readFileSync(ROOT + '/js/04-state.js', 'utf8');
  const schema = stateSource.match(/const STATE_SCHEMA_VERSION\s*=\s*\d+;/)[0];
  vm.runInContext(schema + '\n' + DATA_BLOCK + '\n' + SIDEBAR + '\n' + PICKER, ctx);
  return {
    ctx, downloads, status, searchEl,
    get state() { return ctx.state; },
    sidebar() { ctx.renderSidebar(); return listEl.innerHTML; },
    picker(threadId) {
      ctx.openFolderPicker(threadId, { stopPropagation() {}, currentTarget: {} });
      return popoverHtml;
    },
  };
}

/** Run the real importData against an in-memory file. */
function importInto(h, payload, type) {
  const readers = [];
  h.ctx.FileReader = function () {
    const self = this;
    this.readAsText = () => {
      readers.push(() => self.onload({ target: { result: JSON.stringify(payload) } }));
    };
  };
  vm.runInContext('this.FileReader = FileReader;', h.ctx);
  h.ctx.importData({ files: [{ name: 'f.json' }] }, type);
  readers.forEach(fn => fn());
}

/** Which section of the rendered sidebar a thread was drawn in. */
function sectionOf(html, threadId) {
  const item = `<item:${threadId}>`;
  if (!html.includes(item)) return 'NOT DRAWN';
  // Each section renderSidebar emits ends with its threads div and then its own
  // closing tag at four spaces of indentation; the header's nested divs close
  // at six, so this lazy match stops at the right place.
  for (const m of html.matchAll(/<div class="folder-section[^"]*"( data-folder-id="([^"]+)")?>([\s\S]*?)<\/div>\n    <\/div>/g)) {
    if (!m[3].includes(item)) continue;
    return m[2] ? 'folder:' + m[2] : 'pinned';
  }
  return 'unfiled';   // the unfiled list is emitted outside any folder-section
}

/* ================================================================
   A. THE SIDEBAR DRAWS A THREAD WHOSE FOLDER IS MISSING
================================================================ */
console.log('\n=== A. a folderId with no folder here is unfiled, not invisible ===');
{
  const st = freshState();
  st.threads.push({ id: 't-orphan', name: 'From the other PC', folderId: 'seh7sr2umretz7', pinned: false, messages: [] });
  const h = build(st);
  const html = h.sidebar();

  ok(html.includes('<item:t-orphan>'), 'the orphaned thread is drawn (it was not, before #61)');
  eq(sectionOf(html, 't-orphan'), 'unfiled', 'in the unfiled list');
  eq(sectionOf(html, 't-camp'), 'folder:f-camp', 'a thread whose folder exists stays in that folder');
  eq(sectionOf(html, 't-loose'), 'unfiled', 'a thread with no folder is still unfiled');
  eq(html.split('<item:t-orphan>').length - 1, 1, 'and is drawn exactly once');
  eq(h.state.threads.find(t => t.id === 't-orphan').folderId, 'seh7sr2umretz7',
     'the stored folderId is not rewritten (device sync sees no change)');

  // Search still finds it, and the "no results" line does not fire over it.
  h.searchEl.value = 'other pc';
  const found = h.sidebar();
  ok(found.includes('<item:t-orphan>'), 'a search matching it shows it');
  ok(!found.includes('No threads match'), 'and does not claim nothing matched');

  // If the folder turns up later, the thread goes straight back into it.
  h.searchEl.value = '';
  h.state.folders.push({ id: 'seh7sr2umretz7', name: 'Arrived late', collapsed: false });
  eq(sectionOf(h.sidebar(), 't-orphan'), 'folder:seh7sr2umretz7',
     'when its folder appears, the thread is drawn inside it again');
}

console.log('\n=== A2. pinned orphans and the folder picker ===');
{
  const st = freshState();
  st.threads.push({ id: 't-pin', name: 'Pinned orphan', folderId: 'gone', pinned: true, messages: [] });
  st.threads.push({ id: 't-orphan', name: 'Orphan', folderId: 'gone', pinned: false, messages: [] });
  const h = build(st);
  const html = h.sidebar();
  eq(html.split('<item:t-pin>').length - 1, 1, 'a pinned orphan is drawn once (in PINNED, not also unfiled)');

  const orphanMenu = h.picker('t-orphan');
  ok(/pop-item pop-item-active"[^>]*__none__/.test(orphanMenu),
     '"Move to folder" marks an orphan as "No folder"', orphanMenu);
  const filedMenu = h.picker('t-camp');
  ok(!/pop-item-active"[^>]*__none__/.test(filedMenu) && /pop-item-active"[^>]*f-camp/.test(filedMenu),
     'and still marks a filed thread\'s real folder');
}

/* ================================================================
   B. EXPORTS CARRY THE FOLDERS THEIR THREADS SIT IN
================================================================ */
console.log('\n=== B. thread exports carry their folders ===');
{
  const h = build();
  h.ctx.exportData('threads');
  const bulk = h.downloads[0].data;
  ok(Array.isArray(bulk.folders), 'the bulk threads export has a folders list');
  eq(JSON.stringify(bulk.folders), JSON.stringify([{ id: 'f-camp', name: 'Campaigns' }]),
     'holding only folders a thread is in, as id + name (no collapsed state)');

  h.ctx.exportThread('t-camp');
  const one = h.downloads[1].data;
  eq(JSON.stringify(one.folders), JSON.stringify([{ id: 'f-camp', name: 'Campaigns' }]),
     'a single-thread export carries that thread\'s folder');

  h.ctx.exportThread('t-loose');
  eq(JSON.stringify(h.downloads[2].data.folders), '[]', 'and an unfiled thread carries none');
}

/* ================================================================
   C. IMPORT GIVES EACH NEW THREAD A FOLDER THAT EXISTS HERE
================================================================ */
console.log('\n=== C1. the reporter\'s steps: export, delete thread + folder, import ===');
{
  const h = build();
  h.ctx.exportThread('t-camp');
  const file = h.downloads[0].data;

  // Delete the thread and its folder (what deleteFolder + delete thread do).
  h.state.threads = h.state.threads.filter(t => t.id !== 't-camp');
  h.state.folders = h.state.folders.filter(f => f.id !== 'f-camp');

  importInto(h, file, 'threads');
  const t = h.state.threads.find(t => t.id === 't-camp');
  ok(!!t, 'the thread is imported');
  const f = h.state.folders.find(f => f.id === 'f-camp');
  ok(!!f && f.name === 'Campaigns', 'its folder is recreated with the same id and name');
  eq(sectionOf(h.sidebar(), 't-camp'), 'folder:f-camp', 'and the thread shows up inside it');
  ok(h.status.some(s => s.includes('Recreated 1 folder')), 'the status line says the folder was recreated',
     JSON.stringify(h.status));
}

console.log('\n=== C2. a folder with the same name already exists here ===');
{
  // The reporter's step 4: they made a new "Campaigns" folder by hand first.
  const src = build();
  src.ctx.exportThread('t-camp');
  const file = src.downloads[0].data;

  const st = freshState();
  st.threads = [];
  st.folders = [{ id: 'f-handmade', name: 'Campaigns', collapsed: false }];
  const h = build(st);
  importInto(h, file, 'threads');
  eq(h.state.threads[0].folderId, 'f-handmade', 'the thread goes into the same-named local folder');
  eq(h.state.folders.length, 1, 'and no duplicate "Campaigns" folder is created');
}

console.log('\n=== C3. a file written before this fix (no folders key) ===');
{
  const st = freshState();
  st.threads = [];
  st.folders = [];
  const h = build(st);
  importInto(h, {
    gobbonet_export: 'threads', version: 1,
    threads: [{ id: 't-old', name: 'Old export', folderId: 'seh7sr2umretz7', messages: [] }],
  }, 'threads');
  eq(h.state.threads[0].folderId, null, 'the thread lands in the unfiled list');
  eq(h.state.folders.length, 0, 'no folder is invented for it');
  ok(h.sidebar().includes('<item:t-old>'), 'and it is drawn');
  ok(h.status.some(s => s.includes('1 thread(s) had no matching folder')),
     'the status line says why it is unfiled', JSON.stringify(h.status));
}

console.log('\n=== C4. folders that already exist, shared folders, and untouched threads ===');
{
  const st = freshState();
  // A thread already here with a dangling folderId: import must not rewrite it.
  st.threads.push({ id: 't-here', name: 'Already here', folderId: 'dangling', messages: [] });
  const h = build(st);
  importInto(h, {
    gobbonet_export: 'threads', version: 1,
    folders: [{ id: 'f-camp', name: 'Campaigns' }, { id: 'f-new', name: 'One-shots' }],
    threads: [
      { id: 'n1', name: 'Same folder id', folderId: 'f-camp', messages: [] },
      { id: 'n2', name: 'New folder A', folderId: 'f-new', messages: [] },
      { id: 'n3', name: 'New folder B', folderId: 'f-new', messages: [] },
      { id: 't-here', name: 'Duplicate of a thread already here', folderId: 'f-new', messages: [] },
    ],
  }, 'threads');
  const byId = Object.fromEntries(h.state.threads.map(t => [t.id, t]));
  eq(byId.n1.folderId, 'f-camp', 'a thread whose folder exists here by id keeps it');
  eq(byId.n2.folderId, 'f-new', 'a missing folder is recreated');
  eq(byId.n3.folderId, 'f-new', 'and a second thread from it lands beside the first');
  eq(h.state.folders.filter(f => f.id === 'f-new').length, 1, 'the folder is created once, not per thread');
  eq(byId['t-here'].folderId, 'dangling', 'a thread already here is not touched by the import');
  eq(h.state.threads.length, 6, 'the duplicate thread id is still skipped as before');
}

console.log('\n=== C5. malformed folder records are ignored, not trusted ===');
{
  const st = freshState();
  st.threads = [];
  st.folders = [];
  const h = build(st);
  importInto(h, {
    gobbonet_export: 'threads', version: 1,
    folders: [null, { id: 'f-x' }, { id: 'f-y', name: '   ' }, { name: 'No id' }, 'junk'],
    threads: [
      { id: 'a', name: 'Folder record without a name', folderId: 'f-x', messages: [] },
      { id: 'b', name: 'Folder record with a blank name', folderId: 'f-y', messages: [] },
    ],
  }, 'threads');
  eq(h.state.folders.length, 0, 'no folder is created from a nameless or blank record');
  eq(h.state.threads.map(t => t.folderId).join(','), ',', 'those threads go to the unfiled list');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
