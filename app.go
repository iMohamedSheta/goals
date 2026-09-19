package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"goals/internal/store"
	"goals/internal/tray"

	"github.com/gen2brain/beeep"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	// resume ticking if a timer was left running (e.g. app restarted)
	if _, err := s.GetActiveTimer(); err == nil {
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

// QuitApp stops any running timer (saving it) and quits for real.
func (a *App) QuitApp() {
	if a.store != nil {
		if active, err := a.store.GetActiveTimer(); err == nil {
			_, _ = a.store.StopTimer(active.ID)
		}
	}
	runtime.Quit(a.ctx)
}

// ---------- Wails-bound API (also mirrored as MCP tools) ----------

func (a *App) GetDBPath() string {
	if a.store != nil {
		return a.store.Path()
	}
	return store.ResolveDBPath(a.dbPath)
}

func (a *App) ListContexts() ([]store.Context, error) {
	return a.store.ListContexts()
}

func (a *App) CreateContext(name string, color string) (store.Context, error) {
	return a.store.CreateContext(name, color)
}

func (a *App) UpdateContext(id string, name string, color string) (store.Context, error) {
	return a.store.UpdateContext(id, name, color)
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

func (a *App) ListTimeEntries(taskID string) ([]store.TimeEntry, error) {
	return a.store.ListTimeEntries(taskID)
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
func (a *App) tickOnce(n int) bool {
	if a.store == nil {
		return false
	}
	active, err := a.store.GetActiveTimer()
	if err != nil {
		runtime.WindowSetTitle(a.ctx, "Goals")
		tray.SetTooltip("Goals")
		return false
	}
	elapsed, _, _ := a.store.Elapsed(active.ID)
	if n%30 == 0 {
		_ = a.store.CheckpointRunning()
	}
	title := fmt.Sprintf("\u23F1 %s \u00B7 %s", formatHMS(elapsed), active.Title)
	overtime := active.MaxSeconds > 0 && elapsed >= active.MaxSeconds
	if overtime {
		title = "\u26A0 " + title
	}
	runtime.WindowSetTitle(a.ctx, title)
	tray.SetTooltip("Goals — " + title)
	if overtime {
		a.notifyOvertimeOnce(active.ID, active.Title, elapsed, active.MaxSeconds)
	}
	runtime.EventsEmit(a.ctx, "timer:tick", map[string]any{
		"taskId":  active.ID,
		"title":   active.Title,
		"elapsed": elapsed,
	})
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
		runtime.WindowSetTitle(a.ctx, "Goals")
		tray.SetTooltip("Goals")
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
