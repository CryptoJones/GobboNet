# 1.7.8 — Docs that describe the program that ships

From issue #63 (first-install findings, Solteris-Dev), the docs batch. New
users and AI agents both read these files first, and they described an older
app.

## What was wrong, and what it says now

- **`README.md`**
  - Said Windows only ("doesn't work on Mac or Linux as written"). It now names
    Linux, links `LINUX-START-HERE.md` for the `.deb`, `.rpm`, AppImage and
    ZIP, and says Mac isn't supported.
  - Named `launch.exe` and `launchLAN.exe`, removed in 1.7. It now names
    `launch.bat`, the GobboNet shortcut, and `setup-lan.bat` with *Run as
    administrator*.
  - Gave the installer as 660 KB (1.3's size; 1.7.6's was about 3.9 MB). No
    size now, since any number goes stale with the next release.
  - Mentions Firefox's "not commonly downloaded" hold beside the SmartScreen
    note.
  - "Upcoming changes" said GobboNet was Windows-only today.
- **`TROUBLESHOOTING.md`**
  - Opens with `gobbonet doctor`, and says the server is `gobbonet.exe` since
    1.7. `fileserver.ps1` is only the legacy fallback.
  - *The chat page will not load* quotes the Go server's own messages and its
    `startup-error.log`. The `fileserver.log` table is kept, marked legacy.
  - *Other bind causes*: the Go server's fallback to this PC, and the firewall
    rule. URL reservations and HTTP.SYS apply to the legacy server only.
  - *Password problems*: what `launch.bat` does with `.gobbonet-secret`, why
    `set-password` is the Linux answer and not the Windows one, and a new
    *"My phone keeps asking me to sign in"* (below).
  - *The model will not load*: `llama-server.log` is in the data folder.
  - *Linux / Wine* said "Not supported yet". Linux is supported natively; the
    section now points at `LINUX-START-HERE.md` and `launch.log`.
  - *Still stuck* asks for `gobbonet doctor`, COPY BUILD INFO, and the
    launcher's output, not `fileserver.log`. Security reports go to Elodine on
    Discord.
  - One-run ports use `GOBBONET_LISTEN_PORT`, the current spelling.
- **`docs/PURGE.md`** described `.gobbonet-state.json` in the install folder as
  the server's copy. Since 1.7 it is `state.json` (plus device-sync backups and
  the keyring) in the data folder, cleared by `gobbonet uninstall`, which the
  Windows uninstaller runs. The old file is covered as what an install from
  before 1.7 left behind. Commands for both Windows and Linux.
- **`docs/GO_SERVER.md`** described a `web/` folder beside the binary. The page
  has been compiled in since 1.7.5.
- **`docs/GO_CONFIG_SPLIT.md`** opened with text pasted from the assistant that
  drafted it. It now opens with a status block: what the note is, and what
  changed since. There is no `launch.sh`, and the config is at
  `~/.config/gobbonet/config.toml` on both platforms.
- **`installer-win/gobbonet.nsi`**'s header said `launchLAN.exe` handles
  elevation; the file itself says `launchLAN.exe` is gone. The header now
  points at `RunLanSetup` and its runas verb.
- **`AGENTS.md`**: the *Docs known to be stale* list is replaced by what is
  historical on purpose (`GO_CONFIG_SPLIT.md`, the changelog, anything about
  `fileserver.ps1`).

## Sessions, said where people meet them

A phone asked to sign in again looked like a fault. A sign-in is in memory
(gone when GobboNet restarts), lasts 12 hours by default
(`session_ttl_hours`), and is tied to the device's address and the browser's
identity. So a phone changing network address, or a browser updating, signs in
again. The login page now says so in one line under the plain-text note, and
TROUBLESHOOTING.md explains it.

## The cost of a Defender exclusion

The installer's Defender page and TROUBLESHOOTING.md told people to exclude
the GobboNet folder without saying what that gives up. Both now say it: an
excluded folder is not scanned, so keep only GobboNet in it. The installer
paragraph was reworded to stay within its four lines.

## Found on the way: the LAN Setup shortcut could not elevate

`setup-lan.bat` stops at "This script must be run as Administrator" unless it
is elevated. The finish page's checkbox asks with the runas verb. But the
Start Menu's **GobboNet LAN Setup** shortcut ran it as a plain batch file.
That is the shortcut README.md sends people to for phone access. It broke in
1.7, when `launchLAN.exe`, which asked for elevation in its manifest, was
removed.

The shortcut now carries the *Run as administrator* flag: the installer sets
`SLDF_RUNAS_USER` (0x2000) in the `.lnk` header's LinkFlags, OR-ed in after
`CreateShortcut`, which has no option for it. The main GobboNet shortcuts are
untouched. Installs that already exist keep their old shortcut until the
installer runs again. Until then, right-click `setup-lan.bat` and choose *Run
as administrator*, as the README says.

Not compiled or run on Windows here; it needs one check on a real install:
open the shortcut's Properties → Advanced, and *Run as administrator* should
be ticked.

## Not in this repository

The website lists the installer as 681 KB and 660 KB; the 1.7.6 file was about
3.9 MB. It should also mention Firefox's download warning next to the
SmartScreen note. Both are for the website itself.

## Tests

- **`tests/test-docs-current.py`, 16 checks.** They hold the specific wrong
  claims out, and tie the facts the docs quote to the code:
  - the server messages TROUBLESHOOTING.md quotes are still the program's own;
  - the session lifetime is the config's default;
  - the files PURGE.md names are the ones `gobbonet uninstall` removes;
  - the commands the docs name exist.
- **`tests/test-windows-installer.py`, +3:** the run-as-administrator flag is
  set on the LAN Setup shortcut only, by OR-ing bit 0x20 into byte 0x15, with
  that byte operation replayed on a shell-link header.
- **`internal/auth/login_test.go`:** the login page's sign-in note, and that
  its figure is the configured default.

Every new check fails on the tree before this change.
