package automations

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Who is in a flow (review 2026-11 AT-2): the contacts at each step, the
// flows a contact is in, and taking someone out of a flow.

// AutomationStepContact is one contact at a step.
type AutomationStepContact struct {
	RunID     string `json:"runId"`
	ContactID string `json:"contactId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	// Since is when they reached the step; Until when they move on (a
	// delay or a window), empty when they wait for something else.
	Since time.Time  `json:"since"`
	Until *time.Time `json:"until,omitempty"`
	// Wait is what they wait for: delay, branch, window, webhook, cap; empty
	// while they're about to move on.
	Wait      string    `json:"wait" enum:"delay,branch,window,webhook,cap,"`
	EnteredAt time.Time `json:"enteredAt"`
}

// AutomationStepContacts is a page of the contacts at a step.
type AutomationStepContacts struct {
	Total int64                   `json:"total"`
	Items []AutomationStepContact `json:"items"`
	// Next is the cursor of the next page; empty on the last.
	Next string `json:"next,omitempty"`
}

type stepCursor struct {
	At time.Time `json:"a"`
	ID uuid.UUID `json:"i"`
}

// StepContacts lists the contacts at a step, the latest to arrive first. A
// caller limited to some lists sees, and counts, only the contacts on them,
// as in Audience.
func (s *Service) StepContacts(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, node, cursor string, limit int) (AutomationStepContacts, error) {
	out := AutomationStepContacts{Items: []AutomationStepContact{}}
	if _, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id}); err != nil {
		return out, api.MapError(err, "Automation")
	}
	scope := access.ScopeLists(p)
	page := store.AutomationRunsAtNodeParams{AutomationID: id, NodeID: node, Lim: int32(limit + 1), ScopeLists: scope}
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		var c stepCursor
		if err != nil || json.Unmarshal(raw, &c) != nil {
			return out, api.Invalid("cursor", say(langOf(ctx), txt(mProblemCursor)))
		}
		page.After, page.AfterAt, page.AfterID = true, c.At, c.ID
	}
	rows, err := s.Q.AutomationRunsAtNode(ctx, page)
	if err != nil {
		return out, err
	}
	if out.Total, err = s.Q.CountAutomationRunsAtNode(ctx, store.CountAutomationRunsAtNodeParams{AutomationID: id, NodeID: node, ScopeLists: scope}); err != nil {
		return out, err
	}
	for i, r := range rows {
		if i == limit {
			last := rows[limit-1]
			b, _ := json.Marshal(stepCursor{At: last.NodeEnteredAt, ID: last.ID})
			out.Next = base64.RawURLEncoding.EncodeToString(b)
			break
		}
		it := AutomationStepContact{RunID: r.ID.String(), ContactID: r.ContactID.String(), Email: r.Email,
			Name: strings.TrimSpace(r.FirstName + " " + r.LastName), Since: r.NodeEnteredAt, Wait: r.Wait, EnteredAt: r.EnteredAt}
		if r.Status == "waiting" && r.NextAt != nil && r.Wait != "branch" && r.Wait != "webhook" {
			it.Until = r.NextAt
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// AutomationContactRun is one flow a contact is in or went through.
type AutomationContactRun struct {
	RunID        string `json:"runId"`
	AutomationID string `json:"automationId"`
	Name         string `json:"name"`
	// Status: active (in the flow), completed, exited, goal or failed.
	Status string `json:"status" enum:"active,completed,exited,goal,failed"`
	// Step is where they are, or where they finished.
	Step       string     `json:"step"`
	NodeID     string     `json:"nodeId"`
	ExitReason string     `json:"exitReason,omitempty"`
	EnteredAt  time.Time  `json:"enteredAt"`
	StepSince  time.Time  `json:"stepSince"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// AutomationContactRuns lists a contact's flows.
type AutomationContactRuns struct {
	Items []AutomationContactRun `json:"items"`
}

// ContactRuns lists the flows a contact is in or went through, the ones in
// progress first. A contact off the caller's lists is not found, as their
// profile isn't.
func (s *Service) ContactRuns(ctx context.Context, ws *store.Workspace, p *authz.Principal, contact uuid.UUID) (AutomationContactRuns, error) {
	out := AutomationContactRuns{Items: []AutomationContactRun{}}
	if ok, err := access.ContactInScope(ctx, s.Q, p, contact); err != nil {
		return out, err
	} else if !ok {
		return out, api.NotFound("Contact")
	}
	rows, err := s.Q.ContactAutomationRuns(ctx, store.ContactAutomationRunsParams{ContactID: contact, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	refs, err := s.loadRefs(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	l := langOf(ctx)
	for _, r := range rows {
		status := r.Status
		if status == "waiting" {
			status = "active"
		}
		step := ""
		g := s.graphOf(r.Graph)
		if n := g.Node(r.NodeID); n != nil {
			step = say(l, stepName(g, *n, refs))
		}
		out.Items = append(out.Items, AutomationContactRun{RunID: r.ID.String(), AutomationID: r.AutomationID.String(), Name: r.Name, Status: status,
			Step: step, NodeID: r.NodeID, ExitReason: r.ExitReason, EnteredAt: r.EnteredAt, StepSince: r.NodeEnteredAt, FinishedAt: r.FinishedAt})
	}
	return out, nil
}

// RemoveRunInput says why a contact is taken out of a flow.
type RemoveRunInput struct {
	Reason string `json:"reason" minLength:"3" maxLength:"300"`
}

// RemoveRun takes a contact out of a flow now: they get none of its
// remaining steps. Audited with the reason.
func (s *Service) RemoveRun(ctx context.Context, ws *store.Workspace, id, run uuid.UUID, in RemoveRunInput) error {
	reason := strings.TrimSpace(in.Reason)
	if len([]rune(reason)) < 3 {
		return api.Invalid("body.reason", say(langOf(ctx), txt(mProblemRemoveReason)))
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.LockAutomationRun(ctx, store.LockAutomationRunParams{ID: run, WorkspaceID: ws.ID})
		if err != nil || r.AutomationID != id {
			return api.MapError(pgxNoRows(err), "Run")
		}
		if r.Status != "active" && r.Status != "waiting" {
			return api.Conflict(say(langOf(ctx), txt(mProblemRunFinished)))
		}
		if err := q.UpdateAutomationRun(ctx, store.UpdateAutomationRunParams{ID: r.ID, Version: r.Version, NodeID: r.NodeID, Status: "exited",
			State: json.RawMessage(`{}`), NodeEnteredAt: now, FinishedAt: &now, ExitReason: "removed"}); err != nil {
			return err
		}
		stats := statsBuf{}
		stats.at(r.AutomationID, r.Version, r.NodeID, now).Dropped++
		if err := stats.flush(ctx, q); err != nil {
			return err
		}
		name := strings.TrimSpace(r.FirstName + " " + r.LastName)
		if name == "" {
			name = r.Email
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "automation.contact_remove", TargetType: "contact", TargetID: r.ContactID.String(),
			TargetName: name, Reason: reason, Metadata: map[string]any{"automation": r.AutomationID.String(), "automationName": r.Name, "step": r.NodeID}})
	})
}

// EndRunsOnReply ends the contact's runs in a flow whose version stops on
// reply ("Stop when the contact replies to an email from this flow", M58):
// they get none of its remaining steps. The inbound job calls it, in its
// transaction, for a person's reply (not an auto-reply or spam) to an
// email of the flow. It returns how many runs ended.
func (s *Service) EndRunsOnReply(ctx context.Context, tx pgx.Tx, workspaceID, automationID, contactID uuid.UUID, at time.Time) (int, error) {
	q := store.New(tx)
	runs, err := q.AutomationContactRunsForReply(ctx, store.AutomationContactRunsForReplyParams{AutomationID: automationID, WorkspaceID: workspaceID, ContactID: contactID})
	if err != nil {
		return 0, err
	}
	stats := statsBuf{}
	ended := 0
	for _, r := range runs {
		if !s.graphOf(r.Graph).StopsOnReply() {
			continue
		}
		if err := q.UpdateAutomationRun(ctx, store.UpdateAutomationRunParams{ID: r.ID, Version: r.Version, NodeID: r.NodeID, Status: "exited",
			State: json.RawMessage(`{}`), NodeEnteredAt: at, FinishedAt: &at, ExitReason: ExitReplied}); err != nil {
			return ended, err
		}
		stats.at(automationID, r.Version, r.NodeID, at).Dropped++
		ended++
	}
	return ended, stats.flush(ctx, q)
}

// ExitReplied is the exit reason of a run that ended because the contact
// replied (EndRunsOnReply).
const ExitReplied = "replied"

// pgxNoRows turns a run that isn't this flow's into "not found".
func pgxNoRows(err error) error {
	if err == nil {
		return pgx.ErrNoRows
	}
	return err
}
