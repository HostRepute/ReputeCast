package providers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Text for people. An adapter writes what people read about its provider
// (field labels and hints, terms, pricing, the account's standing, tracking
// details, provisioning rows, a connection test's findings) in English, in
// the fields the contract has always had. Beside each one it may keep the
// message the English comes from, as an i18n.Text (LabelText, NoteText,
// AccountText…), from the adapter's own catalog in its package's
// messages/<locale>.json. The registry's catalogs are global, so core code
// reads the text in the reader's language without importing the adapter:
//
//	label := providers.TextIn(f.Label, f.LabelText, i18n.LangFrom(ctx))
//	d = d.In(i18n.LangFrom(ctx)) // a descriptor with every text translated
//
// A text without a message, a message the catalog no longer has, and every
// text of a community adapter read in the English they were written in.
// Core code that stores such a text for later readers (a setup's state, a
// provider's health) stores the message beside the English.
//
// A built-in adapter's descriptor names its catalog (WithMessages), which
// fills the descriptor's texts from well-known keys, the English (where the
// descriptor leaves it empty) and the message:
//
//	field.<key>.label, field.<key>.help   a connection field's label and hint
//	terms.<n>.variant, terms.<n>.note     the nth Terms (from 0)
//	pricing, dedicatedIps                 Capabilities.Pricing, .DedicatedIPs
//	events, domainSetup, suppressionMode  the Adapters tab's three words
//
// so the catalog is where an adapter's English is written, once.
//
// The contract suite (contracttest) checks that every message says what its
// English says, and that a built-in adapter leaves no text without one.

// TextIn returns a text in the reader's language: the message t when it
// exists, else english. English readers get english exactly as it was
// written, so API clients and agents read what they always did.
func TextIn(english string, t i18n.Text, l i18n.Lang) string {
	if t.IsZero() || l.Locale == i18n.English || l.Locale == "" {
		return english
	}
	if s := t.In(l); s != "" {
		return s
	}
	return english
}

// ErrorIn writes an error for a reader: err.Error(), with the message of a
// CallError the adapter worded itself in the reader's language.
func ErrorIn(err error, l i18n.Lang) string {
	if err == nil {
		return ""
	}
	var ce *CallError
	if !errors.As(err, &ce) || ce.Text.IsZero() || l.Locale == i18n.English || l.Locale == "" {
		return err.Error()
	}
	c := *ce
	c.Message = TextIn(ce.Message, ce.Text, l)
	return strings.Replace(err.Error(), ce.Error(), c.Error(), 1)
}

// MessageIn is the message of a failed call for a reader: a CallError's
// message (in the reader's language when the adapter kept its message),
// else err.Error().
func MessageIn(err error, l i18n.Lang) string {
	var ce *CallError
	if errors.As(err, &ce) && ce.Message != "" {
		return TextIn(ce.Message, ce.Text, l)
	}
	return err.Error()
}

// fill sets a text from the catalog's message for key, when the catalog
// has it: the English where it is empty, and the message unless one is set.
func fill(c *i18n.Catalog, english *string, t *i18n.Text, key string) {
	if c == nil {
		return
	}
	m, err := c.Lookup(key)
	if err != nil {
		return
	}
	if *english == "" {
		*english = m.Text(i18n.Default, nil)
	}
	if t.IsZero() {
		*t = m.Keep(nil)
	}
}

// WithMessages names the adapter's catalog and fills the descriptor's texts
// (and its Capabilities' Pricing and DedicatedIPs) from the catalog's
// well-known keys. Adapters call it from their Descriptor function.
func (d Descriptor) WithMessages(c *i18n.Catalog) Descriptor {
	d.Messages = c
	d.Capabilities = d.Capabilities.WithMessages(c)
	fill(c, &d.Events, &d.EventsText, "events")
	fill(c, &d.DomainSetup, &d.DomainSetupText, "domainSetup")
	fill(c, &d.SuppressionMode, &d.SuppressionModeText, "suppressionMode")
	if len(d.Fields) > 0 {
		fields := make([]Field, len(d.Fields))
		for i, f := range d.Fields {
			fill(c, &f.Label, &f.LabelText, "field."+f.Key+".label")
			fill(c, &f.Help, &f.HelpText, "field."+f.Key+".help")
			fields[i] = f
		}
		d.Fields = fields
	}
	if len(d.Terms) > 0 {
		terms := make([]Terms, len(d.Terms))
		for i, t := range d.Terms {
			n := "terms." + strconv.Itoa(i)
			fill(c, &t.Variant, &t.VariantText, n+".variant")
			fill(c, &t.Note, &t.NoteText, n+".note")
			terms[i] = t
		}
		d.Terms = terms
	}
	return d
}

// WithMessages fills Pricing and DedicatedIPs from the adapter's catalog
// (keys pricing and dedicatedIps), as Descriptor.WithMessages does.
func (c Capabilities) WithMessages(cat *i18n.Catalog) Capabilities {
	fill(cat, &c.Pricing, &c.PricingText, "pricing")
	fill(cat, &c.DedicatedIPs, &c.DedicatedIPsText, "dedicatedIps")
	return c
}

// In returns the capabilities with Pricing and DedicatedIPs in the reader's
// language. Use it for display only: the English values are data too.
func (c Capabilities) In(l i18n.Lang) Capabilities {
	c.Pricing = TextIn(c.Pricing, c.PricingText, l)
	c.DedicatedIPs = TextIn(c.DedicatedIPs, c.DedicatedIPsText, l)
	return c
}

// In returns the field with its label and hint in the reader's language.
func (f Field) In(l i18n.Lang) Field {
	f.Label = TextIn(f.Label, f.LabelText, l)
	f.Help = TextIn(f.Help, f.HelpText, l)
	return f
}

// In returns the terms with their variant and note in the reader's language.
func (t Terms) In(l i18n.Lang) Terms {
	t.Variant = TextIn(t.Variant, t.VariantText, l)
	t.Note = TextIn(t.Note, t.NoteText, l)
	return t
}

// In returns a copy of the descriptor with every text people read in the
// reader's language. Use it for display only: code that decides something
// from a descriptor's words (SuppressionMode "Pull only") reads the
// registry's English one.
func (d Descriptor) In(l i18n.Lang) Descriptor {
	d.Capabilities = d.Capabilities.In(l)
	d.Events = TextIn(d.Events, d.EventsText, l)
	d.DomainSetup = TextIn(d.DomainSetup, d.DomainSetupText, l)
	d.SuppressionMode = TextIn(d.SuppressionMode, d.SuppressionModeText, l)
	if len(d.Fields) > 0 {
		fields := make([]Field, len(d.Fields))
		for i, f := range d.Fields {
			fields[i] = f.In(l)
		}
		d.Fields = fields
	}
	if len(d.Terms) > 0 {
		terms := make([]Terms, len(d.Terms))
		for i, t := range d.Terms {
			terms[i] = t.In(l)
		}
		d.Terms = terms
	}
	return d
}

// Texts lists every text people read in a descriptor with its message, for
// checks (the contract suite): what the text is, its English and its Text.
func (d Descriptor) Texts() []DeclaredText {
	out := []DeclaredText{}
	add := func(what, english string, t i18n.Text) {
		if english != "" {
			out = append(out, DeclaredText{What: what, English: english, Text: t})
		}
	}
	add("pricing", d.Capabilities.Pricing, d.Capabilities.PricingText)
	add("dedicatedIps", d.Capabilities.DedicatedIPs, d.Capabilities.DedicatedIPsText)
	add("events", d.Events, d.EventsText)
	add("domainSetup", d.DomainSetup, d.DomainSetupText)
	add("suppressionMode", d.SuppressionMode, d.SuppressionModeText)
	for _, f := range d.Fields {
		add("field."+f.Key+".label", f.Label, f.LabelText)
		add("field."+f.Key+".help", f.Help, f.HelpText)
	}
	for i, t := range d.Terms {
		n := "terms." + strconv.Itoa(i)
		add(n+".variant", t.Variant, t.VariantText)
		add(n+".note", t.Note, t.NoteText)
	}
	return out
}

// DeclaredText is one text of a descriptor (Texts).
type DeclaredText struct {
	// What names it by its well-known key ("field.api_key.label").
	What    string
	English string
	Text    i18n.Text
}
