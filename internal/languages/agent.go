package languages

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// AgentEdit is an MCP agent's change to a language version: fields left
// nil stay as they are.
type AgentEdit struct {
	Subject   *string
	Preheader *string
	MJML      *string
}

// AgentSaved is what an agent's change produced.
type AgentSaved struct {
	Language string
	Version  int
	Subject  string
	// Before are the subject and preheader before the change.
	BeforeSubject   string
	BeforePreheader string
	BeforeVersion   int
}

// AgentSaveCampaign changes a draft campaign's language version for an MCP
// agent. Agents only change drafts; the MCP server writes the audit entry.
func (s *Service) AgentSaveCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in AgentEdit) (AgentSaved, error) {
	lang = Base(lang)
	var out AgentSaved
	err := s.campaignTx(ctx, ws, id, false, func(q *store.Queries, o Owner, _ Source, c store.LanguagesCampaignSourceRow, _ sourceState, _ Settings) error {
		if c.Status != "draft" {
			return api.Conflict(fmt.Sprintf("Only drafts can be changed by an agent; %q is %s.", c.Name, c.Status))
		}
		t, err := lockVersion(ctx, q, o, lang)
		if err != nil {
			return err
		}
		out = AgentSaved{Language: lang, BeforeSubject: t.Subject, BeforePreheader: t.Preheader, BeforeVersion: int(t.Version)}
		subject, preheader, mj, html := t.Subject, t.Preheader, t.Mjml, t.Html
		if in.Subject != nil {
			subject = *in.Subject
		}
		if in.Preheader != nil {
			preheader = *in.Preheader
		}
		if in.MJML != nil {
			mj, html = *in.MJML, "" // compiled on the server
		} else if (in.Subject != nil || in.Preheader != nil) && strings.TrimSpace(mj) != "" {
			html = "" // the head's title and preview follow the subject and preheader
		}
		saved, _, err := s.saveVersion(ctx, q, o, p, lang, SaveInput{BaseVersion: int(t.Version), Subject: subject, Preheader: preheader, MJML: mj, HTML: html,
			ABTest: t.AbTest, Label: "changed by an agent", Source: "api", Audited: true})
		out.Version, out.Subject = saved.Version, subject
		return err
	})
	if err == nil && s.Changed != nil {
		s.Changed(ctx, ws, id)
	}
	return out, err
}
