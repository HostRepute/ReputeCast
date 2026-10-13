package automations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// The runs engine. A run is one contact's way through one version of a flow.
// Runs are rows in automation_runs: "active" runs are due now, "waiting"
// runs wake at next_at (a delay, a branch window, a send window) or when a
// webhook call finishes. The tick job claims due runs with SKIP LOCKED and
// moves each as far as it can in one transaction, so several workers can run
// side by side and a crash never sends a step twice.

// runState is stored in automation_runs.state.
type runState struct {
	// Wait says what a waiting run waits for: delay, branch, window, webhook.
	Wait  string     `json:"wait,omitempty"`
	Until *time.Time `json:"until,omitempty"`
	// Branch waits watch one message for an open or click.
	Message string `json:"message,omitempty"`
	Want    string `json:"want,omitempty"`
	Human   bool   `json:"human,omitempty"`
	// Done marks a finished webhook call.
	Done bool `json:"done,omitempty"`
	// Messages maps email steps to the message they queued for this contact.
	Messages map[string]string `json:"messages,omitempty"`
	// Event is the trigger event that enrolled the contact.
	Event int64 `json:"event,omitempty"`
}

func parseState(raw []byte) runState {
	var st runState
	_ = json.Unmarshal(raw, &st)
	return st
}

func (st runState) json() json.RawMessage {
	b, _ := json.Marshal(st)
	return b
}

// statKey identifies one step-stats row.
type statKey struct {
	automation uuid.UUID
	version    int32
	node       string
	day        time.Time
}

// statsBuf accumulates counter changes during a batch and writes them once.
type statsBuf map[statKey]*store.BumpAutomationStepStatsParams

func (b statsBuf) at(a uuid.UUID, v int32, node string, now time.Time) *store.BumpAutomationStepStatsParams {
	day := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	k := statKey{a, v, node, day}
	p, ok := b[k]
	if !ok {
		p = &store.BumpAutomationStepStatsParams{AutomationID: a, Version: v, NodeID: node, Day: day, Paths: json.RawMessage(`{}`), Providers: json.RawMessage(`{}`)}
		b[k] = p
	}
	return p
}

func addKey(raw json.RawMessage, key string, n int64) json.RawMessage {
	m := map[string]int64{}
	_ = json.Unmarshal(raw, &m)
	m[key] += n
	out, _ := json.Marshal(m)
	return out
}

func (b statsBuf) flush(ctx context.Context, q *store.Queries) error {
	for _, p := range b {
		if err := q.BumpAutomationStepStats(ctx, *p); err != nil {
			return err
		}
	}
	clear(b)
	return nil
}

// graphCache holds published graphs (they never change).
type graphCache struct {
	s *Service
	q *store.Queries
	m map[string]AutomationGraph
}

func (c *graphCache) get(ctx context.Context, automation uuid.UUID, version int32) (AutomationGraph, error) {
	key := fmt.Sprintf("%s/%d", automation, version)
	if g, ok := c.m[key]; ok {
		return g, nil
	}
	v, err := c.q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: automation, Version: version})
	if err != nil {
		return AutomationGraph{}, err
	}
	g := c.s.graphOf(v.Graph)
	c.m[key] = g
	return g, nil
}

// Dispatch enrolls contacts from pending trigger events. It returns how many
// runs it started.
func (s *Service) Dispatch(ctx context.Context) (int, error) {
	started := 0
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		n, more, err := s.dispatchBatch(ctx, 200)
		started += n
		if err != nil || !more {
			return started, err
		}
	}
	return started, nil
}

func (s *Service) dispatchBatch(ctx context.Context, limit int32) (started int, more bool, err error) {
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		events, err := q.ClaimAutomationEvents(ctx, limit)
		if err != nil {
			return err
		}
		more = len(events) == int(limit)
		if len(events) == 0 {
			return nil
		}
		now := s.now()
		stats := statsBuf{}
		type cacheKey struct {
			ws   uuid.UUID
			kind string
		}
		live := map[cacheKey][]store.LiveAutomationsForTriggerRow{}
		for _, ev := range events {
			k := cacheKey{ev.WorkspaceID, ev.Kind}
			flows, ok := live[k]
			if !ok {
				if flows, err = q.LiveAutomationsForTrigger(ctx, store.LiveAutomationsForTriggerParams{WorkspaceID: ev.WorkspaceID, Kind: ev.Kind}); err != nil {
					return err
				}
				live[k] = flows
			}
			enrolled := 0
			if len(flows) > 0 {
				c, err := q.AutomationContact(ctx, ev.ContactID)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					return err
				}
				if err == nil && c.Status == "active" && c.ErasedAt == nil {
					for _, f := range flows {
						ok, err := s.enroll(ctx, q, stats, f, ev, now)
						if err != nil {
							return err
						}
						if ok {
							enrolled++
						}
					}
				}
			}
			if err := q.MarkAutomationEventProcessed(ctx, store.MarkAutomationEventProcessedParams{ID: ev.ID, Enrolled: int32(enrolled)}); err != nil {
				return err
			}
			started += enrolled
		}
		return stats.flush(ctx, q)
	})
	return started, more, err
}

// triggerMatches decides whether an event starts a flow.
func triggerMatches(t AutomationTrigger, ev store.ClaimAutomationEventsRow) bool {
	switch ev.Kind {
	case TriggerJoinedList:
		if t.ListID != ev.Ref {
			return false
		}
		var p struct {
			Status string `json:"status"`
			Source string `json:"source"`
		}
		_ = json.Unmarshal(ev.Payload, &p)
		if t.ConfirmedOnly && p.Status != "subscribed" {
			return false
		}
		if !t.IncludeImports && strings.HasPrefix(p.Source, "import") {
			return false
		}
		return true
	case TriggerFormSubmitted:
		return t.FormID == ev.Ref
	case TriggerTagAdded:
		return t.Tag == ev.Ref
	case TriggerAPIEvent:
		return t.Event == ev.Ref
	case TriggerDateField:
		return t.Field == ev.Ref
	}
	return false
}

// stepID reads an id stored in a step's settings. A malformed one means
// there is nothing to act on (validation keeps them out of published
// versions), so callers skip instead of failing the run.
func stepID(s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(s)
	return id, err == nil
}

func (s *Service) enroll(ctx context.Context, q *store.Queries, stats statsBuf, f store.LiveAutomationsForTriggerRow, ev store.ClaimAutomationEventsRow, now time.Time) (bool, error) {
	var t AutomationTrigger
	decoded := json.Unmarshal(f.Trigger, &t) == nil
	if !decoded || !triggerMatches(t, ev) || f.LiveVersion == nil {
		return false, nil
	}
	if t.Reentry != "always" {
		exists, err := q.AutomationRunExists(ctx, store.AutomationRunExistsParams{AutomationID: f.ID, ContactID: ev.ContactID})
		if err != nil || exists {
			return false, err
		}
	}
	g := s.graphOf(f.Graph)
	trig := g.Trigger()
	if trig == nil {
		return false, nil
	}
	n, err := q.InsertAutomationRun(ctx, store.InsertAutomationRunParams{
		ID: uuid.Must(uuid.NewV7()), AutomationID: f.ID, Version: *f.LiveVersion, ContactID: ev.ContactID,
		NodeID: trig.ID, State: runState{Event: ev.ID}.json(), EnteredAt: now,
	})
	if err != nil || n == 0 {
		return false, err
	}
	stats.at(f.ID, *f.LiveVersion, trig.ID, now).Entered++
	return true, nil
}

// Tick moves every due run forward. It returns how many runs it processed.
func (s *Service) Tick(ctx context.Context) (int, error) {
	done := 0
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		n, err := s.tickBatch(ctx, 100)
		done += n
		if err != nil || n < 100 {
			return done, err
		}
	}
	return done, nil
}

func (s *Service) tickBatch(ctx context.Context, limit int32) (int, error) {
	count := 0
	now := s.now()
	ctx = withClock(ctx, now)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		runs, err := q.ClaimDueAutomationRuns(ctx, store.ClaimDueAutomationRunsParams{Now: &now, Lim: limit})
		if err != nil {
			return err
		}
		count = len(runs)
		stats := statsBuf{}
		cache := &graphCache{s: s, q: q, m: map[string]AutomationGraph{}}
		for _, r := range runs {
			g, err := cache.get(ctx, r.AutomationID, r.Version)
			if err != nil {
				return err
			}
			ex := &executor{s: s, tx: tx, q: q, stats: stats, g: g, now: now, run: r, st: parseState(r.State)}
			if err := ex.advance(ctx); err != nil {
				return fmt.Errorf("run %s at %s: %w", r.ID, r.NodeID, err)
			}
		}
		return stats.flush(ctx, q)
	})
	return count, err
}

// executor moves one run through its flow.
type executor struct {
	s     *Service
	tx    pgx.Tx
	q     *store.Queries
	stats statsBuf
	g     AutomationGraph
	now   time.Time
	run   store.ClaimDueAutomationRunsRow
	st    runState
	c     *store.AutomationContactRow
}

func (e *executor) stat(node string) *store.BumpAutomationStepStatsParams {
	return e.stats.at(e.run.AutomationID, e.run.Version, node, e.now)
}

func (e *executor) contact(ctx context.Context) (*store.AutomationContactRow, error) {
	if e.c != nil {
		return e.c, nil
	}
	c, err := e.q.AutomationContact(ctx, e.run.ContactID)
	if err != nil {
		return nil, err
	}
	e.c = &c
	return e.c, nil
}

// maxStepsPerTick bounds how far a run moves in one go (flows are acyclic,
// so this only guards against bad data).
const maxStepsPerTick = 60

func (e *executor) advance(ctx context.Context) error {
	for i := 0; i < maxStepsPerTick; i++ {
		n := e.g.Node(e.run.NodeID)
		if n == nil {
			return e.finish(ctx, "failed", "step missing")
		}
		next, path, stop, err := e.step(ctx, *n)
		if err != nil || stop {
			return err
		}
		if err := e.moveOn(ctx, *n, next, path); err != nil {
			return err
		}
		if e.run.Status != "active" {
			return nil
		}
	}
	return e.save(ctx, &e.now)
}

// step runs the current step. It returns the next node (or "" with the path
// to follow), or stop when the run waits or ended.
func (e *executor) step(ctx context.Context, n AutomationNode) (next, path string, stop bool, err error) {
	switch n.Type {
	case NodeTrigger:
		return e.g.Next(n.ID), "", false, nil
	case NodeDelay:
		if n.Data.Delay == nil {
			return e.g.Next(n.ID), "", false, nil
		}
		until := e.run.NodeEnteredAt.Add(n.Data.Delay.Duration())
		if e.st.Wait == "delay" && e.st.Until != nil {
			until = *e.st.Until
		}
		if e.now.Before(until) {
			if e.st.Wait != "delay" {
				e.stat(n.ID).Waiting++
			}
			return "", "", true, e.wait(ctx, "delay", until)
		}
		return e.g.Next(n.ID), "", false, nil
	case NodeEmail:
		return e.email(ctx, n)
	case NodeBranch:
		return e.branch(ctx, n)
	case NodeFilter:
		ok, err := e.eval(ctx, n.Data.Condition)
		if err != nil {
			return "", "", true, err
		}
		if !ok {
			e.stat(n.ID).Dropped++
			return "", "", true, e.finish(ctx, "exited", "filtered")
		}
		return e.g.Next(n.ID), "", false, nil
	case NodeSplit:
		key := pickVariant(n, e.run.ID)
		p := e.stat(n.ID)
		p.Paths = addKey(p.Paths, key, 1)
		return e.g.NextOnPath(n.ID, key), key, false, nil
	case NodeWebhook:
		if e.st.Wait == "webhook" && e.st.Done {
			return e.g.Next(n.ID), "", false, nil
		}
		return "", "", true, e.callWebhook(ctx, n)
	case NodeList:
		if err := e.listAction(ctx, n.Data.Action); err != nil {
			return "", "", true, err
		}
		return e.g.Next(n.ID), "", false, nil
	case NodeGoal:
		ok := n.Data.Condition == nil
		if !ok {
			if ok, err = e.eval(ctx, n.Data.Condition); err != nil {
				return "", "", true, err
			}
		}
		if ok {
			e.stat(n.ID).GoalReached++
			return "", "", true, e.finish(ctx, "goal", "")
		}
		e.stat(n.ID).Exited++
		return "", "", true, e.finish(ctx, "completed", "")
	case NodeExit:
		e.stat(n.ID).Exited++
		return "", "", true, e.finish(ctx, "completed", "")
	}
	return e.g.Next(n.ID), "", false, nil
}

// moveOn leaves a step for the next one; with no next step the flow ends here.
func (e *executor) moveOn(ctx context.Context, from AutomationNode, next, path string) error {
	e.stat(from.ID).Completed++
	if next == "" || e.g.Node(next) == nil {
		// The path ends without an exit step: the contact finished the flow.
		e.stat(from.ID).Exited++
		return e.finish(ctx, "completed", "")
	}
	_ = path
	e.run.NodeID = next
	e.run.NodeEnteredAt = e.now
	e.run.Status = "active"
	e.st = runState{Messages: e.st.Messages, Event: e.st.Event}
	e.stat(next).Entered++
	return nil
}

func (e *executor) wait(ctx context.Context, kind string, until time.Time) error {
	e.st.Wait = kind
	e.st.Until = &until
	e.run.Status = "waiting"
	return e.save(ctx, &until)
}

func (e *executor) save(ctx context.Context, next *time.Time) error {
	return e.q.UpdateAutomationRun(ctx, store.UpdateAutomationRunParams{
		ID: e.run.ID, Version: e.run.Version, NodeID: e.run.NodeID, Status: e.run.Status, State: e.st.json(),
		NodeEnteredAt: e.run.NodeEnteredAt, NextAt: next,
	})
}

func (e *executor) finish(ctx context.Context, status, reason string) error {
	e.run.Status = status
	now := e.now
	return e.q.UpdateAutomationRun(ctx, store.UpdateAutomationRunParams{
		ID: e.run.ID, Version: e.run.Version, NodeID: e.run.NodeID, Status: status, State: e.st.json(),
		NodeEnteredAt: e.run.NodeEnteredAt, FinishedAt: &now, ExitReason: reason,
	})
}

// email runs a "Send email" step: the send window, skip rules, then the mailer.
func (e *executor) email(ctx context.Context, n AutomationNode) (string, string, bool, error) {
	cfg := n.Data.Email
	next := e.g.Next(n.ID)
	if cfg == nil || cfg.EmailID == "" {
		e.stat(n.ID).Skipped++
		return next, "", false, nil
	}
	c, err := e.contact(ctx)
	if err != nil {
		return "", "", true, err
	}
	if c.Status != "active" || c.ErasedAt != nil {
		e.stat(n.ID).Dropped++
		return "", "", true, e.finish(ctx, "exited", "contact is "+c.Status)
	}
	suppressed, err := e.q.AutomationContactSuppressed(ctx, store.AutomationContactSuppressedParams{WorkspaceID: e.run.WorkspaceID, Email: c.Email})
	if err != nil {
		return "", "", true, err
	}
	if suppressed {
		e.stat(n.ID).Dropped++
		return "", "", true, e.finish(ctx, "exited", "suppressed")
	}
	if w := cfg.SendWindow; w != nil && w.Enabled {
		if at, inside := nextWindow(e.now, contactLoc(c.Timezone, e.s.workspaceTZ(ctx, e.q, e.run.WorkspaceID)), w.Start, w.End); !inside {
			if e.st.Wait != "window" {
				e.stat(n.ID).Waiting++
			}
			return "", "", true, e.wait(ctx, "window", at)
		}
	}
	if cfg.SkipRecentHours > 0 {
		got, err := e.q.AutomationContactEmailedSince(ctx, store.AutomationContactEmailedSinceParams{ContactID: &e.run.ContactID, Since: e.now.Add(-time.Duration(cfg.SkipRecentHours) * time.Hour)})
		if err != nil {
			return "", "", true, err
		}
		if got {
			e.stat(n.ID).Skipped++
			return next, "", false, nil
		}
	}
	if cfg.SkipListID != "" {
		if lid, err := uuid.Parse(cfg.SkipListID); err == nil {
			in, err := e.q.AutomationContactInList(ctx, store.AutomationContactInListParams{ListID: lid, ContactID: e.run.ContactID})
			if err != nil {
				return "", "", true, err
			}
			if in {
				e.stat(n.ID).Skipped++
				return next, "", false, nil
			}
		}
	}
	emailID, ok := stepID(cfg.EmailID)
	if !ok {
		e.stat(n.ID).Skipped++
		return next, "", false, nil
	}
	queued, err := e.s.mailer().QueueAutomationEmail(ctx, e.tx, OutgoingEmail{
		WorkspaceID: e.run.WorkspaceID, AutomationID: e.run.AutomationID, Version: int(e.run.Version), NodeID: n.ID,
		RunID: e.run.ID, ContactID: e.run.ContactID, To: c.Email, EmailID: emailID, EmailVersion: cfg.EmailVersion,
		Subject: cfg.Subject, Route: cfg.Route,
	})
	if err != nil {
		return "", "", true, err
	}
	p := e.stat(n.ID)
	p.Sent++
	if queued.ProviderID != uuid.Nil {
		p.Providers = addKey(p.Providers, queued.ProviderID.String(), 1)
	}
	if queued.Held {
		if err := e.s.noteHeld(ctx, e.q, e.run.AutomationID, e.now); err != nil {
			return "", "", true, err
		}
	}
	if e.st.Messages == nil {
		e.st.Messages = map[string]string{}
	}
	e.st.Messages[n.ID] = queued.MessageID.String()
	return next, "", false, nil
}

// branch evaluates a condition. Engagement conditions (opened / clicked)
// wait until the email is opened or clicked, or the window closes.
func (e *executor) branch(ctx context.Context, n AutomationNode) (string, string, bool, error) {
	c := n.Data.Condition
	path := PathNo
	if c != nil && (c.Kind == "opened" || c.Kind == "clicked") {
		met, msg, sentAt, err := e.engaged(ctx, c)
		if err != nil {
			return "", "", true, err
		}
		if met {
			path = PathYes
		} else {
			base := e.run.NodeEnteredAt
			if !sentAt.IsZero() {
				base = sentAt
			}
			deadline := base
			if c.Within != nil {
				deadline = base.Add(c.Within.Duration())
			}
			if e.now.Before(deadline) {
				if e.st.Wait != "branch" {
					e.stat(n.ID).Waiting++
				}
				e.st.Message, e.st.Want, e.st.Human = msg, c.Kind, c.HumanOnly
				return "", "", true, e.wait(ctx, "branch", deadline)
			}
		}
	} else {
		ok, err := e.eval(ctx, c)
		if err != nil {
			return "", "", true, err
		}
		if ok {
			path = PathYes
		}
	}
	p := e.stat(n.ID)
	if path == PathYes {
		p.BranchYes++
	} else {
		p.BranchNo++
	}
	p.Paths = addKey(p.Paths, path, 1)
	return e.g.NextOnPath(n.ID, path), path, false, nil
}

// engaged reports whether the contact opened or clicked the email the
// condition is about (any email of the flow when Step is empty).
func (e *executor) engaged(ctx context.Context, c *AutomationCondition) (met bool, msg string, sentAt time.Time, err error) {
	if c.Step != "" {
		if id, ok := e.st.Messages[c.Step]; ok {
			msg = id
		} else if m, err := e.q.AutomationRunMessage(ctx, store.AutomationRunMessageParams{AutomationID: &e.run.AutomationID, NodeID: &c.Step, ContactID: &e.run.ContactID}); err == nil {
			msg = m.ID.String()
			sentAt = m.QueuedAt
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return false, "", sentAt, err
		}
		if msg == "" {
			return false, "", sentAt, nil
		}
		mid, ok := stepID(msg)
		if !ok {
			return false, "", sentAt, nil
		}
		met, err = e.q.AutomationMessageEngaged(ctx, store.AutomationMessageEngagedParams{MessageID: &mid, Want: c.Kind, HumanOnly: c.HumanOnly, UrlContains: c.URLContains})
		return met, msg, sentAt, err
	}
	met, err = e.q.AutomationEngaged(ctx, store.AutomationEngagedParams{ContactID: &e.run.ContactID, AutomationID: &e.run.AutomationID, Since: e.run.EnteredAt, Want: c.Kind, HumanOnly: c.HumanOnly, UrlContains: c.URLContains})
	return met, "", sentAt, err
}

// eval checks a condition right now.
func (e *executor) eval(ctx context.Context, c *AutomationCondition) (bool, error) {
	if c == nil {
		return false, nil
	}
	switch c.Kind {
	case "opened", "clicked":
		met, _, _, err := e.engaged(ctx, c)
		return met, err
	case "tag":
		ct, err := e.contact(ctx)
		if err != nil {
			return false, err
		}
		return slices.Contains(ct.Tags, c.Tag), nil
	case "list":
		lid, ok := stepID(c.ListID)
		if !ok {
			return false, nil
		}
		return e.q.AutomationContactInList(ctx, store.AutomationContactInListParams{ListID: lid, ContactID: e.run.ContactID})
	case "event":
		return e.q.AutomationContactSentEvent(ctx, store.AutomationContactSentEventParams{WorkspaceID: e.run.WorkspaceID, ContactID: e.run.ContactID, Event: c.Event, Since: e.run.EnteredAt})
	case "field":
		ct, err := e.contact(ctx)
		if err != nil {
			return false, err
		}
		return fieldMatches(ct, c), nil
	}
	return false, nil
}

// fieldMatches compares a contact field.
func fieldMatches(c *store.AutomationContactRow, cond *AutomationCondition) bool {
	var v string
	switch cond.Field {
	case "email":
		v = c.Email
	case "first_name":
		v = c.FirstName
	case "last_name":
		v = c.LastName
	case "timezone":
		v = c.Timezone
	default:
		var attrs map[string]any
		_ = json.Unmarshal(c.Attributes, &attrs)
		if x, ok := attrs[cond.Field]; ok && x != nil {
			v = fmt.Sprint(x)
		}
	}
	switch cond.Op {
	case "is_set":
		return v != ""
	case "is_not_set":
		return v == ""
	case "not_equals":
		return !strings.EqualFold(v, cond.Value)
	case "contains":
		return strings.Contains(strings.ToLower(v), strings.ToLower(cond.Value))
	}
	return strings.EqualFold(v, cond.Value)
}

// pickVariant assigns a contact to a split variant: stable for the run,
// weighted by the split.
func pickVariant(n AutomationNode, run uuid.UUID) string {
	total := 0
	for _, v := range n.Data.Variants {
		total += v.Weight
	}
	if total <= 0 || len(n.Data.Variants) == 0 {
		return ""
	}
	h := fnv.New32a()
	_, _ = h.Write(run[:])
	_, _ = h.Write([]byte(n.ID))
	pick := int(h.Sum32() % uint32(total))
	for _, v := range n.Data.Variants {
		if pick < v.Weight {
			return v.Key
		}
		pick -= v.Weight
	}
	return n.Data.Variants[len(n.Data.Variants)-1].Key
}

func (e *executor) listAction(ctx context.Context, a *AutomationListAction) error {
	if a == nil {
		return nil
	}
	source := "automation:" + e.run.AutomationID.String()
	switch a.Kind {
	case "add_list", "remove_list":
		lid, ok := stepID(a.ListID)
		if !ok {
			return nil
		}
		if a.Kind == "add_list" {
			return e.q.AutomationAddListMembership(ctx, store.AutomationAddListMembershipParams{ListID: lid, ContactID: e.run.ContactID, Source: source})
		}
		return e.q.AutomationRemoveListMembership(ctx, store.AutomationRemoveListMembershipParams{ListID: lid, ContactID: e.run.ContactID})
	case "add_tag":
		return e.q.AutomationAddContactTag(ctx, store.AutomationAddContactTagParams{ID: e.run.ContactID, Tag: a.Tag})
	case "remove_tag":
		return e.q.AutomationRemoveContactTag(ctx, store.AutomationRemoveContactTagParams{ID: e.run.ContactID, Tag: a.Tag})
	case "unsubscribe_all":
		return e.q.AutomationUnsubscribeAll(ctx, e.run.ContactID)
	}
	return nil
}

// callWebhook hands the call to a job (network I/O never runs inside the
// transaction) and waits for it.
func (e *executor) callWebhook(ctx context.Context, n AutomationNode) error {
	e.st.Wait = "webhook"
	e.st.Done = false
	e.run.Status = "waiting"
	ins := e.s.inserter()
	if ins == nil {
		// No job runner in this process: the scan re-enqueues lost calls.
		return e.save(ctx, nil)
	}
	if _, err := ins.InsertTx(ctx, e.tx, WebhookArgs{RunID: e.run.ID, AutomationID: e.run.AutomationID, Version: int(e.run.Version), NodeID: n.ID}, webhookOpts()); err != nil {
		return err
	}
	return e.save(ctx, nil)
}

// noteHeld records that a flow's email is held for want of a healthy provider.
func (s *Service) noteHeld(ctx context.Context, q *store.Queries, automation uuid.UUID, now time.Time) error {
	n, err := q.CountAutomationHeldMessages(ctx, &automation)
	if err != nil {
		return err
	}
	detail, _ := json.Marshal(map[string]any{"queued": n})
	return q.OpenAutomationAlert(ctx, store.OpenAutomationAlertParams{ID: uuid.Must(uuid.NewV7()), AutomationID: automation, Kind: "sending_held", Detail: detail, Failures: 1, At: now})
}

func (s *Service) workspaceTZ(ctx context.Context, q *store.Queries, ws uuid.UUID) string {
	w, err := q.GetWorkspace(ctx, ws)
	if err != nil {
		return "UTC"
	}
	return w.Timezone
}

func contactLoc(contactTZ, workspaceTZ string) *time.Location {
	for _, tz := range []string{contactTZ, workspaceTZ} {
		if tz == "" {
			continue
		}
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return time.UTC
}

// nextWindow reports whether now is inside the local send window, and if not
// when it next opens. Windows may cross midnight (21:00–07:00).
func nextWindow(now time.Time, loc *time.Location, start, end string) (time.Time, bool) {
	sh, sm, ok1 := hhmm(start)
	eh, em, ok2 := hhmm(end)
	if !ok1 || !ok2 {
		return now, true
	}
	local := now.In(loc)
	minutes := local.Hour()*60 + local.Minute()
	s, e := sh*60+sm, eh*60+em
	inside := (s < e && minutes >= s && minutes < e) || (s > e && (minutes >= s || minutes < e))
	if inside || s == e {
		return now, true
	}
	open := time.Date(local.Year(), local.Month(), local.Day(), sh, sm, 0, 0, loc)
	if !open.After(local) {
		open = open.AddDate(0, 0, 1)
	}
	return open.UTC(), false
}

func hhmm(s string) (int, int, bool) {
	if !reHHMM.MatchString(s) {
		return 0, 0, false
	}
	return int(s[0]-'0')*10 + int(s[1]-'0'), int(s[3]-'0')*10 + int(s[4]-'0'), true
}

// Scan does the periodic work besides moving runs: wakes branch waits whose
// email was opened, moves contacts who reached a goal, fires date triggers,
// re-enqueues lost webhook calls, releases held email when the route
// recovers, retires empty versions and trims old rows.
func (s *Service) Scan(ctx context.Context) error {
	now := s.now()
	if _, err := s.Q.WakeAutomationBranchRuns(ctx, &now); err != nil {
		return fmt.Errorf("wake branches: %w", err)
	}
	if err := s.detectGoals(ctx, now); err != nil {
		return fmt.Errorf("goals: %w", err)
	}
	if err := s.fireDateTriggers(ctx, now); err != nil {
		return fmt.Errorf("date triggers: %w", err)
	}
	if err := s.requeueWebhooks(ctx, now); err != nil {
		return fmt.Errorf("webhooks: %w", err)
	}
	if err := s.releaseHeld(ctx); err != nil {
		return fmt.Errorf("held email: %w", err)
	}
	if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rows, err := q.ListRetiringAutomationVersions(ctx)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if err := q.RetireAutomationVersion(ctx, store.RetireAutomationVersionParams(r)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("retire versions: %w", err)
	}
	week := now.Add(-7 * 24 * time.Hour)
	if _, err := s.Q.PurgeAutomationEvents(ctx, &week); err != nil {
		return err
	}
	_, err := s.Q.PurgeAutomationWebhookCalls(ctx, now.Add(-90*24*time.Hour))
	return err
}

// detectGoals moves contacts who met a goal anywhere in the flow to that goal.
func (s *Service) detectGoals(ctx context.Context, now time.Time) error {
	versions, err := s.Q.AutomationActiveRunVersions(ctx)
	if err != nil {
		return err
	}
	for _, v := range versions {
		g := s.graphOf(v.Graph)
		for _, n := range g.Nodes {
			if n.Type != NodeGoal || n.Data.Condition == nil {
				continue
			}
			goal := n
			err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
				q := store.New(tx)
				var ids []uuid.UUID
				c := goal.Data.Condition
				switch c.Kind {
				case "opened", "clicked":
					rows, err := q.AutomationGoalRunsEngaged(ctx, store.AutomationGoalRunsEngagedParams{AutomationID: v.AutomationID, Version: v.Version, Want: c.Kind, HumanOnly: c.HumanOnly, UrlContains: c.URLContains})
					if err != nil {
						return err
					}
					for _, r := range rows {
						ids = append(ids, r.ID)
					}
				case "tag":
					rows, err := q.AutomationGoalRunsTagged(ctx, store.AutomationGoalRunsTaggedParams{AutomationID: v.AutomationID, Version: v.Version, Tag: c.Tag})
					if err != nil {
						return err
					}
					for _, r := range rows {
						ids = append(ids, r.ID)
					}
				case "list":
					lid, ok := stepID(c.ListID)
					if !ok {
						return nil
					}
					rows, err := q.AutomationGoalRunsInList(ctx, store.AutomationGoalRunsInListParams{AutomationID: v.AutomationID, Version: v.Version, ListID: lid})
					if err != nil {
						return err
					}
					for _, r := range rows {
						ids = append(ids, r.ID)
					}
				case "event":
					rows, err := q.AutomationGoalRunsEvent(ctx, store.AutomationGoalRunsEventParams{AutomationID: v.AutomationID, Version: v.Version, Event: c.Event})
					if err != nil {
						return err
					}
					for _, r := range rows {
						ids = append(ids, r.ID)
					}
				}
				if len(ids) == 0 {
					return nil
				}
				stats := statsBuf{}
				for _, id := range ids {
					fin := now
					if err := q.UpdateAutomationRun(ctx, store.UpdateAutomationRunParams{ID: id, Version: v.Version, NodeID: goal.ID, Status: "goal", State: json.RawMessage(`{}`), NodeEnteredAt: now, FinishedAt: &fin}); err != nil {
						return err
					}
					p := stats.at(v.AutomationID, v.Version, goal.ID, now)
					p.Entered++
					p.GoalReached++
				}
				return stats.flush(ctx, q)
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// fireDateTriggers queues date-field events once the trigger's local time
// has passed today.
func (s *Service) fireDateTriggers(ctx context.Context, now time.Time) error {
	rows, err := s.Q.AutomationDateTriggers(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		var t AutomationTrigger
		if json.Unmarshal(r.Trigger, &t) != nil || t.Field == "" {
			continue
		}
		loc := contactLoc("", r.Timezone)
		local := now.In(loc)
		at := t.At
		if at == "" {
			at = "09:00"
		}
		if h, m, ok := hhmm(at); ok && local.Hour()*60+local.Minute() < h*60+m {
			continue
		}
		today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
		ids, err := s.Q.AutomationDateFieldMatches(ctx, store.AutomationDateFieldMatchesParams{WorkspaceID: r.WorkspaceID, Field: t.Field, Tz: loc.String(), Yearly: t.Yearly, OffsetDays: int32(t.OffsetDays), Today: today})
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := s.Q.InsertAutomationEvent(ctx, store.InsertAutomationEventParams{WorkspaceID: r.WorkspaceID, Kind: TriggerDateField, ContactID: id, Ref: t.Field, Payload: json.RawMessage(`{}`), OccurredAt: now}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) requeueWebhooks(ctx context.Context, now time.Time) error {
	rows, err := s.Q.AutomationLostWebhookRuns(ctx, now.Add(-15*time.Minute))
	if err != nil {
		return err
	}
	ins := s.inserter()
	if ins == nil {
		return nil
	}
	for _, r := range rows {
		if _, err := ins.Insert(ctx, WebhookArgs{RunID: r.ID, AutomationID: r.AutomationID, Version: int(r.Version), NodeID: r.NodeID}, webhookOpts()); err != nil {
			return err
		}
		t := now.Add(time.Hour)
		// Mark it so the next scan doesn't enqueue it again right away.
		if err := s.Q.TouchAutomationRunNextAt(ctx, store.TouchAutomationRunNextAtParams{ID: r.ID, NextAt: &t}); err != nil {
			return err
		}
	}
	return nil
}

// releaseHeld sends held automation email once the route can send again.
func (s *Service) releaseHeld(ctx context.Context) error {
	rows, err := s.Q.ListAutomationHeldAlerts(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			pid, err := pickProvider(ctx, q, r.WorkspaceID, "", r.AutomationID)
			if err != nil || pid == uuid.Nil {
				return err
			}
			if _, err := q.ReleaseAutomationHeldMessages(ctx, store.ReleaseAutomationHeldMessagesParams{AutomationID: &r.AutomationID, ProviderID: &pid}); err != nil {
				return err
			}
			_, err = q.ResolveAutomationAlert(ctx, store.ResolveAutomationAlertParams{AutomationID: r.AutomationID, Kind: "sending_held", NodeID: ""})
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// MessageEvent is a delivery or engagement event of an automation email.
type MessageEvent struct {
	MessageID uuid.UUID
	// Type: delivered, bounced, complained, opened, clicked, unsubscribed.
	Type string
	// Machine marks opens by mail privacy proxies (Apple Mail).
	Machine bool
	At      time.Time
}

// RecordMessageEvent adds an automation email's delivery or engagement to
// its step's stats. The tracking and ingest domains call it for messages
// with stream "automation": once per message for delivered, bounced,
// complained and unsubscribed, and only for the first open and first click
// (stats count unique opens and clicks).
func (s *Service) RecordMessageEvent(ctx context.Context, q *store.Queries, ev MessageEvent) error {
	m, err := q.AutomationMessageForStats(ctx, ev.MessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if m.AutomationID == nil || m.AutomationNode == nil {
		return nil
	}
	version := m.DraftVersion
	if m.LiveVersion != nil {
		version = *m.LiveVersion
	}
	at := ev.At
	if at.IsZero() {
		at = s.now()
	}
	b := statsBuf{}
	p := b.at(*m.AutomationID, version, *m.AutomationNode, at)
	switch ev.Type {
	case "delivered":
		p.Delivered++
	case "bounced":
		p.Bounced++
	case "complained":
		p.Complained++
	case "opened":
		p.Opened++
		if !ev.Machine {
			p.OpenedHuman++
		}
	case "clicked":
		p.Clicked++
	case "unsubscribed":
		p.Unsubscribed++
	default:
		return fmt.Errorf("automations: unknown message event %q", ev.Type)
	}
	return b.flush(ctx, q)
}
