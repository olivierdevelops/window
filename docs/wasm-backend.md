# WASM Backend

Instead of a subprocess communicating over a socket, `webview_gui` can load a WebAssembly binary as the backend. The WASM module handles all `BACKEND.call()` requests synchronously. No subprocess, no socket, no IPC overhead.

```yaml
mode: wasm
wasm_backend: ./app.wasm
entry_path: ./static/index.html
static_dirs:
  "/static": ./static/
```

## Module contract

The WASM module must export exactly two functions:

### `alloc(size i32) i32`

Allocate `size` bytes in the module's linear memory and return the pointer. Go uses this to write function names and request data into the module before calling `handle`.

```wat
(func (export "alloc") (param $size i32) (result i32)
  ;; your allocator here — e.g. bump allocator or malloc
)
```

### `handle(fn_ptr i32, fn_len i32, data_ptr i32, data_len i32) i64`

Called once per `BACKEND.call()`. Parameters:

| Param | Type | Description |
|-------|------|-------------|
| `fn_ptr` | `i32` | Pointer to function name bytes in linear memory |
| `fn_len` | `i32` | Length of function name |
| `data_ptr` | `i32` | Pointer to JSON-encoded request data |
| `data_len` | `i32` | Length of JSON data |
| return | `i64` | Packed `resp_ptr << 32 \| resp_len` — pointer and length of JSON response |

The function name and data are UTF-8 / JSON strings. The response must be JSON-encoded and written into linear memory at `resp_ptr`.

### `init()` (optional)

If exported, called once when the module is loaded. Use for global initialization.

## TinyGo example

```go
package main

import (
    "encoding/json"
    "unsafe"
)

var buf [1 << 20]byte // 1 MB response buffer
var heap [1 << 20]byte
var heapOffset uintptr

//export alloc
func alloc(size int32) uintptr {
    ptr := uintptr(unsafe.Pointer(&heap[heapOffset]))
    heapOffset += uintptr(size)
    return ptr
}

//export handle
func handle(fnPtr, fnLen, dataPtr, dataLen int32) int64 {
    fn := string(unsafe.Slice((*byte)(unsafe.Pointer(uintptr(fnPtr))), fnLen))
    dataBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(dataPtr))), dataLen)

    var req map[string]any
    json.Unmarshal(dataBytes, &req)

    var resp map[string]any
    switch fn {
    case "add":
        v, _ := req["value"].(float64)
        resp = map[string]any{"value": v + 1}
    case "sub":
        v, _ := req["value"].(float64)
        resp = map[string]any{"value": v - 1}
    default:
        resp = map[string]any{"error": "unknown function: " + fn}
    }

    b, _ := json.Marshal(resp)
    copy(buf[:], b)
    ptr := uintptr(unsafe.Pointer(&buf[0]))
    return int64(ptr)<<32 | int64(len(b))
}

func main() {} // required for TinyGo
```

Compile:

```bash
tinygo build -o app.wasm -target wasi ./main.go
```

## Rust example (sketch)

```rust
#[no_mangle]
pub extern "C" fn alloc(size: i32) -> *mut u8 {
    let mut v = Vec::with_capacity(size as usize);
    let ptr = v.as_mut_ptr();
    std::mem::forget(v);
    ptr
}

#[no_mangle]
pub extern "C" fn handle(fn_ptr: i32, fn_len: i32, data_ptr: i32, data_len: i32) -> i64 {
    // read fn name + data from linear memory, return packed response pointer
    todo!()
}
```

## Runtime details

- The WASM runtime is [wazero](https://wazero.io/) — pure Go, no CGo, no external runtimes required.
- WASI `preview1` is instantiated automatically so the module can write to stdout/stderr.
- `handle()` is called synchronously from the Go goroutine serving `__CALL_BACKEND`.
- Server-push events (`BACKEND.onEvent`) are **not** supported in WASM mode — use `BACKEND.call` polling or `setInterval` instead.
- The response event fires immediately (no round-trip latency).

## Frontend JS

The same `BACKEND` object works in WASM mode:

```js
BACKEND.call("add", { value: 3 }, ({ data }) => {
  console.log(data.value) // 4
})
```

No other frontend changes are needed when switching from socket to WASM mode.
