# window.yaml Reference

Every `webview_gui` app is driven by a YAML (or JSON) config file. The binary loads this file, starts an HTTP server, and opens a native webview window.

## Minimal example

```yaml
title: "My App"
entry_path: ./static/index.html
static_dirs:
  "/static": ./static/
```

## Full reference

```yaml
# ── Identity ─────────────────────────────────────────────────────────────────

title: "My App"          # Window title bar text (default: "")
debug_mode: false         # Enable webview DevTools inspector (default: false)

# ── Window size ───────────────────────────────────────────────────────────────

size:
  width: 1024             # Pixels (default: 800)
  height: 768             # Pixels (default: 800)

# ── Static content ────────────────────────────────────────────────────────────

entry_path: ./static/index.html   # Served at GET /
html: "<h1>Hello</h1>"            # Inline HTML for GET /  (overrides entry_path)

static_dirs:                       # Map URL prefix → local directory
  "/static": ./static/
  "/assets": ./other/assets/

files:                             # Map URL path → local file
  "/favicon.ico": ./static/favicon.ico

# ── Run mode ─────────────────────────────────────────────────────────────────

mode: ""               # (default) local server + optional backend subprocess
# mode: url            # Navigate to a fixed URL
# mode: proxy          # Reverse-proxy to another server
# mode: browser        # Inject HTML overlay into an existing page
# mode: controlled     # Backend subprocess drives window management
# mode: wasm           # WASM module is the backend

# ── Backend subprocess (server / url / proxy modes) ──────────────────────────

run_backend_script: python3 main.py
# The subprocess receives WINDOW_SOCK_PATH env var pointing to a Unix socket.
# See docs/backend-protocol.md for the wire protocol.

# ── URL mode ──────────────────────────────────────────────────────────────────

url: "https://example.com"

# ── Proxy mode ────────────────────────────────────────────────────────────────

proxy_target: "http://localhost:${PORT}"   # env vars expanded
proxy_command: "npm run dev"               # optional: start target server

# ── Browser mode (overlay injection) ─────────────────────────────────────────

inject_html: ./overlay.html
# HTML injected into an existing page via a closed shadow DOM.
# Pairs with url: or proxy_target:.

# ── Controlled mode ───────────────────────────────────────────────────────────

controlled_script: python3 controller.py
# Subprocess receives WINDOW_CONTROL_SOCK_PATH env var.
# See docs/controlled-mode.md for the command protocol.

# ── WASM backend ─────────────────────────────────────────────────────────────

wasm_backend: ./app.wasm
# Must export: alloc(size i32) i32, handle(...) i64
# See docs/wasm-backend.md for the module contract.

# ── Native OS capabilities ────────────────────────────────────────────────────

native_features:
  - fs       # window.NATIVE.fs.*      — file read/write/watch
  - os       # window.NATIVE.os.*      — exec, env, platform info
  - dialogs  # window.NATIVE.dialogs.* — open/save/message dialogs
  - canvas   # window.NATIVE.canvas.*  — 2D canvas drawing primitives
# See docs/native-features.md

# ── JS library injection ──────────────────────────────────────────────────────

js_inject:
  - "https://cdn.jsdelivr.net/npm/chart.js"
  - "https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"
# Each entry becomes: <script src="..."> injected on every page load/navigation.

# ── macOS app bundle ──────────────────────────────────────────────────────────

mac_app:
  icon: ./icon.icns
  extra_binaries:
    - python3
  files:
    - ./static
  dirs:
    - ./assets
  env:
    MY_VAR: "value"
# Used by: window --mac-app window.yaml
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `WINDOW_PORT` | Fixed port for the local HTTP server (default: random free port) |
| `DEBUG=1` | Enable Go-side request logging to stdout |
| `DEV_MODE=1` | Server mode only: skip opening window (serve only) |

## Mode decision tree

```
mode unset/""  →  start local HTTP server, open window at http://127.0.0.1:<port>
mode: url      →  open window directly at cfg.url (no local server)
mode: proxy    →  reverse-proxy cfg.proxy_target, optionally start proxy_command first
mode: browser  →  open window at cfg.url, inject cfg.inject_html as shadow DOM overlay
mode: controlled → start cfg.controlled_script, accept window commands on control socket
mode: wasm     →  load cfg.wasm_backend via wazero, start local server, open window
```

## Notes

- All file paths in `window.yaml` are relative to the directory containing the file. The binary `chdir`s to that directory on startup.
- `html:` and `entry_path:` are mutually exclusive for the root path. `html:` takes precedence.
- `static_dirs` entries that don't end in `/` get a trailing slash appended automatically.
- `proxy_target` supports `${ENV_VAR}` substitution. If `port` is `0` or missing, a free port is chosen and exported as `$PORT`.
