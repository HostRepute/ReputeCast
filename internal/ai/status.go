package ai

import (
	"context"
	"slices"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// AIStatus is what the entry points need: whether a model is connected and
// works, its name, the features on, this month's use and who can help.
// State, the model, Cap, Usage and Prices are the workspace's model's; a
// feature on the person's own model has its own in Features.
type AIStatus struct {
	// State: ready, not_configured, key_rejected or cap_reached.
	State        string            `json:"state" enum:"ready,not_configured,key_rejected,cap_reached"`
	Provider     string            `json:"provider,omitempty" enum:"anthropic,openai,google,azure,compatible,"`
	ProviderName string            `json:"providerName,omitempty"`
	Model        string            `json:"model,omitempty"`
	ModelName    string            `json:"modelName,omitempty"`
	ImageInput   bool              `json:"imageInput"`
	Features     []AIStatusFeature `json:"features"`
	Agent        AIAgentSettings   `json:"agent"`
	Usage        AIUsageView       `json:"usage"`
	Cap          AICapView         `json:"cap"`
	Prices       AIPricesView      `json:"prices"`
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

// AIStatusFeature is a feature for the person: its switch, and whether it
// can be asked for now with the model it uses for them (their own pick, or
// the workspace's) in the budget that model's requests count against (the
// workspace's cap, or the person's own for their own model).
type AIStatusFeature struct {
	AIFeatureState
	// State is as the top-level State, for this feature's model and budget;
	// On says whether the switches (the feature's, personal keys) allow it.
	State string `json:"state" enum:"ready,not_configured,key_rejected,cap_reached"`
	// Personal: the model is the person's own.
	Personal     bool         `json:"personal"`
	Provider     string       `json:"provider,omitempty" enum:"anthropic,openai,google,azure,compatible,"`
	ProviderName string       `json:"providerName,omitempty"`
	Model        string       `json:"model,omitempty"`
	ModelName    string       `json:"modelName,omitempty"`
	Usage        AIUsageView  `json:"usage"`
	Cap          AICapView    `json:"cap"`
	Prices       AIPricesView `json:"prices"`
}

// Status describes the assistant for a person with ai:use. Each feature is
// on when its own model's account is connected, and has that model's state
// and budget. The model it names is the first account's, or else the first
// connected model a feature uses.
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
	out := AIStatus{State: "not_configured", Features: []AIStatusFeature{}, Usage: AIUsageView{State: "ok", MonthStart: start, ResetsAt: next},
		CanManage: p.Can(authz.IntegrationsManage), Helpers: s.helpers(ctx, ws.ID), HostRepute: s.hostReputeConnected(ctx, ws.ID)}
	out.Agent, err = s.AgentSettings(ctx, ws)
	if err != nil {
		return AIStatus{}, err
	}
	if out.Accounts, err = s.accountViews(ctx, s.Q, m); err != nil {
		return AIStatus{}, err
	}
	out.Defaults = m.defaultViews()
	// Each budget's use this month, read once: the workspace's, and the
	// person's own for their models.
	used := map[bool]usage{}
	budget := func(b *connection) (state string, limit AICapView, view AIUsageView, err error) {
		u, ok := used[b.personal()]
		if !ok {
			if u, err = s.usage(ctx, ws, b); err != nil {
				return "", limit, view, err
			}
			used[b.personal()] = u
		}
		state, view = "ready", AIUsageView{CostMicros: u.costMicros, ReservedCostMicros: u.reservedCost, ReservedTokens: u.reservedTokens, UncertainRequests: u.uncertain,
			InputTokens: u.inputTokens, OutputTokens: u.outputTokens, Tokens: u.tokens, Requests: u.requests, Share: u.share, State: "ok", MonthStart: start, ResetsAt: next}
		limit = AICapView{Kind: b.CapKind, USD: b.CapUsd, Tokens: b.CapTokens}
		if b.personal() {
			limit = AICapView{Kind: "usd", USD: b.PersonalCapUsd}
		}
		switch {
		case u.reached:
			view.State, state = "reached", "cap_reached"
		case u.notice:
			view.State = "notice"
		}
		if b.Status == "key_rejected" {
			state = "key_rejected"
		}
		return state, limit, view, nil
	}
	// A feature follows the person's own pick of their models, if any.
	mine, err := s.memberModels(ctx, s.Q, ws.ID, actorID(p))
	if err != nil {
		return AIStatus{}, err
	}
	for _, f := range Features {
		st := AIStatusFeature{AIFeatureState: AIFeatureState{Key: f, On: false, Available: true}, State: "not_configured", Usage: AIUsageView{State: "ok", MonthStart: start, ResetsAt: next}}
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
		if d.connected() {
			if st.State, st.Cap, st.Usage, err = budget(d); err != nil {
				return AIStatus{}, err
			}
			st.Personal, st.Provider, st.ProviderName, st.Model, st.ModelName = d.personal(), d.Provider, d.providerName(), d.Model, d.modelName()
			st.Prices = AIPricesView{Input: d.InputPrice, Output: d.OutputPrice, Known: d.PriceKnown || d.Demo, From: d.PricesFrom}
		}
		out.Features = append(out.Features, st)
	}
	if !c.connected() {
		return out, nil
	}
	out.Provider, out.ProviderName, out.Model, out.ModelName, out.ImageInput = c.Provider, c.providerName(), c.Model, c.modelName(), c.ImageInput
	out.Prices = AIPricesView{Input: c.InputPrice, Output: c.OutputPrice, Known: c.PriceKnown || c.Demo, From: c.PricesFrom}
	if out.State, out.Cap, out.Usage, err = budget(c); err != nil {
		return AIStatus{}, err
	}
	return out, nil
}
