//go:build !windows

package update

import "fmt"

// StageInstall is only implemented on Windows (cmd-based swap script).
func StageInstall(pendingPath string, pid int) error {
	return fmt.Errorf("self-update install is only supported on Windows")
}
