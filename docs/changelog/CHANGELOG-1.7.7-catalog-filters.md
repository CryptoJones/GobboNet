# 1.7.7 — Add a Model: find a model by type and size

Added at the maintainer's request on 2026-10-12, once in-app downloads worked:
"Right now we're just throwing a giant list at the users."

## What it does

Two rows of buttons above the list in *CONFIG → Add a Model → BROWSE
CATALOGUE*:

```
Type:  [ALL MODELS] [CODING] [ROLEPLAY] [JACK OF ALL TRADES]
Size:  [ANY SIZE] [LIGHT] [MEDIUM] [HEAVY]
```

- **Type** is what the model is for. A model can be in more than one type.
  *Jack of all trades* means the maintainer's "MoE-like or non-specialty
  models".
- **Size** is how much it needs to run, from the model's GPU memory figure (the
  same "suits N GB VRAM or more" each row already shows). The maintainer chose
  the 8 / 16 split:

  | Size | GPU memory |
  |---|---|
  | Light | 8 GB or less |
  | Medium | 9 to 16 GB |
  | Heavy | over 16 GB |

  Without a GPU, the figure is roughly the system memory the model wants
  instead.
- **ALL MODELS and ANY SIZE show everything** — the "see all models" option. The
  window opens on them.
- **Each row now starts with its size**, so the class is visible in the full
  list too: "Light · 5.4 GB download · suits 6 GB VRAM or more · …".
- **The choice lasts while the page is open**, so closing and reopening the
  window keeps it.
- **When nothing matches**, the list says so and how to see more.

## Which model is where

As agreed with the maintainer:

| Model | Coding | Roleplay | Jack of all trades |
|---|---|---|---|
| Gemma 4 E4B IT | | | ✓ |
| Llama 3.2 3B Instruct | | | ✓ |
| Mistral 7B v0.3 | | ✓ | |
| Qwen3.5 9B | ✓ | | |
| Gemma 4 26B-A4B MoE | | ✓ | ✓ |
| Qwen3.6 35B-A3B MoE | ✓ | | ✓ |
| DeepSeek-R1 8B | ✓ | | |
| gpt-oss 20B | ✓ | | ✓ |
| Command R 7B (12-2024) | | | ✓ |
| Command R 35B (08-2024) | | ✓ | |

## Where the tags live

**In the catalogue, so a new model arrives already sorted.** The catalogue's
existing `tags` list carries them, as `coding`, `roleplay` and
`jack-of-all-trades`.

**Jack of all trades has its own tag rather than reusing `general`.** The live
list already puts `general` on nearly every model as a description; reusing it
would put Mistral 7B, a roleplay pick, under *Jack of all trades* too.

### The bundled list

`launch.bat` stays the single source of truth for the model set:

- **Each download block gains a `set "MODEL_TAGS=…"` line**, the way `DL_SHA256`
  works. `launch.bat` itself never reads it.
- **`installer/gen-catalog.py` writes it into `models.ini` as `tags=`:**
  - It stops on a misspelled bucket.
  - It only warns about a model with no tags, which still shows under ALL
    MODELS.
- **`internal/catalog` reads `tags=` from `models.ini`.**
- **`installer/models.ini` is regenerated, and `linux-amd64/models.ini`** is its
  copy.

### The live list on goblincorps.com

**The live list needs the same tags.** It is published separately, so this
build can't change it. Until it carries them, each model **borrows its buckets
from the bundled list**, matched by file. So the filters work today, against
the live list as it stands. A model the live list gives any bucket keeps its
own: the live list wins wherever it has an opinion.

To publish them, add these to each model's `tags` in
`gobbonet_model_list.json`, keeping the tags already there:

```
Gemma 4 E4B IT            "jack-of-all-trades"
Llama 3.2 3B Instruct     "jack-of-all-trades"
Mistral 7B v0.3           "roleplay"
Qwen3.5 9B                "coding"
Gemma 4 26B-A4B MoE       "roleplay", "jack-of-all-trades"
Qwen3.6 35B-A3B MoE       "coding", "jack-of-all-trades"
DeepSeek-R1 8B            "coding"
gpt-oss 20B               "coding", "jack-of-all-trades"
Command R 7B (12-2024)    "jack-of-all-trades"
Command R 35B (08-2024)   "roleplay"
```

**A model added later** to either list without a bucket shows under ALL MODELS
only, until it is tagged.

## Noticed, not changed

**The live list and the bundled list disagree about GPU memory.** For example,
Gemma 4 E4B is 6 GB live and 8 GB bundled, and Gemma 4 26B MoE is 16 live and
18 bundled.

Size follows whichever list is in use. So a PC that can't reach goblincorps.com
sorts a few models one class heavier:

| Model | Live list | Bundled list |
|---|---|---|
| Mistral 7B, Qwen3.5 9B, Command R 7B | Light | Medium |
| Gemma 4 26B MoE | Medium | Heavy |

Which figures are right is the maintainer's call. The bundled ones come from
`hw-recommend.ps1`, which the Windows menu also uses.

## Visual

Nothing new was designed:

- **The buttons** are the existing `btn btn-sm`.
- **The chosen button** is filled with `btn-fill`, the look SAVE & RESTART
  MODEL has beside its neighbours. `aria-pressed` says the same to a screen
  reader.
- **The "Type:" / "Size:" labels** are Auto-Continue's inline label style.
- **The rows** are Performance's wrapping button row.

Checked at 1440 and 390 wide. On a phone, *Jack of all trades* wraps onto a
second line.

## Pinned by

**`internal/catalog/buckets_test.go`** (new):

- **The bundled list** carries exactly the agreed buckets.
- **The live list**, as published today, borrows all ten and keeps its own
  tags.
- **A live bucket wins** over a bundled one.
- **Through `Fetch`:** a cached live list comes back filled, with a note
  saying so.
- **Tag parsing.**

**`internal/server/models_test.go`:** `/catalog.json` sends each model's
`buckets`, always as a list. `general` is not a bucket.

**`tests/test-catalog-buckets.py`** (new):

- **Every model in `launch.bat`** has a `MODEL_TAGS` made only of buckets.
- **The generator and the Go code** agree on the buckets.
- **The committed `models.ini`**, and its `linux-amd64` copy, is exactly the
  generator's output.
- **A misspelled bucket** stops the build.
- **A model with none** is only warned about.

**`tests/test-catalog-filters.mjs`** (new, 39 assertions, on the real list
code):

- **The 8 / 16 boundaries.**
- **Every type and size**, and combinations of the two.
- **An untagged model** is under ALL MODELS only.
- **A model with no figure** gets no size class.
- **The empty result.**
- **The choice surviving a reopen.**
- **An empty catalogue** hides the filters.
- **The markup:** the eight buttons, the type buttons matching the server's
  buckets, and the borrowed styles.

## Verified in the browser

On a password-protected server, signed in, through *CONFIG → Add a Model*:

- **The bundled list:** every type and size listed exactly the models in the
  tables above.
- **The live list**, from a cached copy of what goblincorps.com serves today:
  - its buckets were borrowed;
  - its sizes matched the split agreed with the maintainer: five Light, three
    Medium, two Heavy;
  - the server's log said "took the … buckets for 10 model(s) from the bundled
    list".
- **The choice** survived closing and reopening the window.
- **No page errors** at 1440 or 390 wide.
