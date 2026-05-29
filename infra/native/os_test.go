package native

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestExecCommand_Echo(t *testing.T) {
	stdout, stderr, err := ExecCommand("echo", []string{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
}

func TestExecCommand_NonZeroExit(t *testing.T) {
	_, _, err := ExecCommand("false", nil)
	if err == nil {
		t.Error("expected error for non-zero exit")
	}
}

func TestExecCommand_Stderr(t *testing.T) {
	_, stderr, _ := ExecCommand("sh", []string{"-c", "echo errout >&2"})
	if !strings.Contains(stderr, "errout") {
		t.Errorf("expected stderr to contain 'errout', got %q", stderr)
	}
}

func TestGetEnv_Present(t *testing.T) {
	os.Setenv("TEST_NATIVE_OS_VAR", "abc123")
	defer os.Unsetenv("TEST_NATIVE_OS_VAR")
	if got := GetEnv("TEST_NATIVE_OS_VAR"); got != "abc123" {
		t.Errorf("got %q, want %q", got, "abc123")
	}
}

func TestGetEnv_Missing(t *testing.T) {
	os.Unsetenv("DEFINITELY_NOT_SET_XYZ")
	if got := GetEnv("DEFINITELY_NOT_SET_XYZ"); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestPlatform(t *testing.T) {
	p := Platform()
	if p != runtime.GOOS {
		t.Errorf("Platform() = %q, want %q", p, runtime.GOOS)
	}
}

func TestGetOSInfo(t *testing.T) {
	info := GetOSInfo()
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
}
