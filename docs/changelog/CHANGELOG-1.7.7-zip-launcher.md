# 1.7.7 — The Linux ZIP's `./gobbonet` runs setup, then the chat

Roadmap item #14. Found while packaging batch 1; the maintainer chose the
launcher, because "the entire idea is that this should be easier for folks to
use".

## What was wrong

Two things, about one file.

**The ZIP's `./gobbonet` was the bare server.** `LINUX-START-HERE.md` tells ZIP
users to "extract it, open a terminal in GobboNet and run `./gobbonet`". It
describes the launcher's first run:

- it needs Python and xdg-utils, which only the setup page and the
  browser-opening use;
- it sends people to `launch.log`, which only the launcher writes;
- it says to "substitute `./gobbonet`" for the packages' `gobbonet` command —
  and in the `.deb` that command is a link to the launcher
  (`installer-linux/build-deb.sh`).

But the 1.7.6 ZIP's `./gobbonet` was a byte-identical copy of the server (see the
1.7.6 `SHA256SUMS.txt`). Run as a first-time user, it printed:

```
First run: no config existed, so a commented default was written to:
    ~/.config/gobbonet/config.toml
Review it -- in particular llm_url and server_exe -- then run gobbonet again.
```

and stopped. There was no setup page, no browser and no `launch.log`. A `.deb`
user typing the same word got the friendly first run; a ZIP user got a config
file to edit.

**`make-zip.sh` never built that file.** Its manifest required a top-level
`gobbonet`, and its copy list took whatever file of that name was in the tree.
Nothing produced it: on the maintainer's machine it was a leftover from an
earlier `go build`, of any age. The stamp check at the end looked only at
`linux-amd64/gobbonet`, so a stale top-level server would have shipped without a
word — the same shape as 1.7.4 shipping a 1.7.3 server.

## The change

**`installer-linux/gobbonet-zip-entry`**, a short script that `make-zip.sh`
installs at the top of the archive as `gobbonet`. It runs
`bash linux-amd64/gobbonet-launch "$@"`, the same launcher the packages install.

- **`./gobbonet` on its own** is the launcher's first run: setup page, then
  chat, with `launch.log`.
- **`./gobbonet doctor`, `version`, `config set …`** still reach the server. The
  launcher passes every command that isn't its own straight through, as it
  already does for the packages.
- **Through `bash`, not as a path**, so a lost executable bit on the launcher
  cannot stop it (#11).
- **Found from its real location**, via `readlink -f`, so it works through a
  symlink in `~/bin`.
- **If `linux-amd64/` is missing**, it says to keep the extracted folder
  together, rather than failing obscurely.

It is not stored as a top-level `gobbonet` in the repository, because `go build
./cmd/gobbonet` writes its binary to exactly that path and would replace it
without a word.

**`make-zip.sh`:**

- no longer copies a top-level `gobbonet` from the tree;
- installs the entry script in its place, `0755`;
- after assembly, refuses to build unless the archive's `gobbonet` is that
  script, byte for byte, and executable;
- prints `linux: ./gobbonet -> linux-amd64/gobbonet-launch (setup, then chat)`.

Windows, the `.deb` and the `.rpm` are unaffected.

### Also: `CONTRIBUTORS.md` was never in the archive

Found while checking the 1.7.7 archive from the outside. `make-zip.sh`'s copy
list carried README, LICENSE, AGENTS and the rest, but not `CONTRIBUTORS.md`.
Release commits are made from the extracted archive, so a credit added there —
larckdrakengold's, for #2 — would never have reached the repository. It is in
the list now.

The two other tracked files the archive leaves out, `START-HERE.md` and
`SHA256SUMS.txt`, are the per-release notes and checksums, which are produced
alongside the packages. They stay out.

## Pinned by

**`tests/test-zip-entry.py`** (new, eight tests):

- every argument reaches the launcher untouched, spaces included;
- a bare `./gobbonet` reaches it with no invented arguments;
- it works with the launcher at `0644`;
- it works through a symlink from another folder;
- a missing launcher gets a clear message and exit 1;
- the script parses;
- `make-zip.sh` ships it, checks it, and never copies a top-level `gobbonet`
  from the tree.
- `make-zip.sh` carries `CONTRIBUTORS.md` (an eighth test).

Against the original `make-zip.sh`, the last test fails, naming the leftover
copy.

**`tests/test-linux-onboarding.py`** gains `GOBBONET_TEST_ENTRY` (default
unchanged), so the real first-run test can start from an extracted ZIP's
`./gobbonet` instead of the launcher.

## Verified

From the built 1.7.7 archive, extracted fresh:

- **`GOBBONET_TEST_ENTRY=<zip>/gobbonet` passes both runs**, local and LAN:
  first run, setup page, password, port, remote switch, browser opening, repeat
  launch.
- **`./gobbonet version`** reports the build, passed through to the server.

The engine was a stub, since the test switches to a remote backend before one
would load a model.

## Not changed

- `LINUX-START-HERE.md` already says `./gobbonet`; it is now true.
- The top-level `gobbonet` that a developer's `go build` leaves behind is still
  gitignored, and is simply no longer packaged.
