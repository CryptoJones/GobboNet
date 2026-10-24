# 1.7.8 — Characters read their .md memory files back

From issue #41: *"I would just love it if the characters could read md files. It
can write them but not read."*

## What was happening

When a character wrote a ` ```file:notes.md ` block, the reply got a **Save**
button that downloaded the file, and that was the end of it. Nothing could
give a file back to the character. The nearest thing was pasting the text
into the RAG Storybook, and that went wrong: the Storybook reads a `# heading`
line as a new entity, and nearly every Markdown file starts with one, so a
pasted note came out as a row of tag-less entities.

## The change

A card now has **memory files**: `.md` or `.txt` notes the character reads back
through retrieval.

### Getting a file in

- **Save to memory**, on any ` ```file:*.md ` (or `.txt`) block. The file goes
  to the character who wrote the message: the same speaker the avatar shows,
  so in a cast thread it lands with whoever wrote it. A block in your own
  message goes to the thread's character.
  - A new file is saved at once, with a toast.
  - Saving a file whose name is already there asks before replacing it, so a
    character can keep one file up to date.
- **+ IMPORT .MD** in the card editor, under the Storybook. Each file is
  listed with its size and how many parts it splits into, with **Download**
  and **Remove**.
  - Imports and removals wait for **SAVE**; **CANCEL** drops them. The list
    sits in a hidden textarea, so the unsaved-changes guard sees it like any
    other field.

### How it is read

- Each file is split at its own headings, outside code fences, then packed
  into parts of about 1,200 characters. Each part keeps its heading path.
- The parts join Retriever A's pool beside the Storybook's prose. Only the
  parts that match the recent conversation are injected, under a label:
  `[Memory file: quest-log.md › Quest log › The mill]`.
- One short system line names the card's memory files and says how to
  rewrite one: the whole file again, in a block with the same name. It comes
  after the retrieved parts and before the writing style.
- A card with no memory files gets neither, so its prompt is unchanged to
  the byte.
- **Without the embedding model**, memory parts are matched by shared words.
  That covers Nomic not being installed, the embed server not answering, and
  Retriever A switched off.
  - A part needs two distinct words in common with the recent messages, so
    one common word in passing does not pull in a section.
  - Storybook prose does not take this path; it works exactly as before.
- Telemetry records memory hits as kind `memory`, with trigger `semantic` or
  `keyword`, and counts each card's memory parts.

### Limits and safety

- Up to 20 files per character, 100,000 characters each, 400,000 in total.
  Over a limit, the message says which limit.
- **Names are names, never paths:**
  - the last path segment only;
  - no control or reserved characters, and no leading dots;
  - an `.md`, `.markdown` or `.txt` extension (`.md` is added when there is
    none);
  - at most 100 characters.
- Text only: a file with NUL bytes is refused.
- Every way in uses the same check (`memoryPutFile`): the editor, *Save to
  memory*, and card imports. Every read goes through `cardMemoryFiles`, which
  skips malformed entries from backups or other devices instead of failing a
  turn.
- Editor rows are built as DOM nodes with `textContent`, never from HTML. A
  file name comes from a file someone picked, or from model output.

### Where they go

With the rest of the card:

- **Card exports:** PNG V3 and the JPG/ZIP share, as
  `extensions.gobbonet.memoryFiles`, name and text. Other apps ignore it.
  Exporting from an open editor includes its unsaved memory changes, as it
  does for every other field.
- **Card imports**, with the checks above.
- **Backups:** character and full.
- **Device sync.**

Embedding happens on this machine's own embed server, and no new route or
request was added.

## Checked here

Real Chromium against a built server, with a stand-in llama-server that
records each request, run twice: with a stand-in embedding server, and with
none.

- **Import:** a `quest-log.md` imported in the editor is listed in 2 parts,
  counts as an unsaved change, and reaches the card only on SAVE.
- **Read back:** asked about "the silver key… near the old mill", the request
  that reached the model held the mill section under its label, followed by
  the files index.
  - With the embed server it was found by meaning, and ranked ahead of the
    other section.
  - Without one it was found by shared words, and the other section stayed
    out.
- **Save to memory:**
  - the model wrote a ` ```file:notes.md ` block, and *Save to memory* saved
    it with no question;
  - a second version asked before replacing, then replaced it;
  - the next turn read version 2 back.
- **Sync:** the server's synced copy of the card held both files.
- **PNG round trip:** a V3 PNG export, re-imported, came back as a new card
  with both files.
- No page errors.

Not checked: a real model and the real Nomic embeddings. The stand-in embed
server ranks by shared words, which exercises the ranking path but not
Nomic's sense of meaning.

## Tests

`tests/test-memory-files.mjs`, 71 checks on the real code from `js/`:

- names, limits and validation;
- chunking (heading paths, fences, long paragraphs, packing);
- retrieval:
  - semantic, with a bag-of-words stand-in for the embed server;
  - shared words with the server down, and with Retriever A off;
  - nothing with retrieval off;
  - budget eviction;
  - the Storybook unchanged;
- the prompt (label, index placement, nothing added for a card without
  files);
- the editor (import, refusals, a hostile file name, remove, download,
  SAVE);
- *Save to memory* (speaker in a cast thread, replace declined and accepted,
  the user's own message, an open editor kept in step, over-limit, a deleted
  character);
- export and import (round trip, a hostile card's paths, binary and
  oversized files left behind);
- the page markup.
