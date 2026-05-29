package appio

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"webview_gui/assets"
	"webview_gui/domain"
	"webview_gui/infra"
)

// ParseCLI parses command-line arguments and returns an AppConfig ready to run,
// or nil if the command was handled internally (--version, --init, --mac-app).
func ParseCLI() *domain.AppConfig {
	var configPath string
	if len(os.Args) <= 1 {
		configPath = "./window.yaml"
	} else {
		configPath = os.Args[1]
	}

	switch configPath {
	case "-v", "--version":
		fmt.Println("1.2.0")
		return nil

	case "--init", "-i", "-init":
		dir := ""
		var err error
		if len(os.Args) > 2 {
			dir = os.Args[2]
		} else {
			dir, err = os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "window.yaml"), assets.WindowDefaultYAML, 0666); err != nil {
			log.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "static"), 0755); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "static", "index.html"), assets.IndexHTML, 0666); err != nil {
			log.Fatal(err)
		}
		return nil

	case "--mac-app", "-mac-app":
		cfg := "./window.yaml"
		if len(os.Args) > 2 {
			cfg = os.Args[2]
		}
		if err := infra.BuildMacApp(cfg); err != nil {
			log.Fatal(err)
		}
		return nil

	case "--build", "--capy":
		// Explicitly transpile a .window source and run it. The bare form
		// `window app.window` (handled below by extension) does the same.
		if len(os.Args) <= 2 {
			log.Fatal("usage: window <app.window>")
		}
		if os.Getenv("DEBUG") != "1" {
			log.SetOutput(&noLog{})
		}
		cfg, err := loadWindowLang(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		return cfg
	}

	if os.Getenv("DEBUG") != "1" {
		log.SetOutput(&noLog{})
	}

	ext := strings.ToLower(filepath.Ext(configPath))
	var cfg *domain.AppConfig
	var err error

	switch ext {
	case ".yaml":
		cfg, err = LoadApp(configPath, false)
	case ".json":
		cfg, err = LoadApp(configPath, true)
	case ".window":
		cfg, err = loadWindowLang(configPath)
	default:
		cfg, err = LoadAppForContentView(configPath)
	}

	if err != nil {
		log.Fatal(err)
	}
	return cfg
}

// loadWindowLang transpiles a .window source — window's own declarative app
// language — into a runnable app and returns its config. The Capy engine is
// the transpiler under the hood; the embedded window.capy library defines the
// language. Generated files (window.yaml + static/*) go to a temp dir.
func loadWindowLang(scriptPath string) (*domain.AppConfig, error) {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}
	files, err := infra.GenerateCapyApp(string(assets.WindowCapyLib), string(src))
	if err != nil {
		return nil, fmt.Errorf("transpile %s: %w", scriptPath, err)
	}
	dir, err := os.MkdirTemp("", "winapp_*")
	if err != nil {
		return nil, err
	}
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return nil, err
		}
	}
	return LoadApp(filepath.Join(dir, "window.yaml"), false)
}

type noLog struct{}

func (*noLog) Write(p []byte) (int, error) { return len(p), nil }
