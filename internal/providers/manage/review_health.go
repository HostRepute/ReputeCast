package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// Review 2026-10, PR-7 (the health job's part): no mail moves to a failover
// target that can't take it, and a provider refusing messages is noticed.
// The engine's own holds on identity errors are the sending engine's (C6).

const (
	// RejectWindow is how far back the rejection watch looks.
	RejectWindow = 10 * time.Minute
	// RejectMin and RejectRate: at least this many messages refused for
	// good, and this share of what was handed over, make a provider
	// "rejecting" (an unverified sender, a suspended account).
	RejectMin  = 10
	RejectRate = 0.25
)

// streamSenders is the verification per provider of the senders a stream
// uses, merged: a provider is "verified" only when every one of them is
// verified there. nil when the stream has no stored sender.
func streamSenders(ctx context.Context, q *store.Queries, wsID uuid.UUID, stream providers.Stream) map[string]string {
	rows, err := q.ProvStreamSenderVerification(ctx, store.ProvStreamSenderVerificationParams{WorkspaceID: wsID, Stream: string(stream)})
	if err != nil || len(rows) == 0 {
		return nil
	}
	maps := make([]map[string]string, 0, len(rows))
	keys := map[string]bool{}
	for _, raw := range rows {
		m := state.SenderVerification(raw)
		maps = append(maps, m)
		for k := range m {
			keys[k] = true
		}
	}
	out := map[string]string{}
	for k := range keys {
		out[k] = "verified"
		for _, m := range maps {
			if v := m[k]; v != "verified" {
				out[k] = v
				break
			}
		}
	}
	return out
}

// failoverVetting vets a stream's fallback for what would move to it while
// p is down: the stream's last hour on p (at least one message), with the
// stream's senders and the fallback's warm-up today.
func (s *Service) failoverVetting(ctx context.Context, q *store.Queries, ws store.Workspace, p, fallback store.Provider, stream providers.Stream, now time.Time) state.Vetting {
	need, _ := q.ProvStreamRecentVolume(ctx, store.ProvStreamRecentVolumeParams{ProviderID: &p.ID, Stream: string(stream), Since: ptrTime(now.Add(-time.Hour))})
	load := state.Load{Broadcast: stream == providers.StreamBroadcast, Messages: max(need, 1), Sender: streamSenders(ctx, q, ws.ID, stream),
		Warmup: warmup.Today(ctx, q, fallback.ID, now, state.Location(ws.Timezone))}
	return state.Vet(fallback, load)
}

// vetFailover decides where a stream's mail goes while p is down: its
// fallback, when the policy moves mail on its own and the fallback passes
// every check (a warming fallback only under the automatic policy), or
// nowhere (held). The vetting is returned for the failover's record and
// the outage card.
func (s *Service) vetFailover(ctx context.Context, q *store.Queries, ws store.Workspace, p store.Provider, stream providers.Stream, policy routing.Policy,
	fallback *uuid.UUID, byID map[uuid.UUID]store.Provider, now time.Time) (*uuid.UUID, *state.Vetting) {
	if fallback == nil || *fallback == p.ID {
		return nil, nil
	}
	fp, ok := byID[*fallback]
	if !ok {
		return nil, nil
	}
	v := s.failoverVetting(ctx, q, ws, p, fp, stream, now)
	target, hold := routing.Failover(policy, fallback, func(uuid.UUID) routing.ProviderState {
		return routing.ProviderState{Available: v.Automatic(policy == routing.Automatic), WarmingUp: v.Warming()}
	})
	if hold {
		return nil, &v
	}
	return target, &v
}

// failoverDetail is what a stream_failovers row keeps about the vetting.
type failoverDetail struct {
	Reason string           `json:"reason,omitempty"`
	Checks []state.VetCheck `json:"checks,omitempty"`
	// Fallback is the fallback that was vetted (also when it failed).
	Fallback string `json:"fallback,omitempty"`
}

func failoverDetailJSON(fallback *uuid.UUID, v *state.Vetting) json.RawMessage {
	d := failoverDetail{}
	if v != nil {
		d.Checks = v.Checks
	}
	if fallback != nil {
		d.Fallback = fallback.String()
	}
	b, _ := json.Marshal(d)
	return b
}

// watchRejections updates the rejection watch of a provider from its last
// ten minutes: started when the provider began refusing, ended when it
// stopped.
func (s *Service) watchRejections(ctx context.Context, p store.Provider, h *state.Health, now time.Time) (started, ended bool, err error) {
	wins, err := s.Q.ProvCallWindowTotals(ctx, store.ProvCallWindowTotalsParams{ProviderIds: []uuid.UUID{p.ID}, Since: now.Add(-RejectWindow)})
	if err != nil {
		return false, false, err
	}
	var rejected, total int64
	if len(wins) == 1 {
		rejected, total = wins[0].Rejected, wins[0].Messages+wins[0].Rejected
	}
	rejecting := rejected >= RejectMin && total > 0 && float64(rejected)/float64(total) >= RejectRate
	switch {
	case rejecting:
		if h.Rejecting == nil {
			h.Rejecting = &state.RejectState{Since: now}
			started = true
		}
		h.Rejecting.Rejected, h.Rejecting.Total = rejected, total
		if h.LastRejection != nil {
			h.Rejecting.Code, h.Rejecting.Message = h.LastRejection.Code, h.LastRejection.Message
		}
	case h.Rejecting != nil:
		h.Rejecting = nil
		ended = true
	}
	return started, ended, nil
}

// onRejecting records a provider refusing messages and tells the team:
// the mail it refused wasn't sent.
func (s *Service) onRejecting(ctx context.Context, tx pgx.Tx, ws store.Workspace, p store.Provider, r *state.RejectState) error {
	q := store.New(tx)
	answer := r.Code
	if r.Message != "" {
		answer = r.Code + " " + r.Message
	}
	args := i18n.Args{"name": p.Name, "rejected": r.Rejected, "total": r.Total, "minutes": int(RejectWindow.Minutes()), "answer": answer, "hasAnswer": "no"}
	if answer != "" {
		args["hasAnswer"] = "yes"
	}
	title := fmt.Sprintf("%s refused %d of %d messages in %d min.", p.Name, r.Rejected, r.Total, int(RejectWindow.Minutes()))
	if answer != "" {
		title += " Last answer: " + answer
	}
	if err := s.incident(ctx, q, ws.ID, r.Since, "provider_rejecting", "bad", title, mIncidentRejecting.Keep(args), &p.ID, nil); err != nil {
		return err
	}
	ntitle, ntitleText := keep(mNotifyRejectingTitle, i18n.Args{"name": p.Name})
	body, bodyText := keep(mNotifyRejectingBody, args)
	if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.ProvidersManage, notify.Notification{
		Kind: notify.KindProviderDegraded, Severity: "bad", Title: ntitle, TitleText: ntitleText, Body: body, BodyText: bodyText, Link: "/w/" + ws.Slug + "/providers",
		Data: map[string]any{"providerId": p.ID.String(), "reason": "rejecting"},
	}); err != nil {
		return err
	}
	wsID := ws.ID
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Actor: systemActor, Action: "provider.rejecting", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
		Metadata: map[string]any{"rejected": r.Rejected, "total": r.Total, "code": r.Code}})
}

// onRejectingEnded closes the rejection incident.
func (s *Service) onRejectingEnded(ctx context.Context, tx pgx.Tx, ws store.Workspace, p store.Provider, now time.Time) error {
	q := store.New(tx)
	if err := q.ProvResolveIncidents(ctx, store.ProvResolveIncidentsParams{ProviderID: &p.ID, Kinds: []string{"provider_rejecting"}, ResolvedAt: &now}); err != nil {
		return err
	}
	if err := s.incident(ctx, q, ws.ID, now, "provider_accepting", "ok", p.Name+" accepts messages again.", mIncidentAccepting.Keep(i18n.Args{"name": p.Name}), &p.ID, nil); err != nil {
		return err
	}
	wsID := ws.ID
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Actor: systemActor, Action: "provider.accepting", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name})
}
