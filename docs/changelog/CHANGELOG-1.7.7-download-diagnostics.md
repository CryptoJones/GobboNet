# 1.7.7 — In-app downloads: testable, and every failure explains itself

Roadmap item #8b. **Live for testing, not confirmed working.** The *Add a
Model* button works again so the maintainer can test it on machines the
authoring sandbox can't reach, and says under it that it is being tested.

The full trace of the download path, the hypotheses, and the test matrix are in
[`docs/INVESTIGATION-in-app-downloads.md`](../INVESTIGATION-in-app-downloads.md).

## The changes

### The button is live, and says it is being tested

`js/02-model.js`:

- **`MODEL_CATALOG_ENABLED` is `true` again.** The button reads BROWSE
  CATALOGUE and works, from the PC or a phone.
- **A new `MODEL_CATALOG_CONFIRMED = false`** keeps the existing note under the
  button showing: "Being tested — not confirmed working yet", and where to look
  when a download fails. Set it `true` once testing passes and the note goes.
- **The hold is still one flag.** `MODEL_CATALOG_ENABLED = false` puts NOT
  WORKING YET back, if testing ends in retiring the path.

**The first attempt hid this behind `?test-downloads`** in the page address,
with the button still held by default. In testing the button did nothing, so
there was nothing to test, and the switch was lost on signing in anyway (the
login always returns to the plain address). The maintainer chose the live
button; the switch is gone.

### Every failure names its step

The steps are: checking the models folder, reading the checksum, contacting
Hugging Face, downloading, checking the file, moving it into place.

- **The server reports the step** in a new `stage` field.
- **The download window's title** reads "Download failed while …".

### Every failure says what usually causes it

`internal/modelfetch/diagnose.go`:

- **Windows "access denied"** names Controlled Folder Access, with where to
  allow `gobbonet.exe`, and antivirus.
- **Linux permission errors** name the folder's owner.
- **Disk full** and **read-only drive** are named as such.
- **Network failures** are told apart: DNS, a certificate failure (antivirus
  HTTPS scanning or an inspecting proxy), a refused or cut connection, a
  timeout.
- **Hugging Face's own refusals** are explained: 401/403 a gated model, 404 an
  out-of-date entry, 429 rate limiting, 5xx their side.

### Every step is logged with a `[download]` prefix

The lines go to GobboNet's window on Windows and to `launch.log` on Linux.

### The models folder is tried before downloading

Every file operation the download needs is tried with a few bytes first. A
folder that refuses one fails in a second rather than after gigabytes.

### Antivirus holding the finished file is waited out

On Windows, a sharing violation, lock violation or "access denied" at the final
rename is retried for up to 20 seconds.

### A download can no longer hang

The default HTTP client had no timeouts at all. Now:

| What | Limit |
|---|---|
| Connecting | 30 s |
| The TLS handshake | 30 s |
| Waiting for the server to start answering | 60 s |
| The checksum pointer | 30 s |
| The download itself | No total limit, but no data for 2 minutes stops it |

**The Linux setup wizard** uses the same downloader, so it gains all of the
above.

## Visual

No new elements:

- The button is back to its existing BROWSE CATALOGUE state.
- The note under it reuses the existing hint (`#add-model-unavailable`).
- The failed step goes into the download window's existing title line.

## Pinned by

**`internal/modelfetch/diagnose_test.go`** (new). What it covers:

- **The explanations**, for each platform.
- **The folder check**, including a refused rename.
- **Waiting out a held file**, on Windows only.
- **A run:**
  - stopping at the folder before any request;
  - a 403;
  - a stall;
  - a clean finish.

**The existing checksum tests** pass unchanged.

**`tests/test-download-testing.mjs`** (new, 24 assertions):

- **The button works** with nothing to find first.
- **It says it is not confirmed working**, until marked confirmed.
- **The hold still works as one flag**, and no address switch remains.
- **Each failed step is named.**

**`tests/test-model-catalog.mjs`:** its 68 assertions pass. The one case that
describes the button working normally now sets the confirmed flag, since that
is the state it describes.

## Verified here

On Linux with a real server:

- **An uncreatable models folder:** failed in under a second, "while checking
  the models folder", with no request made.
- **A writable folder with no internet:** failed "while contacting Hugging
  Face": "Could not look up huggingface.co … Is this PC online?". Nothing was
  left behind.
- **The button:** BROWSE CATALOGUE, enabled, with the *Being tested* note.

**Windows, antivirus, OneDrive, real downloads and the swap** are the
maintainer's matrix, in the investigation.
