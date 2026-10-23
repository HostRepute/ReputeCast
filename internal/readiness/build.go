package readiness

import "github.com/hostrepute/reputecast/internal/i18n"

// check is a check as Assemble builds it: its texts are kept messages, and
// result writes their English into the ReadinessCheck a run stores.
type check struct {
	Key, Group, Status string
	// Mono is true when Detail is a record or a header.
	Mono                                                        bool
	Title, FlatTitle, Detail, Label, Evidence, Short, Ack, Note Text
	Fixes                                                       []fix
	Rows                                                        []row
	// every marks the passing links check that names all the links: an
	// email sent in several languages says they respond "in every version".
	every bool
}

// fix is a fix offered with a check (ReadinessFix).
type fix struct {
	Kind, Action, Tone, Href string
	Label                    Text
	Params                   map[string]string
}

// row is a line of a check's table (ReadinessCheckRow).
type row struct {
	Label, Detail, GetsName Text
	Count                   int64
	Language, Gets          string
}

func link(label Text, href string) fix {
	return fix{Kind: "link", Label: label, Href: href}
}

func apply(action string, label Text, tone string, params map[string]string) fix {
	return fix{Kind: "apply", Action: action, Label: label, Tone: tone, Params: params}
}

// result is the check as a run stores it: the English of its texts, and
// the messages they are written from.
func (c check) result() ReadinessCheck {
	t := &checkTexts{Title: c.Title, FlatTitle: c.FlatTitle, Detail: c.Detail, Label: c.Label, Evidence: c.Evidence, Short: c.Short, Ack: c.Ack, Note: c.Note}
	out := ReadinessCheck{Key: c.Key, Group: c.Group, Status: c.Status, Mono: c.Mono, Title: c.Title.en(), FlatTitle: c.FlatTitle.en(), Detail: c.Detail.en(),
		Label: c.Label.en(), Evidence: c.Evidence.sentencesIn(i18n.Default), Short: c.Short.en(), Ack: c.Ack.en(), Note: c.Note.en(),
		Fixes: make([]ReadinessFix, 0, len(c.Fixes)), texts: t}
	for _, f := range c.Fixes {
		out.Fixes = append(out.Fixes, ReadinessFix{Kind: f.Kind, Action: f.Action, Label: f.Label.en(), Tone: f.Tone, Href: f.Href, Params: f.Params})
		t.Fixes = append(t.Fixes, f.Label)
	}
	if len(c.Rows) > 0 {
		out.Rows = make([]ReadinessCheckRow, 0, len(c.Rows))
		for _, r := range c.Rows {
			out.Rows = append(out.Rows, ReadinessCheckRow{Label: r.Label.en(), Count: r.Count, Detail: r.Detail.en(), Language: r.Language, Gets: r.Gets, GetsName: r.GetsName.en()})
			t.Rows = append(t.Rows, rowTexts{Label: r.Label, Detail: r.Detail, GetsName: r.GetsName})
		}
	}
	return out
}
