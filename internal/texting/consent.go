package texting

import (
	"context"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// Phone numbers and SMS consent (brief §3.3). A contact has one number,
// in E.164, with its country and line type. SMS consent is kept per list,
// apart from email consent: an email unsubscribe never touches texts and a
// STOP never touches email. Every change keeps its evidence.

// Evidence is what a consent change is recorded with.
type Evidence struct {
	// Source is how it happened: form:<slug>, preference_center,
	// account_page, import:<file>, api, team.
	Source         string
	Page           string
	ConsentText    string
	ConsentVersion string
	IP             string
	UserAgent      string
	ActorID        *uuid.UUID
	ImportID       *uuid.UUID
	Note           string
	At             time.Time
}

// SmsConsentEvidence is the evidence the API asks for: the exact consent
// text, where it was given and when.
type SmsConsentEvidence struct {
	Source         string    `json:"source" minLength:"1" maxLength:"200" doc:"Where consent was given: a form, an event, a call."`
	Page           string    `json:"page,omitempty" maxLength:"500"`
	ConsentText    string    `json:"consentText" minLength:"1" maxLength:"1000" doc:"The exact text the person agreed to."`
	ConsentVersion string    `json:"consentVersion,omitempty" maxLength:"40"`
	IP             string    `json:"ip,omitempty" maxLength:"45"`
	At             time.Time `json:"at" doc:"When consent was given."`
}

// SmsContactList is a list's consent on each channel.
type SmsContactList struct {
	ListID string `json:"listId"`
	Name   string `json:"name"`
	// Email is the email consent: subscribed, pending, unsubscribed or
	// not_subscribed.
	Email       string     `json:"email" enum:"subscribed,pending,unsubscribed,not_subscribed"`
	EmailAt     *time.Time `json:"emailAt,omitempty"`
	EmailSource string     `json:"emailSource,omitempty"`
	// EmailSourceLabel names the source ("Brevo import", "preference center").
	EmailSourceLabel string `json:"emailSourceLabel,omitempty"`
	// EmailConfirmed is true once the person confirmed: a double opt-in, or
	// a list added in the preference center (as the profile's card says).
	EmailConfirmed bool       `json:"emailConfirmed,omitempty"`
	SMS            string     `json:"sms" enum:"subscribed,pending,opted_out,not_subscribed"`
	SMSAt          *time.Time `json:"smsAt,omitempty"`
	SMSSource      string     `json:"smsSource,omitempty"`
	SMSPage        string     `json:"smsPage,omitempty"`
	SMSVersion     string     `json:"smsVersion,omitempty"`
	SMSIP          string     `json:"smsIp,omitempty"`
	SMSKeyword     string     `json:"smsKeyword,omitempty"`
	// Confirmation is the opt-in confirmation text's status, when one went.
	Confirmation   string     `json:"confirmation,omitempty"`
	ConfirmationAt *time.Time `json:"confirmationAt,omitempty"`
	// WhatsApp is the WhatsApp consent (X22), kept per list like SMS, with
	// its latest evidence: the page and the consent text's version, or the
	// keyword of an opt-out.
	WhatsApp        string     `json:"whatsApp" enum:"subscribed,pending,opted_out,not_subscribed"`
	WhatsAppAt      *time.Time `json:"whatsAppAt,omitempty"`
	WhatsAppSource  string     `json:"whatsAppSource,omitempty"`
	WhatsAppPage    string     `json:"whatsAppPage,omitempty"`
	WhatsAppVersion string     `json:"whatsAppVersion,omitempty"`
	WhatsAppKeyword string     `json:"whatsAppKeyword,omitempty"`
}

// SmsConsentEvent is one change of SMS consent, with its evidence.
type SmsConsentEvent struct {
	ID             string    `json:"id"`
	At             time.Time `json:"at"`
	Action         string    `json:"action" enum:"subscribed,pending,confirmed,opted_out,opted_in,unsubscribed,imported,confirmation_sent,confirmation_failed"`
	List           string    `json:"list,omitempty"`
	Number         string    `json:"number"`
	Source         string    `json:"source"`
	Page           string    `json:"page,omitempty"`
	ConsentText    string    `json:"consentText,omitempty"`
	ConsentVersion string    `json:"consentVersion,omitempty"`
	Keyword        string    `json:"keyword,omitempty"`
	IP             string    `json:"ip,omitempty"`
	Actor          string    `json:"actor,omitempty"`
	Note           string    `json:"note,omitempty"`
}

// SmsContactChannels is a contact's phone and consent per list per channel.
type SmsContactChannels struct {
	Phone      string `json:"phone"`
	Display    string `json:"display"`
	Country    string `json:"country"`
	LineType   string `json:"lineType" enum:"mobile,landline,toll_free,voip,unknown,"`
	LineSource string `json:"lineSource" enum:"number_plan,provider,receipt,"`
	TimeZone   string `json:"timeZone"`
	ZoneSource string `json:"zoneSource" enum:"contact,area_code,country,unknown"`
	// Shared counts the other contacts with the same number: a STOP from it
	// applies to all of them.
	Shared     int               `json:"shared"`
	Suppressed *SmsSuppression   `json:"suppressed,omitempty"`
	Lists      []SmsContactList  `json:"lists"`
	Events     []SmsConsentEvent `json:"events"`
	CanEdit    bool              `json:"canEdit"`
	// SMSSetUp is false until an SMS provider is connected.
	SMSSetUp bool `json:"smsSetUp"`
	// WhatsAppSetUp is true once WhatsApp is connected: the card shows its
	// column then, or while a list still has WhatsApp consent.
	WhatsAppSetUp bool `json:"whatsAppSetUp"`
}

func (s *Service) contact(ctx context.Context, ws *store.Workspace, id uuid.UUID) (phoneRow, error) {
	var c phoneRow
	err := s.Pool.QueryRow(ctx, `SELECT id, phone, phone_country, phone_line_type, phone_line_source, timezone, first_name, last_name, email
		FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL`, id, ws.ID).
		Scan(&c.ID, &c.Phone, &c.Country, &c.LineType, &c.LineSource, &c.Zone, &c.FirstName, &c.LastName, &c.Email)
	if err != nil {
		return c, api.MapError(err, "Contact")
	}
	return c, nil
}

type phoneRow struct {
	ID                         uuid.UUID
	Phone, Country, LineType   string
	LineSource, Zone           string
	FirstName, LastName, Email string
}

// ContactChannels is the contact profile's "Consent per list" card.
func (s *Service) ContactChannels(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SmsContactChannels, error) {
	c, err := s.contact(ctx, ws, id)
	if err != nil {
		return SmsContactChannels{}, err
	}
	out := SmsContactChannels{Phone: c.Phone, Country: c.Country, LineType: c.LineType, LineSource: c.LineSource, Lists: []SmsContactList{},
		Events: []SmsConsentEvent{}, CanEdit: p.Can(authz.SubscribersEdit)}
	if c.Phone != "" {
		out.Display = displayPhone(c.Phone)
		zone, src := phone.RecipientZone(c.Zone, c.Phone)
		out.TimeZone, out.ZoneSource = zone, string(src)
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) - 1 FROM contacts WHERE workspace_id = $1 AND phone = $2 AND erased_at IS NULL`, ws.ID, c.Phone).Scan(&out.Shared)
		out.Shared = max(out.Shared, 0)
		if sup, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "sms", Phone: c.Phone}); err == nil {
			out.Suppressed = &SmsSuppression{ID: sup.ID.String(), Number: maskPhone(sup.Phone), Reason: sup.Reason, Code: sup.OriginalCode, Source: sup.Source,
				At: sup.CreatedAt, Context: sup.Context, Removable: sup.Reason != "opted_out"}
		}
	} else {
		out.ZoneSource = string(phone.ZoneUnknown)
		out.TimeZone = c.Zone
	}
	var n int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM sms_providers WHERE workspace_id = $1 AND removed_at IS NULL`, ws.ID).Scan(&n)
	out.SMSSetUp = n > 0
	if i, ok := channels.Lookup(channels.WhatsApp); ok && i.SetUp != nil {
		out.WhatsAppSetUp, _ = i.SetUp(ctx, ws.ID)
	}
	// Every list, so consent can be recorded on any of them: lists with
	// SMS consent first, then the ones the address is on.
	rows, err := s.Pool.Query(ctx, `
		SELECT l.id, l.name, coalesce(lm.status, 'not_subscribed'), coalesce(lm.subscribed_at, lm.unsubscribed_at, lm.created_at), coalesce(lm.source, ''),
			coalesce(cc.status, 'not_subscribed'), cc.changed_at, coalesce(cc.source, ''), coalesce(cc.keyword, ''),
			EXISTS (SELECT 1 FROM consent_events ce WHERE ce.contact_id = $2 AND ce.list_id = l.id AND (ce.action = 'confirmed' OR ce.source = 'preference_center')),
			coalesce(cw.status, 'not_subscribed'), cw.changed_at, coalesce(cw.source, ''), coalesce(cw.keyword, '')
		FROM lists l
		LEFT JOIN list_memberships lm ON lm.list_id = l.id AND lm.contact_id = $2
		LEFT JOIN channel_consents cc ON cc.list_id = l.id AND cc.contact_id = $2 AND cc.channel = 'sms'
		LEFT JOIN channel_consents cw ON cw.list_id = l.id AND cw.contact_id = $2 AND cw.channel = 'whatsapp'
		WHERE l.workspace_id = $1 AND l.deleted_at IS NULL
		ORDER BY coalesce(cc.status = 'subscribed', false) DESC, coalesce(cw.status = 'subscribed', false) DESC, (lm.contact_id IS NOT NULL) DESC, l.created_at, l.name`, ws.ID, c.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var l SmsContactList
		var lid uuid.UUID
		if err := rows.Scan(&lid, &l.Name, &l.Email, &l.EmailAt, &l.EmailSource, &l.SMS, &l.SMSAt, &l.SMSSource, &l.SMSKeyword, &l.EmailConfirmed,
			&l.WhatsApp, &l.WhatsAppAt, &l.WhatsAppSource, &l.WhatsAppKeyword); err != nil {
			rows.Close()
			return out, err
		}
		l.ListID = lid.String()
		l.EmailConfirmed = l.EmailConfirmed && l.Email == "subscribed"
		if l.EmailSource != "" {
			l.EmailSourceLabel = emailSourceLabel(l.EmailSource)
		}
		out.Lists = append(out.Lists, l)
	}
	rows.Close()
	events, err := s.Q.ChanConsentEvents(ctx, c.ID)
	if err != nil {
		return out, err
	}
	names := map[uuid.UUID]string{}
	for _, l := range out.Lists {
		if id, err := uuid.Parse(l.ListID); err == nil {
			names[id] = l.Name
		}
	}
	for _, e := range events {
		if e.Channel == "whatsapp" && e.ListID != nil {
			// The list's latest WhatsApp evidence, for its column.
			for i := range out.Lists {
				l := &out.Lists[i]
				if l.ListID != e.ListID.String() || l.WhatsAppPage != "" || l.WhatsAppVersion != "" {
					continue
				}
				switch e.Action {
				case "subscribed", "pending", "imported", "confirmed", "opted_in":
					l.WhatsAppPage, l.WhatsAppVersion = e.Page, e.ConsentVersion
				}
			}
		}
		if e.Channel != "sms" {
			continue
		}
		ev := SmsConsentEvent{ID: e.ID.String(), At: e.OccurredAt, Action: e.Action, Number: maskPhone(e.Phone), Source: e.Source, Page: e.Page, ConsentText: e.ConsentText,
			ConsentVersion: e.ConsentVersion, Keyword: e.Keyword, Actor: e.ActorName, Note: e.Note}
		if e.Phone == "" {
			ev.Number = ""
		}
		if e.Ip != nil {
			ev.IP = e.Ip.String()
		}
		if e.ListID != nil {
			ev.List = names[*e.ListID]
		}
		out.Events = append(out.Events, ev)
		// The list's latest evidence, as the card shows it.
		for i := range out.Lists {
			l := &out.Lists[i]
			if e.ListID == nil || l.ListID != e.ListID.String() {
				continue
			}
			switch e.Action {
			case "subscribed", "pending", "imported", "confirmed", "opted_in":
				if l.SMSPage == "" && l.SMSVersion == "" {
					l.SMSPage, l.SMSVersion, l.SMSIP = e.Page, e.ConsentVersion, ev.IP
				}
			}
		}
	}
	// The opt-in confirmation's status, on the lists it was for.
	if c.Phone != "" {
		var status string
		var at time.Time
		var list *uuid.UUID
		if err := s.Pool.QueryRow(ctx, `SELECT status, coalesce(delivered_at, sent_at, created_at), list_id FROM channel_messages
			WHERE contact_id = $1 AND kind = 'confirmation' ORDER BY created_at DESC LIMIT 1`, c.ID).Scan(&status, &at, &list); err == nil {
			for i := range out.Lists {
				if list == nil || out.Lists[i].ListID == list.String() {
					out.Lists[i].Confirmation, out.Lists[i].ConfirmationAt = status, &at
				}
			}
		}
	}
	return out, nil
}

// SmsPhoneInput sets a contact's number.
type SmsPhoneInput struct {
	// Phone in any common format; empty removes it.
	Phone string `json:"phone" maxLength:"40"`
	// Country is used when the number has no country code.
	Country string `json:"country,omitempty" pattern:"^[A-Z]{2}$"`
}

// SetPhone sets or removes a contact's number, audited. Removing it leaves
// SMS consent as it is: there's just nothing to text.
func (s *Service) SetPhone(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsPhoneInput) (SmsContactChannels, error) {
	c, err := s.contact(ctx, ws, id)
	if err != nil {
		return SmsContactChannels{}, err
	}
	var n phone.Number
	if strings.TrimSpace(in.Phone) != "" {
		n, err = phone.Parse(in.Phone, in.Country)
		if err != nil {
			return SmsContactChannels{}, api.Invalid("body.phone", "That isn't a phone number. Use the international format, like +1 312 555 0123, or pick the country.")
		}
	}
	now := s.now()
	line, src := string(phone.Unknown), ""
	if n.E164 != "" {
		line, src = string(n.Type), "number_plan"
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.ChanSetPhone(ctx, store.ChanSetPhoneParams{Phone: n.E164, PhoneCountry: n.Country, PhoneLineType: line, PhoneLineSource: src, At: &now,
			ID: c.ID, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact.phone_changed", TargetType: "contact", TargetID: c.ID.String(), TargetName: c.Email,
			Before: map[string]any{"phone": maskOrEmpty(c.Phone)}, After: map[string]any{"phone": maskOrEmpty(n.E164), "country": n.Country}})
	})
	if err != nil {
		return SmsContactChannels{}, err
	}
	return s.ContactChannels(ctx, ws, p, id)
}

func maskOrEmpty(e164 string) string {
	if e164 == "" {
		return ""
	}
	return maskPhone(e164)
}

// SmsConsentInput changes a contact's SMS consent on a list. Giving
// consent needs its evidence.
type SmsConsentInput struct {
	ListID   string              `json:"listId" format:"uuid"`
	Status   string              `json:"status" enum:"subscribed,not_subscribed"`
	Evidence *SmsConsentEvidence `json:"evidence,omitempty"`
	Note     string              `json:"note,omitempty" maxLength:"500"`
}

// SetConsentByTeam changes SMS consent from the profile or the API
// (subscribers:edit), audited. Consent needs evidence; a text opt-out
// can't be overridden: only the person can come back, by texting START.
func (s *Service) SetConsentByTeam(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsConsentInput) (SmsContactChannels, error) {
	c, err := s.contact(ctx, ws, id)
	if err != nil {
		return SmsContactChannels{}, err
	}
	listID, err := uuid.Parse(in.ListID)
	if err != nil {
		return SmsContactChannels{}, api.Invalid("body.listId", "Pick a list of this workspace.")
	}
	var listName string
	if err := s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, listID, ws.ID).Scan(&listName); err != nil {
		return SmsContactChannels{}, api.Invalid("body.listId", "Pick a list of this workspace.")
	}
	ev := Evidence{Source: "team", ActorID: uuidPtr(p.UserID), Note: in.Note, At: s.now()}
	if !p.IsHuman() {
		ev.Source = "api"
	}
	if in.Status == "subscribed" {
		if c.Phone == "" {
			return SmsContactChannels{}, api.Invalid("body.status", "Add a phone number first.")
		}
		if in.Evidence == nil || strings.TrimSpace(in.Evidence.ConsentText) == "" || strings.TrimSpace(in.Evidence.Source) == "" || in.Evidence.At.IsZero() {
			return SmsContactChannels{}, api.Invalid("body.evidence", "SMS consent needs its evidence: the exact consent text, where it was given and when.")
		}
		if in.Evidence.At.After(s.now().Add(time.Minute)) {
			return SmsContactChannels{}, api.Invalid("body.evidence.at", "Consent can't be given in the future.")
		}
		if sup, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "sms", Phone: c.Phone}); err == nil && sup.Reason == "opted_out" {
			return SmsContactChannels{}, api.Conflict("This number opted out by text. Only the person can come back, by texting START.")
		}
		ev.Source = strings.TrimSpace(in.Evidence.Source)
		ev.Page, ev.ConsentText, ev.ConsentVersion, ev.IP, ev.At = in.Evidence.Page, in.Evidence.ConsentText, in.Evidence.ConsentVersion, in.Evidence.IP, in.Evidence.At
	}
	var confirm bool
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var before string
		_ = tx.QueryRow(ctx, `SELECT status FROM channel_consents WHERE list_id = $1 AND contact_id = $2 AND channel = 'sms'`, listID, c.ID).Scan(&before)
		var err error
		if confirm, err = s.SetConsent(ctx, q, ws, c.ID, c.Phone, listID, in.Status, ev); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact.sms_consent_changed", TargetType: "contact", TargetID: c.ID.String(),
			TargetName: c.Email, Reason: in.Note, Before: map[string]any{"list": listName, "sms": firstNonEmpty(before, "not_subscribed")},
			After: map[string]any{"list": listName, "sms": in.Status, "source": ev.Source, "consent_version": ev.ConsentVersion}})
	})
	if err != nil {
		return SmsContactChannels{}, err
	}
	if confirm {
		s.SendConfirmation(ctx, ws, c.ID, listID)
	}
	return s.ContactChannels(ctx, ws, p, id)
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

// SetConsent records a consent change with its evidence, in the caller's
// transaction; it reports whether an opt-in confirmation should be texted
// (after the transaction commits, with SendConfirmation). Forms, the
// preference center, imports and the API all go through it.
func (s *Service) SetConsent(ctx context.Context, q *store.Queries, ws *store.Workspace, contactID uuid.UUID, ph string, listID uuid.UUID, status string,
	ev Evidence) (bool, error) {
	if ev.At.IsZero() {
		ev.At = s.now()
	}
	r, err := s.rules(ctx, q, ws)
	if err != nil {
		return false, err
	}
	action := status
	switch status {
	case "subscribed":
		if r.RequireYes && r.ConfirmationOn {
			status, action = "pending", "pending"
		}
		if strings.HasPrefix(ev.Source, "import") {
			action = "imported"
		}
	case "not_subscribed", "unsubscribed":
		status, action = "not_subscribed", "unsubscribed"
	case "opted_out":
		action = "opted_out"
	}
	var ip *netip.Addr
	if a, err := netip.ParseAddr(ev.IP); err == nil {
		ip = &a
	}
	if err := q.ChanConsentUpsert(ctx, store.ChanConsentUpsertParams{WorkspaceID: ws.ID, ListID: listID, ContactID: contactID, Channel: "sms", Status: status,
		Source: ev.Source, ChangedAt: ev.At}); err != nil {
		return false, err
	}
	if err := q.ChanConsentEventInsert(ctx, store.ChanConsentEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ContactID: contactID, ListID: &listID,
		Channel: "sms", Action: action, Phone: ph, Source: ev.Source, Page: ev.Page, ConsentText: ev.ConsentText, ConsentVersion: ev.ConsentVersion, Ip: ip,
		UserAgent: ev.UserAgent, ActorID: ev.ActorID, ImportID: ev.ImportID, Note: ev.Note, OccurredAt: ev.At}); err != nil {
		return false, err
	}
	// Imports never text: their consent is old news to the person.
	return r.ConfirmationOn && ph != "" && (status == "subscribed" || status == "pending") && !strings.HasPrefix(ev.Source, "import"), nil
}

// SendConfirmation queues the opt-in confirmation text: once per number and
// list, inside the person's window, held when no sender can text their
// country yet.
func (s *Service) SendConfirmation(ctx context.Context, ws *store.Workspace, contactID, listID uuid.UUID) {
	c, err := s.contact(ctx, ws, contactID)
	if err != nil || c.Phone == "" {
		return
	}
	var n int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE workspace_id = $1 AND kind = 'confirmation' AND recipient = $2 AND list_id = $3
		AND status NOT IN ('failed','cancelled','skipped')`, ws.ID, c.Phone, listID).Scan(&n)
	if n > 0 {
		return
	}
	if _, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "sms", Phone: c.Phone}); err == nil {
		return
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return
	}
	r := Recipient{ContactID: c.ID, ListID: listID, Phone: c.Phone, Country: c.Country, FirstName: c.FirstName, LastName: c.LastName, Email: c.Email, Zone: c.Zone}
	m, err := s.queueText(ctx, ws, g, queuedText{kind: "confirmation", contactID: &c.ID, listID: &listID, recipient: r, text: Text{Body: g.rules.Confirmation},
		hold: true})
	if err != nil {
		s.log().WarnContext(ctx, "sms confirmation", "contact", c.ID, "err", err)
		return
	}
	_ = s.Q.ChanConsentEventInsert(ctx, store.ChanConsentEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ContactID: c.ID, ListID: &listID,
		Channel: "sms", Action: "confirmation_sent", Phone: c.Phone, Source: "reputecast", Page: "Opt-in confirmation", Note: m.Status, OccurredAt: s.now()})
}

// confirmationFailed records a confirmation text that failed; the number
// is suppressed already when the failure says it can't get texts.
func (s *Service) confirmationFailed(ctx context.Context, ws *store.Workspace, m store.ChannelMessage, reason string) {
	_ = s.Q.ChanConsentEventInsert(ctx, store.ChanConsentEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ContactID: *m.ContactID, ListID: m.ListID,
		Channel: "sms", Action: "confirmation_failed", Phone: m.Recipient, Source: "reputecast", Page: "Opt-in confirmation", Note: reason, OccurredAt: s.now()})
}

// queuedText is a text for one person, queued for their window.
type queuedText struct {
	kind         string
	campaignID   *uuid.UUID
	automationID *uuid.UUID
	runID        *uuid.UUID
	step         string
	contactID    *uuid.UUID
	listID       *uuid.UUID
	recipient    Recipient
	text         Text
	name         string
	// hold keeps the text when no sender can text the country yet
	// (confirmations wait for a registration); otherwise it's skipped.
	hold bool
	// skipOutside skips the text when the person is outside their window
	// instead of waiting for it (the automation step's choice).
	skipOutside bool
}

// queueText renders and queues a text: its route and price, its window in
// the person's time zone, its short links.
func (s *Service) queueText(ctx context.Context, ws *store.Workspace, g *graph, t queuedText) (store.ChannelMessage, error) {
	var m store.ChannelMessage
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var err error
		m, err = s.queueTextTx(ctx, q, tx, ws, g, t)
		return err
	})
	return m, err
}

// queueTextTx queues a text in the caller's transaction (an automation
// run's): the dispatcher is woken in it too.
func (s *Service) queueTextTx(ctx context.Context, q *store.Queries, tx pgx.Tx, ws *store.Workspace, g *graph, t queuedText) (store.ChannelMessage, error) {
	n, err := phone.Parse(t.recipient.Phone, "")
	if err != nil {
		return store.ChannelMessage{}, api.Invalid("phone", "That isn't a phone number ReputeCast can text.")
	}
	now := s.now()
	zone, zsrc := phone.RecipientZone(t.recipient.Zone, n.E164)
	loc := location(zone)
	w := g.rules.Window
	d := g.decide(n.Country)
	id := uuid.Must(uuid.NewV7())
	status, reason := "queued", ""
	var sendAfter, expires *time.Time
	var providerID, senderID *uuid.UUID
	sender := ""
	switch {
	case d.Sender == nil && t.hold:
		status, reason = "held", "sender_"+d.Hold
		if d.Hold == "no_route" {
			reason = "no_route"
		}
		if d.Route != nil {
			if sn, ok := g.senders[d.Route.SenderID]; ok {
				providerID, senderID, sender = &sn.ProviderID, &sn.ID, sn.Value
			}
		}
	case d.Sender == nil:
		status, reason = "skipped", "sender_"+d.Hold
		if d.Hold == "no_route" {
			reason = "no_route"
		}
	case t.skipOutside && !w.Inside(now, loc):
		status, reason = "skipped", "quiet_hours"
	default:
		next := w.NextOpen(now, loc)
		closes := w.Closes(next, loc)
		sendAfter, expires = &next, &closes
		providerID, senderID, sender = &d.Sender.ProviderID, &d.Sender.ID, d.Sender.Value
	}
	domain := s.shortDomain(g.rules)
	body := t.text.Body
	links := []store.SmsLinkInsertParams{}
	if domain != "" && t.kind != "confirmation" {
		body = withShortLinks(body, domain, func(i int, l Link) string {
			code := LinkCode(id, uuid.Nil, i)
			links = append(links, store.SmsLinkInsertParams{Code: code, WorkspaceID: ws.ID, MessageID: &id, CampaignID: t.campaignID, AutomationID: t.automationID,
				ContactID: t.contactID, Original: l.URL, Target: target(l.URL, utmCampaign(t.name)), CreatedAt: now, ExpiresAt: now.AddDate(1, 0, 0)})
			return code
		})
	}
	full := Text{Prefix: t.text.Prefix, Body: render(body, t.recipient), OptOut: t.text.OptOut}.Full()
	cnt := sms.CountText(full)
	units := max(cnt.Segments, 1)
	var price *float64
	cost := 0.0
	if d.Price > 0 && status != "skipped" {
		pr := d.Price
		price, cost = &pr, d.Price*float64(units)
	}
	if status == "held" {
		cost = 0
	}
	msg := store.ChanMessageInsertParams{ID: id, WorkspaceID: ws.ID, Channel: "sms", Kind: t.kind, CampaignID: t.campaignID, AutomationID: t.automationID,
		AutomationStep: t.step, RunID: t.runID, ContactID: t.contactID, ListID: t.listID, Recipient: n.E164, Country: n.Country, TimeZone: zone, ZoneSource: string(zsrc),
		ProviderID: providerID, SenderID: senderID, Sender: sender, Body: &full, Units: int32(units), Encoding: string(cnt.Encoding), Price: price, Cost: cost,
		Status: status, Reason: reason, SendAfter: sendAfter, ExpiresAt: expires, CreatedAt: now}
	if err := q.ChanMessageInsert(ctx, msg); err != nil {
		return store.ChannelMessage{}, err
	}
	for _, l := range links {
		if err := q.SmsLinkInsert(ctx, l); err != nil {
			return store.ChannelMessage{}, err
		}
	}
	if c := s.jobs(); c != nil && status == "queued" {
		if _, err := c.InsertTx(ctx, tx, DispatchArgs{}, nil); err != nil {
			return store.ChannelMessage{}, err
		}
	}
	return q.ChanMessage(ctx, id)
}

// emailSourceLabel names where an email subscription came from, as the
// profile's consent card does ("Brevo import", "preference center").
func emailSourceLabel(src string) string {
	switch src {
	case "preference_center":
		return "preference center"
	case "one_click":
		return "List-Unsubscribe one-click"
	case "repermission":
		return "re-permission email"
	case "team", "manual":
		return "added by the team"
	case "api":
		return "the API"
	}
	if strings.HasPrefix(src, "import:") || strings.HasPrefix(src, "form:") {
		return segments.SourceLabel(src)
	}
	return src
}
