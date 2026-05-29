package orchfeatures

import (
	"webview_gui/features"
	infranative "webview_gui/infra/native"
)

// MakeNativeOS builds a NativeOS backed by OS-level operations.
func MakeNativeOS() features.NativeOS {
	return features.NativeOS{
		Exec:     infranative.ExecCommand,
		GetEnv:   infranative.GetEnv,
		Platform: infranative.Platform,
		OSInfo: func() features.OSInfo {
			info := infranative.GetOSInfo()
			return features.OSInfo{OS: info.OS, Arch: info.Arch}
		},
	}
}
