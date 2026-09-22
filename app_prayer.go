package main

import (
	"fmt"
	"strings"
	"time"

	"goals/internal/alert"
	"goals/internal/prayer"

	"github.com/gen2brain/beeep"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------- Prayer-time alerts (opt-in, default OFF) ----------

// ensurePrayerReminder lazily creates the scheduler and wires alerts/events.
func (a *App) ensurePrayerReminder() *prayer.Reminder {
	if a.prayerReminder != nil {
		return a.prayerReminder
	}
	rem := prayer.New(a.store)
	rem.SetOnDue(func(ev prayer.DueEvent) {
		a.notifyPrayerDue(ev)
	})
	a.prayerReminder = rem
	return rem
}

// syncPrayerReminder starts/stops the loop to match the persisted flag.
// Called on startup and after database swaps.
func (a *App) syncPrayerReminder() {
	if a.store == nil {
		return
	}
	rem := a.ensurePrayerReminder()
	rem.SetStore(a.store)
	if prayer.LoadSettings(a.store).Enabled {
		rem.Start()
	} else {
		rem.Stop()
	}
}

// GetPrayerSettings returns the persisted prayer configuration.
func (a *App) GetPrayerSettings() (prayer.Settings, error) {
	if a.store == nil {
		return prayer.DefaultSettings(), nil
	}
	return prayer.LoadSettings(a.store), nil
}

// PrayerSettingsInput mirrors prayer.Settings for the Wails bridge.
type PrayerSettingsInput struct {
	Enabled   bool    `json:"enabled"`
	Method    string  `json:"method"`
	AsrHanafi bool    `json:"asrHanafi"`
	City      string  `json:"city"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	TZ        string  `json:"tz"`
	Clock12h  bool    `json:"clock12h"`
}

// SetPrayerSettings saves the prayer configuration and starts/stops alerts.
func (a *App) SetPrayerSettings(in PrayerSettingsInput) error {
	if a.store == nil {
		return fmt.Errorf("database is not open")
	}
	st := prayer.Settings{
		Enabled: in.Enabled, Method: in.Method, AsrHanafi: in.AsrHanafi,
		City: in.City, Lat: in.Lat, Lng: in.Lng, TZ: in.TZ, Clock12h: in.Clock12h,
	}
	if err := prayer.SaveSettings(a.store, st); err != nil {
		return err
	}
	a.syncPrayerReminder()
	return nil
}

// ListPrayerCities returns the built-in selectable locations.
func (a *App) ListPrayerCities() ([]prayer.City, error) {
	out := append([]prayer.City{}, prayer.Cities...)
	return out, nil
}

// PrayerDayTime is one row of the daily timetable.
type PrayerDayTime struct {
	Key      string `json:"key"`
	Time     string `json:"time"`     // HH:MM in the location's zone
	DateTime string `json:"dateTime"` // RFC3339
}

// GetPrayerTimes returns the six times for date (YYYY-MM-DD, location day;
// empty = today) at the configured location.
func (a *App) GetPrayerTimes(date string) ([]PrayerDayTime, error) {
	st := prayer.DefaultSettings()
	if a.store != nil {
		st = prayer.LoadSettings(a.store)
	}
	at := time.Now()
	if len(date) >= 10 {
		if d, err := time.Parse("2006-01-02", date[:10]); err == nil {
			loc := prayer.ResolveLocation(st.TZ)
			at = time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, loc)
		}
	}
	times := prayer.ComputeTimes(at, st.Lat, st.Lng, st.TZ, st.Method, st.AsrHanafi).Times
	out := []PrayerDayTime{}
	for _, k := range prayer.OrderedKeys {
		if t, ok := times[k]; ok {
			out = append(out, PrayerDayTime{Key: k, Time: t.Format("15:04"), DateTime: t.Format(time.RFC3339)})
		}
	}
	return out, nil
}

// GetPrayerTimesFor returns the six times for explicit coordinates — used by
// the settings preview so it follows the unsaved draft, not saved settings.
func (a *App) GetPrayerTimesFor(lat, lng float64, tz, method string, asrHanafi bool, date string) ([]PrayerDayTime, error) {
	at := time.Now()
	if len(date) >= 10 {
		if d, err := time.Parse("2006-01-02", date[:10]); err == nil {
			loc := prayer.ResolveLocation(tz)
			at = time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, loc)
		}
	}
	times := prayer.ComputeTimes(at, lat, lng, tz, method, asrHanafi).Times
	out := []PrayerDayTime{}
	for _, k := range prayer.OrderedKeys {
		if t, ok := times[k]; ok {
			out = append(out, PrayerDayTime{Key: k, Time: t.Format("15:04"), DateTime: t.Format(time.RFC3339)})
		}
	}
	return out, nil
}

// GetPrayerStatus returns today's timetable plus the next upcoming prayer.
func (a *App) GetPrayerStatus() (prayer.PrayerStatus, error) {
	return prayer.Status(a.store, time.Now()), nil
}

// PrayerSnooze re-rings the current prayer after minutes ("wait a minute…").
func (a *App) PrayerSnooze(minutes int) error {
	if a.prayerReminder == nil {
		return fmt.Errorf("prayer reminders are not running")
	}
	a.prayerReminder.Snooze(minutes)
	return nil
}

// PrayerGoing marks the current prayer as handled ("I'm going to pray").
func (a *App) PrayerGoing() error {
	if a.prayerReminder == nil {
		return fmt.Errorf("prayer reminders are not running")
	}
	a.prayerReminder.Going()
	return nil
}

// notifyPrayerDue fires one OS toast + azan sound + a frontend modal event.
// The sound is just the azan (never the generic chime). The modal carries a
// hadith about prayer from the offline collection (Arabic/English per app
// language) — instant, no AI round-trip. The main window is never forced
// open; re-nags keep the alert coming back until the user goes to pray.
func (a *App) notifyPrayerDue(ev prayer.DueEvent) {
	cityName := ev.City
	if c := prayer.FindCity(ev.City); c != nil {
		cityName = c.Name
	} else if ev.City == "custom" {
		cityName = "Custom location"
	}
	clock12, arabic := true, false
	lang := "en"
	if a.store != nil {
		st := prayer.LoadSettings(a.store)
		clock12 = st.Clock12h
		if m, err := a.store.GetSettings(); err == nil {
			if v := strings.TrimSpace(m["app.language"]); strings.ToLower(v) == "ar" {
				arabic = true
				lang = "ar"
			}
		}
	}
	var title, msg string
	clock := prayer.FormatClock(ev.Time, clock12, arabic)
	if arabic {
		title = fmt.Sprintf("Goals — حان وقت الصلاة (%s)", cityName)
		msg = fmt.Sprintf("حان وقت صلاة %s (%s). حي على الصلاة.", ev.Key, clock)
	} else {
		title = fmt.Sprintf("Goals — prayer time (%s)", cityName)
		msg = fmt.Sprintf("It is time for %s prayer (%s).", ev.Key, clock)
	}
	go func() {
		_ = beeep.Notify(title, msg, "")
		alert.PlayAzan()
	}()
	hadith, hadithSource := prayer.HadithFor(ev.Key, lang, ev.Time)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "prayer:due", map[string]any{
			"key": ev.Key, "city": ev.City, "cityName": cityName,
			"time": ev.Time.Format("15:04"), "time12": prayer.FormatClock(ev.Time, true, arabic),
			"clock12h": clock12, "dateTime": ev.Time.Format(time.RFC3339),
			"lang": lang, "hadith": hadith, "hadithSource": hadithSource, "repeat": ev.Repeat,
		})
	}
}
