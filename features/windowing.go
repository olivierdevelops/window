package features

import "webview_gui/domain"

// Windowing is the set of capabilities for managing native windows.
// All fields are functions; the orchestrator builds the struct by plugging in closures.
type Windowing struct {
	New        func(title string, size *domain.WindowSize, debug bool) (WindowHandle, error)
	Destroy    func(h WindowHandle)
	Navigate   func(h WindowHandle, url string)
	Eval       func(h WindowHandle, js string)
	Bind       func(h WindowHandle, name string, fn any) error
	Init       func(h WindowHandle, js string)
	Run        func(h WindowHandle)
	SetTitle   func(h WindowHandle, title string)
	SendEvent  func(h WindowHandle, eventID string, data any)
}

// WindowHandle is an opaque reference to a native window.
// The concrete type (*infra.WebviewHandle) is known only to the orchestrator,
// which does type assertions when passing handles to infra functions.
type WindowHandle = any
