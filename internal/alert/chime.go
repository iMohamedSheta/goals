// Package alert plays a short audible chime when a running timer passes its
// max-time estimate. Windows plays it through the system speakers (winmm);
// other platforms are silent (toast notification still fires).
package alert

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"
)

const sampleRate = 22050

// chimeDuration is how long the synthesized chime lasts; callers keep the
// WAV bytes alive at least this long for async playback.
const chimeDuration = 550 * time.Millisecond

// chime synthesizes a two-tone WAV (A5 then D6, decaying) as 16-bit mono.
func chime() []byte {
	notes := []struct {
		freq float64
		dur  time.Duration
	}{
		{880.0, 160 * time.Millisecond},
		{1174.66, 260 * time.Millisecond},
	}
	var samples []int16
	rate := float64(sampleRate)
	for _, n := range notes {
		count := int(rate * n.dur.Seconds())
		for i := 0; i < count; i++ {
			t := float64(i) / rate
			// linear attack over 12ms, exponential decay afterwards
			attack := math.Min(1, t/0.012)
			env := attack * math.Exp(-2.2*t/n.dur.Seconds())
			v := math.Sin(2 * math.Pi * n.freq * t)
			// soften with a touch of the octave below for a rounder tone
			v = 0.75*v + 0.25*math.Sin(2*math.Pi*(n.freq/2)*t)
			samples = append(samples, int16(v*env*12000))
		}
		// 30ms silence between tones
		gap := int(rate * 0.03)
		for i := 0; i < gap; i++ {
			samples = append(samples, 0)
		}
	}
	buf := new(bytes.Buffer)
	dataLen := len(samples) * 2
	_, _ = buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataLen))
	_, _ = buf.WriteString("WAVEfmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // mono
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	_, _ = buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataLen))
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}
