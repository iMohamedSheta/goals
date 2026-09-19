//go:build !windows

package ai

import "os/exec"

// hideConsole is a no-op outside Windows (no console windows exist there).
func hideConsole(cmd *exec.Cmd) {}
