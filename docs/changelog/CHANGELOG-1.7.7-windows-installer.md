# 1.7.7 — Windows installer: the dark installer, as in 1.7.5

The maintainer's direction, 2026-10-12: "We want installers to act the **SAME
WAY** they did in the 1.7.5 [not 1.7.6]." Windows comes first; Linux and Fedora
have their own steps and follow separately.

## The wrong installer, built first

The first 1.7.7 test build was made from `installer/`. It was rejected:

> It's not dark mode. Its asking us to fetch the model during the install...
> INITIAL INSTALL PAGE > DESTINATION > WHERE MODELS RUN [shouldn't even BE
> HERE this is LINUX ONLY ON THE INSTALLER RN] > Detect hardware > NO PICKING A
> MODEL UNTIL CMD POPS UP > Start Gobbonet or launch phone access.

That commit is reverted. The question that sent it there was mine and was
wrong: it offered `installer/` as "the 1.7.4/1.7.5 setup wizard".

## Which installer is the real one

Git and the docs in it agree:

- **`installer-win/` is the Windows installer.**
  - Its README, added in the maintainer's 1.7.3 commit (`c89461d`), says it
    "built `GobboNetSetup-1_7_0.exe` and everything before it".
  - It also says `installer/` "has never shipped. Build from **this**
    directory".
  - 1.7.3 and 1.7.6 were built from it.
- **`installer/` is a Go-era design that was never meant to ship.** It picks a
  backend and downloads a model during setup.
  - It was built by mistake for 1.7.4: `CHANGELOG-1.7.4-windows-installer.md`,
    and `INVESTIGATION-cmd-visuals.md`, which calls 1.7.4 "the release where
    the installer was actually built and handed out".
  - It was built by mistake again here.
- **What made the mistake easy:** `docs/BUILDING.md` listed
  `installer/build-installer.sh` as the Windows build, the only scripted
  Windows build in the tree.

## What the user goes through

`installer-win/`'s pages. Nothing about them changed in 1.7.6 or here:

1. **Welcome.** Dark, with the 1.7 art.
2. **Destination:** `%LOCALAPPDATA%\GobboNet`. Per-user, no administrator.
3. **Access Password:** password and confirm, or "Skip - ask me on first run
   instead".
4. **Network Port:** 9066 unless changed.
5. **Install:** copies files. Nothing is downloaded.
6. **One Thing Before You Start:** the Windows Defender notice, with the folder
   to exclude.
7. **Finish:**
   - [x] Launch GobboNet
   - [ ] Set up phone access over the LAN (needs administrator)

**Launch GobboNet opens `launch.bat` in its console window,** as do the Start
menu and desktop shortcuts. On the first run that window:

1. asks for a password if one was skipped;
2. downloads llama.cpp;
3. **detects the hardware;**
4. **offers the model list, recommendation marked;**
5. downloads the model;
6. hands over to `gobbonet.exe` and opens the browser.

**Two places this differs from the flow as you wrote it:**

- **"Detect hardware" happens in that console window, just before the model
  list.** No version of this installer has a hardware page.
- **The Password, Port and Defender pages are not in your list,** but every
  version of this installer has had them, 1.7.5's included. They are kept.

## What changed for 1.7.7

**`models.ini` is installed beside `gobbonet.exe`.**

- **Why it's needed:** two parts of CONFIG → *Add a Model* need it:
  - **The model list offline.** When goblincorps.com cannot be reached, the
    list falls back to it.
  - **The type buttons.** *Coding*, *Roleplay* and *Jack of all trades* take
    each model's type from it until the live list carries the tags.
- **Without it,** an installed copy had no list offline, and every type button
  matched nothing.
- **Where it comes from:** `installer/models.ini`, generated from `launch.bat`.
  That is the same file the zip and the Linux packages carry.
- **The uninstaller removes it.**

**Kept from 1.7.6**, because 1.7.7 needs `gobbonet.exe` too. It is what
`launch.bat` hands over to, and the phone-address box, *Add a Model* and device
sync live there. These additions are behind the scenes and change no page:

- **`gobbonet.exe` is installed.**
- **A running GobboNet is stopped before files are replaced.**
- **The uninstaller runs `gobbonet uninstall --yes --keep-models`,** so the
  Go server's conversations and settings go with the program. That matches
  1.7.5's rule for the PowerShell server's conversations: "Uninstall means
  uninstall."

**Checked, nothing to change:**

- **Helpers:** every helper `launch.bat` runs is installed.
- **Version:** the version literals say 1.7.7 (from batch 3).
- **The port:** the port page and the phone-address box both write
  `.gobbonet-port`, which `launch.bat` reads.
- **The password:** the installer's password is in the form `gobbonet.exe`
  accepts and upgrades on first sign-in.

## Built, not run

`GobboNetSetup-1_7_7.exe`, about 3.7 MB. It was staged exactly as
`installer-win/README.md` lists, with the zip's `gobbonet.exe`.

**Checked without Windows:**

- **makensis:** 0 warnings, 7 install pages and 4 uninstall pages.
- **Extracted with 7-Zip:**
  - the files match what was staged;
  - the `gobbonet.exe` inside reports the zip's build stamp.
- **Version info:** ProductVersion `1.7.7`, FileVersion `1.7.7.0`.

**NSIS 3.12, not 3.09:**

- **Why:** the build sandbox could not reach SourceForge or the distro
  mirrors.
- **What it is:** the official 3.12 Windows stubs and plugins (the
  large-strings build), with a Linux `makensis` compiled from the v3.12
  source, from electron-builder-binaries' `nsis@1.2.1` bundle.
- **Rebuilding with 3.09** changes nothing in the script.

**Not run on Windows.** Worth checking:

1. **Every page is dark,** including the password, port and Defender pages.
2. **Launch GobboNet opens the console:**
   - it detects the hardware;
   - it offers the model list;
   - after the download, the browser opens on the chat.
3. **The password set in the installer works** in the browser.
4. **CONFIG → Add a Model** shows the list, and the type buttons filter it.
5. **Installing over 1.7.6** and **uninstalling.**

## Noticed, not changed

**The console fetches a different llama.cpp from the one the Linux packages
use.**

- **Windows:** `launch.bat` downloads `b9294`, against its own pin
  (`LLAMA_PIN_TAG`).
- **The `.deb` and `.rpm`:** they use `engine.sha256`'s `b10456`.
- **Since when:** this is 1.7.5's behaviour, unchanged here.

## Guarded

**`tests/test-windows-installer.py`** (new, 12 tests). It holds:

- **`installer-win/`** to:
  - its seven pages in order;
  - no backend, model or download step;
  - the dark theme, on every custom page as well;
  - *Launch GobboNet* and the shortcuts opening `launch.bat`;
  - phone access unticked.
- **`launch.bat`** to probing the hardware, and to handing over with
  `--open --model`.
- **The payload:**
  - every helper `launch.bat` runs is installed;
  - `models.ini` is installed and removed;
  - every installed root file is removed on uninstall;
  - `gobbonet.exe` is stopped before it is replaced.
- **The build guide:** `docs/BUILDING.md` names `installer-win/`, and its build
  order no longer runs `build-installer.sh`. `installer/README.md` opens with
  "do not ship".

The `models.ini` and build-guide checks fail on the tree before this change.
The rest guard what was already right.
