package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// errDemoElsewhere: the demo's sample connection outside development.
var errDemoElsewhere = errors.New("the demo's sample model only answers in development")

// task is one request a feature makes.
type task struct {
	feature         string
	action          string
	targetKind      string
	targetID        *uuid.UUID
	targetLabel     string
	req             llm.Request
	selectedVoice   *int32
	voiceRevision   int32
	voiceChars      int
	privateLiterals privateLiterals
	// Draft repairs must recheck the source in the reservation transaction.
	checkSource func(context.Context, *store.Queries, *authz.Principal) error
}

// result is what the model answered, and what it cost.
type result struct {
	RequestID     uuid.UUID
	Response      llm.Response
	JSON          json.RawMessage
	Refusal       string
	CostMicros    int64
	conn          *connection
	voiceRevision int32
	voiceChars    int
}

// AISent says what went to the model, for the card's footer.
type AISent struct {
	Provider     string `json:"provider" enum:"anthropic,openai,google,azure,compatible"`
	ProviderName string `json:"providerName"`
	Model        string `json:"model"`
	ModelName    string `json:"modelName"`
	// Characters of the person's content that were sent.
	Characters int `json:"characters"`
	// CostMicros is the estimated cost in millionths of a dollar.
	CostMicros    int64 `json:"costMicros"`
	InputTokens   int   `json:"inputTokens"`
	OutputTokens  int   `json:"outputTokens"`
	VoiceRevision int32 `json:"voiceRevision,omitempty"`
}

func (r result) sent(chars int) AISent {
	return AISent{Provider: r.conn.Provider, ProviderName: r.conn.providerName(), Model: r.conn.Model, ModelName: r.conn.modelName(), Characters: chars + r.voiceChars,
		CostMicros: r.CostMicros, InputTokens: r.Response.InputTokens, OutputTokens: r.Response.OutputTokens, VoiceRevision: r.voiceRevision}
}

// usable checks that a feature can send a person's request now: a model is
// connected, the feature is on, the key works and the cap isn't reached.
func (s *Service) usable(ctx context.Context, ws *store.Workspace, p *authz.Principal, feature string) (*connection, error) {
	c, err := s.loadFor(ctx, s.Q, ws.ID, actorID(p), feature)
	if err != nil {
		return nil, err
	}
	return s.ready(ctx, ws, c, feature)
}

// ready checks that a model can take a feature's request now. A feature
// whose default model is disconnected waits for it: another model is never
// used instead. A personal key is its owner's to fix.
func (s *Service) ready(ctx context.Context, ws *store.Workspace, c *connection, feature string) (*connection, error) {
	if c.personal() {
		if !c.PersonalKeys {
			return nil, personalKeysOff(ctx)
		}
		if !c.connected() {
			return nil, problem(ctx, http.StatusConflict, codeNotConnected, mPersonalNoKey, i18n.Args{"provider": c.providerName()})
		}
	}
	if !c.connected() {
		return nil, s.notConnected(ctx, ws)
	}
	if feature == FeatureAltText && !c.ImageInput {
		return nil, withHelpers(ctx, http.StatusConflict, codeNoImageInput, mNoImageInput, mNoImageInputAlone, nil, s.helpers(ctx, ws.ID))
	}
	if !c.featureOn(feature) {
		return nil, withHelpers(ctx, http.StatusConflict, codeFeatureOff, mFeatureOff, mFeatureOffAlone, i18n.Args{"feature": feature}, s.helpers(ctx, ws.ID))
	}
	if c.Status == "key_rejected" {
		if c.personal() {
			return nil, problem(ctx, http.StatusConflict, codeKeyRejected, mPersonalKeyRejected, i18n.Args{"provider": c.providerName()})
		}
		return nil, withHelpers(ctx, http.StatusConflict, codeKeyRejected, mKeyRejected, mKeyRejectedAlone, i18n.Args{"provider": c.providerName()}, s.helpers(ctx, ws.ID))
	}
	u, err := s.usage(ctx, ws, c)
	if err != nil {
		return nil, err
	}
	if u.reached {
		if c.personal() {
			return nil, s.personalCapReached(ctx, ws, c, u)
		}
		return nil, s.capReached(ctx, ws, c, u)
	}
	return c, nil
}

func (s *Service) notConnected(ctx context.Context, ws *store.Workspace) *api.Problem {
	if p := api.PrincipalFrom(ctx); p.Can(authz.IntegrationsManage) {
		return problem(ctx, http.StatusConflict, codeNotConnected, mNotConnected, nil)
	}
	return withHelpers(ctx, http.StatusConflict, codeNotConnected, mNotConnectedAsk, mNotConnectedAlone, nil, s.helpers(ctx, ws.ID))
}

func (s *Service) capReached(ctx context.Context, ws *store.Workspace, c *connection, u usage) *api.Problem {
	l := i18n.LangFrom(ctx)
	_, next := s.month(ws)
	args := i18n.Args{"date": l.Region.DayMonth(next), "cap": l.Region.Money(c.CapUsd, 2)}
	if c.CapKind == "tokens" {
		args["cap"] = l.Region.Int(c.CapTokens)
		return withHelpers(ctx, http.StatusConflict, codeCapReached, mTokenCapReached, mCapReachedAlone, args, s.helpers(ctx, ws.ID)).
			WithMeta("resetsAt", next, "capTokens", c.CapTokens)
	}
	return withHelpers(ctx, http.StatusConflict, codeCapReached, mCapReached, mCapReachedAlone, args, s.helpers(ctx, ws.ID)).
		WithMeta("resetsAt", next, "capUsd", c.CapUsd, "spentMicros", u.costMicros)
}

// personalCapReached: a person's own budget for their personal keys is
// used up this month.
func (s *Service) personalCapReached(ctx context.Context, ws *store.Workspace, c *connection, u usage) *api.Problem {
	l := i18n.LangFrom(ctx)
	_, next := s.month(ws)
	return problem(ctx, http.StatusConflict, codePersonalCapReached, mPersonalCapReached, i18n.Args{"date": l.Region.DayMonth(next), "cap": l.Region.Money(c.PersonalCapUsd, 2)}).
		WithMeta("resetsAt", next, "capUsd", c.PersonalCapUsd, "spentMicros", u.costMicros)
}

// personalKeysOff: managers turned personal keys off; the accounts stay.
func personalKeysOff(ctx context.Context) *api.Problem {
	return problem(ctx, http.StatusConflict, codePersonalKeysOff, mPersonalKeysOff, nil)
}

// run sends one request and records it in the activity log. A refusal is
// an answer (Refusal is set); errors are problems that say nothing was
// changed.
func (s *Service) run(ctx context.Context, ws *store.Workspace, p *authz.Principal, t task) (result, error) {
	c, err := s.usable(ctx, ws, p, t.feature)
	if err != nil {
		return result{}, err
	}
	if t.feature == FeatureSubjectIdeas || t.feature == FeatureRewrite || t.feature == FeatureDraft {
		if err := s.withVoice(ctx, ws, &t); err != nil {
			return result{}, err
		}
	}
	return s.send(ctx, ws, p, c, t)
}

func (s *Service) send(ctx context.Context, ws *store.Workspace, p *authz.Principal, c *connection, t task) (result, error) {
	prov, err := s.provider(c, t.feature)
	if errors.Is(err, errDemoElsewhere) {
		return result{}, problem(ctx, http.StatusConflict, codeDemoModel, mDemoModel, nil)
	}
	if err != nil {
		return result{}, err
	}
	id := uuid.Must(uuid.NewV7())
	if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := s.reserveIn(ctx, q, ws, p, c, &t, nil, id); err != nil {
			return err
		}
		// Record the sharing decision before HTTP, including requests that
		// fail or are cancelled after the provider receives their content.
		if t.feature == FeatureBrandVoice || t.feature == FeatureDraft {
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.sharing.reviewed", TargetType: "ai_request", TargetID: id.String(),
				Metadata: map[string]any{"feature": t.feature, "provider": c.Provider, "model": c.Model, "sharing_policy": editorial.SharingPolicy}})
		}
		return nil
	}); err != nil {
		return result{}, err
	}
	start := s.now()
	res, gerr := prov.Generate(ctx, t.req)
	elapsed := time.Since(start)
	if s.Now != nil {
		elapsed = 0
	}
	row := store.AIRequestInsertParams{
		ID: id, WorkspaceID: ws.ID, UserID: actorID(p), Feature: t.feature, Action: t.action, TargetKind: t.targetKind,
		TargetID: t.targetID, TargetLabel: clip(t.targetLabel, 200), Provider: c.Provider, Model: c.Model, Outcome: OutcomePending,
		DurationMs: int32(min(elapsed.Milliseconds(), math.MaxInt32)), CreatedAt: start, VoiceRevision: t.voiceRevision,
	}
	// The log keeps what happened even when the person left the page.
	logCtx := context.WithoutCancel(ctx)
	if gerr != nil {
		row.Outcome, row.OutcomeDetail = OutcomeError, errorDetail(gerr)
		if errors.Is(ctx.Err(), context.Canceled) {
			row.Outcome, row.OutcomeDetail = OutcomeCancelled, ""
		}
		if e, ok := llm.AsError(gerr); ok && e.Kind == llm.ErrAuth {
			s.keyRejected(logCtx, ws, c, e)
		}
		if err := s.account(logCtx, row, res, gerr); err != nil {
			return result{}, err
		}
		if row.Outcome == OutcomeCancelled {
			return result{}, ctx.Err()
		}
		return result{}, s.providerProblem(ctx, c, t.feature, gerr)
	}
	out := result{RequestID: row.ID, Response: res, conn: c, voiceRevision: t.voiceRevision, voiceChars: t.voiceChars}
	out.CostMicros = costMicros(c, res.InputTokens, res.OutputTokens)
	row.InputTokens, row.OutputTokens, row.CostMicros = int32(res.InputTokens), int32(res.OutputTokens), out.CostMicros
	if raw, ok := llm.FirstJSON(res.Text); ok {
		out.JSON = raw
		var r struct {
			Refusal string `json:"refusal"`
		}
		if json.Unmarshal(raw, &r) == nil && strings.TrimSpace(r.Refusal) != "" {
			out.Refusal = strings.TrimSpace(r.Refusal)
		}
	}
	if res.Refused && out.Refusal == "" {
		out.Refusal = strings.TrimSpace(res.Text)
	}
	if res.Refused || out.Refusal != "" {
		row.Outcome = OutcomeRefused
		now := s.now()
		row.DecidedAt = &now
	} else if out.JSON == nil && t.req.JSON && t.feature != FeatureDraft {
		row.Outcome, row.OutcomeDetail = OutcomeError, "unreadable answer"
		if err := s.account(logCtx, row, res, nil); err != nil {
			return result{}, err
		}
		return result{}, problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, i18n.Args{"provider": c.providerName()})
	}
	if err := s.account(logCtx, row, res, nil); err != nil {
		return result{}, err
	}
	if c.Status != "ok" {
		_, _ = s.Q.AIAccountSetStatus(logCtx, store.AIAccountSetStatusParams{ID: c.AccountID, WorkspaceID: ws.ID, Status: "ok", StatusAt: ptr(s.now()), ExpectedRevision: c.AccountRevision})
	}
	return out, nil
}

// finish records a feature's own outcome for a request that answered
// (nothing drafted, for example).
func (s *Service) finish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, outcome, detail string) {
	uid := actorID(p)
	if uid == nil {
		return
	}
	if _, err := s.Q.AIRequestSetOutcome(context.WithoutCancel(ctx), store.AIRequestSetOutcomeParams{ID: id, WorkspaceID: ws.ID, UserID: uid, Outcome: outcome, OutcomeDetail: detail, DecidedAt: ptr(s.now())}); err != nil {
		s.log().WarnContext(ctx, "ai: record an outcome", "err", err)
	}
}

// errorDetail is the activity log's word for a failure ("529 overloaded").
func errorDetail(err error) string {
	e, ok := llm.AsError(err)
	if !ok {
		return clip(err.Error(), 120)
	}
	switch e.Kind {
	case llm.ErrRateLimit:
		return detailRateLimited
	case llm.ErrTimeout:
		return detailTimedOut
	case llm.ErrNetwork:
		return detailUnreachable
	case llm.ErrAuth:
		return fmt.Sprintf("%d%s", e.Status, detailKeyRejected)
	}
	if e.Status > 0 {
		return clip(fmt.Sprintf("%d %s", e.Status, shortCode(e)), 120)
	}
	return clip(string(e.Kind), 120)
}

// shortCode is how the error sentence names a provider's error type:
// "overloaded_error" → "overloaded".
func shortCode(e *llm.Error) string {
	c := strings.TrimSuffix(strings.ToLower(e.Code), "_error")
	switch {
	case e.Kind == llm.ErrOverloaded:
		return "overloaded"
	case c == "":
		return strings.ToLower(http.StatusText(e.Status))
	case c == "unavailable" || c == "serviceunavailable":
		return "unavailable"
	}
	return strings.ReplaceAll(c, "_", " ")
}

// providerProblem says plainly what the provider did; nothing changed.
func (s *Service) providerProblem(ctx context.Context, c *connection, feature string, err error) error {
	name := c.providerName()
	e, ok := llm.AsError(err)
	if !ok {
		return err
	}
	switch e.Kind {
	case llm.ErrRateLimit:
		// 503, not 429: it's the provider limiting the workspace's key, not
		// this API limiting the browser (which the app answers with its own
		// rate-limit notice).
		secs := int(math.Ceil(e.RetryAfter.Seconds()))
		var p *api.Problem
		if secs <= 0 {
			p = problem(ctx, http.StatusServiceUnavailable, codeRateLimited, mRateLimitedNow, i18n.Args{"provider": name})
		} else {
			p = problem(ctx, http.StatusServiceUnavailable, codeRateLimited, mRateLimited, i18n.Args{"provider": name, "seconds": secs})
		}
		return p.WithMeta("retryAfter", secs)
	case llm.ErrAuth:
		if c.personal() {
			return problem(ctx, http.StatusBadGateway, codeKeyRejected, mPersonalKeyRejected, i18n.Args{"provider": name}).WithMeta("status", e.Status, "detail", e.Error())
		}
		return withHelpers(ctx, http.StatusBadGateway, codeKeyRejected, mKeyRejected, mKeyRejectedAlone, i18n.Args{"provider": name}, s.helpers(ctx, c.WorkspaceID)).
			WithMeta("status", e.Status, "detail", e.Error())
	case llm.ErrTimeout:
		return problem(ctx, http.StatusGatewayTimeout, codeProviderError, mTimeout, i18n.Args{"provider": name, "seconds": int(timeoutOf(feature).Seconds())})
	case llm.ErrNetwork:
		if e.Blocked {
			return problem(ctx, http.StatusBadGateway, codePrivate, mBlocked, i18n.Args{"host": hostOf(c.BaseUrl)})
		}
		return problem(ctx, http.StatusBadGateway, codeProviderError, mUnreachable, i18n.Args{"provider": name, "detail": e.Message})
	case llm.ErrBadRequest:
		return problem(ctx, http.StatusBadGateway, codeProviderError, mBadRequest, i18n.Args{"provider": name, "detail": clip(e.Message, 200)}).
			WithMeta("status", e.Status)
	case llm.ErrInvalid, llm.ErrTooLarge:
		return problem(ctx, http.StatusBadGateway, codeProviderError, mUnreadable, i18n.Args{"provider": name})
	}
	args := i18n.Args{"provider": name, "status": e.Status, "code": shortCode(e)}
	m := mProviderError
	if e.Attempts > 1 {
		m = mProviderErrorTwice
	}
	return problem(ctx, http.StatusBadGateway, codeProviderError, m, args).WithMeta("status", e.Status, "attempts", e.Attempts)
}

// keyRejected marks the account's key rejected and tells the members who
// can replace it, once.
func (s *Service) keyRejected(ctx context.Context, ws *store.Workspace, c *connection, e *llm.Error) {
	now := s.now()
	n, err := s.Q.AIAccountSetStatus(ctx, store.AIAccountSetStatusParams{ID: c.AccountID, WorkspaceID: ws.ID, Status: "key_rejected", StatusDetail: clip(e.Error(), 300), StatusAt: &now, ExpectedRevision: c.AccountRevision})
	if err != nil || n == 0 {
		return
	}
	notification := func(l i18n.Lang, body i18n.Msg) notify.Notification {
		return notify.Notification{Kind: notify.KindAIKeyRejected, Severity: "bad", Title: mNotifyKeyTitle.Text(l, nil),
			Body: body.Text(l, i18n.Args{"provider": c.providerName(), "status": e.Status}),
			Link: "/w/" + ws.Slug + "/settings/integrations/ai", WorkspaceID: &ws.ID}
	}
	if c.personal() {
		// A personal key is its owner's to replace.
		err = notify.ToUser(ctx, s.Pool, *c.AccountUserID, notification(applang.ForUserID(ctx, s.Q, *c.AccountUserID), mNotifyPersonalKey))
	} else {
		err = s.notifyMembers(ctx, ws.ID, authz.IntegrationsManage, func(l i18n.Lang) notify.Notification { return notification(l, mNotifyKeyBody) })
	}
	if err != nil {
		s.log().WarnContext(ctx, "ai: notify a rejected key", "err", err)
	}
}

// afterUse sends this month's 80% notice and the cap notice, once each.
func (s *Service) afterUse(ctx context.Context, ws *store.Workspace) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	c, err := s.load(ctx, s.Q, ws.ID)
	if err != nil || !c.connected() || c.CapKind == "none" {
		return
	}
	u, err := s.usage(ctx, ws, c)
	if err != nil {
		return
	}
	start, next := s.month(ws)
	key := monthKey(start)
	// Amounts and the date in each recipient's region.
	args := func(l i18n.Lang) i18n.Args {
		capText, spentText := l.Region.Money(c.CapUsd, 2), l.Region.Money(float64(u.costMicros)/1e6, 2)
		if c.CapKind == "tokens" {
			capText, spentText = l.Region.Int(c.CapTokens), l.Region.Int(u.tokens)
		}
		return i18n.Args{"spent": spentText, "cap": capText, "date": l.Region.DayMonth(next)}
	}
	link := "/w/" + ws.Slug + "/settings/integrations/ai"
	if u.reached {
		if n, err := s.Q.AIWorkspaceSettingsClaimCapNotice(ctx, store.AIWorkspaceSettingsClaimCapNoticeParams{WorkspaceID: ws.ID, Month: key}); err != nil || n == 0 {
			return
		}
		body := mNotifyCapBody
		if c.CapKind == "tokens" {
			body = mNotifyTokenCapBody
		}
		if err := s.notifyMembers(ctx, ws.ID, authz.IntegrationsManage, func(l i18n.Lang) notify.Notification {
			return notify.Notification{Kind: notify.KindAIUsage, Severity: "bad", Title: mNotifyCapTitle.Text(l, nil), Body: body.Text(l, args(l)), Link: link,
				Data: map[string]any{"share": 100}}
		}); err != nil {
			s.log().WarnContext(ctx, "ai: notify the cap", "err", err)
		}
		return
	}
	if !u.notice {
		return
	}
	if n, err := s.Q.AIWorkspaceSettingsClaimNotice(ctx, store.AIWorkspaceSettingsClaimNoticeParams{WorkspaceID: ws.ID, Month: key}); err != nil || n == 0 {
		return
	}
	body := mNotifyUsageBody
	if c.CapKind == "tokens" {
		body = mNotifyTokenUsageBody
	}
	if err := s.notifyMembers(ctx, ws.ID, authz.IntegrationsManage, func(l i18n.Lang) notify.Notification {
		return notify.Notification{Kind: notify.KindAIUsage, Severity: "warn", Title: mNotifyUsageTitle.Text(l, nil), Body: body.Text(l, args(l)), Link: link,
			Data: map[string]any{"share": 80}}
	}); err != nil {
		s.log().WarnContext(ctx, "ai: notify the usage", "err", err)
	}
}

// hostOf is a URL's host name.
func hostOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Hostname()
}

// resolvePrivate resolves a host and reports the first private address
// it has, for the "resolves to 10.0.4.12, a private address" sentence.
func resolvePrivate(ctx context.Context, host string) (string, bool) {
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), ip.IsPrivate() || ip.IsLoopback()
	}
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(rctx, host)
	if err != nil {
		return "", false
	}
	for _, a := range addrs {
		if a.IP.IsPrivate() || a.IP.IsLoopback() {
			return a.IP.String(), true
		}
	}
	return "", false
}

func ptr[T any](v T) *T { return &v }

// atoi parses an int, 0 when it isn't one.
func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
