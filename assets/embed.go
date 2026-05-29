package assets

import _ "embed"

//go:embed backend.js
var BackendJS []byte

//go:embed browser.js
var BrowserJS []byte

//go:embed window_default.yaml
var WindowDefaultYAML []byte

//go:embed default_index.html
var IndexHTML []byte

//go:embed native_fs.js
var NativeFSJS []byte

//go:embed native_os.js
var NativeOSJS []byte

//go:embed native_dialogs.js
var NativeDialogsJS []byte

//go:embed native_canvas.js
var NativeCanvasJS []byte
