package orchusecases

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"webview_gui/domain"
	"webview_gui/features"
	"webview_gui/infra"
)

// MakeRunProxyMode returns a function that runs the app in proxy mode:
// forwards requests to a target URL (optionally starting a subprocess first).
func MakeRunProxyMode(win features.Windowing, srv features.StaticServer, initWindow func(features.WindowHandle, *domain.AppConfig) error) func(*domain.AppConfig) {
	return func(cfg *domain.AppConfig) {
		if cfg.ProxyTarget == "" {
			if cfg.URL != "" {
				MakeRunURLMode(win, initWindow)(cfg)
				return
			}
			log.Fatal("proxy mode requires proxy_target to be set")
		}

		expandedTarget := os.ExpandEnv(cfg.ProxyTarget)
		target, err := url.Parse(expandedTarget)
		if err != nil {
			log.Fatal("invalid proxy_target:", err)
		}

		if target.Port() == "0" || target.Port() == "" {
			port, err := infra.GetFreePort()
			if err != nil {
				log.Fatal("failed to get free port:", err)
			}
			target.Host = fmt.Sprintf("%s:%d", target.Hostname(), port)
			os.Setenv("PORT", fmt.Sprintf("%d", port))
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.Host = target.Host
			},
			FlushInterval: -1,
			Transport:     &http.Transport{DisableCompression: true},
		}

		mux := http.NewServeMux()
		mux.Handle("/", proxy)

		port, err := srv.ResolvePort()
		if err != nil {
			log.Fatal(err)
		}
		addr := fmt.Sprintf(":%d", port)
		log.Printf("Address: http://127.0.0.1:%d → proxying to %s", port, cfg.ProxyTarget)

		var h features.WindowHandle
		srv.Listen(addr, infra.LogMiddleware(mux), func() {
			if h != nil {
				win.Destroy(h)
			}
		})

		if cfg.ProxyCommand != "" {
			cmd := infra.GetRunScriptCMD(cfg.ProxyCommand, nil)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				log.Fatal("failed to start proxy command:", err)
			}
			if err := srv.WaitForServer(target.Host); err != nil {
				cmd.Process.Kill()
				log.Fatal(err)
			}
			log.Println("connected to server, opening window")
			go func() {
				if err := cmd.Wait(); err != nil {
					log.Println("proxy command exited:", err)
				}
				os.Exit(0)
			}()
			h, err = win.New(cfg.Title, cfg.Size, cfg.DebugMode)
			if err != nil {
				log.Fatal(err)
			}
			defer win.Destroy(h)
			win.Navigate(h, fmt.Sprintf("http://127.0.0.1:%d", port))
			win.Run(h)
			cmd.Process.Kill()
			return
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
