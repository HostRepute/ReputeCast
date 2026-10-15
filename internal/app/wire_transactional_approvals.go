package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// The approval rule for publishing transactional templates: the approvals
// domain is the template service's publish gate, and the publish dialog
// asks for approval through it.
func init() {
	transactionalGate = func(a *App) transactional.PublishGate { return &lazyPublishGate{a: a} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		s := newApprovals(a)
		s.TransactionalTemplates = transactionalService(a)
		approvals.RegisterTransactional(h, s)
	})
}

// lazyPublishGate builds the approvals service the first time a publish
// asks: building it while the transactional service is being built would
// go round in a circle (approvals → Send Readiness → deliverability → the
// webhook normalizer → the transactional service).
type lazyPublishGate struct {
	a    *App
	once sync.Once
	gate approvals.TransactionalGate
}

func (l *lazyPublishGate) get() approvals.TransactionalGate {
	l.once.Do(func() { l.gate = approvals.TransactionalGate{S: newApprovals(l.a)} })
	return l.gate
}

// Check implements transactional.PublishGate.
func (l *lazyPublishGate) Check(ctx context.Context, q *store.Queries, req transactional.PublishRequest) (transactional.PublishDecision, error) {
	return l.get().Check(ctx, q, req)
}

// Published implements transactional.PublishGate.
func (l *lazyPublishGate) Published(ctx context.Context, q *store.Queries, req transactional.PublishRequest, version int) error {
	return l.get().Published(ctx, q, req, version)
}
