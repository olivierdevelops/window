package orchfeatures

import (
	"fmt"
	"webview_gui/features"
)

// MakeNativeCanvas builds a NativeCanvas that issues Eval calls on the given window.
func MakeNativeCanvas(win features.Windowing, h features.WindowHandle) features.NativeCanvas {
	eval := func(js string) {
		win.Eval(h, js)
	}
	return features.NativeCanvas{
		DrawRect: func(r features.CanvasRect) error {
			eval(fmt.Sprintf(`(function(){
				var c=document.getElementById(%q); if(!c) return;
				var x=c.getContext('2d');
				x.fillStyle=%q; x.strokeStyle=%q;
				if(%t){x.fillRect(%g,%g,%g,%g);}else{x.strokeRect(%g,%g,%g,%g);}
			})()`, r.CanvasID, r.Color, r.Color, r.Fill,
				r.X, r.Y, r.W, r.H, r.X, r.Y, r.W, r.H))
			return nil
		},
		DrawText: func(t features.CanvasText) error {
			font := t.Font
			if font == "" {
				font = "16px sans-serif"
			}
			eval(fmt.Sprintf(`(function(){
				var c=document.getElementById(%q); if(!c) return;
				var x=c.getContext('2d');
				x.fillStyle=%q; x.font=%q;
				x.fillText(%q,%g,%g);
			})()`, t.CanvasID, t.Color, font, t.Text, t.X, t.Y))
			return nil
		},
		Clear: func(canvasID string) error {
			eval(fmt.Sprintf(`(function(){
				var c=document.getElementById(%q); if(!c) return;
				c.getContext('2d').clearRect(0,0,c.width,c.height);
			})()`, canvasID))
			return nil
		},
		Flush: func(canvasID string) error {
			return nil
		},
	}
}
