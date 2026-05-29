package features

// NativeCanvas exposes a 2D drawing API backed by an HTML5 canvas element.
// The orchestrator wires these capabilities to window.Eval() calls.
type NativeCanvas struct {
	DrawRect func(opts CanvasRect) error
	DrawText func(opts CanvasText) error
	Clear    func(canvasID string) error
	Flush    func(canvasID string) error
}

type CanvasRect struct {
	CanvasID string  `json:"canvas_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
	Color    string  `json:"color"`
	Fill     bool    `json:"fill"`
}

type CanvasText struct {
	CanvasID string  `json:"canvas_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Text     string  `json:"text"`
	Font     string  `json:"font"`
	Color    string  `json:"color"`
}
