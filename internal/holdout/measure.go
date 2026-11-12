package holdout

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/abstats"
	"github.com/hostrepute/reputecast/internal/store"
)

// The measurement: for the open period and the last closed one, over
// three spans (the whole period, its last 30 and 90 days), the population
// of contacts split into the holdout group and everyone else ("the
// marketing group"), with what each group did: converters, orders and
// revenue. It runs nightly at 04:00 workspace time and on request (at most
// once every 10 minutes). What it stores is the groups' sums; the metrics,
// their lift and significance are computed when the results are read, so
// a metric can be added without measuring again.

// Spans of a measurement.
const (
	SpanPeriod = "period"
	Span30     = "30d"
	Span90     = "90d"
)

// Spans lists every span.
var Spans = []string{SpanPeriod, Span30, Span90}

// spanDays is how far back a span reaches from the period's end (0: to
// its start). A span never reaches before the period started: contacts
// weren't held out then.
var spanDays = map[string]int{SpanPeriod: 0, Span30: 30, Span90: 90}

// Thresholds of the verdict.
const (
	// MinHoldout is how many contacts the holdout group needs before its
	// results mean anything.
	MinHoldout = 1000
	// TooEarly is how long a period must have run.
	TooEarly = 14 * 24 * time.Hour
	// Confidence is what a difference needs to count (two-sided).
	Confidence = 0.95
	// MeasureEvery is how often a person may ask for a recompute.
	MeasureEvery = 10 * time.Minute
)

// Inserter queues jobs in a transaction (the River client).
type Inserter interface {
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Service measures holdout periods and reads the results.
type Service struct {
	Pool *pgxpool.Pool
	Q    *store.Queries
	Jobs Inserter
	Log  *slog.Logger
	Now  func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Group is what one group did in a span: sums over its contacts, with the
// sums of squares that their variances need. Revenue is in minor units of
// Measured.Currency.
type Group struct {
	Contacts   int64 `json:"contacts"`
	Converters int64 `json:"converters"`
	// Visitors were seen on a tracked site (Measured.Tracking).
	Visitors  int64   `json:"visitors"`
	Orders    float64 `json:"orders"`
	OrdersSq  float64 `json:"orders_sq"`
	Revenue   float64 `json:"revenue"`
	RevenueSq float64 `json:"revenue_sq"`
}

// OtherCurrency is what a span recorded in a currency other than the main
// one, per group: named apart, never converted or added.
type OtherCurrency struct {
	Currency       string `json:"currency"`
	HoldoutCents   int64  `json:"holdout_cents"`
	MarketingCents int64  `json:"marketing_cents"`
}

// Measured is holdout_results.data: one span of one period.
type Measured struct {
	From              time.Time       `json:"from"`
	To                time.Time       `json:"to"`
	Currency          string          `json:"currency"`
	Holdout           Group           `json:"holdout"`
	Marketing         Group           `json:"marketing"`
	OtherCurrencies   []OtherCurrency `json:"other_currencies"`
	CampaignsIncluded int64           `json:"campaigns_included"`
	// Tracking: the workspace tracked a site (M62) when this was measured,
	// so the groups count site visitors.
	Tracking bool `json:"tracking,omitempty"`
}

// spanWindow is when a span of a period runs: back from its end (now for
// the open one), never before it started.
func spanWindow(p store.HoldoutPeriod, span string, now time.Time) (time.Time, time.Time) {
	to := End(p, now)
	from := p.StartedAt
	if d := spanDays[span]; d > 0 {
		if f := to.AddDate(0, 0, -d); f.After(from) {
			from = f
		}
	}
	return from, to
}

// measureSpan measures one span of a period.
func measureSpan(ctx context.Context, q *store.Queries, p store.HoldoutPeriod, span, currency string, tracking bool, now time.Time) (Measured, error) {
	from, to := spanWindow(p, span, now)
	out := Measured{From: from, To: to, Currency: currency, OtherCurrencies: []OtherCurrency{}, Tracking: tracking}
	rows, err := q.HoldoutMeasureGroups(ctx, store.HoldoutMeasureGroupsParams{Seed: p.Seed, BasisPoints: p.BasisPoints, WorkspaceID: p.WorkspaceID,
		Since: from, Until: to, Currency: currency})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		g := Group{Contacts: r.Contacts, Converters: r.Converters, Orders: r.Orders, OrdersSq: r.OrdersSq, Revenue: r.Revenue, RevenueSq: r.RevenueSq}
		if tracking {
			g.Visitors = r.Visitors
		}
		if r.Held {
			out.Holdout = g
		} else {
			out.Marketing = g
		}
	}
	others, err := q.HoldoutOtherCurrencies(ctx, store.HoldoutOtherCurrenciesParams{Seed: p.Seed, BasisPoints: p.BasisPoints, WorkspaceID: p.WorkspaceID,
		Since: from, Until: to, Currency: currency})
	if err != nil {
		return out, err
	}
	for _, o := range others {
		out.OtherCurrencies = append(out.OtherCurrencies, OtherCurrency{Currency: o.Currency, HoldoutCents: o.HoldoutCents, MarketingCents: o.MarketingCents})
	}
	if p.Campaigns {
		if out.CampaignsIncluded, err = q.HoldoutCampaignsIncluded(ctx, store.HoldoutCampaignsIncludedParams{WorkspaceID: p.WorkspaceID, Since: from, Until: to}); err != nil {
			return out, err
		}
	}
	return out, nil
}

// Measure computes and stores the results of the workspace's open period
// and its last closed one, every span.
func (s *Service) Measure(ctx context.Context, workspaceID uuid.UUID) error {
	periods, err := s.Q.HoldoutRecentPeriods(ctx, store.HoldoutRecentPeriodsParams{WorkspaceID: workspaceID, Lim: 2})
	if err != nil {
		return err
	}
	currency, err := s.Q.HoldoutCurrency(ctx, workspaceID)
	if err != nil {
		return err
	}
	tracking, err := s.Q.HoldoutTracking(ctx, workspaceID)
	if err != nil {
		return err
	}
	now := s.now()
	closed := false
	for _, p := range periods {
		if p.EndedAt != nil {
			if closed {
				break // only the last closed one
			}
			closed = true
		}
		for _, span := range Spans {
			m, err := measureSpan(ctx, s.Q, p, span, currency, tracking, now)
			if err != nil {
				return fmt.Errorf("holdout %s %s: %w", p.ID, span, err)
			}
			data, err := json.Marshal(m)
			if err != nil {
				return err
			}
			if err := s.Q.HoldoutSaveResult(ctx, store.HoldoutSaveResultParams{PeriodID: p.ID, Span: span, ComputedAt: now, Data: data}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Metric keys, in the order the results show them. A metric is computed
// from the stored sums when the results are read.
const (
	MetricConversion = "conversion_rate"
	MetricRevenue    = "revenue_per_contact"
	MetricOrders     = "orders_per_contact"
	// MetricVisitors is the share of a group seen on a tracked site (M62),
	// only when the workspace tracks one.
	MetricVisitors = "site_visitors"
)

// Metric compares the two groups on one measure: the holdout group's
// value and the marketing group's (conversion rate in percent, revenue per
// contact in minor units, orders per contact), the marketing group's lift
// over the holdout's in percent, and the test's confidence (1 − p,
// two-sided). Tested is false when the test can't run or the holdout's
// value is 0 (no lift to express).
type Metric struct {
	Key         string
	Holdout     float64
	Marketing   float64
	Lift        float64
	Confidence  float64
	Tested      bool
	Significant bool
}

// meanVar is the mean and unbiased variance of n values from their sum
// and sum of squares.
func meanVar(sum, sq float64, n int64) (float64, float64) {
	if n <= 0 {
		return 0, 0
	}
	mean := sum / float64(n)
	if n < 2 {
		return mean, 0
	}
	v := (sq - float64(n)*mean*mean) / float64(n-1)
	if v < 0 { // rounding
		v = 0
	}
	return mean, v
}

func metricOf(key string, holdout, marketing, p float64, ok bool) Metric {
	m := Metric{Key: key, Holdout: holdout, Marketing: marketing}
	if ok && holdout > 0 {
		m.Tested = true
		m.Lift = (marketing - holdout) / holdout * 100
		m.Confidence = 1 - p
		m.Significant = m.Confidence >= Confidence
	}
	return m
}

// Metrics compares the groups of a span: conversion rate (a two-proportion
// z-test), revenue and orders per contact (Welch's t-test), and, when the
// workspace tracks a site, the share of site visitors (z-test).
func (m Measured) Metrics() []Metric {
	h, k := m.Holdout, m.Marketing
	out := make([]Metric, 0, 4)
	rh, _ := abstats.Rate(h.Converters, h.Contacts)
	rk, _ := abstats.Rate(k.Converters, k.Contacts)
	_, _, p, ok := abstats.ZTest(h.Converters, h.Contacts, k.Converters, k.Contacts)
	out = append(out, metricOf(MetricConversion, rh, rk, p, ok))
	mh, vh := meanVar(h.Revenue, h.RevenueSq, h.Contacts)
	mk, vk := meanVar(k.Revenue, k.RevenueSq, k.Contacts)
	_, _, p, ok = abstats.WelchT(mh, vh, h.Contacts, mk, vk, k.Contacts)
	out = append(out, metricOf(MetricRevenue, mh, mk, p, ok))
	oh, voh := meanVar(h.Orders, h.OrdersSq, h.Contacts)
	ok2, vok := meanVar(k.Orders, k.OrdersSq, k.Contacts)
	_, _, p, ok = abstats.WelchT(oh, voh, h.Contacts, ok2, vok, k.Contacts)
	out = append(out, metricOf(MetricOrders, oh, ok2, p, ok))
	if m.Tracking {
		vh, _ := abstats.Rate(h.Visitors, h.Contacts)
		vk, _ := abstats.Rate(k.Visitors, k.Contacts)
		_, _, p, ok = abstats.ZTest(h.Visitors, h.Contacts, k.Visitors, k.Contacts)
		out = append(out, metricOf(MetricVisitors, vh, vk, p, ok))
	}
	return out
}

// Verdicts of a span, on revenue per contact.
const (
	VerdictPending      = "pending"       // not measured yet
	VerdictTooFew       = "too_few"       // fewer than MinHoldout in the holdout group
	VerdictTooEarly     = "too_early"     // the open period is younger than TooEarly
	VerdictAdded        = "added"         // marketing brought more, significantly
	VerdictLost         = "lost"          // the holdout group brought more, significantly
	VerdictNoDifference = "no_difference" // not significant
)

// Verdict says what a span's results show.
func (m Measured) Verdict(p store.HoldoutPeriod) string {
	if m.Holdout.Contacts < MinHoldout {
		return VerdictTooFew
	}
	if p.EndedAt == nil && m.To.Sub(p.StartedAt) < TooEarly {
		return VerdictTooEarly
	}
	for _, x := range m.Metrics() {
		if x.Key != MetricRevenue {
			continue
		}
		switch {
		case x.Significant && x.Lift > 0:
			return VerdictAdded
		case x.Significant && x.Lift < 0:
			return VerdictLost
		}
	}
	return VerdictNoDifference
}

// result reads a stored span; ok is false when it was never measured.
func result(ctx context.Context, q *store.Queries, periodID uuid.UUID, span string) (Measured, time.Time, bool, error) {
	r, err := q.HoldoutResult(ctx, store.HoldoutResultParams{PeriodID: periodID, Span: span})
	if errors.Is(err, pgx.ErrNoRows) {
		return Measured{}, time.Time{}, false, nil
	}
	if err != nil {
		return Measured{}, time.Time{}, false, err
	}
	var m Measured
	if err := json.Unmarshal(r.Data, &m); err != nil {
		return Measured{}, time.Time{}, false, err
	}
	if m.OtherCurrencies == nil {
		m.OtherCurrencies = []OtherCurrency{}
	}
	return m, r.ComputedAt, true, nil
}

// round keeps a metric readable in JSON (no float noise).
func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
