package app

import (
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Transactional email (design expansion phase 1).

// transactionalGate decides whether publishing a template needs an
// approval. The approvals domain replaces it with the optional workspace
// rule "Publishing a transactional template needs approval"; nil means
// publishing never needs one.
var transactionalGate = func(*App) transactional.PublishGate { return nil }

// The service keeps per-process state (each key's send rate), so each App
// has exactly one.
var transactionalServices sync.Map // *App → *transactional.Service

func transactionalService(a *App) *transactional.Service {
	if v, ok := transactionalServices.Load(a); ok {
		return v.(*transactional.Service)
	}
	s := &transactional.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Links: a.Links, URLs: a.URLs, Log: a.Log,
		Jobs: func() *jobs.Client { return a.Jobs },
		Checker: &contentcheck.Checker{
			Blocklist: contentcheck.StoreBlocklist{Q: a.Q},
			Prober:    linkProber(a),
		},
		Gate: transactionalGate(a),
		Mail: func() mailer.Sender { return a.Mail },
	}
	v, _ := transactionalServices.LoadOrStore(a, s)
	return v.(*transactional.Service)
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		transactional.RegisterTemplates(h, transactionalService(a))
	})
}
