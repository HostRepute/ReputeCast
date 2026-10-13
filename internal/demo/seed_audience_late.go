package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// audBoardSuppressedEmails are the addresses on the suppression list board;
// none of them is a contact.
var audBoardSuppressedEmails = []string{
	"r.walker@outlook.com", "k.nakamura@hotmail.com", "m.garcia@aol.com", "j.park@gmail.com", "a.silva@yahoo.com",
	"info@northwind-labs.com", "t.lambert@example.org", "c.brown@outlook.com",
}

// audSupp is one suppression of the sample data. Campaigns, automations
// and providers are named here and looked up when it is written.
type audSupp struct {
	email, reason, source, code, context, scope string
	provider, campaign, automation, step, by    string
	at                                          time.Time
	brevo                                       bool // came in with the Brevo import
}

// audGenerateSuppressions builds the 1,874 suppressions: the eight on the
// board's first page, the 1,033 the Brevo import carried over (180
// unsubscribed, 812 hard bounces, 41 complaints) and 833 older ones.
// Totals: 1,402 hard bounces, 96 complaints, 331 unsubscribed, 45 manual.
func audGenerateSuppressions(s *Seeder, a *audSeed) []audSupp {
	g := &audGen{s: s, rng: audRand(1874, 3), emails: a.emails, now: s.Now}
	out := []audSupp{
		{email: "r.walker@outlook.com", reason: "hard_bounce", provider: "ses", code: "550 5.1.1 user unknown", campaign: "Weekly Digest #142", at: audToday(s, 9, 41).Add(12 * time.Second)},
		{email: "k.nakamura@hotmail.com", reason: "hard_bounce", provider: "postal", code: "550 5.4.1 Access denied", campaign: "Weekly Digest #142", at: audToday(s, 9, 41)},
		{email: "m.garcia@aol.com", reason: "complaint", provider: "postmark", code: "ARF · SpamComplaint", automation: "Welcome series", step: "step 1", at: audToday(s, 9, 40)},
		{email: "j.park@gmail.com", reason: "unsubscribe", source: "preference_center", code: "reason: too many emails", campaign: "October Product Update", at: s.designDay(1, 18, 22)},
		{email: "a.silva@yahoo.com", reason: "unsubscribe", source: "one_click", code: "RFC 8058 POST", campaign: "October Product Update", at: s.designDay(2, 7, 51)},
		{email: "info@northwind-labs.com", reason: "manual", source: "manual", code: "role address, customer asked", by: "linh@acme.io", at: s.designDay(3, 16, 5)},
		{email: "t.lambert@example.org", reason: "hard_bounce", source: "bounce_mailbox", code: "DSN 5.1.1 via IMAP", context: "Legacy SMTP relay", at: s.designDay(3, 11, 37)},
		{email: "c.brown@outlook.com", reason: "complaint", provider: "ses", code: "ComplaintFeedbackType abuse", campaign: "Weekly Digest #141", at: s.designDay(5, 13, 18)},
	}
	brevoAt := s.designDay(24, 16, 40)
	pick := func(m string) string {
		d, _ := g.domain(m)
		return d
	}
	mailbox := func() string {
		r := g.rng.IntN(100)
		switch {
		case r < 46:
			return "gmail"
		case r < 67:
			return "outlook"
		case r < 76:
			return "yahoo"
		case r < 84:
			return "icloud"
		}
		return "other"
	}
	addr := func() string {
		first, last := g.name()
		return g.email(first, last, pick(mailbox()))
	}
	for i := 0; i < 1033; i++ {
		reason := "hard_bounce"
		switch {
		case i < 180:
			reason = "unsubscribe"
		case i >= 992:
			reason = "complaint"
		}
		out = append(out, audSupp{email: addr(), reason: reason, source: "import:brevo", code: "blocklisted in Brevo", context: "Migration", brevo: true,
			at: brevoAt.Add(-time.Duration(g.rng.IntN(200)) * time.Second)})
	}
	older := func() time.Time { return g.daysAgo(25, 210) }
	bounceCodes := []string{"550 5.1.1 user unknown", "550 5.1.1 mailbox does not exist", "550 5.2.1 mailbox disabled", "554 5.7.1 recipient rejected"}
	// Pulled from SES and Postmark before the migration. (Postal only joined
	// nine days ago; its one bounce is k.nakamura's, on the first page.)
	for i := 0; i < 587; i++ {
		p := "ses"
		if i >= 557 {
			p = "postmark"
		}
		out = append(out, audSupp{email: addr(), reason: "hard_bounce", provider: p, source: "provider_sync", code: bounceCodes[g.rng.IntN(len(bounceCodes))], at: older()})
	}
	for i := 0; i < 53; i++ {
		p, code := "ses", "ComplaintFeedbackType abuse"
		if i >= 40 {
			p, code = "postmark", "ARF · SpamComplaint"
		}
		out = append(out, audSupp{email: addr(), reason: "complaint", provider: p, source: "provider_sync", code: code, at: older()})
	}
	for i := 0; i < 149; i++ {
		src, code := "preference_center", "reason: "+[]string{"too many emails", "not relevant", "never signed up", "no reason given"}[g.rng.IntN(4)]
		if i >= 70 {
			src, code = "one_click", "RFC 8058 POST"
		}
		out = append(out, audSupp{email: addr(), reason: "unsubscribe", source: src, code: code, at: older()})
	}
	for i := 0; i < 44; i++ {
		by, src, code := "linh@acme.io", "manual", "asked to be removed"
		switch {
		case i >= 34:
			by, src, code = "", "api", "CRM sync: do-not-contact"
		case i >= 20:
			by, code = "marcus@acme.io", "competitor address"
		}
		out = append(out, audSupp{email: addr(), reason: "manual", source: src, code: code, by: by, at: older()})
	}
	for i := range out {
		if out[i].reason == "unsubscribe" {
			out[i].scope = "marketing"
		} else {
			out[i].scope = "all"
		}
	}
	return out
}

// audSeedLate adds what needs IDs from later seeders: suppressions with
// their providers, campaigns and flows; the provider sync state; the
// saved segments; the events behind the segments; Sam's timeline; consent
// history; and the link Sam clicked.
func audSeedLate(ctx context.Context, s *Seeder, a *audSeed) error {
	welcome, err := audAutomation(ctx, s, "Welcome series")
	if err != nil {
		return err
	}
	steps := []struct {
		name string
		fn   func() error
	}{
		{"suppressions", func() error { return audLinkSuppressions(ctx, s, a, welcome) }},
		{"segments", func() error { return audSegments(ctx, s, a) }},
		{"segment events", func() error { return audSegmentEvents(ctx, s, a) }},
		{"timelines", func() error { return audTimelines(ctx, s, a, welcome) }},
		{"consent history", func() error { return audConsentHistory(ctx, s, a) }},
	}
	for _, st := range steps {
		if err := st.fn(); err != nil {
			return fmt.Errorf("audience %s: %w", st.name, err)
		}
	}
	return nil
}

// audAutomation finds a flow by name, creating a small live one when the
// automations seeder doesn't have it.
func audAutomation(ctx context.Context, s *Seeder, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.Tx.QueryRow(ctx, `SELECT id FROM automations WHERE workspace_id = $1 AND name = $2 AND deleted_at IS NULL ORDER BY created_at LIMIT 1`, s.WS().ID, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return id, err
	}
	id = ID()
	_, err = s.Tx.Exec(ctx, `INSERT INTO automations (id, workspace_id, name, description, status, trigger, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,'Three emails over the first week after joining.','live',$4,$5,$6,$6)`,
		id, s.WS().ID, name, json.RawMessage(`{"kind":"joined_list"}`), s.Users["linh@acme.io"].ID, s.Ago(200*24*time.Hour))
	return id, err
}

func audCampaign(s *Seeder, name string) *uuid.UUID {
	if id, ok := s.Campaigns[name]; ok && id != uuid.Nil {
		return &id
	}
	return nil
}

func audProvider(s *Seeder, key string) *uuid.UUID {
	if id, ok := s.Providers[key]; ok && id != uuid.Nil {
		return &id
	}
	return nil
}

// audWriteSuppressions writes the shared suppression list during the
// audience step, before the providers seeder looks for one. Providers,
// campaigns and flows don't exist yet; audLinkSuppressions fills them in.
// (Each provider's sync state belongs to the providers seeder.)
func audWriteSuppressions(ctx context.Context, s *Seeder, a *audSeed) error {
	ws := s.WS()
	rows := make([][]any, 0, len(a.supps))
	for _, x := range a.supps {
		var importID *uuid.UUID
		if x.brevo {
			importID = &a.brevoImport
		}
		var by *uuid.UUID
		if x.by != "" {
			if u, ok := s.Users[x.by]; ok {
				by = &u.ID
			}
		}
		source := x.source
		if x.provider != "" && source == "" {
			source = "webhook"
		}
		rows = append(rows, []any{ID(), ws.ID, x.email, x.reason, source, x.code, by, x.at, x.scope, x.step, x.context, importID})
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"suppressions"}, []string{"id", "workspace_id", "email", "reason", "source",
		"original_code", "created_by", "created_at", "scope", "automation_step", "context", "import_id"}, pgx.CopyFromRows(rows))
	return err
}

// audLinkSuppressions sets the provider, campaign and flow of each
// suppression once those seeders ran.
func audLinkSuppressions(ctx context.Context, s *Seeder, a *audSeed, welcome uuid.UUID) error {
	var emails []string
	var providers, campaigns, autos []*uuid.UUID
	for _, x := range a.supps {
		if x.provider == "" && x.campaign == "" && x.automation == "" {
			continue
		}
		var auto *uuid.UUID
		if x.automation != "" {
			auto = &welcome
		}
		emails = append(emails, x.email)
		providers = append(providers, audProvider(s, x.provider))
		campaigns = append(campaigns, audCampaign(s, x.campaign))
		autos = append(autos, auto)
	}
	_, err := s.Tx.Exec(ctx, `UPDATE suppressions s SET source_provider_id = x.p, campaign_id = x.c, automation_id = x.a
		FROM unnest($2::text[], $3::uuid[], $4::uuid[], $5::uuid[]) AS x(email, p, c, a)
		WHERE s.workspace_id = $1 AND s.email = x.email`, s.WS().ID, emails, providers, campaigns, autos)
	return err
}

// audSegments saves the seven segments of the Segments tab with the counts
// of their last run.
func audSegments(ctx context.Context, s *Seeder, a *audSeed) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"].ID
	digest, beta := s.Lists["Weekly Digest"].String(), s.Lists["Beta Program"].String()
	ffp := ""
	if id := audCampaign(s, "Feature flags preview"); id != nil {
		ffp = id.String()
	}
	type cond = map[string]any
	type group = map[string]any
	segs := []struct {
		name    string
		groups  []group
		count   int64
		tiers   [4]int64
		ms      int
		created int
		counted time.Time
	}{
		{"Engaged Outlook users on paid plans", []group{
			{"match": "all", "conditions": []cond{{"field": "mailbox_provider", "op": "is", "value": "outlook"}, {"field": "opened", "op": "in_last", "amount": 30, "unit": "days"}, {"field": "list", "op": "includes", "value": digest}}},
			{"match": "any", "conditions": []cond{{"field": "custom:plan", "op": "equals", "value": "team"}, {"field": "custom:plan", "op": "equals", "value": "business"}}},
		}, 2118, [4]int64{1821, 297, 0, 0}, 140, 200, s.Now.Add(-2 * time.Minute)},
		{"Active in last 30 days", []group{{"match": "all", "conditions": []cond{{"field": "tier", "op": "is", "value": "active"}}}}, 29412, [4]int64{29412, 0, 0, 0}, 96, 190, a.tiersComputed},
		{"Unengaged 180 days, for sunset", []group{{"match": "all", "conditions": []cond{{"field": "tier", "op": "is", "value": "sunset"}}}}, 3615, [4]int64{0, 0, 0, 3615}, 88, 150, a.tiersComputed},
		{"Mailchimp import, consent pending", []group{{"match": "all", "conditions": []cond{{"field": "source", "op": "is", "value": "import:mailchimp"}, {"field": "consent", "op": "is", "value": "pending"}}}}, 2118, [4]int64{445, 508, 593, 572}, 61, 23, a.tiersComputed},
		{"Beta Program, opened a beta email", []group{{"match": "all", "conditions": []cond{{"field": "list", "op": "includes", "value": beta}, {"field": "opened_campaign", "op": "opened", "value": ffp}}}}, 1402, [4]int64{1402, 0, 0, 0}, 212, 9, a.tiersComputed},
		{"Gmail only, clickers", []group{{"match": "all", "conditions": []cond{{"field": "mailbox_provider", "op": "is", "value": "gmail"}, {"field": "clicked", "op": "in_last", "amount": 90, "unit": "days"}}}}, 4916, [4]int64{3500, 1416, 0, 0}, 377, 7, a.tiersComputed},
		{"Customers with mrr over $50", []group{{"match": "all", "conditions": []cond{{"field": "custom:mrr", "op": "gt", "value": "50"}}}}, 1087, [4]int64{0, 0, 0, 0}, 74, 5, a.tiersComputed},
	}
	// The mrr segment's tier mix comes from the contacts.
	var mix [4]int64
	for _, c := range a.contacts {
		if m, ok := c.attrs["mrr"].(float64); ok && m > 50 {
			mix[map[string]int{"active": 0, "cooling": 1, "at_risk": 2, "sunset": 3}[c.tier]]++
		}
	}
	segs[6].tiers = mix
	for i, sg := range segs {
		def, _ := json.Marshal(map[string]any{"groups": sg.groups})
		tiers, _ := json.Marshal(map[string]int64{"active": sg.tiers[0], "cooling": sg.tiers[1], "at_risk": sg.tiers[2], "sunset": sg.tiers[3]})
		created := s.designDay(sg.created, 10, 12+i)
		updated := created
		if i == 0 {
			updated = sg.counted // being edited on the Segments board
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO segments (id, workspace_id, name, mode, definition, cached_count, counted_at, count_ms, tier_counts, refresh,
			created_by, updated_by, created_at, updated_at) VALUES ($1,$2,$3,'builder',$4,$5,$6,$7,$8,'dynamic',$9,$9,$10,$11)`,
			ID(), ws.ID, sg.name, def, sg.count, sg.counted, sg.ms, tiers, linh, created, updated); err != nil {
			return err
		}
	}
	return nil
}

// audSegmentEvents writes the opens and clicks the segments count.
func audSegmentEvents(ctx context.Context, s *Seeder, a *audSeed) error {
	ws := s.WS()
	var rows [][]any
	for _, c := range a.contacts {
		if c.ev == nil || c.board != "" {
			continue
		}
		ua := ""
		if c.ev.kind == "open" {
			ua = audUserAgent(c.mailbox)
		}
		rows = append(rows, []any{ID(), ws.ID, audCampaign(s, c.ev.campaign), c.id, c.ev.kind, c.ev.at, c.mailbox, false, ua, audClickURL(c.ev), c.email, c.ev.at})
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"delivery_events"}, []string{"id", "workspace_id", "campaign_id", "contact_id", "type", "occurred_at",
		"mailbox_provider", "machine", "user_agent", "url", "recipient", "received_at"}, pgx.CopyFromRows(rows))
	return err
}

func audClickURL(e *audEvent) string {
	if e.kind != "click" {
		return ""
	}
	if e.campaign == "October Product Update" {
		return "https://docs.acme.io/flags"
	}
	return "https://acme.io/blog"
}

func audUserAgent(mailbox string) string {
	switch mailbox {
	case "gmail":
		return "Mozilla/5.0 (Windows NT 5.1; rv:11.0) Gecko Firefox/11.0 (via ggpht.com GoogleImageProxy)"
	case "outlook":
		return "Microsoft Outlook 16.0.17928; ms-office; MSOffice 16 (Windows NT 10.0)"
	case "yahoo":
		return "YahooMailProxy; https://help.yahoo.com/kb/yahoo-mail-proxy-SLN28749.html"
	case "icloud":
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"
	}
	return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko)"
}

// audMessage is one email to a board contact and what happened to it.
type audMessage struct {
	stream, campaign, node, variant, provider, routeNote string
	failover                                             bool
	queued, delivered                                    time.Time
	deliveredEvent                                       bool
	code, diagnostic                                     string
	open                                                 *time.Time
	machine                                              bool // the open was Apple Mail prefetching it
	ua                                                   string
	click                                                *time.Time
	url                                                  string
	softBounce                                           *time.Time
	bounceCode, bounceDiag                               string
}

// audTimelines writes the emails and events of the board's contacts: Sam's
// 61 events (Brevo history included) and a few for the others.
func audTimelines(ctx context.Context, s *Seeder, a *audSeed, welcome uuid.UUID) error {
	t := func(days, h, m int) time.Time { return s.designDay(days, h, m) }
	ptr := func(x time.Time) *time.Time { return &x }
	outlookUA := "Microsoft Outlook 16.0.17928; ms-office; MSOffice 16 (Windows NT 10.0)"
	sam := []audMessage{
		{stream: "broadcast", campaign: "October Product Update", variant: "B", provider: "postal", queued: t(3, 10, 0), delivered: t(3, 10, 4), deliveredEvent: true,
			code: "250 2.6.0", routeNote: "Postal share of the 90/10 split, warm-up day 6", click: ptr(t(3, 10, 22)), url: "https://docs.acme.io/flags"},
		{stream: "broadcast", campaign: "Weekly Digest #141", provider: "ses", queued: t(5, 9, 0), delivered: t(5, 9, 3), deliveredEvent: true,
			code: "250 2.6.0", diagnostic: "Queued mail for delivery · outlook-com.olc.protection.outlook.com", open: ptr(t(5, 9, 41)), ua: outlookUA},
		{stream: "automation", node: "step 2", provider: "postmark", failover: true, queued: t(10, 9, 0), delivered: t(10, 9, 3), deliveredEvent: true,
			routeNote: "SES was down for 14 min; Automations route failed over automatically"},
		{stream: "broadcast", campaign: "Weekly Digest #140", provider: "ses", queued: t(12, 9, 0), delivered: t(12, 9, 3), deliveredEvent: true,
			code: "250 2.6.0", open: ptr(t(12, 10, 12)), ua: outlookUA},
		{stream: "broadcast", campaign: "Weekly Digest #139", provider: "ses", queued: t(19, 9, 0), delivered: t(19, 9, 58), softBounce: ptr(t(19, 9, 12)),
			bounceCode: "452 4.2.2", bounceDiag: "mailbox full"},
		{stream: "automation", node: "step 1", provider: "ses", queued: t(23, 10, 19), delivered: t(23, 10, 20), deliveredEvent: true,
			code: "250 2.6.0", open: ptr(t(23, 10, 31)), ua: outlookUA},
	}
	// Brevo history: #131–#138 opened (three clicked), #116–#127 delivered,
	// ten opened, seven clicked.
	clicked := map[int]bool{138: true, 135: true, 132: true, 127: true, 125: true, 123: true, 121: true, 119: true, 118: true, 116: true}
	notOpened := map[int]bool{126: true, 120: true}
	for n := 138; n >= 116; n-- {
		if n >= 128 && n <= 130 {
			continue // paused in Brevo for the summer
		}
		days := 26 + 7*(138-n)
		m := audMessage{stream: "broadcast", campaign: fmt.Sprintf("Weekly Digest #%d", n), queued: t(days, 9, 0), delivered: t(days, 9, 2), deliveredEvent: true}
		if !notOpened[n] {
			m.open = ptr(t(days, 9, 30+n%20))
			m.ua = outlookUA
		}
		if clicked[n] {
			m.click = ptr(t(days, 9, 52+n%5))
			m.url = "https://acme.io/blog"
		}
		sam = append(sam, m)
	}
	c := a.byBoard["sam"]
	if err := audWriteMessages(ctx, s, c, sam, welcome); err != nil {
		return err
	}
	// The link Sam clicked.
	if opu := audCampaign(s, "October Product Update"); opu != nil {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_links (id, campaign_id, url, label, position, clicks, unique_clicks)
			SELECT $1, $2, 'https://docs.acme.io/flags', 'Read the docs', 1, 3120, 2604
			WHERE NOT EXISTS (SELECT 1 FROM campaign_links WHERE campaign_id = $2 AND url = 'https://docs.acme.io/flags')`, ID(), *opu); err != nil {
			return err
		}
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO contact_preferences (contact_id, frequency, updated_at) VALUES ($1,'weekly',$2)`, c.id, t(9, 9, 52)); err != nil {
		return err
	}
	// The others: the last email they got and their last open.
	for _, key := range []string{"priya", "diego", "hannah", "kenji", "omar", "grace", "tomasz", "aisha"} {
		b := a.byBoard[key]
		var msgs []audMessage
		if b.stopped == nil {
			opu := audMessage{stream: "broadcast", campaign: "October Product Update", provider: "ses", queued: t(3, 10, 0), delivered: t(3, 10, 2), deliveredEvent: true, code: "250 2.0.0"}
			if b.lastOpen != nil && b.lastOpen.After(opu.delivered) && b.lastOpen.Before(t(1, 0, 0)) {
				opu.open, opu.ua = b.lastOpen, audUserAgent(b.mailbox)
			}
			msgs = append(msgs, opu)
		}
		switch key {
		case "kenji":
			msgs = append(msgs,
				audMessage{stream: "broadcast", campaign: "Weekly Digest #141", provider: "ses", queued: t(5, 9, 0), delivered: t(5, 9, 1), deliveredEvent: true, code: "250 2.0.0", open: b.lastOpen, machine: true, ua: "Mozilla/5.0"},
				audMessage{stream: "broadcast", campaign: "Weekly Digest #139", provider: "ses", queued: t(19, 9, 0), delivered: t(19, 9, 1), deliveredEvent: true, code: "250 2.0.0", open: b.lastHuman, ua: audUserAgent("icloud")})
		case "aisha":
			msgs = append(msgs, audMessage{stream: "broadcast", campaign: "Feature flags preview", provider: "ses", queued: t(10, 15, 0), delivered: t(10, 15, 1), deliveredEvent: true, code: "250 2.0.0", open: b.lastHuman, ua: audUserAgent("other")})
		case "diego":
			msgs = append(msgs, audMessage{stream: "broadcast", campaign: "Weekly Digest #133", queued: t(61, 9, 0), delivered: t(61, 9, 2), deliveredEvent: true, open: b.lastOpen, ua: audUserAgent("yahoo")})
		case "grace":
			msgs = append(msgs, audMessage{stream: "broadcast", campaign: "Weekly Digest #109", queued: t(229, 9, 0), delivered: t(229, 9, 2), deliveredEvent: true, open: b.lastOpen, ua: audUserAgent("outlook")})
		}
		if err := audWriteMessages(ctx, s, b, msgs, welcome); err != nil {
			return err
		}
	}
	return nil
}

// audWriteMessages writes a contact's messages and their delivery and
// engagement events.
func audWriteMessages(ctx context.Context, s *Seeder, c *audContact, msgs []audMessage, welcome uuid.UUID) error {
	ws := s.WS()
	for _, m := range msgs {
		campaign := audCampaign(s, m.campaign)
		if m.campaign != "" && campaign == nil {
			continue
		}
		var auto *uuid.UUID
		node := ""
		if m.stream == "automation" {
			auto, node = &welcome, m.node
		}
		provider := audProvider(s, m.provider)
		id := ID()
		status := "delivered"
		var opened, clickedAt *time.Time
		if m.open != nil {
			opened = m.open
		}
		if m.click != nil {
			clickedAt = m.click
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, campaign_id, automation_id, automation_node, contact_id, email, variant,
			provider_id, mailbox_provider, status, queued_at, sent_at, delivered_at, opened_at, clicked_at, failover, route_note)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13,$14,$15,$16,$17,$18)`,
			id, ws.ID, m.stream, campaign, auto, node, c.id, c.email, m.variant, provider, c.mailbox, status, m.queued, m.delivered, opened, clickedAt,
			m.failover, m.routeNote); err != nil {
			return err
		}
		ev := func(typ string, at time.Time, code, diag, ua, url string) error {
			_, err := s.Tx.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, type, occurred_at,
				mailbox_provider, original_code, diagnostic, user_agent, url, machine, recipient, received_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$8)`,
				ID(), ws.ID, id, campaign, c.id, provider, typ, at, c.mailbox, code, diag, ua, url, typ == "open" && m.machine, c.email)
			return err
		}
		if m.softBounce != nil {
			if err := ev("soft_bounce", *m.softBounce, m.bounceCode, m.bounceDiag, "", ""); err != nil {
				return err
			}
		}
		if m.deliveredEvent {
			if err := ev("delivered", m.delivered, m.code, m.diagnostic, "", ""); err != nil {
				return err
			}
		}
		if m.open != nil {
			if err := ev("open", *m.open, "", "", m.ua, ""); err != nil {
				return err
			}
		}
		if m.click != nil {
			if err := ev("click", *m.click, "", "", "", m.url); err != nil {
				return err
			}
		}
	}
	return nil
}

// audConsentHistory records how every contact joined each list: Sam's
// import and re-permission, the Mailchimp cohort's pending import, the
// Brevo migration, signups since, and this month's departures.
func audConsentHistory(ctx context.Context, s *Seeder, a *audSeed) error {
	linh := s.Users["linh@acme.io"].ID
	lists := map[int]uuid.UUID{audInDigest: s.Lists["Weekly Digest"], audInUpdates: s.Lists["Product Updates"], audInBeta: s.Lists["Beta Program"]}
	brevoLists := map[int]string{audInDigest: "Newsletter", audInUpdates: "Product news", audInBeta: "Beta testers"}
	brevoAt := s.designDay(24, 16, 40)
	var rows [][]any
	add := func(contact uuid.UUID, list *uuid.UUID, action, source, detail, note, version string, ip *netip.Addr, actor, importID *uuid.UUID, at time.Time) {
		rows = append(rows, []any{ID(), contact, list, action, source, ip, "", detail, note, version, actor, importID, at})
	}
	sam := a.byBoard["sam"]
	digest, updates := lists[audInDigest], lists[audInUpdates]
	ip := netip.MustParseAddr("203.0.113.9")
	add(sam.id, &digest, "imported", "import:brevo", `list "Newsletter" · status subscribed`, `consent carried over as "pending" until re-permission`, "", nil, &linh, &a.brevoImport, brevoAt)
	add(sam.id, &digest, "confirmed", "repermission", "re-permission email", "confirmation sent via Postmark (System stream)", "v3", &ip, nil, nil, s.designDay(23, 10, 14))
	add(sam.id, &updates, "subscribed", "preference_center", "frequency: every release", "", "v3", nil, nil, nil, s.designDay(9, 9, 52))
	for _, c := range a.contacts {
		if c.board == "sam" {
			continue
		}
		for _, bit := range audListBits {
			if c.lists&bit == 0 {
				continue
			}
			list := lists[bit]
			switch {
			case c.pending:
				add(c.id, &list, "imported", "import:mailchimp", `list "Conference 2025 signups" · status pending`, "never confirmed double opt-in in Mailchimp", "", nil, &linh, &a.mailchimpOld, c.created)
			case c.importID != nil && *c.importID == a.brevoImport && c.subscribed[bit].Before(brevoAt):
				if c.source != "import:brevo" {
					// Joined in Brevo through a form or the API, years ago.
					action := "subscribed"
					if c.consent == "double" {
						action = "confirmed"
					}
					add(c.id, &list, action, c.memberSource[bit], "", "", "v2", nil, nil, nil, c.subscribed[bit])
				}
				add(c.id, &list, "imported", "import:brevo", fmt.Sprintf("list %q · status subscribed", brevoLists[bit]), "", "", nil, &linh, &a.brevoImport, brevoAt)
			default:
				action := "subscribed"
				if c.consent == "double" {
					action = "confirmed"
				}
				add(c.id, &list, action, c.memberSource[bit], "", "", "v3", nil, nil, nil, c.subscribed[bit])
			}
		}
	}
	for _, d := range a.departed {
		list := lists[d.list]
		add(d.c.id, &list, "unsubscribed", "preference_center", "", "", "", nil, nil, nil, d.at)
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"consent_events"}, []string{"id", "contact_id", "list_id", "action", "source", "ip", "user_agent", "detail",
		"note", "consent_version", "actor_id", "import_id", "occurred_at"}, pgx.CopyFromRows(rows))
	return err
}
