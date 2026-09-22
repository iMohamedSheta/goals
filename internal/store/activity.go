package store

import (
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---------- App-usage tracking ----------

// ActivitySegment is one continuous foreground run of an app/window.
// App is the friendly app name (e.g. "Google Chrome"), Title the raw window
// title, Detail the extracted page/video/document name, Domain the detected
// site (e.g. "youtube.com", empty for desktop apps), and Category one of:
// "work" | "distraction" | "other" | "idle".
type ActivitySegment struct {
	ID        string  `json:"id"`
	App       string  `json:"app"`
	Title     string  `json:"title"`
	Detail    string  `json:"detail"`
	Domain    string  `json:"domain"`
	Category  string  `json:"category"`
	StartedAt string  `json:"startedAt"`
	EndedAt   *string `json:"endedAt"`
	Seconds   int64   `json:"seconds"`
}

// ActivityAppRow aggregates one app (or one domain) inside a summary range.
type ActivityAppRow struct {
	Name     string `json:"name"`
	Seconds  int64  `json:"seconds"`
	Sessions int    `json:"sessions"`
	Category string `json:"category"`
}

// ActivitySummary is the insights payload for a day / week / month / overall.
type ActivitySummary struct {
	Range        string           `json:"range"`
	RefDay       string           `json:"refDay"`
	Total        int64            `json:"totalSeconds"`
	Work         int64            `json:"workSeconds"`
	Distraction  int64            `json:"distractionSeconds"`
	Other        int64            `json:"otherSeconds"`
	Idle         int64            `json:"idleSeconds"`
	Sessions     int              `json:"sessions"`
	ByApp        []ActivityAppRow `json:"byApp"`
	ByDomain     []ActivityAppRow `json:"byDomain"`
	TopDistract  []ActivityAppRow `json:"topDistractions"`
	WastedPct    float64          `json:"wastedPct"`
	LiveSeconds  int64            `json:"liveSeconds"`
}

func validActivityCategory(c string) string {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case "work":
		return "work"
	case "distraction":
		return "distraction"
	case "idle":
		return "idle"
	default:
		return "other"
	}
}

// RecordActivitySegment persists one finished foreground run. Durations under
// minSeconds are dropped (polling noise); longer runs are stored whole.
func (s *Store) RecordActivitySegment(app, title, detail, domain, category, startedAt, endedAt string, seconds int64) (ActivitySegment, error) {
	seg := ActivitySegment{}
	category = validActivityCategory(category)
	if seconds < 3 {
		return seg, nil
	}
	if strings.TrimSpace(app) == "" {
		app = "Unknown"
	}
	id := uuid.NewString()
	if _, err := s.db.Exec(`INSERT INTO activity_segments(id,app,title,detail,domain,category,started_at,ended_at,seconds,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, id, app, title, detail, domain, category, startedAt, endedAt, seconds, nowStr()); err != nil {
		return seg, err
	}
	seg = ActivitySegment{ID: id, App: app, Title: title, Detail: detail, Domain: domain, Category: category, StartedAt: startedAt, Seconds: seconds}
	if strings.TrimSpace(endedAt) != "" {
		v := endedAt
		seg.EndedAt = &v
	}
	return seg, nil
}

// activityRangeCond builds the WHERE fragment for a summary range.
func activityRangeCond(rng, refDay string) (string, []any) {
	refDay = normDay(refDay)
	switch strings.ToLower(strings.TrimSpace(rng)) {
	case "week":
		return `substr(started_at,1,10)>=?`, []any{weekStart(refDay)}
	case "month":
		m := refDay[:7]
		return `substr(started_at,1,7)=?`, []any{m}
	case "overall", "all":
		return `1=1`, []any{}
	default: // day
		return `substr(started_at,1,10)=?`, []any{refDay}
	}
}

// GetActivitySummary aggregates usage for day | week | month | overall.
// liveExtra adds the currently-running (unflushed) foreground run so the
// "today" card ticks live without waiting for the segment to close.
func (s *Store) GetActivitySummary(rng, refDay string, liveExtra ActivitySegment) (ActivitySummary, error) {
	rng = strings.ToLower(strings.TrimSpace(rng))
	if rng != "week" && rng != "month" && rng != "overall" && rng != "all" {
		rng = "day"
	}
	if rng == "all" {
		rng = "overall"
	}
	refDay = normDay(refDay)
	cond, args := activityRangeCond(rng, refDay)
	out := ActivitySummary{Range: rng, RefDay: refDay, ByApp: []ActivityAppRow{}, ByDomain: []ActivityAppRow{}, TopDistract: []ActivityAppRow{}}

	row := s.db.QueryRow(`SELECT COALESCE(SUM(seconds),0),
		COALESCE(SUM(CASE WHEN category='work' THEN seconds ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN category='distraction' THEN seconds ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN category='other' THEN seconds ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN category='idle' THEN seconds ELSE 0 END),0),
		COUNT(*) FROM activity_segments WHERE `+cond, args...)
	if err := row.Scan(&out.Total, &out.Work, &out.Distraction, &out.Other, &out.Idle, &out.Sessions); err != nil {
		return out, err
	}
	// Fold the live run into today's totals (day range only, non-idle).
	if rng == "day" && liveExtra.Seconds > 0 && (liveExtra.StartedAt[:10] == refDay || strings.HasPrefix(liveExtra.StartedAt, refDay)) {
		out.Total += liveExtra.Seconds
		out.Sessions++
		out.LiveSeconds = liveExtra.Seconds
		switch liveExtra.Category {
		case "work":
			out.Work += liveExtra.Seconds
		case "distraction":
			out.Distraction += liveExtra.Seconds
		case "idle":
			out.Idle += liveExtra.Seconds
		default:
			out.Other += liveExtra.Seconds
		}
	}
	if out.Total > 0 {
		out.WastedPct = float64(out.Distraction) * 100 / float64(out.Total)
	}

	// Per-app breakdown (top 20).
	rows, err := s.db.Query(`SELECT app, COALESCE(SUM(seconds),0), COUNT(*),
		(SELECT category FROM activity_segments s2 WHERE s2.app=activity_segments.app GROUP BY category ORDER BY SUM(seconds) DESC LIMIT 1)
		FROM activity_segments WHERE `+cond+` GROUP BY app ORDER BY SUM(seconds) DESC LIMIT 20`, args...)
	if err == nil {
		for rows.Next() {
			var r ActivityAppRow
			var cat sql.NullString
			if err := rows.Scan(&r.Name, &r.Seconds, &r.Sessions, &cat); err == nil {
				if cat.Valid {
					r.Category = cat.String
				}
				out.ByApp = append(out.ByApp, r)
			}
		}
		rows.Close()
	}
	// Per-domain breakdown (top 15, skips empty domain).
	rows2, err := s.db.Query(`SELECT domain, COALESCE(SUM(seconds),0), COUNT(*),
		(SELECT category FROM activity_segments s2 WHERE s2.domain=activity_segments.domain GROUP BY category ORDER BY SUM(seconds) DESC LIMIT 1)
		FROM activity_segments WHERE `+cond+` AND domain<>'' GROUP BY domain ORDER BY SUM(seconds) DESC LIMIT 15`, args...)
	if err == nil {
		for rows2.Next() {
			var r ActivityAppRow
			var cat sql.NullString
			if err := rows2.Scan(&r.Name, &r.Seconds, &r.Sessions, &cat); err == nil {
				if cat.Valid {
					r.Category = cat.String
				}
				out.ByDomain = append(out.ByDomain, r)
			}
		}
		rows2.Close()
	}
	// Top distractions by detail (e.g. which YouTube videos ate the day).
	rows3, err := s.db.Query(`SELECT COALESCE(NULLIF(detail,''),title,app), COALESCE(SUM(seconds),0), COUNT(*)
		FROM activity_segments WHERE `+cond+` AND category='distraction' GROUP BY COALESCE(NULLIF(detail,''),title,app) ORDER BY SUM(seconds) DESC LIMIT 10`, args...)
	if err == nil {
		for rows3.Next() {
			var r ActivityAppRow
			if err := rows3.Scan(&r.Name, &r.Seconds, &r.Sessions); err == nil {
				r.Category = "distraction"
				out.TopDistract = append(out.TopDistract, r)
			}
		}
		rows3.Close()
	}
	return out, nil
}

// ListActivitySegments returns the most recent segments for a range (for the
// "everything I did" timeline). appFilter optionally narrows to one app
// (empty/"all" = everything). limit caps rows (default 100, max 500).
func (s *Store) ListActivitySegments(rng, refDay, appFilter string, limit int) ([]ActivitySegment, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rng = strings.ToLower(strings.TrimSpace(rng))
	if rng != "week" && rng != "month" && rng != "overall" && rng != "all" && rng != "day" {
		rng = "day"
	}
	if rng == "all" {
		rng = "overall"
	}
	cond, args := activityRangeCond(rng, normDay(refDay))
	if a := strings.TrimSpace(appFilter); a != "" && !strings.EqualFold(a, "all") {
		cond += ` AND app=?`
		args = append(args, a)
	}
	args = append(args, limit)
	rows, err := s.db.Query(`SELECT id,app,title,detail,domain,category,started_at,ended_at,seconds
		FROM activity_segments WHERE `+cond+` ORDER BY started_at DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ActivitySegment{}
	for rows.Next() {
		var g ActivitySegment
		var ended sql.NullString
		if err := rows.Scan(&g.ID, &g.App, &g.Title, &g.Detail, &g.Domain, &g.Category, &g.StartedAt, &ended, &g.Seconds); err != nil {
			return nil, err
		}
		if ended.Valid {
			v := ended.String
			g.EndedAt = &v
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// ClearActivity deletes stored segments (optionally before a day, empty = all).
func (s *Store) ClearActivity(beforeDay string) (int64, error) {
	if d := strings.TrimSpace(beforeDay); d != "" {
		res, err := s.db.Exec(`DELETE FROM activity_segments WHERE substr(started_at,1,10)<?`, normDay(d))
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return n, nil
	}
	res, err := s.db.Exec(`DELETE FROM activity_segments`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ActivityStats describes stored usage volume (for the retention UI).
type ActivityStats struct {
	Segments int64  `json:"segments"`
	Oldest   string `json:"oldest"` // YYYY-MM-DD of the oldest segment, "" when empty
	Newest   string `json:"newest"`
	DBBytes  int64  `json:"dbBytes"` // main db file size (WAL excluded)
}

// ActivityStats counts segments, finds the stored span and the db file size.
func (s *Store) ActivityStats() (ActivityStats, error) {
	out := ActivityStats{}
	var oldest, newest sql.NullString
	err := s.db.QueryRow(`SELECT COUNT(*), substr(MIN(started_at),1,10), substr(MAX(started_at),1,10) FROM activity_segments`).Scan(&out.Segments, &oldest, &newest)
	if err != nil {
		return out, err
	}
	if oldest.Valid {
		out.Oldest = oldest.String
	}
	if newest.Valid {
		out.Newest = newest.String
	}
	if s.path != "" {
		if fi, err := os.Stat(s.path); err == nil {
			out.DBBytes = fi.Size()
		}
	}
	return out, nil
}

// PruneActivity deletes segments older than keepDays (by started_at date) so
// the database never stacks up unbounded. keepDays<=0 keeps everything
// (no-op). Returns the removed row count; vacuums when anything was removed.
func (s *Store) PruneActivity(keepDays int) (int64, error) {
	if keepDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -keepDays).Format("2006-01-02")
	res, err := s.db.Exec(`DELETE FROM activity_segments WHERE substr(started_at,1,10) < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		// Reclaim the freed pages so the file actually shrinks.
		_, _ = s.db.Exec(`VACUUM`)
	}
	return n, nil
}

// ActivityDayTotal is a tiny helper for distraction threshold checks.
func (s *Store) ActivityDayTotal(category, day string) int64 {
	day = normDay(day)
	var v sql.NullInt64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(seconds),0) FROM activity_segments WHERE category=? AND substr(started_at,1,10)=?`,
		validActivityCategory(category), day).Scan(&v)
	if v.Valid {
		return v.Int64
	}
	return 0
}
