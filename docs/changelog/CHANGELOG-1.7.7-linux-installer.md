# 1.7.7 — Linux installer: the 1.7.5 `.deb` and browser setup

The maintainer's direction, 2026-10-12: the installers act the way they did in
1.7.5, each with its own steps. Linux follows Windows, and comes from the same
place: the maintainer's own commits.

## Which installer it is

**`installer-linux/`, the `.deb`.** It was designed in the maintainer's 1.7.3
commit (`c89461d`) and shipped in 1.7.5 (`28fe0b2`), and there is only one:

- the menu entry and the `gobbonet` command run `gobbonet-launch`;
- on the first run, that opens the **browser setup wizard**, `gobbonet-setup.py`
  with `wizard.html`, and then the chat.

`internal/setup`'s own page is the minimal wizard underneath, not the one
users see. 1.7.5's `LINUX-START-HERE.md` says so too.

Unlike Windows, nothing here was ever swapped for another design. From 1.7.5 to
1.7.6 the Linux installer changed only in docs and two files' executable bits.

## What the user goes through

1. **Install:** `sudo apt install ./gobbonet_1.7.7+go.<sha>-1_amd64.deb`, or
   open it with the desktop's package installer. The Vulkan llama.cpp engine
   is bundled, so nothing is downloaded at install.
2. **Open GobboNet** from the applications menu, or run `gobbonet`. The setup
   wizard opens in the browser:
   1. Welcome
   2. **Install location:** the folder for models and chats
   3. **Set your access password**
   4. **Choose the web port** (9066)
   5. **Firewall and network access:** *Launch on this machine*, or *Set up LAN
      and launch*. Linux has no Defender page, so this explains the firewall
      instead.
   6. **Where should models run?** *On this machine* (the bundled engine), or
      *On a server I already have*. This is the step only Linux has.
   7. **Pick a model**, then the download (or *I already have one*)
   8. **Optional additions:** Nomic embeddings, or skip
   9. **Ready to launch:** *Start at login*, off by default. Then **LAUNCH!**
3. The chat opens in the browser once the server is listening. Opening
   GobboNet again goes straight to the chat.

## What changed since 1.7.5

**One thing, and it was asked for (roadmap #8a).** When the model picked needs
more GPU memory than the card has, the wizard asks before downloading:

- **How it reads the card:** it asks the bundled engine.
- **When it asks nothing:** no card can be read, the GPU is only an integrated
  one, or the check fails.

Nothing else in the launcher, the wizard or the package changed.

**Checked against 1.7.7:**

- **The catalogue:** the package installs `models.ini`, now carrying the type
  tags, where *Add a Model* finds it first (`/usr/lib/gobbonet/models.ini`).
- **The phone-address box (#7):** it saves to the user's own config.
- **In-app downloads (#8b):** they go to the model folder the wizard chose.
- **The ZIP's `linux-amd64/` copy** of the launcher and wizard is identical to
  the package's.

**Docs:** `LINUX-START-HERE.md` names 1.7.7 and mentions the GPU-memory
question. Its Fedora line moves with the Fedora package.

## Built and run here

`gobbonet_1.7.7+go.<sha>-1_amd64.deb`, about 19 MB (101 MB installed).
`installer-linux/build-deb.sh` built it from the zip's `linux-amd64/gobbonet`,
so the server inside carries the zip's build stamp.

- **The engine:**
  - llama.cpp b10456, Vulkan, matched to the SHA-256 pin in `engine.sha256`;
  - payload checked against `payload.manifest`;
  - libc6 floor 2.34, read from the engine.
- **The engine runs from the package layout:** `llama-server --version`
  reports build 10456 through its `$ORIGIN` runpath.
- **`tests/test-linux-onboarding.py`, against the extracted package:** passes
  for both a local and a LAN setup. It covers:
  - the first run and a location with spaces;
  - the password, port rejection and a custom port;
  - the missing-model guard and the switch to a remote server;
  - the browser opening, and a second launch going straight to the chat.
- **Every wizard step in a browser:** walked from Welcome to *Ready to launch*
  in the packaged wizard, with no page errors.

**Not done here:**

- a real `apt install` on a desktop;
- a model download and chat;
- a GPU;
- a non-root user, since this sandbox runs as root.

## Fixed after review: the model step's folder

**"Pick a model" could name the wrong models folder.** The maintainer approved
the fix on 2026-10-13.

- **The bug:** the line "About N GB free on this disk. Existing GGUF models
  belong in: …" was written once, when the wizard loaded. Choosing a different
  folder at *Install location* restarts setup on that folder, but the line
  kept naming the old folder, and the free space of the old disk.
- **Who it caught:** only someone copying their own `.gguf` in by hand. They
  would put it where *LAUNCH!* then said no model was. Downloads, and the check
  at *LAUNCH!*, always used the right folder.
- **Since when:** the same in 1.7.5.
- **The fix:** one function now writes the line. It runs at load, and again
  once a new folder is saved, from setup's own state for it.
- **Checked in a browser:**
  - after choosing a new folder, the model step names `<chosen>/models` and
    that disk's free space;
  - every step still walks through with no page errors.
- **Fedora gets it too:** `installer-fedora/build-rpm.sh` packages this same
  `wizard.html`.

## Guarded

**`tests/test-linux-installer.py`** (new, 8 tests):

- **The launcher:** the menu entry and the `gobbonet` command run
  `gobbonet-launch`, and its first run is the browser wizard.
- **The package:** it carries the wizard, `models.ini` and the Vulkan engine.
- **The ZIP's runtime:** its copy is the same wizard.
- **The wizard:** its steps and their headings, and the order between them,
  as listed above.
- **The model step's note:** it follows the chosen folder. This check fails on
  1.7.5's wizard.
- **The build guide:** `docs/BUILDING.md` builds the `.deb` from
  `installer-linux/`.
