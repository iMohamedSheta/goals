//go:build windows

package autostart

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// IsEnabled reports whether the Goals Run value exists and points at this
// executable (or goals.exe next to it). Missing value = disabled (default).
func IsEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()
	val, _, err := k.GetStringValue(appValueName)
	if err != nil {
		// No value registered → autostart is OFF (the default).
		return false, nil
	}
	val = strings.Trim(strings.TrimSpace(val), `"`)
	exe, _ := os.Executable()
	if exe == "" {
		return val != "", nil
	}
	base := strings.ToLower(filepath.Base(exe))
	dir := strings.ToLower(filepath.Dir(exe))
	lv := strings.ToLower(val)
	// Accept the exact exe path or goals.exe sitting next to the runner
	// (e.g. dev binary registering the shipped goals.exe).
	return strings.Contains(lv, dir) && strings.Contains(lv, strings.TrimSuffix(base, filepath.Ext(base))) ||
		strings.Contains(lv, dir+"\\goals.exe") ||
		strings.Contains(lv, dir+"/goals.exe"), nil
}

// SetEnabled creates (true) or removes (false) the Run value.
// Enabling registers the goals.exe next to the running binary when present,
// otherwise the current executable.
func SetEnabled(on bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		_ = k.DeleteValue(appValueName)
		return nil
	}
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return err
	}
	// Prefer goals.exe next to the runner (the shipped binary name).
	dir := filepath.Dir(exe)
	candidate := filepath.Join(dir, "goals.exe")
	if _, err := os.Stat(candidate); err == nil {
		exe = candidate
	}
	val := `"` + exe + `"`
	return k.SetStringValue(appValueName, val)
}
