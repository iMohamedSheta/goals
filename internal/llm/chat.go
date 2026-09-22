package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"goals/internal/mcp"
	"goals/internal/store"
)

const chatURL = "https://openrouter.ai/api/v1/chat/completions"

// maxIters caps tool-call rounds per user command; maxToolChars caps a
// single tool result so one big dump can't blow the context.
const maxIters = 8
const maxToolChars = 12000
const maxTokens = 4096

// errUnknownModel marks a rejected model slug (caller falls back to CLI).
var errUnknownModel = errors.New("llm:unknown-model")

// IsUnknownModel reports whether err means OpenRouter has no such model.
func IsUnknownModel(err error) bool {
	return errors.Is(err, errUnknownModel)
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    any            `json:"content,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name,omitempty"`
}

type chatToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  any    `json:"parameters"`
	} `json:"function"`
}

func openAITools() []chatTool {
	out := []chatTool{}
	for _, d := range mcp.ToolDefs() {
		t := chatTool{Type: "function"}
		t.Function.Name = d.Name
		t.Function.Description = d.Description
		t.Function.Parameters = d.InputSchema
		out = append(out, t)
	}
	return out
}

func systemPrompt() string {
	return "You are the Goals desktop assistant. Manage the user's tasks, contexts, horizons and timers using the provided tools. " +
		"Reply in the user's language (Arabic if they write Arabic, else English). " +
		"Be concise. Never call delete_task, delete_horizon or set_setting unless the user explicitly asked for that deletion/change in the same message. " +
		"Today is " + time.Now().Format("2006-01-02") + "."
}

// toolText flattens an MCP content-block payload into plain text for the model.
func toolText(v any) string {
	if m, ok := v.(map[string]any); ok {
		if content, ok := m["content"].([]map[string]any); ok {
			var sb strings.Builder
			for _, b := range content {
				if b["type"] == "text" {
					if s, ok := b["text"].(string); ok {
						sb.WriteString(s)
					}
				}
			}
			if sb.Len() > 0 {
				return truncate(sb.String())
			}
		}
		// errResult shape: surface the error text plainly.
		if content, ok := m["content"].([]any); ok {
			var sb strings.Builder
			for _, b := range content {
				if bm, ok := b.(map[string]any); ok {
					if s, ok := bm["text"].(string); ok {
						sb.WriteString(s)
					}
				}
			}
			if sb.Len() > 0 {
				return truncate(sb.String())
			}
		}
	}
	raw, _ := json.Marshal(v)
	return truncate(string(raw))
}

func truncate(s string) string {
	if len(s) > maxToolChars {
		return s[:maxToolChars] + "\n…(truncated)"
	}
	return s
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role      string         `json:"role"`
			Content   string         `json:"content"`
			ToolCalls []chatToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func postChat(ctx context.Context, key, slug string, msgs []chatMessage, tools []chatTool) (*chatResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"model":       slug,
		"messages":    msgs,
		"tools":       tools,
		"tool_choice": "auto",
		"max_tokens":  maxTokens,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", "Goals")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("openrouter: bad response (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		msg := "openrouter: unauthorized — check the stored key (`opencode auth login`)"
		if out.Error != nil && out.Error.Message != "" {
			msg += ": " + out.Error.Message
		}
		return nil, errors.New(msg)
	}
	if out.Error != nil && out.Error.Message != "" {
		if resp.StatusCode == 404 || isModelRejection(out.Error.Message) {
			markBad(slug)
			return nil, fmt.Errorf("%w: %s", errUnknownModel, out.Error.Message)
		}
		return nil, errors.New(out.Error.Message)
	}
	if resp.StatusCode >= 400 {
		snippet := strings.TrimSpace(string(raw))
		if len(snippet) > 300 {
			snippet = snippet[:300] + "…"
		}
		if resp.StatusCode == 404 || isModelRejection(snippet) {
			markBad(slug)
			return nil, fmt.Errorf("%w: HTTP %d %s", errUnknownModel, resp.StatusCode, snippet)
		}
		return nil, fmt.Errorf("openrouter: HTTP %d %s", resp.StatusCode, snippet)
	}
	return &out, nil
}

func isModelRejection(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "not found") ||
		strings.Contains(l, "no endpoints") ||
		strings.Contains(l, "not a valid model") ||
		strings.Contains(l, "unknown model")
}

// Try answers want (opencode-style model id) directly over HTTPS.
// ok=false means direct mode can't serve it — caller falls back to the CLI.
// ok=true carries either the reply or the direct failure.
func Try(ctx context.Context, want, prompt string) (out string, ok bool, err error) {
	key := OpenRouterKey()
	if key == "" || !directCapable(want) {
		return "", false, nil
	}
	slug := ResolveModel(ctx, key, want)
	if slug == "" {
		return "", false, nil
	}
	out, err = Chat(ctx, key, slug, prompt)
	return out, true, err
}

// Chat runs one prompt directly over HTTPS with local tool execution.
// No CLI process is spawned. ctx honors the chat stop button.
func Chat(ctx context.Context, key, slug, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("prompt is empty")
	}
	if key == "" {
		return "", fmt.Errorf("direct mode not configured")
	}
	if slug == "" {
		return "", fmt.Errorf("%w: empty slug", errUnknownModel)
	}
	s, err := store.Open(store.ResolveDBPath(""))
	if err != nil {
		return "", err
	}
	defer s.Close()

	tools := openAITools()
	msgs := []chatMessage{
		{Role: "system", Content: systemPrompt()},
		{Role: "user", Content: prompt},
	}
	for i := 0; i < maxIters; i++ {
		resp, err := postChat(ctx, key, slug, msgs, tools)
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("openrouter: empty choices")
		}
		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			if strings.TrimSpace(msg.Content) == "" {
				return "", fmt.Errorf("openrouter returned an empty reply")
			}
			return strings.TrimSpace(msg.Content), nil
		}
		msgs = append(msgs, chatMessage{Role: "assistant", Content: msg.Content, ToolCalls: msg.ToolCalls})
		for _, tc := range msg.ToolCalls {
			if ctx.Err() != nil {
				return "", fmt.Errorf("cancelled")
			}
			var args map[string]any
			if strings.TrimSpace(tc.Function.Arguments) != "" {
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			}
			if args == nil {
				args = map[string]any{}
			}
			res, unknown := mcp.Call(s, tc.Function.Name, args)
			var text string
			if unknown {
				text = "Error: unknown tool: " + tc.Function.Name
			} else {
				text = toolText(res)
			}
			msgs = append(msgs, chatMessage{Role: "tool", ToolCallID: tc.ID, Content: text})
		}
	}
	return "", fmt.Errorf("openrouter: tool loop exceeded %d rounds", maxIters)
}
