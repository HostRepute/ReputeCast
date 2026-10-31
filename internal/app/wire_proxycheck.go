package app

import (
	"net/netip"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/proxycheck"
)

// Requests through a proxy ReputeCast doesn't trust (review AN-8): the
// detector watches every request before server.RealIP, and instance
// admins are told to list the proxy in http.trusted_proxies.
var proxyDetectors sync.Map // *App → *proxycheck.Detector

func proxyDetector(a *App) *proxycheck.Detector {
	if v, ok := proxyDetectors.Load(a); ok {
		return v.(*proxycheck.Detector)
	}
	d := &proxycheck.Detector{Q: a.Q, Log: a.Log, Trusted: configuredProxies(a)}
	v, _ := proxyDetectors.LoadOrStore(a, d)
	return v.(*proxycheck.Detector)
}

func proxyService(a *App) *proxycheck.Service {
	return &proxycheck.Service{Pool: a.Pool, Q: a.Q, Trusted: configuredProxies(a)}
}

// configuredProxies are the proxies http.trusted_proxies lists; none
// without a config (the OpenAPI command registers the operations only).
func configuredProxies(a *App) []netip.Prefix {
	if a.Cfg == nil {
		return nil
	}
	return trustedProxies(a.Cfg.HTTP.TrustedProxies, a.Log)
}

// ProxyDetector returns the App's untrusted proxy detector (tests flush it).
func (a *App) ProxyDetector() *proxycheck.Detector { return proxyDetector(a) }

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		proxycheck.Register(h, proxyService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &proxycheck.NotifyWorker{S: proxyService(a)})
		a.JobReg.Every(10*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return proxycheck.NotifyArgs{}, nil }, false)
	})
}
