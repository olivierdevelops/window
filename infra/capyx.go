package infra

import (
	"fmt"
	"regexp"
	"strings"
)

// capyx.go — the .capyx compiler entry point. A .capyx file is a single-file
// VHCO app: dumb `component` views, `handler` units (state + on-event mutations)
// that only mutate their own state and call injected ports, `capability`
// protocols with `provide` implementations (the only layer that touches the
// outside world), an optional `orchestrator` that owns shared data and wiring,
// and `mount` lines that place components with their handler. It compiles to a
// window app: window.yaml + static/index.html with an embedded signals runtime.

type capyxComponent struct {
	name       string
	scopeClass string
	css        string
	nodes      []tplNode
}

type capyxState struct {
	name string
	expr string
}

type capyxMethod struct {
	name  string
	args  string
	body  string
	async bool
}

type capyxPort struct {
	cap  string
	name string
}

type capyxHandler struct {
	name      string
	forComp   string
	state     []capyxState
	ports     []capyxPort
	methods   []capyxMethod
	stateSet  map[string]bool
	methodSet map[string]bool
}

type capyxProvider struct {
	cap     string
	name    string
	methods []capyxMethod
}

type capyxMount struct {
	component string
	handler   string
	id        string
}

type capyxOrch struct {
	name    string
	data    []capyxState
	boot    *capyxMethod
	methods []capyxMethod
}

type capyxApp struct {
	title      string
	width      int
	height     int
	theme      map[string]string
	components []capyxComponent
	handlers   []capyxHandler
	providers  []capyxProvider
	capImpl    map[string]string
	orch       *capyxOrch
	mounts     []capyxMount
	runtime    string
}

// CompileCapyx transpiles a .capyx source into the files of a runnable window
// app: window.yaml plus static/index.html (with the signals runtime inlined).
// runtimeJS is the contents of capyx_runtime.js (passed in so infra stays free
// of embedded-asset knowledge).
func CompileCapyx(src, runtimeJS string) (map[string]string, error) {
	app, err := parseCapyx(src)
	if err != nil {
		return nil, err
	}
	app.runtime = runtimeJS
	if err := app.validate(); err != nil {
		return nil, err
	}
	index, err := app.genIndexHTML()
	if err != nil {
		return nil, err
	}
	files := map[string]string{
		"window.yaml":       app.genWindowYAML(),
		"static/index.html": index,
	}
	return files, nil
}

// ── Top-level parsing ──────────────────────────────────────────────────────

var appHeaderRE = regexp.MustCompile(`^app\s+"([^"]*)"\s+(\d+)\s*[xX]\s*(\d+)\s*$`)

func parseCapyx(src string) (*capyxApp, error) {
	app := &capyxApp{
		title:   "Capyx App",
		width:   480,
		height:  600,
		theme:   map[string]string{},
		capImpl: map[string]string{},
	}
	lines := strings.Split(src, "\n")
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}
		word := firstWord(trimmed)
		switch word {
		case "app":
			if m := appHeaderRE.FindStringSubmatch(trimmed); m != nil {
				app.title = m[1]
				app.width = atoi(m[2])
				app.height = atoi(m[3])
			} else {
				app.title = stripQuotes(strings.TrimSpace(trimmed[3:]))
			}
			i++
		case "theme":
			parseThemeLine(trimmed[len("theme"):], app.theme)
			i++
		case "import":
			i++ // single-file demos resolve imports inline; recorded as no-op
		case "mount":
			app.mounts = append(app.mounts, parseMountLine(trimmed))
			i++
		case "component", "handler", "capability", "provide", "orchestrator":
			body, next := collectBlock(lines, i+1)
			if err := app.parseBlock(trimmed, body); err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			i = next
		default:
			return nil, fmt.Errorf("line %d: unknown construct %q", i+1, word)
		}
	}
	return app, nil
}

// collectBlock returns the body lines of a block (from start) up to the first
// `end` at column 0, and the index of the line after it.
func collectBlock(lines []string, start int) ([]string, int) {
	var body []string
	i := start
	for i < len(lines) {
		if lines[i] == "end" || strings.TrimRight(lines[i], " \t") == "end" && !startsWithSpace(lines[i]) {
			return body, i + 1
		}
		body = append(body, lines[i])
		i++
	}
	return body, i
}

func startsWithSpace(s string) bool {
	return len(s) > 0 && (s[0] == ' ' || s[0] == '\t')
}

func (app *capyxApp) parseBlock(header string, body []string) error {
	word := firstWord(header)
	switch word {
	case "component":
		return app.parseComponent(header, body)
	case "handler":
		return app.parseHandler(header, body)
	case "capability":
		return nil // protocol only; validated structurally elsewhere
	case "provide":
		return app.parseProvide(header, body)
	case "orchestrator":
		return app.parseOrchestrator(header, body)
	}
	return nil
}

func (app *capyxApp) parseComponent(header string, body []string) error {
	name := strings.TrimSpace(header[len("component"):])
	name = firstWord(name)
	if name == "" {
		return fmt.Errorf("component is missing a name")
	}
	full := strings.Join(body, "\n")
	css, tpl := extractStyle(full)
	scope := "cz-" + sanitizeClass(name)
	nodes, err := parseTemplate(tpl)
	if err != nil {
		return fmt.Errorf("component %q: %w", name, err)
	}
	app.components = append(app.components, capyxComponent{
		name:       name,
		scopeClass: scope,
		css:        scopeCSS(css, scope, app.theme),
		nodes:      nodes,
	})
	return nil
}

var handlerHeaderRE = regexp.MustCompile(`^handler\s+([A-Za-z][\w-]*)(?:\s+for\s+([A-Za-z][\w-]*))?`)

func (app *capyxApp) parseHandler(header string, body []string) error {
	m := handlerHeaderRE.FindStringSubmatch(header)
	if m == nil {
		return fmt.Errorf("bad handler header: %q", header)
	}
	h := capyxHandler{
		name:      m[1],
		forComp:   m[2],
		stateSet:  map[string]bool{},
		methodSet: map[string]bool{},
	}
	// First pass: collect state + port names so method bodies can resolve them.
	parseHandlerDecls(body, &h)
	scope := handlerScope{state: h.stateSet, ports: portSet(h.ports)}
	// Second pass: compile methods.
	if err := parseHandlerMethods(body, &h, scope); err != nil {
		return fmt.Errorf("handler %q: %w", h.name, err)
	}
	app.handlers = append(app.handlers, h)
	return nil
}

var stateRE = regexp.MustCompile(`^state\s+([A-Za-z_]\w*)\s*(?::[^=]*)?=\s*(.*)$`)
var needsRE = regexp.MustCompile(`^needs\s+([A-Za-z][\w]*)\s+as\s+([A-Za-z_]\w*)`)

func parseHandlerDecls(body []string, h *capyxHandler) {
	for _, raw := range body {
		t := strings.TrimSpace(raw)
		if m := stateRE.FindStringSubmatch(t); m != nil {
			h.state = append(h.state, capyxState{name: m[1], expr: strings.TrimSpace(m[2])})
			h.stateSet[m[1]] = true
		} else if m := needsRE.FindStringSubmatch(t); m != nil {
			h.ports = append(h.ports, capyxPort{cap: m[1], name: m[2]})
		}
	}
}

var onHeaderRE = regexp.MustCompile(`^on\s+([A-Za-z_]\w*)\s*\(([^)]*)\)`)

func parseHandlerMethods(body []string, h *capyxHandler, scope handlerScope) error {
	i := 0
	for i < len(body) {
		t := strings.TrimSpace(body[i])
		if m := onHeaderRE.FindStringSubmatch(t); m != nil {
			method := capyxMethod{name: m[1], args: strings.TrimSpace(m[2])}
			inner, next := collectNested(body, i+1)
			js, async := translateStatements(inner, scope)
			method.body = js
			method.async = async
			h.methods = append(h.methods, method)
			h.methodSet[method.name] = true
			i = next
			continue
		}
		i++
	}
	return nil
}

// collectNested gathers an indented block body up to its matching `end`,
// counting nested if/for/while openers.
func collectNested(lines []string, start int) ([]string, int) {
	var body []string
	depth := 0
	i := start
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if t == "end" {
			if depth == 0 {
				return body, i + 1
			}
			depth--
			body = append(body, lines[i])
			i++
			continue
		}
		if isOpener(t) {
			depth++
		}
		body = append(body, lines[i])
		i++
	}
	return body, i
}

func isOpener(t string) bool {
	// `on name(...)` / `fn name(...)` open a block; a bare statement like
	// `on = !on` (assigning a state named `on`) must NOT be treated as one.
	if onHeaderRE.MatchString(t) || fnHeaderRE.MatchString(t) {
		return true
	}
	return strings.HasPrefix(t, "if ") || strings.HasPrefix(t, "for ") ||
		strings.HasPrefix(t, "while ")
}

var provideHeaderRE = regexp.MustCompile(`^provide\s+([A-Za-z][\w]*)\s+as\s+([A-Za-z][\w]*)`)
var fnHeaderRE = regexp.MustCompile(`^fn\s+([A-Za-z_]\w*)\s*\(([^)]*)\)`)

func (app *capyxApp) parseProvide(header string, body []string) error {
	m := provideHeaderRE.FindStringSubmatch(header)
	if m == nil {
		return fmt.Errorf("bad provide header: %q", header)
	}
	p := capyxProvider{cap: m[1], name: m[2]}
	scope := handlerScope{state: map[string]bool{}, ports: map[string]bool{}}
	i := 0
	for i < len(body) {
		t := strings.TrimSpace(body[i])
		if fm := fnHeaderRE.FindStringSubmatch(t); fm != nil {
			method := capyxMethod{name: fm[1], args: strings.TrimSpace(fm[2])}
			inner, next := collectNested(body, i+1)
			js, async := translateStatements(inner, scope)
			method.body = js
			method.async = async
			p.methods = append(p.methods, method)
			i = next
			continue
		}
		i++
	}
	app.providers = append(app.providers, p)
	app.capImpl[p.cap] = p.name
	return nil
}

func (app *capyxApp) parseOrchestrator(header string, body []string) error {
	name := strings.TrimSpace(header[len("orchestrator"):])
	o := &capyxOrch{name: firstWord(name)}
	scope := handlerScope{state: map[string]bool{}, ports: map[string]bool{}}
	// collect data names first
	i := 0
	for i < len(body) {
		t := strings.TrimSpace(body[i])
		if t == "data" {
			inner, next := collectNested(body, i+1)
			for _, dl := range inner {
				dt := strings.TrimSpace(dl)
				if m := dataRE.FindStringSubmatch(dt); m != nil {
					o.data = append(o.data, capyxState{name: m[1], expr: strings.TrimSpace(m[2])})
					scope.state[m[1]] = true
				}
			}
			i = next
			continue
		}
		i++
	}
	// pass 2: providers via `use`, boot + methods
	i = 0
	for i < len(body) {
		t := strings.TrimSpace(body[i])
		switch {
		case strings.HasPrefix(t, "use "):
			if m := useRE.FindStringSubmatch(t); m != nil {
				app.capImpl[m[1]] = m[2]
			}
			i++
		case strings.HasPrefix(t, "on "):
			if m := onHeaderRE.FindStringSubmatch(t); m != nil {
				method := capyxMethod{name: m[1], args: strings.TrimSpace(m[2])}
				inner, next := collectNested(body, i+1)
				js, async := translateStatements(inner, scope)
				method.body = js
				method.async = async
				if method.name == "boot" {
					o.boot = &method
				} else {
					o.methods = append(o.methods, method)
				}
				i = next
				continue
			}
			i++
		default:
			i++
		}
	}
	// An orchestrator is automatically injectable as a capability named after
	// itself, so a handler can `needs <orch> as app` to share its reactive data.
	app.capImpl[o.name] = o.name
	app.orch = o
	return nil
}

var dataRE = regexp.MustCompile(`^([A-Za-z_]\w*)\s*(?::[^=]*)?=\s*(.*)$`)
var useRE = regexp.MustCompile(`^use\s+([A-Za-z][\w]*)\s+as\s+([A-Za-z][\w]*)`)

func parseMountLine(line string) capyxMount {
	rest := strings.TrimSpace(line[len("mount"):])
	fields := strings.Fields(rest)
	m := capyxMount{}
	if len(fields) > 0 {
		m.component = fields[0]
	}
	for k := 1; k+1 < len(fields)+1 && k < len(fields); k++ {
		switch fields[k] {
		case "as":
			if k+1 < len(fields) {
				m.id = fields[k+1]
			}
		case "use":
			if k+1 < len(fields) {
				m.handler = fields[k+1]
			}
		}
	}
	if m.handler == "" {
		m.handler = m.component
	}
	if m.id == "" {
		m.id = m.component
	}
	return m
}

// ── Validation ─────────────────────────────────────────────────────────────

func (app *capyxApp) validate() error {
	if len(app.components) == 0 {
		return fmt.Errorf("a .capyx app needs at least one component")
	}
	comps := map[string]bool{}
	for _, c := range app.components {
		comps[c.name] = true
	}
	handlers := map[string]bool{}
	for _, h := range app.handlers {
		handlers[h.name] = true
	}
	if len(app.mounts) == 0 {
		// default: mount the last component with its default handler
		last := app.components[len(app.components)-1]
		app.mounts = append(app.mounts, capyxMount{component: last.name, handler: last.name, id: last.name})
	}
	for _, m := range app.mounts {
		if !comps[m.component] {
			return fmt.Errorf("mount references unknown component %q", m.component)
		}
		if !handlers[m.handler] && handlerUsed(app, m.component) {
			return fmt.Errorf("mount %q needs handler %q (declare `handler %s`)", m.component, m.handler, m.handler)
		}
	}
	// every declared port must have a provider
	for _, h := range app.handlers {
		for _, p := range h.ports {
			if _, ok := app.capImpl[p.cap]; !ok {
				return fmt.Errorf("handler %q needs capability %q but nothing provides it", h.name, p.cap)
			}
		}
	}
	return nil
}

// handlerUsed reports whether a component references handler state/events (so a
// missing handler is an error). A purely static component needs no handler.
func handlerUsed(app *capyxApp, comp string) bool {
	for _, c := range app.components {
		if c.name == comp {
			return templateUsesHandler(c.nodes)
		}
	}
	return false
}

func templateUsesHandler(nodes []tplNode) bool {
	for _, n := range nodes {
		switch n.kind {
		case tnMustache:
			return true
		case tnElement:
			for _, a := range n.attrs {
				if a.kind != "static" {
					return true
				}
			}
			if templateUsesHandler(n.kids) {
				return true
			}
		case tnFor, tnIf, tnMatch:
			return true
		}
	}
	return false
}

// ── helpers ────────────────────────────────────────────────────────────────

func portSet(ports []capyxPort) map[string]bool {
	s := map[string]bool{}
	for _, p := range ports {
		s[p.name] = true
	}
	return s
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return s[:i]
		}
	}
	return s
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func sanitizeClass(s string) string {
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

func parseThemeLine(s string, into map[string]string) {
	for _, f := range tokenizeAttrs(s) {
		eq := strings.IndexByte(f, '=')
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(f[:eq])
		v := stripQuotes(strings.TrimSpace(f[eq+1:]))
		if k != "" {
			into[k] = v
		}
	}
}

// tokenizeAttrs splits `a="b c" d=e` into ["a=\"b c\"","d=e"].
func tokenizeAttrs(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := byte(0)
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote != 0 {
			cur.WriteByte(c)
			if c == inQuote {
				inQuote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			inQuote = c
			cur.WriteByte(c)
			continue
		}
		if c == ' ' || c == '\t' {
			flush()
			continue
		}
		cur.WriteByte(c)
	}
	flush()
	return out
}
