# Controlled Mode

In controlled mode, the **backend process drives the window manager** — it can create, navigate, evaluate JS in, and destroy windows at will.

```yaml
mode: controlled
controlled_script: python3 controller.py
```

Go starts the subprocess and sets `WINDOW_CONTROL_SOCK_PATH` to a Unix socket path. The subprocess connects and sends window-management commands.

## Command protocol

All messages are newline-delimited JSON, same transport as the backend protocol.

### Go → backend (replies)

```jsonc
{
  "id": "req1",              // matches the command that triggered this reply
  "window_id": "win1",       // present for create_window and window-scoped commands
  "error": ""                // non-empty string if the command failed
}
```

### Backend → Go (commands)

#### `create_window`

```json
{
  "cmd": "create_window",
  "id": "req1",
  "params": {
    "title": "My Window",
    "url": "https://example.com",
    "width": 800,
    "height": 600
  }
}
```

Reply: `{"id": "req1", "window_id": "win1"}`

The window opens immediately and runs its event loop in a goroutine. `url` is optional; the window can be navigated later.

#### `navigate`

```json
{"cmd": "navigate", "id": "req2", "window_id": "win1", "url": "https://example.org"}
```

#### `eval`

```json
{"cmd": "eval", "id": "req3", "window_id": "win1", "js": "document.title = 'hi'"}
```

#### `close`

```json
{"cmd": "close", "id": "req4", "window_id": "win1"}
```

## Python controller example

Copy the `WindowManager` class from `demos/multiwindow/controller.py`:

```python
import asyncio, json, os, uuid

class WindowManager:
    def __init__(self):
        self.sock_path = os.getenv("WINDOW_CONTROL_SOCK_PATH")
        self.reader = self.writer = None
        self._pending = {}

    async def connect(self):
        for _ in range(20):
            try:
                self.reader, self.writer = await asyncio.open_unix_connection(self.sock_path)
                asyncio.create_task(self._recv_loop())
                return
            except (FileNotFoundError, ConnectionRefusedError):
                await asyncio.sleep(0.2)
        raise RuntimeError("could not connect to control socket")

    async def _recv_loop(self):
        async for line in self.reader:
            msg = json.loads(line)
            if (fut := self._pending.pop(msg.get("id"), None)):
                fut.set_result(msg)

    async def _send(self, cmd):
        cmd["id"] = uuid.uuid4().hex
        fut = asyncio.get_event_loop().create_future()
        self._pending[cmd["id"]] = fut
        self.writer.write((json.dumps(cmd) + "\n").encode())
        await self.writer.drain()
        return await asyncio.wait_for(fut, 5)

    async def create_window(self, title, url, width=800, height=600):
        r = await self._send({"cmd": "create_window",
                               "params": {"title": title, "url": url,
                                          "width": width, "height": height}})
        return r["window_id"]

    async def navigate(self, win_id, url):
        await self._send({"cmd": "navigate", "window_id": win_id, "url": url})

    async def eval(self, win_id, js):
        await self._send({"cmd": "eval", "window_id": win_id, "js": js})

    async def close(self, win_id):
        await self._send({"cmd": "close", "window_id": win_id})


async def main():
    wm = WindowManager()
    await wm.connect()

    win = await wm.create_window("Hello", "https://example.com", 700, 500)
    await asyncio.sleep(2)
    await wm.eval(win, "document.body.style.background = 'red'")
    await asyncio.sleep(2)
    await wm.close(win)

asyncio.run(main())
```

## Platform notes

- **Linux (GTK)**: Multiple simultaneous windows work correctly.
- **macOS (WKWebView)**: Cocoa requires UI on the main thread. Multiple windows created from goroutines may behave unexpectedly. The demo works best on Linux.
- **Windows (WebView2)**: Similar threading constraints to macOS.

## Environment variables

| Variable | Description |
|----------|-------------|
| `WINDOW_CONTROL_SOCK_PATH` | Set by Go automatically; read by the subprocess |
| `WINDOW_SOCK_PATH` | Not set in controlled mode (no backend socket) |
