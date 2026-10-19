# 1.7.7 — The home screen keeps its scroll position

Roadmap item #2. Reported by **larckdrakengold**, with a fix: *"having
characters loaded in home page and scrolling down then letting the list stay
where it sat, will eventually snap back to the top by itself."* The reporter wrote the fix
with AI help and confirmed it worked on their install. It is kept, merged into
the existing code, and extended to one more list.

## What was wrong

The home screen is not drawn once. `js/24-boot.js` checks the connection every
five seconds, and while the home screen is showing it calls `render()` to
refresh the status pill. `render()` calls `renderMessages()`, which rebuilds the
page with

```js
container.innerHTML = renderLandingPage();
```

That replaces every element on the page, including the scroll boxes, and new
ones start at the top. "Eventually" meant within five seconds.

Reproduced in Chromium at 1440×900 with 14 extra characters and 12 schedules:

| List | Scrolled to | One tick later |
|---|---|---|
| YOUR CHARACTERS | 150 px | 0 |
| SCHEDULED TASKS | 100 px | 0 |

At phone width neither list scrolls on its own (`css/10-responsive.css`), so
phones were not affected.

## The change

In the no-thread branch of `renderMessages()` (`js/13-dashboard.js`), every
scroll position is read before the rebuild and put back after — the page's own,
and each list's:

- **As reported:** the page and both character lists.
- **Added:** the SCHEDULED TASKS list, which is its own 240 px scroller on
  desktop and had the same bug.
- **Only when the home screen was already showing.** Arriving from a thread
  does not carry that thread's position onto the home screen.

The lists are matched by order. That is safe because `renderLandingPage()`
always draws the same three — scheduled tasks, your characters, default
characters — whether or not they are empty. The test holds that order.

### Merged, not pasted

The reported block replaced the whole `if (!thread) { … }` block, but it ended
without the two lines that hide the HOME button and without the `return`.
Pasted as written, the function fell through into the thread-rendering code with
no thread, and threw on every five-second tick:

```
TypeError: Cannot read properties of null (reading 'name')
```

The merged version keeps those lines.

## Pinned by

**`tests/test-landing-scroll.mjs`** (new, 12 assertions). It drives the real
`renderMessages()` against a fake container whose `innerHTML` rebuilds every
scroll box at 0. The fake also resets the page's own scroll, which is stricter
than Chromium.

- **A.** A rebuild with no restore loses every position. This proves the fake
  can fail.
- **B.** Through repeated re-renders, the page and all three lists stay put.
- **C.** Coming home from a thread starts at the top.
- **D.** The home screen renders without throwing, the title is GOBBONET, and
  HOME is hidden.
- **E.** Source guards. `renderLandingPage()` draws the three scrollers in the
  order the restore assumes, and `js/24-boot.js` still re-renders on a timer.

**Checked against unfixed code:**

- The original `js/13-dashboard.js` fails four assertions: B's three, plus the
  restore guard in E.
- The reported block pasted literally throws the `TypeError` above.

## Verified in the browser

On a rebuilt binary, at 1440×900:

- Scheduled tasks held at 100 px through two ticks (11 s), and your characters
  at 150 px.
- Into a thread and back home works; the title and HOME button are right both
  ways.
- No page errors.

At 390×844 nothing scrolls on its own, and nothing changed.

## Noticed, not changed

Coming back from a thread, the home **page** itself can start partway down
(330 px in the test above) rather than at the top. That is the same with the
original code — checked by swapping the original `renderMessages()` back in — so
it is older than this fix and outside it. Not touched.
