package appio

import (
	"os"
	"path/filepath"
	"testing"

	"webview_gui/domain"
)

func withCwd(t *testing.T) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return func() { os.Chdir(orig) }
}

func TestLoadApp_YAML(t *testing.T) {
	restore := withCwd(t)
	defer restore()

	dir := t.TempDir()
	yaml := `title: "Test App"
entry_path: ./static/index.html
`
	path := filepath.Join(dir, "window.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadApp(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Test App" {
		t.Errorf("Title = %q, want %q", cfg.Title, "Test App")
	}
	if cfg.EntryPath != "./static/index.html" {
		t.Errorf("EntryPath = %q", cfg.EntryPath)
	}
}

func TestLoadApp_JSON(t *testing.T) {
	restore := withCwd(t)
	defer restore()

	dir := t.TempDir()
	jsonContent := `{"title":"JSON App","entry_path":"./index.html"}`
	path := filepath.Join(dir, "window.json")
	os.WriteFile(path, []byte(jsonContent), 0644)

	cfg, err := LoadApp(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "JSON App" {
		t.Errorf("Title = %q, want %q", cfg.Title, "JSON App")
	}
}

func TestLoadApp_NativeFeatures(t *testing.T) {
	restore := withCwd(t)
	defer restore()

	dir := t.TempDir()
	yaml := `title: "Native App"
native_features:
  - fs
  - os
  - dialogs
`
	path := filepath.Join(dir, "window.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadApp(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.NativeFeatures) != 3 {
		t.Fatalf("expected 3 native features, got %d", len(cfg.NativeFeatures))
	}
	features := map[domain.NativeFeature]bool{}
	for _, f := range cfg.NativeFeatures {
		features[f] = true
	}
	if !features[domain.NativeFS] {
		t.Error("expected NativeFS")
	}
	if !features[domain.NativeOS] {
		t.Error("expected NativeOS")
	}
	if !features[domain.NativeDialogs] {
		t.Error("expected NativeDialogs")
	}
}

func TestLoadApp_NotFound(t *testing.T) {
	_, err := LoadApp("/tmp/does_not_exist_xyz.yaml", false)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadApp_ChdirToFileDir(t *testing.T) {
	restore := withCwd(t)
	defer restore()

	dir := t.TempDir()
	yaml := `title: "Chdir Test"`
	path := filepath.Join(dir, "window.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadApp(path, false)
	if err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	// Resolve symlinks so macOS /var → /private/var comparison works.
	wantResolved, _ := filepath.EvalSymlinks(dir)
	gotResolved, _ := filepath.EvalSymlinks(cwd)
	if gotResolved != wantResolved {
		t.Errorf("cwd = %q, want %q", gotResolved, wantResolved)
	}
}

func TestLoadAppForContentView(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello"), 0644)

	cfg, err := LoadAppForContentView(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "test.md" {
		t.Errorf("Title = %q, want test.md", cfg.Title)
	}
	if cfg.HTML == "" {
		t.Error("expected non-empty HTML")
	}
	urlPath := "/test.md"
	if _, ok := cfg.Files[urlPath]; !ok {
		t.Errorf("expected Files[%q] to be set", urlPath)
	}
}
