package automations

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// A member, API key or agent limited to some lists or sender domains
// publishes and resumes only a flow that stays within them, as they send
// only a campaign that does (internal/access). A live flow sends with
// nobody behind it, so the check is when it goes live or goes on again:
//
//   - Lists: a flow reaches whoever its triggers let in. Only Joined list
//     on one of their lists keeps to those lists; a segment, a tag, a form,
//     a date or an event lets in contacts of any list, as a segment does in
//     a campaign's audience.
//   - Sender domains: a flow's emails go out from the sender the outbox
//     picks for them, the workspace's default (AutomationEmailSenders).
//     A flow has no sender of its own to pick, so saving a draft or an
//     email changes no From address and sends nothing.

var (
	mProblemTriggerScope = msg("problem.triggerScope")
	mProblemSenderScope  = msg("problem.senderScope")
)

// flowScopeProblem is scopeProblem for all that publishing or resuming a
// flow lets send: g, its version that goes live, and, when the flow is
// paused, the versions its waiting contacts are on, which go on again with
// it whoever's triggers let them in.
func (s *Service) flowScopeProblem(ctx context.Context, q *store.Queries, wsID uuid.UUID, p *authz.Principal, a store.Automation, version int32, g AutomationGraph,
	l i18n.Lang) (string, error) {
	problem, err := scopeProblem(ctx, q, wsID, p, g, l)
	if err != nil || problem != "" || p == nil || p.Scope.IsZero() || a.Status != StatusPaused {
		return problem, err
	}
	inflow, err := q.CountAutomationInFlowByNode(ctx, a.ID)
	if err != nil {
		return "", err
	}
	checked := map[int32]bool{version: true}
	for _, r := range inflow {
		if checked[r.Version] {
			continue
		}
		checked[r.Version] = true
		v, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: r.Version})
		if err != nil {
			return "", err
		}
		if problem, err := scopeProblem(ctx, q, wsID, p, s.graphOf(v.Graph), l); err != nil || problem != "" {
			return problem, err
		}
	}
	return "", nil
}

// scopeProblem says, in l, why p can't publish or resume a flow whose
// graph is g, or "" when they can.
func scopeProblem(ctx context.Context, q *store.Queries, wsID uuid.UUID, p *authz.Principal, g AutomationGraph, l i18n.Lang) (string, error) {
	if p == nil || p.Scope.IsZero() {
		return "", nil
	}
	lists, others, emails := []uuid.UUID{}, 0, []uuid.UUID{}
	for _, n := range g.Nodes {
		switch {
		case n.Type == NodeTrigger:
			for _, t := range n.Data.Trigger.Triggers() {
				id, err := uuid.Parse(t.ListID)
				if t.Kind != TriggerJoinedList || err != nil {
					others++
					continue
				}
				lists = append(lists, id)
			}
		case n.Type == NodeEmail && n.Data.Email != nil:
			if id, err := uuid.Parse(n.Data.Email.EmailID); err == nil {
				emails = append(emails, id)
			}
		}
	}
	if !access.AudienceInScope(p.Scope, lists, others) {
		return say(l, txt(mProblemTriggerScope)), nil
	}
	if len(p.Scope.SenderDomains) == 0 || len(emails) == 0 {
		return "", nil
	}
	senders, err := q.AutomationEmailSenders(ctx, store.AutomationEmailSendersParams{EmailIds: emails, WorkspaceID: wsID})
	if err != nil {
		return "", err
	}
	for _, sn := range senders {
		if d := access.SenderDomain(sn.SenderDomain, sn.SenderEmail); d != "" && !p.Scope.AllowsSenderDomain(d) {
			return say(l, txt(mProblemSenderScope, "domain", d)), nil
		}
	}
	return "", nil
}
