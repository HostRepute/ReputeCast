package publicpages

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

type gotView struct {
	Name, Text, AddHref, Offer string
	On                         bool
}

type confirmBody struct {
	Heading, Email, Sender, LatestHref, PrefsHref, Proof string
	Lists                                                []gotView
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
	if res.Blocked {
		h.message(w, r, s, http.StatusOK, "Can't subscribe · "+s.set.Branding.PublisherName, messageBody{Heading: "This address can't be subscribed",
			Text: "Emails to it bounced or were reported as spam, so " + s.set.Branding.PublisherName + " stopped sending to it. If that was a mistake, reply to any of their emails."}, true)
		return
	}
	pc, err := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	body, err := h.confirmBody(r.Context(), s, pc)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	v := h.view(s, "You're in · "+s.set.Branding.PublisherName, body)
	h.render(w, r, s, "confirm", http.StatusOK, v, csp{}, true)
}

func (h *Handler) confirmBody(ctx context.Context, s *site, pc audience.PublicContact) (confirmBody, error) {
	b := confirmBody{Heading: "You're in", Email: pc.Email, Lists: []gotView{}}
	if pc.FirstName != "" {
		b.Heading = "You're in, " + pc.FirstName
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
	for _, l := range lists {
		on := pc.Subscribed(l.ID)
		if !on && !slices.Contains(offered, l.ID) {
			continue
		}
		text := l.Description
		if t := s.set.Pages.Confirm.ListText[l.ID.String()]; t != "" {
			text = t
		}
		if at, ok := next[l.ID]; ok && on {
			text = strings.TrimSpace(text + " Next issue: " + at.In(location(s.ws)).Format("Monday 2 January") + ".")
		}
		offer := s.set.Pages.Confirm.OfferText[l.ID.String()]
		if offer == "" {
			offer = "it"
		}
		b.Lists = append(b.Lists, gotView{Name: l.Name, Text: text, On: on, AddHref: prefs, Offer: offer})
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
		b.Proof = "Confirmed " + pc.Confirmed.At.In(location(s.ws)).Format("2 Jan 2006, 15:04")
		if pc.Confirmed.IP != nil {
			b.Proof += " from " + pc.Confirmed.IP.String()
		}
		b.Proof += " · kept as proof of consent"
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
}

type preferencesBody struct {
	Email, FirstName, Action, NotYouHref, UnsubscribeHref, Shown, Saved, Alert string
	Lists                                                                      []listView
	Frequencies                                                                []choiceView
	Pauses                                                                     []choiceView
	AskName                                                                    bool
}

func (h *Handler) preferencesPage(w http.ResponseWriter, r *http.Request) {
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
	pc, err := h.Audience.LoadPublicContact(r.Context(), h.Q, s.ws, c.ContactID)
	if errors.Is(err, audience.ErrContactGone) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	body, err := h.preferencesBody(r.Context(), s, pc, s.links.Preferences(token), c)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if r.URL.Query().Get("saved") == "1" {
		body.Saved = "Saved. Changes apply to the next send."
	}
	v := h.view(s, "Email preferences · "+s.set.Branding.PublisherName, body)
	h.render(w, r, s, "preferences", http.StatusOK, v, csp{}, true)
}

func (h *Handler) preferencesBody(ctx context.Context, s *site, pc audience.PublicContact, action string, c linktoken.Claims) (preferencesBody, error) {
	p := s.set.Pages.Preferences
	b := preferencesBody{Email: pc.Email, FirstName: pc.FirstName, Action: action, NotYouHref: s.links.Subscribe(""), AskName: p.FirstName,
		UnsubscribeHref: s.links.Unsubscribe(h.sign(s, linktoken.Unsubscribe, pc.ID, c.MessageID))}
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return b, err
	}
	shown := []string{}
	for _, l := range lists {
		offered := l.Public
		if len(p.Lists) > 0 {
			offered = slices.Contains(p.Lists, l.ID.String())
		}
		st := pc.Lists[l.ID]
		if !offered && st != "subscribed" && st != "pending" {
			continue
		}
		text := l.Description
		if t := p.ListText[l.ID.String()]; t != "" {
			text = t
		}
		b.Lists = append(b.Lists, listView{ID: l.ID.String(), Name: l.Name, Text: text, Checked: (st == "subscribed" && pc.Status == "active") || st == "pending"})
		shown = append(shown, l.ID.String())
	}
	b.Shown = strings.Join(shown, ",")
	if p.Frequency {
		for _, f := range []struct{ v, l string }{{"all", "Every issue"}, {"weekly", "Weekly roundup"}, {"monthly", "Monthly best-of"}} {
			b.Frequencies = append(b.Frequencies, choiceView{Value: f.v, Label: f.l, Checked: pc.Frequency == f.v})
		}
	}
	if p.Pause {
		if pc.PausedUntil != nil && pc.PausedUntil.After(h.now()) {
			b.Pauses = []choiceView{
				{Value: "keep", Label: "Paused until " + pc.PausedUntil.In(location(s.ws)).Format("2 Jan"), Checked: true},
				{Value: "0", Label: "Resume now"}, {Value: "30", Label: "Pause 30 days"}, {Value: "90", Label: "Pause 90 days"},
			}
		} else {
			b.Pauses = []choiceView{{Value: "0", Label: "Don't pause", Checked: true}, {Value: "30", Label: "Pause 30 days"}, {Value: "90", Label: "Pause 90 days"}}
		}
	}
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
		http.Error(w, "That form couldn't be read. Go back and send it again.", http.StatusBadRequest)
		return
	}
	f := r.PostForm
	in := audience.PreferencesInput{Frequency: f.Get("frequency"), PauseDays: -1}
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
	err := db.Tx(r.Context(), h.Pool, func(tx pgx.Tx) error {
		_, err := h.Audience.SavePreferences(r.Context(), tx, s.ws, c.ContactID, in, h.evidence(r, audience.SourcePreferences))
		return err
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
		body, berr := h.preferencesBody(r.Context(), s, pc, s.links.Preferences(token), c)
		if berr != nil {
			h.serverError(w, r, s, berr)
			return
		}
		body.Alert = fe.Message
		h.render(w, r, s, "preferences", http.StatusUnprocessableEntity, h.view(s, "Email preferences · "+s.set.Branding.PublisherName, body), csp{}, true)
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
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
	v := h.view(s, "Email preferences · "+s.set.Branding.PublisherName, body)
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
		http.Error(w, "That form couldn't be read. Go back and send it again.", http.StatusBadRequest)
		return
	}
	f := r.PostForm
	email := strings.TrimSpace(f.Get("email"))
	ip := "unknown"
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	publisher := s.set.Branding.PublisherName
	retry := func(status int, emailErr, alert string) {
		body := requestBody{Action: s.links.PreferencesRequest(), Token: h.Forms.FormToken("prefs:"+s.ws.ID.String(), h.now()), Email: email, EmailError: emailErr, Alert: alert}
		h.render(w, r, s, "request", status, h.view(s, "Email preferences · "+publisher, body), csp{}, true)
	}
	if !h.Forms.AllowPerMinute("prefs-ip:"+ip, 3) {
		w.Header().Set("Retry-After", "60")
		retry(http.StatusTooManyRequests, "", "Too many tries from your network. Wait a minute and try again.")
		return
	}
	if email == "" || !strings.Contains(email, "@") {
		retry(http.StatusUnprocessableEntity, "Enter the address you get our emails at.", "")
		return
	}
	sent := func() {
		h.message(w, r, s, http.StatusOK, "Check your inbox · "+publisher, messageBody{Icon: "mail", Heading: "Check your inbox",
			Text: "If", Email: email, TextAfter: " gets emails from " + publisher + ", a link to your preferences is on its way. It works for this address only."}, true)
	}
	// Bots fill the invisible field; they see the same answer and nothing is sent.
	if f.Get("website") != "" {
		sent()
		return
	}
	// Whether mail can go out is the same for every address, so saying so
	// reveals nothing; after that, every address gets the same answer.
	if !h.Forms.SystemMailAvailable(r.Context(), s.ws) {
		retry(http.StatusServiceUnavailable, "", publisher+" can't send emails right now. Use the preferences link at the bottom of any of their emails instead.")
		return
	}
	id, found, err := h.Audience.FindPublicContact(r.Context(), h.Q, s.ws, email)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if found && h.Forms.Allow("prefs-contact:"+id.String(), 10*time.Minute) {
		if err := h.Forms.SendPreferenceLink(r.Context(), s.ws, s.set, id); err != nil {
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
}

type doneBody struct {
	Heading, Before, Email, After, Status, Action, PrefsHref, ArchiveHref, Proof string
	Resubscribe                                                                  bool
}

func listNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

func (h *Handler) unsubscribePage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	c, ok := h.claims(s, token, linktoken.Unsubscribe)
	if !ok {
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
	if pc.Status == "unsubscribed" {
		h.renderUnsubscribed(w, r, s, pc, c, token, "")
		return
	}
	body, err := h.unsubscribeBody(r.Context(), s, pc, c, token)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	v := h.view(s, "Unsubscribe · "+s.set.Branding.PublisherName, body)
	h.render(w, r, s, "unsubscribe", http.StatusOK, v, csp{}, true)
}

func (h *Handler) unsubscribeBody(ctx context.Context, s *site, pc audience.PublicContact, c linktoken.Claims, token string) (unsubscribeBody, error) {
	p := s.set.Pages.Unsubscribe
	b := unsubscribeBody{Email: pc.Email, FirstName: pc.FirstName, Action: s.links.Unsubscribe(token), NotYouHref: s.links.Subscribe(""),
		PrefsHref: s.links.Preferences(h.sign(s, linktoken.Preferences, pc.ID, c.MessageID)),
		Fewer:     p.Fewer && pc.Frequency != "monthly", FewerTitle: p.FewerTitle, FewerText: p.FewerText, FewerButton: p.FewerButton}
	if p.Reasons {
		b.Reasons = unsubscribeReasons
	}
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return b, err
	}
	var on []string
	for _, l := range lists {
		if pc.Subscribed(l.ID) {
			on = append(on, l.Name)
		}
	}
	if len(on) == 0 {
		b.OnLists = "You're not on any list right now."
	} else {
		b.OnLists = "You're on " + listNames(on) + "."
	}
	return b, nil
}

func (h *Handler) renderUnsubscribed(w http.ResponseWriter, r *http.Request, s *site, pc audience.PublicContact, c linktoken.Claims, token, status string) {
	publisher := s.set.Branding.PublisherName
	b := doneBody{Heading: "You're unsubscribed", Email: pc.Email,
		After:  " won't get newsletters or automated emails from " + publisher + " any more. Account emails like password resets still arrive.",
		Status: status, Action: s.links.Unsubscribe(token), Resubscribe: true}
	if s.set.Pages.Archive.Enabled {
		b.ArchiveHref = s.links.Archive()
	}
	b.Proof = "Recorded so you don't get emails again, even after an import"
	h.render(w, r, s, "done", http.StatusOK, h.view(s, "Unsubscribed · "+publisher, b), csp{}, true)
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
	c, ok := h.claims(s, token, linktoken.Unsubscribe)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !h.parseForm(w, r, 16<<10) {
		http.Error(w, "The request couldn't be read.", http.StatusBadRequest)
		return
	}
	f := r.PostForm
	if f.Get("List-Unsubscribe") == "One-Click" {
		_, err := h.unsubscribe(r.Context(), s, c, h.evidence(r, audience.SourceOneClick), "one_click", "")
		if err != nil && !errors.Is(err, audience.ErrContactGone) {
			h.log().ErrorContext(r.Context(), "one-click unsubscribe", "err", err)
			http.Error(w, "Couldn't unsubscribe right now; try again in a minute.", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Unsubscribed.\n"))
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
	switch f.Get("action") {
	case "fewer":
		if err := h.chooseFewer(r, s, c); err != nil {
			h.serverError(w, r, s, err)
			return
		}
		publisher := s.set.Branding.PublisherName
		b := doneBody{Heading: "Done: one email a month", Email: pc.Email, After: " gets the monthly best-of from " + publisher + " and nothing else. Change it any time in your preferences.",
			PrefsHref: s.links.Preferences(h.sign(s, linktoken.Preferences, pc.ID, c.MessageID))}
		if s.set.Pages.Archive.Enabled {
			b.ArchiveHref = s.links.Archive()
		}
		h.render(w, r, s, "done", http.StatusOK, h.view(s, "Fewer emails · "+publisher, b), csp{}, true)
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
		b := doneBody{Heading: "Welcome back", Email: pc.Email, After: " is subscribed to " + publisher + " again.",
			PrefsHref: s.links.Preferences(h.sign(s, linktoken.Preferences, pc.ID, c.MessageID))}
		if len(back) == 0 {
			b.After = " is subscribed to " + publisher + " again. Pick your lists in your preferences."
		}
		if s.set.Pages.Archive.Enabled {
			b.ArchiveHref = s.links.Archive()
		}
		h.render(w, r, s, "done", http.StatusOK, h.view(s, "Welcome back · "+publisher, b), csp{}, true)
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
			if msg.CampaignID != nil && msg.ProviderID != nil {
				if err := q.PublicBumpUnsubscribed(ctx, store.PublicBumpUnsubscribedParams{CampaignID: *msg.CampaignID, ProviderID: *msg.ProviderID,
					MailboxProvider: msg.MailboxProvider, Variant: msg.Variant}); err != nil {
					return err
				}
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
	subject, body := renderEmail(subj, html, pc.Email, pc.FirstName, pc.LastName, pc.Attributes, links, h.URLs.Base())
	doc := injectHead(body, emailHead{Title: subject, Robots: "noindex, nofollow"})
	emailHeaders(w, true)
	_, _ = w.Write([]byte(doc))
}
