package commerce

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// Attribution: an order is credited to the email whose link the buyer
// clicked last, within the store's window (1 to 30 days, 5 by default).
// The click token in the order's landing page decides first; otherwise the
// order's email, when it belongs to a contact who clicked an email within
// the window. One order is credited to one email: a campaign or an
// automation step.

// credit is the email an order is credited to.
type credit struct {
	matched      string
	messageID    *uuid.UUID
	campaignID   *uuid.UUID
	automationID *uuid.UUID
	node         string
	clickedAt    *time.Time
}

func (c credit) none() bool { return c.matched == "" || c.matched == "none" }

// match credits an order.
func (s *Service) match(ctx context.Context, q *store.Queries, st store.Store, o stores.Order, contactID *uuid.UUID) credit {
	window := time.Duration(st.AttributionDays) * 24 * time.Hour
	if o.ClickToken != "" && s.Links != nil {
		if c, err := clicktoken.Verify(s.Links, o.ClickToken); err == nil && c.WorkspaceID == st.WorkspaceID {
			if m, err := q.CommerceClickMessage(ctx, c.MessageID); err == nil && m.WorkspaceID == st.WorkspaceID && m.Stream != "transactional" {
				// The click came before the order (a minute's slack for clocks),
				// within the window.
				if !c.At.After(o.PlacedAt.Add(time.Minute)) && o.PlacedAt.Sub(c.At) <= window {
					at := c.At
					return credit{matched: "click_token", messageID: &m.ID, campaignID: m.CampaignID, automationID: m.AutomationID, node: m.AutomationNode, clickedAt: &at}
				}
			}
		}
	}
	if contactID != nil {
		r, err := q.CommerceLastClick(ctx, store.CommerceLastClickParams{ContactID: contactID, At: o.PlacedAt, Since: o.PlacedAt.Add(-window)})
		if err == nil && r.MessageID != nil {
			at := r.OccurredAt
			return credit{matched: "email", messageID: r.MessageID, campaignID: r.CampaignID, automationID: r.AutomationID, node: r.AutomationNode, clickedAt: &at}
		}
	}
	return credit{matched: "none"}
}

// record writes an order's credit.
func (s *Service) record(ctx context.Context, q *store.Queries, orderID uuid.UUID, c credit) error {
	if c.none() {
		return nil
	}
	return q.CommerceOrderAttribute(ctx, store.CommerceOrderAttributeParams{ID: orderID, Matched: c.matched, MessageID: c.messageID,
		CampaignID: c.campaignID, AutomationID: c.automationID, AutomationNode: c.node, ClickedAt: c.clickedAt})
}

// creditName names the email an order is credited to: the campaign, or the
// automation and the step's number among its emails ("Abandoned cart",
// email 1).
func (s *Service) creditName(ctx context.Context, q *store.Queries, c credit) (name, kind string, step int) {
	switch {
	case c.campaignID != nil:
		n, err := q.CommerceCampaignName(ctx, *c.campaignID)
		if err != nil {
			return "", "", 0
		}
		return n, "campaign", 0
	case c.automationID != nil:
		row, err := q.CommerceAutomationGraph(ctx, *c.automationID)
		if err != nil {
			return "", "", 0
		}
		return row.Name, "automation", emailStep(row.Graph, c.node)
	}
	return "", "", 0
}

// emailStep is an email node's number among the flow's emails, in order.
func emailStep(raw json.RawMessage, node string) int {
	if node == "" {
		return 0
	}
	g, err := automations.ParseGraph(raw)
	if err != nil {
		return 0
	}
	n := 0
	for _, id := range g.Order() {
		nd := g.Node(id)
		if nd == nil || nd.Type != automations.NodeEmail {
			continue
		}
		n++
		if id == node {
			return n
		}
	}
	return 0
}
