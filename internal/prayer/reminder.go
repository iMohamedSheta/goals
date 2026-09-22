package prayer

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"goals/internal/store"
)

// Settings keys (all OFF/neutral by default; prayer alerts are opt-in).
const (
	setEnabledKey = "prayer.enabled"
	setMethodKey  = "prayer.method"
	setAsrKey     = "prayer.asr_hanafi"
	setCityKey    = "prayer.city"
	setLatKey     = "prayer.lat"
	setLngKey     = "prayer.lng"
	setTZKey      = "prayer.tz"
	setClockKey   = "prayer.clock" // "12" (AM/PM, default) or "24"
)

// Settings is the persisted prayer configuration.
type Settings struct {
	Enabled   bool    `json:"enabled"`
	Method    string  `json:"method"`
	AsrHanafi bool    `json:"asrHanafi"`
	City      string  `json:"city"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	TZ        string  `json:"tz"`
	Clock12h  bool    `json:"clock12h"`
}

// DefaultSettings: alerts off, Egyptian method, Cairo, 12h clock.
func DefaultSettings() Settings {
	c := FindCity("cairo")
	return Settings{Enabled: false, Method: "egypt", AsrHanafi: false, City: "cairo", Lat: c.Lat, Lng: c.Lng, TZ: c.TZ, Clock12h: true}
}

// LoadSettings reads the persisted config (defaults when missing).
func LoadSettings(s *store.Store) Settings {
	out := DefaultSettings()
	if s == nil {
		return out
	}
	m, err := s.GetSettings()
	if err != nil {
		return out
	}
	if v := strings.TrimSpace(m[setEnabledKey]); v == "1" || strings.EqualFold(v, "true") {
		out.Enabled = true
	}
	if v := strings.TrimSpace(m[setMethodKey]); v != "" {
		out.Method = ValidMethod(v)
	}
	if v := strings.TrimSpace(m[setAsrKey]); v == "1" || strings.EqualFold(v, "true") {
		out.AsrHanafi = true
	}
	city := strings.TrimSpace(m[setCityKey])
	if city == "" || city == "custom" {
		out.City = "custom"
		if v := strings.TrimSpace(m[setLatKey]); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f >= -90 && f <= 90 {
				out.Lat = f
			}
		}
		if v := strings.TrimSpace(m[setLngKey]); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f >= -180 && f <= 180 {
				out.Lng = f
			}
		}
		if v := strings.TrimSpace(m[setTZKey]); v != "" {
			out.TZ = v
		}
	} else if c := FindCity(city); c != nil {
		out.City = c.ID
		out.Lat, out.Lng, out.TZ = c.Lat, c.Lng, c.TZ
	}
	// Clock defaults to 12h AM/PM; only an explicit "24" switches to 24h.
	out.Clock12h = true
	if v := strings.TrimSpace(m[setClockKey]); v == "24" {
		out.Clock12h = false
	}
	return out
}

// SaveSettings persists the config (city snapshots coordinates so times keep
// working even if the built-in list ever changes).
func SaveSettings(s *store.Store, st Settings) error {
	st.Method = ValidMethod(st.Method)
	en, asr := "0", "0"
	if st.Enabled {
		en = "1"
	}
	if st.AsrHanafi {
		asr = "1"
	}
	clock := "12"
	if !st.Clock12h {
		clock = "24"
	}
	city := strings.TrimSpace(st.City)
	if city == "" {
		city = "custom"
	}
	if c := FindCity(city); c != nil {
		st.City, st.Lat, st.Lng, st.TZ = c.ID, c.Lat, c.Lng, c.TZ
	} else {
		st.City = "custom"
		if st.TZ == "" {
			st.TZ = "Africa/Cairo"
		}
	}
	kv := map[string]string{
		setEnabledKey: en,
		setMethodKey:  st.Method,
		setAsrKey:     asr,
		setCityKey:    st.City,
		setLatKey:     strconv.FormatFloat(st.Lat, 'f', 5, 64),
		setLngKey:     strconv.FormatFloat(st.Lng, 'f', 5, 64),
		setTZKey:      st.TZ,
		setClockKey:   clock,
	}
	for k, v := range kv {
		if err := s.SetSetting(k, v); err != nil {
			return err
		}
	}
	return nil
}

// DueEvent is fired when a prayer time arrives.
// Repeat is true for persistence re-nags (user hasn't gone to pray yet) —
// callers should replay sound + popup but skip one-shot extras (AI hadith).
type DueEvent struct {
	Key    string    `json:"key"`
	City   string    `json:"city"`
	Time   time.Time `json:"time"`
	Repeat bool      `json:"repeat"`
}

// graceWindow: only ring within 45 minutes after the time (so opening the app
// hours later doesn't nag about a long-past prayer).
const graceWindow = 45 * time.Minute
const pollEvery = 20 * time.Second

// nagEvery: while the user hasn't marked "I'm going to pray", re-ring this
// often so the alert persists over whatever they are doing until they pray.
const nagEvery = 5 * time.Minute

// Reminder polls the clock and fires OnDue when a prayer arrives.
// OFF by default — only runs after the user enables it in Settings.
type Reminder struct {
	store *store.Store

	mu        sync.Mutex
	running   bool
	stop      chan struct{}
	onDue     func(DueEvent)
	alerted   map[string]bool      // "YYYY-MM-DD:key" already rang
	done      map[string]bool      // "YYYY-MM-DD:key" user is going/done
	lastNag   map[string]time.Time // "YYYY-MM-DD:key" last ring (for re-nags)
	snoozeKey string
	snoozeDay string
	snoozeAt  time.Time
	day       string // last seen location-day (resets state at midnight)
}

// New returns a stopped reminder bound to the store.
func New(s *store.Store) *Reminder {
	return &Reminder{store: s, alerted: map[string]bool{}, done: map[string]bool{}, lastNag: map[string]time.Time{}}
}

// SetOnDue registers the alert callback (App wires toast + frontend event).
func (r *Reminder) SetOnDue(fn func(DueEvent)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onDue = fn
}

// SetStore rebinds after a database swap (import/restore).
func (r *Reminder) SetStore(s *store.Store) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store = s
}

// IsRunning reports whether the loop is active.
func (r *Reminder) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Start begins polling (idempotent).
func (r *Reminder) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	stop := make(chan struct{})
	r.stop = stop
	r.mu.Unlock()
	go r.loop(stop)
}

// Stop ends polling (idempotent).
func (r *Reminder) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	stop := r.stop
	r.stop = nil
	r.mu.Unlock()
	select {
	case <-stop:
	default:
		close(stop)
	}
}

func (r *Reminder) loop(stop chan struct{}) {
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	r.check(time.Now())
	for {
		select {
		case <-stop:
			return
		case now := <-tick.C:
			r.check(now)
		}
	}
}

func (r *Reminder) check(now time.Time) {
	r.mu.Lock()
	st := LoadSettings(r.store)
	if !st.Enabled {
		r.mu.Unlock()
		return
	}
	loc := ResolveLocation(st.TZ)
	local := now.In(loc)
	dayKey := local.Format("2006-01-02")
	if r.day != dayKey {
		r.day = dayKey
		r.alerted = map[string]bool{}
		r.done = map[string]bool{}
		r.lastNag = map[string]time.Time{}
	}
	times := ComputeTimes(local, st.Lat, st.Lng, st.TZ, st.Method, st.AsrHanafi).Times
	// Current prayer = latest alert prayer whose time has passed.
	current := ""
	var currentTime time.Time
	for _, k := range AlertKeys {
		if t, ok := times[k]; ok && !local.Before(t) {
			current, currentTime = k, t
		}
	}
	var fire *DueEvent
	if current != "" {
		id := dayKey + ":" + current
		snoozing := r.snoozeKey == current && r.snoozeDay == dayKey && !r.snoozeAt.IsZero()
		if !r.done[id] && local.Sub(currentTime) <= graceWindow && (!snoozing || !local.Before(r.snoozeAt)) {
			if !r.alerted[id] {
				r.alerted[id] = true
				r.snoozeAt = time.Time{}
				r.lastNag[id] = local
				fire = &DueEvent{Key: current, City: st.City, Time: currentTime}
			} else if local.Sub(r.lastNag[id]) >= nagEvery {
				// Persistent nag: user hasn't gone to pray yet — ring again
				// (azan + popup) so the alert survives over their work.
				r.lastNag[id] = local
				fire = &DueEvent{Key: current, City: st.City, Time: currentTime, Repeat: true}
			}
		}
	}
	cb := r.onDue
	r.mu.Unlock()
	if fire != nil && cb != nil {
		cb(*fire)
	}
}

// Snooze re-rings the current prayer after minutes ("wait a minute…").
// It un-marks the alert so the next check past snoozeAt fires again.
func (r *Reminder) Snooze(minutes int) {
	if minutes < 1 {
		minutes = 1
	}
	if minutes > 120 {
		minutes = 120
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	st := LoadSettings(r.store)
	loc := ResolveLocation(st.TZ)
	local := time.Now().In(loc)
	dayKey := local.Format("2006-01-02")
	times := ComputeTimes(local, st.Lat, st.Lng, st.TZ, st.Method, st.AsrHanafi).Times
	current := ""
	for _, k := range AlertKeys {
		if t, ok := times[k]; ok && !local.Before(t) && local.Sub(t) <= graceWindow {
			current = k
		}
	}
	if current == "" {
		return
	}
	r.snoozeKey, r.snoozeDay = current, dayKey
	r.snoozeAt = local.Add(time.Duration(minutes) * time.Minute)
	delete(r.alerted, dayKey+":"+current)
}

// Going marks the current prayer as handled ("I'm going to pray") — no more
// rings for it today.
func (r *Reminder) Going() {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := LoadSettings(r.store)
	loc := ResolveLocation(st.TZ)
	local := time.Now().In(loc)
	dayKey := local.Format("2006-01-02")
	times := ComputeTimes(local, st.Lat, st.Lng, st.TZ, st.Method, st.AsrHanafi).Times
	for _, k := range AlertKeys {
		if t, ok := times[k]; ok && !local.Before(t) && local.Sub(t) <= graceWindow {
			r.done[dayKey+":"+k] = true
		}
	}
	r.snoozeAt = time.Time{}
}

// Status describes today + the next upcoming prayer (for the sidebar pill).
type PrayerStatus struct {
	Enabled  bool              `json:"enabled"`
	City     string            `json:"city"`
	CityName string            `json:"cityName"`
	Now      time.Time         `json:"now"`
	Today    map[string]string `json:"today"` // key -> HH:MM
	NextKey  string            `json:"nextKey"`
	NextTime *time.Time        `json:"nextTime"`
	InSecs   int64             `json:"inSeconds"`
}

// Status computes today's times and the next prayer.
func Status(s *store.Store, at time.Time) PrayerStatus {
	st := LoadSettings(s)
	out := PrayerStatus{Enabled: st.Enabled, City: st.City, Today: map[string]string{}}
	loc := ResolveLocation(st.TZ)
	local := at.In(loc)
	out.Now = local
	cityName := st.City
	if c := FindCity(st.City); c != nil {
		cityName = c.Name
	} else if st.City == "custom" {
		cityName = "Custom"
	}
	out.CityName = cityName
	times := ComputeTimes(local, st.Lat, st.Lng, st.TZ, st.Method, st.AsrHanafi).Times
	for _, k := range OrderedKeys {
		if t, ok := times[k]; ok {
			out.Today[k] = t.Format("15:04")
		}
	}
	for _, k := range OrderedKeys {
		if t, ok := times[k]; ok && local.Before(t) {
			out.NextKey = k
			cp := t
			out.NextTime = &cp
			out.InSecs = int64(t.Sub(local).Seconds())
			return out
		}
	}
	// After Isha: next is tomorrow's Fajr.
	tomorrow := ComputeTimes(local.Add(24*time.Hour), st.Lat, st.Lng, st.TZ, st.Method, st.AsrHanafi).Times
	if t, ok := tomorrow[Fajr]; ok {
		out.NextKey = Fajr
		cp := t
		out.NextTime = &cp
		out.InSecs = int64(t.Sub(local).Seconds())
	}
	return out
}
