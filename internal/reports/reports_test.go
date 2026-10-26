package reports_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type report struct {
	Campaign struct {
		ID, Name, Status, Audience string
		Final                      bool
		FinalAt                    *time.Time
		Notes                      []string
	}
	Headline struct {
		Sent, Delivered, HumanOpens, MachineOpens, Clicks, Unsubscribes, Complaints, Bounces, Undelivered int64
		PrivacyOpens, Measurable                                                                          int64
		DeliveredRate, HumanOpenRate, ClickToOpen, HumanOpenRateMeasurable, PrivacyOpenRate               float64
		ComplaintRateReported                                                                             *float64
		NotReportedBy                                                                                     []string
		SentBy                                                                                            []struct {
			Short string
			Count int64
		}
	}
	Engagement struct {
		Available     bool
		Hours         int
		HumanWithin6h float64
		Points        []struct{ Hour int }
		Rollout       *struct{ Variant string }
	}
	Growth struct {
		NewSubscribers, Forwards, Referrals, WebPost, WebViews int64
	}
	Comparisons struct {
		Issues      int
		Unavailable string
		Tiles       []struct {
			Metric, Unit, Tone string
			Change, Median     float64
		}
	}
	Backlog *struct {
		Minutes   int
		Providers []struct {
			Short   string
			Waiting int64
		}
		Current []string
	}
	ClickMap struct {
		Available bool
		HTML      string
		Total     int64
		BotTotal  int64
		Links     []struct {
			Label                string
			Position             int
			Share                float64
			Grouped              bool
			Hot                  bool
			UniqueClicks, Clicks int64
			BotClicks            int64
		}
		Rows []struct {
			Place     string
			Labels    []string
			Positions []int
			Share     float64
			Clicks    int64
		}
		Other *struct {
			Share float64
			Links int
		}
	}
	Mailboxes []struct {
		Key  string
		Sent int64
	}
	Providers struct {
		Split   bool
		Columns []struct {
			Short            string
			Sent             int64
			Percent          float64
			Cost             *float64
			MedianDeliveryMs *int
		}
	}
	Bounces struct {
		Total int64
		Rows  []struct {
			Type, Suppression string
			Count             int64
			Unsuppressed      int64
			Codes             []struct{ Provider, Type, Code string }
		}
	}
	Unsubscribes struct {
		Total      int64
		ChoseFewer int64
		Reasons    []struct {
			Key   string
			Count int64
		}
	}
	ABTest *struct {
		State, Winner string
		Significant   bool
		Confidence    float64
		Lift          float64
		PValue        float64
		Remaining     int64
		Variants      []struct {
			Key        string
			Label      string
			Recipients int64
			Winner     bool
		}
	}
	NotSent *struct {
		Status, Readiness string
		LastReport        *struct{ Name string }
	}
}

func campaignID(t *testing.T, c *apptest.Client, name string) string {
	t.Helper()
	var list struct {
		Items []struct{ ID, Name string }
	}
	c.Do("GET", "/workspaces/acme-newsletter/campaigns?pageSize=100", nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.Name == name {
			return it.ID
		}
	}
	t.Fatalf("campaign %q not found", name)
	return ""
}

func TestCampaignReport(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	oct := campaignID(t, c, "October Product Update")
	base := "/workspaces/acme-newsletter/campaigns/" + oct + "/report"

	var r report
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &r)
	h := r.Headline
	// The Report board's headline (and the spec's sample data).
	if h.Sent != 47980 || h.Delivered != 47548 || h.HumanOpens != 19590 || h.MachineOpens != 1240 || h.Clicks != 2996 ||
		h.Unsubscribes != 112 || h.Complaints != 19 || h.Bounces != 288 || h.Undelivered != 144 {
		t.Fatalf("headline: %+v", h)
	}
	// Opens in three buckets, and the measurable rate (review AN-10):
	// "41.2% of delivered · 52.8% of measurable".
	if h.PrivacyOpens != 10436 || h.Measurable != 37112 || fmt.Sprintf("%.1f", h.HumanOpenRate) != "41.2" || fmt.Sprintf("%.1f", h.HumanOpenRateMeasurable) != "52.8" ||
		fmt.Sprintf("%.1f", h.PrivacyOpenRate) != "21.9" {
		t.Fatalf("open buckets: %+v", h)
	}
	// Complaints where they are reported (review F-2).
	if len(h.NotReportedBy) != 2 || h.NotReportedBy[0] != "Gmail" || h.NotReportedBy[1] != "iCloud" || h.ComplaintRateReported == nil || fmt.Sprintf("%.2f", *h.ComplaintRateReported) != "0.09" {
		t.Fatalf("complaints where reported: %+v", h)
	}
	if len(h.SentBy) != 2 || h.SentBy[0].Short != "SES" || h.SentBy[0].Count != 43182 || h.SentBy[1].Count != 4798 {
		t.Fatalf("sent by: %+v", h.SentBy)
	}
	// The demo's October Product Update finished 72 hours before 12:41 New
	// York time on its "today", which starts at 11:00, so between 11:00 and
	// 12:41 its results are not final yet.
	final := r.Campaign.FinalAt != nil && !time.Now().Before(*r.Campaign.FinalAt)
	if r.Campaign.Final != final || r.Campaign.Audience != "All subscribers" || len(r.Campaign.Notes) == 0 || r.Campaign.Notes[0] != "A/B subject, winner B" {
		t.Fatalf("campaign: %+v", r.Campaign)
	}
	// Each tile compares with the median of the last issues to the same
	// audience (review AN-11): only Welcome to news.acme.io went to all
	// subscribers before; the Weekly Digest has its own six.
	if want := map[bool]string{true: "", false: "not_final"}[final]; r.Comparisons.Unavailable != want || (final && r.Comparisons.Issues != 1) {
		t.Fatalf("comparisons: %+v", r.Comparisons)
	}
	// The growth the issue brought (review AN-12): the board's 38 new
	// subscribers, 9 from forwards, 12 referrals and 17 from the web post.
	if g := r.Growth; g.NewSubscribers != 38 || g.Forwards != 9 || g.Referrals != 12 || g.WebPost != 17 || g.WebViews == 0 {
		t.Fatalf("growth: %+v", g)
	}
	var wd report
	c.Do("GET", "/workspaces/acme-newsletter/campaigns/"+campaignID(t, c, "Weekly Digest #141")+"/report", nil).MustStatus(t, 200).JSON(t, &wd)
	if wd.Comparisons.Issues != 6 || len(wd.Comparisons.Tiles) != 8 || wd.Comparisons.Tiles[0].Metric != "sent" || wd.Comparisons.Tiles[0].Unit != "percent" {
		t.Fatalf("Weekly Digest #141 comparisons: %+v", wd.Comparisons)
	}
	if !r.Engagement.Available || len(r.Engagement.Points) != 73 || r.Engagement.HumanWithin6h < 61.5 || r.Engagement.HumanWithin6h >= 62.5 ||
		r.Engagement.Rollout == nil || r.Engagement.Rollout.Variant != "B" {
		t.Fatalf("engagement: %+v", r.Engagement)
	}
	if len(r.Mailboxes) != 5 || r.Mailboxes[0].Key != "gmail" || r.Mailboxes[0].Sent != 22071 {
		t.Fatalf("mailboxes: %+v", r.Mailboxes)
	}
	p := r.Providers
	if !p.Split || len(p.Columns) != 2 || p.Columns[0].Short != "SES" || p.Columns[0].Percent != 90 || p.Columns[0].Cost == nil || *p.Columns[0].Cost != 4.32 ||
		p.Columns[0].MedianDeliveryMs == nil || *p.Columns[0].MedianDeliveryMs != 1800 || p.Columns[1].Sent != 4798 {
		t.Fatalf("providers: %+v", p)
	}
	wantBounces := map[string]struct {
		n     int64
		state string
	}{"unknown_user": {171, "auto"}, "domain_invalid": {22, "auto"}, "mailbox_full": {41, "retry"}, "deferral_expired": {38, "retry"}, "blocked": {11, "investigate"}, "other": {5, "review"}}
	if r.Bounces.Total != 288 || len(r.Bounces.Rows) != 6 {
		t.Fatalf("bounces: %+v", r.Bounces)
	}
	for _, row := range r.Bounces.Rows {
		w := wantBounces[row.Type]
		if row.Count != w.n || row.Suppression != w.state {
			t.Errorf("bounce %s: %d %s, want %d %s", row.Type, row.Count, row.Suppression, w.n, w.state)
		}
	}
	if code := r.Bounces.Rows[0].Codes[0]; code.Provider != "SES" || code.Type != "Permanent/General" || code.Code != "550 5.1.1" {
		t.Fatalf("original code: %+v", code)
	}
	u := r.Unsubscribes
	if u.Total != 112 || u.ChoseFewer != 28 || len(u.Reasons) != 5 || u.Reasons[0].Key != "too_frequent" || u.Reasons[0].Count != 41 || u.Reasons[2].Key != "none" || u.Reasons[2].Count != 22 {
		t.Fatalf("unsubscribes: %+v", u)
	}
	ab := r.ABTest
	// The board's card: B won on clicks with 98% confidence (3.4% against
	// 2.6% when it was decided).
	if ab == nil || ab.State != "decided" || ab.Winner != "B" || !ab.Significant || ab.Lift != 0.8 || ab.Confidence < 97.5 || ab.Confidence >= 98.5 || ab.Remaining != 38384 ||
		len(ab.Variants) != 2 || ab.Variants[0].Recipients != 4798 || !ab.Variants[1].Winner ||
		ab.Variants[0].Label != "October Product Update: three launches" || ab.Variants[1].Label != "What shipped in October, and what we're retiring" {
		t.Fatalf("A/B: %+v", ab)
	}
	// The click map is drawn over the email as sent (the composer's
	// content): 2,996 people made 3,412 human clicks across its links (each
	// person once per link), "Read the docs" draws the most, the navigation
	// row is one entry and the other links under 5% are grouped.
	cm := r.ClickMap
	if !cm.Available || cm.Total != 3412 || cm.BotTotal != 1108 || len(cm.Links) != 10 {
		t.Fatalf("click map: %d %d %+v", cm.Total, cm.BotTotal, cm.Links)
	}
	inRow := map[int]bool{}
	for _, row := range cm.Rows {
		for _, p := range row.Positions {
			inRow[p] = true
		}
	}
	sum, grouped, top := 0.0, 0, cm.Links[0]
	for _, l := range cm.Links {
		sum += l.Share
		if l.Share > top.Share {
			top = l
		}
		if l.Grouped {
			grouped++
		}
		if l.Grouped != (l.Share < 5 && !inRow[l.Position]) || l.Hot != (l.Share >= 20) || l.Clicks < l.UniqueClicks || l.BotClicks < 110 || l.BotClicks > 111 {
			t.Errorf("%s at %v%%: grouped %v, hot %v, %d clicks by %d, %d bots", l.Label, l.Share, l.Grouped, l.Hot, l.Clicks, l.UniqueClicks, l.BotClicks)
		}
	}
	if top.Label != "Read the docs" || top.UniqueClicks != 1297 || sum < 99.5 || sum > 100.5 || cm.Other == nil || cm.Other.Links != grouped {
		t.Fatalf("click map: top %+v, shares add to %v, other %+v, %d grouped", top, sum, cm.Other, grouped)
	}
	// The board's table: 38%, 24% and 19%, "Header links" 6% and "Other"
	// 13% (the images; preferences and view in browser aren't tracked).
	shown := map[string]float64{}
	for _, l := range cm.Links {
		if !l.Grouped && !inRow[l.Position] {
			shown[l.Label] = l.Share
		}
	}
	if len(shown) != 3 || shown["Read the docs"] != 38 || shown["See changelog"] != 24 || shown["Read the guide"] != 19 ||
		len(cm.Rows) != 1 || cm.Rows[0].Place != "header" || strings.Join(cm.Rows[0].Labels, ",") != "Blog,Docs,Changelog" || cm.Rows[0].Share != 6 || cm.Rows[0].Clicks != 205 ||
		cm.Other.Share != 13 || cm.Other.Links != 4 {
		t.Fatalf("click map: %v, rows %+v, other %+v", shown, cm.Rows, cm.Other)
	}

	// History imported from Brevo: its totals, and nothing ReputeCast can't
	// know (no hourly curve, no clicks per link, no unsubscribe reasons, no
	// cost of a provider it didn't pay).
	var imp report
	c.Do("GET", "/workspaces/acme-newsletter/campaigns/"+campaignID(t, c, "Weekly Digest #138")+"/report", nil).MustStatus(t, 200).JSON(t, &imp)
	if !imp.Campaign.Final || imp.Engagement.Available || len(imp.ClickMap.Links) != 0 || imp.Headline.Clicks == 0 {
		t.Fatalf("imported: %+v %+v", imp.Campaign, imp.Engagement)
	}
	if imp.Unsubscribes.Total == 0 || len(imp.Unsubscribes.Reasons) != 0 {
		t.Fatalf("imported unsubscribes: %+v", imp.Unsubscribes)
	}
	if len(imp.Providers.Columns) != 1 || imp.Providers.Columns[0].Short != "Brevo" || imp.Providers.Columns[0].Cost != nil {
		t.Fatalf("imported providers: %+v", imp.Providers)
	}

	// A campaign that hasn't been sent: the empty state.
	var next report
	c.Do("GET", "/workspaces/acme-newsletter/campaigns/"+campaignID(t, c, "Weekly Digest #142")+"/report", nil).MustStatus(t, 200).JSON(t, &next)
	if next.NotSent == nil || next.NotSent.Status != "scheduled" || next.NotSent.Readiness != "warnings" || next.NotSent.LastReport == nil || next.NotSent.LastReport.Name != "October Product Update" {
		t.Fatalf("not sent: %+v", next.NotSent)
	}

	// Unknown campaigns and other workspaces.
	c.Do("GET", "/workspaces/acme-newsletter/campaigns/01890000-0000-7000-8000-000000000000/report", nil).MustStatus(t, 404)
	c.Do("GET", "/workspaces/acme-newsletter/campaigns/not-a-uuid/report", nil).MustStatus(t, 422)

	// The Review dialog shows what providers said, never an address.
	rev := c.Do("GET", base+"/bounces/other", nil).MustStatus(t, 200)
	if strings.Contains(string(rev.Body), "@") || !strings.Contains(string(rev.Body), "Undetermined") {
		t.Fatalf("review: %s", rev.Body)
	}

	// Viewers read reports but can't export, share or suppress.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", base, nil).MustStatus(t, 200)
	if p := v.Do("GET", base+"/export.csv", nil).MustStatus(t, 403).Problem(t); p["permission"] != "reports:share" {
		t.Fatalf("export denied: %v", p)
	}
	v.Do("POST", base+"/shares", map[string]any{}).MustStatus(t, 403)
	v.Do("GET", base+"/shares", nil).MustStatus(t, 403)
	v.Do("POST", base+"/bounces/blocked/suppress", nil).MustStatus(t, 403)
	// The Content Writer role can view reports too.
	d := env.Client()
	d.SignInDemo("daniel@acme.io")
	d.Do("GET", base, nil).MustStatus(t, 200)
	d.Do("POST", base+"/print", nil).MustStatus(t, 403)
}

func TestReportExportAndSuppress(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	oct := campaignID(t, c, "October Product Update")
	base := "/workspaces/acme-newsletter/campaigns/" + oct + "/report"
	ctx := context.Background()

	res := c.Do("GET", base+"/export.csv", nil).MustStatus(t, 200)
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("content type %q", ct)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, `filename="october-product-update-report.csv"`) {
		t.Fatalf("disposition %q", cd)
	}
	body := string(res.Body)
	for _, want := range []string{"Campaign report,October Product Update", "Gmail,22071", "Hard · unknown user,171", "Too many emails,41", "Two-proportion z-test",
		"Privacy-protected opens,10436", "Scanner opens (left out),1240", "n/a (not reported)", "Bot clicks (left out),1108"} {
		if !strings.Contains(body, want) {
			t.Errorf("CSV lacks %q", want)
		}
	}
	if strings.Contains(body, "@acme") || strings.Contains(body, "reader.") {
		t.Fatal("CSV contains addresses")
	}
	c.Do("POST", base+"/print", nil).MustStatus(t, 204)
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'report.export' AND target_id = $1`, oct).Scan(&n); err != nil || n != 2 {
		t.Fatalf("export audit entries: %d %v", n, err)
	}

	// Soft bounces are retried, not suppressed; addresses blocked for the
	// sender's reputation are valid and are never suppressed (review AN-7).
	c.Do("POST", base+"/bounces/mailbox_full/suppress", nil).MustStatus(t, 409)
	if p := c.Do("POST", base+"/bounces/blocked/suppress", nil).MustStatus(t, 409).Problem(t); !strings.Contains(fmt.Sprint(p["detail"]), "sending IP or domain") {
		t.Fatalf("blocked: %v", p)
	}
	var sr struct{ Added, Already int }
	c.Do("POST", base+"/bounces/other/suppress", nil).MustStatus(t, 200).JSON(t, &sr)
	if sr.Added != 5 {
		t.Fatalf("suppressed: %+v", sr)
	}
	var r report
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &r)
	for _, row := range r.Bounces.Rows {
		if row.Type == "other" && (row.Suppression != "suppressed" || row.Unsuppressed != 0) {
			t.Fatalf("other after suppressing: %+v", row)
		}
		if row.Type == "blocked" && row.Suppression != "investigate" {
			t.Fatalf("blocked: %+v", row)
		}
	}
	c.Do("POST", base+"/bounces/other/suppress", nil).MustStatus(t, 409)
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'suppression.add'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("suppression audit: %d %v", n, err)
	}

	// The review shows what providers said, never an address inside it.
	for i, who := range []string{"jane.doe@example.org", "sam+news@example.net"} {
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, campaign_id, provider_id, type, occurred_at, original_type, original_code, diagnostic, bounce_class, recipient)
			SELECT $1, c.workspace_id, c.id, (SELECT provider_id FROM campaign_routes WHERE campaign_id = c.id LIMIT 1), 'hard_bounce', now(),
			       'Bounce/Permanent/General', '550 5.7.1', 'smtp; 550 5.7.1 <' || $3 || '>: rejected by policy for ' || $3 || '.', 'policy_block', $3
			FROM campaigns c WHERE c.id = $2`, uuid.Must(uuid.NewV7()), oct, who); err != nil {
			t.Fatalf("event %d: %v", i, err)
		}
	}
	var review struct {
		Diagnostics []struct {
			Diagnostic string
			Count      int64
		}
	}
	res = c.Do("GET", base+"/bounces/blocked", nil).MustStatus(t, 200)
	if strings.Contains(string(res.Body), "@example.") {
		t.Fatalf("the review shows addresses: %s", res.Body)
	}
	res.JSON(t, &review)
	found := false
	for _, d := range review.Diagnostics {
		if d.Diagnostic == "smtp; 550 5.7.1 <<address>>: rejected by policy for <address>" && d.Count == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("masked diagnostics: %+v", review.Diagnostics)
	}
}

func TestReportShares(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	oct := campaignID(t, c, "October Product Update")
	base := "/workspaces/acme-newsletter/campaigns/" + oct + "/report"
	ctx := context.Background()

	// Nothing to share before a campaign is sent.
	c.Do("POST", "/workspaces/acme-newsletter/campaigns/"+campaignID(t, c, "Weekly Digest #142")+"/report/shares", map[string]any{}).MustStatus(t, 409)

	// The demo already shares this report once (the Report board's link).
	var list struct {
		Items []struct{ ID, URL string }
	}
	c.Do("GET", base+"/shares", nil).MustStatus(t, 200).JSON(t, &list)
	seeded := len(list.Items)
	if seeded != 1 || list.Items[0].URL == "" {
		t.Fatalf("seeded links: %+v", list)
	}

	var link struct{ ID, URL string }
	c.Do("POST", base+"/shares", map[string]any{}).MustStatus(t, 201).JSON(t, &link)
	prefix := env.App.URLs.Report("")
	if !strings.HasPrefix(link.URL, prefix) || len(link.URL)-len(prefix) != 22 {
		t.Fatalf("link %q", link.URL)
	}
	token := strings.TrimPrefix(link.URL, prefix)
	c.Do("GET", base+"/shares", nil).MustStatus(t, 200).JSON(t, &list)
	// Newest first.
	if len(list.Items) != seeded+1 || list.Items[0].ID != link.ID || list.Items[0].URL != link.URL {
		t.Fatalf("list: %+v", list)
	}
	// The token is stored sealed and hashed, never in the clear.
	var stored int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM report_shares WHERE position($1::bytea IN token_sealed) > 0`, []byte(token)).Scan(&stored); err != nil || stored != 0 {
		t.Fatalf("token in the clear: %d %v", stored, err)
	}

	// Anyone with the link: the aggregate report, no ids, no addresses.
	anon := env.Client()
	res := anon.Do("GET", "/shared-reports/"+token, nil).MustStatus(t, 200)
	if res.Header.Get("X-Robots-Tag") != "noindex, nofollow" || res.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("headers: %v", res.Header)
	}
	var shared struct {
		Workspace string
		Report    report
	}
	res.JSON(t, &shared)
	if shared.Workspace != "Acme Newsletter" || shared.Report.Campaign.ID != "" || shared.Report.Headline.Sent != 47980 {
		t.Fatalf("shared: %s %+v", shared.Workspace, shared.Report.Campaign)
	}
	if strings.Contains(string(res.Body), "reader.") || strings.Contains(string(res.Body), "former.reader") || strings.Contains(string(res.Body), oct) {
		t.Fatal("the shared report leaks addresses or ids")
	}
	// Revoked links stop working.
	c.Do("DELETE", base+"/shares/"+link.ID, nil).MustStatus(t, 204)
	anon.Do("GET", "/shared-reports/"+token, nil).MustStatus(t, 404)
	c.Do("DELETE", base+"/shares/"+link.ID, nil).MustStatus(t, 404)
	// Expired links too.
	var exp struct{ ID, URL string }
	c.Do("POST", base+"/shares", map[string]any{"expiresInDays": 7}).MustStatus(t, 201).JSON(t, &exp)
	anon.Do("GET", "/shared-reports/"+strings.TrimPrefix(exp.URL, prefix), nil).MustStatus(t, 200)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE report_shares SET expires_at = now() - interval '1 minute' WHERE id = $1`, exp.ID); err != nil {
		t.Fatal(err)
	}
	anon.Do("GET", "/shared-reports/"+strings.TrimPrefix(exp.URL, prefix), nil).MustStatus(t, 404)
	c.Do("GET", base+"/shares", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != seeded {
		t.Fatalf("revoked or expired links listed: %+v", list)
	}
	for _, l := range list.Items {
		if l.ID == link.ID || l.ID == exp.ID {
			t.Fatalf("revoked or expired link listed: %+v", l)
		}
	}
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action IN ('report_share.create', 'report_share.revoke')`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("share audit entries: %d %v", n, err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE after::text LIKE '%' || $1 || '%' OR metadata::text LIKE '%' || $1 || '%'`, token).Scan(&n); err != nil || n != 0 {
		t.Fatalf("token in the audit log: %d %v", n, err)
	}
}

// TestReportLiveAndBehind covers a send that ended recently (results still
// coming in), provider events waiting in the ingest queue (ReportStates'
// error) and an email whose stored content is missing.
func TestReportLiveAndBehind(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	id := campaignID(t, c, "Weekly Digest #141")
	base := "/workspaces/acme-newsletter/campaigns/" + id + "/report"
	ctx := context.Background()

	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET started_at = now() - interval '3 hours', completed_at = now() - interval '2 hours', html = ''
		WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	// Each read decodes into a fresh report: fields left out (backlog) stay empty.
	get := func() (r report) {
		t.Helper()
		c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &r)
		return r
	}
	r := get()
	if r.Campaign.Final || r.Campaign.FinalAt == nil || time.Until(*r.Campaign.FinalAt) < 69*time.Hour || time.Until(*r.Campaign.FinalAt) > 70*time.Hour {
		t.Fatalf("live: %+v", r.Campaign)
	}
	if r.Engagement.Hours != 3 || r.Backlog != nil {
		t.Fatalf("live engagement %d hours, backlog %+v", r.Engagement.Hours, r.Backlog)
	}
	// Without the email the map can't be drawn; the clicks per link remain.
	if r.ClickMap.Available || r.ClickMap.HTML != "" || len(r.ClickMap.Links) == 0 || r.ClickMap.Total == 0 {
		t.Fatalf("click map without content: %+v", r.ClickMap)
	}

	// SES events received 12 minutes ago wait to be processed. Postal's
	// unsigned posts are never processed, so they don't make it "behind".
	queue := func(adapter string, n int, signed bool) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO provider_raw_events (id, workspace_id, provider_id, received_at, body, signature_ok)
			SELECT gen_random_uuid(), p.workspace_id, p.id, now() - interval '12 minutes' + g * interval '1 second', '\x7b7d'::bytea, $3
			FROM providers p, generate_series(1, $4::int) g
			WHERE p.adapter = $2 AND p.workspace_id = (SELECT workspace_id FROM campaigns WHERE id = $1)`, id, adapter, signed, n); err != nil {
			t.Fatal(err)
		}
	}
	queue("ses", 40, true)
	queue("postal", 5, false)
	r = get()
	b := r.Backlog
	if b == nil || b.Minutes < 11 || b.Minutes > 12 || len(b.Providers) != 1 || b.Providers[0].Short != "SES" || b.Providers[0].Waiting != 40 ||
		strings.Join(b.Current, ",") != "Postal,Postmark" {
		t.Fatalf("backlog: %+v", b)
	}
	// A queue younger than five minutes isn't behind yet.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE provider_raw_events SET received_at = now() - interval '2 minutes' WHERE normalized_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	r = get()
	if r.Backlog != nil {
		t.Fatalf("a 2-minute queue is current: %+v", r.Backlog)
	}

	// Once the results are final, a late queue no longer matters.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE provider_raw_events SET received_at = now() - interval '12 minutes' WHERE normalized_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET started_at = now() - interval '74 hours', completed_at = now() - interval '73 hours' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	r = get()
	if !r.Campaign.Final || r.Backlog != nil || r.Engagement.Hours != 72 {
		t.Fatalf("final: %+v, backlog %+v, %d hours", r.Campaign, r.Backlog, r.Engagement.Hours)
	}
}

func TestSharedReportRateLimit(t *testing.T) {
	env := apptest.New(t)
	anon := env.Client()
	// Guessing tokens stops after 20 misses from one address.
	for i := 0; i < 20; i++ {
		anon.Do("GET", "/shared-reports/AAAAAAAAAAAAAAAAAAAAAA", nil).MustStatus(t, 404)
	}
	anon.Do("GET", "/shared-reports/AAAAAAAAAAAAAAAAAAAAAA", nil).MustStatus(t, 429)
}
