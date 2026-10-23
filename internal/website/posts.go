package website

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// Post kinds.
const (
	KindCampaign = "campaign"
	KindWeb      = "web"
)

// Post statuses.
const (
	PostDraft       = "draft"
	PostScheduled   = "scheduled"
	PostPublished   = "published"
	PostUnpublished = "unpublished"
)

// Publish modes.
const (
	WhenWithSend = "with_send"
	WhenAt       = "at"
	WhenNow      = "now"
)

// WebsiteTagRef is a tag of a post.
type WebsiteTagRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// WebsiteTierRef is a paid tier a post is for.
type WebsiteTierRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Label is the tier's quiet label on posts ("Insider").
	Label string `json:"label"`
}

// WebsitePostCampaign is the campaign a post comes from.
type WebsitePostCampaign struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
}

// WebsiteByline is who a post is by: a member of the workspace or a written name.
type WebsiteByline struct {
	UserID string `json:"userId,omitempty"`
	Name   string `json:"name"`
	Color  string `json:"color,omitempty"`
}

// WebsitePostApproval is a post's open approval request.
type WebsitePostApproval struct {
	ID          string    `json:"id"`
	Status      string    `json:"status" enum:"pending,approved,changes_requested,rejected"`
	RequestedBy string    `json:"requestedBy"`
	RequestedAt time.Time `json:"requestedAt"`
	Approvers   []string  `json:"approvers"`
	Rule        string    `json:"rule"`
	Note        string    `json:"note,omitempty"`
}

// WebsitePostRow is a post in Website › Posts.
type WebsitePostRow struct {
	ID      string `json:"id"`
	Kind    string `json:"kind" enum:"campaign,web"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
	Slug    string `json:"slug"`
	// URL is the post's address on the site.
	URL string `json:"url"`
	// Status is draft, scheduled (a time, or with a scheduled send),
	// published or unpublished.
	Status          string               `json:"status" enum:"draft,scheduled,published,unpublished"`
	PublishMode     string               `json:"publishMode" enum:"with_send,at,now"`
	PublishAt       *time.Time           `json:"publishAt,omitempty"`
	PublishTZ       string               `json:"publishTz,omitempty"`
	PublishedAt     *time.Time           `json:"publishedAt,omitempty"`
	UnpublishedAt   *time.Time           `json:"unpublishedAt,omitempty"`
	UnpublishedBy   string               `json:"unpublishedBy,omitempty"`
	UnpublishReason string               `json:"unpublishReason,omitempty"`
	RedirectTo      string               `json:"redirectTo,omitempty"`
	Campaign        *WebsitePostCampaign `json:"campaign,omitempty"`
	Byline          WebsiteByline        `json:"byline"`
	Visibility      string               `json:"visibility" enum:"public,members,paid"`
	Tiers           []WebsiteTierRef     `json:"tiers"`
	NeverFree       bool                 `json:"neverFree"`
	HasMarker       bool                 `json:"hasMarker"`
	MarkerAfter     int                  `json:"markerAfter"`
	Featured        bool                 `json:"featured"`
	Tags            []WebsiteTagRef      `json:"tags"`
	// Views, Signups and Upgrades are the last 30 days; ViewsAllTime is
	// every view (an unpublished post's views before it was taken down).
	Views        int64                `json:"views"`
	ViewsAllTime int64                `json:"viewsAllTime"`
	Signups      int64                `json:"signups"`
	Upgrades     int64                `json:"upgrades"`
	Approval     *WebsitePostApproval `json:"approval,omitempty"`
	CreatedBy    string               `json:"createdBy"`
	UpdatedAt    time.Time            `json:"updatedAt"`
}

// WebsitePostCounts are the posts by status.
type WebsitePostCounts struct {
	All           int64 `json:"all"`
	Published     int64 `json:"published"`
	FromCampaigns int64 `json:"fromCampaigns"`
	WebOnly       int64 `json:"webOnly"`
	Scheduled     int64 `json:"scheduled"`
	Draft         int64 `json:"draft"`
	Unpublished   int64 `json:"unpublished"`
	MembersOnly   int64 `json:"membersOnly"`
	Paid          int64 `json:"paid"`
}

// WebsitePostList is a page of Website › Posts.
type WebsitePostList struct {
	Posts    []WebsitePostRow  `json:"posts"`
	Total    int64             `json:"total"`
	Counts   WebsitePostCounts `json:"counts"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	// From and Until are the 30 days of the numbers.
	From  time.Time `json:"from"`
	Until time.Time `json:"until"`
}

// WebsitePostFilter narrows Website › Posts.
type WebsitePostFilter struct {
	Status     string `query:"status" enum:"all,published,scheduled,draft,unpublished," doc:"A status (all by default)"`
	Visibility string `query:"visibility" enum:"public,members,paid,"`
	Tag        string `query:"tag" doc:"A tag's id"`
	Source     string `query:"source" enum:"campaign,web,"`
	Search     string `query:"q" maxLength:"200" doc:"Search by title"`
	Page       int    `query:"page" minimum:"0" maximum:"10000"`
	PageSize   int    `query:"pageSize" minimum:"0" maximum:"100"`
}

// statusExpr is a post's status as Website › Posts shows it: a campaign's
// post set to go live with a scheduled send is scheduled.
const statusExpr = `CASE WHEN p.status IN ('published','unpublished','scheduled') THEN p.status
	WHEN p.kind = 'campaign' AND p.web_on AND p.publish_mode = 'with_send' AND c.status IN ('scheduled','approved') THEN 'scheduled'
	ELSE 'draft' END`

// listed are the posts Website › Posts shows: web-only posts, and posts of
// campaigns that are set to go on the web or went there.
const listed = `p.workspace_id = $1 AND p.deleted_at IS NULL AND (p.kind = 'web' OR p.web_on OR p.status IN ('published','unpublished'))`

// Posts lists the posts of Website › Posts with their 30-day numbers.
func (s *Service) Posts(ctx context.Context, ws *store.Workspace, p *authz.Principal, f WebsitePostFilter) (WebsitePostList, error) {
	if f.PageSize <= 0 {
		f.PageSize = 25
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	now := s.now()
	loc := location(ws)
	out := WebsitePostList{Posts: []WebsitePostRow{}, Page: f.Page, PageSize: f.PageSize}
	out.Until = day(now, loc)
	out.From = out.Until.AddDate(0, 0, -29)
	counts, err := s.counts(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Counts = counts
	where := []string{listed}
	args := []any{ws.ID}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}
	if f.Status != "" && f.Status != "all" {
		add("("+statusExpr+") = ?", f.Status)
	}
	if f.Visibility != "" {
		add("p.visibility = ?", f.Visibility)
	}
	if f.Source != "" {
		add("p.kind = ?", f.Source)
	}
	if f.Tag != "" {
		id, err := uuid.Parse(f.Tag)
		if err != nil {
			return out, api.Invalid("tag", tr(ctx, "tag.notFound", nil))
		}
		add("EXISTS (SELECT 1 FROM website_post_tags t WHERE t.post_id = p.id AND t.tag_id = ?)", id)
	}
	if q := strings.TrimSpace(f.Search); q != "" {
		add("p.title ILIKE ?", "%"+strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q)+"%")
	}
	cond := strings.Join(where, " AND ")
	from := " FROM website_posts p LEFT JOIN campaigns c ON c.id = p.campaign_id WHERE " + cond
	if err := s.Pool.QueryRow(ctx, "SELECT count(*)"+from, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	// Newest first: scheduled and draft posts by when they'll go out, the
	// rest by when they went out.
	rows, err := s.Pool.Query(ctx, `SELECT p.id`+from+`
		ORDER BY coalesce(p.published_at, p.unpublished_at, p.publish_at, c.scheduled_at, p.updated_at) DESC, p.id DESC`+
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return out, err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return out, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	posts, err := s.rows(ctx, ws, ids, out.From, out.Until)
	if err != nil {
		return out, err
	}
	out.Posts = posts
	return out, nil
}

// counts are the posts by status, for the filters and the header.
func (s *Service) counts(ctx context.Context, ws *store.Workspace) (WebsitePostCounts, error) {
	var out WebsitePostCounts
	rows, err := s.Q.WebsitePostCounts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		listedRow := r.Kind == KindWeb || r.WebOn || r.Status == PostPublished || r.Status == PostUnpublished
		if !listedRow {
			continue
		}
		status := r.Status
		if status == PostDraft && r.Kind == KindCampaign && r.WebOn && r.PublishMode == WhenWithSend && (r.CampaignStatus == "scheduled" || r.CampaignStatus == "approved") {
			status = PostScheduled
		}
		out.All += r.N
		switch status {
		case PostPublished:
			out.Published += r.N
			if r.Kind == KindCampaign {
				out.FromCampaigns += r.N
			} else {
				out.WebOnly += r.N
			}
			switch r.Visibility {
			case gate.Members:
				out.MembersOnly += r.N
			case gate.Paid:
				out.Paid += r.N
			}
		case PostScheduled:
			out.Scheduled += r.N
		case PostUnpublished:
			out.Unpublished += r.N
		default:
			out.Draft += r.N
		}
	}
	return out, nil
}

// postRow is a post with what its views need.
type postRow struct {
	store.WebsitePostGetRow
	Tags []WebsiteTagRef
}

// getPost reads one post of a workspace.
func (s *Service) getPost(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (postRow, error) {
	r, err := q.WebsitePostGet(ctx, store.WebsitePostGetParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return postRow{}, notFound(ctx, "post")
	}
	if err != nil {
		return postRow{}, err
	}
	pr := postRow{WebsitePostGetRow: r}
	tags, err := q.WebsitePostTags(ctx, []uuid.UUID{id})
	if err != nil {
		return pr, err
	}
	pr.Tags = []WebsiteTagRef{}
	for _, t := range tags {
		pr.Tags = append(pr.Tags, WebsiteTagRef{ID: t.ID.String(), Name: t.Name, Slug: t.Slug})
	}
	return pr, nil
}

// rows builds the list rows of posts, in the order of ids, with their numbers.
func (s *Service) rows(ctx context.Context, ws *store.Workspace, ids []uuid.UUID, from, until time.Time) ([]WebsitePostRow, error) {
	out := []WebsitePostRow{}
	if len(ids) == 0 {
		return out, nil
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	tiers, err := s.tiers(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	views, err := s.Q.WebsitePostViewCounts(ctx, store.WebsitePostViewCountsParams{WorkspaceID: ws.ID, Posts: ids, FromDay: from, UntilDay: until})
	if err != nil {
		return nil, err
	}
	allViews, err := s.Q.WebsitePostViewCounts(ctx, store.WebsitePostViewCountsParams{WorkspaceID: ws.ID, Posts: ids, FromDay: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), UntilDay: until})
	if err != nil {
		return nil, err
	}
	conv, err := s.Q.WebsiteConversionCounts(ctx, store.WebsiteConversionCountsParams{WorkspaceID: ws.ID, Posts: ids, FromDay: from, UntilDay: until.AddDate(0, 0, 1)})
	if err != nil {
		return nil, err
	}
	v30, vAll := map[uuid.UUID]int64{}, map[uuid.UUID]int64{}
	for _, v := range views {
		v30[v.PostID] = v.Views
	}
	for _, v := range allViews {
		vAll[v.PostID] = v.Views
	}
	signups, upgrades := map[uuid.UUID]int64{}, map[uuid.UUID]int64{}
	for _, r := range conv {
		if r.Kind == "signup" {
			signups[r.PostID] = r.N
		} else {
			upgrades[r.PostID] = r.N
		}
	}
	approvals := s.openApprovals(ctx, ws, ids)
	for _, id := range ids {
		pr, err := s.getPost(ctx, s.Q, ws, id)
		if err != nil {
			continue
		}
		row := s.row(ws, c, tiers, pr)
		row.Views, row.ViewsAllTime, row.Signups, row.Upgrades = v30[id], vAll[id], signups[id], upgrades[id]
		row.Approval = approvals[id]
		out = append(out, row)
	}
	return out, nil
}

// row is a post as Website › Posts lists it.
func (s *Service) row(ws *store.Workspace, c *config, tiers []gate.Tier, pr postRow) WebsitePostRow {
	p := pr.WebsitePost
	r := WebsitePostRow{ID: p.ID.String(), Kind: p.Kind, Title: p.Title, Excerpt: p.Excerpt, Slug: p.Slug, URL: s.postURL(ws, p.Slug),
		Status: displayStatus(p, pr.CampaignStatus), PublishMode: p.PublishMode, PublishAt: p.PublishAt, PublishTZ: p.PublishTz,
		PublishedAt: p.PublishedAt, UnpublishedAt: p.UnpublishedAt, UnpublishedBy: pr.UnpublishedByName, UnpublishReason: p.UnpublishReason,
		RedirectTo: p.RedirectTo, Visibility: p.Visibility, Tiers: tierRefs(p.Tiers, tiers), NeverFree: p.NeverFree,
		HasMarker: p.HasMarker, MarkerAfter: int(p.MarkerAfter), Featured: isFeatured(c, p.ID), Tags: pr.Tags, CreatedBy: pr.CreatedByName,
		UpdatedAt: p.UpdatedAt}
	if p.Kind == KindWeb {
		// A web-only post's marker is in its own content; drafts too.
		if res, err := render.Render(p.Mjml, render.Options{}); err == nil {
			r.HasMarker, r.MarkerAfter = res.HasMarker, res.MarkerAfter
		}
	}
	r.Byline = WebsiteByline{Name: p.BylineName, Color: pr.BylineColor}
	if p.BylineUserID != nil {
		r.Byline.UserID = p.BylineUserID.String()
		if r.Byline.Name == "" {
			r.Byline.Name = pr.BylineUserName
		}
	}
	if p.CampaignID != nil {
		r.Campaign = &WebsitePostCampaign{ID: p.CampaignID.String(), Name: pr.CampaignName, Status: pr.CampaignStatus, SentAt: pr.CampaignStartedAt,
			ScheduledAt: pr.CampaignScheduledAt}
	}
	return r
}

// displayStatus is a post's status as the app shows it.
func displayStatus(p store.WebsitePost, campaignStatus string) string {
	switch p.Status {
	case PostPublished, PostUnpublished, PostScheduled:
		return p.Status
	}
	if p.Kind == KindCampaign && p.WebOn && p.PublishMode == WhenWithSend && (campaignStatus == "scheduled" || campaignStatus == "approved") {
		return PostScheduled
	}
	return PostDraft
}

func isFeatured(c *config, id uuid.UUID) bool {
	for _, f := range c.row.Featured {
		if f == id {
			return true
		}
	}
	return false
}

// tierRefs names a post's tiers.
func tierRefs(ids []uuid.UUID, tiers []gate.Tier) []WebsiteTierRef {
	out := []WebsiteTierRef{}
	for _, id := range ids {
		for _, t := range tiers {
			if t.ID == id {
				out = append(out, WebsiteTierRef{ID: id.String(), Name: t.Name, Label: t.Label})
			}
		}
	}
	return out
}

// postURL is a post's address on the site (its canonical one).
func (s *Service) postURL(ws *store.Workspace, slug string) string {
	return s.sitePath(context.Background(), ws, "posts", slug)
}

// sitePath is an address on the site: the custom domain once verified,
// the instance's /p/<workspace> until then.
func (s *Service) sitePath(ctx context.Context, ws *store.Workspace, parts ...string) string {
	if s.Forms == nil {
		return ""
	}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return ""
	}
	l := s.Forms.Links(ws, set)
	if len(parts) == 0 {
		if l.Custom {
			return l.Origin() + "/"
		}
		return l.Path() + "/"
	}
	return l.Path(parts...)
}

// openApprovals are the pending approval requests of posts.
func (s *Service) openApprovals(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) map[uuid.UUID]*WebsitePostApproval {
	out := map[uuid.UUID]*WebsitePostApproval{}
	rows, err := s.Pool.Query(ctx, `SELECT r.id, r.website_post_id, r.status, r.requested_at, coalesce(u.name, ''), r.matched_rules, r.note
		FROM approval_requests r LEFT JOIN users u ON u.id = r.requested_by
		WHERE r.workspace_id = $1 AND r.website_post_id = ANY ($2) AND r.status = 'pending'`, ws.ID, ids)
	if err != nil {
		s.log().WarnContext(ctx, "post approvals", "err", err)
		return out
	}
	defer rows.Close()
	approvers := s.approverNames(ctx, ws)
	for rows.Next() {
		var a WebsitePostApproval
		var id uuid.UUID
		var post uuid.UUID
		var rules []string
		if err := rows.Scan(&id, &post, &a.Status, &a.RequestedAt, &a.RequestedBy, &rules, &a.Note); err != nil {
			return out
		}
		a.ID = id.String()
		a.Approvers = approvers
		if len(rules) > 0 {
			a.Rule = rules[0]
		}
		aa := a
		out[post] = &aa
	}
	return out
}

// approverNames are the workspace's approvers (campaigns:approve).
func (s *Service) approverNames(ctx context.Context, ws *store.Workspace) []string {
	rows, err := s.Q.ApprovalMembers(ctx, ws.ID)
	if err != nil {
		return []string{}
	}
	out := []string{}
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(authz.CampaignsApprove) {
			out = append(out, r.Name)
		}
	}
	return out
}

// renderNotes are what the renderer found in a post's content: the blocks
// left out on the web and the tokens that show their fallback.
type renderNotes struct {
	Hidden    []render.Hidden   `json:"hidden"`
	Fallbacks []render.Fallback `json:"fallbacks"`
}

// FindPost finds a post by its id or its address (the slug after /posts/).
func (s *Service) FindPost(ctx context.Context, ws *store.Workspace, ref string) (uuid.UUID, error) {
	ref = strings.Trim(strings.TrimSpace(ref), "/")
	ref = strings.TrimPrefix(ref, "posts/")
	if id, err := uuid.Parse(ref); err == nil {
		if _, err := s.getPost(ctx, s.Q, ws, id); err != nil {
			return uuid.Nil, err
		}
		return id, nil
	}
	p, err := s.Q.WebsitePostBySlug(ctx, store.WebsitePostBySlugParams{WorkspaceID: ws.ID, Slug: ref})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("no post with id or address %q in this workspace", ref)
	}
	if err != nil {
		return uuid.Nil, err
	}
	return p.ID, nil
}
