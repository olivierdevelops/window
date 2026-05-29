package features

// NativeDialogs exposes system dialog capabilities to the frontend.
type NativeDialogs struct {
	OpenFile    func(opts FileDialogOptions) ([]string, error)
	SaveFile    func(opts FileDialogOptions) (string, error)
	ShowMessage func(opts MessageDialogOptions) error
}

type FileDialogOptions struct {
	Title   string   `json:"title"`
	Filters []string `json:"filters"`
	Multi   bool     `json:"multi"`
}

type MessageDialogOptions struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Kind    string `json:"kind"` // "info" | "warn" | "error"
}
