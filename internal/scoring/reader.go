package scoring

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// Reading scores elsewhere: the contacts table and its filters, segments,
// the MCP server and, in phase 3, conditional content at send time. Live
// scores are contact_scores.score; a contact without a row scores 0 once
// the workspace has scoring. The band follows from the applied version's
// thresholds.

// AppliedBands returns the thresholds live scores fall into, and whether
// the workspace has scoring at all.
func AppliedBands(ctx context.Context, q *store.Queries, wsID uuid.UUID) (ScoringBands, bool, error) {
	row, err := q.ScoringConfigGet(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.AppliedVersion == 0) {
		return DefaultBands, false, nil
	}
	if err != nil {
		return DefaultBands, false, err
	}
	v, err := q.ScoringVersionGet(ctx, store.ScoringVersionGetParams{WorkspaceID: wsID, Version: row.AppliedVersion})
	if err != nil {
		return DefaultBands, false, err
	}
	cfg, err := ParseConfig(v.Config)
	if err != nil {
		return DefaultBands, false, err
	}
	return cfg.Bands, true, nil
}

// ContactBand is a contact's live score and band.
type ContactBand struct {
	Score int
	Band  string
}

// Bands reads the live score and band of many contacts at once (the
// conditional-content renderer reads its recipients' with it at send
// time). ok is false when the workspace has no scoring: nobody has a band.
func Bands(ctx context.Context, q *store.Queries, wsID uuid.UUID, contacts []uuid.UUID) (out map[uuid.UUID]ContactBand, ok bool, err error) {
	bands, ok, err := AppliedBands(ctx, q, wsID)
	if err != nil || !ok {
		return map[uuid.UUID]ContactBand{}, ok, err
	}
	out = make(map[uuid.UUID]ContactBand, len(contacts))
	for _, id := range contacts {
		out[id] = ContactBand{Score: 0, Band: bands.Band(0)}
	}
	for start := 0; start < len(contacts); start += 10_000 {
		rows, err := q.ScoringScoresOf(ctx, contacts[start:min(start+10_000, len(contacts))])
		if err != nil {
			return nil, true, err
		}
		for _, r := range rows {
			out[r.ContactID] = ContactBand{Score: int(r.Score), Band: bands.Band(int(r.Score))}
		}
	}
	return out, true, nil
}
