package audience

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// SunsetPolicy decides the engagement tiers and what happens to contacts
// who stop opening (List health › Sunset policy).
type SunsetPolicy struct {
	Enabled bool `json:"enabled"`
	// EnabledAt is the day the policy was switched on (2026-09-01).
	EnabledAt string `json:"enabledAt,omitempty"`
	// ActiveDays: Active until this many days without an open (30).
	ActiveDays int `json:"activeDays" minimum:"1" maximum:"3650"`
	// AtRiskDays: At risk from this many days (91); Cooling is in between.
	AtRiskDays int `json:"atRiskDays" minimum:"2" maximum:"3650"`
	// SunsetDays: Sunset from this many days (181).
	SunsetDays int `json:"sunsetDays" minimum:"3" maximum:"3650"`
	// CountMachineOpens counts Apple Mail prefetches as engagement.
	CountMachineOpens bool `json:"countMachineOpens"`
	// Reengage sends one re-engagement email when a contact enters Sunset
	// (the contact is tagged "sunset"; the automation starts on that tag).
	Reengage           bool   `json:"reengage"`
	ReengageAutomation string `json:"reengageAutomation,omitempty" doc:"Name of the automation that sends it"`
	// StopBroadcasts stops Broadcasts StopAfterDays after entering Sunset if
	// there's still no open. Automations and System mail continue.
	StopBroadcasts bool `json:"stopBroadcasts"`
	StopAfterDays  int  `json:"stopAfterDays" minimum:"1" maximum:"365"`
	// DeleteAfter erases contacts after DeleteAfterMonths in Sunset.
	DeleteAfter       bool `json:"deleteAfter"`
	DeleteAfterMonths int  `json:"deleteAfterMonths" minimum:"1" maximum:"120"`
}

// policyDoc is how the policy is stored on the workspace (snake_case, and
// the older {"after_days":180} shape from workspace creation).
type policyDoc struct {
	Enabled            *bool  `json:"enabled,omitempty"`
	EnabledAt          string `json:"enabled_at,omitempty"`
	ActiveDays         int    `json:"active_days,omitempty"`
	AtRiskDays         int    `json:"at_risk_days,omitempty"`
	SunsetDays         int    `json:"sunset_days,omitempty"`
	AfterDays          int    `json:"after_days,omitempty"`
	CountMachineOpens  bool   `json:"count_machine_opens,omitempty"`
	Reengage           *bool  `json:"reengage,omitempty"`
	ReengageAutomation string `json:"reengage_automation,omitempty"`
	StopBroadcasts     *bool  `json:"stop_broadcasts,omitempty"`
	StopAfterDays      int    `json:"stop_after_days,omitempty"`
	DeleteAfter        bool   `json:"delete_after,omitempty"`
	DeleteAfterMonths  int    `json:"delete_after_months,omitempty"`
}

// ParsePolicy reads a workspace's stored policy, filling defaults.
func ParsePolicy(raw json.RawMessage) SunsetPolicy {
	var d policyDoc
	_ = json.Unmarshal(raw, &d)
	p := SunsetPolicy{Enabled: true, ActiveDays: 30, AtRiskDays: 91, SunsetDays: 181, Reengage: true, ReengageAutomation: "Re-engagement",
		StopBroadcasts: true, StopAfterDays: 30, DeleteAfterMonths: 12}
	if d.Enabled != nil {
		p.Enabled = *d.Enabled
	}
	p.EnabledAt = d.EnabledAt
	if d.ActiveDays > 0 {
		p.ActiveDays = d.ActiveDays
	}
	if d.AtRiskDays > 0 {
		p.AtRiskDays = d.AtRiskDays
	}
	switch {
	case d.SunsetDays > 0:
		p.SunsetDays = d.SunsetDays
	case d.AfterDays > 0:
		p.SunsetDays = d.AfterDays + 1
	}
	p.CountMachineOpens = d.CountMachineOpens
	if d.Reengage != nil {
		p.Reengage = *d.Reengage
	}
	if d.ReengageAutomation != "" {
		p.ReengageAutomation = d.ReengageAutomation
	}
	if d.StopBroadcasts != nil {
		p.StopBroadcasts = *d.StopBroadcasts
	}
	if d.StopAfterDays > 0 {
		p.StopAfterDays = d.StopAfterDays
	}
	p.DeleteAfter = d.DeleteAfter
	if d.DeleteAfterMonths > 0 {
		p.DeleteAfterMonths = d.DeleteAfterMonths
	}
	return p
}

func (p SunsetPolicy) doc() policyDoc {
	return policyDoc{Enabled: &p.Enabled, EnabledAt: p.EnabledAt, ActiveDays: p.ActiveDays, AtRiskDays: p.AtRiskDays, SunsetDays: p.SunsetDays,
		CountMachineOpens: p.CountMachineOpens, Reengage: &p.Reengage, ReengageAutomation: p.ReengageAutomation, StopBroadcasts: &p.StopBroadcasts,
		StopAfterDays: p.StopAfterDays, DeleteAfter: p.DeleteAfter, DeleteAfterMonths: p.DeleteAfterMonths}
}

// Validate checks that the tier thresholds are in order.
func (p SunsetPolicy) Validate() error {
	if p.ActiveDays < 1 {
		return api.Invalid("body.activeDays", "Active needs at least 1 day.")
	}
	if p.AtRiskDays <= p.ActiveDays+1 {
		return api.Invalid("body.atRiskDays", fmt.Sprintf("At risk must start after %d days so Cooling has room.", p.ActiveDays+1))
	}
	if p.SunsetDays <= p.AtRiskDays {
		return api.Invalid("body.sunsetDays", fmt.Sprintf("Sunset must start after At risk (%d days).", p.AtRiskDays))
	}
	if p.StopBroadcasts && p.StopAfterDays < 1 {
		return api.Invalid("body.stopAfterDays", "Stop Broadcasts at least 1 day after Sunset.")
	}
	if p.DeleteAfter && p.DeleteAfterMonths < 1 {
		return api.Invalid("body.deleteAfterMonths", "Delete after at least 1 month.")
	}
	return nil
}

// SavePolicy stores the sunset policy. Tiers change at the next nightly run.
func (s *Service) SavePolicy(ctx context.Context, ws *store.Workspace, p SunsetPolicy) (SunsetPolicy, error) {
	if err := p.Validate(); err != nil {
		return p, err
	}
	before := ParsePolicy(ws.SunsetPolicy)
	if p.Enabled && (!before.Enabled || before.EnabledAt == "") {
		p.EnabledAt = s.now().In(workspaceLocation(ws)).Format("2006-01-02")
	} else if p.Enabled {
		p.EnabledAt = before.EnabledAt
	} else {
		p.EnabledAt = ""
	}
	raw, _ := json.Marshal(p.doc())
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetSunsetPolicy(ctx, store.SetSunsetPolicyParams{ID: ws.ID, SunsetPolicy: raw}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sunset_policy.update", TargetType: "workspace", TargetID: ws.ID.String(),
			TargetName: ws.Name, Before: before, After: p})
	})
	return p, err
}

// engagementExpr is the date a contact last engaged under policy p: the
// latest human open (or any open when machine opens count) or click, or
// the day they joined when they never engaged. Imported engagement history
// counts, so a contact imported today who last opened in June is At risk.
func engagementExpr(p SunsetPolicy) string {
	open := "c.last_human_open_at"
	if p.CountMachineOpens {
		open = "c.last_open_at"
	}
	return "coalesce(greatest(" + open + ", c.last_click_at), c.created_at)"
}

// tierExpr assigns the tier from the days since engagement.
func tierExpr(p SunsetPolicy, now string) string {
	e := engagementExpr(p)
	return fmt.Sprintf(`CASE WHEN %[1]s > %[2]s - interval '%[3]d days' THEN 'active'
		WHEN %[1]s > %[2]s - interval '%[4]d days' THEN 'cooling'
		WHEN %[1]s > %[2]s - interval '%[5]d days' THEN 'at_risk'
		ELSE 'sunset' END`, e, now, p.ActiveDays+1, p.AtRiskDays, p.SunsetDays)
}

// TierFor assigns a tier in Go, the same way the nightly SQL does, for
// contacts created between runs (imports).
func TierFor(p SunsetPolicy, lastHumanOpen, lastOpen, lastClick *time.Time, created, now time.Time) string {
	open := lastHumanOpen
	if p.CountMachineOpens {
		open = lastOpen
	}
	var e time.Time
	for _, t := range []*time.Time{open, lastClick} {
		if t != nil && t.After(e) {
			e = *t
		}
	}
	if e.IsZero() {
		e = created
	}
	switch age := now.Sub(e); {
	case age < time.Duration(p.ActiveDays+1)*24*time.Hour:
		return "active"
	case age < time.Duration(p.AtRiskDays)*24*time.Hour:
		return "cooling"
	case age < time.Duration(p.SunsetDays)*24*time.Hour:
		return "at_risk"
	}
	return "sunset"
}
