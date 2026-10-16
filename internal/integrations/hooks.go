package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// IntegrationHookInput subscribes a workflow to a trigger (Zapier's REST
// Hooks, Make's instant triggers, n8n's webhook node).
type IntegrationHookInput struct {
	Event string `json:"event" example:"contact.unsubscribed" doc:"The trigger's event"`
	URL   string `json:"url" maxLength:"500" doc:"Where ReputeCast posts the events: the workflow's webhook address"`
}

// IntegrationHook is a managed webhook as its integration sees it. The
// signing secret goes to the integration once and is never shown again.
type IntegrationHook struct {
	ID        string    `json:"id"`
	Event     string    `json:"event"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	Secret    string    `json:"secret,omitempty" doc:"The signing secret. Only in the answer to the subscribe call."`
	// SignatureHeader and TimestampHeader name the headers to verify.
	SignatureHeader string `json:"signatureHeader"`
	TimestampHeader string `json:"timestampHeader"`
}

// triggerOn returns the integration's trigger for an event, if it is on.
func triggerOn(ik *api.IntegrationKey, items []string, event string) (Item, bool) {
	for _, it := range Items(ik.Kind) {
		if it.Kind == ItemTrigger && it.Event == event {
			for _, k := range items {
				if k == it.Key {
					return it, true
				}
			}
			return it, false
		}
	}
	return Item{}, false
}

func (s *Service) callingIntegration(ctx context.Context) (*api.IntegrationKey, store.IntegrationKeyLimitRow, error) {
	ik := api.IntegrationFrom(ctx)
	if ik == nil {
		return nil, store.IntegrationKeyLimitRow{}, api.Err(http.StatusForbidden, api.CodePermissionDenied,
			"Only an integration's key subscribes to triggers. For your own endpoints, use Settings › Webhooks or the webhooks API.")
	}
	row, err := s.Q.IntegrationKeyLimit(ctx, ik.ID)
	return ik, row, err
}

// Subscribe creates the managed webhook of a workflow's trigger.
func (s *Service) Subscribe(ctx context.Context, ws *store.Workspace, p *authz.Principal, in IntegrationHookInput, workflow string) (IntegrationHook, error) {
	ik, row, err := s.callingIntegration(ctx)
	if err != nil {
		return IntegrationHook{}, err
	}
	event := strings.TrimSpace(in.Event)
	it, on := triggerOn(ik, row.Items, event)
	if it.Key == "" {
		return IntegrationHook{}, api.Invalid("body.event", fmt.Sprintf("%s has no trigger %q.", ik.Name, event))
	}
	if !on {
		return IntegrationHook{}, api.Err(http.StatusForbidden, api.CodeNotForThisKey, fmt.Sprintf(
			"%s can't use the %s trigger: it was turned off. Change what %s can do in Settings › Integrations.", ik.Name, itemLabel(it.Key), ik.Name))
	}
	for _, need := range it.Permissions {
		if !p.Can(need) {
			return IntegrationHook{}, api.Forbidden(need, access.Helpers(ctx, s.Q, ws.ID, need))
		}
	}
	u, err := s.policy().ValidateURL(in.URL)
	if err != nil {
		return IntegrationHook{}, s.privateRefusal(ctx, in.URL, err)
	}
	if addr, private := privateHost(ctx, u.Hostname()); private && !s.allowPrivate() {
		return IntegrationHook{}, s.privateRefusal(ctx, in.URL, fmt.Errorf("%s resolves to %s, a private address", u.Hostname(), addr))
	}
	wf := ParseWorkflow(workflow)
	noteUse(ctx, WorkflowUse{Item: it.Key})
	noteTargetJSON(ctx, CallTarget{Kind: "subscribe", Event: event, Endpoint: maskEndpoint(u.String())})
	id := uuid.Must(uuid.NewV7())
	secret := webhooks.NewSecret()
	enc, err := s.Box.SealString(secret, webhooks.SecretAD(id))
	if err != nil {
		return IntegrationHook{}, err
	}
	name := ik.Name + " · " + firstNonEmpty(wf.Name, itemLabel(it.Key))
	owner := p.UserID
	keyID := p.ID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		h, err := q.HooksCreate(ctx, store.HooksCreateParams{ID: id, WorkspaceID: ws.ID, Name: clip(name, 80), Description: "", Url: u.String(), Events: []string{event},
			SecretEnc: enc, CreatedBy: &owner})
		if err != nil {
			return err
		}
		if err := q.IntegrationWebhookInsert(ctx, store.IntegrationWebhookInsertParams{WebhookID: h.ID, IntegrationID: ik.ID, ApiKeyID: &keyID, Event: event,
			WorkflowKey: wf.Key(), WorkflowName: wf.Name, CreatedAt: h.CreatedAt}); err != nil {
			return err
		}
		summary := fmt.Sprintf("managed by %s · %s", ik.Name, event)
		if wf.Name != "" {
			summary += " · workflow " + wf.Name
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.created", TargetType: "webhook", TargetID: id.String(), TargetName: u.String(),
			After:    map[string]any{"url": u.String(), "events": []string{event}, "managed_by": ik.Kind, "workflow": wf.Name},
			Metadata: s.keyMeta(ctx, map[string]any{"summary": summary, "permission": string(authz.WebhooksManage)})})
	})
	if err != nil {
		return IntegrationHook{}, err
	}
	return IntegrationHook{ID: id.String(), Event: event, URL: u.String(), CreatedAt: s.now(), Secret: secret,
		SignatureHeader: webhooks.HeaderSignature, TimestampHeader: webhooks.HeaderTimestamp}, nil
}

// privateRefusal explains a webhook address the SSRF guard refuses.
func (s *Service) privateRefusal(ctx context.Context, raw string, err error) error {
	msg := err.Error()
	if strings.Contains(msg, "private") || strings.Contains(msg, "loopback") || strings.Contains(msg, "local hostname") {
		admins := s.Reachability(ctx).Admins
		ask := ""
		if len(admins) > 0 {
			ask = " Ask " + access.OrList(admins) + "."
		}
		pr := api.Invalid("body.url", fmt.Sprintf("%s. Webhooks to private addresses stay blocked until an instance admin allows them (webhooks.allow_private_networks).%s",
			capitalize(strings.TrimPrefix(msg, "address not allowed: ")), ask))
		pr.Code = "private_address"
		return pr
	}
	return api.Invalid("body.url", capitalize(msg)+".")
}

// Unsubscribe removes a workflow's managed webhook (the workflow was turned off).
func (s *Service) Unsubscribe(ctx context.Context, ws *store.Workspace, hookID uuid.UUID) error {
	ik, _, err := s.callingIntegration(ctx)
	if err != nil {
		return err
	}
	h, err := s.Q.IntegrationWebhookOf(ctx, store.IntegrationWebhookOfParams{WebhookID: hookID, IntegrationID: ik.ID})
	if err != nil {
		return api.MapError(err, "Webhook")
	}
	api.NoteCall(ctx, noteItem, h.Event)
	noteTargetJSON(ctx, CallTarget{Kind: "unsubscribe", Event: h.Event, Endpoint: maskEndpoint(h.Url)})
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.IntegrationWebhookDelete(ctx, hookID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.deleted", TargetType: "webhook", TargetID: hookID.String(), TargetName: h.Url,
			Before:   map[string]any{"url": h.Url, "events": []string{h.Event}},
			Metadata: s.keyMeta(ctx, map[string]any{"summary": fmt.Sprintf("managed by %s · the workflow was turned off", ik.Name), "permission": string(authz.WebhooksManage)})})
	})
}

// Hook returns one of the calling integration's managed webhooks, without
// its secret: n8n checks that a workflow's webhook still exists whenever the
// workflow is turned on (or n8n restarts), and subscribes again when it
// doesn't, for example after a 410 Gone removed it.
func (s *Service) Hook(ctx context.Context, hookID uuid.UUID) (IntegrationHook, error) {
	ik, _, err := s.callingIntegration(ctx)
	if err != nil {
		return IntegrationHook{}, err
	}
	h, err := s.Q.IntegrationWebhookOf(ctx, store.IntegrationWebhookOfParams{WebhookID: hookID, IntegrationID: ik.ID})
	if err != nil {
		return IntegrationHook{}, api.MapError(err, "Webhook")
	}
	api.NoteCall(ctx, noteItem, h.Event)
	noteTargetJSON(ctx, CallTarget{Kind: "hook", Event: h.Event, Endpoint: maskEndpoint(h.Url)})
	return IntegrationHook{ID: h.WebhookID.String(), Event: h.Event, URL: h.Url, CreatedAt: h.CreatedAt,
		SignatureHeader: webhooks.HeaderSignature, TimestampHeader: webhooks.HeaderTimestamp}, nil
}

// IntegrationFormItem is a signup form, for the Form submitted trigger's
// form filter in the apps.
type IntegrationFormItem struct {
	ID     string `json:"id" example:"frm_8c2e1a" doc:"The form's ID, as form.submitted's form_id"`
	Name   string `json:"name"`
	Status string `json:"status" enum:"draft,live,paused"`
}

// IntegrationFormItems are the workspace's signup forms.
type IntegrationFormItems struct {
	Items []IntegrationFormItem `json:"items"`
}

// TriggerForms lists the signup forms, so a workflow can start from one
// form's submissions only. Only keys with the Form submitted trigger on can
// call it; it reads forms, never contacts.
func (s *Service) TriggerForms(ctx context.Context, ws *store.Workspace) (IntegrationFormItems, error) {
	rows, err := s.Q.WordPressForms(ctx, ws.ID)
	if err != nil {
		return IntegrationFormItems{}, err
	}
	noteTargetJSON(ctx, CallTarget{Kind: "forms"})
	out := IntegrationFormItems{Items: []IntegrationFormItem{}}
	for _, f := range rows {
		out.Items = append(out.Items, IntegrationFormItem{ID: f.PublicID, Name: f.Name, Status: f.Status})
	}
	return out, nil
}

// Gone removes a managed webhook whose endpoint answered 410 Gone, as
// Zapier's REST Hooks ask. The system is the actor.
func (s *Service) Gone(ctx context.Context, webhookID uuid.UUID) {
	h, err := s.Q.IntegrationManagedHook(ctx, webhookID)
	if err != nil {
		return
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.IntegrationWebhookDelete(ctx, webhookID)
		if err != nil || n == 0 {
			return err
		}
		ws := h.WorkspaceID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws, Action: "webhook.deleted", TargetType: "webhook", TargetID: webhookID.String(), TargetName: h.Url,
			Actor:    &audit.Actor{Type: authz.ActorSystem, Name: "System"},
			Before:   map[string]any{"url": h.Url, "events": []string{h.Event}},
			Metadata: map[string]any{"summary": fmt.Sprintf("%s answered 410 Gone: removed, as REST hooks ask · managed by %s", hostOf(h.Url), Name(h.Kind))}})
	})
	if err != nil {
		s.log().WarnContext(ctx, "remove a gone webhook", "webhook", webhookID, "err", err)
	}
}

// ManagedHooks implements webhooks.Managed: the endpoints integrations
// manage in a workspace.
func (s *Service) ManagedHooks(ctx context.Context, workspaceID uuid.UUID) (map[uuid.UUID]webhooks.WebhookManagedBy, error) {
	rows, err := s.Q.IntegrationManagedHooks(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]webhooks.WebhookManagedBy, len(rows))
	for _, r := range rows {
		out[r.WebhookID] = webhooks.WebhookManagedBy{IntegrationID: r.IntegrationID.String(), Kind: r.Kind, Name: Name(r.Kind), Workflow: r.WorkflowName, Endpoint: maskEndpoint(r.Url)}
	}
	return out, nil
}

// ManagedHook implements webhooks.Managed for one endpoint.
func (s *Service) ManagedHook(ctx context.Context, webhookID uuid.UUID) (*webhooks.WebhookManagedBy, error) {
	r, err := s.Q.IntegrationManagedHook(ctx, webhookID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &webhooks.WebhookManagedBy{IntegrationID: r.IntegrationID.String(), Kind: r.Kind, Name: Name(r.Kind), Workflow: r.WorkflowName, Endpoint: maskEndpoint(r.Url)}, nil
}

// IntegrationTriggerSample is a sample delivery of a trigger, for the apps'
// field mapping (Zapier's performList, Make's sample, n8n's test step).
type IntegrationTriggerSample struct {
	Event   string            `json:"event"`
	Payload webhooks.Envelope `json:"payload"`
}

// Sample returns a sample payload of a trigger that is on.
func (s *Service) Sample(ctx context.Context, ws *store.Workspace, p *authz.Principal, event string) (IntegrationTriggerSample, error) {
	ik, row, err := s.callingIntegration(ctx)
	if err != nil {
		return IntegrationTriggerSample{}, err
	}
	it, on := triggerOn(ik, row.Items, event)
	if it.Key == "" {
		return IntegrationTriggerSample{}, api.NotFound("Trigger")
	}
	if !on {
		return IntegrationTriggerSample{}, api.Err(http.StatusForbidden, api.CodeNotForThisKey, fmt.Sprintf("%s's %s trigger is turned off.", ik.Name, itemLabel(it.Key)))
	}
	for _, need := range it.Permissions {
		if !p.Can(need) {
			return IntegrationTriggerSample{}, api.Forbidden(need, access.Helpers(ctx, s.Q, ws.ID, need))
		}
	}
	api.NoteCall(ctx, noteItem, it.Key)
	noteTargetJSON(ctx, CallTarget{Kind: "sample", Event: event})
	return IntegrationTriggerSample{Event: event, Payload: webhooks.Envelope{ID: "evt_sample", Type: event, CreatedAt: s.now().UTC().Truncate(time.Second),
		Workspace: ws.Slug, Data: SampleData(event)}}, nil
}

// SampleData is the data of a sample event, shaped like the real ones.
func SampleData(event string) map[string]any {
	contact := map[string]any{
		"contact_id": "0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01", "email": "lucia.fernandez@example.com", "first_name": "Lucía", "last_name": "Fernández",
		"company": "", "country": "MX", "language": "es-MX", "tags": []string{"beta-applicant"}, "fields": map[string]any{"plan": "Team"},
		"status": "active", "lists": []map[string]any{{"id": "0192a4c1-0000-7000-8000-00000000b3a1", "name": "Weekly Digest", "status": "subscribed"}},
	}
	switch event {
	case "contact.subscribed", "contact.unsubscribed":
		contact["list_id"] = "0192a4c1-0000-7000-8000-00000000b3a1"
		contact["list_name"] = "Weekly Digest"
		contact["source"] = "form"
		return contact
	case "contact.updated":
		contact["changed"] = []string{"language", "tags"}
		return contact
	case "form.submitted":
		return map[string]any{"form_id": "frm_8c2e1a", "form_name": "Newsletter signup", "contact_id": contact["contact_id"], "email": contact["email"],
			"answers": map[string]any{"email": contact["email"], "first_name": "Lucía"}, "lists": []string{"Weekly Digest"},
			"consent_text": "I agree to receive emails from Acme. Unsubscribe any time with one click.", "language": "es", "double_opt_in": true,
			"submitted_at": "2026-10-03T14:02:11Z", "confirmed_at": "2026-10-03T14:05:40Z"}
	case "campaign.completed":
		return map[string]any{"campaign_id": "0192a4c1-0000-7000-8000-0000000c0142", "name": "Weekly Digest #142",
			"subject": "Three launches, one deprecation, and the roadmap for Q4", "status": "sent", "sent": 31604, "delivered": 31352,
			"started_at": "2026-10-14T13:00:00Z", "completed_at": "2026-10-14T13:41:00Z"}
	case "message.bounced":
		return map[string]any{"message_id": "msg_01J9ZK7Q2X8M", "template": "password-reset", "to": "sam.rivera@outlook.com", "subtype": "hard",
			"reason": "550 5.1.1 The email account that you tried to reach does not exist", "provider": "postmark", "tags": []string{"auth"},
			"metadata": map[string]any{"user_id": "u_8812"}}
	}
	return map[string]any{}
}

// maskEndpoint shortens a webhook address for lists: the host and path,
// with long path tokens cut ("n8n.acme.io/webhook/7c1e…/reputecast").
func maskEndpoint(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	for i, p := range parts {
		if len(p) > 12 {
			parts[i] = p[:4] + "…"
		}
	}
	path := strings.Join(parts, "/")
	if path != "" {
		path = "/" + path
	}
	return u.Scheme + "://" + u.Host + path
}

func hostOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Host
	}
	return raw
}

// jsonOf marshals v, ignoring errors (for small fixed shapes).
func jsonOf(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
