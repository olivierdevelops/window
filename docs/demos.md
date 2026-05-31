# Demos

All demos live in `demos/`. Run any with:

```bash
window demos/<name>/window.yaml
```

---

## `hello` — Minimal static app

**Mode:** server (default) · **Backend:** none

The simplest possible app — a static HTML page, no backend, no native features. Good starting point for understanding the project structure.

```
demos/hello/
├── window.yaml
└── static/index.html
```

---

## `counter` — Python backend with pub/sub

**Mode:** server · **Backend:** Python

A counter app demonstrating full bidirectional communication:
- `+` / `−` buttons call `add` / `sub` handlers on the Python backend
- The backend pushes a live clock via `app.publish("timer", {...})`
- Uses the fixed `client.py` (async dispatch, `uuid` IDs)

```
demos/counter/
├── window.yaml
├── main.py          # Python backend
├── client.py        # reusable App / ResponseWriter classes
└── static/
    ├── index.html
    └── app.js       # BACKEND.call + BACKEND.onEvent usage
```

---

## `dashboard` — Chart.js via `js_inject`

**Mode:** server · **Backend:** none · **Features:** `js_inject`

Demonstrates `js_inject`: Chart.js is loaded from CDN and injected before page load. Three live-updating charts simulate CPU, memory, and network data using `setInterval`.

```yaml
js_inject:
  - "https://cdn.jsdelivr.net/npm/chart.js"
```

---

## `file_explorer` — Native file system

**Mode:** server · **Backend:** none · **Features:** `native_features: [fs]`

A two-pane file browser:
- Left: directory listing via `NATIVE.fs.readDir()`
- Right: file preview via `NATIVE.fs.readFile()`
- Navigate up/down, view text file contents, sorted with dirs first

No backend process runs — all file I/O goes through the native Go bindings.

---

## `terminal` — Native OS exec

**Mode:** server · **Backend:** none · **Features:** `native_features: [os]`

A minimal terminal emulator:
- Command input with history (↑/↓ arrows)
- `NATIVE.os.exec(cmd, args)` runs commands and returns stdout/stderr
- Platform badge shows `darwin` / `linux` / `windows` from `NATIVE.os.platform()`

---

## `multiwindow` — Controlled mode

**Mode:** controlled · **Backend:** Python (`controller.py`)

Python drives multiple windows:
1. Opens Window A and Window B simultaneously
2. Waits 2 seconds, then navigates Window A and evals JS in it
3. Closes Window B after 3 more seconds
4. Closes Window A

Demonstrates the full controlled-mode command set: `create_window`, `navigate`, `eval`, `close`.

> Best supported on Linux. macOS has threading constraints with multiple webview instances.

---

## `three` — Three.js 3D game

**Mode:** server · **Backend:** none

A Three.js-powered 3D maze/snake game. Uses locally bundled Three.js. Arrow keys or on-screen buttons for movement.

---

## `browser` — Browser mode overlay

**Mode:** browser · **URL:** YouTube

Injects a custom HTML UI as a closed shadow DOM overlay into an existing web page. The overlay is completely isolated from the host page's CSS and JS. Demonstrates the browser mode and `inject_html:` field.

---

## `simple_py` — Original counter (unmodified)

**Mode:** server · **Backend:** Python

The original counter demo included for reference. The `counter` demo is an improved version of this.

---

## `surla` — Static demo

**Mode:** server · **Backend:** none

A static demo app.

---

## `htmlx/*` — Matched-pair HTML apps (7 demos)

**Format:** `.htmlx` · **Backend:** none (static HTML)

Whole desktop apps written as **real, matched-pair HTML**
(`<tag>…</tag>`). The embedded `assets/htmlx.capy` library parses markup
with Capy sequence closers (nesting validated at transpile time); Go
preprocessors add `<component>` custom tags and compile-time
`{#for}` / `{#if}` / `{#match}` control flow (the same three-brace syntax as `.capyx`).

```bash
window demos/htmlx/hello.htmlx
window demos/htmlx/components.htmlx
window demos/htmlx/control.htmlx
```

| File | Shows |
|------|-------|
| `hello.htmlx` | Minimal app — smallest `.htmlx` |
| `landing.htmlx` | Cards in a responsive grid; badges; buttons |
| `article.htmlx` | Deep inline nesting (`<em>`, `<a>`, `<strong>`, `<blockquote>`) |
| `profile.htmlx` | Lists, badges, a card layout |
| `signup.htmlx` | Self-closing void elements: `<input />`, `<br />`, a `<form>` |
| `components.htmlx` | `<component>` kit: `<card>`, `<stat>`, `<badge>`, void `<avatar/>` |
| `control.htmlx` | Compile-time `{#for}` nav, `{#if}/{#else}` banner, nested `{#for}` + `{#match}` |

Format reference: [`docs/authoring-formats.md`](./authoring-formats.md). Public guide:
[site/htmlx.html](../site/htmlx.html). Tested by `go test ./demos/htmlx/`.

---

## `capyx/*` — Reactive VHCO apps (24 demos)

**Format:** `.capyx` · **Backend:** none (fine-grained signals runtime)

A whole catalogue of single-file **reactive** apps, run directly from a
`.capyx` source:

```bash
window demos/capyx/counter.capyx
```

They progress small → large: `hello`, `greeter`, `toggle`, `counter`,
`temperature`, `tip_calculator`, `color_picker`, `list_basic`, `star_rating`,
`tabs`, `accordion`, `login_form`, `wordcount`, `theme_demo`, `stopwatch`,
`todo`, `shopping_cart`, `quiz`, `kanban`, `calculator`, `dashboard`,
`two_lists`, `notes`, `orchestrator`. The last three demonstrate VHCO
dependency injection: mounting one component twice, `capability`/`provide`, and
an `orchestrator` shared reactive store.

Full catalogue: [`demos/capyx/README.md`](../demos/capyx/README.md). Format
reference: [`docs/capyx-reactive-vhco.md`](./capyx-reactive-vhco.md). Tested by
`go test ./demos/capyx/` (compile + mount + reactivity under a Node DOM shim).

---

## Feature matrix

| Demo | Mode | Backend | `native_features` | `js_inject` |
|------|------|---------|------------------|-------------|
| hello | server | — | — | — |
| counter | server | Python | — | — |
| dashboard | server | — | — | chart.js |
| file_explorer | server | — | fs | — |
| terminal | server | — | os | — |
| multiwindow | controlled | Python | — | — |
| three | server | — | — | — |
| browser | browser | Python | — | — |
| simple_py | server | Python | — | — |
| surla | server | — | — | — |

## Creating a new demo

```bash
window --init demos/myapp
# Scaffolds: demos/myapp/window.yaml + demos/myapp/static/index.html
```

Then edit `demos/myapp/window.yaml` and `demos/myapp/static/index.html`.
