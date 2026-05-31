# Authoring a window app: HTML, JS, custom components, and reactive apps

`window` can build an app from several source formats, all compiled into the
same thing: a `window.yaml` + a `static/` frontend that the native webview runs.

| You write | File | Compiles to | Best for |
|---|---|---|---|
| Matched-pair HTML | `.htmlx` | a full HTML document | static pages, full layout control |
| A JS-like script | `.cs` | real JavaScript | algorithms, scratch logic |
| Custom HTML tags | `.htmlx` + `<component>` | your own components | reusable UI pieces |
| Reactive VHCO app | `.capyx` | HTML + signals runtime | stateful, fine-grained reactive UIs |

The `.htmlx` / `.cs` formats are transpiled by the
[Capy](https://github.com/olivierdevelops/capy) engine; `.capyx` is compiled by
the project's own [`infra/capyx*.go`](../infra) into an inlined signals runtime.

Run any of them the same way:

```sh
window app.htmlx
window app.cs
window app.capyx
```

---

## 1. HTML — `.htmlx`

A `.htmlx` file is a whole app written as **real, matched-pair HTML**:
`<tag>…</tag>`. The root `<app>` sets the window title + size and wraps
everything in a normalized, cross-platform document. Text nodes are
**quoted strings** (escaped once).

### Transpile pipeline

Before the Capy grammar runs, two Go preprocessors expand author-facing
syntax; then `assets/htmlx.capy` parses the result:

```
.htmlx source
    │
    ├─► RewriteHTMLXComponents()   ← <component> → Capy define blocks
    │
    ├─► ExpandControlFlow()        ← {#for}/{#if}/{#match} → static HTML
    │
    └─► Capy htmlx.capy            ← <app>, elements, void tags, quoted text
            │
            ├─► window.yaml
            └─► static/index.html
```

The CLI wires this in `appio/cli.go` → `loadHTMLX()`. Demos are regression-tested
by `go test ./demos/htmlx/`.

### Source

```html
<app title="Hello" width="460" height="320">
  <h1>"Hello, window"</h1>
  <p class="muted">"A whole desktop app, written as matched-pair HTML."</p>
  <a class="btn" href="https://github.com/olivierdevelops/window">"Docs"</a>
</app>
```

### What it produces (`static/index.html`, body excerpt)

```html
<!doctype html>
<html lang="en"><head> … reset CSS … </head>
<body><main>
<h1>Hello, window</h1>
<p class="muted">A whole desktop app, written as matched-pair HTML.</p>
<a class="btn" href="https://github.com/olivierdevelops/window">Docs</a>
</main></body></html>
```

### Why matched pairs

The parser tracks every open tag and demands its own matching close, so a
mismatch is a **transpile error**, not a silently broken page:

```html
<div class="card"><p>"oops"</div>   ← </div> can't close <p>: hard error
```

### What you can write

| Form | Meaning |
|---|---|
| `<app title="…" width="…" height="…"> … </app>` | Root: window metadata + document wrapper |
| `<tag attr="v"> … </tag>` | Any matched-pair element; nests freely |
| `<tag attr="v" />` | Self-closing void element (`<input />`, `<br />`, `<hr />`) |
| `"quoted text"` | A text node — escaped once; one string per line |
| `attr="value"` | Attributes pass through verbatim |
| `# …` | Line comment (stripped before parse; handy above `<component>` blocks) |

The root `<app>` attributes must appear in this **fixed order**:
`title`, then `width`, then `height`. Other attribute orders are not accepted
by the Capy grammar.

Built-in utility classes ship with the wrapper: `.card`, `.row`, `.grid`,
`.btn`, `.muted`, `.badge`.

### Renders the same on every OS and engine

The native webview is a different browser engine on each platform — WKWebView on
macOS, WebView2/Chromium on Windows, WebKitGTK on Linux — and each ships its own
UA defaults, system fonts, native form widgets and scrollbars. The `.htmlx`
wrapper normalizes all of that so a page looks near-identical everywhere:

- **Pinned metrics** — explicit `font-size`, `line-height`, `tab-size`,
  `box-sizing:border-box`, and `font-synthesis:none` (so a missing bold/italic
  cut is never faux-synthesized differently per OS).
- **Form controls reset to zero** — `appearance:none` on buttons and fields, a
  custom SVG `<select>` arrow, normalized number spinners / search decorations,
  a consistent `::placeholder` color, and a single `:focus-visible` ring.
- **Native widgets tinted** — `accent-color` gives checkboxes, radios and ranges
  one color instead of the per-OS blue/green.
- **Styled scrollbars** — identical thin scrollbars via `::-webkit-scrollbar`
  plus `scrollbar-width`/`scrollbar-color`, replacing macOS overlay vs
  Windows/Linux chunky bars.
- **Consistent type** — a deep `monospace` stack for `<pre>`/`<code>`,
  antialiased smoothing, and a `prefers-reduced-motion` guard.

The reset lives in [`assets/htmlx.capy`](../assets/htmlx.capy) (the `app`
function's document wrapper).

---

## 2. JavaScript — CapyScript (`.cs`)

A `.cs` file is a tiny **JavaScript-like** language that compiles to real
JavaScript and runs in a console-style window — `log` output is mirrored
onto the page.

### Source (`fib.cs`)

```
const N = 15

fn fib(n)
    if n < 2
        return n
    else
        return fib(n - 1) + fib(n - 2)
    end
end

let i = 0
while i < N
    log "fib(" + i + ") = " + fib(i)
    do i = i + 1
end
```

### What it produces (`static/app.js`)

```js
const N = 15;
function fib(n) {
  if (n < 2) {
    return n;
  } else {
    return fib(n - 1) + fib(n - 2);
  }
}
let i = 0;
while (i < N) {
  console.log("fib(" + i + ") = " + fib(i));
  i = i + 1;
}
```

### The language

| CapyScript | Compiles to |
|---|---|
| `let x = 1` / `const y = 2` | `let x = 1;` / `const y = 2;` |
| `fn name(args) … end` | `function name(args) { … }` |
| `if cond … else … end` | `if (cond) { … } else { … }` |
| `for v in iter … end` | `for (const v of iter) { … }` |
| `while cond … end` | `while (cond) { … }` |
| `return expr` | `return expr;` |
| `log expr` | `console.log(expr)` (mirrored on-page) |
| `do EXPR` | `EXPR;` — escape hatch for any raw JS |

Blocks (`fn`, `if`, `for`, `while`) close with `end`. There is no catch-all
line rule, so anything that isn't a keyword — an assignment, a method call,
an arrow function — goes through `do`. That keeps the grammar unambiguous
while letting full JavaScript through:

```
fn makeCounter(start)
    do let n = start
    do return () => { n = n + 1; return n; }
end
```

---

## 3. Custom components — `<component>`

Inside a `.htmlx` file you can **define your own tags** with a `<component>`
block, then use them as if they were built in:

- **`props="…"`** — space-separated named attributes, read inside as `{{ prop }}`
  (escaped automatically)
- **`<slot></slot>`** — where the nested children go
- **`void`** — add the bare attribute for a self-closing tag

It's all HTML. No JavaScript, no Capy grammar to learn — `<component>`
expands to a real tag at transpile time. Interpolation uses the same
double-brace `{{ … }}` text binding as `.capyx` and the control-flow tags below.

### Define it once

```html
<component name="card" props="title">
  <section class="card">
    <h3>{{ title }}</h3>
    <slot></slot>
  </section>
</component>
```

### Use it like built-in HTML

```html
<card title="Welcome">
  <badge>"New"</badge>
  <p>"Children fill the slot."</p>
</card>
```

### Expands to

```html
<section class="card">
  <h3>Welcome</h3>
  <span class="badge">New</span>
  <p>Children fill the slot.</p>
</section>
```

### A void (self-closing) component

Add `void`; use the tag self-closed:

```html
<component name="avatar" props="name" void>
  <div class="row">
    <img src="https://ui-avatars.com/api/?name={{ name }}" width="40" height="40"
         style="border-radius:50%" />
    <b>{{ name }}</b>
  </div>
</component>

<avatar name="Ada" />
```

### Notes

- **Props are escaped.** `{{ prop }}` renders the escaped attribute value, so
  components are XSS-safe by default.
- **`<slot>` is the body.** Children between the open/close tags render there;
  compose components freely (`<card>` containing `<badge>`).
- **One named prop per declared name.** A custom tag exposes exactly the props
  you list — they're validated like any other attribute. (The built-in generic
  `<tag>` rule still accepts arbitrary attributes when you need them.)

See [`demos/htmlx/components.htmlx`](../demos/htmlx/components.htmlx) for a
full component kit (`<card>`, `<stat>`, `<badge>`, `<avatar/>`) that builds a
dashboard.

---

## 4. Control flow — `{#for}`, `{#if}`, `{#match}`

A `.htmlx` file can use **compile-time control flow** written in the same
**three-brace** directive syntax as [`.capyx`](./capyx-reactive-vhco.md):
`{#for}`, `{#if}`/`{#elif}`/`{#else}`, and `{#match}`/`{#case}`/`{#default}`,
each closed with `{/…}`. There is no runtime data model — every construct is
*evaluated while the file compiles*, so the output is plain static HTML. They
compose: an `{#if}` inside a `{#for}` sees the loop variable.

Each directive tag sits on its **own line** (indent freely). Loop variables are
referenced with the `{{ name }}` text binding and substituted as raw text before
the page is escaped downstream — the same `{{ … }}` rule components use.

### `{#for}` — repeat a block

```html
{#for label in Home, Docs, About}
  <a class="btn" href="#">"{{ label }}"</a>
{/for}
```

`{#for VAR in A, B, C}` iterates a comma-separated list; `VAR` is the loop
variable. The block is emitted once per item with `{{ VAR }}` filled in.

### `{#if}` / `{#elif}` / `{#else}` — pick a branch

```html
{#if role == admin}
  <p>"Edit access granted."</p>
{#elif role == editor}
  <p>"Can edit."</p>
{#else}
  <p>"Read-only."</p>
{/if}
```

Conditions compare with `==` (string equality). To test membership against a
list, use `in`:

```html
{#if role in admin, editor}
  <p>"Can edit."</p>
{/if}
```

`{#elif}` and `{#else}` are optional; the first matching branch wins.

### `{#match}` — many branches

```html
{#match role}
  {#case admin}
    <p>"Full control."</p>
  {#case editor}
    <p>"Create and edit."</p>
  {#default}
    <p>"View only."</p>
{/match}
```

The first `{#case}` whose value equals the `{#match}` expression wins;
`{#default}` runs if none match. A `{#case}` body runs until the next
`{#case}`, `{#default}`, or `{/match}` — there is no per-case closer.

### Notes

- **Compile-time only.** The directives expand during transpilation, so the
  shipped page is static HTML — no client-side branching or loops.
- **`{{ var }}` is raw substitution.** It fills in before htmlx escapes quoted
  text nodes, so `<a href="#">"{{ label }}"</a>` still escapes the text once.
- **Bare names in headers, `{{ }}` in text.** Directive headers reference the
  loop variable bare (`{#match role}`, `{#if role == admin}`); body text
  interpolates it as `{{ role }}` — exactly the `.capyx` convention.
- **Composes with everything.** Nest control flow inside components, inside each
  other, or wrap component usage in a `{#for}`.

See [`demos/htmlx/control.htmlx`](../demos/htmlx/control.htmlx) for a page that
builds a nav with `{#for}`, gates a banner with `{#if}`, and renders one card
per role with a nested `{#for}` + `{#match}`.

---

## Reactive apps — `.capyx`

When a page needs **state and live updates** rather than static markup, reach
for `.capyx`. It is a single-file **reactive VHCO** format: dumb `component`
views, stateful `handler` units, `capability`/`provide` dependency injection,
and an optional shared `orchestrator` — all compiled to a fine-grained
**signals runtime** that performs surgical DOM updates (mutating one field
re-runs only the effects that read it).

```
handler Counter {
  state { count = 0 }
  on inc { count = count + 1 }
}

component CounterView(H) {
  <button { on:click = H.inc }>"+"</button>
  <span>{{ H.count }}</span>
}

mount CounterView with Counter
```

`{{ expr }}` is a reactive text binding, `{ on:click = … }` wires events,
`{#for}` / `{#if}` / `{#match}` are dynamic regions, and `{ bind:value = … }`
is a two-way model. The full format — the three-brace rule, every top-level
construct, the handler mini-language, capability/provide injection and the
orchestrator shared store — is documented in
[**`docs/capyx-reactive-vhco.md`**](./capyx-reactive-vhco.md), with 25 runnable
demos in [`demos/capyx/`](../demos/capyx/).

Because `.capyx` components are isolated (dumb views + handlers + mockable
capabilities), each one can be tested on its own — mounted alone, state
overridden, services mocked, events fired, expectations asserted — with **no
browser and no Selenium**. Author scenarios in the `.capytest` format and run
them with `window test suite.capytest`, or explore a component by hand with
`window test --ui app.capyx`. See
[**`docs/capyx-testing.md`**](./capyx-testing.md).

---

## Where things live

| | |
|---|---|
| `.htmlx` grammar | [`assets/htmlx.capy`](../assets/htmlx.capy) |
| `.htmlx` component preprocessor | [`infra/htmlx_components.go`](../infra/htmlx_components.go) |
| `.htmlx` control-flow preprocessor | [`infra/htmlx_controlflow.go`](../infra/htmlx_controlflow.go) |
| CapyScript grammar | [`assets/capyscript.capy`](../assets/capyscript.capy) |
| `.capyx` compiler | [`infra/capyx*.go`](../infra) |
| `.capyx` runtime | [`assets/capyx_runtime.js`](../assets/capyx_runtime.js) |
| HTML demos | [`demos/htmlx/`](../demos/htmlx/) |
| CapyScript demos | [`demos/cs/`](../demos/cs/) |
| `.capyx` demos | [`demos/capyx/`](../demos/capyx/) |
| Transpiler engine | [Capy](https://github.com/olivierdevelops/capy) |
