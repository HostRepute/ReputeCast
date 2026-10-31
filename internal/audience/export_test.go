package audience

import (
	"context"

	"github.com/hostrepute/reputecast/internal/store"
)

// ListSQL is the contacts table's page query for a filter, for the plan
// tests.
func ListSQL(ctx context.Context, s *Service, ws *store.Workspace, f ContactFilter, pg ContactPage) (string, []any, error) {
	fs, err := s.filterSQL(ctx, ws.ID, f)
	if err != nil {
		return "", nil, err
	}
	q, err := s.listQuery(ctx, fs, f, pg)
	return q.sql, q.args, err
}
