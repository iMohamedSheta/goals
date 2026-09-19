package main

import (
	"strings"

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

// AskAI sends one chat prompt to the local `opencode run` and returns the
// reply. Pass back response.SessionID to continue the same conversation.
// Empty model ("auto") resolves to a known-good default (see ai.DefaultModel)
// because opencode's global default is usually a gated opencode/* model.
// Runs in workDir = data dir so sessions are per-user; the goals MCP server
// (this exe) is injected via OPENCODE_CONFIG_CONTENT so the assistant can
// read/manage tasks.
func (a *App) AskAI(input ai.AskRequest) (ai.AskResponse, error) {
	model := strings.TrimSpace(input.Model)
	if model == "" && a.store != nil {
		if m, err := a.store.GetSettings(); err == nil {
			model = strings.TrimSpace(m["ai.model"])
		}
	}
	input.Model = model
	resp, err := ai.Ask(store.DataDir(), a.ExePath(), input)
	if err != nil {
		return ai.AskResponse{}, err
	}
	if model != "" && a.store != nil {
		_ = a.store.SetSetting("ai.model", model)
	}
	return resp, nil
}

// CancelAI stops the in-flight AskAI call, if any.
func (a *App) CancelAI() {
	ai.Cancel()
}
