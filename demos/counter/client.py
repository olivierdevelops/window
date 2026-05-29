import asyncio
import json
import os
import uuid
from typing import Callable, Dict


class ResponseWriter:
    def __init__(self, request_id: str, function: str, send_func: Callable):
        self.request_id = request_id
        self.function = function
        self._send_func = send_func
    
    async def send(self, reply: dict):
        await self._send_func(self.function, reply, self.request_id)

    async def eval(self, js_code: str):
        await self._send_func(self.function, None, self.request_id, eval=js_code)


class App:
    def __init__(self):
        self.path = os.getenv("WINDOW_SOCK_PATH")
        if not self.path:
            raise RuntimeError("WINDOW_SOCK_PATH not set")

        self.handlers: Dict[str, Callable] = {}
        self.reader = None
        self.writer = None

    async def run(self):

        # Retry connection in case Go server isn't ready yet
        max_attempts = 20
        for attempt in range(max_attempts):
            try:
                self.reader, self.writer = await asyncio.open_unix_connection(self.path)
                print(f"Connected to Unix socket on attempt {attempt + 1}")
                break
            except FileNotFoundError:
                if attempt < max_attempts - 1:
                    print(f"Socket not found, retrying... (attempt {attempt + 1}/{max_attempts})")
                    await asyncio.sleep(0.1)
                else:
                    raise RuntimeError(f"Failed to connect to socket after {max_attempts} attempts")
        
        asyncio.create_task(self._recv_loop())

    def handle(self, name):
        def wrap(fn):
            self.handlers[name] = fn
            return fn
        return wrap

    async def publish(self, name, data):
        return await self._send(name, data, req_id=f"server:{uuid.uuid4().hex}")

    async def _send(self, name, data, req_id=None, done=True, eval=None):
        if not self.writer:
            raise RuntimeError("Writer not initialized")

        msg = {
            "id": req_id,
            "function": name,
            "done": done  # Added done flag for Go side
        }

        if eval :
            msg["code"] = eval
            msg["data"] = {}
            msg["type"] = "eval"

        else:
            msg["data"] = data
            msg["type"] = "data"

        self.writer.write((json.dumps(msg) + "\n").encode())
        await self.writer.drain()

    async def _recv_loop(self):
        try:
            while True:
                line = await self.reader.readline()
                if not line:
                    print("Connection closed by server")
                    break

                try:
                    msg: dict = json.loads(line.decode())
                except json.JSONDecodeError as e:
                    print(f"JSON decode error: {e}")
                    continue

                req_id = msg.get("id")
                function = msg.get("function")

                if not req_id or not function:
                    await self._send("server:error", {"error": "missing 'id' or 'function'"}, req_id="server:error")
                    continue

                handler = self.handlers.get(function)
                if not handler:
                    await self._send("server:error", {"error": f"missing handler for function='{function}'"}, req_id="server:error")
                    continue

                rw = ResponseWriter(req_id, function, self._send)
                async def _dispatch(h=handler, d=msg.get("data", {}), r=rw, f=function, rid=req_id):
                    try:
                        result = h(d, r)
                        if asyncio.iscoroutine(result):
                            await result
                    except Exception as e:
                        print(f"Error in handler {f}: {e}")
                        await self._send(f, {"error": str(e)}, req_id=rid)
                asyncio.create_task(_dispatch())
        except Exception as e:
            print(f"Error in receive loop: {e}")

    async def close(self):
        if self.writer:
            self.writer.close()
            await self.writer.wait_closed()
