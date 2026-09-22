//go:build windows

package activity

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetForegroundWindow      = modUser32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = modUser32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessID = modUser32.NewProc("GetWindowThreadProcessId")
	procGetLastInputInfo         = modUser32.NewProc("GetLastInputInfo")
	procGetTickCount64           = modKernel32.NewProc("GetTickCount64")
	procOpenProcess              = modKernel32.NewProc("OpenProcess")
	procQueryFullProcessImage    = modKernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
)

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

// Capture reads the current foreground window + owning process + idle time.
// Best-effort: any failure yields a usable zero-ish snapshot, never an error.
func Capture() Snapshot {
	snap := Snapshot{IdleSecs: -1, HasWindow: false}

	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		snap.IdleSecs = idleSeconds()
		return snap
	}

	// Window title.
	var buf [512]uint16
	n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	title := ""
	if n > 0 {
		title = syscall.UTF16ToString(buf[:n])
	}

	// Owning PID.
	var pid uint32
	_, _, _ = procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

	exe := ""
	if pid != 0 {
		exe = exeBaseOfPID(pid)
	}

	snap.Title = strings.TrimSpace(title)
	snap.Exe = strings.ToLower(strings.TrimSpace(exe))
	snap.IdleSecs = idleSeconds()
	snap.HasWindow = true
	return snap
}

func idleSeconds() int64 {
	var info lastInputInfo
	info.cbSize = uint32(unsafe.Sizeof(info))
	r, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return -1
	}
	now, _, _ := procGetTickCount64.Call()
	// Both counters are ms since boot (wrapping is harmless for a delta).
	diff := int64(now) - int64(info.dwTime)
	if diff < 0 {
		diff = 0
	}
	return diff / 1000
}

const processQueryLimitedInformation = 0x1000

func exeBaseOfPID(pid uint32) string {
	h, _, _ := procOpenProcess.Call(uintptr(processQueryLimitedInformation), 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandle.Call(h)
	var buf [512]uint16
	size := uint32(len(buf))
	r, _, _ := procQueryFullProcessImage.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 || size == 0 || int(size) > len(buf) {
		return ""
	}
	full := syscall.UTF16ToString(buf[:size])
	return filepath.Base(full)
}
