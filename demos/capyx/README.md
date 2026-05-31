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

## The 24 demos, small → large

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

## What the tests prove

- **`TestCompileAll`** — all 24 compile and mount to non-empty DOM.
- **`TestCounterReactivity`** — clicking `+`/`-` updates only the bound span.
- **`TestTodoListReactivity`** — typing + add grows the keyed list; remove
  shrinks it.
- **`TestSharedOrchestrator`** — one click in panel A updates **both** panels'
  shared count (`0,0` → `1,1`), proving the orchestrator is a single reactive
  store injected into two handlers.
