package infra

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// capyx_template.go — parses a .capyx component template (HTML plus three-brace
// reactive bindings) into a node tree and compiles it to JavaScript that builds
// DOM through the CAPYX runtime helpers. Every {{ expr }}, class:x={…}, attr={…}
// becomes its own effect (surgical updates); {#for}/{#if}/{#match} become
// dynamic regions whose control effect tracks only the loop/condition.

type tplKind int

const (
	tnText tplKind = iota
	tnMustache
	tnElement
	tnFor
	tnIf
	tnMatch
)

type tplAttr struct {
	kind  string // static | dyn | class | style | on | model
	name  string
	value string
}

type tplNode struct {
	kind  tplKind
	text  string
	tag   string
	attrs []tplAttr
	kids  []tplNode

	forVar   string
	forList  string
	forEmpty []tplNode

	ifBranches []tplBranch

	matchExpr    string
	matchCases   []tplCase
	matchDefault []tplNode
}

type tplBranch struct {
	cond string // "" means else
	body []tplNode
}

type tplCase struct {
	val  string
	body []tplNode
}

// ── Parser ─────────────────────────────────────────────────────────────────

type tplParser struct {
	s string
	i int
}

func parseTemplate(src string) ([]tplNode, error) {
	p := &tplParser{s: src}
	nodes, marker, err := p.parseNodes()
	if err != nil {
		return nil, err
	}
	if marker != "" {
		return nil, fmt.Errorf("unexpected %q in template", marker)
	}
	return nodes, nil
}

// parseNodes consumes nodes until it hits a control marker that belongs to an
// enclosing construct (an else/elif/case/default, a closer like /for, or an
// element close "</"). It returns that marker (already consumed) for the caller.
func (p *tplParser) parseNodes() ([]tplNode, string, error) {
	var nodes []tplNode
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c == '<' {
			if p.i+1 < len(p.s) && p.s[p.i+1] == '/' {
				p.i += 2
				return nodes, "</", nil
			}
			node, err := p.parseElement()
			if err != nil {
				return nil, "", err
			}
			nodes = append(nodes, node)
			continue
		}
		if c == '{' && p.i+1 < len(p.s) {
			next := p.s[p.i+1]
			if next == '{' {
				expr := p.readUntil("}}")
				nodes = append(nodes, tplNode{kind: tnMustache, text: strings.TrimSpace(expr)})
				continue
			}
			if next == '#' || next == '/' {
				marker := p.readMarker()
				switch {
				case strings.HasPrefix(marker, "#for "):
					node, err := p.parseFor(marker)
					if err != nil {
						return nil, "", err
					}
					nodes = append(nodes, node)
				case strings.HasPrefix(marker, "#if "):
					node, err := p.parseIf(marker)
					if err != nil {
						return nil, "", err
					}
					nodes = append(nodes, node)
				case strings.HasPrefix(marker, "#match "):
					node, err := p.parseMatch(marker)
					if err != nil {
						return nil, "", err
					}
					nodes = append(nodes, node)
				default:
					// belongs to an enclosing construct
					return nodes, marker, nil
				}
				continue
			}
		}
		// plain text up to next '<' or '{'
		start := p.i
		for p.i < len(p.s) && p.s[p.i] != '<' && p.s[p.i] != '{' {
			p.i++
		}
		text := p.s[start:p.i]
		if strings.TrimSpace(text) != "" {
			nodes = append(nodes, tplNode{kind: tnText, text: collapseWS(text)})
		}
	}
	return nodes, "", nil
}

func (p *tplParser) readUntil(delim string) string {
	p.i += 2 // skip opener ({{ or {)
	idx := strings.Index(p.s[p.i:], delim)
	if idx < 0 {
		s := p.s[p.i:]
		p.i = len(p.s)
		return s
	}
	s := p.s[p.i : p.i+idx]
	p.i += idx + len(delim)
	return s
}

// readMarker reads `{#...}` or `{/...}`, returning the inside prefixed with #/ .
func (p *tplParser) readMarker() string {
	// p.s[p.i]=='{'
	sigil := p.s[p.i+1] // # or /
	p.i += 2
	idx := strings.IndexByte(p.s[p.i:], '}')
	var body string
	if idx < 0 {
		body = p.s[p.i:]
		p.i = len(p.s)
	} else {
		body = p.s[p.i : p.i+idx]
		p.i += idx + 1
	}
	return string(sigil) + strings.TrimSpace(body)
}

func (p *tplParser) parseElement() (tplNode, error) {
	p.i++ // skip '<'
	// tag name
	start := p.i
	for p.i < len(p.s) && isTagChar(p.s[p.i]) {
		p.i++
	}
	tag := p.s[start:p.i]
	node := tplNode{kind: tnElement, tag: tag}
	// attributes until > or />
	for p.i < len(p.s) {
		p.skipSpace()
		if p.i >= len(p.s) {
			break
		}
		if p.s[p.i] == '/' && p.i+1 < len(p.s) && p.s[p.i+1] == '>' {
			p.i += 2
			return node, nil // self-closing
		}
		if p.s[p.i] == '>' {
			p.i++
			break
		}
		attr, err := p.parseAttr()
		if err != nil {
			return node, err
		}
		node.attrs = append(node.attrs, attr)
	}
	// children
	kids, marker, err := p.parseNodes()
	if err != nil {
		return node, err
	}
	node.kids = kids
	if marker != "</" {
		return node, fmt.Errorf("<%s> not closed (got %q)", tag, marker)
	}
	// closing tag name + >
	p.skipSpace()
	cstart := p.i
	for p.i < len(p.s) && isTagChar(p.s[p.i]) {
		p.i++
	}
	closeName := p.s[cstart:p.i]
	if closeName != tag {
		return node, fmt.Errorf("</%s> does not match <%s>", closeName, tag)
	}
	p.skipSpace()
	if p.i < len(p.s) && p.s[p.i] == '>' {
		p.i++
	}
	return node, nil
}

func (p *tplParser) parseAttr() (tplAttr, error) {
	start := p.i
	for p.i < len(p.s) && isAttrNameChar(p.s[p.i]) {
		p.i++
	}
	name := p.s[start:p.i]
	if name == "" {
		// stray char; skip to avoid infinite loop
		p.i++
		return tplAttr{kind: "static", name: "", value: ""}, nil
	}
	p.skipSpace()
	if p.i >= len(p.s) || p.s[p.i] != '=' {
		// boolean attribute
		return classifyAttr(name, ""), nil
	}
	p.i++ // '='
	p.skipSpace()
	if p.i >= len(p.s) {
		return classifyAttr(name, ""), nil
	}
	if p.s[p.i] == '"' || p.s[p.i] == '\'' {
		q := p.s[p.i]
		p.i++
		vs := p.i
		for p.i < len(p.s) && p.s[p.i] != q {
			p.i++
		}
		val := p.s[vs:p.i]
		if p.i < len(p.s) {
			p.i++
		}
		a := classifyAttr(name, val)
		if a.kind == "dyn" || a.kind == "class" || a.kind == "style" || a.kind == "on" || a.kind == "model" {
			// quoted value for a dynamic attr name → treat literally
			a.kind = "static"
		}
		return a, nil
	}
	if p.s[p.i] == '{' {
		val := p.readBraceExpr()
		return classifyAttr(name, val), nil
	}
	// unquoted value
	vs := p.i
	for p.i < len(p.s) && !isSpace(p.s[p.i]) && p.s[p.i] != '>' && p.s[p.i] != '/' {
		p.i++
	}
	return classifyAttr(name, p.s[vs:p.i]), nil
}

// readBraceExpr reads a `{ ... }` value honouring nested braces.
func (p *tplParser) readBraceExpr() string {
	p.i++ // '{'
	depth := 1
	start := p.i
	for p.i < len(p.s) && depth > 0 {
		switch p.s[p.i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				s := p.s[start:p.i]
				p.i++
				return strings.TrimSpace(s)
			}
		}
		p.i++
	}
	return strings.TrimSpace(p.s[start:p.i])
}

func classifyAttr(name, val string) tplAttr {
	switch {
	case strings.HasPrefix(name, "class:"):
		return tplAttr{kind: "class", name: name[len("class:"):], value: val}
	case strings.HasPrefix(name, "style:"):
		return tplAttr{kind: "style", name: name[len("style:"):], value: val}
	case strings.HasPrefix(name, "on:"):
		return tplAttr{kind: "on", name: name[len("on:"):], value: val}
	case strings.HasPrefix(name, "bind:"):
		return tplAttr{kind: "model", name: name[len("bind:"):], value: val}
	case name == "model":
		return tplAttr{kind: "model", name: "value", value: val}
	default:
		return tplAttr{kind: "dyn", name: name, value: val}
	}
}

func (p *tplParser) parseFor(marker string) (tplNode, error) {
	header := strings.TrimSpace(marker[len("#for "):])
	idx := strings.Index(header, " in ")
	if idx < 0 {
		return tplNode{}, fmt.Errorf("{#for} needs `item in list`: %q", header)
	}
	node := tplNode{
		kind:    tnFor,
		forVar:  strings.TrimSpace(header[:idx]),
		forList: strings.TrimSpace(header[idx+4:]),
	}
	body, marker2, err := p.parseNodes()
	if err != nil {
		return node, err
	}
	node.kids = body
	if marker2 == "#else" {
		empty, marker3, err := p.parseNodes()
		if err != nil {
			return node, err
		}
		node.forEmpty = empty
		marker2 = marker3
	}
	if marker2 != "/for" {
		return node, fmt.Errorf("{#for} not closed with {/for} (got %q)", marker2)
	}
	return node, nil
}

func (p *tplParser) parseIf(marker string) (tplNode, error) {
	cond := strings.TrimSpace(marker[len("#if "):])
	node := tplNode{kind: tnIf}
	for {
		body, m, err := p.parseNodes()
		if err != nil {
			return node, err
		}
		node.ifBranches = append(node.ifBranches, tplBranch{cond: cond, body: body})
		switch {
		case strings.HasPrefix(m, "#elif "):
			cond = strings.TrimSpace(m[len("#elif "):])
		case m == "#else":
			cond = "" // next branch is else
		case m == "/if":
			return node, nil
		default:
			return node, fmt.Errorf("{#if} not closed with {/if} (got %q)", m)
		}
	}
}

func (p *tplParser) parseMatch(marker string) (tplNode, error) {
	node := tplNode{kind: tnMatch, matchExpr: strings.TrimSpace(marker[len("#match "):])}
	// content before first case is ignored
	_, m, err := p.parseNodes()
	if err != nil {
		return node, err
	}
	for {
		switch {
		case strings.HasPrefix(m, "#case "):
			val := strings.TrimSpace(m[len("#case "):])
			body, m2, err := p.parseNodes()
			if err != nil {
				return node, err
			}
			node.matchCases = append(node.matchCases, tplCase{val: val, body: body})
			m = m2
		case m == "#default":
			body, m2, err := p.parseNodes()
			if err != nil {
				return node, err
			}
			node.matchDefault = body
			m = m2
		case m == "/match":
			return node, nil
		default:
			return node, fmt.Errorf("{#match} not closed with {/match} (got %q)", m)
		}
	}
}

// ── Codegen ────────────────────────────────────────────────────────────────

type genCtx struct {
	locals  map[string]bool
	loopVar string
}

func (c genCtx) child() genCtx {
	nl := make(map[string]bool, len(c.locals))
	for k := range c.locals {
		nl[k] = true
	}
	return genCtx{locals: nl, loopVar: c.loopVar}
}

func genNodes(nodes []tplNode, ctx genCtx) (string, error) {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		e, err := genNode(n, ctx)
		if err != nil {
			return "", err
		}
		parts = append(parts, e)
	}
	return strings.Join(parts, ", "), nil
}

// genGroup builds a single node expression from a body that may have multiple
// roots (wraps in a display:contents div when needed).
func genGroup(nodes []tplNode, ctx genCtx) (string, error) {
	if len(nodes) == 1 {
		return genNode(nodes[0], ctx)
	}
	if len(nodes) == 0 {
		return "null", nil
	}
	inner, err := genAppends(nodes, "__g", ctx)
	if err != nil {
		return "", err
	}
	return "CAPYX.el(\"div\", function(__g){__g.className=\"contents\";" + inner + "})", nil
}

func genAppends(nodes []tplNode, parent string, ctx genCtx) (string, error) {
	var b strings.Builder
	for _, n := range nodes {
		e, err := genNode(n, ctx)
		if err != nil {
			return "", err
		}
		b.WriteString(parent + ".appendChild(" + e + ");")
	}
	return b.String(), nil
}

func genNode(n tplNode, ctx genCtx) (string, error) {
	switch n.kind {
	case tnText:
		return "CAPYX.staticText(" + jsStr(n.text) + ")", nil
	case tnMustache:
		return "CAPYX.text(function(){return " + rewriteViewExpr(n.text, ctx.locals) + ";})", nil
	case tnElement:
		return genElement(n, ctx)
	case tnFor:
		return genFor(n, ctx)
	case tnIf:
		return genIf(n, ctx)
	case tnMatch:
		return genMatch(n, ctx)
	}
	return "null", nil
}

func genElement(n tplNode, ctx genCtx) (string, error) {
	var b strings.Builder
	for _, a := range n.attrs {
		if a.name == "" {
			continue
		}
		switch a.kind {
		case "static":
			b.WriteString("__n.setAttribute(" + jsStr(a.name) + "," + jsStr(a.value) + ");")
		case "dyn":
			getter := "function(){return " + rewriteViewExpr(a.value, ctx.locals) + ";}"
			if a.name == "value" || a.name == "checked" {
				b.WriteString("CAPYX.prop(__n," + jsStr(a.name) + "," + getter + ");")
			} else {
				b.WriteString("CAPYX.attr(__n," + jsStr(a.name) + "," + getter + ");")
			}
		case "class":
			b.WriteString("CAPYX.cls(__n," + jsStr(a.name) + ",function(){return " + rewriteViewExpr(a.value, ctx.locals) + ";});")
		case "style":
			b.WriteString("CAPYX.style(__n," + jsStr(a.name) + ",function(){return " + rewriteViewExpr(a.value, ctx.locals) + ";});")
		case "on":
			b.WriteString("CAPYX.on(__n," + jsStr(a.name) + "," + genEventHandler(a.value, ctx) + ");")
		case "model":
			path := rewriteViewExpr(a.value, ctx.locals)
			b.WriteString("CAPYX.model(__n,function(){return " + path + ";},function($v){" + path + "=$v;});")
		}
	}
	appends, err := genAppends(n.kids, "__n", ctx)
	if err != nil {
		return "", err
	}
	b.WriteString(appends)
	return "CAPYX.el(" + jsStr(n.tag) + ",function(__n){" + b.String() + "})", nil
}

var bareIdentRE = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

func genEventHandler(value string, ctx genCtx) string {
	v := strings.TrimSpace(value)
	if bareIdentRE.MatchString(v) {
		arg := ""
		if ctx.loopVar != "" {
			arg = ctx.loopVar
		}
		return "function($e){return H." + v + "(" + arg + ");}"
	}
	return "function($e){" + rewriteViewExpr(v, ctx.locals) + ";}"
}

func genFor(n tplNode, ctx genCtx) (string, error) {
	child := ctx.child()
	child.locals[n.forVar] = true
	child.locals["$i"] = true
	child.loopVar = n.forVar
	body, err := genGroup(n.kids, child)
	if err != nil {
		return "", err
	}
	list := "function(){return " + rewriteViewExpr(n.forList, ctx.locals) + ";}"
	keyer := "function(" + n.forVar + "){return (" + n.forVar + "&&" + n.forVar + ".id!=null)?" + n.forVar + ".id:" + n.forVar + ";}"
	builder := "function(" + n.forVar + ",$i){return " + body + ";}"
	empty := "null"
	if len(n.forEmpty) > 0 {
		eg, err := genGroup(n.forEmpty, ctx)
		if err != nil {
			return "", err
		}
		empty = "function(){return " + eg + ";}"
	}
	return "CAPYX.each(" + list + "," + keyer + "," + builder + "," + empty + ")", nil
}

func genIf(n tplNode, ctx genCtx) (string, error) {
	var keyExpr strings.Builder
	var builders []string
	idx := 0
	for _, br := range n.ifBranches {
		g, err := genGroup(br.body, ctx)
		if err != nil {
			return "", err
		}
		builders = append(builders, "function(){return "+g+";}")
		if br.cond == "" {
			// else: terminal
			break
		}
		keyExpr.WriteString("(" + rewriteViewExpr(br.cond, ctx.locals) + ")?" + fmt.Sprint(idx) + ":")
		idx++
	}
	// If no else branch, trailing index returns null builder.
	hasElse := len(n.ifBranches) > 0 && n.ifBranches[len(n.ifBranches)-1].cond == ""
	if hasElse {
		keyExpr.WriteString(fmt.Sprint(idx))
	} else {
		builders = append(builders, "function(){return null;}")
		keyExpr.WriteString(fmt.Sprint(idx))
	}
	return "CAPYX.when(function(){return " + keyExpr.String() + ";},[" + strings.Join(builders, ",") + "])", nil
}

func genMatch(n tplNode, ctx genCtx) (string, error) {
	var keyExpr strings.Builder
	keyExpr.WriteString("var __v=" + rewriteViewExpr(n.matchExpr, ctx.locals) + ";return ")
	var builders []string
	for i, c := range n.matchCases {
		g, err := genGroup(c.body, ctx)
		if err != nil {
			return "", err
		}
		builders = append(builders, "function(){return "+g+";}")
		keyExpr.WriteString("__v===" + c.val + "?" + fmt.Sprint(i) + ":")
	}
	defIdx := len(n.matchCases)
	if len(n.matchDefault) > 0 {
		g, err := genGroup(n.matchDefault, ctx)
		if err != nil {
			return "", err
		}
		builders = append(builders, "function(){return "+g+";}")
	} else {
		builders = append(builders, "function(){return null;}")
	}
	keyExpr.WriteString(fmt.Sprint(defIdx))
	return "CAPYX.when(function(){" + keyExpr.String() + ";},[" + strings.Join(builders, ",") + "])", nil
}

// ── small helpers ──────────────────────────────────────────────────────────

func jsStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

var wsRE = regexp.MustCompile(`\s+`)

func collapseWS(s string) string {
	return wsRE.ReplaceAllString(s, " ")
}

func isTagChar(c byte) bool {
	return c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isAttrNameChar(c byte) bool {
	return isTagChar(c) || c == ':'
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func (p *tplParser) skipSpace() {
	for p.i < len(p.s) && isSpace(p.s[p.i]) {
		p.i++
	}
}
