package integrations_test

// The parts of the hub's API that the apps in integrations/ (Zapier, Make,
// n8n and the WordPress plugin) rely on beyond the hub itself: n8n's
// webhook check, the Form submitted trigger's form filter, the workflow
// header as the call log shows it, and the signed "Check now".

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/integrations"
)

func TestTriggerHookCheckAndForms(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	// The demo's n8n connection is replaced by one with Form submitted on.
	c.Do("POST", ws+"/integrations/connections/"+connectionID(t, c, "n8n")+"/disconnect", map[string]any{}).MustStatus(t, 204)
	n8n := connect(t, c, "n8n", map[string]any{"items": []string{"form.submitted", "contact.unsubscribed"}})
	srv := newHookServer(t)
	// n8n sends the workflow's name percent-encoded (RFC 8187), with its id.
	wf := map[string]string{"X-ReputeCast-Workflow": "name*=UTF-8''Beta%20signups%20%E2%86%92%20CRM; id=wf_2Lm8", "User-Agent": "ReputeCast-n8n/1.1.0"}
	var hook struct{ ID, Event, URL, Secret string }
	call(t, env, n8n.Secret, "POST", ws+"/integration/hooks", map[string]any{"event": "form.submitted", "url": srv.URL + "/webhook/7c1e/reputecast"}, wf).
		MustStatus(t, 201).JSON(t, &hook)

	// Checking the webhook answers it without its secret.
	var got map[string]any
	call(t, env, n8n.Secret, "GET", ws+"/integration/hooks/"+hook.ID, nil, wf).MustStatus(t, 200).JSON(t, &got)
	if got["id"] != hook.ID || got["event"] != "form.submitted" || got["url"] != srv.URL+"/webhook/7c1e/reputecast" || got["secret"] != nil {
		t.Errorf("the webhook as n8n checks it: %+v", got)
	}
	if got["signatureHeader"] != "X-ReputeCast-Signature" || got["createdAt"] == nil {
		t.Errorf("headers and time: %+v", got)
	}
	// Another integration's key doesn't see it.
	made := connect(t, c, "wordpress", map[string]any{"items": []string{"list_forms"}, "address": "blog.example.org"})
	if r := call(t, env, made.Secret, "GET", ws+"/integration/hooks/"+hook.ID, nil, nil); r.Status != 403 || code(t, r) != "integration_operation_denied" {
		t.Errorf("a site's key checks n8n's webhook: %d %s", r.Status, r.Body)
	}
	publicURL(env)
	other := connect(t, c, "make", map[string]any{"items": []string{"contact.unsubscribed"}})
	if r := call(t, env, other.Secret, "GET", ws+"/integration/hooks/"+hook.ID, nil, nil); r.Status != 404 {
		t.Errorf("another connection's key: %d %s", r.Status, r.Body)
	}

	// The forms of the Form submitted trigger's filter.
	var forms struct {
		Items []struct{ ID, Name, Status string }
	}
	call(t, env, n8n.Secret, "GET", ws+"/integration/forms", nil, wf).MustStatus(t, 200).JSON(t, &forms)
	if len(forms.Items) == 0 || forms.Items[0].ID != "frm_8c2e1a" || forms.Items[0].Name == "" || forms.Items[0].Status == "" {
		t.Errorf("forms: %+v", forms.Items)
	}
	// Only keys with that trigger on list them.
	if r := call(t, env, other.Secret, "GET", ws+"/integration/forms", nil, nil); r.Status != 403 || code(t, r) != "integration_operation_denied" {
		t.Errorf("forms without the Form submitted trigger: %d %s", r.Status, r.Body)
	}

	// The call log shows the header decoded, and the check under its trigger.
	ctx := context.Background()
	var headers []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT request->'headers' FROM integration_calls WHERE operation = 'subscribe-integration-hook' AND integration_id = $1`,
		n8n.IntegrationID).Scan(&headers); err != nil {
		t.Fatal(err)
	}
	var h map[string]string
	_ = json.Unmarshal(headers, &h)
	if h["X-ReputeCast-Workflow"] != "Beta signups → CRM; id=wf_2Lm8" {
		t.Errorf("the workflow header in the call log: %q", h["X-ReputeCast-Workflow"])
	}
	var item, target, client string
	if err := env.App.Pool.QueryRow(ctx, `SELECT item, target->>'kind', client FROM integration_calls WHERE operation = 'get-integration-hook' AND integration_id = $1`,
		n8n.IntegrationID).Scan(&item, &target, &client); err != nil {
		t.Fatal(err)
	}
	if item != "form.submitted" || target != "hook" || client != "n8n, ReputeCast node 1.1.0" {
		t.Errorf("the check in the call log: %q %q %q", item, target, client)
	}
	var name string
	_ = env.App.Pool.QueryRow(ctx, `SELECT name FROM integration_workflows WHERE integration_id = $1 AND key = 'wf_2Lm8'`, n8n.IntegrationID).Scan(&name)
	if name != "Beta signups → CRM" {
		t.Errorf("the workflow's name: %q", name)
	}

	// Once the workflow is turned off, the check answers 404 and n8n subscribes again.
	call(t, env, n8n.Secret, "DELETE", ws+"/integration/hooks/"+hook.ID, nil, wf).MustStatus(t, 204)
	if r := call(t, env, n8n.Secret, "GET", ws+"/integration/hooks/"+hook.ID, nil, wf); r.Status != 404 {
		t.Errorf("a removed webhook: %d %s", r.Status, r.Body)
	}
}

func TestWorkflowHeaderAsRead(t *testing.T) {
	for raw, want := range map[string]string{
		"Beta application (Typeform) → ReputeCast contact; id=284113907":         "Beta application (Typeform) → ReputeCast contact; id=284113907",
		"Stripe payment failed %E2%86%92 payment-failed email; id=284113907":     "Stripe payment failed → payment-failed email; id=284113907",
		"name*=UTF-8''Unsubscribes%20%E2%86%92%20Acme%20Cloud%20CRM; id=wf_2Lm8": "Unsubscribes → Acme Cloud CRM; id=wf_2Lm8",
		"name=\"Weekly %3B report\"; id=4":                                       "Weekly ; report; id=4",
		"id=284113907":                                                           "id=284113907",
		"HubSpot deal won → tag customer":                                        "HubSpot deal won → tag customer",
		"name*=UTF-8''100%25%20done; id=7":                                       "100% done; id=7",
		"name*=UTF-8''Caf%C3%A9%20%3D%20open; id=scenario-12":                    "Café = open; id=scenario-12",
	} {
		if got := integrations.ParseWorkflow(raw).String(); got != want {
			t.Errorf("%q reads %q, want %q", raw, got, want)
		}
	}
}

// statusSite is a WordPress site's status endpoint that answers only calls
// signed with the key it holds, as the plugin does.
type statusSite struct {
	*httptest.Server
	key   atomic.Value
	calls atomic.Int32
}

func newStatusSite(t *testing.T) *statusSite {
	s := &statusSite{}
	s.key.Store("")
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls.Add(1)
		if r.URL.Path != "/wp-json/reputecast/v1/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ts, err := strconv.ParseInt(r.Header.Get("X-ReputeCast-Timestamp"), 10, 64)
		if err != nil || time.Since(time.Unix(ts, 0)).Abs() > 5*time.Minute {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		sum := sha256.Sum256([]byte(s.key.Load().(string)))
		m := hmac.New(sha256.New, sum[:])
		m.Write([]byte(strconv.FormatInt(ts, 10) + ".status"))
		want := hex.EncodeToString(m.Sum(nil))
		for _, sig := range strings.Split(r.Header.Get("X-ReputeCast-Signature"), ",") {
			if hmac.Equal([]byte(strings.TrimSpace(sig)), []byte(want)) {
				_, _ = w.Write([]byte(`{"plugin":"1.3.0","wordpress":"6.9.2","php":"8.3"}`))
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestCheckNowIsSignedWithTheSiteKey(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	site := newStatusSite(t)
	conn := connect(t, c, "wordpress", map[string]any{"items": []string{"list_forms", "create_drafts"}, "address": site.URL})
	var check struct {
		OK      bool
		Outcome string
		Status  int
	}
	// A plugin connected with another key refuses the call.
	site.key.Store("rc_live_" + strings.Repeat("0", 64))
	c.Do("POST", ws+"/integrations/connections/"+conn.IntegrationID+"/check", nil).MustStatus(t, 200).JSON(t, &check)
	if check.OK || check.Outcome != "status" || check.Status != 401 {
		t.Fatalf("a site holding another key: %+v", check)
	}
	site.key.Store(conn.Secret)
	c.Do("POST", ws+"/integrations/connections/"+conn.IntegrationID+"/check", nil).MustStatus(t, 200).JSON(t, &check)
	if !check.OK || check.Outcome != "answered" {
		t.Fatalf("the site's own key: %+v", check)
	}
	// After a rotation the call is signed with both keys, whichever the plugin holds.
	var rotated struct{ Secret string }
	c.Do("POST", ws+"/integrations/connections/"+conn.IntegrationID+"/rotate", nil).MustStatus(t, 200).JSON(t, &rotated)
	c.Do("POST", ws+"/integrations/connections/"+conn.IntegrationID+"/check", nil).MustStatus(t, 200).JSON(t, &check)
	if !check.OK {
		t.Errorf("the old key during its 24 hours: %+v", check)
	}
	site.key.Store(rotated.Secret)
	c.Do("POST", ws+"/integrations/connections/"+conn.IntegrationID+"/check", nil).MustStatus(t, 200).JSON(t, &check)
	if !check.OK {
		t.Errorf("the new key: %+v", check)
	}
	if site.calls.Load() != 4 {
		t.Errorf("Check now calls the site once each time: %d", site.calls.Load())
	}
}

func TestStatusSignature(t *testing.T) {
	key := "rc_live_" + strings.Repeat("ab", 32)
	sum := sha256.Sum256([]byte(key))
	// The plugin's tests (integrations/wordpress) verify the same vector.
	const want = "45f9841e69974b8168b517da522ad7c914b7464255f7dfeb4612923d5d6cb199"
	if got := integrations.StatusSignature(sum[:], 1792138920); got != want {
		t.Errorf("signature: %q, want %q", got, want)
	}
}

// Plugin 1.3.0: a post's WPML or Polylang translations become its draft's
// language versions, marked up to date; one published later joins the
// draft.
func TestPostTranslationsBecomeLanguageVersions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	// The demo's workspace writes in English, Spanish and German; not French.
	var langs struct{ Languages []struct{ Code string } }
	c.Do("GET", ws+"/languages", nil).MustStatus(t, 200).JSON(t, &langs)
	offered := map[string]bool{}
	for _, l := range langs.Languages {
		offered[l.Code] = true
	}
	if !offered["es"] || !offered["de"] || offered["fr"] {
		t.Fatalf("the demo's languages: %+v", langs.Languages)
	}
	site := connect(t, c, "wordpress", map[string]any{"items": []string{"list_forms", "create_drafts"}, "address": "news.example.org"})
	ids := lists(t, env)
	var tpl string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT id FROM templates WHERE name = 'Blog post'`).Scan(&tpl)
	c.Do("PUT", ws+"/integrations/connections/"+site.IntegrationID+"/rule", map[string]any{"enabled": true, "categories": []string{"news"},
		"listIds": []string{ids["Product Updates"]}, "templateId": tpl}).MustStatus(t, 200)

	es := map[string]any{"id": "91", "language": "es", "url": "https://news.example.org/es/despliegues/", "title": "Los despliegues programados ya están aquí",
		"excerpt": "Elige la hora y nosotros hacemos el resto."}
	fr := map[string]any{"id": "92", "language": "fr-FR", "title": "Les déploiements programmés sont là"}
	post := map[string]any{"id": "90", "url": "https://news.example.org/rollouts/", "title": "Scheduled rollouts are here", "excerpt": "Pick the time; we do the rest.",
		"categories": []string{"news"}, "language": "en-US", "translations": []any{es, fr}}
	type result struct {
		Created    bool
		Reason     string
		CampaignID string
		Languages  []string
	}
	send := func(p map[string]any) result {
		var out result
		call(t, env, site.Secret, "POST", ws+"/wordpress/drafts", map[string]any{"post": p}, nil).MustStatus(t, 200).JSON(t, &out)
		return out
	}
	draft := send(post)
	// French isn't one of the workspace's languages: it's left out.
	if !draft.Created || strings.Join(draft.Languages, ",") != "es" {
		t.Fatalf("draft: %+v", draft)
	}
	versions := func(id string) map[string]string {
		var out struct {
			Source   string
			Versions []struct{ Language, Status string }
		}
		c.Do("GET", ws+"/campaigns/"+id+"/languages", nil).MustStatus(t, 200).JSON(t, &out)
		m := map[string]string{"source": out.Source}
		for _, v := range out.Versions {
			m[v.Language] = v.Status
		}
		return m
	}
	if v := versions(draft.CampaignID); v["source"] != "en" || v["es"] != "up_to_date" || v["de"] != "missing" || v["fr"] != "" {
		t.Errorf("versions: %v", v)
	}
	var spanish struct{ Subject, Preheader, MJML string }
	c.Do("GET", ws+"/campaigns/"+draft.CampaignID+"?language=es", nil).MustStatus(t, 200).JSON(t, &spanish)
	if spanish.Subject != "Los despliegues programados ya están aquí" || spanish.Preheader != "Elige la hora y nosotros hacemos el resto." ||
		!strings.Contains(spanish.MJML, "https://news.example.org/es/despliegues/") || strings.Contains(spanish.MJML, "Scheduled rollouts") {
		t.Errorf("the Spanish version: %+v", spanish)
	}
	// Without a featured image, the template's post image is left out.
	if strings.Contains(spanish.MJML, "<mj-image") {
		t.Errorf("an empty post image: %s", spanish.MJML)
	}

	// The German translation, published later, joins the draft.
	de := map[string]any{"id": "93", "language": "de", "title": "Geplante Rollouts sind da"}
	post["translations"] = []any{es, fr, de}
	if joined := send(post); joined.Created || joined.Reason != "exists" || joined.CampaignID != draft.CampaignID || strings.Join(joined.Languages, ",") != "de" {
		t.Errorf("a translation published later: %+v", joined)
	}
	// A translation's own publish finds the draft through the post.
	german := send(map[string]any{"id": "93", "title": "Geplante Rollouts sind da", "language": "de",
		"translations": []any{map[string]any{"id": "90", "language": "en", "title": "Scheduled rollouts are here"}}})
	if german.Created || german.Reason != "exists" || german.CampaignID != draft.CampaignID || len(german.Languages) != 0 {
		t.Errorf("the German post itself: %+v", german)
	}
	if v := versions(draft.CampaignID); v["en"] != "source" || v["es"] != "up_to_date" || v["de"] != "up_to_date" || v["fr"] != "" {
		t.Errorf("versions: %v", v)
	}
	var drafts int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM wordpress_drafts WHERE integration_id = $1`, site.IntegrationID).Scan(&drafts)
	if drafts != 1 {
		t.Errorf("one draft for the post and its translations: %d", drafts)
	}

	// A post in a language the workspace offers is written in it.
	neu := send(map[string]any{"id": "95", "title": "Neue Funktionen", "categories": []string{"news"}, "language": "de-DE"})
	if v := versions(neu.CampaignID); !neu.Created || v["source"] != "de" || len(neu.Languages) != 0 {
		t.Errorf("a German post: %+v %v", neu, v)
	}
}
