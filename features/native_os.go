package features

// NativeOS exposes OS-level capabilities to the frontend via bound JS functions.
type NativeOS struct {
	Exec     func(command string, args []string) (stdout string, stderr string, err error)
	GetEnv   func(key string) string
	Platform func() string
	OSInfo   func() OSInfo
}

type OSInfo struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}
