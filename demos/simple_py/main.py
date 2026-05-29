from client import App, ResponseWriter
from datetime import datetime
import asyncio

app = App()


@app.handle("add")
async def add(req: dict, rw: ResponseWriter):
    value = req.get("value", 0)
    await rw.send({"value": value + 1})


@app.handle("sub")
async def sub(req: dict, rw: ResponseWriter):
    value = req.get("value", 0)
    await rw.send({"value": value - 1})


@app.handle("eval")
async def sub(req: dict, rw: ResponseWriter):
    await rw.eval("console.log('EVAL CALLED')")


async def background_time_printer():
    """Background task that prints the current time every second"""
    print("Background time printer started")
    try:
        while True:
            current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            # print(f"[Background] Current time: {current_time}")
            await app.publish("timer", {"current_time": current_time})
            await asyncio.sleep(0.5)  # Print every second

    except asyncio.CancelledError:
        print("Background time printer stopped")
    except Exception as e:
        print(f"Error in background time printer: {e}")
        

async def main():
    await app.run()
    # asyncio.create_task(background_time_printer())

    # Keep the application running
    try:
        await asyncio.Event().wait()
    except KeyboardInterrupt:
        print("\nShutting down...")
        await app.close()


if __name__ == "__main__":
    asyncio.run(main())