package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"goals/internal/drive"
	"goals/internal/store"
	"goals/internal/tray"
	"goals/internal/update"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/gen2brain/beeep"
)

// App struct — bound to Wails frontend + shares store with MCP server.
type App struct {
	ctx   context.Context
	store *store.Store
	dbPath string

	tickMu   sync.Mutex
	tickStop chan struct{}

	overtimeMu       sync.Mutex
	overtimeNotified map[string]bool
}

// NewApp creates a new App application struct
func NewApp(dbPath string) *App {
	return &App{dbPath: dbPath}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	s, err := store.Open(store.ResolveDBPath(a.dbPath))
	if err != nil {
		panic(fmt.Sprintf("failed to open database: %v", err))
	}
	a.store = s
	// safety net: timestamped local backup on every launch (keep last 7)
	go a.autoLocalBackup()
	// resume ticking if a timer was left running (e.g. app restarted)
	if _, err := s.GetActiveTimer(); err == nil {
		a.ensureTick()
	} else if _, err := s.GetActiveContextTimer(""); err == nil {
		a.ensureTick()
	}
}

// shutdown closes db
func (a *App) shutdown(ctx context.Context) {
	a.stopTick()
	tray.Stop()
	if a.store != nil {
		_ = a.store.CheckpointRunning()
		_ = a.store.Close()
	}
}

// QuitApp stops any running timers (saving them) and quits for real.
func (a *App) QuitApp() {
	if a.store != nil {
		if active, err := a.store.GetActiveTimer(); err == nil {
			_, _ = a.store.StopTimer(active.ID)
		}
		if cactive, err := a.store.GetActiveContextTimer(""); err == nil {
			_, _ = a.store.StopContextTimer(cactive.ID)
		}
	}
	runtime.Quit(a.ctx)
}

// ExePath returns the goals.exe path next to the running binary,
// for copy-paste MCP configs.
func (a *App) ExePath() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "goals.exe")
	}
	return "goals.exe"
}

// ---------- Version & self-update ----------

// GetVersion returns the embedded build version ("dev" for local builds;
// CI bakes the release tag in via ldflags).
func (a *App) GetVersion() string {
	return update.Version
}

// CheckForUpdates asks the public GitHub releases for a newer goals.exe.
// force=true bypasses the 5-minute cache.
func (a *App) CheckForUpdates(force bool) (update.Status, error) {
	return update.Check(update.Version, force)
}

// DownloadUpdate fetches the newest goals.exe next to the running one as
// goals.pending.exe. Nothing is replaced yet.
func (a *App) DownloadUpdate() (update.Download, error) {
	return update.DownloadPending(update.Version)
}

// InstallUpdateAndRestart swaps in the previously downloaded update and
// restarts the app. The swap runs in a detached updater script after this
// process exits (Windows locks the running exe).
func (a *App) InstallUpdateAndRestart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	pending := filepath.Join(filepath.Dir(exe), update.PendingName)
	if _, err := os.Stat(pending); err != nil {
		return fmt.Errorf("no downloaded update found - download it first")
	}
	if err := update.StageInstall(pending, os.Getpid()); err != nil {
		return err
	}
	a.QuitApp()
	return nil
}

// autoLocalBackup snapshots the DB into the data dir's backups folder on
// launch and prunes to the newest 7. Best-effort: never fails startup.
func (a *App) autoLocalBackup() {
	defer func() { _ = recover() }()
	if a.store == nil {
		return
	}
	dir := filepath.Join(store.DataDir(), "backups")
	_ = os.MkdirAll(dir, 0755)
	name := fmt.Sprintf("goals-%s.db", time.Now().Format("2006-01-02-150405"))
	_ = a.store.VacuumInto(filepath.Join(dir, name))
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) <= 7 {
		return
	}
	// ReadDir sorts by filename; timestamped names sort chronologically.
	for _, e := range entries[:len(entries)-7] {
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

// ---------- Wails-bound API (also mirrored as MCP tools) ----------

func (a *App) GetDBPath() string {
	if a.store != nil {
		return a.store.Path()
	}
	return store.ResolveDBPath(a.dbPath)
}

func (a *App) ListContexts(day string) ([]store.Context, error) {
	return a.store.ListContexts(day)
}

func (a *App) GetContext(id string, day string) (store.Context, error) {
	return a.store.GetContext(id, day)
}

func (a *App) CreateContext(name string, color string) (store.Context, error) {
	return a.store.CreateContext(name, color)
}

func (a *App) UpdateContext(id string, name string, color string, dailyTargetSeconds int64, maxSeconds int64, recurrence string, description string, descriptionAr string) (store.Context, error) {
	return a.store.UpdateContext(id, name, color, dailyTargetSeconds, maxSeconds, recurrence, description, descriptionAr)
}

func (a *App) DeleteContext(id string) error {
	return a.store.DeleteContext(id)
}

func (a *App) ListHorizons() ([]store.Horizon, error) {
	return a.store.ListHorizons()
}

func (a *App) UpdateHorizon(key string, label string, labelAr string, defaultDays int, description string, descriptionAr string) (store.Horizon, error) {
	return a.store.UpdateHorizon(key, label, labelAr, defaultDays, description, descriptionAr)
}

func (a *App) CreateHorizon(label string, labelAr string, defaultDays int, description string, descriptionAr string) (store.Horizon, error) {
	return a.store.CreateHorizon(label, labelAr, defaultDays, description, descriptionAr)
}

func (a *App) DeleteHorizon(key string) error {
	return a.store.DeleteHorizon(key)
}

func (a *App) ListTasks(filter store.TaskFilter) ([]store.TaskDetail, error) {
	return a.store.ListTasks(filter)
}

func (a *App) GetTask(id string) (store.TaskDetail, error) {
	return a.store.GetTask(id)
}

func (a *App) CreateTask(input store.TaskInput) (store.TaskDetail, error) {
	return a.store.CreateTask(input)
}

func (a *App) UpdateTask(id string, input store.TaskInput) (store.TaskDetail, error) {
	return a.store.UpdateTask(id, input)
}

func (a *App) MoveTask(id string, status string) (store.TaskDetail, error) {
	return a.store.MoveTask(id, status)
}

func (a *App) SetTaskParent(id string, parentID string) (store.TaskDetail, error) {
	var pid *string
	if parentID != "" {
		pid = &parentID
	}
	return a.store.SetTaskParent(id, pid)
}

func (a *App) ReorderTasks(ids []string) error {
	return a.store.ReorderTasks(ids)
}

func (a *App) ToggleFocus(id string) (store.TaskDetail, error) {
	return a.store.ToggleFocus(id)
}

func (a *App) DeleteTask(id string) error {
	return a.store.DeleteTask(id)
}

func (a *App) GetStats() (store.Stats, error) {
	return a.store.GetStats()
}

// ---------- Timer ----------

func (a *App) StartTimer(id string) (store.TaskDetail, error) {
	t, err := a.store.StartTimer(id)
	if err != nil {
		return t, err
	}
	a.resetOvertime(id)
	a.ensureTick()
	a.emitTick()
	return t, nil
}

func (a *App) StopTimer(id string) (store.TaskDetail, error) {
	t, err := a.store.StopTimer(id)
	if err != nil {
		return t, err
	}
	a.resetOvertime(id)
	a.emitTick()
	return t, nil
}

func (a *App) FinishTask(id string) (store.TaskDetail, error) {
	t, err := a.store.FinishTask(id)
	if err != nil {
		return t, err
	}
	a.resetOvertime(id)
	a.emitTick()
	return t, nil
}

func (a *App) GetActiveTimer() (store.TaskDetail, error) {
	return a.store.GetActiveTimer()
}

func (a *App) GetActiveTimers() ([]store.TaskDetail, error) {
	timers, err := a.store.GetActiveTimers()
	if err != nil {
		return nil, err
	}
	if timers == nil {
		timers = []store.TaskDetail{}
	}
	return timers, nil
}

func (a *App) ListTimeEntries(taskID string) ([]store.TimeEntry, error) {
	return a.store.ListTimeEntries(taskID)
}

// ---------- Context goals (parallel timer: may run alongside the task timer) ----------

func (a *App) StartContextTimer(id string) (store.Context, error) {
	c, err := a.store.StartContextTimer(id)
	if err != nil {
		return c, err
	}
	a.resetOvertime("ctx:" + id)
	a.ensureTick()
	a.emitTick()
	return c, nil
}

func (a *App) StopContextTimer(id string) (store.Context, error) {
	c, err := a.store.StopContextTimer(id)
	if err != nil {
		return c, err
	}
	a.resetOvertime("ctx:" + id)
	a.emitTick()
	return c, nil
}

func (a *App) GetActiveContextTimer(day string) (store.Context, error) {
	return a.store.GetActiveContextTimer(day)
}

func (a *App) ListContextEntries(contextID string) ([]store.ContextTimeEntry, error) {
	return a.store.ListContextEntries(contextID)
}

func (a *App) GetContextToday(id string, day string) (int64, error) {
	total, _, err := a.store.ContextToday(id, day)
	return total, err
}

func (a *App) GetSettings() (map[string]string, error) {
	return a.store.GetSettings()
}

func (a *App) SetSetting(key string, value string) error {
	return a.store.SetSetting(key, value)
}

// ReportError appends a frontend error to a temp log for diagnostics.
func (a *App) ReportError(message string) error {
	p := os.TempDir()
	f, err := os.OpenFile(filepath.Join(p, "goals-error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), message)
	return err
}

// ---------- Data folder, import, Google Drive ----------

func (a *App) GetDataDir() string {
	return store.DataDir()
}

// OpenDataFolder reveals the folder holding goals.db in Explorer.
func (a *App) OpenDataFolder() (string, error) {
	dir := store.DataDir()
	_ = os.MkdirAll(dir, 0755)
	if err := exec.Command("explorer", dir).Start(); err != nil {
		return dir, err
	}
	return dir, nil
}

// swapDatabase atomically replaces the live database file. The store is
// closed first (flushing WAL), then reopened on the new file.
func (a *App) swapDatabase(data []byte) error {
	if !drive.ValidSQLite(data) {
		return fmt.Errorf("file is not a valid Goals database")
	}
	a.stopTick()
	path := a.store.Path()
	if err := a.store.CheckpointRunning(); err != nil {
		return err
	}
	if err := a.store.Close(); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	for _, ext := range []string{"-wal", "-shm", "-journal"} {
		_ = os.Remove(path + ext)
	}
	s, err := store.Open(path)
	if err != nil {
		return err
	}
	a.store = s
	if _, err := s.GetActiveTimer(); err == nil {
		a.ensureTick()
	} else if _, err := s.GetActiveContextTimer(""); err == nil {
		a.ensureTick()
	}
	return nil
}

// ImportDatabaseFile replaces the current DB with a local .db file.
func (a *App) ImportDatabaseFile(srcPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return a.swapDatabase(data)
}

// PickDatabaseFile opens a file picker for a .db file. Empty string = cancelled.
func (a *App) PickDatabaseFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Goals database",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite Database (*.db)", Pattern: "*.db"},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
}

func (a *App) GetDriveStatus() (drive.Status, error) {
	return drive.GetStatus(a.store)
}

func (a *App) SaveDriveCredentials(clientID string, clientSecret string) error {
	return drive.SaveCredentials(a.store, clientID, clientSecret)
}

func (a *App) StartDriveAuth() (drive.DeviceAuth, error) {
	return drive.StartDeviceAuth(a.store)
}

func (a *App) PollDriveAuth() (drive.DevicePoll, error) {
	return drive.PollDeviceAuth(a.store)
}

func (a *App) CancelDriveAuth() {
	drive.CancelAuthFlow()
}

func (a *App) DisconnectDrive() error {
	return drive.Disconnect(a.store)
}

func (a *App) BackupNow() (drive.BackupFile, error) {
	return drive.BackupNow(a.store)
}

func (a *App) ListDriveBackups() ([]drive.BackupFile, error) {
	out, err := drive.ListBackups(a.store)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []drive.BackupFile{}
	}
	return out, nil
}

func (a *App) DeleteDriveBackup(id string) error {
	return drive.DeleteBackup(a.store, id)
}

// RestoreDriveBackup downloads a backup and swaps it in as the live DB.
// The frontend should reload afterwards (WindowReload).
func (a *App) RestoreDriveBackup(id string) error {
	data, err := drive.DownloadBackup(a.store, id)
	if err != nil {
		return err
	}
	return a.swapDatabase(data)
}

// ensureTick runs a 1s loop while a timer is active: updates the window
// title and notifies the frontend. It survives window hide (background mode).
func (a *App) ensureTick() {
	a.tickMu.Lock()
	defer a.tickMu.Unlock()
	if a.tickStop != nil {
		return
	}
	stop := make(chan struct{})
	a.tickStop = stop
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		n := 0
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				n++
				if !a.tickOnce(n) {
					a.tickMu.Lock()
					if a.tickStop == stop {
						a.tickStop = nil
					}
					a.tickMu.Unlock()
					return
				}
			}
		}
	}()
}

// tickOnce returns false when no timer is running (loop should stop).
// Task and context-goal timers are independent and may both run at once.
// Parent+subtask task timers also run in parallel.
func (a *App) tickOnce(n int) bool {
	if a.store == nil {
		return false
	}
	actives, _ := a.store.GetActiveTimers()
	cactive, ctxErr := a.store.GetActiveContextTimer("")
	taskErr := error(nil)
	var active store.TaskDetail
	if len(actives) == 0 {
		taskErr = fmt.Errorf("no active task timer")
	} else {
		active = actives[0]
	}
	if taskErr != nil && ctxErr != nil {
		runtime.WindowSetTitle(a.ctx, "Goals")
		tray.SetTooltip("Goals")
		return false
	}
	if n%30 == 0 {
		_ = a.store.CheckpointRunning()
	}
	title := "Goals"
	over := false
	if taskErr == nil {
		elapsed, _, _ := a.store.Elapsed(active.ID)
		title = fmt.Sprintf("\u23F1 %s \u00B7 %s", formatHMS(elapsed), active.Title)
		if len(actives) > 1 {
			title = fmt.Sprintf("\u23F1 %s \u00B7 %s (+%d)", formatHMS(elapsed), active.Title, len(actives)-1)
		}
		for _, t := range actives {
			el, _, _ := a.store.Elapsed(t.ID)
			if t.MaxSeconds > 0 && el >= t.MaxSeconds {
				over = true
				a.notifyOvertimeOnce(t.ID, t.Title, el, t.MaxSeconds)
			}
		}
		runtime.EventsEmit(a.ctx, "timer:tick", map[string]any{
			"taskId":  active.ID,
			"title":   active.Title,
			"elapsed": elapsed,
			"count":   len(actives),
		})
		runtime.EventsEmit(a.ctx, "timers:tick", actives)
	} else {
		runtime.EventsEmit(a.ctx, "timer:tick", nil)
	}
	if ctxErr == nil {
		celapsed, _, _ := a.store.ContextElapsed(cactive.ID)
		part := fmt.Sprintf("\U0001F3AF %s \u00B7 %s", formatHMS(celapsed), cactive.Name)
		if taskErr == nil {
			title = title + "  |  " + part
		} else {
			title = part
		}
		if cactive.MaxSeconds > 0 && celapsed >= cactive.MaxSeconds {
			over = true
			a.notifyOvertimeOnce("ctx:"+cactive.ID, cactive.Name, celapsed, cactive.MaxSeconds)
		}
		// daily-target toast: fires once when today's total passes the target
		if cactive.DailyTarget > 0 {
			if today, _, _ := a.store.ContextToday(cactive.ID, time.Now().UTC().Format("2006-01-02")); today >= cactive.DailyTarget {
				a.notifyOvertimeOnce("ctxday:"+cactive.ID+":"+time.Now().UTC().Format("2006-01-02"), cactive.Name+" — daily target reached", today, cactive.DailyTarget)
			}
		}
		runtime.EventsEmit(a.ctx, "context:tick", map[string]any{
			"contextId": cactive.ID,
			"title":     cactive.Name,
			"elapsed":   celapsed,
		})
	} else {
		runtime.EventsEmit(a.ctx, "context:tick", nil)
	}
	if over {
		title = "\u26A0 " + title
	}
	runtime.WindowSetTitle(a.ctx, title)
	tray.SetTooltip("Goals — " + title)
	return true
}

func (a *App) emitTick() {
	a.tickMu.Lock()
	running := a.tickStop != nil
	a.tickMu.Unlock()
	if running {
		return
	}
	// one-shot refresh so the UI picks up the change immediately
	if a.store == nil {
		return
	}
	if active, err := a.store.GetActiveTimer(); err == nil {
		elapsed, _, _ := a.store.Elapsed(active.ID)
		runtime.EventsEmit(a.ctx, "timer:tick", map[string]any{"taskId": active.ID, "title": active.Title, "elapsed": elapsed})
	} else {
		runtime.EventsEmit(a.ctx, "timer:tick", nil)
	}
	if oactive, err := a.store.GetActiveContextTimer(""); err == nil {
		oelapsed, _, _ := a.store.ContextElapsed(oactive.ID)
		runtime.EventsEmit(a.ctx, "context:tick", map[string]any{"contextId": oactive.ID, "title": oactive.Name, "elapsed": oelapsed})
	} else {
		runtime.EventsEmit(a.ctx, "context:tick", nil)
	}
	if _, terr := a.store.GetActiveTimer(); terr != nil {
		if _, oerr := a.store.GetActiveContextTimer(""); oerr != nil {
			runtime.WindowSetTitle(a.ctx, "Goals")
			tray.SetTooltip("Goals")
		}
	}
	_ = running
}

func (a *App) stopTick() {
	a.tickMu.Lock()
	defer a.tickMu.Unlock()
	if a.tickStop != nil {
		close(a.tickStop)
		a.tickStop = nil
	}
}

func (a *App) resetOvertime(id string) {
	a.overtimeMu.Lock()
	defer a.overtimeMu.Unlock()
	if a.overtimeNotified == nil {
		a.overtimeNotified = map[string]bool{}
		return
	}
	delete(a.overtimeNotified, id)
}

// notifyOvertimeOnce fires a single OS toast the first time a running timer
// passes its maxSeconds estimate.
func (a *App) notifyOvertimeOnce(id, title string, elapsed, max int64) {
	a.overtimeMu.Lock()
	if a.overtimeNotified == nil {
		a.overtimeNotified = map[string]bool{}
	}
	if a.overtimeNotified[id] {
		a.overtimeMu.Unlock()
		return
	}
	a.overtimeNotified[id] = true
	a.overtimeMu.Unlock()

	msg := fmt.Sprintf("%s\nExpected %s — now at %s. You've exceeded your estimate.", title, formatHMS(max), formatHMS(elapsed))
	go func() {
		_ = beeep.Notify("Goals — max time finished", msg, "")
	}()
	runtime.EventsEmit(a.ctx, "timer:overtime", map[string]any{
		"taskId": id, "title": title, "elapsed": elapsed, "max": max,
	})
}

func formatHMS(s int64) string {
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}
