package remote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

// Kind is the installed_adapters.kind and manifest "kind" of adapters that
// run in their own process and speak this protocol.
const Kind = "remote"

// ProtocolVersion is the adapter protocol version this build speaks.
const ProtocolVersion = 1

// MaxBatch is the largest batch an adapter may declare.
const MaxBatch = 1000

var (
	idRe       = regexp.MustCompile(`^[a-z][a-z0-9-]{1,39}$`)
	fieldKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
	hostRe     = regexp.MustCompile(`^(\*\.)?[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*(:[0-9]{1,5})?$`)
)

// Manifest is a community adapter's manifest.json (kind "remote").
type Manifest struct {
	Protocol     int          `json:"protocol"`
	Kind         string       `json:"kind"`
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Maintainer   string       `json:"maintainer"`
	Description  string       `json:"description,omitempty"`
	Homepage     string       `json:"homepage,omitempty"`
	SourceURL    string       `json:"source_url,omitempty"`
	LastRelease  string       `json:"last_release,omitempty"`
	Latest       string       `json:"latest_version,omitempty"`
	Capabilities Capabilities `json:"capabilities"`
	Terms        Terms        `json:"terms"`
	Pricing      string       `json:"pricing"`
	Config       []Field      `json:"config"`
	Permissions  Permissions  `json:"permissions"`
}

// Capabilities are what an adapter declares it can do; ReputeCast calls the
// operations they enable and nothing else. In JSON the string-or-false
// values are false when absent.
type Capabilities struct {
	Batch            int
	Webhooks         bool
	RegisterWebhooks bool
	DomainSetup      string // "", "full", "dkim_only"
	SuppressionSync  string // "", "pull", "push", "two-way"
	QuotaDiscovery   string // "", "quota", "rate"
	Sandbox          string // "", "detection", "test token", "mode"
	DedicatedIPs     string
	RewritesLinks    bool
	AddsOpenPixel    bool
}

// Terms is what the provider's terms allow.
type Terms struct {
	BulkMarketing providers.BulkPolicy `json:"bulkMarketing"`
	Note          string               `json:"note,omitempty"`
	URL           string               `json:"url,omitempty"`
}

// Field is one input of the connection form.
type Field struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Secret   bool     `json:"secret,omitempty"`
	Required bool     `json:"required,omitempty"`
	Options  []string `json:"options,omitempty"`
	Help     string   `json:"help,omitempty"`
}

// Permissions are what the adapter process may reach.
type Permissions struct {
	// Network lists hosts: "api.example.com", "*.example.com", "host:port"
	// or "settings:<key>" for the host of a URL entered in the connection form.
	Network []string `json:"network,omitempty"`
}

// capabilityKeys are the keys the capabilities object may hold.
var capabilityKeys = []string{"batch", "webhooks", "registerWebhooks", "domainSetup", "suppressionSync", "quotaDiscovery", "sandbox", "dedicatedIps", "rewritesLinks", "addsOpenPixel"}

// UnmarshalJSON reads the capabilities object strictly: unknown keys and
// values of the wrong type are errors, so a typo never drops a capability.
func (c *Capabilities) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return errors.New("capabilities must be an object")
	}
	*c = Capabilities{}
	for k, v := range m {
		var err error
		switch k {
		case "batch":
			var n float64
			if err = json.Unmarshal(v, &n); err == nil && (n != float64(int(n)) || n < 1 || n > MaxBatch) {
				err = fmt.Errorf("must be a whole number from 1 to %d", MaxBatch)
			}
			c.Batch = int(n)
		case "webhooks":
			err = json.Unmarshal(v, &c.Webhooks)
		case "registerWebhooks":
			err = json.Unmarshal(v, &c.RegisterWebhooks)
		case "rewritesLinks":
			err = json.Unmarshal(v, &c.RewritesLinks)
		case "addsOpenPixel":
			err = json.Unmarshal(v, &c.AddsOpenPixel)
		case "domainSetup":
			c.DomainSetup, err = choice(v, "full", "dkim_only")
		case "suppressionSync":
			c.SuppressionSync, err = choice(v, "pull", "push", "two-way")
		case "quotaDiscovery":
			var on bool
			var s string
			switch {
			case json.Unmarshal(v, &on) == nil:
				if on {
					c.QuotaDiscovery = "quota"
				}
			case json.Unmarshal(v, &s) == nil && s == "rate":
				c.QuotaDiscovery = "rate"
			default:
				err = errors.New(`use true, "rate" or false`)
			}
		case "sandbox":
			c.Sandbox, err = choice(v, "detection", "test token", "mode")
		case "dedicatedIps":
			c.DedicatedIPs, err = text(v, 80)
		default:
			return fmt.Errorf("capabilities.%s is not a capability (use %s)", k, strings.Join(capabilityKeys, ", "))
		}
		if err != nil {
			return fmt.Errorf("capabilities.%s: %w", k, err)
		}
	}
	if c.Batch == 0 {
		c.Batch = 1
	}
	return nil
}

// MarshalJSON writes the capabilities object as the manifest spells it.
func (c Capabilities) MarshalJSON() ([]byte, error) {
	m := map[string]any{"batch": max(c.Batch, 1), "webhooks": c.Webhooks}
	if c.RegisterWebhooks {
		m["registerWebhooks"] = true
	}
	orFalse := func(k, v string) {
		if v == "" {
			m[k] = false
		} else {
			m[k] = v
		}
	}
	orFalse("domainSetup", c.DomainSetup)
	orFalse("suppressionSync", c.SuppressionSync)
	switch c.QuotaDiscovery {
	case "rate":
		m["quotaDiscovery"] = "rate"
	case "quota":
		m["quotaDiscovery"] = true
	default:
		m["quotaDiscovery"] = false
	}
	orFalse("sandbox", c.Sandbox)
	orFalse("dedicatedIps", c.DedicatedIPs)
	if c.RewritesLinks {
		m["rewritesLinks"] = true
	}
	if c.AddsOpenPixel {
		m["addsOpenPixel"] = true
	}
	return json.Marshal(m)
}

// choice reads false or one of the allowed strings ("" for false).
func choice(v json.RawMessage, allowed ...string) (string, error) {
	var on bool
	if json.Unmarshal(v, &on) == nil {
		if on {
			return "", fmt.Errorf("use false or one of %q", allowed)
		}
		return "", nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil || !slices.Contains(allowed, s) {
		return "", fmt.Errorf("use false or one of %q", allowed)
	}
	return s, nil
}

// text reads false or a short string.
func text(v json.RawMessage, maxLen int) (string, error) {
	var on bool
	if json.Unmarshal(v, &on) == nil && !on {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil || strings.TrimSpace(s) == "" || len(s) > maxLen {
		return "", fmt.Errorf("use false or a text of at most %d characters", maxLen)
	}
	return strings.TrimSpace(s), nil
}

// Pull reports whether ReputeCast pulls the provider's suppressions.
func (c Capabilities) Pull() bool { return c.SuppressionSync == "pull" || c.SuppressionSync == "two-way" }

// Push reports whether ReputeCast pushes the shared list to the provider.
func (c Capabilities) Push() bool { return c.SuppressionSync == "push" || c.SuppressionSync == "two-way" }

// Key is the registry key of the adapter: "community:<id>".
func (m Manifest) Key() string { return providers.RuntimePrefix + m.ID }

// ParseManifest reads and validates a manifest of kind "remote". Errors are
// sentences that can be shown to the person installing the adapter.
func ParseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&m); err != nil {
		var cerr *json.UnmarshalTypeError
		if errors.As(err, &cerr) {
			return m, fmt.Errorf("the manifest's %s has the wrong type", cerr.Field)
		}
		msg := err.Error()
		if strings.HasPrefix(msg, "capabilities") || strings.Contains(msg, "capabilities") {
			return m, errors.New("the manifest's " + strings.TrimPrefix(msg, "json: "))
		}
		return m, errors.New("the manifest is not valid JSON")
	}
	return m, m.Validate()
}

// Validate checks a manifest.
func (m *Manifest) Validate() error {
	m.ID, m.Name, m.Version, m.Maintainer = strings.TrimSpace(m.ID), strings.TrimSpace(m.Name), strings.TrimSpace(m.Version), strings.TrimSpace(m.Maintainer)
	switch {
	case m.Kind != Kind:
		return fmt.Errorf("the manifest's kind is %q; community adapters that run in their own process use %q", m.Kind, Kind)
	case m.Protocol != ProtocolVersion:
		if m.Protocol > ProtocolVersion {
			return fmt.Errorf("this adapter needs adapter protocol %d and this ReputeCast speaks %d; update ReputeCast first", m.Protocol, ProtocolVersion)
		}
		return fmt.Errorf("the manifest needs \"protocol\": %d", ProtocolVersion)
	case !idRe.MatchString(m.ID):
		return errors.New("the manifest's id must be 2 to 40 characters: lowercase letters, digits and dashes, starting with a letter")
	case m.Name == "" || len(m.Name) > 80:
		return errors.New("the manifest needs a name of at most 80 characters")
	case m.Version == "" || len(m.Version) > 40:
		return errors.New("the manifest needs a version of at most 40 characters")
	case m.Maintainer == "" || len(m.Maintainer) > 80:
		return errors.New("the manifest needs a maintainer of at most 80 characters")
	case len(m.Description) > 500:
		return errors.New("the manifest's description is longer than 500 characters")
	case strings.TrimSpace(m.Pricing) == "" || len(m.Pricing) > 80:
		return errors.New("the manifest needs the provider's pricing model, at most 80 characters (\"$0.80 per 1,000\")")
	}
	if m.LastRelease != "" {
		if _, err := time.Parse(time.RFC3339, m.LastRelease); err != nil {
			return errors.New("the manifest's last_release must be an RFC 3339 time")
		}
	}
	for _, u := range []string{m.Homepage, m.SourceURL, m.Terms.URL} {
		if u == "" {
			continue
		}
		if pu, err := url.Parse(u); err != nil || (pu.Scheme != "https" && pu.Scheme != "http") || pu.Host == "" {
			return fmt.Errorf("the manifest's link %q is not an http(s) URL", u)
		}
	}
	if m.Capabilities.Batch == 0 {
		m.Capabilities.Batch = 1
	}
	if m.Capabilities.RegisterWebhooks && !m.Capabilities.Webhooks {
		return errors.New("the manifest's capabilities.registerWebhooks needs capabilities.webhooks")
	}
	switch m.Terms.BulkMarketing {
	case providers.BulkAllowed, providers.BulkForbidden, providers.BulkYourPolicy:
	default:
		return errors.New("the manifest's terms.bulkMarketing must be allowed, forbidden or your_policy")
	}
	if len(m.Terms.Note) > 500 {
		return errors.New("the manifest's terms.note is longer than 500 characters")
	}
	if len(m.Config) > 30 {
		return errors.New("the manifest's config has more than 30 fields")
	}
	seen := map[string]bool{}
	for i, f := range m.Config {
		f.Key, f.Label = strings.TrimSpace(f.Key), strings.TrimSpace(f.Label)
		m.Config[i] = f
		switch {
		case !fieldKeyRe.MatchString(f.Key):
			return fmt.Errorf("config field %q: keys are lowercase letters, digits and underscores, starting with a letter", f.Key)
		case seen[f.Key]:
			return fmt.Errorf("config field %q is declared twice", f.Key)
		case f.Label == "" || len(f.Label) > 80:
			return fmt.Errorf("config field %q needs a label of at most 80 characters", f.Key)
		case len(f.Help) > 300:
			return fmt.Errorf("config field %q: help is longer than 300 characters", f.Key)
		case looksSecret(f.Key) && !f.Secret:
			return fmt.Errorf("config field %q looks like a secret: mark it \"secret\": true so it is sealed and never shown again", f.Key)
		case f.Secret && len(f.Options) > 0:
			return fmt.Errorf("config field %q can't be both secret and a list of options", f.Key)
		case len(f.Options) > 50:
			return fmt.Errorf("config field %q has more than 50 options", f.Key)
		}
		seen[f.Key] = true
	}
	if len(m.Permissions.Network) > 20 {
		return errors.New("the manifest's permissions.network lists more than 20 hosts")
	}
	for _, h := range m.Permissions.Network {
		if k, ok := strings.CutPrefix(h, "settings:"); ok {
			if !seen[k] {
				return fmt.Errorf("permissions.network %q names a setting the config doesn't declare", h)
			}
			continue
		}
		if !hostRe.MatchString(h) {
			return fmt.Errorf("permissions.network %q is not a host name (api.example.com, *.example.com or host:port)", h)
		}
	}
	return nil
}

// looksSecret mirrors the contract suite's rule for connection fields.
func looksSecret(key string) bool {
	return strings.Contains(key, "password") || strings.Contains(key, "token") || strings.Contains(key, "secret") || strings.HasSuffix(key, "api_key")
}

// ProviderCapabilities maps the declaration to the capability badges.
func (m Manifest) ProviderCapabilities() providers.Capabilities {
	c := m.Capabilities
	out := providers.Capabilities{
		Bulk:            m.Terms.BulkMarketing,
		DomainAPI:       c.DomainSetup,
		SuppressionSync: c.SuppressionSync != "",
		BatchSize:       max(c.Batch, 1),
		DedicatedIPs:    c.DedicatedIPs,
		Sandbox:         c.Sandbox,
		QuotaDiscovery:  c.QuotaDiscovery != "",
		RateDiscovery:   c.QuotaDiscovery == "rate",
		Pricing:         strings.TrimSpace(m.Pricing),
		RewritesLinks:   c.RewritesLinks,
		AddsOpenPixel:   c.AddsOpenPixel,
	}
	if c.Webhooks {
		out.Webhooks = "Webhooks"
		if c.RegisterWebhooks {
			out.Webhooks = "Registered"
		}
	}
	return out
}

// Labels are the declared capabilities as the Adapters tab lists them
// ("webhooks · suppression pull · batch 50").
func (m Manifest) Labels() []string {
	c := m.Capabilities
	out := []string{}
	if c.Webhooks {
		out = append(out, "webhooks")
	}
	switch c.SuppressionSync {
	case "pull":
		out = append(out, "suppression pull")
	case "push":
		out = append(out, "suppression push")
	case "two-way":
		out = append(out, "suppression sync")
	}
	if c.Batch > 1 {
		out = append(out, "batch "+strconv.Itoa(c.Batch))
	}
	switch c.DomainSetup {
	case "full":
		out = append(out, "domain setup via API")
	case "dkim_only":
		out = append(out, "DKIM via API")
	}
	switch c.QuotaDiscovery {
	case "quota":
		out = append(out, "quota discovery")
	case "rate":
		out = append(out, "quota & rate discovery")
	}
	return out
}

// Fields is the connection form.
func (m Manifest) Fields() []providers.Field {
	out := make([]providers.Field, 0, len(m.Config))
	for _, f := range m.Config {
		out = append(out, providers.Field{Key: f.Key, Label: f.Label, Secret: f.Secret, Required: f.Required, Options: slices.Clone(f.Options), Help: f.Help})
	}
	return out
}

// ProviderTerms is what the terms allow, per stream.
func (m Manifest) ProviderTerms() []providers.Terms {
	streams := []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}
	if m.Terms.BulkMarketing == providers.BulkForbidden {
		streams = []providers.Stream{providers.StreamAutomation, providers.StreamSystem}
	}
	note := m.Terms.Note
	if note == "" {
		switch m.Terms.BulkMarketing {
		case providers.BulkAllowed:
			note = "The adapter's manifest says the provider's terms allow bulk marketing to opted-in lists."
		case providers.BulkForbidden:
			note = "The adapter's manifest says the provider's terms forbid bulk marketing: use it for Automations and System mail."
		default:
			note = "Self-hosted: your own policy applies."
		}
	}
	return []providers.Terms{{Bulk: m.Terms.BulkMarketing, Note: note, Streams: streams}}
}

// Descriptor describes the adapter for the catalog; newFn builds a provider.
func (m Manifest) Descriptor(newFn func(providers.Config) (providers.Provider, error)) providers.Descriptor {
	d := providers.Descriptor{
		Adapter:      m.Key(),
		Name:         m.Name,
		Kind:         "community",
		Capabilities: m.ProviderCapabilities(),
		Version:      m.Version,
		Maintainer:   m.Maintainer,
		Fields:       m.Fields(),
		Terms:        m.ProviderTerms(),
		Events:       "None",
		DomainSetup:  "Manual DNS",
		New:          newFn,
	}
	if m.Capabilities.Webhooks {
		d.Events = "Webhooks"
	}
	switch m.Capabilities.DomainSetup {
	case "full":
		d.DomainSetup = "API"
	case "dkim_only":
		d.DomainSetup = "DKIM via API"
	}
	switch m.Capabilities.SuppressionSync {
	case "pull":
		d.SuppressionMode = "Pull only"
	case "push":
		d.SuppressionMode = "Push only"
	case "two-way":
		d.SuppressionMode = "Two-way"
	default:
		d.SuppressionMode = "Local only"
	}
	return d
}

// declared reports whether the manifest's config declares key, and whether
// it is a secret.
func (m Manifest) declared(key string) (bool, bool) {
	for _, f := range m.Config {
		if f.Key == key {
			return true, f.Secret
		}
	}
	return false, false
}
