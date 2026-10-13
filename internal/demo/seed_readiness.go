package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedReadiness = seedReadinessImpl }

// readinessPolicy is the approval policy of the sample data (A5 §4): audience
// ≥ 10,000, a warming-up provider, an author without campaigns:send; one
// approval from an Owner or Admin; Owners can bypass with a reason.
const readinessPolicy = `{"rules":{"audience_min":{"enabled":true,"n":10000},"warming_provider":{"enabled":true},"new_provider":{"enabled":false,"days":14},` +
	`"ready_with_warnings":{"enabled":false},"author_lacks_send":{"enabled":true},"approval_lists":{"enabled":false}},"approvals_required":1,` +
	`"approver_roles":["owner","admin"],"automation_publish":false,"emergency_bypass":true}`

// readinessThresholds are the auto-pause thresholds of the boards.
const readinessThresholds = `{"bounce":2,"complaint":0.1,"mailbox_complaint":0.1,"provider_errors":5,"scope":"route"}`

// seedReadinessImpl seeds Review & send and send approvals: complete Send
// Readiness runs for Weekly Digest #142 (SendReadiness: ready with two
// warnings), the flash sale (SendReadinessBlocked, during a Postal outage)
// and "November Product Update" (Daniel Park's draft waiting for approval,
// A5 §4), the approval policy, the history of the approval inbox (approved,
// changes requested, invalidated) and Weekly Digest's test sends.
//
// Runs go through readiness.Assemble with the boards' sample facts, so they
// read exactly like a run of the engine; re-running the checks recomputes
// them from the workspace's live data.
func seedReadinessImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET approval_policy = $2 WHERE id = $1`, ws.ID, json.RawMessage(readinessPolicy)); err != nil {
		return err
	}
	ws.ApprovalPolicy = json.RawMessage(readinessPolicy)
	r := &readinessSeed{s: s, ws: ws, images: map[string]string{}}
	if err := r.loadImages(ctx); err != nil {
		return err
	}
	steps := []func(context.Context) error{r.weeklyDigest, r.flashSale, r.november, r.history}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

type readinessSeed struct {
	s      *Seeder
	ws     store.Workspace
	images map[string]string
}

func (r *readinessSeed) loadImages(ctx context.Context) error {
	rows, err := r.s.Tx.Query(ctx, `SELECT id, file_name FROM media_assets WHERE workspace_id = $1`, r.ws.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		r.images[name] = media.URL(id, name)
	}
	return rows.Err()
}

func (r *readinessSeed) withImages(src string) string {
	for name, u := range r.images {
		src = strings.ReplaceAll(src, "@@"+name+"@@", u)
	}
	return src
}

func f64(v float64) *float64 { return &v }

// shares are the providers of the sample route: Amazon SES 90%, Postal 10%
// warming up (day 9 of 21, today's cap 6,000).
func (r *readinessSeed) ses(n int64, pct float64) readiness.ShareFacts {
	s := r.s
	return readiness.ShareFacts{ProviderID: s.Providers["ses"].String(), Name: "Amazon SES", Short: "SES", Adapter: "ses", Label: "us-east-1", Recipients: n, Percent: pct,
		Status: state.Healthy, Available: true, Rate: 14, Pricing: state.Pricing{Model: "metered", Per1000: 0.10}, Bounce: f64(0.6), Complaint: f64(0.04),
		Limits:  state.Limits{Bounce: state.Limit{Limit: 5, Review: 5, Pause: 10, Label: "SES"}, Complaint: state.Limit{Limit: 0.1, Review: 0.1, Pause: 0.5, Label: "SES"}},
		Account: "Production · no enforcement", Quota: &state.Quota{Kind: "rolling_24h", Max: 50000, Used: 2140}, BulkAllowed: true, Terms: "yes, with production access",
		Tracking: state.Tracking{Scope: "Configuration set rc-broadcasts", Mode: "reputecast"}, ConnectedAt: s.Ago(210 * 24 * time.Hour)}
}

func (r *readinessSeed) postal(n int64, pct float64) readiness.ShareFacts {
	s := r.s
	return readiness.ShareFacts{ProviderID: s.Providers["postal"].String(), Name: "Postal", Short: "Postal", Adapter: "postal", Label: "self-hosted · mta.acme.io",
		Recipients: n, Percent: pct, Status: state.WarmingUp, Available: true, WarmupDay: 9, WarmupDays: 21, WarmupCap: 6000, WarmupToday: true, Rate: 2, RateRamp: true,
		Pricing: state.Pricing{Model: "self_hosted", Monthly: 24}, Bounce: f64(0.9), Complaint: f64(0.05),
		Limits: state.Limits{Bounce: state.Limit{Limit: 2, Label: "warm-up guard"}, Complaint: state.Limit{Limit: 0.1}}, Account: "Postal 3.3.4 · reachable",
		BulkAllowed: true, Terms: "self-hosted, your own policy", Tracking: state.Tracking{Scope: "mail server mta.acme.io", Mode: "reputecast"},
		IPs: []string{"203.0.113.25"}, ConnectedAt: s.Ago(8*24*time.Hour + 2*time.Hour)}
}

// baseFacts are what every sample run shares: the workspace, sender
// reputation on HostRepute's 84 lists, authentication of news.acme.io for
// SES and Postal, and one-click unsubscribe.
func (r *readinessSeed) baseFacts(checked time.Time) readiness.Facts {
	loc := state.Location("America/New_York")
	domainSources := []string{"Spamhaus DBL", "SURBL", "URIBL", "Barracuda", "Spamhaus ZRD", "Google Safe Browsing", "Abusix DBL", "Invaluement ivmURI"}
	ipSources := []string{"Spamhaus ZEN", "Barracuda", "SpamCop", "Mailspike", "UCEPROTECT Level 1", "UCEPROTECT Level 2", "UCEPROTECT Level 3"}
	return readiness.Facts{
		Now: r.s.Now, Location: loc, Workspace: r.ws.Slug, MailboxRates: readiness.DefaultMailboxRates,
		Reputation: readiness.ReputationFacts{Status: readiness.Warn, Via: "hostrepute", Sources: 84, CheckedAt: &checked, DomainSources: domainSources, IPSources: ipSources,
			Listings: []readiness.ListingFacts{{Asset: "203.0.113.25", Type: "ip", Source: "UCEPROTECT Level 3", Impact: "low", Detail: "Network-range listing (/16), low impact",
				DelistURL: "https://www.uceprotect.net/en/rblcheck.php"}}},
		Auth: readiness.AuthFacts{Domain: "news.acme.io", Checked: true, CheckedAt: &checked,
			SPFRecord: "v=spf1 include:amazonses.com include:spf.mta.acme.io -all",
			SPF:       []readiness.ProviderAuthFacts{{Short: "SES", OK: true}, {Short: "Postal", OK: true}},
			DKIM: []readiness.ProviderAuthFacts{{Short: "SES", OK: true, Detail: "3 Easy DKIM CNAMEs verified", KeyBits: 2048},
				{Short: "Postal", OK: true, Detail: "selector postal-2026 verified", KeyBits: 2048}},
			MailFrom:    []readiness.ProviderAuthFacts{{Short: "SES", OK: true, Domain: "bounce.news.acme.io"}, {Short: "Postal", OK: true, Domain: "rp.news.acme.io"}},
			DMARCStatus: "pass", DMARCPolicy: "quarantine", DMARCRecord: "v=DMARC1; p=quarantine; rua=mailto:dmarc@reports.hostrepute.com; adkim=s", DMARCAligned: f64(100),
			BIMIStatus: "missing"},
		Compliance: readiness.ComplianceFacts{UnsubscribeURL: "https://news.acme.io/u/{token}", HTTPS: true, ComplaintRate: f64(0.04)},
	}
}

// links builds a probed link list.
func probed(entries ...[3]string) []contentcheck.ContentCheckLink {
	out := []contentcheck.ContentCheckLink{}
	for i, e := range entries {
		host := strings.TrimPrefix(strings.SplitN(strings.TrimPrefix(e[0], "https://"), "/", 2)[0], "www.")
		ms := 120 + (i*37)%400
		if strings.Contains(e[0], "docs.acme.io/changelog") {
			ms = 840
		}
		out = append(out, contentcheck.ContentCheckLink{URL: e[0], Host: host, Text: e[1], Kind: e[2], Path: []int{}, Probed: true, Status: 200, Millis: ms})
	}
	return out
}

// storeRun saves a run assembled from facts as the campaign's latest.
func (r *readinessSeed) storeRun(ctx context.Context, id uuid.UUID, f readiness.Facts, ranAt time.Time, by *uuid.UUID, trigger string) (readiness.ReadinessResult, uuid.UUID, error) {
	s := r.s
	res := readiness.Assemble(f)
	checks, est, err := readiness.EncodeResult(res)
	if err != nil {
		return res, uuid.Nil, err
	}
	c, err := s.Q.ReadinessCampaign(ctx, store.ReadinessCampaignParams{WorkspaceID: r.ws.ID, ID: id})
	if err != nil {
		return res, uuid.Nil, err
	}
	// The composer stage left a short run; this one replaces it.
	if _, err := s.Tx.Exec(ctx, `DELETE FROM readiness_runs WHERE campaign_id = $1`, id); err != nil {
		return res, uuid.Nil, err
	}
	runID := ID()
	err = s.Q.ReadinessInsertRun(ctx, store.ReadinessInsertRunParams{ID: runID, CampaignID: id, Trigger: trigger, Verdict: res.Verdict, Checks: checks, Estimates: est,
		RanAt: ranAt, DurationMs: 9400, RanBy: by, CampaignVersion: c.Version, Fingerprint: readiness.Fingerprint(c, readiness.RoutingKey(ctx, s.Q, r.ws.ID))})
	return res, runID, err
}

// ---------------------------------------------------------------- Weekly Digest #142

func (r *readinessSeed) weeklyDigest(ctx context.Context) error {
	s := r.s
	id, ok := s.Campaigns["Weekly Digest #142"]
	if !ok {
		return nil
	}
	linh := s.Users["linh@acme.io"]
	checked := s.designDay(0, 9, 40)
	compiled := s.designDay(0, 9, 39)
	if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET thresholds = $2 WHERE id = $1`, id, json.RawMessage(readinessThresholds)); err != nil {
		return err
	}
	var subject string
	var sched time.Time
	if err := s.Tx.QueryRow(ctx, `SELECT subject, coalesce(scheduled_at, now()) FROM campaigns WHERE id = $1`, id).Scan(&subject, &sched); err != nil {
		return err
	}
	f := r.baseFacts(checked)
	f.Campaign = readiness.CampaignFacts{ID: id.String(), Name: "Weekly Digest #142", Subject: subject, Version: 14, CompiledAt: &compiled, SenderName: "Acme Newsletter",
		SenderEmail: "digest@news.acme.io", SenderDomain: "news.acme.io", StartAt: sched, ScheduleMode: "at"}
	f.Audience = readiness.AudienceFacts{BaseLabel: "Weekly Digest list", GroupLabel: "Weekly Digest", Lists: []string{"Weekly Digest"}, Base: 31988, Suppressed: 331,
		Role: 27, Duplicates: 26, Final: 31604, Unengaged: 412, UnengagedDays: 180,
		Mailboxes: map[string]int64{"gmail": 14538, "outlook": 6637, "yahoo": 2844, "icloud": 2528, "other": 5057}}
	f.Route = readiness.RouteFacts{Stream: "broadcast", Policy: "pause_and_ask", Shares: []readiness.ShareFacts{r.ses(28444, 90), r.postal(3160, 10)}}
	f.Content = readiness.ContentFacts{Checked: true, Address: "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105", InlineCSSBytes: 11 * 1024,
		URIBlocklists: readiness.URIBlocklists, OwnDomain: "acme.io",
		Report: contentcheck.ContentCheckReport{Verdict: readiness.VerdictReady, SizeBytes: 64 * 1024, ClipBytes: contentcheck.ClipBytes, Images: 7, Links: 14,
			SubjectHeuristics: []string{}, Issues: []contentcheck.ContentCheckIssue{}, Sections: []contentcheck.ContentCheckSection{}, PlainText: strings.Repeat("x", 9*1024),
			CheckedAt: checked, ImageShare: 0.38, Words: 1140,
			LinkList: probed(
				[3]string{"https://acme.io/blog", "Blog", "link"}, [3]string{"https://docs.acme.io", "Docs", "link"}, [3]string{"https://acme.io/changelog", "Changelog", "link"},
				[3]string{"https://docs.acme.io/flags", "Read the docs", "button"}, [3]string{"https://docs.acme.io/changelog", "See the changelog", "button"},
				[3]string{"https://docs.acme.io/migrate-v2", "migration guide", "link"}, [3]string{"https://acme.io/q4", "Q4 roadmap", "image"},
				[3]string{"https://acme.io/audit-log", "Audit log exports", "link"}, [3]string{"https://github.com/acme/flags", "GitHub", "social"},
				[3]string{"https://www.youtube.com/@acme", "YouTube", "social"}, [3]string{"https://acme.io/pricing", "Pricing", "link"},
				[3]string{"https://docs.acme.io/api/v2", "v2 API", "link"}, [3]string{"https://acme.io/events", "Events", "link"},
				[3]string{"https://acme.io/blog/flags-ga", "Feature flags are generally available", "link"})}}
	if _, _, err := r.storeRun(ctx, id, f, checked, &linh.ID, readiness.TriggerManual); err != nil {
		return err
	}
	// Two test sends at 09:12 and the schedule confirmed at 09:42.
	tests := []struct{ to, provider, mailbox string }{{"linh@acme.io", "SES", "Gmail"}, {"team-qa@acme.io", "Postal", "Outlook"}}
	for i, t := range tests {
		if err := readinessAudit(ctx, s, s.designDay(0, 9, 12).Add(time.Duration(i)*time.Second), linh, "campaign.test_send", "campaign", id.String(), "Weekly Digest #142",
			map[string]any{"to": []string{t.to}, "variant": "", "provider": t.provider}, map[string]any{"delivered": map[string]string{t.to: t.mailbox}}, "203.0.113.9"); err != nil {
			return err
		}
	}
	return readinessAudit(ctx, s, s.designDay(0, 9, 42), linh, "campaign.scheduled", "campaign", id.String(), "Weekly Digest #142",
		map[string]any{"status": "scheduled", "scheduled_at": sched}, map[string]any{"recipients": 31604, "schedule": sched.In(state.Location("America/New_York")).Format("Jan 2, 15:04")}, "203.0.113.9")
}

// readinessAudit writes one audit entry as a person.
func readinessAudit(ctx context.Context, s *Seeder, at time.Time, u store.User, action, targetType, targetID, target string, after, meta map[string]any, ip string) error {
	ws := s.WS()
	a, _ := json.Marshal(after)
	m, _ := json.Marshal(meta)
	if meta == nil {
		m = []byte(`{}`)
	}
	_, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name, result, ip, user_agent, after, metadata)
		VALUES ($1,$2,$3,'user',$4,$5,$6,$7,$8,$9,'allowed',$10::inet,'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Chrome/129.0 Safari/537.36',$11,$12)`,
		ID(), ws.ID, at, u.ID, u.Name, action, targetType, targetID, target, ip, a, m)
	return err
}

// ---------------------------------------------------------------- flash sale

func (r *readinessSeed) flashSale(ctx context.Context) error {
	s := r.s
	id, ok := s.Campaigns["Flash sale: 48 hours only"]
	if !ok {
		return nil
	}
	linh := s.Users["linh@acme.io"]
	checked := s.designDay(0, 9, 40)
	compiled := s.designDay(0, 9, 31)
	down := s.designDay(0, 8, 52)
	if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET thresholds = $2 WHERE id = $1`, id, json.RawMessage(readinessThresholds)); err != nil {
		return err
	}
	f := r.baseFacts(checked)
	f.Reputation.Listings = []readiness.ListingFacts{}
	f.Reputation.Status = readiness.Pass
	f.Auth.SPF, f.Auth.DKIM, f.Auth.MailFrom = f.Auth.SPF[:1], f.Auth.DKIM[:1], f.Auth.MailFrom[:1]
	f.Campaign = readiness.CampaignFacts{ID: id.String(), Name: "Flash sale: 48 hours only", Subject: "FLASH SALE!!! 48 HOURS ONLY", Version: 6, CompiledAt: &compiled,
		SenderName: "Acme", SenderEmail: "hello@news.acme.io", SenderDomain: "news.acme.io", StartAt: s.Now, ScheduleMode: "now"}
	f.Audience = readiness.AudienceFacts{BaseLabel: "All subscribers", GroupLabel: "All subscribers", Lists: []string{"Weekly Digest", "Product Updates", "Beta Program"},
		Base: 48213, Suppressed: 340, Role: 31, Final: 47842, Unengaged: 1604, UnengagedDays: 180,
		Mailboxes: map[string]int64{"gmail": 22007, "outlook": 10047, "yahoo": 4306, "icloud": 3827, "other": 7655}}
	postal := r.postal(4784, 10)
	postal.Status, postal.Available, postal.DownSince = state.Down, false, &down
	postal.DownDetail = "mta.acme.io has refused connections since 08:52 (ECONNREFUSED on port 25 and 443)"
	f.Route = readiness.RouteFacts{Stream: "broadcast", Policy: "pause_and_ask", Shares: []readiness.ShareFacts{r.ses(43058, 90), postal}}
	blockURL := "https://go.acme-promo.link/sale"
	f.Content = readiness.ContentFacts{Checked: true, FooterRemovedAt: &compiled, InlineCSSBytes: 14 * 1024, URIBlocklists: readiness.URIBlocklists, OwnDomain: "acme.io",
		Report: contentcheck.ContentCheckReport{Verdict: readiness.VerdictBlocked, SizeBytes: 71 * 1024, ClipBytes: contentcheck.ClipBytes, Images: 9, Links: 12,
			SubjectHeuristics: []string{"all caps", "repeated punctuation", "urgency"}, Sections: []contentcheck.ContentCheckSection{}, PlainText: strings.Repeat("x", 6*1024),
			CheckedAt: checked, ImageShare: 0.44, Words: 610,
			Issues: []contentcheck.ContentCheckIssue{
				{ID: "compliance.footer", Severity: contentcheck.Block, Group: "Compliance", Title: "No unsubscribe link or postal address in the body", Section: -1, Path: []int{},
					Fixes: []contentcheck.ContentCheckFix{{Kind: "insert_footer", Label: "Insert footer block"}}},
				{ID: "links.blocklist", Severity: contentcheck.Block, Group: "Links", Title: "Button links to go.acme-promo.link, listed on SURBL and Spamhaus DBL",
					Tag: "Link domain on SURBL and Spamhaus DBL", URL: blockURL, Path: []int{2, 0, 3}, Section: 2,
					Fixes: []contentcheck.ContentCheckFix{{Kind: "replace_link", Label: "Replace with acme.io/sale", Value: "https://acme.io/sale", Path: []int{2, 0, 3}}}},
				{ID: "subject.spam", Severity: contentcheck.Warn, Group: "Subject", Title: "Subject trips 3 spam heuristics", Section: -1, Path: []int{}},
			},
			LinkList: append(probed(
				[3]string{"https://acme.io", "Acme", "image"}, [3]string{"https://acme.io/pricing", "Pricing", "link"}, [3]string{"https://acme.io/plans/starter", "Starter", "button"},
				[3]string{"https://acme.io/plans/team", "Team", "button"}, [3]string{"https://acme.io/plans/business", "Business", "button"},
				[3]string{"https://acme.io/plans/enterprise", "Enterprise", "button"}, [3]string{"https://docs.acme.io/billing", "billing FAQ", "link"},
				[3]string{"https://acme.io/terms", "Terms", "link"}, [3]string{"https://github.com/acme", "GitHub", "social"}, [3]string{"https://www.youtube.com/@acme", "YouTube", "social"},
				[3]string{"https://acme.io/blog", "Blog", "link"}),
				contentcheck.ContentCheckLink{URL: blockURL, Host: "go.acme-promo.link", Text: "Claim your discount", Kind: "button", Path: []int{2, 0, 3}, Probed: true, Status: 200, Millis: 210})}}
	_, _, err := r.storeRun(ctx, id, f, checked, &linh.ID, readiness.TriggerManual)
	return err
}

// ---------------------------------------------------------------- November Product Update

const novemberSubject = "November Product Update: faster exports, new API, and a Postal milestone"
const novemberPreheader = "Plus: the new team permissions and what changed in approvals"

// novemberMJML is Daniel's draft. Its third link goes through bit.ly (the
// URL-shortener warning of the sample data).
func novemberMJML(stage int) string {
	headline := "Faster exports, a new API and a Postal milestone"
	body := `Hi {{ first_name | default: "there" }}, exports now finish in under a minute for 100k rows, the v2 API ships with batch endpoints, and our self-hosted mail server completed warm-up.`
	if stage < 3 {
		headline, body = "What shipped in November", `Hi {{ first_name | default: "there" }}, here's what shipped this month at Acme.`
	}
	permissions := ""
	if stage >= 6 {
		permissions = `
    <mj-section background-color="#ffffff" padding="8px 24px 18px">
      <mj-column>
        <mj-divider border-width="1px" border-color="#E3E6EB" padding="0px 0px 14px"></mj-divider>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Team permissions and approvals</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151">Content Writers can now prepare a send and ask for approval. <a href="https://docs.acme.io/approvals" style="color:#0F766E">How approvals work</a>.</mj-text>
      </mj-column>
    </mj-section>`
	}
	button := ""
	if stage >= 7 {
		button = `
        ` + composerButton("https://bit.ly/3xK9tQ", "Read the update")
	}
	return `<mjml>
  <mj-head>
    <mj-title>` + composerXMLText(novemberSubject) + `</mj-title>
    <mj-preview>` + composerXMLText(novemberPreheader) + `</mj-preview>
    <mj-attributes>
      <mj-all font-family="` + composerPlexFont + `"></mj-all>
      <mj-text color="#1F2937" padding="0px"></mj-text>
    </mj-attributes>
  </mj-head>
  <mj-body width="600px" background-color="#F3F4F6">
    <!-- rc:header-logo -->
    <mj-section background-color="#ffffff" padding="16px 24px" border-bottom="1px solid #EEF0F3">
      <mj-column width="50%" vertical-align="middle">
        <mj-image src="@@acme-logo.png@@" alt="Acme" href="https://acme.io" width="96px" height="28px" align="left" padding="0px"></mj-image>
      </mj-column>
      <mj-column width="50%" vertical-align="middle">
        <mj-text align="right" font-size="12px" color="#5B6472">November 2026</mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#0F766E" padding="0px">
      <mj-column>
        <mj-image src="@@october-hero.png@@" alt="Exports, API and Postal milestones" width="600px" height="180px" padding="0px"></mj-image>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="20px 24px 8px">
      <mj-column>
        <mj-text font-size="22px" line-height="1.25" font-weight="700" color="#0B1220" padding="0px 0px 10px">` + headline + `</mj-text>
        <mj-text font-size="15px" line-height="1.55" padding="0px 0px 14px">` + body + ` Read about the <a href="https://docs.acme.io/exports" style="color:#0F766E">new exports</a>.</mj-text>` + button + `
      </mj-column>
    </mj-section>` + permissions + `
` + composerDigestFooter + `
  </mj-body>
</mjml>
`
}

func (r *readinessSeed) november(ctx context.Context) error {
	s := r.s
	ws := r.ws
	daniel, linh := s.Users["daniel@acme.io"], s.Users["linh@acme.io"]
	sender, err := composerEnsureSender(ctx, s, "news.acme.io", "Acme Newsletter", "product@news.acme.io")
	if err != nil {
		return err
	}
	id, ok := s.Campaigns["November Product Update"]
	sched := s.designDay(-23, 9, 0)
	scheduleJSON := json.RawMessage(fmt.Sprintf(`{"mode":"at","at":%q,"timezone":"America/New_York"}`, sched.UTC().Format(time.RFC3339)))
	audience, _ := json.Marshal(map[string]any{"lists": []string{s.Lists["Weekly Digest"].String(), s.Lists["Product Updates"].String()}})
	requested := s.Ago(2 * time.Hour)
	if !ok {
		id = ID()
		count := int64(47912)
		created := s.Ago(3*24*time.Hour + 4*time.Hour)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, audience, audience_count, schedule, thresholds,
			created_by, updated_by, created_at, updated_at)
			VALUES ($1,$2,'November Product Update','approval',$3,$4,$5,$6,$7,$8,$9,$10,$10,$11,$11)`,
			id, ws.ID, novemberSubject, novemberPreheader, sender, audience, count, scheduleJSON, json.RawMessage(readinessThresholds), daniel.ID, created); err != nil {
			return err
		}
		s.Campaigns["November Product Update"] = id
	}
	// Nine versions by Daniel, the last two hours ago.
	labels := []string{"created from template “Acme digest v3”", "header and date", "hero copy", "subject and preheader", "exports link", "permissions section",
		"button link", "footer", "preheader"}
	var html, src string
	for i, label := range labels {
		src = r.withImages(novemberMJML(i + 1))
		res := mjml.Compile(src)
		if len(res.Errors) > 0 {
			return fmt.Errorf("compile November MJML: %s", res.Errors[0].Message)
		}
		html = res.HTML
		at := s.Ago(3*24*time.Hour + 4*time.Hour - time.Duration(i)*7*time.Hour)
		if i == len(labels)-1 {
			at = requested.Add(-10 * time.Minute)
		}
		source := "autosave"
		if i == 0 {
			source = "template"
		}
		subject := novemberSubject
		if i < 3 {
			subject = "November at Acme"
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, saved_by, saved_at, label, source, size_bytes)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			id, i+1, subject, novemberPreheader, src, html, contentcheck.PlainText(html), daniel.ID, at, label, source, len(html)); err != nil {
			return err
		}
	}
	saved := requested.Add(-10 * time.Minute)
	hash := campaigns.ContentHash(novemberSubject, novemberPreheader, src, html, nil)
	if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET mjml = $2, html = $3, plain_text = $4, content_hash = $5, version = $6, autosaved_at = $7, updated_at = $7 WHERE id = $1`,
		id, src, html, contentcheck.PlainText(html), hash, len(labels), saved); err != nil {
		return err
	}
	// The planned route of the sample data.
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_routes (campaign_id, provider_id, planned, status, cost_micros) VALUES ($1,$2,43121,'pending',4312100),($1,$3,4791,'pending',0)
		ON CONFLICT (campaign_id, provider_id) DO NOTHING`, id, s.Providers["ses"], s.Providers["postal"]); err != nil {
		return err
	}
	// Send Readiness: the composer's real content check (links answered by
	// the demo overlay) and the sample data's audience and route.
	checker := &campaigns.Service{Q: s.Q, Checker: &contentcheck.Checker{Blocklist: contentcheck.StoreBlocklist{Q: s.Q}, Prober: ReadinessProberOverlay(nil), Now: func() time.Time { return requested.Add(-5 * time.Minute) }}}
	chk, err := checker.Checks(ctx, &ws, id, campaigns.ComposerChecksRequest{Subject: novemberSubject, Preheader: novemberPreheader, MJML: src, HTML: html, ProbeLinks: true})
	if err != nil {
		return err
	}
	ranAt := requested.Add(-5 * time.Minute)
	f := r.baseFacts(ranAt)
	f.Reputation.Listings, f.Reputation.Status = []readiness.ListingFacts{}, readiness.Pass
	f.Campaign = readiness.CampaignFacts{ID: id.String(), Name: "November Product Update", Subject: novemberSubject, Preheader: novemberPreheader, Version: len(labels),
		CompiledAt: &saved, SenderName: "Acme Newsletter", SenderEmail: "product@news.acme.io", SenderDomain: "news.acme.io", StartAt: sched, ScheduleMode: "at"}
	mb := Split(47912, []float64{0.46, 0.21, 0.09, 0.08, 0.16})
	f.Audience = readiness.AudienceFacts{BaseLabel: "Weekly Digest + Product Updates", GroupLabel: "Weekly Digest + Product Updates", Lists: []string{"Weekly Digest", "Product Updates"},
		Base: 48240, Suppressed: 301, Role: 27, Final: 47912, Unengaged: 1485, UnengagedDays: 180,
		Mailboxes: map[string]int64{"gmail": mb[0], "outlook": mb[1], "yahoo": mb[2], "icloud": mb[3], "other": mb[4]}}
	postal := r.postal(4791, 10)
	f.Route = readiness.RouteFacts{Stream: "broadcast", Policy: "pause_and_ask", Shares: []readiness.ShareFacts{r.ses(43121, 90), postal}}
	f.Content = readiness.ContentFacts{Checked: true, Report: chk.ContentCheckReport, Address: "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105",
		InlineCSSBytes: 9 * 1024, URIBlocklists: readiness.URIBlocklists, OwnDomain: "acme.io"}
	res, runID, err := r.storeRun(ctx, id, f, ranAt, &daniel.ID, readiness.TriggerApproval)
	if err != nil {
		return err
	}
	// Daniel's request, two hours ago, with Linh's comment an hour later.
	snap := approvals.Snapshot{Version: len(labels), ContentHash: hash, Subject: novemberSubject, Preheader: novemberPreheader, SenderID: sender.String(),
		From: "Acme Newsletter <product@news.acme.io>", Links: approvals.Links(src, html), Sections: approvals.Sections(src), Audience: audience,
		AudienceLabels: []string{"Weekly Digest", "Product Updates"}, AudienceCount: 47912, Suppressed: 301, Role: 27, Schedule: scheduleJSON,
		Thresholds: readiness.ParseThresholds(json.RawMessage(readinessThresholds), nil), AuthorRole: "Content Writer",
		Rules: []approvals.ApprovalRuleMatch{
			{Key: approvals.RuleAudienceMin, Label: "Audience of 47,912 is at least 10,000", Short: "audience ≥ 10,000", Badge: "audience ≥ 10,000"},
			{Key: approvals.RuleWarmingProvider, Label: "Route includes Postal, which is warming up (day 9 of 21)", Short: "warming-up provider", Badge: "Warming-up provider"},
			{Key: approvals.RuleAuthorLacksSend, Label: "Daniel Park doesn't hold campaigns:send · Content Writer", Short: "author lacks send", Badge: "author lacks send", Author: "Daniel Park", Role: "Content Writer"}}}
	r.snapshotRun(&snap, res, runID, ranAt)
	if err := r.routeKey(ctx, &snap); err != nil {
		return err
	}
	note := "The Postal share is intentional: it's day 9 of the warm-up and the planner caps it at 4,800 today. I kept the shortener link because marketing tracks it; happy to swap it if you prefer."
	reqID := ID()
	if err := r.insertRequest(ctx, reqID, id, daniel.ID, requested, note, "pending", snap, &runID, nil, nil); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO approval_comments (id, request_id, author_id, body, created_at) VALUES ($1,$2,$3,$4,$5)`,
		ID(), reqID, linh.ID, "Marcus, can you take this one? I'm out until Monday.", s.Ago(time.Hour)); err != nil {
		return err
	}
	// Marcus hears about it.
	marcus := s.Users["marcus@acme.io"]
	_, err = s.Tx.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, created_at)
		VALUES ($1,$2,$3,'approval_requested','warn','Approval requested',$4,$5,$6,$7)`, ID(), marcus.ID, ws.ID,
		"Daniel Park asked you to approve “November Product Update” · 47,912 recipients",
		"Daniel Park asked for approval: “November Product Update” · 47,912 recipients · rules: audience ≥ 10,000, warming-up provider, author lacks send",
		"/w/"+ws.Slug+"/approvals/"+reqID.String(), requested)
	return err
}

// snapshotRun records the run in a snapshot.
func (r *readinessSeed) snapshotRun(snap *approvals.Snapshot, res readiness.ReadinessResult, runID uuid.UUID, ranAt time.Time) {
	e := res.Estimates
	snap.Minutes, snap.Cost = e.Route.TotalMinutes, e.Route.TotalCost
	for _, sh := range e.Route.Shares {
		snap.Route = append(snap.Route, approvals.SnapshotShare{ProviderID: sh.ProviderID, Name: sh.Name, Short: sh.Short, Label: sh.Label, Adapter: sh.Adapter, Percent: sh.Percent,
			Recipients: sh.Recipients, Cost: sh.Cost, PriceLabel: sh.PriceLabel, Status: sh.Status, StatusLabel: sh.StatusLabel, WarmupDay: sh.WarmupDay, WarmupDays: sh.WarmupDays})
	}
	snap.Readiness = approvals.SnapshotReadiness{RunID: runID.String(), Verdict: res.Verdict, Passed: res.Counts.Passed, Warnings: res.Counts.Warnings, Blocks: res.Counts.Blocks,
		Total: res.Counts.Total, RanAt: ranAt, Issues: []readiness.ReadinessCheck{}}
	for _, ch := range res.Checks {
		if ch.Status != readiness.Pass {
			snap.Readiness.Issues = append(snap.Readiness.Issues, ch)
		}
	}
}

// routeKey is the routing an approval covers (the Broadcasts split).
func (r *readinessSeed) routeKey(ctx context.Context, snap *approvals.Snapshot) error {
	rc, err := (&readiness.Service{Q: r.s.Q}).Route(ctx, &r.ws, nil)
	if err != nil {
		return err
	}
	snap.RouteKey, snap.SplitLabel = rc.Key, rc.StreamLabel
	return nil
}

func (r *readinessSeed) insertRequest(ctx context.Context, id, campaign, by uuid.UUID, at time.Time, note, status string, snap approvals.Snapshot, runID *uuid.UUID,
	decidedBy *uuid.UUID, decidedAt *time.Time) error {
	if snap.Route == nil {
		snap.Route = []approvals.SnapshotShare{}
	}
	sb, _ := json.Marshal(snap)
	keys := []string{}
	for _, rl := range snap.Rules {
		keys = append(keys, rl.Key)
	}
	version := int32(snap.Version)
	_, err := r.s.Tx.Exec(ctx, `INSERT INTO approval_requests (id, workspace_id, subject_type, campaign_id, requested_by, requested_at, note, status, matched_rules, approvals_required,
		snapshot, campaign_version, readiness_run_id, decided_by, decided_at) VALUES ($1,$2,'campaign',$3,$4,$5,$6,$7,$8,1,$9,$10,$11,$12,$13)`,
		id, r.ws.ID, campaign, by, at, note, status, keys, sb, version, runID, decidedBy, decidedAt)
	return err
}

func (r *readinessSeed) decide(ctx context.Context, request uuid.UUID, by store.User, decision, comment string, at time.Time) error {
	_, err := r.s.Tx.Exec(ctx, `INSERT INTO approval_decisions (id, request_id, approver_id, decision, comment, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		ID(), request, by.ID, decision, comment, at)
	return err
}

// ---------------------------------------------------------------- inbox history

func (r *readinessSeed) history(ctx context.Context) error {
	s := r.s
	ws := r.ws
	sofia, daniel, marcus, linh := s.Users["sofia@acme.io"], s.Users["daniel@acme.io"], s.Users["marcus@acme.io"], s.Users["linh@acme.io"]
	if err := r.routeKeyCheck(ctx); err != nil {
		return err
	}
	ses := s.Providers["ses"].String()
	postal := s.Providers["postal"].String()
	share := func(id, name, short, adapter string, n int64, pct float64, cost float64, price, status, label string) approvals.SnapshotShare {
		return approvals.SnapshotShare{ProviderID: id, Name: name, Short: short, Adapter: adapter, Percent: pct, Recipients: n, Cost: cost, PriceLabel: price, Status: status, StatusLabel: label}
	}
	sesShare := func(n int64, pct float64) approvals.SnapshotShare {
		return share(ses, "Amazon SES", "SES", "ses", n, pct, float64(n)/10000, "$0.10 per 1,000", "healthy", "Healthy")
	}
	postalShare := func(n int64, pct float64, day int) approvals.SnapshotShare {
		return share(postal, "Postal", "Postal", "postal", n, pct, 0, "self-hosted", "warming_up", fmt.Sprintf("Warming up · day %d of 21", day))
	}
	audienceRule := approvals.ApprovalRuleMatch{Key: approvals.RuleAudienceMin, Short: "audience ≥ 10,000", Badge: "audience ≥ 10,000"}
	warmingRule := approvals.ApprovalRuleMatch{Key: approvals.RuleWarmingProvider, Short: "warming-up provider", Badge: "Warming-up provider"}
	lacksRule := approvals.ApprovalRuleMatch{Key: approvals.RuleAuthorLacksSend, Short: "author lacks send", Badge: "author lacks send", Author: "Daniel Park", Role: "Content Writer"}
	ip := map[string]string{"marcus@acme.io": "198.51.100.40", "linh@acme.io": "203.0.113.9"}

	// Weekly Digest #142: Sofia's request, approved by Marcus on Oct 10.
	if id, ok := s.Campaigns["Weekly Digest #142"]; ok {
		c, err := s.Q.ReadinessCampaign(ctx, store.ReadinessCampaignParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		requested, decided := s.designDay(2, 13, 40), s.designDay(2, 16, 20)
		audienceRule.Label = "Audience of 31,604 is at least 10,000"
		snap := approvals.Snapshot{Version: int(c.Version), ContentHash: c.ContentHash, Subject: c.Subject, Preheader: c.Preheader, From: "Acme Newsletter <digest@news.acme.io>",
			Links: approvals.Links(c.Mjml, c.Html), Sections: approvals.Sections(c.Mjml), Audience: c.Audience, AudienceLabels: []string{"Weekly Digest"}, AudienceCount: 31604,
			Suppressed: 331, Role: 27, Schedule: c.Schedule, AuthorRole: "Editor", Rules: []approvals.ApprovalRuleMatch{audienceRule},
			Route: []approvals.SnapshotShare{sesShare(28444, 90), postalShare(3160, 10, 7)}, Minutes: 34, Cost: 2.84,
			Thresholds: readiness.ParseThresholds(json.RawMessage(readinessThresholds), nil)}
		if c.SenderID != nil {
			snap.SenderID = c.SenderID.String()
		}
		if err := r.routeKey(ctx, &snap); err != nil {
			return err
		}
		snap.Readiness = approvals.SnapshotReadiness{Verdict: readiness.VerdictWarnings, Passed: 23, Warnings: 2, Total: 25, RanAt: requested.Add(-3 * time.Minute), Issues: []readiness.ReadinessCheck{}}
		reqID := ID()
		if err := r.insertRequest(ctx, reqID, id, sofia.ID, requested, "", "approved", snap, nil, &marcus.ID, &decided); err != nil {
			return err
		}
		if err := r.decide(ctx, reqID, marcus, approvals.DecisionApprove, "", decided); err != nil {
			return err
		}
		if err := readinessAudit(ctx, s, decided, marcus, "campaign.approved", "campaign", id.String(), c.Name,
			map[string]any{"status": "approved"}, map[string]any{"request_id": reqID.String(), "approvals": "1 of 1"}, ip["marcus@acme.io"]); err != nil {
			return err
		}
	}

	// October Product Update: Sofia, approved by Linh on Oct 8, sent Oct 9.
	if id, ok := s.Campaigns["October Product Update"]; ok {
		var started time.Time
		var subject string
		var aud json.RawMessage
		if err := s.Tx.QueryRow(ctx, `SELECT coalesce(started_at, now()), subject, audience FROM campaigns WHERE id = $1`, id).Scan(&started, &subject, &aud); err != nil {
			return err
		}
		requested, decided := started.Add(-26*time.Hour+22*time.Minute), started.Add(-23*time.Hour+2*time.Minute)
		audienceRule.Label = "Audience of 47,980 is at least 10,000"
		warmingRule.Label = "Route includes Postal, which is warming up (day 6 of 21)"
		snap := approvals.Snapshot{Version: 1, Subject: subject, From: "Acme Newsletter <digest@news.acme.io>", Audience: aud, AudienceLabels: []string{"Weekly Digest", "Product Updates", "Beta Program"},
			AudienceCount: 47980, Schedule: json.RawMessage(fmt.Sprintf(`{"mode":"at","at":%q,"timezone":"America/New_York"}`, started.UTC().Format(time.RFC3339))),
			AuthorRole: "Editor", Rules: []approvals.ApprovalRuleMatch{audienceRule, warmingRule}, Route: []approvals.SnapshotShare{sesShare(43182, 90), postalShare(4798, 10, 6)},
			Links: []string{}, Sections: []string{}}
		reqID := ID()
		if err := r.insertRequest(ctx, reqID, id, sofia.ID, requested, "", "approved", snap, nil, &linh.ID, &decided); err != nil {
			return err
		}
		if err := r.decide(ctx, reqID, linh, approvals.DecisionApprove, "", decided); err != nil {
			return err
		}
		if err := readinessAudit(ctx, s, decided, linh, "campaign.approved", "campaign", id.String(), "October Product Update",
			map[string]any{"status": "approved"}, map[string]any{"request_id": reqID.String(), "approvals": "1 of 1"}, ip["linh@acme.io"]); err != nil {
			return err
		}
	}

	// Beta Program · launch invite: sent through Postal alone on Oct 6.
	betaAud := map[string]any{"lists": []uuid.UUID{s.Lists["Beta Program"]}}
	if err := seedSent(ctx, s, sentCampaign{name: "Beta Program · launch invite", audience: "Beta Program", at: 6, hour: 10, perProvider: map[string]int64{"postal": 2113},
		delivered: 99.5, humanOpens: 54.6, machineOpens: 16.8, clicks: 12.4, bounce: 0.3, complaint: 0, unsubscribes: 0.05}, betaAud); err != nil {
		return err
	}
	if id, ok := s.Campaigns["Beta Program · launch invite"]; ok {
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET created_by = $2, updated_by = $2, subject = $3 WHERE id = $1`, id, sofia.ID, "You're invited: the Acme beta launch"); err != nil {
			return err
		}
		requested, decided := s.designDay(6, 7, 8), s.designDay(6, 9, 48)
		warmingRule.Label = "Route includes Postal, which is warming up (day 3 of 21)"
		aud, _ := json.Marshal(betaAud)
		snap := approvals.Snapshot{Version: 1, Subject: "You're invited: the Acme beta launch", From: "Acme Newsletter <digest@news.acme.io>", Audience: aud, AudienceLabels: []string{"Beta Program"},
			AudienceCount: 2113, Schedule: json.RawMessage(fmt.Sprintf(`{"mode":"at","at":%q,"timezone":"America/New_York"}`, s.designDay(6, 10, 0).UTC().Format(time.RFC3339))),
			AuthorRole: "Editor", Rules: []approvals.ApprovalRuleMatch{warmingRule}, Route: []approvals.SnapshotShare{postalShare(2113, 100, 3)}, Links: []string{}, Sections: []string{}}
		reqID := ID()
		if err := r.insertRequest(ctx, reqID, id, sofia.ID, requested, "", "approved", snap, nil, &linh.ID, &decided); err != nil {
			return err
		}
		if err := r.decide(ctx, reqID, linh, approvals.DecisionApprove, "", decided); err != nil {
			return err
		}
		if err := readinessAudit(ctx, s, decided, linh, "campaign.approved", "campaign", id.String(), "Beta Program · launch invite",
			map[string]any{"status": "approved"}, map[string]any{"request_id": reqID.String(), "approvals": "1 of 1"}, ip["linh@acme.io"]); err != nil {
			return err
		}
	}

	// Daniel's drafts: Re-engagement wave 2 (changes requested by Marcus) and
	// Partner webinar (approved by Linh, invalidated when the audience grew 14%).
	drafts := []struct {
		name, subject, preheader string
		list                     string
		count                    int64
		created                  time.Duration
	}{
		{"Re-engagement wave 2", "Still want to hear from us?", "One click keeps you on the list; no click and we'll stop.", "Weekly Digest", 12400, 9 * 24 * time.Hour},
		{"Partner webinar", "Live: building on the Acme API with our partners", "Thursday 4 pm ET · 45 minutes · questions welcome", "Product Updates", 17110, 11 * 24 * time.Hour},
	}
	for _, d := range drafts {
		if _, ok := s.Campaigns[d.name]; ok {
			continue
		}
		id := ID()
		aud, _ := json.Marshal(map[string]any{"lists": []string{s.Lists[d.list].String()}})
		src := r.withImages(simpleDraftMJML(d.subject, d.preheader))
		res := mjml.Compile(src)
		if len(res.Errors) > 0 {
			return fmt.Errorf("compile %s: %s", d.name, res.Errors[0].Message)
		}
		created := s.Ago(d.created)
		hash := campaigns.ContentHash(d.subject, d.preheader, src, res.HTML, nil)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, audience_count, mjml, html, plain_text, content_hash, version,
			thresholds, created_by, updated_by, created_at, updated_at, autosaved_at)
			VALUES ($1,$2,$3,'draft',$4,$5,$6,$7,$8,$9,$10,$11,2,$12,$13,$13,$14,$15,$15)`,
			id, ws.ID, d.name, d.subject, d.preheader, aud, d.count, src, res.HTML, contentcheck.PlainText(res.HTML), hash, json.RawMessage(readinessThresholds), daniel.ID,
			created, created.Add(26*time.Hour)); err != nil {
			return err
		}
		for v := 1; v <= 2; v++ {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, saved_by, saved_at, label, source, size_bytes)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, id, v, d.subject, d.preheader, src, res.HTML, contentcheck.PlainText(res.HTML), daniel.ID,
				created.Add(time.Duration(v-1)*26*time.Hour), map[int]string{1: "created", 2: "copy"}[v], map[int]string{1: "create", 2: "autosave"}[v], len(res.HTML)); err != nil {
				return err
			}
		}
		s.Campaigns[d.name] = id
		var sender uuid.UUID
		if v, ok := s.Senders["digest@news.acme.io"]; ok {
			sender = v
			if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET sender_id = $2 WHERE id = $1`, id, sender); err != nil {
				return err
			}
		}
		reqID := ID()
		switch d.name {
		case "Re-engagement wave 2":
			requested, decided := s.designDay(7, 10, 15), s.designDay(7, 14, 30)
			snap := approvals.Snapshot{Version: 2, ContentHash: hash, Subject: d.subject, Preheader: d.preheader, From: "Acme Newsletter <digest@news.acme.io>", Audience: aud,
				AudienceLabels: []string{d.list}, AudienceCount: d.count, Schedule: json.RawMessage(`{"mode":"now"}`), AuthorRole: "Content Writer",
				Rules: []approvals.ApprovalRuleMatch{withLabel(lacksRule, "Daniel Park doesn't hold campaigns:send · Content Writer")}, Route: []approvals.SnapshotShare{sesShare(d.count, 100)},
				Links: approvals.Links(src, res.HTML), Sections: approvals.Sections(src)}
			if err := r.insertRequest(ctx, reqID, id, daniel.ID, requested, "First wave went well; this one adds the Weekly Digest readers who stopped opening.", "changes_requested", snap, nil, &marcus.ID, &decided); err != nil {
				return err
			}
			if err := r.decide(ctx, reqID, marcus, approvals.DecisionRequestChanges, "exclude 180-day unengaged", decided); err != nil {
				return err
			}
			if err := readinessAudit(ctx, s, decided, marcus, "campaign.changes_requested", "campaign", id.String(), d.name,
				map[string]any{"status": "changes_requested", "comment": "exclude 180-day unengaged"}, map[string]any{"request_id": reqID.String()}, ip["marcus@acme.io"]); err != nil {
				return err
			}
		case "Partner webinar":
			requested, decided, invalidated := s.designDay(9, 9, 5), s.designDay(9, 10, 40), s.designDay(9, 11, 30)
			snap := approvals.Snapshot{Version: 2, ContentHash: hash, Subject: d.subject, Preheader: d.preheader, From: "Acme Newsletter <digest@news.acme.io>", Audience: aud,
				AudienceLabels: []string{d.list}, AudienceCount: 15002, Schedule: json.RawMessage(`{"mode":"now"}`), AuthorRole: "Content Writer",
				Rules: []approvals.ApprovalRuleMatch{withLabel(audienceRule, "Audience of 15,002 is at least 10,000")}, Route: []approvals.SnapshotShare{sesShare(15002, 100)},
				Links: approvals.Links(src, res.HTML), Sections: approvals.Sections(src)}
			if err := r.insertRequest(ctx, reqID, id, daniel.ID, requested, "", "approved", snap, nil, &linh.ID, &decided); err != nil {
				return err
			}
			if err := r.decide(ctx, reqID, linh, approvals.DecisionApprove, "", decided); err != nil {
				return err
			}
			changes, _ := json.Marshal([]approvals.ApprovalChange{{Field: "audience_count", Label: "Audience", Before: "15,002", After: "17,110 (+14%)"}})
			if _, err := s.Tx.Exec(ctx, `UPDATE approval_requests SET status = 'invalidated', invalidated_at = $2, invalidated_reason = 'audience moved +14% after approval',
				changes = $3, invalidated_from = 'approved' WHERE id = $1`, reqID, invalidated, changes); err != nil {
				return err
			}
			if err := readinessAudit(ctx, s, decided, linh, "campaign.approved", "campaign", id.String(), d.name,
				map[string]any{"status": "approved"}, map[string]any{"request_id": reqID.String(), "approvals": "1 of 1"}, ip["linh@acme.io"]); err != nil {
				return err
			}
			if _, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_name, action, target_type, target_id, target_name, result, reason, after, metadata)
				VALUES ($1,$2,$3,'system','System','approval.invalidated','approval_request',$4,$5,'allowed','audience moved +14% after approval',$6,$7)`,
				ID(), ws.ID, invalidated, reqID.String(), d.name, json.RawMessage(`{"status":"invalidated"}`),
				json.RawMessage(fmt.Sprintf(`{"campaign_id":%q,"summary":"audience moved +14%% after approval"}`, id))); err != nil {
				return err
			}
		}
	}
	return nil
}

func withLabel(m approvals.ApprovalRuleMatch, label string) approvals.ApprovalRuleMatch {
	m.Label = label
	return m
}

// routeKeyCheck makes sure the Broadcasts routing exists before snapshots refer to it.
func (r *readinessSeed) routeKeyCheck(ctx context.Context) error {
	var n int
	return r.s.Tx.QueryRow(ctx, `SELECT count(*) FROM routing_rules WHERE workspace_id = $1 AND stream = 'broadcast'`, r.ws.ID).Scan(&n)
}

// simpleDraftMJML is a short single-section email.
func simpleDraftMJML(subject, preheader string) string {
	return `<mjml>
  <mj-head>
    <mj-title>` + composerXMLText(subject) + `</mj-title>
    <mj-preview>` + composerXMLText(preheader) + `</mj-preview>
    <mj-attributes>
      <mj-all font-family="` + composerPlexFont + `"></mj-all>
      <mj-text color="#1F2937" padding="0px"></mj-text>
    </mj-attributes>
  </mj-head>
  <mj-body width="600px" background-color="#F3F4F6">
` + composerDigestHeader + `
    <mj-section background-color="#ffffff" padding="20px 24px 16px">
      <mj-column>
        <mj-text font-size="22px" line-height="1.25" font-weight="700" color="#0B1220" padding="0px 0px 10px">` + composerXMLText(subject) + `</mj-text>
        <mj-text font-size="15px" line-height="1.55" padding="0px 0px 14px">Hi {{ first_name | default: "there" }}, ` + composerXMLText(preheader) + `</mj-text>
        ` + composerButton("https://acme.io/events", "Count me in") + `
      </mj-column>
    </mj-section>
` + composerDigestFooter + `
  </mj-body>
</mjml>
`
}
