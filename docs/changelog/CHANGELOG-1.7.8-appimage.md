# 1.7.8 — An AppImage, built from the .deb

From issue #53 (Deven-Eve), who built a GobboNet AppImage from the official
`.deb`, scripted it, and tested it on CachyOS with KDE Plasma.

## What it is

`installer-linux/build-appimage.sh` turns the `.deb` into
`GobboNet-<deb version>-x86_64.AppImage`: one file, nothing to install,
including on distributions the `.deb` and `.rpm` don't cover. It is the
package's own files, so the binary, engine, launcher and catalogue are the
same bytes.

```sh
cd installer-linux
./build-deb.sh && ./build-appimage.sh
```

## How it's built

1. Unpack the `.deb` into an AppDir. Uses `dpkg-deb`, or `ar` + `tar` on a
   system without dpkg.
2. Hold the payload to `payload.manifest`, check the engine as `build-deb.sh`
   does, and check the size floor.
3. Add `AppRun`, the menu entry with `Exec=gobbonet`, and the icon.
   `usr/bin/gobbonet` becomes a relative link: in the `.deb` it is absolute,
   and inside an image that points at the host. Any other absolute link fails
   the build.
4. Make a zstd squashfs of the AppDir and append it to the AppImage type 2
   runtime.
5. Check the result:
   - it has the type 2 magic;
   - it begins with exactly the pinned runtime;
   - its `.upd_info` and `.sha256_sig` sections are empty.

Two things differ from the approach in the issue:

- **No appimagetool.** It is itself an AppImage, and fetches a runtime of its
  own at build time. Here the runtime is the only download, pinned by SHA-256
  like the engine:
  - release `20251108` of AppImage/type2-runtime, a dated tag rather than
    `continuous`, which upstream replaces in place;
  - the hash was checked against the release's `runtime-x86_64.sig`, which
    verifies with `signing-pubkey.asc` from that tag (key
    `570C 77AC EA40 C0F1 B758 902C BF96 CCA5 6490 F695`);
  - a different release is refused unless its hash is passed too.
- **No update information.** An AppImage can carry an address it checks for
  newer versions at. This one carries none, and the build fails if the runtime
  ever arrives with one. To update, download the new AppImage.

The runtime reads only zlib and zstd images. An xz image builds fine and then
fails to mount with a message about FUSE, so the build makes zstd and refuses
a `mksquashfs` without it.

With `SOURCE_DATE_EPOCH` set, the same `.deb` gives a byte-identical AppImage.

## What changed in the program

- **`gobbonet uninstall`** said to "delete the folder you unpacked GobboNet
  into". From an AppImage that folder is a temporary read-only mount. It now
  names the `.AppImage` file to delete.
  - It goes by `APPIMAGE` only when the binary is inside `APPDIR`. Every child
    of an AppImage inherits both, so a terminal that is itself an AppImage
    would otherwise hand its own file to a `.deb` GobboNet started from it.
- **The launcher checks for python3** before the first-run setup, which needs
  it. The packages depend on python3; the ZIP and the AppImage bring none.
  Without it the first run used to fail with "first-run setup did not finish"
  and `python3: command not found` in the log. Now it says what is missing and
  how to install it, in a dialog when there is one.

### Nothing else needed changing

- The launcher finds its own folder with `readlink -f`, and writes only to
  `~/.config/gobbonet` and `~/.local/share/gobbonet`.
- The engine path saved at setup names that run's mount, which is gone next
  time. `gobbonet serve` already replaces a `server_exe` that no longer exists
  with the engine beside the binary, and saves it.
- The AppImage's own portable mode (`--appimage-portable-home`) works too,
  because the launcher follows `XDG_CONFIG_HOME` and `XDG_DATA_HOME`.

## Needs on the user's machine

- x86_64, glibc 2.34 or newer (the engine's floor, read from the `.deb`).
- `bash`, `fusermount`, and `python3` for the first-run setup. Without FUSE,
  run it with `--appimage-extract-and-run`.
- The engine's libraries come from the host, as for the `.deb`: libstdc++,
  libgomp and OpenSSL 3, plus a Vulkan driver for the GPU. libstdc++ is not
  bundled because the host's Vulkan driver loads the host's own copy.

## Checked here

Built from a `.deb` of this branch, on Ubuntu 24.04 (container):

- the AppImage holds exactly the `.deb`'s files and modes, apart from the
  `usr/bin/gobbonet` link;
- it runs through FUSE: `version`, and `--appimage-updateinformation`, which
  prints nothing;
- `tests/test-linux-onboarding.py`, with `GOBBONET_TEST_ENTRY` set to the
  AppImage, passes both runs: local and LAN, with the location, password and
  port steps, then a repeat launch;
- a relaunch with a `server_exe` naming an old mount heals it to the new one;
- the mounts are gone once GobboNet exits.

Not checked here: a real desktop session (menu integration by Gear Lever or
AppImageLauncher, the browser opening), Vulkan on real GPUs, and distributions
other than Ubuntu. The issue's CachyOS and KDE test covers the first.

## Tests

- **`tests/test-appimage.py`, 12 checks:**
  - the runtime pin and the refused override;
  - no update address and no appimagetool;
  - `AppRun`'s handover;
  - the missing-python3 message;
  - with `GOBBONET_TEST_DEB` and `GOBBONET_TEST_APPIMAGE_RUNTIME`, a full build
    in a scratch tree: the file layout, the contents against the `.deb`, a run,
    and refusals on a wrong runtime hash and a CPU-only `.deb`.
- **`cmd/gobbonet/uninstall_test.go`:** the AppImage hint, and six cases where
  `APPIMAGE` must not count.
