package manage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// RecordCall counts one API call to a provider: the per-minute window
// behind "API error rate · 3 min" and outage detection, the daily stats
// behind the 7-day error rate and cost, and, for failures, the provider's
// recent errors. The sending pipeline calls it after every Send (with the
// number of messages the provider accepted and their cost); the health job
// after every check.
func RecordCall(ctx context.Context, q store.DBTX, p store.Provider, op string, callErr error, accepted int, at time.Time) error {
	qs := store.New(q)
	errs := int32(0)
	if callErr != nil {
		errs = 1
	}
	if err := qs.ProvRecordCall(ctx, store.ProvRecordCallParams{ProviderID: p.ID, At: at, Errors: errs, Messages: int32(accepted)}); err != nil {
		return err
	}
	cost := int64(0)
	if pr := state.PricingOf(p); pr.Model == "metered" && accepted > 0 {
		cost = int64(float64(accepted) * pr.Per1000 * 1000) // micros: per1000 dollars / 1000 * 1e6
	}
	day := at.UTC()
	if err := qs.ProvRecordDaily(ctx, store.ProvRecordDailyParams{ProviderID: p.ID, Day: day, Sent: int64(accepted), Errors: int64(errs), CostMicros: cost}); err != nil {
		return err
	}
	if callErr == nil {
		return nil
	}
	rec := CallRecordOf(op, callErr, at)
	b, _ := json.Marshal(rec)
	_, err := q.Exec(ctx, `UPDATE providers SET health = jsonb_set(
		jsonb_set(coalesce(health, '{}'::jsonb), '{lastError}', $2::jsonb),
		'{recentErrors}', jsonb_path_query_array($3::jsonb || coalesce(health->'recentErrors', '[]'::jsonb), '$[0 to 9]'))
		WHERE id = $1`, p.ID, b, "["+string(b)+"]")
	return err
}

// RecordSend records the outcome of one Send call: the call itself and how
// many messages the provider accepted.
func RecordSend(ctx context.Context, q store.DBTX, p store.Provider, results []providers.SendResult, callErr error, at time.Time) error {
	accepted := 0
	for _, r := range results {
		if r.Err == nil {
			accepted++
		}
	}
	op := "Send"
	var ce *providers.CallError
	if errors.As(callErr, &ce) && ce.Op != "" {
		op = ce.Op
	}
	return RecordCall(ctx, q, p, op, callErr, accepted, at)
}

// CallRecordOf describes a failed call for the recent-errors list.
func CallRecordOf(op string, err error, at time.Time) state.CallRecord {
	rec := state.CallRecord{At: at.UTC(), Op: op, Message: err.Error()}
	var ce *providers.CallError
	if errors.As(err, &ce) {
		if ce.Op != "" {
			rec.Op = ce.Op
		}
		rec.Status, rec.Code, rec.Message = ce.Status, ce.Code, ce.Message
		if rec.Message == "" && ce.Cause != nil {
			rec.Message = ce.Cause.Error()
		}
	}
	return rec
}

// newID returns a fresh UUIDv7.
func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }
