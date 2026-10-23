package whatsapp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
)

// Sending providers › WhatsApp (brief §5.13, board WhatsAppChannel): the
// account card, the connection (write-only credentials, the webhook),
// the stats, the templates, the events and the states (not connected,
// waiting for Meta's verification request, the token expired, read-only).

// WhatsAppAccount is the account card. Credentials are never returned.
type WhatsAppAccount struct {
	ID               string     `json:"id"`
	Status           string     `json:"status" enum:"waiting,connected,token_rejected"`
	StatusDetail     string     `json:"statusDetail,omitempty"`
	StatusSince      *time.Time `json:"statusSince,omitempty"`
	BusinessName     string     `json:"businessName"`
	BusinessHint     string     `json:"businessHint"`
	Number           string     `json:"number"`
	DisplayName      string     `json:"displayName"`
	NameStatus       string     `json:"nameStatus"`
	PhoneStatus      string     `json:"phoneStatus"`
	Quality          string     `json:"quality" enum:"high,medium,low,unknown"`
	PreviousQuality  string     `json:"previousQuality,omitempty"`
	QualityChangedAt *time.Time `json:"qualityChangedAt,omitempty"`
	MessagingLimit   int        `json:"messagingLimit"`
	PreviousLimit    int        `json:"previousLimit,omitempty"`
	LimitChangedAt   *time.Time `json:"limitChangedAt,omitempty"`
	ConnectedAt      time.Time  `json:"connectedAt"`
	ConnectedBy      string     `json:"connectedBy"`
	CheckedAt        *time.Time `json:"checkedAt,omitempty"`
	WebhookURL       string     `json:"webhookUrl"`
	// VerifyToken is shown to who can manage providers, for Meta's app
	// settings.
	VerifyToken       string     `json:"verifyToken,omitempty"`
	VerifiedAt        *time.Time `json:"verifiedAt,omitempty"`
	LastEventAt       *time.Time `json:"lastEventAt,omitempty"`
	TemplatesSyncedAt *time.Time `json:"templatesSyncedAt,omitempty"`
	Demo              bool       `json:"demo"`
	// Held are the sends waiting for a new token: names and how many.
	Held      []string `json:"held"`
	HeldCount int64    `json:"heldCount"`
}

// WhatsAppTemplateCounts are the templates by status.
type WhatsAppTemplateCounts struct {
	Approved int `json:"approved"`
	Pending  int `json:"pending"`
	Rejected int `json:"rejected"`
	Other    int `json:"other"`
}

// WhatsAppOverview is Sending providers › WhatsApp.
type WhatsAppOverview struct {
	Account        *WhatsAppAccount       `json:"account,omitempty"`
	Templates      []WhatsAppTemplate     `json:"templates"`
	TemplateCounts WhatsAppTemplateCounts `json:"templateCounts"`
	Consent        int64                  `json:"consent"`
	ConsentLists   []WhatsAppListConsent  `json:"consentLists"`
	RatesAsOf      time.Time              `json:"ratesAsOf"`
	Quiet          bool                   `json:"quiet"`
	Window         string                 `json:"window"`
	// SameWindowAsSMS is true once the workspace has SMS rules: the
	// window is theirs.
	SameWindowAsSMS bool `json:"sameWindowAsSms"`
	// WebhookURL is where Meta posts (shown before connecting too).
	WebhookURL string `json:"webhookUrl"`
	// TestNumbers are the member's numbers for tests.
	TestNumbers []string `json:"testNumbers"`
	CanManage   bool     `json:"canManage"`
	CanSend     bool     `json:"canSend"`
	Helpers     []string `json:"helpers"`
}

// Overview is Sending providers › WhatsApp.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal) (WhatsAppOverview, error) {
	out := WhatsAppOverview{Templates: []WhatsAppTemplate{}, ConsentLists: []WhatsAppListConsent{}, TestNumbers: []string{}, Helpers: []string{},
		CanManage: p.Can(authz.ProvidersManage), CanSend: canSend(p), WebhookURL: s.webhookURL(ws)}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws, authz.ProvidersManage)
	}
	rl := s.rules(ctx, s.Q, ws)
	out.Quiet, out.Window = rl.Quiet, rl.Window.String()
	if _, err := s.Q.PushChannelWindow(ctx, ws.ID); err == nil {
		out.SameWindowAsSMS = true
	}
	out.RatesAsOf = s.rateCard(ctx, s.Q, ws).asOf
	lists, total, err := s.listConsent(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Consent = total
	for _, l := range lists {
		if l.Consent > 0 {
			out.ConsentLists = append(out.ConsentLists, l)
		}
	}
	if p.IsHuman() {
		rows, err := s.Pool.Query(ctx, `SELECT phone FROM sms_test_numbers WHERE workspace_id = $1 AND user_id = $2 ORDER BY added_at DESC LIMIT 5`, ws.ID, p.UserID)
		if err == nil {
			for rows.Next() {
				var ph string
				if rows.Scan(&ph) == nil {
					out.TestNumbers = append(out.TestNumbers, phone.Display(ph))
				}
			}
			rows.Close()
		}
	}
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	acct := &WhatsAppAccount{ID: a.ID.String(), Status: a.Status, StatusDetail: a.StatusDetail, StatusSince: a.StatusSince, BusinessName: a.BusinessName,
		BusinessHint: a.BusinessHint, Number: phone.Display(a.PhoneNumber), DisplayName: a.DisplayName, NameStatus: a.NameStatus, PhoneStatus: a.PhoneStatus,
		Quality: a.Quality, PreviousQuality: a.PreviousQuality, QualityChangedAt: a.QualityChangedAt, MessagingLimit: int(a.MessagingLimit),
		PreviousLimit: int(a.PreviousLimit), LimitChangedAt: a.LimitChangedAt, ConnectedAt: a.ConnectedAt, ConnectedBy: a.ConnectedByName, CheckedAt: a.CheckedAt,
		WebhookURL: s.webhookURL(ws), VerifiedAt: a.VerifiedAt, LastEventAt: a.LastEventAt, TemplatesSyncedAt: a.TemplatesSyncedAt, Demo: a.Demo, Held: []string{}}
	if out.CanManage {
		acct.VerifyToken = a.VerifyToken
	}
	if a.Status == "token_rejected" {
		acct.Held = s.heldNames(ctx, s.Pool, ws)
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE workspace_id = $1 AND channel = 'whatsapp' AND status = 'held'
			AND reason = 'token_rejected'`, ws.ID).Scan(&acct.HeldCount)
	}
	out.Account = acct
	ts, err := s.Q.WhatsAppTemplates(ctx, a.ID)
	if err != nil {
		return out, err
	}
	l := langOf(ctx)
	for _, t := range ts {
		v := templateView(l, t)
		out.Templates = append(out.Templates, v)
		switch t.Status {
		case "approved":
			out.TemplateCounts.Approved++
		case "pending", "in_appeal":
			out.TemplateCounts.Pending++
		case "rejected":
			out.TemplateCounts.Rejected++
		default:
			out.TemplateCounts.Other++
		}
	}
	return out, nil
}

// WhatsAppEvent is a webhook as Meta sent it, and what came of it.
type WhatsAppEvent struct {
	ID           string     `json:"id"`
	ReceivedAt   time.Time  `json:"receivedAt"`
	Summary      string     `json:"summary"`
	SignatureOK  bool       `json:"signatureOk"`
	NormalizedAt *time.Time `json:"normalizedAt,omitempty"`
	Events       int        `json:"events"`
	Error        string     `json:"error,omitempty"`
	// Kind is what it was about; Number the person's number, masked
	// (+1 312 ••• 0123); Status a message status or a template's.
	Kind   string `json:"kind" enum:"status,message,template,quality,account,unknown"`
	Number string `json:"number,omitempty"`
	Status string `json:"status,omitempty"`
	Code   int    `json:"code,omitempty"`
}

// WhatsAppEventList is a page of events, newest first.
type WhatsAppEventList struct {
	Items []WhatsAppEvent `json:"items"`
	// Next is the time to ask for the page after this one.
	Next *time.Time `json:"next,omitempty"`
}

// Events lists Meta's webhooks, newest first.
func (s *Service) Events(ctx context.Context, ws *store.Workspace, before *time.Time, limit int) (WhatsAppEventList, error) {
	out := WhatsAppEventList{Items: []WhatsAppEvent{}}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	at := s.now().Add(time.Minute)
	if before != nil {
		at = *before
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, received_at, summary, signature_ok, normalized_at, events, normalize_error, body FROM whatsapp_raw_events
		WHERE workspace_id = $1 AND received_at < $2 ORDER BY received_at DESC LIMIT $3`, ws.ID, at, limit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var e WhatsAppEvent
		var id uuid.UUID
		var events int32
		var body []byte
		if err := rows.Scan(&id, &e.ReceivedAt, &e.Summary, &e.SignatureOK, &e.NormalizedAt, &events, &e.Error, &body); err != nil {
			return out, err
		}
		if len(out.Items) == limit {
			out.Next = &out.Items[len(out.Items)-1].ReceivedAt
			break
		}
		e.ID, e.Events, e.Kind = id.String(), int(events), "unknown"
		if e.SignatureOK {
			if evs, err := meta.ParseWebhook(body); err == nil && len(evs) > 0 {
				ev := evs[0]
				e.Kind = ev.Kind
				switch ev.Kind {
				case meta.EventStatus:
					e.Number, e.Status = phone.Mask(ev.Recipient), ev.Status
					if len(ev.Errors) > 0 {
						e.Code = ev.Errors[0].Code
					}
				case meta.EventMessage:
					e.Number, e.Status = phone.Mask(ev.From), ev.Type
				case meta.EventTemplate:
					e.Status = statusOf(ev.TemplateEvent)
				case meta.EventQuality:
					e.Status = ev.QualityEvent
				}
			}
		}
		out.Items = append(out.Items, e)
	}
	return out, rows.Err()
}
