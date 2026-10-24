package auth

// LoginPage renders the sign-in screen. Self-contained and themed to match the
// app's palette, so the first thing a user sees on a new device doesn't look
// like a server error.
//
// The same page is returned for an unauthenticated HTML navigation (with a 401
// status), which is what makes a browser hitting any deep link land on the
// login form instead of a JSON blob.
func LoginPage(failed bool) string {
	errBlock := ""
	if failed {
		errBlock = `<p class="err">Wrong password. Try again.</p>`
	}
	return `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gobbonet -- sign in</title>
` + HomeScreenTags + `
<style>
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
         background:#0a0e0a; color:#7fd97f; font-family:ui-monospace,Menlo,Consolas,monospace; }
  .box { width:min(92vw,360px); padding:28px; border:1px solid #1f3a1f; border-radius:10px; background:#0d140d; }
  h1 { font-size:18px; margin:0 0 4px; color:#9cffa0; letter-spacing:1px; }
  p.sub { margin:0 0 20px; font-size:12px; color:#4f7d4f; }
  label { display:block; font-size:12px; margin-bottom:6px; color:#6fbf6f; }
  input { width:100%; box-sizing:border-box; padding:11px 12px; font-size:15px; background:#060a06;
          border:1px solid #2a4a2a; border-radius:6px; color:#cfeccf; outline:none; }
  input:focus { border-color:#4f9d4f; }
  button { margin-top:16px; width:100%; padding:11px; font-size:14px; font-weight:600; cursor:pointer;
           background:#1c3a1c; color:#bdf5bd; border:1px solid #3a6a3a; border-radius:6px; }
  button:hover { background:#234a23; }
  .err { color:#ff8a8a; font-size:12px; margin:14px 0 0; }
  .note { margin:18px 0 0; padding-top:14px; border-top:1px solid #1f3a1f;
          font-size:11px; line-height:1.5; color:#5a7d5a; }
  .note + .note { margin-top:8px; padding-top:0; border-top:0; }
</style></head>
<body><form class="box" method="POST" action="/login">
  <h1>gobbonet</h1>
  <p class="sub">This server is password-protected. Sign in to continue.</p>
  <label for="pw">Password</label>
  <input type="password" id="pw" name="password" autofocus autocomplete="current-password">
  <button type="submit">Sign in</button>
  ` + errBlock + `
  <p class="note">This connection is over your local network in plain text
  (not encrypted). It's fine for a home network you trust. Avoid using it on
  shared or public Wi-Fi, and don't reuse a password that matters elsewhere.</p>
  <p class="note">A sign-in lasts until GobboNet restarts, 12 hours at most by
  default, and only for this device and browser. A phone that changes network
  address, or a browser that updates, is asked to sign in again.</p>
</form></body></html>
`
}

// HomeScreenTags are chat.html's home-screen tags (issue #63), with absolute
// paths: this page is served at /login and at any deep link a signed-out
// browser opens. A phone usually meets GobboNet here first, so "Add to Home
// Screen" from the sign-in page has to find the same name and icon. The files
// they name are served before sign-in (server.go, publicAssets).
const HomeScreenTags = `<link rel="manifest" href="/manifest.webmanifest">
<link rel="apple-touch-icon" href="/icons/gobbonet-180.png">
<meta name="mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-title" content="GobboNet">
<meta name="apple-mobile-web-app-status-bar-style" content="black">
<meta name="theme-color" content="#020205">`

// TooManyAttemptsPage is shown when the login rate limiter trips on a browser
// navigation, so a locked-out user gets an explanation rather than raw JSON.
func TooManyAttemptsPage() string {
	return `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gobbonet -- too many attempts</title>
<style>
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
         background:#0a0e0a; color:#7fd97f; font-family:ui-monospace,Menlo,Consolas,monospace; }
  .box { width:min(92vw,360px); padding:28px; border:1px solid #3a1f1f; border-radius:10px; background:#140d0d; }
  h1 { font-size:18px; margin:0 0 10px; color:#ff8a8a; letter-spacing:1px; }
  p { margin:0; font-size:12px; line-height:1.6; color:#c08a8a; }
</style></head>
<body><div class="box">
  <h1>Too many attempts</h1>
  <p>Wait about a minute, then reload this page and try again.</p>
</div></body></html>
`
}
