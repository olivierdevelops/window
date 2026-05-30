# Building UI + Backend in One Language with Capy

`webview_gui` already lets you ship a native desktop app from three
independent artifacts: an HTML/CSS/JS frontend, a `window.yaml` config,
and a backend (Python subprocess, WASM module, or none). The catch is
that those three artifacts are written in three different languages,
wired together by hand-maintained string keys — a `BACKEND.call("greet", …)`
in JS must match an `@app.handle("greet")` in Python, which must match a
`run_backend_script:` line in YAML. Nothing checks that they agree.

[**Capy**](https://github.com/olivierdevelops/capy) is a transpiler engine
with **zero default grammar**. You define a tiny source language in a
`.capy` library, and Capy gives you a code generator that emits *any*
textual target — HTML, JS, Python, Go, YAML, all at once. This document
shows how to put one Capy library in front of `webview_gui` so a single
source file generates the frontend, the backend, *and* the config — with
the call/handler contract enforced by the grammar instead of by hope.

It also covers why this shape is ideal for **AI agents**: an agent emits
~50 lines of a domain language you control, Capy expands it into the
800+ lines of a working app, and anything outside the grammar is rejected
before a single file is written.

---

## Table of contents

1. [The wiring problem Capy solves](#the-wiring-problem)
2. [How Capy fits the VHCO architecture](#how-capy-fits)
3. [A 10-minute integration: the `app.capy` language](#ten-minute)
4. [One source → frontend + backend + config](#one-source-many-files)
5. [Enforcing the call/handler contract in the grammar](#contract)
6. [Targeting every run mode from one DSL](#run-modes)
7. [Other interesting things Capy unlocks](#interesting)
8. [AI agents that build apps](#ai-agents)
9. [Embedding the Capy compiler in `window`](#embedding)
10. [Suggested rollout](#rollout)

---

<a name="the-wiring-problem"></a>

## 1. The wiring problem Capy solves

Look at the `counter` demo. It is four files that must agree on strings
and shapes that no compiler verifies:

```
demos/counter/
├── window.yaml      run_backend_script: python3 main.py   ← string "main.py"
├── main.py          @app.handle("add") / @app.handle("sub")  ← strings "add","sub"
├── static/app.js    BACKEND.call("add", {value}, …)        ← must match handler
└── static/index.html  <button onclick="…">                ← must match app.js
```

If you rename `add` → `increment` in `main.py`, nothing tells you the
button in `app.js` is now dead. If the Python handler sends
`{"value": …}` but the JS reads `data.count`, you find out at runtime.
The "app" is an emergent property of four files that share no source of
truth.

Capy lets you declare the app *once*, in a language whose vocabulary is
exactly the concepts of a webview_gui app — windows, actions, handlers,
events, native capabilities — and generate all four files from it. The
shared keys (`add`, the `{value}` shape) are written once and stamped
into every artifact, so they cannot drift.

> Capy never replaces `webview_gui`. It sits *in front of* it as a
> source-generation step. The `window` binary, the socket protocol, the
> `BACKEND`/`NATIVE` JS globals — all unchanged. Capy just authors the
> files you would otherwise hand-write.

---

<a name="how-capy-fits"></a>

## 2. How Capy fits the VHCO architecture

`webview_gui` is layered `domain → infra → features → orchestrator →
appio → main`, where each layer only imports below it. Capy enters at
two possible points, both clean:

- **Build-time (recommended first step).** A `window codegen app.capy`
  step (or a plain `go run`/`Makefile` target) turns one source file
  into `static/`, `main.py`, and `window.yaml` *before* `window` ever
  runs. `webview_gui` needs **zero code changes** — it loads the
  generated `window.yaml` exactly as today. Capy is just a preprocessor
  in your dev loop.

- **Embedded (later, optional).** Add a thin `infra/capy_codegen.go`
  that calls the Capy Go API (`capy.NewLibraryFromFile` → `lib.RunMulti`)
  and an `appio` flag `window --capy app.capy`. This obeys the layer
  rules: `infra` may import third-party packages (Capy is pure Go, no
  CGo), `appio` orchestrates it, `domain` stays untouched. See
  [§9](#embedding).

Because Capy is **pure Go with no CGo**, it co-exists with your existing
`webview_go` (CGo) and `wazero` (pure Go) dependencies without changing
your build matrix.

---

<a name="ten-minute"></a>

## 3. A 10-minute integration: the `app.capy` language

A Capy **library** (`.capy` file) defines your source language. Below is
a starter library that understands a handful of app concepts. Each
`function` block declares a piece of grammar and the text it expands to.

```
# app.capy — a tiny language for webview_gui apps.
# `extension` / output targets are set per-file below.

# ── A window declaration sets the page title and size ────────────────
function window
    arg literal "window"
    arg capture title  string
    arg capture w      int default "800"
    arg capture h      int default "800"
    block_closer end
    template
        <!doctype html>
        <html><head><meta charset="utf-8"><title>${decoded title}</title>
        <link rel="stylesheet" href="/static/app.css"></head>
        <body>
        <main id="app">
        ${body}
        </main>
        <script src="/static/app.js"></script>
        </body></html>
    end
end

# ── A button that calls a backend action ─────────────────────────────
function button
    arg literal "button"
    arg capture label  string
    arg literal "calls"
    arg capture action ident
    template
        <button data-action="${action}">${escapeHtml (decoded label)}</button>
    end
end

# ── A live text region bound to a server-push event ──────────────────
function label
    arg literal "label"
    arg capture id    ident
    arg literal "on"
    arg capture event ident
    template
        <span id="${id}" data-event="${event}"></span>
    end
end
```

A **script** written in that language is what an author (or an agent)
actually writes:

```
# counter.app — written in YOUR language, not HTML/JS/Python
window "Counter" 360 240
    label count on tick
    button "Increment" calls add
    button "Decrement" calls sub
end
```

Run it through Capy and the `window … end` block expands to a complete
`index.html`, with each `button`/`label` stamped consistently. The author
never touched a `<div>`, a `BACKEND.call`, or an `@app.handle`. The
vocabulary they *can* type is exactly the vocabulary your library defines
— everything else is a parse error (see [§5](#contract)).

Key library primitives you'll use (full list in Capy's docs):

| Directive | Purpose |
|-----------|---------|
| `arg literal "x"` | a fixed keyword the source must contain |
| `arg capture name TYPE` | a typed hole: `string`, `ident`, `int`, `bool`, `tail`, … |
| `… default "v"` | optional trailing arg with a default |
| `block_closer end` | the function wraps a nested body, closed by `end` |
| `block_verbatim end` | body captured as **raw bytes** (for embedded code) |
| `template … end` | multi-line output (sugar for a backtick `write`) |
| `${body}` `${line}` `${depth}` | render locals injected by the engine |
| `${escapeHtml …}` `${decoded …}` `${pascalCase …}` | template helpers |

---

<a name="one-source-many-files"></a>

## 4. One source → frontend + backend + config

The real payoff is **multi-file output**. A Capy library can declare
several output files and emit all of them from one script in a single
pass. In the Go API this is `lib.RunMulti(script)`, which returns the
primary output plus a `map[string]string` of every additional file.

Conceptually, one `counter.app` source fans out to:

```
counter.app  ──capy──▶  static/index.html      (the UI)
                        static/app.js           (BACKEND.call wiring)
                        main.py                 (@app.handle stubs)
                        window.yaml             (run_backend_script: python3 main.py)
```

Within the library, the same `button "Increment" calls add` statement
contributes to **three** files at once:

- to `index.html`: `<button data-action="add">Increment</button>`
- to `app.js`: a click listener that fires `BACKEND.call("add", payload, render)`
- to `main.py`: an `@app.handle("add")` stub with the right signature

Because all three come from the single token `add` in the source, the
JS call key, the Python handler name, and the DOM `data-action` are
**guaranteed identical**. Rename it in the source and all three move
together. That is the "one language for UI and backend" thesis made
concrete: the frontend and backend aren't *connected*, they're *projected
from the same sentence*.

A sketch of the generated `app.js` contribution (what your `button`
function's JS-target template emits):

```js
document.querySelectorAll('[data-action]').forEach(b =>
  b.addEventListener('click', () =>
    BACKEND.call(b.dataset.action, {}, ({ data }) => render(data))));

BACKEND.onEvent("tick", d =>
  document.querySelectorAll('[data-event="tick"]')
          .forEach(el => el.textContent = d.value));
```

…and the generated `main.py` contribution:

```python
@app.handle("add")
async def add(req, rw):
    await rw.send({"value": req.get("value", 0) + 1})   # ← author fills the body
```

Capy generates the *boundary* — the handler registration, the call site,
the event subscription, the `window.yaml` glue — which is the error-prone
part. Business logic lives in clearly marked body slots the author
completes.

---

<a name="contract"></a>

## 5. Enforcing the call/handler contract in the grammar

The deepest benefit isn't code volume saved — it's that **the parser is
the contract**. In Capy there is no built-in grammar, so the *only* valid
source is what your `.capy` library accepts. That has two consequences
for webview_gui:

1. **No orphan calls.** `button "x" calls frobnicate` only type-checks if
   you also declared a backend action `frobnicate` in the same source (a
   library can validate cross-references using its inner DSL — `set`,
   `if`, `error`, `regex_match`). A call with no handler becomes a
   *compile error*, not a silent dead button.

2. **Shape agreement.** If your `action` declaration names its payload
   fields, the same field list is stamped into the JS `BACKEND.call`
   payload and the Python handler's expected `req` keys. The
   `{"value": …}` mismatch from [§1](#the-wiring-problem) is unrepresentable.

Capy already ships the machinery for this: typed captures, optional/named
args, library-defined types (including group types like `[label](url)`),
and load-time validation that reports a clear error (with `${line}`/`${col}`
source positions) when a source violates the grammar. You're not building
a type checker — you're declaring rules the engine enforces.

---

<a name="run-modes"></a>

## 6. Targeting every run mode from one DSL

`webview_gui` has six run modes (server, url, proxy, browser, controlled,
wasm). Capy is target-agnostic, so the *same* app source can emit
different backends by swapping which output-file templates the library
uses — the UI source never changes:

| Target backend | Capy emits | webview_gui `mode` |
|----------------|-----------|--------------------|
| **Python subprocess** | `main.py` + `run_backend_script` | `""` (server) |
| **WASM module** | a TinyGo `main.go` with `alloc`/`handle` exports, built to `app.wasm` | `wasm` |
| **Native-only** | `app.js` calling `NATIVE.fs/os/dialogs/canvas`, `native_features:` list | `""` (no backend) |
| **Controlled** | `controller.py` driving `create_window`/`navigate`/`eval`/`close` | `controlled` |

For the **WASM** path, Capy's `block_verbatim` (raw-byte capture) is
exactly right for emitting the Go `handle()` switch: each declared action
becomes a `case "add":` arm, and the module contract
(`alloc(size i32) i32`, `handle(...) i64`, packed `ptr<<32|len` return)
is boilerplate the library stamps once — the author only writes the per-
action logic. One source, swap a flag, get a zero-dependency WASM app
instead of a Python one, with the *same* frontend.

For **native features**, the library can derive the `native_features:`
list automatically: if any statement in the source uses a `read_file` or
`exec` verb, the generated `window.yaml` includes `fs` / `os`
respectively. The capability surface is inferred from usage instead of
hand-maintained.

---

<a name="interesting"></a>

## 7. Other interesting things Capy unlocks

Beyond the core "one language" story, Capy's design enables several
patterns that are awkward today:

- **Progressive abstraction.** The same library can expose both a
  high-level verb (`form login`) and the low-level primitives it expands
  to (`button`, `field`, `label`). Authors pick their altitude; power
  users drop down without leaving the language.

- **Component libraries as `.capy` files.** A design system (cards,
  modals, tab strips) is a Capy library. Ship `corp-ui.capy`; every app
  built on it renders consistent HTML/CSS and the matching JS behavior.
  Update the library, regenerate, every app inherits the fix.

- **Multilingual / themable output.** Because the source is abstract,
  the same `counter.app` can render an English UI or a French one, a
  light theme or dark, just by selecting a different render context — no
  source duplication. (Capy's render locals and helpers like
  `${decoded}` make i18n-by-context natural.)

- **Embedded code blocks, untouched.** `block_verbatim` captures a body
  byte-for-byte — blank lines, `#` comments, indentation preserved — so
  you can embed a literal SQL query, a shader, or a Python snippet inside
  your app source and have it land in the output verbatim. No escaping
  gymnastics.

- **Host capabilities at generation time.** A Capy library can read
  `env`, CLI `args`, or `read_file` *during generation* (via the host
  interface) — e.g. bake the app version or a feature flag into the
  generated `window.yaml`. In sandboxed contexts (an agent, the
  playground) you swap in a `NoOpHost` so generation can touch nothing.

- **Auto-generated docs.** `RenderLibraryDocs(lib)` produces reference
  docs for *your* app language directly from the `.capy` file — so the
  vocabulary your team (or an agent) may use is always documented and in
  sync.

---

<a name="ai-agents"></a>

## 8. AI agents that build apps

This is where the combination is strongest. The goal: a user says
"build me a habit tracker," an AI agent produces a working native app,
and `webview_gui` runs it. Capy is the safe, compact interface between
the model and the machine.

### Why a DSL beats raw codegen

If you ask a model to emit `index.html` + `app.js` + `main.py` +
`window.yaml` directly, three things go wrong: it's ~800–1500 tokens of
output (slow, expensive), the four files drift out of sync (the model
forgets a handler), and you must execute whatever it produced (an
arbitrary-code risk).

Instead, the agent emits the **Capy source** — your app language:

```
window "Habit Tracker" 480 640
    list habits on habits_changed
    button "Add habit" calls add_habit
    button "Reset week"  calls reset
end
```

That's ~40 tokens. Your library expands it to the full multi-file app.
Measured on real libraries this is a **5–10× token reduction** on the
generation step, and the *expansion* is deterministic Go code you wrote
— not model output.

### Parser-as-grammar = a sandbox

Because Capy has zero default grammar, **anything the agent emits that
isn't in your library is rejected at parse time**, before any renderer
runs and before any file is written. The model literally cannot emit a
`<script>` tag, a `subprocess.run`, or an arbitrary import — those tokens
aren't in the grammar. The blast radius of a hallucination is "parse
error," not "shell command executed." Pair this with `NoOpHost` during
generation and the codegen step is fully sandboxed.

### MCP: hand the agent the tools, not the syntax

Capy ships an MCP server (`cmd/capy-mcp`) and supports MCP widgets. An
agent connected over MCP can:

- call **introspection** (`lib.Introspect()`) to discover every function,
  its args, types, and docstrings — so it learns your app language at
  runtime instead of you pasting a grammar into the prompt;
- **validate** a candidate source (`capy check`) and get structured
  errors with `${line}`/`${col}` positions to self-correct;
- **render** the source to the multi-file app (`lib.RunMulti`).

A typical agent loop:

```
1. introspect app.capy          → "here are the verbs you may use"
2. draft counter.app            → ~40 tokens of DSL
3. capy check counter.app       → parse error? feed it back, retry
4. lib.RunMulti(counter.app)    → static/ + main.py + window.yaml
5. window window.yaml           → the user sees their app
```

Steps 3–4 are deterministic and sandboxed; only step 2 is the model. The
agent's creative latitude is bounded to *what your grammar allows* — which
is exactly the safety property you want when generated code opens a native
window on a user's machine.

### Hot reload while iterating

Capy compiles in-browser via WASM (`cmd/capy-wasm`). You can embed that
in a `webview_gui` dev window so an author — or an agent in a chat panel
— edits app source and sees the regenerated app live, with no rebuild.
The same compiler that runs server-side runs in the webview, so
"preview" and "build" can't diverge.

---

<a name="embedding"></a>

## 9. Embedding the Capy compiler in `window`

When you're ready to go beyond a build step, the embedded path is small
and respects VHCO. The Capy Go API is pure Go:

```go
// infra/capy_codegen.go  (infra may import third-party, no domain logic)
package infra

import "github.com/olivierdevelops/capy" // pure Go, no CGo

// GenerateApp turns one .capy source into a set of output files
// (path → contents), ready to write next to a window.yaml.
func GenerateApp(libPath, scriptSrc string) (map[string]string, error) {
    lib, err := capy.NewLibraryFromFile(libPath)
    if err != nil {
        return nil, err
    }
    // Sandbox generation: no env/file/exec access during codegen.
    // lib.SetHost(capy.NoOpHost{})   // if/when you accept untrusted source
    primary, others, err := lib.RunMulti(scriptSrc)
    if err != nil {
        return nil, err
    }
    out := map[string]string{lib.OutputFile(): primary}
    for path, contents := range others {
        out[path] = contents
    }
    return out, nil
}
```

Then an `appio` flag wires it into the CLI you already have:

```go
// appio/cli.go — add a --capy mode
//   window --capy app.capy counter.app   → generate, then run window.yaml
```

This obeys the layer rules: `infra` owns the Capy dependency and file I/O,
`appio` orchestrates the flag, `orchestrator`/`features`/`domain` are
untouched. The generated `window.yaml` flows through your existing
`appio/config_loader.go` exactly as a hand-written one would.

For an **agent-driven** build, expose the same `GenerateApp` over your
backend socket or MCP so a `BACKEND.call("build_app", {source})` returns
the rendered files (or writes them and navigates the window to the result).

---

<a name="rollout"></a>

## 10. Suggested rollout

A low-risk path that delivers value at every step:

1. **Pilot library, build step only.** Write `app.capy` covering the
   `counter` demo's vocabulary. Generate its four files with a `go run`
   step. Diff against the hand-written demo until they match. No `window`
   changes yet.
2. **Add the contract checks.** Make orphan `calls` / shape mismatches a
   compile error. This is the first thing that catches a real bug the
   current setup can't.
3. **Second backend target.** Emit the WASM `main.go` from the *same*
   source; prove one app source produces both a Python and a WASM build
   by flag.
4. **Embed (`window --capy`).** Add `infra/capy_codegen.go` + the flag
   ([§9](#embedding)) so generation is one command.
5. **Agent surface.** Stand up the MCP server, expose introspection +
   check + render, and let an agent author apps end-to-end with
   `NoOpHost` sandboxing.

Each step is independently shippable, and at no point does the existing
`webview_gui` runtime — the socket protocol, `BACKEND`/`NATIVE`, the run
modes — have to change. Capy only ever authors the files; `window` runs
them.

---

### One-paragraph summary

`webview_gui` runs apps assembled from a frontend, a backend, and a
config that no compiler checks against each other. Capy lets you declare
the whole app once in a small language *you* define, then projects that
single source into all three artifacts — with the call/handler contract
enforced by the grammar instead of by convention. The same source can
target a Python, WASM, native, or controlled backend by a flag. And
because Capy has zero default grammar, it's the ideal interface for AI
agents: they emit a few dozen tokens of your app language, the library
deterministically expands it into a working native app, and anything
outside the grammar is rejected before a file is ever written.
