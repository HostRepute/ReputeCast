package website

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// A custom page's "Show in navigation" (the page builder's Page tab,
// design expansion phase 7 M29) keeps a header link to the page: added
// when the page goes live with it on, renamed or moved with the page,
// removed when it's turned off or the page leaves the site. The header
// still holds at most MaxHeaderLinks links; a page that doesn't fit isn't
// linked (the Page tab says so).

// PageLinkState is a page's place in the header.
type PageLinkState struct {
	// Linked: the header links to the page.
	Linked bool
	// Full: the header has no room for another link.
	Full bool
}

// PageLink says whether the header links to a page's address.
func (s *Service) PageLink(ctx context.Context, q *store.Queries, ws *store.Workspace, slug string) (PageLinkState, error) {
	c, err := s.load(ctx, q, ws)
	if err != nil {
		return PageLinkState{}, err
	}
	out := PageLinkState{Full: len(c.Header) >= MaxHeaderLinks}
	for _, l := range c.Header {
		if l.Kind == NavPage && strings.Trim(l.Ref, "/") == slug {
			out.Linked = true
		}
	}
	return out, nil
}

// SetPageLink adds, updates or removes the header link of a page, in the
// caller's transaction. from is the address the link pointed at before (a
// page that moved); label overrides the page's name ("Field Guide").
// It reports whether the header changed.
func (s *Service) SetPageLink(ctx context.Context, q *store.Queries, ws *store.Workspace, by *uuid.UUID, from, slug, label string, on bool) (bool, error) {
	c, err := s.lock(ctx, q, ws)
	if err != nil {
		return false, err
	}
	before := slices.Clone(c.Header)
	idx := -1
	for i, l := range c.Header {
		ref := strings.Trim(l.Ref, "/")
		if l.Kind == NavPage && (ref == slug || (from != "" && ref == from)) {
			idx = i
			break
		}
	}
	label = strings.TrimSpace(label)
	if len([]rune(label)) > 40 {
		label = string([]rune(label)[:40])
	}
	switch {
	case on && idx >= 0:
		c.Header[idx].Ref, c.Header[idx].Label = slug, label
	case on && len(c.Header) < MaxHeaderLinks:
		c.Header = append(c.Header, WebsiteNavLink{Kind: NavPage, Ref: slug, Label: label})
	case !on && idx >= 0:
		c.Header = slices.Delete(c.Header, idx, idx+1)
	}
	if slices.Equal(before, c.Header) {
		return false, nil
	}
	return true, s.save(ctx, q, ws, c, by)
}
