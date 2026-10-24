# 1.7.8 — Edit config.toml from CONFIG

> Add config.toml edit functionality in the config menu so users dont have to
> hunt for it

## What was happening

`config.toml` holds the server's own settings, and the panels in CONFIG cover
only a few of them. Everything else meant finding the file, which is in a
different place on every platform:

- **Windows:** `%USERPROFILE%\.config\gobbonet\`.
- **Linux:** `~/.config/gobbonet/`.
- **Portable or custom installs:** beside `launch.bat`, or wherever `--config`
  or `$GOBBONET_CONFIG` points.

After that, it meant editing the file with nothing to catch a mistake before
the next start refused to run.

## The change

CONFIG has a new **Config File — config.toml** box, next to Network access.
It names the file in use, and **OPEN EDITOR** shows the whole file, comments
and all, in a plain-text editor.

- **SAVE TO CONFIG.TOML:**
  - **Checks first.** The file is parsed exactly as GobboNet parses it at
    startup, with the same checks the server makes before it runs.
  - **What gets refused.** Anything that would stop the next start, with the
    reason:
    - broken TOML, with the line;
    - a wrong type;
    - a misspelt key ("did you mean …");
    - a port out of range;
    - a `server_exe` that does not exist;
    - an `llm_api_key_file` that cannot be read.
  - **The cursor** goes to the line an error names.
  - **Address or port changes** get the same checks as Phone & Other Devices:
    - an address this PC has;
    - a port GobboNet allows and the AI engine does not use;
    - a port that is free right now.

    A new port is also written to the Windows launcher's port file, as that
    panel does.
  - **What it reports:** which settings changed, and that they apply the next
    time GobboNet starts. A key this version does not know is accepted and
    mentioned.
- **Conflicts:** another panel, a launcher, or a sign-in may have rewritten
  the file since it was opened (a sign-in can upgrade the password hash). A
  save is then refused, the edit stays in the box, and **RELOAD** shows the
  file as it is now.
- **UNDO LAST SAVE:** the version a save replaced is kept as
  `config.toml.bak`, and this button puts it back through the same checks.
- **Unsaved edits:** CONFIG's own **SAVE** never writes the file. Closing
  CONFIG (button, backdrop or Escape) or pressing RELOAD with unsaved edits
  asks first.
- **Overrides:** the box says when an environment variable or CONFIG >
  Performance overrides what the file says.

### Privacy and safety

- **Secrets never leave the server.**
  - `access_secret` (the password hash) and `llm_api_key` are shown as
    `(hidden)` and put back from the file when it is saved.
  - The password cannot be changed here. A save that alters or drops it is
    refused, and the message says to use `gobbonet set-password`.
  - A new upstream key can be typed in place of `(hidden)`, and is never shown
    back.
  - UNDO LAST SAVE keeps the password as it is now, so a backup from before a
    `set-password` cannot bring the old one back.
- **It opens only on the PC GobboNet runs on.** `config.toml` decides which
  programs GobboNet starts and which folders it reads, so from anywhere else
  the editor would be a way to run programs on this PC. That covers:
  - a phone;
  - another computer;
  - someone who has read a session cookie off the LAN.

  A browser on this PC can do nothing here that its user could not do by
  opening the file. The server enforces this (`internal/server/configfile.go`):
  - **Address:** the request must come from loopback and name a loopback host.
  - **Proxies:** a proxy-forwarding header disqualifies it, so a reverse proxy
    on this PC cannot pass remote visitors through.
  - **Other sites:** browser requests must be same-origin, so another website
    cannot read or save the file.

  Anywhere else, the box says to open GobboNet at `http://127.0.0.1:<port>` on
  that PC, and the server sends nothing of the file, not even its path.
- **The file and its backup are written atomically** and readable by this user
  only (0600).
- **`gobbonet uninstall`** removes the backup along with the config file,
  since it holds the password hash too.

## Checked here

These were checked in real Chromium against a built server.

- **On this PC:**
  - the box names the file, and the editor shows it with both secrets as
    `(hidden)`; neither secret appears anywhere in the page or its responses;
  - an edit saves with its secrets and comments, leaves a 0600 `.bak`, and
    reports what changed;
  - broken TOML is refused with the line, and the cursor goes there;
  - a misspelt key, a password change and the engine's port are each refused;
  - a save after `gobbonet config set` changed the file is refused, with the
    edit kept and the other change not overwritten;
  - RELOAD and UNDO LAST SAVE work, and closing with unsaved edits asks;
  - GobboNet restarts on the saved file.
- **From the LAN address, at phone size:** the box says where to open it, has
  no editor, and receives nothing of the file. A save from there is refused.

**Not checked:** Windows, including how the launcher's port file and a
`%USERPROFILE%` path look in the box.

## Tests

- **`internal/config/editfile_test.go`:**
  - secrets are hidden and restored;
  - a round trip keeps comments and a 0600 backup;
  - the password rules;
  - everything that would stop the server is refused without writing;
  - future keys produce a warning;
  - conflicts, at prepare and at commit;
  - UNDO keeps today's password;
  - the shipped default config saves back unchanged.
- **`internal/server/configfile_test.go`:**
  - this PC only: loopback, `localhost` and `::1` count; the LAN, IPv6 LAN,
    this PC by its LAN address, and forwarding headers do not;
  - other sites cannot read or save;
  - every save rule;
  - conflicts, invalid files, the engine's port, a foreign address, a busy
    port, and restore;
  - the password is still required.
- **`cmd/gobbonet/uninstall_test.go`:** the backup is removed with the
  config.
- **`tests/test-config-file-panel.mjs`:** 37 checks on the panel, run for real
  against a stub page and server, covering:
  - the markup;
  - opening (this PC, elsewhere, signed out, opened as a file);
  - override notes;
  - saving, refusals with the cursor on the line, and conflicts;
  - undo, reload, and closing with unsaved edits;
  - CONFIG's SAVE never writing the file.
