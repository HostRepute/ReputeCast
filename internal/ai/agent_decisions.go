package ai

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

type AIAgentArtifactRevision struct {
	ID       uuid.UUID `json:"id"`
	Revision int64     `json:"revision" minimum:"1"`
}

type AIAgentDecision struct {
	Version   int64                     `json:"version" minimum:"1"`
	Decision  string                    `json:"decision" enum:"kept,discarded,accepted,rejected"`
	Artifacts []AIAgentArtifactRevision `json:"artifacts" minItems:"1" maxItems:"10"`
}

func agentPlanDiscards(plan AIAgentPlan, artifacts []store.AiAgentArtifact) []AIAgentArtifactRevision {
	out := []AIAgentArtifactRevision{}
	for _, a := range artifacts {
		if a.Decision == "" && !slices.ContainsFunc(plan.Items, func(item AIAgentPlanItem) bool { return item.Handle == a.Handle }) {
			out = append(out, AIAgentArtifactRevision{ID: a.ID, Revision: a.Revision})
		}
	}
	return out
}

// Replacing scope is a human decision, not a side effect of propose_plan.
// Exact acknowledgments and all removals commit with Build's run fence.
func (s *Service) confirmAgentPlan(ctx context.Context, q *store.Queries, p *authz.Principal, r store.AiAgentRun, c *agentContext, confirmed []AIAgentArtifactRevision) error {
	artifacts, err := q.AIAgentArtifacts(ctx, r.ID)
	if err != nil {
		return err
	}
	expected := agentPlanDiscards(c.Plan, artifacts)
	if len(confirmed) != len(expected) {
		return agentProblem(ctx, "bundle")
	}
	seen := map[uuid.UUID]bool{}
	for _, item := range confirmed {
		if seen[item.ID] || !slices.Contains(expected, item) {
			return agentProblem(ctx, "changed")
		}
		seen[item.ID] = true
	}
	for _, a := range artifacts {
		if !seen[a.ID] {
			continue
		}
		if err := agentKindAccess(p, a.Kind, true); err != nil {
			return err
		}
		decision := "rejected"
		if a.TargetID == nil {
			decision = "discarded"
			if c.Discarded == nil {
				c.Discarded = map[string]bool{}
			}
			c.Discarded[a.Bundle] = true
		}
		if err := q.AIAgentArtifactDecide(ctx, store.AIAgentArtifactDecideParams{ID: a.ID, Decision: decision, UpdatedAt: s.now()}); err != nil {
			return err
		}
		if err := q.AIAgentArtifactDiscard(ctx, a.ID); err != nil {
			return err
		}
		if err := s.agentAudit(ctx, q, p, r, decision, map[string]any{"artifact_id": a.ID, "plan_revision": r.PlanRevision}); err != nil {
			return err
		}
	}
	c.PreviousPlan = nil
	return nil
}

// DecideAgent is deliberately absent from the model registry. A person
// decides exact revisions of a whole dependency bundle, and domain writes,
// history, provenance and decisions share the same transaction.
func (s *Service) DecideAgent(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIAgentDecision) error {
	if validateDraftJSON(in) != nil {
		return agentProblem(ctx, "input")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.AIAgentRunLock(ctx, store.AIAgentRunLockParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "AI run")
		}
		if !s.now().Before(r.ExpiresAt) {
			return agentProblem(ctx, "expired")
		}
		c, err := loadAgentContext(ctx, q, id, true)
		if err != nil {
			return err
		}
		fresh, err := s.agentAccess(ctx, q, ws, p, r, c, false)
		if err != nil {
			return err
		}
		ctx = audit.WithPrincipal(ctx, fresh)
		all, err := q.AIAgentArtifacts(ctx, id)
		if err != nil {
			return err
		}
		selected := map[uuid.UUID]bool{}
		bundle := ""
		repeated := true
		for _, expected := range in.Artifacts {
			i := slices.IndexFunc(all, func(a store.AiAgentArtifact) bool { return a.ID == expected.ID })
			if i < 0 || selected[expected.ID] {
				return agentProblem(ctx, "bundle")
			}
			a := all[i]
			if a.Revision != expected.Revision {
				return agentProblem(ctx, "changed")
			}
			if err := agentKindAccess(fresh, a.Kind, true); err != nil {
				return err
			}
			if (a.Kind != "campaign" && a.Kind != "segment" && a.Kind != "automation" && a.Kind != "post") || (a.TargetID == nil) != (in.Decision == "kept" || in.Decision == "discarded") {
				return agentProblem(ctx, "input")
			}
			if a.Decision != "" && a.Decision != in.Decision {
				return agentProblem(ctx, "decided")
			}
			repeated = repeated && a.Decision == in.Decision
			if bundle != "" && a.Bundle != bundle {
				return agentProblem(ctx, "bundle")
			}
			bundle = a.Bundle
			selected[a.ID] = true
		}
		// Exact repeated decisions succeed even after the run version moved,
		// but never reapply content over subsequent human edits.
		if repeated {
			return nil
		}
		if in.Decision == "kept" || in.Decision == "accepted" {
			if err := c.poolAccess(ctx, q, ws, fresh, true); err != nil {
				return err
			}
		}
		if r.Version != in.Version || agentRunning(r.State) || r.AcceptedRevision == 0 {
			return agentProblem(ctx, "changed")
		}
		for _, a := range all {
			if a.Bundle == bundle && a.Decision == "" && !selected[a.ID] {
				return agentProblem(ctx, "bundle")
			}
		}
		if in.Decision == "kept" {
			for _, item := range c.Plan.Items {
				if item.Bundle == bundle && !slices.ContainsFunc(all, func(a store.AiAgentArtifact) bool {
					return a.Handle == item.Handle && selected[a.ID] && a.Decision == ""
				}) {
					return agentProblem(ctx, "bundle")
				}
			}
		}
		// Plans are validated in topological order. Database ordering by
		// handle is only for display and cannot decide dependency writes.
		for _, item := range c.Plan.Items {
			i := slices.IndexFunc(all, func(a store.AiAgentArtifact) bool { return a.Handle == item.Handle })
			if i < 0 {
				continue
			}
			a := &all[i]
			if !selected[a.ID] || a.Decision != "" {
				continue
			}
			if in.Decision == "kept" && a.Kind == "segment" {
				var definition segments.SegmentDefinition
				if err := decodeDraft(a.Content, &definition); err != nil {
					return draftProblem(ctx, 422, "invalid")
				}
				result, err := s.Segments.KeepProposalIn(ctx, tx, ws, fresh, item.Name, definition, audit.ProposalOrigin{ArtifactID: a.ID, RunID: &id})
				if err != nil {
					return segments.Problem(ctx, err)
				}
				a.ResultID = &result
			} else if (in.Decision == "kept" || in.Decision == "accepted") && a.Kind == "automation" {
				result, err := s.keepAgentAutomation(ctx, tx, ws, fresh, c, *a, all)
				if err != nil {
					return err
				}
				a.ResultID = &result
			} else if (in.Decision == "kept" || in.Decision == "accepted") && a.Kind == "post" {
				var post AIAgentPost
				var d draftContext
				if decodeDraft(a.Content, &post) != nil || json.Unmarshal(a.Context, &d) != nil {
					return draftProblem(ctx, 422, "invalid")
				}
				if err := d.validateSaved(ctx, q, ws, AIDraftEmail{Subject: post.Title, Preheader: post.Excerpt, MJML: post.MJML}); err != nil {
					return draftProblem(ctx, 422, "invalid")
				}
				var result uuid.UUID
				if a.TargetID == nil {
					result, err = s.Website.KeepDraftIn(ctx, tx, ws, fresh, post.Title, post.Excerpt, post.MJML, audit.ProposalOrigin{ArtifactID: a.ID, RunID: &id})
				} else {
					source, ok := c.proposalFor(*a)
					if !ok {
						return agentProblem(ctx, "changed")
					}
					result, err = s.Website.AcceptDraftIn(ctx, tx, ws, fresh, *a.TargetID, a.BaseRevision, source.Snapshot, post.Title, post.Excerpt, post.MJML, audit.ProposalOrigin{ArtifactID: a.ID, RunID: &id})
				}
				if err != nil {
					return err
				}
				a.ResultID = &result
			} else if in.Decision == "kept" || in.Decision == "accepted" {
				var d draftContext
				var email AIDraftEmail
				if err := json.Unmarshal(a.Context, &d); err != nil {
					return err
				}
				if err := decodeDraft(a.Content, &email); err != nil {
					return draftProblem(ctx, 422, "invalid")
				}
				if err := d.validateSaved(ctx, q, ws, email); err != nil {
					return draftProblem(ctx, 422, "invalid")
				}
				result, _, err := s.Campaigns.AcceptDraftIn(ctx, tx, ws, fresh, a.TargetID, a.BaseRevision, d.Name,
					email.Subject, email.Preheader, email.MJML, audit.ProposalOrigin{ProposalID: a.ID, RunID: &id})
				if err != nil {
					return err
				}
				a.ResultID = &result
			} else if err := s.agentAudit(ctx, q, fresh, r, in.Decision, map[string]any{"artifact_id": a.ID}); err != nil {
				return err
			}
			a.Decision = in.Decision
			if err := q.AIAgentArtifactDecide(ctx, store.AIAgentArtifactDecideParams{ID: a.ID, Decision: in.Decision, ResultID: a.ResultID, UpdatedAt: s.now()}); err != nil {
				return err
			}
			if in.Decision == "discarded" {
				if err := q.AIAgentArtifactDiscard(ctx, a.ID); err != nil {
					return err
				}
			}
		}
		if in.Decision == "discarded" {
			// A paused response may still contain later calls for this bundle.
			// Record its removal so continuation cannot recreate those drafts.
			if c.Discarded == nil {
				c.Discarded = map[string]bool{}
			}
			c.Discarded[bundle] = true
			if err := saveAgentContext(ctx, q, id, c); err != nil {
				return err
			}
		}
		r.Version++
		if r.State == "review" && !slices.ContainsFunc(all, func(a store.AiAgentArtifact) bool { return a.Decision == "" }) {
			r.State, r.ClosedAt = "completed", ptr(s.now())
		}
		return s.saveAgentRun(ctx, q, &r)
	})
}

// purgeAgentContent locks the same run rows as continuations and decisions.
// A sweep cannot race a worker into resurrecting purged messages or drafts.
func (s *Service) purgeAgentContent(ctx context.Context) (int64, error) {
	var count int64
	for {
		batch := 0
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			rows, err := q.AIAgentRetentionBatch(ctx, s.now())
			if err != nil {
				return err
			}
			batch = len(rows)
			for _, r := range rows {
				expired := !s.now().Before(r.ExpiresAt)
				if err := q.AIAgentPurgeRunSteps(ctx, store.AIAgentPurgeRunStepsParams{RunID: r.ID, Expired: expired}); err != nil {
					return err
				}
				if expired {
					if err := q.AIAgentPurgeRunContext(ctx, r.ID); err != nil {
						return err
					}
					if err := q.AIAgentPurgeRunArtifacts(ctx, r.ID); err != nil {
						return err
					}
					r.State, r.PauseReason = "expired", ""
					r.Version++
					if err := s.saveAgentRun(ctx, q, &r); err != nil {
						return err
					}
					if err := s.agentAudit(ctx, q, agentOwner(r), r, "expired", nil); err != nil {
						return err
					}
				} else if err := q.AIAgentPurgeRunMessages(ctx, r.ID); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return count, err
		}
		count += int64(batch)
		if batch < 100 {
			return count, nil
		}
	}
}
