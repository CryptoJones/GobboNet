# 1.7.7 — Saved passwords no longer land in the conversation search

Reported by the maintainer in 1.7.7 testing: on launch, with password autofill
on, a chunk of text appeared in the search box at the top of the conversation
list, highlighted, and ran a search.

## What was happening

Reproduced in a real Chromium 141 with a GobboNet sign-in saved in its password
manager.

**The chat page had no `<form>` at all.** Chrome therefore read the whole page
as one formless login form:

- **Its "password"** was the sign-in box that appears after a server restart
  (`#unlock-password`, rightly marked `autocomplete="current-password"`).
- **Its "username"** was the first text box on the page: the conversation
  search (`#thread-search`).

So on every load, Chrome put the saved username into the search box,
highlighted it as autofilled, and its input event ran the search. The search
box's `autocomplete="off"` did nothing: Chrome ignores it for this.

**Not new in 1.7.7.** The sign-in box has been in the page since an earlier
release. It surfaces when the browser holds a saved GobboNet login that has a
username.

## The change

**1. The sign-in box gets its own `<form>`**, the only one on the page. Chrome
now sees the sign-in for what it is: one password field and its button.

- Submitting is stopped in the form and done by `submitUnlock()`, as before.
- Enter and the Unlock button both submit through the form. The old
  key-press and click handlers are gone, so it can't sign in twice.

**2. The Search API Key field is marked `autocomplete="new-password"`.** This
was needed because step 1 alone moved the problem: the rest of the page became
the "login form", and Chrome filled:

- the username into the search box again;
- the **GobboNet password into the Search API Key field**, ready to overwrite
  the real key the next time CONFIG was saved.

`new-password` tells Chrome the field is not a saved sign-in. `off` does not
work; Chrome ignores it on password fields, which the same repro showed.

## Result, in the same Chromium

| Saved login | Before | After |
|---|---|---|
| With a username | Username in the search box, search run; password in the sign-in box | Nothing filled anywhere |
| Without a username | Password in the sign-in box | Password in the sign-in box, where it belongs |

## Side effect to know about

For a Chrome user signed in to Google with password sync, clicking into the
Search API Key field may offer to "suggest a strong password", because that is
what Chrome does for a `new-password` field. Ignore it, or type the key as
usual.

## Visual

None:

- **The sign-in box** was screenshotted before and after, at 1440 and 390
  wide. The phone shot is pixel-identical. The desktop shot differs in 26
  pixels by at most 2/255 at the edge of the focus glow, which is animation
  timing.
- **The API key field** only gained an attribute.

## Pinned by

**`tests/test-password-autofill.mjs`** (new, 12 assertions). The markup rules
the fix rests on:

- The sign-in box is the page's only form.
- Submitting it never leaves the page.
- It has one submit path.
- It holds no text box a password manager could take for a username.
- Every other password field is `new-password`.
- The sign-in code still finds its elements.

Against the old page, 7 of them fail.

**The browser half needs a real Chromium with a saved login and a display**,
so it was run by hand, not in the suite. The method:

1. Write a saved login into a fresh Chromium profile's `Login Data`, encrypted
   the way Chromium's Linux "basic" password store does.
2. Sign in to GobboNet in that browser.
3. Read which fields Chrome marked `:autofill`.

**The sign-in box still works:**

- A wrong password with Enter shows "That password was not accepted."
- The right password with Enter, or with the button, signs in and closes the
  box.
- The page never reloads or navigates.
- No page errors.
