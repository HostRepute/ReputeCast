package demo

import (
	"context"
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/scoring"
)

// What reads the score (ScoringRules › Used by, ScoringUses): Marcus
// Chen's "Sales-ready handoff", live since yesterday with its 37 contacts,
// and Sofia Rossi's "Hot on paid plans" segment; with the audit history of
// the set-up. Conditional content's "Book a demo" block is phase 3's.

func (w *sgWorld) uses(ctx context.Context) error {
	s := w.s
	tx := s.Tx
	marcus, sofia := s.Users["marcus@acme.io"], s.Users["sofia@acme.io"]

	graph := automations.Layout(automations.AutomationGraph{
		Nodes: []autoFlowNode{
			autoNode("n1", automations.NodeTrigger, 0, 0, autoFlowData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerScoreCrosses,
				Direction: "rises", Threshold: 50, OnceInDays: 90, Reentry: "always"}}),
			autoNode("n2", automations.NodeFilter, 0, 0, autoFlowData{Condition: &automations.AutomationCondition{Kind: "field", Field: "plan", Op: "is_one_of", Value: "team,business"}}),
			autoNode("n3", automations.NodeList, 0, 0, autoFlowData{Action: &automations.AutomationListAction{Kind: "add_tag", Tag: "sales-ready"}}),
			// A demo step: its address is fictional, so it is never called.
			autoNode("n4", automations.NodeWebhook, 0, 0, autoFlowData{Webhook: &automations.AutomationWebhook{Demo: true, Name: "Send to the CRM",
				URL: "https://api.acme.io/crm/leads", Method: "POST"}}),
		},
		Edges: []automations.AutomationEdge{autoEdge("n1", "n2", ""), autoEdge("n2", "n3", ""), autoEdge("n3", "n4", "")},
	})
	flow := ID()
	created := s.designDay(2, 16, 5)
	if err := autoInsertFlow(ctx, s, autoFlowSeed{
		id: flow, name: "Sales-ready handoff", description: "Tag Hot customers on paid plans and send them to the CRM", status: "live",
		live: 1, created: created, createdBy: marcus.ID, updated: w.liveSince,
		versions: []autoVersionSeed{{graph: graph, by: marcus.ID, created: created, published: autoPtrTime(w.liveSince), summary: "first published version"}},
	}); err != nil {
		return err
	}

	// The 37 who crossed 50 since: 31 passed the filter, were tagged and
	// sent to the CRM; 6 aren't on a paid plan and left the flow.
	var runs [][]any
	var tagged []uuid.UUID
	passed, filtered := 0, 0
	for _, c := range w.contacts {
		if c.sc.role != "entrant" {
			continue
		}
		at := crossedAt(c, 50)
		if at.IsZero() {
			continue
		}
		entered := at.Add(time.Duration(5+w.rng.IntN(40)) * time.Second)
		if c.sc.fields > 0 {
			runs = append(runs, []any{ID(), flow, int32(1), c.id, "n4", "completed", json.RawMessage(`{}`), entered, entered.Add(2 * time.Second), entered.Add(3 * time.Second), ""})
			tagged = append(tagged, c.id)
			passed++
		} else {
			runs = append(runs, []any{ID(), flow, int32(1), c.id, "n2", "exited", json.RawMessage(`{}`), entered, entered, entered.Add(time.Second), "filtered"})
			filtered++
		}
	}
	if err := sgCopy(ctx, tx, "automation_runs", []string{"id", "automation_id", "version", "contact_id", "node_id", "status", "state", "entered_at",
		"node_entered_at", "finished_at", "exit_reason"}, runs); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE contacts SET tags = array_append(tags, 'sales-ready') WHERE id = ANY ($1) AND NOT ('sales-ready' = ANY (tags))`, tagged); err != nil {
		return err
	}
	entered := passed + filtered
	since := int(w.day(0).Sub(w.liveSince.In(w.loc)).Hours()/24) + 1
	if err := autoInsertStats(ctx, s, []autoNodeStats{
		{a: flow, v: 1, node: "n1", from: since, c: autoCounters{"entered": int64(entered), "completed": int64(entered)}},
		{a: flow, v: 1, node: "n2", from: since, c: autoCounters{"entered": int64(entered), "completed": int64(passed), "dropped": int64(filtered)}},
		{a: flow, v: 1, node: "n3", from: since, c: autoCounters{"entered": int64(passed), "completed": int64(passed)}},
		{a: flow, v: 1, node: "n4", from: since, c: autoCounters{"entered": int64(passed), "completed": int64(passed)}},
	}); err != nil {
		return err
	}

	// Sofia's segment: Hot on paid plans.
	def, _ := json.Marshal(map[string]any{"groups": []any{map[string]any{"match": "all", "conditions": []any{
		map[string]any{"field": "score_band", "op": "is", "value": "hot"},
		map[string]any{"field": "custom:plan", "op": "is_one_of", "value": "team,business"},
	}}}})
	hot := 0
	for _, c := range w.contacts {
		if c.board != "erased" && c.sc.fields > 0 && c.sc.res.Score >= 50 {
			hot++
		}
	}
	seg := ID()
	if _, err := tx.Exec(ctx, `INSERT INTO segments (id, workspace_id, name, mode, definition, cached_count, counted_at, created_by, updated_by, created_at, updated_at)
		VALUES ($1,$2,'Hot on paid plans','builder',$3,$4,$5,$6,$6,$7,$5)`, seg, w.ws.ID, def, hot, w.now.Add(-6*time.Minute), sofia.ID, s.designDay(2, 14, 30)); err != nil {
		return err
	}

	// The audit history.
	ip := netip.MustParseAddr("203.0.113.9")
	audit := func(at time.Time, actor uuid.UUID, name, action, targetType string, target uuid.UUID, targetName string, meta map[string]any) error {
		m, _ := json.Marshal(meta)
		_, err := tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
			result, ip, user_agent, request_id, metadata) VALUES ($1,$2,$3,'user',$4,$5,$6,$7,$8,$9,'allowed',$10,'',$11,$12)`,
			ID(), w.ws.ID, at, actor, name, action, targetType, target.String(), targetName, ip, "req_"+ID().String()[:8], m)
		return err
	}
	setUp := s.designDay(2, 11, 40)
	if err := audit(setUp, sofia.ID, sofia.Name, "scoring.rules_created", "scoring", w.ws.ID, "Scoring",
		map[string]any{"summary": "8 rules · decay after 4 weeks, 10% a week · Warm from 20, Hot from 50", "version": 1}); err != nil {
		return err
	}
	if err := audit(setUp.Add(time.Second), sofia.ID, sofia.Name, "scoring.recalculation_started", "scoring", w.ws.ID, "Scoring",
		map[string]any{"summary": "Set up with 8 rules · 48,213 contacts", "version": 1}); err != nil {
		return err
	}
	if err := audit(s.designDay(2, 14, 30), sofia.ID, sofia.Name, "segment.create", "segment", seg, "Hot on paid plans",
		map[string]any{"summary": "Score band is Hot, plan is team or business"}); err != nil {
		return err
	}
	if err := audit(created, marcus.ID, marcus.Name, "automation.create", "automation", flow, "Sales-ready handoff",
		map[string]any{"summary": "Trigger: score rises to 50 or more"}); err != nil {
		return err
	}
	return audit(w.liveSince, marcus.ID, marcus.Name, "automation.publish", "automation", flow, "Sales-ready handoff",
		map[string]any{"summary": "v1 live · trigger: score rises to 50 or more, at most once per contact in 90 days"})
}

// crossedAt is when a contact's score last rose to t or more.
func crossedAt(c *sgContact, t int) time.Time {
	var at time.Time
	prev := scoring.Total(0, c.sc.fields)
	for _, st := range c.sc.res.Steps {
		next := scoring.Total(st.After, c.sc.fields)
		if prev < t && next >= t {
			at = st.At
		}
		prev = next
	}
	return at
}
