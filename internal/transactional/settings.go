package transactional

import (
	"context"
	"encoding/json"
	"math"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// TransactionalSenderRef is a template's From.
type TransactionalSenderRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TransactionalSenderOption is a sender the Transactional stream can use:
// a sender on a domain marked for transactional mail.
type TransactionalSenderOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	ReplyTo  string `json:"replyTo"`
	Domain   string `json:"domain"`
	Verified bool   `json:"verified"`
}

// TransactionalSettings is Settings › Transactional (except categories).
type TransactionalSettings struct {
	LinkContacts         bool    `json:"linkContacts" doc:"Link recipients who match a contact to that contact (never creates contacts)"`
	ContentRetentionDays int     `json:"contentRetentionDays" minimum:"0" maximum:"90"`
	AlertBounceRate      float64 `json:"alertBounceRate" doc:"Percent of a template's messages in an hour"`
	AlertFailed          int     `json:"alertFailed" doc:"Failed messages in an hour"`
	AlertMedianSeconds   int     `json:"alertMedianSeconds" doc:"Median delivery time over an hour"`
	// Defaults for new templates.
	DefaultSender      *TransactionalSenderRef `json:"defaultSender,omitempty"`
	DefaultReplyTo     string                  `json:"defaultReplyTo"`
	DefaultTrackOpens  bool                    `json:"defaultTrackOpens"`
	DefaultTrackClicks bool                    `json:"defaultTrackClicks"`
	// Senders are the verified Transactional senders to pick from.
	Senders []TransactionalSenderOption `json:"senders"`
	// ProductName is what recipients see ("Notifications from Acme Cloud"):
	// the default sender's name.
	ProductName string     `json:"productName"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy   string     `json:"updatedBy,omitempty"`
	// CanEdit is transactional:publish; Reason says who can help otherwise.
	CanEdit bool   `json:"canEdit"`
	Reason  string `json:"reason,omitempty"`
}

// TransactionalSettingsInput saves Settings › Transactional.
type TransactionalSettingsInput struct {
	LinkContacts         bool    `json:"linkContacts"`
	ContentRetentionDays int     `json:"contentRetentionDays" minimum:"0" maximum:"90"`
	AlertBounceRate      float64 `json:"alertBounceRate" minimum:"0.1" maximum:"100"`
	AlertFailed          int     `json:"alertFailed" minimum:"1" maximum:"100000"`
	AlertMedianSeconds   int     `json:"alertMedianSeconds" minimum:"1" maximum:"3600"`
	DefaultSenderID      string  `json:"defaultSenderId,omitempty" doc:"A Transactional sender, or empty"`
	DefaultReplyTo       string  `json:"defaultReplyTo,omitempty" maxLength:"254"`
	DefaultTrackOpens    bool    `json:"defaultTrackOpens"`
	DefaultTrackClicks   bool    `json:"defaultTrackClicks"`
}

// settingsRow loads (creating on first use) the workspace's settings.
func (s *Service) settingsRow(ctx context.Context, q *store.Queries, wsID uuid.UUID) (store.TxnSettingsRow, error) {
	if err := q.TxnSettingsEnsure(ctx, wsID); err != nil {
		return store.TxnSettingsRow{}, err
	}
	return q.TxnSettings(ctx, wsID)
}

// senderVerified reports whether a sender can send: its domain verified,
// or at least one provider verified the address.
func senderVerified(domainStatus string, verification json.RawMessage) bool {
	if domainStatus == "verified" {
		return true
	}
	var v map[string]string
	_ = json.Unmarshal(verification, &v)
	for _, st := range v {
		if st == "verified" {
			return true
		}
	}
	return false
}

// transactionalSenders lists the senders on Transactional domains.
func (s *Service) transactionalSenders(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]TransactionalSenderOption, error) {
	rows, err := q.TxnSenders(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := []TransactionalSenderOption{}
	for _, r := range rows {
		if !r.Transactional {
			continue
		}
		out = append(out, TransactionalSenderOption{ID: r.ID.String(), Name: r.Name, Email: r.Email, ReplyTo: r.ReplyTo, Domain: r.Domain,
			Verified: senderVerified(r.DomainStatus, r.Verification)})
	}
	return out, nil
}

// productName is how recipients know the product: the default
// Transactional sender's name, else the first sender's, else the workspace.
func productName(st store.TxnSettingsRow, senders []TransactionalSenderOption, ws *store.Workspace) string {
	if st.DefaultSenderName != "" {
		return st.DefaultSenderName
	}
	for _, sn := range senders {
		if sn.Name != "" {
			return sn.Name
		}
	}
	return ws.Name
}

// Settings returns Settings › Transactional.
func (s *Service) Settings(ctx context.Context, ws *store.Workspace, p *authz.Principal) (TransactionalSettings, error) {
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return TransactionalSettings{}, err
	}
	senders, err := s.transactionalSenders(ctx, s.Q, ws.ID)
	if err != nil {
		return TransactionalSettings{}, err
	}
	out := TransactionalSettings{
		LinkContacts: st.LinkContacts, ContentRetentionDays: int(st.ContentRetentionDays),
		AlertBounceRate: st.AlertBounceRate, AlertFailed: int(st.AlertFailed), AlertMedianSeconds: int(st.AlertMedianSeconds),
		DefaultReplyTo: st.DefaultReplyTo, DefaultTrackOpens: st.DefaultTrackOpens, DefaultTrackClicks: st.DefaultTrackClicks,
		Senders: senders, ProductName: productName(st, senders, ws), UpdatedBy: st.UpdatedByName, CanEdit: p.Can(authz.TransactionalPublish),
	}
	if st.UpdatedBy != nil {
		out.UpdatedAt = &st.UpdatedAt
	}
	if st.DefaultSenderID != nil {
		out.DefaultSender = &TransactionalSenderRef{ID: st.DefaultSenderID.String(), Name: st.DefaultSenderName, Email: st.DefaultSenderEmail}
	}
	if !out.CanEdit {
		out.Reason = deniedMessage(ctx, "settings", s.helpers(ctx, ws.ID, authz.TransactionalPublish, 3))
	}
	return out, nil
}

// SaveSettings changes Settings › Transactional.
func (s *Service) SaveSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, in TransactionalSettingsInput) (TransactionalSettings, error) {
	if err := s.requireSettings(ctx, ws, p); err != nil {
		return TransactionalSettings{}, err
	}
	in.DefaultReplyTo = strings.TrimSpace(in.DefaultReplyTo)
	if in.DefaultReplyTo != "" {
		if _, err := mail.ParseAddress(in.DefaultReplyTo); err != nil {
			return TransactionalSettings{}, api.Invalid("body.defaultReplyTo", say(ctx, "replyTo.invalid", nil))
		}
	}
	if in.ContentRetentionDays < 0 || in.ContentRetentionDays > 90 {
		return TransactionalSettings{}, api.Invalid("body.contentRetentionDays", say(ctx, "settings.retentionRange", nil))
	}
	in.AlertBounceRate = math.Round(in.AlertBounceRate*100) / 100
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.TxnSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		cur, err := q.TxnSettingsForUpdate(ctx, ws.ID)
		if err != nil {
			return err
		}
		var sender *uuid.UUID
		if in.DefaultSenderID != "" {
			id, err := uuid.Parse(in.DefaultSenderID)
			if err != nil {
				return api.Invalid("body.defaultSenderId", say(ctx, "sender.pick", nil))
			}
			snd, err := q.TxnSenderByID(ctx, store.TxnSenderByIDParams{WorkspaceID: ws.ID, ID: id})
			if err != nil {
				return api.Invalid("body.defaultSenderId", say(ctx, "sender.gone", nil))
			}
			if !snd.Transactional {
				return api.Invalid("body.defaultSenderId", say(ctx, "sender.notTransactional", i18n.Args{"email": snd.Email}))
			}
			sender = &id
		}
		before := map[string]any{"link_contacts": cur.LinkContacts, "content_retention_days": cur.ContentRetentionDays, "alert_bounce_rate": cur.AlertBounceRate,
			"alert_failed": cur.AlertFailed, "alert_median_seconds": cur.AlertMedianSeconds, "default_sender_id": idString(cur.DefaultSenderID),
			"default_reply_to": cur.DefaultReplyTo, "default_track_opens": cur.DefaultTrackOpens, "default_track_clicks": cur.DefaultTrackClicks}
		after := map[string]any{"link_contacts": in.LinkContacts, "content_retention_days": in.ContentRetentionDays, "alert_bounce_rate": in.AlertBounceRate,
			"alert_failed": in.AlertFailed, "alert_median_seconds": in.AlertMedianSeconds, "default_sender_id": idString(sender),
			"default_reply_to": in.DefaultReplyTo, "default_track_opens": in.DefaultTrackOpens, "default_track_clicks": in.DefaultTrackClicks}
		changedBefore, changedAfter := diffMaps(before, after)
		if len(changedAfter) == 0 {
			return nil
		}
		if err := q.TxnSettingsUpdate(ctx, store.TxnSettingsUpdateParams{WorkspaceID: ws.ID, LinkContacts: in.LinkContacts, ContentRetentionDays: int32(in.ContentRetentionDays),
			AlertBounceRate: in.AlertBounceRate, AlertFailed: int32(in.AlertFailed), AlertMedianSeconds: int32(in.AlertMedianSeconds), DefaultSenderID: sender,
			DefaultReplyTo: in.DefaultReplyTo, DefaultTrackOpens: in.DefaultTrackOpens, DefaultTrackClicks: in.DefaultTrackClicks, UpdatedBy: userRef(p), UpdatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.settings_updated", TargetType: "transactional_settings", TargetID: ws.ID.String(), TargetName: "Transactional settings",
			Before: changedBefore, After: changedAfter, Metadata: map[string]any{"summary": settingsSummary(changedBefore, changedAfter)}})
	})
	if err != nil {
		return TransactionalSettings{}, err
	}
	return s.Settings(ctx, ws, p)
}

func idString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// diffMaps keeps the keys whose values differ.
func diffMaps(before, after map[string]any) (map[string]any, map[string]any) {
	b, a := map[string]any{}, map[string]any{}
	for k, av := range after {
		bv := before[k]
		bj, _ := json.Marshal(bv)
		aj, _ := json.Marshal(av)
		if string(bj) != string(aj) {
			b[k], a[k] = bv, av
		}
	}
	return b, a
}

var settingLabels = map[string]string{
	"link_contacts": "contact linking", "content_retention_days": "content retention", "alert_bounce_rate": "bounce alert limit",
	"alert_failed": "failed-messages alert limit", "alert_median_seconds": "delivery-time alert limit", "default_sender_id": "default sender",
	"default_reply_to": "default reply-to", "default_track_opens": "open tracking default", "default_track_clicks": "click tracking default",
}

// settingsSummary is the audit entry's line: "Bounce alert limit changed
// from 3% to 2%" for one change, the list of what changed otherwise.
func settingsSummary(before, after map[string]any) string {
	if len(after) == 1 {
		for k, v := range after {
			switch k {
			case "alert_bounce_rate":
				return "Bounce alert limit changed from " + pctText(before[k]) + " to " + pctText(v)
			case "content_retention_days":
				return "Content retention changed from " + daysText(before[k]) + " to " + daysText(v)
			}
		}
	}
	parts := []string{}
	for _, k := range []string{"link_contacts", "content_retention_days", "alert_bounce_rate", "alert_failed", "alert_median_seconds", "default_sender_id", "default_reply_to", "default_track_opens", "default_track_clicks"} {
		if _, ok := after[k]; ok {
			parts = append(parts, settingLabels[k])
		}
	}
	if len(parts) == 0 {
		return ""
	}
	s := strings.Join(parts, ", ")
	return strings.ToUpper(s[:1]) + s[1:] + " changed"
}

func pctText(v any) string {
	f, _ := v.(float64)
	return trimFloat(f) + "%"
}

func daysText(v any) string {
	var n int
	switch x := v.(type) {
	case int:
		n = x
	case int32:
		n = int(x)
	case float64:
		n = int(x)
	}
	if n == 0 {
		return "none"
	}
	return itoa(n) + " days"
}

func trimFloat(f float64) string {
	return strconv.FormatFloat(math.Round(f*100)/100, 'f', -1, 64)
}
