package website

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// Comments on posts (M61, G-27; publish-sell brief "Comments"). Members
// comment under the posts they can read, when Website › Access lets them
// (signed-in members, or paid members only) and the post doesn't turn
// comments off. A comment waits for review when the site reviews every
// comment, when it's the member's first, or when it holds two links or
// more. The team decides in Website › Comments (publish, hide, delete, with
// Undo for 10 seconds), pins comments, replies as the publication and
// blocks commenters. Member-written comments aren't audited (high-volume
// reader content, as page views and sign-ins); every moderator action is.

// Comment statuses.
const (
	CommentPending   = "pending"
	CommentPublished = "published"
	CommentHidden    = "hidden"
	CommentDeleted   = "deleted"
)

// Who may comment (Website › Access).
const (
	CommentsOff     = "off"
	CommentsMembers = "members"
	CommentsPaid    = "paid"
)

// Which comments wait for review.
const (
	ReviewAll   = "all"
	ReviewFirst = "first"
	ReviewNone  = "none"
)

// Why a comment waits for review.
const (
	HeldFirstComment = "first_comment"
	HeldReviewAll    = "review_all"
	HeldLinks        = "links"
	HeldBlocked      = "blocked"
)

const (
	// MaxCommentLength is a comment's longest text, in characters.
	MaxCommentLength = 5000
	// MaxDisplayName is a member's longest display name.
	MaxDisplayName = 60
	// CommentsPerPage is how many top-level comments a page of a post shows,
	// each with its replies.
	CommentsPerPage = 50
	// CommentEditFor is how long a member can edit their own comment.
	CommentEditFor = 15 * time.Minute
	// CommentUndoFor is how long Undo takes a moderation decision back.
	CommentUndoFor = 10 * time.Second
	// commentHeldLinks is how many links hold a comment for review.
	commentHeldLinks = 2
)

// WebsiteComment is a comment in Website › Comments.
type WebsiteComment struct {
	ID        string `json:"id"`
	PostID    string `json:"postId"`
	PostTitle string `json:"postTitle"`
	// URL is the comment on the site (…/posts/<slug>#comment-<id>).
	URL string `json:"url"`
	// ParentID is the comment a reply answers ("" for a top-level
	// comment), ParentName who wrote it (the publication's name for a team
	// reply).
	ParentID   string `json:"parentId,omitempty"`
	ParentName string `json:"parentName,omitempty"`
	// A member's comment: their contact, the name it shows and their
	// address (for the team only, never on the site), whether they're
	// blocked. A team reply shows as the publication; By is who wrote it.
	ContactID  string `json:"contactId,omitempty"`
	Name       string `json:"name"`
	Email      string `json:"email,omitempty"`
	Blocked    bool   `json:"blocked"`
	Team       bool   `json:"team"`
	By         string `json:"by,omitempty"`
	Body       string `json:"body"`
	Status     string `json:"status" enum:"pending,published,hidden,deleted"`
	HeldReason string `json:"heldReason,omitempty" enum:"first_comment,review_all,links,blocked,"`
	Pinned     bool   `json:"pinned"`
	Replies    int64  `json:"replies"`
	// Links is how many links the text holds (two or more hold it).
	Links     int        `json:"links"`
	CreatedAt time.Time  `json:"createdAt"`
	EditedAt  *time.Time `json:"editedAt,omitempty"`
	// PublishedAt is when it first showed on the site.
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	ModeratedBy string     `json:"moderatedBy,omitempty"`
	ModeratedAt *time.Time `json:"moderatedAt,omitempty"`
	// FirstReply is the first published reply, for the thread's preview.
	FirstReply *WebsiteCommentReply `json:"firstReply,omitempty"`
}

// WebsiteCommentReply is a reply as a thread's preview shows it: who (the
// publication for a team reply, By who wrote it) and its text.
type WebsiteCommentReply struct {
	Name      string    `json:"name"`
	Team      bool      `json:"team"`
	By        string    `json:"by,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// WebsiteCommentCounts are the tabs' counts.
type WebsiteCommentCounts struct {
	Pending   int64 `json:"pending"`
	Published int64 `json:"published"`
	Hidden    int64 `json:"hidden"`
}

// WebsiteCommentList is a page of one tab of Website › Comments.
type WebsiteCommentList struct {
	Items      []WebsiteComment     `json:"items"`
	NextCursor string               `json:"nextCursor,omitempty"`
	Counts     WebsiteCommentCounts `json:"counts"`
	// Comments and Review are the site's settings (Website › Access);
	// Publication is the name team replies show ("Reply as …").
	Comments    string `json:"comments" enum:"off,members,paid"`
	Review      string `json:"review" enum:"all,first,none"`
	Publication string `json:"publication"`
	// PublicationMark is the letter the site's mark shows (its publisher's
	// initial), on team replies.
	PublicationMark string `json:"publicationMark"`
	// Posts are the posts with comments, for the post filter.
	Posts []WebsiteCommentPost `json:"posts"`
}

// WebsiteCommentPost is a post with comments (waiting, published or hidden).
type WebsiteCommentPost struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Comments int64  `json:"comments"`
}

// WebsiteCommentFilter narrows Website › Comments.
type WebsiteCommentFilter struct {
	Status string `query:"status" enum:"pending,published,hidden," doc:"The tab: waiting for review (the default), published or hidden"`
	Post   string `query:"post" maxLength:"64" doc:"A post's id"`
	Search string `query:"q" maxLength:"200" doc:"Words in the comment, the commenter's name or address"`
	Cursor string `query:"cursor" maxLength:"200" doc:"nextCursor of the page before"`
	Limit  int    `query:"limit" minimum:"0" maximum:"100" doc:"Comments per page (50 by default)"`
}

// WebsiteCommentDecisionInput publishes, hides or deletes comments.
type WebsiteCommentDecisionInput struct {
	IDs      []string `json:"ids" minItems:"1" maxItems:"200"`
	Decision string   `json:"decision" enum:"publish,hide,delete"`
}

// WebsiteCommentDecision is a decision made, with how long Undo works.
type WebsiteCommentDecision struct {
	DecisionID string    `json:"decisionId"`
	UndoUntil  time.Time `json:"undoUntil"`
	Count      int       `json:"count"`
}

// WebsiteCommentReplyInput is a reply as the publication.
type WebsiteCommentReplyInput struct {
	Body string `json:"body" minLength:"1" maxLength:"5000"`
}

// WebsiteCommenter is a member as a commenter: blocked or not.
type WebsiteCommenter struct {
	ContactID string     `json:"contactId"`
	Name      string     `json:"name"`
	Blocked   bool       `json:"blocked"`
	BlockedAt *time.Time `json:"blockedAt,omitempty"`
	// Held are their comments waiting for review, now marked as a blocked
	// commenter's.
	Held int64 `json:"held"`
}

// CommenterName is the name a member's comments show: theirs, else their
// name on the list, else "" (the page says "A reader"). Never their address.
func CommenterName(display, first, last string) string {
	if d := strings.TrimSpace(display); d != "" {
		return d
	}
	return clip(strings.TrimSpace(strings.TrimSpace(first)+" "+strings.TrimSpace(last)), MaxDisplayName)
}

// commentURL finds the addresses in a comment: http(s) links and www. hosts.
var commentURL = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s<>"'` + "`" + `]+`)

// CommentPart is a piece of a comment's text: plain text, or a link (Href
// set) showing Text.
type CommentPart struct {
	Text, Href string
}

// CommentParts splits a comment's text into plain text and its links, for
// the page to escape and link (rel="nofollow ugc noopener"). Punctuation
// after an address ("see https://acme.io/docs.") stays text.
func CommentParts(body string) []CommentPart {
	var out []CommentPart
	last := 0
	for _, m := range commentURL.FindAllStringIndex(body, -1) {
		start, end := m[0], m[1]
		end = start + len(trimURL(body[start:end]))
		if end <= start {
			continue
		}
		if start > last {
			out = append(out, CommentPart{Text: body[last:start]})
		}
		u := body[start:end]
		href := u
		if !strings.HasPrefix(strings.ToLower(u), "http") {
			href = "https://" + u
		}
		out = append(out, CommentPart{Text: u, Href: href})
		last = end
	}
	if last < len(body) {
		out = append(out, CommentPart{Text: body[last:]})
	}
	return out
}

// trimURL drops the punctuation that ends a sentence after an address, and
// a closing bracket the address didn't open.
func trimURL(u string) string {
	for u != "" {
		r, size := utf8.DecodeLastRuneInString(u)
		switch r {
		case '.', ',', ';', ':', '!', '?', '\'', '"', '*':
			u = u[:len(u)-size]
			continue
		case ')', ']', '}':
			open := map[rune]rune{')': '(', ']': '[', '}': '{'}[r]
			if strings.Count(u, string(open)) < strings.Count(u, string(r)) {
				u = u[:len(u)-size]
				continue
			}
		}
		return u
	}
	return u
}

// commentLinks counts the links in a comment.
func commentLinks(body string) int {
	n := 0
	for _, p := range CommentParts(body) {
		if p.Href != "" {
			n++
		}
	}
	return n
}

// cleanComment is a comment's text as stored: valid UTF-8, no control
// characters but line breaks and tabs, line breaks as \n, no space around.
func cleanComment(s string) string {
	s = strings.ToValidUTF8(s, "")
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

// cleanName is a display name as stored: one line, no control characters.
func cleanName(s string) string {
	s = strings.ToValidUTF8(s, "")
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

// WebsiteCommentPath is a comment's path.
type WebsiteCommentPath struct {
	api.WorkspaceInput
	Comment string `path:"comment" format:"uuid" doc:"The comment's id"`
}

// WebsiteCommentDecisionPath is a moderation decision's path.
type WebsiteCommentDecisionPath struct {
	api.WorkspaceInput
	Decision string `path:"decision" format:"uuid" doc:"The decision's id (decisionId)"`
}

// WebsiteMemberPath is a website member's path.
type WebsiteMemberPath struct {
	api.WorkspaceInput
	Contact string `path:"contact" format:"uuid" doc:"The member's contact id"`
}

type commentsInput struct {
	api.WorkspaceInput
	WebsiteCommentFilter
}

type commentReplyBody struct {
	WebsiteCommentPath
	Body WebsiteCommentReplyInput
}

// registerComments adds Website › Comments' operations: the list reads with
// website:view, every decision needs comments:moderate (members only, as
// a person decides what readers see).
func registerComments(h huma.API, s *Service) {
	tags := []string{"Website"}
	base := "/workspaces/{workspace}/website"
	moderate := api.WorkspaceHuman(authz.CommentsModerate)
	api.Get(h, api.Op{ID: "list-website-comments", Path: base + "/comments", Summary: "Website › Comments: comments waiting for review, published or hidden",
		Tags: tags, Access: api.Workspace(authz.WebsiteView)},
		func(ctx context.Context, in *commentsInput) (*api.Body[WebsiteCommentList], error) {
			out, err := s.Comments(ctx, api.WorkspaceFrom(ctx), in.WebsiteCommentFilter)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "decide-website-comments", Path: base + "/comments/decide", Summary: "Publish, hide or delete comments",
		Description: "Undo works for 10 seconds with the decision's id. A deleted comment's text is gone once Undo has passed.", Tags: tags, Access: moderate},
		func(ctx context.Context, in *body[WebsiteCommentDecisionInput]) (*api.Body[WebsiteCommentDecision], error) {
			out, err := s.DecideComments(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "undo-website-comment-decision", Path: base + "/comments/decisions/{decision}/undo", Summary: "Take back a decision on comments within 10 seconds",
		Tags: tags, Access: moderate},
		func(ctx context.Context, in *WebsiteCommentDecisionPath) (*api.Empty, error) {
			id, err := parseID(ctx, in.Decision, "decision")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.UndoCommentDecision(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	for _, pin := range []bool{true, false} {
		id, path, summary := "pin-website-comment", "/pin", "Pin a comment to the top of its post"
		if !pin {
			id, path, summary = "unpin-website-comment", "/unpin", "Unpin a comment"
		}
		api.Post(h, api.Op{ID: id, Path: base + "/comments/{comment}" + path, Summary: summary, Tags: tags, Access: moderate},
			func(ctx context.Context, in *WebsiteCommentPath) (*api.Body[WebsiteComment], error) {
				cid, err := parseID(ctx, in.Comment, "comment")
				if err != nil {
					return nil, err
				}
				out, err := s.PinComment(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), cid, pin)
				return api.OK(out), err
			})
	}
	api.Post(h, api.Op{ID: "reply-to-website-comment", Path: base + "/comments/{comment}/reply", Summary: "Reply to a comment as the publication",
		Description: "The reply is published at once and shows the publication's name with a Team badge; the commenter gets the reply email.",
		Tags:        tags, Access: moderate, Status: http.StatusCreated},
		func(ctx context.Context, in *commentReplyBody) (*api.Body[WebsiteComment], error) {
			cid, err := parseID(ctx, in.Comment, "comment")
			if err != nil {
				return nil, err
			}
			out, err := s.ReplyToComment(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), cid, in.Body)
			return api.OK(out), err
		})
	for _, block := range []bool{true, false} {
		id, path, summary := "block-website-commenter", "/comments-block", "Block a member from commenting"
		description := "Their comments waiting for review stay there, marked as a blocked commenter's; published ones stay up until you hide them."
		if !block {
			id, path, summary, description = "unblock-website-commenter", "/comments-unblock", "Let a blocked member comment again", ""
		}
		api.Post(h, api.Op{ID: id, Path: base + "/members/{contact}" + path, Summary: summary, Description: description, Tags: tags, Access: moderate},
			func(ctx context.Context, in *WebsiteMemberPath) (*api.Body[WebsiteCommenter], error) {
				cid, err := parseID(ctx, in.Contact, "contact")
				if err != nil {
					return nil, err
				}
				out, err := s.BlockCommenter(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), cid, block)
				return api.OK(out), err
			})
	}
}

func encodeCommentCursor(pinned bool, at time.Time, id uuid.UUID) string {
	p := "0"
	if pinned {
		p = "1"
	}
	return base64.RawURLEncoding.EncodeToString([]byte(p + "|" + at.UTC().Format(time.RFC3339Nano) + "|" + id.String()))
}

// decodeCommentCursor reads a page's cursor: the last comment's pin, time
// and id (pinned comments come first).
func decodeCommentCursor(c string) (bool, time.Time, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return false, time.Time{}, uuid.Nil, err
	}
	parts := strings.Split(string(b), "|")
	if len(parts) != 3 || (parts[0] != "0" && parts[0] != "1") {
		return false, time.Time{}, uuid.Nil, fmt.Errorf("bad cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return false, time.Time{}, uuid.Nil, err
	}
	u, err := uuid.Parse(parts[2])
	return parts[0] == "1", t, u, err
}

// likeOf is a search's ILIKE pattern.
func likeOf(q string) *string {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	p := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
	return &p
}

// commentsBase is the address of the site's posts ("…/posts/"), for the
// comments' links.
func (s *Service) commentsBase(ctx context.Context, ws *store.Workspace) string {
	return strings.TrimSuffix(s.sitePath(ctx, ws, "posts", "x"), "x")
}

// commentItem is a comment as Website › Comments lists it.
func commentItem(r store.WebsiteCommentListRow, base, publication string) WebsiteComment {
	c := r.WebsiteComment
	out := WebsiteComment{ID: c.ID.String(), PostID: c.PostID.String(), PostTitle: r.PostTitle, URL: base + r.PostSlug + "#comment-" + c.ID.String(),
		Body: c.Body, Status: c.Status, HeldReason: c.HeldReason, Pinned: c.PinnedAt != nil, Replies: r.Replies, CreatedAt: c.CreatedAt,
		EditedAt: c.EditedAt, PublishedAt: c.PublishedAt, ModeratedBy: r.ModeratedByName, ModeratedAt: c.ModeratedAt, Blocked: r.Blocked,
		Links: commentLinks(c.Body)}
	if c.Status != CommentPending {
		out.HeldReason = ""
	}
	switch {
	case c.Team:
		out.Team, out.Name, out.By = true, publication, r.UserName
	case c.ContactID != nil:
		out.ContactID = c.ContactID.String()
		out.Name = CommenterName(r.DisplayName, r.FirstName, r.LastName)
		out.Email = r.Email
	}
	if r.FirstReplyBody != "" {
		// A published reply always has text.
		reply := &WebsiteCommentReply{Name: CommenterName(r.FirstReplyName, "", ""), Team: r.FirstReplyTeam, Body: r.FirstReplyBody, CreatedAt: r.FirstReplyAt}
		if reply.Team {
			reply.Name, reply.By = publication, r.FirstReplyBy
		}
		out.FirstReply = reply
	}
	if c.ParentID != nil {
		out.ParentID = c.ParentID.String()
		out.ParentName = r.ParentName
		if r.ParentTeam {
			out.ParentName = publication
		}
	}
	return out
}

// publication is the name team replies show: the site's title.
func (s *Service) publication(ctx context.Context, ws *store.Workspace, c *config) string {
	if s.Forms == nil {
		return c.row.Title
	}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return c.row.Title
	}
	return siteTitle(c, set)
}

// publicationMark is the initial the site's mark shows.
func (s *Service) publicationMark(ctx context.Context, ws *store.Workspace) string {
	if s.Forms == nil {
		return ""
	}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return ""
	}
	return set.Branding.Initial()
}

// Comments lists one tab of Website › Comments, pinned first, then newest.
func (s *Service) Comments(ctx context.Context, ws *store.Workspace, f WebsiteCommentFilter) (WebsiteCommentList, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteCommentList{}, err
	}
	out := WebsiteCommentList{Items: []WebsiteComment{}, Comments: c.row.Comments, Review: c.row.CommentsReview, Publication: s.publication(ctx, ws, c),
		PublicationMark: s.publicationMark(ctx, ws),
		Posts:           []WebsiteCommentPost{}}
	status := f.Status
	if status == "" {
		status = CommentPending
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	var post *uuid.UUID
	if f.Post != "" {
		id, err := uuid.Parse(f.Post)
		if err != nil {
			return out, api.Invalid("post", tr(ctx, "notFound.post", nil))
		}
		post = &id
	}
	params := store.WebsiteCommentListParams{WorkspaceID: ws.ID, Status: &status, Post: post, Q: likeOf(f.Search), Lim: int32(limit + 1)}
	if f.Cursor != "" {
		pinned, at, id, err := decodeCommentCursor(f.Cursor)
		if err != nil {
			return out, api.Invalid("cursor", tr(ctx, "comments.cursorInvalid", nil))
		}
		pin := int32(0)
		if pinned {
			pin = 1
		}
		params.BeforePinned, params.BeforeAt, params.BeforeID = &pin, &at, &id
	}
	rows, err := s.Q.WebsiteCommentList(ctx, params)
	if err != nil {
		return out, err
	}
	counts, err := s.Q.WebsiteCommentStatusCounts(ctx, store.WebsiteCommentStatusCountsParams{WorkspaceID: ws.ID, Post: post, Q: params.Q})
	if err != nil {
		return out, err
	}
	out.Counts = WebsiteCommentCounts{Pending: counts.Pending, Published: counts.Published, Hidden: counts.Hidden}
	posts, err := s.Q.WebsiteCommentPosts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, p := range posts {
		out.Posts = append(out.Posts, WebsiteCommentPost{ID: p.ID.String(), Title: p.Title, Comments: p.Comments})
	}
	base := s.commentsBase(ctx, ws)
	for i, r := range rows {
		if i == limit {
			last := rows[limit-1].WebsiteComment
			out.NextCursor = encodeCommentCursor(last.PinnedAt != nil, last.CreatedAt, last.ID)
			break
		}
		out.Items = append(out.Items, commentItem(r, base, out.Publication))
	}
	return out, nil
}

// comment is one comment as Website › Comments shows it.
func (s *Service) comment(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WebsiteComment, error) {
	rows, err := s.Q.WebsiteCommentList(ctx, store.WebsiteCommentListParams{WorkspaceID: ws.ID, ID: &id, Lim: 1})
	if err != nil {
		return WebsiteComment{}, err
	}
	if len(rows) == 0 {
		return WebsiteComment{}, notFound(ctx, "comment")
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteComment{}, err
	}
	return commentItem(rows[0], s.commentsBase(ctx, ws), s.publication(ctx, ws, c)), nil
}

// commentLabels name comments for their audit entries: who wrote them and
// under which post.
func commentLabels(ctx context.Context, q *store.Queries, ws *store.Workspace, ids []uuid.UUID) (map[uuid.UUID]store.WebsiteCommentLabelsRow, error) {
	rows, err := q.WebsiteCommentLabels(ctx, store.WebsiteCommentLabelsParams{WorkspaceID: ws.ID, Ids: ids})
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]store.WebsiteCommentLabelsRow, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out, nil
}

// labelName is who wrote a comment, for its audit entry.
func labelName(l store.WebsiteCommentLabelsRow, publication string) string {
	if l.Team {
		return publication
	}
	return CommenterName(l.DisplayName, l.FirstName, l.LastName)
}

var decisionStatus = map[string]string{"publish": CommentPublished, "hide": CommentHidden, "delete": CommentDeleted}

// DecideComments publishes, hides or deletes comments. Undo takes it back
// for 10 seconds; a deleted comment's text goes once it has passed.
func (s *Service) DecideComments(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebsiteCommentDecisionInput) (WebsiteCommentDecision, error) {
	status, ok := decisionStatus[in.Decision]
	if !ok {
		return WebsiteCommentDecision{}, api.Invalid("body.decision", tr(ctx, "comments.decisionInvalid", nil))
	}
	ids := make([]uuid.UUID, 0, len(in.IDs))
	seen := map[uuid.UUID]bool{}
	for _, raw := range in.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return WebsiteCommentDecision{}, api.Invalid("body.ids", tr(ctx, "comments.notFound", nil))
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 || len(ids) > 200 {
		return WebsiteCommentDecision{}, api.Invalid("body.ids", tr(ctx, "comments.pick", nil))
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteCommentDecision{}, err
	}
	publication := s.publication(ctx, ws, c)
	now := s.now()
	undo := now.Add(CommentUndoFor)
	decision := newID()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rows, err := q.WebsiteCommentsLock(ctx, store.WebsiteCommentsLockParams{WorkspaceID: ws.ID, Ids: ids})
		if err != nil {
			return err
		}
		if len(rows) != len(ids) {
			return api.Err(http.StatusNotFound, api.CodeNotFound, tr(ctx, "comments.notFound", nil))
		}
		for _, r := range rows {
			if r.Status == CommentDeleted {
				return api.Err(http.StatusConflict, "comment_deleted", tr(ctx, "comments.alreadyDeleted", nil))
			}
		}
		labels, err := commentLabels(ctx, q, ws, ids)
		if err != nil {
			return err
		}
		if err := q.WebsiteCommentsDecide(ctx, store.WebsiteCommentsDecideParams{Status: status, At: now, ModeratedBy: userID(p), DecisionID: &decision,
			UndoUntil: &undo, WorkspaceID: ws.ID, Ids: ids}); err != nil {
			return err
		}
		if status == CommentPublished {
			if err := q.WebsiteCommentNoticesQueue(ctx, store.WebsiteCommentNoticesQueueParams{At: now, Replies: ids}); err != nil {
				return err
			}
		} else if err := q.WebsiteCommentNoticesUnqueue(ctx, ids); err != nil {
			return err
		}
		for _, r := range rows {
			if r.Status == status {
				continue
			}
			l := labels[r.ID]
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "comment." + status, TargetType: "website_comment", TargetID: r.ID.String(),
				TargetName: labelName(l, publication), Before: map[string]any{"status": r.Status}, After: map[string]any{"status": status},
				Metadata: map[string]any{"post": l.PostTitle, "post_id": r.PostID.String(), "decision": decision.String()}}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return WebsiteCommentDecision{}, err
	}
	return WebsiteCommentDecision{DecisionID: decision.String(), UndoUntil: undo, Count: len(ids)}, nil
}

// UndoCommentDecision takes a decision on comments back within 10 seconds:
// each comment is as it was, and reply emails follow.
func (s *Service) UndoCommentDecision(ctx context.Context, ws *store.Workspace, p *authz.Principal, decision uuid.UUID) error {
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rows, err := q.WebsiteCommentsOfDecision(ctx, store.WebsiteCommentsOfDecisionParams{WorkspaceID: ws.ID, DecisionID: &decision})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return notFound(ctx, "decision")
		}
		var unpublished, republished []uuid.UUID
		for _, r := range rows {
			if r.UndoUntil == nil || now.After(*r.UndoUntil) {
				return api.Err(http.StatusConflict, "undo_late", tr(ctx, "comments.undoLate", nil))
			}
			switch {
			case r.Status == CommentPublished && r.BeforeStatus != CommentPublished:
				unpublished = append(unpublished, r.ID)
			case r.Status != CommentPublished && r.BeforeStatus == CommentPublished:
				republished = append(republished, r.ID)
			}
		}
		if err := q.WebsiteCommentsUndo(ctx, store.WebsiteCommentsUndoParams{WorkspaceID: ws.ID, DecisionID: &decision}); err != nil {
			return err
		}
		if len(unpublished) > 0 {
			if err := q.WebsiteCommentNoticesUnqueue(ctx, unpublished); err != nil {
				return err
			}
		}
		if len(republished) > 0 {
			if err := q.WebsiteCommentNoticesQueue(ctx, store.WebsiteCommentNoticesQueueParams{At: now, Replies: republished}); err != nil {
				return err
			}
		}
		ids := make([]string, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID.String())
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "comment.decision_undone", TargetType: "website_comment", TargetID: rows[0].ID.String(),
			After: map[string]any{"status": rows[0].BeforeStatus}, Metadata: map[string]any{"decision": decision.String(), "comments": ids}})
	})
}

// PinComment pins a published top-level comment to the top of its post,
// or unpins it.
func (s *Service) PinComment(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, pin bool) (WebsiteComment, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteComment{}, err
	}
	publication := s.publication(ctx, ws, c)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsiteCommentLock(ctx, store.WebsiteCommentLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx, "comment")
		}
		if err != nil {
			return err
		}
		if (cur.PinnedAt != nil) == pin {
			return nil
		}
		if pin && cur.ParentID != nil {
			return api.Err(http.StatusConflict, "comment_reply", tr(ctx, "comments.pinReply", nil))
		}
		if pin && cur.Status != CommentPublished {
			return api.Err(http.StatusConflict, "comment_not_published", tr(ctx, "comments.pinUnpublished", nil))
		}
		var at *time.Time
		action := "comment.unpinned"
		if pin {
			now := s.now()
			at, action = &now, "comment.pinned"
		}
		if err := q.WebsiteCommentPin(ctx, store.WebsiteCommentPinParams{PinnedAt: at, WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		labels, err := commentLabels(ctx, q, ws, []uuid.UUID{id})
		if err != nil {
			return err
		}
		l := labels[id]
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "website_comment", TargetID: id.String(),
			TargetName: labelName(l, publication), Metadata: map[string]any{"post": l.PostTitle, "post_id": cur.PostID.String()}})
	})
	if err != nil {
		return WebsiteComment{}, err
	}
	return s.comment(ctx, ws, id)
}

// ReplyToComment replies to a comment as the publication: published at
// once, under the comment's thread, written by the member who sent it.
func (s *Service) ReplyToComment(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsiteCommentReplyInput) (WebsiteComment, error) {
	text := cleanComment(in.Body)
	if text == "" {
		return WebsiteComment{}, api.Invalid("body.body", tr(ctx, "comments.empty", nil))
	}
	if utf8.RuneCountInString(text) > MaxCommentLength {
		return WebsiteComment{}, api.Invalid("body.body", tr(ctx, "comments.tooLong", i18n.Args{"max": MaxCommentLength}))
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteComment{}, err
	}
	publication := s.publication(ctx, ws, c)
	reply := newID()
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		parent, err := q.WebsiteCommentLock(ctx, store.WebsiteCommentLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx, "comment")
		}
		if err != nil {
			return err
		}
		// One level of replies: a reply to a reply joins its thread.
		top := parent
		if parent.ParentID != nil {
			if top, err = q.WebsiteCommentLock(ctx, store.WebsiteCommentLockParams{WorkspaceID: ws.ID, ID: *parent.ParentID}); err != nil {
				return err
			}
		}
		if parent.Status != CommentPublished || top.Status != CommentPublished {
			return api.Err(http.StatusConflict, "comment_not_published", tr(ctx, "comments.replyUnpublished", nil))
		}
		if err := q.WebsiteCommentInsert(ctx, store.WebsiteCommentInsertParams{ID: reply, WorkspaceID: ws.ID, PostID: top.PostID, ParentID: &top.ID,
			Team: true, UserID: userID(p), Body: text, Status: CommentPublished, At: now, PublishedAt: &now}); err != nil {
			return err
		}
		if err := q.WebsiteCommentNoticesQueue(ctx, store.WebsiteCommentNoticesQueueParams{At: now, Replies: []uuid.UUID{reply}}); err != nil {
			return err
		}
		labels, err := commentLabels(ctx, q, ws, []uuid.UUID{parent.ID})
		if err != nil {
			return err
		}
		l := labels[parent.ID]
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "comment.team_replied", TargetType: "website_comment", TargetID: reply.String(),
			TargetName: labelName(l, publication), Metadata: map[string]any{"post": l.PostTitle, "post_id": top.PostID.String(), "replied_to": parent.ID.String()}})
	})
	if err != nil {
		return WebsiteComment{}, err
	}
	return s.comment(ctx, ws, reply)
}

// BlockCommenter stops a member from commenting, or lets them again. Their
// comments waiting for review stay there, marked as a blocked commenter's.
func (s *Service) BlockCommenter(ctx context.Context, ws *store.Workspace, p *authz.Principal, contact uuid.UUID, block bool) (WebsiteCommenter, error) {
	var out WebsiteCommenter
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		who, err := q.WebsiteCommenter(ctx, store.WebsiteCommenterParams{WorkspaceID: ws.ID, ContactID: contact})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx, "contact")
		}
		if err != nil {
			return err
		}
		if !who.Member {
			return api.Err(http.StatusConflict, "not_a_member", tr(ctx, "comments.notMember", nil))
		}
		name := CommenterName(who.DisplayName, who.FirstName, who.LastName)
		if name == "" {
			name = who.Email
		}
		out = WebsiteCommenter{ContactID: contact.String(), Name: name, Blocked: who.CommentsBlockedAt != nil, BlockedAt: who.CommentsBlockedAt}
		if out.Blocked == block {
			return nil
		}
		var at *time.Time
		action := "website_member.comments_unblocked"
		if block {
			now := s.now()
			at, action = &now, "website_member.comments_blocked"
		}
		if _, err := q.WebsiteCommenterBlock(ctx, store.WebsiteCommenterBlockParams{At: at, ContactID: contact, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		out.Blocked, out.BlockedAt = block, at
		if block {
			if out.Held, err = q.WebsiteCommentsHoldBlocked(ctx, store.WebsiteCommentsHoldBlockedParams{WorkspaceID: ws.ID, ContactID: &contact}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "contact", TargetID: contact.String(), TargetName: name,
			Before: map[string]any{"blocked": !block}, After: map[string]any{"blocked": block}, Metadata: map[string]any{"held": out.Held}})
	})
	return out, err
}

// commentsWaiting tells the people who moderate that comments wait for
// review: when the queue goes from empty to waiting, then at most once an
// hour.
func (s *Service) commentsWaiting(ctx context.Context, ws *store.Workspace, wasEmpty bool) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		s.log().WarnContext(ctx, "comments waiting", "err", err)
		return
	}
	now := s.now()
	if last := c.row.CommentsNotifiedAt; !wasEmpty && last != nil && now.Sub(*last) < time.Hour {
		return
	}
	n, err := s.Q.WebsiteCommentsPending(ctx, ws.ID)
	if err != nil || n == 0 {
		return
	}
	site := s.publication(ctx, ws, c)
	note := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindCommentsWaiting, Severity: "info",
		TitleText: msgs.M("notify.commentsTitle").Keep(i18n.Args{"count": n}),
		BodyText:  msgs.M("notify.commentsBody").Keep(i18n.Args{"count": n, "site": site}),
		Link:      "/w/" + ws.Slug + "/forms/website/comments?status=pending", Data: map[string]any{"pending": n}}
	if err := notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.CommentsModerate, note); err != nil {
		s.log().WarnContext(ctx, "comments waiting", "err", err)
		return
	}
	if err := s.Q.WebsiteCommentsNotified(ctx, store.WebsiteCommentsNotifiedParams{At: &now, WorkspaceID: ws.ID}); err != nil {
		s.log().WarnContext(ctx, "comments waiting", "err", err)
	}
}
