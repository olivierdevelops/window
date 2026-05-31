package infra

import (
	"fmt"
	"regexp"
	"strings"
)

// RewriteHTMLXComponents translates HTML-native component definitions into the
// Capy `define … end` blocks that the htmlx grammar understands.
//
// Authors write a clean, HTML-looking definition:
//
//	<component name="card" props="title">
//	  <section class="card">
//	    <h3>{{ title }}</h3>
//	    <slot></slot>
//	  </section>
//	</component>
//
// and then use it like any built-in tag: <card title="Welcome">…</card>.
// Inside a definition body, {{ prop }} interpolates an escaped prop value and
// <slot></slot> marks where nested children go. Add the bare `void` attribute
// for a self-closing tag (<avatar name="Ada" />).
//
// Each block is removed from the source and an equivalent `define` is prepended
// (defines must precede use and sit at column 0). The returned source is fed to
// the htmlx library unchanged otherwise.
func RewriteHTMLXComponents(source string) (string, error) {
	matches := componentBlockRE.FindAllStringSubmatchIndex(source, -1)
	if len(matches) == 0 {
		return source, nil
	}

	var defines []string
	var b strings.Builder
	last := 0
	for _, m := range matches {
		b.WriteString(source[last:m[0]])
		last = m[1]
		attrs := source[m[2]:m[3]]
		body := source[m[4]:m[5]]
		def, err := componentToDefine(attrs, body)
		if err != nil {
			return "", err
		}
		defines = append(defines, def)
	}
	b.WriteString(source[last:])

	return strings.Join(defines, "\n") + "\n" + b.String(), nil
}

var (
	// Anchored to line start (optionally indented) so a `<component>` mentioned
	// in a # comment or a "quoted" text node is never mistaken for a definition.
	componentBlockRE = regexp.MustCompile(`(?ms)^[ \t]*<component\b([^>]*)>(.*?)^[ \t]*</component>[ \t]*$`)
	attrNameRE       = regexp.MustCompile(`name\s*=\s*"([^"]*)"`)
	attrPropsRE      = regexp.MustCompile(`props\s*=\s*"([^"]*)"`)
	identRE          = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	slotRE           = regexp.MustCompile(`<slot\s*>\s*</slot>|<slot\s*/>`)
)

func componentToDefine(attrs, body string) (string, error) {
	nameM := attrNameRE.FindStringSubmatch(attrs)
	if nameM == nil {
		return "", fmt.Errorf("<component> is missing a name=\"…\" attribute")
	}
	name := nameM[1]
	if !identRE.MatchString(name) {
		return "", fmt.Errorf("<component name=%q>: name must be a tag identifier", name)
	}

	var props []string
	if pm := attrPropsRE.FindStringSubmatch(attrs); pm != nil {
		for _, p := range strings.Fields(pm[1]) {
			if !identRE.MatchString(p) {
				return "", fmt.Errorf("<component name=%q>: invalid prop %q", name, p)
			}
			props = append(props, p)
		}
	}
	void := regexp.MustCompile(`\bvoid\b`).MatchString(attrs)

	var lines []string
	lines = append(lines, "define "+name)
	lines = append(lines, `    arg literal "<"`)
	lines = append(lines, fmt.Sprintf(`    arg literal %q`, name))
	for _, p := range props {
		lines = append(lines, fmt.Sprintf(`    arg literal %q`, p))
		lines = append(lines, `    arg literal "="`)
		lines = append(lines, fmt.Sprintf(`    arg capture %s raw`, p))
	}
	if void {
		lines = append(lines, `    arg literal "/"`)
		lines = append(lines, `    arg literal ">"`)
	} else {
		lines = append(lines, `    arg literal ">"`)
		lines = append(lines, fmt.Sprintf(`    block_close_seq "</" %q ">"`, name))
	}
	lines = append(lines, "    template")
	lines = append(lines, rewriteBody(body, props))
	lines = append(lines, "    end")
	lines = append(lines, "end")
	return strings.Join(lines, "\n"), nil
}

// rewriteBody turns {{ prop }} into ${escapeHtml prop}, <slot></slot> into
// ${body}, then dedents and re-indents the template body to 8 spaces.
func rewriteBody(body string, props []string) string {
	body = slotRE.ReplaceAllLiteralString(body, "${body}")
	for _, p := range props {
		// {{ prop }} with optional surrounding whitespace.
		re := regexp.MustCompile(`\{\{\s*` + regexp.QuoteMeta(p) + `\s*\}\}`)
		body = re.ReplaceAllString(body, "${escapeHtml "+p+"}")
	}

	raw := strings.Split(body, "\n")
	// Drop leading/trailing blank lines.
	for len(raw) > 0 && strings.TrimSpace(raw[0]) == "" {
		raw = raw[1:]
	}
	for len(raw) > 0 && strings.TrimSpace(raw[len(raw)-1]) == "" {
		raw = raw[:len(raw)-1]
	}
	// Strip the common leading indentation.
	minIndent := -1
	for _, ln := range raw {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		n := len(ln) - len(strings.TrimLeft(ln, " \t"))
		if minIndent == -1 || n < minIndent {
			minIndent = n
		}
	}
	if minIndent < 0 {
		minIndent = 0
	}
	out := make([]string, len(raw))
	for i, ln := range raw {
		if len(ln) >= minIndent {
			ln = ln[minIndent:]
		}
		out[i] = "        " + ln
	}
	return strings.Join(out, "\n")
}
