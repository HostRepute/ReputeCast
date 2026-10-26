package reports_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// TestReportGrowth: an issue's growth (review AN-12) and its views on the
// web (AN-13). New subscribers by how they came: from a view-online page
// of one of the issue's messages or a subscribe link tagged as a forward,
// a referral whose referrer last clicked this issue, the issue's archive
// page; and the archive page's views, counted on the server without
// cookies, once per visitor and day.
func TestReportGrowth(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "Growth issue")
	ids := f.HandedOver(sendtest.ContactSpec{N: 3, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	var started time.Time
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT started_at FROM campaigns WHERE id = $1`, f.Campaign).Scan(&started); err != nil {
		t.Fatal(err)
	}
	f.Exec(`UPDATE campaigns SET status = 'sent', completed_at = started_at + interval '5 minutes' WHERE id = $1`, f.Campaign)
	// On the public archive, with a list the archive shows.
	var archived uuid.UUID
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT list_id FROM archive_issues WHERE workspace_id = $1 AND NOT hidden AND list_id IS NOT NULL LIMIT 1`, f.WS).Scan(&archived); err != nil {
		t.Fatal(err)
	}
	f.Exec(`INSERT INTO archive_issues (campaign_id, workspace_id, slug, list_id, title) VALUES ($1, $2, 'growth-issue', $3, 'Growth issue')`, f.Campaign, f.WS, archived)
	host := "news.example.org/p/acme-newsletter"
	join := func(page, referrer, medium, campaign string, at time.Time) {
		t.Helper()
		f.Exec(`INSERT INTO contact_growth (contact_id, workspace_id, source_kind, joined_at, page, referrer, utm_medium, utm_campaign, captured)
			VALUES ($1, $2, 'subscribe_page', $3, $4, $5, $6, $7, true)`, uuid.Must(uuid.NewV7()), f.WS, at, page, referrer, medium, campaign)
	}
	at := started.Add(time.Hour)
	// The web post: two signups on the archive page, one from its address.
	join(host+"/subscribe", host+"/archive/growth-issue", "", "", at)
	join(host+"/archive/growth-issue", "", "", "", at)
	// Forwards: a forwarded copy read in the browser, a tagged link.
	var contact uuid.UUID
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT contact_id FROM messages WHERE id = $1`, ids[0]).Scan(&contact); err != nil {
		t.Fatal(err)
	}
	view := f.Env.App.Links.Sign(linktoken.Claims{Kind: linktoken.ViewOnline, WorkspaceID: f.WS, MessageID: ids[0], ContactID: contact})
	join(host+"/subscribe", host+"/view/"+view, "", "", at)
	join(host+"/subscribe", "", "forward", f.Campaign.String(), at)
	// Not this issue's: another campaign's forward, a page elsewhere.
	join(host+"/subscribe", "", "forward", uuid.Must(uuid.NewV7()).String(), at)
	join(host+"/subscribe", "blog.example.org/post", "", "", at)
	// Before the issue went out: not counted.
	join(host+"/archive/growth-issue", "", "", "", started.Add(-time.Hour))

	// Referrals: the referrer's last click before the signup was this
	// issue; another referrer clicked a later email after this one.
	program := uuid.Must(uuid.NewV7())
	f.Exec(`INSERT INTO referral_programs (id, workspace_id, list_id) VALUES ($1, $2, $3)`, program, f.WS, f.List)
	clicked := started.Add(2 * time.Hour)
	f.Exec(`UPDATE messages SET clicked_at = $2 WHERE id = ANY($1)`, []uuid.UUID{ids[1], ids[2]}, clicked)
	referrer := func(id uuid.UUID) uuid.UUID {
		var c uuid.UUID
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT contact_id FROM messages WHERE id = $1`, id).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	for i, id := range []uuid.UUID{ids[1], ids[2]} {
		f.Exec(`INSERT INTO referrals (id, workspace_id, program_id, referrer_id, seq, status, signed_up_at) VALUES ($1, $2, $3, $4, $5, 'counted', $6)`,
			uuid.Must(uuid.NewV7()), f.WS, program, referrer(id), i+1, clicked.Add(time.Hour))
	}
	// The second referrer clicked another campaign's email afterwards: that
	// one gets the credit.
	other := uuid.Must(uuid.NewV7())
	f.Exec(`INSERT INTO campaigns (id, workspace_id, name, status, subject) VALUES ($1, $2, 'Later issue', 'sent', 'Later')`, other, f.WS)
	f.Exec(`INSERT INTO messages (id, workspace_id, stream, campaign_id, contact_id, email, status, queued_at, sent_at, clicked_at)
		SELECT $1, workspace_id, stream, $2, contact_id, email, 'delivered', $3, $3, $4 FROM messages WHERE id = $5`,
		uuid.Must(uuid.NewV7()), other, clicked, clicked.Add(30*time.Minute), ids[2])

	// Views of the archive page: once per visitor and day, from a salted
	// hash of the address and browser (no cookie); crawlers don't count.
	view1 := func(ua string) {
		t.Helper()
		req, err := http.NewRequestWithContext(f.Ctx, "GET", f.Env.Server.URL+"/p/acme-newsletter/archive/growth-issue", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", ua)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("the issue page: %d", res.StatusCode)
		}
	}
	view1("Mozilla/5.0 (Macintosh) Firefox/131.0")
	view1("Mozilla/5.0 (Macintosh) Firefox/131.0")
	view1("Mozilla/5.0 (iPhone) Safari/604.1")
	view1("Googlebot/2.1")

	c := f.Env.Client()
	c.SignInDemo("linh@acme.io")
	var r struct {
		Growth struct {
			NewSubscribers, Forwards, Referrals, WebPost, WebViews int64
		}
	}
	c.Do("GET", fmt.Sprintf("/workspaces/acme-newsletter/campaigns/%s/report", f.Campaign), nil).MustStatus(t, 200).JSON(t, &r)
	g := r.Growth
	if g.WebPost != 2 || g.Forwards != 2 || g.Referrals != 1 || g.NewSubscribers != 5 {
		t.Fatalf("growth: %+v, want 2 from the web post, 2 forwards, 1 referral", g)
	}
	if g.WebViews != 2 {
		t.Fatalf("web views: %d, want 2 (two visitors; the repeat and the crawler not counted)", g.WebViews)
	}
}
