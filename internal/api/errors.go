package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
)

// Problem is ReputeCast's RFC 9457 error body: Huma's model plus a stable
// machine-readable code and, for permission errors, who can help.
type Problem struct {
	huma.ErrorModel
	Code       string   `json:"code,omitempty" doc:"Stable error code, e.g. permission_denied"`
	Permission string   `json:"permission,omitempty" doc:"The missing permission key"`
	Helpers    []string `json:"helpers,omitempty" doc:"Members who hold the missing permission"`
	// Meta carries structured details for specific codes, e.g. lockedUntil
	// and attemptsLeft for sign-in errors.
	Meta map[string]any `json:"meta,omitempty"`
}

// Error codes used across the API.
const (
	CodeUnauthenticated     = "unauthenticated"
	CodeSessionExpired      = "session_expired"
	CodePasswordChange      = "password_change_required"
	CodeTwoFactorSetup      = "two_factor_setup_required"
	CodePermissionDenied    = "permission_denied"
	CodeEscalation          = "escalation_denied"
	CodeNotFound            = "not_found"
	CodeConflict            = "conflict"
	CodeValidation          = "validation_failed"
	CodeRateLimited         = "rate_limited"
	CodeLocked              = "locked"
	CodeInstanceNotSetUp    = "instance_not_set_up"
	CodeHumansOnly          = "humans_only"
	CodeWorkspaceNotFound   = "workspace_not_found"
	CodeLastOwner           = "last_owner"
	CodeInvalidCredentials  = "invalid_credentials"
	CodeProviderUnavailable = "provider_unavailable"
)

func init() {
	// Lists are never null in responses: handlers return empty slices, and the
	// generated TypeScript types say T[] rather than T[] | null.
	huma.DefaultArrayNullable = false

	// Every error Huma creates (validation, 404, 500…) uses our model.
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		for _, e := range errs {
			if m, ok := e.(problemMarker); ok {
				return m.p
			}
		}
		details := make([]*huma.ErrorDetail, 0, len(errs))
		for _, e := range errs {
			if e == nil {
				continue
			}
			var d *huma.ErrorDetail
			if errors.As(e, &d) {
				details = append(details, d)
				continue
			}
			details = append(details, &huma.ErrorDetail{Message: e.Error()})
		}
		p := &Problem{ErrorModel: huma.ErrorModel{
			Status: status,
			Title:  http.StatusText(status),
			Detail: msg,
			Errors: details,
		}}
		if status == http.StatusUnprocessableEntity || status == http.StatusBadRequest {
			p.Code = CodeValidation
		}
		return p
	}
}

// Err builds a problem with a code.
func Err(status int, code, detail string) *Problem {
	return &Problem{ErrorModel: huma.ErrorModel{Status: status, Title: http.StatusText(status), Detail: detail}, Code: code}
}

// WithMeta adds structured details to a problem.
func (p *Problem) WithMeta(kv ...any) *Problem {
	if p.Meta == nil {
		p.Meta = map[string]any{}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			p.Meta[k] = kv[i+1]
		}
	}
	return p
}

// NotFound returns a 404 for a named thing ("Campaign").
func NotFound(what string) *Problem {
	return Err(http.StatusNotFound, CodeNotFound, fmt.Sprintf("%s not found.", what))
}

// Conflict returns a 409 with an explanation.
func Conflict(detail string) *Problem { return Err(http.StatusConflict, CodeConflict, detail) }

// BadRequest returns a 400 with an explanation.
func BadRequest(detail string) *Problem { return Err(http.StatusBadRequest, CodeValidation, detail) }

// Invalid returns a 422 for one field: Invalid("body.email", "Use a work email.").
func Invalid(location, message string) *Problem {
	p := Err(http.StatusUnprocessableEntity, CodeValidation, message)
	p.Errors = []*huma.ErrorDetail{{Location: location, Message: message}}
	return p
}

// Forbidden returns a 403 for a missing permission, naming who can help.
func Forbidden(perm authz.Permission, helpers []string) *Problem {
	p := Err(http.StatusForbidden, CodePermissionDenied, authz.DeniedMessage(perm, helpers))
	p.Permission = string(perm)
	p.Helpers = helpers
	return p
}

// MapError turns domain errors into problems: pgx.ErrNoRows becomes 404,
// authz denials become 403, and anything else passes through unchanged.
func MapError(err error, what string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return NotFound(what)
	}
	var esc *authz.EscalationError
	if errors.As(err, &esc) {
		p := Err(http.StatusForbidden, CodeEscalation, capitalize(esc.Error())+".")
		if len(esc.Missing) == 1 {
			p.Permission = string(esc.Missing[0])
		}
		return p
	}
	var den *authz.DeniedError
	if errors.As(err, &den) {
		return Forbidden(den.Permission, den.Helpers)
	}
	return err
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 'a' - 'A'
	}
	return string(b)
}
