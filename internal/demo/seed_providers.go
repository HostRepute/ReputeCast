package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/warmup"
)

func init() { seedProviders = seedProvidersImpl }

// The sending providers of the sample data (spec §6, boards Providers*):
// Amazon SES us-east-1 (Broadcasts 90%, Automations primary, System
// fallback), Postmark (System primary, Automations fallback) and a
// self-hosted Postal server on a dedicated IP, day 9 of a 21-day warm-up
// (Broadcasts 10%). The healthy multi-provider setup is the default; the
// outage, webhooks-silent and bulk-warning boards are reproduced by data
// changes (see internal/providers/manage tests and docs/dev/providers-qa.sql).
//
// Every demo provider carries config "demo": true so background jobs never
// call a real provider API with the sample credentials.

type demoProvider struct {
	key, adapter, name, label, status, detail, ipMode string
	ips                                               []string
	config                                            map[string]any
	caps                                              state.Snapshot
	limits                                            state.Limits
	health                                            state.Health
	pricing                                           state.Pricing
	tracking                                          state.Tracking
	connected, lastEvent                              time.Duration
}

func seedProvidersImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	loc := state.Location(ws.Timezone)
	checked := s.Ago(time.Minute)
	sesCaps := capsOf("ses")
	pmCaps := capsOf("postmark")
	pmCaps.Pricing = "plan, $15 per month"
	postalCaps := capsOf("postal")
	provs := []demoProvider{
		{
			key: "ses", adapter: "ses", name: "Amazon SES", label: "us-east-1", status: state.Healthy, detail: "Production · no enforcement", ipMode: "shared",
			config: map[string]any{"demo": true, "region": "us-east-1", "configuration_set": "rc-broadcasts", "access": "role",
				"role_arn": "arn:aws:iam::123456789012:role/reputecast-ses", "topic_arns": []string{"arn:aws:sns:us-east-1:123456789012:reputecast-acme-newsletter"}},
			caps:   sesCaps,
			limits: state.Limits{Bounce: state.Limit{Limit: 5, Review: 5, Pause: 10, Label: "SES"}, Complaint: state.Limit{Limit: 0.1, Review: 0.1, Pause: 0.5, Label: "SES"}},
			health: state.Health{Quota: &state.Quota{Kind: "rolling_24h", Max: 50000, Used: 2140}, Rate: &state.Rate{Max: 14, PeakToday: 14},
				Account: "Production · no enforcement", LastSuccessAt: &checked, LastSuccessOp: "GetAccount"},
			pricing:   state.Pricing{Model: "metered", Per1000: 0.10, Source: "AWS pricing"},
			tracking:  state.Tracking{Detail: "Configuration set rc-broadcasts: open and click tracking off. Custom redirect domain not needed.", Scope: "Configuration set rc-broadcasts", Mode: "reputecast", CheckedAt: &checked},
			connected: 210 * 24 * time.Hour, lastEvent: 2 * time.Minute,
		},
		{
			key: "postmark", adapter: "postmark", name: "Postmark", label: "Transactional stream", status: state.Healthy, detail: "Approved · 10,000 per month plan", ipMode: "shared",
			config: map[string]any{"demo": true, "server": "Acme Newsletter", "server_id": 4821, "message_stream": "outbound", "monthly_limit": 10000},
			caps:   pmCaps,
			limits: state.Limits{Bounce: state.Limit{Limit: 10, Label: "Postmark"}, Complaint: state.Limit{Limit: 0.1, Label: "Postmark"}},
			health: state.Health{Quota: &state.Quota{Kind: "monthly_plan", Max: 10000, Used: 3412}, Rate: &state.Rate{PeakToday: 3},
				Account: "Approved · 10,000 per month plan", LastSuccessAt: &checked, LastSuccessOp: "GET /server",
				// Postmark reports its own rates (outbound stats).
				Reputation: &state.Reputation{Bounce: 0.2, Complaint: 0.01, Window: "7 d", CheckedAt: checked}},
			pricing: state.Pricing{Model: "plan", Monthly: 15, NextTier: &state.NextTier{Adds: 50000, Price: 115}},
			tracking: state.Tracking{OpenPixel: true, LinkRewrite: true, Scope: "Transactional stream", Mode: "reputecast", CheckedAt: &checked,
				Detail: "Server setting TrackOpens is on, and TrackLinks is HtmlAndText. Opens on System and Automations mail would count twice, and links get rewritten to click.pstmrk.it before ReputeCast's redirect."},
			connected: 210 * 24 * time.Hour, lastEvent: 6 * time.Minute,
		},
		{
			key: "postal", adapter: "postal", name: "Postal", label: "self-hosted · mta.acme.io", status: state.WarmingUp, detail: "Warming up · day 9 of 21", ipMode: "dedicated", ips: []string{"203.0.113.25"},
			config:    map[string]any{"demo": true, "server_url": "https://mta.acme.io", "version": "3.3.4"},
			caps:      postalCaps,
			limits:    state.Limits{Bounce: state.Limit{Limit: 2, Label: "warm-up guard"}, Complaint: state.Limit{Limit: 0.1}},
			health:    state.Health{Account: "Postal 3.3.4 · reachable", Version: "3.3.4", LastSuccessAt: &checked, LastSuccessOp: "POST /api/v1/messages/message"},
			pricing:   state.Pricing{Model: "self_hosted", Monthly: 24},
			tracking:  state.Tracking{Detail: "Track clicks and opens disabled on the mail server; no link rewriting.", Scope: "mail server mta.acme.io", Mode: "reputecast", CheckedAt: &checked},
			connected: 8*24*time.Hour + 2*time.Hour, lastEvent: time.Minute,
		},
	}
	linh := s.Users["linh@acme.io"]
	for _, p := range provs {
		id := ID()
		if err := insertProvider(ctx, s, id, ws.ID, p, linh.ID, checked); err != nil {
			return err
		}
		s.Providers[p.key] = id
	}
	ses, postmark, postal := s.Providers["ses"], s.Providers["postmark"], s.Providers["postal"]
	if err := seedEngineeringBlogProvider(ctx, s, sesCaps, linh.ID, checked); err != nil {
		return err
	}

	// Routing: Broadcasts SES 90 / Postal 10 with two conditions, Automations
	// SES → Postmark, System Postmark → SES.
	beta := s.Lists["Beta Program"]
	rules := []struct {
		stream, name, conditions, splits string
		priority                         int
		fallback                         *uuid.UUID
	}{
		{"broadcast", "Outlook and iCloud are strict with new IPs", `{"mailbox_providers":["outlook","icloud"]}`, `[{"provider_id":"` + ses.String() + `","weight":100}]`, 10, nil},
		{"broadcast", "small, high-value list", `{"lists":["` + beta.String() + `"]}`, `[{"provider_id":"` + ses.String() + `","weight":100}]`, 20, nil},
		{"broadcast", "", `{}`, `[{"provider_id":"` + ses.String() + `","weight":90},{"provider_id":"` + postal.String() + `","weight":10}]`, 100, nil},
		{"automation", "", `{}`, `[{"provider_id":"` + ses.String() + `","weight":100}]`, 100, &postmark},
		{"system", "", `{}`, `[{"provider_id":"` + postmark.String() + `","weight":100}]`, 100, &ses},
	}
	for _, r := range rules {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, ID(), ws.ID, r.stream, r.priority, r.name, json.RawMessage(r.conditions), json.RawMessage(r.splits), r.fallback, linh.ID, s.Ago(48*time.Hour)); err != nil {
			return err
		}
	}
	for stream, policy := range map[string]string{"broadcast": "pause_and_ask", "automation": "automatic", "system": "automatic"} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO stream_policies (workspace_id, stream, failover) VALUES ($1,$2,$3)
			ON CONFLICT (workspace_id, stream) DO UPDATE SET failover = EXCLUDED.failover`, ws.ID, stream, policy); err != nil {
			return err
		}
	}

	// Postal's warm-up: 21 days, day 9 today (the workspace's calendar day,
	// which the warm-up job counts in), one slowdown on day 6.
	plan := postalPlan()
	wsNow := s.Now.In(loc)
	start := time.Date(wsNow.Year(), wsNow.Month(), wsNow.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -8)
	if err := seedWarmup(ctx, s, postal, start, plan, loc, linh.ID); err != nil {
		return err
	}

	if err := seedDailyStats(ctx, s, loc, plan, start); err != nil {
		return err
	}

	// Suppression sync per provider (Postal and SMTP have no suppression API:
	// ReputeCast enforces the shared list before sending).
	pushed := s.Ago(time.Minute)
	for _, row := range []struct {
		id             uuid.UUID
		pulled, pushed int64
	}{{ses, 612, 1262}, {postmark, 188, 1686}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO provider_suppression_sync (provider_id, mode, last_push_at, last_pull_at, pushed, pulled, pending, status)
			VALUES ($1, 'two_way', $2, $2, $3, $4, 0, 'ok')`, row.id, pushed, row.pushed, row.pulled); err != nil {
			return err
		}
	}
	if err := seedSuppressionsIfEmpty(ctx, s); err != nil {
		return err
	}
	if err := seedStreamMessages(ctx, s, loc); err != nil {
		return err
	}
	if err := seedEventLog(ctx, s, linh.ID); err != nil {
		return err
	}
	if err := seedCommunityAdapters(ctx, s); err != nil {
		return err
	}
	// The senders seeder owns sending domains, so only fill in what's
	// missing once every seeder ran.
	s.After(seedProviderDKIM)
	return nil
}

// seedEngineeringBlogProvider gives Acme Engineering Blog its own SES
// account (Instance admin lists it under the workspace), routed for every
// stream.
func seedEngineeringBlogProvider(ctx context.Context, s *Seeder, caps state.Snapshot, by uuid.UUID, checked time.Time) error {
	eng, ok := s.Workspaces["acme-engineering-blog"]
	if !ok {
		return nil
	}
	p := demoProvider{
		key: "eng-ses", adapter: "ses", name: "Amazon SES", label: "us-west-2", status: state.Healthy, detail: "Production · no enforcement", ipMode: "shared",
		config: map[string]any{"demo": true, "region": "us-west-2", "configuration_set": "rc-blog", "access": "role",
			"role_arn": "arn:aws:iam::123456789012:role/reputecast-ses-blog"},
		caps:   caps,
		limits: state.Limits{Bounce: state.Limit{Limit: 5, Review: 5, Pause: 10, Label: "SES"}, Complaint: state.Limit{Limit: 0.1, Review: 0.1, Pause: 0.5, Label: "SES"}},
		health: state.Health{Quota: &state.Quota{Kind: "rolling_24h", Max: 50000, Used: 0}, Rate: &state.Rate{Max: 14},
			Account: "Production · no enforcement", LastSuccessAt: &checked, LastSuccessOp: "GetAccount"},
		pricing:   state.Pricing{Model: "metered", Per1000: 0.10, Source: "AWS pricing"},
		tracking:  state.Tracking{Detail: "Configuration set rc-blog: open and click tracking off.", Scope: "Configuration set rc-blog", Mode: "reputecast", CheckedAt: &checked},
		connected: 120 * 24 * time.Hour, lastEvent: 26 * time.Hour,
	}
	id := ID()
	if err := insertProvider(ctx, s, id, eng.ID, p, by, checked); err != nil {
		return err
	}
	splits := `[{"provider_id":"` + id.String() + `","weight":100}]`
	for _, stream := range []string{"broadcast", "automation", "system"} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, updated_by, updated_at)
			VALUES ($1,$2,$3,100,'','{}',$4,$5,$6)`, ID(), eng.ID, stream, json.RawMessage(splits), by, s.Ago(120*24*time.Hour)); err != nil {
			return err
		}
	}
	return nil
}

func capsOf(adapter string) state.Snapshot {
	d, _ := providers.Lookup(adapter)
	return state.Snapshot{Capabilities: d.Capabilities}
}

func insertProvider(ctx context.Context, s *Seeder, id, wsID uuid.UUID, p demoProvider, by uuid.UUID, checked time.Time) error {
	token, _ := authn.NewToken()
	ips := p.ips
	if ips == nil {
		ips = []string{}
	}
	creds, err := s.Box.Seal(demoCredentials(p.adapter), "provider:"+id.String()+":credentials")
	if err != nil {
		return err
	}
	secretToken, _ := authn.NewToken()
	whsec, err := s.Box.SealString(secretToken, "provider:"+id.String()+":webhook_secret")
	if err != nil {
		return err
	}
	cfg, _ := json.Marshal(p.config)
	caps, _ := json.Marshal(p.caps)
	limits, _ := json.Marshal(p.limits)
	health, _ := json.Marshal(p.health)
	pricing, _ := json.Marshal(p.pricing)
	tracking, _ := json.Marshal(p.tracking)
	_, err = s.Tx.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, webhook_secret_enc, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, last_event_at, last_checked_at, connected_by, connected_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
		id, wsID, p.adapter, p.name, p.label, cfg, creds, token, whsec, p.status, p.detail,
		p.ipMode, ips, caps, limits, health, pricing, tracking, s.Ago(p.lastEvent), checked, by, s.Ago(p.connected))
	return err
}

// postalPlan is the warm-up of the Postal board: caps per day, the rate
// limit rising with the volume, engaged tiers first and the Broadcast share
// stepping from 10% to 40%.
func postalPlan() []warmup.Day {
	caps := []int64{200, 400, 700, 1200, 1800, 2600, 3600, 4800, 6000, 7500, 9000, 11000, 14000, 17000, 20000, 24000, 28000, 33000, 38000, 43000, 48000}
	out := make([]warmup.Day, len(caps))
	for i, c := range caps {
		d := warmup.Day{Day: i + 1, Volume: c}
		switch {
		case d.Day < 12:
			d.Rate = 2
		case d.Day < 16:
			d.Rate = 4
		case d.Day < 19:
			d.Rate = 6
		case d.Day < 21:
			d.Rate = 10
		default:
			d.Rate = 14
		}
		switch {
		case d.Day <= 7:
			d.Tiers = []string{"active"}
		case d.Day <= 11:
			d.Tiers = []string{"active", "cooling"}
		case d.Day < 21:
			d.Tiers = []string{"active", "cooling", "at_risk"}
		default:
			d.Tiers = warmup.Tiers
		}
		switch {
		case d.Day < 12:
			d.Share = 10
		case d.Day < 16:
			d.Share = 15
		case d.Day < 21:
			d.Share = 25
		default:
			d.Share = 40
		}
		out[i] = d
	}
	return out
}

// postalDays are the finished warm-up days: sent, bounce and complaint
// rates (percent) as on the warm-up board.
var postalDays = []struct {
	sent              int64
	bounce, complaint float64
	slowed, held      bool
}{
	{200, 0.5, 0, false, false}, {400, 0.8, 0, false, false}, {700, 0.6, 0, false, false}, {1200, 1.0, 0, false, false}, {1800, 1.2, 0, false, false},
	{2600, 2.3, 0.04, true, false}, {2600, 0.8, 0.04, false, true}, {4798, 0.9, 0.05, false, false},
}

func seedWarmup(ctx context.Context, s *Seeder, provider uuid.UUID, start time.Time, sched []warmup.Day, loc *time.Location, by uuid.UUID) error {
	progress := []warmup.Result{}
	for i, d := range postalDays {
		day := i + 1
		cap := sched[i].Volume
		if d.held {
			cap = postalDays[i-1].sent
		}
		progress = append(progress, warmup.Result{Day: day, Date: start.AddDate(0, 0, i).Format("2006-01-02"), Cap: cap, Sent: d.sent,
			Bounce: d.bounce, Complaint: d.complaint, Slowed: d.slowed, Held: d.held})
	}
	sb, _ := json.Marshal(sched)
	pb, _ := json.Marshal(progress)
	id := ID()
	startDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO warmup_plans (id, provider_id, ip, start_date, days, schedule, engaged_first, slowdown_bounce, slowdown_complaint,
		status, status_detail, progress, guard_action, spread_mailbox, created_by, created_at, updated_at)
		VALUES ($1,$2,'203.0.113.25',$3,21,$4,true,2.0,0.10,'active','',$5,'hold',true,$6,$7,$7)`, id, provider, startDate, sb, pb, by, start.Add(9*time.Hour)); err != nil {
		return err
	}
	dayAt := func(day int, h int) time.Time { return start.AddDate(0, 0, day-1).Add(time.Duration(h) * time.Hour) }
	log := []struct {
		at          time.Time
		kind, title string
		body        string
	}{
		{dayAt(1, 9), "started", "", "Plan started by Linh Nguyen: 21 days, engaged-first."},
		{dayAt(6, 23), "slowdown", "Automatic slowdown.", "Bounces hit 2.3% (guard 2%), mostly Yahoo 421 deferrals that turned into soft bounces. Day 7 cap held at 2,600 and the Yahoo throttle dropped to 1 /s."},
		{dayAt(7, 23), "resumed", "", "Ramp resumed: bounces back to 0.8%."},
		{dayAt(8, 23), "day_complete", "", `Day 8 complete: 4,798 sent inside "October Product Update", bounce 0.9%.`},
	}
	for _, e := range log {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO warmup_events (id, plan_id, occurred_at, kind, title, body) VALUES ($1,$2,$3,$4,$5,$6)`, ID(), id, e.at, e.kind, e.title, e.body); err != nil {
			return err
		}
	}
	_ = loc
	return nil
}

// seedDailyStats writes provider_daily_stats: month-to-date volume and
// cost (Providers KPIs), 7-day bounce, complaint and error rates, and
// Postal's warm-up days.
func seedDailyStats(ctx context.Context, s *Seeder, loc *time.Location, sched []warmup.Day, start time.Time) error {
	now := s.Now.In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	days := int(today.Sub(monthStart).Hours()/24) + 1
	weekStart := today.AddDate(0, 0, -6)
	type spec struct {
		id                uuid.UUID
		month             int64
		bounce, complaint float64 // percent over the last seven days
		errRate           float64 // percent of API calls
		perCall           int64   // messages per API call
		costMicros        int64
	}
	specs := []spec{
		{s.Providers["ses"], 131604, 0.6, 0.04, 0.1, 1, 14620000},
		{s.Providers["postmark"], 3412, 0.2, 0.01, 0, 50, 0},
	}
	for _, sp := range specs {
		weights := make([]float64, days)
		for i := range weights {
			weights[i] = 1
		}
		sent := Split(sp.month, weights)
		cost := Split(sp.costMicros, weights)
		rows := make([]dailyRow, 0, days+7)
		for i := 0; i < days; i++ {
			rows = append(rows, dailyRow{day: monthStart.AddDate(0, 0, i), sent: sent[i], cost: cost[i]})
		}
		// Early in a month the last week reaches into the previous one,
		// which needs volume too for the 7-day rates.
		for d := weekStart; d.Before(monthStart); d = d.AddDate(0, 0, 1) {
			rows = append(rows, dailyRow{day: d, sent: sp.month / int64(max(days, 7))})
		}
		spreadOutcomes(rows, weekStart, sp.bounce, sp.complaint)
		for _, r := range rows {
			if err := insertDaily(ctx, s, sp.id, r, sp.errRate, sp.perCall); err != nil {
				return err
			}
		}
	}
	// Postal: the warm-up days (0.4% API errors), with the last seven days
	// adding up to the boards' 0.9% bounces and 0.05% complaints (the plan
	// keeps each day's own rate for the warm-up table).
	postal := s.Providers["postal"]
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	postalRows := make([]dailyRow, 0, len(postalDays))
	for i, d := range postalDays {
		postalRows = append(postalRows, dailyRow{day: startDay.AddDate(0, 0, i), sent: d.sent})
	}
	spreadOutcomes(postalRows, weekStart, 0.9, 0.05)
	for _, r := range postalRows {
		if err := insertDaily(ctx, s, postal, r, 0.4, 1); err != nil {
			return err
		}
	}
	_ = sched
	return nil
}

// dailyRow is one provider_daily_stats row of the demo.
type dailyRow struct {
	day                       time.Time
	sent, bounced, complained int64
	cost                      int64
}

// spreadOutcomes sets bounces and complaints so the rows from weekStart on
// add up to exactly the given rates (percent); older rows get the same
// rates rounded per day.
func spreadOutcomes(rows []dailyRow, weekStart time.Time, bounce, complaint float64) {
	var weekSent int64
	var idx []int
	var weights []float64
	for i, r := range rows {
		if r.day.Before(weekStart) {
			rows[i].bounced = int64(math.Round(float64(r.sent) * bounce / 100))
			rows[i].complained = int64(math.Round(float64(r.sent-rows[i].bounced) * complaint / 100))
			continue
		}
		idx = append(idx, i)
		weights = append(weights, float64(r.sent))
		weekSent += r.sent
	}
	bounced := Split(int64(math.Round(float64(weekSent)*bounce/100)), weights)
	var weekBounced int64
	for _, b := range bounced {
		weekBounced += b
	}
	complained := Split(int64(math.Round(float64(weekSent-weekBounced)*complaint/100)), weights)
	for j, i := range idx {
		rows[i].bounced, rows[i].complained = bounced[j], complained[j]
	}
}

func insertDaily(ctx context.Context, s *Seeder, provider uuid.UUID, r dailyRow, errRate float64, perCall int64) error {
	delivered := r.sent - r.bounced
	calls := max(r.sent/max(perCall, 1), 0)
	if calls == 0 && r.sent > 0 {
		calls = 1
	}
	errs := int64(math.Round(float64(calls) * errRate / 100))
	_, err := s.Tx.Exec(ctx, `INSERT INTO provider_daily_stats (provider_id, day, sent, delivered, bounced, complained, api_calls, api_errors, cost_micros)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (provider_id, day) DO UPDATE SET sent = EXCLUDED.sent, delivered = EXCLUDED.delivered, bounced = EXCLUDED.bounced,
		complained = EXCLUDED.complained, api_calls = EXCLUDED.api_calls, api_errors = EXCLUDED.api_errors, cost_micros = EXCLUDED.cost_micros`,
		provider, r.day, r.sent, delivered, r.bounced, r.complained, calls, errs, r.cost)
	return err
}

// seedSuppressionsIfEmpty creates the sample shared suppression list
// (1,874 addresses) unless the audience seeder already did.
func seedSuppressionsIfEmpty(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	var n int
	if err := s.Tx.QueryRow(ctx, `SELECT count(*) FROM suppressions WHERE workspace_id = $1`, ws.ID).Scan(&n); err != nil || n > 0 {
		return err
	}
	ses, postmark, postal := s.Providers["ses"], s.Providers["postmark"], s.Providers["postal"]
	brevo := s.designDay(24, 10, 0)
	type group struct {
		reason, source string
		provider       *uuid.UUID
		count          int
		at             time.Time
	}
	groups := []group{
		{"hard_bounce", "import:brevo", nil, 870, brevo},
		{"hard_bounce", "webhook", &ses, 331, s.Ago(9 * 24 * time.Hour)},
		{"hard_bounce", "webhook", &postmark, 120, s.Ago(6 * 24 * time.Hour)},
		// Postal's warm-up bounces ("41 pulled in" on Events & sync).
		{"hard_bounce", "webhook", &postal, 41, s.Ago(2 * time.Minute)},
		{"hard_bounce", "bounce_mailbox", nil, 40, s.Ago(20 * 24 * time.Hour)},
		{"complaint", "import:brevo", nil, 52, brevo},
		{"complaint", "webhook", &ses, 36, s.Ago(4 * 24 * time.Hour)},
		{"complaint", "webhook", &postmark, 8, s.Ago(3 * 24 * time.Hour)},
		{"unsubscribe", "import:brevo", nil, 111, brevo},
		{"unsubscribe", "list_unsubscribe", nil, 220, s.Ago(2 * 24 * time.Hour)},
		{"manual", "manual", nil, 45, s.Ago(30 * 24 * time.Hour)},
	}
	domains := []string{"gmail.com", "outlook.com", "yahoo.com", "icloud.com", "example.org"}
	seq := 0
	for _, g := range groups {
		rows := make([][]any, 0, g.count)
		for i := 0; i < g.count; i++ {
			seq++
			email := fmt.Sprintf("former.reader%04d@%s", seq, domains[seq%len(domains)])
			at := g.at.Add(-time.Duration(i) * 37 * time.Minute)
			if g.source == "import:brevo" {
				at = g.at
			}
			code := map[string]string{"hard_bounce": "550 5.1.1", "complaint": "abuse"}[g.reason]
			rows = append(rows, []any{ID(), ws.ID, email, g.reason, g.source, g.provider, code, at})
		}
		if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"suppressions"}, []string{"id", "workspace_id", "email", "reason", "source", "source_provider_id", "original_code", "created_at"}, pgx.CopyFromRows(rows)); err != nil {
			return err
		}
	}
	return nil
}

// seedStreamMessages adds Automations and System mail for each of the last
// seven whole days and today so far, so the routing screen shows each
// stream's volume ("about 1,900 per day") all day long.
func seedStreamMessages(ctx context.Context, s *Seeder, loc *time.Location) error {
	ws := s.WS()
	ses, postmark := s.Providers["ses"], s.Providers["postmark"]
	kinds := []string{"double_opt_in", "double_opt_in", "double_opt_in", "preference_link", "invite"}
	now := s.Now.In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	rows := make([][]any, 0, 8*2020)
	for d := 0; d <= 7; d++ {
		day, next := today.AddDate(0, 0, -d), today.AddDate(0, 0, 1-d)
		in := func(at time.Time) bool { return at.Before(next) && !at.After(s.Now) }
		for i := 0; i < 1900; i++ {
			at := day.Add(5*time.Minute + time.Duration(i)*45*time.Second)
			if !in(at) {
				break
			}
			rows = append(rows, []any{ID(), ws.ID, "automation", fmt.Sprintf("subscriber%05d@example.com", (d*1900+i)%9973), &ses, "delivered", at, at, at, "", routingMailbox(i)})
		}
		for i := 0; i < 120; i++ {
			at := day.Add(2*time.Hour + time.Duration(i)*11*time.Minute)
			if !in(at) {
				break
			}
			rows = append(rows, []any{ID(), ws.ID, "system", fmt.Sprintf("new.reader%04d@example.com", d*120+i), &postmark, "delivered", at, at, at, kinds[i%len(kinds)], routingMailbox(i)})
		}
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"messages"}, []string{"id", "workspace_id", "stream", "email", "provider_id", "status", "queued_at", "sent_at", "delivered_at", "kind", "mailbox_provider"}, pgx.CopyFromRows(rows))
	return err
}

func routingMailbox(i int) string {
	switch {
	case i%100 < 46:
		return "gmail"
	case i%100 < 67:
		return "outlook"
	case i%100 < 76:
		return "yahoo"
	case i%100 < 84:
		return "icloud"
	}
	return "other"
}

// seedEventLog writes the provider events of the Events & sync board: each
// raw payload beside its normalized event and original code.
func seedEventLog(ctx context.Context, s *Seeder, by uuid.UUID) error {
	ws := s.WS()
	ses, postmark, postal := s.Providers["ses"], s.Providers["postmark"], s.Providers["postal"]
	// A legacy SMTP relay, disconnected, whose bounce mailbox still reported.
	legacy := ID()
	token, _ := authn.NewToken()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, webhook_token, status, status_detail, capabilities, connected_by, connected_at, disconnected_at)
		VALUES ($1,$2,'smtp','Legacy SMTP relay','relay.acme.io','{"demo":true,"host":"relay.acme.io","port":587}',$3,'paused','Disconnected',$4,$5,$6,$7)`,
		legacy, ws.ID, token, mustJSON(capsOf("smtp")), by, s.Ago(400*24*time.Hour), s.Ago(200*24*time.Hour)); err != nil {
		return err
	}
	type ev struct {
		ago                          time.Duration
		provider                     uuid.UUID
		channel                      string
		raw, typ, code, diag         string
		recipient, providerMsg, from string
		class                        string
	}
	events := []ev{
		{19 * time.Second, ses, "webhook", "Bounce · Permanent · General", "hard_bounce", "550 5.1.1", "<r.w···@outlook.com>: Recipient address rejected: User unknown", "r.wong@outlook.com", "0100018f2b4d5c6e-6f7a8b9c-000000", "Weekly Digest #142", "mailbox_unknown"},
		{20 * time.Second, postal, "webhook", "MessageDeliveryFailed · HardFail", "hard_bounce", "550 5.4.1", "Recipient address rejected: Access denied (Outlook)", "k.nguyen@hotmail.com", "12346", "Weekly Digest #142", "policy_block"},
		{23 * time.Second, ses, "webhook", "Delivery", "delivered", "250 2.0.0", "OK · smtpResponse from gmail-smtp-in.l.google.com", "a.kim@gmail.com", "0100018f2b4d5c6e-11111111-000000", "Weekly Digest #142", ""},
		{26 * time.Second, ses, "webhook", "DeliveryDelay · MailboxFull", "deferred", "452 4.2.2", "Mailbox full · retry scheduled " + s.Now.In(state.Location(ws.Timezone)).Add(14*time.Minute).Format("15:04"), "b.hall@yahoo.com", "0100018f2b4d5c6e-33333333-000000", "Weekly Digest #142", "mailbox_full"},
		{32 * time.Second, postmark, "webhook", "SpamComplaint", "complaint", "ARF feedback via Postmark", "· Type 512 · Details \"Spam notification\"", "m.garcia@aol.com", "883953f4-6105-42a2-a16a-77a8eac79400", "Welcome series · step 1", ""},
		{50 * time.Second, postal, "webhook", "MessageDeliveryFailed · SoftFail", "soft_bounce", "421 4.7.0", "[TSS04] Messages from 203.0.113.25 temporarily deferred (Yahoo)", "p.silva@yahoo.com", "12347", "Weekly Digest #142", "reputation"},
		{69 * time.Second, ses, "webhook", "Reject · Virus detected", "rejected", "", "SES refused the message before sending · attachment flagged", "j.chen@acme-partner.co", "0100018f2b4d5c6e-44444444-000000", "API · transactional", "spam_block"},
		{91 * time.Second, postmark, "webhook", "Delivery", "delivered", "250 2.6.0", "Queued mail for delivery (Outlook)", "d.fischer@outlook.com", "883953f4-6105-42a2-a16a-77a8eac79483", "Double opt-in confirmation", ""},
		{120 * time.Second, legacy, "bounce_mailbox", "DSN · 5.1.1", "hard_bounce", "", "Parsed from bounces@news.acme.io via IMAP · SMTP has no webhooks", "t.lee@example.org", "<0199d5a8-legacy@news.acme.io>", "Legacy SMTP relay (disabled)", "mailbox_unknown"},
	}
	for i, e := range events {
		at := s.Ago(e.ago)
		rawID := ID()
		body, _ := json.Marshal(map[string]any{"sample": true, "event": e.raw, "recipient": e.recipient, "provider_message_id": e.providerMsg})
		ct := "application/json"
		if e.channel == "bounce_mailbox" {
			ct = "message/rfc822"
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO provider_raw_events (id, workspace_id, provider_id, received_at, content_type, body, headers, signature_ok, normalized_at, events, source)
			VALUES ($1,$2,$3,$4,$5,$6,'{}',true,$4,1,$7)`, rawID, ws.ID, e.provider, at, ct, body, e.channel); err != nil {
			return err
		}
		mbp := routingMailboxOf(e.recipient)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, provider_id, raw_event_id, type, occurred_at, mailbox_provider, original_type, original_code,
			diagnostic, bounce_class, recipient, provider_message_id, source, received_at, dedupe_key)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			ID(), ws.ID, e.provider, rawID, e.typ, at.Add(-2*time.Second), mbp, e.raw, e.code, e.diag, e.class, e.recipient, e.providerMsg, e.from, at, fmt.Sprintf("demo-%d", i)); err != nil {
			return err
		}
	}
	return nil
}

func routingMailboxOf(email string) string {
	switch {
	case containsAny(email, "@gmail."):
		return "gmail"
	case containsAny(email, "@outlook.", "@hotmail."):
		return "outlook"
	case containsAny(email, "@yahoo.", "@aol."):
		return "yahoo"
	case containsAny(email, "@icloud.", "@me.com"):
		return "icloud"
	}
	return "other"
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		for i := 0; i+len(x) <= len(s); i++ {
			if s[i:i+len(x)] == x {
				return true
			}
		}
	}
	return false
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// seedProviderDKIM gives news.acme.io a DKIM record per provider (the route
// tester's "DKIM selector") when the senders seeder hasn't.
func seedProviderDKIM(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	selectors := map[string]string{"ses": "ses-2026", "postmark": "pm-2026", "postal": "postal-2026"}
	var records json.RawMessage
	// The senders seeder owns the sending domains; use its news.acme.io.
	domainID, known := s.Domains["news.acme.io"]
	var err error
	if known {
		err = s.Tx.QueryRow(ctx, `SELECT dns_records FROM sending_domains WHERE id = $1`, domainID).Scan(&records)
	} else {
		err = s.Tx.QueryRow(ctx, `SELECT id, dns_records FROM sending_domains WHERE workspace_id = $1 AND domain = 'news.acme.io'`, ws.ID).Scan(&domainID, &records)
	}
	if !known && errors.Is(err, pgx.ErrNoRows) {
		domainID = ID()
		records = json.RawMessage(`[]`)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sending_domains (id, workspace_id, domain, dns_records, auth_summary, status, last_checked_at, created_at)
			VALUES ($1,$2,'news.acme.io','[]','{"spf":"pass","dkim":"pass","dmarc":"quarantine","bimi":"missing","return_path":"aligned"}','verified',$3,$4)`,
			domainID, ws.ID, s.Ago(5*time.Minute), s.Ago(220*24*time.Hour)); err != nil {
			return err
		}
		s.Domains["news.acme.io"] = domainID
	} else if err != nil {
		return err
	}
	var recs []map[string]any
	_ = json.Unmarshal(records, &recs)
	has := map[string]bool{}
	for _, r := range recs {
		if r["purpose"] == "dkim" {
			if pid, ok := r["provider_id"].(string); ok {
				has[pid] = true
			}
		}
	}
	for key, sel := range selectors {
		pid := s.Providers[key].String()
		if has[pid] {
			continue
		}
		recs = append(recs, map[string]any{"provider_id": pid, "type": "CNAME", "name": sel + "._domainkey.news.acme.io", "value": sel + ".dkim." + key + ".example",
			"purpose": "dkim", "status": "verified", "checked_at": s.Ago(5 * time.Minute)})
	}
	b, _ := json.Marshal(recs)
	_, err = s.Tx.Exec(ctx, `UPDATE sending_domains SET dns_records = $2 WHERE id = $1`, domainID, b)
	return err
}

// demoCredentials fills every secret field of the adapter with a sample
// value, so the Settings dialog shows them as set. Demo providers never
// call the provider, so the values are never sent anywhere.
func demoCredentials(adapter string) []byte {
	m := map[string]any{"demo": "sample credentials, never sent"}
	if d, ok := providers.Lookup(adapter); ok {
		for _, f := range d.Fields {
			if f.Secret {
				m[f.Key] = "demo-" + f.Key
			}
		}
	}
	b, _ := json.Marshal(m)
	return b
}

// seedCommunityAdapters installs the three community adapters of the
// Adapters board (instance-wide, none connected). The maintainer handles
// are the board's placeholders.
func seedCommunityAdapters(ctx context.Context, s *Seeder) error {
	for _, a := range []struct {
		name, version, latest string
		caps                  []string
		lastRelease           time.Time
		installed             time.Duration
	}{
		{"Scaleway Transactional Email", "0.6.1", "0.7.0", []string{"webhooks", "suppression pull", "batch 50"}, s.Ago(9 * 24 * time.Hour), 160 * 24 * time.Hour},
		{"Mailtrap Sending", "1.1.0", "1.1.0", []string{"webhooks", "domain setup via API"}, s.Ago(52 * 24 * time.Hour), 130 * 24 * time.Hour},
		{"MailerSend", "0.3.4", "0.3.4", []string{"webhooks"}, s.Now.AddDate(0, -14, -3), 100 * 24 * time.Hour},
	} {
		manifest, _ := json.Marshal(map[string]any{"demo": true, "kind": "wasm", "capabilities": a.caps, "last_release": a.lastRelease.UTC().Format(time.RFC3339)})
		if _, err := s.Tx.Exec(ctx, `INSERT INTO installed_adapters (id, name, kind, version, maintainer, source_url, manifest, latest_version, installed_at)
			VALUES ($1, $2, 'wasm', $3, '[MAINTAINER HANDLE]', 'https://github.com/hostrepute/reputecast-adapters', $4, $5, $6)
			ON CONFLICT (name) DO NOTHING`, ID(), a.name, a.version, manifest, a.latest, s.Ago(a.installed)); err != nil {
			return err
		}
	}
	return nil
}
