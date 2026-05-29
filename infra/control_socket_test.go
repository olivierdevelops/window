package infra

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

func shortTempSock(t *testing.T, prefix string) string {
	t.Helper()
	f, err := os.CreateTemp("", fmt.Sprintf("%s_*.sock", prefix))
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path)
	t.Cleanup(func() { os.Remove(path) })
	return path
}

func TestControlSocket_RoundTrip(t *testing.T) {
	sockPath := shortTempSock(t, "ctl_rt")

	cs, cmdCh, err := StartControlSocket(sockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	// Verify env var is set
	if got := os.Getenv("WINDOW_CONTROL_SOCK_PATH"); got != sockPath {
		t.Errorf("WINDOW_CONTROL_SOCK_PATH = %q, want %q", got, sockPath)
	}

	// Connect as fake backend
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Give Accept loop time to register conn on cs
	time.Sleep(50 * time.Millisecond)

	// Send a command
	cmd := ControlCmd{
		Cmd: "create_window",
		ID:  "req1",
		Params: map[string]any{
			"title": "Test",
			"url":   "https://example.com",
		},
	}
	b, _ := json.Marshal(cmd)
	b = append(b, '\n')
	conn.Write(b)

	// Receive command on Go side
	select {
	case received := <-cmdCh:
		if received.Cmd != "create_window" {
			t.Errorf("Cmd = %q, want create_window", received.Cmd)
		}
		if received.ID != "req1" {
			t.Errorf("ID = %q, want req1", received.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for command")
	}

	// Send reply and read it back on the fake backend side
	go cs.Reply(ControlReply{ID: "req1", WindowID: "win1"})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("expected reply line")
	}
	var reply ControlReply
	if err := json.Unmarshal(scanner.Bytes(), &reply); err != nil {
		t.Fatalf("bad reply JSON: %v", err)
	}
	if reply.ID != "req1" {
		t.Errorf("reply ID = %q, want req1", reply.ID)
	}
	if reply.WindowID != "win1" {
		t.Errorf("reply WindowID = %q, want win1", reply.WindowID)
	}
}

func TestControlSocket_Close(t *testing.T) {
	sockPath := shortTempSock(t, "ctl_cl")
	cs, cmdCh, err := StartControlSocket(sockPath)
	if err != nil {
		t.Fatal(err)
	}

	cs.Close()

	// Channel should be closed when listener closes
	select {
	case _, ok := <-cmdCh:
		if ok {
			t.Error("expected channel to be closed")
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for channel to close")
	}
}
