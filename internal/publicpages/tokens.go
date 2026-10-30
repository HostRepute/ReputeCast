package publicpages

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

type gotView struct {
	Name, Text, AddHref, Offer string
	On                         bool
}

type confirmBody struct {
	Heading, Email, Sender, LatestHref, PrefsHref, Proof string
	Lists                                                []gotView
	// Download is the form's download after confirmation (sell_confirm.go).
	Download *confirmDownload
	// Grow is "While you're here": recommendations (grow_recommends.go).
	Grow *growConfirm
	// Texted is "We've also texted +1 312 ••• 0123 to confirm." when the
	// signup gave SMS consent (phase 8).
	Texted string
}

func (h *Handler) evidence(r *http.Request, source string) audience.PublicEvidence {
	return audience.PublicEvidence{Source: source, IP: clientIP(r), UserAgent: r.UserAgent(), At: h.now()}
}

// confirm completes double opt-in. GET is the link in the email (HEAD,
// what link scanners send, never confirms).
func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	c, ok := h.claims(s, chi.URLParam(r, "token"), linktoken.Confirm)
	if !ok {
		h.notFound(w, r, s)
		return
	}
	res, err := h.Forms.Confirm(r.Context(), s.ws, c.ContactID, h.evidence(r, audience.SourceConfirmLink))
	if errors.Is(err, audience.ErrContactGone) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	pc, err := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	pl := h.confirmLang(w, r, s, pc)
	if res.Blocked {
		h.messageIn(w, r, s, pl, http.StatusOK, pageTitle(pl, s, "confirm.blocked.title"), messageBody{Heading: pl.Labels.Text("confirm.blocked.heading", nil),
			Text: pl.Labels.Text("confirm.blocked.text", i18n.Args{"publisher": s.set.Branding.PublisherName})}, true)
		return
	}
	body, err := h.confirmBody(r.Context(), s, pc, pl)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	body.Download = h.confirmDownloadFor(r.Context(), s, pc, pl.Code)
	body.Grow = h.growConfirmSection(r.Context(), s, pc.ID, chi.URLParam(r, "token"), "", "", true)
	v := h.view(s, pageTitle(pl, s, "confirm.title"), body)
	h.setLang(v, r, s, pl, nil)
	h.render(w, r, s, "confirm", http.StatusOK, v, csp{}, true)
}

// confirmLang is the confirmation page's language: the one the signup form
// was shown in (the confirmation email's), else the contact's.
func (h *Handler) confirmLang(w http.ResponseWriter, r *http.Request, s *site, pc audience.PublicContact) pageLang {
	if picked(r) == "" {
		if code, err := h.Q.FormSubmissionLatestLanguage(r.Context(), pc.ID); err == nil {
			if cp := h.copyOf(r.Context(), s); cp != nil && cp.Live(code) {
				return pageLang{Code: code, Labels: cp.Labels(code)}
			}
		}
	}
	return h.contactLang(w, r, s, pc.Language)
}

func (h *Handler) confirmBody(ctx context.Context, s *site, pc audience.PublicContact, pl pageLang) (confirmBody, error) {
	l := pl.Labels
	b := confirmBody{Heading: l.Text("confirm.heading", nil), Email: pc.Email, Lists: []gotView{}}
	if h.Forms != nil && h.Forms.Texts != nil {
		b.Texted = h.Forms.Texts.TextedContact(ctx, s.ws, pc.ID, pl.Code)
	}
	if pc.FirstName != "" {
		b.Heading = l.Text("confirm.headingName", i18n.Args{"name": pc.FirstName})
	}
	cp := h.copyOf(ctx, s)
	pages := s.set.Pages
	if cp != nil {
		pages = cp.Pages(pl.Code, pages)
	}
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return b, err
	}
	// The lists the form offered, then any other list the contact is on.
	offered := []uuid.UUID{}
	if last, err := h.Q.FormLastSubmissionForContact(ctx, pc.ID); err == nil && last.FormID != nil {
		if f, err := h.Q.FormGet(ctx, store.FormGetParams{WorkspaceID: s.ws.ID, ID: *last.FormID}); err == nil {
			offered = f.ListIds
		}
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return b, err
	}
	if len(offered) == 0 {
		// Not from a form: the lists of the subscribe page's form.
		if id := s.set.HostedFormID(); id != nil {
			if f, err := h.Q.FormGet(ctx, store.FormGetParams{WorkspaceID: s.ws.ID, ID: *id}); err == nil {
				offered = f.ListIds
			}
		}
	}
	if len(offered) == 0 {
		for _, l := range lists {
			if l.Public {
				offered = append(offered, l.ID)
			}
		}
	}
	next, err := h.nextIssues(ctx, s)
	if err != nil {
		return b, err
	}
	prefs := s.links.Preferences(h.sign(s, linktoken.Preferences, pc.ID, uuid.Nil))
	for _, pl2 := range lists {
		on := pc.Subscribed(pl2.ID)
		if !on && !slices.Contains(offered, pl2.ID) {
			continue
		}
		text, name := pl2.Description, pl2.Name
		if cp != nil {
			text, name = cp.ListDescription(pl.Code, pl2), cp.ListName(pl.Code, pl2)
		}
		if t := pages.Confirm.ListText[pl2.ID.String()]; t != "" {
			text = t
		}
		if at, ok := next[pl2.ID]; ok && on {
			text = strings.TrimSpace(text + " " + l.Text("confirm.nextIssue", i18n.Args{"date": l.NextDate(at.In(location(s.ws)))}))
		}
		b.Lists = append(b.Lists, gotView{Name: name, Text: text, On: on, AddHref: prefs, Offer: pages.Confirm.OfferText[pl2.ID.String()]})
	}
	if s.set.Pages.Confirm.InboxTip {
		if sender, err := h.Q.FormsDefaultSender(ctx, s.ws.ID); err == nil {
			b.Sender = sender.Email
		}
	}
	if s.set.Pages.Archive.Enabled {
		if slug, err := h.Q.PublicLatestIssueSlug(ctx, store.PublicLatestIssueSlugParams{WorkspaceID: s.ws.ID, Lists: h.archiveLists(ctx, s)}); err == nil {
			b.LatestHref = s.links.Issue(slug)
		}
	}
	b.PrefsHref = prefs
	if pc.Confirmed != nil {
		at := l.Stamp(pc.Confirmed.At.In(location(s.ws)))
		if pc.Confirmed.IP != nil {
			b.Proof = l.Text("confirm.proofIp", i18n.Args{"date": at, "ip": pc.Confirmed.IP.String()})
		} else {
			b.Proof = l.Text("confirm.proof", i18n.Args{"date": at})
		}
	}
	return b, nil
}

// nextIssues is the next scheduled send of each list.
func (h *Handler) nextIssues(ctx context.Context, s *site) (map[uuid.UUID]time.Time, error) {
	rows, err := h.Q.PublicScheduledCampaigns(ctx, store.PublicScheduledCampaignsParams{WorkspaceID: s.ws.ID, After: h.now()})
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]time.Time{}
	for _, r := range rows {
		var aud struct {
			Lists []uuid.UUID `json:"lists"`
		}
		_ = json.Unmarshal(r.Audience, &aud)
		for _, l := range aud.Lists {
			if t, ok := out[l]; !ok || r.ScheduledAt.Before(t) {
				out[l] = r.ScheduledAt
			}
		}
	}
	return out, nil
}

func location(ws *store.Workspace) *time.Location {
	if loc, err := time.LoadLocation(ws.Timezone); err == nil {
		return loc
	}
	return time.UTC
}

type choiceView struct {
	Value, Label string
	Checked      bool
	// Lang is the language a choice is written in (Email language).
	Lang string
}

type preferencesBody struct {
	Email, FirstName, Action, NotYouHref, UnsubscribeHref, Shown, Saved, Alert string
	Lists                                                                      []listView
	Frequencies                                                                []choiceView
	Pauses                                                                     []choiceView
	AskName                                                                    bool
	// Languages is "Email language": every workspace language, in its own
	// name, the contact's checked.
	Languages    []choiceView
	LanguageNote string
	// Notifications is transactional mail's Optional categories; Note
	// stands in for it when there are none.
	Notifications *notificationsView
	Note          string
	// Paid is "Your paid subscription" (Earn), nil when there's nothing to show.
	Paid *earn.PreferencesPaid
	// Account links a member signed in on the website to their account page.
	Account *prefsAccount
	// Invite is Grow's "Invite friends" card (grow_recommends.go).
	Invite *growInvite
	// Texts are the Texts switches beside the lists (SMS, phase 8), nil
	// when the contact has no number or the workspace doesn't text.
	Texts *textsView
	// Push is "Turn off notifications in this browser" (X22).
	Push *PushTurnOff
}

// prefsAccount is the preference center's line for a reader signed in on
// the website as the contact the link is for.
type prefsAccount struct{ Text, Link, Href string }

func (h *Handler) prefsAccountFor(r *http.Request, s *site, contact uuid.UUID, pl pageLang) *prefsAccount {
	if h.Website == nil || s.preview || !h.siteOn(r.Context(), s) {
		return nil
	}
	sess := h.session(r, s)
	if sess == nil || sess.ContactID != contact {
		return nil
	}
	sv, err := h.siteViewIn(r.Context(), s, pl)
	if err != nil {
		h.log().WarnContext(r.Context(), "preference center account link", "err", err)
		return nil
	}
	host := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSuffix(s.canonPath(), "/"), "https://"), "http://")
	return &prefsAccount{Text: sv.T("site.prefsSignedIn", "host", host), Link: sv.T("site.prefsAccount"), Href: sv.Href.Account}
}

func (h *Handler) preferencesPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	// A placement test copy's links carry its message alone, as
	// transactional mail's do; they never change anything.
	if h.placementCopy(r, s, token, linktoken.Preferences) {
		h.placementNotice(w, r, s)
		return
	}
	c, ok := h.claims(s, token, linktoken.Preferences)
	if !ok {
		h.notFound(w, r, s)
		return
	}
	pc, err := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID)
	if errors.Is(err, audience.ErrContactGone) {
		// Transactional mail reaches people who aren't contacts.
		if m, ok, terr := h.transactionalMessage(r.Context(), s, c); terr == nil && ok {
			h.notificationsPage(w, r, s, c, m, token, r.URL.Query().Get("saved") == "1", http.StatusOK)
			return
		}
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	pl := h.contactLang(w, r, s, pc.Language)
	body, err := h.preferencesBody(r.Context(), s, pc, s.links.Preferences(token), c, pl)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if r.URL.Query().Get("saved") == "1" {
		body.Saved = pl.Labels.Text("prefs.saved", nil)
	}
	body.Account = h.prefsAccountFor(r, s, pc.ID, pl)
	pushOn := ""
	if host := pushHost(s); host != "" && h.Push != nil {
		if body.Push = h.Push.TurnOff(r.Context(), s.ws, host, pl.Labels.Lang()); body.Push != nil {
			pushOn = host
		}
	}
	v := h.view(s, pageTitle(pl, s, "prefs.heading"), body)
	v.Scripts = append(v.Scripts, h.pushScripts(r.Context(), s, pushOn)...)
	h.setLang(v, r, s, pl, nil)
	h.render(w, r, s, "preferences", http.StatusOK, v, csp{}, true)
}

func (h *Handler) preferencesBody(ctx context.Context, s *site, pc audience.PublicContact, action string, c linktoken.Claims, pl pageLang) (preferencesBody, error) {
	l := pl.Labels
	cp := h.copyOf(ctx, s)
	p := s.set.Pages.Preferences
	if cp != nil {
		p = cp.Pages(pl.Code, s.set.Pages).Preferences
	}
	b := preferencesBody{Email: pc.Email, FirstName: pc.FirstName, Action: action, NotYouHref: s.links.Subscribe(""), AskName: p.FirstName,
		UnsubscribeHref: s.links.Unsubscribe(h.sign(s, linktoken.Unsubscribe, pc.ID, c.MessageID))}
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return b, err
	}
	shown := []string{}
	for _, pl2 := range lists {
		offered := pl2.Public
		if len(p.Lists) > 0 {
			offered = slices.Contains(p.Lists, pl2.ID.String())
		}
		st := pc.Lists[pl2.ID]
		if !offered && st != "subscribed" && st != "pending" {
			continue
		}
		text, name := pl2.Description, pl2.Name
		if cp != nil {
			text, name = cp.ListDescription(pl.Code, pl2), cp.ListName(pl.Code, pl2)
		}
		if t := p.ListText[pl2.ID.String()]; t != "" {
			text = t
		}
		b.Lists = append(b.Lists, listView{ID: pl2.ID.String(), Name: name, Text: text, Checked: (st == "subscribed" && pc.Status == "active") || st == "pending"})
		shown = append(shown, pl2.ID.String())
	}
	b.Shown = strings.Join(shown, ",")
	h.prefTexts(ctx, s, pc.ID, pl.Code, &b)
	if b.Notifications, err = h.notificationsSection(ctx, s, pc.Email, l); err != nil {
		return b, err
	}
	if h.Earn != nil {
		if b.Paid, err = h.Earn.PreferencesPaid(ctx, s.ws, pc.ID, l.Lang()); err != nil {
			return b, err
		}
	}
	if p.Frequency {
		for _, f := range []struct{ v, key string }{{"all", "prefs.all"}, {"weekly", "prefs.weekly"}, {"monthly", "prefs.monthly"}} {
			b.Frequencies = append(b.Frequencies, choiceView{Value: f.v, Label: l.Text(f.key, nil), Checked: pc.Frequency == f.v})
		}
	}
	if p.Pause {
		if pc.PausedUntil != nil && pc.PausedUntil.After(h.now()) {
			b.Pauses = []choiceView{
				{Value: "keep", Label: l.Text("prefs.pausedUntil", i18n.Args{"date": l.DayMonth(pc.PausedUntil.In(location(s.ws)))}), Checked: true},
				{Value: "0", Label: l.Text("prefs.resume", nil)}, {Value: "30", Label: l.Text("prefs.pause30", nil)}, {Value: "90", Label: l.Text("prefs.pause90", nil)},
			}
		} else {
			b.Pauses = []choiceView{{Value: "0", Label: l.Text("prefs.noPause", nil), Checked: true}, {Value: "30", Label: l.Text("prefs.pause30", nil)},
				{Value: "90", Label: l.Text("prefs.pause90", nil)}}
		}
	}
	// "Email language": every workspace language, even one whose pages
	// aren't Live yet.
	if cp != nil && len(cp.Languages.Languages) > 1 {
		mine := languages.Base(pc.Language)
		for _, code := range cp.Languages.Languages {
			meta, _ := languages.Lookup(code)
			b.Languages = append(b.Languages, choiceView{Value: code, Label: meta.Native, Checked: code == mine, Lang: code})
		}
		fallback, _ := languages.Lookup(cp.Languages.Fallback)
		b.LanguageNote = l.Text("prefs.languageNote", i18n.Args{"publisher": s.set.Branding.PublisherName, "fallback": fallback.Native})
	}
	b.Invite = h.growInvite(ctx, s, pc.ID)
	return b, nil
}

func (h *Handler) preferencesPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	c, ok := h.claims(s, token, linktoken.Preferences)
	if !ok {
		h.notFound(w, r, s)
		return
	}
	if !h.parseForm(w, r, 16<<10) {
		http.Error(w, h.visitorLang(nil, r, s, nil).Labels.Text("page.unreadable", nil), http.StatusBadRequest)
		return
	}
	f := r.PostForm
	catOn, catShown := notificationsForm(r)
	if c.ContactID == uuid.Nil {
		// Someone who got transactional mail but isn't a contact.
		m, ok, err := h.transactionalMessage(r.Context(), s, c)
		if err != nil {
			h.serverError(w, r, s, err)
			return
		}
		if !ok {
			h.notFound(w, r, s)
			return
		}
		mid := c.MessageID
		if err := db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
			return h.Notifications.Save(r.Context(), tx, s.ws, m.Email, catOn, catShown, &mid)
		}); err != nil {
			h.serverError(w, r, s, err)
			return
		}
		http.Redirect(w, r, s.links.Preferences(token)+"?saved=1", http.StatusSeeOther) //nolint:gosec // our own origin; the token was verified and is path-escaped
		return
	}
	in := audience.PreferencesInput{Frequency: f.Get("frequency"), PauseDays: -1}
	if f.Has("language") {
		lang := f.Get("language")
		in.Language = &lang
	}
	for _, raw := range strings.Split(f.Get("shown"), ",") {
		if id, err := uuid.Parse(strings.TrimSpace(raw)); err == nil {
			in.Shown = append(in.Shown, id)
		}
	}
	for _, raw := range f["list"] {
		if id, err := uuid.Parse(raw); err == nil {
			in.Subscribed = append(in.Subscribed, id)
		}
	}
	switch f.Get("pause") {
	case "0":
		in.PauseDays = 0
	case "30":
		in.PauseDays = 30
	case "90":
		in.PauseDays = 90
	}
	if s.set.Pages.Preferences.FirstName && f.Has("first_name") {
		name := f.Get("first_name")
		in.FirstName = &name
	}
	var confirmTexts []uuid.UUID
	err := db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
		_, err := h.Audience.SavePreferences(r.Context(), tx, s.ws, c.ContactID, in, h.evidence(r, audience.SourcePreferences))
		if err != nil {
			return err
		}
		// Phase 8: the Texts switches.
		if confirmTexts, err = h.saveTexts(r.Context(), tx, r, s, c.ContactID, "preference_center"); err != nil {
			return err
		}
		if h.Notifications == nil || len(catShown) == 0 {
			return nil
		}
		pc, err := h.Audience.LoadPublicContact(r.Context(), store.New(tx), s.ws, c.ContactID)
		if err != nil {
			return err
		}
		var mid *uuid.UUID
		if c.MessageID != uuid.Nil {
			id := c.MessageID
			mid = &id
		}
		return h.Notifications.Save(r.Context(), tx, s.ws, pc.Email, catOn, catShown, mid)
	})
	var fe *audience.FieldError
	switch {
	case errors.Is(err, audience.ErrContactGone):
		h.notFound(w, r, s)
		return
	case errors.As(err, &fe):
		pc, lerr := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID)
		if lerr != nil {
			h.serverError(w, r, s, lerr)
			return
		}
		pl := h.contactLang(w, r, s, pc.Language)
		body, berr := h.preferencesBody(r.Context(), s, pc, s.links.Preferences(token), c, pl)
		if berr != nil {
			h.serverError(w, r, s, berr)
			return
		}
		body.Alert = fieldProblem(fe, pl.Labels)
		body.Account = h.prefsAccountFor(r, s, pc.ID, pl)
		v := h.view(s, pageTitle(pl, s, "prefs.heading"), body)
		h.setLang(v, r, s, pl, nil)
		h.render(w, r, s, "preferences", http.StatusUnprocessableEntity, v, csp{}, true)
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	}
	if len(confirmTexts) > 0 {
		h.TextPrefs.Confirm(r.Context(), s.ws, c.ContactID, confirmTexts)
	}
	http.Redirect(w, r, s.links.Preferences(token)+"?saved=1", http.StatusSeeOther) //nolint:gosec // our own origin; the token was verified and is path-escaped
}

type requestBody struct {
	Action, Token, Email, EmailError, Alert string
}

func (h *Handler) requestPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	body := requestBody{Action: s.links.PreferencesRequest(), Token: h.Forms.FormToken("prefs:"+s.ws.ID.String(), h.now())}
	pl := h.visitorLang(w, r, s, nil)
	v := h.view(s, pageTitle(pl, s, "prefs.heading"), body)
	h.setLang(v, r, s, pl, nil)
	v.Robots = "noindex, follow"
	v.Canonical = s.canon.PreferencesRequest()
	h.render(w, r, s, "request", http.StatusOK, v, csp{}, false)
}

func (h *Handler) requestPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if !h.parseForm(w, r, 8<<10) {
		http.Error(w, h.visitorLang(nil, r, s, nil).Labels.Text("page.unreadable", nil), http.StatusBadRequest)
		return
	}
	f := r.PostForm
	pl := h.langOf(w, r, s, languages.Base(f.Get("_lang")))
	l := pl.Labels
	email := strings.TrimSpace(f.Get("email"))
	ip := "unknown"
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	publisher := s.set.Branding.PublisherName
	retry := func(status int, emailErr, alert string) {
		body := requestBody{Action: s.links.PreferencesRequest(), Token: h.Forms.FormToken("prefs:"+s.ws.ID.String(), h.now()), Email: email, EmailError: emailErr, Alert: alert}
		v := h.view(s, pageTitle(pl, s, "prefs.heading"), body)
		h.setLang(v, r, s, pl, nil)
		h.render(w, r, s, "request", status, v, csp{}, true)
	}
	if !h.Forms.AllowPerMinute("prefs-ip:"+ip, 3) {
		w.Header().Set("Retry-After", "60")
		retry(http.StatusTooManyRequests, "", l.Text("error.rateLimited", nil))
		return
	}
	if email == "" || !strings.Contains(email, "@") {
		retry(http.StatusUnprocessableEntity, l.Text("prefs.request.missing", nil), "")
		return
	}
	sent := func() {
		h.messageIn(w, r, s, pl, http.StatusOK, pageTitle(pl, s, "form.confirm.heading"), messageBody{Icon: "mail", Heading: l.Text("form.confirm.heading", nil),
			Lead: labelHTML(l, "prefs.request.sent", i18n.Args{"email": email, "publisher": publisher}, "")}, true)
	}
	// Bots fill the invisible field; they see the same answer and nothing is sent.
	if f.Get("website") != "" {
		sent()
		return
	}
	// Whether mail can go out is the same for every address, so saying so
	// reveals nothing; after that, every address gets the same answer.
	if !h.Forms.SystemMailAvailable(r.Context(), s.ws) {
		retry(http.StatusServiceUnavailable, "", l.Text("prefs.request.unavailable", i18n.Args{"publisher": publisher}))
		return
	}
	id, found, err := h.Audience.FindPublicContact(r.Context(), h.Q, s.ws, email)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if found && h.Forms.Allow("prefs-contact:"+id.String(), 10*time.Minute) {
		if err := h.Forms.SendPreferenceLinkIn(r.Context(), s.ws, s.set, id, pl.Code); err != nil {
			h.log().WarnContext(r.Context(), "send preference link", "err", err)
		}
	}
	sent()
}

type reasonView struct{ Value, Label string }

// unsubscribeReasons are the answers to "Why are you leaving?", stored in
// unsubscribe_feedback (the report shows them per campaign).
var unsubscribeReasons = []reasonView{
	{"too_frequent", "Too many emails"},
	{"not_relevant", "Content isn't relevant to me"},
	{"never_signed_up", "I didn't sign up for this"},
	{"switching", "I'm switching email addresses"},
	{"other", "Other"},
}

type unsubscribeBody struct {
	Email, FirstName, OnLists, Action, PrefsHref, NotYouHref, Alert string
	Fewer                                                           bool
	FewerTitle, FewerText, FewerButton                              string
	Reasons                                                         []reasonView
	// Paid tells a paid member that unsubscribing doesn't cancel (Earn).
	Paid *earn.UnsubscribePaid
}

type doneBody struct {
	Heading, Before, Email, After, Status, Action, PrefsHref, ArchiveHref, Proof string
	Resubscribe                                                                  bool
	// Lead is the lead paragraph as HTML, in place of Before, Email and After.
	Lead template.HTML
}

// fieldProblem is a preference change's problem in the page's language:
// every code audience.FieldError names, as the signup forms write them
// (forms' fieldErrorText).
func fieldProblem(fe *audience.FieldError, l pagelabels.Set) string {
	switch fe.Code {
	case "frequency":
		return l.Text("error.frequency", nil)
	case "language":
		return l.Text("error.language", nil)
	case "email":
		return l.Text("error.emailInvalid", nil)
	case "pick_list":
		return l.Text("error.pickList", nil)
	case "list_gone":
		return l.Text("error.listGone", nil)
	case "number", "date", "boolean":
		return l.Text("error."+fe.Code, nil)
	case "select":
		return l.Text("error.select", i18n.Args{"options": strings.Join(fe.Options, ", ")})
	case "too_long":
		return l.Text("error.tooLong", nil)
	}
	return fe.Message
}

func (h *Handler) unsubscribePage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	if h.placementCopy(r, s, token, linktoken.Unsubscribe) {
		h.placementNotice(w, r, s)
		return
	}
	c, ok := h.claims(s, token, linktoken.Unsubscribe)
	if !ok {
		h.notFound(w, r, s)
		return
	}
	// A transactional email's link turns off just its category.
	if h.categoryUnsubscribe(w, r, s, c) {
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
	pl := h.contactLang(w, r, s, pc.Language)
	if pc.Status == "unsubscribed" {
		h.renderUnsubscribed(w, r, s, pc, c, token, "", pl)
		return
	}
	body, err := h.unsubscribeBody(r.Context(), s, pc, c, token, pl)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	v := h.view(s, pageTitle(pl, s, "unsub.title"), body)
	h.setLang(v, r, s, pl, nil)
	h.render(w, r, s, "unsubscribe", http.StatusOK, v, csp{}, true)
}

func (h *Handler) unsubscribeBody(ctx context.Context, s *site, pc audience.PublicContact, c linktoken.Claims, token string, pl pageLang) (unsubscribeBody, error) {
	l := pl.Labels
	cp := h.copyOf(ctx, s)
	p := s.set.Pages.Unsubscribe
	if cp != nil {
		p = cp.Pages(pl.Code, s.set.Pages).Unsubscribe
	}
	b := unsubscribeBody{Email: pc.Email, FirstName: pc.FirstName, Action: s.links.Unsubscribe(token), NotYouHref: s.links.Subscribe(""),
		PrefsHref: s.links.Preferences(h.sign(s, linktoken.Preferences, pc.ID, c.MessageID)),
		Fewer:     p.Fewer && pc.Frequency != "monthly", FewerTitle: p.FewerTitle, FewerText: p.FewerText, FewerButton: p.FewerButton}
	if p.Reasons {
		for _, rv := range unsubscribeReasons {
			b.Reasons = append(b.Reasons, reasonView{Value: rv.Value, Label: l.Text("unsub.reason."+rv.Value, nil)})
		}
	}
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return b, err
	}
	var on []string
	for _, pl2 := range lists {
		if pc.Subscribed(pl2.ID) {
			name := pl2.Name
			if cp != nil {
				name = cp.ListName(pl.Code, pl2)
			}
			on = append(on, name)
		}
	}
	if len(on) == 0 {
		b.OnLists = l.Text("unsub.onNone", nil)
	} else {
		b.OnLists = l.Text("unsub.on", i18n.Args{"lists": l.Join(on)})
	}
	if h.Earn != nil {
		var err error
		if b.Paid, err = h.Earn.UnsubscribePaid(ctx, s.ws, pc.ID, l.Lang()); err != nil {
			return b, err
		}
	}
	return b, nil
}

func (h *Handler) renderUnsubscribed(w http.ResponseWriter, r *http.Request, s *site, pc audience.PublicContact, c linktoken.Claims, token, status string, pl pageLang) {
	publisher := s.set.Branding.PublisherName
	l := pl.Labels
	b := doneBody{Heading: l.Text("unsub.done.heading", nil), Email: pc.Email,
		Lead:   labelHTML(l, "unsub.done.text", i18n.Args{"email": pc.Email, "publisher": publisher}, ""),
		Status: status, Action: s.links.Unsubscribe(token), Resubscribe: true}
	if s.set.Pages.Archive.Enabled {
		b.ArchiveHref = s.links.Archive()
	}
	b.Proof = l.Text("unsub.done.proof", nil)
	v := h.view(s, pageTitle(pl, s, "unsub.done.title"), b)
	h.setLang(v, r, s, pl, nil)
	h.render(w, r, s, "done", http.StatusOK, v, csp{}, true)
}

// unsubscribePost is the page's buttons and RFC 8058 one-click: mail apps
// POST "List-Unsubscribe=One-Click" to the URL in List-Unsubscribe, without
// cookies, JavaScript or a CSRF token; the signed token authenticates it.
func (h *Handler) unsubscribePost(w http.ResponseWriter, r *http.Request) {
	s, err := h.site(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	token := chi.URLParam(r, "token")
	if h.placementCopy(r, s, token, linktoken.Unsubscribe) {
		if h.parseForm(w, r, 16<<10) && r.PostForm.Get("List-Unsubscribe") == "One-Click" {
			h.placementOneClick(w, r, s)
			return
		}
		h.placementNotice(w, r, s)
		return
	}
	c, ok := h.claims(s, token, linktoken.Unsubscribe)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !h.parseForm(w, r, 16<<10) {
		http.Error(w, h.visitorLang(nil, r, s, nil).Labels.Text("unsub.unreadable", nil), http.StatusBadRequest)
		return
	}
	if h.categoryPost(w, r, s, c) {
		return
	}
	f := r.PostForm
	if f.Get("List-Unsubscribe") == "One-Click" {
		_, err := h.unsubscribe(r.Context(), s, c, h.evidence(r, audience.SourceOneClick), "one_click", "")
		if err != nil && !errors.Is(err, audience.ErrContactGone) {
			h.log().ErrorContext(r.Context(), "one-click unsubscribe", "err", err)
			http.Error(w, h.visitorLang(nil, r, s, nil).Labels.Text("unsub.failed", nil), http.StatusInternalServerError)
			return
		}
		oneClickDone(w, h.visitorLang(nil, r, s, nil))
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
	pl := h.contactLang(w, r, s, pc.Language)
	l := pl.Labels
	switch f.Get("action") {
	case "fewer":
		if err := h.chooseFewer(r, s, c); err != nil {
			h.serverError(w, r, s, err)
			return
		}
		publisher := s.set.Branding.PublisherName
		b := doneBody{Heading: l.Text("unsub.fewer.heading", nil), Email: pc.Email, Lead: labelHTML(l, "unsub.fewer.text", i18n.Args{"email": pc.Email, "publisher": publisher}, ""),
			PrefsHref: s.links.Preferences(h.sign(s, linktoken.Preferences, pc.ID, c.MessageID))}
		if s.set.Pages.Archive.Enabled {
			b.ArchiveHref = s.links.Archive()
		}
		v := h.view(s, pageTitle(pl, s, "unsub.fewer.title"), b)
		h.setLang(v, r, s, pl, nil)
		h.render(w, r, s, "done", http.StatusOK, v, csp{}, true)
	case "resubscribe":
		var back []uuid.UUID
		err := db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
			var err error
			back, err = h.Audience.Resubscribe(r.Context(), tx, s.ws, c.ContactID, h.evidence(r, audience.SourceUnsubscribePage))
			return err
		})
		if err != nil {
			h.serverError(w, r, s, err)
			return
		}
		publisher := s.set.Branding.PublisherName
		key := "unsub.back.text"
		if len(back) == 0 {
			key = "unsub.back.textPick"
		}
		b := doneBody{Heading: l.Text("unsub.back.heading", nil), Email: pc.Email, Lead: labelHTML(l, key, i18n.Args{"email": pc.Email, "publisher": publisher}, ""),
			PrefsHref: s.links.Preferences(h.sign(s, linktoken.Preferences, pc.ID, c.MessageID))}
		if s.set.Pages.Archive.Enabled {
			b.ArchiveHref = s.links.Archive()
		}
		v := h.view(s, pageTitle(pl, s, "unsub.back.title"), b)
		h.setLang(v, r, s, pl, nil)
		h.render(w, r, s, "done", http.StatusOK, v, csp{}, true)
	default:
		reason, comment := "", ""
		switch v := f.Get("reason"); v {
		case "too_frequent", "not_relevant", "never_signed_up", "other":
			reason = v
		case "switching":
			reason, comment = "other", "switching email addresses"
		}
		if _, err := h.unsubscribe(r.Context(), s, c, h.evidence(r, audience.SourceUnsubscribePage), reason, comment); err != nil {
			if errors.Is(err, audience.ErrContactGone) {
				h.notFound(w, r, s)
				return
			}
			h.serverError(w, r, s, err)
			return
		}
		http.Redirect(w, r, s.links.Unsubscribe(token), http.StatusSeeOther) //nolint:gosec // our own origin; the token was verified and is path-escaped
	}
}

// unsubscribe removes the contact from every list and records what the
// email it came from needs: the campaign's unsubscribe count, the event on
// the contact's timeline and, when given, the reason.
func (h *Handler) unsubscribe(ctx context.Context, s *site, c linktoken.Claims, ev audience.PublicEvidence, reason, comment string) (audience.UnsubscribeResult, error) {
	var res audience.UnsubscribeResult
	err := db.Tx(ctx, h.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var msg *store.PublicMessageRow
		var campaign *uuid.UUID
		if c.MessageID != uuid.Nil {
			if m, err := q.PublicMessage(ctx, c.MessageID); err == nil && m.WorkspaceID == s.ws.ID {
				msg, campaign = &m, m.CampaignID
			}
		}
		var err error
		res, err = h.Audience.Unsubscribe(ctx, tx, s.ws, c.ContactID, campaign, ev)
		if err != nil || res.Already {
			return err
		}
		at := ev.At
		if msg != nil {
			// The message moves to unsubscribed once; its campaign's count
			// follows (internal/counters).
			if err := counters.Unsubscribed(ctx, q, msg.ID, at); err != nil {
				return err
			}
			source := "Unsubscribe page"
			if ev.Source == audience.SourceOneClick {
				source = "List-Unsubscribe one-click"
			}
			if err := q.PublicInsertUnsubscribeEvent(ctx, store.PublicInsertUnsubscribeEventParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: s.ws.ID,
				MessageID: &msg.ID, CampaignID: msg.CampaignID, ContactID: &c.ContactID, ProviderID: msg.ProviderID, At: at, MailboxProvider: msg.MailboxProvider,
				OriginalType: source, UserAgent: truncate(ev.UserAgent, 500), Ip: ev.IP, Recipient: msg.Email, Source: source}); err != nil {
				return err
			}
		}
		if reason != "" {
			return q.PublicInsertFeedback(ctx, store.PublicInsertFeedbackParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: s.ws.ID, ContactID: &c.ContactID,
				CampaignID: campaign, Reason: reason, Comment: comment, At: at})
		}
		return nil
	})
	return res, err
}

// chooseFewer is the unsubscribe page's offer: the monthly best-of only.
func (h *Handler) chooseFewer(r *http.Request, s *site, c linktoken.Claims) error {
	ctx := r.Context()
	return db.Tx(ctx, h.Pool, func(tx pgx.Tx) error {
		if _, err := h.Audience.SavePreferences(ctx, tx, s.ws, c.ContactID, audience.PreferencesInput{Frequency: "monthly", PauseDays: -1},
			h.evidence(r, audience.SourceUnsubscribePage)); err != nil {
			return err
		}
		q := store.New(tx)
		var campaign *uuid.UUID
		if c.MessageID != uuid.Nil {
			if m, err := q.PublicMessage(ctx, c.MessageID); err == nil && m.WorkspaceID == s.ws.ID {
				campaign = m.CampaignID
			}
		}
		return q.PublicInsertFeedback(ctx, store.PublicInsertFeedbackParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: s.ws.ID, ContactID: &c.ContactID,
			CampaignID: campaign, Reason: "too_frequent", ChoseFewer: true, At: h.now()})
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (h *Handler) viewOnline(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	c, ok := h.claims(s, token, linktoken.ViewOnline)
	if !ok || c.MessageID == uuid.Nil {
		if h.placementCopy(r, s, token, linktoken.ViewOnline) {
			h.placementNotice(w, r, s)
			return
		}
		h.notFound(w, r, s)
		return
	}
	m, err := h.Q.PublicMessage(r.Context(), c.MessageID)
	if err != nil || m.WorkspaceID != s.ws.ID || m.CampaignID == nil {
		h.notFound(w, r, s)
		return
	}
	cp, err := h.Q.PublicCampaign(r.Context(), store.PublicCampaignParams{WorkspaceID: s.ws.ID, ID: *m.CampaignID})
	if err != nil || strings.TrimSpace(cp.Html) == "" {
		h.notFound(w, r, s)
		return
	}
	pc, err := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID)
	if err != nil {
		h.notFound(w, r, s)
		return
	}
	links := map[string]string{
		"unsubscribe_url":     s.links.Unsubscribe(h.sign(s, linktoken.Unsubscribe, pc.ID, m.ID)),
		"preferences_url":     s.links.Preferences(h.sign(s, linktoken.Preferences, pc.ID, m.ID)),
		"view_in_browser_url": s.links.View(token),
	}
	subj, html := variantContent(cp.Subject, cp.Html, cp.AbTest, m.Variant)
	// The version the message delivered: its language's, when it wasn't
	// the campaign's own.
	lang := ""
	if mv, err := h.Q.PublicMessageVersion(r.Context(), store.PublicMessageVersionParams{ID: m.ID, WorkspaceID: s.ws.ID}); err == nil {
		lang = mv.Language
		if lang == "" {
			lang = mv.Source
		}
		if mv.Translated && strings.TrimSpace(mv.Html) != "" {
			subj, html = variantContent(mv.Subject, mv.Html, mv.AbTest, m.Variant)
		}
	}
	// The reader's version of conditional content; poll links answer as
	// someone who isn't a contact (design expansion, X11).
	if h.Browser != nil {
		html = h.Browser.BrowserHTML(r.Context(), s.ws, cp.ID, m.ID, html)
		for k, v := range h.Browser.BrowserLinks(r.Context(), s.ws, cp.ID) {
			links[k] = v
		}
	}
	vars := liquid.Vars(pc.Email, pc.FirstName, pc.LastName, pc.Attributes, links)
	if h.Recipients != nil {
		// Grow's blocks show the reader's own link and progress.
		vars = liquid.Fill(r.Context(), h.Recipients(), s.ws.ID, liquid.Recipient{ContactID: pc.ID, MessageID: m.ID}, liquid.ExpandBlocks(subj+"\n"+html), vars)
	}
	subject, body := renderEmailVars(subj, html, vars, h.URLs.Base())
	doc := setDocLang(injectHead(body, emailHead{Title: subject, Robots: "noindex, nofollow"}), lang)
	emailHeaders(w, true)
	_, _ = w.Write([]byte(doc))
}
