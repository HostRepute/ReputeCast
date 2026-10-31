package readiness_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
)

// undoReview is the part of the review Undo is about.
type undoReview struct {
	Fixes *readiness.ReadinessFixBatch
}

// R1.U3: fixes applied together are kept as one batch, and Undo puts the
// campaign back as it was before them: the email through a restore of the
// version before them (a version of its own), the content settings, the
// route override and the audience, each change audited. A batch that
// changed the route needs the right to send to undo, as applying it did;
// a batch can be undone once, and not after the campaign changed since.
func TestUndoFixes(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	id := campaignID(t, linh, "Weekly Digest #142")
	base := ws + "/campaigns/" + id
	type row struct {
		Route, Audience, Settings json.RawMessage
		Version                   int
	}
	campaign := func() row {
		t.Helper()
		var r row
		if err := env.App.Pool.QueryRow(ctx, `SELECT coalesce(route_override, 'null'::jsonb), audience, coalesce(content_settings, '{}'::jsonb), version FROM campaigns WHERE id = $1`, id).
			Scan(&r.Route, &r.Audience, &r.Settings, &r.Version); err != nil {
			t.Fatal(err)
		}
		return r
	}
	content := func() composerContent {
		t.Helper()
		var out composerContent
		linh.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &out)
		return out
	}
	audits := func(action string) int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = $2`, id, action).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	undone := func(batch string) int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE target_id = $1 AND metadata->>'undo' = $2`, id, batch).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	review := func(c *apptest.Client) undoReview {
		t.Helper()
		var r undoReview
		c.Do("GET", base+"/review", nil).MustStatus(t, 200).JSON(t, &r)
		return r
	}
	same := func(a, b json.RawMessage) bool {
		var x, y any
		_ = json.Unmarshal(a, &x)
		_ = json.Unmarshal(b, &y)
		return fmt.Sprint(x) == fmt.Sprint(y)
	}

	run := runChecks(t, env, linh, id)
	ses := ""
	for _, s := range run.Estimates.Route.Shares {
		if s.Adapter == "ses" {
			ses = s.ProviderID
		}
	}
	if ses == "" {
		t.Fatalf("no SES in the route: %+v", run.Estimates.Route.Shares)
	}
	before, beforeContent := campaign(), content()

	// Four fixes at once: the subject (content), Minify (content settings),
	// Send via SES only (the route) and Include them (the audience).
	type fixOut struct {
		Applied []string
		Version int
		Batch   *readiness.ReadinessFixBatch
	}
	// fresh empties what a fix answered before the next one is read into
	// it: JSON leaves fields the answer omits as they were.
	fresh := func(o *fixOut) *fixOut {
		*o = fixOut{}
		return o
	}
	var out fixOut
	readinesstest.Fix(t, env, linh, base+"/readiness/fixes", map[string]any{"fixes": []map[string]any{
		{"action": "use_subject", "params": map[string]string{"subject": "Weekly Digest: the November changelog"}},
		{"action": "minify"},
		{"action": "route_only", "params": map[string]string{"providerId": ses}},
		{"action": "include_unengaged"},
	}}).MustStatus(t, 200).JSON(t, &out)
	b := out.Batch
	if b == nil || !b.Undoable || !b.Route || b.Count != 4 || len(b.Applied) != 4 || b.Version != out.Version || out.Version <= before.Version || b.AppliedBy != "Linh Nguyen" {
		t.Fatalf("the batch: %+v (version %d, before %d)", b, out.Version, before.Version)
	}
	fixed, fixedContent := campaign(), content()
	if same(fixed.Route, before.Route) || same(fixed.Audience, before.Audience) || same(fixed.Settings, before.Settings) || fixedContent.Subject != "Weekly Digest: the November changelog" {
		t.Fatalf("the fixes didn't change the campaign: %+v", fixed)
	}
	// Minified HTML has no white space between its tags.
	spaced := regexp.MustCompile(`>\s+<`)
	if !spaced.MatchString(beforeContent.HTML) || spaced.MatchString(fixedContent.HTML) {
		t.Fatal("Minify didn't minify the email")
	}

	// The review offers it; a Vietnamese reader reads how many, not the
	// English labels.
	if r := review(linh); r.Fixes == nil || r.Fixes.ID != b.ID || len(r.Fixes.Applied) != 4 {
		t.Fatalf("the review's fixes: %+v", r.Fixes)
	}
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	if r := review(vi); r.Fixes == nil || len(r.Fixes.Applied) != 0 || r.Fixes.Count != 4 {
		t.Fatalf("the review's fixes in Vietnamese: %+v", r.Fixes)
	}

	// A Content Writer can't put the route back: who can is named, never a
	// permission key, and nothing changes.
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")
	p := daniel.Do("POST", base+"/readiness/fixes/"+b.ID+"/undo", nil).MustStatus(t, 403).Problem(t)
	if d := fmt.Sprint(p["detail"]); !strings.HasPrefix(d, "These fixes changed the route, which your role can’t change back. Ask ") || strings.Contains(d, "campaigns:") {
		t.Errorf("a writer's undo: %v", p)
	}
	if got := campaign(); got.Version != fixed.Version || !same(got.Route, fixed.Route) {
		t.Fatal("a refused undo changed the campaign")
	}

	// Linh undoes them: everything is back as it was, the email as a new
	// version restored from the one before the fixes.
	var res struct {
		Count, Restored, Version int
		Running                  *readiness.ReadinessProgress
	}
	linh.Do("POST", base+"/readiness/fixes/"+b.ID+"/undo", nil).MustStatus(t, 200).JSON(t, &res)
	if res.Count != 4 || res.Restored != before.Version || res.Version != out.Version+1 || res.Running == nil {
		t.Fatalf("undo: %+v (before v%d, fixed v%d)", res, before.Version, out.Version)
	}
	readinesstest.Work(t, env)
	after := campaign()
	if !same(after.Route, before.Route) || !same(after.Audience, before.Audience) || !same(after.Settings, before.Settings) || after.Version != res.Version {
		t.Fatalf("after undo: %+v, before %+v", after, before)
	}
	// The email comes back as it was, not minified: Minify is off again.
	if got := content(); got.Subject != beforeContent.Subject || got.MJML != beforeContent.MJML || !spaced.MatchString(got.HTML) {
		t.Fatalf("the email after undo: %q", got.Subject)
	}
	var source string
	if err := env.App.Pool.QueryRow(ctx, `SELECT source FROM campaign_versions WHERE campaign_id = $1 AND version = $2`, id, res.Version).Scan(&source); err != nil || source != "restore" {
		t.Fatalf("the undo's version: %q (%v)", source, err)
	}
	// Audited: the undo, and each part it put back (route, audience, settings).
	if audits("readiness.fixes_undone") != 1 || undone(b.ID) != 3 {
		t.Fatalf("audit: %d undo entries, %d changes", audits("readiness.fixes_undone"), undone(b.ID))
	}
	if r := review(linh); r.Fixes != nil {
		t.Fatalf("undone fixes are still offered: %+v", r.Fixes)
	}
	// Once only.
	if p := linh.Do("POST", base+"/readiness/fixes/"+b.ID+"/undo", nil).MustStatus(t, 409).Problem(t); p["code"] != "fixes_undone" {
		t.Errorf("undo twice: %v", p)
	}

	// Fixes the campaign moved past can't be undone: a save after them…
	readinesstest.Fix(t, env, linh, base+"/readiness/fixes", map[string]any{"fixes": []map[string]any{
		{"action": "use_subject", "params": map[string]string{"subject": "Weekly Digest #142 · fixed"}},
	}}).MustStatus(t, 200).JSON(t, fresh(&out))
	edited := content()
	linh.Do("PUT", base+"/content", map[string]any{"baseVersion": edited.Version, "subject": "Weekly Digest #142 · edited", "mjml": edited.MJML, "source": "manual"}).
		MustStatus(t, 200)
	if r := review(linh); r.Fixes != nil {
		t.Fatalf("fixes the campaign moved past are offered: %+v", r.Fixes)
	}
	p = linh.Do("POST", base+"/readiness/fixes/"+out.Batch.ID+"/undo", nil).MustStatus(t, 409).Problem(t)
	if p["code"] != "fixes_changed" || !strings.Contains(fmt.Sprint(p["detail"]), "Version history") {
		t.Errorf("undo after a save: %v", p)
	}
	if got := content(); got.Subject != "Weekly Digest #142 · edited" {
		t.Fatalf("a refused undo changed the email: %q", got.Subject)
	}

	// …or an audience edit after them.
	readinesstest.Fix(t, env, linh, base+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": "include_unengaged"}}}).MustStatus(t, 200).JSON(t, fresh(&out))
	if out.Batch == nil || out.Batch.Route || out.Batch.Version != 0 {
		t.Fatalf("an audience fix: %+v", out.Batch)
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	exec(`UPDATE campaigns SET audience = audience || '{"exclude_lists": []}'::jsonb || jsonb_build_object('segments', '["s-edited"]'::jsonb) WHERE id = $1`, id)
	if p := linh.Do("POST", base+"/readiness/fixes/"+out.Batch.ID+"/undo", nil).MustStatus(t, 409).Problem(t); p["code"] != "fixes_changed" {
		t.Errorf("undo after an audience edit: %v", p)
	}
	// An unknown batch is not found.
	linh.Do("POST", base+"/readiness/fixes/0199f0a4-0000-7000-8000-000000000001/undo", nil).MustStatus(t, 404)
}
