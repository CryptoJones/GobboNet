# Building the packages

> Renamed from `BUILDING-1.7.3.md`. A build guide whose *filename* carries a
> version is a file that is wrong from the next release onward — the same
> disease as the frontend build stamp that sat at `1.6.0-no-encoded-payload`
> through the whole of 1.7.x. `VERSION` is the one place a release number
> belongs.

Three artefacts, three commands, in this order. Everything below was verified
against the tree as it ships except where marked.

---

## Order matters

`build-release.sh` produces the binaries the other two package. Run it first.

```sh
./build-release.sh                    # Go binaries, stamped from VERSION
# Windows: installer-win/ -- stage the payload, then makensis (installer-win/README.md)
cd installer-linux && ./build-deb.sh  # gobbonet_<VERSION>+go.<sha>-1_amd64.deb
```

**The Windows installer is `installer-win/`, not `installer/`.** It is the dark
installer: Welcome, Destination, Password, Port, Install, the Defender notice,
then Finish with *Launch GobboNet* and *Set up phone access*. The first launch
opens `launch.bat` in a console window, which fetches llama.cpp, detects the
hardware and offers the model list. It is the Windows installer from 1.7.0 on,
1.7.4 apart (below).

`installer/` is a separate design that was never meant to ship. It chooses a
backend and downloads a model during setup. It was built by mistake for 1.7.4,
and again for the first 1.7.7 test build, and both times it was the wrong
installer. Do not build it for a release.

The Debian revision — the `-1` — counts PACKAGING-only changes to one upstream
version and resets when `VERSION` moves. `build-deb.sh` defaults it to 1 so a
new release resets it with nobody having to remember; a packaging-only rebuild
of the same upstream version passes the next number:

```sh
DEB_REVISION=2 ./build-deb.sh
```

It used to be the literal `-3`, left over from 1.7.3's three packaging repairs,
which meant 1.7.4 shipped claiming two earlier Debian revisions that never
existed.

`VERSION` is the single input to all three. Tag `v` + `VERSION` (for 1.7.7,
`v1.7.7`) before building, or `TestVersionFileMatchesUpstreamRelease` will fail:
it compares `VERSION` against the nearest git tag, on purpose.

Two places carry the number as a literal and move with `VERSION`. A test fails
if either is left behind:

- `GOBBONET_UI_VERSION` in `js/01-config.js` (`tests/test-version-stamp.mjs`);
- `APPVER`, `OutFile` and `VIProductVersion` in `installer-win/gobbonet.nsi`
  (`tests/test-version-strings.py`).

---

## Preflight

| Need | Check | Status here |
|---|---|---|
| Go ≥ 1.25 | `go version` | **unavailable in the authoring sandbox** — see below |
| NSIS 3.09 | `makensis -VERSION` | verified, 3.09 |
| `dpkg-deb`, `zstd`, `fakeroot` | for the deb | |
| Network to GitHub releases | `build-deb.sh` fetches the engine | |

`build-installer.sh` warns if NSIS is not 3.09, because Elodine's 1.3 build used
it and a version match keeps the output byte-comparable for review. 3.09 is what
this was compiled against.

---

## The one thing not done here

**The Go binaries are not built, so no `.deb` and no `.exe` are in the zip.**

The sandbox has Go 1.22.2 from apt against a `go 1.25.0` module, and both the
toolchain download and `proxy.golang.org` are blocked:

```
$ go build ./cmd/gobbonet
go: downloading go1.25.0 (linux/amd64)
go: download go1.25.0 for linux/amd64: toolchain not available

$ GOTOOLCHAIN=local go build ./cmd/gobbonet
go: go.mod requires go >= 1.25.0 (running go 1.22.2; GOTOOLCHAIN=local)

$ curl -o /dev/null -w '%{http_code}' https://proxy.golang.org/...   403
$ curl -o /dev/null -w '%{http_code}' https://go.googlesource.com/crypto   403
```

`GOOS=windows` fails at the same line, so there is no `.exe` either — the
installer is compile-verified but has no binary to wrap.

Working around that would mean building Go from source through a two-stage
bootstrap and then hand-redirecting every module to a GitHub mirror, because
`golang.org/x/...` resolves through a blocked host. The result would not verify
against `go.sum`. For a project that pins a SHA-256 on the engine, on the embed
model and on every catalogue entry, a binary assembled from unverified
dependency sources is not something to hand over — it would be the one
unpinned thing in the build.

**The alternative was worse.** Repacking the 1.7 binary out of the uploaded
`.deb` and relabelling it 1.7.4 would produce packages that pass every
smoke test and contain none of this release: no upstream `/v1/models`, so #47,
#48 and #27 still broken; no ranked LAN addresses; no `doctor` firewall section.
The version string would name a release it was not built from — exactly what
`TestVersionFileMatchesUpstreamRelease` exists to catch.

So: run the three commands above on a machine with a real toolchain. The
packaging inputs are all verified and waiting.

---

## What *was* verified

**`installer/gobbonet.nsi`** — compiled clean with NSIS 3.09, 0 errors,
0 warnings, producing a 772,678-byte installer, against a payload built from the
real `stage-web.sh` output and the real `.bat` / `.ps1` files. Only
`gobbonet.exe` and the `llama-cpp/` contents were stubbed, and neither is parsed
at compile time — they are `File` directives.

**`stage-web.sh`** — runs clean; `chat.html`'s 24 js and 17 css references match
the tree, and all five JS modules changed this release reach the payload.

**Layout** — the deb tree `build-deb.sh` produces was diffed against the shipped
`gobbonet_1_7_go_9051875-1_amd64.deb`: same paths, same `usr/lib/gobbonet/`
layout, same `usr/bin/gobbonet` shim, same `web/` placement.

**`VERSION`** — read by `build-release.sh`, `build-installer.sh` and
`build-deb.sh`; `installer/gobbonet.nsi` takes it as a `-D` define and hardcodes
nothing. (This section was verified for 1.7.4. Since then `installer-win/`'s
script and the page stamp carry literals of their own; see "Order matters"
above.)

---

## Engine fetch

**`engine.sha256` at the repo root is the single pin for all three builds.** It
carries `LLAMA_BUILD` and one hash per platform archive; bumping the build means
updating every hash in the same commit.

It moved here from `installer-linux/` in 1.7.4. Before that only the deb build
pinned anything — the Windows installer bundled whatever was sitting in
`installer/vendor/`, with a look for `ggml-vulkan.dll` as the only check. An
`.exe` and a `.deb` carrying the same version number could therefore contain
different engines, with nothing recording which. That is precisely the question
you want answered when one platform reproduces a bug and the other does not.

All three builders now fetch from GitHub releases and refuse a mismatch. The
Windows payload also carries `llama-cpp/ENGINE.txt`, so the bundled engine is
still identifiable after installation. To reuse an engine you already have:

```sh
SKIP_ENGINE_FETCH=1 ./build-deb.sh          # uses installer-linux/vendor/
BUNDLE_CPU_ENGINE=1  ./build-deb.sh         # also ship the CPU-only archive

SKIP_ENGINE_FETCH=1 ./build-installer.sh    # installer/ only, which does not ship
LLAMA_CPP=/path/to/llama-cpp ./build-installer.sh   # bypasses the pin
```

The Windows installer (`installer-win/`) bundles no engine. `launch.bat`
downloads one on first launch and checks it against its own pin,
`LLAMA_PIN_TAG` and `LLAMA_PIN_SHA256` near the top of the file. That pin is
separate from `engine.sha256`.

`LLAMA_CPP=` is the one route that skips verification, because a hand-placed
directory cannot be hashed against an archive nobody kept. The build says so
rather than implying a check that did not happen, and `ENGINE.txt` in the
payload records it as unverified.

The uploaded 1.7 deb carries a usable Vulkan engine at
`usr/lib/gobbonet/llama-cpp/` if you want to populate `vendor/` from it rather
than re-downloading — but check `engine.sha256` first, since 1.7.4 may pin a
newer `LLAMA_BUILD` than that package was cut against.
