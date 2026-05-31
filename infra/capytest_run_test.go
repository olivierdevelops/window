package infra

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"webview_gui/assets"
)

// capytest_run_test.go — proves the headless .capytest pipeline end to end:
// parse a .capytest suite, compile the referenced .capyx in harness mode, run
// every scenario under Node, and assert the structured pass/fail results. This
// is the browser-free, Selenium-free interface test the feature promises.

func readDemo(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "demos", "capyx", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func TestCapyTestCounterSuite(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH; headless .capytest runner needs it")
	}
	suite, err := ParseCapyTest(readDemo(t, "counter.capytest"))
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	if len(suite.Scenarios) != 5 {
		t.Fatalf("expected 5 scenarios, got %d", len(suite.Scenarios))
	}
	res, err := RunCapyTest(suite, readDemo(t, "counter.capyx"),
		string(assets.CapyxRuntimeJS), string(assets.CapyxTestkitJS), string(assets.CapyxDOMShimJS))
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	if res.Failed != 0 {
		for _, sc := range res.Results {
			if !sc.Ok {
				t.Errorf("scenario %q failed (error=%q):", sc.Name, sc.Error)
				for _, st := range sc.Steps {
					if !st.Ok {
						t.Errorf("    step %q: %s", st.Op, st.Message)
					}
				}
			}
		}
		t.Fatalf("%d/%d scenarios failed", res.Failed, res.Passed+res.Failed)
	}
	if res.Passed != 5 {
		t.Fatalf("expected 5 passing scenarios, got %d", res.Passed)
	}
}

// TestCapyTestNotesMocks proves capability mocking: a handler whose mount()
// lifecycle and events call an injected Store capability is tested in isolation
// against canned (and sequenced) mock returns, with call counts asserted.
func TestCapyTestNotesMocks(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	suite, err := ParseCapyTest(readDemo(t, "notes.capytest"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	res, err := RunCapyTest(suite, readDemo(t, "notes.capyx"),
		string(assets.CapyxRuntimeJS), string(assets.CapyxTestkitJS), string(assets.CapyxDOMShimJS))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Failed != 0 {
		for _, sc := range res.Results {
			if !sc.Ok {
				t.Errorf("scenario %q failed (error=%q):", sc.Name, sc.Error)
				for _, st := range sc.Steps {
					if !st.Ok {
						t.Errorf("    step %q: %s", st.Op, st.Message)
					}
				}
			}
		}
		t.Fatalf("%d scenarios failed", res.Failed)
	}
}

// TestCapyTestDetectsRegressions proves a wrong expectation actually fails (the
// runner isn't trivially green).
func TestCapyTestDetectsRegressions(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	src := `suite "neg"
use ./counter.capyx
scenario "wrong expectation"
    mount counter
    click "+"
    expect state count 99
`
	suite, err := ParseCapyTest(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	res, err := RunCapyTest(suite, readDemo(t, "counter.capyx"),
		string(assets.CapyxRuntimeJS), string(assets.CapyxTestkitJS), string(assets.CapyxDOMShimJS))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Failed != 1 {
		t.Fatalf("expected the bogus scenario to fail, got passed=%d failed=%d", res.Passed, res.Failed)
	}
}
