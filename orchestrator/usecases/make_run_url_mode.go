package orchusecases

import (
	"log"
	"webview_gui/domain"
	"webview_gui/features"
)

// MakeRunURLMode returns a function that opens a webview navigating to cfg.URL.
func MakeRunURLMode(win features.Windowing, initWindow func(features.WindowHandle, *domain.AppConfig) error) func(*domain.AppConfig) {
	return func(cfg *domain.AppConfig) {
		h, err := win.New(cfg.Title, cfg.Size, cfg.DebugMode)
		if err != nil {
			log.Fatal(err)
		}
		defer win.Destroy(h)

		if err := initWindow(h, cfg); err != nil {
			win.Destroy(h)
			log.Fatal(err)
		}

		win.Navigate(h, cfg.URL)
		win.Run(h)
	}
}
