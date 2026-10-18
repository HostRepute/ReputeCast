package verification

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

// Recipient is one of a send's final recipients, as Send Readiness has them.
type Recipient struct {
	Email string
	Tier  string
}

// AudienceFacts are a send's address-check results for Send Readiness's
// Audience quality (internal/readiness converts them).
type AudienceFacts struct {
	Connected   bool
	Service     string
	ServiceName string
	Price       float64
	Balance     *int64

	AtRisk, AtRiskUnchecked int64
	Sunset, SunsetUnchecked int64
	AtRiskCheckedOn         *time.Time
	SunsetCheckedOn         *time.Time

	Risky          int64
	RiskyReasons   map[string]int64
	RiskyEmails    []string
	RiskyCheckedOn *time.Time

	InvalidSuppressed int64
	InvalidCheckedOn  *time.Time
}

// AudienceChecks reads the latest results of a send's final recipients
// and of its suppressed contacts.
func (s *Service) AudienceChecks(ctx context.Context, ws *store.Workspace, recipients []Recipient, suppressed []string) (AudienceFacts, error) {
	out := AudienceFacts{RiskyReasons: map[string]int64{}, RiskyEmails: []string{}}
	chosen, all, err := s.pick(ctx, ws, "")
	if err != nil {
		return out, err
	}
	out.Connected = len(all) > 0
	if chosen != nil {
		out.Service, out.ServiceName = chosen.Service, ServiceName(chosen.Service)
		out.Price, out.Balance = Money(chosen.PriceMicros).Dollars(), chosen.Balance
	}
	emails := make([]string, 0, len(recipients))
	for _, r := range recipients {
		emails = append(emails, strings.ToLower(r.Email))
	}
	checks := map[string]store.VerifChecksByEmailRow{}
	for start := 0; start < len(emails); start += 10000 {
		end := min(start+10000, len(emails))
		rows, err := s.Q.VerifChecksByEmail(ctx, store.VerifChecksByEmailParams{WorkspaceID: ws.ID, Emails: emails[start:end]})
		if err != nil {
			return out, err
		}
		for _, r := range rows {
			checks[r.Email] = r
		}
	}
	since := s.now().Add(-SkipWithin)
	later := func(t **time.Time, at time.Time) {
		if *t == nil || at.After(**t) {
			v := at
			*t = &v
		}
	}
	for i, r := range recipients {
		c, ok := checks[emails[i]]
		recent := ok && c.CheckedAt.After(since)
		switch r.Tier {
		case "at_risk":
			out.AtRisk++
			if !recent {
				out.AtRiskUnchecked++
			} else {
				later(&out.AtRiskCheckedOn, c.CheckedAt)
			}
		case "sunset":
			out.Sunset++
			if !recent {
				out.SunsetUnchecked++
			} else {
				later(&out.SunsetCheckedOn, c.CheckedAt)
			}
		}
		if ok && c.Result == string(adapter.Risky) {
			out.Risky++
			out.RiskyReasons[c.Reason]++
			out.RiskyEmails = append(out.RiskyEmails, emails[i])
			later(&out.RiskyCheckedOn, c.CheckedAt)
		}
	}
	if len(suppressed) > 0 {
		lower := make([]string, len(suppressed))
		for i, e := range suppressed {
			lower[i] = strings.ToLower(e)
		}
		rows, err := s.Q.VerifChecksByEmail(ctx, store.VerifChecksByEmailParams{WorkspaceID: ws.ID, Emails: lower})
		if err != nil {
			return out, err
		}
		for _, r := range rows {
			if r.Result == string(adapter.Invalid) {
				out.InvalidSuppressed++
				later(&out.InvalidCheckedOn, r.CheckedAt)
			}
		}
	}
	return out, nil
}

// RiskySegment makes or refreshes a static segment of the contacts with
// these addresses ("Risky addresses · Re-engagement wave 3"), for a
// campaign to exclude.
func (s *Service) RiskySegment(ctx context.Context, ws *store.Workspace, campaign string, emails []string) (uuid.UUID, error) {
	name := "Risky addresses · " + campaign
	if len(name) > 120 {
		name = name[:120]
	}
	var id uuid.UUID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		err := tx.QueryRow(ctx, `SELECT id FROM segments WHERE workspace_id = $1 AND name = $2 AND refresh = 'static'`, ws.ID, name).Scan(&id)
		switch {
		case err == nil:
			if err := q.ClearSegmentMembers(ctx, id); err != nil {
				return err
			}
		case errors.Is(err, pgx.ErrNoRows):
			id = uuid.Must(uuid.NewV7())
			def, _ := json.Marshal(map[string]any{"op": "and", "groups": []any{}})
			if _, err := q.CreateSegment(ctx, store.CreateSegmentParams{ID: id, WorkspaceID: ws.ID, Name: name,
				Description: "Addresses a verification service found risky, left out of " + campaign + " by Send Readiness.",
				Mode:        "builder", Refresh: "static", Definition: def, CreatedBy: actor(ctx)}); err != nil {
				return err
			}
		default:
			return err
		}
		contacts, err := q.VerifContactsByEmail(ctx, store.VerifContactsByEmailParams{WorkspaceID: ws.ID, Emails: emails})
		if err != nil {
			return err
		}
		ids := make([]uuid.UUID, len(contacts))
		for i, c := range contacts {
			ids[i] = c.ID
		}
		return q.AddSegmentMembers(ctx, store.AddSegmentMembersParams{SegmentID: id, ContactIds: ids})
	})
	return id, err
}
