# 1.7.7 — Version strings

Release housekeeping, done now at the maintainer's request, so the test build of
batches 1–3 identifies itself as 1.7.7.

## Changed

| File | Was | Now |
|---|---|---|
| `VERSION` | `1.7.6` | `1.7.7` |
| `js/01-config.js`, `GOBBONET_UI_VERSION` | `'1.7.6'` | `'1.7.7'` |
| `installer-win/gobbonet.nsi`, `APPVER` | `"1.7.6"` | `"1.7.7"` |
| `installer-win/gobbonet.nsi`, `OutFile` | `GobboNetSetup-1_7_6.exe` | `GobboNetSetup-1_7_7.exe` |
| `installer-win/gobbonet.nsi`, `VIProductVersion` | `"1.7.6.0"` | `"1.7.7.0"` |

**Everything else derives from `VERSION`:**

- the build stamp, `1.7.7-go-<sha>`, via `make-zip.sh` and `build-release.sh`;
- `installer/gobbonet.nsi` (a `-D` define from `build-installer.sh`);
- `build-deb.sh`;
- `build-rpm.sh` and the spec's `%{gn_version}`.

`installer-win/` carries its own literals. Its README says only `APPVER` changes
between releases and the other two "track it", so all three move together.

## Checked

- **`tests/test-version-stamp.mjs`** holds the page stamp equal to `VERSION`.
  Passes.
- **`make-zip.sh`** refuses to build if the binary's stamp or the page's version
  disagree with `VERSION`. Built clean as `1.7.7-go-<sha>`.
- **`TestVersionFileMatchesUpstreamRelease`** compares `VERSION` with the
  nearest `v*` tag. In the authoring tree the bump commit is tagged `v1.7.7`
  **locally** (never pushed), so the test checks the bump rather than failing on
  it.
  - In the published repository, the real `v1.7.7` tag goes on the release
    commit when 1.7.7 is released, as `docs/BUILDING.md` already says. Until
    then, this one test reports the gap, which is what it is for.

## Not changed, on purpose

These name files that were actually shipped for 1.7.6:

- `START-HERE.md`
- `LINUX-START-HERE.md`'s title and package filenames
- `installer-fedora/FEDORA.md`
- the root `SHA256SUMS.txt` (checksums of the 1.7.6 binaries)

1.7.4 set the rule (its roadmap, item 13): bumping them before the 1.7.7
packages exist would point people at files that are not there. They move with
the packaging step.

Comments that record history — "(1.7.6)", "until 1.7.6" — and the `.spec`'s
changelog entries are history and stay.

## Second pass — before the installers (2026-10-12)

A sweep of the tree for any other release number, at the maintainer's request.

**Nothing else names 1.7.6** except history: comments recording when something
changed ("(1.7.6)", "until 1.7.6"), and the Fedora spec's changelog entries.

### Fixed

**`docs/BUILDING.md`** opens by warning that a version in a filename is wrong
from the next release onward. Its body still said:

- `VERSION` "already says `1.7.4`";
- tag `v1.7.4`;
- the outputs are named `GobboNetSetup-1.7.4-go-<sha>.exe` and
  `gobbonet_1.7.4+go.<sha>-1_amd64.deb`;
- `VERSION` "is the only version literal".

The last stopped being true when `installer-win/` gained its own. Now:

- the commands read `<VERSION>`;
- the tag instruction reads `v` + `VERSION`;
- the guide names the two literals that move with `VERSION`;
- the old verification note says it was for 1.7.4.

### Guarded

`installer-win/gobbonet.nsi`'s `APPVER`, `OutFile` and `VIProductVersion` were
held to `VERSION` by nothing but its README ("only `APPVER` changes … the other
two track it"). The new **`tests/test-version-strings.py`** fails if any of the
three disagrees with `VERSION`, or names another release's installer. It also
holds `installer/gobbonet.nsi` to its `-D` define.

Set to 1.7.6 for a moment, `APPVER` fails the test with: "should say `!define
APPVER "1.7.7"` (VERSION is 1.7.7)".

### Still left for the installer step

These name real package files, build stamps and checksums, which exist only once
the 1.7.7 packages are built:

- `START-HERE.md`;
- `LINUX-START-HERE.md`;
- `installer-fedora/FEDORA.md`;
- the root `SHA256SUMS.txt`;
- a `1.7.7-1` entry in `installer-fedora/gobbonet.spec`'s changelog.

