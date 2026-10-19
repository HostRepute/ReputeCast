package pages

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// What the public site needs of pages: the page at an address, as it was
// published.

// LivePage is a page as readers see it.
type LivePage struct {
	ID       uuid.UUID
	Slug     string
	Name     string
	Template string
	Version  int32
	HTML     string
	Settings PageSettings
	// PublishedAt is when this version went live.
	PublishedAt *time.Time
}

// Resolved is what an address on the site holds.
type Resolved struct {
	Page *LivePage
	// MovedTo is the address of a page that moved (a redirect).
	MovedTo string
	// RedirectTo is where an unpublished page sends readers.
	RedirectTo string
	// Gone: the page was taken off the site or deleted.
	Gone bool
}

// Resolve reads the page at an address.
func (s *Service) Resolve(ctx context.Context, ws *store.Workspace, slug string) (Resolved, error) {
	if !slugRe.MatchString(slug) {
		return Resolved{}, nil
	}
	pg, err := s.Q.PageByLiveSlug(ctx, store.PageByLiveSlugParams{WorkspaceID: ws.ID, Slug: &slug})
	switch {
	case err == nil:
		if pg.Status == StatusUnpublished || pg.LiveVersion == nil {
			if pg.RedirectTo != "" {
				return Resolved{RedirectTo: pg.RedirectTo}, nil
			}
			return Resolved{Gone: true}, nil
		}
		lp, err := s.live(ctx, pg)
		if err != nil {
			return Resolved{}, err
		}
		return Resolved{Page: lp}, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return Resolved{}, err
	}
	if id, err := s.Q.PageRedirectGet(ctx, store.PageRedirectGetParams{WorkspaceID: ws.ID, Slug: slug}); err == nil {
		moved, err := s.Q.PageGet(ctx, store.PageGetParams{WorkspaceID: ws.ID, ID: id})
		if err == nil && moved.LiveSlug != nil && *moved.LiveSlug != slug {
			return Resolved{MovedTo: *moved.LiveSlug}, nil
		}
	}
	if _, err := s.Q.PageEverLiveBySlug(ctx, store.PageEverLiveBySlugParams{WorkspaceID: ws.ID, Slug: &slug}); err == nil {
		return Resolved{Gone: true}, nil
	}
	return Resolved{}, nil
}

// live reads the version readers see.
func (s *Service) live(ctx context.Context, pg store.Page) (*LivePage, error) {
	v, err := s.Q.PageVersionGet(ctx, store.PageVersionGetParams{PageID: pg.ID, Version: *pg.LiveVersion})
	if err != nil {
		return nil, err
	}
	set := parseSettings(v.PageVersion.Settings)
	return &LivePage{ID: pg.ID, Slug: *pg.LiveSlug, Name: v.PageVersion.Name, Template: pg.Template, Version: v.PageVersion.Version, HTML: v.PageVersion.Html,
		PublishedAt: v.PageVersion.PublishedAt,
		Settings: PageSettings{SEOTitle: set.SEOTitle, SEODescription: set.SEODescription, SocialImage: set.SocialImage, Visibility: set.Visibility,
			Tiers: set.Tiers, HideHeader: set.HideHeader, ShowInNav: set.ShowInNav, NavLabel: set.NavLabel}}, nil
}

// Draft is a page's draft as the builder's preview draws it.
func (s *Service) Draft(ctx context.Context, ws *store.Workspace, id uuid.UUID) (*LivePage, error) {
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return nil, err
	}
	set := settingsOf(pg)
	return &LivePage{ID: pg.ID, Slug: pg.Slug, Name: pg.Name, Template: pg.Template, Version: pg.Version, HTML: pg.Html,
		Settings: PageSettings{SEOTitle: set.SEOTitle, SEODescription: set.SEODescription, SocialImage: set.SocialImage, Visibility: set.Visibility,
			Tiers: set.Tiers, HideHeader: set.HideHeader, ShowInNav: set.ShowInNav, NavLabel: set.NavLabel}}, nil
}

// Conversion records what a page brought in: a signup, a registration.
func (s *Service) Conversion(ctx context.Context, ws *store.Workspace, page uuid.UUID, kind string, contact *uuid.UUID) error {
	pid := page
	return s.Q.PageConversionInsert(ctx, store.PageConversionInsertParams{ID: newID(), WorkspaceID: ws.ID, PageID: &pid, Source: "page", Kind: kind,
		ContactID: contact, OccurredAt: s.now()})
}

// ShopOn says whether the shop (/products) is on.
func (s *Service) ShopOn(ctx context.Context, ws *store.Workspace) (bool, error) {
	set, err := s.Q.ShopSettingsGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		// The shop is on until someone turns it off.
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return set.ShopOn, nil
}

// ShopText is the shop's title and line ("" for the defaults).
func (s *Service) ShopText(ctx context.Context, ws *store.Workspace) (string, string) {
	set, err := s.Q.ShopSettingsGet(ctx, ws.ID)
	if err != nil {
		return "", ""
	}
	return set.Title, set.Intro
}
