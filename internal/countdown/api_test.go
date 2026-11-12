package countdown_test

import (
	"bytes"
	"context"
	"image/gif"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/countdown"
	"github.com/hostrepute/reputecast/internal/demo"
)

func settings(endsAt string) map[string]any {
	return map[string]any{"endsAt": endsAt, "timeZone": "America/New_York", "units": "dhms", "language": "en", "size": "l",
		"background": "#0F172A", "digits": "#FFFFFF", "labels": "#CBD5E1", "ended": map[string]any{"mode": "zeros"}}
}

// The composer posts a block's settings and gets its image; the image is
// drawn on each open, never cached while it runs.
func TestCountdownImage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	ends := time.Now().In(mustZone(t)).AddDate(0, 0, 3).Format("2006-01-02") + "T23:59"
	var img countdown.CountdownImage
	linh.Do("POST", "/workspaces/acme-newsletter/countdowns", settings(ends)).MustStatus(t, 200).JSON(t, &img)
	if img.ID == "" || !strings.Contains(img.URL, "/t/cd/") || !strings.HasSuffix(img.URL, ".gif") || !strings.HasPrefix(img.Alt, "Offer ends ") ||
		!strings.HasSuffix(img.Alt, "at 23:59 (EST)") || img.Width != 600 || img.Height != 150 {
		t.Fatalf("image %+v", img)
	}
	var again countdown.CountdownImage
	linh.Do("POST", "/workspaces/acme-newsletter/countdowns", settings(ends)).MustStatus(t, 200).JSON(t, &again)
	if again.ID != img.ID || again.URL != img.URL {
		t.Fatalf("the same settings gave %s, then %s", img.ID, again.ID)
	}
	var audits int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'countdown.created' AND target_id = $1`, img.ID).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audit entries %d %v", audits, err)
	}
	low := settings(ends)
	low["digits"] = "#334155"
	p := linh.Do("POST", "/workspaces/acme-newsletter/countdowns", low).MustStatus(t, 422).Problem(t)
	if errs, _ := p["errors"].([]any); len(errs) == 0 || errs[0].(map[string]any)["location"] != "body.digits" || !strings.Contains(p["detail"].(string), "1.7:1") {
		t.Fatalf("low contrast: %v", p)
	}
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("POST", "/workspaces/acme-newsletter/countdowns", settings(ends)).MustStatus(t, 403)

	u, err := url.Parse(img.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := linh.DoRaw("GET", u.Path, "", nil).MustStatus(t, 200)
	h := r.Header
	if h.Get("Content-Type") != "image/gif" || !strings.Contains(h.Get("Cache-Control"), "no-store") || h.Get("Pragma") != "no-cache" ||
		h.Get("Cross-Origin-Resource-Policy") != "cross-origin" || !strings.Contains(h.Get("X-Robots-Tag"), "noindex") {
		t.Fatalf("headers %v", h)
	}
	g, err := gif.DecodeAll(bytes.NewReader(r.Body))
	if err != nil || len(g.Image) != 60 || g.LoopCount != -1 {
		t.Fatalf("gif: %v", err)
	}
	if head := linh.DoRaw("HEAD", u.Path, "", nil).MustStatus(t, 200); head.Header.Get("Content-Type") != "image/gif" || len(head.Body) != 0 {
		t.Fatalf("HEAD %v", head.Header)
	}
	linh.DoRaw("GET", "/t/cd/"+strings.Repeat("A", 92)+".gif", "", nil).MustStatus(t, 404)
	// An open token isn't a countdown's.
	open := strings.TrimSuffix(strings.TrimPrefix(u.Path, "/t/cd/"), ".gif")
	linh.DoRaw("GET", "/t/cd/"+open[:len(open)-2]+"xx.gif", "", nil).MustStatus(t, 404)

	// Ended: one frame, cached an hour.
	var past countdown.CountdownImage
	linh.Do("POST", "/workspaces/acme-newsletter/countdowns", settings("2026-01-02T09:00")).MustStatus(t, 200).JSON(t, &past)
	pu, _ := url.Parse(past.URL)
	r = linh.DoRaw("GET", pu.Path, "", nil).MustStatus(t, 200)
	if r.Header.Get("Cache-Control") != "public, max-age=3600" {
		t.Fatalf("ended headers %v", r.Header)
	}
	if g, err := gif.DecodeAll(bytes.NewReader(r.Body)); err != nil || len(g.Image) != 1 {
		t.Fatalf("ended gif: %v", err)
	}
}

// The demo's Black Friday draft shows a running countdown and its image.
func TestDemoCountdown(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	var src string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT mjml FROM campaigns WHERE name = $1`, demo.CountdownCampaign).Scan(&src); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "rc-countdown=") || !strings.Contains(src, `<!-- rc:coupon {"pool":"welcome15"`) {
		t.Fatalf("mjml %s", src)
	}
	i := strings.Index(src, "/t/cd/")
	path := src[i : i+strings.Index(src[i:], `"`)]
	c := env.Client()
	r := c.DoRaw("GET", path, "", nil).MustStatus(t, 200)
	if g, err := gif.DecodeAll(bytes.NewReader(r.Body)); err != nil || len(g.Image) != 60 {
		t.Fatalf("demo gif: %v", err)
	}
}

func mustZone(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}
