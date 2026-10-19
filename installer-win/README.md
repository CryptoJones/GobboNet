# Windows installer

**This is the Windows installer.** Build releases from here, not from
`installer/`; see "Not to be confused with `installer/`" below.

This is the script that built `GobboNetSetup-1_7_0.exe` and everything before
it. It installs the batch/PowerShell path — `launch.bat` and `fileserver.ps1` —
and needs no Go binary.

```sh
# stage the payload next to this directory as ../staging, then:
makensis gobbonet.nsi
```

Only `!define APPVER` changes between releases. `OutFile` and
`VIProductVersion` track it.

## 1.7.6: the same installer, plus gobbonet.exe

Same pages, same first-run flow (`launch.bat` fetches llama.cpp and offers the
model catalogue on first launch); nothing is downloaded during setup. Three
additions, all forced by shipping the Go server:

- **`gobbonet.exe` is installed.** `launch.bat` hands the server role to it
  when present. `fileserver.ps1` alone has no per-conversation sync routes, so
  1.7.6's chat backup and device sync would fail without it.
- **GobboNet is stopped before files are replaced or removed**
  (`stop-gobbonet.bat /quiet`), because a running `gobbonet.exe` holds its own
  file open.
- **Uninstall runs `gobbonet uninstall --yes --keep-models`**, which removes
  the conversations and settings the Go server keeps in the user profile, then
  deletes `gobbonet.exe` and the `web\` folder it writes. Models are still the
  uninstaller's own question.

Staging for a build: copy into `../staging` the files the script lists
(`launch.bat`, the other `.bat`/`.ps1` helpers, `chat.html`,
`default-characters.json`, `LICENSE`, `TROUBLESHOOTING.md`, `SECURITY.md`,
`js/`, `css/`), plus:

- `gobbonet.exe` from `make-zip.sh`;
- `PURGE.md` from `docs/PURGE.md`. It moved there in 1.7.2, but the install
  keeps it at the root, where `TROUBLESHOOTING.md` points;
- `models.ini` from `installer/models.ini` (1.7.7).

## 1.7.7: the same pages again

The maintainer's direction: the installer acts the way it did in 1.7.5. The
pages are the ones this script has always had:

1. Welcome
2. Destination
3. Access Password (or "Skip - ask me on first run instead")
4. Network Port (9066)
5. Install. It copies files only and downloads nothing.
6. One Thing Before You Start, the Windows Defender notice
7. Finish: [x] Launch GobboNet, [ ] Set up phone access over the LAN

*Launch GobboNet* opens `launch.bat` in its console window. That window
fetches llama.cpp, detects the hardware and offers the model list, then hands
over to `gobbonet.exe` and opens the browser.

One addition: **`models.ini` is installed.**

- *Add a Model* in CONFIG falls back to it when goblincorps.com can't be
  reached.
- Its *Coding*, *Roleplay* and *Jack of all trades* buttons take each model's
  type from it until the live list carries the tags.

Without it, an installed copy had no list offline, and the type buttons
matched nothing. See `docs/changelog/CHANGELOG-1.7.7-windows-installer.md`.

## Not to be confused with `installer/`

`installer/gobbonet.nsi` is a different, unreleased script for a Go-server
Windows install: it adds backend, GPU-probe and model-download pages, bundles
`gobbonet.exe` and llama.cpp, and downloads a model during setup behind an
`inetc /BANNER` popup. Build from **this** directory unless you are
deliberately working on that one.

It was built by mistake twice: for 1.7.4, and for the first 1.7.7 test build.
Both times it replaced this installer's flow with one the maintainer did not
want. `tests/test-windows-installer.py` now holds this script to its pages and
`docs/BUILDING.md` to naming it.

## art/

`art/modern-header.bmp` carries the version as painted pixels — `v1.7` covers
the whole 1.7.x line. It is not generated, so it does not follow `APPVER`; on a
minor bump the bitmap has to be repainted or the wizard shows the old number
while every other surface shows the new one. The copy under `installer/art/`
was stale at `v1.3` for exactly this reason and has been replaced with the
shipped one.
