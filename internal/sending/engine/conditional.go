package engine

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Conditional content, polls and surveys (design expansion, M21).
//
// An email with conditional blocks carries markers around each one in its
// compiled HTML; the audience build decides once which blocks each
// recipient doesn't get (messages.hidden_blocks) and the send renders that
// version from the same parsed HTML, cut at the markers: no MJML is compiled
// per recipient. Poll answer links and survey links are filled in per
// recipient like the unsubscribe link, never through the click redirect.

// Conditional plugs conditional content and polls into a send
// (internal/visibility and internal/polls). Nil sends every block and
// leaves poll links empty.
type Conditional interface {
	// Plan reads a campaign's Visibility rules when its audience build
	// starts; a nil Planner means the email has no conditional blocks.
	Plan(ctx context.Context, workspaceID, campaignID uuid.UUID) (Planner, error)
	// Links returns the per-recipient link variables of a campaign's polls
	// and surveys, creating the polls of its content when a send starts.
	Links(ctx context.Context, workspaceID, campaignID uuid.UUID) ([]RecipientLink, error)
	// Cut removes the hidden blocks from compiled HTML and every marker.
	Cut(html, hidden string) string
}

// Planner decides the hidden blocks of a page of recipients.
type Planner interface {
	Hidden(ctx context.Context, contacts []uuid.UUID) (map[uuid.UUID]string, error)
}

// RecipientLink is a Liquid variable filled in per recipient: a poll's
// answer link base ({{ rc_poll_a1b2c3 }}/2) or a survey's link.
type RecipientLink struct {
	Var string
	// URL builds the recipient's link from the message and contact ids.
	URL func(messageID, contactID uuid.UUID) string
}

// hiddenFor is the hidden blocks of a page's recipients, "" for everyone
// when the email has no conditional blocks.
func (e *Engine) hiddenFor(ctx context.Context, p Planner, contacts []uuid.UUID) (map[uuid.UUID]string, error) {
	if p == nil || len(contacts) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	return p.Hidden(ctx, contacts)
}

// versionKey is a template's key for a set of hidden blocks.
func versionKey(key, hidden string) string {
	if hidden == "" {
		return key
	}
	return key + "#" + hidden
}

// cutContent is a content's version without the hidden blocks, its plain
// text derived again from the HTML that is left.
func (e *Engine) cutContent(c Content, hidden string) (Content, func(string) string) {
	if hidden == "" || e.Conditional == nil {
		return c, nil
	}
	cut := func(html string) string { return e.Conditional.Cut(html, hidden) }
	if strings.Contains(c.HTML, "<!--rc:") {
		c.Text = ""
	}
	return c, cut
}

// versionTemplate returns the parsed template of a message's version: the
// render key's template when nothing is hidden, otherwise one prepared from
// the same content with the hidden blocks cut out after the links were
// numbered, so link positions stay those of campaign_links.
func (e *Engine) versionTemplate(rs *runState, key, hidden string) (*Template, error) {
	if hidden == "" || e.Conditional == nil {
		return rs.templates[key], nil
	}
	vk := versionKey(key, hidden)
	e.templMu.Lock()
	t, ok := rs.templates[vk]
	e.templMu.Unlock()
	if ok {
		return t, nil
	}
	content, cut := e.cutContent(rs.content.forVariant(key), hidden)
	t, err := PrepareCut(content, rs.content.Tracking, cut)
	if err != nil {
		return nil, err
	}
	e.templMu.Lock()
	rs.templates[vk] = t
	e.templMu.Unlock()
	return t, nil
}
