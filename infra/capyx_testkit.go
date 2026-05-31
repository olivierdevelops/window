package infra

import (
	"encoding/json"
	"strings"
)

// capyx_testkit.go — the .capyx isolation-test harness build. Where the normal
// build auto-mounts the app, the harness build exposes every component, handler
// factory, capability provider and an introspection meta on
// globalThis.__CAPYX_TEST__, so a test kernel can mount any single component in
// full isolation, override its initial state, replace its capabilities with
// recording mocks, drive events, and assert on the resulting DOM/state — with no
// browser. The interactive Test Bench and the headless .capytest runner both
// build on this surface.

// CompileCapyxHarness transpiles a .capyx source into a self-contained harness
// page: window.yaml plus static/index.html with the signals runtime, the app's
// definitions exposed on __CAPYX_TEST__, and the test kernel inlined (in that
// script order, so the kernel sees the registry). testkitJS is the contents of
// capyx_testkit.js.
func CompileCapyxHarness(src, runtimeJS, testkitJS string) (map[string]string, error) {
	app, err := parseCapyx(src)
	if err != nil {
		return nil, err
	}
	app.runtime = runtimeJS
	if err := app.validate(); err != nil {
		return nil, err
	}
	harnessJS, err := app.genHarnessJS()
	if err != nil {
		return nil, err
	}
	index, err := app.genHarnessHTML(harnessJS, testkitJS)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"window.yaml":       app.genWindowYAML(),
		"static/index.html": index,
	}, nil
}

// CompileCapyxHarnessJS returns just the harness application JavaScript (the
// definitions plus the __CAPYX_TEST__ registry), without HTML or runtime. The
// headless .capytest runner assembles this with the DOM shim, runtime, kernel
// and a driver into a single Node program.
func CompileCapyxHarnessJS(src, runtimeJS string) (string, error) {
	app, err := parseCapyx(src)
	if err != nil {
		return "", err
	}
	app.runtime = runtimeJS
	if err := app.validate(); err != nil {
		return "", err
	}
	return app.genHarnessJS()
}

// genHarnessJS emits the app definitions plus a globalThis.__CAPYX_TEST__
// registry instead of a mount/bootstrap. The registry hands the kernel the
// runtime, the live providers object, every component render fn and handler
// factory, the capability→provider wiring, and an introspection meta.
func (app *capyxApp) genHarnessJS() (string, error) {
	var b strings.Builder
	b.WriteString("(function(){\n\"use strict\";\n")
	if err := app.genDefsJS(&b); err != nil {
		return "", err
	}

	b.WriteString("globalThis.__CAPYX_TEST__ = {\n")
	b.WriteString("CAPYX: CAPYX,\n")
	b.WriteString("providers: PROVIDERS,\n")

	b.WriteString("components: {")
	for i, c := range app.components {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(jsStr(c.name) + ":C_" + jsName(c.name))
	}
	b.WriteString("},\n")

	b.WriteString("handlers: {")
	for i, h := range app.handlers {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(jsStr(h.name) + ":H_" + jsName(h.name))
	}
	b.WriteString("},\n")

	capImpl, _ := json.Marshal(app.capImpl)
	b.WriteString("capImpl: " + string(capImpl) + ",\n")

	orch := "null"
	if app.orch != nil {
		orch = jsStr(app.orch.name)
	}
	b.WriteString("orchestrator: " + orch + ",\n")

	meta, err := json.Marshal(app.harnessMeta())
	if err != nil {
		return "", err
	}
	b.WriteString("meta: " + string(meta) + "\n")
	b.WriteString("};\n})();\n")
	return b.String(), nil
}

// harnessMeta is the JSON-serialisable introspection the kernel and Test Bench
// read to build their UI: what components exist, which handler each defaults to,
// each handler's state fields / event methods / capability ports, and the method
// set of every capability.
func (app *capyxApp) harnessMeta() map[string]any {
	handlerNames := map[string]bool{}
	for _, h := range app.handlers {
		handlerNames[h.name] = true
	}

	components := map[string]any{}
	for _, c := range app.components {
		dh := ""
		if handlerNames[c.name] {
			dh = c.name
		}
		components[c.name] = map[string]any{
			"handlerUsed":    handlerUsed(app, c.name),
			"defaultHandler": dh,
		}
	}

	handlers := map[string]any{}
	for _, h := range app.handlers {
		state := []map[string]any{}
		for _, s := range h.state {
			state = append(state, map[string]any{"name": s.name, "expr": s.expr})
		}
		methods := []map[string]any{}
		for _, m := range h.methods {
			methods = append(methods, map[string]any{"name": m.name, "arity": arity(m.args)})
		}
		ports := []map[string]any{}
		for _, p := range h.ports {
			ports = append(ports, map[string]any{"cap": p.cap, "name": p.name})
		}
		handlers[h.name] = map[string]any{"state": state, "methods": methods, "ports": ports}
	}

	// Capability → method names, learned from the providers that implement them
	// (plus the orchestrator, which is injectable as a capability).
	capabilities := map[string]any{}
	for _, p := range app.providers {
		names := []string{}
		for _, m := range p.methods {
			names = append(names, m.name)
		}
		capabilities[p.cap] = names
	}
	if app.orch != nil {
		names := []string{}
		for _, m := range app.orch.methods {
			names = append(names, m.name)
		}
		capabilities[app.orch.name] = names
	}

	return map[string]any{
		"app":          map[string]any{"title": app.title, "width": app.width, "height": app.height},
		"components":   components,
		"handlers":     handlers,
		"capabilities": capabilities,
	}
}

// arity counts the comma-separated parameters in a method's arg string.
func arity(args string) int {
	args = strings.TrimSpace(args)
	if args == "" {
		return 0
	}
	return strings.Count(args, ",") + 1
}

// capyxBenchCSS styles the interactive Test Bench chrome (cb-* classes); the
// component preview itself keeps its own scoped (cz-*) styles.
const capyxBenchCSS = `
.cb-header{display:flex;align-items:baseline;gap:8px;padding:6px 4px 12px}
.cb-sub{color:#8b8b9e;font-size:13px}
.cb-main{display:grid;grid-template-columns:180px 1fr 300px;gap:14px;align-items:start}
.cb-h{font-size:12px;text-transform:uppercase;letter-spacing:.04em;color:#8b8b9e;margin:10px 0 6px}
.cb-left,.cb-center,.cb-right{min-width:0}
.cb-list{display:flex;flex-direction:column;gap:4px}
.cb-comp{text-align:left;border:1px solid #e6e6ef;background:#fff;border-radius:8px;padding:7px 10px;cursor:pointer;font:inherit}
.cb-comp:hover{background:#f0f0f7}
.cb-active{background:#7c7cff;color:#fff;border-color:#7c7cff}
.cb-preview{border:1px dashed #c9c9de;border-radius:12px;padding:14px;background:#fff;min-height:80px}
.cb-readout{background:#0f1021;color:#d7d7ef;border-radius:8px;padding:8px 10px;font:12px/1.45 ui-monospace,Menlo,monospace;white-space:pre-wrap;margin:0 0 6px;max-height:140px;overflow:auto}
.cb-field{display:flex;gap:6px;align-items:center;margin-bottom:6px}
.cb-label{font-size:12px;color:#5a5a6e;min-width:64px}
.cb-input{flex:1;min-width:0;font:13px ui-monospace,Menlo,monospace}
.cb-btn{border:1px solid #d8d8e6;background:#fff;border-radius:8px;padding:5px 10px;cursor:pointer;font:13px inherit}
.cb-btn:hover{background:#f0f0f7}
.cb-primary{background:#7c7cff;color:#fff;border-color:#7c7cff}
.cb-row{display:flex;gap:6px;align-items:center;flex-wrap:wrap;margin-bottom:6px}
.cb-bottom{margin-top:16px;border-top:1px solid #e6e6ef;padding-top:10px}
.cb-scenario{background:#0f1021;color:#9effa5;border-radius:8px;padding:10px 12px;font:12px/1.5 ui-monospace,Menlo,monospace;white-space:pre-wrap;margin:0 0 8px}
.cb-status{font-weight:600;margin:4px 0}
.cb-status.cb-ok{color:#1b7a3d}
.cb-status.cb-fail{color:#c0392b}
.cb-steps{display:flex;flex-direction:column;gap:2px}
.cb-step{display:flex;gap:8px;align-items:baseline;font-size:13px;padding:2px 6px;border-radius:6px}
.cb-step.cb-ok{background:#eafaf0}
.cb-step.cb-fail{background:#fdecea}
.cb-mark{width:14px}
.cb-step.cb-ok .cb-mark{color:#1b7a3d}
.cb-step.cb-fail .cb-mark{color:#c0392b}
.cb-op{font-family:ui-monospace,Menlo,monospace}
.cb-msg{color:#8b8b9e}
#app{max-width:none}
`

// CompileCapyxBench transpiles a .capyx source into the interactive Test Bench
// page: the app compiled in harness mode (exposing __CAPYX_TEST__) plus the
// test kernel and the bench UI, wrapped in a bench layout. The bench mounts each
// component in an isolated preview and records interactions as a runnable
// .capytest. testkitJS and benchJS are the embedded assets.
func CompileCapyxBench(src, runtimeJS, testkitJS, benchJS string) (map[string]string, error) {
	app, err := parseCapyx(src)
	if err != nil {
		return nil, err
	}
	app.runtime = runtimeJS
	if err := app.validate(); err != nil {
		return nil, err
	}
	harnessJS, err := app.genHarnessJS()
	if err != nil {
		return nil, err
	}

	var css strings.Builder
	css.WriteString(capyxBaseCSS)
	css.WriteString(app.genThemeCSS())
	for _, c := range app.components {
		if strings.TrimSpace(c.css) != "" {
			css.WriteString("\n/* " + c.name + " */\n")
			css.WriteString(c.css)
		}
	}
	css.WriteString(capyxBenchCSS)

	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>" + htmlEscape(app.title) + " — Test Bench</title>\n")
	b.WriteString("<style>\n" + css.String() + "</style>\n")
	b.WriteString("</head>\n<body>\n<div id=\"capyx-bench\"></div>\n")
	b.WriteString("<script>\n" + app.runtime + "\n</script>\n")
	b.WriteString("<script>\n" + harnessJS + "\n</script>\n")
	b.WriteString("<script>\n" + testkitJS + "\n</script>\n")
	b.WriteString("<script>\n" + benchJS + "\n</script>\n")
	b.WriteString("</body>\n</html>\n")

	// A roomier default window for the bench.
	app.width = max2(app.width, 1024)
	app.height = max2(app.height, 720)
	return map[string]string{
		"window.yaml":       app.genWindowYAML(),
		"static/index.html": b.String(),
	}, nil
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// genHarnessHTML wraps the harness scripts into a page: base CSS + theme +
// component CSS, then runtime, harness defs (exposing __CAPYX_TEST__), and the
// test kernel — in that order so the kernel sees the registry.
func (app *capyxApp) genHarnessHTML(harnessJS, testkitJS string) (string, error) {
	var css strings.Builder
	css.WriteString(capyxBaseCSS)
	css.WriteString(app.genThemeCSS())
	for _, c := range app.components {
		if strings.TrimSpace(c.css) != "" {
			css.WriteString("\n/* " + c.name + " */\n")
			css.WriteString(c.css)
		}
	}
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>" + htmlEscape(app.title) + " — harness</title>\n")
	b.WriteString("<style>\n" + css.String() + "</style>\n")
	b.WriteString("</head>\n<body>\n<div id=\"app\"></div>\n")
	b.WriteString("<script>\n" + app.runtime + "\n</script>\n")
	b.WriteString("<script>\n" + harnessJS + "\n</script>\n")
	b.WriteString("<script>\n" + testkitJS + "\n</script>\n")
	b.WriteString("</body>\n</html>\n")
	return b.String(), nil
}
