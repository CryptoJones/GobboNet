# 1.7.8 — Version strings

Release housekeeping, done at the maintainer's request, so the build carrying
the open-issue fixes (the `fix/quick-wins-61-65-63` branch, on top of the 1.7.7
release f2a0d4a) identifies itself as 1.7.8. It follows the 1.7.7 bump
(`CHANGELOG-1.7.7-version-bump.md`) step for step.

## Changed

| File | Was | Now |
|---|---|---|
| `VERSION` | `1.7.7` | `1.7.8` |
| `js/01-config.js`, `GOBBONET_UI_VERSION` | `'1.7.7'` | `'1.7.8'` |
| `installer-win/gobbonet.nsi`, `APPVER` | `"1.7.7"` | `"1.7.8"` |
| `installer-win/gobbonet.nsi`, `OutFile` | `GobboNetSetup-1_7_7.exe` | `GobboNetSetup-1_7_8.exe` |
| `installer-win/gobbonet.nsi`, `VIProductVersion` | `"1.7.7.0"` | `"1.7.8.0"` |
| `.github/ISSUE_TEMPLATE/bug.yml`, version placeholder | `1.7.7` | `1.7.8` |
| `docs/BUILDING.md`, the tag example | `v1.7.7` | `v1.7.8` |

**Everything else derives from `VERSION`:**

- the build stamp, `1.7.8-go-<sha>`, via `make-zip.sh` and `build-release.sh`;
- the ZIP's name, `GobboNet-1_7_8.zip`;
- `installer/gobbonet.nsi` (a `-D` define from `build-installer.sh`);
- `build-deb.sh`, `build-appimage.sh`;
- `build-rpm.sh` and the spec's `%{gn_version}`.

## This branch's own notes move to 1.7.8

Everything on this branch postdates the 1.7.7 release, so it ships in 1.7.8.
Its changelogs and comments said 1.7.7 because they were written before this
bump.

- **14 changelogs renamed** from `CHANGELOG-1.7.7-*.md` to
  `CHANGELOG-1.7.8-*.md`, with their titles:
  - #41 memory files, #51 banned phrases;
  - #53 AppImage;
  - #61 thread folders;
  - #63: the docs pass, first-run papercuts, home screen, model menu, one
    engine pin, password handoff, stop script, uninstall;
  - #65 MoE small cards;
  - the hotfix's installer side.

  Every link to them was updated. That covers READMEs, tests,
  `PATCH-README.txt`, and two of 1.7.7's own changelogs.
- **Statements about this branch's changes** now say 1.7.8. That covers:
  - the README's *Recently fixed* (banned words) and the AppImage line;
  - `AGENTS.md`'s "as of" line;
  - `RAG_INFO.md`'s memory-files heading;
  - `GO_CONFIG_SPLIT.md`;
  - the installer README's engine pin and hotfix sections;
  - comments in `setup-lan.bat`, `teardown-lan.bat` and the Windows
    installer;
  - comments in `internal/models/classify*.go` and three tests.

  Both batch files stay CRLF.
- **Left as they are:**
  - sentences about the released 1.7.7 itself ("fails on the 1.7.7 files", "in
    1.7.7");
  - 1.7.7's roadmap and changelogs;
  - the `1.7.7` literals that are test fixtures.

## Tag

The bump commit is tagged **`v1.7.8` locally**, and the tag travels in the
bundle. It was not pushed. `TestVersionFileMatchesUpstreamRelease` compares
`VERSION` with the nearest `v*` tag, so the tag goes with the bump. On GitHub,
the real `v1.7.8` belongs on whichever commit is released as 1.7.8, as
`docs/BUILDING.md` says.

## Checked

- `tests/test-version-stamp.mjs`: the page's version equals `VERSION`.
- `tests/test-version-strings.py`: the installer's three literals equal
  `VERSION`, and nothing names another release's installer.
- `TestVersionFileMatchesUpstreamRelease`, with `v1.7.8` in place.
- `make-zip.sh` builds only when the binary's stamp and the page's version
  agree with `VERSION`. It built `GobboNet-1_7_8.zip`, stamped `1.7.8-go-<sha>`.

## Left for the packaging step

The rule from 1.7.4, as in the 1.7.7 bump: these name package files, build
stamps and checksums that exist only once the 1.7.8 packages are built.

- `LINUX-START-HERE.md`, title and package names (still 1.7.7). **It ships
  inside the ZIP.**
- `installer-fedora/FEDORA.md` (1.7.7).
- `START-HERE.md`, still at 1.7.6. The 1.7.7 packaging step did not reach it
  either.
- The root `SHA256SUMS.txt`, checksums of the 1.7.7 package's files.
- A `1.7.8-1` entry in `installer-fedora/gobbonet.spec`'s changelog.
