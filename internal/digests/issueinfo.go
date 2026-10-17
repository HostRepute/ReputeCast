package digests

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// DigestIssueInfo is a campaign's digest issue, for the screens that show
// a campaign: the approvals inbox and review, Review & send and the report.
type DigestIssueInfo struct {
	CampaignID string `json:"campaignId"`
	DigestID   string `json:"digestId"`
	DigestName string `json:"digestName"`
	IssueID    string `json:"issueId"`
	// Number is the issue's number, or the one it gets when it goes out.
	Number     int        `json:"number"`
	Status     string     `json:"status" enum:"collecting,preparing,prepared,waiting,held,draft,sending,sent,skipped,not_sent"`
	HoldReason string     `json:"holdReason,omitempty" enum:"readiness,changes,rejected,no_approvers,"`
	Mode       string     `json:"mode" enum:"auto,draft"`
	PreparedAt *time.Time `json:"preparedAt,omitempty"`
	SendAt     *time.Time `json:"sendAt,omitempty"`
	// Deadline is the late limit: not out by then, the issue is skipped.
	Deadline *time.Time    `json:"deadline,omitempty"`
	Owner    *DigestPerson `json:"owner,omitempty"`
	// Posts are the issue's posts, in its order.
	Posts []DigestFeedItem `json:"posts"`
	// Since is what changed since the last approved issue (nil for the first).
	Since *DigestIssueSince `json:"since,omitempty"`
	// PreviousCampaignID is the previous sent issue, for the report's comparison.
	PreviousCampaignID string `json:"previousCampaignId,omitempty"`
	PreviousNumber     int    `json:"previousNumber,omitempty"`
	// ReleaseHelpers name who can release a held issue (campaigns:send).
	ReleaseHelpers []string `json:"releaseHelpers"`
}

// DigestIssueSince compares an issue with the last approved one.
type DigestIssueSince struct {
	Number     int        `json:"number"`
	ApprovedAt *time.Time `json:"approvedAt,omitempty"`
	SentAt     *time.Time `json:"sentAt,omitempty"`
	// Changed: template, audience, route, schedule; empty when nothing changed.
	Changed []string `json:"changed"`
}

// DigestIssueInfos are the digest issues among some campaigns.
type DigestIssueInfos struct {
	Items []DigestIssueInfo `json:"items"`
}

// IssueInfos describes the campaigns that are digest issues; the others
// are left out.
func (s *Service) IssueInfos(ctx context.Context, ws *store.Workspace, campaignIDs []uuid.UUID) (DigestIssueInfos, error) {
	out := DigestIssueInfos{Items: []DigestIssueInfo{}}
	if len(campaignIDs) == 0 {
		return out, nil
	}
	rows, err := s.Q.DigestIssuesByCampaigns(ctx, store.DigestIssuesByCampaignsParams{WorkspaceID: ws.ID, CampaignIds: campaignIDs})
	if err != nil {
		return out, err
	}
	var ms []member
	if len(rows) > 0 {
		if ms, err = s.members(ctx, ws.ID); err != nil {
			return out, err
		}
	}
	seen := map[uuid.UUID]bool{}
	for _, r := range rows {
		if r.CampaignID == nil || seen[*r.CampaignID] {
			continue
		}
		seen[*r.CampaignID] = true
		info := DigestIssueInfo{CampaignID: r.CampaignID.String(), DigestID: r.DigestID.String(), DigestName: r.DigestName, IssueID: r.ID.String(),
			HoldReason: r.HoldReason, Mode: r.Mode, PreparedAt: r.PreparedAt, SendAt: r.SendAt, Deadline: r.Deadline, Posts: []DigestFeedItem{},
			ReleaseHelpers: helpers(ms, authz.CampaignsSend, r.DigestOwnerID)}
		if r.DigestOwnerID != nil && r.OwnerName != "" {
			info.Owner = person(*r.DigestOwnerID, r.OwnerName, r.OwnerColor)
		}
		if r.Number != nil {
			info.Number = int(*r.Number)
		} else if n, err := s.Q.DigestNextNumber(ctx, r.DigestID); err == nil {
			info.Number = int(n)
		}
		var approval *DigestIssueApproval
		if open, err := s.Q.DigestOpenApproval(ctx, r.CampaignID); err == nil {
			approval = &DigestIssueApproval{ID: open.ID.String(), Status: open.Status}
		}
		info.Status = displayStatus(r.Status, approval)
		posts, err := s.Q.DigestIssueItems(ctx, r.ID)
		if err != nil {
			return out, err
		}
		for _, it := range posts {
			status, sentIn := "new", 0
			if r.Status == "sending" || r.Status == "sent" {
				status, sentIn = "sent", info.Number
			}
			info.Posts = append(info.Posts, DigestFeedItem{ID: it.ID.String(), Title: it.Title, URL: it.Url, Author: it.Author, Published: it.PublishedAt,
				Categories: nonNil(it.Categories), Tags: nonNil(it.Tags), Excerpt: it.Excerpt, Image: it.MediaUrl, ImageAlt: it.ImageAlt,
				AltFromTitle: it.ImageAlt == "" || it.ImageAlt == it.Title, Status: status, SentIn: sentIn, FirstSeenAt: it.FirstSeenAt, FirstSeenVia: it.FirstSeenVia})
		}
		if last, err := s.Q.DigestLastApprovedIssue(ctx, store.DigestLastApprovedIssueParams{DigestID: r.DigestID, ID: r.ID}); err == nil && last.Number != nil {
			if last.CampaignID != nil {
				info.PreviousCampaignID, info.PreviousNumber = last.CampaignID.String(), int(*last.Number)
			}
			var was, now issueBasis
			_ = json.Unmarshal(last.Basis, &was)
			_ = json.Unmarshal(r.Basis, &now)
			since := &DigestIssueSince{Number: int(*last.Number), ApprovedAt: last.ApprovedAt, SentAt: last.StartedAt, Changed: []string{}}
			// An issue from before bases were kept is compared with nothing.
			if len(last.Basis) > 2 && len(r.Basis) > 2 {
				since.Changed = changedSince(was, now)
			}
			info.Since = since
		}
		out.Items = append(out.Items, info)
	}
	return out, nil
}
