package orchfeatures

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"
	"webview_gui/features"
)

func shortSockPath(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "ctl_*.sock")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path)
	t.Cleanup(func() { os.Remove(path) })
	return path
}

type controlledModeRecorder struct {
	createdTitle  string
	createdURL    string
	createdWidth  int
	createdHeight int
	navigatedWin  string
	navigatedURL  string
	evaledWin     string
	evaledJS      string
	closedWin     string
}

func makeTestExecutor(r *controlledModeRecorder, winID string) features.WindowCommandExecutor {
	return features.WindowCommandExecutor{
		CreateWindow: func(title, url string, width, height int) (string, error) {
			r.createdTitle = title
			r.createdURL = url
			r.createdWidth = width
			r.createdHeight = height
			return winID, nil
		},
		NavigateWindow: func(windowID, url string) error {
			r.navigatedWin = windowID
			r.navigatedURL = url
			return nil
		},
		EvalWindow: func(windowID, js string) error {
			r.evaledWin = windowID
			r.evaledJS = js
			return nil
		},
		DestroyWindow: func(windowID string) error {
			r.closedWin = windowID
			return nil
		},
	}
}

type controlReply struct {
	ID       string `json:"id"`
	WindowID string `json:"window_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

func sendAndRecv(t *testing.T, conn net.Conn, cmd any) controlReply {
	t.Helper()
	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal cmd: %v", err)
	}
	b = append(b, '\n')
	if _, err := conn.Write(b); err != nil {
		t.Fatalf("write cmd: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("expected reply line from server")
	}
	var reply controlReply
	if err := json.Unmarshal(scanner.Bytes(), &reply); err != nil {
		t.Fatalf("unmarshal reply: %v", err)
	}
	return reply
}

func setupControlledMode(t *testing.T, executor features.WindowCommandExecutor) net.Conn {
	t.Helper()
	sockPath := shortSockPath(t)

	cm := MakeControlledMode()
	if err := cm.StartManagementSocket(sockPath, executor); err != nil {
		t.Fatalf("StartManagementSocket: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial unix socket: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	time.Sleep(50 * time.Millisecond)
	return conn
}

func TestControlledMode_CreateWindow(t *testing.T) {
	r := &controlledModeRecorder{}
	exec := makeTestExecutor(r, "win1")
	conn := setupControlledMode(t, exec)

	cmd := map[string]any{
		"cmd": "create_window",
		"id":  "r1",
		"params": map[string]any{
			"title":  "T",
			"url":    "http://x.com",
			"width":  800,
			"height": 600,
		},
	}
	reply := sendAndRecv(t, conn, cmd)

	if reply.ID != "r1" {
		t.Errorf("reply.ID = %q, want %q", reply.ID, "r1")
	}
	if reply.WindowID != "win1" {
		t.Errorf("reply.WindowID = %q, want %q", reply.WindowID, "win1")
	}
	if reply.Error != "" {
		t.Errorf("unexpected error: %s", reply.Error)
	}
	if r.createdTitle != "T" {
		t.Errorf("CreateWindow title = %q, want %q", r.createdTitle, "T")
	}
	if r.createdURL != "http://x.com" {
		t.Errorf("CreateWindow url = %q, want %q", r.createdURL, "http://x.com")
	}
}

func TestControlledMode_Navigate(t *testing.T) {
	r := &controlledModeRecorder{}
	exec := makeTestExecutor(r, "win1")
	conn := setupControlledMode(t, exec)

	cmd := map[string]any{
		"cmd":       "navigate",
		"id":        "r2",
		"window_id": "win1",
		"url":       "http://y.com",
	}
	reply := sendAndRecv(t, conn, cmd)

	if reply.ID != "r2" {
		t.Errorf("reply.ID = %q, want %q", reply.ID, "r2")
	}
	if reply.Error != "" {
		t.Errorf("unexpected error: %s", reply.Error)
	}
	if r.navigatedWin != "win1" {
		t.Errorf("NavigateWindow windowID = %q, want %q", r.navigatedWin, "win1")
	}
	if r.navigatedURL != "http://y.com" {
		t.Errorf("NavigateWindow url = %q, want %q", r.navigatedURL, "http://y.com")
	}
}

func TestControlledMode_Eval(t *testing.T) {
	r := &controlledModeRecorder{}
	exec := makeTestExecutor(r, "win1")
	conn := setupControlledMode(t, exec)

	cmd := map[string]any{
		"cmd":       "eval",
		"id":        "r3",
		"window_id": "win1",
		"js":        "1+1",
	}
	reply := sendAndRecv(t, conn, cmd)

	if reply.ID != "r3" {
		t.Errorf("reply.ID = %q, want %q", reply.ID, "r3")
	}
	if reply.Error != "" {
		t.Errorf("unexpected error: %s", reply.Error)
	}
	if r.evaledWin != "win1" {
		t.Errorf("EvalWindow windowID = %q, want %q", r.evaledWin, "win1")
	}
	if r.evaledJS != "1+1" {
		t.Errorf("EvalWindow js = %q, want %q", r.evaledJS, "1+1")
	}
}

func TestControlledMode_Close(t *testing.T) {
	r := &controlledModeRecorder{}
	exec := makeTestExecutor(r, "win1")
	conn := setupControlledMode(t, exec)

	cmd := map[string]any{
		"cmd":       "close",
		"id":        "r4",
		"window_id": "win1",
	}
	reply := sendAndRecv(t, conn, cmd)

	if reply.ID != "r4" {
		t.Errorf("reply.ID = %q, want %q", reply.ID, "r4")
	}
	if reply.Error != "" {
		t.Errorf("unexpected error: %s", reply.Error)
	}
	if r.closedWin != "win1" {
		t.Errorf("DestroyWindow windowID = %q, want %q", r.closedWin, "win1")
	}
}

func TestControlledMode_UnknownCmd(t *testing.T) {
	r := &controlledModeRecorder{}
	exec := makeTestExecutor(r, "win1")
	conn := setupControlledMode(t, exec)

	cmd := map[string]any{
		"cmd": "bogus",
		"id":  "r5",
	}
	reply := sendAndRecv(t, conn, cmd)

	if reply.ID != "r5" {
		t.Errorf("reply.ID = %q, want %q", reply.ID, "r5")
	}
	if reply.Error == "" {
		t.Error("expected non-empty error for unknown command, got empty")
	}
}
