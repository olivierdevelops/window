# Native Features

Enable native OS capabilities in the frontend without a backend subprocess by adding them to `native_features` in `window.yaml`.

```yaml
native_features:
  - fs
  - os
  - dialogs
  - canvas
```

Each feature binds Go functions into the webview and injects a JS wrapper that exposes them under `window.NATIVE.*`.

All `NATIVE.*` functions return **Promises**.

---

## `fs` — File System

```yaml
native_features:
  - fs
```

### API

```js
// Read a file as a UTF-8 string
const text = await window.NATIVE.fs.readFile("/path/to/file.txt")

// Write a file (perm defaults to 0o644 = 420)
await window.NATIVE.fs.writeFile("/path/to/file.txt", "hello", 420)

// List a directory → [{name, is_dir, size}]
const entries = await window.NATIVE.fs.readDir("/path/to/dir")
entries.forEach(e => console.log(e.name, e.is_dir, e.size))

// Watch a file for changes
window.NATIVE.fs.watchFile("/path/to/file.txt", (newContent) => {
  console.log("file changed:", newContent)
})
```

### `readDir` entry shape

```ts
{
  name: string    // filename only, no path
  is_dir: boolean
  size: number    // bytes (0 for directories)
}
```

### File watch events

`watchFile` internally calls `__native_fs_watchFile(path)` and registers a listener for the `CustomEvent` `native:fs:watch:<path>` dispatched by Go when the file is written or created. The event detail is `{ content: string }`.

---

## `os` — Operating System

```yaml
native_features:
  - os
```

### API

```js
// Execute a command and get stdout/stderr
const r = await window.NATIVE.os.exec("echo", ["hello", "world"])
console.log(r.stdout)  // "hello world\n"
console.log(r.stderr)  // ""
console.log(r.error)   // undefined if success, error string if failed

// Read an environment variable
const home = await window.NATIVE.os.getEnv("HOME")

// Get the current platform
const plat = await window.NATIVE.os.platform()
// → "darwin" | "linux" | "windows"

// Get OS + architecture info
const info = await window.NATIVE.os.info()
// → { os: "darwin", arch: "arm64" }
```

### `exec` result shape

```ts
{
  stdout: string
  stderr: string
  error?: string   // present only when the process exits non-zero
}
```

---

## `dialogs` — Native OS Dialogs

Uses [zenity](https://github.com/ncruces/zenity) (pure Go, no external deps at runtime).

```yaml
native_features:
  - dialogs
```

### API

```js
// Open file picker → string[] of selected paths
const paths = await window.NATIVE.dialogs.openFile({
  title: "Choose a file",
  multi: false,          // true to allow multiple selection
  filters: ["*.txt", "*.md"],
})

// Save file dialog → string path or empty string if cancelled
const savePath = await window.NATIVE.dialogs.saveFile({
  title: "Save as…",
})

// Show a notification/dialog
await window.NATIVE.dialogs.showMessage({
  title: "Done",
  message: "File saved successfully.",
  kind: "info",    // "info" | "warn" | "error"
})
```

---

## `canvas` — 2D Drawing Primitives

Exposes a command-driven 2D drawing API targeting an HTML5 `<canvas>` element. Useful when the backend (or Go side) needs to draw without JS charting libraries.

```yaml
native_features:
  - canvas
```

Your HTML must include a `<canvas>` with a known `id`:

```html
<canvas id="myCanvas" width="600" height="400"></canvas>
```

### API

```js
// Draw a filled or stroked rectangle
await window.NATIVE.canvas.drawRect({
  canvas_id: "myCanvas",
  x: 10, y: 10, w: 200, h: 100,
  color: "#7c7cff",
  fill: true,           // false → strokeRect
})

// Draw text
await window.NATIVE.canvas.drawText({
  canvas_id: "myCanvas",
  x: 10, y: 150,
  text: "Hello canvas",
  font: "20px sans-serif",   // CSS font string (default: "16px sans-serif")
  color: "#ffffff",
})

// Clear the entire canvas
await window.NATIVE.canvas.clear("myCanvas")

// flush() is a no-op in the current implementation (canvas draws immediately)
await window.NATIVE.canvas.flush("myCanvas")
```

### Backend-driven drawing

A backend subprocess can also issue canvas commands by calling the bound Go functions through the socket protocol. Each draw call is translated to a `window.Eval()` that runs the HTML5 Canvas 2D API directly.

---

## Using multiple features together

```yaml
native_features:
  - fs
  - os
  - dialogs
```

```js
document.getElementById("open-btn").onclick = async () => {
  const [path] = await window.NATIVE.dialogs.openFile({ title: "Open file" })
  if (!path) return
  const content = await window.NATIVE.fs.readFile(path)
  document.getElementById("editor").value = content
}

document.getElementById("run-btn").onclick = async () => {
  const r = await window.NATIVE.os.exec("wc", ["-l", "/tmp/file.txt"])
  document.getElementById("output").textContent = r.stdout
}
```

## When `NATIVE` is available

The bindings are registered via `webview.Init()`, which runs before the page's JS. You can use `NATIVE` immediately in inline `<script>` blocks or after `DOMContentLoaded`. If you're not sure when a library has loaded, poll:

```js
function waitForNative(fn) {
  if (window.NATIVE) { fn(); return; }
  setTimeout(() => waitForNative(fn), 50);
}
waitForNative(() => {
  window.NATIVE.os.platform().then(p => console.log("running on", p));
});
```
