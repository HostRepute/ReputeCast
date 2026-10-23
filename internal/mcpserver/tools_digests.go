package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/digests"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// DigestReader lists recurring digests and their issues (X10).
type DigestReader interface {
	List(ctx context.Context, ws *store.Workspace) (digests.DigestList, error)
	Issues(ctx context.Context, ws *store.Workspace, id uuid.UUID) (digests.DigestIssues, error)
}

// DigestRef names a digest by name or id.
type DigestRef struct {
	Digest string `json:"digest" jsonschema:"the digest's name (exact) or id"`
}

// addDigestTools adds list_digests and get_digest_issues. Both only read:
// agents can't turn a digest on, release or skip an issue, and never send.
func (s *Service) addDigestTools(srv *mcp.Server) {
	tool(s, srv, "list_digests", authz.CampaignsView, func(ctx context.Context, c *agentCall, _ struct{}) (any, entry, error) {
		e := entry{action: "digest.listed", targetType: "digest", targetText: keep("target.digests", nil)}
		if s.Digests == nil {
			return map[string]any{"digests": []any{}, "message": "Recurring digests aren't available on this server."}, e, nil
		}
		list, err := s.Digests.List(ctx, &c.Workspace)
		if err != nil {
			return nil, e, err
		}
		type issue struct {
			Number     int        `json:"number,omitempty"`
			Status     string     `json:"status"`
			SendAt     *time.Time `json:"send_at,omitempty"`
			Posts      int        `json:"posts"`
			HumanOpens *float64   `json:"human_opens_pct,omitempty"`
			Approval   bool       `json:"needs_approval"`
		}
		type row struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			State    string `json:"state"`
			Mode     string `json:"mode"`
			Feed     string `json:"feed"`
			Schedule string `json:"schedule"`
			Owner    string `json:"owner,omitempty"`
			Next     *issue `json:"next_issue,omitempty"`
			Last     *issue `json:"last_issue,omitempty"`
		}
		ref := func(r *digests.DigestIssueRef) *issue {
			if r == nil {
				return nil
			}
			return &issue{Number: r.Number, Status: r.Status, SendAt: r.SendAt, Posts: r.Posts, HumanOpens: r.HumanOpens, Approval: r.NeedsApproval}
		}
		rows := []row{}
		for _, d := range list.Items {
			r := row{ID: d.ID, Name: d.Name, State: d.State, Mode: d.Mode, Feed: d.Source.Label, Schedule: d.Schedule.Describe(), Next: ref(d.Next), Last: ref(d.Last)}
			if len(d.Source.Categories) > 0 {
				r.Feed += " · " + strings.Join(d.Source.Categories, ", ")
			}
			if d.Owner != nil {
				r.Owner = d.Owner.Name
			}
			rows = append(rows, r)
		}
		e.detailText = keep("detail.digests", i18n.Args{"count": len(rows)})
		return map[string]any{"digests": rows,
			"note": "Issues go out on their owner's authority after Send Readiness and the approval rules. Agents can read digests but can't turn one on, release or skip an issue."}, e, nil
	}, nil)

	tool(s, srv, "get_digest_issues", authz.CampaignsView, func(ctx context.Context, c *agentCall, in DigestRef) (any, entry, error) {
		e := entry{action: "digest.issues_read", targetType: "digest", targetName: in.Digest}
		if s.Digests == nil {
			return map[string]any{"issues": []any{}, "message": "Recurring digests aren't available on this server."}, e, nil
		}
		list, err := s.Digests.List(ctx, &c.Workspace)
		if err != nil {
			return nil, e, err
		}
		var found *digests.DigestListItem
		for i, d := range list.Items {
			if d.ID == strings.TrimSpace(in.Digest) || strings.EqualFold(d.Name, strings.TrimSpace(in.Digest)) {
				found = &list.Items[i]
				break
			}
		}
		if found == nil {
			return nil, e, fmt.Errorf("no digest named or with id %q in this workspace", in.Digest)
		}
		e.targetID, e.targetName = found.ID, found.Name
		id, _ := uuid.Parse(found.ID)
		issues, err := s.Digests.Issues(ctx, &c.Workspace, id)
		if err != nil {
			return nil, e, err
		}
		type row struct {
			Number     *int       `json:"number,omitempty"`
			Status     string     `json:"status"`
			SendAt     *time.Time `json:"send_at,omitempty"`
			Posts      int        `json:"posts"`
			Recipients *int64     `json:"recipients,omitempty"`
			Estimated  bool       `json:"recipients_estimated,omitempty"`
			HumanOpens *float64   `json:"human_opens_pct,omitempty"`
			Clicks     *float64   `json:"clicks_pct,omitempty"`
			SkipReason string     `json:"skip_reason,omitempty"`
			HoldReason string     `json:"hold_reason,omitempty"`
			ApprovedBy string     `json:"approved_by,omitempty"`
			CampaignID string     `json:"campaign_id,omitempty"`
		}
		rows := []row{}
		for _, is := range issues.Items {
			r := row{Number: is.Number, Status: is.Status, SendAt: is.SendAt, Posts: is.Posts, Recipients: is.Recipients, Estimated: is.Estimated,
				HumanOpens: is.HumanOpens, Clicks: is.Clicks, SkipReason: is.SkipReason, HoldReason: is.HoldReason, CampaignID: is.CampaignID}
			if is.Approval != nil && is.Approval.DecidedBy != nil && is.Approval.Status != "pending" {
				r.ApprovedBy = is.Approval.DecidedBy.Name
			}
			rows = append(rows, r)
		}
		e.detailText = keep("detail.issues", i18n.Args{"count": len(rows)})
		return map[string]any{"digest": found.Name, "issues": rows}, e, nil
	}, nil)
}
