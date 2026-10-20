package publicpages

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	pagesvc "github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// A published page at its address (/{slug}): its blocks on the site's
// theme, for whoever may read it. Members and Paid pages show a sign-in
// or upgrade card to the others. A page that moved redirects; one taken
// off answers 410, or sends readers where its publisher chose.

// pageBody is a custom page.
type pageBody struct {
	HTML   template.HTML
	Notice template.HTML
	BuyBar *buyBarView
}

// gateBody is a Members or Paid page for someone who can't read it.
type gateBody struct {
	Heading, Text string
	Actions       []linkAction
}

// TokenEvent signs a registrant's calendar file (the joining link is in it).
const TokenEvent linktoken.Kind = 194

// customPage is /{slug}.
func (h *Handler) customPage(w http.ResponseWriter, r *http.Request) {
	h.servePage(w, r, nil)
}

// servePage draws a page; failed is a signup that came back with errors.
func (h *Handler) servePage(w http.ResponseWriter, r *http.Request, failed *formFailure) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Pages == nil {
		h.siteNotFound(w, r, s)
		return
	}
	ctx := r.Context()
	slug := chi.URLParam(r, "slug")
	res, err := h.Pages.Resolve(ctx, s.ws, slug)
	switch {
	case err != nil:
		h.siteFail(w, r, s, err)
		return
	case res.MovedTo != "":
		to := s.path(res.MovedTo)
		if q := r.URL.RawQuery; q != "" {
			to += "?" + q
		}
		http.Redirect(w, r, to, http.StatusMovedPermanently) //nolint:gosec // G710: a path on this site, the page's own address
		return
	case res.RedirectTo != "":
		http.Redirect(w, r, s.sitePath(res.RedirectTo), http.StatusFound) //nolint:gosec // G710: a path on this site, chosen when the page was unpublished
		return
	case res.Gone:
		h.pageGone(w, r, s)
		return
	case res.Page == nil:
		h.siteNotFound(w, r, s)
		return
	}
	lp := res.Page
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	h.sellView(v, s)
	h.pageMeta(s, v, lp)
	v.Focus = lp.Settings.HideHeader
	v.mark("page:"+lp.Slug, v.Frame)
	// Who may read it.
	if lp.Settings.Visibility != pagesvc.VisibilityPublic {
		v.private = true
		if !h.canRead(ctx, s, v, lp) {
			h.renderGate(w, r, s, v, lp)
			return
		}
	}
	pr := &pageRender{h: h, ctx: ctx, s: s, v: v, pageID: lp.ID, pageURL: s.path(lp.Slug), source: "page", now: h.now(), failed: failed}
	body, err := pr.render(lp.HTML)
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	b := pageBody{HTML: body, BuyBar: pr.buyBar}
	if r.URL.Query().Get("checkout") == "cancelled" {
		b.Notice = v.SH("buy.cancelled")
		v.private = true
	}
	v.Body = b
	v.cspFrames = append(v.cspFrames, pr.frames...)
	status := http.StatusOK
	if failed != nil {
		status = http.StatusUnprocessableEntity
		v.private = true
	} else if r.Method == http.MethodGet && h.Website != nil {
		h.Website.CountView(ctx, s.ws, r, website.PagePage, lp.ID)
	}
	h.renderSite(w, r, s, "sell-page", status, v)
}

// pageMeta sets a page's title, description and social tags.
func (h *Handler) pageMeta(s *site, v *siteView, lp *pagesvc.LivePage) {
	title := lp.Name
	if t := strings.TrimSpace(lp.Settings.SEOTitle); t != "" {
		v.Title = t
	} else {
		v.Title = v.T("site.title", "page", title, "site", v.Frame.Title)
	}
	v.Description = lp.Settings.SEODescription
	v.Canonical = s.canonPath(lp.Slug)
	image := h.mediaURL(lp.Settings.SocialImage)
	v.OG = &og{Type: "website", SiteName: v.Frame.Title, Title: title, Description: v.Description, URL: v.Canonical, Image: image}
	if lp.Settings.Visibility != pagesvc.VisibilityPublic {
		v.Robots = "noindex, follow"
	}
}

// canRead decides whether the reader may read a Members or Paid page.
func (h *Handler) canRead(ctx context.Context, s *site, v *siteView, lp *pagesvc.LivePage) bool {
	if h.Website == nil {
		return false
	}
	reader, err := h.Website.Reader(ctx, s.ws, v.Session, "")
	if err != nil {
		return false
	}
	rules, _, err := h.Website.Rules(ctx, s.ws)
	if err != nil {
		return false
	}
	vis := gate.Visibility{Level: lp.Settings.Visibility}
	for _, t := range lp.Settings.Tiers {
		if id, err := uuid.Parse(t); err == nil {
			vis.Tiers = append(vis.Tiers, id)
		}
	}
	return gate.Page(vis, reader, rules).Readable()
}

// renderGate is the card a reader who can't read a page gets.
func (h *Handler) renderGate(w http.ResponseWriter, r *http.Request, s *site, v *siteView, lp *pagesvc.LivePage) {
	b := gateBody{Heading: v.S("gate.membersTitle"), Text: v.S("gate.membersText")}
	if lp.Settings.Visibility == pagesvc.VisibilityPaid {
		b.Heading, b.Text = v.S("gate.paidTitle"), v.S("gate.paidText")
		b.Actions = append(b.Actions, linkAction{Label: v.T("site.upgrade"), Href: s.path("upgrade"), Primary: true})
	}
	if v.Frame.SigninOn && v.Member == nil {
		b.Actions = append(b.Actions, linkAction{Label: v.T("site.signIn"), Href: h.signinHref(s, s.path(lp.Slug)), Primary: len(b.Actions) == 0})
	}
	if !v.Frame.SigninOn {
		b.Text = v.S("gate.signinOff")
	}
	v.Body = b
	v.Robots = "noindex, follow"
	h.renderSite(w, r, s, "sell-gate", http.StatusOK, v)
}

// pageGone is a page taken off the site: 410 with the latest posts in the
// site's frame when the website is on, a plain message page otherwise.
func (h *Handler) pageGone(w http.ResponseWriter, r *http.Request, s *site) {
	if h.Website != nil && h.siteOn(r.Context(), s) {
		v, ok := h.siteViewFor(w, r, s)
		if !ok {
			return
		}
		v.Title = v.T("site.title", "page", v.S("gone.title"), "site", v.Frame.Title)
		v.Robots = "noindex, follow"
		b, err := h.latestBody(r.Context(), s, v, "gone", v.S("gone.title"), v.S("gone.text"))
		if err != nil {
			h.siteFail(w, r, s, err)
			return
		}
		v.Body = b
		h.renderSite(w, r, s, "gone", http.StatusGone, v)
		return
	}
	pl := h.visitorLang(w, r, s, nil)
	l := i18n.For(i18n.English)
	if loc, ok := i18n.ParseLocale(pl.Code); ok {
		l = i18n.For(loc)
	}
	title := sellMsgs.M("gone.title").Text(l, nil)
	h.messageIn(w, r, s, pl, http.StatusGone, pl.Labels.Text("page.title", i18n.Args{"heading": title, "publisher": s.set.Branding.PublisherName}),
		messageBody{Heading: title, Text: sellMsgs.M("gone.textPlain").Text(l, nil)}, true)
}

// customForm is a signup through a page's form block: the form's own
// pipeline (consent, double opt-in, Turnstile), then the page's result. A
// webinar page answers with its registered page; a failed signup comes
// back to the page with the errors in place.
func (h *Handler) customForm(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Pages == nil {
		h.siteNotFound(w, r, s)
		return
	}
	if !h.parseForm(w, r, 64<<10) {
		http.Error(w, h.visitorLang(nil, r, s, nil).Labels.Text("page.unreadable", nil), http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	res, err := h.Pages.Resolve(ctx, s.ws, chi.URLParam(r, "slug"))
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	if res.Page == nil {
		h.siteNotFound(w, r, s)
		return
	}
	lp := res.Page
	formID, err := uuid.Parse(r.PostForm.Get("_form"))
	if err != nil || !pageHasForm(lp.HTML, formID) {
		// Only the forms the page shows take signups here.
		h.notFound(w, r, s)
		return
	}
	f, err := h.Q.FormGet(ctx, store.FormGetParams{WorkspaceID: s.ws.ID, ID: formID})
	if err != nil {
		h.notFound(w, r, s)
		return
	}
	c := forms.ConfigOf(f)
	sub := submissionFrom(r)
	if sub.Page == "" {
		sub.Page = s.path(lp.Slug)
	}
	out := h.Forms.Submit(ctx, s.ws, s.set, c, sub)
	switch out.Status {
	case forms.OutcomeConfirm, forms.OutcomeSubscribed:
		u := pagesvc.Analyze(lp.HTML)
		kind := "signup"
		if len(u.Events) > 0 {
			kind = "registration"
		}
		contact := h.contactOf(ctx, s, out.Email)
		if err := h.Pages.Conversion(ctx, s.ws, lp.ID, kind, contact); err != nil {
			h.log().WarnContext(ctx, "page conversion", "err", err)
		}
		// A webinar's registrants get its date in their date field, for
		// the reminder automation.
		if ev, ok := pagesvc.FirstEvent(lp.HTML); ok && ev.Field != "" && contact != nil {
			if err := h.Pages.RecordEventDate(ctx, s.ws, *contact, ev); err != nil {
				h.log().WarnContext(ctx, "event date field", "err", err)
			}
		}
		if kind == "registration" && out.RedirectURL == "" {
			h.registered(w, r, s, lp, out, contact)
			return
		}
		h.subscribeResult(w, r, s, c, false, sub, out)
	case forms.OutcomeInvalid, forms.OutcomeError:
		r.Method = http.MethodGet
		h.servePage(w, r, &formFailure{block: r.PostForm.Get("_block"), values: sub, out: out})
	default:
		h.subscribeResult(w, r, s, c, false, sub, out)
	}
}

// pageHasForm reports whether a page shows a form.
func pageHasForm(src string, id uuid.UUID) bool {
	for _, f := range pagesvc.FormsIn(src) {
		if f == id {
			return true
		}
	}
	return false
}

// contactOf is the contact an address belongs to (nil: none yet).
func (h *Handler) contactOf(ctx context.Context, s *site, email string) *uuid.UUID {
	if email == "" {
		return nil
	}
	var id uuid.UUID
	if err := h.Pool.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND lower(email) = lower($2) AND erased_at IS NULL`, s.ws.ID, email).Scan(&id); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			h.log().WarnContext(ctx, "page signup contact", "err", err)
		}
		return nil
	}
	return &id
}

// registeredBody is a webinar's "You're registered".
type registeredBody struct {
	Title, When, Emailed, CalendarNote string
	// Link is the joining link, shown to whoever registered.
	Link                 string
	Google, Outlook, ICS string
}

// registered answers a webinar registration: the event in the reader's
// time, the joining link, the reminders when a live automation sends them,
// and Add to calendar.
func (h *Handler) registered(w http.ResponseWriter, r *http.Request, s *site, lp *pagesvc.LivePage, out forms.Outcome, contact *uuid.UUID) {
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	h.sellView(v, s)
	noStore(w, v)
	v.Focus = lp.Settings.HideHeader
	v.Title = v.T("site.title", "page", v.S("event.registered"), "site", v.Frame.Title)
	v.Robots = "noindex, nofollow"
	pr := &pageRender{h: h, ctx: r.Context(), s: s, v: v, pageID: lp.ID, pageURL: s.path(lp.Slug), source: "page", now: h.now()}
	if _, err := pr.render(lp.HTML); err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	ev := pr.event
	b := registeredBody{Title: lp.Name}
	if ev != nil {
		if ev.Title != "" {
			b.Title = ev.Title
		}
		b.When = v.S("event.whenPlain", "date", v.longDate(ev.Start.In(ev.TZ)), "time", timeRange(ev.Start, ev.End, ev.TZ))
		b.Link = ev.Link
		if ev.Reminders != "" && h.remindersLive(r.Context(), s, lp.ID) {
			b.Emailed = v.S("event.remindersTo", "email", out.Email, "reminders", ev.Reminders)
		}
		details := lp.Name
		if ev.Link != "" {
			details += "\n" + ev.Link
		}
		b.Google = googleCalendar(b.Title, details, ev.Start, ev.End)
		b.Outlook = outlookCalendar(b.Title, details, ev.Start, ev.End)
		if contact != nil && h.Tokens != nil {
			t := h.Tokens.Sign(linktoken.Claims{Kind: TokenEvent, WorkspaceID: s.ws.ID, ContactID: *contact, MessageID: lp.ID})
			b.ICS = s.path(lp.Slug, "event.ics") + "?t=" + url.QueryEscape(t)
		}
		b.CalendarNote = v.S("event.calendarNote")
	}
	v.Body = b
	h.renderSite(w, r, s, "sell-registered", http.StatusOK, v)
}

// remindersLive reports whether a live automation sends the session's
// reminders (the page's reminder automation).
func (h *Handler) remindersLive(ctx context.Context, s *site, pageID uuid.UUID) bool {
	if h.Pages == nil {
		return false
	}
	rem, err := h.Pages.Reminders(ctx, s.ws, pageID)
	return err == nil && rem.Flow != nil && rem.Flow.Status == "live"
}

func calStamp(t time.Time) string { return t.UTC().Format("20060102T150405Z") }

func googleCalendar(title, details string, start, end time.Time) string {
	q := url.Values{"action": {"TEMPLATE"}, "text": {title}, "dates": {calStamp(start) + "/" + calStamp(end)}, "details": {details}}
	return "https://calendar.google.com/calendar/render?" + q.Encode()
}

func outlookCalendar(title, details string, start, end time.Time) string {
	q := url.Values{"path": {"/calendar/action/compose"}, "rru": {"addevent"}, "subject": {title}, "startdt": {start.UTC().Format(time.RFC3339)},
		"enddt": {end.UTC().Format(time.RFC3339)}, "body": {details}}
	return "https://outlook.live.com/calendar/0/deeplink/compose?" + q.Encode()
}

// eventICS is a registrant's calendar file (Apple Calendar), signed: it
// holds the joining link.
func (h *Handler) eventICS(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Pages == nil || h.Tokens == nil {
		h.notFound(w, r, s)
		return
	}
	c, err := h.Tokens.Verify(r.URL.Query().Get("t"), TokenEvent)
	if err != nil || c.WorkspaceID != s.ws.ID {
		h.notFound(w, r, s)
		return
	}
	res, err := h.Pages.Resolve(r.Context(), s.ws, chi.URLParam(r, "slug"))
	if err != nil || res.Page == nil || res.Page.ID != c.MessageID {
		h.notFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	pr := &pageRender{h: h, ctx: r.Context(), s: s, v: v, pageID: res.Page.ID, pageURL: s.path(res.Page.Slug), source: "page", now: h.now()}
	if _, err := pr.render(res.Page.HTML); err != nil || pr.event == nil {
		h.notFound(w, r, s)
		return
	}
	ev := pr.event
	title := ev.Title
	if title == "" {
		title = res.Page.Name
	}
	desc := res.Page.Name
	if ev.Link != "" {
		desc += "\n" + ev.Link
	}
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//ReputeCast//Pages//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\nBEGIN:VEVENT\r\n")
	fmt.Fprintf(&b, "UID:%s-%s@reputecast\r\nDTSTAMP:%s\r\nDTSTART:%s\r\nDTEND:%s\r\n", res.Page.ID, c.ContactID, calStamp(h.now()), calStamp(ev.Start), calStamp(ev.End))
	fmt.Fprintf(&b, "SUMMARY:%s\r\nDESCRIPTION:%s\r\n", icsText(title), icsText(desc))
	if ev.Link != "" {
		fmt.Fprintf(&b, "URL:%s\r\n", icsText(ev.Link))
	}
	if ev.Where != "" {
		fmt.Fprintf(&b, "LOCATION:%s\r\n", icsText(ev.Where))
	}
	b.WriteString("END:VEVENT\r\nEND:VCALENDAR\r\n")
	hd := w.Header()
	hd.Set("Content-Type", "text/calendar; charset=utf-8")
	hd.Set("Content-Disposition", `attachment; filename="`+res.Page.Slug+`.ics"`)
	hd.Set("Cache-Control", "private, no-store")
	hd.Set("X-Robots-Tag", "noindex, nofollow")
	_, _ = w.Write([]byte(b.String()))
}

// icsText escapes a calendar value.
func icsText(s string) string {
	r := strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\r\n", "\\n", "\n", "\\n")
	return r.Replace(s)
}
