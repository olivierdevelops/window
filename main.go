package main

import (
	"webview_gui/appio"
	"webview_gui/orchestrator"
)

func main() {
	cfg := appio.ParseCLI()
	if cfg == nil {
		return
	}
	orchestrator.Run(cfg)
}
