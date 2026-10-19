# 1.7.7 — No 30-minute stop; the runtime limit is yours

Roadmap item #4: *"30min hard stop on responses? Investigate + Cull if so, users
determine when a response stops."* The maintainer's decision: make it a
variable setting in the user config, in CONFIG.

## What was there

`internal/jobs/jobs.go` stopped **every** generation on the normal chat path
(`/llm/jobs`) at 30 minutes:

```go
jobTimeout = 30 * time.Minute
ctx, cancel := context.WithTimeoutCause(context.Background(), m.maxRuntime, errJobExpired)
```

It fired even while tokens were still arriving. It was the only total cap in the
app:

- the browser's runtime limit is CONFIG → *Infinite Runtime Bug Fix*, off by
  default;
- chat requests send `max_tokens: -1`;
- the direct proxy has no total cap.

That CONFIG limit had its own gap. It is a timer in the page
(`startCotWatchdog`, `js/10-chat.js`), so it only ran while the page was open.
A reply that carried on with the tab closed or the phone asleep ran into the
server's fixed 30 minutes instead.

## The change

**One setting, the one CONFIG already had, now enforced wherever the reply is
running.**

- **The page** sends its limit with every job, beside the existing `?thread=`:
  `POST /llm/jobs?thread=…&max_runtime_s=300`.
  - Auto-cull **off** sends nothing.
  - The request body stays byte-exact.
  - `cotTimeoutLimitMinutes()` is the one reader for both the page's timer and
    the request, so the two cannot disagree.
- **The server** enforces that limit on the job. **No limit, no deadline at
  all.** `jobTimeout` is gone, and a new manager's `maxRuntime` is 0.
  - A malformed value (`abc`, `-5`, `1.5`, an overflow) is refused with 400
    *before* `supersede()`. A bad request still cannot kill the reply already
    running.
- **The server allows 15 seconds past the limit (`pageCapGrace`).** With the
  page open, its own timer and the server's copy would otherwise race, and a
  server win would show an error where the user expects the page's note
  ("auto-stopped — runtime exceeded N min limit…"). The grace makes the page's
  stop land first; the server's copy is for the tab that was closed.
- **The server's message names the user's limit and where it lives:**
  *"this generation ran past its 1m0s runtime limit and was stopped. The limit
  is set in CONFIG."* It reports the limit as set, without the grace.

**Kept:** the two watchdogs that never cut off a reply that is still streaming —
5 minutes for a first byte and 10 minutes of silence between chunks. They catch
a dead server, not a long answer. Memory stays bounded by the 128 MiB per-job
buffer cap.

**An older server** (1.7.6 and before) ignores the parameter and keeps its fixed
30 minutes. Nothing breaks either way round.

### CONFIG

No new control. One sentence is added to the *Infinite Runtime Bug Fix* hint, so
it says what the setting now does:

> This is the only time limit on a reply: there is no fixed cut-off behind it,
> and it still applies if you close this page or your phone goes to sleep.

## Pinned by

**`internal/jobs/runtime_limit_test.go`** (new, six tests):

- **`TestNewManagerHasNoRuntimeCap`** — the default is no cap.
- **`TestZeroLimitMeansNone`** — `max_runtime_s=0` sets no deadline at all,
  even over a manager-level cap.
- **`TestJobWithoutALimitRunsToTheEnd`** — a job with no limit runs to the end.
- **`TestPageLimitStopsAJob`** — a 1 s limit stops a live stream, as `error`,
  with a message naming "1s runtime limit" and CONFIG.
- **`TestPageLimitGetsItsGrace`** — still running at limit + 0.5 s, stopped
  after the grace, and the message names the limit, not the padded deadline.
- **`TestMalformedLimitIsRefusedBeforeSupersede`** — four bad values get 400,
  and the running job survives.

`TestExpiredJobReportsAnError` passes unchanged.

**`tests/test-user-limits.mjs`, section D** drives the real
`startGenerationJob()`:

- Off sends no parameter.
- 5 min sends `max_runtime_s=300`; 90 min is sent as is.
- A cleared minutes field means the CONFIG default of 2, as before.
- The page's timer reads the same function.
- The CONFIG hint says what the setting now does.

## Verified end to end

A real browser, the rebuilt server, and a stand-in model server that streams a
token every half second, forever.

| Scenario | What happened |
|---|---|
| Page open, auto-cull 1 min | Request carried `max_runtime_s=60`. The page stopped the reply at **60.5 s** with its usual note; the server never had to. |
| Tab closed 2 s after sending, auto-cull 1 min | Still running at 65 s and 72 s (the grace). Stopped at **75.6 s** with *"…ran past its 1m0s runtime limit…set in CONFIG."* |
| Auto-cull off | No limit sent. The job ran until Stop (6.0 s). |

Before this change, the closed-tab case was the 30-minute stop.

## Docs

`docs/GO_SERVER.md`: the *Total runtime* row now reads "none, unless the user
sets one", and a paragraph explains the parameter, the grace, and the 400.

## Not changed

- The *Infinite Runtime Bug Fix* box's layout and controls.
- Its `max="60"` on the minutes field. That only limits the spinner — a typed 90
  is read and sent as 90 — so it was left alone.
- The legacy `fileserver.ps1` path. Its 30-minute values are per-wait, not a
  total.
