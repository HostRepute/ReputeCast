// Package connect implements "Connect a sending provider" (spec 5.7, 7.2):
// the provider catalog built from the adapter registry, and a six-step
// setup (provider, connection, domain verification, test send, streams,
// contacts) whose progress is kept in provider_setups so a reload resumes
// the same step. The connection step works for every adapter through the
// Provider contract and its optional interfaces: credential checks
// (Verifier), webhook registration (WebhookRegistrar) behind a reachability
// test of the public URL, domain records (DomainManager), provisioning with
// per-step progress and retry (Provisioner, SES), region scans and
// production-access requests (SES), connection and bounce-mailbox tests
// (SMTP). Core code never imports an adapter.
//
// It also serves Settings › Providers: a summary per provider and the
// workspace-wide sending defaults (auto-pause thresholds, throttles per
// mailbox provider, suppression sync interval, webhook secret rotation).
package connect

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/suppressions"
	"github.com/hostrepute/reputecast/internal/systemmail"
)

// Service is the connect flow and Settings › Providers.
type Service struct {
	Pool         *pgxpool.Pool
	Q            *store.Queries
	Audit        *audit.Logger
	Box          *secrets.Box
	Factory      *manage.Factory
	Domains      *domains.Service
	Routing      *routing.Service
	Suppressions *suppressions.Service
	Providers    *manage.Service
	Deliver      *systemmail.Deliverer
	Links        *linktoken.Signer
	Jobs         func() *jobs.Client
	// URLs is the instance's public URL (App.URLs): provider webhooks and
	// the test message's open pixel point at it.
	URLs publicurl.Builder
	// TemplateURL is the published CloudFormation template (SES).
	TemplateURL string
	// Reachable tests a webhook URL from outside; nil uses the HTTP
	// self-test (tests replace it).
	Reachable func(ctx context.Context, url string, p store.Provider) SelfTestState
	// AllowPrivateURL lets the webhook self-test reach a private or
	// loopback public URL (integration tests only).
	AllowPrivateURL bool
	// OnRoutingChange is called after the streams step saved routing (the
	// system mailer forgets its cached availability).
	OnRoutingChange func()
	Log             *slog.Logger
	Now             func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func (s *Service) jobs() *jobs.Client {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// randomToken returns a URL-safe random token of n bytes.
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func principalUser(ctx context.Context) *uuid.UUID {
	if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
		u := p.UserID
		return &u
	}
	return nil
}

func canManage(ctx context.Context) bool {
	p := api.PrincipalFrom(ctx)
	return p != nil && p.Can(authz.ProvidersManage)
}

// helpers names who can manage providers when the caller can't ("Ask Linh
// Nguyen or Marcus Chen"); empty for those who can.
func (s *Service) helpers(ctx context.Context, wsID uuid.UUID) []string {
	if canManage(ctx) {
		return []string{}
	}
	if h := access.Helpers(ctx, s.Q, wsID, authz.ProvidersManage); h != nil {
		return h
	}
	return []string{}
}

// descriptor returns a registered adapter or a 404 problem.
func descriptor(adapter string) (providers.Descriptor, error) {
	d, ok := providers.Lookup(adapter)
	if !ok || d.New == nil {
		return providers.Descriptor{}, api.Err(http.StatusNotFound, api.CodeNotFound, "This build has no "+adapter+" adapter. Pick a provider from the catalog.")
	}
	return d, nil
}

// setup loads a setup of the workspace.
func (s *Service) setup(ctx context.Context, ws *store.Workspace, rawID string) (store.ProviderSetup, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return store.ProviderSetup{}, api.NotFound("Setup")
	}
	st, err := s.Q.SetupGet(ctx, store.SetupGetParams{WorkspaceID: ws.ID, ID: id})
	return st, api.MapError(err, "Setup")
}

// save writes a setup's step, status and state.
func (s *Service) save(ctx context.Context, q *store.Queries, st store.ProviderSetup, state State) (store.ProviderSetup, error) {
	return q.SetupSave(ctx, store.SetupSaveParams{ID: st.ID, Step: st.Step, Status: st.Status, State: state.encode(), ProviderID: st.ProviderID,
		UpdatedAt: s.now(), FinishedAt: st.FinishedAt})
}

// update loads a setup for update, lets fn change it, and saves it with an
// audit entry when action is set.
func (s *Service) update(ctx context.Context, ws *store.Workspace, id uuid.UUID, action string, meta map[string]any, fn func(st *store.ProviderSetup, state *State) error) (store.ProviderSetup, error) {
	var out store.ProviderSetup
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		st, err := q.SetupLock(ctx, id)
		if err != nil {
			return api.MapError(err, "Setup")
		}
		if st.WorkspaceID != ws.ID {
			return api.NotFound("Setup")
		}
		state := decodeState(st.State)
		if err := fn(&st, &state); err != nil {
			return err
		}
		if out, err = s.save(ctx, q, st, state); err != nil {
			return err
		}
		if action == "" {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "provider_setup", TargetID: st.ID.String(), TargetName: st.Adapter, Metadata: meta})
	})
	return out, err
}

// ConnectSetupInput starts a setup.
type ConnectSetupInput struct {
	Adapter string `json:"adapter" minLength:"2" maxLength:"120"`
}

// Create starts a setup for an adapter, or returns the open one to resume.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, in ConnectSetupInput) (ConnectSetup, error) {
	d, err := descriptor(in.Adapter)
	if err != nil {
		return ConnectSetup{}, err
	}
	adapter := d.Adapter
	if open, err := s.Q.SetupOpen(ctx, store.SetupOpenParams{WorkspaceID: ws.ID, Adapter: &adapter}); err == nil && len(open) > 0 {
		return s.View(ctx, ws, open[0])
	}
	state := State{}
	if flowOf(d) == "ses" {
		state.Connection.Access, state.Connection.ExternalID, state.Connection.Region = "role", "rc-"+randomToken(18), "us-east-1"
	}
	id := newID()
	var st store.ProviderSetup
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if st, err = q.SetupCreate(ctx, store.SetupCreateParams{ID: id, WorkspaceID: ws.ID, Adapter: d.Adapter, State: state.encode(), CreatedBy: principalUser(ctx), CreatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "provider.setup_start", TargetType: "provider_setup", TargetID: id.String(), TargetName: d.Name,
			After: map[string]any{"adapter": d.Adapter}})
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, st)
}

// ConnectSetupSummary is one setup of the list.
type ConnectSetupSummary struct {
	ID      string `json:"id"`
	Adapter string `json:"adapter"`
	// Name is the adapter's display name ("Amazon SES").
	Name       string     `json:"name"`
	Step       string     `json:"step"`
	Status     string     `json:"status" enum:"open,finished,abandoned"`
	Provider   string     `json:"provider"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// ConnectSetupList is a workspace's setups, newest first.
type ConnectSetupList struct {
	Items []ConnectSetupSummary `json:"items"`
}

// List returns the workspace's latest setups, optionally of one adapter or
// status.
func (s *Service) List(ctx context.Context, ws *store.Workspace, adapter, status string) (ConnectSetupList, error) {
	out := ConnectSetupList{Items: []ConnectSetupSummary{}}
	p := store.SetupListParams{WorkspaceID: ws.ID}
	if adapter != "" {
		p.Adapter = &adapter
	}
	if status != "" {
		p.Status = &status
	}
	rows, err := s.Q.SetupList(ctx, p)
	if err != nil {
		return out, err
	}
	for _, st := range rows {
		it := ConnectSetupSummary{ID: st.ID.String(), Adapter: st.Adapter, Name: st.Adapter, Step: st.Step, Status: st.Status, CreatedAt: st.CreatedAt,
			UpdatedAt: st.UpdatedAt, FinishedAt: st.FinishedAt}
		if d, ok := providers.Lookup(st.Adapter); ok {
			it.Name = copyFor(d).Display
		}
		if st.ProviderID != nil {
			if p, err := s.Q.ProvGetAny(ctx, *st.ProviderID); err == nil {
				it.Provider = p.Name
			}
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// Get returns a setup.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, rawID string) (ConnectSetup, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, st)
}

// ConnectStepInput moves a setup to a step (Back, Continue, Save and
// finish later).
type ConnectStepInput struct {
	Step string `json:"step" enum:"connection,domain,test,streams,contacts"`
}

// SetStep records the step the person is on, so a reload resumes there.
func (s *Service) SetStep(ctx context.Context, ws *store.Workspace, rawID string, in ConnectStepInput) (ConnectSetup, error) {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	st, err := s.update(ctx, ws, cur.ID, "", nil, func(st *store.ProviderSetup, state *State) error {
		if st.Status == "abandoned" {
			return api.Conflict("This setup was cancelled. Start again from the catalog.")
		}
		if in.Step != "connection" && st.ProviderID == nil {
			return api.Conflict("Connect the provider first: the next steps need it.")
		}
		if st.Status == "open" {
			st.Step = in.Step
		}
		return nil
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, st)
}

// Abandon cancels an open setup. A provider it connected that no route
// uses yet is disconnected again.
func (s *Service) Abandon(ctx context.Context, ws *store.Workspace, rawID string) error {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return err
	}
	if cur.Status != "open" {
		return api.Conflict("Only an unfinished setup can be cancelled.")
	}
	disconnect := false
	if cur.ProviderID != nil {
		routed, err := s.Q.ConnectProviderRouted(ctx, store.ConnectProviderRoutedParams{WorkspaceID: ws.ID, ProviderID: *cur.ProviderID})
		if err != nil {
			return err
		}
		disconnect = !routed
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		st, err := q.SetupLock(ctx, cur.ID)
		if err != nil {
			return err
		}
		st.Status = "abandoned"
		if _, err := s.save(ctx, q, st, decodeState(st.State)); err != nil {
			return err
		}
		if disconnect {
			if err := q.ProvDisconnect(ctx, *cur.ProviderID); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "provider.setup_cancel", TargetType: "provider_setup", TargetID: cur.ID.String(), TargetName: cur.Adapter,
			Metadata: map[string]any{"provider_disconnected": disconnect}})
	})
}

// provider loads the setup's provider.
func (s *Service) provider(ctx context.Context, ws *store.Workspace, st store.ProviderSetup) (store.Provider, error) {
	if st.ProviderID == nil {
		return store.Provider{}, api.Conflict("Connect the provider first.")
	}
	p, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: *st.ProviderID})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Provider{}, api.Conflict("The provider of this setup was disconnected. Start again from the catalog.")
	}
	return p, err
}

// providerName names a new connection: the adapter's name, with its label
// when the workspace already has one ("Amazon SES (eu-west-1)").
func (s *Service) providerName(ctx context.Context, wsID uuid.UUID, d providers.Descriptor, label string) string {
	names, _ := s.Q.ConnectProviderNames(ctx, wsID)
	name := d.Name
	if !slices.Contains(names, name) {
		return name
	}
	if label != "" && !slices.Contains(names, name+" ("+label+")") {
		return name + " (" + label + ")"
	}
	for i := 2; ; i++ {
		n := name + " (" + itoa(i) + ")"
		if !slices.Contains(names, n) {
			return n
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// clean trims every value of a form map.
func clean(m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}
