# Patch 2: preserve history through the split-storage upgrade

## Report and confirmed code paths

Users can see an empty library after upgrading even when older history remains
on disk. The exact affected database was not supplied. Two concrete gaps were
identified and reproduced in regression tests:

- The PowerShell server saved `.gobbonet-state.json` in the installation root;
  the Go server reads `state.json` in its configured data directory, with no
  import between them.
- Browser migration committed IndexedDB metadata before conversation records.
  An interrupted or failed second transaction left a database that appeared
  initialized but contained no conversations. Restores also cleared records
  separately from writing their replacements.

Per-conversation server sync still uses one compatible JSON document on disk.
It does not require destructively splitting an existing server file.

## Automatic migration

After authentication/unlock, before any state read or write, the server checks
for the documented old filename in its configured data/config/web directories,
executable directory and launch working directory. It does not search arbitrary
folders or other users' data. One old copy with no current state is imported
as the shared history. Existing current state is never overwritten. Distinct
older copies are preserved as separate `legacy-...` recovery profiles instead.

Original files remain unchanged. New files are written atomically and sealed
when encryption is enabled. `legacy-import.json` records completion; retain it
after deleting history so old originals cannot reappear automatically. Invalid
sources pause state writes with an explanatory error instead of accepting an
empty replacement. Existing named profiles are not combined or reformatted.

The browser now commits metadata and conversations in one IndexedDB transaction
for migration, full saves and restores. It retains original localStorage keys,
retries a failed migration, recognizes a merged meta document, and recovers
conversation records whose metadata is missing. Existing per-thread records
win over conflicting merged records, with the original merged copy retained.
Read failures pause automatic saves/sync instead of persisting default state.

## Recovery UI

DATA > Upgrade recovery makes separately preserved server histories and older
browser copies available. A visible notice also works with the mobile sidebar
closed. An empty unmarked database plus an old backup is ambiguous: the user
may have intentionally deleted those chats. Offer recovery rather than restore
stale data without asking. Recovery retains the current browser snapshot first;
it can be downloaded from the panel. Browser recovery leaves sync off until
the user chooses a target, protecting existing shared server history. Server
recovery opens a separate profile. Restored custom code stays disabled.

Copies remain local. Old plaintext source files are deliberately left unchanged,
even when the new server copy is encrypted. Normal local purge clears browser
recovery records as well. Browser data on a different origin or old files moved
outside known installation directories still require the existing import flow.

## Verification

Go tests: legacy discovery through full/per-thread HTTP routes, existing shared
and named profiles, multiple originals, malformed sources, metadata-only data,
encrypted import, restart/idempotence, and no resurrection after deletion.
JavaScript tests: transaction abort and retry, merged/orphaned records, retained
originals, ambiguous empty databases, read failures, recovery cancellation,
safety copies, code quarantine and failed profile-switch rollback.
Transaction commit/rollback was additionally verified with fake-indexeddb using
the production persistence functions. Windows runtime, real browser visual QA
and a reproduction with the reporting user's database remain outstanding.
