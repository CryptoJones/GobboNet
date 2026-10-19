# 1.7.7 — The build works without the executable bit

Roadmap item #11. Reported against a fresh clone: `./build-release.sh` would not
run at all.

## What was wrong

Every build script in the tree was committed `100644`, and had been since the
day it was added:

```
build-release.sh  make-zip.sh  stage-web.sh  verify-payload.sh
installer/build-installer.sh
installer-linux/build-deb.sh  installer-linux/package-permissions.sh
installer-fedora/build-rpm.sh  check-rpm.sh  test-in-fedora.sh
```

Release commits are made from Windows, where Git does not record the bit. The
only three files that have it (both copies of `gobbonet-launch`, and
`installer/gen-catalog.py`) got it through explicit mode-change commits.

In a clone with every script at `0644` — which is every clone:

```
$ ./build-release.sh
bash: ./build-release.sh: Permission denied
$ bash build-release.sh
build-release.sh: line 81: ./stage-web.sh: Permission denied
```

The second failure is the important one. Working around the first by typing
`bash` does not help, because the scripts call each other as bare paths.

### Worse: two steps that skipped silently

Two places tested for the bit instead of running the file, so a missing bit did
not fail them — it switched them off, without a word:

- **`installer-fedora/check-rpm.sh`** ran its payload verification only
  `if [ -x "$ROOT/verify-payload.sh" ]`. In every checkout, an rpm could pass
  its checks without its payload ever being compared to `payload.manifest`.
- **`installer-linux/build-deb.sh`** regenerated `installer/models.ini` only
  `if [ -x "$ROOT/installer/gen-catalog.py" ]`. That file does have the bit
  today; if it ever lost it, the `.deb` would have shipped whatever catalogue was
  lying around.

## The change

**Scripts run each other through their interpreter.** Every script-to-script
call is `bash x.sh` or `python3 x.py`, never a bare path, and the two `-x`
tests are now `-f`. Eleven call sites:

| File | Line(s) |
|---|---|
| `build-release.sh` | `stage-web.sh` |
| `make-zip.sh` | `stage-web.sh` |
| `installer/build-installer.sh` | `gen-catalog.py` |
| `installer-linux/build-deb.sh` | `gen-catalog.py` (and its `-x` test), `verify-payload.sh`, `package-permissions.sh` |
| `installer-fedora/build-rpm.sh` | `verify-payload.sh`, `check-rpm.sh` |
| `installer-fedora/check-rpm.sh` | `verify-payload.sh` (and its `-x` test) |

**The ten entry points carry the bit** in this tree and in the archive built
from it. That half does not survive a Windows commit, so it also needs one
command on the maintainer's side, once:

```
git update-index --chmod=+x build-release.sh make-zip.sh stage-web.sh verify-payload.sh installer/build-installer.sh installer-linux/build-deb.sh installer-linux/package-permissions.sh installer-fedora/build-rpm.sh installer-fedora/check-rpm.sh installer-fedora/test-in-fedora.sh
```

Even without it, everything now works through `bash build-release.sh` and
friends. The bit only makes `./build-release.sh` work as well.

## Pinned by

**`tests/test-build-scripts.py`** (new, three tests):

- **No script runs another as a bare path or behind `[ -x ]`.** It scans every
  `*.sh` in the tree, skipping build scratch.
- **The check catches what it exists to catch.** It is run against the exact
  lines this change replaced, which must be flagged, and against their
  replacements plus prose that merely names a script, which must not be.
- **Every build script still parses** (`bash -n`).

Run against the original scripts, the check reports all eleven sites.

## Verified

In a copy of the tree with **every** `.sh` set to `0644`, `bash
build-release.sh --allow-dirty` ran to the end and built all five platforms:
linux amd64/arm64, windows amd64, darwin amd64/arm64. The original
`build-release.sh`, dropped into the same copy, stops at line 81 as shown above.

The `.deb` and `.rpm` builders need `dpkg-deb`, `rpmbuild` and the pinned engine
download, which the sandbox does not have. Their changes are covered by the
static check and the parse check, not by a package build.

## Not changed

- `gobbonet-setup.py` stays `0644`. It is run through `python3`, and packaged
  `0644` on purpose (`build-deb.sh`).
- `postinst` and `postrm` are given `0755` when packaged.
- The documented commands (`./build-release.sh` in `docs/BUILDING.md`) are left
  as they are; with the bit set they work as written.
