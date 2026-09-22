package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OpenRouter hosts every vendor/model slug, so opencode-style ids
// ("google/gemini-…", "anthropic/…", "openrouter/…") pass through — except
// "opencode/*", which only exists behind the local CLI.
func directCapable(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" || strings.HasPrefix(m, "opencode/") {
		return false
	}
	return strings.Contains(m, "/")
}

var (
	modelsMu    sync.Mutex
	modelsCache []string
	modelsAt    time.Time
	badSlugs    = map[string]bool{}
	httpClient  = &http.Client{Timeout: 60 * time.Second}
)

const modelsURL = "https://openrouter.ai/api/v1/models"

// listModels fetches OpenRouter model slugs (free, no tokens), cached 1h.
func listModels(ctx context.Context, key string) ([]string, error) {
	modelsMu.Lock()
	if time.Since(modelsAt) < time.Hour && len(modelsCache) > 0 {
		out := append([]string(nil), modelsCache...)
		modelsMu.Unlock()
		return out, nil
	}
	modelsMu.Unlock()

	req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("X-Title", "Goals")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("openrouter models: HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	var ids []string
	for _, m := range payload.Data {
		if id := strings.TrimSpace(m.ID); id != "" {
			ids = append(ids, id)
		}
	}
	modelsMu.Lock()
	modelsCache = append([]string(nil), ids...)
	modelsAt = time.Now()
	modelsMu.Unlock()
	return ids, nil
}

// alphaTokens splits "google/gemini-3.5-flash-lite" into [google gemini flash lite].
func alphaTokens(s string) []string {
	var toks []string
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r < 'a' || r > 'z'
	}) {
		if len(f) > 1 {
			toks = append(toks, f)
		}
	}
	return toks
}

// pickSlug resolves a wanted id to a real OpenRouter slug: exact match wins,
// else best token-overlap within the same vendor family. "" = unknown.
func pickSlug(want string, ids []string) string {
	w := strings.ToLower(strings.TrimSpace(want))
	w = strings.TrimPrefix(w, "openrouter/")
	for _, id := range ids {
		if strings.EqualFold(id, w) || strings.EqualFold(id, want) {
			return id
		}
	}
	vendor := w
	if i := strings.Index(vendor, "/"); i >= 0 {
		vendor = vendor[:i]
	}
	wantToks := map[string]bool{}
	for _, t := range alphaTokens(w) {
		wantToks[t] = true
	}
	score := func(id string) int {
		lid := strings.ToLower(id)
		if vendor != "" && !strings.HasPrefix(lid, vendor+"/") && !strings.Contains(lid, vendor) {
			return 0
		}
		s := 0
		for _, t := range alphaTokens(id) {
			if wantToks[t] {
				s += 2
			}
		}
		s -= len(id) / 64 // tie-break toward shorter slugs
		return s
	}
	// Two passes: "~"-prefixed catalog aliases only as a last resort.
	best, bestScore := "", 0
	for _, id := range ids {
		if strings.HasPrefix(id, "~") {
			continue
		}
		if s := score(id); s > bestScore {
			best, bestScore = id, s
		}
	}
	if best == "" {
		for _, id := range ids {
			if s := score(id); s > bestScore {
				best, bestScore = id, s
			}
		}
	}
	return best
}

// markBad remembers slugs the API rejected so later calls skip straight to
// fallback instead of burning another request.
func markBad(slug string) {
	if slug == "" {
		return
	}
	modelsMu.Lock()
	badSlugs[strings.ToLower(slug)] = true
	modelsMu.Unlock()
}

func isBad(slug string) bool {
	modelsMu.Lock()
	defer modelsMu.Unlock()
	return badSlugs[strings.ToLower(slug)]
}

// ResolveModel returns the OpenRouter slug to use for want, or "" when
// direct mode can't serve it (caller falls back to the CLI).
func ResolveModel(ctx context.Context, key, want string) string {
	if !directCapable(want) || key == "" {
		return ""
	}
	ids, err := listModels(ctx, key)
	if err != nil || len(ids) == 0 {
		return ""
	}
	slug := pickSlug(want, ids)
	if slug == "" || isBad(slug) {
		return ""
	}
	return slug
}
