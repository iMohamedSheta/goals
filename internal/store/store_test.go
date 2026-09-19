package store

import (
	"os"
	"path/filepath"
	"testing"
)

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
