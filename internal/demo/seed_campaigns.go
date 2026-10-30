package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/mailbox"
)

func init() { seedCampaigns = seedCampaignsImpl }

// MailboxShare is the sample split of recipients by mailbox provider.
var MailboxShare = []struct {
	Key   string
	Share float64
}{{"gmail", 0.46}, {"outlook", 0.21}, {"yahoo", 0.09}, {"icloud", 0.08}, {"other", 0.16}}

// The October Product Update's subject test, as on the Report board: A is
// the campaign's subject (and the email's headline), B won.
const (
	OctoberSubjectA = "October Product Update: three launches"
	OctoberSubjectB = "What shipped in October, and what we're retiring"
)

var octoberVariants = []map[string]string{{"key": "A", "subject": OctoberSubjectA}, {"key": "B", "subject": OctoberSubjectB}}

type sentCampaign struct {
	name, audience, note string
	at                   time.Duration // before "design today" 10:00
	hour, minute         int
	perProvider          map[string]int64 // provider key → sent
	delivered            float64          // % of sent
	humanOpens           float64          // % of delivered
	machineOpens         float64
	clicks               float64
	bounce, complaint    float64 // % of sent / delivered
	unsubscribes         float64
	ab                   string
}

// designDay returns hour:minute (New York time) daysAgo days before the
// demo's "today". The prototype's clock reads Sunday 12 Oct 2026, 11:00; the
// demo anchors dates to the real calendar instead. "Today" is the current
// New York date from 11:00 on and the day before until then, so every time
// the boards show for today (08:55, 09:44…) is always in the past.
func (s *Seeder) designDay(daysAgo int, hour, minute int) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	n := s.Now.In(loc).Add(-11 * time.Hour)
	d := time.Date(n.Year(), n.Month(), n.Day(), hour, minute, 0, 0, loc)
	return d.AddDate(0, 0, -daysAgo)
}

func seedCampaignsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	digest, updates, beta := s.Lists["Weekly Digest"], s.Lists["Product Updates"], s.Lists["Beta Program"]
	all := map[string]any{"lists": []uuid.UUID{digest, updates, beta}}
	audiences := map[string]map[string]any{
		"Weekly Digest":   {"lists": []uuid.UUID{digest}},
		"Product Updates": {"lists": []uuid.UUID{updates}},
		"Beta Program":    {"lists": []uuid.UUID{beta}},
		"All subscribers": all,
	}

	insert := func(name, status string, audience map[string]any, count *int64, extra map[string]any) (uuid.UUID, error) {
		id := ID()
		aud, _ := json.Marshal(audience)
		if audience == nil {
			aud = []byte(`{}`)
		}
		_, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, audience_count, mjml, html, created_by, updated_by, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'','',$9,$9,$10,$10)`,
			id, ws.ID, name, status, extra["subject"], extra["preheader"], aud, count, linh.ID, extra["created"])
		s.Campaigns[name] = id
		return id, err
	}

	// Scheduled: Weekly Digest #142, in 1 day 22 h: 31,604 after
	// suppressions, minus the 412 unengaged a send leaves out (SR-11).
	n := int64(31192)
	created := s.Ago(3 * 24 * time.Hour)
	id, err := insert("Weekly Digest #142", "scheduled", audiences["Weekly Digest"], &n, map[string]any{
		"subject": "Three launches, one deprecation, and the roadmap for Q4", "preheader": "Feature flags are GA, audit log exports, and what we're retiring in November.", "created": created,
	})
	if err != nil {
		return err
	}
	sched := s.designDay(-2, 9, 0)
	if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET scheduled_at = $2, schedule = $3, autosaved_at = $4, version = 14, updated_at = $4 WHERE id = $1`,
		id, sched, json.RawMessage(fmt.Sprintf(`{"mode":"at","at":%q,"timezone":"America/New_York"}`, sched.Format(time.RFC3339))), s.Ago(90*time.Minute)); err != nil {
		return err
	}
	if err := insertReadinessRun(ctx, s, id, "warnings", []map[string]string{
		{"key": "audience.unengaged", "group": "Audience quality", "status": "warn", "title": "412 unengaged contacts (no opens in 180 days) are in the audience"},
		{"key": "reputation.blocklist", "group": "Sender reputation", "status": "warn", "title": "Postal IP listed on UCEPROTECT L3, a low-impact list"},
	}); err != nil {
		return err
	}
	if err := planRoutes(ctx, s, id, n, map[string]float64{"ses": 0.9, "postal": 0.1}); err != nil {
		return err
	}

	// Blocked draft.
	n2 := int64(47842)
	id, err = insert("Flash sale: 48 hours only", "draft", audiences["All subscribers"], &n2, map[string]any{
		"subject": "48 hours only: 40% off every plan", "preheader": "Ends Thursday at midnight.", "created": s.Ago(26 * time.Hour),
	})
	if err != nil {
		return err
	}
	edited := s.designDay(0, 9, 31)
	if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET updated_at = $2, autosaved_at = $2 WHERE id = $1`, id, edited); err != nil {
		return err
	}
	if err := insertReadinessRun(ctx, s, id, "blocked", []map[string]string{
		{"key": "content.unsubscribe", "group": "Compliance", "status": "block", "title": "No unsubscribe link or postal address in the footer"},
		{"key": "links.blocklist", "group": "Links", "status": "block", "title": "go.acme-promo.link is on SURBL and Spamhaus DBL"},
		{"key": "content.alt_text", "group": "Content", "status": "block", "title": "Hero image has no alt text and the email is 96% image"},
	}); err != nil {
		return err
	}
	if err := planRoutes(ctx, s, id, n2, map[string]float64{"ses": 0.9, "postal": 0.1}); err != nil {
		return err
	}

	// Empty draft.
	if _, err = insert("Untitled campaign", "draft", nil, nil, map[string]any{"subject": "", "preheader": "", "created": s.designDay(0, 8, 55)}); err != nil {
		return err
	}

	sent := []sentCampaign{
		{name: "October Product Update", audience: "All subscribers", at: 3, hour: 10, perProvider: map[string]int64{"ses": 43182, "postal": 4798}, delivered: 99.1, humanOpens: 41.2, machineOpens: 18.6, clicks: 6.3, bounce: 0.6, complaint: 0.04, unsubscribes: 0.21, ab: "subject"},
		{name: "Weekly Digest #141", audience: "Weekly Digest", at: 5, hour: 9, perProvider: map[string]int64{"ses": 28368, "postal": 3152}, delivered: 99.3, humanOpens: 44.8, machineOpens: 17.9, clicks: 7.1, bounce: 0.5, complaint: 0.03, unsubscribes: 0.12},
		{name: "Feature flags preview", audience: "Beta Program", at: 10, hour: 15, perProvider: map[string]int64{"ses": 1888, "postal": 210}, delivered: 99.6, humanOpens: 58.2, machineOpens: 16.1, clicks: 14.9, bounce: 0.3, complaint: 0.0, unsubscribes: 0.05},
		{name: "Weekly Digest #140", audience: "Weekly Digest", at: 12, hour: 9, perProvider: map[string]int64{"ses": 28239, "postal": 3138}, delivered: 99.2, humanOpens: 43.9, machineOpens: 18.2, clicks: 6.8, bounce: 0.6, complaint: 0.04, unsubscribes: 0.14},
		{name: "Weekly Digest #139", audience: "Weekly Digest", at: 19, hour: 9, perProvider: map[string]int64{"ses": 31204}, delivered: 98.9, humanOpens: 42.1, machineOpens: 18.4, clicks: 6.2, bounce: 0.8, complaint: 0.05, unsubscribes: 0.15},
		{name: "Welcome to news.acme.io", audience: "All subscribers", note: "first send after Brevo migration", at: 24, hour: 11, perProvider: map[string]int64{"ses": 46118}, delivered: 98.4, humanOpens: 38.0, machineOpens: 19.5, clicks: 4.9, bounce: 1.1, complaint: 0.06, unsubscribes: 0.42},
	}
	for _, c := range sent {
		if err := seedSent(ctx, s, c, audiences[c.audience]); err != nil {
			return err
		}
	}
	// 39 earlier digests sent from Brevo before the migration, imported with their stats.
	for i := 0; i < 39; i++ {
		num := 138 - i
		c := sentCampaign{
			name: fmt.Sprintf("Weekly Digest #%d", num), audience: "Weekly Digest", at: time.Duration(26 + 7*i), hour: 9,
			perProvider: map[string]int64{"brevo": int64(29000 - i*35)}, delivered: 98.8, humanOpens: 40.5 - float64(i%5)*0.4, machineOpens: 18, clicks: 6.0 - float64(i%4)*0.2, bounce: 0.9, complaint: 0.05, unsubscribes: 0.2,
		}
		if err := seedSent(ctx, s, c, audiences["Weekly Digest"]); err != nil {
			return err
		}
	}
	return nil
}

func seedSent(ctx context.Context, s *Seeder, c sentCampaign, audience map[string]any) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	var total int64
	for _, v := range c.perProvider {
		total += v
	}
	start := s.designDay(int(c.at), c.hour, c.minute)
	done := start.Add(time.Duration(float64(total)/14/60) * time.Minute)
	aud, _ := json.Marshal(audience)
	id := ID()
	var ab any
	if c.ab != "" {
		raw, err := json.Marshal(map[string]any{"kind": "subject", "variants": octoberVariants, "test_percent": 20, "metric": "clicks", "duration_hours": 4, "winner": "B"})
		if err != nil {
			return err
		}
		ab = json.RawMessage(raw)
	}
	origin, note := "", c.note
	if _, imported := c.perProvider["brevo"]; imported {
		origin, note = "brevo", ""
	}
	// Gmail and iCloud never report complaints (review F-2). The report
	// shaping (seed_reports.go) moves the complaints of the sends it shapes
	// to the mailbox providers that report them; history imported from
	// Brevo and sends seeded after it keep their cells as seeded here.
	asSeeded := origin != "" || s.reportsShaped
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, audience_count, ab_test, schedule,
		created_by, updated_by, created_at, updated_at, scheduled_at, started_by, started_at, completed_at, origin, note)
		VALUES ($1,$2,$3,'sent',$4,'',$5,$6,$7,$8,$9,$9,$10,$11,$12,$9,$12,$11,$13,$14)`,
		id, ws.ID, c.name, c.name, aud, total, ab, json.RawMessage(`{"mode":"at"}`), linh.ID, start.Add(-72*time.Hour), done, start, origin, note); err != nil {
		return err
	}
	if c.name == "Weekly Digest #139" {
		// The auto-pause that shows as "auto-paused once, Yahoo deferrals".
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_incidents (id, campaign_id, kind, scope, mailbox_provider, metric, value, threshold, held, tripped_at, resolved_at, resolution, resolved_by, provider_id)
			VALUES ($1,$2,'threshold','mailbox_provider','yahoo','deferral',6.4,5.0,2810,$3,$4,'resumed',$5,$6)`,
			ID(), id, start.Add(38*time.Minute), start.Add(64*time.Minute), linh.ID, s.Providers["ses"]); err != nil {
			return err
		}
	}
	s.Campaigns[c.name] = id
	for key, n := range c.perProvider {
		pid := s.Providers[key]
		if key != "brevo" {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_routes (campaign_id, provider_id, planned, status, cost_micros) VALUES ($1,$2,$3,'done',$4)`,
				id, pid, n, costMicros(key, n)); err != nil {
				return err
			}
		}
		shares := make([]float64, len(MailboxShare))
		for i, mb := range MailboxShare {
			shares[i] = mb.Share
		}
		perMailbox := Split(n, shares)
		for i, mb := range MailboxShare {
			sentN := perMailbox[i]
			delivered := int64(math.Round(float64(sentN) * c.delivered / 100))
			bounced := int64(math.Round(float64(sentN) * c.bounce / 100))
			hard := bounced * 2 / 3
			soft := bounced - hard
			complained := int64(math.Round(float64(delivered) * c.complaint / 100))
			if asSeeded && !mailbox.ReportsComplaints(mb.Key) {
				complained = 0
			}
			human := int64(math.Round(float64(delivered) * c.humanOpens / 100))
			// machineOpens are every open no person made: most of them Apple
			// Mail's privacy prefetches, the rest scanners'.
			opened := int64(math.Round(float64(delivered) * c.machineOpens / 100))
			privacy := int64(math.Round(float64(opened) * privacyShare(mb.Key)))
			machine := opened - privacy
			clicked := int64(math.Round(float64(delivered) * c.clicks / 100))
			unsub := int64(math.Round(float64(delivered) * c.unsubscribes / 100))
			if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_counters (campaign_id, provider_id, mailbox_provider, sent, delivered, hard_bounced, soft_bounced, complained,
				opened_human, opened_privacy, opened_machine, clicked, unsubscribed)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, id, pid, mb.Key, sentN, delivered, hard, soft, complained, human, privacy, machine, clicked, unsub); err != nil {
				return err
			}
		}
	}
	return nil
}

// privacyShare is the part of a mailbox provider's machine opens that are
// Apple Mail's privacy prefetches (the rest are scanners'), as in the
// October Product Update's report.
func privacyShare(mailbox string) float64 {
	if mailbox == "icloud" {
		return 0.97
	}
	return 0.86
}

// costMicros is the sample cost of a send: SES $0.10 per 1,000, the rest free.
func costMicros(provider string, n int64) int64 {
	if provider == "ses" {
		return n * 100 // $0.0001 per email = 100 micros
	}
	return 0
}

func planRoutes(ctx context.Context, s *Seeder, campaign uuid.UUID, total int64, split map[string]float64) error {
	for key, share := range split {
		n := int64(math.Round(float64(total) * share))
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_routes (campaign_id, provider_id, planned, status, cost_micros) VALUES ($1,$2,$3,'pending',$4)`,
			campaign, s.Providers[key], n, costMicros(key, n)); err != nil {
			return err
		}
	}
	return nil
}

func insertReadinessRun(ctx context.Context, s *Seeder, campaign uuid.UUID, verdict string, issues []map[string]string) error {
	checks, _ := json.Marshal(map[string]any{"checks": issues})
	_, err := s.Tx.Exec(ctx, `INSERT INTO readiness_runs (id, campaign_id, trigger, verdict, checks, ran_at) VALUES ($1,$2,'composer',$3,$4,$5)`,
		ID(), campaign, verdict, checks, s.Ago(40*time.Minute))
	return err
}

// Split divides n by weights so the parts add up to exactly n (largest
// remainder method), keeping demo totals consistent across breakdowns.
func Split(n int64, weights []float64) []int64 {
	var total float64
	for _, w := range weights {
		total += w
	}
	out := make([]int64, len(weights))
	if total == 0 || n == 0 {
		return out
	}
	type rem struct {
		i int
		r float64
	}
	rems := make([]rem, len(weights))
	var used int64
	for i, w := range weights {
		exact := float64(n) * w / total
		out[i] = int64(math.Floor(exact))
		used += out[i]
		rems[i] = rem{i, exact - math.Floor(exact)}
	}
	for left := n - used; left > 0; left-- {
		best := 0
		for j := range rems {
			if rems[j].r > rems[best].r {
				best = j
			}
		}
		out[rems[best].i]++
		rems[best].r = -1
	}
	return out
}
