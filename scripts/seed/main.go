// Command seed writes a fresh demo goals.db with sample tasks.
//
// It is used to screenshot the app with realistic content (locally into
// docs/screenshot.png and in the release workflow before capturing
// screenshot.png for the GitHub Release).
//
// Usage: go run ./scripts/seed -db <path-to-goals.db>
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"goals/internal/store"
)

func strp(s string) *string { return &s }

func main() {
	dbPath := flag.String("db", "", "destination goals.db path (created fresh)")
	flag.Parse()
	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "seed: -db is required")
		os.Exit(2)
	}
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "seed: mkdir:", err)
		os.Exit(1)
	}
	_ = os.Remove(*dbPath)

	s, err := store.Open(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed: open:", err)
		os.Exit(1)
	}
	defer s.Close()

	ctxIDs := map[string]string{}
	if ctxs, err := s.ListContexts(); err == nil {
		for _, c := range ctxs {
			ctxIDs[c.Name] = c.ID
		}
	}

	tasks := []store.TaskInput{
		{Title: "إطلاق النسخة الجديدة", Description: "تجهيز ملاحظات الإصدار والنشر", Horizon: "short", Status: "todo", Priority: "high", Focus: true},
		{Title: "مراجعة التقرير الأسبوعي", Horizon: "short", Status: "in_progress", Priority: "medium"},
		{Title: "جلسة رياضية صباحية", Horizon: "short", Status: "done", Priority: "low"},
		{Title: "حجز إجازة الصيف", Horizon: "medium", Status: "todo", Priority: "medium"},
		{Title: "خطة التوسع السنوية", Horizon: "long", Status: "todo", Priority: "urgent"},
	}
	for i := range tasks {
		if i < 2 {
			if id, ok := ctxIDs["Work"]; ok {
				tasks[i].ContextID = strp(id)
			}
		} else {
			if id, ok := ctxIDs["Life"]; ok {
				tasks[i].ContextID = strp(id)
			}
		}
		if _, err := s.CreateTask(tasks[i]); err != nil {
			fmt.Fprintln(os.Stderr, "seed: create task:", err)
			os.Exit(1)
		}
	}
	fmt.Println("seed: wrote", *dbPath)
}
