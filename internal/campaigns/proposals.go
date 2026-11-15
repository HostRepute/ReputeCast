package campaigns

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// AcceptDraftIn applies a human-accepted proposal in the caller's transaction,
// so its decision, content, history and audit entry commit together. The
// caller refreshes the person's membership/scope and validates model output.
// A nil target creates an email draft without an audience or sender. Existing
// drafts require an exact base; no auto-merge, force or approval cancellation.
func (s *Service) AcceptDraftIn(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, target *uuid.UUID, base int64, name, subject, preheader, src string, origin audit.ProposalOrigin) (uuid.UUID, int64, error) {
	if !p.IsHuman() || p.WorkspaceID != ws.ID || !p.Can(authz.CampaignsEdit) {
		return uuid.Nil, 0, api.Forbidden(authz.CampaignsEdit, nil)
	}
	l := i18n.LangFrom(ctx)
	if strings.TrimSpace(name) == "" {
		return uuid.Nil, 0, api.Invalid("body.name", mProblemName.Text(l, nil))
	}
	if err := validateSubject(l, subject); err != nil {
		return uuid.Nil, 0, err
	}
	q := store.New(tx)
	in := ComposerSaveContent{Subject: subject, Preheader: preheader, MJML: src}
	meta := origin.Metadata()
	if target == nil {
		c, err := s.prepareContent(ctx, q, ws, uuid.Nil, in, ComposerContentSettings{})
		if err != nil {
			return uuid.Nil, 0, err
		}
		id, err := s.insertDraftIn(ctx, q, ws, p, draftRow{name: name, subject: c.subject, preheader: c.preheader, mjml: c.mjml, html: c.html,
			ab: c.ab, audience: json.RawMessage(`{}`), label: origin.Label(), source: "api"},
			&audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", Metadata: meta})
		return id, 1, err
	}
	cur, err := q.ComposerLockCampaign(ctx, store.ComposerLockCampaignParams{WorkspaceID: ws.ID, ID: *target})
	if err != nil {
		return uuid.Nil, 0, api.MapError(err, "Campaign")
	}
	if int64(cur.Version) != base || cur.Status != "draft" {
		return uuid.Nil, 0, api.Conflict(mProblemRevisionForProposal.Text(l, nil))
	}
	if approval, err := s.openApproval(ctx, q, *target); err != nil {
		return uuid.Nil, 0, err
	} else if approval != nil {
		return uuid.Nil, 0, approvalWouldCancel(l, approval)
	}
	in.ABTest = abFromDB(cur.AbTest)
	var settings ComposerContentSettings
	if err := json.Unmarshal(cur.ContentSettings, &settings); err != nil {
		return uuid.Nil, 0, err
	}
	c, err := s.prepareContent(ctx, q, ws, *target, in, settings)
	if err != nil {
		return uuid.Nil, 0, err
	}
	c.audited = true // One content-free domain event below, with provenance.
	saved, err := s.storeVersion(ctx, q, ws, p, *target, cur, c, origin.Label(), "api")
	if err != nil {
		return uuid.Nil, 0, err
	}
	meta["before_version"], meta["version"] = base, saved.Version
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.content.save", TargetType: "campaign", TargetID: target.String(), Metadata: meta}); err != nil {
		return uuid.Nil, 0, err
	}
	db.AfterCommit(tx, func() { s.changed(ctx, ws, *target) })
	return *target, int64(saved.Version), nil
}

var mProblemRevisionForProposal = msgs.M("problem.proposalRevision")
