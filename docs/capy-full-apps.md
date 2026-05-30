# Full `window` apps with Capy — every feature, by example

> One source file, one binary. This cookbook shows how a Capy library
> abstracts the **UI, the event handling, the state, and the config** of a
> `window` app — and, when an app truly needs server-side logic, generates
> a **Go handler that runs inside the `window` process itself**. No Python,
> no Node, no socket, no sidecar. Minimal management: one language for any
> backend (Go), and most apps need no backend at all.
>
> Read [using-capy.md](using-capy.md) first (install + the model). The
> authoritative grammar/API is the
> [Capy integration guide](https://github.com/olivierdevelops/capy).

---

## Table of contents

- [1. The model](#1-the-model)
- [2. The base library](#2-the-base-library)
- **Core trio**
  - [3. Building UI](#3-building-ui)
  - [4. Event handling](#4-event-handling)
  - [5. State management](#5-state-management)
- [6. In-process Go handlers](#6-in-process-go-handlers)
- **Native capabilities (frontend, no backend)**
  - [7. `fs` — Notepad](#7-fs-notepad) · [8. `os` — Runner](#8-os-command-runner) · [9. `dialogs`](#9-dialogs-open-save) · [10. `canvas` — Sketch](#10-canvas-sketch)
  - [11. `camera`](#11-camera-photo-booth) · [12. `mic`](#12-mic-voice-recorder) · [13. `speech`](#13-speech-read-aloud-dictation) · [14. `screen`](#14-screen-screen-recorder) · [15. `input`](#15-input-command-palette)
- [16. `js_inject` — Chart.js](#16-js_inject-chartjs-dashboard)
- [17. Run modes: url / proxy / browser](#17-run-modes-url-proxy-browser)
- [18. Multi-window — in-process](#18-multi-window-in-process)
- [19. macOS app bundle](#19-macos-app-bundle)
- [20. A full app: Media Studio](#20-a-full-app-media-studio)
- [Appendix — the complete `window.capy`](#appendix-the-complete-windowcapy)

---

## 1. The model

Every statement you write projects into the app's files. UI and behavior go
to the frontend; config goes to `window.yaml`; and **only if** an app needs
server-side logic does Capy emit a Go handler that is compiled into the
`window` binary:

```
   state count = 0
   on "#inc" click do increment
   action increment : count = count + 1
        │
        ├─▶ static/index.html   markup
        └─▶ static/app.js        store field + listener + transition  (no backend)

   handler search                # only when logic must be server-side
        │
        └─▶ app_handlers.go       a Go func in THIS process (no sidecar)
```

> **Rule:** never generate a separate backend. UI, events, and state are
> frontend JS; any real backend logic is a Go function in the `window`
> process. If something's missing to support that, add it to the Go process
> — not to a new service.

The two tiers, and most apps stop at the first:

| Tier | Lives in | Generated artifact |
|------|----------|--------------------|
| **Frontend** | the page | `static/index.html`, `static/app.js` |
| **In-process Go** | the `window` binary | `app_handlers.go` (compiled in) |

---

## 2. The base library

A shared `context` accumulates what statements declare; `file` blocks read
it back. Full listing in the [appendix](#appendix-the-complete-windowcapy);
the trimmed core:

```
extension yaml
comments
    line "#"
end

context
    title    "App"
    width    480
    height   640
    mode     ""
    body     []         # HTML fragments
    state    []         # {name, value} reactive store fields
    transitions []      # {name, expr} frontend state actions
    calls    []         # {name, args, bind} in-process Go handler calls
    events   []         # {selector, type, action} DOM bindings
    setup    []         # raw JS init lines (native setup, etc.)
    features []         # native_features
    handlers []         # in-process Go handler names
    inject   []         # js_inject URLs
end

function app
    arg literal "app"
    arg capture title string
    arg capture w int default "480"
    arg capture h int default "640"
    block_closer end
    set context.title title
    set context.width w
    set context.height h
end
function end
end
```

Run any app:

```sh
window app.window     # `.window` is a built-in filetype: transpile + run, one binary
```

---

## 3. Building UI

UI verbs append HTML to `context.body`. The library author designs the
vocabulary; here are the staples.

```
function heading
    arg literal "heading"
    arg capture text string
    append context.body "<h1>${escapeHtml (unquote text)}</h1>"
end

function text
    arg literal "text"
    arg capture bind ident
    append context.body "<span data-bind=\"${bind}\"></span>"
end

function list
    arg literal "list"
    arg capture bind ident
    append context.body "<ul data-bind=\"${bind}\"></ul>"
end

function field
    arg literal "field"
    arg capture id ident
    arg literal "placeholder"
    arg capture ph string
    append context.body "<input id=\"${id}\" placeholder=${toQuoted (unquote ph)}>"
end

function button
    arg literal "button"
    arg capture label string
    arg literal "id"
    arg capture id ident
    append context.body "<button id=\"${id}\">${escapeHtml (unquote label)}</button>"
end
```

**Source**
```
app "Todo" 420 520
    heading "My Todos"
    field draft placeholder "New todo…"
    button "Add" id add
    list todos
end
```

**Generated `static/index.html` (body)**
```html
<h1>My Todos</h1>
<input id="draft" placeholder="New todo…">
<button id="add">Add</button>
<ul data-bind="todos"></ul>
```

`data-bind` ties elements to the store; `text` renders a scalar, `list`
renders an array. `escapeHtml (unquote …)` is the XSS guard.

---

## 4. Event handling

Two kinds, both abstracted:

**DOM events** — `on <selector> <type> do <action>` records a binding the
`app.js` block turns into an `addEventListener` calling the named action.

```
function on
    arg literal "on"
    arg capture sel string
    arg capture type ident
    arg literal "do"
    arg capture action ident
    append context.events {selector: sel, type: type, action: action}
end
```

**Push events** — anything the Go side dispatches (a file-watch tick, a
value from an in-process handler) arrives as a window event; a bound element
updates from its payload. `NATIVE.fs.watchFile`, for instance, already
pushes events the page can subscribe to.

**Source (with the Todo UI above)**
```
    on "#add" click do add_todo
    on "#draft" keydown do maybe_add
```

**Generated `static/app.js`**
```js
document.querySelector("#add").addEventListener("click", () => ACTIONS.add_todo());
document.querySelector("#draft").addEventListener("keydown", e => ACTIONS.maybe_add(e));
```

The wiring is generated; an action's *behavior* is defined once (next
section) and never duplicated across markup and script.

---

## 5. State management

This is where "minimal management" pays off: a stateful app with **no
backend** — the state is a small reactive store generated into `app.js`.

```
function state
    arg literal "state"
    arg capture name ident
    arg literal "="
    arg capture value any
    append context.state {name: name, value: value}
end

# action <name> set <field> = <expr>     (expr is JS evaluated over `state`)
function action
    arg literal "action"
    arg capture name  ident
    arg literal "set"
    arg capture field ident
    arg literal "="
    arg capture expr  tail
    append context.transitions {name: name, field: field, expr: expr}
end
```

**Source — a complete to-do app, zero backend**
```
app "Todo" 420 520
    state todos = []
    state draft = ""
    field draft placeholder "New todo…"
    button "Add" id add
    list todos
    on "#add" click do add_todo
    action add_todo set todos = [...state.todos, document.getElementById('draft').value]
end
```

**Generated `static/app.js`**
```js
const state = { todos: [], draft: "" };
function render(){
  document.querySelectorAll("[data-bind]").forEach(el => {
    const v = state[el.dataset.bind];
    if (Array.isArray(v)) el.innerHTML = v.map(x => `<li>${x}</li>`).join("");
    else el.textContent = v;
  });
}
function setState(patch){ Object.assign(state, patch); render(); }

const ACTIONS = {
  add_todo: () => setState({ todos: [...state.todos, document.getElementById('draft').value] }),
};

document.querySelector("#add").addEventListener("click", () => ACTIONS.add_todo());
render();
```

**Result:** a reactive to-do list — store, bindings, action, and event
wiring — generated entirely into the page. One `window` binary, no extra
process. Pair it with `persist` (§7) to save to disk via `NATIVE.fs`, still
with no backend.

---

## 6. In-process Go handlers

When logic must run server-side (a database, secrets, heavy compute), it's a
**Go function inside the `window` process** — declared with `handler`, called
from the frontend through `BACKEND.call`, dispatched in-process by a small
registry. No subprocess, no socket, no `run_backend_script`.

```
function handler
    arg literal "handler"
    arg capture name ident
    append context.handlers name
end

# call <handler> with <field> into <bind>
function call
    arg literal "call"
    arg capture name  ident
    arg literal "with"
    arg capture field ident
    arg literal "into"
    arg capture bind  ident
    append context.calls {name: name, field: field, bind: bind}
end
```

**Source**
```
app "Search" 600 480
    state rows = []
    field q placeholder "Query…"
    button "Search" id go
    list rows
    handler search
    on "#go" click do run_search
    call search with q into rows
end
```

**Generated frontend (`static/app.js`)**
```js
ACTIONS.run_search = () =>
  BACKEND.call("search", { q: document.getElementById("q").value },
    ({ data }) => setState({ rows: data.rows }));
```

**Generated in-process Go (`app_handlers.go`, compiled into the binary)**
```go
package main

// registerAppHandlers wires Capy-generated handlers into THIS process.
func registerAppHandlers(app *AppHandlers) {
	app.Handle("search", func(req map[string]any) (map[string]any, error) {
		// TODO: real Go logic, in-process (db, compute, network)
		return map[string]any{"rows": []any{}}, nil
	})
}
```

The generated `window.yaml` has **no** `run_backend_script`. The handler
ships inside the binary; `BACKEND.call("search", …)` dispatches to the
registry locally. If `window` is missing something to support a handler, you
add it here, in the Go process — never as a separate service.

> **Reach for `NATIVE.*` first.** The device features below are already
> in-process Go bindings exposed to JS. Read a file with `NATIVE.fs.readFile`
> directly from the page; you rarely need a custom Go handler at all.

---

## 7. `fs` — Notepad

Native capabilities are one-word component verbs: markup + `NATIVE.*` setup
+ the `native_features` entry. All frontend; no handler, no backend.

```
function notepad
    arg literal "notepad"
    arg capture path string
    append context.body "<textarea id=\"np\"></textarea><button id=\"np-save\">Save</button>"
    append context.setup "(async()=>{try{np.value=await NATIVE.fs.readFile(${asString path});}catch(e){}})();"
    append context.setup "np_save.onclick=()=>NATIVE.fs.writeFile(${asString path},np.value);"
    append context.features "fs"
end
```

**Source**  `app "Notepad" 520 420 \n notepad "/tmp/note.txt" \n end`
→ `window.yaml` gets `native_features: [fs]`; the page loads/saves a real
file with **no backend process**.

---

## 8. `os` — Command runner

```
function runner
    arg literal "runner"
    append context.body "<input id=\"rn\" placeholder=\"command\"><button id=\"rn-go\">Run</button><pre id=\"rn-out\"></pre>"
    append context.setup "rn_go.onclick=async()=>{const[c,...a]=rn.value.split(' ');const r=await NATIVE.os.exec(c,a);rn_out.textContent=r.stdout+r.stderr;};"
    append context.features "os"
end
```
**Source** `runner` → run `git status` from the UI; `native_features: [os]`.

---

## 9. `dialogs` — Open & save

```
function filepicker
    arg literal "filepicker"
    append context.body "<button id=\"fp\">Open…</button><span data-bind=\"picked\"></span>"
    append context.setup "fp.onclick=async()=>{const[p]=await NATIVE.dialogs.openFile({title:'Choose a file'});setState({picked:p||'(cancelled)'});};"
    append context.features "dialogs"
end
```
**Source** `state picked = ""` + `filepicker` → a native OS open dialog; the
chosen path lands in the store and renders. Pairs with `notepad`.

---

## 10. `canvas` — Sketch

```
function sketch
    arg literal "sketch"
    arg capture id ident
    append context.body "<canvas id=\"${id}\" width=\"400\" height=\"300\"></canvas>"
    append context.setup "let dn=false;${id}.onmousedown=()=>dn=true;${id}.onmouseup=()=>dn=false;${id}.onmousemove=e=>{if(dn)NATIVE.canvas.drawRect({canvas_id:'${id}',x:e.offsetX,y:e.offsetY,w:3,h:3,color:'#7c7cff',fill:true});};"
    append context.features "canvas"
end
```
**Source** `sketch pad` → drag to draw via `NATIVE.canvas`.

---

## 11. `camera` — Photo booth

```
function photobooth
    arg literal "photobooth"
    append context.body "<video id=\"pb-cam\" autoplay muted></video><button id=\"pb-shoot\">Take photo</button><img id=\"pb-shot\">"
    append context.setup "NATIVE.camera.stream({into:'#pb-cam'});"
    append context.setup "pb_shoot.onclick=async()=>{pb_shot.src=await NATIVE.camera.snapshot();};"
    append context.features "camera"
end
```
**Source** `photobooth` → live preview + capture, from one word.
`native_features: [camera]`.

---

## 12. `mic` — Voice recorder

```
function recorder
    arg literal "recorder"
    append context.body "<button id=\"rec\">Record</button><audio id=\"play\" controls></audio>"
    append context.setup "let h=null;rec.onclick=async()=>{if(!h){h=await NATIVE.mic.record();rec.textContent='Stop';}else{play.src=await h.stop();h=null;rec.textContent='Record';}};"
    append context.features "mic"
end
```
**Source** `recorder` → toggle to record; clip loads into `<audio>`.

---

## 13. `speech` — Read-aloud & dictation

```
function dictate
    arg literal "dictate"
    append context.body "<textarea id=\"dc\"></textarea><button id=\"dc-l\">Dictate</button><button id=\"dc-s\">Read aloud</button>"
    append context.setup "dc_l.onclick=()=>NATIVE.speech.listen(r=>{if(r.isFinal)dc.value+=r.text+' ';});"
    append context.setup "dc_s.onclick=()=>NATIVE.speech.say(dc.value);"
    append context.features "speech"
end
```
**Source** `dictate` → speak to fill the textarea (STT) and read it back (TTS).

---

## 14. `screen` — Screen recorder

```
function screencast
    arg literal "screencast"
    append context.body "<button id=\"sc\">Record screen</button><a id=\"sc-dl\" download=\"capture.webm\">Download</a>"
    append context.setup "let h=null;sc.onclick=async()=>{if(!h){h=await NATIVE.screen.record({audio:true});sc.textContent='Stop';}else{sc_dl.href=await h.stop();h=null;sc.textContent='Record screen';}};"
    append context.features "screen"
end
```
**Source** `screencast` → OS picker, then a downloadable `.webm`.

---

## 15. `input` — Command palette

```
function palette
    arg literal "palette"
    arg capture combo string
    append context.body "<div id=\"pal\" hidden>Command palette</div>"
    append context.setup "NATIVE.input.hotkey(${asString combo},()=>pal.hidden=!pal.hidden);"
    append context.setup "NATIVE.input.onKey(e=>{if(e.key==='Escape')pal.hidden=true;});"
    append context.features "input"
end
```
**Source** `palette "ctrl+k"` → Ctrl/⌘+K toggles a palette, Esc closes.

---

## 16. `js_inject` — Chart.js dashboard

Add a CDN library to `js_inject` (loaded before your page) and use it as a
global. Still no backend.

```
function chart
    arg literal "chart"
    arg capture id ident
    append context.inject "https://cdn.jsdelivr.net/npm/chart.js"
    append context.body "<canvas id=\"${id}\"></canvas>"
    append context.setup "new Chart(${id},{type:'line',data:{labels:['a','b','c'],datasets:[{data:[3,7,5]}]}});"
end
```
**Generated `window.yaml`**
```yaml
js_inject:
  - "https://cdn.jsdelivr.net/npm/chart.js"
```
**Source** `chart sales` → `Chart` is global before your code; the chart
renders. No bundler, no npm.

---

## 17. Run modes: url / proxy / browser

A `mode` family sets `context.mode` and the matching `window.yaml` fields —
config only, no generated backend.

```
function open_url
    arg literal "open"
    arg literal "url"
    arg capture u string
    set context.mode "url"
    set context.url u
end

function proxy
    arg literal "proxy"
    arg capture target string
    arg literal "via"
    arg capture cmd tail
    set context.mode "proxy"
    set context.proxy_target target
    set context.proxy_command cmd
end

function overlay
    arg literal "overlay"
    arg capture u string
    arg literal "with"
    arg capture html_path string
    set context.mode "browser"
    set context.url u
    set context.inject_html html_path
end
```

| Source | Generated `window.yaml` |
|--------|--------------------------|
| `open url "https://example.com"` | `mode: url` / `url: "…"` |
| `proxy "http://localhost:5173" via npm run dev` | `mode: proxy` / `proxy_target` / `proxy_command` |
| `overlay "https://news.ycombinator.com" with "./overlay.html"` | `mode: browser` / `url` / `inject_html` |

`proxy` is the "wrap an existing dev server" mode — note the wrapped server
is *your* app, separate from `window`; `window` itself still runs as one
process.

---

## 18. Multi-window — in-process

Need a second window? The `window` process opens it directly — no controller
subprocess. A `spawn` statement records a window the Go process creates on
launch (window creation is already an in-process capability,
`Windowing.New`).

```
function spawn
    arg literal "spawn"
    arg capture title string
    arg literal "->"
    arg capture url string
    append context.windows {title: title, url: url}
end
```

**Source**
```
app "Studio" 800 600
    spawn "Preview" -> "http://127.0.0.1:PORT/preview"
end
```

**Generated `app_handlers.go`**
```go
func registerAppHandlers(app *AppHandlers) {
	app.OnReady(func(w *WindowProcess) {
		w.OpenWindow("Preview", "http://127.0.0.1:"+w.Port()+"/preview", 480, 600)
	})
}
```

The extra window is opened from the **same** Go process — consistent with
the minimal-management rule. (This is the in-process counterpart to
`window`'s controlled mode; no separate `controlled_script` is generated.)

---

## 19. macOS app bundle

`bundle` fills the `mac_app` config so `window --mac-app` produces a `.app`.
Pure config.

```
function bundle
    arg literal "bundle"
    arg literal "icon"
    arg capture icon string
    set context.mac_icon icon
end
```
**Source** `bundle icon "./icon.icns"` → `mac_app: { icon: ./icon.icns }`;
`window --mac-app window.yaml` packages a double-clickable app. Because the
backend is in-process Go, there's **no extra binary to bundle** — just the
one `window` executable.

---

## 20. A full app: Media Studio

Combine many features. This captures a photo, dictates a caption, persists
it to disk, and asks an in-process Go handler to post-process — all from one
source, all in one binary.

**Source — `studio.window`**
```
app "Media Studio" 720 640
    state caption = ""
    photobooth                       # camera (frontend)
    dictate                          # speech (frontend)
    notepad "/tmp/caption.txt"       # fs (frontend)
    button "Stamp" id stamp
    handler stamp_photo              # in-process Go: heavy image work
    on "#stamp" click do run_stamp
    call stamp_photo with caption into status
    text status
end
```

**What Capy generates**

- `window.yaml`:
  ```yaml
  native_features: [camera, speech, fs]   # inferred from the verbs used
  ```
  …and **no** `run_backend_script`.
- `static/index.html` + `static/app.js`: the video, recorder, textarea,
  notepad, button; the `NATIVE.*` setup; the `{ caption: "", status: … }`
  store; and the `run_stamp` action calling the in-process handler.
- `app_handlers.go` (compiled into `window`):
  ```go
  func registerAppHandlers(app *AppHandlers) {
      app.Handle("stamp_photo", func(req map[string]any) (map[string]any, error) {
          // real Go image processing, in-process
          return map[string]any{"status": "stamped ✓"}, nil
      })
  }
  ```

**Result:** a multi-capability desktop app — camera, speech, files, reactive
state, and a Go handler for the heavy lifting — that is still **one process,
one language for the backend, nothing else to install or manage.**

---

## Appendix — the complete `window.capy`

The full library powering every section. It assembles four files from the
shared `context`. Note the **absence** of any backend-script generation: the
only backend artifact is `app_handlers.go`, compiled into the `window`
binary.

```
extension yaml
comments
    line "#"
end

context
    title "App"
    width 480
    height 640
    mode ""
    url ""
    proxy_target ""
    proxy_command ""
    inject_html ""
    mac_icon ""
    body []            # HTML fragments
    state []           # {name, value}
    transitions []     # {name, expr}      frontend actions
    calls []           # {name, field, bind}  in-process handler calls
    events []          # {selector, type, action}
    setup []           # raw JS lines
    features []        # native_features
    handlers []        # in-process Go handler names
    inject []          # js_inject URLs
    windows []         # {title, url}  extra windows (in-process)
end

# ── window + UI ──────────────────────────────────────────────────────
function app
    arg literal "app"
    arg capture title string
    arg capture w int default "480"
    arg capture h int default "640"
    block_closer end
    set context.title title
    set context.width w
    set context.height h
end

function heading
    arg literal "heading"
    arg capture text string
    append context.body "<h1>${escapeHtml (unquote text)}</h1>"
end

function text
    arg literal "text"
    arg capture bind ident
    append context.body "<span data-bind=\"${bind}\"></span>"
end

function list
    arg literal "list"
    arg capture bind ident
    append context.body "<ul data-bind=\"${bind}\"></ul>"
end

function field
    arg literal "field"
    arg capture id ident
    arg literal "placeholder"
    arg capture ph string
    append context.body "<input id=\"${id}\" placeholder=${toQuoted (unquote ph)}>"
end

function button
    arg literal "button"
    arg capture label string
    arg literal "id"
    arg capture id ident
    append context.body "<button id=\"${id}\">${escapeHtml (unquote label)}</button>"
end

# ── state + events ───────────────────────────────────────────────────
function state
    arg literal "state"
    arg capture name ident
    arg literal "="
    arg capture value any
    append context.state {name: name, value: value}
end

function action
    arg literal "action"
    arg capture name ident
    arg literal "set"
    arg capture field ident
    arg literal "="
    arg capture expr tail
    append context.transitions {name: name, field: field, expr: expr}
end

function on
    arg literal "on"
    arg capture sel string
    arg capture type ident
    arg literal "do"
    arg capture action ident
    append context.events {selector: sel, type: type, action: action}
end

# ── in-process Go handlers ───────────────────────────────────────────
function handler
    arg literal "handler"
    arg capture name ident
    append context.handlers name
end

function call
    arg literal "call"
    arg capture name ident
    arg literal "with"
    arg capture field ident
    arg literal "into"
    arg capture bind ident
    append context.calls {name: name, field: field, bind: bind}
end

# ── native components (frontend; each adds a native_feature) ─────────
function notepad
    arg literal "notepad"
    arg capture path string
    append context.body "<textarea id=\"np\"></textarea><button id=\"np-save\">Save</button>"
    append context.setup "(async()=>{try{np.value=await NATIVE.fs.readFile(${asString path});}catch(e){}})();"
    append context.setup "np_save.onclick=()=>NATIVE.fs.writeFile(${asString path},np.value);"
    append context.features "fs"
end

function runner
    arg literal "runner"
    append context.body "<input id=\"rn\" placeholder=\"command\"><button id=\"rn-go\">Run</button><pre id=\"rn-out\"></pre>"
    append context.setup "rn_go.onclick=async()=>{const[c,...a]=rn.value.split(' ');const r=await NATIVE.os.exec(c,a);rn_out.textContent=r.stdout+r.stderr;};"
    append context.features "os"
end

function filepicker
    arg literal "filepicker"
    append context.body "<button id=\"fp\">Open…</button><span data-bind=\"picked\"></span>"
    append context.setup "fp.onclick=async()=>{const[p]=await NATIVE.dialogs.openFile({title:'Choose a file'});setState({picked:p||'(cancelled)'});};"
    append context.features "dialogs"
end

function sketch
    arg literal "sketch"
    arg capture id ident
    append context.body "<canvas id=\"${id}\" width=\"400\" height=\"300\"></canvas>"
    append context.setup "let dn=false;${id}.onmousedown=()=>dn=true;${id}.onmouseup=()=>dn=false;${id}.onmousemove=e=>{if(dn)NATIVE.canvas.drawRect({canvas_id:'${id}',x:e.offsetX,y:e.offsetY,w:3,h:3,color:'#7c7cff',fill:true});};"
    append context.features "canvas"
end

function photobooth
    arg literal "photobooth"
    append context.body "<video id=\"pb-cam\" autoplay muted></video><button id=\"pb-shoot\">Take photo</button><img id=\"pb-shot\">"
    append context.setup "NATIVE.camera.stream({into:'#pb-cam'});"
    append context.setup "pb_shoot.onclick=async()=>{pb_shot.src=await NATIVE.camera.snapshot();};"
    append context.features "camera"
end

function recorder
    arg literal "recorder"
    append context.body "<button id=\"rec\">Record</button><audio id=\"play\" controls></audio>"
    append context.setup "let h=null;rec.onclick=async()=>{if(!h){h=await NATIVE.mic.record();rec.textContent='Stop';}else{play.src=await h.stop();h=null;rec.textContent='Record';}};"
    append context.features "mic"
end

function dictate
    arg literal "dictate"
    append context.body "<textarea id=\"dc\"></textarea><button id=\"dc-l\">Dictate</button><button id=\"dc-s\">Read aloud</button>"
    append context.setup "dc_l.onclick=()=>NATIVE.speech.listen(r=>{if(r.isFinal)dc.value+=r.text+' ';});"
    append context.setup "dc_s.onclick=()=>NATIVE.speech.say(dc.value);"
    append context.features "speech"
end

function screencast
    arg literal "screencast"
    append context.body "<button id=\"sc\">Record screen</button><a id=\"sc-dl\" download=\"capture.webm\">Download</a>"
    append context.setup "let h=null;sc.onclick=async()=>{if(!h){h=await NATIVE.screen.record({audio:true});sc.textContent='Stop';}else{sc_dl.href=await h.stop();h=null;sc.textContent='Record screen';}};"
    append context.features "screen"
end

function palette
    arg literal "palette"
    arg capture combo string
    append context.body "<div id=\"pal\" hidden>Command palette</div>"
    append context.setup "NATIVE.input.hotkey(${asString combo},()=>pal.hidden=!pal.hidden);"
    append context.setup "NATIVE.input.onKey(e=>{if(e.key==='Escape')pal.hidden=true;});"
    append context.features "input"
end

function chart
    arg literal "chart"
    arg capture id ident
    append context.inject "https://cdn.jsdelivr.net/npm/chart.js"
    append context.body "<canvas id=\"${id}\"></canvas>"
    append context.setup "new Chart(${id},{type:'line',data:{labels:['a','b','c'],datasets:[{data:[3,7,5]}]}});"
end

# ── run modes / multi-window / bundle ────────────────────────────────
function open_url
    arg literal "open"
    arg literal "url"
    arg capture u string
    set context.mode "url"
    set context.url u
end

function proxy
    arg literal "proxy"
    arg capture target string
    arg literal "via"
    arg capture cmd tail
    set context.mode "proxy"
    set context.proxy_target target
    set context.proxy_command cmd
end

function overlay
    arg literal "overlay"
    arg capture u string
    arg literal "with"
    arg capture html_path string
    set context.mode "browser"
    set context.url u
    set context.inject_html html_path
end

function spawn
    arg literal "spawn"
    arg capture title string
    arg literal "->"
    arg capture url string
    append context.windows {title: title, url: url}
end

function bundle
    arg literal "bundle"
    arg literal "icon"
    arg capture icon string
    set context.mac_icon icon
end

function end
end

# ── output: window.yaml (NO backend script — ever) ──────────────────
file "window.yaml"
    write `title: ${context.title}
`
    if context.mode
        write `mode: ${context.mode}
`
    end
    if context.url
        write `url: ${context.url}
`
    end
    if context.proxy_target
        write `proxy_target: ${context.proxy_target}
proxy_command: ${context.proxy_command}
`
    end
    if context.inject_html
        write `inject_html: ${context.inject_html}
`
    end
    if context.body
        write `entry_path: ./static/index.html
size:
  width: ${context.width}
  height: ${context.height}
`
    end
    if context.features
        write `native_features:
`
        for f in context.features
            write `  - ${f}
`
        end
    end
    if context.inject
        write `js_inject:
`
        for u in context.inject
            write `  - ${toQuoted u}
`
        end
    end
    if context.mac_icon
        write `mac_app:
  icon: ${context.mac_icon}
`
    end
end

# ── output: static/index.html ────────────────────────────────────────
file "static/index.html"
    write `<!doctype html>
<html><head><meta charset="utf-8"><title>${unquote context.title}</title></head>
<body><main id="app">
`
    for frag in context.body
        write `  ${frag}
`
    end
    write `</main>
<script src="/static/app.js"></script>
</body></html>
`
end

# ── output: static/app.js (store + events + actions + native setup) ──
file "static/app.js"
    write `const state = {
`
    for s in context.state
        write `  ${s.name}: ${toJSON s.value},
`
    end
    write `};
function render(){
  document.querySelectorAll("[data-bind]").forEach(el => {
    const v = state[el.dataset.bind];
    if (Array.isArray(v)) el.innerHTML = v.map(x => "<li>" + x + "</li>").join("");
    else el.textContent = v;
  });
}
function setState(patch){ Object.assign(state, patch); render(); }

const ACTIONS = {
`
    for t in context.transitions
        write `  ${t.name}: () => setState({ ${t.field}: ${t.expr} }),
`
    end
    for c in context.calls
        write `  run_${c.name}: () => BACKEND.call(${toQuoted c.name}, { ${c.field}: document.getElementById(${toQuoted c.field}).value }, ({ data }) => setState({ ${c.bind}: data.${c.bind} })),
`
    end
    write `};
`
    for e in context.events
        write `document.querySelector(${toQuoted e.selector}).addEventListener(${toQuoted e.type}, ev => ACTIONS.${e.action}(ev));
`
    end
    for line in context.setup
        write `${line}
`
    end
    write `render();
`
end

# ── output: app_handlers.go (in-process; only if handlers declared) ──
file "app_handlers.go"
    if context.handlers
        write `package main

// registerAppHandlers binds Capy-generated handlers into the window process.
// These run IN-PROCESS — no subprocess, no socket. Fill in the Go logic.
func registerAppHandlers(app *AppHandlers) {
`
        for h in context.handlers
            write `	app.Handle(${toQuoted h}, func(req map[string]any) (map[string]any, error) {
		return map[string]any{}, nil   // TODO: ${h}
	})
`
        end
        write `}
`
    end
end
```

> The inner-DSL helpers (`html`, `unquote`, `asString`, `toQuoted`, `toJSON`)
> and any cross-reference validation are the library author's design choice —
> see the [Capy integration guide](https://github.com/olivierdevelops/capy). The
> structural point stands: **one source → UI + events + state + config, with
> any backend logic as in-process Go in the single `window` binary.**
