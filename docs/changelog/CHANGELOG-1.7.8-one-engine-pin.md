# 1.7.8 — One llama.cpp pin for Windows and Linux

From issue #63 (first-install findings, Solteris-Dev), item 11.

## What was happening

`engine.sha256` describes itself as "the build BOTH installers bundle". The
Linux packages, the ZIP and `gobbonet engine install` read it (b10456).
`launch.bat` carried its own pin (b9294), so every Windows first run
downloaded an older engine than Linux shipped.

The Windows installer also didn't install `engine.sha256`. On an installed
copy, `gobbonet engine install` couldn't find a pin at all.

## The change

- **`launch.bat` reads `LLAMA_BUILD` and `WIN_GPU_SHA256` from
  `engine.sha256`** beside it.
  - **File absent** (an install from before it shipped): the launcher uses a
    built-in copy of the same pin, and `tests/test-engine-pin.py` fails if
    the copy ever differs from the file.
  - **File present but missing a value:** that value stays empty and the
    existing checks fail closed. A wrong tag can't match the hash, and a
    missing hash asks for an explicit YES before anything runs.
- **The unpinned-download hint** now says to add the hash to `engine.sha256`,
  not to edit `launch.bat`.
- **`installer-win` installs `engine.sha256`** and its uninstaller removes it.
  It is added to the staging list in `installer-win/README.md`.

## Checked before switching Windows to b10456

- **The Windows asset** `llama-b10456-bin-win-vulkan-x64.zip` downloads, and
  its SHA-256 matches `WIN_GPU_SHA256`. `llama-server.exe` is at the top of
  the archive, where the extract step looks first. From this build on it is
  a small launcher beside `llama-server-impl.dll`. Both extract to the same
  folder, as they do on Linux, which already runs b10456.
- **Every built-in chat template name GobboNet can emit exists at b10456.**
  Its `src/llama-chat.cpp` has 54, a superset of the 53 that `classify_test.go`
  pins at b9294. The test list stays at b9294 on purpose, because existing
  Windows installs keep the engine they already downloaded.
- **GPU-offload confirmation** needs `-lv 4` on builds from b9829. The launcher
  already passes it.

## Worth knowing

Dae's measurements in #65 (8 GB RTX 4070 Laptop, Vulkan) compared the two
builds:

- b9294 could not load gpt-oss 20B with automatic placement.
- b10456 processed prompts about half as fast for Qwen3.6 35B-A3B with its
  experts in system RAM: 72–100 vs 135–160 tokens/s. Generation speed was the
  same.

Existing Windows installs are not moved: the launcher only downloads an engine
when none is present.

## Tests

`tests/test-engine-pin.py`, 18 checks:

- an emulation of the launcher's exact `for /f` over the real
  `engine.sha256`, with LF and CRLF line endings;
- the fallback pin matches the file;
- the asset name agrees across `launch.bat`, `internal/engine` and
  `installer/build-installer.sh`;
- the installer installs and removes the file, and the ZIP requires it.

8 of these checks fail on the 1.7.7 files. The installer was not compiled here
(no NSIS).
