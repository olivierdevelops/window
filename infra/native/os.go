package native

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
)

type OSInfo struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

func ExecCommand(command string, args []string) (stdout string, stderr string, err error) {
	var outBuf, errBuf bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func GetEnv(key string) string {
	return os.Getenv(key)
}

func Platform() string {
	return runtime.GOOS
}

func GetOSInfo() OSInfo {
	return OSInfo{OS: runtime.GOOS, Arch: runtime.GOARCH}
}
