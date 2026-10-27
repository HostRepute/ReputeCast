package campaigns

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

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

// Comments on a campaign (review 2026-10, CO-10): threads on a block (the
// composer marks it with rc-comment="<id>", left out of the compiled email)
// or on the whole email, replies, @mentions that notify the people named,
// and resolving. Reviewers with a review link comment too (review.go).

// AttrComment marks the block a comment thread is on.
const AttrComment = "rc-comment"

var anchorRe = regexp.MustCompile(`^[a-z0-9]{0,40}$`)

// ComposerComment is one comment of a thread.
type ComposerComment struct {
	ID   string `json:"id"`
	Body string `json:"body"`
	// Author is the member's name, or the name a reviewer gave.
	Author   string `json:"author"`
	AuthorID string `json:"authorId,omitempty"`
	// AuthorColor is the member's avatar colour (empty for a reviewer).
	AuthorColor string `json:"authorColor,omitempty"`
	// Reviewer is true for a comment through a review link (no account).
	Reviewer  bool      `json:"reviewer"`
	Version   int       `json:"version" doc:"The campaign version it was written on"`
	Mentions  []string  `json:"mentions" doc:"Members it notified (user ids)"`
	CreatedAt time.Time `json:"createdAt"`
}

// ComposerThread is a comment and its replies.
type ComposerThread struct {
	ID string `json:"id"`
	// Anchor is the rc-comment id of the block it's on; empty for the
	// whole email. AnchorLabel says what the block was when it started
	// ("the button “Read the docs”").
	Anchor      string            `json:"anchor"`
	AnchorLabel string            `json:"anchorLabel"`
	Comments    []ComposerComment `json:"comments"`
	ResolvedAt  *time.Time        `json:"resolvedAt,omitempty"`
	ResolvedBy  string            `json:"resolvedBy,omitempty"`
}

// ComposerThreadList is a campaign's comments: open threads, then resolved
// ones, each oldest first.
type ComposerThreadList struct {
	Threads  []ComposerThread `json:"threads"`
	Open     int              `json:"open"`
	Resolved int              `json:"resolved"`
}

// ComposerCommentInput starts a thread or replies to one.
type ComposerCommentInput struct {
	Body string `json:"body" minLength:"1" maxLength:"4000"`
	// ThreadID replies to that thread; Anchor and AnchorLabel are then ignored.
	ThreadID    string `json:"threadId,omitempty" format:"uuid"`
	Anchor      string `json:"anchor,omitempty" maxLength:"40" doc:"The block's rc-comment id; empty for the whole email"`
	AnchorLabel string `json:"anchorLabel,omitempty" maxLength:"200"`
	// Mentions are the members to notify (user ids): they get a
	// notification, by email too unless they turned that off.
	Mentions []string `json:"mentions,omitempty" maxItems:"20"`
}

// ComposerResolveInput resolves a thread or opens it again.
type ComposerResolveInput struct {
	Resolved bool `json:"resolved"`
}

// ComposerMember is someone who can be @mentioned.
type ComposerMember struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Color string `json:"color" doc:"Avatar colour"`
}

// ComposerMemberList is who can be @mentioned in a campaign's comments.
type ComposerMemberList struct {
	Items []ComposerMember `json:"items"`
}

// Comments lists a campaign's threads.
func (s *Service) Comments(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ComposerThreadList, error) {
	if _, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id}); err != nil {
		return ComposerThreadList{}, api.MapError(err, "Campaign")
	}
	rows, err := s.Q.CommentsList(ctx, store.CommentsListParams{WorkspaceID: ws.ID, CampaignID: id})
	if err != nil {
		return ComposerThreadList{}, err
	}
	return threadsOf(rows), nil
}

func commentOf(r store.CommentsListRow) ComposerComment {
	c := ComposerComment{ID: r.ID.String(), Body: r.Body, Author: r.AuthorName, AuthorColor: r.AuthorColor, Reviewer: r.ReviewLinkID != nil, Version: int(r.Version), CreatedAt: r.CreatedAt, Mentions: []string{}}
	if r.AuthorID != nil {
		c.AuthorID = r.AuthorID.String()
	}
	if c.Reviewer {
		c.Author = r.ReviewerName
	}
	for _, m := range r.Mentions {
		c.Mentions = append(c.Mentions, m.String())
	}
	return c
}

func threadsOf(rows []store.CommentsListRow) ComposerThreadList {
	byID := map[uuid.UUID]*ComposerThread{}
	var order []uuid.UUID
	for _, r := range rows {
		if r.ThreadID == nil {
			t := &ComposerThread{ID: r.ID.String(), Anchor: r.Anchor, AnchorLabel: r.AnchorLabel, Comments: []ComposerComment{commentOf(r)}, ResolvedAt: r.ResolvedAt, ResolvedBy: r.ResolvedByName}
			byID[r.ID] = t
			order = append(order, r.ID)
		}
	}
	for _, r := range rows {
		if r.ThreadID != nil {
			if t := byID[*r.ThreadID]; t != nil {
				t.Comments = append(t.Comments, commentOf(r))
			}
		}
	}
	out := ComposerThreadList{Threads: []ComposerThread{}}
	for _, resolved := range []bool{false, true} {
		for _, id := range order {
			t := byID[id]
			if (t.ResolvedAt != nil) != resolved {
				continue
			}
			out.Threads = append(out.Threads, *t)
			if resolved {
				out.Resolved++
			} else {
				out.Open++
			}
		}
	}
	return out
}

// excerpt is a comment's first words, for notifications.
func excerpt(body string) string {
	body = strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(body) <= 120 {
		return body
	}
	r := []rune(body)
	return strings.TrimSpace(string(r[:117])) + "…"
}

// AddComment starts a thread (on a block or the whole email) or replies to
// one, and notifies the members it mentions.
func (s *Service) AddComment(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ComposerCommentInput) (ComposerThread, error) {
	l := i18n.LangFrom(ctx)
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return ComposerThread{}, api.Invalid("body.body", mCommentEmpty.Text(l, nil))
	}
	if !anchorRe.MatchString(in.Anchor) {
		return ComposerThread{}, api.Invalid("body.anchor", mCommentAnchor.Text(l, nil))
	}
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerThread{}, api.MapError(err, "Campaign")
	}
	members, err := s.Q.ComposerMentionable(ctx, ws.ID)
	if err != nil {
		return ComposerThread{}, err
	}
	known := map[uuid.UUID]bool{}
	for _, m := range members {
		known[m.ID] = true
	}
	var mentions []uuid.UUID
	for _, raw := range in.Mentions {
		m, err := uuid.Parse(raw)
		if err != nil || !known[m] {
			return ComposerThread{}, api.Invalid("body.mentions", mCommentMention.Text(l, nil))
		}
		if p != nil && m == p.UserID {
			continue
		}
		mentions = append(mentions, m)
	}
	var threadID *uuid.UUID
	anchor, label := in.Anchor, strings.TrimSpace(in.AnchorLabel)
	if in.ThreadID != "" {
		tid, err := uuid.Parse(in.ThreadID)
		if err != nil {
			return ComposerThread{}, api.NotFound("Comment")
		}
		root, err := s.Q.CommentGet(ctx, store.CommentGetParams{WorkspaceID: ws.ID, CampaignID: id, ID: tid})
		if errors.Is(err, pgx.ErrNoRows) || err == nil && root.ThreadID != nil {
			return ComposerThread{}, api.NotFound("Comment")
		}
		if err != nil {
			return ComposerThread{}, err
		}
		threadID, anchor, label = &tid, root.Anchor, root.AnchorLabel
	}
	cid := uuid.Must(uuid.NewV7())
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CommentInsert(ctx, store.CommentInsertParams{ID: cid, WorkspaceID: ws.ID, CampaignID: id, ThreadID: threadID, Anchor: anchor, AnchorLabel: label,
			Body: body, AuthorID: &p.UserID, Version: c.Version, Mentions: nonNilIDs(mentions), CreatedAt: now}); err != nil {
			return err
		}
		root := cid
		if threadID != nil {
			root = *threadID
		}
		for _, m := range mentions {
			n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindCommentMention, Severity: "info",
				Link:        "/w/" + ws.Slug + "/campaigns/" + id.String() + "/edit?panel=comments&thread=" + root.String(),
				Data:        map[string]any{"campaign_id": id.String(), "thread_id": root.String()},
				TitleText:   mNotifyMentionTitle.Keep(i18n.Args{"name": p.Name}),
				SummaryText: mNotifyMentionSummary.Keep(i18n.Args{"name": p.Name, "campaign": c.Name, "excerpt": excerpt(body)}),
				BodyText:    mNotifyMentionBody.Keep(i18n.Args{"name": p.Name, "campaign": c.Name, "excerpt": excerpt(body)})}
			if err := notify.ToUser(ctx, tx, m, n); err != nil {
				return err
			}
		}
		after := map[string]any{"commentId": cid.String(), "anchorLabel": label, "mentions": len(mentions)}
		if threadID != nil {
			after["threadId"] = threadID.String()
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.comment", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name, After: after})
	})
	if err != nil {
		return ComposerThread{}, err
	}
	list, err := s.Comments(ctx, ws, id)
	if err != nil {
		return ComposerThread{}, err
	}
	root := cid.String()
	if threadID != nil {
		root = threadID.String()
	}
	for _, t := range list.Threads {
		if t.ID == root {
			return t, nil
		}
	}
	return ComposerThread{}, api.NotFound("Comment")
}

func nonNilIDs(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

// ResolveThread resolves a thread, or opens it again.
func (s *Service) ResolveThread(ctx context.Context, ws *store.Workspace, p *authz.Principal, id, threadID uuid.UUID, in ComposerResolveInput) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		root, err := q.CommentGet(ctx, store.CommentGetParams{WorkspaceID: ws.ID, CampaignID: id, ID: threadID})
		if errors.Is(err, pgx.ErrNoRows) || err == nil && root.ThreadID != nil {
			return api.NotFound("Comment")
		}
		if err != nil {
			return err
		}
		var at *time.Time
		var by *uuid.UUID
		if in.Resolved {
			now := s.now()
			at, by = &now, &p.UserID
		}
		if _, err := q.CommentResolve(ctx, store.CommentResolveParams{WorkspaceID: ws.ID, CampaignID: id, ID: threadID, ResolvedAt: at, ResolvedBy: by}); err != nil {
			return err
		}
		action := "campaign.comment.resolve"
		if !in.Resolved {
			action = "campaign.comment.reopen"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "campaign", TargetID: id.String(),
			After: map[string]any{"threadId": threadID.String(), "anchorLabel": root.AnchorLabel}})
	})
}

// Mentionable lists the members who can be @mentioned: the workspace's
// people, by name.
func (s *Service) Mentionable(ctx context.Context, ws *store.Workspace) (ComposerMemberList, error) {
	rows, err := s.Q.ComposerMentionable(ctx, ws.ID)
	if err != nil {
		return ComposerMemberList{}, err
	}
	out := ComposerMemberList{Items: make([]ComposerMember, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, ComposerMember{ID: r.ID.String(), Name: r.Name, Role: r.RoleName, Color: r.AvatarColor})
	}
	return out, nil
}
