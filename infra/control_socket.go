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
)

// ControlCmd is a window-management command from the backend subprocess.
type ControlCmd struct {
	Cmd      string         `json:"cmd"`
	ID       string         `json:"id"`
	WindowID string         `json:"window_id,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
	URL      string         `json:"url,omitempty"`
	JS       string         `json:"js,omitempty"`
}

// ControlReply is sent back to the backend after executing a command.
type ControlReply struct {
	ID       string `json:"id"`
	WindowID string `json:"window_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ControlSocket accepts commands from a backend subprocess over a Unix socket.
type ControlSocket struct {
	ln   net.Listener
	conn net.Conn
}

func controlSocketAddr() string {
	filename := fmt.Sprintf("control_%d.sock", os.Getpid())
	switch runtime.GOOS {
	case "windows":
		return `\\.\pipe\` + filename
	default:
		return filepath.Join(os.TempDir(), filename)
	}
}

// StartControlSocket binds a Unix socket and returns a channel of incoming commands.
// The caller must call Reply on the returned socket to respond to each command.
func StartControlSocket(sockPath string) (*ControlSocket, <-chan ControlCmd, error) {
	if sockPath == "" {
		sockPath = controlSocketAddr()
	}
	os.Remove(sockPath)
	os.Setenv("WINDOW_CONTROL_SOCK_PATH", sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, nil, fmt.Errorf("control socket listen: %w", err)
	}

	log.Println("Control socket:", sockPath)
	cs := &ControlSocket{ln: ln}
	ch := make(chan ControlCmd, 16)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				close(ch)
				return
			}
			cs.conn = conn
			go cs.readLoop(conn, ch)
		}
	}()

	return cs, ch, nil
}

func (cs *ControlSocket) readLoop(conn net.Conn, ch chan<- ControlCmd) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var cmd ControlCmd
		if err := json.Unmarshal(scanner.Bytes(), &cmd); err != nil {
			log.Printf("control socket: bad JSON: %v", err)
			continue
		}
		ch <- cmd
	}
}

// Reply sends a ControlReply back to the connected backend.
func (cs *ControlSocket) Reply(r ControlReply) {
	if cs.conn == nil {
		return
	}
	b, _ := json.Marshal(r)
	b = append(b, '\n')
	cs.conn.Write(b)
}

// Close shuts down the control socket.
func (cs *ControlSocket) Close() {
	cs.ln.Close()
}
