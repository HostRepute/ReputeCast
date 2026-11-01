package logging

import (
	"context"
	"log/slog"
	"sync"
)

// A request's log lines carry its request id, route pattern, workspace and
// caller (review 2026-11 OP-7). The HTTP stack starts the fields; the API's
// authentication fills in the workspace and the caller once it knows them;
// ContextHandler adds them to every line logged with the request's context
// (slog's *Context methods), the access log line included.

// Fields are what a request's log lines say about it. They are filled in as
// the request goes through the stack, so they are read under a lock.
type Fields struct {
	mu        sync.Mutex
	requestID string
	route     func() string
	workspace string
	caller    string
	job       string
}

type fieldsKey struct{}

// WithRequest starts a request's fields. route reads the route pattern
// (known only once routing is done), and may be nil.
func WithRequest(ctx context.Context, requestID string, route func() string) context.Context {
	return context.WithValue(ctx, fieldsKey{}, &Fields{requestID: requestID, route: route})
}

// WithJob starts a background job's fields ("kind:id").
func WithJob(ctx context.Context, job string) context.Context {
	return context.WithValue(ctx, fieldsKey{}, &Fields{job: job})
}

// FieldsFrom returns the fields of ctx's request or job, or nil.
func FieldsFrom(ctx context.Context) *Fields {
	if ctx == nil {
		return nil
	}
	f, _ := ctx.Value(fieldsKey{}).(*Fields)
	return f
}

// SetWorkspace names the workspace a request acts in (its slug).
func SetWorkspace(ctx context.Context, slug string) {
	if f := FieldsFrom(ctx); f != nil {
		f.mu.Lock()
		f.workspace = slug
		f.mu.Unlock()
	}
}

// SetCaller names who makes a request: "user:<id>", "key:<id>" or
// "agent:<id>". Never a name or an email.
func SetCaller(ctx context.Context, caller string) {
	if f := FieldsFrom(ctx); f != nil {
		f.mu.Lock()
		f.caller = caller
		f.mu.Unlock()
	}
}

// Attrs returns the fields as log attributes, leaving out the empty ones.
func (f *Fields) Attrs() []slog.Attr {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]slog.Attr, 0, 5)
	if f.requestID != "" {
		out = append(out, slog.String("request_id", f.requestID))
	}
	if f.route != nil {
		if r := f.route(); r != "" {
			out = append(out, slog.String("route", r))
		}
	}
	if f.workspace != "" {
		out = append(out, slog.String("workspace", f.workspace))
	}
	if f.caller != "" {
		out = append(out, slog.String("caller", f.caller))
	}
	if f.job != "" {
		out = append(out, slog.String("job", f.job))
	}
	return out
}

// ContextHandler adds the context's request or job fields to each record.
type ContextHandler struct{ slog.Handler }

// Handle implements slog.Handler.
func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if f := FieldsFrom(ctx); f != nil {
		r.AddAttrs(f.Attrs()...)
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (h ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ContextHandler{h.Handler.WithAttrs(attrs)}
}

// WithGroup implements slog.Handler.
func (h ContextHandler) WithGroup(name string) slog.Handler {
	return ContextHandler{h.Handler.WithGroup(name)}
}
