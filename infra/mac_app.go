//go:build darwin

package infra

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"webview_gui/domain"
)

// BuildMacApp bundles the running binary into a macOS .app for the given config.
func BuildMacApp(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	var cfg domain.AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}
	if cfg.Title == "" {
		cfg.Title = "App"
	}

	baseDir := filepath.Dir(configPath)
	appName := cfg.Title
	appBundle := appName + ".app"
	macOS := filepath.Join(appBundle, "Contents", "MacOS")
	resources := filepath.Join(appBundle, "Contents", "Resources")

	if err := os.RemoveAll(appBundle); err != nil {
		return err
	}
	for _, dir := range []string{macOS, resources} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	fmt.Println("→ copying binary...")
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}
	binaryDst := filepath.Join(macOS, "window")
	if err := CopyFile(self, binaryDst); err != nil {
		return err
	}
	if err := os.Chmod(binaryDst, 0755); err != nil {
		return err
	}

	fmt.Printf("→ copying config %s\n", filepath.Base(configPath))
	if err := CopyFile(configPath, filepath.Join(macOS, filepath.Base(configPath))); err != nil {
		return err
	}

	for _, bin := range cfg.MacApp.ExtraBinaries {
		src := bin
		if !filepath.IsAbs(bin) && !strings.Contains(bin, "/") {
			src, err = exec.LookPath(bin)
			if err != nil {
				return fmt.Errorf("binary %q not found on PATH: %w", bin, err)
			}
		}
		dst := filepath.Join(macOS, filepath.Base(bin))
		fmt.Printf("→ copying binary %s\n", filepath.Base(bin))
		if err := CopyFile(src, dst); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0755); err != nil {
			return err
		}
	}

	for _, f := range cfg.MacApp.Files {
		src := filepath.Join(baseDir, f)
		dst := filepath.Join(macOS, f)
		fmt.Printf("→ copying file %s\n", f)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if err := CopyFile(src, dst); err != nil {
			return err
		}
	}

	for _, d := range cfg.MacApp.Dirs {
		src := filepath.Join(baseDir, d)
		dst := filepath.Join(macOS, d)
		fmt.Printf("→ copying dir %s\n", d)
		if err := CopyDir(src, dst); err != nil {
			return err
		}
	}

	if cfg.MacApp.Icon != "" {
		src := filepath.Join(baseDir, cfg.MacApp.Icon)
		dst := filepath.Join(resources, filepath.Base(cfg.MacApp.Icon))
		fmt.Printf("→ copying icon %s\n", cfg.MacApp.Icon)
		if err := CopyFile(src, dst); err != nil {
			return err
		}
	}

	fmt.Println("→ writing Info.plist...")
	bundleID := "com.window." + strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
	iconKey := ""
	if cfg.MacApp.Icon != "" {
		iconKey = fmt.Sprintf(`
    <key>CFBundleIconFile</key>
    <string>%s</string>`, strings.TrimSuffix(filepath.Base(cfg.MacApp.Icon), ".icns"))
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>%s</string>
    <key>CFBundleDisplayName</key>
    <string>%s</string>
    <key>CFBundleIdentifier</key>
    <string>%s</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleExecutable</key>
    <string>launch</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>NSHighResolutionCapable</key>
    <true/>%s
</dict>
</plist>`, appName, appName, bundleID, iconKey)

	if err := os.WriteFile(filepath.Join(appBundle, "Contents", "Info.plist"), []byte(plist), 0644); err != nil {
		return err
	}

	fmt.Println("→ writing launcher...")
	var envLines strings.Builder
	for k, v := range cfg.MacApp.Env {
		envLines.WriteString(fmt.Sprintf("export %s=%q\n", k, v))
	}
	launcher := fmt.Sprintf(`#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"
export PATH="$DIR:$PATH"
%s
exec "$DIR/window" %s
`, envLines.String(), filepath.Base(configPath))

	launchPath := filepath.Join(macOS, "launch")
	if err := os.WriteFile(launchPath, []byte(launcher), 0755); err != nil {
		return err
	}

	fmt.Printf("\n✓ built %s\n", appBundle)
	fmt.Printf("  open %s\n", appBundle)
	return nil
}
