package forms

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// formSubmittedEvent is the data of a form.submitted webhook event: the
// form, the contact, the answers and the consent text shown. It fires when
// a signup is confirmed (at once for single opt-in), at the same moment as
// the "Form submitted" automation trigger.
type formSubmittedEvent struct {
	FormID      string            `json:"form_id"`
	FormName    string            `json:"form_name"`
	ContactID   string            `json:"contact_id"`
	Email       string            `json:"email"`
	Answers     map[string]string `json:"answers"`
	Lists       []string          `json:"lists"`
	ConsentText string            `json:"consent_text,omitempty"`
	Language    string            `json:"language,omitempty"`
	DoubleOptIn bool              `json:"double_opt_in"`
	SubmittedAt time.Time         `json:"submitted_at"`
	ConfirmedAt time.Time         `json:"confirmed_at"`
}

// emitFormSubmitted queues the form.submitted event for the endpoints that
// listen, inside the transaction that confirms the signup.
func emitFormSubmitted(ctx context.Context, tx store.DBTX, ws *store.Workspace, formID, contactID uuid.UUID, answers map[string]string, submitted, confirmed time.Time) error {
	q := store.New(tx)
	f, err := q.FormGet(ctx, store.FormGetParams{WorkspaceID: ws.ID, ID: formID})
	if err != nil {
		return err
	}
	c, err := q.GetContact(ctx, store.GetContactParams{WorkspaceID: ws.ID, ID: contactID})
	if err != nil {
		return err
	}
	if answers == nil {
		answers = contactAnswers(c)
	}
	for k, v := range answers {
		if v == "" {
			delete(answers, k)
		}
	}
	answers["email"] = c.Email
	ev := formSubmittedEvent{FormID: f.PublicID, FormName: f.Name, ContactID: c.ID.String(), Email: c.Email, Answers: answers, Lists: []string{},
		ConsentText: f.ConsentText, Language: c.Language, DoubleOptIn: f.DoubleOptIn, SubmittedAt: submitted.UTC(), ConfirmedAt: confirmed.UTC()}
	names, err := q.ListWorkspaceLists(ctx, ws.ID)
	if err == nil {
		for _, l := range names {
			for _, id := range f.ListIds {
				if l.ID == id {
					ev.Lists = append(ev.Lists, l.Name)
				}
			}
		}
	}
	_, err = webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "form.submitted", ev)
	return err
}

// contactAnswers are a confirmed contact's answers, as the form wrote them.
func contactAnswers(c store.Contact) map[string]string {
	out := map[string]string{}
	if c.FirstName != "" {
		out[FieldFirstName] = c.FirstName
	}
	if c.LastName != "" {
		out[FieldLastName] = c.LastName
	}
	if c.Company != "" {
		out[FieldCompany] = c.Company
	}
	var attrs map[string]any
	if json.Unmarshal(c.Attributes, &attrs) == nil {
		for k, v := range attrs {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}
