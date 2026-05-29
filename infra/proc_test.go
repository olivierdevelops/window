package infra

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestGetRunScriptCMD_Default(t *testing.T) {
	cmd := GetRunScriptCMD("echo hello", nil)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "hello") {
		t.Errorf("expected output to contain 'hello', got %q", out)
	}
}

func TestGetRunScriptCMD_CustomRunner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	// runner="echo" + script="runner_test" → exec.Command("echo", "runner_test")
	runner := "echo"
	cmd := GetRunScriptCMD("runner_test", &runner)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "runner_test") {
		t.Errorf("expected output to contain 'runner_test', got %q", out)
	}
}

func TestWriterWrapper(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriterWrapper("test", &buf)
	n, err := w.Write([]byte("hello\n"))
	if err != nil {
		t.Fatal(err)
	}
	if n != len("hello\n") {
		t.Errorf("Write returned n=%d, want %d", n, len("hello\n"))
	}
	got := buf.String()
	if !strings.Contains(got, "[backend]") {
		t.Errorf("expected prefix in output, got %q", got)
	}
	if !strings.Contains(got, "test") {
		t.Errorf("expected tag in output, got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("expected content in output, got %q", got)
	}
}
