# 1.7.8 — Banned Phrases that ban what they say

From issue #51, which asked for classifier-free guidance (CFG): a negative
prompt that steers the model away from what it describes. This is the
stand-in. Real CFG stays parked.

## Why not real CFG

CFG runs the model twice per token, once on the real prompt and once on a
negative one, and mixes the two full sets of logits. llama-server exposes only
the top few probabilities over HTTP, not the full logits. Doing it properly
means patching and shipping our own engine, which breaks the SHA-256 engine
pin. The two-sequence method from LlamaSharp, described in the issue, stays
the spec if that ever happens.

The commonest thing a negative prompt is used for is "stop writing X", and
llama-server *can* do that through `logit_bias`. Each card already had a
**Banned Phrases** list wired to it. It didn't work well, and the README
listed it as broken.

## What was wrong

`logit_bias` works on the model's **tokens**, not on words, and the list
ignored that:

- **It banned every token of everything on the list.** Each phrase was
  tokenized in eight case variants, and every resulting token was banned.
  - A phrase banned its everyday words with it: "shivers down her spine" put
    `" her"` and `" down"` on the list, everywhere.
  - A word that splits banned fragments that other words share. On Gemma 4
    and Llama 3, "Chuckle" at the start of a line is `[Ch][uckle]`, so
    `[Ch]` was banned for any word that starts a line with that piece. On
    Gemma 4, "SUDDENLY" is `[S][UD][DEN][LY]`, so the single letter `[S]` was
    banned too.
- **The far end of the slider was a strong nudge, not a ban.** -100 is a
  large penalty, but it is not "never". The label even said "near-ban".
- **It asked the engine eight times per phrase, on every message.**

## The change

All of it lives in `js/06-state-sync.js`. The request keeps its shape, a
`logit_bias` object keyed by token id, on both paths that send it: a new
message, and a regenerate (a reroll, an edited message, a scheduled message).

### Picking the word

- **A single word is banned as itself.**
- **A phrase is stopped at its rarest word**: its longest word of three
  letters or more that isn't one of about 100 of the commonest English words.
  - "a mix of" is stopped at *mix*.
  - If that word can't be banned safely (below), the next longest is tried,
    up to four. "shivers down her spine" is stopped at *spine*, because
    `" shivers"` is `[ sh][ivers]` on every tokenizer we checked.
  - The word is then banned everywhere, not only inside the phrase. The
    editor says so.
  - A phrase made only of common words ("of the") is not banned at all.
- Each word is tried in four spellings (as typed, lower, Title, UPPER), with
  and without a leading space.

### Picking the token

For each of those forms:

- **One token:** that token is banned.
- **Several tokens:** the leading piece is banned only when all of these
  hold. Otherwise that form is left alone.
  - It is at least four letters.
  - It is more than half the word: `" Chuck"` of "Chuckle" passes;
    `" tape"` of "tapestry", a word of its own, does not.
  - It isn't a common word itself: "whatever" is `[ what][ever]`, and
    `" what"` is never banned.
- **Space-only pieces in front are skipped.** SentencePiece models
  tokenize `" spine"` as `[ ][ sp][ine]`, and a bare space is never banned.
- `/tokenize` is asked for pieces (`with_pieces`). An older engine that
  sends ids only still gets single-token words banned. A word that splits is
  then left alone, because nothing about its pieces is known.
- `parse_special` is off. A list entry like `<|im_end|>` is read as plain
  text and can never become the end-of-turn token.

### Strength

- **-100 now bans outright.** It sends `false`, which llama-server reads as
  minus infinity, so the token is never produced. The slider's label says
  *Ban outright*.
- The strength is held to -1…-100. A positive value, from an imported card
  or a hand-edited backup, would *promote* the banned words; it is pulled
  back to -1.
- An unreadable value counts as the default, -20, not as a ban.

### Cost

- Tokenization is cached per model, so after the first message a turn makes
  **no extra requests**. A word added to the list costs only its own forms.
- Switching models asks again, because token ids belong to a model.
- A word's forms are tokenized together rather than one after another.
- If `/tokenize` isn't answering (no model loaded, or the engine starting
  up), the message goes out without `logit_bias`, with one console warning,
  and the next message tries again.

### CHECK

A **CHECK** button under the list asks the loaded model how each entry
tokenizes, and says in plain text what will be banned. This is its output
for five entries on the Gemma 4 tokenizer (the first line names the loaded
model's file; here, the test vocabulary's):

```
9 token(s) pushed down by -20 with ggml-vocab-gemma-4.gguf:
suddenly — banned: " suddenly", " Suddenly", "Suddenly". Left alone in 3 form(s) that split too finely, e.g. "suddenly".
Chuckle — banned: " Chuck" (start of the word), " chuckle". Left alone in 4 form(s) that split too finely, e.g. "Chuckle".
shivers down her spine — stopped at "spine" ("shivers" splits too finely), everywhere it appears, not only in this phrase: " spine", "spine", " Spine", " SPINE". Left alone in 2 form(s) that split too finely, e.g. "Spine".
of the — not banned: every word in it is too common to ban without changing ordinary text.
Elara — not banned: "Elara" splits into pieces too small or too common to ban safely with this model.
```

- `(start of the word)` marks a banned leading piece. Banning `" Chuck"` for
  "Chuckle" also bans the name Chuck, and the line shows it.
- The readout is written as text, never HTML: entries and pieces are user
  and model text.
- It is cleared when another card is opened. A result for a list that changed
  while it was checking is not shown.
- The console still logs a per-phrase breakdown on each message.

The hint under the list now says how it works: single words work best, a
phrase is stopped at its rarest word, common words are never banned.

### Docs

- README: removed from **Known bugs** and listed under **Recently fixed**. The
  feature list no longer calls it bugged.
- The built-in **Gobbonet Informant** no longer tells people to leave the list
  empty, and explains the list and the slider.
- AGENTS.md has a line on where the logic lives and why it is per token.

## What it can't do

- **It works per token, not per spelling.** A form that splits into small
  pieces can still be written. CHECK lists each form it leaves alone.
- **Older SentencePiece models (Llama 2, early Mistral) can ban very
  little.** Most words there start with a short piece. On the Llama 2 vocab,
  "smirk", "Chuckle" and "spine" have nothing safe to ban. CHECK says so.
- **Invented names usually can't be banned.** On every vocab checked, "Elara"
  and "Lyra" split into short pieces.
- Nothing here guides the model the way CFG would. It only removes tokens.

## Checked here

- **Real tokenizers.** The code was run against llama.cpp b10456's own
  tokenizer (`llama-tokenize`, the tokenizer code `/tokenize` uses) on four
  vocab-only GGUFs from llama.cpp's test models: Gemma 4, Llama 3 (BPE),
  Llama 2 (SentencePiece) and Qwen 3.5. The list was 19 common entries
  ("suddenly", "Chuckle", "smirked", "a mix of", "Elara", "tapestry",
  "barely above a whisper"…).
  - No common word or short fragment (`" the"`, `" of"`, `" a"`, `" what"`,
    `" Ch"`, `"Ch"`, `" S"`, `" El"`, `" Ly"`) was banned on any vocab,
    unless the list asked for that word by name.
  - That run is what found the two fixes in this change: falling back from
    "shivers", which splits on all four vocabs, and not banning `" tape"` for
    "tapestry".
- **Real Chromium against a built server**, with a stand-in llama-server
  whose `/tokenize` answers like b10456's:
  - CHECK listed what each entry bans, a phrase with markup was shown as
    text and never run, and SAVE stored the list and -100;
  - the chat request carried exactly the tokens CHECK listed, each `false`,
    and none of the fragments or common words;
  - a second message made no `/tokenize` requests and sent the same bans;
  - reopening the editor cleared the readout, and at -20 the same tokens went
    as -20;
  - with `/tokenize` answering 503, CHECK said it could not check, and the
    chat went through without `logit_bias`;
  - no page errors.
- **Not checked:** a real model generating with the bans on. llama-server's
  handling of `false` and of token-id keys was read in b10456's
  `server-schema.cpp`, not watched at work. The tokenizer checks used vocab
  files, not full models.

## Tests

`tests/test-banned-phrases.mjs`, 93 checks on the real code from `js/`,
against a stand-in `/tokenize` with a small vocabulary:

- picking the word: single words, phrases, common words, candidates in order,
  de-duplication, the 200-entry cap, a list that isn't text;
- picking the token: own token, a safe leading piece, never a short fragment,
  half a word, a common word or a space;
- what is sent: the strength, `false` at -100, out-of-range strengths;
- engines: SentencePiece-style spaces, ids without pieces, 503, unreachable;
- the cache;
- CHECK's readout: every kind of line, markup kept as text, a changed list;
- the page, both request sites, and the docs that called it broken.

Run against the code from before this change, 26 checks fail.
