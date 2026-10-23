package warmup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
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

// Evaluate brings one plan up to date. Running it twice a day is harmless.
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
		plan := ParsePlan(r.Schedule, r.Progress)
		today := state.WarmupDay(r.StartDate, now, loc)
		start := time.Date(r.StartDate.Year(), r.StartDate.Month(), r.StartDate.Day(), 0, 0, 0, 0, time.UTC)
		status, detail := r.Status, kept{English: r.StatusDetail}
		detailTexts := r.Texts
		setStatus := func(st string, d kept) {
			status, detail, detailTexts = st, d, textsJSON(map[string]i18n.Text{"status_detail": d.Text})
		}
		events := []store.WarmupAddEventParams{}
		addEvent := func(kind string, title, body kept) {
			events = append(events, store.WarmupAddEventParams{ID: newID(), PlanID: r.ID, OccurredAt: now, Kind: kind, Title: title.English, Body: body.English,
				Texts: textsJSON(map[string]i18n.Text{"title": title.Text, "body": body.Text})})
		}
		// Record every finished day that has no final result yet.
		for d := 1; d < today && d <= int(r.Days); d++ {
			if res, ok := plan.Result(d); ok && (res.Sent > 0 || res.Date != "" && !res.Held) {
				continue
			}
			date := start.AddDate(0, 0, d-1)
			stats, err := q.WarmupDailyStats(ctx, store.WarmupDailyStatsParams{ProviderID: r.ProviderID, Since: date, Until: date})
			if err != nil {
				return err
			}
			var st store.WarmupDailyStatsRow
			if len(stats) == 1 {
				st = stats[0]
			}
			res := Result{Day: d, Date: date.Format("2006-01-02"), Cap: plan.Cap(d), Sent: st.Sent}
			if prev, ok := plan.Result(d); ok && prev.Held {
				res.Held = true
			}
			if st.Sent > 0 {
				res.Bounce = round1(float64(st.Bounced) / float64(st.Sent) * 100)
			}
			if st.Delivered > 0 {
				res.Complaint = round2(float64(st.Complained) / float64(st.Delivered) * 100)
			}
			tripped := st.Sent > 0 && (res.Bounce > r.SlowdownBounce || res.Complaint > r.SlowdownComplaint)
			res.Slowed = tripped
			plan.set(res)
			switch {
			case tripped:
				// "Bounces hit 2.3% (guard 2%), mostly Yahoo. Day 7 cap held at 4,000."
				args := i18n.Args{"metric": "bounce", "rate": res.Bounce / 100, "guard": r.SlowdownBounce / 100, "guardDigits": digits(r.SlowdownBounce)}
				if res.Complaint > r.SlowdownComplaint {
					args = i18n.Args{"metric": "complaint", "rate": res.Complaint / 100, "guard": r.SlowdownComplaint / 100, "guardDigits": digits(r.SlowdownComplaint)}
				}
				mb := dominantMailbox(ctx, tx, r.ProviderID, date)
				args["mailbox"] = mailboxArg(mb, routing.MailboxName(mb))
				title := keep("log.slowdownTitle", nil)
				switch r.GuardAction {
				case "pause":
					setStatus("paused", keep("status.pausedByGuard", i18n.Args{"day": d}))
					addEvent("slowdown", title, keep("log.slowdown", with(args, "action", "pause")))
				case "step_back":
					capNext := plan.Cap(max(d-1, 1))
					plan.set(Result{Day: d + 1, Cap: capNext, Held: true})
					setStatus("slowed", keep("status.steppedBack", i18n.Args{"day": d}))
					addEvent("slowdown", title, keep("log.slowdown", with(args, "action", "step_back", "day", d+1, "cap", capNext)))
				default:
					capNext := res.Cap
					plan.set(Result{Day: d + 1, Cap: capNext, Held: true})
					setStatus("slowed", keep("status.capHeld", i18n.Args{"day": d + 1}))
					addEvent("slowdown", title, keep("log.slowdown", with(args, "action", "hold", "day", d+1, "cap", capNext)))
				}
			case res.Held && st.Sent > 0:
				setStatus("active", kept{})
				addEvent("resumed", kept{}, keep("log.rampResumed", i18n.Args{"bounce": res.Bounce / 100}))
			case st.Sent > 0:
				args := i18n.Args{"day": d, "sent": st.Sent, "bounce": res.Bounce / 100, "inside": "no", "campaign": ""}
				if name := campaignOn(ctx, tx, r.ProviderID, date); name != "" {
					args["inside"], args["campaign"] = "yes", name
				}
				addEvent("day_complete", kept{}, keep("log.dayComplete", args))
			}
		}
		// Raise the Broadcast share on schedule.
		if today >= 2 && today <= int(r.Days) && plan.Share(today) != plan.Share(today-1) && plan.Share(today) > 0 {
			if raised, err := s.raiseShare(ctx, tx, ws, r.ProviderID, plan.Share(today)); err != nil {
				return err
			} else if raised {
				addEvent("share_raised", kept{}, keep("log.shareRaised", i18n.Args{"share": float64(plan.Share(today)) / 100, "day": today}))
			}
		}
		if today > int(r.Days) {
			setStatus("done", keep("status.complete", nil))
			addEvent("done", keep("log.doneTitle", nil), keep("log.done", i18n.Args{"provider": r.ProviderName}))
			if _, err := tx.Exec(ctx, `UPDATE providers SET status = 'healthy', status_detail = '' WHERE id = $1 AND status = 'warming_up'`, r.ProviderID); err != nil {
				return err
			}
		} else if status != "done" {
			if _, err := tx.Exec(ctx, `UPDATE providers SET status_detail = $2 WHERE id = $1 AND status = 'warming_up'`, r.ProviderID, fmt.Sprintf("Warming up · day %d of %d", max(today, 1), r.Days)); err != nil {
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

// set records a day's result, replacing an earlier one.
func (p *Plan) set(r Result) {
	for i := range p.Progress {
		if p.Progress[i].Day == r.Day {
			p.Progress[i] = r
			return
		}
	}
	p.Progress = append(p.Progress, r)
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
