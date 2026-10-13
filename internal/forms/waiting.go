package forms

// Confirmations that couldn't go out. Double opt-in emails travel on the
// System stream; while nothing can send System mail (no provider routed, the
// provider refused the message) the signups wait, the confirmations job
// retries them for three days, and the builder lists them so someone can
// send a person their link by hand: the copy-link fallback.

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// waitingWindow is how far back the builder lists signups whose
// confirmation email never went out.
const waitingWindow = 30 * 24 * time.Hour

// waitingPageSize bounds the list.
const waitingPageSize = 50

// SignupFormWaitingItem is a signup whose confirmation email hasn't gone out.
type SignupFormWaitingItem struct {
	ID        string    `json:"id" doc:"The submission"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	Lists     []string  `json:"lists" doc:"The lists the signup waits to join"`
	CreatedAt time.Time `json:"createdAt"`
	Attempts  int       `json:"attempts" doc:"How many times sending the email was tried"`
}

// SignupFormWaiting lists the signups of a form whose confirmation email
// hasn't gone out, newest first.
type SignupFormWaiting struct {
	Items []SignupFormWaitingItem `json:"items"`
	Total int64                   `json:"total"`
	// SystemMail reports whether System mail can go out now; the
	// confirmations job then sends what's waiting (signups of the last 3 days).
	SystemMail bool `json:"systemMail"`
}

// Waiting lists a form's signups waiting for their confirmation email.
func (s *Service) Waiting(ctx context.Context, ws *store.Workspace, id string) (SignupFormWaiting, error) {
	f, err := s.form(ctx, s.Q, ws, id)
	if err != nil {
		return SignupFormWaiting{}, err
	}
	since := s.now().Add(-waitingWindow)
	out := SignupFormWaiting{Items: []SignupFormWaitingItem{}, SystemMail: s.SystemMailAvailable(ctx, ws)}
	if out.Total, err = s.Q.FormWaitingCount(ctx, store.FormWaitingCountParams{FormID: &f.ID, Since: since}); err != nil {
		return out, err
	}
	rows, err := s.Q.FormWaitingSubmissions(ctx, store.FormWaitingSubmissionsParams{FormID: &f.ID, Since: since, Lim: waitingPageSize})
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ContactID)
	}
	names, err := s.Audience.PublicContactNames(ctx, s.Q, ws, ids)
	if err != nil {
		return out, err
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	listName := map[uuid.UUID]string{}
	for _, l := range lists {
		listName[l.ID] = l.Name
	}
	for _, r := range rows {
		n, ok := names[r.ContactID]
		if !ok {
			continue // erased since
		}
		item := SignupFormWaitingItem{ID: r.ID.String(), Email: n.Email, FirstName: n.FirstName, Lists: []string{}, CreatedAt: r.CreatedAt, Attempts: int(r.SendAttempts)}
		for _, l := range r.ListIds {
			if name, ok := listName[l]; ok {
				item.Lists = append(item.Lists, name)
			}
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// SignupFormConfirmLink is a confirmation link to send by hand.
type SignupFormConfirmLink struct {
	URL   string `json:"url"`
	Email string `json:"email"`
}

// WaitingLink returns the confirmation link of a waiting signup, to send by
// hand. Handing out a link that subscribes someone is written to the audit
// log; the confirmation itself records who clicked, from where, as always.
func (s *Service) WaitingLink(ctx context.Context, ws *store.Workspace, formID, submissionID string) (SignupFormConfirmLink, error) {
	f, err := s.form(ctx, s.Q, ws, formID)
	if err != nil {
		return SignupFormConfirmLink{}, err
	}
	sid, err := uuid.Parse(submissionID)
	if err != nil {
		return SignupFormConfirmLink{}, api.NotFound("Signup")
	}
	sub, err := s.Q.FormSubmissionByID(ctx, store.FormSubmissionByIDParams{ID: sid, WorkspaceID: ws.ID, FormID: &f.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return SignupFormConfirmLink{}, api.NotFound("Signup")
	}
	if err != nil {
		return SignupFormConfirmLink{}, err
	}
	if !sub.DoubleOptIn || sub.ConfirmedAt != nil {
		return SignupFormConfirmLink{}, api.Conflict("This signup is confirmed already; there's no link to send.")
	}
	names, err := s.Audience.PublicContactNames(ctx, s.Q, ws, []uuid.UUID{sub.ContactID})
	if err != nil {
		return SignupFormConfirmLink{}, err
	}
	n, ok := names[sub.ContactID]
	if !ok {
		return SignupFormConfirmLink{}, api.NotFound("Signup")
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "form.confirm_link", TargetType: "contact", TargetID: sub.ContactID.String(), TargetName: n.Email,
		Metadata: map[string]any{"form": f.Name, "formId": f.ID.String(), "submission": sub.ID.String()}}); err != nil {
		return SignupFormConfirmLink{}, err
	}
	return SignupFormConfirmLink{URL: s.ConfirmURL(ws, sub.ContactID), Email: n.Email}, nil
}
