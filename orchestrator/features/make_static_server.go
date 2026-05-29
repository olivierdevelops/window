package orchfeatures

import (
	"net/http"
	"time"
	"webview_gui/features"
	"webview_gui/infra"
)

// MakeStaticServer builds a StaticServer backed by infra HTTP helpers.
func MakeStaticServer() features.StaticServer {
	return features.StaticServer{
		BuildMux: func(cfg features.StaticMuxConfig) *http.ServeMux {
			return infra.BuildStaticMux(cfg.HTML, cfg.EntryPath, cfg.Dirs, cfg.Files)
		},
		Listen:      infra.ListenAndServe,
		ResolvePort: infra.ResolvePort,
		WaitForServer: func(addr string) error {
			return infra.WaitForServer(addr, 10*time.Second)
		},
	}
}
