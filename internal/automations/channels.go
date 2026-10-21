package automations

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// Channel steps (design expansion phase 8): "Send SMS" now, "Send push" and
// "Send WhatsApp" next. Each channel's package sends its steps and answers
// its branch conditions through ChannelStep, registered with
// RegisterChannelStep; the engine never imports a channel.

// NodeSMS is the "Send SMS" step.
const NodeSMS = "sms"

// Channel conditions of branches, filters and goals.
const (
	// CondSMSConsent: the contact has SMS consent on the condition's list
	// (any list when it names none).
	CondSMSConsent = "sms_consent"
	// CondSMSClicked: the contact clicked a link of the step's text (any
	// text of the flow when it names no step), within the window.
	CondSMSClicked = "sms_clicked"
	// CondSMSReplied: the contact replied to the step's text.
	CondSMSReplied = "sms_replied"
)

// AutomationSMSStep is a "Send SMS" step.
type AutomationSMSStep struct {
	// Body is the text between the rules' brand prefix and opt-out text,
	// with personalization tokens and links.
	Body string `json:"body,omitempty" maxLength:"1000"`
	// ConsentListID is the list whose SMS consent the step needs: contacts
	// without it are skipped. The rule can't be removed.
	ConsentListID string `json:"consentListId,omitempty" maxLength:"40"`
	// Outside quiet hours: "wait" for the window (default) or "skip".
	Outside string `json:"outside,omitempty" enum:"wait,skip,"`
	// SkipRecentHours skips the text when the contact got a text in the
	// last N hours (0 = off).
	SkipRecentHours int `json:"skipRecentHours,omitempty" minimum:"0" maximum:"720"`
	// NoOptOut leaves the rules' opt-out text out of this step's text
	// (it's on by default in steps; the rules can require it).
	NoOptOut bool `json:"noOptOut,omitempty"`
}

// ChannelSend is a channel step reached by a run.
type ChannelSend struct {
	WorkspaceID  uuid.UUID
	AutomationID uuid.UUID
	Version      int
	RunID        uuid.UUID
	ContactID    uuid.UUID
	Node         AutomationNode
	Now          time.Time
	// Waited is true when the run already waited for this step's window.
	Waited bool
}

// ChannelSent is what a channel step did.
type ChannelSent struct {
	// MessageID is the queued message.
	MessageID uuid.UUID
	// Wait holds the run until then (outside quiet hours).
	Wait *time.Time
	// Skipped says why the contact got nothing: no_consent, no_phone,
	// suppressed, recent, quiet_hours, no_route…
	Skipped string
}

// ChannelCheck asks a channel condition about a run's contact.
type ChannelCheck struct {
	WorkspaceID  uuid.UUID
	AutomationID uuid.UUID
	RunID        uuid.UUID
	ContactID    uuid.UUID
	EnteredAt    time.Time
	Condition    AutomationCondition
}

// ChannelStep sends one channel's steps.
type ChannelStep interface {
	// Send queues the step's message for the run's contact, in the run's
	// transaction.
	Send(ctx context.Context, tx pgx.Tx, in ChannelSend) (ChannelSent, error)
	// Condition answers a channel condition; sentAt is when the step's
	// message went (the window of an engagement condition starts there).
	Condition(ctx context.Context, tx pgx.Tx, in ChannelCheck) (met bool, sentAt time.Time, err error)
	// Permission is what publishing a flow with this step needs, on top
	// of publishing automations.
	Permission() authz.Permission
	// Validate checks a step's settings for the canvas, beyond what the
	// engine checks itself.
	Validate(n AutomationNode) []ChannelIssue
}

// ChannelIssue is a problem with a channel step's settings.
type ChannelIssue struct {
	Code    string
	Message string
}

var (
	channelMu    sync.RWMutex
	channelSteps = map[string]ChannelStep{}
	// channelConds maps a condition kind to the step type answering it.
	channelConds = map[string]string{CondSMSConsent: NodeSMS, CondSMSClicked: NodeSMS, CondSMSReplied: NodeSMS}
)

// RegisterChannelStep plugs a channel's steps in (the app wires it).
func RegisterChannelStep(nodeType string, c ChannelStep) {
	channelMu.Lock()
	defer channelMu.Unlock()
	channelSteps[nodeType] = c
}

func channelStep(nodeType string) (ChannelStep, bool) {
	channelMu.RLock()
	defer channelMu.RUnlock()
	c, ok := channelSteps[nodeType]
	return c, ok
}

// isChannelNode reports whether a step is sent by a channel's package.
func isChannelNode(t string) bool { return t == NodeSMS }

// isChannelCondition reports whether a condition is answered by a channel.
func isChannelCondition(kind string) bool {
	_, ok := channelConds[kind]
	return ok
}

// channelEngagement reports whether a channel condition waits for
// something to happen (a click, a reply) rather than reading a state.
func channelEngagement(kind string) bool { return kind == CondSMSClicked || kind == CondSMSReplied }

// channelPermissions lists what publishing a graph needs because of its
// channel steps.
func channelPermissions(g AutomationGraph) []authz.Permission {
	out := []authz.Permission{}
	for _, n := range g.Nodes {
		if !isChannelNode(n.Type) {
			continue
		}
		if c, ok := channelStep(n.Type); ok {
			p := c.Permission()
			dup := false
			for _, x := range out {
				if x == p {
					dup = true
				}
			}
			if !dup {
				out = append(out, p)
			}
		}
	}
	return out
}

// missingPermissions lists the permissions a graph's channel steps need
// that the caller lacks (none for the system, which publishes approved
// versions).
func missingPermissions(ctx context.Context, g AutomationGraph) []authz.Permission {
	p := api.PrincipalFrom(ctx)
	if p == nil {
		return nil
	}
	out := []authz.Permission{}
	for _, perm := range channelPermissions(g) {
		if !p.Can(perm) {
			out = append(out, perm)
		}
	}
	return out
}

// checkChannelNode validates a channel step on the canvas.
func checkChannelNode(n AutomationNode, listExists func(string) bool) []ChannelIssue {
	out := []ChannelIssue{}
	bad := func(code, msg string) { out = append(out, ChannelIssue{Code: code, Message: msg}) }
	if n.Type == NodeSMS {
		st := n.Data.SMS
		switch {
		case st == nil || strings.TrimSpace(st.Body) == "":
			bad("missing_text", "Write the text.")
		case st.ConsentListID == "":
			bad("missing_ref", "Pick the list whose SMS consent the step needs.")
		case !listExists(st.ConsentListID):
			bad("missing_ref", "The step's consent list was deleted. Pick another list.")
		}
	}
	if c, ok := channelStep(n.Type); ok {
		out = append(out, c.Validate(n)...)
	}
	return out
}

// channelSend runs a channel step: the contact's state first, then the
// channel's package.
func (e *executor) channelSend(ctx context.Context, n AutomationNode) (string, string, bool, error) {
	next := e.g.Next(n.ID)
	c, ok := channelStep(n.Type)
	if !ok {
		e.stat(n.ID).Skipped++
		return next, "", false, nil
	}
	ct, err := e.contact(ctx)
	if err != nil {
		return "", "", true, err
	}
	if ct.Status != "active" || ct.ErasedAt != nil {
		e.stat(n.ID).Dropped++
		return "", "", true, e.finish(ctx, "exited", "contact is "+ct.Status)
	}
	res, err := c.Send(ctx, e.tx, ChannelSend{WorkspaceID: e.run.WorkspaceID, AutomationID: e.run.AutomationID, Version: int(e.run.Version), RunID: e.run.ID,
		ContactID: e.run.ContactID, Node: n, Now: e.now, Waited: e.st.Wait == "window"})
	if err != nil {
		return "", "", true, err
	}
	switch {
	case res.Wait != nil:
		if e.st.Wait != "window" {
			e.stat(n.ID).Waiting++
		}
		return "", "", true, e.wait(ctx, "window", *res.Wait)
	case res.Skipped != "":
		e.stat(n.ID).Skipped++
		return next, "", false, nil
	}
	e.stat(n.ID).Sent++
	if e.st.Messages == nil {
		e.st.Messages = map[string]string{}
	}
	e.st.Messages[n.ID] = res.MessageID.String()
	return next, "", false, nil
}

// channelCondition answers a channel condition.
func (e *executor) channelCondition(ctx context.Context, cond *AutomationCondition) (bool, time.Time, error) {
	c, ok := channelStep(channelConds[cond.Kind])
	if !ok {
		return false, time.Time{}, nil
	}
	return c.Condition(ctx, e.tx, ChannelCheck{WorkspaceID: e.run.WorkspaceID, AutomationID: e.run.AutomationID, RunID: e.run.ID, ContactID: e.run.ContactID,
		EnteredAt: e.run.EnteredAt, Condition: *cond})
}

// ChannelCoster is a channel step that can say what its messages cost:
// the automations list shows each flow's cost over its window.
type ChannelCoster interface {
	CostSince(ctx context.Context, workspaceID uuid.UUID, automationIDs []uuid.UUID, since time.Time) (map[uuid.UUID]float64, error)
}

// graphChannels lists the channels a graph sends on, email first.
func graphChannels(g AutomationGraph) []string {
	out := []string{}
	for _, c := range []struct{ node, channel string }{{NodeEmail, "email"}, {NodeSMS, "sms"}} {
		for _, n := range g.Nodes {
			if n.Type == c.node {
				out = append(out, c.channel)
				break
			}
		}
	}
	return out
}

// channelCosts fills in what each listed flow's texts cost since a time,
// when the SMS step reports costs.
func (s *Service) channelCosts(ctx context.Context, workspaceID uuid.UUID, items []AutomationListItem, since time.Time) error {
	ids := []uuid.UUID{}
	for _, it := range items {
		if slices.Contains(it.Channels, "sms") {
			if id, err := uuid.Parse(it.ID); err == nil {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	c, ok := channelStep(NodeSMS)
	if !ok {
		return nil
	}
	coster, ok := c.(ChannelCoster)
	if !ok {
		return nil
	}
	costs, err := coster.CostSince(ctx, workspaceID, ids, since)
	if err != nil {
		return err
	}
	for i := range items {
		id, _ := uuid.Parse(items[i].ID)
		if v, ok := costs[id]; ok {
			items[i].ChannelCost = &v
		}
	}
	return nil
}

// ChannelPublishChecker is a channel step that adds badges to the publish
// dialog (brief §6): for SMS, quiet hours and the senders' registration in
// the countries its steps text. Its badges warn; they don't block.
type ChannelPublishChecker interface {
	PublishChecks(ctx context.Context, workspaceID uuid.UUID, nodes []AutomationNode) ([]AutomationPublishCheck, error)
}

// channelPublishChecks asks each channel with steps in the flow for its
// publish badges, in the order its steps first appear.
func channelPublishChecks(ctx context.Context, workspaceID uuid.UUID, g AutomationGraph) []AutomationPublishCheck {
	byType := map[string][]AutomationNode{}
	var types []string
	for _, n := range g.Nodes {
		if !isChannelNode(n.Type) {
			continue
		}
		if _, ok := byType[n.Type]; !ok {
			types = append(types, n.Type)
		}
		byType[n.Type] = append(byType[n.Type], n)
	}
	var out []AutomationPublishCheck
	for _, t := range types {
		c, ok := channelStep(t)
		if !ok {
			continue
		}
		pc, ok := c.(ChannelPublishChecker)
		if !ok {
			continue
		}
		checks, err := pc.PublishChecks(ctx, workspaceID, byType[t])
		if err != nil {
			slog.WarnContext(ctx, "automations: channel publish checks", "type", t, "err", err)
			continue
		}
		out = append(out, checks...)
	}
	return out
}
