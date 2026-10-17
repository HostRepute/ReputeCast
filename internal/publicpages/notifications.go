package publicpages

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// Notifications are transactional mail's Optional categories ("Reminders",
// "Product updates"): the preference center's "Notifications from Acme
// Cloud" section, and the unsubscribe link of a category, which turns off
// just that category. Recipients of transactional mail don't have to be
// contacts, so these pages also work for an address alone.
type Notifications interface {
	// Message is the transactional message a link came with; ok is false
	// for other mail.
	Message(ctx context.Context, ws *store.Workspace, messageID uuid.UUID) (msg NotificationMessage, ok bool, err error)
	// For lists the Optional categories and whether email gets each.
	For(ctx context.Context, ws *store.Workspace, email string) (NotificationSet, error)
	// Save turns the categories the page showed on (those in on) or off.
	Save(ctx context.Context, tx pgx.Tx, ws *store.Workspace, email string, on, shown []uuid.UUID, messageID *uuid.UUID) error
	// TurnOff and TurnOn change one category; source is unsubscribe_link
	// or one_click.
	TurnOff(ctx context.Context, tx pgx.Tx, ws *store.Workspace, email string, category uuid.UUID, source string, messageID uuid.UUID) error
	TurnOn(ctx context.Context, tx pgx.Tx, ws *store.Workspace, email string, category uuid.UUID) error
	// Reason keeps the answer to "Why did you turn it off?".
	Reason(ctx context.Context, ws *store.Workspace, email string, category uuid.UUID, reason string) error
}

// NotificationMessage is a transactional message's recipient and category.
type NotificationMessage struct {
	Email        string
	CategoryID   *uuid.UUID
	CategoryName string
	Optional     bool
	Product      string
}

// NotificationSet is the "Notifications from …" section for an address.
type NotificationSet struct {
	Product    string
	Categories []NotificationCategory
}

// NotificationCategory is one Optional category.
type NotificationCategory struct {
	ID          uuid.UUID
	Name        string
	Description string
	On          bool
	OptedOutAt  *time.Time
}

// notificationReasons are the answers to "Why did you turn it off?".
var notificationReasons = []reasonView{
	{"too_often", "Too often"},
	{"not_useful", "Not useful to me"},
	{"in_app", "I read this in the app"},
	{"other", "Other"},
}

// notificationsView is the preference center's section.
type notificationsView struct {
	Title string
	Items []listView
	Shown string
}

func (h *Handler) notificationsSection(ctx context.Context, s *site, email string) (*notificationsView, error) {
	if h.Notifications == nil || email == "" {
		return nil, nil
	}
	set, err := h.Notifications.For(ctx, s.ws, email)
	if err != nil || len(set.Categories) == 0 {
		return nil, err
	}
	v := &notificationsView{Title: "Notifications from " + set.Product}
	shown := make([]string, 0, len(set.Categories))
	for _, c := range set.Categories {
		v.Items = append(v.Items, listView{ID: c.ID.String(), Name: c.Name, Text: c.Description, Checked: c.On})
		shown = append(shown, c.ID.String())
	}
	v.Shown = strings.Join(shown, ",")
	return v, nil
}

// notificationsForm reads the section's checkboxes.
func notificationsForm(r *http.Request) (on, shown []uuid.UUID) {
	for _, raw := range strings.Split(r.PostForm.Get("categories_shown"), ",") {
		if id, err := uuid.Parse(strings.TrimSpace(raw)); err == nil {
			shown = append(shown, id)
		}
	}
	for _, raw := range r.PostForm["category"] {
		if id, err := uuid.Parse(raw); err == nil {
			on = append(on, id)
		}
	}
	return on, shown
}

// transactionalMessage returns the transactional message of a link.
func (h *Handler) transactionalMessage(ctx context.Context, s *site, c linktoken.Claims) (NotificationMessage, bool, error) {
	if h.Notifications == nil || c.MessageID == uuid.Nil {
		return NotificationMessage{}, false, nil
	}
	return h.Notifications.Message(ctx, s.ws, c.MessageID)
}

// notificationsPage is the preference center for someone who got
// transactional mail but isn't a contact: just the notifications section.
func (h *Handler) notificationsPage(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims, m NotificationMessage, token, saved, alert string, status int) {
	body := preferencesBody{Email: m.Email, Action: s.links.Preferences(token), Saved: saved, Alert: alert}
	sec, err := h.notificationsSection(r.Context(), s, m.Email)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	body.Notifications = sec
	if sec == nil {
		body.Note = "Emails from " + m.Product + " to this address are account and billing emails, like password resets and invoices. They always arrive."
	}
	v := h.view(s, "Email preferences · "+m.Product, body)
	v.Brand.Name, v.Brand.Initial = m.Product, initialOf(m.Product)
	h.render(w, r, s, "preferences", status, v, csp{}, true)
}

func initialOf(name string) string {
	for _, r := range strings.TrimSpace(name) {
		return strings.ToUpper(string(r))
	}
	return "?"
}

type categoryBody struct {
	Heading, Email, Text, Status, Action, PrefsHref, Category, Others string
	Off                                                               bool
	Reasons                                                           []reasonView
}

// categoryPage answers a category's unsubscribe link: the category is off
// (or back on), and the page says what still arrives.
func (h *Handler) categoryPage(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims, m NotificationMessage, token string, off bool, status string) {
	name := m.CategoryName
	b := categoryBody{Email: m.Email, Action: s.links.Unsubscribe(token), Category: name, Off: off, Status: status,
		PrefsHref: s.links.Preferences(h.sign(s, linktoken.Preferences, c.ContactID, c.MessageID))}
	if off {
		b.Heading = "You won't get " + name + " anymore"
		b.Text = " is off " + strings.ToLower(name) + " emails from " + m.Product + "."
		if status == "" {
			b.Reasons = notificationReasons
		}
	} else {
		b.Heading = name + " is on again"
		b.Text = " gets " + strings.ToLower(name) + " emails from " + m.Product + " again."
	}
	if set, err := h.Notifications.For(r.Context(), s.ws, m.Email); err == nil {
		var on []string
		for _, nc := range set.Categories {
			if nc.On && (m.CategoryID == nil || nc.ID != *m.CategoryID) {
				on = append(on, nc.Name)
			}
		}
		if len(on) > 0 {
			b.Others = listNames(on) + " " + map[bool]string{true: "stays", false: "stay"}[len(on) == 1] + " on"
		}
	}
	v := h.view(s, name+" · "+m.Product, b)
	v.Brand.Name, v.Brand.Initial = m.Product, initialOf(m.Product)
	h.render(w, r, s, "category", http.StatusOK, v, csp{}, true)
}

// categoryUnsubscribe handles the unsubscribe link of a transactional
// message (GET turns its Optional category off; HEAD, what link scanners
// send, never does). It reports whether the request was one.
func (h *Handler) categoryUnsubscribe(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims) bool {
	m, ok, err := h.transactionalMessage(r.Context(), s, c)
	if err != nil {
		h.serverError(w, r, s, err)
		return true
	}
	if !ok {
		return false
	}
	token := chi.URLParam(r, "token")
	if m.CategoryID == nil || !m.Optional {
		h.message(w, r, s, http.StatusOK, "Email preferences · "+m.Product, messageBody{Heading: "These emails can't be turned off",
			Text:    "Account and billing emails from " + m.Product + ", like password resets and invoices, aren't marketing, and your account needs them.",
			Actions: []action{{Label: "Email preferences", Href: s.links.Preferences(h.sign(s, linktoken.Preferences, c.ContactID, c.MessageID))}}}, true)
		return true
	}
	err = db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
		return h.Notifications.TurnOff(r.Context(), tx, s.ws, m.Email, *m.CategoryID, "unsubscribe_link", c.MessageID)
	})
	if err != nil {
		h.serverError(w, r, s, err)
		return true
	}
	h.categoryPage(w, r, s, c, m, token, true, "")
	return true
}

// categoryPost handles a POST to a transactional message's unsubscribe
// link: RFC 8058 one-click, and the page's buttons. It reports whether the
// request was one.
func (h *Handler) categoryPost(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims) bool {
	m, ok, err := h.transactionalMessage(r.Context(), s, c)
	if err != nil {
		h.log().ErrorContext(r.Context(), "category unsubscribe", "err", err)
		http.Error(w, "Couldn't change it right now; try again in a minute.", http.StatusInternalServerError)
		return true
	}
	if !ok {
		return false
	}
	f := r.PostForm
	oneClick := f.Get("List-Unsubscribe") == "One-Click"
	if m.CategoryID == nil || !m.Optional {
		if oneClick {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Account emails can't be turned off.\n"))
			return true
		}
		h.notFound(w, r, s)
		return true
	}
	cat := *m.CategoryID
	token := chi.URLParam(r, "token")
	if oneClick {
		err := db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
			return h.Notifications.TurnOff(r.Context(), tx, s.ws, m.Email, cat, "one_click", c.MessageID)
		})
		if err != nil {
			h.log().ErrorContext(r.Context(), "one-click category unsubscribe", "err", err)
			http.Error(w, "Couldn't unsubscribe right now; try again in a minute.", http.StatusInternalServerError)
			return true
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Unsubscribed.\n"))
		return true
	}
	switch f.Get("action") {
	case "on":
		err = db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error { return h.Notifications.TurnOn(r.Context(), tx, s.ws, m.Email, cat) })
		if err == nil {
			h.categoryPage(w, r, s, c, m, token, false, "")
		}
	case "reason":
		err = h.Notifications.Reason(r.Context(), s.ws, m.Email, cat, f.Get("reason"))
		if err == nil {
			h.categoryPage(w, r, s, c, m, token, true, "Thanks for telling us.")
		}
	default:
		err = db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
			return h.Notifications.TurnOff(r.Context(), tx, s.ws, m.Email, cat, "unsubscribe_link", c.MessageID)
		})
		if err == nil {
			h.categoryPage(w, r, s, c, m, token, true, "")
		}
	}
	if err != nil {
		h.serverError(w, r, s, err)
	}
	return true
}
