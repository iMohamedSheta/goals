package llm

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"goals/internal/mcp"
	"goals/internal/store"
)

func TestKeyFromAuthBytes(t *testing.T) {
	raw := []byte(`{"openrouter":{"type":"api","key":"sk-or-0123456789abcdef0123456789abcdef"},"other":{"type":"x","key":"short"}}`)
	got := keyFromAuthBytes(raw, "openrouter")
	if !strings.HasPrefix(got, "sk-or-") {
		t.Fatalf("got %q", got)
	}
	if keyFromAuthBytes(raw, "missing") != "" {
		t.Fatalf("expected empty for missing provider")
	}
	if keyFromAuthBytes([]byte("nope"), "openrouter") != "" {
		t.Fatalf("expected empty for bad json")
	}
}

func TestDirectCapable(t *testing.T) {
	for _, m := range []string{"google/gemini-flash", "openrouter/anthropic/x", "anthropic/claude"} {
		if !directCapable(m) {
			t.Fatalf("%q should be direct-capable", m)
		}
	}
	for _, m := range []string{"", "opencode/big", "plainmodel"} {
		if directCapable(m) {
			t.Fatalf("%q should NOT be direct-capable", m)
		}
	}
}

func TestPickSlug(t *testing.T) {
	ids := []string{"google/gemini-2.0-flash-001", "google/gemini-2.0-flash-lite-001", "anthropic/claude-3-5-haiku-20241022", "openai/gpt-4o-mini"}
	if got := pickSlug("openrouter/anthropic/claude-3-5-haiku-20241022", ids); got != "anthropic/claude-3-5-haiku-20241022" {
		t.Fatalf("exact/strip match: %q", got)
	}
	if got := pickSlug("google/gemini-9.9-flash-lite-xyz", ids); got != "google/gemini-2.0-flash-lite-001" {
		t.Fatalf("family fallback: %q", got)
	}
	if got := pickSlug("weirdvendor/nonexistent-thing", ids); got != "" {
		t.Fatalf("unknown vendor should be empty: %q", got)
	}
}

func TestOpenAIToolsShape(t *testing.T) {
	tools := openAITools()
	if len(tools) < 20 {
		t.Fatalf("only %d tools", len(tools))
	}
	raw, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back []map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	for _, tl := range back {
		fn := tl["function"].(map[string]any)
		params := fn["parameters"].(map[string]any)
		if params["type"] != "object" {
			t.Fatalf("tool %v params not object", fn["name"])
		}
	}
}

func TestCallSmoke(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "goals.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ctx, err := s.CreateContext("DirectSmoke", "#fff")
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	res, unknown := mcp.Call(s, "create_task", map[string]any{"title": "direct smoke", "contextId": ctx.ID})
	if unknown {
		t.Fatalf("create_task unknown")
	}
	if toolText(res) == "" {
		t.Fatalf("empty create result")
	}
	res, unknown = mcp.Call(s, "list_tasks", map[string]any{"contextId": ctx.ID})
	if unknown {
		t.Fatalf("list_tasks unknown")
	}
	if !strings.Contains(toolText(res), "direct smoke") {
		t.Fatalf("task not listed: %s", toolText(res))
	}
	if _, unknown := mcp.Call(s, "nope", map[string]any{}); !unknown {
		t.Fatalf("expected unknown tool")
	}
}
