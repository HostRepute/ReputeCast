package accessibility_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/languages"
)

const ws = "/workspaces/acme-newsletter"

func campaignID(t *testing.T, c *apptest.Client, name string) string {
	t.Helper()
	var list struct {
		Items []struct{ ID, Name string }
	}
	c.Do("GET", ws+"/campaigns?pageSize=100", nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.Name == name {
			return it.ID
		}
	}
	t.Fatalf("campaign %q not found", name)
	return ""
}

func audits(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func setPolicy(t *testing.T, c *apptest.Client, onSerious string, min int) {
	t.Helper()
	c.Do("PUT", ws+"/accessibility/policy", map[string]any{"onSerious": onSerious, "minTextPx": min}).MustStatus(t, 200)
}

func TestPolicy(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh, daniel, aisha := env.Client(), env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	aisha.SignInDemo("aisha@acme.io")

	// The sample: Warn at 12 px, set by Linh Nguyen.
	var p accessibility.AccessibilityPolicy
	linh.Do("GET", ws+"/accessibility/policy", nil).MustStatus(t, 200).JSON(t, &p)
	if p.OnSerious != "warn" || p.MinTextPx != 12 || p.UpdatedBy != "Linh Nguyen" || p.UpdatedAt == nil || !p.CanManage {
		t.Fatalf("policy: %+v", p)
	}
	// Everyone reads it; only workspace:manage changes it, and says who can.
	daniel.Do("GET", ws+"/accessibility/policy", nil).MustStatus(t, 200).JSON(t, &p)
	if p.CanManage || len(p.Helpers) == 0 {
		t.Fatalf("read-only: %+v", p)
	}
	daniel.Do("PUT", ws+"/accessibility/policy", map[string]any{"onSerious": "block", "minTextPx": 12}).MustStatus(t, 403)
	aisha.Do("PUT", ws+"/accessibility/policy", map[string]any{"onSerious": "block", "minTextPx": 12}).MustStatus(t, 403)
	linh.Do("POST", ws+"/accessibility/policy/impact", map[string]any{"onSerious": "block", "minTextPx": 12}).MustStatus(t, 200)
	daniel.Do("POST", ws+"/accessibility/policy/impact", map[string]any{"onSerious": "block", "minTextPx": 12}).MustStatus(t, 403)

	// Validation.
	linh.Do("PUT", ws+"/accessibility/policy", map[string]any{"onSerious": "block", "minTextPx": 20}).MustStatus(t, 422)
	linh.Do("PUT", ws+"/accessibility/policy", map[string]any{"onSerious": "maybe", "minTextPx": 12}).MustStatus(t, 422)

	// A change is audited once with before and after; the same value again isn't.
	before := audits(t, env, "workspace.accessibility_policy")
	linh.Do("PUT", ws+"/accessibility/policy", map[string]any{"onSerious": "block", "minTextPx": 14}).MustStatus(t, 200).JSON(t, &p)
	if p.OnSerious != "block" || p.MinTextPx != 14 || p.UpdatedBy != "Linh Nguyen" {
		t.Fatalf("saved: %+v", p)
	}
	linh.Do("PUT", ws+"/accessibility/policy", map[string]any{"onSerious": "block", "minTextPx": 14}).MustStatus(t, 200)
	if n := audits(t, env, "workspace.accessibility_policy"); n != before+1 {
		t.Fatalf("audit entries: %d, want %d", n, before+1)
	}
	var b, a map[string]any
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT before, after FROM audit_events WHERE action = 'workspace.accessibility_policy' ORDER BY occurred_at DESC LIMIT 1`).Scan(&b, &a); err != nil {
		t.Fatal(err)
	}
	if b["onSerious"] != "warn" || a["onSerious"] != "block" || a["minTextPx"] != float64(14) {
		t.Fatalf("audit before/after: %v → %v", b, a)
	}
}

// The impact of Block before saving: #143 and the Welcome series draft's
// step 3 B have serious issues; no scheduled campaign would be held.
func TestImpact(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var im accessibility.AccessibilityImpact
	linh.Do("POST", ws+"/accessibility/policy/impact", map[string]any{"onSerious": "block", "minTextPx": 12}).MustStatus(t, 200).JSON(t, &im)
	var digest *accessibility.AccessibilityImpactCampaign
	for i := range im.Campaigns {
		if im.Campaigns[i].Name == "Weekly Digest #143" {
			digest = &im.Campaigns[i]
		}
	}
	if digest == nil || digest.Serious != 2 || digest.Summary != "the hero image's alt text and the button contrast" || digest.Held {
		t.Fatalf("#143: %+v (all: %+v)", digest, im.Campaigns)
	}
	var welcome *accessibility.AccessibilityImpactFlow
	for i := range im.Flows {
		if im.Flows[i].Name == "Welcome series" {
			welcome = &im.Flows[i]
		}
	}
	if welcome == nil || welcome.Step != "step 3 B" || welcome.Serious != 1 || welcome.Summary != "button contrast" || welcome.Version != 4 || welcome.LiveVersion == nil || *welcome.LiveVersion != 3 {
		t.Fatalf("Welcome series: %+v (all: %+v)", welcome, im.Flows)
	}
	if im.Held != 0 {
		t.Fatalf("held: %d", im.Held)
	}
	// A scheduled campaign with a serious issue would be held at its recheck.
	digest142 := campaignID(t, linh, "Weekly Digest #142")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaigns SET mjml = replace(mjml, 'alt="Q4 roadmap illustration" ', '') WHERE id = $1`, digest142); err != nil {
		t.Fatal(err)
	}
	linh.Do("POST", ws+"/accessibility/policy/impact", map[string]any{"onSerious": "block", "minTextPx": 12}).MustStatus(t, 200).JSON(t, &im)
	if im.Held != 1 {
		t.Fatalf("held after the alt text went: %+v", im)
	}
}

func TestCampaignChecks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh, aisha := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	aisha.SignInDemo("aisha@acme.io")
	id := campaignID(t, linh, "Weekly Digest #143")

	// The saved content: six issues; "Click here" gets the docs page's title.
	var r accessibility.AccessibilityReport
	aisha.Do("GET", ws+"/campaigns/"+id+"/accessibility", nil).MustStatus(t, 200).JSON(t, &r)
	if len(r.Issues) != 6 || r.Serious != 2 || r.Policy != "warn" || r.Counts.Images != 7 || r.Counts.LinksOK != 13 {
		t.Fatalf("report: %d issues, %d serious, %+v", len(r.Issues), r.Serious, r.Counts)
	}
	if v := r.Issues[2].Versions; len(v) != 1 || v[0].Suggestion != "Scheduled rollouts · Acme Docs" {
		t.Fatalf("click here: %+v", r.Issues[2])
	}

	// The composer's unsaved content: alt text written on the hero.
	var c struct{ MJML string }
	linh.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &c)
	fixed := strings.Replace(c.MJML, `width="552px" height="200px"`, `alt="A flag change scheduled for 2 a.m." width="552px" height="200px"`, 1)
	linh.Do("POST", ws+"/campaigns/"+id+"/accessibility", map[string]any{"mjml": fixed}).MustStatus(t, 200).JSON(t, &r)
	if len(r.Issues) != 5 || r.Serious != 1 {
		t.Fatalf("after alt text: %d issues, %d serious", len(r.Issues), r.Serious)
	}
	// MJML that doesn't compile can't be checked; the edits are safe.
	pr := linh.Do("POST", ws+"/campaigns/"+id+"/accessibility", map[string]any{"mjml": "<mjml><mj-body><mj-section></mj-body></mjml>"}).MustStatus(t, 422).Problem(t)
	if pr["code"] != "accessibility_unchecked" {
		t.Fatalf("problem: %v", pr)
	}
}

// Saving content stores lang, dir and the layout tables' role.
func TestFinalizeOnSave(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	id := campaignID(t, linh, "Weekly Digest #143")
	var c struct {
		MJML, Subject, Preheader string
		Version                  int
	}
	linh.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &c)
	html := `<!doctype html><html><body><div><table><tr><td>Hi</td></tr></table><table><tr><th>Plan</th></tr></table></div></body></html>`
	var saved struct{ HTML string }
	linh.Do("PUT", ws+"/campaigns/"+id+"/content", map[string]any{"baseVersion": c.Version, "subject": c.Subject, "preheader": c.Preheader, "mjml": c.MJML, "html": html}).
		MustStatus(t, 200).JSON(t, &saved)
	if !strings.Contains(saved.HTML, `<html lang="en" dir="ltr">`) || !strings.Contains(saved.HTML, `<table role="presentation"><tr><td>Hi`) || strings.Contains(saved.HTML, `role="presentation"><tr><th>`) {
		t.Fatalf("stored HTML: %s", saved.HTML)
	}
}

type readinessRun struct {
	Verdict string
	Counts  struct{ Blocks, Warnings int }
	Groups  []struct{ Key, Caption, Note string }
	Checks  []struct {
		Key, Group, Status, Title, Detail string
		Fixes                             []struct {
			Kind, Action, Label string
			Params              map[string]string
		}
	}
}

func (r readinessRun) group(key string) []string {
	var out []string
	for _, c := range r.Checks {
		if c.Group == key {
			out = append(out, c.Status+" "+c.Key)
		}
	}
	return out
}

func TestReadinessGroupAndBlock(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	id := campaignID(t, linh, "Weekly Digest #143")

	var run readinessRun
	linh.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 200).JSON(t, &run)
	got := strings.Join(run.group("accessibility"), ", ")
	if !strings.HasPrefix(got, "warn accessibility.alt_text") || !strings.Contains(got, "warn accessibility.contrast") || !strings.Contains(got, "warn accessibility.moderate") ||
		!strings.Contains(got, "pass accessibility.lang_dir") || !strings.Contains(got, "pass accessibility.layout_tables") {
		t.Fatalf("group: %s", got)
	}
	for _, c := range run.Checks {
		if c.Key == "content.alt_text" {
			t.Fatal("alt text moved to Accessibility")
		}
	}
	if run.Verdict == "blocked" {
		t.Fatalf("Warn doesn't block: %+v", run.Counts)
	}

	// Block: the serious rows block, the verdict turns Blocked and says why.
	setPolicy(t, linh, "block", 12)
	linh.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 200).JSON(t, &run)
	if run.Verdict != "blocked" || run.Counts.Blocks != 2 {
		t.Fatalf("block: %s %+v", run.Verdict, run.Counts)
	}
	for _, g := range run.Groups {
		if g.Key == "accessibility" && !strings.HasPrefix(g.Note, "Your workspace blocks sends with serious accessibility issues (Settings › General). Changed by Linh Nguyen") {
			t.Fatalf("note: %q", g.Note)
		}
	}
	// Sending is refused.
	resp := linh.Do("POST", ws+"/campaigns/"+id+"/send", map[string]any{"runId": "", "acknowledged": []string{}, "confirm": "31604"})
	if resp.Status < 400 {
		t.Fatalf("a blocked send went through: %d", resp.Status)
	}
}

func TestReadinessFixes(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh, aisha := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	aisha.SignInDemo("aisha@acme.io")
	id := campaignID(t, linh, "Weekly Digest #143")
	fix := func(c *apptest.Client, action string, params map[string]string) *apptest.Response {
		return c.Do("POST", ws+"/campaigns/"+id+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": action, "params": params}}})
	}
	aisha.Do("POST", ws+"/campaigns/"+id+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": "accessibility_one_click"}}}).MustStatus(t, 403)

	// "Fix 2 in one click": headings and the footer size, audited, the checks run again.
	before := audits(t, env, "campaign.accessibility_fix")
	var out struct {
		Applied []string
		Version int
		Run     *readinessRun
	}
	fix(linh, "accessibility_one_click", nil).MustStatus(t, 200).JSON(t, &out)
	if len(out.Applied) != 1 || out.Applied[0] != "Changed 2 headings to Heading 2 and set footer text to 12 px" || out.Version != 10 || out.Run == nil {
		t.Fatalf("one click: %+v", out)
	}
	if audits(t, env, "campaign.accessibility_fix") != before+1 {
		t.Fatal("the fix is audited")
	}
	var fixedRow, moderate string
	for _, c := range out.Run.Checks {
		switch c.Key {
		case "accessibility.fixed":
			fixedRow = c.Title + " | " + c.Detail
		case "accessibility.moderate":
			moderate = c.Title
		}
	}
	if !strings.HasPrefix(fixedRow, "Card titles are Heading 2, and footer text is 12 px | Fixed by Linh Nguyen at ") || !strings.HasSuffix(fixedRow, "in English · recorded in the audit log") {
		t.Fatalf("fixed row: %q", fixedRow)
	}
	if moderate != "2 moderate issues: “Click here” and a bare URL" {
		t.Fatalf("moderate: %q", moderate)
	}
	// Nothing left to fix in one click.
	fix(linh, "accessibility_one_click", nil).MustStatus(t, 422)

	// The suggested color and a decorative hero clear the serious issues.
	var params map[string]string
	for _, c := range out.Run.Checks {
		if strings.HasPrefix(c.Key, "accessibility.contrast") {
			params = c.Fixes[0].Params
		}
	}
	if params["color"] != "#0F766E" {
		t.Fatalf("contrast fix: %v", params)
	}
	fix(linh, "accessibility_color", params).MustStatus(t, 200).JSON(t, &out)
	fix(linh, "accessibility_decorative", map[string]string{"path": "1-0-0", "index": "0"}).MustStatus(t, 200).JSON(t, &out)
	for _, c := range out.Run.Checks {
		if c.Group == "accessibility" && (c.Status == "warn" || c.Status == "block") && c.Key != "accessibility.moderate" {
			t.Fatalf("still serious: %s", c.Key)
		}
	}
	// The fix points at an element that isn't there any more.
	fix(linh, "accessibility_decorative", map[string]string{"path": "40-0-0", "index": "0"}).MustStatus(t, 422)
}

func TestAutomationPublishGate(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var list struct {
		Items []struct{ ID, Name string }
	}
	linh.Do("GET", ws+"/automations", nil).MustStatus(t, 200).JSON(t, &list)
	id := ""
	for _, a := range list.Items {
		if a.Name == "Welcome series" {
			id = a.ID
		}
	}
	type preview struct {
		ToVersion  int
		CanPublish bool
		Blocker    string
		Checks     []struct{ Status, Label string }
	}
	var p preview
	linh.Do("GET", ws+"/automations/"+id+"/publish-preview", nil).MustStatus(t, 200).JSON(t, &p)
	badge := func(p preview) string {
		for _, c := range p.Checks {
			if strings.HasPrefix(c.Label, "Step 3 B:") {
				return c.Status + " " + c.Label
			}
		}
		return ""
	}
	if badge(p) != "warn Step 3 B: 1 serious issue, button contrast" || !p.CanPublish {
		t.Fatalf("warn: %q %+v", badge(p), p)
	}

	setPolicy(t, linh, "block", 12)
	linh.Do("GET", ws+"/automations/"+id+"/publish-preview", nil).MustStatus(t, 200).JSON(t, &p)
	if badge(p) != "bad Step 3 B: 1 serious issue, button contrast" || p.CanPublish ||
		p.Blocker != "Your workspace blocks publishing with serious accessibility issues (Settings › General). Fix the button contrast in step 3 B to publish v4. The live v3 keeps sending." {
		t.Fatalf("block: %q %+v", badge(p), p)
	}
	pr := linh.Do("POST", ws+"/automations/"+id+"/publish", map[string]any{"version": p.ToVersion, "migration": "finish"}).MustStatus(t, 409).Problem(t)
	if pr["code"] != "accessibility_blocked" {
		t.Fatalf("publish: %v", pr)
	}
}

func TestGallery(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var g accessibility.AccessibilityGallery
	linh.Do("GET", ws+"/accessibility/templates", nil).MustStatus(t, 200).JSON(t, &g)
	found := false
	for _, r := range g.Templates {
		if strings.HasPrefix(r.ID, "starter:") && r.Issues != 0 {
			t.Errorf("starter %s: %+v", r.ID, r)
		}
		if r.Serious == 1 && r.Issues == 2 {
			found = true
			if r.Summary != "Serious: the linked logo has no alt text, so its link has no text. Also two “Read more” links." {
				t.Errorf("Acme digest v3: %q", r.Summary)
			}
		}
	}
	if !found || g.Language != "en" {
		t.Fatalf("gallery: %+v", g)
	}
	if g.LastCampaign == nil || g.LastCampaign.Issues != 0 {
		t.Fatalf("Copy of October Product Update: %+v", g.LastCampaign)
	}
	raw, _ := json.Marshal(g)
	if strings.Contains(string(raw), "null") {
		t.Fatalf("no nulls in lists: %s", raw)
	}
}

// Language versions are checked as they are sent, are stored in their own
// language, and get the fixes that need no words.
func TestLanguageVersions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	ctx := context.Background()
	id := campaignID(t, linh, "Weekly Digest #143")
	if resp := linh.Do("POST", ws+"/languages", map[string]any{"code": "es"}); resp.Status >= 300 {
		t.Fatalf("add Spanish: %d %s", resp.Status, resp.Body)
	}
	// The Spanish version, added and marked translated through the content
	// languages service.
	acme, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	u, err := env.App.Q.GetUserByEmail(ctx, "linh@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	p := &authz.Principal{Type: authz.ActorUser, ID: u.ID, UserID: u.ID, Name: u.Name, WorkspaceID: acme.ID, Permissions: authz.All()}
	content := &languages.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}
	cid := uuid.MustParse(id)
	if _, err := content.AddCampaignLanguage(ctx, &acme, p, cid, languages.AddLanguageInput{Language: "es"}); err != nil {
		t.Fatal(err)
	}
	stored := func() (mj, html string) {
		t.Helper()
		if err := env.App.Pool.QueryRow(ctx, `SELECT mjml, html FROM content_translations WHERE owner_id = $1 AND language = 'es'`, id).Scan(&mj, &html); err != nil {
			t.Fatal(err)
		}
		return mj, html
	}
	if _, html := stored(); !strings.Contains(html, `<html lang="es" dir="ltr"`) {
		t.Fatalf("the Spanish version is stored in Spanish: %.300s", html)
	}
	langsOf := func(r accessibility.AccessibilityReport) string {
		var codes []string
		for _, l := range r.Languages {
			codes = append(codes, l.Code)
		}
		return strings.Join(codes, ",")
	}

	// A Draft isn't sent, so it isn't checked.
	var r accessibility.AccessibilityReport
	linh.Do("GET", ws+"/campaigns/"+id+"/accessibility", nil).MustStatus(t, 200).JSON(t, &r)
	if langsOf(r) != "en" {
		t.Fatalf("draft checked: %s", langsOf(r))
	}
	// Translated, it is: each issue is listed once, with both languages.
	if _, err := content.MarkCampaignLanguage(ctx, &acme, p, cid, "es", true, languages.MarkInput{}); err != nil {
		t.Fatal(err)
	}
	linh.Do("GET", ws+"/campaigns/"+id+"/accessibility", nil).MustStatus(t, 200).JSON(t, &r)
	if langsOf(r) != "en,es" || len(r.Issues) != 6 || strings.Join(r.Issues[0].Languages, ",") != "en,es" {
		t.Fatalf("both versions: %s, %d issues, %+v", langsOf(r), len(r.Issues), r.Issues[0].Languages)
	}
	// The Spanish canvas replaces the stored Spanish version: alt text
	// written there leaves the issue in English only.
	mj, _ := stored()
	es := strings.Replace(mj, `width="552px" height="200px"`, `alt="Un cambio programado para las 2 a. m." width="552px" height="200px"`, 1)
	linh.Do("POST", ws+"/campaigns/"+id+"/accessibility", map[string]any{"lang": "es", "mjml": es}).MustStatus(t, 200).JSON(t, &r)
	if r.Issues[0].Kind != accessibility.KindAltText || strings.Join(r.Issues[0].Languages, ",") != "en" {
		t.Fatalf("alt text in Spanish: %+v", r.Issues[0])
	}

	// "Fix 2 in one click" fixes every version and says so.
	var out struct {
		Applied []string
		Run     *readinessRun
	}
	linh.Do("POST", ws+"/campaigns/"+id+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": "accessibility_one_click"}}}).MustStatus(t, 200).JSON(t, &out)
	if mj, _ := stored(); strings.Count(mj, "<h2") < 2 || strings.Contains(mj, "<h3") {
		t.Fatalf("the Spanish card titles stay Heading 3: %s", mj)
	}
	var langs []string
	if err := env.App.Pool.QueryRow(ctx, `SELECT languages FROM accessibility_fixes WHERE campaign_id = $1 ORDER BY fixed_at DESC LIMIT 1`, id).Scan(&langs); err != nil {
		t.Fatal(err)
	}
	if strings.Join(langs, ",") != "en,es" {
		t.Fatalf("fixed languages: %v", langs)
	}
	if out.Run == nil {
		t.Fatal("the checks run again")
	}
	for _, c := range out.Run.Checks {
		if c.Key == "accessibility.fixed" && !strings.HasSuffix(c.Detail, "in English and Spanish · recorded in the audit log") {
			t.Fatalf("fixed row: %q", c.Detail)
		}
	}
}

// Transactional templates: the Accessibility group in their checks, lang
// and dir on the stored draft, and the Block policy holding publishing.
func TestTransactionalTemplates(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	ctx := context.Background()
	const tx = ws + "/transactional/templates"
	mj := `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-image src="https://acme.io/img/lock.png" width="120px"></mj-image>` +
		`<mj-text><p>Use the button below to choose a new password.</p></mj-text>` +
		`<mj-button href="https://acme.io/reset" background-color="#5EBFB0" color="#FFFFFF">Choose a new password</mj-button>` +
		`</mj-column></mj-section></mj-body></mjml>`
	linh.Do("POST", tx, map[string]any{"name": "A11y reset", "key": "a11y-reset", "subject": "Choose a new password", "editorMode": "mjml", "mjml": mj}).MustStatus(t, 201)
	var html string
	if err := env.App.Pool.QueryRow(ctx, `SELECT html FROM transactional_templates WHERE key = 'a11y-reset'`).Scan(&html); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `<html lang="en" dir="ltr"`) {
		t.Fatalf("the draft is stored with lang and dir: %.200s", html)
	}

	type check struct{ ID, Severity, Group, Title, Detail string }
	var checks struct {
		Verdict string
		Checks  []check
	}
	group := func() []string {
		var out []string
		for _, c := range checks.Checks {
			if c.Group == "Accessibility" {
				out = append(out, c.Severity+" "+c.ID)
			}
			if c.ID == "content.alt_text" {
				t.Fatal("alt text is in the Accessibility group")
			}
		}
		return out
	}
	linh.Do("POST", tx+"/a11y-reset/checks", map[string]any{}).MustStatus(t, 200).JSON(t, &checks)
	got := strings.Join(group(), ", ")
	if !strings.Contains(got, "warn accessibility.alt_text:") || !strings.Contains(got, "warn accessibility.contrast:") || !strings.Contains(got, "pass accessibility.lang_dir") {
		t.Fatalf("Warn: %s", got)
	}

	// Block: publishing waits for the fix.
	setPolicy(t, linh, "block", 12)
	linh.Do("POST", tx+"/a11y-reset/checks", map[string]any{}).MustStatus(t, 200).JSON(t, &checks)
	if got := strings.Join(group(), ", "); checks.Verdict != "blocked" || !strings.Contains(got, "block accessibility.contrast:") {
		t.Fatalf("Block: %s %s", checks.Verdict, got)
	}
	var pv struct {
		CanPublish bool
		Blocking   []string
	}
	linh.Do("POST", tx+"/a11y-reset/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.CanPublish || len(pv.Blocking) < 2 {
		t.Fatalf("publish preview: %+v", pv)
	}
	var tpl struct{ DraftRevision int }
	linh.Do("GET", tx+"/a11y-reset", nil).MustStatus(t, 200).JSON(t, &tpl)
	linh.Do("POST", tx+"/a11y-reset/publish", map[string]any{"baseRevision": tpl.DraftRevision}).MustStatus(t, 409)

	// Alt text and a darker button: it publishes.
	fixed := strings.Replace(strings.Replace(mj, `width="120px"`, `alt="A padlock" width="120px"`, 1), "#5EBFB0", "#0F766E", 1)
	var saved struct{ DraftRevision int }
	linh.Do("PUT", tx+"/a11y-reset/draft", map[string]any{"baseRevision": tpl.DraftRevision, "subject": "Choose a new password", "editorMode": "mjml", "mjml": fixed}).
		MustStatus(t, 200).JSON(t, &saved)

	// The Accessibility group comes after the other checks; the publish
	// dialog keeps "No accessibility issues" of its passing checks.
	passes := func(list []check) []string {
		var out []string
		for _, c := range list {
			if c.Severity == "pass" {
				out = append(out, c.ID)
			}
		}
		return out
	}
	linh.Do("POST", tx+"/a11y-reset/checks", map[string]any{}).MustStatus(t, 200).JSON(t, &checks)
	tab := passes(checks.Checks)
	if n := len(tab); n < 4 || strings.HasPrefix(tab[0], "accessibility.") || strings.Join(tab[n-3:], ",") != "accessibility.ok,accessibility.lang_dir,accessibility.layout_tables" {
		t.Fatalf("Checks tab: %v", tab)
	}
	var dialog struct {
		Checks struct {
			Passed int
			Checks []check
		}
	}
	linh.Do("POST", tx+"/a11y-reset/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &dialog)
	shown := passes(dialog.Checks.Checks)
	if n := len(shown); n == 0 || dialog.Checks.Passed != n || shown[n-1] != "accessibility.ok" || strings.HasPrefix(shown[0], "accessibility.") || strings.Contains(strings.Join(shown, ","), "lang_dir") {
		t.Fatalf("publish dialog: %d %v", dialog.Checks.Passed, shown)
	}
	// HTML stored without the additions (seeded, or saved before them) gets
	// them when it is published.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE transactional_templates SET html = replace(html, ' lang="en" dir="ltr"', '') WHERE key = 'a11y-reset'`); err != nil {
		t.Fatal(err)
	}
	linh.Do("POST", tx+"/a11y-reset/publish", map[string]any{"baseRevision": saved.DraftRevision}).MustStatus(t, 200)
	var published string
	if err := env.App.Pool.QueryRow(ctx, `SELECT v.html FROM transactional_template_versions v JOIN transactional_templates t ON t.id = v.template_id
		WHERE t.key = 'a11y-reset' ORDER BY v.version DESC LIMIT 1`).Scan(&published); err != nil {
		t.Fatal(err)
	}
	_, rest, _ := strings.Cut(published, "<html")
	if tag, _, _ := strings.Cut(rest, ">"); !strings.Contains(tag, ` lang="en" dir="ltr"`) {
		t.Fatalf("the published version has lang and dir: %.200s", published)
	}
}

// The composer's fixes change the MJML the way Review & send's do; people
// who can't edit campaigns can't use them, and editors learn who can.
func TestComposerFix(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh, aisha := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	aisha.SignInDemo("aisha@acme.io")
	id := campaignID(t, linh, "Weekly Digest #143")
	var c struct{ MJML string }
	linh.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &c)
	var r accessibility.AccessibilityReport
	linh.Do("GET", ws+"/campaigns/"+id+"/accessibility", nil).MustStatus(t, 200).JSON(t, &r)
	var heading accessibility.AccessibilityIssue
	for _, is := range r.Issues {
		if is.Kind == accessibility.KindHeading {
			heading = is
		}
	}
	if heading.Fix.Subject != "Card titles" || len(heading.Fix.Headings) != 2 {
		t.Fatalf("heading issue: %+v", heading.Fix)
	}
	body := map[string]any{"mjml": c.MJML, "kind": "heading", "path": heading.Path, "headings": heading.Fix.Headings}
	aisha.Do("POST", ws+"/accessibility/fix", body).MustStatus(t, 403)
	var fixed struct{ MJML string }
	linh.Do("POST", ws+"/accessibility/fix", body).MustStatus(t, 200).JSON(t, &fixed)
	if strings.Count(fixed.MJML, "<h2") != strings.Count(c.MJML, "<h2")+2 || strings.Contains(fixed.MJML, "<h3") {
		t.Fatalf("headings not changed")
	}
	// Nothing is saved: the campaign keeps its content.
	linh.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &c)
	if !strings.Contains(c.MJML, "<h3") {
		t.Fatal("the fix endpoint saved the campaign")
	}
	// An element that isn't there any more.
	pr := linh.Do("POST", ws+"/accessibility/fix", map[string]any{"mjml": c.MJML, "kind": "decorative", "path": []int{40, 0, 0}}).MustStatus(t, 422).Problem(t)
	if !strings.Contains(fmt.Sprint(pr["detail"]), "changed since the checks ran") {
		t.Fatalf("problem: %v", pr)
	}
	// Aisha can't edit: the composer names who can.
	var detail struct {
		Access struct {
			CanEdit     bool
			EditHelpers []string
		}
	}
	aisha.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &detail)
	if detail.Access.CanEdit || len(detail.Access.EditHelpers) == 0 {
		t.Fatalf("edit helpers: %+v", detail.Access)
	}
}
