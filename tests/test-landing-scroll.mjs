/**
 * The home screen keeps its scroll position — roadmap 1.7.7 item #2.
 *
 * The report: scroll down a character list on the home screen, leave it, and
 * it snaps back to the top by itself. The cause is that the landing page is
 * not drawn once. js/24-boot.js re-renders it every five seconds to refresh
 * the connection pill, and renderMessages() rebuilds it with innerHTML, which
 * creates every scroll box afresh at 0.
 *
 * The fix came with the report and is kept: read every position before the
 * rebuild, put it back after. Two things were added here, and both are
 * pinned below:
 *
 *   - the SCHEDULED TASKS list is a scroller too (240 px, desktop) and had
 *     the same bug, so it is restored alongside the character lists;
 *   - the fix is merged INTO the block rather than pasted over it. The
 *     reported block ended without the home-button lines and the `return`,
 *     so pasted as written it fell through to `thread.name` with no thread
 *     and threw on every tick. Section D holds that.
 *
 * SECTION A proves the fake can fail: a rebuild with no restore loses every
 * position. A fake that kept them anyway would pass a broken fix.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const SRC = fs.readFileSync(ROOT + '/js/13-dashboard.js', 'utf8');
const BOOT = fs.readFileSync(ROOT + '/js/24-boot.js', 'utf8');

// renderMessages() whole: from its declaration to the closing brace at column
// 0. Only its no-thread branch runs here, and that branch returns early, so the
// thread-rendering code after it is compiled but never reached.
const start = SRC.indexOf('function renderMessages()');
const end = start < 0 ? -1 : SRC.indexOf('\n}\n', start);
if (start < 0 || end < 0) {
  console.error('could not locate renderMessages() in js/13-dashboard.js');
  process.exit(1);
}
const RENDER_MESSAGES = SRC.slice(start, end + 2);

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};
const eq = (a, b, l) => ok(a === b, l,
  a === b ? '' : `got:  ${JSON.stringify(a)}\n      want: ${JSON.stringify(b)}`);

/* ================================================================
   A fake #messages

   Assigning innerHTML builds fresh scroll boxes for every scroller class in
   the markup, in document order, each at scrollTop 0 — which is what the
   browser does and is the whole bug. The container's own scrollTop is reset
   too. Chromium happens to keep that one across the rebuild, but nothing
   guarantees it, so the fake takes the stricter view.

   Every scrollTop clamps to its content like a real one: a list that got
   shorter (a character deleted) cannot be restored past its new end.
================================================================ */
function makeScroller(cls, max) {
  const el = { className: cls };
  let top = 0;
  Object.defineProperty(el, 'scrollTop', {
    get: () => top,
    set: (v) => { top = Math.max(0, Math.min(Number(v) || 0, max)); },
  });
  return el;
}

function makeContainer() {
  let html = '';
  let kids = [];
  const c = makeScroller('messages', 3000);
  Object.defineProperty(c, 'innerHTML', {
    get: () => html,
    set: (v) => {
      html = String(v);
      kids = [...html.matchAll(/class="(landing-sched-list|landing-char-list)"/g)]
        .map(m => makeScroller(m[1], 600));
      c.scrollTop = 0;
    },
  });
  const matches = (sel, el) => sel.split(',').map(s => s.trim().replace(/^\./, '')).includes(el.className);
  c.querySelector = (sel) =>
    sel === '.landing-page' ? (html.includes('class="landing-page"') ? {} : null)
                            : (kids.find(k => matches(sel, k)) || null);
  c.querySelectorAll = (sel) => kids.filter(k => matches(sel, k));
  return c;
}

// The three scrollers in the order renderLandingPage() draws them. Section E
// checks the real template still draws them in this order.
const LANDING_HTML =
  '<div class="landing-page">' +
  '<div class="landing-sched-list">s</div>' +
  '<div class="landing-char-list">yours</div>' +
  '<div class="landing-char-list">defaults</div>' +
  '</div>';

function build({ thread = null } = {}) {
  const container = makeContainer();
  const title = { textContent: '' };
  const homeClasses = new Set();
  const homeBtn = { classList: { add: (c) => homeClasses.add(c), remove: (c) => homeClasses.delete(c) } };
  const els = { messages: container, 'thread-title': title, 'home-btn': homeBtn };
  const ctx = {
    console: { log() {}, warn() {}, error() {} },
    document: { getElementById: (id) => els[id] || null },
    getActiveThread: () => thread,
    renderLandingPage: () => LANDING_HTML,
    Math, Date, JSON, Object, Array, String, Number, Map, Set,
  };
  vm.createContext(ctx);
  vm.runInContext(RENDER_MESSAGES, ctx);
  const lists = () => container.querySelectorAll('.landing-sched-list, .landing-char-list').map(l => l.scrollTop);
  return { ctx, container, title, homeClasses, lists, render: () => ctx.renderMessages() };
}

/* ================================================================
   A. THE FAKE CAN FAIL
================================================================ */
console.log('\n=== A. a rebuild with no restore loses every position ===');
{
  const c = makeContainer();
  c.innerHTML = LANDING_HTML;
  c.scrollTop = 400;
  c.querySelectorAll('.landing-sched-list, .landing-char-list').forEach(l => { l.scrollTop = 150; });
  c.innerHTML = LANDING_HTML;
  eq(c.scrollTop, 0, 'the page is back at the top');
  eq(c.querySelectorAll('.landing-sched-list, .landing-char-list').map(l => l.scrollTop).join(','), '0,0,0',
     'every list is back at the top — the bug, reproduced in the fake');
}

/* ================================================================
   B. THE FIX: A RE-RENDER ON THE HOME SCREEN KEEPS EVERY POSITION
================================================================ */
console.log('\n=== B. the five-second re-render keeps every scroll position ===');
{
  const t = build();
  t.render();                                   // first draw
  t.container.scrollTop = 400;
  const [sched, mine, defaults] = t.container.querySelectorAll('.landing-sched-list, .landing-char-list');
  sched.scrollTop = 100; mine.scrollTop = 150; defaults.scrollTop = 30;

  t.render();                                   // the timer's re-render
  eq(t.container.scrollTop, 400, 'the page stays where it was');
  eq(t.lists().join(','), '100,150,30',
     'scheduled tasks, your characters and default characters all stay put');

  t.render(); t.render();                       // and it holds tick after tick
  eq(t.lists().join(','), '100,150,30', 'still put after more ticks');
}

/* ================================================================
   C. ARRIVING FROM A THREAD DOES NOT INHERIT ITS POSITION
================================================================ */
console.log('\n=== C. coming home from a thread starts at the top ===');
{
  const t = build();
  t.container.innerHTML = '<div class="message">a long thread</div>';
  t.container.scrollTop = 2500;                 // deep in a thread
  t.render();                                   // goHome() → landing page
  eq(t.lists().join(','), '0,0,0', 'the lists start at the top, not at the thread’s position');
}

/* ================================================================
   D. MERGED, NOT PASTED: THE BRANCH STILL HIDES HOME AND RETURNS
================================================================ */
console.log('\n=== D. the no-thread branch still hides HOME and returns ===');
{
  const t = build();
  let threw = null;
  try { t.render(); t.render(); } catch (e) { threw = e; }
  ok(threw === null, 'rendering the home screen does not throw',
     threw && ('threw: ' + threw.message + ' — the early return is missing'));
  eq(t.title.textContent, 'GOBBONET', 'the title is GOBBONET');
  ok(t.homeClasses.has('hidden'), 'the HOME button is hidden on the home screen');
}

/* ================================================================
   E. SOURCE GUARDS — WHAT THE INDEX MATCHING RELIES ON
================================================================ */
console.log('\n=== E. the real page and timer match what the fix assumes ===');
{
  const lp = SRC.slice(SRC.indexOf('function renderLandingPage()'), SRC.indexOf('function renderMessages()'));
  const order = [...lp.matchAll(/class="(landing-sched-list|landing-char-list)"/g)].map(m => m[1]);
  eq(order.join(' > '), 'landing-sched-list > landing-char-list > landing-char-list',
     'renderLandingPage() always draws the three scrollers in this order');
  ok(/LANDING_SCROLLERS\s*=\s*'[^']*\.landing-sched-list[^']*'/.test(RENDER_MESSAGES)
     && /LANDING_SCROLLERS\s*=\s*'[^']*\.landing-char-list[^']*'/.test(RENDER_MESSAGES),
     'the restore covers both scroller classes');
  ok(/setInterval\(\(\) => \{[\s\S]{0,1800}?render\(\);[\s\S]{0,80}?\}, 5000\)/.test(BOOT),
     'js/24-boot.js still re-renders the home screen on a timer — the reason this exists');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
