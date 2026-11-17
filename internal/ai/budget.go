package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// reserveIn serializes every spending path of a budget, across all its
// accounts and models, even before a connection exists: the workspace's,
// or a person's own for their personal keys. A caller handling a run also
// locks that run before calling this function.
// No database lock spans an HTTP call. Request IDs survive worker retries.
func (s *Service) reserveIn(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, c *connection, t *task, run *store.AiAgentRun, id uuid.UUID) error {
	fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
	if err != nil {
		return err
	}
	// A workspace account's test is a manager's; a personal key's its
	// owner's, with ai:use.
	permission := authz.AIUse
	if t.feature == FeatureConnectionTest && !c.personal() {
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
	// Domain continuations may already hold the configuration lock. Always
	// acquire it before the budget lock; accounting needs only the latter.
	// A first connection's test makes the settings row, with the default cap.
	row, err := lockSettings(ctx, q, ws.ID)
	if err != nil {
		return err
	}
	// A personal key works only for its owner, and only while managers
	// allow personal keys; its use is in the owner's own budget.
	owner := c.AccountUserID
	if owner != nil {
		if fresh.UserID != *owner {
			return api.NotFound("AI model")
		}
		if !row.PersonalKeys {
			return personalKeysOff(ctx)
		}
	}
	if err := budgetLock(ctx, q, ws.ID, owner); err != nil {
		return err
	}
	if t.feature != FeatureConnectionTest {
		// The request is bound to its model, that model's account and their
		// prices as they were when it was prepared; any change since refuses it.
		cur, err := s.binding(ctx, q, ws.ID, &c.ModelID)
		if err != nil {
			return err
		}
		if cur == nil || cur.SettingsRevision != c.SettingsRevision || cur.AccountRevision != c.AccountRevision || cur.ModelRevision != c.ModelRevision ||
			!cur.connected() || cur.Status == "key_rejected" || !cur.featureOn(t.feature) {
			return budgetProblem(ctx, "connection_changed")
		}
		// A one-shot request goes to its feature's model (the person's own
		// pick, or the workspace's default); once that changes, it isn't
		// sent. A run keeps the model it began with.
		if run == nil {
			model, ok, err := featureModel(ctx, q, ws.ID, &fresh.UserID, t.feature)
			if err != nil {
				return err
			}
			if !ok || model != c.ModelID {
				return budgetProblem(ctx, "connection_changed")
			}
		}
		c = cur
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
	if owner != nil {
		// A personal key's use is its owner's: their own monthly cap, never
		// the workspace's.
		used, err := q.AIPersonalUsageTotal(ctx, store.AIPersonalUsageTotalParams{WorkspaceID: ws.ID, UserID: owner, Since: start})
		if err != nil {
			return err
		}
		held, err := q.AIBudgetPersonal(ctx, store.AIBudgetPersonalParams{WorkspaceID: ws.ID, UserID: owner})
		if err != nil {
			return err
		}
		output = min(output, affordableOutput(int64(math.Floor(row.PersonalCapUsd*1e6))-used.CostMicros-held.CostMicros, input, c.InputPrice, c.OutputPrice))
	} else {
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
	pricing, _ := json.Marshal([]any{c.AccountID, c.AccountRevision, c.ModelID, c.ModelRevision, c.Provider, c.Model, c.InputPrice, c.OutputPrice})
	digest := sha256.Sum256(pricing)
	// A candidate being tested has no account or model yet.
	var account, model *uuid.UUID
	if c.ModelID != uuid.Nil {
		account, model = &c.AccountID, &c.ModelID
	}
	return q.AIBudgetInsert(ctx, store.AIBudgetInsertParams{ID: id, WorkspaceID: ws.ID, UserID: actorID(p), RunID: runID, Feature: t.feature,
		Provider: c.Provider, Model: c.Model, PricingRevision: hex.EncodeToString(digest[:]), InputPrice: c.InputPrice, OutputPrice: c.OutputPrice,
		InputTokens: input, OutputTokens: output, CostMicros: int64(math.Ceil(float64(input)*c.InputPrice + float64(output)*c.OutputPrice)), CreatedAt: s.now(),
		AccountID: account, ModelID: model, Personal: owner != nil})
}

// budgetLock takes a budget's lock, after the configuration lock: the
// workspace's (person nil), or a person's own for their personal keys. The
// two never wait for each other.
func budgetLock(ctx context.Context, q *store.Queries, ws uuid.UUID, person *uuid.UUID) error {
	if person != nil {
		return q.AIBudgetPersonLock(ctx, store.AIBudgetPersonLockParams{WorkspaceID: ws, UserID: *person})
	}
	return q.AIBudgetWorkspaceLock(ctx, ws)
}

// reservationLock takes the lock of the budget a reservation is in, then the
// reservation's row. Whose budget it is never changes once it is made.
func reservationLock(ctx context.Context, q *store.Queries, ws, id uuid.UUID) (store.AiBudgetReservation, error) {
	scope, err := q.AIBudgetScope(ctx, store.AIBudgetScopeParams{ID: id, WorkspaceID: ws})
	if err != nil {
		return store.AiBudgetReservation{}, err
	}
	var person *uuid.UUID
	if scope.Personal {
		person = scope.UserID
	}
	if err := budgetLock(ctx, q, ws, person); err != nil {
		return store.AiBudgetReservation{}, err
	}
	return q.AIBudgetLock(ctx, store.AIBudgetLockParams{WorkspaceID: ws, ID: id})
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
	r, err := reservationLock(ctx, q, row.WorkspaceID, row.ID)
	if err != nil {
		return false, err
	}
	if r.State == "settled" {
		return true, nil
	}
	// The log names the model and account the reservation bound, whatever
	// the caller passed, and whose budget it was in.
	row.Provider, row.Model, row.AccountID, row.ModelID, row.Personal = r.Provider, r.Model, r.AccountID, r.ModelID, r.Personal
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
	return s.budgetHolds(ctx, ws, p, false)
}

// PersonalBudgetHolds lists the caller's own holds on their personal keys.
func (s *Service) PersonalBudgetHolds(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AIBudgetHolds, error) {
	return s.budgetHolds(ctx, ws, p, true)
}

// budgetHolds lists the workspace's holds (for integration managers) or the
// caller's own on their personal keys (personal, with ai:use).
func (s *Service) budgetHolds(ctx context.Context, ws *store.Workspace, p *authz.Principal, personal bool) (AIBudgetHolds, error) {
	out := AIBudgetHolds{Items: []AIBudgetHold{}}
	fresh, err := brandkit.VoiceActor(ctx, s.Q, ws, p)
	if err != nil {
		return out, err
	}
	permission := holdsPermission(personal)
	if !fresh.Can(permission) {
		return out, api.Forbidden(permission, nil)
	}
	var rows []store.AiBudgetReservation
	if personal {
		rows, err = s.Q.AIBudgetPersonalPending(ctx, store.AIBudgetPersonalPendingParams{WorkspaceID: ws.ID, UserID: &fresh.UserID})
	} else {
		rows, err = s.Q.AIBudgetPending(ctx, ws.ID)
	}
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, AIBudgetHold{ID: r.ID, Provider: r.Provider, Model: r.Model, CreatedAt: r.CreatedAt,
			ReservedTokens: r.InputTokens + r.OutputTokens, ReservedCostMicros: r.CostMicros, ReconcileAfter: r.CreatedAt.Add(10 * time.Minute)})
	}
	return out, nil
}

// holdsPermission is what the holds take: integrations:manage for the
// workspace's, ai:use for one's own personal keys'.
func holdsPermission(personal bool) authz.Permission {
	if personal {
		return authz.AIUse
	}
	return authz.IntegrationsManage
}

// ReconcileBudget records provider-verified counts. It does not restart a
// run, recover a lost response, accept a draft or reset cumulative limits.
func (s *Service) ReconcileBudget(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIBudgetReconcileInput) error {
	return s.reconcile(ctx, ws, p, id, in, false)
}

// ReconcilePersonalBudget records provider-verified counts for one of the
// caller's own holds on their personal keys, in their own budget.
func (s *Service) ReconcilePersonalBudget(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIBudgetReconcileInput) error {
	return s.reconcile(ctx, ws, p, id, in, true)
}

// reconcile settles a workspace hold, or (personal) one of the caller's own.
// Someone else's personal hold isn't there, even for a manager.
func (s *Service) reconcile(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIBudgetReconcileInput, personal bool) error {
	if !in.Confirmed || in.InputTokens < 0 || in.OutputTokens < 0 || in.InputTokens > 1_000_000_000 || in.OutputTokens > 1_000_000_000 {
		return budgetProblem(ctx, "reconcile_confirm")
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
		if err != nil {
			return err
		}
		permission := holdsPermission(personal)
		if !fresh.Can(permission) {
			return api.Forbidden(permission, nil)
		}
		// A personal key's holds are its owner's, not the workspace's.
		scope, err := q.AIBudgetScope(ctx, store.AIBudgetScopeParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "AI request")
		}
		if scope.Personal != personal || (personal && (scope.UserID == nil || *scope.UserID != fresh.UserID)) {
			return api.NotFound("AI request")
		}
		r, err := reservationLock(ctx, q, ws.ID, id)
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
		meta := map[string]any{"pricing_revision": r.PricingRevision, "run_id": r.RunID}
		if personal {
			meta["personal"] = true
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.usage_reconciled", TargetType: "ai_request", TargetID: id.String(),
			Before:   map[string]any{"reserved_tokens": r.InputTokens + r.OutputTokens, "reserved_cost_micros": r.CostMicros},
			After:    map[string]any{"input_tokens": in.InputTokens, "output_tokens": in.OutputTokens},
			Metadata: meta})
	})
	if err == nil {
		s.afterUse(ctx, ws)
	}
	return err
}
