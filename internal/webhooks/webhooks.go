// Package webhooks delivers normalized ReputeCast events to the endpoints a
// workspace registers (main spec 7.13). Every request is signed with
// HMAC-SHA256 using a write-only secret, retried with backoff (then kept in
// the dead-letter queue for 7 days), and protected against SSRF.
package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// Headers sent with every delivery.
const (
	HeaderSignature = "X-ReputeCast-Signature"
	HeaderTimestamp = "X-ReputeCast-Timestamp"
	HeaderEvent     = "X-ReputeCast-Event"
	HeaderDelivery  = "X-ReputeCast-Delivery"
)

// MaxAttempts is how often a delivery is tried before it goes to the
// dead-letter queue.
const MaxAttempts = 8

// Backoff after attempt n (1-based): 1, 2, 4, 8, 30, 90 and 225 minutes, so
// eight attempts span six hours.
var Backoff = []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 30 * time.Minute, 90 * time.Minute, 225 * time.Minute}

// WebhookEventInfo describes an event endpoints can subscribe to.
type WebhookEventInfo struct {
	Name        string `json:"name" example:"email.bounced"`
	Group       string `json:"group" example:"Email"`
	Description string `json:"description"`
}

// Events is the catalogue of events, in display order.
var Events = []WebhookEventInfo{
	{Name: "contact.subscribed", Group: "Contacts"},
	{Name: "contact.unsubscribed", Group: "Contacts"},
	{Name: "contact.updated", Group: "Contacts"},
	{Name: "contact.erased", Group: "Contacts"},
	{Name: "email.sent", Group: "Email"},
	{Name: "email.delivered", Group: "Email"},
	{Name: "email.deferred", Group: "Email"},
	{Name: "email.bounced", Group: "Email"},
	{Name: "email.complained", Group: "Email"},
	{Name: "email.opened", Group: "Email"},
	{Name: "email.clicked", Group: "Email"},
	{Name: "campaign.scheduled", Group: "Campaigns"},
	{Name: "campaign.started", Group: "Campaigns"},
	{Name: "campaign.paused", Group: "Campaigns"},
	{Name: "campaign.resumed", Group: "Campaigns"},
	{Name: "campaign.completed", Group: "Campaigns"},
	{Name: "approval.requested", Group: "Approvals"},
	{Name: "approval.decided", Group: "Approvals"},
	{Name: "provider.down", Group: "Sending providers"},
	{Name: "provider.recovered", Group: "Sending providers"},
	{Name: "automation.published", Group: "Automations"},
	{Name: "message.sent", Group: "Transactional messages"},
	{Name: "message.delivered", Group: "Transactional messages"},
	{Name: "message.deferred", Group: "Transactional messages"},
	{Name: "message.bounced", Group: "Transactional messages"},
	{Name: "message.complained", Group: "Transactional messages"},
	{Name: "message.rejected", Group: "Transactional messages"},
	{Name: "message.failed", Group: "Transactional messages"},
	{Name: "message.opened", Group: "Transactional messages"},
	{Name: "message.clicked", Group: "Transactional messages"},
	{Name: "form.submitted", Group: "Forms"},
	// Grow (design expansion phase 5): referral programs.
	{Name: "referral.counted", Group: "Referrals"},
	{Name: "referral.held", Group: "Referrals"},
	{Name: "referral.not_counted", Group: "Referrals"},
	{Name: "referral.milestone_reached", Group: "Referrals"},
	{Name: "referral.reward_sent", Group: "Referrals"},
	{Name: "referral.reward_failed", Group: "Referrals"},
}

// ValidEvent reports whether name is a known event or wildcard ("email.*", "*").
func ValidEvent(name string) bool {
	if name == "*" {
		return true
	}
	if strings.HasSuffix(name, ".*") {
		g := strings.TrimSuffix(name, ".*")
		for _, e := range Events {
			if strings.HasPrefix(e.Name, g+".") {
				return true
			}
		}
		return false
	}
	return slices.ContainsFunc(Events, func(e WebhookEventInfo) bool { return e.Name == name })
}

type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Jobs  *jobs.Client
	// AllowPrivate reports whether endpoints may resolve to private
	// networks (webhooks.allow_private_networks); read at each use.
	AllowPrivate func() bool
	Now          func() time.Time
	// HTTP overrides the delivery client (tests).
	HTTP *http.Client
	// Managed knows the endpoints integrations manage (internal/integrations):
	// they can't be edited by hand, other API keys don't see them, and one
	// that answers 410 Gone is removed.
	Managed Managed
}

// WebhookManagedBy names the integration that manages an endpoint.
type WebhookManagedBy struct {
	IntegrationID string `json:"integrationId"`
	Kind          string `json:"kind" example:"n8n"`
	Name          string `json:"name" example:"n8n"`
	// Workflow is the workflow that created it ("Unsubscribes → Acme Cloud CRM").
	Workflow string `json:"workflow,omitempty"`
	// Endpoint is its URL masked, as the integration's page shows it: a
	// workflow tool's webhook address often carries a token.
	Endpoint string `json:"endpoint,omitempty" example:"https://n8n.acme.io/webhook/7c1e…/reputecast"`
}

// Managed tells which endpoints integrations manage.
type Managed interface {
	ManagedHooks(ctx context.Context, workspaceID uuid.UUID) (map[uuid.UUID]WebhookManagedBy, error)
	ManagedHook(ctx context.Context, webhookID uuid.UUID) (*WebhookManagedBy, error)
	// Gone removes a managed endpoint that answered 410 Gone.
	Gone(ctx context.Context, webhookID uuid.UUID)
}

// refuseManaged answers 409 for hand edits of an endpoint an integration manages.
func (s *Service) refuseManaged(ctx context.Context, id uuid.UUID) error {
	if s.Managed == nil {
		return nil
	}
	m, err := s.Managed.ManagedHook(ctx, id)
	if err != nil {
		return err
	}
	if m != nil {
		return api.Err(http.StatusConflict, "managed_by_integration", mManaged.Text(i18n.LangFrom(ctx), i18n.Args{"integration": m.Name})).
			WithMeta("integration", m.Kind)
	}
	return nil
}

// hidesManaged reports whether the caller is a key: endpoints an integration
// created are only for people and the integration itself.
func hidesManaged(ctx context.Context) bool {
	p := api.PrincipalFrom(ctx)
	return p != nil && p.Type != authz.ActorUser
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return s.policy().Client(10 * time.Second)
}

// Envelope is the JSON body of every delivery.
type Envelope struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Workspace string    `json:"workspace"`
	Data      any       `json:"data"`
}

// Emit queues an event for every endpoint of the workspace subscribed to it.
// Call it inside the transaction that changes the state the event reports;
// a periodic job picks the deliveries up within seconds. It returns the
// number of deliveries queued.
func Emit(ctx context.Context, q store.DBTX, workspaceID uuid.UUID, workspaceSlug, event string, data any) (int, error) {
	qs := store.New(q)
	hooks, err := qs.HooksMatching(ctx, store.HooksMatchingParams{WorkspaceID: workspaceID, Event: event})
	if err != nil || len(hooks) == 0 {
		return 0, err
	}
	eventID := uuid.Must(uuid.NewV7())
	body, err := json.Marshal(Envelope{ID: eventID.String(), Type: event, CreatedAt: time.Now().UTC(), Workspace: workspaceSlug, Data: data})
	if err != nil {
		return 0, err
	}
	now := time.Now()
	for _, h := range hooks {
		state := "pending"
		if !h.Enabled {
			state = "held"
		}
		if err := qs.HooksInsertDelivery(ctx, store.HooksInsertDeliveryParams{
			ID: uuid.Must(uuid.NewV7()), WebhookID: h.ID, Event: event, EventID: &eventID, Payload: body,
			State: state, NextAttemptAt: &now, MaxAttempts: MaxAttempts,
		}); err != nil {
			return 0, err
		}
	}
	return len(hooks), nil
}

// Sign returns the hex HMAC-SHA256 of "<timestamp>.<body>".
func Sign(secret string, timestamp int64, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(strconv.FormatInt(timestamp, 10)))
	m.Write([]byte("."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// Verify checks a signature and rejects timestamps older than tolerance.
func Verify(secret, signature string, timestamp int64, body []byte, now time.Time, tolerance time.Duration) bool {
	if d := now.Sub(time.Unix(timestamp, 0)); d > tolerance || d < -tolerance {
		return false
	}
	want := Sign(secret, timestamp, body)
	return hmac.Equal([]byte(want), []byte(signature))
}

// NewSecret returns a new signing secret ("whsec_…").
func NewSecret() string { return newSecret() }

// SecretAD is the associated data an endpoint's secret is sealed with.
func SecretAD(id uuid.UUID) string { return secretAD(id) }

func newSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "whsec_" + base64.RawURLEncoding.EncodeToString(b)
}

func secretAD(id uuid.UUID) string { return "webhook:" + id.String() + ":secret" }

// WebhookView is an endpoint as listed. The secret is never returned.
type WebhookView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Enabled     bool     `json:"enabled"`
	// Status is new (no delivery yet), healthy, failing or paused.
	Status          string     `json:"status" enum:"new,healthy,failing,paused"`
	LastStatus      *int32     `json:"lastStatus,omitempty"`
	LastDeliveryAt  *time.Time `json:"lastDeliveryAt,omitempty"`
	FailingSince    *time.Time `json:"failingSince,omitempty"`
	SecretRotatedAt *time.Time `json:"secretRotatedAt,omitempty" doc:"Set when the secret was replaced after creation"`
	CreatedAt       time.Time  `json:"createdAt"`
	Total24h        int64      `json:"total24h" doc:"Deliveries attempted in the last 24 hours"`
	Success24h      *float64   `json:"success24h,omitempty" doc:"Percent delivered in the last 24 hours"`
	Queued          int64      `json:"queued" doc:"Deliveries waiting for an attempt or a retry"`
	// ManagedBy is set for endpoints an integration created: their URL,
	// events and secret change only in the integration.
	ManagedBy *WebhookManagedBy `json:"managedBy,omitempty"`
}

// WebhookList is the outgoing webhooks card.
type WebhookList struct {
	Items  []WebhookView      `json:"items"`
	Events []WebhookEventInfo `json:"events"`
	// Headers name the signature headers for the "Verify a signature" card.
	SignatureHeader string `json:"signatureHeader"`
	TimestampHeader string `json:"timestampHeader"`
	MaxAttempts     int    `json:"maxAttempts"`
}

func toView(h store.HooksListRow) WebhookView {
	v := WebhookView{
		ID: h.ID.String(), Name: h.Name, Description: h.Description, URL: h.Url, Events: h.Events, Enabled: h.Enabled,
		LastStatus: h.LastStatus, LastDeliveryAt: h.LastDeliveryAt, FailingSince: h.FailingSince, CreatedAt: h.CreatedAt,
		Total24h: h.Total24h, Queued: h.Queued,
	}
	if v.Events == nil {
		v.Events = []string{}
	}
	if h.SecretRotatedAt != nil && h.SecretRotatedAt.Sub(h.CreatedAt) > time.Minute {
		v.SecretRotatedAt = h.SecretRotatedAt
	}
	if h.Total24h > 0 {
		pct := float64(h.Ok24h) / float64(h.Total24h) * 100
		pct = float64(int64(pct*10+0.5)) / 10
		v.Success24h = &pct
	}
	switch {
	case !h.Enabled:
		v.Status = "paused"
	case h.FailingSince != nil:
		v.Status = "failing"
	case h.LastDeliveryAt == nil:
		v.Status = "new"
	default:
		v.Status = "healthy"
	}
	return v
}

// List returns the workspace's endpoints with 24-hour delivery stats.
func (s *Service) List(ctx context.Context, ws *store.Workspace) (WebhookList, error) {
	rows, err := s.Q.HooksList(ctx, ws.ID)
	if err != nil {
		return WebhookList{}, err
	}
	out := WebhookList{Items: []WebhookView{}, Events: eventsIn(i18n.LangFrom(ctx)), SignatureHeader: HeaderSignature, TimestampHeader: HeaderTimestamp, MaxAttempts: MaxAttempts}
	managed, err := s.managed(ctx, ws.ID)
	if err != nil {
		return WebhookList{}, err
	}
	for _, r := range rows {
		v := toView(r)
		if m, ok := managed[r.ID]; ok {
			if hidesManaged(ctx) {
				continue
			}
			v.ManagedBy = &m
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

func (s *Service) managed(ctx context.Context, wsID uuid.UUID) (map[uuid.UUID]WebhookManagedBy, error) {
	if s.Managed == nil {
		return map[uuid.UUID]WebhookManagedBy{}, nil
	}
	return s.Managed.ManagedHooks(ctx, wsID)
}

// WebhookInput creates or updates an endpoint.
type WebhookInput struct {
	Name        string   `json:"name" maxLength:"80"`
	Description string   `json:"description,omitempty" maxLength:"120"`
	URL         string   `json:"url" maxLength:"500"`
	Events      []string `json:"events" minItems:"1" maxItems:"40"`
}

// WebhookCreated carries the signing secret, shown once.
type WebhookCreated struct {
	Webhook WebhookView `json:"webhook"`
	Secret  string      `json:"secret" doc:"The signing secret. Shown only in this response."`
}

func (s *Service) validate(l i18n.Lang, in WebhookInput) (string, []string, error) {
	u, err := s.policy().ValidateURL(in.URL)
	if err != nil {
		msg := err.Error()
		return "", nil, api.Invalid("body.url", strings.ToUpper(msg[:1])+msg[1:]+".")
	}
	var events []string
	for _, e := range in.Events {
		e = strings.TrimSpace(e)
		if !ValidEvent(e) {
			return "", nil, api.Invalid("body.events", mUnknownEvent.Text(l, i18n.Args{"event": strconv.Quote(e)}))
		}
		if !slices.Contains(events, e) {
			events = append(events, e)
		}
	}
	return u.String(), events, nil
}

// Create registers an endpoint and returns its signing secret once.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebhookInput) (WebhookCreated, error) {
	u, events, err := s.validate(i18n.LangFrom(ctx), in)
	if err != nil {
		return WebhookCreated{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		pu, _ := url.Parse(u)
		name = pu.Host
	}
	id := uuid.Must(uuid.NewV7())
	secret := newSecret()
	enc, err := s.Box.SealString(secret, secretAD(id))
	if err != nil {
		return WebhookCreated{}, err
	}
	var out WebhookCreated
	creator := p.UserID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		h, err := q.HooksCreate(ctx, store.HooksCreateParams{ID: id, WorkspaceID: ws.ID, Name: name, Description: strings.TrimSpace(in.Description), Url: u, Events: events, SecretEnc: enc, CreatedBy: &creator})
		if err != nil {
			return err
		}
		out = WebhookCreated{Webhook: toView(store.HooksListRow{ID: h.ID, WorkspaceID: h.WorkspaceID, Name: h.Name, Description: h.Description, Url: h.Url, Events: h.Events, Enabled: h.Enabled, CreatedAt: h.CreatedAt}), Secret: secret}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.created", TargetType: "webhook", TargetID: id.String(), TargetName: u,
			After: map[string]any{"url": u, "events": events, "name": name}, Metadata: map[string]any{"summary": strings.Join(events, ", "), "permission": string(authz.WebhooksManage)}})
	})
	return out, err
}

// Update changes an endpoint's URL, name or events.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, id uuid.UUID, in WebhookInput) (WebhookView, error) {
	old, err := s.Q.HooksGet(ctx, store.HooksGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return WebhookView{}, api.MapError(err, "Webhook")
	}
	if err := s.refuseManaged(ctx, id); err != nil {
		return WebhookView{}, err
	}
	u, events, err := s.validate(i18n.LangFrom(ctx), in)
	if err != nil {
		return WebhookView{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = old.Name
	}
	var out WebhookView
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		h, err := q.HooksUpdate(ctx, store.HooksUpdateParams{ID: id, WorkspaceID: ws.ID, Name: name, Description: strings.TrimSpace(in.Description), Url: u, Events: events})
		if err != nil {
			return err
		}
		out = toView(store.HooksListRow{ID: h.ID, WorkspaceID: h.WorkspaceID, Name: h.Name, Description: h.Description, Url: h.Url, Events: h.Events, Enabled: h.Enabled, CreatedAt: h.CreatedAt,
			LastStatus: h.LastStatus, LastDeliveryAt: h.LastDeliveryAt, FailingSince: h.FailingSince, SecretRotatedAt: h.SecretRotatedAt})
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.updated", TargetType: "webhook", TargetID: id.String(), TargetName: u,
			Before: map[string]any{"url": old.Url, "events": old.Events, "name": old.Name}, After: map[string]any{"url": u, "events": events, "name": name},
			Metadata: map[string]any{"permission": string(authz.WebhooksManage)}})
	})
	return out, err
}

// Delete removes an endpoint and its delivery log.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	old, err := s.Q.HooksGet(ctx, store.HooksGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Webhook")
	}
	if err := s.refuseManaged(ctx, id); err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.HooksDelete(ctx, store.HooksDeleteParams{ID: id, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.deleted", TargetType: "webhook", TargetID: id.String(), TargetName: old.Url,
			Before: map[string]any{"url": old.Url, "events": old.Events}, Metadata: map[string]any{"permission": string(authz.WebhooksManage)}})
	})
}

// WebhookSecretResult carries a new signing secret, shown once.
type WebhookSecretResult struct {
	Secret string `json:"secret"`
}

// RotateSecret replaces the signing secret. Deliveries sign with the new one at once.
func (s *Service) RotateSecret(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WebhookSecretResult, error) {
	old, err := s.Q.HooksGet(ctx, store.HooksGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return WebhookSecretResult{}, api.MapError(err, "Webhook")
	}
	if err := s.refuseManaged(ctx, id); err != nil {
		return WebhookSecretResult{}, err
	}
	secret := newSecret()
	enc, err := s.Box.SealString(secret, secretAD(id))
	if err != nil {
		return WebhookSecretResult{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.HooksSetSecret(ctx, store.HooksSetSecretParams{ID: id, WorkspaceID: ws.ID, SecretEnc: enc}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.secret_rotated", TargetType: "webhook", TargetID: id.String(), TargetName: old.Url,
			Metadata: map[string]any{"permission": string(authz.WebhooksManage)}})
	})
	return WebhookSecretResult{Secret: secret}, err
}

// SetEnabled pauses or resumes an endpoint. Events keep queuing while it is
// paused and are delivered in order when it resumes.
func (s *Service) SetEnabled(ctx context.Context, ws *store.Workspace, id uuid.UUID, enabled bool) error {
	old, err := s.Q.HooksGet(ctx, store.HooksGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Webhook")
	}
	if err := s.refuseManaged(ctx, id); err != nil {
		return err
	}
	if old.Enabled == enabled {
		return nil
	}
	var released []uuid.UUID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.HooksSetEnabled(ctx, store.HooksSetEnabledParams{ID: id, WorkspaceID: ws.ID, Enabled: enabled}); err != nil {
			return err
		}
		if enabled {
			if released, err = q.HooksReleaseHeld(ctx, id); err != nil {
				return err
			}
		}
		verb := "paused"
		if enabled {
			verb = "resumed"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook." + verb, TargetType: "webhook", TargetID: id.String(), TargetName: old.Url,
			Before: map[string]any{"enabled": old.Enabled}, After: map[string]any{"enabled": enabled}, Metadata: map[string]any{"permission": string(authz.WebhooksManage)}})
	})
	if err == nil && len(released) > 0 {
		s.kick(ctx, released)
	}
	return err
}

// WebhookDeliveryView is one row of the delivery log.
type WebhookDeliveryView struct {
	ID        string `json:"id"`
	WebhookID string `json:"webhookId"`
	Endpoint  string `json:"endpoint" doc:"Host and path, e.g. api.acme.io/hooks/reputecast"`
	Event     string `json:"event"`
	// Subtype qualifies the event ("hard" for email.bounced).
	Subtype       string     `json:"subtype,omitempty"`
	State         string     `json:"state" enum:"pending,delivering,delivered,retrying,dead,held"`
	StatusCode    *int32     `json:"statusCode,omitempty"`
	StatusText    string     `json:"statusText,omitempty" doc:"Bad Gateway"`
	DurationMs    *int32     `json:"durationMs,omitempty"`
	Error         string     `json:"error,omitempty"`
	Attempts      int32      `json:"attempts"`
	MaxAttempts   int32      `json:"maxAttempts"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastAttemptAt *time.Time `json:"lastAttemptAt,omitempty"`
	NextAttemptAt *time.Time `json:"nextAttemptAt,omitempty"`
	// BackoffMinutes is the wait before the next attempt.
	BackoffMinutes *int `json:"backoffMinutes,omitempty"`
}

// WebhookDeliveryList is the delivery log.
type WebhookDeliveryList struct {
	Items []WebhookDeliveryView `json:"items"`
	// Queued counts deliveries waiting per endpoint id.
	Queued map[string]int64 `json:"queued"`
}

func endpointLabel(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	// A label, not a link: "…" in a masked address reads as typed.
	return strings.TrimSuffix(u.Host+u.Path, "/")
}

func deliveryView(id, webhookID uuid.UUID, hookURL, event, subtype, state string, code, dur *int32, errText string, attempts, maxAttempts int32, created time.Time, last, next *time.Time) WebhookDeliveryView {
	v := WebhookDeliveryView{ID: id.String(), WebhookID: webhookID.String(), Endpoint: endpointLabel(hookURL), Event: event, Subtype: subtype, State: state,
		StatusCode: code, DurationMs: dur, Error: errText, Attempts: attempts, MaxAttempts: maxAttempts, CreatedAt: created, LastAttemptAt: last}
	if code != nil {
		v.StatusText = http.StatusText(int(*code))
	}
	if state == "retrying" || state == "pending" {
		v.NextAttemptAt = next
		if next != nil && last != nil {
			m := int(next.Sub(*last).Round(time.Minute) / time.Minute)
			v.BackoffMinutes = &m
		}
	}
	return v
}

// Deliveries returns the latest deliveries, newest first.
func (s *Service) Deliveries(ctx context.Context, ws *store.Workspace, webhookID *uuid.UUID, state string, limit int) (WebhookDeliveryList, error) {
	var st *string
	if state != "" {
		st = &state
	}
	rows, err := s.Q.HooksDeliveries(ctx, store.HooksDeliveriesParams{WorkspaceID: ws.ID, WebhookID: webhookID, State: st, Lim: int32(limit)})
	if err != nil {
		return WebhookDeliveryList{}, err
	}
	out := WebhookDeliveryList{Items: []WebhookDeliveryView{}, Queued: map[string]int64{}}
	// Keys don't see the endpoints integrations manage; people see their
	// addresses masked, as on the integration's page.
	managed, err := s.managed(ctx, ws.ID)
	if err != nil {
		return WebhookDeliveryList{}, err
	}
	hide := hidesManaged(ctx)
	for _, r := range rows {
		url := r.WebhookUrl
		if m, ok := managed[r.WebhookID]; ok {
			if hide {
				continue
			}
			if m.Endpoint != "" {
				url = m.Endpoint
			}
		}
		out.Items = append(out.Items, deliveryView(r.ID, r.WebhookID, url, r.Event, r.Subtype, r.State, r.StatusCode, r.DurationMs, r.Error, r.Attempt, r.MaxAttempts, r.CreatedAt, r.LastAttemptAt, r.NextAttemptAt))
	}
	hooks, err := s.Q.HooksList(ctx, ws.ID)
	if err != nil {
		return WebhookDeliveryList{}, err
	}
	for _, h := range hooks {
		if _, ok := managed[h.ID]; ok && hide {
			continue
		}
		out.Queued[h.ID.String()] = h.Queued
	}
	return out, nil
}

// WebhookDeliveryDetail adds the payload and the endpoint's response.
type WebhookDeliveryDetail struct {
	Delivery WebhookDeliveryView `json:"delivery"`
	Payload  json.RawMessage     `json:"payload"`
	Response string              `json:"response" doc:"First kilobyte of the endpoint's last response"`
}

// Delivery returns one delivery with its payload.
func (s *Service) Delivery(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WebhookDeliveryDetail, error) {
	d, err := s.Q.HooksGetDelivery(ctx, store.HooksGetDeliveryParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return WebhookDeliveryDetail{}, api.MapError(err, "Delivery")
	}
	var sub struct {
		Data struct {
			Subtype string `json:"subtype"`
		} `json:"data"`
	}
	_ = json.Unmarshal(d.Payload, &sub)
	url := d.WebhookUrl
	if s.Managed != nil {
		if m, err := s.Managed.ManagedHook(ctx, d.WebhookID); err == nil && m != nil {
			if hidesManaged(ctx) {
				return WebhookDeliveryDetail{}, api.NotFound("Delivery")
			}
			if m.Endpoint != "" {
				url = m.Endpoint
			}
		}
	}
	return WebhookDeliveryDetail{
		Delivery: deliveryView(d.ID, d.WebhookID, url, d.Event, sub.Data.Subtype, d.State, d.StatusCode, d.DurationMs, d.Error, d.Attempt, d.MaxAttempts, d.CreatedAt, d.LastAttemptAt, d.NextAttemptAt),
		Payload:  d.Payload, Response: d.Response,
	}, nil
}

// WebhookRetryResult counts the deliveries queued again.
type WebhookRetryResult struct {
	Queued int `json:"queued"`
}

// Retry queues failed deliveries for an immediate attempt: one delivery,
// every failed one of an endpoint, or every failed one of the workspace.
// Dead-lettered deliveries start a fresh round of attempts (replay).
func (s *Service) Retry(ctx context.Context, ws *store.Workspace, deliveryID, webhookID *uuid.UUID) (WebhookRetryResult, error) {
	var ids []uuid.UUID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		ids, err = q.HooksRequeue(ctx, store.HooksRequeueParams{WorkspaceID: ws.ID, ID: deliveryID, WebhookID: webhookID})
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			if deliveryID != nil {
				return api.Conflict(mOnlyFailed.Text(i18n.LangFrom(ctx), nil))
			}
			return nil
		}
		target, name := "workspace", ws.Name
		switch {
		case deliveryID != nil:
			target, name = "webhook_delivery", deliveryID.String()
		case webhookID != nil:
			target, name = "webhook", webhookID.String()
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.retried", TargetType: target, TargetID: name, TargetName: name,
			Metadata: map[string]any{"summary": fmt.Sprintf("%d %s queued again", len(ids), plural(len(ids), "delivery", "deliveries")), "permission": string(authz.WebhooksManage)}})
	})
	if err != nil {
		return WebhookRetryResult{}, err
	}
	s.kick(ctx, ids)
	return WebhookRetryResult{Queued: len(ids)}, nil
}

// WebhookTestResult is the outcome of a test delivery.
type WebhookTestResult struct {
	Delivery WebhookDeliveryView `json:"delivery"`
}

// Test sends a webhook.test event right away and returns the result.
func (s *Service) Test(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WebhookTestResult, error) {
	h, err := s.Q.HooksGet(ctx, store.HooksGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return WebhookTestResult{}, api.MapError(err, "Webhook")
	}
	eventID := uuid.Must(uuid.NewV7())
	body, _ := json.Marshal(Envelope{ID: eventID.String(), Type: "webhook.test", CreatedAt: s.now().UTC(), Workspace: ws.Slug, Data: map[string]any{
		"message": "Test event from ReputeCast. Verify the signature, then answer 2xx.",
	}})
	deliveryID := uuid.Must(uuid.NewV7())
	now := s.now()
	if err := s.Q.HooksInsertDelivery(ctx, store.HooksInsertDeliveryParams{ID: deliveryID, WebhookID: h.ID, Event: "webhook.test", EventID: &eventID, Payload: body, State: "pending", NextAttemptAt: &now, MaxAttempts: 1}); err != nil {
		return WebhookTestResult{}, err
	}
	if _, err := s.Attempt(ctx, deliveryID); err != nil {
		return WebhookTestResult{}, err
	}
	d, err := s.Delivery(ctx, ws, deliveryID)
	if err != nil {
		return WebhookTestResult{}, err
	}
	_ = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.tested", TargetType: "webhook", TargetID: h.ID.String(), TargetName: h.Url,
		Result: map[bool]string{true: audit.Allowed, false: audit.Failed}[d.Delivery.State == "delivered"], Metadata: map[string]any{"status": d.Delivery.StatusCode}})
	return WebhookTestResult{Delivery: d.Delivery}, nil
}

// Attempt makes one delivery attempt and records the result. It returns
// the time of the next attempt when the delivery will be retried.
func (s *Service) Attempt(ctx context.Context, deliveryID uuid.UUID) (*time.Time, error) {
	d, err := s.Q.HooksClaimDelivery(ctx, deliveryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // delivered, dead or taken by another worker
		}
		return nil, err
	}
	if d.Demo {
		// Demo endpoints point at fictional addresses: nothing is sent, and
		// the endpoint's health is left as the demo shows it.
		return nil, s.Q.HooksFinishDelivery(ctx, store.HooksFinishDeliveryParams{ID: d.ID, State: "dead",
			Error: "Demo endpoint: ReputeCast never sends demo data. Point it at your own URL to receive events."})
	}
	secret, err := s.Box.OpenString(d.SecretEnc, secretAD(d.WebhookID))
	if err != nil {
		return nil, err
	}
	ts := s.now().Unix()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.Url, strings.NewReader(string(d.Payload)))
	if err != nil {
		return s.finish(ctx, d, nil, 0, "", "Invalid URL")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ReputeCast-Webhooks/"+version.Get().Version)
	req.Header.Set(HeaderSignature, Sign(secret, ts, d.Payload))
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(HeaderEvent, d.Event)
	req.Header.Set(HeaderDelivery, d.ID.String())
	start := time.Now()
	resp, err := s.client().Do(req)
	dur := int32(time.Since(start).Milliseconds())
	if err != nil {
		return s.finish(ctx, d, nil, dur, "", netguard.Describe(ctx, err))
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	code := int32(resp.StatusCode)
	return s.finish(ctx, d, &code, dur, string(b), "")
}

func (s *Service) finish(ctx context.Context, d store.HooksClaimDeliveryRow, code *int32, dur int32, body, errText string) (*time.Time, error) {
	ok := code != nil && *code >= 200 && *code < 300
	attempt := d.Attempt + 1
	state := "delivered"
	var next *time.Time
	switch {
	case ok:
	case attempt >= d.MaxAttempts:
		state = "dead"
	default:
		state = "retrying"
		wait := Backoff[min(int(attempt)-1, len(Backoff)-1)]
		t := s.now().Add(wait)
		next = &t
	}
	if code != nil && !ok && errText == "" {
		errText = fmt.Sprintf("HTTP %d %s", *code, http.StatusText(int(*code)))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.HooksFinishDelivery(ctx, store.HooksFinishDeliveryParams{ID: d.ID, State: state, StatusCode: code, DurationMs: &dur, Response: body, Error: errText, NextAttemptAt: next}); err != nil {
			return err
		}
		var status *int32
		if code != nil {
			status = code
		} else {
			zero := int32(0)
			status = &zero
		}
		return q.HooksRecordResult(ctx, store.HooksRecordResultParams{ID: d.WebhookID, LastStatus: status, Ok: ok})
	})
	if err == nil && code != nil && *code == http.StatusGone && s.Managed != nil {
		// Zapier's REST Hooks: a 410 means the workflow is gone. Managed
		// endpoints are removed; others keep retrying as usual.
		s.Managed.Gone(ctx, d.WebhookID)
		return nil, nil
	}
	return next, err
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (s *Service) policy() netguard.Policy {
	return netguard.Policy{AllowPrivate: s.AllowPrivate != nil && s.AllowPrivate()}
}
