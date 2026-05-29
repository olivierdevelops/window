package orchusecases

import (
	"fmt"
	"log"
	"webview_gui/domain"
	"webview_gui/features"
	"webview_gui/infra"
	"webview_gui/infra/wasm"
)

// MakeRunWASMMode returns a function that runs a WASM binary as the backend.
// The WASM module receives __CALL_BACKEND calls instead of a subprocess socket.
func MakeRunWASMMode(win features.Windowing, srv features.StaticServer) func(*domain.AppConfig) {
	return func(cfg *domain.AppConfig) {
		rt, err := wasm.Load(cfg.WASMBackend)
		if err != nil {
			log.Fatal("load wasm:", err)
		}
		defer rt.Close()

		port, err := srv.ResolvePort()
		if err != nil {
			log.Fatal(err)
		}
		addr := fmt.Sprintf(":%d", port)

		mux := srv.BuildMux(features.StaticMuxConfig{
			HTML:      cfg.HTML,
			EntryPath: cfg.EntryPath,
			Dirs:      cfg.Dirs,
			Files:     cfg.Files,
		})

		var h features.WindowHandle
		srv.Listen(addr, infra.LogMiddleware(mux), func() {
			if h != nil {
				win.Destroy(h)
			}
		})

		h, err = win.New(cfg.Title, cfg.Size, cfg.DebugMode)
		if err != nil {
			log.Fatal(err)
		}
		defer win.Destroy(h)

		if err := win.Bind(h, "logFromJS", func(level, msg string) {
			log.Println("[js]\t", level+": "+msg)
		}); err != nil {
			log.Fatal(err)
		}

		if err := win.Bind(h, "__CALL_BACKEND", func(functionName string, data map[string]any) map[string]any {
			result, err := rt.Handle(functionName, data)
			if err != nil {
				return map[string]any{"err": err.Error()}
			}
			return map[string]any{"result": result}
		}); err != nil {
			log.Fatal(err)
		}

		win.Init(h, string(wasmBackendJS))
		win.Navigate(h, fmt.Sprintf("http://127.0.0.1:%d", port))
		win.Run(h)
	}
}

// wasmBackendJS is a BACKEND shim for WASM mode.
// __CALL_BACKEND returns {result} directly (synchronous WASM, no event roundtrip).
var wasmBackendJS = []byte(`(function(){
const levels=["log","warn","error","info"];
levels.forEach(l=>{const o=console[l];console[l]=function(...a){logFromJS(l,a.map(JSON.stringify).join(" "));o.apply(console,a);}});
window.BACKEND=new (class{call(n,p,cb){__CALL_BACKEND(n,p||{}).then(({result,err})=>{if(err){cb&&cb(null,err);return;}cb&&cb({data:result});});}onEvent(id,cb){}})();
})()`)
