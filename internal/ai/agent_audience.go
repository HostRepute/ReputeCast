package ai

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Definition contains bound symbols, not user literals. The count snapshot
// is for human review only: even rounded, fixed-per-run counts would allow
// differencing between runs with overlapping arbitrary predicates.
type agentAudience struct {
	Definition segments.SegmentDefinition `json:"definition"`
	AIAgentAudienceSnapshot
}

type AIAgentAudienceSnapshot struct {
	Count    *int64    `json:"count,omitempty"`
	Total    *int64    `json:"total,omitempty"`
	Status   string    `json:"status"`
	At       time.Time `json:"at"`
	Rounding int64     `json:"rounding"`
}

func (a *agentAudience) modelJSON() ([]byte, error) {
	return json.Marshal(struct {
		Definition segments.SegmentDefinition `json:"definition"`
		Status     string                     `json:"status"`
	}{a.Definition, "local_review_only"})
}

type agentStageSegment struct {
	Handle string `json:"handle" pattern:"^[a-z][a-z0-9_]{0,39}$"`
}

// prepareAgentAudience uses only the explicit builder selection, never a
// contact-derived catalog. Every compared literal and custom field key gets
// a run-local symbol, including occurrences in the brief itself.
func (s *Service) prepareAgentAudience(ctx context.Context, ws *store.Workspace, p *authz.Principal, c *agentContext) error {
	d := c.Brief.Audience
	if d == nil {
		return nil
	}
	if !p.Can(authz.SegmentsEdit) {
		return agentProblem(ctx, "permission")
	}
	if len(d.Groups) == 0 || len(d.Groups) > 5 {
		return agentProblem(ctx, "input")
	}
	for _, g := range d.Groups {
		if len(g.Conditions) > 10 {
			return agentProblem(ctx, "input")
		}
	}
	if _, err := s.Segments.CheckProposal(ctx, s.Q, ws.ID, p, *d); err != nil {
		return segments.Problem(ctx, err)
	}
	projected, symbols := segments.ProjectDefinition(*d)
	c.Literals = symbols
	audience := &agentAudience{Definition: projected, AIAgentAudienceSnapshot: AIAgentAudienceSnapshot{Status: "unavailable", At: s.now(), Rounding: 10}}
	count, total, err := s.Segments.PreviewProposal(ctx, ws, p, *d)
	if err != nil && !errors.Is(err, segments.ErrTimeout) {
		return err
	}
	if err == nil {
		audience.Status = "withheld"
		// Suppress small cohorts AND their complements. Rounding and a
		// fixed predicate reduce disclosure; they do not anonymize free text.
		if count >= 20 && total-count >= 20 {
			count, total = count/10*10, total/10*10
			audience.Count, audience.Total, audience.Status = &count, &total, "rounded"
		}
	}
	c.Audience = audience
	return nil
}

// privateLiterals is stored only with local bindings. Apply it to each
// selected text field before serialization, never to JSON keys, instructions
// or the provider's replay signatures. Opaque source bindings stay intact.
type privateLiterals map[string]string

func (l privateLiterals) mask(text string) string {
	if len(l) == 0 {
		return text
	}
	values := make([]string, 0, len(l))
	for value := range l {
		values = append(values, value)
	}
	slices.SortFunc(values, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	patterns := []string{`RC_DRAFT_VALUE_[A-Za-z0-9_]+_END|https://reputecast\.invalid/(?:link|image)/[A-Za-z0-9_]+/end|\bbound_\d+\b`}
	for _, value := range values {
		patterns = append(patterns, regexp.QuoteMeta(norm.NFKC.String(value)))
	}
	// One pass prevents shorter literals from altering replacement symbols.
	return regexp.MustCompile("(?i)"+strings.Join(patterns, "|")).ReplaceAllStringFunc(norm.NFKC.String(text), func(value string) string {
		if draftReserved.MatchString(value) || strings.HasPrefix(strings.ToLower(value), "bound_") {
			return value
		}
		for _, literal := range values {
			if strings.EqualFold(value, norm.NFKC.String(literal)) {
				return l[literal]
			}
		}
		return value
	})
}

func (l privateLiterals) source(t *campaignDraftTask) {
	t.Brief, t.Source = l.mask(t.Brief), l.mask(t.Source)
	t.Subject, t.Preheader = l.mask(t.Subject), l.mask(t.Preheader)
}

func (l privateLiterals) contains(text string) bool {
	for range 3 {
		text = html.UnescapeString(text)
	}
	text = norm.NFKC.String(text)
	return l.mask(text) != text
}

func (s *Service) stageAgentSegment(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, run *store.AiAgentRun, c *agentContext, call llm.ToolCall) (json.RawMessage, bool, error) {
	denied := json.RawMessage(`{"ok":false,"code":"tool_not_allowed"}`)
	var in agentStageSegment
	if decodeDraft(call.Arguments, &in) != nil || c.Brief.Audience == nil || c.Refine != "" {
		return denied, true, nil //nolint:nilerr // Persist the denied tool result without disclosing private arguments.
	}
	i := slices.IndexFunc(c.Plan.Items, func(item AIAgentPlanItem) bool {
		return item.Handle == in.Handle && item.Kind == "segment" && item.Target == ""
	})
	if i < 0 || !p.Can(authz.SegmentsEdit) {
		return denied, true, nil
	}
	item := c.Plan.Items[i]
	if c.Discarded[item.Bundle] {
		return json.RawMessage(`{"ok":false,"code":"bundle_discarded"}`), false, nil
	}
	all, err := q.AIAgentArtifacts(ctx, run.ID)
	if err != nil {
		return nil, false, err
	}
	if slices.ContainsFunc(all, func(a store.AiAgentArtifact) bool { return a.Handle == in.Handle && a.Decision != "" }) {
		return json.RawMessage(`{"ok":false,"code":"already_decided"}`), true, nil
	}
	if _, err := s.Segments.CheckProposal(ctx, q, ws.ID, p, *c.Brief.Audience); err != nil {
		return nil, false, err
	}
	content, err := json.Marshal(c.Brief.Audience)
	if err != nil {
		return nil, false, err
	}
	if err := q.AIAgentArtifactSave(ctx, store.AIAgentArtifactSaveParams{ID: uuid.Must(uuid.NewV7()), RunID: run.ID, Handle: in.Handle, Kind: "segment", Bundle: item.Bundle,
		Content: content, Context: json.RawMessage(`{}`), UpdatedAt: s.now()}); err != nil {
		return nil, false, err
	}
	if err := s.agentAudit(ctx, q, p, *run, "draft", map[string]any{"handle": in.Handle}); err != nil {
		return nil, false, err
	}
	result, _ := json.Marshal(map[string]any{"ok": true, "handle": in.Handle, "staged": true})
	return result, false, nil
}
