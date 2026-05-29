package infra

import (
	"encoding/json"
	"fmt"
	"log"

	webview "github.com/webview/webview_go"
)

// WebviewHandle is the infra-layer concrete window reference.
// The orchestrator casts features.WindowHandle → *WebviewHandle.
type WebviewHandle struct {
	WV webview.WebView
}

// NewWebviewHandle creates a new native window.
func NewWebviewHandle(title string, width, height int, debug bool) *WebviewHandle {
	wv := webview.New(debug)
	wv.SetTitle(title)
	wv.SetSize(width, height, webview.HintNone)
	return &WebviewHandle{WV: wv}
}

func DestroyWebview(h *WebviewHandle) {
	h.WV.Destroy()
}

func NavigateWebview(h *WebviewHandle, url string) {
	h.WV.Navigate(url)
}

func EvalWebview(h *WebviewHandle, js string) {
	h.WV.Eval(js)
}

func BindWebview(h *WebviewHandle, name string, fn any) error {
	return h.WV.Bind(name, fn)
}

func InitWebview(h *WebviewHandle, js string) {
	h.WV.Init(js)
}

func RunWebview(h *WebviewHandle) {
	h.WV.Run()
}

func SetTitleWebview(h *WebviewHandle, title string) {
	h.WV.SetTitle(title)
}

func SendEventWebview(h *WebviewHandle, eventID string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		log.Println("[SendEvent] marshal error:", err)
		return
	}
	h.WV.Eval(fmt.Sprintf(
		`window.dispatchEvent(new CustomEvent(%q, { detail: %s }))`,
		eventID, string(b),
	))
}
