// Package search answers the command palette (review 2026-11 FE-1): one
// call searches campaigns, contacts, automations, lists, segments, signup
// forms and website pages by name, five of a kind. A kind the caller's
// role can't open is never searched, so results never show what they
// can't open.
package search

import (
	"context"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service runs searches.
type Service struct {
	Q *store.Queries
}

// SearchHit is one result.
type SearchHit struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Status is the item's own status word (sent, draft, live…), for the
	// web app to phrase.
	Status string `json:"status,omitempty"`
	// Detail is a secondary line that needs no translating: a contact's
	// name, a form's kind.
	Detail string `json:"detail,omitempty"`
	// Channel is a campaign's channel (email, sms, push, whatsapp).
	Channel string `json:"channel,omitempty"`
	// Count is a campaign's recipients, a list's subscribed contacts or a
	// segment's last count.
	Count *int64     `json:"count,omitempty"`
	At    *time.Time `json:"at,omitempty" doc:"A campaign's start, or its scheduled time"`
	// Slug is a page's address.
	Slug string `json:"slug,omitempty"`
}

// SearchGroup is one kind's results.
type SearchGroup struct {
	Kind  string      `json:"kind" enum:"campaigns,contacts,automations,lists,segments,forms,pages"`
	Items []SearchHit `json:"items"`
}

// SearchResults is a search's answer, kinds in a fixed order, empty kinds left
// out.
type SearchResults struct {
	Query  string        `json:"query"`
	Groups []SearchGroup `json:"groups"`
}

// kinds lists what is searched and the permission that opens it.
var kinds = []struct {
	kind string
	perm authz.Permission
}{
	{"campaigns", authz.CampaignsView},
	{"contacts", authz.SubscribersView},
	{"automations", authz.AutomationsView},
	{"lists", authz.SubscribersView},
	{"segments", authz.SubscribersView},
	{"forms", authz.SubscribersView},
	{"pages", authz.WebsiteView},
}

// Search finds q in every kind the principal may open.
func (s *Service) Search(ctx context.Context, ws *store.Workspace, p *authz.Principal, q string) (SearchResults, error) {
	q = strings.TrimSpace(q)
	out := SearchResults{Query: q, Groups: []SearchGroup{}}
	if len([]rune(q)) < 2 {
		return out, nil
	}
	lower := strings.ToLower(q)
	pattern, prefix := "%"+db.EscapeLike(lower)+"%", db.EscapeLike(lower)+"%"
	for _, k := range kinds {
		if !p.Can(k.perm) {
			continue
		}
		var hits []SearchHit
		var err error
		switch k.kind {
		case "campaigns":
			hits, err = s.campaigns(ctx, ws, pattern, prefix)
		case "contacts":
			hits, err = s.contacts(ctx, ws, p, lower, pattern, prefix)
		case "automations":
			hits, err = s.automations(ctx, ws, pattern, prefix)
		case "lists":
			hits, err = s.lists(ctx, ws, p, pattern, prefix)
		case "segments":
			hits, err = s.segments(ctx, ws, pattern, prefix)
		case "forms":
			hits, err = s.forms(ctx, ws, pattern, prefix)
		case "pages":
			hits, err = s.pages(ctx, ws, pattern, prefix)
		}
		if err != nil {
			return out, err
		}
		if len(hits) > 0 {
			out.Groups = append(out.Groups, SearchGroup{Kind: k.kind, Items: hits})
		}
	}
	return out, nil
}

func (s *Service) campaigns(ctx context.Context, ws *store.Workspace, pattern, prefix string) ([]SearchHit, error) {
	rows, err := s.Q.SearchCampaigns(ctx, store.SearchCampaignsParams{WorkspaceID: ws.ID, Pattern: pattern, Prefix: prefix})
	out := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		at := r.StartedAt
		if at == nil {
			at = r.ScheduledAt
		}
		out = append(out, SearchHit{ID: r.ID.String(), Title: r.Name, Status: r.Status, Channel: r.Channel, Count: r.AudienceCount, At: at})
	}
	return out, err
}

func (s *Service) contacts(ctx context.Context, ws *store.Workspace, p *authz.Principal, lower, pattern, prefix string) ([]SearchHit, error) {
	exact := ""
	if strings.Contains(lower, "@") && !strings.ContainsAny(lower, " %_") {
		exact = lower
	}
	rows, err := s.Q.SearchContacts(ctx, store.SearchContactsParams{WorkspaceID: ws.ID, Scope: p.Scope.Lists, Exact: exact, Pattern: pattern, Prefix: prefix})
	out := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, SearchHit{ID: r.ID.String(), Title: r.Email, Detail: strings.TrimSpace(r.FirstName + " " + r.LastName), Status: r.Status})
	}
	return out, err
}

func (s *Service) automations(ctx context.Context, ws *store.Workspace, pattern, prefix string) ([]SearchHit, error) {
	rows, err := s.Q.SearchAutomations(ctx, store.SearchAutomationsParams{WorkspaceID: ws.ID, Pattern: pattern, Prefix: prefix})
	out := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, SearchHit{ID: r.ID.String(), Title: r.Name, Status: r.Status})
	}
	return out, err
}

func (s *Service) lists(ctx context.Context, ws *store.Workspace, p *authz.Principal, pattern, prefix string) ([]SearchHit, error) {
	rows, err := s.Q.SearchLists(ctx, store.SearchListsParams{WorkspaceID: ws.ID, Scope: p.Scope.Lists, Pattern: pattern, Prefix: prefix})
	out := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		n := r.Subscribed
		out = append(out, SearchHit{ID: r.ID.String(), Title: r.Name, Count: &n})
	}
	return out, err
}

func (s *Service) segments(ctx context.Context, ws *store.Workspace, pattern, prefix string) ([]SearchHit, error) {
	rows, err := s.Q.SearchSegments(ctx, store.SearchSegmentsParams{WorkspaceID: ws.ID, Pattern: pattern, Prefix: prefix})
	out := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, SearchHit{ID: r.ID.String(), Title: r.Name, Count: r.CachedCount})
	}
	return out, err
}

func (s *Service) forms(ctx context.Context, ws *store.Workspace, pattern, prefix string) ([]SearchHit, error) {
	rows, err := s.Q.SearchForms(ctx, store.SearchFormsParams{WorkspaceID: ws.ID, Pattern: pattern, Prefix: prefix})
	out := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, SearchHit{ID: r.ID.String(), Title: r.Name, Detail: r.Kind, Status: r.Status})
	}
	return out, err
}

func (s *Service) pages(ctx context.Context, ws *store.Workspace, pattern, prefix string) ([]SearchHit, error) {
	rows, err := s.Q.SearchPages(ctx, store.SearchPagesParams{WorkspaceID: ws.ID, Pattern: pattern, Prefix: prefix})
	out := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, SearchHit{ID: r.ID.String(), Title: r.Name, Status: r.Status, Slug: r.Slug})
	}
	return out, err
}

// Register adds the search operation. Any member may call it; each kind is
// searched only with the permission that opens it.
func Register(h huma.API, s *Service) {
	api.Get(h, api.Op{ID: "search-workspace", Path: "/workspaces/{workspace}/search",
		Summary:     "Search campaigns, contacts, automations, lists, segments, forms and pages by name (the command palette)",
		Description: "Five results a kind. Kinds the caller's role can't open aren't searched; a member scoped to lists finds only their contacts and lists.",
		Tags:        []string{"Search"}, Access: api.Workspace("")},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Q string `query:"q" maxLength:"200" doc:"At least two characters"`
		}) (*api.Body[SearchResults], error) {
			out, err := s.Search(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Q)
			return api.OK(out), err
		})
}
