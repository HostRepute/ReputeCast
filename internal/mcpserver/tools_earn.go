package mcpserver

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/store"
)

// PaidMembersReader counts paid members (internal/earn).
type PaidMembersReader interface {
	PaidMemberCounts(ctx context.Context, ws *store.Workspace) (earn.PaidMemberSummary, error)
}

// Design expansion (append-only): Earn's read-only tool.
func init() {
	Tools = append(Tools, MCPToolInfo{"get_paid_member_counts", "Paid members by status and tier, and the MRR: counts only, never names or emails.", string(authz.EarnView), false})
}

// addEarnTools registers get_paid_member_counts.
func (s *Service) addEarnTools(srv *mcp.Server) {
	tool(s, srv, "get_paid_member_counts", authz.EarnView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "earn.paid_member_counts_read", targetType: "workspace", targetID: c.Workspace.ID.String(), targetName: c.Workspace.Name}
		if s.PaidMembers == nil {
			return nil, e, errors.New("paid members aren't available in this process")
		}
		ws := c.Workspace
		sum, err := s.PaidMembers.PaidMemberCounts(ctx, &ws)
		if err != nil {
			return nil, e, err
		}
		return map[string]any{"counts": sum.Counts, "mrr_cents": sum.MRRCents, "currency": sum.Currency, "tiers": sum.Tiers, "mode": sum.Mode,
			"note": "Live paid members only. MRR counts yearly prices as a twelfth, after discounts; trials, complimentary access and gifts add nothing to it."}, e, nil
	}, nil)
}
