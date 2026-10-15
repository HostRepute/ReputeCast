package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// The approval rule for publishing transactional templates: the approvals
// domain is the template service's publish gate, and the publish dialog
// asks for approval through it.
func init() {
	transactionalGate = func(a *App) transactional.PublishGate { return lazyTransactionalGate{a: a} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		s := newApprovals(a)
		s.TransactionalTemplates = transactionalService(a)
		approvals.RegisterTransactional(h, s)
	})
}

// lazyTransactionalGate builds the approvals gate when a publish asks for
// it rather than with the transactional service: the approvals service
// reaches the sending engine, whose normalizer reaches the transactional
// service, so building it up front never ends.
type lazyTransactionalGate struct{ a *App }

func (g lazyTransactionalGate) gate() approvals.TransactionalGate {
	return approvals.TransactionalGate{S: newApprovals(g.a)}
}

func (g lazyTransactionalGate) Check(ctx context.Context, q *store.Queries, req transactional.PublishRequest) (transactional.PublishDecision, error) {
	return g.gate().Check(ctx, q, req)
}

func (g lazyTransactionalGate) Published(ctx context.Context, q *store.Queries, req transactional.PublishRequest, version int) error {
	return g.gate().Published(ctx, q, req, version)
}
