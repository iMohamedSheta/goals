// Package store is the SQLite persistence layer for tasks, contexts, horizons and settings.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// ---------- Models ----------

type Context struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedAt string `json:"createdAt"`
	// Short note shown under the name (e.g. "Deep work & clients").
	Description   string `json:"description"`
	DescriptionAr string `json:"descriptionAr"`
	// Goal fields: every context doubles as a focus goal (e.g. Work 5h/day
	// or Gym 3h/week). Target is the goal per recurrence period in seconds
	// (0 = none); MaxSeconds is the lifetime estimate (0 = none, overtime
	// alert like tasks).
	DailyTarget       int64   `json:"dailyTargetSeconds"`
	MaxSeconds        int64   `json:"maxSeconds"`
	ElapsedSecs       int64   `json:"elapsedSeconds"`
	TimerStarted      *string `json:"timerStartedAt"`
	// TotalSeconds is the lifetime own-timer total (stored + live running
	// delta). elapsedSeconds is kept for compat (stored only); prefer
	// totalSeconds which already includes the live tick.
	TotalSeconds      int64 `json:"totalSeconds"`
	TodaySeconds      int64 `json:"todaySeconds"`
	TasksTodaySeconds int64 `json:"tasksTodaySeconds"`
	// TasksTotalSeconds is the lifetime rollup of task time inside this
	// context (all time_entries + live running task deltas). It never
	// resets; Today/Week reset every day/window automatically.
	TasksTotalSeconds int64 `json:"tasksTotalSeconds"`
	// Recurrence is the goal period: "daily" or "weekly" (rolling 7 days).
	// WeekSeconds/WeekTasksSeconds mirror the day fields over that window.
	Recurrence       string `json:"recurrence"`
	WeekSeconds      int64  `json:"weekSeconds"`
	WeekTasksSeconds int64  `json:"weekTasksSeconds"`
}

type ContextTimeEntry struct {
	ID        string  `json:"id"`
	ContextID string  `json:"contextId"`
	StartedAt string  `json:"startedAt"`
	EndedAt   *string `json:"endedAt"`
	Seconds   int64   `json:"seconds"`
}

type Horizon struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	LabelAr       string `json:"labelAr"`
	DefaultDays   int    `json:"defaultDays"`
	Description   string `json:"description"`
	DescriptionAr string `json:"descriptionAr"`
	Position      int    `json:"position"`
}

type Task struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Horizon       string  `json:"horizon"`
	Status        string  `json:"status"`
	ContextID     *string `json:"contextId"`
	ParentID      *string `json:"parentId"`
	Priority      string  `json:"priority"`
	StartDate     *string `json:"startDate"`
	DueDate       *string `json:"dueDate"`
	Focus         bool    `json:"focus"`
	SortOrder     int     `json:"sortOrder"`
	ElapsedSecs   int64   `json:"elapsedSeconds"`
	TimerStarted  *string `json:"timerStartedAt"`
	MaxSeconds    int64   `json:"maxSeconds"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	CompletedAt   *string `json:"completedAt"`
}

type TimeEntry struct {
	ID        string  `json:"id"`
	TaskID    string  `json:"taskId"`
	StartedAt string  `json:"startedAt"`
	EndedAt   *string `json:"endedAt"`
	Seconds   int64   `json:"seconds"`
}

type TaskDetail struct {
	Task
	ContextName  *string `json:"contextName"`
	ContextColor *string `json:"contextColor"`
	ChildCount   int     `json:"childCount"`
	// TodaySeconds resets every day (UTC date of started_at): stored day
	// segments + the live running portion belonging to today. TotalSeconds
	// is the lifetime total (stored elapsed + full live delta) and never
	// resets — it is saved on every stop/checkpoint.
	TodaySeconds int64 `json:"todaySeconds"`
	TotalSeconds int64 `json:"totalSeconds"`
}

type TaskFilter struct {
	Horizon   string `json:"horizon"`
	Status    string `json:"status"`
	ContextID string `json:"contextId"`
	FocusOnly bool   `json:"focusOnly"`
	Search    string `json:"search"`
}

type TaskInput struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Horizon     string  `json:"horizon"`
	Status      string  `json:"status"`
	ContextID   *string `json:"contextId"`
	ParentID    *string `json:"parentId"`
	Priority    string  `json:"priority"`
	StartDate   *string `json:"startDate"`
	DueDate     *string `json:"dueDate"`
	Focus       bool    `json:"focus"`
	MaxSeconds  int64   `json:"maxSeconds"`
}

type Stats struct {
	ByHorizon map[string]map[string]int `json:"byHorizon"`
	Total     int                       `json:"total"`
	Focused   int                       `json:"focused"`
	Done      int                       `json:"done"`
}

// ---------- Store ----------

type Store struct {
	db   *sql.DB
	path string
}

// DataDir is the proper per-user home for the database:
//
//	Windows: %APPDATA%\Goals   (e.g. C:\Users\you\AppData\Roaming\Goals)
//	macOS:   ~/Library/Application Support/Goals
//	Linux:   ~/.config/Goals
func DataDir() string {
	if cfg, err := os.UserConfigDir(); err == nil {
		return filepath.Join(cfg, "Goals")
	}
	return "."
}

// LegacyDBPath is where versions before the data-dir move kept goals.db.
func LegacyDBPath() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if low := strings.ToLower(exe); !strings.Contains(low, "temp") && !strings.Contains(low, "tmp") && !strings.Contains(low, "go-build") {
			return filepath.Join(dir, "goals.db")
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "goals.db")
	}
	return "goals.db"
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// MigrateLegacyDB copies a database left next to the exe by older versions
// into the new data dir (once). It never deletes the original.
func MigrateLegacyDB(newPath string) (migratedFrom string, err error) {
	if fileExists(newPath) {
		return "", nil
	}
	if old := LegacyDBPath(); old != newPath && fileExists(old) {
		if mkErr := os.MkdirAll(filepath.Dir(newPath), 0755); mkErr != nil {
			return "", mkErr
		}
		data, rErr := os.ReadFile(old)
		if rErr != nil {
			return "", rErr
		}
		if wErr := os.WriteFile(newPath, data, 0644); wErr != nil {
			return "", wErr
		}
		for _, ext := range []string{"-wal", "-shm", "-journal"} {
			_ = os.Remove(newPath + ext)
		}
		return old, nil
	}
	return "", nil
}

func ResolveDBPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if v := os.Getenv("GOALS_DB_PATH"); v != "" {
		return v
	}
	return filepath.Join(DataDir(), "goals.db")
}

func Open(path string) (*Store, error) {
	// Only auto-migrate into the standard data dir — never clobber an
	// explicitly chosen database.
	if path == filepath.Join(DataDir(), "goals.db") {
		if _, err := MigrateLegacyDB(path); err != nil {
			return nil, fmt.Errorf("migrate legacy db: %w", err)
		}
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`); err != nil {
		return nil, err
	}
	s := &Store{db: db, path: path}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Path() string { return s.path }

// VacuumInto writes a clean, self-contained snapshot of the database to dest.
// Safe while open. Overwrites dest.
func (s *Store) VacuumInto(dest string) error {
	if dir := filepath.Dir(dest); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	_ = os.Remove(dest)
	safe := strings.ReplaceAll(dest, "'", "''")
	if _, err := s.db.Exec(`VACUUM INTO '` + safe + `'`); err != nil {
		return err
	}
	return nil
}

func nowStr() string { return time.Now().UTC().Format(time.RFC3339) }

func (s *Store) ensureColumn(table, column, ddl string) error {
	rows, err := s.db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt any
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = s.db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, ddl))
	return err
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS contexts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			color TEXT NOT NULL DEFAULT '#6366f1',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS horizons (
			key TEXT PRIMARY KEY,
			label TEXT NOT NULL,
			default_days INTEGER NOT NULL,
			description TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			horizon TEXT NOT NULL DEFAULT 'short',
			status TEXT NOT NULL DEFAULT 'todo',
			context_id TEXT REFERENCES contexts(id) ON DELETE SET NULL,
			parent_id TEXT,
			priority TEXT NOT NULL DEFAULT 'medium',
			start_date TEXT,
			due_date TEXT,
			focus INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			completed_at TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_horizon ON tasks(horizon);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_context ON tasks(context_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_focus ON tasks(focus);`,
		`CREATE TABLE IF NOT EXISTS time_entries (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			started_at TEXT NOT NULL,
			ended_at TEXT,
			seconds INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_entries_task ON time_entries(task_id);`,
		`CREATE TABLE IF NOT EXISTS context_time_entries (
			id TEXT PRIMARY KEY,
			context_id TEXT NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
			started_at TEXT NOT NULL,
			ended_at TEXT,
			seconds INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_ctx_entries_ctx ON context_time_entries(context_id);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS activity_segments (
			id TEXT PRIMARY KEY,
			app TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '',
			domain TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT 'other',
			started_at TEXT NOT NULL,
			ended_at TEXT,
			seconds INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_activity_started ON activity_segments(started_at);`,
		`CREATE INDEX IF NOT EXISTS idx_activity_app ON activity_segments(app);`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// additive timer columns for DBs created before the timer feature
	if err := s.ensureColumn("tasks", "elapsed_seconds", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("tasks", "timer_started_at", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("tasks", "max_seconds", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	// subtasks: parent_id links a child to its parent (NULL = top-level).
	// No hard FK so deletes can promote children instead of losing them.
	if err := s.ensureColumn("tasks", "parent_id", "TEXT"); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_id)`); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	// goal columns on contexts (older DBs lack them; existing rows get zeros)
	if err := s.ensureColumn("contexts", "daily_target", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("contexts", "max_seconds", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("contexts", "elapsed_seconds", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("contexts", "timer_started_at", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("contexts", "recurrence", "TEXT NOT NULL DEFAULT 'daily'"); err != nil {
		return err
	}
	if err := s.ensureColumn("contexts", "description", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("contexts", "description_ar", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("horizons", "label_ar", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("horizons", "description_ar", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("horizons", "position", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	// seed horizons
	seeds := []Horizon{
		{Key: "short", Label: "Short-Term", LabelAr: "قصيرة المدى", DefaultDays: 7, Description: "1 week goal planning", DescriptionAr: "تخطيط الأهداف لمدة أسبوع"},
		{Key: "medium", Label: "Medium-Term", LabelAr: "متوسطة المدى", DefaultDays: 30, Description: "1 month goal planning", DescriptionAr: "تخطيط الأهداف لمدة شهر"},
		{Key: "long", Label: "Long-Term", LabelAr: "طويلة المدى", DefaultDays: 90, Description: "3 months goal planning", DescriptionAr: "تخطيط الأهداف لمدة 3 أشهر"},
	}
	for _, h := range seeds {
		_, err := s.db.Exec(`INSERT INTO horizons(key,label,default_days,description) VALUES(?,?,?,?)
			ON CONFLICT(key) DO NOTHING`, h.Key, h.Label, h.DefaultDays, h.Description)
		if err != nil {
			return err
		}
		// backfill Arabic labels for DBs seeded before i18n
		_, err = s.db.Exec(`UPDATE horizons SET label_ar=?, description_ar=? WHERE key=? AND (label_ar='' OR label_ar IS NULL)`, h.LabelAr, h.DescriptionAr, h.Key)
		if err != nil {
			return err
		}
	}
	// backfill tab ordering for the built-in horizons
	_, _ = s.db.Exec(`UPDATE horizons SET position=0 WHERE key='short'`)
	_, _ = s.db.Exec(`UPDATE horizons SET position=1 WHERE key='medium'`)
	_, _ = s.db.Exec(`UPDATE horizons SET position=2 WHERE key='long'`)
	// seed default contexts if empty
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM contexts`).Scan(&n)
	if n == 0 {
		now := nowStr()
		defaults := []Context{
			{ID: uuid.NewString(), Name: "Work", Color: "#3b82f6", CreatedAt: now},
			{ID: uuid.NewString(), Name: "Life", Color: "#22c55e", CreatedAt: now},
			{ID: uuid.NewString(), Name: "Health", Color: "#ef4444", CreatedAt: now},
			{ID: uuid.NewString(), Name: "Learning", Color: "#a855f7", CreatedAt: now},
		}
		for _, c := range defaults {
			_, _ = s.db.Exec(`INSERT INTO contexts(id,name,color,created_at) VALUES(?,?,?,?)`, c.ID, c.Name, c.Color, c.CreatedAt)
		}
	}
	return nil
}

// ---------- Contexts (each one doubles as a focus goal) ----------

const contextSelect = `SELECT id,name,color,created_at,description,description_ar,daily_target,max_seconds,elapsed_seconds,timer_started_at,recurrence FROM contexts`

func validRecurrence(r string) string {
	switch strings.TrimSpace(r) {
	case "weekly":
		return "weekly"
	default:
		return "daily"
	}
}

func scanContextRows(rows *sql.Rows) (Context, error) {
	var c Context
	var tstarted sql.NullString
	var recurrence sql.NullString
	err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.CreatedAt, &c.Description, &c.DescriptionAr, &c.DailyTarget, &c.MaxSeconds, &c.ElapsedSecs, &tstarted, &recurrence)
	if err != nil {
		return c, err
	}
	if tstarted.Valid {
		v := tstarted.String
		c.TimerStarted = &v
	}
	c.Recurrence = "daily"
	if recurrence.Valid && recurrence.String != "" {
		c.Recurrence = validRecurrence(recurrence.String)
	}
	return c, nil
}

func scanContextRow(row *sql.Row) (Context, error) {
	var c Context
	var tstarted sql.NullString
	var recurrence sql.NullString
	err := row.Scan(&c.ID, &c.Name, &c.Color, &c.CreatedAt, &c.Description, &c.DescriptionAr, &c.DailyTarget, &c.MaxSeconds, &c.ElapsedSecs, &tstarted, &recurrence)
	if err != nil {
		return c, err
	}
	if tstarted.Valid {
		v := tstarted.String
		c.TimerStarted = &v
	}
	c.Recurrence = "daily"
	if recurrence.Valid && recurrence.String != "" {
		c.Recurrence = validRecurrence(recurrence.String)
	}
	return c, nil
}

func (s *Store) fillContextDay(c *Context, day string) {
	ts := ""
	if c.TimerStarted != nil {
		ts = *c.TimerStarted
	}
	now := time.Now().UTC()
	// Lifetime own-timer total (stored + live) — never resets, always saved
	// on stop/checkpoint via elapsed_seconds.
	c.TotalSeconds = c.ElapsedSecs + liveFullDelta(ts, now)
	today, _ := s.contextOwnDaySum(c.ID, ts, day)
	c.TodaySeconds = today
	c.TasksTodaySeconds = s.contextDaySum(c.ID, day)
	c.TasksTotalSeconds = s.contextTasksTotal(c.ID)
	week, _ := s.contextOwnWeekSum(c.ID, ts, day)
	c.WeekSeconds = week
	c.WeekTasksSeconds = s.contextWeekSum(c.ID, day)
}

// liveFullDelta is the full running delta for a timerStartedAt value.
func liveFullDelta(timerStarted string, now time.Time) int64 {
	if strings.TrimSpace(timerStarted) == "" {
		return 0
	}
	start, err := time.Parse(time.RFC3339, timerStarted)
	if err != nil {
		return 0
	}
	if d := int64(now.Sub(start).Seconds()); d > 0 {
		return d
	}
	return 0
}

// liveDayPortion returns only the running seconds belonging to `day`
// (YYYY-MM-DD, UTC). A timer started yesterday and still running today only
// contributes seconds since midnight — so the daily counter resets cleanly.
func liveDayPortion(timerStarted, day string, now time.Time) int64 {
	if strings.TrimSpace(timerStarted) == "" {
		return 0
	}
	start, err := time.Parse(time.RFC3339, timerStarted)
	if err != nil {
		return 0
	}
	d, err := time.Parse("2006-01-02", normDay(day))
	if err != nil {
		d = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
	dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	s := start
	if s.Before(dayStart) {
		s = dayStart
	}
	e := now
	if e.After(dayEnd) {
		e = dayEnd
	}
	if e.Before(s) {
		return 0
	}
	return int64(e.Sub(s).Seconds())
}

// liveWeekPortion mirrors liveDayPortion over the rolling 7-day window
// [weekStart(day) 00:00, now].
func liveWeekPortion(timerStarted, day string, now time.Time) int64 {
	if strings.TrimSpace(timerStarted) == "" {
		return 0
	}
	start, err := time.Parse(time.RFC3339, timerStarted)
	if err != nil {
		return 0
	}
	ws, err := time.Parse("2006-01-02", weekStart(normDay(day)))
	if err != nil {
		return liveFullDelta(timerStarted, now)
	}
	weekBegin := time.Date(ws.Year(), ws.Month(), ws.Day(), 0, 0, 0, 0, time.UTC)
	s := start
	if s.Before(weekBegin) {
		s = weekBegin
	}
	if now.Before(s) {
		return 0
	}
	return int64(now.Sub(s).Seconds())
}

// weekStart returns the UTC date 6 days before day (rolling 7-day window).
func weekStart(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		t = time.Now().UTC()
	}
	return t.AddDate(0, 0, -6).Format("2006-01-02")
}

// contextOwnWeekSum mirrors contextOwnDaySum over the rolling 7-day window.
func (s *Store) contextOwnWeekSum(id, timerStarted string, day string) (int64, bool) {
	var sum sql.NullInt64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(seconds),0) FROM context_time_entries WHERE context_id=? AND substr(started_at,1,10)>=?`, id, weekStart(day)).Scan(&sum)
	total := int64(0)
	if sum.Valid {
		total = sum.Int64
	}
	running := strings.TrimSpace(timerStarted) != ""
	if running {
		total += liveWeekPortion(timerStarted, day, time.Now().UTC())
	}
	return total, running
}

// contextWeekSum mirrors contextDaySum over the rolling 7-day window.
// Live running task deltas in this context are included (week portion).
func (s *Store) contextWeekSum(ctxID, day string) int64 {
	if strings.TrimSpace(ctxID) == "" {
		return 0
	}
	var sum sql.NullInt64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(e.seconds),0) FROM time_entries e JOIN tasks t ON t.id=e.task_id WHERE t.context_id=? AND substr(e.started_at,1,10)>=?`, ctxID, weekStart(day)).Scan(&sum)
	total := int64(0)
	if sum.Valid {
		total = sum.Int64
	}
	total += s.runningTasksWeekPortion(ctxID, day)
	return total
}

// contextTasksTotal is the lifetime rollup of task time inside a context:
// all stored segments + full live deltas of currently running tasks there.
// It never resets — the daily/weekly fields reset, this one accumulates.
func (s *Store) contextTasksTotal(ctxID string) int64 {
	if strings.TrimSpace(ctxID) == "" {
		return 0
	}
	var sum sql.NullInt64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(e.seconds),0) FROM time_entries e JOIN tasks t ON t.id=e.task_id WHERE t.context_id=?`, ctxID).Scan(&sum)
	total := int64(0)
	if sum.Valid {
		total = sum.Int64
	}
	rows, err := s.db.Query(`SELECT timer_started_at FROM tasks WHERE context_id=? AND timer_started_at IS NOT NULL`, ctxID)
	if err != nil {
		return total
	}
	defer rows.Close()
	now := time.Now().UTC()
	for rows.Next() {
		var ts string
		if err := rows.Scan(&ts); err != nil {
			continue
		}
		total += liveFullDelta(ts, now)
	}
	return total
}

// runningTasksDayPortion sums live today-portions of running tasks in a context.
func (s *Store) runningTasksDayPortion(ctxID, day string) int64 {
	rows, err := s.db.Query(`SELECT timer_started_at FROM tasks WHERE context_id=? AND timer_started_at IS NOT NULL`, ctxID)
	if err != nil {
		return 0
	}
	defer rows.Close()
	now := time.Now().UTC()
	var total int64
	for rows.Next() {
		var ts string
		if err := rows.Scan(&ts); err != nil {
			continue
		}
		total += liveDayPortion(ts, day, now)
	}
	return total
}

// runningTasksWeekPortion sums live week-portions of running tasks in a context.
func (s *Store) runningTasksWeekPortion(ctxID, day string) int64 {
	rows, err := s.db.Query(`SELECT timer_started_at FROM tasks WHERE context_id=? AND timer_started_at IS NOT NULL`, ctxID)
	if err != nil {
		return 0
	}
	defer rows.Close()
	now := time.Now().UTC()
	var total int64
	for rows.Next() {
		var ts string
		if err := rows.Scan(&ts); err != nil {
			continue
		}
		total += liveWeekPortion(ts, day, now)
	}
	return total
}

// ListContexts returns all contexts with per-day goal progress attached.
func (s *Store) ListContexts(day string) ([]Context, error) {
	day = normDay(day)
	rows, err := s.db.Query(contextSelect + ` ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	out := []Context{}
	for rows.Next() {
		c, err := scanContextRows(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	// Enrich only after rows are closed: the pool is limited to a single
	// connection, so nested queries must never run while rows are open.
	for i := range out {
		s.fillContextDay(&out[i], day)
	}
	return out, nil
}

// GetContext returns one context with per-day goal progress attached.
func (s *Store) GetContext(id, day string) (Context, error) {
	day = normDay(day)
	c, err := scanContextRow(s.db.QueryRow(contextSelect+` WHERE id=?`, id))
	if err != nil {
		return c, err
	}
	s.fillContextDay(&c, day)
	return c, nil
}

func (s *Store) CreateContext(name, color string) (Context, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Context{}, fmt.Errorf("context name is required")
	}
	if color == "" {
		color = "#6366f1"
	}
	c := Context{ID: uuid.NewString(), Name: name, Color: color, CreatedAt: nowStr()}
	if _, err := s.db.Exec(`INSERT INTO contexts(id,name,color,created_at) VALUES(?,?,?,?)`, c.ID, c.Name, c.Color, c.CreatedAt); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Context{}, fmt.Errorf("context %q already exists", name)
		}
		return Context{}, err
	}
	return s.GetContext(c.ID, "")
}

func (s *Store) UpdateContext(id, name, color string, dailyTarget, maxSeconds int64, recurrence, description, descriptionAr string) (Context, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Context{}, fmt.Errorf("context name is required")
	}
	if dailyTarget < 0 {
		dailyTarget = 0
	}
	if maxSeconds < 0 {
		maxSeconds = 0
	}
	recurrence = validRecurrence(recurrence)
	if _, err := s.db.Exec(`UPDATE contexts SET name=?, color=?, daily_target=?, max_seconds=?, recurrence=?, description=?, description_ar=? WHERE id=?`, name, color, dailyTarget, maxSeconds, recurrence, description, descriptionAr, id); err != nil {
		return Context{}, err
	}
	return s.GetContext(id, "")
}

func (s *Store) DeleteContext(id string) error {
	_, err := s.db.Exec(`DELETE FROM contexts WHERE id=?`, id)
	return err
}

// ---------- Horizons ----------

func (s *Store) ListHorizons() ([]Horizon, error) {
	rows, err := s.db.Query(`SELECT key,label,COALESCE(label_ar,''),default_days,description,COALESCE(description_ar,''),position FROM horizons ORDER BY position ASC, key ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Horizon{}
	for rows.Next() {
		var h Horizon
		if err := rows.Scan(&h.Key, &h.Label, &h.LabelAr, &h.DefaultDays, &h.Description, &h.DescriptionAr, &h.Position); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) UpdateHorizon(key, label, labelAr string, defaultDays int, description, descriptionAr string) (Horizon, error) {
	if strings.TrimSpace(key) == "" {
		return Horizon{}, fmt.Errorf("unknown horizon")
	}
	if _, err := s.GetHorizon(key); err != nil {
		return Horizon{}, fmt.Errorf("horizon not found")
	}
	if strings.TrimSpace(label) == "" {
		return Horizon{}, fmt.Errorf("label is required")
	}
	if defaultDays < 1 {
		defaultDays = 1
	}
	if defaultDays > 3650 {
		defaultDays = 3650
	}
	if _, err := s.db.Exec(`UPDATE horizons SET label=?, label_ar=?, default_days=?, description=?, description_ar=? WHERE key=?`, label, labelAr, defaultDays, description, descriptionAr, key); err != nil {
		return Horizon{}, err
	}
	var h Horizon
	err := s.db.QueryRow(`SELECT key,label,COALESCE(label_ar,''),default_days,description,COALESCE(description_ar,''),position FROM horizons WHERE key=?`, key).Scan(&h.Key, &h.Label, &h.LabelAr, &h.DefaultDays, &h.Description, &h.DescriptionAr, &h.Position)
	return h, err
}

func (s *Store) GetHorizon(key string) (Horizon, error) {
	var h Horizon
	err := s.db.QueryRow(`SELECT key,label,COALESCE(label_ar,''),default_days,description,COALESCE(description_ar,''),position FROM horizons WHERE key=?`, key).Scan(&h.Key, &h.Label, &h.LabelAr, &h.DefaultDays, &h.Description, &h.DescriptionAr, &h.Position)
	return h, err
}

// CreateHorizon adds a custom planning tab at the end.
func (s *Store) CreateHorizon(label, labelAr string, defaultDays int, description, descriptionAr string) (Horizon, error) {
	label = strings.TrimSpace(label)
	if label == "" && strings.TrimSpace(labelAr) == "" {
		return Horizon{}, fmt.Errorf("label is required")
	}
	if label == "" {
		label = labelAr
	}
	if defaultDays < 1 {
		defaultDays = 7
	}
	if defaultDays > 3650 {
		defaultDays = 3650
	}
	var maxPos int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(position),-1) FROM horizons`).Scan(&maxPos)
	h := Horizon{
		Key: "c" + strings.ReplaceAll(uuid.NewString()[:8], "-", ""),
		Label: label, LabelAr: labelAr, DefaultDays: defaultDays,
		Description: description, DescriptionAr: descriptionAr, Position: maxPos + 1,
	}
	if _, err := s.db.Exec(`INSERT INTO horizons(key,label,label_ar,default_days,description,description_ar,position) VALUES(?,?,?,?,?,?,?)`,
		h.Key, h.Label, h.LabelAr, h.DefaultDays, h.Description, h.DescriptionAr, h.Position); err != nil {
		return Horizon{}, err
	}
	return h, nil
}

// DeleteHorizon removes a custom tab. Built-ins and non-empty tabs are guarded.
func (s *Store) DeleteHorizon(key string) error {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM horizons`).Scan(&n)
	if n <= 1 {
		return fmt.Errorf("cannot delete the last planning tab")
	}
	var tasks int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE horizon=?`, key).Scan(&tasks)
	if tasks > 0 {
		return fmt.Errorf("tab has %d tasks — move or delete them first", tasks)
	}
	res, err := s.db.Exec(`DELETE FROM horizons WHERE key=?`, key)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("horizon not found")
	}
	return nil
}

// normalizeHorizon keeps tasks on a valid tab (falls back to the first tab).
func (s *Store) normalizeHorizon(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		h = "short"
	}
	var key string
	if err := s.db.QueryRow(`SELECT key FROM horizons WHERE key=?`, h).Scan(&key); err == nil {
		return key
	}
	_ = s.db.QueryRow(`SELECT key FROM horizons ORDER BY position ASC, key ASC LIMIT 1`).Scan(&key)
	if key == "" {
		return "short"
	}
	return key
}

// ---------- Tasks ----------

func validStatus(st string) string {
	switch st {
	case "todo", "in_progress", "done", "blocked":
		return st
	default:
		return "todo"
	}
}

func validPriority(p string) string {
	switch p {
	case "low", "medium", "high", "urgent":
		return p
	default:
		return "medium"
	}
}

func normDate(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func scanTaskDetail(rows *sql.Rows) (TaskDetail, error) {
	var t TaskDetail
	var ctxID, parentID sql.NullString
	var start, due, completed, tstarted sql.NullString
	var ctxName, ctxColor sql.NullString
	var focus int
	err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Horizon, &t.Status, &ctxID, &parentID, &t.Priority, &start, &due, &focus, &t.SortOrder, &t.ElapsedSecs, &tstarted, &t.MaxSeconds, &t.CreatedAt, &t.UpdatedAt, &completed, &ctxName, &ctxColor, &t.ChildCount)
	if err != nil {
		return t, err
	}
	if ctxID.Valid {
		v := ctxID.String
		t.ContextID = &v
	}
	if parentID.Valid && strings.TrimSpace(parentID.String) != "" {
		v := parentID.String
		t.ParentID = &v
	}
	if start.Valid {
		v := start.String
		t.StartDate = &v
	}
	if due.Valid {
		v := due.String
		t.DueDate = &v
	}
	if completed.Valid {
		v := completed.String
		t.CompletedAt = &v
	}
	if tstarted.Valid {
		v := tstarted.String
		t.TimerStarted = &v
	}
	if ctxName.Valid {
		v := ctxName.String
		t.ContextName = &v
	}
	if ctxColor.Valid {
		v := ctxColor.String
		t.ContextColor = &v
	}
	t.Focus = focus == 1
	return t, nil
}

const taskDetailSelect = `SELECT t.id,t.title,t.description,t.horizon,t.status,t.context_id,t.parent_id,t.priority,t.start_date,t.due_date,t.focus,t.sort_order,t.elapsed_seconds,t.timer_started_at,t.max_seconds,t.created_at,t.updated_at,t.completed_at,c.name,c.color,(SELECT COUNT(*) FROM tasks ch WHERE ch.parent_id=t.id)
	FROM tasks t LEFT JOIN contexts c ON c.id=t.context_id`

// ---------- Task time: daily (resets) + total (never resets) ----------

// taskDaySum sums stored task segments for one calendar day (UTC date prefix
// of started_at). Live running time is added by the caller via liveDayPortion.
func (s *Store) taskDaySum(taskID, day string) int64 {
	var sum sql.NullInt64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(seconds),0) FROM time_entries WHERE task_id=? AND substr(started_at,1,10)=?`, taskID, day).Scan(&sum)
	if sum.Valid {
		return sum.Int64
	}
	return 0
}

// fillTaskTime attaches TodaySeconds (resets every day) and TotalSeconds
// (lifetime total, stored + live) to a task detail. Both are derived from
// saved segments so they survive restarts; the timer itself never needs a
// manual reset — TodaySeconds is recomputed per calendar day.
func (s *Store) fillTaskTime(t *TaskDetail, day string) {
	day = normDay(day)
	now := time.Now().UTC()
	ts := ""
	if t.TimerStarted != nil {
		ts = *t.TimerStarted
	}
	t.TotalSeconds = t.ElapsedSecs + liveFullDelta(ts, now)
	t.TodaySeconds = s.taskDaySum(t.ID, day) + liveDayPortion(ts, day, now)
}

// TaskToday returns today's tracked seconds for a task (resets daily).
func (s *Store) TaskToday(id, day string) (int64, bool, error) {
	day = normDay(day)
	var stored int64
	var tstarted sql.NullString
	if err := s.db.QueryRow(`SELECT elapsed_seconds, timer_started_at FROM tasks WHERE id=?`, id).Scan(&stored, &tstarted); err != nil {
		return 0, false, err
	}
	ts := ""
	running := false
	if tstarted.Valid {
		ts = tstarted.String
		running = strings.TrimSpace(ts) != ""
	}
	total := s.taskDaySum(id, day) + liveDayPortion(ts, day, time.Now().UTC())
	return total, running, nil
}

// TaskTotal returns the lifetime total for a task (never resets).
func (s *Store) TaskTotal(id string) (int64, bool, error) {
	return s.Elapsed(id)
}

// ContextTotal returns the lifetime own-timer total for a context.
func (s *Store) ContextTotal(id string) (int64, bool, error) {
	return s.ContextElapsed(id)
}

// ContextTasksTotal returns the lifetime task rollup inside a context.
func (s *Store) ContextTasksTotal(id string) (int64, error) {
	if _, err := s.GetContext(id, ""); err != nil {
		return 0, err
	}
	return s.contextTasksTotal(id), nil
}

func (s *Store) ListTasks(f TaskFilter) ([]TaskDetail, error) {
	conds := []string{}
	args := []any{}
	if f.Horizon != "" && f.Horizon != "all" {
		conds = append(conds, "t.horizon=?")
		args = append(args, f.Horizon)
	}
	if f.Status != "" && f.Status != "all" {
		conds = append(conds, "t.status=?")
		args = append(args, f.Status)
	}
	if f.ContextID == "none" {
		// tasks with no context at all
		conds = append(conds, "t.context_id IS NULL")
	} else if f.ContextID == "with-context" {
		// tasks belonging to any context ("All contexts")
		conds = append(conds, "t.context_id IS NOT NULL")
	} else if f.ContextID != "" && f.ContextID != "all" {
		conds = append(conds, "t.context_id=?")
		args = append(args, f.ContextID)
	}
	if f.FocusOnly {
		conds = append(conds, "t.focus=1")
	}
	if q := strings.TrimSpace(f.Search); q != "" {
		conds = append(conds, "(t.title LIKE ? OR t.description LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	q := taskDetailSelect
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY t.focus DESC, t.sort_order ASC, t.created_at DESC"
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	out := []TaskDetail{}
	for rows.Next() {
		t, err := scanTaskDetail(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	// Enrich after close (single-connection pool): daily + total per task.
	day := time.Now().UTC().Format("2006-01-02")
	for i := range out {
		s.fillTaskTime(&out[i], day)
	}
	return out, nil
}

func (s *Store) GetTask(id string) (TaskDetail, error) {
	row := s.db.QueryRow(taskDetailSelect+` WHERE t.id=?`, id)
	var t TaskDetail
	var ctxID, parentID sql.NullString
	var start, due, completed, tstarted sql.NullString
	var ctxName, ctxColor sql.NullString
	var focus int
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Horizon, &t.Status, &ctxID, &parentID, &t.Priority, &start, &due, &focus, &t.SortOrder, &t.ElapsedSecs, &tstarted, &t.MaxSeconds, &t.CreatedAt, &t.UpdatedAt, &completed, &ctxName, &ctxColor, &t.ChildCount)
	if err != nil {
		return t, err
	}
	if ctxID.Valid {
		v := ctxID.String
		t.ContextID = &v
	}
	if parentID.Valid && strings.TrimSpace(parentID.String) != "" {
		v := parentID.String
		t.ParentID = &v
	}
	if start.Valid {
		v := start.String
		t.StartDate = &v
	}
	if due.Valid {
		v := due.String
		t.DueDate = &v
	}
	if completed.Valid {
		v := completed.String
		t.CompletedAt = &v
	}
	if tstarted.Valid {
		v := tstarted.String
		t.TimerStarted = &v
	}
	if ctxName.Valid {
		v := ctxName.String
		t.ContextName = &v
	}
	if ctxColor.Valid {
		v := ctxColor.String
		t.ContextColor = &v
	}
	t.Focus = focus == 1
	s.fillTaskTime(&t, time.Now().UTC().Format("2006-01-02"))
	return t, nil
}

func (s *Store) defaultDueDate(horizon string) *string {
	h, err := s.GetHorizon(horizon)
	if err != nil {
		return nil
	}
	d := time.Now().AddDate(0, 0, h.DefaultDays).Format("2006-01-02")
	return &d
}

func (s *Store) CreateTask(in TaskInput) (TaskDetail, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return TaskDetail{}, fmt.Errorf("title is required")
	}
	horizon := s.normalizeHorizon(in.Horizon)
	status := validStatus(in.Status)
	priority := validPriority(in.Priority)
	due := normDate(in.DueDate)
	if due == nil {
		due = s.defaultDueDate(horizon)
	}
	start := normDate(in.StartDate)
	var ctxID any
	if in.ContextID != nil && strings.TrimSpace(*in.ContextID) != "" {
		ctxID = strings.TrimSpace(*in.ContextID)
	} else {
		ctxID = nil
	}
	// Subtask: validate parent, adopt its horizon so the tree stays on one tab.
	var parentAny any
	var parentID *string
	if in.ParentID != nil && strings.TrimSpace(*in.ParentID) != "" {
		pid := strings.TrimSpace(*in.ParentID)
		p, err := s.GetTask(pid)
		if err != nil {
			return TaskDetail{}, fmt.Errorf("parent task not found")
		}
		horizon = p.Horizon
		parentAny = pid
		parentID = &pid
	} else {
		parentAny = nil
	}
	var maxOrder int
	if parentID != nil {
		_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order),0) FROM tasks WHERE parent_id=? AND horizon=? AND status=?`, *parentID, horizon, status).Scan(&maxOrder)
	} else {
		_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order),0) FROM tasks WHERE parent_id IS NULL AND horizon=? AND status=?`, horizon, status).Scan(&maxOrder)
	}
	id := uuid.NewString()
	now := nowStr()
	var completed any
	if status == "done" {
		completed = now
	}
	var startAny, dueAny any
	if start != nil {
		startAny = *start
	}
	if due != nil {
		dueAny = *due
	}
	maxSecs := in.MaxSeconds
	if maxSecs < 0 {
		maxSecs = 0
	}
	_, err := s.db.Exec(`INSERT INTO tasks(id,title,description,horizon,status,context_id,parent_id,priority,start_date,due_date,focus,sort_order,max_seconds,created_at,updated_at,completed_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, title, in.Description, horizon, status, ctxID, parentAny, priority, startAny, dueAny, boolToInt(in.Focus), maxOrder+1, maxSecs, now, now, completed)
	if err != nil {
		return TaskDetail{}, err
	}
	return s.GetTask(id)
}

func (s *Store) UpdateTask(id string, in TaskInput) (TaskDetail, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return TaskDetail{}, fmt.Errorf("title is required")
	}
	horizon := s.normalizeHorizon(in.Horizon)
	status := validStatus(in.Status)
	priority := validPriority(in.Priority)
	due := normDate(in.DueDate)
	start := normDate(in.StartDate)
	var ctxID any
	if in.ContextID != nil && strings.TrimSpace(*in.ContextID) != "" {
		ctxID = strings.TrimSpace(*in.ContextID)
	} else {
		ctxID = nil
	}
	now := nowStr()
	prev, err := s.GetTask(id)
	if err != nil {
		return TaskDetail{}, fmt.Errorf("task not found")
	}
	// Parent is managed via SetTaskParent (avoids accidental un-nesting when
	// the edit form omits parentId). Keep the existing link. If the task is a
	// subtask, stay on the parent's horizon.
	if prev.ParentID != nil && strings.TrimSpace(*prev.ParentID) != "" {
		if p, pErr := s.GetTask(strings.TrimSpace(*prev.ParentID)); pErr == nil {
			horizon = p.Horizon
		}
	}
	var completed any
	if status == "done" && prev.Status != "done" {
		completed = now
	} else if status == "done" {
		if prev.CompletedAt != nil {
			completed = *prev.CompletedAt
		} else {
			completed = now
		}
	}
	var startAny, dueAny any
	if start != nil {
		startAny = *start
	}
	if due != nil {
		dueAny = *due
	}
	maxSecs := in.MaxSeconds
	if maxSecs < 0 {
		maxSecs = 0
	}
	_, err = s.db.Exec(`UPDATE tasks SET title=?,description=?,horizon=?,status=?,context_id=?,priority=?,start_date=?,due_date=?,focus=?,max_seconds=?,updated_at=?,completed_at=? WHERE id=?`,
		title, in.Description, horizon, status, ctxID, priority, startAny, dueAny, boolToInt(in.Focus), maxSecs, now, completed, id)
	if err != nil {
		return TaskDetail{}, err
	}
	return s.GetTask(id)
}

func (s *Store) MoveTask(id, status string) (TaskDetail, error) {
	status = validStatus(status)
	now := nowStr()
	var completed any
	if status == "done" {
		completed = now
	}
	// Keep the subtask link; place at the end of the sibling group.
	cur, err := s.GetTask(id)
	if err != nil {
		return TaskDetail{}, fmt.Errorf("task not found")
	}
	var maxOrder int
	if cur.ParentID != nil && strings.TrimSpace(*cur.ParentID) != "" {
		_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order),0) FROM tasks WHERE parent_id=? AND horizon=? AND status=? AND id<>?`, strings.TrimSpace(*cur.ParentID), cur.Horizon, status, id).Scan(&maxOrder)
	} else {
		_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order),0) FROM tasks WHERE parent_id IS NULL AND horizon=? AND status=? AND id<>?`, cur.Horizon, status, id).Scan(&maxOrder)
	}
	if _, err := s.db.Exec(`UPDATE tasks SET status=?, sort_order=?, updated_at=?, completed_at=CASE WHEN ?='done' THEN ? ELSE NULL END WHERE id=?`, status, maxOrder+1, now, status, completed, id); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTask(id)
}

func (s *Store) ToggleFocus(id string) (TaskDetail, error) {
	t, err := s.GetTask(id)
	if err != nil {
		return t, fmt.Errorf("task not found")
	}
	_, err = s.db.Exec(`UPDATE tasks SET focus=?, updated_at=? WHERE id=?`, boolToInt(!t.Focus), nowStr(), id)
	if err != nil {
		return t, err
	}
	return s.GetTask(id)
}

func (s *Store) DeleteTask(id string) error {
	cur, err := s.GetTask(id)
	if err != nil {
		return fmt.Errorf("task not found")
	}
	// Promote children to the deleted task's parent (top-level when the
	// deleted task was top-level) so no subtask is lost.
	var newParent any
	if cur.ParentID != nil && strings.TrimSpace(*cur.ParentID) != "" {
		newParent = strings.TrimSpace(*cur.ParentID)
	} else {
		newParent = nil
	}
	if _, err := s.db.Exec(`UPDATE tasks SET parent_id=?, updated_at=? WHERE parent_id=?`, newParent, nowStr(), id); err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

func (s *Store) ReorderTask(id string, newOrder int) error {
	_, err := s.db.Exec(`UPDATE tasks SET sort_order=?, updated_at=? WHERE id=?`, newOrder, nowStr(), id)
	return err
}

// ReorderTasks persists a manual drag order: ids[0] gets sort_order 0, etc.
// Callers pass the sibling IDs in their new visual order (top-level or one
// parent's children within one status column).
func (s *Store) ReorderTasks(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := nowStr()
	for i, id := range ids {
		if strings.TrimSpace(id) == "" {
			continue
		}
		if _, err := s.db.Exec(`UPDATE tasks SET sort_order=?, updated_at=? WHERE id=?`, i, now, strings.TrimSpace(id)); err != nil {
			return err
		}
	}
	return nil
}

// ancestors returns the parent chain of id, nearest first.
func (s *Store) ancestors(id string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{strings.TrimSpace(id): true}
	cur := strings.TrimSpace(id)
	for {
		var parent sql.NullString
		if err := s.db.QueryRow(`SELECT parent_id FROM tasks WHERE id=?`, cur).Scan(&parent); err != nil {
			return out, err
		}
		if !parent.Valid || strings.TrimSpace(parent.String) == "" {
			return out, nil
		}
		pid := strings.TrimSpace(parent.String)
		if seen[pid] {
			return out, fmt.Errorf("cyclic parent link")
		}
		seen[pid] = true
		out = append(out, pid)
		cur = pid
	}
}

// descendants returns all (transitive) children of id.
func (s *Store) descendants(id string) ([]string, error) {
	out := []string{}
	queue := []string{strings.TrimSpace(id)}
	seen := map[string]bool{strings.TrimSpace(id): true}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		rows, err := s.db.Query(`SELECT id FROM tasks WHERE parent_id=?`, cur)
		if err != nil {
			return out, err
		}
		var kids []string
		for rows.Next() {
			var kid string
			if err := rows.Scan(&kid); err != nil {
				rows.Close()
				return out, err
			}
			kids = append(kids, kid)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return out, err
		}
		for _, k := range kids {
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, k)
			queue = append(queue, k)
		}
	}
	return out, nil
}

// SetTaskParent nests id inside parentID (nil/empty = top-level). It adopts
// the parent horizon, guards cycles, and appends at the end of the new
// sibling group. Setting a task's parent to itself or its own descendant
// is refused.
func (s *Store) SetTaskParent(id string, parentID *string) (TaskDetail, error) {
	cur, err := s.GetTask(id)
	if err != nil {
		return TaskDetail{}, fmt.Errorf("task not found")
	}
	var pid *string
	if parentID != nil && strings.TrimSpace(*parentID) != "" {
		v := strings.TrimSpace(*parentID)
		pid = &v
	}
	if pid != nil {
		if *pid == cur.ID {
			return TaskDetail{}, fmt.Errorf("a task cannot be its own parent")
		}
		p, err := s.GetTask(*pid)
		if err != nil {
			return TaskDetail{}, fmt.Errorf("parent task not found")
		}
		kids, err := s.descendants(cur.ID)
		if err != nil {
			return TaskDetail{}, err
		}
		for _, k := range kids {
			if k == *pid {
				return TaskDetail{}, fmt.Errorf("cannot nest a task inside its own subtask")
			}
		}
		// Adopt the parent horizon so the whole tree lives on one tab.
		var maxOrder int
		_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order),0) FROM tasks WHERE parent_id=? AND id<>?`, *pid, cur.ID).Scan(&maxOrder)
		var parentAny any = *pid
		if _, err := s.db.Exec(`UPDATE tasks SET parent_id=?, horizon=?, sort_order=?, updated_at=? WHERE id=?`, parentAny, p.Horizon, maxOrder+1, nowStr(), cur.ID); err != nil {
			return TaskDetail{}, err
		}
		return s.GetTask(cur.ID)
	}
	// Un-nest to top-level, appended at the end of its status group.
	var maxOrder int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order),0) FROM tasks WHERE parent_id IS NULL AND horizon=? AND status=? AND id<>?`, cur.Horizon, cur.Status, cur.ID).Scan(&maxOrder)
	if _, err := s.db.Exec(`UPDATE tasks SET parent_id=NULL, sort_order=?, updated_at=? WHERE id=?`, maxOrder+1, nowStr(), cur.ID); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTask(cur.ID)
}

func (s *Store) GetStats() (Stats, error) {
	st := Stats{ByHorizon: map[string]map[string]int{}, Total: 0, Focused: 0, Done: 0}
	rows, err := s.db.Query(`SELECT horizon,status,COUNT(*) FROM tasks GROUP BY horizon,status`)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	for rows.Next() {
		var h, status string
		var c int
		if err := rows.Scan(&h, &status, &c); err != nil {
			return st, err
		}
		if _, ok := st.ByHorizon[h]; !ok {
			st.ByHorizon[h] = map[string]int{"todo": 0, "in_progress": 0, "done": 0, "blocked": 0, "total": 0}
		}
		st.ByHorizon[h][status] = c
		st.ByHorizon[h]["total"] += c
		st.Total += c
		if status == "done" {
			st.Done += c
		}
	}
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE focus=1`).Scan(&st.Focused)
	return st, rows.Err()
}

// ---------- Timer ----------

// Elapsed returns stored seconds + live running delta.
func (s *Store) Elapsed(id string) (int64, bool, error) {
	var stored int64
	var tstarted sql.NullString
	err := s.db.QueryRow(`SELECT elapsed_seconds, timer_started_at FROM tasks WHERE id=?`, id).Scan(&stored, &tstarted)
	if err != nil {
		return 0, false, err
	}
	if !tstarted.Valid {
		return stored, false, nil
	}
	start, err := time.Parse(time.RFC3339, tstarted.String)
	if err != nil {
		return stored, false, nil
	}
	d := int64(time.Since(start).Seconds())
	if d < 0 {
		d = 0
	}
	return stored + d, true, nil
}

// stopLocked finalizes the running segment of one task (caller handles tx/locking).
func (s *Store) stopLocked(id string, now time.Time) (int64, error) {
	var stored int64
	var tstarted sql.NullString
	if err := s.db.QueryRow(`SELECT elapsed_seconds, timer_started_at FROM tasks WHERE id=?`, id).Scan(&stored, &tstarted); err != nil {
		return 0, err
	}
	if !tstarted.Valid {
		return stored, nil
	}
	start, err := time.Parse(time.RFC3339, tstarted.String)
	if err != nil {
		_, _ = s.db.Exec(`UPDATE tasks SET timer_started_at=NULL, updated_at=? WHERE id=?`, nowStr(), id)
		return stored, nil
	}
	delta := int64(now.Sub(start).Seconds())
	if delta < 0 {
		delta = 0
	}
	total := stored + delta
	if _, err := s.db.Exec(`UPDATE tasks SET elapsed_seconds=?, timer_started_at=NULL, updated_at=? WHERE id=?`, total, nowStr(), id); err != nil {
		return 0, err
	}
	_, err = s.db.Exec(`INSERT INTO time_entries(id,task_id,started_at,ended_at,seconds,created_at) VALUES(?,?,?,?,?,?)`,
		uuid.NewString(), id, tstarted.String, now.UTC().Format(time.RFC3339), delta, nowStr())
	return total, err
}

// runningTaskIDs returns all task timers currently running.
func (s *Store) runningTaskIDs() ([]string, error) {
	rows, err := s.db.Query(`SELECT id FROM tasks WHERE timer_started_at IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// StartTimer starts tracking on a task; starting a subtask also starts its
// parent chain (both clocks run in parallel). Unrelated running tasks are
// auto-paused; ancestors/descendants/siblings-in-the-same-tree keep running
// except that switching between siblings pauses the previous sibling.
func (s *Store) StartTimer(id string) (TaskDetail, error) {
	cur, err := s.GetTask(id)
	if err != nil {
		return TaskDetail{}, fmt.Errorf("task not found")
	}
	_ = cur
	now := time.Now().UTC()
	anc, err := s.ancestors(id)
	if err != nil {
		return TaskDetail{}, err
	}
	desc, err := s.descendants(id)
	if err != nil {
		return TaskDetail{}, err
	}
	keep := map[string]bool{id: true}
	for _, a := range anc {
		keep[a] = true
	}
	for _, d := range desc {
		keep[d] = true
	}
	running, err := s.runningTaskIDs()
	if err != nil {
		return TaskDetail{}, err
	}
	for _, r := range running {
		if keep[r] {
			continue
		}
		// Sibling switch: a running task in the same tree that is neither an
		// ancestor nor a descendant gets paused (only one branch at a time),
		// while the shared parents keep running.
		if _, err := s.stopLocked(r, now); err != nil {
			return TaskDetail{}, err
		}
	}
	// Start the task plus any ancestor not already running (parent inherits).
	toStart := append([]string{id}, anc...)
	for _, tid := range toStart {
		var tstarted sql.NullString
		if err := s.db.QueryRow(`SELECT timer_started_at FROM tasks WHERE id=?`, tid).Scan(&tstarted); err != nil {
			return TaskDetail{}, err
		}
		if tstarted.Valid && strings.TrimSpace(tstarted.String) != "" {
			continue
		}
		if _, err := s.db.Exec(`UPDATE tasks SET timer_started_at=?, updated_at=? WHERE id=?`, now.Format(time.RFC3339), nowStr(), tid); err != nil {
			return TaskDetail{}, err
		}
	}
	return s.GetTask(id)
}

// StopTimer pauses tracking and saves the segment.
func (s *Store) StopTimer(id string) (TaskDetail, error) {
	if _, err := s.GetTask(id); err != nil {
		return TaskDetail{}, fmt.Errorf("task not found")
	}
	if _, err := s.stopLocked(id, time.Now().UTC()); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTask(id)
}

// CheckpointRunning folds live deltas into elapsed_seconds (crash safety).
// It handles both task and context timers, which run independently.
func (s *Store) CheckpointRunning() error {
	if err := s.checkpointTasks(); err != nil {
		return err
	}
	return s.checkpointContexts()
}

func (s *Store) checkpointTasks() error {
	rows, err := s.db.Query(`SELECT id, elapsed_seconds, timer_started_at FROM tasks WHERE timer_started_at IS NOT NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type rec struct {
		id      string
		stored  int64
		started string
	}
	var recs []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.stored, &r.started); err != nil {
			return err
		}
		recs = append(recs, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, r := range recs {
		start, err := time.Parse(time.RFC3339, r.started)
		if err != nil {
			continue
		}
		delta := int64(now.Sub(start).Seconds())
		if delta < 1 {
			continue
		}
		_, err = s.db.Exec(`UPDATE tasks SET elapsed_seconds=?, timer_started_at=?, updated_at=? WHERE id=? AND timer_started_at=?`,
			r.stored+delta, now.Format(time.RFC3339), nowStr(), r.id, r.started)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(`INSERT INTO time_entries(id,task_id,started_at,ended_at,seconds,created_at) VALUES(?,?,?,?,?,?)`,
			uuid.NewString(), r.id, r.started, now.Format(time.RFC3339), delta, nowStr())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) checkpointContexts() error {
	rows, err := s.db.Query(`SELECT id, elapsed_seconds, timer_started_at FROM contexts WHERE timer_started_at IS NOT NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type rec struct {
		id      string
		stored  int64
		started string
	}
	var recs []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.stored, &r.started); err != nil {
			return err
		}
		recs = append(recs, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, r := range recs {
		start, err := time.Parse(time.RFC3339, r.started)
		if err != nil {
			continue
		}
		delta := int64(now.Sub(start).Seconds())
		if delta < 1 {
			continue
		}
		_, err = s.db.Exec(`UPDATE contexts SET elapsed_seconds=?, timer_started_at=? WHERE id=? AND timer_started_at=?`,
			r.stored+delta, now.Format(time.RFC3339), r.id, r.started)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(`INSERT INTO context_time_entries(id,context_id,started_at,ended_at,seconds,created_at) VALUES(?,?,?,?,?,?)`,
			uuid.NewString(), r.id, r.started, now.Format(time.RFC3339), delta, nowStr())
		if err != nil {
			return err
		}
	}
	return nil
}

// FinishTask stops the timer (saving time) and marks the task done.
// It also stops any running subtask timers so the subtree never keeps
// ticking under a completed parent.
func (s *Store) FinishTask(id string) (TaskDetail, error) {
	if _, err := s.GetTask(id); err != nil {
		return TaskDetail{}, fmt.Errorf("task not found")
	}
	now := time.Now().UTC()
	if kids, err := s.descendants(id); err == nil {
		for _, k := range kids {
			_, _ = s.stopLocked(k, now)
		}
	}
	if _, err := s.stopLocked(id, now); err != nil {
		return TaskDetail{}, err
	}
	nowStrVal := nowStr()
	if _, err := s.db.Exec(`UPDATE tasks SET status='done', completed_at=?, updated_at=? WHERE id=?`, nowStrVal, nowStrVal, id); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTask(id)
}

// GetActiveTimer returns the most recently started running task
// (the subtask when a parent+child pair runs together), or sql.ErrNoRows.
func (s *Store) GetActiveTimer() (TaskDetail, error) {
	row := s.db.QueryRow(taskDetailSelect+` WHERE t.timer_started_at IS NOT NULL ORDER BY t.timer_started_at DESC LIMIT 1`)
	var t TaskDetail
	var ctxID, parentID sql.NullString
	var start, due, completed, tstarted sql.NullString
	var ctxName, ctxColor sql.NullString
	var focus int
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Horizon, &t.Status, &ctxID, &parentID, &t.Priority, &start, &due, &focus, &t.SortOrder, &t.ElapsedSecs, &tstarted, &t.MaxSeconds, &t.CreatedAt, &t.UpdatedAt, &completed, &ctxName, &ctxColor, &t.ChildCount)
	if err != nil {
		return t, err
	}
	if ctxID.Valid {
		v := ctxID.String
		t.ContextID = &v
	}
	if parentID.Valid && strings.TrimSpace(parentID.String) != "" {
		v := parentID.String
		t.ParentID = &v
	}
	if start.Valid {
		v := start.String
		t.StartDate = &v
	}
	if due.Valid {
		v := due.String
		t.DueDate = &v
	}
	if completed.Valid {
		v := completed.String
		t.CompletedAt = &v
	}
	if tstarted.Valid {
		v := tstarted.String
		t.TimerStarted = &v
	}
	if ctxName.Valid {
		v := ctxName.String
		t.ContextName = &v
	}
	if ctxColor.Valid {
		v := ctxColor.String
		t.ContextColor = &v
	}
	t.Focus = focus == 1
	s.fillTaskTime(&t, time.Now().UTC().Format("2006-01-02"))
	return t, nil
}

// GetActiveTimers returns every running task timer (parent+subtask pairs run
// in parallel), most-recent first.
func (s *Store) GetActiveTimers() ([]TaskDetail, error) {
	rows, err := s.db.Query(taskDetailSelect + ` WHERE t.timer_started_at IS NOT NULL ORDER BY t.timer_started_at DESC`)
	if err != nil {
		return nil, err
	}
	out := []TaskDetail{}
	for rows.Next() {
		t, err := scanTaskDetail(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	day := time.Now().UTC().Format("2006-01-02")
	for i := range out {
		s.fillTaskTime(&out[i], day)
	}
	return out, nil
}

func (s *Store) ListTimeEntries(taskID string) ([]TimeEntry, error) {

	rows, err := s.db.Query(`SELECT id,task_id,started_at,ended_at,seconds FROM time_entries WHERE task_id=? ORDER BY started_at DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TimeEntry{}
	for rows.Next() {
		var e TimeEntry
		var ended sql.NullString
		if err := rows.Scan(&e.ID, &e.TaskID, &e.StartedAt, &ended, &e.Seconds); err != nil {
			return nil, err
		}
		if ended.Valid {
			v := ended.String
			e.EndedAt = &v
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------- Context goals ----------
// Every context doubles as a focus goal (e.g. Work 5h/day): it carries its
// own independent timer (parallel with the task timer), a daily target and a
// lifetime estimate. Task times in the context roll up as info (TasksToday)
// next to the context's own tracked time — the two clocks are independent.

// normDay returns today as YYYY-MM-DD (UTC) when s is empty/garbage.
func normDay(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		return s[:10]
	}
	return time.Now().UTC().Format("2006-01-02")
}

// contextOwnDaySum sums stored context entry seconds for one calendar day
// (by UTC date prefix of started_at) plus the live running portion that
// belongs to that day — so the daily counter resets every day.
func (s *Store) contextOwnDaySum(id, timerStarted string, day string) (int64, bool) {
	var sum sql.NullInt64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(seconds),0) FROM context_time_entries WHERE context_id=? AND substr(started_at,1,10)=?`, id, day).Scan(&sum)
	total := int64(0)
	if sum.Valid {
		total = sum.Int64
	}
	running := strings.TrimSpace(timerStarted) != ""
	if running {
		total += liveDayPortion(timerStarted, day, time.Now().UTC())
	}
	return total, running
}

// contextDaySum sums task time entries for one day whose task sits in ctxID,
// plus live running task portions for that day (so the daily rollup ticks
// live and resets the next day).
func (s *Store) contextDaySum(ctxID, day string) int64 {
	if strings.TrimSpace(ctxID) == "" {
		return 0
	}
	var sum sql.NullInt64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(e.seconds),0) FROM time_entries e JOIN tasks t ON t.id=e.task_id WHERE t.context_id=? AND substr(e.started_at,1,10)=?`, ctxID, day).Scan(&sum)
	total := int64(0)
	if sum.Valid {
		total = sum.Int64
	}
	total += s.runningTasksDayPortion(ctxID, day)
	return total
}

// ---------- Context timer (independent from the task timer; both may run) ----------

// ContextElapsed returns stored seconds + live running delta.
func (s *Store) ContextElapsed(id string) (int64, bool, error) {
	var stored int64
	var tstarted sql.NullString
	if err := s.db.QueryRow(`SELECT elapsed_seconds, timer_started_at FROM contexts WHERE id=?`, id).Scan(&stored, &tstarted); err != nil {
		return 0, false, err
	}
	if !tstarted.Valid {
		return stored, false, nil
	}
	start, err := time.Parse(time.RFC3339, tstarted.String)
	if err != nil {
		return stored, false, nil
	}
	d := int64(time.Since(start).Seconds())
	if d < 0 {
		d = 0
	}
	return stored + d, true, nil
}

// ContextToday returns today's tracked seconds (stored day segments + live delta).
func (s *Store) ContextToday(id, day string) (int64, bool, error) {
	day = normDay(day)
	var tstarted sql.NullString
	if err := s.db.QueryRow(`SELECT timer_started_at FROM contexts WHERE id=?`, id).Scan(&tstarted); err != nil {
		return 0, false, err
	}
	ts := ""
	running := false
	if tstarted.Valid {
		ts = tstarted.String
		running = true
	}
	total, _ := s.contextOwnDaySum(id, ts, day)
	return total, running, nil
}

func (s *Store) contextStopLocked(id string, now time.Time) (int64, error) {
	var stored int64
	var tstarted sql.NullString
	if err := s.db.QueryRow(`SELECT elapsed_seconds, timer_started_at FROM contexts WHERE id=?`, id).Scan(&stored, &tstarted); err != nil {
		return 0, err
	}
	if !tstarted.Valid {
		return stored, nil
	}
	start, err := time.Parse(time.RFC3339, tstarted.String)
	if err != nil {
		_, _ = s.db.Exec(`UPDATE contexts SET timer_started_at=NULL WHERE id=?`, id)
		return stored, nil
	}
	delta := int64(now.Sub(start).Seconds())
	if delta < 0 {
		delta = 0
	}
	total := stored + delta
	if _, err := s.db.Exec(`UPDATE contexts SET elapsed_seconds=?, timer_started_at=NULL WHERE id=?`, total, id); err != nil {
		return 0, err
	}
	_, err = s.db.Exec(`INSERT INTO context_time_entries(id,context_id,started_at,ended_at,seconds,created_at) VALUES(?,?,?,?,?,?)`,
		uuid.NewString(), id, tstarted.String, now.UTC().Format(time.RFC3339), delta, nowStr())
	return total, err
}

func (s *Store) runningContextLocked(except string) (string, error) {
	rows, err := s.db.Query(`SELECT id FROM contexts WHERE timer_started_at IS NOT NULL`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		if id != except {
			return id, nil
		}
	}
	return "", rows.Err()
}

// StartContextTimer starts a context goal timer; it pauses any other running
// context timer but never touches the task timer (parallel by design).
func (s *Store) StartContextTimer(id string) (Context, error) {
	if _, err := s.GetContext(id, ""); err != nil {
		return Context{}, fmt.Errorf("context not found")
	}
	now := time.Now().UTC()
	if other, err := s.runningContextLocked(id); err != nil {
		return Context{}, err
	} else if other != "" {
		if _, err := s.contextStopLocked(other, now); err != nil {
			return Context{}, err
		}
	}
	if _, err := s.db.Exec(`UPDATE contexts SET timer_started_at=? WHERE id=?`, now.Format(time.RFC3339), id); err != nil {
		return Context{}, err
	}
	return s.GetContext(id, "")
}

// StopContextTimer pauses a context goal timer and saves the segment.
func (s *Store) StopContextTimer(id string) (Context, error) {
	if _, err := s.GetContext(id, ""); err != nil {
		return Context{}, fmt.Errorf("context not found")
	}
	if _, err := s.contextStopLocked(id, time.Now().UTC()); err != nil {
		return Context{}, err
	}
	return s.GetContext(id, "")
}

// GetActiveContextTimer returns the running context goal, or sql.ErrNoRows.
func (s *Store) GetActiveContextTimer(day string) (Context, error) {
	day = normDay(day)
	c, err := scanContextRow(s.db.QueryRow(contextSelect+` WHERE timer_started_at IS NOT NULL LIMIT 1`))
	if err != nil {
		return c, err
	}
	s.fillContextDay(&c, day)
	return c, nil
}

func (s *Store) ListContextEntries(contextID string) ([]ContextTimeEntry, error) {
	rows, err := s.db.Query(`SELECT id,context_id,started_at,ended_at,seconds FROM context_time_entries WHERE context_id=? ORDER BY started_at DESC`, contextID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ContextTimeEntry{}
	for rows.Next() {
		var e ContextTimeEntry
		var ended sql.NullString
		if err := rows.Scan(&e.ID, &e.ContextID, &e.StartedAt, &ended, &e.Seconds); err != nil {
			return nil, err
		}
		if ended.Valid {
			v := ended.String
			e.EndedAt = &v
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------- Settings (appearance etc.) ----------

func (s *Store) GetSettings() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (s *Store) SetSetting(key, value string) error {
	if key == "" {
		return fmt.Errorf("setting key is required")
	}
	_, err := s.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (s *Store) DeleteSetting(key string) error {
	_, err := s.db.Exec(`DELETE FROM settings WHERE key=?`, key)
	return err
}
