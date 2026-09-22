//go:build !windows

package autostart

import "fmt"

// IsEnabled on non-Windows builds: autostart is unsupported, default off.
func IsEnabled() (bool, error) { return false, nil }

// SetEnabled on non-Windows builds always fails with a clear message.
func SetEnabled(on bool) error {
	if !on {
		return nil
	}
	return fmt.Errorf("autostart is only supported on Windows")
}
