package engine

import (
	"context"

	"github.com/google/uuid"
)

// LockSendForTest takes a send's lock as a process running it would.
func LockSendForTest(e *Engine, ctx context.Context, id uuid.UUID) (func(), error) {
	return e.lockSend(ctx, id)
}
