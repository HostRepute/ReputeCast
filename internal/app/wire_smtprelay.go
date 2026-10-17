package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/smtprelay"
)

// The SMTP relay (design expansion phase 1): off until an instance admin
// turns it on; web processes run its listeners.

var smtpRelays sync.Map // *App → *smtprelay.Relay

func smtpRelay(a *App) *smtprelay.Relay {
	if v, ok := smtpRelays.Load(a); ok {
		return v.(*smtprelay.Relay)
	}
	r := &smtprelay.Relay{Pool: a.Pool, Q: a.Q, Box: a.Secrets, Audit: a.Audit, Log: a.Log, Sender: transactionalService(a)}
	v, _ := smtpRelays.LoadOrStore(a, r)
	return v.(*smtprelay.Relay)
}

// SMTPRelay is the App's SMTP relay (tests drive it).
func (a *App) SMTPRelay() *smtprelay.Relay { return smtpRelay(a) }

// RunSMTPRelay runs the SMTP relay's listeners until ctx ends, in processes
// with the web role (the relay is an API surface, like the HTTP API). It
// returns once the listeners stopped gracefully.
func (a *App) RunSMTPRelay(ctx context.Context) error {
	if !a.Cfg.HasRole(config.RoleWeb) {
		<-ctx.Done()
		return nil
	}
	return smtpRelay(a).Run(ctx)
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		smtprelay.Register(h, smtpRelay(a), a.currentPublicURL)
	})
	// Let's Encrypt's HTTP-01 challenge for the relay's hostname.
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		r.Handle("/.well-known/acme-challenge/*", smtpRelay(a).ACMEHandler())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		smtprelay.RegisterJobs(a.JobReg, smtpRelay(a), a.currentPublicURL)
	})
}
