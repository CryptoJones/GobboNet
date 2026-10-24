# 1.7.8 — Prompt Order: lorebook snippets next to the newest message (optional)

This came from reports of power-supply trouble. The aim was less work per
message, at the lowest cost.

The proposal:

> Right now the prompt is stacked like this:
>
> 1. Character and lore
> 2. Lorebook snippets (these change from message to message)
> 3. The whole conversation
> 4. Your new message
>
> Because item 2 sits near the top and keeps changing, the AI rereads the
> entire conversation below it nearly every time. The fix moves the snippets
> down to just above your new message:
>
> 1. Character and lore
> 2. The whole conversation
> 3. Lorebook snippets
> 4. Your new message

The request was to make it optional, as a toggle in CONFIG, and to note the
plan to put the whole prompt order in the user's hands later.

## What was happening

llama-server keeps the last prompt it read in its cache. For the next request
it reuses that prompt up to the first token that differs, and reads
everything after that point again. That reading (prompt processing) is the
heavy, all-cores or full-GPU burst before a reply starts.

`buildContextMessages` (`js/08-rag.js`) puts the retrieved snippets among the
system blocks at the top. The snippets come from the RAG Storybook and memory
files, and they change whenever the scene does. Each time they changed, the
reuse stopped at the top, and the whole conversation was read again.

## The change

### CONFIG > Prompt Order

A new **Prompt Order** box sits above Performance, with one switch: **Put
lorebook snippets just before my newest message**.

- **Off (the default):** the order is as before.
  - This holds for settings saved before this option and for any value other
    than `true`.
  - Off sends exactly what the previous build sent: the same messages, byte
    for byte, compared for a card with a storybook.
- **On:** character and lore → the conversation → lorebook snippets → your
  new message.
  - The snippets go ahead of the response direction and the persona, which
    already sat beside the newest message, so those stay closest to it.
  - A chat template that allows only one system message gets the snippets as
    the `[System: ...]` note on the newest message. The persona and response
    direction already arrive that way.
- **The same snippets reach the model either way,** with the same token
  count; only their place changes. A model may weigh them differently near the
  end, which is why the switch is off by default and the box says to try both.
- **It only matters for a character with a RAG Storybook or memory files.** A
  turn with no snippets is identical either way.
- **Edge cases:**
  - A chat with no user message yet keeps the snippets at the top.
  - A reroll puts them before the user message being answered.
  - DeepSeek and Qwen models fold the system block into the first message;
    that block now stays the same from turn to turn.
- **The box ends with the plan:** "Planned: arranging the whole prompt
  yourself. This switch moves the one part that changes most." The same note
  is in the code, above the assembly.

### Saving and presets

- **Where it is stored:** the setting is `retrievalNearEnd` in
  `DEFAULT_SETTINGS` and is saved with CONFIG's SAVE.
- **Presetting it for every browser:** add `retrieval_near_end = true` under
  `[ui]` in `config.toml`.
- **The retrieval record:** each turn's record, kept in the page and never
  sent anywhere, notes which order was used (`config.near_end`).

### One wording change

The memory-files note told the model that a matching part "appears above as
[Memory file: ...]". With the switch on, the part is below the note, so the
wording is now "is included as [Memory file: ...]", which holds either way.

## Measured on the real engine

There was no real model to download here, so `make_tiny_gguf.py` built a tiny
llama-architecture GGUF with random weights.

- **Engine:** the ZIP's own pinned llama-server (b10456), with the flags the
  supervisor uses: `--parallel 1 --jinja --reasoning-format auto`, prompt
  cache on by default.
- **Setup:** a real gobbonet server and real Chromium. The character had a
  tag-fired storybook of six towns, and the chat had 40 messages already.
- **The test:** five more messages, each naming a different town, so the
  snippets changed every time. This was run once with Prompt Order off and
  once with it on, switched in CONFIG.
- **The figures:** read from the engine's own log of tokens read per request.
  The model writes nonsense, but how much of each prompt it reads is the
  engine's real behaviour.

| Message | Off: read / prompt (tokens) | Off: time | On: read / prompt (tokens) | On: time |
| --- | --- | --- | --- | --- |
| 2 | 15,199 / 15,451 | 7.6 s | 449 / 15,461 | 0.4 s |
| 3 | 15,538 / 15,695 | 6.2 s | 541 / 15,705 | 1.0 s |
| 4 | 15,788 / 15,945 | 7.1 s | 640 / 15,955 | 0.5 s |
| 5 | 15,928 / 16,085 | 8.2 s | 633 / 16,095 | 0.5 s |
| **Total** | **62,453 of 63,176 (99%)** | **29.1 s** | **2,263 of 63,216 (4%)** | **2.5 s** |

With the switch on, the engine read about 28 times fewer tokens. On a real
model on a GPU, that is the burst of full load before each reply, shrunk to
the last exchange.

The first message in a new chat is left out of the table, because it reads
everything either way. The page's own retrieval records confirm every message
carried snippets and that they changed each time.

**Not checked:**

- **A real model on a GPU.** Power draw was not measured. The table is tokens
  and CPU time on a tiny model; on a GPU the saving is the same share of
  tokens, but not the same seconds.
- **Reply quality with the snippets at the end,** on any real model.
- **Windows.**

## Not changed, and worth knowing

- **A lore compression pass resets the cache once.** It sends a different
  prompt on the same engine slot. The engine's own prompt cache in RAM
  (`--cache-ram`, on by default in this build) can restore the chat prompt
  afterwards. That was not measured here.
- **Anything else that changes at the top costs the same full reread,** for
  example a lore summary that has just been rewritten. A future "arrange the
  whole prompt" feature should keep blocks that change every message at the
  end.

## Tests

`tests/test-prompt-order.mjs` has 42 checks. It runs the real
`buildContextMessages` and the real normaliser:

- **Off:** absent, `false`, or any non-boolean value gives the old order.
- **On:** the full order, the top unchanged, one system message, the
  `[System: ...]` note on the newest message, and alternation that strict
  templates accept.
- **The rendered prompt over four turns:** the share read again is 98% with
  the switch off and 2% with it on. The reuse stops exactly at the previous
  message.
- **Edge cases:** no user message, a reroll, Qwen-style folding, and a turn
  without snippets.
- **Nothing lost or added:** the same snippets, the same tokens and the same
  words either way, including with the real tag retriever.
- **Wiring:** the CONFIG box, opening and saving, the `[ui]` preset key, the
  retrieval record, and the memory-files wording.

On the previous commit 14 of the 42 checks fail.
