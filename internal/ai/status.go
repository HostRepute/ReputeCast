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
	Agent        AIAgentSettings  `json:"agent"`
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
	// Accounts are the model accounts with their models; Defaults the
	// model each feature uses.
	Accounts []AIAccountView    `json:"accounts"`
	Defaults []AIFeatureDefault `json:"defaults"`
}

// Status describes the assistant for a person with ai:use. Each feature is
// on when its own model's account is connected. The model it names is the
// first account's, or else the first connected model a feature uses.
func (s *Service) Status(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AIStatus, error) {
	m, err := s.models(ctx, s.Q, ws.ID)
	if err != nil {
		return AIStatus{}, err
	}
	var c *connection
	if len(m.list) > 0 && m.list[0].connected() {
		c = m.list[0]
	}
	for _, f := range Features {
		if d := m.of(f); c == nil && d.connected() {
			c = d
		}
	}
	start, next := s.month(ws)
	out := AIStatus{State: "not_configured", Features: []AIFeatureState{}, Usage: AIUsageView{State: "ok", MonthStart: start, ResetsAt: next},
		CanManage: p.Can(authz.IntegrationsManage), Helpers: s.helpers(ctx, ws.ID), HostRepute: s.hostReputeConnected(ctx, ws.ID)}
	out.Agent, err = s.AgentSettings(ctx, ws)
	if err != nil {
		return AIStatus{}, err
	}
	if out.Accounts, err = s.accountViews(ctx, s.Q, m); err != nil {
		return AIStatus{}, err
	}
	out.Defaults = m.defaultViews()
	// A feature follows the person's own pick of their models, if any.
	mine, err := s.memberModels(ctx, s.Q, ws.ID, actorID(p))
	if err != nil {
		return AIStatus{}, err
	}
	for _, f := range Features {
		st := AIFeatureState{Key: f, On: false, Available: true}
		d := m.of(f)
		if o, ok := mine[f]; ok {
			d = o
		}
		if d != nil {
			st.ModelID = &d.ModelID
			if d.connected() && (!d.personal() || d.PersonalKeys) {
				st.On = !slices.Contains(d.FeaturesOff, f)
				if f == FeatureAltText && !d.ImageInput {
					st.On, st.Available = false, false
				}
			}
		}
		out.Features = append(out.Features, st)
	}
	if !c.connected() {
		return out, nil
	}
	out.State, out.Provider, out.ProviderName, out.Model, out.ModelName, out.ImageInput = "ready", c.Provider, c.providerName(), c.Model, c.modelName(), c.ImageInput
	out.Cap = AICapView{Kind: c.CapKind, USD: c.CapUsd, Tokens: c.CapTokens}
	out.Prices = AIPricesView{Input: c.InputPrice, Output: c.OutputPrice, Known: c.PriceKnown || c.Demo, From: c.PricesFrom}
	u, err := s.usage(ctx, ws, c)
	if err != nil {
		return AIStatus{}, err
	}
	out.Usage.CostMicros, out.Usage.InputTokens, out.Usage.OutputTokens, out.Usage.Tokens, out.Usage.Requests, out.Usage.Share =
		u.costMicros, u.inputTokens, u.outputTokens, u.tokens, u.requests, u.share
	out.Usage.ReservedCostMicros, out.Usage.ReservedTokens, out.Usage.UncertainRequests = u.reservedCost, u.reservedTokens, u.uncertain
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
