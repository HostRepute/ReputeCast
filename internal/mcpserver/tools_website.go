package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// The publication website (design expansion phase 7): agents list and read
// posts, draft web-only posts and propose publishing one. They never
// publish, unpublish or change who can read what: a person does that.

// WebsitePosts reads and drafts posts (internal/website).
type WebsitePosts interface {
	Posts(ctx context.Context, ws *store.Workspace, p *authz.Principal, f website.WebsitePostFilter) (website.WebsitePostList, error)
	Post(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (website.WebsitePost, error)
	CreatePost(ctx context.Context, ws *store.Workspace, p *authz.Principal, in website.WebsitePostCreate) (website.WebsitePost, error)
	FindPost(ctx context.Context, ws *store.Workspace, ref string) (uuid.UUID, error)
}

// Design expansion (append-only): the website's tools.
func init() {
	Tools = append(Tools,
		MCPToolInfo{"list_posts", "The website's posts with their status, visibility, address and 30-day views. No member data.", string(authz.WebsiteView), false},
		MCPToolInfo{"get_post", "One post: its settings, tags, web checks and, for a web-only post, its MJML.", string(authz.WebsiteView), false},
		MCPToolInfo{"create_post_draft", "Draft a web-only post from a title and MJML. It stays a draft: agents never publish.", string(authz.WebsiteEdit), true},
		MCPToolInfo{"propose_post_publish", "Propose publishing a post. A person with website:publish publishes it in ReputeCast (approvals still apply).", string(authz.WebsiteEdit), true},
	)
}

// proposeVerbs are what a proposal asks to do, by tool, where the tool's
// name doesn't say it ("asked to publish Q4 roadmap deep dive").
var proposeVerbs = map[string]string{"propose_post_publish": "publish"}

type postsIn struct {
	Status string `json:"status,omitempty" jsonschema:"all (default), published, scheduled, draft or unpublished"`
	Query  string `json:"query,omitempty" jsonschema:"words of the title"`
}

type postRef struct {
	Post string `json:"post" jsonschema:"the post's id or its address (the slug after /posts/)"`
}

type postDraftIn struct {
	Title string `json:"title" jsonschema:"the post's title"`
	MJML  string `json:"mjml,omitempty" jsonschema:"the post's content as MJML (mjml > mj-body > sections)"`
}

type proposePostIn struct {
	Post string `json:"post" jsonschema:"the post's id or its address (the slug after /posts/)"`
	Note string `json:"note,omitempty" jsonschema:"why it's ready, for the person who publishes it"`
}

// addWebsiteTools registers the website's tools.
func (s *Service) addWebsiteTools(srv *mcp.Server) {
	unavailable := errors.New("the website isn't available on this server")
	tool(s, srv, "list_posts", authz.WebsiteView, func(ctx context.Context, c *agentCall, in postsIn) (any, entry, error) {
		e := entry{action: "post.list_read", targetType: "website", targetID: c.Workspace.ID.String(), targetName: c.Workspace.Name}
		if s.Website == nil {
			return nil, e, unavailable
		}
		status := in.Status
		switch status {
		case "", "all", "published", "scheduled", "draft", "unpublished":
		default:
			status = "all"
		}
		ws := c.Workspace
		list, err := s.Website.Posts(ctx, &ws, c.Principal, website.WebsitePostFilter{Status: status, Search: strings.TrimSpace(in.Query), PageSize: 50})
		if err != nil {
			return nil, e, err
		}
		posts := make([]map[string]any, 0, len(list.Posts))
		for _, p := range list.Posts {
			tags := make([]string, 0, len(p.Tags))
			for _, t := range p.Tags {
				tags = append(tags, t.Name)
			}
			m := map[string]any{"id": p.ID, "title": p.Title, "slug": p.Slug, "url": p.URL, "kind": p.Kind, "status": p.Status, "visibility": p.Visibility,
				"tags": tags, "views_30d": p.Views, "signups_30d": p.Signups, "upgrades_30d": p.Upgrades}
			if p.PublishedAt != nil {
				m["published_at"] = p.PublishedAt
			}
			if p.PublishAt != nil && p.Status == website.PostScheduled {
				m["publish_at"] = p.PublishAt
			}
			if p.Campaign != nil {
				m["campaign"] = p.Campaign.Name
			}
			posts = append(posts, m)
		}
		e.summary = fmt.Sprintf("%d posts", len(posts))
		return map[string]any{"posts": posts, "total": list.Total, "counts": list.Counts, "note": "Numbers are the last 30 days, counted on the server without cookies."}, e, nil
	}, nil)

	tool(s, srv, "get_post", authz.WebsiteView, func(ctx context.Context, c *agentCall, in postRef) (any, entry, error) {
		e := entry{action: "post.read", targetType: "website_post", targetName: in.Post}
		if s.Website == nil {
			return nil, e, unavailable
		}
		ws := c.Workspace
		id, err := s.Website.FindPost(ctx, &ws, in.Post)
		if err != nil {
			return nil, e, err
		}
		p, err := s.Website.Post(ctx, &ws, c.Principal, id)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = p.ID, p.Title
		tags := make([]string, 0, len(p.Tags))
		for _, t := range p.Tags {
			tags = append(tags, t.Name)
		}
		tiers := make([]string, 0, len(p.Tiers))
		for _, t := range p.Tiers {
			tiers = append(tiers, t.Name)
		}
		checks := make([]map[string]any, 0, len(p.Checks))
		for _, ch := range p.Checks {
			checks = append(checks, map[string]any{"check": ch.Key, "status": ch.Status})
		}
		out := map[string]any{"id": p.ID, "title": p.Title, "excerpt": p.Excerpt, "slug": p.Slug, "url": p.URL, "kind": p.Kind, "status": p.Status,
			"visibility": p.Visibility, "tiers": tiers, "never_free": p.NeverFree, "tags": tags, "byline": p.Byline.Name, "words": p.Words,
			"reading_minutes": p.ReadingMinutes, "featured": p.Featured, "free_preview_marker": p.HasMarker, "web_checks": checks,
			"can_publish": false, "note": "Agents never publish. Use propose_post_publish to ask a person."}
		if p.Kind == website.KindWeb {
			out["mjml"] = p.MJML
		}
		if p.Campaign != nil {
			out["campaign"] = p.Campaign.Name
		}
		return out, e, nil
	}, nil)

	tool(s, srv, "create_post_draft", authz.WebsiteEdit, func(ctx context.Context, c *agentCall, in postDraftIn) (any, entry, error) {
		title := strings.TrimSpace(in.Title)
		e := entry{action: "post.draft_created", targetType: "website_post", targetName: title}
		if s.Website == nil {
			return nil, e, unavailable
		}
		if title == "" {
			return nil, e, errors.New("give the post a title")
		}
		ws := c.Workspace
		p, err := s.Website.CreatePost(ctx, &ws, c.Principal, website.WebsitePostCreate{Title: title, MJML: in.MJML})
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.after, e.detail = p.ID, map[string]any{"title": p.Title, "slug": p.Slug, "visibility": p.Visibility}, "draft"
		return map[string]any{"id": p.ID, "title": p.Title, "slug": p.Slug, "status": p.Status, "visibility": p.Visibility,
			"message": "Draft created. A person publishes it from Website › Posts; propose_post_publish asks one to."}, e, nil
	}, nil)

	tool(s, srv, "propose_post_publish", authz.WebsiteEdit, func(ctx context.Context, c *agentCall, in proposePostIn) (any, entry, error) {
		e := entry{action: "post.publish_proposed", targetType: "website_post", targetName: in.Post}
		if s.Website == nil {
			return nil, e, unavailable
		}
		ws := c.Workspace
		id, err := s.Website.FindPost(ctx, &ws, in.Post)
		if err != nil {
			return nil, e, err
		}
		p, err := s.Website.Post(ctx, &ws, c.Principal, id)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = p.ID, p.Title
		if p.Status == website.PostPublished {
			return nil, e, fmt.Errorf("%q is published already", p.Title)
		}
		for _, ch := range p.Checks {
			if ch.Status == "block" {
				return nil, e, fmt.Errorf("%q can't be published yet: its %s check fails; fix it first", p.Title, ch.Key)
			}
		}
		e.detail = strings.TrimSpace(in.Note)
		s.propose(ctx, c, "propose_post_publish", authz.WebsitePublish, entry{targetType: "website_post", targetID: p.ID, targetName: p.Title, detail: e.detail})
		who := "a member with website:publish"
		if helpers := access.Helpers(ctx, s.Q, c.Workspace.ID, authz.WebsitePublish); len(helpers) > 0 {
			who = helpers[0]
		}
		return map[string]any{"proposed": true, "post": p.Title, "url": p.URL,
			"message": fmt.Sprintf("Publishing needs a person: the post was proposed to %s, who publishes it in ReputeCast (approvals still apply). The proposal expires in %d hours.", who, c.Settings.ApprovalExpiryHours)}, e, nil
	}, nil)
}
