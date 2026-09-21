package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "goals.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestVacuumInto(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goals.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := s.CreateTask(TaskInput{Title: "snapshot-me"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	snap := filepath.Join(dir, "snap.db")
	if err := s.VacuumInto(snap); err != nil {
		t.Fatalf("vacuum: %v", err)
	}
	raw, err := os.ReadFile(snap)
	if err != nil {
		t.Fatalf("read snap: %v", err)
	}
	if len(raw) == 0 || string(raw[:16]) != "SQLite format 3\x00" {
		t.Fatalf("snapshot is not a sqlite file")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	s2, err := Open(snap)
	if err != nil {
		t.Fatalf("reopen snapshot: %v", err)
	}
	defer s2.Close()
	tasks, err := s2.ListTasks(TaskFilter{Horizon: "all"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "snapshot-me" {
		t.Fatalf("unexpected snapshot content: %+v", tasks)
	}
}

func TestDataDirNotEmpty(t *testing.T) {
	if DataDir() == "" {
		t.Fatal("empty data dir")
	}
}

func TestContextGoalParallelTimers(t *testing.T) {
	s := openTestStore(t)
	// Seed data uses "Work"; use a fresh context as the goal.
	ctx, err := s.CreateContext("WorkGoal", "#3b82f6")
	if err != nil {
		t.Fatalf("create context: %v", err)
	}
	ctx, err = s.UpdateContext(ctx.ID, "WorkGoal", "#3b82f6", 5*3600, 100*3600, "daily", "Deep work & clients", "عمل عميق وعملاء")
	if err != nil {
		t.Fatalf("set goal targets: %v", err)
	}
	if ctx.DailyTarget != 5*3600 || ctx.MaxSeconds != 100*3600 || ctx.Recurrence != "daily" {
		t.Fatalf("targets not saved: %+v", ctx)
	}
	if ctx.Description != "Deep work & clients" || ctx.DescriptionAr != "عمل عميق وعملاء" {
		t.Fatalf("descriptions not saved: %+v", ctx)
	}
	task, err := s.CreateTask(TaskInput{Title: "Write report", ContextID: &ctx.ID})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Both timers may run at the same time (parallel by design).
	if _, err := s.StartContextTimer(ctx.ID); err != nil {
		t.Fatalf("start context: %v", err)
	}
	if _, err := s.StartTimer(task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}
	if _, err := s.GetActiveTimer(); err != nil {
		t.Fatalf("task timer should still run alongside context: %v", err)
	}
	if _, err := s.GetActiveContextTimer(""); err != nil {
		t.Fatalf("context timer should still run alongside task: %v", err)
	}

	time.Sleep(1100 * time.Millisecond)

	if _, err := s.StopTimer(task.ID); err != nil {
		t.Fatalf("stop task: %v", err)
	}
	if _, err := s.StopContextTimer(ctx.ID); err != nil {
		t.Fatalf("stop context: %v", err)
	}

	total, running, err := s.ContextElapsed(ctx.ID)
	if err != nil || running {
		t.Fatalf("elapsed=%d running=%v err=%v", total, running, err)
	}
	if total < 1 {
		t.Fatalf("expected >=1s tracked, got %d", total)
	}
	entries, err := s.ListContextEntries(ctx.ID)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected context entries, got %+v err=%v", entries, err)
	}

	day := time.Now().UTC().Format("2006-01-02")
	today, _, err := s.ContextToday(ctx.ID, day)
	if err != nil || today < 1 {
		t.Fatalf("expected today>=1s, got %d err=%v", today, err)
	}
	got, err := s.GetContext(ctx.ID, day)
	if err != nil {
		t.Fatalf("get context: %v", err)
	}
	if got.TodaySeconds < 1 {
		t.Fatalf("detail todaySeconds=%d, want >=1", got.TodaySeconds)
	}
	if got.TasksTodaySeconds < 1 {
		t.Fatalf("detail tasksTodaySeconds=%d, want >=1 (task time rolls up)", got.TasksTodaySeconds)
	}
	// ListContexts must not deadlock: it enriches rows after closing them
	// (single-connection pool) — this is the app boot path.
	all, err := s.ListContexts(day)
	if err != nil {
		t.Fatalf("list contexts: %v", err)
	}
	found := false
	for _, c := range all {
		if c.ID == ctx.ID {
			found = true
			if c.TodaySeconds < 1 || c.TasksTodaySeconds < 1 {
				t.Fatalf("list detail missing day progress: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("created context missing from list")
	}
	// weekly recurrence persists and reports rolling-week progress
	ctx, err = s.UpdateContext(ctx.ID, "WorkGoal", "#3b82f6", 10*3600, 0, "weekly", "Deep work & clients", "عمل عميق وعملاء")
	if err != nil {
		t.Fatalf("set weekly: %v", err)
	}
	if ctx.Recurrence != "weekly" {
		t.Fatalf("recurrence not saved: %+v", ctx)
	}
	if ctx.WeekSeconds < 1 || ctx.WeekTasksSeconds < 1 {
		t.Fatalf("week progress missing: %+v", ctx)
	}
	// garbage recurrence normalizes to daily
	ctx, err = s.UpdateContext(ctx.ID, "WorkGoal", "#3b82f6", 5*3600, 100*3600, "monthly", "Deep work & clients", "عمل عميق وعملاء")
	if err != nil {
		t.Fatalf("set bogus recurrence: %v", err)
	}
	if ctx.Recurrence != "daily" {
		t.Fatalf("bogus recurrence should normalize to daily: %+v", ctx)
	}

	// Starting a second context timer pauses the first, never the task timer.
	ctx2, err := s.CreateContext("LifeGoal", "#22c55e")
	if err != nil {
		t.Fatalf("create ctx2: %v", err)
	}
	if _, err := s.StartContextTimer(ctx.ID); err != nil {
		t.Fatalf("restart ctx1: %v", err)
	}
	if _, err := s.StartTimer(task.ID); err != nil {
		t.Fatalf("restart task: %v", err)
	}
	if _, err := s.StartContextTimer(ctx2.ID); err != nil {
		t.Fatalf("start ctx2: %v", err)
	}
	if _, err := s.GetActiveTimer(); err != nil {
		t.Fatalf("task timer must survive context switch: %v", err)
	}
	active, err := s.GetActiveContextTimer("")
	if err != nil || active.ID != ctx2.ID {
		t.Fatalf("active context=%+v err=%v, want ctx2", active, err)
	}
	if err := s.CheckpointRunning(); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
}
