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
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/authz"
)

// Version is the URL prefix of the public API.
const Version = "/api/v1"

// AccessKind says who may call an operation.
type AccessKind string

// Access kinds.
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
	switch o.Access.Kind {
	case AccessPublic:
	case AccessWorkspace:
		if o.Access.HumansOnly {
			op.Security = []map[string][]string{{"session": {}}}
		} else {
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
	huma.Register(a, o.operation(), h)
}

// Get, Post, Put, Patch and Delete are shorthands for Register.
func Get[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodGet
	Register(a, o, h)
}

// Post registers a POST operation.
func Post[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodPost
	Register(a, o, h)
}

// Put registers a PUT operation.
func Put[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodPut
	Register(a, o, h)
}

// Patch registers a PATCH operation.
func Patch[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodPatch
	Register(a, o, h)
}

// Delete registers a DELETE operation.
func Delete[I, O any](a huma.API, o Op, h func(context.Context, *I) (*O, error)) {
	o.Method = http.MethodDelete
	if o.Status == 0 {
		o.Status = http.StatusNoContent
	}
	Register(a, o, h)
}

// accessOf returns the access requirement of an operation.
func accessOf(op *huma.Operation) Access {
	if op == nil {
		return Public
	}
	if a, ok := op.Metadata[accessMetaKey].(Access); ok {
		return a
	}
	return Public
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
