package campaigns

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// Drafts AI agents write through the MCP server (create_campaign_draft,
// update_campaign_content) are stored the way the composer's are (review
// 2026-10, R4.L3, CO-3): the server compiles the MJML with gomjml and
// checks it, the HTML that is sent is that compile, and every change is a
// version in the history. The MCP server writes the audit entry of the
// tool call, with the agent as its actor.

// AgentContent is what an agent sets on a draft; nil leaves a field as it is.
type AgentContent struct {
	Subject, Preheader, MJML *string
}

// AgentSaved is what an agent's save stored.
type AgentSaved struct {
	Version, BeforeVersion     int
	Subject, BeforeSubject     string
	Preheader, BeforePreheader string
	SizeBytes                  int
	ContentHash                string
	// Unchanged is true when the draft already had this content: no
	// version was added.
	Unchanged bool
}

// SaveForAgent stores an agent's change to a draft as its next version,
// like a person's save that sends no HTML: the MJML is compiled and
// checked on the server (an MJML error is refused), the head's title and
// preview follow the subject and preheader, A/B variants keep their
// content, and approvals and Send Readiness see the change. It never
// cancels a person's approval or pending request (409
// approval_would_cancel), and it leaves the audit entry to the caller.
func (s *Service) SaveForAgent(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AgentContent) (AgentSaved, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AgentSaved{}, api.MapError(err, "Campaign")
	}
	save := ComposerSaveContent{BaseVersion: int(c.Version), Subject: c.Subject, Preheader: c.Preheader, MJML: c.Mjml, HTML: c.Html, ABTest: abFromDB(c.AbTest),
		Label: mVersionAgent.Text(i18n.Default, nil), Source: "api", askApproval: true, audited: true}
	if in.Subject != nil {
		save.Subject = *in.Subject
	}
	if in.Preheader != nil {
		save.Preheader = *in.Preheader
	}
	switch {
	case in.MJML != nil:
		// The HTML is the server's compile of the new MJML (CO-3).
		save.MJML, save.HTML = *in.MJML, ""
	case strings.TrimSpace(save.MJML) != "":
		// Compiled again: the head's title and preview follow the subject
		// and preheader. An HTML-only email keeps its HTML.
		save.HTML = ""
	}
	out := AgentSaved{BeforeVersion: int(c.Version), BeforeSubject: c.Subject, BeforePreheader: c.Preheader, Subject: save.Subject, Preheader: save.Preheader}
	saved, err := s.SaveIn(ctx, ws, p, id, "", save)
	if err != nil {
		return AgentSaved{}, err
	}
	out.Version, out.SizeBytes, out.ContentHash, out.Unchanged = saved.Version, saved.SizeBytes, saved.ContentHash, saved.Unchanged
	return out, nil
}

// CreateForAgent makes a draft an agent wrote: its MJML compiled and checked
// on the server (an MJML error is refused), the workspace's default sender
// as for any new draft, and a first version "created by an agent". It
// leaves the audit entry to the caller.
func (s *Service) CreateForAgent(ctx context.Context, ws *store.Workspace, p *authz.Principal, name string, in AgentContent) (ComposerCampaign, error) {
	l := i18n.LangFrom(ctx)
	name = strings.TrimSpace(name)
	if name == "" {
		return ComposerCampaign{}, api.Invalid("body.name", mProblemName.Text(l, nil))
	}
	var subject, preheader, src string
	if in.Subject != nil {
		subject = *in.Subject
	}
	if in.Preheader != nil {
		preheader = *in.Preheader
	}
	if in.MJML != nil {
		src = *in.MJML
	}
	if err := validateSubject(l, subject); err != nil {
		return ComposerCampaign{}, err
	}
	html := ""
	if strings.TrimSpace(src) != "" {
		src = mjml.SetHeadMeta(src, subject, preheader)
		compiled, _, errs := mjml.Checked(src, "")
		if errs != nil {
			// The compiler's own message stays as it wrote it.
			return ComposerCampaign{}, api.Invalid("body.mjml", mProblemMJMLCompile.Text(l, i18n.Args{"error": errs[0].Message}))
		}
		html = accessibility.FinalizeStored(ctx, s.Q, ws.ID, accessibility.OwnerCampaign, uuid.Nil, 0, compiled)
	}
	id, err := s.insertDraft(ctx, ws, p, draftRow{name: name, subject: subject, preheader: preheader, mjml: src, html: html,
		label: mVersionAgentCreated.Text(i18n.Default, nil), source: "api", audience: []byte(`{}`), sender: s.defaultSender(ctx, ws)}, nil)
	if err != nil {
		return ComposerCampaign{}, err
	}
	return s.Detail(ctx, ws, p, id)
}
