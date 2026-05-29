package infra

import "github.com/luowensheng/capy"

// GenerateCapyApp compiles a Capy library and runs a .window source through it,
// returning the generated app files (path -> contents): window.yaml, the
// static/ frontend, and any other file blocks the library declares.
//
// The default Capy host is NoOpHost, so generation cannot touch the
// environment or filesystem — safe to run on untrusted source.
func GenerateCapyApp(librarySrc, scriptSrc string) (map[string]string, error) {
	lib, err := capy.NewLibrary(librarySrc)
	if err != nil {
		return nil, err
	}
	primary, files, err := lib.RunMulti(scriptSrc)
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = map[string]string{}
	}
	// If the library wrote a primary output (no file blocks), fall back to its
	// declared output file name.
	if primary != "" {
		if out := lib.OutputFile(); out != "" {
			files[out] = primary
		}
	}
	return files, nil
}
