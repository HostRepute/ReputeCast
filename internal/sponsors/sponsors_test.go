package sponsors_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/sponsors"
	"github.com/hostrepute/reputecast/internal/store"
)

const ws = "/workspaces/acme-newsletter"

func id(t *testing.T, v string) uuid.UUID {
	t.Helper()
	u, err := uuid.Parse(v)
	if err != nil {
		t.Fatalf("not an id: %q", v)
	}
	return u
}

// testSlot adds a sponsor and a weekly slot on Product Updates whose first
// issue is next week (the demo's own sponsors stay as they are).
func testSlot(t *testing.T, env *apptest.Env, c *apptest.Client) (sponsor, slot string, first time.Time) {
	t.Helper()
	ctx := context.Background()
	var list uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM lists WHERE name = 'Product Updates'`).Scan(&list); err != nil {
		t.Fatal(err)
	}
	var sp sponsors.SponsorItem
	c.Do("POST", ws+"/sponsors", map[string]any{"name": "Fabrikam Tools X17", "website": "https://fabrikam-x17.example/", "contactName": "Ana Ruiz",
		"contactEmail": "ana@fabrikam-x17.example", "termsDays": 10}).MustStatus(t, 201).JSON(t, &sp)
	if sp.Website != "fabrikam-x17.example" || sp.Initials != "FT" || sp.TermsDays != 10 {
		t.Fatalf("sponsor: %+v", sp)
	}
	first = time.Now().In(time.UTC).AddDate(0, 0, 8)
	var sl sponsors.SponsorSlotItem
	c.Do("POST", ws+"/sponsor-slots", map[string]any{"listId": list.String(), "name": "Top banner X17", "placement": "Before the first story", "priceCents": 50000,
		"firstNumber": 12, "firstOn": first.Format("2006-01-02"), "everyDays": 7, "creativeDays": 3}).MustStatus(t, 201).JSON(t, &sl)
	if sl.FirstNumber != 12 || sl.EveryDays != 7 || sl.PriceCents != 50000 || sl.ListName != "Product Updates" {
		t.Fatalf("slot: %+v", sl)
	}
	return sp.ID, sl.ID, first
}

// readerPage gets a reader-facing page (outside /api/v1) without signing in.
func readerPage(t *testing.T, env *apptest.Env, path string) (int, http.Header, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, env.Server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := env.Server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, string(b)
}

func auditCount(t *testing.T, env *apptest.Env, action, target string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND ($2 = '' OR target_id = $2)`, action, target).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSponsorBookingFlow(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("sofia@acme.io") // Editor: sponsors:manage
	sponsor, slot, first := testSlot(t, env, c)

	// Book issue #13 (a week after the first).
	var b sponsors.SponsorBookingDetail
	c.Do("POST", fmt.Sprintf("%s/sponsor-slots/%s/issues/13/book", ws, slot), map[string]any{"sponsorId": sponsor, "status": "booked"}).MustStatus(t, 201).JSON(t, &b)
	issueOn := first.AddDate(0, 0, 7)
	if b.Status != "booked" || b.Issue != 13 || b.PriceCents != 50000 || b.CreativeSt != "waiting" {
		t.Fatalf("booking: %+v", b)
	}
	if got := b.IssueOn.Format("2006-01-02"); got != issueOn.Format("2006-01-02") {
		t.Errorf("issue date %s, want %s", got, issueOn.Format("2006-01-02"))
	}
	if b.CreativeDue == nil || b.CreativeDue.Format("2006-01-02") != issueOn.AddDate(0, 0, -3).Format("2006-01-02") {
		t.Errorf("creative due %v", b.CreativeDue)
	}
	if b.Payment.Due == nil || b.Payment.Due.Format("2006-01-02") != issueOn.AddDate(0, 0, 10).Format("2006-01-02") || b.Payment.Status != "unpaid" {
		t.Errorf("payment %+v", b.Payment)
	}
	if b.Creative.UTM.Source != "acme-product-updates" || b.Creative.UTM.Campaign != "13" || b.Creative.Link != "https://fabrikam-x17.example" {
		t.Errorf("creative defaults: %+v", b.Creative)
	}
	if auditCount(t, env, "sponsors.booking_created", b.ID) != 1 {
		t.Error("booking not audited")
	}
	// One booking per issue.
	p := c.Do("POST", fmt.Sprintf("%s/sponsor-slots/%s/issues/13/book", ws, slot), map[string]any{"sponsorId": sponsor, "status": "hold", "holdUntil": time.Now().AddDate(0, 0, 2).Format("2006-01-02")}).
		MustStatus(t, 409).Problem(t)
	if !strings.Contains(fmt.Sprint(p["detail"]), "#13 is already booked") {
		t.Errorf("second booking: %v", p["detail"])
	}

	// The creative can't be ready until it has a headline, text and link.
	p = c.Do("POST", ws+"/sponsor-bookings/"+b.ID+"/creative/ready", nil).MustStatus(t, 422).Problem(t)
	if !strings.Contains(fmt.Sprint(p["detail"]), "headline, text and link") {
		t.Errorf("incomplete creative: %v", p)
	}
	c.Do("PUT", ws+"/sponsor-bookings/"+b.ID+"/creative", map[string]any{"headline": "Tools that last", "text": strings.Repeat("x", 281), "button": "Shop", "link": "fabrikam-x17.example/shop",
		"utm": map[string]any{"source": "acme-product-updates", "medium": "email", "campaign": "13"}}).MustStatus(t, 422)
	c.Do("PUT", ws+"/sponsor-bookings/"+b.ID+"/creative", map[string]any{"headline": "Tools that last", "text": "Built for a lifetime of fixes.", "button": "Shop", "link": "fabrikam-x17.example/shop",
		"utm": map[string]any{"source": "acme-product-updates", "medium": "email", "campaign": "13"}}).MustStatus(t, 200).JSON(t, &b)
	if b.Creative.Link != "https://fabrikam-x17.example/shop" {
		t.Errorf("link: %s", b.Creative.Link)
	}
	c.Do("POST", ws+"/sponsor-bookings/"+b.ID+"/creative/ready", nil).MustStatus(t, 200).JSON(t, &b)
	if b.CreativeSt != "ready" || b.ReadyAt == nil {
		t.Errorf("creative not ready: %+v", b)
	}
	if auditCount(t, env, "sponsors.creative_ready", b.ID) != 1 {
		t.Error("ready not audited")
	}

	// A Viewer reads, never changes; recording a payment needs a person.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/sponsor-bookings/"+b.ID, nil).MustStatus(t, 200)
	v.Do("PUT", ws+"/sponsor-bookings/"+b.ID+"/payment", map[string]any{"paid": true}).MustStatus(t, 403)
	v.Do("POST", ws+"/sponsors", map[string]any{"name": "Nope"}).MustStatus(t, 403)

	c.Do("PUT", ws+"/sponsor-bookings/"+b.ID+"/payment", map[string]any{"paid": true, "paidOn": time.Now().AddDate(0, 0, 3).Format("2006-01-02")}).MustStatus(t, 422)
	c.Do("PUT", ws+"/sponsor-bookings/"+b.ID+"/payment", map[string]any{"paid": true, "amountCents": 45000, "note": "Card, ref 77"}).MustStatus(t, 200).JSON(t, &b)
	if b.Payment.Status != "paid" || b.Payment.PaidCents == nil || *b.Payment.PaidCents != 45000 || b.Payment.Note != "Card, ref 77" || b.Payment.RecordedBy != "Sofia Rossi" {
		t.Errorf("payment: %+v", b.Payment)
	}
	if auditCount(t, env, "sponsors.payment_recorded", b.ID) != 1 {
		t.Error("payment not audited")
	}

	// Hold the next issue, then cancel the booking: the issue opens again.
	var h sponsors.SponsorBookingDetail
	c.Do("POST", fmt.Sprintf("%s/sponsor-slots/%s/issues/14/book", ws, slot), map[string]any{"sponsorId": sponsor, "status": "hold",
		"holdUntil": first.AddDate(0, 0, 30).Format("2006-01-02")}).MustStatus(t, 422)
	c.Do("POST", fmt.Sprintf("%s/sponsor-slots/%s/issues/14/book", ws, slot), map[string]any{"sponsorId": sponsor, "status": "hold",
		"holdUntil": time.Now().AddDate(0, 0, 4).Format("2006-01-02")}).MustStatus(t, 201).JSON(t, &h)
	if h.Status != "hold" || h.HoldUntil == nil {
		t.Errorf("hold: %+v", h)
	}
	c.Do("POST", ws+"/sponsor-bookings/"+h.ID+"/cancel", map[string]any{"reason": "Moved their launch"}).MustStatus(t, 204)
	var issue sponsors.SponsorIssueDetail
	c.Do("GET", fmt.Sprintf("%s/sponsor-slots/%s/issues/14", ws, slot), nil).MustStatus(t, 200).JSON(t, &issue)
	if issue.BookingID != "" {
		t.Errorf("the canceled hold still holds #14: %+v", issue)
	}
	if auditCount(t, env, "sponsors.booking_canceled", h.ID) != 1 {
		t.Error("cancel not audited")
	}

	// The overview of the slot: #12 open, #13 booked.
	var ov sponsors.SponsorOverview
	c.Do("GET", ws+"/sponsors/overview?slot="+slot, nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.SlotID != slot || !ov.CanManage {
		t.Fatalf("overview: slot %s, canManage %v", ov.SlotID, ov.CanManage)
	}
	var found13 bool
	for _, r := range ov.Upcoming {
		if r.Number == 13 && r.Booking != nil && r.Booking.Status == "booked" && r.Booking.Sponsor != nil && r.Booking.Sponsor.Name == "Fabrikam Tools X17" {
			found13 = true
		}
		if r.Number == 12 && r.Booking != nil {
			t.Errorf("#12 should be open: %+v", r.Booking)
		}
	}
	if !found13 {
		t.Errorf("#13 not in the upcoming issues: %+v", ov.Upcoming)
	}
	var vo sponsors.SponsorOverview
	v.Do("GET", ws+"/sponsors/overview", nil).MustStatus(t, 200).JSON(t, &vo)
	if vo.CanManage || len(vo.Helpers) == 0 || vo.Helpers[0] != "Sofia Rossi" {
		t.Errorf("viewer's helpers: %v", vo.Helpers)
	}
}

func TestSponsorReportCountsOnlyBlockRecipients(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pool := env.App.Pool
	var wsID, list uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT w.id, l.id FROM workspaces w JOIN lists l ON l.workspace_id = w.id WHERE w.slug = 'acme-newsletter' AND l.name = 'Product Updates'`).Scan(&wsID, &list); err != nil {
		t.Fatal(err)
	}
	// A campaign sent four days ago whose Sponsor block shows to some
	// recipients only (Visibility rule sp1x17).
	camp := uuid.Must(uuid.NewV7())
	sent := time.Now().Add(-80 * time.Hour)
	mjml := `<mjml><mj-body><mj-section><mj-column><mj-text>Story</mj-text></mj-column></mj-section>` +
		`<mj-section rc-sponsor="1" rc-visible='{"id":"sp1x17","show":"only","match":"all","conditions":[]}'><mj-column><mj-text>Ad</mj-text></mj-column></mj-section></mj-body></mjml>`
	if _, err := pool.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, audience, mjml, started_at, completed_at) VALUES ($1, $2, 'Report test X17', 'sent', $3, $4, $5, $5)`,
		camp, wsID, fmt.Sprintf(`{"lists":["%s"]}`, list), mjml, sent); err != nil {
		t.Fatal(err)
	}
	link := "https://fabrikam-x17.example/shop?utm_campaign=9&utm_medium=email&utm_source=acme-product-updates"
	for i, l := range []struct {
		url, label string
		clicks     int
	}{{link, "Shop now", 5}, {link, "", 2}, {"https://acme.io/blog", "Read more", 9}} {
		if _, err := pool.Exec(ctx, `INSERT INTO campaign_links (id, campaign_id, url, label, position, clicks) VALUES ($1, $2, $3, $4, $5, $6)`,
			uuid.Must(uuid.NewV7()), camp, l.url, l.label, i, l.clicks); err != nil {
			t.Fatal(err)
		}
	}
	msgs := make([]uuid.UUID, 4)
	for i := range msgs {
		msgs[i] = uuid.Must(uuid.NewV7())
		hidden := ""
		if i == 3 {
			hidden = "abc123,sp1x17" // this recipient's version left the block out
		}
		var opened *time.Time
		if i != 2 {
			at := sent.Add(time.Hour)
			opened = &at
		}
		if _, err := pool.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, campaign_id, email, status, queued_at, sent_at, delivered_at, human_opened_at, hidden_blocks)
			VALUES ($1, $2, 'broadcast', $3, $4, 'delivered', $5, $5, $5, $6, $7)`, msgs[i], wsID, camp, fmt.Sprintf("r%d@x17.example", i), sent, opened, hidden); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range [][2]int{{0, 0}, {0, 1}, {1, 1}, {3, 0}, {2, 2}} {
		if _, err := pool.Exec(ctx, `INSERT INTO tracking_link_clicks (message_id, link, clicked_at) VALUES ($1, $2, $3)`, msgs[c[0]], c[1], sent.Add(2*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	c := env.Client()
	c.SignInDemo("sofia@acme.io")
	sponsor, slot, _ := testSlot(t, env, c)
	// The booking of an issue that went out with that campaign.
	bid := uuid.Must(uuid.NewV7())
	cr := `{"headline":"Tools that last","text":"Built to last.","button":"Shop now","link":"https://fabrikam-x17.example/shop","utm":{"source":"acme-product-updates","medium":"email","campaign":"9"}}`
	if _, err := pool.Exec(ctx, `INSERT INTO sponsor_bookings (id, workspace_id, slot_id, issue_number, issue_on, sponsor_id, campaign_id, price_cents, currency, status, creative, creative_status, sent_at)
		VALUES ($1, $2, $3, 9, $4, $5, $6, 50000, 'usd', 'sent', $7, 'ready', $8)`, bid, wsID, id(t, slot), sent, id(t, sponsor), camp, cr, sent); err != nil {
		t.Fatal(err)
	}
	var r sponsors.SponsorReport
	c.Do("GET", ws+"/sponsor-bookings/"+bid.String()+"/report", nil).MustStatus(t, 200).JSON(t, &r)
	// Delivered: the three recipients who got the block; their human opens:
	// two; unique clickers on the sponsor's links: two (the hidden one's
	// click doesn't count); clicks: the sponsor's two links.
	if r.Delivered != 3 || r.HumanOpens != 2 || r.UniqueClickers != 2 || r.Clicks != 7 || !r.Final {
		t.Fatalf("report: delivered %d, opens %d, clickers %d, clicks %d, final %v", r.Delivered, r.HumanOpens, r.UniqueClickers, r.Clicks, r.Final)
	}
	if len(r.Links) != 2 || r.Links[0].Kind != "button" || r.Links[0].Clicks != 5 || r.Links[1].Kind != "logo" {
		t.Errorf("links: %+v", r.Links)
	}
	if r.LinkHost != "fabrikam-x17.example/shop" || r.LinkCampaign != "9" {
		t.Errorf("link line: %s %s", r.LinkHost, r.LinkCampaign)
	}
	// The campaign's report shows the booking in its Sponsor section, and
	// the Campaigns list's chip names the sponsor.
	var cs sponsors.CampaignSponsor
	c.Do("GET", ws+"/campaigns/"+camp.String()+"/sponsor", nil).MustStatus(t, 200).JSON(t, &cs)
	if cs.Booking == nil || cs.Booking.ID != bid.String() || cs.Booking.Report == nil || cs.Booking.Report.Delivered != 3 {
		t.Fatalf("campaign sponsor: %+v", cs.Booking)
	}
	chip := func(cl *apptest.Client) string {
		var chips sponsors.CampaignChips
		cl.Do("GET", ws+"/sponsors/campaign-chips", nil).MustStatus(t, 200).JSON(t, &chips)
		for _, ch := range chips.Items {
			if ch.CampaignID == camp.String() {
				return ch.Sponsor
			}
		}
		return ""
	}
	if chip(c) == "" {
		t.Error("no chip for the sponsored campaign")
	}

	// Share it: the link works without signing in, shows no contact data,
	// and stops working once revoked.
	var sh sponsors.SponsorShareLink
	c.Do("POST", ws+"/sponsor-bookings/"+bid.String()+"/shares", map[string]any{"expiresInDays": 90}).MustStatus(t, 201).JSON(t, &sh)
	token := sh.URL[strings.LastIndex(sh.URL, "/")+1:]
	got, err := env.App.Sponsors().Shared(ctx, token)
	if err != nil || len(got.Reports) != 1 || got.Reports[0].Delivered != 3 {
		t.Fatalf("shared: %v %+v", err, got)
	}
	raw, _ := json.Marshal(got.Reports)
	if strings.Contains(string(raw), "x17.example\"") || strings.Contains(string(raw), "r0@") {
		t.Errorf("shared report leaks contact data: %s", raw)
	}
	if auditCount(t, env, "sponsors.report_link_created", "") < 1 {
		t.Error("share not audited")
	}
	// The page anyone with the link opens: the numbers, never indexed.
	status, header, body := readerPage(t, env, "/r/"+token)
	if status != 200 || !strings.Contains(body, "Sponsor report") || !strings.Contains(body, "recipients who got the block") ||
		strings.Contains(body, "r0@") || header.Get("Referrer-Policy") != "no-referrer" || !strings.Contains(header.Get("X-Robots-Tag")+body, "noindex") {
		t.Fatalf("report page: %d %v %s", status, header, body)
	}
	c.Do("DELETE", ws+"/sponsor-report-shares/"+sh.ID, nil).MustStatus(t, 204)
	if _, err := env.App.Sponsors().Shared(ctx, token); !errors.Is(err, sponsors.ErrShareGone) {
		t.Errorf("revoked link still works: %v", err)
	}
	if status, _, body := readerPage(t, env, "/r/"+token); status != 410 || !strings.Contains(body, "This report isn’t available") {
		t.Errorf("revoked page: %d %s", status, body)
	}
	if auditCount(t, env, "sponsors.report_link_revoked", sh.ID) != 1 {
		t.Error("revoke not audited")
	}
	// A Content Writer has no reports:share.
	d := env.Client()
	d.SignInDemo("daniel@acme.io")
	d.Do("POST", ws+"/sponsor-bookings/"+bid.String()+"/shares", map[string]any{}).MustStatus(t, 403)
	// He sees the chip (campaigns:view) but not the Sponsor section (earn:view).
	if chip(d) == "" {
		t.Error("no chip for a Content Writer")
	}
	d.Do("GET", ws+"/campaigns/"+camp.String()+"/sponsor", nil).MustStatus(t, 403)
}

func TestSponsorProposalAndBlock(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("sofia@acme.io")
	sponsor, slot, _ := testSlot(t, env, c)
	s := env.App.Sponsors()
	wsRow, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	agent := &authz.Principal{Type: authz.ActorMCPAgent, ID: uuid.Must(uuid.NewV7()), Name: "Claude — analytics", WorkspaceID: wsRow.ID,
		Permissions: authz.NewSet(authz.SponsorsManage, authz.EarnView)}
	issue, err := s.Propose(ctx, &wsRow, agent, sponsors.ProposeInput{SlotID: id(t, slot), Issue: 15, SponsorID: id(t, sponsor), Note: "They asked for a second issue."})
	if err != nil {
		t.Fatal(err)
	}
	if len(issue.Proposals) != 1 || issue.Proposals[0].Agent != "Claude — analytics" || issue.BookingID != "" {
		t.Fatalf("proposal: %+v", issue)
	}
	// A proposal doesn't book the issue until a person confirms it.
	var b sponsors.SponsorBookingDetail
	c.Do("POST", ws+"/sponsor-bookings/"+issue.Proposals[0].ID+"/confirm", nil).MustStatus(t, 200).JSON(t, &b)
	if b.Status != "booked" || b.Issue != 15 {
		t.Fatalf("confirmed: %+v", b)
	}
	if _, err := s.Propose(ctx, &wsRow, agent, sponsors.ProposeInput{SlotID: id(t, slot), Issue: 15, SponsorID: id(t, sponsor)}); err == nil {
		t.Error("an agent can't propose a booked issue")
	}

	// A campaign of the slot's list on the issue's date gets its Sponsor
	// block from the booking: a placeholder while the creative waits.
	var list uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM lists WHERE name = 'Product Updates'`).Scan(&list); err != nil {
		t.Fatal(err)
	}
	camp := uuid.Must(uuid.NewV7())
	at := b.IssueOn.Add(-3 * time.Hour)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, audience, scheduled_at) VALUES ($1, $2, 'Block test X17', 'scheduled', $3, $4)`,
		camp, wsRow.ID, fmt.Sprintf(`{"lists":["%s"]}`, list), at); err != nil {
		t.Fatal(err)
	}
	var blk sponsors.SponsorBlock
	c.Do("GET", ws+"/campaigns/"+camp.String()+"/sponsor-block", nil).MustStatus(t, 200).JSON(t, &blk)
	if blk.State != sponsors.BlockWaiting || blk.Issue != 15 || blk.Sponsor != "Fabrikam Tools X17" || blk.Label != "Sponsored by Fabrikam Tools X17" {
		t.Fatalf("block: %+v", blk)
	}
	if !strings.Contains(blk.MJML, `rc-sponsor="`) || !strings.Contains(blk.MJML, "The creative is due") {
		t.Errorf("placeholder MJML: %s", blk.MJML)
	}
	if blk.PriceCents == nil || !blk.CanSeePrice {
		t.Error("an Editor sees the price")
	}
	d := env.Client()
	d.SignInDemo("daniel@acme.io")
	var dblk sponsors.SponsorBlock
	d.Do("GET", ws+"/campaigns/"+camp.String()+"/sponsor-block", nil).MustStatus(t, 200).JSON(t, &dblk)
	if dblk.PriceCents != nil || dblk.CanSeePrice {
		t.Error("a Content Writer doesn't see the price")
	}
	// Send Readiness reads the block: waiting, and stale once the creative
	// is ready and the email still holds the placeholder.
	src := `<mjml><mj-body>` + strings.Replace(blk.MJML, `rc-sponsor="`, `rc-sponsor-hash="`+blk.Hash+`" rc-sponsor="`, 1) + `</mj-body></mjml>`
	f := s.ReadinessFacts(ctx, &wsRow, camp, src)
	if !f.HasBlock || f.State != sponsors.BlockWaiting || f.Stale || !f.Labelled {
		t.Errorf("facts: %+v", f)
	}
	c.Do("PUT", ws+"/sponsor-bookings/"+b.ID+"/creative", map[string]any{"headline": "Tools that last", "text": "Built to last.", "button": "Shop", "link": "https://fabrikam-x17.example/shop",
		"utm": map[string]any{"source": "acme-product-updates", "medium": "email", "campaign": "15"}}).MustStatus(t, 200)
	c.Do("POST", ws+"/sponsor-bookings/"+b.ID+"/creative/ready", nil).MustStatus(t, 200)
	f = s.ReadinessFacts(ctx, &wsRow, camp, src)
	if f.State != sponsors.BlockReady || !f.Stale {
		t.Errorf("after ready: %+v", f)
	}
	next, _, err := s.FixMJML(ctx, &wsRow, camp, src, sponsors.FixRefreshBlock)
	if err != nil || !strings.Contains(next, "Tools that last") || !strings.Contains(next, "utm_campaign=15") {
		t.Fatalf("refresh: %v %s", err, next)
	}
	if f := s.ReadinessFacts(ctx, &wsRow, camp, next); f.Stale {
		t.Error("the refreshed block is up to date")
	}
	removed, _, err := s.FixMJML(ctx, &wsRow, camp, next, sponsors.FixRemoveBlock)
	if err != nil || strings.Contains(removed, "rc-sponsor") {
		t.Errorf("remove: %v %s", err, removed)
	}
	_ = store.Workspace{}
}
