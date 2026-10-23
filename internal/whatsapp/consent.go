package whatsapp

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// WhatsApp consent per list (brief §3.9: like SMS, with its own text) and
// the contact profile's WhatsApp card: the consent per list with its
// evidence, the customer service window ("Can get free-form replies until
// 15 Oct 18:02"), the conversation and a reply while the window is open.
// Consent changes by a person are audited; keyword changes are kept in the
// consent evidence.

// WhatsAppContactList is a list's WhatsApp consent.
type WhatsAppContactList struct {
	ListID  string     `json:"listId"`
	Name    string     `json:"name"`
	Status  string     `json:"status" enum:"subscribed,pending,opted_out,not_subscribed"`
	At      *time.Time `json:"at,omitempty"`
	Source  string     `json:"source"`
	Keyword string     `json:"keyword,omitempty"`
	// Page and Version are the evidence's: where and with which text.
	Page    string `json:"page,omitempty"`
	Version string `json:"version,omitempty"`
}

// WhatsAppSuppression is a number that gets no WhatsApp messages.
type WhatsAppSuppression struct {
	Reason  string    `json:"reason"`
	Keyword string    `json:"keyword,omitempty"`
	At      time.Time `json:"at"`
}

// WhatsAppWindow is a number's customer service window.
type WhatsAppWindow struct {
	Open bool `json:"open"`
	// Until is when it closes (or closed): 24 hours after LastAt.
	Until    *time.Time `json:"until,omitempty"`
	LastAt   *time.Time `json:"lastAt,omitempty"`
	LastText string     `json:"lastText,omitempty"`
}

// WhatsAppThreadItem is one message of the conversation.
type WhatsAppThreadItem struct {
	ID        string    `json:"id"`
	Direction string    `json:"direction" enum:"in,out"`
	Kind      string    `json:"kind" enum:"template,text,reply,keyword,button"`
	Text      string    `json:"text"`
	Template  string    `json:"template,omitempty"`
	At        time.Time `json:"at"`
	Status    string    `json:"status,omitempty"`
	Read      bool      `json:"read"`
	Campaign  string    `json:"campaign,omitempty"`
}

// WhatsAppBusiness is who the person talks to.
type WhatsAppBusiness struct {
	DisplayName string `json:"displayName"`
	Number      string `json:"number"`
}

// WhatsAppContactCard is the contact profile's WhatsApp card.
type WhatsAppContactCard struct {
	Connected  bool                  `json:"connected"`
	Name       string                `json:"name"`
	Email      string                `json:"email"`
	Phone      string                `json:"phone"`
	Display    string                `json:"display"`
	Country    string                `json:"country"`
	LineType   string                `json:"lineType"`
	TimeZone   string                `json:"timeZone"`
	Lists      []WhatsAppContactList `json:"lists"`
	Suppressed *WhatsAppSuppression  `json:"suppressed,omitempty"`
	Window     WhatsAppWindow        `json:"window"`
	Thread     []WhatsAppThreadItem  `json:"thread"`
	Business   *WhatsAppBusiness     `json:"business,omitempty"`
	CanEdit    bool                  `json:"canEdit"`
	// CanReply: the window is open, the account sends and the viewer holds
	// whatsapp:send.
	CanReply bool `json:"canReply"`
}

type contactRow struct {
	ID                         uuid.UUID
	Email, FirstName, LastName string
	Zone, Phone, Country, Line string
	Attributes                 []byte
}

func (s *Service) contact(ctx context.Context, ws *store.Workspace, id uuid.UUID) (contactRow, error) {
	var c contactRow
	err := s.Pool.QueryRow(ctx, `SELECT id, email, first_name, last_name, timezone, phone, phone_country, phone_line_type, attributes FROM contacts
		WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL`, id, ws.ID).
		Scan(&c.ID, &c.Email, &c.FirstName, &c.LastName, &c.Zone, &c.Phone, &c.Country, &c.Line, &c.Attributes)
	if err != nil {
		return c, api.NotFound("Contact")
	}
	return c, nil
}

func fullName(first, last string) string { return strings.TrimSpace(first + " " + last) }

// ContactCard is a contact's WhatsApp card.
func (s *Service) ContactCard(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (WhatsAppContactCard, error) {
	c, err := s.contact(ctx, ws, id)
	if err != nil {
		return WhatsAppContactCard{}, err
	}
	out := WhatsAppContactCard{Name: fullName(c.FirstName, c.LastName), Email: c.Email, Phone: c.Phone, Country: c.Country, LineType: c.Line,
		Lists: []WhatsAppContactList{}, Thread: []WhatsAppThreadItem{}, CanEdit: p.Can(authz.SubscribersEdit)}
	if c.Phone != "" {
		out.Display = phone.Display(c.Phone)
		out.TimeZone, _ = phone.RecipientZone(c.Zone, c.Phone)
	}
	a, aerr := s.account(ctx, s.Q, ws)
	out.Connected = aerr == nil
	if aerr == nil {
		out.Business = &WhatsAppBusiness{DisplayName: a.DisplayName, Number: phone.Display(a.PhoneNumber)}
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT l.id, l.name, coalesce(cc.status, 'not_subscribed'), cc.changed_at, coalesce(cc.source, ''), coalesce(cc.keyword, ''),
			coalesce(ev.page, ''), coalesce(ev.consent_version, '')
		FROM lists l
		LEFT JOIN list_memberships lm ON lm.list_id = l.id AND lm.contact_id = $2
		LEFT JOIN channel_consents cc ON cc.list_id = l.id AND cc.contact_id = $2 AND cc.channel = 'whatsapp'
		LEFT JOIN LATERAL (SELECT e.page, e.consent_version FROM channel_consent_events e WHERE e.contact_id = $2 AND e.list_id = l.id AND e.channel = 'whatsapp'
			AND e.action IN ('subscribed','imported','confirmed','opted_in') ORDER BY e.occurred_at DESC LIMIT 1) ev ON true
		WHERE l.workspace_id = $1 AND l.deleted_at IS NULL AND (lm.contact_id IS NOT NULL OR cc.contact_id IS NOT NULL)
		ORDER BY l.created_at, l.name`, ws.ID, c.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var l WhatsAppContactList
		var lid uuid.UUID
		if err := rows.Scan(&lid, &l.Name, &l.Status, &l.At, &l.Source, &l.Keyword, &l.Page, &l.Version); err != nil {
			rows.Close()
			return out, err
		}
		l.ListID = lid.String()
		out.Lists = append(out.Lists, l)
	}
	rows.Close()
	if c.Phone == "" {
		return out, nil
	}
	if sup, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "whatsapp", Phone: c.Phone}); err == nil {
		out.Suppressed = &WhatsAppSuppression{Reason: sup.Reason, Keyword: sup.Keyword, At: sup.CreatedAt}
	}
	now := s.now()
	open, until, last := s.windowOpen(ctx, s.Q, ws, c.Phone, now)
	out.Window = WhatsAppWindow{Open: open, Until: until}
	if last != nil {
		out.Window.LastAt, out.Window.LastText = &last.ReceivedAt, clip(last.Body, 200)
	}
	out.Thread, err = s.thread(ctx, ws, c.Phone, 30)
	if err != nil {
		return out, err
	}
	out.CanReply = open && aerr == nil && a.Status != "token_rejected" && p.Can(authz.WhatsAppSend) && out.Suppressed == nil
	return out, nil
}

// thread is the last messages to and from a number, oldest first.
func (s *Service) thread(ctx context.Context, ws *store.Workspace, e164 string, limit int) ([]WhatsAppThreadItem, error) {
	rows, err := s.Pool.Query(ctx, `SELECT * FROM (
		SELECT m.id, 'out' AS dir, CASE WHEN m.kind = 'reply' THEN 'reply' ELSE w.kind END AS kind, coalesce(m.body, '') AS text, w.template,
			coalesce(m.sent_at, m.send_after, m.created_at) AS at, m.status, w.read_at IS NOT NULL AS read, coalesce(c.name, a.name, '') AS campaign
		FROM channel_messages m JOIN whatsapp_messages w ON w.message_id = m.id
		LEFT JOIN campaigns c ON c.id = m.campaign_id LEFT JOIN automations a ON a.id = m.automation_id
		WHERE m.workspace_id = $1 AND m.channel = 'whatsapp' AND m.recipient = $2 AND m.kind <> 'test' AND m.status NOT IN ('skipped','cancelled')
		UNION ALL
		SELECT i.id, 'in', CASE i.kind WHEN 'reply' THEN 'text' WHEN 'button' THEN 'button' ELSE 'keyword' END, i.body, '', i.received_at, '', false, ''
		FROM whatsapp_inbound i WHERE i.workspace_id = $1 AND i.phone = $2
		ORDER BY at DESC LIMIT $3) t ORDER BY at`, ws.ID, e164, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WhatsAppThreadItem{}
	for rows.Next() {
		var it WhatsAppThreadItem
		var id uuid.UUID
		if err := rows.Scan(&id, &it.Direction, &it.Kind, &it.Text, &it.Template, &it.At, &it.Status, &it.Read, &it.Campaign); err != nil {
			return nil, err
		}
		it.ID = id.String()
		out = append(out, it)
	}
	return out, rows.Err()
}

// WhatsAppConsentEvidence is what a person's consent was given with.
type WhatsAppConsentEvidence struct {
	ConsentText    string    `json:"consentText" maxLength:"1000"`
	ConsentVersion string    `json:"consentVersion,omitempty" maxLength:"40"`
	Source         string    `json:"source" maxLength:"200" doc:"Where it was given: a form, a call, an event"`
	Page           string    `json:"page,omitempty" maxLength:"500"`
	IP             string    `json:"ip,omitempty" maxLength:"45"`
	At             time.Time `json:"at"`
}

// WhatsAppConsentInput changes a contact's WhatsApp consent on a list.
type WhatsAppConsentInput struct {
	ListID   string                   `json:"listId" format:"uuid"`
	Status   string                   `json:"status" enum:"subscribed,not_subscribed"`
	Evidence *WhatsAppConsentEvidence `json:"evidence,omitempty"`
	Note     string                   `json:"note,omitempty" maxLength:"500"`
}

// SetConsentByTeam changes WhatsApp consent from the profile or the API
// (subscribers:edit), audited. Consent needs its evidence; an opt-out by
// keyword can't be overridden: only the person can come back.
func (s *Service) SetConsentByTeam(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WhatsAppConsentInput) (WhatsAppContactCard, error) {
	c, err := s.contact(ctx, ws, id)
	if err != nil {
		return WhatsAppContactCard{}, err
	}
	l := langOf(ctx)
	listID, err := uuid.Parse(in.ListID)
	if err != nil {
		return WhatsAppContactCard{}, api.Invalid("body.listId", mPickList.Text(l, nil))
	}
	var listName string
	if err := s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, listID, ws.ID).Scan(&listName); err != nil {
		return WhatsAppContactCard{}, api.Invalid("body.listId", mPickList.Text(l, nil))
	}
	now := s.now()
	ev := store.ChanConsentEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ContactID: c.ID, ListID: &listID, Channel: "whatsapp", Phone: c.Phone,
		Source: "team", ActorID: uuidPtr(p.UserID), Note: in.Note, OccurredAt: now}
	if !p.IsHuman() {
		ev.Source = "api"
	}
	status := in.Status
	switch status {
	case "subscribed":
		if c.Phone == "" {
			return WhatsAppContactCard{}, api.Invalid("body.status", mPhoneFirst.Text(l, nil))
		}
		e := in.Evidence
		if e == nil || strings.TrimSpace(e.ConsentText) == "" || strings.TrimSpace(e.Source) == "" || e.At.IsZero() {
			return WhatsAppContactCard{}, api.Invalid("body.evidence", mEvidence.Text(l, nil))
		}
		if e.At.After(now.Add(time.Minute)) {
			return WhatsAppContactCard{}, api.Invalid("body.evidence.at", mFutureConsent.Text(l, nil))
		}
		if sup, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "whatsapp", Phone: c.Phone}); err == nil &&
			sup.Reason == "opted_out" {
			return WhatsAppContactCard{}, api.Conflict(mOptedOutStart.Text(l, nil))
		}
		ev.Action, ev.Source, ev.Page, ev.ConsentText, ev.ConsentVersion, ev.OccurredAt = "subscribed", strings.TrimSpace(e.Source), e.Page, e.ConsentText,
			e.ConsentVersion, e.At
		if a, err := netip.ParseAddr(e.IP); err == nil {
			ev.Ip = &a
		}
	default:
		status, ev.Action = "not_subscribed", "unsubscribed"
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var before string
		_ = tx.QueryRow(ctx, `SELECT status FROM channel_consents WHERE list_id = $1 AND contact_id = $2 AND channel = 'whatsapp'`, listID, c.ID).Scan(&before)
		if err := q.ChanConsentUpsert(ctx, store.ChanConsentUpsertParams{WorkspaceID: ws.ID, ListID: listID, ContactID: c.ID, Channel: "whatsapp", Status: status,
			Source: ev.Source, ChangedAt: ev.OccurredAt}); err != nil {
			return err
		}
		if err := q.ChanConsentEventInsert(ctx, ev); err != nil {
			return err
		}
		if before == "" {
			before = "not_subscribed"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact.whatsapp_consent_changed", TargetType: "contact", TargetID: c.ID.String(),
			TargetName: c.Email, Reason: in.Note, Before: map[string]any{"list": listName, "whatsapp": before},
			After: map[string]any{"list": listName, "whatsapp": status, "source": ev.Source, "consent_version": ev.ConsentVersion}})
	})
	if err != nil {
		return WhatsAppContactCard{}, err
	}
	return s.ContactCard(ctx, ws, p, id)
}

// WhatsAppReplyInput is a free-form reply inside the window.
type WhatsAppReplyInput struct {
	Text string `json:"text" minLength:"1" maxLength:"4096"`
}

// Reply sends free text to a contact who wrote in the last 24 hours. Like
// every message it stays in the message log, not the audit log.
func (s *Service) Reply(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WhatsAppReplyInput) (WhatsAppThreadItem, error) {
	c, err := s.contact(ctx, ws, id)
	if err != nil {
		return WhatsAppThreadItem{}, err
	}
	l := langOf(ctx)
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return WhatsAppThreadItem{}, api.Conflict(mConnectFirst.Text(l, nil))
	}
	if err != nil {
		return WhatsAppThreadItem{}, err
	}
	if a.Status == "token_rejected" {
		return WhatsAppThreadItem{}, api.Conflict(mTokenReplaceFirst.Text(l, nil))
	}
	text := strings.TrimSpace(in.Text)
	if text == "" || c.Phone == "" {
		return WhatsAppThreadItem{}, api.Invalid("body.text", mWriteReply.Text(l, nil))
	}
	now := s.now()
	open, until, _ := s.windowOpen(ctx, s.Q, ws, c.Phone, now)
	if !open {
		msg := mWindowClosed.Text(l, nil)
		if until != nil {
			at := until.In(wsLocation(ws))
			msg = mWindowClosedAt.Text(l, i18n.Args{"date": dayMonth(l, at), "time": clock(l, at)})
		}
		return WhatsAppThreadItem{}, api.Err(409, "window_closed", msg)
	}
	if _, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "whatsapp", Phone: c.Phone}); err == nil {
		return WhatsAppThreadItem{}, api.Conflict(mOptedOut.Text(l, nil))
	}
	mid := uuid.Must(uuid.NewV7())
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		return insertMessage(ctx, q, ws, a.ID, outgoing{id: mid, kind: "reply", contactID: &c.ID, recipient: c.Phone, country: c.Country, body: text,
			status: "queued", after: &now}, now)
	})
	if err != nil {
		return WhatsAppThreadItem{}, err
	}
	s.kick(ctx)
	return WhatsAppThreadItem{ID: mid.String(), Direction: "out", Kind: "reply", Text: text, At: now, Status: "queued"}, nil
}

// WhatsAppListConsent is a list's people with WhatsApp consent.
type WhatsAppListConsent struct {
	ListID  string `json:"listId"`
	Name    string `json:"name"`
	Consent int64  `json:"consent"`
	Members int64  `json:"members"`
}

// listConsent counts WhatsApp consent per list, most first.
func (s *Service) listConsent(ctx context.Context, ws *store.Workspace) ([]WhatsAppListConsent, int64, error) {
	rows, err := s.Pool.Query(ctx, `SELECT l.id, l.name,
			(SELECT count(*) FROM channel_consents cc JOIN contacts c ON c.id = cc.contact_id AND c.erased_at IS NULL AND c.phone <> ''
				WHERE cc.list_id = l.id AND cc.channel = 'whatsapp' AND cc.status = 'subscribed'),
			(SELECT count(*) FROM list_memberships lm WHERE lm.list_id = l.id AND lm.status = 'subscribed')
		FROM lists l WHERE l.workspace_id = $1 AND l.deleted_at IS NULL ORDER BY l.created_at DESC, l.name`, ws.ID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []WhatsAppListConsent{}
	for rows.Next() {
		var l WhatsAppListConsent
		var id uuid.UUID
		if err := rows.Scan(&id, &l.Name, &l.Consent, &l.Members); err != nil {
			return nil, 0, err
		}
		l.ListID = id.String()
		out = append(out, l)
	}
	var total int64
	err = s.Pool.QueryRow(ctx, `SELECT count(DISTINCT cc.contact_id) FROM channel_consents cc JOIN contacts c ON c.id = cc.contact_id AND c.erased_at IS NULL
		AND c.phone <> '' WHERE cc.workspace_id = $1 AND cc.channel = 'whatsapp' AND cc.status = 'subscribed'`, ws.ID).Scan(&total)
	return out, total, err
}
