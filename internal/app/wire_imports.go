package app

import (
	"net"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/jobs"
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
	s := &imports.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Audience: a.audience(), Mail: a.Mail,
		MX:   &imports.MXChecker{Q: a.Q, Resolver: net.DefaultResolver},
		HTTP: &http.Client{Timeout: 60 * time.Second},
		Jobs: func() *jobs.Client { return a.Jobs },
	}
	if a.Secrets != nil {
		s.Box = a.Secrets
	}
	return s
}
