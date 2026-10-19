# 1.7.7 — Each character sets its own context budget

Roadmap item #6d: the "token reservation for lorebook compression" from the
update notes, item 5's example of a hard-coded limit.

## What was there

Four constants in `buildContextMessages` (`js/08-rag.js`) decided how much of
the context a character's prompt may use and when older messages are compressed
into lore:

| | Value | What it does |
|---|---|---|
| Input budget | 0.90 of the context | the most the prompt may fill |
| Reply room | 0.20 of that | when less is left for the reply, compression starts |
| Land at | 0.65 of that | where compression brings the chat back down to |
| Keep verbatim | 0.22 of that | the most recent turns, never summarised |

A second copy of the first two lived in the context meter under the chat
(`updateContextInfo`, `js/13-dashboard.js`), with a comment asking that the two
be kept in sync by hand.

## The change

**Per character, with defaults.** Four card fields:

- `contextInputPercent`
- `contextReplyPercent`
- `contextLandPercent`
- `contextKeepPercent`

Each resolves card → setting → the old constant. **Blank (null) means
inherit.** The settings carry the old values, so nothing moves for anyone who
leaves them alone, and they can be preset for every device from `gobbonet.toml`:

```toml
[ui]
context_input_percent = 85
context_land_percent = 55
```

**One resolver, `resolveContextBudget(card)`** (`js/04-state.js`). It is read by
the budgeter *and* the meter, so the meter can no longer drift from the trigger
it displays.

**Combinations that would stall compression are adjusted, and said.** The four
are not independent:

- A landing point at or above where compression starts frees almost nothing, so
  compression fires again on every message.
- A verbatim reserve at or above the landing point means compression can never
  get down to it.

So each value is first held to its own range:

| Field | Range |
|---|---|
| Input | 10–99% (100% would leave no room to reply) |
| Reply room | 0–60% |
| Land at | 10–95% |
| Keep verbatim | 0–90% |

Then the values are held in order, with a 5-point gap: keep < land < where
compression starts. **Every adjustment comes back as a sentence**, shown in the
card editor. The user's number is stored as typed (within its field range) and
adjusted at use, never rewritten behind their back.

### The card editor

Four fields in the *Context & Reply Length* box, in the same `form-row` layout
as the context limit and reply cap at its top:

- **Input budget** (% of context)
- **Reply room** (% of input budget)
- **Land at** (% of input budget)
- **Keep verbatim** (% of input budget)

Each has a one-line explanation. A blank field shows its default as the
placeholder ("90 (default)").

**A note under them updates as you type**, in terms of the whole context: "Compression
starts at 72% of the context and brings the chat back to 59%; the last 20% is
kept word for word." Any adjustment is added to it.

**The context-limit hint follows the card.** It used to say a fixed "90% is the
input budget"; it now names the card's own figure.

## Pinned by

**`tests/test-context-budget.mjs`** (new, 35 assertions, on the real resolver):

- **Defaults:** the old constants, unchanged and with no notes.
- **Precedence:** card beats setting beats constant; blank, null and `''`
  inherit; 0 is honoured for reply room and keep.
- **Ranges:** out-of-range values are clamped and reported.
- **Combinations:** both stalling combinations are adjusted with their reasons,
  and they chain. Across a grid of field values, keep < land < trigger always
  holds.
- **Wiring:** the budgeter and the meter both read the resolver, with no
  constant left behind; the editor has the four fields, the note and the live
  hint.
- **Saving:** blank stores null, and typed values are kept within their field
  range rather than replaced by the adjusted value.

Against the original files, it fails at once: the resolver does not exist there.

## Verified in the browser

A rebuilt server, through the real card editor:

- **Placeholders** read "90 (default)", "20 (default)", "65 (default)" and
  "22 (default)".
- **The note** with defaults: "Compression starts at 72% of the context and
  brings the chat back to 59%; the last 20% is kept word for word."
- **Typing land 85 and input 50:** the note adjusts — "Land at 85% would be at
  or above where compression starts (80%, with 20% reply room), so it would fire
  on every message. 75% is used." — and the context-limit hint changes to "50%
  is the input budget".
- **Saved and reopened** as `[50, blank, 85, blank]`: blanks stay inherited.
- **The meter** on a 4,096-token context:

  | Card | Trigger |
  |---|---|
  | Defaults | 2,949 tokens (unchanged from before) |
  | Input 50% | 1,639 |
  | Input 50%, reply room 0% | 2,048 |

- **Layout:** at 1440 and 390 wide, the fields match the box's existing pair.
- **No page errors.**
