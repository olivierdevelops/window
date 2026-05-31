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

//go:embed native_camera.js
var NativeCameraJS []byte

//go:embed native_mic.js
var NativeMicJS []byte

//go:embed native_speech.js
var NativeSpeechJS []byte

//go:embed native_screen.js
var NativeScreenJS []byte

//go:embed native_input.js
var NativeInputJS []byte

//go:embed window.capy
var WindowCapyLib []byte

//go:embed htmlx.capy
var HtmlxCapyLib []byte

//go:embed capyscript.capy
var CapyScriptLib []byte

//go:embed capyx_runtime.js
var CapyxRuntimeJS []byte
