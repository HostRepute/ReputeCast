package warmup

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// A warm-up cap reached mid-send holds the warming provider's share with
// choices (review PR-8, GS-9's warm-up hold): it waits for tomorrow's cap
// (and goes on by itself then), moves to a provider that can take it, or
// the held rest is cancelled. The sending engine raises the hold
// (engine/warmup_send.go).

// capHoldReason is the route detail's reason (engine.CapHoldReason).
const capHoldReason = "warmup_cap"

// capDetail is a held route's status detail.
type capDetail struct {
	Reason string     `json:"reason,omitempty"`
	Until  *time.Time `json:"until,omitempty"`
	// Paused: the plan was paused; the share goes on once it resumes.
	Paused   bool   `json:"paused,omitempty"`
	Decision string `json:"decision,omitempty"`
	To       string `json:"to,omitempty"`
	Resumed  bool   `json:"resumed,omitempty"`
}

// WarmupCapHold is a campaign's share waiting at the warm-up cap.
type WarmupCapHold struct {
	CampaignID string `json:"campaignId"`
	Campaign   string `json:"campaign"`
	// Held is how many recipients wait; ResumeAt when tomorrow's cap
	// starts (the share goes on by itself then); Paused when the plan's
	// pause holds it (it goes on when the plan resumes).
	Held     int64      `json:"held"`
	ResumeAt *time.Time `json:"resumeAt,omitempty"`
	Paused   bool       `json:"paused"`
	// Decision is "wait" once someone chose to wait.
	Decision     string                  `json:"decision,omitempty"`
	Alternatives []WarmupHoldAlternative `json:"alternatives"`
}

// WarmupHoldAlternative is a provider the held share could move to, with
// the checks it was vetted with (state.Vet).
type WarmupHoldAlternative struct {
	ProviderID string           `json:"providerId"`
	Name       string           `json:"name"`
	Short      string           `json:"short"`
	Allowed    bool             `json:"allowed"`
	Checks     []state.VetCheck `json:"checks"`
}

// capHolds lists the shares held at a provider's warm-up cap.
func (s *Service) capHolds(ctx context.Context, ws *store.Workspace, providerID uuid.UUID) ([]WarmupCapHold, error) {
	rows, err := s.Q.WarmupCapHolds(ctx, providerID)
	if err != nil {
		return nil, err
	}
	out := []WarmupCapHold{}
	if len(rows) == 0 {
		return out, nil
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	l := i18n.LangFrom(ctx)
	loc := state.Location(ws.Timezone)
	for _, r := range rows {
		var d capDetail
		_ = json.Unmarshal([]byte(r.StatusDetail), &d)
		h := WarmupCapHold{CampaignID: r.CampaignID.String(), Campaign: r.CampaignName, Held: r.Held, ResumeAt: d.Until, Paused: d.Paused, Decision: d.Decision,
			Alternatives: []WarmupHoldAlternative{}}
		sender := s.senderOf(ctx, ws.ID, r.SenderID)
		for _, p := range provs {
			if p.ID == providerID {
				continue
			}
			v := state.Vet(p, state.Load{Broadcast: true, Messages: r.Held, Sender: sender, Warmup: Today(ctx, s.Q, p.ID, s.now(), loc)})
			// Providers that can't take mail at all aren't alternatives.
			if failed, ok := v.Failed(); ok && (failed.Key == state.VetAvailable || failed.Key == state.VetBulkTerms) {
				continue
			}
			short := state.Short(p.Name, p.Adapter)
			h.Alternatives = append(h.Alternatives, WarmupHoldAlternative{ProviderID: p.ID.String(), Name: p.Name, Short: short, Allowed: v.Allowed(), Checks: v.In(l, short).Checks})
		}
		out = append(out, h)
	}
	return out, nil
}

func (s *Service) senderOf(ctx context.Context, wsID uuid.UUID, id *uuid.UUID) map[string]string {
	if id == nil {
		return nil
	}
	snd, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: wsID, ID: *id})
	if err != nil {
		return nil
	}
	return state.SenderVerification(snd.Verification)
}

// WarmupHoldDecision is what to do with a share held at the warm-up cap.
type WarmupHoldDecision struct {
	Action string `json:"action" enum:"wait,move,cancel"`
	// TargetProviderID is where "move" sends the held share.
	TargetProviderID string `json:"targetProviderId,omitempty" format:"uuid"`
}

// WarmupHoldDecisionResult says what happened.
type WarmupHoldDecisionResult struct {
	Action string `json:"action"`
	// Moved is how many recipients moved, Cancelled how many won't get it.
	Moved     int64 `json:"moved"`
	Cancelled int64 `json:"cancelled"`
}

// DecideCapHold waits, moves or cancels a share held at a plan's cap.
// Waiting and moving resume sending (people who can send); cancelling the
// held rest stops it (people who can pause sends). A move is vetted first.
func (s *Service) DecideCapHold(ctx context.Context, ws *store.Workspace, planID, campaignID string, in WarmupHoldDecision) (WarmupHoldDecisionResult, error) {
	r, err := s.plan(ctx, ws, planID)
	if err != nil {
		return WarmupHoldDecisionResult{}, err
	}
	cid, err := uuid.Parse(campaignID)
	if err != nil {
		return WarmupHoldDecisionResult{}, api.NotFound("Campaign")
	}
	pr := api.PrincipalFrom(ctx)
	need := authz.CampaignsSend
	if in.Action == "cancel" {
		need = authz.CampaignsPause
	}
	if pr == nil || !pr.Can(need) {
		return WarmupHoldDecisionResult{}, api.Forbidden(need, nil)
	}
	holds, err := s.Q.WarmupCapHolds(ctx, r.ProviderID)
	if err != nil {
		return WarmupHoldDecisionResult{}, err
	}
	var hold *store.WarmupCapHoldsRow
	for i := range holds {
		if holds[i].CampaignID == cid {
			hold = &holds[i]
		}
	}
	if hold == nil {
		return WarmupHoldDecisionResult{}, api.Conflict(say(ctx, "problem.noHold", i18n.Args{"provider": r.ProviderName}))
	}
	var d capDetail
	_ = json.Unmarshal([]byte(hold.StatusDetail), &d)
	now := s.now()
	var by *uuid.UUID
	if pr.UserID != uuid.Nil {
		u := pr.UserID
		by = &u
	}
	out := WarmupHoldDecisionResult{Action: in.Action}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		meta := map[string]any{"campaign_id": cid.String(), "campaign": hold.CampaignName, "held": hold.Held}
		switch in.Action {
		case "wait":
			d.Decision = "wait"
			b, _ := json.Marshal(d)
			if _, err := q.ProvSetRouteStatus(ctx, store.ProvSetRouteStatusParams{CampaignID: cid, ProviderID: r.ProviderID, Status: "held", StatusDetail: string(b)}); err != nil {
				return err
			}
		case "move":
			tid, err := uuid.Parse(in.TargetProviderID)
			if err != nil {
				return api.Invalid("body.targetProviderId", say(ctx, "problem.pickTarget", nil))
			}
			target, err := q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: tid})
			if err != nil || target.ID == r.ProviderID {
				return api.Invalid("body.targetProviderId", say(ctx, "problem.pickTarget", nil))
			}
			v := state.Vet(target, state.Load{Broadcast: true, Messages: hold.Held, Sender: s.senderOf(ctx, ws.ID, hold.SenderID),
				Warmup: Today(ctx, q, target.ID, now, nil)})
			if failed, ok := v.Failed(); ok {
				return api.Conflict(failed.In(i18n.LangFrom(ctx), target.Name))
			}
			tag, err := tx.Exec(ctx, `UPDATE messages SET provider_id = $3, claimed_at = NULL WHERE campaign_id = $1 AND provider_id = $2 AND status = 'queued'`, cid, r.ProviderID, target.ID)
			if err != nil {
				return err
			}
			out.Moved = tag.RowsAffected()
			if _, err := tx.Exec(ctx, `INSERT INTO campaign_routes (campaign_id, provider_id, planned, status) VALUES ($1, $2, $3, 'pending')
				ON CONFLICT (campaign_id, provider_id) DO UPDATE SET planned = campaign_routes.planned + EXCLUDED.planned,
				status = CASE WHEN campaign_routes.status IN ('done', 'held') THEN 'pending' ELSE campaign_routes.status END`, cid, target.ID, out.Moved); err != nil {
				return err
			}
			d.Decision, d.To = "move", target.Name
			b, _ := json.Marshal(d)
			if _, err := tx.Exec(ctx, `UPDATE campaign_routes SET planned = greatest(planned - $3, 0), status = 'done', status_detail = $4 WHERE campaign_id = $1 AND provider_id = $2`,
				cid, r.ProviderID, out.Moved, string(b)); err != nil {
				return err
			}
			if err := q.WarmupResolveCapHold(ctx, store.WarmupResolveCapHoldParams{CampaignID: cid, ProviderID: &r.ProviderID, At: &now, Resolution: "moved", ResolvedBy: by}); err != nil {
				return err
			}
			meta["target"], meta["moved"] = target.Name, out.Moved
		case "cancel":
			tag, err := tx.Exec(ctx, `UPDATE messages SET status = 'cancelled', error = 'The warm-up cap held this share and someone cancelled the rest.'
				WHERE campaign_id = $1 AND provider_id = $2 AND status = 'queued'`, cid, r.ProviderID)
			if err != nil {
				return err
			}
			out.Cancelled = tag.RowsAffected()
			d.Decision = "cancel"
			b, _ := json.Marshal(d)
			if _, err := q.ProvSetRouteStatus(ctx, store.ProvSetRouteStatusParams{CampaignID: cid, ProviderID: r.ProviderID, Status: "cancelled", StatusDetail: string(b)}); err != nil {
				return err
			}
			if err := q.WarmupResolveCapHold(ctx, store.WarmupResolveCapHoldParams{CampaignID: cid, ProviderID: &r.ProviderID, At: &now, Resolution: "cancelled", ResolvedBy: by}); err != nil {
				return err
			}
			meta["cancelled"] = out.Cancelled
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "campaign.warmup_hold_" + in.Action, TargetType: "campaign", TargetID: cid.String(),
			TargetName: hold.CampaignName, Metadata: meta})
	})
	return out, err
}

// ResumeArgs sends on the shares a warm-up cap held, every five minutes:
// a share waiting for tomorrow's cap goes on soon after midnight in the
// workspace's time zone.
type ResumeArgs struct{}

// Kind implements river.JobArgs.
func (ResumeArgs) Kind() string { return "warmup.resume_holds" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ResumeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
}

// ResumeWorker runs ResumeArgs.
type ResumeWorker struct {
	river.WorkerDefaults[ResumeArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ResumeWorker) Work(ctx context.Context, _ *river.Job[ResumeArgs]) error {
	return w.S.ResumeCapHolds(ctx)
}

// ResumeCapHolds sends on the shares a warm-up cap held once their
// provider's new day has started, and the shares a paused plan held once
// it runs again (unless someone moved or cancelled them). The send loop
// picks a resumed share up within half a minute.
func (s *Service) ResumeCapHolds(ctx context.Context) error {
	return s.resumeCapHolds(ctx, uuid.Nil)
}

// resumeCapHolds is ResumeCapHolds for one provider (uuid.Nil: all).
func (s *Service) resumeCapHolds(ctx context.Context, only uuid.UUID) error {
	rows, err := s.Q.WarmupCapHoldsDue(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	for _, r := range rows {
		if only != uuid.Nil && r.ProviderID != only {
			continue
		}
		var d capDetail
		_ = json.Unmarshal([]byte(r.StatusDetail), &d)
		if d.Reason != capHoldReason || d.Decision == "move" || d.Decision == "cancel" {
			continue
		}
		if d.Paused {
			if a, ok, err := AllowanceFor(ctx, s.Q, r.ProviderID, now, nil); err != nil || (ok && a.Paused) {
				continue
			}
		} else if d.Until == nil || now.Before(*d.Until) {
			continue
		}
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			d.Resumed = true
			b, _ := json.Marshal(d)
			if _, err := q.ProvSetRouteStatus(ctx, store.ProvSetRouteStatusParams{CampaignID: r.CampaignID, ProviderID: r.ProviderID, Status: "sending", StatusDetail: string(b)}); err != nil {
				return err
			}
			pid := r.ProviderID
			return q.WarmupResolveCapHold(ctx, store.WarmupResolveCapHoldParams{CampaignID: r.CampaignID, ProviderID: &pid, At: &now, Resolution: "auto_recovered"})
		})
		if err != nil {
			return err
		}
	}
	return nil
}
