package push_test

// Web push over the API on a workspace of its own (never the demo's): the
// keys (the private key sealed and never returned), sites and the service
// worker check, subscriptions from listed sites only, linking to contacts,
// a campaign sent through the in-process push service, subscriptions the
// push service reports gone (404, 410) removed, shown and click tracking,
// the frequency cap, notifications held while a site's worker is missing,
// rotation, prompts and permissions. Nothing reaches a real push service.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/push"
	"github.com/hostrepute/reputecast/internal/push/webpush"
	"github.com/hostrepute/reputecast/internal/push/webpush/webpushtest"
	"github.com/hostrepute/reputecast/internal/store"
)

const (
	chromeUA  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	firefoxUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 15.6; rv:143.0) Gecko/20100101 Firefox/143.0"
	edgeUA    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0"
	androidUA = "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36"
	siteURL   = "https://help.x22.test"
)

func one[T any](t *testing.T, env *apptest.Env, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

// fixture is a workspace of the test's own with its owner signed in.
type fixture struct {
	env   *apptest.Env
	owner *apptest.Client
	ws    store.Workspace
	path  string
	// keys are each subscription's p256dh and auth, by endpoint.
	keys map[string][2]string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "owner@x22.test", Name: "Pat Owner"})
	ws := env.CreateWorkspace("X22 Push Test", u.ID)
	c := env.Client()
	c.SignIn(u.Email, "")
	return &fixture{env: env, owner: c, ws: ws, path: "/workspaces/" + ws.Slug, keys: map[string][2]string{}}
}

// pushOn turns push on with a snippet site whose worker is found, and
// quiet hours off so the clock of the run doesn't matter.
func (f *fixture) pushOn(t *testing.T, capPerDay int) push.PushSite {
	t.Helper()
	f.owner.Do(http.MethodPost, f.path+"/push/enable", map[string]any{}).MustStatus(t, 200)
	var site push.PushSite
	f.owner.Do(http.MethodPost, f.path+"/push/sites", map[string]any{"address": "help.x22.test", "label": "Help center"}).MustStatus(t, 201).JSON(t, &site)
	f.serveWorker()
	f.owner.Do(http.MethodPost, f.path+"/push/sites/"+site.ID+"/check", nil).MustStatus(t, 200).JSON(t, &site)
	if site.Worker.Status != push.WorkerFound {
		t.Fatalf("worker: %+v", site.Worker)
	}
	f.owner.Do(http.MethodPut, f.path+"/push/settings", map[string]any{"frequencyCap": capPerDay, "quietHours": false}).MustStatus(t, 200)
	return site
}

func (f *fixture) serveWorker() {
	f.env.App.PushFake().Serve(siteURL+"/"+push.WorkerFile, webpushtest.Page{Status: 200,
		Header: http.Header{"Content-Type": {"text/javascript; charset=utf-8"}}, Body: push.HostedWorker()})
}

// browser is a client on a listed site's page: cross-site requests with
// the site's Origin and a browser's User-Agent.
func (f *fixture) browser(origin, ua string) *apptest.Client {
	c := f.env.Client()
	c.Header = http.Header{"Origin": {origin}, "Sec-Fetch-Site": {"cross-site"}, "User-Agent": {ua}}
	return c
}

type subscribed struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Linked bool   `json:"linked"`
}

// subscribe allows notifications in a browser of the fake push service.
func (f *fixture) subscribe(t *testing.T, kind, ua string, extra map[string]any) (string, subscribed) {
	t.Helper()
	endpoint, p256dh, auth := f.env.App.PushFake().Subscribe(kind)
	f.keys[endpoint] = [2]string{p256dh, auth}
	body := map[string]any{"endpoint": endpoint, "keys": map[string]string{"p256dh": p256dh, "auth": auth}, "timeZone": "America/Chicago",
		"page": siteURL + "/docs/flags"}
	for k, v := range extra {
		body[k] = v
	}
	var out subscribed
	f.browser(siteURL, ua).Do(http.MethodPost, "/p/"+f.ws.Slug+"/push/subscriptions", body).MustStatus(t, 200).JSON(t, &out)
	if out.Status != "subscribed" || out.ID == "" {
		t.Fatalf("subscribe: %+v", out)
	}
	return endpoint, out
}

func (f *fixture) contact(t *testing.T, email string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	if _, err := f.env.App.Pool.Exec(context.Background(), `INSERT INTO contacts (id, workspace_id, email, first_name, timezone, attributes)
		VALUES ($1, $2, $3, 'Sam', 'America/Chicago', '{}')`, id, f.ws.ID, email); err != nil {
		t.Fatal(err)
	}
	return id
}

// send makes a campaign and sends it as reviewed, then runs the starter
// and the dispatcher (tests run no workers).
func (f *fixture) send(t *testing.T, name, title string) push.PushCampaign {
	t.Helper()
	c := f.sendOnly(t, name, title)
	f.run(t)
	return c
}

// sendOnly makes a campaign and sends it as reviewed; nothing runs yet.
func (f *fixture) sendOnly(t *testing.T, name, title string) push.PushCampaign {
	t.Helper()
	var c push.PushCampaign
	f.owner.Do(http.MethodPost, f.path+"/push-campaigns", map[string]any{"name": name, "title": title,
		"message": "How we ship to 2% of accounts first.", "link": "help.x22.test/blog/flags"}).MustStatus(t, 201).JSON(t, &c)
	var rv push.PushReview
	f.owner.Do(http.MethodGet, f.path+"/push-campaigns/"+c.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Verdict == "blocked" {
		t.Fatalf("review blocked: %+v", rv.Checks)
	}
	var res push.PushSendResult
	f.owner.Do(http.MethodPost, f.path+"/push-campaigns/"+c.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "sending" {
		t.Fatalf("send: %+v", res)
	}
	return c
}

func (f *fixture) run(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	s := f.env.App.Push()
	if err := s.StartDue(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Dispatch(ctx, 10*time.Second); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) actions(t *testing.T) map[string]int {
	t.Helper()
	rows, err := f.env.App.Pool.Query(context.Background(), `SELECT action FROM audit_events WHERE workspace_id = $1`, f.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			t.Fatal(err)
		}
		out[a]++
	}
	return out
}

func TestPushKeysAndSites(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path
	ctx := context.Background()

	var setup push.PushSetup
	c.Do(http.MethodGet, p+"/push/setup", nil).MustStatus(t, 200).JSON(t, &setup)
	if setup.On || setup.Keys != nil || !setup.CanManage {
		t.Fatalf("before: %+v", setup)
	}
	// Sites use the keys: none before push is on.
	c.Do(http.MethodPost, p+"/push/sites", map[string]any{"address": "help.x22.test"}).MustStatus(t, 409)

	r := c.Do(http.MethodPost, p+"/push/enable", map[string]any{}).MustStatus(t, 200)
	r.JSON(t, &setup)
	if !setup.On || setup.Keys == nil || setup.Keys.Contact != "mailto:owner@x22.test" || setup.Cap != push.DefaultFrequencyCap || !setup.Quiet {
		t.Fatalf("after: %+v", setup)
	}
	pub, err := webpush.Decode(setup.Keys.PublicKey)
	if err != nil || len(pub) != 65 || pub[0] != 4 {
		t.Fatalf("public key %q: %v", setup.Keys.PublicKey, err)
	}
	// The private key is sealed and never returned.
	var keyID uuid.UUID
	var enc []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT id, private_enc FROM push_keys WHERE workspace_id = $1`, f.ws.ID).Scan(&keyID, &enc); err != nil {
		t.Fatal(err)
	}
	raw, err := env.App.Secrets.Open(enc, "push_key:"+keyID.String()+":private")
	if err != nil || len(raw) != 32 {
		t.Fatalf("sealed key: %d bytes, %v", len(raw), err)
	}
	if k, err := webpush.ParseVAPIDKey(raw); err != nil || k.PublicKeyString() != setup.Keys.PublicKey {
		t.Fatalf("the sealed key isn't the public key's pair: %v", err)
	}
	setupBody := c.Do(http.MethodGet, p+"/push/setup", nil).MustStatus(t, 200).Body
	for _, b := range [][]byte{r.Body, setupBody} {
		for _, secret := range []string{webpush.Encode(raw), base64.StdEncoding.EncodeToString(raw), base64.StdEncoding.EncodeToString(enc)} {
			if bytes.Contains(b, []byte(secret)) {
				t.Fatal("the private key is in an answer")
			}
		}
	}
	c.Do(http.MethodPost, p+"/push/enable", map[string]any{}).MustStatus(t, 409)
	c.Do(http.MethodPut, p+"/push/contact", map[string]any{"contact": "ftp://acme"}).MustStatus(t, 422)

	// Sites: https only (http for localhost), no addresses by IP.
	c.Do(http.MethodPost, p+"/push/sites", map[string]any{"address": "http://help.x22.test"}).MustStatus(t, 422)
	c.Do(http.MethodPost, p+"/push/sites", map[string]any{"address": "https://203.0.113.9"}).MustStatus(t, 422)
	var site push.PushSite
	c.Do(http.MethodPost, p+"/push/sites", map[string]any{"address": "https://help.x22.test/docs", "label": "Help center"}).MustStatus(t, 201).JSON(t, &site)
	if site.Kind != push.KindSnippet || site.Host != "help.x22.test" || site.Origin != siteURL || site.Worker.Status != push.WorkerUnchecked ||
		site.Worker.URL != siteURL+"/rc-push-sw.js" {
		t.Fatalf("site: %+v", site)
	}
	c.Do(http.MethodPost, p+"/push/sites", map[string]any{"address": "help.x22.test"}).MustStatus(t, 409)

	// The check: nothing at the address yet.
	c.Do(http.MethodPost, p+"/push/sites/"+site.ID+"/check", nil).MustStatus(t, 200).JSON(t, &site)
	if site.Worker.Status != push.WorkerMissing || site.Worker.HTTPStatus == nil || *site.Worker.HTTPStatus != 404 || site.Worker.CheckedAt == nil {
		t.Fatalf("missing: %+v", site.Worker)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM notifications WHERE workspace_id = $1 AND kind = 'push_worker_missing'`, f.ws.ID); n != 1 {
		t.Fatalf("worker-missing notifications: %d", n)
	}
	// The file uploaded: found, version 1, for the whole site.
	f.serveWorker()
	c.Do(http.MethodPost, p+"/push/sites/"+site.ID+"/check", nil).MustStatus(t, 200).JSON(t, &site)
	if site.Worker.Status != push.WorkerFound || site.Worker.Version == nil || *site.Worker.Version != push.WorkerVersion || site.Worker.Scope != "/" {
		t.Fatalf("found: %+v", site.Worker)
	}
	// The snippet the site installs, and the file to host.
	c.Do(http.MethodGet, p+"/push/setup", nil).MustStatus(t, 200).JSON(t, &setup)
	if len(setup.Sites) != 1 || !strings.Contains(setup.Snippet.Script, "/push/v1/prompt.js") || setup.Snippet.WorkerFile != push.WorkerFile ||
		setup.Snippet.WorkerVersion != push.WorkerVersion {
		t.Fatalf("setup: %+v", setup)
	}
	file := c.DoRaw(http.MethodGet, "/push/v1/"+push.WorkerFile, "", nil).MustStatus(t, 200)
	if !strings.Contains(file.Header.Get("Content-Disposition"), "attachment") || !bytes.Equal(file.Body, push.HostedWorker()) {
		t.Fatalf("download: %v", file.Header)
	}

	// Removing a site is logged; adding it again works.
	c.Do(http.MethodDelete, p+"/push/sites/"+site.ID, nil).MustStatus(t, 204)
	c.Do(http.MethodPost, p+"/push/sites", map[string]any{"address": "help.x22.test"}).MustStatus(t, 201)
	a := f.actions(t)
	for _, want := range []string{"push_keys.created", "push_site.added", "push_site.removed"} {
		if a[want] == 0 {
			t.Fatalf("audit: no %s in %v", want, a)
		}
	}
}

func TestFetchWorker(t *testing.T) {
	old := bytes.Replace(push.HostedWorker(), []byte("version 1"), []byte("version 0"), 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Service-Worker") != "script" {
			http.Error(w, "not a worker request", http.StatusBadRequest)
			return
		}
		switch r.URL.Path {
		case "/rc-push-sw.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write(push.HostedWorker())
		case "/sw.js":
			w.Header().Set("Content-Type", "text/javascript")
			w.Header().Set("Service-Worker-Allowed", "/")
			_, _ = w.Write([]byte("self.addEventListener('fetch', () => {});\nimportScripts('https://reputecast.acme.io/push/v1/sw.js');\n"))
		case "/app/sw.js":
			w.Header().Set("Content-Type", "text/javascript")
			_, _ = w.Write([]byte("self.addEventListener('fetch', () => {});\n"))
		case "/old/rc-push-sw.js":
			w.Header().Set("Content-Type", "text/javascript")
			_, _ = w.Write(old)
		case "/html/rc-push-sw.js":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><title>Not found</title>"))
		case "/moved/rc-push-sw.js":
			http.Redirect(w, r, "/rc-push-sw.js", http.StatusMovedPermanently)
		case "/busy/rc-push-sw.js":
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	ctx := context.Background()
	for _, tc := range []struct {
		path, mode, status, detail, scope string
		version, code                     int
	}{
		{"/rc-push-sw.js", "file", push.WorkerFound, "", "/", 1, 200},
		{"/sw.js", "import", push.WorkerFound, "", "/", 1, 200},
		{"/app/sw.js", "import", push.WorkerNotAWorker, "no_import", "/app/", 0, 200},
		{"/old/rc-push-sw.js", "file", push.WorkerOld, "", "/old/", 0, 200},
		{"/html/rc-push-sw.js", "file", push.WorkerNotAWorker, "content_type:text/html", "", 0, 200},
		{"/moved/rc-push-sw.js", "file", push.WorkerNotAWorker, "redirect", "", 0, 301},
		{"/busy/rc-push-sw.js", "file", push.WorkerUnreach, "", "", 0, 503},
		{"/gone/rc-push-sw.js", "file", push.WorkerMissing, "", "", 0, 404},
	} {
		res := push.FetchWorker(ctx, srv.Client(), srv.URL+tc.path, tc.mode)
		if res.Status != tc.status || res.Detail != tc.detail || res.Scope != tc.scope || res.HTTPStatus == nil || *res.HTTPStatus != tc.code {
			t.Errorf("%s: %+v (code %v)", tc.path, res, res.HTTPStatus)
			continue
		}
		if tc.version > 0 && (res.Version == nil || *res.Version != tc.version) {
			t.Errorf("%s: version %v", tc.path, res.Version)
		}
		if tc.status == push.WorkerOld && (res.Version == nil || *res.Version != 0) {
			t.Errorf("%s: old version %v", tc.path, res.Version)
		}
	}
	// Nothing listening.
	srv.Close()
	if res := push.FetchWorker(ctx, srv.Client(), srv.URL+"/rc-push-sw.js", "file"); res.Status != push.WorkerUnreach || res.HTTPStatus != nil {
		t.Fatalf("closed: %+v", res)
	}
}

func TestPushSubscriptionLifecycle(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path
	ctx := context.Background()
	slug := f.ws.Slug
	fake := env.App.PushFake()

	// Before push is on, no site is listed.
	ep, k, a := fake.Subscribe("chrome")
	sub := map[string]any{"endpoint": ep, "keys": map[string]string{"p256dh": k, "auth": a}}
	f.browser(siteURL, chromeUA).Do(http.MethodPost, "/p/"+slug+"/push/subscriptions", sub).MustStatus(t, 403)
	f.pushOn(t, 2)

	// Only the workspace's listed sites, and only push services.
	f.browser("https://evil.example", chromeUA).Do(http.MethodPost, "/p/"+slug+"/push/subscriptions", sub).MustStatus(t, 403)
	bad := map[string]any{"endpoint": "https://push.evil.example/x/1", "keys": map[string]string{"p256dh": k, "auth": a}}
	f.browser(siteURL, chromeUA).Do(http.MethodPost, "/p/"+slug+"/push/subscriptions", bad).MustStatus(t, 400)
	short := map[string]any{"endpoint": ep, "keys": map[string]string{"p256dh": k, "auth": "AAAA"}}
	f.browser(siteURL, chromeUA).Do(http.MethodPost, "/p/"+slug+"/push/subscriptions", short).MustStatus(t, 400)
	// The preflight answers listed sites only.
	pre := f.browser(siteURL, chromeUA).Do(http.MethodOptions, "/p/"+slug+"/push/subscriptions", nil).MustStatus(t, 204)
	if pre.Header.Get("Access-Control-Allow-Origin") != siteURL {
		t.Fatalf("preflight: %v", pre.Header)
	}
	f.browser("https://evil.example", chromeUA).Do(http.MethodOptions, "/p/"+slug+"/push/subscriptions", nil).MustStatus(t, 403)

	// Anonymous subscriptions: Chrome on Windows, Edge, Chrome on Android.
	chrome, s1 := f.subscribe(t, "chrome", chromeUA, nil)
	edge, sEdge := f.subscribe(t, "edge", edgeUA, nil)
	android, sAndroid := f.subscribe(t, "android", androidUA, nil)
	if s1.Linked {
		t.Fatal("an anonymous subscription is linked")
	}
	// The same browser again is the same subscription.
	var again subscribed
	f.browser(siteURL, chromeUA).Do(http.MethodPost, "/p/"+slug+"/push/subscriptions", map[string]any{"endpoint": chrome,
		"keys": map[string]string{"p256dh": f.keys[chrome][0], "auth": f.keys[chrome][1]}, "timeZone": "America/Chicago"}).MustStatus(t, 200).JSON(t, &again)
	if again.ID != s1.ID {
		t.Fatalf("refresh made a new subscription: %s, %s", again.ID, s1.ID)
	}
	var browser, platform, zone, page string
	if err := env.App.Pool.QueryRow(ctx, `SELECT browser, platform, time_zone, page FROM push_subscriptions WHERE id = $1`, s1.ID).
		Scan(&browser, &platform, &zone, &page); err != nil {
		t.Fatal(err)
	}
	if browser != push.BrowserChrome || platform != "Windows" || zone != "America/Chicago" || page != siteURL+"/docs/flags" {
		t.Fatalf("consent record: %s %s %s %s", browser, platform, zone, page)
	}

	// A reader who came from a personal link is linked to their contact.
	sam := f.contact(t, "sam@x22.test")
	tok := env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Preferences, WorkspaceID: f.ws.ID, ContactID: sam})
	firefox, s4 := f.subscribe(t, "firefox", firefoxUA, map[string]any{"token": tok})
	if !s4.Linked {
		t.Fatal("the subscription from a personal link isn't linked")
	}
	// Another workspace's token links nobody.
	other := env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Preferences, WorkspaceID: uuid.Must(uuid.NewV7()), ContactID: sam})
	if _, s5 := f.subscribe(t, "safari", "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15",
		map[string]any{"token": other}); s5.Linked {
		t.Fatal("another workspace's token linked a subscription")
	}
	// The contacts table's "Push subscribed" filter: Sam only.
	var contacts struct {
		Total int64 `json:"total"`
	}
	c.Do(http.MethodGet, p+"/contacts?pushSubscribed=true", nil).MustStatus(t, 200).JSON(t, &contacts)
	if contacts.Total != 1 {
		t.Fatalf("contacts with push: %d", contacts.Total)
	}
	// The segment builder's "Push subscribed": linked subscriptions only.
	for _, tc := range []struct {
		field, value string
		want         int64
	}{{"push_site", "any", 1}, {"push_browser", "firefox", 1}, {"push_browser", "chrome", 0}} {
		var cnt struct {
			Count int64 `json:"count"`
		}
		c.Do(http.MethodPost, p+"/segments/preview", map[string]any{"mode": "builder", "definition": map[string]any{"groups": []any{map[string]any{"match": "all",
			"conditions": []any{map[string]any{"field": tc.field, "op": "is", "value": tc.value}}}}}}).MustStatus(t, 200).JSON(t, &cnt)
		if cnt.Count != tc.want {
			t.Fatalf("segment %s is %s: %d, want %d", tc.field, tc.value, cnt.Count, tc.want)
		}
	}
	var card push.PushContactCard
	c.Do(http.MethodGet, p+"/contacts/"+sam.String()+"/push", nil).MustStatus(t, 200).JSON(t, &card)
	if !card.PushOn || len(card.Subscriptions) != 1 || card.Subscriptions[0].Browser != push.BrowserFirefox || card.Subscriptions[0].LinkedBy != "link" {
		t.Fatalf("contact card: %+v", card)
	}

	// Edge's subscription ended (410) and Android's isn't known (404):
	// both are removed when the push service says so.
	fake.Answer(edge, http.StatusGone)
	fake.Answer(android, http.StatusNotFound)
	fake.Reset()
	camp := f.send(t, "X22 flags post", "Feature flags at scale")
	got := fake.Received()
	if len(got) != 5 {
		t.Fatalf("the push service got %d requests", len(got))
	}
	var payload push.Payload
	for _, r := range got {
		if r.Endpoint == chrome {
			if err := json.Unmarshal(r.Payload, &payload); err != nil {
				t.Fatalf("payload %q: %v", r.Payload, err)
			}
			if r.Claims.Aud != "https://fcm.googleapis.com" || r.Claims.Sub != "mailto:owner@x22.test" || r.Header.Get("Urgency") != "normal" ||
				r.Header.Get("TTL") != "14400" {
				t.Fatalf("request: %+v %v", r.Claims, r.Header)
			}
		}
	}
	if payload.Title != "Feature flags at scale" || !strings.Contains(payload.URL, "/t/p/"+payload.ID+".") || payload.Token == "" ||
		!strings.HasSuffix(payload.Events, "/p/"+slug+"/push/events") {
		t.Fatalf("payload: %+v", payload)
	}
	for _, id := range []string{sEdge.ID, sAndroid.ID} {
		var reason, endpoint string
		var removed *time.Time
		if err := env.App.Pool.QueryRow(ctx, `SELECT removed_reason, removed_at, endpoint FROM push_subscriptions WHERE id = $1`, uuid.MustParse(id)).
			Scan(&reason, &removed, &endpoint); err != nil {
			t.Fatal(err)
		}
		if reason != "gone" || removed == nil || endpoint != "" {
			t.Fatalf("%s: %s %v %q", id, reason, removed, endpoint)
		}
	}
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'failed' AND reason = 'gone'`,
		uuid.MustParse(camp.ID)); n != 2 {
		t.Fatalf("failed as gone: %d", n)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM push_subscriptions WHERE workspace_id = $1 AND removed_at IS NULL`, f.ws.ID); n != 3 {
		t.Fatalf("live subscriptions: %d", n)
	}

	// The worker reports it shown; the reader clicks it.
	f.browser(siteURL, chromeUA).Do(http.MethodPost, "/p/"+slug+"/push/events", map[string]any{"kind": "notification", "event": "shown",
		"id": payload.ID, "t": payload.Token}).MustStatus(t, 204)
	// A wrong token counts nothing.
	f.browser(siteURL, chromeUA).Do(http.MethodPost, "/p/"+slug+"/push/events", map[string]any{"kind": "notification", "event": "shown",
		"id": payload.ID, "t": "nope"}).MustStatus(t, 204)
	click := strings.TrimPrefix(payload.URL, env.Server.URL)
	head := c.DoRaw(http.MethodHead, click, "", nil).MustStatus(t, 302)
	if n := one[int64](t, env, `SELECT count(*) FROM push_clicks WHERE workspace_id = $1`, f.ws.ID); n != 0 || head.Header.Get("Location") == "" {
		t.Fatalf("a HEAD request counted as a click: %d", n)
	}
	r := c.DoRaw(http.MethodGet, click, "", nil).MustStatus(t, 302)
	loc := r.Header.Get("Location")
	if !strings.HasPrefix(loc, siteURL+"/blog/flags?") || !strings.Contains(loc, "utm_source=push") || !strings.Contains(loc, "utm_medium=push") ||
		!strings.Contains(loc, "utm_campaign=x22-flags-post") {
		t.Fatalf("click goes to %q", loc)
	}
	c.DoRaw(http.MethodGet, "/t/p/"+payload.ID+".wrong", "", nil).MustStatus(t, 404)

	var rep push.PushReport
	c.Do(http.MethodGet, p+"/push-campaigns/"+camp.ID+"/report", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Status != "sent" || rep.Planned != 5 || rep.Sent != 3 || rep.Failed != 2 || rep.Expired != 2 || rep.Shown != 1 || rep.Clicks != 1 {
		t.Fatalf("report: %+v", rep)
	}
	if len(rep.ByBrowser) != 3 || len(rep.BySite) != 1 || rep.BySite[0].Host != "help.x22.test" || rep.BySite[0].Sent != 3 {
		t.Fatalf("report by browser %+v, by site %+v", rep.ByBrowser, rep.BySite)
	}

	// The frequency cap (2 a day): of two more campaigns today, the second
	// reaches nobody.
	fake.Reset()
	second := f.sendOnly(t, "X22 second post", "Second post")
	third := f.sendOnly(t, "X22 third post", "Third post")
	f.run(t)
	if n := len(fake.Received()); n != 3 {
		t.Fatalf("the second campaign went to %d", n)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'sent'`, uuid.MustParse(second.ID)); n != 3 {
		t.Fatalf("second campaign sent: %d", n)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'skipped' AND reason = 'frequency_cap'`,
		uuid.MustParse(third.ID)); n != 3 {
		t.Fatalf("skipped for the cap: %d", n)
	}
	// Review & send says so before a fourth: everyone had their two.
	var fourth push.PushCampaign
	c.Do(http.MethodPost, p+"/push-campaigns", map[string]any{"name": "X22 fourth", "title": "Fourth", "link": "help.x22.test"}).MustStatus(t, 201).JSON(t, &fourth)
	var rv push.PushReview
	c.Do(http.MethodGet, p+"/push-campaigns/"+fourth.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Verdict != "blocked" || rv.Audience.OverCap != 3 || rv.Audience.Final != 0 {
		t.Fatalf("review over the cap: %s %+v", rv.Verdict, rv.Audience)
	}
	c.Do(http.MethodPost, p+"/push-campaigns/"+fourth.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 409)

	// Sam's timeline: the subscription and the notification.
	var tl struct {
		Items []struct {
			Type   string `json:"type"`
			Answer string `json:"answer"`
		} `json:"items"`
	}
	c.Do(http.MethodGet, p+"/contacts/"+sam.String()+"/activity?limit=20", nil).MustStatus(t, 200).JSON(t, &tl)
	types := map[string]bool{}
	for _, it := range tl.Items {
		types[it.Type] = true
	}
	if !types["push_subscribed"] || !types["push_sent"] {
		t.Fatalf("timeline: %+v", tl.Items)
	}
	// The reader turns notifications off in the browser.
	f.browser(siteURL, firefoxUA).Do(http.MethodPost, "/p/"+slug+"/push/unsubscribe", map[string]any{"endpoint": firefox}).MustStatus(t, 200)
	c.Do(http.MethodGet, p+"/contacts/"+sam.String()+"/push", nil).MustStatus(t, 200).JSON(t, &card)
	if len(card.Subscriptions) != 0 {
		t.Fatalf("after unsubscribe: %+v", card)
	}
	acts := f.actions(t)
	for _, want := range []string{"campaign.create", "campaign.scheduled", "campaign.send_started", "campaign.completed", "push_settings.updated"} {
		if acts[want] == 0 {
			t.Fatalf("audit: no %s in %v", want, acts)
		}
	}
}

func TestPushHeldWhileWorkerMissing(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path
	fake := env.App.PushFake()
	site := f.pushOn(t, 5)
	f.subscribe(t, "chrome", chromeUA, nil)
	f.subscribe(t, "firefox", firefoxUA, nil)

	// The file is gone: notifications wait for it.
	fake.Serve(siteURL+"/"+push.WorkerFile, webpushtest.Page{Status: http.StatusNotFound})
	c.Do(http.MethodPost, p+"/push/sites/"+site.ID+"/check", nil).MustStatus(t, 200)
	var rv push.PushReview
	var camp push.PushCampaign
	c.Do(http.MethodPost, p+"/push-campaigns", map[string]any{"name": "X22 held", "title": "Held", "link": "help.x22.test/held"}).MustStatus(t, 201).JSON(t, &camp)
	c.Do(http.MethodGet, p+"/push-campaigns/"+camp.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Verdict != "blocked" {
		t.Fatalf("every site of the audience lacks its worker: %s", rv.Verdict)
	}
	// Two sites, one of them fine: the campaign can go, the missing site's
	// share waits.
	var other push.PushSite
	c.Do(http.MethodPost, p+"/push/sites", map[string]any{"address": "docs.x22.test"}).MustStatus(t, 201).JSON(t, &other)
	fake.Serve("https://docs.x22.test/"+push.WorkerFile, webpushtest.Page{Status: 200, Header: http.Header{"Content-Type": {"text/javascript"}},
		Body: push.HostedWorker()})
	c.Do(http.MethodPost, p+"/push/sites/"+other.ID+"/check", nil).MustStatus(t, 200)
	endpoint, k, a := fake.Subscribe("edge")
	f.browser("https://docs.x22.test", edgeUA).Do(http.MethodPost, "/p/"+f.ws.Slug+"/push/subscriptions", map[string]any{"endpoint": endpoint,
		"keys": map[string]string{"p256dh": k, "auth": a}}).MustStatus(t, 200)
	c.Do(http.MethodPatch, p+"/push-campaigns/"+camp.ID, map[string]any{"name": "X22 held", "title": "Held", "message": "", "link": "help.x22.test/held",
		"buttons": []any{}, "ttlMinutes": 240, "tag": "", "replaceTag": false, "siteIds": []string{site.ID, other.ID}, "lists": []string{}, "segments": []string{},
		"version": camp.Version}).MustStatus(t, 200)
	c.Do(http.MethodGet, p+"/push-campaigns/"+camp.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Verdict == "blocked" {
		t.Fatalf("review: %+v", rv.Checks)
	}
	fake.Reset()
	c.Do(http.MethodPost, p+"/push-campaigns/"+camp.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200)
	f.run(t)
	if n := len(fake.Received()); n != 1 {
		t.Fatalf("sent %d, want the docs site's 1", n)
	}
	id := uuid.MustParse(camp.ID)
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'held' AND reason = 'worker_missing'`, id); n != 2 {
		t.Fatalf("held: %d", n)
	}
	var rep push.PushReport
	c.Do(http.MethodGet, p+"/push-campaigns/"+camp.ID+"/report", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Status != "sending" || rep.Held != 2 || rep.Sent != 1 {
		t.Fatalf("report while held: %+v", rep)
	}
	// The file is back: what waited goes, and the campaign completes.
	f.serveWorker()
	c.Do(http.MethodPost, p+"/push/sites/"+site.ID+"/check", nil).MustStatus(t, 200)
	f.run(t)
	if n := len(fake.Received()); n != 3 {
		t.Fatalf("after the file came back: %d sent", n)
	}
	c.Do(http.MethodGet, p+"/push-campaigns/"+camp.ID+"/report", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Status != "sent" || rep.Held != 0 || rep.Sent != 3 {
		t.Fatalf("report after: %+v", rep)
	}
}

func TestPushRotationPromptsAndPermissions(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path
	ctx := context.Background()
	slug := f.ws.Slug
	site := f.pushOn(t, 2)

	// A prompt (a form type): designed, published, served to its site.
	var pr push.PushPrompt
	c.Do(http.MethodPost, p+"/push/prompts", map[string]any{"name": "Help center bell", "siteId": site.ID}).MustStatus(t, 201).JSON(t, &pr)
	if pr.Status != "draft" || pr.Draft.Style != push.StyleBar {
		t.Fatalf("new prompt: %+v", pr)
	}
	// A form type: listed with the forms, never a signup form.
	var forms struct {
		Forms []struct {
			ID       string `json:"id"`
			Kind     string `json:"kind"`
			PublicID string `json:"publicId"`
			Slug     string `json:"slug"`
		} `json:"forms"`
	}
	c.Do(http.MethodGet, p+"/forms", nil).MustStatus(t, 200).JSON(t, &forms)
	if len(forms.Forms) != 1 || forms.Forms[0].Kind != "push" || forms.Forms[0].ID != pr.FormID {
		t.Fatalf("forms list: %+v", forms.Forms)
	}
	c.Do(http.MethodGet, p+"/forms/"+pr.FormID, nil).MustStatus(t, 404)
	c.Do(http.MethodPut, p+"/public-pages/pages", map[string]any{"hostedFormId": pr.FormID}).MustStatus(t, 422)
	c.DoRaw(http.MethodGet, "/p/"+slug+"/forms/"+forms.Forms[0].PublicID+".js", "", nil).MustStatus(t, 404)
	cfg := pr.Draft
	cfg.Style, cfg.Title, cfg.Message = push.StyleBell, "Get release notes as notifications", "One a week at most."
	c.Do(http.MethodPut, p+"/push/prompts/"+pr.FormID, map[string]any{"config": cfg, "siteId": site.ID}).MustStatus(t, 200).JSON(t, &pr)
	long := cfg
	long.Title = strings.Repeat("x", push.PromptTitleMax+1)
	c.Do(http.MethodPut, p+"/push/prompts/"+pr.FormID, map[string]any{"config": long}).MustStatus(t, 422)
	c.Do(http.MethodPost, p+"/push/prompts/"+pr.FormID+"/publish", nil).MustStatus(t, 200).JSON(t, &pr)
	if pr.Status != "live" || pr.LiveVersion != 1 || pr.Draft.Corner != "bottom_right" {
		t.Fatalf("published: %+v", pr)
	}
	var conf push.PushPublicConfig
	f.browser(siteURL, chromeUA).Do(http.MethodGet, "/p/"+slug+"/push/config?site="+siteURL, nil).MustStatus(t, 200).JSON(t, &conf)
	if conf.Key == "" || conf.Site != "help.x22.test" || conf.Worker != "/"+push.WorkerFile || len(conf.Prompts) != 1 || conf.Prompts[0].Config.Title != cfg.Title ||
		conf.Text["iosShare"] == "" {
		t.Fatalf("config: %+v", conf)
	}
	f.browser("https://evil.example", chromeUA).Do(http.MethodGet, "/p/"+slug+"/push/config?site="+siteURL, nil).MustStatus(t, 403)
	for _, ev := range []string{"bell", "shown", "allow", "not_a_count"} {
		f.browser(siteURL, chromeUA).Do(http.MethodPost, "/p/"+slug+"/push/events", map[string]any{"kind": "prompt", "event": ev, "prompt": pr.FormID}).MustStatus(t, 204)
	}
	// The subscription keeps the exact prompt text and version.
	_, sub := f.subscribe(t, "chrome", chromeUA, map[string]any{"prompt": pr.FormID, "version": 1})
	var text string
	var version int32
	if err := env.App.Pool.QueryRow(ctx, `SELECT prompt_text, prompt_version FROM push_subscriptions WHERE id = $1`, sub.ID).Scan(&text, &version); err != nil {
		t.Fatal(err)
	}
	if text != "Get release notes as notifications · One a week at most." || version != 1 {
		t.Fatalf("consent record: %q v%d", text, version)
	}
	c.Do(http.MethodGet, p+"/push/prompts/"+pr.FormID, nil).MustStatus(t, 200).JSON(t, &pr)
	if pr.Stats.BellClicked != 1 || pr.Stats.Shown != 1 || pr.Stats.Allowed != 1 || pr.Stats.Granted != 1 {
		t.Fatalf("prompt numbers: %+v", pr.Stats)
	}
	c.Do(http.MethodPut, p+"/push/prompts/"+pr.FormID+"/status", map[string]any{"status": "paused"}).MustStatus(t, 200)
	f.browser(siteURL, chromeUA).Do(http.MethodGet, "/p/"+slug+"/push/config?site="+siteURL, nil).MustStatus(t, 200).JSON(t, &conf)
	if len(conf.Prompts) != 0 {
		t.Fatalf("a paused prompt is served: %+v", conf.Prompts)
	}

	// Permissions: a viewer sees Setup but changes nothing; an editor makes
	// and sends campaigns but can't rotate the keys.
	viewer := env.CreateUser(apptest.UserOpts{Email: "viewer@x22.test", Name: "Vic Viewer"})
	editor := env.CreateUser(apptest.UserOpts{Email: "editor@x22.test", Name: "Eve Editor"})
	env.AddMember(f.ws, viewer.ID, "viewer")
	env.AddMember(f.ws, editor.ID, "editor")
	vc, ec := env.Client(), env.Client()
	vc.SignInDemo(viewer.Email) // a session past the second factor
	ec.SignInDemo(editor.Email)
	var setup push.PushSetup
	vc.Do(http.MethodGet, p+"/push/setup", nil).MustStatus(t, 200).JSON(t, &setup)
	if setup.CanManage || len(setup.Helpers) == 0 {
		t.Fatalf("viewer's setup: %+v", setup)
	}
	vc.Do(http.MethodPost, p+"/push/sites", map[string]any{"address": "blog.x22.test"}).MustStatus(t, 403)
	vc.Do(http.MethodPost, p+"/push-campaigns", map[string]any{"name": "Nope"}).MustStatus(t, 403)
	ec.Do(http.MethodGet, p+"/push/keys/rotation", nil).MustStatus(t, 403)
	var camp push.PushCampaign
	ec.Do(http.MethodPost, p+"/push-campaigns", map[string]any{"name": "X22 editor", "title": "Hello", "link": "help.x22.test"}).MustStatus(t, 201).JSON(t, &camp)
	if !camp.Access.CanSend {
		t.Fatalf("editor access: %+v", camp.Access)
	}
	vc.Do(http.MethodPost, p+"/push-campaigns/"+camp.ID+"/send", map[string]any{"version": camp.Version}).MustStatus(t, 403)

	// Rotation: every subscription ends; the confirmation is their number.
	var rot push.PushRotation
	c.Do(http.MethodGet, p+"/push/keys/rotation", nil).MustStatus(t, 200).JSON(t, &rot)
	if rot.Subscriptions != 1 || rot.Confirm != "1" || len(rot.Sites) != 1 {
		t.Fatalf("rotation: %+v", rot)
	}
	c.Do(http.MethodPost, p+"/push/keys/rotate", map[string]any{"confirm": "2"}).MustStatus(t, 422)
	before := one[string](t, env, `SELECT public_key FROM push_keys WHERE workspace_id = $1 AND rotated_at IS NULL`, f.ws.ID)
	c.Do(http.MethodPost, p+"/push/keys/rotate", map[string]any{"confirm": "1"}).MustStatus(t, 200)
	after := one[string](t, env, `SELECT public_key FROM push_keys WHERE workspace_id = $1 AND rotated_at IS NULL`, f.ws.ID)
	if after == before {
		t.Fatal("the keys didn't change")
	}
	if n := one[int64](t, env, `SELECT count(*) FROM push_subscriptions WHERE workspace_id = $1 AND removed_at IS NULL`, f.ws.ID); n != 0 {
		t.Fatalf("subscriptions left after rotation: %d", n)
	}
	if r := one[string](t, env, `SELECT removed_reason FROM push_subscriptions WHERE id = $1`, uuid.MustParse(sub.ID)); r != "rotated" {
		t.Fatalf("removed for %q", r)
	}
	a := f.actions(t)
	for _, want := range []string{"form.create", "push_prompt.published", "push_prompt.paused", "push_keys.rotated"} {
		if a[want] == 0 {
			t.Fatalf("audit: no %s in %v", want, a)
		}
	}
}
