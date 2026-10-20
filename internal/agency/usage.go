package agency

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Usage is counted per UTC day and summed per calendar month (UTC). The
// hourly job rolls up yesterday and today; earlier days keep their totals.

// rollupDay recomputes one workspace's usage for one day from the message
// tables, the providers' daily statistics and the other channels.
func (s *Service) rollupDay(ctx context.Context, wsID uuid.UUID, day time.Time) error {
	from := dateOnly(day)
	month := monthStart(from)
	if demo, err := s.Q.AgencyUsageMonthHasDemo(ctx, store.AgencyUsageMonthHasDemoParams{WorkspaceID: wsID, FromDay: month, ToDay: month.AddDate(0, 1, 0)}); err != nil {
		return err
	} else if demo {
		// A month the demo seeded is the demo's.
		return nil
	}
	provs, err := s.Q.AgencyUsageProviders(ctx, []uuid.UUID{wsID})
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(provs))
	for _, p := range provs {
		ids = append(ids, p.ID)
	}
	rows := []UsageRow{}
	if len(ids) > 0 {
		next := from.AddDate(0, 0, 1)
		streams, err := s.Q.AgencyUsageEmailStreams(ctx, store.AgencyUsageEmailStreamsParams{ProviderIds: ids, FromAt: &from, ToAt: &next})
		if err != nil {
			return err
		}
		for _, st := range streams {
			rows = append(rows, UsageRow{Dimension: "stream", Key: st.Stream, Units: st.N})
		}
		days, err := s.Q.AgencyUsageProviderDays(ctx, store.AgencyUsageProviderDaysParams{WorkspaceID: wsID, Day: from})
		if err != nil {
			return err
		}
		for _, d := range days {
			rows = append(rows, UsageRow{Dimension: "provider", Key: d.ID.String(), Label: d.Name, Units: d.Sent, CostMicros: d.CostMicros})
		}
	}
	type channelRows struct {
		channel string
		rows    []UsageRow
	}
	others := []channelRows{}
	for _, c := range s.deps().Channels {
		rs, err := c.Day(ctx, s.Pool, wsID, from)
		if err != nil {
			return err
		}
		others = append(others, channelRows{c.Channel(), rs})
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.AgencyUsageClearDay(ctx, store.AgencyUsageClearDayParams{WorkspaceID: wsID, Day: from}); err != nil {
			return err
		}
		put := func(channel string, r UsageRow) error {
			if r.Units == 0 && r.CostMicros == 0 {
				return nil
			}
			return q.AgencyUsagePut(ctx, store.AgencyUsagePutParams{WorkspaceID: wsID, Day: from, Channel: channel, Dimension: r.Dimension, Key: r.Key,
				Label: r.Label, Units: r.Units, CostMicros: r.CostMicros, ComputedAt: now})
		}
		for _, r := range rows {
			if err := put("email", r); err != nil {
				return err
			}
		}
		for _, o := range others {
			for _, r := range o.rows {
				if err := put(o.channel, r); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func monthStart(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// AgencyUsageView is Agency › Usage for one month.
type AgencyUsageView struct {
	// Month is "2026-09"; Partial marks the month in progress.
	Month   string `json:"month"`
	Partial bool   `json:"partial"`
	// From and To are the days covered (the 1st to today for the month
	// in progress).
	From      time.Time            `json:"from"`
	To        time.Time            `json:"to"`
	UpdatedAt *time.Time           `json:"updatedAt,omitempty"`
	Months    []AgencyUsageMonth   `json:"months"`
	Rows      []AgencyUsageRow     `json:"rows"`
	Totals    AgencyUsageTotals    `json:"totals"`
	Entered   []AgencyEnteredPrice `json:"entered"`
	// CanSeeCosts: you hold agency:usage; without it the table shows
	// volumes only.
	CanSeeCosts bool     `json:"canSeeCosts"`
	Reason      string   `json:"reason,omitempty"`
	Helpers     []string `json:"helpers"`
	// PricingDate is the day the providers' prices were read.
	PricingDate time.Time `json:"pricingDate"`
}

// AgencyUsageMonth is a choice of the month picker.
type AgencyUsageMonth struct {
	Month   string `json:"month"`
	Partial bool   `json:"partial"`
}

// AgencyUsageRow is one client's month.
type AgencyUsageRow struct {
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Emails    int64              `json:"emails"`
	Segments  int64              `json:"segments"`
	Push      int64              `json:"push"`
	WhatsApp  int64              `json:"whatsApp"`
	// Costs is each provider's cost; empty without agency:usage.
	Costs []AgencyProviderCost `json:"costs"`
	// Rate is nil when the client isn't billed.
	Rate      *AgencyRate  `json:"rate,omitempty"`
	RateSince *time.Time   `json:"rateSince,omitempty"`
	Rebill    AgencyRebill `json:"rebill"`
	// Streams are the emails per stream; Countries the SMS segments per country.
	Streams   []AgencyUsageCount `json:"streams"`
	Countries []AgencyUsageCount `json:"countries"`
	// Providers says whether the client has a provider at all ("No provider yet").
	Providers int `json:"providers"`
}

// AgencyProviderCost is one provider's cost in a month.
type AgencyProviderCost struct {
	Provider string  `json:"provider"`
	Adapter  string  `json:"adapter"`
	Channel  string  `json:"channel"`
	Units    int64   `json:"units"`
	Cost     float64 `json:"cost"`
	// Basis is metered (a price per unit) or plan (a monthly price).
	Basis string `json:"basis" enum:"metered,plan"`
	// Per1000 is the metered price per 1,000 emails.
	Per1000 *float64 `json:"per1000,omitempty"`
	// Entered: the price was set in the workspace, not read from the provider.
	Entered bool `json:"entered"`
}

// AgencyUsageCount is a count by key (stream or country).
type AgencyUsageCount struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Units int64  `json:"units"`
}

// AgencyUsageTotals sums the table.
type AgencyUsageTotals struct {
	Workspaces int     `json:"workspaces"`
	Clients    int     `json:"clients"`
	Billed     int     `json:"billed"`
	Emails     int64   `json:"emails"`
	Segments   int64   `json:"segments"`
	Push       int64   `json:"push"`
	WhatsApp   int64   `json:"whatsApp"`
	Cost       float64 `json:"cost"`
	ToRebill   float64 `json:"toRebill"`
}

// AgencyEnteredPrice is a provider whose cost is a price set in the
// workspace (no per-message pricing from its API).
type AgencyEnteredPrice struct {
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Provider  string             `json:"provider"`
	Monthly   float64            `json:"monthly"`
}

// parseMonth reads "2026-09"; empty is the last full month.
func (s *Service) parseMonth(ctx context.Context, v string) (time.Time, error) {
	if strings.TrimSpace(v) == "" {
		return monthStart(s.now()).AddDate(0, -1, 0), nil
	}
	t, err := time.Parse("2006-01", v)
	if err != nil {
		return time.Time{}, api.Invalid("query.month", mMonthInvalid.Text(i18n.LangFrom(ctx), nil))
	}
	return t.UTC(), nil
}

// Usage returns each client's usage and what to rebill for a month.
func (s *Service) Usage(ctx context.Context, month string) (AgencyUsageView, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyUsageView{}, err
	}
	from, err := s.parseMonth(ctx, month)
	if err != nil {
		return AgencyUsageView{}, err
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return AgencyUsageView{}, err
	}
	out, err := s.usage(ctx, set.visible, from)
	if err != nil {
		return out, err
	}
	out.CanSeeCosts, out.Helpers = me.Can(authz.AgencyUsage), s.managerNames(ctx, 2)
	if !out.CanSeeCosts {
		out.Reason = s.denied(ctx, authz.AgencyUsage).Detail
		out.Entered = []AgencyEnteredPrice{}
		out.Totals.Cost, out.Totals.ToRebill = 0, 0
		for i := range out.Rows {
			out.Rows[i].Costs, out.Rows[i].Rate, out.Rows[i].RateSince, out.Rows[i].Rebill = []AgencyProviderCost{}, nil, nil, AgencyRebill{}
		}
	}
	return out, nil
}

// usage computes the month for the given workspaces.
func (s *Service) usage(ctx context.Context, rows []store.AgencyWorkspacesRow, from time.Time) (AgencyUsageView, error) {
	now := s.now()
	to := from.AddDate(0, 1, 0)
	out := AgencyUsageView{Month: from.Format("2006-01"), From: from, To: to.AddDate(0, 0, -1), Months: []AgencyUsageMonth{}, Rows: []AgencyUsageRow{},
		Entered: []AgencyEnteredPrice{}, Helpers: []string{}}
	current := monthStart(now)
	if !from.Before(current) {
		out.Partial = true
		out.To = dateOnly(now)
	}
	out.PricingDate = out.To
	// The month picker: the last 12 months, newest first.
	for i := 0; i < 12; i++ {
		m := current.AddDate(0, -i, 0)
		out.Months = append(out.Months, AgencyUsageMonth{Month: m.Format("2006-01"), Partial: i == 0})
	}
	ordered := slices.Clone(rows)
	slices.SortStableFunc(ordered, func(a, b store.AgencyWorkspacesRow) int {
		if a.Kind != b.Kind {
			if a.Kind == "agency" {
				return 1
			}
			return -1
		}
		return a.AddedAt.Compare(b.AddedAt)
	})
	ids := make([]uuid.UUID, 0, len(ordered))
	for _, r := range ordered {
		ids = append(ids, r.ID)
	}
	figures, err := s.Q.AgencyUsageRange(ctx, store.AgencyUsageRangeParams{WorkspaceIds: ids, FromDay: from, ToDay: to})
	if err != nil {
		return out, err
	}
	if sec, err := s.Q.AgencyUsageComputedAt(ctx, store.AgencyUsageComputedAtParams{WorkspaceIds: ids, FromDay: from, ToDay: to}); err == nil {
		out.UpdatedAt = unixTime(sec)
	}
	provs, err := s.Q.AgencyUsageProviders(ctx, ids)
	if err != nil {
		return out, err
	}
	rates, err := s.Q.AgencyRates(ctx, ids)
	if err != nil {
		return out, err
	}
	for _, w := range ordered {
		row := s.usageRow(w, figures, provs, rates, from, to)
		for _, c := range row.Costs {
			if c.Basis == "plan" && c.Entered {
				out.Entered = append(out.Entered, AgencyEnteredPrice{Workspace: row.Workspace, Provider: c.Provider, Monthly: c.Cost})
			}
		}
		out.Rows = append(out.Rows, row)
		out.Totals.Workspaces++
		if w.Kind == "client" {
			out.Totals.Clients++
		}
		out.Totals.Emails += row.Emails
		out.Totals.Segments += row.Segments
		out.Totals.Push += row.Push
		out.Totals.WhatsApp += row.WhatsApp
		out.Totals.Cost = cents(out.Totals.Cost + row.Rebill.Cost)
		if row.Rebill.ToRebill != nil {
			out.Totals.ToRebill = cents(out.Totals.ToRebill + *row.Rebill.ToRebill)
			out.Totals.Billed++
		}
	}
	return out, nil
}

// usageRow is one workspace's month: volumes, costs per provider, the
// rate in force and what it makes.
func (s *Service) usageRow(w store.AgencyWorkspacesRow, figures []store.AgencyUsageRangeRow, provs []store.AgencyUsageProvidersRow, rates []store.AgencyRatesRow,
	from, to time.Time) AgencyUsageRow {
	row := AgencyUsageRow{Workspace: wsRef(w), Costs: []AgencyProviderCost{}, Streams: []AgencyUsageCount{}, Countries: []AgencyUsageCount{}}
	var costMicros int64
	known := map[string]store.AgencyUsageProvidersRow{}
	for _, p := range provs {
		if p.WorkspaceID == w.ID {
			known[p.ID.String()] = p
		}
	}
	// Each provider's units and cost, in the order they first appear.
	order := []string{}
	costs := map[string]*AgencyProviderCost{}
	for _, f := range figures {
		if f.WorkspaceID != w.ID {
			continue
		}
		switch {
		case f.Channel == "email" && f.Dimension == "stream":
			row.Emails += f.Units
			row.Streams = append(row.Streams, AgencyUsageCount{Key: f.Key, Label: f.Label, Units: f.Units})
		case f.Dimension == "provider":
			switch f.Channel {
			case "sms":
				row.Segments += f.Units
			case "push":
				row.Push += f.Units
			case "whatsapp":
				row.WhatsApp += f.Units
			}
			key := f.Channel + "|" + f.Key
			c, ok := costs[key]
			if !ok {
				c = &AgencyProviderCost{Provider: f.Label, Channel: f.Channel, Basis: "metered"}
				if p, ok := known[f.Key]; ok {
					c.Provider, c.Adapter = p.Name, p.Adapter
					if pr := state.PricingOf(store.Provider{Pricing: p.Pricing}); pr.Model == "metered" && pr.Per1000 > 0 && f.Channel == "email" {
						v := pr.Per1000
						c.Per1000 = &v
					}
				}
				costs[key] = c
				order = append(order, key)
			}
			c.Units += f.Units
			c.Cost = cents(c.Cost + float64(f.CostMicros)/1e6)
			costMicros += f.CostMicros
		case f.Channel == "sms" && f.Dimension == "country":
			row.Countries = append(row.Countries, AgencyUsageCount{Key: f.Key, Label: f.Label, Units: f.Units})
		}
	}
	// Plans count in full for every month the provider was connected.
	for _, p := range provs {
		if p.WorkspaceID != w.ID {
			continue
		}
		row.Providers++
		pr := state.PricingOf(store.Provider{Pricing: p.Pricing})
		if (pr.Model != "plan" && pr.Model != "self_hosted") || pr.Monthly <= 0 {
			continue
		}
		if !p.ConnectedAt.Before(to) || (p.DisconnectedAt != nil && p.DisconnectedAt.Before(from)) {
			continue
		}
		key := "email|" + p.ID.String()
		c, ok := costs[key]
		if !ok {
			c = &AgencyProviderCost{Provider: p.Name, Adapter: p.Adapter, Channel: "email"}
			costs[key] = c
			order = append(order, key)
		} else {
			costMicros -= int64(c.Cost * 1e6)
		}
		c.Basis, c.Cost, c.Per1000 = "plan", cents(pr.Monthly), nil
		c.Entered = pr.Source == "workspace" || pr.Source == "manual"
		costMicros += int64(pr.Monthly * 1e6)
	}
	for _, k := range order {
		row.Costs = append(row.Costs, *costs[k])
	}
	slices.SortStableFunc(row.Streams, func(a, b AgencyUsageCount) int { return streamRank(a.Key) - streamRank(b.Key) })
	slices.SortStableFunc(row.Countries, func(a, b AgencyUsageCount) int { return int(b.Units - a.Units) })
	if r, since := rateFor(rates, w.ID, from); r != nil {
		row.Rate, row.RateSince = r, since
		row.Rebill = rebill(*r, costMicros, row.Emails, row.Segments)
	} else {
		row.Rebill = rebill(AgencyRate{}, costMicros, row.Emails, row.Segments)
	}
	return row
}

func streamRank(k string) int {
	switch k {
	case "broadcast":
		return 0
	case "automation":
		return 1
	case "transactional":
		return 2
	case "system":
		return 3
	}
	return 4
}

// rateFor is the rate in force in a month: the latest that took effect on
// or before its first day. Nil when the client isn't billed.
func rateFor(rates []store.AgencyRatesRow, wsID uuid.UUID, month time.Time) (*AgencyRate, *time.Time) {
	var found *store.AgencyRatesRow
	for i := range rates {
		r := rates[i]
		if r.WorkspaceID != wsID || r.EffectiveFrom.After(month) {
			continue
		}
		if found == nil || r.EffectiveFrom.After(found.EffectiveFrom) {
			found = &rates[i]
		}
	}
	if found == nil {
		return nil, nil
	}
	rate := parseRate(found.Rate)
	if !rate.Billed() {
		return nil, nil
	}
	since := found.EffectiveFrom
	// "Since" is when this rate started: the earliest of the identical
	// rates in a row before it.
	for i := len(rates) - 1; i >= 0; i-- {
		r := rates[i]
		if r.WorkspaceID != wsID || !r.EffectiveFrom.Before(since) {
			continue
		}
		if string(normalizeRate(r.Rate)) == string(normalizeRate(found.Rate)) {
			since = r.EffectiveFrom
		} else {
			break
		}
	}
	return &rate, &since
}

func normalizeRate(raw json.RawMessage) []byte {
	b, _ := json.Marshal(parseRate(raw))
	return b
}

// AgencyRateInput sets a client's rate.
type AgencyRateInput struct {
	Rate AgencyRate `json:"rate"`
}

// AgencyRateSaved says from when the rate applies.
type AgencyRateSaved struct {
	EffectiveFrom time.Time `json:"effectiveFrom"`
}

// SetRate saves a client's rebill rate. It applies from the first day of
// the current month; earlier months keep the rate they had.
func (s *Service) SetRate(ctx context.Context, wsID string, in AgencyRateInput) (AgencyRateSaved, error) {
	me, _, err := s.require(ctx, authz.AgencyUsage)
	if err != nil {
		return AgencyRateSaved{}, err
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return AgencyRateSaved{}, err
	}
	id, err := uuid.Parse(wsID)
	if err != nil {
		return AgencyRateSaved{}, api.NotFound("Client")
	}
	w, ok := set.find(id)
	if !ok {
		return AgencyRateSaved{}, api.NotFound("Client")
	}
	from := monthStart(s.now())
	rates, err := s.Q.AgencyRates(ctx, []uuid.UUID{id})
	if err != nil {
		return AgencyRateSaved{}, err
	}
	before, _ := rateFor(rates, id, from)
	raw, _ := json.Marshal(in.Rate)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.AgencyRateUpsert(ctx, store.AgencyRateUpsertParams{WorkspaceID: id, EffectiveFrom: from, Rate: raw, SetBy: &me.UserID}); err != nil {
			return err
		}
		var b any
		if before != nil {
			b = before
		}
		wsIDp := id
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsIDp, Action: "agency.rate_changed", TargetType: "workspace", TargetID: id.String(), TargetName: w.Name,
			Before: map[string]any{"rate": b}, After: map[string]any{"rate": in.Rate, "effective_from": from.Format("2006-01-02")},
			Metadata: map[string]any{"permission": string(authz.AgencyUsage)}})
	})
	return AgencyRateSaved{EffectiveFrom: from}, err
}

// ExportUsage writes the month as CSV: one row per client and channel.
func (s *Service) ExportUsage(ctx context.Context, month string, w io.Writer) error {
	me, _, err := s.require(ctx, authz.AgencyUsage)
	if err != nil {
		return err
	}
	from, err := s.parseMonth(ctx, month)
	if err != nil {
		return err
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return err
	}
	view, err := s.usage(ctx, set.visible, from)
	if err != nil {
		return err
	}
	if err := s.Audit.Log(ctx, audit.Event{Action: "agency.usage_exported", TargetType: "agency_usage", TargetName: view.Month,
		After: map[string]any{"month": view.Month, "partial": view.Partial, "workspaces": len(view.Rows)}, Metadata: map[string]any{"permission": string(authz.AgencyUsage)}}); err != nil {
		return err
	}
	return writeUsageCSV(w, view)
}

// writeUsageCSV writes one row per client and channel; the rate and what
// to rebill are the client's and repeat on each of its rows.
func writeUsageCSV(w io.Writer, view AgencyUsageView) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"client", "workspace", "month", "partial", "channel", "units", "unit", "provider_cost_usd", "rebill_rate", "to_rebill_usd"}); err != nil {
		return err
	}
	money := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
	for _, r := range view.Rows {
		rate, toRebill := "not billed", ""
		if r.Rate != nil {
			rate = rateText(*r.Rate)
		}
		if r.Rebill.ToRebill != nil {
			toRebill = money(*r.Rebill.ToRebill)
		}
		costBy := map[string]float64{}
		for _, c := range r.Costs {
			costBy[c.Channel] = cents(costBy[c.Channel] + c.Cost)
		}
		for _, ch := range []struct {
			channel, unit string
			units         int64
		}{{"email", "emails", r.Emails}, {"sms", "segments", r.Segments}, {"push", "notifications", r.Push}, {"whatsapp", "messages", r.WhatsApp}} {
			if ch.units == 0 && costBy[ch.channel] == 0 && ch.channel != "email" {
				continue
			}
			if err := cw.Write([]string{r.Workspace.Name, r.Workspace.Slug, view.Month, strconv.FormatBool(view.Partial), ch.channel,
				strconv.FormatInt(ch.units, 10), ch.unit, money(costBy[ch.channel]), rate, toRebill}); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

// rateText writes a rate the way the CSV's readers expect: "cost + 20% + $150/month".
func rateText(r AgencyRate) string {
	if r.Flat != nil {
		return fmt.Sprintf("$%.2f/month flat", *r.Flat)
	}
	parts := []string{}
	if r.CostPlusPct != nil {
		parts = append(parts, "cost + "+strconv.FormatFloat(*r.CostPlusPct, 'f', -1, 64)+"%")
	}
	if r.Per1000Emails != nil {
		parts = append(parts, fmt.Sprintf("$%s per 1000 emails", strconv.FormatFloat(*r.Per1000Emails, 'f', -1, 64)))
	}
	if r.PerSegment != nil {
		parts = append(parts, fmt.Sprintf("$%s per SMS segment", strconv.FormatFloat(*r.PerSegment, 'f', -1, 64)))
	}
	if r.MonthlyFee != nil {
		parts = append(parts, fmt.Sprintf("$%.2f/month", *r.MonthlyFee))
	}
	return strings.Join(parts, " + ")
}

// AgencyClientDetails is what the agency keeps about a client: its
// contact, rate, origin, providers, people and the last months' rebill.
type AgencyClientDetails struct {
	Workspace      AgencyWorkspaceRef `json:"workspace"`
	AccountManager *AgencyPersonRef   `json:"accountManager,omitempty"`
	Rate           *AgencyRate        `json:"rate,omitempty"`
	RateSince      *time.Time         `json:"rateSince,omitempty"`
	ClientSince    time.Time          `json:"clientSince"`
	Template       string             `json:"template,omitempty"`
	Providers      []string           `json:"providers"`
	People         []AgencyClientPerson `json:"people"`
	Billed         []AgencyBilledMonth  `json:"billed"`
	CanSeeCosts    bool                 `json:"canSeeCosts"`
	// Staff are the agency's staff who can be the client's contact.
	Staff []AgencyPersonRef `json:"staff"`
	// CanManage: you can change the contact (agency:manage).
	CanManage bool `json:"canManage"`
}

// AgencyClientPerson is one of the client's own people.
type AgencyClientPerson struct {
	Name    string        `json:"name"`
	Email   string        `json:"email"`
	Role    AgencyRoleRef `json:"role"`
	Invited bool          `json:"invited"`
}

// AgencyBilledMonth is a past month's rebill.
type AgencyBilledMonth struct {
	Month    string      `json:"month"`
	Rate     *AgencyRate `json:"rate,omitempty"`
	ToRebill *float64    `json:"toRebill,omitempty"`
}

// ClientDetails returns a client's details for the agency.
func (s *Service) ClientDetails(ctx context.Context, wsID string) (AgencyClientDetails, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyClientDetails{}, err
	}
	l := i18n.LangFrom(ctx)
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return AgencyClientDetails{}, err
	}
	id, err := uuid.Parse(wsID)
	if err != nil {
		return AgencyClientDetails{}, api.NotFound("Client")
	}
	w, ok := set.find(id)
	if !ok {
		return AgencyClientDetails{}, api.NotFound("Client")
	}
	out := AgencyClientDetails{Workspace: wsRef(w), ClientSince: w.ClientSince, Template: w.TemplateName, Providers: []string{}, People: []AgencyClientPerson{},
		Billed: []AgencyBilledMonth{}, Staff: []AgencyPersonRef{}, CanSeeCosts: me.Can(authz.AgencyUsage), CanManage: me.Can(authz.AgencyManage)}
	if w.AccountManagerID != nil {
		out.AccountManager = &AgencyPersonRef{ID: w.AccountManagerID.String(), Name: displayName(w.AccountManagerName, w.AccountManagerEmail), Email: w.AccountManagerEmail, AvatarColor: w.AccountManagerColor}
	}
	provs, err := s.Q.AgencyUsageProviders(ctx, []uuid.UUID{id})
	if err != nil {
		return out, err
	}
	for _, p := range provs {
		if p.DisconnectedAt == nil {
			out.Providers = append(out.Providers, providerLabel(p))
		}
	}
	people, err := s.people(ctx)
	if err != nil {
		return out, err
	}
	staffIDs := []uuid.UUID{}
	for _, p := range people {
		if p.UserID != nil {
			staffIDs = append(staffIDs, *p.UserID)
			if p.Active {
				out.Staff = append(out.Staff, AgencyPersonRef{ID: p.UserID.String(), Name: p.Name, Email: p.Email, AvatarColor: p.Color})
			}
		}
	}
	cp, err := s.Q.AgencyClientPeople(ctx, store.AgencyClientPeopleParams{WorkspaceID: id, StaffIds: staffIDs})
	if err != nil {
		return out, err
	}
	for _, p := range cp {
		out.People = append(out.People, AgencyClientPerson{Name: displayName(p.Name, p.Email), Email: p.Email, Invited: p.Invited,
			Role: AgencyRoleRef{Key: p.RoleKey, Name: api.RoleName(l, p.RoleKey, p.RoleName), Builtin: p.RoleBuiltin}})
	}
	if !out.CanSeeCosts {
		return out, nil
	}
	rates, err := s.Q.AgencyRates(ctx, []uuid.UUID{id})
	if err != nil {
		return out, err
	}
	out.Rate, out.RateSince = rateFor(rates, id, monthStart(s.now()))
	// The last five full months, newest first.
	current := monthStart(s.now())
	figures, err := s.Q.AgencyUsageRange(ctx, store.AgencyUsageRangeParams{WorkspaceIds: []uuid.UUID{id}, FromDay: current.AddDate(0, -5, 0), ToDay: current})
	if err != nil {
		return out, err
	}
	_ = figures
	for i := 1; i <= 5; i++ {
		m := current.AddDate(0, -i, 0)
		if m.Before(monthStart(w.ClientSince)) {
			break
		}
		monthFigures, err := s.Q.AgencyUsageRange(ctx, store.AgencyUsageRangeParams{WorkspaceIds: []uuid.UUID{id}, FromDay: m, ToDay: m.AddDate(0, 1, 0)})
		if err != nil {
			return out, err
		}
		row := s.usageRow(w, monthFigures, provs, rates, m, m.AddDate(0, 1, 0))
		out.Billed = append(out.Billed, AgencyBilledMonth{Month: m.Format("2006-01"), Rate: row.Rate, ToRebill: row.Rebill.ToRebill})
	}
	return out, nil
}

// providerLabel names a provider with its region where it has one
// ("Amazon SES us-east-1").
func providerLabel(p store.AgencyUsageProvidersRow) string {
	if p.Label != "" && !strings.EqualFold(p.Label, p.Name) {
		return p.Name + " " + p.Label
	}
	return p.Name
}

// AgencyClientInput changes a client's details.
type AgencyClientInput struct {
	AccountManagerID string `json:"accountManagerId,omitempty" format:"uuid"`
}

// UpdateClient changes a client's contact at the agency.
func (s *Service) UpdateClient(ctx context.Context, wsID string, in AgencyClientInput) error {
	me, _, err := s.require(ctx, authz.AgencyManage)
	if err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	id, err := uuid.Parse(wsID)
	if err != nil {
		return api.NotFound("Client")
	}
	w, err := s.Q.AgencyWorkspace(ctx, id)
	if err != nil {
		return api.MapError(err, "Client")
	}
	var manager *uuid.UUID
	if in.AccountManagerID != "" {
		mid, err := uuid.Parse(in.AccountManagerID)
		if err != nil {
			return api.Invalid("body.accountManagerId", mPickStaff.Text(l, nil))
		}
		people, err := s.people(ctx)
		if err != nil {
			return err
		}
		found := false
		for _, p := range people {
			if p.UserID != nil && *p.UserID == mid {
				found = true
			}
		}
		if !found {
			return api.Invalid("body.accountManagerId", mPickStaff.Text(l, nil))
		}
		manager = &mid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.AgencyWorkspaceSetManager(ctx, store.AgencyWorkspaceSetManagerParams{WorkspaceID: id, AccountManagerID: manager}); err != nil {
			return err
		}
		wsIDp := id
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsIDp, Action: "agency.client_changed", TargetType: "workspace", TargetID: id.String(), TargetName: w.Name,
			Before: map[string]any{"account_manager": w.AccountManagerName}, After: map[string]any{"account_manager": idText(manager)},
			Metadata: map[string]any{"permission": string(authz.AgencyManage), "by": me.Name}})
	})
}
