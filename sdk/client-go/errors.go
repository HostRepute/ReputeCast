package reputecast

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Error is an error answer of the API (an RFC 9457 problem): its HTTP
// status, the stable Code (permission_denied, validation_failed,
// not_found…), the sentence that explains it, the fields at fault and, for
// a missing permission, the members who can help.
type Error struct {
	Status int
	// Code is the stable machine-readable code, "" when the answer had none.
	Code   string
	Title  string
	Detail string
	Type   string
	// Errors are the fields at fault, for validation errors.
	Errors []FieldError
	// Permission is the missing permission key, for permission errors, and
	// Helpers the members who hold it.
	Permission string
	Helpers    []string
	// Meta holds structured details for some codes (a sign-in lock, an
	// error reference).
	Meta   map[string]any
	Header http.Header
}

// FieldError is one field's error in a problem.
type FieldError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
	Value    any    `json:"value,omitempty"`
}

func (e *Error) Error() string {
	msg := e.Detail
	if msg == "" {
		msg = e.Title
	}
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	if e.Code != "" {
		return fmt.Sprintf("reputecast: %d %s: %s", e.Status, e.Code, msg)
	}
	return fmt.Sprintf("reputecast: %d: %s", e.Status, msg)
}

// NewError reads an error answer: a problem document, other JSON or text.
func NewError(status int, header http.Header, body []byte) *Error {
	var p struct {
		Code       string         `json:"code"`
		Title      string         `json:"title"`
		Detail     string         `json:"detail"`
		Type       string         `json:"type"`
		Errors     []FieldError   `json:"errors"`
		Permission string         `json:"permission"`
		Helpers    []string       `json:"helpers"`
		Meta       map[string]any `json:"meta"`
	}
	e := &Error{Status: status, Header: header}
	if err := json.Unmarshal(body, &p); err == nil {
		e.Code, e.Title, e.Detail, e.Type = p.Code, p.Title, p.Detail, p.Type
		e.Errors, e.Permission, e.Helpers, e.Meta = p.Errors, p.Permission, p.Helpers, p.Meta
		return e
	}
	e.Title = http.StatusText(status)
	text := strings.TrimSpace(strings.ToValidUTF8(string(body), "\uFFFD"))
	if r := []rune(text); len(r) > 500 {
		text = string(r[:500])
	}
	e.Detail = text
	return e
}
