package ai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Agent runs never store product objects until a human keeps their staging
// bundle. Resource IDs and bindings below stay local, outside model messages.
type AIAgentBrief struct {
	SharingReviewed bool                        `json:"sharingReviewed" doc:"Confirm review of the brief, selected content, residual personal-data risk and provider data policy before sharing"`
	Brief           string                      `json:"brief" minLength:"1" maxLength:"4000"`
	Language        string                      `json:"language" enum:"en,vi,de,es,fr"`
	Kind            string                      `json:"kind,omitempty" enum:"campaign,post" doc:"Primary draft kind; campaign when omitted"`
	CampaignID      *uuid.UUID                  `json:"campaignId,omitempty"`
	BaseRevision    int64                       `json:"baseRevision,omitempty" minimum:"0"`
	TemplateID      string                      `json:"templateId,omitempty" maxLength:"100"`
	Links           []string                    `json:"links,omitempty" maxItems:"10"`
	VoiceRevision   *int32                      `json:"voiceRevision,omitempty" minimum:"0"`
	Audience        *segments.SegmentDefinition `json:"audience,omitempty" doc:"Explicit builder conditions; literal values stay on this instance"`
	Automation      *AIAgentAutomationSelection `json:"automation,omitempty"`
	ReportID        *uuid.UUID                  `json:"reportId,omitempty" doc:"Explicitly selected completed campaign; only its fixed aggregate snapshot reaches the model"`
	PoolID          *uuid.UUID                  `json:"poolId,omitempty" doc:"Selected discount pool for new emails; codes are never read or reserved"`
	RequestKey      string                      `json:"requestKey,omitempty" pattern:"^[A-Za-z0-9_-]{1,80}$" doc:"Generated once per brief by the client: retrying the same brief with it after a lost response returns the run already created"`
}

type AIAgentPlanItem struct {
	Handle    string   `json:"handle" pattern:"^[a-z][a-z0-9_]{0,39}$"`
	Kind      string   `json:"kind" enum:"campaign,segment,automation,post"`
	Name      string   `json:"name" minLength:"1" maxLength:"200"`
	Purpose   string   `json:"purpose" minLength:"1" maxLength:"1000"`
	Bundle    string   `json:"bundle" pattern:"^[a-z][a-z0-9_]{0,39}$"`
	Target    string   `json:"target,omitempty" maxLength:"40"`
	DependsOn []string `json:"dependsOn" maxItems:"10" uniqueItems:"true"`
}

type AIAgentPlan struct {
	Summary   string            `json:"summary" minLength:"1" maxLength:"2000"`
	Items     []AIAgentPlanItem `json:"items" maxItems:"10"`
	Questions []string          `json:"questions" maxItems:"3"`
}

type AIAgentLimits struct {
	ToolCalls  int32 `json:"toolCalls" minimum:"1" maximum:"200"`
	Tokens     int64 `json:"tokens" minimum:"1000" maximum:"2000000"`
	CostMicros int64 `json:"costMicros" minimum:"1" maximum:"100000000"`
}

type AIAgentSettings struct {
	Enabled    bool               `json:"enabled"`
	Limits     AIAgentLimits      `json:"limits"`
	Revision   int64              `json:"revision"`
	Capability *AIAgentCapability `json:"capability,omitempty" readOnly:"true"`
}

type AIAgentLimitChange struct {
	Version int64         `json:"version" minimum:"1"`
	Limits  AIAgentLimits `json:"limits"`
}

// Cost scenarios are planning aids, not reservations or invoice guarantees.
// They exclude work already staged and use the connection's current prices.
type AIAgentCostEstimate struct {
	LowMicros  int64     `json:"lowMicros"`
	HighMicros int64     `json:"highMicros"`
	PricingAt  time.Time `json:"pricingAt"`
}

type AIAgentRunView struct {
	ID               uuid.UUID                 `json:"id"`
	OwnerID          uuid.UUID                 `json:"ownerId"`
	State            string                    `json:"state" enum:"brief,planning,needs_answers,ready,queued,building,paused,review,completed,cancelled,expired"`
	Version          int64                     `json:"version"`
	PlanRevision     int64                     `json:"planRevision"`
	AcceptedRevision int64                     `json:"acceptedRevision"`
	PauseReason      string                    `json:"pauseReason"`
	Limits           AIAgentLimits             `json:"limits"`
	ToolCalls        int32                     `json:"toolCalls"`
	Tokens           int64                     `json:"tokens"`
	CostMicros       int64                     `json:"costMicros"`
	Uncertain        bool                      `json:"uncertain"`
	Model            string                    `json:"model"`
	QueuePosition    int64                     `json:"queuePosition,omitempty" doc:"Waiting position snapshot; per-user execution limits can change ordering"`
	CreatedAt        time.Time                 `json:"createdAt"`
	UpdatedAt        time.Time                 `json:"updatedAt"`
	ExpiresAt        time.Time                 `json:"expiresAt"`
	Brief            *AIAgentBrief             `json:"brief,omitempty"`
	Plan             *AIAgentPlan              `json:"plan,omitempty"`
	PlanDiscards     []AIAgentArtifactRevision `json:"planDiscards" doc:"Exact staging revisions that confirming this plan discards; kept resources are never included"`
	Audience         *AIAgentAudienceSnapshot  `json:"audience,omitempty"`
	Report           *reports.ModelSnapshot    `json:"report,omitempty"`
	Pool             *AIAgentPoolSnapshot      `json:"pool,omitempty"`
	Estimate         *AIAgentCostEstimate      `json:"estimate,omitempty" doc:"Remaining build scenarios, not a quote or spending authority; absent when pricing or the model connection is unavailable"`
	Artifacts        []AIAgentArtifact         `json:"artifacts"`
	Steps            []AIAgentStepView         `json:"steps"`
	Conversation     []AIAgentMessage          `json:"conversation"`
}

// Conversation contains only visible user/assistant text. Provider messages,
// function arguments and private thought parts remain separate checkpoints.
type AIAgentMessage struct {
	Role   string    `json:"role" enum:"user,assistant"`
	Text   string    `json:"text"`
	Handle string    `json:"handle,omitempty"`
	At     time.Time `json:"at"`
}

// A public step names operations and fixed outcomes, never provider messages,
// arguments, reasoning, credentials or a raw validator/database error.
type AIAgentStepView struct {
	Sequence   int32              `json:"sequence"`
	State      string             `json:"state"`
	CreatedAt  time.Time          `json:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt"`
	Operations []AIAgentOperation `json:"operations"`
}

type AIAgentOperation struct {
	Tool   string `json:"tool" enum:"propose_plan,stage_campaign,stage_segment,stage_automation,stage_post,inspect_audience,read_report,denied"`
	Handle string `json:"handle,omitempty"`
	Code   string `json:"code"`
	Failed bool   `json:"failed"`
}

type AIAgentArtifact struct {
	ID               uuid.UUID                    `json:"id"`
	Handle           string                       `json:"handle"`
	Kind             string                       `json:"kind" enum:"campaign,segment,automation,post"`
	Bundle           string                       `json:"bundle"`
	Revision         int64                        `json:"revision"`
	Decision         string                       `json:"decision"`
	TargetID         *uuid.UUID                   `json:"targetId,omitempty"`
	ResultID         *uuid.UUID                   `json:"resultId,omitempty"`
	Name             string                       `json:"name"`
	Before           *AIDraftEmail                `json:"before,omitempty"`
	Email            *AIDraftEmail                `json:"email,omitempty"`
	Post             *AIAgentPost                 `json:"post,omitempty"`
	HTML             string                       `json:"html,omitempty"`
	Segment          *segments.SegmentDefinition  `json:"segment,omitempty"`
	Automation       *AIAutomationContent         `json:"automation,omitempty"`
	TriggerName      string                       `json:"triggerName,omitempty" doc:"Human review only; the selected list or segment name is never model context"`
	Previews         []string                     `json:"previews,omitempty"`
	BeforeAutomation *automations.AutomationGraph `json:"beforeAutomation,omitempty"`
	BeforeEmails     []AIAutomationEmailBefore    `json:"beforeEmails,omitempty"`
}

type AIAgentRunList struct {
	Items []AIAgentRunListItem `json:"items"`
	Next  string               `json:"next,omitempty"`
}

type AIAgentRunListItem struct {
	ID         uuid.UUID `json:"id"`
	OwnerID    uuid.UUID `json:"ownerId"`
	OwnerName  string    `json:"ownerName"`
	Title      string    `json:"title"`
	Model      string    `json:"model"`
	State      string    `json:"state"`
	UpdatedAt  time.Time `json:"updatedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Tokens     int64     `json:"tokens"`
	CostMicros int64     `json:"costMicros"`
	Uncertain  bool      `json:"uncertain"`
}

type AIAgentAction struct {
	SharingReviewed bool                      `json:"sharingReviewed,omitempty" doc:"Required for answer, revise, replan, propose and refresh: confirm review of newly shared content and the data policy"`
	Version         int64                     `json:"version" minimum:"1"`
	PlanRevision    int64                     `json:"planRevision" minimum:"0"`
	Action          string                    `json:"action" enum:"build,cancel,continue,answer,revise,replan,propose,refresh"`
	Message         string                    `json:"message,omitempty" maxLength:"4000"`
	Handle          string                    `json:"handle,omitempty" pattern:"^[a-z][a-z0-9_]{0,39}$"`
	PlanDiscards    []AIAgentArtifactRevision `json:"planDiscards,omitempty" maxItems:"10" doc:"Build only: explicitly acknowledge all replaced staging revisions shown by the run"`
}

type agentContext struct {
	Brief        AIAgentBrief
	Plan         AIAgentPlan
	PreviousPlan *AIAgentPlan
	Source       campaignDraftTask
	Draft        draftContext
	Messages     []llm.Message
	Failures     map[string]int
	Actions      map[int64][32]byte
	Discarded    map[string]bool
	Model        agentModel
	Conversation []AIAgentMessage
	Proposals    map[string]agentProposalSource
	Audience     *agentAudience
	Report       *reports.ModelSnapshot
	Pool         *AIAgentPoolSnapshot
	Literals     privateLiterals
	// Refine confines copy editing to one still-unkept draft. A scope
	// change instead clears the accepted revision and requires another Build.
	Refine string
}

type agentProposalSource struct {
	ID       uuid.UUID
	Revision int64
	Draft    draftContext
	Kind     string
	Snapshot string
	Flow     *agentFlowSource
}

func (s agentProposalSource) kind() string {
	if s.Kind == "" {
		return "campaign"
	}
	return s.Kind
}

// Provider context (notably Gemini signatures) must never be replayed to a
// different account endpoint or model after a connection is reconfigured.
type agentModel struct {
	Provider, Model, BaseURL, Deployment, APIVersion string
}

func agentModelOf(c store.AIBindingRow) agentModel {
	return agentModel{c.Provider, c.Model, c.BaseUrl, c.Deployment, c.ApiVersion}
}

// agentModel is the model a run uses: the one it started with, kept when
// the agent's default changes; a new run takes the agent's default. Nil when
// the run's model was removed.
func (s *Service) agentModel(ctx context.Context, q *store.Queries, ws uuid.UUID, r store.AiAgentRun, c agentContext) (*connection, error) {
	if r.ModelID == nil && c.Model.Provider == "" {
		return s.loadFor(ctx, q, ws, FeatureAgent)
	}
	return s.binding(ctx, q, ws, r.ModelID)
}

func agentBuildEstimate(c agentContext, artifacts []store.AiAgentArtifact, conn *connection) *AIAgentCostEstimate {
	if !conn.connected() || (!conn.PriceKnown && !conn.Demo) || c.Model != agentModelOf(conn.AIBindingRow) {
		return nil
	}
	outputs := []int64{}
	for _, item := range c.Plan.Items {
		if slices.ContainsFunc(artifacts, func(a store.AiAgentArtifact) bool {
			return a.Handle == item.Handle || (a.Bundle == item.Bundle && a.Decision == "discarded")
		}) {
			continue
		}
		output := int64(2000)
		switch item.Kind {
		case "segment":
			output = 500
		case "automation":
			output = 8000
		}
		outputs = append(outputs, output)
	}
	estimate := &AIAgentCostEstimate{PricingAt: conn.ModelUpdatedAt}
	if len(outputs) == 0 {
		return estimate
	}
	req := llm.Request{System: agentSystem, Messages: c.Messages, Tools: agentTools(true, c.Audience != nil, c.Brief.Automation != nil, c.Report != nil), MaxTokens: 8000}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil
	}
	// Scenarios, not statistical confidence bounds: one request per missing
	// item plus a recap, versus two repairs per item and full 8k responses.
	// Both include resent context; the high scenario uses the reservation's
	// byte-based input estimate. Never clamp the quote to remaining budget:
	// a plan larger than its ceiling must remain visibly larger.
	lowInput := int64((len(raw)+3)/4 + 450)
	highInput := int64(len(raw) + 4096 + len(req.Messages)*128 + len(req.Tools)*512)
	var low, high float64
	for _, output := range append(outputs, 200) {
		low += float64(lowInput)*conn.InputPrice + float64(output)*conn.OutputPrice
		lowInput += output + 128
	}
	for i := 0; i < len(outputs)*3+1; i++ {
		high += float64(highInput)*conn.InputPrice + 8000*conn.OutputPrice
		highInput += 8000*4 + 512
	}
	estimate.LowMicros, estimate.HighMicros = int64(math.Ceil(low)), int64(math.Ceil(high))
	return estimate
}

func agentProblem(ctx context.Context, key string) error {
	return api.Err(409, "ai_agent_"+key, text(ctx, msgs.M("agent."+key), nil))
}

func agentLimits(r store.AiAgentRun) AIAgentLimits {
	return AIAgentLimits{ToolCalls: r.ToolLimit, Tokens: r.TokenLimit, CostMicros: r.CostLimitMicros}
}

func (s *Service) AgentSettings(ctx context.Context, ws *store.Workspace) (AIAgentSettings, error) {
	r, err := s.Q.AIAgentSettings(ctx, ws.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AIAgentSettings{}, err
	}
	out := AIAgentSettings{Enabled: r.Enabled, Limits: AIAgentLimits{ToolCalls: r.ToolLimit, Tokens: r.TokenLimit, CostMicros: r.CostLimitMicros}, Revision: r.Revision}
	if errors.Is(err, pgx.ErrNoRows) {
		out.Limits = AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}
	}
	c, err := s.loadFor(ctx, s.Q, ws.ID, FeatureAgent)
	if err != nil {
		return AIAgentSettings{}, err
	}
	capability, err := agentCapability(ctx, s.Q, c)
	out.Capability = &capability
	return out, err
}

func (s *Service) SetAgentSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIAgentSettings) error {
	if err := validateDraftJSON(in); err != nil {
		return agentProblem(ctx, "input")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, false, authz.IntegrationsManage)
		if err != nil {
			return err
		}
		// The AI settings row is the shared configuration lock, including the
		// first insertion where no agent settings row exists to lock yet.
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return err
		}
		if _, err := q.AIPrimaryModel(ctx, ws.ID); err != nil {
			return api.MapError(err, "AI connection")
		}
		cur, err := q.AIAgentSettings(ctx, ws.ID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if cur.Revision != in.Revision {
			return agentProblem(ctx, "changed")
		}
		if err := q.AIAgentSettingsSave(ctx, store.AIAgentSettingsSaveParams{WorkspaceID: ws.ID, Enabled: in.Enabled,
			ToolLimit: in.Limits.ToolCalls, TokenLimit: in.Limits.Tokens, CostLimitMicros: in.Limits.CostMicros, UpdatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(audit.WithPrincipal(ctx, fresh), q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.agent.settings", TargetType: "ai_assistant", TargetID: ws.ID.String(),
			After: map[string]any{"enabled": in.Enabled, "limits": in.Limits}})
	})
}

func loadAgentContext(ctx context.Context, q *store.Queries, id uuid.UUID, messages bool) (agentContext, error) {
	var c agentContext
	r, err := q.AIAgentContext(ctx, store.AIAgentContextParams{RunID: id, WithMessages: messages})
	if err != nil {
		return c, err
	}
	for _, part := range []struct {
		raw  []byte
		dest any
	}{{r.Bindings, &c}, {r.Brief, &c.Brief}, {r.Plan, &c.Plan}, {r.Messages, &c.Messages}} {
		if err := json.Unmarshal(part.raw, part.dest); err != nil {
			return c, err
		}
	}
	return c, nil
}

func saveAgentContext(ctx context.Context, q *store.Queries, id uuid.UUID, c agentContext) error {
	brief, err := json.Marshal(c.Brief)
	if err != nil {
		return err
	}
	plan, err := json.Marshal(c.Plan)
	if err != nil {
		return err
	}
	messages, err := json.Marshal(c.Messages)
	if err != nil {
		return err
	}
	c.Brief, c.Plan, c.Messages = AIAgentBrief{}, AIAgentPlan{}, nil
	bindings, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return q.AIAgentContextSave(ctx, store.AIAgentContextSaveParams{RunID: id, Brief: brief, Plan: plan, Bindings: bindings, Messages: messages})
}

func (s *Service) saveAgentRun(ctx context.Context, q *store.Queries, r *store.AiAgentRun) error {
	r.UpdatedAt = s.now()
	return q.AIAgentRunSave(ctx, store.AIAgentRunSaveParams{ID: r.ID, State: r.State, Version: r.Version, PlanRevision: r.PlanRevision,
		AcceptedRevision: r.AcceptedRevision, ToolLimit: r.ToolLimit, TokenLimit: r.TokenLimit, CostLimitMicros: r.CostLimitMicros,
		ToolCalls: r.ToolCalls, PauseReason: r.PauseReason, UpdatedAt: r.UpdatedAt, ClosedAt: r.ClosedAt})
}

func agentKindAccess(p *authz.Principal, kind string, edit bool) error {
	var read, write authz.Permission
	switch kind {
	case "campaign":
		read, write = authz.CampaignsView, authz.CampaignsEdit
	case "automation":
		read, write = authz.AutomationsView, authz.AutomationsEdit
	case "segment":
		read, write = authz.SubscribersView, authz.SegmentsEdit
	case "post":
		read, write = authz.WebsiteView, authz.WebsiteEdit
	default:
		return api.Forbidden(authz.AIUse, nil)
	}
	if !p.Can(read) || (kind == "segment" && len(p.Scope.Lists) != 0) {
		return api.Forbidden(read, nil)
	}
	if edit && !p.Can(write) {
		return api.Forbidden(write, nil)
	}
	return nil
}

// agentAccess checks current membership and every source, even for managers.
// Managers can inspect another person's run; they cannot act as its author.
func (s *Service) agentAccess(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, r store.AiAgentRun, c agentContext, execute bool) (*authz.Principal, error) {
	fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
	if err != nil {
		return nil, err
	}
	if fresh.UserID != r.UserID && (!fresh.Can(authz.IntegrationsManage) || execute) {
		return nil, api.NotFound("AI run")
	}
	if execute && !fresh.Can(authz.AIUse) {
		return nil, agentProblem(ctx, "permission")
	}
	if err := c.poolAccess(ctx, q, ws, fresh, execute); err != nil {
		return nil, err
	}
	if c.Brief.CampaignID != nil || (c.Brief.Kind != "post" && c.Brief.Brief != "" && c.Brief.Audience == nil && c.Brief.Automation == nil) {
		if err := agentKindAccess(fresh, "campaign", execute); err != nil {
			return nil, err
		}
	}
	if c.Brief.Kind == "post" {
		if err := agentKindAccess(fresh, "post", execute); err != nil {
			return nil, err
		}
	}
	for _, item := range c.Plan.Items {
		if err := agentKindAccess(fresh, item.Kind, execute); err != nil {
			return nil, err
		}
	}
	if c.Brief.Audience != nil {
		if !fresh.Can(authz.SubscribersView) || len(fresh.Scope.Lists) != 0 {
			return nil, api.Forbidden(authz.SubscribersView, nil)
		}
		if execute {
			if !fresh.Can(authz.SegmentsEdit) {
				return nil, api.Forbidden(authz.SegmentsEdit, nil)
			}
			if _, err := s.Segments.CheckProposal(ctx, q, ws.ID, fresh, *c.Brief.Audience); err != nil {
				// A field or list deleted since selection is a domain denial
				// that pauses the run, not an operational failure to retry.
				return nil, segments.Problem(ctx, err)
			}
		}
	}
	if c.Brief.Automation != nil {
		if err := s.agentAutomationAccess(ctx, q, ws, fresh, c, execute); err != nil {
			return nil, err
		}
	}
	// Kept objects are ordinary resources: their current scope still gates
	// access to this run. Different runs may select the same resources in a
	// different order, so lock every reference by kind and ID first.
	type reference struct {
		kind string
		id   uuid.UUID
		// optional references may since have been deleted. A deleted
		// resource no longer gates viewing or deciding the rest of the run,
		// nor a step after its draft was decided; an existing one's scope
		// still does.
		optional bool
	}
	refs := []reference{}
	if c.Brief.CampaignID != nil {
		refs = append(refs, reference{"campaign", *c.Brief.CampaignID, !execute})
	}
	for _, source := range c.Proposals {
		refs = append(refs, reference{source.kind(), source.ID, !execute})
	}
	if c.Brief.ReportID != nil {
		refs = append(refs, reference{"report", *c.Brief.ReportID, !execute})
	}
	artifacts, err := q.AIAgentReferences(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		if err := agentKindAccess(fresh, a.Kind, false); err != nil {
			return nil, err
		}
		for _, id := range []*uuid.UUID{a.TargetID, a.ResultID} {
			if id != nil && a.Kind != "segment" {
				refs = append(refs, reference{a.Kind, *id, !execute || a.Decision != ""})
			}
		}
	}
	gone := func(kind string, id uuid.UUID) (bool, error) {
		present, err := q.AIAgentReferenceExists(ctx, store.AIAgentReferenceExistsParams{Kind: kind, WorkspaceID: ws.ID, ID: id})
		return !present, err
	}
	// Required references sort first, so compaction keeps them.
	slices.SortFunc(refs, func(a, b reference) int {
		if kind := strings.Compare(a.kind, b.kind); kind != 0 {
			return kind
		}
		if id := strings.Compare(a.id.String(), b.id.String()); id != 0 {
			return id
		}
		if a.optional == b.optional {
			return 0
		}
		if a.optional {
			return 1
		}
		return -1
	})
	for _, ref := range slices.CompactFunc(refs, func(a, b reference) bool { return a.kind == b.kind && a.id == b.id }) {
		if ref.optional {
			deleted, err := gone(ref.kind, ref.id)
			if err != nil {
				return nil, err
			}
			if deleted {
				continue
			}
		}
		if ref.kind == "automation" {
			if _, _, err := s.agentFlow(ctx, q, ws, fresh, ref.id); err != nil {
				return nil, err
			}
			continue
		}
		if ref.kind == "post" {
			if _, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: ref.id}); err != nil {
				return nil, api.MapError(err, "Post")
			}
			continue
		}
		if ref.kind == "report" {
			if _, err := reports.ModelCampaign(ctx, q, fresh, ref.id); err != nil {
				return nil, api.Forbidden(authz.ReportsView, nil)
			}
			continue
		}
		id := ref.id
		row, err := draftCampaign(ctx, q, ws, fresh, id)
		if err != nil {
			return nil, err
		}
		if execute && c.Brief.CampaignID != nil && id == *c.Brief.CampaignID && (row.Status != "draft" || int64(row.Version) != c.Brief.BaseRevision) {
			// Once the original proposal is decided, only a freshly selected
			// proposal source can authorize another edit to this resource.
			if !slices.ContainsFunc(artifacts, func(a store.AIAgentReferencesRow) bool {
				return a.TargetID != nil && *a.TargetID == id && a.Decision != ""
			}) {
				return nil, draftProblem(ctx, 409, "changed")
			}
		}
		if execute {
			for handle, source := range c.Proposals {
				if source.kind() != "campaign" || source.ID != id || !slices.ContainsFunc(c.Plan.Items, func(item AIAgentPlanItem) bool { return item.Target == handle }) {
					continue
				}
				decided := slices.ContainsFunc(artifacts, func(a store.AIAgentReferencesRow) bool { return a.Handle == handle && a.Decision != "" })
				if !decided && (row.Status != "draft" || int64(row.Version) != source.Revision) {
					return nil, draftProblem(ctx, 409, "changed")
				}
			}
		}
	}
	for handle, source := range c.Proposals {
		if source.kind() == "campaign" {
			continue
		}
		pending := slices.ContainsFunc(c.Plan.Items, func(item AIAgentPlanItem) bool { return item.Target == handle }) &&
			!slices.ContainsFunc(artifacts, func(a store.AIAgentReferencesRow) bool { return a.Handle == handle && a.Decision != "" })
		if !execute {
			deleted, err := gone(source.kind(), source.ID)
			if err != nil {
				return nil, err
			}
			if deleted {
				continue
			}
		}
		if err := s.agentProposalAccess(ctx, q, ws, fresh, source, execute && pending); err != nil {
			return nil, err
		}
	}
	if err := c.Draft.checkTemplate(ctx, q, ws, fresh, execute); err != nil {
		return nil, err
	}
	if execute {
		// Reading under the configuration lock fences switch changes with the
		// same lock used when a request reserves or a response commits.
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return nil, err
		}
		conn, err := s.agentModel(ctx, q, ws.ID, r, c)
		if err != nil {
			return nil, err
		}
		if conn == nil && r.ModelID == nil && c.Model.Provider == "" {
			return nil, api.NotFound("AI connection")
		}
		if conn == nil {
			return nil, budgetProblem(ctx, "connection_changed")
		}
		settings, err := q.AIAgentSettings(ctx, ws.ID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if !settings.Enabled || !conn.connected() {
			return nil, agentProblem(ctx, "unavailable")
		}
		capability, err := agentCapability(ctx, q, conn)
		if err != nil {
			return nil, err
		}
		if capability.State == "unsupported" {
			return nil, agentProblem(ctx, "tools_unavailable")
		}
		if c.Model.Provider != "" && c.Model != agentModelOf(conn.AIBindingRow) {
			return nil, budgetProblem(ctx, "connection_changed")
		}
	}
	return fresh, nil
}

func (s *Service) AgentRuns(ctx context.Context, ws *store.Workspace, p *authz.Principal, cursor string) (AIAgentRunList, error) {
	out := AIAgentRunList{Items: []AIAgentRunListItem{}}
	fresh, err := brandkit.VoiceActor(ctx, s.Q, ws, p)
	if err != nil {
		return out, err
	}
	before, beforeID := parseCursor(cursor)
	if cursor != "" && before == nil {
		return out, agentProblem(ctx, "input")
	}
	filter := store.AIAgentRunsParams{WorkspaceID: ws.ID, Before: before, BeforeID: beforeID, Lim: 100}
	if !fresh.Can(authz.IntegrationsManage) {
		filter.UserID = &fresh.UserID
	}
	rows, err := s.Q.AIAgentRuns(ctx, filter)
	if err != nil {
		return out, err
	}
	for i, r := range rows {
		view, err := s.agentRun(ctx, ws, fresh, r.ID, false)
		if err != nil {
			var problem *api.Problem
			if !errors.As(err, &problem) || (problem.Status != 403 && problem.Status != 404) {
				return out, err
			}
		} else {
			title := ""
			if view.Plan != nil {
				title = view.Plan.Summary
			} else if view.Brief != nil {
				title = view.Brief.Brief
			}
			out.Items = append(out.Items, AIAgentRunListItem{ID: r.ID, OwnerID: r.UserID, OwnerName: r.OwnerName, Title: title, Model: view.Model, State: view.State,
				UpdatedAt: view.UpdatedAt, ExpiresAt: view.ExpiresAt, Tokens: view.Tokens, CostMicros: view.CostMicros, Uncertain: view.Uncertain})
		}
		// Bound authorization work too. A page may contain fewer visible
		// rows; its cursor continues past inaccessible runs without exposing them.
		if len(out.Items) == 20 || i == 99 {
			out.Next = cursorOf(r.UpdatedAt, r.ID)
			break
		}
	}
	return out, nil
}

func (s *Service) AgentRun(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (AIAgentRunView, error) {
	return s.agentRun(ctx, ws, p, id, true)
}

func (s *Service) agentRun(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, detailed bool) (AIAgentRunView, error) {
	var out AIAgentRunView
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.AIAgentRunLock(ctx, store.AIAgentRunLockParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "AI run")
		}
		c, err := loadAgentContext(ctx, q, id, false)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err := s.agentAccess(ctx, q, ws, p, r, c, false); err != nil {
			return err
		}
		out = AIAgentRunView{ID: id, OwnerID: r.UserID, State: r.State, Version: r.Version, PlanRevision: r.PlanRevision, AcceptedRevision: r.AcceptedRevision,
			PauseReason: r.PauseReason, Limits: agentLimits(r), ToolCalls: r.ToolCalls, Model: r.Model, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, ExpiresAt: r.ExpiresAt, PlanDiscards: []AIAgentArtifactRevision{}, Artifacts: []AIAgentArtifact{}, Steps: []AIAgentStepView{}, Conversation: []AIAgentMessage{}}
		u, err := q.AIBudgetRun(ctx, &id)
		if err != nil {
			return err
		}
		out.Tokens, out.CostMicros, out.Uncertain = u.Tokens, u.CostMicros, u.Uncertain > 0
		if !s.now().Before(r.ExpiresAt) {
			out.State = "expired"
			return nil
		}
		if detailed && agentRunning(r.State) {
			queue, err := q.AIAgentQueueStatus(ctx, store.AIAgentQueueStatusParams{ID: r.ID, WorkspaceID: ws.ID, CreatedAt: r.CreatedAt, Now: s.now()})
			if err != nil {
				return err
			}
			if !queue.Executing {
				out.QueuePosition = queue.Position
			}
		}
		out.Brief = &c.Brief
		if r.PlanRevision > 0 {
			out.Plan = &c.Plan
		}
		if !detailed {
			return nil
		}
		if c.Audience != nil {
			out.Audience = &c.Audience.AIAgentAudienceSnapshot
		}
		out.Report = c.Report
		out.Pool = c.Pool
		out.Conversation = c.Conversation
		if out.Conversation == nil {
			out.Conversation = []AIAgentMessage{}
		}
		steps, err := q.AIAgentSteps(ctx, id)
		if err != nil {
			return err
		}
		for _, step := range steps {
			view := AIAgentStepView{Sequence: step.Sequence, State: step.State, CreatedAt: step.CreatedAt, UpdatedAt: step.UpdatedAt, Operations: []AIAgentOperation{}}
			var summary struct {
				Operations []AIAgentOperation `json:"operations"`
			}
			if err := json.Unmarshal(step.Summary, &summary); err != nil {
				return err
			}
			if summary.Operations != nil {
				view.Operations = summary.Operations
			}
			out.Steps = append(out.Steps, view)
		}
		artifacts, err := q.AIAgentArtifacts(ctx, id)
		if err != nil {
			return err
		}
		if r.State == "ready" && len(c.Plan.Questions) == 0 {
			conn, err := s.binding(ctx, q, ws.ID, r.ModelID)
			if err != nil {
				return err
			}
			// Ordinary run reads avoid the model transcript. A build quote
			// must count it because the next request sends the full context.
			priced, err := loadAgentContext(ctx, q, id, true)
			if err != nil {
				return err
			}
			out.Estimate = agentBuildEstimate(priced, artifacts, conn)
		}
		out.PlanDiscards = agentPlanDiscards(c.Plan, artifacts)
		for _, a := range artifacts {
			view := AIAgentArtifact{ID: a.ID, Handle: a.Handle, Kind: a.Kind, Bundle: a.Bundle, Revision: a.Revision,
				Decision: a.Decision, TargetID: a.TargetID, ResultID: a.ResultID}
			items := c.Plan.Items
			if c.PreviousPlan != nil {
				items = append(slices.Clone(items), c.PreviousPlan.Items...)
			}
			for _, item := range items {
				if item.Handle == a.Handle {
					view.Name = item.Name
					break
				}
			}
			if a.Content == nil || a.Context == nil {
				out.Artifacts = append(out.Artifacts, view)
				continue
			}
			if a.Kind == "segment" {
				var d segments.SegmentDefinition
				if err := json.Unmarshal(a.Content, &d); err != nil {
					return err
				}
				view.Segment = &d
				out.Artifacts = append(out.Artifacts, view)
				continue
			}
			if a.Kind == "automation" {
				var content AIAutomationContent
				var local agentAutomationContext
				if err := json.Unmarshal(a.Content, &content); err != nil {
					return err
				}
				if err := json.Unmarshal(a.Context, &local); err != nil {
					return err
				}
				view.Name, view.Automation, view.Previews = local.Name, &content, local.Previews
				var listID, segmentID string
				if local.Source != nil {
					view.BeforeAutomation, view.BeforeEmails = local.Source.Context.Before, local.Source.Context.Sources
					listID, segmentID = local.Source.Trigger.ListID, local.Source.Trigger.SegmentID
				} else if selection := c.Brief.Automation; selection != nil {
					switch {
					case selection.ListID != nil:
						listID = selection.ListID.String()
					case selection.SegmentID != nil:
						segmentID = selection.SegmentID.String()
					case selection.FromAudience:
						for _, item := range items {
							if item.Kind == "segment" && item.Bundle == a.Bundle {
								view.TriggerName = item.Name
							}
						}
					}
				}
				// Access to these references was checked above. Resolve names
				// only for the human view, not the persisted model checkpoint.
				if listID != "" {
					row, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: uuid.MustParse(listID)})
					if err != nil {
						return api.MapError(err, "List")
					}
					view.TriggerName = row.Name
				} else if segmentID != "" {
					row, err := q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: ws.ID, ID: uuid.MustParse(segmentID)})
					if err != nil {
						return api.MapError(err, "Segment")
					}
					view.TriggerName = row.Name
				}
				out.Artifacts = append(out.Artifacts, view)
				continue
			}
			if a.Kind == "post" {
				var post AIAgentPost
				var d draftContext
				if err := json.Unmarshal(a.Content, &post); err != nil {
					return err
				}
				if err := json.Unmarshal(a.Context, &d); err != nil {
					return err
				}
				view.Name, view.Post, view.Before, view.HTML = post.Title, &post, d.Before, d.PreviewHTML
				out.Artifacts = append(out.Artifacts, view)
				continue
			}
			var d draftContext
			var email AIDraftEmail
			if err := json.Unmarshal(a.Context, &d); err != nil {
				return err
			}
			if err := json.Unmarshal(a.Content, &email); err != nil {
				return err
			}
			view.Name, view.Before, view.Email, view.HTML = d.Name, d.Before, &email, d.PreviewHTML
			out.Artifacts = append(out.Artifacts, view)
		}
		return nil
	})
	return out, err
}

func (s *Service) AgentAction(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIAgentAction) error {
	if validateDraftJSON(in) != nil || (in.Action != "build" && len(in.PlanDiscards) != 0) {
		return agentProblem(ctx, "input")
	}
	sharesContent := in.Action == "answer" || in.Action == "revise" || in.Action == "replan" || in.Action == "propose" || in.Action == "refresh"
	if sharesContent && !in.SharingReviewed {
		return draftProblem(ctx, 422, "sharing_review")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.AIAgentRunLock(ctx, store.AIAgentRunLockParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "AI run")
		}
		c, err := loadAgentContext(ctx, q, id, true)
		if err != nil {
			return api.MapError(err, "AI run")
		}
		fresh, err := s.agentAccess(ctx, q, ws, p, r, c, in.Action != "cancel" && in.Action != "propose" && in.Action != "refresh")
		if err != nil {
			return err
		}
		if !s.now().Before(r.ExpiresAt) {
			return agentProblem(ctx, "expired")
		}
		if fresh.UserID != r.UserID {
			return agentProblem(ctx, "permission")
		}
		// The version is a human-action key. An exact replay after a lost
		// response succeeds without another job, charge or audit event.
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		if recorded, ok := c.Actions[in.Version]; ok && recorded == digest {
			return nil
		}
		if r.Version != in.Version || (in.Action == "build" && r.PlanRevision != in.PlanRevision) {
			return agentProblem(ctx, "changed")
		}
		var resume uuid.UUID
		switch in.Action {
		case "cancel":
			if r.State == "cancelled" || r.State == "completed" {
				return nil
			}
			// An unconfirmed replan never removes the prior review bundle.
			// Restore its plan so cancellation still allows Keep/Discard.
			if c.PreviousPlan != nil {
				c.Plan, c.PreviousPlan = *c.PreviousPlan, nil
				r.PlanRevision++
				r.AcceptedRevision = r.PlanRevision
			}
			r.State, r.ClosedAt = "cancelled", ptr(s.now())
		case "build":
			if r.State != "ready" || len(c.Plan.Questions) != 0 || len(c.Plan.Items) == 0 {
				return agentProblem(ctx, "changed")
			}
			if err := s.confirmAgentPlan(ctx, q, fresh, r, &c, in.PlanDiscards); err != nil {
				return err
			}
			c.Refine = ""
			r.AcceptedRevision, r.State = r.PlanRevision, "queued"
			c.Messages = append(c.Messages, llm.Message{Role: "user", Parts: []llm.Part{textPart("The person confirmed this exact plan. Build its staging drafts only.")}})
		case "propose", "refresh":
			// Kept and stale drafts both require a fresh source and another
			// human-confirmed plan. Refresh rejects the old diff; it never
			// rebases old model output onto a newer resource revision.
			if r.State != "review" && r.State != "completed" && r.State != "paused" {
				return agentProblem(ctx, "changed")
			}
			if len(c.Plan.Items) >= 10 || len(c.Messages) == 0 || len(c.Conversation) >= 500 || !fresh.Can(authz.AIUse) {
				return agentProblem(ctx, "input")
			}
			last, err := q.AIAgentStepLast(ctx, id)
			if err != nil {
				return err
			}
			if last.State != "done" {
				return agentProblem(ctx, "changed")
			}
			message, safe := voiceText(in.Message)
			if !safe || strings.TrimSpace(message) == "" || voiceURL.MatchString(in.Message) || voiceLiquid.MatchString(in.Message) || draftReserved.MatchString(in.Message) {
				return draftProblem(ctx, 422, "privacy")
			}
			message = c.Literals.mask(message)
			artifacts, err := q.AIAgentArtifacts(ctx, id)
			if err != nil {
				return err
			}
			i := slices.IndexFunc(artifacts, func(a store.AiAgentArtifact) bool {
				if a.Kind != "campaign" && a.Kind != "post" && a.Kind != "automation" {
					return false
				}
				if in.Action == "refresh" {
					return a.Handle == in.Handle && a.TargetID != nil && a.Decision == ""
				}
				return a.Handle == in.Handle && a.ResultID != nil && (a.Decision == "kept" || a.Decision == "accepted")
			})
			if i < 0 {
				return agentProblem(ctx, "decided")
			}
			target := artifacts[i].ResultID
			if in.Action == "refresh" {
				target = artifacts[i].TargetID
			}
			if slices.ContainsFunc(artifacts, func(a store.AiAgentArtifact) bool {
				return a.ID != artifacts[i].ID && a.TargetID != nil && *a.TargetID == *target && a.Decision == ""
			}) {
				return draftProblem(ctx, 409, "changed")
			}
			if in.Action == "refresh" {
				if err := q.AIAgentArtifactDecide(ctx, store.AIAgentArtifactDecideParams{ID: artifacts[i].ID, Decision: "rejected", UpdatedAt: s.now()}); err != nil {
					return err
				}
				if err := s.agentAudit(ctx, q, fresh, r, "rejected", map[string]any{"artifact_id": artifacts[i].ID}); err != nil {
					return err
				}
			}
			// Alphabetic handles cannot accidentally resemble a phone number
			// at the model-output privacy boundary. This is a one-to-one hex
			// encoding, retaining all 48 random bits of the suffix.
			handle := "edit_" + strings.Map(func(r rune) rune {
				if r >= '0' && r <= '9' {
					return 'g' + r - '0'
				}
				return r
			}, uuid.Must(uuid.NewV7()).String()[24:])
			local, source, err := s.prepareAgentProposal(ctx, q, ws, fresh, artifacts[i].Kind, *target, handle, c.Literals)
			if err != nil {
				return err
			}
			if c.Proposals == nil {
				c.Proposals = map[string]agentProposalSource{}
			}
			c.Proposals[handle] = local
			// Resource titles can contain personal data; the model only needs
			// the source handle and explicit projected editorial context.
			name := "Draft revision"
			for _, prior := range c.Plan.Items {
				if prior.Handle == in.Handle {
					name = prior.Name
				}
			}
			item := AIAgentPlanItem{Handle: handle, Kind: artifacts[i].Kind, Name: name, Purpose: clip(message, 1000), Bundle: handle, Target: handle, DependsOn: []string{}}
			c.Plan.Items = append(c.Plan.Items, item)
			c.Plan.Questions = []string{}
			c.Refine, r.State, r.ClosedAt = "", "ready", nil
			r.PlanRevision++
			if _, err := s.agentAccess(ctx, q, ws, fresh, r, c, true); err != nil {
				return err
			}
			payload, err := json.Marshal(struct {
				Item   AIAgentPlanItem
				Source any
			}{item, source})
			if err != nil {
				return err
			}
			c.Messages = append(c.Messages, llm.Message{Role: "user", Parts: []llm.Part{textPart("A person selected a current draft for a new proposal. Use only this target handle and its source bindings. The new plan still needs Build confirmation.", tag("proposal_source", string(payload)))}})
			c.Conversation = append(c.Conversation, AIAgentMessage{Role: "user", Text: message, Handle: handle, At: s.now()})
		case "answer", "replan", "revise":
			if in.Action == "answer" && r.State != "needs_answers" && r.State != "ready" {
				return agentProblem(ctx, "changed")
			}
			if in.Action != "answer" && r.State != "review" && r.State != "ready" && r.State != "paused" {
				return agentProblem(ctx, "changed")
			}
			message, safe := voiceText(in.Message)
			if !safe || strings.TrimSpace(message) == "" || voiceURL.MatchString(in.Message) || voiceLiquid.MatchString(in.Message) || draftReserved.MatchString(in.Message) {
				return draftProblem(ctx, 422, "privacy")
			}
			message = c.Literals.mask(message)
			last, err := q.AIAgentStepLast(ctx, id)
			if err != nil {
				return err
			}
			// Do not abandon a partial response or evade an uncertain charge.
			if last.State != "done" || len(c.Messages) == 0 || len(c.Conversation) >= 500 {
				return agentProblem(ctx, "changed")
			}
			u, err := q.AIBudgetRun(ctx, &id)
			if err != nil {
				return err
			}
			if u.Uncertain > 0 {
				return budgetProblem(ctx, "usage_unknown")
			}
			c.Refine = ""
			if in.Action == "revise" {
				if r.AcceptedRevision == 0 || r.AcceptedRevision != r.PlanRevision {
					return agentProblem(ctx, "changed")
				}
				artifacts, err := q.AIAgentArtifacts(ctx, id)
				if err != nil {
					return err
				}
				if !slices.ContainsFunc(artifacts, func(a store.AiAgentArtifact) bool {
					return a.Handle == in.Handle && a.Decision == "" && a.TargetID == nil && !c.Discarded[a.Bundle]
				}) {
					return agentProblem(ctx, "decided")
				}
				c.Refine, r.State = in.Handle, "queued"
				// The person's revision request starts that output's repairs afresh.
				for _, a := range artifacts {
					if a.Handle == in.Handle {
						delete(c.Failures, a.Kind+":"+a.Handle)
					}
				}
				c.Messages = append(c.Messages, llm.Message{Role: "user", Parts: []llm.Part{textPart("Revise only the unkept draft "+in.Handle+". Preserve its accepted scope and dependencies. If the request needs a different scope, leave drafts unchanged and explain that a revised plan needs confirmation.", tag("copy_revision", message))}})
			} else {
				if c.PreviousPlan == nil && r.AcceptedRevision > 0 {
					previous := c.Plan
					c.PreviousPlan = &previous
				}
				r.AcceptedRevision, r.State = 0, "planning"
				// A person's new instruction starts every repair count afresh.
				c.Failures = nil
				c.Messages = append(c.Messages, llm.Message{Role: "user", Parts: []llm.Part{textPart("Prepare a revised plan for confirmation. Preserve decided items. Retain existing staging items unchanged, or replace their entire bundle using fresh handles and a new bundle. Replacements discard the old staging bundle only after the person explicitly confirms its revisions. Do not rebuild before confirmation.", tag("answer", message))}})
			}
			c.Conversation = append(c.Conversation, AIAgentMessage{Role: "user", Text: message, Handle: in.Handle, At: s.now()})
		case "continue":
			if r.State != "paused" {
				return agentProblem(ctx, "changed")
			}
			if r.PauseReason == "validation_exhausted" {
				return agentProblem(ctx, "manual")
			}
			u, err := q.AIBudgetRun(ctx, &id)
			if err != nil {
				return err
			}
			if u.Uncertain > 0 {
				return budgetProblem(ctx, "usage_unknown")
			}
			if r.ToolCalls >= r.ToolLimit || u.Tokens >= r.TokenLimit || u.CostMicros >= r.CostLimitMicros {
				return budgetProblem(ctx, "run_cap")
			}
			r.State = "planning"
			if r.AcceptedRevision > 0 {
				r.State = "queued"
			}
			step, err := q.AIAgentStepLast(ctx, r.ID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if step.State == "responded" {
				resume = step.ID
			} else if r.PauseReason == "incomplete_plan" {
				instruction := "The person requested continuation. Finish the plan for human confirmation; do not create drafts yet."
				if r.AcceptedRevision > 0 {
					artifacts, err := q.AIAgentReferences(ctx, r.ID)
					if err != nil {
						return err
					}
					missing := []string{}
					for _, item := range c.Plan.Items {
						if !c.Discarded[item.Bundle] && !slices.ContainsFunc(artifacts, func(a store.AIAgentReferencesRow) bool { return a.Handle == item.Handle }) {
							missing = append(missing, item.Handle)
						}
					}
					instruction = "The person requested continuation of the confirmed plan. Stage only these missing item handles: " + strings.Join(missing, ", ") + ". Keep already staged and decided items unchanged. Previous validation failures and cumulative limits still apply."
				}
				c.Messages = append(c.Messages, llm.Message{Role: "user", Parts: []llm.Part{textPart(instruction)}})
			}
		}
		if c.Actions == nil {
			c.Actions = map[int64][32]byte{}
		}
		c.Actions[in.Version] = digest
		r.Version++
		r.PauseReason = ""
		if resume != uuid.Nil {
			if err := q.AIAgentStepResume(ctx, store.AIAgentStepResumeParams{ID: resume, RunVersion: r.Version, Now: s.now()}); err != nil {
				return err
			}
		}
		if err := saveAgentContext(ctx, q, id, c); err != nil {
			return err
		}
		if err := s.saveAgentRun(ctx, q, &r); err != nil {
			return err
		}
		metadata := map[string]any{}
		if sharesContent {
			metadata["sharing_policy"] = editorial.SharingPolicy
		}
		if err := s.agentAudit(ctx, q, fresh, r, in.Action, metadata); err != nil {
			return err
		}
		if in.Action == "cancel" || in.Action == "propose" || in.Action == "refresh" {
			return nil
		}
		return s.enqueueAgent(ctx, tx, r)
	})
}

// SetAgentRunLimits changes only the cumulative ceiling, never counters or
// execution state. Continuation remains the initiator's explicit decision.
func (s *Service) SetAgentRunLimits(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIAgentLimitChange) error {
	if validateDraftJSON(in) != nil {
		return agentProblem(ctx, "input")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.AIAgentRunLock(ctx, store.AIAgentRunLockParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "AI run")
		}
		c, err := loadAgentContext(ctx, q, id, false)
		if err != nil {
			return api.MapError(err, "AI run")
		}
		fresh, err := s.agentAccess(ctx, q, ws, p, r, c, false)
		if err != nil {
			return err
		}
		if !fresh.Can(authz.IntegrationsManage) {
			return api.Forbidden(authz.IntegrationsManage, nil)
		}
		if !s.now().Before(r.ExpiresAt) {
			return agentProblem(ctx, "expired")
		}
		if r.Version != in.Version || r.State != "paused" {
			return agentProblem(ctx, "changed")
		}
		before := agentLimits(r)
		if in.Limits.ToolCalls < before.ToolCalls || in.Limits.Tokens < before.Tokens || in.Limits.CostMicros < before.CostMicros {
			return agentProblem(ctx, "input")
		}
		if before == in.Limits {
			return nil
		}
		r.ToolLimit, r.TokenLimit, r.CostLimitMicros = in.Limits.ToolCalls, in.Limits.Tokens, in.Limits.CostMicros
		r.Version++
		if err := s.saveAgentRun(ctx, q, &r); err != nil {
			return err
		}
		return s.agentAudit(ctx, q, fresh, r, "limits", map[string]any{"before": before, "after": in.Limits})
	})
}

func (s *Service) agentAudit(ctx context.Context, q *store.Queries, p *authz.Principal, r store.AiAgentRun, action string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["via"], metadata["run_id"] = "ai_agent", r.ID.String()
	return s.Audit.LogWith(audit.WithPrincipal(ctx, p), q, audit.Event{WorkspaceID: &r.WorkspaceID, Action: "ai.agent." + action, TargetType: "ai_run", TargetID: r.ID.String(), Metadata: metadata})
}
