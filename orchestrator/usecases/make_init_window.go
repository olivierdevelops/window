package orchusecases

import (
	"fmt"
	"log"
	"webview_gui/assets"
	"webview_gui/domain"
	"webview_gui/features"
	"webview_gui/infra"
	orchfeatures "webview_gui/orchestrator/features"
)

// MakeInitWindow returns a function that binds logFromJS, injects backend.js,
// binds requested native features, and starts the backend subprocess if configured.
func MakeInitWindow(win features.Windowing, bridge features.BackendBridge) func(features.WindowHandle, *domain.AppConfig) error {
	return func(h features.WindowHandle, cfg *domain.AppConfig) error {
		if err := win.Bind(h, "logFromJS", func(level, msg string) {
			log.Println("[js]\t", level+": "+msg)
		}); err != nil {
			return err
		}

		win.Init(h, string(assets.BackendJS))

		if err := bindNativeFeatures(win, h, cfg.NativeFeatures); err != nil {
			return err
		}

		for _, lib := range cfg.JSInject {
			win.Init(h, fmt.Sprintf(
				`(function(){var s=document.createElement('script');s.src=%q;document.head.appendChild(s)})()`,
				lib))
		}

		if cfg.RunBackendScript == "" {
			// No subprocess backend: serve BACKEND.call() from in-process Go
			// handlers — minimal management, one binary, no socket.
			reg := infra.BuiltinHandlers()
			return win.Bind(h, "__CALL_BACKEND", func(functionName string, data map[string]any) map[string]any {
				result, err := reg.Dispatch(functionName, data)
				if err != nil {
					return map[string]any{"err": err.Error()}
				}
				return map[string]any{"result": result}
			})
		}

		bridge.OnServerPush(func(msg *domain.Message) {
			win.SendEvent(h, msg.Function, msg.Data)
		})

		if err := bridge.Start(cfg.RunBackendScript); err != nil {
			return err
		}

		return win.Bind(h, "__CALL_BACKEND", func(functionName string, data map[string]any) map[string]any {
			var eventID string
			id, err := bridge.HandleRequest(functionName, data, func(msg *domain.Message) {
				win.SendEvent(h, eventID, msg.Data)
			})
			if err != nil {
				return map[string]any{"err": err.Error()}
			}
			eventID = fmt.Sprintf("backend:result:%s", id)
			return map[string]any{"eventId": eventID}
		})
	}
}

func bindNativeFeatures(win features.Windowing, h features.WindowHandle, nativeFeatures []domain.NativeFeature) error {
	for _, nf := range nativeFeatures {
		var err error
		switch nf {
		case domain.NativeFS:
			err = bindNativeFS(win, h, orchfeatures.MakeNativeFS())
		case domain.NativeOS:
			err = bindNativeOS(win, h, orchfeatures.MakeNativeOS())
		case domain.NativeDialogs:
			err = bindNativeDialogs(win, h, orchfeatures.MakeNativeDialogs())
		case domain.NativeCanvas:
			err = bindNativeCanvas(win, h, orchfeatures.MakeNativeCanvas(win, h))
		case domain.NativeCamera:
			win.Init(h, string(assets.NativeCameraJS))
		case domain.NativeMic:
			win.Init(h, string(assets.NativeMicJS))
		case domain.NativeSpeech:
			win.Init(h, string(assets.NativeSpeechJS))
		case domain.NativeScreen:
			win.Init(h, string(assets.NativeScreenJS))
		case domain.NativeInput:
			win.Init(h, string(assets.NativeInputJS))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func bindNativeFS(win features.Windowing, h features.WindowHandle, nfs features.NativeFS) error {
	win.Init(h, string(assets.NativeFSJS))

	if err := win.Bind(h, "__native_fs_readFile", func(path string) (string, error) {
		b, err := nfs.ReadFile(path)
		return string(b), err
	}); err != nil {
		return err
	}
	if err := win.Bind(h, "__native_fs_writeFile", func(path, data string, perm uint32) error {
		return nfs.WriteFile(path, []byte(data), perm)
	}); err != nil {
		return err
	}
	if err := win.Bind(h, "__native_fs_readDir", func(path string) ([]features.DirEntry, error) {
		return nfs.ReadDir(path)
	}); err != nil {
		return err
	}
	return win.Bind(h, "__native_fs_watchFile", func(path string) error {
		_, err := nfs.WatchFile(path, func(content []byte) {
			win.SendEvent(h, "native:fs:watch:"+path, map[string]any{"content": string(content)})
		})
		return err
	})
}

func bindNativeOS(win features.Windowing, h features.WindowHandle, nos features.NativeOS) error {
	win.Init(h, string(assets.NativeOSJS))

	if err := win.Bind(h, "__native_os_exec", func(cmd string, args []string) map[string]any {
		stdout, stderr, err := nos.Exec(cmd, args)
		result := map[string]any{"stdout": stdout, "stderr": stderr}
		if err != nil {
			result["error"] = err.Error()
		}
		return result
	}); err != nil {
		return err
	}
	if err := win.Bind(h, "__native_os_getEnv", func(key string) string {
		return nos.GetEnv(key)
	}); err != nil {
		return err
	}
	if err := win.Bind(h, "__native_os_platform", func() string {
		return nos.Platform()
	}); err != nil {
		return err
	}
	return win.Bind(h, "__native_os_info", func() features.OSInfo {
		return nos.OSInfo()
	})
}

func bindNativeDialogs(win features.Windowing, h features.WindowHandle, nd features.NativeDialogs) error {
	win.Init(h, string(assets.NativeDialogsJS))

	if err := win.Bind(h, "__native_dialogs_openFile", func(opts features.FileDialogOptions) ([]string, error) {
		return nd.OpenFile(opts)
	}); err != nil {
		return err
	}
	if err := win.Bind(h, "__native_dialogs_saveFile", func(opts features.FileDialogOptions) (string, error) {
		return nd.SaveFile(opts)
	}); err != nil {
		return err
	}
	return win.Bind(h, "__native_dialogs_showMessage", func(opts features.MessageDialogOptions) error {
		return nd.ShowMessage(opts)
	})
}

func bindNativeCanvas(win features.Windowing, h features.WindowHandle, nc features.NativeCanvas) error {
	win.Init(h, string(assets.NativeCanvasJS))

	if err := win.Bind(h, "__native_canvas_drawRect", func(opts features.CanvasRect) error {
		return nc.DrawRect(opts)
	}); err != nil {
		return err
	}
	if err := win.Bind(h, "__native_canvas_drawText", func(opts features.CanvasText) error {
		return nc.DrawText(opts)
	}); err != nil {
		return err
	}
	if err := win.Bind(h, "__native_canvas_clear", func(canvasID string) error {
		return nc.Clear(canvasID)
	}); err != nil {
		return err
	}
	return win.Bind(h, "__native_canvas_flush", func(canvasID string) error {
		return nc.Flush(canvasID)
	})
}
