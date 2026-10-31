package campaigns

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Review links (review 2026-10, CO-10): anyone with the link sees one
// version of one campaign, the email only, and can comment without signing
// in, until the link expires (7 days) or someone revokes it. Lookups are
// limited per client address, and comments per link.

const (
	// ReviewLinkDays is how long a review link works.
	ReviewLinkDays = 7
	// reviewCommentsPerHour caps the comments through one link.
	reviewCommentsPerHour = 30
)

// ComposerReviewLink is a link members share with reviewers.
type ComposerReviewLink struct {
	ID        string    `json:"id"`
	URL       string    `json:"url" doc:"Shown to the workspace's members only"`
	Version   int       `json:"version" doc:"The version reviewers see"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ComposerReviewLinkList is a campaign's working review links, newest first.
type ComposerReviewLinkList struct {
	Items []ComposerReviewLink `json:"items"`
}

// ReviewPage is what a reviewer sees: the email of one version and the
// comments written through the link. Nothing about the audience, settings
// or other campaigns.
type ReviewPage struct {
	Workspace string `json:"workspace"`
	Campaign  string `json:"campaign"`
	Version   int    `json:"version"`
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	HTML      string `json:"html"`
	// Sample is the sample reader the email is personalized for.
	Sample    string           `json:"sample"`
	FromName  string           `json:"fromName"`
	ExpiresAt time.Time        `json:"expiresAt"`
	Threads   []ComposerThread `json:"threads"`
}

// ReviewCommentInput is a reviewer's comment.
type ReviewCommentInput struct {
	Name string `json:"name" minLength:"1" maxLength:"80" doc:"How the reviewer signs their comment"`
	Body string `json:"body" minLength:"1" maxLength:"4000"`
	// ThreadID replies to a thread written through the same link.
	ThreadID string `json:"threadId,omitempty" format:"uuid"`
}

func reviewTokenAD(id uuid.UUID) string { return "review-link:" + id.String() }

func newReviewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) reviewURL(token string) string {
	return strings.TrimRight(s.URLs.Base(), "/") + "/review/" + token
}

// CreateReviewLink makes a review link for the campaign's current saved
// version (named "Shared for review" when it has no name, so it isn't
// pruned from the history).
func (s *Service) CreateReviewLink(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (ComposerReviewLink, error) {
	if s.Box == nil {
		return ComposerReviewLink{}, errors.New("campaigns: review links need a secrets box")
	}
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerReviewLink{}, api.MapError(err, "Campaign")
	}
	token, err := newReviewToken()
	if err != nil {
		return ComposerReviewLink{}, err
	}
	lid := uuid.Must(uuid.NewV7())
	sealed, err := s.Box.Seal([]byte(token), reviewTokenAD(lid))
	if err != nil {
		return ComposerReviewLink{}, err
	}
	now := s.now()
	expires := now.Add(ReviewLinkDays * 24 * time.Hour)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReviewLinkInsert(ctx, store.ReviewLinkInsertParams{ID: lid, WorkspaceID: ws.ID, CampaignID: id, Version: c.Version, TokenHash: authn.HashToken(token),
			TokenSealed: sealed, CreatedBy: &p.UserID, CreatedAt: now, ExpiresAt: expires}); err != nil {
			return err
		}
		// Keep the version reviewers see in the history.
		if _, err := q.ComposerNameVersionIfUnnamed(ctx, store.ComposerNameVersionIfUnnamedParams{CampaignID: id, Version: c.Version, Name: mVersionSharedForReview.Text(i18n.LangFrom(ctx), nil)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.review_link.create", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			After: map[string]any{"linkId": lid.String(), "version": c.Version, "expiresAt": expires}})
	})
	if err != nil {
		return ComposerReviewLink{}, err
	}
	return ComposerReviewLink{ID: lid.String(), URL: s.reviewURL(token), Version: int(c.Version), CreatedBy: p.Name, CreatedAt: now, ExpiresAt: expires}, nil
}

// ReviewLinks lists a campaign's working review links.
func (s *Service) ReviewLinks(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ComposerReviewLinkList, error) {
	rows, err := s.Q.ReviewLinksActive(ctx, store.ReviewLinksActiveParams{WorkspaceID: ws.ID, CampaignID: id, Now: s.now()})
	if err != nil {
		return ComposerReviewLinkList{}, err
	}
	out := ComposerReviewLinkList{Items: make([]ComposerReviewLink, 0, len(rows))}
	for _, r := range rows {
		token, err := s.Box.Open(r.TokenSealed, reviewTokenAD(r.ID))
		if err != nil {
			continue
		}
		out.Items = append(out.Items, ComposerReviewLink{ID: r.ID.String(), URL: s.reviewURL(string(token)), Version: int(r.Version), CreatedBy: r.CreatedByName, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt})
	}
	return out, nil
}

// RevokeReviewLink stops a review link from working.
func (s *Service) RevokeReviewLink(ctx context.Context, ws *store.Workspace, p *authz.Principal, id, linkID uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		n, err := q.ReviewLinkRevoke(ctx, store.ReviewLinkRevokeParams{WorkspaceID: ws.ID, CampaignID: id, ID: linkID, Now: &now, RevokedBy: &p.UserID})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.NotFound("Link")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.review_link.revoke", TargetType: "campaign", TargetID: id.String(),
			After: map[string]any{"linkId": linkID.String()}})
	})
}

// reviewLimiter limits review link lookups per client address: requests
// per minute, and far fewer unknown links, so guessing tokens is pointless.
var reviewLimiter = struct {
	sync.Mutex
	hits, miss map[string]*reviewWindow
}{hits: map[string]*reviewWindow{}, miss: map[string]*reviewWindow{}}

type reviewWindow struct {
	start time.Time
	n     int
}

const (
	reviewRequestsPerMinute = 60
	reviewMissesPer10Min    = 20
)

func reviewBump(m map[string]*reviewWindow, key string, now time.Time, size time.Duration) int {
	w := m[key]
	if w == nil || now.Sub(w.start) >= size {
		w = &reviewWindow{start: now}
		m[key] = w
	}
	w.n++
	return w.n
}

func reviewAllow(client string, now time.Time) bool {
	reviewLimiter.Lock()
	defer reviewLimiter.Unlock()
	if len(reviewLimiter.hits) > 10000 {
		reviewLimiter.hits, reviewLimiter.miss = map[string]*reviewWindow{}, map[string]*reviewWindow{}
	}
	if w := reviewLimiter.miss[client]; w != nil && now.Sub(w.start) < 10*time.Minute && w.n >= reviewMissesPer10Min {
		return false
	}
	return reviewBump(reviewLimiter.hits, client, now, time.Minute) <= reviewRequestsPerMinute
}

func reviewMiss(client string, now time.Time) {
	reviewLimiter.Lock()
	defer reviewLimiter.Unlock()
	reviewBump(reviewLimiter.miss, client, now, 10*time.Minute)
}

// reviewLink finds a working link by its token, limited per client.
func (s *Service) reviewLink(ctx context.Context, token, client string) (store.ReviewLinkByTokenRow, error) {
	l := i18n.LangFrom(ctx)
	now := s.now()
	if !reviewAllow(client, now) {
		return store.ReviewLinkByTokenRow{}, api.Err(http.StatusTooManyRequests, api.CodeRateLimited, mReviewLimited.Text(l, nil))
	}
	gone := api.Err(http.StatusNotFound, api.CodeNotFound, mReviewGone.Text(l, nil))
	token = strings.TrimSpace(token)
	if len(token) < 32 || len(token) > 64 {
		reviewMiss(client, now)
		return store.ReviewLinkByTokenRow{}, gone
	}
	link, err := s.Q.ReviewLinkByToken(ctx, authn.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) || err == nil && (link.RevokedAt != nil || !link.ExpiresAt.After(now)) {
		reviewMiss(client, now)
		return store.ReviewLinkByTokenRow{}, gone
	}
	return link, err
}

// Review is the page behind a review link.
func (s *Service) Review(ctx context.Context, token, client string) (ReviewPage, error) {
	link, err := s.reviewLink(ctx, token, client)
	if err != nil {
		return ReviewPage{}, err
	}
	v, err := s.Q.GetComposerVersion(ctx, store.GetComposerVersionParams{CampaignID: link.CampaignID, Version: link.Version})
	if errors.Is(err, pgx.ErrNoRows) {
		return ReviewPage{}, api.Err(http.StatusNotFound, api.CodeNotFound, mReviewGone.Text(i18n.LangFrom(ctx), nil))
	}
	if err != nil {
		return ReviewPage{}, err
	}
	// Personalized for the sample reader, as the composer's preview is:
	// reviewers never see a contact's data, nor raw Liquid.
	ws, err := s.Q.GetWorkspace(ctx, link.WorkspaceID)
	if err != nil {
		return ReviewPage{}, err
	}
	pv, err := s.Preview(ctx, &ws, nil, link.CampaignID, ComposerPreviewRequest{synthetic: true, Subject: &v.Subject, Preheader: &v.Preheader, HTML: &v.Html})
	if err != nil {
		return ReviewPage{}, err
	}
	rows, err := s.Q.CommentsByLink(ctx, &link.ID)
	if err != nil {
		return ReviewPage{}, err
	}
	list := make([]store.CommentsListRow, len(rows))
	for i, r := range rows {
		list[i] = store.CommentsListRow(r)
	}
	threads := threadsOf(list).Threads
	// Reviewers see members' names on their replies, nothing more.
	for i := range threads {
		threads[i].ResolvedBy = ""
		for j := range threads[i].Comments {
			threads[i].Comments[j].AuthorID, threads[i].Comments[j].Mentions = "", []string{}
		}
	}
	sample := ""
	if pv.Contact != nil {
		sample = pv.Contact.Name
	}
	return ReviewPage{Workspace: link.WorkspaceName, Campaign: link.CampaignName, Version: int(link.Version), Subject: pv.Subject, Preheader: pv.Preheader,
		HTML: pv.HTML, Sample: sample, FromName: pv.FromName, ExpiresAt: link.ExpiresAt, Threads: threads}, nil
}

// ReviewComment is a reviewer's comment through a link. The campaign's
// members see it in the composer's Comments, marked as a reviewer's.
func (s *Service) ReviewComment(ctx context.Context, token, client string, in ReviewCommentInput) (ComposerThread, error) {
	l := i18n.LangFrom(ctx)
	link, err := s.reviewLink(ctx, token, client)
	if err != nil {
		return ComposerThread{}, err
	}
	name, body := strings.TrimSpace(in.Name), strings.TrimSpace(in.Body)
	if name == "" {
		return ComposerThread{}, api.Invalid("body.name", mReviewName.Text(l, nil))
	}
	if body == "" {
		return ComposerThread{}, api.Invalid("body.body", mCommentEmpty.Text(l, nil))
	}
	now := s.now()
	if n, err := s.Q.ReviewLinkCommentCount(ctx, store.ReviewLinkCommentCountParams{ReviewLinkID: &link.ID, Since: now.Add(-time.Hour)}); err != nil {
		return ComposerThread{}, err
	} else if n >= reviewCommentsPerHour {
		return ComposerThread{}, api.Err(http.StatusTooManyRequests, api.CodeRateLimited, mReviewLimited.Text(l, nil))
	}
	var threadID *uuid.UUID
	if in.ThreadID != "" {
		tid, err := uuid.Parse(in.ThreadID)
		if err != nil {
			return ComposerThread{}, api.NotFound("Comment")
		}
		// Only threads written through this link.
		rows, err := s.Q.CommentsByLink(ctx, &link.ID)
		if err != nil {
			return ComposerThread{}, err
		}
		found := false
		for _, r := range rows {
			if r.ID == tid && r.ThreadID == nil {
				found = true
			}
		}
		if !found {
			return ComposerThread{}, api.NotFound("Comment")
		}
		threadID = &tid
	}
	cid := uuid.Must(uuid.NewV7())
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CommentInsert(ctx, store.CommentInsertParams{ID: cid, WorkspaceID: link.WorkspaceID, CampaignID: link.CampaignID, ThreadID: threadID, Body: body,
			ReviewerName: name, ReviewLinkID: &link.ID, Version: link.Version, Mentions: []uuid.UUID{}, CreatedAt: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &link.WorkspaceID, Action: "campaign.comment", TargetType: "campaign", TargetID: link.CampaignID.String(),
			TargetName: link.CampaignName, After: map[string]any{"commentId": cid.String(), "reviewer": name, "linkId": link.ID.String()}})
	})
	if err != nil {
		return ComposerThread{}, err
	}
	page, err := s.Review(ctx, token, client)
	if err != nil {
		return ComposerThread{}, err
	}
	root := cid.String()
	if threadID != nil {
		root = threadID.String()
	}
	for _, t := range page.Threads {
		if t.ID == root {
			return t, nil
		}
	}
	return ComposerThread{}, api.NotFound("Comment")
}
