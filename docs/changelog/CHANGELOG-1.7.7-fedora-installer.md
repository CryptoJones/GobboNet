# 1.7.7 — Fedora installer: the `.rpm`, the Debian flow, the 1.7.6 packaging

The maintainer's direction, 2026-10-12: the installers act the way they did in
1.7.5, each with its own steps. Fedora is the last of the three, and comes from
the same place as Windows and Linux: the maintainer's own commits.

## Which installer it is

**`installer-fedora/`, the `.rpm`.** The maintainer added it in 1.7.3
(`c89461d`). Two later releases changed it, and neither changed the user's
flow:

- **1.7.4 repaired it.** `INVESTIGATION-fedora-1.7.4.md` found it could not be
  built and shipped without the setup wizard.
- **1.7.6 rebuilt its packaging.** `CHANGELOG-1.7.6-fedora-rpm.md` covers:
  - requirements generated from the engine;
  - the Fedora 45 and OpenSSL 4 fix;
  - `check-rpm.sh` and `test-in-fedora.sh`.

**The user goes through Debian's flow, on Fedora's terms:**

1. **Install:** `sudo dnf install ./gobbonet-1.7.7-1.go.<sha>.fedora.x86_64.rpm`.
   Use dnf, not `rpm -i`: from Fedora 45, rpm refuses unsigned packages, and dnf
   also fetches what the package needs. On Fedora 45 and later that includes
   `openssl3-libs`, which the bundled engine needs.
2. **Open GobboNet** from the applications menu, or run `gobbonet`. The same
   browser setup wizard as the `.deb` opens, with two Fedora lines:
   - it names `/usr/lib64/gobbonet`;
   - its firewall step gives the firewalld command instead of UFW's.

   The steps are in `CHANGELOG-1.7.7-linux-installer.md`.
3. **The chat opens.** Opening GobboNet again goes straight to it.

**Why 1.7.6's packaging stays.** "Act like 1.7.5, not 1.7.6" is about what
the user goes through, and on Fedora that did not change in 1.7.6. What 1.7.6
changed is what makes the engine start on Fedora 45:

- **What changed in Fedora 45:** it moved to OpenSSL 4.
- **What the 1.7.5-era package would do there:** install, then fail on the
  first model.

## What changed for 1.7.7

**`gobbonet uninstall` names dnf on Fedora.**

- **The bug:** it ended every non-Windows run with
  `sudo apt remove gobbonet`.
- **History:** 1.7.6 listed this under "Still outstanding", to be fixed with
  the next binary, and put the dnf command in `FEDORA.md` meanwhile.
- **The fix:** it now goes by where the program runs from, not by the
  distribution:
  - `/usr/lib64/gobbonet` (the `.rpm`): `sudo dnf remove gobbonet`;
  - `/usr/lib/gobbonet` (the `.deb`): `sudo apt remove gobbonet`;
  - Windows: Settings > Apps, as before;
  - anywhere else, a portable ZIP: delete the folder it was unpacked into.
    It used to say apt there too.
- **The test:** `TestRemoveHintFollowsTheInstall` in
  `cmd/gobbonet/uninstall_test.go`.

**Carried in from the Linux step,** because the rpm packages the same wizard:

- the GPU-memory question before an oversized download (#8a);
- the model step naming the folder actually chosen.

**Docs:**

- `FEDORA.md` names 1.7.7 and drops its note about the apt hint.
  Its test note now says what was tested for 1.7.7.
- `LINUX-START-HERE.md`'s Fedora line names 1.7.7.
- `gobbonet.spec` gains its `1.7.7-1` changelog entry, as release housekeeping
  planned.

**The test harness:**

- **The bug:** `test-in-fedora.sh --upgrade-from` installed the old package
  with plain `rpm -i`. That only worked for 1.7.3-3, which asked for
  `openssl-libs >= 3.0`. 1.7.6 asks for OpenSSL 3 by soname, so on Fedora 45
  and Rawhide the old package could not even go in.
- **The fix:** it now goes in with the same OpenSSL 3 stand-in the new package
  uses.

## Built and tested

`gobbonet-1.7.7-1.go.<sha>.fedora.x86_64.rpm`, 20 MB (101 MB installed):

- **How it was built:** `installer-fedora/build-rpm.sh` with the zip's
  `linux-amd64/gobbonet`.
- **The engine:** llama.cpp b10456, matched to the SHA-256 pin.
- **`check-rpm.sh`:** passes every check:
  - its requirements: the engine's are generated from its own files,
    `libssl.so.3` by soname included;
  - nothing offered to the system;
  - the launcher, wizard and catalogue byte for byte from their sources;
  - the payload against `payload.manifest`.

**`test-in-fedora.sh`,** on the official Fedora root filesystems of
2026-10-11 (fedora-cloud/docker-brew-fedora), each with its own rpm, dnf,
glibc and OpenSSL:

| Fedora | rpm | OpenSSL | Python | Result |
|---|---|---|---|---|
| 43 | 6.0.2 | 3.5.8 | 3.14 | 12/12 |
| 44 | 6.0.2 | 3.5.9 | 3.14 | 12/12; upgrade run 19/19 |
| 45 | 6.1.0 | 4.0.3 | 3.15 | 13/13 |
| Rawhide (46) | 6.1.0 | 4.0.3 | 3.15 | 20/20, with upgrade |

**Each fresh install:**

- installed with dnf, offline;
- `rpm -V` clean before and after the first run;
- the engine starts as an ordinary user;
- the repo's own first-run test passes as that user, local and LAN;
- GobboNet in local mode starts the engine on a model and answers a chat;
- removal leaves nothing behind and the user's data alone.

**On Fedora 45 and Rawhide also:** dnf refuses the package, naming
`libssl.so.3`, when nothing provides OpenSSL 3.

**The upgrade from 1.7.6:**

- dnf upgrades it;
- `rpm -V` is clean;
- every file under `/usr/lib64/gobbonet` belongs to the new package;
- the engine path in the existing config still runs;
- the chat page is served.

**The old package was rebuilt here** from the `v1.7.6` tag with that tag's own
scripts, since the shipped one is not in this repository.

**How the build machine got there,** since only GitHub is reachable from it:

- **rpm 4.19.1:** built from rpm-software-management/rpm, with Lua, popt and
  the libmagic/libelf/zstd headers from their GitHub sources.
- **A small `cpio`** for the one extraction `check-rpm.sh` does.

They are build tools in the sandbox only, and none of it is in the package.

**Not done here:**

- SELinux;
- a real desktop session and GNOME Software;
- GPU inference.

The same three as 1.7.6; they need Fedora hardware.
