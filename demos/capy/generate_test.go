package capydemos_test

// Regression test: every demos/capy/*.window source must compile through the
// embedded window.capy library and produce the expected app files. Imports the
// Capy engine directly (pure Go, no CGo/webview), so it runs fast in CI.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luowensheng/capy"
)

func TestCapyDemosGenerate(t *testing.T) {
	librarySrc, err := os.ReadFile(filepath.Join("..", "..", "assets", "window.capy"))
	if err != nil {
		t.Fatalf("read window.capy: %v", err)
	}
	lib, err := capy.NewLibrary(string(librarySrc))
	if err != nil {
		t.Fatalf("compile window.capy: %v", err)
	}

	sources, err := filepath.Glob("*.window")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) < 30 {
		t.Fatalf("expected 30+ demos, found %d", len(sources))
	}

	for _, src := range sources {
		src := src
		t.Run(strings.TrimSuffix(src, ".window"), func(t *testing.T) {
			b, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}
			_, files, err := lib.RunMulti(string(b))
			if err != nil {
				t.Fatalf("%s did not compile: %v", src, err)
			}
			for _, want := range []string{"window.yaml", "static/index.html", "static/app.js"} {
				if _, ok := files[want]; !ok {
					t.Errorf("%s: missing generated %s", src, want)
				}
			}
			if !strings.Contains(files["window.yaml"], "entry_path:") {
				t.Errorf("%s: window.yaml missing entry_path", src)
			}
		})
	}
}
