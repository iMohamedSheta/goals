// Package llm talks to AI providers directly over HTTPS — no local CLI
// process, no bun runtime, no RAM spikes, no console popups. The assistant
// keeps full task powers via an agentic tool loop executed against the same
// dispatcher as the MCP server (mcp.Call).
package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// authEntry mirrors one entry of opencode's auth.json:
// {"provider": {"type": "...", "key": "..."}}.
type authEntry struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

func keyFromAuthBytes(raw []byte, provider string) string {
	var all map[string]authEntry
	if err := json.Unmarshal(raw, &all); err != nil {
		return ""
	}
	if e, ok := all[strings.ToLower(strings.TrimSpace(provider))]; ok {
		if len(e.Key) > 20 {
			return e.Key
		}
	}
	return ""
}

// OpenRouterKey returns the OpenRouter API key: OPENROUTER_API_KEY wins,
// then the key stored by `opencode auth login`. Empty = not configured.
// The value is never logged.
func OpenRouterKey() string {
	if k := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")); len(k) > 20 {
		return k
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(home, ".local", "share", "opencode", "auth.json"))
	if err != nil {
		return ""
	}
	return keyFromAuthBytes(raw, "openrouter")
}

// Configured reports whether direct HTTPS mode is available.
func Configured() bool {
	return OpenRouterKey() != ""
}
