# `.htmlx` demos — matched-pair angle-bracket HTML

A `.htmlx` file is a `window` app written as **real, matched-pair HTML**:
`<tag attr="v">…</tag>`. Before the Capy grammar runs, Go preprocessors expand
`<component>` custom tags and compile-time `{#for}` / `{#if}` / `{#match}` control
flow; then the embedded `assets/htmlx.capy` library parses the result with Capy
*sequence closers* (`block_close_seq "</" name ">"`), so:

- **One generic rule** handles every tag — the closer references the captured
  tag name, so `</div>` closes only `<div>`.
- **Nesting is validated.** A stray `</div>` where a `</p>` was expected is a
  hard transpile error, not a silent bug.
- The body is wrapped in a normalized, cross-platform HTML document
  (WebKit / WebView2 / WebKitGTK) with a built-in reset stylesheet.

## Run

```sh
window demos/htmlx/landing.htmlx
window demos/htmlx/components.htmlx
window demos/htmlx/control.htmlx
```

`window` detects the `.htmlx` extension, runs the preprocessors, transpiles
(window.yaml + static/index.html), and opens the app.

## The shape

```
<app title="Hello" width="460" height="320">
  <h1>"Hello, window"</h1>
  <p class="muted">"Written as angle-bracket HTML."</p>
  <a class="btn" href="https://example.com">"Docs"</a>
</app>
```

- The root `<app title="…" width="…" height="…">` sets the window title/size
  and wraps everything in a full HTML document. Attributes must appear in that
  **fixed order**: `title`, then `width`, then `height`.
- Any `<tag>…</tag>` nests freely; self-closing `<input />`, `<br />`, `<hr />`
  work too.
- **Text is quoted** (`"Hello"`). One quoted string is one token, escaped once,
  so you can even put literal markup like `"<p>…</p>"` inside `<code>`.
- **`#` line comments** are stripped before parse — handy above `<component>`
  blocks (see `components.htmlx`).

## Demos

| File | Shows |
|---|---|
| `hello.htmlx` | Minimal app — the smallest `.htmlx`. |
| `landing.htmlx` | Cards in a responsive grid; badges; buttons. |
| `article.htmlx` | Deep inline nesting (`<em>`, `<a>`, `<strong>`, `<blockquote>`). |
| `profile.htmlx` | Lists, badges, a card layout. |
| `signup.htmlx` | Self-closing void elements: `<input />`, `<br />`, a `<form>`. |
| `components.htmlx` | `<component>` kit: `<card>`, `<stat>`, `<badge>`, void `<avatar/>`. |
| `control.htmlx` | Compile-time `{#for}` nav, `{#if}/{#else}` banner, nested `{#for}` + `{#match}`. |

## Custom components

Define reusable tags with a `<component>` block, then use them like built-in HTML:

```
<component name="card" props="title">
  <section class="card"><h3>{{ title }}</h3><slot></slot></section>
</component>
```

- `props="a b"` — named attributes, read as `{{ a }}` / `{{ b }}` (escaped)
- `<slot></slot>` — where nested children go
- `void` — self-closing usage (`<avatar name="Ada" />`)

See `components.htmlx` for a full dashboard built from custom tags.

## Control flow

Compile-time directives use the same **three-brace** syntax as `.capyx` and
expand to static HTML during transpilation:

- `{#for x in a, b, c}` … `{/for}`
- `{#if x == y}` … `{#elif x == z}` … `{#else}` … `{/if}` (or `{#if x in a, b}` for membership)
- `{#match v}` `{#case a}` … `{#default}` … `{/match}`

Loop variables interpolate as `{{ x }}`. Each directive tag must sit on its
**own line**. See `control.htmlx`.

## Styling

The wrapper ships utility classes: `.card`, `.row`, `.grid`, `.btn`, `.muted`,
`.badge`. Combine them with inline `style="…"` attributes for one-offs.

## `.htmlx` vs `.window`

- **`.htmlx`** — you write the markup yourself, in HTML. Maximum control over
  structure; no state/binding layer.
- **`.window`** — a higher-level app language (widgets, state, events,
  in-process backend calls). See `demos/capy/`.

Both are transpiled by the Capy engine; both produce a plain `window` app.

Format reference: [`docs/authoring-formats.md`](../../docs/authoring-formats.md).
Public guide: [`site/htmlx.html`](../../site/htmlx.html).

Regression-tested by `go test ./demos/htmlx/`.
