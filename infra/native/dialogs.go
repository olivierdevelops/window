package native

import "github.com/ncruces/zenity"

type FileDialogOpts struct {
	Title   string
	Filters []string
	Multi   bool
}

type MessageDialogOpts struct {
	Title   string
	Message string
	Kind    string // "info" | "warn" | "error"
}

func OpenFile(opts FileDialogOpts) ([]string, error) {
	zopts := []zenity.Option{zenity.Title(opts.Title)}
	if opts.Multi {
		return zenity.SelectFileMultiple(zopts...)
	}
	f, err := zenity.SelectFile(zopts...)
	if err != nil {
		return nil, err
	}
	return []string{f}, nil
}

func SaveFile(opts FileDialogOpts) (string, error) {
	return zenity.SelectFileSave(zenity.Title(opts.Title))
}

func ShowMessage(opts MessageDialogOpts) error {
	switch opts.Kind {
	case "warn":
		return zenity.Warning(opts.Message, zenity.Title(opts.Title))
	case "error":
		return zenity.Error(opts.Message, zenity.Title(opts.Title))
	default:
		return zenity.Info(opts.Message, zenity.Title(opts.Title))
	}
}
