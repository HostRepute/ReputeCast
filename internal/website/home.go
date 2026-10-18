package website

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// Home is the home page's data, section by section in their order.
type Home struct {
	Layout   string
	Sections []HomeSection
	// Total is every published post ("All 190 posts in the archive").
	Total int64
	// Tiers are the paid tiers (for labels).
	Tiers []gate.Tier
}

// HomeSection is one shown section of the home page.
type HomeSection struct {
	Kind string
	// Subscribe: the hero's note under the form.
	Note string
	// Featured, latest and tag rows: their posts.
	Posts []PublicPost
	// A tag row: the tag and its posts' count.
	Tag      *TagRef
	TagCount int64
	// About: its title, text and link (a path on the site or an address).
	AboutTitle, AboutText, AboutLink string
	// Tiers: the upgrade page's cards (nil without paid tiers or payments).
	Upgrade *earn.UpgradeView
}

// Home builds the home page. period is the tiers band's Monthly · Yearly pick.
func (s *Service) Home(ctx context.Context, ws *store.Workspace, l i18n.Lang, period string) (Home, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return Home{}, err
	}
	return s.home(ctx, s.Q, ws, c, l, period)
}

func (s *Service) home(ctx context.Context, q *store.Queries, ws *store.Workspace, c *config, l i18n.Lang, period string) (Home, error) {
	h := Home{Layout: c.row.HomeLayout, Sections: []HomeSection{}}
	_, tiers, err := s.rules(ctx, q, ws, c)
	if err != nil {
		return h, err
	}
	h.Tiers = tiers
	_, total, err := s.published(ctx, q, ws, PublicQuery{Limit: 1})
	if err != nil {
		return h, err
	}
	h.Total = total
	featured, err := s.featured(ctx, q, ws, c)
	if err != nil {
		return h, err
	}
	shownFeatured := false
	for _, sec := range c.Sections {
		if !sec.On {
			continue
		}
		hs := HomeSection{Kind: sec.Kind, Note: sec.Note}
		switch sec.Kind {
		case SectionSubscribe:
		case SectionFeatured:
			if len(featured) == 0 {
				continue
			}
			hs.Posts, shownFeatured = featured, true
		case SectionLatest:
			n := sec.Count
			if n <= 0 {
				n = 6
			}
			var exclude []uuid.UUID
			if shownFeatured || slices.ContainsFunc(c.Sections, func(x WebsiteHomeSection) bool { return x.Kind == SectionFeatured && x.On }) {
				for _, p := range featured {
					exclude = append(exclude, p.ID)
				}
			}
			posts, _, err := s.published(ctx, q, ws, PublicQuery{Limit: n, Exclude: exclude})
			if err != nil {
				return h, err
			}
			if len(posts) == 0 {
				continue
			}
			hs.Posts = posts
		case SectionTag:
			id, err := uuid.Parse(sec.Tag)
			if err != nil {
				continue
			}
			t, err := q.WebsiteTagGet(ctx, store.WebsiteTagGetParams{WorkspaceID: ws.ID, ID: id})
			if err != nil {
				continue
			}
			n := sec.Count
			if n <= 0 {
				n = 3
			}
			posts, count, err := s.published(ctx, q, ws, PublicQuery{Tag: &id, Limit: n})
			if err != nil {
				return h, err
			}
			if len(posts) == 0 {
				continue
			}
			hs.Posts, hs.Tag, hs.TagCount = posts, &TagRef{ID: t.ID, Name: t.Name, Slug: t.Slug}, count
		case SectionAbout:
			if c.row.AboutText == "" {
				continue
			}
			hs.AboutTitle, hs.AboutText, hs.AboutLink = c.row.AboutTitle, c.row.AboutText, c.row.AboutLink
		case SectionTiers:
			e := s.earn()
			if e == nil || len(tiers) == 0 {
				continue
			}
			v, err := e.Upgrade(ctx, ws, earn.UpgradeRequest{Lang: l, Period: period})
			if err != nil {
				s.log().WarnContext(ctx, "home tiers band", "err", err)
				continue
			}
			if v.Unavailable || len(v.Cards) == 0 {
				continue
			}
			hs.Upgrade = &v
		default:
			continue
		}
		h.Sections = append(h.Sections, hs)
	}
	return h, nil
}

// featured are the home's featured posts that are published, in order.
func (s *Service) featured(ctx context.Context, q *store.Queries, ws *store.Workspace, c *config) ([]PublicPost, error) {
	if len(c.row.Featured) == 0 {
		return []PublicPost{}, nil
	}
	posts, _, err := s.published(ctx, q, ws, PublicQuery{IDs: c.row.Featured, Limit: MaxFeatured})
	return posts, err
}

// Label is a post's visibility label in a language.
func (s *Service) Label(v gate.Visibility, tiers []gate.Tier, l i18n.Lang) PostLabel {
	return labelOf(v, tiers, WordsIn(l))
}
