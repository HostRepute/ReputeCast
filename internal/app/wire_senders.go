package app

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/jobs"
)

// W10 · Senders and domains (Settings › Senders, the domain step of the
// connect flow) and their periodic DNS re-verification.

// domainsService builds the senders and domains service. In development the
// Acme sample domains answer from the demo DNS zone, so the demo matches the
// design instead of failing on real DNS.
func (a *App) domainsService() *domains.Service {
	cfg := a.Cfg
	if cfg == nil { // OpenAPI generation registers operations on an empty App
		d := config.Defaults()
		cfg = &d
	}
	resolver := dnscheck.NewResolver(dnscheck.Options{Servers: cfg.DNS.Servers, Timeout: cfg.DNS.Timeout})
	ns := lookupNS(cfg)
	ttl := domains.LookupTTL(cfg.DNS.Servers, cfg.DNS.Timeout)
	switch {
	case cfg.IsDevelopment():
		resolver = demo.SendersDNSOverlay(demo.DeliverabilityDNSOverlay(resolver), a.URLs.Base)
		ns = demo.SendersNS(ns)
		ttl = demo.SendersTTL(ttl)
	case cfg.Env == "test":
		// Integration tests never touch real DNS: the demo zone answers,
		// everything else doesn't exist.
		resolver = demo.SendersDNSOverlay(demo.DeliverabilityDNSOverlay(dnscheck.NewZone()), a.URLs.Base)
		ns = demo.SendersNS(nil)
		ttl = demo.SendersTTL(nil)
	}
	return &domains.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Factory: a.providerFactory(), DNS: resolver, NS: ns, TTL: ttl,
		URLs:       a.URLs,
		Cloudflare: &domains.Cloudflare{HTTP: &http.Client{Timeout: 20 * time.Second}},
		Jobs:       func() *jobs.Client { return a.Jobs }, Log: a.Log,
	}
}

// lookupNS returns the nameservers of a zone through the configured DNS
// servers (or the system resolver).
func lookupNS(cfg *config.Config) func(context.Context, string) ([]string, error) {
	r := net.DefaultResolver
	if len(cfg.DNS.Servers) > 0 {
		server := cfg.DNS.Servers[0]
		if _, _, err := net.SplitHostPort(server); err != nil {
			server = net.JoinHostPort(server, "53")
		}
		d := &net.Dialer{Timeout: cfg.DNS.Timeout}
		r = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return d.DialContext(ctx, network, server)
		}}
	}
	return func(ctx context.Context, name string) ([]string, error) {
		recs, err := r.LookupNS(ctx, name)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(recs))
		for _, n := range recs {
			out = append(out, n.Host)
		}
		return out, nil
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		domains.Register(h, a.domainsService())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &domains.CheckWorker{S: a.domainsService()})
		a.JobReg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return domains.CheckArgs{}, nil }, false)
	})
}
