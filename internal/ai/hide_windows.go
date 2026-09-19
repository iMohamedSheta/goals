//go:build windows

package ai

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps child CLI processes (opencode) from flashing a visible
// console window. SW_HIDE (not CREATE_NO_WINDOW) is deliberate: grandchildren
// the child spawns (e.g. MCP servers) inherit the same hidden console instead
// of allocating their own visible one. Stdio pipes keep working untouched.
func hideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
