package orchusecases

import (
	"log"
	"os"
	"strings"
	"webview_gui/assets"
	"webview_gui/domain"
	"webview_gui/features"
)

// MakeRunBrowserMode returns a function that opens a webview with injected UI HTML.
func MakeRunBrowserMode(win features.Windowing) func(*domain.AppConfig) {
	return func(cfg *domain.AppConfig) {
		h, err := win.New(cfg.Title, cfg.Size, cfg.DebugMode)
		if err != nil {
			log.Fatal(err)
		}
		defer win.Destroy(h)

		uiHTML, err := os.ReadFile(cfg.InjectHTMLPath)
		if err != nil {
			log.Fatal(err)
		}
		script := strings.ReplaceAll(string(assets.BrowserJS), "<HTML>", string(uiHTML))
		win.Eval(h, script)
		win.Run(h)
	}
}
