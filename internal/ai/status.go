package ai

import (
	"context"
	"slices"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// AIStatus is what the entry points need: whether a model is connected and
// works, its name, the features on, this month's use and who can help.
type AIStatus struct {
	// State: ready, not_configured, key_rejected or cap_reached.
	State        string           `json:"state" enum:"ready,not_configured,key_rejected,cap_reached"`
	Provider     string           `json:"provider,omitempty" enum:"anthropic,openai,google,azure,compatible,"`
	ProviderName string           `json:"providerName,omitempty"`
	Model        string           `json:"model,omitempty"`
	ModelName    string           `json:"modelName,omitempty"`
	ImageInput   bool             `json:"imageInput"`
	Features     []AIFeatureState `json:"features"`
	Usage        AIUsageView      `json:"usage"`
	Cap          AICapView        `json:"cap"`
	Prices       AIPricesView     `json:"prices"`
	// CanManage: the person can connect a model, raise the cap or turn a
	// feature on; Helpers can.
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
	// HostRepute: HostRepute is connected, so reputation questions go to
	// its assistant.
	HostRepute bool `json:"hostRepute"`
}

// Status describes the assistant for a person with ai:use.
func (s *Service) Status(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AIStatus, error) {
	c, err := s.load(ctx, s.Q, ws.ID)
	if err != nil {
		return AIStatus{}, err
	}
	start, next := s.month(ws)
	out := AIStatus{State: "not_configured", Features: []AIFeatureState{}, Usage: AIUsageView{State: "ok", MonthStart: start, ResetsAt: next},
		CanManage: p.Can(authz.IntegrationsManage), Helpers: s.helpers(ctx, ws.ID), HostRepute: s.hostReputeConnected(ctx, ws.ID)}
	for _, f := range Features {
		st := AIFeatureState{Key: f, On: false, Available: true}
		if c.connected() {
			st.On = !slices.Contains(c.FeaturesOff, f)
			if f == FeatureAltText && !c.ImageInput {
				st.On, st.Available = false, false
			}
		}
		out.Features = append(out.Features, st)
	}
	if !c.connected() {
		return out, nil
	}
	out.State, out.Provider, out.ProviderName, out.Model, out.ModelName, out.ImageInput = "ready", c.Provider, c.providerName(), c.Model, c.modelName(), c.ImageInput
	out.Cap = AICapView{Kind: c.CapKind, USD: c.CapUsd, Tokens: c.CapTokens}
	out.Prices = AIPricesView{Input: c.InputPrice, Output: c.OutputPrice, From: c.PricesFrom}
	u, err := s.usage(ctx, ws, c)
	if err != nil {
		return AIStatus{}, err
	}
	out.Usage.CostMicros, out.Usage.InputTokens, out.Usage.OutputTokens, out.Usage.Tokens, out.Usage.Requests, out.Usage.Share =
		u.costMicros, u.inputTokens, u.outputTokens, u.tokens, u.requests, u.share
	switch {
	case u.reached:
		out.Usage.State, out.State = "reached", "cap_reached"
	case u.notice:
		out.Usage.State = "notice"
	}
	if c.Status == "key_rejected" {
		out.State = "key_rejected"
	}
	return out, nil
}
