# Architecture

`webview_gui` follows the **VHCO** (Vertical Hierarchy, Closed-world Operations) pattern. Every package has a single, fixed responsibility and may only import packages *below* it in the hierarchy.

```
main.go
  └── appio/           ← CLI + config loading
  └── orchestrator/    ← wiring: builds features, runs use-cases
        ├── features/  ← factory functions (make_*.go)
        └── usecases/  ← run-mode functions (make_run_*.go)
              uses ↓
  └── features/        ← capability shapes (structs of functions)
  └── domain/          ← pure data types, no logic
  └── infra/           ← OS implementations, no domain logic
        ├── native/    ← fs, os, dialogs (native capabilities)
        └── wasm/      ← wazero WASM runtime
  └── assets/          ← embedded JS + YAML templates
```

## Layer rules

| Layer | May import | Must not import |
|-------|-----------|-----------------|
| `main` | `appio`, `orchestrator` | everything else |
| `appio` | `domain`, `infra`, `assets` | `features`, `orchestrator` |
| `orchestrator` | everything below | — |
| `features` | `domain`, `net/http` | `infra`, `orchestrator` |
| `infra` | `domain`, stdlib, third-party | `features`, `orchestrator`, `appio` |
| `domain` | stdlib only | everything above |

## Package tour

### `domain/`
Pure value types shared across the whole app. No business logic.

| File | Contents |
|------|----------|
| `config.go` | `AppConfig`, `WindowSize`, `RunMode` constants, `NativeFeature` constants, `MacAppConfig` |
| `message.go` | `Message` — the IPC envelope used by the backend socket protocol |

### `features/`
Each file defines a **capability struct** — a struct whose fields are function values. No implementations live here; the orchestrator wires them in.

| Struct | Capabilities |
|--------|-------------|
| `Windowing` | `New`, `Destroy`, `Navigate`, `Eval`, `Bind`, `Init`, `Run`, `SetTitle`, `SendEvent` |
| `BackendBridge` | `Start`, `HandleRequest`, `OnServerPush`, `Close` |
| `StaticServer` | `BuildMux`, `Listen`, `ResolvePort`, `WaitForServer` |
| `NativeFS` | `ReadFile`, `WriteFile`, `ReadDir`, `WatchFile` |
| `NativeOS` | `Exec`, `GetEnv`, `Platform`, `OSInfo` |
| `NativeDialogs` | `OpenFile`, `SaveFile`, `ShowMessage` |
| `NativeCanvas` | `DrawRect`, `DrawText`, `Clear`, `Flush` |
| `ControlledMode` | `StartManagementSocket` |

`WindowHandle` is defined as `type WindowHandle = any`. The concrete type (`*infra.WebviewHandle`) is only known to orchestrator code, which type-asserts when needed.

### `infra/`
Concrete implementations. Each file is independent and has no domain knowledge.

| File | Provides |
|------|----------|
| `webview_adapter.go` | `WebviewHandle`, thin wrappers around `webview_go` |
| `http_server.go` | `BuildStaticMux`, `ListenAndServe`, `ResolvePort`, `WaitForServer` |
| `socket_server.go` | `SocketServer` — Unix socket IPC with subprocess backends |
| `control_socket.go` | `ControlSocket` — management socket for controlled mode |
| `proc.go` | `GetRunScriptCMD`, `WriterWrapper` — subprocess launching |
| `log.go` | `LogMiddleware` — HTTP request logging |
| `cert.go` | `EnsureLocalCert`, `TrustCert`, `CopyFile`, `CopyDir` |
| `renderer.go` | `StringToHTML`, `RenderToHTML` — Markdown → HTML |
| `htmlx_components.go` | `RewriteHTMLXComponents` — `<component>` → Capy `define` blocks |
| `htmlx_controlflow.go` | `ExpandControlFlow` — compile-time `{#for}/{#if}/{#match}` |
| `capyx*.go` | `.capyx` reactive VHCO compiler |
| `mac_app.go` (darwin) | `BuildMacApp` — `.app` bundle builder |
| `native/fs.go` | `ReadFile`, `WriteFile`, `ReadDir`, `WatchFile` (fsnotify) |
| `native/os.go` | `ExecCommand`, `GetEnv`, `Platform`, `GetOSInfo` |
| `native/dialogs.go` | `OpenFile`, `SaveFile`, `ShowMessage` (zenity) |
| `wasm/runtime.go` | `Runtime` — wazero WASM loader and `Handle` dispatcher |

### `orchestrator/`
The only code that knows about both `features` and `infra`. All `make_*.go` files follow one pattern: build a feature struct by closing over infra functions.

```
orchestrator/
├── app.go                        Run(cfg) — dispatches by mode
├── features/
│   ├── make_windowing.go         MakeWindowing()
│   ├── make_backend_bridge.go    MakeBackendBridge()
│   ├── make_static_server.go     MakeStaticServer()
│   ├── make_controlled_mode.go   MakeControlledMode()
│   ├── make_native_fs.go         MakeNativeFS()
│   ├── make_native_os.go         MakeNativeOS()
│   ├── make_native_dialogs.go    MakeNativeDialogs()
│   └── make_native_canvas.go     MakeNativeCanvas(win, h)
└── usecases/
    ├── make_init_window.go       MakeInitWindow(win, bridge) → init fn
    ├── make_run_server_mode.go   MakeRunServerMode(…)
    ├── make_run_url_mode.go      MakeRunURLMode(…)
    ├── make_run_proxy_mode.go    MakeRunProxyMode(…)
    ├── make_run_browser_mode.go  MakeRunBrowserMode(…)
    ├── make_run_controlled_mode.go MakeRunControlledMode(…)
    └── make_run_wasm_mode.go     MakeRunWASMMode(…)
```

### `appio/`
CLI argument handling and config loading. Talks to `infra` for `BuildMacApp`, talks to `domain` for `AppConfig`.

| File | Responsibility |
|------|---------------|
| `cli.go` | `ParseCLI()` — flags, `--version`, `--init`, `--mac-app`; dispatches `.htmlx`, `.window`, `.capyx`, `.cs` to transpilers |
| `config_loader.go` | `LoadApp()`, `LoadAppForContentView()` |

### `assets/`
All embedded static files. `embed.go` declares Go variables for each asset.

| Asset | Used by |
|-------|---------|
| `htmlx.capy` | `.htmlx` — matched-pair HTML app language |
| `window.capy` | `.window` — declarative app language |
| `capyscript.capy` | `.cs` — CapyScript → JavaScript |
| `capyx_runtime.js` | `.capyx` — fine-grained signals runtime |
| `backend.js` | Injected into every window — provides `BACKEND` class and `console` forwarding |
| `browser.js` | Browser mode — shadow-DOM overlay template |
| `native_fs.js` | `native_features: [fs]` — exposes `window.NATIVE.fs.*` |
| `native_os.js` | `native_features: [os]` — exposes `window.NATIVE.os.*` |
| `native_dialogs.js` | `native_features: [dialogs]` — exposes `window.NATIVE.dialogs.*` |
| `native_canvas.js` | `native_features: [canvas]` — exposes `window.NATIVE.canvas.*` |
| `window_default.yaml` | Template written by `--init` |
| `default_index.html` | Template index written by `--init` |

## Request lifecycle (server mode)

```
User clicks button
  → JS: BACKEND.call("myFunc", {x: 1}, callback)
  → JS calls __CALL_BACKEND("myFunc", {x:1})  [bound by Go]
  → Go: BackendBridge.HandleRequest → writes JSON to Unix socket
  → Backend process reads line, calls handler, writes response
  → Go: reads response, calls win.SendEvent(h, eventID, data)
  → JS: CustomEvent fires on eventID, BACKEND calls callback({data})
```

## Trust script
A JavaScript snippet (`trustScript` in `make_windowing.go`) is injected via `webview.Init()` before every page load. It:
- Polyfills `navigator.clipboard` using `execCommand` (no permission prompts)
- Patches `navigator.permissions.query` to return `granted` for clipboard
- Intercepts Ctrl/Cmd+C/V/X and routes them through the clipboard polyfill
- Sets `window.isSecureContext = true`
- Polyfills `crypto.randomUUID` if absent
