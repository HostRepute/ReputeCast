package publicpages_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/website"
)

// oldSite answers every image address with a small PNG, as the old site
// would (no network in tests).
type oldSite struct{ png []byte }

func (o oldSite) Get(_ context.Context, r feed.Request) (*feed.Response, error) {
	return &feed.Response{URL: r.URL, Status: 200, Body: o.png, ContentType: "image/png"}, nil
}

func smallPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for x := range 40 {
		for y := range 30 {
			img.Set(x, y, color.RGBA{R: uint8(x * 6), G: 120, B: uint8(y * 8), A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// Importing posts (M61, G-26): only people with website:publish import; a
// file that isn't an export is refused; the Ghost fixture imports with its
// original dates, without post.published webhooks or an audit entry per
// post, as one website.posts_imported entry; the old addresses answer 301
// on the site's custom domain and anything else there is the site's 404;
// the same export again finds every post imported and imports nothing.
func TestImportGhostPosts(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	env.App.Website().Images = func() website.ImageFetcher { return oldSite{png: smallPNG(t)} }
	ghost, err := os.ReadFile("../website/importer/testdata/ghost.json")
	if err != nil {
		t.Fatal(err)
	}
	// A webhook that listens for published posts.
	exec(`INSERT INTO webhooks (id, workspace_id, url, events, secret_enc) VALUES ($1, $2, 'https://example.test/hook', '{post.published}', '\x00')`,
		uuid.Must(uuid.NewV7()), ids.ws)
	const imports = wsAPI + "/website/imports"
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.DoRaw(http.MethodPost, imports+"?fileName=acme.ghost.json", "application/octet-stream", bytes.NewReader(ghost)).MustStatus(t, http.StatusForbidden)
	writer := env.Client()
	writer.SignInDemo("daniel@acme.io")
	writer.DoRaw(http.MethodPost, imports+"?fileName=acme.ghost.json", "application/octet-stream", bytes.NewReader(ghost)).MustStatus(t, http.StatusForbidden)

	o := owner(env)
	csv := []byte("email,first_name\nsam.rivera@outlook.com,Sam\n")
	r := o.DoRaw(http.MethodPost, imports+"?fileName=acme-subscribers.csv", "application/octet-stream", bytes.NewReader(csv)).MustStatus(t, http.StatusUnprocessableEntity)
	if !strings.Contains(string(r.Body), "isn't a Substack, Ghost or WordPress export") {
		t.Errorf("a CSV: %s", r.Body)
	}

	// The site off: nowhere for the posts to go.
	exec(`UPDATE website_settings SET status = 'off' WHERE workspace_id = $1`, ids.ws)
	o.DoRaw(http.MethodPost, imports+"?fileName=acme.ghost.json", "application/octet-stream", bytes.NewReader(ghost)).MustStatus(t, http.StatusConflict)
	exec(`UPDATE website_settings SET status = 'on' WHERE workspace_id = $1`, ids.ws)

	// The background jobs (website.import_analyse, website.import_run) run
	// here as the worker would.
	upload := func() website.WebsiteImport {
		t.Helper()
		var imp website.WebsiteImport
		o.DoRaw(http.MethodPost, imports+"?fileName=acme.ghost.json", "application/octet-stream", bytes.NewReader(ghost)).MustStatus(t, http.StatusCreated).JSON(t, &imp)
		if err := env.App.Website().RunImportAnalyse(ctx, uuid.MustParse(imp.ID)); err != nil {
			t.Fatal(err)
		}
		o.Do(http.MethodGet, imports+"/"+imp.ID, nil).MustStatus(t, http.StatusOK).JSON(t, &imp)
		return imp
	}
	imp := upload()
	if imp.Source != "ghost" || imp.Status != website.ImportReady || imp.FileName != "acme.ghost.json" {
		t.Fatalf("after the upload: %s %s %q", imp.Source, imp.Status, imp.FileName)
	}
	if imp.Counts.Found != 5 || imp.Counts.ToImport != 4 || imp.Counts.DraftsSkipped != 1 || imp.Counts.Paid != 2 || imp.Counts.Members != 1 {
		t.Fatalf("the review's numbers: %+v", imp.Counts)
	}
	path := imports + "/" + imp.ID
	writer.Do(http.MethodPost, path+"/start", nil).MustStatus(t, http.StatusForbidden)
	o.Do(http.MethodGet, path+"/report.csv", nil).MustStatus(t, http.StatusNotFound)
	o.Do(http.MethodPut, path, map[string]any{"tiers": map[string]string{"Founding": "not-a-tier"}}).MustStatus(t, http.StatusUnprocessableEntity)

	o.Do(http.MethodPost, path+"/start", nil).MustStatus(t, http.StatusOK)
	if err := env.App.Website().RunImport(ctx, uuid.MustParse(imp.ID)); err != nil {
		t.Fatal(err)
	}
	o.Do(http.MethodGet, path, nil).MustStatus(t, http.StatusOK).JSON(t, &imp)
	if imp.Status != website.ImportDone || imp.Progress == nil || imp.Progress.Published != 4 || imp.Progress.Skipped != 1 {
		t.Fatalf("after the run: %s %+v", imp.Status, imp.Progress)
	}
	if imp.ReportURL == "" {
		t.Error("no report")
	}

	// Published with the original date, the old address kept.
	at := one[time.Time](t, env, `SELECT published_at FROM website_posts WHERE workspace_id = $1 AND import_source = 'ghost' AND import_source_id = '65f1c0a2b7e4d90001a3c401'`, ids.ws)
	if want := time.Date(2025, 9, 9, 14, 0, 0, 0, time.UTC); !at.Equal(want) {
		t.Errorf("my-old-slug published %s, want %s", at, want)
	}
	if n := count(t, env, `SELECT count(*) FROM website_posts WHERE workspace_id = $1 AND import_source = 'ghost' AND status = 'published'`, ids.ws); n != 4 {
		t.Errorf("%d imported posts published", n)
	}
	if n := count(t, env, `SELECT count(*) FROM webhook_deliveries d JOIN webhooks h ON h.id = d.webhook_id WHERE h.workspace_id = $1 AND d.event = 'post.published'`, ids.ws); n != 0 {
		t.Errorf("%d post.published deliveries", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'post.published' AND target_id IN
		(SELECT id::text FROM website_posts WHERE workspace_id = $1 AND import_source = 'ghost')`, ids.ws); n != 0 {
		t.Errorf("%d post.published audit entries", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'website.posts_imported' AND target_id = $2`, ids.ws, imp.ID); n != 1 {
		t.Errorf("%d website.posts_imported entries", n)
	}
	if n := count(t, env, `SELECT count(*) FROM website_import_files WHERE import_id = $1`, imp.ID); n != 0 {
		t.Error("the export is kept after the import")
	}
	rep := o.Do(http.MethodGet, path+"/report.csv", nil).MustStatus(t, http.StatusOK)
	if ct := rep.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") || !strings.Contains(string(rep.Body), "Title,Old address,Outcome,Problems") ||
		!strings.Contains(string(rep.Body), "Published") || !strings.Contains(string(rep.Body), "Skipped") {
		t.Errorf("the report (%s):\n%s", ct, rep.Body)
	}

	// The old addresses on the site's custom domain.
	slug := one[string](t, env, `SELECT slug FROM website_posts WHERE workspace_id = $1 AND import_source_id = '65f1c0a2b7e4d90001a3c401'`, ids.ws)
	for _, old := range []string{"/my-old-slug/", "/my-old-slug", "/MY-OLD-SLUG/"} {
		r := do(t, env, http.MethodGet, old, nil, newsHost)
		if r.Status != http.StatusMovedPermanently || r.Header.Get("Location") != "http://news.acme.io/posts/"+slug {
			t.Errorf("%s: %d → %q", old, r.Status, r.Header.Get("Location"))
		}
	}
	for old, want := range map[string]string{"/rss/": "/feed", "/feed/": "/feed", "/posts/" + slug + "/": "/posts/" + slug} {
		r := do(t, env, http.MethodGet, old, nil, newsHost)
		if r.Status != http.StatusMovedPermanently || r.Header.Get("Location") != "http://news.acme.io"+want {
			t.Errorf("%s: %d → %q, want %s", old, r.Status, r.Header.Get("Location"), want)
		}
	}
	r404 := do(t, env, http.MethodGet, "/2019/04/nothing-here/", nil, newsHost)
	must(t, r404, http.StatusNotFound)
	lacks(t, r404, `<div id="root">`)
	// The instance's own host keeps the web app.
	if r := do(t, env, http.MethodGet, "/my-old-slug/", nil, nil); r.Status == http.StatusMovedPermanently {
		t.Error("an old address redirects on the instance's own host")
	}

	// The same export again: every post is found, nothing to import.
	again := upload()
	if again.Counts.AlreadyImported != 4 || again.Counts.ToImport != 0 {
		t.Fatalf("the second review: %+v", again.Counts)
	}
	o.Do(http.MethodPost, imports+"/"+again.ID+"/start", nil).MustStatus(t, http.StatusConflict)
	if n := count(t, env, `SELECT count(*) FROM website_posts WHERE workspace_id = $1 AND import_source = 'ghost'`, ids.ws); n != 4 {
		t.Errorf("%d imported posts after the second export", n)
	}
	o.Do(http.MethodPost, imports+"/"+again.ID+"/cancel", nil).MustStatus(t, http.StatusOK)
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'website.import_canceled' AND target_id = $2`, ids.ws, again.ID); n != 1 {
		t.Errorf("%d website.import_canceled entries", n)
	}
}
