package infra

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetFreePort(t *testing.T) {
	port, err := GetFreePort()
	if err != nil {
		t.Fatal(err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("port %d out of range", port)
	}
}

func TestResolvePort_Default(t *testing.T) {
	os.Unsetenv("WINDOW_PORT")
	port, err := ResolvePort()
	if err != nil {
		t.Fatal(err)
	}
	if port <= 0 {
		t.Errorf("expected positive port, got %d", port)
	}
}

func TestResolvePort_EnvVar(t *testing.T) {
	os.Setenv("WINDOW_PORT", "19876")
	defer os.Unsetenv("WINDOW_PORT")
	port, err := ResolvePort()
	if err != nil {
		t.Fatal(err)
	}
	if port != 19876 {
		t.Errorf("expected 19876, got %d", port)
	}
}

func TestWaitForServer_Timeout(t *testing.T) {
	// Use a port that's almost certainly not listening
	err := WaitForServer("127.0.0.1:19999", 300*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
	te, ok := err.(*ServerTimeoutError)
	if !ok {
		t.Fatalf("expected *ServerTimeoutError, got %T: %v", err, err)
	}
	if te.Addr != "127.0.0.1:19999" {
		t.Errorf("unexpected addr %q", te.Addr)
	}
}

func TestWaitForServer_Ready(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	// Strip scheme from srv.URL
	addr := srv.Listener.Addr().String()
	err := WaitForServer(addr, 2*time.Second)
	if err != nil {
		t.Errorf("expected server to be ready: %v", err)
	}
}

func TestBuildStaticMux_HTMLRoot(t *testing.T) {
	mux := BuildStaticMux("<html>hi</html>", "", nil, nil)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "<html>hi</html>" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestBuildStaticMux_FileRoute(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("content"), 0644)

	mux := BuildStaticMux("", "", nil, map[string]string{"/test.txt": path})
	req := httptest.NewRequest("GET", "/test.txt", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "content" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestServerTimeoutError_Message(t *testing.T) {
	e := &ServerTimeoutError{Addr: "127.0.0.1:1234", Timeout: 5 * time.Second}
	msg := e.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}
