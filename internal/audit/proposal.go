package audit

import "github.com/google/uuid"

// ProposalOrigin records how content reached a human decision. It never
// substitutes an external connection's identity for the deciding person.
type ProposalOrigin struct {
	ProposalID uuid.UUID
	ArtifactID uuid.UUID
	RunID      *uuid.UUID
	GrantID    *uuid.UUID
}

func (p ProposalOrigin) Metadata() map[string]any {
	m := map[string]any{"via": "ai_assistant"}
	if p.ProposalID != uuid.Nil {
		m["proposal_id"] = p.ProposalID.String()
	}
	if p.ArtifactID != uuid.Nil {
		m["artifact_id"] = p.ArtifactID.String()
	}
	if p.RunID != nil {
		m["via"], m["run_id"] = "ai_agent", p.RunID.String()
	}
	if p.GrantID != nil {
		m["via"], m["grant_id"] = "external_mcp", p.GrantID.String()
	}
	return m
}

func (p ProposalOrigin) Label() string {
	if p.GrantID != nil {
		return "Accepted external proposal"
	}
	return "Accepted AI draft"
}
