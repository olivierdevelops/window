package orchfeatures

import (
	"webview_gui/domain"
	"webview_gui/features"
	"webview_gui/infra"
)

// MakeBackendBridge builds a BackendBridge backed by infra.SocketServer.
func MakeBackendBridge() features.BackendBridge {
	var ss *infra.SocketServer
	var pushCb func(*domain.Message)

	return features.BackendBridge{
		Start: func(script string) error {
			server, err := infra.StartSocketServer(script)
			if err != nil {
				return err
			}
			ss = server
			go ss.HandleMessages(func(msg *domain.Message) {
				if pushCb != nil {
					pushCb(msg)
				}
			})
			return nil
		},
		HandleRequest: func(fn string, data map[string]any, onReply func(*domain.Message)) (string, error) {
			return ss.HandleRequest(fn, data, onReply)
		},
		OnServerPush: func(cb func(*domain.Message)) {
			pushCb = cb
		},
		Close: func() {
			if ss != nil {
				ss.Close()
			}
		},
	}
}
