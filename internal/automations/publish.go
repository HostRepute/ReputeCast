package automations

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

var (
	mCheckRouteHealthy   = msg("check.routeHealthy")
	mCheckRouteFallback  = msg("check.routeFallback")
	mCheckRouteDown      = msg("check.routeDown")
	mCheckEmailsFail     = msg("check.emailsFail")
	mCheckEmailsPass     = msg("check.emailsPass")
	mCheckNoDeletedRefs  = msg("check.noDeletedRefs")
	mCheckBranchPaths    = msg("check.branchPaths")
	mCheckBranchesPaths  = msg("check.branchesPaths")
	mCheckSplitPaths     = msg("check.splitPaths")
	mCheckNoEmailPicked  = msg("check.noEmailPicked")
	mCheckEmailDeleted   = msg("check.emailDeleted")
	mBlockerNoChanges    = msg("blocker.noChanges")
	mBlockerOnlyMoves    = msg("blocker.onlyMoves")
	mEstimateNow         = msg("estimate.now")
	mEstimateWithinHour  = msg("estimate.withinHour")
	mEstimateHours       = msg("estimate.hours")
	mEstimateDays        = msg("estimate.days")
	mMoveWaiting         = msg("move.waiting")
	mMoveRemoved         = msg("move.removed")
	mMoveKindEmail       = msg("move.kindEmail")
	mMoveKindStep        = msg("move.kindStep")
	mMoveNextStep        = msg("move.nextStep")
	mMoveImpact          = msg("move.impact")
	mMoveNone            = msg("move.none")
	mProblemDraftMoved   = msg("problem.draftMoved")
	mProblemAlreadyLive  = msg("problem.alreadyLive")
	mProblemFixOnCanvas  = msg("problem.fixOnCanvas")
	mProblemChangedAfter = msg("problem.changedAfterApproval")
	mProblemPauseReason  = msg("problem.pauseReason")
	mProblemPauseNotLive = msg("problem.pauseNotLive")
	mProblemResumeNot    = msg("problem.resumeNotPaused")
	mSummaryFirst        = msg("summary.first")
	mSummaryAdded        = msg("summary.added")
	mSummaryRemoved      = msg("summary.removed")
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
	// Missing are permissions the flow's channel steps need that the
	// publisher lacks (sms:send for a "Send SMS" step): publishing goes
	// for approval by someone who holds them.
	Missing []authz.Permission
}

// GateDecision tells the publish whether to go ahead.
type GateDecision struct {
	// Required stops the publish: an approval request was created instead.
	Required   bool
	ApprovalID uuid.UUID
	// Message explains who must approve ("Waiting for Marcus Chen or Linh Nguyen").
	Message string
}

// PublishGate is the approvals seam (add-on §3.8). Check runs inside the
// publish transaction. When approval is required, the approvals domain
// creates an approval_requests row in tx and returns Required; once approved
// it calls Service.CompletePublish.
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
	// Requires are permissions the flow's channel steps need that the
	// viewer lacks: publishing asks for approval instead ("This flow has an
	// SMS step. Publishing needs sms:send.").
	Requires []string `json:"requires"`
	// ChannelSteps counts the flow's steps by channel ("sms": 1).
	ChannelSteps map[string]int `json:"channelSteps"`
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

// loadPublish gathers what publishing the draft would do. The diff is in
// English with its texts (the approval request and the audit log keep the
// English; the dialog writes it in the reader's language); the problems are
// in l.
func (s *Service) loadPublish(ctx context.Context, q *store.Queries, ws *store.Workspace, a store.Automation, l i18n.Lang) (*publishState, error) {
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
		checkText: func(emailID string, version int) i18n.Text { return s.checkEmail(ctx, q, ws.ID, emailID, version) },
	})
	st.items = s.languageDiff(ctx, q, ws, from, st.draft, st.items)
	st.issues = validate(l, st.draft, st.refs)
	if st.inflow, err = q.CountAutomationInFlowByNode(ctx, a.ID); err != nil {
		return nil, err
	}
	return st, nil
}

// checkEmail runs the content checks on an email version: nothing when it
// passes, otherwise its first problem.
func (s *Service) checkEmail(ctx context.Context, q *store.Queries, wsID uuid.UUID, emailID string, version int) i18n.Text {
	id, err := uuid.Parse(emailID)
	if err != nil {
		return txt(mCheckNoEmailPicked)
	}
	e, err := q.GetAutomationEmailVersion(ctx, store.GetAutomationEmailVersionParams{WorkspaceID: wsID, ID: id, Version: int32(version)})
	if err != nil {
		return txt(mCheckEmailDeleted)
	}
	c := EmailContent{Subject: e.Subject, Preheader: e.Preheader, MJML: e.Mjml, HTML: e.Html, Text: e.PlainText}
	checks := s.checks()
	if _, basic := checks.(BasicContentChecks); basic {
		return basicCheck(c)
	}
	if p := checks.Check(c); p != "" {
		return name(p)
	}
	return i18n.Text{}
}

// PublishPreview prepares the publish dialog.
func (s *Service) PublishPreview(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AutomationPublishPreview, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AutomationPublishPreview{}, api.MapError(err, "Automation")
	}
	l := langOf(ctx)
	st, err := s.loadPublish(ctx, s.Q, ws, automationOf(a), l)
	if err != nil {
		return AutomationPublishPreview{}, err
	}
	out := AutomationPublishPreview{Name: a.Name, ToVersion: int(a.DraftVersion), Items: diffIn(l, st.items), Changes: Changes(st.items), Checks: []AutomationPublishCheck{},
		Requires: []string{}, ChannelSteps: map[string]int{}}
	for _, n := range st.draft.Nodes {
		if isChannelNode(n.Type) {
			out.ChannelSteps[n.Type]++
		}
	}
	for _, perm := range missingPermissions(ctx, st.draft) {
		out.Requires = append(out.Requires, string(perm))
	}
	if st.liveV != nil {
		v := int(*a.LiveVersion)
		out.FromVersion = &v
		out.LivePublishedAt, out.LivePublishedBy = st.liveV.PublishedAt, st.liveV.PublishedByName
	}
	for _, n := range st.inflow {
		out.InFlow += n.N
	}
	if st.live != nil {
		out.FinishEstimate = say(l, finishEstimateText(*st.live, st.inflow))
		out.MoveImpact = say(l, moveImpactText(*st.live, st.draft, st.items, st.inflow, st.refs))
	}
	route, err := s.route(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Checks = s.publishChecks(ctx, ws, st, route, l)
	a11y, err := s.accessibility(ctx, s.Q, ws, automationOf(a), st.draft)
	if err != nil {
		return out, err
	}
	out.Checks = append(out.Checks, a11y.Checks...)
	out.CanPublish = true
	for _, c := range out.Checks {
		if c.Status == "bad" {
			out.CanPublish = false
			out.Blocker = c.Label
			if a11y.Blocked && slices.ContainsFunc(a11y.Checks, func(x AutomationPublishCheck) bool { return x.Label == c.Label }) {
				out.Blocker = a11y.Blocker
			}
			break
		}
	}
	if out.CanPublish && st.live != nil && a.DraftVersion == *a.LiveVersion {
		out.CanPublish = false
		out.Blocker = say(l, txt(mBlockerNoChanges, "version", itoaPlain(int(*a.LiveVersion))))
	} else if out.CanPublish && st.live != nil && out.Changes == 0 {
		out.CanPublish = false
		out.Blocker = say(l, txt(mBlockerOnlyMoves, "version", itoaPlain(int(*a.LiveVersion))))
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
func (s *Service) publishChecks(ctx context.Context, ws *store.Workspace, st *publishState, route AutomationRoute, l i18n.Lang) []AutomationPublishCheck {
	var out []AutomationPublishCheck
	add := func(status string, t i18n.Text) {
		out = append(out, AutomationPublishCheck{Status: status, Label: say(l, t)})
	}
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
		add("ok", txt(mCheckRouteHealthy, "providers", strings.Join(primary, ", ")))
	case route.Fallback != nil && healthyStatus(route.Fallback.Status):
		add("warn", txt(mCheckRouteFallback, "provider", route.Fallback.Short))
	default:
		add("warn", txt(mCheckRouteDown))
	}
	emails, failing := 0, 0
	for _, n := range st.draft.Nodes {
		if n.Type != NodeEmail {
			continue
		}
		emails++
		e := n.Data.Email
		if e == nil || !emptyText(s.checkEmail(ctx, s.Q, ws.ID, e.EmailID, e.EmailVersion)) {
			failing++
		}
	}
	switch {
	case emails == 0:
	case failing > 0:
		add("bad", txt(mCheckEmailsFail, "failing", failing, "total", emails))
	default:
		add("ok", txt(mCheckEmailsPass, "count", emails))
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
		add("ok", txt(mCheckNoDeletedRefs))
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
		add("ok", txt(mCheckBranchPaths))
	case branches > 1:
		add("ok", txt(mCheckBranchesPaths))
	case forks > 0:
		add("ok", txt(mCheckSplitPaths))
	}
	if other != "" {
		out = append(out, AutomationPublishCheck{Status: "bad", Label: other})
	}
	// Phase 8: the channels' badges (an SMS step's quiet hours and senders).
	out = append(out, channelPublishChecks(ctx, ws.ID, st.draft)...)
	return append(out, s.languageChecks(ctx, ws, st.draft, l)...)
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
	return say(english, finishEstimateText(g, inflow))
}

func finishEstimateText(g AutomationGraph, inflow []store.CountAutomationInFlowByNodeRow) i18n.Text {
	memo := map[string]time.Duration{}
	var longest time.Duration
	for _, n := range inflow {
		longest = max(longest, remaining(g, n.NodeID, memo))
	}
	switch {
	case longest <= 0:
		return txt(mEstimateNow)
	case longest < time.Hour:
		return txt(mEstimateWithinHour)
	case longest < 24*time.Hour:
		return txt(mEstimateHours, "count", int(math.Ceil(longest.Hours())))
	}
	return txt(mEstimateDays, "count", int(math.Ceil(longest.Hours()/24)))
}

// moveImpact says what moving contacts to the new version would do.
func moveImpact(live, draft AutomationGraph, items []AutomationDiffItem, inflow []store.CountAutomationInFlowByNodeRow, refs *Refs) string {
	return say(english, moveImpactText(live, draft, items, inflow, refs))
}

func moveImpactText(live, draft AutomationGraph, items []AutomationDiffItem, inflow []store.CountAutomationInFlowByNodeRow, refs *Refs) i18n.Text {
	at := map[string]int64{}
	for _, n := range inflow {
		at[n.NodeID] += n.N
	}
	var parts []i18n.Text
	for _, it := range items {
		switch it.Kind {
		case "changed":
			for _, id := range it.NodeIDs {
				n := draft.Node(id)
				if n == nil || n.Type != NodeDelay {
					continue
				}
				target := txt(mMoveNextStep)
				if next := draft.Next(id); next != "" {
					target = inlineRef(draft, next, refs)
				}
				// Everyone who hasn't passed the delay yet: those in it and
				// those on the way to it.
				var waiting int64
				for node := range upstream(live, id) {
					waiting += at[node]
				}
				parts = append(parts, txt(mMoveWaiting, "count", waiting, "step", target, "delay", durationAdjective(n.Data.Delay)))
			}
		case "removed":
			var n int64
			kind := txt(mMoveKindStep)
			for _, id := range it.NodeIDs {
				n += at[id]
				if node := live.Node(id); node != nil {
					kind = lowercase(typeName(node.Type))
					if node.Type == NodeEmail {
						kind = txt(mMoveKindEmail)
					}
				}
			}
			parts = append(parts, txt(mMoveRemoved, "count", n, "kind", kind))
		}
	}
	if len(parts) == 0 {
		return txt(mMoveNone)
	}
	return txt(mMoveImpact, "clauses", upper(joined("; ", parts)))
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
	l := langOf(ctx)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if int(a.DraftVersion) != version {
			return api.Err(http.StatusConflict, api.CodeConflict, say(l, txt(mProblemDraftMoved, "version", itoaPlain(int(a.DraftVersion)))))
		}
		if a.LiveVersion != nil && *a.LiveVersion == a.DraftVersion {
			return api.Conflict(say(l, txt(mProblemAlreadyLive, "version", itoaPlain(int(a.DraftVersion)))))
		}
		st, err := s.loadPublish(ctx, q, ws, a, l)
		if err != nil {
			return err
		}
		if errs := Errors(st.issues); len(errs) > 0 {
			return api.Err(http.StatusUnprocessableEntity, api.CodeValidation, say(l, txt(mProblemFixOnCanvas, "problem", errs[0].Message))).
				WithMeta("issues", errs)
		}
		if v, err := s.accessibility(ctx, q, ws, a, st.draft); err != nil {
			return err
		} else if v.Blocked {
			return api.Err(http.StatusConflict, CodeAccessibilityBlocked, v.Blocker)
		}
		req := PublishRequest{WorkspaceID: ws.ID, AutomationID: id, Name: a.Name, Version: version, Migration: migration, By: by, Changes: st.items,
			Missing: missingPermissions(ctx, st.draft)}
		dec, err := s.gate().Check(ctx, tx, req)
		if err != nil {
			return err
		}
		wsID := ws.ID
		if !dec.Required && len(req.Missing) > 0 {
			// Nobody can approve for the publisher: they need the permission.
			return api.Forbidden(req.Missing[0], nil)
		}
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
			return api.Conflict(say(english, txt(mProblemChangedAfter, "version", itoaPlain(int(a.DraftVersion)))))
		}
		// In English: the approvals domain writes what went wrong into a
		// notification it stores.
		st, err := s.loadPublish(ctx, q, ws, a, english)
		if err != nil {
			return err
		}
		if errs := Errors(st.issues); len(errs) > 0 {
			return api.Err(http.StatusUnprocessableEntity, api.CodeValidation, errs[0].Message)
		}
		if v, err := s.accessibility(ctx, q, ws, a, st.draft); err != nil {
			return err
		} else if v.Blocked {
			return api.Err(http.StatusConflict, CodeAccessibilityBlocked, v.Blocker)
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
	if err := q.PublishAutomationVersion(ctx, store.PublishAutomationVersionParams{AutomationID: a.ID, Version: version, PublishedBy: &by, Summary: say(english, summary),
		SummaryText: keepText(summary), Migration: migration}); err != nil {
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
		if err := q.RetireAutomationVersion(ctx, store.RetireAutomationVersionParams(r)); err != nil {
			return err
		}
	}
	return nil
}

// versionSummary is the version history's line: what this version
// changed. sayVersionSummary writes it, cut to 160 characters.
func versionSummary(items []AutomationDiffItem, first bool) i18n.Text {
	if first {
		return txt(mSummaryFirst)
	}
	var parts []i18n.Text
	for _, it := range items {
		if it.Kind == "unchanged" {
			continue
		}
		t := lower(it.TitleText)
		switch it.Kind {
		case "added":
			t = txt(mSummaryAdded, "change", t)
		case "removed":
			t = txt(mSummaryRemoved, "change", t)
		}
		parts = append(parts, t)
	}
	return joined("; ", parts)
}

// sayVersionSummary writes a version's summary as the history shows it,
// cut to 160 characters.
func sayVersionSummary(l i18n.Lang, t i18n.Text) string {
	s := say(l, t)
	if r := []rune(s); len(r) > 160 {
		s = string(r[:157]) + "…"
	}
	return s
}

// keepText stores a text for later readers (nil when there's none).
func keepText(t i18n.Text) json.RawMessage {
	if emptyText(t) {
		return nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return nil
	}
	return b
}

// keptText reads a stored text (empty for rows written before it was kept).
func keptText(raw json.RawMessage) i18n.Text {
	var t i18n.Text
	if len(raw) == 0 || json.Unmarshal(raw, &t) != nil {
		return i18n.Text{}
	}
	return t
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
		return api.Invalid("body.reason", say(langOf(ctx), txt(mProblemPauseReason)))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if a.Status != StatusLive {
			return api.Conflict(say(langOf(ctx), txt(mProblemPauseNotLive)))
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
			return api.Conflict(say(langOf(ctx), txt(mProblemResumeNot)))
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
