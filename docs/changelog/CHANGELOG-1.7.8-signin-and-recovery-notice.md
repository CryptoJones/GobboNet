# 1.7.8 — Asked to sign in once, and an older-history notice that stays out of the way

Two reports from testing the 1.7.8 build on Windows.

## 1. "Sign in, tab out, immediately asked to sign back in"

> sign in, tab out, immediately get asked to sign back in, including on dev
> tool use, this seems like it may be a little aggressive, looks like it
> happens just on initial load

The address was `127.0.0.1:9066`, in a single tab.

### What was found

This was not reproduced exactly. A clean sign-in in Chromium, followed by
switching tabs or windows, stayed signed in. The same held:

- with a launcher-style password (the SHA-256 pair upgraded on first login);
- with an encrypted history;
- with no model;
- with history in the browser and on the server, a restart, and four
  tab-outs.

Reading the code turned up three things that fit the report.

- **One 401 was enough.** Coming back to a tab (focus, which DevTools also
  fires) runs a sync check and re-follows pending replies. Any one of those
  requests answering 401 locked the tab and opened SIGNED OUT on the spot,
  with no second look.
- **A prompt, once up, stayed up.** The session cookie belongs to the
  browser, not the tab. A tab still showed SIGNED OUT after the browser had
  signed in elsewhere, so the user signed in twice.
- **Sessions from this PC broke for no reason.** Every session is bound to
  the client's address and browser string. On loopback that binding:
  - protects nothing: no one off the network can reach 127.0.0.1;
  - breaks easily:
    - a Go listener on `0.0.0.0` is dual-stack (`favoriteAddrFamily` picks
      AF_INET6 for a wildcard), so one browser can arrive as `127.0.0.1`
      and as `::1`;
    - a browser extension, or DevTools' network conditions, can change the
      User-Agent mid-session.

### The change

- **The page checks before it asks** (`noteLocked`, `js/06-state-sync.js`).
  - A 401 now triggers one fresh GET: `/state/info`, which checks the
    session and, on an encrypted install, the unlocked key, just as the
    state routes do.
  - Only if that is refused too, or cannot be asked, does the tab lock and
    ask.
  - A tab still signed in carries on. The console names the request that
    was refused, and the caller's own retry goes out as usual.
  - A burst of 401s makes one check.
- **A locked tab heals itself.**
  - While SIGNED OUT is up, the tab looks again every time you come back to
    it, and every 5 s while it is on screen.
  - Once the browser is signed in (another tab, or the same tab's password
    manager in another window), it closes the prompt and resumes sync.
  - The prompt says so: "Signed in again in another tab? This one notices
    and carries on by itself."
- **Sessions from this PC are one client** (`auth.ClientFingerprint`).
  - Any loopback address (`127.0.0.0/8`, `::1`, `::ffff:127.x`) is the
    client "loopback", whatever its browser string.
  - From the network, nothing changes: address and browser are both bound.
    An IPv4-mapped IPv6 address counts as its IPv4 address.
- **The server says why it refused a session** (`internal/server/authlog.go`):
  - one line in the `gobbonet.exe` window, the first time each refused token
    and reason is seen;
  - it names the request and the reason: not issued by this run (a restart,
    or another GobboNet on the same address but another port, since cookies
    ignore ports), past `session_ttl_hours`, or a different address or
    browser;
  - only a hash of the token is kept, the token is never logged, and nothing
    is sent anywhere.

`TROUBLESHOOTING.md` has a new entry, "It asked me to sign in again right
after I signed in", listing those lines. `AGENTS.md` describes the binding.

**If it still happens on Windows,** the `[auth]` line in the gobbonet window
and the `[auth] signed out:` line in the browser console together say which
request was refused and why.

## 2. The older-history notice sat on the send button

> I've got a button in the lower right of the screen saying 'older history /
> storage notice - open data' its sitting on top of a few things like the send
> button and text entry field

### What was happening

The 1.7.6 hotfix's recovery notice was a fixed button at the bottom right
(`bottom:12px; right:12px; z-index:1000`), so it covered the right end of the
message box and the send button.

It shows whenever older history was kept through an upgrade: a server
recovery copy, or an older browser copy. Those copies are kept for good, so
the button came back on every load.

Reproduced in Chromium on the 1.7.8 build. A data folder holding a current
history plus an older `.gobbonet-state.json` put the button over both the
message box and the send button.

### The change

- **It is a bar under the chat header,** in the chat column's own flow. It
  pushes the conversation down rather than floating over it. It says "Older
  history from before an upgrade was kept separately. Nothing is lost.", or
  the storage warning in its own words, then **OPEN DATA** and an **×**.
- **The × hides it until what it reports changes**: the same older browser
  copy and the same server recovery copies.
  - A new older copy brings it back.
  - A storage warning (something wrong now) is hidden for that visit only.
  - Browser storage that refuses still lets the × work for the visit.
- **DATA's sidebar button** said "// DATA — RECOVERY", which is wider than
  its half of the row and ran out past the sidebar's edge. It now says
  "// DATA", with an amber dot and a tooltip. DATA's *Upgrade recovery*
  section is unchanged.

## Checked here

Real Chromium against a built server, the 1.7.8 build and this one side by
side:

- **Notice, at 1280×800 and at 390×844 (a phone):**
  - on 1.7.8 it overlapped the message box, the send button and the input
    area;
  - now it overlaps none of them, nothing is drawn over the message box, and
    it sits under the chat header;
  - × hides it, a reload keeps it hidden, and DATA still offers the
    recovery;
  - DATA's button stays inside the sidebar.
- **Sign-in:**
  - a stray 401 while signed in no longer opens SIGNED OUT;
  - after a restart, tab A is signed out and asks, which is correct;
  - signing in in tab B, then returning to A, lets A carry on with no second
    sign-in;
  - after another restart, signing the browser in elsewhere (same cookie
    jar, another User-Agent) let tab A notice by itself in about 3 s, with
    no click or focus;
  - the gobbonet log said why A was refused.

  On 1.7.8, 7 of those 12 checks fail.
- The earlier sign-in runs (launcher password, encrypted, no model,
  returning user) and the earlier browser checks still pass.

**Not checked:** Windows, and the exact sequence reported, which was not
reproduced. IPv6 could not be exercised here (the container has none), so the
`::1` half is covered by unit tests on the fingerprint, not a live
connection.

## Tests

- `tests/test-signin-prompt.mjs`, 27 checks on the real code: the second
  look, locking, one check per burst, the self-heal on return and on a
  timer, and the wiring.
- `tests/test-recovery-notice.mjs`, 23 checks: where the bar is, the CSS,
  when it shows, the ×, coming back, warnings, refused storage, and DATA's
  dot.
- `internal/auth/session_test.go`: loopback is one client across address
  family and browser string, network sessions stay bound, and `Check` gives
  the reason.
- `internal/server/authlog_test.go`: the same through `ServeHTTP`, one log
  line per refused session, and no token in the log.
