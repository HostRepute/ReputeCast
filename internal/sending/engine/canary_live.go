package engine

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// LiveCanary is the canary on the live monitor (LiveMonitorCanary; review
// GS-10): why it runs, how far it is, its gates and its seed placement.
type LiveCanary struct {
	Phase string `json:"phase" enum:"sending,watching,held,passed,continued,stopped"`
	// Chosen: a person asked for it in Review & send; otherwise Reasons
	// say why Traffic rules ran it (new_domain, never_mailed, over_usual).
	Chosen  bool     `json:"chosen"`
	Reasons []string `json:"reasons"`
	Percent float64  `json:"percent"`
	// Size is the canary's recipients; Sent how many went; Seeds the seed
	// inboxes that got it too.
	Size  int64 `json:"size"`
	Sent  int64 `json:"sent"`
	Seeds int   `json:"seeds"`
	// Queued are the recipients waiting for it to pass.
	Queued      int64  `json:"queued"`
	NeverMailed int64  `json:"neverMailed"`
	Usual       int64  `json:"usual"`
	Domain      string `json:"domain"`
	// StartedAt is when the canary started, SentAt when its last message
	// went, Until when the watch ends (the main send starts by itself then).
	StartedAt *time.Time `json:"startedAt,omitempty"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
	Until     *time.Time `json:"until,omitempty"`
	// WatchMinutes is the watch's length, Extended the minutes people added.
	WatchMinutes int          `json:"watchMinutes"`
	Extended     int          `json:"extended"`
	DecidedAt    *time.Time   `json:"decidedAt,omitempty"`
	DecidedBy    string       `json:"decidedBy,omitempty"`
	Gates        []CanaryGate `json:"gates"`
	// Seeds' placement, per mailbox provider (nil until a copy is found).
	Placement *SeedPlacement `json:"placement,omitempty"`
	// MainSeconds is how long the main send takes once it starts, at the
	// routes' and mailbox providers' rates (the monitor's "everyone has it
	// by about").
	MainSeconds int64 `json:"mainSeconds"`
}

// liveCanary is a send's canary for the snapshot (nil without one).
func liveCanary(cs canaryState, ts tallies, ty typed, sent int64, queue map[queueKey]queueCell, now time.Time) *LiveCanary {
	if cs.Phase == "" {
		return nil
	}
	out := &LiveCanary{Phase: cs.Phase, Chosen: cs.Choice == CanaryOn, Reasons: cs.Reasons, Percent: cs.Percent, Size: cs.Size, Sent: min(sent, cs.Size),
		Seeds: cs.Seeds, NeverMailed: cs.NeverMailed, Usual: cs.Usual, Domain: cs.Domain, StartedAt: cs.Started, SentAt: cs.SentAt, Until: cs.Until,
		WatchMinutes: cs.Watch, Extended: cs.Extended, DecidedAt: cs.Decided, DecidedBy: cs.DecidedBy, Gates: canaryGates(cs, ts, ty), Placement: cs.SeedResult}
	if out.Reasons == nil {
		out.Reasons = []string{}
	}
	for _, c := range queue {
		out.Queued += c.canary
	}
	if cs.Phase == canarySending && cs.Until == nil && cs.Started != nil {
		// Before the last canary message the watch's end isn't known: the
		// monitor shows when it would end if the canary went now.
		until := now.Add(time.Duration(cs.Watch+cs.Extended) * time.Minute)
		out.Until = &until
	}
	return out
}

// registerCanary adds the canary's decisions to the live monitor (review
// GS-10): people who can approve sends continue early; people who can
// send watch longer or stop it.
func registerCanary(h huma.API, e *Engine, base string, tags []string) {
	api.Post(h, api.Op{ID: "continue-live-canary", Path: base + "/canary/continue", Summary: "Send everyone now instead of when the canary's watch ends",
		Tags: tags, Access: api.Workspace(authz.CampaignsApprove)},
		func(ctx context.Context, in *liveInput) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			if err := e.ContinueCanary(ctx, id, by); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
	api.Post(h, api.Op{ID: "extend-live-canary", Path: base + "/canary/extend", Summary: "Watch the canary 10 more minutes", Tags: tags,
		Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *liveInput) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			if err := e.ExtendCanary(ctx, id, by); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
	api.Post(h, api.Op{ID: "stop-live-canary", Path: base + "/canary/stop", Summary: "Stop the send after its canary: the recipients it held don't get it",
		Tags: tags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *liveInput) (*api.Body[LiveActionResult], error) {
			ctx = ownAPI(ctx)
			id, by, err := e.actionIDs(ctx, in.CampaignID)
			if err != nil {
				return nil, err
			}
			if err := e.StopCanary(ctx, id, by); err != nil {
				return nil, mapErr(ctx, err)
			}
			return e.result(ctx, id, LiveActionResult{})
		})
}
