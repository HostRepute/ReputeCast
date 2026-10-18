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
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// WebsitePastCampaign is a sent campaign that isn't a post yet ("Publish
// past campaigns").
type WebsitePastCampaign struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Subject    string    `json:"subject"`
	Lists      []string  `json:"lists"`
	SentAt     time.Time `json:"sentAt"`
	Recipients int64     `json:"recipients"`
	// Note says why it isn't on the web yet: left out of the archive by
	// hand, or its list isn't public ("" when it simply wasn't archived).
	Note string `json:"note" enum:"archive_hidden,list_private,"`
	// Visibility is what "Follow the audience" gives it.
	Visibility string `json:"visibility" enum:"public,members,paid"`
}

// PastCampaigns lists the sent campaigns that aren't posts yet, newest first.
func (s *Service) PastCampaigns(ctx context.Context, ws *store.Workspace) ([]WebsitePastCampaign, error) {
	rows, err := s.Q.WebsiteSentWithoutPost(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	public := map[uuid.UUID]bool{}
	if pl, err := s.Audience.PublicLists(ctx, s.Q, ws); err == nil {
		for _, l := range pl {
			public[l.ID] = true
		}
	}
	names := s.listNames(ctx, ws)
	out := []WebsitePastCampaign{}
	for _, r := range rows {
		lists := audienceLists(r.Audience)
		pc := WebsitePastCampaign{ID: r.ID.String(), Name: r.Name, Subject: forms.RenderTitle(r.Subject), Lists: []string{}, SentAt: r.SentAt,
			Recipients: r.Recipients, Visibility: gate.Public}
		anyPublic := false
		for _, l := range lists {
			if n, ok := names[l]; ok {
				pc.Lists = append(pc.Lists, n)
			}
			anyPublic = anyPublic || public[l]
		}
		switch {
		case r.ArchiveHidden:
			pc.Note = "archive_hidden"
		case len(lists) > 0 && !anyPublic:
			pc.Note = "list_private"
		}
		if paidOnly(r.Audience) {
			pc.Visibility = gate.Paid
		}
		out = append(out, pc)
	}
	return out, nil
}

// listNames are the workspace's lists by id.
func (s *Service) listNames(ctx context.Context, ws *store.Workspace) map[uuid.UUID]string {
	out := map[uuid.UUID]string{}
	rows, err := s.Pool.Query(ctx, `SELECT id, name FROM lists WHERE workspace_id = $1`, ws.ID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var n string
		if rows.Scan(&id, &n) == nil {
			out[id] = n
		}
	}
	return out
}

// audienceLists are the lists of a campaign's audience.
func audienceLists(raw json.RawMessage) []uuid.UUID {
	var a struct {
		Lists []uuid.UUID `json:"lists"`
	}
	_ = json.Unmarshal(raw, &a)
	return a.Lists
}

// WebsitePublishPastInput publishes sent campaigns as posts.
type WebsitePublishPastInput struct {
	Campaigns []string `json:"campaigns" minItems:"1" maxItems:"100"`
	// Visibility: audience follows each campaign's audience (public, or
	// paid for a paid-only send).
	Visibility string `json:"visibility" enum:"audience,public,members"`
}

// WebsitePublishPastResult is how many posts went live.
type WebsitePublishPastResult struct {
	Published int `json:"published"`
}

// PublishPast makes posts of sent campaigns, live at their send dates, in
// the site's theme, with the tags of their lists.
func (s *Service) PublishPast(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebsitePublishPastInput) (WebsitePublishPastResult, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsitePublishPastResult{}, err
	}
	if !c.On() {
		return WebsitePublishPastResult{}, api.Conflict(tr(ctx, "site.off", nil))
	}
	rows, err := s.Q.WebsiteSentWithoutPost(ctx, ws.ID)
	if err != nil {
		return WebsitePublishPastResult{}, err
	}
	want := map[uuid.UUID]bool{}
	for _, id := range in.Campaigns {
		if u, err := uuid.Parse(id); err == nil {
			want[u] = true
		}
	}
	out := WebsitePublishPastResult{}
	for _, r := range rows {
		if !want[r.ID] {
			continue
		}
		vis := in.Visibility
		if vis == "" || vis == "audience" {
			vis = gate.Public
			if paidOnly(r.Audience) {
				vis = gate.Paid
			}
		}
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			return s.publishPast(ctx, tx, ws, p, r, vis)
		})
		if err != nil {
			return out, err
		}
		out.Published++
	}
	return out, nil
}

func (s *Service) publishPast(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, r store.WebsiteSentWithoutPostRow, vis string) error {
	q := store.New(tx)
	var mjmlSrc string
	_ = tx.QueryRow(ctx, `SELECT mjml FROM campaigns WHERE id = $1`, r.ID).Scan(&mjmlSrc)
	title := forms.RenderTitle(r.Subject)
	if title == "" {
		title = r.Name
	}
	excerpt := forms.RenderTitle(r.Preheader)
	if len([]rune(excerpt)) > 200 {
		excerpt = forms.Excerpt(excerpt, 195)
	}
	cover, alt := firstImage(mjmlSrc)
	cur, err := q.WebsitePostByCampaign(ctx, &r.ID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		id := newID()
		slug := ""
		if ai, err := q.ArchiveIssueGet(ctx, store.ArchiveIssueGetParams{WorkspaceID: ws.ID, CampaignID: r.ID}); err == nil && slugRe.MatchString(ai.Slug) {
			if taken, err := q.WebsitePostSlugTaken(ctx, store.WebsitePostSlugTakenParams{WorkspaceID: ws.ID, Slug: ai.Slug, ID: id}); err == nil && !taken {
				slug = ai.Slug
			}
		}
		if slug == "" {
			slug = s.freeSlug(ctx, ws, id, title)
		}
		if err := q.WebsitePostInsert(ctx, store.WebsitePostInsertParams{ID: id, WorkspaceID: ws.ID, Kind: KindCampaign, CampaignID: &r.ID, WebOn: true,
			Status: PostDraft, PublishMode: WhenNow, Title: title, Slug: slug, Excerpt: excerpt, CoverUrl: cover, CoverAlt: alt, Visibility: vis,
			Tiers: []uuid.UUID{}, CreatedBy: userID(p), At: s.now()}); err != nil {
			return err
		}
		for i, t := range s.listTags(ctx, ws, audienceLists(r.Audience)) {
			tid, _ := uuid.Parse(t.ID)
			if err := q.WebsitePostAddTag(ctx, store.WebsitePostAddTagParams{PostID: id, TagID: tid, Position: int16(i)}); err != nil {
				return err
			}
		}
		if cur, err = q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if cur, err = q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: cur.ID}); err != nil {
			return err
		}
		if cur.Status == PostPublished {
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE website_posts SET web_on = true, publish_mode = 'now', visibility = $2, updated_at = $3 WHERE id = $1`,
			cur.ID, vis, s.now()); err != nil {
			return err
		}
		cur.WebOn, cur.PublishMode, cur.Visibility = true, WhenNow, vis
	}
	d, err := s.decision(ctx, ws, p, cur)
	if err != nil {
		return err
	}
	if d.NeedsApproval {
		return approvalProblem(ctx, d)
	}
	if !d.Allowed {
		return api.Forbidden(authz.WebsitePublish, s.helpers(ctx, ws, authz.WebsitePublish))
	}
	return s.publishNow(ctx, tx, ws, cur, userID(p), map[string]any{"name": p.Name, "source": "publish past campaigns", "at": r.SentAt,
		"by": p.UserID.String()}, d.Approved)
}

// WebsiteSentAsCampaign is the campaign draft made from a post.
type WebsiteSentAsCampaign struct {
	CampaignID string `json:"campaignId"`
}

// SendAsCampaign makes a campaign draft of a web-only post: its title as
// the subject, its excerpt as the preheader and its blocks as the email.
// The audience is left to pick in the composer.
func (s *Service) SendAsCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (WebsiteSentAsCampaign, error) {
	if s.Campaigns == nil || s.Campaigns() == nil {
		return WebsiteSentAsCampaign{}, errors.New("campaign drafts aren't wired")
	}
	pr, err := s.getPost(ctx, s.Q, ws, id)
	if err != nil {
		return WebsiteSentAsCampaign{}, err
	}
	post := pr.WebsitePost
	if post.Kind != KindWeb {
		return WebsiteSentAsCampaign{}, api.Conflict(tr(ctx, "post.notWebOnly", nil))
	}
	if post.SentAsCampaign != nil {
		return WebsiteSentAsCampaign{}, api.Conflict(tr(ctx, "post.sentAsCampaign", nil))
	}
	src := strings.ReplaceAll(post.Mjml, render.PreviewComment, "")
	cid, err := s.Campaigns().DraftFromPost(ctx, ws, p, post.Title, post.Title, post.Excerpt, src)
	if err != nil {
		return WebsiteSentAsCampaign{}, err
	}
	if err := s.Q.WebsitePostSetSentAs(ctx, store.WebsitePostSetSentAsParams{WorkspaceID: ws.ID, ID: post.ID, CampaignID: &cid, At: s.now()}); err != nil {
		return WebsiteSentAsCampaign{}, err
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "post.sent_as_campaign", TargetType: "website_post", TargetID: post.ID.String(),
		TargetName: post.Title, After: map[string]any{"campaign_id": cid.String()}}); err != nil {
		return WebsiteSentAsCampaign{}, err
	}
	return WebsiteSentAsCampaign{CampaignID: cid.String()}, nil
}

// syncIssues makes posts of campaigns sent to archived lists since the
// website went on, live at their send, as the archive filed them. A
// campaign whose Web tab was saved has a post already and goes its own way.
func (s *Service) syncIssues(ctx context.Context) error {
	ids, err := s.Q.WebsitesOn(ctx)
	if err != nil {
		return err
	}
	for _, wsID := range ids {
		ws, err := s.Q.GetWorkspace(ctx, wsID)
		if err != nil {
			continue
		}
		set, err := s.Forms.LoadSettings(ctx, &ws)
		if err != nil {
			continue
		}
		lists, err := s.Forms.ArchiveLists(ctx, s.Q, &ws, set)
		if err != nil || len(lists) == 0 {
			continue
		}
		issues, err := s.Q.WebsiteNewIssues(ctx, store.WebsiteNewIssuesParams{WorkspaceID: ws.ID, Lists: lists})
		if err != nil {
			return err
		}
		tagOf := map[uuid.UUID]uuid.UUID{}
		for _, is := range issues {
			if is.ListID != nil && !slices.Contains(lists, *is.ListID) {
				continue
			}
			if is.ListID != nil {
				if _, ok := tagOf[*is.ListID]; !ok {
					if t, err := s.Q.WebsiteTagByList(ctx, store.WebsiteTagByListParams{WorkspaceID: ws.ID, ListID: is.ListID}); err == nil {
						tagOf[*is.ListID] = t.ID
					}
				}
			}
			row := store.WebsiteArchiveCampaignsRow(is)
			if err := s.postForIssue(ctx, &ws, row, tagOf); err != nil {
				s.log().WarnContext(ctx, "post for a new issue", "campaign", is.CampaignID, "err", err)
			}
		}
	}
	return nil
}
