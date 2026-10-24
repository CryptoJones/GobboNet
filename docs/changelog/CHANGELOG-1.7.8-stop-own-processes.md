# 1.7.8 — The stop script stops only this install's processes

From issue #63 (first-install findings, Solteris-Dev), item 13. Includes
PR #59 by orcmaul, reworked.

## What was happening

`stop-gobbonet.bat` ended `llama-server.exe` by image name. Stopping
GobboNet, or running its uninstaller or installer, also killed any llama.cpp
the user was running on their own. The legacy `fileserver.ps1` was matched by
name anywhere on a command line, the same over-reach with a rarer name.

PR #59 added stopping the `launch.bat` windows. That's useful: on the legacy
path, the health monitor in that window restarts llama-server. But it matched
"launch.bat" on any command line, so it would have closed other programs'
`launch.bat` windows, and possibly the window running the stop script itself.

## The change

**One routine, `:stop_matching`, stops the processes of a given name that
belong to this install.**

- **llama-server:** its executable lives inside the install folder.
- **The `launch.bat` window and the legacy `fileserver.ps1`:** that folder's
  file is on the command line.
- **Never touched:** the window running the script, and every window above it.
- **Quoting:** the folder, name and file reach PowerShell through environment
  variables, so brackets, apostrophes or spaces in a folder name can't break
  it.
- **Unchanged:** `gobbonet.exe` is still stopped first by name. An engine
  configured to run from outside the install stops with `gobbonet.exe`, its
  parent.

**Which install.** By default the script stops processes from its own folder.
Before replacing files, the installer runs a copy of the script from its
temporary plugins folder, so it now passes `$INSTDIR`, both there and on
uninstall.

**From PR #59:**

- the `launch.bat` windows are stopped, now scoped to this install;
- the window pauses only when something would not stop, and otherwise closes
  itself after 10 seconds, or at once on a keypress (the PR's `/nobreak` is
  dropped);
- the wait after a forced kill is longer.

Not taken: the switch from `!_IMG!` to `%_IMG%`. No bug was found with the
original, and the house rule is `!VAR!` inside `( )` blocks.

## Tests

`tests/test-stop-script.py`:

- **Static checks** on the script and the installer.
- **The script's real PowerShell routine**, run in PowerShell 7.4 against a
  fake process table. The stub process table covers:
  - this install's engine, with differing case and a subfolder;
  - the user's own llama.cpp build;
  - a sibling `GobboNet2` folder;
  - a process with no visible path;
  - another program's `launch.bat`;
  - the stop script's own window and the `launch.bat` window it was started
    from;
  - a folder name with brackets, an apostrophe and spaces;
  - a process that won't die (exit code 11).

On the 1.7.7 files the static half fails. Not run in a real Windows console or
in Windows PowerShell 5.1 here; the routine uses only syntax 5.1 accepts.
