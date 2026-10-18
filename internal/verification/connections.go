package verification

import (
	"context"
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
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

// VerificationMapping is one row of "How ZeroBounce's answers map".
type VerificationMapping struct {
	Codes  []string `json:"codes"`
	Result string   `json:"result" enum:"valid,invalid,risky,unknown"`
	Reason string   `json:"reason,omitempty"`
	// Tag is what a risky or unknown answer tags the contact with.
	Tag string `json:"tag,omitempty"`
}

// VerificationCapabilities are a service's catalog badges.
type VerificationCapabilities struct {
	BulkLists     bool `json:"bulkLists"`
	RealTime      bool `json:"realTime"`
	CatchAll      bool `json:"catchAll"`
	CreditBalance bool `json:"creditBalance"`
	DataRegion    bool `json:"dataRegion"`
}

// VerificationRegion is a data region a service offers.
type VerificationRegion struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// VerificationServiceView is one service on the Verification services
// page: what it can do and, when connected, its settings and health. The
// API key is never returned.
type VerificationServiceView struct {
	Key          string                   `json:"key" enum:"zerobounce,kickbox,neverbounce,bouncer,emailable"`
	Name         string                   `json:"name"`
	Initials     string                   `json:"initials"`
	Capabilities VerificationCapabilities `json:"capabilities"`
	Regions      []VerificationRegion     `json:"regions"`
	Website      string                   `json:"website"`
	KeyHelp      string                   `json:"keyHelp"`
	Mappings     []VerificationMapping    `json:"mappings"`
	Connected    bool                     `json:"connected"`
	// Status: not_connected, connected, low_credits (under 1,000 or less
	// than a waiting job needs) or error (the last call failed).
	Status string `json:"status" enum:"not_connected,connected,low_credits,error"`
	// Region and RegionLabel: where addresses are processed ("United States").
	Region       string     `json:"region"`
	RegionLabel  string     `json:"regionLabel"`
	Price        float64    `json:"price" doc:"Price per check in dollars, from the workspace's plan"`
	PriceBy      string     `json:"priceBy,omitempty" doc:"Who entered the price"`
	DefaultBulk  bool       `json:"defaultBulk"`
	DefaultForms bool       `json:"defaultForms"`
	Balance      *int64     `json:"balance,omitempty"`
	BalanceAt    *time.Time `json:"balanceAt,omitempty"`
	ConnectedBy  string     `json:"connectedBy,omitempty"`
	ConnectedAt  *time.Time `json:"connectedAt,omitempty"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
	LastUsedBy   string     `json:"lastUsedBy,omitempty"`
	LastUsedWhat string     `json:"lastUsedWhat,omitempty"`
	// The last error, when it happened and since when the service failed.
	LastError      string     `json:"lastError,omitempty"`
	LastErrorAt    *time.Time `json:"lastErrorAt,omitempty"`
	LastErrorSince *time.Time `json:"lastErrorSince,omitempty"`
	// Down: the service failed and hasn't answered since.
	Down            bool       `json:"down"`
	LastSuccessAt   *time.Time `json:"lastSuccessAt,omitempty"`
	LastSuccessWhat string     `json:"lastSuccessWhat,omitempty"`
	TestedAt        *time.Time `json:"testedAt,omitempty"`
	TestOK          *bool      `json:"testOk,omitempty"`
	// Last 30 days: checks, bulk uses and what they cost.
	Checks30d   int64   `json:"checks30d"`
	BulkUses30d int64   `json:"bulkUses30d"`
	Spent30d    float64 `json:"spent30d"`
	// Needed is a waiting job's size when the balance can't cover it.
	Needed      int64  `json:"needed,omitempty"`
	NeededLabel string `json:"neededLabel,omitempty"`
	// PausedJob is a job waiting for the service to answer again.
	PausedJob string `json:"pausedJob,omitempty"`
}

// VerificationUse is one use of a service: a bulk job, or a service's
// real-time checks on signup forms since it became the forms default.
type VerificationUse struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind" enum:"import,list,segment,tier,selection,contact,realtime"`
	Service     string     `json:"service"`
	ServiceName string     `json:"serviceName"`
	At          time.Time  `json:"at"`
	By          string     `json:"by"`
	Label       string     `json:"label"`
	Status      string     `json:"status"`
	Addresses   int64      `json:"addresses"`
	Cost        float64    `json:"cost"`
	Valid       int64      `json:"valid"`
	Invalid     int64      `json:"invalid"`
	Risky       int64      `json:"risky"`
	Unknown     int64      `json:"unknown"`
	Suppressed  int64      `json:"suppressed"`
	NoAnswer    int64      `json:"noAnswer"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
	// Forms: the real-time row's checks per form ("Newsletter signup (338)").
	Forms []VerificationFormCount `json:"forms"`
}

// VerificationFormCount is one form's real-time checks.
type VerificationFormCount struct {
	FormKey  string `json:"formKey"`
	Name     string `json:"name"`
	Checks   int64  `json:"checks"`
	Valid    int64  `json:"valid"`
	Invalid  int64  `json:"invalid"`
	Risky    int64  `json:"risky"`
	Unknown  int64  `json:"unknown"`
	NoAnswer int64  `json:"noAnswer"`
}

// VerificationOverview is Settings › Integrations › Verification services.
type VerificationOverview struct {
	Services     []VerificationServiceView `json:"services"`
	DefaultBulk  string                    `json:"defaultBulk,omitempty"`
	DefaultForms string                    `json:"defaultForms,omitempty"`
	// Uses of the last 30 days, newest first.
	Uses []VerificationUse `json:"uses"`
	// CanManage: integrations:manage (connect, change, remove).
	CanManage     bool     `json:"canManage"`
	ManageHelpers []string `json:"manageHelpers"`
	// CanVerify: subscribers:verify (send addresses to a service).
	CanVerify     bool     `json:"canVerify"`
	VerifyHelpers []string `json:"verifyHelpers"`
}

func view(d adapter.Descriptor) VerificationServiceView {
	v := VerificationServiceView{
		Key: d.Key, Name: d.Name, Initials: d.Initials, Website: d.Website, KeyHelp: d.KeyHelp,
		Capabilities: VerificationCapabilities(d.Capabilities), Regions: []VerificationRegion{}, Mappings: []VerificationMapping{},
		Status: "not_connected", RegionLabel: d.DefaultRegion,
	}
	for _, r := range d.Regions {
		v.Regions = append(v.Regions, VerificationRegion(r))
	}
	for _, m := range d.Mappings {
		row := VerificationMapping{Codes: m.Codes, Result: string(m.Result), Reason: m.Reason}
		row.Tag = adapter.Answer{Result: m.Result, Reason: m.Reason}.Tag()
		v.Mappings = append(v.Mappings, row)
	}
	return v
}

// Helpers names up to three active members who hold perm, narrowest role
// first and the Owner last ("Ask Sofia Rossi, Marcus Chen or Linh Nguyen").
func Helpers(ctx context.Context, q *store.Queries, wsID uuid.UUID, perm authz.Permission) []string {
	out := []string{}
	rows, err := q.ListMembersWithRoles(ctx, wsID)
	if err != nil {
		return out
	}
	rank := func(key string) int {
		switch key {
		case authz.RoleOwner:
			return 2
		case authz.RoleAdmin:
			return 1
		}
		return 0
	}
	slices.SortStableFunc(rows, func(a, b store.ListMembersWithRolesRow) int { return rank(a.RoleKey) - rank(b.RoleKey) })
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			out = append(out, r.Name)
			if len(out) == 3 {
				break
			}
		}
	}
	return out
}

// Overview lists the catalog with the workspace's connections and uses.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace) (VerificationOverview, error) {
	p := api.PrincipalFrom(ctx)
	out := VerificationOverview{Services: []VerificationServiceView{}, Uses: []VerificationUse{},
		CanManage: p.Can(authz.IntegrationsManage), CanVerify: p.Can(authz.SubscribersVerify),
		ManageHelpers: Helpers(ctx, s.Q, ws.ID, authz.IntegrationsManage), VerifyHelpers: Helpers(ctx, s.Q, ws.ID, authz.SubscribersVerify)}
	rows, err := s.Q.VerifServiceList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	since := s.now().Add(-30 * 24 * time.Hour)
	jobs, err := s.Q.VerifJobList(ctx, store.VerifJobListParams{WorkspaceID: ws.ID, Since: since, Lim: 200})
	if err != nil {
		return out, err
	}
	by := map[string]store.VerifServiceListRow{}
	for _, r := range rows {
		v := r.VerificationService
		by[v.Service] = r
		if v.DefaultBulk {
			out.DefaultBulk = v.Service
		}
		if v.DefaultForms {
			out.DefaultForms = v.Service
		}
	}
	for _, d := range adapter.Catalog() {
		v := view(d)
		if r, ok := by[d.Key]; ok {
			s.fill(&v, r)
			for _, jr := range jobs {
				j := jr.VerificationJob
				if j.Service != d.Key || j.CreatedAt.Before(since) {
					continue
				}
				v.Checks30d += int64(j.Checked)
				v.BulkUses30d++
				v.Spent30d += cost(int64(j.Checked), j.PriceMicros)
				if j.Status == "stopped" && j.StopReason == "no_credits" {
					left := int64(j.Total - j.Checked)
					if v.Balance != nil && *v.Balance < left && int64(j.Total) > v.Needed {
						v.Needed, v.NeededLabel = int64(j.Total), j.Label
					}
				}
				if j.Status == "paused" && v.PausedJob == "" {
					v.PausedJob = j.ID.String()
				}
			}
			if v.Status == "connected" && (v.Balance != nil && *v.Balance < LowCredits || v.Needed > 0) {
				v.Status = "low_credits"
			}
			if r.VerificationService.DefaultForms {
				rt, err := s.realtimeUse(ctx, ws, r.VerificationService, since)
				if err != nil {
					return out, err
				}
				if rt != nil {
					v.Checks30d += rt.Addresses
					v.Spent30d += rt.Cost
					out.Uses = append(out.Uses, *rt)
				}
			}
		}
		out.Services = append(out.Services, v)
	}
	for _, j := range jobs {
		out.Uses = append(out.Uses, useOf(j.VerificationJob, j.StartedByName))
	}
	slices.SortStableFunc(out.Uses, func(a, b VerificationUse) int { return b.At.Compare(a.At) })
	return out, nil
}

func (s *Service) fill(v *VerificationServiceView, row store.VerifServiceListRow) {
	r := row.VerificationService
	v.Connected = true
	v.Status = "connected"
	if r.Region != "" {
		v.Region = r.Region
		v.RegionLabel = RegionLabel(r.Service, r.Region)
	}
	v.Price = Money(r.PriceMicros).Dollars()
	v.PriceBy = row.UpdatedByName
	v.DefaultBulk, v.DefaultForms = r.DefaultBulk, r.DefaultForms
	v.Balance, v.BalanceAt = r.Balance, r.BalanceAt
	v.ConnectedBy, v.ConnectedAt = row.ConnectedByName, &r.ConnectedAt
	v.LastUsedAt, v.LastUsedBy, v.LastUsedWhat = r.LastUsedAt, row.LastUsedByName, r.LastUsedWhat
	v.LastError, v.LastErrorAt, v.LastErrorSince = r.LastError, r.LastErrorAt, r.LastErrorSince
	v.LastSuccessAt, v.LastSuccessWhat = r.LastSuccessAt, r.LastSuccessWhat
	v.TestedAt, v.TestOK = r.TestedAt, r.TestOk
	// Down while the last error is newer than the last answer.
	if r.LastErrorSince != nil && r.LastErrorAt != nil && (r.LastSuccessAt == nil || r.LastSuccessAt.Before(*r.LastErrorAt)) {
		v.Down = true
		v.Status = "error"
	}
}

func useOf(j store.VerificationJob, by string) VerificationUse {
	u := VerificationUse{ID: j.ID.String(), Kind: j.Kind, Service: j.Service, ServiceName: ServiceName(j.Service), At: j.CreatedAt, By: by,
		Label: j.Label, Status: j.Status, Addresses: int64(j.Total), Valid: int64(j.Valid), Invalid: int64(j.Invalid), Risky: int64(j.Risky),
		Unknown: int64(j.Unknown), Suppressed: int64(j.Suppressed), FinishedAt: j.FinishedAt, Forms: []VerificationFormCount{}}
	n := int64(j.Checked)
	if j.Status == "finished" && n < u.Addresses {
		u.Addresses = n
	}
	u.Cost = cost(max(n, 0), j.PriceMicros)
	if j.Status == "queued" || j.Status == "running" {
		u.Cost = cost(int64(j.Total), j.PriceMicros)
	}
	return u
}

// VerificationKeyInput tests or saves a service's API key.
type VerificationKeyInput struct {
	Service string `json:"service" enum:"zerobounce,kickbox,neverbounce,bouncer,emailable"`
	APIKey  string `json:"apiKey" minLength:"8" maxLength:"500"`
	Region  string `json:"region,omitempty" enum:"us,eu,"`
}

// VerificationKeyTest is "Test connection": the key reads the balance and
// no address is sent.
type VerificationKeyTest struct {
	OK      bool       `json:"ok"`
	Balance *int64     `json:"balance,omitempty"`
	Status  int        `json:"status,omitempty" doc:"The service's HTTP status when it refused"`
	Problem string     `json:"problem,omitempty"`
	At      time.Time  `json:"at"`
	Kind    string     `json:"kind,omitempty" enum:"key_rejected,no_credits,unreachable,"`
	Error   *time.Time `json:"error,omitempty"`
}

// TestKey checks a key by reading the balance. Nothing is saved.
func (s *Service) TestKey(ctx context.Context, ws *store.Workspace, in VerificationKeyInput) (VerificationKeyTest, error) {
	d, ok := adapter.Lookup(in.Service)
	if !ok {
		return VerificationKeyTest{}, api.Invalid("body.service", mUnknownService.Text(i18n.LangFrom(ctx), nil))
	}
	a, err := d.New(s.config(in.Service, strings.TrimSpace(in.APIKey), in.Region))
	if err != nil {
		return VerificationKeyTest{}, err
	}
	return s.test(ctx, d, a), nil
}

func (s *Service) test(ctx context.Context, d adapter.Descriptor, a adapter.Adapter) VerificationKeyTest {
	out := VerificationKeyTest{At: s.now()}
	cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	n, err := a.Balance(cctx)
	if err == nil {
		out.OK, out.Balance = true, &n
		return out
	}
	l := i18n.LangFrom(ctx)
	out.Status = adapter.Status(err)
	switch {
	case errors.Is(err, adapter.ErrAuth):
		out.Kind = "key_rejected"
		out.Problem = mKeyRejected.Text(l, i18n.Args{"service": d.Name, "status": statusOr(out.Status, 401)})
	case errors.Is(err, adapter.ErrNoCredits):
		out.Kind = "no_credits"
		out.Problem = mNoCreditsKey.Text(l, i18n.Args{"service": d.Name})
	default:
		out.Kind = "unreachable"
		out.Problem = mUnreachable.Text(l, i18n.Args{"service": d.Name})
	}
	return out
}

func statusOr(s, def int) int {
	if s == 0 {
		return def
	}
	return s
}

// VerificationConnectInput connects a service.
type VerificationConnectInput struct {
	VerificationKeyInput
	Price        float64 `json:"price" minimum:"0" maximum:"10" doc:"Price per check in dollars"`
	DefaultBulk  bool    `json:"defaultBulk"`
	DefaultForms bool    `json:"defaultForms"`
}

// Connect tests the key (it must read the balance) and saves the service
// with its key sealed.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, in VerificationConnectInput) (VerificationServiceView, error) {
	d, ok := adapter.Lookup(in.Service)
	if !ok {
		return VerificationServiceView{}, api.Invalid("body.service", mUnknownService.Text(i18n.LangFrom(ctx), nil))
	}
	if _, err := s.Q.VerifServiceByKey(ctx, store.VerifServiceByKeyParams{WorkspaceID: ws.ID, Service: in.Service}); err == nil {
		return VerificationServiceView{}, api.Conflict(mAlreadyConnected.Text(i18n.LangFrom(ctx), i18n.Args{"service": d.Name}))
	}
	key := strings.TrimSpace(in.APIKey)
	a, err := d.New(s.config(in.Service, key, in.Region))
	if err != nil {
		return VerificationServiceView{}, err
	}
	t := s.test(ctx, d, a)
	if !t.OK {
		return VerificationServiceView{}, api.Invalid("body.apiKey", t.Problem).WithMeta("kind", t.Kind, "status", t.Status)
	}
	sealed, err := SealKey(s.Box, ws.ID, in.Service, key, false)
	if err != nil {
		return VerificationServiceView{}, err
	}
	now := s.now()
	id := uuid.Must(uuid.NewV7())
	by := actor(ctx)
	ok2 := true
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if in.DefaultBulk || in.DefaultForms {
			if err := q.VerifServiceClearDefaults(ctx, store.VerifServiceClearDefaultsParams{WorkspaceID: ws.ID, KeepID: id, Bulk: in.DefaultBulk, Forms: in.DefaultForms}); err != nil {
				return err
			}
		}
		if err := q.VerifServiceInsert(ctx, store.VerifServiceInsertParams{ID: id, WorkspaceID: ws.ID, Service: in.Service, Region: in.Region,
			CredentialsEnc: sealed, PriceMicros: int64(FromDollars(in.Price)), DefaultBulk: in.DefaultBulk, DefaultForms: in.DefaultForms,
			Balance: t.Balance, BalanceAt: &now, TestedAt: &now, TestOk: &ok2, ConnectedBy: by, ConnectedAt: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "verification.service_connect", TargetType: "verification_service",
			TargetID: id.String(), TargetName: d.Name,
			After:    map[string]any{"service": in.Service, "region": in.Region, "price": in.Price, "defaultBulk": in.DefaultBulk, "defaultForms": in.DefaultForms},
			Metadata: map[string]any{"summary": "Connected " + d.Name, "balance": ptr(t.Balance)}})
	})
	if err != nil {
		return VerificationServiceView{}, err
	}
	return s.Get(ctx, ws, in.Service)
}

// Get returns one service's view.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, service string) (VerificationServiceView, error) {
	o, err := s.Overview(ctx, ws)
	if err != nil {
		return VerificationServiceView{}, err
	}
	for _, v := range o.Services {
		if v.Key == service {
			return v, nil
		}
	}
	return VerificationServiceView{}, api.NotFound("Verification service")
}

func (s *Service) connected(ctx context.Context, ws *store.Workspace, service string) (store.VerificationService, error) {
	v, err := s.Q.VerifServiceByKey(ctx, store.VerifServiceByKeyParams{WorkspaceID: ws.ID, Service: service})
	if errors.Is(err, pgx.ErrNoRows) {
		return v, api.Err(http.StatusNotFound, api.CodeNotFound, mNotConnected.Text(i18n.LangFrom(ctx), i18n.Args{"service": ServiceName(service)}))
	}
	return v, err
}

// VerificationSettingsInput changes a service's price, region and default uses.
type VerificationSettingsInput struct {
	Price        float64 `json:"price" minimum:"0" maximum:"10"`
	Region       string  `json:"region,omitempty" enum:"us,eu,"`
	DefaultBulk  bool    `json:"defaultBulk"`
	DefaultForms bool    `json:"defaultForms"`
}

// UpdateSettings saves the price and defaults (audited with before and after).
func (s *Service) UpdateSettings(ctx context.Context, ws *store.Workspace, service string, in VerificationSettingsInput) (VerificationServiceView, error) {
	v, err := s.connected(ctx, ws, service)
	if err != nil {
		return VerificationServiceView{}, err
	}
	before := map[string]any{"price": Money(v.PriceMicros).Dollars(), "region": v.Region, "defaultBulk": v.DefaultBulk, "defaultForms": v.DefaultForms}
	after := map[string]any{"price": in.Price, "region": in.Region, "defaultBulk": in.DefaultBulk, "defaultForms": in.DefaultForms}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.VerifServiceClearDefaults(ctx, store.VerifServiceClearDefaultsParams{WorkspaceID: ws.ID, KeepID: v.ID, Bulk: in.DefaultBulk, Forms: in.DefaultForms}); err != nil {
			return err
		}
		if err := q.VerifServiceSettings(ctx, store.VerifServiceSettingsParams{ID: v.ID, PriceMicros: int64(FromDollars(in.Price)), Region: in.Region,
			DefaultBulk: in.DefaultBulk, DefaultForms: in.DefaultForms, UpdatedBy: actor(ctx), UpdatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "verification.service_update", TargetType: "verification_service",
			TargetID: v.ID.String(), TargetName: ServiceName(service), Before: before, After: after})
	})
	if err != nil {
		return VerificationServiceView{}, err
	}
	return s.Get(ctx, ws, service)
}

// VerificationReplaceKeyInput replaces a service's API key.
type VerificationReplaceKeyInput struct {
	APIKey string `json:"apiKey" minLength:"8" maxLength:"500"`
}

// ReplaceKey tests and seals a new key (audited; the key never is).
func (s *Service) ReplaceKey(ctx context.Context, ws *store.Workspace, service string, in VerificationReplaceKeyInput) (VerificationServiceView, error) {
	v, err := s.connected(ctx, ws, service)
	if err != nil {
		return VerificationServiceView{}, err
	}
	d, _ := adapter.Lookup(service)
	key := strings.TrimSpace(in.APIKey)
	a, err := d.New(s.config(service, key, v.Region))
	if err != nil {
		return VerificationServiceView{}, err
	}
	t := s.test(ctx, d, a)
	if !t.OK {
		return VerificationServiceView{}, api.Invalid("body.apiKey", t.Problem).WithMeta("kind", t.Kind, "status", t.Status)
	}
	sealed, err := SealKey(s.Box, ws.ID, service, key, false)
	if err != nil {
		return VerificationServiceView{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.VerifServiceSetKey(ctx, store.VerifServiceSetKeyParams{ID: v.ID, CredentialsEnc: sealed, UpdatedBy: actor(ctx), UpdatedAt: now}); err != nil {
			return err
		}
		if err := q.VerifServiceSetBalance(ctx, store.VerifServiceSetBalanceParams{ID: v.ID, Balance: t.Balance, BalanceAt: &now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "verification.key_replace", TargetType: "verification_service",
			TargetID: v.ID.String(), TargetName: d.Name})
	})
	if err != nil {
		return VerificationServiceView{}, err
	}
	return s.Get(ctx, ws, service)
}

// TestConnection reads a connected service's balance again ("Read the
// balance again", "Test connection"). It sends no address.
func (s *Service) TestConnection(ctx context.Context, ws *store.Workspace, service string) (VerificationKeyTest, error) {
	v, err := s.connected(ctx, ws, service)
	if err != nil {
		return VerificationKeyTest{}, err
	}
	a, d, err := s.open(v)
	if err != nil {
		return VerificationKeyTest{}, err
	}
	t := s.test(ctx, d, a)
	now := s.now()
	ok := t.OK
	_ = s.Q.VerifServiceTested(ctx, store.VerifServiceTestedParams{ID: v.ID, TestedAt: &now, TestOk: &ok})
	if t.OK {
		_ = s.Q.VerifServiceSetBalance(ctx, store.VerifServiceSetBalanceParams{ID: v.ID, Balance: t.Balance, BalanceAt: &now})
		s.checkLow(ctx, ws, v.ID)
	} else {
		_ = s.Q.VerifServiceFailed(ctx, store.VerifServiceFailedParams{ID: v.ID, Error: t.Problem, At: &now})
	}
	return t, nil
}

// VerificationDisconnectResult says what stopped.
type VerificationDisconnectResult struct {
	Service string `json:"service"`
	// Forms that stopped checking addresses as people sign up.
	Forms []string `json:"forms"`
	// Bulk is true when imports and bulk checks no longer offer it.
	Bulk bool `json:"bulk"`
}

// Disconnect removes a service and its key. Results already on contacts
// and the daily counts of past checks stay.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, service string, forms []string) (VerificationDisconnectResult, error) {
	v, err := s.connected(ctx, ws, service)
	if err != nil {
		return VerificationDisconnectResult{}, err
	}
	out := VerificationDisconnectResult{Service: service, Forms: forms, Bulk: v.DefaultBulk}
	if out.Forms == nil {
		out.Forms = []string{}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := tx.Exec(ctx, `UPDATE verification_jobs SET status = 'cancelled', stop_reason = 'cancelled', finished_at = now()
			WHERE service_id = $1 AND status IN ('queued', 'running', 'paused')`, v.ID); err != nil {
			return err
		}
		if _, err := q.VerifServiceDelete(ctx, store.VerifServiceDeleteParams{ID: v.ID, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "verification.service_disconnect", TargetType: "verification_service",
			TargetID: v.ID.String(), TargetName: ServiceName(service),
			Before:   map[string]any{"defaultBulk": v.DefaultBulk, "defaultForms": v.DefaultForms, "price": Money(v.PriceMicros).Dollars()},
			Metadata: map[string]any{"forms": out.Forms}})
	})
	return out, err
}

// RefreshBalances reads every connected service's balance (hourly).
func (s *Service) RefreshBalances(ctx context.Context) error {
	rows, err := s.Q.VerifServiceAll(ctx)
	if err != nil {
		return err
	}
	for _, v := range rows {
		a, _, err := s.open(v)
		if err != nil {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		n, err := a.Balance(cctx)
		cancel()
		now := s.now()
		if err != nil {
			_ = s.Q.VerifServiceFailed(ctx, store.VerifServiceFailedParams{ID: v.ID, Error: err.Error(), At: &now})
			continue
		}
		_ = s.Q.VerifServiceSetBalance(ctx, store.VerifServiceSetBalanceParams{ID: v.ID, Balance: &n, BalanceAt: &now})
		if ws, err := s.Q.GetWorkspace(ctx, v.WorkspaceID); err == nil {
			s.checkLow(ctx, &ws, v.ID)
		}
	}
	return nil
}

// checkLow tells members with integrations:manage once when a balance runs
// under 1,000 credits or under what a waiting job needs, and forgets it
// once the balance recovers.
func (s *Service) checkLow(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
	v, err := s.Q.VerifServiceByID(ctx, id)
	if err != nil || v.Balance == nil {
		return
	}
	var needed int64
	label := ""
	jobs, _ := s.Q.VerifJobList(ctx, store.VerifJobListParams{WorkspaceID: ws.ID, Since: s.now().Add(-30 * 24 * time.Hour), Lim: 50})
	for _, jr := range jobs {
		j := jr.VerificationJob
		if j.Service == v.Service && j.Status == "stopped" && j.StopReason == "no_credits" {
			needed, label = int64(j.Total), j.Label
			break
		}
	}
	low := *v.Balance < LowCredits || needed > 0 && *v.Balance < needed
	switch {
	case low && v.LowNotifiedAt == nil:
		now := s.now()
		_ = s.Q.VerifServiceLowNotified(ctx, store.VerifServiceLowNotifiedParams{ID: v.ID, At: &now})
		if s.Notify != nil {
			s.Notify.CreditsLow(ctx, ws, v, needed, label)
		}
	case !low && v.LowNotifiedAt != nil:
		_ = s.Q.VerifServiceLowNotified(ctx, store.VerifServiceLowNotifiedParams{ID: v.ID})
	}
}
