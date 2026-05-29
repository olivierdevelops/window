package features

// ControlledMode lets a backend subprocess drive the window manager.
type ControlledMode struct {
	StartManagementSocket func(sockPath string, executor WindowCommandExecutor) error
}

// WindowCommandExecutor is the shape of what the controlled mode needs
// to execute commands received from the backend.
type WindowCommandExecutor struct {
	CreateWindow   func(title, url string, width, height int) (windowID string, err error)
	DestroyWindow  func(windowID string) error
	NavigateWindow func(windowID, url string) error
	EvalWindow     func(windowID, js string) error
}
