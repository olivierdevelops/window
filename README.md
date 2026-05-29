# webview_gui

Build native desktop apps with **any language** as the backend. Write your UI in HTML/CSS/JS and connect a backend in Python, Node.js, Rust, Go, or any other language over a simple socket protocol. Alternatively, ship a WebAssembly module as a zero-dependency backend, or use built-in native OS capabilities without a backend at all.

`webview_gui` is a single binary — `window` — that reads a `window.yaml` config and opens a native webview window.

---

## Quick start

```bash
# Scaffold a new project
window --init myapp
cd myapp

# Edit static/index.html, then run
window window.yaml
```

Or run any demo:

```bash
window demos/hello/window.yaml        # static HTML
window demos/dashboard/window.yaml    # Chart.js via js_inject
window demos/file_explorer/window.yaml # native file system
window demos/terminal/window.yaml     # native OS exec
window demos/counter/window.yaml      # Python backend
```

---

## Features

### Any-language backends

Connect a backend subprocess over a Unix socket using newline-delimited JSON. `webview_gui` provides a ready-made Python client library (`client.py`) but the protocol is simple enough to implement in any language.

```yaml
run_backend_script: python3 main.py
```

```python
from client import App, ResponseWriter
import asyncio

app = App()

@app.handle("greet")
async def greet(req, rw):
    await rw.send({"message": f"Hello, {req['name']}!"})

asyncio.run(app.run())
```

```js
// Frontend
BACKEND.call("greet", { name: "World" }, ({ data }) => {
  console.log(data.message) // Hello, World!
})
```

See [docs/backend-protocol.md](docs/backend-protocol.md) for the full protocol.

---

### Native OS capabilities

Expose file system, OS, dialog, and canvas APIs to the frontend — no backend process needed.

```yaml
native_features:
  - fs       # NATIVE.fs.readFile / writeFile / readDir / watchFile
  - os       # NATIVE.os.exec / getEnv / platform / info
  - dialogs  # NATIVE.dialogs.openFile / saveFile / showMessage
  - canvas   # NATIVE.canvas.drawRect / drawText / clear
```

```js
const entries = await window.NATIVE.fs.readDir("/home/user/Documents")
const result  = await window.NATIVE.os.exec("git", ["log", "--oneline"])
const [path]  = await window.NATIVE.dialogs.openFile({ title: "Open…" })
await window.NATIVE.canvas.drawRect({ canvas_id: "c", x:10, y:10, w:100, h:50, color:"#7c7cff", fill:true })
```

See [docs/native-features.md](docs/native-features.md).

---

### JS library injection

Load any JS library from CDN before your page — no HTML changes required.

```yaml
js_inject:
  - "https://cdn.jsdelivr.net/npm/chart.js"
  - "https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"
```

Libraries are injected via `webview.Init()` so they survive SPA navigation.

---

### WASM backend

Compile your backend to WebAssembly and ship it as a single binary — no subprocess, no socket, no runtime dependencies.

```yaml
mode: wasm
wasm_backend: ./app.wasm
```

The WASM module exposes two exports — `alloc` and `handle` — and receives `BACKEND.call()` requests as JSON. Powered by [wazero](https://wazero.io/) (pure Go, no CGo).

See [docs/wasm-backend.md](docs/wasm-backend.md).

---

### Controlled mode

Let the backend drive the window manager — create, navigate, and close windows programmatically.

```yaml
mode: controlled
controlled_script: python3 controller.py
```

```python
wm = WindowManager()
await wm.connect()
win = await wm.create_window("App", "https://example.com", 800, 600)
await asyncio.sleep(2)
await wm.eval(win, "document.body.style.background = 'navy'")
await wm.close(win)
```

See [docs/controlled-mode.md](docs/controlled-mode.md).

---

### Proxy mode

Forward requests to an existing dev server and open a native window pointing at it.

```yaml
mode: proxy
proxy_target: "http://localhost:${PORT}"
proxy_command: "npm run dev"     # optional: start the server first
```

---

### Browser mode

Inject a custom HTML overlay into any web page via a closed shadow DOM.

```yaml
mode: browser
url: "https://example.com"
inject_html: ./overlay.html
```

---

### macOS `.app` bundles

Package your app for distribution — no Go toolchain required on the target machine.

```bash
window --mac-app window.yaml
# → MyApp.app/
```

---

## Configuration

A minimal `window.yaml`:

```yaml
title: "My App"
entry_path: ./static/index.html
static_dirs:
  "/static": ./static/
```

Full reference: [docs/configuration.md](docs/configuration.md)

---

## Run modes

| `mode` | Description |
|--------|-------------|
| _(empty)_ | Local HTTP server + optional backend subprocess |
| `url` | Navigate to a fixed URL |
| `proxy` | Reverse-proxy to another server |
| `browser` | Inject HTML overlay into an existing page |
| `controlled` | Backend subprocess drives window management |
| `wasm` | WASM module is the backend |

---

## Demos

| Demo | What it shows |
|------|--------------|
| `hello` | Minimal static app |
| `counter` | Python backend, bidirectional calls, server push |
| `dashboard` | `js_inject` with Chart.js — live charts |
| `file_explorer` | `native_features: [fs]` — no backend needed |
| `terminal` | `native_features: [os]` — run shell commands |
| `multiwindow` | Controlled mode — Python opens/navigates/closes windows |
| `three` | Three.js 3D scene |
| `browser` | Browser mode overlay via shadow DOM |

Full descriptions: [docs/demos.md](docs/demos.md)

---

## Building from source

Requires Go 1.21+ and a C compiler (for `webview_go`).

```bash
git clone https://github.com/your-org/webview_gui
cd webview_gui
go build -o window .
```

**macOS:** Xcode command-line tools required.  
**Linux:** `libgtk-3-dev libwebkit2gtk-4.0-dev` (or `libwebkit2gtk-4.1-dev`).  
**Windows:** WebView2 Runtime required (ships with Edge/Windows 11).

---

## Project structure

```
webview_gui/
├── main.go                    10-line entry point
├── appio/                     CLI + config loading
├── domain/                    Pure data types
├── features/                  Capability interfaces (structs of functions)
├── infra/                     OS implementations
│   ├── native/                fs, os, dialogs
│   └── wasm/                  wazero runtime
├── orchestrator/              Wiring
│   ├── features/              Factory functions
│   └── usecases/              Run-mode functions
├── assets/                    Embedded JS + YAML
└── demos/                     Example apps
```

Architecture details: [docs/architecture.md](docs/architecture.md)

---

## Documentation

| Document | Contents |
|----------|----------|
| [docs/configuration.md](docs/configuration.md) | Full `window.yaml` reference |
| [docs/backend-protocol.md](docs/backend-protocol.md) | Socket IPC protocol, Python client, JS API |
| [docs/native-features.md](docs/native-features.md) | `NATIVE.fs`, `NATIVE.os`, `NATIVE.dialogs`, `NATIVE.canvas` |
| [docs/controlled-mode.md](docs/controlled-mode.md) | Controlled mode command protocol |
| [docs/wasm-backend.md](docs/wasm-backend.md) | WASM module contract, TinyGo example |
| [docs/demos.md](docs/demos.md) | Demo descriptions and feature matrix |
| [docs/cli.md](docs/cli.md) | CLI flags and environment variables |
| [docs/architecture.md](docs/architecture.md) | Package structure, VHCO layers, data flow |

---

## Version

`1.2.0`
