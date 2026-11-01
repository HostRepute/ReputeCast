package webhooks

import "context"

// FoldBatches runs the dispatcher's batching step (tests).
func (s *Service) FoldBatches(ctx context.Context, limit int) (int, error) {
	return s.foldBatches(ctx, limit)
}
