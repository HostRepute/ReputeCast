package tracker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/tracker"
)

const (
	sitesPath = "/workspaces/acme-newsletter/tracker/sites"
	chrome    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
)

type created struct {
	Site           tracker.TrackerSite `json:"site"`
	IdentifySecret string              `json:"identifySecret"`
}

func one[T any](t *testing.T, env *apptest.Env, q string, args ...any) T {
	t.Helper()
	var v T
	if err := env.App.Pool.QueryRow(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return v
}

// Sites: created, listed, changed, the secret rotated and deleted, each
// audited; the secret only in the create and rotate answers; validation in
// sentences; a Viewer reads but can't change.
func TestSitesAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	var out created
	c.Do("POST", sitesPath, map[string]any{"name": "Acme shop", "origin": "Shop.Example.org/products", "waitConsent": true}).MustStatus(t, 201).JSON(t, &out)
	s := out.Site
	if s.Origin != "https://shop.example.org" || !s.WaitConsent || !s.HasIdentifySecret || !strings.HasPrefix(out.IdentifySecret, "rcid_") || len(out.IdentifySecret) != 48 {
		t.Fatalf("created: %+v %q", s, out.IdentifySecret)
	}
	if !strings.Contains(s.Snippet.Script, `/track/v1/rc.js" data-workspace="acme-newsletter" data-site="`+s.ID+`"`) || strings.Contains(s.Snippet.Script, "data-wait-consent") ||
		!strings.Contains(s.Snippet.ShopifyPixel, `analytics.subscribe("product_viewed"`) || !strings.Contains(s.Snippet.ShopifyPixel, "/p/acme-newsletter/track/"+s.ID+"/e") ||
		!strings.Contains(s.Snippet.WordPress, s.ID) {
		t.Fatalf("snippet: %+v", s.Snippet)
	}
	list := c.Do("GET", sitesPath, nil).MustStatus(t, 200)
	if bytes.Contains(list.Body, []byte(out.IdentifySecret)) || bytes.Contains(list.Body, []byte("rcid_")) {
		t.Fatal("the list holds the secret")
	}
	var l tracker.TrackerSiteList
	list.JSON(t, &l)
	// The demo's Acme Merch and Docs, then the new site.
	if len(l.Items) != 3 || l.Items[2].ID != s.ID || !strings.HasSuffix(l.ScriptURL, "/track/v1/rc.js") || len(l.Stores) == 0 || l.Items[2].Refused == nil ||
		!l.CanManage || len(l.Helpers) == 0 {
		t.Fatalf("list: %+v", l)
	}

	// Validation, in sentences.
	for _, tc := range []struct {
		body   map[string]any
		status int
		says   string
	}{
		{map[string]any{"name": "", "origin": "https://a.example.org", "waitConsent": true}, 422, "Name the site"},
		{map[string]any{"name": "Plain", "origin": "http://plain.example.org", "waitConsent": true}, 422, "https://shop.acme.io"},
		{map[string]any{"name": "IP", "origin": "https://192.0.2.1", "waitConsent": true}, 422, "https://shop.acme.io"},
		{map[string]any{"name": "Again", "origin": "https://shop.example.org", "waitConsent": true}, 409, "already a tracked site"},
		{map[string]any{"name": "Store", "origin": "https://b.example.org", "storeId": uuid.NewString(), "waitConsent": true}, 422, "Pick a store"},
	} {
		r := c.Do("POST", sitesPath, tc.body)
		if r.Status != tc.status || !strings.Contains(string(r.Body), tc.says) {
			t.Errorf("%v: %d %s", tc.body, r.Status, r.Body)
		}
	}
	// ReputeCast's own pages are never tracked: the workspace's custom
	// domain (the demo's news.acme.io).
	if r := c.Do("POST", sitesPath, map[string]any{"name": "Own", "origin": "https://news.acme.io", "waitConsent": true}); r.Status != 422 || !strings.Contains(string(r.Body), "never tracked") {
		t.Errorf("own host: %d %s", r.Status, r.Body)
	}

	var upd tracker.TrackerSite
	c.Do("PUT", sitesPath+"/"+s.ID, map[string]any{"name": "Acme merch", "waitConsent": false}).MustStatus(t, 200).JSON(t, &upd)
	if upd.Name != "Acme merch" || upd.WaitConsent || !strings.Contains(upd.Snippet.Script, `data-wait-consent="false"`) {
		t.Fatalf("updated: %+v", upd)
	}
	var sec tracker.TrackerSecret
	c.Do("POST", sitesPath+"/"+s.ID+"/secret", nil).MustStatus(t, 200).JSON(t, &sec)
	if !strings.HasPrefix(sec.IdentifySecret, "rcid_") || sec.IdentifySecret == out.IdentifySecret {
		t.Fatalf("rotated: %q", sec.IdentifySecret)
	}
	var recent tracker.TrackerRecentEvents
	c.Do("GET", sitesPath+"/"+s.ID+"/events/recent", nil).MustStatus(t, 200).JSON(t, &recent)
	if recent.Items == nil || len(recent.Items) != 0 {
		t.Fatalf("recent: %+v", recent)
	}

	// The Viewer reads, and can't change.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", sitesPath, nil).MustStatus(t, 200)
	v.Do("POST", sitesPath, map[string]any{"name": "Nope", "origin": "https://nope.example.org", "waitConsent": true}).MustStatus(t, 403)
	v.Do("DELETE", sitesPath+"/"+s.ID, nil).MustStatus(t, 403)

	c.Do("DELETE", sitesPath+"/"+s.ID, nil).MustStatus(t, 204)
	c.Do("GET", sitesPath+"/"+s.ID+"/events/recent", nil).MustStatus(t, 404)
	c.Do("DELETE", sitesPath+"/"+s.ID, nil).MustStatus(t, 404)

	rows, err := env.App.Pool.Query(context.Background(), `SELECT action, coalesce(after::text, '') || coalesce(before::text, '') FROM audit_events WHERE target_type = 'tracker_site' AND target_id = $1 ORDER BY occurred_at, id`, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	var actions []string
	for rows.Next() {
		var a, data string
		_ = rows.Scan(&a, &data)
		if strings.Contains(data, "rcid_") {
			t.Errorf("%s logged the secret", a)
		}
		actions = append(actions, a)
	}
	rows.Close()
	if strings.Join(actions, ",") != "tracker_site.created,tracker_site.updated,tracker_site.secret_rotated,tracker_site.deleted" {
		t.Fatalf("audit: %v", actions)
	}
}

// collector posts like rc.js does.
type collector struct {
	t   *testing.T
	url string
}

func (c *collector) post(origin, ua string, body any) int {
	c.t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", ua)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func page(path string, extra map[string]any) map[string]any {
	ev := map[string]any{"t": "page", "at": time.Now().UnixMilli(), "url": "https://shop.example.org" + path}
	for k, v := range extra {
		ev[k] = v
	}
	return ev
}

// The collect endpoint: events from the site's origin are written, from
// another origin refused and counted; bots, caps and a waiting site
// without consent keep nothing; a bad signature links nobody; a click
// token or a form's link token links the visitor and joins their last 30
// days to the contact; product views reach the contact's interest.
func TestCollect(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	store := one[uuid.UUID](t, env, `SELECT s.id FROM stores s JOIN workspaces w ON w.id = s.workspace_id WHERE w.slug = 'acme-newsletter' AND s.status IN ('connected','importing') ORDER BY s.created_at LIMIT 1`)
	product := one[string](t, env, `SELECT external_id FROM store_products WHERE store_id = $1 AND removed_at IS NULL ORDER BY external_id LIMIT 1`, store)
	var out created
	c.Do("POST", sitesPath, map[string]any{"name": "Acme shop", "origin": "https://shop.example.org", "storeId": store.String(), "waitConsent": true}).MustStatus(t, 201).JSON(t, &out)
	site := out.Site.ID
	ws := one[uuid.UUID](t, env, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`)
	contact := one[uuid.UUID](t, env, `SELECT id FROM contacts c WHERE workspace_id = $1 AND erased_at IS NULL AND status = 'active'
		AND NOT EXISTS (SELECT 1 FROM tracker_visitors v WHERE v.contact_id = c.id) ORDER BY id LIMIT 1`, ws)
	col := &collector{t: t, url: env.Server.URL + "/p/acme-newsletter/track/" + site + "/e"}
	flush := func() {
		t.Helper()
		if err := env.App.Tracker().Close(ctx); err != nil {
			t.Fatal(err)
		}
	}
	events := func(where string, args ...any) int {
		t.Helper()
		return one[int](t, env, `SELECT count(*)::int FROM tracker_events WHERE site_id = '`+site+`' AND `+where, args...)
	}
	const origin = "https://shop.example.org"
	vid := "AAAAAAAAAAAAAAAAAAAAAA"

	// A waiting site: nothing without consent; with it, the page is kept
	// without its query but utm_*.
	if s := col.post(origin, chrome, map[string]any{"vid": vid, "events": []any{page("/", nil)}}); s != 204 {
		t.Fatalf("no consent: %d", s)
	}
	col.post(origin, chrome, map[string]any{"vid": vid, "c": 1, "events": []any{page("/hoodie?utm_source=news&token=x", map[string]any{"ref": "https://www.google.com/"})}})
	flush()
	if n := events("true"); n != 1 {
		t.Fatalf("events %d", n)
	}
	if p := one[string](t, env, `SELECT path || '|' || referrer_domain FROM tracker_events WHERE site_id = $1`, site); p != "/hoodie?utm_source=news|google.com" {
		t.Fatalf("stored %q", p)
	}

	// Another origin: refused and counted; a bot, Global Privacy Control and
	// a deleted site's id: nothing.
	if s := col.post("https://copycat.example", chrome, map[string]any{"vid": vid, "c": 1, "events": []any{page("/", nil)}}); s != 403 {
		t.Fatalf("other origin: %d", s)
	}
	col.post(origin, "Googlebot/2.1 (+http://www.google.com/bot.html)", map[string]any{"vid": vid, "c": 1, "events": []any{page("/bot", nil)}})
	col.post(origin, "", map[string]any{"vid": vid, "c": 1, "events": []any{page("/empty-ua", nil)}})
	big := map[string]any{"vid": vid, "c": 1, "events": []any{page("/"+strings.Repeat("x", 9000), nil)}}
	if s := col.post(origin, chrome, big); s != 413 {
		t.Fatalf("a 9 KB body: %d", s)
	}
	if s := (&collector{t: t, url: env.Server.URL + "/p/acme-newsletter/track/" + uuid.NewString() + "/e"}).post(origin, chrome, map[string]any{"vid": vid, "c": 1, "events": []any{}}); s != 404 {
		t.Fatalf("unknown site: %d", s)
	}
	flush()
	if n := events("true"); n != 1 {
		t.Fatalf("bots or refusals kept: %d events", n)
	}
	if n := one[int64](t, env, `SELECT count FROM tracker_site_refusals WHERE site_id = $1 AND origin = 'https://copycat.example'`, site); n != 1 {
		t.Fatalf("refusals %d", n)
	}
	var l tracker.TrackerSiteList
	c.Do("GET", sitesPath, nil).MustStatus(t, 200).JSON(t, &l)
	var mine tracker.TrackerSite
	for _, it := range l.Items {
		if it.ID == site {
			mine = it
		}
	}
	if len(mine.Refused) != 1 || mine.Refused[0].Origin != "https://copycat.example" || mine.EventsToday != 1 || mine.LastEventAt == nil {
		t.Fatalf("site: %+v", mine)
	}

	// An identify call with a bad signature links nobody.
	email := one[string](t, env, `SELECT email FROM contacts WHERE id = $1`, contact)
	col.post(origin, chrome, map[string]any{"vid": vid, "c": 1, "identify": map[string]any{"email": email, "signature": strings.Repeat("ab", 32)}, "events": []any{page("/x", nil)}})
	flush()
	if n := one[int](t, env, `SELECT count(*)::int FROM tracker_visitors WHERE contact_id = $1`, contact); n != 0 {
		t.Fatal("a bad signature linked the visitor")
	}

	// An event from 40 days ago stays the visitor's; the click token links
	// the visitor and joins the last 30 days.
	visitor := one[uuid.UUID](t, env, `SELECT visitor_id FROM tracker_events WHERE site_id = $1 LIMIT 1`, site)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO tracker_events (id, workspace_id, site_id, visitor_id, kind, path, occurred_at) VALUES ($1, $2, $3, $4, 1, '/old', now() - interval '40 days')`,
		uuid.Must(uuid.NewV7()), ws, out.Site.ID, visitor); err != nil {
		t.Fatal(err)
	}
	rc := clicktoken.Sign(env.App.Links, clicktoken.Click{WorkspaceID: ws, MessageID: uuid.Must(uuid.NewV7()), ContactID: contact, At: time.Now()})
	col.post(origin, chrome, map[string]any{"vid": vid, "c": 1, "rc": rc, "events": []any{map[string]any{"t": "product", "at": time.Now().UnixMilli(), "url": origin + "/p", "id": product}}})
	flush()
	if by := one[string](t, env, `SELECT identified_by FROM tracker_visitors WHERE id = $1 AND contact_id = $2`, visitor, contact); by != "click" {
		t.Fatalf("identified by %q", by)
	}
	if n := events("contact_id = $1", contact); n != 3 {
		t.Fatalf("%d events joined the contact, want the 2 recent and the new one", n)
	}
	if n := events("contact_id IS NULL AND path = '/old'"); n != 1 {
		t.Fatal("an event older than 30 days joined the contact")
	}
	if n := one[int](t, env, `SELECT views FROM tracker_product_views v JOIN store_products p ON p.id = v.product_id WHERE v.contact_id = $1 AND p.external_id = $2 AND p.store_id = $3`, contact, product, store); n != 1 {
		t.Fatalf("views %d", n)
	}

	// A form's link token links another browser; an unknown product stays
	// a product view without a product.
	other := one[uuid.UUID](t, env, `SELECT id FROM contacts c WHERE workspace_id = $1 AND erased_at IS NULL AND id <> $2
		AND NOT EXISTS (SELECT 1 FROM tracker_visitors v WHERE v.contact_id = c.id) ORDER BY id LIMIT 1`, ws, contact)
	link := tracker.FormLink(env.App.Links, ws, other, time.Now())
	col.post(origin, chrome, map[string]any{"vid": "BBBBBBBBBBBBBBBBBBBBBB", "c": 1, "link": link, "events": []any{map[string]any{"t": "product", "url": origin + "/p", "id": "no-such-product"}}})
	flush()
	if by := one[string](t, env, `SELECT identified_by FROM tracker_visitors WHERE contact_id = $1`, other); by != "form" {
		t.Fatalf("form link: %q", by)
	}
	if n := events("contact_id = $1 AND kind = 2 AND product_id IS NULL", other); n != 1 {
		t.Fatalf("unknown product: %d", n)
	}

	// The timeline has a line for the day.
	var tl struct {
		Items []struct{ Type, Subject string } `json:"items"`
	}
	c.Do("GET", fmt.Sprintf("/workspaces/acme-newsletter/contacts/%s/activity", contact), nil).JSON(t, &tl)
	found := false
	for _, it := range tl.Items {
		found = found || (it.Type == "site_visit" && strings.HasPrefix(it.Subject, "Viewed 1 product on shop.example.org"))
	}
	if !found {
		t.Fatalf("timeline: %+v", tl.Items)
	}

	// A visitor nobody links stays anonymous (the retention below).
	col.post(origin, chrome, map[string]any{"vid": "CCCCCCCCCCCCCCCCCCCCCC", "c": 1, "events": []any{page("/anon", nil)}})
	flush()
	if n := events("path = '/anon' AND contact_id IS NULL"); n != 1 {
		t.Fatalf("anonymous events %d", n)
	}

	// The per-address limit: 240 calls a minute.
	limited := &collector{t: t, url: col.url}
	status := 0
	for range 241 {
		status = limited.post(origin, chrome, map[string]any{"vid": vid, "c": 1, "events": []any{}})
	}
	if status != 429 {
		t.Fatalf("the 241st call: %d", status)
	}

	// Retention: anonymous visitors unseen 30 days go with their events.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE tracker_visitors SET last_seen = now() - interval '31 days' WHERE contact_id IS NULL`); err != nil {
		t.Fatal(err)
	}
	res, err := env.App.Tracker().Retain(ctx)
	if err != nil || res.Visitors < 1 {
		t.Fatalf("retention: %+v %v", res, err)
	}
	if n := events("path = '/anon'"); n != 0 {
		t.Fatalf("%d anonymous events left", n)
	}
	// A linked visitor's events older than 30 days stay theirs until the
	// 90 days: retention leaves them.
	if n := events("path = '/old'"); n != 1 {
		t.Fatalf("the linked visitor's old event: %d", n)
	}
}

// BenchmarkCollect drives the collect endpoint and the writer end to end
// into tracker_events: 20 events a call from distinct addresses, and
// reports events written per second.
func BenchmarkCollect(b *testing.B) {
	env := apptest.New(b)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "bench@tracker.test"})
	ws := env.CreateWorkspace("Tracker bench", u.ID)
	site := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO tracker_sites (id, workspace_id, name, origin, wait_consent, created_at, updated_at) VALUES ($1, $2, 'Bench', 'https://bench.example.org', false, now(), now())`,
		site, ws.ID); err != nil {
		b.Fatal(err)
	}
	s := env.App.Tracker()
	r := chi.NewRouter()
	s.Routes(r)
	evs := make([]any, tracker.MaxEvents)
	for i := range evs {
		evs[i] = map[string]any{"t": "page", "url": fmt.Sprintf("https://bench.example.org/p/%d?utm_source=bench", i)}
	}
	path := "/p/" + ws.Slug + "/track/" + site.String() + "/e"
	before := s.WriterStats()
	b.ResetTimer()
	start := time.Now()
	for i := range b.N {
		body, _ := json.Marshal(map[string]any{"vid": fmt.Sprintf("bench%017d", i%5000), "events": evs})
		req := httptest.NewRequest("POST", path, bytes.NewReader(body))
		req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", (i>>16)&255, (i>>8)&255, i&255)
		req.Header.Set("Origin", "https://bench.example.org")
		req.Header.Set("User-Agent", chrome)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 204 {
			b.Fatalf("call %d: %d", i, w.Code)
		}
	}
	if err := s.Close(ctx); err != nil {
		b.Fatal(err)
	}
	took := time.Since(start)
	b.StopTimer()
	st := s.WriterStats()
	written := st.Written - before.Written
	if st.Dropped > before.Dropped {
		b.Logf("dropped %d events (the queue was full)", st.Dropped-before.Dropped)
	}
	b.ReportMetric(float64(written)/took.Seconds(), "events/s")
}
