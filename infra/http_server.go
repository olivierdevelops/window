package infra

import (
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetFreePort returns an available TCP port chosen by the OS.
func GetFreePort() (int, error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// ResolvePort returns the port from WINDOW_PORT env var, or a free port.
func ResolvePort() (int, error) {
	if s := os.Getenv("WINDOW_PORT"); s != "" {
		return strconv.Atoi(s)
	}
	return GetFreePort()
}

// ListenAndServe starts an HTTP server in a goroutine; calls onFail on error.
func ListenAndServe(addr string, handler http.Handler, onFail func()) {
	go func() {
		if err := http.ListenAndServe(addr, handler); err != nil {
			if onFail != nil {
				onFail()
			}
			log.Fatal(err)
		}
	}()
}

// WaitForServer polls addr until it accepts connections or timeout elapses.
func WaitForServer(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return &ServerTimeoutError{Addr: addr, Timeout: timeout}
}

type ServerTimeoutError struct {
	Addr    string
	Timeout time.Duration
}

func (e *ServerTimeoutError) Error() string {
	return "server at " + e.Addr + " did not become ready within " + e.Timeout.String()
}

// BuildStaticMux constructs an HTTP mux for serving static files and directories.
func BuildStaticMux(html, entryPath string, dirs, files map[string]string) *http.ServeMux {
	mux := http.NewServeMux()

	for urlPath, path := range files {
		path := path
		mux.HandleFunc(urlPath, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, path)
		})
	}

	for urlPath, path := range dirs {
		info, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}
		if !info.IsDir() {
			log.Fatalf("%s must be a directory", path)
		}
		if !strings.HasSuffix(urlPath, "/") {
			urlPath += "/"
		}
		root := http.Dir(path)
		fs := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("looking for:", r.URL.Path)
			f, err := root.Open(r.URL.Path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			info, err := f.Stat()
			if err != nil || info.IsDir() {
				http.NotFound(w, r)
				return
			}
			http.FileServer(root).ServeHTTP(w, r)
		})
		mux.Handle(urlPath, http.StripPrefix(urlPath, fs))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if html != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, entryPath, time.Now(), strings.NewReader(html))
		} else if entryPath != "" {
			http.ServeFile(w, r, entryPath)
		}
	})

	return mux
}
