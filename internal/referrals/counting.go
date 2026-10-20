package referrals

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// countReferral makes a referral final: it counts toward its referrer's
// milestones from now on, even if its subscriber leaves later.
func (s *Service) countReferral(ctx context.Context, tx pgx.Tx, ws *store.Workspace, id uuid.UUID, at time.Time) error {
	q := store.New(tx)
	if _, err := tx.Exec(ctx, `UPDATE referrals SET status = 'counted', counted_at = $2, updated_at = $2 WHERE id = $1 AND status IN ('hold', 'review')`, id, at); err != nil {
		return err
	}
	r, err := q.ReferralGet(ctx, store.ReferralGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return err
	}
	if r.LinkID == nil {
		return nil
	}
	counted, err := q.ReferralCountedOf(ctx, store.ReferralCountedOfParams{ProgramID: r.ProgramID, LinkID: r.LinkID})
	if err != nil {
		return err
	}
	if err := writeLog(ctx, q, ws.ID, logEntry{program: r.ProgramID, referral: &r.ID, link: r.LinkID, contact: r.ReferrerID, kind: EventCounted, at: at,
		detail: map[string]any{"counted": counted, "referred": r.ContactID}}); err != nil {
		return err
	}
	referrer := hookContact{}
	if r.ReferrerID != nil {
		referrer.ContactID = r.ReferrerID.String()
		var email, first string
		if err := tx.QueryRow(ctx, `SELECT email, first_name FROM contacts WHERE id = $1`, *r.ReferrerID).Scan(&email, &first); err == nil {
			referrer.Email, referrer.FirstName = email, first
		}
	}
	if err := emitHook(ctx, tx, ws, HookCounted, map[string]any{"program": r.ProgramID.String(), "referral": referralHook(r), "referrer": referrer,
		"counted": counted}); err != nil {
		return err
	}
	if r.ReferrerID == nil {
		return nil // the referrer was erased: their rewards went with them
	}
	return s.reachMilestones(ctx, tx, ws, r.ProgramID, *r.LinkID, *r.ReferrerID, counted, at)
}

// reachMilestones gives a referrer every milestone they have reached and
// not received yet, in order: one reward per milestone per referrer.
func (s *Service) reachMilestones(ctx context.Context, tx pgx.Tx, ws *store.Workspace, programID, linkID, contactID uuid.UUID, counted int64, at time.Time) error {
	q := store.New(tx)
	ms, err := q.ReferralMilestones(ctx, programID)
	if err != nil {
		return err
	}
	for _, m := range ms {
		if int64(m.Count) > counted {
			break
		}
		if err := s.issueReward(ctx, tx, ws, m, linkID, contactID, counted, at); err != nil {
			return err
		}
	}
	return nil
}

// ProcessHolds counts the referrals whose hold has ended: still subscribed,
// not hard-bounced or complained (unless "Left during the hold" is off).
func (s *Service) ProcessHolds(ctx context.Context) (int, error) {
	now := s.now()
	total := 0
	for {
		n := 0
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			rows, err := q.ReferralsHoldDue(ctx, store.ReferralsHoldDueParams{Now: now, Lim: 200})
			if err != nil {
				return err
			}
			n = len(rows)
			wsCache := map[uuid.UUID]*store.Workspace{}
			for _, r := range rows {
				ws, err := workspaceOf(ctx, q, wsCache, r.WorkspaceID)
				if err != nil {
					return err
				}
				left := r.Membership != "subscribed" || r.Bounced
				if left && r.RuleLeft {
					kind := "unsubscribed"
					if r.Bounced {
						kind = r.BounceReason
					}
					if err := s.notCount(ctx, tx, ws, r.ID, ReasonLeft, kind, now); err != nil {
						return err
					}
					continue
				}
				if err := s.countReferral(ctx, tx, ws, r.ID, now); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return total, err
		}
		total += n
		if n < 200 {
			return total, nil
		}
	}
}

func workspaceOf(ctx context.Context, q *store.Queries, cache map[uuid.UUID]*store.Workspace, id uuid.UUID) (*store.Workspace, error) {
	if ws, ok := cache[id]; ok {
		return ws, nil
	}
	ws, err := q.GetWorkspace(ctx, id)
	if err != nil {
		return nil, err
	}
	cache[id] = &ws
	return &ws, nil
}

// notCount excludes a referral with its reason.
func (s *Service) notCount(ctx context.Context, tx pgx.Tx, ws *store.Workspace, id uuid.UUID, reason, leftKind string, at time.Time) error {
	q := store.New(tx)
	if err := q.ReferralSetNotCounted(ctx, store.ReferralSetNotCountedParams{ID: id, Reason: reason, LeftKind: leftKind, Now: at}); err != nil {
		return err
	}
	r, err := q.ReferralGet(ctx, store.ReferralGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return err
	}
	if err := writeLog(ctx, q, ws.ID, logEntry{program: r.ProgramID, referral: &r.ID, link: r.LinkID, contact: r.ContactID, kind: EventNotCounted, at: at,
		detail: map[string]any{"reason": reason, "left": leftKind}}); err != nil {
		return err
	}
	referrer := hookContact{}
	if r.ReferrerID != nil {
		referrer.ContactID = r.ReferrerID.String()
	}
	return emitHook(ctx, tx, ws, HookNotCounted, map[string]any{"program": r.ProgramID.String(), "referral": referralHook(r), "referrer": referrer})
}

// ProcessNeverConfirmed excludes referrals with no confirmation within 7
// days. The usual confirmation reminder still went out after 24 hours.
func (s *Service) ProcessNeverConfirmed(ctx context.Context) (int, error) {
	now := s.now()
	n := 0
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rows, err := q.ReferralsNeverConfirmed(ctx, store.ReferralsNeverConfirmedParams{Now: now, Before: now.Add(-NeverConfirmedAfter)})
		if err != nil {
			return err
		}
		n = len(rows)
		wsCache := map[uuid.UUID]*store.Workspace{}
		for _, r := range rows {
			ws, err := workspaceOf(ctx, q, wsCache, r.WorkspaceID)
			if err != nil {
				return err
			}
			id := r.ID
			if err := writeLog(ctx, q, ws.ID, logEntry{program: r.ProgramID, referral: &id, link: r.LinkID, contact: r.ContactID, kind: EventNotCounted, at: now,
				detail: map[string]any{"reason": ReasonNeverConfirmed}}); err != nil {
				return err
			}
			ref, err := q.ReferralGet(ctx, store.ReferralGetParams{WorkspaceID: ws.ID, ID: id})
			if err != nil {
				return err
			}
			referrer := hookContact{}
			if r.ReferrerID != nil {
				referrer.ContactID = r.ReferrerID.String()
			}
			if err := emitHook(ctx, tx, ws, HookNotCounted, map[string]any{"program": r.ProgramID.String(), "referral": referralHook(ref), "referrer": referrer}); err != nil {
				return err
			}
		}
		return nil
	})
	return n, err
}

// EnsureLinks gives the confirmed subscribers of every program that is on a
// link (subscribers added by an import or the API, which don't confirm
// through a form), a few thousand at a time.
func (s *Service) EnsureLinks(ctx context.Context) (int64, error) {
	progs, err := s.Q.ReferralProgramsActive(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, p := range progs {
		if p.Status != "on" {
			continue
		}
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			n, err := s.ensureLinks(ctx, tx, p, 5000)
			total += n
			return err
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			return total, err
		}
	}
	return total, nil
}
