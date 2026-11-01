package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Batched email events (review 2026-11 EN-8). An endpoint that batches
// gets its email.* events as one POST of type email.batch with up to
// BatchSize events, so a million-recipient send is ten thousand requests
// rather than a million.
const (
	// EmailBatch is the type of a batched delivery.
	EmailBatch = "email.batch"
	// BatchSize is the most events one batch carries.
	BatchSize = 100
	// batchesPerSweep bounds the batches one dispatcher sweep makes, like
	// the deliveries it queues.
	batchesPerSweep = 500
)

// BatchData is the data of an email.batch event: the events in the order
// they happened, each the envelope it would have been posted as alone.
type BatchData struct {
	Count  int               `json:"count"`
	Events []json.RawMessage `json:"events"`
}

// foldBatches turns the email.* events waiting for a batch into deliveries
// of up to BatchSize events, at most limit of them. Each batch is made in
// its own transaction; it returns how many were made.
func (s *Service) foldBatches(ctx context.Context, limit int) (int, error) {
	hooks, err := s.Q.HooksBatching(ctx)
	if err != nil {
		return 0, err
	}
	made := 0
	for _, h := range hooks {
		for made < limit {
			n, err := s.foldBatch(ctx, h)
			if err != nil {
				return made, err
			}
			if n == 0 {
				break
			}
			made++
			if n < BatchSize {
				break
			}
		}
	}
	return made, nil
}

// foldBatch makes one batch from an endpoint's oldest waiting events and
// returns how many it took. The batch is held while the endpoint is paused,
// like any other delivery.
func (s *Service) foldBatch(ctx context.Context, h store.HooksBatchingRow) (int, error) {
	n := 0
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rows, err := q.HooksTakeBatch(ctx, store.HooksTakeBatchParams{WebhookID: h.ID, Lim: BatchSize})
		if err != nil || len(rows) == 0 {
			return err
		}
		slices.SortFunc(rows, func(a, b store.HooksTakeBatchRow) int {
			if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
				return c
			}
			return bytes.Compare(a.ID[:], b.ID[:])
		})
		events := make([]json.RawMessage, len(rows))
		for i, r := range rows {
			events[i] = r.Payload
		}
		now := time.Now() // due at once, as in Emit
		eventID := uuid.Must(uuid.NewV7())
		body, err := json.Marshal(Envelope{ID: eventID.String(), Type: EmailBatch, CreatedAt: now.UTC(), Workspace: h.Slug,
			Data: BatchData{Count: len(rows), Events: events}})
		if err != nil {
			return err
		}
		state := "pending"
		if !h.Enabled {
			state = "held"
		}
		n = len(rows)
		return q.HooksInsertDelivery(ctx, store.HooksInsertDeliveryParams{
			ID: uuid.Must(uuid.NewV7()), WebhookID: h.ID, Event: EmailBatch, EventID: &eventID, Payload: body,
			State: state, NextAttemptAt: &now, MaxAttempts: MaxAttempts,
		})
	})
	return n, err
}
