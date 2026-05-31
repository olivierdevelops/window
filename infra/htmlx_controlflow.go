package infra

import (
	"fmt"
	"regexp"
	"strings"
)

// ExpandControlFlow expands HTML-native, compile-time control flow in a .htmlx
// source using the same three-brace directive syntax as .capyx: {#for}, {#if}
// /{#elif}/{#else}, and {#match}/{#case}/{#default}. Each construct is evaluated
// during transpilation (there is no runtime data model) and composes with the
// others — an {#if} inside a {#for} sees the loop variable.
//
//	{#for label in Home, About, Contact}
//	  <li><a href="#">"{{ label }}"</a></li>
//	{/for}
//
//	{#if role == admin}
//	  …
//	{#elif role in editor, owner}
//	  …
//	{#else}
//	  …
//	{/if}
//
//	{#match state}
//	  {#case ok}
//	    …
//	  {#default}
//	    …
//	{/match}
//
// Loop variables are referenced as {{ name }} and substituted as raw text
// (htmlx still escapes quoted text nodes downstream). Conditions compare with
// `==` (equality) or `in` (comma-list membership). Each directive tag must sit
// on its own line (optionally indented); a {#for} mentioned in a comment or a
// "quoted" text node is left untouched.
func ExpandControlFlow(source string) (string, error) {
	nodes, _, err := parseSeq(strings.Split(source, "\n"), 0, nil)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := renderNodes(&b, nodes, map[string]string{}); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// ── node model ────────────────────────────────────────────────────────

type cfNode interface{}

type cfText struct{ line string }
type cfFor struct {
	as, each string
	body     []cfNode
}
type cfCond struct {
	lhs, op, rhs string // op is "==" or "in"; rhs is a value or comma list
}
type cfBranch struct {
	cond cfCond
	body []cfNode
}
type cfIf struct {
	branches []cfBranch // the {#if} plus any {#elif}
	else_    []cfNode
}
type cfMatch struct {
	expr     string
	cases    []cfCase
	default_ []cfNode
}
type cfCase struct {
	value string
	body  []cfNode
}

// ── line classification ───────────────────────────────────────────────

var (
	reForOpen    = regexp.MustCompile(`^\{#for\s+(\w+)\s+in\s+(.+?)\s*\}$`)
	reForClose   = regexp.MustCompile(`^\{/for\}$`)
	reIfOpen     = regexp.MustCompile(`^\{#if\s+(.+?)\s*\}$`)
	reElif       = regexp.MustCompile(`^\{#elif\s+(.+?)\s*\}$`)
	reElse       = regexp.MustCompile(`^\{#else\}$`)
	reIfClose    = regexp.MustCompile(`^\{/if\}$`)
	reMatchOpen  = regexp.MustCompile(`^\{#match\s+(.+?)\s*\}$`)
	reMatchClose = regexp.MustCompile(`^\{/match\}$`)
	reCase       = regexp.MustCompile(`^\{#case\s+(.+?)\s*\}$`)
	reDefault    = regexp.MustCompile(`^\{#default\}$`)
	reInterp     = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)
)

// parseSeq reads sibling nodes starting at line i, stopping (without consuming)
// at the first line matching any regex in stops. Returns the nodes and the
// index of the stopping line (or len(lines) at EOF).
func parseSeq(lines []string, i int, stops []*regexp.Regexp) ([]cfNode, int, error) {
	var nodes []cfNode
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		for _, st := range stops {
			if st.MatchString(trimmed) {
				return nodes, i, nil
			}
		}
		switch {
		case reForOpen.MatchString(trimmed):
			n, next, err := parseFor(lines, i)
			if err != nil {
				return nil, 0, err
			}
			nodes, i = append(nodes, n), next
		case reIfOpen.MatchString(trimmed):
			n, next, err := parseIf(lines, i)
			if err != nil {
				return nil, 0, err
			}
			nodes, i = append(nodes, n), next
		case reMatchOpen.MatchString(trimmed):
			n, next, err := parseMatch(lines, i)
			if err != nil {
				return nil, 0, err
			}
			nodes, i = append(nodes, n), next
		default:
			nodes = append(nodes, cfText{lines[i]})
			i++
		}
	}
	return nodes, i, nil
}

func parseFor(lines []string, i int) (cfNode, int, error) {
	m := reForOpen.FindStringSubmatch(strings.TrimSpace(lines[i]))
	body, next, err := parseSeq(lines, i+1, []*regexp.Regexp{reForClose})
	if err != nil {
		return nil, 0, err
	}
	if next >= len(lines) {
		return nil, 0, fmt.Errorf("line %d: {#for} is never closed", i+1)
	}
	return cfFor{as: m[1], each: m[2], body: body}, next + 1, nil
}

func parseCond(s string) (cfCond, error) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, " in "); idx >= 0 {
		return cfCond{
			lhs: strings.TrimSpace(s[:idx]),
			op:  "in",
			rhs: strings.TrimSpace(s[idx+len(" in "):]),
		}, nil
	}
	if idx := strings.Index(s, "=="); idx >= 0 {
		return cfCond{
			lhs: strings.TrimSpace(s[:idx]),
			op:  "==",
			rhs: strings.TrimSpace(s[idx+2:]),
		}, nil
	}
	return cfCond{}, fmt.Errorf("condition %q must use `==` or `in`", s)
}

func parseIf(lines []string, i int) (cfNode, int, error) {
	start := i
	branchStops := []*regexp.Regexp{reElif, reElse, reIfClose}

	cond, err := parseCond(reIfOpen.FindStringSubmatch(strings.TrimSpace(lines[i]))[1])
	if err != nil {
		return nil, 0, fmt.Errorf("line %d: %v", i+1, err)
	}
	body, next, err := parseSeq(lines, i+1, branchStops)
	if err != nil {
		return nil, 0, err
	}
	node := cfIf{branches: []cfBranch{{cond: cond, body: body}}}
	i = next

	for i < len(lines) && reElif.MatchString(strings.TrimSpace(lines[i])) {
		c, err := parseCond(reElif.FindStringSubmatch(strings.TrimSpace(lines[i]))[1])
		if err != nil {
			return nil, 0, fmt.Errorf("line %d: %v", i+1, err)
		}
		b, n, err := parseSeq(lines, i+1, branchStops)
		if err != nil {
			return nil, 0, err
		}
		node.branches = append(node.branches, cfBranch{cond: c, body: b})
		i = n
	}

	if i < len(lines) && reElse.MatchString(strings.TrimSpace(lines[i])) {
		b, n, err := parseSeq(lines, i+1, []*regexp.Regexp{reIfClose})
		if err != nil {
			return nil, 0, err
		}
		node.else_ = b
		i = n
	}

	if i >= len(lines) {
		return nil, 0, fmt.Errorf("line %d: {#if} is never closed", start+1)
	}
	return node, i + 1, nil
}

func parseMatch(lines []string, i int) (cfNode, int, error) {
	start := i
	m := cfMatch{expr: strings.TrimSpace(reMatchOpen.FindStringSubmatch(strings.TrimSpace(lines[i]))[1])}
	caseStops := []*regexp.Regexp{reCase, reDefault, reMatchClose}
	i++
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		switch {
		case reMatchClose.MatchString(trimmed):
			return m, i + 1, nil
		case reCase.MatchString(trimmed):
			val := strings.TrimSpace(reCase.FindStringSubmatch(trimmed)[1])
			body, next, err := parseSeq(lines, i+1, caseStops)
			if err != nil {
				return nil, 0, err
			}
			m.cases = append(m.cases, cfCase{value: val, body: body})
			i = next
		case reDefault.MatchString(trimmed):
			body, next, err := parseSeq(lines, i+1, caseStops)
			if err != nil {
				return nil, 0, err
			}
			m.default_ = body
			i = next
		case trimmed == "":
			i++ // tolerate blank lines between cases
		default:
			return nil, 0, fmt.Errorf("line %d: only {#case}/{#default} may appear inside {#match}", i+1)
		}
	}
	return nil, 0, fmt.Errorf("line %d: {#match} is never closed", start+1)
}

// ── rendering ─────────────────────────────────────────────────────────

func renderNodes(b *strings.Builder, nodes []cfNode, env map[string]string) error {
	for _, n := range nodes {
		switch v := n.(type) {
		case cfText:
			b.WriteString(substVars(v.line, env))
			b.WriteByte('\n')
		case cfFor:
			for _, item := range splitList(substVars(v.each, env)) {
				child := cloneEnv(env)
				child[v.as] = resolve(item, env)
				if err := renderNodes(b, v.body, child); err != nil {
					return err
				}
			}
		case cfIf:
			matched := false
			for _, br := range v.branches {
				if evalCond(br.cond, env) {
					if err := renderNodes(b, br.body, env); err != nil {
						return err
					}
					matched = true
					break
				}
			}
			if !matched {
				if err := renderNodes(b, v.else_, env); err != nil {
					return err
				}
			}
		case cfMatch:
			val := resolve(v.expr, env)
			done := false
			for _, c := range v.cases {
				if resolve(c.value, env) == val {
					if err := renderNodes(b, c.body, env); err != nil {
						return err
					}
					done = true
					break
				}
			}
			if !done {
				if err := renderNodes(b, v.default_, env); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// evalCond evaluates a compile-time condition. `==` is string equality; `in`
// tests membership against a comma-separated list. Both sides resolve loop
// variables (bare identifiers / {{ name }}) against the environment, falling
// back to the literal token.
func evalCond(c cfCond, env map[string]string) bool {
	lhs := resolve(c.lhs, env)
	if c.op == "in" {
		for _, opt := range splitList(substVars(c.rhs, env)) {
			if resolve(opt, env) == lhs {
				return true
			}
		}
		return false
	}
	return lhs == resolve(c.rhs, env)
}

// resolve turns a directive-header operand into its value: a bare loop
// variable becomes its bound value, {{ name }} is interpolated, and anything
// else is treated as a literal.
func resolve(tok string, env map[string]string) string {
	tok = strings.TrimSpace(tok)
	if v, ok := env[tok]; ok {
		return v
	}
	return substVars(tok, env)
}

// substVars replaces {{ name }} text bindings with their bound value, leaving
// unbound bindings untouched.
func substVars(s string, env map[string]string) string {
	return reInterp.ReplaceAllStringFunc(s, func(match string) string {
		name := reInterp.FindStringSubmatch(match)[1]
		if v, ok := env[name]; ok {
			return v
		}
		return match
	})
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func cloneEnv(env map[string]string) map[string]string {
	c := make(map[string]string, len(env)+1)
	for k, v := range env {
		c[k] = v
	}
	return c
}
