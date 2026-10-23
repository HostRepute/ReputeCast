package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// SyncInterval is how often every process re-reads the installed adapters,
// so an install, update or removal made by another process (or the CLI)
// reaches it.
const SyncInterval = 15 * time.Second

// PreviousSecretGrace is how long the previous signing secret keeps working
// after a new one is issued.
const PreviousSecretGrace = 24 * time.Hour

// SecretAD is the associated data sealing an installed adapter's signing
// secret (the current and the previous one).
func SecretAD(adapterID uuid.UUID) string { return "adapter:" + adapterID.String() + ":secret" }

// KindDescriptor builds the registry entry of an installed adapter of a
// kind other than "remote" (the app wires the generic HTTP adapter's
// httpgeneric.CommunityDescriptor for "http"). key is
// providers.CommunityKey(name).
type KindDescriptor func(key, name, version, maintainer string, manifest []byte) (providers.Descriptor, error)

// Runtime keeps the provider registry in step with the installed community
// adapters: kind "remote" as "community:<id>", proxied to the adapter
// process, and the kinds in Kinds as "community:<name>". It implements
// manage.CommunityRuntime.
type Runtime struct {
	Pool *pgxpool.Pool
	Box  *secrets.Box
	Log  *slog.Logger
	// AllowPrivate reports providers.allow_private_networks; read at each sync.
	AllowPrivate func() bool
	// PublicURL is ReputeCast's public base URL (forwarded webhook URLs).
	PublicURL func() string
	Now       func() time.Time
	// HTTP overrides every installation's client (tests).
	HTTP *http.Client
	// Interval overrides SyncInterval.
	Interval time.Duration
	// Kinds registers installed adapters of other kinds ("http").
	Kinds map[string]KindDescriptor

	syncMu   sync.Mutex
	mu       sync.Mutex
	installs map[string]*Installation
	breakers map[string]*breaker
	finger   string
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func (rt *Runtime) now() time.Time {
	if rt.Now != nil {
		return rt.Now()
	}
	return time.Now()
}

func (rt *Runtime) log() *slog.Logger {
	if rt.Log != nil {
		return rt.Log
	}
	return slog.Default()
}

func (rt *Runtime) allowPrivate() bool { return rt.AllowPrivate != nil && rt.AllowPrivate() }

func (rt *Runtime) publicURL() string {
	if rt.PublicURL != nil {
		return rt.PublicURL()
	}
	return ""
}

// Installation returns a registered installation by key ("community:<id>").
func (rt *Runtime) Installation(key string) (*Installation, bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	in, ok := rt.installs[key]
	return in, ok
}

// Start registers the installed adapters and keeps them in step every
// SyncInterval until Stop or until the database pool is closed. A first sync
// that fails (for example before migrations ran) is retried by the loop.
func (rt *Runtime) Start(ctx context.Context) {
	if err := rt.Sync(ctx); err != nil {
		rt.syncFailed(err)
	}
	rt.mu.Lock()
	if rt.stop != nil {
		rt.mu.Unlock()
		return
	}
	rt.stop, rt.done = make(chan struct{}), make(chan struct{})
	stop, done := rt.stop, rt.done
	rt.mu.Unlock()
	interval := rt.Interval
	if interval <= 0 {
		interval = SyncInterval
	}
	// The loop outlives the start-up context: it ends with Stop or the pool.
	base := context.WithoutCancel(ctx)
	go func() {
		defer close(done)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
			}
			cctx, cancel := context.WithTimeout(base, 10*time.Second)
			err := rt.Sync(cctx)
			cancel()
			if err != nil {
				if strings.Contains(err.Error(), "closed pool") {
					return
				}
				rt.syncFailed(err)
			}
		}
	}()
}

// Stop ends the sync loop and waits for a sync in progress to finish. A
// runtime stopped before Start never starts its loop; Sync and Changed keep
// working.
func (rt *Runtime) Stop() {
	rt.stopOnce.Do(func() {
		rt.mu.Lock()
		stop, done := rt.stop, rt.done
		if stop == nil {
			rt.stop = make(chan struct{})
		}
		rt.mu.Unlock()
		if stop != nil {
			close(stop)
			<-done
		}
	})
}

func (rt *Runtime) syncFailed(err error) {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "42P01" {
		rt.log().Info("community adapters not loaded yet: the database needs its migrations")
		return
	}
	rt.log().Warn("community adapters: sync failed", "err", err)
}

// Sync reads the installed community adapters and registers them: remote
// adapters, and the kinds in Kinds. A row that can't be used (a manifest
// that no longer validates, a secret that can't be opened, a key another
// adapter already has) is skipped and reported; the others are registered.
func (rt *Runtime) Sync(ctx context.Context) error {
	rt.syncMu.Lock()
	defer rt.syncMu.Unlock()
	q := store.New(rt.Pool)
	rows, err := q.RemoteAdapterList(ctx)
	if err != nil {
		return err
	}
	others := []store.CommunityAdapterListRow{}
	if len(rt.Kinds) > 0 {
		all, err := q.CommunityAdapterList(ctx)
		if err != nil {
			return err
		}
		for _, a := range all {
			if rt.Kinds[a.Kind] != nil {
				others = append(others, a)
			}
		}
	}
	now := rt.now()
	allow, public := rt.allowPrivate(), rt.publicURL()
	h := sha256.New()
	fmt.Fprintf(h, "%v|%s|", allow, public)
	for _, r := range rows {
		prevValid := r.PreviousValidUntil != nil && now.Before(*r.PreviousValidUntil)
		fmt.Fprintf(h, "%s|%s|%s|%x|%v|%x|", r.ID, r.AdapterKey, r.EndpointUrl, r.SecretEnc, prevValid, r.PreviousSecretEnc)
		h.Write(r.Manifest)
	}
	for _, a := range others {
		fmt.Fprintf(h, "%s|%s|%s|%s|%s|", a.ID, a.Kind, a.Name, a.Version, a.Maintainer)
		h.Write(a.Manifest)
	}
	finger := hex.EncodeToString(h.Sum(nil))
	rt.mu.Lock()
	unchanged := finger == rt.finger
	rt.mu.Unlock()
	if unchanged {
		return nil
	}
	installs := map[string]*Installation{}
	descs := []providers.Descriptor{}
	var errs []error
	for _, r := range rows {
		in, err := rt.build(store.RemoteAdapterGetRow(r), now, allow, public)
		if err != nil {
			errs = append(errs, err)
			rt.log().Warn("community adapter skipped", "adapter", r.AdapterKey, "err", err)
			continue
		}
		installs[in.Manifest.Key()] = in
		descs = append(descs, in.Manifest.Descriptor(in.Open))
	}
	for _, a := range others {
		key := providers.InstalledKey(a.Kind, a.Name, a.Manifest)
		if slices.ContainsFunc(descs, func(d providers.Descriptor) bool { return d.Adapter == key }) {
			err := fmt.Errorf("%s: another installed adapter is registered as %s", a.Name, key)
			errs = append(errs, err)
			rt.log().Warn("community adapter skipped", "adapter", key, "err", err)
			continue
		}
		d, err := rt.Kinds[a.Kind](key, a.Name, a.Version, a.Maintainer, a.Manifest)
		if err != nil {
			err = fmt.Errorf("%s: the stored %s manifest can't be used: %w", a.Name, a.Kind, err)
			errs = append(errs, err)
			rt.log().Warn("community adapter skipped", "adapter", key, "err", err)
			continue
		}
		descs = append(descs, d)
	}
	if err := providers.SetRuntime(rt, descs); err != nil {
		return err
	}
	rt.mu.Lock()
	rt.installs, rt.finger = installs, finger
	rt.mu.Unlock()
	rt.log().Debug("community adapters registered", "adapters", slices.Sorted(maps.Keys(installs)))
	return errors.Join(errs...)
}

// build turns a stored row into an installation.
func (rt *Runtime) build(r store.RemoteAdapterGetRow, now time.Time, allow bool, public string) (*Installation, error) {
	m, err := ParseManifest(r.Manifest)
	if err != nil {
		return nil, fmt.Errorf("%s: the stored manifest no longer validates: %w", r.Name, err)
	}
	if m.Key() != r.AdapterKey {
		return nil, fmt.Errorf("%s: the manifest id %q doesn't match the registered %q", r.Name, m.ID, r.AdapterKey)
	}
	if rt.Box == nil {
		return nil, fmt.Errorf("%s: no secrets box", r.Name)
	}
	cur, err := rt.Box.OpenString(r.SecretEnc, SecretAD(r.ID))
	if err != nil {
		return nil, fmt.Errorf("%s: the signing secret can't be opened (was the instance key changed?): %w", r.Name, err)
	}
	keys := []string{cur}
	if len(r.PreviousSecretEnc) > 0 && r.PreviousValidUntil != nil && now.Before(*r.PreviousValidUntil) {
		if prev, err := rt.Box.OpenString(r.PreviousSecretEnc, SecretAD(r.ID)); err == nil {
			keys = append(keys, prev)
		}
	}
	rt.mu.Lock()
	if rt.breakers == nil {
		rt.breakers = map[string]*breaker{}
	}
	br, ok := rt.breakers[r.AdapterKey]
	if !ok {
		br = newBreaker(rt.Now)
		rt.breakers[r.AdapterKey] = br
	}
	rt.mu.Unlock()
	opts := Options{ID: r.ID, Manifest: m, Endpoint: r.EndpointUrl, Secrets: keys, AllowPrivate: allow, PublicURL: public,
		HTTP: rt.HTTP, Now: rt.Now, Log: rt.log(), breaker: br}
	in, err := NewInstallation(opts)
	if err != nil {
		opts.Endpoint = ""
		in, err2 := NewInstallation(opts)
		if err2 != nil {
			return nil, err2
		}
		in.endpointErr = err.Error()
		return in, nil
	}
	return in, nil
}

// ParseManifest implements manage.CommunityRuntime: it validates a manifest
// of kind "remote" and returns its registry key and the capabilities the
// Adapters tab lists, in the reader's language.
func (rt *Runtime) ParseManifest(raw []byte, l i18n.Lang) (string, []string, error) {
	m, err := ParseManifest(raw)
	if err != nil {
		return "", nil, err
	}
	return m.Key(), m.LabelsIn(l), nil
}

// Installed implements manage.CommunityRuntime: inside the transaction that
// installs an adapter it creates the runtime row with a new signing secret,
// sealed (an instance admin issues one to read with `reputecast adapter
// secret`, and sets the address with `reputecast adapter set-url`).
func (rt *Runtime) Installed(ctx context.Context, tx pgx.Tx, adapterID uuid.UUID, raw []byte) error {
	m, err := ParseManifest(raw)
	if err != nil {
		return api.Invalid("body.manifestUrl", mProblemManifest.Text(i18n.LangFrom(ctx), i18n.Args{"error": err.Error()}))
	}
	_, err = createRuntime(ctx, store.New(tx), rt.Box, adapterID, m, "")
	return err
}

// Changed implements manage.CommunityRuntime: it re-registers the installed
// adapters after an install, update or removal is committed.
func (rt *Runtime) Changed(ctx context.Context) {
	if err := rt.Sync(context.WithoutCancel(ctx)); err != nil {
		rt.syncFailed(err)
	}
}

// keyTaken returns the name of the installed adapter, other than except,
// registered under key ("" when there is none).
func keyTaken(ctx context.Context, q *store.Queries, key string, except uuid.UUID) (string, error) {
	all, err := q.CommunityAdapterList(ctx)
	if err != nil {
		return "", err
	}
	for _, a := range all {
		if a.ID != except && providers.InstalledKey(a.Kind, a.Name, a.Manifest) == key {
			return a.Name, nil
		}
	}
	return "", nil
}

// createRuntime stores the runtime row of a newly installed adapter and
// returns its signing secret (to show once, or never).
func createRuntime(ctx context.Context, q *store.Queries, box *secrets.Box, adapterID uuid.UUID, m Manifest, endpoint string) (string, error) {
	if other, err := keyTaken(ctx, q, m.Key(), adapterID); err != nil {
		return "", err
	} else if other != "" {
		return "", api.Conflict(mProblemKeyTaken.Text(i18n.LangFrom(ctx), i18n.Args{"other": other, "key": m.Key()}))
	}
	secret := NewSecret()
	sealed, err := box.SealString(secret, SecretAD(adapterID))
	if err != nil {
		return "", err
	}
	if err := q.RemoteRuntimeCreate(ctx, store.RemoteRuntimeCreateParams{AdapterID: adapterID, AdapterKey: m.Key(), EndpointUrl: endpoint, SecretEnc: sealed}); err != nil {
		return "", err
	}
	return secret, nil
}
