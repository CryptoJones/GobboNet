# 1.7.8 — How long a model may load, and a reply may take to start: settings, not constants

> looks like i found TWO hard coded switches that need to be moved to the user
> config.
>
> 3 minutes to load a model: swapTimeout, supervisor.go:42, and
> pollSwapStatus(180000) in three places in js/02-model.js
>
> 5 minutes waiting for the engine to start the reply (jobs.go:125, proxy.go:35)

## What was happening

- **A model load had three minutes.** That limit was `swapTimeout` in
  `internal/supervisor`. A big model off a slow disk, or onto the CPU, could
  not finish, and nothing could change it.
  - The page had its own copy of the number: `pollSwapStatus(180000)` for the
    header swap, the lore-model borrow, and the CONFIG > Performance restart.
  - The two clocks ran out together, so the page often gave up first. It
    showed "Timed out waiting for model to load" instead of the server's
    reason.
  - When the server did report, its message said "within 3m0s" whatever the
    deadline really was.
- **A reply had five minutes to start.** That was `responseHeaderTimeout` in
  `internal/jobs` and `internal/proxy`. On the CPU, a long chat's prompt can
  take longer to read than that. The failure then read as `net/http: timeout
  awaiting response headers`.

## The change

Two settings in `config.toml`. Both are documented in the shipped file, can be
set with `gobbonet config set`, and can be edited in CONFIG > Config File.

| Setting | Default | Range | Bounds |
|---|---|---|---|
| `model_load_timeout_seconds` | 180 | 10–7200 | a model load: start-up, a swap, the reload after idle stand-down, a CONFIG > Performance restart |
| `reply_start_timeout_seconds` | 300 | 10–86400 | the wait for the reply's first byte while llama.cpp reads the prompt: every reply (detached jobs), and the `/llm` route lore compression uses |

- **Defaults:** when a setting is absent, or 0 or less, the old constant is
  used, so nothing changes for anyone who leaves them alone.
- **Out of range:** a value outside the range stops the server at start-up,
  with the key and the range named. CONFIG > Config File refuses to save it
  in the first place.
- **The page takes the server's number.** `/swap-status` now reports
  `load_timeout_seconds`. The page waits that long plus 30 seconds, the most a
  swap spends stopping the old engine first.
  - The server's verdict and its reason therefore always arrive before the
    page gives up.
  - The page's own clock is only a backstop for a server that stops answering,
    and its message now says how long it waited.
  - A server that does not report a limit (the legacy PowerShell server, or one
    older than 1.7.8) gets three minutes plus the margin.
  - There is no `180000` left in the page.
- **Both failures say what to do.** For example:
  - "model did not finish loading within 15s. A big model on a slow disk or
    the CPU can need longer: raise model_load_timeout_seconds in config.toml
    (CONFIG > Config File) and restart GobboNet"
  - "the model did not start replying within 10s. … raise
    reply_start_timeout_seconds …"

  The `/llm` route puts the same advice in its error detail.
- **Unchanged:**
  - `/search` and `/embed` keep their five minutes, since a slow prompt is not
    their problem.
  - The between-chunks watchdog (10 minutes of silence mid-reply) is
    unchanged.
  - The total-runtime limit is still CONFIG's *Infinite Runtime Bug Fix*.

## Checked here

These were checked end to end in real Chromium against a built server.

- **Reply start:** a fake engine held every reply back 15 seconds.
  - At `reply_start_timeout_seconds = 10`, the reply failed after 11 seconds,
    and the chat showed "did not start replying within 10s" with the setting's
    name.
  - Raised to 30 in CONFIG > Config File and restarted, the same reply arrived
    after 16 seconds.
- **Model load:** the ZIP's real llama-server ran a tiny model through a
  wrapper that waits 25 seconds before starting it, standing in for a slow
  disk.
  - At `model_load_timeout_seconds = 15`, start-up gave up. The window named
    the setting, and `/swap-status` reported 15.
  - A swap from the page then failed after 17 seconds with the server's
    reason, not the page's own timeout.
  - Raised to 40 in CONFIG > Config File and restarted, start-up loaded the
    model. A swap from the page waited 26 seconds and succeeded.

**Not checked:** Windows, and a real large model on a slow disk. The slow load
was simulated, but the engine and the timeout path were the real ones.

## Tests

- **`internal/config/timeouts_test.go`:**
  - defaults and set values;
  - 0 or less reads as the default;
  - out of range or the wrong type is refused, naming the key and range;
  - `Keys()` and `config set`;
  - the shipped file documents both.
- **`internal/supervisor/load_timeout_test.go`:** the configured timeout
  applies, `/swap-status` reports it, and the message names it and the
  setting, never "3m0s".
- **`internal/jobs/reply_start_test.go`:** the default is 5 minutes; a silent
  engine is ended at the configured bound with the actionable message; a slow
  start inside the bound is a reply.
- **`internal/proxy/reply_start_test.go`:** the same for `/llm`.
- **`internal/server/configfile_test.go`:** the setting reaches the jobs and
  `/llm`, and not `/search` or `/embed`.
- **`internal/config/editfile_test.go`:** CONFIG > Config File refuses an
  out-of-range load timeout.
- **`tests/test-swap-timeout.mjs`:** the real `pollSwapStatus` on a fake
  clock:
  - 600 → 630 s; 60 → 90 s;
  - no report → 210 s;
  - the server's reason arrives before the page gives up;
  - ready ends the wait;
  - a server that stops answering keeps its budget;
  - an explicit caller budget wins;
  - no caller passes its own number.
