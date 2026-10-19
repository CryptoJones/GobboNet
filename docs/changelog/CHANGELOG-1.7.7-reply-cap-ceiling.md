# 1.7.7 — The reply cap is the number you typed

Roadmap item #6a, the first of the hard-coded limits moved to the user (#5 lists
them all).

## What was wrong

A character's reply cap — *Cap this character's reply length*, in the card's
*Context & Reply Length* box — was the user's own number, and it was clamped to
8,192 in three places, with nothing saying so:

| Where | What it did |
|---|---|
| `js/04-state.js`, `resolveSmartLimit()` | `Math.min(8192, …)` when the cap was read |
| `js/15-cards.js`, `saveCard()` | `Math.min(8192, …)` when the card was saved |
| `chat.html`, `#card-smart-limit-tokens` | `max="8192"` on the input |

Type 20,000 and the reply was cut at 8,192.

## The change

The ceiling is gone from all three. The floor of 25 stays: below it the
sentence-end cut in `js/03-generation.js` has nothing to work with. Any ceiling
we picked instead would just be another hard-coded limit, which is what #6 is
removing.

**Visual:** none. The same field, without a maximum.

## Pinned by

**`tests/test-user-limits.mjs`, section A**, which drives the real
`resolveSmartLimit()`:

- 20,000 and 8,193 come back as typed, 300 is untouched, and 10 is raised to 25.
- Source guards: saving a card no longer clamps, and the input has `min="25"`
  and no `max`.

Against the original files, all four ceiling assertions fail.

## Not changed

- The global `smartLimitTokens` default (300).
- The context-size migration in `js/05-persistence.js`, which also mentions
  8192. It is about context size, not the reply cap.
