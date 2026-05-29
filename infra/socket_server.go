package infra

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"webview_gui/domain"
)

type SocketServer struct {
	ln              net.Listener
	cmd             interface{ Kill() error }
	requestChannel  chan map[string]any
	responseChannel chan map[string]any
	conn            net.Conn
	requests        *sync.Map
	addr            string
}

func socketAddr() string {
	filename := fmt.Sprintf("echo_%d.sock", os.Getpid())
	switch runtime.GOOS {
	case "windows":
		return `\\.\pipe\` + filename
	default:
		return filepath.Join(os.TempDir(), filename)
	}
}

func NewSocketServer(script string) (*SocketServer, error) {
	addr := socketAddr()
	os.Remove(addr)

	ln, err := net.Listen("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	cmd := GetRunScriptCMD(script, nil)
	if cmd == nil {
		ln.Close()
		return nil, fmt.Errorf("failed to create command for: %s", script)
	}

	log.Println("Socket address:", addr)
	cmd.Env = append(os.Environ(), fmt.Sprintf("WINDOW_SOCK_PATH=%s", addr))
	cmd.Stdout = NewWriterWrapper("Stdout", os.Stdout)
	cmd.Stderr = NewWriterWrapper("Stderr", os.Stderr)

	return &SocketServer{
		ln:              ln,
		requestChannel:  make(chan map[string]any),
		responseChannel: make(chan map[string]any),
		requests:        &sync.Map{},
		cmd:             cmd.Process,
		addr:            addr,
	}, nil
}

// startSocketServer creates a server and starts the backend process.
// Returns the server ready to accept one connection.
func StartSocketServer(script string) (*SocketServer, error) {
	addr := socketAddr()
	os.Remove(addr)

	ln, err := net.Listen("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	cmd := GetRunScriptCMD(script, nil)
	if cmd == nil {
		ln.Close()
		return nil, fmt.Errorf("failed to create command for: %s", script)
	}

	log.Println("Socket address:", addr)
	cmd.Env = append(os.Environ(), fmt.Sprintf("WINDOW_SOCK_PATH=%s", addr))
	cmd.Stdout = NewWriterWrapper("Stdout", os.Stdout)
	cmd.Stderr = NewWriterWrapper("Stderr", os.Stderr)

	if err := cmd.Start(); err != nil {
		ln.Close()
		return nil, fmt.Errorf("failed to start backend: %w", err)
	}

	ss := &SocketServer{
		ln:       ln,
		requests: &sync.Map{},
		addr:     addr,
	}
	ss.cmd = cmd.Process

	// Wait for backend to connect
	connCh := make(chan net.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errCh <- err
			return
		}
		connCh <- conn
	}()

	select {
	case conn := <-connCh:
		ss.conn = conn
		log.Println(">> backend connected")
	case err := <-errCh:
		cmd.Process.Kill()
		ln.Close()
		return nil, fmt.Errorf("failed to accept backend connection: %w", err)
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		ln.Close()
		return nil, fmt.Errorf("timeout waiting for backend connection")
	}

	return ss, nil
}

func (ss *SocketServer) Close() {
	ss.ln.Close()
	if ss.cmd != nil {
		ss.cmd.Kill()
	}
}

func (ss *SocketServer) HandleMessages(onServerPush func(*domain.Message)) {
	defer ss.conn.Close()
	reader := bufio.NewReader(ss.conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("(ReadString)", err)
			continue
		}
		log.Println("[backend]:", line)

		var msg domain.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("JSON unmarshal error: %v for %q", err, line)
			continue
		}

		if onServerPush != nil && len(msg.ID) > 6 && msg.ID[:7] == "server:" {
			onServerPush(&msg)
			continue
		}

		val, ok := ss.requests.Load(msg.ID)
		if !ok {
			log.Printf("request ID not found: %s", msg.ID)
			continue
		}
		ch, ok := val.(chan *domain.Message)
		if !ok {
			log.Printf("invalid channel type for ID: %s", msg.ID)
			continue
		}
		select {
		case ch <- &msg:
		default:
			log.Printf("channel blocked for ID: %s", msg.ID)
		}
		if msg.Done {
			close(ch)
			ss.requests.Delete(msg.ID)
		}
	}
}

func (ss *SocketServer) HandleRequest(fn string, data map[string]any, onReply func(*domain.Message)) (string, error) {
	if ss.conn == nil {
		return "", fmt.Errorf("no active connection")
	}
	if onReply == nil {
		return "", fmt.Errorf("onReply is nil")
	}

	id := uuid.NewString()
	req := domain.Message{
		ID:       id,
		Data:     data,
		Function: fn,
		Done:     true,
	}
	b, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	ch := make(chan *domain.Message, 1)
	ss.requests.Store(id, ch)

	go func() {
		for msg := range ch {
			log.Println("GOT REPLY ->", msg.ID)
			if s, ok := msg.Data.(string); ok {
				var items map[string]any
				if err := json.Unmarshal([]byte(s), &items); err == nil {
					msg.Data = items
				}
			}
			onReply(msg)
		}
	}()

	b = append(b, '\n')
	if _, err := ss.conn.Write(b); err != nil {
		ss.requests.Delete(id)
		close(ch)
		return "", fmt.Errorf("failed to write request: %w", err)
	}

	return id, nil
}
