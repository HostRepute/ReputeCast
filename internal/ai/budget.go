package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

const maxModelContextBytes = 1_000_000

func budgetProblem(ctx context.Context, key string) error {
	return api.Err(409, "ai_"+key, text(ctx, msgs.M("budget."+key), nil))
}

// reserveIn serializes every spending path, even before a connection exists.
// A caller handling a run also locks that run before calling this function.
// No database lock spans an HTTP call. Request IDs survive worker retries.
func (s *Service) reserveIn(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, c *connection, t *task, run *store.AiAgentRun, id uuid.UUID) error {
	fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
	if err != nil {
		return err
	}
	permission := authz.AIUse
	if t.feature == FeatureConnectionTest {
		permission = authz.IntegrationsManage
	}
	if !fresh.Can(permission) {
		return api.Forbidden(permission, nil)
	}
	if t.checkSource != nil {
		if err := t.checkSource(ctx, q, fresh); err != nil {
			return err
		}
	}
	row, err := q.AIConnectionForUpdate(ctx, ws.ID)
	missing := errors.Is(err, pgx.ErrNoRows)
	if errors.Is(err, pgx.ErrNoRows) && t.feature == FeatureConnectionTest {
		row, err = c.AiConnection, nil
	}
	if err != nil {
		return api.MapError(err, "AI connection")
	}
	// Domain continuations may already hold the configuration lock. Always
	// acquire it before the budget lock; accounting needs only the latter.
	if err := q.AIBudgetWorkspaceLock(ctx, ws.ID); err != nil {
		return err
	}
	if missing {
		// A first connection may have committed while this probe waited.
		// Its cap applies now; never take a row lock after the budget lock.
		current, err := q.AIConnection(ctx, ws.ID)
		if err == nil {
			row = current
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	if t.feature != FeatureConnectionTest && (!row.UpdatedAt.Equal(c.UpdatedAt) || row.DisconnectedAt != nil || row.Status == "key_rejected" || !(&connection{row}).featureOn(t.feature)) {
		return budgetProblem(ctx, "connection_changed")
	}
	if !c.PriceKnown && !c.Demo {
		return budgetProblem(ctx, "price_unknown")
	}
	// Bound text separately from binary image transport. Base64 length is
	// neither an image's token count nor part of the model's text context.
	// Clone before stripping bytes so the provider still receives the image.
	req := t.req
	req.Task = nil
	req.Messages = slices.Clone(req.Messages)
	var imageTokens int64
	for i := range req.Messages {
		req.Messages[i].Parts = slices.Clone(req.Messages[i].Parts)
		for j := range req.Messages[i].Parts {
			part := &req.Messages[i].Parts[j]
			if part.Image == nil {
				continue
			}
			cfg, _, err := image.DecodeConfig(bytes.NewReader(part.Image.Data))
			if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
				return budgetProblem(ctx, "context_limit")
			}
			// Hosted tile models cost at most 48,169 image tokens (GPT-4o
			// mini: 2,833 + 8 × 5,667). Reserve extra headroom, and scale
			// up for large patch-based inputs instead of trusting compression.
			// https://developers.openai.com/api/docs/guides/images-vision
			patches := ((int64(cfg.Width) + 31) / 32) * ((int64(cfg.Height) + 31) / 32)
			imageTokens += max(65_536, patches*32)
			part.Image = &llm.Image{MediaType: part.Image.MediaType}
		}
	}
	raw, err := json.Marshal(req)
	if err != nil || len(raw) > maxModelContextBytes {
		return budgetProblem(ctx, "context_limit")
	}
	input := int64(len(raw)+4096+len(req.Messages)*128+len(req.Tools)*512) + imageTokens
	output := int64(t.req.MaxTokens)
	if output <= 0 {
		output = llm.DefaultMaxTokens
	}
	output = min(output, 16384)
	minimumOutput := min(output, 64)
	start, _ := s.month(ws)
	used, err := q.AIUsageTotal(ctx, store.AIUsageTotalParams{WorkspaceID: ws.ID, Since: start})
	if err != nil {
		return err
	}
	held, err := q.AIBudgetMonthly(ctx, ws.ID)
	if err != nil {
		return err
	}
	switch row.CapKind {
	case "tokens":
		output = min(output, row.CapTokens-used.InputTokens-used.OutputTokens-held.Tokens-input)
	case "usd":
		output = min(output, affordableOutput(int64(math.Floor(row.CapUsd*1e6))-used.CostMicros-held.CostMicros, input, c.InputPrice, c.OutputPrice))
	}
	if output < minimumOutput {
		return budgetProblem(ctx, "reservation_cap")
	}
	var runID *uuid.UUID
	if run != nil {
		runID = &run.ID
		used, err := q.AIBudgetRun(ctx, runID)
		if err != nil {
			return err
		}
		if used.Uncertain > 0 {
			return budgetProblem(ctx, "usage_unknown")
		}
		output = min(output, run.TokenLimit-used.Tokens-input,
			affordableOutput(run.CostLimitMicros-used.CostMicros, input, c.InputPrice, c.OutputPrice))
		if output < minimumOutput {
			return budgetProblem(ctx, "run_cap")
		}
	}
	t.req.MaxTokens = int(output)
	pricing, _ := json.Marshal([]any{c.Provider, c.Model, c.InputPrice, c.OutputPrice, c.UpdatedAt})
	digest := sha256.Sum256(pricing)
	return q.AIBudgetInsert(ctx, store.AIBudgetInsertParams{ID: id, WorkspaceID: ws.ID, UserID: actorID(p), RunID: runID, Feature: t.feature,
		Provider: c.Provider, Model: c.Model, PricingRevision: hex.EncodeToString(digest[:]), InputPrice: c.InputPrice, OutputPrice: c.OutputPrice,
		InputTokens: input, OutputTokens: output, CostMicros: int64(math.Ceil(float64(input)*c.InputPrice + float64(output)*c.OutputPrice)), CreatedAt: s.now()})
}

func affordableOutput(micros, input int64, inputPrice, outputPrice float64) int64 {
	remaining := float64(micros) - math.Ceil(float64(input)*inputPrice)
	if remaining < 0 {
		return -1
	}
	if outputPrice == 0 {
		return 16384
	}
	return int64(math.Min(16384, math.Floor(remaining/outputPrice)))
}

// accountIn commits measured usage and releases the reservation together.
// An absent counter is not zero; a lost answer retains the upper bound.
// The same accounting is safe to replay after a worker loses acknowledgment.
func (s *Service) accountIn(ctx context.Context, q *store.Queries, row store.AIRequestInsertParams, res llm.Response, callErr error) (bool, error) {
	// Use the reservation lock order: the monthly total and held amounts
	// must not change between the two reads in reserveIn.
	if err := q.AIBudgetWorkspaceLock(ctx, row.WorkspaceID); err != nil {
		return false, err
	}
	r, err := q.AIBudgetLock(ctx, store.AIBudgetLockParams{WorkspaceID: row.WorkspaceID, ID: row.ID})
	if err != nil {
		return false, err
	}
	if r.State == "settled" {
		return true, nil
	}
	known := res.UsageKnown
	// Explicit pre-execution HTTP rejections are non-billable. Timeouts,
	// server errors and invalid/oversized responses are not safe to retry.
	if e, ok := llm.AsError(callErr); ok && !known && (e.Kind == llm.ErrAuth || e.Kind == llm.ErrRateLimit || e.Kind == llm.ErrBadRequest) {
		known = true
	}
	if !known {
		if err := q.AIBudgetUncertain(ctx, row.ID); err != nil {
			return false, err
		}
		row.InputTokens, row.OutputTokens, row.CostMicros = 0, 0, 0
		if row.Outcome != OutcomeCancelled {
			row.Outcome = OutcomeError
		}
		row.OutcomeDetail = "usage unknown"
		return false, q.AIRequestInsert(ctx, row)
	}
	row.UsageKnown = true
	row.InputTokens, row.OutputTokens = int32(res.InputTokens), int32(res.OutputTokens)
	row.CostMicros = int64(math.Ceil(float64(res.InputTokens)*r.InputPrice + float64(res.OutputTokens)*r.OutputPrice))
	// A provider reporting more than the conservative reservation is
	// accounted fully; its run will pause rather than hiding the excess.
	if err := q.AIBudgetSettle(ctx, store.AIBudgetSettleParams{ID: row.ID, InputTokens: int64(row.InputTokens), OutputTokens: int64(row.OutputTokens), CostMicros: row.CostMicros, SettledAt: ptr(s.now())}); err != nil {
		return false, err
	}
	return true, q.AIRequestInsert(ctx, row)
}

func (s *Service) account(ctx context.Context, row store.AIRequestInsertParams, res llm.Response, callErr error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	known := false
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		known, err = s.accountIn(ctx, store.New(tx), row, res, callErr)
		return err
	})
	if err != nil {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, row.WorkspaceID)
	if err == nil {
		s.afterUse(ctx, &ws)
	}
	if !known && callErr == nil {
		return budgetProblem(ctx, "usage_unknown")
	}
	return nil
}

// AIBudgetHold is content-free; projections never expose the saved prompt,
// tool results, credentials or a resource's title to integration managers.
type AIBudgetHold struct {
	ID                 uuid.UUID `json:"id"`
	Provider           string    `json:"provider"`
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"createdAt"`
	ReservedTokens     int64     `json:"reservedTokens"`
	ReservedCostMicros int64     `json:"reservedCostMicros"`
	ReconcileAfter     time.Time `json:"reconcileAfter"`
}

type AIBudgetHolds struct {
	Items []AIBudgetHold `json:"items"`
}

type AIBudgetReconcileInput struct {
	InputTokens  int  `json:"inputTokens" minimum:"0" maximum:"1000000000"`
	OutputTokens int  `json:"outputTokens" minimum:"0" maximum:"1000000000"`
	Confirmed    bool `json:"confirmed" doc:"The manager verified these counts with the provider; zero means confirmed non-billable, not unknown"`
}

// BudgetHolds also shows abandoned reserved calls before the maintenance
// sweep runs. Reconciliation waits longer than every possible HTTP timeout.
func (s *Service) BudgetHolds(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AIBudgetHolds, error) {
	out := AIBudgetHolds{Items: []AIBudgetHold{}}
	fresh, err := brandkit.VoiceActor(ctx, s.Q, ws, p)
	if err != nil {
		return out, err
	}
	if !fresh.Can(authz.IntegrationsManage) {
		return out, api.Forbidden(authz.IntegrationsManage, nil)
	}
	rows, err := s.Q.AIBudgetPending(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, AIBudgetHold{ID: r.ID, Provider: r.Provider, Model: r.Model, CreatedAt: r.CreatedAt,
			ReservedTokens: r.InputTokens + r.OutputTokens, ReservedCostMicros: r.CostMicros, ReconcileAfter: r.CreatedAt.Add(10 * time.Minute)})
	}
	return out, nil
}

// ReconcileBudget records provider-verified counts. It does not restart a
// run, recover a lost response, accept a draft or reset cumulative limits.
func (s *Service) ReconcileBudget(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIBudgetReconcileInput) error {
	if !in.Confirmed || in.InputTokens < 0 || in.OutputTokens < 0 || in.InputTokens > 1_000_000_000 || in.OutputTokens > 1_000_000_000 {
		return budgetProblem(ctx, "reconcile_confirm")
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
		if err != nil {
			return err
		}
		if !fresh.Can(authz.IntegrationsManage) {
			return api.Forbidden(authz.IntegrationsManage, nil)
		}
		if err := q.AIBudgetWorkspaceLock(ctx, ws.ID); err != nil {
			return err
		}
		r, err := q.AIBudgetLock(ctx, store.AIBudgetLockParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "AI request")
		}
		if r.State == "settled" {
			if r.InputTokens == int64(in.InputTokens) && r.OutputTokens == int64(in.OutputTokens) {
				return nil
			}
			return budgetProblem(ctx, "reconcile_changed")
		}
		if s.now().Before(r.CreatedAt.Add(10 * time.Minute)) {
			return budgetProblem(ctx, "reconcile_wait")
		}
		res := llm.Response{InputTokens: in.InputTokens, OutputTokens: in.OutputTokens, UsageKnown: true}
		_, err = s.accountIn(ctx, q, store.AIRequestInsertParams{ID: id, WorkspaceID: ws.ID, UserID: r.UserID,
			Provider: r.Provider, Model: r.Model, Feature: r.Feature, Outcome: OutcomeError, OutcomeDetail: "response unavailable", CreatedAt: r.CreatedAt}, res, nil)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.usage_reconciled", TargetType: "ai_request", TargetID: id.String(),
			Before:   map[string]any{"reserved_tokens": r.InputTokens + r.OutputTokens, "reserved_cost_micros": r.CostMicros},
			After:    map[string]any{"input_tokens": in.InputTokens, "output_tokens": in.OutputTokens},
			Metadata: map[string]any{"pricing_revision": r.PricingRevision, "run_id": r.RunID}})
	})
	if err == nil {
		s.afterUse(ctx, ws)
	}
	return err
}
