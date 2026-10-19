# 1.7.7 — Auto-continue's pause and chain cap are settings

Roadmap item #6b.

## What was there

`{{auto_continue_N}}` in a message makes the model keep going on its own for N
posts. Two numbers governing it were constants in `js/09-threads.js`:

- `AUTO_CONTINUE_DELAY_MS = 20000` — the pause between posts, shown as a
  countdown above the input.
- `AUTO_CONTINUE_MAX_CHAIN = 50` — the most posts one macro may run, so that
  `{{auto_continue_9999}}` cannot run away.

## The change

**Two settings**, with the old values as defaults, so nothing changes for anyone
who leaves them alone:

| Setting | Default | Range |
|---|---|---|
| `autoContinueDelaySeconds` | 20 | 0 or more |
| `autoContinueMaxChain` | 50 | 1 or more |

- **0 s is a real choice** (no pause between posts) and is honoured.
- **The chain cap's floor is 1**, because the post being sent is always the
  first.
- **No upper limits.**

**Read at the moment of use,** through `autoContinueDelayMs()` and
`autoContinueMaxChain()`, so a CONFIG change applies to the next step of a chain
without a reload. A missing or nonsense value falls back to the default rather
than to 0: a cleared field must not quietly mean "fire instantly" or "never
continue".

`maybeContinueChain()` reads the pause **once** per step, so the countdown and
the timer cannot disagree if CONFIG changes mid-wait.

Both settings are in `DEFAULT_SETTINGS`, so they can also be preset from
`gobbonet.toml`:

```toml
[ui]
auto_continue_delay_seconds = 10
auto_continue_max_chain = 100
```

### CONFIG

A new **↻ Auto-Continue** box directly after *Reply Delivery*:

- **Box:** a copy of Reply Delivery's.
- **Number rows:** copies of *Infinite Runtime Bug Fix*'s "Max runtime: [ n ]
  min" — "Wait between posts: [ 20 ] s" and "Most posts per chain: [ 50 ]".
- **Hint:** explains the macro, and that Stop, sending something yourself, or
  switching thread ends a chain early.
- **Icon:** ↻ is the app's existing "repeats" mark (scheduled tasks use it for
  *Daily*).

## Pinned by

**`tests/test-user-limits.mjs`, sections B and E** (real `js/09-threads.js`):

- **The readers:** defaults; 0 s honoured; cleared or negative values fall back;
  200 allowed; a chain cap of 0 falls back to 50.
- **The macro parser:** `{{auto_continue_120}}` runs 50 by default and 120 when
  the cap is 200.
- **The chain stepper:** with the pause at 7 s it schedules the next post 7 s
  out, the countdown starts at 7, and firing sends the next post.
- **CONFIG wiring:** the fields exist, are read and written, and a cleared pause
  saves the default rather than `NaN`.

## Verified in the browser

- **Round trip.** CONFIG opens showing 20 / 50. Saving 7 / 120 reopens as
  7 / 120, and the readers return 7000 ms / 120. Saving 0 / 1 keeps 0 / 1.
  Saving blanks restores 20 / 50.
- **Layout.** At 1440 and 390 wide the box matches its neighbours, and the two
  rows wrap onto separate lines on a phone.
- **No page errors.**
