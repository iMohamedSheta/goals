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
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		);`,
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

// ---------- Contexts ----------

func (s *Store) ListContexts() ([]Context, error) {
	rows, err := s.db.Query(`SELECT id,name,color,created_at FROM contexts ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Context{}
	for rows.Next() {
		var c Context
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
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
	return c, nil
}

func (s *Store) UpdateContext(id, name, color string) (Context, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Context{}, fmt.Errorf("context name is required")
	}
	if _, err := s.db.Exec(`UPDATE contexts SET name=?, color=? WHERE id=?`, name, color, id); err != nil {
		return Context{}, err
	}
	var c Context
	err := s.db.QueryRow(`SELECT id,name,color,created_at FROM contexts WHERE id=?`, id).Scan(&c.ID, &c.Name, &c.Color, &c.CreatedAt)
	return c, err
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
	var ctxID sql.NullString
	var start, due, completed, tstarted sql.NullString
	var ctxName, ctxColor sql.NullString
	var focus int
	err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Horizon, &t.Status, &ctxID, &t.Priority, &start, &due, &focus, &t.SortOrder, &t.ElapsedSecs, &tstarted, &t.MaxSeconds, &t.CreatedAt, &t.UpdatedAt, &completed, &ctxName, &ctxColor)
	if err != nil {
		return t, err
	}
	if ctxID.Valid {
		v := ctxID.String
		t.ContextID = &v
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

const taskDetailSelect = `SELECT t.id,t.title,t.description,t.horizon,t.status,t.context_id,t.priority,t.start_date,t.due_date,t.focus,t.sort_order,t.elapsed_seconds,t.timer_started_at,t.max_seconds,t.created_at,t.updated_at,t.completed_at,c.name,c.color
	FROM tasks t LEFT JOIN contexts c ON c.id=t.context_id`

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
	if f.ContextID != "" && f.ContextID != "all" {
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
	defer rows.Close()
	out := []TaskDetail{}
	for rows.Next() {
		t, err := scanTaskDetail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTask(id string) (TaskDetail, error) {
	row := s.db.QueryRow(taskDetailSelect+` WHERE t.id=?`, id)
	var t TaskDetail
	var ctxID sql.NullString
	var start, due, completed, tstarted sql.NullString
	var ctxName, ctxColor sql.NullString
	var focus int
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Horizon, &t.Status, &ctxID, &t.Priority, &start, &due, &focus, &t.SortOrder, &t.ElapsedSecs, &tstarted, &t.MaxSeconds, &t.CreatedAt, &t.UpdatedAt, &completed, &ctxName, &ctxColor)
	if err != nil {
		return t, err
	}
	if ctxID.Valid {
		v := ctxID.String
		t.ContextID = &v
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
	var maxOrder int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order),0) FROM tasks WHERE horizon=? AND status=?`, horizon, status).Scan(&maxOrder)
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
	_, err := s.db.Exec(`INSERT INTO tasks(id,title,description,horizon,status,context_id,priority,start_date,due_date,focus,sort_order,max_seconds,created_at,updated_at,completed_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, title, in.Description, horizon, status, ctxID, priority, startAny, dueAny, boolToInt(in.Focus), maxOrder+1, maxSecs, now, now, completed)
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
	var maxOrder int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order),0) FROM tasks WHERE status=?`, status).Scan(&maxOrder)
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

// runningTaskLocked returns the id of the currently running timer, if any.
func (s *Store) runningTaskLocked(except string) (string, error) {
	rows, err := s.db.Query(`SELECT id FROM tasks WHERE timer_started_at IS NOT NULL`)
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

// StartTimer starts tracking on a task; auto-pauses any other running timer.
func (s *Store) StartTimer(id string) (TaskDetail, error) {
	if _, err := s.GetTask(id); err != nil {
		return TaskDetail{}, fmt.Errorf("task not found")
	}
	now := time.Now().UTC()
	if other, err := s.runningTaskLocked(id); err != nil {
		return TaskDetail{}, err
	} else if other != "" {
		if _, err := s.stopLocked(other, now); err != nil {
			return TaskDetail{}, err
		}
	}
	if _, err := s.db.Exec(`UPDATE tasks SET timer_started_at=?, updated_at=? WHERE id=?`, now.Format(time.RFC3339), nowStr(), id); err != nil {
		return TaskDetail{}, err
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
func (s *Store) CheckpointRunning() error {
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

// FinishTask stops the timer (saving time) and marks the task done.
func (s *Store) FinishTask(id string) (TaskDetail, error) {
	if _, err := s.GetTask(id); err != nil {
		return TaskDetail{}, fmt.Errorf("task not found")
	}
	if _, err := s.stopLocked(id, time.Now().UTC()); err != nil {
		return TaskDetail{}, err
	}
	now := nowStr()
	if _, err := s.db.Exec(`UPDATE tasks SET status='done', completed_at=?, updated_at=? WHERE id=?`, now, now, id); err != nil {
		return TaskDetail{}, err
	}
	return s.GetTask(id)
}

// GetActiveTimer returns the currently running task, or sql.ErrNoRows.
func (s *Store) GetActiveTimer() (TaskDetail, error) {
	row := s.db.QueryRow(taskDetailSelect+` WHERE t.timer_started_at IS NOT NULL LIMIT 1`)
	var t TaskDetail
	var ctxID sql.NullString
	var start, due, completed, tstarted sql.NullString
	var ctxName, ctxColor sql.NullString
	var focus int
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Horizon, &t.Status, &ctxID, &t.Priority, &start, &due, &focus, &t.SortOrder, &t.ElapsedSecs, &tstarted, &t.MaxSeconds, &t.CreatedAt, &t.UpdatedAt, &completed, &ctxName, &ctxColor)
	if err != nil {
		return t, err
	}
	if ctxID.Valid {
		v := ctxID.String
		t.ContextID = &v
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
