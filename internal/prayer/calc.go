// Package prayer computes daily Islamic prayer times offline (no internet
// needed) and reminds the user when a prayer is due, with snooze and
// "I'm going to pray" actions. Calculation follows the well-known
// pray-times algorithm (solar position + configurable twilight angles).
package prayer

import (
	"fmt"
	"math"
	"time"

	// Embedded IANA database so time.LoadLocation("Africa/Cairo") etc. works
	// on Windows without system zoneinfo.
	_ "time/tzdata"
)

const deg2rad = math.Pi / 180
const rad2deg = 180 / math.Pi

// Prayer keys in chronological order (Sunrise carries no alert).
const (
	Fajr    = "fajr"
	Sunrise = "sunrise"
	Dhuhr   = "dhuhr"
	Asr     = "asr"
	Maghrib = "maghrib"
	Isha    = "isha"
)

// OrderedKeys lists every computed time; AlertKeys lists the ones that ring.
var OrderedKeys = []string{Fajr, Sunrise, Dhuhr, Asr, Maghrib, Isha}
var AlertKeys = []string{Fajr, Dhuhr, Asr, Maghrib, Isha}

// Method holds one calculation convention's twilight angles (degrees).
// IshaMinutes>0 means "Isha is N minutes after Maghrib" (Umm al-Qura style)
// instead of an angle.
type Method struct {
	FajrAngle      float64
	IshaAngle      float64
	IshaMinutes    float64
	MaghribMinutes float64
}

// Methods maps a settings key to its convention.
var Methods = map[string]Method{
	"egypt":   {FajrAngle: 19.5, IshaAngle: 17.5},              // Egyptian General Authority (default)
	"mwl":     {FajrAngle: 18, IshaAngle: 17},                  // Muslim World League
	"isna":    {FajrAngle: 15, IshaAngle: 15},                  // ISNA (North America)
	"makkah":  {FajrAngle: 18.5, IshaMinutes: 90},              // Umm al-Qura, Makkah
	"karachi": {FajrAngle: 18, IshaAngle: 18},                  // Karachi
	"gulf":    {FajrAngle: 19.5, IshaMinutes: 90},              // UAE/Qatar-style (19.5 + 90 min)
	"jafari":  {FajrAngle: 16, IshaAngle: 14},                  // Jafari
}

// ValidMethod falls back to egypt on unknown keys.
func ValidMethod(m string) string {
	if _, ok := Methods[m]; ok {
		return m
	}
	return "egypt"
}

func fixAngle(a float64) float64 {
	a -= 360 * math.Floor(a/360)
	if a < 0 {
		a += 360
	}
	return a
}

func fixHour(h float64) float64 {
	h -= 24 * math.Floor(h/24)
	if h < 0 {
		h += 24
	}
	return h
}

func julianDay(year, month, day int) float64 {
	y, m := year, month
	if m <= 2 {
		y--
		m += 12
	}
	a := math.Floor(float64(y) / 100)
	b := 2 - a + math.Floor(a/4)
	return math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + float64(day) + b - 1524.5
}

type sunPos struct{ decl, eqt float64 }

func sunPosition(jd float64) sunPos {
	d := jd - 2451545.0
	g := fixAngle(357.529 + 0.98560028*d)
	q := fixAngle(280.459 + 0.98564736*d)
	l := fixAngle(q + 1.915*math.Sin(g*deg2rad) + 0.020*math.Sin(2*g*deg2rad))
	e := 23.439 - 0.00000036*d
	ra := math.Atan2(math.Cos(e*deg2rad)*math.Sin(l*deg2rad), math.Cos(l*deg2rad)) * rad2deg / 15
	eqt := q/15 - fixHour(ra)
	decl := math.Asin(math.Sin(e*deg2rad)*math.Sin(l*deg2rad)) * rad2deg
	return sunPos{decl, eqt}
}

// hourAngle returns the hour-angle in hours for a solar depression angle.
func hourAngle(lat, decl, angle float64) float64 {
	num := math.Sin(angle*deg2rad) - math.Sin(lat*deg2rad)*math.Sin(decl*deg2rad)
	den := math.Cos(lat*deg2rad) * math.Cos(decl*deg2rad)
	x := num / den
	if x < -1 {
		x = -1
	}
	if x > 1 {
		x = 1
	}
	return math.Acos(x) * rad2deg / 15
}

// asrHourAngle uses the shadow factor: 1 = Standard, 2 = Hanafi.
// NOTE: the resulting angle is passed to hourAngle as a *positive* value —
// this matches the pray-times convention (G = -arccot(...) fed through the
// negated-sine form), which is NOT the same as the solar altitude.
func asrHourAngle(factor, lat, decl float64) float64 {
	angle := math.Atan(1/(factor+math.Tan(math.Abs(lat-decl)*deg2rad))) * rad2deg
	return hourAngle(lat, decl, angle)
}

// DayTimes holds one day's local prayer times.
type DayTimes struct {
	Times map[string]time.Time `json:"times"`
}

// timeOfDay converts fractional hours to a time on day in loc (rounded).
func timeOfDay(day time.Time, loc *time.Location, frac float64) time.Time {
	frac = fixHour(frac)
	mins := int(math.Floor(frac*60 + 0.5))
	h := (mins / 60) % 24
	m := mins % 60
	y, mo, d := day.Date()
	return time.Date(y, mo, d, h, m, 0, 0, loc)
}

// ComputeTimes calculates the six times for day (any instant that day) at
// lat/lng, using methodKey and the Hanafi Asr factor when asrHanafi is true.
// tzName is an IANA zone (e.g. "Africa/Cairo"); unknown zones fall back to
// the machine's local zone.
func ComputeTimes(day time.Time, lat, lng float64, tzName, methodKey string, asrHanafi bool) DayTimes {
	method := Methods[ValidMethod(methodKey)]
	loc := ResolveLocation(tzName)
	inLoc := day.In(loc)
	y, mo, d := inLoc.Date()

	// UTC offset in hours for that date (handles DST automatically).
	_, offset := time.Date(y, mo, d, 12, 0, 0, 0, loc).Zone()
	tz := float64(offset) / 3600

	jd := julianDay(y, int(mo), d) - lng/360.0
	sp := sunPosition(jd)
	noon := fixHour(12 + tz - lng/15 - sp.eqt)

	factor := 1.0
	if asrHanafi {
		factor = 2.0
	}
	out := map[string]time.Time{}
	out[Fajr] = timeOfDay(inLoc, loc, noon-hourAngle(lat, sp.decl, -method.FajrAngle))
	out[Sunrise] = timeOfDay(inLoc, loc, noon-hourAngle(lat, sp.decl, -0.833))
	out[Dhuhr] = timeOfDay(inLoc, loc, noon)
	out[Asr] = timeOfDay(inLoc, loc, noon+asrHourAngle(factor, lat, sp.decl))
	sunset := noon + hourAngle(lat, sp.decl, -0.833)
	out[Maghrib] = timeOfDay(inLoc, loc, sunset+method.MaghribMinutes/60)
	if method.IshaMinutes > 0 {
		maghribFrac := sunset + method.MaghribMinutes/60
		out[Isha] = timeOfDay(inLoc, loc, maghribFrac+method.IshaMinutes/60)
	} else {
		out[Isha] = timeOfDay(inLoc, loc, noon+hourAngle(lat, sp.decl, -method.IshaAngle))
	}
	return DayTimes{Times: out}
}

// FormatClock renders t as "3:04 PM" when use12h is true, else "15:04".
// Arabic suffixes (ص/م) are used when arabic is true.
func FormatClock(t time.Time, use12h bool, arabic bool) string {
	if !use12h {
		return t.Format("15:04")
	}
	h, m := t.Hour(), t.Minute()
	suffix := "AM"
	if h >= 12 {
		suffix = "PM"
	}
	if arabic {
		if h >= 12 {
			suffix = "م"
		} else {
			suffix = "ص"
		}
	}
	h12 := h % 12
	if h12 == 0 {
		h12 = 12
	}
	return fmt.Sprintf("%d:%02d %s", h12, m, suffix)
}

// ResolveLocation loads an IANA zone, falling back to local time.
func ResolveLocation(tz string) *time.Location {
	if tz == "" {
		return time.Local
	}
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.Local
}
