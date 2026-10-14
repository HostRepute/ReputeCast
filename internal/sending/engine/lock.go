package engine

import (
	"context"
	"errors"
	"hash/fnv"

	"github.com/google/uuid"
)

// lockNamespace is the first key of the send loops' advisory locks.
const lockNamespace = 0x5e4d

// errBusy means another process runs this campaign's send loop right now.
var errBusy = errors.New("the send loop runs in another process")

// lockSend holds a session advisory lock for a campaign's send loop, so a
// send never runs twice at once: River already keeps one run job per
// campaign, and the lock also covers the demo command's fast-forward and a
// worker taking over from one that stalled. The lock goes away with the
// connection if the process dies.
func (e *Engine) lockSend(ctx context.Context, id uuid.UUID) (func(), error) {
	conn, err := e.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	h := fnv.New32a()
	_, _ = h.Write(id[:])
	key := int32(h.Sum32())
	var ok bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1, $2)`, int32(lockNamespace), key).Scan(&ok); err != nil {
		conn.Release()
		return nil, err
	}
	if !ok {
		conn.Release()
		return nil, errBusy
	}
	return func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1, $2)`, int32(lockNamespace), key)
		conn.Release()
	}, nil
}
