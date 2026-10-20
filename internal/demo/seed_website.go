package demo

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
	"github.com/hostrepute/reputecast/internal/website/render"
)

//go:embed assets/website/*.png
var websiteAssets embed.FS

func init() { seedWebsite = seedWebsiteImpl }

// The publication website of the expansion's sample (publish-sell brief
// §4): news.acme.io went live on 29 Sep; every archived campaign is a post
// at its archive address, the archived lists are tags, plus Engineering,
// Deliverability, Insider, Announcements, Ask Acme and Beta Program.
// Web-only posts: Sofia's runbook (Paid, a free preview of three
// paragraphs, featured), Marcus's "Ask Acme" (Members), Linh's AMA
// recording (Founding member, never free, scheduled for 16 Oct) and Daniel
// Park's "Q4 roadmap deep dive" (Members, a draft waiting for approval),
// with Sofia's rate limits post (Paid) for the meter's second paid post.
// Weekly Digest #142 goes on the web with its send; "Feature flags
// preview" was unpublished on 10 Oct. Views, signups and upgrades over 30
// days; 2,804 members signed in since the site went live, Sam Rivera among
// them with his free paid post for October read.
type siteSeed struct {
	s      *Seeder
	ws     store.Workspace
	rng    *rand.Rand
	covers map[string]string
	tags   map[string]uuid.UUID
	posts  map[string]uuid.UUID
	tiers  map[string]uuid.UUID
	lists  map[string]uuid.UUID
	live   time.Time
	// published are the published posts with their days, for the views.
	published []sitePost
}

type sitePost struct {
	id     uuid.UUID
	at     time.Time
	weight float64
}

func seedWebsiteImpl(ctx context.Context, s *Seeder) error {
	w := &siteSeed{s: s, ws: s.WS(), rng: rand.New(rand.NewPCG(18, 7)), //nolint:gosec // G404: the demo's sample numbers, the same every time
		covers: map[string]string{}, tags: map[string]uuid.UUID{}, posts: map[string]uuid.UUID{},
		tiers: map[string]uuid.UUID{}, lists: map[string]uuid.UUID{}, live: s.designDay(16, 9, 30)}
	for _, step := range []func(context.Context) error{w.covers_, w.settings, w.tagsAndIssues, w.campaignPosts, w.webPosts, w.featured, w.views, w.members, w.approval, w.audit} {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

// covers_ stores the posts' covers in the image library.
func (w *siteSeed) covers_(ctx context.Context) error {
	s := w.s
	entries, err := websiteAssets.ReadDir("assets/website")
	if err != nil {
		return err
	}
	for i, e := range entries {
		data, err := websiteAssets.ReadFile("assets/website/" + e.Name())
		if err != nil {
			return err
		}
		ctype, wd, ht, alpha, err := media.Inspect(data)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		sum := sha256.Sum256(data)
		var id uuid.UUID
		if err := s.Tx.QueryRow(ctx, `INSERT INTO media_assets (id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, sha256, data, created_by, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (workspace_id, sha256) DO UPDATE SET file_name = EXCLUDED.file_name RETURNING id`,
			ID(), w.ws.ID, e.Name(), ctype, len(data), wd, ht, alpha, sum[:], data, s.Users["sofia@acme.io"].ID, w.live.Add(-time.Duration(len(entries)-i)*time.Hour)).Scan(&id); err != nil {
			return err
		}
		w.covers[strings.TrimSuffix(e.Name(), ".png")] = media.URL(id, e.Name())
	}
	return nil
}

// settings turns the website on with the sample's home, navigation,
// members, meter and paywall.
func (w *siteSeed) settings(ctx context.Context) error {
	s := w.s
	linh := s.Users["linh@acme.io"]
	for _, name := range []string{"Weekly Digest", "Product Updates", "Beta Program"} {
		w.lists[name] = s.Lists[name]
	}
	rows, err := s.Tx.Query(ctx, `SELECT id, name FROM earn_tiers WHERE workspace_id = $1 AND deleted_at IS NULL`, w.ws.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		w.tiers[name] = id
	}
	rows.Close()
	progress, _ := json.Marshal(map[string]any{"startedAt": w.live, "finishedAt": w.live.Add(3 * time.Minute), "pages": "done",
		"tags":  map[string]any{"done": 2, "total": 2, "names": []string{"Weekly Digest", "Product Updates"}},
		"posts": map[string]any{"done": 44, "total": 44}, "redirects": map[string]any{"done": 44, "total": 44}})
	signup, _ := json.Marshal([]website.WebsiteSignupList{
		{ListID: w.lists["Weekly Digest"].String(), Checked: true, Consent: "Also send me the Weekly Digest every Tuesday. Unsubscribe any time."},
		{ListID: w.lists["Product Updates"].String(), Checked: false, Consent: "Send me Product Updates when something ships."}})
	paywall := website.DefaultPaywall()
	paywall.Text = "Deep dives, runbooks and the monthly AMA, from the people who build Acme."
	for _, n := range []string{"Weekly Digest Insider", "Founding member"} {
		if id, ok := w.tiers[n]; ok {
			paywall.Tiers = append(paywall.Tiers, id.String())
		}
	}
	pw, _ := json.Marshal(paywall)
	footer, _ := json.Marshal(website.WebsiteFooter{Archive: true, RSS: true, Preferences: true})
	_, err = s.Tx.Exec(ctx, `INSERT INTO website_settings (workspace_id, status, turned_on_at, turned_on_by, progress, title, tagline, description, index_site,
		home_layout, sections, about_title, about_text, about_link, header_links, footer, signin_on, link_minutes, session_days, signup_on, signup_lists,
		default_visibility, campaign_visibility, meter_visitors_on, meter_visitors, meter_members_on, meter_members, paywall, updated_at, updated_by)
		VALUES ($1,'on',$2,$3,$4,'The Acme newsletter','What shipped, what''s next and the occasional deep dive.',
		'News from the team that builds Acme: what shipped, what''s next, and deep dives for paid members.', true,
		'featured', '[]', 'About the newsletter',
		'Acme builds feature flags, audit logs and deploy tooling for software teams. The newsletter is written by the people who build it: Sofia Rossi edits, Marcus Chen answers the deliverability questions, and Linh Nguyen hosts the monthly AMA.',
		'/about', '[]', $5, true, 30, 90, true, $6, 'public', 'audience', true, 2, true, 1, $7, $8, $3)
		ON CONFLICT (workspace_id) DO NOTHING`, w.ws.ID, w.live, linh.ID, progress, footer, signup, pw, w.live.Add(40*time.Minute))
	return err
}

// tagsAndIssues makes the tags, and a post of every archived campaign at
// its archive address, live at its send.
func (w *siteSeed) tagsAndIssues(ctx context.Context) error {
	s := w.s
	linh := s.Users["linh@acme.io"]
	for _, t := range []struct {
		name, slug, desc string
		list             string
		days             int
	}{
		{"Weekly Digest", "weekly-digest", "Every issue, every Tuesday.", "Weekly Digest", 16},
		{"Product Updates", "product-updates", "What shipped and what's retiring.", "Product Updates", 16},
		{"Engineering", "engineering", "How we build and run Acme.", "", 16},
		{"Deliverability", "deliverability", "Getting email to the inbox.", "", 15},
		{"Insider", "insider", "Deep dives for paid members.", "", 15},
		{"Announcements", "announcements", "News about the newsletter itself.", "", 15},
		{"Ask Acme", "ask-acme", "Your questions, answered by the team.", "", 12},
		{"Beta Program", "beta-program", "Previews for Beta Program readers.", "Beta Program", 8},
	} {
		id := ID()
		var list *uuid.UUID
		if t.list != "" {
			l := w.lists[t.list]
			list = &l
		}
		at := s.designDay(t.days, 9, 32)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_tags (id, workspace_id, name, slug, description, list_id, created_by, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)`, id, w.ws.ID, t.name, t.slug, t.desc, list, linh.ID, at); err != nil {
			return err
		}
		w.tags[t.name] = id
	}
	// The header: the three tags, then the Field Guide and About pages
	// (shown once the page builder publishes them).
	header := []website.WebsiteNavLink{{Kind: website.NavTag, Ref: w.tags["Weekly Digest"].String()}, {Kind: website.NavTag, Ref: w.tags["Product Updates"].String()},
		{Kind: website.NavTag, Ref: w.tags["Engineering"].String()}, {Kind: website.NavPage, Ref: "field-guide"}, {Kind: website.NavPage, Ref: "about"}}
	sections := []website.WebsiteHomeSection{{Kind: website.SectionSubscribe, On: true, Note: "Free, every Tuesday. Unsubscribe any time with one click.",
		Text: "The Weekly Digest, every Tuesday. Join 48,000 people building on Acme."}, {Kind: website.SectionFeatured, On: true},
		{Kind: website.SectionLatest, On: true, Count: 6}, {Kind: website.SectionTag, On: true, Tag: w.tags["Engineering"].String(), Count: 3},
		{Kind: website.SectionAbout, On: true}, {Kind: website.SectionTiers, On: true}}
	hj, _ := json.Marshal(header)
	sj, _ := json.Marshal(sections)
	if _, err := s.Tx.Exec(ctx, `UPDATE website_settings SET header_links = $2, sections = $3 WHERE workspace_id = $1`, w.ws.ID, hj, sj); err != nil {
		return err
	}
	// Every archived campaign: a post at its archive's address.
	type issue struct {
		campaign             uuid.UUID
		slug, title, excerpt string
		list                 uuid.UUID
		subject, preheader   string
		mjml                 string
		sent                 time.Time
		name                 string
	}
	rows, err := s.Tx.Query(ctx, `SELECT ai.campaign_id, ai.slug, ai.title, ai.excerpt, ai.list_id, c.subject, c.preheader, c.mjml,
		coalesce(c.started_at, c.completed_at, c.created_at), c.name
		FROM archive_issues ai JOIN campaigns c ON c.id = ai.campaign_id
		WHERE ai.workspace_id = $1 AND NOT ai.hidden ORDER BY coalesce(c.started_at, c.completed_at, c.created_at)`, w.ws.ID)
	if err != nil {
		return err
	}
	var issues []issue
	for rows.Next() {
		var is issue
		var list *uuid.UUID
		if err := rows.Scan(&is.campaign, &is.slug, &is.title, &is.excerpt, &list, &is.subject, &is.preheader, &is.mjml, &is.sent, &is.name); err != nil {
			rows.Close()
			return err
		}
		if list != nil {
			is.list = *list
		}
		issues = append(issues, is)
	}
	rows.Close()
	// Covers and extra tags of the posts the boards name.
	extra := map[string][]string{
		"the-case-for-boring-infrastructure":                    {"Engineering"},
		"shipping-on-fridays-revisited":                         {"Engineering"},
		"reading-the-audit-log-like-a-detective":                {"Engineering"},
		"134-how-we-test-migrations-before-they-ship":           {"Engineering"},
		"136-a-field-guide-to-rate-limits":                      {"Engineering"},
		"131-inside-our-incident-review":                        {"Engineering"},
		"138-postgres-18-in-production-the-first-month":         {"Engineering"},
		"we-have-moved-to-news-acme-io":                         {"Announcements"},
		"108-year-in-review-what-shipped-in-2025":               {"Announcements"},
		"137-what-our-support-queue-taught-us-about-onboarding": {"Deliverability"},
	}
	covers := map[string]string{
		"what-shipped-in-october":                     "october-update",
		"the-case-for-boring-infrastructure":          "boring-infrastructure",
		"shipping-on-fridays-revisited":               "fridays",
		"reading-the-audit-log-like-a-detective":      "audit-log",
		"134-how-we-test-migrations-before-they-ship": "migration-tests",
		"we-have-moved-to-news-acme-io":               "welcome-site",
		"131-inside-our-incident-review":              "on-call",
		"128-why-we-write-design-docs-before-code":    "write-the-digest",
	}
	for _, is := range issues {
		var tags []uuid.UUID
		if t, ok := w.tagOfList(is.list); ok {
			tags = append(tags, t)
		}
		for _, n := range extra[is.slug] {
			tags = append(tags, w.tags[n])
		}
		cover := w.covers[covers[is.slug]]
		alt := coverAlts[covers[is.slug]]
		id, err := w.post(ctx, postSpec{kind: website.KindCampaign, campaign: &is.campaign, title: is.title, slug: is.slug, excerpt: is.excerpt, cover: cover,
			coverAlt: alt, visibility: "public", mjml: is.mjml, status: website.PostPublished, mode: website.WhenWithSend, publishedAt: is.sent, tags: tags,
			createdAt: w.live, byline: "The Acme team"})
		if err != nil {
			return fmt.Errorf("post %s: %w", is.slug, err)
		}
		w.posts[is.slug] = id
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_redirects (workspace_id, path, post_id, created_at) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
			w.ws.ID, "archive/"+is.slug, id, w.live); err != nil {
			return err
		}
		weight := 1.0
		switch is.slug {
		case "what-shipped-in-october":
			weight = 9874
		case "the-case-for-boring-infrastructure":
			weight = 7140
		}
		w.published = append(w.published, sitePost{id: id, at: is.sent, weight: weight})
	}
	return nil
}

// coverAlts describe the covers.
var coverAlts = map[string]string{
	"october-update":        "A month calendar with most days ticked off and the last week struck through",
	"postgres-runbook":      "Two database cylinders joined by a dashed arrow, the old one outlined, the new one filled in indigo",
	"feature-flags":         "Rows of on and off switches, most of them on, in indigo and lavender",
	"ask-acme":              "Two speech bubbles, a question in white and an answer in indigo",
	"boring-infrastructure": "Four plain server racks in a row, one highlighted in indigo",
	"fridays":               "A week of seven days with Friday highlighted and a check mark on it",
	"audit-log":             "Lines of a log with a magnifying glass over one highlighted line",
	"migration-tests":       "A database cylinder with a check mark beside it",
	"welcome-site":          "A browser window with a page layout inside",
	"on-call":               "A phone showing an alert next to a clock face",
	"write-the-digest":      "A page of text lines with a pen resting across it",
	"rate-limits":           "Bars of different heights under a dashed limit line, one bar highlighted",
}

func (w *siteSeed) tagOfList(list uuid.UUID) (uuid.UUID, bool) {
	for name, l := range w.lists {
		if l == list {
			t, ok := w.tags[name]
			return t, ok
		}
	}
	return uuid.Nil, false
}

// postSpec is one post of the sample.
type postSpec struct {
	kind                   string
	campaign               *uuid.UUID
	title, slug, excerpt   string
	cover, coverAlt        string
	visibility             string
	tiers                  []uuid.UUID
	neverFree, webOn       bool
	mjml                   string
	status, mode           string
	publishAt, publishedAt time.Time
	publishTZ              string
	tags                   []uuid.UUID
	byUser                 *uuid.UUID
	byline                 string
	createdBy, publishedBy *uuid.UUID
	createdAt              time.Time
	unpublishedAt          time.Time
	unpublishedBy          *uuid.UUID
	reason                 string
	version                int
}

// post writes a post, rendered as publishing it would.
func (w *siteSeed) post(ctx context.Context, p postSpec) (uuid.UUID, error) {
	s := w.s
	id := ID()
	res, err := render.Render(p.mjml, render.Options{Title: p.title, CoverSrc: p.cover,
		Links: map[string]string{"preferences_url": "/preferences", "unsubscribe_url": "/preferences", "view_in_browser_url": "/posts/" + p.slug}})
	if err != nil {
		res = render.Result{}
	}
	notes, _ := json.Marshal(map[string]any{"hidden": nonNil(res.Hidden), "fallbacks": nonNil(res.Fallbacks)})
	paid, _ := json.Marshal(nonNil(res.Paid))
	tiers := p.tiers
	if tiers == nil {
		tiers = []uuid.UUID{}
	}
	webOn := p.webOn || p.kind == website.KindWeb || p.status == website.PostPublished || p.status == website.PostUnpublished
	var publishedAt, firstPublished, publishAt, unpublishedAt *time.Time
	if !p.publishedAt.IsZero() {
		publishedAt, firstPublished = &p.publishedAt, &p.publishedAt
	}
	if !p.publishAt.IsZero() {
		publishAt = &p.publishAt
	}
	if !p.unpublishedAt.IsZero() {
		unpublishedAt = &p.unpublishedAt
	}
	version := p.version
	if version == 0 && p.kind == website.KindWeb {
		version = 1
	}
	created := p.createdAt
	if created.IsZero() {
		created = w.live
	}
	mjmlSrc := ""
	if p.kind == website.KindWeb {
		mjmlSrc = p.mjml
	}
	published := ""
	if publishedAt != nil {
		published = p.mjml
	}
	row := store.WebsitePost{ID: id, Kind: p.kind, CampaignID: p.campaign, Title: p.title, Excerpt: p.excerpt, CoverUrl: p.cover, Slug: p.slug, Visibility: p.visibility,
		Tiers: tiers, NeverFree: p.neverFree, PublishMode: p.mode, PublishAt: publishAt}
	hash := website.PostHash(row, p.mjml)
	byName := p.byline
	if p.byUser != nil {
		byName = ""
	}
	_, err = s.Tx.Exec(ctx, `INSERT INTO website_posts (id, workspace_id, kind, campaign_id, web_on, status, publish_mode, publish_at, publish_tz, title, title_edited,
		slug, excerpt, excerpt_edited, cover_url, cover_alt, byline_user_id, byline_name, visibility, tiers, never_free, mjml, version, body_html, preview_html,
		has_marker, marker_after, body_text, preview_text, words, render_notes, rendered_hash, published_at, published_by, first_published_at, unpublished_at,
		unpublished_by, unpublish_reason, created_by, created_at, updated_by, updated_at, published_mjml, published_version, paid_blocks)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,true,$11,$12,true,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,
		$35,$36,$37,$38,$37,$38,$39,$40,$41)`,
		id, w.ws.ID, p.kind, p.campaign, webOn, p.status, p.mode, publishAt, p.publishTZ, p.title, p.slug, p.excerpt, p.cover, p.coverAlt, p.byUser, byName,
		p.visibility, tiers, p.neverFree, mjmlSrc, version, res.Body, res.Preview, res.HasMarker, res.MarkerAfter, res.Text, res.PreviewText, res.Words, notes, hash,
		publishedAt, p.publishedBy, firstPublished, unpublishedAt, p.unpublishedBy, p.reason, p.createdBy, created, published, version, paid)
	if err != nil {
		return id, err
	}
	for i, t := range p.tags {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_post_tags (post_id, tag_id, position) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, id, t, i); err != nil {
			return id, err
		}
	}
	if p.kind == website.KindWeb {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_post_versions (post_id, version, title, mjml, saved_by, saved_at, label) VALUES ($1,$2,$3,$4,$5,$6,'')`,
			id, version, p.title, p.mjml, p.createdBy, created); err != nil {
			return id, err
		}
	}
	return id, nil
}

func nonNil[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}

// campaignPosts: #142 goes on the web with its send; "Feature flags
// preview" went up for Beta Program readers and came down on 10 Oct.
func (w *siteSeed) campaignPosts(ctx context.Context) error {
	s := w.s
	sofia := s.Users["sofia@acme.io"].ID
	for _, c := range []struct {
		name, title, slug, excerpt, cover string
		tags                              []string
		status                            string
		published, unpublished            time.Time
		reason                            string
	}{
		{name: "Weekly Digest #142", title: "Feature flags at scale: what 10,000 flags taught us", slug: "feature-flags-at-scale",
			excerpt: "Feature flags are generally available. Here's what 10,000 flags in our own codebase taught us about naming, cleanup and rollouts.",
			cover:   "feature-flags", tags: []string{"Weekly Digest", "Engineering"}, status: website.PostDraft},
		{name: "Feature flags preview", title: "Feature flags preview", slug: "feature-flags-preview",
			excerpt: "Flag dependencies and scheduled rollouts, for Beta Program readers.", tags: []string{"Beta Program"}, status: website.PostUnpublished,
			published: s.designDay(13, 15, 0), unpublished: s.designDay(5, 11, 20), reason: "Beta content, not for the public."},
	} {
		cid, ok := s.Campaigns[c.name]
		if !ok {
			if err := s.Tx.QueryRow(ctx, `SELECT id FROM campaigns WHERE workspace_id = $1 AND name = $2 AND deleted_at IS NULL`, w.ws.ID, c.name).Scan(&cid); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				return err
			}
		}
		var src string
		if err := s.Tx.QueryRow(ctx, `SELECT mjml FROM campaigns WHERE id = $1`, cid).Scan(&src); err != nil {
			return err
		}
		var tags []uuid.UUID
		for _, n := range c.tags {
			tags = append(tags, w.tags[n])
		}
		spec := postSpec{kind: website.KindCampaign, campaign: &cid, title: c.title, slug: c.slug, excerpt: c.excerpt, cover: w.covers[c.cover],
			coverAlt: coverAlts[c.cover], visibility: "public", webOn: true, mjml: src, status: c.status, mode: website.WhenWithSend, tags: tags,
			createdBy: &sofia, createdAt: s.designDay(3, 15, 10), byline: "The Acme team"}
		if c.status == website.PostUnpublished {
			spec.publishedAt, spec.unpublishedAt, spec.unpublishedBy, spec.reason, spec.publishedBy = c.published, c.unpublished, &sofia, c.reason, &sofia
			spec.createdAt = c.published.Add(-time.Hour)
		}
		id, err := w.post(ctx, spec)
		if err != nil {
			return err
		}
		w.posts[c.slug] = id
		if c.status == website.PostUnpublished {
			// 412 views before it came down.
			if err := w.addViews(ctx, id, c.published, c.unpublished, 412); err != nil {
				return err
			}
		}
	}
	return nil
}

// webPosts are the posts written for the web.
func (w *siteSeed) webPosts(ctx context.Context) error {
	s := w.s
	sofia, marcus, linh, daniel := s.Users["sofia@acme.io"].ID, s.Users["marcus@acme.io"].ID, s.Users["linh@acme.io"].ID, s.Users["daniel@acme.io"].ID
	insider, founding := w.tiers["Weekly Digest Insider"], w.tiers["Founding member"]
	specs := []postSpec{
		{slug: "postgres-migration-runbook", title: "Inside our Postgres migration: the runbook", cover: "postgres-runbook", visibility: "paid",
			excerpt: "How we moved 2.1 TB from Postgres 14 to 17 with a 41-second write pause: sizing, seeding, nine days of replication and the cutover.",
			mjml:    websiteRunbookMJML, status: website.PostPublished, mode: website.WhenAt, publishedAt: s.designDay(2, 8, 0), publishAt: s.designDay(2, 8, 0),
			publishTZ: "America/New_York", tags: []uuid.UUID{w.tags["Engineering"], w.tags["Insider"]}, byUser: &sofia, createdBy: &sofia, publishedBy: &sofia,
			createdAt: s.designDay(6, 10, 15), version: 14},
		{slug: "ask-acme-deliverability-questions", title: "Ask Acme: your deliverability questions, answered", cover: "ask-acme", visibility: "members",
			excerpt: "Why Gmail sends some of your mail to Promotions, whether to warm up a new IP, and what a DMARC report is really telling you.",
			mjml:    websiteAskAcmeMJML, status: website.PostPublished, mode: website.WhenNow, publishedAt: s.designDay(5, 9, 0),
			tags: []uuid.UUID{w.tags["Ask Acme"], w.tags["Deliverability"]}, byUser: &marcus, createdBy: &marcus, publishedBy: &marcus, createdAt: s.designDay(7, 14, 0), version: 6},
		{slug: "rate-limits-best-customers", title: "Rate limits that don't punish your best customers", cover: "rate-limits", visibility: "paid",
			excerpt: "Token buckets per customer, not per IP: how we stopped throttling the accounts that send the most.",
			mjml:    websiteRateLimitsMJML, status: website.PostPublished, mode: website.WhenNow, publishedAt: s.designDay(9, 9, 0),
			tags: []uuid.UUID{w.tags["Engineering"], w.tags["Insider"]}, byUser: &sofia, createdBy: &sofia, publishedBy: &sofia, createdAt: s.designDay(12, 11, 0), version: 8},
		{slug: "founding-members-ama-october", title: "Founding members' AMA: recording and notes", visibility: "paid", tiers: []uuid.UUID{founding}, neverFree: true,
			excerpt: "The full recording of the October AMA, with notes and timestamps for every question.", mjml: websiteAMAMJML, status: website.PostScheduled,
			mode: website.WhenAt, publishAt: s.designDay(-1, 17, 0), publishTZ: "America/New_York", tags: []uuid.UUID{w.tags["Ask Acme"], w.tags["Insider"]},
			byUser: &linh, createdBy: &linh, createdAt: s.designDay(1, 11, 30), version: 3},
		{slug: "q4-roadmap-deep-dive", title: "Q4 roadmap deep dive", visibility: "members",
			excerpt: "What we're building this quarter, what we're cutting, and why the v1 API retirement comes first.", mjml: websiteRoadmapMJML,
			status: website.PostDraft, mode: website.WhenNow, tags: []uuid.UUID{w.tags["Product Updates"]}, byUser: &daniel, createdBy: &daniel,
			createdAt: s.designDay(1, 13, 40), version: 9},
	}
	_ = insider
	for _, p := range specs {
		p.kind = website.KindWeb
		cover := p.cover
		p.cover, p.coverAlt = w.covers[cover], coverAlts[cover]
		id, err := w.post(ctx, p)
		if err != nil {
			return fmt.Errorf("post %s: %w", p.slug, err)
		}
		w.posts[p.slug] = id
		switch p.slug {
		case "postgres-migration-runbook":
			w.published = append(w.published, sitePost{id: id, at: p.publishedAt, weight: 3912})
		case "ask-acme-deliverability-questions":
			w.published = append(w.published, sitePost{id: id, at: p.publishedAt, weight: 2288})
		case "rate-limits-best-customers":
			w.published = append(w.published, sitePost{id: id, at: p.publishedAt, weight: 1210})
		}
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE website_posts SET autosaved_at = $2 WHERE id = $1`, w.posts["q4-roadmap-deep-dive"], s.designDay(1, 16, 2)); err != nil {
		return err
	}
	return nil
}

// featured: the October update and the runbook.
func (w *siteSeed) featured(ctx context.Context) error {
	ids := []uuid.UUID{}
	for _, slug := range []string{"what-shipped-in-october", "postgres-migration-runbook"} {
		if id, ok := w.posts[slug]; ok {
			ids = append(ids, id)
		}
	}
	_, err := w.s.Tx.Exec(ctx, `UPDATE website_settings SET featured = $2 WHERE workspace_id = $1`, w.ws.ID, ids)
	return err
}

// addViews spreads a post's views over its days on the site.
func (w *siteSeed) addViews(ctx context.Context, post uuid.UUID, from, until time.Time, total int64) error {
	days := int(until.Sub(from).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	left := total
	for d := 0; d < days && left > 0; d++ {
		// Most views in the first days.
		share := total * int64(days-d) * 2 / int64(days*(days+1))
		if d == days-1 || share > left {
			share = left
		}
		if share <= 0 {
			share = 1
		}
		day := dayOf(from.AddDate(0, 0, d))
		if _, err := w.s.Tx.Exec(ctx, `INSERT INTO website_view_days (workspace_id, day, page_kind, page_id, views) VALUES ($1,$2,'post',$3,$4)
			ON CONFLICT (workspace_id, day, page_kind, page_id) DO UPDATE SET views = website_view_days.views + EXCLUDED.views`, w.ws.ID, day, post, share); err != nil {
			return err
		}
		left -= share
	}
	return nil
}

func dayOf(t time.Time) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

// views are the site's last 30 days: 61,420 views, 23,910 visitors; the
// posts' own numbers as the boards show them; where the October update's
// readers came from; signups and upgrades from the posts.
func (w *siteSeed) views(ctx context.Context) error {
	s := w.s
	today := s.designDay(0, 14, 0)
	start := today.AddDate(0, 0, -29)
	var postsTotal int64
	for _, p := range w.published {
		if p.weight <= 1 {
			continue
		}
		from := p.at
		if from.Before(start) {
			from = start
		}
		if err := w.addViews(ctx, p.id, from, today, int64(p.weight)); err != nil {
			return err
		}
		postsTotal += int64(p.weight)
	}
	// The rest of the archive's posts share 9,000 views, the home page,
	// the archive and tag pages the remainder.
	var older []sitePost
	for _, p := range w.published {
		if p.weight <= 1 {
			older = append(older, p)
		}
	}
	rest := int64(9000)
	for i, p := range older {
		n := rest / int64(len(older)-i)
		if i < len(older)-1 {
			n = n * int64(80+w.rng.IntN(40)) / 100
		}
		if err := w.addViews(ctx, p.id, start, today, max(n, 1)); err != nil {
			return err
		}
		rest -= n
		postsTotal += n
	}
	pages := int64(61420) - postsTotal - 412
	for d := 0; d < 30; d++ {
		day := dayOf(start.AddDate(0, 0, d))
		home := pages * 6 / 10 / 30
		arch := pages * 2 / 10 / 30
		tag := pages*2/10/30 + 0
		if d == 29 {
			home = pages*6/10 - home*29
			arch = pages*2/10 - arch*29
			tag = pages - pages*6/10 - pages*2/10 - tag*29
		}
		for _, v := range []struct {
			kind  string
			id    uuid.UUID
			views int64
		}{{"home", uuid.Nil, home}, {"archive", uuid.Nil, arch}, {"tag", w.tags["Engineering"], tag}} {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO website_view_days (workspace_id, day, page_kind, page_id, views) VALUES ($1,$2,$3,$4,$5)
				ON CONFLICT (workspace_id, day, page_kind, page_id) DO UPDATE SET views = website_view_days.views + EXCLUDED.views`,
				w.ws.ID, day, v.kind, v.id, v.views); err != nil {
				return err
			}
		}
		visitors := int64(23910) / 30
		if d == 29 {
			visitors = 23910 - visitors*29
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_visitor_days (workspace_id, day, visitors) VALUES ($1,$2,$3)
			ON CONFLICT (workspace_id, day) DO UPDATE SET visitors = EXCLUDED.visitors`, w.ws.ID, day, visitors); err != nil {
			return err
		}
	}
	// Where the October update's readers came from.
	if id, ok := w.posts["what-shipped-in-october"]; ok {
		at := dayOf(s.designDay(5, 12, 0))
		for _, src := range []struct {
			dim, key string
			views    int64
		}{{"referrer", "", 3402}, {"referrer", "google.com", 2611}, {"referrer", "news.ycombinator.com", 1980}, {"referrer", "linkedin.com", 1104},
			{"referrer", "x.com", 301}, {"referrer", "reddit.com", 268}, {"referrer", "bsky.app", 208}, {"utm", "newsletter", 2140}, {"utm", "linkedin", 610}} {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO website_view_sources (workspace_id, post_id, day, dim, key, views) VALUES ($1,$2,$3,$4,$5,$6)`,
				w.ws.ID, id, at, src.dim, src.key, src.views); err != nil {
				return err
			}
		}
	}
	// Signups and upgrades from posts: 1,312 and 41 in 30 days.
	conv := []struct {
		slug              string
		signups, upgrades int
	}{{"what-shipped-in-october", 212, 6}, {"the-case-for-boring-infrastructure", 164, 4}, {"postgres-migration-runbook", 46, 17},
		{"ask-acme-deliverability-questions", 131, 2}, {"rate-limits-best-customers", 21, 5}}
	signups, upgrades := 1312, 41
	for _, c := range conv {
		signups -= c.signups
		upgrades -= c.upgrades
	}
	conv = append(conv, struct {
		slug              string
		signups, upgrades int
	}{"shipping-on-fridays-revisited", signups, upgrades})
	for _, c := range conv {
		id, ok := w.posts[c.slug]
		if !ok {
			continue
		}
		for kind, n := range map[string]int{"signup": c.signups, "upgrade": c.upgrades} {
			for i := 0; i < n; i++ {
				at := today.Add(-time.Duration(w.rng.IntN(29*24)) * time.Hour)
				if _, err := s.Tx.Exec(ctx, `INSERT INTO website_post_conversions (id, workspace_id, post_id, kind, occurred_at) VALUES ($1,$2,$3,$4,$5)`,
					ID(), w.ws.ID, id, kind, at); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// members: 2,804 contacts signed in since the site went live, 1,126 in
// the last 7 days. Sam Rivera (a Weekly Digest Insider in Earn's sample)
// signed in with a link on 12 Oct; Alex Murphy, a free member, read his
// free paid post for October, the runbook, on 13 Oct.
func (w *siteSeed) members(ctx context.Context) error {
	s := w.s
	now := s.designDay(0, 14, 20)
	week := now.AddDate(0, 0, -7)
	for _, m := range []struct {
		email, method, key string
		first, last        time.Time
		joined             string
		read               bool
	}{
		{email: "sam.rivera@outlook.com", method: "link", key: "sam", first: s.designDay(27, 10, 5), last: s.designDay(3, 18, 41),
			joined: "/posts/ask-acme-deliverability-questions"},
		{email: "a.murphy@icloud.com", method: "code", key: "alex", first: s.designDay(13, 8, 50), last: s.designDay(2, 19, 22),
			joined: "/posts/postgres-migration-runbook", read: true},
	} {
		var id uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = $2`, w.ws.ID, m.email).Scan(&id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_members (contact_id, workspace_id, first_signed_in, last_signed_in, last_method, sign_ins)
			VALUES ($1,$2,$3,$4,$5,3) ON CONFLICT (contact_id) DO NOTHING`, id, w.ws.ID, m.first, m.last, m.method); err != nil {
			return err
		}
		same := true
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_member_events (id, workspace_id, contact_id, kind, method, same_browser, detail, occurred_at)
			VALUES ($1,$2,$3,'signed_in',$4,$5,'',$6), ($7,$2,$3,'joined','signup',$5,$8,$9)`,
			ID(), w.ws.ID, id, m.method, same, m.last, ID(), m.joined, m.first); err != nil {
			return err
		}
		// The session's cookie is "demo-session-<key>-<workspace id>" (QA signs in with it).
		tok := sha256.Sum256([]byte("demo-session-" + m.key + "-" + w.ws.ID.String()))
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_sessions (id, workspace_id, contact_id, token_hash, method, created_at, last_seen_at, expires_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, ID(), w.ws.ID, id, tok[:], m.method, m.last, s.designDay(0, 9, 12), m.last.AddDate(0, 0, 90)); err != nil {
			return err
		}
		if post, ok := w.posts["postgres-migration-runbook"]; ok && m.read {
			read := s.designDay(2, 19, 30)
			if _, err := s.Tx.Exec(ctx, `INSERT INTO website_meter_reads (workspace_id, month, reader, post_id, rule, read_at) VALUES ($1,$2,$3,$4,'member',$5)
				ON CONFLICT DO NOTHING`, w.ws.ID, monthStart(read), "c:"+id.String(), post, read); err != nil {
				return err
			}
		}
	}
	// The rest: Weekly Digest subscribers, up to 2,804 members, 1,126 of
	// them signed in during the last 7 days.
	_, err := s.Tx.Exec(ctx, `INSERT INTO website_members (contact_id, workspace_id, first_signed_in, last_signed_in, last_method, sign_ins)
		SELECT c.id, c.workspace_id, $3::timestamptz + (row_number() OVER (ORDER BY c.id) % 16) * interval '1 day',
		       CASE WHEN row_number() OVER (ORDER BY c.id) <= 1124 THEN $4::timestamptz + (row_number() OVER (ORDER BY c.id) % 7) * interval '1 day'
		            ELSE $3::timestamptz + (row_number() OVER (ORDER BY c.id) % 8) * interval '1 day' END,
		       CASE WHEN c.id::text < '8' THEN 'link' ELSE 'code' END, 1 + (row_number() OVER (ORDER BY c.id) % 4)::int
		FROM contacts c
		JOIN list_memberships lm ON lm.contact_id = c.id AND lm.list_id = $2 AND lm.status = 'subscribed'
		WHERE c.workspace_id = $1 AND c.status = 'active' AND c.erased_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM website_members m WHERE m.contact_id = c.id)
		ORDER BY c.id
		LIMIT GREATEST(2804 - (SELECT count(*) FROM website_members m WHERE m.workspace_id = $1), 0)`, w.ws.ID, w.lists["Weekly Digest"], w.live, week)
	return err
}

func monthStart(t time.Time) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// approval: Daniel Park asked to publish "Q4 roadmap deep dive" on 14 Oct
// at 16:10; Marcus Chen or Linh Nguyen can approve.
func (w *siteSeed) approval(ctx context.Context) error {
	s := w.s
	id, ok := w.posts["q4-roadmap-deep-dive"]
	if !ok {
		return nil
	}
	daniel := s.Users["daniel@acme.io"].ID
	row := store.WebsitePost{ID: id, Kind: website.KindWeb, Title: "Q4 roadmap deep dive", Excerpt: "What we're building this quarter, what we're cutting, and why the v1 API retirement comes first.",
		Slug: "q4-roadmap-deep-dive", Visibility: "members", Tiers: []uuid.UUID{}, PublishMode: website.WhenNow}
	hash := website.PostHash(row, websiteRoadmapMJML)
	snap, _ := json.Marshal(map[string]any{"draft_hash": hash, "subject": row.Title, "preheader": row.Excerpt, "author_role": "Content Writer",
		"rules": []map[string]any{{"key": "author_lacks_send", "label": "The author lacks the send permission (website:publish)", "short": "author lacks publish",
			"badge": "Author lacks publish", "author": "Daniel Park", "role": "Content Writer"}},
		"website": map[string]any{"postId": id.String(), "title": row.Title, "kind": "web", "slug": row.Slug, "path": "/posts/" + row.Slug, "visibility": "members",
			"visibilityLabel": "Members", "when": "now", "excerpt": row.Excerpt, "published": false}})
	_, err := s.Tx.Exec(ctx, `INSERT INTO approval_requests (id, workspace_id, subject_type, website_post_id, requested_by, requested_at, note, status, matched_rules,
		approvals_required, snapshot, bypass_reason) VALUES ($1,$2,'website_post',$3,$4,$5,$6,'pending',ARRAY['author_lacks_send'],1,$7,'')`,
		ID(), w.ws.ID, id, daniel, s.designDay(1, 16, 10), "Ready for Thursday. The cut list is final; Marcus checked the API dates.", snap)
	return err
}

// audit: the website's history in the audit log.
func (w *siteSeed) audit(ctx context.Context) error {
	s := w.s
	linh, sofia, daniel := s.Users["linh@acme.io"], s.Users["sofia@acme.io"], s.Users["daniel@acme.io"]
	type entry struct {
		at                   time.Time
		actorType            string
		actorID              *uuid.UUID
		actorName            string
		action, targetType   string
		targetID, targetName string
		after                map[string]any
	}
	ip := netip.MustParseAddr("203.0.113.24")
	entries := []entry{
		{w.live, "user", &linh.ID, linh.Name, "website.turned_on", "website", w.ws.ID.String(), w.ws.Name, map[string]any{"status": "turning_on", "address": "news.acme.io", "layout": "featured", "archived_campaigns": 44}},
		{w.live.Add(3 * time.Minute), "system", nil, "ReputeCast", "website.live", "website", w.ws.ID.String(), w.ws.Name, map[string]any{"status": "on", "posts": 44, "redirects": 44}},
		{w.live.Add(40 * time.Minute), "user", &linh.ID, linh.Name, "website.meter_updated", "website", w.ws.ID.String(), w.ws.Name, map[string]any{"visitors": 2, "members": 1}},
		{s.designDay(2, 8, 0), "user", &sofia.ID, sofia.Name, "post.published", "website_post", w.posts["postgres-migration-runbook"].String(), "Inside our Postgres migration: the runbook", map[string]any{"status": "published", "visibility": "paid"}},
		{s.designDay(5, 11, 20), "user", &sofia.ID, sofia.Name, "post.unpublished", "website_post", w.posts["feature-flags-preview"].String(), "Feature flags preview", map[string]any{"status": "unpublished", "reason": "Beta content, not for the public."}},
		{s.designDay(1, 16, 10), "user", &daniel.ID, daniel.Name, "post.publish_approval_requested", "website_post", w.posts["q4-roadmap-deep-dive"].String(), "Q4 roadmap deep dive", map[string]any{"status": "pending", "path": "/posts/q4-roadmap-deep-dive", "visibility": "members"}},
	}
	for _, e := range entries {
		if e.targetID == uuid.Nil.String() {
			continue
		}
		after, _ := json.Marshal(e.after)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
			result, ip, user_agent, request_id, after) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'allowed',$11,'',$12,$13)`,
			ID(), w.ws.ID, e.at, e.actorType, e.actorID, e.actorName, e.action, e.targetType, e.targetID, e.targetName, ip, "req_"+ID().String()[:8], after); err != nil {
			return err
		}
	}
	return nil
}
