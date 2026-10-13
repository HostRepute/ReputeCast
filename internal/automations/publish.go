package automations

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Migration choices for contacts already in a flow when a version is published.
const (
	// MigrationFinish lets contacts finish on the version they started; new
	// entrants start on the new one. The old version retires when empty.
	MigrationFinish = "finish"
	// MigrationMove moves contacts to the new version at their current step.
	MigrationMove = "move"
)

// PublishRequest is what the approval gate sees.
type PublishRequest struct {
	WorkspaceID  uuid.UUID
	AutomationID uuid.UUID
	Name         string
	Version      int
	Migration    string
	By           uuid.UUID
	Changes      []AutomationDiffItem
}

// GateDecision tells the publish whether to go ahead.
type GateDecision struct {
	// Required stops the publish: an approval request was created instead.
	Required   bool
	ApprovalID uuid.UUID
	// Message explains who must approve ("Waiting for Marcus Chen or Linh Nguyen").
	Message string
}

// PublishGate is the approvals seam (add-on §3.8: "Optionally, publishing
// an automation can also require approval"). Check runs inside the publish
// transaction. When the workspace policy requires approval, the approvals
// domain creates an approval_requests row (subject_type 'automation') in tx
// and returns Required; once approved it calls Service.CompletePublish.
type PublishGate interface {
	Check(ctx context.Context, tx pgx.Tx, req PublishRequest) (GateDecision, error)
}

// NoApproval publishes right away.
type NoApproval struct{}

// Check implements PublishGate.
func (NoApproval) Check(context.Context, pgx.Tx, PublishRequest) (GateDecision, error) {
	return GateDecision{}, nil
}

// AutomationPublishCheck is one pre-publish check badge.
type AutomationPublishCheck struct {
	Status string `json:"status" enum:"ok,warn,bad"`
	Label  string `json:"label"`
}

// AutomationPublishPreview is the publish dialog: the diff, what happens to contacts
// already in the flow, and the pre-publish checks.
type AutomationPublishPreview struct {
	Name            string                   `json:"name"`
	FromVersion     *int                     `json:"fromVersion,omitempty"`
	ToVersion       int                      `json:"toVersion"`
	LivePublishedAt *time.Time               `json:"livePublishedAt,omitempty"`
	LivePublishedBy string                   `json:"livePublishedBy,omitempty"`
	Items           []AutomationDiffItem     `json:"items"`
	Changes         int                      `json:"changes"`
	InFlow          int64                    `json:"inFlow"`
	FinishEstimate  string                   `json:"finishEstimate" example:"about 9 days"`
	MoveImpact      string                   `json:"moveImpact"`
	Checks          []AutomationPublishCheck `json:"checks"`
	CanPublish      bool                     `json:"canPublish"`
	Blocker         string                   `json:"blocker,omitempty"`
}

type publishState struct {
	a        store.Automation
	refs     *Refs
	draft    AutomationGraph
	live     *AutomationGraph
	liveV    *store.GetAutomationVersionRow
	items    []AutomationDiffItem
	issues   []AutomationGraphIssue
	inflow   []store.CountAutomationInFlowByNodeRow
	failures map[string]WebhookFailure
}

func (s *Service) loadPublish(ctx context.Context, q *store.Queries, ws *store.Workspace, a store.Automation) (*publishState, error) {
	st := &publishState{a: a, failures: map[string]WebhookFailure{}}
	var err error
	if st.refs, err = s.loadRefs(ctx, q, ws.ID); err != nil {
		return nil, err
	}
	dv, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: a.DraftVersion})
	if err != nil {
		return nil, err
	}
	st.draft = s.graphOf(dv.Graph)
	if a.LiveVersion != nil {
		lv, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: *a.LiveVersion})
		if err != nil {
			return nil, err
		}
		g := s.graphOf(lv.Graph)
		st.live, st.liveV = &g, &lv
	}
	fails, err := q.AutomationStepFailures(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	for _, f := range fails {
		var d struct {
			URL    string `json:"url"`
			Status int    `json:"status"`
		}
		_ = json.Unmarshal(f.Detail, &d)
		st.failures[f.NodeID] = WebhookFailure{URL: d.URL, Status: d.Status, Since: f.FirstAt, Failures: f.Failures}
	}
	from := AutomationGraph{Nodes: []AutomationNode{}, Edges: []AutomationEdge{}}
	if st.live != nil {
		from = *st.live
	}
	st.items = Diff(from, st.draft, DiffContext{
		FromRefs: st.refs, ToVersion: int(a.DraftVersion), Loc: workspaceLoc(ws), Failures: st.failures,
		Check: func(emailID string, version int) string { return s.checkEmail(ctx, q, ws.ID, emailID, version) },
	})
	st.issues = Validate(st.draft, st.refs)
	if st.inflow, err = q.CountAutomationInFlowByNode(ctx, a.ID); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *Service) checkEmail(ctx context.Context, q *store.Queries, wsID uuid.UUID, emailID string, version int) string {
	id, err := uuid.Parse(emailID)
	if err != nil {
		return "no email picked"
	}
	e, err := q.GetAutomationEmailVersion(ctx, store.GetAutomationEmailVersionParams{WorkspaceID: wsID, ID: id, Version: int32(version)})
	if err != nil {
		return "the email was deleted"
	}
	return s.checks().Check(EmailContent{Subject: e.Subject, Preheader: e.Preheader, MJML: e.Mjml, HTML: e.Html, Text: e.PlainText})
}

// AutomationPublishPreview prepares the publish dialog.
func (s *Service) PublishPreview(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AutomationPublishPreview, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AutomationPublishPreview{}, api.MapError(err, "Automation")
	}
	st, err := s.loadPublish(ctx, s.Q, ws, automationOf(a))
	if err != nil {
		return AutomationPublishPreview{}, err
	}
	out := AutomationPublishPreview{Name: a.Name, ToVersion: int(a.DraftVersion), Items: st.items, Changes: Changes(st.items), Checks: []AutomationPublishCheck{}}
	if st.liveV != nil {
		v := int(*a.LiveVersion)
		out.FromVersion = &v
		out.LivePublishedAt, out.LivePublishedBy = st.liveV.PublishedAt, st.liveV.PublishedByName
	}
	for _, n := range st.inflow {
		out.InFlow += n.N
	}
	if st.live != nil {
		out.FinishEstimate = finishEstimate(*st.live, st.inflow)
		out.MoveImpact = moveImpact(*st.live, st.draft, st.items, st.inflow, st.refs)
	}
	route, err := s.route(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Checks = s.publishChecks(ctx, ws, st, route)
	out.CanPublish = true
	for _, c := range out.Checks {
		if c.Status == "bad" {
			out.CanPublish = false
			out.Blocker = c.Label
			break
		}
	}
	if out.CanPublish && st.live != nil && a.DraftVersion == *a.LiveVersion {
		out.CanPublish = false
		out.Blocker = fmt.Sprintf("Nothing to publish: there are no changes since v%d.", *a.LiveVersion)
	} else if out.CanPublish && st.live != nil && out.Changes == 0 {
		out.CanPublish = false
		out.Blocker = fmt.Sprintf("Nothing to publish: the draft only moves steps around v%d.", *a.LiveVersion)
	}
	return out, nil
}

func automationOf(a store.GetAutomationRow) store.Automation {
	return store.Automation{ID: a.ID, WorkspaceID: a.WorkspaceID, Name: a.Name, Description: a.Description, Status: a.Status,
		TemplateKey: a.TemplateKey, LiveVersion: a.LiveVersion, DraftVersion: a.DraftVersion, Trigger: a.Trigger,
		CreatedBy: a.CreatedBy, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt, PausedAt: a.PausedAt, PausedBy: a.PausedBy,
		PauseReason: a.PauseReason, DraftRevision: a.DraftRevision, UpdatedBy: a.UpdatedBy}
}

// publishChecks are the badges at the bottom of the publish dialog.
func (s *Service) publishChecks(ctx context.Context, ws *store.Workspace, st *publishState, route AutomationRoute) []AutomationPublishCheck {
	var out []AutomationPublishCheck
	// Route health.
	primaryOK := false
	var primary []string
	for _, p := range route.Primary {
		if healthyStatus(p.Status) {
			primaryOK = true
			primary = append(primary, p.Short)
		}
	}
	switch {
	case primaryOK:
		out = append(out, AutomationPublishCheck{Status: "ok", Label: "Automations route healthy · " + strings.Join(primary, ", ")})
	case route.Fallback != nil && healthyStatus(route.Fallback.Status):
		out = append(out, AutomationPublishCheck{Status: "warn", Label: "Automations route on its fallback · " + route.Fallback.Short})
	default:
		out = append(out, AutomationPublishCheck{Status: "warn", Label: "No healthy provider on the Automations route: emails wait until one is"})
	}
	// Email content.
	emails, failing := 0, 0
	for _, n := range st.draft.Nodes {
		if n.Type != NodeEmail {
			continue
		}
		emails++
		e := n.Data.Email
		if e == nil || s.checkEmail(ctx, s.Q, ws.ID, e.EmailID, e.EmailVersion) != "" {
			failing++
		}
	}
	switch {
	case emails == 0:
	case failing > 0:
		out = append(out, AutomationPublishCheck{Status: "bad", Label: fmt.Sprintf("%d of %d emails fail content checks", failing, emails)})
	case emails == 1:
		out = append(out, AutomationPublishCheck{Status: "ok", Label: "1 email passes content checks"})
	default:
		out = append(out, AutomationPublishCheck{Status: "ok", Label: fmt.Sprintf("%d emails pass content checks", emails)})
	}
	// References, forks, then anything else structural.
	var refIssue, pathIssue, other string
	for _, i := range Errors(st.issues) {
		switch i.Code {
		case "missing_ref":
			if refIssue == "" {
				refIssue = i.Message
			}
		case "missing_path", "duplicate_path", "unknown_path":
			if pathIssue == "" {
				pathIssue = i.Message
			}
		default:
			if other == "" {
				other = i.Message
			}
		}
	}
	if refIssue != "" {
		out = append(out, AutomationPublishCheck{Status: "bad", Label: refIssue})
	} else {
		out = append(out, AutomationPublishCheck{Status: "ok", Label: "No step references a deleted list or tag"})
	}
	forks, branches := 0, 0
	for _, n := range st.draft.Nodes {
		if n.Type == NodeBranch || n.Type == NodeSplit {
			forks++
			if n.Type == NodeBranch {
				branches++
			}
		}
	}
	switch {
	case pathIssue != "":
		out = append(out, AutomationPublishCheck{Status: "bad", Label: pathIssue})
	case branches == 1:
		out = append(out, AutomationPublishCheck{Status: "ok", Label: "Branch has both paths connected"})
	case branches > 1:
		out = append(out, AutomationPublishCheck{Status: "ok", Label: "Branches have both paths connected"})
	case forks > 0:
		out = append(out, AutomationPublishCheck{Status: "ok", Label: "A/B split has every path connected"})
	}
	if other != "" {
		out = append(out, AutomationPublishCheck{Status: "bad", Label: other})
	}
	return out
}

// remaining is the longest time a contact can still spend in the flow from
// a step: delays and branch windows along the slowest path.
func remaining(g AutomationGraph, id string, memo map[string]time.Duration) time.Duration {
	if d, ok := memo[id]; ok {
		return d
	}
	memo[id] = 0 // cycle guard
	n := g.Node(id)
	if n == nil {
		return 0
	}
	var here time.Duration
	switch n.Type {
	case NodeDelay:
		if n.Data.Delay != nil {
			here = n.Data.Delay.Duration()
		}
	case NodeBranch:
		if c := n.Data.Condition; c != nil && c.Within != nil {
			here = c.Within.Duration()
		}
	}
	var next time.Duration
	for _, e := range g.Out(id) {
		next = max(next, remaining(g, e.Target, memo))
	}
	memo[id] = here + next
	return memo[id]
}

// finishEstimate says how long until the last contact leaves the live version.
func finishEstimate(g AutomationGraph, inflow []store.CountAutomationInFlowByNodeRow) string {
	memo := map[string]time.Duration{}
	var longest time.Duration
	for _, n := range inflow {
		longest = max(longest, remaining(g, n.NodeID, memo))
	}
	switch {
	case longest <= 0:
		return "right away"
	case longest < time.Hour:
		return "within the hour"
	case longest < 24*time.Hour:
		h := int(math.Ceil(longest.Hours()))
		if h == 1 {
			return "about 1 hour"
		}
		return fmt.Sprintf("about %d hours", h)
	}
	d := int(math.Ceil(longest.Hours() / 24))
	if d == 1 {
		return "about 1 day"
	}
	return fmt.Sprintf("about %d days", d)
}

// durationAdjective reads "2-day", "10-minute".
func durationAdjective(d *AutomationDuration) string {
	if d == nil {
		return ""
	}
	return fmt.Sprintf("%d-%s", d.Amount, strings.TrimSuffix(d.Unit, "s"))
}

// moveImpact says what moving contacts to the new version would do.
func moveImpact(live, draft AutomationGraph, items []AutomationDiffItem, inflow []store.CountAutomationInFlowByNodeRow, refs *Refs) string {
	at := map[string]int64{}
	for _, n := range inflow {
		at[n.NodeID] += n.N
	}
	var parts []string
	for _, it := range items {
		switch it.Kind {
		case "changed":
			for _, id := range it.NodeIDs {
				n := draft.Node(id)
				if n == nil || n.Type != NodeDelay {
					continue
				}
				target := "the next step"
				if next := draft.Next(id); next != "" {
					target = inlineRef(draft, next, refs)
				}
				// Everyone who hasn't passed the delay yet: those in it and
				// those on the way to it.
				var waiting int64
				for node := range upstream(live, id) {
					waiting += at[node]
				}
				parts = append(parts, fmt.Sprintf("%s waiting before %s would get the %s delay", contacts(waiting), target, durationAdjective(n.Data.Delay)))
			}
		case "removed":
			var n int64
			kind := "step"
			for _, id := range it.NodeIDs {
				n += at[id]
				if node := live.Node(id); node != nil {
					kind = strings.ToLower(TypeName(node.Type))
					if node.Type == NodeEmail {
						kind = "email"
					}
				}
			}
			verb := "are"
			if n == 1 {
				verb = "is"
			}
			parts = append(parts, fmt.Sprintf("%s %s affected by the removed %s", contacts(n), verb, kind))
		}
	}
	if len(parts) == 0 {
		return "Contacts keep their place; changed steps apply from where each contact is now."
	}
	return capFirst(strings.Join(parts, "; ")) + "."
}

// upstream is id and every step that leads to it in g.
func upstream(g AutomationGraph, id string) map[string]bool {
	seen := map[string]bool{}
	stack := []string{id}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[cur] {
			continue
		}
		seen[cur] = true
		for _, e := range g.Edges {
			if e.Target == cur {
				stack = append(stack, e.Source)
			}
		}
	}
	return seen
}

func contacts(n int64) string {
	if n == 1 {
		return "1 contact"
	}
	return formatCount(n) + " contacts"
}

// AutomationPublishResult says whether the version went live or is waiting for approval.
type AutomationPublishResult struct {
	Status     string `json:"status" enum:"published,approval_required"`
	Version    int    `json:"version"`
	ApprovalID string `json:"approvalId,omitempty"`
	Message    string `json:"message,omitempty"`
}

// Publish makes the draft the live version, after the approval gate.
func (s *Service) Publish(ctx context.Context, ws *store.Workspace, by uuid.UUID, id uuid.UUID, version int, migration string) (AutomationPublishResult, error) {
	if migration == "" {
		migration = MigrationFinish
	}
	var out AutomationPublishResult
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if int(a.DraftVersion) != version {
			return api.Err(http.StatusConflict, api.CodeConflict, fmt.Sprintf("The draft is now v%d. Review the changes again before publishing.", a.DraftVersion))
		}
		if a.LiveVersion != nil && *a.LiveVersion == a.DraftVersion {
			return api.Conflict(fmt.Sprintf("v%d is already live. Edit the flow to start a new version.", a.DraftVersion))
		}
		st, err := s.loadPublish(ctx, q, ws, a)
		if err != nil {
			return err
		}
		if errs := Errors(st.issues); len(errs) > 0 {
			return api.Err(http.StatusUnprocessableEntity, api.CodeValidation, errs[0].Message+" Fix it on the canvas, then publish.").
				WithMeta("issues", errs)
		}
		dec, err := s.gate().Check(ctx, tx, PublishRequest{WorkspaceID: ws.ID, AutomationID: id, Name: a.Name, Version: version, Migration: migration, By: by, Changes: st.items})
		if err != nil {
			return err
		}
		wsID := ws.ID
		if dec.Required {
			out = AutomationPublishResult{Status: "approval_required", Version: version, ApprovalID: dec.ApprovalID.String(), Message: dec.Message}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.publish_requested", TargetType: "automation", TargetID: id.String(), TargetName: a.Name,
				Metadata: map[string]any{"version": version, "migration": migration, "approval_id": dec.ApprovalID.String()}})
		}
		if err := s.publishTx(ctx, tx, ws, a, st, by, migration); err != nil {
			return err
		}
		out = AutomationPublishResult{Status: "published", Version: version}
		return nil
	})
	if err == nil && out.Status == "published" {
		s.kick(ctx, nil)
	}
	return out, err
}

// CompletePublish publishes a version once its approval is granted. The
// approvals domain calls it with the approver's decision in place.
func (s *Service) CompletePublish(ctx context.Context, ws *store.Workspace, by uuid.UUID, id uuid.UUID, version int, migration string) error {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if int(a.DraftVersion) != version {
			return api.Conflict(fmt.Sprintf("The draft changed after approval (now v%d); request approval again.", a.DraftVersion))
		}
		st, err := s.loadPublish(ctx, q, ws, a)
		if err != nil {
			return err
		}
		if errs := Errors(st.issues); len(errs) > 0 {
			return api.Err(http.StatusUnprocessableEntity, api.CodeValidation, errs[0].Message)
		}
		return s.publishTx(ctx, tx, ws, a, st, by, migration)
	})
	if err == nil {
		s.kick(ctx, nil)
	}
	return err
}

func (s *Service) publishTx(ctx context.Context, tx pgx.Tx, ws *store.Workspace, a store.Automation, st *publishState, by uuid.UUID, migration string) error {
	q := store.New(tx)
	version := a.DraftVersion
	summary := versionSummary(st.items, a.LiveVersion == nil)
	if err := q.PublishAutomationVersion(ctx, store.PublishAutomationVersionParams{AutomationID: a.ID, Version: version, PublishedBy: &by, Summary: summary, Migration: migration}); err != nil {
		return err
	}
	moved := int64(0)
	if a.LiveVersion != nil && migration == MigrationMove {
		n, err := s.migrateRuns(ctx, q, a, version, st.draft)
		if err != nil {
			return err
		}
		moved = n
	}
	if err := q.SetAutomationLive(ctx, store.SetAutomationLiveParams{ID: a.ID, LiveVersion: &version, Trigger: triggerOf(st.draft), UpdatedBy: &by}); err != nil {
		return err
	}
	// Older versions without contacts retire now; the others when they empty.
	if err := retireEmpty(ctx, q, a.ID); err != nil {
		return err
	}
	// Alerts about steps the new version removed are over.
	for node := range st.failures {
		if st.draft.Node(node) == nil {
			if _, err := q.ResolveAutomationAlert(ctx, store.ResolveAutomationAlertParams{AutomationID: a.ID, Kind: "step_failing", NodeID: node}); err != nil {
				return err
			}
		}
	}
	var changes []string
	for _, it := range st.items {
		if it.Kind != "unchanged" {
			changes = append(changes, it.Kind+": "+it.Title)
		}
	}
	wsID := ws.ID
	before := map[string]any{"status": a.Status, "live_version": a.LiveVersion}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.publish", TargetType: "automation", TargetID: a.ID.String(), TargetName: a.Name,
		Before: before, After: map[string]any{"status": StatusLive, "live_version": version},
		Metadata: map[string]any{"migration": migration, "changes": changes, "contacts_moved": moved}})
}

func retireEmpty(ctx context.Context, q *store.Queries, automationID uuid.UUID) error {
	rows, err := q.ListRetiringAutomationVersions(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.AutomationID != automationID {
			continue
		}
		if err := q.RetireAutomationVersion(ctx, store.RetireAutomationVersionParams{AutomationID: r.AutomationID, Version: r.Version}); err != nil {
			return err
		}
	}
	return nil
}

// versionSummary is the version history's line: what this version changed.
func versionSummary(items []AutomationDiffItem, first bool) string {
	if first {
		return "first published version"
	}
	var parts []string
	for _, it := range items {
		if it.Kind == "unchanged" {
			continue
		}
		t := it.Title
		switch it.Kind {
		case "added":
			t = "added " + lowerFirst(t)
		case "removed":
			t = "removed " + lowerFirst(t)
		default:
			t = lowerFirst(t)
		}
		parts = append(parts, t)
	}
	s := strings.Join(parts, "; ")
	if r := []rune(s); len(r) > 160 {
		s = string(r[:157]) + "…"
	}
	return s
}

// migrateRuns moves contacts on older versions to the new one at their
// current step. Steps that no longer exist hand the contact to the next step
// that does; a changed delay restarts from when the contact reached it.
func (s *Service) migrateRuns(ctx context.Context, q *store.Queries, a store.Automation, version int32, g AutomationGraph) (int64, error) {
	runs, err := q.ListAutomationRunsForMigration(ctx, store.ListAutomationRunsForMigrationParams{AutomationID: a.ID, Version: version})
	if err != nil {
		return 0, err
	}
	graphs := map[int32]AutomationGraph{}
	now := s.now()
	var moved int64
	for _, r := range runs {
		old, ok := graphs[r.Version]
		if !ok {
			v, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: r.Version})
			if err != nil {
				return moved, err
			}
			old = s.graphOf(v.Graph)
			graphs[r.Version] = old
		}
		node := r.NodeID
		state := r.State
		nextAt := r.NextAt
		enteredAt := r.NodeEnteredAt
		status := r.Status
		if g.Node(node) == nil {
			node = forwardTo(old, g, node)
			state = json.RawMessage(`{}`)
			enteredAt = now
			status = "active"
			nextAt = &now
		} else if n := g.Node(node); n.Type == NodeDelay && n.Data.Delay != nil && r.Status == "waiting" {
			until := r.NodeEnteredAt.Add(n.Data.Delay.Duration())
			nextAt = &until
			state = json.RawMessage(fmt.Sprintf(`{"wait":"delay","until":%q}`, until.UTC().Format(time.RFC3339Nano)))
		}
		if node == "" {
			if err := q.UpdateAutomationRun(ctx, store.UpdateAutomationRunParams{ID: r.ID, Version: version, NodeID: r.NodeID, Status: "completed", State: state, NodeEnteredAt: enteredAt, FinishedAt: &now, ExitReason: "migrated"}); err != nil {
				return moved, err
			}
			continue
		}
		if err := q.UpdateAutomationRun(ctx, store.UpdateAutomationRunParams{ID: r.ID, Version: version, NodeID: node, Status: status, State: state, NodeEnteredAt: enteredAt, NextAt: nextAt}); err != nil {
			return moved, err
		}
		moved++
	}
	return moved, nil
}

// forwardTo walks from a removed step to the first following step that still
// exists in the new graph ("" when the path ends).
func forwardTo(old, g AutomationGraph, from string) string {
	seen := map[string]bool{}
	queue := []string{from}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		if id != from && g.Node(id) != nil {
			return id
		}
		for _, e := range old.Out(id) {
			queue = append(queue, e.Target)
		}
	}
	return ""
}

// Pause stops a live flow: nobody new enters, contacts stay where they are
// and nothing is sent until it resumes.
func (s *Service) Pause(ctx context.Context, ws *store.Workspace, by uuid.UUID, id uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) > 200 {
		return api.Invalid("body.reason", "Keep the reason under 200 characters.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if a.Status != StatusLive {
			return api.Conflict("Only a live automation can be paused.")
		}
		if err := q.PauseAutomation(ctx, store.PauseAutomationParams{ID: id, PausedBy: &by, PauseReason: reason}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.pause", TargetType: "automation", TargetID: id.String(), TargetName: a.Name,
			Before: map[string]any{"status": a.Status}, After: map[string]any{"status": StatusPaused}, Reason: reason})
	})
}

// Resume restarts a paused flow; delays that ended meanwhile run right away.
func (s *Service) Resume(ctx context.Context, ws *store.Workspace, by uuid.UUID, id uuid.UUID) error {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if a.Status != StatusPaused {
			return api.Conflict("Only a paused automation can be resumed.")
		}
		if err := q.ResumeAutomation(ctx, store.ResumeAutomationParams{ID: id, UpdatedBy: &by}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.resume", TargetType: "automation", TargetID: id.String(), TargetName: a.Name,
			Before: map[string]any{"status": a.Status}, After: map[string]any{"status": StatusLive}})
	})
	if err == nil {
		s.kick(ctx, nil)
	}
	return err
}
