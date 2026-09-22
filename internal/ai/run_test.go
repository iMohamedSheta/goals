//go:build windows

package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

// A hung child must never wedge runCmd: cancel/timeout has to return
// promptly (stdio goes to temp files, so no pipe can block Wait; the tree
// is swept via taskkill).
func TestRunCmdTimeoutKillsTree(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	_, _, err := runCmd(ctx, "cmd", false,
		[]string{"/D", "/C", "ping", "-n", "60", "127.0.0.1"},
		"", nil)
	took := time.Since(start)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if took > 20*time.Second {
		t.Fatalf("runCmd took %v, wedged", took)
	}
	t.Logf("returned after %v with err=%v", took, err)
}

// Normal path still captures output.
func TestRunCmdCapturesOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stdout, _, err := runCmd(ctx, "cmd", false, []string{"/D", "/C", "echo", "hello-ai"}, "", nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(string(stdout), "hello-ai") {
		t.Fatalf("unexpected stdout %q", stdout)
	}
}
