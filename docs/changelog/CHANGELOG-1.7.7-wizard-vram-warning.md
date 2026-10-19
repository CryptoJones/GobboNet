# 1.7.7 — The Linux setup wizard warns before a model too big for the GPU

Roadmap item #8a.

## What was there

Windows has asked for a long time, in two places:

- **`launch.bat`:** "Heads up: this model wants about N GB of GPU memory, but
  only M GB was detected… Download it anyway?"
- **The Windows installer:** the same question, as a dialog.

The Linux setup wizard only printed "suits N GB VRAM or more" under each model.
The Go side had no GPU memory figure to compare against, so nothing asked.

## The change

**The figure comes from the bundled engine.** `llama-server --list-devices`
asks every backend the engine was built with, through the same drivers it will
run the model on. It needs no extra tools and loads no model. The output format
was checked in llama.cpp's source at the pinned tag b10456
(`common/arg.cpp:1139-1162`):

```
Available devices:
  CUDA0: NVIDIA GeForce RTX 4060 (8188 MiB, 7900 MiB free)
```

It prints `(none)` when there is no GPU.

### Reading the list

The rules are `hardware-probe.ps1`'s, which has met real machines. They are
ported to Go in `internal/engine/devices.go`.

- **Names can contain their own parentheses**, as in
  `Intel(R) Graphics (ARL) (11577 MiB, …)`. The memory group is anchored to the
  end of the line.
- **An integrated GPU is not VRAM.** Its "total" is a share of system memory, so
  it is left out of the figure. It is recognised:
  - by Vulkan's `uma: 1` line, which is authoritative when present;
  - otherwise by name (UHD, Iris, Radeon 780M, Arc 140V…);
  - or by holding most of system memory.
- **The share-of-memory test never applies to known discrete names** (NVIDIA,
  Radeon RX, Arc A/B). Without that exception, an 8 GB card in an 8 GB machine
  would be called integrated. That is a common budget build, and the sandbox
  this was written on is one.
- **Capacity is rounded to the card's label**, so a 16 GB card's 16368 MiB is
  16, not 15. That is `ConvertTo-NominalGB`, case for case.
- **The largest dedicated card is the figure**, not the sum. A sum would be
  right for two discrete cards, but one misread integrated GPU would then hide
  a real shortfall. An extra warning costs a click; a missing one costs a 20 GB
  download that crawls.

### Running the probe

**It starts with the wizard**, so the answer is ready by the model step,
several screens later. It is cached per engine file and redone if the engine
is replaced, the way the supervisor caches `--help`.

**It gives up after 20 seconds.** The Windows probe has seen a stuck graphics
driver hang device listing, and the answer then is "not known", never a wizard
left waiting.

### Asking

**`GET /api/gpu`** on the setup server returns the figure, the card's name and
the devices. The Linux packages' Python setup shell proxies it alongside
`/api/download`.

**The wizard asks before downloading**, with the browser's own `confirm()`,
like every other confirmation in GobboNet:

> Gemma 4 26B-A4B MoE wants about 18 GB of GPU memory, but only 8 GB was
> detected (NVIDIA GeForce RTX 4060).
>
> It may run far more slowly than a model that fits: whatever does not fit on
> the GPU runs from system memory instead.
>
> Download it anyway?

- **Declined:** nothing downloads, and Download can be pressed again.
- **Accepted:** the download goes ahead as before.

**The first sentence and the question are `launch.bat`'s. The middle sentence
is not.** `launch.bat` says a large shortfall makes llama-server "fail to start
at all rather than fall back to system RAM". That is true of its legacy fixed
layer count. GobboNet's server places layers automatically (`gpu_layers = -1`)
and does fall back to system memory, so it is slow rather than failing. The
roadmap asked for "may run slowly, not will not run".

**It never blocks on a failed probe.** That is `launch.bat`'s rule. The wizard
asks nothing when there is no honest figure:

- no engine;
- no GPU listed;
- only an integrated GPU;
- the probe failed or timed out;
- an older setup server without the route.

**Both wizards carry it:**

- the one the Linux packages serve (`installer-linux/wizard.html`, copied into
  `linux-amd64/`);
- the one built into the program for `gobbonet setup` run on its own
  (`internal/setup/wizard.html`).

## Visual

None beyond the browser's own confirm dialog. The model list's "suits N GB VRAM
or more" line is unchanged.

## Pinned by

**`internal/engine/devices_test.go`** (new):

- **Parsing:**
  - names with parentheses;
  - CRLF line endings;
  - the Vulkan `uma` flag beating the name, either way;
  - CPU entries, `(none)` and nonsense listing nothing.
- **Integrated or discrete**, for a table of real names, including the budget
  builds above.
- **Nominal GB**, case for case with the Windows probe.
- **The probe:**
  - runs a stand-in engine;
  - caches its answer;
  - redoes it when the engine is replaced;
  - times out on a hang;
  - reports a missing engine or one without `--list-devices`.

**`internal/setup/gpu_test.go`** (new):

- `/api/gpu` reports a dedicated card.
- Every unknown case answers `known: false`, with the reason.
- Both copies of the Python shell proxy the route.

**`tests/test-wizard-vram.mjs`** (new, 46 assertions, on each wizard's real
code):

- **When it asks:** a model too big asks, in exactly the wording above.
- **When it doesn't:**
  - a model that fits;
  - no figure;
  - a failed request;
  - an older server;
  - a model with no minimum.
- **The answers:** declining sends nothing and leaves Download clickable, and
  accepting downloads that model.
- **The copies:** `linux-amd64/wizard.html` is still an exact copy.

## Verified through the real Linux launcher

`gobbonet-launch` was run from a built install folder, which starts the Python
setup shell, which starts `gobbonet setup`. A stand-in engine printed the
b10456 format. The wizard was clicked through in a browser to the model step:

| Stand-in engine | Model | What happened |
|---|---|---|
| RTX 4060, 8188 MiB | Gemma 4 26B (18 GB), declined | The dialog above. The download stayed idle, and Download was clickable again. |
| RTX 4060, 8188 MiB | Gemma 4 E4B (8 GB) | No question; the download started. |
| RTX 4060, 8188 MiB | Qwen3.6 35B (24 GB), accepted | One question; the download started. |
| `(none)` | Gemma 4 26B | No question; the download started. |

Downloads that started then failed only because the sandbox can't reach
Hugging Face.

No page errors.

## Not tested here

**Real GPUs.** Which figure a real NVIDIA, AMD or Intel card and driver report
through the bundled engine needs a real machine. That matters most for
integrated GPUs on Vulkan, where the `uma` flag is what decides.
