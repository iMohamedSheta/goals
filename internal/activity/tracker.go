// Package activity tracks the foreground app/window over time, classifies
// usage (work/distraction/other/idle) and fires distraction alerts.
// Tracking is opt-in (default OFF).
package activity

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"goals/internal/store"
)

// DistractionInfo describes one fired alert.
type DistractionInfo struct {
	Kind        string `json:"kind"` // "session" | "daily"
	App         string `json:"app"`
	Detail      string `json:"detail"`
	Domain      string `json:"domain"`
	SessionSecs int64  `json:"sessionSeconds"`
	DaySecs     int64  `json:"daySeconds"`
}

// CurrentStatus is the live foreground run (for the "right now" pill).
type CurrentStatus struct {
	App      string `json:"app"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Domain   string `json:"domain"`
	Category string `json:"category"`
	Elapsed  int64  `json:"elapsedSeconds"`
	Enabled  bool   `json:"enabled"`
}

const (
	setEnabledKey  = "activity.enabled"
	setAlertsKey   = "activity.alerts"
	setDailyKey    = "activity.daily_threshold"
	setSessionKey  = "activity.session_threshold"
	setRetentionKey = "activity.retention_days"
	defaultDaily   = int64(1800) // 30 min of distraction/day before the daily alert
	defaultSession = int64(600)  // 10 min continuous distraction before the session alert
	// DefaultRetentionDays bounds the activity history: segments older than
	// this are auto-deleted so the database never stacks up. 0 = keep forever.
	DefaultRetentionDays = 90
	pollEvery      = 3 * time.Second
	idleAfter      = int64(120) // no input for 2 min → "Idle" owns the time
	minSegment     = int64(4)   // shorter runs are polling noise, dropped
)

// RetentionDays reads the persisted prune policy (default 90, 0 = keep all).
func RetentionDays(s *store.Store) int {
	if s == nil {
		return DefaultRetentionDays
	}
	m, err := s.GetSettings()
	if err != nil {
		return DefaultRetentionDays
	}
	v := strings.TrimSpace(m[setRetentionKey])
	if v == "" {
		return DefaultRetentionDays
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return DefaultRetentionDays
	}
	if n < 0 {
		n = 0
	}
	if n > 3650 {
		n = 3650
	}
	return n
}

// PruneByPolicy deletes activity segments older than the retention setting.
// Safe to call any time (also runs when tracking is off, via the App).
func PruneByPolicy(s *store.Store) (int64, error) {
	if s == nil {
		return 0, nil
	}
	if n := RetentionDays(s); n > 0 {
		return s.PruneActivity(n)
	}
	return 0, nil
}

// Tracker polls the foreground window and persists runs as activity segments.
// It is OFF by default and only runs after SetEnabled(true) (Settings toggle).
type Tracker struct {
	store *store.Store

	mu             sync.Mutex
	running        bool
	stop           chan struct{}
	onDistraction  func(DistractionInfo)
	onTick         func(CurrentStatus)
	settingsCache  map[string]string
	settingsLoaded time.Time

	curKey         string
	curApp         string
	curTitle       string
	curDetail      string
	curDomain      string
	curCat         string
	curStart       time.Time
	sessionAlerted bool
	dailyAlertDay  string
}

// New returns a stopped tracker bound to the store.
func New(s *store.Store) *Tracker {
	return &Tracker{store: s, settingsCache: map[string]string{}}
}

// SetOnDistraction registers the alert callback (App wires toast + event).
func (t *Tracker) SetOnDistraction(fn func(DistractionInfo)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onDistraction = fn
}

// SetOnTick registers the live-status callback (App forwards to the frontend).
func (t *Tracker) SetOnTick(fn func(CurrentStatus)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onTick = fn
}

// IsRunning reports whether the poll loop is active.
func (t *Tracker) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// Start begins the poll loop (idempotent).
func (t *Tracker) Start() {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.running = true
	stop := make(chan struct{})
	t.stop = stop
	t.mu.Unlock()

	go t.loop(stop)
}

// Stop ends the loop and flushes the open run (idempotent).
func (t *Tracker) Stop() {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return
	}
	t.running = false
	stop := t.stop
	t.stop = nil
	t.mu.Unlock()

	select {
	case <-stop:
	default:
		close(stop)
	}
	t.mu.Lock()
	t.flushLocked(time.Now().UTC())
	t.mu.Unlock()
}

// Current returns the live run for "right now" displays.
func (t *Tracker) Current() CurrentStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running || t.curKey == "" {
		return CurrentStatus{Enabled: t.running}
	}
	return CurrentStatus{
		App: t.curApp, Title: t.curTitle, Detail: t.curDetail,
		Domain: t.curDomain, Category: t.curCat,
		Elapsed: int64(time.Since(t.curStart).Seconds()),
		Enabled: true,
	}
}

// LiveExtra converts the open run into a store.ActivitySegment fragment so
// day summaries can include unflushed seconds (today ticks live).
func (t *Tracker) LiveExtra() store.ActivitySegment {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running || t.curKey == "" {
		return store.ActivitySegment{}
	}
	el := int64(time.Since(t.curStart).Seconds())
	if el < 0 {
		el = 0
	}
	return store.ActivitySegment{
		App: t.curApp, Title: t.curTitle, Detail: t.curDetail,
		Domain: t.curDomain, Category: t.curCat,
		StartedAt: t.curStart.UTC().Format(time.RFC3339),
		Seconds:   el,
	}
}

func (t *Tracker) loop(stop chan struct{}) {
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	prune := time.NewTicker(time.Hour)
	defer prune.Stop()
	// Drop old history on start so a big backlog never piles up.
	t.pruneByPolicy()
	for {
		select {
		case <-stop:
			return
		case now := <-tick.C:
			t.pollOnce(now.UTC())
		case <-prune.C:
			t.pruneByPolicy()
		}
	}
}

func (t *Tracker) pruneByPolicy() {
	t.mu.Lock()
	s := t.store
	t.mu.Unlock()
	_, _ = PruneByPolicy(s)
}

func (t *Tracker) pollOnce(now time.Time) {
	snap := Capture()

	var app, detail, domain, cat, title string
	title = snap.Title
	if snap.IdleSecs >= 0 && snap.IdleSecs >= idleAfter {
		app, detail, domain, cat = "Idle", "Away from keyboard", "", "idle"
	} else if !snap.HasWindow && strings.TrimSpace(snap.Exe) == "" {
		// Nothing readable (e.g. unsupported platform) — skip this sample
		// so "Unknown" doesn't flood the timeline.
		return
	} else {
		app, detail, domain, cat = Classify(snap.Exe, snap.Title)
	}

	key := app + "\x00" + domain + "\x00" + detail

	t.mu.Lock()
	if t.curKey == "" {
		t.openLocked(key, app, title, detail, domain, cat, now)
		st := t.statusLocked()
		cb := t.onTick
		t.mu.Unlock()
		if cb != nil {
			cb(st)
		}
		return
	}
	if key != t.curKey {
		t.flushLocked(now)
		t.openLocked(key, app, title, detail, domain, cat, now)
		st := t.statusLocked()
		cb := t.onTick
		t.mu.Unlock()
		if cb != nil {
			cb(st)
		}
		return
	}
	// Same run continues — check distraction thresholds.
	elapsed := int64(now.Sub(t.curStart).Seconds())
	dailyThr, sessionThr, alertsOn := t.thresholdsLocked()
	var fire *DistractionInfo
	if cat == "distraction" && alertsOn {
		if !t.sessionAlerted && elapsed >= sessionThr && sessionThr > 0 {
			t.sessionAlerted = true
			daySecs := t.dayDistractionLocked(now) + elapsedContribLocked(t.curStart, now)
			fire = &DistractionInfo{Kind: "session", App: app, Detail: detail, Domain: domain, SessionSecs: elapsed, DaySecs: daySecs}
		} else {
			daySecs := t.dayDistractionLocked(now) + elapsedContribLocked(t.curStart, now)
			today := now.Format("2006-01-02")
			if t.dailyAlertDay != today && dailyThr > 0 && daySecs >= dailyThr {
				t.dailyAlertDay = today
				fire = &DistractionInfo{Kind: "daily", App: app, Detail: detail, Domain: domain, SessionSecs: elapsed, DaySecs: daySecs}
			}
		}
	}
	st := t.statusLocked()
	cbTick := t.onTick
	cbAlert := t.onDistraction
	t.mu.Unlock()

	if fire != nil && cbAlert != nil {
		cbAlert(*fire)
	}
	if cbTick != nil {
		cbTick(st)
	}
}

// openLocked starts a new run (caller holds mu).
func (t *Tracker) openLocked(key, app, title, detail, domain, cat string, now time.Time) {
	t.curKey = key
	t.curApp = app
	t.curTitle = title
	t.curDetail = detail
	t.curDomain = domain
	t.curCat = cat
	t.curStart = now
	t.sessionAlerted = false
}

// flushLocked persists the open run (caller holds mu).
func (t *Tracker) flushLocked(now time.Time) {
	if t.curKey == "" || t.store == nil {
		t.curKey = ""
		return
	}
	secs := int64(now.Sub(t.curStart).Seconds())
	app, title, detail, domain, cat := t.curApp, t.curTitle, t.curDetail, t.curDomain, t.curCat
	start := t.curStart
	t.curKey = ""
	if secs < minSegment {
		return
	}
	end := now.Format(time.RFC3339)
	_, _ = t.store.RecordActivitySegment(app, title, detail, domain, cat, start.Format(time.RFC3339), end, secs)
}

func (t *Tracker) statusLocked() CurrentStatus {
	if t.curKey == "" {
		return CurrentStatus{Enabled: t.running}
	}
	return CurrentStatus{
		App: t.curApp, Title: t.curTitle, Detail: t.curDetail,
		Domain: t.curDomain, Category: t.curCat,
		Elapsed: int64(time.Since(t.curStart).Seconds()),
		Enabled: true,
	}
}

// thresholdsLocked reads alert settings (cached 30s; caller holds mu).
func (t *Tracker) thresholdsLocked() (daily, session int64, alertsOn bool) {
	daily, session, alertsOn = defaultDaily, defaultSession, true
	if t.store == nil {
		return
	}
	if time.Since(t.settingsLoaded) > 30*time.Second || t.settingsCache == nil {
		if m, err := t.store.GetSettings(); err == nil {
			t.settingsCache = m
			t.settingsLoaded = time.Now()
		}
	}
	m := t.settingsCache
	if v, ok := m[setDailyKey]; ok && strings.TrimSpace(v) != "" {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n >= 0 {
			daily = n
		}
	}
	if v, ok := m[setSessionKey]; ok && strings.TrimSpace(v) != "" {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n >= 0 {
			session = n
		}
	}
	if v, ok := m[setAlertsKey]; ok {
		alertsOn = strings.TrimSpace(v) != "0" && strings.ToLower(strings.TrimSpace(v)) != "false"
	}
	// daily==0 or session==0 disables that alert.
	return
}

// dayDistractionLocked sums stored distraction seconds for now's UTC day.
func (t *Tracker) dayDistractionLocked(now time.Time) int64 {
	if t.store == nil {
		return 0
	}
	return t.store.ActivityDayTotal("distraction", now.Format("2006-01-02"))
}

// elapsedContribLocked counts only the part of the open run belonging to
// today's UTC date (a run started yesterday contributes since midnight).
func elapsedContribLocked(start, now time.Time) int64 {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	s := start
	if s.Before(dayStart) {
		s = dayStart
	}
	if now.Before(s) {
		return 0
	}
	return int64(now.Sub(s).Seconds())
}

// SetStore rebinds the tracker after a database swap (import/restore).
func (t *Tracker) SetStore(s *store.Store) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store = s
}

// IsEnabledSetting reads the persisted on/off flag (default OFF).
func IsEnabledSetting(s *store.Store) bool {
	if s == nil {
		return false
	}
	m, err := s.GetSettings()
	if err != nil {
		return false
	}
	v := strings.TrimSpace(m[setEnabledKey])
	return v == "1" || strings.EqualFold(v, "true")
}
