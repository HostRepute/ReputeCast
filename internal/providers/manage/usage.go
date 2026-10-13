package manage

import (
	"context"
	"time"

	"github.com/google/uuid"

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
	// (today included); nil when nothing was sent or delivered.
	Bounce7d    *float64 `json:"bounce7d,omitempty"`
	Complaint7d *float64 `json:"complaint7d,omitempty"`
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
		if w.Delivered > 0 {
			v := round2(float64(w.Complained) / float64(w.Delivered) * 100)
			u.Complaint7d = &v
		}
		out[p.ID] = u
	}
	return out, nil
}
