package digests

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// A member limited to some lists or sender domains sets a digest up, turns
// it on and releases its issues only within them, by the rule and in the
// words a campaign's send uses (internal/access): the issues go out to the
// template's audience from its sender, on the authority of whoever turned
// the digest on. The issues the schedule prepares and starts have no
// member behind them and aren't checked again.

// sendScope refuses, as a permission problem, what would let a digest's
// issues go out beyond the caller's lists or sender domains. campaignID is
// the digest's template or one of its issues, read with q as it stands.
func sendScope(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) error {
	msg, err := access.CampaignSendProblem(ctx, q, i18n.LangFrom(ctx), p, ws.ID, campaignID)
	if err != nil {
		return api.MapError(err, "Campaign")
	}
	if msg != "" {
		return api.Err(http.StatusForbidden, api.CodePermissionDenied, msg)
	}
	return nil
}
