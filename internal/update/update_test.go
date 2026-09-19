package update

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewerThan(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.2.3", "v1.2.3", false},
		{"v1.2.4", "v1.2.3", true},
		{"v1.3.0", "v1.2.9", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.2.3", "v1.2.4", false},
		{"v1.2", "v1.2.0", false},
		{"1.2.3", "v1.2.2", true}, // missing v tolerated
		{"v1.2.3", "dev", true},   // local builds always offered
		{"v1.2.3", "", true},
		{"", "v1.0.0", false}, // unknown latest never offers
		{"oops", "v1.0.0", false},
	}
	for _, c := range cases {
		if got := newerThan(c.latest, c.current); got != c.want {
			t.Errorf("newerThan(%q,%q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

// stubReleases serves the GitHub "latest release" JSON plus the asset bytes.
func stubReleases(t *testing.T, tag string, assetBytes []byte) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+Owner+"/"+Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tag_name":%q,"html_url":%q,"body":"notes here","published_at":"2026-01-01T00:00:00Z",`+
			`"assets":[{"name":"goals.exe","browser_download_url":%q,"size":%d}]}`,
			tag, srv.URL+"/pages/v", srv.URL+"/dl/goals.exe", len(assetBytes))
	})
	mux.HandleFunc("/dl/goals.exe", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(assetBytes)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func bigMZ(size int) []byte {
	b := make([]byte, size)
	b[0], b[1] = 'M', 'Z'
	for i := 2; i < size; i++ {
		b[i] = byte(i)
	}
	return b
}

func TestCheckAgainstStub(t *testing.T) {
	srv := stubReleases(t, "v0.2.0", bigMZ(2<<20))
	t.Setenv("GOALS_UPDATE_API", srv.URL)

	st, err := Check("v0.1.0", true)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !st.Available || st.Latest != "v0.2.0" || st.Notes != "notes here" {
		t.Fatalf("unexpected status: %+v", st)
	}
	st, err = Check("v9.9.9", true)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if st.Available {
		t.Fatalf("should not offer an update when current is newer: %+v", st)
	}
}

func TestDownloadPending(t *testing.T) {
	srv := stubReleases(t, "v0.2.0", bigMZ(2<<20))
	t.Setenv("GOALS_UPDATE_API", srv.URL)

	// os.Executable points at the test binary — download lands next to it.
	// Redirect via GOALS_UPDATE_API host allowance; clean the pending file after.
	// Distinct current from other tests: the package caches per version and
	// each test gets its own stub server URL.
	dl, err := DownloadPending("v0.0.5")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer os.Remove(dl.Path)
	if filepath.Base(dl.Path) != PendingName || dl.Latest != "v0.2.0" {
		t.Fatalf("unexpected download: %+v", dl)
	}
	raw, err := os.ReadFile(dl.Path)
	if err != nil || len(raw) != 2<<20 || string(raw[:2]) != "MZ" {
		t.Fatalf("bad staged file: %v %d", err, len(raw))
	}
}

func TestDownloadRejects(t *testing.T) {
	for _, u := range []string{
		"https://evil.com/goals.exe",
		"https://github.com/other/repo/releases/download/v1/goals.exe",
		"https://github.com/" + Owner + "/" + Repo + "/releases/download/v1/notes.txt",
		"http://github.com/" + Owner + "/" + Repo + "/releases/download/v1/goals.exe",
	} {
		if downloadAllowed(u) {
			t.Errorf("should refuse %s", u)
		}
	}
	ok := "https://github.com/" + Owner + "/" + Repo + "/releases/download/v0.2.0/goals.exe"
	if !downloadAllowed(ok) {
		t.Errorf("should allow %s", ok)
	}
	if !strings.HasPrefix(ok, "https://github.com/") {
		t.Fatal("test bug")
	}
}
