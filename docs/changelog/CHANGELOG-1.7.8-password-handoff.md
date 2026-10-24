# 1.7.8 — The password is asked for once, not twice

From issue #63 (first-install findings, Solteris-Dev), item 4.

## What was happening

`launch.bat` asks for a password on first run and stores it as a salt:hash
pair in `.gobbonet-secret`. It copied that into `gobbonet.exe`'s `config.toml`
only at `:launch`, the last step before the server starts.

Any run that stopped earlier left `config.toml` with no password: a cancelled
llama.cpp download, a closed window, or the model-menu `[FATAL]`. The Start
menu and desktop shortcuts run `gobbonet.exe launch` directly (since the 1.7.6
hotfix), so the next start asked the user to choose a password again.

## The change

The same `config set access_secret` now also runs right after the stored
secret is validated, before the first prompt that can end the run. The write
at `:launch` is unchanged. Both use the default config file and both are
skipped for the legacy server, or when `gobbonet.exe` is missing.

The server already accepted the salt:hash form and upgrades it to Argon2id on
the first login; that is unchanged.

## Tests

`tests/test-launch-secret-handoff.py`, 14 checks:

- **Static:** where the write sits in `launch.bat`, its guards, and CRLF.
- **Against the built server:** with no secret, a server with no terminal
  refuses to start. After the launcher's exact command with a launcher-style
  salt:hash, it starts without asking, refuses a wrong password, accepts the
  right one, and upgrades it to Argon2id.

Not run in a real Windows console here.
