package orchfeatures

import (
	"webview_gui/features"
	infranative "webview_gui/infra/native"
)

// MakeNativeFS builds a NativeFS backed by OS file operations.
func MakeNativeFS() features.NativeFS {
	return features.NativeFS{
		ReadFile:  infranative.ReadFile,
		WriteFile: infranative.WriteFile,
		ReadDir: func(path string) ([]features.DirEntry, error) {
			entries, err := infranative.ReadDir(path)
			if err != nil {
				return nil, err
			}
			result := make([]features.DirEntry, len(entries))
			for i, e := range entries {
				result[i] = features.DirEntry{Name: e.Name, IsDir: e.IsDir, Size: e.Size}
			}
			return result, nil
		},
		WatchFile: infranative.WatchFile,
	}
}
