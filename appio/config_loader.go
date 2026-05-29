package appio

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"webview_gui/domain"
	"webview_gui/infra"
)

// LoadApp reads an AppConfig from a YAML or JSON file and chdirs to its directory.
func LoadApp(path string, isJSON bool) (*domain.AppConfig, error) {
	cfg := &domain.AppConfig{}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if isJSON {
		err = json.Unmarshal(b, cfg)
	} else {
		err = yaml.Unmarshal(b, cfg)
	}
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadAppForContentView creates an AppConfig to view a single file.
func LoadAppForContentView(path string) (*domain.AppConfig, error) {
	cfg := &domain.AppConfig{}
	cfg.Title = filepath.Base(path)
	cfg.Files = map[string]string{}
	urlPath := "/" + filepath.Base(path)
	cfg.Files[urlPath] = path
	cfg.HTML = infra.StringToHTML(urlPath)
	return cfg, nil
}
