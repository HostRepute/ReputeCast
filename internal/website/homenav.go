package website

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// WebsiteFeaturedPost is a featured post in Home & navigation.
type WebsiteFeaturedPost struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Tag         string     `json:"tag,omitempty"`
	Visibility  string     `json:"visibility"`
	Tiers       []string   `json:"tiers"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}

// WebsiteNavItem is a header link with what it opens.
type WebsiteNavItem struct {
	WebsiteNavLink
	// Name is the tag's or page's name; Detail what it is ("Tag", 143
	// posts; a page's address).
	Name  string `json:"name"`
	Posts int64  `json:"posts"`
	Path  string `json:"path"`
}

// WebsiteIdentity is the site's title, tagline, description and images.
type WebsiteIdentity struct {
	Title       string `json:"title" maxLength:"80"`
	Tagline     string `json:"tagline" maxLength:"160"`
	Description string `json:"description" maxLength:"300"`
	SocialImage string `json:"socialImage" maxLength:"1000"`
}

// WebsiteFeedURLs are the site's sitemap and feed addresses.
type WebsiteFeedURLs struct {
	Sitemap     string `json:"sitemap"`
	Feed        string `json:"feed"`
	TagFeed     string `json:"tagFeed"`
	ArchiveFeed string `json:"archiveFeed"`
}

// WebsitePageOption is a page header links can point at (the page builder's).
type WebsitePageOption struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// WebsiteHomeNav is Website › Home & navigation.
type WebsiteHomeNav struct {
	Layout   string                `json:"layout" enum:"featured,list,grid"`
	Featured []WebsiteFeaturedPost `json:"featured"`
	Sections []WebsiteHomeSection  `json:"sections"`
	About    WebsiteAbout          `json:"about"`
	Header   []WebsiteNavItem      `json:"header"`
	Footer   WebsiteFooter         `json:"footer"`
	// PrivacyURL and PoweredBy are the branding's (Forms & pages ›
	// Branding & domain), shown and changed here too.
	PrivacyURL string              `json:"privacyUrl"`
	PoweredBy  bool                `json:"poweredBy"`
	Identity   WebsiteIdentity     `json:"identity"`
	Index      bool                `json:"index"`
	URLs       WebsiteFeedURLs     `json:"urls"`
	Tags       []WebsiteTag        `json:"tags"`
	Pages      []WebsitePageOption `json:"pages"`
	// Logo is the branding's logo (the favicon), "" for the initial.
	Logo      string    `json:"logo"`
	Initial   string    `json:"initial"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
}

// WebsiteAbout is the home page's about section.
type WebsiteAbout struct {
	Title string `json:"title" maxLength:"80"`
	Text  string `json:"text" maxLength:"1200"`
	// Link is a path on the site ("/about") or an address.
	Link string `json:"link" maxLength:"500"`
}

// WebsiteHomeNavInput saves Website › Home & navigation.
type WebsiteHomeNavInput struct {
	Layout     string               `json:"layout" enum:"featured,list,grid"`
	Featured   []string             `json:"featured" maxItems:"3"`
	Sections   []WebsiteHomeSection `json:"sections" maxItems:"30"`
	About      WebsiteAbout         `json:"about"`
	Header     []WebsiteNavLink     `json:"header" maxItems:"12"`
	Footer     WebsiteFooter        `json:"footer"`
	PrivacyURL string               `json:"privacyUrl" maxLength:"500"`
	PoweredBy  bool                 `json:"poweredBy"`
	Identity   WebsiteIdentity      `json:"identity"`
	Index      bool                 `json:"index"`
}

// HomeNav returns Website › Home & navigation.
func (s *Service) HomeNav(ctx context.Context, ws *store.Workspace, p *authz.Principal) (WebsiteHomeNav, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteHomeNav{}, err
	}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return WebsiteHomeNav{}, err
	}
	b := set.Branding
	out := WebsiteHomeNav{Layout: c.row.HomeLayout, Sections: c.Sections, Footer: c.Footer, PrivacyURL: b.PrivacyURL, PoweredBy: b.PoweredBy,
		Identity: WebsiteIdentity{Title: siteTitle(c, set), Tagline: c.row.Tagline, Description: c.row.Description, SocialImage: c.row.SocialImage},
		Index:    c.row.IndexSite, About: WebsiteAbout{Title: c.row.AboutTitle, Text: c.row.AboutText, Link: c.row.AboutLink},
		Featured: []WebsiteFeaturedPost{}, Header: []WebsiteNavItem{}, Pages: []WebsitePageOption{}, Initial: b.Initial(), UpdatedAt: c.row.UpdatedAt}
	if by, err := s.Q.WebsiteSettingsUpdatedBy(ctx, ws.ID); err == nil {
		out.UpdatedBy = by
	}
	if b.Logo != nil {
		out.Logo = s.Forms.Links(ws, set).Logo(b.Logo.Hash)
	}
	out.URLs = WebsiteFeedURLs{Sitemap: hostOnly(s.sitePath(ctx, ws, "sitemap.xml")), Feed: hostOnly(s.sitePath(ctx, ws, "feed")),
		TagFeed: hostOnly(s.sitePath(ctx, ws, "tag")) + "/{slug}/feed", ArchiveFeed: hostOnly(s.Forms.Links(ws, set).Feed())}
	if out.Tags, err = s.tagViews(ctx, s.Q, ws, c); err != nil {
		return out, err
	}
	tiers, err := s.tiers(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	for _, id := range c.row.Featured {
		pr, err := s.getPost(ctx, s.Q, ws, id)
		if err != nil {
			continue
		}
		fp := WebsiteFeaturedPost{ID: id.String(), Title: pr.WebsitePost.Title, Visibility: pr.WebsitePost.Visibility, Tiers: []string{},
			Status: displayStatus(pr.WebsitePost, pr.CampaignStatus), PublishedAt: pr.WebsitePost.PublishedAt}
		if len(pr.Tags) > 0 {
			fp.Tag = pr.Tags[0].Name
		}
		for _, t := range tierRefs(pr.WebsitePost.Tiers, tiers) {
			fp.Tiers = append(fp.Tiers, t.Label)
		}
		out.Featured = append(out.Featured, fp)
	}
	var slugs []string
	for _, l := range c.Header {
		if l.Kind == NavPage {
			slugs = append(slugs, strings.Trim(l.Ref, "/"))
		}
	}
	names := resolvePageNames(ctx, s.Q, ws.ID, slugs)
	for _, l := range c.Header {
		item := WebsiteNavItem{WebsiteNavLink: l}
		switch l.Kind {
		case NavTag:
			for _, t := range out.Tags {
				if t.ID == l.Ref {
					item.Name, item.Posts, item.Path = t.Name, t.Posts, "/tag/"+t.Slug
				}
			}
			if item.Name == "" {
				continue
			}
		case NavPage:
			slug := strings.Trim(l.Ref, "/")
			item.Name, item.Path = names[slug], "/"+slug
			if item.Name == "" {
				item.Name = titleOfSlug(slug)
			}
		case NavHome:
			item.Path = "/"
		case NavArchive:
			item.Path = "/archive"
		case NavURL:
			item.Path = l.URL
		}
		if l.Label != "" {
			item.Name = l.Label
		}
		out.Header = append(out.Header, item)
	}
	for slug, title := range names {
		out.Pages = append(out.Pages, WebsitePageOption{Slug: slug, Title: title})
	}
	return out, nil
}

// SaveHomeNav saves Website › Home & navigation, with an audit entry per
// part changed: the home page, the navigation, the site's identity.
func (s *Service) SaveHomeNav(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebsiteHomeNavInput) (WebsiteHomeNav, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := s.lock(ctx, q, ws)
		if err != nil {
			return err
		}
		before := *c
		beforeSections := slices.Clone(c.Sections)
		beforeHeader := slices.Clone(c.Header)
		if err := s.applyHomeNav(ctx, q, ws, c, in); err != nil {
			return err
		}
		if err := s.save(ctx, q, ws, c, userID(p)); err != nil {
			return err
		}
		set, err := s.Forms.LoadSettings(ctx, ws)
		if err != nil {
			return err
		}
		if set.Branding.PoweredBy != in.PoweredBy || set.Branding.PrivacyURL != strings.TrimSpace(in.PrivacyURL) {
			if err := s.Forms.SetFooter(ctx, q, ws, in.PoweredBy, in.PrivacyURL); err != nil {
				return err
			}
		}
		log := func(action string, b, a map[string]any) error {
			if fmt.Sprint(b) == fmt.Sprint(a) {
				return nil
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "website", TargetID: ws.ID.String(), TargetName: ws.Name,
				Before: b, After: a})
		}
		if err := log("website.home_updated",
			map[string]any{"layout": before.row.HomeLayout, "featured": before.row.Featured, "sections": beforeSections,
				"about": []string{before.row.AboutTitle, before.row.AboutText, before.row.AboutLink}},
			map[string]any{"layout": c.row.HomeLayout, "featured": c.row.Featured, "sections": c.Sections,
				"about": []string{c.row.AboutTitle, c.row.AboutText, c.row.AboutLink}}); err != nil {
			return err
		}
		if err := log("website.navigation_updated",
			map[string]any{"header": beforeHeader, "footer": before.Footer, "powered_by": set.Branding.PoweredBy, "privacy_url": set.Branding.PrivacyURL},
			map[string]any{"header": c.Header, "footer": c.Footer, "powered_by": in.PoweredBy, "privacy_url": strings.TrimSpace(in.PrivacyURL)}); err != nil {
			return err
		}
		return log("website.identity_updated",
			map[string]any{"title": before.row.Title, "tagline": before.row.Tagline, "description": before.row.Description, "social_image": before.row.SocialImage, "index": before.row.IndexSite},
			map[string]any{"title": c.row.Title, "tagline": c.row.Tagline, "description": c.row.Description, "social_image": c.row.SocialImage, "index": c.row.IndexSite})
	})
	if err != nil {
		return WebsiteHomeNav{}, err
	}
	return s.HomeNav(ctx, ws, p)
}

// applyHomeNav checks Home & navigation settings and lays them over c.
func (s *Service) applyHomeNav(ctx context.Context, q *store.Queries, ws *store.Workspace, c *config, in WebsiteHomeNavInput) error {
	if len(in.Header) > MaxHeaderLinks {
		return api.Invalid("body.header", tr(ctx, "nav.headerFull", nil))
	}
	if strings.TrimSpace(in.Identity.Title) == "" {
		return api.Invalid("body.identity.title", tr(ctx, "identity.titleRequired", nil))
	}
	switch in.Layout {
	case LayoutFeatured, LayoutList, LayoutGrid:
		c.row.HomeLayout = in.Layout
	default:
		return api.Invalid("body.layout", tr(ctx, "home.layoutInvalid", nil))
	}
	featured := []uuid.UUID{}
	for _, raw := range in.Featured {
		id, err := uuid.Parse(raw)
		if err != nil {
			return api.Invalid("body.featured", tr(ctx, "post.notFound", nil))
		}
		if _, err := q.WebsitePostGet(ctx, store.WebsitePostGetParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return api.Invalid("body.featured", tr(ctx, "post.notFound", nil))
		}
		if !slices.Contains(featured, id) {
			featured = append(featured, id)
		}
	}
	if len(featured) > MaxFeatured {
		return api.Invalid("body.featured", tr(ctx, "post.featuredFull", nil))
	}
	c.row.Featured = featured
	sections := []WebsiteHomeSection{}
	for _, sec := range in.Sections {
		switch sec.Kind {
		case SectionSubscribe, SectionFeatured, SectionLatest, SectionAbout, SectionTiers:
		case SectionTag:
			id, err := uuid.Parse(sec.Tag)
			if err != nil {
				return api.Invalid("body.sections", tr(ctx, "tag.notFound", nil))
			}
			if _, err := q.WebsiteTagGet(ctx, store.WebsiteTagGetParams{WorkspaceID: ws.ID, ID: id}); err != nil {
				return api.Invalid("body.sections", tr(ctx, "tag.notFound", nil))
			}
		default:
			return api.Invalid("body.sections", tr(ctx, "home.sectionInvalid", nil))
		}
		if sec.Kind == SectionLatest && (sec.Count < 1 || sec.Count > 24) {
			sec.Count = 6
		}
		if sec.Kind == SectionTag && (sec.Count < 1 || sec.Count > 12) {
			sec.Count = 3
		}
		sec.Note, sec.Text = strings.TrimSpace(sec.Note), strings.TrimSpace(sec.Text)
		sections = append(sections, sec)
	}
	c.Sections = sections
	header := []WebsiteNavLink{}
	for _, l := range in.Header {
		l.Label = strings.TrimSpace(l.Label)
		switch l.Kind {
		case NavHome, NavArchive:
		case NavTag:
			id, err := uuid.Parse(l.Ref)
			if err != nil {
				return api.Invalid("body.header", tr(ctx, "tag.notFound", nil))
			}
			if _, err := q.WebsiteTagGet(ctx, store.WebsiteTagGetParams{WorkspaceID: ws.ID, ID: id}); err != nil {
				return api.Invalid("body.header", tr(ctx, "tag.notFound", nil))
			}
		case NavPage:
			l.Ref = strings.Trim(strings.TrimSpace(l.Ref), "/")
			if !slugRe.MatchString(l.Ref) {
				return api.Invalid("body.header", tr(ctx, "nav.pageInvalid", nil))
			}
		case NavURL:
			u, err := url.Parse(strings.TrimSpace(l.URL))
			if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || l.Label == "" {
				return api.Invalid("body.header", tr(ctx, "nav.urlInvalid", nil))
			}
		default:
			return api.Invalid("body.header", tr(ctx, "nav.kindInvalid", nil))
		}
		if utf8.RuneCountInString(l.Label) > 40 {
			return api.Invalid("body.header", tr(ctx, "nav.labelLong", nil))
		}
		header = append(header, l)
	}
	c.Header, c.Footer = header, in.Footer
	c.row.AboutTitle, c.row.AboutText, c.row.AboutLink = strings.TrimSpace(in.About.Title), strings.TrimSpace(in.About.Text), strings.TrimSpace(in.About.Link)
	c.row.Title, c.row.Tagline = strings.TrimSpace(in.Identity.Title), strings.TrimSpace(in.Identity.Tagline)
	c.row.Description, c.row.SocialImage = strings.TrimSpace(in.Identity.Description), strings.TrimSpace(in.Identity.SocialImage)
	c.row.IndexSite = in.Index
	return nil
}

// homeDraftKey carries unsaved Home & navigation settings (WithHomeNavDraft).
type homeDraftKey struct{}

type homeDraft struct {
	ws uuid.UUID
	c  *config
}

// WithHomeNavDraft returns ctx with unsaved Home & navigation settings laid
// over the saved ones: the site's pages built in it (the tab's live
// preview) read them. Nothing is saved; settings that wouldn't save are
// refused the same way.
func (s *Service) WithHomeNavDraft(ctx context.Context, ws *store.Workspace, in WebsiteHomeNavInput) (context.Context, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return ctx, err
	}
	if err := s.applyHomeNav(ctx, s.Q, ws, c, in); err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, homeDraftKey{}, &homeDraft{ws: ws.ID, c: c}), nil
}

// draftConfig is the unsaved settings ctx carries for ws, if any.
func draftConfig(ctx context.Context, ws uuid.UUID) *config {
	if d, ok := ctx.Value(homeDraftKey{}).(*homeDraft); ok && d.ws == ws {
		cp := *d.c
		return &cp
	}
	return nil
}
