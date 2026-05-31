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

	case "test":
		// window test <suite.capytest>      → headless run (CI-friendly)
		// window test --ui <app.capyx>       → open the interactive Test Bench
		return handleTestCommand(os.Args[2:])

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
	case ".htmlx":
		cfg, err = loadHTMLX(configPath)
	case ".capyx":
		cfg, err = loadCapyx(configPath)
	case ".capytest":
		runCapyTestSuite(configPath) // prints results and exits
		return nil
	case ".cs":
		cfg, err = loadCapyScript(configPath)
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
	return transpileApp(assets.WindowCapyLib, scriptPath)
}

// loadHTMLX transpiles a .htmlx source — matched-pair, angle-bracket HTML
// (`<tag>…</tag>`) parsed by the embedded htmlx.capy library — into a runnable
// window app. Capy's sequence closers validate tag nesting; the wrapped,
// normalized HTML document becomes static/index.html.
func loadHTMLX(scriptPath string) (*domain.AppConfig, error) {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}
	// Expand HTML-native <component name="…">…</component> definitions into the
	// Capy `define` blocks the htmlx grammar consumes, then evaluate compile-time
	// control flow (<for>/<if>/<switch>), then transpile.
	rewritten, err := infra.RewriteHTMLXComponents(string(src))
	if err != nil {
		return nil, fmt.Errorf("transpile %s: %w", scriptPath, err)
	}
	expanded, err := infra.ExpandControlFlow(rewritten)
	if err != nil {
		return nil, fmt.Errorf("transpile %s: %w", scriptPath, err)
	}
	return transpileSource(assets.HtmlxCapyLib, expanded, scriptPath)
}

// loadCapyx transpiles a .capyx source — a single-file reactive VHCO app (dumb
// `component` views, `handler` state/event units, `capability`/`provide`
// boundaries, an optional `orchestrator`, and `mount` lines) — into a runnable
// window app. The fine-grained signals runtime is inlined into the generated
// static/index.html. Generated files go to a temp dir.
func loadCapyx(scriptPath string) (*domain.AppConfig, error) {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}
	files, err := infra.CompileCapyx(string(src), string(assets.CapyxRuntimeJS))
	if err != nil {
		return nil, fmt.Errorf("transpile %s: %w", scriptPath, err)
	}
	dir, err := os.MkdirTemp("", "capyx_*")
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

// loadCapyScript transpiles a .cs source — a tiny JS-like scripting language
// (let/const/fn/if-else/for-in/while/return/log) parsed by the embedded
// capyscript.capy library — into a runnable window app. The compiled
// JavaScript becomes static/app.js and runs inside a console-style window.
func loadCapyScript(scriptPath string) (*domain.AppConfig, error) {
	return transpileApp(assets.CapyScriptLib, scriptPath)
}

// transpileApp runs a source file through a Capy library, writes the generated
// app files (window.yaml + static/*) to a temp dir, and loads the resulting
// app config.
func transpileApp(library []byte, scriptPath string) (*domain.AppConfig, error) {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}
	return transpileSource(library, string(src), scriptPath)
}

// transpileSource runs an in-memory source string through a Capy library,
// writes the generated app files (window.yaml + static/*) to a temp dir, and
// loads the resulting app config.
func transpileSource(library []byte, src, label string) (*domain.AppConfig, error) {
	files, err := infra.GenerateCapyApp(string(library), src)
	if err != nil {
		return nil, fmt.Errorf("transpile %s: %w", label, err)
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

// handleTestCommand implements `window test …`. With --ui it opens the
// interactive Test Bench for a .capyx app (returning a runnable AppConfig);
// otherwise it runs a .capytest suite headlessly and exits with a CI status.
func handleTestCommand(args []string) *domain.AppConfig {
	ui := false
	var path string
	for _, a := range args {
		if a == "--ui" || a == "-ui" {
			ui = true
		} else if path == "" {
			path = a
		}
	}
	if path == "" {
		log.Fatal("usage: window test <suite.capytest>  |  window test --ui <app.capyx>")
	}
	if os.Getenv("DEBUG") != "1" {
		log.SetOutput(&noLog{})
	}
	if ui {
		cfg, err := loadCapyxBench(path)
		if err != nil {
			log.Fatal(err)
		}
		return cfg
	}
	runCapyTestSuite(path)
	return nil
}

// loadCapyxBench compiles a .capyx app into the interactive Test Bench page and
// loads it as a runnable window app.
func loadCapyxBench(scriptPath string) (*domain.AppConfig, error) {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}
	files, err := infra.CompileCapyxBench(string(src),
		string(assets.CapyxRuntimeJS), string(assets.CapyxTestkitJS), string(assets.CapyxTestbenchJS))
	if err != nil {
		return nil, fmt.Errorf("build test bench for %s: %w", scriptPath, err)
	}
	dir, err := materialize(files)
	if err != nil {
		return nil, err
	}
	return LoadApp(filepath.Join(dir, "window.yaml"), false)
}

// runCapyTestSuite parses a .capytest file, runs every scenario headlessly, and
// exits non-zero if any scenario fails.
func runCapyTestSuite(testPath string) {
	src, err := os.ReadFile(testPath)
	if err != nil {
		log.Fatal(err)
	}
	suite, err := infra.ParseCapyTest(string(src))
	if err != nil {
		log.Fatalf("%s: %v", testPath, err)
	}
	capyxPath := suite.Use
	if capyxPath == "" {
		log.Fatalf("%s: suite has no `use <app.capyx>`", testPath)
	}
	if !filepath.IsAbs(capyxPath) {
		capyxPath = filepath.Join(filepath.Dir(testPath), capyxPath)
	}
	capyxSrc, err := os.ReadFile(capyxPath)
	if err != nil {
		log.Fatalf("read app under test %s: %v", capyxPath, err)
	}
	res, err := infra.RunCapyTest(suite, string(capyxSrc),
		string(assets.CapyxRuntimeJS), string(assets.CapyxTestkitJS), string(assets.CapyxDOMShimJS))
	if err != nil {
		log.Fatal(err)
	}
	printSuiteResult(res)
	if res.Failed > 0 {
		os.Exit(1)
	}
}

// printSuiteResult renders a human-readable pass/fail report to stdout.
func printSuiteResult(res *infra.SuiteResult) {
	if res.Name != "" {
		fmt.Printf("\n  %s\n", res.Name)
	}
	for _, sc := range res.Results {
		mark := "✓"
		if !sc.Ok {
			mark = "✗"
		}
		fmt.Printf("  %s %s\n", mark, sc.Name)
		if !sc.Ok {
			for _, st := range sc.Steps {
				if !st.Ok {
					fmt.Printf("      ✗ %s: %s\n", st.Op, st.Message)
				}
			}
			if sc.Error != "" {
				fmt.Printf("      ! %s\n", sc.Error)
			}
		}
	}
	fmt.Printf("\n  %d passed, %d failed\n\n", res.Passed, res.Failed)
}

// materialize writes a generated file map to a fresh temp dir and returns it.
func materialize(files map[string]string) (string, error) {
	dir, err := os.MkdirTemp("", "capyx_*")
	if err != nil {
		return "", err
	}
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return "", err
		}
	}
	return dir, nil
}

type noLog struct{}

func (*noLog) Write(p []byte) (int, error) { return len(p), nil }
