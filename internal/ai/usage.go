package ai

import (
	"context"
	"slices"

	"github.com/hostrepute/reputecast/internal/store"
)

// usage is this month's use against the cap.
type usage struct {
	costMicros   int64
	inputTokens  int64
	outputTokens int64
	tokens       int64
	requests     int64
	// share of the cap used (0 without a cap), notice at 80%, reached at 100%.
	share   float64
	notice  bool
	reached bool
}

// usage sums this month's requests.
func (s *Service) usage(ctx context.Context, ws *store.Workspace, c *connection) (usage, error) {
	start, _ := s.month(ws)
	t, err := s.Q.AIUsageTotal(ctx, store.AIUsageTotalParams{WorkspaceID: ws.ID, Since: start})
	if err != nil {
		return usage{}, err
	}
	u := usage{costMicros: t.CostMicros, inputTokens: t.InputTokens, outputTokens: t.OutputTokens, tokens: t.InputTokens + t.OutputTokens, requests: t.Requests}
	switch c.CapKind {
	case "usd":
		if c.CapUsd > 0 {
			u.share = float64(u.costMicros) / (c.CapUsd * 1e6)
		}
	case "tokens":
		if c.CapTokens > 0 {
			u.share = float64(u.tokens) / float64(c.CapTokens)
		}
	}
	u.reached = c.CapKind != "none" && u.share >= 1
	u.notice = !u.reached && u.share >= NoticeShare
	return u, nil
}

// AIUsageFeature is one feature's use this month.
type AIUsageFeature struct {
	Feature    string `json:"feature" enum:"subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary"`
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
	out := make([]AIUsageFeature, 0, len(Features))
	for _, f := range Features {
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
