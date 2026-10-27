package ingest

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/store"
)

// Opens and clicks a sending provider reports count when the workspace
// keeps the provider's tracking instead of ReputeCast's (review AN-4). The
// tracking service records them as it records its own: an open is a
// person's, a privacy proxy's or a scanner's; a click settles before it
// counts and is matched to the email's link; the message moves and the
// campaign's counters follow, in the one lock order (internal/counters).

// ProviderEngagement is an open or a click a provider reported, already
// stored as the delivery event EventID.
type ProviderEngagement struct {
	EventID   uuid.UUID
	MessageID uuid.UUID
	Click     bool
	// URL is the clicked link as the provider reports it (the email's own
	// link, before the provider's redirect).
	URL       string
	UserAgent string
	IP        string
	At        time.Time
}

// Engagement records provider-reported opens and clicks (the tracking
// service implements it).
type Engagement interface {
	// Begin starts a transaction's opens and clicks. Their counter moves
	// go to ledger, which the normalizer flushes with the transaction's
	// other moves; the batch's Flush applies the rest (the contacts'
	// activity) after everything else, in contact order.
	Begin(ledger *counters.Ledger) EngagementBatch
}

// EngagementBatch records one transaction's provider opens and clicks.
type EngagementBatch interface {
	Record(ctx context.Context, tx pgx.Tx, ev ProviderEngagement) error
	Flush(ctx context.Context, q *store.Queries) error
}
