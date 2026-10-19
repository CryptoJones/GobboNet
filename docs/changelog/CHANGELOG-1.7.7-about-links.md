# 1.7.7 — ABOUT links to mods and the Discord

Roadmap item #3: *"Integrate link to mods, discord, and placeholder for auto
update feature in the [ABOUT]."*

## What was there

Nothing. ABOUT had no links of any kind, and neither address appeared anywhere
in the repository.

## The change

One new box at the end of ABOUT, after YOUR DATA:

> **🔗 LINKS** — Mods, extras and the community live outside the app. Each opens
> in a new tab, and nothing is contacted until you click it.
>
> `MODS` `DISCORD` `CHECK FOR UPDATES`
>
> Update checks are coming in a later version. New releases are announced in the
> Discord.

| Button | Opens |
|---|---|
| MODS | `https://goblincorps.com/gobbonet_extras` |
| DISCORD | `https://discord.gg/rcmN8uEbaa` |
| CHECK FOR UPDATES | nothing — a disabled placeholder |

### Nothing new to look at

Every part of the box is copied from what ABOUT already has:

- **Box and heading:** YOUR DATA's inline styles, character for character.
- **Buttons:** `btn btn-sm`, the same as COPY BUILD INFO. Their computed styles
  were compared in the browser and are identical.
- **Spacing:** the 12 px gap between buttons is `.modal-actions`', and the 12 px
  above them is `#about-copy-btn`'s.
- **The placeholder:** disabled exactly the way Add a Model's NOT WORKING YET
  is, with the reason written underneath. `.btn` has no disabled style of its
  own, so like NOT WORKING YET it looks like an ordinary button. A dimmed look
  would be a new visual and was not invented here.

### Buttons, not links

They are `<button>`s calling `openAboutLink()` (`js/22-scheduler.js`), not
`<a class="btn">`, because `.btn` does not reset `text-decoration` and a link
would render underlined — the one button in the app that looked different.

`openAboutLink()` opens a new tab with `noopener,noreferrer`:

- **noopener:** the opened page gets no `window.opener`, so it cannot navigate
  or script GobboNet.
- **noreferrer:** it is not told where it was opened from. On a LAN, that
  address is the PC's private IP.

### Privacy

PRIVATE BY DESIGN promises that "the only network calls left are ones you ask
for". Links clicked by the user keep that promise. The placeholder does
nothing, and the addresses appear only in the buttons' click handlers, never in
a script that could run on its own.

When the real update check is built, it has to be user-triggered or opt-in, or
that sentence has to change.

## Pinned by

**`tests/test-about-links.mjs`** (new, 25 assertions):

- **A.** Each button opens exactly the address given; they are buttons, not
  links.
- **B.** `openAboutLink()` opens a new tab with `noopener` and `noreferrer`, and
  does not treat `window.open`'s `null` (which `noopener` causes) as an error.
- **C.** Neither address appears in any script, and the placeholder is disabled,
  has no handler, and says why.
- **D.** The box and heading styles equal YOUR DATA's, and the box sits after
  YOUR DATA and before CLOSE.

## Verified in the browser

Rebuilt binary, Chromium:

- **No requests on their own.** Loading the app and opening ABOUT made no
  request outside the machine.
- **MODS and DISCORD:** each opened a new tab requesting exactly its address,
  with no `Referer`, and with `window.opener` null in the new tab.
- **CHECK FOR UPDATES:** reports disabled; even a forced click made no request
  and opened no tab.
- **Layout:** at 1440 and at 390 wide, all three buttons fit on one row; no page
  errors.

## Not changed

- No link to the GitHub releases page; it was offered and not asked for.
- `tests/test-version-stamp.mjs`, which also reads ABOUT, still passes all 59
  assertions.
