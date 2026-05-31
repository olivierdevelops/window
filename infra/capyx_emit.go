package infra

import (
	"fmt"
	"regexp"
	"strings"
)

// capyx_emit.go — assembles a parsed .capyx app into window.yaml and a single
// static/index.html with scoped CSS, the signals runtime, and the compiled
// component/handler/capability/orchestrator JavaScript inlined.

const capyxBaseCSS = `*{box-sizing:border-box}
html,body{margin:0;padding:0}
body{font:15px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;
  color:#1d1d29;background:#f6f6fb;padding:20px}
#app{max-width:920px;margin:0 auto}
.contents,.capyx-comp{display:contents}
h1,h2,h3{margin:0 0 10px;line-height:1.2}
.muted{color:var(--muted,#8b8b9e)}
.row{display:flex;gap:10px;align-items:center;flex-wrap:wrap}
.col{display:flex;flex-direction:column;gap:10px}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:12px}
.card{background:#fff;border:1px solid #e6e6ef;border-radius:var(--radius,12px);padding:16px;
  box-shadow:0 1px 2px rgba(20,20,40,.04)}
.btn,.pill{display:inline-flex;align-items:center;gap:6px;border:1px solid #d8d8e6;background:#fff;
  color:#1d1d29;border-radius:8px;padding:7px 14px;cursor:pointer;font:inherit}
.btn:hover,.pill:hover{background:#f0f0f7}
.pill{background:var(--primary,#7c7cff);color:#fff;border:none;border-radius:999px}
.pill:hover{filter:brightness(1.07)}
.badge{display:inline-block;background:#eef;border-radius:999px;padding:2px 10px;font-size:12px;color:#5a5ad8}
input,select,textarea{font:inherit;padding:8px 10px;border:1px solid #d8d8e6;border-radius:8px;background:#fff}
input:focus,select:focus,textarea:focus{outline:2px solid var(--primary,#7c7cff);outline-offset:0}
ul{list-style:none;margin:0;padding:0}
li{padding:6px 0}
s{color:#9a9aab}
`

func (app *capyxApp) genWindowYAML() string {
	var b strings.Builder
	b.WriteString("title: " + jsStr(app.title) + "\n")
	b.WriteString("size:\n")
	b.WriteString(fmt.Sprintf("  width: %d\n", app.width))
	b.WriteString(fmt.Sprintf("  height: %d\n", app.height))
	b.WriteString("entry_path: \"./static/index.html\"\n")
	b.WriteString("static_dirs:\n")
	b.WriteString("  \"/static\": ./static/\n")
	return b.String()
}

func (app *capyxApp) genIndexHTML() (string, error) {
	appJS, err := app.genAppJS()
	if err != nil {
		return "", err
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
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>" + htmlEscape(app.title) + "</title>\n")
	b.WriteString("<style>\n" + css.String() + "</style>\n")
	b.WriteString("</head>\n<body>\n<div id=\"app\"></div>\n")
	b.WriteString("<script>\n" + app.runtime + "\n</script>\n")
	b.WriteString("<script>\n" + appJS + "\n</script>\n")
	b.WriteString("</body>\n</html>\n")
	return b.String(), nil
}

func (app *capyxApp) genThemeCSS() string {
	if len(app.theme) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n:root{")
	for k, v := range app.theme {
		b.WriteString("--" + k + ":" + v + ";")
	}
	b.WriteString("}\n")
	return b.String()
}

func (app *capyxApp) genAppJS() (string, error) {
	var b strings.Builder
	b.WriteString("(function(){\n\"use strict\";\n")
	if err := app.genDefsJS(&b); err != nil {
		return "", err
	}

	// Mounts + bootstrap.
	b.WriteString("function __capyxStart(){\nvar __root = document.getElementById(\"app\");\n")
	handlerNames := map[string]bool{}
	for _, h := range app.handlers {
		handlerNames[h.name] = true
	}
	for _, m := range app.mounts {
		hexpr := "null"
		if handlerNames[m.handler] {
			hexpr = "H_" + jsName(m.handler) + "()"
		}
		b.WriteString("CAPYX.mount(__root, (function(__H){return function(){return C_" +
			jsName(m.component) + "(__H);};})(" + hexpr + "));\n")
	}
	if app.orch != nil && app.orch.boot != nil {
		b.WriteString("if (APP && APP.boot) APP.boot();\n")
	}
	b.WriteString("}\nif (document.readyState !== \"loading\") __capyxStart(); else document.addEventListener(\"DOMContentLoaded\", __capyxStart);\n")
	b.WriteString("})();\n")
	return b.String(), nil
}

// genDefsJS emits the shared definitions of an app — capability providers, the
// optional orchestrator (as APP), handler factories (H_<name>), and component
// render functions (C_<name>) — without any mount/bootstrap. Both the runnable
// app build and the isolation harness build reuse it.
func (app *capyxApp) genDefsJS(b *strings.Builder) error {
	b.WriteString("var PROVIDERS = {};\n")

	// Providers (capability implementations).
	for _, p := range app.providers {
		b.WriteString("PROVIDERS[" + jsStr(p.name) + "] = {")
		for i, m := range p.methods {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(genMethodLiteral(m))
		}
		b.WriteString("};\n")
	}

	// Orchestrator (shared data + boot + wiring), exposed as APP.
	if app.orch != nil {
		b.WriteString("var APP = CAPYX.makeHandler({state:{")
		for i, s := range app.orch.data {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(s.name + ":" + s.expr)
		}
		b.WriteString("},methods:{")
		methods := app.orch.methods
		if app.orch.boot != nil {
			methods = append(methods, *app.orch.boot)
		}
		for i, m := range methods {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(genMethodLiteral(m))
		}
		b.WriteString("}});\n")
		b.WriteString("PROVIDERS[" + jsStr(app.orch.name) + "] = APP;\n")
	}

	// Handler factories.
	for _, h := range app.handlers {
		b.WriteString(app.genHandlerFactory(h))
	}

	// Component render functions.
	for _, c := range app.components {
		js, err := app.genComponentFn(c)
		if err != nil {
			return err
		}
		b.WriteString(js)
	}
	return nil
}

func (app *capyxApp) genHandlerFactory(h capyxHandler) string {
	var b strings.Builder
	b.WriteString("function H_" + jsName(h.name) + "(){\nreturn CAPYX.makeHandler({state:{")
	for i, s := range h.state {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(s.name + ":" + s.expr)
	}
	b.WriteString("},methods:{")
	for i, m := range h.methods {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(genMethodLiteral(m))
	}
	b.WriteString("},ports:{")
	for i, p := range h.ports {
		if i > 0 {
			b.WriteString(",")
		}
		provider := app.capImpl[p.cap]
		b.WriteString(p.name + ":PROVIDERS[" + jsStr(provider) + "]")
	}
	b.WriteString("}});\n}\n")
	return b.String()
}

func genMethodLiteral(m capyxMethod) string {
	prefix := "function"
	if m.async {
		prefix = "async function"
	}
	return m.name + ":" + prefix + "(" + m.args + "){\n" + m.body + "}"
}

func (app *capyxApp) genComponentFn(c capyxComponent) (string, error) {
	ctx := genCtx{locals: map[string]bool{}, loopVar: ""}
	appends, err := genAppends(c.nodes, "__r", ctx)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("function C_" + jsName(c.name) + "(H){\n")
	b.WriteString("return CAPYX.el(\"div\",function(__r){__r.className=" +
		jsStr(c.scopeClass+" capyx-comp") + ";" + appends + "});\n}\n")
	return b.String(), nil
}

// ── CSS + style extraction ─────────────────────────────────────────────────

var styleRE = regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)

func extractStyle(full string) (css, tpl string) {
	var styles []string
	tpl = styleRE.ReplaceAllStringFunc(full, func(m string) string {
		sub := styleRE.FindStringSubmatch(m)
		if len(sub) > 1 {
			styles = append(styles, sub[1])
		}
		return ""
	})
	return strings.Join(styles, "\n"), tpl
}

var themeVarRE = regexp.MustCompile(`\{\{\s*theme\.([A-Za-z0-9_-]+)\s*\}\}`)

func themeVars(css string) string {
	return themeVarRE.ReplaceAllString(css, "var(--$1)")
}

// scopeCSS qualifies every selector with the component's scope class so styles
// can't leak, and resolves {{ theme.* }} to CSS custom properties.
func scopeCSS(css, scope string, theme map[string]string) string {
	css = themeVars(css)
	var out strings.Builder
	i := 0
	for i < len(css) {
		br := strings.IndexByte(css[i:], '{')
		if br < 0 {
			out.WriteString(css[i:])
			break
		}
		sel := strings.TrimSpace(css[i : i+br])
		rest := css[i+br+1:]
		end := strings.IndexByte(rest, '}')
		if end < 0 {
			end = len(rest)
		}
		body := strings.TrimSpace(rest[:end])
		if sel != "" {
			parts := strings.Split(sel, ",")
			for j, p := range parts {
				p = strings.TrimSpace(p)
				switch {
				case p == "" || strings.HasPrefix(p, "@"):
				case strings.Contains(p, ":global("):
					p = stripGlobal(p)
				default:
					p = "." + scope + " " + p
				}
				parts[j] = p
			}
			out.WriteString(strings.Join(parts, ", "))
			out.WriteString(" { " + body + " }\n")
		}
		i = i + br + 1 + end + 1
	}
	return out.String()
}

var globalRE = regexp.MustCompile(`:global\(([^)]*)\)`)

func stripGlobal(sel string) string {
	return strings.TrimSpace(globalRE.ReplaceAllString(sel, "$1"))
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// jsName converts an identifier (which may contain dashes) into a valid JS
// identifier fragment.
func jsName(s string) string {
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}
