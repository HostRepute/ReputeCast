package website

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// What readers do with comments on the site (internal/publicpages draws
// it): a post's comments as one reader sees them, writing, editing and
// deleting their own, and the account page's Comments section.

// What a reader may do under a post.
const (
	// CommentCanWrite: the form.
	CommentCanWrite = "write"
	// CommentCanSignin: a visitor; "Sign in to join the conversation".
	CommentCanSignin = "signin"
	// CommentCanUpgrade: a free member on a site where paid members comment.
	CommentCanUpgrade = "paid"
	// CommentCanBlocked: a moderator blocked the member.
	CommentCanBlocked = "blocked"
)

// What came of a comment a reader sent.
const (
	CommentPosted  = "posted"
	CommentHeld    = "held"
	CommentInvalid = "invalid"
	CommentLimited = "limited"
	CommentDenied  = "denied"
)

// Comment limits per member and per network.
const (
	commentsPer10Min      = 10
	commentsPerDay        = 50
	commentsPerNetworkHr  = 30
	commentsMaxShownPages = 20
)

// PublicComment is a comment as a reader sees it.
type PublicComment struct {
	ID uuid.UUID
	// Name is the member's display name ("" for a team reply, which shows
	// as the publication, and for a member who never named themselves).
	Name string
	Team bool
	Body string
	// Status is published, pending (the reader's own, waiting for
	// review) or removed (hidden or deleted, kept for its replies).
	Status string
	Pinned bool
	// Own: the reader wrote it; Editable: still within 15 minutes.
	Own, Editable bool
	CreatedAt     time.Time
	EditedAt      *time.Time
	Replies       []PublicComment
}

// Removed is what a removed comment with replies says instead of its text.
const commentRemoved = "removed"

// CommentsView is a post's comments section as one reader gets it.
type CommentsView struct {
	// Count is the post's published comments ("Comments (13)").
	Count int64
	// Threads are the top-level comments shown, with their replies, and
	// Shown how many threads that is of Threads. More: another page.
	Threads       []PublicComment
	Shown, Total  int
	Page          int
	More          bool
	Access        string
	Review        string
	FirstReviewed bool
	// NeedsName: the member's first comment asks for a display name,
	// filled in with NameDraft (their name on the list, never the email).
	NeedsName bool
	NameDraft string
	// DisplayName is the name the member's comments show.
	DisplayName string
	// Reply is the thread ?reply= answers; Edit and Delete the reader's own
	// comment ?edit= and ?delete= open.
	Reply, Edit *PublicComment
	// PaidOnly: only paid members comment; SignupOn: visitors can sign up.
	PaidOnly, SignupOn bool
}

// CommentsQuery is what the post's address asks for: a page of threads,
// the thread to reply to, the comment to edit.
type CommentsQuery struct {
	Page        int
	Reply, Edit uuid.UUID
}

// commentsOn reports whether a post shows comments: the site lets members
// comment, the post doesn't turn them off, and it's published.
func commentsOn(c *config, p store.WebsitePost) bool {
	return c.On() && c.row.Comments != CommentsOff && c.row.Comments != "" && !p.CommentsOff && p.Status == PostPublished && p.DeletedAt == nil
}

// commentAccess is what a member (or a visitor, who == nil) may do.
func commentAccess(c *config, who *store.WebsiteCommenterRow, reader gate.Reader) string {
	switch {
	case who == nil:
		return CommentCanSignin
	case who.CommentsBlockedAt != nil:
		return CommentCanBlocked
	case c.row.Comments == CommentsPaid && !reader.Member.Paid():
		return CommentCanUpgrade
	}
	return CommentCanWrite
}

// commenter is a signed-in member as their comments know them.
func (s *Service) commenter(ctx context.Context, q *store.Queries, ws *store.Workspace, sess *Session) (*store.WebsiteCommenterRow, error) {
	if sess == nil {
		return nil, nil
	}
	who, err := q.WebsiteCommenter(ctx, store.WebsiteCommenterParams{WorkspaceID: ws.ID, ContactID: sess.ContactID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &who, nil
}

// readable reports whether a reader may read a post whole, as its page
// decides, without counting a read on the meter.
func (s *Service) readable(ctx context.Context, ws *store.Workspace, c *config, p store.WebsitePost, reader gate.Reader) (bool, error) {
	rules, _, err := s.rules(ctx, s.Q, ws, c)
	if err != nil {
		return false, err
	}
	v := gate.Visibility{Level: p.Visibility, Tiers: p.Tiers, NeverFree: p.NeverFree}
	var state gate.MeterState
	if rule := gate.MeterRule(v, reader, rules); rule != "" && reader.Key() != "" {
		st, err := s.Q.WebsiteMeterState(ctx, store.WebsiteMeterStateParams{WorkspaceID: ws.ID, Month: monthOf(s.now(), location(ws)), Reader: reader.Key(),
			Rule: rule, PostID: p.ID})
		if err != nil {
			return false, err
		}
		state = gate.MeterState{Used: int(st.Used), Read: st.Read}
	}
	return gate.Decide(v, reader, rules, state).Readable(), nil
}

// CommentAccess is what a reader may do under a post: CommentCanWrite, or
// why not. Off (""): the post shows no comments to this reader.
func (s *Service) CommentAccess(ctx context.Context, ws *store.Workspace, p store.WebsitePost, sess *Session, reader gate.Reader) (string, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return "", err
	}
	if !commentsOn(c, p) {
		return "", nil
	}
	ok, err := s.readable(ctx, ws, c, p, reader)
	if err != nil || !ok {
		return "", err
	}
	who, err := s.commenter(ctx, s.Q, ws, sess)
	if err != nil {
		return "", err
	}
	return commentAccess(c, who, reader), nil
}

// publicComment is a stored comment as the reader sees it.
func publicComment(c store.WebsiteComment, display, first, last string, viewer *uuid.UUID, canWrite bool, now time.Time) PublicComment {
	out := PublicComment{ID: c.ID, Body: c.Body, Status: c.Status, Pinned: c.PinnedAt != nil && c.Status == CommentPublished, CreatedAt: c.CreatedAt,
		EditedAt: c.EditedAt, Replies: []PublicComment{}}
	if c.Status == CommentHidden || c.Status == CommentDeleted || (!c.Team && c.ContactID == nil) {
		return PublicComment{ID: c.ID, Status: commentRemoved, CreatedAt: c.CreatedAt, Replies: []PublicComment{}}
	}
	if c.Team {
		out.Team = true
	} else {
		out.Name = CommenterName(display, first, last)
		out.Own = viewer != nil && *c.ContactID == *viewer
	}
	out.Editable = out.Own && canWrite && now.Sub(c.CreatedAt) < CommentEditFor
	return out
}

// PostComments is a post's comments section for a reader (nil: the post
// shows no comments). The caller has decided the reader may read the post.
func (s *Service) PostComments(ctx context.Context, ws *store.Workspace, p store.WebsitePost, sess *Session, reader gate.Reader, q CommentsQuery) (*CommentsView, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	if !commentsOn(c, p) {
		return nil, nil
	}
	who, err := s.commenter(ctx, s.Q, ws, sess)
	if err != nil {
		return nil, err
	}
	out := &CommentsView{Access: commentAccess(c, who, reader), Review: c.row.CommentsReview, Threads: []PublicComment{},
		PaidOnly: c.row.Comments == CommentsPaid, SignupOn: c.row.SignupOn && c.row.SigninOn}
	var viewer *uuid.UUID
	if who != nil {
		id := who.ID
		viewer = &id
		out.DisplayName = CommenterName(who.DisplayName, "", "")
		if out.Access == CommentCanWrite && out.DisplayName == "" {
			out.NeedsName, out.NameDraft = true, CommenterName("", who.FirstName, who.LastName)
		}
		if out.Access == CommentCanWrite && c.row.CommentsReview == ReviewFirst {
			has, err := s.Q.WebsiteCommenterHasPublished(ctx, &id)
			if err != nil {
				return nil, err
			}
			out.FirstReviewed = !has
		}
	}
	if out.Count, err = s.Q.WebsiteCommentCount(ctx, p.ID); err != nil {
		return nil, err
	}
	total, err := s.Q.WebsiteCommentThreadCount(ctx, store.WebsiteCommentThreadCountParams{PostID: p.ID, Viewer: viewer})
	if err != nil {
		return nil, err
	}
	// "Show more comments" adds a page of 50 threads each time.
	pages := max(1, (int(total)+CommentsPerPage-1)/CommentsPerPage)
	out.Page = min(max(q.Page, 1), pages, commentsMaxShownPages)
	rows, err := s.Q.WebsiteCommentThreads(ctx, store.WebsiteCommentThreadsParams{PostID: p.ID, Viewer: viewer, Lim: int32(out.Page * CommentsPerPage)})
	if err != nil {
		return nil, err
	}
	out.Total, out.Shown, out.More = int(total), len(rows), len(rows) < int(total) && out.Page < commentsMaxShownPages
	now := s.now()
	canWrite := out.Access == CommentCanWrite
	ids := make([]uuid.UUID, 0, len(rows))
	index := map[uuid.UUID]int{}
	for _, r := range rows {
		index[r.WebsiteComment.ID] = len(out.Threads)
		ids = append(ids, r.WebsiteComment.ID)
		out.Threads = append(out.Threads, publicComment(r.WebsiteComment, r.DisplayName, r.FirstName, r.LastName, viewer, canWrite, now))
	}
	if len(ids) > 0 {
		replies, err := s.Q.WebsiteCommentReplies(ctx, store.WebsiteCommentRepliesParams{Parents: ids, Viewer: viewer})
		if err != nil {
			return nil, err
		}
		for _, r := range replies {
			if i, ok := index[*r.WebsiteComment.ParentID]; ok {
				out.Threads[i].Replies = append(out.Threads[i].Replies, publicComment(r.WebsiteComment, r.DisplayName, r.FirstName, r.LastName, viewer, canWrite, now))
			}
		}
	}
	if !canWrite {
		return out, nil
	}
	for i := range out.Threads {
		t := &out.Threads[i]
		if q.Reply != uuid.Nil && t.ID == q.Reply && t.Status == CommentPublished {
			out.Reply = t
		}
		if q.Edit == uuid.Nil {
			continue
		}
		if t.ID == q.Edit && t.Editable {
			out.Edit = t
		}
		for j := range t.Replies {
			if r := &t.Replies[j]; r.ID == q.Edit && r.Editable {
				out.Edit = r
			}
		}
	}
	return out, nil
}

// CommentInput is a comment a member sends from a post.
type CommentInput struct {
	Body string
	// Parent is the thread a reply answers (uuid.Nil: a new thread).
	Parent uuid.UUID
	// Name is the display name the first comment asks for.
	Name string
	IP   *netip.Addr
	Lang i18n.Lang
}

// CommentOutcome is what came of a comment: posted or held (with its page
// of the post), or why not, with the field at fault.
type CommentOutcome struct {
	Status  string
	ID      uuid.UUID
	Page    int
	Field   string
	Message string
}

func refused(status, field, message string) CommentOutcome {
	return CommentOutcome{Status: status, Field: field, Message: message}
}

// commentBody checks a comment's text: 1 to 5,000 characters.
func commentBody(w Words, body string) (string, CommentOutcome, bool) {
	text := cleanComment(body)
	if text == "" {
		return "", refused(CommentInvalid, "body", w.Text("comments.empty", nil)), false
	}
	if utf8.RuneCountInString(text) > MaxCommentLength {
		return "", refused(CommentInvalid, "body", w.Text("comments.tooLong", i18n.Args{"max": MaxCommentLength})), false
	}
	return text, CommentOutcome{}, true
}

// holdFor says why a comment waits for review ("" when it doesn't): the
// site reviews every comment, it's the member's first, or it has two links
// or more.
func holdFor(c *config, firstComment bool, body string) string {
	switch {
	case c.row.CommentsReview == ReviewAll:
		return HeldReviewAll
	case c.row.CommentsReview == ReviewFirst && firstComment:
		return HeldFirstComment
	case commentLinks(body) >= commentHeldLinks:
		return HeldLinks
	}
	return ""
}

// sessionMethod is a session's sign-in method as website_members keeps it.
func sessionMethod(sess *Session) string {
	switch sess.Method {
	case "code", "signup":
		return sess.Method
	}
	return "link"
}

// WriteComment posts a member's comment under a post, or a reply to one
// of its threads, after the limits (10 in 10 minutes and 50 a day per
// member, 30 an hour per network) and the checks. A comment that needs
// review waits for it; the people who moderate hear about it.
func (s *Service) WriteComment(ctx context.Context, ws *store.Workspace, p store.WebsitePost, sess *Session, reader gate.Reader, in CommentInput) (CommentOutcome, error) {
	w := WordsIn(in.Lang)
	if sess == nil {
		return refused(CommentDenied, "", w.Text("comments.denied", nil)), nil
	}
	access, err := s.CommentAccess(ctx, ws, p, sess, reader)
	if err != nil {
		return CommentOutcome{}, err
	}
	if access != CommentCanWrite {
		return refused(CommentDenied, "", w.Text("comments.denied", nil)), nil
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return CommentOutcome{}, err
	}
	who, err := s.commenter(ctx, s.Q, ws, sess)
	if err != nil || who == nil {
		return refused(CommentDenied, "", w.Text("comments.denied", nil)), err
	}
	now := s.now()
	contact := sess.ContactID
	mine, err := s.Q.WebsiteCommentsByMember(ctx, store.WebsiteCommentsByMemberParams{Recent: now.Add(-10 * time.Minute), ContactID: &contact, DayStart: now.Add(-24 * time.Hour)})
	if err != nil {
		return CommentOutcome{}, err
	}
	if mine.Recent >= commentsPer10Min || mine.Day >= commentsPerDay {
		return refused(CommentLimited, "body", w.Text("comments.limited", nil)), nil
	}
	if in.IP != nil {
		n, err := s.Q.WebsiteCommentsByNetwork(ctx, store.WebsiteCommentsByNetworkParams{WorkspaceID: ws.ID, Since: now.Add(-time.Hour), Network: forms.ClientKey(*in.IP)})
		if err != nil {
			return CommentOutcome{}, err
		}
		if n >= commentsPerNetworkHr {
			return refused(CommentLimited, "body", w.Text("comments.limited", nil)), nil
		}
	}
	text, bad, ok := commentBody(w, in.Body)
	if !ok {
		return bad, nil
	}
	name := ""
	if strings.TrimSpace(who.DisplayName) == "" {
		if name = cleanName(in.Name); name == "" {
			return refused(CommentInvalid, "name", w.Text("comments.nameRequired", nil)), nil
		}
		if utf8.RuneCountInString(name) > MaxDisplayName {
			return refused(CommentInvalid, "name", w.Text("comments.nameLong", i18n.Args{"max": MaxDisplayName})), nil
		}
	}
	thread := uuid.Nil
	var parent *uuid.UUID
	if in.Parent != uuid.Nil {
		top, err := s.Q.WebsiteCommentGet(ctx, store.WebsiteCommentGetParams{WorkspaceID: ws.ID, ID: in.Parent})
		if err == nil && top.ParentID != nil {
			top, err = s.Q.WebsiteCommentGet(ctx, store.WebsiteCommentGetParams{WorkspaceID: ws.ID, ID: *top.ParentID})
		}
		// A thread takes replies while it shows: published, or removed
		// with its replies still up.
		if errors.Is(err, pgx.ErrNoRows) || err == nil && (top.PostID != p.ID || top.Status == CommentPending) {
			return refused(CommentInvalid, "parent", w.Text("comments.parentGone", nil)), nil
		}
		if err != nil {
			return CommentOutcome{}, err
		}
		thread, parent = top.ID, &top.ID
	}
	first := false
	if c.row.CommentsReview == ReviewFirst {
		has, err := s.Q.WebsiteCommenterHasPublished(ctx, &contact)
		if err != nil {
			return CommentOutcome{}, err
		}
		first = !has
	}
	held := holdFor(c, first, text)
	status := CommentPublished
	var published *time.Time
	if held != "" {
		status = CommentPending
	} else {
		published = &now
	}
	id := newID()
	if thread == uuid.Nil {
		thread = id
	}
	var pendingBefore int64
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if pendingBefore, err = q.WebsiteCommentsPending(ctx, ws.ID); err != nil {
			return err
		}
		if name != "" {
			if err := q.WebsiteCommenterName(ctx, store.WebsiteCommenterNameParams{ContactID: contact, WorkspaceID: ws.ID, At: now, Method: sessionMethod(sess),
				DisplayName: name}); err != nil {
				return err
			}
		}
		if err := q.WebsiteCommentInsert(ctx, store.WebsiteCommentInsertParams{ID: id, WorkspaceID: ws.ID, PostID: p.ID, ParentID: parent, ContactID: &contact,
			Body: text, Status: status, HeldReason: held, Ip: in.IP, At: now, PublishedAt: published}); err != nil {
			return err
		}
		if status == CommentPublished && parent != nil {
			return q.WebsiteCommentNoticesQueue(ctx, store.WebsiteCommentNoticesQueueParams{At: now, Replies: []uuid.UUID{id}})
		}
		return nil
	})
	if err != nil {
		return CommentOutcome{}, err
	}
	out := CommentOutcome{Status: CommentPosted, ID: id, Page: s.threadPage(ctx, ws, p.ID, thread, &contact)}
	if status == CommentPending {
		out.Status = CommentHeld
		s.commentsWaiting(ctx, ws, pendingBefore == 0)
	}
	return out, nil
}

// threadPage is the page of a post's comments a thread is on, for the
// reader (1 when it can't tell).
func (s *Service) threadPage(ctx context.Context, ws *store.Workspace, post, thread uuid.UUID, viewer *uuid.UUID) int {
	t, err := s.Q.WebsiteCommentGet(ctx, store.WebsiteCommentGetParams{WorkspaceID: ws.ID, ID: thread})
	if err != nil || t.PostID != post {
		return 1
	}
	var pinned *time.Time
	if t.Status == CommentPublished {
		pinned = t.PinnedAt
	}
	before, err := s.Q.WebsiteCommentThreadsBefore(ctx, store.WebsiteCommentThreadsBeforeParams{PostID: post, ID: thread, Viewer: viewer, Pinned: pinned,
		CreatedAt: t.CreatedAt})
	if err != nil {
		return 1
	}
	return int(before)/CommentsPerPage + 1
}

// ownComment locks a member's own comment under a post.
func ownComment(ctx context.Context, q *store.Queries, ws *store.Workspace, p store.WebsitePost, sess *Session, id uuid.UUID) (store.WebsiteComment, bool, error) {
	cur, err := q.WebsiteCommentLock(ctx, store.WebsiteCommentLockParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return cur, false, nil
	}
	if err != nil {
		return cur, false, err
	}
	return cur, sess != nil && cur.PostID == p.ID && cur.ContactID != nil && *cur.ContactID == sess.ContactID, nil
}

// EditComment changes a member's own comment within 15 minutes. It waits
// for review again when the site reviews every comment, or when a
// published comment now holds two links or more.
func (s *Service) EditComment(ctx context.Context, ws *store.Workspace, p store.WebsitePost, sess *Session, reader gate.Reader, id uuid.UUID, body string, l i18n.Lang) (CommentOutcome, error) {
	w := WordsIn(l)
	if sess == nil {
		return refused(CommentDenied, "", w.Text("comments.denied", nil)), nil
	}
	access, err := s.CommentAccess(ctx, ws, p, sess, reader)
	if err != nil {
		return CommentOutcome{}, err
	}
	if access != CommentCanWrite {
		return refused(CommentDenied, "", w.Text("comments.denied", nil)), nil
	}
	text, bad, ok := commentBody(w, body)
	if !ok {
		return bad, nil
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return CommentOutcome{}, err
	}
	now := s.now()
	out := CommentOutcome{Status: CommentPosted, ID: id}
	thread := id
	var pendingBefore int64
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, mine, err := ownComment(ctx, q, ws, p, sess, id)
		if err != nil {
			return err
		}
		if !mine || (cur.Status != CommentPublished && cur.Status != CommentPending) {
			out = refused(CommentDenied, "", w.Text("comments.notYours", nil))
			return nil
		}
		if now.Sub(cur.CreatedAt) >= CommentEditFor {
			out = refused(CommentDenied, "", w.Text("comments.editLate", nil))
			return nil
		}
		if cur.ParentID != nil {
			thread = *cur.ParentID
		}
		status, held := cur.Status, cur.HeldReason
		switch {
		case c.row.CommentsReview == ReviewAll && cur.Status == CommentPublished:
			status, held = CommentPending, HeldReviewAll
		case cur.Status == CommentPublished && commentLinks(text) >= commentHeldLinks:
			status, held = CommentPending, HeldLinks
		}
		if pendingBefore, err = q.WebsiteCommentsPending(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.WebsiteCommentEdit(ctx, store.WebsiteCommentEditParams{Body: text, Status: status, HeldReason: held, At: &now, ID: id}); err != nil {
			return err
		}
		if status == CommentPending {
			out.Status = CommentHeld
		}
		if cur.Status == CommentPublished && status == CommentPending {
			return q.WebsiteCommentNoticesUnqueue(ctx, []uuid.UUID{id})
		}
		return nil
	})
	if err != nil || out.Status == CommentDenied {
		return out, err
	}
	out.Page = s.threadPage(ctx, ws, p.ID, thread, &sess.ContactID)
	if out.Status == CommentHeld {
		s.commentsWaiting(ctx, ws, pendingBefore == 0)
	}
	return out, nil
}

// DeleteOwnComment deletes a member's own comment, any time: its text goes
// at once; its replies stay under "This comment was removed."
func (s *Service) DeleteOwnComment(ctx context.Context, ws *store.Workspace, p store.WebsitePost, sess *Session, id uuid.UUID, l i18n.Lang) (CommentOutcome, error) {
	w := WordsIn(l)
	if sess == nil {
		return refused(CommentDenied, "", w.Text("comments.denied", nil)), nil
	}
	cur, err := s.Q.WebsiteCommentGet(ctx, store.WebsiteCommentGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return CommentOutcome{}, err
	}
	thread := id
	if cur.ParentID != nil {
		thread = *cur.ParentID
	}
	out := CommentOutcome{Status: CommentPosted, ID: id, Page: s.threadPage(ctx, ws, p.ID, thread, &sess.ContactID)}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, mine, err := ownComment(ctx, q, ws, p, sess, id)
		if err != nil {
			return err
		}
		if !mine {
			out = refused(CommentDenied, "", w.Text("comments.notYours", nil))
			return nil
		}
		if cur.Status == CommentDeleted {
			return nil
		}
		if err := q.WebsiteCommentDeleteOwn(ctx, id); err != nil {
			return err
		}
		return q.WebsiteCommentNoticesUnqueue(ctx, []uuid.UUID{id})
	})
	return out, err
}

// SaveCommentSettings saves the account page's Comments section: the
// display name (empty: the next comment asks; cut at 60 characters) and
// reply emails.
func (s *Service) SaveCommentSettings(ctx context.Context, ws *store.Workspace, sess *Session, name string, emails bool) error {
	return s.Q.WebsiteCommenterSettings(ctx, store.WebsiteCommenterSettingsParams{ContactID: sess.ContactID, WorkspaceID: ws.ID, At: s.now(),
		Method: sessionMethod(sess), DisplayName: clip(cleanName(name), MaxDisplayName), CommentEmails: emails})
}

// MuteCommentEmails turns a member's reply emails off ("Stop these
// emails"). ok is false when the contact isn't a member of the site.
func (s *Service) MuteCommentEmails(ctx context.Context, ws *store.Workspace, contact uuid.UUID) (bool, error) {
	n, err := s.Q.WebsiteCommenterMute(ctx, store.WebsiteCommenterMuteParams{ContactID: contact, WorkspaceID: ws.ID})
	return n > 0, err
}

// CommentEmailsOf is a member's reply-email setting and address, for the
// "Stop these emails" page.
func (s *Service) CommentEmailsOf(ctx context.Context, ws *store.Workspace, contact uuid.UUID) (email string, on bool, err error) {
	who, err := s.Q.WebsiteCommenter(ctx, store.WebsiteCommenterParams{WorkspaceID: ws.ID, ContactID: contact})
	if err != nil {
		return "", false, err
	}
	return who.Email, who.Member && who.CommentEmails, nil
}
