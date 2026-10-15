package accessibility_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/apptest"
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
	if moderate != "2 moderate issues: “Click here”, a bare URL" {
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
