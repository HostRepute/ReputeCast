package campaigns

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Follow-ups from a report (review 2026-11 CA-4): resend to the people who
// didn't open, or send to those who clicked, as a new draft whose audience
// is a segment of the report's buckets. A privacy-protected open may be a
// person's, so those readers are left out of a resend rather than mailed
// twice.

var (
	mFollowUpName     = msgs.M("followUp.name")
	mFollowUpSegment  = msgs.M("followUp.segment")
	mProblemFollowUp  = msgs.M("problem.followUpSent")
	mProblemFollowSub = msgs.M("problem.followUpSubject")
)

// CampaignFollowUpCounts says who a follow-up would reach.
type CampaignFollowUpCounts struct {
	Received   int64 `json:"received"`
	NonOpeners int64 `json:"nonOpeners" doc:"No open at all"`
	// MachineOnly opened only through privacy protection (Apple Mail): they
	// count as unknown and are left out of a resend.
	MachineOnly int64      `json:"machineOnly"`
	Clickers    int64      `json:"clickers"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	// Subject is the original's, for the follow-up's suggested subject.
	Subject string `json:"subject"`
}

// FollowUpCounts counts the report's buckets for a follow-up.
func (s *Service) FollowUpCounts(ctx context.Context, ws *store.Workspace, id uuid.UUID) (CampaignFollowUpCounts, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return CampaignFollowUpCounts{}, api.MapError(err, "Campaign")
	}
	n, err := s.Q.CampaignFollowUpCounts(ctx, &id)
	if err != nil {
		return CampaignFollowUpCounts{}, err
	}
	return CampaignFollowUpCounts{Received: n.Received, NonOpeners: n.NonOpeners, MachineOnly: n.MachineOnly, Clickers: n.Clickers, SentAt: c.StartedAt, Subject: c.Subject}, nil
}

// CampaignFollowUpInput is a follow-up to make.
type CampaignFollowUpInput struct {
	Kind    string `json:"kind" enum:"non_openers,clickers"`
	Subject string `json:"subject" minLength:"1" maxLength:"998"`
	// DelayHours after the original went out suggests when to send it
	// (Review & send takes it); 0 leaves the time to Review & send.
	DelayHours int `json:"delayHours,omitempty" minimum:"0" maximum:"720"`
}

// FollowUp makes the follow-up draft: a copy of the campaign with its own
// subject, sent to a segment of the report's bucket.
func (s *Service) FollowUp(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in CampaignFollowUpInput) (ComposerCampaign, error) {
	l := i18n.LangFrom(ctx)
	src, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerCampaign{}, api.MapError(err, "Campaign")
	}
	if src.Status != "sent" && src.Status != "sending" && src.Status != "paused" {
		return ComposerCampaign{}, api.Conflict(mProblemFollowUp.Text(l, nil))
	}
	subject := strings.TrimSpace(in.Subject)
	if subject == "" {
		return ComposerCampaign{}, api.Invalid("body.subject", mProblemFollowSub.Text(l, nil))
	}
	var conds []map[string]string
	conds = append(conds, map[string]string{"field": "received_campaign", "op": "received", "value": id.String()})
	if in.Kind == "clickers" {
		conds = append(conds, map[string]string{"field": "clicked_campaign", "op": "clicked", "value": id.String()})
	} else {
		conds = append(conds, map[string]string{"field": "opened_campaign", "op": "not_opened_any", "value": id.String()})
	}
	def, _ := json.Marshal(map[string]any{"groups": []map[string]any{{"match": "all", "conditions": conds}}})
	by := userRef(p)
	segID := uuid.Must(uuid.NewV7())
	if _, err := s.Q.CreateSegment(ctx, store.CreateSegmentParams{ID: segID, WorkspaceID: ws.ID, Name: mFollowUpSegment.Text(l, i18n.Args{"kind": in.Kind, "name": src.Name}),
		Mode: "builder", Refresh: "dynamic", Definition: def, CreatedBy: by}); err != nil {
		return ComposerCampaign{}, err
	}
	draft, err := s.Create(ctx, ws, p, ComposerCreateCampaign{Name: mFollowUpName.Text(l, i18n.Args{"name": src.Name}), FromCampaignID: id.String()})
	if err != nil {
		return ComposerCampaign{}, err
	}
	did, _ := uuid.Parse(draft.ID)
	if _, err := s.Update(ctx, ws, p, did, ComposerUpdateCampaign{Audience: &ComposerAudienceInput{Segments: []string{segID.String()}}}); err != nil {
		return ComposerCampaign{}, err
	}
	if _, err := s.Save(ctx, ws, p, did, ComposerSaveContent{BaseVersion: draft.Version, Subject: subject, Preheader: draft.Preheader, MJML: draft.MJML, HTML: draft.HTML,
		Source: "manual", Label: mFollowUpName.Text(l, i18n.Args{"name": src.Name})}); err != nil {
		return ComposerCampaign{}, err
	}
	if in.DelayHours > 0 && src.StartedAt != nil {
		if at := src.StartedAt.Add(time.Duration(in.DelayHours) * time.Hour); at.After(s.now()) {
			sb, _ := json.Marshal(map[string]string{"mode": "at", "at": at.UTC().Format(time.RFC3339), "timezone": ws.Timezone})
			if err := s.Q.SetCampaignScheduleDraft(ctx, store.SetCampaignScheduleDraftParams{ID: did, Schedule: sb}); err != nil {
				return ComposerCampaign{}, err
			}
		}
	}
	if err := s.Audit.Log(ctx, auditFollowUp(ws, did, draft.Name, id, in)); err != nil {
		return ComposerCampaign{}, err
	}
	return s.Detail(ctx, ws, p, did)
}

func auditFollowUp(ws *store.Workspace, id uuid.UUID, name string, from uuid.UUID, in CampaignFollowUpInput) audit.Event {
	return audit.Event{WorkspaceID: &ws.ID, Action: "campaign.follow_up", TargetType: "campaign", TargetID: id.String(), TargetName: name,
		Metadata: map[string]any{"from": from.String(), "kind": in.Kind, "delayHours": in.DelayHours}}
}
