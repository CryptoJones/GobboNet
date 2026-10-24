# 1.7.8 — The 1.7.6 hotfix, installer side

The 1.7.6 hotfix (cumulative patch 3, released as `GobboNet-1.7.6-hotfix.zip`
and `GobboNetSetup-1.7.6-hotfix.exe`) was carried forward in two halves.
The application half went into 1.7.7, in commit f2a0d4a, *1.7.7 hotfix
carryover* (`CHANGELOG-1.7.7-hotfix-integration.md`), which left "installer
integration" for later. This is that half, in 1.7.8.

## How it was checked against the hotfix

The hotfix ZIP was compared file by file with its base, 314bdc9. Then each line
it added was looked for in 1.7.7 and in this branch:

- **Application** (Go server, chat page, repair scripts, tests): every added
  line is in 1.7.7, and this branch kept all of it.
- **The hotfix installer's 57 payload files** are byte-identical to the ZIP's.
  So the installer source in the ZIP is what that installer shipped.
- **Missing in both 1.7.7 and this branch:**
  - the installer changes;
  - the `.deb` build fix;
  - the hotfix's installer test;
  - the packaging changelog;
  - a `.gitignore` line.

  They are below.

## The change

### Windows installer (`installer-win/gobbonet.nsi`)

As in the hotfix, on top of 1.7.7's script, which keeps its own changes: the
LAN Setup link flagged *Run as administrator*, `engine.sha256`, `models.ini`
and the Defender page.

- **GobboNet** in the Start menu and on the desktop runs `gobbonet.exe
  launch`, not `launch.bat`.
  - It serves the saved settings and opens the browser, with no batch
    questions.
  - With no settings yet, it runs `launch.bat`'s first-run setup.
- **Launch GobboNet** is a new link to the same command.
- **Setup or add a model** is a new link to `launch.bat`: the engine download,
  hardware check and model menu.
- The finish page's *Launch GobboNet* runs the same command, in the install
  folder.
- Every link is recreated on each install. Updating a 1.7.7 install made from
  the old script re-points its links.
- **The uninstaller removes the two new links.** Before this, a link the
  hotfix installer had made was left in the Start menu after uninstalling,
  and kept the GobboNet folder there from being removed.

### The unreleased Go installer (`installer/gobbonet.nsi`)

The hotfix's changes as they were:

- the main and desktop links, and the finish page, run `gobbonet.exe launch`;
- a **Launch GobboNet** link is added;
- each link starts in the install folder.

### An interaction with #63, handled

`gobbonet.exe launch` reruns setup only when there is no `config.toml`. On
this branch, `launch.bat` writes the password into `config.toml` early, right
after the password step (#63, so a first run that stops does not ask for a
second password). That early write is the first thing to create the file.

So a first run closed during the llama.cpp download or the model menu left a
`config.toml` with a password and no engine. With these shortcuts, every click
on GobboNet would then have served that config, opening a chat with no model,
and setup would never have been offered again.

Now:

- **The mark:** `launch.bat` leaves `.gobbonet-setup-pending` beside itself
  from that early write until the end of setup (`:launch`), where it is
  deleted.
- **What `launch` does with it:** on Windows, when the mark is there and the
  config names no engine (`server_exe` empty), it runs setup instead of
  serving (`launchSetupUnfinished`, `cmd/gobbonet/launch.go`).
- Both are needed. *Setup or add a model* also runs `launch.bat` on a working
  install, and stopping that leaves the mark beside a config whose engine is
  set. That install keeps serving.
- **Unchanged:**
  - a config with no engine and no mark is still served as remote mode, as
    the hotfix's own test requires;
  - `--config` and Linux behave as before.
- The Windows uninstaller deletes the mark, and `.gitignore` lists it.

### Linux `.deb` (`installer-linux/build-deb.sh`)

The engine size check now follows symlinks (`stat -L`), as in the hotfix.

- b10456 ships `libllama.so -> libllama.so.0 -> libllama.so.0.1.0`, and the
  check takes whichever of those `find` lists first.
- When that is a link, it reads 13 bytes and stops the build with "only 0 KB".
- It passed here only because this machine lists the real file first.
- Fedora's `build-rpm.sh` already matches regular files only (`-type f`), so
  it needs no change.

The hotfix's other `.deb` fix, running its checks through `bash`, was already
in 1.7.7.

### Also from the hotfix

- `tests/test-windows-launch.py`, the hotfix's own test, taken as it was. It
  holds both installer scripts and the ZIP's repair script to the shortcut
  contract. Added to it: the uninstaller removes both new links, and the ZIP
  ships the repair scripts.
- `docs/changelog/CHANGELOG-1.7.6-packaging.md`, verbatim. It describes the
  1.7.6 packages, not this change.
- `.gitignore`: `legacy-import.json`, the record the history import leaves
  beside `state.json`.
- `installer-win/README.md`: the installer ships `gobbonet.exe` (since 1.7.6),
  and a section on the new links. `tests/README.md`, `AGENTS.md` and the
  README's quick reference say the same.
- `PATCH-README.txt` and `CHANGELOG-1.7.7-hotfix-integration.md` no longer say
  installer integration is still to come.

## Checked here

- **Against the hotfix:**
  - its test fails on 1.7.7 (at its first installer check) and passes now;
  - comparing again, every installer, packaging and test line the hotfix
    added is in this branch.
- **The setup mark**, in Go tests:
  - no mark: a config with no engine is served (remote mode);
  - a mark with no `launch.bat` beside it: served;
  - mark plus no engine: setup runs again;
  - mark plus an engine: served;
  - the config file is not touched.

  `tests/test-launch-secret-handoff.py` checks that `launch.bat` sets the
  mark inside the early password block and clears it at `:launch` after the
  engine is written. It also checks that `launch.go` and the uninstaller use
  the same name, and that `launch.bat` is still CRLF.
- The full suites pass, and the ZIP was rebuilt and checked from its own
  copy.
- **Not checked:**
  - **No NSIS compiler is available here**, so neither installer script was
    compiled. The checks are on the source.
  - **Nothing ran on Windows**: no real install, update over the hotfix
    installer, uninstall, or first run closed partway.

  One install and one uninstall on Windows would cover what is left.
