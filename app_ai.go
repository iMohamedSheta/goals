package main

import (
	"strconv"
	"strings"
	"time"

	"goals/internal/ai"
	"goals/internal/store"
)

// ---------- In-app AI chat (powered by the local OpenCode CLI) ----------

// AIStatus reports whether `opencode` is usable for chat.
func (a *App) AIStatus() (ai.Status, error) {
	return ai.Check(), nil
}

// AIModels lists models from `opencode models` (falls back to defaults).
func (a *App) AIModels() ([]string, error) {
	return ai.Models(), nil
}

// AIBusy reports whether an AI reply is currently streaming/running.
func (a *App) AIBusy() (bool, error) {
	return ai.Busy(), nil
}

// GetAIModel returns the saved model ("ai.model" setting, "" = auto).
func (a *App) GetAIModel() (string, error) {
	if a.store == nil {
		return "", nil
	}
	m, err := a.store.GetSettings()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(m["ai.model"]), nil
}

// SetAIModel saves the preferred model ("ai.model", "" = auto).
func (a *App) SetAIModel(model string) error {
	if a.store == nil {
		return nil
	}
	return a.store.SetSetting("ai.model", strings.TrimSpace(model))
}

// AskAI sends one command prompt to the local `opencode run` and returns the
// reply. Every call is an independent one-shot (no session resume), so each
// answer stands alone. Empty model ("auto") uses the cached fastest model
// when one was measured, else a known-good default (see ai.DefaultModel)
// because opencode's global default is usually a gated opencode/* model.
// Runs in workDir = data dir so sessions are per-user; the goals MCP server
// (this exe) is injected via OPENCODE_CONFIG_CONTENT so the assistant can
// read/manage tasks.
func (a *App) AskAI(input ai.AskRequest) (ai.AskResponse, error) {
	model := strings.TrimSpace(input.Model)
	auto := model == ""
	if auto && a.store != nil {
		if m, err := a.store.GetSettings(); err == nil {
			model = strings.TrimSpace(m["ai.model"])
			if model == "" {
				// No picked model: reuse the fastest one measured so far.
				model = strings.TrimSpace(m["ai.bestmodel"])
			}
		}
	}
	input.Model = model
	// One-shot commands never resume sessions (avoids stale-session empties).
	input.SessionID = ""
	start := time.Now()
	resp, err := ai.Ask(store.DataDir(), a.ExePath(), input)
	if err != nil {
		return ai.AskResponse{}, err
	}
	if a.store != nil {
		if model != "" && !auto {
			_ = a.store.SetSetting("ai.model", model)
		}
		// Remember the fastest successful model for future auto picks.
		if ms := time.Since(start).Milliseconds(); ms > 0 {
			if m, err := a.store.GetSettings(); err == nil {
				if best, _ := strconv.ParseInt(strings.TrimSpace(m["ai.bestms"]), 10, 64); best <= 0 || ms < best {
					_ = a.store.SetSetting("ai.bestmodel", resp.Model)
					_ = a.store.SetSetting("ai.bestms", strconv.FormatInt(ms, 10))
				}
			}
		}
	}
	return resp, nil
}

// CancelAI stops the in-flight AskAI call, if any.
func (a *App) CancelAI() {
	ai.Cancel()
}
