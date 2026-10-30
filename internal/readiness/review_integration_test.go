package readiness_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
)

// runChecks runs Send Readiness on a campaign and returns the stored run.
func runChecks(t *testing.T, env *apptest.Env, c *apptest.Client, id string) readiness.ReadinessRun {
	t.Helper()
	return readinesstest.Run(t, env, c, "acme-newsletter", id)
}

func checkOf(run readiness.ReadinessRun, key string) (readiness.ReadinessCheck, bool) {
	for _, c := range run.Checks {
		if c.Key == key {
			return c, true
		}
	}
	return readiness.ReadinessCheck{}, false
}

// SR-1: on a real install (no demo row in link_domain_listings), a listed
// link domain is found by a live lookup: it blocks in Review and shows in
// the composer's checks, and the answer is kept for the next check.
func TestLinkDomainsLookedUpLive(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `DELETE FROM link_domain_listings`); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	id := campaignID(t, c, "Flash sale: 48 hours only")

	// The composer's inline checks.
	var checks struct {
		Issues []struct{ ID, Severity, Title string }
	}
	c.Do("POST", ws+"/campaigns/"+id+"/checks", map[string]any{"subject": "Flash sale",
		"html": `<p>Acme Inc · 500 Market St, San Francisco, CA 94105</p><a href="https://go.acme-promo.link/sale">Claim your discount</a> <a href="{{ unsubscribe_url }}">Unsubscribe</a>`}).
		MustStatus(t, 200).JSON(t, &checks)
	found := false
	for _, is := range checks.Issues {
		if is.ID == "links.blocklist" && strings.Contains(is.Title, "go.acme-promo.link") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the composer doesn't flag the listed domain: %+v", checks.Issues)
	}
	var source string
	var lists []string
	if err := env.App.Pool.QueryRow(ctx, `SELECT source, lists FROM link_domain_listings WHERE domain = 'go.acme-promo.link'`).Scan(&source, &lists); err != nil {
		t.Fatalf("the answer isn't kept: %v", err)
	}
	if source != "dns" || len(lists) != 2 {
		t.Errorf("kept %s %v", source, lists)
	}

	// Review & send blocks.
	run := runChecks(t, env, c, id)
	ch, ok := checkOf(run, "links.blocklist.go.acme-promo.link")
	if !ok || ch.Status != readiness.Block || !strings.Contains(ch.Title, "Spamhaus DBL") {
		t.Fatalf("Review: %+v", ch)
	}
	if run.Verdict != readiness.VerdictBlocked {
		t.Errorf("verdict %s", run.Verdict)
	}
}

// SR-2: acme.io listed (Deliverability watches the parent domain) shows on
// the review of a news.acme.io campaign.
func TestParentDomainListingShows(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	tag, err := env.App.Pool.Exec(ctx, `UPDATE blocklist_results SET listed = true, impact = 'high', listed_since = now()
		WHERE asset = 'acme.io' AND source = 'SURBL' AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`)
	if err != nil || tag.RowsAffected() == 0 {
		t.Fatalf("listing acme.io: %v (%d rows)", err, tag.RowsAffected())
	}
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	run := runChecks(t, env, c, campaignID(t, c, "Weekly Digest #143"))
	ch, _ := checkOf(run, "reputation.domain_blocklists")
	if ch.Status != readiness.Block || !strings.Contains(ch.Title, "acme.io, the parent domain of news.acme.io") {
		t.Fatalf("parent listing: %s %q", ch.Status, ch.Title)
	}
}

// R1.L2, F-3: a parent domain HostRepute hasn't checked yet is looked up
// live on Spamhaus DBL and SURBL, as link domains are: listed, it blocks;
// clean, the check says it was looked up live. It is never taken for clean
// unchecked.
func TestUncheckedParentDomainIsLookedUpLive(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	// HostRepute's results don't cover acme.io (not checked yet).
	tag, err := env.App.Pool.Exec(ctx, `DELETE FROM blocklist_results WHERE asset = 'acme.io' AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`)
	if err != nil || tag.RowsAffected() == 0 {
		t.Fatalf("forgetting acme.io: %v (%d rows)", err, tag.RowsAffected())
	}
	// A fresh live answer: acme.io is on Spamhaus DBL.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO link_domain_listings (domain, lists, source, checked_at) VALUES ('acme.io', '{Spamhaus DBL}', 'dns', now())
		ON CONFLICT (domain) DO UPDATE SET lists = EXCLUDED.lists, source = EXCLUDED.source, checked_at = EXCLUDED.checked_at`); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	id := campaignID(t, c, "Weekly Digest #143")
	run := runChecks(t, env, c, id)
	ch, _ := checkOf(run, "reputation.domain_blocklists")
	if ch.Status != readiness.Block || ch.Title != "acme.io, the parent domain of news.acme.io, is listed on Spamhaus DBL" {
		t.Fatalf("parent listed, looked up live: %s %q", ch.Status, ch.Title)
	}

	// No answer kept: the run looks it up (the demo's lists don't list it).
	if _, err := env.App.Pool.Exec(ctx, `DELETE FROM link_domain_listings WHERE domain = 'acme.io'`); err != nil {
		t.Fatal(err)
	}
	run = runChecks(t, env, c, id)
	ch, _ = checkOf(run, "reputation.domain_blocklists")
	if ch.Status != readiness.Pass || !strings.Contains(ch.Detail, "acme.io (parent domain)") || !strings.Contains(ch.Detail, "looked up live") ||
		strings.Contains(ch.Title, " and acme.io ") {
		t.Fatalf("parent looked up live, clean: %s %q %q", ch.Status, ch.Title, ch.Detail)
	}
	var lists []string
	if err := env.App.Pool.QueryRow(ctx, `SELECT lists FROM link_domain_listings WHERE domain = 'acme.io'`).Scan(&lists); err != nil || len(lists) != 0 {
		t.Errorf("the live answer is kept: %v %v", lists, err)
	}
}

type composerContent struct {
	ID, Subject, MJML, HTML string
	Version                 int
}

// SR-4: one-click fixes keep a campaign written in HTML intact: Minify and
// Use suggestion keep its body, Replace link edits the HTML, a fix that only
// edits MJML is refused; saving an empty body over a body is refused.
func TestFixesKeepHTMLOnlyCampaigns(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var cp composerContent
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "HTML only", "fromCampaignId": campaignID(t, c, "Weekly Digest #143")}).MustStatus(t, 201).JSON(t, &cp)
	html := `<html><body><table><tr><td>  <p>Hello   there</p>   <a href="https://bit.ly/x">Read the post</a>
		<p>Acme Inc · 500 Market St, San Francisco, CA 94105</p> <a href="{{ unsubscribe_url }}">Unsubscribe</a></td></tr></table></body></html>`
	c.Do("PUT", ws+"/campaigns/"+cp.ID+"/content", map[string]any{"baseVersion": cp.Version, "subject": "FLASH SALE!!! 48 HOURS ONLY", "html": html, "source": "manual"}).
		MustStatus(t, 200)
	get := func() composerContent {
		var out composerContent
		c.Do("GET", ws+"/campaigns/"+cp.ID, nil).MustStatus(t, 200).JSON(t, &out)
		return out
	}
	fix := func(want int, fixes ...map[string]any) *apptest.Response {
		return c.Do("POST", ws+"/campaigns/"+cp.ID+"/readiness/fixes", map[string]any{"fixes": fixes}).MustStatus(t, want)
	}
	if before := get(); before.MJML != "" || !strings.Contains(before.HTML, "Hello") {
		t.Fatalf("not HTML only: %+v", before)
	}

	fix(200, map[string]any{"action": "minify"}, map[string]any{"action": "use_subject", "params": map[string]string{"subject": "48-hour flash sale"}})
	after := get()
	if after.Subject != "48-hour flash sale" || after.MJML != "" || !strings.Contains(after.HTML, "Hello") || !strings.Contains(after.HTML, "Read the post") ||
		!strings.Contains(after.HTML, "bit.ly/x") {
		t.Fatalf("after Minify and Use suggestion: %+v", after)
	}
	if strings.Contains(after.HTML, "\n") || strings.Contains(after.HTML, "</a> <p>") {
		t.Errorf("not minified: %s", after.HTML)
	}

	// A replacement address on the URI blocklists is refused, and nothing
	// changes (R1.U3).
	listed := fix(422, map[string]any{"action": "replace_link", "params": map[string]string{"from": "https://bit.ly/x", "to": "https://go.acme-promo.link/sale"}}).Problem(t)
	if p := fmt.Sprint(listed); !strings.Contains(p, "go.acme-promo.link is on ") || !strings.Contains(p, "Spamhaus DBL") || !strings.Contains(p, " too, so the link wasn’t changed") {
		t.Errorf("listed destination: %v", listed)
	}
	if got := get(); got.Version != after.Version || !strings.Contains(got.HTML, "bit.ly/x") {
		t.Fatalf("a refused replacement changed the campaign: v%d", got.Version)
	}
	fix(200, map[string]any{"action": "replace_link", "params": map[string]string{"from": "https://bit.ly/x", "to": "https://acme.io/blog/post"}})
	if after = get(); !strings.Contains(after.HTML, `href="https://acme.io/blog/post"`) || strings.Contains(after.HTML, "bit.ly") || after.MJML != "" {
		t.Fatalf("after Replace link: %s", after.HTML)
	}

	// The footer block is MJML: the editor adds it to HTML.
	if p := fix(422, map[string]any{"action": "insert_footer"}).Problem(t); !strings.Contains(fmt.Sprint(p), "written in HTML") {
		t.Errorf("insert footer: %v", p)
	}
	if got := get(); got.Version != after.Version || got.HTML != after.HTML {
		t.Errorf("a refused fix changed the campaign")
	}

	// Saving neither MJML nor HTML over a body is refused, and nothing changes.
	p := c.Do("PUT", ws+"/campaigns/"+cp.ID+"/content", map[string]any{"baseVersion": after.Version, "subject": "x"}).MustStatus(t, 422).Problem(t)
	if !strings.Contains(fmt.Sprint(p), "leave the email empty") {
		t.Errorf("empty save: %v", p)
	}
	if got := get(); got.HTML != after.HTML || got.Version != after.Version {
		t.Errorf("an empty save changed the campaign: v%d", got.Version)
	}
}

// SR-8: asking for a run queues it and answers at once (no checks run
// inside the request); one run at a time per campaign; the poll is cheap
// and says when the new run landed, or why it failed.
func TestRunsAreBackgroundJobs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	id := campaignID(t, c, "Weekly Digest #143")
	before := runChecks(t, env, c, id)

	var poll readiness.ReadinessPoll
	c.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 202).JSON(t, &poll)
	if poll.Running == nil || !poll.Running.Queued || poll.Latest == nil || poll.Latest.ID != before.ID {
		t.Fatalf("queued: %+v", poll)
	}
	var runs int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM readiness_runs WHERE campaign_id = $1 AND ran_at > $2`, id, before.RanAt).Scan(&runs); err != nil || runs != 0 {
		t.Fatalf("a run was stored inside the request: %d %v", runs, err)
	}
	// One run at a time: a second request waits for the first.
	if p := c.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 409).Problem(t); p["code"] != "readiness_running" {
		t.Errorf("second run: %v", p)
	}
	// Viewers poll too; the queued run shows its groups.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	poll = readiness.ReadinessPoll{}
	aisha.Do("GET", ws+"/campaigns/"+id+"/readiness/progress", nil).MustStatus(t, 200).JSON(t, &poll)
	if poll.Running == nil || len(poll.Running.Groups) < 8 || poll.Running.Groups[0].Status != "queued" {
		t.Fatalf("poll while queued: %+v", poll.Running)
	}
	aisha.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 403)

	if n := readinesstest.Work(t, env); n != 1 {
		t.Fatalf("worked %d runs", n)
	}
	poll = readiness.ReadinessPoll{}
	c.Do("GET", ws+"/campaigns/"+id+"/readiness/progress", nil).MustStatus(t, 200).JSON(t, &poll)
	if poll.Running != nil || poll.Failed != nil || poll.Latest == nil || poll.Latest.ID == before.ID {
		t.Fatalf("after the run: %+v", poll)
	}
	if after := readinesstest.Latest(t, c, "acme-newsletter", id); after.ID != poll.Latest.ID || after.Trigger != "manual" || after.RanBy != "Linh Nguyen" || after.RanByType != "" {
		t.Errorf("latest run: %s %s by %q (%s)", after.ID, after.Trigger, after.RanBy, after.RanByType)
	}
	var actor string
	if err := env.App.Pool.QueryRow(ctx, `SELECT actor_name FROM audit_events WHERE action = 'readiness.run' AND target_id = $1 ORDER BY occurred_at DESC LIMIT 1`, id).Scan(&actor); err != nil || actor != "Linh Nguyen" {
		t.Errorf("the run is audited as who asked for it: %q %v", actor, err)
	}

	// A run that stopped on an error says so until the next run.
	c.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 202).JSON(t, &poll)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE readiness_progress SET failed_at = now(), error = 'Audience: connection reset.' WHERE campaign_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	poll = readiness.ReadinessPoll{}
	c.Do("GET", ws+"/campaigns/"+id+"/readiness/progress", nil).MustStatus(t, 200).JSON(t, &poll)
	if poll.Running != nil || poll.Failed == nil || poll.Failed.Error != "Audience: connection reset." {
		t.Fatalf("failed run: %+v", poll)
	}
	// The failed run's job finds the row isn't its own any more and stops.
	readinesstest.Work(t, env)
	c.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 202)
	readinesstest.Work(t, env)
	poll = readiness.ReadinessPoll{}
	c.Do("GET", ws+"/campaigns/"+id+"/readiness/progress", nil).MustStatus(t, 200).JSON(t, &poll)
	if poll.Running != nil || poll.Failed != nil {
		t.Fatalf("a new run clears the failure: %+v", poll)
	}

	// A run an API key asked for is the key's in the audit log, though a
	// worker runs it: not the person's who made the key.
	var key struct {
		Key    struct{ ID string }
		Secret string
	}
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Readiness bot", "permissions": []string{"campaigns:edit"}}).MustStatus(t, 201).JSON(t, &key)
	bot := env.Client()
	bot.Bearer = key.Secret
	poll = readiness.ReadinessPoll{}
	bot.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 202).JSON(t, &poll)
	// Waiting and done, the run names the key, never Linh, who made it.
	if poll.Running == nil || poll.Running.StartedBy != "Readiness bot" || poll.Running.StartedByType != "api_key" {
		t.Errorf("the key's queued run: %+v", poll.Running)
	}
	if n := readinesstest.Work(t, env); n != 1 {
		t.Fatalf("worked %d runs", n)
	}
	if run := readinesstest.Latest(t, c, "acme-newsletter", id); run.RanBy != "Readiness bot" || run.RanByType != "api_key" {
		t.Errorf("the key's run is by %q (%s), want the key", run.RanBy, run.RanByType)
	}
	var ranBy *string
	if err := env.App.Pool.QueryRow(ctx, `SELECT ran_by::text FROM readiness_runs WHERE campaign_id = $1 ORDER BY ran_at DESC LIMIT 1`, id).Scan(&ranBy); err != nil || ranBy != nil {
		t.Errorf("the key's run names a person: %v %v", ranBy, err)
	}
	var typ, name, actorID string
	if err := env.App.Pool.QueryRow(ctx, `SELECT actor_type, actor_name, coalesce(actor_id::text, '') FROM audit_events
		WHERE action = 'readiness.run' AND target_id = $1 ORDER BY occurred_at DESC LIMIT 1`, id).Scan(&typ, &name, &actorID); err != nil ||
		typ != "api_key" || name != "Readiness bot" || actorID != key.Key.ID {
		t.Errorf("the key's run is audited as %s %q (%s), want the key %s: %v", typ, name, actorID, key.Key.ID, err)
	}
}

// SR-7: a run gathers what the new checks need: variant B of an A/B
// subject test, the recipients without a first name, Gmail Postmaster's
// spam rate and the reverse DNS of Postal's dedicated IP.
func TestReviewGathersTheNewChecks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	id := campaignID(t, c, "Weekly Digest #143")
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET ab_test = '{"kind":"subject","variants":[{"key":"A"},{"key":"B","subject":"LAST CHANCE!!! 50% OFF"}],"test_percent":20,"metric":"human_opens","duration_hours":2}',
		subject = 'Hi {{ first_name }}, what shipped this week' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE postmaster_daily SET spam_rate = 0.0035 WHERE domain = 'news.acme.io' AND day = (SELECT max(day) FROM postmaster_daily WHERE domain = 'news.acme.io')`); err != nil {
		t.Fatal(err)
	}
	if tag, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET first_name = '' WHERE workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')
		AND status = 'active' AND hashtext(email) % 4 = 0`); err != nil || tag.RowsAffected() == 0 {
		t.Fatalf("contacts without a first name: %v", err)
	}
	run := runChecks(t, env, c, id)

	if ch, ok := checkOf(run, "content.spam.variant.B"); !ok || ch.Status != readiness.Warn {
		t.Errorf("variant B: %+v", ch)
	}
	if ch, ok := checkOf(run, "content.personalization"); !ok || ch.Status != readiness.Warn || !strings.Contains(ch.Title, "no first name") {
		t.Errorf("first name without a fallback: %+v", ch)
	}
	if ch, _ := checkOf(run, "compliance.bulk_sender"); ch.Status != readiness.Block || !strings.Contains(ch.Title, "Gmail spam rate 0.35%") {
		t.Errorf("Postmaster spam rate: %+v", ch)
	}
	if ch, ok := checkOf(run, "content.size"); !ok || !strings.Contains(ch.Title, "with tracking links") || !strings.Contains(ch.Detail, "Spanish") {
		t.Errorf("size with tracking: %+v", ch)
	}
	if _, ok := checkOf(run, "reputation.rdns.203.0.113.25"); ok {
		t.Errorf("Postal's IP has reverse DNS in the demo zone")
	}
	if ch, ok := checkOf(run, "reputation.ip_blocklists.203.0.113.25"); ok && ch.Status == readiness.Pass && !strings.Contains(ch.Detail, "mta.acme.io") {
		t.Errorf("reverse DNS named: %+v", ch)
	}
}

// SR-5: a run's estimates come from the sending engine: its rates (SES's
// published maximum, Postal's warm-up ramp), the A/B test's sample and
// wait, and how long the whole send takes from its start.
func TestEstimatesComeFromTheEngine(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	run := runChecks(t, env, c, campaignID(t, c, "Weekly Digest #143"))
	r := run.Estimates.Route
	if r.Timeline == nil || len(r.Timeline.Basis) == 0 || r.Timeline.BasisLabel == "" || !r.Timeline.End.After(r.Timeline.Start) {
		t.Fatalf("timeline: %+v", r.Timeline)
	}
	if want := int(r.Timeline.End.Sub(r.Timeline.Start).Minutes()); r.TotalMinutes < want || r.TotalMinutes > want+1 {
		t.Errorf("total %d minutes, the timeline says %d", r.TotalMinutes, want)
	}
	for _, s := range r.Shares {
		if s.Recipients > 0 && s.Available && (s.RateFrom == "" || s.Rate <= 0) {
			t.Errorf("share without the engine's rate: %+v", s)
		}
	}
}
