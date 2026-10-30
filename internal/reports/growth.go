package reports

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// Growth from an issue (review AN-12): the subscribers it brought, by how
// they came, and its views on the web. Upgrades and attributed revenue are
// Earn's (the campaign's revenue, the same figures as the report's Revenue
// section), credited when this was the last email clicked within the
// attribution window before checkout.
//
//   - Forwards: people who subscribed from a view-online page of one of the
//     issue's messages (someone forwarded it, or shared its web version),
//     or through a subscribe link tagged utm_medium=forward with the
//     campaign's id as utm_campaign.
//   - Referrals: referral signups whose referrer's last click, as a
//     person, in the 7 days before was on the issue (the rule upgrades are
//     credited by).
//   - Web post: people who subscribed from the issue's archive page or its
//     website post.
//   - Web views: the issue's archive page and website post, counted on the
//     server without cookies (one view per visitor, page and day).

// ReportGrowth is "Growth & revenue from this issue" without the money,
// which comes from Earn.
type ReportGrowth struct {
	// Since is when the issue went out: the counts are since then.
	Since          *time.Time `json:"since,omitempty"`
	NewSubscribers int64      `json:"newSubscribers"`
	Forwards       int64      `json:"forwards"`
	Referrals      int64      `json:"referrals"`
	WebPost        int64      `json:"webPost"`
	WebViews       int64      `json:"webViews"`
}

func (s *Service) growth(ctx context.Context, ws *store.Workspace, c store.ReportCampaignRow) (ReportGrowth, error) {
	out := ReportGrowth{Since: c.StartedAt}
	if c.StartedAt == nil {
		return out, nil
	}
	since := *c.StartedAt
	pages, err := s.Q.ReportIssuePages(ctx, store.ReportIssuePagesParams{WorkspaceID: ws.ID, CampaignID: c.ID})
	if err != nil {
		return out, err
	}
	postPath := "-"
	if pages.PostSlug != "" {
		postPath = "%/" + pages.PostSlug
	}
	joins, err := s.Q.ReportGrowthJoins(ctx, store.ReportGrowthJoinsParams{WorkspaceID: ws.ID, Since: since, PostPath: postPath})
	if err != nil {
		return out, err
	}
	// View-online pages: each one's message, then which are the issue's.
	viewed := map[uuid.UUID][]int{}
	var ids []uuid.UUID
	for i, j := range joins {
		switch {
		case strings.EqualFold(j.UtmMedium, "forward"):
			// A subscribe link for readers of a forwarded copy, tagged
			// with the campaign.
			if strings.EqualFold(j.UtmCampaign, c.ID.String()) {
				out.Forwards++
			}
		case onIssuePage(j.Page, ws.Slug, pages.ArchiveSlug, pages.PostSlug) || onIssuePage(j.Referrer, ws.Slug, pages.ArchiveSlug, pages.PostSlug):
			out.WebPost++
		default:
			if id, ok := s.viewedMessage(j.Referrer); ok {
				if viewed[id] == nil {
					ids = append(ids, id)
				}
				viewed[id] = append(viewed[id], i)
			}
		}
	}
	if len(ids) > 0 {
		mine, err := s.Q.ReportViewedMessages(ctx, store.ReportViewedMessagesParams{CampaignID: &c.ID, Ids: ids})
		if err != nil {
			return out, err
		}
		for _, id := range mine {
			out.Forwards += int64(len(viewed[id]))
		}
	}
	if out.Referrals, err = s.Q.ReportGrowthReferrals(ctx, store.ReportGrowthReferralsParams{WorkspaceID: ws.ID, Since: since, CampaignID: &c.ID}); err != nil {
		return out, err
	}
	out.NewSubscribers = out.Forwards + out.Referrals + out.WebPost
	if out.WebViews, err = s.Q.ReportWebViews(ctx, store.ReportWebViewsParams{WorkspaceID: ws.ID, CampaignID: c.ID}); err != nil {
		return out, err
	}
	return out, nil
}

// onIssuePage reports whether a page (host and path, as growth stores it)
// is the issue's archive page or website post.
func onIssuePage(page, workspace, archiveSlug, postSlug string) bool {
	if page == "" {
		return false
	}
	path := page
	if i := strings.Index(path, "/"); i >= 0 {
		path = path[i:]
	}
	if archiveSlug != "" {
		issue := "/archive/" + archiveSlug
		if strings.HasSuffix(path, issue) || strings.Contains(path, issue+"/") {
			return strings.Contains(path, "/p/"+workspace+"/") || !strings.Contains(path, "/p/")
		}
	}
	return postSlug != "" && strings.HasSuffix(path, "/"+postSlug)
}

// viewedMessage reads the message of a view-online page's address
// (…/p/<workspace>/view/<token>).
func (s *Service) viewedMessage(page string) (uuid.UUID, bool) {
	i := strings.Index(page, "/view/")
	if i < 0 || s.Links == nil {
		return uuid.Nil, false
	}
	token := page[i+len("/view/"):]
	if j := strings.IndexAny(token, "/?#"); j >= 0 {
		token = token[:j]
	}
	claims, err := s.Links.Verify(token, linktoken.ViewOnline)
	if err != nil || claims.MessageID == uuid.Nil {
		return uuid.Nil, false
	}
	return claims.MessageID, true
}
