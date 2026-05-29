"""
Controlled mode demo: Python drives two webview windows.
Requires: WINDOW_CONTROL_SOCK_PATH env var (set by Go automatically).
"""
import asyncio
import json
import os
import uuid


class WindowManager:
    def __init__(self):
        self.sock_path = os.getenv("WINDOW_CONTROL_SOCK_PATH")
        if not self.sock_path:
            raise RuntimeError("WINDOW_CONTROL_SOCK_PATH not set")
        self.reader = None
        self.writer = None
        self._pending: dict = {}

    async def connect(self):
        for attempt in range(20):
            try:
                self.reader, self.writer = await asyncio.open_unix_connection(self.sock_path)
                print(f"Connected to control socket (attempt {attempt + 1})")
                asyncio.create_task(self._recv_loop())
                return
            except (FileNotFoundError, ConnectionRefusedError):
                await asyncio.sleep(0.2)
        raise RuntimeError("Could not connect to control socket")

    async def _recv_loop(self):
        async for line in self.reader:
            try:
                msg = json.loads(line)
                req_id = msg.get("id")
                if req_id and req_id in self._pending:
                    fut = self._pending.pop(req_id)
                    if not fut.done():
                        fut.set_result(msg)
            except Exception as e:
                print(f"recv error: {e}")

    async def _send(self, cmd: dict) -> dict:
        req_id = uuid.uuid4().hex
        cmd["id"] = req_id
        fut = asyncio.get_event_loop().create_future()
        self._pending[req_id] = fut
        self.writer.write((json.dumps(cmd) + "\n").encode())
        await self.writer.drain()
        return await asyncio.wait_for(fut, timeout=5.0)

    async def create_window(self, title: str, url: str, width=800, height=600) -> str:
        r = await self._send({"cmd": "create_window", "params": {
            "title": title, "url": url, "width": width, "height": height
        }})
        if r.get("error"):
            raise RuntimeError(r["error"])
        return r["window_id"]

    async def navigate(self, window_id: str, url: str):
        r = await self._send({"cmd": "navigate", "window_id": window_id, "url": url})
        if r.get("error"):
            raise RuntimeError(r["error"])

    async def eval(self, window_id: str, js: str):
        r = await self._send({"cmd": "eval", "window_id": window_id, "js": js})
        if r.get("error"):
            raise RuntimeError(r["error"])

    async def close(self, window_id: str):
        r = await self._send({"cmd": "close", "window_id": window_id})
        if r.get("error"):
            raise RuntimeError(r["error"])


async def main():
    wm = WindowManager()
    await wm.connect()

    print("Opening two windows...")
    win1 = await wm.create_window("Window A", "https://example.com", 600, 500)
    win2 = await wm.create_window("Window B", "https://example.com", 600, 500)
    print(f"  win1={win1}  win2={win2}")

    await asyncio.sleep(2)

    print("Navigating Window A to a different page...")
    await wm.eval(win1, "document.title = 'Window A — controlled'")
    await wm.navigate(win1, "https://example.org")

    await asyncio.sleep(3)

    print("Closing Window B...")
    await wm.close(win2)

    await asyncio.sleep(2)
    print("Done — closing Window A.")
    await wm.close(win1)


if __name__ == "__main__":
    asyncio.run(main())
