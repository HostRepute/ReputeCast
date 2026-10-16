package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// CallRetention is how long the call log keeps calls.
const CallRetention = 7 * 24 * time.Hour

// WorkflowRetention is how long a workflow stays listed after its last call.
const WorkflowRetention = 30 * 24 * time.Hour

// Notes an operation can leave on its call (api.NoteCall).
const (
	noteItem    = "item"
	noteTarget  = "target"
	noteContact = "contact"
	noteUses    = "uses"
)

// CallTarget says what a call was about. Kind picks the sentence the app
// shows ("payment-failed to finance@initech.example"); Email is replaced by
// "a contact" for members without subscribers:view.
type CallTarget struct {
	Kind     string `json:"kind"`
	Email    string `json:"email,omitempty"`
	List     string `json:"list,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Event    string `json:"event,omitempty"`
	Template string `json:"template,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Title    string `json:"title,omitempty"`
	Version  string `json:"version,omitempty"`
}

func noteTargetJSON(ctx context.Context, t CallTarget) {
	b, _ := json.Marshal(t)
	api.NoteCall(ctx, noteTarget, string(b))
}

// keyMeta adds to an audit entry's metadata how the audit log shows a
// change an integration key made: "integration key", from "api" with the
// key's masked value.
func (s *Service) keyMeta(ctx context.Context, m map[string]any) map[string]any {
	ik := api.IntegrationFrom(ctx)
	p := api.PrincipalFrom(ctx)
	if ik == nil || p == nil {
		return m
	}
	m["actor_label"] = "integration key"
	m["source"] = "api"
	if k, err := s.Q.GetAPIKey(ctx, store.GetAPIKeyParams{ID: p.ID, WorkspaceID: ik.WorkspaceID}); err == nil {
		m["via"] = k.Prefix
	}
	return m
}

// itemOf maps an operation to the catalog item it belongs to.
func itemOf(kind, op string) string {
	switch op {
	case OpUpsert, OpFields:
		return "upsert_contact"
	case OpAddToList:
		return "add_to_list"
	case OpTag:
		return "add_tag"
	case OpEvent:
		return "record_event"
	case OpSendMessage, OpTemplates:
		return "send_transactional"
	case OpWPForms:
		return "list_forms"
	case OpWPDraft:
		return "create_drafts"
	case OpLists:
		return "upsert_contact"
	case OpForms:
		return "form.submitted"
	}
	return ""
}

type problemBody struct {
	Code    string   `json:"code"`
	Detail  string   `json:"detail"`
	Title   string   `json:"title"`
	Missing []string `json:"missing"`
}

// Record implements api.Integrations: it keeps one call in the 7-day log,
// moves the connection's last call and remembers the workflow. Errors are
// logged, never returned: the call itself already happened.
func (s *Service) Record(ctx context.Context, c api.IntegrationCall) {
	if c.Integration == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ik := c.Integration
	wf := ParseWorkflow(c.Workflow)
	item := c.Notes[noteItem]
	if item == "" {
		item = itemOf(ik.Kind, c.Operation)
	}
	var code, detail string
	if c.Status >= 400 {
		var p problemBody
		if json.Unmarshal(c.Response, &p) == nil {
			code, detail = p.Code, p.Detail
			if detail == "" {
				detail = p.Title
			}
		}
	}
	target := json.RawMessage(c.Notes[noteTarget])
	var contactID *uuid.UUID
	if id, err := uuid.Parse(c.Notes[noteContact]); err == nil {
		contactID = &id
	}
	if len(target) == 0 {
		t, cid := s.describe(ctx, ik.WorkspaceID, c)
		target, _ = json.Marshal(t)
		if contactID == nil {
			contactID = cid
		}
	}
	keyID := c.KeyID
	err := s.Q.IntegrationCallInsert(ctx, store.IntegrationCallInsertParams{
		ID: uuid.Must(uuid.NewV7()), IntegrationID: ik.ID, WorkspaceID: ik.WorkspaceID, ApiKeyID: &keyID, OccurredAt: c.At,
		WorkflowKey: wf.Key(), WorkflowName: wf.Name, Operation: c.Operation, Item: item, Method: c.Method, Path: c.Path,
		Status: int32(c.Status), Code: code, Detail: clip(detail, 1000), DurationMs: int32(c.Duration.Milliseconds()),
		Target: target, ContactID: contactID, Request: requestLog(readableHeaders(c.Headers, wf), s.maskBody(ctx, ik.WorkspaceID, c.Operation, c.Request)),
		Response: jsonOrNil(c.Response), Ip: c.IP, Client: clientLabel(c.UserAgent),
	})
	if err != nil {
		s.log().WarnContext(ctx, "record integration call", "integration", ik.ID, "err", err)
		return
	}
	_ = s.Q.IntegrationTouchCall(ctx, store.IntegrationTouchCallParams{ID: ik.ID, At: c.At})
	_ = s.Q.IntegrationKeyTouchClient(ctx, store.IntegrationKeyTouchClientParams{ID: keyID, Client: clientLabel(c.UserAgent)})
	if !IsPlatform(ik.Kind) || c.Operation == OpMe || item == "" && c.Notes[noteUses] == "" {
		return
	}
	var t CallTarget
	_ = json.Unmarshal(target, &t)
	uses := s.workflowUses(ctx, ik.ID, wf.Key(), item, c.Notes[noteUses], t)
	_ = s.Q.IntegrationWorkflowSeen(ctx, store.IntegrationWorkflowSeenParams{
		IntegrationID: ik.ID, Key: wf.Key(), ExternalID: wf.ID, Name: wf.Name, Uses: uses, At: c.At,
	})
}

// WorkflowUse is one thing a workflow uses: a trigger, or an action with
// what it was last called with (the app words it: "Beta Program · tag
// beta-applicant · double opt-in").
type WorkflowUse struct {
	Item string `json:"item"`
	// Lists, Tags and OptIn describe a contact action.
	Lists []string `json:"lists,omitempty"`
	Tags  []string `json:"tags,omitempty"`
	OptIn string   `json:"optIn,omitempty" enum:"single,double"`
	// Event is the event "Record an event" records; Template the template
	// "Send a transactional email" sends.
	Event    string `json:"event,omitempty"`
	Template string `json:"template,omitempty"`
	// Starts and GoalOf name the live automations the event starts, or
	// that have it as their goal (worked out when the page is read).
	Starts []string `json:"starts,omitempty"`
	GoalOf []string `json:"goalOf,omitempty"`
}

// merge takes what a newer call says about the same use.
func (u *WorkflowUse) merge(n WorkflowUse) {
	if len(n.Lists) > 0 {
		u.Lists = n.Lists
	}
	if len(n.Tags) > 0 {
		u.Tags = n.Tags
	}
	if n.OptIn != "" {
		u.OptIn = n.OptIn
	}
	if n.Event != "" {
		u.Event = n.Event
	}
	if n.Template != "" {
		u.Template = n.Template
	}
}

// workflowUses adds what this call used to what the workflow used before.
// Without a note from the operation, the call's target says the event or
// the template.
func (s *Service) workflowUses(ctx context.Context, integrationID uuid.UUID, key, item, note string, target CallTarget) json.RawMessage {
	var uses []WorkflowUse
	if w, err := s.Q.IntegrationWorkflow(ctx, store.IntegrationWorkflowParams{IntegrationID: integrationID, Key: key}); err == nil {
		_ = json.Unmarshal(w.Uses, &uses)
	}
	use := WorkflowUse{Item: item}
	if note != "" {
		_ = json.Unmarshal([]byte(note), &use)
	} else {
		use.Event, use.Template = target.Event, target.Template
	}
	if use.Item != "" {
		replaced := false
		for i := range uses {
			if uses[i].Item == use.Item {
				uses[i].merge(use)
				replaced = true
			}
		}
		if !replaced {
			uses = append(uses, use)
		}
	}
	if uses == nil {
		uses = []WorkflowUse{}
	}
	b, _ := json.Marshal(uses)
	return b
}

func noteUse(ctx context.Context, use WorkflowUse) {
	b, _ := json.Marshal(use)
	api.NoteCall(ctx, noteUses, string(b))
	api.NoteCall(ctx, noteItem, use.Item)
}

// describe works out the target of a call to another package's operation
// (record an event, send a transactional email) from its bodies.
func (s *Service) describe(ctx context.Context, wsID uuid.UUID, c api.IntegrationCall) (CallTarget, *uuid.UUID) {
	var req map[string]any
	_ = json.Unmarshal(c.Request, &req)
	var resp map[string]any
	_ = json.Unmarshal(c.Response, &resp)
	str := func(m map[string]any, k string) string {
		v, _ := m[k].(string)
		return v
	}
	switch c.Operation {
	case OpMe:
		return CallTarget{Kind: "test"}, nil
	case OpEvent:
		email := strings.ToLower(str(req, "email"))
		var cid *uuid.UUID
		if id, err := uuid.Parse(str(resp, "contactId")); err == nil {
			cid = &id
		}
		return CallTarget{Kind: "record_event", Event: str(req, "event"), Email: email}, cid
	case OpSendMessage:
		email := ""
		if to, ok := req["to"].(map[string]any); ok {
			email = strings.ToLower(str(to, "email"))
		} else {
			email = strings.ToLower(str(req, "to"))
		}
		var cid *uuid.UUID
		if email != "" {
			if ct, err := s.Q.GetContactByEmail(ctx, store.GetContactByEmailParams{WorkspaceID: wsID, Email: email}); err == nil {
				cid = &ct.ID
			}
		}
		return CallTarget{Kind: "send_transactional", Template: str(req, "template"), Email: email}, cid
	}
	return CallTarget{Kind: "call"}, nil
}

// CallRequest is a logged request: the headers the log shows and the body
// with its sensitive values masked.
type CallRequest struct {
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// readableHeaders shows the workflow header as a person reads it: the apps
// percent-encode a name that isn't plain ASCII ("Stripe payment failed
// %E2%86%92 payment-failed email; id=284113907"), the call log shows
// "Stripe payment failed → payment-failed email; id=284113907".
func readableHeaders(headers map[string]string, wf Workflow) map[string]string {
	raw, ok := headers[api.HeaderWorkflow]
	if !ok || raw == "" {
		return headers
	}
	out := maps.Clone(headers)
	if s := wf.String(); s != "" {
		out[api.HeaderWorkflow] = s
	}
	return out
}

func requestLog(headers map[string]string, body json.RawMessage) json.RawMessage {
	if headers == nil {
		headers = map[string]string{}
	}
	b, _ := json.Marshal(CallRequest{Headers: headers, Body: body})
	return b
}

func jsonOrNil(b []byte) json.RawMessage {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || !json.Valid(b) {
		return nil
	}
	return json.RawMessage(b)
}

// Masked replaces a sensitive value in a stored body.
const Masked = "••••••"

// maskBody keeps a request body for the call log with its sensitive values
// masked: the variables a transactional template marks sensitive (never
// stored, as in the message log) and anything that looks like a secret.
// Keys keep the order they were sent in.
func (s *Service) maskBody(ctx context.Context, wsID uuid.UUID, op string, body []byte) json.RawMessage {
	raw := jsonOrNil(body)
	if raw == nil {
		return nil
	}
	var sensitive map[string]bool
	if op == OpSendMessage {
		var m struct {
			Template string `json:"template"`
		}
		if json.Unmarshal(raw, &m) == nil && m.Template != "" {
			sensitive = s.sensitiveVariables(ctx, wsID, m.Template)
		}
	}
	out, err := maskJSON(raw, sensitive)
	if err != nil {
		return nil
	}
	return out
}

// sensitiveVariables returns the variables a transactional template's live
// version marks sensitive: never stored, as in the message log.
func (s *Service) sensitiveVariables(ctx context.Context, wsID uuid.UUID, template string) map[string]bool {
	raw, err := s.Q.IntegrationTemplateVariables(ctx, store.IntegrationTemplateVariablesParams{WorkspaceID: wsID, Key: template})
	if err != nil {
		return nil
	}
	var vars []IntegrationTemplateVariable
	_ = json.Unmarshal(raw, &vars)
	out := map[string]bool{}
	for _, v := range vars {
		if v.Sensitive {
			out[v.Name] = true
		}
	}
	return out
}

var secretKey = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|authorization)`)

// maskJSON copies a JSON document, keys in their order, with the
// sensitive values masked: inside "data" or "variables", the template's
// sensitive variables and anything secret-like; elsewhere, secret-like
// string fields.
func maskJSON(raw []byte, sensitive map[string]bool) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out bytes.Buffer
	if err := maskNode(dec, &out, "", sensitive); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// maskNode copies one value; key is the field it's the value of.
func maskNode(dec *json.Decoder, out *bytes.Buffer, key string, sensitive map[string]bool) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, isDelim := tok.(json.Delim)
	if !isDelim {
		return writeScalar(out, tok)
	}
	if delim == '[' {
		out.WriteByte('[')
		for i := 0; dec.More(); i++ {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := maskNode(dec, out, key, sensitive); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil {
			return err
		}
		out.WriteByte(']')
		return nil
	}
	variables := key == "data" || key == "variables"
	out.WriteByte('{')
	for i := 0; dec.More(); i++ {
		kt, err := dec.Token()
		if err != nil {
			return err
		}
		k, _ := kt.(string)
		if i > 0 {
			out.WriteByte(',')
		}
		if err := writeScalar(out, k); err != nil {
			return err
		}
		out.WriteByte(':')
		secret := secretKey.MatchString(k)
		if variables && (sensitive[k] || secret) {
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return err
			}
			if err := writeScalar(out, Masked); err != nil {
				return err
			}
			continue
		}
		if secret {
			var v json.RawMessage
			if err := dec.Decode(&v); err != nil {
				return err
			}
			if len(v) > 0 && v[0] == '"' {
				if err := writeScalar(out, Masked); err != nil {
					return err
				}
				continue
			}
			inner := json.NewDecoder(bytes.NewReader(v))
			inner.UseNumber()
			if err := maskNode(inner, out, k, sensitive); err != nil {
				return err
			}
			continue
		}
		if err := maskNode(dec, out, k, sensitive); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	out.WriteByte('}')
	return nil
}

// writeScalar writes a string, number, boolean or null as JSON, leaving
// <, > and & as they are.
func writeScalar(out *bytes.Buffer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	out.Truncate(out.Len() - 1) // Encode adds a newline
	return nil
}

var emailRe = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)

// hideAddresses replaces email addresses in a body with a word, for members
// who can't see contacts or for contacts erased since the call.
func hideAddresses(b json.RawMessage, with string) json.RawMessage {
	if len(b) == 0 {
		return b
	}
	return json.RawMessage(emailRe.ReplaceAllString(string(b), with))
}

// clientLabel turns a User-Agent into the app that called: "Zapier,
// ReputeCast app 1.2.0", "Make, ReputeCast app 1.0.4", "WordPress 6.9.2".
func clientLabel(ua string) string {
	l := strings.ToLower(ua)
	app := ""
	if i := strings.Index(l, "reputecast-"); i >= 0 {
		rest := ua[i+len("reputecast-"):]
		name, ver, _ := strings.Cut(strings.Fields(rest + " ")[0], "/")
		switch strings.ToLower(name) {
		case "zapier":
			app = "Zapier, ReputeCast app"
		case "make":
			app = "Make, ReputeCast app"
		case "n8n":
			app = "n8n, ReputeCast node"
		case "wordpress":
			app = "WordPress, ReputeCast plugin"
		}
		if app != "" && ver != "" {
			return app + " " + ver
		}
		if app != "" {
			return app
		}
	}
	switch {
	case ua == "":
		return ""
	case strings.Contains(l, "zapier"):
		return "Zapier"
	case strings.Contains(l, "make.com") || strings.HasPrefix(l, "make/"):
		return "Make"
	case strings.Contains(l, "n8n"):
		return "n8n"
	case strings.Contains(l, "wordpress"):
		return "WordPress"
	}
	tok := strings.Fields(ua)[0]
	if i := strings.IndexByte(tok, '/'); i > 0 {
		tok = tok[:i]
	}
	return clip(tok, 40)
}
