package orchfeatures

import (
	"webview_gui/features"
	infranative "webview_gui/infra/native"
)

// MakeNativeDialogs builds a NativeDialogs backed by native OS dialog boxes.
func MakeNativeDialogs() features.NativeDialogs {
	return features.NativeDialogs{
		OpenFile: func(opts features.FileDialogOptions) ([]string, error) {
			return infranative.OpenFile(infranative.FileDialogOpts{
				Title:   opts.Title,
				Filters: opts.Filters,
				Multi:   opts.Multi,
			})
		},
		SaveFile: func(opts features.FileDialogOptions) (string, error) {
			return infranative.SaveFile(infranative.FileDialogOpts{
				Title:   opts.Title,
				Filters: opts.Filters,
			})
		},
		ShowMessage: func(opts features.MessageDialogOptions) error {
			return infranative.ShowMessage(infranative.MessageDialogOpts{
				Title:   opts.Title,
				Message: opts.Message,
				Kind:    opts.Kind,
			})
		},
	}
}
