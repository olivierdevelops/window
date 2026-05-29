module webview_gui

go 1.24.1 //.12

//1.24.1

require (
	github.com/fsnotify/fsnotify v1.7.0
	github.com/google/uuid v1.6.0
	github.com/luowensheng/capy v0.20.0
	github.com/ncruces/zenity v0.10.14
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/tetratelabs/wazero v1.7.0
	github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6
	gopkg.in/yaml.v3 v3.0.1
)

// Use the local checkout for the latest Capy (ahead of the tagged release).
replace github.com/luowensheng/capy => /Users/oliverlaleau/Documents/projects/capylang-claude

require (
	github.com/akavel/rsrc v0.10.2 // indirect
	github.com/dchest/jsmin v0.0.0-20220218165748-59f39799265f // indirect
	github.com/josephspurrier/goversioninfo v1.4.1 // indirect
	github.com/randall77/makefat v0.0.0-20210315173500-7ddd0e42c844 // indirect
	golang.org/x/image v0.20.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
)
