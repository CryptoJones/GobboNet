# 1.7.7 — The lore compression timeout is a setting

Roadmap item #6c.

## What was there

When a chat outgrows the model's context, the oldest messages are summarised
into the lore log before the user's message is sent. Each pass was cancelled
after `LORE_TIMEOUT_MS = 45000` (`js/07-prompt.js`). On a timeout:

- the message goes through,
- the pass is cancelled, and
- the messages it was summarising leave the context **without** a summary.

Forty-five seconds is fine on a GPU. A CPU-only install, running exactly the
"slow local model" the constant's own comment mentions, could hit it every time.

## The change

**`loreTimeoutSeconds`**, default 45, read once per pass through
`loreTimeoutMs()`:

- **Floor 5 s.** Below that no model finishes, and every pass would fail. A
  smaller value is raised to it.
- **Nonsense falls back to 45.**
- **The timeout message reports the limit that applied,** read from the same
  value as the timer ("timed out after 90s…"), not from the constant.

It is in `DEFAULT_SETTINGS`, so `lore_timeout_seconds = 120` under `[ui]` in
`gobbonet.toml` presets it.

### CONFIG

A new **📝 Lore Compression** box after *Infinite Runtime Bug Fix*:

- **Box:** a copy of Reply Delivery's.
- **Number row:** a copy of *Infinite Runtime*'s — "Give up after: [ 45 ] s".
- **Hint:** says plainly what a timeout costs: those oldest messages leave the
  context without a summary.
- **Icon:** 📝 is the one the lore notices in the chat already use.

## Pinned by

**`tests/test-user-limits.mjs`, sections C and E** (real `js/07-prompt.js`):

- **The reader:** default 45 s, 180 s honoured, 2 s raised to 5, nonsense
  falls back to 45.
- **Source guards:** `summarizeForLore()` arms its abort timer from
  `loreTimeoutMs()`, the timeout message uses the same value, and the pass no
  longer reads the constant.
- **CONFIG wiring:** a value under 5 saves as 5.

## Verified in the browser

- **Round trip.** Saving 90 reopens as 90, and the reader returns 90000 ms.
  Saving 2 reopens as 5. A blank restores 45.
- **Layout.** At both widths the box matches its neighbours.
- **No page errors.**

## Not changed

- The pass's other limits — its output budget and its input size (24 messages /
  12,000 characters). They are listed in the #5 inventory as "later, if asked".
