import asyncio
from client import App, ResponseWriter

app = App()


@app.handle("add")
async def add(req: dict, rw: ResponseWriter):
    await rw.send({"value": req.get("value", 0) + 1})


@app.handle("sub")
async def sub(req: dict, rw: ResponseWriter):
    await rw.send({"value": req.get("value", 0) - 1})


async def timer_loop():
    from datetime import datetime
    while True:
        await app.publish("timer", {"current_time": datetime.now().strftime("%H:%M:%S")})
        await asyncio.sleep(1)


async def main():
    await app.run()
    asyncio.create_task(timer_loop())
    try:
        await asyncio.Event().wait()
    except KeyboardInterrupt:
        await app.close()


if __name__ == "__main__":
    asyncio.run(main())
