# 1.7.7 — Thinking on or off, per character

Roadmap item #9: *"Can we make thinking optional on COT models given the current
setup?"* Yes, mostly. The maintainer chose a character card option.

## What was there

Nothing turned thinking off. `thinkingFormat` only tells the reply parser how to
split reasoning from the answer for display.

## What the engine already supports

The pinned llama.cpp (b10456) runs most models with Jinja templates and
`--reasoning-format auto`. Its server reads `chat_template_kwargs` from each
request (`tools/server/server-common.cpp:1279-1293`, read in its source at that
tag) and passes `enable_thinking` to the template. Two details from that code
shaped the change:

- **It must be a JSON boolean.** The string `"false"` is rejected with
  `invalid type for "enable_thinking" (expected boolean, got string)`.
- **Templates that don't support it** simply never use the value.

The jobs path forwards the request body byte for byte, so the server needed no
change.

## The change

**A checkbox on every character**, in the card's *Context & Reply Length* box,
built exactly like *Cap this character's reply length*:

> ☑ LET THIS CHARACTER THINK (IF THE MODEL SUPPORTS SWITCHING IT OFF)
> Off asks the model to answer straight away, without its reasoning step. Models
> built to always reason, such as DeepSeek-R1 or gpt-oss, may think anyway.

- **On** (the default) sends nothing extra. A thinking character's requests are
  byte-identical to before.
- **Off** adds `chat_template_kwargs: { enable_thinking: false }` to that
  character's requests: new messages, regenerate, auto-continue and scheduled
  sends alike. All of them go through the two request builders in
  `js/10-chat.js`.
- **The field is `thinkingEnabled`**, read as `!== false` (`cardThinkingEnabled`,
  `js/06-state-sync.js`). Cards from older installs, other apps or imports, which
  have no field, think as they always did. New cards and the default card start
  with it on.
- **It travels with the card**, exactly as the reply cap does: in backups,
  single-character export and device sync. Not in the V3 PNG, which by design
  carries only V3 fields.

## Honest limits

- **Always-reasoning models ignore it.** DeepSeek-R1 distills always reason;
  gpt-oss can be turned down but not off. The hint says so.
- **Untested on real models.** Which catalogue models honour it — Qwen3.5 9B,
  Qwen3.6 35B-A3B, DeepSeek-R1 8B, gpt-oss 20B, Gemma 4 — needs a real install;
  the authoring sandbox has no GPU or engine.

## Pinned by

**`tests/test-thinking-switch.mjs`** (new, 17 assertions):

- **Off** sends exactly `{"chat_template_kwargs":{"enable_thinking":false}}`,
  with a boolean.
- **On** sends nothing, and a request is byte-identical to one without the
  switch.
- **No field, `undefined` or `0`** all think; only an explicit `false` turns it
  off.
- **Wiring:** both request bodies carry it; the row is in the right box, built
  like the reply-cap row; opening and saving a card round-trip it; new and
  default cards start on.

Against the original files, it fails at once: the switch does not exist there.

## Verified in the browser

A rebuilt server, with a stand-in model server recording every request body:

- **Through the real card editor:** with the box unticked and saved, a new
  message and a regenerate both sent `{"enable_thinking": false}`. Ticked again,
  the next request sent no such field.
- **The setting persisted** through a page reload and server sync. The editor
  shows the stored value, and a card with no field opens ticked.
- **Layout:** at 1440 and 390 wide, the row matches the reply-cap row in the
  same box.
- **No page errors.**

## Not changed

- **Lore compression passes** still think as before. Sending the same switch
  there was offered as a follow-on and not taken up.
