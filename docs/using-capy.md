# Using Capy with `window`

> A hands-on guide to authoring `window` apps from a single source file.
> You define a tiny app language once in a `.capy` library; one source then
> generates the **frontend** (HTML/CSS/JS) and the **`window.yaml`** config.
> Backend logic — when an app needs any — is a **Go function in the
> `window` process itself**, never a separate subprocess.
>
> For the conceptual case see [capy-integration.md](capy-integration.md);
> for every feature with demos see [capy-full-apps.md](capy-full-apps.md).
> The authoritative grammar/API is the
> [Capy integration guide](https://github.com/luowensheng/capy).

---

## Table of contents

1. [The model: one binary, no sidecars](#1-the-model-one-binary-no-sidecars)
2. [Install Capy](#2-install-capy)
3. [What Capy generates](#3-what-capy-generates)
4. [Building UI](#4-building-ui)
5. [Event handling](#5-event-handling)
6. [State management](#6-state-management)
7. [When you need real backend logic — in-process Go](#7-when-you-need-real-backend-logic-in-process-go)
8. [A complete worked example](#8-a-complete-worked-example)
9. [Embedding the generator (`window --capy`)](#9-embedding-the-generator-window---capy)
10. [The grammar is the contract](#10-the-grammar-is-the-contract)
11. [For AI agents](#11-for-ai-agents)

---

## 1. The model: one binary, no sidecars

A `window` app is normally a frontend, a `window.yaml`, and *maybe* a
backend. Capy lets you author the first two from **one source file** in a
language you design — and it deliberately does **not** generate a separate
backend process.

> **Principle: minimal management.** A separate backend (a Python script, a
> Node process, a socket) is another runtime to install, ship, and keep
> alive. We avoid it. If an app genuinely needs server-side logic, that
> logic is a **Go function compiled into the `window` binary** — one
> process, one language, nothing extra to manage. Anything missing is
> implemented in the main Go process, never in a sidecar.

So there are two tiers, and most apps stop at the first:

| Tier | Where the logic lives | When |
|------|-----------------------|------|
| **Frontend-only** | generated JS in the page (state + events) + `NATIVE.*` | the large majority of apps |
| **In-process Go** | a Go handler inside the `window` process | filesystem/DB/compute that must be server-side |

There is no third "spin up Python" tier. Capy generates the UI; the Go
binary holds any handlers.

---

## 2. Install Capy

```sh
# CLI — for a build step or your dev loop
go install github.com/luowensheng/capy/cmd/capy@latest
capy version

# Go library — to embed generation in the window binary (see §9)
go get github.com/luowensheng/capy@latest
```

Confirm it works:

```sh
capy init demo && cd demo
capy run lib.capy script.capy   # should print rendered output
```

---

## 3. What Capy generates

A Capy library declares `file "path":` blocks; each renders from a shared
`context` the statements populate. For a `window` app the targets are the
frontend and the config — **not** a backend script:

```
app.window  ──capy──▶  window.yaml          (the manifest)
                       static/index.html     (markup)
                       static/app.js         (state + events + native setup)
```

That's a complete, runnable app. No `run_backend_script`, no `main.py`, no
socket. You run it with `window window.yaml`.

The base library everything builds on:

```
# window.capy (core)
extension yaml
comments
    line "#"
end

context
    title    "App"
    width    480
    height   640
    body     []        # HTML fragments, in order
    state    []        # {name, value} reactive store fields
    events   []        # {selector, type, action} DOM event bindings
    setup    []        # raw JS init lines
    features []        # native_features to enable
    handlers []        # in-process Go handler names (tier 2)
end

function app
    arg literal "app"
    arg capture title string
    arg capture w int default "480"
    arg capture h int default "640"
    block_closer end
    set context.title title
    set context.width w
    set context.height h
end
function end
end
```

The full file blocks are in [capy-full-apps.md](capy-full-apps.md#appendix-the-complete-windowcapy);
the sections below show the statements that build UI, events, and state.

---

## 4. Building UI

UI statements append HTML fragments to `context.body`. Each one is a small
component verb; the library author picks the vocabulary.

```
function heading
    arg literal "heading"
    arg capture text string
    append context.body "<h1>${escapeHtml (unquote text)}</h1>"
end

function field
    arg literal "field"
    arg capture id ident
    arg literal "placeholder"
    arg capture ph string
    append context.body "<input id=\"${id}\" placeholder=${toQuoted (unquote ph)}>"
end

function text
    arg literal "text"
    arg capture bind ident          # bind a store field into the DOM
    append context.body "<span data-bind=\"${bind}\"></span>"
end
```

**Source**
```
app "Hello" 320 160
    heading "Welcome"
    field name placeholder "Your name"
    text greeting
end
```

**Generated `static/index.html` (body)**
```html
<h1>Welcome</h1>
<input id="name" placeholder="Your name">
<span data-bind="greeting"></span>
```

`escapeHtml (unquote …)` strips source quotes and HTML-escapes — your XSS guard
at the template boundary. `data-bind` ties an element to a store field,
which the next two sections wire up.

---

## 5. Event handling

There are two kinds of events, both abstracted by statements:

**DOM events** — a user interaction triggers an *action* (a named bit of
behavior). The library records `{selector, type, action}`; the `app.js`
block emits an `addEventListener` that runs the action.

```
function on
    arg literal "on"
    arg capture sel string          # a CSS selector or #id
    arg capture type ident          # click, input, keydown, …
    arg literal "do"
    arg capture action ident
    append context.events {selector: sel, type: type, action: action}
end
```

**Push events** — a server-side `NATIVE`/Go side can dispatch a custom
event the page listens for (e.g. a file-watch notification, or a value from
an in-process handler). A `text … bound to a push` is just a `data-bind`
updated from the event payload.

**Source**
```
app "Counter" 280 160
    text count
    on "#inc" click do increment
    on "#dec" click do decrement
```
…plus buttons declared by a `button` verb. The actions `increment` /
`decrement` are defined as **state transitions** (next section) — no
backend involved.

**Generated `static/app.js`**
```js
function action(name){ return ACTIONS[name] || (()=>{}); }
document.querySelector("#inc").addEventListener("click", action("increment"));
document.querySelector("#dec").addEventListener("click", action("decrement"));
```

The wiring is generated; what an action *does* is defined once (in JS for
state, or in Go for backend logic) and never duplicated.

---

## 6. State management

Most `window` apps are stateful UIs with **no backend at all** — the state
lives in a tiny generated store in `app.js`. Declare store fields and the
transitions that mutate them; bound `text`/inputs re-render automatically.

```
function state
    arg literal "state"
    arg capture name  ident
    arg literal "="
    arg capture value any
    append context.state {name: name, value: value}
end

# action <name> set <field> = <expr>   (expr is JS evaluated over `state`)
function action
    arg literal "action"
    arg capture name  ident
    arg literal "set"
    arg capture field ident
    arg literal "="
    arg capture expr  tail
    append context.transitions {name: name, field: field, expr: expr}
end
```

**Source**
```
app "Counter" 280 160
    state count = 0
    text count
    button "+" id inc
    button "-" id dec
    on "#inc" click do increment
    on "#dec" click do decrement
    action increment set count = state.count + 1
    action decrement set count = state.count - 1
end
```

**Generated `static/app.js`**
```js
const state = { count: 0 };
function render(){
  document.querySelectorAll("[data-bind]").forEach(el => {
    el.textContent = state[el.dataset.bind];
  });
}
function setState(patch){ Object.assign(state, patch); render(); }

const ACTIONS = {
  increment: () => setState({ count: state.count + 1 }),
  decrement: () => setState({ count: state.count - 1 }),
};

document.querySelector("#inc").addEventListener("click", ACTIONS.increment);
document.querySelector("#dec").addEventListener("click", ACTIONS.decrement);
render();
```

**Result:** a fully working reactive counter — a store, bindings, actions,
and event wiring — all generated, all in the page. **Zero backend, zero
extra processes.** This is the common case and the whole "minimal
management" payoff: a stateful desktop app that is just one `window` binary
opening generated files.

---

## 7. When you need real backend logic — in-process Go

Some logic can't (or shouldn't) live in the page: reading a database,
heavy computation, touching the network with secrets. For those, the
handler is a **Go function in the `window` process** — registered in-process,
reachable from the frontend, with no subprocess and no socket of its own.

`window` exposes a small in-process handler registry for this (this is the
"implement what's missing in the main Go process" part — you add a Go
function, not a new service):

```go
// in the window process — a Capy-generated registration
app.Handle("search", func(req map[string]any) (map[string]any, error) {
    rows := db.Query(req["q"].(string))   // real Go, in-process
    return map[string]any{"rows": rows}, nil
})
```

A Capy `handler` statement declares one of these; the frontend calls it
through the existing `BACKEND.call` API, which now dispatches **in-process**
to the registry instead of over a socket:

```
function handler
    arg literal "handler"
    arg capture name ident
    append context.handlers name
end

function call
    arg literal "call"
    arg capture action ident
    arg literal "into"
    arg capture bind ident
    append context.events {selector: "", type: "call", action: action, bind: bind}
end
```

**Source**
```
app "Search" 600 480
    field q placeholder "Query…"
    button "Search" id go
    text rows
    handler search                 # → a Go function in the window process
    on "#go" click do run_search
end
```

**Generated**
- `static/app.js` calls the in-process handler:
  ```js
  ACTIONS.run_search = () =>
    BACKEND.call("search", { q: document.getElementById("q").value },
      ({ data }) => setState({ rows: data.rows }));
  ```
- `app_handlers.go` (compiled into the binary) — the stub you fill with Go:
  ```go
  func registerAppHandlers(app *AppHandlers) {
      app.Handle("search", func(req map[string]any) (map[string]any, error) {
          // TODO: your Go logic, in-process
          return map[string]any{"rows": []any{}}, nil
      })
  }
  ```

There is no `run_backend_script` in the generated `window.yaml`. The handler
ships **inside** the binary. One language (Go), one process. If a feature is
missing to support a handler, you add it to the Go process here — never by
introducing a separate backend.

> Native device APIs (`fs`, `os`, `dialogs`, `canvas`, `camera`, `mic`,
> `speech`, `screen`, `input`) are already in-process Go bindings exposed as
> `NATIVE.*`. Reach for those first — e.g. read a file with
> `NATIVE.fs.readFile` straight from the page before writing a Go handler.

---

## 8. A complete worked example

A note-taker: a reactive list in the frontend, persisted to disk with the
in-process `fs` capability — **no backend process anywhere.**

**Source — `notes.window`**
```
app "Notes" 460 560
    field draft placeholder "New note…"
    button "Add" id add
    list notes
    on "#add" click do add_note
    persist notes to "/tmp/notes.json"
end
```

What Capy generates:

- `window.yaml` with `native_features: [fs]` (inferred from `persist`).
- `static/index.html` with the input, button, and a `<ul data-bind="notes">`.
- `static/app.js` with a `{ notes: [] }` store, an `add_note` action that
  pushes the draft and re-renders the list, and load/save via
  `NATIVE.fs.readFile`/`writeFile` on the given path.

Two commands to run it:

```sh
capy run window.capy notes.window
window window.yaml
```

A persistent note app — UI, events, state, and disk persistence — with one
binary and no sidecar.

---

## 9. Embedding the generator (`window --capy`)

To make it one command (and to compile in any Go handlers from §7), embed
the generator. This respects `window`'s layering: `infra/` owns the
third-party dependency and file I/O, `appio/` adds the flag.

```go
// infra/capy_codegen.go
package infra

import "github.com/luowensheng/capy"

// GenerateApp turns one .window source into the app's files
// (window.yaml, static/*, and app_handlers.go for in-process Go handlers).
func GenerateApp(libPath, scriptSrc string) (map[string]string, error) {
	lib, err := capy.NewLibraryFromFile(libPath)
	if err != nil {
		return nil, err
	}
	_, files, err := lib.RunMulti(scriptSrc)   // map[path]content
	return files, err
}
```

```go
// appio/cli.go — add a --capy mode
//   window --capy window.capy notes.window
//   → generate the files, compile any app_handlers.go into this process, run window.yaml
```

`NewLibraryFromFile` compiles the grammar once; `RunMulti` is re-entrant.
The generated `window.yaml` flows through the existing
`appio/config_loader.go` unchanged. The generated `app_handlers.go` is the
*only* backend artifact, and it lives in the **same** Go module — no
separate runtime ever enters the picture.

---

## 10. The grammar is the contract

Because Capy has no built-in grammar, the only valid source is what your
library accepts — the parser *is* the contract. Two payoffs:

- **No orphan calls.** Make `on "#go" click do run_search` a compile error
  unless `run_search` is a defined action or handler. A dead button becomes
  a build failure, not a silent no-op.
- **Typed captures.** A `type` with a `base`/`pattern`/`options` rejects bad
  input at parse time with a caret-pointed `line:col` — e.g. a `Feature`
  enum so `native blockchain` can't slip through.

---

## 11. For AI agents

This shape is ideal for AI-authored apps: the agent emits a few dozen tokens
of your DSL, and the library deterministically expands them into the app.
Crucially, the output is **a frontend plus optional in-process Go handlers**
— there's no separate process for a hallucination to misuse, and:

- **Parser-as-sandbox.** Anything outside the grammar is a parse error
  before a file is written — the model can't emit a stray `<script>` or a
  `subprocess.run`.
- **No host access during generation.** After `NewLibrary` the host is
  `NoOpHost` (no env/args/file reads). Opt into `OSHost` only for trusted
  libraries.

A typical loop:

```
1. lib.Introspect()       → the verbs the agent may use
2. draft notes.window     → ~30 tokens of your DSL (the model)
3. capy check window.capy → parse error? feed it back
4. lib.RunMulti(source)   → window.yaml + static/ (+ app_handlers.go)
5. window window.yaml     → the user sees their app — one binary, no sidecar
```

---

*See also: [capy-full-apps.md](capy-full-apps.md) (every feature, with
demos) and [capy-integration.md](capy-integration.md) (the rationale).*
