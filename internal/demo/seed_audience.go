package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func init() { seedAudience = seedAudienceImpl }

// seedAudienceImpl creates the audience of the sample data: three lists
// with 48,213 contacts whose tiers, mailbox providers, consent and growth
// add up to the numbers on the boards, custom fields, saved views, the
// sunset policy and its 12-week history, the Brevo and Mailchimp imports
// and the Mailchimp draft waiting in the importer. Suppressions, segments
// and events need campaign, automation and provider IDs, so they are added
// after every seeder ran (seed_audience_late.go).
//
// Contacts are generated in Go and written with COPY rather than with
// generate_series: every cohort the boards count (Outlook users on paid
// plans who opened in 30 days, the 1,204 contacts entering Sunset on
// Wednesday…) is placed exactly, the result is the same on every run, and
// COPY of 50,000 rows takes well under a second.
func seedAudienceImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	lists := []struct {
		name, desc       string
		requiresApproval bool
		created          time.Duration
	}{
		{"Weekly Digest", "Every Tuesday: product news, guides and the changelog.", false, 400 * 24 * time.Hour},
		{"Product Updates", "Launches and deprecations, about twice a month.", false, 380 * 24 * time.Hour},
		{"Beta Program", "Early access to features before they ship.", false, 120 * 24 * time.Hour},
	}
	for _, l := range lists {
		id := ID()
		if _, err := s.Tx.Exec(ctx, `INSERT INTO lists (id, workspace_id, name, description, double_opt_in, requires_approval, created_at) VALUES ($1,$2,$3,$4,true,$5,$6)`,
			id, ws.ID, l.name, l.desc, l.requiresApproval, s.Ago(l.created)); err != nil {
			return err
		}
		s.Lists[l.name] = id
	}
	eng := s.Workspaces["acme-engineering-blog"]
	id := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO lists (id, workspace_id, name, description, created_at) VALUES ($1,$2,'Engineering Blog','New posts from the Acme engineering team.',$3)`, id, eng.ID, s.Ago(300*24*time.Hour)); err != nil {
		return err
	}
	s.Lists["Engineering Blog"] = id
	bits := map[int]uuid.UUID{audInDigest: s.Lists["Weekly Digest"], audInUpdates: s.Lists["Product Updates"], audInBeta: s.Lists["Beta Program"]}

	a := &audSeed{byBoard: map[string]*audContact{}, emails: map[string]bool{}, tiersComputed: audTierRun(s),
		brevoImport: ID(), mailchimpOld: ID(), mailchimpDraft: ID()}
	// Addresses on the suppression list board are never contacts.
	for _, e := range audBoardSuppressedEmails {
		a.emails[e] = true
	}
	steps := []struct {
		name string
		fn   func() error
	}{
		{"custom fields", func() error { return audCustomFields(ctx, s) }},
		{"sunset policy", func() error { return audPolicy(ctx, s, a) }},
		{"imports", func() error { return audSeedImports(ctx, s, a) }},
		{"contacts", func() error {
			audGenerateContacts(s, a)
			a.supps = audGenerateSuppressions(s, a)
			return audWriteContacts(ctx, s, a, bits)
		}},
		{"import reports", func() error { return audImportReports(ctx, s, a) }},
		{"engineering blog", func() error { return audSeedEngineeringBlog(ctx, s) }},
		{"saved views", func() error { return audSavedViews(ctx, s) }},
		{"rollups", func() error { return audRollups(ctx, s, a, bits) }},
		{"import draft", func() error { return audStageDraft(ctx, s, a) }},
	}
	for _, st := range steps {
		if err := st.fn(); err != nil {
			return fmt.Errorf("%s: %w", st.name, err)
		}
	}
	s.After(func(ctx context.Context, s *Seeder) error { return audSeedLate(ctx, s, a) })
	return nil
}

// audToday is "today hh:mm" on the boards, drawn at 11:00. When the demo
// is seeded earlier in the day every such time moves back by the same
// amount, so nothing is in the future and the order is kept.
func audToday(s *Seeder, hour, minute int) time.Time {
	t := s.designDay(0, hour, minute)
	if lead := s.designDay(0, 11, 0).Sub(s.Now); lead > 0 {
		t = t.Add(-lead)
	}
	return t
}

// audTierRun is the last nightly tier run: 03:00 today in the workspace
// timezone, or yesterday when the demo is seeded before 03:00.
func audTierRun(s *Seeder) time.Time {
	t := s.designDay(0, 3, 0)
	if t.After(s.Now) {
		t = t.AddDate(0, 0, -1)
	}
	return t
}

func audCustomFields(ctx context.Context, s *Seeder) error {
	fields := []struct {
		key, label, typ, display string
		options                  []string
	}{
		{"plan", "Plan", "select", "", []string{"free", "starter", "team", "business"}},
		{"mrr", "MRR", "number", "currency", nil},
		{"signup_source", "Signup source", "text", "", nil},
	}
	for i, f := range fields {
		opts, _ := json.Marshal(f.options)
		if f.options == nil {
			opts = []byte(`[]`)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO custom_fields (id, workspace_id, key, label, type, options, display, position, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			ID(), s.WS().ID, f.key, f.label, f.typ, opts, f.display, i, s.Ago(time.Duration(300-i)*24*time.Hour)); err != nil {
			return err
		}
	}
	return nil
}

// audPolicy switches the sunset policy on (since 1 Sep) and records the
// tier run and the re-engagement results so far (828 of 4,500 came back).
func audPolicy(ctx context.Context, s *Seeder, a *audSeed) error {
	enabled := s.designDay(41, 0, 0).Format("2006-01-02")
	policy := fmt.Sprintf(`{"enabled":true,"enabled_at":%q,"active_days":30,"at_risk_days":91,"sunset_days":181,"reengage":true,"reengage_automation":"Re-engagement","stop_broadcasts":true,"stop_after_days":30,"delete_after_months":12}`, enabled)
	if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET sunset_policy = $2 WHERE id = $1`, s.WS().ID, json.RawMessage(policy)); err != nil {
		return err
	}
	for slug, counts := range map[string][2]int64{"acme-newsletter": {4500, 828}, "acme-engineering-blog": {0, 0}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO audience_state (workspace_id, tiers_computed_at, reengaged, won_back, updated_at) VALUES ($1,$2,$3,$4,$2)`,
			s.Workspaces[slug].ID, a.tiersComputed, counts[0], counts[1]); err != nil {
			return err
		}
	}
	return nil
}

// audSavedViews are the shared views on the Contacts board; the Mailchimp
// cohort is followed in List health.
func audSavedViews(ctx context.Context, s *Seeder) error {
	views := []struct {
		name    string
		filter  map[string]string
		health  bool
		created int
	}{
		{"Active", map[string]string{"tier": "active"}, false, 200},
		{"Cooling", map[string]string{"tier": "cooling"}, false, 200},
		{"At risk", map[string]string{"tier": "at_risk"}, false, 200},
		{"Sunset", map[string]string{"tier": "sunset"}, false, 200},
		{"Beta Program", map[string]string{"list": s.Lists["Beta Program"].String()}, false, 110},
		{"Mailchimp import, consent pending", map[string]string{"source": "import:mailchimp", "consent": "pending"}, true, 23},
	}
	for i, v := range views {
		f, _ := json.Marshal(v.filter)
		id := ID()
		if _, err := s.Tx.Exec(ctx, `INSERT INTO saved_views (id, workspace_id, user_id, name, filters, columns, position, created_at, show_in_health)
			VALUES ($1,$2,NULL,$3,$4,'{}',$5,$6,$7)`, id, s.WS().ID, v.name, f, i+1, s.designDay(v.created, 10, 0), v.health); err != nil {
			return err
		}
		if v.health {
			s.Extra["audience.view.mailchimp"] = id
		}
	}
	return nil
}

// audRollups writes the daily rows the nightly tier job keeps: the tier
// mix of the last 95 days (shares drifting from 55.0 / 20.2 / 13.8 / 11.0
// twelve weeks ago to today's, with the month-ago mix the cards compare
// against) and 30 days of Broadcast results per list and for the followed
// Mailchimp view.
func audRollups(ctx context.Context, s *Seeder, a *audSeed, bits map[int]uuid.UUID) error {
	ws := s.WS()
	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}
	day := func(t time.Time) time.Time {
		l := t.In(loc)
		return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
	}
	var now [4]int64
	for _, c := range a.contacts {
		switch c.tier {
		case "active":
			now[0]++
		case "cooling":
			now[1]++
		case "at_risk":
			now[2]++
		default:
			now[3]++
		}
	}
	total := float64(len(a.contacts))
	today := [4]float64{float64(now[0]) / total, float64(now[1]) / total, float64(now[2]) / total, float64(now[3]) / total}
	start := [4]float64{0.550, 0.202, 0.138, 0.110}
	month := [4]float64{0.589, 0.201, 0.122, 0.088}
	monthCounts := [4]int64{28095, 9588, 5819, 4198} // of 47,700
	var rows [][]any
	for d := 95; d >= 1; d-- {
		var n int64
		var shares [4]float64
		switch {
		case d >= 28 && d <= 32:
			n = 47700
		case d > 30:
			n = 47700 - int64(d-30)*20
		default:
			n = 47700 + int64(math.Round(float64(30-d)*(total-47700)/30))
		}
		for i := range shares {
			switch {
			case d >= 84:
				shares[i] = start[i]
			case d > 30:
				shares[i] = start[i] + (month[i]-start[i])*float64(84-d)/54
			default:
				shares[i] = month[i] + (today[i]-month[i])*float64(30-d)/30
			}
		}
		counts := [4]int64{}
		if d >= 28 && d <= 32 {
			counts = monthCounts
		} else {
			counts[0] = int64(math.Round(float64(n) * shares[0]))
			counts[1] = int64(math.Round(float64(n) * shares[1]))
			counts[2] = int64(math.Round(float64(n) * shares[2]))
			counts[3] = n - counts[0] - counts[1] - counts[2]
		}
		rows = append(rows, []any{ws.ID, "workspace", ws.ID, day(s.Now.AddDate(0, 0, -d)), n, counts[0], counts[1], counts[2], counts[3]})
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"audience_daily"}, []string{"workspace_id", "scope", "scope_id", "day", "contacts", "active", "cooling", "at_risk", "sunset"},
		pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	// Broadcast results over 30 days: Weekly Digest 44.1% human opens,
	// 0.24% unsubscribes, 0.04% complaints, 0.6% bounces; Product Updates
	// 40.2 / 0.31 / 0.05 / 0.7; Beta Program 58.2 / 0.09 / 0.00 / 0.4; the
	// Mailchimp cohort 18.6 / 0.71 / 0.18 / 1.9.
	type result struct {
		scope                                          string
		id                                             uuid.UUID
		sends                                          []string
		sent, bounced, opened, unsubscribed, complaint int64
	}
	results := []result{
		{"list", bits[audInDigest], []string{"Weekly Digest #139", "Weekly Digest #140", "Weekly Digest #141", "October Product Update"}, 126000, 756, 55233, 301, 50},
		{"list", bits[audInUpdates], []string{"October Product Update"}, 28000, 196, 11177, 86, 14},
		{"list", bits[audInBeta], []string{"Feature flags preview", "October Product Update"}, 4418, 18, 2561, 4, 0},
	}
	if v, ok := s.Extra["audience.view.mailchimp"].(uuid.UUID); ok {
		results = append(results, result{"view", v, []string{"Weekly Digest #139", "Weekly Digest #140", "Weekly Digest #141"}, 6354, 121, 1159, 44, 11})
	}
	rows = rows[:0]
	for _, r := range results {
		w := make([]float64, len(r.sends))
		for i := range w {
			w[i] = 1
		}
		sent, bounced, opened, unsub, compl := Split(r.sent, w), Split(r.bounced, w), Split(r.opened, w), Split(r.unsubscribed, w), Split(r.complaint, w)
		for i, name := range r.sends {
			rows = append(rows, []any{ws.ID, r.scope, r.id, day(a.send(name).at), sent[i], sent[i] - bounced[i], opened[i], unsub[i], compl[i], bounced[i]})
		}
	}
	_, err = s.Tx.CopyFrom(ctx, pgx.Identifier{"audience_daily"}, []string{"workspace_id", "scope", "scope_id", "day", "sent", "delivered", "opened_human", "unsubscribed", "complained", "bounced"},
		pgx.CopyFromRows(rows))
	return err
}
