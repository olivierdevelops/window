# Integrating Capy: define your own syntax, generate any text

> **Version:** targets **Capy `v0.20.0`** (module
> `github.com/olivierdevelops/capy`). `window` pins it in `go.mod`; during
> local development a `replace` points at a checkout of the Capy repo. Capy is
> pre-1.0 — the library `.capy` schema may change between minor versions, so
> check [Capy's CHANGELOG](https://github.com/olivierdevelops/capy/blob/main/CHANGELOG.md)
> when bumping.

[**Capy**](https://github.com/olivierdevelops/capy) is a transpiler engine with
**zero default grammar**. You define a small source language in a `.capy`
*library*, and Capy gives you a parser + code generator that turns scripts
written in *your* language into *any* textual output — HTML, JS, Python, Go,
YAML, TS, SQL — one file or many, in a single pass. There is no separate
template or config language: the library is itself written in Capy's native
syntax, and the renderer walks the parsed AST directly.

This document is a practical, self-contained integration guide. It is written
so **any Go application** (not just `window`) can embed Capy, with `window` used
as the running example. You'll find: the embedding API, the current library
syntax, the multi-file pattern, grammar-as-contract validation, introspection
for editors/agents, host sandboxing, and the v0.20.0 CLI/tooling.

---

## Table of contents

1. [Why embed Capy](#why)
2. [Embed in any Go program — the canonical flow](#embed)
3. [Authoring a library: current syntax reference](#syntax)
4. [One source → many files](#multi-file)
5. [Grammar-as-contract: validation & errors](#contract)
6. [Introspection, docs & editor/agent integration](#introspect)
7. [Host capabilities & sandboxing](#hosts)
8. [CLI & tooling (v0.20.0)](#cli)
9. [`window` specifics: formats, run modes, `--capy`](#window)
10. [AI agents that build apps](#ai-agents)
11. [Versioning & compatibility](#versioning)
12. [Suggested rollout](#rollout)

---

<a name="why"></a>

## 1. Why embed Capy

A real app is usually several artifacts that must agree on strings and shapes
no compiler checks. `window`'s `counter` demo is four files:

```
demos/counter/
├── window.yaml      run_backend_script: python3 main.py   ← string "main.py"
├── main.py          @app.handle("add") / @app.handle("sub")  ← strings "add","sub"
├── static/app.js    BACKEND.call("add", {value}, …)        ← must match handler
└── static/index.html  <button onclick="…">                ← must match app.js
```

Rename `add` → `increment` in `main.py` and nothing tells you the JS button is
now dead; send `{"value": …}` from Python while JS reads `data.count` and you
find out at runtime. The "app" is an emergent property of files that share no
source of truth.

Capy lets you declare the app **once**, in a language whose vocabulary is
exactly your domain's concepts, and project that single source into every
artifact. Shared keys are written once and stamped into each output, so they
cannot drift. Embedding Capy is worthwhile whenever you:

- ship a CLI/app that takes config in a friendlier-than-YAML DSL — write the
  parser in ~50 lines of Capy instead of 500 of Go;
- build a code generator (schema → migrations, IDL → client+server) and want
  users to write `model User { name : string }` instead of a Go builder API;
- want hot-swappable grammars — read a library at startup, let users contribute
  new ones with no recompile;
- need a **safe** interface for AI-generated input: anything outside the
  grammar is a parse error, before any file is written ([§10](#ai-agents)).

> Capy never replaces your runtime. It sits *in front of* it as a
> source-generation step — it authors the files you would otherwise hand-write.

---

<a name="embed"></a>

## 2. Embed in any Go program — the canonical flow

Capy is **pure Go, no CGo**, so it drops into any module without changing your
build matrix. The whole API is small:

```go
import "github.com/olivierdevelops/capy"

// 1. Compile a library (your language definition). Reuse it across many runs.
lib, err := capy.NewLibrary(librarySrc)        // from an in-memory string
//  or: lib, err := capy.NewLibraryFromFile("app.capy")

// 2a. Single-output: run a script, get the rendered text.
out, err := lib.Run(scriptSrc)                 // string

// 2b. Multi-output: get the primary string + every `file "..."` block.
primary, files, err := lib.RunMulti(scriptSrc) // (string, map[string]string, error)
```

`Library` is safe to reuse across many `Run`/`RunMulti` calls (each call runs a
fresh accumulating context); it is not safe for concurrent *mutation*, but
`Run` itself is re-entrant on a fixed library. Other library methods:

| Method | Returns |
|---|---|
| `lib.Extension()` | the library's declared `extension` (e.g. `"yaml"`) — pick the right output suffix |
| `lib.OutputFile()` | optional declared `output_file` name for single-output libraries |
| `lib.FunctionNames()` | sorted names of every declared function |
| `lib.Introspect()` | `[]FunctionInfo` — args, types, block kind, docs ([§6](#introspect)) |
| `lib.CommentMarkers()` | the library's declared line-comment markers |
| `lib.SetHost(h)` | install a `domain.Host` for `env`/`arg`/`read_file` ([§7](#hosts)) |
| `capy.RenderLibraryDocs(lib)` | Markdown reference docs for your language |

### The integration `window` actually ships

`window`'s `infra/capy_codegen.go` is the whole embedding — it obeys the VHCO
layering (only `infra` imports the third-party package; `appio` orchestrates a
flag; `domain`/`features`/`orchestrator` are untouched):

```go
package infra

import "github.com/olivierdevelops/capy" // pure Go, no CGo

// GenerateCapyApp compiles a Capy library and runs a source script through it,
// returning the generated app files (path -> contents): window.yaml, the
// static/ frontend, and any other `file "..."` blocks the library declares.
//
// The default Capy host is NoOpHost, so generation cannot touch the
// environment or filesystem — safe to run on untrusted source.
func GenerateCapyApp(librarySrc, scriptSrc string) (map[string]string, error) {
	lib, err := capy.NewLibrary(librarySrc)
	if err != nil {
		return nil, err
	}
	primary, files, err := lib.RunMulti(scriptSrc)
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = map[string]string{}
	}
	// If the library produced a primary output (no file blocks), fall back to
	// its declared output_file name.
	if primary != "" {
		if out := lib.OutputFile(); out != "" {
			files[out] = primary
		}
	}
	return files, nil
}
```

`appio/cli.go` then writes those files to a temp dir and loads the generated
`window.yaml` exactly like a hand-written one — see
[`loadWindowLang`/`transpileSource`](../appio/cli.go).

---

<a name="syntax"></a>

## 3. Authoring a library: current syntax reference

A `.capy` library has three parts: **header declarations**, **function
definitions** (your grammar), and **output blocks** (the renderer). The
snippets below use the *current* (v0.20.0) syntax — the same syntax the shipped
[`assets/window.capy`](../assets/window.capy) library uses.

### 3.1 Header declarations

```capy
extension yaml          # default output file suffix (no dot)

comments                # line-comment markers your scripts may use
    line "#"
end

context                 # the accumulator: named fields with defaults that
    title  "App"        # functions mutate and output blocks read.
    width  480
    ui     []           # lists start empty; push to them with `append`
    calls  []
end
```

- `extension <ext>` — the suffix `Extension()` reports.
- `comments { line "x" … }` — without it, scripts have no comment syntax.
- `context { … }` — the shared, mutable state for one run. Scalars (`"App"`,
  `480`) and lists (`[]`) are declared with defaults; functions read and write
  them with `set` / `append`; output blocks read them with `${ … }`.

### 3.2 Function definitions — the grammar

Each `function` declares one statement of your language: the literal keywords it
must contain, the typed holes it captures, whether it wraps a block, and what it
does to the `context`.

```capy
# A block statement: `app "Title" 800 600 … end`
function app
    arg literal "app"
    arg capture title string
    arg capture w     int default "480"     # optional trailing arg + default
    arg capture h     int default "640"
    block_closer end                          # this function wraps a body
    set context.title title
    set context.width  w
    set context.height h
end

# A leaf statement: `button "Increment" calls add`
function button
    arg literal "button"
    arg capture label  string
    arg literal "calls"
    arg capture action ident
    append context.ui    {kind: "button", label: label, action: action}
    append context.calls {name: action}       # remembered for the backend file
end
```

**Argument directives**

| Directive | Meaning |
|---|---|
| `arg literal "x"` | a fixed keyword the source must contain at this position |
| `arg capture NAME TYPE` | a typed hole bound to `NAME` |
| `arg capture NAME TYPE default "v"` | optional trailing arg with a default |
| `block_closer end` | the function wraps a nested body, closed by `end` |
| `block_verbatim end` | body captured as **raw bytes** (embedded code, SQL, shaders) — indentation/blank lines/comments preserved |

**Capture types:** `string`, `int`, `bool`, `ident`, `dotted_ident`, `word`,
`tail` (rest-of-line). (`string` values arrive quoted — use `${decoded …}` /
`unquote` to strip the quotes; see helpers below.)

**Body verbs (the inner DSL):** `set`, `append`, `prepend`, `if`/`else`, `for`,
`while`, `write`, `error`, `return`. These run at parse time as each statement
matches, building up `context`.

### 3.3 Output blocks — rendering

A `file "path"` block (or a single `file_template`) renders text. Inside it you
`write` backtick literals with `${ … }` interpolation, and loop/branch over the
`context` you accumulated:

```capy
file "static/index.html"
    write `<!doctype html>
<title>${escapeHtml context.title}</title>
<main id="app">
`
    for n in context.ui
        if n.kind == "button"
            write `  <button data-action="${n.action}">${escapeHtml n.label}</button>
`
        end
    end
    write `</main>`
end
```

Whitespace inside backtick literals is significant (the formatter never touches
it). `if`/`for` blocks are closed by `end`.

**Template helpers** (use inside `${ … }`):

| Group | Helpers |
|---|---|
| Strings | `decoded`, `unquote`, `unescape`, `escapeHtml`, `toQuoted`, `trim`, `trimPrefix`, `trimSuffix`, `lower`, `indent`, `align`, `join`, `len` |
| Case | `camelCase`, `pascalCase`, `snakeCase`, `dasherize` |
| Data | `toJSON`, `toJSONIndent`, `toPyLit`, `asString` |
| Arithmetic | `add`, `sub`, `mul`, `div`, `mod`, `percent` |

Helpers compose with parentheses, and `file` paths can themselves be templated:

```capy
file "components/${dasherize (unquote context.name)}.tsx"
    write `export const ${pascalCase (unquote context.name)} = () => …`
end
```

A complete, runnable example of all of this is
[`assets/window.capy`](../assets/window.capy) (UI → HTML + JS + YAML) and the
samples in the [Capy repo](https://github.com/olivierdevelops/capy/tree/main/samples).

---

<a name="multi-file"></a>

## 4. One source → many files

The payoff is **multi-file output**: a library declares several `file "path"`
blocks and `RunMulti` emits all of them from one script in a single pass,
returning `(primary, map[path]contents, err)`.

A script in your language:

```capy
# counter.window — written in YOUR language, not HTML/JS/Python
app "Counter" 360 240
    text count
    button "Increment" calls add
    button "Decrement" calls sub
end
```

…fans out (in `window`'s case) to a complete frontend + config:

```
counter.window ──capy──▶ static/index.html   (the UI)
                         static/app.js        (event wiring / BACKEND.call)
                         window.yaml          (entry_path, size, native_features)
```

Because all three derive from the single token `add` in the source, the DOM
`data-action`, the JS call key, and the handler name are **guaranteed
identical**. Rename it in the source and all three move together — the frontend
and backend aren't *connected*, they're *projected from the same sentence*.

Scripts can also do light **metaprogramming**: a `define NAME … end` block at
the top of a script is extracted and merged into the library before evaluation,
so a script can add a one-off function to its own language. `RunMulti` honors
this exactly as the CLI does. See
[`docs/metaprogramming.md`](https://github.com/olivierdevelops/capy/blob/main/docs/metaprogramming.md).

---

<a name="contract"></a>

## 5. Grammar-as-contract: validation & errors

Because Capy has **no built-in grammar**, the *only* valid source is what your
library accepts — **the parser is the contract**:

1. **No orphan references.** `button "x" calls frobnicate` can be made to
   type-check only if `frobnicate` was also declared in the same source — a
   library validates cross-references with its inner DSL (`set`, `if`, `error`)
   and raises a parse-time `error` otherwise. A call with no handler is a
   *compile error*, not a silent dead button.

2. **Shape agreement.** If an `action`'s payload fields are named in the source,
   the same field list is stamped into both the JS call payload and the
   backend handler's expected keys. Shape mismatches become unrepresentable.

Errors carry source positions. From Go, a failed `Run`/`RunMulti` returns an
`error` whose message includes line/column; on the CLI, `capy check` reports the
same. You're not writing a type checker — you declare rules and the engine
enforces them. See
[`docs/grammar-as-contract.md`](https://github.com/olivierdevelops/capy/blob/main/docs/grammar-as-contract.md).

---

<a name="introspect"></a>

## 6. Introspection, docs & editor/agent integration

A library is self-describing, so you never hand-maintain a parallel catalogue of
"what verbs exist":

```go
for _, fn := range lib.Introspect() { // []capy.FunctionInfo
	// fn.Name, fn.Description, fn.Block (none/closer/verbatim), fn.Priority
	for _, a := range fn.Args {        // []capy.ArgInfo
		// a.Literal ("button") OR a.Capture ("label") + a.Type ("string") + a.Default
	}
}

names := lib.FunctionNames()          // []string, sorted
markers := lib.CommentMarkers()       // []string, for a syntax highlighter
md := capy.RenderLibraryDocs(lib)     // Markdown reference for YOUR language
```

This is exactly what an editor needs for autocomplete / hover / highlighting,
and what an agent reads to *learn your language at runtime* instead of you
pasting a grammar into a prompt. `capy docs <lib>` writes the same Markdown
`RenderLibraryDocs` returns.

---

<a name="hosts"></a>

## 7. Host capabilities & sandboxing

A library's inner DSL can read the outside world during generation — `env`, CLI
`arg`s, `read_file` — to bake a version string or feature flag into the output.
That access is mediated by a `Host`:

```go
import (
	"github.com/olivierdevelops/capy"
	capyinfra "github.com/olivierdevelops/capy/infra"
)

lib, _ := capy.NewLibrary(src)
// Default after NewLibrary is domain.NoOpHost: env/arg return zero values and
// read_file errors out — generation cannot touch anything. Safe for untrusted
// (e.g. AI-generated) source.

lib.SetHost(capyinfra.OSHost{}) // opt IN to real os.Getenv / os.Args / os.ReadFile
//                                  — only when the library source is trusted.
```

`window`'s `GenerateCapyApp` keeps the default `NoOpHost`, so running a
generated app from arbitrary source can't read your environment or files. Flip
to `OSHost` only for first-party libraries you control. See
[`docs/host-capabilities.md`](https://github.com/olivierdevelops/capy/blob/main/docs/host-capabilities.md).

---

<a name="cli"></a>

## 8. CLI & tooling (v0.20.0)

You don't have to embed Capy to use it — the `capy` binary covers the whole
dev loop, and a library can ship as its own CLI:

| Command | What it does |
|---|---|
| `capy run <lib.capy> <script>` | transpile a script through a library (legacy invocation) |
| `capy <lib> <command> [args…]` | **library command dispatch** — libraries declare commands and run as their own CLI ("libraries as CLIs") |
| `capy check <lib> <script>` | parse + validate; structured errors with positions (CI gate) |
| `capy docs <lib>` | render Markdown reference docs for the library's language |
| `capy fmt <files…>` | conservative formatter (`--check` / `--diff` / `--stdout`); never touches backtick-literal internals |
| `capy watch <lib> [args…]` | re-run on any change to the library dir or file-path args (250ms poll) |
| `capy lib add <git-url\|path> [--as name]` | install a library onto `CAPY_LIBS`; `capy lib remove <name>` to delete |
| `capy build <lib> [-o out]` | compile a **standalone binary** with the library baked in — runs with no Capy install on the target (cross-compile via `GOOS`/`GOARCH`) |

Beyond the CLI, Capy ships an **MCP server** (`cmd/capy-mcp`) so an agent can
introspect/validate/render over MCP, and an **in-browser WASM** build
(`cmd/capy-wasm`) so the *same* compiler runs in a webview for live preview —
"preview" and "build" can't diverge. See
[`docs/cli.md`](https://github.com/olivierdevelops/capy/blob/main/docs/cli.md),
[`docs/library-commands.md`](https://github.com/olivierdevelops/capy/blob/main/docs/library-commands.md),
[`docs/mcp.md`](https://github.com/olivierdevelops/capy/blob/main/docs/mcp.md).

---

<a name="window"></a>

## 9. `window` specifics: formats, run modes, `--capy`

`window` already embeds Capy for three of its authoring formats — each is a
`.capy` library compiled by `GenerateCapyApp`:

| You write | Library | Compiles to |
|---|---|---|
| `.window` | [`assets/window.capy`](../assets/window.capy) | `static/` + `window.yaml` (in-process Go backend) |
| `.htmlx` | [`assets/htmlx.capy`](../assets/htmlx.capy) | a normalized HTML document |
| `.cs` (CapyScript) | [`assets/capyscript.capy`](../assets/capyscript.capy) | JavaScript |

(The fourth format, `.capyx`, is compiled by `window`'s own `infra/capyx*.go`,
not by Capy — see [authoring-formats.md](./authoring-formats.md).)

Run any of them by extension:

```sh
window app.window     # transpiled by window.capy
window app.htmlx      # transpiled by htmlx.capy
window app.cs         # transpiled by capyscript.capy
```

Because Capy is target-agnostic, one app source can emit different backends by
swapping which `file` blocks the library renders, while the UI source stays the
same — Python subprocess (`main.py` + `run_backend_script`), a zero-dependency
WASM module (TinyGo `main.go` via `block_verbatim`, built to `app.wasm`,
`mode: wasm`), native-only (`NATIVE.*` + `native_features:`), or controlled
mode. The capability surface can even be **inferred from usage** — if a
statement uses a `read_file`/`exec` verb, the generated `window.yaml` adds the
matching `native_features:` entry.

---

<a name="ai-agents"></a>

## 10. AI agents that build apps

This is where the combination is strongest. Instead of asking a model to emit
`index.html` + `app.js` + `main.py` + `window.yaml` directly (~800–1500 tokens,
four files that drift, arbitrary code you must execute), the agent emits **your
app language**:

```capy
app "Habit Tracker" 480 640
    list habits
    button "Add habit" calls add_habit
    button "Reset week" calls reset
end
```

That's ~40 tokens; your library deterministically expands it into the full
multi-file app — a **5–10× token reduction** on the generation step, and the
expansion is Go code you wrote, not model output.

**Parser-as-sandbox.** Anything the agent emits that isn't in your grammar is
rejected at parse time, before any renderer runs and before any file is written.
The model literally cannot emit a `<script>` tag or a `subprocess.run` — those
tokens aren't in the grammar. Combined with the default `NoOpHost`
([§7](#hosts)), codegen is fully sandboxed.

A typical agent loop (all of it available over MCP, [§8](#cli)):

```
1. lib.Introspect()            → "here are the verbs you may use"
2. draft counter.window        → ~40 tokens of DSL
3. capy check                  → parse error? feed it back, retry
4. lib.RunMulti(source)        → static/ + window.yaml (deterministic, sandboxed)
5. window window.yaml          → the user sees their app
```

Only step 2 is the model; steps 3–4 are deterministic and sandboxed. See
[`docs/ai-agents.md`](https://github.com/olivierdevelops/capy/blob/main/docs/ai-agents.md).

---

<a name="versioning"></a>

## 11. Versioning & compatibility

- **Pinned version:** `github.com/olivierdevelops/capy v0.20.0` in
  [`go.mod`](../go.mod). A local `replace` points at a Capy checkout for
  development; remove/adjust it when consuming the tagged release.
- **Pre-1.0 caveat:** the `.capy` library schema may break between *minor*
  versions. Library directives (`function`, `arg`, `context`, `file`, helpers)
  are stable within a minor; when bumping, re-run `capy check` on your libraries
  and skim the [CHANGELOG](https://github.com/olivierdevelops/capy/blob/main/CHANGELOG.md).
- **Migrating libraries:** [`docs/migration-guide.md`](https://github.com/olivierdevelops/capy/blob/main/docs/migration-guide.md)
  tracks breaking syntax changes between versions.
- **No CGo:** Capy is pure Go; it co-exists with `window`'s `webview_go` (CGo)
  and `wazero` (pure Go) deps without changing the build matrix.

---

<a name="rollout"></a>

## 12. Suggested rollout

A low-risk path that delivers value at each step:

1. **Pilot library, build step only.** Write a `.capy` covering one demo's
   vocabulary; generate its files with a `go run` step and diff against the
   hand-written version. No host code changes.
2. **Add contract checks.** Make orphan references / shape mismatches a parse
   `error` — the first thing that catches a bug the current setup can't.
3. **Second output target.** Emit a different backend (e.g. WASM `main.go`) from
   the *same* source to prove one source → many builds.
4. **Embed.** Add an `infra/` codegen helper ([§2](#embed)) + a CLI flag, so
   generation is one command.
5. **Agent surface.** Stand up the MCP server, expose introspect + check +
   render, and let an agent author end-to-end with `NoOpHost` sandboxing.

Each step is independently shippable, and the host runtime never has to change —
Capy only ever authors the files.

---

### One-paragraph summary

Capy is a pure-Go transpiler engine with no built-in grammar: you define a small
language in a `.capy` library (`extension` + `comments` + `context` header,
`function`/`arg`/`capture` grammar, `file "…"` output blocks with `${ … }`
helpers), compile it with `capy.NewLibrary`, and project one source script into
one or many output files with `lib.RunMulti`. The parser *is* the contract, so
invalid input fails before a file is written; libraries are self-describing
(`Introspect`, `RenderLibraryDocs`) for editors and agents; generation is
sandboxed by default (`NoOpHost`); and the v0.20.0 CLI (`run`, `check`, `docs`,
`fmt`, `watch`, `lib`, `build`) plus MCP/WASM cover authoring, validation, and
shipping. `window` embeds it today for its `.window`, `.htmlx`, and `.cs`
formats — and any Go application can embed it the same way.
