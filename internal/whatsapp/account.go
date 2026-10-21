package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
)

// The account (brief §3.9, board WhatsAppChannel › Account): connecting
// with four write-only values, Meta's verification request, the card as
// Meta reports it, replacing the credentials (Reconnect), the hourly check
// and disconnecting. A token Meta rejects holds every WhatsApp send until
// it's replaced.

// WhatsAppConnectInput connects an account. Every value is write-only.
type WhatsAppConnectInput struct {
	BusinessAccountID string `json:"businessAccountId" minLength:"1" maxLength:"40" doc:"The WhatsApp Business Account ID"`
	PhoneNumberID     string `json:"phoneNumberId" minLength:"1" maxLength:"40"`
	AccessToken       string `json:"accessToken" minLength:"1" maxLength:"1024" doc:"A system user's permanent access token"`
	AppSecret         string `json:"appSecret" minLength:"1" maxLength:"200" doc:"The app secret Meta signs webhooks with"`
}

// WhatsAppCredentialsInput replaces credentials: a new access token
// (Reconnect), and the other values when they changed. Empty keeps one.
type WhatsAppCredentialsInput struct {
	BusinessAccountID string `json:"businessAccountId,omitempty" maxLength:"40"`
	PhoneNumberID     string `json:"phoneNumberId,omitempty" maxLength:"40"`
	AccessToken       string `json:"accessToken,omitempty" maxLength:"1024"`
	AppSecret         string `json:"appSecret,omitempty" maxLength:"200"`
}

var idRe = regexp.MustCompile(`^[0-9]{5,30}$`)

// metaProblem words a Graph API failure for the person connecting.
func metaProblem(err error, field string) error {
	var me *meta.Error
	if errors.As(err, &me) {
		switch {
		case me.TokenRejected():
			return api.Invalid("body.accessToken", "Meta rejected the access token: "+clip(me.Message, 160))
		case me.Code == 100 || me.Status == 404:
			return api.Invalid(field, "Meta doesn't know this ID, or the token can't reach it: "+clip(me.Message, 160))
		case me.Unavailable():
			return api.Err(502, "meta_unavailable", "Meta's Graph API didn't answer. Try again in a moment.")
		}
		return api.Err(502, "meta_error", "Meta answered: "+clip(me.Message, 200))
	}
	return api.Err(502, "meta_unavailable", "Meta's Graph API didn't answer. Try again in a moment.")
}

// info is what Meta reports for an account.
type info struct {
	phone   meta.PhoneNumber
	account meta.BusinessAccount
}

// fetch reads the number and the business account from Meta.
func (s *Service) fetch(ctx context.Context, c *meta.Client) (info, error) {
	var out info
	var err error
	if out.phone, err = c.Phone(ctx); err != nil {
		return out, err
	}
	if out.account, err = c.Account(ctx); err != nil {
		return out, err
	}
	return out, nil
}

func nameStatus(s string) string {
	switch strings.ToUpper(s) {
	case "APPROVED", "AVAILABLE_WITHOUT_REVIEW":
		return "approved"
	case "PENDING_REVIEW", "PENDING":
		return "pending"
	case "DECLINED":
		return "declined"
	case "":
		return ""
	}
	return strings.ToLower(s)
}

// setInfo stores what Meta reported, in q's transaction.
func setInfo(ctx context.Context, q *store.Queries, id uuid.UUID, in info, at time.Time) error {
	number := meta.ParsePhoneDisplay(in.phone.DisplayPhoneNumber)
	return q.WhatsAppAccountSetInfo(ctx, store.WhatsAppAccountSetInfoParams{BusinessName: in.account.Name, PhoneNumber: number, DisplayName: in.phone.VerifiedName,
		NameStatus: nameStatus(in.phone.NameStatus), PhoneStatus: strings.ToLower(in.phone.Status), Quality: meta.Quality(in.phone.QualityRating), At: &at,
		MessagingLimit: int32(meta.LimitOf(in.phone.MessagingLimitTier)), ID: id})
}

// Connect seals the credentials, checks them with Meta and shows the
// webhook to paste in Meta's app settings. Nothing is sent until Meta
// verifies the webhook.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WhatsAppConnectInput) error {
	if _, err := s.account(ctx, s.Q, ws); err == nil {
		return api.Conflict("WhatsApp is already connected. Disconnect it first to use another account.")
	} else if !errors.Is(err, errNoAccount) {
		return err
	}
	creds := meta.Credentials{BusinessAccountID: strings.TrimSpace(in.BusinessAccountID), PhoneNumberID: strings.TrimSpace(in.PhoneNumberID),
		AccessToken: strings.TrimSpace(in.AccessToken), AppSecret: strings.TrimSpace(in.AppSecret)}
	if !idRe.MatchString(creds.BusinessAccountID) {
		return api.Invalid("body.businessAccountId", "The WhatsApp Business Account ID is a number, from WhatsApp Manager › Account tools.")
	}
	if !idRe.MatchString(creds.PhoneNumberID) {
		return api.Invalid("body.phoneNumberId", "The phone number ID is a number, from your app's WhatsApp › API setup page.")
	}
	c := s.clientFor(creds, false)
	got, err := s.fetch(ctx, c)
	if err != nil {
		return metaProblem(err, "body.phoneNumberId")
	}
	// Meta posts webhooks for the account only to subscribed apps.
	if err := c.SubscribeApp(ctx); err != nil {
		s.log().WarnContext(ctx, "whatsapp subscribe app", "err", err)
	}
	id := uuid.Must(uuid.NewV7())
	raw, _ := json.Marshal(creds) //nolint:gosec // G117: sealed with internal/secrets at once, never stored or returned as is
	sealed, err := s.Box.Seal(raw, credAD(id))
	if err != nil {
		return err
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.WhatsAppAccountInsert(ctx, store.WhatsAppAccountInsertParams{ID: id, WorkspaceID: ws.ID, CredentialsEnc: sealed,
			BusinessHint: hint(creds.BusinessAccountID), VerifyToken: randomToken(), ConnectedAt: &now, ConnectedBy: uuidPtr(p.UserID)}); err != nil {
			if strings.Contains(err.Error(), "whatsapp_accounts_live_idx") {
				return api.Conflict("WhatsApp is already connected.")
			}
			return err
		}
		if err := setInfo(ctx, q, id, got, now); err != nil {
			return err
		}
		if err := s.ensureSettings(ctx, q, ws, now); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp.connected", TargetType: "whatsapp_account", TargetID: id.String(),
			TargetName: got.account.Name, After: map[string]any{"business_account": hint(creds.BusinessAccountID), "number": meta.ParsePhoneDisplay(got.phone.DisplayPhoneNumber),
				"display_name": got.phone.VerifiedName}})
	})
	if err != nil {
		return err
	}
	// The templates come at once; the hourly sync keeps them.
	if _, err := s.Sync(ctx, ws, p, false); err != nil {
		s.log().WarnContext(ctx, "whatsapp first sync", "err", err)
	}
	return nil
}

// ReplaceCredentials seals new credentials after Meta accepts them. A
// rejected token's held messages go again (Reconnect).
func (s *Service) ReplaceCredentials(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WhatsAppCredentialsInput) error {
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return api.Conflict("Connect WhatsApp first.")
	}
	if err != nil {
		return err
	}
	creds, err := s.credentials(a.ID, a.CredentialsEnc)
	if err != nil {
		return err
	}
	changed := []string{}
	if v := strings.TrimSpace(in.BusinessAccountID); v != "" {
		if !idRe.MatchString(v) {
			return api.Invalid("body.businessAccountId", "The WhatsApp Business Account ID is a number, from WhatsApp Manager › Account tools.")
		}
		creds.BusinessAccountID, changed = v, append(changed, "business_account_id")
	}
	if v := strings.TrimSpace(in.PhoneNumberID); v != "" {
		if !idRe.MatchString(v) {
			return api.Invalid("body.phoneNumberId", "The phone number ID is a number, from your app's WhatsApp › API setup page.")
		}
		creds.PhoneNumberID, changed = v, append(changed, "phone_number_id")
	}
	if v := strings.TrimSpace(in.AccessToken); v != "" {
		creds.AccessToken, changed = v, append(changed, "access_token")
	}
	if v := strings.TrimSpace(in.AppSecret); v != "" {
		creds.AppSecret, changed = v, append(changed, "app_secret")
	}
	if len(changed) == 0 {
		return api.Invalid("body", "Enter the value to replace: the access token, the app secret or an ID.")
	}
	c := s.clientFor(creds, a.Demo)
	got, err := s.fetch(ctx, c)
	if err != nil {
		return metaProblem(err, "body.phoneNumberId")
	}
	raw, _ := json.Marshal(creds) //nolint:gosec // G117: sealed with internal/secrets at once, never stored or returned as is
	sealed, err := s.Box.Seal(raw, credAD(a.ID))
	if err != nil {
		return err
	}
	now := s.now()
	var released int64
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.WhatsAppAccountSetCredentials(ctx, store.WhatsAppAccountSetCredentialsParams{CredentialsEnc: sealed, BusinessHint: hint(creds.BusinessAccountID),
			ID: a.ID}); err != nil {
			return err
		}
		if err := setInfo(ctx, q, a.ID, got, now); err != nil {
			return err
		}
		status := "connected"
		if a.VerifiedAt == nil {
			status = "waiting"
		}
		if err := q.WhatsAppAccountSetStatus(ctx, store.WhatsAppAccountSetStatusParams{Status: status, At: now, ID: a.ID}); err != nil {
			return err
		}
		// What the rejected token held goes again.
		tag, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'queued', reason = '', updated_at = $2
			WHERE workspace_id = $1 AND channel = 'whatsapp' AND status = 'held' AND reason = 'token_rejected'`, ws.ID, now)
		if err != nil {
			return err
		}
		released = tag.RowsAffected()
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp.credentials_replaced", TargetType: "whatsapp_account",
			TargetID: a.ID.String(), TargetName: a.BusinessName, Metadata: map[string]any{"replaced": changed, "released": released}})
	})
	if err != nil {
		return err
	}
	if released > 0 {
		s.kick(ctx)
	}
	return nil
}

// Disconnect removes the account: messages waiting to go are cancelled.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal) error {
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return api.NotFound("WhatsApp account")
	}
	if err != nil {
		return err
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.WhatsAppAccountRemove(ctx, store.WhatsAppAccountRemoveParams{At: &now, RemovedBy: uuidPtr(p.UserID), ID: a.ID}); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'cancelled', reason = 'disconnected', updated_at = $2
			WHERE workspace_id = $1 AND channel = 'whatsapp' AND status IN ('queued','held')`, ws.ID, now)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp.disconnected", TargetType: "whatsapp_account", TargetID: a.ID.String(),
			TargetName: a.BusinessName, Before: map[string]any{"number": a.PhoneNumber, "business_account": a.BusinessHint},
			Metadata: map[string]any{"cancelled": tag.RowsAffected()}})
	})
}

// Verify answers Meta's verification request for a workspace's webhook:
// the challenge when the verify token matches.
func (s *Service) Verify(ctx context.Context, slug, mode, token, challenge string) (string, bool) {
	a, err := s.Q.WhatsAppAccountBySlug(ctx, slug)
	if err != nil || mode != "subscribe" || token == "" {
		return "", false
	}
	if !constantEqual(token, a.VerifyToken) {
		return "", false
	}
	if err := s.Q.WhatsAppAccountVerified(ctx, store.WhatsAppAccountVerifiedParams{At: s.now(), ID: a.ID}); err != nil {
		s.log().WarnContext(ctx, "whatsapp verified", "err", err)
	}
	return challenge, true
}

// Check reads the account from Meta now (the card's numbers, "Check
// again") and syncs the templates. A rejected token holds the sends.
func (s *Service) Check(ctx context.Context, ws *store.Workspace, p *authz.Principal) error {
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return api.Conflict("Connect WhatsApp first.")
	}
	if err != nil {
		return err
	}
	if err := s.check(ctx, ws, accountOf(a)); err != nil {
		return err
	}
	_, err = s.Sync(ctx, ws, p, false)
	var me *meta.Error
	if errors.As(err, &me) {
		return nil // the account's status says what happened
	}
	return err
}

// check reads one account from Meta, stores it and reacts: a rejected
// token holds the sends and tells the providers' managers; a lower
// quality rating tells them too.
func (s *Service) check(ctx context.Context, ws *store.Workspace, a store.WhatsappAccount) error {
	c, err := s.client(a.ID, a.CredentialsEnc, a.Demo)
	if err != nil {
		return err
	}
	got, err := s.fetch(ctx, c)
	now := s.now()
	if err != nil {
		var me *meta.Error
		if errors.As(err, &me) && me.TokenRejected() {
			return s.tokenRejected(ctx, ws, a, me)
		}
		return nil // Meta didn't answer: the next check tries again
	}
	before := a.Quality
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := setInfo(ctx, q, a.ID, got, now); err != nil {
			return err
		}
		if a.Status == "token_rejected" {
			// Meta accepts the token again (it was renewed in Meta).
			status := "connected"
			if a.VerifiedAt == nil {
				status = "waiting"
			}
			return q.WhatsAppAccountSetStatus(ctx, store.WhatsAppAccountSetStatusParams{Status: status, At: now, ID: a.ID})
		}
		return nil
	})
	if err != nil {
		return err
	}
	after := meta.Quality(got.phone.QualityRating)
	if qualityRank(after) < qualityRank(before) && before != "unknown" {
		s.notifyQuality(ctx, ws, before, after, meta.LimitOf(got.phone.MessagingLimitTier), int(a.MessagingLimit))
	}
	return nil
}

func qualityRank(q string) int {
	switch q {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	}
	return 0
}

// qualityLabel is "High", "Medium", "Low".
func qualityLabel(q string) string {
	switch q {
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	}
	return "Unknown"
}

// tokenRejected marks the account, holds what waits to go and tells the
// providers' managers, once.
func (s *Service) tokenRejected(ctx context.Context, ws *store.Workspace, a store.WhatsappAccount, me *meta.Error) error {
	now := s.now()
	detail := "rejected"
	if me.Expired() {
		detail = "expired"
	}
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'held', reason = 'token_rejected', updated_at = $2
			WHERE workspace_id = $1 AND channel = 'whatsapp' AND status = 'queued'`, ws.ID, now); err != nil {
			return err
		}
		if a.Status == "token_rejected" {
			return nil
		}
		if err := q.WhatsAppAccountSetStatus(ctx, store.WhatsAppAccountSetStatusParams{Status: "token_rejected", StatusDetail: detail, At: now, ID: a.ID}); err != nil {
			return err
		}
		held := s.heldNames(ctx, tx, ws)
		summary := "WhatsApp sends are held until the token is replaced"
		if len(held) > 0 {
			summary = "Held: " + strings.Join(held, ", ")
		}
		return notify.ToMembersWith(ctx, tx, ws.ID, authz.ProvidersManage, notify.Notification{Kind: notify.KindWhatsAppTokenFails, Severity: "bad",
			Title:   fmt.Sprintf("Meta rejected the WhatsApp access token: it %s", map[bool]string{true: "expired", false: "was refused"}[me.Expired()]),
			Summary: summary,
			Body:    "WhatsApp sends are held until the token is replaced. Nothing has been lost yet. Enter a new permanent access token in Sending providers › WhatsApp to reconnect.",
			Link:    "/w/" + ws.Slug + "/providers/whatsapp", Data: map[string]any{"code": me.Code, "subcode": me.Subcode}})
	})
}

// heldNames are the campaigns and automations whose messages wait.
func (s *Service) heldNames(ctx context.Context, q store.DBTX, ws *store.Workspace) []string {
	rows, err := q.Query(ctx, `SELECT DISTINCT coalesce(c.name, a.name, '') FROM channel_messages m
		LEFT JOIN campaigns c ON c.id = m.campaign_id LEFT JOIN automations a ON a.id = m.automation_id
		WHERE m.workspace_id = $1 AND m.channel = 'whatsapp' AND m.status IN ('held','queued') AND m.kind IN ('campaign','automation')
		ORDER BY 1 LIMIT 5`, ws.ID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil && n != "" {
			out = append(out, n)
		}
	}
	return out
}

// notifyQuality tells the providers' managers the quality rating dropped.
func (s *Service) notifyQuality(ctx context.Context, ws *store.Workspace, before, after string, limit, prevLimit int) {
	summary := "Quality rating " + qualityLabel(after) + ", was " + qualityLabel(before)
	if limit > 0 && limit < prevLimit {
		summary += fmt.Sprintf(" · messaging limit %s a day, was %s", count(int64(limit)), count(int64(prevLimit)))
	}
	if err := notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.ProvidersView, notify.Notification{Kind: notify.KindWhatsAppAttention, Severity: "warn",
		Title: "WhatsApp quality dropped to " + qualityLabel(after), Summary: summary,
		Body: "More people blocked or reported recent messages. At Low, Meta can limit the number of people you can message further.",
		Link: "/w/" + ws.Slug + "/providers/whatsapp", Data: map[string]any{"quality": after, "previous": before, "limit": limit}}); err != nil {
		s.log().WarnContext(ctx, "whatsapp quality notification", "err", err)
	}
}

// WhatsAppTestInput sends a test to a member's WhatsApp.
type WhatsAppTestInput struct {
	// Phone is the member's number, in any common format.
	Phone string `json:"phone" minLength:"1" maxLength:"40"`
	// TemplateID is the approved template to send; empty picks the first.
	TemplateID string `json:"templateId,omitempty" maxLength:"40"`
}

// WhatsAppTestResult is a test message and what Meta reported so far.
type WhatsAppTestResult struct {
	ID       string    `json:"id"`
	At       time.Time `json:"at"`
	To       string    `json:"to"`
	Template string    `json:"template"`
	Status   string    `json:"status" enum:"queued,sending,sent,delivered,failed,held,cancelled,skipped,undelivered"`
	Read     bool      `json:"read"`
	Reason   string    `json:"reason,omitempty"`
}

// SendAccountTest sends an approved template to a member's WhatsApp
// (logged); the test waits for Meta's delivery report.
func (s *Service) SendAccountTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WhatsAppTestInput) (WhatsAppTestResult, error) {
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return WhatsAppTestResult{}, api.Conflict("Connect WhatsApp first.")
	}
	if err != nil {
		return WhatsAppTestResult{}, err
	}
	n, err := phone.Parse(in.Phone, "")
	if err != nil {
		return WhatsAppTestResult{}, api.Invalid("body.phone", "Enter the number with its country code, like +1 312 555 0123.")
	}
	tpl, err := s.testTemplate(ctx, a.ID, ws, in.TemplateID)
	if err != nil {
		return WhatsAppTestResult{}, err
	}
	vars := exampleVars(tpl)
	m, err := s.queueTest(ctx, ws, p, a, nil, tpl, vars, n, "whatsapp.test_sent", "whatsapp_account", a.ID.String(), a.BusinessName)
	if err != nil {
		return WhatsAppTestResult{}, err
	}
	return m, nil
}

// testTemplate is a test's template: the one asked for, or the first
// approved one.
func (s *Service) testTemplate(ctx context.Context, accountID uuid.UUID, ws *store.Workspace, id string) (store.WhatsappTemplate, error) {
	if id != "" {
		tid, err := uuid.Parse(id)
		if err != nil {
			return store.WhatsappTemplate{}, api.Invalid("body.templateId", "Pick one of the account's approved templates.")
		}
		t, err := s.Q.WhatsAppTemplate(ctx, store.WhatsAppTemplateParams{ID: tid, WorkspaceID: ws.ID})
		if err != nil || t.Status != "approved" || t.RemovedAt != nil {
			return store.WhatsappTemplate{}, api.Invalid("body.templateId", "Pick one of the account's approved templates.")
		}
		return t, nil
	}
	ts, err := s.Q.WhatsAppTemplates(ctx, accountID)
	if err != nil {
		return store.WhatsappTemplate{}, err
	}
	for _, t := range ts {
		if t.Status == "approved" {
			return t, nil
		}
	}
	return store.WhatsappTemplate{}, api.Conflict("The account has no approved template yet: a test is a template message. Create one in WhatsApp Manager.")
}

// queueTest writes a test message, remembers the member's number and
// audits it; the dispatcher sends it at once.
func (s *Service) queueTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, a store.WhatsAppAccountRow, campaignID *uuid.UUID,
	tpl store.WhatsappTemplate, params []string, n phone.Number, action, targetType, targetID, targetName string) (WhatsAppTestResult, error) {
	now := s.now()
	id := uuid.Must(uuid.NewV7())
	text := render(tpl.Body, params)
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := insertMessage(ctx, q, ws, a.ID, outgoing{id: id, kind: "test", campaignID: campaignID, recipient: n.E164, country: n.Country,
			body: text, status: "queued", after: &now, template: tpl.Name, language: tpl.Language, category: tpl.Category, params: params}, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO sms_test_numbers (id, workspace_id, user_id, phone, added_by, added_at) VALUES ($1, $2, $3, $4, $3, $5)
			ON CONFLICT (workspace_id, phone) DO NOTHING`, uuid.Must(uuid.NewV7()), ws.ID, uuidPtr(p.UserID), n.E164, now); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: targetType, TargetID: targetID, TargetName: targetName,
			After: map[string]any{"channel": "whatsapp", "to": phone.Mask(n.E164), "template": tpl.Name}})
	})
	if err != nil {
		return WhatsAppTestResult{}, err
	}
	s.kick(ctx)
	return WhatsAppTestResult{ID: id.String(), At: now, To: n.Display, Template: tpl.Name, Status: "queued"}, nil
}

// TestStatus is a test message's status now.
func (s *Service) TestStatus(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WhatsAppTestResult, error) {
	var out WhatsAppTestResult
	var to string
	var read *time.Time
	err := s.Pool.QueryRow(ctx, `SELECT m.id::text, m.created_at, m.recipient, w.template, m.status, w.read_at, m.reason FROM channel_messages m
		JOIN whatsapp_messages w ON w.message_id = m.id WHERE m.id = $1 AND m.workspace_id = $2 AND m.kind = 'test'`, id, ws.ID).
		Scan(&out.ID, &out.At, &to, &out.Template, &out.Status, &read, &out.Reason)
	if err != nil {
		return out, api.NotFound("Test message")
	}
	out.To, out.Read = phone.Display(to), read != nil
	return out, nil
}

func constantEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
