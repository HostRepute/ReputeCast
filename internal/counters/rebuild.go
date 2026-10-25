package counters

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Why a campaign's counters can't be rebuilt.
const (
	// SkipImported: history imported from another platform has its final
	// numbers but no message per recipient.
	SkipImported = "imported"
	// SkipNoHistory: fewer messages are stored than the counters count
	// (sample data), so a rebuild would lose numbers.
	SkipNoHistory = "no_history"
	// SkipNotStarted: nothing was sent.
	SkipNotStarted = "not_started"
)

// ErrNotFound means the campaign doesn't exist.
var ErrNotFound = errors.New("counters: campaign not found")

// Result says what a rebuild did.
type Result struct {
	Rebuilt bool
	Skipped string
}

// Rebuild recomputes a campaign's counters, hourly engagement and clicks
// per link from its messages and their stored events, in one transaction:
// the states the status can't tell apart are filled from the events first,
// then every counter is replaced by what the states add up to. Writers
// that move the campaign's messages meanwhile wait for it (it locks the
// counter rows first). A campaign whose per-message history isn't stored
// is left as it is, with the reason.
func Rebuild(ctx context.Context, pool *pgxpool.Pool, campaign uuid.UUID, now time.Time) (Result, error) {
	var res Result
	err := db.Tx(ctx, pool, func(tx pgx.Tx) error {
		res = Result{}
		q := store.New(tx)
		h, err := q.CountersCampaignHistory(ctx, campaign)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		switch {
		case h.Origin != "":
			res.Skipped = SkipImported
		case h.StartedAt == nil:
			res.Skipped = SkipNotStarted
		case h.MessagesSent < h.CountedSent:
			res.Skipped = SkipNoHistory
		}
		if res.Skipped == "" {
			if err := q.CountersLockCampaign(ctx, campaign); err != nil {
				return err
			}
			cid := campaign
			if err := q.CountersRefreshStates(ctx, &cid); err != nil {
				return err
			}
			if err := q.CountersRebuildCells(ctx, campaign); err != nil {
				return err
			}
			if err := q.CountersRebuildHourly(ctx, store.CountersRebuildHourlyParams{CampaignID: campaign, StartedAt: *h.StartedAt}); err != nil {
				return err
			}
			if err := q.CountersRebuildLinks(ctx, campaign); err != nil {
				return err
			}
			res.Rebuilt = true
		}
		return q.CountersBuildSave(ctx, store.CountersBuildSaveParams{CampaignID: campaign, Version: Version, BuiltAt: now, Skipped: res.Skipped})
	})
	return res, err
}

// Unsubscribed records that a reader unsubscribed from a message (the
// unsubscribe page, one-click, or the provider's own unsubscribe): the
// message moves once, and its campaign's unsubscribe count with it.
func Unsubscribed(ctx context.Context, q *store.Queries, message uuid.UUID, at time.Time) error {
	mv, err := q.CountersMoveDelivery(ctx, store.CountersMoveDeliveryParams{ID: message, Kind: "unsubscribe", At: at})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var l Ledger
	l.Add(FromDelivery(mv))
	return l.Flush(ctx, q)
}
