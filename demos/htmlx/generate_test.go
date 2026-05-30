package htmlxdemos_test

// Regression test: every demos/htmlx/*.htmlx source must compile through the
// embedded htmlx.capy library and produce a runnable app (window.yaml +
// static/index.html). Also asserts that matched-pair nesting is enforced: a
// mismatched closing tag must be a hard transpile error. Imports the Capy
// engine directly (pure Go, no CGo/webview), so it runs fast in CI.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olivierdevelops/capy"
	"webview_gui/infra"
)

func loadLib(t *testing.T) *capy.Library {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", "assets", "htmlx.capy"))
	if err != nil {
		t.Fatalf("read htmlx.capy: %v", err)
	}
	lib, err := capy.NewLibrary(string(src))
	if err != nil {
		t.Fatalf("compile htmlx.capy: %v", err)
	}
	return lib
}

func TestHTMLXDemosGenerate(t *testing.T) {
	lib := loadLib(t)

	sources, err := filepath.Glob("*.htmlx")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) < 4 {
		t.Fatalf("expected several demos, found %d", len(sources))
	}

	for _, src := range sources {
		src := src
		t.Run(strings.TrimSuffix(src, ".htmlx"), func(t *testing.T) {
			b, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}
			// Expand HTML-native <component> definitions and <for>/<if>/<switch>
			// control flow the same way the CLI does.
			rewritten, err := infra.RewriteHTMLXComponents(string(b))
			if err != nil {
				t.Fatalf("%s: component rewrite failed: %v", src, err)
			}
			expanded, err := infra.ExpandControlFlow(rewritten)
			if err != nil {
				t.Fatalf("%s: control-flow expansion failed: %v", src, err)
			}
			primary, files, err := lib.RunMulti(expanded)
			if err != nil {
				t.Fatalf("%s did not compile: %v", src, err)
			}
			if !strings.Contains(primary, "<!doctype html>") {
				t.Errorf("%s: primary output is not a full HTML document", src)
			}
			if !strings.Contains(files["window.yaml"], "entry_path:") {
				t.Errorf("%s: window.yaml missing entry_path", src)
			}
		})
	}
}

func TestHTMLXRejectsMismatchedTags(t *testing.T) {
	lib := loadLib(t)
	bad := `<app title="X" width="640" height="520">
  <div class="card"><p>"oops"</div>
</app>`
	if _, _, err := lib.RunMulti(bad); err == nil {
		t.Fatal("expected a transpile error for mismatched </div> closing a <p>, got none")
	}
}
