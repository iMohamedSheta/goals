// Package update checks the public GitHub releases for a newer goals.exe,
// downloads it next to the running one, and swaps it in on restart
// (Windows locks the running exe, so the swap happens after exit).
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Owner      = "iMohamedSheta"
	Repo       = "goals"
	assetName  = "goals.exe"
	minExeSize = 1 << 20 // 1 MiB — anything smaller is not our exe
)

// PendingName is the downloaded update staged next to the running exe.
const PendingName = "goals.pending.exe"

// Version is the running build's version. Local builds report "dev";
// CI bakes the release tag in:
// wails build -ldflags "-X goals/internal/update.Version=v1.2.3".
var Version = "dev"

func apiBase() string {
	if v := strings.TrimSuffix(os.Getenv("GOALS_UPDATE_API"), "/"); v != "" {
		return v
	}
	return "https://api.github.com"
}

// Status is what the frontend shows in Settings → Data.
type Status struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	PageURL   string `json:"pageUrl"`
	Notes     string `json:"notes"`
	Published string `json:"publishedAt"`
}

// Download is the staged update waiting for a restart.
type Download struct {
	Latest string `json:"latest"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
}

type releaseJSON struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

var (
	cacheMu     sync.Mutex
	cachedFor   string
	cachedAt    time.Time
	cachedStat  Status
	cachedAsset string
)

func parseVer(s string) (maj, min, pat int, ok bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.ToLower(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return 0, 0, 0, false
	}
	nums := []int{0, 0, 0}
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 0 {
			return 0, 0, 0, false
		}
		nums[i] = n
	}
	return nums[0], nums[1], nums[2], true
}

// newerThan reports whether latest is a newer release than current.
// An unparseable current (e.g. local "dev" builds) always offers the update;
// an unparseable latest never does.
func newerThan(latest, current string) bool {
	lmaj, lmin, lpat, lok := parseVer(latest)
	if !lok {
		return false
	}
	cmaj, cmin, cpat, cok := parseVer(current)
	if !cok {
		return true
	}
	if lmaj != cmaj {
		return lmaj > cmaj
	}
	if lmin != cmin {
		return lmin > cmin
	}
	return lpat > cpat
}

func fetchRelease(ctx context.Context) (releaseJSON, error) {
	var rel releaseJSON
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase()+"/repos/"+Owner+"/"+Repo+"/releases/latest", nil)
	if err != nil {
		return rel, err
	}
	// GitHub API rejects requests without a User-Agent.
	req.Header.Set("User-Agent", "goals-app")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rel, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return rel, fmt.Errorf("github releases: HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return rel, err
	}
	return rel, nil
}

// checkRelease contacts GitHub (or the 5-minute cache) and returns the
// status plus the exe asset URL.
func checkRelease(current string, force bool) (Status, string, error) {
	cacheMu.Lock()
	fresh := !force && cachedFor == current && time.Since(cachedAt) < 5*time.Minute
	st, asset := cachedStat, cachedAsset
	cacheMu.Unlock()
	if fresh && st.Latest != "" {
		return st, asset, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rel, err := fetchRelease(ctx)
	if err != nil {
		return Status{Current: current}, "", err
	}
	asset = ""
	for _, a := range rel.Assets {
		if strings.EqualFold(a.Name, assetName) {
			asset = a.BrowserDownloadURL
			break
		}
	}
	st = Status{
		Current:   current,
		Latest:    rel.TagName,
		Available: asset != "" && newerThan(rel.TagName, current),
		PageURL:   rel.HTMLURL,
		Notes:     rel.Body,
		Published: rel.PublishedAt,
	}
	cacheMu.Lock()
	cachedFor, cachedAt, cachedStat, cachedAsset = current, time.Now(), st, asset
	cacheMu.Unlock()
	return st, asset, nil
}

// Check reports whether a newer release exists (5-minute cached).
func Check(current string, force bool) (Status, error) {
	st, _, err := checkRelease(current, force)
	return st, err
}

// downloadAllowed ensures we only fetch the exe from the official release
// assets (or from the GOALS_UPDATE_API override host used in tests).
func downloadAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if base := apiBase(); base != "https://api.github.com" {
		if bu, err := url.Parse(base); err == nil && u.Host == bu.Host {
			return (u.Scheme == "https" || u.Scheme == "http") &&
				strings.HasSuffix(strings.ToLower(u.Path), ".exe")
		}
		return false
	}
	return u.Scheme == "https" && u.Host == "github.com" &&
		strings.HasPrefix(u.Path, "/"+Owner+"/"+Repo+"/releases/download/") &&
		strings.HasSuffix(strings.ToLower(u.Path), ".exe")
}

// probeWritable reports whether dir accepts new files (needed to stage the
// update next to the running exe).
func probeWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".goals-wtest")
	if err != nil {
		return fmt.Errorf("app folder is not writable: %v", err)
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return nil
}

// DownloadPending fetches the newest goals.exe next to the running one.
// Nothing is replaced — InstallUpdateAndRestart finishes the job.
func DownloadPending(current string) (Download, error) {
	st, assetURL, err := checkRelease(current, false)
	if err != nil {
		return Download{}, err
	}
	if !st.Available {
		return Download{}, fmt.Errorf("already on the latest version")
	}
	if !downloadAllowed(assetURL) {
		return Download{}, fmt.Errorf("refusing unexpected download URL")
	}
	exe, err := os.Executable()
	if err != nil {
		return Download{}, err
	}
	dir := filepath.Dir(exe)
	if err := probeWritable(dir); err != nil {
		return Download{}, fmt.Errorf("%v - download it manually from %s", err, st.PageURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", assetURL, nil)
	if err != nil {
		return Download{}, err
	}
	req.Header.Set("User-Agent", "goals-app")
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Download{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Download{}, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	dest := filepath.Join(dir, PendingName)
	f, err := os.Create(dest)
	if err != nil {
		return Download{}, err
	}
	n, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(dest)
		return Download{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return Download{}, closeErr
	}
	// Sanity: our exe is ~20 MB and starts with the MZ header.
	if n < minExeSize {
		_ = os.Remove(dest)
		return Download{}, fmt.Errorf("download looks truncated (%d bytes)", n)
	}
	head := make([]byte, 2)
	if rf, err := os.Open(dest); err == nil {
		_, _ = io.ReadFull(rf, head)
		_ = rf.Close()
	}
	if string(head) != "MZ" {
		_ = os.Remove(dest)
		return Download{}, fmt.Errorf("download is not a Windows executable")
	}
	return Download{Latest: st.Latest, Path: dest, Size: n}, nil
}
