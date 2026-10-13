package httpgeneric

import (
	"encoding/json"
	"slices"

	"github.com/hostrepute/reputecast/internal/providers"
)

// This file is how installed HTTP adapters (installed_adapters rows of kind
// "http": a mapping saved from the Adapters tab or a community manifest)
// join the provider registry. The community adapter runtime
// (internal/providers/remote) registers each one under its key,
// providers.CommunityKey(name), with CommunityDescriptor; a provider
// connected through it is opened with FromManifest and the stored manifest.

// FieldName is the mapping field an error is about ("body", "headers",
// "webhook.auth"), for forms that show errors next to the field.
func (e *FieldError) FieldName() string { return e.Field }

// CommunityDescriptor describes an installed HTTP adapter for the catalog
// and the connect flow: the adapter's own connection fields plus the
// mapping's, the capabilities and terms its manifest gives, and a
// constructor that opens connections from the stored manifest.
func CommunityDescriptor(key, name, version, maintainer string, raw []byte) (providers.Descriptor, error) {
	m, err := ParseManifest(raw)
	if err != nil {
		return providers.Descriptor{}, err
	}
	probe, err := FromManifest(raw, providers.Config{})
	if err != nil {
		return providers.Descriptor{}, err
	}
	p := provider(probe)
	manifest := slices.Clone(raw)
	d := Descriptor()
	d.Adapter, d.Name, d.Kind, d.Version, d.Maintainer = key, name, "community", version, maintainer
	d.Capabilities = probe.Capabilities()
	if d.Capabilities.Webhooks == "" {
		d.Events = "None"
	}
	terms := p.Terms()
	streams := []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}
	if terms.Bulk == providers.BulkForbidden || terms.Bulk == providers.BulkBroadcastStreams {
		streams = []providers.Stream{providers.StreamAutomation, providers.StreamSystem}
	}
	d.Terms = []providers.Terms{{Bulk: terms.Bulk, Note: terms.Note, Streams: streams}}
	for _, f := range m.HTTP.Fields {
		if !slices.ContainsFunc(d.Fields, func(x providers.Field) bool { return x.Key == f.Key }) {
			d.Fields = append(d.Fields, f)
		}
	}
	d.New = func(cfg providers.Config) (providers.Provider, error) { return FromManifest(manifest, cfg) }
	return d, nil
}

// PreviewManifest renders the request an HTTP adapter's manifest would send
// for the sample message, with every secret shown as "••••" (the Adapters
// tab's "Test with a sample message").
func PreviewManifest(raw []byte) (Rendered, error) {
	m, err := ParseManifest(raw)
	if err != nil {
		return Rendered{}, err
	}
	creds := map[string]string{"api_key": "x", "webhook_signing_key": "x"}
	for _, f := range m.HTTP.Fields {
		if f.Secret {
			creds[f.Key] = "x"
		}
	}
	b, _ := json.Marshal(creds)
	pr, err := FromManifest(raw, providers.Config{Credentials: b})
	if err != nil {
		return Rendered{}, err
	}
	return provider(pr).Preview(sampleMessage())
}

// provider unwraps the adapter FromManifest returns.
func provider(p providers.Provider) *Provider {
	if c, ok := p.(*Checked); ok {
		return c.Provider
	}
	return p.(*Provider)
}
