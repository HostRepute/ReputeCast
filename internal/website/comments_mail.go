package website

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// Reply emails (M61): when a reply is published under a member's comment
// and they want reply emails, a notice waits; every 5 minutes the job
// sends each member at most one email an hour, in their language, about
// the newest reply (and how many more there are). "Stop these emails"
// opens /comments/mute/{token}, a page with one button.

// CommentMute is the token of a reply email's "Stop these emails".
const CommentMute linktoken.Kind = 231

// commentNoticeEvery is how often a member may get a reply email.
const commentNoticeEvery = time.Hour

// CommentNoticesArgs is the job that sends reply emails.
type CommentNoticesArgs struct{}

// Kind implements river.JobArgs.
func (CommentNoticesArgs) Kind() string { return "website.comment_notices" }

// InsertOpts runs once at a time.
func (CommentNoticesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
}

type commentNoticesWorker struct {
	river.WorkerDefaults[CommentNoticesArgs]
	s *Service
}

func (w *commentNoticesWorker) Work(ctx context.Context, _ *river.Job[CommentNoticesArgs]) error {
	_, err := w.s.SendCommentNotices(ctx)
	return err
}

func (w *commentNoticesWorker) Timeout(*river.Job[CommentNoticesArgs]) time.Duration {
	return 5 * time.Minute
}

// noticeGroup is one member's waiting notices in one workspace.
type noticeGroup struct {
	ws, contact uuid.UUID
	comments    []uuid.UUID
}

// SendCommentNotices sends the reply emails that are due: one per member
// who got none in the last hour. It returns how many went.
func (s *Service) SendCommentNotices(ctx context.Context) (int, error) {
	now := s.now()
	since := now.Add(-commentNoticeEvery)
	rows, err := s.Q.WebsiteCommentNoticesDue(ctx, &since)
	if err != nil {
		return 0, err
	}
	var groups []noticeGroup
	for _, r := range rows {
		if n := len(groups); n > 0 && groups[n-1].ws == r.WorkspaceID && groups[n-1].contact == r.ContactID {
			groups[n-1].comments = append(groups[n-1].comments, r.CommentID)
			continue
		}
		groups = append(groups, noticeGroup{ws: r.WorkspaceID, contact: r.ContactID, comments: []uuid.UUID{r.CommentID}})
	}
	sent := 0
	for _, g := range groups {
		ok, err := s.sendNotice(ctx, g, now)
		if err != nil {
			s.log().WarnContext(ctx, "comment reply email", "workspace", g.ws, "err", err)
			continue
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

// sendNotice sends one member the email about their newest reply; notices
// whose reply left the site, or a member who turned the emails off, are
// dropped.
func (s *Service) sendNotice(ctx context.Context, g noticeGroup, now time.Time) (bool, error) {
	drop := func(ids []uuid.UUID) error {
		if len(ids) == 0 {
			return nil
		}
		return s.Q.WebsiteCommentNoticesDrop(ctx, store.WebsiteCommentNoticesDropParams{ContactID: g.contact, Comments: ids})
	}
	to, err := s.Q.WebsiteCommentRecipient(ctx, g.contact)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, drop(g.comments)
	}
	if err != nil {
		return false, err
	}
	if to.ErasedAt != nil || !to.CommentEmails || to.Email == "" {
		return false, drop(g.comments)
	}
	ws, err := s.Q.GetWorkspace(ctx, g.ws)
	if err != nil {
		return false, err
	}
	// A bounced or complaining address gets nothing (an unsubscribe from
	// the newsletters doesn't stop reply emails the member asked for).
	if sup, err := s.Q.SignupSuppression(ctx, store.SignupSuppressionParams{WorkspaceID: ws.ID, Email: to.Email}); err == nil && sup.Reason != "unsubscribe" {
		return false, drop(g.comments)
	}
	var live []store.WebsiteCommentNoticeReplyRow
	var gone []uuid.UUID
	for _, id := range g.comments {
		r, err := s.Q.WebsiteCommentNoticeReply(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			gone = append(gone, id)
			continue
		}
		if err != nil {
			return false, err
		}
		if r.Status != CommentPublished || r.ParentStatus != CommentPublished || r.PostStatus != PostPublished || !r.PostLive || (!r.Team && r.ContactID == nil) {
			gone = append(gone, id)
			continue
		}
		live = append(live, r)
	}
	if err := drop(gone); err != nil {
		return false, err
	}
	if len(live) == 0 {
		return false, nil
	}
	c, err := s.load(ctx, s.Q, &ws)
	if err != nil {
		return false, err
	}
	set, err := s.Forms.LoadSettings(ctx, &ws)
	if err != nil {
		return false, err
	}
	if !c.On() {
		return false, nil
	}
	reply := live[len(live)-1]
	loc, _ := i18n.ParseLocale(to.Language)
	l := i18n.For(loc)
	site := siteTitle(c, set)
	name := site
	if !reply.Team {
		if name = CommenterName(reply.DisplayName, reply.FirstName, reply.LastName); name == "" {
			name = WordsIn(l).Text("comments.reader", nil)
		}
	}
	link := s.sitePath(ctx, &ws, "posts", reply.PostSlug)
	if page := s.threadPage(ctx, &ws, reply.ParentPost, reply.ParentID, nil); page > 1 {
		link += "?comments=" + strconv.Itoa(page)
	}
	link += "#comment-" + reply.ID.String()
	mute := ""
	if s.Links != nil {
		mute = s.sitePath(ctx, &ws, "comments", "mute", s.Links.Sign(linktoken.Claims{Kind: CommentMute, WorkspaceID: ws.ID, ContactID: g.contact}))
	}
	msg := s.commentReplyEmail(ctx, &ws, c, set, l, commentEmail{name: name, post: reply.PostTitle, comment: reply.ParentBody, reply: reply.Body, link: link,
		mute: mute, more: len(live) - 1})
	msg.WorkspaceID, msg.To, msg.ToName = &ws.ID, to.Email, to.FirstName
	if _, err := s.mail().SendSystem(ctx, msg); err != nil {
		// The notices wait for the next run (a week at most).
		return false, err
	}
	ids := make([]uuid.UUID, len(live))
	for i, r := range live {
		ids[i] = r.ID
	}
	return true, s.Q.WebsiteCommentNoticesSent(ctx, store.WebsiteCommentNoticesSentParams{At: &now, ContactID: g.contact, Comments: ids})
}

// PurgeComments is the hourly keeping: comments' addresses after 30
// days, deleted comments' text once Undo has passed, sent notices after
// a week.
func (s *Service) PurgeComments(ctx context.Context) error {
	now := s.now()
	if err := s.Q.WebsiteCommentIPsClear(ctx, now.AddDate(0, 0, -30)); err != nil {
		return err
	}
	if err := s.Q.WebsiteCommentsDeletedClear(ctx, &now); err != nil {
		return err
	}
	week := now.AddDate(0, 0, -7)
	return s.Q.WebsiteCommentNoticesPurge(ctx, &week)
}
