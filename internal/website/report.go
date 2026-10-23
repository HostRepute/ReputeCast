package website

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// WebsiteSource is one row of a post's sources: a referrer or a UTM source.
type WebsiteSource struct {
	// Key is the referring domain ("" for email clients and direct, "other"
	// for the rest) or the utm_source value.
	Key   string `json:"key"`
	Label string `json:"label"`
	Views int64  `json:"views"`
}

// WebsiteCampaignReport is the Web section of a campaign's report.
type WebsiteCampaignReport struct {
	PostID      string     `json:"postId"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Status      string     `json:"status" enum:"draft,scheduled,published,unpublished"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	// Views are the last 30 days; ViewsAllTime every view.
	Views        int64 `json:"views"`
	ViewsAllTime int64 `json:"viewsAllTime"`
	// Signups are members who joined from the post, Upgrades readers who
	// upgraded to a paid tier from it (30 days).
	Signups   int64           `json:"signups"`
	Upgrades  int64           `json:"upgrades"`
	Referrers []WebsiteSource `json:"referrers"`
	UTM       []WebsiteSource `json:"utm"`
	From      time.Time       `json:"from"`
	Until     time.Time       `json:"until"`
}

// referrerNames are the well-known referrers by their domain.
var referrerNames = map[string]string{
	"google.com": "Google", "bing.com": "Bing", "duckduckgo.com": "DuckDuckGo", "linkedin.com": "LinkedIn", "lnkd.in": "LinkedIn",
	"x.com": "X", "twitter.com": "X", "t.co": "X", "facebook.com": "Facebook", "reddit.com": "Reddit", "bsky.app": "Bluesky",
}

// referrerLabel names a referrer in the reader's language: its domain, a
// well-known name, or "Email clients and direct" for none.
func referrerLabel(ctx context.Context, key string) string {
	if key == "" {
		return tr(ctx, "report.direct", nil)
	}
	k := strings.TrimPrefix(key, "www.")
	if n, ok := referrerNames[k]; ok {
		return n
	}
	for d, n := range referrerNames {
		if strings.HasSuffix(k, "."+d) {
			return n
		}
	}
	return k
}

// CampaignReport is the Web section of a campaign's report (nil when the
// campaign has no post on the site).
func (s *Service) CampaignReport(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (*WebsiteCampaignReport, error) {
	post, err := s.Q.WebsitePostByCampaign(ctx, &campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if post.WorkspaceID != ws.ID || (post.Status != PostPublished && post.Status != PostUnpublished) {
		return nil, nil
	}
	loc := location(ws)
	until := day(s.now(), loc)
	from := until.AddDate(0, 0, -29)
	out := &WebsiteCampaignReport{PostID: post.ID.String(), Title: post.Title, URL: s.postURL(ws, post.Slug), Status: post.Status, PublishedAt: post.PublishedAt,
		Referrers: []WebsiteSource{}, UTM: []WebsiteSource{}, From: from, Until: until}
	counts, err := s.Q.WebsitePostViewCounts(ctx, store.WebsitePostViewCountsParams{WorkspaceID: ws.ID, Posts: []uuid.UUID{post.ID}, FromDay: from, UntilDay: until})
	if err != nil {
		return nil, err
	}
	for _, c := range counts {
		out.Views += c.Views
	}
	all, err := s.Q.WebsitePostViewCounts(ctx, store.WebsitePostViewCountsParams{WorkspaceID: ws.ID, Posts: []uuid.UUID{post.ID}, FromDay: time.Time{}, UntilDay: until})
	if err != nil {
		return nil, err
	}
	for _, c := range all {
		out.ViewsAllTime += c.Views
	}
	conv, err := s.Q.WebsiteConversionCounts(ctx, store.WebsiteConversionCountsParams{WorkspaceID: ws.ID, Posts: []uuid.UUID{post.ID}, FromDay: from,
		UntilDay: until.AddDate(0, 0, 1)})
	if err != nil {
		return nil, err
	}
	for _, c := range conv {
		switch c.Kind {
		case "signup":
			out.Signups += c.N
		case "upgrade":
			out.Upgrades += c.N
		}
	}
	src, err := s.Q.WebsitePostSources(ctx, store.WebsitePostSourcesParams{PostID: post.ID, FromDay: from, UntilDay: until})
	if err != nil {
		return nil, err
	}
	refs := map[string]int64{}
	order := []string{}
	for _, r := range src {
		switch r.Dim {
		case "referrer":
			label := referrerLabel(ctx, r.Key)
			if _, ok := refs[label]; !ok {
				order = append(order, label)
			}
			refs[label] += r.Views
		case "utm":
			out.UTM = append(out.UTM, WebsiteSource{Key: r.Key, Label: "utm_source=" + r.Key, Views: r.Views})
		}
	}
	// The top four referrers, the rest as Other.
	sortByViews(order, refs)
	var other int64
	for i, label := range order {
		if i < 4 {
			out.Referrers = append(out.Referrers, WebsiteSource{Key: label, Label: label, Views: refs[label]})
			continue
		}
		other += refs[label]
	}
	if other > 0 {
		out.Referrers = append(out.Referrers, WebsiteSource{Key: "other", Label: tr(ctx, "report.other", nil), Views: other})
	}
	if len(out.UTM) > 5 {
		out.UTM = out.UTM[:5]
	}
	return out, nil
}

// sortByViews orders labels by their views, most first (stable).
func sortByViews(labels []string, views map[string]int64) {
	for i := 1; i < len(labels); i++ {
		for j := i; j > 0 && views[labels[j]] > views[labels[j-1]]; j-- {
			labels[j], labels[j-1] = labels[j-1], labels[j]
		}
	}
}
