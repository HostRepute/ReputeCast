package visibility

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Planner decides the hidden blocks of a send's recipients, page by page,
// while its audience is built. The rules are the source's: language
// versions keep the source's block ids.
type Planner struct {
	q    *store.Queries
	ws   uuid.UUID
	e    Email
	n    *Names
	segs Segments
}

// SendPlanner reads a campaign's rules when its send starts. It returns
// nil when the email has no conditional blocks.
func (s *Service) SendPlanner(ctx context.Context, wsID, campaignID uuid.UUID) (*Planner, error) {
	c, err := s.Q.VisibilityCampaign(ctx, store.VisibilityCampaignParams{WorkspaceID: wsID, ID: campaignID})
	if err != nil {
		return nil, err
	}
	e := Read(c.Mjml)
	if len(e.Blocks) == 0 {
		return nil, nil
	}
	n, err := LoadNames(ctx, s.Q, wsID)
	if err != nil {
		return nil, err
	}
	segs, err := LoadSegments(ctx, s.Segments, wsID, e.Segments(), n)
	if err != nil {
		return nil, err
	}
	return &Planner{q: s.Q, ws: wsID, e: e, n: n, segs: segs}, nil
}

// Hidden decides a page of recipients: their hidden blocks, as
// messages.hidden_blocks stores them ("" for those who get every block).
func (p *Planner) Hidden(ctx context.Context, contacts []uuid.UUID) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string, len(contacts))
	facts, err := LoadFacts(ctx, p.q, p.ws, contacts, p.segs, p.e)
	if err != nil {
		return nil, err
	}
	for id, f := range facts {
		if h := Decide(p.e, f, p.n).Hidden.String(); h != "" {
			out[id] = h
		}
	}
	return out, nil
}

// Cut is the email a recipient gets from the compiled HTML of the whole
// email and their hidden blocks.
func Cut(html, hidden string) string { return Apply(html, ParseHidden(hidden)) }

// visitor is someone who isn't a contact: they match no condition.
func visitor() *Facts {
	return &Facts{Lists: map[string]bool{}, Segments: map[string]bool{}, Tags: map[string]bool{}, Attributes: map[string]any{}, Extra: map[string]any{}}
}

// PublicHidden is the blocks someone who isn't a contact doesn't get (the
// web archive): every "Only contacts who match" block, none of the
// "Everyone except" ones.
func PublicHidden(src string) string {
	e := Read(src)
	if len(e.Blocks) == 0 {
		return ""
	}
	return Decide(e, visitor(), nil).Hidden.String()
}

// BrowserHTML is an email shown in the browser: a message's recipient gets
// their version (view online), anyone else the archive's (no message).
func (s *Service) BrowserHTML(ctx context.Context, wsID, campaignID, messageID uuid.UUID, html string) string {
	if !strings.Contains(html, "<!--rc:") {
		return html
	}
	if messageID != uuid.Nil {
		if hidden, err := s.Q.VisibilityMessageHidden(ctx, store.VisibilityMessageHiddenParams{ID: messageID, WorkspaceID: wsID}); err == nil {
			return Cut(html, hidden)
		}
	}
	c, err := s.Q.VisibilityCampaign(ctx, store.VisibilityCampaignParams{WorkspaceID: wsID, ID: campaignID})
	if err != nil {
		return Cut(html, "")
	}
	return Cut(html, PublicHidden(c.Mjml))
}

// HiddenFor is the blocks a contact doesn't get in a campaign's saved
// content (a test email sent as that contact).
func (s *Service) HiddenFor(ctx context.Context, wsID, campaignID, contactID uuid.UUID) (string, error) {
	c, err := s.Q.VisibilityCampaign(ctx, store.VisibilityCampaignParams{WorkspaceID: wsID, ID: campaignID})
	if err != nil {
		return "", err
	}
	e := Read(c.Mjml)
	if len(e.Blocks) == 0 {
		return "", nil
	}
	n, err := LoadNames(ctx, s.Q, wsID)
	if err != nil {
		return "", err
	}
	segs, err := LoadSegments(ctx, s.Segments, wsID, e.Segments(), n)
	if err != nil {
		return "", err
	}
	facts, err := LoadFacts(ctx, s.Q, wsID, []uuid.UUID{contactID}, segs, e)
	if err != nil {
		return "", err
	}
	f, ok := facts[contactID]
	if !ok {
		return "", nil
	}
	return Decide(e, f, n).Hidden.String(), nil
}
