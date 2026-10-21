package sponsors_test

import (
	"context"
	"html"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/sponsors"
)

// sponsorRun is the part of a Send Readiness run the Sponsor block checks read.
type sponsorRun struct {
	Checks []struct {
		Key, Status, Title string
		Fixes              []struct{ Kind, Action, Href string }
	}
}

// runReadiness runs Send Readiness now, waiting out a run in progress.
func runReadiness(t *testing.T, c *apptest.Client, id string) sponsorRun {
	t.Helper()
	for range 20 {
		res := c.Do("POST", ws+"/campaigns/"+id+"/readiness", nil)
		if res.Status == 409 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var r sponsorRun
		res.MustStatus(t, 200).JSON(t, &r)
		return r
	}
	t.Fatal("Send Readiness kept running")
	return sponsorRun{}
}

// applyFix applies a Review & send fix and returns the run after it.
func applyFix(t *testing.T, c *apptest.Client, id, action string) sponsorRun {
	t.Helper()
	var out struct{ Run *sponsorRun }
	c.Do("POST", ws+"/campaigns/"+id+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": action}}}).MustStatus(t, 200).JSON(t, &out)
	if out.Run != nil {
		return *out.Run
	}
	return runReadiness(t, c, id)
}

func (r sponsorRun) check(key string) (status, title string, actions []string) {
	for _, c := range r.Checks {
		if c.Key == key {
			for _, f := range c.Fixes {
				actions = append(actions, f.Kind+":"+f.Action+f.Href)
			}
			return c.Status, c.Title, actions
		}
	}
	return "", "", nil
}

// The board's story (ComposerEarnBlocks): Weekly Digest #143 gets a Sponsor
// block while Northwind Analytics' creative is still waiting. The
// composer's checks and Review & send hold the issue; once the creative is
// ready, Review & send updates the block; with the booking canceled the
// block is left out of the email.
func TestSponsorBlockInComposerAndReview(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var id string
	if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM campaigns WHERE name = 'Weekly Digest #143'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	var c struct {
		MJML, Subject, Preheader string
		Version                  int
	}
	linh.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &c)
	var blk sponsors.SponsorBlock
	linh.Do("GET", ws+"/campaigns/"+id+"/sponsor-block", nil).MustStatus(t, 200).JSON(t, &blk)
	if blk.State != sponsors.BlockWaiting || blk.Issue != 143 || blk.Sponsor != "Northwind Analytics" || blk.BookingID == "" {
		t.Fatalf("block: %+v", blk)
	}
	// The composer puts the block after the first section, with what filled it.
	// Shown to free subscribers only, as the writer set its Visibility.
	rule := `{"id":"spons1","show":"only","match":"all","conditions":[{"kind":"paid","op":"free"}]}`
	block := "<!-- rc:sponsor -->" + strings.Replace(blk.MJML, `rc-sponsor="`,
		`rc-name="Sponsor" rc-visible="`+html.EscapeString(rule)+`" rc-sponsor-hash="`+blk.Hash+`" rc-sponsor="`, 1)
	body := strings.Index(c.MJML, "<mj-body")
	first := strings.Index(c.MJML[body:], "</mj-section>")
	if body < 0 || first < 0 {
		t.Fatalf("no first section in #143:\n%s", c.MJML)
	}
	at := body + first + len("</mj-section>")
	src := c.MJML[:at] + block + c.MJML[at:]
	var saved struct{ Version int }
	linh.Do("PUT", ws+"/campaigns/"+id+"/content", map[string]any{"baseVersion": c.Version, "subject": c.Subject, "preheader": c.Preheader, "mjml": src}).
		MustStatus(t, 200).JSON(t, &saved)

	// The composer's checks block the issue, with the booking and the block.
	var chk struct {
		Verdict string
		Issues  []struct {
			ID, Severity, Title string
			Path                []int
			Fixes               []struct {
				Kind, Value string
				Path        []int
			}
		}
	}
	linh.Do("POST", ws+"/campaigns/"+id+"/checks", map[string]any{"subject": c.Subject, "mjml": src}).MustStatus(t, 200).JSON(t, &chk)
	found := false
	for _, is := range chk.Issues {
		if is.ID != "content.sponsor_creative" {
			continue
		}
		found = true
		if is.Severity != "block" || is.Title != "Sponsor block: Northwind Analytics’ creative is missing" || len(is.Fixes) != 2 ||
			is.Fixes[0].Kind != "open" || is.Fixes[0].Value != "/w/acme-newsletter/earn/sponsors?booking="+blk.BookingID ||
			is.Fixes[1].Kind != "remove_section" || len(is.Path) == 0 || is.Fixes[1].Path[0] != is.Path[0] {
			t.Errorf("composer issue: %+v", is)
		}
	}
	if !found || chk.Verdict != "blocked" {
		t.Fatalf("composer checks: %+v", chk)
	}

	// Review & send says the same, and that sponsored content is labelled.
	run := runReadiness(t, linh, id)
	if st, title, fixes := run.check("content.sponsor.creative"); st != "block" || !strings.Contains(title, "Northwind Analytics") ||
		strings.Join(fixes, " ") != "link:/w/acme-newsletter/earn/sponsors?booking="+blk.BookingID+" apply:sponsor_remove_block" {
		t.Errorf("readiness, waiting: %s %s %v", st, title, fixes)
	}
	if st, _, _ := run.check("content.sponsor.label"); st != "pass" {
		t.Errorf("label: %s", st)
	}

	// The creative arrives: the email still holds the placeholder until
	// Review & send updates the block.
	linh.Do("PUT", ws+"/sponsor-bookings/"+blk.BookingID+"/creative", map[string]any{"headline": "Find the dashboards nobody opens",
		"text": "Northwind shows which reports your team actually uses.", "button": "Try Northwind free", "link": "https://northwind-analytics.example/trial",
		"utm": map[string]any{"source": "acme-weekly-digest", "medium": "email", "campaign": "143"}}).MustStatus(t, 200)
	linh.Do("POST", ws+"/sponsor-bookings/"+blk.BookingID+"/creative/ready", nil).MustStatus(t, 200)
	run = runReadiness(t, linh, id)
	if st, _, fixes := run.check("content.sponsor.stale"); st != "block" || !strings.Contains(strings.Join(fixes, " "), "apply:sponsor_refresh_block") {
		t.Fatalf("readiness, stale: %s %v", st, fixes)
	}
	run = applyFix(t, linh, id, "sponsor_refresh_block")
	for _, ch := range run.Checks {
		if strings.HasPrefix(ch.Key, "content.sponsor") && ch.Status != "pass" {
			t.Errorf("after the update: %+v", ch)
		}
	}
	var after struct{ MJML, HTML string }
	linh.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &after)
	if !strings.Contains(after.MJML, "Find the dashboards nobody opens") || !strings.Contains(after.MJML, "utm_campaign=143") || !strings.Contains(after.MJML, "Sponsored by Northwind Analytics") ||
		!strings.Contains(after.MJML, `rc-name="Sponsor"`) || !strings.Contains(after.MJML, "spons1") {
		t.Errorf("updated block:\n%s", after.MJML)
	}
	chk.Issues = nil
	linh.Do("POST", ws+"/campaigns/"+id+"/checks", map[string]any{"subject": c.Subject, "mjml": after.MJML}).MustStatus(t, 200).JSON(t, &chk)
	for _, is := range chk.Issues {
		if is.ID == "content.sponsor_creative" {
			t.Errorf("the ready creative still blocks: %+v", is)
		}
	}

	// Canceled: the block says why in the composer and the email leaves it out.
	linh.Do("POST", ws+"/sponsor-bookings/"+blk.BookingID+"/cancel", map[string]any{"reason": "Moved to #146"}).MustStatus(t, 204)
	var gone sponsors.SponsorBlock
	linh.Do("GET", ws+"/campaigns/"+id+"/sponsor-block", nil).MustStatus(t, 200).JSON(t, &gone)
	if gone.State != sponsors.BlockNone || gone.Issue != 143 || !strings.Contains(gone.MJML, `rc-omit="1"`) || !strings.Contains(gone.MJML, "No sponsor is booked for #143") {
		t.Fatalf("canceled block: %+v", gone)
	}
	run = runReadiness(t, linh, id)
	if st, _, _ := run.check("content.sponsor.stale"); st != "block" {
		t.Errorf("readiness after the cancel: %s", st)
	}
	run = applyFix(t, linh, id, "sponsor_refresh_block")
	if st, title, _ := run.check("content.sponsor.none"); st != "pass" || !strings.Contains(title, "#143") {
		t.Errorf("left out: %s %s", st, title)
	}
	after.MJML, after.HTML = "", ""
	linh.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &after)
	if !strings.Contains(after.MJML, "No sponsor is booked for #143") || strings.Contains(after.HTML, "No sponsor is booked") || strings.Contains(after.HTML, "Find the dashboards") {
		t.Errorf("the email still shows the block: %s", after.HTML)
	}

	// Booked again: the draft, still without a date, fills its block from
	// the issue's new booking.
	var northwind string
	if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM sponsors WHERE name = 'Northwind Analytics'`).Scan(&northwind); err != nil {
		t.Fatal(err)
	}
	var nb sponsors.SponsorBookingDetail
	linh.Do("POST", ws+"/sponsor-slots/"+gone.SlotID+"/issues/143/book", map[string]any{"sponsorId": northwind, "status": "booked"}).MustStatus(t, 201).JSON(t, &nb)
	var again sponsors.SponsorBlock
	linh.Do("GET", ws+"/campaigns/"+id+"/sponsor-block", nil).MustStatus(t, 200).JSON(t, &again)
	if again.State != sponsors.BlockWaiting || again.BookingID != nb.ID || again.Issue != 143 {
		t.Errorf("booked again: %+v", again)
	}
}
