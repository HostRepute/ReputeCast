package inbound

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/optout"
	"github.com/hostrepute/reputecast/internal/store"
)

// Audience › Replies (G-17): what people wrote back, by view.

// Reply views.
const (
	ViewOpen         = "open"
	ViewDone         = "done"
	ViewAuto         = "auto"
	ViewSpam         = "spam"
	ViewOther        = "other"
	ViewUnsubscribes = "unsubscribes"
)

// previewLen is how much of a reply the list shows.
const previewLen = 160

// ReplyListParams filter the replies inbox.
type ReplyListParams struct {
	View         string
	CampaignID   *uuid.UUID
	AutomationID *uuid.UUID
	Query        string
	Cursor       string
	Limit        int
}

// ReplyList is one view of the replies inbox, newest first.
type ReplyList struct {
	Items      []ReplyListItem `json:"items"`
	NextCursor string          `json:"nextCursor,omitempty"`
	Counts     ReplyCounts     `json:"counts"`
	// Receiving says whether the workspace has an inbound route (the empty
	// state differs).
	Receiving bool `json:"receiving"`
	// CanManage: the caller may mark replies and unsubscribe contacts.
	CanManage bool `json:"canManage"`
}

// ReplyCounts are how many messages each view holds, with the same filters.
type ReplyCounts struct {
	Open         int64 `json:"open"`
	Done         int64 `json:"done"`
	Auto         int64 `json:"auto"`
	Spam         int64 `json:"spam"`
	Other        int64 `json:"other"`
	Unsubscribes int64 `json:"unsubscribes"`
}

// ReplyPerson is who wrote.
type ReplyPerson struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ReplySource is the email a reply answers: a campaign or an automation.
type ReplySource struct {
	Type string `json:"type" enum:"campaign,automation"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReplyListItem is one row of the list.
type ReplyListItem struct {
	ID              string       `json:"id"`
	Kind            string       `json:"kind" enum:"reply,unsubscribe,other"`
	From            ReplyPerson  `json:"from"`
	Subject         string       `json:"subject"`
	Preview         string       `json:"preview"`
	ReceivedAt      time.Time    `json:"receivedAt"`
	Status          string       `json:"status" enum:"open,done"`
	AutoReply       bool         `json:"autoReply"`
	Spam            bool         `json:"spam"`
	AsksUnsubscribe bool         `json:"asksUnsubscribe"`
	Source          *ReplySource `json:"source,omitempty"`
	ContactID       string       `json:"contactId,omitempty"`
}

// ReplyAttachment is an attachment as it's kept: its name, type and size.
type ReplyAttachment struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

// ReplyAuth are the receiving server's verdicts.
type ReplyAuth struct {
	SPF   string `json:"spf,omitempty"`
	DKIM  string `json:"dkim,omitempty"`
	DMARC string `json:"dmarc,omitempty"`
}

// ReplyContact is the contact a reply came from.
type ReplyContact struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

// ReplyDetail is one reply, opened.
type ReplyDetail struct {
	ReplyListItem
	To          string            `json:"to"`
	Text        string            `json:"text"`
	HasHTML     bool              `json:"hasHtml"`
	Attachments []ReplyAttachment `json:"attachments"`
	Auth        ReplyAuth         `json:"auth"`
	// SentAt is when the email it answers went out.
	SentAt    *time.Time    `json:"sentAt,omitempty"`
	Contact   *ReplyContact `json:"contact,omitempty"`
	HandledAt *time.Time    `json:"handledAt,omitempty"`
	CanManage bool          `json:"canManage"`
	// CanUnsubscribe: the caller may unsubscribe the contact (also
	// subscribers:edit), with UnsubscribeReason when not.
	CanUnsubscribe    bool     `json:"canUnsubscribe"`
	UnsubscribeReason string   `json:"unsubscribeReason,omitempty"`
	Helpers           []string `json:"helpers"`
}

// ReplyStatusInput marks replies done or open.
type ReplyStatusInput struct {
	IDs    []string `json:"ids" minItems:"1" maxItems:"500"`
	Status string   `json:"status" enum:"open,done"`
}

// ReplyStatusResult is how many replies changed.
type ReplyStatusResult struct {
	Updated int `json:"updated"`
}

// ReplyUnsubscribeResult is the contact after Unsubscribe contact.
type ReplyUnsubscribeResult struct {
	ContactID string `json:"contactId"`
	Already   bool   `json:"already"`
}

func encodeCursor(at time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id.String()))
}

func decodeCursor(c string) (time.Time, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	at, id, ok := strings.Cut(string(b), "|")
	if !ok {
		return time.Time{}, uuid.Nil, fmt.Errorf("bad cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	u, err := uuid.Parse(id)
	return t, u, err
}

// likeEscape keeps a search's % and _ literal in ILIKE.
var likeEscape = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Replies lists one view of the inbox.
func (s *Service) Replies(ctx context.Context, ws *store.Workspace, p ReplyListParams) (ReplyList, error) {
	out := ReplyList{Items: []ReplyListItem{}}
	if p.View == "" {
		p.View = ViewOpen
	}
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 50
	}
	q := likeEscape.Replace(strings.TrimSpace(p.Query))
	if utf8.RuneCountInString(q) > 200 {
		return out, api.Invalid("q", say(ctx, "problem.searchLong", nil))
	}
	lp := store.RepliesListParams{WorkspaceID: ws.ID, View: p.View, CampaignID: p.CampaignID, AutomationID: p.AutomationID, Q: q, Lim: int32(p.Limit + 1)}
	if p.Cursor != "" {
		at, id, err := decodeCursor(p.Cursor)
		if err != nil {
			return out, api.Invalid("cursor", say(ctx, "problem.cursor", nil))
		}
		lp.BeforeAt, lp.BeforeID = &at, id
	}
	rows, err := s.Q.RepliesList(ctx, lp)
	if err != nil {
		return out, err
	}
	if len(rows) > p.Limit {
		last := rows[p.Limit-1]
		out.NextCursor = encodeCursor(last.ReceivedAt, last.ID)
		rows = rows[:p.Limit]
	}
	for _, r := range rows {
		out.Items = append(out.Items, listItem(r.ID, r.Kind, r.FromName, r.FromEmail, r.Subject, r.Preview, r.ReceivedAt, r.Status, r.AutoReply, r.Spam,
			r.AsksUnsubscribe, r.ContactID, r.CampaignID, r.CampaignName, r.AutomationID, r.AutomationName))
	}
	// The chips count each view as it is, whatever the search (RepliesStates).
	c, err := s.Q.RepliesCounts(ctx, store.RepliesCountsParams{WorkspaceID: ws.ID, CampaignID: p.CampaignID, AutomationID: p.AutomationID})
	if err != nil {
		return out, err
	}
	out.Counts = ReplyCounts(c)
	if _, err := s.Q.InboundRouteGet(ctx, ws.ID); err == nil {
		out.Receiving = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	out.CanManage = api.PrincipalFrom(ctx).Can(authz.RepliesManage)
	return out, nil
}

func listItem(id uuid.UUID, kind, fromName, fromEmail, subject, text string, at time.Time, status string, auto, spam, asks bool,
	contact, campaign *uuid.UUID, campaignName string, automation *uuid.UUID, automationName string) ReplyListItem {
	it := ReplyListItem{ID: id.String(), Kind: kind, From: ReplyPerson{Name: fromName, Email: fromEmail}, Subject: subject, Preview: preview(text),
		ReceivedAt: at, Status: status, AutoReply: auto, Spam: spam, AsksUnsubscribe: asks}
	if it.Kind != KindReply && it.Kind != KindUnsubscribe {
		it.Kind = KindOther
	}
	if contact != nil {
		it.ContactID = contact.String()
	}
	switch {
	case campaign != nil:
		it.Source = &ReplySource{Type: "campaign", ID: campaign.String(), Name: campaignName}
	case automation != nil:
		it.Source = &ReplySource{Type: "automation", ID: automation.String(), Name: automationName}
	}
	return it
}

// preview is the start of what someone wrote: quoted lines and the line
// introducing them left out, spaces collapsed.
func preview(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ">") || quoteIntro(line) {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(line)
		if b.Len() > previewLen*4 {
			break
		}
	}
	s := strings.Join(strings.FieldsFunc(b.String(), unicode.IsSpace), " ")
	if utf8.RuneCountInString(s) <= previewLen {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:previewLen-1])) + "…"
}

// quoteIntro is the line a mail client puts before a quote ("On Tue, Acme
// wrote:").
func quoteIntro(line string) bool {
	return strings.HasSuffix(line, "wrote:") || strings.HasSuffix(line, "schrieb:") || strings.HasSuffix(line, "a écrit :") ||
		strings.HasSuffix(line, "escribió:") || strings.HasSuffix(line, "đã viết:")
}

// Reply opens one reply.
func (s *Service) Reply(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReplyDetail, error) {
	r, err := s.Q.ReplyGet(ctx, store.ReplyGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return ReplyDetail{}, api.MapError(err, "Reply")
	}
	if r.Kind != KindReply && r.Kind != KindUnsubscribe && r.Kind != KindOther {
		return ReplyDetail{}, api.NotFound("Reply")
	}
	p := api.PrincipalFrom(ctx)
	out := ReplyDetail{
		ReplyListItem: listItem(r.ID, r.Kind, r.FromName, r.FromEmail, r.Subject, r.TextBody, r.ReceivedAt, r.Status, r.AutoReply, r.Spam,
			r.AsksUnsubscribe, r.ContactID, r.CampaignID, r.CampaignName, r.AutomationID, r.AutomationName),
		To: r.ToAddress, Text: r.TextBody, HasHTML: r.HasHtml, Attachments: []ReplyAttachment{}, SentAt: r.MessageSentAt, HandledAt: r.HandledAt,
		CanManage: p.Can(authz.RepliesManage), Helpers: []string{},
	}
	var atts []attachmentView
	_ = json.Unmarshal(r.Attachments, &atts)
	for _, a := range atts {
		out.Attachments = append(out.Attachments, ReplyAttachment(a))
	}
	var auth authView
	_ = json.Unmarshal(r.Auth, &auth)
	out.Auth = ReplyAuth(auth)
	if r.ContactID != nil && r.ContactEmail != "" {
		out.Contact = &ReplyContact{ID: r.ContactID.String(), Name: strings.TrimSpace(r.ContactFirstName + " " + r.ContactLastName), Email: r.ContactEmail,
			Status: r.ContactStatus}
		switch {
		case !out.CanManage:
			out.UnsubscribeReason = say(ctx, "reason.manageReplies", nil)
			if h := access.Helpers(ctx, s.Q, ws.ID, authz.RepliesManage); h != nil {
				out.Helpers = h
			}
		case !p.Can(authz.SubscribersEdit):
			out.UnsubscribeReason = say(ctx, "reason.editSubscribers", nil)
			if h := access.Helpers(ctx, s.Q, ws.ID, authz.SubscribersEdit); h != nil {
				out.Helpers = h
			}
		default:
			out.CanUnsubscribe = true
		}
	}
	return out, nil
}

// SetStatus marks replies done or open.
func (s *Service) SetStatus(ctx context.Context, ws *store.Workspace, in ReplyStatusInput) (ReplyStatusResult, error) {
	ids := make([]uuid.UUID, 0, len(in.IDs))
	seen := map[uuid.UUID]bool{}
	for _, raw := range in.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return ReplyStatusResult{}, api.Invalid("body.ids", say(ctx, "problem.badIds", nil))
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	var by *uuid.UUID
	if p := api.PrincipalFrom(ctx); p != nil && p.Type == authz.ActorUser {
		u := p.UserID
		by = &u
	}
	var changed []uuid.UUID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		changed, err = q.RepliesSetStatus(ctx, store.RepliesSetStatusParams{WorkspaceID: ws.ID, Ids: ids, Status: in.Status, HandledBy: by, At: s.now()})
		if err != nil || len(changed) == 0 {
			return err
		}
		strs := make([]string, 0, len(changed))
		for _, id := range changed {
			strs = append(strs, id.String())
		}
		target := ""
		if len(changed) == 1 {
			target = changed[0].String()
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "reply.status_updated", TargetType: "inbound_message", TargetID: target,
			Metadata: map[string]any{"status": in.Status, "count": len(changed), "ids": strs}})
	})
	return ReplyStatusResult{Updated: len(changed)}, err
}

// UnsubscribeContact unsubscribes the contact a reply came from, as the
// one-click link of the email it answers would (also needs
// subscribers:edit).
func (s *Service) UnsubscribeContact(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReplyUnsubscribeResult, error) {
	p := api.PrincipalFrom(ctx)
	if !p.Can(authz.SubscribersEdit) {
		return ReplyUnsubscribeResult{}, api.Forbidden(authz.SubscribersEdit, access.Helpers(ctx, s.Q, ws.ID, authz.SubscribersEdit))
	}
	r, err := s.Q.ReplyGet(ctx, store.ReplyGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return ReplyUnsubscribeResult{}, api.MapError(err, "Reply")
	}
	if r.ContactID == nil || r.ContactEmail == "" {
		return ReplyUnsubscribeResult{}, api.Conflict(say(ctx, "problem.noContact", nil))
	}
	ref := optout.MessageRef{ContactID: *r.ContactID, CampaignID: r.CampaignID}
	if r.MessageID != nil {
		ref.MessageID = *r.MessageID
	}
	args := i18n.Args{"email": r.FromEmail}
	ev := audience.PublicEvidence{Source: audience.SourceReply, Note: mUnsubscribedFromReply.Text(i18n.Default, args), NoteText: mUnsubscribedFromReply.Keep(args), At: s.now()}
	var res audience.UnsubscribeResult
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		res, err = optout.Apply(ctx, tx, optout.Deps{Audience: s.Audience, MessageUnsubscribed: s.MessageUnsubscribed}, ws, ref, ev, "", "")
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &ws.ID, Action: "reply.contact_unsubscribed", TargetType: "inbound_message",
			TargetID: r.ID.String(), TargetName: r.Subject, Metadata: map[string]any{"contact_id": r.ContactID.String(), "already": res.Already}})
	})
	if errors.Is(err, audience.ErrContactGone) {
		return ReplyUnsubscribeResult{}, api.NotFound("Contact")
	}
	if err != nil {
		return ReplyUnsubscribeResult{}, err
	}
	return ReplyUnsubscribeResult{ContactID: r.ContactID.String(), Already: res.Already}, nil
}
