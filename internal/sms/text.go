package sms

import "github.com/hostrepute/reputecast/internal/i18n"

// Text for people, as sending adapters have it (internal/providers,
// text.go): an adapter's manifest writes what people read (the catalog
// card's summary, a badge's note, the connection fields' labels and hints)
// in its own catalog, messages/<locale>.json, with well-known keys:
//
//	summary                                the catalog card's line
//	badge.<capability>.note                a badge's note ("read only")
//	field.<key>.label, field.<key>.help    a connection field
//
// WithMessages fills the manifest from them (the English where it is
// empty, and the message beside it); the SMS domain shows a manifest with
// In(lang), so it reads in each reader's language without importing the
// adapter. The contract suite (contracttest) checks that every text has a
// message saying what its English says, translated into every app language.

// TextIn returns a text in the reader's language: the message t when it
// exists, else english. English readers get english exactly as written.
func TextIn(english string, t i18n.Text, l i18n.Lang) string {
	if t.IsZero() || l.Locale == i18n.English || l.Locale == "" {
		return english
	}
	if s := t.In(l); s != "" {
		return s
	}
	return english
}

// fill sets a text from the catalog's message for key, when it has one:
// the English where it is empty, and the message unless one is set.
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

// WithMessages names the adapter's catalog and fills the manifest's texts
// from its well-known keys. Adapters call it from their Descriptor.
func (m Manifest) WithMessages(c *i18n.Catalog) Manifest {
	m.Messages = c
	fill(c, &m.Summary, &m.SummaryText, "summary")
	if len(m.Badges) > 0 {
		badges := make([]Badge, len(m.Badges))
		for i, b := range m.Badges {
			fill(c, &b.Note, &b.NoteText, "badge."+string(b.Capability)+".note")
			badges[i] = b
		}
		m.Badges = badges
	}
	if len(m.Fields) > 0 {
		fields := make([]Field, len(m.Fields))
		for i, f := range m.Fields {
			fill(c, &f.Label, &f.LabelText, "field."+f.Key+".label")
			fill(c, &f.Help, &f.HelpText, "field."+f.Key+".help")
			fields[i] = f
		}
		m.Fields = fields
	}
	return m
}

// In returns a copy of the manifest with every text in the reader's
// language, for display.
func (m Manifest) In(l i18n.Lang) Manifest {
	m.Summary = TextIn(m.Summary, m.SummaryText, l)
	if len(m.Badges) > 0 {
		badges := make([]Badge, len(m.Badges))
		for i, b := range m.Badges {
			b.Note = TextIn(b.Note, b.NoteText, l)
			badges[i] = b
		}
		m.Badges = badges
	}
	if len(m.Fields) > 0 {
		fields := make([]Field, len(m.Fields))
		for i, f := range m.Fields {
			fields[i] = f.In(l)
		}
		m.Fields = fields
	}
	return m
}

// In returns the field with its label and hint in the reader's language.
func (f Field) In(l i18n.Lang) Field {
	f.Label = TextIn(f.Label, f.LabelText, l)
	f.Help = TextIn(f.Help, f.HelpText, l)
	return f
}

// DeclaredText is one text of a manifest, for checks: its well-known key,
// its English and its message.
type DeclaredText struct {
	What    string
	English string
	Text    i18n.Text
}

// Texts lists every text people read in a manifest.
func (m Manifest) Texts() []DeclaredText {
	out := []DeclaredText{}
	add := func(what, english string, t i18n.Text) {
		if english != "" {
			out = append(out, DeclaredText{What: what, English: english, Text: t})
		}
	}
	add("summary", m.Summary, m.SummaryText)
	for _, b := range m.Badges {
		add("badge."+string(b.Capability)+".note", b.Note, b.NoteText)
	}
	for _, f := range m.Fields {
		add("field."+f.Key+".label", f.Label, f.LabelText)
		add("field."+f.Key+".help", f.Help, f.HelpText)
	}
	return out
}
