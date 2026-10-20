package referrals

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// TickArgs is the minute job: holds that ended, referrals never confirmed,
// rewards to deliver, links for new subscribers and the notifications.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "referrals.tick" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// TickWorker runs the minute job.
type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	S *Service
}

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error { return w.S.Tick(ctx) }

// Timeout implements river.Worker.
func (w *TickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return 5 * time.Minute }

// HourlyArgs is the hourly job: the leaderboards, and what privacy asks to
// forget (visit keys after 24 hours, referrers' addresses after 30 days,
// signup addresses reduced to their network after 90 days, shipping
// addresses 30 days after their reward was fulfilled).
type HourlyArgs struct{}

// Kind implements river.JobArgs.
func (HourlyArgs) Kind() string { return "referrals.hourly" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (HourlyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 2, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

// HourlyWorker runs the hourly job.
type HourlyWorker struct {
	river.WorkerDefaults[HourlyArgs]
	S *Service
}

// Work implements river.Worker.
func (w *HourlyWorker) Work(ctx context.Context, _ *river.Job[HourlyArgs]) error {
	return w.S.Hourly(ctx)
}

// RegisterJobs adds the referrals' workers and schedules.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &TickWorker{S: s})
	river.AddWorker(reg.Workers, &HourlyWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, false)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return HourlyArgs{}, nil }, true)
}

// Tick is one pass of the minute job.
func (s *Service) Tick(ctx context.Context) error {
	var errs []error
	if _, err := s.ProcessHolds(ctx); err != nil {
		errs = append(errs, err)
	}
	if _, err := s.ProcessNeverConfirmed(ctx); err != nil {
		errs = append(errs, err)
	}
	if _, err := s.DeliverDue(ctx); err != nil {
		errs = append(errs, err)
	}
	if _, err := s.EnsureLinks(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.notifyReview(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.alertCodes(ctx); err != nil {
		errs = append(errs, err)
	}
	if _, err := s.Pool.Exec(ctx, `UPDATE referral_rewards SET status = 'expired', updated_at = $1
		WHERE reward_type = 'coupon' AND status = 'sent' AND used_at IS NULL AND expires_at IS NOT NULL AND expires_at < $1`, s.now()); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Hourly is one pass of the hourly job.
func (s *Service) Hourly(ctx context.Context) error {
	now := s.now()
	var errs []error
	if err := s.RefreshLeaderboards(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.Q.ReferralVisitKeysPrune(ctx, now.Add(-24*time.Hour)); err != nil {
		errs = append(errs, err)
	}
	if err := s.Q.ReferralReferrerIPsPrune(ctx, now.Add(-SameIPWindow)); err != nil {
		errs = append(errs, err)
	}
	if err := s.Q.ReferralsReduceIPs(ctx, store.ReferralsReduceIPsParams{Now: now, Before: now.Add(-IPKeptFor)}); err != nil {
		errs = append(errs, err)
	}
	if err := s.Q.ReferralRewardsAddressPurge(ctx, now); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// notifyReview tells members with grow:review about held referrals: when a
// queue goes from empty to waiting, then at most once a day while it isn't
// empty.
func (s *Service) notifyReview(ctx context.Context) error {
	progs, err := s.Q.ReferralProgramsActive(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	for _, p := range progs {
		counts, err := s.queueCounts(ctx, p.ID)
		if err != nil {
			return err
		}
		if counts.held == 0 {
			if p.ReviewNotifiedAt != nil {
				if err := s.Q.ReferralProgramReviewNotified(ctx, store.ReferralProgramReviewNotifiedParams{ID: p.ID}); err != nil {
					return err
				}
			}
			continue
		}
		if p.ReviewNotifiedAt != nil && now.Sub(*p.ReviewNotifiedAt) < 24*time.Hour {
			continue
		}
		ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
		if err != nil {
			return err
		}
		names, err := s.heldNames(ctx, p.ID)
		if err != nil {
			return err
		}
		l := i18n.For(i18n.English)
		n := notify.Notification{Kind: notify.KindReferralsHeld, Severity: "warn",
			Title: msgs.M("notify.heldTitle").Text(l, i18n.Args{"count": counts.held}),
			Body:  msgs.M("notify.heldBody").Text(l, i18n.Args{"names": names}),
			Link:  "/w/" + ws.Slug + "/grow/referrals/review",
			Data:  map[string]any{"program": p.ID.String(), "held": counts.held}}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.GrowReview, n); err != nil {
				return err
			}
			at := now
			return store.New(tx).ReferralProgramReviewNotified(ctx, store.ReferralProgramReviewNotifiedParams{ID: p.ID, At: &at})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// heldNames is "Alex Moreau (6), Mateo Silva (3)".
func (s *Service) heldNames(ctx context.Context, programID uuid.UUID) (string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT c.first_name, c.last_name, c.email, count(*) FROM referrals r
		JOIN referral_links rl ON rl.id = r.link_id JOIN contacts c ON c.id = rl.contact_id
		WHERE r.program_id = $1 AND r.status = 'review' GROUP BY 1, 2, 3 ORDER BY 4 DESC, 1 LIMIT 5`, programID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var first, last, email string
		var n int64
		if err := rows.Scan(&first, &last, &email, &n); err != nil {
			return "", err
		}
		parts = append(parts, displayName(first, last, email)+" ("+formatInt(n)+")")
	}
	return strings.Join(parts, ", "), rows.Err()
}

// alertCodes tells members with grow:edit when a code reward runs low or
// out, once each time.
func (s *Service) alertCodes(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT m.id FROM referral_milestones m JOIN referral_programs p ON p.id = m.program_id AND p.deleted_at IS NULL
		WHERE m.reward_type = 'codes' AND m.deleted_at IS NULL`)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	for _, id := range ids {
		m, err := s.Q.ReferralMilestoneByID(ctx, id)
		if err != nil {
			return err
		}
		left, err := s.Q.ReferralCodesLeft(ctx, id)
		if err != nil {
			return err
		}
		pace := s.codesPace(ctx, id)
		low := left < int64(m.CodesAlertBelow) || (pace > 0 && float64(left)/pace < 14)
		state := ""
		switch {
		case left == 0:
			state = "out"
		case low:
			state = "low"
		}
		if state == "" || state == m.CodesAlerted {
			continue
		}
		ws, err := s.Q.GetWorkspace(ctx, m.WorkspaceID)
		if err != nil {
			return err
		}
		l := i18n.For(i18n.English)
		n := notify.Notification{Kind: notify.KindRewardCodesLow, Severity: "warn", Link: "/w/" + ws.Slug + "/grow/referrals/milestones?edit=" + id.String(),
			Data: map[string]any{"milestone": id.String(), "left": left}}
		if state == "out" {
			var waiting int64
			_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM referral_rewards WHERE milestone_id = $1 AND status = 'waiting_codes'`, id).Scan(&waiting)
			n.Severity = "bad"
			n.Title = msgs.M("notify.codesOutTitle").Text(l, i18n.Args{"reward": m.RewardName})
			n.Body = msgs.M("notify.codesOutBody").Text(l, i18n.Args{"waiting": waiting})
		} else {
			n.Title = msgs.M("notify.codesLowTitle").Text(l, i18n.Args{"reward": m.RewardName, "left": left})
			n.Body = msgs.M("notify.codesLowBody").Text(l, nil)
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.GrowEdit, n); err != nil {
				return err
			}
			return store.New(tx).ReferralMilestoneCodesAlerted(ctx, store.ReferralMilestoneCodesAlertedParams{ID: id, CodesAlerted: state})
		})
		if err != nil {
			return err
		}
	}
	return nil
}
