package readiness_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// runChecks runs Send Readiness on a campaign and returns the stored run.
func runChecks(t *testing.T, c *apptest.Client, id string) readiness.ReadinessRun {
	t.Helper()
	var run readiness.ReadinessRun
	c.Do("POST", ws+"/campaigns/"+id+"/readiness", nil).MustStatus(t, 200).JSON(t, &run)
	return run
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
	run := runChecks(t, c, id)
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
	run := runChecks(t, c, campaignID(t, c, "Weekly Digest #143"))
	ch, _ := checkOf(run, "reputation.domain_blocklists")
	if ch.Status != readiness.Block || !strings.Contains(ch.Title, "acme.io, the parent domain of news.acme.io") {
		t.Fatalf("parent listing: %s %q", ch.Status, ch.Title)
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
