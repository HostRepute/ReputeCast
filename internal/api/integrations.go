package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
)

// Error codes of calls made with an integration key (the integrations hub
// shows them in each integration's call log).
const (
	CodeKeyRevoked       = "key_revoked"
	CodeKeyExpired       = "key_expired"
	CodeOwnerDeactivated = "owner_deactivated"
	CodeNotForThisKey    = "integration_operation_denied"
)

// HeaderWorkflow names the workflow (a Zap, a Make scenario, an n8n
// workflow) behind a call: "<name>; id=<id>", the name percent-encoded when
// it isn't ASCII.
const HeaderWorkflow = "X-ReputeCast-Workflow"

// MaxCallBody is how much of a request and a response the call log keeps.
const MaxCallBody = 16 << 10

// IntegrationKey describes the integration an API key belongs to. Such a key
// works only for the operations of the triggers and actions turned on, even
// where its permissions would allow more.
type IntegrationKey struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	// Kind is the integration ("zapier", "make", "n8n", "wordpress"); Name
	// is how people call it ("Zapier").
	Kind string
	Name string
	// Operations are the operation IDs the key may call.
	Operations   map[string]bool
	Disconnected bool
}

// Allows reports whether the key may call an operation.
func (k *IntegrationKey) Allows(operationID string) bool {
	return k != nil && !k.Disconnected && k.Operations[operationID]
}

// IntegrationCall is one call made with an integration key, for its call log.
type IntegrationCall struct {
	Integration *IntegrationKey
	KeyID       uuid.UUID
	Operation   string
	Method      string
	Path        string
	Status      int
	At          time.Time
	Duration    time.Duration
	// Workflow is the raw X-ReputeCast-Workflow header.
	Workflow string
	// Request and Response are the first MaxCallBody bytes of each body.
	Request  []byte
	Response []byte
	IP       *netip.Addr
	// UserAgent identifies the app ("ReputeCast-Zapier/1.2.0").
	UserAgent string
	// Notes are what the operation said about the call (NoteCall).
	Notes map[string]string
}

// Integrations connects the authenticator to the integrations hub
// (internal/integrations), which knows what each integration key may call
// and keeps their call log.
type Integrations interface {
	// Key returns the integration of an integration key (its integration_id).
	Key(ctx context.Context, integrationID uuid.UUID) (*IntegrationKey, error)
	// Record logs a call made with an integration key. It must not fail the call.
	Record(ctx context.Context, call IntegrationCall)
}

const (
	integrationKey ctxKey = 100 + iota
	callNotesKey
)

// callNotes collects what an operation says about a call made with an
// integration key, for its call log.
type callNotes struct {
	mu sync.Mutex
	m  map[string]string
}

// NoteCall tells the call log something about the current call ("item",
// "target", "contact"). It does nothing outside integration calls.
func NoteCall(ctx context.Context, key, value string) {
	n, _ := ctx.Value(callNotesKey).(*callNotes)
	if n == nil {
		return
	}
	n.mu.Lock()
	n.m[key] = value
	n.mu.Unlock()
}

// IntegrationFrom returns the integration of the calling key, or nil when
// the caller isn't an integration key.
func IntegrationFrom(ctx context.Context) *IntegrationKey {
	k, _ := ctx.Value(integrationKey).(*IntegrationKey)
	return k
}

// WithIntegration stores the calling integration (tests and internal callers).
func WithIntegration(ctx context.Context, k *IntegrationKey) context.Context {
	return context.WithValue(ctx, integrationKey, k)
}

// capped keeps the first MaxCallBody bytes written to it and drops the rest,
// so it never slows down or fails the request it watches.
type capped struct{ b []byte }

func (c *capped) Write(p []byte) (int, error) {
	if room := MaxCallBody - len(c.b); room > 0 {
		if len(p) < room {
			room = len(p)
		}
		c.b = append(c.b, p[:room]...)
	}
	return len(p), nil
}

// humaContext lets recorder embed a huma.Context and keep its Context method.
type humaContext huma.Context

// recorder is the huma.Context of a call made with an integration key: it
// keeps a copy of both bodies for the call log.
type recorder struct {
	humaContext
	req, resp capped
	r         io.Reader
	w         io.Writer
}

func (c *recorder) BodyReader() io.Reader {
	if c.r == nil {
		c.r = io.TeeReader(c.humaContext.BodyReader(), &c.req)
	}
	return c.r
}

func (c *recorder) BodyWriter() io.Writer {
	if c.w == nil {
		c.w = io.MultiWriter(c.humaContext.BodyWriter(), &c.resp)
	}
	return c.w
}

// callOf starts the log entry of a call made with an integration key.
func (a *Authenticator) callOf(ctx context.Context, hctx huma.Context, ik *IntegrationKey, keyID uuid.UUID, start time.Time) IntegrationCall {
	u := hctx.URL()
	c := IntegrationCall{Integration: ik, KeyID: keyID, Method: hctx.Method(), Path: u.Path, At: start,
		Workflow: hctx.Header(HeaderWorkflow), UserAgent: hctx.Header("User-Agent")}
	if op := hctx.Operation(); op != nil {
		c.Operation = op.OperationID
	}
	c.IP = audit.RequestFrom(ctx).IP
	return c
}

// recordRefused logs a call ReputeCast refused before running it.
func (a *Authenticator) recordRefused(ctx context.Context, hctx huma.Context, ik *IntegrationKey, keyID uuid.UUID, start time.Time, p *Problem) {
	if a.Integrations == nil || ik == nil {
		return
	}
	c := a.callOf(ctx, hctx, ik, keyID, start)
	c.Status = p.Status
	c.Duration = a.now().Sub(start)
	c.Response, _ = json.Marshal(p)
	a.Integrations.Record(context.WithoutCancel(ctx), c)
}

// serveRecorded runs an operation for an integration key and logs the call.
func (a *Authenticator) serveRecorded(ctx context.Context, hctx huma.Context, ik *IntegrationKey, keyID uuid.UUID, start time.Time, next func(huma.Context)) {
	notes := &callNotes{m: map[string]string{}}
	ctx = context.WithValue(ctx, callNotesKey, notes)
	rc := &recorder{humaContext: huma.WithContext(hctx, ctx)}
	next(rc)
	c := a.callOf(ctx, hctx, ik, keyID, start)
	notes.mu.Lock()
	c.Notes = notes.m
	notes.mu.Unlock()
	c.Status = rc.Status()
	if c.Status == 0 {
		c.Status = http.StatusOK
	}
	c.Duration = a.now().Sub(start)
	c.Request, c.Response = rc.req.b, rc.resp.b
	a.Integrations.Record(context.WithoutCancel(ctx), c)
}
