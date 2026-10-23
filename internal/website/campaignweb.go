package website

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// WebsiteCampaignWeb is a campaign's Web tab in the composer: whether it
// goes on the website and how.
type WebsiteCampaignWeb struct {
	// SiteOn: the website is on (off, the tab says so and offers nothing).
	SiteOn bool   `json:"siteOn"`
	Host   string `json:"host"`
	// Chip is the composer header's Web chip.
	Chip string `json:"chip" enum:"off,with_send,scheduled,published,unpublished"`
	// Post is the campaign's post: saved settings, or the defaults (ID "").
	Post WebsitePost `json:"post"`
	// CampaignStatus, ScheduledAt and SentAt say when "with the send" is.
	CampaignStatus string     `json:"campaignStatus"`
	ScheduledAt    *time.Time `json:"scheduledAt,omitempty"`
	SentAt         *time.Time `json:"sentAt,omitempty"`
	// FollowsAudience is the visibility rule for posts from campaigns
	// ("Follows the audience: the Weekly Digest is free.").
	FollowsAudience bool     `json:"followsAudience"`
	AudienceLists   []string `json:"audienceLists"`
}

// CampaignWeb returns a campaign's Web tab.
func (s *Service) CampaignWeb(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (WebsiteCampaignWeb, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteCampaignWeb{}, err
	}
	camp, err := s.campaign(ctx, ws, campaignID)
	if err != nil {
		return WebsiteCampaignWeb{}, err
	}
	out := WebsiteCampaignWeb{SiteOn: c.On(), Host: hostOnly(strings.TrimSuffix(s.sitePath(ctx, ws), "/")), CampaignStatus: camp.Status,
		ScheduledAt: camp.ScheduledAt, SentAt: camp.StartedAt, FollowsAudience: c.row.CampaignVisibility == "audience", AudienceLists: camp.listNames}
	post, err := s.Q.WebsitePostByCampaign(ctx, &campaignID)
	switch {
	case err == nil:
		if out.Post, err = s.Post(ctx, ws, p, post.ID); err != nil {
			return out, err
		}
	case errors.Is(err, pgx.ErrNoRows):
		out.Post = s.defaultPost(ctx, ws, c, camp)
	default:
		return out, err
	}
	out.Chip = campaignChip(out.Post)
	return out, nil
}

// campaignChip is the Web chip: "Not on the web", "Publishes with the
// send", "Scheduled for the web", "Published".
func campaignChip(p WebsitePost) string {
	switch {
	case p.Status == PostPublished:
		return "published"
	case p.Status == PostUnpublished:
		return "unpublished"
	case !p.WebOn:
		return "off"
	case p.PublishMode == WhenAt:
		return "scheduled"
	}
	return "with_send"
}

// campaignFacts is what a campaign's post needs of it.
type campaignFacts struct {
	ID          uuid.UUID
	Name        string
	Status      string
	Subject     string
	Preheader   string
	MJML        string
	Audience    json.RawMessage
	ScheduledAt *time.Time
	StartedAt   *time.Time
	lists       []uuid.UUID
	listNames   []string
}

func (s *Service) campaign(ctx context.Context, ws *store.Workspace, id uuid.UUID) (campaignFacts, error) {
	var f campaignFacts
	err := s.Pool.QueryRow(ctx, `SELECT id, name, status, subject, preheader, mjml, audience, scheduled_at, started_at
		FROM campaigns WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, id, ws.ID).
		Scan(&f.ID, &f.Name, &f.Status, &f.Subject, &f.Preheader, &f.MJML, &f.Audience, &f.ScheduledAt, &f.StartedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, notFound(ctx, "campaign")
	}
	if err != nil {
		return f, err
	}
	var aud struct {
		Lists []uuid.UUID `json:"lists"`
	}
	_ = json.Unmarshal(f.Audience, &aud)
	f.lists, f.listNames = aud.Lists, []string{}
	if len(aud.Lists) > 0 {
		rows, err := s.Pool.Query(ctx, `SELECT name FROM lists WHERE id = ANY ($1) ORDER BY created_at`, aud.Lists)
		if err == nil {
			for rows.Next() {
				var n string
				if rows.Scan(&n) == nil {
					f.listNames = append(f.listNames, n)
				}
			}
			rows.Close()
		}
	}
	return f, nil
}

// paidAudience reports a campaign sent to paid members only (Earn's "Who on
// this list", X16) and its lowest tier (nil: any paid tier): its post
// follows it behind the paywall, for that tier and above.
func paidAudience(audience json.RawMessage) (bool, []uuid.UUID) {
	a := paidaccess.Parse(audience)
	if a.Who != paidaccess.Paid {
		return false, nil
	}
	if id, err := uuid.Parse(a.Tier); err == nil {
		return true, []uuid.UUID{id}
	}
	return true, nil
}

// paidOnly reports a campaign sent to paid members only.
func paidOnly(audience json.RawMessage) bool {
	paid, _ := paidAudience(audience)
	return paid
}

// defaultPost is a campaign's post before anyone saves its Web tab: the
// subject as the title, the preheader as the excerpt, the first image as
// the cover, its lists' tags, the visibility that follows the audience,
// on the web when its lists are in the archive.
func (s *Service) defaultPost(ctx context.Context, ws *store.Workspace, c *config, camp campaignFacts) WebsitePost {
	title := forms.RenderTitle(camp.Subject)
	if title == "" {
		title = camp.Name
	}
	excerpt := forms.RenderTitle(camp.Preheader)
	if len([]rune(excerpt)) > 200 {
		excerpt = forms.Excerpt(excerpt, 195)
	}
	cover, alt := firstImage(camp.MJML)
	vis := c.row.CampaignVisibility
	tiers := []WebsiteTierRef{}
	if vis == "audience" {
		vis = gate.Public
		if paid, ids := paidAudience(camp.Audience); paid {
			vis = gate.Paid
			if len(ids) > 0 {
				all, _ := s.tiers(ctx, s.Q, ws)
				tiers = tierRefs(ids, all)
			}
		}
	}
	mode := WhenWithSend
	if camp.StartedAt != nil {
		mode = WhenNow
	}
	set, _ := s.Forms.LoadSettings(ctx, ws)
	byline := ""
	if set != nil {
		byline = tr(ctx, "post.teamByline", map[string]any{"publisher": set.Branding.PublisherName})
	}
	post := WebsitePost{WebOn: s.inArchive(ctx, ws, camp.lists), Base: hostOnly(strings.TrimSuffix(s.sitePath(ctx, ws, "posts", "x"), "x")),
		CoverURL: cover, CoverAlt: alt, Checks: []WebsiteWebCheck{}}
	post.WebsitePostRow = WebsitePostRow{Kind: KindCampaign, Title: title, Excerpt: excerpt, Slug: s.freeSlug(ctx, ws, uuid.Nil, title), Status: PostDraft,
		PublishMode: mode, Visibility: vis, Tiers: tiers, Tags: s.listTags(ctx, ws, camp.lists), Byline: WebsiteByline{Name: byline}}
	post.URL = s.postURL(ws, post.Slug)
	post.Campaign = &WebsitePostCampaign{ID: camp.ID.String(), Name: camp.Name, Status: camp.Status, SentAt: camp.StartedAt, ScheduledAt: camp.ScheduledAt}
	post.Publish = WebsitePublishState{Approvers: []string{}}
	return post
}

// inArchive reports whether a campaign's lists are in the archive: its
// post goes on the web with the send by default, as its issue went in the archive.
func (s *Service) inArchive(ctx context.Context, ws *store.Workspace, lists []uuid.UUID) bool {
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return false
	}
	archived, err := s.Forms.ArchiveLists(ctx, s.Q, ws, set)
	if err != nil {
		return false
	}
	for _, l := range lists {
		if slices.Contains(archived, l) {
			return true
		}
	}
	return false
}

// listTags are the tags made from lists.
func (s *Service) listTags(ctx context.Context, ws *store.Workspace, lists []uuid.UUID) []WebsiteTagRef {
	out := []WebsiteTagRef{}
	for _, l := range lists {
		id := l
		if t, err := s.Q.WebsiteTagByList(ctx, store.WebsiteTagByListParams{WorkspaceID: ws.ID, ListID: &id}); err == nil {
			out = append(out, WebsiteTagRef{ID: t.ID.String(), Name: t.Name, Slug: t.Slug})
		}
	}
	return out
}

// SaveCampaignWeb saves a campaign's Web tab: its post is made on the
// first save.
func (s *Service) SaveCampaignWeb(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, in WebsitePostInput) (WebsiteCampaignWeb, error) {
	camp, err := s.campaign(ctx, ws, campaignID)
	if err != nil {
		return WebsiteCampaignWeb{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostByCampaign(ctx, &campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			c, err := s.load(ctx, q, ws)
			if err != nil {
				return err
			}
			d := s.defaultPost(ctx, ws, c, camp)
			id := newID()
			slug := d.Slug
			if in.Slug != "" {
				slug = in.Slug
			}
			if !slugRe.MatchString(slug) {
				slug = s.freeSlug(ctx, ws, id, d.Title)
			}
			if err := q.WebsitePostInsert(ctx, store.WebsitePostInsertParams{ID: id, WorkspaceID: ws.ID, Kind: KindCampaign, CampaignID: &campaignID, WebOn: in.WebOn,
				Status: PostDraft, PublishMode: WhenWithSend, Title: d.Title, Slug: slug, Excerpt: d.Excerpt, CoverUrl: d.CoverURL, CoverAlt: d.CoverAlt,
				BylineName: d.Byline.Name, Visibility: d.Visibility, Tiers: []uuid.UUID{}, CreatedBy: userID(p), At: s.now()}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.created", TargetType: "website_post", TargetID: id.String(),
				TargetName: d.Title, After: map[string]any{"kind": KindCampaign, "campaign": camp.Name, "slug": slug, "web_on": in.WebOn}}); err != nil {
				return err
			}
			cur, err = q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if cur, err = q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: cur.ID}); err != nil {
			return err
		}
		if cur.Status == PostPublished && !in.WebOn {
			return api.Conflict(tr(ctx, "post.publishedUseUnpublish", nil))
		}
		if (cur.Status == PostPublished || cur.Status == PostUnpublished) && !p.Can(authz.WebsiteEdit) {
			// A live post's settings are the website's.
			return api.Forbidden(authz.WebsiteEdit, s.helpers(ctx, ws, authz.WebsiteEdit))
		}
		return s.applySettings(ctx, q, ws, p, cur, in)
	})
	if err != nil {
		return WebsiteCampaignWeb{}, err
	}
	if post, err := s.Q.WebsitePostByCampaign(ctx, &campaignID); err == nil {
		s.changed(ctx, ws, p, post.ID)
	}
	return s.CampaignWeb(ctx, ws, p, campaignID)
}

// WebsiteCampaignWebState is a campaign's post in the Campaigns list's Web
// column and in Review & send.
type WebsiteCampaignWebState struct {
	CampaignID  string     `json:"campaignId"`
	PostID      string     `json:"postId,omitempty"`
	Chip        string     `json:"chip" enum:"off,with_send,scheduled,published,unpublished"`
	Slug        string     `json:"slug,omitempty"`
	URL         string     `json:"url,omitempty"`
	Visibility  string     `json:"visibility,omitempty"`
	Featured    bool       `json:"featured"`
	PublishAt   *time.Time `json:"publishAt,omitempty"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	// UnpublishedAt is when a published post was taken down.
	UnpublishedAt *time.Time `json:"unpublishedAt,omitempty"`
}

// CampaignWebStates are the Web column of campaigns: their posts' states
// (campaigns without one aren't listed).
func (s *Service) CampaignWebStates(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) ([]WebsiteCampaignWebState, error) {
	out := []WebsiteCampaignWebState{}
	if len(ids) == 0 {
		return out, nil
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	rows, err := s.Q.WebsitePostsByCampaigns(ctx, store.WebsitePostsByCampaignsParams{WorkspaceID: ws.ID, Campaigns: ids})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.CampaignID == nil {
			continue
		}
		st := WebsiteCampaignWebState{CampaignID: r.CampaignID.String(), PostID: r.ID.String(), Slug: r.Slug, URL: s.postURL(ws, r.Slug),
			Visibility: r.Visibility, Featured: isFeatured(c, r.ID), PublishAt: r.PublishAt, PublishedAt: r.PublishedAt, UnpublishedAt: r.UnpublishedAt}
		switch {
		case r.Status == PostPublished:
			st.Chip = "published"
		case r.Status == PostUnpublished:
			st.Chip = "unpublished"
		case !r.WebOn:
			st.Chip = "off"
		case r.PublishMode == WhenAt:
			st.Chip = "scheduled"
		default:
			st.Chip = "with_send"
		}
		out = append(out, st)
	}
	return out, nil
}
