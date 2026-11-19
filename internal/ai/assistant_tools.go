package ai

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// What the assistant's tools use of this package (B3, step 7–8): a
// feature's switch, an attached campaign as the model may read it, a
// proposal on it through the drafts' review, and the one-shot suggestions.

// ContextTokens is the model's context window, as the price list knows it.
func (m *AssistantModel) ContextTokens() int { return llm.ContextTokens(m.c.Provider, m.c.Model) }

// AssistantFeatureOn reports whether managers left a feature on.
func (s *Service) AssistantFeatureOn(ctx context.Context, q *store.Queries, ws uuid.UUID, feature string) (bool, error) {
	row, err := q.AIWorkspaceSettings(ctx, ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	return err == nil && !slices.Contains(row.FeaturesOff, feature), err
}

// AssistantCampaign is an attached campaign as the model reads it: its
// editorial text, with links, images and Liquid tokens as opaque handles
// that stay on the server.
type AssistantCampaign struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Revision  int64     `json:"revision"`
	Subject   string    `json:"subject"`
	Preheader string    `json:"preheader"`
	Source    string    `json:"source"`
	Links     []string  `json:"links"`
	Images    []string  `json:"images"`
	Tokens    []string  `json:"tokens"`
}

// assistantDraft reads a campaign the person may read and projects it.
func (s *Service) assistantDraft(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (store.AIDraftCampaignRow, draftContext, campaignDraftTask, error) {
	var c draftContext
	fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
	if err != nil {
		return store.AIDraftCampaignRow{}, c, campaignDraftTask{}, err
	}
	if err := agentKindAccess(fresh, "campaign", false); err != nil {
		return store.AIDraftCampaignRow{}, c, campaignDraftTask{}, err
	}
	cur, err := draftCampaign(ctx, q, ws, fresh, id)
	if err != nil {
		return cur, c, campaignDraftTask{}, err
	}
	c = draftContext{Name: cur.Name, Language: "en", EmailContext: editorial.EmailContext{Bindings: []editorial.Binding{}},
		Before: &AIDraftEmail{Subject: cur.Subject, Preheader: cur.Preheader, MJML: cur.Mjml}}
	t, err := c.projectSource(ctx, cur.Mjml, nil)
	return cur, c, t, err
}

// AssistantReadCampaign is an attached campaign as the model may read it
// now: the person's current access decides, and Liquid stays unexpanded.
func (s *Service) AssistantReadCampaign(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (AssistantCampaign, error) {
	cur, _, t, err := s.assistantDraft(ctx, q, ws, p, id)
	if err != nil {
		return AssistantCampaign{}, err
	}
	return AssistantCampaign{ID: cur.ID, Name: cur.Name, Status: cur.Status, Revision: int64(cur.Version), Subject: t.Subject, Preheader: t.Preheader, Source: t.Source,
		Links: nonNil(t.Links), Images: nonNil(t.Images), Tokens: nonNil(t.Tokens)}, nil
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// AssistantProposeCampaign saves the model's edit of a draft campaign as a
// proposal for the person's review (the drafts' own: accepting it is the
// person's, with campaigns:edit, at the exact revision it was made on). The
// email is the projection read_attached gave, edited; its handles are
// restored here, and it must still be valid saved content.
func (s *Service) AssistantProposeCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread, id uuid.UUID, base int64, email AIDraftEmail) (uuid.UUID, error) {
	proposal := uuid.Must(uuid.NewV7())
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, true, authz.CampaignsEdit)
		if err != nil {
			return err
		}
		cur, c, _, err := s.assistantDraft(ctx, q, ws, fresh, id)
		if err != nil {
			return err
		}
		if int64(cur.Version) != base || cur.Status != "draft" {
			return draftProblem(ctx, 409, "changed")
		}
		if c.restoreEmail(&email) != nil {
			return draftProblem(ctx, 422, "invalid")
		}
		c.FinalPolicy = savedDraftPolicy(email.MJML)
		if err := c.validateSaved(ctx, q, ws, email); err != nil {
			if errors.Is(err, errDraftContent) {
				return draftProblem(ctx, 422, "invalid")
			}
			return err
		}
		if _, _, errs := mjml.Checked(email.MJML, ""); errs != nil {
			return draftProblem(ctx, 422, "invalid")
		}
		c.ThreadID = &thread
		contextJSON, err := json.Marshal(c)
		if err != nil {
			return err
		}
		content, err := json.Marshal(email)
		if err != nil {
			return err
		}
		now := s.now()
		if err := q.AIDraftInsert(ctx, store.AIDraftInsertParams{ID: proposal, WorkspaceID: ws.ID, UserID: fresh.UserID, Kind: "campaign", TargetID: &id,
			BaseRevision: base, Context: contextJSON, Content: content, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.draft.propose", TargetType: "ai_request", TargetID: proposal.String(),
			Metadata: map[string]any{"kind": "campaign", "base_revision": base, "sharing_policy": editorial.SharingPolicy, "via": "ai_assistant", "thread_id": thread.String()}})
	})
	return proposal, err
}

// AssistantSubjectIdeas runs the subject ideas on an attached campaign as
// it is saved: their own switch, model and reservation. What they send is
// the editorial projection read_attached gives (the same privacy checks,
// refused the same way), its Liquid tokens as written.
func (s *Service) AssistantSubjectIdeas(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (AISubjectIdeas, error) {
	_, c, t, err := s.assistantDraft(ctx, s.Q, ws, p, id)
	if err != nil {
		return AISubjectIdeas{}, err
	}
	return s.subjectIdeas(ctx, ws, p, AISubjectIdeasInput{CampaignID: id.String(), Subject: c.tokensBack(t.Subject), Preheader: c.tokensBack(t.Preheader)},
		c.tokensBack(t.Source))
}

// AssistantRewrite runs the rewrite on a passage of an attached campaign:
// its own switch, model and reservation. The campaign must pass the
// editorial projection, and the passage and the instruction (the model
// wrote both) its privacy checks, as read_attached's text does.
func (s *Service) AssistantRewrite(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, text, action, instruction string) (AIRewrite, error) {
	if _, _, _, err := s.assistantDraft(ctx, s.Q, ws, p, id); err != nil {
		return AIRewrite{}, err
	}
	passage, ok := editorial.Text(text)
	if !ok {
		return AIRewrite{}, draftProblem(ctx, 422, "privacy")
	}
	if instruction != "" {
		if instruction, ok = editorial.Text(instruction); !ok {
			return AIRewrite{}, draftProblem(ctx, 422, "privacy")
		}
	}
	return s.Rewrite(ctx, ws, p, AIRewriteInput{CampaignID: id.String(), Action: action, Instruction: instruction, HTML: "<p>" + html.EscapeString(passage) + "</p>"})
}

// tokensBack is projected text with its Liquid tokens as written; links
// and images stay out of it.
func (c *draftContext) tokensBack(text string) string {
	pairs := []string{}
	for _, b := range c.Bindings {
		if b.Kind == "token" {
			pairs = append(pairs, b.Handle, b.Value)
		}
	}
	return strings.NewReplacer(pairs...).Replace(text)
}
