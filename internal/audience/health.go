package audience

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// HealthTierCard is one of the four tier cards.
type HealthTierCard struct {
	Tier  string  `json:"tier" enum:"active,cooling,at_risk,sunset"`
	Count int64   `json:"count"`
	Pct   float64 `json:"pct" doc:"Share of all contacts, percent"`
	// DeltaPts is the change in share vs about 30 days ago, in points.
	DeltaPts *float64 `json:"deltaPts,omitempty"`
	// Good says whether the change is good news (more Active, less Sunset).
	Good bool `json:"good"`
}

// HealthTrendWeek is one column of the tier trend: shares of the list (0–1).
type HealthTrendWeek struct {
	// Day is the workspace-local calendar day of the snapshot (today for the
	// newest column).
	Day     string  `json:"day" format:"date"`
	Active  float64 `json:"active"`
	Cooling float64 `json:"cooling"`
	AtRisk  float64 `json:"atRisk"`
	Sunset  float64 `json:"sunset"`
}

// ListHealthRow is one row of "Health per list": a list or a followed view.
type ListHealthRow struct {
	Kind     string             `json:"kind" enum:"list,view"`
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Contacts int64              `json:"contacts"`
	Tiers    AudienceTierCounts `json:"tiers"`
	// NetGrowth is joins minus departures over 30 days.
	NetGrowth int64 `json:"netGrowth"`
	// Rates over 30 days, in percent; nil when nothing was sent.
	HumanOpens    *float64 `json:"humanOpens,omitempty"`
	UnsubRate     *float64 `json:"unsubRate,omitempty"`
	ComplaintRate *float64 `json:"complaintRate,omitempty"`
	BounceRate    *float64 `json:"bounceRate,omitempty"`
	// ComplaintHigh flags complaint rates at or above 0.1%.
	ComplaintHigh bool   `json:"complaintHigh"`
	Health        string `json:"health" doc:"Excellent, Good, Needs attention, Re-permission needed"`
	HealthTone    string `json:"healthTone" enum:"ok,warn,bad"`
}

// SunsetThisWeek projects the sunset policy over the next 7 days.
type SunsetThisWeek struct {
	Entering int64 `json:"entering" doc:"Contacts entering Sunset"`
	// EnterDay names the day most of them enter, when most fall on one day.
	EnterDay string `json:"enterDay,omitempty"`
	Stopping int64  `json:"stopping" doc:"Contacts whose Broadcasts stop"`
	StopDay  string `json:"stopDay,omitempty"`
}

// ListHealthInsight is the sentence under the per-list table.
type ListHealthInsight struct {
	Text string `json:"text"`
}

// ListHealth is the List health screen.
type ListHealth struct {
	ComputedAt *time.Time        `json:"computedAt,omitempty"`
	Total      int64             `json:"total"`
	Tiers      []HealthTierCard  `json:"tiers"`
	Trend      []HealthTrendWeek `json:"trend"`
	PolicyWeek int               `json:"policyWeek" doc:"Index of the trend column when the policy started, -1 if none"`
	Policy     SunsetPolicy      `json:"policy"`
	// ReengageFlowID is the automation the policy names, when it exists.
	ReengageFlowID *uuid.UUID `json:"reengageFlowId,omitempty"`
	// WinBackRate is the share of re-engaged contacts who came back, percent.
	WinBackRate *float64           `json:"winBackRate,omitempty"`
	ThisWeek    SunsetThisWeek     `json:"thisWeek"`
	Rows        []ListHealthRow    `json:"rows"`
	Insight     *ListHealthInsight `json:"insight,omitempty"`
}

// Health returns tiers, the 12-week trend, the policy and health per list.
func (s *Service) Health(ctx context.Context, ws *store.Workspace) (ListHealth, error) {
	p := ParsePolicy(ws.SunsetPolicy)
	now := s.now()
	out := ListHealth{Policy: p, PolicyWeek: -1, Tiers: []HealthTierCard{}, Trend: []HealthTrendWeek{}, Rows: []ListHealthRow{}}
	c, err := s.Q.AudienceCounts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Total = c.Contacts
	if st, err := s.Q.GetAudienceState(ctx, ws.ID); err == nil {
		out.ComputedAt = st.TiersComputedAt
		if st.Reengaged > 0 {
			r := math.Round(float64(st.WonBack)/float64(st.Reengaged)*1000) / 10
			out.WinBackRate = &r
		}
	}
	history, err := s.Q.WorkspaceTierHistory(ctx, store.WorkspaceTierHistoryParams{WorkspaceID: ws.ID, Since: now.AddDate(0, 0, -95)})
	if err != nil {
		return out, err
	}
	current := []int64{c.Active, c.Cooling, c.AtRisk, c.Sunset}
	// Snapshots are keyed by the workspace-local calendar day.
	loc := workspaceLocation(ws)
	l := now.In(loc)
	today := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
	monthAgo := snapshotAt(history, today.AddDate(0, 0, -30))
	tiers := []string{"active", "cooling", "at_risk", "sunset"}
	for i, t := range tiers {
		card := HealthTierCard{Tier: t, Count: current[i], Pct: share(current[i], c.Contacts)}
		if monthAgo != nil && monthAgo.Contacts > 0 {
			before := []int64{monthAgo.Active, monthAgo.Cooling, monthAgo.AtRisk, monthAgo.Sunset}[i]
			d := math.Round((card.Pct-share(before, monthAgo.Contacts))*10) / 10
			card.DeltaPts = &d
			card.Good = (t == "active") == (d >= 0)
		}
		out.Tiers = append(out.Tiers, card)
	}
	// Twelve weekly columns, oldest first; the newest is today.
	for w := 11; w >= 0; w-- {
		var tw HealthTrendWeek
		if w == 0 {
			tw = HealthTrendWeek{Day: today.Format(time.DateOnly)}
			if c.Contacts > 0 {
				tw.Active, tw.Cooling, tw.AtRisk, tw.Sunset = frac(c.Active, c.Contacts), frac(c.Cooling, c.Contacts), frac(c.AtRisk, c.Contacts), frac(c.Sunset, c.Contacts)
			}
		} else {
			snap := snapshotAt(history, today.AddDate(0, 0, -7*w))
			if snap == nil || snap.Contacts == 0 {
				continue
			}
			tw = HealthTrendWeek{Day: snap.Day.Format(time.DateOnly), Active: frac(snap.Active, snap.Contacts), Cooling: frac(snap.Cooling, snap.Contacts),
				AtRisk: frac(snap.AtRisk, snap.Contacts), Sunset: frac(snap.Sunset, snap.Contacts)}
		}
		out.Trend = append(out.Trend, tw)
	}
	// The policy column is the first snapshot taken after the day it was
	// switched on (the nightly run on that day predates it).
	if p.Enabled && p.EnabledAt != "" {
		for i, tw := range out.Trend {
			if tw.Day > p.EnabledAt {
				out.PolicyWeek = i
				break
			}
		}
	}
	if p.ReengageAutomation != "" {
		id, err := s.Q.AudienceAutomationByName(ctx, store.AudienceAutomationByNameParams{WorkspaceID: ws.ID, Name: p.ReengageAutomation})
		switch {
		case err == nil:
			out.ReengageFlowID = &id
		case !errors.Is(err, pgx.ErrNoRows):
			return out, err
		}
	}
	if out.ThisWeek, err = s.thisWeek(ctx, ws, p); err != nil {
		return out, err
	}
	if out.Rows, err = s.healthRows(ctx, ws); err != nil {
		return out, err
	}
	out.Insight, err = s.insight(ctx, ws, out.Rows)
	return out, err
}

func share(n, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(n)/float64(total)*1000) / 10
}

func frac(n, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(n) / float64(total)
}

// snapshotAt returns the latest snapshot on or before day (a calendar day
// at UTC midnight, like the date column).
func snapshotAt(rows []store.WorkspaceTierHistoryRow, day time.Time) *store.WorkspaceTierHistoryRow {
	var best *store.WorkspaceTierHistoryRow
	for i := range rows {
		if !rows[i].Day.After(day) {
			best = &rows[i]
		}
	}
	return best
}

// thisWeek counts contacts entering Sunset and Broadcasts stopping in the
// next 7 days, and the weekday most of them fall on.
func (s *Service) thisWeek(ctx context.Context, ws *store.Workspace, p SunsetPolicy) (SunsetThisWeek, error) {
	var out SunsetThisWeek
	if !p.Enabled {
		return out, nil
	}
	now := s.now()
	tz := ws.Timezone
	if _, err := time.LoadLocation(tz); err != nil {
		tz = "UTC"
	}
	q := fmt.Sprintf(`SELECT to_char((%s + interval '%d days') AT TIME ZONE $3, 'FMDay') AS day, count(*)
		FROM contacts c
		WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.tier <> 'sunset'
		  AND %[1]s + interval '%[2]d days' > $2 AND %[1]s + interval '%[2]d days' <= $2::timestamptz + interval '7 days'
		GROUP BY 1`, engagementExpr(p), p.SunsetDays)
	var err error
	if out.Entering, out.EnterDay, err = s.byDay(ctx, q, ws.ID, now, tz); err != nil {
		return out, err
	}
	if p.StopBroadcasts {
		q = fmt.Sprintf(`SELECT to_char((c.sunset_at + interval '%d days') AT TIME ZONE $3, 'FMDay') AS day, count(*)
			FROM contacts c
			WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.tier = 'sunset' AND c.broadcasts_stopped_at IS NULL AND c.sunset_at IS NOT NULL
			  AND c.sunset_at + interval '%[1]d days' > $2 AND c.sunset_at + interval '%[1]d days' <= $2::timestamptz + interval '7 days'
			GROUP BY 1`, p.StopAfterDays)
		if out.Stopping, out.StopDay, err = s.byDay(ctx, q, ws.ID, now, tz); err != nil {
			return out, err
		}
	}
	return out, nil
}

// byDay sums counts per weekday and names the day when it holds at least
// 80% of them.
func (s *Service) byDay(ctx context.Context, q string, ws uuid.UUID, now time.Time, tz string) (int64, string, error) {
	rows, err := s.Pool.Query(ctx, q, ws, now, tz)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	var total, best int64
	var bestDay string
	for rows.Next() {
		var day string
		var n int64
		if err := rows.Scan(&day, &n); err != nil {
			return 0, "", err
		}
		total += n
		if n > best {
			best, bestDay = n, day
		}
	}
	if total == 0 || float64(best) < 0.8*float64(total) {
		bestDay = ""
	}
	return total, bestDay, rows.Err()
}

// healthRows builds a row per list and per followed view.
func (s *Service) healthRows(ctx context.Context, ws *store.Workspace) ([]ListHealthRow, error) {
	since := s.now().AddDate(0, 0, -30)
	lists, err := s.Q.ListListsWithCounts(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	mix, err := s.Q.ListTierMix(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	growth, err := s.Q.ListGrowthSince(ctx, store.ListGrowthSinceParams{WorkspaceID: ws.ID, Since: &since})
	if err != nil {
		return nil, err
	}
	totals, err := s.Q.ScopeTotalsSince(ctx, store.ScopeTotalsSinceParams{WorkspaceID: ws.ID, Since: since})
	if err != nil {
		return nil, err
	}
	mixBy := map[uuid.UUID]store.ListTierMixRow{}
	for _, m := range mix {
		mixBy[m.ListID] = m
	}
	growthBy := map[uuid.UUID]store.ListGrowthSinceRow{}
	for _, g := range growth {
		growthBy[g.ListID] = g
	}
	totalsBy := map[string]store.ScopeTotalsSinceRow{}
	for _, t := range totals {
		totalsBy[t.Scope+":"+t.ScopeID.String()] = t
	}
	out := []ListHealthRow{}
	for _, l := range lists {
		m := mixBy[l.ID]
		g := growthBy[l.ID]
		row := ListHealthRow{Kind: "list", ID: l.ID.String(), Name: l.Name, Contacts: l.Subscribed,
			Tiers: AudienceTierCounts{m.Active, m.Cooling, m.AtRisk, m.Sunset}, NetGrowth: g.Joined - g.Departed}
		applyRates(&row, totalsBy["list:"+l.ID.String()], 0)
		out = append(out, row)
	}
	views, err := s.Q.HealthViews(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, v := range views {
		var f ContactFilter
		_ = json.Unmarshal(v.Filters, &f)
		fs, err := s.filterSQL(ctx, ws.ID, f)
		if err != nil {
			continue
		}
		var row ListHealthRow
		var pending int64
		if err := s.Pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE c.tier = 'active'), count(*) FILTER (WHERE c.tier = 'cooling'),
			count(*) FILTER (WHERE c.tier = 'at_risk'), count(*) FILTER (WHERE c.tier = 'sunset'), count(*) FILTER (WHERE c.consent = 'pending')
			FROM contacts c WHERE `+fs.where, fs.args...).Scan(&row.Contacts, &row.Tiers.Active, &row.Tiers.Cooling, &row.Tiers.AtRisk, &row.Tiers.Sunset, &pending); err != nil {
			return nil, err
		}
		row.Kind, row.ID, row.Name = "view", v.ID.String(), v.Name
		t := totalsBy["view:"+v.ID.String()]
		row.NetGrowth = t.Joined - t.Departed
		applyRates(&row, t, frac(pending, row.Contacts))
		out = append(out, row)
	}
	return out, nil
}

func rate(n, d int64, digits float64) *float64 {
	if d <= 0 {
		return nil
	}
	v := math.Round(float64(n)/float64(d)*100*digits) / digits
	return &v
}

// applyRates fills the 30-day rates and the health verdict.
func applyRates(r *ListHealthRow, t store.ScopeTotalsSinceRow, pendingShare float64) {
	r.HumanOpens = rate(t.OpenedHuman, t.Delivered, 10)
	r.UnsubRate = rate(t.Unsubscribed, t.Delivered, 100)
	r.ComplaintRate = rate(t.Complained, t.Delivered, 100)
	r.BounceRate = rate(t.Bounced, t.Sent, 10)
	r.ComplaintHigh = r.ComplaintRate != nil && *r.ComplaintRate >= 0.1
	val := func(p *float64) float64 {
		if p == nil {
			return 0
		}
		return *p
	}
	switch {
	case pendingShare > 0.5:
		r.Health, r.HealthTone = "Re-permission needed", "warn"
	case r.ComplaintHigh || val(r.BounceRate) >= 2 || val(r.UnsubRate) >= 0.5:
		r.Health, r.HealthTone = "Needs attention", "warn"
	case r.HumanOpens != nil && *r.HumanOpens >= 50 && val(r.ComplaintRate) < 0.02 && val(r.BounceRate) < 1:
		r.Health, r.HealthTone = "Excellent", "ok"
	default:
		r.Health, r.HealthTone = "Good", "ok"
	}
}

// insight explains the worst row, naming the mailbox provider most of its
// contacts use ("The Mailchimp import drives most of the Outlook complaints").
func (s *Service) insight(ctx context.Context, ws *store.Workspace, rows []ListHealthRow) (*ListHealthInsight, error) {
	var worst *ListHealthRow
	for i := range rows {
		r := &rows[i]
		if r.HealthTone == "ok" {
			continue
		}
		if worst == nil || (r.ComplaintRate != nil && (worst.ComplaintRate == nil || *r.ComplaintRate > *worst.ComplaintRate)) {
			worst = r
		}
	}
	if worst == nil {
		return nil, nil
	}
	name, _, _ := strings.Cut(worst.Name, ",")
	mailbox := ""
	if worst.Kind == "view" {
		id, _ := uuid.Parse(worst.ID)
		v, err := s.Q.GetSavedView(ctx, store.GetSavedViewParams{WorkspaceID: ws.ID, ID: id})
		if err == nil {
			var f ContactFilter
			_ = json.Unmarshal(v.Filters, &f)
			if fs, err := s.filterSQL(ctx, ws.ID, f); err == nil {
				_ = s.Pool.QueryRow(ctx, `SELECT c.mailbox_provider FROM contacts c WHERE `+fs.where+` GROUP BY 1 ORDER BY count(*) DESC LIMIT 1`, fs.args...).Scan(&mailbox)
			}
		}
	}
	l := i18n.LangFrom(ctx)
	if worst.Health == "Re-permission needed" {
		if mailbox != "" {
			return &ListHealthInsight{Text: mInsightRepermit.Text(l, i18n.Args{"name": name, "mailbox": mailboxNameIn(l, mailbox)})}, nil
		}
		return &ListHealthInsight{Text: mInsightRepAll.Text(l, i18n.Args{"name": name})}, nil
	}
	return &ListHealthInsight{Text: mInsightAttend.Text(l, i18n.Args{"name": name})}, nil
}

// mailboxNameIn is a mailbox provider's name, with "Other" in the reader's
// language.
func mailboxNameIn(l i18n.Lang, provider string) string {
	if provider == emailaddr.Other {
		return mMailboxOther.Text(l, nil)
	}
	return emailaddr.MailboxName(provider)
}
