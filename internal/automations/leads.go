package automations

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// "Submitted a lead ad form" (M66, G-41): internal/leads queues the event
// when a Facebook or Instagram lead is recorded for a contact, or, with
// double opt-in, once they confirm. The ref is the lead form's id; the
// trigger listens to one form (LeadFormID) or to any.
const TriggerLeadAdSubmitted = "lead_ad_submitted"

var (
	mTriggerLeadAdSubmitted        = msg("trigger.leadAdSubmitted")
	mRefAnyLeadForm                = msg("ref.anyLeadForm")
	mRefALeadForm                  = msg("ref.aLeadForm")
	mProblemTriggerLeadFormDeleted = msg("problem.triggerLeadFormDeleted")
)

// EmitLeadAdSubmitted queues a "Submitted a lead ad form" event inside the
// transaction that recorded the lead (or its confirmation).
func EmitLeadAdSubmitted(ctx context.Context, q *store.Queries, workspaceID, contactID, leadFormID uuid.UUID) error {
	return q.InsertAutomationEvent(ctx, store.InsertAutomationEventParams{
		WorkspaceID: workspaceID, Kind: TriggerLeadAdSubmitted, ContactID: contactID, Ref: leadFormID.String(),
		Payload: json.RawMessage(`{}`), OccurredAt: nowUTC(ctx),
	})
}

// AutomationLeadFormOption is a lead form the trigger can listen to.
type AutomationLeadFormOption struct {
	ID     string `json:"id"`
	Name   string `json:"name" example:"Spring webinar signup"`
	Page   string `json:"page" example:"Acme Inc" doc:"The Facebook Page the form belongs to"`
	Paused bool   `json:"paused" doc:"Paused, or not mapped yet: it records no leads"`
}

// leadFormLabel names a lead form with its Page ("Spring webinar signup ·
// Acme Inc").
func leadFormLabel(name, page string) string {
	if page == "" {
		return name
	}
	return name + " · " + page
}

// loadLeadFormRefs adds the workspace's lead forms to refs.
func loadLeadFormRefs(ctx context.Context, q *store.Queries, wsID uuid.UUID, r *Refs) error {
	rows, err := q.LeadFormNames(ctx, wsID)
	if err != nil {
		return err
	}
	r.LeadForms = make(map[string]string, len(rows))
	for _, f := range rows {
		r.LeadForms[f.ID.String()] = leadFormLabel(f.Name, f.PageName)
	}
	return nil
}

// leadFormOptions are the lead forms the trigger offers.
func leadFormOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]AutomationLeadFormOption, error) {
	rows, err := q.LeadFormNames(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make([]AutomationLeadFormOption, 0, len(rows))
	for _, f := range rows {
		out = append(out, AutomationLeadFormOption{ID: f.ID.String(), Name: f.Name, Page: f.PageName, Paused: f.Status != "active" || f.MappedAt == nil})
	}
	return out, nil
}

// checkLeadTrigger checks the trigger's form, when it names one.
func checkLeadTrigger(t *AutomationTrigger, refs *Refs) []configProblem {
	if t.LeadFormID == "" || refs == nil || refs.LeadForms == nil {
		return nil
	}
	if _, ok := refs.LeadForms[t.LeadFormID]; !ok {
		return []configProblem{{"missing_ref", txt(mProblemTriggerLeadFormDeleted)}}
	}
	return nil
}

// leadTriggerRef reads "any lead form" or the form and its Page.
func leadTriggerRef(t *AutomationTrigger, refs *Refs) i18n.Text {
	if t.LeadFormID == "" {
		return txt(mRefAnyLeadForm)
	}
	if refs != nil {
		if n, ok := refs.LeadForms[t.LeadFormID]; ok {
			return nameOrNothing(n)
		}
	}
	return txt(mRefALeadForm)
}

// leadTriggerMatches: any form, or the trigger's.
func leadTriggerMatches(t AutomationTrigger, ev store.ClaimAutomationEventsRow) bool {
	return t.LeadFormID == "" || t.LeadFormID == ev.Ref
}
