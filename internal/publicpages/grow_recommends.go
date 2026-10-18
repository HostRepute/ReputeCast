package publicpages

import (
	"context"
	"errors"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/recommendations"
	"github.com/hostrepute/reputecast/internal/store"
)

// growRecView is a recommendation on a public page.
type growRecView struct {
	ID, Kind, Name, Description, Initials, LogoURL string
	VisitURL, VisitNote                            string
	Consent, Button                                string
	// Outcome is the answer after Subscribe, in place of the button.
	Outcome template.HTML
}

func recViews(items []recommendations.PublicRec, email, picked, outcome string) []growRecView {
	out := make([]growRecView, 0, len(items))
	for _, it := range items {
		v := growRecView{ID: it.ID, Kind: it.Kind, Name: it.Name, Description: it.Description, Initials: it.Initials, LogoURL: it.LogoURL, VisitURL: it.VisitURL}
		if it.Kind == recommendations.KindLink {
			v.VisitNote, v.Button = gt("rec.visitNote", "host", it.Host), gt("rec.visit")
		} else {
			v.Consent, v.Button = gt("rec.consent", "partner", it.Name), gt("rec.subscribe")
		}
		if it.ID == picked && outcome != "" {
			switch outcome {
			case recommendations.OutcomeAccepted:
				v.Outcome = gh("rec.accepted", "partner", it.Name, "email", email)
			case recommendations.OutcomePassed:
				v.Outcome = gh("rec.passed", "partner", it.Name)
			case recommendations.OutcomeLimit:
				v.Outcome = gh("rec.limit", "partner", it.Name)
			default:
				v.Outcome = gh("rec.unavailable", "partner", it.Name)
			}
		}
		out = append(out, v)
	}
	return out
}

// --- "While you're here" on the confirmation page ----------------------------------------

type growConfirm struct {
	Heading, Lead string
	Action        string
	Items         []growRecView
}

// growConfirmSection is the confirmation page's recommendations (count:
// the page was shown to the reader, not re-rendered after Subscribe).
func (h *Handler) growConfirmSection(ctx context.Context, s *site, contactID uuid.UUID, token, picked, outcome string, count bool) *growConfirm {
	rs := h.recommendations()
	if rs == nil || s.preview {
		return nil
	}
	sec, err := rs.ConfirmSection(ctx, s.ws, contactID, count)
	if err != nil {
		h.log().WarnContext(ctx, "recommendations on the confirmation page", "err", err)
		return nil
	}
	if sec == nil || len(sec.Items) == 0 {
		return nil
	}
	pub := s.set.Branding.PublisherName
	g := &growConfirm{Heading: gt("rec.confirmHeading", "publisher", pub), Action: s.links.Confirm(token), Items: recViews(sec.Items, sec.Email, picked, outcome)}
	if sec.ListName != "" {
		g.Lead = gt("rec.confirmLead", "publisher", pub, "list", sec.ListName)
	}
	return g
}

// confirmRecommend is Subscribe on the confirmation page: the request goes
// to the partner and the page shows what happens next.
func (h *Handler) confirmRecommend(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	c, ok := h.claims(s, token, linktoken.Confirm)
	rs := h.recommendations()
	if !ok || rs == nil || !h.parseForm(w, r, 4<<10) {
		h.notFound(w, r, s)
		return
	}
	pc, err := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID)
	if errors.Is(err, audience.ErrContactGone) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	picked := r.PostForm.Get("rec")
	outcome := ""
	if id, err := uuid.Parse(picked); err == nil {
		if outcome, err = rs.Subscribe(r.Context(), s.ws, id, c.ContactID, "", "confirm"); err != nil {
			outcome = recommendations.OutcomeUnavailable
		}
	}
	pl := h.confirmLang(w, r, s, pc)
	body, err := h.confirmBody(r.Context(), s, pc, pl)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	body.Grow = h.growConfirmSection(r.Context(), s, pc.ID, token, picked, outcome, false)
	v := h.view(s, pageTitle(pl, s, "confirm.title"), body)
	h.setLang(v, r, s, pl, nil)
	h.render(w, r, s, "confirm", http.StatusOK, v, csp{}, true)
}

// --- The recommendations page (/recommends) ------------------------------------------------

type recommendsBody struct {
	Heading      string
	For          template.HTML
	Lead         string
	Action       string
	Ask          bool
	AskEmail     string
	AskError     string
	Items        []growRecView
	None         bool
	NoneText     string
	NoneList     string
	LatestHref   string
	PrefsHref    string
	Footer       string
	HasPartner   bool
	EmailFieldID string
}

func (h *Handler) recommendsBody(ctx context.Context, s *site, contactID uuid.UUID, token string) (recommendsBody, *recommendations.PublicRecs, error) {
	rs := h.recommendations()
	page, err := rs.Page(ctx, s.ws, contactID)
	if err != nil {
		return recommendsBody{}, nil, err
	}
	pub := s.set.Branding.PublisherName
	b := recommendsBody{Heading: gt("rec.title", "publisher", pub), Footer: gt("rec.footer", "publisher", pub), EmailFieldID: "rec-email"}
	if page.ListName != "" {
		b.Lead = gt("rec.lead", "publisher", pub, "list", page.ListName)
	} else {
		b.Lead = gt("rec.leadNoList", "publisher", pub)
	}
	if contactID != uuid.Nil {
		b.Action = s.links.Recommends(token)
		b.For = gh("rec.for", "email", page.Email, "href", s.links.Recommends(""))
		b.PrefsHref = s.links.Preferences(h.sign(s, linktoken.Preferences, contactID, uuid.Nil))
	} else {
		b.Action = s.links.Recommends("")
		b.Ask = true
	}
	if len(page.Items) == 0 {
		b.None = true
		b.NoneText = gt("rec.none", "publisher", pub)
		if page.ListName != "" {
			b.NoneList = gt("rec.noneList", "list", page.ListName)
		}
		if s.set.Pages.Archive.Enabled {
			if slug, err := h.Q.PublicLatestIssueSlug(ctx, store.PublicLatestIssueSlugParams{WorkspaceID: s.ws.ID, Lists: h.archiveLists(ctx, s)}); err == nil {
				b.LatestHref = s.links.Issue(slug)
			}
		}
	}
	for _, it := range page.Items {
		if it.Kind == recommendations.KindPartner {
			b.HasPartner = true
		}
	}
	return b, page, nil
}

// recommendsReader is the reader of a recommendations page link, or
// uuid.Nil when the link expired (the page asks for an address).
func (h *Handler) recommendsReader(s *site, rs *recommendations.Service, token string) uuid.UUID {
	if token == "" {
		return uuid.Nil
	}
	id, err := rs.Reader(s.ws, token)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func (h *Handler) recommendsPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	rs := h.recommendations()
	if rs == nil {
		h.notFound(w, r, s)
		return
	}
	token := chi.URLParam(r, "token")
	reader := h.recommendsReader(s, rs, token)
	b, page, err := h.recommendsBody(r.Context(), s, reader, token)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if reader != uuid.Nil {
		b.Items = recViews(page.Items, page.Email, "", "")
	} else {
		b.Items = recViews(page.Items, "", "", "")
		if token != "" {
			// An expired personal link asks for the address first.
			b.Ask = true
		}
	}
	if !b.HasPartner {
		b.Ask = false
	}
	h.growPage(w, r, s, "grow-recommends", http.StatusOK, b.Heading, b, token != "")
}

// recommendsPost is Subscribe on the recommendations page: for the reader
// of the link, or the address typed when the link expired.
func (h *Handler) recommendsPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	rs := h.recommendations()
	if rs == nil || !h.parseForm(w, r, 8<<10) {
		h.notFound(w, r, s)
		return
	}
	token := chi.URLParam(r, "token")
	reader := h.recommendsReader(s, rs, token)
	b, page, err := h.recommendsBody(r.Context(), s, reader, token)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	picked := r.PostForm.Get("rec")
	email := page.Email
	status := http.StatusOK
	outcome := ""
	if reader == uuid.Nil {
		b.Ask = true
		b.AskEmail = r.PostForm.Get("email")
		addr, problem := emailaddr.Check(b.AskEmail)
		if problem != "" {
			b.AskError = gt("rec.emailInvalid")
			b.Items = recViews(page.Items, "", "", "")
			h.growPage(w, r, s, "grow-recommends", http.StatusUnprocessableEntity, b.Heading, b, true)
			return
		}
		email = addr
	}
	ip := ""
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	if id, err := uuid.Parse(picked); err == nil {
		if !h.Forms.AllowPerMinute("recommends:"+ip, 10) {
			outcome = recommendations.OutcomePassed
		} else if outcome, err = rs.Subscribe(r.Context(), s.ws, id, reader, email, "page"); err != nil {
			outcome = recommendations.OutcomeUnavailable
		}
	}
	b.Items = recViews(page.Items, email, picked, outcome)
	h.growPage(w, r, s, "grow-recommends", status, b.Heading, b, true)
}

// recommendsVisit counts a click on a link recommendation and sends the
// reader to it.
func (h *Handler) recommendsVisit(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	rs := h.recommendations()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if rs == nil || err != nil {
		h.notFound(w, r, s)
		return
	}
	to, ok := rs.Visit(r.Context(), s.ws, id)
	if !ok {
		h.notFound(w, r, s)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, to, http.StatusFound) //nolint:gosec // the recommendation's address, set by the team
}

// --- "Invite friends" in the preference center -----------------------------------------------

type growInvite struct {
	Text, Href string
}

// growInvite is the preference center's card for a reader on a program's
// list while it's on.
func (h *Handler) growInvite(ctx context.Context, s *site, contactID uuid.UUID) *growInvite {
	rs := h.referrals()
	if rs == nil || s.preview {
		return nil
	}
	card, err := rs.PreferenceCardFor(ctx, s.ws, contactID)
	if err != nil || card == nil {
		return nil
	}
	g := &growInvite{Href: card.HubURL}
	switch {
	case card.NextCount == 0:
		g.Text = gt("prefs.inviteAll", "counted", card.Counted)
	case card.Counted == 0:
		g.Text = gt("prefs.inviteNone", "list", card.ListName, "count", card.NextCount, "reward", articled(card.NextReward))
	default:
		g.Text = gt("prefs.inviteSome", "counted", card.Counted, "count", int64(card.NextCount)-card.Counted, "reward", articled(card.NextReward))
	}
	return g
}
