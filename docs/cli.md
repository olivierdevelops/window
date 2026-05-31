# CLI Reference

## Usage

```
window [flags | path]
```

## Commands

### Run an app

```bash
window demos/hello/window.yaml       # from YAML config
window demos/hello/window.json       # from JSON config
window demos/htmlx/hello.htmlx         # matched-pair HTML (transpiled)
window demos/capy/counter.window       # .window app language (transpiled)
window demos/capyx/counter.capyx       # reactive VHCO app (compiled)
window demos/cs/fib.cs                 # CapyScript → JavaScript (transpiled)
window /path/to/document.md            # view a single file
window /path/to/image.png              # view a single file
```

**Authoring formats** (`.htmlx`, `.window`, `.capyx`, `.cs`) are transpiled or
compiled into a temp `window.yaml` + `static/` tree, then run like any other app.
See [authoring-formats.md](./authoring-formats.md).

When given a path that is not a recognized config or authoring format,
`webview_gui` creates a minimal single-file viewer config.

### `test` — test components in isolation

Run `.capyx` component tests with no browser and no Selenium.

```bash
window test demos/capyx/counter.capytest   # run a suite headlessly (CI; exits non-zero on failure)
window test --ui demos/capyx/counter.capyx  # open the interactive Test Bench
```

The headless runner needs Node.js on `PATH`; the `--ui` bench runs in the
webview and does not. See [capyx-testing.md](./capyx-testing.md) for the
`.capytest` format and the bench walkthrough.

### `--version` / `-v`

Print the version string and exit.

```bash
$ window --version
1.2.0
```

### `--init [dir]`

Scaffold a new project in the given directory (default: current directory).

```bash
window --init myapp
```

Creates:
```
myapp/
├── window.yaml          # template config with all fields documented
└── static/
    └── index.html       # starter HTML page
```

### `--mac-app [config]` (macOS only)

Build a self-contained `.app` bundle.

```bash
window --mac-app window.yaml
```

Reads the `mac_app:` section of the config and produces `<title>.app/` in the current directory. The bundle contains:
- The running `window` binary (no recompile needed)
- `window.yaml` and all files/dirs listed in `mac_app.files` / `mac_app.dirs`
- Any extra binaries (e.g. `python3`) from `mac_app.extra_binaries`
- An optional `.icns` icon
- `Info.plist` with bundle metadata
- A `launch` shell script that sets `$PATH` and environment variables

```yaml
mac_app:
  icon: icon.icns
  extra_binaries:
    - python3
  files:
    - ./static
    - ./window.yaml
  dirs:
    - ./assets
  env:
    MY_API_KEY: "secret"
```

## Environment variables

| Variable | Effect |
|----------|--------|
| `DEBUG=1` | Enable Go HTTP request logging to stdout |
| `DEV_MODE=1` | Server mode only: start the HTTP server but do not open a window |
| `WINDOW_PORT=<n>` | Use a specific port for the local HTTP server |
| `WINDOW_SOCK_PATH` | Set by Go; read by backend subprocesses |
| `WINDOW_CONTROL_SOCK_PATH` | Set by Go; read by controlled-mode subprocesses |

## Exit behaviour

- The process exits when the webview window is closed by the user.
- In controlled mode, the process exits when the controlled subprocess exits.
- In proxy mode with a `proxy_command`, the process exits when the proxy command exits.
