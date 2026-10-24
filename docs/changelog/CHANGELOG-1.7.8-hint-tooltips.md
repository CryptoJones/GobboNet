# 1.7.8 — Feature notes: one sentence, the rest behind a "?"

> Notes beneath features are getting clunky, we need a hover with that data
> working as a hover over or click on a small ? tooltip. The goal: A BLUF style
> message that's one sentence to supplement current text positions on feature
> details, and moving the expanded text to the tooltip box

## What was happening

Over several releases, the notes under features in CONFIG and the character
editor grew into paragraphs. Some were up to 500 characters long, and the
longest ran to six sentences. CONFIG alone carried about 5,000 characters of
notes. That made the settings themselves hard to find.

## The change

**46 notes** are now one sentence that stays where the note was, with a small
**?** button beside it. The original note, word for word, is in the tooltip.

- **Where:** CONFIG, the character editor, the persona editor, the scheduler,
  extensions and macros, DATA, and the lore summary viewer.
- **Which notes changed:** every static note that was over 120 characters and
  longer than one sentence.
- **Which stayed as they were:**
  - short notes;
  - the slider scale labels;
  - status lines that the page fills in;
  - the sign-in prompt, which is a message rather than a feature note.
- **Warnings stay in the visible sentence,** not only in the tooltip:
  - Performance: changing these restarts the model;
  - custom code: imported cards arrive with it switched off;
  - importing: a full backup replaces everything;
  - Phone & Other Devices: type the address with `http://`;
  - Remote Images: loading one tells that server your IP address;
  - Privacy Status: what goes online, and that it is optional.

  "Cannot be undone" under PURGE was already one line and is unchanged.
- **Opening the tooltip:**
  - **Mouse:** hovering over the **?** opens it, and moving away closes it.
    The pointer can move into the tooltip to read it.
  - **Click or tap:** this pins it open. A second click, a click elsewhere, or
    **Escape** closes it. Escape closes only the tooltip, not the window
    under it.
  - **Phone:** one tap opens it. The **?** is drawn at 16 px but answers taps
    about 12 px around it, a target of roughly 32 px.
  - **Keyboard:** Tab reaches every **?**, and focusing one opens its note.
  - **Screen readers:** each button is labelled "More about …" and described
    by its note (`aria-describedby`), so the whole note is read with it.
- **Placement:**
  - It always fits the window, opening above the **?** when there is no room
    below.
  - It closes when the **?** scrolls out of view.
  - It is one popup on the page body, so a scrolling window can never clip
    it.

### For contributors

The pattern, from `chat.html`:

```html
<div class="form-hint">One sentence.<button type="button" class="tip-btn"
  aria-label="More about X" aria-describedby="tip-x" aria-expanded="false">?</button></div>
<div class="tip-body" id="tip-x" hidden>The full note.</div>
```

- **Behaviour:** `js/18-utils.js` (HINT TOOLTIPS).
- **Styles:** `css/13-components.css`.
- **Content:** the note is cloned into the popup, never parsed from a string.

The file is also now called by its real name in two places:

- **DATA > Server presets** said `gobbonet.toml` for the file that is
  `config.toml`.
- **Code comments** had the same mistake, and it was corrected there too.

## Checked here

These were checked in real Chromium against a built server.

- **Desktop, with mouse and keyboard,** in CONFIG and the character editor:
  - hover, pin, unpin, and click-away;
  - Escape, with CONFIG left open and focus kept on the **?**;
  - Tab to a **?**;
  - flipping above near the bottom of the window;
  - closing when scrolled away;
  - no unsaved-changes flag in the editor;
  - no page errors.
- **A 390 × 844 phone, by touch:** one tap opens, a tap elsewhere or a second
  tap closes, the tooltip stays on screen, and the touch target is the right
  size.
- **Nothing was lost:** a script compared the page before and after. Every
  moved note is in its tooltip unchanged. Undoing the conversion leaves only
  whitespace differences.

**Not checked:** Safari and Firefox, and a real screen reader.

## Tests

`tests/test-hint-tooltips.mjs` has 24 checks:

- every **?** points at its own hidden note, right after it;
- ids are unique, every button is labelled, and none sits inside a `<label>`;
- every visible line is exactly one sentence of at most 130 characters, and
  says less than its note;
- no other long note is left in the open;
- the warnings above are still visible;
- the popup's clone-only rendering, Escape handling, touch target and focus
  ring.

On the previous commit the suite fails.
