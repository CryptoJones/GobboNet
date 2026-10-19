# 1.7.7 — The Linux onboarding test reports its real failure

Roadmap item #12. Found while getting a baseline for the 1.7.7 cycle.

## What was wrong

`tests/test-linux-onboarding.py` starts the launcher in its own process group
and, in a `finally`, kills that group. When the launcher has already exited —
which is the usual reason the test is failing at all — the group is empty, and
`os.killpg` raises `ProcessLookupError`. That exception replaced the real one
as the error the run ended on:

```
AssertionError: launcher stopped

During handling of the above exception, another exception occurred:
  ...
ProcessLookupError: [Errno 3] No such process
```

So a failing run's headline was "no such process", which describes the
cleanup, not the problem.

## The change

The cleanup tolerates a group that has already gone. Nothing the test checks
has changed. On a checkout without a built install, the run now ends on the
reason it actually stopped:

```
AssertionError: launcher stopped
```

## Verified

- On a bare checkout: the error is now `AssertionError: launcher stopped`, with
  no `ProcessLookupError`.
- Against a real prefix: a freshly built `gobbonet` beside a copy of
  `linux-amd64/`, with a stub `llama-cpp/llama-server` that answers `--version`
  (the test switches to a remote backend before an engine would ever load a
  model). Run with `GOBBONET_TEST_PREFIX`, **both passes are green**, local and
  LAN. This is the eighth Python suite, which could not run in the authoring
  sandbox before.

## Not changed

What the test asserts, and its compact style.
