# 1.7.7 — Choose the address your phone connects to

Roadmap item #7. The maintainer's clarification: the phone uses something like
`192.168.x.x:9066`, the PC uses `127.0.0.1:9066`, and users want to be able to
change the address their phone connects to. The design in the roadmap (show the
addresses, then choose the address and port) was given the go-ahead on
2026-10-12.

## What was there

- **The server already knew the answer.** It listens on every address by
  default, ranks them by adapter (ordinary network adapter, then virtual, then
  VPN), and sent the list to the page in `/health`. The page never showed it.
- **The only place the addresses appeared was the console banner**, which
  scrolls away.
- **Changing the address or port** meant re-running setup or editing
  `config.toml` by hand.
- **Pinning one address had a trap.** A `listen_host` naming one network
  address was a single listener, so `127.0.0.1` stopped working on the PC
  itself — the address the launchers open, and the one the Linux launcher
  checks to see whether GobboNet is already running.

## The change

### A new CONFIG box: *Phone & Other Devices*

It sits after *Performance*, the other server setting, and has two parts.

**1. The addresses phones can use.** Every address the server is answering on:

- named after its network connection (`Wi-Fi`, `Ethernet`, `eth0`);
- the most likely first;
- virtual adapters and VPNs marked "unlikely", as in the startup banner;
- each with its own **COPY** button.

When phones can't connect at all, the box says so instead, and why:

- set to answer on this PC only;
- the network address was refused, with the real error;
- or no network address was found.

**The one thing GobboNet can't do** is said in one line: the numbers come from
the router, and only the router can make one permanent (a "DHCP reservation").

**2. Choose the address and port.**

- **Answer on:**
  - *Automatic (all addresses)*, the default;
  - any one address this PC has;
  - *This PC only (no phones)*, which the Linux setup wizard can already
    choose.
- **Port**: 1024–32767 on Windows, 1024–65535 elsewhere (see below).
- **SAVE FOR NEXT START** writes `listen_host` and `listen_port` to the config
  file. The change applies the next time GobboNet starts.

### Nobody is cut off by a save

**The new address is shown before saving.** As you choose, a line reads:
"After the next start: phones use `http://192.168.1.20:9070/`; this PC uses
`http://127.0.0.1:9070/`."

It warns when the address the page is open at will stop working, and when a
phone is about to switch itself off with *This PC only*.

**Nothing moves under the page.** The running server keeps its address until
it restarts, the same as Performance's *SAVE FOR NEXT START*.

**A pinned address keeps loopback beside it.** The server now also answers on
`127.0.0.1` (or `::1`), so the PC itself keeps working and the launchers'
browser tab still lands. The startup banner's "ON THIS PC" line shows loopback
for a pinned address too.

If something else already holds loopback on that port, the pinned address is
still served. The banner and the box then name the pinned address for this PC
instead.

### A save is checked before it is written

| Asked for | What happens |
|---|---|
| An address this PC doesn't have | Refused, with "Pick one from the list, or Automatic." |
| A port outside the range | Refused. On Windows the range is 1024–32767, the bounds `launch.bat` and `setup-lan.bat` enforce: a port outside them would be silently replaced with 9066 on the next launch, which would look like a broken setting. Elsewhere it is 1024–65535, the Linux setup wizard's own range, so a port chosen there is never refused here. |
| The AI engine's port | Refused: 11437 and 11436 (as the Linux wizard refuses them), and the ports `llm_url` and `embed_url` use on this PC. |
| A port that is already taken, or that Windows has reserved | Refused before saving, by trying it, with the reason. Otherwise the next start would fail on a port the user could only fix by hand. |
| Another site's page trying to change it | Refused. A present `Origin` must be GobboNet's own. |

The `Origin` rule is the one check this route makes beyond its neighbours'.
Without a password, any page could otherwise switch *This PC only* to every
address.

### The things a save can't do on its own, said where they happen

- **Windows' firewall rule** from *GobboNet LAN Setup* is for one port. After a
  port change, the box says to run LAN Setup again.
- **Windows' `launch.bat`** takes the port from `.gobbonet-port` and writes it
  into the config before starting the server. A port saved only to the config
  would therefore be put back on the next launch. The save writes
  `.gobbonet-port` too, so the launcher carries the new port forward. If it
  can't, the box says so.
- **`GOBBONET_LISTEN_PORT` / `_HOST`** (or the `GEMMA_` spellings) set in the
  environment win over the file. The box names any that are set, as a problem
  rather than a success.

### When the box can't work

It is disabled with the reason, like Idle Stand-Down:

- opened from a file;
- served by the PowerShell file server;
- an older program file without `/lan`, which gets the same one-file-update
  advice.

## Visual

Nothing new was designed. The box is copied from existing components:

| Part | Copied from |
|---|---|
| The box | Idle Stand-Down's cyan box, character for character |
| The address rows | ABOUT's build rows (label, then monospace value), with `btn btn-sm` COPY buttons |
| Answer on / Port | Performance's `form-row`, select and number input |
| SAVE FOR NEXT START | Performance's button, wording and class |
| The status line | Idle Stand-Down's |

Checked at 1440 and 390 wide. On a phone the pair stacks, exactly as
Performance's does.

## Pinned by

**`internal/server/lan_test.go`** (new):

- **A pinned address serves loopback too.** A real request to each answers 200.
- **Closing stops both, and serving ends cleanly.**
- **Only a pinned address gets the companion.** Wildcards and loopback are
  unchanged.
- **A taken loopback port** doesn't stop the pinned address.
- **`GET /lan`:** the fields the page reads, and the config file's values
  rather than the running ones.
- **`POST /lan`:**
  - writes the file, and `config.Load` reads it back;
  - refuses an address the PC doesn't have, a hostname, ports outside the range,
    the engine's ports and a port in use, all without touching the file;
  - doesn't probe the port it is already on;
  - refuses other origins;
  - reports environment overrides.

**`internal/config/listen_test.go`** (new):

- **`ReadListen`** reads the file alone.
- **The override list** is correct.
- **The panel's port range** matches `launch.bat` and `setup-lan.bat` on
  Windows and the Linux wizard elsewhere, so they can't drift apart.
- **The engine ports** include the defaults and this PC's `llm_url` and
  `embed_url` ports, and not another machine's.

**`cmd/gobbonet/ready_test.go`:** a pinned address with loopback shows
loopback as "ON THIS PC".

**`tests/test-lan-panel.mjs`** (new, 79 assertions, on the real panel code):

- **Every disabled state** says why and sends nothing.
- **The rows:** order, labels, COPY, and escaping of adapter names.
- **The choice:**
  - is filled from the file;
  - keeps a saved `::`;
  - shows a saved address that has left the PC as such.
- **"Phones can't connect"**, each way.
- **The preview:**
  - from this PC and from a phone;
  - pinned, this PC only, a new port;
  - the Windows-only firewall line;
  - an IPv4 example for `0.0.0.0`.
- **Saving:**
  - sends exactly `{host, port}`;
  - refuses a bad port locally;
  - shows the server's refusals;
  - flags environment overrides and an unwritable `.gobbonet-port`.
- **The pieces it is built from.**

## Verified with a real server

On a Linux sandbox with one network adapter (`eth0`, 192.0.2.2):

- **Defaults:** the box listed `eth0  http://192.0.2.2:9066/  COPY`. COPY put
  exactly that address on the clipboard.
- **Pinned to 192.0.2.2 and port 9071, then saved:**
  - the file read `listen_host = "192.0.2.2"` and `listen_port = 9071`;
  - the box said a change was waiting;
  - after a restart, `192.0.2.2:9071` and `127.0.0.1:9071` both answered 200;
  - the banner showed `ON THIS PC http://127.0.0.1:9071/`, and the box worked
    through either address.
- **Set to this PC only:** the box said phones can't connect and how to let
  them in.
- **Opened from the network address:** choosing *This PC only* warned that this
  device would be cut off. Changing the port warned that the page's address
  would change.
- **No page errors.**

## Not tested here

- **Windows:** the `.gobbonet-port` hand-over to `launch.bat`, and the firewall
  rule after re-running LAN Setup.
- **A machine with several adapters**, where the ordering is the whole point.
  It is covered by the existing `lanaddr` tests with simulated adapters.
