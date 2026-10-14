package manage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// CommunityKindRemote is the kind of community adapters that run in their
// own process and speak the adapter protocol (docs/adapters.md).
const CommunityKindRemote = "remote"

// CommunityRuntime registers installed community adapters in the provider
// registry (internal/providers/remote implements it), so an install, update
// or removal takes effect at once. nil means this build runs none, and
// manifests of kind "remote" are refused.
type CommunityRuntime interface {
	// ParseManifest validates a manifest of kind "remote" and returns its
	// registry key ("community:<id>") and the capabilities to list.
	ParseManifest(raw []byte) (key string, capabilities []string, err error)
	// Installed runs inside the transaction that installs such an adapter.
	Installed(ctx context.Context, tx pgx.Tx, adapterID uuid.UUID, raw []byte) error
	// Changed runs after an install, update or removal is committed.
	Changed(ctx context.Context)
}

// HTTPMappings checks and renders generic HTTP adapter mappings (the app
// wires internal/providers/httpgeneric; core code never imports an adapter).
type HTTPMappings interface {
	// Check validates a manifest of kind "http". Errors may name the field
	// they are about (FieldName: "url", "body", "headers", "webhook.auth"…).
	Check(raw []byte) error
	// Preview renders the request the mapping sends for a sample message,
	// with secrets shown as "••••".
	Preview(raw []byte) (AdapterHTTPPreview, error)
}

// AdapterHTTPPreview is the request a generic HTTP mapping would send for
// a sample message. Secrets are never shown.
type AdapterHTTPPreview struct {
	Method  string                     `json:"method"`
	URL     string                     `json:"url"`
	Headers []AdapterHTTPPreviewHeader `json:"headers"`
	Body    string                     `json:"body"`
}

// AdapterHTTPPreviewHeader is one header of a preview.
type AdapterHTTPPreviewHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// communityKey is the registry key of an installed community adapter (see
// providers.InstalledKey) and, for a remote adapter whose manifest the
// runtime accepts, the capabilities it declares.
func (s *Service) communityKey(kind, name string, manifest []byte) (string, []string, bool) {
	if kind == CommunityKindRemote && s.Community != nil {
		if key, caps, err := s.Community.ParseManifest(manifest); err == nil {
			return key, caps, true
		}
	}
	return providers.InstalledKey(kind, name, manifest), nil, false
}

// communityChanged re-registers the installed adapters after a change.
func (s *Service) communityChanged(ctx context.Context) {
	if s.Community != nil {
		s.Community.Changed(ctx)
	}
}

// communityKeyTaken returns the name of the installed adapter, other than
// except, already registered under key ("" when there is none).
func (s *Service) communityKeyTaken(ctx context.Context, q *store.Queries, key string, except uuid.UUID) (string, error) {
	all, err := q.CommunityAdapterList(ctx)
	if err != nil {
		return "", err
	}
	for _, a := range all {
		if a.ID == except {
			continue
		}
		if k, _, _ := s.communityKey(a.Kind, a.Name, a.Manifest); k == key {
			return a.Name, nil
		}
	}
	return "", nil
}

// mappingProblem turns a mapping validation error into a field error of
// the HTTP adapter form.
func mappingProblem(err error) error {
	var fe interface{ FieldName() string }
	field := ""
	if errors.As(err, &fe) {
		field = fe.FieldName()
	}
	msg := err.Error()
	if field != "" {
		msg = msg[min(len(field)+2, len(msg)):]
	}
	loc := "body.endpoint"
	paths := map[string]string{"webhook.events_path": "events", "webhook.type_path": "type", "webhook.recipient_path": "recipient", "webhook.message_id_path": "messageId"}
	switch {
	case field == "body":
		loc = "body.bodyTemplate"
	case field == "headers":
		loc = "body.headers"
	case field == "message_id_path":
		loc = "body.messageIdPath"
	case field == "webhook_mapping":
		loc = "body.webhookMapping"
	case paths[field] != "":
		loc = "body.webhookPaths." + paths[field]
	case field == "webhook.token_header" || field == "webhook.signature_header":
		loc = "body.webhookHeader"
	case strings.HasPrefix(field, "webhook"):
		loc = "body.webhookAuth"
	}
	return api.Invalid(loc, msg)
}

// NewAdapterHTTPPreview builds a preview from a rendered request (used by
// the app's wiring of HTTPMappings): headers in name order, the body
// indented as it was written.
func NewAdapterHTTPPreview(method, url string, header map[string][]string, body string) AdapterHTTPPreview {
	out := AdapterHTTPPreview{Method: method, URL: url, Headers: []AdapterHTTPPreviewHeader{}, Body: body}
	for _, k := range slices.Sorted(maps.Keys(header)) {
		for _, v := range header[k] {
			out.Headers = append(out.Headers, AdapterHTTPPreviewHeader{Name: k, Value: v})
		}
	}
	var b bytes.Buffer
	if json.Indent(&b, []byte(body), "", "  ") == nil {
		out.Body = b.String()
	}
	return out
}
