package orchusecases

import (
	"fmt"
	"log"
	"os"
	"time"
	"webview_gui/domain"
	"webview_gui/features"
	"webview_gui/infra"
)

// MakeRunServerMode returns a function that runs the app in server mode:
// starts a local HTTP server and opens a webview pointing to it.
func MakeRunServerMode(win features.Windowing, srv features.StaticServer, initWindow func(features.WindowHandle, *domain.AppConfig) error) func(*domain.AppConfig) {
	return func(cfg *domain.AppConfig) {
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

		if os.Getenv("DEV_MODE") == "1" {
			for {
				time.Sleep(200 * time.Millisecond)
			}
		}

		h, err = win.New(cfg.Title, cfg.Size, cfg.DebugMode)
		if err != nil {
			log.Fatal(err)
		}
		defer win.Destroy(h)

		if err := initWindow(h, cfg); err != nil {
			win.Destroy(h)
			log.Fatal(err)
		}

		win.Navigate(h, fmt.Sprintf("http://127.0.0.1:%d", port))
		win.Run(h)
	}
}
