import os
from typing import Callable
import json
import asyncio
import json
import os
from typing import Callable, Dict

class ResponseWriter:
    def __init__(self, request_id: str, function: str, send: Callable):
        self.request_id = request_id
        self.function = function
        self.send = send
    
    def send(self, reply: dict):
        asyncio.create_task(self.send(self.function, reply, self.request_id))

class App:
    def __init__(self):
        self.path = os.getenv("WINDOW_SOCK_PATH")
        print(f"WINDOW_SOCK_PATH: {self.path}")
        if not self.path:
            raise RuntimeError("WINDOW_SOCK_PATH not set")

        # self.pending: Dict[str, asyncio.Future] = {}
        self.handlers: Dict[str, Callable] = {}

    async def run(self):
        print(f"@@@WINDOW_SOCK_PATH: {self.path}")

        self.reader, self.writer = await asyncio.open_unix_connection(self.path)
        asyncio.create_task(self._recv_loop())

    def handle(self, name):
        def wrap(fn):
            self.handlers[name] = fn
            return fn
        return wrap

    async def send(self, name, data, req_id=None):

        msg = {
            "id": req_id,
            "function": name,
            "data": data,
        }

        self.writer.write((json.dumps(msg) + "\n").encode())
        await self.writer.drain()

    async def _recv_loop(self):
        while True:
            line = await self.reader.readline()
            if not line:
                break

            msg: dict = json.loads(line.decode())
            req_id = msg.get("id")
            function = msg.get("function")

            if not req_id or not function:
                asyncio.create_task(app.send("server:error", {"error": f"missing 'id' or 'function'"}, req_id="server:error"))
                continue

            handler = self.handlers.get(function)
            if not handler:
                asyncio.create_task(app.send("server:error", {"error": f"missing handler for '{function=}'"}, req_id="server:error"))
                continue

            rw = ResponseWriter(req_id, function)
            
            asyncio.create_task(handler(msg.get("data"), rw))


    async def close(self):
        self.writer.close()
        await self.writer.wait_closed()


app = App()

@app.handle("add")
def add(req: dict, rw: ResponseWriter):
    rw.send({"data": req["value"] + 1})

@app.handle("sub")
def sub(req: dict, rw: ResponseWriter):
    rw.send({"data": req["value"] - 1})


async def main():
    await app.run()
 

if __name__ == "__main__":
    asyncio.run(main())

