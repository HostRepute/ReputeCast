package publicpages

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/website"
)

// The member's account page (PublicAccount): profile, plan, newsletters
// (the preference center's), sessions, and the sections later phases add.
// It is a signed-in page: never cached, never indexed.

// accountListView is a newsletter on the account page.
type accountListView struct {
	ID, Name, Description string
	Checked, Pending      bool
}

// radioView is one choice of a group of radios.
type radioView struct {
	Value, Label string
	Checked      bool
}

// accountSessionView is a session line.
type accountSessionView struct {
	Title, Text string
	Current     bool
}

// accountSectionView is a section in the side menu.
type accountSectionView struct {
	ID, Title string
	HTML      template.HTML
}

// accountBody is the account page.
type accountBody struct {
	Initials, Color, Name, Line string
	SignoutAction               string
	Sections                    []accountSectionView
	// Profile.
	Email, EmailNote        string
	EmailAction, NameAction string
	EditEmail, EditName     bool
	EmailHref, NameHref     string
	FirstName, LastName     string
	DisplayName             string
	EmailError, EmailSent   string
	CancelHref              string
	// Plan.
	Plan *accountPlanView
	// Newsletters.
	Lists          []accountListView
	ListIDs        []string
	Frequency      []radioView
	FrequencyHint  string
	Pause          []radioView
	PausedUntil    string
	NewslettersURL string
	UnsubscribeURL string
	// Sessions.
	Sessions      []accountSessionView
	OthersText    string
	EverywhereURL string
	SessionsNote  string
	Extra         []accountSectionView
}

// accountPlanView is the Plan section.
type accountPlanView struct {
	Kind, Heading, Price, Line string
	Badge                      string
	Manage, ManageLabel        string
	Cancel, CancelLabel        string
	Upgrade, UpgradeLabel      string
	Meter                      string
	Test                       bool
	// A free member's offer: the first tier, and the others in a line.
	Offer     *paywallTierView
	OfferText []string
	Others    template.HTML
	Note      string
}

// accountPage is /account.
func (h *Handler) accountPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil || !h.siteOn(r.Context(), s) {
		h.siteNotFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	if v.Session == nil {
		http.Redirect(w, r, h.signinHref(s, s.path("account")), http.StatusSeeOther) //nolint:gosec // G710: the site's own sign-in page
		return
	}
	q := r.URL.Query()
	h.renderAccount(w, r, s, v, http.StatusOK, accountState{editEmail: q.Get("edit") == "email", editName: q.Get("edit") == "name"})
}

// accountState is what the page shows besides the account itself.
type accountState struct {
	editEmail, editName bool
	email, emailError   string
	emailSent           string
}

func (h *Handler) renderAccount(w http.ResponseWriter, r *http.Request, s *site, v *siteView, status int, st accountState) {
	ctx := r.Context()
	acc, err := h.Website.Account(ctx, s.ws, v.Session, v.labels.Lang())
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	loc := location(s.ws)
	name := strings.TrimSpace(acc.FirstName + " " + acc.LastName)
	plan := v.T("account.freeMember")
	if len(acc.Tiers) > 0 {
		plan = website.TierLabel(acc.Tiers[0])
	}
	self := s.path("account")
	b := accountBody{Initials: initials(firstNonEmpty(name, acc.Email)), Color: avatarColor(firstNonEmpty(name, acc.Email)), Name: v.T("account.title"),
		Line: v.T("account.since", "email", acc.Email, "plan", plan, "date", v.labels.FullDate(acc.Since.In(loc))), SignoutAction: s.path("signout"),
		Email: acc.Email, EmailNote: v.T("account.emailNote"), EmailAction: s.path("account", "email"), NameAction: s.path("account", "name"),
		EditEmail: st.editEmail || st.emailError != "", EditName: st.editName, EmailHref: self + "?edit=email#profile", NameHref: self + "?edit=name#profile",
		FirstName: acc.FirstName, LastName: acc.LastName, DisplayName: name, EmailError: st.emailError, EmailSent: st.emailSent, CancelHref: self + "#profile",
		NewslettersURL: s.path("account", "newsletters"), UnsubscribeURL: s.path("account", "unsubscribe"), EverywhereURL: s.path("account", "everywhere"),
		SessionsNote: v.T("account.sessionsNote")}
	if strings.HasPrefix(v.labels.Code(), "en") {
		b.Line = v.T("account.since", "email", acc.Email, "plan", plan, "date", acc.Since.In(loc).Format("2 Jan 2006"))
	}
	if b.DisplayName == "" {
		b.DisplayName = v.T("account.noName")
	}
	if st.email != "" {
		b.Email = st.email
	}
	b.Sections = []accountSectionView{{ID: "profile", Title: v.T("account.profile")}, {ID: "plan", Title: v.T("account.plan")},
		{ID: "newsletters", Title: v.T("account.newsletters")}}
	// Plan.
	pv := &accountPlanView{Kind: "free", Heading: v.T("account.freeMember"), Badge: v.T("account.planFree")}
	if p := acc.Plan; p != nil {
		pv.Kind, pv.Test = p.Kind, p.Test
		if p.Kind != "free" {
			pv.Heading, pv.Price, pv.Line, pv.Badge = p.Heading, p.Price, p.Line, ""
			pv.Manage, pv.ManageLabel, pv.Cancel, pv.CancelLabel = p.Manage, p.ManageLabel, p.Cancel, p.CancelLabel
		}
		pv.Upgrade, pv.UpgradeLabel = p.Upgrade, p.UpgradeLabel
	}
	if m := acc.Meter; m != nil {
		titles := ""
		if len(m.Reads) > 0 {
			parts := make([]string, 0, len(m.Reads))
			for _, rd := range m.Reads {
				parts = append(parts, rd.Title+", "+v.labels.DayMonth(rd.At.In(loc)))
			}
			titles = " (" + strings.Join(parts, "; ") + ")"
		}
		pv.Meter = v.T("account.meter", "used", m.Used, "limit", m.Limit, "titles", titles, "next", nextName(v, m.Next))
	}
	if pv.Kind == "free" {
		h.freeOffer(r, s, v, pv)
	} else if pv.Manage != "" {
		pv.Note = v.T("account.billingNote", "publisher", s.set.Branding.PublisherName)
	}
	b.Plan = pv
	// Newsletters.
	copyOf := h.copyOf(ctx, s)
	for _, l := range acc.Lists {
		nm := l.Name
		if copyOf != nil {
			nm = copyOf.ListName(v.Lang, audience.PublicList{ID: l.ID, Name: l.Name, Description: l.Description})
		}
		b.Lists = append(b.Lists, accountListView{ID: l.ID.String(), Name: nm, Description: l.Description, Checked: l.Subscribed, Pending: l.Pending})
		b.ListIDs = append(b.ListIDs, l.ID.String())
	}
	freq := acc.Frequency
	if freq == "" {
		freq = "all"
	}
	for _, f := range audience.Frequencies {
		b.Frequency = append(b.Frequency, radioView{Value: f, Label: v.T("account.freq." + f), Checked: f == freq})
	}
	paused := acc.PausedUntil != nil && acc.PausedUntil.After(h.now())
	b.Pause = []radioView{{Value: "0", Label: v.T("account.pause.none"), Checked: !paused}, {Value: "30", Label: v.T("account.pause.30")},
		{Value: "90", Label: v.T("account.pause.90")}}
	if paused {
		b.PausedUntil = v.T("account.pausedUntil", "date", v.labels.FullDate(acc.PausedUntil.In(loc)))
	}
	// The sections later phases add.
	for _, sec := range acc.Sections {
		b.Extra = append(b.Extra, accountSectionView{ID: sec.ID, Title: sec.Title, HTML: template.HTML(sec.HTML)}) //nolint:gosec // drawn by ReputeCast's own sections
		b.Sections = append(b.Sections, accountSectionView{ID: sec.ID, Title: sec.Title})
	}
	b.Sections = append(b.Sections, accountSectionView{ID: "sessions", Title: v.T("account.sessions")})
	// Sessions.
	others := 0
	for _, se := range acc.Sessions {
		if !se.Current {
			others++
			continue
		}
		method := v.T("account.method." + se.Method)
		b.Sessions = append(b.Sessions, accountSessionView{Title: v.T("account.thisBrowser"), Current: true,
			Text: v.T("account.signedInWith", "date", v.labels.DayMonth(se.CreatedAt.In(loc)), "method", method, "days", acc.SessionDays)})
	}
	if others > 0 {
		b.OthersText = v.T("account.otherSessions", "count", others)
	}
	v.Body = b
	v.Title = v.T("site.title", "page", v.T("account.title"), "site", v.Frame.Title)
	v.Robots = "noindex, nofollow"
	v.private = true
	v.mark("account", v.Frame)
	h.renderSite(w, r, s, "account", status, v)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// accountPost applies one of the account page's forms.
func (h *Handler) accountPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil || !h.parseForm(w, r, 16<<10) {
		h.siteNotFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	if v.Session == nil {
		http.Redirect(w, r, h.signinHref(s, s.path("account")), http.StatusSeeOther) //nolint:gosec // G710: the site's own sign-in page
		return
	}
	ctx := r.Context()
	sess := v.Session
	back := func(kind, anchor string) {
		if kind != "" {
			h.flash(w, r, s, kind)
		}
		http.Redirect(w, r, s.path("account")+anchor, http.StatusSeeOther) //nolint:gosec // G710: the site's own account page
	}
	switch chi.URLParam(r, "action") {
	case "name":
		if err := h.Website.SaveName(ctx, s.ws, sess, r.PostForm.Get("first_name"), r.PostForm.Get("last_name")); err != nil {
			h.siteFail(w, r, s, err)
			return
		}
		back("name", "#profile")
	case "email":
		out, err := h.Website.ChangeEmail(ctx, s.ws, sess, r.PostForm.Get("email"), h.browserCookie(w, r, s))
		if err != nil {
			h.siteFail(w, r, s, err)
			return
		}
		switch out.Status {
		case website.SigninInvalid:
			h.renderAccount(w, r, s, v, http.StatusUnprocessableEntity, accountState{editEmail: true, email: r.PostForm.Get("email"), emailError: out.Message})
		case website.SigninLimited:
			h.signinLimited(w, r, s, out, s.path("account"))
		default:
			h.renderAccount(w, r, s, v, http.StatusOK, accountState{emailSent: v.T("account.emailSent", "email", out.Email)})
		}
	case "newsletters":
		in := audience.PreferencesInput{Frequency: r.PostForm.Get("frequency"), PauseDays: -1}
		for _, raw := range r.PostForm["shown"] {
			if id, err := uuid.Parse(raw); err == nil {
				in.Shown = append(in.Shown, id)
			}
		}
		for _, raw := range r.PostForm["list"] {
			if id, err := uuid.Parse(raw); err == nil {
				in.Subscribed = append(in.Subscribed, id)
			}
		}
		if p, err := strconv.Atoi(r.PostForm.Get("pause")); err == nil && (p == 0 || p == 30 || p == 90) {
			in.PauseDays = p
		}
		if in.Frequency != "" && !contains(audience.Frequencies, in.Frequency) {
			in.Frequency = ""
		}
		if err := h.Website.SaveNewsletters(ctx, s.ws, sess, in, h.evidence(r, "website_account")); err != nil {
			h.siteFail(w, r, s, err)
			return
		}
		back("newsletters", "#newsletters")
	case "unsubscribe":
		if err := h.Website.UnsubscribeAll(ctx, s.ws, sess, h.evidence(r, "website_account")); err != nil {
			h.siteFail(w, r, s, err)
			return
		}
		back("unsubscribed", "#newsletters")
	case "everywhere":
		if err := h.Website.SignOutEverywhereSelf(ctx, s.ws, sess); err != nil {
			h.siteFail(w, r, s, err)
			return
		}
		h.setCookie(w, r, s, website.SessionCookie, "", -1)
		h.flash(w, r, s, "everywhere")
		http.Redirect(w, r, s.path(), http.StatusSeeOther) //nolint:gosec // G710: the site's home page
	default:
		h.siteNotFound(w, r, s)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// freeOffer is what a free member could buy: the first tier with its
// benefits, the others on the upgrade page.
func (h *Handler) freeOffer(r *http.Request, s *site, v *siteView, pv *accountPlanView) {
	ctx := r.Context()
	_, tiers, err := h.Website.Rules(ctx, s.ws)
	if err != nil || len(tiers) == 0 {
		return
	}
	offers, pay, err := h.Website.PaywallTiers(ctx, s.ws, tiers)
	if err != nil || len(offers) == 0 {
		return
	}
	up := s.path("upgrade")
	if h.Earn != nil {
		up = h.Earn.UpgradeURL(ctx, s.ws, v.Session.ContactID, uuid.Nil)
	}
	first := offers[0]
	pv.Offer = &paywallTierView{Name: first.Name, Price: tierPrice(v, website.PaywallTier{Monthly: first.Monthly, Yearly: first.Yearly, TrialDays: first.TrialDays}),
		Href: up, Button: v.T("site.upgrade"), Primary: true}
	pv.OfferText = first.Benefits
	if len(offers) > 1 {
		names := make([]string, 0, len(offers)-1)
		for _, o := range offers[1:] {
			price := o.Yearly + " " + v.T("gate.perYear")
			if o.Yearly == "" {
				price = o.Monthly + " " + v.T("gate.perMonth")
			}
			names = append(names, o.Name+", "+price)
		}
		pv.Others = v.H("account.others", "tiers", strings.Join(names, "; "), "href", up)
	}
	if pay.Provider != "" {
		pv.Note = v.T("account.paymentsVia", "publisher", s.set.Branding.PublisherName, "provider", pay.Provider)
	}
}
