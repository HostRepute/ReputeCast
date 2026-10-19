package ai

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// AIActivityRow is one request of the AI activity log. It never holds what
// was sent or returned.
type AIActivityRow struct {
	ID           string    `json:"id"`
	At           time.Time `json:"at"`
	UserID       string    `json:"userId,omitempty"`
	Member       string    `json:"member"`
	MemberColor  string    `json:"memberColor,omitempty"`
	Feature      string    `json:"feature" enum:"subject_ideas,rewrite,translate,alt_text,accessibility_fix,segment_draft,report_summary"`
	Action       string    `json:"action,omitempty" doc:"The rewrite's action, or the language translated to"`
	TargetKind   string    `json:"targetKind,omitempty"`
	TargetID     string    `json:"targetId,omitempty"`
	On           string    `json:"on" doc:"What it was about: Weekly Digest #143"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Tokens       int64     `json:"tokens"`
	InputTokens  int64     `json:"inputTokens"`
	OutputTokens int64     `json:"outputTokens"`
	CostMicros   int64     `json:"costMicros"`
	Outcome      string    `json:"outcome" enum:"pending,accepted,accepted_edits,kept,rejected,discarded,refused,error,nothing_drafted,cancelled"`
	Detail       string    `json:"detail,omitempty" doc:"1 of 5, 529 overloaded"`
}

// AIActivityList is a page of the log.
type AIActivityList struct {
	Items []AIActivityRow `json:"items"`
	// MonthRequests counts this month's requests; Members are the members
	// in the log, for the filter.
	MonthRequests int64              `json:"monthRequests"`
	Members       []AIActivityMember `json:"members"`
	Next          string             `json:"next,omitempty"`
	RetentionDays int                `json:"retentionDays"`
}

// AIActivityMember is a member who asked this month, for the filter.
type AIActivityMember struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AIActivityFilter narrows the log.
type AIActivityFilter struct {
	Member  string
	Feature string
	// Outcome: accepted (with edits and kept too), rejected (discarded too),
	// refused, error, or any.
	Outcome string
	Cursor  string
	Limit   int
}

func outcomesOf(f string) []string {
	switch f {
	case "accepted":
		return []string{OutcomeAccepted, OutcomeAcceptedEdits}
	case "kept":
		return []string{OutcomeKept}
	case "rejected":
		return []string{OutcomeRejected, OutcomeDiscarded}
	case "refused":
		return []string{OutcomeRefused}
	case "error":
		return []string{OutcomeError, OutcomeCancelled}
	case "nothing_drafted":
		return []string{OutcomeNothingDrafted}
	case "pending":
		return []string{OutcomePending}
	}
	return nil
}

func cursorOf(at time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id.String()))
}

func parseCursor(c string) (*time.Time, *uuid.UUID) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return nil, nil
	}
	at, id, ok := strings.Cut(string(b), "|")
	if !ok {
		return nil, nil
	}
	t, err1 := time.Parse(time.RFC3339Nano, at)
	u, err2 := uuid.Parse(id)
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	return &t, &u
}

// Activity lists the last 90 days of requests, newest first.
func (s *Service) Activity(ctx context.Context, ws *store.Workspace, f AIActivityFilter) (AIActivityList, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	p := store.AIRequestsParams{WorkspaceID: ws.ID, Since: s.now().Add(-Retention), Lim: int32(limit + 1)}
	if id, err := uuid.Parse(f.Member); err == nil {
		p.UserID = &id
	}
	if f.Feature != "" {
		if !validFeature(f.Feature) {
			return AIActivityList{}, invalid(ctx, "query.feature", mUnknownFeature, nil)
		}
		p.Feature = &f.Feature
	}
	if o := outcomesOf(f.Outcome); o != nil {
		p.Outcomes = o
	}
	if f.Cursor != "" {
		p.Before, p.BeforeID = parseCursor(f.Cursor)
	}
	rows, err := s.Q.AIRequests(ctx, p)
	if err != nil {
		return AIActivityList{}, err
	}
	out := AIActivityList{Items: []AIActivityRow{}, Members: []AIActivityMember{}, RetentionDays: int(Retention.Hours() / 24)}
	for i, r := range rows {
		if i == limit {
			out.Next = cursorOf(rows[i-1].CreatedAt, rows[i-1].ID)
			break
		}
		row := AIActivityRow{ID: r.ID.String(), At: r.CreatedAt, Member: r.UserName, MemberColor: r.UserColor, Feature: r.Feature, Action: r.Action, TargetKind: r.TargetKind,
			On: r.TargetLabel, Provider: r.Provider, Model: r.Model, InputTokens: int64(r.InputTokens), OutputTokens: int64(r.OutputTokens),
			Tokens: int64(r.InputTokens) + int64(r.OutputTokens), CostMicros: r.CostMicros, Outcome: r.Outcome, Detail: r.OutcomeDetail}
		if row.Member == "" {
			row.Member = r.UserEmail
		}
		if r.UserID != nil {
			row.UserID = r.UserID.String()
		}
		if r.TargetID != nil {
			row.TargetID = r.TargetID.String()
		}
		out.Items = append(out.Items, row)
	}
	start, _ := s.month(ws)
	if n, err := s.Q.AIRequestCount(ctx, store.AIRequestCountParams{WorkspaceID: ws.ID, Since: start}); err == nil {
		out.MonthRequests = n
	}
	members, err := s.byMember(ctx, ws)
	if err == nil {
		for _, m := range members {
			if m.UserID != "" {
				out.Members = append(out.Members, AIActivityMember{ID: m.UserID, Name: m.Name})
			}
		}
	}
	return out, nil
}

// AIOutcomeInput records what a person did with a suggestion.
type AIOutcomeInput struct {
	Outcome string `json:"outcome" enum:"accepted,accepted_edits,kept,rejected,discarded"`
	// Detail: "1 of 5" for subject ideas.
	Detail string `json:"detail,omitempty" maxLength:"40"`
}

// RecordOutcome stores the outcome of a suggestion the person asked for.
func (s *Service) RecordOutcome(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIOutcomeInput) error {
	uid := actorID(p)
	if uid == nil {
		return api.Forbidden(authz.AIUse, nil)
	}
	r, err := s.Q.AIRequest(ctx, store.AIRequestParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return api.NotFound("Suggestion")
	}
	if err != nil {
		return err
	}
	if r.UserID == nil || *r.UserID != *uid || s.now().Sub(r.CreatedAt) > 24*time.Hour {
		return problem(ctx, http.StatusConflict, "ai_suggestion_gone", mSuggestionGone, nil)
	}
	n, err := s.Q.AIRequestSetOutcome(ctx, store.AIRequestSetOutcomeParams{ID: id, WorkspaceID: ws.ID, UserID: uid, Outcome: in.Outcome, OutcomeDetail: strings.TrimSpace(in.Detail), DecidedAt: ptr(s.now())})
	if err != nil {
		return err
	}
	if n == 0 {
		return problem(ctx, http.StatusConflict, "ai_suggestion_gone", mSuggestionGone, nil)
	}
	return nil
}

// Purge drops the activity log's rows older than 90 days.
func (s *Service) Purge(ctx context.Context) (int64, error) {
	return s.Q.AIRequestsPurge(ctx, s.now().Add(-Retention))
}
