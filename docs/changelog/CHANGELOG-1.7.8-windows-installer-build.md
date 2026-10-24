# 1.7.8 — The Windows installer: built to match 1.7.7, and checked against it

> I don't want the installers to change anything that they don't have to since
> the setup has worked so well for people in 1.7.7. Start with windows, produce
> the zip and a copy of the exe beside it.

## How it was built

**The same NSIS as 1.7.7.** The official `GobboNetSetup-1_7_7.exe` was
compiled by NSIS 3.12:

- its installer stub is timestamped 2026-04-19 20:38 UTC, six minutes after
  the NSIS `v312` tag;
- it is the large-strings build, because its compiled script reserves 8192
  characters per string slot (the section record is 16,408 bytes).

`START-HERE.md` and `docs/BUILDING.md` said NSIS 3.09; both now say 3.12
large strings.

No NSIS 3.12 binary could be downloaded here. It was built from the NSIS
`v312` source:

- **Toolchain:** SCons 4.8.1 and the llvm-mingw 20260922 cross compiler.
- **Options:** `NSIS_MAX_STRLEN=8192 NSIS_CONFIG_CONST_DATA_PATH=no`.
- **Patches:** three build-only patches, kept in
  `installer-win/nsis-3.12-llvm-mingw.patch`.
  - **Linker flags.** lld has no `--exclude-libs` and no linker scripts, so
    both are dropped. The stub's sections still come out in the order
    makensis needs: `.text .rdata .data .ndata .rsrc`, with `.ndata` last
    before `.rsrc`. Instead it gets `--gc-sections`, without which lld keeps
    all 64 KB of llvm-mingw's one-object `libuuid`.
  - **`memset`/`memcpy`.** NSIS's own copies get C linkage and `volatile`, so
    clang compiles them as functions and does not turn the loop back into a
    call to itself.
  - **`.func`/`.endfunc`.** These are debug-only directives in the System
    plugin's assembly, which clang does not have, so they are skipped under
    clang.

**It reproduces 1.7.7.** Two inputs went into a rebuild:

- the `v1.7.7` script;
- 1.7.7's own payload, extracted from the official installer.

The rebuild's compiled script matches the official one byte for byte, 70,830
bytes, except 97 bytes. Those 97 are data offsets that move by 1,024 because
clang's `nsDialogs.dll` is 1 KB bigger than MSVC's. Decoded into an
instruction listing, the two are identical.

**What is not the same:** the stub and plugins are compiled from the same NSIS
3.12 source, but by clang rather than MSVC, so they are not the same binaries.
The manifest's description reads "Nullsoft Install System v3.12", where the
official build's reads "v23-May-2026.cvs". Like 1.7.7, the installer is not
code-signed.

**Staged from the ZIP.** The new `installer-win/stage-payload.sh` takes its
file list from the script's own `File` lines and copies from the unpacked
release ZIP. Every file in the installer is byte-identical to the same file in
the ZIP delivered beside it (65 files, checked after extracting the installer
with 7-Zip).

## What the 1.7.8 installer does differently from 1.7.7

Both installers were decoded into instruction listings, for the installer and
for the uninstaller it writes, and the listings were diffed. The differences
are exactly the script's diff since `v1.7.7`:

| Change | Why |
|---|---|
| 1.7.7 → 1.7.8: welcome title, `DisplayVersion`, file version, file name | the release |
| **GobboNet** (Start menu and desktop) and the finish page run `gobbonet.exe launch` instead of `launch.bat`; new **Launch GobboNet** and **Setup or add a model** links | the 1.7.6 hotfix's patch 3, integrated at the maintainer's direction this release (`CHANGELOG-1.7.8-hotfix-installers.md`) |
| The Defender page's last paragraph names the cost of an exclusion | issue #63 |
| **GobboNet LAN Setup** is flagged *Run as administrator* | issue #63: it ran `setup-lan.bat` unelevated, which stops at "must be run as Administrator" |
| `stop-gobbonet.bat` is told which install to stop (`/quiet "$INSTDIR"`), at install and uninstall | issue #63 / PR #59: it stopped other installs' processes |
| `engine.sha256` is installed, and removed on uninstall | issue #63: `launch.bat` reads the llama.cpp pin from it |
| Uninstall also removes `.gobbonet-setup-pending` and the two new links | follows from the rows above |

Nothing else changed: pages, texts, colours, password and port handling,
registry entries, folder, compression and the rest of the uninstaller are
1.7.7's.

The payload is 1.7.7's 64 files plus `engine.sha256`, with this release's
contents.

## Not checked

- **Running it on Windows:** installing, upgrading over 1.7.7, the finish
  page, the shortcuts, uninstalling. There is no Windows or Wine here, so
  everything above was read out of the built file rather than seen working.
- **SmartScreen's verdict** on an installer built by a different compiler.
  Like 1.7.7, it is unsigned.

## Tests

`tests/test-windows-staging.py` runs `stage-payload.sh` against the repo with
a stand-in `gobbonet.exe`. It checks that:

- exactly the files `gobbonet.nsi` installs are staged, byte for byte, from
  where `README.md` says;
- staging refuses without `gobbonet.exe`;
- `installer-win/README.md` names the script and the NSIS build.

`tests/test-windows-installer.py` and `tests/test-windows-launch.py` still
hold the script to 1.7.7's pages and the hotfix's shortcuts.
