package roles

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
)

// RoleImpactPending is an approval request by a holder that stays pending
// when the role changes (RoleEditorStates: "Their pending request for
// November Product Update stays pending; the policy still matches
// (audience ≥ 10,000)").
type RoleImpactPending struct {
	Requester string `json:"requester"`
	// Subject is the campaign, automation or website post the request is for.
	Subject string `json:"subject"`
	// StillMatches are the policy rules that keep requiring approval after
	// the change, in their short words ("audience ≥ 10,000").
	StillMatches []string `json:"stillMatches"`
}

// pendingRequests lists the holders' pending approval requests with the
// rules that still match once the role has the permissions in next: a holder
// who gains campaigns:send no longer lacks it (website:publish, for a
// website post's request).
func (s *Service) pendingRequests(ctx context.Context, wsID uuid.UUID, holders []uuid.UUID, next authz.Set) ([]RoleImpactPending, error) {
	rows, err := s.Pool.Query(ctx, `SELECT coalesce(u.name, u.email, ''), coalesce(c.name, au.name, wp.title, ''), coalesce(a.snapshot->'rules', '[]'::jsonb),
			a.website_post_id IS NOT NULL
		FROM approval_requests a
		LEFT JOIN users u ON u.id = a.requested_by
		LEFT JOIN campaigns c ON c.id = a.campaign_id
		LEFT JOIN automations au ON au.id = a.automation_id
		LEFT JOIN website_posts wp ON wp.id = a.website_post_id
		WHERE a.workspace_id = $1 AND a.status = 'pending' AND a.requested_by = ANY($2)
		ORDER BY a.requested_at, a.id`, wsID, holders)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RoleImpactPending{}
	for rows.Next() {
		var p RoleImpactPending
		var raw []byte
		var post bool
		if err := rows.Scan(&p.Requester, &p.Subject, &raw, &post); err != nil {
			return nil, err
		}
		// The send permission a rule asks of the author: publishing, for a post.
		send := authz.CampaignsSend
		if post {
			send = authz.WebsitePublish
		}
		var rules []struct {
			Key   string `json:"key"`
			Short string `json:"short"`
		}
		_ = json.Unmarshal(raw, &rules)
		p.StillMatches = []string{}
		for _, r := range rules {
			if r.Short == "" || (r.Key == "author_lacks_send" && next.Has(send)) {
				continue
			}
			p.StillMatches = append(p.StillMatches, r.Short)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
