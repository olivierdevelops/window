package orchestrator

import (
	"webview_gui/domain"
	orchfeatures "webview_gui/orchestrator/features"
	orchusecases "webview_gui/orchestrator/usecases"
)

// Run wires all features together and dispatches to the configured run mode.
func Run(cfg *domain.AppConfig) {
	win := orchfeatures.MakeWindowing()
	srv := orchfeatures.MakeStaticServer()
	bridge := orchfeatures.MakeBackendBridge()
	ctrl := orchfeatures.MakeControlledMode()

	initWindow := orchusecases.MakeInitWindow(win, bridge)

	switch cfg.Mode {
	case domain.ModeBrowser:
		orchusecases.MakeRunBrowserMode(win)(cfg)
	case domain.ModeProxy:
		orchusecases.MakeRunProxyMode(win, srv, initWindow)(cfg)
	case domain.ModeURL:
		orchusecases.MakeRunURLMode(win, initWindow)(cfg)
	case domain.ModeControlled:
		orchusecases.MakeRunControlledMode(win, ctrl)(cfg)
	case domain.ModeWASM:
		orchusecases.MakeRunWASMMode(win, srv)(cfg)
	default:
		orchusecases.MakeRunServerMode(win, srv, initWindow)(cfg)
	}
}
