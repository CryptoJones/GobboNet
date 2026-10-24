# 1.7.8 — "Lucia" in people's chats: a lore prompt that names no one, and a box to edit it

> Users report the name Lucia leaking into their chats which makes sense
> because its the example name in the RAG. We need an agnostic, simple, and
> short prompt the AI can follow and then a checkmark near the RAG field that
> lets folks edit that prompt.

## What was happening

When Lore System compresses older messages, it sends them to the model with
instructions to write one short entry (`summarizeForLore`,
`js/07-prompt.js`). Those instructions showed the format with a worked
example: a vampire story with two characters, Lucia and Vasch.

Small models copy examples. An entry such as "Lucia fed on Vasch." went
into the thread's lore. Lore is sent back with every later turn as
established fact, so from then on the character spoke of Lucia as if she
belonged to the story.

The old instructions were 177 words. The example was most of what a small
model took from them.

## The change

### A short default that names no one

`LORE_DEFAULT_PROMPT` in `js/07-prompt.js`, 127 words in full:

```text
You keep a short log of what happens in a story or conversation. You are
given the entries recorded so far and the newest messages. Write ONE new
entry about the newest messages only.

Rules:
1. One sentence, under 20 words, past tense.
2. Only what is new. Never repeat a recorded entry, the setting, or who anyone is.
3. Concrete: who did what, and what was decided, found out or changed. No mood or scenery.
4. Use only names that appear in the messages. Never invent a name.
5. Write the sentence alone: no label, bullet or quotes.
6. If nothing important happened, write exactly: SKIP

An entry looks like this, with the <...> parts taken from the messages:
<who> <did what> <to whom or where>.
```

- It has no names, no setting and no genre, so there is nothing for a model to
  copy into someone else's story. It works for a role-play, a coding chat or
  anything else.
- It still shows the shape of an entry, but as a template whose parts the
  model has to fill from the messages.
- Rule 4 says outright to use only names from the messages.
- The closing line of the request still repeats the core rule ("One sentence.
  Nothing else."), because small models weight the end of a prompt most.
- A reply that is the template itself, such as `<who> <did what> ...`, is not
  stored. The pass counts as failed, and a gap note is written instead, just
  as when the model returns nothing. Storing it would have made it permanent
  "fact" in the lore.

### A box to edit it, beside the RAG Storybook

The card editor has a new checkbox between **Starting Lore** and **RAG
Storybook**: **Edit the lore summary prompt**.

- **Unticked (the default):** the built-in prompt is used.
- **Ticked:** a box opens, filled with the built-in prompt to edit, along with
  a **RESET TO DEFAULT** button.
  - `{{char}}` and `{{user}}` work in it.
  - It is limited to 4,000 characters, because the prompt is sent with every
    compression pass and takes room from the messages being summarised.
- **Unticking hides the box and keeps what was typed,** so ticking again
  brings it back.
- **A prompt saved unchanged from the built-in one is stored as nothing.** That
  card then follows the built-in prompt if a later release improves it.
- **Whatever the prompt says, the rules around it stay the same:** each reply
  is cut to two sentences and added as one line, and `SKIP` adds nothing.
  - These rules are what keep the lore from growing without limit.
  - The box's hint says so, and it warns against example names.
- **With a custom prompt, the closing line of the request is neutral:** "Write
  the new entry now." The built-in rules are not added after the user's own.

The card fields are `loreCustomPrompt` and `lorePrompt`, saved like any
other card field and synced with it.

- **Export:** a card that uses its own prompt exports it as
  `extensions.gobbonet.lorePrompt`.
- **Import:** reads it back, ticked, capped at 4,000 characters with NUL
  characters removed. It is plain text, visible in the editor, and sent only to
  the local model. A card from another app arrives on the built-in prompt.

### Lore that already has the leak

This change stops new leaks. It does not touch lore that was already written:
nothing in the lore is deleted on the user's behalf. To remove "Lucia" lines
from a thread:

1. Click the lore chip in the chat header.
2. Choose **EDIT**.
3. Delete those lines, then save.

The character stops seeing them from the next message.

## Checked here

- **What reaches the model, old code against new**, for one compression pass
  of the same chat:
  - the old request contained "Lucia" and "Vasch", with a 177-word system
    prompt;
  - the new request names neither, with a 127-word system prompt.
- **Real Chromium**, against a built server and a fake llama-server that
  records each request:
  - The box is there, unticked, between Starting Lore and RAG Storybook.
  - Ticking it shows the built-in prompt, and counts as an unsaved change.
  - SAVE stores it, and it survives a reload.
  - A chat long enough to compress sent the card's own prompt as the lore
    pass's system message, with `{{char}}` and `{{user}}` filled in and
    without the built-in rules added.
  - Unticked again, the next pass sent the built-in prompt.
  - No request said Lucia or Vasch, and there were no page errors.

**Not checked:**

- **A real model.** No model could be downloaded here, so how well a given
  small model follows the new wording has not been measured. The leak itself
  is gone either way, because the name is no longer in the prompt.
- **Windows.**

## Tests

`tests/test-lore-prompt.mjs` has 70 checks and runs the whole of
`js/07-prompt.js` plus the editor and import/export code. It covers:

- the default: no names, no setting, length, the six rules, the template;
- that the default is what reaches the model, and that nothing in the request
  says Lucia or Vasch;
- the card's own prompt: when it is used, the macros, trimming, the 4,000
  limit, and the closing line;
- template echoes not being stored, and `SKIP`;
- no Lucia or Vasch left in any string in `js/` or in `chat.html`;
- the editor box: placement, ticking, unticking, reset, save, and a page
  without the box;
- export and import, including bad values.

On the previous commit the suite fails. `loreSystemPrompt` does not exist
there, and its request contains both names.
