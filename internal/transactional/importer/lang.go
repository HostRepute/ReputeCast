package importer

import (
	"context"
	"embed"
	"errors"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The import's words are in messages/<locale>.json: the conversion report's
// notes and its Notes cell, a provider's failure and the dialog's problems.
// A note keeps its English in Message, as the API, agents and the stored
// report have it, and its message beside it (Note.Text), so the report reads
// in each member's language (Note.In). Code in backticks, a provider's own
// answer and the Liquid engine's and MJML compiler's findings stay as they
// are written.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "templateimport")

// say is a sentence of the import kept with its arguments.
func say(key string, args i18n.Args) i18n.Text { return msgs.M(key).Keep(args) }

// sayIn is a sentence of the import in the request's language.
func sayIn(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// noteOf is a note whose sentence is t: its English in Message, t kept.
func noteOf(kind NoteKind, field string, line int, construct string, t i18n.Text) Note {
	return Note{Kind: kind, Field: field, Line: line, Construct: construct, Message: t.In(i18n.Default), Text: t}
}

// In is the note in a reader's language: its kept message's words, else
// the English it has.
func (n Note) In(l i18n.Lang) Note {
	if n.Text.IsZero() || l.Locale == i18n.English {
		return n
	}
	if s := n.Text.In(l); s != "" {
		n.Message = s
	}
	return n
}

// errorText is a provider failure in a reader's language: a SourceError's
// kept message, else its English.
func errorText(l i18n.Lang, err error) string {
	var se *SourceError
	if errors.As(err, &se) && !se.Text.IsZero() && l.Locale != i18n.English {
		if s := se.Text.In(l); s != "" {
			return s
		}
	}
	return err.Error()
}
