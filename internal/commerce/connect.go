package commerce

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

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
	"github.com/hostrepute/reputecast/internal/stores/woocommerce"
)

// CommerceBeginInput starts connecting a store: its platform and address.
type CommerceBeginInput struct {
	Platform string `json:"platform" enum:"shopify,woocommerce"`
	Address  string `json:"address" maxLength:"255"`
}

// CommerceKeysInput is a Shopify app's credentials (step 3).
type CommerceKeysInput struct {
	ClientID     string `json:"clientId" maxLength:"100"`
	ClientSecret string `json:"clientSecret" maxLength:"200"`
}

// CommerceWooKeyInput is a WooCommerce REST API key.
type CommerceWooKeyInput struct {
	Key    string `json:"key" maxLength:"100"`
	Secret string `json:"secret" maxLength:"100"`
}

// CommerceSyncInput is what syncs (Shopify's step 5; WooCommerce's finish).
type CommerceSyncInput struct {
	Products  bool `json:"products"`
	Customers bool `json:"customers"`
	// ListID is an existing list; NewList creates one named after the store.
	ListID    string `json:"listId,omitempty" maxLength:"36"`
	NewList   bool   `json:"newList,omitempty"`
	Orders    bool   `json:"orders"`
	Months    int    `json:"months" enum:"3,6,12"`
	Checkouts bool   `json:"checkouts"`
	WriteBack bool   `json:"writeBack"`
	// WithoutPlugin connects a WooCommerce store without the ReputeCast
	// plugin: products and orders only.
	WithoutPlugin bool `json:"withoutPlugin,omitempty"`
}

// SampleAddress reports whether a store address is one of the names
// reserved for examples (RFC 2606): no real store answers there, so the
// connect dialog's sample values connect a sample store.
func SampleAddress(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	for _, d := range []string{"example.com", "example.net", "example.org"} {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return strings.HasSuffix(host, ".example")
}

// problem maps a platform error onto the API's answer.
func (s *Service) problem(ctx context.Context, platform string, err error) error {
	l := i18n.LangFrom(ctx)
	if wait, ok := stores.IsRateLimited(err); ok {
		return api.Err(http.StatusTooManyRequests, "store_rate_limited", mStoreRateLimited.Text(l, i18n.Args{"platform": platformName(platform)})).
			WithMeta("retryAfter", int(wait.Seconds())+1)
	}
	if errors.Is(err, stores.ErrUnavailable) {
		return api.Err(http.StatusServiceUnavailable, "store_unavailable", mStoreUnavailable.Text(l, i18n.Args{"platform": platformName(platform)}))
	}
	return err
}

// Begin starts connecting a store: the address is checked against the
// platform, and the dialog moves to the app (Shopify) or the key
// (WooCommerce). A store disconnected before reconnects with its data.
func (s *Service) Begin(ctx context.Context, ws *store.Workspace, p *authz.Principal, in CommerceBeginInput) (CommerceConnectState, error) {
	l := i18n.LangFrom(ctx)
	host := stores.NormalizeHost(in.Address)
	if host == "" || !strings.Contains(host, ".") {
		return CommerceConnectState{}, api.Invalid("body.address", mAddressMissing.Text(l, nil))
	}
	existing, err := s.Q.CommerceStoreByAddress(ctx, store.CommerceStoreByAddressParams{WorkspaceID: ws.ID, Platform: in.Platform, Address: host})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return CommerceConnectState{}, err
	}
	found := err == nil
	if found && (existing.Status == StatusConnected || existing.Status == StatusImporting) {
		return CommerceConnectState{}, api.Conflict(mStoreConnected.Text(l, i18n.Args{"name": existing.Name}))
	}
	// The demo's stores, and sample addresses (example.com and the other
	// names reserved for examples), answer from in-process fakes.
	demo := found && existing.Demo || SampleAddress(host)
	cfg := s.config(in.Platform, host, demo)
	var name, domain, public, currency, step string
	switch in.Platform {
	case Shopify:
		sf, err := shopify.Resolve(ctx, cfg, host)
		if err != nil {
			if errors.Is(err, netguard.ErrBlocked) {
				return CommerceConnectState{}, api.Invalid("body.address", mAddressBlocked.Text(l, i18n.Args{"address": host}))
			}
			return CommerceConnectState{}, api.Invalid("body.address", mAddressNotShopify.Text(l, i18n.Args{"address": host}))
		}
		name, domain, currency, step = sf.Name, sf.Domain, sf.Currency, StepApp
		if host != sf.Domain {
			public = host
		}
		if found && existing.ClientID != "" && len(existing.CredentialsSealed) > 0 {
			// The app exists already: approve it again.
			step = StepInstall
		}
	default:
		pr, err := woocommerce.ProbeSite(ctx, cfg)
		if err != nil {
			if errors.Is(err, netguard.ErrBlocked) {
				return CommerceConnectState{}, api.Invalid("body.address", mAddressBlocked.Text(l, i18n.Args{"address": host}))
			}
			return CommerceConnectState{}, api.Invalid("body.address", mAddressNotWoo.Text(l, i18n.Args{"address": host}))
		}
		name, domain, public, step = pr.Name, host, host, StepKey
	}
	if name == "" {
		name = host
	}
	now := s.now()
	var st store.Store
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if found {
			st, err = q.CommerceStoreRestart(ctx, store.CommerceStoreRestartParams{ID: existing.ID, Step: step, Name: name, Domain: domain,
				PublicDomain: public, Currency: currency, At: now})
		} else {
			st, err = q.CommerceStoreInsert(ctx, store.CommerceStoreInsertParams{ID: newID(), WorkspaceID: ws.ID, Platform: in.Platform, Step: step,
				Name: name, Address: host, Domain: domain, PublicDomain: public, Currency: currency, ApiVersion: apiVersion(in.Platform),
				AppName: appName(in.Platform), WebhookToken: ptr(randomToken(24)), Demo: demo, CreatedBy: userID(p), At: now})
		}
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store.connect_started", TargetType: "store", TargetID: st.ID.String(),
			TargetName: name, After: map[string]any{"platform": in.Platform, "address": host}})
	})
	if err != nil {
		return CommerceConnectState{}, err
	}
	return s.connectState(ctx, ws, st), nil
}

func apiVersion(platform string) string {
	if platform == Shopify {
		return shopify.APIVersion
	}
	return "wc/v3"
}

func appName(platform string) string {
	if platform == Shopify {
		return "ReputeCast"
	}
	return ""
}

// ConnectState is the connect dialog of a store.
func (s *Service) ConnectState(ctx context.Context, ws *store.Workspace, id uuid.UUID) (CommerceConnectState, error) {
	st, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return CommerceConnectState{}, s.notFound(ctx, err)
	}
	return s.connectState(ctx, ws, st), nil
}

func (s *Service) notFound(ctx context.Context, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return api.Err(http.StatusNotFound, api.CodeNotFound, mStoreNotFound.Text(i18n.LangFrom(ctx), nil))
	}
	return err
}

func (s *Service) connectState(ctx context.Context, ws *store.Workspace, st store.Store) CommerceConnectState {
	out := CommerceConnectState{Store: s.summary(ctx, st), Step: st.Step, AppURL: s.AppURL(), RedirectURL: s.CallbackURL(),
		Scopes: slices.Clone(shopify.Scopes), OptionalScope: shopify.OptionalScope, InstalledAt: st.InstalledAt, Granted: st.Scopes,
		WriteGranted: slices.Contains(st.Scopes, shopify.OptionalScope), Error: st.ConnectError, Lists: s.listOptions(ctx, ws.ID),
		NewListName: st.Name, Months: int(st.ImportMonths), KeyHint: st.KeyHint, Demo: st.Demo}
	if out.Granted == nil {
		out.Granted = []string{}
	}
	if st.ClientID != "" {
		out.ClientIDHint = hint(st.ClientID)
	}
	if s.PublicCheck != nil {
		ok, at := s.PublicCheck(ctx)
		out.Reachable = ok
		if !at.IsZero() {
			out.ReachableCheckedAt = &at
		}
	}
	if st.Demo {
		out.Reachable = true
	}
	if hooks, err := s.Q.CommerceWebhooks(ctx, st.ID); err == nil {
		for _, h := range hooks {
			if h.Status == "registered" && h.Position < lifecyclePosition {
				out.Webhooks++
			}
		}
	}
	if st.Platform == WooCommerce {
		out.Plugin = s.pluginState(ctx, st)
	}
	return out
}

// hint is "4f1c…a9e2".
func hint(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:4] + "…" + s[len(s)-4:]
}

func (s *Service) listOptions(ctx context.Context, wsID uuid.UUID) []CommerceListOption {
	out := []CommerceListOption{}
	rows, err := s.Q.CommerceLists(ctx, wsID)
	if err != nil {
		return out
	}
	for _, r := range rows {
		out = append(out, CommerceListOption{ID: r.ID.String(), Name: r.Name, Members: r.Subscribed, Store: r.Origin == "store"})
	}
	return out
}

// load reads a store of the workspace for a change.
func (s *Service) load(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.Store, error) {
	st, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return st, s.notFound(ctx, err)
	}
	return st, nil
}

func (s *Service) wrongStep(ctx context.Context, st store.Store) error {
	return api.Conflict(mStoreWrongStep.Text(i18n.LangFrom(ctx), i18n.Args{"name": st.Name}))
}

// SaveKeys keeps a Shopify app's client ID and its secret (sealed,
// write-only); the dialog moves to the install step.
func (s *Service) SaveKeys(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in CommerceKeysInput) (CommerceConnectState, error) {
	l := i18n.LangFrom(ctx)
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceConnectState{}, err
	}
	if st.Platform != Shopify || !slices.Contains([]string{StepApp, StepKeys, StepInstall}, st.Step) {
		return CommerceConnectState{}, s.wrongStep(ctx, st)
	}
	clientID, secret := strings.TrimSpace(in.ClientID), strings.TrimSpace(in.ClientSecret)
	if clientID == "" {
		return CommerceConnectState{}, api.Invalid("body.clientId", mKeysClientID.Text(l, nil))
	}
	if !strings.HasPrefix(secret, "shpss_") || len(secret) < 12 {
		return CommerceConnectState{}, api.Invalid("body.clientSecret", mKeysSecretFormat.Text(l, nil))
	}
	sealed, err := s.seal(st.ID, credentials{ClientSecret: secret})
	if err != nil {
		return CommerceConnectState{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CommerceStoreSetKeys(ctx, store.CommerceStoreSetKeysParams{ID: st.ID, ClientID: clientID, CredentialsSealed: sealed, KeyHint: "",
			Step: StepInstall, At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store.keys_saved", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, After: map[string]any{"clientId": hint(clientID), "clientSecret": "sealed"}})
	})
	if err != nil {
		return CommerceConnectState{}, err
	}
	return s.ConnectState(ctx, ws, st.ID)
}

// Install returns Shopify's approval page for the app. The state names the
// person who started it; it works once, for 30 minutes.
func (s *Service) Install(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (CommerceInstall, error) {
	l := i18n.LangFrom(ctx)
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceInstall{}, err
	}
	if st.Platform != Shopify || !slices.Contains([]string{StepInstall, StepKeys, StepSync}, st.Step) {
		return CommerceInstall{}, s.wrongStep(ctx, st)
	}
	if st.ClientID == "" || len(st.CredentialsSealed) == 0 {
		return CommerceInstall{}, api.Conflict(mKeysMissing.Text(l, nil))
	}
	by := "-"
	if u := userID(p); u != nil {
		by = u.String()
	}
	state := randomToken(24) + "~" + by
	now := s.now()
	if err := s.Q.CommerceStoreSetState(ctx, store.CommerceStoreSetStateParams{ID: st.ID, OauthState: &state, ExpiresAt: ptr(now.Add(stateTTL)), At: now}); err != nil {
		return CommerceInstall{}, err
	}
	scopes := append(slices.Clone(shopify.Scopes), shopify.OptionalScope)
	if st.Demo && s.DemoApprove != nil {
		// The demo's store approves at once, in process.
		return CommerceInstall{URL: s.DemoApprove(s.CallbackURL(), state)}, nil
	}
	return CommerceInstall{URL: shopify.AuthorizeURL(st.Domain, st.ClientID, s.CallbackURL(), state, scopes)}, nil
}

// stateUser is the person who started the approval.
func stateUser(state string) *uuid.UUID {
	_, after, ok := strings.Cut(state, "~")
	if !ok {
		return nil
	}
	id, err := uuid.Parse(after)
	if err != nil {
		return nil
	}
	return &id
}

// Installed finishes Shopify's approval (the redirect URL): the code is
// exchanged for the access token with the app's secret. A reconnecting
// store goes straight back to syncing; a new one to step 5.
func (s *Service) Installed(ctx context.Context, q map[string][]string) (store.Store, string, error) {
	vals := url.Values(q)
	st, err := s.Q.CommerceStoreByState(ctx, ptr(vals.Get("state")))
	if err != nil {
		return st, "state_expired", err
	}
	now := s.now()
	if st.OauthStateExpiresAt == nil || now.After(*st.OauthStateExpiresAt) {
		_ = s.Q.CommerceStoreSetStep(ctx, store.CommerceStoreSetStepParams{ID: st.ID, Step: StepInstall, ConnectError: "state_expired", At: now})
		return st, "state_expired", nil
	}
	if stores.NormalizeHost(vals.Get("shop")) != st.Domain {
		return st, "state_expired", nil
	}
	creds, err := s.credentials(st)
	if err != nil {
		return st, "secret_refused", nil //nolint:nilerr // keys that can't be opened are pasted again
	}
	by := stateUser(*st.OauthState)
	refuse := func() (store.Store, string, error) {
		_ = s.Q.CommerceStoreSetStep(ctx, store.CommerceStoreSetStepParams{ID: st.ID, Step: StepKeys, ConnectError: "secret_refused", At: now})
		return st, "secret_refused", nil
	}
	// Shopify signs the redirect with the app's real secret: a pasted
	// secret that doesn't verify it is the wrong one.
	if !st.Demo && !shopify.VerifyQuery(vals, creds.ClientSecret, now) {
		return refuse()
	}
	cfg := s.config(Shopify, st.Domain, st.Demo)
	tok, err := shopify.Exchange(ctx, cfg, st.ClientID, creds.ClientSecret, vals.Get("code"))
	if errors.Is(err, stores.ErrUnauthorized) {
		return refuse()
	}
	if err != nil {
		_ = s.Q.CommerceStoreSetStep(ctx, store.CommerceStoreSetStepParams{ID: st.ID, Step: StepInstall, ConnectError: "", At: now})
		return st, "unavailable", err
	}
	creds.AccessToken, creds.RefreshToken, creds.ExpiresAt = tok.AccessToken, tok.RefreshToken, tok.ExpiresAt
	sealed, err := s.seal(st.ID, creds)
	if err != nil {
		return st, "unavailable", err
	}
	cfg.Token = tok.AccessToken
	name, currency, public := "", "", ""
	if c, err := shopify.New(cfg); err == nil {
		if shop, err := c.Shop(ctx); err == nil {
			name, currency = shop.Name, shop.Currency
			if shop.PublicDomain != "" && shop.PublicDomain != st.Domain {
				public = shop.PublicDomain
			}
		}
	}
	reconnect := st.ImportFinishedAt != nil
	step := StepSync
	if reconnect {
		step = StepDone
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CommerceStoreInstalled(ctx, store.CommerceStoreInstalledParams{ID: st.ID, CredentialsSealed: sealed, Scopes: tok.Scopes, At: now,
			Step: step, Name: name, Currency: currency, PublicDomain: public}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &st.WorkspaceID, Action: "store.installed", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, Actor: s.actor(ctx, by), After: map[string]any{"scopes": tok.Scopes}})
	})
	if err != nil {
		return st, "unavailable", err
	}
	st, err = s.Q.CommerceStoreByID(ctx, st.ID)
	if err != nil {
		return st, "unavailable", err
	}
	if reconnect {
		if err := s.reconnected(ctx, st, by); err != nil {
			s.log().WarnContext(ctx, "store reconnected, sync not started", "store", st.ID, "err", err)
		}
	}
	return st, "", nil
}

// actor is the person who started something a later request finishes.
func (s *Service) actor(ctx context.Context, id *uuid.UUID) *audit.Actor {
	if id == nil {
		return nil
	}
	return &audit.Actor{Type: authz.ActorUser, ID: id, Name: s.userName(ctx, id)}
}

// StartSync is Shopify's step 5: what syncs, the list, then the webhooks
// and the first import.
func (s *Service) StartSync(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in CommerceSyncInput) (CommerceConnectState, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceConnectState{}, err
	}
	switch {
	case st.Platform == Shopify && st.Step == StepSync:
	case st.Platform == WooCommerce && st.Step == StepPlugin:
		if !st.PluginFound && !in.WithoutPlugin {
			return CommerceConnectState{}, s.wrongStep(ctx, st)
		}
	default:
		return CommerceConnectState{}, s.wrongStep(ctx, st)
	}
	if err := s.start(ctx, ws, p, st, in); err != nil {
		return CommerceConnectState{}, err
	}
	return s.ConnectState(ctx, ws, st.ID)
}

// start registers the webhooks and starts the first import.
func (s *Service) start(ctx context.Context, ws *store.Workspace, p *authz.Principal, st store.Store, in CommerceSyncInput) error {
	l := i18n.LangFrom(ctx)
	months := in.Months
	if months != 3 && months != 6 {
		months = 12
	}
	writeBack := in.WriteBack && st.Platform == Shopify && slices.Contains(st.Scopes, shopify.OptionalScope)
	customers := in.Customers
	var listID *uuid.UUID
	newList := false
	if customers {
		switch {
		case in.ListID != "":
			lid, err := uuid.Parse(in.ListID)
			if err != nil || !slices.ContainsFunc(s.listOptions(ctx, ws.ID), func(o CommerceListOption) bool { return o.ID == lid.String() }) {
				return api.Invalid("body.listId", mListUnknown.Text(l, nil))
			}
			listID = &lid
		default:
			if _, err := s.Q.CommerceListByName(ctx, store.CommerceListByNameParams{WorkspaceID: ws.ID, Name: st.Name}); err == nil {
				return api.Invalid("body.listId", mListExists.Text(l, i18n.Args{"name": st.Name}))
			}
			lid := newID()
			listID, newList = &lid, true
		}
	}
	by := userID(p)
	if err := s.registerWebhooks(ctx, st, by, false); err != nil {
		return s.problem(ctx, st.Platform, err)
	}
	now := s.now()
	progress := importProgress{Phase: "products", Months: months, By: s.userName(ctx, by), CustomersSkipped: st.Platform == WooCommerce}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if newList {
			if err := q.CommerceListCreate(ctx, store.CommerceListCreateParams{ID: *listID, WorkspaceID: ws.ID, Name: st.Name,
				Description: "Customers of " + st.Name + " who accepted marketing", At: now}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "list.create", TargetType: "list", TargetID: listID.String(),
				TargetName: st.Name, After: map[string]any{"name": st.Name, "store": st.Name}}); err != nil {
				return err
			}
		}
		started, err := q.CommerceStoreStartSync(ctx, store.CommerceStoreStartSyncParams{ID: st.ID, SyncProducts: in.Products || st.Platform == WooCommerce,
			SyncCustomers: customers, SyncOrders: in.Orders || st.Platform == WooCommerce, SyncCheckouts: in.Checkouts || st.Platform == WooCommerce,
			ImportMonths: int32(months), ListID: listID, WriteBack: writeBack, PluginSkipped: in.WithoutPlugin, ImportProgress: mustJSON(progress),
			By: by, At: now})
		if err != nil {
			return err
		}
		after := map[string]any{"platform": st.Platform, "address": st.Address, "products": started.SyncProducts, "customers": customers,
			"orders": started.SyncOrders, "months": months, "checkouts": started.SyncCheckouts, "writeBack": writeBack}
		if listID != nil {
			after["list"] = listID.String()
		}
		if in.WithoutPlugin {
			after["withoutPlugin"] = true
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store.connected", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, After: after}); err != nil {
			return err
		}
		if err := s.activity(ctx, q, st, activity{Kind: "import_started", Tone: "info", By: s.userName(ctx, by)}); err != nil {
			return err
		}
		if jc := s.jobs(); jc != nil {
			if _, err := jc.InsertTx(ctx, tx, ImportArgs{StoreID: st.ID.String()}, nil); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveWooKey checks a WooCommerce REST API key, seals it and registers the
// webhooks with a new secret; the dialog moves to the plugin check.
func (s *Service) SaveWooKey(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in CommerceWooKeyInput) (CommerceConnectState, error) {
	l := i18n.LangFrom(ctx)
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceConnectState{}, err
	}
	if st.Platform != WooCommerce || !slices.Contains([]string{StepKey, StepWebhooks, StepPlugin}, st.Step) {
		return CommerceConnectState{}, s.wrongStep(ctx, st)
	}
	key, secret := strings.TrimSpace(in.Key), strings.TrimSpace(in.Secret)
	if !woocommerce.KeyFormat(key, secret) {
		return CommerceConnectState{}, api.Invalid("body.key", mWooKeyFormat.Text(l, nil))
	}
	cfg := s.config(WooCommerce, st.Domain, st.Demo)
	cfg.Key, cfg.Secret = key, secret
	c, err := woocommerce.New(cfg)
	if err != nil {
		return CommerceConnectState{}, err
	}
	shop, err := c.Shop(ctx)
	if errors.Is(err, stores.ErrUnauthorized) || errors.Is(err, stores.ErrPermission) {
		return CommerceConnectState{}, api.Invalid("body.key", mWooKeyRefused.Text(l, nil))
	}
	if err != nil {
		return CommerceConnectState{}, s.problem(ctx, WooCommerce, err)
	}
	sealed, err := s.seal(st.ID, credentials{Key: key, Secret: secret})
	if err != nil {
		return CommerceConnectState{}, err
	}
	whSecret, err := s.Box.SealString(randomToken(32), webhookAD(st.ID))
	if err != nil {
		return CommerceConnectState{}, err
	}
	now := s.now()
	if err := s.Q.CommerceStoreSetKeys(ctx, store.CommerceStoreSetKeysParams{ID: st.ID, ClientID: "", CredentialsSealed: sealed, KeyHint: hint(key),
		Step: StepWebhooks, At: now}); err != nil {
		return CommerceConnectState{}, err
	}
	if err := s.Q.CommerceStoreSetWebhookSecret(ctx, store.CommerceStoreSetWebhookSecretParams{ID: st.ID, WebhookSecretSealed: whSecret, At: now}); err != nil {
		return CommerceConnectState{}, err
	}
	if shop.Currency != "" || shop.Name != "" {
		_, _ = s.Q.CommerceStoreRestart(ctx, store.CommerceStoreRestartParams{ID: st.ID, Step: StepWebhooks, Name: shop.Name, Domain: st.Domain,
			PublicDomain: st.PublicDomain, Currency: shop.Currency, At: now})
	}
	st, _ = s.Q.CommerceStoreByID(ctx, st.ID)
	if err := s.registerWebhooks(ctx, st, userID(p), false); err != nil {
		if errors.Is(err, stores.ErrUnauthorized) || errors.Is(err, stores.ErrPermission) {
			_ = s.Q.CommerceStoreSetStep(ctx, store.CommerceStoreSetStepParams{ID: st.ID, Step: StepKey, ConnectError: "", At: now})
			return CommerceConnectState{}, api.Invalid("body.key", mWooReadOnly.Text(l, nil))
		}
		return CommerceConnectState{}, s.problem(ctx, WooCommerce, err)
	}
	found := s.checkPlugin(ctx, st)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CommerceStoreSetStep(ctx, store.CommerceStoreSetStepParams{ID: st.ID, Step: StepPlugin, ConnectError: "", At: now}); err != nil {
			return err
		}
		if err := q.CommerceStoreSetPlugin(ctx, store.CommerceStoreSetPluginParams{ID: st.ID, PluginFound: found, At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store.keys_saved", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, After: map[string]any{"key": hint(key), "secret": "sealed"}})
	})
	if err != nil {
		return CommerceConnectState{}, err
	}
	return s.ConnectState(ctx, ws, st.ID)
}

// CheckPlugin looks for the ReputeCast plugin on a WooCommerce store's site
// ("Check again").
func (s *Service) CheckPlugin(ctx context.Context, ws *store.Workspace, id uuid.UUID) (CommerceConnectState, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceConnectState{}, err
	}
	if st.Platform != WooCommerce {
		return CommerceConnectState{}, s.wrongStep(ctx, st)
	}
	found := s.checkPlugin(ctx, st)
	if err := s.Q.CommerceStoreSetPlugin(ctx, store.CommerceStoreSetPluginParams{ID: st.ID, PluginFound: found, At: s.now()}); err != nil {
		return CommerceConnectState{}, err
	}
	return s.ConnectState(ctx, ws, st.ID)
}

// checkPlugin: the plugin's REST namespace answers on the site, or a
// WordPress site connected to the workspace at the same address reports
// WooCommerce with the plugin's checkout features.
func (s *Service) checkPlugin(ctx context.Context, st store.Store) bool {
	if s.Sites != nil {
		if site, ok := s.Sites(ctx, st.WorkspaceID, st.Domain); ok && site.WooCommerce != "" && site.Consent {
			return true
		}
	}
	pr, err := woocommerce.ProbeSite(ctx, s.config(WooCommerce, st.Domain, st.Demo))
	return err == nil && pr.Plugin
}

func (s *Service) pluginState(ctx context.Context, st store.Store) *CommercePluginState {
	out := &CommercePluginState{Found: st.PluginFound, Skipped: st.PluginSkipped, Host: st.Domain, CheckedAt: st.PluginCheckedAt}
	if s.Sites != nil {
		if site, ok := s.Sites(ctx, st.WorkspaceID, st.Domain); ok {
			out.Version, out.Consent, out.Checkouts, out.SiteID = site.PluginVersion, site.Consent, site.Checkouts, site.ID.String()
		}
	}
	if out.Found && !out.Consent && !out.Checkouts {
		// Found through the site's REST index: both features come with it.
		out.Consent, out.Checkouts = true, true
	}
	return out
}

// Reconnect starts connecting a disconnected store again: Shopify asks for
// approval again (or the app's keys, when a person disconnected it),
// WooCommerce for a key.
func (s *Service) Reconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (CommerceConnectState, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceConnectState{}, err
	}
	if st.Status != StatusDisconnected {
		return CommerceConnectState{}, s.wrongStep(ctx, st)
	}
	step := StepKey
	if st.Platform == Shopify {
		step = StepKeys
		if st.ClientID != "" && len(st.CredentialsSealed) > 0 {
			step = StepInstall
		}
	}
	if err := s.Q.CommerceStoreSetStep(ctx, store.CommerceStoreSetStepParams{ID: st.ID, Step: step, ConnectError: "", At: s.now()}); err != nil {
		return CommerceConnectState{}, err
	}
	return s.ConnectState(ctx, ws, st.ID)
}

// reconnected brings a store that had finished its first import back:
// webhooks registered again, automations it paused resumed, and a
// reconciliation fetches what changed while it was disconnected.
func (s *Service) reconnected(ctx context.Context, st store.Store, by *uuid.UUID) error {
	if err := s.registerWebhooks(ctx, st, by, false); err != nil {
		return err
	}
	now := s.now()
	since := now.Add(-24 * time.Hour)
	if st.DisconnectedAt != nil {
		since = st.DisconnectedAt.Add(-time.Hour)
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		st2, err := q.CommerceStoreReconnected(ctx, store.CommerceStoreReconnectedParams{ID: st.ID, At: now})
		if err != nil {
			return err
		}
		if err := s.resumeFlows(ctx, q, st2); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &st.WorkspaceID, Action: "store.reconnected", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, Actor: s.actor(ctx, by)}); err != nil {
			return err
		}
		if err := s.activity(ctx, q, st, activity{Kind: "reconnected", Tone: "ok", By: s.userName(ctx, by)}); err != nil {
			return err
		}
		if jc := s.jobs(); jc != nil {
			if _, err := jc.InsertTx(ctx, tx, ReconcileArgs{StoreID: st.ID.String(), Since: since}, nil); err != nil {
				return err
			}
		}
		return nil
	})
}

// FinishWoo connects a WooCommerce store once its key and webhooks are in:
// with the plugin, or without it ("Connect without the plugin").
func (s *Service) FinishWoo(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, withoutPlugin bool) (CommerceConnectState, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceConnectState{}, err
	}
	if st.Platform != WooCommerce || st.Step != StepPlugin {
		return CommerceConnectState{}, s.wrongStep(ctx, st)
	}
	if st.ImportFinishedAt != nil {
		if err := s.reconnected(ctx, st, userID(p)); err != nil {
			return CommerceConnectState{}, s.problem(ctx, WooCommerce, err)
		}
		return s.ConnectState(ctx, ws, st.ID)
	}
	in := CommerceSyncInput{Products: true, Orders: true, Months: 12, Checkouts: true, Customers: true, WithoutPlugin: withoutPlugin}
	if lid := s.existingStoreList(ctx, ws.ID, st.Name); lid != "" {
		in.ListID = lid
	}
	return s.StartSync(ctx, ws, p, id, in)
}

// existingStoreList is a list a store connection made before with this name.
func (s *Service) existingStoreList(ctx context.Context, wsID uuid.UUID, name string) string {
	for _, o := range s.listOptions(ctx, wsID) {
		if o.Store && strings.EqualFold(o.Name, name) {
			return o.ID
		}
	}
	return ""
}
