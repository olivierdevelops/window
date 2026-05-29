package orchusecases

import (
	"fmt"
	"log"
	"os"
	"sync"
	"webview_gui/domain"
	"webview_gui/features"
	"webview_gui/infra"
)

// MakeRunControlledMode returns a function that runs in controlled mode:
// the backend subprocess drives window creation, navigation, and destruction
// via a Unix socket at WINDOW_CONTROL_SOCK_PATH.
func MakeRunControlledMode(win features.Windowing, ctrl features.ControlledMode) func(*domain.AppConfig) {
	return func(cfg *domain.AppConfig) {
		var mu sync.Mutex
		windows := map[string]features.WindowHandle{}
		nextID := 0

		executor := features.WindowCommandExecutor{
			CreateWindow: func(title, url string, width, height int) (string, error) {
				if title == "" {
					title = "Window"
				}
				size := &domain.WindowSize{Width: width, Height: height}
				h, err := win.New(title, size, cfg.DebugMode)
				if err != nil {
					return "", err
				}
				mu.Lock()
				nextID++
				id := fmt.Sprintf("win%d", nextID)
				windows[id] = h
				mu.Unlock()

				go func() {
					if url != "" {
						win.Navigate(h, url)
					}
					win.Run(h)
					win.Destroy(h)
					mu.Lock()
					delete(windows, id)
					mu.Unlock()
				}()
				return id, nil
			},
			DestroyWindow: func(windowID string) error {
				mu.Lock()
				h, ok := windows[windowID]
				mu.Unlock()
				if !ok {
					return fmt.Errorf("window %q not found", windowID)
				}
				win.Destroy(h)
				return nil
			},
			NavigateWindow: func(windowID, url string) error {
				mu.Lock()
				h, ok := windows[windowID]
				mu.Unlock()
				if !ok {
					return fmt.Errorf("window %q not found", windowID)
				}
				win.Navigate(h, url)
				return nil
			},
			EvalWindow: func(windowID, js string) error {
				mu.Lock()
				h, ok := windows[windowID]
				mu.Unlock()
				if !ok {
					return fmt.Errorf("window %q not found", windowID)
				}
				win.Eval(h, js)
				return nil
			},
		}

		if err := ctrl.StartManagementSocket("", executor); err != nil {
			log.Fatal("control socket:", err)
		}

		if cfg.ControlledScript != "" {
			cmd := infra.GetRunScriptCMD(cfg.ControlledScript, nil)
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("WINDOW_CONTROL_SOCK_PATH=%s", os.Getenv("WINDOW_CONTROL_SOCK_PATH")),
			)
			cmd.Stdout = infra.NewWriterWrapper("ctrl", os.Stdout)
			cmd.Stderr = infra.NewWriterWrapper("ctrl", os.Stderr)

			if err := cmd.Start(); err != nil {
				log.Fatal("controlled script:", err)
			}
			go func() {
				if err := cmd.Wait(); err != nil {
					log.Println("controlled script exited:", err)
				}
				os.Exit(0)
			}()
		}

		// Block the main goroutine — windows run in their own goroutines.
		select {}
	}
}
