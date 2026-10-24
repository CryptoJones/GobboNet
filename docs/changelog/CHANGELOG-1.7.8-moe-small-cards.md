# 1.7.8 — MoE models offered to 8–12 GB cards

Follow-up to issue #65 and PR #66 (Dae).

## What was happening

PR #66 measured the three Mixture-of-Experts models in the catalogue on an
8 GB RTX 4070 Laptop with 32 GB of RAM, once automatic placement left them
2 GB of headroom:

| Model | Download | Generation |
|---|---|---|
| Qwen3.6 35B-A3B | ~22 GB | 21.6 t/s |
| Gemma 4 26B-A4B | ~16 GB | 21.3 t/s |
| gpt-oss 20B | ~12 GB | 20.3 t/s |

Every place that offers a model still knew only one rule: the whole model has
to fit on the card. So on that machine:

- the `launch.bat` menu marked all three `[ needs ~N GB VRAM - will be slow ]`;
- choosing one brought up the "Heads up" warning;
- the Linux setup wizard asked "Download it anyway?";
- the Add a Model window said "suits 24 GB VRAM or more" and nothing else.

## The change

A MoE model now carries a second rule beside `min_vram`: **it also fits a card
of at least 8 GB when system RAM covers its `min_ram`.**

| Model | Whole model on the card | Or: card + system RAM |
|---|---|---|
| Gemma 4 26B-A4B | 18 GB | 8 GB + 24 GB RAM |
| Qwen3.6 35B-A3B | 24 GB | 8 GB + 32 GB RAM |
| gpt-oss 20B | 14 GB | 8 GB + 16 GB RAM |

The RAM figure is the download, less the ~6 GB an 8 GB card holds, plus ~10 GB
for Windows, the browser and the KV cache, rounded up to a common RAM size.
The 32 GB for Qwen3.6 is the machine it was measured on.

- **The `launch.bat` menu** (`hw-recommend.ps1`) marks a model that fits this
  way `[ fits: MoE, uses system RAM ]`, so it doesn't look unmarked by
  mistake. The figures are `$moeVram` and `$moeRam`.
- **The download warning** (`launch.bat`) doesn't fire when the rule holds.
  When it doesn't hold, the warning adds what would make the model fit and how
  much RAM this PC reports. The figures are `MOE_MIN_VRAM` and `PICK_MOE_RAM`.
- **`models.ini`** carries `moe_min_vram` and `min_ram` for those three models,
  written by `gen-catalog.py`. Like the VRAM gate, the generator refuses to
  build when the two Windows tables disagree.
- **The Linux setup wizard** gets system RAM from `/api/gpu` (`ram_gb`, read
  from `/proc/meminfo` and rounded to the installed size). It doesn't ask when
  the rule holds. Its list says "suits 24 GB VRAM or more, or 8 GB with 32 GB
  system RAM".
- **The Add a Model window** says the same in its description.

### Unchanged

- **The recommendation.** An 8 GB card is still recommended Gemma 4 E4B, and a
  12 GB card Qwen3.5 9B. The MoE models are offered, not pushed: this follows
  one machine's measurements, and the safe default stays the safe default.
- **The whole-model `min_vram` figures**, and the size filter that uses them
  (Qwen3.6 is still Heavy).
- **Below 8 GB, and with no GPU,** nothing changes.
- **Unknown RAM** is 0, which confirms nothing. An older setup server that
  sends no `ram_gb` gets the warning as before.

## The live catalogue

The catalogue served from goblincorps.com doesn't carry the allowance yet. Until
it does, the program takes it from the bundled `models.ini` by file name, the
same way it already borrows the coding/roleplay buckets. Where the live list
states its own figures, they win.

When the live list is next published, it should carry:

- `moe_min_vram_gb` and `min_ram_gb` for the three models (optional fields;
  absent means dense);
- `moe` in gpt-oss 20B's tags. It is a Mixture-of-Experts model (32 experts,
  4 active per token), and PR #66's headroom rule already treats it as one, but
  the live list tags only Gemma and Qwen.

## Also fixed

`gen-catalog.py` read a commented-out row of a PowerShell table as a real row,
and a `::` comment quoting a `PICK_MIN` line as a real line. Both are skipped
now, for the new tables and for the existing `$min` / `PICK_MIN` check.

## Tests

- **`tests/test-moe-offer.py`, 16 checks:**
  - the generator's output, and its refusal on four kinds of drift;
  - `hw-recommend.ps1` run under `pwsh` for 8 GB at 16/24/31/32 GB RAM, 12 GB,
    6 GB, 24 GB, no GPU and unknown RAM;
  - the `launch.bat` gate and warning, still CRLF, with no parentheses in its
    echo lines.
- **`tests/test-wizard-vram.mjs`, +30:** all three wizard copies on PR #66's
  machine, too little RAM, a 6 GB card, an older server and unknown RAM.
- **`tests/test-catalog-filters.mjs`, +2:** the Add a Model description.
- **Go:**
  - `internal/catalog/moe_test.go`: `Entry.Fits`, the `.ini` and JSON fields,
    the shipped figures, and borrowing from the bundled list;
  - `internal/setup/ram_test.go`: `/proc/meminfo` rounding, `ram_gb` and the
    state fields;
  - `internal/server/models_test.go`: the catalogue JSON.

Every new check fails on the 1.7.7 files. The Windows menu and warning were run
under PowerShell 7 on Linux, against the real script. They still want one look
on a real Windows machine with an 8 GB card.
