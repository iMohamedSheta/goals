//go:build !windows

package activity

// Capture on non-Windows builds: no foreground API is wired, so tracking
// stays idle instead of crashing. Windows is the supported platform.
func Capture() Snapshot {
	return Snapshot{Exe: "", Title: "", IdleSecs: -1, HasWindow: false}
}
