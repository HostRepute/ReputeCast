package revenue

import (
	"context"
	"time"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// silentAfter is how long without an event from the provider, while
// subscriptions are active, the card warns about the webhook.
const silentAfter = 24 * time.Hour

// Card is the Earn card of the workspace Overview: MRR, paid members,
// churn and trials over 30 days, the next sponsored issue, and a webhook
// that went quiet; before a payment provider, an invitation to set Earn up.
func (s *Service) Card(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnCard, error) {
	out := EarnCard{State: "none", Currency: "usd", CanSetUp: p.Can(authz.PaymentsManage)}
	conns, err := s.Q.RevenueConnections(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	var conn *store.RevenueConnectionsRow
	for i := range conns {
		if conns[i].Status == "connected" && (conn == nil || conns[i].Mode == "live") {
			conn = &conns[i]
		}
	}
	if conn == nil {
		out.State = "invite"
		return out, nil
	}
	out.State, out.Account, out.Mode = "live", conn.AccountName, conn.Mode
	if d, ok := payments.Lookup(conn.Provider); ok {
		out.ProviderName = d.Name
	}
	if st, err := s.Q.RevenueAttributionSettings(ctx, ws.ID); err == nil && st.Currency != "" {
		out.Currency = st.Currency
	}
	from, to := s.window(ws, "30d")
	ov := RevenueOverview{}
	if err := s.movement(ctx, ws, from, to, &ov); err != nil {
		return out, err
	}
	out.MRRCents, out.MRRDeltaCents = ov.Tiles.MRRCents, ov.Tiles.MRRDeltaCents
	out.Members, out.MembersDelta, out.ChurnRate = ov.Tiles.Members, ov.Tiles.MembersDelta, ov.Tiles.ChurnRate
	tr, err := s.Q.RevenueTrials(ctx, store.RevenueTrialsParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to, Soon: to})
	if err != nil {
		return out, err
	}
	out.Trialing = tr.Trialing
	loc := location(ws)
	next, err := s.Q.RevenueNextSponsor(ctx, store.RevenueNextSponsorParams{WorkspaceID: ws.ID, Today: civilDay(s.now(), loc)})
	if err != nil {
		return out, err
	}
	if len(next) == 1 {
		n := next[0]
		ci := &CardIssue{BookingID: n.ID.String(), Issue: int(n.IssueNumber), Sponsor: n.Sponsor, Ready: n.CreativeStatus == "ready"}
		if n.CreativeDue != nil {
			d := time.Date(n.CreativeDue.Year(), n.CreativeDue.Month(), n.CreativeDue.Day(), 12, 0, 0, 0, loc)
			ci.CreativeDue = &d
		}
		out.NextSponsor = ci
	}
	active, err := s.Q.RevenueActiveSubscriptions(ctx, store.RevenueActiveSubscriptionsParams{WorkspaceID: ws.ID, Mode: conn.Mode})
	if err != nil {
		return out, err
	}
	out.Active = active
	if active > 0 && conn.LastEventAt != nil && s.now().Sub(*conn.LastEventAt) > silentAfter {
		out.SilentHours = int64(s.now().Sub(*conn.LastEventAt) / time.Hour)
	}
	return out, nil
}
