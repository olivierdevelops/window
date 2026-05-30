# Authoring a window app: HTML, JS, and custom components

`window` can build an app from three source formats, all transpiled by the
[Capy](https://github.com/olivierdevelops/capy) engine into the same thing: a
`window.yaml` + a `static/` frontend that the native webview runs.

| You write | File | Compiles to | Best for |
|---|---|---|---|
| Matched-pair HTML | `.htmlx` | a full HTML document | static pages, full layout control |
| A JS-like script | `.cs` | real JavaScript | algorithms, scratch logic |
| Custom HTML tags | `.htmlx` + `define` | your own components | reusable UI pieces |

Run any of them the same way:

```sh
window app.htmlx
window app.cs
```

---

## 1. HTML — `.htmlx`

A `.htmlx` file is a whole app written as **real, matched-pair HTML**:
`<tag>…</tag>`. The root `<app>` sets the window title + size and wraps
everything in a normalized, cross-platform document. Text nodes are
**quoted strings** (escaped once).

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

Built-in utility classes ship with the wrapper: `.card`, `.row`, `.grid`,
`.btn`, `.muted`, `.badge`.

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

- **`props="…"`** — space-separated named attributes, read inside as `{prop}`
  (escaped automatically)
- **`<slot></slot>`** — where the nested children go
- **`void`** — add the bare attribute for a self-closing tag

It's all HTML. No JavaScript, no Capy grammar to learn — `<component>`
expands to a real tag at transpile time.

### Define it once

```html
<component name="card" props="title">
  <section class="card">
    <h3>{title}</h3>
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
    <img src="https://ui-avatars.com/api/?name={name}" width="40" height="40"
         style="border-radius:50%" />
    <b>{name}</b>
  </div>
</component>

<avatar name="Ada" />
```

### Notes

- **Props are escaped.** `{prop}` renders the escaped attribute value, so
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

## 4. Control flow — `<for>`, `<if>`, `<switch>`

A `.htmlx` file can use **HTML-native control flow** that runs at transpile
time. There is no runtime data model — each construct is *evaluated while the
file compiles*, so the output is plain static HTML. They compose: an `<if>`
inside a `<for>` sees the loop variable.

Each control tag sits on its **own line** (indent freely). Loop variables and
values are referenced as `{name}` and substituted as raw text before the page
is escaped downstream.

### `<for>` — repeat a block

```html
<for each="Home, Docs, About" as="label">
  <a class="btn" href="#">"{label}"</a>
</for>
```

`each` is a comma-separated list; `as` names the loop variable. The block is
emitted once per item with `{label}` filled in.

### `<if>` / `<else>` — pick a branch

```html
<if value="admin" is="admin">
  <p>"Edit access granted."</p>
<else>
  <p>"Read-only."</p>
</if>
```

`value` is compared to `is`; the `<else>` branch is optional. To test
membership against a list, use `in` instead of `is`:

```html
<if value="{role}" in="admin, editor">
  <p>"Can edit."</p>
</if>
```

### `<switch>` — many branches

```html
<switch value="{role}">
  <case is="admin">  <p>"Full control."</p>      </case>
  <case is="editor"> <p>"Create and edit."</p>   </case>
  <default>          <p>"View only."</p>          </default>
</switch>
```

The first `<case>` whose `is` equals `value` wins; `<default>` runs if none
match.

### Notes

- **Compile-time only.** The constructs expand during transpilation, so the
  shipped page is static HTML — no client-side branching or loops.
- **`{var}` is raw substitution.** It fills in before htmlx escapes quoted text
  nodes, so `<a href="#">"{label}"</a>` still escapes the text once.
- **Composes with everything.** Nest control flow inside components, inside each
  other, or wrap component usage in a `<for>`.

See [`demos/htmlx/control.htmlx`](../demos/htmlx/control.htmlx) for a page that
builds a nav with `<for>`, gates a banner with `<if>`, and renders one card per
role with a nested `<for>` + `<switch>`.

---

## Where things live

| | |
|---|---|
| `.htmlx` grammar | [`assets/htmlx.capy`](../assets/htmlx.capy) |
| CapyScript grammar | [`assets/capyscript.capy`](../assets/capyscript.capy) |
| HTML demos | [`demos/htmlx/`](../demos/htmlx/) |
| CapyScript demos | [`demos/cs/`](../demos/cs/) |
| Transpiler engine | [Capy](https://github.com/olivierdevelops/capy) |
