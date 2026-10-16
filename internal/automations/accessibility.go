package automations

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Accessibility checks before publishing (design expansion, X06): the
// publish dialog shows the result of the draft's emails, and a workspace
// that blocks on serious issues can't publish the version until they're
// fixed. The live version keeps sending.

// CodeAccessibilityBlocked is the problem code of a publish the workspace's
// accessibility policy stops.
const CodeAccessibilityBlocked = "accessibility_blocked"

// AccessibilityEmail is an email step of a version, as the checks see it.
type AccessibilityEmail struct {
	NodeID  string
	EmailID uuid.UUID
	Version int
	// Step names the step the way the publish dialog does ("Step 3 B").
	Step string
	MJML string
	HTML string
}

// AccessibilityDraft is the version being published.
type AccessibilityDraft struct {
	Version     int
	LiveVersion *int
	Emails      []AccessibilityEmail
}

// AccessibilityVerdict is what the publish dialog shows.
type AccessibilityVerdict struct {
	// Checks are the dialog's accessibility badges.
	Checks []AutomationPublishCheck
	// Blocked is set when the workspace blocks publishing with serious
	// issues; Blocker says what to fix.
	Blocked bool
	Blocker string
}

// AccessibilityGate checks a version's emails before it's published.
type AccessibilityGate interface {
	CheckDraft(ctx context.Context, ws *store.Workspace, d AccessibilityDraft) (AccessibilityVerdict, error)
}

// accessibility runs the gate on the draft, when one is wired.
func (s *Service) accessibility(ctx context.Context, q *store.Queries, ws *store.Workspace, a store.Automation, g AutomationGraph) (AccessibilityVerdict, error) {
	if s.Accessibility == nil {
		return AccessibilityVerdict{}, nil
	}
	d := AccessibilityDraft{Version: int(a.DraftVersion), Emails: s.emailsOf(ctx, q, ws.ID, g)}
	if a.LiveVersion != nil {
		v := int(*a.LiveVersion)
		d.LiveVersion = &v
	}
	return s.Accessibility.CheckDraft(ctx, ws, d)
}

// VersionEmails lists the email steps of an automation version with their
// content and step names.
func (s *Service) VersionEmails(ctx context.Context, wsID, automationID uuid.UUID, version int32) ([]AccessibilityEmail, error) {
	v, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: automationID, Version: version})
	if err != nil {
		return nil, err
	}
	return s.emailsOf(ctx, s.Q, wsID, s.graphOf(v.Graph)), nil
}

func (s *Service) emailsOf(ctx context.Context, q *store.Queries, wsID uuid.UUID, g AutomationGraph) []AccessibilityEmail {
	out := []AccessibilityEmail{}
	for _, n := range g.Nodes {
		if n.Type != NodeEmail || n.Data.Email == nil {
			continue
		}
		id, err := uuid.Parse(n.Data.Email.EmailID)
		if err != nil {
			continue
		}
		e, err := q.GetAutomationEmailVersion(ctx, store.GetAutomationEmailVersionParams{WorkspaceID: wsID, ID: id, Version: int32(n.Data.Email.EmailVersion)})
		if err != nil {
			continue
		}
		out = append(out, AccessibilityEmail{NodeID: n.ID, EmailID: id, Version: n.Data.Email.EmailVersion, Step: stepRef(g, n, e.Name), MJML: e.Mjml, HTML: e.Html})
	}
	return out
}

// stepRef is "Step 3 B": the email's step, with the path of the A/B split
// that leads to it.
func stepRef(g AutomationGraph, n AutomationNode, emailName string) string {
	ref := n.Data.Label
	if ref == "" {
		ref = EmailShortRef(emailName)
	}
	if ref == "" {
		ref = "Send email"
	}
	ref = strings.ToUpper(ref[:1]) + ref[1:]
	for _, e := range g.Edges {
		if e.Target != n.ID || e.Data.Path == "" {
			continue
		}
		if src := g.Node(e.Source); src != nil && src.Type == NodeSplit {
			ref += " " + e.Data.Path
		}
	}
	return ref
}
