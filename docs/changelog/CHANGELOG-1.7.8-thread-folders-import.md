# 1.7.8 — Imported threads no longer vanish when their folder is missing

Issue #61, reported by VeroomTershel against 1.7.3 and 1.7.6.

## What was happening

A thread names its folder by id (`folderId`). The threads export wrote the
threads but not the folders, so a thread imported on another install, or after
its folder was deleted, pointed at an id that did not exist there.

The sidebar draws a thread either inside a folder section or in the unfiled
list, and the unfiled list only took threads with no `folderId` at all. A
thread with a dangling one matched neither, so it was stored but never drawn:

- the import said "Imported 1 thread(s)";
- importing again said it already existed;
- re-exporting still included it;
- the only way to see it was to edit the JSON and set `folderId` to `null`.

Making a folder with the same name did not help, because matching was by id.

## The change

**1. The sidebar draws a thread whose folder is missing in the unfiled list**
(`renderSidebar`, `js/12-render.js`). This rescues anyone already affected
without a re-import. It is decided when the list is drawn, not by rewriting
`folderId` on load, so device sync sees no change and the thread moves back
into its folder if that folder turns up later. "Move to folder" marks such a
thread as "No folder" for the same reason.

**2. Thread exports carry the folders their threads sit in** (`folders`:
id and name only, in both the bulk and the single-thread export).

**3. Import gives each new thread a folder that exists here**
(`_placeImportedThreads`, `js/21-data.js`), in this order:

1. a local folder with the same name, so recreating "Campaigns" by hand and
   importing into it now works;
2. otherwise the folder from the file, recreated with its id;
3. a file written before this change carries no folders, so those threads go
   to the unfiled list, and the status line says so.

Threads already on this install are never touched by an import.

## Tests

`tests/test-folder-import.mjs` runs the real `renderSidebar`,
`openFolderPicker`, `exportData`, `exportThread` and `importData`: 34 checks,
including the reporter's exact steps. Against the 1.7.7 code it fails 19 of
them; with this change all pass.
