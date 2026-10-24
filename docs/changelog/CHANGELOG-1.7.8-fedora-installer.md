# 1.7.8 — The Fedora installer: the .rpm, checked against 1.7.7

> I don't want the installers to change anything that they don't have to since
> the setup has worked so well for people in 1.7.7.

## How it was built

**`installer-fedora/build-rpm.sh`, as for 1.7.7,** with the ZIP's own
`linux-amd64/gobbonet`, so the package's `Release` names the ZIP's build. The
engine is llama.cpp b10456, matched to the pin in `engine.sha256`.
`check-rpm.sh` passes.

**The same rpm as 1.7.7.** The official 1.7.7 package's header says it was
built by RPM 4.19.1 for platform `x86_64-vendor-linux`, which is an rpm built
from source. So rpm 4.19.1 was built here from
rpm-software-management/rpm, the same way 1.7.7's changelog describes:

- Lua and popt built from their GitHub sources;
- the headers for the system's libmagic, libelf and zstd taken from theirs;
- for `check-rpm.sh`'s one extraction, libarchive's `bsdcpio` standing in
  for `cpio`.

These are build tools on the build machine only; none of it is in the package.

**The toolchain reproduces 1.7.7.** Two inputs went into a rebuild:

- the `v1.7.7` scripts;
- 1.7.7's own binary, lifted out of the official package.

The result matches `gobbonet-1.7.7-1.go.a1a1ce8.fedora.x86_64.rpm` in every
header field except the build time:

- all 72 requirements, 5 provides and 4 recommends;
- the scriptlets;
- the changelog;
- all 79 files' contents, modes, owners, links and flags.

## What the 1.7.8 `.rpm` does differently from 1.7.7's

The official 1.7.7 package was downloaded from the release page.

**The same:**

- the summary, description, licence, URL, platform, payload format and
  compression;
- the scriptlets: `%post`, `%preun` and `%postun`;
- all 72 requirements (the engine's generated sonames, OpenSSL 3 among them),
  the 4 recommends, and `bundled(llama.cpp) = b10456`;
- all 79 files, with the same types, modes, owners, links and flags;
- 58 of the files, byte for byte. That covers the whole engine, the menu
  entry, the icon, `gobbonet-setup.py` and both licences.

**Changed** — six files. Five are the `.deb`'s changes
(`CHANGELOG-1.7.8-linux-installer.md`). The sixth is the package's own doc.

| File | Why |
|---|---|
| `gobbonet` | the 1.7.8 server |
| `gobbonet-launch` | says when python3 is missing (#53) |
| `wizard.html` | Enter submits each step (#63); MoE models on 8 GB cards (#65). The spec's two Fedora rewrites, `/usr/lib64` and firewalld, still apply. |
| `models.ini` | `moe_min_vram` and `min_ram` (#65) |
| `llama-cpp/ENGINE.txt` | its "bundled by" line names the package version |
| `/usr/share/doc/gobbonet/FEDORA.md` | names the 1.7.8 package, and says the upgrade tested was from 1.7.7 |

**Also new:** a `1.7.8-1` changelog entry in the spec. The entries before it
are 1.7.7's.

## Docs

- **`installer-fedora/FEDORA.md`:** the `dnf install` line names 1.7.8.
- **`START-HERE.md`** was still at 1.7.6; the 1.7.7 packaging step never
  reached it. It now covers:
  - the 1.7.8 release and its five downloads;
  - the install line for each;
  - what is in 1.7.8.

  The build commit appears as a placeholder, `<commit>`. A file inside the
  commit cannot name the commit.

## Checked here

### `test-in-fedora.sh`

It runs on the official Fedora root filesystems of 2026-10-18
(fedora-cloud/docker-brew-fedora). Each has its own rpm, dnf, glibc and
OpenSSL. Each run also tests the upgrade from the official 1.7.7 package, and
has a model to load:

| Fedora | rpm | OpenSSL | Python | Result |
|---|---|---|---|---|
| 43 | 6.0.2 | 3.5.9 | 3.14 | 20/20 |
| 44 | 6.0.2 | 3.5.9 | 3.14 | 20/20 |
| 45 | 6.1.0 | 4.0.3 | 3.15 | 21/21 |
| Rawhide (46) | 6.1.0 | 4.0.3 | 3.15 | 21/21 |

**Each fresh install:**

- an offline `dnf install`;
- `rpm -V` clean before and after the first run;
- the engine starts as an ordinary user;
- the repo's first-run test passes as that user, local and LAN;
- GobboNet in local mode starts the engine on a model and answers a chat;
- a root-run server's `web/` is cleaned up;
- removal leaves nothing, and the user's data alone.

**On Fedora 45 and Rawhide also:** dnf refuses the package, naming
`libssl.so.3`, when nothing provides OpenSSL 3.

**The upgrade from 1.7.7:**

- dnf upgrades it, and `rpm -V` is clean;
- every file under `/usr/lib64/gobbonet` belongs to the new package;
- the engine path in the existing config still runs;
- the chat page is served.

### As a Fedora user, in a browser

On Fedora 44 and 45, inside the same roots: the user runs GobboNet, and
headless Chromium on the host plays the browser. Both releases passed all 49
checks.

**An upgrade from 1.7.7:**

1. The official 1.7.7 `.rpm` went in with `dnf`. Its wizard was walked by
   mouse:
   - it names `/usr/lib64/gobbonet`;
   - it gives the firewalld command and no UFW one;
   - the page had no errors.
2. A chat was had through the bundled engine.
3. `dnf` upgraded the package, with no removal note. `rpm -V` was clean, and
   the user's `config.toml` and data folder were untouched.
4. `gobbonet` went straight to the chat and said "setup: already complete".
   - The page said 1.7.8, and the 1.7.7 conversation was there.
   - A new message got a reply.
   - CONFIG > Config File found the user's `config.toml`.
5. `gobbonet uninstall` cleared the user's data, and its last line names
   `sudo dnf remove gobbonet`.
6. `dnf remove` printed its note once.

**A fresh install of 1.7.8:**

1. The wizard was walked with Enter in every field, into a folder whose name
   has a space. It showed the MoE line.
2. A chat was had.
3. `rpm -V` was still clean.
4. `dnf remove` left the user's config and chats.

**Allowances:**

- **Stand-ins for what every Fedora desktop has.** The minimal roots have no
  python3, xdg-utils or hicolor-icon-theme, so a test-only package provides
  them, with a standalone CPython of the release's Python version.
- **OpenSSL 3 on 45 and later.** Fedora 44's OpenSSL 3 libraries stand in for
  `openssl3-libs`, which dnf would fetch.
- **The test models.** For the harness, a 500 KB random-weight model with
  32-wide heads, because GobboNet's q8_0 cache needs them. The browser runs
  use the same allowances as the Linux round.

**One note for whoever runs the harness next:** run one at a time. The
engine's port, 11437, is fixed. Two runs at once share it, and local mode then
fails with "model is not ready". Run alone, every release passed.

## Not checked

The same as 1.7.7; they need Fedora hardware:

- SELinux, which the chroot has none of;
- a real desktop session and GNOME Software;
- GPU inference.

## Tests

`tests/test-version-strings.py` gains `test_fedora_package_docs_follow_version`.
It checks two things:

- `FEDORA.md`'s `dnf install` line names the release in `VERSION`;
- the spec's newest changelog entry is for that release.

Both were left at 1.7.7 until this step. The check fails on the 1.7.7 files.

## Left for the maintainer

- **The root `SHA256SUMS.txt`.** It lists the release binaries' hashes, and
  those are stamped with the commit they are built from. So it can only be
  written for the build actually released, after that commit.
- **The real `v1.7.8` tag.** It belongs on whichever commit is released, as
  `CHANGELOG-1.7.8-version-bump.md` says.
