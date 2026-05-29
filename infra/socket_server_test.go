package infra

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"webview_gui/domain"
)

func shortSockPath(t *testing.T, prefix string) string {
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

// makeSocketServerWithConn creates a SocketServer wired to an in-process pair,
// bypassing the subprocess and real socket file.
func makeSocketServerWithConn(t *testing.T) (*SocketServer, net.Conn) {
	t.Helper()
	clientConn, serverConn := inProcessSocketPair(t)
	ss := &SocketServer{
		conn:     serverConn,
		requests: &sync.Map{},
	}
	return ss, clientConn
}

func inProcessSocketPair(t *testing.T) (client, server net.Conn) {
	t.Helper()
	sockPath := shortSockPath(t, "ss_pair")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}

	connCh := make(chan net.Conn, 1)
	go func() {
		c, _ := ln.Accept()
		ln.Close()
		connCh <- c
	}()

	client, err = net.Dial("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}
	server = <-connCh
	t.Cleanup(func() { client.Close(); server.Close() })
	return client, server
}

func TestHandleRequest_SendsJSON(t *testing.T) {
	ss, clientConn := makeSocketServerWithConn(t)

	replyCh := make(chan *domain.Message, 1)
	id, err := ss.HandleRequest("greet", map[string]any{"name": "World"}, func(msg *domain.Message) {
		replyCh <- msg
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Error("expected non-empty request ID")
	}

	// Read what was sent to the fake backend
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	scanner := bufio.NewScanner(clientConn)
	if !scanner.Scan() {
		t.Fatal("expected JSON line from server")
	}
	var sent domain.Message
	if err := json.Unmarshal(scanner.Bytes(), &sent); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if sent.Function != "greet" {
		t.Errorf("Function = %q, want greet", sent.Function)
	}
	if sent.ID != id {
		t.Errorf("ID = %q, want %q", sent.ID, id)
	}
}

func TestHandleMessages_DispatchesReply(t *testing.T) {
	ss, clientConn := makeSocketServerWithConn(t)

	replyCh := make(chan *domain.Message, 1)
	id, err := ss.HandleRequest("add", map[string]any{"x": 1}, func(msg *domain.Message) {
		replyCh <- msg
	})
	if err != nil {
		t.Fatal(err)
	}

	// Drain the sent request
	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	bufio.NewScanner(clientConn).Scan()
	clientConn.SetReadDeadline(time.Time{})

	// Run HandleMessages in background; it will exit when clientConn closes
	done := make(chan struct{})
	go func() {
		defer close(done)
		ss.HandleMessages(nil)
	}()

	// Send a reply from the fake backend
	reply := domain.Message{ID: id, Data: map[string]any{"value": 42}, Done: true}
	b, _ := json.Marshal(reply)
	clientConn.Write(append(b, '\n'))

	select {
	case msg := <-replyCh:
		if msg.ID != id {
			t.Errorf("ID = %q, want %q", msg.ID, id)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for reply")
	}

	// Close the client to stop HandleMessages
	clientConn.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}

func TestHandleMessages_ServerPush(t *testing.T) {
	ss, clientConn := makeSocketServerWithConn(t)

	pushCh := make(chan *domain.Message, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ss.HandleMessages(func(msg *domain.Message) {
			select {
			case pushCh <- msg:
			default:
			}
		})
	}()

	// Send a server-push message (ID starts with "server:")
	push := domain.Message{ID: "server:timer", Data: map[string]any{"tick": 1}}
	b, _ := json.Marshal(push)
	clientConn.Write(append(b, '\n'))

	select {
	case msg := <-pushCh:
		if msg.ID != "server:timer" {
			t.Errorf("ID = %q, want server:timer", msg.ID)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for server push")
	}

	clientConn.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}

func TestHandleRequest_NoConn(t *testing.T) {
	ss := &SocketServer{requests: &sync.Map{}}
	_, err := ss.HandleRequest("fn", nil, func(*domain.Message) {})
	if err == nil {
		t.Error("expected error when conn is nil")
	}
}

func TestHandleRequest_NilCallback(t *testing.T) {
	_, serverConn := inProcessSocketPair(t)
	ss := &SocketServer{conn: serverConn, requests: &sync.Map{}}
	_, err := ss.HandleRequest("fn", nil, nil)
	if err == nil {
		t.Error("expected error for nil callback")
	}
}
