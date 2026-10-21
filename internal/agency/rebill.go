package agency

import (
	"encoding/json"
	"math"
)

// AgencyRate is how the agency rebills a client for its usage. The parts
// can be combined; a flat amount replaces everything else. A rate with no
// part means the client isn't billed.
type AgencyRate struct {
	// CostPlusPct adds a percentage to each provider's cost.
	CostPlusPct *float64 `json:"costPlusPct,omitempty" minimum:"0" maximum:"1000"`
	// Per1000Emails is a price per 1,000 emails, counted across every stream.
	Per1000Emails *float64 `json:"per1000Emails,omitempty" minimum:"0" maximum:"100000"`
	// PerSegment is a price per SMS segment delivered or failed.
	PerSegment *float64 `json:"perSegment,omitempty" minimum:"0" maximum:"1000"`
	// MonthlyFee is added once a month.
	MonthlyFee *float64 `json:"monthlyFee,omitempty" minimum:"0" maximum:"10000000"`
	// Flat replaces everything above with one monthly amount.
	Flat *float64 `json:"flat,omitempty" minimum:"0" maximum:"10000000"`
}

// Billed reports whether the rate bills anything.
func (r AgencyRate) Billed() bool {
	return r.CostPlusPct != nil || r.Per1000Emails != nil || r.PerSegment != nil || r.MonthlyFee != nil || r.Flat != nil
}

func parseRate(raw json.RawMessage) AgencyRate {
	var r AgencyRate
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &r)
	}
	return r
}

// AgencyRebill is what a client's rate makes of a month's usage. Every
// part is rounded to the cent, and To rebill is their sum, so the lines of
// the breakdown always add up to the total.
type AgencyRebill struct {
	// Cost is the provider cost, rounded.
	Cost float64 `json:"cost"`
	// CostPlus is the percentage on the cost ("15% on cost $95.04").
	CostPlus float64 `json:"costPlus"`
	// Emails is the per-1,000-emails part; Segments the per-segment part.
	Emails   float64 `json:"emails"`
	Segments float64 `json:"segments"`
	Monthly  float64 `json:"monthly"`
	Flat     float64 `json:"flat"`
	// ToRebill is nil when the client isn't billed.
	ToRebill *float64 `json:"toRebill,omitempty"`
}

// rebill applies a rate to a month's usage. costMicros is the unrounded
// provider cost (micro-dollars): the percentage is taken on it, so "cost +
// 15%" of $633.588 is $95.04 and the month comes to $1,028.63.
func rebill(rate AgencyRate, costMicros int64, emails, segments int64) AgencyRebill {
	cost := float64(costMicros) / 1e6
	out := AgencyRebill{Cost: cents(cost)}
	if !rate.Billed() {
		return out
	}
	total := 0.0
	if rate.Flat != nil {
		out.Flat = cents(*rate.Flat)
		total = out.Flat
		out.ToRebill = &total
		return out
	}
	if rate.CostPlusPct != nil {
		out.CostPlus = cents(cost * *rate.CostPlusPct / 100)
		total += out.Cost + out.CostPlus
	}
	if rate.Per1000Emails != nil {
		out.Emails = cents(float64(emails) / 1000 * *rate.Per1000Emails)
		total += out.Emails
	}
	if rate.PerSegment != nil {
		out.Segments = cents(float64(segments) * *rate.PerSegment)
		total += out.Segments
	}
	if rate.MonthlyFee != nil {
		out.Monthly = cents(*rate.MonthlyFee)
		total += out.Monthly
	}
	total = cents(total)
	out.ToRebill = &total
	return out
}

// cents rounds an amount to the cent, halves up (as money is written).
func cents(v float64) float64 {
	return math.Floor(v*100+0.5+1e-9) / 100
}
