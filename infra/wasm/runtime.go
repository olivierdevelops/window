package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Runtime wraps a wazero WASM runtime and a loaded module.
type Runtime struct {
	ctx    context.Context
	rt     wazero.Runtime
	mod    api.Module
	alloc  api.Function
	handle api.Function
}

// Load reads a .wasm file and instantiates it. The module must export:
//
//	alloc(size i32) i32        — allocate bytes, return pointer
//	handle(fn_ptr i32, fn_len i32, data_ptr i32, data_len i32) i64
//	                           — packed resp_ptr<<32 | resp_len; receives/returns JSON
//
// An optional "init" export is called once at startup.
func Load(wasmPath string) (*Runtime, error) {
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("read wasm: %w", err)
	}

	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	mod, err := rt.InstantiateWithConfig(ctx, wasmBytes,
		wazero.NewModuleConfig().WithStdout(os.Stdout).WithStderr(os.Stderr))
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("instantiate wasm: %w", err)
	}

	allocFn := mod.ExportedFunction("alloc")
	handleFn := mod.ExportedFunction("handle")
	if allocFn == nil || handleFn == nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("wasm module must export 'alloc' and 'handle'")
	}

	r := &Runtime{ctx: ctx, rt: rt, mod: mod, alloc: allocFn, handle: handleFn}

	if initFn := mod.ExportedFunction("init"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			r.Close()
			return nil, fmt.Errorf("wasm init: %w", err)
		}
	}

	return r, nil
}

// Handle calls the WASM module's handle function with the given function name
// and data (as JSON). Returns the response JSON bytes.
func (r *Runtime) Handle(funcName string, data map[string]any) (map[string]any, error) {
	fnBytes := []byte(funcName)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}

	fnPtr, err := r.writeBytes(fnBytes)
	if err != nil {
		return nil, err
	}
	dataPtr, err := r.writeBytes(dataBytes)
	if err != nil {
		return nil, err
	}

	results, err := r.handle.Call(r.ctx,
		uint64(fnPtr), uint64(len(fnBytes)),
		uint64(dataPtr), uint64(len(dataBytes)))
	if err != nil {
		return nil, fmt.Errorf("wasm handle: %w", err)
	}

	packed := results[0]
	respPtr := uint32(packed >> 32)
	respLen := uint32(packed & 0xffffffff)

	respBytes, ok := r.mod.Memory().Read(respPtr, respLen)
	if !ok {
		return nil, fmt.Errorf("wasm: cannot read response memory")
	}

	var out map[string]any
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("unmarshal wasm response: %w", err)
	}
	return out, nil
}

func (r *Runtime) writeBytes(data []byte) (uint32, error) {
	results, err := r.alloc.Call(r.ctx, uint64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("wasm alloc: %w", err)
	}
	ptr := uint32(results[0])
	if !r.mod.Memory().Write(ptr, data) {
		return 0, fmt.Errorf("wasm: cannot write to memory at %d", ptr)
	}
	return ptr, nil
}

// Close shuts down the WASM runtime.
func (r *Runtime) Close() {
	r.rt.Close(r.ctx)
}
