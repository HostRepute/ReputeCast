package transactional

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// Each key's send rate (requests per second, set on the key; 50 by
// default) is a token bucket in Postgres, so every web node behind a load
// balancer takes from the same tokens (review 2026-11 TX-4). The bucket
// holds a second's worth; a batch larger than that waits for a full one.

// takeTokens takes n requests' worth from the key's bucket. When it can't,
// it returns how long until it can. A take is one statement that commits
// without waiting for its WAL flush, so a key's senders don't queue behind
// each other's round trips and commits (review 2026-11 OP-13: the take
// used to hold the row's lock for four round trips and a WAL flush, so a
// key's concurrent sends went through it one at a time).
func (s *Service) takeTokens(ctx context.Context, key uuid.UUID, perSecond, n int, now time.Time) (bool, time.Duration, error) {
	if perSecond <= 0 {
		perSecond = 50
	}
	rate, need := float64(perSecond), float64(min(n, perSecond))
	wait := time.Second / time.Duration(perSecond)
	// A take races only a node that creates the bucket or empties it
	// between the two reads below; three goes settle it.
	for range 3 {
		_, err := s.Q.TxnRateBucketTake(ctx, store.TxnRateBucketTakeParams{BucketKey: key, Rate: rate, Need: need, Now: now})
		if err == nil {
			return true, 0, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, 0, err
		}
		b, err := s.Q.TxnRateBucketGet(ctx, key)
		if errors.Is(err, pgx.ErrNoRows) {
			// The key's first take: its bucket starts full.
			created, err := s.Q.TxnRateBucketCreate(ctx, store.TxnRateBucketCreateParams{BucketKey: key, Tokens: rate - need, Now: now})
			if err != nil {
				return false, 0, err
			}
			if created == 1 {
				return true, 0, nil
			}
			continue
		}
		if err != nil {
			return false, 0, err
		}
		tokens := b.Tokens
		if now.After(b.RefilledAt) {
			tokens += now.Sub(b.RefilledAt).Seconds() * rate
		}
		if tokens = math.Min(tokens, rate); tokens < need {
			return false, time.Duration((need - tokens) / rate * float64(time.Second)), nil
		}
	}
	return false, wait, nil
}

func retrySeconds(d time.Duration) int { return max(1, int(math.Ceil(d.Seconds()))) }

func invalidDetail(location, message string) *huma.ErrorDetail {
	return &huma.ErrorDetail{Location: location, Message: message}
}
