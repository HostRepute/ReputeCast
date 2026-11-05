package webhooks

import (
	"context"
	"strings"
)

// FoldBatches runs the dispatcher's batching step (tests).
func (s *Service) FoldBatches(ctx context.Context, limit int) (int, error) {
	return s.foldBatches(ctx, limit)
}

// Described reports whether someone decided what an event needs: its own
// entry or its group's, not the fallback for groups nobody described.
func Described(event string) bool {
	if _, ok := eventNeeds[event]; ok {
		return true
	}
	group, _, _ := strings.Cut(event, ".")
	_, ok := groupNeeds[group]
	return ok
}
