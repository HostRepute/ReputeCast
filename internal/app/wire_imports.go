package app

import (
	"net"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		imports.Register(h, a.imports())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		imports.RegisterJobs(a.JobReg, a.imports())
	})
}

// imports returns the importer service. The River client is created after
// the job registrars run, so the service reads it lazily.
func (a *App) imports() *imports.Service {
	// listmonk and Keila addresses come from workspace editors: the importer
	// dials through the SSRF guard (metadata and link-local addresses are
	// always refused), like provider URLs.
	guard := netguard.Policy{AllowPrivate: a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks}
	s := &imports.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Audience: a.audience(), Mail: a.Mail,
		MX:    &imports.MXChecker{Q: a.Q, Resolver: net.DefaultResolver},
		HTTP:  guard.Client(60 * time.Second),
		Guard: guard,
		Jobs:  func() *jobs.Client { return a.Jobs },
		// Notifuse's templates come along as transactional drafts and saved templates.
		Templates: notifuseTemplates{a: a},
		// Verification jobs run in the background too (built when asked:
		// the verification service uses this one).
		JobSources: []imports.JobSource{verificationTray(a)},
		// The optional "Verify the new addresses" folds a service's answers
		// into the preview and the import (X14), read when asked.
		Checks: importChecks{a: a},
	}
	if a.Secrets != nil {
		s.Box = a.Secrets
	}
	return s
}
