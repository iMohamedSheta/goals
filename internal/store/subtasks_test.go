package store

import (
	"testing"
	"time"
)

func TestSubtasksNestReorderTimer(t *testing.T) {
	s := openTestStore(t)
	parent, err := s.CreateTask(TaskInput{Title: "Parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child, err := s.CreateTask(TaskInput{Title: "Child", ParentID: &parent.ID})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatalf("child parent not saved: %+v", child)
	}
	if child.Horizon != parent.Horizon {
		t.Fatalf("child should adopt parent horizon: %+v vs %+v", child.Horizon, parent.Horizon)
	}
	// cycle guard
	if _, err := s.SetTaskParent(parent.ID, &child.ID); err == nil {
		t.Fatal("nesting parent inside its own child should fail")
	}
	if _, err := s.SetTaskParent(parent.ID, &parent.ID); err == nil {
		t.Fatal("self-parent should fail")
	}
	// timer inheritance: starting child also starts parent
	if _, err := s.StartTimer(child.ID); err != nil {
		t.Fatalf("start child: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, running, _ := s.Elapsed(parent.ID); !running {
		t.Fatal("parent timer should run when child starts")
	}
	if _, running, _ := s.Elapsed(child.ID); !running {
		t.Fatal("child timer should run")
	}
	actives, err := s.GetActiveTimers()
	if err != nil || len(actives) != 2 {
		t.Fatalf("expected 2 active timers, got %+v err=%v", actives, err)
	}
	// sibling switch pauses previous sibling but keeps parent
	child2, err := s.CreateTask(TaskInput{Title: "Child2", ParentID: &parent.ID})
	if err != nil {
		t.Fatalf("create child2: %v", err)
	}
	if _, err := s.StartTimer(child2.ID); err != nil {
		t.Fatalf("start child2: %v", err)
	}
	if _, running, _ := s.Elapsed(child.ID); running {
		t.Fatal("previous sibling should pause on switch")
	}
	if _, running, _ := s.Elapsed(child2.ID); !running {
		t.Fatal("child2 should run")
	}
	if _, running, _ := s.Elapsed(parent.ID); !running {
		t.Fatal("parent should keep running on sibling switch")
	}
	// reorder persists
	all, err := s.ListTasks(TaskFilter{Horizon: "all"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	ids := []string{}
	for _, x := range all {
		ids = append(ids, x.ID)
	}
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}
	if err := s.ReorderTasks(ids); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	// delete promotes children to top-level
	if err := s.DeleteTask(parent.ID); err != nil {
		t.Fatalf("delete parent: %v", err)
	}
	got, err := s.GetTask(child.ID)
	if err != nil {
		t.Fatalf("get promoted child: %v", err)
	}
	if got.ParentID != nil {
		t.Fatalf("child should be top-level after parent delete: %+v", got.ParentID)
	}
}
