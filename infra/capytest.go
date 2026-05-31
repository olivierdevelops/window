package infra

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// capytest.go — parser for the .capytest DSL: a readable, diff-friendly test
// format that a non-developer can author and that the interactive Test Bench
// can read and write. A suite names the .capyx app under test and lists
// scenarios; each scenario is a sequence of plain-English steps:
//
//   suite "Counter"
//   use ./counter.capyx
//
//   scenario "increments from zero"
//       mount counter
//       expect text "0"
//       click "+"
//       click "+"
//       expect text "2"
//
//   scenario "clock mock drives the timer"
//       mount timer
//       mock Clock.now returns 1000
//       call tick
//       expect state t 1000
//       expect called Clock.now 1
//
// It parses into a struct that serialises straight to the step schema the
// capyx_testkit.js interpreter consumes.

// CapyTest is a parsed .capytest suite.
type CapyTest struct {
	Suite     string         `json:"name"`
	Use       string         `json:"use"`
	Scenarios []CapyScenario `json:"scenarios"`
}

// CapyScenario is one named test case: an ordered list of steps.
type CapyScenario struct {
	Name  string           `json:"name"`
	Steps []map[string]any `json:"steps"`
}

// ParseCapyTest parses a .capytest source into a suite. Step ops mirror the
// kernel's interpreter schema exactly so the result can be JSON-marshalled and
// handed straight to runSuite.
func ParseCapyTest(src string) (*CapyTest, error) {
	t := &CapyTest{}
	lines := strings.Split(src, "\n")
	var cur *CapyScenario
	for n, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		word := firstWord(line)
		rest := strings.TrimSpace(line[len(word):])
		switch word {
		case "suite":
			t.Suite = stripQuotes(rest)
			cur = nil
		case "use":
			t.Use = stripQuotes(rest)
			cur = nil
		case "scenario":
			t.Scenarios = append(t.Scenarios, CapyScenario{Name: stripQuotes(rest)})
			cur = &t.Scenarios[len(t.Scenarios)-1]
		default:
			if cur == nil {
				return nil, fmt.Errorf("line %d: step %q before any scenario", n+1, word)
			}
			step, err := parseTestStep(word, rest)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", n+1, err)
			}
			cur.Steps = append(cur.Steps, step)
		}
	}
	if len(t.Scenarios) == 0 {
		return nil, fmt.Errorf("no scenarios found")
	}
	return t, nil
}

func parseTestStep(op, rest string) (map[string]any, error) {
	switch op {
	case "mount":
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return nil, fmt.Errorf("mount needs a component name")
		}
		step := map[string]any{"op": "mount", "component": fields[0]}
		for i := 1; i+1 < len(fields)+1 && i < len(fields); i++ {
			if fields[i] == "use" && i+1 < len(fields) {
				step["handler"] = fields[i+1]
			}
		}
		return step, nil

	case "set":
		field, val := splitFirst(rest)
		if field == "" {
			return nil, fmt.Errorf("set needs a field and value")
		}
		return map[string]any{"op": "set", "field": field, "value": parseValue(val)}, nil

	case "mock":
		// mock Cap.method returns <value>   |   mock Cap.method returns seq [a, b]
		head, tail := splitFirst(rest)
		cap, method, ok := splitDot(head)
		if !ok {
			return nil, fmt.Errorf("mock needs Capability.method, got %q", head)
		}
		kw, valStr := splitFirst(tail)
		if kw != "returns" {
			return nil, fmt.Errorf("mock %s.%s must be followed by `returns`", cap, method)
		}
		step := map[string]any{"op": "mock", "cap": cap, "method": method}
		if seqWord, seqRest := splitFirst(valStr); seqWord == "seq" {
			var vals []any
			if err := json.Unmarshal([]byte(seqRest), &vals); err != nil {
				return nil, fmt.Errorf("mock seq expects a JSON array: %w", err)
			}
			step["returns"] = map[string]any{"__seq": true, "values": vals}
		} else {
			step["returns"] = parseValue(valStr)
		}
		return step, nil

	case "click":
		return withTarget(map[string]any{"op": "click"}, rest), nil

	case "input":
		tgt, val := splitTargetValue(rest)
		return withTarget(map[string]any{"op": "input", "value": parseValue(val)}, tgt), nil

	case "fire":
		event, tgt := splitFirst(rest)
		if event == "" {
			return nil, fmt.Errorf("fire needs an event name")
		}
		return withTarget(map[string]any{"op": "fire", "event": event}, tgt), nil

	case "call":
		method, argStr := splitFirst(rest)
		if method == "" {
			return nil, fmt.Errorf("call needs a method name")
		}
		step := map[string]any{"op": "call", "method": method}
		if strings.TrimSpace(argStr) != "" {
			args, err := parseArgs(argStr)
			if err != nil {
				return nil, err
			}
			step["args"] = args
		}
		return step, nil

	case "expect":
		return parseExpect(rest)
	}
	return nil, fmt.Errorf("unknown step %q", op)
}

func parseExpect(rest string) (map[string]any, error) {
	kind, tail := splitFirst(rest)
	switch kind {
	case "text":
		return map[string]any{"op": "expectText", "value": stripQuotes(tail)}, nil
	case "no":
		k2, t2 := splitFirst(tail)
		if k2 != "text" {
			return nil, fmt.Errorf("expect no must be `expect no text ...`")
		}
		return map[string]any{"op": "expectText", "value": stripQuotes(t2), "negate": true}, nil
	case "state":
		field, val := splitFirst(tail)
		if field == "" {
			return nil, fmt.Errorf("expect state needs a field and value")
		}
		return map[string]any{"op": "expectState", "field": field, "value": parseValue(val)}, nil
	case "count":
		sel, nStr := splitFirst(tail)
		n, err := strconv.Atoi(strings.TrimSpace(nStr))
		if err != nil {
			return nil, fmt.Errorf("expect count needs a number: %w", err)
		}
		return map[string]any{"op": "expectCount", "selector": sel, "count": n}, nil
	case "called":
		head, nStr := splitFirst(tail)
		cap, method, ok := splitDot(head)
		if !ok {
			return nil, fmt.Errorf("expect called needs Capability.method")
		}
		step := map[string]any{"op": "expectCalled", "cap": cap, "method": method}
		if strings.TrimSpace(nStr) != "" {
			n, err := strconv.Atoi(strings.TrimSpace(nStr))
			if err != nil {
				return nil, fmt.Errorf("expect called count must be a number: %w", err)
			}
			step["count"] = n
		}
		return step, nil
	case "class":
		sel, cls := splitFirst(tail)
		return map[string]any{"op": "expectClass", "selector": sel, "class": stripQuotes(cls)}, nil
	}
	return nil, fmt.Errorf("unknown expectation %q", kind)
}

// ── helpers ──────────────────────────────────────────────────────────────

// withTarget attaches a click/input/fire target to a step: a quoted string is
// matched by visible text; a token starting with "." is a class selector.
func withTarget(step map[string]any, tgt string) map[string]any {
	tgt = strings.TrimSpace(tgt)
	if strings.HasPrefix(tgt, ".") {
		step["selector"] = tgt
	} else {
		step["text"] = stripQuotes(tgt)
	}
	return step
}

// splitTargetValue splits `<target> <value>` where target may be a quoted
// string (with spaces) or a bare token/selector.
func splitTargetValue(s string) (target, value string) {
	s = strings.TrimSpace(s)
	if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
		q := s[0]
		if end := strings.IndexByte(s[1:], q); end >= 0 {
			return s[:end+2], strings.TrimSpace(s[end+2:])
		}
	}
	return splitFirst(s)
}

// splitFirst splits a string into its first whitespace-delimited token and the
// remainder.
func splitFirst(s string) (head, tail string) {
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return s[:i], strings.TrimSpace(s[i+1:])
		}
	}
	return s, ""
}

func splitDot(s string) (left, right string, ok bool) {
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

// parseValue interprets a value token as JSON; a bare unquoted word that isn't
// valid JSON is taken as a string literal.
func parseValue(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	return stripQuotes(s)
}

// parseArgs interprets call arguments: a JSON array `[a, b]`, or a single value.
func parseArgs(s string) ([]any, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		var args []any
		if err := json.Unmarshal([]byte(s), &args); err != nil {
			return nil, fmt.Errorf("call args must be a JSON array: %w", err)
		}
		return args, nil
	}
	return []any{parseValue(s)}, nil
}
