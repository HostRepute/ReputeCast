package forms_test

// M60 G-23–G-25 end to end on the demo: steps and choice questions saved
// and refused, targeting answered by the session (schedule, countries
// through a GeoIP stand-in), the step beacon, and A/B tests from the
// demo's running one to assignment, counting, ending and auto-ending.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/forms"
)

// geoStandIn stands in for a GeoIP database (no .mmdb file in tests): every
// address is in country.
type geoStandIn struct{ country string }

func (g *geoStandIn) Lookup(netip.Addr) (string, string) { return g.country, "" }
func (g *geoStandIn) Loaded() bool                       { return true }

const browser = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15"

type pubResp struct {
	Status int
	Header http.Header
	Body   string
}

// pub sends a request to a public path as a browser on origin would.
func pub(t *testing.T, env *apptest.Env, method, path, body, contentType, origin string) pubResp {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, env.Server.URL+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", browser)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	return pubResp{Status: res.StatusCode, Header: res.Header, Body: string(b)}
}

func pubJSON(t *testing.T, r pubResp) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(r.Body), &m); err != nil {
		t.Fatalf("not JSON (%d): %s", r.Status, r.Body)
	}
	return m
}

func viewsOf(t *testing.T, env *apptest.Env, form string) int64 {
	t.Helper()
	var n int64
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT coalesce(sum(views), 0) FROM form_daily WHERE form_id = $1`, form).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func token(env *apptest.Env, publicID string) string {
	return (&forms.Service{Box: env.App.Secrets}).FormToken(publicID, time.Now().Add(-time.Minute))
}

type abJSON struct {
	Status, StartedOn, Outcome, Leader, RunnerUp, Winner string
	Lift, Confidence                                     float64
	FewestViews                                          int64
	MinViews                                             int
	Variants                                             []struct {
		Key                           string
		Weight                        int
		Views, Submissions, Confirmed int64
		Rate                          float64
		Content                       map[string]any
	}
}

func getAB(t *testing.T, c *apptest.Client, form string) abJSON {
	t.Helper()
	var ab abJSON
	c.Do("GET", api+"/forms/"+form+"/ab", nil).MustStatus(t, 200).JSON(t, &ab)
	return ab
}

// The demo's Footer mini form test is the FormBuilderABTest board: B leads
// A 4.1% to 3.2% (+28%) with 97% confidence.
func TestFormABTestDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")
	footer := formByName(t, overview(t, c), "Footer mini form")
	ab := getAB(t, c, footer)
	if ab.Status != "running" || len(ab.Variants) != 2 || ab.Outcome != "winner" || ab.Leader != "B" || ab.RunnerUp != "A" || ab.Lift != 28 || ab.Confidence != 97 || ab.MinViews != 500 {
		t.Fatalf("result: %+v", ab)
	}
	a, b := ab.Variants[0], ab.Variants[1]
	if a.Views != 4096 || a.Submissions != 131 || a.Confirmed != 118 || a.Rate != 3.2 || b.Views != 4104 || b.Submissions != 168 || b.Confirmed != 152 || b.Rate != 4.1 {
		t.Fatalf("variants: %+v", ab.Variants)
	}
	if b.Content["heading"] != "Join 48,000 readers of the Weekly Digest" || a.Content != nil || b.Weight != 50 {
		t.Fatalf("B's content: %+v", b)
	}
	want := time.Now().In(mustLoc(t, "America/New_York")).AddDate(0, 0, -26).Format(time.DateOnly)
	if ab.StartedOn != want {
		t.Errorf("started on %s, want %s", ab.StartedOn, want)
	}
	f := getForm(t, c, footer)
	if f.Stats.Views != 9412 || f.Stats.Submissions != 338 || f.Stats.Confirmed != 304 || f.Stats.Blocked != 512 {
		t.Fatalf("30-day numbers: %+v", f.Stats)
	}
	// The script carries B's copy for visitors who see the form in English.
	js := pub(t, env, "GET", "/p/acme-newsletter/forms/frm_3f91d0.js", "", "", "")
	if js.Status != 200 || !strings.Contains(js.Body, `"ab":{"B":{"heading":"Join 48,000 readers of the Weekly Digest"`) || !strings.Contains(js.Body, `"wait":true`) {
		t.Fatalf("script: %d %s", js.Status, js.Body[:min(len(js.Body), 300)])
	}

	// A Viewer reads the test and can't change it.
	viewer := signedIn(env, "aisha@acme.io")
	viewer.Do("GET", api+"/forms/"+footer+"/ab", nil).MustStatus(t, 200)
	viewer.Do("POST", api+"/forms/"+footer+"/ab/end", map[string]any{"keep": "B"}).MustStatus(t, 403)

	// Ending on its own: once a variant wins, the leader is kept and the
	// people who edit forms are told.
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE forms SET ab = ab || '{"autoEnd": true}' WHERE id = $1`, footer); err != nil {
		t.Fatal(err)
	}
	before := audits(t, env, "form.ab_ended")
	if err := env.App.Forms().AutoEndTests(ctx); err != nil {
		t.Fatal(err)
	}
	ab = getAB(t, c, footer)
	f = getForm(t, c, footer)
	if ab.Status != "ended" || ab.Winner != "B" || f.Content["heading"] != "Join 48,000 readers of the Weekly Digest" || f.Content["button"] != "Get the Digest" {
		t.Fatalf("auto-ended: %+v %v", ab, f.Content)
	}
	if audits(t, env, "form.ab_ended") != before+1 {
		t.Error("form.ab_ended audit entry missing")
	}
	var notes int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'form_test_ended'`).Scan(&notes); err != nil || notes == 0 {
		t.Errorf("form editors weren't told: %d %v", notes, err)
	}
	var left int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM form_variants WHERE form_id = $1`, footer).Scan(&left)
	if left != 0 {
		t.Errorf("%d variants left", left)
	}
}

func TestFormStepsAndChoices(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")
	signup := formByName(t, overview(t, c), "Newsletter signup")
	f := getForm(t, c, signup)
	in := inputOf(f)
	fields := in["fields"].([]map[string]any)
	fields = append(fields, map[string]any{"key": "topics", "label": "What are you into?", "type": "choice", "multiple": true, "required": true,
		"options": []map[string]any{{"label": "Product news", "tag": "product"}, {"label": "Engineering deep dives", "tag": "Engineering"}}})
	in["fields"] = fields
	content := in["content"].(map[string]any)

	// Every field in exactly one step, the email in one.
	content["steps"] = []map[string]any{{"id": "s1", "fields": []string{"email", "topics"}}, {"id": "s2", "fields": []string{"topics", "first_name"}}}
	p := c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 422).Problem(t)
	if !strings.Contains(string(mustJSON(t, p)), "Put every field in one step.") {
		t.Fatalf("field in two steps: %v", p)
	}
	content["steps"] = []map[string]any{{"id": "s1", "fields": []string{"topics"}}, {"id": "s2", "fields": []string{"first_name"}}}
	p = c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 422).Problem(t)
	if !strings.Contains(string(mustJSON(t, p)), "Keep the email field in a step.") {
		t.Fatalf("no email: %v", p)
	}
	bad := map[string]any{"key": "company", "label": "Company?", "type": "choice", "options": []map[string]any{{"label": "A", "tag": "a"}, {"label": "B", "tag": "b"}}}
	in["fields"] = append(fields, bad)
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 422)
	in["fields"] = fields

	content["steps"] = []map[string]any{
		{"id": "s1", "fields": []string{"email"}, "heading": "Get the Weekly Digest", "button": "Next"},
		{"id": "s2", "fields": []string{"topics"}, "heading": "What are you into?", "description": "Pick as many as you like."},
		{"id": "s3", "fields": []string{"first_name"}, "heading": "Almost done"},
	}
	var saved struct {
		Fields []struct {
			Key, Type string
			Options   []struct{ Label, Tag string }
		}
		Content struct {
			Steps []struct {
				ID      string
				Fields  []string
				Heading string
			}
		}
		PlainHTML string
		URLs      struct{ ScriptKB int }
	}
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 200).JSON(t, &saved)
	if len(saved.Content.Steps) != 3 || saved.Content.Steps[1].Heading != "What are you into?" || saved.Fields[len(saved.Fields)-1].Options[1].Tag != "engineering" {
		t.Fatalf("saved: %+v", saved)
	}
	// The plain HTML copy shows every step at once, each a fieldset (three
	// steps, the choice question and the lists).
	if strings.Count(saved.PlainHTML, "<legend>") != 5 || !strings.Contains(saved.PlainHTML, `name="c.topics" value="1"`) {
		t.Errorf("plain HTML: %s", saved.PlainHTML)
	}
	// Steps and the choice question stay within the script's budget.
	if saved.URLs.ScriptKB < 2 || saved.URLs.ScriptKB > 12 {
		t.Errorf("script %d KB", saved.URLs.ScriptKB)
	}
	// The hosted page shows the steps as fieldsets with their headings as legends.
	page := pub(t, env, "GET", "/p/acme-newsletter/subscribe", "", "", "")
	if page.Status != 200 || !strings.Contains(page.Body, `<fieldset class="step"><legend>Get the Weekly Digest</legend>`) || !strings.Contains(page.Body, `<p class="step-d">Pick as many as you like.</p>`) ||
		!strings.Contains(page.Body, `type="checkbox" name="c.topics" value="0"`) {
		t.Fatalf("hosted page: %d %s", page.Status, page.Body)
	}

	// A required choice question needs an answer; an answer adds its tag.
	body := url.Values{"_t": {token(env, "frm_8c2e1a")}, "email": {"quinn@fastmail.com"}, "list": {f.DefaultListIDs[0]}, "consent": {"1"}, "_js": {"1"}}
	r := pub(t, env, "POST", "/p/acme-newsletter/forms/frm_8c2e1a/submit", body.Encode(), "application/x-www-form-urlencoded", "https://acme.io")
	if errs, _ := pubJSON(t, r)["errors"].(map[string]any); r.Status != 422 || errs["topics"] == nil {
		t.Fatalf("no answer: %d %s", r.Status, r.Body)
	}
	body["c.topics"] = []string{"1", "0"}
	r = pub(t, env, "POST", "/p/acme-newsletter/forms/frm_8c2e1a/submit", body.Encode(), "application/x-www-form-urlencoded", "https://acme.io")
	if r.Status != 200 || pubJSON(t, r)["status"] != "confirm" {
		t.Fatalf("submit: %d %s", r.Status, r.Body)
	}
	var tags []string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT tags FROM contacts WHERE email = 'quinn@fastmail.com'`).Scan(&tags); err != nil {
		t.Fatal(err)
	}
	if strings.Join(tags, ",") != "engineering,product" {
		t.Errorf("tags %v", tags)
	}

	// The step beacon counts a later step's view, from the form's sites only.
	beacon := url.Values{"n": {"2"}, "t": {token(env, "frm_8c2e1a")}}.Encode()
	if r := pub(t, env, "POST", "/p/acme-newsletter/forms/frm_8c2e1a/step", beacon, "text/plain;charset=UTF-8", "https://acme.io"); r.Status != 204 {
		t.Fatalf("beacon: %d %s", r.Status, r.Body)
	}
	if r := pub(t, env, "POST", "/p/acme-newsletter/forms/frm_8c2e1a/step", beacon, "text/plain;charset=UTF-8", "https://evil.example"); r.Status != 403 {
		t.Errorf("beacon from another site: %d", r.Status)
	}
	pub(t, env, "POST", "/p/acme-newsletter/forms/frm_8c2e1a/step", url.Values{"n": {"3"}, "t": {"forged"}}.Encode(), "text/plain", "https://acme.io")
	var stepViews map[int]int64
	rows, err := env.App.Pool.Query(context.Background(), `SELECT step, sum(views) FROM form_step_daily WHERE form_id = $1 GROUP BY step`, signup)
	if err != nil {
		t.Fatal(err)
	}
	stepViews = map[int]int64{}
	for rows.Next() {
		var s int
		var n int64
		_ = rows.Scan(&s, &n)
		stepViews[s] = n
	}
	rows.Close()
	if stepViews[2] != 1 || stepViews[3] != 0 {
		t.Errorf("step views %v: one for step 2, none for a forged token", stepViews)
	}
	got := getForm(t, c, signup)
	var funnel struct {
		Stats struct {
			Steps []struct {
				Step  int
				Views int64
			}
			StepSignups int64
		}
	}
	c.Do("GET", api+"/forms/"+signup, nil).MustStatus(t, 200).JSON(t, &funnel)
	if len(funnel.Stats.Steps) != 3 || funnel.Stats.Steps[0].Views != got.Stats.Views || funnel.Stats.Steps[1].Views != 1 || funnel.Stats.StepSignups != got.Stats.Submissions {
		t.Errorf("funnel %+v (views %d, submissions %d)", funnel.Stats, got.Stats.Views, got.Stats.Submissions)
	}
}

func TestFormTargetingSession(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	geo := &geoStandIn{country: "DE"}
	env.App.Forms().GeoIP = geo
	c := signedIn(env, "linh@acme.io")
	signup := formByName(t, overview(t, c), "Newsletter signup")
	f := getForm(t, c, signup)
	in := inputOf(f)
	content := in["content"].(map[string]any)
	content["targeting"] = map[string]any{
		"pages": map[string]any{"mode": "only", "rules": []map[string]any{{"op": "starts_with", "value": "/blog"}}}, "sources": map[string]any{"utmSource": []string{}, "utmMedium": []string{}, "utmCampaign": []string{}, "referrer": "any"},
		"devices": []string{"mobile"}, "countries": map[string]any{"mode": "except", "codes": []string{"de"}}, "visitors": "all", "schedule": map[string]any{},
	}
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 200)
	var o struct{ GeoIP bool }
	c.Do("GET", api+"/forms", nil).MustStatus(t, 200).JSON(t, &o)
	if !o.GeoIP {
		t.Error("the overview doesn't say country rules work")
	}
	session := "/p/acme-newsletter/forms/frm_8c2e1a/session"
	views := viewsOf(t, env, signup)
	r := pub(t, env, "GET", session, "", "", "https://acme.io")
	if m := pubJSON(t, r); r.Status != 200 || m["show"] != false || m["t"] != nil {
		t.Fatalf("an excluded country: %d %s", r.Status, r.Body)
	}
	if viewsOf(t, env, signup) != views {
		t.Error("a form not shown counted a view")
	}
	geo.country = "US"
	r = pub(t, env, "GET", session, "", "", "https://acme.io")
	if m := pubJSON(t, r); m["show"] != true || m["t"] == "" {
		t.Fatalf("another country: %s", r.Body)
	}
	if viewsOf(t, env, signup) != views+1 {
		t.Error("the view wasn't counted")
	}
	// A peek (a popup before it opens) counts nothing.
	pub(t, env, "GET", session+"?peek=1", "", "", "https://acme.io")
	if viewsOf(t, env, signup) != views+1 {
		t.Error("a peek counted a view")
	}
	// The script carries the rules it checks itself; the session the others.
	js := pub(t, env, "GET", "/p/acme-newsletter/forms/frm_8c2e1a.js", "", "", "")
	if !strings.Contains(js.Body, `"target":{"pages":{"mode":"only","rules":[{"op":"starts_with","value":"/blog"}]}`) || !strings.Contains(js.Body, `"devices":["mobile"]`) || strings.Contains(js.Body, `"codes"`) {
		t.Errorf("script targeting: %s", js.Body[:min(len(js.Body), 400)])
	}

	// A schedule that has ended closes the form, its hosted page included.
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	content["targeting"] = map[string]any{"pages": map[string]any{"mode": "all", "rules": []any{}}, "sources": map[string]any{"utmSource": []string{}, "utmMedium": []string{}, "utmCampaign": []string{}, "referrer": "any"},
		"devices": []string{}, "countries": map[string]any{"mode": "all", "codes": []string{}}, "visitors": "all", "schedule": map[string]any{"endsAt": past}}
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 200)
	if m := pubJSON(t, pub(t, env, "GET", session, "", "", "https://acme.io")); m["show"] != false {
		t.Errorf("an ended schedule shows the form: %v", m)
	}
	if r := pub(t, env, "GET", "/p/acme-newsletter/subscribe", "", "", ""); r.Status != 410 || !strings.Contains(r.Body, "This form is closed") {
		t.Errorf("hosted page after the schedule: %d", r.Status)
	}
	body := url.Values{"_t": {token(env, "frm_8c2e1a")}, "email": {"late@fastmail.com"}, "list": {f.DefaultListIDs[0]}, "consent": {"1"}, "_js": {"1"}}
	if r := pub(t, env, "POST", "/p/acme-newsletter/forms/frm_8c2e1a/submit", body.Encode(), "application/x-www-form-urlencoded", "https://acme.io"); r.Status != 410 {
		t.Errorf("a submission after the schedule: %d %s", r.Status, r.Body)
	}
}

func TestFormABTestFlow(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")
	o := overview(t, c)
	signup := formByName(t, o, "Newsletter signup")
	ctx := context.Background()

	// Hosted forms can't be tested; a Viewer can't start a test.
	b := map[string]any{"heading": "Join 48,000 builders", "description": "One email a week.", "button": "Count me in", "popupDelay": 8, "popupScroll": 0}
	start := map[string]any{"variants": []map[string]any{{"key": "A", "weight": 50}, {"key": "B", "weight": 50, "content": b}}, "autoEnd": false, "minViews": 500, "confidence": 0.95}
	c.Do("PUT", api+"/forms/"+formByName(t, o, "Beta apply")+"/ab", start).MustStatus(t, 409)
	signedIn(env, "aisha@acme.io").Do("PUT", api+"/forms/"+signup+"/ab", start).MustStatus(t, 403)
	c.Do("PUT", api+"/forms/"+signup+"/ab", map[string]any{"variants": []map[string]any{{"key": "A", "weight": 60}, {"key": "B", "weight": 50, "content": b}}, "autoEnd": false, "minViews": 500, "confidence": 0.95}).MustStatus(t, 422)

	before := audits(t, env, "form.ab_updated")
	var ab abJSON
	c.Do("PUT", api+"/forms/"+signup+"/ab", start).MustStatus(t, 200).JSON(t, &ab)
	today := time.Now().In(mustLoc(t, "America/New_York")).Format(time.DateOnly)
	if ab.Status != "running" || ab.StartedOn != today || len(ab.Variants) != 2 || ab.Outcome != "too_small" || ab.Variants[0].Views != 0 {
		t.Fatalf("started: %+v", ab)
	}
	if audits(t, env, "form.ab_updated") != before+1 {
		t.Error("form.ab_updated audit entry missing")
	}

	// The same browser always gets the same variant; it counts a view there.
	session := "/p/acme-newsletter/forms/frm_8c2e1a/session?ab=1&vid="
	vid := "Qk3x9Yw2Lm8Np4Rs7Tv0Ua"
	first := pubJSON(t, pub(t, env, "GET", session+vid, "", "", "https://acme.io"))
	variant, _ := first["variant"].(string)
	if variant != "A" && variant != "B" {
		t.Fatalf("no variant: %v", first)
	}
	for range 3 {
		if again := pubJSON(t, pub(t, env, "GET", session+vid, "", "", "https://acme.io")); again["variant"] != variant {
			t.Fatalf("variant changed: %v then %v", variant, again["variant"])
		}
	}
	// Without ab=1 (another language) the visitor isn't in the test.
	if m := pubJSON(t, pub(t, env, "GET", "/p/acme-newsletter/forms/frm_8c2e1a/session?vid="+vid, "", "", "https://acme.io")); m["variant"] != "" {
		t.Errorf("outside the test: %v", m)
	}
	// A signup counts for the variant it saw.
	signupForm := getForm(t, c, signup)
	body := url.Values{"_t": {token(env, "frm_8c2e1a")}, "email": {"variant.reader@fastmail.com"}, "list": {signupForm.DefaultListIDs[0]}, "consent": {"1"}, "_js": {"1"}, "_v": {variant}}
	if r := pub(t, env, "POST", "/p/acme-newsletter/forms/frm_8c2e1a/submit", body.Encode(), "application/x-www-form-urlencoded", "https://acme.io"); r.Status != 200 {
		t.Fatalf("submit: %d %s", r.Status, r.Body)
	}
	ab = getAB(t, c, signup)
	for _, v := range ab.Variants {
		wantViews, wantSubs := int64(0), int64(0)
		if v.Key == variant {
			wantViews, wantSubs = 4, 1
		}
		if v.Views != wantViews || v.Submissions != wantSubs {
			t.Errorf("variant %s: %+v", v.Key, v)
		}
	}
	var stored string
	_ = env.App.Pool.QueryRow(ctx, `SELECT variant FROM form_submissions WHERE form_id = $1 ORDER BY created_at DESC LIMIT 1`, signup).Scan(&stored)
	if stored != variant {
		t.Errorf("submission's variant %q", stored)
	}
	// A forged variant counts outside the test.
	body["email"], body["_v"] = []string{"forged.variant@fastmail.com"}, []string{"C"}
	pub(t, env, "POST", "/p/acme-newsletter/forms/frm_8c2e1a/submit", body.Encode(), "application/x-www-form-urlencoded", "https://acme.io")
	_ = env.App.Pool.QueryRow(ctx, `SELECT variant FROM form_submissions WHERE form_id = $1 ORDER BY created_at DESC LIMIT 1`, signup).Scan(&stored)
	if stored != "" {
		t.Errorf("forged variant stored as %q", stored)
	}

	// Ending with B makes B's copy the form's and deletes the variants.
	ended := audits(t, env, "form.ab_ended")
	c.Do("POST", api+"/forms/"+signup+"/ab/end", map[string]any{"keep": "C"}).MustStatus(t, 422)
	c.Do("POST", api+"/forms/"+signup+"/ab/end", map[string]any{"keep": "B"}).MustStatus(t, 200).JSON(t, &ab)
	f := getForm(t, c, signup)
	if ab.Status != "ended" || ab.Winner != "B" || f.Content["heading"] != "Join 48,000 builders" || f.Content["button"] != "Count me in" || f.Content["footnote"] == "" {
		t.Fatalf("ended: %+v %v", ab, f.Content)
	}
	if audits(t, env, "form.ab_ended") != ended+1 {
		t.Error("form.ab_ended audit entry missing")
	}
	var meta string
	_ = env.App.Pool.QueryRow(ctx, `SELECT metadata::text FROM audit_events WHERE action = 'form.ab_ended' ORDER BY occurred_at DESC LIMIT 1`).Scan(&meta)
	if !strings.Contains(meta, `"submissions": 1`) && !strings.Contains(meta, `"submissions":1`) {
		t.Errorf("the audit entry doesn't keep the numbers: %s", meta)
	}
	c.Do("POST", api+"/forms/"+signup+"/ab/end", map[string]any{"keep": "B"}).MustStatus(t, 409)
	if js := pub(t, env, "GET", "/p/acme-newsletter/forms/frm_8c2e1a.js", "", "", ""); strings.Contains(js.Body, `"ab":`) {
		t.Error("the script still carries variants")
	}
}
