package warmup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// DailyArgs evaluates every running plan: it records finished days, slows
// down when a guard trips, raises the Broadcast share on schedule and
// finishes plans.
type DailyArgs struct{}

// Kind implements river.JobArgs.
func (DailyArgs) Kind() string { return "warmup.daily" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (DailyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

type DailyWorker struct {
	river.WorkerDefaults[DailyArgs]
	S   *Service
	Log *slog.Logger
	// Skip leaves a provider alone (sample data).
	Skip func(store.Provider) bool
}

// Work implements river.Worker.
func (w *DailyWorker) Work(ctx context.Context, _ *river.Job[DailyArgs]) error {
	plans, err := w.S.Q.WarmupActivePlans(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, pl := range plans {
		if w.Skip != nil {
			if p, err := w.S.Q.ProvGetAny(ctx, pl.ProviderID); err == nil && w.Skip(p) {
				continue
			}
		}
		if err := w.S.Evaluate(ctx, pl.ID); err != nil {
			errs = append(errs, fmt.Errorf("plan %s: %w", pl.ID, err))
		}
	}
	return errors.Join(errs...)
}

var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

// Evaluate brings one plan up to date (PR-8): it judges every finished
// day of the workspace's calendar since the last run, moves the plan on by
// the days that counted (a day that didn't count repeats), holds the cap
// after a guard trips until two healthy days in a row without moving the
// plan meanwhile (the ramp resumes from the held cap, Hold.Moves), raises
// the Broadcast share on schedule and finishes the plan once every day
// counted. A paused plan isn't judged: its days wait. Running it twice a
// day is harmless.
func (s *Service) Evaluate(ctx context.Context, planID uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var wsID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT p.workspace_id FROM warmup_plans w JOIN providers p ON p.id = w.provider_id WHERE w.id = $1 FOR UPDATE OF w`, planID).Scan(&wsID); err != nil {
			return err
		}
		ws, err := q.GetWorkspace(ctx, wsID)
		if err != nil {
			return err
		}
		r, err := q.WarmupGetPlan(ctx, store.WarmupGetPlanParams{ID: planID, WorkspaceID: wsID})
		if err != nil {
			return err
		}
		if r.Status == "done" || r.Status == "paused" {
			return nil
		}
		loc := state.Location(ws.Timezone)
		now := s.now()
		today := dateIn(now, loc)
		plan := ParsePlan(r.Schedule, r.Progress)
		start := localDate(r.StartDate, loc)
		from := localDate(r.StartedOn, loc)
		evaluated := r.EvaluatedThrough
		if evaluated != nil {
			from = localDate(*evaluated, loc).AddDate(0, 0, 1)
		}
		hold := Hold{Cap: r.HeldCap, Streak: int(r.HealthyStreak)}
		guards := Guards{Bounce: r.SlowdownBounce, Complaint: r.SlowdownComplaint, MinSample: int64(r.MinSample), Action: r.GuardAction}
		days := int(r.Days)
		status, detail := r.Status, kept{English: r.StatusDetail}
		detailTexts := r.Texts
		setStatus := func(st string, d kept) {
			status, detail, detailTexts = st, d, textsJSON(map[string]i18n.Text{"status_detail": d.Text})
		}
		events := []store.WarmupAddEventParams{}
		addEvent := func(at time.Time, kind string, title, body kept) {
			events = append(events, store.WarmupAddEventParams{ID: newID(), PlanID: r.ID, OccurredAt: at, Kind: kind, Title: title.English, Body: body.English,
				Texts: textsJSON(map[string]i18n.Text{"title": title.Text, "body": body.Text})})
		}
		for d := from; d.Before(today); d = d.AddDate(0, 0, 1) {
			day := dayNumber(start, d)
			if day > days {
				break
			}
			planned := plan.PlannedCap(day)
			dayCap := hold.CapFor(planned)
			st, err := dayStats(ctx, q, r.ProviderID, d, d.AddDate(0, 0, 1))
			if err != nil {
				return err
			}
			v := Judge(st, dayCap, guards)
			res := Result{Day: day, Date: d.Format("2006-01-02"), Cap: dayCap, Sent: st.Sent, Bounced: st.Bounced, Complained: st.Complained, Delivered: st.Delivered,
				Bounce: v.Bounce, Complaint: v.Complaint,
				Counted: v.Counted, Judged: v.Judged, Short: v.Short, Slowed: v.Tripped, Metric: v.Metric, Held: hold.Cap != nil}
			if hold.Cap != nil && planned != dayCap {
				res.PlannedCap = planned
			}
			prevCap := plan.PlannedCap(max(day-1, 1))
			next, step := hold.After(v, dayCap, prevCap, r.GuardAction)
			if step == HoldHealthy || step == HoldEnded {
				res.Healthy = hold.Streak + 1
			}
			// A healthy day under a hold proves the held cap but doesn't
			// move the plan on, so the cap never jumps when the hold ends.
			moves := hold.Moves(v, step, planned)
			res.Counted = moves
			plan.set(res)
			// The log entry is dated on the day it is about.
			at := d.AddDate(0, 0, 1).Add(-time.Second)
			bounce := 0.0
			if v.Bounce != nil {
				bounce = *v.Bounce / 100
			}
			switch step {
			case HoldStarted, HoldPause:
				args := i18n.Args{"metric": "bounce", "rate": bounce, "guard": r.SlowdownBounce / 100, "guardDigits": digits(r.SlowdownBounce)}
				if v.Metric == "complaint" && v.Complaint != nil {
					args = i18n.Args{"metric": "complaint", "rate": *v.Complaint / 100, "guard": r.SlowdownComplaint / 100, "guardDigits": digits(r.SlowdownComplaint)}
				}
				mb := dominantMailbox(ctx, tx, r.ProviderID, d)
				args["mailbox"] = mailboxArg(mb, routing.MailboxName(mb))
				action := r.GuardAction
				if step == HoldPause {
					action = "pause"
					setStatus("paused", keep("status.pausedByGuard", i18n.Args{"day": day}))
				} else {
					setStatus("slowed", keep("status.held", i18n.Args{"cap": *next.Cap, "healthy": HealthyDays}))
				}
				addEvent(at, "slowdown", keep("log.slowdownTitle", nil), keep("log.slowdownCounted", with(args, "action", action, "day", day, "cap", *next.Cap, "healthy", HealthyDays)))
			case HoldHealthy:
				// The table says "healthy day 1 of 2"; the log keeps what changed.
			case HoldEnded:
				setStatus("active", kept{})
				// The ramp resumes from the held cap: the next day, or the
				// day the guard tripped after a step back.
				nextDay := day
				if moves {
					nextDay = min(day+1, days)
				}
				addEvent(at, "resumed", kept{}, keep("log.holdEnded", i18n.Args{"bounce": bounce, "day": nextDay, "cap": plan.PlannedCap(nextDay)}))
			default:
				// The log keeps what changed: a day a campaign used, a day
				// that didn't count. Quiet counted days are in the table.
				switch {
				case v.Counted:
					if name := campaignOn(ctx, tx, r.ProviderID, d); name != "" {
						addEvent(at, "day_complete", kept{}, keep("log.dayCounted", i18n.Args{"day": day, "sent": st.Sent, "bounce": bounce, "inside": "yes", "campaign": name}))
					}
				default:
					addEvent(at, "day_complete", kept{}, keep("log.dayShort", i18n.Args{"day": day, "sent": st.Sent, "cap": dayCap, "weekday": weekdayKey(d.AddDate(0, 0, 1))}))
				}
			}
			hold = next
			if !moves {
				// The day repeats: the plan's day 1 moves a day later.
				start = start.AddDate(0, 0, 1)
			}
			dd := d
			evaluated = &dd
			if step == HoldPause {
				break
			}
		}
		todayDay := dayNumber(start, today)
		// Raise the Broadcast share on schedule.
		if status != "paused" && todayDay >= 2 && todayDay <= days && plan.Share(todayDay) != plan.Share(todayDay-1) && plan.Share(todayDay) > 0 {
			if raised, err := s.raiseShare(ctx, tx, ws, r.ProviderID, plan.Share(todayDay)); err != nil {
				return err
			} else if raised {
				addEvent(now, "share_raised", kept{}, keep("log.shareRaised", i18n.Args{"share": float64(plan.Share(todayDay)) / 100, "day": todayDay}))
			}
		}
		if todayDay > days && status != "paused" {
			setStatus("done", keep("status.complete", nil))
			addEvent(now, "done", keep("log.doneTitle", nil), keep("log.done", i18n.Args{"provider": r.ProviderName}))
			if _, err := tx.Exec(ctx, `UPDATE providers SET status = 'healthy', status_detail = '' WHERE id = $1 AND status = 'warming_up'`, r.ProviderID); err != nil {
				return err
			}
		} else if status != "done" {
			if _, err := tx.Exec(ctx, `UPDATE providers SET status_detail = $2 WHERE id = $1 AND status = 'warming_up'`, r.ProviderID, fmt.Sprintf("Warming up · day %d of %d", min(max(todayDay, 1), days), r.Days)); err != nil {
				return err
			}
		}
		var heldCap *int64
		if hold.Cap != nil {
			c := *hold.Cap
			heldCap = &c
		}
		var through *time.Time
		if evaluated != nil {
			t := dayUTC(*evaluated)
			through = &t
		}
		if err := q.WarmupMove(ctx, store.WarmupMoveParams{ID: r.ID, StartDate: dayUTC(start), EvaluatedThrough: through, HeldCap: heldCap, HealthyStreak: int32(hold.Streak)}); err != nil {
			return err
		}
		if status == "paused" && r.Status != "paused" {
			at := now
			if err := q.WarmupSetStatus(ctx, store.WarmupSetStatusParams{ID: r.ID, Status: status, StatusDetail: detail.English, PausedAt: &at, Texts: detailTexts}); err != nil {
				return err
			}
		}
		pb, _ := json.Marshal(plan.Progress)
		if err := q.WarmupSetProgress(ctx, store.WarmupSetProgressParams{ID: r.ID, Progress: pb, Status: status, StatusDetail: detail.English, Texts: detailTexts}); err != nil {
			return err
		}
		for _, e := range events {
			if err := q.WarmupAddEvent(ctx, e); err != nil {
				return err
			}
		}
		if status != r.Status {
			ws := ws.ID
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws, Actor: systemActor, Action: "warmup.status", TargetType: "warmup_plan", TargetID: r.ID.String(), TargetName: r.ProviderName,
				Before: map[string]string{"status": r.Status}, After: map[string]string{"status": status, "detail": detail.English}})
		}
		return nil
	})
}

// weekdayKey is a date's weekday for the messages' select ("monday").
func weekdayKey(d time.Time) string { return strings.ToLower(d.Weekday().String()) }

// dayStats is what a provider sent on one of the workspace's days. Only
// mailbox providers that report complaints count toward the complaint rate.
func dayStats(ctx context.Context, q *store.Queries, providerID uuid.UUID, from, until time.Time) (DayStats, error) {
	rows, err := q.WarmupDayStats(ctx, store.WarmupDayStatsParams{ProviderID: &providerID, Since: from, Until: until})
	if err != nil {
		return DayStats{}, err
	}
	var st DayStats
	for _, r := range rows {
		st.Sent += r.Sent
		st.Bounced += r.Bounced
		if mailbox.ReportsComplaints(r.MailboxProvider) {
			st.Complained += r.Complained
			st.Delivered += r.Delivered
		}
	}
	return st, nil
}

// raiseShare sets the provider's weight in the Broadcasts default split,
// taking the difference from the largest other share.
func (s *Service) raiseShare(ctx context.Context, tx pgx.Tx, ws store.Workspace, providerID uuid.UUID, share int) (bool, error) {
	q := store.New(tx)
	set, err := routing.Load(ctx, q, ws.ID)
	if err != nil {
		return false, err
	}
	def, ok := set.Default(providers.StreamBroadcast)
	if !ok || len(def.Splits) < 2 {
		return false, nil
	}
	idx, other := -1, -1
	for i, sp := range def.Splits {
		if sp.ProviderID == providerID {
			idx = i
		} else if other < 0 || sp.Weight > def.Splits[other].Weight {
			other = i
		}
	}
	if idx < 0 || other < 0 || def.Splits[idx].Weight >= share {
		return false, nil
	}
	before := append([]routing.Split(nil), def.Splits...)
	delta := share - def.Splits[idx].Weight
	if def.Splits[other].Weight-delta < 0 {
		return false, nil
	}
	def.Splits[idx].Weight = share
	def.Splits[other].Weight -= delta
	b, _ := json.Marshal(def.Splits)
	if _, err := tx.Exec(ctx, `UPDATE routing_rules SET splits = $2, updated_at = $3 WHERE id = $1`, def.ID, b, s.now()); err != nil {
		return false, err
	}
	wsID := ws.ID
	return true, s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Actor: systemActor, Action: "routing.update", TargetType: "routing", TargetID: ws.ID.String(), TargetName: ws.Name,
		Before: before, After: def.Splits, Metadata: map[string]any{"reason": "warm-up schedule", "stream": "broadcast"}})
}

// dominantMailbox is the mailbox provider with most bounces on a day (its key).
func dominantMailbox(ctx context.Context, tx pgx.Tx, providerID uuid.UUID, day time.Time) string {
	var mb string
	var n int64
	err := tx.QueryRow(ctx, `SELECT mailbox_provider, count(*) FROM delivery_events
		WHERE provider_id = $1 AND type IN ('hard_bounce', 'soft_bounce') AND occurred_at >= $2 AND occurred_at < $2 + interval '1 day'
		GROUP BY 1 ORDER BY 2 DESC LIMIT 1`, providerID, day).Scan(&mb, &n)
	if err != nil || n == 0 {
		return ""
	}
	return mb
}

// campaignOn names the campaign that used the provider on a day, when one did.
func campaignOn(ctx context.Context, tx pgx.Tx, providerID uuid.UUID, day time.Time) string {
	rows, err := tx.Query(ctx, `SELECT DISTINCT c.name FROM campaign_routes cr JOIN campaigns c ON c.id = cr.campaign_id
		WHERE cr.provider_id = $1 AND c.started_at >= $2 AND c.started_at < $2 + interval '1 day' LIMIT 2`, providerID, day)
	if err != nil {
		return ""
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			names = append(names, n)
		}
	}
	if len(names) == 1 {
		return names[0]
	}
	return ""
}
