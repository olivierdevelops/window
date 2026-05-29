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

	case "--capy", "-capy":
		if len(os.Args) <= 2 {
			log.Fatal("usage: window --capy <app.window>")
		}
		src, err := os.ReadFile(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		files, err := infra.GenerateCapyApp(string(assets.WindowCapyLib), string(src))
		if err != nil {
			log.Fatalf("capy: %v", err)
		}
		dir, err := os.MkdirTemp("", "capyapp_*")
		if err != nil {
			log.Fatal(err)
		}
		for rel, content := range files {
			full := filepath.Join(dir, rel)
			if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
				log.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(content), 0644); err != nil {
				log.Fatal(err)
			}
		}
		if os.Getenv("DEBUG") != "1" {
			log.SetOutput(&noLog{})
		}
		cfg, err := LoadApp(filepath.Join(dir, "window.yaml"), false)
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
	default:
		cfg, err = LoadAppForContentView(configPath)
	}

	if err != nil {
		log.Fatal(err)
	}
	return cfg
}

type noLog struct{}

func (*noLog) Write(p []byte) (int, error) { return len(p), nil }
