//go:build windows

package ai

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps child CLI processes (opencode, a bun runtime) from
// flashing visible console windows. CREATE_NO_WINDOW is deliberate: the
// child gets no console at all, so grandchildren it spawns (subagents,
// MCP servers, helpers) inherit "no console" instead of allocating their
// own visible one. Temp-file stdio keeps working untouched.
func hideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags = 0x08000000 // CREATE_NO_WINDOW
}
