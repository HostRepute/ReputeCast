package website

import (
	"cmp"
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// WebsiteTag is a tag with its posts and its place in the navigation.
type WebsiteTag struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	// Posts are its published posts; Unpublished its posts taken down
	// ("its only post is unpublished").
	Posts       int64 `json:"posts"`
	Unpublished int64 `json:"unpublished"`
	// InNav is "Show in navigation": the tag is one of the header links.
	InNav bool `json:"inNav"`
	// URL and Feed are its page and its feed.
	URL  string `json:"url"`
	Feed string `json:"feed"`
	// List is the list it was made from when the website was turned on.
	List string `json:"list,omitempty"`
}

// WebsiteTagInput creates or changes a tag.
type WebsiteTagInput struct {
	Name        string `json:"name" minLength:"1" maxLength:"60"`
	Slug        string `json:"slug,omitempty" maxLength:"80"`
	Description string `json:"description,omitempty" maxLength:"300"`
	InNav       bool   `json:"inNav"`
}

// Tags lists the website's tags.
func (s *Service) Tags(ctx context.Context, ws *store.Workspace) ([]WebsiteTag, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	return s.tagViews(ctx, s.Q, ws, c)
}

func (s *Service) tagViews(ctx context.Context, q *store.Queries, ws *store.Workspace, c *config) ([]WebsiteTag, error) {
	rows, err := q.WebsiteTags(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	inNav := map[string]bool{}
	navAt := map[string]int{}
	for i, l := range c.Header {
		if l.Kind == NavTag {
			inNav[l.Ref] = true
			navAt[l.Ref] = i
		}
	}
	out := make([]WebsiteTag, 0, len(rows))
	for _, t := range rows {
		v := WebsiteTag{ID: t.ID.String(), Name: t.Name, Slug: t.Slug, Description: t.Description, Posts: t.Published, Unpublished: t.Unpublished,
			InNav: inNav[t.ID.String()], URL: s.sitePath(ctx, ws, "tag", t.Slug), Feed: s.sitePath(ctx, ws, "tag", t.Slug, "feed")}
		if t.ListID != nil {
			v.List = t.ListID.String()
		}
		out = append(out, v)
	}
	// The header's tags first, in the header's order; then the most used.
	slices.SortStableFunc(out, func(a, b WebsiteTag) int {
		ai, aNav := navAt[a.ID]
		bi, bNav := navAt[b.ID]
		switch {
		case aNav && bNav:
			return cmp.Compare(ai, bi)
		case aNav != bNav:
			if aNav {
				return -1
			}
			return 1
		case a.Posts != b.Posts:
			return cmp.Compare(b.Posts, a.Posts)
		}
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

// CreateTag makes a tag.
func (s *Service) CreateTag(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebsiteTagInput) (WebsiteTag, error) {
	id := newID()
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		name, slug, desc, err := s.tagFields(ctx, q, ws, id, in)
		if err != nil {
			return err
		}
		if err := q.WebsiteTagInsert(ctx, store.WebsiteTagInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Slug: slug, Description: desc,
			CreatedBy: userID(p), At: s.now()}); err != nil {
			return err
		}
		if err := s.setTagNav(ctx, q, ws, p, id, in.InNav); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "tag.created", TargetType: "website_tag", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "slug": slug, "in_nav": in.InNav}})
	})
	if err != nil {
		return WebsiteTag{}, err
	}
	return s.tag(ctx, ws, id)
}

// UpdateTag changes a tag.
func (s *Service) UpdateTag(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsiteTagInput) (WebsiteTag, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsiteTagGet(ctx, store.WebsiteTagGetParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx, "tag")
		}
		if err != nil {
			return err
		}
		name, slug, desc, err := s.tagFields(ctx, q, ws, id, in)
		if err != nil {
			return err
		}
		if err := q.WebsiteTagUpdate(ctx, store.WebsiteTagUpdateParams{WorkspaceID: ws.ID, ID: id, Name: name, Slug: slug, Description: desc, At: s.now()}); err != nil {
			return err
		}
		c, err := s.lock(ctx, q, ws)
		if err != nil {
			return err
		}
		wasNav := false
		for _, l := range c.Header {
			if l.Kind == NavTag && l.Ref == id.String() {
				wasNav = true
			}
		}
		if err := s.setTagNav(ctx, q, ws, p, id, in.InNav); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "tag.updated", TargetType: "website_tag", TargetID: id.String(), TargetName: name,
			Before: map[string]any{"name": cur.Name, "slug": cur.Slug, "description": cur.Description, "in_nav": wasNav},
			After:  map[string]any{"name": name, "slug": slug, "description": desc, "in_nav": in.InNav}})
	})
	if err != nil {
		return WebsiteTag{}, err
	}
	return s.tag(ctx, ws, id)
}

// DeleteTag deletes a tag; its posts lose it and the header its link.
func (s *Service) DeleteTag(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsiteTagGet(ctx, store.WebsiteTagGetParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx, "tag")
		}
		if err != nil {
			return err
		}
		if err := s.setTagNav(ctx, q, ws, p, id, false); err != nil {
			return err
		}
		// Tag rows of the home page go with it.
		c, err := s.lock(ctx, q, ws)
		if err != nil {
			return err
		}
		kept := c.Sections[:0]
		for _, sec := range c.Sections {
			if sec.Kind == SectionTag && sec.Tag == id.String() {
				continue
			}
			kept = append(kept, sec)
		}
		c.Sections = kept
		if err := s.save(ctx, q, ws, c, userID(p)); err != nil {
			return err
		}
		if err := q.WebsiteTagDelete(ctx, store.WebsiteTagDeleteParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "tag.deleted", TargetType: "website_tag", TargetID: id.String(), TargetName: cur.Name,
			Before: map[string]any{"name": cur.Name, "slug": cur.Slug}})
	})
}

func (s *Service) tag(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WebsiteTag, error) {
	tags, err := s.Tags(ctx, ws)
	if err != nil {
		return WebsiteTag{}, err
	}
	for _, t := range tags {
		if t.ID == id.String() {
			return t, nil
		}
	}
	return WebsiteTag{}, notFound(ctx, "tag")
}

// tagFields validates a tag's name, address and description.
func (s *Service) tagFields(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, in WebsiteTagInput) (string, string, string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 60 {
		return "", "", "", api.Invalid("body.name", tr(ctx, "tag.nameRequired", nil))
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = slugify(name)
	}
	if !slugRe.MatchString(slug) || len(slug) > 80 {
		return "", "", "", api.Invalid("body.slug", tr(ctx, "post.slugInvalid", nil))
	}
	taken, err := q.WebsiteTagSlugTaken(ctx, store.WebsiteTagSlugTakenParams{WorkspaceID: ws.ID, Slug: slug, ID: id})
	if err != nil {
		return "", "", "", err
	}
	if taken {
		return "", "", "", api.Err(http.StatusConflict, "slug_taken", tr(ctx, "tag.slugTaken", map[string]any{"path": "/tag/" + slug}))
	}
	return name, slug, strings.TrimSpace(in.Description), nil
}

// setTagNav adds a tag to the header links or takes it out. The header
// holds 6 links.
func (s *Service) setTagNav(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, id uuid.UUID, on bool) error {
	c, err := s.lock(ctx, q, ws)
	if err != nil {
		return err
	}
	has := -1
	for i, l := range c.Header {
		if l.Kind == NavTag && l.Ref == id.String() {
			has = i
		}
	}
	switch {
	case on && has < 0:
		if len(c.Header) >= MaxHeaderLinks {
			return api.Err(http.StatusConflict, "header_full", tr(ctx, "nav.headerFull", nil))
		}
		c.Header = append(c.Header, WebsiteNavLink{Kind: NavTag, Ref: id.String()})
	case !on && has >= 0:
		c.Header = append(c.Header[:has], c.Header[has+1:]...)
	default:
		return nil
	}
	return s.save(ctx, q, ws, c, userID(p))
}
