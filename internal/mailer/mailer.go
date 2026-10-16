// Package mailer sends ReputeCast's own system mail (password resets, team
// invites, double opt-in, approval notifications) through whichever sending
// provider is routed to the System stream. Until one is connected every
// flow that would email has a fallback (copy a link, run a CLI command).
package mailer

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// ErrNoSystemProvider means no provider is connected to the System stream yet.
var ErrNoSystemProvider = errors.New("no sending provider is connected to the System stream")

// Message is one system email.
type Message struct {
	WorkspaceID *uuid.UUID
	To          string
	ToName      string
	Subject     string
	Text        string
	HTML        string
	// Kind tags the message for logs and the event timeline ("password_reset", "invite").
	Kind string
	// Lang is the language of the layout around the body (why it was sent,
	// who sent it): the member's app language for team mail. English when
	// unset.
	Lang i18n.Lang
}

// Sender delivers system mail. It returns the name of the provider used.
type Sender interface {
	SendSystem(ctx context.Context, m Message) (provider string, err error)
	// SystemAvailable reports whether a System-stream provider exists, so the
	// UI can offer copyable links before trying to send.
	SystemAvailable(ctx context.Context, workspaceID *uuid.UUID) bool
}

// Unavailable is the Sender used when no provider can send system mail.
type Unavailable struct{}

// SendSystem always fails with ErrNoSystemProvider.
func (Unavailable) SendSystem(context.Context, Message) (string, error) {
	return "", ErrNoSystemProvider
}

// SystemAvailable always reports false.
func (Unavailable) SystemAvailable(context.Context, *uuid.UUID) bool { return false }
