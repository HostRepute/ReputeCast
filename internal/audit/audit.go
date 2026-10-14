// Package audit writes the immutable audit log: who did what to which
// target, before and after, from where, and whether it was allowed.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Result of an audited action.
const (
	Allowed = "allowed"
	Denied  = "denied"
	Failed  = "error"
)

// Event is one audit entry. Use Log to write it.
type Event struct {
	WorkspaceID *uuid.UUID
	Action      string // dotted verb: "routing.update", "apikey.create", "auth.sign_in"
	TargetType  string
	TargetID    string
	TargetName  string
	Result      string
	Reason      string
	Before      any
	After       any
	Metadata    map[string]any
	// Actor overrides the actor taken from the context (system jobs, sign-in).
	Actor *Actor
}

// Actor identifies who acted.
type Actor struct {
	Type authz.ActorType
	ID   *uuid.UUID
	Name string
}

// Request carries per-request details stored with every entry.
type Request struct {
	IP        *netip.Addr
	UserAgent string
	RequestID string
}

type ctxKey int

const (
	requestKey ctxKey = iota
	principalKey
)

// WithRequest stores request details in ctx for later entries.
func WithRequest(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, requestKey, r)
}

// RequestFrom returns the request details stored in ctx.
func RequestFrom(ctx context.Context) Request {
	r, _ := ctx.Value(requestKey).(Request)
	return r
}

// WithPrincipal stores the acting principal in ctx.
func WithPrincipal(ctx context.Context, p *authz.Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFrom returns the acting principal stored in ctx, if any.
func PrincipalFrom(ctx context.Context) *authz.Principal {
	p, _ := ctx.Value(principalKey).(*authz.Principal)
	return p
}

// Logger writes audit entries. Writes go through the caller's transaction
// when one is passed, so an action and its audit entry commit together.
type Logger struct {
	q   store.Querier
	log *slog.Logger
	now func() time.Time
}

// New returns an audit logger.
func New(q store.Querier, log *slog.Logger) *Logger {
	return &Logger{q: q, log: log, now: time.Now}
}

// Log writes an entry using the logger's own connection.
func (l *Logger) Log(ctx context.Context, e Event) error { return l.LogWith(ctx, l.q, e) }

// LogWith writes an entry through q (for example a transaction's Queries).
func (l *Logger) LogWith(ctx context.Context, q store.Querier, e Event) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	req := RequestFrom(ctx)
	actor := Actor{Type: authz.ActorSystem, Name: "System"}
	if e.Actor != nil {
		actor = *e.Actor
	} else if p := PrincipalFrom(ctx); p != nil {
		pid := p.ID
		actor = Actor{Type: p.Type, ID: &pid, Name: p.Name}
		if e.WorkspaceID == nil && p.WorkspaceID != uuid.Nil {
			ws := p.WorkspaceID
			e.WorkspaceID = &ws
		}
	}
	if e.Result == "" {
		e.Result = Allowed
	}
	meta := e.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	params := store.InsertAuditEventParams{
		ID:          id,
		WorkspaceID: e.WorkspaceID,
		OccurredAt:  l.now(),
		ActorType:   string(actor.Type),
		ActorID:     actor.ID,
		ActorName:   actor.Name,
		Action:      e.Action,
		TargetType:  e.TargetType,
		TargetID:    e.TargetID,
		TargetName:  e.TargetName,
		Result:      e.Result,
		Reason:      e.Reason,
		Ip:          req.IP,
		UserAgent:   req.UserAgent,
		RequestID:   req.RequestID,
		Before:      marshal(e.Before),
		After:       marshal(e.After),
		Metadata:    marshal(meta),
	}
	if err := q.InsertAuditEvent(ctx, params); err != nil {
		l.log.ErrorContext(ctx, "audit write failed", "action", e.Action, "err", err)
		return err
	}
	return nil
}

func marshal(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	if raw, ok := v.(json.RawMessage); ok {
		return raw
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
