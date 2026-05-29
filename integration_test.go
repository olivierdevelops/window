package main_test

// Integration tests that build and run the `window` binary.
// They use DEV_MODE=1 to start the HTTP server without opening a native window,
// then make HTTP requests to verify content is served as advertised.
//
// These tests require a working Go toolchain and the demos/ directory.

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// buildBinary compiles the window binary once per test run.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "window")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build failed: %v", err)
	}
	return bin
}

// freePort returns an available TCP port.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

// startDemoServer starts `window <yamlPath>` with DEV_MODE=1 and WINDOW_PORT=<port>.
// Returns a cleanup function that kills the process.
func startDemoServer(t *testing.T, bin, yamlPath string, port int) {
	t.Helper()
	cmd := exec.Command(bin, yamlPath)
	cmd.Env = append(os.Environ(), "DEV_MODE=1", fmt.Sprintf("WINDOW_PORT=%d", port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", yamlPath, err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})

	// Wait up to 5s for the server to accept connections
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready", addr)
}

// httpGet fetches a URL and returns the response body as a string.
func httpGet(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ---- CLI integration tests ----

func TestCLI_Version(t *testing.T) {
	bin := buildBinary(t)
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatalf("--version: %v", err)
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		t.Error("expected non-empty version string")
	}
	if !strings.Contains(v, ".") {
		t.Errorf("version %q does not look like a semver string", v)
	}
}

func TestCLI_Init_CreatesScaffold(t *testing.T) {
	bin := buildBinary(t)
	dir := filepath.Join(t.TempDir(), "myapp")

	cmd := exec.Command(bin, "--init", dir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("--init: %v", err)
	}

	// Verify window.yaml exists
	if _, err := os.Stat(filepath.Join(dir, "window.yaml")); err != nil {
		t.Errorf("window.yaml missing: %v", err)
	}
	// Verify static/index.html exists
	if _, err := os.Stat(filepath.Join(dir, "static", "index.html")); err != nil {
		t.Errorf("static/index.html missing: %v", err)
	}
	// window.yaml should be non-empty
	b, _ := os.ReadFile(filepath.Join(dir, "window.yaml"))
	if len(b) == 0 {
		t.Error("window.yaml is empty")
	}
}

// ---- Demo HTTP serving integration tests ----

func TestDemo_Hello_ServesHTML(t *testing.T) {
	bin := buildBinary(t)
	port := freePort(t)
	yamlPath, err := filepath.Abs("demos/hello/window.yaml")
	if err != nil {
		t.Fatal(err)
	}
	startDemoServer(t, bin, yamlPath, port)

	body := httpGet(t, fmt.Sprintf("http://127.0.0.1:%d/", port))
	if !strings.Contains(body, "Hello") {
		t.Errorf("expected 'Hello' in body, got:\n%s", body[:min(len(body), 500)])
	}
}

func TestDemo_Hello_StaticDir_NotFoundFor404(t *testing.T) {
	bin := buildBinary(t)
	port := freePort(t)
	yamlPath, _ := filepath.Abs("demos/hello/window.yaml")
	startDemoServer(t, bin, yamlPath, port)

	// Nonexistent path under /static/ should return 404 (not panic or 500)
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/static/does_not_exist.png", port))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for missing static file, got %d", resp.StatusCode)
	}
}

func TestDemo_Dashboard_ServesHTML(t *testing.T) {
	bin := buildBinary(t)
	port := freePort(t)
	yamlPath, _ := filepath.Abs("demos/dashboard/window.yaml")
	startDemoServer(t, bin, yamlPath, port)

	body := httpGet(t, fmt.Sprintf("http://127.0.0.1:%d/", port))
	if len(body) == 0 {
		t.Error("expected non-empty response from dashboard")
	}
}

func TestDemo_FileExplorer_ServesHTML(t *testing.T) {
	bin := buildBinary(t)
	port := freePort(t)
	yamlPath, _ := filepath.Abs("demos/file_explorer/window.yaml")
	startDemoServer(t, bin, yamlPath, port)

	body := httpGet(t, fmt.Sprintf("http://127.0.0.1:%d/", port))
	if len(body) == 0 {
		t.Error("expected non-empty response from file_explorer")
	}
}

func TestDemo_Terminal_ServesHTML(t *testing.T) {
	bin := buildBinary(t)
	port := freePort(t)
	yamlPath, _ := filepath.Abs("demos/terminal/window.yaml")
	startDemoServer(t, bin, yamlPath, port)

	body := httpGet(t, fmt.Sprintf("http://127.0.0.1:%d/", port))
	if len(body) == 0 {
		t.Error("expected non-empty response from terminal")
	}
}

func TestDemo_Counter_ServesHTML(t *testing.T) {
	bin := buildBinary(t)
	port := freePort(t)
	yamlPath, _ := filepath.Abs("demos/counter/window.yaml")
	startDemoServer(t, bin, yamlPath, port)

	body := httpGet(t, fmt.Sprintf("http://127.0.0.1:%d/", port))
	if len(body) == 0 {
		t.Error("expected non-empty response from counter")
	}
}

// ---- orchestrator.Run dispatch ----
// Run() itself creates real webview handles (CGo) and cannot run in CI without a display.
// The dispatch logic is covered by mode function tests in orchestrator/usecases.
// This test verifies the binary starts up and the HTTP stack works end-to-end with a
// minimal inline config (DEV_MODE=1 prevents window creation).

func TestBinary_ServerMode_HTTP(t *testing.T) {
	bin := buildBinary(t)
	port := freePort(t)

	dir := t.TempDir()
	html := `<!DOCTYPE html><html><body>integration</body></html>`
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(html), 0644)
	yaml := fmt.Sprintf("title: Test\nentry_path: ./index.html\n")
	yamlPath := filepath.Join(dir, "window.yaml")
	os.WriteFile(yamlPath, []byte(yaml), 0644)

	startDemoServer(t, bin, yamlPath, port)

	body := httpGet(t, fmt.Sprintf("http://127.0.0.1:%d/", port))
	if !strings.Contains(body, "integration") {
		t.Errorf("expected 'integration' in body, got:\n%s", body[:min(len(body), 500)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
