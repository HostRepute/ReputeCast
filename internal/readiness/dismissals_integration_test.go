package readiness_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// CO-14 on Review & send: a warning dismissed in the composer is listed
// with who dismissed it, why and their note, in the reader's language; the
// send doesn't ask to confirm it; Undo makes it a warning to confirm again.
func TestComposerDismissalsOnReview(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var cp composerContent
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Short links", "fromCampaignId": campaignID(t, c, "Weekly Digest #143")}).MustStatus(t, 201).JSON(t, &cp)
	html := `<html><body><table><tr><td><p>Hello there</p><a href="https://bit.ly/x">Read the post</a>
		<p>Acme Inc · 500 Market St, San Francisco, CA 94105</p><a href="{{ unsubscribe_url }}">Unsubscribe</a></td></tr></table></body></html>`
	c.Do("PUT", ws+"/campaigns/"+cp.ID+"/content", map[string]any{"baseVersion": cp.Version, "subject": "The October changelog", "html": html, "source": "manual"}).
		MustStatus(t, 200)
	var saved composerContent
	c.Do("GET", ws+"/campaigns/"+cp.ID, nil).MustStatus(t, 200).JSON(t, &saved)

	// The composer's checks of the campaign as stored, and its keys.
	var checks struct {
		Issues []struct {
			ID, Severity, Title string
		}
		IssueKeys []string
	}
	c.Do("POST", ws+"/campaigns/"+cp.ID+"/checks", map[string]any{"subject": saved.Subject, "html": saved.HTML}).MustStatus(t, 200).JSON(t, &checks)
	key, title := "", ""
	for i, is := range checks.Issues {
		if is.ID == "links.shortener" && i < len(checks.IssueKeys) {
			key, title = checks.IssueKeys[i], is.Title
		}
	}
	if key == "" {
		t.Fatalf("the composer shows no shortener warning: %+v", checks.Issues)
	}
	var d struct{ ID string }
	c.Do("POST", ws+"/campaigns/"+cp.ID+"/dismissals", map[string]any{"key": key, "checkId": "links.shortener", "title": title, "reason": "wrong",
		"note": "Our own short domain"}).MustStatus(t, 201).JSON(t, &d)

	runChecks(t, env, c, cp.ID)
	shortener := func(cl *apptest.Client) readiness.ReadinessCheck {
		t.Helper()
		var r struct{ Run readiness.ReadinessRun }
		cl.Do("GET", ws+"/campaigns/"+cp.ID+"/review", nil).MustStatus(t, 200).JSON(t, &r)
		return find(r.Run, "links.shortener")
	}
	got := shortener(c)
	if got.Status != readiness.Warn || got.Dismissal == nil || got.Dismissal.ID != d.ID || got.Dismissal.By != "Linh Nguyen" ||
		got.Dismissal.ReasonText != "the check is wrong here" || got.Dismissal.Note != "Our own short domain" {
		t.Fatalf("dismissed on Review & send: %+v", got)
	}
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	if v := shortener(vi); v.Dismissal == nil || v.Dismissal.ReasonText == got.Dismissal.ReasonText {
		t.Errorf("in Vietnamese: %+v", v.Dismissal)
	}

	// Sending doesn't ask to confirm it: whatever else stops this send,
	// the shortener isn't among the warnings to confirm.
	r := c.Do("POST", ws+"/campaigns/"+cp.ID+"/send", map[string]any{"acknowledged": []string{}})
	if body := string(r.Body); strings.Contains(body, "shortener") || strings.Contains(body, "bit.ly") {
		t.Errorf("asked to confirm a dismissed warning: %d %s", r.Status, r.Body)
	}

	// Undo: a warning to confirm again.
	c.Do("DELETE", ws+"/campaigns/"+cp.ID+"/dismissals/"+d.ID, nil).MustStatus(t, 204)
	if got := shortener(c); got.Dismissal != nil {
		t.Fatalf("after Undo: %+v", got.Dismissal)
	}
}
