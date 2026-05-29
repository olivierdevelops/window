package orchusecases

import (
	"strings"
	"testing"
	"webview_gui/assets"
	"webview_gui/domain"
	"webview_gui/features"
)

type recorder struct {
	bindings []string
	inits    []string
	events   []string
	started  string
}

func fakeWin(r *recorder) features.Windowing {
	return features.Windowing{
		Bind: func(_ features.WindowHandle, name string, _ any) error {
			r.bindings = append(r.bindings, name)
			return nil
		},
		Init: func(_ features.WindowHandle, js string) {
			r.inits = append(r.inits, js)
		},
		SendEvent: func(_ features.WindowHandle, id string, _ any) {
			r.events = append(r.events, id)
		},
	}
}

func fakeBridge(r *recorder) features.BackendBridge {
	return features.BackendBridge{
		Start: func(script string) error {
			r.started = script
			return nil
		},
		OnServerPush: func(cb func(*domain.Message)) {},
		HandleRequest: func(fn string, data map[string]any, onReply func(*domain.Message)) (string, error) {
			return "id1", nil
		},
		Close: func() {},
	}
}

func containsBinding(bindings []string, name string) bool {
	for _, b := range bindings {
		if b == name {
			return true
		}
	}
	return false
}

func TestMakeInitWindow_AlwaysBindsLogFromJS(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	if !containsBinding(r.bindings, "logFromJS") {
		t.Errorf("bindings %v do not contain logFromJS", r.bindings)
	}
}

func TestMakeInitWindow_AlwaysInjectsBackendJS(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	if len(r.inits) == 0 {
		t.Fatal("no Init calls were made")
	}
	backendJS := string(assets.BackendJS)
	if backendJS == "" {
		t.Fatal("assets.BackendJS is empty — embed may be broken")
	}

	found := false
	for _, js := range r.inits {
		if strings.Contains(js, backendJS) || js == backendJS {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no Init call contained BackendJS content. inits: %v", r.inits)
	}
}

func TestMakeInitWindow_JSInjectLoop(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{
		JSInject: []string{"https://cdn.example.com/lib.js"},
	}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	found := false
	for _, js := range r.inits {
		if strings.Contains(js, "https://cdn.example.com/lib.js") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no Init call contained JSInject URL. inits: %v", r.inits)
	}
}

func TestMakeInitWindow_NoBackendScript(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{RunBackendScript: ""}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	if r.started != "" {
		t.Errorf("bridge.Start was called with %q, expected no call", r.started)
	}
}

func TestMakeInitWindow_WithBackendScript(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{RunBackendScript: "python3 main.py"}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	if r.started != "python3 main.py" {
		t.Errorf("bridge.Start called with %q, want %q", r.started, "python3 main.py")
	}
}

func TestMakeInitWindow_NativeFS_BindsAll(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{NativeFeatures: []domain.NativeFeature{domain.NativeFS}}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	for _, name := range []string{
		"__native_fs_readFile",
		"__native_fs_writeFile",
		"__native_fs_readDir",
		"__native_fs_watchFile",
	} {
		if !containsBinding(r.bindings, name) {
			t.Errorf("bindings %v do not contain %q", r.bindings, name)
		}
	}
}

func TestMakeInitWindow_NativeOS_BindsAll(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{NativeFeatures: []domain.NativeFeature{domain.NativeOS}}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	for _, name := range []string{
		"__native_os_exec",
		"__native_os_getEnv",
		"__native_os_platform",
		"__native_os_info",
	} {
		if !containsBinding(r.bindings, name) {
			t.Errorf("bindings %v do not contain %q", r.bindings, name)
		}
	}
}

func TestMakeInitWindow_NativeCanvas_BindsAll(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{NativeFeatures: []domain.NativeFeature{domain.NativeCanvas}}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	for _, name := range []string{
		"__native_canvas_drawRect",
		"__native_canvas_drawText",
		"__native_canvas_clear",
		"__native_canvas_flush",
	} {
		if !containsBinding(r.bindings, name) {
			t.Errorf("bindings %v do not contain %q", r.bindings, name)
		}
	}
}

func TestMakeInitWindow_CALL_BACKEND_Registered(t *testing.T) {
	r := &recorder{}
	win := fakeWin(r)
	bridge := fakeBridge(r)
	initFn := MakeInitWindow(win, bridge)

	cfg := &domain.AppConfig{RunBackendScript: "python3 main.py"}
	if err := initFn("h", cfg); err != nil {
		t.Fatalf("initFn: %v", err)
	}

	if !containsBinding(r.bindings, "__CALL_BACKEND") {
		t.Errorf("bindings %v do not contain __CALL_BACKEND", r.bindings)
	}
}
