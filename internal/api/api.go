// Package api holds the Huma (OpenAPI 3.1) setup shared by every domain:
// operation helpers that declare the required permission, the middleware
// that authenticates users, API keys and MCP agents and authorizes them with
// internal/authz, and RFC 9457 error helpers.
//
// Domain packages register their operations with Register* helpers and read
// the caller with PrincipalFrom. This package never imports domain packages.
package api

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/authz"
)

// Version is the URL prefix of the public API.
const Version = "/api/v1"

// AccessKind says who may call an operation.
type AccessKind string

const (
	// AccessPublic needs no authentication (sign-in, public pages, webhooks).
	AccessPublic AccessKind = "public"
	// AccessUser needs a signed-in person (account, workspace list).
	AccessUser AccessKind = "user"
	// AccessInstanceAdmin needs an instance admin.
	AccessInstanceAdmin AccessKind = "instance_admin"
	// AccessWorkspace needs a member, API key or agent holding Permission in
	// the workspace named by the {workspace} path parameter.
	AccessWorkspace AccessKind = "workspace"
	// AccessKey needs an API key; the workspace is the key's own, so the
	// caller needs only the instance URL and the key (integrations' "Test
	// connection").
	AccessKey AccessKind = "key"
)

// Access is the requirement attached to an operation.
type Access struct {
	Kind       AccessKind
	Permission authz.Permission
	// AllowPending lets users with a temporary password or missing required
	// two-factor call the operation (the setup and account endpoints).
	AllowPending bool
	// HumansOnly rejects API keys and agents (account security, approvals).
	HumansOnly bool
	// Integration limits the operation to the keys of one kind of
	// integration ("wordpress"), or of any integration ("*").
	Integration string
}

// Common access requirements.
var (
	Public        = Access{Kind: AccessPublic}
	Authenticated = Access{Kind: AccessUser}
	Pending       = Access{Kind: AccessUser, AllowPending: true}
	InstanceAdmin = Access{Kind: AccessInstanceAdmin, HumansOnly: true}
)

// Workspace requires perm in the workspace from the {workspace} path parameter.
func Workspace(perm authz.Permission) Access {
	return Access{Kind: AccessWorkspace, Permission: perm}
}

// WorkspaceHuman is Workspace but rejects API keys and agents.
func WorkspaceHuman(perm authz.Permission) Access {
	return Access{Kind: AccessWorkspace, Permission: perm, HumansOnly: true}
}

// CallingKey is for operations about the API key that calls them, in its
// own workspace. It needs no permission: a key may always describe itself.
var CallingKey = Access{Kind: AccessKey}

// IntegrationOnly is Workspace for the keys of one kind of integration
// ("wordpress"), or of any integration ("*"): the operations the apps and
// the WordPress plugin call, which people and other keys don't.
func IntegrationOnly(kind string, perm authz.Permission) Access {
	return Access{Kind: AccessWorkspace, Permission: perm, Integration: kind}
}

const accessMetaKey = "reputecast.access"

// Op describes an operation. Path is relative to /api/v1 and workspace
// operations start with "/workspaces/{workspace}".
type Op struct {
	ID          string
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	Access      Access
	// Status overrides the default success status (200, or 204 without a body).
	Status int
	// MaxBodyBytes overrides Huma's default 1 MiB request limit.
	MaxBodyBytes int64
}

func (o Op) operation() huma.Operation {
	op := huma.Operation{
		OperationID:   o.ID,
		Method:        o.Method,
		Path:          Version + o.Path,
		Summary:       o.Summary,
		Description:   o.Description,
		Tags:          o.Tags,
		DefaultStatus: o.Status,
		MaxBodyBytes:  o.MaxBodyBytes,
		Metadata:      map[string]any{accessMetaKey: o.Access},
		Extensions:    map[string]any{"x-access": string(o.Access.Kind)},
	}
	if o.Access.Permission != "" {
		op.Extensions["x-permission"] = string(o.Access.Permission)
	}
	if o.Access.Integration != "" {
		op.Extensions["x-integration"] = o.Access.Integration
	}
	switch o.Access.Kind {
	case AccessPublic:
	case AccessKey:
		op.Security = []map[string][]string{{"bearer": {}}}
	case AccessWorkspace:
		switch {
		case o.Access.HumansOnly:
			op.Security = []map[string][]string{{"session": {}}}
		case o.Access.Integration != "":
			op.Security = []map[string][]string{{"bearer": {}}}
		default:
			op.Security = []map[string][]string{{"session": {}}, {"bearer": {}}}
		}
	default:
		op.Security = []map[string][]string{{"session": {}}}
	}
	if o.Access.Kind == AccessWorkspace && !strings.Contains(o.Path, "{workspace}") {
		panic("api: workspace operation " + o.ID + " must have a {workspace} path parameter")
	}
	return op
}

// Register adds an operation with a typed handler.
func Register[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	in := reflect.TypeFor[I]()
	checkInput(in, o.ID)
	op := o.operation()
	if rawBody(in) {
		op.BodyReadTimeout = uploadReadTimeout(a)
	}
	huma.Register(a, op, h)
}

// UploadReadTimeout is how long the body of a raw upload (an input with
// RawBody []byte: an import, a CSV, a DMARC report, a file's chunk) may
// take to arrive on an API not wrapped by WithReadTimeout. It is
// http.read_timeout's default. Since v2.40 Huma reads raw bodies with the
// 5 s deadline it gives JSON ones, unless the operation sets its own.
const UploadReadTimeout = 30 * time.Second

// WithReadTimeout returns a, registering raw uploads with d, the server's
// read timeout, to read their body.
func WithReadTimeout(a huma.API, d time.Duration) huma.API {
	if d <= 0 {
		return a
	}
	return readTimeoutAPI{API: a, d: d}
}

type readTimeoutAPI struct {
	huma.API
	d time.Duration
}

func uploadReadTimeout(a huma.API) time.Duration {
	if r, ok := a.(readTimeoutAPI); ok {
		return r.d
	}
	return UploadReadTimeout
}

// rawBody reports whether an input takes its body as raw bytes.
func rawBody(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	f, ok := t.FieldByName("RawBody")
	return ok && f.Type.Kind() == reflect.Slice && f.Type.Elem().Kind() == reflect.Uint8
}

// Get, Post, Put, Patch and Delete are shorthands for Register.
func Get[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodGet
	Register(a, o, h)
}

func Post[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodPost
	Register(a, o, h)
}

func Put[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodPut
	Register(a, o, h)
}

func Patch[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodPatch
	Register(a, o, h)
}

func Delete[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodDelete
	if o.Status == 0 {
		o.Status = http.StatusNoContent
	}
	Register(a, o, h)
}

// accessOf returns the access requirement of an operation.
func accessOf(op *huma.Operation) Access {
	a, _ := AccessOf(op)
	return a
}

// AccessOf returns the access requirement the middleware enforces on an
// operation, and whether it declares one (an operation that doesn't is
// public). Tests walk the registered operations with it.
func AccessOf(op *huma.Operation) (Access, bool) {
	if op == nil {
		return Public, false
	}
	a, ok := op.Metadata[accessMetaKey].(Access)
	if !ok {
		return Public, false
	}
	return a, true
}

// WorkspaceInput is embedded by inputs of workspace operations.
type WorkspaceInput struct {
	Workspace string `path:"workspace" doc:"Workspace slug" example:"acme-newsletter" pattern:"^[a-z0-9][a-z0-9-]{1,62}$"`
}

// Empty is the output of operations without a body.
type Empty struct{}

// NoContent is returned by operations that answer 204.
var NoContent = &Empty{}

// Body wraps a JSON response body.
type Body[T any] struct {
	Body T
}

// OK wraps v as a response body.
func OK[T any](v T) *Body[T] { return &Body[T]{Body: v} }

// Page is the envelope of cursor-paginated lists.
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty" doc:"Pass as ?cursor= to fetch the next page"`
	Total      *int64 `json:"total,omitempty"`
}

// PageInput is embedded by inputs of paginated lists.
type PageInput struct {
	Cursor string `query:"cursor" doc:"Opaque cursor from a previous page"`
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500"`
}
