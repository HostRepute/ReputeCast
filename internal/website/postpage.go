package website

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// PostAddress is what an address under /posts/ leads to.
type PostAddress struct {
	// Post is the published post at the address.
	Post *store.WebsitePost
	// RedirectTo is a path on the site: a renamed post's new address, an
	// old archive address, or where an unpublished post now leads.
	RedirectTo string
	// Gone: the post was taken off the site or deleted (410).
	Gone bool
}

// ResolvePost finds the post at /posts/{slug} while the website is on (a
// zero PostAddress: nothing there, 404).
func (s *Service) ResolvePost(ctx context.Context, ws *store.Workspace, slug string) (PostAddress, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil || !c.On() || !slugRe.MatchString(slug) {
		return PostAddress{}, err
	}
	return s.resolve(ctx, ws, "posts/"+slug, slug)
}

// ResolveIssue is where an archive issue's old address leads once the
// website is on: its post (a zero PostAddress: the archive shows it).
func (s *Service) ResolveIssue(ctx context.Context, ws *store.Workspace, slug string) (PostAddress, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil || !c.On() {
		return PostAddress{}, err
	}
	p, err := s.Q.WebsiteRedirectGet(ctx, store.WebsiteRedirectGetParams{WorkspaceID: ws.ID, Path: "archive/" + slug})
	if errors.Is(err, pgx.ErrNoRows) {
		return PostAddress{}, nil
	}
	if err != nil {
		return PostAddress{}, err
	}
	return addressOf(p, true), nil
}

func (s *Service) resolve(ctx context.Context, ws *store.Workspace, path, slug string) (PostAddress, error) {
	p, err := s.Q.WebsitePostBySlug(ctx, store.WebsitePostBySlugParams{WorkspaceID: ws.ID, Slug: slug})
	switch {
	case err == nil:
		return addressOf(p, false), nil
	case !errors.Is(err, pgx.ErrNoRows):
		return PostAddress{}, err
	}
	// An earlier address of a post.
	if p, err := s.Q.WebsiteRedirectGet(ctx, store.WebsiteRedirectGetParams{WorkspaceID: ws.ID, Path: path}); err == nil {
		return addressOf(p, true), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return PostAddress{}, err
	}
	// A deleted post that was on the site.
	var gone bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM website_posts WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NOT NULL
		AND first_published_at IS NOT NULL)`, ws.ID, slug).Scan(&gone); err != nil {
		return PostAddress{}, err
	}
	return PostAddress{Gone: gone}, nil
}

// addressOf is where a post's address leads: the post, its current
// address (moved), or 410 and its redirect once unpublished.
func addressOf(p store.WebsitePost, moved bool) PostAddress {
	switch p.Status {
	case PostPublished:
		if moved {
			return PostAddress{RedirectTo: "/posts/" + p.Slug}
		}
		return PostAddress{Post: &p}
	case PostUnpublished:
		if p.RedirectTo != "" {
			return PostAddress{RedirectTo: p.RedirectTo}
		}
		return PostAddress{Gone: true}
	}
	if p.FirstPublishedAt != nil {
		return PostAddress{Gone: true}
	}
	return PostAddress{}
}

// SignupOffer is a newsletter the paywall and the sign-up form offer.
type SignupOffer struct {
	ID      uuid.UUID
	Name    string
	Consent string
	Checked bool
}

// PostView is a published post as one reader gets it. Text the reader may
// not read isn't in it: the preview stops at the marker, paid blocks they
// don't hold are a card.
type PostView struct {
	Post     PublicPost
	Decision gate.Decision
	// Body is what the reader gets: the whole post, or the free preview
	// ("" without a marker: the title, the cover and the excerpt).
	Body string
	// Readers are the tiers that read the post, lowest first (the paywall
	// offers them); Rules the site's access rules.
	Readers []gate.Tier
	Rules   gate.Rules
	Paywall WebsitePaywall
	// SignupOn: readers can sign up from the paywall; Offers are its lists.
	SignupOn bool
	Offers   []SignupOffer
	// Month is the meter's month; Next the first day of the next one.
	Month, Next time.Time
	// More are other posts of the post's first tag.
	More      []PublicPost
	MoreTag   *TagRef
	MoreTotal int64
	SEO       WebsitePostSEO
	Minutes   int
	// BandText is the subscribe band's line under the post (Home &
	// navigation's subscribe section).
	BandText string
	// SessionDays is how long members stay signed in.
	SessionDays int
}

// ReadOptions are how a post is read.
type ReadOptions struct {
	Lang i18n.Lang
	// Preview reads it for the app: nothing is recorded, and Meter, when
	// set, stands in for the reader's month.
	Preview bool
	Meter   *gate.MeterState
}

// ReadPost decides a published post for a reader and records the meter's
// read when it counts (never for previews).
func (s *Service) ReadPost(ctx context.Context, ws *store.Workspace, p store.WebsitePost, reader gate.Reader, o ReadOptions) (PostView, error) {
	preview, l := o.Preview, o.Lang
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return PostView{}, err
	}
	rules, tiers, err := s.rules(ctx, s.Q, ws, c)
	if err != nil {
		return PostView{}, err
	}
	posts, _, err := s.published(ctx, s.Q, ws, PublicQuery{IDs: []uuid.UUID{p.ID}, Limit: 1})
	if err != nil {
		return PostView{}, err
	}
	var pp PublicPost
	if len(posts) == 1 {
		pp = posts[0]
	} else {
		// A preview of a post that isn't published yet.
		pp = PublicPost{ID: p.ID, Kind: p.Kind, CampaignID: p.CampaignID, Slug: p.Slug, Title: p.Title, Excerpt: p.Excerpt, CoverURL: p.CoverUrl,
			CoverAlt: p.CoverAlt, PublishedAt: s.now(), UpdatedAt: p.UpdatedAt, Visibility: gate.Visibility{Level: p.Visibility, Tiers: p.Tiers,
				NeverFree: p.NeverFree}, Tags: []TagRef{}, Words: int(p.Words), Byline: p.BylineName}
		one := []PublicPost{pp}
		if err := s.withTags(ctx, s.Q, one); err != nil {
			return PostView{}, err
		}
		pp = one[0]
	}
	pp.Label = labelOf(pp.Visibility, tiers, WordsIn(l))
	loc := location(ws)
	month := monthOf(s.now(), loc)
	out := PostView{Post: pp, Rules: rules, Readers: gate.Readers(pp.Visibility, tiers), Paywall: c.paywallIn(l), SignupOn: c.row.SignupOn && c.row.SigninOn,
		Month: month, Next: month.AddDate(0, 1, 0), Minutes: render.ReadingMinutes(pp.Words), SessionDays: int(c.row.SessionDays),
		SEO: WebsitePostSEO{Title: p.SeoTitle, Description: p.SeoDescription, SocialImage: p.SocialImage, CanonicalURL: p.CanonicalUrl, HideFromSearch: p.HideFromSearch}}
	for _, sec := range c.Sections {
		if sec.Kind == SectionSubscribe {
			out.BandText = sec.Text
		}
	}
	if pp.Visibility.Level == gate.Paid && len(c.Paywall.Tiers) > 0 {
		// The paywall card shows the tiers Access picks, among the readers.
		picked := []gate.Tier{}
		for _, t := range out.Readers {
			for _, id := range c.Paywall.Tiers {
				if id == t.ID.String() {
					picked = append(picked, t)
				}
			}
		}
		if len(picked) > 0 {
			out.Readers = picked
		}
	}
	// The meter, then the decision.
	var state gate.MeterState
	rule := gate.MeterRule(pp.Visibility, reader, rules)
	switch {
	case o.Meter != nil:
		state = *o.Meter
	case rule != "" && reader.Key() != "":
		st, err := s.Q.WebsiteMeterState(ctx, store.WebsiteMeterStateParams{WorkspaceID: ws.ID, Month: month, Reader: reader.Key(), Rule: rule, PostID: p.ID})
		if err != nil {
			return out, err
		}
		state = gate.MeterState{Used: int(st.Used), Read: st.Read}
	}
	d := gate.Decide(pp.Visibility, reader, rules, state)
	if preview {
		d.Count = false
	}
	if d.Count {
		if err := s.Q.WebsiteMeterRead(ctx, store.WebsiteMeterReadParams{WorkspaceID: ws.ID, Month: month, Reader: reader.Key(), PostID: p.ID, Rule: rule,
			At: s.now()}); err != nil {
			return out, err
		}
	}
	out.Decision = d
	var paid []render.Paid
	_ = json.Unmarshal(p.PaidBlocks, &paid)
	body := p.PreviewHtml
	if d.Readable() {
		body = p.BodyHtml
	} else if !p.HasMarker {
		body = ""
	}
	out.Body = s.paidBlocks(ctx, ws, body, paid, reader, rules, tiers, l)
	if !d.Readable() {
		offers, err := s.offers(ctx, ws, c, l)
		if err != nil {
			return out, err
		}
		out.Offers = offers
	}
	// More from the post's first tag.
	if len(pp.Tags) > 0 {
		t := pp.Tags[0]
		more, total, err := s.published(ctx, s.Q, ws, PublicQuery{Tag: &t.ID, Exclude: []uuid.UUID{p.ID}, Limit: 3})
		if err != nil {
			return out, err
		}
		for i := range more {
			more[i].Label = labelOf(more[i].Visibility, tiers, WordsIn(l))
		}
		out.More, out.MoreTag, out.MoreTotal = more, &t, total+1
	}
	return out, nil
}

// paidBlocks puts each paid block of a body in its place: the block for a
// reader holding its tier, a short card for everyone else.
func (s *Service) paidBlocks(ctx context.Context, ws *store.Workspace, body string, paid []render.Paid, reader gate.Reader, rules gate.Rules,
	tiers []gate.Tier, l i18n.Lang) string {
	if !strings.Contains(body, "<!--rc:paid ") {
		return body
	}
	byID := map[string]render.Paid{}
	for _, p := range paid {
		byID[p.ID] = p
	}
	w := WordsIn(l)
	upgrade := s.sitePath(ctx, ws, "upgrade")
	return render.PaidPlaceholderRe.ReplaceAllStringFunc(body, func(m string) string {
		id := render.PaidPlaceholderRe.FindStringSubmatch(m)[1]
		b, ok := byID[id]
		if !ok {
			return ""
		}
		v := gate.Visibility{Level: gate.Paid}
		for _, t := range b.Tiers {
			if u, err := uuid.Parse(t); err == nil {
				v.Tiers = append(v.Tiers, u)
			}
		}
		if reader.Member != nil && gate.Qualifies(v, reader.Member.Tiers, rules.Tiers) != nil {
			return b.HTML
		}
		text := w.Text("gate.paidBlockAny", nil)
		if readers := gate.Readers(v, tiers); len(readers) > 0 && len(v.Tiers) > 0 {
			text = w.Text("gate.paidBlock", i18n.Args{"tier": readers[0].Label})
		}
		return `<aside class="st-paidblock"><p>` + html.EscapeString(text) + `</p><a href="` + html.EscapeString(upgrade) + `" class="st-lnk">` +
			html.EscapeString(w.Text("gate.paidBlockUpgrade", nil)) + `</a></aside>`
	})
}

// offers are the newsletters the paywall and the sign-up form offer, with
// ReputeCast's consent text in the page's language where the workspace
// wrote none.
func (s *Service) offers(ctx context.Context, ws *store.Workspace, c *config, l i18n.Lang) ([]SignupOffer, error) {
	out := []SignupOffer{}
	if !c.row.SignupOn {
		return out, nil
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	names := map[uuid.UUID]string{}
	for _, l := range lists {
		names[l.ID] = l.Name
	}
	for _, sl := range c.SignupLists {
		id, err := uuid.Parse(sl.ListID)
		if err != nil {
			continue
		}
		name, ok := names[id]
		if !ok {
			continue
		}
		consent := strings.TrimSpace(sl.Consent)
		if consent == "" {
			consent = defaultConsent(l, name)
		}
		out = append(out, SignupOffer{ID: id, Name: name, Consent: consent, Checked: sl.Checked})
	}
	return out, nil
}

// PublicPosts lists the site's published posts (the archive, a tag page,
// feeds), newest first, with the total. Their labels are in l.
func (s *Service) PublicPosts(ctx context.Context, ws *store.Workspace, f PublicQuery, l i18n.Lang) ([]PublicPost, int64, error) {
	posts, total, err := s.published(ctx, s.Q, ws, f)
	if err != nil {
		return nil, 0, err
	}
	tiers, err := s.tiers(ctx, s.Q, ws)
	if err != nil {
		return nil, 0, err
	}
	for i := range posts {
		posts[i].Label = labelOf(posts[i].Visibility, tiers, WordsIn(l))
	}
	return posts, total, nil
}

// PublicTag is a tag's page: its name, description and number of posts.
type PublicTag struct {
	TagRef
	Description string
	Posts       int64
}

// TagBySlug finds a tag of the site while it is on (nil: none).
func (s *Service) TagBySlug(ctx context.Context, ws *store.Workspace, slug string) (*PublicTag, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil || !c.On() {
		return nil, err
	}
	t, err := s.Q.WebsiteTagBySlug(ctx, store.WebsiteTagBySlugParams{WorkspaceID: ws.ID, Slug: slug})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_, total, err := s.published(ctx, s.Q, ws, PublicQuery{Tag: &t.ID, Limit: 1})
	if err != nil {
		return nil, err
	}
	return &PublicTag{TagRef: TagRef{ID: t.ID, Name: t.Name, Slug: t.Slug}, Description: t.Description, Posts: total}, nil
}

// SiteTags are the tags with published posts, most posts first (the
// archive's filter).
func (s *Service) SiteTags(ctx context.Context, ws *store.Workspace) ([]PublicTag, error) {
	rows, err := s.Pool.Query(ctx, `SELECT t.id, t.name, t.slug, t.description, count(p.id)
		FROM website_tags t
		JOIN website_post_tags pt ON pt.tag_id = t.id
		JOIN website_posts p ON p.id = pt.post_id AND p.status = 'published' AND p.deleted_at IS NULL
		WHERE t.workspace_id = $1
		GROUP BY t.id ORDER BY count(p.id) DESC, t.name`, ws.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PublicTag{}
	for rows.Next() {
		var t PublicTag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Description, &t.Posts); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// FeedItem is a post in a feed: public posts whole, gated posts only as
// far as anyone may read them.
type FeedItem struct {
	Post PublicPost
	// HTML is the post's body for a public post, else its free preview ("").
	HTML string
	// Gated: a members-only or paid post.
	Gated bool
}

// Feed lists the newest posts for the site's feed (or a tag's).
func (s *Service) Feed(ctx context.Context, ws *store.Workspace, tag *uuid.UUID, limit int) ([]FeedItem, error) {
	posts, _, err := s.published(ctx, s.Q, ws, PublicQuery{Tag: tag, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]FeedItem, 0, len(posts))
	for _, p := range posts {
		it := FeedItem{Post: p, Gated: p.Visibility.Level != gate.Public}
		var body, preview string
		var paid json.RawMessage
		if err := s.Pool.QueryRow(ctx, `SELECT body_html, preview_html, paid_blocks FROM website_posts WHERE id = $1`, p.ID).Scan(&body, &preview, &paid); err != nil {
			return nil, err
		}
		if it.Gated {
			body = preview
		}
		// Paid blocks are left out of feeds, as for a visitor.
		it.HTML = render.PaidPlaceholderRe.ReplaceAllString(body, "")
		out = append(out, it)
	}
	return out, nil
}

// SitemapEntry is an address of the site for search engines.
type SitemapEntry struct {
	Path    []string
	LastMod time.Time
}

// Sitemap lists the site's posts (but those hidden from search) and tags.
func (s *Service) Sitemap(ctx context.Context, ws *store.Workspace) ([]SitemapEntry, error) {
	out := []SitemapEntry{}
	rows, err := s.Pool.Query(ctx, `SELECT slug, updated_at FROM website_posts
		WHERE workspace_id = $1 AND status = 'published' AND deleted_at IS NULL AND NOT hide_from_search
		ORDER BY published_at DESC LIMIT 50000`, ws.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var slug string
		var at time.Time
		if err := rows.Scan(&slug, &at); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, SitemapEntry{Path: []string{"posts", slug}, LastMod: at})
	}
	rows.Close()
	tags, err := s.SiteTags(ctx, ws)
	if err != nil {
		return nil, err
	}
	for _, t := range tags {
		out = append(out, SitemapEntry{Path: []string{"tag", t.Slug}})
	}
	return out, nil
}

// Preview readers ("Preview on web as …").
const (
	AsVisitor     = "visitor"
	AsFree        = "free"
	AsFreeUsedUp  = "free_used"
	AsPaid        = "paid"
	AsVisitorUsed = "visitor_used"
)

// PreviewPost reads a post as one kind of reader for the app, from its
// content as it is now (published or not). tier is a paid reader's tier
// ("" for the highest).
func (s *Service) PreviewPost(ctx context.Context, ws *store.Workspace, id uuid.UUID, as, tier string, l i18n.Lang) (store.WebsitePost, PostView, gate.Reader, error) {
	pr, err := s.getPost(ctx, s.Q, ws, id)
	if err != nil {
		return store.WebsitePost{}, PostView{}, gate.Reader{}, err
	}
	p := pr.WebsitePost
	src, err := s.sourceMJML(ctx, ws, p)
	if err != nil {
		return p, PostView{}, gate.Reader{}, err
	}
	res, err := render.Render(src, s.renderOptions(ctx, ws, p))
	if err != nil {
		res = render.Result{}
	}
	p.BodyHtml, p.PreviewHtml, p.HasMarker, p.Words = res.Body, res.Preview, res.HasMarker, int32(res.Words)
	paid, _ := json.Marshal(res.Paid)
	p.PaidBlocks = paid
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return p, PostView{}, gate.Reader{}, err
	}
	_, tiers, err := s.rules(ctx, s.Q, ws, c)
	if err != nil {
		return p, PostView{}, gate.Reader{}, err
	}
	sample := &gate.Member{ContactID: uuid.Nil, Email: "sam@example.com", Name: "Sam Rivera", Tiers: []gate.Tier{}}
	var reader gate.Reader
	meter := &gate.MeterState{}
	switch as {
	case AsFree:
		reader.Member = sample
	case AsFreeUsedUp:
		reader.Member = sample
		meter.Used = int(c.row.MeterMembers)
	case AsPaid:
		reader.Member = sample
		for i := len(tiers) - 1; i >= 0; i-- {
			if tier == "" || tiers[i].ID.String() == tier {
				reader.Member.Tiers = []gate.Tier{tiers[i]}
				break
			}
		}
	case AsVisitorUsed:
		reader.Browser = "preview"
		meter.Used = int(c.row.MeterVisitors)
	default:
		reader.Browser = "preview"
	}
	v, err := s.ReadPost(ctx, ws, p, reader, ReadOptions{Lang: l, Preview: true, Meter: meter})
	return p, v, reader, err
}
