package reports_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/migrations"
)

// A read-only link works for 1 to 90 days whoever makes it: the API's
// schema holds its callers to that, and agents (MCP) and other callers of
// CreateShare are held to it there (review AN-16).
func TestShareDaysLimit(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	ws, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse(campaignID(t, c, "October Product Update"))
	s := &reports.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, URLs: env.App.URLs, Limits: reports.NewLimiter()}
	for _, days := range []int{-1, 91, 36500} {
		_, err := s.CreateShare(ctx, &ws, id, reports.ShareInput{ExpiresInDays: days})
		var p *api.Problem
		if !errors.As(err, &p) || p.Status != http.StatusUnprocessableEntity || !strings.Contains(p.Detail, "1 to 90 days") {
			t.Fatalf("a %d-day link: %v", days, err)
		}
	}
	// Absent, 30 days.
	for days, want := range map[int]int{0: 30, 1: 1, 90: 90} {
		l, err := s.CreateShare(ctx, &ws, id, reports.ShareInput{ExpiresInDays: days})
		if err != nil || l.ExpiresAt == nil || time.Until(*l.ExpiresAt) < time.Duration(want)*24*time.Hour-time.Hour ||
			time.Until(*l.ExpiresAt) > time.Duration(want)*24*time.Hour {
			t.Fatalf("a link asked for %d days: %+v, %v", days, l, err)
		}
	}
}

// Links made before every link expired (migration 03208): each one that
// still works expires 30 days after the upgrade; revoked links and links
// with their own expiry stay as they were.
func TestShareExpiryBackfill(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	campaign := campaignID(t, c, "October Product Update")
	p, err := goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(env.App.Pool), migrations.FS, goose.WithAllowOutofOrder(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	const version = 3208
	if _, err := p.ApplyVersion(ctx, version, false); err != nil {
		t.Fatal(err)
	}
	dated := time.Now().Add(7 * 24 * time.Hour).Truncate(time.Second)
	links := map[string]struct {
		expires *time.Time
		revoked bool
	}{"open": {}, "revoked": {revoked: true}, "dated": {expires: &dated}}
	ids := map[string]uuid.UUID{}
	for name, l := range links {
		ids[name] = uuid.Must(uuid.NewV7())
		var revoked *time.Time
		if l.revoked {
			now := time.Now()
			revoked = &now
		}
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO report_shares (id, campaign_id, token_hash, expires_at, revoked_at) VALUES ($1, $2, $3, $4, $5)`,
			ids[name], campaign, []byte(name), l.expires, revoked); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := p.ApplyVersion(ctx, version, true); err != nil {
		t.Fatal(err)
	}
	expiry := func(name string) *time.Time {
		t.Helper()
		var at *time.Time
		if err := env.App.Pool.QueryRow(ctx, `SELECT expires_at FROM report_shares WHERE id = $1`, ids[name]).Scan(&at); err != nil {
			t.Fatal(err)
		}
		return at
	}
	if at := expiry("open"); at == nil || time.Until(*at) < 30*24*time.Hour-time.Hour || time.Until(*at) > 30*24*time.Hour {
		t.Fatalf("a link without an expiry: %v", at)
	}
	if at := expiry("revoked"); at != nil {
		t.Fatalf("a revoked link: %v", at)
	}
	if at := expiry("dated"); at == nil || !at.Equal(dated) {
		t.Fatalf("a link with its own expiry: %v, want %v", at, dated)
	}
}

// A read-only link that doesn't share the warm-up state never carries the
// warm-up day: not in the sending providers' columns, not in an insight's
// words. A link that shares it does (review AN-16).
func TestSharedReportWarmup(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	base := "/workspaces/acme-newsletter/campaigns/" + campaignID(t, c, "October Product Update") + "/report"
	type insight struct{ Kind, Detail string }
	blocked := func(ins []insight) string {
		for _, i := range ins {
			if i.Kind == "blocked" {
				return i.Detail
			}
		}
		return ""
	}
	var rep struct{ Insights []insight }
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &rep)
	// The report says which warm-up day the block came on (the Report
	// board: "All 11 went through Postal's IP 203.0.113.25, on warm-up day 6").
	inReport := blocked(rep.Insights)
	if !strings.Contains(inReport, ", on warm-up day ") {
		t.Fatalf("the report's blocked insight: %q", inReport)
	}
	shared := func(body map[string]any, lang string) (string, []byte) {
		t.Helper()
		var link struct{ URL string }
		c.Do("POST", base+"/shares", body).MustStatus(t, 201).JSON(t, &link)
		anon := env.Client()
		if lang != "" {
			anon.Header = http.Header{"Accept-Language": {lang}}
		}
		res := anon.Do("GET", "/shared-reports/"+link.URL[strings.LastIndex(link.URL, "/")+1:], nil).MustStatus(t, 200)
		var out struct{ Report struct{ Insights []insight } }
		res.JSON(t, &out)
		return blocked(out.Report.Insights), res.Body
	}
	got, body := shared(map[string]any{}, "")
	if want := regexp.MustCompile(`, on warm-up day \d+`).ReplaceAllString(inReport, ""); got != want ||
		bytes.Contains(body, []byte("warm-up day")) || bytes.Contains(body, []byte("warmupDay")) {
		t.Fatalf("a link without the warm-up state: %q, want %q", got, want)
	}
	if got, body := shared(map[string]any{}, "vi"); got == "" || bytes.Contains(body, []byte("ngày khởi động")) {
		t.Fatalf("a link without the warm-up state, in Vietnamese: %q", got)
	}
	if got, _ := shared(map[string]any{"includeWarmup": true}, ""); got != inReport {
		t.Fatalf("a link with the warm-up state: %q, want %q", got, inReport)
	}
}
