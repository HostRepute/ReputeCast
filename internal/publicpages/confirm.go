package publicpages

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// Double opt-in links that link scanners can't trigger (review 2026-11
// DP-3). Opening the link in the email shows a page with one Confirm button;
// only that button (a POST) confirms. Corporate scanners (Safe Links and
// the like) open every link in an email, so a link that confirmed on GET
// confirmed addresses bots had submitted. A link works for 30 days after it
// was sent; an expired one offers a new link instead.

// confirmAskBody is the page a confirmation link opens: "ask" (the Confirm
// button), "expired" (Send a new link) or "sent".
type confirmAskBody struct {
	Mode        string
	Email       string
	Lists       string
	Publisher   string
	Action      string
	Unavailable bool
}

// confirm answers GET /confirm/{token}: the Confirm button, the confirmed
// page when nothing is left to confirm, or the expired page. It never
// changes anything (HEAD, what some scanners send, answers headOnly).
func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	s, c, pc, ok := h.confirmClaims(w, r)
	if !ok {
		return
	}
	pl := h.confirmLang(w, r, s, pc)
	// Nothing left to confirm: the confirmed page, as on a second click.
	pending := h.confirmLists(r, s, c, pc)
	if len(pending) == 0 {
		h.confirmedPage(w, r, s, pc, pl)
		return
	}
	if h.confirmExpired(r, s, c) {
		h.confirmPage(w, r, s, pl, "confirm.expired.title", confirmAskBody{Mode: "expired", Email: emailaddr.Mask(pc.Email), Action: r.URL.RequestURI()})
		return
	}
	names, err := h.listNames(r, s, pl, pending)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.confirmPage(w, r, s, pl, "confirm.ask.title", confirmAskBody{Mode: "ask", Email: pc.Email, Lists: names, Publisher: s.set.Branding.PublisherName,
		Action: r.URL.RequestURI()})
}

// confirmPost answers POST /confirm/{token}: the Confirm button, "Send a
// new link" (do=resend) and the recommendations' Subscribe buttons (rec).
func (h *Handler) confirmPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if !h.parseForm(w, r, 4<<10) {
		h.notFound(w, r, s)
		return
	}
	if r.PostForm.Has("rec") {
		h.confirmRecommend(w, r)
		return
	}
	s, c, pc, ok := h.confirmClaims(w, r)
	if !ok {
		return
	}
	pl := h.confirmLang(w, r, s, pc)
	if r.PostForm.Get("do") == "resend" {
		h.confirmResend(w, r, s, c, pc, pl)
		return
	}
	if len(h.confirmLists(r, s, c, pc)) > 0 && h.confirmExpired(r, s, c) {
		h.confirmPage(w, r, s, pl, "confirm.expired.title", confirmAskBody{Mode: "expired", Email: emailaddr.Mask(pc.Email), Action: r.URL.RequestURI()})
		return
	}
	h.confirmDo(w, r, s, c)
}

// confirmResend emails a new confirmation link to the address an expired
// link was for. The page says the same whatever happened, so it tells
// nobody whether the address is waiting or how recently it was emailed.
func (h *Handler) confirmResend(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims, pc audience.PublicContact, pl pageLang) {
	body := confirmAskBody{Mode: "sent", Email: emailaddr.Mask(pc.Email)}
	ip := ""
	if a := clientIP(r); a != nil {
		ip = forms.ClientKey(*a)
	}
	if h.Forms.AllowPerMinute("confirm-resend:"+ip, 5) && len(h.confirmLists(r, s, linktoken.Claims{ContactID: c.ContactID}, pc)) > 0 {
		if err := h.Forms.ResendConfirmation(r.Context(), s.ws, c.ContactID); err != nil && !errors.Is(err, audience.ErrOptInCooldown) {
			h.log().WarnContext(r.Context(), "send a new confirmation link", "err", err)
			body.Unavailable = true
		}
	}
	h.confirmPage(w, r, s, pl, "confirm.sent.title", body)
}

// confirmClaims checks a confirmation link and loads its contact; a broken
// link or a contact who is gone is the not-found page.
func (h *Handler) confirmClaims(w http.ResponseWriter, r *http.Request) (*site, linktoken.Claims, audience.PublicContact, bool) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return nil, linktoken.Claims{}, audience.PublicContact{}, false
	}
	c, ok := h.claims(s, chi.URLParam(r, "token"), linktoken.Confirm)
	if !ok {
		h.notFound(w, r, s)
		return nil, c, audience.PublicContact{}, false
	}
	pc, err := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID)
	if errors.Is(err, audience.ErrContactGone) {
		h.notFound(w, r, s)
		return nil, c, pc, false
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return nil, c, pc, false
	}
	return s, c, pc, true
}

// confirmExpired reports whether a link is past its 30 days. A link sent
// before links carried their day works while the contact has a signup or
// a confirmation email from the last 30 days.
func (h *Handler) confirmExpired(r *http.Request, s *site, c linktoken.Claims) bool {
	now := h.now()
	if c.Link > 0 {
		return linktoken.Expired(c, linktoken.ConfirmValid, now)
	}
	var live bool
	err := h.Pool.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM consent_events WHERE contact_id = $1 AND action = 'optin_sent' AND occurred_at > $2)
		OR EXISTS (SELECT 1 FROM form_submissions WHERE contact_id = $1 AND double_opt_in AND confirmed_at IS NULL AND created_at > $2)`,
		c.ContactID, now.Add(-linktoken.ConfirmValid)).Scan(&live)
	return err == nil && !live
}

// confirmLists are the lists a link would confirm now: the pending lists of
// the signup it was sent for, or every pending list.
func (h *Handler) confirmLists(r *http.Request, s *site, c linktoken.Claims, pc audience.PublicContact) []uuid.UUID {
	var only []uuid.UUID
	if c.MessageID != uuid.Nil {
		if err := h.Pool.QueryRow(r.Context(), `SELECT list_ids FROM form_submissions WHERE id = $1 AND contact_id = $2`, c.MessageID, c.ContactID).Scan(&only); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			only = nil
		}
	}
	var out []uuid.UUID
	for id, st := range pc.Lists {
		if st == "pending" && (len(only) == 0 || slices.Contains(only, id)) {
			out = append(out, id)
		}
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return out
}

// listNames names lists in the page's language, joined ("Weekly Digest
// and Product Updates").
func (h *Handler) listNames(r *http.Request, s *site, pl pageLang, ids []uuid.UUID) (string, error) {
	lists, err := h.Audience.PublicLists(r.Context(), h.Q, s.ws)
	if err != nil {
		return "", err
	}
	cp := h.copyOf(r.Context(), s)
	var names []string
	for _, l := range lists {
		if !slices.Contains(ids, l.ID) {
			continue
		}
		name := l.Name
		if cp != nil {
			name = cp.ListName(pl.Code, l)
		}
		names = append(names, name)
	}
	return strings.Join(names, ", "), nil
}

func (h *Handler) confirmPage(w http.ResponseWriter, r *http.Request, s *site, pl pageLang, titleKey string, body confirmAskBody) {
	v := h.view(s, pageTitle(pl, s, titleKey), body)
	h.setLang(v, r, s, pl, nil)
	h.render(w, r, s, "confirm-ask", http.StatusOK, v, csp{}, true)
}
