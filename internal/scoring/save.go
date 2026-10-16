package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// ScoringSaveInput saves the whole set-up: rules, decay and bands.
type ScoringSaveInput struct {
	Config ScoringConfig `json:"config"`
	// BaseVersion is the version the editor opened; a newer one means
	// someone else saved in between.
	BaseVersion int `json:"baseVersion" minimum:"0"`
	// StartAutomations answers the save dialog: start "Score crosses…"
	// automations for contacts who cross only because the rules changed.
	// Not starting them is the default (and recommended).
	StartAutomations bool `json:"startAutomations,omitempty"`
}

func unmarshal(raw []byte, v any) error { return json.Unmarshal(raw, v) }

// refs reads what a configuration may point at.
func refs(ctx context.Context, q *store.Queries, wsID uuid.UUID) (Refs, error) {
	out := Refs{Lists: map[uuid.UUID]bool{}, Forms: map[uuid.UUID]bool{}, Fields: map[string]string{}}
	lists, err := q.ScoringListsOf(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists[l.ID] = true
	}
	forms, err := q.ScoringFormsOf(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, f := range forms {
		out.Forms[f.ID] = true
	}
	fields, err := q.ScoringFieldsOf(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, f := range fields {
		out.Fields[f.Key] = f.Type
	}
	return out, nil
}

// problem turns a validation error into a 422 in the reader's language.
func problem(ctx context.Context, err error) error {
	var fe *FieldError
	if errors.As(err, &fe) {
		return api.Invalid(fe.Location, text(i18n.LangFrom(ctx), fe.Key, fe.Args))
	}
	return err
}

// Save stores a new version of the set-up, audits what changed and starts
// a recalculation when the rules or decay changed. Moving only the bands
// applies at once: scores don't change, only which band they fall in.
func (s *Service) Save(ctx context.Context, ws *store.Workspace, in ScoringSaveInput) (ScoringPage, error) {
	cfg := in.Config.Clone()
	cfg.Normalize()
	r, err := refs(ctx, s.Q, ws.ID)
	if err != nil {
		return ScoringPage{}, err
	}
	if err := cfg.Validate(r); err != nil {
		return ScoringPage{}, problem(ctx, err)
	}
	now := s.now()
	actor := actorID(ctx)
	var recalcID uuid.UUID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.ScoringConfigGetForUpdate(ctx, ws.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			row, err = q.ScoringConfigCreate(ctx, store.ScoringConfigCreateParams{WorkspaceID: ws.ID, SetUpBy: actor, At: now})
		}
		if err != nil {
			return err
		}
		if int(row.Version) != in.BaseVersion {
			return api.Conflict(text(i18n.LangFrom(ctx), "error.conflict", nil))
		}
		prev := emptyConfig()
		if row.Version > 0 {
			v, err := q.ScoringVersionGet(ctx, store.ScoringVersionGetParams{WorkspaceID: ws.ID, Version: row.Version})
			if err != nil {
				return err
			}
			if prev, err = ParseConfig(v.Config); err != nil {
				return err
			}
		}
		next := row.Version + 1
		if err := q.ScoringVersionInsert(ctx, store.ScoringVersionInsertParams{WorkspaceID: ws.ID, Version: next, Config: marshal(cfg), CreatedBy: actor, CreatedAt: now}); err != nil {
			return err
		}
		if err := q.ScoringConfigSetVersion(ctx, store.ScoringConfigSetVersionParams{WorkspaceID: ws.ID, Version: next, At: now}); err != nil {
			return err
		}
		names, err := nameRefs(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if err := s.auditChanges(ctx, q, ws, prev, cfg, row.Version == 0, names); err != nil {
			return err
		}
		running, err := q.ScoringRecalcRunning(ctx, ws.ID)
		hasRunning := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if row.Version > 0 && !RulesChanged(prev, cfg) && !hasRunning && row.AppliedVersion == row.Version {
			// Only the bands moved: nothing to recalculate.
			return q.ScoringConfigSetApplied(ctx, store.ScoringConfigSetAppliedParams{WorkspaceID: ws.ID, AppliedVersion: next, At: now})
		}
		if err := q.ScoringRecalcSupersede(ctx, store.ScoringRecalcSupersedeParams{WorkspaceID: ws.ID, At: &now}); err != nil {
			return err
		}
		if hasRunning {
			if err := q.ScoringRecalcClearStage(ctx, running.ID); err != nil {
				return err
			}
		}
		total, err := q.ScoringCountContacts(ctx, ws.ID)
		if err != nil {
			return err
		}
		reason := "rules"
		if row.Version == 0 {
			reason = "setup"
		}
		label := changeLabel(prev, cfg, row.Version == 0, names)
		rc, err := q.ScoringRecalcInsert(ctx, store.ScoringRecalcInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Version: next, Reason: reason,
			StartAutomations: in.StartAutomations, Total: total, Label: label, Changes: marshal(changesOf(prev, cfg, row.Version == 0)), StartedBy: actor, StartedAt: now})
		if err != nil {
			return err
		}
		recalcID = rc.ID
		choice := "not started"
		if in.StartAutomations {
			choice = "started"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "scoring.recalculation_started", TargetType: "scoring", TargetID: ws.ID.String(), TargetName: "Scoring",
			After:    map[string]any{"version": next, "contacts": total, "start_automations": in.StartAutomations},
			Metadata: map[string]any{"summary": fmt.Sprintf("%s · %s contacts · \"Score crosses…\" automations %s for contacts who cross only because of the change", label, thousands(total), choice)}})
	})
	if err != nil {
		return ScoringPage{}, err
	}
	if recalcID != uuid.Nil {
		if err := s.startRecalc(ctx, recalcID); err != nil {
			return ScoringPage{}, err
		}
	}
	return s.Page(ctx, ws)
}

// Retry starts a failed recalculation again where it stopped.
func (s *Service) Retry(ctx context.Context, ws *store.Workspace) (ScoringPage, error) {
	now := s.now()
	var id uuid.UUID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rc, err := q.ScoringRecalcLatest(ctx, ws.ID)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && rc.Status != "failed") {
			return api.Conflict(text(i18n.LangFrom(ctx), "error.nothingToRetry", nil))
		}
		if err != nil {
			return err
		}
		id = rc.ID
		if err := q.ScoringRecalcRestart(ctx, store.ScoringRecalcRestartParams{ID: rc.ID, StartedBy: actorID(ctx), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "scoring.recalculation_started", TargetType: "scoring", TargetID: ws.ID.String(), TargetName: "Scoring",
			After:    map[string]any{"version": rc.Version, "contacts": rc.Total, "start_automations": rc.StartAutomations, "retry": true},
			Metadata: map[string]any{"summary": fmt.Sprintf("Try again · resumes at %s of %s", thousands(rc.Done), thousands(rc.Total))}})
	})
	if err != nil {
		return ScoringPage{}, err
	}
	if err := s.startRecalc(ctx, id); err != nil {
		return ScoringPage{}, err
	}
	return s.Page(ctx, ws)
}

// refNames are the names rules point at, for audit entries and labels.
type refNames struct {
	lists map[string]string
	forms map[string]string
}

func nameRefs(ctx context.Context, q *store.Queries, wsID uuid.UUID) (refNames, error) {
	out := refNames{lists: map[string]string{}, forms: map[string]string{}}
	lists, err := q.ScoringListsOf(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.lists[l.ID.String()] = l.Name
	}
	forms, err := q.ScoringFormsOf(ctx, wsID)
	if err != nil {
		return out, err
	}
	for _, f := range forms {
		out.forms[f.ID.String()] = f.Name
	}
	return out, nil
}

// Describe says a rule in words, in English, for the audit log: "Click on
// a link containing acme.io/pricing". The app writes its own sentence.
func Describe(r ScoringRule, n refNames) string {
	switch r.Kind {
	case KindOpen:
		if l := n.lists[r.ListID]; l != "" {
			return "Human open of a " + l + " email"
		}
		return "Human open of any email"
	case KindClick:
		if r.Contains != "" {
			return "Click on a link containing " + r.Contains
		}
		return "Click in any email"
	case KindForm:
		if f := n.forms[r.FormID]; f != "" {
			return "Submitted the form “" + f + "”"
		}
		return "Submitted a form"
	case KindAPIEvent:
		return "API event " + r.Event
	case KindSoftBounce:
		return "Soft bounce, not delivered after the retries"
	case KindUnsubscribe:
		if l := n.lists[r.ListID]; l != "" {
			return "Unsubscribed from " + l
		}
		return "Unsubscribed from a list"
	case KindField:
		switch r.Op {
		case OpIsSet:
			return "Field " + r.Field + " is set"
		case OpIsNot:
			return "Field " + r.Field + " is not " + orList(r.Values)
		}
		return "Field " + r.Field + " is " + orList(r.Values)
	}
	return r.Kind
}

func orList(vs []string) string {
	switch len(vs) {
	case 0:
		return ""
	case 1:
		return vs[0]
	}
	return strings.Join(vs[:len(vs)-1], ", ") + " or " + vs[len(vs)-1]
}

// capWords says a cap in words for the audit log ("6 a week", "once a month").
func capWords(c ScoringRuleCap) string {
	switch c.Kind {
	case CapOnce:
		return "once"
	case CapOnceDay:
		return "once a day"
	case CapOnceWeek:
		return "once a week"
	case CapOnceMonth:
		return "once a month"
	case CapPointsDay:
		return fmt.Sprintf("%d a day", c.Points)
	case CapPointsWeek:
		return fmt.Sprintf("%d a week", c.Points)
	case CapPointsMonth:
		return fmt.Sprintf("%d a month", c.Points)
	case CapWhileTrue:
		return "while true"
	}
	return "no cap"
}

func signed(n int) string {
	if n > 0 {
		return fmt.Sprintf("+%d", n)
	}
	return fmt.Sprintf("−%d", -n)
}

// ruleIndex maps rule ids to their position (1-based, as the table numbers them).
func ruleIndex(c ScoringConfig) map[string]int {
	out := map[string]int{}
	for i, r := range c.Rules {
		out[r.ID] = i + 1
	}
	return out
}

func ruleSnapshot(r ScoringRule, n refNames) map[string]any {
	return map[string]any{"rule": Describe(r, n), "points": r.Points, "cap": capWords(r.Cap), "on": r.Enabled}
}

// auditChanges writes an entry per changed rule, and for decay and bands.
func (s *Service) auditChanges(ctx context.Context, q *store.Queries, ws *store.Workspace, prev, next ScoringConfig, first bool, n refNames) error {
	log := func(action, target string, before, after any, summary string) error {
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "scoring", TargetID: ws.ID.String(), TargetName: target,
			Before: before, After: after, Metadata: map[string]any{"summary": summary}})
	}
	if first {
		rules := []map[string]any{}
		for _, r := range next.Rules {
			rules = append(rules, ruleSnapshot(r, n))
		}
		decay := "decay off"
		if next.Decay.Enabled {
			decay = fmt.Sprintf("decay after %d weeks, −%d%% a week", next.Decay.AfterWeeks, next.Decay.Percent)
		}
		summary := fmt.Sprintf("bands Warm %d, Hot %d · %s", next.Bands.WarmFrom, next.Bands.HotFrom, decay)
		return log("scoring.rules_created", fmt.Sprintf("Scoring · %d rules", len(next.Rules)), nil,
			map[string]any{"rules": rules, "decay": next.Decay, "bands": next.Bands}, summary)
	}
	pi, ni := ruleIndex(prev), ruleIndex(next)
	old := map[string]ScoringRule{}
	for _, r := range prev.Rules {
		old[r.ID] = r
	}
	for _, r := range next.Rules {
		o, ok := old[r.ID]
		target := fmt.Sprintf("Rule %d · %s", ni[r.ID], Describe(r, n))
		switch {
		case !ok:
			if err := log("scoring.rule_created", target, nil, ruleSnapshot(r, n), fmt.Sprintf("%s · %s", signed(r.Points), capWords(r.Cap))); err != nil {
				return err
			}
		case sameRule(o, r):
		case o.Enabled && !r.Enabled && sameMatch(o, r) && o.Points == r.Points && o.Cap == r.Cap:
			if err := log("scoring.rule_switched_off", target, ruleSnapshot(o, n), ruleSnapshot(r, n), "stops counting; kept"); err != nil {
				return err
			}
		case !o.Enabled && r.Enabled && sameMatch(o, r) && o.Points == r.Points && o.Cap == r.Cap:
			if err := log("scoring.rule_switched_on", target, ruleSnapshot(o, n), ruleSnapshot(r, n), "counts again"); err != nil {
				return err
			}
		default:
			if err := log("scoring.rule_updated", target, ruleSnapshot(o, n), ruleSnapshot(r, n), ruleDiff(o, r, n)); err != nil {
				return err
			}
		}
	}
	for _, o := range prev.Rules {
		if _, ok := ni[o.ID]; !ok {
			if err := log("scoring.rule_deleted", fmt.Sprintf("Rule %d · %s", pi[o.ID], Describe(o, n)), ruleSnapshot(o, n), nil, "deleted"); err != nil {
				return err
			}
		}
	}
	if prev.Decay != next.Decay {
		if err := log("scoring.decay_updated", "Scoring · decay", prev.Decay, next.Decay, decayWords(prev.Decay)+" → "+decayWords(next.Decay)); err != nil {
			return err
		}
	}
	if prev.Bands != next.Bands {
		summary := fmt.Sprintf("Warm %d, Hot %d → Warm %d, Hot %d", prev.Bands.WarmFrom, prev.Bands.HotFrom, next.Bands.WarmFrom, next.Bands.HotFrom)
		if err := log("scoring.bands_updated", "Scoring · bands", prev.Bands, next.Bands, summary); err != nil {
			return err
		}
	}
	return nil
}

func decayWords(d ScoringDecay) string {
	if !d.Enabled {
		return "off"
	}
	return fmt.Sprintf("after %d weeks, −%d%% a week", d.AfterWeeks, d.Percent)
}

func ruleDiff(o, r ScoringRule, n refNames) string {
	var parts []string
	if !sameMatch(o, r) {
		parts = append(parts, Describe(o, n)+" → "+Describe(r, n))
	}
	if o.Points != r.Points {
		parts = append(parts, signed(o.Points)+" → "+signed(r.Points))
	}
	if o.Cap != r.Cap {
		parts = append(parts, "cap "+capWords(o.Cap)+" → "+capWords(r.Cap))
	}
	if o.Enabled != r.Enabled {
		if r.Enabled {
			parts = append(parts, "switched on")
		} else {
			parts = append(parts, "switched off")
		}
	}
	return strings.Join(parts, " · ")
}

// changeLabel says what a save changed, for the recalculation: "Rule 3:
// +15 → +25", "8 rules set up", "3 changes".
func changeLabel(prev, next ScoringConfig, first bool, n refNames) string {
	if first {
		if len(next.Rules) == 1 {
			return "1 rule set up"
		}
		return fmt.Sprintf("%d rules set up", len(next.Rules))
	}
	var changes []string
	pi, ni := ruleIndex(prev), ruleIndex(next)
	old := map[string]ScoringRule{}
	for _, r := range prev.Rules {
		old[r.ID] = r
	}
	for _, r := range next.Rules {
		o, ok := old[r.ID]
		switch {
		case !ok:
			changes = append(changes, fmt.Sprintf("Rule %d added", ni[r.ID]))
		case !sameRule(o, r):
			changes = append(changes, fmt.Sprintf("Rule %d: %s", ni[r.ID], ruleDiff(o, r, n)))
		}
	}
	for _, o := range prev.Rules {
		if _, ok := ni[o.ID]; !ok {
			changes = append(changes, fmt.Sprintf("Rule %d deleted", pi[o.ID]))
		}
	}
	if prev.Decay != next.Decay {
		changes = append(changes, "Decay: "+decayWords(next.Decay))
	}
	if prev.Bands != next.Bands {
		changes = append(changes, fmt.Sprintf("Bands: Warm %d, Hot %d", next.Bands.WarmFrom, next.Bands.HotFrom))
	}
	switch len(changes) {
	case 0:
		return "No changes"
	case 1:
		return changes[0]
	}
	return fmt.Sprintf("%d changes", len(changes))
}

// ScoringChanges is what a save changed, as data: the numbers of the rules
// added, changed or deleted, and whether decay or the bands moved.
type ScoringChanges struct {
	Setup bool  `json:"setup"`
	Rules []int `json:"rules"`
	Decay bool  `json:"decay"`
	Bands bool  `json:"bands"`
}

func changesOf(prev, next ScoringConfig, first bool) ScoringChanges {
	out := ScoringChanges{Setup: first, Rules: []int{}, Decay: prev.Decay != next.Decay, Bands: prev.Bands != next.Bands}
	if first {
		return out
	}
	pi, ni := ruleIndex(prev), ruleIndex(next)
	old := map[string]ScoringRule{}
	for _, r := range prev.Rules {
		old[r.ID] = r
	}
	for _, r := range next.Rules {
		if o, ok := old[r.ID]; !ok || !sameRule(o, r) {
			out.Rules = append(out.Rules, ni[r.ID])
		}
	}
	for _, o := range prev.Rules {
		if _, ok := ni[o.ID]; !ok {
			out.Rules = append(out.Rules, pi[o.ID])
		}
	}
	return out
}

// actorID is the signed-in person, nil for API keys and agents.
func actorID(ctx context.Context) *uuid.UUID {
	if u := api.UserFrom(ctx); u != nil {
		id := u.ID
		return &id
	}
	return nil
}
