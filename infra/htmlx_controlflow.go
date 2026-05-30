package infra

import (
	"fmt"
	"regexp"
	"strings"
)

// ExpandControlFlow expands HTML-native, compile-time control flow in a .htmlx
// source: <for>, <if>/<else>, and <switch>/<case>/<default>. Each construct is
// evaluated during transpilation (there is no runtime data model) and composes
// with the others — an <if> inside a <for> sees the loop variable.
//
//	<for each="Home, About, Contact" as="label">
//	  <li><a href="#">"{label}"</a></li>
//	</for>
//
//	<if value="{role}" is="admin"> … <else> … </if>
//
//	<switch value="{state}">
//	  <case is="ok"> … </case>
//	  <default> … </default>
//	</switch>
//
// Loop variables and props are referenced as {name} and substituted as raw text
// (htmlx still escapes quoted text nodes downstream). Each control tag must sit
// on its own line (optionally indented); a <for> mentioned in a comment or a
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
	each, as string
	body     []cfNode
}
type cfIf struct {
	value, is, in   string
	then_, else_    []cfNode
}
type cfSwitch struct {
	value    string
	cases    []cfCase
	default_ []cfNode
}
type cfCase struct {
	is   string
	body []cfNode
}

// ── line classification ───────────────────────────────────────────────

var (
	reForOpen     = regexp.MustCompile(`^<for\s+(.*?)\s*>$`)
	reForClose    = regexp.MustCompile(`^</for>$`)
	reIfOpen      = regexp.MustCompile(`^<if\s+(.*?)\s*>$`)
	reIfClose     = regexp.MustCompile(`^</if>$`)
	reElse        = regexp.MustCompile(`^<else\s*/?>$`)
	reSwitchOpen  = regexp.MustCompile(`^<switch\s+(.*?)\s*>$`)
	reSwitchClose = regexp.MustCompile(`^</switch>$`)
	reCaseOpen    = regexp.MustCompile(`^<case\s+(.*?)\s*>$`)
	reCaseClose   = regexp.MustCompile(`^</case>$`)
	reDefaultOpen = regexp.MustCompile(`^<default\s*>$`)
	reDefaultClose = regexp.MustCompile(`^</default>$`)
	reAttr        = regexp.MustCompile(`(\w+)\s*=\s*"([^"]*)"`)
)

func attrMap(s string) map[string]string {
	m := map[string]string{}
	for _, kv := range reAttr.FindAllStringSubmatch(s, -1) {
		m[kv[1]] = kv[2]
	}
	return m
}

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
		case reSwitchOpen.MatchString(trimmed):
			n, next, err := parseSwitch(lines, i)
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
	a := attrMap(reForOpen.FindStringSubmatch(strings.TrimSpace(lines[i]))[1])
	if a["each"] == "" || a["as"] == "" {
		return nil, 0, fmt.Errorf("line %d: <for> needs each=\"…\" and as=\"…\"", i+1)
	}
	body, next, err := parseSeq(lines, i+1, []*regexp.Regexp{reForClose})
	if err != nil {
		return nil, 0, err
	}
	if next >= len(lines) {
		return nil, 0, fmt.Errorf("line %d: <for> is never closed", i+1)
	}
	return cfFor{each: a["each"], as: a["as"], body: body}, next + 1, nil
}

func parseIf(lines []string, i int) (cfNode, int, error) {
	a := attrMap(reIfOpen.FindStringSubmatch(strings.TrimSpace(lines[i]))[1])
	if a["value"] == "" || (a["is"] == "" && a["in"] == "") {
		return nil, 0, fmt.Errorf("line %d: <if> needs value=\"…\" and is=\"…\" (or in=\"…\")", i+1)
	}
	then_, next, err := parseSeq(lines, i+1, []*regexp.Regexp{reIfClose, reElse})
	if err != nil {
		return nil, 0, err
	}
	if next >= len(lines) {
		return nil, 0, fmt.Errorf("line %d: <if> is never closed", i+1)
	}
	var else_ []cfNode
	if reElse.MatchString(strings.TrimSpace(lines[next])) {
		else_, next, err = parseSeq(lines, next+1, []*regexp.Regexp{reIfClose})
		if err != nil {
			return nil, 0, err
		}
		if next >= len(lines) {
			return nil, 0, fmt.Errorf("line %d: <if> is never closed", i+1)
		}
	}
	return cfIf{value: a["value"], is: a["is"], in: a["in"], then_: then_, else_: else_}, next + 1, nil
}

func parseSwitch(lines []string, i int) (cfNode, int, error) {
	a := attrMap(reSwitchOpen.FindStringSubmatch(strings.TrimSpace(lines[i]))[1])
	if a["value"] == "" {
		return nil, 0, fmt.Errorf("line %d: <switch> needs value=\"…\"", i+1)
	}
	sw := cfSwitch{value: a["value"]}
	i++
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		switch {
		case reSwitchClose.MatchString(trimmed):
			return sw, i + 1, nil
		case reCaseOpen.MatchString(trimmed):
			ca := attrMap(reCaseOpen.FindStringSubmatch(trimmed)[1])
			body, next, err := parseSeq(lines, i+1, []*regexp.Regexp{reCaseClose})
			if err != nil {
				return nil, 0, err
			}
			if next >= len(lines) {
				return nil, 0, fmt.Errorf("line %d: <case> is never closed", i+1)
			}
			sw.cases = append(sw.cases, cfCase{is: ca["is"], body: body})
			i = next + 1
		case reDefaultOpen.MatchString(trimmed):
			body, next, err := parseSeq(lines, i+1, []*regexp.Regexp{reDefaultClose})
			if err != nil {
				return nil, 0, err
			}
			if next >= len(lines) {
				return nil, 0, fmt.Errorf("line %d: <default> is never closed", i+1)
			}
			sw.default_ = body
			i = next + 1
		case trimmed == "":
			i++ // tolerate blank lines between cases
		default:
			return nil, 0, fmt.Errorf("line %d: only <case>/<default> may appear inside <switch>", i+1)
		}
	}
	return nil, 0, fmt.Errorf("line %d: <switch> is never closed", i+1)
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
				child[v.as] = item
				if err := renderNodes(b, v.body, child); err != nil {
					return err
				}
			}
		case cfIf:
			val := substVars(v.value, env)
			match := false
			if v.in != "" {
				for _, opt := range splitList(substVars(v.in, env)) {
					if opt == val {
						match = true
						break
					}
				}
			} else {
				match = val == substVars(v.is, env)
			}
			if match {
				if err := renderNodes(b, v.then_, env); err != nil {
					return err
				}
			} else if err := renderNodes(b, v.else_, env); err != nil {
				return err
			}
		case cfSwitch:
			val := substVars(v.value, env)
			done := false
			for _, c := range v.cases {
				if substVars(c.is, env) == val {
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

func substVars(s string, env map[string]string) string {
	for k, val := range env {
		s = strings.ReplaceAll(s, "{"+k+"}", val)
	}
	return s
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
