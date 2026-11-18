package ai

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// usage is this month's use against the cap.
type usage struct {
	costMicros     int64
	inputTokens    int64
	outputTokens   int64
	tokens         int64
	requests       int64
	reservedCost   int64
	reservedTokens int64
	uncertain      int64
	// share of the cap used (0 without a cap), notice at 80%, reached at 100%.
	share   float64
	notice  bool
	reached bool
}

// usage sums this month's requests against a model's budget: the
// workspace's, or for a personal key its owner's own.
func (s *Service) usage(ctx context.Context, ws *store.Workspace, c *connection) (usage, error) {
	return s.usageIn(ctx, s.Q, ws, c)
}

// usageIn is usage read through q: a caller in a transaction passes its
// own, so the check never waits for a second connection of the pool while
// it holds one (and its locks).
func (s *Service) usageIn(ctx context.Context, q *store.Queries, ws *store.Workspace, c *connection) (usage, error) {
	start, _ := s.month(ws)
	if c.personal() {
		return s.personalUsageIn(ctx, q, ws.ID, *c.AccountUserID, start, c.PersonalCapUsd)
	}
	t, err := q.AIUsageTotal(ctx, store.AIUsageTotalParams{WorkspaceID: ws.ID, Since: start})
	if err != nil {
		return usage{}, err
	}
	u := usage{costMicros: t.CostMicros, inputTokens: t.InputTokens, outputTokens: t.OutputTokens, tokens: t.InputTokens + t.OutputTokens, requests: t.Requests}
	held, err := q.AIBudgetMonthly(ctx, ws.ID)
	if err != nil {
		return usage{}, err
	}
	u.reservedCost, u.reservedTokens, u.uncertain = held.CostMicros, held.Tokens, held.Uncertain
	switch c.CapKind {
	case "usd":
		if c.CapUsd > 0 {
			u.share = float64(u.costMicros+u.reservedCost) / (c.CapUsd * 1e6)
		}
	case "tokens":
		if c.CapTokens > 0 {
			u.share = float64(u.tokens+u.reservedTokens) / float64(c.CapTokens)
		}
	}
	u.reached = c.CapKind != "none" && u.share >= 1
	u.notice = !u.reached && u.share >= NoticeShare
	return u, nil
}

// personalUsage is a person's use of their own keys this month against
// their cap (in dollars).
func (s *Service) personalUsage(ctx context.Context, ws, user uuid.UUID, start time.Time, capUSD float64) (usage, error) {
	return s.personalUsageIn(ctx, s.Q, ws, user, start, capUSD)
}

func (s *Service) personalUsageIn(ctx context.Context, q *store.Queries, ws, user uuid.UUID, start time.Time, capUSD float64) (usage, error) {
	t, err := q.AIPersonalUsageTotal(ctx, store.AIPersonalUsageTotalParams{WorkspaceID: ws, UserID: &user, Since: start})
	if err != nil {
		return usage{}, err
	}
	held, err := q.AIBudgetPersonal(ctx, store.AIBudgetPersonalParams{WorkspaceID: ws, UserID: &user})
	if err != nil {
		return usage{}, err
	}
	u := usage{costMicros: t.CostMicros, requests: t.Requests, reservedCost: held.CostMicros, uncertain: held.Uncertain}
	if capUSD > 0 {
		u.share = float64(u.costMicros+u.reservedCost) / (capUSD * 1e6)
	}
	u.reached = u.share >= 1
	u.notice = !u.reached && u.share >= NoticeShare
	return u, nil
}

// AIUsageFeature is one feature's use this month.
type AIUsageFeature struct {
	Feature    string `json:"feature" enum:"subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary,brand_voice,draft,agent,connection_test,assistant"`
	CostMicros int64  `json:"costMicros"`
	Tokens     int64  `json:"tokens"`
	Requests   int64  `json:"requests"`
}

// AIUsageMember is one member's use this month.
type AIUsageMember struct {
	UserID     string `json:"userId,omitempty"`
	Name       string `json:"name"`
	CostMicros int64  `json:"costMicros"`
	Tokens     int64  `json:"tokens"`
	Requests   int64  `json:"requests"`
}

// byFeature lists every feature, in the settings' order, with its use.
func (s *Service) byFeature(ctx context.Context, ws *store.Workspace) ([]AIUsageFeature, error) {
	start, _ := s.month(ws)
	rows, err := s.Q.AIUsageByFeature(ctx, store.AIUsageByFeatureParams{WorkspaceID: ws.ID, Since: start})
	if err != nil {
		return nil, err
	}
	features := append(slices.Clone(Features), FeatureAgent, FeatureConnectionTest, FeatureAssistant)
	out := make([]AIUsageFeature, 0, len(features))
	for _, f := range features {
		e := AIUsageFeature{Feature: f}
		for _, r := range rows {
			if r.Feature == f {
				e.CostMicros, e.Tokens, e.Requests = r.CostMicros, r.Tokens, r.Requests
			}
		}
		out = append(out, e)
	}
	return out, nil
}

// byMember lists the members who asked this month, the biggest first.
func (s *Service) byMember(ctx context.Context, ws *store.Workspace) ([]AIUsageMember, error) {
	start, _ := s.month(ws)
	rows, err := s.Q.AIUsageByMember(ctx, store.AIUsageByMemberParams{WorkspaceID: ws.ID, Since: start})
	if err != nil {
		return nil, err
	}
	out := make([]AIUsageMember, 0, len(rows))
	for _, r := range rows {
		m := AIUsageMember{Name: r.UserName, CostMicros: r.CostMicros, Tokens: r.Tokens, Requests: r.Requests}
		if r.UserID != nil {
			m.UserID = r.UserID.String()
		}
		if m.Name == "" {
			m.Name = r.UserEmail
		}
		out = append(out, m)
	}
	slices.SortStableFunc(out, func(a, b AIUsageMember) int {
		switch {
		case a.CostMicros > b.CostMicros:
			return -1
		case a.CostMicros < b.CostMicros:
			return 1
		}
		return 0
	})
	return out, nil
}
