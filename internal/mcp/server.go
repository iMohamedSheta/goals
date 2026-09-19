// Package mcp exposes the Goals database as an MCP server over stdio.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goals/internal/store"
)

// Minimal MCP (Model Context Protocol) stdio server, JSON-RPC 2.0 NDJSON.
// Compatible with opencode `type: local, command: ["goals.exe","mcp"]`.
// Protocol version: 2024-11-05

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcErr `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type toolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

func strPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v := s
	return &v
}

func tools() []toolDef {
	obj := func(props map[string]any, required []string) map[string]any {
		return map[string]any{"type": "object", "properties": props, "required": required}
	}
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	boolean := func(desc string) map[string]any { return map[string]any{"type": "boolean", "description": desc} }
	return []toolDef{
		{Name: "list_tasks", Description: "List goal tasks with optional filters (horizon: short|medium|long|all, status, contextId, focusOnly, search).", InputSchema: obj(map[string]any{
			"horizon": str("short|medium|long|all (default all)"), "status": str("todo|in_progress|done|blocked|all"),
			"contextId": str("filter by context id"), "focusOnly": boolean("only focused tasks"), "search": str("search title/description"),
		}, []string{})},
		{Name: "get_task", Description: "Get a single task by id.", InputSchema: obj(map[string]any{"id": str("task id")}, []string{"id"})},
		{Name: "create_task", Description: "Create a goal task. horizon short(~week)/medium(~month)/long(~3mo). status todo|in_progress|done|blocked.", InputSchema: obj(map[string]any{
			"title": str("task title (required)"), "description": str("details"), "horizon": str("short|medium|long"),
			"status": str("todo|in_progress|done|blocked"), "contextId": str("context id (Work/Life/...)"),
			"priority": str("low|medium|high|urgent"), "startDate": str("YYYY-MM-DD"), "dueDate": str("YYYY-MM-DD (defaults from horizon)"), "focus": boolean("pin to focus list"),
			"maxSeconds": map[string]any{"type": "integer", "description": "max expected time in seconds (overtime alert)"},
		}, []string{"title"})},
		{Name: "update_task", Description: "Update all editable fields of a task.", InputSchema: obj(map[string]any{
			"id": str("task id"), "title": str("title"), "description": str("details"), "horizon": str("short|medium|long"),
			"status": str("todo|in_progress|done|blocked"), "contextId": str("context id"), "priority": str("low|medium|high|urgent"),
			"startDate": str("YYYY-MM-DD"), "dueDate": str("YYYY-MM-DD"), "focus": boolean("focus flag"),
			"maxSeconds": map[string]any{"type": "integer", "description": "max expected time in seconds (overtime alert)"},
		}, []string{"id", "title"})},
		{Name: "move_task", Description: "Move a task to another kanban column.", InputSchema: obj(map[string]any{"id": str("task id"), "status": str("todo|in_progress|done|blocked")}, []string{"id", "status"})},
		{Name: "toggle_focus", Description: "Toggle a task's focus pin.", InputSchema: obj(map[string]any{"id": str("task id")}, []string{"id"})},
		{Name: "delete_task", Description: "Delete a task.", InputSchema: obj(map[string]any{"id": str("task id")}, []string{"id"})},
		{Name: "list_contexts", Description: "List contexts (Work, Life, ...).", InputSchema: obj(map[string]any{}, []string{})},
		{Name: "create_context", Description: "Create a context (e.g. Work, Life).", InputSchema: obj(map[string]any{"name": str("name"), "color": str("hex color")}, []string{"name"})},
		{Name: "list_horizons", Description: "List planning horizons (short/medium/long) with their timeline defaults.", InputSchema: obj(map[string]any{}, []string{})},
		{Name: "update_horizon", Description: "Modify a horizon timeline (e.g. change short from 7 to 14 days).", InputSchema: obj(map[string]any{
			"key": str("short|medium|long"), "label": str("label"), "labelAr": str("arabic label"), "defaultDays": map[string]any{"type": "integer", "description": "default duration in days"}, "description": str("description"), "descriptionAr": str("arabic description"),
		}, []string{"key"})},
		{Name: "create_horizon", Description: "Add a new planning tab (custom timeline).", InputSchema: obj(map[string]any{
			"label": str("label"), "labelAr": str("arabic label"), "defaultDays": map[string]any{"type": "integer", "description": "default duration in days"}, "description": str("description"), "descriptionAr": str("arabic description"),
		}, []string{})},
		{Name: "delete_horizon", Description: "Delete a planning tab (refused when it still has tasks).", InputSchema: obj(map[string]any{"key": str("horizon key")}, []string{"key"})},
		{Name: "focus_list", Description: "List only focused tasks (your current focus).", InputSchema: obj(map[string]any{"horizon": str("short|medium|long|all")}, []string{})},
		{Name: "stats", Description: "Counts by horizon/status, total, focused, done.", InputSchema: obj(map[string]any{}, []string{})},
		{Name: "start_timer", Description: "Start the time tracker on a task (auto-pauses any other running timer). Keeps running even if the app window is closed.", InputSchema: obj(map[string]any{"id": str("task id")}, []string{"id"})},
		{Name: "stop_timer", Description: "Pause the time tracker on a task and save the segment.", InputSchema: obj(map[string]any{"id": str("task id")}, []string{"id"})},
		{Name: "finish_task", Description: "Stop the timer (saving tracked time) and mark the task done, recording completion time.", InputSchema: obj(map[string]any{"id": str("task id")}, []string{"id"})},
		{Name: "active_timer", Description: "Show the currently running timer, if any.", InputSchema: obj(map[string]any{}, []string{})},
		{Name: "task_time", Description: "Tracked time for a task: total seconds + history entries.", InputSchema: obj(map[string]any{"id": str("task id")}, []string{"id"})},
		{Name: "get_settings", Description: "Read app settings (appearance: theme, accent, font, radius, density, ...).", InputSchema: obj(map[string]any{}, []string{})},
		{Name: "set_setting", Description: "Change an app setting, e.g. appearance.theme=light, appearance.accent=emerald.", InputSchema: obj(map[string]any{"key": str("setting key"), "value": str("setting value")}, []string{"key", "value"})},
	}
}

func textResult(v any) map[string]any {
	b, _ := json.MarshalIndent(v, "", "  ")
	return map[string]any{"content": []map[string]any{{"type": "text", "text": string(b)}}}
}

func errResult(err error) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": "Error: " + err.Error()}}, "isError": true}
}

// Run starts the stdio loop. dbPath may be "" (auto-resolve).
func Run(dbPath string) int {
	s, err := store.Open(store.ResolveDBPath(dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "goals mcp: open db: %v\n", err)
		return 1
	}
	defer s.Close()

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1024*1024), 1024*1024*10)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	write := func(id any, result any, rerr *rpcErr) {
		resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result, Error: rerr}
		b, _ := json.Marshal(resp)
		b = append(b, '\n')
		_, _ = out.Write(b)
		_ = out.Flush()
	}

	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		if req.JSONRPC == "" {
			req.JSONRPC = "2.0"
		}
		// notifications (no id) -> no response
		isNotification := req.ID == nil

		switch req.Method {
		case "initialize":
			write(req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
				"serverInfo":      map[string]any{"name": "goals", "version": "1.0.0"},
			}, nil)
		case "notifications/initialized", "notifications/cancelled":
			// no-op
		case "ping":
			if !isNotification {
				write(req.ID, map[string]any{}, nil)
			}
		case "tools/list":
			names := tools()
			write(req.ID, map[string]any{"tools": names}, nil)
		case "tools/call":
			var p struct {
				Name      string         `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			_ = json.Unmarshal(req.Params, &p)
			if p.Name == "" {
				// some clients nest as params.arguments directly
				var alt map[string]any
				_ = json.Unmarshal(req.Params, &alt)
				if n, ok := alt["name"].(string); ok {
					p.Name = n
					p.Arguments, _ = json.Marshal(alt["arguments"])
				}
			}
			args := map[string]any{}
			if len(p.Arguments) > 0 {
				_ = json.Unmarshal(p.Arguments, &args)
			}
			getStr := func(k string) string {
				if v, ok := args[k].(string); ok {
					return v
				}
				return ""
			}
			getBool := func(k string) bool {
				if v, ok := args[k].(bool); ok {
					return v
				}
				return false
			}
			getInt := func(k string) int64 {
				if v, ok := args[k].(float64); ok {
					return int64(v)
				}
				return 0
			}
			var result, rerr any
			switch p.Name {
			case "list_tasks":
				f := store.TaskFilter{Horizon: getStr("horizon"), Status: getStr("status"), ContextID: getStr("contextId"), Search: getStr("search"), FocusOnly: getBool("focusOnly")}
				if f.Horizon == "" {
					f.Horizon = "all"
				}
				tasks, err := s.ListTasks(f)
				if err != nil {
					result = errResult(err)
				} else {
					if tasks == nil {
						tasks = []store.TaskDetail{}
					}
					result = textResult(tasks)
				}
			case "focus_list":
				h := getStr("horizon")
				if h == "" {
					h = "all"
				}
				tasks, err := s.ListTasks(store.TaskFilter{Horizon: h, FocusOnly: true})
				if err != nil {
					result = errResult(err)
				} else {
					if tasks == nil {
						tasks = []store.TaskDetail{}
					}
					result = textResult(tasks)
				}
			case "get_task":
				t, err := s.GetTask(getStr("id"))
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(t)
				}
			case "create_task":
				in := store.TaskInput{Title: getStr("title"), Description: getStr("description"), Horizon: getStr("horizon"), Status: getStr("status"), Priority: getStr("priority"), ContextID: strPtr(getStr("contextId")), StartDate: strPtr(getStr("startDate")), DueDate: strPtr(getStr("dueDate")), Focus: getBool("focus"), MaxSeconds: getInt("maxSeconds")}
				t, err := s.CreateTask(in)
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(t)
				}
			case "update_task":
				in := store.TaskInput{Title: getStr("title"), Description: getStr("description"), Horizon: getStr("horizon"), Status: getStr("status"), Priority: getStr("priority"), ContextID: strPtr(getStr("contextId")), StartDate: strPtr(getStr("startDate")), DueDate: strPtr(getStr("dueDate")), Focus: getBool("focus"), MaxSeconds: getInt("maxSeconds")}
				// preserve focus if not provided? bool defaults false — acceptable, UI passes explicit.
				t, err := s.UpdateTask(getStr("id"), in)
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(t)
				}
			case "move_task":
				t, err := s.MoveTask(getStr("id"), getStr("status"))
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(t)
				}
			case "toggle_focus":
				t, err := s.ToggleFocus(getStr("id"))
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(t)
				}
			case "delete_task":
				if err := s.DeleteTask(getStr("id")); err != nil {
					result = errResult(err)
				} else {
					result = textResult(map[string]any{"deleted": getStr("id")})
				}
			case "list_contexts":
				cs, err := s.ListContexts()
				if err != nil {
					result = errResult(err)
				} else {
					if cs == nil {
						cs = []store.Context{}
					}
					result = textResult(cs)
				}
			case "create_context":
				c, err := s.CreateContext(getStr("name"), getStr("color"))
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(c)
				}
			case "list_horizons":
				hs, err := s.ListHorizons()
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(hs)
				}
			case "update_horizon":
				label := getStr("label")
				labelAr := getStr("labelAr")
				desc := getStr("description")
				descAr := getStr("descriptionAr")
				days := 0
				if v, ok := args["defaultDays"].(float64); ok {
					days = int(v)
				}
				key := getStr("key")
				if label == "" || days == 0 {
					// fill from existing so partial updates work
					if h, err := s.GetHorizon(key); err == nil {
						if label == "" {
							label = h.Label
						}
						if labelAr == "" {
							labelAr = h.LabelAr
						}
						if days == 0 {
							days = h.DefaultDays
						}
						if desc == "" {
							desc = h.Description
						}
						if descAr == "" {
							descAr = h.DescriptionAr
						}
					}
				}
				h, err := s.UpdateHorizon(key, label, labelAr, days, desc, descAr)
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(h)
				}
			case "create_horizon":
				days := 30
				if v, ok := args["defaultDays"].(float64); ok && int(v) > 0 {
					days = int(v)
				}
				h, err := s.CreateHorizon(getStr("label"), getStr("labelAr"), days, getStr("description"), getStr("descriptionAr"))
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(h)
				}
			case "delete_horizon":
				if err := s.DeleteHorizon(getStr("key")); err != nil {
					result = errResult(err)
				} else {
					result = textResult(map[string]any{"deleted": getStr("key")})
				}
			case "stats":
				st, err := s.GetStats()
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(st)
				}
			case "start_timer":
				t, err := s.StartTimer(getStr("id"))
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(t)
				}
			case "stop_timer":
				t, err := s.StopTimer(getStr("id"))
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(t)
				}
			case "finish_task":
				t, err := s.FinishTask(getStr("id"))
				if err != nil {
					result = errResult(err)
				} else {
					result = textResult(t)
				}
			case "active_timer":
				t, err := s.GetActiveTimer()
				if err != nil {
					result = textResult(map[string]any{"running": false})
				} else {
					elapsed, _, _ := s.Elapsed(t.ID)
					result = textResult(map[string]any{"running": true, "elapsedSeconds": elapsed, "task": t})
				}
			case "task_time":
				id := getStr("id")
				t, err := s.GetTask(id)
				if err != nil {
					result = errResult(err)
				} else {
					elapsed, running, _ := s.Elapsed(id)
					entries, _ := s.ListTimeEntries(id)
					if entries == nil {
						entries = []store.TimeEntry{}
					}
					result = textResult(map[string]any{"task": t.Title, "running": running, "totalSeconds": elapsed, "entries": entries})
				}
			case "get_settings":				settings, err := s.GetSettings()
				if err != nil {
					result = errResult(err)
				} else {
					if settings == nil {
						settings = map[string]string{}
					}
					result = textResult(settings)
				}
			case "set_setting":
				if err := s.SetSetting(getStr("key"), getStr("value")); err != nil {
					result = errResult(err)
				} else {
					result = textResult(map[string]any{"key": getStr("key"), "value": getStr("value")})
				}
			default:
				rerr = &rpcErr{Code: -32601, Message: "unknown tool: " + p.Name}
			}
			if rerr != nil {
				write(req.ID, nil, rerr.(*rpcErr))
			} else {
				write(req.ID, result, nil)
			}
		default:
			if !isNotification {
				write(req.ID, nil, &rpcErr{Code: -32601, Message: "method not found: " + req.Method})
			}
		}
	}
	return 0
}
