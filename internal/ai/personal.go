package ai

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Personal keys (assistant brief D4): when integration managers turn them
// on, a member with ai:use may add their own model accounts. Only their
// owner's requests use them, and those count against the owner's own
// monthly cap, never the workspace's. A person picks one of their own
// models per feature for their own requests (ai_member_feature_models); the
// workspace's defaults are never personal. Managers see who has personal
// accounts and their use, never a key, and may remove them.

// AIPersonalSettings are the personal keys' part of the AI settings: the
// switch, the cap, the caller's own accounts and, for managers, who has
// personal accounts.
type AIPersonalSettings struct {
	// Enabled: integration managers allow personal keys.
	Enabled bool `json:"enabled"`
	// CapUSD is each person's monthly budget for their own keys, in dollars.
	CapUSD   float64    `json:"capUsd"`
	CapSetBy string     `json:"capSetBy,omitempty"`
	CapSetAt *time.Time `json:"capSetAt,omitempty"`
	// CanAdd: personal keys are on and the caller holds ai:use.
	CanAdd bool `json:"canAdd"`
	// Accounts are the caller's own accounts with their models; a model's
	// features are the ones the caller picked it for.
	Accounts []AIAccountView `json:"accounts"`
	// Choices are the caller's own models per feature, for their requests.
	Choices []AIFeatureDefault  `json:"choices"`
	Usage   AIPersonalUsageView `json:"usage"`
	// People are the members with personal accounts or personal use this
	// month, for integration managers (empty otherwise). Never a key.
	People []AIPersonalPerson `json:"people"`
	// Catalog is every provider a personal account may use, with the
	// models and prices of ReputeCast's price list (as /ai gives them);
	// PriceListDate is that list's date.
	Catalog       []AIProviderEntry `json:"catalog"`
	PriceListDate time.Time         `json:"priceListDate"`
}

// AIPersonalUsageView is a person's use of their own keys this month
// against their cap.
type AIPersonalUsageView struct {
	CostMicros         int64     `json:"costMicros"`
	ReservedCostMicros int64     `json:"reservedCostMicros"`
	UncertainRequests  int64     `json:"uncertainRequests"`
	Requests           int64     `json:"requests"`
	Share              float64   `json:"share" doc:"Share of the cap used, 0 to 1"`
	State              string    `json:"state" enum:"ok,notice,reached"`
	ResetsAt           time.Time `json:"resetsAt"`
}

// AIPersonalPerson is a member's personal accounts and their use this
// month, as managers see them.
type AIPersonalPerson struct {
	UserID     uuid.UUID           `json:"userId"`
	Name       string              `json:"name"`
	Accounts   []AIPersonalAccount `json:"accounts"`
	CostMicros int64               `json:"costMicros"`
	Requests   int64               `json:"requests"`
}

// AIPersonalAccount is a member's personal account as managers see it:
// the provider and its models' names, never the key.
type AIPersonalAccount struct {
	ID           uuid.UUID `json:"id"`
	Provider     string    `json:"provider" enum:"anthropic,openai,google,azure,compatible"`
	ProviderName string    `json:"providerName"`
	Status       string    `json:"status" enum:"connected,key_rejected,disconnected"`
	Models       []string  `json:"models"`
}

// AIPersonalKeysInput sets the personal-keys switch and each person's cap.
type AIPersonalKeysInput struct {
	Enabled bool `json:"enabled"`
	// CapUSD is each person's monthly budget for their own keys, in dollars.
	CapUSD float64 `json:"capUsd" minimum:"0"`
}

// ownerPermission is what changing an account takes: integrations:manage
// for a workspace account, ai:use for one's own.
func ownerPermission(owner *uuid.UUID) authz.Permission {
	if owner != nil {
		return authz.AIUse
	}
	return authz.IntegrationsManage
}

// personalMeta marks an audit entry about a personal key.
func personalMeta(owner *uuid.UUID, meta map[string]any) map[string]any {
	if owner != nil {
		meta["personal"] = true
	}
	return meta
}

func errOr(err, otherwise error) error {
	if err != nil {
		return err
	}
	return otherwise
}

// personalKeysOn reports whether managers allow personal keys.
func (s *Service) personalKeysOn(ctx context.Context, ws uuid.UUID) (bool, error) {
	row, err := s.Q.AIWorkspaceSettings(ctx, ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return row.PersonalKeys, err
}

// memberModels are the models a person picked for their own requests, by
// feature.
func (s *Service) memberModels(ctx context.Context, q *store.Queries, ws uuid.UUID, user *uuid.UUID) (map[string]*connection, error) {
	out := map[string]*connection{}
	if user == nil {
		return out, nil
	}
	rows, err := q.AIMemberFeatureModels(ctx, store.AIMemberFeatureModelsParams{WorkspaceID: ws, UserID: *user})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		c, err := s.binding(ctx, q, ws, &r.ModelID)
		if err != nil {
			return nil, err
		}
		if c != nil {
			out[r.Feature] = c
		}
	}
	return out, nil
}

// pickedFeatures are the features a person picked one of their models
// matching for, in the settings' order.
func (s *Service) pickedFeatures(ctx context.Context, q *store.Queries, ws, user uuid.UUID, match func(*connection) bool) ([]string, error) {
	mine, err := s.memberModels(ctx, q, ws, &user)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, f := range defaultFeatures {
		if c, ok := mine[f]; ok && match(c) {
			out = append(out, f)
		}
	}
	return out, nil
}

// Personal returns the personal keys' settings for the caller.
func (s *Service) Personal(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AIPersonalSettings, error) {
	start, next := s.month(ws)
	out := AIPersonalSettings{CapUSD: 10, Accounts: []AIAccountView{}, Choices: []AIFeatureDefault{}, People: []AIPersonalPerson{},
		Usage: AIPersonalUsageView{State: "ok", ResetsAt: next}, Catalog: catalog(), PriceListDate: llm.PriceListDate}
	row, err := s.Q.AIWorkspaceSettings(ctx, ws.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if err == nil {
		out.Enabled, out.CapUSD, out.CapSetBy, out.CapSetAt = row.PersonalKeys, row.PersonalCapUsd, s.userName(ctx, row.PersonalCapSetBy), row.PersonalCapSetAt
	}
	out.CanAdd = out.Enabled && p.Can(authz.AIUse)
	if user := actorID(p); user != nil {
		ids, err := s.Q.AIPersonalModels(ctx, store.AIPersonalModelsParams{WorkspaceID: ws.ID, UserID: user})
		if err != nil {
			return out, err
		}
		m := models{list: []*connection{}, defaults: map[string]uuid.UUID{}}
		for _, id := range ids {
			c, err := s.binding(ctx, s.Q, ws.ID, &id)
			if err != nil {
				return out, err
			}
			if c != nil {
				m.list = append(m.list, c)
			}
		}
		picks, err := s.Q.AIMemberFeatureModels(ctx, store.AIMemberFeatureModelsParams{WorkspaceID: ws.ID, UserID: *user})
		if err != nil {
			return out, err
		}
		for _, r := range picks {
			m.defaults[r.Feature] = r.ModelID
		}
		if out.Accounts, err = s.accountViews(ctx, s.Q, m); err != nil {
			return out, err
		}
		for i := range out.Accounts {
			out.Accounts[i].Primary = false
		}
		out.Choices = m.defaultViews()
		u, err := s.personalUsage(ctx, ws.ID, *user, start, out.CapUSD)
		if err != nil {
			return out, err
		}
		out.Usage.CostMicros, out.Usage.ReservedCostMicros, out.Usage.UncertainRequests, out.Usage.Requests, out.Usage.Share =
			u.costMicros, u.reservedCost, u.uncertain, u.requests, u.share
		switch {
		case u.reached:
			out.Usage.State = "reached"
		case u.notice:
			out.Usage.State = "notice"
		}
	}
	if p.Can(authz.IntegrationsManage) {
		if out.People, err = s.personalPeople(ctx, ws.ID, start); err != nil {
			return out, err
		}
	}
	return out, nil
}

// personalPeople lists who has personal accounts or used one this month.
func (s *Service) personalPeople(ctx context.Context, ws uuid.UUID, start time.Time) ([]AIPersonalPerson, error) {
	out := []AIPersonalPerson{}
	rows, err := s.Q.AIPersonalAccounts(ctx, ws)
	if err != nil {
		return nil, err
	}
	at := map[uuid.UUID]int{}
	for _, r := range rows {
		if r.UserID == nil {
			continue
		}
		i, ok := at[*r.UserID]
		if !ok {
			i = len(out)
			at[*r.UserID] = i
			out = append(out, AIPersonalPerson{UserID: *r.UserID, Name: r.UserName, Accounts: []AIPersonalAccount{}})
		}
		a := AIPersonalAccount{ID: r.ID, Provider: r.Provider, ProviderName: (&connection{store.AIBindingRow{Provider: r.Provider, BaseUrl: r.BaseUrl}}).providerName(),
			Status: "connected", Models: append([]string{}, r.Models...)}
		switch {
		case r.DisconnectedAt != nil:
			a.Status = "disconnected"
		case r.Status == "key_rejected":
			a.Status = "key_rejected"
		}
		out[i].Accounts = append(out[i].Accounts, a)
	}
	used, err := s.Q.AIPersonalUsageByUser(ctx, store.AIPersonalUsageByUserParams{WorkspaceID: ws, Since: start})
	if err != nil {
		return nil, err
	}
	for _, u := range used {
		i, ok := at[*u.UserID]
		if !ok {
			i = len(out)
			at[*u.UserID] = i
			out = append(out, AIPersonalPerson{UserID: *u.UserID, Name: s.userName(ctx, u.UserID), Accounts: []AIPersonalAccount{}})
		}
		out[i].CostMicros, out[i].Requests = u.CostMicros, u.Requests
	}
	return out, nil
}

// SetPersonalKeys turns personal keys on or off and sets each person's
// monthly cap. Turning them off keeps the accounts; their requests are
// refused until they are turned on again.
func (s *Service) SetPersonalKeys(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIPersonalKeysInput) (AISettings, error) {
	if err := validCap(ctx, "usd", in.CapUSD, 0); err != nil {
		return AISettings{}, err
	}
	capUSD := math.Round(in.CapUSD*100) / 100
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, authz.IntegrationsManage); err != nil {
			return err
		}
		row, err := lockSettings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		switched, capChanged := in.Enabled != row.PersonalKeys, capUSD != row.PersonalCapUsd
		if !switched && !capChanged {
			return nil
		}
		by, now := actorID(p), s.now()
		params := store.AIWorkspacePersonalSaveParams{WorkspaceID: ws.ID, PersonalKeys: in.Enabled, PersonalCapUsd: capUSD,
			PersonalCapSetBy: row.PersonalCapSetBy, PersonalCapSetAt: row.PersonalCapSetAt, UpdatedBy: by, UpdatedAt: now}
		if capChanged {
			params.PersonalCapSetBy, params.PersonalCapSetAt = by, &now
		}
		if err := q.AIWorkspacePersonalSave(ctx, params); err != nil {
			return err
		}
		if switched {
			action, summary := "ai_personal_keys.disabled", "personal keys turned off · the accounts are kept"
			if in.Enabled {
				action, summary = "ai_personal_keys.enabled", "personal keys turned on"
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "ai_assistant", TargetID: ws.ID.String(), TargetName: "Personal keys",
				Before: map[string]any{"enabled": row.PersonalKeys}, After: map[string]any{"enabled": in.Enabled}, Metadata: map[string]any{"summary": summary}}); err != nil {
				return err
			}
		}
		if capChanged {
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_personal_keys.cap_updated", TargetType: "ai_assistant", TargetID: ws.ID.String(), TargetName: "Personal keys",
				Before: map[string]any{"capUsd": row.PersonalCapUsd}, After: map[string]any{"capUsd": capUSD},
				Metadata: map[string]any{"summary": "personal cap " + usdText(row.PersonalCapUsd) + " → " + usdText(capUSD) + " a month per person"}})
		}
		return nil
	})
	if err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

// AddPersonalAccount adds one of the caller's own accounts with its first
// model, after the fixed test prompt passes (in the caller's own budget).
func (s *Service) AddPersonalAccount(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIAccountInput) (AIPersonalSettings, error) {
	if err := s.addAccount(ctx, ws, p, in, actorID(p)); err != nil {
		return AIPersonalSettings{}, err
	}
	return s.Personal(ctx, ws, p)
}

// RemovePersonalAccount removes one of the caller's own accounts, its key
// and its models; refused while the caller picked one of them for a
// feature.
func (s *Service) RemovePersonalAccount(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID) (AIPersonalSettings, error) {
	user := actorID(p)
	if user == nil {
		return AIPersonalSettings{}, api.NotFound("AI account")
	}
	err := s.removeAccount(ctx, ws, p, account, authz.AIUse, func(c *connection) bool { return c.ownedBy(user) })
	if err != nil {
		return AIPersonalSettings{}, err
	}
	return s.Personal(ctx, ws, p)
}

// RemoveMemberAccount removes a member's personal account, its key and its
// models, for integration managers (offboarding). The member's picks of its
// models go with it.
func (s *Service) RemoveMemberAccount(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID) (AISettings, error) {
	if err := s.removeAccount(ctx, ws, p, account, authz.IntegrationsManage, (*connection).personal); err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

// ReplacePersonalKey tests a new key for one of the caller's own accounts
// and saves it; an account whose key was removed is connected again.
func (s *Service) ReplacePersonalKey(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID, in AIKeyInput) (AIPersonalSettings, AITestResult, error) {
	user := actorID(p)
	if user == nil {
		return AIPersonalSettings{}, AITestResult{}, api.NotFound("AI account")
	}
	test, err := s.replaceAccountKey(ctx, ws, p, account, in, user)
	if err != nil {
		return AIPersonalSettings{}, test, err
	}
	out, err := s.Personal(ctx, ws, p)
	return out, test, err
}

// AddPersonalModel adds a model to one of the caller's own accounts.
func (s *Service) AddPersonalModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID, in AIAccountModelInput) (AIPersonalSettings, error) {
	user := actorID(p)
	if user == nil {
		return AIPersonalSettings{}, api.NotFound("AI account")
	}
	if err := s.addModel(ctx, ws, p, account, in, user); err != nil {
		return AIPersonalSettings{}, err
	}
	return s.Personal(ctx, ws, p)
}

// SetPersonalModelPrices changes the prices of one of the caller's models.
func (s *Service) SetPersonalModelPrices(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID, in AIPricesInput) (AIPersonalSettings, error) {
	user := actorID(p)
	if user == nil {
		return AIPersonalSettings{}, api.NotFound("AI model")
	}
	if err := s.setModelPrices(ctx, ws, p, account, model, in, user); err != nil {
		return AIPersonalSettings{}, err
	}
	return s.Personal(ctx, ws, p)
}

// RemovePersonalModel removes one of the caller's models; refused while
// they picked it for a feature.
func (s *Service) RemovePersonalModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID) (AIPersonalSettings, error) {
	user := actorID(p)
	if user == nil {
		return AIPersonalSettings{}, api.NotFound("AI model")
	}
	if err := s.removeModel(ctx, ws, p, account, model, user); err != nil {
		return AIPersonalSettings{}, err
	}
	return s.Personal(ctx, ws, p)
}

// personalModel is one of the caller's own models: 404 for anyone else's.
func (s *Service) personalModel(ctx context.Context, p *authz.Principal, ws uuid.UUID, account, model uuid.UUID) (*connection, error) {
	user := actorID(p)
	if user == nil {
		return nil, api.NotFound("AI model")
	}
	return s.accountsModel(ctx, s.Q, ws, account, model, user)
}

// TestPersonalModel runs Test connection on one of the caller's models.
func (s *Service) TestPersonalModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID) (AITestResult, error) {
	c, err := s.personalModel(ctx, p, ws.ID, account, model)
	if err != nil {
		return AITestResult{}, err
	}
	return s.testSaved(ctx, ws, p, c)
}

// TestPersonalModelTools runs the tool-calling check on one of the caller's
// models: the agent and the assistant use a personal model only once it
// passed.
func (s *Service) TestPersonalModelTools(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID) (AIAgentCapability, error) {
	c, err := s.personalModel(ctx, p, ws.ID, account, model)
	if err != nil {
		return AIAgentCapability{}, err
	}
	return s.testTools(ctx, ws, p, c)
}

// SetPersonalFeatureModel picks one of the caller's own models for their
// own requests of a feature, with the rules of the workspace's defaults.
// Requests already prepared with another model aren't sent; the caller's
// agent runs keep the model they began with.
func (s *Service) SetPersonalFeatureModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, feature string, in AIFeatureModelInput) (AIPersonalSettings, error) {
	if !slices.Contains(defaultFeatures, feature) {
		return AIPersonalSettings{}, invalid(ctx, "path.feature", mUnknownFeature, nil)
	}
	user := actorID(p)
	if user == nil {
		return AIPersonalSettings{}, api.NotFound("AI model")
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, authz.AIUse); err != nil {
			return err
		}
		settings, err := lockSettings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		c, err := s.binding(ctx, q, ws.ID, &in.ModelID)
		if err != nil {
			return err
		}
		if c == nil || !c.ownedBy(user) {
			return api.NotFound("AI model")
		}
		if !settings.PersonalKeys {
			return personalKeysOff(ctx)
		}
		if err := suits(ctx, q, c, feature); err != nil {
			return err
		}
		before := map[string]any{"feature": feature}
		if prev, err := q.AIMemberFeatureModel(ctx, store.AIMemberFeatureModelParams{WorkspaceID: ws.ID, UserID: *user, Feature: feature}); err == nil {
			before["modelId"] = prev
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := q.AIMemberFeatureModelSet(ctx, store.AIMemberFeatureModelSetParams{WorkspaceID: ws.ID, UserID: *user, Feature: feature, ModelID: c.ModelID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_feature.model_changed", TargetType: "ai_assistant", TargetID: feature,
			TargetName: c.modelName(), Before: before, After: map[string]any{"feature": feature, "modelId": c.ModelID},
			Metadata: map[string]any{"personal": true, "summary": feature + " → own model " + c.Model}})
	})
	if err != nil {
		return AIPersonalSettings{}, err
	}
	return s.Personal(ctx, ws, p)
}

// ClearPersonalFeatureModel goes back to the workspace's model for the
// caller's own requests of a feature.
func (s *Service) ClearPersonalFeatureModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, feature string) (AIPersonalSettings, error) {
	if !slices.Contains(defaultFeatures, feature) {
		return AIPersonalSettings{}, invalid(ctx, "path.feature", mUnknownFeature, nil)
	}
	user := actorID(p)
	if user == nil {
		return AIPersonalSettings{}, api.NotFound("AI model")
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, authz.AIUse); err != nil {
			return err
		}
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return err
		}
		prev, err := q.AIMemberFeatureModel(ctx, store.AIMemberFeatureModelParams{WorkspaceID: ws.ID, UserID: *user, Feature: feature})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := q.AIMemberFeatureModelClear(ctx, store.AIMemberFeatureModelClearParams{WorkspaceID: ws.ID, UserID: *user, Feature: feature}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_feature.model_changed", TargetType: "ai_assistant", TargetID: feature,
			Before: map[string]any{"feature": feature, "modelId": prev}, After: map[string]any{"feature": feature},
			Metadata: map[string]any{"personal": true, "summary": feature + " → the workspace's model"}})
	})
	if err != nil {
		return AIPersonalSettings{}, err
	}
	return s.Personal(ctx, ws, p)
}
