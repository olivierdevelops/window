# `.capyx` — reactive VHCO apps in one file

A `.capyx` file is a **single-file reactive desktop app**. You write four kinds
of unit — dumb **components**, stateful **handlers**, I/O **capabilities**, and
an optional shared **orchestrator** — and the compiler
([`infra/capyx*.go`](../infra)) emits a `window.yaml` plus one
`static/index.html` with a fine-grained **signals runtime**
([`assets/capyx_runtime.js`](../assets/capyx_runtime.js)) inlined. No build
step, no node_modules, no framework download.

```sh
window app.capyx
```

The model is **VHCO** (View · Handler · Capability · Orchestrator): a strict
separation of concerns borrowed from the rest of this project, applied to a
reactive UI. Each layer may only touch what it owns.

---

## 1. The compile pipeline

```
app.capyx
   │  infra/capyx*.go  (pure text transform — no DOM, no eval)
   ▼
window.yaml  +  static/index.html
                   │  CAPYX signals runtime inlined
                   ▼
            window  →  native webview
```

`CompileCapyx(src, runtime)` returns the two files. The CLI dispatches any
path ending in `.capyx` to a temp build dir and hands it to the normal
`LoadApp` path, so a `.capyx` app runs exactly like any other `window` app.

---

## 2. The four layers (VHCO)

| Layer | Keyword | Owns | May touch |
|---|---|---|---|
| **View** | `component` | markup only | its props + injected handler `H` |
| **Handler** | `handler` | local state + `on` event methods | its own state + injected `$ports` |
| **Capability** | `capability` / `provide` | the outside world (I/O) | anything — this is the only impure layer |
| **Orchestrator** | `orchestrator` | shared reactive data | acts as an injectable capability |

The rules are mechanical:

- A **component** is a pure function of its inputs. It renders bindings; it
  never mutates state directly. It calls handler methods (`H.add()`).
- A **handler** holds state and mutates **only** its own fields and the ports
  injected into it. It performs no I/O itself — it asks a capability.
- A **capability** is the single seam where side effects live. `provide`
  supplies a concrete implementation for a declared `capability` shape.
- An **orchestrator** is a handler-like unit that owns data shared across
  panels. It is auto-registered as a capability under its own name, so any
  handler can inject it and read/write the one shared reactive store.

---

## 3. A 60-second tour — `counter.capyx`

```
handler Counter {
  state {
    count = 0
  }
  on inc { count = count + 1 }
  on dec { count = count - 1 }
}

component CounterView(H) {
  <div class="counter">
    <button { on:click = H.dec }>-</button>
    <span>{{ H.count }}</span>
    <button { on:click = H.inc }>+</button>
  </div>
}

mount CounterView with Counter
```

- `state { … }` declares reactive fields.
- `on inc { … }` is an event method; inside it bare `count` rewrites to
  `this.count`.
- In the view, `{{ H.count }}` is a reactive text binding and
  `{ on:click = H.inc }` wires the event.
- `mount CounterView with Counter` instantiates the handler, injects it as `H`,
  and appends the view to `#app`.

Clicking `+` re-runs **only** the one text effect bound to `count` — never the
whole component.

---

## 4. Fine-grained reactivity

The runtime is a tiny **signals** core: reactive `Proxy` objects `track` reads
during an effect and `trigger` the dependent effects on write.

- Every `{{ expr }}`, `class:x={…}`, `style:p={…}` and `attr={…}` compiles to
  its **own** effect that tracks exactly the fields it reads.
- `{#if}` / `{#elif}` / `{#else}` and `{#match}` compile to a **dynamic
  region** whose control effect tracks only the condition; the chosen branch's
  inner bindings own their own effects.
- `{#for}` compiles to a **keyed list**: the control effect tracks only list
  structure (length/keys), and each row's bindings track that row's fields.
  Mutating one item updates only that item's nodes.

The result is surgical DOM updates: in the `dashboard` demo, bumping one bar's
value re-renders that one bar and nothing else.

---

## 5. Syntax reference

### The three-brace rule

| You write | Meaning |
|---|---|
| `{{ expr }}` | reactive, escaped **text** binding |
| `{#for x in xs}` … `{/for}` | loop directive + closer |
| `{#if c}` … `{#elif c}` … `{#else}` … `{/if}` | conditional |
| `{#match v}` `{#case a}` … `{#default}` … `{/match}` | multi-way branch |
| `{ attr = expr }` | attribute / property binding |
| `{ on:event = H.method }` | event wiring |
| `{ bind:value = H.field }` | two-way input model |
| `{ class:name = expr }` | conditional class |
| `{ style:prop = expr }` | reactive inline style |

Inside a `{#for x in xs}` the loop var `x` and the index `$i` are in scope.

### Top-level constructs

| Construct | Purpose |
|---|---|
| `component Name(Props) { … }` | a view |
| `handler Name { state { … } on evt { … } }` | stateful unit |
| `capability Name { … }` | a declared I/O shape |
| `provide Name { … }` | a concrete capability implementation |
| `orchestrator Name { state { … } on evt { … } }` | shared store |
| `mount View with Handler` | instantiate + attach |
| `theme { token = value }` | tokens compiled to CSS variables |

### The handler mini-language

Inside `on`/`fn` blocks you write a small imperative dialect that compiles to
JavaScript:

- bare state field `count` → `this.count`
- injected port `store` → `this.$ports.store`
- `if` / `for` / `while` blocks use header + `end` (no braces required)
- arrow-function params and globals are left untouched by identifier rewriting

---

## 6. Dependency injection — capability / provide

```
capability Clock {
  now() -> number
}

provide Clock {
  now() { return Date.now() }
}

handler Timer {
  state { t = 0 }
  on tick { t = $ports.clock.now() }
}

mount TimerView with Timer(clock: Clock)
```

The handler names the port it needs (`clock`); the `mount` line injects the
provided `Clock`. The handler never imports `Date` — all impurity lives behind
the capability seam.

---

## 7. The orchestrator — one shared reactive store

An `orchestrator` owns data that two or more panels share. It is auto-registered
as a capability under its own name, so any handler can inject it:

```
orchestrator Store {
  state { count = 0 }
  on bump { count = count + 1 }
}

handler PanelA { on click { $ports.store.bump() } }
handler PanelB { }

mount PanelAView with PanelA(store: Store)
mount PanelBView with PanelB(store: Store)
```

Both handlers receive the **same** reactive `Store` instance. A click in panel A
calls `store.bump()`, and because both panels' `{{ store.count }}` bindings
track the same field, both update from `0` to `1` in lockstep. This is proven by
`TestSharedOrchestrator` in [`demos/capyx/generate_test.go`](../demos/capyx/generate_test.go).

---

## 8. The 24 demos

All demos live in [`demos/capyx/`](../demos/capyx) and run with
`window demos/capyx/<name>.capyx`. They are catalogued small → large in
[`demos/capyx/README.md`](../demos/capyx/README.md):

| # | Demo | Concepts |
|---|------|----------|
| 1 | `hello` | static component, no handler |
| 2 | `greeter` | `bind:value`, `{#if}/{#else}`, derived text |
| 3 | `toggle` | boolean state, `class:on={…}`, theme token |
| 4 | `counter` | events, `{#if}/{#elif}/{#else}` |
| 5 | `temperature` | one source field, multiple derived bindings |
| 6 | `tip_calculator` | two inputs, computed totals |
| 7 | `color_picker` | range inputs, reactive `style:` bindings |
| 8 | `list_basic` | `{#for}` over a primitive array, empty state |
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
| 20 | `calculator` | keypad `{#for}`, expression engine |
| 21 | `dashboard` | per-row effects (bump one bar, others untouched) |
| 22 | `two_lists` | one component mounted twice, two handlers |
| 23 | `notes` | capability / provide dependency injection |
| 24 | `orchestrator` | orchestrator as injectable capability; shared store |

### What the tests prove

Run `go test ./demos/capyx/`:

- **`TestCompileAll`** — all 24 compile and mount to non-empty DOM under a Node
  DOM shim (no browser).
- **`TestCounterReactivity`** — clicking `+`/`-` updates only the bound span.
- **`TestTodoListReactivity`** — typing + add grows the keyed list; remove
  shrinks it.
- **`TestSharedOrchestrator`** — one click in panel A updates **both** panels'
  shared count, proving the orchestrator is a single injected reactive store.

---

## See also

- [Authoring formats overview](./authoring-formats.md) — `.capyx` vs `.htmlx`
  vs `.cs` vs `.window`.
- [`demos/capyx/README.md`](../demos/capyx/README.md) — the demo catalogue.
- [`assets/capyx_runtime.js`](../assets/capyx_runtime.js) — the signals runtime.
