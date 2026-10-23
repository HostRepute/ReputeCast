package automations

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// EmitFormSubmitted queues a "form submitted" trigger event. The forms
// domain calls it inside the transaction that records the submission (after
// double opt-in when the form asks for it). List joins and new tags need no
// call: database triggers queue them.
func EmitFormSubmitted(ctx context.Context, q *store.Queries, workspaceID, contactID, formID uuid.UUID) error {
	return q.InsertAutomationEvent(ctx, store.InsertAutomationEventParams{
		WorkspaceID: workspaceID, Kind: TriggerFormSubmitted, ContactID: contactID, Ref: formID.String(),
		Payload: json.RawMessage(`{}`), OccurredAt: nowUTC(ctx),
	})
}

// APIEvent is an event sent through the API for one contact.
type APIEvent struct {
	Event      string
	Email      string
	ContactID  string
	Properties map[string]any
}

// AutomationEventResult says which contact the event was recorded for.
type AutomationEventResult struct {
	Accepted  bool   `json:"accepted"`
	ContactID string `json:"contactId"`
	Flows     int    `json:"flows" doc:"Live automations triggered by this event"`
}

var (
	mProblemEventName = msg("problem.eventName")
	mProblemEventSize = msg("problem.eventSize")
	mProblemContactID = msg("problem.contactId")
	mProblemNoContact = msg("problem.noContact")
	mProblemEventWho  = msg("problem.eventWho")
)

// SendEvent records an API event ("workspace.created") for a contact. Flows
// triggered by it enroll the contact on the next engine tick (seconds), and
// goals and branches waiting for it see it.
func (s *Service) SendEvent(ctx context.Context, ws *store.Workspace, ev APIEvent) (AutomationEventResult, error) {
	var out AutomationEventResult
	l := langOf(ctx)
	name := strings.TrimSpace(ev.Event)
	if !reEvent.MatchString(name) {
		return out, api.Invalid("body.event", say(l, txt(mProblemEventName)))
	}
	props, _ := json.Marshal(ev.Properties)
	if ev.Properties == nil {
		props = []byte(`{}`)
	}
	if len(props) > 16<<10 {
		return out, api.Invalid("body.properties", say(l, txt(mProblemEventSize)))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var contactID uuid.UUID
		var status string
		switch {
		case ev.ContactID != "":
			id, err := uuid.Parse(ev.ContactID)
			if err != nil {
				return api.Invalid("body.contactId", say(l, txt(mProblemContactID)))
			}
			c, err := q.AutomationContactInWorkspace(ctx, store.AutomationContactInWorkspaceParams{WorkspaceID: ws.ID, ID: id})
			if errors.Is(err, pgx.ErrNoRows) {
				return api.NotFound("Contact")
			}
			if err != nil {
				return err
			}
			contactID, status = c.ID, c.Status
		case ev.Email != "":
			c, err := q.AutomationContactByEmail(ctx, store.AutomationContactByEmailParams{WorkspaceID: ws.ID, Email: strings.ToLower(strings.TrimSpace(ev.Email))})
			if errors.Is(err, pgx.ErrNoRows) {
				return api.Err(404, api.CodeNotFound, say(l, txt(mProblemNoContact)))
			}
			if err != nil {
				return err
			}
			contactID, status = c.ID, c.Status
		default:
			return api.Invalid("body.email", say(l, txt(mProblemEventWho)))
		}
		if err := q.InsertAutomationEvent(ctx, store.InsertAutomationEventParams{
			WorkspaceID: ws.ID, Kind: TriggerAPIEvent, ContactID: contactID, Ref: name, Payload: props, OccurredAt: s.now(),
		}); err != nil {
			return err
		}
		flows, err := q.LiveAutomationsForTrigger(ctx, store.LiveAutomationsForTriggerParams{WorkspaceID: ws.ID, Kind: TriggerAPIEvent})
		if err != nil {
			return err
		}
		for _, f := range flows {
			var t AutomationTrigger
			if json.Unmarshal(f.Trigger, &t) == nil && t.Event == name {
				out.Flows++
			}
		}
		out.Accepted = true
		out.ContactID = contactID.String()
		_ = status
		// Events are contact activity, like opens and clicks: the event row
		// is the record, not the audit log (an integration may send
		// thousands a day).
		return nil
	})
	if err == nil && out.Flows > 0 {
		s.kick(ctx, nil)
	}
	return out, err
}
