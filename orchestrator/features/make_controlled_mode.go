package orchfeatures

import (
	"fmt"
	"webview_gui/features"
	"webview_gui/infra"
)

// MakeControlledMode builds a ControlledMode backed by infra.ControlSocket.
func MakeControlledMode() features.ControlledMode {
	return features.ControlledMode{
		StartManagementSocket: func(sockPath string, exec features.WindowCommandExecutor) error {
			cs, cmds, err := infra.StartControlSocket(sockPath)
			if err != nil {
				return err
			}

			go func() {
				defer cs.Close()
				windows := map[string]string{} // id → windowID (for tracking)
				_ = windows

				for cmd := range cmds {
					switch cmd.Cmd {
					case "create_window":
						title, _ := cmd.Params["title"].(string)
						url, _ := cmd.Params["url"].(string)
						w, h := 800, 600
						if v, ok := cmd.Params["width"].(float64); ok {
							w = int(v)
						}
						if v, ok := cmd.Params["height"].(float64); ok {
							h = int(v)
						}
						winID, err := exec.CreateWindow(title, url, w, h)
						if err != nil {
							cs.Reply(infra.ControlReply{ID: cmd.ID, Error: err.Error()})
						} else {
							cs.Reply(infra.ControlReply{ID: cmd.ID, WindowID: winID})
						}

					case "navigate":
						err := exec.NavigateWindow(cmd.WindowID, cmd.URL)
						if err != nil {
							cs.Reply(infra.ControlReply{ID: cmd.ID, Error: err.Error()})
						} else {
							cs.Reply(infra.ControlReply{ID: cmd.ID, WindowID: cmd.WindowID})
						}

					case "eval":
						err := exec.EvalWindow(cmd.WindowID, cmd.JS)
						if err != nil {
							cs.Reply(infra.ControlReply{ID: cmd.ID, Error: err.Error()})
						} else {
							cs.Reply(infra.ControlReply{ID: cmd.ID, WindowID: cmd.WindowID})
						}

					case "close":
						err := exec.DestroyWindow(cmd.WindowID)
						if err != nil {
							cs.Reply(infra.ControlReply{ID: cmd.ID, Error: err.Error()})
						} else {
							cs.Reply(infra.ControlReply{ID: cmd.ID})
						}

					default:
						cs.Reply(infra.ControlReply{ID: cmd.ID, Error: fmt.Sprintf("unknown cmd: %s", cmd.Cmd)})
					}
				}
			}()

			return nil
		},
	}
}
