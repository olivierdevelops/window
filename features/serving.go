package features

import "net/http"

// StaticServer builds and serves static HTTP content.
type StaticServer struct {
	BuildMux      func(cfg StaticMuxConfig) *http.ServeMux
	Listen        func(addr string, handler http.Handler, onFail func())
	ResolvePort   func() (int, error)
	WaitForServer func(addr string) error
}

type StaticMuxConfig struct {
	HTML      string
	EntryPath string
	Dirs      map[string]string
	Files     map[string]string
}
