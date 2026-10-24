# 1.7.8 — GobboNet on a phone's home screen

From issue #63 (first-install findings, Solteris-Dev), the last of its short
fixes.

## What was happening

GobboNet had no web app manifest and no touch icon. *Add to Home Screen* still
made a shortcut, but with a screenshot or a letter for an icon and the page's
long title, `GOBBONET // LOCAL_TERMINAL [llama.cpp]`, cut short for a name.

The shortcut matters more than its looks. Typed without `http://`, the address
sends Safari to `https://`, which a plain-HTTP server on the LAN cannot answer,
and the error says nothing useful. A home-screen icon keeps the whole address,
so after the first visit there is nothing to type.

## The change

- **`manifest.webmanifest`**: name and short name *GobboNet*, `display:
  standalone`, the page's own background colour, and two icons. `start_url`,
  `scope` and `id` are `./`, relative to the manifest, so it works at whatever
  address GobboNet is reached on. It names no URL anywhere else.
- **`icons/`**, made from the package's own art
  (`installer-linux/icon/gobbonet-256.png`):
  - `gobbonet-180.png` is the iPhone and iPad touch icon;
  - `gobbonet-192.png` and `gobbonet-256.png` are for Android;
  - the 256 is that file byte for byte, and the smaller two were resized from it.

  There is no 512: the largest art in the repository is 256, and an upscaled
  icon would only look worse. Chrome would want a 512 to *install* the page as
  an app, but it only installs over HTTPS, which a LAN address isn't. *Add to
  Home screen* works without it.
- **The tags, on both pages a phone meets:** the manifest and touch-icon links,
  `mobile-web-app-capable` and `apple-mobile-web-app-capable`, the
  home-screen title, a black status bar and the theme colour.
  - `chat.html` uses relative paths, like its CSS.
  - The login page uses absolute ones, because it is also served at any deep
    link a signed-out browser opens. The phone usually meets GobboNet at the
    login page first, so that is where *Add to Home Screen* happens.
  - The status bar is `black`, not `black-translucent`: the page has no
    safe-area padding, so a translucent bar would cover the header.
- **Served before sign-in.** The manifest and the three icons are on a list,
  `publicAssets` in `server.go`, of files answered before the password check:
  - The login page has to show the right icon.
  - Chrome fetches a manifest without cookies in any case.
  - They are named one by one, not as an `/icons/` prefix, so nothing put in
    the web root later becomes public by accident.
  - GET and HEAD only.
  - None of them says anything about the user.
- `.webmanifest` is served as `application/manifest+json`.
- `stage-web.sh` requires the manifest and `icons/` and builds them into the
  binary. `make-zip.sh` ships their source.

### On the phone

- **iPhone and iPad:** the icon opens GobboNet full screen, without Safari's
  bars. iOS keeps a home-screen app's sign-in and storage apart from Safari's,
  so it signs in once more the first time. Its storage starts empty, and the
  page's existing rule fills it from the PC with no prompt (an empty browser
  and a server with history restore automatically).
- **Android:** over plain HTTP, Chrome makes a shortcut, which opens in Chrome
  as before, with its sign-in shared.

README.md (step 3, phone access) says how, and says to type `http://`.
TROUBLESHOOTING.md explains the extra sign-in.

## Checked here

- **Real Chromium against a built server** with a password set:
  - signed out on the login page, the browser finds `/manifest.webmanifest`
    and parses it with no errors: name *GobboNet*, standalone, start URL and
    scope the server's root, two icons on this server;
  - each icon and the manifest load signed out, with the right types;
  - `chat.html` is still a 401;
  - signed in, the chat page reports the same manifest, and the iPhone tags
    are there;
  - nothing was requested from anywhere but the server.
- **Not checked:** a real iPhone or Android phone. What iOS does with the
  standalone flag and the touch icon is Apple's, and wants one look on a
  device.

## Tests

- **`tests/test-home-screen.py`, 12 checks:**
  - the manifest's fields, and that it names nothing off the machine;
  - the icon sizes it gives are the PNGs' real sizes;
  - the icons are the package art;
  - the chat page and login page carry the same seven tags;
  - `publicAssets` is exactly the files they name;
  - staging and the ZIP include them.
- **`internal/server/homescreen_test.go`:**
  - the four files are served signed out, GET and HEAD, with their types;
  - a neighbouring file, `chat.html`, a POST, case and path tricks are still
    401;
  - in a staged build, the built-in page, manifest and login page name only
    public files, each the size it claims.
- **`internal/auth/login_test.go`:** the login page carries the tags, with
  absolute paths, in its `<head>`.
- **`tests/test-package-permissions.py`** now reads `stage-web.sh`'s list of
  inputs instead of keeping its own.
