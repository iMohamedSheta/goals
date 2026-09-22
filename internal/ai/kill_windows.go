//go:build windows

package ai

import (
	"os/exec"
	"strconv"
)

// killTree force-kills a process and its whole child tree (taskkill /T).
// Best-effort: used after cancel/timeout, which CommandContext alone can't
// cover — it only terminates the direct child, leaving grandchildren
// (e.g. goals.exe mcp) orphaned.
func killTree(pid int) {
	if pid <= 0 {
		return
	}
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	hideConsole(cmd)
	_ = cmd.Run()
}
