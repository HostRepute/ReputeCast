package earn

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/connect"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Problem codes of the connect flow.
const (
	CodeSecretKey     = "secret_key"
	CodeKeyFormat     = "key_format"
	CodeKeyRejected   = "key_rejected"
	CodeModeConnected = "mode_connected"
	CodeUnavailable   = "provider_unavailable"
	CodeOtherAccount  = "other_account"
	CodeNotConnected  = "not_connected"
	CodeConfirm       = "confirmation_mismatch"
)

// Wizard steps.
const (
	StepCheck    = "check"
	StepWebhook  = "webhook"
	StepSettings = "settings"
	StepDone     = "done"
)

// Health of a connection, worst first.
const (
	HealthSetup              = "setup"
	HealthKeyRejected        = "key_rejected"
	HealthWebhookMissing     = "webhook_missing"
	HealthMissingPermissions = "missing_permissions"
	HealthWebhookSilent      = "webhook_silent"
	HealthConnected          = "connected"
)

// SilentAfter is how long without an event makes a webhook silent, while
// at least SilentMinimum subscriptions are active.
const (
	SilentAfter   = 24 * time.Hour
	SilentMinimum = 50
)

// EarnProviderCard is a payment provider in the catalog (3.1).
type EarnProviderCard struct {
	Key              string   `json:"key" enum:"stripe,paddle,lemonsqueezy"`
	Name             string   `json:"name"`
	Available        bool     `json:"available"`
	MerchantOfRecord bool     `json:"merchantOfRecord"`
	Capabilities     []string `json:"capabilities"`
}

// EarnPermissionCheck is one permission the restricted key needs and,
// once checked, whether it has it.
type EarnPermissionCheck struct {
	Resource string `json:"resource"`
	Access   string `json:"access" enum:"read,write"`
	OK       *bool  `json:"ok,omitempty"`
}

// EarnAccount is the connected account.
type EarnAccount struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Country    string `json:"country"`
	Currency   string `json:"currency"`
	TaxEnabled bool   `json:"taxEnabled"`
}

// EarnReach is the last check of whether the provider can reach the
// webhook address.
type EarnReach struct {
	OK      bool       `json:"ok"`
	Checked bool       `json:"checked"`
	At      *time.Time `json:"at,omitempty"`
	Code    string     `json:"code,omitempty"`
	Reason  string     `json:"reason,omitempty"`
	// Host is the public URL's host ("reputecast.acme.io").
	Host string `json:"host"`
}

// Reach is what a reachability check found.
type Reach = connect.SelfTestState

// EarnGapRun is one run of the hourly check for missed events.
type EarnGapRun struct {
	At      time.Time      `json:"at"`
	Applied int            `json:"applied"`
	Types   map[string]int `json:"types"`
}

// EarnEventCount is the events of one type in the last 24 hours.
type EarnEventCount struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// EarnWebhook is the connection's webhook and its freshness.
type EarnWebhook struct {
	URL          string     `json:"url"`
	ID           string     `json:"id,omitempty"`
	Registered   bool       `json:"registered"`
	RegisteredAt *time.Time `json:"registeredAt,omitempty"`
	Manual       bool       `json:"manual"`
	Events       []string   `json:"events"`
	SecretStored bool       `json:"secretStored"`
	MissingAt    *time.Time `json:"missingAt,omitempty"`
	Reach        EarnReach  `json:"reach"`
	// Status: unregistered, waiting (no event yet), receiving, silent, missing.
	Status         string           `json:"status" enum:"unregistered,waiting,receiving,silent,missing"`
	FirstEventAt   *time.Time       `json:"firstEventAt,omitempty"`
	FirstEventType string           `json:"firstEventType,omitempty"`
	LastEventAt    *time.Time       `json:"lastEventAt,omitempty"`
	LastEventType  string           `json:"lastEventType,omitempty"`
	Events24h      int64            `json:"events24h"`
	Rejected24h    int64            `json:"rejected24h"`
	ByType         []EarnEventCount `json:"byType"`
	GapCheckedAt   *time.Time       `json:"gapCheckedAt,omitempty"`
	GapRuns        []EarnGapRun     `json:"gapRuns"`
	NextGapAt      *time.Time       `json:"nextGapAt,omitempty"`
	// SilentHours is how long no event came, when silent.
	SilentHours int `json:"silentHours,omitempty"`
}

// EarnConnection is a payment connection: the connect wizard's state and
// the Payments tab's card.
type EarnConnection struct {
	ID            string                `json:"id"`
	Provider      string                `json:"provider" enum:"stripe"`
	ProviderName  string                `json:"providerName"`
	Mode          string                `json:"mode" enum:"live,test"`
	Status        string                `json:"status" enum:"setup,connected"`
	Step          string                `json:"step" enum:"check,webhook,settings,done"`
	Health        string                `json:"health" enum:"setup,key_rejected,webhook_missing,missing_permissions,webhook_silent,connected"`
	KeyHint       string                `json:"keyHint"`
	Account       EarnAccount           `json:"account"`
	Permissions   []EarnPermissionCheck `json:"permissions"`
	Granted       int                   `json:"granted"`
	CheckedAt     *time.Time            `json:"checkedAt,omitempty"`
	KeyRejectedAt *time.Time            `json:"keyRejectedAt,omitempty"`
	Webhook       EarnWebhook           `json:"webhook"`
	ConnectedAt   *time.Time            `json:"connectedAt,omitempty"`
	ConnectedBy   string                `json:"connectedBy,omitempty"`
	PortalSet     bool                  `json:"portalSet"`
	Demo          bool                  `json:"demo"`
	// Links into the provider's dashboard.
	DashboardURL string `json:"dashboardUrl"`
	KeysURL      string `json:"keysUrl"`
	CreateKeyURL string `json:"createKeyUrl"`
	WebhooksURL  string `json:"webhooksUrl"`
	EmailsURL    string `json:"emailsUrl"`
}

// EarnCheckoutSettings are checkout's settings: currency, Stripe Tax and
// what the customer portal lets readers do.
type EarnCheckoutSettings struct {
	Currency string `json:"currency"`
	// CurrencyFixed is true once a tier has prices.
	CurrencyFixed    bool       `json:"currencyFixed"`
	TaxAvailable     bool       `json:"taxAvailable"`
	TaxAtCheckout    bool       `json:"taxAtCheckout"`
	PricesIncludeTax bool       `json:"pricesIncludeTax"`
	Portal           EarnPortal `json:"portal"`
}

// EarnPortal is what the customer portal lets readers do.
type EarnPortal struct {
	UpdateCard   bool `json:"updateCard"`
	Invoices     bool `json:"invoices"`
	BillingEmail bool `json:"billingEmail"`
	SwitchPeriod bool `json:"switchPeriod"`
	Cancel       bool `json:"cancel"`
}

// defaultPortal turns every portal feature on.
var defaultPortal = EarnPortal{UpdateCard: true, Invoices: true, BillingEmail: true, SwitchPeriod: true, Cancel: true}

// EarnPayments is Earn › Settings › Payments.
type EarnPayments struct {
	Providers     []EarnProviderCard    `json:"providers"`
	Live          *EarnConnection       `json:"live,omitempty"`
	Test          *EarnConnection       `json:"test,omitempty"`
	Checkout      EarnCheckoutSettings  `json:"checkout"`
	Permissions   []EarnPermissionCheck `json:"permissions"`
	WebhookEvents []string              `json:"webhookEvents"`
	Unmapped      []EarnUnmappedPrice   `json:"unmapped"`
	// Active is the mode public pages use: live once connected, else test.
	Active string `json:"active" enum:"live,test,"`
	// ActiveSubscriptions counts the live paying subscriptions (the silent
	// webhook's sentence).
	ActiveSubscriptions int64 `json:"activeSubscriptions"`
	TestMembers         int64 `json:"testMembers"`
	CanManage           bool  `json:"canManage"`
	// CanEditPrices maps unmapped prices (earn:edit).
	CanEditPrices bool     `json:"canEditPrices"`
	Helpers       []string `json:"helpers"`
}

// EarnStatus is the connection chip on every Earn screen.
type EarnStatus struct {
	Provider     string `json:"provider,omitempty"`
	ProviderName string `json:"providerName,omitempty"`
	Account      string `json:"account,omitempty"`
	// Mode is the mode public pages use; "" without a payment provider.
	Mode   string `json:"mode" enum:"live,test,"`
	Health string `json:"health,omitempty"`
	// TestOnly: only a test key is connected (the Test mode banner).
	TestOnly  bool `json:"testOnly"`
	CanManage bool `json:"canManage"`
	// Setup is a connect wizard in progress, to resume.
	Setup string `json:"setup,omitempty"`
	// UpgradeURL is the upgrade page ("Open upgrade page").
	UpgradeURL string `json:"upgradeUrl"`
}

// connRow is a connection with the name of who connected it.
type connRow struct {
	store.PaymentConnection
	ConnectedByName string
}

func (s *Service) connections(ctx context.Context, wsID uuid.UUID) ([]connRow, error) {
	rows, err := s.Q.PayConnections(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make([]connRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, connRow{PaymentConnection: r.PaymentConnection, ConnectedByName: r.ConnectedByName})
	}
	return out, nil
}

func (s *Service) connection(ctx context.Context, wsID uuid.UUID, id string) (connRow, error) {
	cid, err := uuid.Parse(id)
	if err != nil {
		return connRow{}, api.NotFound("Payment connection")
	}
	r, err := s.Q.PayConnection(ctx, store.PayConnectionParams{ID: cid, WorkspaceID: wsID})
	if err != nil {
		return connRow{}, api.MapError(err, "Payment connection")
	}
	if r.PaymentConnection.Status == "disconnected" {
		return connRow{}, api.NotFound("Payment connection")
	}
	return connRow{PaymentConnection: r.PaymentConnection, ConnectedByName: r.ConnectedByName}, nil
}

// active returns the connection public pages use: live once connected,
// else test; ok is false without one.
func active(conns []connRow) (connRow, bool) {
	for _, m := range []string{"live", "test"} {
		for _, c := range conns {
			if c.Mode == m && c.Status == "connected" {
				return c, true
			}
		}
	}
	return connRow{}, false
}

func connected(conns []connRow, mode string) (connRow, bool) {
	for _, c := range conns {
		if c.Mode == mode && c.Status == "connected" {
			return c, true
		}
	}
	return connRow{}, false
}

// WebhookURL is the connection's webhook address on the public URL.
func (s *Service) WebhookURL(c store.PaymentConnection) string {
	tok := ""
	if c.WebhookToken != nil {
		tok = *c.WebhookToken
	}
	return strings.TrimRight(s.URLs.Base(), "/") + "/hooks/" + c.Provider + "/" + tok
}

func publicHost(base string) string {
	h := strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	if i := strings.IndexAny(h, "/"); i >= 0 {
		h = h[:i]
	}
	return h
}

func newToken() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func permissionChecks(d payments.Descriptor, raw json.RawMessage) []EarnPermissionCheck {
	var done []payments.PermissionCheck
	_ = json.Unmarshal(raw, &done)
	byRes := map[string]bool{}
	for _, p := range done {
		byRes[p.Resource] = p.OK
	}
	out := make([]EarnPermissionCheck, 0, len(d.Permissions))
	for _, p := range d.Permissions {
		pc := EarnPermissionCheck{Resource: p.Resource, Access: p.Access}
		if ok, seen := byRes[p.Resource]; seen {
			v := ok
			pc.OK = &v
		}
		out = append(out, pc)
	}
	return out
}

func catalogCards() []EarnProviderCard {
	out := []EarnProviderCard{}
	for _, d := range payments.Catalog() {
		caps := make([]string, 0, len(d.Capabilities))
		for _, c := range d.Capabilities {
			caps = append(caps, string(c))
		}
		out = append(out, EarnProviderCard{Key: d.Key, Name: d.Name, Available: d.Available, MerchantOfRecord: d.MerchantOfRecord, Capabilities: caps})
	}
	return out
}

func (s *Service) view(ctx context.Context, c connRow, activeSubs int64) EarnConnection {
	d, _ := payments.Lookup(c.Provider)
	mode := payments.Mode(c.Mode)
	out := EarnConnection{ID: c.ID.String(), Provider: c.Provider, ProviderName: d.Name, Mode: c.Mode, Status: c.Status, Step: c.Step, KeyHint: c.KeyHint,
		Account:     EarnAccount{ID: c.AccountID, Name: c.AccountName, Country: c.AccountCountry, Currency: c.Currency, TaxEnabled: c.TaxEnabled},
		Permissions: permissionChecks(d, c.Permissions), CheckedAt: c.CheckedAt, KeyRejectedAt: c.KeyRejectedAt,
		ConnectedAt: c.ConnectedAt, ConnectedBy: c.ConnectedByName, PortalSet: c.PortalConfiguration != "", Demo: c.Demo,
		DashboardURL: dashboard(c.Provider, mode, "", ""), KeysURL: dashboard(c.Provider, mode, "apikeys", ""), CreateKeyURL: dashboard(c.Provider, mode, "apikeys_create", ""),
		WebhooksURL: dashboard(c.Provider, mode, "webhooks", c.WebhookID), EmailsURL: dashboard(c.Provider, mode, "emails", "")}
	for _, p := range out.Permissions {
		if p.OK != nil && *p.OK {
			out.Granted++
		}
	}
	now := s.now()
	w := EarnWebhook{URL: s.WebhookURL(c.PaymentConnection), ID: c.WebhookID, Registered: c.WebhookRegisteredAt != nil, RegisteredAt: c.WebhookRegisteredAt,
		Manual: c.WebhookManual, Events: c.WebhookEvents, SecretStored: len(c.WebhookSecretSealed) > 0, MissingAt: c.WebhookMissingAt,
		FirstEventAt: c.FirstEventAt, FirstEventType: c.FirstEventType, LastEventAt: c.LastEventAt, LastEventType: c.LastEventType,
		GapCheckedAt: c.GapCheckedAt, ByType: []EarnEventCount{}, GapRuns: []EarnGapRun{}}
	if w.Events == nil {
		w.Events = []string{}
	}
	w.Reach.Host = publicHost(s.URLs.Base())
	var reach Reach
	if json.Unmarshal(c.WebhookReach, &reach) == nil && !reach.At.IsZero() {
		w.Reach = EarnReach{OK: reach.OK, Checked: true, At: &reach.At, Code: reach.Code, Reason: reach.Reason, Host: w.Reach.Host}
	}
	_ = json.Unmarshal(c.GapHistory, &w.GapRuns)
	if w.GapRuns == nil {
		w.GapRuns = []EarnGapRun{}
	}
	if c.GapCheckedAt != nil {
		next := c.GapCheckedAt.Truncate(time.Hour).Add(time.Hour)
		w.NextGapAt = &next
	}
	if c.Status == "connected" || c.WebhookRegisteredAt != nil {
		since := now.Add(-24 * time.Hour)
		if rows, err := s.Q.PayEventTypes24h(ctx, store.PayEventTypes24hParams{ConnectionID: &c.ID, Since: since}); err == nil {
			for _, r := range rows {
				w.ByType = append(w.ByType, EarnEventCount{Type: r.Type, Count: r.N})
				w.Events24h += r.N
			}
		}
		w.Rejected24h, _ = s.Q.PayRejectedSince(ctx, store.PayRejectedSinceParams{ConnectionID: &c.ID, Since: since})
	}
	switch {
	case !w.Registered && !w.SecretStored:
		w.Status = "unregistered"
	case c.WebhookMissingAt != nil:
		w.Status = "missing"
	case c.LastEventAt == nil:
		w.Status = "waiting"
	case now.Sub(*c.LastEventAt) > SilentAfter && activeSubs >= SilentMinimum:
		w.Status = "silent"
		w.SilentHours = int(now.Sub(*c.LastEventAt).Hours())
	default:
		w.Status = "receiving"
	}
	out.Webhook = w
	out.Health = health(c, out, w)
	return out
}

func health(c connRow, v EarnConnection, w EarnWebhook) string {
	switch {
	case c.Status != "connected":
		return HealthSetup
	case c.KeyRejectedAt != nil:
		return HealthKeyRejected
	case w.Status == "missing":
		return HealthWebhookMissing
	case v.Granted < len(v.Permissions):
		return HealthMissingPermissions
	case w.Status == "silent":
		return HealthWebhookSilent
	}
	return HealthConnected
}

// Status is the connection chip of the Earn screens.
func (s *Service) Status(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnStatus, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return EarnStatus{}, err
	}
	out := EarnStatus{CanManage: p.Can(authz.PaymentsManage), UpgradeURL: s.pageURL(ctx, ws, "upgrade")}
	for _, c := range conns {
		if c.Status == "setup" && out.Setup == "" {
			out.Setup = c.ID.String()
		}
	}
	c, ok := active(conns)
	if !ok {
		return out, nil
	}
	d, _ := payments.Lookup(c.Provider)
	out.Provider, out.ProviderName, out.Account, out.Mode = c.Provider, d.Name, c.AccountName, c.Mode
	out.TestOnly = c.Mode == "test"
	subs := s.activeSubscriptions(ctx, ws.ID, c.Mode)
	out.Health = s.view(ctx, c, subs).Health
	return out, nil
}

func (s *Service) activeSubscriptions(ctx context.Context, wsID uuid.UUID, mode string) int64 {
	var n int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM paid_subscriptions WHERE workspace_id = $1 AND mode = $2 AND kind = 'stripe'
		AND ended_at IS NULL AND status IN ('active','past_due')`, wsID, mode).Scan(&n)
	return n
}

// Payments is Earn › Settings › Payments.
func (s *Service) Payments(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnPayments, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return EarnPayments{}, err
	}
	d, _ := payments.Lookup(DefaultProvider)
	out := EarnPayments{Providers: catalogCards(), Permissions: permissionChecks(d, nil), WebhookEvents: d.WebhookEvents, Unmapped: []EarnUnmappedPrice{},
		CanManage: p.Can(authz.PaymentsManage), CanEditPrices: p.Can(authz.EarnEdit), Helpers: s.helpers(ctx, ws.ID, authz.PaymentsManage, p.UserID)}
	out.ActiveSubscriptions = s.activeSubscriptions(ctx, ws.ID, "live")
	for _, c := range conns {
		if c.Status != "connected" {
			continue
		}
		v := s.view(ctx, c, s.activeSubscriptions(ctx, ws.ID, c.Mode))
		if c.Mode == "live" {
			out.Live = &v
		} else {
			out.Test = &v
		}
	}
	if c, ok := active(conns); ok {
		out.Active = c.Mode
	}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'test'`, ws.ID).Scan(&out.TestMembers)
	if out.Checkout, err = s.checkoutSettings(ctx, ws, conns); err != nil {
		return out, err
	}
	if out.Unmapped, err = s.unmapped(ctx, ws); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) settingsRow(ctx context.Context, q *store.Queries, wsID uuid.UUID) (store.EarnSetting, error) {
	if err := q.EarnSettingsEnsure(ctx, wsID); err != nil {
		return store.EarnSetting{}, err
	}
	return q.EarnSettings(ctx, wsID)
}

func (s *Service) checkoutSettings(ctx context.Context, ws *store.Workspace, conns []connRow) (EarnCheckoutSettings, error) {
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return EarnCheckoutSettings{}, err
	}
	out := EarnCheckoutSettings{Currency: st.Currency, TaxAtCheckout: st.TaxAtCheckout, PricesIncludeTax: st.PricesIncludeTax, Portal: defaultPortal}
	_ = json.Unmarshal(st.Portal, &out.Portal)
	if c, ok := active(conns); ok {
		out.TaxAvailable = c.TaxEnabled
		if out.Currency == "" {
			out.Currency = c.Currency
		}
	} else {
		for _, c := range conns {
			if out.Currency == "" {
				out.Currency = c.Currency
			}
			out.TaxAvailable = out.TaxAvailable || c.TaxEnabled
		}
	}
	var priced bool
	_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM earn_prices WHERE workspace_id = $1)`, ws.ID).Scan(&priced)
	out.CurrencyFixed = priced
	return out, nil
}

// EarnConnectInput starts a connection with a restricted key (the wizard's
// "Check key").
type EarnConnectInput struct {
	Provider string `json:"provider" enum:"stripe" default:"stripe"`
	Key      string `json:"key" minLength:"1" maxLength:"300" doc:"A restricted key (rk_live_… or rk_test_…). Write-only: it's sealed and never returned."`
}

// keyProblem turns a refused key into the board's sentences. Nothing was
// stored and, for a secret key, nothing was sent.
func keyProblem(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, payments.ErrSecretKey):
		return problem(ctx, http.StatusUnprocessableEntity, CodeSecretKey, "connect.secretKey", nil)
	case errors.Is(err, payments.ErrKeyFormat):
		return problem(ctx, http.StatusUnprocessableEntity, CodeKeyFormat, "connect.keyFormat", nil)
	case errors.Is(err, payments.ErrKeyRejected):
		return problem(ctx, http.StatusUnprocessableEntity, CodeKeyRejected, "connect.keyRejected", nil)
	case errors.Is(err, payments.ErrUnavailable):
		return problem(ctx, http.StatusServiceUnavailable, CodeUnavailable, "connect.unavailable", nil)
	}
	return err
}

// Connect checks a restricted key and keeps the connection in progress:
// the account, its mode (from the key's prefix) and each permission. A key
// that lacks permissions is kept, so "Check again" works once the key is
// edited in Stripe; a refused key is never stored.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, p *authz.Principal, in EarnConnectInput) (EarnConnection, error) {
	key := strings.TrimSpace(in.Key)
	provider := in.Provider
	if provider == "" {
		provider = DefaultProvider
	}
	d, ok := payments.Lookup(provider)
	if !ok || d.Open == nil {
		return EarnConnection{}, problem(ctx, http.StatusUnprocessableEntity, "provider_not_available", "connect.notAvailable", i18n.Args{"provider": d.Name})
	}
	client, err := s.openKey(provider, key, false)
	if err != nil {
		return EarnConnection{}, keyProblem(ctx, err)
	}
	mode := string(client.Mode())
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return EarnConnection{}, err
	}
	if _, ok := connected(conns, mode); ok {
		return EarnConnection{}, problem(ctx, http.StatusConflict, CodeModeConnected, "connect.modeConnected", i18n.Args{"mode": mode})
	}
	chk, err := client.CheckKey(ctx)
	if err != nil {
		return EarnConnection{}, keyProblem(ctx, err)
	}
	now := s.now()
	perms, _ := json.Marshal(chk.Permissions)
	step := StepCheck
	if len(chk.Missing()) == 0 {
		step = StepWebhook
	}
	var id uuid.UUID
	for _, c := range conns {
		if c.Status == "setup" && c.Mode == mode {
			id = c.ID
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if id == uuid.Nil {
			id = newID()
			sealed, err := s.Box.Seal([]byte(key), keyAD(id))
			if err != nil {
				return err
			}
			tok := newToken()
			if err := q.PayConnectionInsert(ctx, store.PayConnectionInsertParams{ID: id, WorkspaceID: ws.ID, Provider: provider, Mode: mode, Step: step,
				KeySealed: sealed, KeyHint: keyHint(provider, key), AccountID: chk.Account.ID, AccountName: chk.Account.Name, AccountCountry: chk.Account.Country,
				Currency: chk.Account.Currency, TaxEnabled: chk.Account.TaxEnabled, Permissions: perms, CheckedAt: &now, WebhookToken: &tok,
				CreatedBy: userID(p)}); err != nil {
				return err
			}
		} else {
			sealed, err := s.Box.Seal([]byte(key), keyAD(id))
			if err != nil {
				return err
			}
			if err := q.PayConnectionChecked(ctx, store.PayConnectionCheckedParams{ID: id, KeySealed: sealed, KeyHint: keyHint(provider, key), AccountID: chk.Account.ID,
				AccountName: chk.Account.Name, AccountCountry: chk.Account.Country, Currency: chk.Account.Currency, TaxEnabled: chk.Account.TaxEnabled,
				Permissions: perms, CheckedAt: &now, Step: step}); err != nil {
				return err
			}
		}
		// The key's check is part of connecting; the audit entry comes when
		// it's finished (or a key that lacks permissions is kept).
		return nil
	})
	if err != nil {
		return EarnConnection{}, err
	}
	c, err := s.connection(ctx, ws.ID, id.String())
	if err != nil {
		return EarnConnection{}, err
	}
	return s.view(ctx, c, 0), nil
}

// Connection is one connection's state (the wizard resumes from it).
func (s *Service) Connection(ctx context.Context, ws *store.Workspace, id string) (EarnConnection, error) {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return EarnConnection{}, err
	}
	return s.view(ctx, c, s.activeSubscriptions(ctx, ws.ID, c.Mode)), nil
}

// Recheck checks the stored key's permissions again ("Check again").
func (s *Service) Recheck(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) (EarnConnection, error) {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return EarnConnection{}, err
	}
	client, err := s.open(c.PaymentConnection)
	if err != nil {
		return EarnConnection{}, err
	}
	chk, err := client.CheckKey(ctx)
	now := s.now()
	if errors.Is(err, payments.ErrKeyRejected) {
		_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: c.ID, At: now})
		return EarnConnection{}, keyProblem(ctx, err)
	}
	if err != nil {
		return EarnConnection{}, keyProblem(ctx, err)
	}
	perms, _ := json.Marshal(chk.Permissions)
	step := c.Step
	if c.Status == "setup" && c.Step == StepCheck && len(chk.Missing()) == 0 {
		step = StepWebhook
	}
	if err := s.Q.PayConnectionChecked(ctx, store.PayConnectionCheckedParams{ID: c.ID, KeyHint: c.KeyHint, AccountID: chk.Account.ID, AccountName: chk.Account.Name,
		AccountCountry: chk.Account.Country, Currency: chk.Account.Currency, TaxEnabled: chk.Account.TaxEnabled, Permissions: perms, CheckedAt: &now, Step: step}); err != nil {
		return EarnConnection{}, err
	}
	return s.Connection(ctx, ws, id)
}

// reach checks that the provider can reach the webhook address: localhost
// and private addresses fail up front, then a signed self-test must come
// back through the instance's proxy.
func (s *Service) reach(ctx context.Context, c store.PaymentConnection) Reach {
	url := s.WebhookURL(c)
	tok := ""
	if c.WebhookToken != nil {
		tok = *c.WebhookToken
	}
	if s.Reach != nil {
		return s.Reach(ctx, url, tok)
	}
	if c.Demo {
		return Reach{OK: true, At: s.now()}
	}
	if bad := connect.Preflight(ctx, url, s.now(), s.AllowPrivate); bad != nil {
		return *bad
	}
	return connect.SelfTest(ctx, url, tok, s.now(), s.AllowPrivate)
}

// RegisterWebhook registers the webhook in the provider: the endpoint, its
// events, and the sealed signing secret. When the public URL can't be
// reached from the internet nothing is registered and the state says so
// (the manual path).
func (s *Service) RegisterWebhook(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) (EarnConnection, error) {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return EarnConnection{}, err
	}
	if err := s.registerWebhook(ctx, ws, p, c, false); err != nil {
		return EarnConnection{}, err
	}
	return s.Connection(ctx, ws, id)
}

func (s *Service) registerWebhook(ctx context.Context, ws *store.Workspace, p *authz.Principal, c connRow, again bool) error {
	now := s.now()
	d, _ := payments.Lookup(c.Provider)
	tok := ""
	if c.WebhookToken != nil {
		tok = *c.WebhookToken
	}
	if again || tok == "" {
		tok = newToken()
		c.WebhookToken = &tok
	}
	r := s.reach(ctx, c.PaymentConnection)
	rb, _ := json.Marshal(r)
	if err := s.Q.PayConnectionSetReach(ctx, store.PayConnectionSetReachParams{ID: c.ID, WebhookReach: rb, At: now}); err != nil {
		return err
	}
	if !r.OK {
		if again {
			return problem(ctx, http.StatusConflict, "public_url_unreachable", "connect.unreachable", i18n.Args{"host": publicHost(s.URLs.Base())})
		}
		return nil
	}
	client, err := s.open(c.PaymentConnection)
	if err != nil {
		return err
	}
	w, err := client.RegisterWebhook(ctx, s.WebhookURL(c.PaymentConnection), d.WebhookEvents)
	if err != nil {
		if errors.Is(err, payments.ErrKeyRejected) {
			_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: c.ID, At: now})
		}
		return keyProblem(ctx, err)
	}
	sealed, err := s.Box.Seal([]byte(w.Secret), secretAD(c.ID))
	if err != nil {
		return err
	}
	old := c.WebhookID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PayConnectionSetWebhook(ctx, store.PayConnectionSetWebhookParams{ID: c.ID, WebhookToken: &tok, WebhookID: w.ID, WebhookSecretSealed: sealed,
			WebhookManual: false, WebhookEvents: d.WebhookEvents, At: &now}); err != nil {
			return err
		}
		if c.Status == "setup" && c.Step != StepSettings && c.Step != StepDone {
			if err := q.PayConnectionSetStep(ctx, store.PayConnectionSetStepParams{ID: c.ID, Step: StepWebhook, At: now}); err != nil {
				return err
			}
		}
		if !again {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "payments.webhook_registered", TargetType: "payment_connection",
			TargetID: c.ID.String(), TargetName: connName(c), Metadata: map[string]any{"mode": c.Mode, "events": len(d.WebhookEvents), "replaced": old}})
	})
	if err != nil {
		return err
	}
	if again && old != "" && old != w.ID {
		// The old endpoint goes, if Stripe still has it.
		_ = client.DeleteWebhook(ctx, old)
	}
	return nil
}

func connName(c connRow) string {
	d, _ := payments.Lookup(c.Provider)
	name := d.Name
	if c.AccountName != "" {
		name += " · " + c.AccountName
	}
	if c.Mode == "test" {
		name += " · Test"
	} else {
		name += " · Live"
	}
	return name
}

// EarnWebhookSecretInput is a signing secret pasted by hand (the manual
// path, when the public URL can't be reached).
type EarnWebhookSecretInput struct {
	Secret string `json:"secret" minLength:"10" maxLength:"300" doc:"The endpoint's signing secret (whsec_…). Write-only."`
}

// WebhookSecret stores a signing secret for a webhook added by hand.
func (s *Service) WebhookSecret(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnWebhookSecretInput) (EarnConnection, error) {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return EarnConnection{}, err
	}
	secret := strings.TrimSpace(in.Secret)
	if !strings.HasPrefix(secret, "whsec_") {
		return EarnConnection{}, invalid(ctx, "body.secret", "connect.secretFormat", nil)
	}
	d, _ := payments.Lookup(c.Provider)
	sealed, err := s.Box.Seal([]byte(secret), secretAD(c.ID))
	if err != nil {
		return EarnConnection{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PayConnectionSetWebhook(ctx, store.PayConnectionSetWebhookParams{ID: c.ID, WebhookToken: c.WebhookToken, WebhookID: "", WebhookSecretSealed: sealed,
			WebhookManual: true, WebhookEvents: d.WebhookEvents, At: &now}); err != nil {
			return err
		}
		if c.Status == "setup" && c.Step == StepCheck {
			if err := q.PayConnectionSetStep(ctx, store.PayConnectionSetStepParams{ID: c.ID, Step: StepWebhook, At: now}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "payments.webhook_secret_added", TargetType: "payment_connection",
			TargetID: c.ID.String(), TargetName: connName(c), Metadata: map[string]any{"mode": c.Mode, "manual": true}})
	})
	if err != nil {
		return EarnConnection{}, err
	}
	return s.Connection(ctx, ws, id)
}

// EarnStepInput moves the wizard on (Continue to settings).
type EarnStepInput struct {
	Step string `json:"step" enum:"webhook,settings"`
}

// SetStep moves a connection in progress to a step.
func (s *Service) SetStep(ctx context.Context, ws *store.Workspace, id string, in EarnStepInput) (EarnConnection, error) {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return EarnConnection{}, err
	}
	if c.Status != "setup" {
		return s.Connection(ctx, ws, id)
	}
	if in.Step == StepSettings && len(c.WebhookSecretSealed) == 0 {
		return EarnConnection{}, problem(ctx, http.StatusConflict, "webhook_first", "connect.webhookFirst", nil)
	}
	if err := s.Q.PayConnectionSetStep(ctx, store.PayConnectionSetStepParams{ID: c.ID, Step: in.Step, At: s.now()}); err != nil {
		return EarnConnection{}, err
	}
	return s.Connection(ctx, ws, id)
}

// EarnFinishInput is the wizard's last settings: Stripe Tax at checkout,
// prices with or without tax, and what the portal lets readers do.
type EarnFinishInput struct {
	TaxAtCheckout    bool       `json:"taxAtCheckout"`
	PricesIncludeTax bool       `json:"pricesIncludeTax"`
	Portal           EarnPortal `json:"portal"`
}

// EarnFinished is a finished connection and what came with it.
type EarnFinished struct {
	Connection EarnConnection `json:"connection"`
	// Drafts is how many member emails were created as drafts.
	Drafts int `json:"drafts"`
	// Tiers is how many tiers got their product in the provider.
	Tiers int `json:"tiers"`
}

// Finish saves the checkout settings, sets up the customer portal and
// finishes the connection; the first connection also creates the eight
// member emails as drafts, and drafted tiers get their products.
func (s *Service) Finish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnFinishInput) (EarnFinished, error) {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return EarnFinished{}, err
	}
	if len(c.WebhookSecretSealed) == 0 {
		return EarnFinished{}, problem(ctx, http.StatusConflict, "webhook_first", "connect.webhookFirst", nil)
	}
	if in.TaxAtCheckout && !c.TaxEnabled {
		return EarnFinished{}, invalid(ctx, "body.taxAtCheckout", "connect.taxOff", i18n.Args{"account": c.AccountName})
	}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return EarnFinished{}, err
	}
	first := true
	for _, x := range conns {
		if x.Status == "connected" {
			first = false
		}
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return EarnFinished{}, err
	}
	client, err := s.open(c.PaymentConnection)
	if err != nil {
		return EarnFinished{}, err
	}
	portalID, err := client.ConfigurePortal(ctx, c.PortalConfiguration, s.portalConfig(ctx, ws, in.Portal, st, c.Mode))
	if err != nil {
		return EarnFinished{}, keyProblem(ctx, err)
	}
	portal, _ := json.Marshal(in.Portal)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PayConnectionFinish(ctx, store.PayConnectionFinishParams{ID: c.ID, PortalConfiguration: portalID, ConnectedBy: userID(p), At: now}); err != nil {
			return err
		}
		if err := q.EarnSettingsSetCheckout(ctx, store.EarnSettingsSetCheckoutParams{WorkspaceID: ws.ID, TaxAtCheckout: in.TaxAtCheckout,
			PricesIncludeTax: in.PricesIncludeTax, Portal: portal, Currency: c.Currency, At: now, UpdatedBy: userID(p)}); err != nil {
			return err
		}
		action := "payments.connected"
		if c.Mode == "test" && !first {
			action = "payments.test_key_added"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "payment_connection", TargetID: c.ID.String(),
			TargetName: connName(c), After: map[string]any{"provider": c.Provider, "mode": c.Mode, "account": c.AccountName, "key": c.KeyHint,
				"tax_at_checkout": in.TaxAtCheckout, "prices_include_tax": in.PricesIncludeTax, "webhook_manual": c.WebhookManual}})
	})
	if err != nil {
		return EarnFinished{}, err
	}
	out := EarnFinished{}
	if s.Mailer != nil {
		n, err := s.ensureMemberEmails(ctx, ws, p)
		if err != nil {
			s.log().WarnContext(ctx, "member email drafts", "workspace", ws.ID, "err", err)
		}
		out.Drafts = n
	}
	// Tiers drafted before this connection get their product now.
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err == nil {
		for _, t := range tiers {
			if (c.Mode == "live" && t.StripeProductLive == "") || (c.Mode == "test" && t.StripeProductTest == "") {
				if _, err := s.syncTier(ctx, ws, t.ID); err == nil {
					out.Tiers++
				}
			}
		}
	}
	if out.Connection, err = s.Connection(ctx, ws, id); err != nil {
		return out, err
	}
	return out, nil
}

// portalConfig is what the provider's portal is set up with.
func (s *Service) portalConfig(ctx context.Context, ws *store.Workspace, pc EarnPortal, st store.EarnSetting, mode string) payments.PortalConfig {
	out := payments.PortalConfig{UpdateCard: pc.UpdateCard, Invoices: pc.Invoices, BillingEmail: pc.BillingEmail, SwitchPeriod: pc.SwitchPeriod, Cancel: pc.Cancel,
		ReturnURL: s.pageURL(ctx, ws, "upgrade/welcome"), TermsURL: st.TermsUrl, Products: map[string][]string{}}
	if !pc.SwitchPeriod {
		return out
	}
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return out
	}
	prices, _ := s.Q.EarnPrices(ctx, ws.ID)
	for _, t := range tiers {
		product := t.StripeProductLive
		if mode == "test" {
			product = t.StripeProductTest
		}
		if product == "" || t.Visibility == "archived" {
			continue
		}
		for _, pr := range prices {
			id := pr.StripeLiveID
			if mode == "test" {
				id = pr.StripeTestID
			}
			if pr.TierID == t.ID && pr.Status == "current" && id != "" {
				out.Products[product] = append(out.Products[product], id)
			}
		}
		if len(out.Products[product]) < 2 {
			delete(out.Products, product)
		}
	}
	return out
}

// EarnCheckoutInput changes checkout's settings (Earn › Settings › Payments).
type EarnCheckoutInput struct {
	TaxAtCheckout    bool       `json:"taxAtCheckout"`
	PricesIncludeTax bool       `json:"pricesIncludeTax"`
	Portal           EarnPortal `json:"portal"`
}

// SaveCheckout saves checkout's settings and updates the portal.
func (s *Service) SaveCheckout(ctx context.Context, ws *store.Workspace, p *authz.Principal, in EarnCheckoutInput) (EarnCheckoutSettings, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return EarnCheckoutSettings{}, err
	}
	before, err := s.checkoutSettings(ctx, ws, conns)
	if err != nil {
		return EarnCheckoutSettings{}, err
	}
	if in.TaxAtCheckout && !before.TaxAvailable {
		return EarnCheckoutSettings{}, invalid(ctx, "body.taxAtCheckout", "connect.taxOffAny", nil)
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return EarnCheckoutSettings{}, err
	}
	for _, c := range conns {
		if c.Status != "connected" {
			continue
		}
		client, err := s.open(c.PaymentConnection)
		if err != nil {
			continue
		}
		if id, err := client.ConfigurePortal(ctx, c.PortalConfiguration, s.portalConfig(ctx, ws, in.Portal, st, c.Mode)); err == nil && id != c.PortalConfiguration {
			_ = s.Q.PayConnectionSetPortal(ctx, store.PayConnectionSetPortalParams{ID: c.ID, PortalConfiguration: id, At: s.now()})
		}
	}
	portal, _ := json.Marshal(in.Portal)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnSettingsSetCheckout(ctx, store.EarnSettingsSetCheckoutParams{WorkspaceID: ws.ID, TaxAtCheckout: in.TaxAtCheckout, PricesIncludeTax: in.PricesIncludeTax,
			Portal: portal, Currency: before.Currency, At: now, UpdatedBy: userID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "payments.checkout_settings", TargetType: "earn_settings", TargetID: ws.ID.String(),
			TargetName: "Checkout and tax",
			Before:     map[string]any{"tax_at_checkout": before.TaxAtCheckout, "prices_include_tax": before.PricesIncludeTax, "portal": before.Portal},
			After:      map[string]any{"tax_at_checkout": in.TaxAtCheckout, "prices_include_tax": in.PricesIncludeTax, "portal": in.Portal}})
	})
	if err != nil {
		return EarnCheckoutSettings{}, err
	}
	conns, _ = s.connections(ctx, ws.ID)
	return s.checkoutSettings(ctx, ws, conns)
}

// EarnReplaceKeyInput is a new restricted key for a connection.
type EarnReplaceKeyInput struct {
	Key string `json:"key" minLength:"1" maxLength:"300" doc:"A restricted key of the same account and mode. Write-only."`
}

// ReplaceKey swaps a connection's key for a new one of the same account
// and mode, once its permissions are checked. The webhook stays as it is.
func (s *Service) ReplaceKey(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnReplaceKeyInput) (EarnConnection, error) {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return EarnConnection{}, err
	}
	key := strings.TrimSpace(in.Key)
	client, err := s.openKey(c.Provider, key, c.Demo)
	if err != nil {
		return EarnConnection{}, keyProblem(ctx, err)
	}
	if string(client.Mode()) != c.Mode {
		return EarnConnection{}, invalid(ctx, "body.key", "connect.otherMode", i18n.Args{"mode": c.Mode})
	}
	chk, err := client.CheckKey(ctx)
	if err != nil {
		return EarnConnection{}, keyProblem(ctx, err)
	}
	if c.AccountID != "" && chk.Account.ID != "" && chk.Account.ID != c.AccountID {
		return EarnConnection{}, problem(ctx, http.StatusUnprocessableEntity, CodeOtherAccount, "connect.otherAccount", i18n.Args{"account": c.AccountName})
	}
	if miss := chk.Missing(); len(miss) > 0 {
		return EarnConnection{}, problem(ctx, http.StatusUnprocessableEntity, "missing_permissions", "connect.replaceMissing",
			i18n.Args{"count": len(miss), "first": miss[0].Resource})
	}
	sealed, err := s.Box.Seal([]byte(key), keyAD(c.ID))
	if err != nil {
		return EarnConnection{}, err
	}
	perms, _ := json.Marshal(chk.Permissions)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PayConnectionChecked(ctx, store.PayConnectionCheckedParams{ID: c.ID, KeySealed: sealed, KeyHint: keyHint(c.Provider, key), AccountID: chk.Account.ID,
			AccountName: firstNonEmpty(chk.Account.Name, c.AccountName), AccountCountry: chk.Account.Country, Currency: chk.Account.Currency,
			TaxEnabled: chk.Account.TaxEnabled, Permissions: perms, CheckedAt: &now, Step: c.Step}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "payments.key_replaced", TargetType: "payment_connection", TargetID: c.ID.String(),
			TargetName: connName(c), Before: map[string]any{"key": c.KeyHint}, After: map[string]any{"key": keyHint(c.Provider, key)}, Metadata: map[string]any{"mode": c.Mode}})
	})
	if err != nil {
		return EarnConnection{}, err
	}
	return s.Connection(ctx, ws, id)
}

// RegisterAgain registers a new webhook endpoint (the old one is deleted
// in the provider) and reads what was missed since the last event.
func (s *Service) RegisterAgain(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) (EarnConnection, error) {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return EarnConnection{}, err
	}
	if err := s.registerWebhook(ctx, ws, p, c, true); err != nil {
		return EarnConnection{}, err
	}
	if c.Status == "connected" {
		c, err = s.connection(ctx, ws.ID, id)
		if err == nil {
			if _, err := s.gapCheck(ctx, c.PaymentConnection); err != nil {
				s.log().WarnContext(ctx, "gap check after registering again", "connection", c.ID, "err", err)
			}
		}
	}
	return s.Connection(ctx, ws, id)
}

// EarnDisconnectInput confirms a disconnection by typing the account's name.
type EarnDisconnectInput struct {
	Confirm string `json:"confirm" maxLength:"200" doc:"The account's name, as the dialog asks (Acme Media)"`
}

// Disconnect stops checkout, the portal links and status updates from the
// provider. Subscriptions keep billing there: disconnecting never cancels
// anyone, and paid members keep access until the end of what they paid.
// The key and the signing secret are deleted from ReputeCast.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnDisconnectInput) error {
	c, err := s.connection(ctx, ws.ID, id)
	if err != nil {
		return err
	}
	want := c.AccountName
	if want == "" {
		want = c.KeyHint
	}
	// A test key in a workspace with a live key goes without the typed
	// confirmation (Remove test key).
	needConfirm := c.Status == "connected" && (c.Mode != "test" || !s.hasLive(ctx, ws.ID))
	if needConfirm && !strings.EqualFold(strings.TrimSpace(in.Confirm), strings.TrimSpace(want)) {
		return invalid(ctx, "body.confirm", "connect.confirm", i18n.Args{"account": want})
	}
	if c.WebhookID != "" {
		if client, err := s.open(c.PaymentConnection); err == nil {
			_ = client.DeleteWebhook(ctx, c.WebhookID)
		}
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.PayConnectionDisconnect(ctx, store.PayConnectionDisconnectParams{ID: c.ID, WorkspaceID: ws.ID, At: &now, By: userID(p)})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.NotFound("Payment connection")
		}
		action := "payments.disconnected"
		switch {
		case c.Status == "setup":
			action = "payments.setup_canceled"
		case c.Mode == "test" && s.hasLiveTx(ctx, q, ws.ID):
			action = "payments.test_key_removed"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "payment_connection", TargetID: c.ID.String(),
			TargetName: connName(c), Before: map[string]any{"provider": c.Provider, "mode": c.Mode, "account": c.AccountName, "key": c.KeyHint},
			Metadata: map[string]any{"subscriptions_keep_billing": true}})
	})
}

func (s *Service) hasLive(ctx context.Context, wsID uuid.UUID) bool {
	return s.hasLiveTx(ctx, s.Q, wsID)
}

func (s *Service) hasLiveTx(ctx context.Context, q *store.Queries, wsID uuid.UUID) bool {
	rows, err := q.PayConnections(ctx, wsID)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(rows, func(r store.PayConnectionsRow) bool {
		return r.PaymentConnection.Mode == "live" && r.PaymentConnection.Status == "connected"
	})
}

// EarnTestDataCleared says what clearing test data removed.
type EarnTestDataCleared struct {
	Members int64 `json:"members"`
}

// ClearTestData deletes test paid members, their timelines and test
// checkouts. Contacts stay; live data is never touched.
func (s *Service) ClearTestData(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnTestDataCleared, error) {
	var out EarnTestDataCleared
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnClearTestEvents(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.EarnClearTestCheckouts(ctx, ws.ID); err != nil {
			return err
		}
		n, err := q.EarnClearTestData(ctx, ws.ID)
		if err != nil {
			return err
		}
		out.Members = n
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "payments.test_data_cleared", TargetType: "workspace", TargetID: ws.ID.String(),
			TargetName: ws.Name, Metadata: map[string]any{"test_paid_members": n}})
	})
	return out, err
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
