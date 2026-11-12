package holdout

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// HoldoutPeriodItem is one period of the holdout group: its share, scope
// and when it ran.
type HoldoutPeriodItem struct {
	ID          string     `json:"id"`
	StartedAt   time.Time  `json:"startedAt"`
	EndedAt     *time.Time `json:"endedAt,omitempty"`
	BasisPoints int        `json:"basisPoints"`
	Percent     float64    `json:"percent"`
	Campaigns   bool       `json:"campaigns"`
	Automations bool       `json:"automations"`
	Open        bool       `json:"open"`
}

// HoldoutGroupResult is what one group did: its contacts, how many of them
// converted, their orders and revenue (minor units of the main currency).
type HoldoutGroupResult struct {
	Contacts   int64 `json:"contacts"`
	Converters int64 `json:"converters"`
	// Visitors were seen on a tracked site (HoldoutResults.Tracking).
	Visitors     int64 `json:"visitors"`
	Orders       int64 `json:"orders"`
	RevenueCents int64 `json:"revenueCents"`
}

// HoldoutMetric compares the groups on one measure. Values are in the
// metric's unit: conversion_rate and site_visitors in percent,
// revenue_per_contact in minor units of the main currency,
// orders_per_contact in orders. site_visitors is there only when the
// workspace tracks a site (Website tracking). Lift is the
// marketing group's over the holdout group's, in percent; Confidence is
// 1 − p (two-sided); Tested is false when there was nothing to test.
type HoldoutMetric struct {
	Key         string  `json:"key" enum:"conversion_rate,revenue_per_contact,orders_per_contact,site_visitors"`
	Holdout     float64 `json:"holdout"`
	Marketing   float64 `json:"marketing"`
	Lift        float64 `json:"lift"`
	Confidence  float64 `json:"confidence"`
	Tested      bool    `json:"tested"`
	Significant bool    `json:"significant"`
}

// HoldoutOtherCurrency is what was recorded in another currency: named
// apart, never converted.
type HoldoutOtherCurrency struct {
	Currency       string `json:"currency"`
	HoldoutCents   int64  `json:"holdoutCents"`
	MarketingCents int64  `json:"marketingCents"`
}

// HoldoutResults is Audience › Holdout: the period and span shown, its
// verdict on revenue per contact, the groups, the metrics, and whether a
// recompute is running or allowed.
type HoldoutResults struct {
	// Periods are the open period and the last closed one, newest first;
	// empty when the workspace never had a holdout group.
	Periods []HoldoutPeriodItem `json:"periods"`
	Period  *HoldoutPeriodItem  `json:"period,omitempty"`
	Window  string              `json:"window" enum:"period,30d,90d"`
	// From and To are the span's dates (omitted until measured).
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
	// Verdict: none (no holdout group ever), pending (not measured yet),
	// too_few, too_early, added, lost or no_difference.
	Verdict    string     `json:"verdict" enum:"none,pending,too_few,too_early,added,lost,no_difference"`
	ComputedAt *time.Time `json:"computedAt,omitempty"`
	// Computing is true while a requested recompute hasn't finished;
	// NextMeasureAt is when Recompute is allowed again.
	Computing       bool                   `json:"computing"`
	NextMeasureAt   *time.Time             `json:"nextMeasureAt,omitempty"`
	Holdout         HoldoutGroupResult     `json:"holdout"`
	Marketing       HoldoutGroupResult     `json:"marketing"`
	Metrics         []HoldoutMetric        `json:"metrics"`
	Currency        string                 `json:"currency"`
	OtherCurrencies []HoldoutOtherCurrency `json:"otherCurrencies"`
	// CampaignsIncluded counts the email campaigns of the span that went
	// to the holdout group too.
	CampaignsIncluded int64 `json:"campaignsIncluded"`
	MinHoldout        int64 `json:"minHoldout"`
	// Tracking: the workspace tracked a site when this span was measured,
	// so the groups count site visitors and the metrics include them.
	Tracking bool `json:"tracking"`
}

func periodItem(p store.HoldoutPeriod) HoldoutPeriodItem {
	return HoldoutPeriodItem{ID: p.ID.String(), StartedAt: p.StartedAt, EndedAt: p.EndedAt, BasisPoints: int(p.BasisPoints), Percent: Percent(p.BasisPoints),
		Campaigns: p.Campaigns, Automations: p.Automations, Open: p.EndedAt == nil}
}

func groupResult(g Group) HoldoutGroupResult {
	return HoldoutGroupResult{Contacts: g.Contacts, Converters: g.Converters, Visitors: g.Visitors, Orders: int64(math.Round(g.Orders)),
		RevenueCents: int64(math.Round(g.Revenue))}
}

// Results reads a period's span (the newest period when periodID is nil).
func (s *Service) Results(ctx context.Context, ws *store.Workspace, periodID *uuid.UUID, span string) (HoldoutResults, error) {
	if span == "" {
		span = SpanPeriod
	}
	out := HoldoutResults{Periods: []HoldoutPeriodItem{}, Window: span, Verdict: "none", Metrics: []HoldoutMetric{}, Currency: "usd",
		OtherCurrencies: []HoldoutOtherCurrency{}, MinHoldout: MinHoldout}
	recent, err := s.Q.HoldoutRecentPeriods(ctx, store.HoldoutRecentPeriodsParams{WorkspaceID: ws.ID, Lim: 2})
	if err != nil {
		return out, err
	}
	// The open period and the last closed one (Measure's pair).
	if len(recent) == 2 && recent[0].EndedAt != nil {
		recent = recent[:1]
	}
	for _, p := range recent {
		out.Periods = append(out.Periods, periodItem(p))
	}
	if len(recent) == 0 {
		return out, nil
	}
	p := recent[0]
	if periodID != nil {
		found := false
		for _, r := range recent {
			if r.ID == *periodID {
				p, found = r, true
			}
		}
		if !found {
			// An older period, still the workspace's.
			old, err := s.Q.HoldoutGetPeriod(ctx, store.HoldoutGetPeriodParams{WorkspaceID: ws.ID, ID: *periodID})
			if err != nil {
				return out, api.MapError(err, "Holdout period")
			}
			p = old
		}
	}
	item := periodItem(p)
	out.Period = &item
	if c, err := s.Q.HoldoutCurrency(ctx, ws.ID); err == nil {
		out.Currency = c
	}
	m, computed, ok, err := result(ctx, s.Q, p.ID, span)
	if err != nil {
		return out, err
	}
	if latest := recent[0]; latest.MeasureRequestedAt != nil {
		next := latest.MeasureRequestedAt.Add(MeasureEvery)
		if next.After(s.now()) {
			out.NextMeasureAt = &next
		}
		out.Computing = !ok || computed.Before(*latest.MeasureRequestedAt)
		if ok && p.ID != latest.ID {
			// Measure computes both periods at once.
			if _, c, ok2, err := result(ctx, s.Q, latest.ID, span); err == nil && ok2 {
				out.Computing = c.Before(*latest.MeasureRequestedAt)
			}
		}
	}
	if !ok {
		out.Verdict = VerdictPending
		return out, nil
	}
	out.ComputedAt, out.From, out.To = &computed, &m.From, &m.To
	out.Currency = m.Currency
	out.Holdout, out.Marketing = groupResult(m.Holdout), groupResult(m.Marketing)
	for _, x := range m.Metrics() {
		places := 6
		if x.Key == MetricRevenue {
			places = 4
		}
		out.Metrics = append(out.Metrics, HoldoutMetric{Key: x.Key, Holdout: round(x.Holdout, places), Marketing: round(x.Marketing, places), Lift: round(x.Lift, 4),
			Confidence: round(x.Confidence, 6), Tested: x.Tested, Significant: x.Significant})
	}
	for _, o := range m.OtherCurrencies {
		out.OtherCurrencies = append(out.OtherCurrencies, HoldoutOtherCurrency(o))
	}
	out.CampaignsIncluded = m.CampaignsIncluded
	out.Tracking = m.Tracking
	out.Verdict = m.Verdict(p)
	return out, nil
}

// RequestMeasure queues a recompute of the workspace's results, at most
// once every MeasureEvery.
func (s *Service) RequestMeasure(ctx context.Context, ws *store.Workspace) (HoldoutResults, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.HoldoutLockWorkspace(ctx, ws.ID); err != nil {
			return err
		}
		p, err := q.HoldoutLatestPeriod(ctx, ws.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Err(http.StatusConflict, api.CodeConflict, text(ctx, "problem.noHoldout", nil))
		}
		if err != nil {
			return err
		}
		now := s.now()
		if p.MeasureRequestedAt != nil && now.Sub(*p.MeasureRequestedAt) < MeasureEvery {
			left := MeasureEvery - now.Sub(*p.MeasureRequestedAt)
			pr := api.Err(http.StatusTooManyRequests, api.CodeRateLimited, text(ctx, "problem.measureSoon", i18n.Args{"minutes": int(math.Ceil(left.Minutes()))}))
			return huma.ErrorWithHeaders(pr, http.Header{"Retry-After": {strconv.Itoa(int(math.Ceil(left.Seconds())))}})
		}
		if err := q.HoldoutRequestMeasure(ctx, store.HoldoutRequestMeasureParams{ID: p.ID, At: &now}); err != nil {
			return err
		}
		_, err = s.Jobs.InsertTx(ctx, tx, MeasureArgs{WorkspaceID: ws.ID}, nil)
		return err
	})
	if err != nil {
		return HoldoutResults{}, err
	}
	return s.Results(ctx, ws, nil, SpanPeriod)
}

type resultsInput struct {
	api.WorkspaceInput
	Period string `query:"period" format:"uuid" required:"false" doc:"A period's id; the newest by default"`
	Window string `query:"window" enum:"period,30d,90d" required:"false" doc:"The whole period (default) or its last 30 or 90 days"`
}

// Register adds Audience › Holdout's results (reports:view).
func Register(h huma.API, s *Service) {
	tags := []string{"Holdout group"}
	api.Get(h, api.Op{ID: "get-holdout-results", Path: "/workspaces/{workspace}/holdout/results", Summary: "What marketing added, measured against the holdout group",
		Description: "The holdout group's and everyone else's conversion rate, revenue and orders per contact, with the lift and its confidence. " +
			"Measured every night at 04:00 workspace time.",
		Tags: tags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *resultsInput) (*api.Body[HoldoutResults], error) {
			var id *uuid.UUID
			if in.Period != "" {
				u, err := uuid.Parse(in.Period)
				if err != nil {
					return nil, api.NotFound("Holdout period")
				}
				id = &u
			}
			out, err := s.Results(ctx, api.WorkspaceFrom(ctx), id, in.Window)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "measure-holdout", Path: "/workspaces/{workspace}/holdout/measure", Summary: "Recompute the holdout group's results",
		Description: "At most once every 10 minutes (429 with Retry-After otherwise). The results say computing until the new ones are ready.",
		Tags:        tags, Access: api.Workspace(authz.ReportsView), Status: http.StatusAccepted},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[HoldoutResults], error) {
			out, err := s.RequestMeasure(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
}
