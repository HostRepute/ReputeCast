package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service manages the runtime of installed community adapters: where each
// process listens, its signing secret and its health. The Adapters tab's
// install, update and remove (internal/providers/manage) cover the rest; the
// server CLI (`reputecast adapter`) uses this service for everything.
type Service struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Audit   *audit.Logger
	Box     *secrets.Box
	Runtime *Runtime
	// AllowPrivate reports adapters.allow_private_networks.
	AllowPrivate func() bool
	Now          func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) allowPrivate() bool { return s.AllowPrivate != nil && s.AllowPrivate() }

// CLIActor is the audit actor of the server CLI.
var CLIActor = &audit.Actor{Type: authz.ActorSystem, Name: "CLI (server)"}

// AdapterRuntime is the runtime of an installed community adapter, for
// instance admins. The signing secret is never returned.
type AdapterRuntime struct {
	AdapterID string `json:"adapterId" format:"uuid"`
	// Adapter is the registry key: "community:<id>".
	Adapter string `json:"adapter" example:"community:example-esp"`
	Name    string `json:"name"`
	Version string `json:"version" doc:"Version of the installed manifest"`
	// EndpointURL is where the adapter process listens ("" until it is set).
	EndpointURL string `json:"endpointUrl"`
	// EndpointProblem explains why the stored address is refused.
	EndpointProblem          string     `json:"endpointProblem,omitempty"`
	SecretIssuedAt           time.Time  `json:"secretIssuedAt"`
	PreviousSecretValidUntil *time.Time `json:"previousSecretValidUntil,omitempty"`
	// Operations are the protocol operations ReputeCast may call, from the
	// declared capabilities.
	Operations   []string `json:"operations"`
	Capabilities []string `json:"capabilities"`
	// Network lists the hosts the manifest declares.
	Network []string `json:"network"`
	// Breaker is this process's circuit breaker for the adapter.
	Breaker          string     `json:"breaker" enum:"closed,open,half_open"`
	BreakerOpenUntil *time.Time `json:"breakerOpenUntil,omitempty"`
	LastFailure      string     `json:"lastFailure,omitempty"`
	// Registered is true when this process has the adapter in its registry.
	Registered bool                 `json:"registered"`
	Check      *AdapterRuntimeCheck `json:"check,omitempty"`
	Connected  int64                `json:"connected" doc:"Connected providers using the adapter, in every workspace"`
}

// AdapterRuntimeCheck is the last health check of the adapter process.
type AdapterRuntimeCheck struct {
	At             time.Time `json:"at"`
	OK             bool      `json:"ok"`
	Detail         string    `json:"detail"`
	RunningVersion string    `json:"runningVersion,omitempty"`
}

// AdapterRuntimeInput sets where the adapter process listens.
type AdapterRuntimeInput struct {
	EndpointURL string `json:"endpointUrl" minLength:"8" maxLength:"2048" doc:"Base URL of the adapter process, like http://example-esp:4600"`
}

// AdapterRuntimeSecret carries a new signing secret, shown once.
type AdapterRuntimeSecret struct {
	Secret string `json:"secret" doc:"The new signing secret. Shown only in this response; set it on the adapter process as REPUTECAST_ADAPTER_SECRET."`
	// PreviousSecretValidUntil is when the previous secret stops working.
	PreviousSecretValidUntil *time.Time `json:"previousSecretValidUntil,omitempty"`
}

func (s *Service) row(ctx context.Context, id uuid.UUID) (store.RemoteAdapterGetRow, error) {
	r, err := s.Q.RemoteAdapterGet(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, api.NotFound("Community adapter")
	}
	return r, err
}

// RowByKey finds an installed remote adapter by its id or registry key.
func (s *Service) RowByKey(ctx context.Context, idOrKey string) (store.RemoteAdapterGetRow, error) {
	key := idOrKey
	if len(key) < len(providers.RuntimePrefix) || key[:len(providers.RuntimePrefix)] != providers.RuntimePrefix {
		key = providers.RuntimePrefix + key
	}
	r, err := s.Q.RemoteAdapterGetByKey(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.RemoteAdapterGetRow{}, fmt.Errorf("no community adapter %q is installed", idOrKey)
	}
	return store.RemoteAdapterGetRow(r), err
}

// View describes an installed adapter's runtime.
func (s *Service) View(ctx context.Context, r store.RemoteAdapterGetRow) (AdapterRuntime, error) {
	v := AdapterRuntime{AdapterID: r.ID.String(), Adapter: r.AdapterKey, Name: r.Name, Version: r.Version, EndpointURL: r.EndpointUrl,
		SecretIssuedAt: r.SecretIssuedAt, Operations: []string{}, Capabilities: []string{}, Network: []string{}, Breaker: BreakerClosed}
	if r.PreviousValidUntil != nil && s.now().Before(*r.PreviousValidUntil) {
		v.PreviousSecretValidUntil = r.PreviousValidUntil
	}
	if m, err := ParseManifest(r.Manifest); err == nil {
		v.Operations, v.Capabilities = Operations(m.Capabilities), m.Labels()
		if len(m.Permissions.Network) > 0 {
			v.Network = m.Permissions.Network
		}
	}
	if r.EndpointUrl != "" {
		if _, err := CheckEndpoint(r.EndpointUrl, s.allowPrivate()); err != nil {
			v.EndpointProblem = err.Error()
		}
	}
	if s.Runtime != nil {
		if in, ok := s.Runtime.Installation(r.AdapterKey); ok {
			v.Registered = true
			state, until, last := in.BreakerState()
			v.Breaker, v.LastFailure = state, last
			if !until.IsZero() {
				v.BreakerOpenUntil = &until
			}
		}
	}
	if r.CheckedAt != nil && r.CheckOk != nil {
		v.Check = &AdapterRuntimeCheck{At: *r.CheckedAt, OK: *r.CheckOk, Detail: r.CheckDetail, RunningVersion: r.RunningVersion}
	}
	n, err := s.Q.RemoteAdapterConnected(ctx, r.AdapterKey)
	if err != nil {
		return v, err
	}
	v.Connected = n
	return v, nil
}

// List describes every installed remote adapter.
func (s *Service) List(ctx context.Context) ([]AdapterRuntime, error) {
	rows, err := s.Q.RemoteAdapterList(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdapterRuntime, 0, len(rows))
	for _, r := range rows {
		v, err := s.View(ctx, store.RemoteAdapterGetRow(r))
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Service) sync(ctx context.Context) {
	if s.Runtime != nil {
		s.Runtime.Changed(ctx)
	}
}

// SetEndpoint sets where the adapter process listens. actor is nil for
// people (taken from the request) or CLIActor; wsID is the workspace the
// change was made from, nil from the CLI.
func (s *Service) SetEndpoint(ctx context.Context, wsID *uuid.UUID, r store.RemoteAdapterGetRow, endpoint string, actor *audit.Actor) (store.RemoteAdapterGetRow, error) {
	u, err := CheckEndpoint(endpoint, s.allowPrivate())
	if err != nil {
		return r, api.Invalid("body.endpointUrl", "This address can't be used: "+err.Error()+".")
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RemoteRuntimeSetEndpoint(ctx, store.RemoteRuntimeSetEndpointParams{AdapterID: r.ID, EndpointUrl: u}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: wsID, Actor: actor, Action: "adapter.runtime_update", TargetType: "adapter", TargetID: r.ID.String(), TargetName: r.Name,
			Before: map[string]any{"endpoint": r.EndpointUrl}, After: map[string]any{"endpoint": u}})
	})
	if err != nil {
		return r, err
	}
	s.sync(ctx)
	return s.row(ctx, r.ID)
}

// IssueSecret replaces the signing secret and returns the new one, once.
// The previous secret keeps working for PreviousSecretGrace.
func (s *Service) IssueSecret(ctx context.Context, wsID *uuid.UUID, r store.RemoteAdapterGetRow, actor *audit.Actor) (AdapterRuntimeSecret, error) {
	secret := NewSecret()
	sealed, err := s.Box.SealString(secret, SecretAD(r.ID))
	if err != nil {
		return AdapterRuntimeSecret{}, err
	}
	now := s.now()
	until := now.Add(PreviousSecretGrace)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RemoteRuntimeSetSecret(ctx, store.RemoteRuntimeSetSecretParams{AdapterID: r.ID, SecretEnc: sealed, IssuedAt: now, PreviousValidUntil: &until}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: wsID, Actor: actor, Action: "adapter.secret_issue", TargetType: "adapter", TargetID: r.ID.String(), TargetName: r.Name,
			Metadata: map[string]any{"previous_valid_until": until.UTC().Format(time.RFC3339)}})
	})
	if err != nil {
		return AdapterRuntimeSecret{}, err
	}
	s.sync(ctx)
	return AdapterRuntimeSecret{Secret: secret, PreviousSecretValidUntil: &until}, nil
}

// Check calls the adapter's health operation now and records the result
// (audited like a provider check).
func (s *Service) Check(ctx context.Context, wsID *uuid.UUID, r store.RemoteAdapterGetRow, actor *audit.Actor) (store.RemoteAdapterGetRow, error) {
	var in *Installation
	if s.Runtime != nil {
		s.Runtime.Changed(ctx)
		in, _ = s.Runtime.Installation(r.AdapterKey)
	}
	var (
		h   HealthOutput
		err error
	)
	if in == nil {
		// Not registered: the stored manifest or secret can't be used.
		rt := s.Runtime
		if rt == nil {
			rt = &Runtime{Pool: s.Pool, Box: s.Box, AllowPrivate: s.AllowPrivate, Now: s.Now}
		}
		in, err = rt.build(r, s.now(), s.allowPrivate(), rt.publicURL())
	}
	if in != nil {
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		h, err = in.Health(cctx)
		cancel()
	}
	ok, detail := err == nil, ""
	if err != nil {
		detail = err.Error()
		var ce *providers.CallError
		if errors.As(err, &ce) {
			detail = ce.Message
		}
	} else {
		detail = "Answered as " + h.Adapter + " " + h.Version + ", protocol " + fmt.Sprint(h.Protocol)
		if h.Version != r.Version {
			detail += "; the installed manifest is version " + r.Version
		}
		detail += "."
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RemoteRuntimeRecordCheck(ctx, store.RemoteRuntimeRecordCheckParams{AdapterID: r.ID, CheckedAt: &now, CheckOk: &ok, CheckDetail: cut(detail), RunningVersion: cut(h.Version)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: wsID, Actor: actor, Action: "adapter.check", TargetType: "adapter", TargetID: r.ID.String(), TargetName: r.Name,
			Metadata: map[string]any{"ok": ok, "detail": cut(detail)}})
	})
	if err != nil {
		return r, err
	}
	return s.row(ctx, r.ID)
}

// InstallResult is what the CLI prints after an install.
type InstallResult struct {
	Row     store.RemoteAdapterGetRow
	Updated bool
	// Secret is the new signing secret on a first install ("" on update).
	Secret string
}

// Install installs a remote adapter from its manifest, or updates the
// manifest when its id is already installed (server CLI). source is where
// the manifest came from ("" for a local file).
func (s *Service) Install(ctx context.Context, raw []byte, source, endpoint string, actor *audit.Actor) (InstallResult, error) {
	m, err := ParseManifest(raw)
	if err != nil {
		return InstallResult{}, err
	}
	canonical, _ := json.Marshal(json.RawMessage(raw))
	if endpoint != "" {
		if endpoint, err = CheckEndpoint(endpoint, s.allowPrivate()); err != nil {
			return InstallResult{}, err
		}
	}
	var res InstallResult
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.RemoteAdapterGetByKey(ctx, m.Key())
		switch {
		case err == nil:
			res.Updated = true
			if err := q.RemoteAdapterUpdateManifest(ctx, store.RemoteAdapterUpdateManifestParams{ID: cur.ID, Name: m.Name, Version: m.Version, Maintainer: m.Maintainer, SourceUrl: source, Manifest: canonical}); err != nil {
				return err
			}
			if err := q.RemoteRuntimeTouch(ctx, cur.ID); err != nil {
				return err
			}
			if endpoint != "" && endpoint != cur.EndpointUrl {
				if err := q.RemoteRuntimeSetEndpoint(ctx, store.RemoteRuntimeSetEndpointParams{AdapterID: cur.ID, EndpointUrl: endpoint}); err != nil {
					return err
				}
			}
			res.Row.ID = cur.ID
			return s.Audit.LogWith(ctx, q, audit.Event{Actor: actor, Action: "adapter.update", TargetType: "adapter", TargetID: cur.ID.String(), TargetName: m.Name,
				Before: map[string]any{"version": cur.Version}, After: map[string]any{"version": m.Version, "source": source}})
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		id := uuid.Must(uuid.NewV7())
		var latest *string
		if m.Latest != "" {
			latest = &m.Latest
		}
		if err := q.RemoteAdapterInsert(ctx, store.RemoteAdapterInsertParams{ID: id, Name: m.Name, Version: m.Version, Maintainer: m.Maintainer, SourceUrl: source, Manifest: canonical, LatestVersion: latest}); err != nil {
			var pe interface{ SQLState() string }
			if errors.As(err, &pe) && pe.SQLState() == "23505" {
				return fmt.Errorf("an adapter named %q is already installed", m.Name)
			}
			return err
		}
		secret, err := createRuntime(ctx, q, s.Box, id, m, endpoint)
		if err != nil {
			return err
		}
		res.Row.ID, res.Secret = id, secret
		return s.Audit.LogWith(ctx, q, audit.Event{Actor: actor, Action: "adapter.install", TargetType: "adapter", TargetID: id.String(), TargetName: m.Name,
			After: map[string]any{"version": m.Version, "maintainer": m.Maintainer, "source": source, "kind": Kind, "endpoint": endpoint}})
	})
	if err != nil {
		return InstallResult{}, err
	}
	s.sync(ctx)
	res.Row, err = s.row(ctx, res.Row.ID)
	return res, err
}

// Remove uninstalls a remote adapter no connected provider uses (server CLI;
// the Adapters tab uses manage.RemoveAdapter).
func (s *Service) Remove(ctx context.Context, r store.RemoteAdapterGetRow, actor *audit.Actor) error {
	n, err := s.Q.RemoteAdapterConnected(ctx, r.AdapterKey)
	if err != nil {
		return err
	}
	if n > 0 {
		return api.Conflict(fmt.Sprintf("%d connected provider(s) use %s. Disconnect them first.", n, r.Name))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.RemoteAdapterDelete(ctx, r.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Actor: actor, Action: "adapter.remove", TargetType: "adapter", TargetID: r.ID.String(), TargetName: r.Name})
	})
	if err != nil {
		return err
	}
	s.sync(ctx)
	return nil
}

// instanceAdmin refuses people who aren't instance admins: adapters apply
// to every workspace on the instance.
func instanceAdmin(ctx context.Context) error {
	if p := api.PrincipalFrom(ctx); p == nil || !p.IsInstanceAdmin {
		return api.Err(http.StatusForbidden, api.CodePermissionDenied, "Community adapters run for every workspace on this instance, so only instance admins can see or change where they run.")
	}
	return nil
}
