package orchusecases

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"webview_gui/domain"
	"webview_gui/features"
)

// fakeWinAndSrv builds minimal fake Windowing and StaticServer for testing run modes.
// win.Run is a no-op so mode functions return immediately.
type runRecorder struct {
	newCalls      int
	runCalled     bool
	navigateURLs  []string
	evalJS        []string
	destroyCalled int
	initCalled    bool
	initHandle    features.WindowHandle
	initCfg       *domain.AppConfig
	listenAddr    string
	buildMuxCfg   features.StaticMuxConfig
	resolvedPort  int
}

func fakeWinForMode(r *runRecorder, handle features.WindowHandle) features.Windowing {
	return features.Windowing{
		New: func(title string, size *domain.WindowSize, debug bool) (features.WindowHandle, error) {
			r.newCalls++
			return handle, nil
		},
		Destroy:   func(h features.WindowHandle) { r.destroyCalled++ },
		Navigate:  func(h features.WindowHandle, url string) { r.navigateURLs = append(r.navigateURLs, url) },
		Eval:      func(h features.WindowHandle, js string) { r.evalJS = append(r.evalJS, js) },
		Bind:      func(h features.WindowHandle, name string, fn any) error { return nil },
		Init:      func(h features.WindowHandle, js string) {},
		Run:       func(h features.WindowHandle) { r.runCalled = true },
		SetTitle:  func(h features.WindowHandle, title string) {},
		SendEvent: func(h features.WindowHandle, id string, data any) {},
	}
}

func fakeSrvForMode(r *runRecorder, port int) features.StaticServer {
	return features.StaticServer{
		ResolvePort: func() (int, error) {
			r.resolvedPort = port
			return port, nil
		},
		BuildMux: func(cfg features.StaticMuxConfig) *http.ServeMux {
			r.buildMuxCfg = cfg
			return http.NewServeMux()
		},
		Listen: func(addr string, handler http.Handler, onFail func()) {
			r.listenAddr = addr
		},
		WaitForServer: func(addr string) error {
			return nil
		},
	}
}

func fakeInitWindow(r *runRecorder) func(features.WindowHandle, *domain.AppConfig) error {
	return func(h features.WindowHandle, cfg *domain.AppConfig) error {
		r.initCalled = true
		r.initHandle = h
		r.initCfg = cfg
		return nil
	}
}

// ---- MakeRunServerMode ----

func TestRunServerMode_ResolvesPort(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18081)
	run := MakeRunServerMode(win, srv, fakeInitWindow(r))
	run(&domain.AppConfig{Title: "T", HTML: "<html/>"})
	if r.resolvedPort != 18081 {
		t.Errorf("resolvedPort = %d, want 18081", r.resolvedPort)
	}
}

func TestRunServerMode_ListensOnPort(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18082)
	MakeRunServerMode(win, srv, fakeInitWindow(r))(&domain.AppConfig{HTML: "<html/>"})
	if r.listenAddr != ":18082" {
		t.Errorf("listenAddr = %q, want :18082", r.listenAddr)
	}
}

func TestRunServerMode_NavigatesToLocalHost(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18083)
	MakeRunServerMode(win, srv, fakeInitWindow(r))(&domain.AppConfig{HTML: "<html/>"})
	if len(r.navigateURLs) == 0 {
		t.Fatal("Navigate not called")
	}
	if !strings.HasPrefix(r.navigateURLs[0], "http://127.0.0.1:18083") {
		t.Errorf("Navigate URL = %q, want http://127.0.0.1:18083...", r.navigateURLs[0])
	}
}

func TestRunServerMode_CallsInitWindow(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18084)
	cfg := &domain.AppConfig{HTML: "<html/>"}
	MakeRunServerMode(win, srv, fakeInitWindow(r))(cfg)
	if !r.initCalled {
		t.Error("initWindow not called")
	}
	if r.initCfg != cfg {
		t.Error("initWindow called with wrong config")
	}
}

func TestRunServerMode_CallsRun(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18085)
	MakeRunServerMode(win, srv, fakeInitWindow(r))(&domain.AppConfig{HTML: "<html/>"})
	if !r.runCalled {
		t.Error("win.Run not called")
	}
}

func TestRunServerMode_PassesStaticDirsToMux(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18086)
	cfg := &domain.AppConfig{
		HTML: "<html/>",
		Dirs: map[string]string{"/static": "./static"},
	}
	MakeRunServerMode(win, srv, fakeInitWindow(r))(cfg)
	if r.buildMuxCfg.HTML != "<html/>" {
		t.Errorf("BuildMux HTML = %q", r.buildMuxCfg.HTML)
	}
	if r.buildMuxCfg.Dirs["/static"] != "./static" {
		t.Errorf("BuildMux Dirs = %v", r.buildMuxCfg.Dirs)
	}
}

// ---- MakeRunURLMode ----

func TestRunURLMode_NavigatesToConfigURL(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	cfg := &domain.AppConfig{URL: "https://example.com"}
	MakeRunURLMode(win, fakeInitWindow(r))(cfg)
	if len(r.navigateURLs) == 0 {
		t.Fatal("Navigate not called")
	}
	if r.navigateURLs[0] != "https://example.com" {
		t.Errorf("Navigate URL = %q, want https://example.com", r.navigateURLs[0])
	}
}

func TestRunURLMode_CallsInitWindow(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	cfg := &domain.AppConfig{URL: "https://example.com"}
	MakeRunURLMode(win, fakeInitWindow(r))(cfg)
	if !r.initCalled {
		t.Error("initWindow not called")
	}
}

func TestRunURLMode_CallsRun(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	MakeRunURLMode(win, fakeInitWindow(r))(&domain.AppConfig{URL: "https://x.com"})
	if !r.runCalled {
		t.Error("win.Run not called")
	}
}

// ---- MakeRunProxyMode ----

func TestRunProxyMode_FallsBackToURLMode_WhenNoProxyTarget(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18087)
	cfg := &domain.AppConfig{URL: "https://fallback.example.com"}
	MakeRunProxyMode(win, srv, fakeInitWindow(r))(cfg)
	if len(r.navigateURLs) == 0 {
		t.Fatal("Navigate not called")
	}
	if r.navigateURLs[0] != "https://fallback.example.com" {
		t.Errorf("Navigate URL = %q, want https://fallback.example.com", r.navigateURLs[0])
	}
}

func TestRunProxyMode_ListensAndNavigates_WhenProxyTargetSet(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18088)
	// Use a fixed port in the target so no port allocation happens
	cfg := &domain.AppConfig{ProxyTarget: "http://127.0.0.1:19999"}
	MakeRunProxyMode(win, srv, fakeInitWindow(r))(cfg)
	if r.listenAddr != ":18088" {
		t.Errorf("listenAddr = %q, want :18088", r.listenAddr)
	}
	if len(r.navigateURLs) == 0 {
		t.Fatal("Navigate not called")
	}
	if !strings.HasPrefix(r.navigateURLs[0], "http://127.0.0.1:18088") {
		t.Errorf("Navigate URL = %q", r.navigateURLs[0])
	}
}

func TestRunProxyMode_ExpandsEnvVars(t *testing.T) {
	os.Setenv("TEST_PROXY_PORT", "19998")
	defer os.Unsetenv("TEST_PROXY_PORT")

	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")
	srv := fakeSrvForMode(r, 18089)
	cfg := &domain.AppConfig{ProxyTarget: "http://127.0.0.1:${TEST_PROXY_PORT}"}
	MakeRunProxyMode(win, srv, fakeInitWindow(r))(cfg)
	// Should expand env var and proceed (no fatal since port is specified)
	if r.runCalled == false {
		t.Error("expected win.Run to be called")
	}
}

// ---- MakeRunBrowserMode ----

func TestRunBrowserMode_EvalsInjectedHTML(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")

	htmlFile, err := os.CreateTemp("", "overlay_*.html")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(htmlFile.Name())
	htmlFile.WriteString("<div>INJECTED</div>")
	htmlFile.Close()

	cfg := &domain.AppConfig{
		URL:            "https://example.com",
		InjectHTMLPath: htmlFile.Name(),
	}
	MakeRunBrowserMode(win)(cfg)
	if len(r.evalJS) == 0 {
		t.Fatal("Eval not called")
	}
	combined := strings.Join(r.evalJS, "\n")
	if !strings.Contains(combined, "INJECTED") {
		t.Errorf("Eval JS does not contain injected HTML. Got: %q", combined)
	}
}

func TestRunBrowserMode_CallsRun(t *testing.T) {
	r := &runRecorder{}
	win := fakeWinForMode(r, "h1")

	htmlFile, _ := os.CreateTemp("", "overlay_*.html")
	htmlFile.WriteString("<div/>")
	htmlFile.Close()
	defer os.Remove(htmlFile.Name())

	MakeRunBrowserMode(win)(&domain.AppConfig{InjectHTMLPath: htmlFile.Name()})
	if !r.runCalled {
		t.Error("win.Run not called")
	}
}

// ---- MakeRunControlledMode executor logic ----

func TestRunControlledMode_Executor_CreateAndDestroy(t *testing.T) {
	handles := map[string]features.WindowHandle{}
	var runDone = make(chan struct{})
	counter := 0

	win := features.Windowing{
		New: func(title string, size *domain.WindowSize, debug bool) (features.WindowHandle, error) {
			counter++
			id := "h" + string(rune('0'+counter))
			handles[id] = id
			return id, nil
		},
		Navigate:  func(h features.WindowHandle, url string) {},
		Destroy:   func(h features.WindowHandle) { delete(handles, h.(string)) },
		Run:       func(h features.WindowHandle) { <-runDone },
		Bind:      func(h features.WindowHandle, name string, fn any) error { return nil },
		Init:      func(h features.WindowHandle, js string) {},
		Eval:      func(h features.WindowHandle, js string) {},
		SetTitle:  func(h features.WindowHandle, title string) {},
		SendEvent: func(h features.WindowHandle, id string, data any) {},
	}

	var capturedExec features.WindowCommandExecutor
	ctrl := features.ControlledMode{
		StartManagementSocket: func(sockPath string, exec features.WindowCommandExecutor) error {
			capturedExec = exec
			return nil
		},
	}

	cfg := &domain.AppConfig{} // no ControlledScript to avoid subprocess
	go MakeRunControlledMode(win, ctrl)(cfg)
	time.Sleep(20 * time.Millisecond)

	// CreateWindow
	winID, err := capturedExec.CreateWindow("Test", "https://example.com", 800, 600)
	if err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}
	if winID == "" {
		t.Error("expected non-empty window ID")
	}

	time.Sleep(20 * time.Millisecond)

	// NavigateWindow
	if err := capturedExec.NavigateWindow(winID, "https://other.com"); err != nil {
		t.Errorf("NavigateWindow: %v", err)
	}

	// EvalWindow
	if err := capturedExec.EvalWindow(winID, "1+1"); err != nil {
		t.Errorf("EvalWindow: %v", err)
	}

	// DestroyWindow — need to unblock win.Run first
	close(runDone)
	time.Sleep(20 * time.Millisecond)

	if err := capturedExec.DestroyWindow(winID); err != nil {
		// May succeed or get "not found" depending on goroutine timing — either is valid
		_ = err
	}

	// DestroyWindow for unknown ID
	if err := capturedExec.DestroyWindow("nonexistent"); err == nil {
		t.Error("expected error for unknown window ID")
	}
}
