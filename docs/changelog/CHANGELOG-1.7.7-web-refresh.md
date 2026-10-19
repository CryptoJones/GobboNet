# 1.7.7 — An update that keeps its version still updates the page

Roadmap item #13. Found while testing batch 2, whose zip carried the same
version stamp as batch 1. Added to the list at the maintainer's request.

## What was wrong

The server unpacks its built-in chat page into `web/` beside itself and serves
it from there, so the interface can be read and hand-edited. A stamp file
records which build wrote the folder. On start, `webui.NeedsExport` rewrote the
folder only when the stamp's version **string** differed from the running
build's.

So two builds with the same version shared one page:

- Run the first, then drop in the second, and the second build's interface
  never reached the screen.
- Nothing said so. ABOUT's version check compares the server's and the page's
  version strings, which agreed.

For test batches this meant "delete `web/` before starting". For real releases,
a re-cut build that keeps its version — a hotfix from the same commit, a rebuilt
binary dropped over an install — would have shown users the previous page.

## The change

**The stamp records the page as well as the build.** `Export` writes a
`content=` line: a SHA-256 over every file of the page built into the binary
(path and bytes, in a fixed order). `NeedsExport` rewrites the folder when the
version **or** that fingerprint differs.

**Fingerprinting the built-in page, never the files on disk.** The stamp
invites people to edit the unpacked files ("Edit these files and reload the
page"). Comparing the disk would wipe those edits on every start. Comparing the
binary's own page replaces them only when the build's page actually changed —
the promise the stamp already made for version changes, and now says in its
wording.

**Upgrading from an older stamp.** A stamp written before 1.7.7 has no `content`
line, so it is rewritten once, like a version change.

**When the server refreshes for this reason**, its startup notes now say that
this build of the same version carries a different interface. They used to read
"refreshed from 1.7.7-go-x to 1.7.7-go-x".

The code (`internal/webui/webui.go`):

- `Fingerprint`, and `BuiltFingerprint` (computed once);
- `ExportedFingerprint`;
- `Export` and `NeedsExport` built on a core that takes the page as a
  parameter, so it can be tested with two different pages under one version.

`internal/server/webroot.go` gains the new note.

## Pinned by

**`internal/webui/webui_test.go`**, the first tests the package has had:

- the fingerprint follows content and file names, and ignores `.gitkeep`;
- same version + different page → rewritten;
- hand edits survive restarts of the same build;
- a pre-1.7.7 stamp is rewritten once, not on every start;
- a version change, a lost `chat.html`, a missing stamp and an unreadable
  built-in page all still rewrite;
- the stamp records both lines and never writes `.gitkeep`.

**`internal/server/webroot_refresh_test.go`** gains the same-version case
through the real `builtInSource`, including the new note. The two existing
tests — a new build refreshes, and the same build keeps hand edits — pass
unchanged.

## Verified with real binaries

Two servers were built from one tree with the **same** version stamp, the second
with a marker added to its built-in page. Each was run in turn from the same
folder.

| Code | After A, then B — does `web/` hold B's page? |
|---|---|
| before this change | **no** — the bug |
| after | **yes**, and B's startup notes say why |

## Effect

- The "delete the `web` folder first" step is gone for good, for test batches and
  real releases alike.
- The per-batch local commits already avoided it for these batches; this fixes
  the cause.
