package features

// NativeFS exposes file-system capabilities to the frontend via bound JS functions.
type NativeFS struct {
	ReadFile  func(path string) ([]byte, error)
	WriteFile func(path string, data []byte, perm uint32) error
	ReadDir   func(path string) ([]DirEntry, error)
	WatchFile func(path string, onChange func([]byte)) (stop func(), err error)
}

type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}
