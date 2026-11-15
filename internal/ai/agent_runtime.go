package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

type AgentArgs struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	RunID       uuid.UUID `json:"runId"`
	Version     int64     `json:"version"`
}

func (AgentArgs) Kind() string { return "ai.agent" }
func (AgentArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 12}
}

type AgentWorker struct {
	river.WorkerDefaults[AgentArgs]
	S *Service
}

// agentLease outlives one provider call and the response's checkpoint; the
// job outlives the lease, so a slow answer is not mistaken for a lost worker.
const agentLease = longestTimeout + time.Minute

func (w *AgentWorker) Timeout(*river.Job[AgentArgs]) time.Duration { return agentLease + time.Minute }
func (w *AgentWorker) Work(ctx context.Context, job *river.Job[AgentArgs]) error {
	return w.S.ExecuteAgent(ctx, job.Args)
}

func (s *Service) enqueueAgent(ctx context.Context, tx pgx.Tx, r store.AiAgentRun) error {
	if s.Jobs == nil || s.Jobs() == nil {
		return errors.New("agent job queue unavailable")
	}
	_, err := s.Jobs().InsertTx(ctx, tx, AgentArgs{WorkspaceID: r.WorkspaceID, RunID: r.ID, Version: r.Version}, nil)
	return err
}

func (s *Service) StartAgent(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIAgentBrief) (AIAgentRunView, error) {
	if !in.SharingReviewed {
		return AIAgentRunView{}, draftProblem(ctx, 422, "sharing_review")
	}
	if validateDraftJSON(in) != nil || (in.CampaignID == nil) != (in.BaseRevision == 0) || (in.CampaignID != nil && in.TemplateID != "") {
		return AIAgentRunView{}, agentProblem(ctx, "input")
	}
	brief, safe := voiceText(in.Brief)
	if !safe || strings.TrimSpace(brief) == "" || voiceURL.MatchString(in.Brief) || voiceLiquid.MatchString(in.Brief) || draftReserved.MatchString(in.Brief) {
		return AIAgentRunView{}, draftProblem(ctx, 422, "privacy")
	}
	p, err := draftActor(ctx, s.Q, ws, p, true, authz.AIUse)
	if err != nil {
		return AIAgentRunView{}, err
	}
	var key *string
	var digest []byte
	if in.RequestKey != "" {
		raw, _ := json.Marshal(in)
		sum := sha256.Sum256(raw)
		key, digest = &in.RequestKey, sum[:]
		if run, retried, err := s.agentRetry(ctx, ws, p, in.RequestKey, digest); retried {
			return run, err
		}
	}
	c := agentContext{Brief: in, Draft: draftContext{Language: in.Language, TemplateID: in.TemplateID, EmailContext: editorial.EmailContext{Bindings: []editorial.Binding{}}}, Messages: []llm.Message{},
		Conversation: []AIAgentMessage{{Role: "user", Text: brief, At: s.now()}}}
	if err := s.prepareAgentAudience(ctx, ws, p, &c); err != nil {
		return AIAgentRunView{}, err
	}
	source, _ := templates.StarterMJML("blank", "")
	if in.CampaignID != nil {
		row, err := draftCampaign(ctx, s.Q, ws, p, *in.CampaignID)
		if err != nil {
			return AIAgentRunView{}, err
		}
		if row.Status != "draft" || int64(row.Version) != in.BaseRevision {
			return AIAgentRunView{}, draftProblem(ctx, 409, "changed")
		}
		c.Draft.Before = &AIDraftEmail{Subject: row.Subject, Preheader: row.Preheader, MJML: row.Mjml}
		c.Draft.Name, source = row.Name, row.Mjml
	} else if in.TemplateID != "" {
		if !p.Can(authz.TemplatesView) {
			return AIAgentRunView{}, api.Forbidden(authz.TemplatesView, nil)
		}
		tpl, err := s.Campaigns.Templates.Get(ctx, ws, in.TemplateID, "")
		if err != nil {
			return AIAgentRunView{}, err
		}
		source, c.Draft.TemplateHash = tpl.MJML, sha256.Sum256([]byte(tpl.MJML))
	}
	source, err = s.prepareAgentPool(ctx, ws, p, &c, source)
	if err != nil {
		return AIAgentRunView{}, err
	}
	c.Source, err = c.Draft.projectSource(ctx, source, in.Links)
	if err != nil {
		return AIAgentRunView{}, err
	}
	c.Source.Brief, c.Source.Language = brief, languageName(in.Language)
	c.Literals.source(&c.Source)
	projected, err := json.Marshal(c.Source)
	if err != nil {
		return AIAgentRunView{}, err
	}
	t := task{selectedVoice: in.VoiceRevision, privateLiterals: c.Literals, req: llm.Request{Messages: []llm.Message{{Role: "user", Parts: []llm.Part{textPart(tag("editorial_context", string(projected)))}}}}}
	if in.Kind == "post" {
		t.req.Messages[0].Parts = append(t.req.Messages[0].Parts, textPart("The person selected a new web-only post, not an email campaign. Plan a post item without a target; its title and excerpt are at most 200 characters each."))
	}
	if in.CampaignID != nil {
		t.req.Messages[0].Parts = append(t.req.Messages[0].Parts, textPart("The selected source is an existing draft, referenced only as selected_campaign."))
	}
	if c.Pool != nil {
		t.req.Messages[0].Parts = append(t.req.Messages[0].Parts, textPart("The person selected a discount pool for new emails. Its code and redemption link are opaque source bindings; preserve both without inventing discount terms. Inventory is advisory, not a reservation or guaranteed sends.", tag("selected_discount_pool", string(c.Pool.modelJSON()))))
	}
	if c.Audience != nil {
		raw, err := c.Audience.modelJSON()
		if err != nil {
			return AIAgentRunView{}, err
		}
		t.req.Messages[0].Parts = append(t.req.Messages[0].Parts, textPart("The person selected this exact audience. Bound symbols are local values, not placeholders to guess. Audience changes require local selection and another confirmed plan.", tag("selected_audience", string(raw))))
	}
	if in.Automation != nil {
		if err := s.agentAutomationAccess(ctx, s.Q, ws, p, c, true); err != nil {
			return AIAgentRunView{}, err
		}
		// Project the selection without serializing its resource IDs.
		projection := c
		selection := *in.Automation
		selection.FromAudience = false
		projection.Brief.Automation = &selection
		trigger, err := projection.automationTrigger(AIAgentPlanItem{})
		if err != nil {
			return AIAgentRunView{}, err
		}
		raw, err := json.Marshal(trigger)
		if err != nil {
			return AIAgentRunView{}, err
		}
		instruction := "Use this exact selected trigger. Existing members are not enrolled by Keep."
		if in.Automation.FromAudience {
			instruction += " Plan a new segment from the selected audience. Replace selected_segment with that segment's handle, the automation's single dependency in the same bundle."
		}
		t.req.Messages[0].Parts = append(t.req.Messages[0].Parts, textPart(instruction, tag("selected_trigger", string(raw))))
	}
	if err := s.withVoice(ctx, ws, &t); err != nil {
		return AIAgentRunView{}, err
	}
	c.Brief.VoiceRevision, c.Messages = &t.voiceRevision, t.req.Messages
	id := uuid.Must(uuid.NewV7())
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r := store.AiAgentRun{ID: id, WorkspaceID: ws.ID, UserID: p.UserID, State: "planning", Version: 1}
		fresh, err := s.agentAccess(ctx, q, ws, p, r, c, true)
		if err != nil {
			return err
		}
		now := s.now()
		expires := now.Add(30 * 24 * time.Hour)
		if in.ReportID != nil {
			snapshot, err := reports.ReadModelSnapshot(ctx, q, fresh, *in.ReportID, now)
			if err != nil {
				return agentProblem(ctx, "report_unavailable")
			}
			c.Report = &snapshot
			// The selected copy follows this run's fixed 30-day retention.
			// Expiry of the shared source prevents new selections, not recovery
			// of drafts in an already-open run. Never refresh its numbers.
			c.Messages[0].Parts = append(c.Messages[0].Parts, textPart("The person selected one campaign report. Use read_report for its fixed totals. Names, content, links and audience filters are not selected by this report reference."))
		}
		settings, err := q.AIAgentSettings(ctx, ws.ID)
		if err != nil {
			return err
		}
		conn, err := q.AIConnection(ctx, ws.ID)
		if err != nil {
			return err
		}
		if !conn.PriceKnown && !conn.Demo {
			return budgetProblem(ctx, "price_unknown")
		}
		c.Model = agentModelOf(conn)
		if err := q.AIAgentRunInsert(ctx, store.AIAgentRunInsertParams{ID: id, WorkspaceID: ws.ID, UserID: p.UserID, Model: conn.Model,
			ToolLimit: settings.ToolLimit, TokenLimit: settings.TokenLimit, CostLimitMicros: settings.CostLimitMicros, Now: now, ExpiresAt: expires,
			RequestKey: key, RequestHash: digest}); err != nil {
			return err
		}
		if err := q.AIAgentContextInsert(ctx, store.AIAgentContextInsertParams{RunID: id, Brief: json.RawMessage(`{}`), Bindings: json.RawMessage(`{}`)}); err != nil {
			return err
		}
		if err := saveAgentContext(ctx, q, id, c); err != nil {
			return err
		}
		if err := s.agentAudit(ctx, q, fresh, r, "start", map[string]any{"sharing_policy": editorial.SharingPolicy}); err != nil {
			return err
		}
		return s.enqueueAgent(ctx, tx, r)
	})
	var duplicate *pgconn.PgError
	if key != nil && errors.As(err, &duplicate) && duplicate.Code == "23505" && duplicate.ConstraintName == "ai_agent_runs_request_key_idx" {
		// The same brief arrived twice at once; the other request created it.
		run, _, err := s.agentRetry(ctx, ws, p, in.RequestKey, digest)
		return run, err
	}
	if err != nil {
		return AIAgentRunView{}, err
	}
	return s.AgentRun(ctx, ws, p, id)
}

// agentRetry answers a brief retried after a lost response with the run it
// already created, without another plan or charge. The same key with a
// different brief is refused rather than guessed.
func (s *Service) agentRetry(ctx context.Context, ws *store.Workspace, p *authz.Principal, key string, digest []byte) (AIAgentRunView, bool, error) {
	prior, err := s.Q.AIAgentRunByRequest(ctx, store.AIAgentRunByRequestParams{WorkspaceID: ws.ID, UserID: p.UserID, RequestKey: &key})
	if errors.Is(err, pgx.ErrNoRows) {
		return AIAgentRunView{}, false, nil
	}
	if err != nil {
		return AIAgentRunView{}, true, err
	}
	if !bytes.Equal(prior.RequestHash, digest) {
		return AIAgentRunView{}, true, agentProblem(ctx, "request_reused")
	}
	run, err := s.AgentRun(ctx, ws, p, prior.ID)
	return run, true, err
}

func agentRunning(state string) bool {
	return state == "planning" || state == "queued" || state == "building"
}

func agentOwner(r store.AiAgentRun) *authz.Principal {
	return &authz.Principal{Type: authz.ActorUser, ID: r.UserID, UserID: r.UserID, WorkspaceID: r.WorkspaceID}
}

func (s *Service) pauseAgent(ctx context.Context, q *store.Queries, r *store.AiAgentRun, reason string) error {
	r.State, r.PauseReason = "paused", reason
	r.Version++
	if err := s.saveAgentRun(ctx, q, r); err != nil {
		return err
	}
	if err := s.agentAudit(ctx, q, agentOwner(*r), *r, "pause", map[string]any{"reason": reason}); err != nil {
		return err
	}
	return notifyAgent(ctx, q, *r)
}

// Called only for worker transitions, in their checkpoint transaction. The
// run fence deduplicates retries. Notifications contain no task content.
func notifyAgent(ctx context.Context, q *store.Queries, r store.AiAgentRun) error {
	if r.State != "ready" && r.State != "needs_answers" && r.State != "paused" && r.State != "review" {
		return nil
	}
	ws, err := q.GetWorkspace(ctx, r.WorkspaceID)
	if err != nil {
		return err
	}
	return notify.ToUserWith(ctx, q, r.UserID, notify.Notification{
		WorkspaceID: &r.WorkspaceID, Kind: notify.KindAIAgentReady,
		TitleText: msgs.M("notify.agentTitle").Keep(nil), BodyText: msgs.M("notify.agentBody").Keep(nil),
		Link: "/w/" + ws.Slug + "/agent/" + r.ID.String(),
		Data: map[string]any{"run_id": r.ID.String(), "state": r.State, "version": r.Version},
	})
}

// Operational failures must retry the transaction, not masquerade as a
// user's permission or budget problem. Only domain denials pause the run.
func (s *Service) pauseAgentFor(ctx context.Context, q *store.Queries, r *store.AiAgentRun, err error) error {
	var p *api.Problem
	if !errors.As(err, &p) || p.Status >= 500 {
		return err
	}
	reason := "access_changed"
	if strings.HasPrefix(p.Code, "ai_") {
		reason = strings.TrimPrefix(p.Code, "ai_")
	}
	return s.pauseAgent(ctx, q, r, reason)
}

// ExecuteAgent performs at most one paid request per job. Before HTTP, the
// reservation and requested checkpoint commit together. A crash there is
// uncertain, never an implicit second charge. Saved responses resume tools.
func (s *Service) ExecuteAgent(ctx context.Context, args AgentArgs) error {
	ws, err := s.Q.GetWorkspace(ctx, args.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var step store.AiAgentStep
	var req llm.Request
	var conn *connection
	var owner *authz.Principal
	var lease uuid.UUID
	ready, send, busy := false, false, false
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.AIAgentRunLock(ctx, store.AIAgentRunLockParams{WorkspaceID: ws.ID, ID: args.RunID})
		if err != nil {
			return err
		}
		if r.Version != args.Version || !agentRunning(r.State) || !s.now().Before(r.ExpiresAt) {
			return nil
		}
		c, err := loadAgentContext(ctx, q, r.ID, true)
		if err != nil {
			return err
		}
		owner, err = s.agentAccess(ctx, q, &ws, agentOwner(r), r, c, true)
		if err != nil {
			return s.pauseAgentFor(ctx, q, &r, err)
		}
		step, err = q.AIAgentStepLast(ctx, r.ID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if step.ID != uuid.Nil && step.RunVersion == r.Version {
			if step.State == "responded" {
				ready = true
				return nil
			}
			if step.State == "requested" || step.State == "uncertain" {
				if _, err := q.AIAgentSlot(ctx, store.AIAgentSlotParams{RunID: r.ID, Now: s.now()}); err == nil {
					busy = true
					return nil
				} else if !errors.Is(err, pgx.ErrNoRows) {
					return err
				}
				if err := q.AIBudgetUncertain(ctx, step.ID); err != nil {
					return err
				}
				return s.pauseAgent(ctx, q, &r, "usage_unknown")
			}
		}
		if step.Sequence >= 250 || r.ToolCalls >= r.ToolLimit {
			return s.pauseAgent(ctx, q, &r, "run_cap")
		}
		if err := q.AIAgentSlotsExpire(ctx, s.now()); err != nil {
			return err
		}
		lease = uuid.Must(uuid.NewV7())
		if _, err := q.AIAgentSlotClaim(ctx, store.AIAgentSlotClaimParams{UserID: r.UserID, WorkspaceID: ws.ID, RunID: r.ID, LeaseID: lease, ExpiresAt: s.now().Add(agentLease)}); errors.Is(err, pgx.ErrNoRows) {
			busy = true
			return nil
		} else if err != nil {
			return err
		}
		conn, err = s.load(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		req = llm.Request{System: agentSystem, Messages: c.Messages, Tools: agentTools(r.AcceptedRevision > 0, c.Audience != nil, c.Brief.Automation != nil, c.Report != nil), MaxTokens: 8000}
		t := task{feature: FeatureAgent, req: req}
		id := uuid.Must(uuid.NewV7())
		if err := s.reserveIn(ctx, q, &ws, owner, conn, &t, &r, id); err != nil {
			if releaseErr := q.AIAgentSlotRelease(ctx, store.AIAgentSlotReleaseParams{RunID: r.ID, LeaseID: lease}); releaseErr != nil {
				return releaseErr
			}
			return s.pauseAgentFor(ctx, q, &r, err)
		}
		req = t.req
		raw, err := json.Marshal(req)
		if err != nil {
			return err
		}
		step = store.AiAgentStep{ID: id, RunID: r.ID, RunVersion: r.Version, Sequence: step.Sequence + 1, CreatedAt: s.now()}
		if err := q.AIAgentStepInsert(ctx, store.AIAgentStepInsertParams{ID: id, RunID: r.ID, RunVersion: r.Version, Sequence: step.Sequence, Request: raw, Now: s.now()}); err != nil {
			return err
		}
		if err := q.AIAgentStepRequested(ctx, store.AIAgentStepRequestedParams{ID: id, Request: raw, Now: s.now()}); err != nil {
			return err
		}
		if r.AcceptedRevision > 0 {
			r.State = "building"
		}
		if err := s.saveAgentRun(ctx, q, &r); err != nil {
			return err
		}
		ready, send = true, true
		return nil
	})
	if err != nil {
		return err
	}
	if busy {
		return river.JobSnooze(5 * time.Second)
	}
	if !ready {
		return nil
	}
	if send {
		prov, callErr := s.provider(conn, FeatureAgent)
		res := llm.Response{}
		if callErr == nil {
			callCtx, cancel := context.WithTimeout(ctx, timeoutOf(FeatureAgent))
			res, callErr = prov.Generate(callCtx, req)
			cancel()
		} else {
			// Provider construction happens locally, before any HTTP call.
			res.UsageKnown = true
		}
		if err := s.recordAgentResponse(ctx, &ws, args, step, lease, conn, res, callErr); err != nil {
			return err
		}
	}
	return s.executeAgentResponse(ctx, &ws, args)
}

func (s *Service) recordAgentResponse(ctx context.Context, ws *store.Workspace, args AgentArgs, step store.AiAgentStep, lease uuid.UUID, conn *connection, res llm.Response, callErr error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.AIAgentRunLock(ctx, store.AIAgentRunLockParams{ID: args.RunID, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		// Match the request/decision order: current membership and resource
		// scope, then connection/budget. Accounting still happens when access
		// was revoked or cancellation won the run fence.
		active := r.Version == args.Version && agentRunning(r.State) && s.now().Before(r.ExpiresAt)
		var accessErr error
		var literals privateLiterals
		if active {
			c, err := loadAgentContext(ctx, q, r.ID, false)
			if err != nil {
				return err
			}
			literals = c.Literals
			_, accessErr = s.agentAccess(ctx, q, ws, agentOwner(r), r, c, true)
		}
		outcome := OutcomePending
		if !active {
			outcome = OutcomeCancelled
		} else if callErr != nil || accessErr != nil {
			outcome = OutcomeError
		} else if res.Refused {
			outcome = OutcomeRefused
		}
		known, err := s.accountIn(ctx, q, store.AIRequestInsertParams{ID: step.ID, WorkspaceID: ws.ID, UserID: &r.UserID, Feature: FeatureAgent,
			Provider: conn.Provider, Model: conn.Model, Outcome: outcome, CreatedAt: step.CreatedAt}, res, callErr)
		if err != nil {
			return err
		}
		slot, slotErr := q.AIAgentSlot(ctx, store.AIAgentSlotParams{RunID: r.ID, Now: s.now()})
		if slotErr != nil && !errors.Is(slotErr, pgx.ErrNoRows) {
			return slotErr
		}
		if err := q.AIAgentSlotRelease(ctx, store.AIAgentSlotReleaseParams{RunID: r.ID, LeaseID: lease}); err != nil {
			return err
		}
		// Even cancelled or expired calls are accounted. They cannot restore
		// content, stage a draft or advance a newer version of the run.
		if r.Version != args.Version || !agentRunning(r.State) || !s.now().Before(r.ExpiresAt) {
			return nil
		}
		if slotErr != nil || slot.LeaseID != lease {
			return s.pauseAgent(ctx, q, &r, "lease_expired")
		}
		if accessErr != nil {
			return s.pauseAgentFor(ctx, q, &r, accessErr)
		}
		if callErr != nil {
			if !known {
				return s.pauseAgent(ctx, q, &r, "usage_unknown")
			}
			return s.pauseAgent(ctx, q, &r, "provider")
		}
		if res.Refused || !safeAgentResponse(res, literals) {
			return s.pauseAgent(ctx, q, &r, "refused")
		}
		if res.StopReason != llm.StopEnd && res.StopReason != llm.StopTools {
			return s.pauseAgent(ctx, q, &r, "incomplete_response")
		}
		raw, err := json.Marshal(res)
		if err != nil {
			return err
		}
		if len(raw) > maxModelContextBytes {
			return s.pauseAgent(ctx, q, &r, "context_limit")
		}
		if err := q.AIAgentStepSave(ctx, store.AIAgentStepSaveParams{ID: step.ID, State: "responded", Response: raw, Results: json.RawMessage(`[]`), Summary: json.RawMessage(`{}`), UpdatedAt: s.now()}); err != nil {
			return err
		}
		if !known {
			return s.pauseAgent(ctx, q, &r, "usage_unknown")
		}
		u, err := q.AIBudgetRun(ctx, &r.ID)
		if err != nil {
			return err
		}
		if u.Tokens > r.TokenLimit || u.CostMicros > r.CostLimitMicros {
			return s.pauseAgent(ctx, q, &r, "run_cap")
		}
		return nil
	})
	if err == nil {
		if e, ok := llm.AsError(callErr); ok && e.Kind == llm.ErrAuth {
			s.keyRejected(ctx, ws, conn, e)
		}
		s.afterUse(ctx, ws)
	}
	return err
}

// Each tool write and its recorded result share one transaction. Repeating
// this method after commit but before acknowledgment cannot repeat the write.
func (s *Service) executeAgentResponse(ctx context.Context, ws *store.Workspace, args AgentArgs) error {
	for {
		more := false
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			r, err := q.AIAgentRunLock(ctx, store.AIAgentRunLockParams{ID: args.RunID, WorkspaceID: ws.ID})
			if err != nil {
				return err
			}
			if r.Version != args.Version || !s.now().Before(r.ExpiresAt) || !agentRunning(r.State) {
				return nil
			}
			step, err := q.AIAgentStepLast(ctx, r.ID)
			if err != nil {
				return err
			}
			if step.State != "responded" || step.RunVersion != args.Version {
				return nil
			}
			c, err := loadAgentContext(ctx, q, r.ID, true)
			if err != nil {
				return err
			}
			p, err := s.agentAccess(ctx, q, ws, agentOwner(r), r, c, true)
			if err != nil {
				return s.pauseAgentFor(ctx, q, &r, err)
			}
			var res llm.Response
			if err := json.Unmarshal(step.Response, &res); err != nil {
				return err
			}
			calls := []llm.ToolCall{}
			for _, part := range res.Message.Parts {
				if part.ToolCall != nil {
					calls = append(calls, *part.ToolCall)
				}
			}
			var results []llm.Part
			if err := json.Unmarshal(step.Results, &results); err != nil {
				return err
			}
			if int(step.NextCall) < len(calls) {
				if r.ToolCalls >= r.ToolLimit {
					return s.pauseAgent(ctx, q, &r, "run_cap")
				}
				call := calls[step.NextCall]
				r.ToolCalls++
				if c.Failures == nil {
					c.Failures = map[string]int{}
				}
				key := agentFailureKey(c.Plan, call)
				content, failed := json.RawMessage(`{"ok":false,"code":"validation_exhausted"}`), true
				if c.Failures[key] < 3 {
					content, failed, err = s.executeAgentTool(ctx, q, ws, p, &r, &c, call)
					if err != nil {
						return err
					}
				}
				if failed {
					c.Failures[key] = min(3, c.Failures[key]+1)
				} else {
					delete(c.Failures, key)
				}
				results = append(results, llm.Part{ToolResult: &llm.ToolResult{ID: call.ID, Name: call.Name, Content: content, IsError: failed}})
				step.NextCall++
				step.Results, err = json.Marshal(results)
				if err != nil {
					return err
				}
				if err := q.AIAgentStepSave(ctx, store.AIAgentStepSaveParams{ID: step.ID, State: "responded", Response: step.Response, Results: step.Results,
					NextCall: step.NextCall, Summary: agentStepSummary(c.Plan, calls, results), UpdatedAt: s.now()}); err != nil {
					return err
				}
				if err := saveAgentContext(ctx, q, r.ID, c); err != nil {
					return err
				}
				if err := s.saveAgentRun(ctx, q, &r); err != nil {
					return err
				}
				more = true
				return nil
			}
			c.Messages = append(c.Messages, res.Message)
			if len(results) > 0 {
				c.Messages = append(c.Messages, llm.Message{Role: "tool", Parts: results})
			}
			var visible strings.Builder
			for _, part := range res.Message.Parts {
				if !part.Thought && part.Text != "" {
					visible.WriteString(part.Text)
					visible.WriteByte('\n')
				}
			}
			if text, safe := voiceText(visible.String()); safe && strings.TrimSpace(text) != "" {
				c.Conversation = append(c.Conversation, AIAgentMessage{Role: "assistant", Text: clip(strings.TrimSpace(text), 2000), Handle: c.Refine, At: s.now()})
			}
			if r.AcceptedRevision == 0 {
				for _, result := range results {
					if result.ToolResult.Name == "propose_plan" && !result.ToolResult.IsError {
						r.State = "ready"
						if len(c.Plan.Questions) > 0 {
							r.State = "needs_answers"
						}
					}
				}
			}
			if len(calls) == 0 {
				artifacts, err := q.AIAgentArtifacts(ctx, r.ID)
				if err != nil {
					return err
				}
				missing := slices.ContainsFunc(c.Plan.Items, func(item AIAgentPlanItem) bool {
					return !c.Discarded[item.Bundle] && !slices.ContainsFunc(artifacts, func(a store.AiAgentArtifact) bool { return a.Handle == item.Handle })
				})
				if r.AcceptedRevision == 0 || missing {
					r.State, r.PauseReason = "paused", "incomplete_plan"
				} else {
					r.State = "review"
					if !slices.ContainsFunc(artifacts, func(a store.AiAgentArtifact) bool { return a.Decision == "" }) {
						r.State, r.ClosedAt = "completed", ptr(s.now())
					}
				}
			}
			// Finish the saved response before stopping: a failed output must
			// not throw away independent successful drafts later in that response.
			if c.exhausted() {
				r.State, r.PauseReason = "paused", "validation_exhausted"
			}
			if err := q.AIAgentStepSave(ctx, store.AIAgentStepSaveParams{ID: step.ID, State: "done", Response: step.Response, Results: step.Results,
				NextCall: step.NextCall, Summary: agentStepSummary(c.Plan, calls, results), UpdatedAt: s.now()}); err != nil {
				return err
			}
			if err := saveAgentContext(ctx, q, r.ID, c); err != nil {
				return err
			}
			r.Version++
			if err := s.saveAgentRun(ctx, q, &r); err != nil {
				return err
			}
			if agentRunning(r.State) {
				return s.enqueueAgent(ctx, tx, r)
			}
			if err := s.agentAudit(ctx, q, p, r, r.State, nil); err != nil {
				return err
			}
			return notifyAgent(ctx, q, r)
		})
		if err != nil {
			return err
		}
		if !more {
			return nil
		}
	}
}
