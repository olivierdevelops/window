# Backend Protocol

When `run_backend_script` is set, `webview_gui` launches a subprocess and communicates with it over a **Unix domain socket** using newline-delimited JSON.

## Connection

1. Go creates a Unix socket at a temp path.
2. The subprocess is started with `WINDOW_SOCK_PATH` set to that path.
3. The subprocess connects to the socket.
4. Both sides exchange newline-delimited JSON messages.

On Windows, a named pipe is used instead of a Unix socket (`\\.\pipe\echo_<pid>.sock`).

## Message format

Every message — in both directions — is a single JSON object followed by `\n`.

```jsonc
{
  "id":       "uuid-or-server:uuid",   // request correlation ID
  "function": "myFunction",            // handler name
  "data":     { ... },                 // payload (any JSON object)
  "done":     true,                    // true when this is the final reply
  "type":     "data",                  // "data" | "eval"
  "code":     ""                       // JS to eval when type == "eval"
}
```

### Go → backend (requests)

Go sends a request when the frontend calls `BACKEND.call(name, params, cb)`:

```json
{"id": "550e8400-e29b-41d4-a716-446655440000", "function": "add", "data": {"value": 5}, "done": true}
```

### Backend → Go (replies)

The backend sends one or more replies for each request ID. The final reply must have `"done": true`.

```json
{"id": "550e8400-e29b-41d4-a716-446655440000", "function": "add", "data": {"value": 6}, "done": true}
```

### Backend → Go (server push)

The backend can push events to the frontend at any time by using an ID with the `server:` prefix:

```json
{"id": "server:abc123", "function": "timer", "data": {"current_time": "12:34:56"}, "done": true}
```

Go dispatches this as a `CustomEvent` named `"timer"` on `window`. The frontend listens with:

```js
BACKEND.onEvent("timer", ({ current_time }) => { ... })
// or equivalently:
window.addEventListener("timer", e => console.log(e.detail))
```

### Eval (backend → frontend JS)

The backend can execute arbitrary JS in the webview by sending a message with `"type": "eval"`:

```json
{"id": "server:xyz", "function": "noop", "type": "eval", "code": "document.title = 'hello'", "done": true}
```

## Frontend JS API

`backend.js` is injected into every page and provides a global `BACKEND` instance.

### `BACKEND.call(name, params, onReply)`

Calls a backend function. `onReply` is called once per reply chunk with `{ data, err }`.

```js
BACKEND.call("add", { value: 5 }, ({ data, err }) => {
  if (err) { console.error(err); return; }
  console.log(data.value); // 6
});
```

### `BACKEND.onEvent(eventId, callback)`

Subscribes to server-push events. Callback receives the event detail directly.

```js
BACKEND.onEvent("timer", ({ current_time }) => {
  document.getElementById("clock").textContent = current_time;
});
```

`console.log/warn/error/info` are automatically forwarded to Go's logger.

## Python client library (`client.py`)

Copy `demos/simple_py/client.py` into your project.

```python
from client import App, ResponseWriter
import asyncio

app = App()

@app.handle("greet")
async def greet(req: dict, rw: ResponseWriter):
    name = req.get("name", "World")
    await rw.send({"message": f"Hello, {name}!"})

async def main():
    await app.run()
    # optional: start background tasks here
    try:
        await asyncio.Event().wait()
    except KeyboardInterrupt:
        await app.close()

asyncio.run(main())
```

### `App` API

| Method | Description |
|--------|-------------|
| `app.handle(name)` | Decorator: register a handler for function `name` |
| `await app.run()` | Connect to socket, start receive loop |
| `await app.publish(name, data)` | Push event to frontend (ID gets `server:` prefix) |
| `await app.close()` | Gracefully close the socket |

### `ResponseWriter` API

Passed as the second argument to every handler.

| Method | Description |
|--------|-------------|
| `await rw.send(data)` | Send a reply dict to the frontend |
| `await rw.eval(js)` | Execute JS in the webview from the backend |

### Handler signature

Handlers can be sync or async functions. They receive `(req: dict, rw: ResponseWriter)`.

```python
@app.handle("add")
async def add(req, rw):
    await rw.send({"value": req["value"] + 1})
```

## Running the backend

The subprocess is started with `sh -c <script>` on Unix and `cmd.exe /C <script>` on Windows. You can use any language as long as it:

1. Reads `WINDOW_SOCK_PATH` from the environment.
2. Connects to the Unix socket.
3. Reads/writes newline-delimited JSON.

Example minimal Node.js backend:

```js
const net = require('net')
const path = process.env.WINDOW_SOCK_PATH
const client = net.connect(path, () => {
  client.on('data', buf => {
    const msg = JSON.parse(buf)
    const reply = { id: msg.id, function: msg.function, data: { result: 'ok' }, done: true }
    client.write(JSON.stringify(reply) + '\n')
  })
})
```
