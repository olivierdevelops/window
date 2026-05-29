package orchfeatures

import (
	"runtime"
	"strings"
	"testing"
)

func TestMakeNativeOS_Exec(t *testing.T) {
	nos := MakeNativeOS()
	stdout, _, err := nos.Exec("echo", []string{"hello"})
	if err != nil {
		t.Fatalf("Exec echo: %v", err)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("stdout = %q, want to contain %q", stdout, "hello")
	}
}

func TestMakeNativeOS_ExecError(t *testing.T) {
	nos := MakeNativeOS()
	_, _, err := nos.Exec("false", nil)
	if err == nil {
		t.Error("expected non-nil error from 'false', got nil")
	}
}

func TestMakeNativeOS_GetEnv(t *testing.T) {
	t.Setenv("TEST_ORCHFEATURES_GETENV", "orchvalue42")
	nos := MakeNativeOS()
	got := nos.GetEnv("TEST_ORCHFEATURES_GETENV")
	if got != "orchvalue42" {
		t.Errorf("GetEnv = %q, want %q", got, "orchvalue42")
	}
}

func TestMakeNativeOS_Platform(t *testing.T) {
	nos := MakeNativeOS()
	got := nos.Platform()
	if got != runtime.GOOS {
		t.Errorf("Platform() = %q, want %q", got, runtime.GOOS)
	}
}

func TestMakeNativeOS_OSInfo(t *testing.T) {
	nos := MakeNativeOS()
	info := nos.OSInfo()
	if info.OS != runtime.GOOS {
		t.Errorf("OSInfo.OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("OSInfo.Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
}
