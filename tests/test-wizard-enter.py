"""Enter submits each setup-wizard step that has a text field (issue #63).

Both wizards -- the Go one embedded in `gobbonet setup` and the Linux packages'
Python-served one -- had their fields outside any <form>, with Continue wired
to a click handler and no key handler. Pressing Enter in the password field
(or the folder, port or server-address field) did nothing.

The fix makes each such step's card a <form class="card" novalidate>. A form
turns Enter in a field into a click on its submit button, which is the step's
Continue button, so the existing click handler runs unchanged; the script
cancels the submission itself so nothing navigates. `novalidate` keeps the
browser's own URL check from popping up over the server-address field after
the handler has already run.

This pins that structure statically, the way the other wizard tests do (no
browser here). It was also checked in headless Chromium: Enter posts once and
advances on every step listed below, clicking still posts once, and the page
never navigates.

Run: python3 tests/test-wizard-enter.py
"""
import pathlib, re, sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
WIZARDS = {
    "internal/setup/wizard.html": ["s1", "s2"],
    "installer-linux/wizard.html": ["location", "port", "s1", "s2"],
}
passed = failed = 0


def ok(cond, label, extra=""):
    global passed, failed
    if cond:
        passed += 1
        print("  ✓ " + label)
    else:
        failed += 1
        print("  ✗ " + label + ("\n      " + str(extra) if extra else ""))


def sections(html):
    for m in re.finditer(r'<section class="step[^"]*" id="([^"]+)">(.*?)</section>', html, re.S):
        yield m.group(1), m.group(2)


for rel, expected in WIZARDS.items():
    html = (ROOT / rel).read_text()
    print(f"\n=== {rel} ===")
    with_inputs = [sid for sid, body in sections(html) if "<input" in body]
    ok(sorted(with_inputs) == sorted(expected),
       "the steps with text fields are the ones this test covers: " + ", ".join(expected), with_inputs)
    for sid, body in sections(html):
        if sid not in with_inputs:
            continue
        ok(body.lstrip().startswith('<form class="card" novalidate>') and body.rstrip().endswith("</form>"),
           f"step {sid}: its card is a <form novalidate>, so Enter reaches Continue", body[:80])
        buttons = re.findall(r"<button\b[^>]*>", body)
        ok(len(buttons) == 1 and "type=" not in buttons[0],
           f"step {sid}: exactly one button, a submit button by default, so Enter has one target", buttons)
    ok(re.search(r'querySelectorAll\("form\.card"\)\.forEach\(function \(f\) \{\s*'
                 r'f\.addEventListener\("submit", function \(e\) \{ e\.preventDefault\(\); \}\);', html),
       "the script cancels the form submission, so the page never navigates")
    markup = re.sub(r"<script>.*?</script>", "", html, flags=re.S)
    ok("<form" not in markup.replace('<form class="card" novalidate>', ""),
       "no other <form> in the page markup")

zip_copy = ROOT / "linux-amd64" / "wizard.html"
ok(zip_copy.read_bytes() == (ROOT / "installer-linux" / "wizard.html").read_bytes(),
   "linux-amd64/wizard.html (the ZIP's copy) matches installer-linux/wizard.html")

print(f"\n{passed} passed, {failed} failed")
sys.exit(1 if failed else 0)
