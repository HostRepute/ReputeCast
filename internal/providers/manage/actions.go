package manage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

func (s *Service) provider(ctx context.Context, ws *store.Workspace, id string) (store.Provider, error) {
	pid, err := uuid.Parse(id)
	if err != nil {
		return store.Provider{}, api.NotFound("Provider")
	}
	p, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: pid})
	return p, api.MapError(err, "Provider")
}

func (s *Service) audit(ctx context.Context, q *store.Queries, ws *store.Workspace, p store.Provider, action string, before, after any, meta map[string]any) error {
	wsID := ws.ID
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: action, TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name, Before: before, After: after, Metadata: meta})
}

// CheckNow runs a health check immediately ("Re-check SES now").
func (s *Service) CheckNow(ctx context.Context, ws *store.Workspace, id string) (ProviderCard, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderCard{}, err
	}
	if !IsDemo(p) {
		if p, err = s.Check(ctx, p, true); err != nil {
			return ProviderCard{}, err
		}
	}
	if err := s.audit(ctx, s.Q, ws, p, "provider.check", nil, nil, map[string]any{"status": p.Status}); err != nil {
		return ProviderCard{}, err
	}
	return s.Card(ctx, ws, p.ID)
}

// Card returns one provider's card.
func (s *Service) Card(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ProviderCard, error) {
	sn, err := s.load(ctx, ws)
	if err != nil {
		return ProviderCard{}, err
	}
	p, ok := sn.byID[id]
	if !ok {
		return ProviderCard{}, api.NotFound("Provider")
	}
	return s.card(sn, p), nil
}

// ProviderCredentialsInput replaces some or all write-only credentials.
type ProviderCredentialsInput struct {
	Values map[string]string `json:"values" doc:"Credential keys and their new values. Keys not listed keep their current value."`
}

// ReplaceCredentials checks new credentials with the provider and stores
// them encrypted. Values are never returned.
func (s *Service) ReplaceCredentials(ctx context.Context, ws *store.Workspace, id string, in ProviderCredentialsInput) (ProviderCard, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderCard{}, err
	}
	d, ok := providers.Lookup(p.Adapter)
	if !ok {
		return ProviderCard{}, api.Conflict("This provider's adapter is not available in this build.")
	}
	secret := map[string]bool{}
	for _, f := range d.Fields {
		if f.Secret {
			secret[f.Key] = true
		}
	}
	if len(in.Values) == 0 {
		return ProviderCard{}, api.Invalid("body.values", "Enter the new value to save.")
	}
	cur, err := s.Factory.Credentials(p)
	if err != nil {
		return ProviderCard{}, err
	}
	creds := map[string]any{}
	_ = json.Unmarshal(cur, &creds)
	keys := []string{}
	for k, v := range in.Values {
		if !secret[k] {
			return ProviderCard{}, api.Invalid("body.values."+k, "This is not a credential of "+p.Name+".")
		}
		v = strings.TrimSpace(v)
		if v == "" {
			return ProviderCard{}, api.Invalid("body.values."+k, "Enter the new value.")
		}
		creds[k] = v
		keys = append(keys, k)
	}
	raw, _ := json.Marshal(creds)
	adapter, err := s.Factory.OpenWith(p, raw)
	if err != nil {
		return ProviderCard{}, api.Err(http.StatusUnprocessableEntity, api.CodeValidation, "These credentials can't be used: "+err.Error())
	}
	if !IsDemo(p) {
		if v, ok := adapter.(providers.Verifier); ok {
			cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			_, verr := v.Verify(cctx)
			cancel()
			if verr != nil {
				msg := p.Name + " didn't accept them: " + verr.Error() + ". Nothing was saved."
				if errors.Is(verr, providers.ErrUnauthorized) {
					msg = p.Name + " rejected these credentials. Nothing was saved; check the value and try again."
				}
				return ProviderCard{}, api.Err(http.StatusUnprocessableEntity, api.CodeInvalidCredentials, msg)
			}
		}
	}
	sealed, err := s.Factory.Box.Seal(raw, CredentialsAD(p))
	if err != nil {
		return ProviderCard{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ProvUpdateCredentials(ctx, store.ProvUpdateCredentialsParams{ID: p.ID, CredentialsEnc: sealed}); err != nil {
			return err
		}
		return s.audit(ctx, q, ws, p, "provider.credentials_replace", nil, nil, map[string]any{"keys": keys})
	})
	if err != nil {
		return ProviderCard{}, err
	}
	if !IsDemo(p) {
		if _, err := s.Check(ctx, p, true); err != nil {
			s.log().WarnContext(ctx, "check after credential change", "provider", p.ID, "err", err)
		}
	}
	return s.Card(ctx, ws, p.ID)
}

// ProviderSettingsInput changes a provider's label, pricing and limits.
type ProviderSettingsInput struct {
	Label   *string               `json:"label,omitempty" maxLength:"120"`
	Pricing *ProviderPricingInput `json:"pricing,omitempty"`
	Limits  *ProviderLimitsInput  `json:"limits,omitempty"`
}

// ProviderPricingInput is how the provider is paid for.
type ProviderPricingInput struct {
	Model   string  `json:"model" enum:"metered,plan,self_hosted"`
	Per1000 float64 `json:"per1000,omitempty" minimum:"0" maximum:"1000"`
	Monthly float64 `json:"monthly,omitempty" minimum:"0" maximum:"1000000"`
	// NextTierAdds / NextTierPrice describe the plan's next step.
	NextTierAdds  int64   `json:"nextTierAdds,omitempty" minimum:"0"`
	NextTierPrice float64 `json:"nextTierPrice,omitempty" minimum:"0"`
}

// ProviderLimitsInput are the bounce and complaint lines drawn on the meters.
type ProviderLimitsInput struct {
	Bounce    float64 `json:"bounce" minimum:"0.1" maximum:"50"`
	Complaint float64 `json:"complaint" minimum:"0.01" maximum:"5"`
}

// UpdateSettings changes a provider's label, pricing or limits.
func (s *Service) UpdateSettings(ctx context.Context, ws *store.Workspace, id string, in ProviderSettingsInput) (ProviderCard, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderCard{}, err
	}
	label := p.Label
	if in.Label != nil {
		label = strings.TrimSpace(*in.Label)
	}
	pricing := state.PricingOf(p)
	if in.Pricing != nil {
		pricing.Model, pricing.Per1000, pricing.Monthly = in.Pricing.Model, in.Pricing.Per1000, in.Pricing.Monthly
		pricing.NextTier = nil
		if in.Pricing.NextTierAdds > 0 {
			pricing.NextTier = &state.NextTier{Adds: in.Pricing.NextTierAdds, Price: in.Pricing.NextTierPrice}
		}
		if pricing.Model == "metered" && pricing.Per1000 <= 0 {
			return ProviderCard{}, api.Invalid("body.pricing.per1000", "Enter the price per 1,000 emails.")
		}
	}
	limits := state.LimitsOf(p)
	if in.Limits != nil {
		limits.Bounce.Limit, limits.Complaint.Limit = in.Limits.Bounce, in.Limits.Complaint
	}
	pb, _ := json.Marshal(pricing)
	lb, _ := json.Marshal(limits)
	before := map[string]any{"label": p.Label, "pricing": p.Pricing, "limits": p.Limits}
	after := map[string]any{"label": label, "pricing": json.RawMessage(pb), "limits": json.RawMessage(lb)}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ProvUpdateSettings(ctx, store.ProvUpdateSettingsParams{ID: p.ID, Label: label, Pricing: pb, Limits: lb}); err != nil {
			return err
		}
		return s.audit(ctx, q, ws, p, "provider.update", before, after, nil)
	})
	if err != nil {
		return ProviderCard{}, err
	}
	return s.Card(ctx, ws, p.ID)
}

// Disconnect removes a provider that no route uses any more.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, id string) error {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return err
	}
	set, err := routing.Load(ctx, s.Q, ws.ID)
	if err != nil {
		return err
	}
	var uses []string
	for _, stream := range routing.Streams {
		for _, r := range set.StreamRules(stream) {
			used := r.Fallback != nil && *r.Fallback == p.ID
			for _, sp := range r.Splits {
				if sp.ProviderID == p.ID && sp.Weight > 0 {
					used = true
				}
			}
			if used {
				uses = append(uses, streamLabel(string(stream)))
				break
			}
		}
	}
	if len(uses) > 0 {
		return api.Conflict(fmt.Sprintf("%s still carries %s. Change the routing first, then disconnect it.", p.Name, strings.Join(uses, " and ")))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ProvDisconnect(ctx, p.ID); err != nil {
			return err
		}
		now := s.now()
		if _, err := q.ProvEndFailovers(ctx, store.ProvEndFailoversParams{ProviderID: p.ID, EndedAt: &now, EndReason: "disconnected"}); err != nil {
			return err
		}
		return s.audit(ctx, q, ws, p, "provider.disconnect", map[string]any{"status": p.Status}, nil, nil)
	})
}

// ProviderTrackingCheck is one row of "Provider-side tracking check".
type ProviderTrackingCheck struct {
	ProviderID  string     `json:"providerId"`
	Provider    string     `json:"provider"`
	Status      string     `json:"status" enum:"ok,warn,unknown"`
	OpenPixel   bool       `json:"openPixel"`
	LinkRewrite bool       `json:"linkRewrite"`
	Detail      string     `json:"detail"`
	Scope       string     `json:"scope"`
	Mode        string     `json:"mode" enum:"reputecast,provider"`
	CheckedAt   *time.Time `json:"checkedAt,omitempty"`
	CanDisable  bool       `json:"canDisable"`
}

// ProviderTrackingChecks lists provider-side tracking for every provider.
func (s *Service) ProviderTrackingChecks(ctx context.Context, ws *store.Workspace) ([]ProviderTrackingCheck, error) {
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := []ProviderTrackingCheck{}
	for _, p := range provs {
		t := state.TrackingOf(p)
		tc := ProviderTrackingCheck{ProviderID: p.ID.String(), Provider: p.Name, Status: "ok", OpenPixel: t.OpenPixel, LinkRewrite: t.LinkRewrite,
			Detail: t.Detail, Scope: t.Scope, Mode: t.Mode, CheckedAt: t.CheckedAt}
		if (t.OpenPixel || t.LinkRewrite) && t.Mode != "provider" {
			tc.Status = "warn"
		}
		if t.CheckedAt == nil && t.Detail == "" {
			tc.Status = "unknown"
		}
		if adapter, err := s.Factory.Open(ctx, p); err == nil {
			_, tc.CanDisable = adapter.(providers.TrackingManager)
		}
		out = append(out, tc)
	}
	return out, nil
}

// ProviderTrackingActionInput acts on provider-side tracking.
type ProviderTrackingActionInput struct {
	// Action: disable (turn it off at the provider), keep_provider (keep the
	// provider's tracking and ignore ReputeCast's), use_reputecast (back to
	// ReputeCast's own tracking), recheck.
	Action string `json:"action" enum:"disable,keep_provider,use_reputecast,recheck"`
}

// TrackingAction applies a tracking decision.
func (s *Service) TrackingAction(ctx context.Context, ws *store.Workspace, id string, in ProviderTrackingActionInput) (ProviderTrackingCheck, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderTrackingCheck{}, err
	}
	before := state.TrackingOf(p)
	switch in.Action {
	case "disable":
		if IsDemo(p) {
			t := before
			t.OpenPixel, t.LinkRewrite, t.Mode = false, false, "reputecast"
			t.Detail = strings.NewReplacer("is on", "is off", "HtmlAndText", "None", "HtmlOnly", "None", "TextOnly", "None").Replace(t.Detail)
			b, _ := json.Marshal(t)
			if err := s.Q.ProvUpdateTracking(ctx, store.ProvUpdateTrackingParams{ID: p.ID, Tracking: b}); err != nil {
				return ProviderTrackingCheck{}, err
			}
			break
		}
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			return ProviderTrackingCheck{}, err
		}
		tm, ok := adapter.(providers.TrackingManager)
		if !ok {
			return ProviderTrackingCheck{}, api.Conflict(p.Name + " can't change tracking over its API. Turn it off in " + p.Name + "'s settings.")
		}
		cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		err = tm.DisableTracking(cctx)
		cancel()
		if err != nil {
			return ProviderTrackingCheck{}, api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, p.Name+" didn't accept the change: "+err.Error())
		}
		if _, err := s.refreshTracking(ctx, p); err != nil {
			return ProviderTrackingCheck{}, err
		}
	case "keep_provider", "use_reputecast":
		t := before
		t.Mode = map[string]string{"keep_provider": "provider", "use_reputecast": "reputecast"}[in.Action]
		b, _ := json.Marshal(t)
		if err := s.Q.ProvUpdateTracking(ctx, store.ProvUpdateTrackingParams{ID: p.ID, Tracking: b}); err != nil {
			return ProviderTrackingCheck{}, err
		}
	case "recheck":
		if !IsDemo(p) {
			if _, err := s.refreshTracking(ctx, p); err != nil {
				return ProviderTrackingCheck{}, err
			}
		}
	}
	p, err = s.Q.ProvGetAny(ctx, p.ID)
	if err != nil {
		return ProviderTrackingCheck{}, err
	}
	after := state.TrackingOf(p)
	if in.Action != "recheck" {
		if err := s.audit(ctx, s.Q, ws, p, "provider.tracking_"+in.Action, before, after, nil); err != nil {
			return ProviderTrackingCheck{}, err
		}
	}
	list, err := s.ProviderTrackingChecks(ctx, ws)
	if err != nil {
		return ProviderTrackingCheck{}, err
	}
	for _, tc := range list {
		if tc.ProviderID == p.ID.String() {
			return tc, nil
		}
	}
	return ProviderTrackingCheck{}, api.NotFound("Provider")
}

// ProviderWebhookTestResult is the outcome of "Send test event".
type ProviderWebhookTestResult struct {
	EndpointOK     bool   `json:"endpointOk"`
	EndpointStatus int    `json:"endpointStatus"`
	EndpointDetail string `json:"endpointDetail"`
	// ProviderTest is true when the provider was asked to send a test event.
	ProviderTest       bool   `json:"providerTest"`
	ProviderTestDetail string `json:"providerTestDetail,omitempty"`
}

// TestWebhook checks that the webhook endpoint is reachable from outside
// and, when the provider can, asks it to send a test event.
func (s *Service) TestWebhook(ctx context.Context, ws *store.Workspace, id string) (ProviderWebhookTestResult, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderWebhookTestResult{}, err
	}
	out := ProviderWebhookTestResult{}
	code, err := s.selfTest(ctx, p)
	out.EndpointStatus = code
	switch {
	case err != nil:
		out.EndpointDetail = "Couldn't reach " + WebhookURL(s.URLs, p) + ": " + err.Error()
	case code == http.StatusOK:
		out.EndpointOK, out.EndpointDetail = true, "200 OK on a self-test"
	default:
		out.EndpointDetail = fmt.Sprintf("%d on a self-test", code)
	}
	if !IsDemo(p) {
		if adapter, err := s.Factory.Open(ctx, p); err == nil {
			if wt, ok := adapter.(providers.WebhookTester); ok {
				cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
				err := wt.SendTestEvent(cctx, WebhookURL(s.URLs, p))
				cancel()
				out.ProviderTest = err == nil
				if err != nil {
					out.ProviderTestDetail = err.Error()
				}
			}
		}
	}
	return out, s.audit(ctx, s.Q, ws, p, "provider.webhook_test", nil, nil, map[string]any{"endpoint_ok": out.EndpointOK, "provider_test": out.ProviderTest})
}

// RegisterWebhook (re-)registers the webhook at the provider.
func (s *Service) RegisterWebhook(ctx context.Context, ws *store.Workspace, id string) error {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return err
	}
	if !IsDemo(p) {
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			return err
		}
		wr, ok := adapter.(providers.WebhookRegistrar)
		if !ok {
			return api.Conflict(p.Name + " webhooks are set up in " + p.Name + " itself. Point them at " + WebhookURL(s.URLs, p) + ".")
		}
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = wr.RegisterWebhook(cctx, WebhookURL(s.URLs, p))
		cancel()
		if err != nil {
			return api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, p.Name+" didn't accept the webhook: "+err.Error())
		}
	}
	return s.audit(ctx, s.Q, ws, p, "provider.webhook_register", nil, nil, map[string]any{"url": WebhookURL(s.URLs, p)})
}

// BackfillArgs pulls message status from a provider's API.
type BackfillArgs struct {
	ProviderID uuid.UUID `json:"provider_id"`
}

// Kind implements river.JobArgs.
func (BackfillArgs) Kind() string { return "providers.backfill" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (BackfillArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 5 * time.Minute}}
}

// ProviderBackfillResult says how many messages will be looked up.
type ProviderBackfillResult struct {
	Queued int64 `json:"queued"`
}

// Backfill queues a job that pulls the status of messages with no event.
func (s *Service) Backfill(ctx context.Context, ws *store.Workspace, id string) (ProviderBackfillResult, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderBackfillResult{}, err
	}
	since := s.now().Add(-7 * 24 * time.Hour)
	if h := state.HealthOf(p); h.Webhooks != nil && h.Webhooks.SilentSince != nil {
		since = *h.Webhooks.SilentSince
	}
	cnt, err := s.Q.ProvAwaitingCount(ctx, store.ProvAwaitingCountParams{ProviderID: &p.ID, Since: &since})
	if err != nil {
		return ProviderBackfillResult{}, err
	}
	if !IsDemo(p) {
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			return ProviderBackfillResult{}, err
		}
		if _, ok := adapter.(providers.EventBackfiller); !ok {
			return ProviderBackfillResult{}, api.Conflict(p.Name + "'s API can't report message status, so there is nothing to backfill from.")
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if c := s.jobs(); c != nil && !IsDemo(p) {
			if _, err := c.InsertTx(ctx, tx, BackfillArgs{ProviderID: p.ID}, nil); err != nil {
				return err
			}
		}
		return s.audit(ctx, store.New(tx), ws, p, "provider.backfill", nil, nil, map[string]any{"messages": cnt.Total})
	})
	return ProviderBackfillResult{Queued: cnt.Total}, err
}

func (s *Service) jobs() *jobs.Client {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

// ProviderBroadcastStreamResult names the stream created.
type ProviderBroadcastStreamResult struct {
	StreamID string `json:"streamId"`
}

// CreateBroadcastStream creates the stream a provider's terms require for
// bulk mail (Postmark) and registers its webhook.
func (s *Service) CreateBroadcastStream(ctx context.Context, ws *store.Workspace, id string) (ProviderBroadcastStreamResult, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderBroadcastStreamResult{}, err
	}
	streamID := "broadcast"
	if !IsDemo(p) {
		adapter, err := s.Factory.Open(ctx, p)
		if err != nil {
			return ProviderBroadcastStreamResult{}, err
		}
		bc, ok := adapter.(providers.BroadcastStreamCreator)
		if !ok {
			return ProviderBroadcastStreamResult{}, api.Conflict(p.Name + " has no separate stream for bulk mail.")
		}
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		streamID, err = bc.CreateBroadcastStream(cctx, "ReputeCast Broadcasts")
		cancel()
		if err != nil {
			return ProviderBroadcastStreamResult{}, api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, p.Name+" didn't create the stream: "+err.Error())
		}
	}
	cfg := map[string]any{}
	_ = json.Unmarshal(p.Config, &cfg)
	cfg["broadcast_stream"] = streamID
	cb, _ := json.Marshal(cfg)
	caps := state.Caps(p)
	caps.BroadcastStream = true
	capb, _ := json.Marshal(caps)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ProvUpdateConfig(ctx, store.ProvUpdateConfigParams{ID: p.ID, Config: cb}); err != nil {
			return err
		}
		if err := q.ProvUpdateCapabilities(ctx, store.ProvUpdateCapabilitiesParams{ID: p.ID, Capabilities: capb}); err != nil {
			return err
		}
		return s.audit(ctx, q, ws, p, "provider.broadcast_stream_create", nil, map[string]any{"stream": streamID}, nil)
	})
	if err != nil {
		return ProviderBroadcastStreamResult{}, err
	}
	if !IsDemo(p) {
		p.Config = cb
		if adapter, err := s.Factory.Open(ctx, p); err == nil {
			if wr, ok := adapter.(providers.WebhookRegistrar); ok {
				cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				if err := wr.RegisterWebhook(cctx, WebhookURL(s.URLs, p)); err != nil {
					s.log().WarnContext(ctx, "register webhook for broadcast stream", "provider", p.ID, "err", err)
				}
				cancel()
			}
		}
	}
	return ProviderBroadcastStreamResult{StreamID: streamID}, nil
}

// ProviderHeldDecisionInput decides what happens to a broadcast's held share.
type ProviderHeldDecisionInput struct {
	// Action: wait (resume automatically when the provider recovers),
	// move (send the held recipients through another provider), cancel
	// (cancel the send).
	Action   string `json:"action" enum:"wait,move,cancel"`
	TargetID string `json:"targetProviderId,omitempty" doc:"Provider to move the held share to (move)"`
}

// ProviderHeldDecisionResult is what changed.
type ProviderHeldDecisionResult struct {
	Action string `json:"action"`
	Moved  int64  `json:"moved"`
}

// DecideHeld resolves a broadcast share held on a provider that is down.
func (s *Service) DecideHeld(ctx context.Context, ws *store.Workspace, providerID, campaignID string, in ProviderHeldDecisionInput) (ProviderHeldDecisionResult, error) {
	p, err := s.provider(ctx, ws, providerID)
	if err != nil {
		return ProviderHeldDecisionResult{}, err
	}
	cid, err := uuid.Parse(campaignID)
	if err != nil {
		return ProviderHeldDecisionResult{}, api.NotFound("Campaign")
	}
	held, err := s.Q.ProvHeldRoutes(ctx, store.ProvHeldRoutesParams{WorkspaceID: ws.ID, ProviderID: p.ID})
	if err != nil {
		return ProviderHeldDecisionResult{}, err
	}
	var route *store.ProvHeldRoutesRow
	for i := range held {
		if held[i].CampaignID == cid {
			route = &held[i]
		}
	}
	if route == nil {
		return ProviderHeldDecisionResult{}, api.Conflict("Nothing of this campaign is held on " + p.Name + " any more.")
	}
	pr := api.PrincipalFrom(ctx)
	// Waiting and moving resume sending (campaigns:send); cancelling stops
	// it (campaigns:pause).
	need := authz.CampaignsSend
	if in.Action == "cancel" {
		need = authz.CampaignsPause
	}
	if pr == nil || !pr.Can(need) {
		return ProviderHeldDecisionResult{}, api.Forbidden(need, nil)
	}
	now := s.now()
	out := ProviderHeldDecisionResult{Action: in.Action}
	if in.Action == "cancel" {
		var by uuid.UUID
		if pr != nil {
			by = pr.UserID
		}
		if s.Sending != nil {
			if err := s.Sending().Cancel(ctx, cid, by); err == nil {
				return out, s.audit(ctx, s.Q, ws, p, "campaign.held_cancel", nil, nil, map[string]any{"campaign_id": cid.String()})
			} else if !errors.Is(err, sending.ErrUnavailable) {
				return out, err
			}
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		switch in.Action {
		case "wait":
			if _, err := q.ProvSetRouteStatus(ctx, store.ProvSetRouteStatusParams{CampaignID: cid, ProviderID: p.ID, Status: "held", StatusDetail: `{"decision":"wait"}`}); err != nil {
				return err
			}
		case "move":
			tid, err := uuid.Parse(in.TargetID)
			if err != nil {
				return api.Invalid("body.targetProviderId", "Pick the provider to move the held recipients to.")
			}
			target, err := q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: tid})
			if err != nil {
				return api.Invalid("body.targetProviderId", "That provider is not connected to this workspace.")
			}
			if target.ID == p.ID || !state.Available(target) {
				return api.Conflict(target.Name + " can't take mail right now.")
			}
			if !state.Caps(target).BulkAllowed() {
				return api.Conflict(target.Name + "'s terms don't allow bulk marketing.")
			}
			moved := max(route.Planned-route.Sent, 0)
			if _, err := tx.Exec(ctx, `INSERT INTO campaign_routes (campaign_id, provider_id, planned, status) VALUES ($1, $2, $3, 'pending')
				ON CONFLICT (campaign_id, provider_id) DO UPDATE SET planned = campaign_routes.planned + EXCLUDED.planned,
				status = CASE WHEN campaign_routes.status = 'done' THEN 'pending' ELSE campaign_routes.status END`, cid, target.ID, moved); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE campaign_routes SET planned = $3, status = 'done', status_detail = $4 WHERE campaign_id = $1 AND provider_id = $2`,
				cid, p.ID, route.Sent, fmt.Sprintf(`{"decision":"move","to":%q}`, target.Name)); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE messages SET provider_id = $3, status = 'queued' WHERE campaign_id = $1 AND provider_id = $2 AND status IN ('held', 'queued')`, cid, p.ID, target.ID); err != nil {
				return err
			}
			out.Moved = moved
		case "cancel":
			if _, err := tx.Exec(ctx, `UPDATE campaigns SET status = 'cancelled', completed_at = $2, updated_at = $2 WHERE id = $1 AND status IN ('sending', 'paused', 'scheduled')`, cid, now); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE campaign_routes SET status = 'cancelled' WHERE campaign_id = $1 AND status IN ('pending', 'sending', 'held', 'paused')`, cid); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE messages SET status = 'cancelled' WHERE campaign_id = $1 AND status IN ('queued', 'held')`, cid); err != nil {
				return err
			}
		}
		resolution := map[string]string{"move": "moved", "cancel": "cancelled"}[in.Action]
		if resolution != "" {
			var by *uuid.UUID
			if pr != nil && pr.UserID != uuid.Nil {
				u := pr.UserID
				by = &u
			}
			if _, err := tx.Exec(ctx, `UPDATE campaign_incidents SET resolved_at = $3, resolution = $4, resolved_by = $5
				WHERE campaign_id = $1 AND provider_id = $2 AND kind = 'provider_outage' AND resolved_at IS NULL`, cid, p.ID, now, resolution, by); err != nil {
				return err
			}
		}
		return s.audit(ctx, q, ws, p, "campaign.held_"+in.Action, nil, nil, map[string]any{"campaign_id": cid.String(), "campaign": route.CampaignName, "target": in.TargetID, "moved": out.Moved})
	})
	return out, err
}
