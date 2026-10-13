package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/providers/httpgeneric"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/remote"
)

// W16 · Community adapter runtime. One runtime per App registers the
// installed community adapters in the provider registry and keeps them in
// step with the database (every process re-reads them every 15 seconds):
// remote adapters (their own process, the adapter protocol) and installed
// HTTP mappings, opened through the generic HTTP adapter.
var remoteRuntimes sync.Map // *App → *remote.Runtime

func (a *App) remoteRuntime() *remote.Runtime {
	if rt, ok := remoteRuntimes.Load(a); ok {
		return rt.(*remote.Runtime)
	}
	rt := &remote.Runtime{
		Pool: a.Pool, Box: a.Secrets, Log: a.Log,
		AllowPrivate: func() bool { return a.Cfg != nil && a.Cfg.Adapters.AllowPrivateNetworks },
		PublicURL:    func() string { return a.URLs.Base() },
		Kinds:        map[string]remote.KindDescriptor{"http": httpgeneric.CommunityDescriptor},
	}
	actual, _ := remoteRuntimes.LoadOrStore(a, rt)
	return actual.(*remote.Runtime)
}

// CommunityRuntime is the App's community adapter runtime (tests that change
// the configuration stop its sync loop first).
func (a *App) CommunityRuntime() *remote.Runtime { return a.remoteRuntime() }

func (a *App) remoteService() *remote.Service {
	return &remote.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Runtime: a.remoteRuntime(),
		AllowPrivate: func() bool { return a.Cfg != nil && a.Cfg.Adapters.AllowPrivateNetworks },
	}
}

// httpMappings gives the Adapters tab the generic HTTP adapter's mapping
// check and preview (manage never imports an adapter).
type httpMappings struct{}

func (httpMappings) Check(raw []byte) error {
	_, err := httpgeneric.ParseManifest(raw)
	return err
}

func (httpMappings) Preview(raw []byte) (manage.AdapterHTTPPreview, error) {
	r, err := httpgeneric.PreviewManifest(raw)
	if err != nil {
		return manage.AdapterHTTPPreview{}, err
	}
	return manage.NewAdapterHTTPPreview(r.Method, r.URL, r.Header, r.Body), nil
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		remote.Register(h, a.remoteService())
	})
	startHooks = append(startHooks, func(a *App) error {
		a.remoteRuntime().Start(context.Background())
		return nil
	})
}
