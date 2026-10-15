package integrations

import (
	"bytes"
	"context"
	"encoding/json"
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
	case OpWPDraft, OpWPCategories:
		return "create_drafts"
	case OpLists:
		return "upsert_contact"
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
		Target: target, ContactID: contactID, Request: s.maskBody(ctx, ik.WorkspaceID, c.Operation, c.Request),
		Response: jsonOrNil(c.Response), Ip: c.IP, Client: clientLabel(c.UserAgent),
	})
	if err != nil {
		s.log().WarnContext(ctx, "record integration call", "integration", ik.ID, "err", err)
		return
	}
	_ = s.Q.IntegrationTouchCall(ctx, store.IntegrationTouchCallParams{ID: ik.ID, At: c.At})
	if !IsPlatform(ik.Kind) || c.Operation == OpMe || item == "" && c.Notes[noteUses] == "" {
		return
	}
	uses := s.workflowUses(ctx, ik.ID, wf.Key(), item, c.Notes[noteUses])
	_ = s.Q.IntegrationWorkflowSeen(ctx, store.IntegrationWorkflowSeenParams{
		IntegrationID: ik.ID, Key: wf.Key(), ExternalID: wf.ID, Name: wf.Name, Uses: uses, At: c.At,
	})
}

// WorkflowUse is one thing a workflow uses: an action with its detail
// ("Beta Program · tag beta-applicant · double opt-in"), or a trigger.
type WorkflowUse struct {
	Item   string `json:"item"`
	Detail string `json:"detail,omitempty"`
}

// workflowUses adds what this call used to what the workflow used before.
func (s *Service) workflowUses(ctx context.Context, integrationID uuid.UUID, key, item, note string) json.RawMessage {
	var uses []WorkflowUse
	if w, err := s.Q.IntegrationWorkflow(ctx, store.IntegrationWorkflowParams{IntegrationID: integrationID, Key: key}); err == nil {
		_ = json.Unmarshal(w.Uses, &uses)
	}
	use := WorkflowUse{Item: item}
	if note != "" {
		_ = json.Unmarshal([]byte(note), &use)
	}
	if use.Item != "" {
		replaced := false
		for i := range uses {
			if uses[i].Item == use.Item {
				if use.Detail != "" {
					uses[i].Detail = use.Detail
				}
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

func noteUse(ctx context.Context, item, detail string) {
	b, _ := json.Marshal(WorkflowUse{Item: item, Detail: detail})
	api.NoteCall(ctx, noteUses, string(b))
	api.NoteCall(ctx, noteItem, item)
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
func (s *Service) maskBody(ctx context.Context, wsID uuid.UUID, op string, body []byte) json.RawMessage {
	raw := jsonOrNil(body)
	if raw == nil {
		return nil
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	var sensitive map[string]bool
	if op == OpSendMessage && SensitiveVariables != nil {
		if m, ok := v.(map[string]any); ok {
			if tpl, _ := m["template"].(string); tpl != "" {
				sensitive = SensitiveVariables(ctx, s.Q, wsID, tpl)
			}
		}
	}
	v = maskValue(v, "", sensitive)
	out, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return out
}

// SensitiveVariables returns the variables a transactional template marks
// sensitive. Transactional email (M15) sets it; without it only secret-like
// fields are masked.
var SensitiveVariables func(ctx context.Context, q *store.Queries, workspaceID uuid.UUID, template string) map[string]bool

var secretKey = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|authorization)`)

func maskValue(v any, key string, sensitive map[string]bool) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			if k == "data" || k == "variables" {
				if d, ok := x.(map[string]any); ok {
					for dk, dv := range d {
						if sensitive[dk] || secretKey.MatchString(dk) {
							d[dk] = Masked
						} else {
							d[dk] = maskValue(dv, dk, sensitive)
						}
					}
					continue
				}
			}
			if secretKey.MatchString(k) {
				if _, isStr := x.(string); isStr {
					t[k] = Masked
					continue
				}
			}
			t[k] = maskValue(x, k, sensitive)
		}
		return t
	case []any:
		for i := range t {
			t[i] = maskValue(t[i], key, sensitive)
		}
		return t
	}
	return v
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
