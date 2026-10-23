package publicpages

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/referrals"
)

// The invite page of a referral link (/join/{code}, brief §5.10): the
// hosted subscribe page for the program's list, with "Sam invited you" (or
// "A reader invited you") and who gets credit. A paused or unknown link, or
// a program that's off, shows the normal subscribe page: no invite line,
// and the signup doesn't count.

type joinBody struct {
	subscribeBody
	ListName, ListText string
	Credit             string
}

// joinForm is the hosted page's form for a program's list only.
func (h *Handler) joinForm(ctx context.Context, s *site, iv *referrals.InviteView) (forms.Config, bool, error) {
	c, page, err := h.resolveForm(ctx, s, "", false)
	if err != nil {
		return c, page, err
	}
	c.Lists, c.DefaultLists, c.ListChoice = []uuid.UUID{iv.ListID}, []uuid.UUID{iv.ListID}, false
	return c, page, nil
}

func (h *Handler) joinBody(w http.ResponseWriter, r *http.Request, s *site, iv *referrals.InviteView, code string, values *forms.Submission, out forms.Outcome) (joinBody, formLang, error) {
	c, page, err := h.joinForm(r.Context(), s, iv)
	if err != nil {
		return joinBody{}, formLang{}, err
	}
	fl, err := h.subscribeLang(w, r, s, c, "")
	if err != nil {
		return joinBody{}, formLang{}, err
	}
	b, err := h.joinBodyWith(r.Context(), s, iv, code, c, page, fl, values, out)
	return b, fl, err
}

// joinBodyWith is the invite page's body in a language already picked
// (joinBody, and the app's preview).
func (h *Handler) joinBodyWith(ctx context.Context, s *site, iv *referrals.InviteView, code string, c forms.Config, page bool, fl formLang, values *forms.Submission, out forms.Outcome) (joinBody, error) {
	fl.form.Lists, fl.form.DefaultLists, fl.form.ListChoice = c.Lists, c.DefaultLists, false
	sb, err := h.subscribeBody(ctx, s, fl, page, values, out)
	if err != nil {
		return joinBody{}, err
	}
	b := joinBody{subscribeBody: sb}
	name := iv.Name
	if name == "" {
		name = s.gt("join.anon")
	}
	b.Heading = strings.ReplaceAll(iv.Headline, "{first_name}", name)
	if strings.TrimSpace(iv.Text) != "" {
		b.Description = iv.Text
	}
	b.Action = s.links.Join(code)
	b.Lists = nil
	if iv.Name != "" {
		b.Credit = s.gt("join.creditName", "name", iv.Name)
	} else {
		b.Credit = s.gt("join.creditAnon")
	}
	if lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws); err == nil {
		for _, l := range lists {
			if l.ID == iv.ListID {
				b.ListName = l.Name
				b.ListText = strings.TrimSpace(strings.TrimSpace(l.Description) + " " + s.gt("join.readers", "count", l.Subscribed))
			}
		}
	}
	return b, nil
}

func (h *Handler) joinPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	rs := h.referrals()
	code := chi.URLParam(r, "code")
	var iv *referrals.InviteView
	if rs != nil {
		var err error
		if iv, err = rs.Invite(r.Context(), s.ws, code); err != nil {
			h.serverError(w, r, s, err)
			return
		}
	}
	if iv == nil {
		// The normal subscribe page: no invite line, nothing counts.
		h.subscribePage(w, r)
		return
	}
	if err := rs.RecordVisit(r.Context(), s.ws, iv.LinkID, iv.ProgramID, r.URL.Query().Get("via"), r.Method, r.UserAgent(), clientIP(r)); err != nil {
		h.log().WarnContext(r.Context(), "referral visit", "err", err)
	}
	b, fl, err := h.joinBody(w, r, s, iv, code, nil, forms.Outcome{})
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	b.Page, b.Ref = visitedURL(r, s.links.Join(code)), clipURL(r.Referer())
	h.renderJoin(w, r, s, fl, b, http.StatusOK)
}

func (h *Handler) renderJoin(w http.ResponseWriter, r *http.Request, s *site, fl formLang, b joinBody, status int) {
	v := h.view(s, b.Heading, b)
	h.setLang(v, r, s, fl.pageLang, fl.live)
	v.Description = b.Description
	// Personal to the referrer: not indexed, and the link is the canonical.
	v.Robots = "noindex, nofollow"
	h.render(w, r, s, "grow-join", status, v, csp{}, status != http.StatusOK)
}

func (h *Handler) joinPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	rs := h.referrals()
	code := chi.URLParam(r, "code")
	var iv *referrals.InviteView
	if rs != nil {
		var err error
		if iv, err = rs.Invite(r.Context(), s.ws, code); err != nil {
			h.serverError(w, r, s, err)
			return
		}
	}
	if iv == nil {
		h.subscribePost(w, r)
		return
	}
	if !h.parseForm(w, r, 64<<10) {
		http.Error(w, h.visitorLang(nil, r, s, nil).Labels.Text("page.unreadable", nil), http.StatusBadRequest)
		return
	}
	c, page, err := h.joinForm(r.Context(), s, iv)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	sub := submissionFrom(r)
	if !strings.Contains(sub.Page, "/join/") {
		sub.Page = s.links.Join(code)
	}
	sub.Lists = nil
	out := h.Forms.Submit(r.Context(), s.ws, s.set, c, sub)
	if out.Status == forms.OutcomeInvalid || out.Status == forms.OutcomeError {
		status := http.StatusUnprocessableEntity
		if out.Status == forms.OutcomeError {
			status = http.StatusServiceUnavailable
		}
		b, fl, err := h.joinBody(w, r, s, iv, code, &sub, out)
		if err != nil {
			h.serverError(w, r, s, err)
			return
		}
		b.Page, b.Ref = sub.Page, sub.Referrer
		h.renderJoin(w, r, s, fl, b, status)
		return
	}
	h.subscribeResult(w, r, s, c, page, sub, out)
}
