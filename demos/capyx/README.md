# `.capyx` demos

A `.capyx` file is a **single-file reactive VHCO app**: dumb `component` views,
`handler` units (state + `on`-event mutations that only touch their own state
and injected ports), `capability` / `provide` boundaries (the only layer that
touches the outside world), an optional `orchestrator` that owns shared data,
and `mount` lines. The compiler ([`infra/capyx*.go`](../../infra)) emits a
`window.yaml` + a single `static/index.html` with a fine-grained **signals
runtime** ([`assets/capyx_runtime.js`](../../assets/capyx_runtime.js)) inlined.

Run any demo:

```sh
window demos/capyx/counter.capyx
```

Every demo here is compiled, mounted under a Node DOM shim, and (for several)
exercised for real reactivity by [`generate_test.go`](./generate_test.go):

```sh
go test ./demos/capyx/
```

## The 37 demos, small → large

| # | Demo | Concepts |
|---|------|----------|
| 1 | `hello` | static component, no handler |
| 2 | `greeter` | `bind:value`, `{#if}/{#else}`, derived text |
| 3 | `toggle` | boolean state, `class:on={…}`, theme token in CSS |
| 4 | `counter` | events, `{#if}/{#elif}/{#else}` |
| 5 | `temperature` | one source field, multiple derived `{{ }}` bindings |
| 6 | `tip_calculator` | two inputs, computed totals |
| 7 | `color_picker` | range inputs, reactive `style:` bindings |
| 8 | `list_basic` | `{#for}` over a primitive array, `{#else}` empty state |
| 9 | `star_rating` | `{#for}` + hover state, ternary glyphs |
| 10 | `tabs` | `{#match}/{#case}/{#default}` |
| 11 | `accordion` | `{#for}` with `$i`, per-row `{#if}` |
| 12 | `login_form` | multi-field validation, conditional message |
| 13 | `wordcount` | `bind:value` on `<textarea>`, live stats |
| 14 | `theme_demo` | `theme` tokens compiled to CSS variables |
| 15 | `stopwatch` | `setInterval` in a handler, start/stop/reset |
| 16 | `todo` | keyed list CRUD, toggle/remove, `.filter` |
| 17 | `shopping_cart` | two lists, `.reduce` totals, add/drop |
| 18 | `quiz` | indexed flow, scoring, restart |
| 19 | `kanban` | one list filtered into 3 columns, move between |
| 20 | `calculator` | keypad `{#for}`, `eval` expression engine |
| 21 | `dashboard` | per-row effects (bump one bar, others untouched) |
| 22 | `two_lists` | **Option-1**: one component mounted twice, two handlers |
| 23 | `notes` | **capability / provide** dependency injection |
| 24 | `orchestrator` | **orchestrator** as injectable capability; shared store across two panels |
| 25 | `control` | reactive `{#for}` nav, `{#if}` toggle, `{#match}` tabs + role cards — the runtime twin of `demos/htmlx/control.htmlx` |

### Reactive showcase (input-driven, each with a `.capytest`)

These twelve demonstrate components reacting live to user input. Every one ships
a matching `<name>.capytest` suite that drives the real signals runtime
headlessly:

| # | Demo | Category | Concepts |
|---|------|----------|----------|
| 26 | `live_preview` | writing text | `bind:value` textarea → live preview + counts, `{#if}/{#else}` |
| 27 | `username_picker` | writing text | live slug + nested `{#if}` availability check |
| 28 | `text_transformer` | writing text | one input → many derived `{{ }}` transforms |
| 29 | `checklist` | todo app | add/toggle/remove, `{#for}…{#else}`, completion banner, `class:` |
| 30 | `tag_input` | todo app | add + de-dupe + remove tags, reactive count |
| 31 | `product_tabs` | tabs | `{#match}` panels with `class:active` |
| 32 | `setup_wizard` | tabs / steps | multi-step next/back, conditional summary |
| 33 | `password_strength` | conditional | nested `{#if}` tiers + reactive meter width |
| 34 | `signup_form` | conditional | live inline validation as you type |
| 35 | `shipping_options` | conditional | `<select>` + toggle drive `{#match}` / `{#if}` |
| 36 | `search_filter` | conditional | live `{#for}`+`{#if}` filter with empty state |
| 37 | `bmi_check` | conditional | number inputs → computed value + nested `{#if}` |

Run any suite headlessly:

```sh
window test demos/capyx/checklist.capytest
```

## What the tests prove

- **`TestCompileAll`** — all 25 compile and mount to non-empty DOM.
- **`TestCounterReactivity`** — clicking `+`/`-` updates only the bound span.
- **`TestTodoListReactivity`** — typing + add grows the keyed list; remove
  shrinks it.
- **`TestSharedOrchestrator`** — one click in panel A updates **both** panels'
  shared count (`0,0` → `1,1`), proving the orchestrator is a single reactive
  store injected into two handlers.
- **`TestAllCapyTestSuites`** — every `*.capytest` suite in this folder (the
  input-driven showcase plus `counter`/`notes`) parses, compiles in harness
  mode, and passes under the Node DOM shim — proving the demos truly react to
  typing, selecting, toggling and clicking.
