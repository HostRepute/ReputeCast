package readiness

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/holdout"
	"github.com/hostrepute/reputecast/internal/store"
)

// The holdout group's settings (M63 G-33), next to the marketing cap: a
// period per set of settings (holdout_periods). Saving different settings
// ends the open period and opens the next one with the same seed, so the
// same people stay held out (a larger share keeps all of them); "Start
// over" opens one with a new seed. Campaigns pick a change up from their
// next audience count, automations from their next step.

// The share and scope offered before a workspace ever used the holdout.
const (
	defaultHoldoutBasisPoints = 500
)

// holdoutSettings is the holdout group as Settings shows it.
func holdoutSettings(ctx context.Context, q *store.Queries, wsID uuid.UUID) (HoldoutSettings, error) {
	out := HoldoutSettings{BasisPoints: defaultHoldoutBasisPoints, Percent: holdout.Percent(defaultHoldoutBasisPoints), Campaigns: true, Automations: true}
	last, err := q.HoldoutLatestPeriod(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Measured = true
	out.BasisPoints, out.Percent, out.Campaigns, out.Automations = int(last.BasisPoints), holdout.Percent(last.BasisPoints), last.Campaigns, last.Automations
	if last.EndedAt != nil {
		return out, nil
	}
	out.On = true
	started := last.StartedAt
	out.StartedAt = &started
	if out.HeldOut, err = q.HoldoutHeldOutCount(ctx, store.HoldoutHeldOutCountParams{WorkspaceID: wsID, Seed: last.Seed, BasisPoints: last.BasisPoints}); err != nil {
		return out, err
	}
	return out, nil
}

// holdoutAudit is a period's settings as the audit log keeps them.
func holdoutAudit(p *store.HoldoutPeriod) map[string]any {
	if p == nil {
		return map[string]any{"on": false}
	}
	return map[string]any{"on": true, "basis_points": p.BasisPoints, "campaigns": p.Campaigns, "automations": p.Automations, "period": p.ID.String()}
}

// newSeed is a random seed for a new set of people.
func newSeed() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.BigEndian.Uint64(b[:]))
}

// SaveHoldout turns the holdout group on or off or changes its share and
// scope (workspace:manage, audited as holdout.updated).
func (s *Service) SaveHoldout(ctx context.Context, ws *store.Workspace, p *authz.Principal, in HoldoutInput) (SendingRulesView, error) {
	if in.BasisPoints < holdout.MinBasisPoints || in.BasisPoints > holdout.MaxBasisPoints {
		return SendingRulesView{}, api.Invalid("body.basisPoints", say(ctx, "problem.holdoutShare", nil))
	}
	if in.On && !in.Campaigns && !in.Automations {
		return SendingRulesView{}, api.Invalid("body.campaigns", say(ctx, "problem.holdoutScope", nil))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.HoldoutLockWorkspace(ctx, ws.ID); err != nil {
			return err
		}
		open, err := holdout.Open(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if open == nil && !in.On {
			return nil
		}
		if open != nil && in.On && int(open.BasisPoints) == in.BasisPoints && open.Campaigns == in.Campaigns && open.Automations == in.Automations {
			return nil
		}
		now := s.now()
		if err := q.HoldoutEndPeriod(ctx, store.HoldoutEndPeriodParams{WorkspaceID: ws.ID, At: now}); err != nil {
			return err
		}
		var next *store.HoldoutPeriod
		if in.On {
			// The same people as the last period, unless there was none.
			seed := newSeed()
			if last, err := q.HoldoutLatestPeriod(ctx, ws.ID); err == nil {
				seed = last.Seed
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if next, err = s.openHoldout(ctx, q, ws, p, seed, int32(in.BasisPoints), in.Campaigns, in.Automations, ""); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "holdout.updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: holdoutAudit(open), After: holdoutAudit(next)})
	})
	if err != nil {
		return SendingRulesView{}, err
	}
	return s.SendingRules(ctx, ws, p)
}

// RestartHoldout ends the open period and holds out a new random share of
// contacts from now on (workspace:manage, audited as holdout.restarted).
func (s *Service) RestartHoldout(ctx context.Context, ws *store.Workspace, p *authz.Principal, in HoldoutRestartInput) (SendingRulesView, error) {
	reason := strings.TrimSpace(in.Reason)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.HoldoutLockWorkspace(ctx, ws.ID); err != nil {
			return err
		}
		open, err := holdout.Open(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if open == nil {
			return api.Err(http.StatusConflict, api.CodeConflict, say(ctx, "problem.holdoutOff", nil))
		}
		if err := q.HoldoutEndPeriod(ctx, store.HoldoutEndPeriodParams{WorkspaceID: ws.ID, At: s.now()}); err != nil {
			return err
		}
		next, err := s.openHoldout(ctx, q, ws, p, newSeed(), open.BasisPoints, open.Campaigns, open.Automations, reason)
		if err != nil {
			return err
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "holdout.restarted", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: holdoutAudit(open), After: holdoutAudit(next)}
		if reason != "" {
			ev.Metadata = map[string]any{"reason": reason}
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if err != nil {
		return SendingRulesView{}, err
	}
	return s.SendingRules(ctx, ws, p)
}

func (s *Service) openHoldout(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, seed int64, bp int32, campaigns, automations bool, reason string) (*store.HoldoutPeriod, error) {
	next := store.HoldoutPeriod{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Seed: seed, BasisPoints: bp, Campaigns: campaigns, Automations: automations,
		StartedAt: s.now(), StartedBy: userOf(p), Reason: reason}
	err := q.HoldoutInsertPeriod(ctx, store.HoldoutInsertPeriodParams{ID: next.ID, WorkspaceID: ws.ID, Seed: seed, BasisPoints: bp, Campaigns: campaigns,
		Automations: automations, StartedAt: next.StartedAt, StartedBy: next.StartedBy, Reason: reason})
	return &next, err
}
