// Package alert plays audible cues through the system speakers on Windows
// (winmm); other platforms are silent (toast notification still fires).
// Overtime timers use the short chime (Play); prayer-time alerts use only
// the dedicated azan sound (PlayAzan) — never the chime.
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
	return encodeWAV(samples)
}

// azanDuration is how long the synthesized azan alert lasts; callers keep
// the WAV bytes alive at least this long for async playback.
const azanDuration = 4200 * time.Millisecond

// azanTone synthesizes the dedicated prayer alert: a solemn, slow low-pitch
// call-like sequence (G4 → A4 → C5 → G4, sustained) as 16-bit mono WAV.
// It is deliberately distinct from the short overtime chime. To use a real
// recorded azan instead, drop an `azan.wav` next to the exe — PlayAzan on
// Windows prefers that file when present and only falls back to this tone.
func azanTone() []byte {
	notes := []struct {
		freq float64
		dur  time.Duration
	}{
		{392.0, 800 * time.Millisecond},
		{440.0, 800 * time.Millisecond},
		{523.25, 1000 * time.Millisecond},
		{392.0, 1000 * time.Millisecond},
	}
	var samples []int16
	rate := float64(sampleRate)
	for _, n := range notes {
		count := int(rate * n.dur.Seconds())
		for i := 0; i < count; i++ {
			t := float64(i) / rate
			// slow reverent attack over 80ms, gentle sustain decay
			attack := math.Min(1, t/0.08)
			env := attack * math.Exp(-0.9*t/n.dur.Seconds())
			v := math.Sin(2 * math.Pi * n.freq * t)
			// warm with octave below + soft fifth above
			v = 0.65*v + 0.25*math.Sin(2*math.Pi*(n.freq/2)*t) + 0.10*math.Sin(2*math.Pi*(n.freq*1.5)*t)
			samples = append(samples, int16(v*env*11000))
		}
		// 120ms breath between phrases
		gap := int(rate * 0.12)
		for i := 0; i < gap; i++ {
			samples = append(samples, 0)
		}
	}
	return encodeWAV(samples)
}

func encodeWAV(samples []int16) []byte {
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
