package transactional

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/netip"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// Mail that arrives through the SMTP relay was rendered by the app: it is
// logged as "Raw (SMTP)" instead of with a template, and goes through the
// same route, suppression rules, opt-outs and log as the API's messages.

// MaxRawRecipients is how many recipients one SMTP message may have.
const MaxRawRecipients = 50

// RawAttachment is a file of a message that arrived by SMTP.
type RawAttachment struct {
	Filename    string
	ContentType string
	ContentID   string
	Content     []byte
}

// RawMessage is a message that arrived through the SMTP relay.
type RawMessage struct {
	From    mail.Address
	ReplyTo *mail.Address
	// Recipients are the envelope's (RCPT TO): each gets its own message.
	Recipients  []string
	Subject     string
	HTML, Text  string
	Attachments []RawAttachment
	// Tags and Category come from X-ReputeCast-Tags and
	// X-ReputeCast-Category (a category's name or ID).
	Tags     []string
	Category string
	IP       *netip.Addr
}

// RawResult is what happened to one recipient.
type RawResult struct {
	Recipient string
	ID        string
	// Status: queued, rejected (suppressed or opted out) or not_delivered
	// (a test-mode credential's message to an address that isn't allowed).
	Status string
	Detail string
}

// SMTPCredential is an authenticated SMTP credential.
type SMTPCredential struct {
	Key       store.TxnKeyRow
	Workspace store.Workspace
}

// Errors of SMTP authentication; their text is the SMTP reply's.
var (
	ErrSMTPAuthFailed      = errors.New("the username or password is wrong: the username is the credential's ID and the password its secret, both shown once when it was created")
	ErrSMTPNotSMTPKey      = errors.New("this key is an API key; create an SMTP credential in Transactional › Setup")
	ErrSMTPKeyRevoked      = errors.New("this SMTP credential was revoked; create a new one in Transactional › Setup")
	ErrSMTPKeyExpired      = errors.New("this SMTP credential has expired; create a new one in Transactional › Setup")
	ErrSMTPCannotSend      = errors.New("this credential doesn't hold transactional:send")
	ErrSMTPWorkspaceGone   = errors.New("the credential's workspace was deleted")
	ErrSMTPCreatorLeft     = errors.New("the person who created this credential is no longer an active member of the workspace; create a new one in Transactional › Setup")
	ErrSMTPCreatorCantSend = errors.New("the person who created this credential no longer holds transactional:send; create a new one in Transactional › Setup")
	// ErrSMTPIntegrationKey: an integration's key (Zapier, Make, n8n,
	// WordPress) only calls its own integration's operations.
	ErrSMTPIntegrationKey = errors.New("this key belongs to an integration, which can only call its own operations; create an SMTP credential in Transactional › Setup")
)

// smtpIntegrationKeyError names the integration a refused key belongs to.
type smtpIntegrationKeyError struct{ name string }

func (e smtpIntegrationKeyError) Error() string {
	return "this key belongs to the " + e.name + " integration, which can only call its own operations; create an SMTP credential in Transactional › Setup"
}

func (e smtpIntegrationKeyError) Unwrap() error { return ErrSMTPIntegrationKey }

// integrationName is how people call an integration's kind.
func integrationName(kind string) string {
	switch kind {
	case "zapier":
		return "Zapier"
	case "make":
		return "Make"
	case "n8n":
		return "n8n"
	case "wordpress":
		return "WordPress"
	}
	return ""
}

// AuthenticateSMTP checks an SMTP credential: the username is the key's ID
// and the password its secret.
func (s *Service) AuthenticateSMTP(ctx context.Context, username, password string, ip *netip.Addr) (SMTPCredential, error) {
	id, err := uuid.Parse(strings.TrimSpace(username))
	if err != nil || !authn.LooksLikeAPIKey(password) {
		return SMTPCredential{}, ErrSMTPAuthFailed
	}
	k, err := s.Q.KeysByToken(ctx, authn.HashToken(password))
	if errors.Is(err, pgx.ErrNoRows) {
		return SMTPCredential{}, ErrSMTPAuthFailed
	}
	if err != nil {
		return SMTPCredential{}, err
	}
	if k.ID != id {
		return SMTPCredential{}, ErrSMTPAuthFailed
	}
	if k.RevokedAt != nil {
		return SMTPCredential{}, ErrSMTPKeyRevoked
	}
	if k.ExpiresAt != nil && !k.ExpiresAt.After(s.now()) {
		return SMTPCredential{}, ErrSMTPKeyExpired
	}
	tk, err := s.Q.TxnKey(ctx, store.TxnKeyParams{ID: k.ID, WorkspaceID: k.WorkspaceID})
	if err != nil {
		return SMTPCredential{}, err
	}
	// Integration keys call their integration's operations only, whatever
	// they hold.
	if tk.IntegrationID != nil {
		if name := integrationName(tk.IntegrationKind); name != "" {
			return SMTPCredential{}, smtpIntegrationKeyError{name: name}
		}
		return SMTPCredential{}, ErrSMTPIntegrationKey
	}
	if !slices.Contains(k.Permissions, string(authz.TransactionalSend)) {
		return SMTPCredential{}, ErrSMTPCannotSend
	}
	// As for API keys, the creator must still be active and hold transactional:send.
	if k.CreatedBy == nil {
		return SMTPCredential{}, ErrSMTPCreatorLeft
	}
	m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: k.WorkspaceID, UserID: *k.CreatedBy})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return SMTPCredential{}, ErrSMTPCreatorLeft
	case err != nil:
		return SMTPCredential{}, err
	case m.Status != "active":
		return SMTPCredential{}, ErrSMTPCreatorLeft
	case !authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions).Has(authz.TransactionalSend):
		return SMTPCredential{}, ErrSMTPCreatorCantSend
	}
	if tk.Preset != "smtp" {
		return SMTPCredential{}, ErrSMTPNotSMTPKey
	}
	ws, err := s.Q.GetWorkspace(ctx, k.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return SMTPCredential{}, ErrSMTPWorkspaceGone
	}
	if err != nil {
		return SMTPCredential{}, err
	}
	_ = s.Q.KeysTouch(ctx, store.KeysTouchParams{ID: k.ID, Ip: ip, Client: "SMTP relay", LastCall: "SMTP AUTH"})
	return SMTPCredential{Key: tk, Workspace: ws}, nil
}

// SendRaw queues a message that arrived by SMTP, one per recipient. An
// error stops the whole message (the SMTP reply says why); a recipient who
// is suppressed or opted out is logged as rejected, as with the API.
func (s *Service) SendRaw(ctx context.Context, cred SMTPCredential, m RawMessage) ([]RawResult, error) {
	ws := &cred.Workspace
	key := cred.Key
	c := caller{key: &key, ip: m.IP, source: "smtp"}
	if len(m.Recipients) == 0 {
		return nil, problemInvalid("rcpt", "The message has no recipient.")
	}
	if len(m.Recipients) > MaxRawRecipients {
		return nil, problemInvalid("rcpt", fmt.Sprintf("Send to at most %d recipients per message.", MaxRawRecipients))
	}
	if err := s.rateLimit(c, len(m.Recipients)); err != nil {
		return nil, err
	}
	q := s.Q
	from, defaultReply, err := s.resolveSender(ctx, q, ws.ID, m.From.String(), nil)
	if err != nil {
		return nil, err
	}
	if m.From.Name != "" {
		from.Name = m.From.Name
	}
	replyTo := m.ReplyTo
	if replyTo == nil && defaultReply != "" {
		if a, err := parseAddress(defaultReply); err == nil {
			replyTo = &a
		}
	}
	lt, err := s.rawCategory(ctx, ws, key, m.Category)
	if err != nil {
		return nil, err
	}
	accepted := s.now()
	size := len(m.HTML) + len(m.Text)
	var files []payloadFile
	var fileMeta []map[string]any
	for _, a := range m.Attachments {
		files = append(files, payloadFile(a))
		fileMeta = append(fileMeta, map[string]any{"filename": a.Filename, "content_type": a.ContentType, "size": len(a.Content), "content_id": a.ContentID})
		size += base64.StdEncoding.EncodedLen(len(a.Content))
	}
	st, err := s.settingsRow(ctx, q, ws.ID)
	if err != nil {
		return nil, err
	}
	tags := cleanTags(m.Tags)
	var preps []*prepared
	for _, rcpt := range m.Recipients {
		email, ok := validEmail(rcpt)
		if !ok {
			return nil, problem(http.StatusUnprocessableEntity, CodeInvalidRecipient, fmt.Sprintf("“%s” isn't a valid email address.", clip(rcpt, 100)))
		}
		p := &prepared{id: newID(), accepted: accepted, rendered: accepted, lt: lt, raw: true, to: mail.Address{Address: email}, from: from, replyTo: replyTo,
			data: map[string]any{}, masked: map[string]any{}, tags: tags, metadata: map[string]string{}, files: files, fileMeta: fileMeta, size: size}
		p.mailbox = routing.MailboxProvider(email)
		if st.LinkContacts {
			if ct, err := q.TxnContactByEmail(ctx, store.TxnContactByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
				id := ct.ID
				p.contactID = &id
				if ct.MailboxProvider != "" && ct.MailboxProvider != routing.Other {
					p.mailbox = ct.MailboxProvider
				}
			}
		}
		p.out = rendered{Subject: m.Subject, HTML: m.HTML, Text: m.Text}
		stored := p.out
		p.stored = &stored
		categoryID := ""
		if lt.CategoryID != nil {
			categoryID = lt.CategoryID.String()
		}
		if p.route, _, err = s.route(ctx, q, ws.ID, email, p.mailbox, categoryID, len(files) > 0, int64(size)); err != nil {
			return nil, err
		}
		p.headers = map[string]string{"Feedback-ID": "raw:" + ws.Slug + ":transactional:reputecast"}
		if lt.CategoryKind == KindOptional {
			links := s.messageLinks(ws, p.id, p.contactID, true)
			p.headers["List-Unsubscribe"] = "<" + links["unsubscribe_url"] + ">"
			p.headers["List-Unsubscribe-Post"] = "List-Unsubscribe=One-Click"
		}
		preps = append(preps, p)
	}
	out := make([]RawResult, 0, len(preps))
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		out = out[:0]
		for _, p := range preps {
			status, res, err := s.sendOne(ctx, tx, ws, c, p)
			if err != nil {
				return err
			}
			r := RawResult{Recipient: p.to.Address, ID: p.id.String(), Status: "queued"}
			switch v := res.(type) {
			case *Problem:
				r.Status, r.Detail = "rejected", v.Detail
			case TransactionalSendResult:
				r.Status, r.Detail = v.Status, v.Detail
			}
			_ = status
			out = append(out, r)
		}
		return nil
	})
	return out, err
}

// rawCategory picks the category of mail that arrived by SMTP: the
// X-ReputeCast-Category header (a name or an ID), else the credential's
// only category.
func (s *Service) rawCategory(ctx context.Context, ws *store.Workspace, key store.TxnKeyRow, header string) (liveTemplate, error) {
	lt := liveTemplate{}
	header = strings.TrimSpace(header)
	cats, err := s.Q.TxnCategories(ctx, ws.ID)
	if err != nil {
		return lt, err
	}
	var pick *store.TxnCategoriesRow
	for i, c := range cats {
		if header != "" && (strings.EqualFold(c.Name, header) || c.ID.String() == strings.ToLower(header)) {
			pick = &cats[i]
		}
	}
	if header != "" && pick == nil {
		return lt, problemInvalid("header.X-ReputeCast-Category", fmt.Sprintf("There is no category “%s” in this workspace. Name one of Transactional › Settings › Categories.", clip(header, 60)))
	}
	if pick == nil && len(key.Categories) == 1 {
		for i, c := range cats {
			if c.ID == key.Categories[0] {
				pick = &cats[i]
			}
		}
	}
	if len(key.Categories) > 0 && (pick == nil || !slices.Contains(key.Categories, pick.ID)) {
		names := []string{}
		for _, c := range cats {
			if slices.Contains(key.Categories, c.ID) {
				names = append(names, c.Name)
			}
		}
		return lt, problem(http.StatusForbidden, CodeTemplateNotAllowed, fmt.Sprintf("The credential “%s” can only send %s; set X-ReputeCast-Category to it.", key.Name, joinAnd(names)))
	}
	if pick != nil {
		id := pick.ID
		lt.CategoryID, lt.CategoryName, lt.CategoryKind = &id, pick.Name, pick.Kind
	}
	return lt, nil
}
