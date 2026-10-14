package sending

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/db"
)

// Tx runs fn in a transaction and retries it when Postgres aborts it for a
// deadlock or a serialization failure. Concurrent batches update the same
// counter rows and can lock them in different orders; an aborted attempt
// commits nothing, so retrying is safe. fn must not keep state between
// attempts.
func Tx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	var err error
	for attempt := range 6 {
		if err = db.Tx(ctx, pool, fn); !Retryable(err) {
			return err
		}
		wait := time.Duration(10<<attempt)*time.Millisecond + time.Duration(rand.IntN(10))*time.Millisecond //nolint:gosec // jitter, not a secret
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return err
}

// Retryable reports whether err is a deadlock or a serialization failure.
func Retryable(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && (pg.Code == "40P01" || pg.Code == "40001")
}
