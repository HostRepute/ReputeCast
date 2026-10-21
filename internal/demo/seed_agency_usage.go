package demo

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// The agency sample's usage (Agency › Usage), rebill rates, the sender
// health its overview draws, and its history: notifications and the audit
// log's Agency entries.
//
// Usage is seeded per day into the agency's own rollup (marked demo, so the
// hourly job leaves those months alone): SMS segments and web push
// notifications too, which the SMS and push channels report once they're
// built.

// nwMonth is a client's usage in a month.
type nwMonth struct {
	emails   [4]int64 // broadcast, automation, transactional, system
	sesCost  bool     // the emails went through SES at $0.10 per 1,000
	segments int64    // SMS segments, through Twilio
	smsCost  int64    // micro-dollars
	us       int64    // segments to the US (the rest to Canada)
	push     int64
}

// streams splits a month's emails like most clients do.
func streams(total int64) [4]int64 {
	p := Split(total, []float64{0.80, 0.15, 0.04, 0.01})
	return [4]int64{p[0], p[1], p[2], p[3]}
}

func (n *nwSeed) usage(ctx context.Context) error {
	cur := time.Date(n.Now.UTC().Year(), n.Now.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	last := cur.AddDate(0, -1, 0)
	todayUTC := time.Date(n.Now.UTC().Year(), n.Now.UTC().Month(), n.Now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	months := map[string]map[time.Time]nwMonth{
		// The month before: the Usage board.
		"contoso":   {last: {emails: streams(846210), sesCost: true}, cur: {emails: streams(412880), sesCost: true}},
		"fabrikam":  {last: {emails: [4]int64{1612400, 498230, 96410, 7840}, sesCost: true, segments: 38412, smsCost: 412100000, us: 36190}, cur: {emails: streams(1180420), sesCost: true, segments: 21640, smsCost: 232180000, us: 20410}},
		"tailspin":  {last: {emails: streams(402118), sesCost: true, push: 12040}, cur: {emails: streams(188050), sesCost: true, push: 6210}},
		"alpine":    {last: {emails: streams(96440), sesCost: true, segments: 4210, smsCost: 46310000, us: 4210}, cur: {emails: streams(51200), sesCost: true, segments: 3980, smsCost: 43780000, us: 3980}},
		"coho":      {last: {emails: streams(41320)}, cur: {emails: streams(18960)}},
		"litware":   {last: {emails: streams(33960), push: 2880}, cur: {emails: streams(15420), push: 1340}},
		"northwind": {last: {emails: streams(6820), sesCost: true}, cur: {emails: streams(3390), sesCost: true}},
	}
	// Fabrikam's earlier months, for "Billed, last months" (its rate went
	// from cost + 10% plus $250 to cost + 15% plus $300 four months ago).
	for i, m := range []struct {
		emails, segments, smsCost int64
	}{{1702300, 22410, 240840000}, {1856000, 24190, 260000000}, {1994870, 28950, 311000000}, {2105651, 32300, 347000000}} {
		months["fabrikam"][cur.AddDate(0, i-5, 0)] = nwMonth{emails: streams(m.emails), sesCost: true, segments: m.segments, smsCost: m.smsCost, us: m.segments * 94 / 100}
	}
	// The other clients' earlier months, a little smaller each month back.
	for key, ms := range months {
		if key == "fabrikam" {
			continue
		}
		base := ms[last]
		for back, scale := range map[int]float64{2: 0.95, 3: 0.9, 4: 0.86} {
			m := cur.AddDate(0, -back, 0)
			if !m.After(monthOf(n.ws[key].CreatedAt)) {
				continue
			}
			var total int64
			for _, e := range base.emails {
				total += e
			}
			ms[m] = nwMonth{emails: streams(int64(math.Round(float64(total) * scale))), sesCost: base.sesCost, segments: int64(math.Round(float64(base.segments) * scale)),
				smsCost: int64(math.Round(float64(base.smsCost) * scale)), us: int64(math.Round(float64(base.us) * scale)), push: int64(math.Round(float64(base.push) * scale))}
		}
	}
	rows := [][]any{}
	computed := n.at(0, 10, 0)
	if computed.After(n.Now) {
		computed = n.Now.Add(-5 * time.Minute)
	}
	for key, ms := range months {
		ws := n.ws[key]
		ses := n.providers[key+"/ses"]
		mainID := n.mainProvider(key)
		for month, m := range ms {
			end := month.AddDate(0, 1, 0)
			if month.Equal(cur) {
				end = todayUTC.AddDate(0, 0, 1)
			}
			days := int(end.Sub(month).Hours() / 24)
			w := make([]float64, days)
			for d := range w {
				w[d] = 1 + 0.35*float64((d*7)%5)/4
				if wd := month.AddDate(0, 0, d).Weekday(); wd == time.Saturday || wd == time.Sunday {
					w[d] *= 0.6
				}
			}
			var emails [4][]int64
			for s := range m.emails {
				emails[s] = Split(m.emails[s], w)
			}
			sms, smsCost, us, push := Split(m.segments, w), Split(m.smsCost, w), Split(m.us, w), Split(m.push, w)
			for d := 0; d < days; d++ {
				day := month.AddDate(0, 0, d)
				var total int64
				for s, key := range []string{"broadcast", "automation", "transactional", "system"} {
					if emails[s][d] > 0 {
						rows = append(rows, []any{ws.ID, day, "email", "stream", key, "", emails[s][d], int64(0), true, computed})
					}
					total += emails[s][d]
				}
				if total > 0 && mainID != uuid.Nil {
					cost := int64(0)
					provider, label := mainID, n.providerName(mainID)
					if m.sesCost {
						provider, label, cost = ses, "Amazon SES", total*100
					}
					rows = append(rows, []any{ws.ID, day, "email", "provider", provider.String(), label, total, cost, true, computed})
				}
				if sms[d] > 0 {
					rows = append(rows, []any{ws.ID, day, "sms", "provider", "twilio", "Twilio", sms[d], smsCost[d], true, computed},
						[]any{ws.ID, day, "sms", "country", "US", "United States", us[d], int64(0), true, computed})
					if ca := sms[d] - us[d]; ca > 0 {
						rows = append(rows, []any{ws.ID, day, "sms", "country", "CA", "Canada", ca, int64(0), true, computed})
					}
				}
				if push[d] > 0 {
					rows = append(rows, []any{ws.ID, day, "push", "provider", "webpush", "Web push", push[d], int64(0), true, computed})
				}
			}
		}
	}
	if _, err := n.Tx.CopyFrom(ctx, pgx.Identifier{"agency_usage"}, []string{"workspace_id", "day", "channel", "dimension", "key", "label", "units", "cost_micros", "demo", "computed_at"},
		pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	if err := n.rates(ctx, cur); err != nil {
		return err
	}
	return n.health(ctx, todayUTC)
}

func monthOf(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (n *nwSeed) providerName(id uuid.UUID) string {
	var name string
	_ = n.Tx.QueryRow(context.Background(), `SELECT name FROM providers WHERE id = $1`, id).Scan(&name)
	return name
}

// rates are the clients' rebill rates; the agency's own workspace and the
// new client aren't billed.
func (n *nwSeed) rates(ctx context.Context, cur time.Time) error {
	tomas, nadia := n.people["tomas"], n.people["nadia"]
	for _, r := range []struct {
		ws   string
		from time.Time
		rate string
		by   uuid.UUID
	}{
		{"contoso", monthOf(n.ws["contoso"].CreatedAt), `{"costPlusPct":20,"monthlyFee":150}`, nadia.ID},
		{"fabrikam", monthOf(n.ws["fabrikam"].CreatedAt), `{"costPlusPct":10,"monthlyFee":250}`, nadia.ID},
		{"fabrikam", cur.AddDate(0, -4, 0), `{"costPlusPct":15,"monthlyFee":300}`, tomas.ID},
		{"tailspin", monthOf(n.ws["tailspin"].CreatedAt), `{"per1000Emails":0.25,"monthlyFee":100}`, nadia.ID},
		{"alpine", monthOf(n.ws["alpine"].CreatedAt), `{"costPlusPct":20,"monthlyFee":100}`, tomas.ID},
		{"coho", monthOf(n.ws["coho"].CreatedAt), `{"flat":120}`, tomas.ID},
		{"litware", monthOf(n.ws["litware"].CreatedAt), `{"costPlusPct":25,"monthlyFee":80}`, tomas.ID},
	} {
		by := r.by
		if err := n.Q.AgencyRateUpsert(ctx, store.AgencyRateUpsertParams{WorkspaceID: n.ws[r.ws].ID, EffectiveFrom: r.from, Rate: json.RawMessage(r.rate), SetBy: &by}); err != nil {
			return err
		}
		if _, err := n.Tx.Exec(ctx, `UPDATE agency_rates SET set_at = $3 WHERE workspace_id = $1 AND effective_from = $2`, n.ws[r.ws].ID, r.from, r.from.Add(-9*24*time.Hour+15*time.Hour)); err != nil {
			return err
		}
	}
	return nil
}

// health is 30 days of each client's sender health: Fabrikam sliding to
// 71 since the Yahoo deferrals, Litware dropping to 62 on the day of the
// Mailgun outage, the others steady. Wide World Importers has no domain
// verified yet.
func (n *nwSeed) health(ctx context.Context, today time.Time) error {
	trend := map[string]func(daysAgo int) int{
		"contoso":  func(d int) int { return 94 - (d % 3 / 2) },
		"fabrikam": func(d int) int { return []int{71, 72, 74, 79, 81, 82, 83}[min(d, 6)] + min(max(d-6, 0)/6, 3) },
		"tailspin": func(d int) int { return 88 + (d%4)/3 - (d%5)/4 },
		"alpine":   func(d int) int { return 90 - (d%6)/5 },
		"coho":     func(d int) int { return 97 - (d%7)/6 },
		"litware": func(d int) int {
			if d <= 3 {
				return 62
			}
			return 85 + (d%3)/2
		},
		"northwind": func(d int) int { return 99 - (d%9)/8 },
	}
	hostRepute := map[string]string{"contoso": "clean", "fabrikam": "clean", "alpine": "clean", "northwind": "clean"}
	rows := [][]any{}
	computed := n.Ago(40 * time.Minute)
	for key, fn := range trend {
		ws := n.ws[key]
		for d := 29; d >= 0; d-- {
			day := today.AddDate(0, 0, -d)
			if day.Before(monthOf(ws.CreatedAt)) {
				continue
			}
			score := fn(d)
			tone := "ok"
			switch {
			case score < 65:
				tone = "bad"
			case score < 80:
				tone = "warn"
			}
			rows = append(rows, []any{ws.ID, day, "ready", int32(score), tone, hostRepute[key], int32(0), true, computed.AddDate(0, 0, -d)})
		}
	}
	rows = append(rows, []any{n.ws["wwi"].ID, today, "pending", int32(0), "neutral", "", int32(0), true, computed})
	_, err := n.Tx.CopyFrom(ctx, pgx.Identifier{"agency_health"}, []string{"workspace_id", "day", "state", "score", "tone", "hostrepute", "listings", "demo", "computed_at"},
		pgx.CopyFromRows(rows))
	return err
}

// history writes the notifications the staff got and the Agency entries of
// the audit log (instance and workspaces).
func (n *nwSeed) history(ctx context.Context) error {
	tomas, mei, ella, nadia, ravi := n.people["tomas"], n.people["mei"], n.people["ella"], n.people["nadia"], n.people["ravi"]
	fab, lit := n.ws["fabrikam"], n.ws["litware"]
	// Staff hear about incidents in their clients: Tomás hasn't read the
	// week's two yet.
	for _, nt := range []struct {
		user                                store.User
		ws                                  store.Workspace
		kind, severity, title, summary, url string
		at                                  time.Time
		read                                bool
	}{
		{tomas, fab, "send_paused", "warn", "Send paused", "“Fall clearance” auto-paused: Yahoo deferrals reached 4.1% (limit 3%)", "/w/" + fab.Slug + "/campaigns", n.at(-2, 7, 42), false},
		{tomas, lit, "provider_down", "bad", "Provider down", "Mailgun is down: automations failed over to Postmark", "/w/" + lit.Slug + "/providers", n.at(-3, 14, 10), false},
		{mei, fab, "send_paused", "warn", "Send paused", "“Fall clearance” auto-paused: Yahoo deferrals reached 4.1% (limit 3%)", "/w/" + fab.Slug + "/campaigns", n.at(-2, 7, 42), true},
		{ella, lit, "provider_down", "bad", "Provider down", "Mailgun is down: automations failed over to Postmark", "/w/" + lit.Slug + "/providers", n.at(-3, 14, 10), true},
	} {
		var read *time.Time
		if nt.read {
			t := nt.at.Add(20 * time.Minute)
			read = &t
		}
		if _, err := n.Tx.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, created_at, read_at, email_status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$7,$8,$9,$10,'sent')`, ID(), nt.user.ID, nt.ws.ID, nt.kind, nt.severity, nt.title, nt.summary, nt.url, nt.at, read); err != nil {
			return err
		}
	}
	type entry struct {
		at                        time.Time
		by                        store.User
		ws                        *store.Workspace
		action, tType, tID, tName string
		after, meta               map[string]any
		result                    string
	}
	wwi, contoso, alpine := n.ws["wwi"], n.ws["contoso"], n.ws["alpine"]
	retail := n.templates[nwRetail]
	entries := []entry{
		{at: n.onBoards(date(2026, 8, 1)), by: nadia, action: "agency.mode_enabled", tType: "agency", tName: "Northwind Studio",
			after: map[string]any{"enabled": true, "name": "Northwind Studio", "clients": 6}},
		{at: n.onBoards(date(2026, 8, 1)).Add(10 * time.Minute), by: nadia, action: "agency.staff_added", tType: "agency_staff", tName: "Tomás Herrera",
			after: map[string]any{"level": "agency_manager"}},
		{at: n.onBoards(date(2026, 8, 1)).Add(12 * time.Minute), by: nadia, action: "agency.staff_added", tType: "agency_staff", tName: "Mei Lin",
			after: map[string]any{"level": "account_manager", "clients": []string{"Contoso Coffee", "Fabrikam Outdoors", "Coho Winery"}}},
		{at: n.onBoards(date(2026, 8, 1)).Add(14 * time.Minute), by: nadia, action: "agency.staff_added", tType: "agency_staff", tName: "Ella Fischer",
			after: map[string]any{"level": "account_manager", "clients": []string{"Tailspin Toys", "Alpine Ski House", "Litware Books"}}},
		{at: n.onBoards(date(2026, 8, 1)).Add(16 * time.Minute), by: nadia, action: "agency.staff_added", tType: "agency_staff", tName: "Ravi Menon",
			after: map[string]any{"level": "account_manager", "title": "designer"}},
		{at: n.onBoards(date(2026, 10, 2)).Add(4 * time.Hour), by: tomas, action: "agency.template_changed", tType: "agency_template", tID: retail.String(), tName: nwRetail,
			after: map[string]any{"sections": 14, "templates": 6}},
		{at: n.at(-3, 16, 2), by: ella, ws: &alpine, action: "agency.library_item_copied", tType: "template", tName: "Holiday hero (snow)",
			after: map[string]any{"item": "Holiday hero (snow)", "version": 3, "kind": "section"}, meta: map[string]any{"permission": "templates:edit"}},
		{at: n.at(-1, 11, 5), by: ravi, ws: &contoso, action: "agency.assignment_changed", tType: "member", tName: "Ravi Menon", result: "denied",
			after: map[string]any{"role": "Admin"}, meta: map[string]any{"reason": "You can assign only roles whose permissions you hold in that workspace."}},
		{at: n.at(-2, 9, 14), by: tomas, action: "agency.contact_searched", tType: "contact", tName: "j.doe@example.com",
			after: map[string]any{"found_in": []string{"Contoso Coffee", "Fabrikam Outdoors"}}, meta: map[string]any{"permission": "subscribers:view"}},
		{at: n.at(-2, 9, 14), by: tomas, ws: &contoso, action: "agency.contact_searched", tType: "contact", tName: "j.doe@example.com", meta: map[string]any{"permission": "subscribers:view"}},
		{at: n.at(-2, 9, 14), by: tomas, ws: &fab, action: "agency.contact_searched", tType: "contact", tName: "j.doe@example.com", meta: map[string]any{"permission": "subscribers:view"}},
		{at: n.at(0, 9, 40), by: tomas, action: "agency.client_created", tType: "workspace", tID: wwi.ID.String(), tName: wwi.Name,
			after: map[string]any{"name": wwi.Name, "slug": wwi.Slug, "timezone": "America/Chicago", "owner": "Nadia Okafor", "template": nwRetail}},
		{at: n.at(0, 9, 40), by: tomas, ws: &wwi, action: "agency.client_created", tType: "workspace", tID: wwi.ID.String(), tName: wwi.Name,
			after: map[string]any{"name": wwi.Name, "template": nwRetail}},
		{at: n.at(0, 9, 40), by: tomas, ws: &wwi, action: "agency.template_applied", tType: "agency_template", tID: retail.String(), tName: nwRetail,
			after: map[string]any{"lists": 3, "automations": 4, "sections": 14, "templates": 6, "client_signoff": "on"}},
		{at: n.at(0, 9, 41), by: tomas, ws: &wwi, action: "agency.assigned", tType: "member", tName: "Mei Lin", after: map[string]any{"role": "Admin"}},
	}
	for _, e := range entries {
		var wsID *uuid.UUID
		if e.ws != nil {
			id := e.ws.ID
			wsID = &id
		}
		result := e.result
		if result == "" {
			result = "allowed"
		}
		a, _ := json.Marshal(e.after)
		if e.after == nil {
			a = []byte(`{}`)
		}
		m, _ := json.Marshal(e.meta)
		if e.meta == nil {
			m = []byte(`{}`)
		}
		if _, err := n.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name, result, ip,
			user_agent, after, metadata) VALUES ($1,$2,$3,'user',$4,$5,$6,$7,$8,$9,$10,'198.51.100.24'::inet,'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Chrome/129.0 Safari/537.36',$11,$12)`,
			ID(), wsID, e.at, e.by.ID, e.by.Name, e.action, e.tType, e.tID, e.tName, result, a, m); err != nil {
			return err
		}
	}
	return nil
}
