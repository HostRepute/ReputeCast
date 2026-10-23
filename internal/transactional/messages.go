package transactional

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
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

// The message log (Messages tab) and the message detail.

// Retention of the message log: metadata and events.
const (
	MetadataRetentionDays = 90
	// ExportMax is the most rows one CSV export carries.
	ExportMax = 50000
)

// Log statuses.
var logStatuses = []string{"queued", "sent", "delivered", "deferred", "bounced", "complained", "rejected", "failed", "test"}

// TransactionalRef names something.
type TransactionalRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TransactionalLogTemplate is a message's template and version.
type TransactionalLogTemplate struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Version *int32 `json:"version,omitempty"`
}

// TransactionalBounce is a bounced message's last bounce.
type TransactionalBounce struct {
	Kind       string `json:"kind" enum:"hard,soft"`
	Class      string `json:"class,omitempty" doc:"mailbox_unknown, mailbox_full, domain_invalid, policy_block, spam_block, reputation, temporary, other"`
	Diagnostic string `json:"diagnostic,omitempty" example:"452 4.2.2 mailbox full"`
}

// TransactionalSuppressionInfo is the suppression that stopped a message.
type TransactionalSuppressionInfo struct {
	ID     string    `json:"id,omitempty" doc:"The suppression as it is now; empty once it was removed"`
	Reason string    `json:"reason" example:"hard_bounce"`
	Since  time.Time `json:"since"`
	Code   string    `json:"code,omitempty" example:"550 5.1.1"`
	Detail string    `json:"detail,omitempty"`
	Source string    `json:"source" example:"Postmark"`
	Scope  string    `json:"scope,omitempty" enum:"all,marketing"`
}

// TransactionalOptOutInfo is the opt-out that stopped a message.
type TransactionalOptOutInfo struct {
	Category string    `json:"category"`
	Since    time.Time `json:"since"`
	Source   string    `json:"source" enum:"preference_center,unsubscribe_link,one_click,api,member,"`
}

// TransactionalRejection says why a message stopped before sending.
type TransactionalRejection struct {
	Reason      string                        `json:"reason" enum:"suppressed,opted_out,test_not_allowed,provider"`
	Suppression *TransactionalSuppressionInfo `json:"suppression,omitempty"`
	OptOut      *TransactionalOptOutInfo      `json:"optOut,omitempty"`
	// Allowed are a test-mode key's allowed addresses (besides members).
	Allowed []string `json:"allowed,omitempty"`
	// Error is the provider's answer when it refused the message.
	Error string `json:"error,omitempty"`
}

// TransactionalFailover says a message went through the route's fallback.
type TransactionalFailover struct {
	From string     `json:"from"`
	To   string     `json:"to"`
	At   *time.Time `json:"at,omitempty"`
}

// TransactionalLogItem is one message in the log.
type TransactionalLogItem struct {
	ID        string    `json:"id"`
	At        time.Time `json:"at" doc:"When the call was accepted"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	ContactID string    `json:"contactId,omitempty"`
	// Template is nil for mail that arrived by SMTP (Raw).
	Template *TransactionalLogTemplate `json:"template,omitempty"`
	Raw      bool                      `json:"raw"`
	Subject  string                    `json:"subject"`
	Provider *TransactionalRef         `json:"provider,omitempty"`
	Status   string                    `json:"status" enum:"queued,sent,delivered,deferred,bounced,complained,rejected,failed,test"`
	// DeliveryMs is how long after the call it was delivered.
	DeliveryMs    *int64                  `json:"deliveryMs,omitempty"`
	Deferrals     int                     `json:"deferrals"`
	Bounce        *TransactionalBounce    `json:"bounce,omitempty"`
	Rejection     *TransactionalRejection `json:"rejection,omitempty"`
	TestDelivered bool                    `json:"testDelivered"`
	FailedOver    *TransactionalFailover  `json:"failedOver,omitempty"`
	// RoleAddress: its marketing-only role suppression doesn't apply.
	RoleAddress bool       `json:"roleAddress"`
	Tracked     bool       `json:"tracked"`
	OpenedAt    *time.Time `json:"openedAt,omitempty"`
	ClickedAt   *time.Time `json:"clickedAt,omitempty"`
	Error       string     `json:"error,omitempty"`
	Key         string     `json:"key,omitempty"`
	Category    string     `json:"category,omitempty"`
	// Language is the language version the message went out in (a base
	// code such as es); empty for a template in one language.
	Language string `json:"language,omitempty"`
}

// TransactionalMessageLanguage is why a message went out in its language:
// "Spanish, from the contact", "Requested pt-BR, sent English".
type TransactionalMessageLanguage struct {
	// Sent is the version's language; Contact the contact's language, or
	// the one the call asked for.
	Sent    string `json:"sent"`
	Contact string `json:"contact,omitempty"`
	Reason  string `json:"reason,omitempty" enum:"match,no_language,not_offered,missing,draft,held,requested,"`
}

// TransactionalLogCounts are the log's numbers per status.
type TransactionalLogCounts struct {
	All        int64 `json:"all"`
	Queued     int64 `json:"queued"`
	Sent       int64 `json:"sent"`
	Delivered  int64 `json:"delivered"`
	Deferred   int64 `json:"deferred"`
	Bounced    int64 `json:"bounced"`
	Complained int64 `json:"complained"`
	Rejected   int64 `json:"rejected"`
	Failed     int64 `json:"failed"`
	Test       int64 `json:"test"`
}

// TransactionalLogFilters are the log's filter choices.
type TransactionalLogFilters struct {
	Templates []TransactionalRef `json:"templates" doc:"id is the template's ID; raw is mail that arrived by SMTP"`
	Providers []TransactionalRef `json:"providers"`
	Keys      []TransactionalRef `json:"keys"`
}

// TransactionalLog is a page of the message log.
type TransactionalLog struct {
	Items  []TransactionalLogItem `json:"items"`
	Counts TransactionalLogCounts `json:"counts"`
	// Next pages to older messages; Newest is the cursor of the first item,
	// for live updates (after=Newest returns only newer messages).
	Next    string                  `json:"next,omitempty"`
	Newest  string                  `json:"newest,omitempty"`
	Since   time.Time               `json:"since"`
	Until   time.Time               `json:"until"`
	Filters TransactionalLogFilters `json:"filters"`
	// Retention: metadata and events are kept MetadataDays, rendered
	// content ContentDays.
	MetadataDays int `json:"metadataDays"`
	ContentDays  int `json:"contentDays"`
	// CanExport: exporting needs transactional:content.
	CanExport bool `json:"canExport"`
}

// LogFilter narrows the log.
type LogFilter struct {
	Status   string
	Template string
	Provider string
	Key      string
	Range    string // 1h, 24h, 7d, 30d, custom
	From, To string // custom dates (YYYY-MM-DD, the workspace's time zone)
	Query    string
	Before   string
	After    string
	Limit    int
}

// cursor is "<accepted_at unix nanos>.<message id>".
func encodeCursor(at time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(at.UnixNano(), 10) + "." + id.String()))
}

func decodeCursor(s string) (*time.Time, *uuid.UUID, error) {
	if s == "" {
		return nil, nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, nil, err
	}
	ns, rest, ok := strings.Cut(string(b), ".")
	if !ok {
		return nil, nil, errors.New("bad cursor")
	}
	n, err := strconv.ParseInt(ns, 10, 64)
	if err != nil {
		return nil, nil, err
	}
	id, err := uuid.Parse(rest)
	if err != nil {
		return nil, nil, err
	}
	at := time.Unix(0, n)
	return &at, &id, nil
}

// window turns a filter's range into times: the last hour, day, week or
// 30 days, or custom dates within the last 90 days.
func (s *Service) window(ctx context.Context, ws *store.Workspace, f LogFilter) (time.Time, time.Time, error) {
	now := s.now()
	until := now.Add(time.Minute)
	oldest := now.Add(-MetadataRetentionDays * 24 * time.Hour)
	switch f.Range {
	case "1h":
		return now.Add(-time.Hour), until, nil
	case "", "24h":
		return now.Add(-24 * time.Hour), until, nil
	case "7d":
		return now.Add(-7 * 24 * time.Hour), until, nil
	case "30d":
		return now.Add(-30 * 24 * time.Hour), until, nil
	case "custom":
		loc := location(ws)
		from, err := time.ParseInLocation("2006-01-02", f.From, loc)
		if err != nil {
			return time.Time{}, time.Time{}, api.Invalid("query.from", say(ctx, "log.startDate", nil))
		}
		to := now
		if f.To != "" {
			t, err := time.ParseInLocation("2006-01-02", f.To, loc)
			if err != nil {
				return time.Time{}, time.Time{}, api.Invalid("query.to", say(ctx, "log.endDate", nil))
			}
			to = t.AddDate(0, 0, 1)
		}
		if to.Before(from) {
			return time.Time{}, time.Time{}, api.Invalid("query.to", say(ctx, "log.endBeforeStart", nil))
		}
		if from.Before(oldest.Add(-24 * time.Hour)) {
			return time.Time{}, time.Time{}, api.Invalid("query.from", say(ctx, "log.tooOld", i18n.Args{"days": MetadataRetentionDays, "date": dayIn(ctx, oldest.In(loc))}))
		}
		return from, to, nil
	}
	return time.Time{}, time.Time{}, api.Invalid("query.range", say(ctx, "log.range", nil))
}

func optionalUUID(ctx context.Context, field, s string) (*uuid.UUID, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, api.Invalid(field, say(ctx, "log.pickOne", nil))
	}
	return &id, nil
}

func optionalText(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// logParams turns a filter into the log query's parameters.
func (s *Service) logParams(ctx context.Context, ws *store.Workspace, f LogFilter) (store.TxnLogParams, error) {
	since, until, err := s.window(ctx, ws, f)
	if err != nil {
		return store.TxnLogParams{}, err
	}
	p := store.TxnLogParams{WorkspaceID: ws.ID, Since: since, Until: until, Lim: int32(f.Limit)}
	if p.Lim <= 0 || p.Lim > 100 {
		p.Lim = 25
	}
	if f.Status != "" && f.Status != "all" {
		if !slices.Contains(logStatuses, f.Status) {
			return p, api.Invalid("query.status", say(ctx, "log.status", nil))
		}
		p.Status = &f.Status
	}
	if t := strings.ToLower(strings.TrimSpace(f.Template)); t != "" {
		p.Template = &t
	}
	if p.ProviderID, err = optionalUUID(ctx, "query.provider", f.Provider); err != nil {
		return p, err
	}
	if p.KeyID, err = optionalUUID(ctx, "query.key", f.Key); err != nil {
		return p, err
	}
	p.Q = optionalText(strings.ToLower(f.Query))
	if p.BeforeAt, p.BeforeID, err = decodeCursor(f.Before); err != nil {
		return p, api.Invalid("query.before", say(ctx, "log.pageStale", nil))
	}
	if p.AfterAt, p.AfterID, err = decodeCursor(f.After); err != nil {
		return p, api.Invalid("query.after", say(ctx, "log.cursorStale", nil))
	}
	return p, nil
}

// detailMap reads a message's reject_detail.
func detailMap(raw json.RawMessage) map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func mapString(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

func mapTime(m map[string]any, k string) time.Time {
	if s, ok := m[k].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func mapStrings(m map[string]any, k string) []string {
	out := []string{}
	if list, ok := m[k].([]any); ok {
		for _, v := range list {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// rejectionOf explains a message stopped before sending (or refused by
// its provider).
func rejectionOf(reason string, raw json.RawMessage, status, errText string) *TransactionalRejection {
	d := detailMap(raw)
	switch reason {
	case "suppressed":
		return &TransactionalRejection{Reason: "suppressed", Suppression: &TransactionalSuppressionInfo{Reason: mapString(d, "reason"), Since: mapTime(d, "since"),
			Code: mapString(d, "code"), Detail: mapString(d, "detail"), Source: mapString(d, "source"), Scope: "all"}}
	case "opted_out":
		return &TransactionalRejection{Reason: "opted_out", OptOut: &TransactionalOptOutInfo{Category: mapString(d, "category"), Since: mapTime(d, "since"), Source: mapString(d, "source")}}
	case "test_not_allowed":
		return &TransactionalRejection{Reason: "test_not_allowed", Allowed: mapStrings(d, "allowed")}
	}
	if status == "rejected" {
		return &TransactionalRejection{Reason: "provider", Error: errText}
	}
	return nil
}

func bounceOf(s string) *TransactionalBounce {
	if s == "" {
		return nil
	}
	kind, rest, _ := strings.Cut(s, ":")
	class, diag, _ := strings.Cut(rest, ":")
	b := &TransactionalBounce{Kind: "soft", Class: class, Diagnostic: strings.TrimSpace(diag)}
	if kind == "hard_bounce" {
		b.Kind = "hard"
	}
	return b
}

func ms(from time.Time, to *time.Time) *int64 {
	if to == nil || to.Before(from) {
		return nil
	}
	n := to.Sub(from).Milliseconds()
	return &n
}

func logItem(r store.TxnLogRow) TransactionalLogItem {
	it := TransactionalLogItem{ID: r.MessageID.String(), At: r.AcceptedAt, Email: r.Email, Name: r.ToName, Subject: r.Subject, Status: r.Status,
		Deferrals: int(r.Deferrals), TestDelivered: r.TestDelivered, RoleAddress: r.RoleAddress, Tracked: r.TrackOpens || r.TrackClicks,
		OpenedAt: r.OpenedAt, ClickedAt: r.ClickedAt, Key: r.KeyName, Category: r.CategoryName, Language: r.Language}
	if r.ContactID != nil {
		it.ContactID = r.ContactID.String()
	}
	if r.TemplateKey == "" && r.Source == "smtp" {
		it.Raw = true
	} else {
		it.Template = &TransactionalLogTemplate{Key: r.TemplateKey, Name: r.TemplateName, Version: r.TemplateVersion}
	}
	if r.ProviderID != nil {
		it.Provider = &TransactionalRef{ID: r.ProviderID.String(), Name: r.ProviderName}
	}
	if r.Status == "delivered" || r.Status == "complained" {
		it.DeliveryMs = ms(r.AcceptedAt, r.DeliveredAt)
	}
	if r.Status == "bounced" {
		it.Bounce = bounceOf(r.Bounce)
		if it.Bounce == nil {
			it.Bounce = &TransactionalBounce{Kind: "soft", Diagnostic: r.Error}
		}
	}
	it.Rejection = rejectionOf(r.RejectReason, r.RejectDetail, r.Status, r.Error)
	if r.FailedOverFrom != nil && r.ProviderID != nil {
		it.FailedOver = &TransactionalFailover{From: r.FailedOverFromName, To: r.ProviderName, At: r.FailedOverAt}
	}
	if r.Status == "failed" {
		it.Error = r.Error
	}
	return it
}

// MessageLog lists the message log.
func (s *Service) MessageLog(ctx context.Context, ws *store.Workspace, f LogFilter) (TransactionalLog, error) {
	p, err := s.logParams(ctx, ws, f)
	if err != nil {
		return TransactionalLog{}, err
	}
	out := TransactionalLog{Items: []TransactionalLogItem{}, Since: p.Since, Until: p.Until, MetadataDays: MetadataRetentionDays, ContentDays: 30}
	rows, err := s.Q.TxnLog(ctx, p)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, itemIn(lang(ctx), logItem(r)))
	}
	if len(rows) > 0 {
		out.Newest = encodeCursor(rows[0].AcceptedAt, rows[0].MessageID)
		if len(rows) == int(p.Lim) {
			last := rows[len(rows)-1]
			out.Next = encodeCursor(last.AcceptedAt, last.MessageID)
		}
	}
	counts, err := s.Q.TxnLogCounts(ctx, store.TxnLogCountsParams{WorkspaceID: ws.ID, Since: p.Since, Until: p.Until, Template: p.Template, ProviderID: p.ProviderID, KeyID: p.KeyID, Q: p.Q})
	if err != nil {
		return out, err
	}
	for _, c := range counts {
		out.Counts.All += c.N
		switch c.Status {
		case "queued":
			out.Counts.Queued += c.N
		case "sent":
			out.Counts.Sent += c.N
		case "delivered":
			out.Counts.Delivered += c.N
		case "deferred":
			out.Counts.Deferred += c.N
		case "bounced":
			out.Counts.Bounced += c.N
		case "complained":
			out.Counts.Complained += c.N
		case "rejected":
			out.Counts.Rejected += c.N
		case "failed":
			out.Counts.Failed += c.N
		case "test":
			out.Counts.Test += c.N
		}
	}
	if out.Filters, err = s.logFilters(ctx, ws.ID); err != nil {
		return out, err
	}
	if st, err := s.settingsRow(ctx, s.Q, ws.ID); err == nil {
		out.ContentDays = int(st.ContentRetentionDays)
	}
	if pr := api.PrincipalFrom(ctx); pr != nil {
		out.CanExport = pr.Can(authz.TransactionalContent)
	}
	return out, nil
}

func (s *Service) logFilters(ctx context.Context, wsID uuid.UUID) (TransactionalLogFilters, error) {
	out := TransactionalLogFilters{Templates: []TransactionalRef{}, Providers: []TransactionalRef{}, Keys: []TransactionalRef{}}
	tpls, err := s.Q.TxnLogTemplates(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, t := range tpls {
		out.Templates = append(out.Templates, TransactionalRef{ID: t.Key, Name: t.Name})
	}
	out.Templates = append(out.Templates, TransactionalRef{ID: "raw", Name: "Raw (SMTP)"})
	provs, err := s.Q.ProvListActive(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, p := range provs {
		out.Providers = append(out.Providers, TransactionalRef{ID: p.ID.String(), Name: p.Name})
	}
	keys, err := s.Q.TxnLogKeys(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, k := range keys {
		out.Keys = append(out.Keys, TransactionalRef{ID: k.ID.String(), Name: k.Name})
	}
	return out, nil
}

// TransactionalTimelineEntry is one step of a message's life.
type TransactionalTimelineEntry struct {
	At   time.Time `json:"at"`
	Kind string    `json:"kind" enum:"accepted,rendered,scheduled,routed,handed,retry,failed_over,failed,rejected,not_delivered,delivered,deferred,bounced,complained,opened,clicked,provider_rejected,webhook"`
	// Provider is who handed, retried, delivered or bounced.
	Provider          string `json:"provider,omitempty"`
	ProviderMessageID string `json:"providerMessageId,omitempty"`
	// Code and Detail are the provider's answer ("250 2.6.0", "Queued mail
	// for delivery"), or the error of an attempt.
	Code   string `json:"code,omitempty"`
	Detail string `json:"detail,omitempty"`
	// Status is the API's answer (accepted: 202; rejected: 422).
	Status int `json:"status,omitempty"`
	// To is the provider a failover went to; Down says the provider was
	// marked down.
	To   string `json:"to,omitempty"`
	Down bool   `json:"down,omitempty"`
	// Event and URL describe a webhook delivery, URL a click too.
	Event string `json:"event,omitempty"`
	URL   string `json:"url,omitempty"`
	State string `json:"state,omitempty"`
	// Machine marks an open by a mail client's image proxy.
	Machine bool `json:"machine,omitempty"`
}

// TransactionalAttempt is one provider call for a message.
type TransactionalAttempt struct {
	Provider  string    `json:"provider"`
	At        time.Time `json:"at"`
	OK        bool      `json:"ok"`
	Error     string    `json:"error,omitempty"`
	Temporary bool      `json:"temporary,omitempty"`
	GaveUp    bool      `json:"gaveUp,omitempty"`
	Down      bool      `json:"down,omitempty"`
}

// TransactionalRequest is the call that produced a message, as stored:
// sensitive values are ••••••.
type TransactionalRequest struct {
	Method         string          `json:"method"`
	Path           string          `json:"path"`
	IdempotencyKey string          `json:"idempotencyKey,omitempty"`
	Body           json.RawMessage `json:"body"`
	// SMTP: mail that arrived through the relay (no JSON body).
	SMTP bool `json:"smtp"`
}

// TransactionalContentState says whether the rendered content can be shown.
type TransactionalContentState struct {
	Available bool       `json:"available"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	RemovedAt *time.Time `json:"removedAt,omitempty"`
	// CanView: viewing content needs transactional:content (and is
	// audited); Helpers can grant it.
	CanView       bool      `json:"canView"`
	Helpers       []string  `json:"helpers"`
	RetentionDays int       `json:"retentionDays"`
	MetadataUntil time.Time `json:"metadataUntil"`
	// Masked names the sensitive variables shown as ••••••.
	Masked []string `json:"masked"`
}

// TransactionalResendState says whether a person can resend the message.
type TransactionalResendState struct {
	Available bool `json:"available"`
	// Reason when it isn't: sensitive (a sensitive variable wasn't stored),
	// attachments (files weren't stored), raw (SMTP mail), template (the
	// template isn't published any more), permission (needs
	// transactional:send), not_sent (it was never accepted for sending).
	Reason    string   `json:"reason,omitempty" enum:"sensitive,attachments,raw,template,permission,not_sent,"`
	Sensitive []string `json:"sensitive,omitempty"`
}

// TransactionalWebhookDelivery is a message.* event sent to an endpoint.
type TransactionalWebhookDelivery struct {
	Event      string     `json:"event"`
	State      string     `json:"state"`
	At         time.Time  `json:"at"`
	Delivered  *time.Time `json:"deliveredAt,omitempty"`
	StatusCode *int32     `json:"statusCode,omitempty"`
	URL        string     `json:"url"`
}

// TransactionalAttachmentMeta describes a file sent with a message (files
// aren't stored).
type TransactionalAttachmentMeta struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
	ContentID   string `json:"contentId,omitempty"`
}

// TransactionalMessageDetail is the message detail panel.
type TransactionalMessageDetail struct {
	TransactionalLogItem
	Contact           *TransactionalRef `json:"contact,omitempty"`
	CategoryKind      string            `json:"categoryKind,omitempty"`
	ProviderMessageID string            `json:"providerMessageId,omitempty"`
	KeyRef            *TransactionalRef `json:"keyRef,omitempty"`
	IP                string            `json:"ip,omitempty"`
	IdempotencyKey    string            `json:"idempotencyKey,omitempty"`
	Tags              []string          `json:"tags"`
	Metadata          map[string]string `json:"metadata"`
	From              string            `json:"from"`
	ReplyTo           string            `json:"replyTo,omitempty"`
	Cc                []string          `json:"cc"`
	Bcc               []string          `json:"bcc"`
	// Dropped are copies left out because the address is suppressed.
	Dropped     []string                      `json:"dropped"`
	Attachments []TransactionalAttachmentMeta `json:"attachments"`
	// ContentBytes is the HTML and text; SizeBytes adds the attachments.
	ContentBytes int `json:"contentBytes"`
	SizeBytes    int `json:"sizeBytes"`
	Variables    int `json:"variables"`
	// SensitiveSent are the sensitive variables the call sent.
	SensitiveSent []string `json:"sensitiveSent"`
	// Source: api, batch, smtp, resend, guide or test; SentBy is the person
	// for resend, guide and test sends.
	Source     string                         `json:"source"`
	SentBy     string                         `json:"sentBy,omitempty"`
	ResentFrom string                         `json:"resentFrom,omitempty"`
	SendAt     *time.Time                     `json:"sendAt,omitempty"`
	Route      TransactionalRouteInfo         `json:"route"`
	Timeline   []TransactionalTimelineEntry   `json:"timeline"`
	Attempts   []TransactionalAttempt         `json:"attempts"`
	Request    TransactionalRequest           `json:"request"`
	Webhooks   []TransactionalWebhookDelivery `json:"webhooks"`
	// Suppression is the address's suppression as it is now (bounced
	// messages: added by this bounce).
	Suppression *TransactionalSuppressionInfo `json:"suppression,omitempty"`
	// SoftBounces counts the address's soft bounces in 7 days (limit 3).
	SoftBounces     *int                      `json:"softBounces,omitempty"`
	SoftBounceLimit int                       `json:"softBounceLimit"`
	Content         TransactionalContentState `json:"content"`
	Resend          TransactionalResendState  `json:"resend"`
	// CanRemoveSuppression: removing needs subscribers:edit.
	CanRemoveSuppression bool `json:"canRemoveSuppression"`
	// Language: the version sent and why; nil for a template in one language.
	Language *TransactionalMessageLanguage `json:"languageInfo,omitempty"`
}

// TransactionalRouteInfo is how a message was routed.
type TransactionalRouteInfo struct {
	Rule     string `json:"rule"`
	Matched  string `json:"matched,omitempty" doc:"The condition's field: category or mailbox_provider"`
	Value    string `json:"value,omitempty"`
	Primary  string `json:"primary,omitempty"`
	Fallback string `json:"fallback,omitempty"`
	Policy   string `json:"policy,omitempty"`
	// FailedOver: the primary was down when the call came, so the message
	// went to the fallback at once.
	FailedOver bool `json:"failedOver"`
}

// messageID parses a message's ID from its path.
func messageID(ctx context.Context, s string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return uuid.Nil, problem(http.StatusNotFound, "not_found", say(ctx, "message.notFound", i18n.Args{"id": clip(s, 40), "days": MetadataRetentionDays}))
	}
	return id, nil
}

func (s *Service) detailRow(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.TxnMessageDetailRow, error) {
	r, err := s.Q.TxnMessageDetail(ctx, store.TxnMessageDetailParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return r, problem(http.StatusNotFound, "not_found", say(ctx, "message.notFound", i18n.Args{"id": id.String(), "days": MetadataRetentionDays}))
	}
	return r, err
}

// Message returns the message detail.
func (s *Service) Message(ctx context.Context, ws *store.Workspace, id uuid.UUID) (TransactionalMessageDetail, error) {
	r, err := s.detailRow(ctx, ws, id)
	if err != nil {
		return TransactionalMessageDetail{}, err
	}
	pr := api.PrincipalFrom(ctx)
	item := itemIn(lang(ctx), logItem(store.TxnLogRow{MessageID: r.MessageID, AcceptedAt: r.AcceptedAt, Email: r.Email, ContactID: r.ContactID, ToName: r.ToName, Source: r.Source,
		TemplateKey: r.TemplateKey, TemplateName: r.TemplateName, TemplateVersion: r.TemplateVersion, Subject: r.Subject, ProviderID: r.ProviderID,
		ProviderName: r.ProviderName, Error: r.Error, SentAt: r.SentAt, DeliveredAt: r.DeliveredAt, OpenedAt: r.OpenedAt, ClickedAt: r.ClickedAt,
		Test: r.Test, TestDelivered: r.TestDelivered, RejectReason: r.RejectReason, RejectDetail: r.RejectDetail, TrackOpens: r.TrackOpens,
		TrackClicks: r.TrackClicks, FailedOverFrom: r.FailedOverFrom, FailedOverFromName: r.FailedOverFromName, FailedOverAt: r.FailedOverAt,
		KeyID: r.KeyID, KeyName: r.KeyName, IdempotencyKey: r.IdempotencyKey, Tags: r.Tags, Metadata: r.Metadata, CategoryName: r.CategoryName,
		CategoryKind: r.CategoryKind, Status: r.Status, RoleAddress: r.RoleAddress, Language: r.Language}))
	out := TransactionalMessageDetail{TransactionalLogItem: item, CategoryKind: r.CategoryKind, ProviderMessageID: r.ProviderMessageID,
		IdempotencyKey: r.IdempotencyKey, Tags: nonNilStrings(r.Tags), Metadata: map[string]string{}, From: addrLine(r.FromName, r.FromEmail), ReplyTo: r.ReplyTo,
		Cc: nonNilStrings(r.Cc), Bcc: nonNilStrings(r.Bcc), Dropped: []string{}, Attachments: []TransactionalAttachmentMeta{}, SizeBytes: int(r.SizeBytes),
		Variables: len(r.DataKeys), SensitiveSent: []string{}, Source: r.Source, SendAt: r.SendAt, SoftBounceLimit: SoftBounceLimit,
		Timeline: []TransactionalTimelineEntry{}, Attempts: []TransactionalAttempt{}, Webhooks: []TransactionalWebhookDelivery{}}
	_ = json.Unmarshal(r.Metadata, &out.Metadata)
	if r.Language != "" || r.LanguageReason != "" {
		out.Language = &TransactionalMessageLanguage{Sent: r.Language, Contact: r.ContactLanguage, Reason: r.LanguageReason}
	}
	if r.ContactID != nil && r.ContactExists {
		name := r.ContactName
		if name == "" {
			name = r.Email
		}
		out.Contact = &TransactionalRef{ID: r.ContactID.String(), Name: name}
	}
	if r.KeyID != nil {
		out.KeyRef = &TransactionalRef{ID: r.KeyID.String(), Name: r.KeyName}
	}
	if r.Ip != nil {
		out.IP = r.Ip.String()
	}
	if r.Source == "resend" || r.Source == "guide" || r.Source == "test" {
		out.SentBy = r.UserName
	}
	if r.ResentFrom != nil {
		out.ResentFrom = r.ResentFrom.String()
	}
	d := detailMap(r.RejectDetail)
	out.Dropped = mapStrings(d, "dropped")
	var files []map[string]any
	_ = json.Unmarshal(r.Attachments, &files)
	attachBytes := 0
	for _, f := range files {
		size := 0
		if v, ok := f["size"].(float64); ok {
			size = int(v)
		}
		attachBytes += base64.StdEncoding.EncodedLen(size)
		out.Attachments = append(out.Attachments, TransactionalAttachmentMeta{Filename: mapString(f, "filename"), ContentType: mapString(f, "content_type"), Size: size, ContentID: mapString(f, "content_id")})
	}
	out.ContentBytes = max(int(r.SizeBytes)-attachBytes, 0)
	for _, k := range r.DataKeys {
		if slices.Contains(r.Sensitive, k) {
			out.SensitiveSent = append(out.SensitiveSent, k)
		}
	}
	var rc routeChoice
	_ = json.Unmarshal(r.Route, &rc)
	out.Route = TransactionalRouteInfo{Rule: rc.Rule, Matched: rc.Matched, Value: rc.Value, Primary: rc.PrimaryName, Fallback: rc.FallbackName, Policy: rc.Policy, FailedOver: rc.FailedOver}
	var tried []struct {
		Provider  string    `json:"provider"`
		At        time.Time `json:"at"`
		OK        bool      `json:"ok"`
		Error     string    `json:"error"`
		Temporary bool      `json:"temporary"`
		GaveUp    bool      `json:"gave_up"`
		Down      bool      `json:"down"`
		PMID      string    `json:"provider_message_id"`
	}
	_ = json.Unmarshal(r.Attempts, &tried)
	for _, a := range tried {
		out.Attempts = append(out.Attempts, TransactionalAttempt{Provider: a.Provider, At: a.At, OK: a.OK, Error: a.Error, Temporary: a.Temporary, GaveUp: a.GaveUp, Down: a.Down})
	}
	// The request lists the data in the template's order, as the app's call did.
	var order []string
	if r.TemplateID != nil && r.TemplateVersion != nil {
		if v, err := s.Q.TxnVersion(ctx, store.TxnVersionParams{TemplateID: *r.TemplateID, Version: *r.TemplateVersion}); err == nil {
			var vars []TransactionalVariable
			_ = json.Unmarshal(v.Variables, &vars)
			for _, x := range vars {
				order = append(order, x.Name)
			}
		}
	}
	out.Request = requestOf(ws, r, order)

	// The timeline: the call, the render, the route, each provider call,
	// then the provider's events and the webhook deliveries.
	tl := []TransactionalTimelineEntry{}
	accepted := TransactionalTimelineEntry{At: r.AcceptedAt, Kind: "accepted", Status: http.StatusAccepted}
	if r.RejectReason == "suppressed" || r.RejectReason == "opted_out" {
		accepted.Status = http.StatusUnprocessableEntity
	}
	tl = append(tl, accepted)
	if r.RenderedAt != nil {
		tl = append(tl, TransactionalTimelineEntry{At: *r.RenderedAt, Kind: "rendered"})
	}
	switch r.RejectReason {
	case "suppressed", "opted_out":
		tl = append(tl, TransactionalTimelineEntry{At: renderedOr(r), Kind: "rejected", Detail: r.RejectReason})
	case "test_not_allowed":
		tl = append(tl, TransactionalTimelineEntry{At: renderedOr(r), Kind: "not_delivered"})
	default:
		routed := TransactionalTimelineEntry{At: renderedOr(r).Add(6 * time.Millisecond), Kind: "routed", Provider: rc.ProviderName}
		if rc.FailedOver {
			routed.Detail, routed.To = rc.PrimaryName, rc.ProviderName
		}
		tl = append(tl, routed)
		if r.SendAt != nil {
			tl = append(tl, TransactionalTimelineEntry{At: renderedOr(r).Add(7 * time.Millisecond), Kind: "scheduled", Detail: r.SendAt.UTC().Format(time.RFC3339)})
		}
	}
	for i, a := range tried {
		e := TransactionalTimelineEntry{At: a.At, Provider: a.Provider}
		switch {
		case a.OK:
			e.Kind, e.ProviderMessageID = "handed", a.PMID
		case a.GaveUp && a.Down:
			e.Kind, e.Down, e.Detail = "failed_over", true, a.Error
		case a.GaveUp:
			e.Kind, e.Detail = "failed_over", a.Error
		default:
			e.Kind, e.Detail = "retry", a.Error
		}
		if e.Kind == "failed_over" {
			// Where it went next: the next attempt's provider, or nowhere.
			for _, b := range tried[i+1:] {
				if b.Provider != a.Provider {
					e.To = b.Provider
					break
				}
			}
			if e.To == "" && r.RawStatus == "failed" {
				e.Kind = "failed"
			} else if e.To == "" && r.ProviderName != a.Provider {
				e.To = r.ProviderName
			}
		}
		tl = append(tl, e)
	}
	if r.RawStatus == "failed" && (len(tried) == 0 || !tried[len(tried)-1].GaveUp) {
		tl = append(tl, TransactionalTimelineEntry{At: lastAt(tl), Kind: "failed", Detail: r.Error})
	}
	events, err := s.Q.TxnMessageEvents(ctx, &id)
	if err != nil {
		return out, err
	}
	for _, e := range events {
		en := TransactionalTimelineEntry{At: e.OccurredAt, Provider: e.ProviderName, Code: e.OriginalCode, Detail: e.Diagnostic, Machine: e.Machine}
		switch e.Type {
		case "delivered":
			en.Kind = "delivered"
		case "deferred":
			en.Kind = "deferred"
		case "hard_bounce", "soft_bounce":
			en.Kind = "bounced"
		case "complaint":
			en.Kind = "complained"
		case "open":
			en.Kind = "opened"
		case "click":
			en.Kind, en.URL = "clicked", e.Url
		case "rejected":
			en.Kind = "provider_rejected"
		default:
			continue
		}
		tl = append(tl, en)
	}
	hooks, err := s.Q.TxnMessageWebhooks(ctx, store.TxnMessageWebhooksParams{MessageID: id.String(), WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	for _, h := range hooks {
		out.Webhooks = append(out.Webhooks, TransactionalWebhookDelivery{Event: h.Event, State: h.State, At: h.CreatedAt, Delivered: h.DeliveredAt, StatusCode: h.StatusCode, URL: h.Url})
		at := h.CreatedAt
		if h.DeliveredAt != nil {
			at = *h.DeliveredAt
		}
		tl = append(tl, TransactionalTimelineEntry{At: at, Kind: "webhook", Event: h.Event, URL: h.Url, State: h.State})
	}
	sort.SliceStable(tl, func(i, j int) bool { return tl[i].At.Before(tl[j].At) })
	out.Timeline = tl

	if sup, err := s.Q.TxnSuppressionFor(ctx, store.TxnSuppressionForParams{WorkspaceID: ws.ID, Email: r.Email}); err == nil {
		out.Suppression = &TransactionalSuppressionInfo{ID: sup.ID.String(), Reason: sup.Reason, Since: sup.CreatedAt, Code: sup.OriginalCode, Detail: sup.Detail,
			Source: suppressionSource(sup.Source, sup.ProviderName), Scope: sup.Scope}
		if out.Rejection != nil && out.Rejection.Suppression != nil {
			out.Rejection.Suppression.ID = sup.ID.String()
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if out.Bounce != nil && out.Bounce.Kind == "soft" {
		n, err := s.Q.TxnSoftBounces(ctx, store.TxnSoftBouncesParams{WorkspaceID: ws.ID, Recipient: r.Email, Since: s.now().Add(-SoftBounceWindow)})
		if err != nil {
			return out, err
		}
		c := int(n)
		out.SoftBounces = &c
	}
	retention := 30
	if st, err := s.settingsRow(ctx, s.Q, ws.ID); err == nil {
		retention = int(st.ContentRetentionDays)
	}
	out.Content = TransactionalContentState{Available: r.HasContent, ExpiresAt: r.ContentExpiresAt, RemovedAt: r.ContentRemovedAt, RetentionDays: retention,
		MetadataUntil: r.AcceptedAt.Add(MetadataRetentionDays * 24 * time.Hour), Masked: out.SensitiveSent, Helpers: []string{}}
	if !r.HasContent {
		out.Content.ExpiresAt = nil
	}
	if pr != nil {
		out.Content.CanView = pr.Can(authz.TransactionalContent)
		out.CanRemoveSuppression = pr.Can(authz.SubscribersEdit)
	}
	if !out.Content.CanView {
		out.Content.Helpers = s.helpers(ctx, ws.ID, authz.TransactionalContent, 2)
	}
	out.Resend = s.resendState(ctx, ws, r, pr)
	return out, nil
}

func renderedOr(r store.TxnMessageDetailRow) time.Time {
	if r.RenderedAt != nil {
		return *r.RenderedAt
	}
	return r.AcceptedAt
}

func lastAt(tl []TransactionalTimelineEntry) time.Time {
	at := time.Time{}
	for _, e := range tl {
		if e.At.After(at) {
			at = e.At
		}
	}
	return at
}

func addrLine(name, email string) string {
	if name == "" {
		return email
	}
	return name + " <" + email + ">"
}

// requestOf rebuilds the call as stored: sensitive values are ••••••, files
// are listed without their content. Its fields come in the order an app
// writes them (template, to, data in the template's order, tags,
// metadata), not alphabetically.
func requestOf(ws *store.Workspace, r store.TxnMessageDetailRow, order []string) TransactionalRequest {
	out := TransactionalRequest{Method: http.MethodPost, Path: "/api/v1/workspaces/" + ws.Slug + "/transactional/messages", IdempotencyKey: r.IdempotencyKey}
	if r.Source == "smtp" {
		out.SMTP, out.Method, out.Path = true, "SMTP", ""
		out.Body = json.RawMessage(`{}`)
		return out
	}
	if r.Source == "batch" {
		out.Path += "/batch"
	}
	body := ordered{{"template", r.TemplateKey}}
	to := ordered{{"email", r.Email}}
	if r.ToName != "" {
		to = append(to, orderedField{"name", r.ToName})
	}
	body = append(body, orderedField{"to", to})
	addrs := func(list []string) []map[string]string {
		out := []map[string]string{}
		for _, a := range list {
			out = append(out, map[string]string{"email": a})
		}
		return out
	}
	if len(r.Cc) > 0 {
		body = append(body, orderedField{"cc", addrs(r.Cc)})
	}
	if len(r.Bcc) > 0 {
		body = append(body, orderedField{"bcc", addrs(r.Bcc)})
	}
	var data map[string]any
	_ = json.Unmarshal(r.Data, &data)
	if len(data) > 0 {
		body = append(body, orderedField{"data", inOrder(data, order)})
	}
	if len(r.Tags) > 0 {
		body = append(body, orderedField{"tags", r.Tags})
	}
	var meta map[string]any
	_ = json.Unmarshal(r.Metadata, &meta)
	if len(meta) > 0 {
		body = append(body, orderedField{"metadata", inOrder(meta, nil)})
	}
	if r.SendAt != nil {
		body = append(body, orderedField{"send_at", r.SendAt.UTC().Format(time.RFC3339)})
	}
	var files []map[string]any
	_ = json.Unmarshal(r.Attachments, &files)
	if len(files) > 0 {
		list := []map[string]any{}
		for _, f := range files {
			list = append(list, map[string]any{"filename": f["filename"], "content_type": f["content_type"], "content": "(not stored)"})
		}
		body = append(body, orderedField{"attachments", list})
	}
	out.Body = mustJSON(body)
	return out
}

// ordered is a JSON object that keeps its fields in order.
type ordered []orderedField

type orderedField struct {
	Key   string
	Value any
}

func (o ordered) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, f := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		k, err := json.Marshal(f.Key)
		if err != nil {
			return nil, err
		}
		v, err := json.Marshal(f.Value)
		if err != nil {
			return nil, err
		}
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// inOrder lists a map's fields in the given order, then the others
// alphabetically.
func inOrder(m map[string]any, order []string) ordered {
	out := make(ordered, 0, len(m))
	seen := map[string]bool{}
	for _, k := range order {
		if v, ok := m[k]; ok && !seen[k] {
			out = append(out, orderedField{k, v})
			seen[k] = true
		}
	}
	rest := make([]string, 0, len(m))
	for k := range m {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		out = append(out, orderedField{k, m[k]})
	}
	return out
}

// resendState says whether a person can resend the message now.
func (s *Service) resendState(ctx context.Context, ws *store.Workspace, r store.TxnMessageDetailRow, pr *authz.Principal) TransactionalResendState {
	switch {
	case r.Source == "smtp":
		return TransactionalResendState{Reason: "raw"}
	case len(sensitiveIn(r.DataKeys, r.Sensitive)) > 0:
		return TransactionalResendState{Reason: "sensitive", Sensitive: sensitiveIn(r.DataKeys, r.Sensitive)}
	case string(r.Attachments) != "" && string(r.Attachments) != "[]" && string(r.Attachments) != "null":
		return TransactionalResendState{Reason: "attachments"}
	case pr == nil || !pr.Can(authz.TransactionalSend) || pr.Type != authz.ActorUser:
		return TransactionalResendState{Reason: "permission"}
	}
	if _, err := s.live(ctx, s.Q, ws.ID, r.TemplateKey); err != nil {
		return TransactionalResendState{Reason: "template"}
	}
	return TransactionalResendState{Available: true}
}

func sensitiveIn(keys, sensitive []string) []string {
	out := []string{}
	for _, k := range keys {
		if slices.Contains(sensitive, k) {
			out = append(out, k)
		}
	}
	return out
}

// TransactionalContent is a message's rendered content, as stored:
// sensitive values are ••••••.
type TransactionalContent struct {
	HTML      string                    `json:"html"`
	Text      string                    `json:"text"`
	Headers   []TransactionalHeaderPair `json:"headers"`
	ExpiresAt time.Time                 `json:"expiresAt"`
	Masked    []string                  `json:"masked"`
}

// TransactionalHeaderPair is one header.
type TransactionalHeaderPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

var headerOrder = []string{"Date", "From", "To", "Cc", "Reply-To", "Subject", "Message-ID", "List-Unsubscribe", "List-Unsubscribe-Post", "Feedback-ID"}

// Content returns a message's rendered content; every view is audited,
// because content can hold personal data.
func (s *Service) Content(ctx context.Context, ws *store.Workspace, id uuid.UUID) (TransactionalContent, error) {
	r, err := s.detailRow(ctx, ws, id)
	if err != nil {
		return TransactionalContent{}, err
	}
	c, err := s.Q.TxnContent(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		detail := say(ctx, "content.notKept", nil)
		if r.ContentRemovedAt != nil {
			detail = say(ctx, "content.removed", i18n.Args{"date": dayIn(ctx, r.ContentRemovedAt.In(location(ws))),
				"until": dayIn(ctx, r.AcceptedAt.Add(MetadataRetentionDays*24*time.Hour).In(location(ws)))})
		}
		return TransactionalContent{}, problem(http.StatusGone, "content_removed", detail)
	}
	if err != nil {
		return TransactionalContent{}, err
	}
	out := TransactionalContent{HTML: c.Html, Text: c.Text, ExpiresAt: c.ExpiresAt, Headers: []TransactionalHeaderPair{}, Masked: sensitiveIn(r.DataKeys, r.Sensitive)}
	hdr := map[string]string{}
	_ = json.Unmarshal(c.Headers, &hdr)
	for _, k := range headerOrder {
		if v, ok := hdr[k]; ok {
			out.Headers = append(out.Headers, TransactionalHeaderPair{Name: k, Value: v})
			delete(hdr, k)
		}
	}
	rest := make([]string, 0, len(hdr))
	for k := range hdr {
		rest = append(rest, k)
	}
	sort.Strings(rest)
	for _, k := range rest {
		out.Headers = append(out.Headers, TransactionalHeaderPair{Name: k, Value: hdr[k]})
	}
	wsID := ws.ID
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: "transactional.content_viewed", TargetType: "transactional_message", TargetID: id.String(),
		TargetName: logName(r.TemplateName, r.Subject) + " to " + r.Email, Metadata: map[string]any{"template": r.TemplateKey, "recipient": r.Email, "summary": "Viewed the content of " + logName(r.TemplateName, r.Subject) + " to " + r.Email}}); err != nil {
		return TransactionalContent{}, err
	}
	return out, nil
}

// logName names a message: its template, or its subject for Raw (SMTP).
func logName(template, subject string) string {
	if template != "" {
		return template
	}
	return "Raw (SMTP) “" + clip(subject, 60) + "”"
}

// Resend sends a message again with the same data and a new message ID.
// People only; audited.
func (s *Service) Resend(ctx context.Context, ws *store.Workspace, id uuid.UUID) (TransactionalSendResult, error) {
	r, err := s.detailRow(ctx, ws, id)
	if err != nil {
		return TransactionalSendResult{}, err
	}
	pr := api.PrincipalFrom(ctx)
	st := s.resendState(ctx, ws, r, pr)
	if !st.Available {
		switch st.Reason {
		case "sensitive":
			return TransactionalSendResult{}, problem(http.StatusConflict, "resend_unavailable",
				say(ctx, "resend.sensitive", i18n.Args{"names": i18n.AndList(i18n.LangFrom(ctx).Locale, st.Sensitive), "count": len(st.Sensitive)}))
		case "attachments":
			return TransactionalSendResult{}, problem(http.StatusConflict, "resend_unavailable", say(ctx, "resend.attachments", nil))
		case "raw":
			return TransactionalSendResult{}, problem(http.StatusConflict, "resend_unavailable", say(ctx, "resend.raw", nil))
		case "template":
			return TransactionalSendResult{}, problem(http.StatusConflict, "resend_unavailable", say(ctx, "resend.template", i18n.Args{"key": r.TemplateKey}))
		default:
			return TransactionalSendResult{}, api.Err(http.StatusForbidden, api.CodePermissionDenied, deniedMessage(ctx, "resend", authz.TransactionalSend, s.helpers(ctx, ws.ID, authz.TransactionalSend, 3)))
		}
	}
	in := TransactionalSend{Template: r.TemplateKey, To: TransactionalAddress{Email: r.Email, Name: r.ToName}, Tags: r.Tags}
	_ = json.Unmarshal(r.Data, &in.Data)
	_ = json.Unmarshal(r.Metadata, &in.Metadata)
	for _, a := range r.Cc {
		in.Cc = append(in.Cc, TransactionalAddress{Email: a})
	}
	for _, a := range r.Bcc {
		in.Bcc = append(in.Bcc, TransactionalAddress{Email: a})
	}
	if r.FromEmail != "" {
		in.From = addrLine(r.FromName, r.FromEmail)
	}
	c, err := s.callerFrom(ctx, ws, "resend")
	if err != nil {
		return TransactionalSendResult{}, err
	}
	if err := s.rateLimit(c, 1); err != nil {
		return TransactionalSendResult{}, err
	}
	p, err := s.prepare(ctx, ws, c, in, s.now())
	if err != nil {
		return TransactionalSendResult{}, err
	}
	p.resentFrom = &id
	var res any
	var status int
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		if status, res, err = s.sendOne(ctx, tx, ws, c, p); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &wsID, Action: "transactional.message_resent", TargetType: "transactional_message", TargetID: p.id.String(),
			TargetName: r.TemplateName + " to " + r.Email, Metadata: map[string]any{"resent_from": id.String(), "template": r.TemplateKey, "recipient": r.Email,
				"summary": "Resent " + r.TemplateName + " to " + r.Email}})
	})
	if err != nil {
		return TransactionalSendResult{}, err
	}
	if status >= 400 {
		if pr, ok := res.(*Problem); ok {
			return TransactionalSendResult{}, pr
		}
	}
	out, _ := res.(TransactionalSendResult)
	return out, nil
}

// Export writes the filtered log as CSV (up to ExportMax rows). It needs
// transactional:content, because it holds addresses, and is audited.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, f LogFilter) ([]byte, int, error) {
	f.Before, f.After = "", ""
	p, err := s.logParams(ctx, ws, f)
	if err != nil {
		return nil, 0, err
	}
	p.Lim = ExportMax
	rows, err := s.Q.TxnLog(ctx, p)
	if err != nil {
		return nil, 0, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"accepted_at", "message_id", "recipient", "template", "version", "subject", "provider", "status", "detail", "key", "idempotency_key", "tags", "metadata"})
	for _, r := range rows {
		it := logItem(r)
		tpl, version := "Raw (SMTP)", ""
		if it.Template != nil {
			tpl = it.Template.Key
			if it.Template.Version != nil {
				version = strconv.Itoa(int(*it.Template.Version))
			}
		}
		provider := ""
		if it.Provider != nil {
			provider = it.Provider.Name
		}
		_ = w.Write([]string{r.AcceptedAt.UTC().Format(time.RFC3339Nano), it.ID, r.Email, tpl, version, r.Subject, provider, r.Status, exportDetail(it), r.KeyName,
			r.IdempotencyKey, strings.Join(r.Tags, " "), string(r.Metadata)})
	}
	w.Flush()
	wsID := ws.ID
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: "transactional.log_exported", TargetType: "transactional_log", TargetID: ws.ID.String(), TargetName: ws.Name,
		Metadata: map[string]any{"rows": len(rows), "status": f.Status, "template": f.Template, "provider": f.Provider, "key": f.Key, "range": f.Range, "query": f.Query,
			"summary": fmt.Sprintf("Exported %d %s of the message log", len(rows), plural(len(rows), "message", "messages"))}}); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(rows), nil
}

// exportDetail is the status detail in words for the CSV.
func exportDetail(it TransactionalLogItem) string {
	switch {
	case it.Rejection != nil && it.Rejection.Reason == "suppressed" && it.Rejection.Suppression != nil:
		return "suppressed: " + reasonLabel(it.Rejection.Suppression.Reason)
	case it.Rejection != nil && it.Rejection.Reason == "opted_out" && it.Rejection.OptOut != nil:
		return "opted out of " + it.Rejection.OptOut.Category
	case it.Rejection != nil && it.Rejection.Reason == "test_not_allowed":
		return "test: not delivered"
	case it.Status == "test":
		return "test: delivered to an allowed address"
	case it.Bounce != nil:
		return it.Bounce.Kind + ": " + it.Bounce.Diagnostic
	case it.DeliveryMs != nil:
		return fmt.Sprintf("%.1f s", float64(*it.DeliveryMs)/1000)
	case it.Error != "":
		return it.Error
	}
	return ""
}

// TransactionalContactActivity is a contact's transactional mail.
type TransactionalContactActivity struct {
	Messages []TransactionalContactMessage `json:"messages"`
	// OptOuts are the Optional categories the address turned off.
	OptOuts []TransactionalContactOptOut `json:"optOuts"`
	// Categories are every Optional category with whether it is on.
	Categories []TransactionalContactCategory `json:"categories"`
	// CanViewContent: the message content needs transactional:content.
	CanViewContent bool `json:"canViewContent"`
}

// TransactionalContactMessage is one message in a contact's timeline.
type TransactionalContactMessage struct {
	ID       string                    `json:"id"`
	At       time.Time                 `json:"at"`
	Template *TransactionalLogTemplate `json:"template,omitempty"`
	Raw      bool                      `json:"raw"`
	Subject  string                    `json:"subject"`
	Status   string                    `json:"status" enum:"queued,sent,delivered,deferred,bounced,complained,rejected,failed,test"`
	Provider string                    `json:"provider,omitempty"`
	Category string                    `json:"category,omitempty"`
	// DeliveryMs: from the call to the delivery event, once delivered.
	DeliveryMs *int64 `json:"deliveryMs,omitempty"`
}

// TransactionalContactOptOut is an Optional category an address turned off.
type TransactionalContactOptOut struct {
	CategoryID string    `json:"categoryId"`
	Category   string    `json:"category"`
	Since      time.Time `json:"since"`
	Source     string    `json:"source"`
	Reason     string    `json:"reason,omitempty"`
}

// TransactionalContactCategory is an Optional category for an address.
type TransactionalContactCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	On   bool   `json:"on"`
}

// ContactActivity lists a contact's transactional mail and opt-outs.
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) (TransactionalContactActivity, error) {
	out := TransactionalContactActivity{Messages: []TransactionalContactMessage{}, OptOuts: []TransactionalContactOptOut{}, Categories: []TransactionalContactCategory{}}
	c, err := s.Q.GetContact(ctx, store.GetContactParams{WorkspaceID: ws.ID, ID: contactID})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, notFound(ctx, "contact")
	}
	if err != nil {
		return out, err
	}
	rows, err := s.Q.TxnContactMessages(ctx, store.TxnContactMessagesParams{WorkspaceID: ws.ID, ContactID: &contactID, Email: c.Email, Lim: 50})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		m := TransactionalContactMessage{ID: r.MessageID.String(), At: r.AcceptedAt, Subject: r.Subject, Status: r.Status, Provider: r.ProviderName, Category: r.CategoryName}
		if r.Status == "delivered" || r.Status == "complained" {
			m.DeliveryMs = ms(r.AcceptedAt, r.DeliveredAt)
		}
		if r.TemplateKey == "" && r.Source == "smtp" {
			m.Raw = true
		} else {
			m.Template = &TransactionalLogTemplate{Key: r.TemplateKey, Name: r.TemplateName}
		}
		out.Messages = append(out.Messages, m)
	}
	offs, err := s.Q.TxnOptOutsFor(ctx, store.TxnOptOutsForParams{WorkspaceID: ws.ID, Email: c.Email})
	if err != nil {
		return out, err
	}
	for _, o := range offs {
		out.OptOuts = append(out.OptOuts, TransactionalContactOptOut{CategoryID: o.CategoryID.String(), Category: o.CategoryName, Since: o.OptedOutAt, Source: o.Source, Reason: o.Reason})
	}
	set, err := s.Notifications(ctx, ws, c.Email)
	if err != nil {
		return out, err
	}
	for _, nc := range set.Categories {
		out.Categories = append(out.Categories, TransactionalContactCategory{ID: nc.ID.String(), Name: nc.Name, On: nc.On})
	}
	if pr := api.PrincipalFrom(ctx); pr != nil {
		out.CanViewContent = pr.Can(authz.TransactionalContent)
	}
	return out, nil
}
