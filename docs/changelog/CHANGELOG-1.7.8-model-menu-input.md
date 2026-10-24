# 1.7.8 — The first-run model menu asks again instead of closing

From issue #63 (first-install findings, Solteris-Dev), item 1.

## What was happening

`launch.bat` reads the model menu answer into `MODEL_CHOICE` and tests it
against "1" through "10". Anything else fell through every branch to the [11]
"add my own .gguf" path, which ends at `:fatal`.

The prompt shown just before the menu is "Download llama.cpp now? (Y/N)", so
`y` is the natural thing to type. It closed the window with "[FATAL] See the
error above", with no error above it. The same happened to `0`, `12`, ` 3` with
a stray space, or ENTER when there was no recommended pick.

## The change

Right after the read, the answer is checked against 1 to 11. Anything else
prints a one-line hint and asks again, without reprinting the menu, so the hint
stays on screen. ENTER still takes the recommended pick, and [11] still means
"skip".

The retry is capped at 5. With no console behind it (input at end of file),
`set /p` returns nothing forever, and an uncapped loop would spin where the old
code exited. After the cap it takes the same [11] path it always did. The
typed text is never echoed back.

## Tests

`tests/test-model-menu-input.py` lifts the block out of the real `launch.bat`
and runs it through a small interpreter for exactly the cmd subset it uses
(`set /p` keeps its real behaviour: an empty line leaves the variable alone).
50 checks, covering the reporter's `y`, every valid choice, ENTER with and
without a recommendation, a dozen bad inputs, end of input, CRLF line endings,
and parentheses inside the `( )` blocks. Not run in a real Windows console here;
that is the one thing left to check by hand.
