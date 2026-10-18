package website

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/store"
)

// SiteFrame is what every page of the site shows around its content (the
// header, the navigation and the footer): the frame the pages of later
// phases share (internal/publicpages/templates/site/frame.html).
type SiteFrame struct {
	// On is the website being on: the navigation, Subscribe and Sign in.
	// Off, public pages and the shop keep the logo in place of the
	// navigation.
	On bool
	// SigninOn is "Members can sign in" (Sign in in the header).
	SigninOn bool
	// Title is the site's title ("The Acme newsletter") and Short the
	// publisher's name for phones ("Acme"); Tagline is under the title.
	Title   string
	Short   string
	Tagline string
	// Description is the site's description (search engines, link previews).
	Description string
	SocialImage string
	Index       bool
	// Nav are the header links, resolved: a path on the site ("/tag/engineering")
	// or an outside address.
	Nav    []FrameLink
	Footer WebsiteFooter
}

// FrameLink is a header link.
type FrameLink struct {
	Kind  string
	Label string
	// Path is a path on the site ("/", "/tag/engineering", "/about"), or
	// URL an outside address.
	Path string
	URL  string
	// Key identifies the page the link opens ("tag:<slug>", "page:<slug>",
	// "home", "archive") so the current page can mark it.
	Key string
}

// PageNames resolves the custom pages header links point at (the page
// builder registers it, design expansion phase 7 M29): a page's title by
// its slug, ok false when it doesn't exist or isn't published.
type PageNames func(ctx context.Context, q *store.Queries, workspaceID uuid.UUID, slugs []string) (map[string]string, error)

var (
	pageNamesMu sync.RWMutex
	pageNames   PageNames
)

// RegisterPageNames lets the page builder name the pages of header links.
func RegisterPageNames(f PageNames) {
	pageNamesMu.Lock()
	defer pageNamesMu.Unlock()
	pageNames = f
}

func resolvePageNames(ctx context.Context, q *store.Queries, wsID uuid.UUID, slugs []string) map[string]string {
	pageNamesMu.RLock()
	f := pageNames
	pageNamesMu.RUnlock()
	if f == nil || len(slugs) == 0 {
		return map[string]string{}
	}
	out, err := f(ctx, q, wsID, slugs)
	if err != nil || out == nil {
		return map[string]string{}
	}
	return out
}

// Frame returns the site's frame for a workspace.
func (s *Service) Frame(ctx context.Context, ws *store.Workspace, set *forms.Settings) (SiteFrame, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return SiteFrame{}, err
	}
	return s.frame(ctx, s.Q, ws, set, c)
}

func (s *Service) frame(ctx context.Context, q *store.Queries, ws *store.Workspace, set *forms.Settings, c *config) (SiteFrame, error) {
	f := SiteFrame{On: c.On(), SigninOn: c.On() && c.row.SigninOn, Title: siteTitle(c, set), Short: set.Branding.PublisherName,
		Tagline: c.row.Tagline, Description: c.row.Description, SocialImage: c.row.SocialImage, Index: c.row.IndexSite,
		Nav: []FrameLink{}, Footer: c.Footer}
	if f.Short == "" {
		f.Short = f.Title
	}
	if !f.On {
		return f, nil
	}
	links, err := s.resolveNav(ctx, q, ws, c.Header)
	if err != nil {
		return f, err
	}
	f.Nav = links
	return f, nil
}

// resolveNav turns header links into labels and paths; links to tags or
// pages that are gone are left out.
func (s *Service) resolveNav(ctx context.Context, q *store.Queries, ws *store.Workspace, header []WebsiteNavLink) ([]FrameLink, error) {
	out := []FrameLink{}
	if len(header) == 0 {
		return out, nil
	}
	tags := map[string]store.WebsiteTagsRow{}
	needTags := false
	var pages []string
	for _, l := range header {
		switch l.Kind {
		case NavTag:
			needTags = true
		case NavPage:
			pages = append(pages, l.Ref)
		}
	}
	if needTags {
		rows, err := q.WebsiteTags(ctx, ws.ID)
		if err != nil {
			return nil, err
		}
		for _, t := range rows {
			tags[t.ID.String()] = t
		}
	}
	names := resolvePageNames(ctx, q, ws.ID, pages)
	for _, l := range header {
		fl := FrameLink{Kind: l.Kind, Label: strings.TrimSpace(l.Label)}
		switch l.Kind {
		case NavHome:
			fl.Path, fl.Key = "/", "home"
		case NavArchive:
			fl.Path, fl.Key = "/archive", "archive"
		case NavTag:
			t, ok := tags[l.Ref]
			if !ok {
				continue
			}
			if fl.Label == "" {
				fl.Label = t.Name
			}
			fl.Path, fl.Key = "/tag/"+t.Slug, "tag:"+t.Slug
		case NavPage:
			slug := strings.Trim(l.Ref, "/")
			if slug == "" {
				continue
			}
			if fl.Label == "" {
				fl.Label = names[slug]
			}
			if fl.Label == "" {
				fl.Label = titleOfSlug(slug)
			}
			fl.Path, fl.Key = "/"+slug, "page:"+slug
		case NavURL:
			if !strings.HasPrefix(l.URL, "https://") && !strings.HasPrefix(l.URL, "http://") {
				continue
			}
			fl.URL = l.URL
			if fl.Label == "" {
				fl.Label = strings.TrimPrefix(strings.TrimPrefix(l.URL, "https://"), "http://")
			}
		default:
			continue
		}
		if fl.Label == "" {
			continue
		}
		out = append(out, fl)
	}
	return out, nil
}

// titleOfSlug names a page from its address ("field-guide" → "Field guide").
func titleOfSlug(slug string) string {
	s := strings.ReplaceAll(slug, "-", " ")
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
