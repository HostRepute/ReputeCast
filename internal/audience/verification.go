package audience

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// SelectionIDs resolves a selection to the contact ids the caller can see
// (their list scope included). The verification domain's bulk checks of a
// list, a tier, a segment, a selection or a saved view use it.
func (s *Service) SelectionIDs(ctx context.Context, ws *store.Workspace, sel Selection) ([]uuid.UUID, error) {
	return s.selectIDs(ctx, ws, sel)
}
