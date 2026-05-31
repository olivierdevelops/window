package infra

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// capytest_run.go — the headless .capytest runner. It compiles the app under
// test in harness mode, assembles a single self-contained Node program (DOM
// shim + signals runtime + harness app JS + test kernel + the suite as JSON +
// a driver that runs it), executes it under Node, and parses the structured
// results. No browser, no Selenium — the same kernel the interactive Test Bench
// uses, driven headlessly for CI.

// StepResult is the outcome of a single scenario step.
type StepResult struct {
	Op      string `json:"op"`
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
}

// ScenarioResult is the outcome of one scenario.
type ScenarioResult struct {
	Name  string       `json:"name"`
	Ok    bool         `json:"ok"`
	Steps []StepResult `json:"steps"`
	Error string       `json:"error"`
}

// SuiteResult is the outcome of a whole suite run.
type SuiteResult struct {
	Name    string           `json:"name"`
	Results []ScenarioResult `json:"results"`
	Passed  int              `json:"passed"`
	Failed  int              `json:"failed"`
}

// RunCapyTest compiles capyxSrc in harness mode and runs the suite under Node,
// returning the structured results. runtimeJS, testkitJS and domShimJS are the
// embedded assets (passed in so infra stays free of asset knowledge).
func RunCapyTest(suite *CapyTest, capyxSrc, runtimeJS, testkitJS, domShimJS string) (*SuiteResult, error) {
	node, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("headless .capytest runs require Node.js on PATH (the interactive bench does not): %w", err)
	}
	harnessJS, err := CompileCapyxHarnessJS(capyxSrc, runtimeJS)
	if err != nil {
		return nil, fmt.Errorf("compile harness: %w", err)
	}
	suiteJSON, err := json.Marshal(suite)
	if err != nil {
		return nil, err
	}

	var prog strings.Builder
	prog.WriteString(domShimJS)
	prog.WriteString("\n")
	prog.WriteString(runtimeJS)
	prog.WriteString("\n")
	prog.WriteString(harnessJS)
	prog.WriteString("\n")
	prog.WriteString(testkitJS)
	prog.WriteString("\nvar __SUITE__ = ")
	prog.Write(suiteJSON)
	prog.WriteString(";\n")
	prog.WriteString("try { var __OUT__ = CAPYX_TEST_KIT.runSuite(__SUITE__); console.log(\"__RESULT__\" + JSON.stringify(__OUT__)); }\n")
	prog.WriteString("catch (e) { console.log(\"__ERROR__\" + (e && e.stack ? e.stack : e)); process.exit(1); }\n")

	dir, err := os.MkdirTemp("", "capytest_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	progPath := filepath.Join(dir, "run.js")
	if err := os.WriteFile(progPath, []byte(prog.String()), 0644); err != nil {
		return nil, err
	}

	out, err := exec.Command(node, progPath).CombinedOutput()
	text := string(out)
	if err != nil {
		return nil, fmt.Errorf("node execution failed: %v\n%s", err, text)
	}
	line := resultLine(text, "__RESULT__")
	if line == "" {
		if e := resultLine(text, "__ERROR__"); e != "" {
			return nil, fmt.Errorf("kernel error: %s", e)
		}
		return nil, fmt.Errorf("no result emitted; output:\n%s", text)
	}
	var res SuiteResult
	if err := json.Unmarshal([]byte(line), &res); err != nil {
		return nil, fmt.Errorf("parse result: %w\nline: %s", err, line)
	}
	return &res, nil
}

func resultLine(out, prefix string) string {
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, prefix) {
			return strings.TrimPrefix(ln, prefix)
		}
	}
	return ""
}
