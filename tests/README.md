# Tests and preview pages

Developer tooling. Nothing here ships, and nothing here is needed to run
GobboNet.

## Frontend suites (`*.mjs`)

Plain Node, no framework and no install step. They read the real files out of
`js/` and evaluate them, so they test what actually ships rather than a copy.

```sh
node tests/test-markdown-render.mjs     # one suite
for f in tests/*.mjs; do node "$f" || echo "FAILED: $f"; done   # all of them
```

`test-engine-args.py` pins the llama-server command line across the two launch
paths — `launch.bat`'s `:start_server` block and `Supervisor.BuildArgs`, the
latter being what the `.deb` and the Windows installer both use. Every flag
present on one side and not the other has to be listed as a known, reasoned
difference or the test fails. It also holds the `-lv` / offload-detection
pairing from #33: if GPU-offload confirmation is ever added to the Go path,
`-lv` must arrive with it, because llama.cpp files those log lines above the
default verbosity threshold.

```sh
python3 tests/test-engine-args.py
```

`test-prompt-safety.py` and `test-image-url-gate.mjs` cover the three fixes from
PR #30 (John McCardle). The first pins the confirmation-prompt hygiene in
`launch.bat` and the sanitizer routing in `identify-model.ps1` — neither can run
on CI, since one needs an interactive Windows console and the other a real GGUF.
The PR proposed parsing the PowerShell with its own AST under a container; this
does the same job statically so it runs anywhere the rest of the suite does. The
second drives the real `safeImageUrl` and asserts nothing it returns can carry a
character that ends an HTML attribute or a CSS `url()`. Background is in
[`docs/changelog/CHANGELOG-1.7.3-prompt-and-sanitizer-fixes.md`](../docs/changelog/CHANGELOG-1.7.3-prompt-and-sanitizer-fixes.md).

`test-remote-models.mjs` covers the model dropdown in remote mode — the shape
behind #47, #48 and #27. It drives the real `loadModelsList` /
`onHeaderModelChange` in a fake DOM and includes source guards that no request
path has drifted back to a hardcoded model name. Background is in
[`docs/changelog/CHANGELOG-1.7.3-remote-model-list.md`](../docs/changelog/CHANGELOG-1.7.3-remote-model-list.md).

`test-code-copy.mjs` covers the Copy button on a code block, driving the real
`parseMarkdown` and `copyCodeBlock` over a node tree built from the rendered
HTML. The DOM is a shim at the top of the file rather than a dependency, since
nothing in `tests/` has an install step. It pairs with section B3 of
`test-markdown-render.mjs`: that one covers what renders, this one covers what
reaches the clipboard. Background is in
[`docs/changelog/CHANGELOG-1.7.4-code-copy-linebreaks.md`](../docs/changelog/CHANGELOG-1.7.4-code-copy-linebreaks.md).

`test-standdown-panel.mjs` covers the CONFIG control for idle stand-down. The
mechanism itself is server-side and tested in Go
(`internal/supervisor/standdown_test.go` for the decision rule,
`internal/server/standdown_endpoint_test.go` for the endpoint); this covers the
one thing only the browser can get wrong, which is showing a switch that cannot
work. Background is in
[`docs/changelog/CHANGELOG-1.7.4-idle-standdown.md`](../docs/changelog/CHANGELOG-1.7.4-idle-standdown.md).

`test-lore-editing.mjs` covers hand-editing the lore summary. Section A is the
one that matters: a compression pass `await`s a model for seconds and then
writes the summary back, so an edit made during that window would be silently
replaced by something derived from the version it replaced. It pins that the
edit survives *and* the pass's beat is kept where the two can be separated.
Background is in
[`docs/changelog/CHANGELOG-1.7.4-lore-editing.md`](../docs/changelog/CHANGELOG-1.7.4-lore-editing.md).

`test-version-stamp.mjs` covers the build identity in the ABOUT panel. Section
A is the load-bearing one: it asserts the frontend's `GOBBONET_UI_VERSION`
equals the `VERSION` file, so bumping one without the other fails the suite.
That guard exists because the string it replaced read
`1.6.0-no-encoded-payload` for the whole of 1.7.x — the one thing in the app
claiming to identify the build was wrong by two minor releases and nothing
noticed. The rest covers the server-vs-page mismatch warning, which is how a
cached frontend gets told apart from a real regression. Background is in
[`docs/changelog/CHANGELOG-1.7.4-about-version.md`](../docs/changelog/CHANGELOG-1.7.4-about-version.md).

`test-server-presets.mjs` covers the `[ui]` table in `gobbonet.toml`, which
seeds browser settings from the config file. The rule it exists to protect is
seed-not-lock: a device that has its own settings keeps them and is only
offered the presets. Section D asserts the key list is generated from
`DEFAULT_SETTINGS` by adding a setting at runtime and checking it is
immediately presettable — the property that keeps Go and the browser from
needing two lists. The Go halves are `internal/server/ui_defaults_test.go` and
`internal/config/ui_table_test.go`. Background is in
[`docs/changelog/CHANGELOG-1.7.4-ui-presets.md`](../docs/changelog/CHANGELOG-1.7.4-ui-presets.md).

`test-single-export.mjs` covers exporting one thread or one character, driving
the real export and import with the download captured. The fixture is built so
the easy implementation fails: the thread's cast is not the whole roster, and
one character appears only on a message rather than on the thread, so an export
that read `thread.cardId` alone would miss it. Section C is the full round trip
through the real `importData`. Background is in
[`docs/changelog/CHANGELOG-1.7.4-single-export.md`](../docs/changelog/CHANGELOG-1.7.4-single-export.md).

`test-scroll-modes.mjs` covers how the viewport behaves while a reply streams
— follow-but-let-go, always follow, or never — driving the real functions
against a fake scroll container that clamps `scrollTop` the way a real one
does. Section D is the one that earns its keep: it pins that the end-of-stream
rebuild puts the reading position back instead of leaving the user at the top
of the thread. Background is in
[`docs/changelog/CHANGELOG-1.7.4-scroll-modes.md`](../docs/changelog/CHANGELOG-1.7.4-scroll-modes.md).

`test-sync-targets.mjs` covers which server-side backup a device syncs with —
the shared one, a profile of its own, or none — driving the real target logic
against a fake localStorage and a fake fetch, so the URLs asserted on are the
ones a browser would request. Section A exists for one reason: an install with
no target stored must resolve to *shared*, or every existing user boots into an
empty chat with their history still on the server. The Go half is
`internal/server/state_profiles_test.go`. Background is in
[`docs/changelog/CHANGELOG-1.7.4-device-sync.md`](../docs/changelog/CHANGELOG-1.7.4-device-sync.md).

`test-card-share-export.mjs` covers the FOR SHARING character-card export — a
JPG with a ZIP archive appended — driving the real ZIP writer and reader over a
small JPEG embedded in the test file. The assertion that earns its keep is that
a *naively* concatenated archive fails to read: the offsets have to be written
biased by the length of the image in front of them, or GobboNet cannot open its
own export. Background is in
[`docs/changelog/CHANGELOG-1.7.4-share-export.md`](../docs/changelog/CHANGELOG-1.7.4-share-export.md).

`test-char-modal.mjs` covers dismissal of the CAST modal — the sticky setting,
the unsaved-changes guard, and the notice shown when a dismissal is refused —
by running the real handlers from `js/22-scheduler.js` in a fake DOM. Section A
is the original v1.7 device/pointer/target matrix and is deliberately frozen:
with the editor clean and sticky on, every answer must still be what it was.
Background is in
[`docs/changelog/CHANGELOG-1.7.4-sticky-cards.md`](../docs/changelog/CHANGELOG-1.7.4-sticky-cards.md).

Two of them cover the same area from opposite ends: `test-cast-identity.mjs`
checks that a past message keeps the character that wrote it, and
`test-cast-mismatch.mjs` checks the notice shown when the *next* reply would
come from someone else.

Each resolves the repo root from its own location, so the working directory does
not matter.

`test-conversation-order.mjs` covers where a conversation sits in the sidebar:
activity order, a drag holding until the chat is next used (the 1.7.5 rule),
respacing, placements from builds that recorded no activity mark, carrying a
saved `threadOrder` list into the keys through the real `applyLoadedState` and
`loadState`, and the two sync guards -- a placement never reads as a message
conflict, and reconcile keeps a placement's two fields together. Background is
in [`docs/changelog/CHANGELOG-feature-restore.md`](../docs/changelog/CHANGELOG-feature-restore.md).

`test-device-sync.mjs` covers the DEVICE SYNC choices against the real sync
layer and an in-memory server with the protocol's rules: seeding a target
records the versions it wrote (so a new profile, "keep this device", or sync
back on ends synced, not in conflict), a write from another device in between
is not adopted, and a chat missing the loader's defaults does not read as a
conflict. It also holds `THREAD_LOAD_DEFAULTS` equal to the loader's block.
Background is in [`docs/changelog/CHANGELOG-feature-restore.md`](../docs/changelog/CHANGELOG-feature-restore.md).

`test-boot-resume.mjs` drives the real `24-boot.js` against the real sync layer:
pending replies resume after the startup state check whether or not it
succeeded, and the one-time notice flag reaches storage through
`saveState({ localOnly })` without scheduling a push.

`test-landing-scroll.mjs` covers the home screen keeping its scroll position
through the five-second re-render in `24-boot.js`. It drives the real
`renderMessages()` against a container whose rebuild resets every scroll box,
and section A proves that fake can fail before B relies on it. D holds the
early `return` the reported fix dropped. Background is in
[`docs/changelog/CHANGELOG-1.7.7-landing-scroll.md`](../docs/changelog/CHANGELOG-1.7.7-landing-scroll.md).

`test-about-links.mjs` covers the MODS and DISCORD buttons in ABOUT: the exact
addresses, `noopener` and `noreferrer`, no address in any script that could run
on its own, a placeholder that does nothing, and a box copied from YOUR DATA
rather than designed. Background is in
[`docs/changelog/CHANGELOG-1.7.7-about-links.md`](../docs/changelog/CHANGELOG-1.7.7-about-links.md).

`test-user-limits.mjs` covers four limits that were constants and are now the
user's: the per-character reply cap (no 8,192 ceiling), auto-continue's pause
and chain cap, the lore compression timeout, and the runtime limit the page now
sends with every job in place of the server's fixed 30 minutes. Each reader is
held to two rules: the old value is the default, and a missing or nonsense
stored value falls back to it, never to 0. The server half of the runtime limit
is in `internal/jobs/runtime_limit_test.go`. Background is in the
`docs/changelog/CHANGELOG-1.7.7-*` entries for roadmap items #6a, #6b, #6c and
#4.

`test-thinking-switch.mjs` covers the per-character thinking switch. A character
with it off sends exactly `chat_template_kwargs: { enable_thinking: false }`, as
a JSON boolean, since llama.cpp rejects a string. A character with it on sends
nothing extra, and its request is byte-identical to one without the switch. A
card with no field, as from an older install or an import, thinks as it always
did. Background is in
[`docs/changelog/CHANGELOG-1.7.7-thinking-switch.md`](../docs/changelog/CHANGELOG-1.7.7-thinking-switch.md).

`test-context-budget.mjs` covers the four per-character numbers that decide when
older messages are compressed into lore: input budget, reply room, landing point
and verbatim reserve. It runs the real `resolveContextBudget`:

- Left alone, they are the old constants.
- The card beats the setting, which beats the constant; blank inherits.
- The combinations that would stall compression are adjusted and explained.
- The budgeter and the context meter both read the one resolver.

Background is in
[`docs/changelog/CHANGELOG-1.7.7-context-budget.md`](../docs/changelog/CHANGELOG-1.7.7-context-budget.md).

`test-lan-panel.mjs` covers CONFIG's *Phone & Other Devices* box, run on the
real panel code:

- **When it can't work** (opened from a file, the PowerShell server, an older
  program file), it says why and sends nothing.
- **It lists** the addresses phones can use, with COPY.
- **It fills the choice** from what the config file says.
- **It shows the new address before saving**, warning when the page's own
  address is about to stop working.
- **Saving sends exactly `{host, port}`**, and reports anything that will stop
  it applying.

The server half is in `internal/server/lan_test.go`. Background is in
[`docs/changelog/CHANGELOG-1.7.7-phone-address.md`](../docs/changelog/CHANGELOG-1.7.7-phone-address.md).

`test-wizard-vram.mjs` covers the setup wizard's oversized-model warning, in
each wizard page (the packaged one, its `linux-amd64/` copy, and the one built
into the program). It asks only when the engine reports a GPU smaller than the
model wants:

- It asks nothing when the figure is unknown or the request fails.
- Declining sends nothing.
- Accepting downloads.

The GPU reading is in `internal/engine/devices_test.go` and
`internal/setup/gpu_test.go`. Background is in
[`docs/changelog/CHANGELOG-1.7.7-wizard-vram-warning.md`](../docs/changelog/CHANGELOG-1.7.7-wizard-vram-warning.md).

`test-password-autofill.mjs` keeps Chrome's password manager from treating
the chat page as a login form. When it did, the saved username was filled into
the conversation search on every load, and ran a search. The test pins two
rules:

- The sign-in box is the page's only `<form>`.
- Every other password field is `autocomplete="new-password"`.

Background, including how it was reproduced in a real Chromium, is in
[`docs/changelog/CHANGELOG-1.7.7-search-autofill.md`](../docs/changelog/CHANGELOG-1.7.7-search-autofill.md).

`test-catalog-filters.mjs` covers the Add a Model window's filters, on the real
list code:

- **Type** (coding, roleplay, jack of all trades) comes from each model's
  buckets.
- **Size** splits at 8 and 16 GB of GPU memory.
- **Combinations** of the two narrow the list together.
- **ALL MODELS / ANY SIZE** show everything.
- **The choice** lasts while the page is open.
- **The buttons** are built from existing pieces.

`test-catalog-buckets.py` follows the tags from `launch.bat`'s `MODEL_TAGS`
through `installer/gen-catalog.py` into `models.ini`. The Go side, including
the live list borrowing buckets from the bundled one, is in
`internal/catalog/buckets_test.go`. Background is in
[`docs/changelog/CHANGELOG-1.7.7-catalog-filters.md`](../docs/changelog/CHANGELOG-1.7.7-catalog-filters.md).

```sh
python3 tests/test-catalog-buckets.py
```

`test-download-testing.mjs` covers the *Add a Model* button, live again for
testing but not listed as working. It checks four things:

- The button works, with nothing to find first.
- The box says it is not confirmed working, until marked confirmed.
- The hold still works as one flag.
- A failed download names the step it stopped at.

The downloader's own diagnostics are in
`internal/modelfetch/diagnose_test.go`. Background is in
[`docs/INVESTIGATION-in-app-downloads.md`](../docs/INVESTIGATION-in-app-downloads.md).

## Launcher invariants

`test-launch-gpu-detect.py` checks the arrangement in `launch.bat` that lets it
confirm GPU offload: that `-lv` is passed, that the verbosity is high enough for
llama.cpp to emit the lines the check reads, and that the check accepts more than
one spelling of them. Background is in
[`docs/changelog/CHANGELOG-1.7-llamacpp-log-verbosity.md`](../docs/changelog/CHANGELOG-1.7-llamacpp-log-verbosity.md).

```sh
python3 tests/test-launch-gpu-detect.py
```

`test-setup-lan.py` does the same job for `setup-lan.bat` and
`teardown-lan.bat`, neither of which can be exercised on CI — they need
Administrator on Windows and a real firewall to talk to. It pins how the web
port is resolved, that no allow rule is ever wider than `LocalSubnet`, that
every rule added has a matching removal in the teardown, and that no `%VAR%`
read sits inside a parenthesised block. Background is in
[`docs/changelog/CHANGELOG-1.7.3-lan-address-and-firewall.md`](../docs/changelog/CHANGELOG-1.7.3-lan-address-and-firewall.md).

```sh
python3 tests/test-setup-lan.py
```

`test-launch-gpu-layers.py` pins where `launch.bat` turns automatic GPU
placement (`-1`) into 99 layers: on the legacy path only, before llama-server
and the hot-swap variables see it, never on the managed hand-off to
`gobbonet.exe`, which does its own capability-checked fitting.

## Build scripts

`test-build-scripts.py` holds the rule that no build script runs another as a
bare path or behind `[ -x ]` — always `bash x.sh` or `python3 x.py`. The
executable bit does not survive a commit made from Windows, and before this every
build script had lost it: `./build-release.sh` was refused, and two steps skipped
themselves silently. It tests its own pattern against the exact lines it
replaced, so it cannot pass by matching nothing. Background is in
[`docs/changelog/CHANGELOG-1.7.7-build-scripts.md`](../docs/changelog/CHANGELOG-1.7.7-build-scripts.md).

```sh
python3 tests/test-build-scripts.py
```

`test-linux-onboarding.py` runs the real first-run flow of an installed Linux
build. It needs a built install, not a checkout: point `GOBBONET_TEST_PREFIX` at
one — a `gobbonet` binary beside the contents of `linux-amd64/`, with an engine
under `llama-cpp/`. A stub engine that answers `--version` is enough, since the
test switches to a remote backend before anything would load a model. To run the
same first run through an extracted ZIP's front door, also set
`GOBBONET_TEST_ENTRY` to that ZIP's top-level `./gobbonet`.

`test-version-strings.py` holds the release number's literal copies to
`VERSION`: `installer-win/gobbonet.nsi`'s `APPVER`, `OutFile` and
`VIProductVersion`, and `installer/gobbonet.nsi`'s use of its define. The page
stamp has its own test, `test-version-stamp.mjs`.

`test-windows-installer.py` holds the Windows installer to being
`installer-win/`, with the flow it has had since 1.7.0:

- **The pages:** Welcome, Destination, Password, Port, Install, the Defender
  notice, Finish.
- **The look:** dark throughout.
- **No model step:** no backend choice and no model download during setup.
- **The console:** *Launch GobboNet* and the shortcuts open `launch.bat`,
  which detects the hardware, offers the models, then hands over with `--open`.
- **The payload:**
  - every helper `launch.bat` runs is installed;
  - `models.ini` is installed and removed;
  - everything installed at the root is removed on uninstall.
- **The build guide:** it holds `docs/BUILDING.md` to naming `installer-win/`.
  `installer/` was built by mistake for 1.7.4 and for the first 1.7.7 test
  build.

It does not compile the installer. Background is in
[`docs/changelog/CHANGELOG-1.7.7-windows-installer.md`](../docs/changelog/CHANGELOG-1.7.7-windows-installer.md).

```sh
python3 tests/test-windows-installer.py
```

`test-linux-installer.py` does the same for Linux. It holds `installer-linux/`
— the `.deb`, whose menu entry runs `gobbonet-launch` and its browser setup
wizard — to the 1.7.5 flow `LINUX-START-HERE.md` names:

- **The wizard:** its steps, their headings and the order between them.
- **The model step's note:** it names the folder chosen at *Install location*,
  not the one the wizard loaded with.
- **The package:** it carries the wizard, `models.ini` and the Vulkan engine.
- **The ZIP's runtime:** the copy in `linux-amd64/` is the same wizard.
- **The Fedora package:** `build-rpm.sh` packages that same launcher and wizard.
- **The build guide:** it builds the `.deb` from `installer-linux/`.

`test-linux-onboarding.py` runs that flow for real; this checks the source.
Background is in
[`docs/changelog/CHANGELOG-1.7.7-linux-installer.md`](../docs/changelog/CHANGELOG-1.7.7-linux-installer.md).

```sh
python3 tests/test-linux-installer.py
```

`test-zip-entry.py` covers the ZIP's front door: the Linux ZIP's `./gobbonet` is a
short script that hands over to `linux-amd64/gobbonet-launch`, so a ZIP user
gets the setup page and then the chat, as a `.deb` user does. It checks the
arguments, the symlink and missing-folder cases, and that `make-zip.sh` ships the
script rather than whatever binary is lying in the tree. Background is in
[`docs/changelog/CHANGELOG-1.7.7-zip-launcher.md`](../docs/changelog/CHANGELOG-1.7.7-zip-launcher.md).

## Preview pages (`preview/`)

A suite tells you the renderer produced the right markup. It cannot tell you
whether the result *looks* right. Open these in a browser for that:

- `preview/math-preview.html` — LaTeX and math rendering
- `preview/render-preview.html` — markdown, code blocks, chat bubbles

They load the real `css/` and `js/` from the repo root rather than copies, so
they cannot drift from what ships.

## Go tests

Alongside the code they cover, run the usual way:

```sh
go test ./...
```

`internal/webui/webui_test.go` covers when the unpacked `web/` folder is
rewritten: when the version changes, and when the page built into the binary
changes under the same version. Hand edits survive restarts of the same build.
Background is in
[`docs/changelog/CHANGELOG-1.7.7-web-refresh.md`](../docs/changelog/CHANGELOG-1.7.7-web-refresh.md).
