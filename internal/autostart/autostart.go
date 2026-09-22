// Package autostart manages "run at Windows startup" via the per-user
// Run registry key (HKCU\...\Run). Default is OFF: nothing is written
// unless the user explicitly enables it in Settings → General.
package autostart

import (
	"os"
)

const appValueName = "Goals"

// ExePath returns the quoted executable path to register.
func ExePath() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe
	}
	return ""
}
