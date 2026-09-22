//go:build windows

package alert

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var winmm = syscall.NewLazyDLL("winmm.dll")

var playSoundW = winmm.NewProc("PlaySoundW")

const (
	sndAsync     = 0x0001
	sndNodefault = 0x0002
	sndMemory    = 0x0004
	sndFilename  = 0x00020000
)

// Play emits the overtime chime through the system speakers without blocking.
// The WAV bytes are kept alive for the full chime duration because async
// playback reads them after PlaySoundW returns.
func Play() {
	go func() {
		defer func() { _ = recover() }()
		wav := chime()
		if len(wav) == 0 {
			return
		}
		_, _, _ = playSoundW.Call(
			uintptr(unsafe.Pointer(&wav[0])),
			0,
			uintptr(sndAsync|sndMemory|sndNodefault),
		)
		time.Sleep(chimeDuration + 200*time.Millisecond)
		_ = wav
	}()
}

// PlayAzan is the ONLY sound for prayer-time alerts — just the azan, never
// the overtime chime. When a real `azan.wav` sits next to the exe it is
// played as-is; otherwise a solemn synthesized azan tone is used.
func PlayAzan() {
	go func() {
		defer func() { _ = recover() }()
		if p := azanFile(); p != "" {
			ptr, err := syscall.UTF16PtrFromString(p)
			if err == nil {
				_, _, _ = playSoundW.Call(
					uintptr(unsafe.Pointer(ptr)),
					0,
					uintptr(sndAsync|sndFilename|sndNodefault),
				)
				return
			}
		}
		wav := azanTone()
		if len(wav) == 0 {
			return
		}
		_, _, _ = playSoundW.Call(
			uintptr(unsafe.Pointer(&wav[0])),
			0,
			uintptr(sndAsync|sndMemory|sndNodefault),
		)
		time.Sleep(azanDuration + 200*time.Millisecond)
		_ = wav
	}()
}

// azanFile returns the path to a user-provided azan.wav recording when one
// exists next to the running exe, else "" (fall back to the synth tone).
func azanFile() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	p := filepath.Join(filepath.Dir(exe), "azan.wav")
	if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Size() > 44 {
		return p
	}
	return ""
}
