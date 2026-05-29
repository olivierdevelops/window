//go:build !darwin

package infra

import "fmt"

// BuildMacApp is not supported on non-macOS platforms.
func BuildMacApp(_ string) error {
	return fmt.Errorf("--mac-app is only supported on macOS")
}
