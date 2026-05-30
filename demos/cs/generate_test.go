package csdemos_test

// Regression test: every demos/cs/*.cs source must compile through the
// embedded capyscript.capy library into JavaScript (primary output) plus a
// runnable app (window.yaml + static/index.html). Also asserts that an
// unclosed block is a hard transpile error. Imports the Capy engine directly
// (pure Go, no CGo/webview), so it runs fast in CI.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olivierdevelops/capy"
)

func loadLib(t *testing.T) *capy.Library {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", "assets", "capyscript.capy"))
	if err != nil {
		t.Fatalf("read capyscript.capy: %v", err)
	}
	lib, err := capy.NewLibrary(string(src))
	if err != nil {
		t.Fatalf("compile capyscript.capy: %v", err)
	}
	return lib
}

func TestCapyScriptDemosGenerate(t *testing.T) {
	lib := loadLib(t)

	sources, err := filepath.Glob("*.cs")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) < 4 {
		t.Fatalf("expected several demos, found %d", len(sources))
	}

	for _, src := range sources {
		src := src
		t.Run(strings.TrimSuffix(src, ".cs"), func(t *testing.T) {
			b, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}
			primary, files, err := lib.RunMulti(string(b))
			if err != nil {
				t.Fatalf("%s did not compile: %v", src, err)
			}
			if !strings.Contains(primary, "console.log") {
				t.Errorf("%s: compiled JS has no console.log output", src)
			}
			if !strings.Contains(files["window.yaml"], "entry_path:") {
				t.Errorf("%s: window.yaml missing entry_path", src)
			}
			if !strings.Contains(files["static/index.html"], "/static/app.js") {
				t.Errorf("%s: index.html does not load the compiled app.js", src)
			}
		})
	}
}

func TestCapyScriptRejectsUnclosedBlock(t *testing.T) {
	lib := loadLib(t)
	bad := `fn broken(n)
    return n
`
	if _, _, err := lib.RunMulti(bad); err == nil {
		t.Fatal("expected a transpile error for an unclosed fn block, got none")
	}
}
