package texting

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// BudgetAlerts checks the month's spend against the budget now and sends
// its notices, as the dispatcher does after a round.
func (s *Service) BudgetAlerts(ctx context.Context, ws *store.Workspace) error {
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return err
	}
	s.budgetAlerts(ctx, ws, r, nil)
	return nil
}

// CheckAutoPause runs a send's auto-pause check, as a receipt or an
// opt-out does.
func (s *Service) CheckAutoPause(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) {
	s.checkAutoPause(ctx, ws, campaignID)
}
