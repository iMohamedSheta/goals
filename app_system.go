package main

import (
	"fmt"
	"strings"
	"time"

	"goals/internal/activity"
	"goals/internal/alert"
	"goals/internal/autostart"
	"goals/internal/store"

	"github.com/gen2brain/beeep"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------- OS startup (default OFF) ----------

// GetAutostartEnabled reports whether Goals launches when the OS starts.
// Default is false — nothing is registered unless the user enables it.
func (a *App) GetAutostartEnabled() (bool, error) {
	return autostart.IsEnabled()
}

// SetAutostartEnabled turns "run at OS startup" on or off.
func (a *App) SetAutostartEnabled(on bool) error {
	return autostart.SetEnabled(on)
}

// ---------- App language (persisted, surfaced in Settings → General) ----------

// GetAppLanguage returns the saved UI language ("ar" | "en", default "ar").
func (a *App) GetAppLanguage() (string, error) {
	if a.store == nil {
		return "ar", nil
	}
	m, err := a.store.GetSettings()
	if err != nil {
		return "ar", err
	}
	v := strings.ToLower(strings.TrimSpace(m["app.language"]))
	if v == "en" {
		return "en", nil
	}
	return "ar", nil
}

// SetAppLanguage persists the UI language ("ar" | "en").
func (a *App) SetAppLanguage(lang string) error {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang != "en" {
		lang = "ar"
	}
	if a.store == nil {
		return nil
	}
	return a.store.SetSetting("app.language", lang)
}

// ---------- App-usage tracking (opt-in, default OFF) ----------

// ensureActivityTracker lazily creates the tracker and wires alerts/events.
func (a *App) ensureActivityTracker() *activity.Tracker {
	if a.activityTracker != nil {
		return a.activityTracker
	}
	tr := activity.New(a.store)
	tr.SetOnTick(func(st activity.CurrentStatus) {
		if a.ctx == nil {
			return
		}
		runtime.EventsEmit(a.ctx, "activity:tick", st)
	})
	tr.SetOnDistraction(func(info activity.DistractionInfo) {
		a.notifyDistraction(info)
	})
	a.activityTracker = tr
	return tr
}

// syncActivityTracker starts/stops the loop to match the persisted flag and
// prunes history older than the retention policy (runs even when tracking is
// off, so the database never stacks up). Called on startup and after swaps.
func (a *App) syncActivityTracker() {
	if a.store == nil {
		return
	}
	tr := a.ensureActivityTracker()
	tr.SetStore(a.store)
	_, _ = activity.PruneByPolicy(a.store)
	if activity.IsEnabledSetting(a.store) {
		tr.Start()
	} else {
		tr.Stop()
	}
}

// GetActivityEnabled reports whether usage recording is ON (default OFF).
func (a *App) GetActivityEnabled() (bool, error) {
	if a.store == nil {
		return false, nil
	}
	return activity.IsEnabledSetting(a.store), nil
}

// SetActivityEnabled turns usage recording on/off and starts/stops the tracker.
func (a *App) SetActivityEnabled(on bool) error {
	if a.store == nil {
		return fmt.Errorf("database is not open")
	}
	val := "0"
	if on {
		val = "1"
	}
	if err := a.store.SetSetting("activity.enabled", val); err != nil {
		return err
	}
	tr := a.ensureActivityTracker()
	tr.SetStore(a.store)
	if on {
		tr.Start()
	} else {
		tr.Stop()
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "activity:enabled", on)
	}
	return nil
}

// GetActivityStatus returns the live foreground run ("what am I doing now").
func (a *App) GetActivityStatus() (activity.CurrentStatus, error) {
	if a.activityTracker == nil {
		return activity.CurrentStatus{Enabled: false}, nil
	}
	return a.activityTracker.Current(), nil
}

// GetActivitySummary aggregates usage for range=day|week|month|overall and
// refDay=YYYY-MM-DD. The open (unflushed) run is folded into day totals so
// "today" ticks live.
func (a *App) GetActivitySummary(rng string, refDay string) (store.ActivitySummary, error) {
	if a.store == nil {
		return store.ActivitySummary{}, fmt.Errorf("database is not open")
	}
	live := store.ActivitySegment{}
	if a.activityTracker != nil {
		live = a.activityTracker.LiveExtra()
	}
	if strings.TrimSpace(refDay) == "" {
		refDay = time.Now().UTC().Format("2006-01-02")
	}
	return a.store.GetActivitySummary(rng, refDay, live)
}

// ListActivitySegments returns the "everything I did" timeline for a range,
// optionally narrowed to one app (empty = all apps).
func (a *App) ListActivitySegments(rng string, refDay string, appFilter string, limit int) ([]store.ActivitySegment, error) {
	if a.store == nil {
		return nil, fmt.Errorf("database is not open")
	}
	if strings.TrimSpace(refDay) == "" {
		refDay = time.Now().UTC().Format("2006-01-02")
	}
	out, err := a.store.ListActivitySegments(rng, refDay, appFilter, limit)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []store.ActivitySegment{}
	}
	return out, nil
}

// GetActivityRetention returns the prune policy in days (default 90,
// 0 = keep forever).
func (a *App) GetActivityRetention() (int, error) {
	if a.store == nil {
		return activity.DefaultRetentionDays, nil
	}
	return activity.RetentionDays(a.store), nil
}

// SetActivityRetention saves the prune policy in days (0 = keep forever,
// clamped to 0..3650) and immediately deletes anything older than it.
func (a *App) SetActivityRetention(days int) error {
	if a.store == nil {
		return fmt.Errorf("database is not open")
	}
	if days < 0 {
		days = 0
	}
	if days > 3650 {
		days = 3650
	}
	if err := a.store.SetSetting("activity.retention_days", fmt.Sprint(days)); err != nil {
		return err
	}
	_, _ = activity.PruneByPolicy(a.store)
	return nil
}

// PruneActivityNow deletes history older than the retention policy right away.
// Returns the removed segment count.
func (a *App) PruneActivityNow() (int64, error) {
	if a.store == nil {
		return 0, fmt.Errorf("database is not open")
	}
	return activity.PruneByPolicy(a.store)
}

// GetActivityStats reports stored usage volume (segments, span, db size).
func (a *App) GetActivityStats() (store.ActivityStats, error) {
	if a.store == nil {
		return store.ActivityStats{}, fmt.Errorf("database is not open")
	}
	return a.store.ActivityStats()
}

// ClearActivity deletes recorded usage (beforeDay=YYYY-MM-DD keeps newer data,
// empty wipes everything). Returns the removed row count.
func (a *App) ClearActivity(beforeDay string) (int64, error) {
	if a.store == nil {
		return 0, fmt.Errorf("database is not open")
	}
	return a.store.ClearActivity(strings.TrimSpace(beforeDay))
}

// notifyDistraction fires one OS toast + a frontend event when YouTube/social
// (or any distraction) passes the session or daily threshold while tracking.
func (a *App) notifyDistraction(info activity.DistractionInfo) {
	what := info.Detail
	if strings.TrimSpace(what) == "" {
		what = info.App
	}
	var title, msg string
	if info.Kind == "daily" {
		title = "Goals — distraction budget spent"
		msg = fmt.Sprintf("%s\nDistraction today: %s (limit %s). Wasted time keeps counting in Insights.",
			what, formatHMS(info.DaySecs), formatHMS(distractionDailyLimit(a)))
	} else {
		title = "Goals — still on something distracting?"
		msg = fmt.Sprintf("%s\n%v in one go on %s. If this isn't work, switch back — it's counted as wasted time.",
			what, formatHMS(info.SessionSecs), info.App)
	}
	go func() {
		_ = beeep.Notify(title, msg, "")
		alert.Play()
	}()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "activity:distraction", info)
	}
}

func distractionDailyLimit(a *App) int64 {
	if a.store == nil {
		return 1800
	}
	if m, err := a.store.GetSettings(); err == nil {
		if v := strings.TrimSpace(m["activity.daily_threshold"]); v != "" {
			var n int64
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n >= 0 {
				return n
			}
		}
	}
	return 1800
}
