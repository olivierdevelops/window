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
