package manage

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Usage is a provider's volume as other screens show it (Deliverability,
// Overview, send readiness). It is part of the providers contract: fields
// are only ever added.
type Usage struct {
	// MonthSent is the mail sent this calendar month, counted per day in the
	// workspace time zone (provider_daily_stats.sent).
	MonthSent int64 `json:"monthSent"`
	// MonthCost is the cost to date this month in US dollars: metered price
	// times volume, the plan price, or 0 for self-hosted servers.
	MonthCost float64 `json:"monthCost"`
	// Bounce7d and Complaint7d are percentages over the last seven days
	// (today included); nil when nothing was sent or delivered. When the
	// provider reports its own rates (state.ReputationOf), those are used.
	// Complaint7d counts only mail delivered to mailbox providers that
	// report complaints (review F-2); ComplaintReason says why it is nil.
	Bounce7d        *float64 `json:"bounce7d,omitempty"`
	Complaint7d     *float64 `json:"complaint7d,omitempty"`
	ComplaintReason string   `json:"complaintReason,omitempty" enum:"no_data,not_reported"`
}

// Why a complaint rate is empty (review F-3).
const (
	RateNoData      = "no_data"
	RateNotReported = mailbox.NotReported
)

// ComplaintRate is a provider's complaint rate over days of totals, in
// percent: complaints over the mail delivered to mailbox providers that
// report complaints, on the days that counted it (review F-2). Without
// such mail it is nil, with the reason.
func ComplaintRate(t store.ProvDailyTotalsRow) (*float64, string) {
	switch {
	case t.DeliveredReporting > 0:
		v := round2(float64(t.ComplainedReporting) / float64(t.DeliveredReporting) * 100)
		return &v, ""
	case t.DeliveredCounted > 0:
		return nil, RateNotReported
	}
	return nil, RateNoData
}

// UsageOf returns the usage of each given provider of a workspace. Providers
// without any recorded volume are present with zero values.
func UsageOf(ctx context.Context, q *store.Queries, ws *store.Workspace, provs []store.Provider, now time.Time) (map[uuid.UUID]Usage, error) {
	out := make(map[uuid.UUID]Usage, len(provs))
	if len(provs) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, 0, len(provs))
	for _, p := range provs {
		ids = append(ids, p.ID)
	}
	loc := state.Location(ws.Timezone)
	today := dateOf(now, loc)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	month, err := q.ProvDailyTotals(ctx, store.ProvDailyTotalsParams{ProviderIds: ids, Since: monthStart, Until: today})
	if err != nil {
		return nil, err
	}
	week, err := q.ProvDailyTotals(ctx, store.ProvDailyTotalsParams{ProviderIds: ids, Since: today.AddDate(0, 0, -6), Until: today})
	if err != nil {
		return nil, err
	}
	byMonth := map[uuid.UUID]store.ProvDailyTotalsRow{}
	for _, r := range month {
		byMonth[r.ProviderID] = r
	}
	byWeek := map[uuid.UUID]store.ProvDailyTotalsRow{}
	for _, r := range week {
		byWeek[r.ProviderID] = r
	}
	for _, p := range provs {
		m, w := byMonth[p.ID], byWeek[p.ID]
		u := Usage{MonthSent: m.Sent, MonthCost: round2(costToDate(p, m))}
		if w.Sent > 0 {
			v := round2(float64(w.Bounced) / float64(w.Sent) * 100)
			u.Bounce7d = &v
		}
		u.Complaint7d, u.ComplaintReason = ComplaintRate(w)
		if r := state.ReputationOf(state.HealthOf(p), now); r != nil {
			b, c := r.Bounce, r.Complaint
			u.Bounce7d, u.Complaint7d, u.ComplaintReason = &b, &c, ""
		}
		out[p.ID] = u
	}
	return out, nil
}
