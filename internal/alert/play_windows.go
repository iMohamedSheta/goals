//go:build windows

package alert

import (
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
