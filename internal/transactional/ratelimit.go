package transactional

import (
	"context"
	"math"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Each key's send rate (requests per second, set on the key; 50 by
// default) is a token bucket in Postgres, so every web node behind a load
// balancer takes from the same tokens (review 2026-11 TX-4). The bucket
// holds a second's worth; a batch larger than that waits for a full one.

// takeTokens takes n requests' worth from the key's bucket. When it can't,
// it returns how long until it can.
func (s *Service) takeTokens(ctx context.Context, key uuid.UUID, perSecond, n int, now time.Time) (bool, time.Duration, error) {
	if perSecond <= 0 {
		perSecond = 50
	}
	if n > perSecond {
		n = perSecond
	}
	rate, need := float64(perSecond), float64(n)
	var ok bool
	var wait time.Duration
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		b, err := q.TxnRateBucketLock(ctx, store.TxnRateBucketLockParams{BucketKey: key, Burst: rate, Now: now})
		if err != nil {
			return err
		}
		// Refill by the time since the last take; a node whose clock is
		// behind the last taker's adds nothing.
		at, tokens := b.RefilledAt, b.Tokens
		if now.After(at) {
			tokens += now.Sub(at).Seconds() * rate
			at = now
		}
		tokens = math.Min(tokens, rate)
		if tokens >= need {
			ok, tokens = true, tokens-need
		} else {
			wait = time.Duration((need - tokens) / rate * float64(time.Second))
		}
		return q.TxnRateBucketSet(ctx, store.TxnRateBucketSetParams{BucketKey: key, Tokens: tokens, RefilledAt: at})
	})
	return ok, wait, err
}

func retrySeconds(d time.Duration) int { return max(1, int(math.Ceil(d.Seconds()))) }

func invalidDetail(location, message string) *huma.ErrorDetail {
	return &huma.ErrorDetail{Location: location, Message: message}
}
