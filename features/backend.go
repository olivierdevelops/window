package features

import "webview_gui/domain"

// BackendBridge manages IPC with a subprocess backend over a Unix socket.
type BackendBridge struct {
	Start         func(script string) error
	HandleRequest func(fn string, data map[string]any, onReply func(*domain.Message)) (id string, err error)
	OnServerPush  func(cb func(*domain.Message))
	Close         func()
}
