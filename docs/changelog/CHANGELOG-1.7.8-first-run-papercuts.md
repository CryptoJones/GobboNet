# 1.7.8 — First-run papercuts from the #63 test run

Issue #63 (Solteris-Dev) reported a first install of 1.7.6 on Windows and
Linux. Five of its smaller findings are fixed here. The model-menu exit and the
double password prompt have their own notes:
`CHANGELOG-1.7.8-model-menu-input.md` and `CHANGELOG-1.7.8-password-handoff.md`.

## Enter now submits the setup wizards

The password step's fields were outside any form, with Continue wired to a
click handler, so pressing Enter did nothing. The same was true of every step
with a text field.

Each such step's card is now a `<form novalidate>`. A form turns Enter into a
click on its Continue button, so the existing handlers run unchanged, and the
script cancels the submission itself.

- **Go wizard** (`gobbonet setup`): password, backend.
- **Linux wizard** (deb/rpm and the ZIP): folder, port, password, backend.
- **Tests:** `tests/test-wizard-enter.py`. Also checked in headless Chromium.

## "Gemma 4" is gone from LAN setup

`setup-lan.bat`'s window title and banner said "Gemma 4", and its firewall
rules were named `Gemma4-Web` and `Gemma4-mDNS`. These are now `GobboNet-Web`
and `GobboNet-mDNS`.

- **Upgraded machines:** each old rule is deleted only after its replacement
  is confirmed to exist. A failed add never leaves the phone without a rule.
- **Teardown:** `teardown-lan.bat` removes both the new and the old names.
- **`gobbonet doctor`:** still recognises `Gemma4-Web` as a working rule under
  its old name, rather than reporting "NO RULE".
- **Tests:** `tests/test-setup-lan.py` and `cmd/gobbonet/doctor_firewall_test.go`.

## The console stops repeating the same proxy error

With no model loaded, the chat page's five-second health poll was refused and
logged every time, one line every five seconds. That was hard to read past,
and worse through a screen reader.

The first failure is still logged. Repeats of the same failure are counted,
and one line reports the count when the failure changes or the engine answers
again. Clients still receive the error on every request.

- **Tests:** `internal/proxy/proxy_test.go`.

## Linux gets Linux advice

The console panel told Linux users to run `setup-lan.bat` and `netsh`. A
server started from the desktop launcher, which has no terminal, was told to
keep its window open and press Ctrl+C.

- **Linux and macOS:** their own bind-failure steps, and a pointer to
  `gobbonet doctor`.
- **No terminal:** the panel says the server is running in the background and
  how to stop it (`pkill -x gobbonet`).
- **Windows:** the text is unchanged.

## Type the phone address with http://

Typed without the scheme, iOS Safari tries https. This plain-HTTP server can't
answer that, and Safari's error never mentions GobboNet (reported in a comment
on #63). The console panel and the CONFIG phone-address hint now say to type
the address as shown, with `http://` included.
