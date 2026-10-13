package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReadinessThresholds are the auto-pause thresholds of a send. The sending
// engine enforces them; they're stored on campaigns.thresholds as
// {"bounce":2.0,"complaint":0.10,"mailbox_complaint":0.10,"provider_errors":5,"scope":"route"}.
type ReadinessThresholds struct {
	// Bounce rate in percent.
	Bounce float64 `json:"bounce" minimum:"0.1" maximum:"20"`
	// Complaint rate in percent, overall and per mailbox provider.
	Complaint        float64 `json:"complaint" minimum:"0.01" maximum:"5"`
	MailboxComplaint float64 `json:"mailboxComplaint" minimum:"0.01" maximum:"5"`
	// ProviderErrors is the share of failed API calls over 3 minutes, in percent.
	ProviderErrors float64 `json:"providerErrors" minimum:"1" maximum:"100"`
	// Scope: pause only the affected route or mailbox provider, or the whole campaign.
	Scope string `json:"scope" enum:"route,campaign"`
}

type dbThresholds struct {
	Bounce           *float64 `json:"bounce,omitempty"`
	HardBounce       *float64 `json:"hard_bounce,omitempty"`
	Complaint        *float64 `json:"complaint,omitempty"`
	MailboxComplaint *float64 `json:"mailbox_complaint,omitempty"`
	ProviderErrors   *float64 `json:"provider_errors,omitempty"`
	Scope            string   `json:"scope,omitempty"`
}

// ParseThresholds reads a campaign's thresholds, falling back to the
// workspace's defaults (send_thresholds) and then to 2% / 0.10% / 0.10% / 5%.
func ParseThresholds(campaign, workspace json.RawMessage) ReadinessThresholds {
	t := ReadinessThresholds{Bounce: 2, Complaint: 0.1, MailboxComplaint: 0.1, ProviderErrors: 5, Scope: "route"}
	for _, raw := range []json.RawMessage{workspace, campaign} {
		var d dbThresholds
		if len(raw) == 0 || json.Unmarshal(raw, &d) != nil {
			continue
		}
		if d.HardBounce != nil {
			t.Bounce = *d.HardBounce
		}
		if d.Bounce != nil {
			t.Bounce = *d.Bounce
		}
		if d.Complaint != nil {
			t.Complaint = *d.Complaint
		}
		if d.MailboxComplaint != nil {
			t.MailboxComplaint = *d.MailboxComplaint
		}
		if d.ProviderErrors != nil {
			t.ProviderErrors = *d.ProviderErrors
		}
		if d.Scope == "route" || d.Scope == "campaign" {
			t.Scope = d.Scope
		}
	}
	return t
}

func (t ReadinessThresholds) toDB() json.RawMessage {
	r := func(v float64) *float64 { x := math.Round(v*100) / 100; return &x }
	b, _ := json.Marshal(dbThresholds{Bounce: r(t.Bounce), Complaint: r(t.Complaint), MailboxComplaint: r(t.MailboxComplaint), ProviderErrors: r(t.ProviderErrors), Scope: t.Scope})
	return b
}

// ReadinessSchedule is when the campaign goes out: now, at a local date and
// time in a time zone, or at each recipient's usual open time within 24 h of
// that time.
type ReadinessSchedule struct {
	Mode string `json:"mode" enum:"now,at,best_time"`
	// Date and Time are local to Timezone ("2026-10-14", "09:00").
	Date     string `json:"date,omitempty" pattern:"^\\d{4}-\\d{2}-\\d{2}$"`
	Time     string `json:"time,omitempty" pattern:"^([01]\\d|2[0-3]):[0-5]\\d$"`
	Timezone string `json:"timezone,omitempty" maxLength:"64"`
	// At is the resulting instant (read-only).
	At *time.Time `json:"at,omitempty" readOnly:"true"`
}

// ScheduleView turns the stored schedule into the API shape, in the
// workspace's zone when the schedule has none.
func ScheduleView(raw json.RawMessage, wsTZ string) ReadinessSchedule {
	s := ParseSchedule(raw)
	out := ReadinessSchedule{Mode: s.Mode, Timezone: s.Timezone}
	if out.Timezone == "" {
		out.Timezone = wsTZ
	}
	if s.At != "" {
		if t, err := time.Parse(time.RFC3339, s.At); err == nil {
			loc := state.Location(out.Timezone)
			lt := t.In(loc)
			out.Date, out.Time, out.At = lt.Format("2006-01-02"), lt.Format("15:04"), &t
		}
	}
	return out
}

// Resolve validates the schedule and returns its stored shape.
func (in ReadinessSchedule) Resolve(field string) (Schedule, *time.Time, error) {
	switch in.Mode {
	case "now", "":
		return Schedule{Mode: "now"}, nil, nil
	case "at", "best_time":
	default:
		return Schedule{}, nil, api.Invalid(field+".mode", "Pick when to send.")
	}
	tz := strings.TrimSpace(in.Timezone)
	loc, err := time.LoadLocation(tz)
	if tz == "" || err != nil {
		return Schedule{}, nil, api.Invalid(field+".timezone", "Pick a time zone from the list.")
	}
	at, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.Time, loc)
	if err != nil {
		return Schedule{}, nil, api.Invalid(field+".date", "Pick a date and a time.")
	}
	return Schedule{Mode: in.Mode, At: at.UTC().Format(time.RFC3339), Timezone: tz}, &at, nil
}

// ReadinessSendSettings are the editable settings of Review & send.
type ReadinessSendSettings struct {
	Thresholds *ReadinessThresholds `json:"thresholds,omitempty"`
	Schedule   *ReadinessSchedule   `json:"schedule,omitempty"`
}

// UpdateSendSettings saves the thresholds and the schedule the person chose.
// Changing the schedule after approval invalidates the approval.
func (s *Service) UpdateSendSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ReadinessSendSettings) error {
	c, err := s.Campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	if !editable(c.Status) {
		return api.Conflict("This campaign already started sending; its thresholds are on the live monitor.")
	}
	if c.Status == "scheduled" && in.Schedule != nil {
		return api.Conflict("The campaign is scheduled. Unschedule it to change when it goes out.")
	}
	thresholds, schedule := c.Thresholds, c.Schedule
	before, after := map[string]any{}, map[string]any{}
	if in.Thresholds != nil {
		t := *in.Thresholds
		if t.Scope == "" {
			t.Scope = "route"
		}
		if t.MailboxComplaint > t.Complaint*5 {
			return api.Invalid("body.thresholds.mailboxComplaint", "Keep the per-mailbox complaint threshold within five times the overall one.")
		}
		// Compared as values: Postgres hands jsonb back reformatted, so the
		// stored text never equals freshly marshalled JSON.
		if next := t.toDB(); ParseThresholds(next, nil) != ParseThresholds(thresholds, ws.SendThresholds) {
			before["thresholds"], after["thresholds"] = nonEmptyJSON(thresholds), next
			thresholds = next
		}
	}
	if in.Schedule != nil {
		sch, _, err := in.Schedule.Resolve("body.schedule")
		if err != nil {
			return err
		}
		if sch != ParseSchedule(schedule) {
			next, _ := json.Marshal(sch)
			before["schedule"], after["schedule"] = nonEmptyJSON(schedule), json.RawMessage(next)
			schedule = next
		}
	}
	if len(after) == 0 {
		return nil
	}
	// A new time doesn't change what the checks found: a run that described
	// the campaign keeps describing it (the send-time recheck runs anyway).
	var carry *uuid.UUID
	if _, ok := after["schedule"]; ok {
		if run, err := s.Latest(ctx, id); err == nil && run != nil && run.Fingerprint == s.CurrentFingerprint(ctx, ws, c) {
			if rid, err := uuid.Parse(run.ID); err == nil {
				carry = &rid
			}
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReadinessSetSendSettings(ctx, store.ReadinessSetSendSettingsParams{Thresholds: thresholds, Schedule: schedule, UpdatedBy: &p.UserID, Now: s.now(), WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		if carry != nil {
			// Fingerprint the row as Postgres stored it, as every later
			// comparison does.
			moved, err := q.ReadinessCampaign(ctx, store.ReadinessCampaignParams{WorkspaceID: ws.ID, ID: id})
			if err != nil {
				return err
			}
			if err := q.ReadinessSetFingerprint(ctx, store.ReadinessSetFingerprintParams{Fingerprint: s.CurrentFingerprint(ctx, ws, moved), ID: *carry}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.send_settings_updated", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: before, After: after})
	})
	if err != nil {
		return err
	}
	if _, ok := after["schedule"]; ok {
		s.changed(ctx, ws, id)
	}
	return nil
}

func nonEmptyJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

// ThresholdsLine is "bounce 2.0% · complaint 0.10%".
func ThresholdsLine(t ReadinessThresholds) string {
	return fmt.Sprintf("bounce %.1f%% · complaint %.2f%%", t.Bounce, t.Complaint)
}
