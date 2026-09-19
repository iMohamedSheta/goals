//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// StageInstall writes a self-deleting updater script that waits for pid to
// exit, swaps pendingPath over the running exe, and relaunches it.
// It returns immediately; the caller must quit the app right after.
func StageInstall(pendingPath string, pid int) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	if err := probeWritable(dir); err != nil {
		return err
	}
	// NOTE: %% escapes are for fmt.Sprintf; the .bat sees single % signs.
	script := fmt.Sprintf("@echo off\r\n"+
		"set \"TARGET=%s\"\r\n"+
		"set \"SOURCE=%s\"\r\n"+
		"set \"PID=%d\"\r\n"+
		"set /a tries=0\r\n"+
		":wait\r\n"+
		"tasklist /FI \"PID eq %%PID%%\" 2>NUL | find \"%%PID%%\" >NUL\r\n"+
		"if errorlevel 1 goto swap\r\n"+
		"set /a tries+=1\r\n"+
		"if %%tries%% GEQ 120 goto swap\r\n"+
		"timeout /t 1 /nobreak >NUL\r\n"+
		"goto wait\r\n"+
		":swap\r\n"+
		"move /Y \"%%SOURCE%%\" \"%%TARGET%%\" >NUL\r\n"+
		"start \"\" \"%%TARGET%%\"\r\n"+
		"del \"%%~f0\"\r\n",
		exe, pendingPath, pid)
	batPath := filepath.Join(os.TempDir(), "goals-update.bat")
	if err := os.WriteFile(batPath, []byte(script), 0644); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/C", batPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start() // detached — survives this process exiting
}
