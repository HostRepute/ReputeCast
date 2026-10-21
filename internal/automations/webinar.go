package automations

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// A webinar page's reminder automation (the page builder's "Create the
// reminder automation?", design expansion phase 7): a date-field trigger
// on the field the registration form writes (dmarc_live_at), a reminder
// the day before at the session's time with the joining link, and another
// an hour before. It's created as a draft, for the publisher to check and
// publish.

// EventReminders describes the session the reminders are for.
type EventReminders struct {
	// Name is the flow's name ("DMARC live reminders").
	Name  string
	Title string
	// Field is the contact date field registrations write.
	Field string
	// At is the session's start, "HH:MM" in the workspace's time zone.
	At string
	// Link is the joining link the reminders carry.
	Link string
	// When is the session's date and time as the emails write it.
	When string
}

// CreateEventReminders makes the draft flow and its two emails.
func (s *Service) CreateEventReminders(ctx context.Context, ws *store.Workspace, by *uuid.UUID, in EventReminders) (uuid.UUID, error) {
	if !reHHMM.MatchString(in.At) || strings.TrimSpace(in.Field) == "" {
		return uuid.Nil, api.Invalid("body", "The event needs a start time and a date field.")
	}
	id := uuid.Must(uuid.NewV7())
	dayBefore, hourBefore := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	link := strings.TrimSpace(in.Link)
	joinLine := "The joining link: " + link
	if link == "" {
		joinLine = "The joining link is in your registration email."
	}
	emails := []struct {
		id                         uuid.UUID
		name, subject, pre, header string
		body                       []string
	}{
		{dayBefore, in.Title + " · tomorrow", "Tomorrow: " + in.Title, "Your joining link, and when it starts.", "See you tomorrow",
			[]string{"Hi {{ first_name }}, " + in.Title + " starts " + in.When + ".", joinLine}},
		{hourBefore, in.Title + " · in an hour", "Starting in an hour: " + in.Title, "Join from the link below.", "Starting in an hour",
			[]string{in.Title + " starts in an hour.", joinLine}},
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		graph := AutomationGraph{
			Nodes: []AutomationNode{
				{ID: "n1", Type: NodeTrigger, Position: AutomationNodePosition{X: 370, Y: 24}, Data: AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerDateField,
					Field: in.Field, OffsetDays: -1, At: in.At, Reentry: "always"}}},
				{ID: "n2", Type: NodeEmail, Position: AutomationNodePosition{X: 370, Y: 144}, Data: AutomationNodeData{Email: &AutomationEmailStep{EmailID: dayBefore.String(),
					EmailVersion: 1, Subject: emails[0].subject}}},
				{ID: "n3", Type: NodeDelay, Position: AutomationNodePosition{X: 370, Y: 284}, Data: AutomationNodeData{Delay: &AutomationDuration{Amount: 23, Unit: "hours"}}},
				{ID: "n4", Type: NodeEmail, Position: AutomationNodePosition{X: 370, Y: 364}, Data: AutomationNodeData{Email: &AutomationEmailStep{EmailID: hourBefore.String(),
					EmailVersion: 1, Subject: emails[1].subject}}},
				{ID: "n5", Type: NodeExit, Position: AutomationNodePosition{X: 370, Y: 504}},
			},
			Edges: []AutomationEdge{{ID: "e-n1-n2", Source: "n1", Target: "n2"}, {ID: "e-n2-n3", Source: "n2", Target: "n3"}, {ID: "e-n3-n4", Source: "n3", Target: "n4"},
				{ID: "e-n4-n5", Source: "n4", Target: "n5"}},
		}
		if _, err := q.CreateAutomation(ctx, store.CreateAutomationParams{ID: id, WorkspaceID: ws.ID, Name: in.Name,
			Description: fmt.Sprintf("Reminders for %s: the day before and an hour before.", in.Title), Trigger: triggerOf(graph), CreatedBy: by}); err != nil {
			return err
		}
		for _, e := range emails {
			c := DemoEmail(e.subject, e.pre, e.header, "", e.body...)
			if err := q.CreateAutomationEmail(ctx, store.CreateAutomationEmailParams{ID: e.id, WorkspaceID: ws.ID, AutomationID: &id, Name: e.name, CreatedBy: by}); err != nil {
				return err
			}
			if err := q.InsertAutomationEmailVersion(ctx, store.InsertAutomationEmailVersionParams{EmailID: e.id, Version: 1, Subject: c.Subject, Preheader: c.Preheader,
				Mjml: c.MJML, Html: accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerAutomationEmail, e.id, 1, c.HTML), PlainText: c.Text, CreatedBy: by}); err != nil {
				return err
			}
		}
		if err := q.InsertAutomationVersion(ctx, store.InsertAutomationVersionParams{AutomationID: id, Version: 1, Graph: graph.MustJSON(), CreatedBy: by,
			Summary: "created from a webinar page"}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.create", TargetType: "automation", TargetID: id.String(), TargetName: in.Name,
			After: map[string]any{"name": in.Name, "trigger": TriggerDateField, "field": in.Field, "steps": len(graph.Nodes)}, Metadata: map[string]any{"source": "page"}})
	})
	return id, err
}
