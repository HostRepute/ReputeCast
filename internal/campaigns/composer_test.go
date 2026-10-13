package campaigns_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type detail struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Subject       string `json:"subject"`
	Preheader     string `json:"preheader"`
	MJML          string `json:"mjml"`
	HTML          string `json:"html"`
	Version       int    `json:"version"`
	VersionCount  int    `json:"versionCount"`
	AudienceLabel string `json:"audienceLabel"`
	AudienceCount *int64 `json:"audienceCount"`
	AutosavedAt   *string
	Sender        *struct{ Name, Email string }
	Audience      struct{ Lists []string }
	Route         struct {
		Source string
		Shares []struct {
			Short   string
			Percent float64
		}
	}
	ABTest *struct {
		Kind     string
		Variants []struct{ Key, Subject string }
	}
	Access struct {
		Editable            bool
		CanEdit             bool
		CanSend             bool
		SendHelpers         []string
		CanManageRouting    bool
		CanSendTestToOthers bool
	}
}

func campaignID(t *testing.T, env *apptest.Env, name string) string {
	t.Helper()
	var id string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id::text FROM campaigns WHERE name = $1`, name).Scan(&id); err != nil {
		t.Fatalf("campaign %s: %v", name, err)
	}
	return id
}

func auditCount(t *testing.T, env *apptest.Env, action, target string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND target_id = $2`, action, target).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

const ws = "/workspaces/acme-newsletter"

func TestComposerDraftLifecycle(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	// A new draft: blank, version 1, never saved.
	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{}).MustStatus(t, 201).JSON(t, &d)
	if d.Name != "Untitled campaign" || d.Status != "draft" || d.Version != 1 || d.VersionCount != 1 || d.AutosavedAt != nil || d.MJML != "" {
		t.Fatalf("new draft: %+v", d)
	}
	if d.Sender == nil || d.Sender.Email != "digest@news.acme.io" {
		t.Fatalf("default sender: %+v", d.Sender)
	}
	if d.Route.Source != "rule" || len(d.Route.Shares) != 2 || d.Route.Shares[0].Short != "SES" || d.Route.Shares[0].Percent != 90 {
		t.Fatalf("broadcast route: %+v", d.Route)
	}
	if auditCount(t, env, "campaign.create", d.ID) != 1 {
		t.Fatal("create not audited")
	}

	mjmlSrc := `<mjml><mj-body><mj-section><mj-column><mj-text>Hi {{ first_name | default: "there" }}</mj-text></mj-column></mj-section></mj-body></mjml>`
	var saved struct {
		Version     int
		ContentHash string
		SizeBytes   int
		PlainText   string
	}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Hello", "preheader": "Pre", "mjml": mjmlSrc, "label": "first text"}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Version != 2 || saved.SizeBytes == 0 || saved.ContentHash == "" || !strings.Contains(saved.PlainText, "Hi {{ first_name") {
		t.Fatalf("save: %+v", saved)
	}

	// Someone saves on top of an old version: conflict, then "Keep mine".
	r := c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Mine", "mjml": mjmlSrc})
	r.MustStatus(t, 409)
	p := r.Problem(t)
	meta, _ := p["meta"].(map[string]any)
	if p["code"] != "version_conflict" || meta["savedBy"] != "Linh Nguyen" || meta["version"].(float64) != 2 {
		t.Fatalf("conflict problem: %v", p)
	}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Mine", "mjml": mjmlSrc, "force": true}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Version != 3 {
		t.Fatalf("forced save: %+v", saved)
	}
	// Subjects are one line.
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 3, "subject": "Two\nlines"}).MustStatus(t, 422)

	var versions struct {
		Items []struct {
			Version int
			Label   string
			Source  string
			SavedBy string
			Current bool
		}
		Total int
	}
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &versions)
	if versions.Total != 3 || versions.Items[0].Version != 3 || !versions.Items[0].Current || versions.Items[1].Label != "first text" || versions.Items[2].Source != "create" {
		t.Fatalf("versions: %+v", versions)
	}
	var v2 struct {
		Version int
		Subject string
		MJML    string
	}
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions/2", nil).MustStatus(t, 200).JSON(t, &v2)
	if v2.Subject != "Hello" || !strings.Contains(v2.MJML, "first_name") {
		t.Fatalf("version 2: %+v", v2)
	}
	c.Do("POST", ws+"/campaigns/"+d.ID+"/versions/2/restore", map[string]any{"baseVersion": 3}).MustStatus(t, 200).JSON(t, &saved)
	c.Do("GET", ws+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if saved.Version != 4 || d.Subject != "Hello" || d.Version != 4 {
		t.Fatalf("restore: %+v %+v", saved, d)
	}
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &versions)
	if versions.Items[0].Label != "restored v2" || versions.Items[0].Source != "restore" {
		t.Fatalf("restore label: %+v", versions.Items[0])
	}
	if n := auditCount(t, env, "campaign.content.save", d.ID); n != 3 {
		t.Fatalf("content saves audited: %d", n)
	}

	// Rename and pick an audience: the count is computed by the server.
	weekly := ""
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id::text FROM lists WHERE name = 'Weekly Digest'`).Scan(&weekly); err != nil {
		t.Fatal(err)
	}
	c.Do("PATCH", ws+"/campaigns/"+d.ID, map[string]any{"name": "Launch notes", "audience": map[string]any{"lists": []string{weekly}}}).MustStatus(t, 200).JSON(t, &d)
	if d.Name != "Launch notes" || d.AudienceLabel != "Weekly Digest" || d.AudienceCount == nil {
		t.Fatalf("update: %+v", d)
	}
	c.Do("PATCH", ws+"/campaigns/"+d.ID, map[string]any{"audience": map[string]any{"lists": []string{"01970000-0000-7000-8000-000000000000"}}}).MustStatus(t, 422)
	if auditCount(t, env, "campaign.update", d.ID) != 1 {
		t.Fatal("update not audited")
	}

	// Duplicate a sent campaign into a draft.
	oct := campaignID(t, env, "October Product Update")
	var dup detail
	c.Do("POST", ws+"/campaigns/"+oct+"/duplicate", nil).MustStatus(t, 201).JSON(t, &dup)
	if dup.Name != "Copy of October Product Update" || dup.Status != "draft" || !strings.Contains(dup.MJML, "October at Acme") || dup.Version != 1 {
		t.Fatalf("duplicate: %+v", dup)
	}
	// Sent campaigns are read-only.
	var octD detail
	c.Do("GET", ws+"/campaigns/"+oct, nil).MustStatus(t, 200).JSON(t, &octD)
	if octD.Access.Editable {
		t.Fatal("a sent campaign must not be editable")
	}
	c.Do("PUT", ws+"/campaigns/"+oct+"/content", map[string]any{"baseVersion": octD.Version, "subject": "x"}).MustStatus(t, 409)

	// From a starter template: images point at the starter assets.
	var fromTpl detail
	c.Do("POST", ws+"/campaigns", map[string]any{"templateId": "starter:weekly-digest"}).MustStatus(t, 201).JSON(t, &fromTpl)
	if !strings.Contains(fromTpl.MJML, "/media/starters/hero.png") || !strings.Contains(fromTpl.MJML, "500 Market St") || fromTpl.HTML == "" {
		t.Fatalf("from starter: %s", fromTpl.MJML[:200])
	}
}

func TestComposerDetailOfTheDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var d detail
	c.Do("GET", ws+"/campaigns/"+campaignID(t, env, "Weekly Digest #142"), nil).MustStatus(t, 200).JSON(t, &d)
	if d.Version != 14 || d.VersionCount != 14 || d.AudienceLabel != "Weekly Digest" || d.AudienceCount == nil || *d.AudienceCount != 31604 {
		t.Fatalf("digest: %+v", d)
	}
	if d.Subject != "Three launches, one deprecation, and the roadmap for Q4" || !strings.Contains(d.MJML, "mj-section") || d.Sender.Email != "digest@news.acme.io" {
		t.Fatalf("digest content: %+v", d)
	}
	if d.ABTest == nil || d.ABTest.Kind != "subject" || len(d.ABTest.Variants) != 2 || d.ABTest.Variants[1].Subject == "" {
		t.Fatalf("A/B: %+v", d.ABTest)
	}
	if d.Route.Source != "campaign" || len(d.Route.Shares) != 2 {
		t.Fatalf("route: %+v", d.Route)
	}
	if !d.Access.CanSend || !d.Access.CanManageRouting {
		t.Fatalf("owner access: %+v", d.Access)
	}
}

func TestComposerPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	digest := campaignID(t, env, "Weekly Digest #142")

	// Viewer: reads, can't change.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	var d detail
	v.Do("GET", ws+"/campaigns/"+digest, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Access.CanEdit {
		t.Fatal("viewer can't edit")
	}
	r := v.Do("PUT", ws+"/campaigns/"+digest+"/content", map[string]any{"baseVersion": 14, "subject": "x"})
	r.MustStatus(t, 403)
	if r.Problem(t)["permission"] != "campaigns:edit" {
		t.Fatalf("viewer problem: %s", r.Body)
	}
	v.Do("POST", ws+"/campaigns", map[string]any{}).MustStatus(t, 403)
	v.Do("POST", ws+"/templates", map[string]any{"name": "x", "mjml": "<mjml><mj-body></mj-body></mjml>"}).MustStatus(t, 403)

	// Content Writer: edits, can't send; review becomes a request for approval.
	w := env.Client()
	w.SignInDemo("daniel@acme.io")
	w.Do("GET", ws+"/campaigns/"+digest, nil).MustStatus(t, 200).JSON(t, &d)
	if !d.Access.CanEdit || d.Access.CanSend || d.Access.CanManageRouting || d.Access.CanSendTestToOthers {
		t.Fatalf("content writer access: %+v", d.Access)
	}
	if strings.Join(d.Access.SendHelpers, ",") != "Marcus Chen,Linh Nguyen" {
		t.Fatalf("helpers: %v", d.Access.SendHelpers)
	}
	w.Do("PUT", ws+"/campaigns/"+digest+"/content", map[string]any{"baseVersion": 14, "subject": d.Subject, "preheader": d.Preheader, "mjml": d.MJML, "html": d.HTML, "label": "typo"}).MustStatus(t, 200)
	// Test emails only to himself.
	r = w.Do("POST", ws+"/campaigns/"+digest+"/test", map[string]any{"to": []string{"marcus@acme.io"}})
	r.MustStatus(t, 403)
	if r.Problem(t)["permission"] != "campaigns:send" {
		t.Fatalf("test to others: %s", r.Body)
	}
	// To himself it is allowed, and fails only because no provider can send.
	r = w.Do("POST", ws+"/campaigns/"+digest+"/test", map[string]any{"to": []string{"daniel@acme.io"}})
	r.MustStatus(t, 409)
	if r.Problem(t)["code"] != "provider_unavailable" {
		t.Fatalf("no provider: %s", r.Body)
	}

	// Other workspaces' campaigns don't exist here.
	o := env.Client()
	o.SignInDemo("linh@acme.io")
	o.Do("GET", "/workspaces/acme-engineering-blog/campaigns/"+digest, nil).MustStatus(t, 404)
	o.Do("GET", "/workspaces/acme-engineering-blog/campaigns/"+digest+"/versions/1", nil).MustStatus(t, 404)
}

func TestComposerChecksAndPreview(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	flash := campaignID(t, env, "Flash sale: 48 hours only")
	var d detail
	c.Do("GET", ws+"/campaigns/"+flash, nil).MustStatus(t, 200).JSON(t, &d)
	var checks struct {
		Verdict           string
		Blocks, Warnings  int
		SizeBytes         int
		ImagesMissingAlt  int
		LinksBlocklisted  int
		SubjectHeuristics []string
		FooterRemovedAt   *string
		Issues            []struct {
			ID, Severity, Group, Title, Tag, Detail string
			Path                                    []int
			Fixes                                   []struct{ Kind, Label, Value string }
		}
	}
	c.Do("POST", ws+"/campaigns/"+flash+"/checks", map[string]any{"subject": d.Subject, "preheader": d.Preheader, "mjml": d.MJML, "html": d.HTML}).MustStatus(t, 200).JSON(t, &checks)
	if checks.Verdict != "blocked" || checks.ImagesMissingAlt != 1 || checks.LinksBlocklisted != 1 || checks.SizeBytes < 85*1024 || checks.FooterRemovedAt == nil {
		t.Fatalf("flash sale checks: %+v", checks)
	}
	if strings.Join(checks.SubjectHeuristics, ", ") != "all caps, repeated punctuation, urgency" {
		t.Fatalf("subject heuristics: %v", checks.SubjectHeuristics)
	}
	found := map[string]bool{}
	for _, is := range checks.Issues {
		found[is.ID] = true
		switch is.ID {
		case "links.blocklist":
			if is.Severity != "block" || is.Title != "Button links to go.acme-promo.link, listed on SURBL and Spamhaus DBL" || is.Fixes[0].Label != "Replace with acme.io/sale" || len(is.Path) == 0 {
				t.Fatalf("blocklist issue: %+v", is)
			}
		case "compliance.footer":
			if is.Severity != "block" || is.Title != "No unsubscribe link or postal address in the body" {
				t.Fatalf("footer issue: %+v", is)
			}
		case "content.alt_text":
			if is.Title != "Hero image has no alt text" || len(is.Path) != 3 {
				t.Fatalf("alt issue: %+v", is)
			}
		case "subject.spam":
			if is.Fixes[0].Value != "48-hour flash sale" {
				t.Fatalf("subject suggestion: %+v", is)
			}
		case "content.size":
			if !strings.Contains(is.Detail, "Four product cards with inline CSS are the bulk of it.") || len(is.Fixes) == 0 || is.Fixes[len(is.Fixes)-1].Label != "Remove the second row" {
				t.Fatalf("size issue: %+v", is)
			}
		}
	}
	// Exactly the ComposerWarnings board's set: two blocks, alt text, size, subject.
	for _, id := range []string{"links.blocklist", "compliance.footer", "content.alt_text", "content.size", "subject.spam"} {
		if !found[id] {
			t.Fatalf("missing %s in %+v", id, checks.Issues)
		}
	}
	if len(found) != 5 {
		t.Fatalf("unexpected issues: %+v", checks.Issues)
	}

	// The digest is clean.
	digest := campaignID(t, env, "Weekly Digest #142")
	c.Do("GET", ws+"/campaigns/"+digest, nil).MustStatus(t, 200).JSON(t, &d)
	c.Do("POST", ws+"/campaigns/"+digest+"/checks", map[string]any{"subject": d.Subject, "preheader": d.Preheader, "mjml": d.MJML, "html": d.HTML}).MustStatus(t, 200).JSON(t, &checks)
	if checks.Verdict != "ready" {
		t.Fatalf("digest checks: %+v", checks.Issues)
	}

	// Preview with the fallback.
	var pv struct {
		Subject string
		HTML    string
		Tokens  []struct {
			Key          string
			Value        string
			FallbackUsed bool
			PerRecipient bool
		}
	}
	c.Do("POST", ws+"/campaigns/"+digest+"/preview", map[string]any{"sample": "no_first_name"}).MustStatus(t, 200).JSON(t, &pv)
	if !strings.Contains(pv.HTML, "Hi there,") || strings.Contains(pv.HTML, "{{") || !strings.Contains(pv.HTML, "/media/") {
		t.Fatalf("preview html")
	}
	if len(pv.Tokens) == 0 || pv.Tokens[0].Key != "first_name" || !pv.Tokens[0].FallbackUsed || pv.Tokens[0].Value != "there" {
		t.Fatalf("preview tokens: %+v", pv.Tokens)
	}
	c.Do("POST", ws+"/campaigns/"+digest+"/preview", map[string]any{"variant": "B"}).MustStatus(t, 200).JSON(t, &pv)
	if pv.Subject != "What shipped this week, and what we're retiring" {
		t.Fatalf("variant B subject: %q", pv.Subject)
	}

	// Composer context.
	var ctxOut struct {
		Senders []struct{ Email string }
		Lists   []struct{ Name string }
		Tokens  []struct{ Key, Kind string }
		Members []struct{ Email string }
	}
	c.Do("GET", ws+"/composer", nil).MustStatus(t, 200).JSON(t, &ctxOut)
	// Other seeders add senders and lists too: assert the ones the composer demo needs.
	has := func(want string, got []string) bool {
		for _, g := range got {
			if g == want {
				return true
			}
		}
		return false
	}
	var senders, lists []string
	for _, s := range ctxOut.Senders {
		senders = append(senders, s.Email)
	}
	for _, l := range ctxOut.Lists {
		lists = append(lists, l.Name)
	}
	if !has("digest@news.acme.io", senders) || !has("hello@news.acme.io", senders) || !has("Weekly Digest", lists) || !has("Product Updates", lists) || len(ctxOut.Members) == 0 {
		t.Fatalf("context: %+v", ctxOut)
	}
	keys := []string{}
	for _, tk := range ctxOut.Tokens {
		keys = append(keys, tk.Key)
	}
	for _, want := range []string{"first_name", "last_name", "email", "plan", "unsubscribe_url"} {
		if !has(want, keys) {
			t.Fatalf("token %s missing: %v", want, keys)
		}
	}

	// Server-side compile for API clients.
	var comp struct {
		HTML     string
		Errors   []struct{ Line int }
		Warnings []struct{ Message string }
	}
	c.Do("POST", ws+"/mjml/compile", map[string]any{"mjml": `<mjml><mj-body><mj-section padding="0 24px"><mj-column width="50%"><mj-image src="https://x.io/a.png" alt="" width="600px"></mj-image></mj-column><mj-column width="50%"></mj-column></mj-section></mj-body></mjml>`}).MustStatus(t, 200).JSON(t, &comp)
	if comp.HTML == "" || len(comp.Errors) != 0 || len(comp.Warnings) != 1 || !strings.Contains(comp.Warnings[0].Message, "exceeds the 50% column (276px)") {
		t.Fatalf("compile: %+v", comp.Warnings)
	}
	c.Do("POST", ws+"/mjml/compile", map[string]any{"mjml": "<mjml><mj-body><mj-section><mj-column>\n<mj-text>x</mj-text></mj-section></mj-body></mjml>"}).MustStatus(t, 200).JSON(t, &comp)
	if len(comp.Errors) != 1 || comp.Errors[0].Line != 1 {
		t.Fatalf("compile error: %+v", comp.Errors)
	}
}
