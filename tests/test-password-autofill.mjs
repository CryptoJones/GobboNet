/**
 * Chrome's password manager must not treat the chat page as a login form.
 *
 * Reported in 1.7.7 testing: on launch, with saved passwords, a chunk of text
 * appeared in the conversation search box, highlighted, and ran a search.
 * Reproduced in a real Chromium 141 with a saved GobboNet login:
 *
 *   - The page had no <form> at all, so Chrome read the whole page as one
 *     formless login form. Its password was the sign-in box's field
 *     (#unlock-password, autocomplete="current-password") and its "username"
 *     the first text box on the page -- #thread-search. The saved username
 *     went there on every load, and its input event ran filterThreads().
 *   - autocomplete="off" on the search box did nothing: Chrome ignores it here.
 *   - Giving the sign-in box its own <form> alone moved the problem: the rest
 *     of the page became the login form, and the GobboNet password went into
 *     the Search API Key field (#set-apikey), ready to overwrite the real key
 *     on the next save.
 *   - autocomplete="new-password" on #set-apikey finishes it: Chrome does not
 *     fill a saved sign-in into it. "off" does not -- also ignored on password
 *     fields.
 *
 * After both: a saved login with a username fills nothing anywhere; one
 * without fills only the sign-in box, where it belongs.
 *
 * The browser half cannot run in this suite (it needs a real Chromium with a
 * saved login and a display). These are the markup rules it rests on.
 */
import { fileURLToPath } from 'node:url';
import fs from 'fs';

const ROOT = fileURLToPath(new URL('..', import.meta.url)).replace(/\/$/, '');
const HTML = fs.readFileSync(ROOT + '/chat.html', 'utf8');
const SYNC = fs.readFileSync(ROOT + '/js/06-state-sync.js', 'utf8');

let pass = 0, fail = 0;
const ok = (c, l, extra) => {
  if (c) { pass++; console.log('  ✓ ' + l); }
  else { fail++; console.log('  ✗ ' + l + (extra ? '\n      ' + extra : '')); }
};

// Comments mention <form> in prose; only real tags count.
const markup = HTML.replace(/<!--[\s\S]*?-->/g, '');

console.log('\n=== the sign-in box is the page’s only form ===');
const forms = [...markup.matchAll(/<form\b[^>]*>[\s\S]*?<\/form>/g)].map(m => m[0]);
ok(forms.length === 1, 'there is exactly one <form>', `found ${forms.length}`);
const unlock = forms[0] || '';
ok(/^<form id="unlock-form" onsubmit="event\.preventDefault\(\);submitUnlock\(\);">/.test(unlock),
   'it is the sign-in box, and submitting it never leaves the page');
ok(/<input type="password" id="unlock-password" name="password" autocomplete="current-password"/.test(unlock),
   'it holds the sign-in password, marked as one');
ok(/<button class="btn btn-fill" id="unlock-submit" type="submit">Unlock<\/button>/.test(unlock),
   'and its Unlock button submits it');
ok(!/onclick="submitUnlock\(\)"/.test(markup) && !/onkeydown="if\(event\.key==='Enter'\)submitUnlock\(\);"/.test(markup),
   'with no second handler left to sign in twice (Enter and the click both go through the form)');
ok(!/<input[^>]*type="(text|search|email)"/.test(unlock), 'and no text box a password manager could take for a username');

console.log('\n=== every other password field is not a sign-in ===');
const outside = markup.replace(unlock, '');
const others = [...outside.matchAll(/<input\b[^>]*type="password"[^>]*>/g)].map(m => m[0]);
ok(others.length >= 1, 'the Search API Key field is still a password field', String(others.length));
for (const f of others) {
  const id = (f.match(/id="([^"]+)"/) || [])[1];
  ok(/autocomplete="new-password"/.test(f), `#${id} is marked new-password, so a saved sign-in is never filled into it`, f);
}

console.log('\n=== the sign-in code still finds what it needs ===');
for (const id of ['unlock-modal', 'unlock-password', 'unlock-error', 'unlock-submit']) {
  ok(markup.includes(`id="${id}"`) && SYNC.includes(`'${id}'`), `#${id} is in the page and read by js/06-state-sync.js`);
}

console.log(`\n${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
