// Package ai bridges the Goals database to an AI chat backend.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Status describes whether the local OpenCode CLI can be used.
type Status struct {
	Available     bool   `json:"available"`
	Binary        string `json:"binary"`
	Version       string `json:"version"`
	Authenticated bool   `json:"authenticated"`
	Hint          string `json:"hint"`
}

// AskRequest is a single chat turn. SessionID empty = new session.
type AskRequest struct {
	Prompt    string `json:"prompt"`
	SessionID string `json:"sessionID"`
	Model     string `json:"model"`
}

// AskResponse holds the assistant text and the session to continue with.
type AskResponse struct {
	Reply     string `json:"reply"`
	SessionID string `json:"sessionID"`
	Model     string `json:"model"`
}

// DefaultModel is used when the user picked "auto" (no saved model).
// google/gemini-3.5-flash-lite is cheap and works with a plain Google key,
// while opencode/* models are gated: free-tier refuses non-TUI clients and
// paid ones need a billing method on the workspace.
const DefaultModel = "google/gemini-3.5-flash-lite"

// DefaultModels is a fallback list when `opencode models` fails.
// Keep it short: cheap + known to work with stored credentials.
var DefaultModels = []string{
	"google/gemini-3.5-flash-lite",
	"google/gemini-3.5-flash",
	"openrouter/anthropic/claude-haiku-latest",
	"openrouter/google/gemini-flash-latest",
}

const (
	runTimeout     = 180 * time.Second
	modelsTimeout  = 15 * time.Second
	versionTimeout = 10 * time.Second
)

// ---------- binary resolution ----------

// resolveBinary prefers a real opencode.exe on PATH so Go can exec it
// directly. npm's global install only puts opencode.cmd on PATH, which
// needs cmd.exe wrapping (CreateProcess can't run .cmd files).
func resolveBinary() (bin string, viaShell bool, err error) {
	// 1. Prefer <somedrive>:\...\opencode.exe found on PATH.
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		cand := filepath.Join(dir, "opencode.exe")
		if st, e := os.Stat(cand); e == nil && !st.IsDir() {
			return cand, false, nil
		}
	}
	// 2. Explicit well-known install location (opencode managed install).
	if home, e := os.UserHomeDir(); e == nil {
		cand := filepath.Join(home, ".opencode", "bin", "opencode.exe")
		if st, e := os.Stat(cand); e == nil && !st.IsDir() {
			return cand, false, nil
		}
	}
	// 3. Fallback to whatever LookPath finds (often opencode.cmd via npm).
	if p, e := exec.LookPath("opencode"); e == nil {
		lower := strings.ToLower(p)
		if strings.HasSuffix(lower, ".cmd") || strings.HasSuffix(lower, ".bat") {
			return p, true, nil
		}
		return p, false, nil
	}
	return "", false, fmt.Errorf("opencode CLI not found on PATH — install it from https://opencode.ai")
}

func runCmd(ctx context.Context, bin string, viaShell bool, args []string, dir string, extraEnv []string) (stdout, stderr []byte, err error) {
	var cmd *exec.Cmd
	if viaShell {
		full := append([]string{"/D", "/C", bin}, args...)
		cmd = exec.CommandContext(ctx, "cmd", full...)
	} else {
		cmd = exec.CommandContext(ctx, bin, args...)
	}
	hideConsole(cmd)
	if dir != "" {
		cmd.Dir = dir
	}
	if extraEnv != nil {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// Version returns e.g. "1.18.31" or "" when it can't be determined.
func Version() string {
	bin, viaShell, err := resolveBinary()
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), versionTimeout)
	defer cancel()
	out, _, err := runCmd(ctx, bin, viaShell, []string{"--version"}, "", nil)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Authenticated reports whether an opencode credentials file exists.
// It does not validate keys — a bad key surfaces as an API error on Ask.
func Authenticated() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	p := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	st, err := os.Stat(p)
	if err != nil || st.IsDir() || st.Size() < 10 {
		return false
	}
	return true
}

// Check probes the local CLI. Never errors — encodes problems in Hint.
func Check() Status {
	bin, _, err := resolveBinary()
	if err != nil {
		return Status{Available: false, Hint: err.Error()}
	}
	v := Version()
	auth := Authenticated()
	hint := ""
	if !auth {
		hint = "No opencode credentials found — run `opencode auth login` once in a terminal."
	}
	return Status{Available: true, Binary: bin, Version: v, Authenticated: auth, Hint: hint}
}

// Models lists available provider/model names, falling back to DefaultModels.
func Models() []string {
	bin, viaShell, err := resolveBinary()
	if err != nil {
		return append([]string(nil), DefaultModels...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), modelsTimeout)
	defer cancel()
	out, _, err := runCmd(ctx, bin, viaShell, []string{"models"}, "", nil)
	if err != nil {
		return append([]string(nil), DefaultModels...)
	}
	var got []string
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// `opencode models` prints one `provider/model` per line; skip tables/notes.
		if !strings.Contains(line, "/") || strings.ContainsAny(line, " \t") {
			continue
		}
		got = append(got, line)
		if len(got) >= 200 {
			break
		}
	}
	if len(got) == 0 {
		return append([]string(nil), DefaultModels...)
	}
	return got
}

// mcpConfigEnv injects our goals MCP server into the child opencode process
// so the assistant can read/manage tasks even when the working directory has
// no opencode.json (e.g. production builds running from the data dir).
// Empty exePath disables the injection.
func mcpConfigEnv(exePath string) []string {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return nil
	}
	cfg := map[string]any{
		"mcp": map[string]any{
			"goals": map[string]any{
				"type":    "local",
				"command": []string{exePath, "mcp"},
				"enabled": true,
			},
		},
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil
	}
	return []string{"OPENCODE_CONFIG_CONTENT=" + string(raw)}
}

// ---------- single-flight runner with cancel ----------

var runnerMu sync.Mutex
var runnerCancel context.CancelFunc
var runnerBusy bool

// Busy reports whether an Ask is currently running.
func Busy() bool {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	return runnerBusy
}

// Cancel stops the in-flight Ask, if any. No-op when idle.
func Cancel() {
	runnerMu.Lock()
	cancel := runnerCancel
	runnerMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Ask sends one prompt to `opencode run --format json` and returns the
// assistant text plus the session id for continuing the conversation.
// workDir is the directory opencode runs in (use the app data dir).
// model empty = opencode's configured default. Only one Ask runs at a time.
func Ask(workDir, exePath string, req AskRequest) (AskResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return AskResponse{}, fmt.Errorf("prompt is empty")
	}
	// "auto" resolves to a known-good default instead of opencode's global
	// default, which is usually an opencode/* model gated behind the TUI
	// or a workspace billing method.
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = DefaultModel
	}

	runnerMu.Lock()
	if runnerBusy {
		runnerMu.Unlock()
		return AskResponse{}, fmt.Errorf("an AI reply is already in progress — wait or cancel it first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	runnerCancel = cancel
	runnerBusy = true
	runnerMu.Unlock()
	defer func() {
		cancel()
		runnerMu.Lock()
		runnerCancel = nil
		runnerBusy = false
		runnerMu.Unlock()
	}()

	bin, viaShell, err := resolveBinary()
	if err != nil {
		return AskResponse{}, err
	}

	args := []string{"run", "--format", "json"}
	if model != "" {
		args = append(args, "--model", model)
	}
	if sid := strings.TrimSpace(req.SessionID); sid != "" {
		args = append(args, "--session", sid, "--continue")
	}
	args = append(args, prompt)

	stdout, stderr, err := runCmd(ctx, bin, viaShell, args, workDir, mcpConfigEnv(exePath))
	reply, sessionID, apiErr := parseRunJSON(stdout)
	if ctx.Err() == context.DeadlineExceeded {
		return AskResponse{}, fmt.Errorf("opencode timed out after %s", runTimeout)
	}
	if ctx.Err() == context.Canceled {
		return AskResponse{}, fmt.Errorf("cancelled")
	}
	if err != nil {
		// opencode exits non-zero on API errors; prefer the parsed message.
		if apiErr != "" {
			return AskResponse{}, fmt.Errorf("%s", friendlyError(apiErr))
		}
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = strings.TrimSpace(string(stdout))
		}
		if len(detail) > 500 {
			detail = detail[:500] + "…"
		}
		if detail == "" {
			detail = err.Error()
		}
		return AskResponse{}, fmt.Errorf("opencode failed: %s", detail)
	}
	if apiErr != "" {
		return AskResponse{}, fmt.Errorf("%s", friendlyError(apiErr))
	}
	if strings.TrimSpace(reply) == "" {
		return AskResponse{}, fmt.Errorf("opencode returned an empty reply")
	}
	return AskResponse{Reply: strings.TrimSpace(reply), SessionID: sessionID, Model: model}, nil
}

// friendlyError appends actionable guidance to known provider rejections
// so the chat shows what to do instead of a raw API message.
func friendlyError(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "free tier"):
		return msg + " — OpenCode free-tier models only work inside the OpenCode TUI. " +
			"Pick a Google/OpenRouter model from the model picker in the chat (e.g. google/gemini-3.5-flash-lite)."
	case strings.Contains(lower, "payment") || strings.Contains(lower, "billing") || strings.Contains(lower, "credits"):
		return msg + " — That model needs billing on your provider workspace. " +
			"Pick a model from a provider you authenticated (Google/OpenRouter) in the chat's model picker."
	}
	return msg
}

// runEvent is the subset of `opencode run --format json` lines we care about.
type runEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionID"`
	Part      *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"part"`
	Error *struct {
		Name string `json:"name"`
		Data *struct {
			Message string `json:"message"`
		} `json:"data"`
	} `json:"error"`
}

// parseRunJSON reads newline-delimited JSON events, concatenating text parts.
// Returns (reply, sessionID, apiErrorMessage).
func parseRunJSON(raw []byte) (string, string, string) {
	var sb strings.Builder
	sessionID := ""
	apiErr := ""
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var ev runEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if ev.SessionID != "" {
			sessionID = ev.SessionID
		}
		switch ev.Type {
		case "text":
			if ev.Part != nil && ev.Part.Text != "" {
				sb.WriteString(ev.Part.Text)
			}
		case "error":
			if ev.Error != nil {
				if ev.Error.Data != nil && ev.Error.Data.Message != "" {
					apiErr = ev.Error.Data.Message
				} else if ev.Error.Name != "" {
					apiErr = ev.Error.Name
				}
			}
		}
	}
	return sb.String(), sessionID, apiErr
}
