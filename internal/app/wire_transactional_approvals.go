package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// The approval rule for publishing transactional templates: the approvals
// domain is the template service's publish gate, and the publish dialog
// asks for approval through it.
func init() {
	transactionalGate = func(a *App) transactional.PublishGate { return approvals.TransactionalGate{S: newApprovals(a)} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		s := newApprovals(a)
		s.TransactionalTemplates = transactionalService(a)
		approvals.RegisterTransactional(h, s)
	})
}
