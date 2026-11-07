package ingest

import (
	"context"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// Tokens keeps the one-time tokens of webhook signatures that cover a token
// but not the event (Mailgun's) in webhook_tokens, where every process and
// every restart sees them (providers.TokenStore). The sweep drops them once
// their signature is no longer accepted.
type Tokens struct {
	Q *store.Queries
}

var _ providers.TokenStore = Tokens{}

// BindToken implements providers.TokenStore in one statement: of two
// requests with the same token at once, the second gets the first one's
// payload back.
func (t Tokens) BindToken(ctx context.Context, scope string, token, payload [32]byte, now, expires time.Time) ([32]byte, error) {
	var bound [32]byte
	got, err := t.Q.IngestBindToken(ctx, store.IngestBindTokenParams{Scope: scope, TokenHash: token[:], PayloadHash: payload[:], ExpiresAt: expires, Now: now})
	if err != nil {
		return bound, err
	}
	copy(bound[:], got)
	return bound, nil
}
