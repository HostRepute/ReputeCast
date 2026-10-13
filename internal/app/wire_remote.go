package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/providers/remote"
)

// W16 · Community adapter runtime. One runtime per App registers the
// installed community adapters in the provider registry and keeps them in
// step with the database (every process re-reads them every 15 seconds).
var remoteRuntimes sync.Map // *App → *remote.Runtime

func (a *App) remoteRuntime() *remote.Runtime {
	if rt, ok := remoteRuntimes.Load(a); ok {
		return rt.(*remote.Runtime)
	}
	rt := &remote.Runtime{
		Pool: a.Pool, Box: a.Secrets, Log: a.Log,
		AllowPrivate: func() bool { return a.Cfg != nil && a.Cfg.Adapters.AllowPrivateNetworks },
		PublicURL:    func() string { return a.URLs.Base() },
	}
	actual, _ := remoteRuntimes.LoadOrStore(a, rt)
	return actual.(*remote.Runtime)
}

func (a *App) remoteService() *remote.Service {
	return &remote.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Runtime: a.remoteRuntime(),
		AllowPrivate: func() bool { return a.Cfg != nil && a.Cfg.Adapters.AllowPrivateNetworks },
	}
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
