package ai

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// A workspace's model accounts, their models and the model each feature
// uses (assistant brief D3). The single-model operations (Connect, Change
// model, Replace key…) act on the first account's first model.

// AIAccountView is a model account with its models. The key is never
// returned: only whether one is set, and when.
type AIAccountView struct {
	ID           uuid.UUID `json:"id"`
	Provider     string    `json:"provider" enum:"anthropic,openai,google,azure,compatible"`
	ProviderName string    `json:"providerName"`
	// Preset is the hosted OpenAI-compatible service the address belongs to.
	Preset     string    `json:"preset,omitempty"`
	Label      string    `json:"label,omitempty"`
	BaseURL    string    `json:"baseUrl,omitempty"`
	APIVersion string    `json:"apiVersion,omitempty"`
	Key        AIKeyView `json:"key"`
	// Status: connected, key_rejected (the provider rejected the saved key)
	// or disconnected (the key was removed).
	Status       string      `json:"status" enum:"connected,key_rejected,disconnected"`
	StatusDetail string      `json:"statusDetail,omitempty"`
	StatusAt     *time.Time  `json:"statusAt,omitempty"`
	LastTest     *AITestView `json:"lastTest,omitempty"`
	ConnectedAt  time.Time   `json:"connectedAt"`
	ConnectedBy  string      `json:"connectedBy,omitempty"`
	// PrivateNetwork: the server's address is a private one.
	PrivateNetwork bool `json:"privateNetwork"`
	Demo           bool `json:"demo"`
	// Primary is the first account: the one the single-model settings show.
	Primary bool          `json:"primary"`
	Models  []AIModelView `json:"models"`
}

// AIModelView is a model of an account, with its prices and what it can do.
type AIModelView struct {
	ID         uuid.UUID    `json:"id"`
	Model      string       `json:"model"`
	ModelName  string       `json:"modelName"`
	Deployment string       `json:"deployment,omitempty"`
	ImageInput bool         `json:"imageInput"`
	Prices     AIPricesView `json:"prices"`
	// Tools is the tool-calling check the agent and the assistant need.
	Tools AIAgentCapability `json:"tools"`
	// Features are the features this model is the default of.
	Features []string `json:"features" enum:"assistant,agent,subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary,brand_voice,draft"`
}

// AIFeatureDefault is the model a feature uses.
type AIFeatureDefault struct {
	Feature string    `json:"feature" enum:"assistant,agent,subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary,brand_voice,draft"`
	ModelID uuid.UUID `json:"modelId"`
}

// models are the workspace's models with their accounts (the first
// account's first model first) and the model each feature uses.
type models struct {
	list     []*connection
	defaults map[string]uuid.UUID
}

func (s *Service) models(ctx context.Context, q *store.Queries, ws uuid.UUID) (models, error) {
	out := models{list: []*connection{}, defaults: map[string]uuid.UUID{}}
	ids, err := q.AIWorkspaceModels(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, id := range ids {
		c, err := s.binding(ctx, q, ws, &id)
		if err != nil {
			return out, err
		}
		if c != nil {
			out.list = append(out.list, c)
		}
	}
	rows, err := q.AIFeatureModels(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.defaults[r.Feature] = r.ModelID
	}
	return out, nil
}

// of returns the model a feature uses, nil when it has none.
func (m models) of(feature string) *connection {
	id, ok := m.defaults[feature]
	if !ok {
		return nil
	}
	for _, c := range m.list {
		if c.ModelID == id {
			return c
		}
	}
	return nil
}

func (m models) defaultViews() []AIFeatureDefault {
	out := []AIFeatureDefault{}
	for _, f := range defaultFeatures {
		if id, ok := m.defaults[f]; ok {
			out = append(out, AIFeatureDefault{Feature: f, ModelID: id})
		}
	}
	return out
}

// accountViews lists the accounts, each with its models.
func (s *Service) accountViews(ctx context.Context, q *store.Queries, m models) ([]AIAccountView, error) {
	out := []AIAccountView{}
	for _, c := range m.list {
		if len(out) == 0 || out[len(out)-1].ID != c.AccountID {
			cv := s.connectionView(ctx, c)
			v := AIAccountView{ID: c.AccountID, Provider: cv.Provider, ProviderName: cv.ProviderName, Preset: cv.Preset, Label: cv.Label, BaseURL: cv.BaseURL,
				APIVersion: cv.APIVersion, Key: cv.Key, Status: cv.Status, StatusDetail: cv.StatusDetail, StatusAt: cv.StatusAt, LastTest: cv.LastTest,
				ConnectedAt: cv.ConnectedAt, ConnectedBy: cv.ConnectedBy, PrivateNetwork: cv.PrivateNetwork, Demo: cv.Demo, Primary: len(out) == 0, Models: []AIModelView{}}
			if !c.connected() {
				v.Status = "disconnected"
			}
			out = append(out, v)
		}
		tools, err := agentCapability(ctx, q, c)
		if err != nil {
			return nil, err
		}
		mv := AIModelView{ID: c.ModelID, Model: c.Model, ModelName: c.modelName(), Deployment: c.Deployment, ImageInput: c.ImageInput,
			Prices: AIPricesView{Input: c.InputPrice, Output: c.OutputPrice, Known: c.PriceKnown || c.Demo, From: c.PricesFrom}, Tools: tools, Features: []string{}}
		for _, f := range defaultFeatures {
			if id, ok := m.defaults[f]; ok && id == c.ModelID {
				mv.Features = append(mv.Features, f)
			}
		}
		out[len(out)-1].Models = append(out[len(out)-1].Models, mv)
	}
	return out, nil
}

// accountModel returns an account's first model, nil when the workspace has
// no such account of owner (nil: a workspace account). Someone else's
// personal account is as absent as one that doesn't exist.
func (s *Service) accountModel(ctx context.Context, q *store.Queries, ws, account uuid.UUID, owner *uuid.UUID) (*connection, error) {
	id, err := q.AIAccountFirstModel(ctx, store.AIAccountFirstModelParams{WorkspaceID: ws, AccountID: account})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c, err := s.binding(ctx, q, ws, &id)
	if err != nil || c == nil || !c.ownedBy(owner) {
		return nil, err
	}
	return c, nil
}

// accountsModel returns a model of an account of owner (nil: a workspace
// account): 404 when the workspace has no such model on that account.
func (s *Service) accountsModel(ctx context.Context, q *store.Queries, ws, account, model uuid.UUID, owner *uuid.UUID) (*connection, error) {
	c, err := s.binding(ctx, q, ws, &model)
	if err != nil {
		return nil, err
	}
	if c == nil || c.AccountID != account || !c.ownedBy(owner) {
		return nil, api.NotFound("AI model")
	}
	return c, nil
}

// modelExists is a second model of the same name (and Azure deployment) on
// an account: a conflict to resolve, not a server error.
func modelExists(ctx context.Context, err error, model string) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "ai_models_account_id_model_deployment_key" {
		return problem(ctx, http.StatusConflict, codeModelExists, mModelExists, i18n.Args{"model": model})
	}
	return err
}

// inUse refuses to remove what features still use, naming them.
func inUse(ctx context.Context, m i18n.Msg, features []string) error {
	l := i18n.LangFrom(ctx)
	names := make([]string, 0, len(features))
	for _, f := range features {
		names = append(names, mFeatureName.Text(l, i18n.Args{"feature": f}))
	}
	return problem(ctx, http.StatusConflict, codeModelInUse, m, i18n.Args{"features": i18n.AndList(l.Locale, names), "count": len(names)}).
		WithMeta("features", features)
}

// modelAudit is what the audit entry keeps of a model.
func modelAudit(c store.AIBindingRow) map[string]any {
	m := map[string]any{"account": c.AccountID, "model": c.Model, "inputPrice": c.InputPrice, "outputPrice": c.OutputPrice}
	if c.Deployment != "" {
		m["deployment"] = c.Deployment
	}
	return m
}

// accountAudit is what the audit entry keeps of an account: never the key.
func accountAudit(c store.AIBindingRow) map[string]any {
	m := map[string]any{"provider": c.Provider, "keySet": len(c.KeySealed) > 0}
	if c.BaseUrl != "" {
		m["baseUrl"] = c.BaseUrl
	}
	if c.Label != "" {
		m["label"] = c.Label
	}
	return m
}

// AIAccountInput adds a model account with its first model.
type AIAccountInput struct {
	Provider string `json:"provider" enum:"anthropic,openai,google,azure,compatible"`
	// Label names a server the workspace runs ("Ollama on the office network").
	Label string `json:"label,omitempty" maxLength:"80"`
	// BaseURL: the Azure endpoint, or a compatible server's address with /v1.
	BaseURL    string `json:"baseUrl,omitempty" maxLength:"500"`
	Deployment string `json:"deployment,omitempty" maxLength:"120"`
	APIVersion string `json:"apiVersion,omitempty" maxLength:"40"`
	// APIKey is write-only; optional for a server on the workspace's network.
	APIKey    string `json:"apiKey,omitempty" maxLength:"500"`
	Model     string `json:"model" minLength:"1" maxLength:"120"`
	ModelName string `json:"modelName,omitempty" maxLength:"120"`
	// Prices in dollars per million tokens; empty takes the price list's.
	InputPrice  *float64 `json:"inputPrice,omitempty" minimum:"0"`
	OutputPrice *float64 `json:"outputPrice,omitempty" minimum:"0"`
}

// AddAccount adds a model account, after the fixed test prompt passes with
// its first model. A workspace's first model is every feature's default.
func (s *Service) AddAccount(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIAccountInput) (AISettings, error) {
	if err := s.addAccount(ctx, ws, p, in, nil); err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

// addAccount adds a workspace account (owner nil) or a person's own.
func (s *Service) addAccount(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIAccountInput, owner *uuid.UUID) error {
	if owner != nil {
		if on, err := s.personalKeysOn(ctx, ws.ID); err != nil || !on {
			return errOr(err, personalKeysOff(ctx))
		}
	}
	c := candidate{provider: in.Provider, label: strings.TrimSpace(in.Label), baseURL: in.BaseURL, deployment: strings.TrimSpace(in.Deployment), apiVersion: strings.TrimSpace(in.APIVersion),
		key: strings.TrimSpace(in.APIKey), model: strings.TrimSpace(in.Model)}
	if err := s.validCandidate(ctx, &c); err != nil {
		return err
	}
	if name, need := keyNeeded(c.provider, c.baseURL); need && c.key == "" {
		return invalid(ctx, "body.apiKey", mKeyNeeded, i18n.Args{"provider": name})
	}
	input, output, from := pricesOf(c.provider, c.model, in.InputPrice, in.OutputPrice)
	if err := validPrices(ctx, input, output); err != nil {
		return err
	}
	name := (&connection{store.AIBindingRow{Provider: c.provider, BaseUrl: c.baseURL}}).providerName()
	priced, err := s.testPricing(ctx, ws, c, in.InputPrice, in.OutputPrice)
	if err != nil {
		return err
	}
	// A personal key's test is in its owner's budget.
	priced.AccountUserID = owner
	res, err := s.test(ctx, ws, p, c, priced)
	if err != nil {
		return err
	}
	if !res.OK {
		s.explain(ctx, &res, name, owner != nil)
		return api.Err(http.StatusUnprocessableEntity, "ai_test_failed", res.ErrorMessage).WithMeta("test", res)
	}
	now := s.now()
	actor := actorID(p)
	row := store.AIBindingRow{ModelID: uuid.Must(uuid.NewV7()), AccountID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Provider: c.provider, Label: c.label,
		BaseUrl: c.baseURL, Deployment: c.deployment, ApiVersion: c.apiVersion, Model: c.model, ModelName: strings.TrimSpace(in.ModelName),
		ImageInput: llm.ImageInput(c.provider, c.model), ConnectedBy: actor, ConnectedAt: now, Status: "ok", LastTestAt: &now, LastTestOk: ptr(true),
		LastTestMs: ptr(int32(res.Ms)), InputPrice: input, OutputPrice: output, PricesFrom: from, PriceKnown: pricesKnown(c.provider, c.model, in.InputPrice, in.OutputPrice),
		AccountUserID: owner}
	if row.ModelName == "" {
		row.ModelName = llm.ModelName(c.provider, c.model)
	}
	if c.key != "" {
		if row.KeySealed, err = s.Box.SealString(c.key, keyAD(row.AccountID)); err != nil {
			return err
		}
		row.KeyAddedBy, row.KeyAddedAt = actor, &now
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, ownerPermission(owner)); err != nil {
			return err
		}
		settings, err := lockSettings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if owner != nil && !settings.PersonalKeys {
			return personalKeysOff(ctx)
		}
		if err := q.AIAccountSave(ctx, accountParams(row, actor, now)); err != nil {
			return err
		}
		if err := q.AIModelSave(ctx, modelParams(row, actor, now)); err != nil {
			return modelExists(ctx, err, row.Model)
		}
		// A personal model is never a workspace default.
		if owner == nil {
			if err := q.AIFeatureModelsFill(ctx, store.AIFeatureModelsFillParams{WorkspaceID: ws.ID, ModelID: row.ModelID, Features: defaultFeatures}); err != nil {
				return err
			}
		}
		key := keyNone
		if c.key != "" {
			key = keyAdded
		}
		after := accountAudit(row)
		after["models"] = []any{modelAudit(row)}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_account.add", TargetType: "ai_assistant", TargetID: row.AccountID.String(),
			TargetName: name, After: after, Metadata: personalMeta(owner, map[string]any{"test": map[string]any{"ok": true, "ms": res.Ms}, "summary": "model " + row.Model + " · " + key})})
	})
}

// RemoveAccount removes an account, its key and its models. The activity
// log keeps their requests. An account a feature uses stays.
func (s *Service) RemoveAccount(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID) (AISettings, error) {
	if err := s.removeAccount(ctx, ws, p, account, authz.IntegrationsManage, func(c *connection) bool { return !c.personal() }); err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

// removeAccount removes an account mine accepts: a workspace account, the
// caller's own, or for managers any person's own (offboarding).
func (s *Service) removeAccount(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID, permission authz.Permission, mine func(*connection) bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, permission); err != nil {
			return err
		}
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return err
		}
		id, err := q.AIAccountFirstModel(ctx, store.AIAccountFirstModelParams{WorkspaceID: ws.ID, AccountID: account})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("AI account")
		}
		if err != nil {
			return err
		}
		c, err := s.binding(ctx, q, ws.ID, &id)
		if err != nil {
			return err
		}
		if c == nil || !mine(c) {
			return api.NotFound("AI account")
		}
		used, err := q.AIFeaturesOfAccount(ctx, store.AIFeaturesOfAccountParams{WorkspaceID: ws.ID, AccountID: account})
		if err != nil {
			return err
		}
		// Its owner picked its models for their own requests: theirs to
		// change first (a manager's removal takes the picks with it).
		if c.personal() && c.ownedBy(actorID(p)) {
			if used, err = s.pickedFeatures(ctx, q, ws.ID, *c.AccountUserID, func(m *connection) bool { return m.AccountID == account }); err != nil {
				return err
			}
		}
		if len(used) > 0 {
			return inUse(ctx, mAccountInUse, used)
		}
		if _, err := q.AIAccountDelete(ctx, store.AIAccountDeleteParams{WorkspaceID: ws.ID, ID: account}); err != nil {
			return err
		}
		meta := personalMeta(c.AccountUserID, map[string]any{"summary": "account removed · key deleted"})
		if c.personal() {
			meta["owner"] = c.AccountUserID.String()
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_account.remove", TargetType: "ai_assistant", TargetID: account.String(),
			TargetName: c.providerName(), Before: accountAudit(c.AIBindingRow), Metadata: meta})
	})
}

// ReplaceAccountKey tests a new key with the account's first model and
// saves it.
func (s *Service) ReplaceAccountKey(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID, in AIKeyInput) (AISettings, AITestResult, error) {
	test, err := s.replaceAccountKey(ctx, ws, p, account, in, nil)
	if err != nil {
		return AISettings{}, test, err
	}
	out, err := s.Settings(ctx, ws, p)
	return out, test, err
}

func (s *Service) replaceAccountKey(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID, in AIKeyInput, owner *uuid.UUID) (AITestResult, error) {
	cur, err := s.accountModel(ctx, s.Q, ws.ID, account, owner)
	if err != nil {
		return AITestResult{}, err
	}
	if cur == nil {
		return AITestResult{}, api.NotFound("AI account")
	}
	return s.replaceKey(ctx, ws, p, cur, in, "ai_account.key_replaced", account.String())
}

// AIAccountModelInput adds a model to an account.
type AIAccountModelInput struct {
	Model     string `json:"model" minLength:"1" maxLength:"120"`
	ModelName string `json:"modelName,omitempty" maxLength:"120"`
	// Deployment: Azure OpenAI's deployment of the model (the model's name
	// when empty).
	Deployment string `json:"deployment,omitempty" maxLength:"120"`
	// Prices follow the price list for a known model unless given.
	InputPrice  *float64 `json:"inputPrice,omitempty" minimum:"0"`
	OutputPrice *float64 `json:"outputPrice,omitempty" minimum:"0"`
}

// AddModel adds a model to a connected account.
func (s *Service) AddModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID, in AIAccountModelInput) (AISettings, error) {
	if err := s.addModel(ctx, ws, p, account, in, nil); err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

func (s *Service) addModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, account uuid.UUID, in AIAccountModelInput, owner *uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, ownerPermission(owner)); err != nil {
			return err
		}
		settings, err := lockSettings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		c, err := s.accountModel(ctx, q, ws.ID, account, owner)
		if err != nil {
			return err
		}
		if c == nil {
			return api.NotFound("AI account")
		}
		if owner != nil && !settings.PersonalKeys {
			return personalKeysOff(ctx)
		}
		if !c.connected() {
			return problem(ctx, http.StatusConflict, codeNotConnected, mNotConnectedYet, nil)
		}
		model := strings.TrimSpace(in.Model)
		input, output, from := pricesOf(c.Provider, model, in.InputPrice, in.OutputPrice)
		if err := validPrices(ctx, input, output); err != nil {
			return err
		}
		row := c.AIBindingRow
		row.ModelID, row.Model, row.ModelName, row.Deployment = uuid.Must(uuid.NewV7()), model, strings.TrimSpace(in.ModelName), ""
		if row.ModelName == "" {
			row.ModelName = llm.ModelName(c.Provider, model)
		}
		if c.Provider == llm.Azure {
			row.Deployment = strings.TrimSpace(in.Deployment)
			if row.Deployment == "" {
				row.Deployment = model
			}
		}
		row.ImageInput, row.InputPrice, row.OutputPrice, row.PricesFrom = llm.ImageInput(c.Provider, model), input, output, from
		row.PriceKnown = pricesKnown(c.Provider, model, in.InputPrice, in.OutputPrice)
		if err := q.AIModelSave(ctx, modelParams(row, actorID(p), s.now())); err != nil {
			return modelExists(ctx, err, model)
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_model.add", TargetType: "ai_assistant", TargetID: row.ModelID.String(),
			TargetName: row.ModelName, After: modelAudit(row), Metadata: personalMeta(owner, map[string]any{"summary": "model " + model + " · prices " + pricesText(input, output) + " per million tokens"})})
	})
}

// SetModelPrices changes the prices a model's requests are estimated with.
func (s *Service) SetModelPrices(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID, in AIPricesInput) (AISettings, error) {
	if err := s.setModelPrices(ctx, ws, p, account, model, in, nil); err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

func (s *Service) setModelPrices(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID, in AIPricesInput, owner *uuid.UUID) error {
	if err := validPrices(ctx, in.Input, in.Output); err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, ownerPermission(owner)); err != nil {
			return err
		}
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return err
		}
		c, err := s.accountsModel(ctx, q, ws.ID, account, model, owner)
		if err != nil {
			return err
		}
		row := c.AIBindingRow
		in1, out1 := in.Input, in.Output
		row.InputPrice, row.OutputPrice, row.PricesFrom = pricesOf(row.Provider, row.Model, &in1, &out1)
		row.PriceKnown = true
		if err := q.AIModelSave(ctx, modelParams(row, actorID(p), s.now())); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_model.prices_updated", TargetType: "ai_assistant", TargetID: model.String(),
			TargetName: c.modelName(), Before: map[string]any{"inputPrice": c.InputPrice, "outputPrice": c.OutputPrice},
			After: map[string]any{"inputPrice": in.Input, "outputPrice": in.Output}, Metadata: personalMeta(owner, map[string]any{"summary": pricesLine(c.InputPrice, c.OutputPrice, in.Input, in.Output)})})
	})
}

// RemoveModel removes a model no feature uses. An account keeps at least
// one model: removing the last one is removing the account.
func (s *Service) RemoveModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID) (AISettings, error) {
	if err := s.removeModel(ctx, ws, p, account, model, nil); err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

func (s *Service) removeModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID, owner *uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, ownerPermission(owner)); err != nil {
			return err
		}
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return err
		}
		c, err := s.accountsModel(ctx, q, ws.ID, account, model, owner)
		if err != nil {
			return err
		}
		used, err := q.AIFeaturesOfModel(ctx, store.AIFeaturesOfModelParams{WorkspaceID: ws.ID, ModelID: model})
		if err != nil {
			return err
		}
		if owner != nil {
			if used, err = s.pickedFeatures(ctx, q, ws.ID, *owner, func(m *connection) bool { return m.ModelID == model }); err != nil {
				return err
			}
		}
		if len(used) > 0 {
			return inUse(ctx, mModelInUse, used)
		}
		n, err := q.AIAccountModelCount(ctx, store.AIAccountModelCountParams{WorkspaceID: ws.ID, AccountID: account})
		if err != nil {
			return err
		}
		if n <= 1 {
			return problem(ctx, http.StatusConflict, codeLastModel, mLastModel, nil)
		}
		if _, err := q.AIModelDelete(ctx, store.AIModelDeleteParams{WorkspaceID: ws.ID, ID: model}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_model.remove", TargetType: "ai_assistant", TargetID: model.String(),
			TargetName: c.modelName(), Before: modelAudit(c.AIBindingRow), Metadata: personalMeta(owner, map[string]any{"summary": "model " + c.Model + " removed"})})
	})
}

// AIFeatureModelInput picks the model a feature uses.
type AIFeatureModelInput struct {
	ModelID uuid.UUID `json:"modelId" format:"uuid"`
}

// SetFeatureModel picks the model a feature uses: a model of a connected
// account whose key works, that reads images for alt text, and that passed
// the tool-calling check for the agent and the assistant. Requests already
// prepared with the previous model are refused, not moved.
func (s *Service) SetFeatureModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, feature string, in AIFeatureModelInput) (AISettings, error) {
	if !slices.Contains(defaultFeatures, feature) {
		return AISettings{}, invalid(ctx, "path.feature", mUnknownFeature, nil)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := draftActor(ctx, q, ws, p, false, authz.IntegrationsManage); err != nil {
			return err
		}
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return err
		}
		c, err := s.binding(ctx, q, ws.ID, &in.ModelID)
		if err != nil {
			return err
		}
		if c == nil {
			return api.NotFound("AI model")
		}
		// A person's own key answers only them: the workspace's features
		// never use it.
		if c.personal() {
			return problem(ctx, http.StatusConflict, codePersonalModel, mPersonalModel, nil)
		}
		if err := suits(ctx, q, c, feature); err != nil {
			return err
		}
		before := map[string]any{"feature": feature}
		if prev, err := q.AIFeatureModel(ctx, store.AIFeatureModelParams{WorkspaceID: ws.ID, Feature: feature}); err == nil {
			before["modelId"] = prev
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := q.AIFeatureModelSet(ctx, store.AIFeatureModelSetParams{WorkspaceID: ws.ID, Feature: feature, ModelID: c.ModelID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_feature.model_changed", TargetType: "ai_assistant", TargetID: feature,
			TargetName: c.modelName(), Before: before, After: map[string]any{"feature": feature, "modelId": c.ModelID},
			Metadata: map[string]any{"summary": feature + " → model " + c.Model}})
	})
	if err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

// suits checks that a model can be a feature's: its account is connected
// and its key works, it reads images for alt text, and it passed the
// tool-calling check for the agent and the assistant.
func suits(ctx context.Context, q *store.Queries, c *connection, feature string) error {
	switch {
	case !c.connected():
		return problem(ctx, http.StatusConflict, codeAccountDisconnected, mAccountDisconnected, nil)
	case c.Status == "key_rejected":
		return problem(ctx, http.StatusConflict, codeKeyRejected, mKeyRejectedModel, i18n.Args{"provider": c.providerName()})
	case feature == FeatureAltText && !c.ImageInput:
		return problem(ctx, http.StatusConflict, codeNoImageInput, mDefaultNoImages, nil)
	}
	if feature == FeatureAgent || feature == FeatureAssistant {
		tools, err := agentCapability(ctx, q, c)
		if err != nil {
			return err
		}
		if tools.State != "supported" {
			return problem(ctx, http.StatusConflict, codeToolsUnchecked, mToolsUnchecked, nil)
		}
	}
	return nil
}

// TestModelTools runs the tool-calling check on one model of an account.
func (s *Service) TestModelTools(ctx context.Context, ws *store.Workspace, p *authz.Principal, account, model uuid.UUID) (AIAgentCapability, error) {
	c, err := s.accountsModel(ctx, s.Q, ws.ID, account, model, nil)
	if err != nil {
		return AIAgentCapability{}, err
	}
	return s.testTools(ctx, ws, p, c)
}
