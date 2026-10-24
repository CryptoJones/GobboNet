# 1.7.8 — The Linux installer: the .deb and the AppImage, checked against 1.7.7

> I don't want the installers to change anything that they don't have to since
> the setup has worked so well for people in 1.7.7.

## How they were built

**The `.deb`:** `installer-linux/build-deb.sh`, as for 1.7.7, with the ZIP's
own `linux-amd64/gobbonet` as `GOBBONET_BIN`. The server inside therefore
carries the ZIP's build stamp. The engine is llama.cpp b10456, Vulkan,
matched to the pin in `engine.sha256`. `dpkg-deb` writes zstd members
(`control.tar.zst`, `data.tar.zst`), as in 1.7.7's package.

**The AppImage:** `installer-linux/build-appimage.sh`, made from that `.deb`
(issue #53). 1.7.7 had no AppImage, so it was checked against the `.deb`
instead.

## What the 1.7.8 `.deb` does differently from 1.7.7's

The official `gobbonet_1.7.7go.a1a1ce8-1_amd64.deb` was downloaded from the
1.7.7 release page and taken apart beside the new one.

**The same:**

- every control field except `Version` and `Installed-Size`;
- `postinst` and `postrm`, byte for byte;
- all 88 entries, with the same types, modes, owners and link targets;
- 57 of the files, byte for byte. That covers the whole engine, the menu entry,
  the icon, `gobbonet-setup.py` and the copyright file.

**Changed** — six files, each by this release's changes to its source since
`v1.7.7`:

| File | Why |
|---|---|
| `usr/lib/gobbonet/gobbonet` | the 1.7.8 server |
| `usr/lib/gobbonet/gobbonet-launch` | says when python3 is missing, instead of "first-run setup did not finish" (#53) |
| `usr/lib/gobbonet/wizard.html` | Enter submits each step (#63); MoE models on 8 GB cards with enough RAM (#65) |
| `usr/lib/gobbonet/models.ini` | the MoE fields, `moe_min_vram` and `min_ram` (#65) |
| `usr/lib/gobbonet/llama-cpp/ENGINE.txt` | its "bundled by" line names the package version |
| `usr/share/doc/gobbonet/README.md` | the AppImage section (#53) |

The official package's text files are `v1.7.7`'s sources, byte for byte, so
this table is the whole difference.

## One doc fix: the `apt` command

**`LINUX-START-HERE.md`'s install command did not match the downloaded
file.** It said `sudo apt install ./gobbonet_1.7.7+go.*_amd64.deb`:

- that is the name `build-deb.sh` gives the package;
- GitHub drops the `+` from release asset names, so the file on the release
  page is `gobbonet_1.7.7go.a1a1ce8-1_amd64.deb`;
- the pattern matched nothing, the shell passed it to apt as it was, and apt
  answered `E: Unsupported file ./gobbonet_1.7.7+go.*_amd64.deb given on
  commandline`. That was reproduced here with the official file.

It now says `./gobbonet_1.7.8*_amd64.deb`, which matches both names. The
AppImage and `.rpm` patterns already matched both.

The page is also titled 1.7.8 and names the 1.7.8 `.rpm`. Its first-launch
section gains one line on what is new in 1.7.8.

## Checked here

On Ubuntu 24.04 (container), in headless Chromium, with the real bundled
engine. GobboNet ran as an ordinary user each time, made for the test, as the
package says to run it.

**A fresh install:**

1. `apt install` of the 1.7.8 `.deb`: every file in place, `gobbonet` on PATH
   is the launcher, and the engine runs from `/usr/lib/gobbonet`.
2. `gobbonet` opens the wizard. It was walked with **Enter** in every field,
   into a folder whose name has a space. The page showed the MoE line and had
   no errors.
3. The chat opened at `:9066`, and a message streamed a reply from the
   engine.
4. `config.toml` is the user's own file, mode 0600.
5. `apt remove`: program files, command and menu entry gone. The note about
   user data printed once. Config and chats stayed.
6. `apt purge` printed nothing more and left them too.

**An upgrade from 1.7.7:**

1. The official 1.7.7 `.deb` was installed with `apt`. Its wizard was walked
   by mouse, and a chat was had.
2. `apt install` of the 1.7.8 `.deb` reported *Unpacking gobbonet (1.7.8…)
   over (1.7.7…)*, with no removal note. The user's `config.toml` and data
   folder were untouched.
3. `gobbonet` went straight to the chat, with no wizard, and said "setup:
   already complete".
   - The page said 1.7.8, and the 1.7.7 conversation was there.
   - A new message got a reply.
   - CONFIG > Config File found the user's `config.toml`.
4. `gobbonet uninstall --yes --remove-models`, then `apt remove` and
   `apt purge`.

**The AppImage, as a user with FUSE as on a desktop:**

1. Downloaded, `chmod +x`, run: the wizard, then a chat. It ran from a FUSE
   mount, and setup recorded the engine inside that mount.
2. Closing GobboNet unmounted the image.
3. Run again: straight to the chat. The setting naming the engine moved to the
   new mount, and the engine there answered.
4. The `.deb` was installed beside it, and `gobbonet` ran from the package:
   - straight to the chat, with the AppImage's conversations;
   - the engine setting moved to `/usr/lib/gobbonet`.
5. The `.deb` was removed again, and the AppImage run again: still straight to
   the chat, with the package's conversations.
6. `./GobboNet-*.AppImage uninstall` cleared the config and said to delete the
   `.AppImage` file. No mounts were left.

**Test suites:**

- `tests/test-appimage.py`: all 12 checks, including the full build from this
  `.deb`.
- `tests/test-linux-onboarding.py`: against the installed package, as an
  ordinary user, for a local and a LAN setup.

**Allowances for the test model.** The model is 500 KB of random weights, so
three settings were changed, and a real model needs none of them:

- `kv_cache_type = "f16"`: its 16-wide attention heads do not fit q8_0 cache
  blocks;
- `ctx_size = 2048`: it never ends a reply;
- Stop is pressed once the reply is streaming.

The browser opener was a stand-in that records the address it is given. The
test then opened that address in Chromium.

## Found, and left as they are

All three behave the same in 1.7.7.

- **`gobbonet uninstall` leaves some files behind:**
  - what is left: `~/.local/share/gobbonet/launch.log`, the launcher's log,
    and the `embeddings/` folder the optional Nomic download goes into;
  - because of them, the data folder stays too;
  - all while the command says "GobboNet's data is gone".
- **"GPU acceleration confirmed" with no GPU.** On a machine with no GPU at
  all, start-up says this, because llama.cpp prints "offloaded 3/3 layers to
  GPU" anyway.
- **"I already have one":** the models folder the step names does not exist
  until something is downloaded, so the user has to make it.

## Not checked

- **A real desktop session:** double-clicking the `.deb`, the menu entry, the
  browser opening by itself, and AppImage menu integration.
- **Hardware and downloads:** a GPU or a real Vulkan driver, a model download,
  and Debian itself rather than Ubuntu.
- **The AppImage on other distributions.**

## Tests

**`tests/test-linux-start-here.py`** (new) holds `LINUX-START-HERE.md`'s
`apt`, `dnf` and AppImage commands to both names each file can have:

- as built, `gobbonet_<version>+go.<sha>-1_amd64.deb`;
- as downloaded, the same without the `+`.

It also checks:

- the page is titled for the release in `VERSION`;
- the build scripts still name their files that way.

It fails on the 1.7.7 page.

## Left for the Fedora step and the release

These are from `CHANGELOG-1.7.8-version-bump.md`'s list.
`LINUX-START-HERE.md` is done here. Still to do:

- `installer-fedora/FEDORA.md`;
- the spec's changelog entry;
- `START-HERE.md`;
- the root `SHA256SUMS.txt`.
