package infra

import (
	"strings"
)

// capyx_expr.go — expression and statement translation for the .capyx language.
//
// The .capyx handler body is a tiny Python-flavoured language (if/elif/else,
// for-in, while, return, log, let, do, and bare expression statements). It
// compiles to JavaScript. The key transformation is identifier rewriting: a
// bare reference to a handler's state field (or injected port) is rewritten so
// it flows through the reactive proxy — e.g. inside a handler `count = count+1`
// becomes `this.count = this.count + 1`, and in a view `{{ count }}` becomes
// `H.count`. Object-literal keys and the property side of `a.b` are never
// rewritten, so reactivity is surgical and authoring stays clean.

// rewriteIdents walks JS-ish source and, for every identifier in *value
// position*, asks resolve whether to replace it. String/template/comment spans
// are skipped. The property side of `a.b`, and object-literal keys (`{ k: … }`),
// are left untouched.
func rewriteIdents(code string, resolve func(name string) (string, bool)) string {
	var b strings.Builder
	r := []rune(code)
	n := len(r)
	// Identifiers that are arrow-function parameters (e.g. x in `x => x.id`) are
	// locals of an inline closure and must never be rewritten to this./H. .
	skip := arrowParams(r)
	prevSig := rune(0) // previous significant (non-space) char
	for i := 0; i < n; {
		c := r[i]
		// String literals.
		if c == '"' || c == '\'' || c == '`' {
			j := i + 1
			for j < n {
				if r[j] == '\\' {
					j += 2
					continue
				}
				if r[j] == c {
					j++
					break
				}
				j++
			}
			b.WriteString(string(r[i:j]))
			prevSig = c
			i = j
			continue
		}
		// Comments.
		if c == '/' && i+1 < n && r[i+1] == '/' {
			j := i
			for j < n && r[j] != '\n' {
				j++
			}
			b.WriteString(string(r[i:j]))
			i = j
			continue
		}
		if c == '/' && i+1 < n && r[i+1] == '*' {
			j := i + 2
			for j < n && !(r[j] == '*' && j+1 < n && r[j+1] == '/') {
				j++
			}
			if j < n {
				j += 2
			}
			b.WriteString(string(r[i:j]))
			i = j
			continue
		}
		// Identifiers.
		if isIdentStart(c) {
			j := i + 1
			for j < n && isIdentPart(r[j]) {
				j++
			}
			name := string(r[i:j])
			// Look ahead/behind to classify.
			afterDot := prevSig == '.'
			// next significant char
			k := j
			for k < n && (r[k] == ' ' || r[k] == '\t') {
				k++
			}
			var nextSig rune
			if k < n {
				nextSig = r[k]
			}
			isObjKey := nextSig == ':' && (prevSig == '{' || prevSig == ',')
			if !afterDot && !isObjKey && !skip[name] {
				if repl, ok := resolve(name); ok {
					b.WriteString(repl)
					prevSig = 'x'
					i = j
					continue
				}
			}
			b.WriteString(name)
			prevSig = 'x'
			i = j
			continue
		}
		b.WriteRune(c)
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			prevSig = c
		}
		i++
	}
	return b.String()
}

// arrowParams scans an expression for arrow-function parameter names so they
// can be excluded from identifier rewriting. It recognises both `x => …` and
// `(a, b) => …` forms.
func arrowParams(r []rune) map[string]bool {
	set := map[string]bool{}
	n := len(r)
	for i := 0; i+1 < n; i++ {
		if r[i] != '=' || r[i+1] != '>' {
			continue
		}
		j := i - 1
		for j >= 0 && (r[j] == ' ' || r[j] == '\t') {
			j--
		}
		if j < 0 {
			continue
		}
		if r[j] == ')' {
			depth := 1
			k := j - 1
			for k >= 0 {
				if r[k] == ')' {
					depth++
				} else if r[k] == '(' {
					depth--
					if depth == 0 {
						break
					}
				}
				k--
			}
			for _, name := range splitArrowParams(r[k+1 : j]) {
				set[name] = true
			}
		} else if isIdentPart(r[j]) {
			end := j + 1
			for j >= 0 && isIdentPart(r[j]) {
				j--
			}
			name := string(r[j+1 : end])
			if name != "" && isIdentStart(rune(name[0])) {
				set[name] = true
			}
		}
	}
	return set
}

// splitArrowParams extracts the leading identifier of each comma-separated
// parameter (ignoring defaults/destructuring detail).
func splitArrowParams(r []rune) []string {
	var out []string
	for _, part := range strings.Split(string(r), ",") {
		part = strings.TrimSpace(part)
		i := 0
		for i < len(part) && (part[i] == ' ' || part[i] == '.' || part[i] == '{' || part[i] == '[') {
			i++
		}
		start := i
		for i < len(part) && isIdentPart(rune(part[i])) {
			i++
		}
		if i > start {
			name := part[start:i]
			if isIdentStart(rune(name[0])) {
				out = append(out, name)
			}
		}
	}
	return out
}

func isIdentStart(c rune) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c rune) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// viewGlobals are identifiers a view may reference without them resolving to a
// handler field (browser/runtime globals and literals).
var viewGlobals = map[string]bool{
	"true": true, "false": true, "null": true, "undefined": true,
	"Math": true, "JSON": true, "Object": true, "Array": true,
	"String": true, "Number": true, "Boolean": true, "Date": true,
	"console": true, "window": true, "document": true, "parseInt": true,
	"parseFloat": true, "isNaN": true, "NaN": true, "Infinity": true,
}

// rewriteViewExpr prefixes handler-field references with `H.`, leaving loop
// variables (locals), globals and literals alone.
func rewriteViewExpr(expr string, locals map[string]bool) string {
	return rewriteIdents(expr, func(name string) (string, bool) {
		if locals[name] || viewGlobals[name] {
			return "", false
		}
		return "H." + name, true
	})
}

// handlerScope captures the names that should be rewritten inside a handler
// method body: state fields map to `this.X`, injected ports to `this.$ports.X`.
type handlerScope struct {
	state map[string]bool
	ports map[string]bool
}

func (s handlerScope) resolve(name string) (string, bool) {
	if s.state[name] {
		return "this." + name, true
	}
	if s.ports[name] {
		return "this.$ports." + name, true
	}
	return "", false
}

// translateStatements compiles indented handler-body lines into a JS block.
// Returns the body source and whether it must be async (contains `await`).
func translateStatements(lines []string, scope handlerScope) (string, bool) {
	var b strings.Builder
	async := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			b.WriteString("// " + strings.TrimPrefix(line, "#") + "\n")
			continue
		}
		if strings.Contains(line, "await ") || strings.HasPrefix(line, "await") {
			async = true
		}
		js := translateStatement(line, scope)
		b.WriteString(js + "\n")
	}
	return b.String(), async
}

func translateStatement(line string, scope handlerScope) string {
	rw := func(s string) string { return rewriteIdents(s, scope.resolve) }
	switch {
	case line == "end":
		return "}"
	case line == "else":
		return "} else {"
	case strings.HasPrefix(line, "elif "):
		return "} else if (" + rw(strings.TrimSpace(line[5:])) + ") {"
	case strings.HasPrefix(line, "if "):
		return "if (" + rw(strings.TrimSpace(line[3:])) + ") {"
	case strings.HasPrefix(line, "while "):
		return "while (" + rw(strings.TrimSpace(line[6:])) + ") {"
	case strings.HasPrefix(line, "for "):
		rest := strings.TrimSpace(line[4:])
		if idx := strings.Index(rest, " in "); idx >= 0 {
			v := strings.TrimSpace(rest[:idx])
			it := strings.TrimSpace(rest[idx+4:])
			return "for (const " + v + " of " + rw(it) + ") {"
		}
		return "for (" + rw(rest) + ") {"
	case strings.HasPrefix(line, "return "):
		return "return " + rw(strings.TrimSpace(line[7:])) + ";"
	case line == "return":
		return "return;"
	case strings.HasPrefix(line, "log "):
		return "console.log(" + rw(strings.TrimSpace(line[4:])) + ");"
	case strings.HasPrefix(line, "let "):
		return "let " + rw(strings.TrimSpace(line[4:])) + ";"
	case strings.HasPrefix(line, "do "):
		return rw(strings.TrimSpace(line[3:])) + ";"
	default:
		return rw(line) + ";"
	}
}
