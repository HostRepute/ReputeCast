package manage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// Links on the Adapters tab.
const (
	SDKURL      = "https://github.com/hostrepute/reputecast/blob/main/docs/adapters.md#community-adapters"
	TemplateURL = "https://github.com/hostrepute/reputecast/tree/main/sdk/adapter-template"
	RegistryURL = "https://github.com/hostrepute/reputecast-adapters"
)

// adapterOrder is the catalogue order of built-in adapters.
var adapterOrder = []string{"ses", "postmark", "postal", "mailgun", "sendgrid", "resend", "sparkpost", "mailjet", "brevo", "azureacs", "smtp"}

// AdapterBuiltin is one row of "Built-in adapters".
type AdapterBuiltin struct {
	Adapter         string `json:"adapter"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Maintainer      string `json:"maintainer"`
	Events          string `json:"events"`
	DomainSetup     string `json:"domainSetup"`
	SuppressionSync string `json:"suppressionSync"`
	Connected       int    `json:"connected"`
}

// AdapterCommunity is one installed community adapter.
type AdapterCommunity struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Kind         string     `json:"kind" enum:"http,wasm,remote"`
	Version      string     `json:"version"`
	Maintainer   string     `json:"maintainer"`
	Capabilities []string   `json:"capabilities"`
	Latest       string     `json:"latest,omitempty"`
	LastRelease  *time.Time `json:"lastRelease,omitempty"`
	// Update: up_to_date, update_available, unmaintained (no release for a year).
	Update    string    `json:"update" enum:"up_to_date,update_available,unmaintained"`
	Connected int       `json:"connected"`
	SourceURL string    `json:"sourceUrl,omitempty"`
	Installed time.Time `json:"installedAt"`
}

// AdaptersOverview is the Adapters tab.
type AdaptersOverview struct {
	Version   string             `json:"version"`
	Latest    string             `json:"latest,omitempty"`
	UpToDate  bool               `json:"upToDate"`
	Builtin   []AdapterBuiltin   `json:"builtin"`
	Community []AdapterCommunity `json:"community"`
	// HTTPAdapter is true when this build includes the generic HTTP adapter.
	HTTPAdapter bool   `json:"httpAdapter"`
	SDKURL      string `json:"sdkUrl"`
	TemplateURL string `json:"templateUrl"`
	RegistryURL string `json:"registryUrl"`
	// CanInstall is true for instance admins (adapters apply to every workspace).
	CanInstall bool `json:"canInstall"`
}

// Adapters returns the Adapters tab.
func (s *Service) Adapters(ctx context.Context, ws *store.Workspace) (AdaptersOverview, error) {
	l := i18n.LangFrom(ctx)
	cur := strings.TrimPrefix(version.Get().Version, "v")
	out := AdaptersOverview{Version: cur, Builtin: []AdapterBuiltin{}, Community: []AdapterCommunity{}, SDKURL: SDKURL, TemplateURL: TemplateURL, RegistryURL: RegistryURL}
	if st, err := s.Q.GetInstanceSettings(ctx); err == nil && len(st.LatestRelease) > 0 {
		var rel struct {
			Tag string `json:"tag"`
		}
		_ = json.Unmarshal(st.LatestRelease, &rel)
		out.Latest = strings.TrimPrefix(rel.Tag, "v")
	}
	out.UpToDate = out.Latest == "" || out.Latest == cur || cur == "dev"
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	connected := map[string]int{}
	for _, p := range provs {
		connected[p.Adapter]++
	}
	cat := providers.Catalog()
	slices.SortFunc(cat, func(a, b providers.Descriptor) int {
		ia, ib := slices.Index(adapterOrder, a.Adapter), slices.Index(adapterOrder, b.Adapter)
		if ia < 0 {
			ia = len(adapterOrder)
		}
		if ib < 0 {
			ib = len(adapterOrder)
		}
		if ia != ib {
			return ia - ib
		}
		return strings.Compare(a.Name, b.Name)
	})
	for _, d := range cat {
		if d.Kind == "community" {
			continue
		}
		if d.Adapter == "httpgeneric" {
			out.HTTPAdapter = true
			continue
		}
		v := strings.TrimPrefix(d.Version, "v")
		words := d.In(l)
		out.Builtin = append(out.Builtin, AdapterBuiltin{Adapter: d.Adapter, Name: d.Name, Version: v, Maintainer: d.Maintainer,
			Events: words.Events, DomainSetup: words.DomainSetup, SuppressionSync: words.SuppressionMode, Connected: connected[d.Adapter]})
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, name, kind, version, maintainer, source_url, manifest, latest_version, installed_at FROM installed_adapters ORDER BY installed_at, name`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var c AdapterCommunity
		var id uuid.UUID
		var manifest json.RawMessage
		var latest *string
		if err := rows.Scan(&id, &c.Name, &c.Kind, &c.Version, &c.Maintainer, &c.SourceURL, &manifest, &latest, &c.Installed); err != nil {
			return out, err
		}
		c.ID = id.String()
		var m struct {
			Capabilities json.RawMessage `json:"capabilities"`
			LastRelease  time.Time       `json:"last_release"`
		}
		_ = json.Unmarshal(manifest, &m)
		key, labels, remote := s.communityKey(l, c.Kind, c.Name, manifest)
		if remote {
			c.Capabilities = labels
		} else {
			_ = json.Unmarshal(m.Capabilities, &c.Capabilities)
		}
		if c.Capabilities == nil {
			c.Capabilities = []string{}
		}
		if !m.LastRelease.IsZero() {
			lr := m.LastRelease
			c.LastRelease = &lr
		}
		c.Update = "up_to_date"
		if latest != nil && *latest != "" && *latest != c.Version {
			c.Latest, c.Update = *latest, "update_available"
		} else if c.LastRelease != nil && s.now().Sub(*c.LastRelease) > 365*24*time.Hour {
			c.Update = "unmaintained"
		}
		c.Connected = connected[key]
		out.Community = append(out.Community, c)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if p := api.PrincipalFrom(ctx); p != nil && p.IsInstanceAdmin {
		out.CanInstall = true
	}
	return out, nil
}

func instanceAdmin(ctx context.Context) error {
	if p := api.PrincipalFrom(ctx); p == nil || !p.IsInstanceAdmin {
		return api.Err(http.StatusForbidden, api.CodePermissionDenied, mAdminOnly.Text(i18n.LangFrom(ctx), nil))
	}
	return nil
}

// AdapterManifest is a community adapter's manifest.json. Capabilities is
// a list of labels for http and wasm adapters and an object for remote ones
// (docs/adapters.md).
type AdapterManifest struct {
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Maintainer   string          `json:"maintainer"`
	Kind         string          `json:"kind"`
	Capabilities json.RawMessage `json:"capabilities"`
	SourceURL    string          `json:"source_url"`
	Latest       string          `json:"latest_version"`
	LastRelease  string          `json:"last_release"`
	Terms        json.RawMessage `json:"terms"`
	HTTP         json.RawMessage `json:"http"`
}

// AdapterInstallInput installs a community adapter from its manifest URL.
type AdapterInstallInput struct {
	ManifestURL string `json:"manifestUrl" format:"uri" maxLength:"2048" doc:"HTTPS URL of the adapter's manifest.json"`
}

// InstallAdapter fetches a manifest and installs the adapter.
func (s *Service) InstallAdapter(ctx context.Context, ws *store.Workspace, in AdapterInstallInput) (AdapterCommunity, error) {
	if err := instanceAdmin(ctx); err != nil {
		return AdapterCommunity{}, err
	}
	l := i18n.LangFrom(ctx)
	m, raw, err := fetchManifest(ctx, in.ManifestURL)
	if err != nil {
		return AdapterCommunity{}, api.Invalid("body.manifestUrl", problemIn(l, err))
	}
	switch {
	case m.Kind == CommunityKindRemote && s.Community == nil:
		return AdapterCommunity{}, api.Invalid("body.manifestUrl", mNoRemote.Text(l, nil))
	case m.Kind == CommunityKindRemote:
		if _, _, err := s.Community.ParseManifest(raw, l); err != nil {
			return AdapterCommunity{}, api.Invalid("body.manifestUrl", mManifestUnusable.Text(l, i18n.Args{"error": err.Error()}))
		}
	case m.Kind == "http" && s.HTTPMappings != nil:
		if err := s.HTTPMappings.Check(raw); err != nil {
			return AdapterCommunity{}, api.Invalid("body.manifestUrl", mMappingUnusable.Text(l, i18n.Args{"error": mappingError(l, err)}))
		}
	}
	if m.SourceURL == "" {
		m.SourceURL = in.ManifestURL
	}
	key, _, _ := s.communityKey(l, m.Kind, m.Name, raw)
	id := newID()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if other, err := s.communityKeyTaken(ctx, store.New(tx), key, uuid.Nil); err != nil {
			return err
		} else if other != "" {
			return api.Conflict(mKeyTaken.Text(l, i18n.Args{"other": other, "key": key}))
		}
		if _, err := tx.Exec(ctx, `INSERT INTO installed_adapters (id, name, kind, version, maintainer, source_url, manifest, latest_version)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, m.Name, m.Kind, m.Version, m.Maintainer, in.ManifestURL, raw, nullable(m.Latest)); err != nil {
			if strings.Contains(err.Error(), "installed_adapters_name_key") {
				return api.Conflict(mInstalled.Text(l, i18n.Args{"name": m.Name}))
			}
			return err
		}
		if m.Kind == CommunityKindRemote {
			if err := s.Community.Installed(ctx, tx, id, raw); err != nil {
				return err
			}
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &wsID, Action: "adapter.install", TargetType: "adapter", TargetID: id.String(), TargetName: m.Name,
			After: map[string]any{"version": m.Version, "maintainer": m.Maintainer, "source": in.ManifestURL, "kind": m.Kind}})
	})
	if err != nil {
		return AdapterCommunity{}, err
	}
	s.communityChanged(ctx)
	return s.community(ctx, ws, id)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) community(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AdapterCommunity, error) {
	ov, err := s.Adapters(ctx, ws)
	if err != nil {
		return AdapterCommunity{}, err
	}
	for _, c := range ov.Community {
		if c.ID == id.String() {
			return c, nil
		}
	}
	return AdapterCommunity{}, api.NotFound("Adapter")
}

// UpdateAdapter re-reads an adapter's manifest and installs its new version.
func (s *Service) UpdateAdapter(ctx context.Context, ws *store.Workspace, id string) (AdapterCommunity, error) {
	if err := instanceAdmin(ctx); err != nil {
		return AdapterCommunity{}, err
	}
	l := i18n.LangFrom(ctx)
	aid, err := uuid.Parse(id)
	if err != nil {
		return AdapterCommunity{}, api.NotFound("Adapter")
	}
	var name, kind, ver, src string
	var current json.RawMessage
	var latest *string
	if err := s.Pool.QueryRow(ctx, `SELECT name, kind, version, source_url, manifest, latest_version FROM installed_adapters WHERE id = $1`, aid).Scan(&name, &kind, &ver, &src, &current, &latest); err != nil {
		return AdapterCommunity{}, api.MapError(err, "Adapter")
	}
	var m AdapterManifest
	var raw json.RawMessage
	if isDemoManifest(current) {
		// The demo's sample adapters have no manifest to download: move to
		// the advertised version.
		if latest == nil || *latest == "" {
			return AdapterCommunity{}, api.Conflict(mUpToDate.Text(l, i18n.Args{"name": name}))
		}
		_ = json.Unmarshal(current, &m)
		m.Name, m.Version = name, *latest
		raw = current
	} else {
		m, raw, err = fetchManifest(ctx, src)
		if err != nil {
			return AdapterCommunity{}, api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, mUnreadable.Text(l, i18n.Args{"url": src, "error": problemIn(l, err)}))
		}
	}
	if m.Name != name {
		return AdapterCommunity{}, api.Conflict(mRenamed.Text(l, i18n.Args{"url": src, "name": m.Name, "old": name}))
	}
	if kind == CommunityKindRemote || m.Kind == CommunityKindRemote {
		if m.Kind != kind {
			return AdapterCommunity{}, api.Conflict(mKindChanged.Text(l, i18n.Args{"url": src, "kind": m.Kind, "old": kind}))
		}
		if s.Community == nil {
			return AdapterCommunity{}, api.Conflict(mNoRemote.Text(l, nil))
		}
		newKey, _, err := s.Community.ParseManifest(raw, l)
		if err != nil {
			return AdapterCommunity{}, api.Err(http.StatusUnprocessableEntity, api.CodeValidation, mNewUnusable.Text(l, i18n.Args{"url": src, "error": err.Error()}))
		}
		if oldKey, _, _ := s.communityKey(l, kind, name, current); newKey != oldKey {
			return AdapterCommunity{}, api.Conflict(mKeyChanged.Text(l, i18n.Args{"url": src, "key": newKey, "old": oldKey}))
		}
	}
	if kind == "http" && s.HTTPMappings != nil && !isDemoManifest(current) {
		if err := s.HTTPMappings.Check(raw); err != nil {
			return AdapterCommunity{}, api.Err(http.StatusUnprocessableEntity, api.CodeValidation, mNewMappingUnusable.Text(l, i18n.Args{"url": src, "error": mappingError(l, err)}))
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE installed_adapters SET version = $2, maintainer = $3, manifest = $4, latest_version = NULL WHERE id = $1`, aid, m.Version, m.Maintainer, raw); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &wsID, Action: "adapter.update", TargetType: "adapter", TargetID: aid.String(), TargetName: name,
			Before: map[string]any{"version": ver}, After: map[string]any{"version": m.Version}})
	})
	if err != nil {
		return AdapterCommunity{}, err
	}
	s.communityChanged(ctx)
	return s.community(ctx, ws, aid)
}

// RemoveAdapter uninstalls a community adapter no provider uses.
func (s *Service) RemoveAdapter(ctx context.Context, ws *store.Workspace, id string) error {
	if err := instanceAdmin(ctx); err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	aid, err := uuid.Parse(id)
	if err != nil {
		return api.NotFound("Adapter")
	}
	var name, kind string
	var manifest json.RawMessage
	if err := s.Pool.QueryRow(ctx, `SELECT name, kind, manifest FROM installed_adapters WHERE id = $1`, aid).Scan(&name, &kind, &manifest); err != nil {
		return api.MapError(err, "Adapter")
	}
	key, _, _ := s.communityKey(l, kind, name, manifest)
	var used int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM providers WHERE adapter = $1 AND disconnected_at IS NULL`, key).Scan(&used); err != nil {
		return err
	}
	if used > 0 {
		return api.Conflict(mInUse.Text(l, i18n.Args{"count": used, "name": name}))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM installed_adapters WHERE id = $1`, aid); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &wsID, Action: "adapter.remove", TargetType: "adapter", TargetID: aid.String(), TargetName: name})
	})
	if err != nil {
		return err
	}
	s.communityChanged(ctx)
	return nil
}

// AdapterHTTPMapping is the generic HTTP adapter form: the send request,
// where the response carries the message id, and how webhooks are
// authenticated and read (docs/adapters.md › The generic HTTP adapter).
type AdapterHTTPMapping struct {
	Endpoint string `json:"endpoint" minLength:"10" maxLength:"2048" doc:"Method and URL, e.g. POST https://api.example-esp.com/v1/messages"`
	// Headers are request header templates, usually the authentication:
	// Authorization: Bearer {{ secrets.api_key }}.
	Headers        []AdapterHTTPHeader `json:"headers,omitempty" maxItems:"20"`
	BodyTemplate   string              `json:"bodyTemplate" maxLength:"20000"`
	MessageIDPath  string              `json:"messageIdPath" maxLength:"200" doc:"JSONPath of the message id in the response, e.g. $.data.id"`
	WebhookMapping string              `json:"webhookMapping" enum:"status,jsonpath"`
	// WebhookAuth is how a webhook request proves it comes from the provider.
	WebhookAuth string `json:"webhookAuth,omitempty" enum:"hmac,standard,basic,header,none" doc:"hmac (default): a signature header; standard: Standard Webhooks; basic: basic auth; header: the secret in a header; none: only the webhook URL"`
	// WebhookHeader names the signature header (hmac, default X-Signature) or
	// the header carrying the secret (header).
	WebhookHeader string `json:"webhookHeader,omitempty" maxLength:"100"`
	// WebhookPaths locate the events of a custom JSONPath map.
	WebhookPaths *AdapterHTTPWebhookPaths `json:"webhookPaths,omitempty"`
}

// AdapterHTTPWebhookPaths are the JSONPaths a custom webhook map
// (webhookMapping "jsonpath") reads: where the events are in a request and,
// inside one event, its type, recipient and message id.
type AdapterHTTPWebhookPaths struct {
	Events    string `json:"events,omitempty" maxLength:"200" doc:"The events in one request, e.g. $.events[*] (default $: one event or an array of events)"`
	Type      string `json:"type,omitempty" maxLength:"200" doc:"The event type, e.g. $.event"`
	Recipient string `json:"recipient,omitempty" maxLength:"200" doc:"The recipient's address, e.g. $.email"`
	MessageID string `json:"messageId,omitempty" maxLength:"200" doc:"The provider's message id, e.g. $.message_id"`
}

// AdapterHTTPHeader is one request header template.
type AdapterHTTPHeader struct {
	Name  string `json:"name" minLength:"1" maxLength:"100"`
	Value string `json:"value" maxLength:"2000"`
}

// AdapterHTTPInput saves a generic HTTP adapter mapping under a name.
type AdapterHTTPInput struct {
	Name string `json:"name" minLength:"2" maxLength:"60"`
	AdapterHTTPMapping
}

// httpManifest builds the manifest (kind "http") of a mapping.
func httpManifest(l i18n.Lang, name string, in AdapterHTTPMapping) ([]byte, error) {
	method, target, err := parseEndpoint(in.Endpoint)
	if err != nil {
		return nil, api.Invalid("body.endpoint", problemIn(l, err))
	}
	mapping := map[string]any{"method": method, "url": target, "body": in.BodyTemplate, "message_id_path": in.MessageIDPath, "webhook_mapping": in.WebhookMapping}
	if len(in.Headers) > 0 {
		headers := map[string]string{}
		for _, h := range in.Headers {
			n := strings.TrimSpace(h.Name)
			if _, dup := headers[n]; dup {
				return nil, api.Invalid("body.headers", mHeaderTwice.Text(l, i18n.Args{"name": n}))
			}
			headers[n] = strings.TrimSpace(h.Value)
		}
		mapping["headers"] = headers
	}
	hook := map[string]any{}
	if in.WebhookAuth != "" {
		hook["auth"] = in.WebhookAuth
	}
	if h := strings.TrimSpace(in.WebhookHeader); h != "" {
		switch in.WebhookAuth {
		case "", "hmac":
			hook["signature_header"] = h
		case "header":
			hook["token_header"] = h
		}
	}
	if p := in.WebhookPaths; p != nil {
		for k, v := range map[string]string{"events_path": p.Events, "type_path": p.Type, "recipient_path": p.Recipient, "message_id_path": p.MessageID} {
			if v = strings.TrimSpace(v); v != "" {
				hook[k] = v
			}
		}
	}
	if len(hook) > 0 {
		mapping["webhook"] = hook
	}
	m := map[string]any{"kind": "http", "capabilities": []string{"webhooks"}, "http": mapping}
	if name != "" {
		m["name"] = name
	}
	return json.Marshal(m)
}

// checkMapping validates a mapping with the generic HTTP adapter when the
// build has it, else checks that the body template is JSON.
func (s *Service) checkMapping(l i18n.Lang, in AdapterHTTPMapping, manifest []byte) error {
	if s.HTTPMappings == nil {
		if in.BodyTemplate != "" && !json.Valid([]byte(placeholderSafe(in.BodyTemplate))) {
			return api.Invalid("body.bodyTemplate", mBodyNotJSON.Text(l, nil))
		}
		return nil
	}
	if err := s.HTTPMappings.Check(manifest); err != nil {
		return mappingProblem(l, err)
	}
	return nil
}

// SaveHTTPAdapter checks a generic HTTP adapter mapping and stores it as an
// installed adapter (kind "http", registered as community:<name>).
func (s *Service) SaveHTTPAdapter(ctx context.Context, ws *store.Workspace, in AdapterHTTPInput) (AdapterCommunity, error) {
	if err := instanceAdmin(ctx); err != nil {
		return AdapterCommunity{}, err
	}
	l := i18n.LangFrom(ctx)
	in.Name = strings.TrimSpace(in.Name)
	manifest, err := httpManifest(l, in.Name, in.AdapterHTTPMapping)
	if err != nil {
		return AdapterCommunity{}, err
	}
	if err := s.checkMapping(l, in.AdapterHTTPMapping, manifest); err != nil {
		return AdapterCommunity{}, err
	}
	method, target, _ := parseEndpoint(in.Endpoint)
	key := providers.CommunityKey(in.Name)
	headerNames := []string{}
	for _, h := range in.Headers {
		headerNames = append(headerNames, strings.TrimSpace(h.Name))
	}
	id := newID()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if other, err := s.communityKeyTaken(ctx, store.New(tx), key, uuid.Nil); err != nil {
			return err
		} else if other != "" {
			return api.Invalid("body.name", mNameTaken.Text(l, i18n.Args{"other": other, "key": key}))
		}
		if _, err := tx.Exec(ctx, `INSERT INTO installed_adapters (id, name, kind, version, maintainer, source_url, manifest)
			VALUES ($1, $2, 'http', '0.1.0', $3, '', $4)`, id, in.Name, "local · "+principalName(ctx), manifest); err != nil {
			if strings.Contains(err.Error(), "installed_adapters_name_key") {
				return api.Conflict(mNameInstalled.Text(l, i18n.Args{"name": in.Name}))
			}
			return err
		}
		wsID := ws.ID
		// Header values are templates that may hold a pasted credential:
		// only their names are audited.
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &wsID, Action: "adapter.http_save", TargetType: "adapter", TargetID: id.String(), TargetName: in.Name,
			After: map[string]any{"method": method, "url": target, "headers": headerNames, "webhook_mapping": in.WebhookMapping, "webhook_auth": in.WebhookAuth, "registry_key": key}})
	})
	if err != nil {
		return AdapterCommunity{}, err
	}
	s.communityChanged(ctx)
	return s.community(ctx, ws, id)
}

// PreviewHTTPAdapter renders the request a mapping would send for a sample
// message, with secrets shown as "••••" ("Test with a sample message").
func (s *Service) PreviewHTTPAdapter(ctx context.Context, in AdapterHTTPMapping) (AdapterHTTPPreview, error) {
	if err := instanceAdmin(ctx); err != nil {
		return AdapterHTTPPreview{}, err
	}
	l := i18n.LangFrom(ctx)
	if s.HTTPMappings == nil {
		return AdapterHTTPPreview{}, api.Err(http.StatusNotImplemented, api.CodeUnavailable, mNoHTTP.Text(l, nil))
	}
	manifest, err := httpManifest(l, "", in)
	if err != nil {
		return AdapterHTTPPreview{}, err
	}
	if err := s.checkMapping(l, in, manifest); err != nil {
		return AdapterHTTPPreview{}, err
	}
	p, err := s.HTTPMappings.Preview(manifest)
	if err != nil {
		return AdapterHTTPPreview{}, mappingProblem(l, err)
	}
	return p, nil
}

func principalName(ctx context.Context) string {
	if p := api.PrincipalFrom(ctx); p != nil {
		return p.Name
	}
	return "admin"
}

// placeholderSafe fills {{ x }} placeholders so a template can be checked as
// JSON: inside a string with a letter, elsewhere with null.
func placeholderSafe(t string) string {
	var b strings.Builder
	inString, escaped := false, false
	for i := 0; i < len(t); i++ {
		if !inString || !escaped {
			if strings.HasPrefix(t[i:], "{{") {
				if j := strings.Index(t[i:], "}}"); j >= 0 {
					if inString {
						b.WriteString("x")
					} else {
						b.WriteString("null")
					}
					i += j + 1
					continue
				}
			}
		}
		c := t[i]
		b.WriteByte(c)
		switch {
		case escaped:
			escaped = false
		case c == '\\' && inString:
			escaped = true
		case c == '"':
			inString = !inString
		}
	}
	return b.String()
}

// isDemoManifest reports a sample adapter of the demo data ("demo": true).
func isDemoManifest(raw json.RawMessage) bool {
	var m struct {
		Demo bool `json:"demo"`
	}
	_ = json.Unmarshal(raw, &m)
	return m.Demo
}

func parseEndpoint(s string) (string, string, error) {
	method, rest, ok := strings.Cut(strings.TrimSpace(s), " ")
	if !ok {
		method, rest = "POST", s
	}
	method = strings.ToUpper(method)
	if !slices.Contains([]string{"POST", "PUT", "PATCH"}, method) {
		return "", "", problem{m: mEndpointMethod}
	}
	u, err := url.Parse(strings.TrimSpace(rest))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", "", problem{m: mEndpointURL}
	}
	return method, u.String(), nil
}

// fetchManifest downloads a manifest over HTTPS from a public address.
func fetchManifest(ctx context.Context, raw string) (AdapterManifest, json.RawMessage, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return AdapterManifest{}, nil, problem{m: mManifestURL}
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{DialContext: publicDialer(), TLSHandshakeTimeout: 5 * time.Second},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 3 || req.URL.Scheme != "https" {
				return errors.New("too many or insecure redirects")
			}
			return nil
		}}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, http.MethodGet, u.String(), nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return AdapterManifest{}, nil, problem{m: mDownload, args: i18n.Args{"error": err.Error()}}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return AdapterManifest{}, nil, problem{m: mStatus, args: i18n.Args{"status": strconv.Itoa(resp.StatusCode)}}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return AdapterManifest{}, nil, err
	}
	var m AdapterManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return AdapterManifest{}, nil, problem{m: mNotJSON}
	}
	if m.Name == "" || m.Version == "" || (m.Kind != "http" && m.Kind != "wasm" && m.Kind != CommunityKindRemote) {
		return AdapterManifest{}, nil, problem{m: mIncomplete}
	}
	// The Adapters tab shows "community · <maintainer>".
	m.Maintainer = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(m.Maintainer), "community · "))
	if m.Maintainer == "" || m.Maintainer == "community" {
		m.Maintainer = "unknown"
	}
	return m, body, nil
}

// publicDialer refuses private, loopback and link-local addresses, so a
// manifest URL can't be used to reach the instance's internal network.
func publicDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 5 * time.Second, Control: func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip, err := netip.ParseAddr(host)
		if err != nil {
			return err
		}
		if !PublicIP(ip) {
			return fmt.Errorf("%s is not a public address", ip)
		}
		return nil
	}}
	return d.DialContext
}

// PublicIP reports whether ip is a public unicast address.
func PublicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	for _, p := range []string{"100.64.0.0/10", "192.0.0.0/24", "198.18.0.0/15", "240.0.0.0/4", "64:ff9b::/96", "fc00::/7"} {
		if netip.MustParsePrefix(p).Contains(ip) {
			return false
		}
	}
	return true
}
