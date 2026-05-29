package infra

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// GetRunScriptCMD builds an exec.Cmd to run the given script string.
func GetRunScriptCMD(script string, runcmd *string) *exec.Cmd {
	if runcmd != nil && *runcmd != "" {
		parts := strings.Fields(*runcmd + " " + script)
		return exec.Command(parts[0], parts[1:]...)
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd.exe", "/C", script)
	default:
		return exec.Command("sh", "-c", script)
	}
}

// WriterWrapper prefixes each write with a tag for log identification.
type WriterWrapper struct {
	Tag string
	w   io.Writer
}

func (w *WriterWrapper) Write(p []byte) (n int, err error) {
	prefix := fmt.Sprintf("[backend] [%s]: ", w.Tag)
	data := append([]byte(prefix), p...)
	w.w.Write(data)
	return len(p), nil
}

func NewWriterWrapper(tag string, w io.Writer) *WriterWrapper {
	return &WriterWrapper{Tag: tag, w: w}
}
