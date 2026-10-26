// Package rules holds a workspace's traffic rules (Sending providers ›
// Traffic rules, review 2026-10 M34): the canary defaults (GS-10), the
// complaint watch after a send (GS-6), the thresholds by bounce type
// (GS-11) and the mailbox provider response rules (GS-12). The sending
// engine applies them; this package reads, validates and stores them and
// serves the screen's API.
package rules

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Canary is when a send starts with a canary and how it is judged.
type Canary struct {
	// Enabled runs a canary automatically when one of the reasons below
	// applies; a person can still choose one in Review & send.
	Enabled bool `json:"enabled"`
	// NewDomain: the sending domain or IP is new (first 30 days of sending
	// from it).
	NewDomain bool `json:"newDomain"`
	// NeverMailed: some recipients never got mail from this domain.
	NeverMailed bool `json:"neverMailed"`
	// OverUsual: the audience is over twice the usual size (the median of
	// the campaign's last 8 sends, or the workspace's).
	OverUsual bool `json:"overUsual"`
	// Percent of the audience, the most engaged first.
	Percent float64 `json:"percent" minimum:"1" maximum:"20"`
	// Seeds adds the workspace's connected seed inboxes.
	Seeds bool `json:"seeds"`
	// WatchMinutes is how long the canary is watched before the gates.
	WatchMinutes int         `json:"watchMinutes" enum:"10,20,30,60"`
	Gates        CanaryGates `json:"gates"`
}

// CanaryGates are what the canary must stay under, in percent (SeedSpam: a
// count of seed inboxes).
type CanaryGates struct {
	Bounce      float64 `json:"bounce" minimum:"0.1" maximum:"20"`
	BlockBounce float64 `json:"blockBounce" minimum:"0.01" maximum:"20"`
	Deferral    float64 `json:"deferral" minimum:"1" maximum:"100"`
	Complaint   float64 `json:"complaint" minimum:"0.01" maximum:"5"`
	SeedSpam    int     `json:"seedSpam" minimum:"0" maximum:"100"`
}

// DefaultCanary is the canary of a workspace that set none.
var DefaultCanary = Canary{Enabled: true, NewDomain: true, NeverMailed: true, OverUsual: true, Percent: 5, Seeds: true, WatchMinutes: 20,
	Gates: CanaryGates{Bounce: 1.0, BlockBounce: 0.2, Deferral: 15, Complaint: 0.10, SeedSpam: 2}}

// Watch is the complaint watch after a send: complaints keep arriving for
// days after delivery.
type Watch struct {
	Enabled bool `json:"enabled"`
	// Hours the watch lasts after the send ends.
	Hours int `json:"hours" enum:"24,48,72,168"`
	// Alert tells the people who can resume sends when a mailbox provider
	// passes its complaint limit during the watch.
	Alert bool `json:"alert"`
	// Postmaster reads Gmail's daily spam rate from Postmaster Tools.
	Postmaster bool `json:"postmaster"`
}

// DefaultWatch is the watch of a workspace that set none.
var DefaultWatch = Watch{Enabled: true, Hours: 72, Alert: true, Postmaster: true}

// ParseCanary reads stored canary settings over the defaults.
func ParseCanary(raw []byte) Canary {
	c := DefaultCanary
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &c)
	}
	if c.Percent <= 0 || c.Percent > 20 {
		c.Percent = DefaultCanary.Percent
	}
	switch c.WatchMinutes {
	case 10, 20, 30, 60:
	default:
		c.WatchMinutes = DefaultCanary.WatchMinutes
	}
	return c
}

// ParseWatch reads stored watch settings over the defaults.
func ParseWatch(raw []byte) Watch {
	w := DefaultWatch
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &w)
	}
	switch w.Hours {
	case 24, 48, 72, 168:
	default:
		w.Hours = DefaultWatch.Hours
	}
	return w
}

// Load reads a workspace's canary and watch settings (the defaults when it
// stored none).
func Load(ctx context.Context, q *store.Queries, ws uuid.UUID) (Canary, Watch, error) {
	rows, err := q.TrafficRulesGet(ctx, ws)
	if err != nil {
		return DefaultCanary, DefaultWatch, err
	}
	if len(rows) == 0 {
		return DefaultCanary, DefaultWatch, nil
	}
	return ParseCanary(rows[0].Canary), ParseWatch(rows[0].Watch), nil
}
