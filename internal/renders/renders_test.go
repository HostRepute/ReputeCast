package renders_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/renders"
)

// CO-13: real renders from the workspace's own Litmus or Mailgun Inspect
// account. The fakes answer the way the services' APIs do.

const ws = "/workspaces/acme-newsletter"

// fakeLitmus is the Litmus Instant API: clients, emails, previews.
type fakeLitmus struct {
	mu         sync.Mutex
	key        string
	html       []string
	subjects   []string
	prefetched []string
	asked      map[string]int
}

func (f *fakeLitmus) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	user, _, _ := r.BasicAuth()
	if user != f.key {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/v1/clients":
		_ = json.NewEncoder(w).Encode([]string{"OL2019", "GMAILNEW", "IPHONE13", "APPLEMAIL16", "ANDROID12", "OL2007", "THUNDERBIRDLATEST"})
	case r.Method == http.MethodPost && r.URL.Path == "/v1/emails":
		var in struct {
			HTML    string `json:"html_text"`
			Subject string `json:"subject"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.html = append(f.html, in.HTML)
		f.subjects = append(f.subjects, in.Subject)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"email_guid":"9f2c51e0-7a1b-4c3d-8e9f-0a1b2c3d4e5f"}`))
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/previews/prefetch"):
		var in struct {
			Configurations []struct{ Client string } `json:"configurations"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		for _, c := range in.Configurations {
			f.prefetched = append(f.prefetched, c.Client)
		}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/emails/9f2c51e0-7a1b-4c3d-8e9f-0a1b2c3d4e5f/previews/"):
		client := strings.TrimPrefix(r.URL.Path, "/v1/emails/9f2c51e0-7a1b-4c3d-8e9f-0a1b2c3d4e5f/previews/")
		f.asked[client]++
		switch {
		case client == "OL2007":
			w.WriteHeader(http.StatusBadRequest) // not captured for this account
		case f.asked[client] == 1:
			w.WriteHeader(http.StatusAccepted) // still capturing
		case client == "IPHONE13":
			_, _ = w.Write([]byte(`{"full_url":"javascript:alert(1)","thumb_url":"javascript:alert(2)"}`))
		default:
			base := "https://" + client + ".capture.litmuscdn.com/9f2c51e0/results/"
			_, _ = w.Write([]byte(`{"full_url":"` + base + `vertical-allowed-1366.png","thumb_url":"` + base + `vertical-allowed-1366-thumb.png","thumb450_url":"` + base + `vertical-allowed-1366-thumb450.png"}`))
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeLitmus) sent() (html, subject string, prefetched []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.html) == 0 {
		return "", "", nil
	}
	return f.html[len(f.html)-1], f.subjects[len(f.subjects)-1], append([]string{}, f.prefetched...)
}

func (f *fakeLitmus) revoke() {
	f.mu.Lock()
	f.key = "revoked"
	f.mu.Unlock()
}

type render struct {
	ID, Status, Variant, Error, Service, RequestedBy string
	Version                                          int
	Results                                          []struct{ Client, Name, Status, ImageURL, ThumbURL string }
}

func draft(t *testing.T, env *apptest.Env) string {
	t.Helper()
	var id string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT c.id::text FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.status = 'draft' AND c.deleted_at IS NULL AND c.html <> '' ORDER BY c.created_at LIMIT 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestLitmusRenders(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	f := &fakeLitmus{key: "lit_live_7c1e9a4b52d3", asked: map[string]int{}}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	renders.UseEndpoints(t, map[string]string{"litmus": srv.URL})

	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	sofia := env.Client() // edits campaigns, doesn't manage integrations
	sofia.SignInDemo("sofia@acme.io")
	aisha := env.Client() // views
	aisha.SignInDemo("aisha@acme.io")

	var rs struct {
		Connected                           bool
		Service, KeyHint, Status, LastError string
		Clients                             []struct {
			ID, Name string
			Default  bool
		}
	}
	aisha.Do("GET", ws+"/rendering-service", nil).MustStatus(t, 200).JSON(t, &rs)
	if rs.Connected {
		t.Fatal("nothing connected yet")
	}

	// A wrong key is refused and nothing is kept.
	p := linh.Do("PUT", ws+"/rendering-service", map[string]any{"service": "litmus", "apiKey": "lit_live_wrong000"}).MustStatus(t, 422).Problem(t)
	if !strings.Contains(p["detail"].(string), "Litmus refused the API key") {
		t.Fatalf("refused: %v", p)
	}
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	p = vi.Do("PUT", ws+"/rendering-service", map[string]any{"service": "litmus", "apiKey": "lit_live_wrong000"}).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Litmus từ chối khóa API. Hãy kiểm tra khóa trong tài khoản Litmus của bạn rồi dán lại." {
		t.Fatalf("refused, in Vietnamese: %v", p)
	}
	sofia.Do("PUT", ws+"/rendering-service", map[string]any{"service": "litmus", "apiKey": f.key}).MustStatus(t, 403)

	r := linh.Do("PUT", ws+"/rendering-service", map[string]any{"service": "litmus", "apiKey": f.key})
	r.MustStatus(t, 200).JSON(t, &rs)
	if !rs.Connected || rs.Service != "litmus" || rs.KeyHint != "52d3" || rs.Status != "connected" || len(rs.Clients) != 7 {
		t.Fatalf("connected: %+v", rs)
	}
	if bytes.Contains(r.Body, []byte(f.key)) {
		t.Fatal("the key must never be returned")
	}
	names, defaults := map[string]string{}, []string{}
	for _, c := range rs.Clients {
		names[c.ID] = c.Name
		if c.Default {
			defaults = append(defaults, c.ID)
		}
	}
	if names["OL2019"] != "Outlook 2019" || names["GMAILNEW"] != "Gmail" || names["IPHONE13"] != "iPhone 13" || names["APPLEMAIL16"] != "Apple Mail 16" {
		t.Fatalf("names: %v", names)
	}
	if strings.Join(defaults, ",") != "OL2019,GMAILNEW,IPHONE13,APPLEMAIL16,ANDROID12" {
		t.Fatalf("defaults: %v", defaults)
	}
	var sealed []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT credentials FROM rendering_services`).Scan(&sealed); err != nil || bytes.Contains(sealed, []byte(f.key)) {
		t.Fatalf("the key must be sealed at rest: %v", err)
	}

	// A render: the saved campaign, personalized for the sample contact.
	cid := draft(t, env)
	renders := ws + "/campaigns/" + cid + "/renders"
	aisha.Do("POST", renders, map[string]any{"clients": []string{"OL2019"}}).MustStatus(t, 403)
	linh.Do("POST", renders, map[string]any{"clients": []string{"NOTACLIENT"}}).MustStatus(t, 422)
	var started render
	sofia.Do("POST", renders, map[string]any{"clients": []string{"OL2019", "GMAILNEW", "IPHONE13", "OL2007"}}).MustStatus(t, 201).JSON(t, &started)
	if started.Status != "pending" || started.Variant != "A" || started.Version < 1 || len(started.Results) != 4 || started.Results[0].Name != "Outlook 2019" || started.RequestedBy == "" {
		t.Fatalf("started: %+v", started)
	}
	html, subject, prefetched := f.sent()
	if html == "" || subject == "" || strings.Join(prefetched, ",") != "OL2019,GMAILNEW,IPHONE13,OL2007" {
		t.Fatalf("sent: %q %q %v", subject, html[:min(len(html), 80)], prefetched)
	}
	if strings.Contains(html, "{{") {
		t.Fatal("Liquid must be filled in for the sample contact")
	}
	sofia.Do("POST", renders, map[string]any{"clients": []string{"OL2019"}}).MustStatus(t, 409)

	var list struct{ Items []render }
	aisha.Do("GET", renders, nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Status != "pending" {
		t.Fatalf("first look: %+v", list.Items)
	}
	// Asking again right away doesn't call the service again.
	asked := f.asked["OL2019"]
	aisha.Do("GET", renders, nil).MustStatus(t, 200)
	if f.asked["OL2019"] != asked {
		t.Fatal("polled too soon")
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaign_renders SET polled_at = polled_at - interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	aisha.Do("GET", renders, nil).MustStatus(t, 200).JSON(t, &list)
	got := list.Items[0]
	if got.Status != "done" {
		t.Fatalf("second look: %+v", got)
	}
	by := map[string]struct{ Status, Image, Thumb string }{}
	for _, x := range got.Results {
		by[x.Client] = struct{ Status, Image, Thumb string }{x.Status, x.ImageURL, x.ThumbURL}
	}
	if by["OL2019"].Status != "done" || !strings.HasPrefix(by["OL2019"].Image, "https://OL2019.capture.litmuscdn.com/") || !strings.HasSuffix(by["OL2019"].Thumb, "thumb450.png") {
		t.Fatalf("OL2019: %+v", by["OL2019"])
	}
	if by["OL2007"].Status != "failed" {
		t.Fatalf("a client the service won't capture fails: %+v", by["OL2007"])
	}
	if by["IPHONE13"].Status != "failed" || by["IPHONE13"].Image != "" {
		t.Fatalf("only https screenshots are shown: %+v", by["IPHONE13"])
	}

	// The key stops working: a test says so.
	f.revoke()
	linh.Do("POST", ws+"/rendering-service/test", nil).MustStatus(t, 200).JSON(t, &rs)
	if rs.Status != "failing" || !strings.Contains(rs.LastError, "Litmus refused the API key") || len(rs.Clients) != 7 {
		t.Fatalf("test: %+v", rs)
	}
	sofia.Do("DELETE", ws+"/rendering-service", nil).MustStatus(t, 403)
	linh.Do("DELETE", ws+"/rendering-service", nil).MustStatus(t, 204)
	linh.Do("DELETE", ws+"/rendering-service", nil).MustStatus(t, 409)
	aisha.Do("GET", ws+"/rendering-service", nil).MustStatus(t, 200).JSON(t, &rs)
	if rs.Connected {
		t.Fatal("disconnected")
	}
	aisha.Do("GET", renders, nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Status != "done" {
		t.Fatal("renders already made stay")
	}
	sofia.Do("POST", renders, map[string]any{"clients": []string{"OL2019"}}).MustStatus(t, 409)

	for action, want := range map[string]int{"rendering.connect": 1, "campaign.render": 1, "rendering.test": 1, "rendering.disconnect": 1} {
		var n int
		var leaked bool
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*), coalesce(bool_or(after::text LIKE '%'||$2||'%' OR before::text LIKE '%'||$2||'%'), false) FROM audit_events WHERE action = $1`, action, "lit_live_7c1e9a4b").Scan(&n, &leaked); err != nil || n != want || leaked {
			t.Fatalf("audit %s: %d (want %d), key leaked %v, %v", action, n, want, leaked, err)
		}
	}
}

// fakeMailgun is Mailgun Inspect's email previews.
type fakeMailgun struct {
	mu      sync.Mutex
	created []map[string]any
}

func (f *fakeMailgun) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if user, pass, _ := r.BasicAuth(); user != "api" || pass != "key-3f9a1c2e8b7d" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/v1/preview/clients":
		_, _ = w.Write([]byte(`{"clients":{"outlook19":{"id":"outlook19","client":"Outlook 2019","os":"Windows","default":true},
			"gmail_chr":{"id":"gmail_chr","client":"Gmail","os":"Chrome"},"iphone15_ios17":{"id":"iphone15_ios17","client":"iPhone 15","os":"iOS 17"}}}`))
	case r.Method == http.MethodPost && r.URL.Path == "/v2/preview/tests":
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.created = append(f.created, in)
		_, _ = w.Write([]byte(`{"id":"t_8KQ2M4XA"}`))
	case r.Method == http.MethodGet && r.URL.Path == "/v1/preview/tests/t_8KQ2M4XA/results":
		_, _ = w.Write([]byte(`{"outlook19":{"status":"Complete","screenshots":{"default":"https://screenshots.mailgun.net/t_8KQ2M4XA/outlook19.png"},"thumbnail":"https://screenshots.mailgun.net/t_8KQ2M4XA/outlook19-thumb.png"},
			"gmail_chr":{"status":"Processing"}}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestMailgunInspectRenders(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	f := &fakeMailgun{}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	// The EU account's region picks the EU host.
	renders.UseEndpoints(t, map[string]string{"mailgun_inspect/eu": srv.URL, "mailgun_inspect/us": "http://127.0.0.1:1"})
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")

	var rs struct {
		Connected       bool
		Service, Region string
		Clients         []struct{ ID, Name string }
	}
	linh.Do("PUT", ws+"/rendering-service", map[string]any{"service": "mailgun_inspect", "apiKey": "key-3f9a1c2e8b7d", "region": "eu"}).MustStatus(t, 200).JSON(t, &rs)
	if rs.Service != "mailgun_inspect" || rs.Region != "eu" || len(rs.Clients) != 3 || rs.Clients[0].Name != "Gmail · Chrome" {
		t.Fatalf("connected: %+v", rs)
	}
	// The US host can't be reached: a 502 that says so, nothing kept.
	p := linh.Do("PUT", ws+"/rendering-service", map[string]any{"service": "mailgun_inspect", "apiKey": "key-3f9a1c2e8b7d", "region": "us"}).MustStatus(t, 502).Problem(t)
	if !strings.Contains(p["detail"].(string), "Mailgun Inspect couldn't be reached") {
		t.Fatalf("unreachable: %v", p)
	}

	cid := draft(t, env)
	var started render
	linh.Do("POST", ws+"/campaigns/"+cid+"/renders", map[string]any{"clients": []string{"outlook19", "gmail_chr"}}).MustStatus(t, 201).JSON(t, &started)
	if len(f.created) != 1 || f.created[0]["subject"] == "" || len(f.created[0]["clients"].([]any)) != 2 {
		t.Fatalf("created: %v", f.created)
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaign_renders SET requested_at = requested_at - interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	var list struct{ Items []render }
	linh.Do("GET", ws+"/campaigns/"+cid+"/renders", nil).MustStatus(t, 200).JSON(t, &list)
	got := list.Items[0]
	if got.Status != "pending" || got.Results[0].Status != "done" || got.Results[0].ImageURL != "https://screenshots.mailgun.net/t_8KQ2M4XA/outlook19.png" || got.Results[1].Status != "pending" {
		t.Fatalf("results: %+v", got)
	}
	// A render the service never finishes fails after 15 minutes.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaign_renders SET requested_at = requested_at - interval '20 minutes', polled_at = NULL`); err != nil {
		t.Fatal(err)
	}
	linh.Do("GET", ws+"/campaigns/"+cid+"/renders", nil).MustStatus(t, 200).JSON(t, &list)
	if got := list.Items[0]; got.Status != "failed" || !strings.Contains(got.Error, "didn't finish within 15 minutes") || got.Results[0].Status != "done" {
		t.Fatalf("gave up: %+v", got)
	}
}

func TestLitmusNames(t *testing.T) {
	for code, want := range map[string]string{"OL2019": "Outlook 2019", "GMAILNEW": "Gmail", "IPHONE15PRO": "iPhone 15 Pro", "APPLEMAIL16": "Apple Mail 16", "XYZ9": "XYZ9", "OFFICE365": "Outlook (Microsoft 365)"} {
		if got := renders.LitmusName(code); got != want {
			t.Errorf("%s: %q, want %q", code, got, want)
		}
	}
}
