package orchfeatures

import (
	"strings"
	"testing"
	"webview_gui/features"
)

func makeTestCanvas(t *testing.T) (features.NativeCanvas, *[]string) {
	t.Helper()
	var evalCalls []string
	win := features.Windowing{
		Eval: func(h features.WindowHandle, js string) {
			evalCalls = append(evalCalls, js)
		},
	}
	h := "testhandle"
	nc := MakeNativeCanvas(win, h)
	return nc, &evalCalls
}

func TestMakeNativeCanvas_DrawRect(t *testing.T) {
	nc, calls := makeTestCanvas(t)
	err := nc.DrawRect(features.CanvasRect{
		CanvasID: "c",
		X:        10, Y: 20, W: 100, H: 50,
		Color: "#ff0000",
		Fill:  true,
	})
	if err != nil {
		t.Fatalf("DrawRect: %v", err)
	}
	if len(*calls) == 0 {
		t.Fatal("Eval was not called")
	}
	js := (*calls)[0]
	for _, want := range []string{"fillRect", "#ff0000", "c"} {
		if !strings.Contains(js, want) {
			t.Errorf("DrawRect JS missing %q in: %s", want, js)
		}
	}
}

func TestMakeNativeCanvas_DrawRect_Stroke(t *testing.T) {
	nc, calls := makeTestCanvas(t)
	err := nc.DrawRect(features.CanvasRect{
		CanvasID: "c",
		X:        0, Y: 0, W: 50, H: 50,
		Color: "#00ff00",
		Fill:  false,
	})
	if err != nil {
		t.Fatalf("DrawRect stroke: %v", err)
	}
	if len(*calls) == 0 {
		t.Fatal("Eval was not called")
	}
	js := (*calls)[0]
	if !strings.Contains(js, "strokeRect") {
		t.Errorf("DrawRect stroke JS missing %q in: %s", "strokeRect", js)
	}
}

func TestMakeNativeCanvas_DrawText(t *testing.T) {
	nc, calls := makeTestCanvas(t)
	err := nc.DrawText(features.CanvasText{
		CanvasID: "c",
		X:        5, Y: 15,
		Text:  "hi",
		Color: "blue",
	})
	if err != nil {
		t.Fatalf("DrawText: %v", err)
	}
	if len(*calls) == 0 {
		t.Fatal("Eval was not called")
	}
	js := (*calls)[0]
	for _, want := range []string{"fillText", "hi", "c"} {
		if !strings.Contains(js, want) {
			t.Errorf("DrawText JS missing %q in: %s", want, js)
		}
	}
}

func TestMakeNativeCanvas_DrawText_DefaultFont(t *testing.T) {
	nc, calls := makeTestCanvas(t)
	err := nc.DrawText(features.CanvasText{
		CanvasID: "c",
		X:        0, Y: 0,
		Text:  "hello",
		Font:  "",
		Color: "black",
	})
	if err != nil {
		t.Fatalf("DrawText default font: %v", err)
	}
	if len(*calls) == 0 {
		t.Fatal("Eval was not called")
	}
	js := (*calls)[0]
	if !strings.Contains(js, "sans-serif") {
		t.Errorf("DrawText default font JS missing %q in: %s", "sans-serif", js)
	}
}

func TestMakeNativeCanvas_Clear(t *testing.T) {
	nc, calls := makeTestCanvas(t)
	err := nc.Clear("c")
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if len(*calls) == 0 {
		t.Fatal("Eval was not called")
	}
	js := (*calls)[0]
	for _, want := range []string{"clearRect", "c"} {
		if !strings.Contains(js, want) {
			t.Errorf("Clear JS missing %q in: %s", want, js)
		}
	}
}

func TestMakeNativeCanvas_Flush(t *testing.T) {
	nc, calls := makeTestCanvas(t)
	err := nc.Flush("c")
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if len(*calls) != 0 {
		t.Errorf("Flush should not call Eval, but got %d call(s)", len(*calls))
	}
}
