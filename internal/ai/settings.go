package ai

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
)

// AIProviderEntry is a provider of the catalog.
type AIProviderEntry struct {
	Kind         string   `json:"kind" enum:"anthropic,openai,google,azure,compatible"`
	Name         string   `json:"name"`
	NeedsKey     bool     `json:"needsKey"`
	NeedsBaseURL bool     `json:"needsBaseUrl"`
	Capabilities []string `json:"capabilities" enum:"text,image_input,structured_output,own_network,data_region"`
	DefaultModel string   `json:"defaultModel,omitempty"`
	// Models are the price list's models of the provider.
	Models []AIModelOption `json:"models"`
	// Presets are hosted services that speak this provider's API, with their
	// address filled in (OpenAI-compatible only).
	Presets []AIProviderPreset `json:"presets"`
}

// AIProviderPreset is a hosted OpenAI-compatible service. It needs a key;
// its models come from the service and its prices from the person.
type AIProviderPreset struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}

// AIModelOption is a model to pick, with its prices from the price list.
type AIModelOption struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Known      bool    `json:"known" doc:"On ReputeCast's price list"`
	Input      float64 `json:"input" doc:"Dollars per million input tokens"`
	Output     float64 `json:"output" doc:"Dollars per million output tokens"`
	ImageInput bool    `json:"imageInput"`
}

// AIKeyView is the key, write-only: only whether it is set, and when.
type AIKeyView struct {
	Set     bool       `json:"set"`
	AddedAt *time.Time `json:"addedAt,omitempty"`
	AddedBy string     `json:"addedBy,omitempty"`
}

// AITestView is the last test of the connection.
type AITestView struct {
	At time.Time `json:"at"`
	OK bool      `json:"ok"`
	Ms int       `json:"ms"`
}

// AICapView is the monthly cap.
type AICapView struct {
	Kind   string     `json:"kind" enum:"usd,tokens,none"`
	USD    float64    `json:"usd"`
	Tokens int64      `json:"tokens"`
	SetBy  string     `json:"setBy,omitempty"`
	SetAt  *time.Time `json:"setAt,omitempty"`
}

// AIPricesView are the prices the cap is estimated with.
type AIPricesView struct {
	Input  float64 `json:"input" doc:"Dollars per million input tokens"`
	Output float64 `json:"output" doc:"Dollars per million output tokens"`
	Known  bool    `json:"known" doc:"Prices are configured, including an explicitly free model"`
	// From is the price list's date when the prices came from it.
	From *time.Time `json:"from,omitempty"`
}

// AIConnectionView is the workspace's model. The key is never returned.
type AIConnectionView struct {
	Provider     string    `json:"provider" enum:"anthropic,openai,google,azure,compatible"`
	ProviderName string    `json:"providerName"`
	Label        string    `json:"label,omitempty"`
	BaseURL      string    `json:"baseUrl,omitempty"`
	Deployment   string    `json:"deployment,omitempty"`
	APIVersion   string    `json:"apiVersion,omitempty"`
	Model        string    `json:"model"`
	ModelName    string    `json:"modelName"`
	ImageInput   bool      `json:"imageInput"`
	Key          AIKeyView `json:"key"`
	ConnectedAt  time.Time `json:"connectedAt"`
	ConnectedBy  string    `json:"connectedBy,omitempty"`
	// Status: connected, or key_rejected (the provider rejected the saved key).
	Status       string       `json:"status" enum:"connected,key_rejected"`
	StatusDetail string       `json:"statusDetail,omitempty"`
	StatusAt     *time.Time   `json:"statusAt,omitempty"`
	LastTest     *AITestView  `json:"lastTest,omitempty"`
	Cap          AICapView    `json:"cap"`
	Prices       AIPricesView `json:"prices"`
	// PrivateNetwork: the server's address is a private one ("Runs on your
	// own network").
	PrivateNetwork bool `json:"privateNetwork"`
	// Demo is the demo's sample connection (answers from recordings).
	Demo bool `json:"demo"`
	// Preset is the hosted OpenAI-compatible service the address belongs to.
	Preset string `json:"preset,omitempty"`
}

// AIUsageView is this month's use against the cap.
type AIUsageView struct {
	CostMicros         int64     `json:"costMicros"`
	ReservedCostMicros int64     `json:"reservedCostMicros"`
	ReservedTokens     int64     `json:"reservedTokens"`
	UncertainRequests  int64     `json:"uncertainRequests"`
	InputTokens        int64     `json:"inputTokens"`
	OutputTokens       int64     `json:"outputTokens"`
	Tokens             int64     `json:"tokens"`
	Requests           int64     `json:"requests"`
	Share              float64   `json:"share" doc:"Share of the cap used, 0 to 1 (0 without a cap)"`
	State              string    `json:"state" enum:"ok,notice,reached"`
	MonthStart         time.Time `json:"monthStart"`
	ResetsAt           time.Time `json:"resetsAt"`
}

// AIFeatureState is a feature's switch.
type AIFeatureState struct {
	Key string `json:"key" enum:"subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary,brand_voice,draft"`
	On  bool   `json:"on"`
	// Available is false for alt text with a model that doesn't read images.
	Available bool `json:"available"`
	// ModelID is the model the feature uses.
	ModelID *uuid.UUID `json:"modelId,omitempty" format:"uuid"`
}

// AIMemberRole is a member and their role.
type AIMemberRole struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// AISettings is Settings › Integrations › AI models.
type AISettings struct {
	Connected bool `json:"connected"`
	// Connection is the model, also while disconnected (Disconnected says
	// so): its model, cap and prices are kept for the next connection.
	Connection   *AIConnectionView `json:"connection,omitempty"`
	Disconnected bool              `json:"disconnected"`
	Catalog      []AIProviderEntry `json:"catalog"`
	Usage        AIUsageView       `json:"usage"`
	ByFeature    []AIUsageFeature  `json:"byFeature"`
	ByMember     []AIUsageMember   `json:"byMember"`
	Features     []AIFeatureState  `json:"features"`
	// CanManage: the caller holds integrations:manage; Helpers hold it.
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
	// AIUseRoles are the roles whose members hold ai:use; WithoutAIUse the
	// members who don't (the board's "Aisha Bello, a Viewer").
	AIUseRoles   []string       `json:"aiUseRoles"`
	WithoutAIUse []AIMemberRole `json:"withoutAiUse"`
	// PriceListDate is the date of ReputeCast's price list.
	PriceListDate time.Time `json:"priceListDate"`
	// AllowPrivate: this instance lets model servers resolve to private
	// addresses; InstanceAdmins can change that.
	AllowPrivate   bool     `json:"allowPrivate"`
	InstanceAdmins []string `json:"instanceAdmins"`
	// Accounts are the model accounts with their models, the first
	// (Connection's) first; Defaults the model each feature uses.
	Accounts []AIAccountView    `json:"accounts"`
	Defaults []AIFeatureDefault `json:"defaults"`
	// Personal is the personal-keys switch, each person's cap, the caller's
	// own accounts and, for managers, who has personal accounts.
	Personal AIPersonalSettings `json:"personal"`
	// Assistant is the assistant's switch, its limits per thread and the
	// turns answering at once.
	Assistant AIAssistantSettings `json:"assistant"`
}

func catalog() []AIProviderEntry {
	out := make([]AIProviderEntry, 0, len(llm.Providers))
	for _, p := range llm.Providers {
		e := AIProviderEntry{Kind: p.Kind, Name: p.Name, NeedsKey: p.NeedsKey, NeedsBaseURL: p.NeedsBaseURL, Capabilities: slices.Clone(p.Capabilities), DefaultModel: p.DefaultModel, Models: []AIModelOption{}, Presets: []AIProviderPreset{}}
		if p.Kind == llm.Compatible {
			for _, pr := range llm.Presets {
				e.Presets = append(e.Presets, AIProviderPreset{ID: pr.ID, Name: pr.Name, BaseURL: pr.BaseURL})
			}
		}
		for _, k := range llm.PriceList {
			if k.Provider == p.Kind {
				e.Models = append(e.Models, AIModelOption{ID: k.ID, Name: k.Name, Known: true, Input: k.Input, Output: k.Output, ImageInput: k.ImageInput})
			}
		}
		out = append(out, e)
	}
	return out
}

// userName returns a member's name.
func (s *Service) userName(ctx context.Context, id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	u, err := s.Q.GetUser(ctx, *id)
	if err != nil {
		return ""
	}
	if u.Name != "" {
		return u.Name
	}
	return u.Email
}

func (s *Service) connectionView(ctx context.Context, c *connection) *AIConnectionView {
	v := &AIConnectionView{Provider: c.Provider, ProviderName: c.providerName(), Label: c.Label, BaseURL: c.BaseUrl, Deployment: c.Deployment, APIVersion: c.ApiVersion,
		Model: c.Model, ModelName: c.modelName(), ImageInput: c.ImageInput, ConnectedAt: c.ConnectedAt, ConnectedBy: s.userName(ctx, c.ConnectedBy),
		Status: "connected", Cap: AICapView{Kind: c.CapKind, USD: c.CapUsd, Tokens: c.CapTokens, SetBy: s.userName(ctx, c.CapSetBy), SetAt: c.CapSetAt},
		Prices: AIPricesView{Input: c.InputPrice, Output: c.OutputPrice, Known: c.PriceKnown || c.Demo, From: c.PricesFrom}, Demo: c.Demo}
	if p, ok := c.preset(); ok {
		v.Preset = p.ID
	}
	if len(c.KeySealed) > 0 {
		v.Key = AIKeyView{Set: true, AddedAt: c.KeyAddedAt, AddedBy: s.userName(ctx, c.KeyAddedBy)}
	}
	if c.Status == "key_rejected" {
		v.Status, v.StatusDetail, v.StatusAt = "key_rejected", c.StatusDetail, c.StatusAt
	}
	if c.LastTestAt != nil && c.LastTestOk != nil {
		v.LastTest = &AITestView{At: *c.LastTestAt, OK: *c.LastTestOk}
		if c.LastTestMs != nil {
			v.LastTest.Ms = int(*c.LastTestMs)
		}
	}
	if c.BaseUrl != "" {
		if _, private := resolvePrivate(ctx, hostOf(c.BaseUrl)); private {
			v.PrivateNetwork = true
		}
	}
	return v
}

// Settings returns the AI assistant's page.
func (s *Service) Settings(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AISettings, error) {
	c, err := s.load(ctx, s.Q, ws.ID)
	if err != nil {
		return AISettings{}, err
	}
	out := AISettings{Catalog: catalog(), ByFeature: []AIUsageFeature{}, ByMember: []AIUsageMember{}, Features: []AIFeatureState{},
		CanManage: p.Can(authz.IntegrationsManage), Helpers: s.helpers(ctx, ws.ID), PriceListDate: llm.PriceListDate,
		AllowPrivate: s.policy().AllowPrivate, InstanceAdmins: s.instanceAdmins(ctx), AIUseRoles: []string{}, WithoutAIUse: []AIMemberRole{}}
	start, next := s.month(ws)
	out.Usage = AIUsageView{State: "ok", MonthStart: start, ResetsAt: next}
	if c != nil {
		out.Connected = c.connected()
		out.Disconnected = !c.connected()
		out.Connection = s.connectionView(ctx, c)
	}
	cap := c
	if cap == nil {
		cap = &connection{store.AIBindingRow{CapKind: "usd", CapUsd: 25}}
	}
	u, err := s.usage(ctx, ws, cap)
	if err != nil {
		return AISettings{}, err
	}
	out.Usage.CostMicros, out.Usage.InputTokens, out.Usage.OutputTokens, out.Usage.Tokens, out.Usage.Requests, out.Usage.Share =
		u.costMicros, u.inputTokens, u.outputTokens, u.tokens, u.requests, u.share
	out.Usage.ReservedCostMicros, out.Usage.ReservedTokens, out.Usage.UncertainRequests = u.reservedCost, u.reservedTokens, u.uncertain
	switch {
	case u.reached:
		out.Usage.State = "reached"
	case u.notice:
		out.Usage.State = "notice"
	}
	m, err := s.models(ctx, s.Q, ws.ID)
	if err != nil {
		return AISettings{}, err
	}
	if out.Accounts, err = s.accountViews(ctx, s.Q, m); err != nil {
		return AISettings{}, err
	}
	out.Defaults = m.defaultViews()
	for _, f := range Features {
		st := AIFeatureState{Key: f, On: true, Available: true}
		if c != nil {
			st.On = !slices.Contains(c.FeaturesOff, f)
		}
		// Alt text needs the model it uses to read images.
		if d := m.of(f); d != nil {
			st.ModelID = &d.ModelID
			if f == FeatureAltText && !d.ImageInput {
				st.On, st.Available = false, false
			}
		}
		out.Features = append(out.Features, st)
	}
	if out.ByFeature, err = s.byFeature(ctx, ws); err != nil {
		return AISettings{}, err
	}
	if out.ByMember, err = s.byMember(ctx, ws); err != nil {
		return AISettings{}, err
	}
	out.AIUseRoles, out.WithoutAIUse = s.aiUseHolders(ctx, ws)
	if out.Personal, err = s.Personal(ctx, ws, p); err != nil {
		return AISettings{}, err
	}
	if out.Assistant, err = s.AssistantSettings(ctx, s.Q, ws.ID); err != nil {
		return AISettings{}, err
	}
	return out, nil
}

// aiUseHolders names the roles that hold ai:use (in the roles' order) and
// the members who don't.
func (s *Service) aiUseHolders(ctx context.Context, ws *store.Workspace) ([]string, []AIMemberRole) {
	roles := []string{}
	without := []AIMemberRole{}
	rows, err := s.Q.AIMembersRoles(ctx, ws.ID)
	if err != nil {
		return roles, without
	}
	type held struct {
		name  string
		order int
	}
	seen := map[string]held{}
	rank := map[string]int{authz.RoleOwner: 0, authz.RoleAdmin: 1, authz.RoleEditor: 2, authz.RoleViewer: 4}
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(authz.AIUse) {
			order, ok := rank[r.RoleKey]
			if !ok {
				order = 3
			}
			seen[r.RoleName] = held{name: r.RoleName, order: order}
			continue
		}
		if len(without) < 3 {
			without = append(without, AIMemberRole{Name: r.Name, Role: r.RoleName})
		}
	}
	list := make([]held, 0, len(seen))
	for _, h := range seen {
		list = append(list, h)
	}
	slices.SortFunc(list, func(a, b held) int {
		if a.order != b.order {
			return a.order - b.order
		}
		return strings.Compare(a.name, b.name)
	})
	for _, h := range list {
		roles = append(roles, h.name)
	}
	return roles, without
}

// instanceAdmins are the people who can allow private addresses.
func (s *Service) instanceAdmins(ctx context.Context) []string {
	rows, err := s.Q.AIInstanceAdmins(ctx)
	if err != nil {
		return []string{}
	}
	return append(make([]string, 0, len(rows)), rows...)
}

// AIConnectInput connects a model.
type AIConnectInput struct {
	Provider string `json:"provider" enum:"anthropic,openai,google,azure,compatible"`
	// Label names a server the workspace runs ("Ollama on the office network").
	Label string `json:"label,omitempty" maxLength:"80"`
	// BaseURL: the Azure endpoint, or a compatible server's address with /v1.
	BaseURL    string `json:"baseUrl,omitempty" maxLength:"500"`
	Deployment string `json:"deployment,omitempty" maxLength:"120"`
	APIVersion string `json:"apiVersion,omitempty" maxLength:"40"`
	// APIKey is write-only; optional for compatible servers.
	APIKey    string  `json:"apiKey,omitempty" maxLength:"500"`
	Model     string  `json:"model" minLength:"1" maxLength:"120"`
	ModelName string  `json:"modelName,omitempty" maxLength:"120"`
	CapKind   string  `json:"capKind" enum:"usd,tokens,none"`
	CapUSD    float64 `json:"capUsd,omitempty" minimum:"0"`
	CapTokens int64   `json:"capTokens,omitempty" minimum:"0"`
	// Prices in dollars per million tokens; empty takes the price list's.
	InputPrice  *float64 `json:"inputPrice,omitempty" minimum:"0"`
	OutputPrice *float64 `json:"outputPrice,omitempty" minimum:"0"`
	// Features to switch on (all when empty).
	Features []string `json:"features,omitempty" maxItems:"8"`
}

// candidate is a connection being set up or changed.
type candidate struct {
	provider, label, baseURL, deployment, apiVersion, key, model string
}

// checkAddress validates a server's address against the SSRF guard: a
// private address works only when the instance allows private addresses.
func (s *Service) checkAddress(ctx context.Context, kind, raw string) (string, *api.Problem) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	example := "http://ollama.acme.internal:11434/v1"
	if kind == llm.Azure {
		example = "https://acme.openai.azure.com"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return "", invalid(ctx, "body.baseUrl", mBadAddress, i18n.Args{"example": example})
	}
	if kind == llm.Azure && u.Scheme != "https" {
		return "", invalid(ctx, "body.baseUrl", mBadAddress, i18n.Args{"example": example})
	}
	u.RawQuery, u.Fragment = "", ""
	clean := strings.TrimRight(u.String(), "/")
	host := u.Hostname()
	if s.policy().AllowPrivate {
		return clean, nil
	}
	if ip, private := resolvePrivate(ctx, host); private {
		if ip == host {
			return "", problem(ctx, http.StatusUnprocessableEntity, codePrivate, mPrivateHost, i18n.Args{"host": host}).WithMeta("host", host, "ip", ip)
		}
		return "", problem(ctx, http.StatusUnprocessableEntity, codePrivate, mPrivateAddress, i18n.Args{"host": host, "ip": ip}).WithMeta("host", host, "ip", ip)
	}
	if _, err := s.policy().ValidateURL(clean); err != nil {
		return "", problem(ctx, http.StatusUnprocessableEntity, codePrivate, mPrivateHost, i18n.Args{"host": host}).WithMeta("host", host)
	}
	return clean, nil
}

func (s *Service) validCandidate(ctx context.Context, c *candidate) error {
	if !llm.Valid(c.provider) {
		return invalid(ctx, "body.provider", mPickProvider, nil)
	}
	info := llm.Info(c.provider)
	switch c.provider {
	case llm.Azure:
		if strings.TrimSpace(c.baseURL) == "" || strings.TrimSpace(c.deployment) == "" {
			return invalid(ctx, "body.baseUrl", mAzureNeeds, nil)
		}
		if c.model == "" {
			c.model = c.deployment
		}
	case llm.Compatible:
		if strings.TrimSpace(c.baseURL) == "" {
			return invalid(ctx, "body.baseUrl", mBadAddress, i18n.Args{"example": "http://ollama.acme.internal:11434/v1"})
		}
	default:
		c.baseURL, c.deployment, c.apiVersion = "", "", ""
	}
	if c.baseURL != "" {
		clean, perr := s.checkAddress(ctx, c.provider, c.baseURL)
		if perr != nil {
			return perr
		}
		c.baseURL = clean
	}
	if info.NeedsKey && strings.TrimSpace(c.key) == "" {
		return invalid(ctx, "body.apiKey", mKeyNeeded, i18n.Args{"provider": info.Name})
	}
	if strings.TrimSpace(c.model) == "" {
		return invalid(ctx, "body.model", mModelNeeded, nil)
	}
	return nil
}

// keyNeeded reports whether a candidate can't be reached without a key, and
// the provider's name for the message: the hosted providers, and a hosted
// OpenAI-compatible service (unlike a server on the workspace's network).
func keyNeeded(provider, baseURL string) (string, bool) {
	if provider == llm.Compatible {
		p, ok := llm.PresetOf(baseURL)
		return p.Name, ok
	}
	info := llm.Info(provider)
	return info.Name, info.NeedsKey
}

// savedKey is the key a form left blank stands for: the saved one, for the
// same provider at the same server address (Change model, Test connection,
// saving other settings). A saved key is never sent to an address it wasn't
// saved with, so a new address needs the key pasted again.
func (s *Service) savedKey(ctx context.Context, ws *store.Workspace, cur *connection, c candidate) (string, error) {
	if c.key != "" || !cur.connected() || cur.Provider != c.provider || len(cur.KeySealed) == 0 {
		return c.key, nil
	}
	// Only Azure OpenAI and OpenAI-compatible servers are reached at an
	// address of the workspace's choosing; the others have their own.
	if (c.provider == llm.Azure || c.provider == llm.Compatible) && !netguard.SameAddress(c.baseURL, cur.BaseUrl) {
		return "", invalid(ctx, "body.apiKey", mKeyAgain, nil)
	}
	return s.Box.OpenString(cur.KeySealed, keyAD(cur.AccountID))
}

// AITestResult is Test connection: the fixed prompt, the answer and the time.
type AITestResult struct {
	OK           bool      `json:"ok"`
	Prompt       string    `json:"prompt"`
	SentAt       time.Time `json:"sentAt"`
	AnsweredAt   time.Time `json:"answeredAt"`
	Ms           int       `json:"ms"`
	Answer       string    `json:"answer,omitempty"`
	Model        string    `json:"model,omitempty"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	// Error: what the provider answered when the test failed.
	ErrorKind    string `json:"errorKind,omitempty" enum:"auth,rate_limit,overloaded,unavailable,bad_request,timeout,network,too_large,invalid,private"`
	ErrorStatus  int    `json:"errorStatus,omitempty"`
	ErrorDetail  string `json:"errorDetail,omitempty" doc:"401 authentication_error: invalid x-api-key"`
	ErrorMessage string `json:"errorMessage,omitempty" doc:"What it means, as a sentence"`
	// Private network: the server's address is private (and allowed).
	PrivateHost string `json:"privateHost,omitempty"`
	PrivateIP   string `json:"privateIp,omitempty"`
}

// test sends only the fixed prompt. Onboarding, saved-key tests and key
// replacement all reserve/account usage, without granting ai:use to managers.
func (s *Service) test(ctx context.Context, ws *store.Workspace, p *authz.Principal, c candidate, priced *connection) (AITestResult, error) {
	prompt := mTestPrompt.Text(i18n.For(i18n.English), nil)
	out := AITestResult{Prompt: prompt, SentAt: s.now()}
	if c.baseURL != "" {
		if ip, private := resolvePrivate(ctx, hostOf(c.baseURL)); private {
			out.PrivateHost, out.PrivateIP = hostOf(c.baseURL), ip
		}
	}
	var prov llm.Provider
	var err error
	if priced.Demo {
		if !s.Development {
			out.ErrorKind, out.ErrorMessage = "bad_request", errDemoElsewhere.Error()
			out.AnsweredAt = s.now()
			return out, nil
		}
		prov = demoModel{}
	} else {
		retries := 0
		cfg := llm.Config{Kind: c.provider, BaseURL: c.baseURL, APIKey: c.key, Model: c.model, Deployment: c.deployment, APIVersion: c.apiVersion,
			HTTP: s.client(), Timeout: 30 * time.Second, Retries: &retries, RetryWait: s.RetryWait}
		if s.Provider != nil {
			prov, err = s.Provider(cfg)
		} else {
			prov, err = llm.New(cfg)
		}
		if err != nil {
			out.ErrorKind, out.ErrorMessage, out.AnsweredAt = "bad_request", err.Error(), s.now()
			return out, nil //nolint:nilerr // The connection-test result carries the failure for the settings UI.
		}
	}
	t := task{feature: FeatureConnectionTest, req: llm.Request{MaxTokens: 16, Messages: []llm.Message{{Role: "user", Parts: []llm.Part{{Text: prompt}}}}, Task: testTask{}}}
	id := uuid.Must(uuid.NewV7())
	if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return s.reserveIn(ctx, store.New(tx), ws, p, priced, &t, nil, id)
	}); err != nil {
		return out, err
	}
	start := time.Now()
	res, err := prov.Generate(ctx, t.req)
	out.Ms = int(time.Since(start).Milliseconds())
	out.AnsweredAt = s.now()
	row := store.AIRequestInsertParams{ID: id, WorkspaceID: ws.ID, UserID: actorID(p), Feature: FeatureConnectionTest,
		Provider: c.provider, Model: c.model, Outcome: OutcomeNothingDrafted, CreatedAt: out.SentAt, DecidedAt: &out.AnsweredAt, DurationMs: int32(out.Ms)}
	if err != nil {
		row.Outcome, row.OutcomeDetail = OutcomeError, errorDetail(err)
	}
	if accountErr := s.account(ctx, row, res, err); accountErr != nil {
		return out, accountErr
	}
	if err != nil {
		e, ok := llm.AsError(err)
		if !ok {
			out.ErrorKind, out.ErrorMessage = "invalid", err.Error()
			return out, nil
		}
		out.ErrorKind, out.ErrorStatus, out.ErrorDetail = string(e.Kind), e.Status, e.Error()
		if e.Blocked {
			out.ErrorKind = "private"
		}
		return out, nil
	}
	out.OK, out.Answer, out.Model, out.InputTokens, out.OutputTokens = true, strings.TrimSpace(res.Text), res.Model, res.InputTokens, res.OutputTokens
	if out.Model == "" {
		out.Model = c.model
	}
	if priced.Demo {
		// The sample answers at once; it reports the recording's time.
		out.Ms = demoTestMs
		out.AnsweredAt = out.SentAt.Add(demoTestMs * time.Millisecond)
	}
	return out, nil
}

// A candidate's own prices must never inherit another model's prices. The
// workspace's cap applies inside reserveIn: before a first connection, the
// settings row's default of $25 a month.
func (s *Service) testPricing(ctx context.Context, ws *store.Workspace, c candidate, in, out *float64) (*connection, error) {
	input, output, from := pricesOf(c.provider, c.model, in, out)
	if err := validPrices(ctx, input, output); err != nil {
		return nil, err
	}
	return &connection{store.AIBindingRow{WorkspaceID: ws.ID, Provider: c.provider, Model: c.model,
		InputPrice: input, OutputPrice: output, PricesFrom: from, PriceKnown: pricesKnown(c.provider, c.model, in, out)}}, nil
}

// demoTestMs is how long the recorded test took (AISettings: "answered in 1.2 s").
const demoTestMs = 1200

// testTask marks the test prompt for in-process models.
type testTask struct{}

// AITestInput tests a connection: empty tests the saved one.
type AITestInput struct {
	Provider    string   `json:"provider,omitempty" enum:"anthropic,openai,google,azure,compatible"`
	BaseURL     string   `json:"baseUrl,omitempty" maxLength:"500"`
	Deployment  string   `json:"deployment,omitempty" maxLength:"120"`
	APIVersion  string   `json:"apiVersion,omitempty" maxLength:"40"`
	APIKey      string   `json:"apiKey,omitempty" maxLength:"500"`
	Model       string   `json:"model,omitempty" maxLength:"120"`
	InputPrice  *float64 `json:"inputPrice,omitempty" minimum:"0" maximum:"1000"`
	OutputPrice *float64 `json:"outputPrice,omitempty" minimum:"0" maximum:"1000"`
	// ModelID tests a saved model instead of the first account's; AccountID
	// tests an account's first model, or lends its saved key to a model
	// being set up on it.
	ModelID   *uuid.UUID `json:"modelId,omitempty" format:"uuid"`
	AccountID *uuid.UUID `json:"accountId,omitempty" format:"uuid"`
}

// testedModel is the workspace's saved model a test or a model list is
// about: the one named, an account's first, or the first account's. A
// personal key's model is its owner's (the personal operations).
func (s *Service) testedModel(ctx context.Context, ws uuid.UUID, model, account *uuid.UUID) (*connection, error) {
	switch {
	case model != nil:
		c, err := s.binding(ctx, s.Q, ws, model)
		if err == nil && (c == nil || c.personal()) {
			return nil, api.NotFound("AI model")
		}
		return c, err
	case account != nil:
		c, err := s.accountModel(ctx, s.Q, ws, *account, nil)
		if err == nil && c == nil {
			return nil, api.NotFound("AI account")
		}
		return c, err
	}
	return s.load(ctx, s.Q, ws)
}

// Test runs Test connection on the saved model, or on the one being set up.
func (s *Service) Test(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AITestInput) (AITestResult, error) {
	cur, err := s.testedModel(ctx, ws.ID, in.ModelID, in.AccountID)
	if err != nil {
		return AITestResult{}, err
	}
	if in.Provider == "" {
		return s.testSaved(ctx, ws, p, cur)
	}
	c := candidate{provider: in.Provider, baseURL: in.BaseURL, deployment: in.Deployment, apiVersion: in.APIVersion, key: in.APIKey, model: in.Model}
	if c.key, err = s.savedKey(ctx, ws, cur, c); err != nil {
		return AITestResult{}, err
	}
	if err := s.validCandidate(ctx, &c); err != nil {
		return AITestResult{}, err
	}
	name, need := keyNeeded(c.provider, c.baseURL)
	if need && c.key == "" {
		return AITestResult{}, invalid(ctx, "body.apiKey", mKeyNeeded, i18n.Args{"provider": name})
	}
	priced, err := s.testPricing(ctx, ws, c, in.InputPrice, in.OutputPrice)
	if err != nil {
		return AITestResult{}, err
	}
	res, err := s.test(ctx, ws, p, c, priced)
	s.explain(ctx, &res, (&connection{store.AIBindingRow{Provider: c.provider, BaseUrl: c.baseURL}}).providerName(), false)
	return res, err
}

// testSaved tests a saved model with its saved key and records the test on
// its account.
func (s *Service) testSaved(ctx context.Context, ws *store.Workspace, p *authz.Principal, cur *connection) (AITestResult, error) {
	if !cur.connected() {
		return AITestResult{}, problem(ctx, http.StatusConflict, codeNotConnected, mNotConnectedYet, nil)
	}
	c := candidate{provider: cur.Provider, baseURL: cur.BaseUrl, deployment: cur.Deployment, apiVersion: cur.ApiVersion, model: cur.Model}
	if len(cur.KeySealed) > 0 {
		var err error
		if c.key, err = s.Box.OpenString(cur.KeySealed, keyAD(cur.AccountID)); err != nil {
			return AITestResult{}, err
		}
	}
	res, err := s.test(ctx, ws, p, c, cur)
	if err != nil {
		return res, err
	}
	ms := int32(res.Ms)
	now := s.now()
	if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := checkTestConnection(ctx, q, ws, p, cur); err != nil {
			return err
		}
		if err := q.AIAccountSetTest(ctx, store.AIAccountSetTestParams{ID: cur.AccountID, WorkspaceID: ws.ID, LastTestAt: &now, LastTestOk: &res.OK, LastTestMs: &ms}); err != nil {
			return err
		}
		if res.OK && cur.Status != "ok" {
			_, err := q.AIAccountSetStatus(ctx, store.AIAccountSetStatusParams{ID: cur.AccountID, WorkspaceID: ws.ID, Status: "ok", StatusAt: &now, ExpectedRevision: cur.AccountRevision})
			return err
		}
		return nil
	}); err != nil {
		return AITestResult{}, err
	}
	if res.ErrorKind == string(llm.ErrAuth) {
		s.keyRejected(ctx, ws, cur, &llm.Error{Kind: llm.ErrAuth, Status: res.ErrorStatus, Message: res.ErrorDetail})
	}
	s.explain(ctx, &res, cur.providerName(), cur.personal())
	return res, nil
}

// explain words a failed test for the person (whose own key a personal
// one is).
func (s *Service) explain(ctx context.Context, res *AITestResult, provider string, personal bool) {
	if res.OK || res.ErrorKind == "" {
		return
	}
	switch llm.ErrorKind(res.ErrorKind) {
	case llm.ErrAuth:
		res.ErrorMessage = text(ctx, mKeyRejectedAlone, i18n.Args{"provider": provider})
		if personal {
			res.ErrorMessage = text(ctx, mPersonalKeyRejected, i18n.Args{"provider": provider})
		}
	case llm.ErrRateLimit:
		res.ErrorMessage = text(ctx, mRateLimitedNow, i18n.Args{"provider": provider})
	case llm.ErrTimeout:
		res.ErrorMessage = text(ctx, mTimeout, i18n.Args{"provider": provider, "seconds": 30})
	case llm.ErrNetwork:
		res.ErrorMessage = text(ctx, mUnreachable, i18n.Args{"provider": provider, "detail": res.ErrorDetail})
	case "private":
		res.ErrorMessage = text(ctx, mPrivateHost, i18n.Args{"host": res.PrivateHost})
	}
	if res.ErrorMessage == "" && res.ErrorDetail != "" {
		res.ErrorMessage = text(ctx, mBadRequest, i18n.Args{"provider": provider, "detail": res.ErrorDetail})
	}
}

// AIModelsInput asks for a provider's models.
type AIModelsInput struct {
	Provider   string `json:"provider" enum:"anthropic,openai,google,azure,compatible"`
	BaseURL    string `json:"baseUrl,omitempty" maxLength:"500"`
	Deployment string `json:"deployment,omitempty" maxLength:"120"`
	APIVersion string `json:"apiVersion,omitempty" maxLength:"40"`
	APIKey     string `json:"apiKey,omitempty" maxLength:"500"`
	// AccountID lends that account's saved key (at the same address)
	// instead of the first account's.
	AccountID *uuid.UUID `json:"accountId,omitempty" format:"uuid"`
}

// AIModelList is a provider's models, the price list's first.
type AIModelList struct {
	Models []AIModelOption `json:"models"`
	// FromProvider: the list came from the provider; otherwise it is the
	// price list's (the key wasn't given yet, or the provider didn't answer).
	FromProvider bool   `json:"fromProvider"`
	Error        string `json:"error,omitempty"`
}

// Models lists the models a key can use.
func (s *Service) Models(ctx context.Context, ws *store.Workspace, in AIModelsInput) (AIModelList, error) {
	out := AIModelList{Models: []AIModelOption{}}
	for _, k := range llm.PriceList {
		if k.Provider == in.Provider || (in.Provider == llm.Azure && k.Provider == llm.OpenAI) {
			out.Models = append(out.Models, AIModelOption{ID: k.ID, Name: k.Name, Known: true, Input: k.Input, Output: k.Output, ImageInput: k.ImageInput})
		}
	}
	c := candidate{provider: in.Provider, baseURL: in.BaseURL, deployment: in.Deployment, apiVersion: in.APIVersion, key: in.APIKey, model: "list"}
	cur, err := s.testedModel(ctx, ws.ID, nil, in.AccountID)
	if err != nil {
		return out, err
	}
	if c.key, err = s.savedKey(ctx, ws, cur, c); err != nil {
		return out, err
	}
	if cur.connected() && cur.Demo && cur.Provider == c.provider && in.APIKey == "" {
		return out, nil
	}
	if _, need := keyNeeded(c.provider, c.baseURL); need && c.key == "" {
		return out, nil
	}
	if err := s.validCandidate(ctx, &c); err != nil {
		return out, err
	}
	cfg := llm.Config{Kind: c.provider, BaseURL: c.baseURL, APIKey: c.key, Model: c.model, Deployment: c.deployment, APIVersion: c.apiVersion, HTTP: s.client(), Timeout: 20 * time.Second}
	models, lerr := s.listModels(ctx, cfg)
	if lerr != nil {
		// The price list's models stay offered, with what the provider said.
		out.Error = lerr.Error()
		return out, nil //nolint:nilerr // a provider that doesn't answer still leaves the list to pick from
	}
	known := map[string]bool{}
	for _, m := range out.Models {
		known[m.ID] = true
	}
	out.FromProvider = true
	listed := []AIModelOption{}
	for _, m := range models {
		if strings.Contains(m.ID, "embedding") || strings.HasPrefix(m.ID, "tts-") || strings.HasPrefix(m.ID, "whisper") || strings.HasPrefix(m.ID, "dall-e") {
			continue
		}
		if k, ok := llm.Known(c.provider, m.ID); ok && k.ID == m.ID {
			continue
		}
		name := m.Name
		if name == "" {
			name = m.ID
		}
		listed = append(listed, AIModelOption{ID: m.ID, Name: name, ImageInput: llm.ImageInput(c.provider, m.ID)})
	}
	// The price list's models the provider offers come first.
	offered := map[string]bool{}
	for _, m := range models {
		offered[m.ID] = true
	}
	first := []AIModelOption{}
	for _, m := range out.Models {
		if offered[m.ID] || c.provider == llm.Azure {
			first = append(first, m)
		}
	}
	out.Models = append(first, listed...)
	return out, nil
}

// listModels asks a provider for its models.
func (s *Service) listModels(ctx context.Context, cfg llm.Config) ([]llm.Model, error) {
	newProvider := llm.New
	if s.Provider != nil {
		newProvider = s.Provider
	}
	prov, err := newProvider(cfg)
	if err != nil {
		return nil, err
	}
	return prov.Models(ctx)
}

func pricesOf(provider, model string, in, outp *float64) (float64, float64, *time.Time) {
	k, known := llm.Known(provider, model)
	input, output := 0.0, 0.0
	var from *time.Time
	if known {
		input, output = k.Input, k.Output
		d := llm.PriceListDate
		from = &d
	}
	if in != nil {
		if !known || *in != k.Input {
			from = nil
		}
		input = *in
	}
	if outp != nil {
		if !known || *outp != k.Output {
			from = nil
		}
		output = *outp
	}
	return input, output, from
}

func pricesKnown(provider, model string, input, output *float64) bool {
	_, known := llm.Known(provider, model)
	return known || (input != nil && output != nil)
}

func validCap(ctx context.Context, kind string, usd float64, tokens int64) error {
	switch kind {
	case "usd":
		if usd < 1 || usd > 100000 {
			r := i18n.LangFrom(ctx).Region
			return invalid(ctx, "body.capUsd", mCapRange, i18n.Args{"min": r.Money(1, 2), "max": r.Money(100000, 2)})
		}
	case "tokens":
		if tokens < 10000 {
			return invalid(ctx, "body.capTokens", mTokenCapRange, i18n.Args{"min": i18n.LangFrom(ctx).Region.Int(10000)})
		}
	}
	return nil
}

func validPrices(ctx context.Context, in, out float64) error {
	if in < 0 || in > 1000 || out < 0 || out > 1000 {
		r := i18n.LangFrom(ctx).Region
		return invalid(ctx, "body.inputPrice", mPriceRange, i18n.Args{"min": r.Money(0, 0), "max": r.Money(1000, 0)})
	}
	return nil
}

// featuresOff turns the features to switch on into the ones switched off.
func featuresOff(ctx context.Context, on []string) ([]string, error) {
	if len(on) == 0 {
		return []string{}, nil
	}
	for _, f := range on {
		if !validFeature(f) {
			return nil, invalid(ctx, "body.features", mUnknownFeature, nil)
		}
	}
	off := []string{}
	for _, f := range Features {
		if !slices.Contains(on, f) {
			off = append(off, f)
		}
	}
	return off, nil
}

// auditView is what the audit entry keeps of a connection: never the key.
func auditView(c store.AIBindingRow) map[string]any {
	on := []string{}
	for _, f := range Features {
		if !slices.Contains(c.FeaturesOff, f) {
			on = append(on, f)
		}
	}
	m := map[string]any{"provider": c.Provider, "model": c.Model, "capKind": c.CapKind, "inputPrice": c.InputPrice, "outputPrice": c.OutputPrice, "features": on, "keySet": len(c.KeySealed) > 0}
	switch c.CapKind {
	case "usd":
		m["capUsd"] = c.CapUsd
	case "tokens":
		m["capTokens"] = c.CapTokens
	}
	if c.BaseUrl != "" {
		m["baseUrl"] = c.BaseUrl
	}
	if c.Deployment != "" {
		m["deployment"] = c.Deployment
	}
	if c.Label != "" {
		m["label"] = c.Label
	}
	return m
}

// What of a connection each table keeps. Saving bumps the row's revision,
// which the requests bound to it compare.
func settingsParams(c store.AIBindingRow, by *uuid.UUID, at time.Time) store.AIWorkspaceSettingsSaveParams {
	return store.AIWorkspaceSettingsSaveParams{WorkspaceID: c.WorkspaceID, CapKind: c.CapKind, CapUsd: c.CapUsd, CapTokens: c.CapTokens, CapSetBy: c.CapSetBy, CapSetAt: c.CapSetAt,
		FeaturesOff: c.FeaturesOff, NoticeMonth: c.NoticeMonth, CapNoticeMonth: c.CapNoticeMonth, UpdatedBy: by, UpdatedAt: at}
}

func accountParams(c store.AIBindingRow, by *uuid.UUID, at time.Time) store.AIAccountSaveParams {
	return store.AIAccountSaveParams{ID: c.AccountID, WorkspaceID: c.WorkspaceID, Provider: c.Provider, Label: c.Label, BaseUrl: c.BaseUrl, ApiVersion: c.ApiVersion,
		KeySealed: c.KeySealed, KeyAddedBy: c.KeyAddedBy, KeyAddedAt: c.KeyAddedAt, ConnectedBy: c.ConnectedBy, ConnectedAt: c.ConnectedAt,
		DisconnectedBy: c.DisconnectedBy, DisconnectedAt: c.DisconnectedAt, Status: c.Status, StatusDetail: c.StatusDetail, StatusAt: c.StatusAt,
		LastTestAt: c.LastTestAt, LastTestOk: c.LastTestOk, LastTestMs: c.LastTestMs, Demo: c.Demo, UpdatedBy: by, UpdatedAt: at, UserID: c.AccountUserID}
}

func modelParams(c store.AIBindingRow, by *uuid.UUID, at time.Time) store.AIModelSaveParams {
	return store.AIModelSaveParams{ID: c.ModelID, WorkspaceID: c.WorkspaceID, AccountID: c.AccountID, Model: c.Model, ModelName: c.ModelName, Deployment: c.Deployment,
		ImageInput: c.ImageInput, InputPrice: c.InputPrice, OutputPrice: c.OutputPrice, PricesFrom: c.PricesFrom, PriceKnown: c.PriceKnown, UpdatedBy: by, UpdatedAt: at}
}

// Connect connects the workspace's model (its first account's first model),
// after the fixed test prompt passes. Here another provider needs a
// disconnect first. A first model becomes every feature's default.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIConnectInput) (AISettings, error) {
	c := candidate{provider: in.Provider, label: strings.TrimSpace(in.Label), baseURL: in.BaseURL, deployment: strings.TrimSpace(in.Deployment), apiVersion: strings.TrimSpace(in.APIVersion),
		key: strings.TrimSpace(in.APIKey), model: strings.TrimSpace(in.Model)}
	if err := s.validCandidate(ctx, &c); err != nil {
		return AISettings{}, err
	}
	if err := validCap(ctx, in.CapKind, in.CapUSD, in.CapTokens); err != nil {
		return AISettings{}, err
	}
	input, output, from := pricesOf(c.provider, c.model, in.InputPrice, in.OutputPrice)
	if err := validPrices(ctx, input, output); err != nil {
		return AISettings{}, err
	}
	off, err := featuresOff(ctx, in.Features)
	if err != nil {
		return AISettings{}, err
	}
	cur, err := s.load(ctx, s.Q, ws.ID)
	if err != nil {
		return AISettings{}, err
	}
	if cur.connected() && cur.Provider != c.provider {
		return AISettings{}, problem(ctx, http.StatusConflict, codeOneModel, mOneModel, i18n.Args{"provider": cur.providerName()})
	}
	if cur.connected() && !cur.Demo {
		if c.key, err = s.savedKey(ctx, ws, cur, c); err != nil {
			return AISettings{}, err
		}
	}
	if name, need := keyNeeded(c.provider, c.baseURL); need && c.key == "" {
		return AISettings{}, invalid(ctx, "body.apiKey", mKeyNeeded, i18n.Args{"provider": name})
	}
	priced, err := s.testPricing(ctx, ws, c, in.InputPrice, in.OutputPrice)
	if err != nil {
		return AISettings{}, err
	}
	res, err := s.test(ctx, ws, p, c, priced)
	if err != nil {
		return AISettings{}, err
	}
	if !res.OK {
		s.explain(ctx, &res, llm.Info(c.provider).Name, false)
		return AISettings{}, api.Err(http.StatusUnprocessableEntity, "ai_test_failed", res.ErrorMessage).WithMeta("test", res)
	}
	now := s.now()
	actor := actorID(p)
	// The first account and model of the workspace, or the ones it has.
	accountID, modelID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	if cur != nil {
		accountID, modelID = cur.AccountID, cur.ModelID
	}
	var sealed []byte
	if c.key != "" {
		if sealed, err = s.Box.SealString(c.key, keyAD(accountID)); err != nil {
			return AISettings{}, err
		}
	}
	name := strings.TrimSpace(in.ModelName)
	if name == "" {
		name = llm.ModelName(c.provider, c.model)
	}
	ms := int32(res.Ms)
	ok := true
	row := store.AIBindingRow{
		ModelID: modelID, AccountID: accountID, WorkspaceID: ws.ID, Provider: c.provider, Label: c.label, BaseUrl: c.baseURL, Deployment: c.deployment, ApiVersion: c.apiVersion,
		Model: c.model, ModelName: name, ImageInput: llm.ImageInput(c.provider, c.model), KeySealed: sealed, KeyAddedBy: actor, KeyAddedAt: &now,
		ConnectedBy: actor, ConnectedAt: now, Status: "ok", LastTestAt: &now, LastTestOk: &ok, LastTestMs: &ms,
		CapKind: in.CapKind, CapUsd: in.CapUSD, CapTokens: in.CapTokens, CapSetBy: actor, CapSetAt: &now,
		InputPrice: input, OutputPrice: output, PricesFrom: from, FeaturesOff: off,
		PriceKnown: pricesKnown(c.provider, c.model, in.InputPrice, in.OutputPrice),
	}
	if row.CapKind == "" {
		row.CapKind = "usd"
	}
	if len(sealed) == 0 {
		row.KeyAddedBy, row.KeyAddedAt = nil, nil
	}
	action := "ai.connection_created"
	var before any
	if cur != nil {
		row.NoticeMonth, row.CapNoticeMonth = cur.NoticeMonth, cur.CapNoticeMonth
		if cur.connected() {
			action = "ai.connection_updated"
			before = auditView(cur.AIBindingRow)
			row.ConnectedBy, row.ConnectedAt = cur.ConnectedBy, cur.ConnectedAt
			if in.APIKey == "" && len(cur.KeySealed) > 0 {
				row.KeyAddedBy, row.KeyAddedAt = cur.KeyAddedBy, cur.KeyAddedAt
			}
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := checkTestConnection(ctx, q, ws, p, cur); err != nil {
			return err
		}
		if err := q.AIWorkspaceSettingsSave(ctx, settingsParams(row, actor, now)); err != nil {
			return err
		}
		if err := q.AIAccountSave(ctx, accountParams(row, actor, now)); err != nil {
			return err
		}
		if cur != nil && cur.Provider != row.Provider {
			// The account's other models were the previous provider's: their
			// features move to the one connected now.
			if err := q.AIFeatureModelsMove(ctx, store.AIFeatureModelsMoveParams{WorkspaceID: ws.ID, AccountID: accountID, KeepID: modelID}); err != nil {
				return err
			}
			if err := q.AIModelsDeleteOthers(ctx, store.AIModelsDeleteOthersParams{WorkspaceID: ws.ID, AccountID: accountID, KeepID: modelID}); err != nil {
				return err
			}
		}
		if err := q.AIModelSave(ctx, modelParams(row, actor, now)); err != nil {
			return modelExists(ctx, err, row.Model)
		}
		if err := q.AIFeatureModelsFill(ctx, store.AIFeatureModelsFillParams{WorkspaceID: ws.ID, ModelID: modelID, Features: defaultFeatures}); err != nil {
			return err
		}
		meta := map[string]any{"test": map[string]any{"ok": true, "ms": res.Ms}}
		key := keyNone
		switch {
		case in.APIKey != "" && action == "ai.connection_updated":
			key = keyReplaced
		case in.APIKey != "":
			key = keyAdded
		case len(row.KeySealed) > 0:
			key = keyKept
		}
		if in.APIKey != "" {
			meta["key"] = "added (write-only)"
		}
		meta["summary"] = connectionLine(row, key)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "ai_assistant", TargetID: ws.ID.String(),
			TargetName: llm.Info(c.provider).Name, Before: before, After: auditView(row), Metadata: meta})
	})
	if err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

// AIKeyInput replaces the key.
type AIKeyInput struct {
	APIKey string `json:"apiKey" minLength:"1" maxLength:"500"`
}

// ReplaceKey tests a new key and saves it.
func (s *Service) ReplaceKey(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIKeyInput) (AISettings, AITestResult, error) {
	cur, err := s.load(ctx, s.Q, ws.ID)
	if err != nil {
		return AISettings{}, AITestResult{}, err
	}
	test, err := s.replaceKey(ctx, ws, p, cur, in, "ai.key_replaced", ws.ID.String())
	if err != nil {
		return AISettings{}, test, err
	}
	out, err := s.Settings(ctx, ws, p)
	return out, test, err
}

// replaceKey tests a new key for a model's account with that model, and
// saves it. A person's own account whose key was removed (they left the
// workspace and came back) is connected again with a new key.
func (s *Service) replaceKey(ctx context.Context, ws *store.Workspace, p *authz.Principal, cur *connection, in AIKeyInput, action, target string) (AITestResult, error) {
	if !cur.connected() && !cur.personal() {
		return AITestResult{}, problem(ctx, http.StatusConflict, codeNotConnected, mNotConnectedYet, nil)
	}
	c := candidate{provider: cur.Provider, baseURL: cur.BaseUrl, deployment: cur.Deployment, apiVersion: cur.ApiVersion, model: cur.Model, key: strings.TrimSpace(in.APIKey)}
	priced := *cur
	priced.Demo = false
	res, err := s.test(ctx, ws, p, c, &priced)
	if err != nil {
		return res, err
	}
	if !res.OK {
		s.explain(ctx, &res, cur.providerName(), cur.personal())
		return res, api.Err(http.StatusUnprocessableEntity, "ai_test_failed", res.ErrorMessage).WithMeta("test", res)
	}
	sealed, err := s.Box.SealString(c.key, keyAD(cur.AccountID))
	if err != nil {
		return res, err
	}
	now := s.now()
	row := cur.AIBindingRow
	ok := true
	ms := int32(res.Ms)
	row.KeySealed, row.KeyAddedBy, row.KeyAddedAt, row.Status, row.StatusDetail, row.StatusAt = sealed, actorID(p), &now, "ok", "", &now
	row.LastTestAt, row.LastTestOk, row.LastTestMs, row.Demo = &now, &ok, &ms, false
	if !cur.connected() {
		row.ConnectedBy, row.ConnectedAt, row.DisconnectedBy, row.DisconnectedAt = actorID(p), now, nil, nil
	}
	meta := map[string]any{"key": "replaced (write-only)", "test": map[string]any{"ok": true, "ms": res.Ms}, "summary": keyReplaced + " · test passed"}
	if cur.personal() {
		meta["personal"] = true
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := checkTestConnection(ctx, q, ws, p, cur); err != nil {
			return err
		}
		if err := q.AIAccountSave(ctx, accountParams(row, actorID(p), now)); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "ai_assistant", TargetID: target,
			TargetName: cur.providerName(), Metadata: meta})
	})
	return res, err
}

// A successful test cannot overwrite newer settings or outlive the manager's
// access. Onboarding serializes the absent model with other first connections.
func checkTestConnection(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, before *connection) error {
	// A personal key's owner tests and saves it with ai:use.
	permission := authz.IntegrationsManage
	if before.personal() {
		permission = authz.AIUse
	}
	if _, err := draftActor(ctx, q, ws, p, false, permission); err != nil {
		return err
	}
	if _, err := lockSettings(ctx, q, ws.ID); err != nil {
		return err
	}
	if before == nil {
		_, err := q.AIPrimaryModel(ctx, ws.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return budgetProblem(ctx, "connection_changed")
	}
	row, err := q.AIBinding(ctx, store.AIBindingParams{WorkspaceID: ws.ID, ModelID: before.ModelID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err != nil || row.SettingsRevision != before.SettingsRevision || row.AccountRevision != before.AccountRevision || row.ModelRevision != before.ModelRevision {
		return budgetProblem(ctx, "connection_changed")
	}
	return nil
}

// AIModelInput changes the model (Change model).
type AIModelInput struct {
	Model     string `json:"model" minLength:"1" maxLength:"120"`
	ModelName string `json:"modelName,omitempty" maxLength:"120"`
	// Prices follow the price list for a known model unless given.
	InputPrice  *float64 `json:"inputPrice,omitempty" minimum:"0"`
	OutputPrice *float64 `json:"outputPrice,omitempty" minimum:"0"`
}

// auditChange is what a change records: before and after, and the details line.
type auditChange struct {
	before, after map[string]any
	summary       string
}

// What a change saves: the workspace's settings, the account or the model.
const (
	saveSettings = 1 << iota
	saveAccount
	saveModel
)

// update applies a change to the connected model (the workspace's first),
// its account or the workspace's settings, and audits it.
func (s *Service) update(ctx context.Context, ws *store.Workspace, p *authz.Principal, action string, parts int, change func(row *store.AIBindingRow) (auditChange, error)) (AISettings, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return err
		}
		c, err := s.load(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if !c.connected() {
			return problem(ctx, http.StatusConflict, codeNotConnected, mNotConnectedYet, nil)
		}
		row := c.AIBindingRow
		ch, err := change(&row)
		if err != nil {
			return err
		}
		by, at := actorID(p), s.now()
		if parts&saveSettings != 0 {
			if err := q.AIWorkspaceSettingsSave(ctx, settingsParams(row, by, at)); err != nil {
				return err
			}
		}
		if parts&saveAccount != 0 {
			if err := q.AIAccountSave(ctx, accountParams(row, by, at)); err != nil {
				return err
			}
		}
		if parts&saveModel != 0 {
			if err := q.AIModelSave(ctx, modelParams(row, by, at)); err != nil {
				return modelExists(ctx, err, row.Model)
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "ai_assistant", TargetID: ws.ID.String(),
			TargetName: llm.Info(row.Provider).Name, Before: ch.before, After: ch.after, Metadata: map[string]any{"summary": ch.summary}})
	})
	if err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

// ChangeModel picks another model of the same provider.
func (s *Service) ChangeModel(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIModelInput) (AISettings, error) {
	return s.update(ctx, ws, p, "ai.model_changed", saveModel, func(row *store.AIBindingRow) (auditChange, error) {
		model := strings.TrimSpace(in.Model)
		input, output, from := pricesOf(row.Provider, model, in.InputPrice, in.OutputPrice)
		if err := validPrices(ctx, input, output); err != nil {
			return auditChange{}, err
		}
		before := map[string]any{"model": row.Model, "inputPrice": row.InputPrice, "outputPrice": row.OutputPrice}
		summary := modelLine(row.Model, model, row.InputPrice, row.OutputPrice, input, output)
		name := strings.TrimSpace(in.ModelName)
		if name == "" {
			name = llm.ModelName(row.Provider, model)
		}
		row.Model, row.ModelName, row.ImageInput, row.InputPrice, row.OutputPrice, row.PricesFrom = model, name, llm.ImageInput(row.Provider, model), input, output, from
		row.PriceKnown = pricesKnown(row.Provider, model, in.InputPrice, in.OutputPrice)
		if row.Provider == llm.Azure {
			row.Deployment = model
		}
		return auditChange{before, map[string]any{"model": model, "inputPrice": input, "outputPrice": output}, summary}, nil
	})
}

// AICapInput sets the monthly cap.
type AICapInput struct {
	Kind   string  `json:"kind" enum:"usd,tokens,none"`
	USD    float64 `json:"usd,omitempty" minimum:"0"`
	Tokens int64   `json:"tokens,omitempty" minimum:"0"`
}

// SetCap changes the monthly cap.
func (s *Service) SetCap(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AICapInput) (AISettings, error) {
	if err := validCap(ctx, in.Kind, in.USD, in.Tokens); err != nil {
		return AISettings{}, err
	}
	return s.update(ctx, ws, p, "ai.cap_updated", saveSettings, func(row *store.AIBindingRow) (auditChange, error) {
		before := map[string]any{"capKind": row.CapKind, "capUsd": row.CapUsd, "capTokens": row.CapTokens}
		summary := capLine(*row, in.Kind, in.USD, in.Tokens)
		now := s.now()
		row.CapKind, row.CapUsd, row.CapTokens, row.CapSetBy, row.CapSetAt = in.Kind, in.USD, in.Tokens, actorID(p), &now
		// A raised cap can notify again this month.
		row.NoticeMonth, row.CapNoticeMonth = "", ""
		return auditChange{before, map[string]any{"capKind": in.Kind, "capUsd": in.USD, "capTokens": in.Tokens}, summary}, nil
	})
}

// AIPricesInput sets the prices.
type AIPricesInput struct {
	Input  float64 `json:"input" minimum:"0" doc:"Dollars per million input tokens"`
	Output float64 `json:"output" minimum:"0" doc:"Dollars per million output tokens"`
}

// SetPrices changes the prices the cap is estimated with.
func (s *Service) SetPrices(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIPricesInput) (AISettings, error) {
	if err := validPrices(ctx, in.Input, in.Output); err != nil {
		return AISettings{}, err
	}
	return s.update(ctx, ws, p, "ai.prices_updated", saveModel, func(row *store.AIBindingRow) (auditChange, error) {
		before := map[string]any{"inputPrice": row.InputPrice, "outputPrice": row.OutputPrice}
		summary := pricesLine(row.InputPrice, row.OutputPrice, in.Input, in.Output)
		in1, out1 := in.Input, in.Output
		row.InputPrice, row.OutputPrice, row.PricesFrom = pricesOf(row.Provider, row.Model, &in1, &out1)
		row.PriceKnown = true
		return auditChange{before, map[string]any{"inputPrice": in.Input, "outputPrice": in.Output}, summary}, nil
	})
}

// SetFeature switches a feature on or off.
func (s *Service) SetFeature(ctx context.Context, ws *store.Workspace, p *authz.Principal, feature string, on bool) (AISettings, error) {
	if !validFeature(feature) {
		return AISettings{}, invalid(ctx, "path.feature", mUnknownFeature, nil)
	}
	action := "ai.feature_disabled"
	if on {
		action = "ai.feature_enabled"
	}
	return s.update(ctx, ws, p, action, saveSettings, func(row *store.AIBindingRow) (auditChange, error) {
		was := !slices.Contains(row.FeaturesOff, feature)
		off := slices.DeleteFunc(slices.Clone(row.FeaturesOff), func(f string) bool { return f == feature })
		if !on {
			off = append(off, feature)
		}
		row.FeaturesOff = off
		return auditChange{map[string]any{"feature": feature, "on": was}, map[string]any{"feature": feature, "on": on}, featureLine(feature, on)}, nil
	})
}

// Disconnect removes the key. The model, the cap and the prices stay for
// the next connection; the activity log, kept summaries and accepted text
// stay too.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AISettings, error) {
	return s.update(ctx, ws, p, "ai.connection_removed", saveAccount, func(row *store.AIBindingRow) (auditChange, error) {
		before := auditView(*row)
		now := s.now()
		row.KeySealed, row.KeyAddedBy, row.KeyAddedAt, row.DisconnectedAt, row.DisconnectedBy, row.Status, row.StatusDetail = nil, nil, nil, &now, actorID(p), "ok", ""
		return auditChange{before, map[string]any{"key": "deleted", "kept": "the model, the cap and the prices"}, disconnectLine}, nil
	})
}
