/**
 * The "older history" notice no longer covers the message box.
 *
 * Reported against 1.7.8: "a button in the lower right of the screen saying
 * 'older history / storage notice - open data' ... sitting on top of a few
 * things like the send button and text entry field". It was a fixed button at
 * bottom:12px right:12px, z-index 1000, and the older copies it points at are
 * kept for good, so it came back on every load.
 *
 * Now it is a bar under the chat header, in the column's own flow, with a
 * button to DATA and an x that hides it until what it reports changes.
 *
 *   A. the page: where the bar is, and that nothing places it over the chat
 *   B. js/21-data.js: when it shows, what it says, hiding it, and coming back
 * The layout is checked in real Chromium as well (see the changelog).
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';
import vm from 'vm';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const read = (p) => fs.readFileSync(ROOT + '/' + p, 'utf8');
const HTML = read('chat.html');
const CSS = fs.readdirSync(ROOT + '/css').map(f => read('css/' + f)).join('\n');
const DATA = read('js/21-data.js');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};

console.log('\nA. The page');
{
  const tag = (HTML.match(/<[^>]*id="upgrade-recovery-notice"[^>]*>/) || [''])[0];
  ok(tag.length > 0, 'the notice is still on the page');
  ok(!/position\s*:\s*(fixed|absolute)/.test(tag) && !/bottom\s*:|z-index/.test(tag), 'it is not positioned over the page any more', tag);
  ok(/class="recovery-notice"/.test(tag) && /role="status"/.test(tag), 'it is the recovery bar, announced as a status');
  const chat = HTML.indexOf('<main id="chat">'), header = HTML.indexOf('class="chat-header"', chat),
        bar = HTML.indexOf('id="upgrade-recovery-notice"'), messages = HTML.indexOf('id="messages"'),
        input = HTML.indexOf('class="input-area"');
  ok(chat > 0 && chat < header && header < bar && bar < messages && messages < input,
    'in the chat column, after the header and before the conversation and the message box', [chat, header, bar, messages, input]);
  const body = HTML.slice(bar, HTML.indexOf('</div>', bar));
  ok(/onclick="openDataManager\(\)"[^>]*>OPEN DATA</.test(body), 'it opens DATA');
  ok(/onclick="dismissUpgradeRecoveryNotice\(\)"/.test(body) && /aria-label="Hide this notice"/.test(body), 'and has an x to hide it');
  ok(/#data-btn\.has-recovery::after\s*\{/.test(CSS), 'DATA\'s flag is a dot drawn by CSS, not a longer label');
  const rule = (CSS.match(/\.recovery-notice\s*\{[^}]*\}/) || [''])[0];
  ok(rule && !/position\s*:\s*(fixed|absolute|sticky)/.test(rule), 'its CSS keeps it in the flow', rule);
  ok(!CSS.split('\n').some(l => /#upgrade-recovery-notice/.test(l) && /fixed/.test(l)), 'no stylesheet fixes it in place either');
}

console.log('\nB. Showing, hiding, coming back');
function load({ profiles = [], browser = null, message = '', store = new Map(), storageThrows = false } = {}) {
  const els = {};
  const el = (id) => els[id] ??= ({ id, textContent: '', title: '', style: {}, value: '', appendChild() {}, flags: {},
    classList: { toggle(name, on) { el(id).flags[name] = !!on; } } });
  const ctx = {
    console, JSON, Date, document: { getElementById: el, createElement: () => ({}) },
    legacyBrowserRecovery: browser, storageRecoveryMessage: message,
    localStorage: storageThrows
      ? { getItem() { throw new Error('denied'); }, setItem() { throw new Error('denied'); } }
      : { getItem: (k) => store.has(k) ? store.get(k) : null, setItem: (k, v) => store.set(k, String(v)) },
  };
  vm.createContext(ctx);
  vm.runInContext(DATA.slice(DATA.indexOf('/* UPGRADE RECOVERY')), ctx);
  vm.runInContext('upgradeServerProfiles = ' + JSON.stringify(profiles), ctx);
  ctx.renderUpgradeRecovery();
  return { ctx, els, store, shown: () => els['upgrade-recovery-notice'].style.display !== 'none' };
}
{
  const p = load();
  ok(!p.shown() && p.els['data-btn'].textContent === '// DATA' && !p.els['data-btn'].flags['has-recovery'], 'nothing to recover: no bar, plain DATA');
}
{
  const store = new Map();
  let p = load({ profiles: ['legacy-1234567890abcdef'], store });
  ok(p.shown(), 'an older server copy: the bar shows');
  ok(/kept separately\. Nothing is lost\./.test(p.els['upgrade-recovery-notice-text'].textContent), 'and says what it is', p.els['upgrade-recovery-notice-text'].textContent);
  const flagged = (btn) => btn.textContent === '// DATA' && btn.flags['has-recovery'] === true && /Upgrade recovery/.test(btn.title);
  ok(flagged(p.els['data-btn']), 'DATA is flagged: an amber dot (has-recovery) and a tooltip, and a label that still fits', JSON.stringify(p.els['data-btn']));
  p.ctx.dismissUpgradeRecoveryNotice();
  ok(!p.shown(), 'x: the bar hides');
  ok(flagged(p.els['data-btn']) && p.els['upgrade-recovery'].style.display !== 'none',
    'DATA is still flagged, and its Upgrade recovery section is still there');
  p = load({ profiles: ['legacy-1234567890abcdef'], store });
  ok(!p.shown(), 'next load, same offer: still hidden (the older copies are kept for good)');
  p = load({ profiles: ['legacy-1234567890abcdef', 'legacy-fedcba0987654321'], store });
  ok(p.shown(), 'another older copy turns up: the bar comes back');
  p = load({ profiles: ['legacy-1234567890abcdef'], browser: { raw: '{"threads":[]}' }, store });
  ok(p.shown(), 'an older browser copy as well: the bar comes back');
}
{
  const store = new Map();
  const warning = 'Could not read the browser database. Automatic saves and sync are paused to protect it.';
  let p = load({ message: warning, store });
  ok(p.shown() && p.els['upgrade-recovery-notice-text'].textContent === warning, 'a storage warning: shown, in its own words');
  p.ctx.dismissUpgradeRecoveryNotice();
  ok(!p.shown() && store.size === 0, 'x hides a warning for this visit only');
  p = load({ message: warning, store });
  ok(p.shown(), 'next load, still wrong: the warning is back');
}
{
  const p = load({ profiles: ['legacy-1234567890abcdef'], storageThrows: true });
  ok(p.shown(), 'browser storage refused: the bar still shows');
  p.ctx.dismissUpgradeRecoveryNotice();
  ok(!p.shown(), 'and x still hides it for this visit');
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
