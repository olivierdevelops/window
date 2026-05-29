package domain

type RunMode string

const (
	ModeServer     RunMode = ""
	ModeBrowser    RunMode = "browser"
	ModeProxy      RunMode = "proxy"
	ModeURL        RunMode = "url"
	ModeControlled RunMode = "controlled"
	ModeWASM       RunMode = "wasm"
)

type NativeFeature string

const (
	NativeFS      NativeFeature = "fs"
	NativeOS      NativeFeature = "os"
	NativeDialogs NativeFeature = "dialogs"
	NativeCanvas  NativeFeature = "canvas"
	NativeCamera  NativeFeature = "camera"
	NativeMic     NativeFeature = "mic"
	NativeSpeech  NativeFeature = "speech"
	NativeScreen  NativeFeature = "screen"
	NativeInput   NativeFeature = "input"
)

type WindowSize struct {
	Width  int `yaml:"width" json:"width"`
	Height int `yaml:"height" json:"height"`
}

type AppConfig struct {
	Title            string            `yaml:"title" json:"title"`
	DebugMode        bool              `yaml:"debug_mode" json:"debug_mode"`
	Mode             RunMode           `yaml:"mode" json:"mode"`
	HTML             string            `yaml:"html" json:"html"`
	EntryPath        string            `yaml:"entry_path" json:"entry_path"`
	Dirs             map[string]string `yaml:"static_dirs" json:"static_dirs"`
	Files            map[string]string `yaml:"files" json:"files"`
	URL              string            `yaml:"url" json:"url"`
	Size             *WindowSize       `yaml:"size" json:"size"`
	RunBackendScript string            `yaml:"run_backend_script" json:"run_backend_script"`
	InjectHTMLPath   string            `yaml:"inject_html" json:"inject_html"`
	ProxyTarget      string            `yaml:"proxy_target" json:"proxy_target"`
	ProxyCommand     string            `yaml:"proxy_command" json:"proxy_command"`
	NativeFeatures   []NativeFeature   `yaml:"native_features" json:"native_features"`
	JSInject         []string          `yaml:"js_inject" json:"js_inject"`
	WASMBackend      string            `yaml:"wasm_backend" json:"wasm_backend"`
	ControlledScript string            `yaml:"controlled_script" json:"controlled_script"`
	MacApp           MacAppConfig      `yaml:"mac_app" json:"mac_app"`
}

type MacAppConfig struct {
	Icon          string            `yaml:"icon" json:"icon"`
	ExtraBinaries []string          `yaml:"extra_binaries" json:"extra_binaries"`
	Files         []string          `yaml:"files" json:"files"`
	Dirs          []string          `yaml:"dirs" json:"dirs"`
	Env           map[string]string `yaml:"env" json:"env"`
}
