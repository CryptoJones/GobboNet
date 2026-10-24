# 1.7.8 — `gobbonet uninstall` deletes only GobboNet's files

From issue #63 (first-install findings, Solteris-Dev), item 14.

## What was happening

`gobbonet uninstall` removed the config by deleting the whole folder the
config file sat in. That is right for the folder GobboNet creates for itself
(`~/.config/gobbonet`), and wrong anywhere else:

- **A shared folder.** `--config` or `$GOBBONET_CONFIG` pointing into a shared
  folder lost everything in that folder, not just GobboNet's file.
- **The install folder.** Config discovery's last fallback is `./config.toml`
  beside `launch.bat`, which is the install folder. A relative `model_dir`
  puts the models there too. The folder went before the models question was
  asked, so `--keep-models` (what the Windows uninstaller passes) couldn't
  save them, or the program files.

## The change

- **GobboNet's own folder** is still removed whole.
- **Any other folder** loses only the files GobboNet writes beside a config
  it is pointed at:
  - the config file;
  - a `perf.toml` GobboNet wrote, recognised by its first line, because the
    name is generic;
  - leftovers from an interrupted atomic write (`.<name>.*.tmp`).

  The folder itself is removed only if that leaves it empty. Otherwise the
  output names the file it removed and says the folder was left in place.

## Tests

`cmd/gobbonet/uninstall_test.go`, four new cases:

- a shared folder keeps the user's files;
- a `perf.toml` GobboNet didn't write survives;
- the install-folder layout keeps its models and program under
  `--yes --keep-models`;
- a pointed-at folder left empty is removed.

The first three fail on the 1.7.7 code. The existing test still checks that
the default folder goes whole.
