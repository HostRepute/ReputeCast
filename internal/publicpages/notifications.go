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
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/pagelabels"
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

// notificationReasons are the answers to "Why did you turn it off?"
// (labels category.reason.<value>).
var notificationReasons = []string{"too_often", "not_useful", "in_app", "other"}

func notificationReasonsIn(l pagelabels.Set) []reasonView {
	out := make([]reasonView, 0, len(notificationReasons))
	for _, v := range notificationReasons {
		out = append(out, reasonView{Value: v, Label: l.Text("category.reason."+v, nil)})
	}
	return out
}

// messageLang is the language of a page reached from a transactional
// message: the menu's pick, then the language the message was sent in, then
// the recipient contact's, then the fallback.
func (h *Handler) messageLang(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims) pageLang {
	code := ""
	if c.MessageID != uuid.Nil {
		if mv, err := h.Q.PublicMessageVersion(r.Context(), store.PublicMessageVersionParams{ID: c.MessageID, WorkspaceID: s.ws.ID}); err == nil {
			code = mv.Language
		}
	}
	if code == "" && c.ContactID != uuid.Nil {
		if pc, err := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID); err == nil {
			code = pc.Language
		}
	}
	return h.contactLang(w, r, s, code)
}

// productTitle is the browser tab's title of a product's page: "Email
// preferences · Acme Cloud".
func productTitle(l pagelabels.Set, heading, product string) string {
	return l.Text("page.title", i18n.Args{"heading": heading, "publisher": product})
}

// notificationsView is the preference center's section.
type notificationsView struct {
	Title string
	Items []listView
	Shown string
}

func (h *Handler) notificationsSection(ctx context.Context, s *site, email string, l pagelabels.Set) (*notificationsView, error) {
	if h.Notifications == nil || email == "" {
		return nil, nil
	}
	set, err := h.Notifications.For(ctx, s.ws, email)
	if err != nil || len(set.Categories) == 0 {
		return nil, err
	}
	v := &notificationsView{Title: l.Text("prefs.notifications", i18n.Args{"product": set.Product})}
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
func (h *Handler) notificationsPage(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims, m NotificationMessage, token string, saved bool, status int) {
	pl := h.messageLang(w, r, s, c)
	l := pl.Labels
	body := preferencesBody{Email: m.Email, Action: s.links.Preferences(token)}
	if saved {
		body.Saved = l.Text("prefs.savedNotifications", nil)
	}
	sec, err := h.notificationsSection(r.Context(), s, m.Email, l)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	body.Notifications = sec
	if sec == nil {
		body.Note = l.Text("prefs.accountOnly", i18n.Args{"product": m.Product})
	}
	v := h.view(s, productTitle(l, l.Text("prefs.heading", nil), m.Product), body)
	v.Brand.Name, v.Brand.Initial = m.Product, initialOf(m.Product)
	h.setLang(v, r, s, pl, nil)
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
func (h *Handler) categoryPage(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims, m NotificationMessage, token string, off, thanks bool) {
	pl := h.messageLang(w, r, s, c)
	l := pl.Labels
	name := m.CategoryName
	// In a sentence English writes the category in lower case ("is off
	// product updates emails"); other languages keep its name.
	inText := name
	if l.Code() == "en" {
		inText = strings.ToLower(name)
	}
	args := i18n.Args{"email": m.Email, "category": inText, "product": m.Product}
	b := categoryBody{Email: m.Email, Action: s.links.Unsubscribe(token), Category: name, Off: off,
		PrefsHref: s.links.Preferences(h.sign(s, linktoken.Preferences, c.ContactID, c.MessageID))}
	if thanks {
		b.Status = l.Text("category.thanks", nil)
	}
	if off {
		b.Heading = l.Text("category.off.heading", i18n.Args{"category": name})
		b.Text = l.Text("category.off.text", args)
		if !thanks {
			b.Reasons = notificationReasonsIn(l)
		}
	} else {
		b.Heading = l.Text("category.on.heading", i18n.Args{"category": name})
		b.Text = l.Text("category.on.text", args)
	}
	if set, err := h.Notifications.For(r.Context(), s.ws, m.Email); err == nil {
		var on []string
		for _, nc := range set.Categories {
			if nc.On && (m.CategoryID == nil || nc.ID != *m.CategoryID) {
				on = append(on, nc.Name)
			}
		}
		if len(on) > 0 {
			b.Others = l.Text("category.stayOn", i18n.Args{"names": l.Join(on), "count": len(on)})
		}
	}
	v := h.view(s, productTitle(l, name, m.Product), b)
	v.Brand.Name, v.Brand.Initial = m.Product, initialOf(m.Product)
	h.setLang(v, r, s, pl, nil)
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
		pl := h.messageLang(w, r, s, c)
		l := pl.Labels
		h.messageIn(w, r, s, pl, http.StatusOK, productTitle(l, l.Text("prefs.heading", nil), m.Product), messageBody{Heading: l.Text("category.locked.heading", nil),
			Text:    l.Text("category.locked.text", i18n.Args{"product": m.Product}),
			Actions: []action{{Label: l.Text("page.preferences", nil), Href: s.links.Preferences(h.sign(s, linktoken.Preferences, c.ContactID, c.MessageID))}}}, true)
		return true
	}
	err = db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
		return h.Notifications.TurnOff(r.Context(), tx, s.ws, m.Email, *m.CategoryID, "unsubscribe_link", c.MessageID)
	})
	if err != nil {
		h.serverError(w, r, s, err)
		return true
	}
	h.categoryPage(w, r, s, c, m, token, true, false)
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
			h.categoryPage(w, r, s, c, m, token, false, false)
		}
	case "reason":
		err = h.Notifications.Reason(r.Context(), s.ws, m.Email, cat, f.Get("reason"))
		if err == nil {
			h.categoryPage(w, r, s, c, m, token, true, true)
		}
	default:
		err = db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
			return h.Notifications.TurnOff(r.Context(), tx, s.ws, m.Email, cat, "unsubscribe_link", c.MessageID)
		})
		if err == nil {
			h.categoryPage(w, r, s, c, m, token, true, false)
		}
	}
	if err != nil {
		h.serverError(w, r, s, err)
	}
	return true
}
