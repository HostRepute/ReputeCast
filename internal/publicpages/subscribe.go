package publicpages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// fieldView is one input of a public form.
type fieldView struct {
	Key, Name, ID, Label, Type, Value, Placeholder, Autocomplete string
	Required, Optional                                           bool
	Options                                                      []string
	Error                                                        string
}

type listView struct {
	ID, Name, Text string
	Checked        bool
}

// consentParts is the consent sentence with its words for the privacy
// policy linked.
type consentParts struct {
	Text, Before, Link, After, Href string
}

// splitConsent links phrase (the language's words for the privacy policy,
// as the consent text writes them) to privacy.
func splitConsent(text, privacy, phrase string) consentParts {
	c := consentParts{Text: text, Before: text}
	if privacy == "" {
		return c
	}
	if i, j := forms.PhraseIndex(text, phrase); i >= 0 {
		c.Before, c.Link, c.After, c.Href = text[:i], text[i:j], text[j:], privacy
	}
	return c
}

type subscribeBody struct {
	Heading, Description, Footnote, Button, ListsLabel string
	Action, Token                                      string
	Fields                                             []fieldView
	Lists                                              []listView
	ListError                                          string
	Consent                                            consentParts
	ConsentChecked                                     bool
	ConsentError                                       string
	Honeypot                                           bool
	Alert                                              string
	ArchiveURL                                         string
}

var errNoForm = errors.New("publicpages: no such form")

// resolveForm finds the form of a hosted page: /subscribe uses the
// workspace's hosted page form (or the built-in one), /subscribe/{slug} a
// form by its slug. Drafts aren't public.
func (h *Handler) resolveForm(ctx context.Context, s *site, slug string, preview bool) (forms.Config, bool, error) {
	if slug == "" {
		if id := s.set.HostedFormID(); id != nil {
			f, err := h.Q.FormGet(ctx, store.FormGetParams{WorkspaceID: s.ws.ID, ID: *id})
			if err == nil && (f.Status != forms.StatusDraft || preview) {
				return forms.ConfigOf(f), true, nil
			}
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return forms.Config{}, true, err
			}
		}
		lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
		if err != nil {
			return forms.Config{}, true, err
		}
		return forms.BuiltinForm(s.ws, s.set.Branding.PublisherName, lists), true, nil
	}
	f, err := h.Q.FormBySlug(ctx, store.FormBySlugParams{WorkspaceID: s.ws.ID, Slug: slug})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && f.Status == forms.StatusDraft && !preview) {
		return forms.Config{}, false, errNoForm
	}
	if err != nil {
		return forms.Config{}, false, err
	}
	hosted := s.set.HostedFormID() != nil && *s.set.HostedFormID() == f.ID
	return forms.ConfigOf(f), hosted, nil
}

// formLang is the hosted page of a form in a language: the form there,
// and whether a language can be shown (the pages and the form are Live).
type formLang struct {
	pageLang
	form forms.Config
	live func(string) bool
}

// subscribeLang picks the language of a form's hosted page: the one the
// form always shows, else the visitor's (the menu, then the browser, then
// the default), among those the pages and the form are Live in. posted is
// the language a submission says it was shown in.
func (h *Handler) subscribeLang(w http.ResponseWriter, r *http.Request, s *site, c forms.Config, posted string) (formLang, error) {
	fc, err := h.Forms.FormCopyFor(r.Context(), s.ws, s.set, c)
	if err != nil {
		return formLang{}, err
	}
	pc := h.copyOf(r.Context(), s)
	live := func(code string) bool { return fc.Live(code) && (pc == nil || pc.Live(code)) }
	var pl pageLang
	switch {
	case c.LanguageMode == forms.LanguageFixed && c.FixedLanguage != "" && live(c.FixedLanguage):
		pl = h.chooseLang(w, r, s, func(code string) bool { return code == c.FixedLanguage }, c.FixedLanguage)
	case posted != "" && live(languages.Base(posted)):
		pl = h.chooseLang(w, r, s, live, posted)
	default:
		pl = h.visitorLang(w, r, s, live)
	}
	lists, err := h.Audience.PublicLists(r.Context(), h.Q, s.ws)
	if err != nil {
		return formLang{}, err
	}
	return formLang{pageLang: pl, form: fc.Localized(pl.Code, lists), live: live}, nil
}

// formLangIn is a form's hosted page in a language asked for by name (the
// app's previews).
func (h *Handler) formLangIn(ctx context.Context, s *site, c forms.Config, code string) (formLang, error) {
	fc, err := h.Forms.FormCopyFor(ctx, s.ws, s.set, c)
	if err != nil {
		return formLang{}, err
	}
	pc := h.copyOf(ctx, s)
	live := func(code string) bool { return fc.Live(code) && (pc == nil || pc.Live(code)) }
	pl := h.fixedLang(ctx, s, code, live)
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return formLang{}, err
	}
	return formLang{pageLang: pl, form: fc.Localized(pl.Code, lists), live: live}, nil
}

// subscribeBody builds the hosted page of a form in a language (fl). values
// and out carry a submission back to the page with its errors.
func (h *Handler) subscribeBody(ctx context.Context, s *site, fl formLang, page bool, values *forms.Submission, out forms.Outcome) (subscribeBody, error) {
	c, l := fl.form, fl.Labels
	pages := s.set.Pages.Subscribe
	pc := h.copyOf(ctx, s)
	if pc != nil {
		pages = pc.Pages(fl.Code, s.set.Pages).Subscribe
	}
	privacy := s.set.Branding.PrivacyURL
	if pc != nil {
		privacy = pc.PrivacyURL(fl.Code, privacy)
	}
	b := subscribeBody{
		Heading: c.Content.Heading, Description: c.Content.Description, Footnote: c.Content.Footnote, Button: c.Content.Button,
		ListsLabel: c.Content.ListsLabel, Honeypot: c.Spam.Honeypot, Token: h.Forms.FormToken(formKey(c), h.now()),
		Action: s.links.Subscribe(c.Slug), Consent: splitConsent(c.ConsentText, privacy, l.Text("form.privacy", nil)),
	}
	if page {
		b.Action = s.links.Subscribe("")
		if pages.Heading != "" {
			b.Heading = pages.Heading
		}
		if pages.Description != "" {
			b.Description = pages.Description
		}
		if pages.Footnote != "" {
			b.Footnote = pages.Footnote
		}
	}
	if c.Builtin {
		b.Action = s.links.Subscribe("")
	}
	if b.Heading == "" {
		b.Heading = l.Text("form.heading", i18n.Args{"publisher": s.set.Branding.PublisherName})
	}
	if s.set.Pages.Archive.Enabled {
		b.ArchiveURL = s.links.Archive()
	}
	customs, err := h.Q.ListCustomFields(ctx, s.ws.ID)
	if err != nil {
		return b, err
	}
	for _, f := range c.Fields {
		fv := fieldView{Key: f.Key, Name: forms.FieldName(f.Key), ID: "f-" + f.Key, Label: forms.FieldLabelIn(f, customs, l), Type: "text",
			Required: f.Required || f.Key == forms.FieldEmail, Optional: !f.Required && f.Key != forms.FieldEmail}
		switch f.Key {
		case forms.FieldEmail:
			fv.Type, fv.Placeholder, fv.Autocomplete = "email", c.Content.EmailPlaceholder, "email"
		case forms.FieldFirstName:
			fv.Placeholder, fv.Autocomplete = c.Content.NamePlaceholder, "given-name"
		case forms.FieldLastName:
			fv.Autocomplete = "family-name"
		case forms.FieldCompany:
			fv.Autocomplete = "organization"
		}
		for _, cf := range customs {
			if cf.Key != f.Key {
				continue
			}
			switch cf.Type {
			case "number":
				fv.Type = "number"
			case "date":
				fv.Type = "date"
			case "select":
				fv.Type = "select"
				_ = json.Unmarshal(cf.Options, &fv.Options)
			}
		}
		if values != nil {
			switch f.Key {
			case forms.FieldEmail:
				fv.Value = values.Email
			case forms.FieldFirstName:
				fv.Value = values.FirstName
			case forms.FieldLastName:
				fv.Value = values.LastName
			case forms.FieldCompany:
				fv.Value = values.Company
			default:
				fv.Value = values.Fields[f.Key]
			}
		}
		fv.Error = out.Errors[f.Key]
		b.Fields = append(b.Fields, fv)
	}
	if c.ListChoice && len(c.Lists) > 1 {
		lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
		if err != nil {
			return b, err
		}
		byID := map[uuid.UUID]audience.PublicList{}
		for _, l := range lists {
			byID[l.ID] = l
		}
		checked := map[string]bool{}
		if values != nil {
			for _, id := range values.Lists {
				checked[id] = true
			}
		} else {
			for _, id := range c.DefaultLists {
				checked[id.String()] = true
			}
		}
		for _, id := range c.Lists {
			pl, ok := byID[id]
			if !ok {
				continue
			}
			text := pl.Description
			name := pl.Name
			if pc != nil {
				text, name = pc.ListDescription(fl.Code, pl), pc.ListName(fl.Code, pl)
			}
			if t, ok := pages.ListText[id.String()]; ok && t != "" {
				text = t
			}
			text = strings.ReplaceAll(text, "{subscribers}", l.Count(pl.Subscribed))
			b.Lists = append(b.Lists, listView{ID: id.String(), Name: name, Text: text, Checked: checked[id.String()]})
		}
		b.ListError = out.Errors["lists"]
	}
	if values != nil {
		b.ConsentChecked = values.Consent
	}
	b.ConsentError = out.Errors["consent"]
	if out.Status == forms.OutcomeError || (out.Status == forms.OutcomeInvalid && len(out.Errors) == 0) {
		b.Alert = out.Message
	}
	return b, nil
}

func formKey(c forms.Config) string {
	if c.Builtin {
		return "builtin:" + c.WorkspaceID.String()
	}
	return c.PublicID
}

// formatCount writes 31988 as "31,988".
func formatCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// subscribeTitle is "Subscribe to the Acme newsletter".
func subscribeTitle(l pagelabels.Set, heading string) string {
	if l.Code() == "en" && strings.HasPrefix(heading, "The ") {
		heading = "the " + heading[4:]
	}
	return l.Text("form.title", i18n.Args{"heading": heading})
}

func (h *Handler) renderSubscribe(w http.ResponseWriter, r *http.Request, s *site, fl formLang, page bool, body subscribeBody, status int, preview bool) {
	c := fl.form
	v := h.view(s, subscribeTitle(fl.Labels, body.Heading), body)
	h.setLang(v, r, s, fl.pageLang, fl.live)
	v.Description = body.Description
	slug := c.Slug
	if page || c.Builtin {
		slug = ""
	}
	v.Canonical = s.canon.Subscribe(slug)
	v.Preview = preview
	if s.set.Pages.Subscribe.Index && !preview && status == http.StatusOK {
		v.Robots = "index, follow"
	}
	if !preview && status == http.StatusOK {
		// The page in each language it's Live in, for search engines; a
		// language picked by address is the page's own address.
		v.Alternates = h.alternates(r.Context(), s, v.Canonical, fl.live)
		if r.URL.Query().Get(langParam) != "" && fl.picked {
			v.Canonical = withParam(v.Canonical, langParam, fl.Code)
		}
	}
	v.OG = &og{Type: "website", SiteName: s.set.Branding.PublisherName, Title: v.Title, Description: v.Description, URL: v.Canonical}
	if s.set.Pages.Archive.Enabled {
		v.Feed, v.FeedTitle = s.links.Feed(), fl.Labels.Text("archive.title", i18n.Args{"publisher": s.set.Branding.PublisherName})
	}
	policy := csp{}
	if c.AfterSubmit == "redirect" && c.RedirectURL != "" {
		if u, err := url.Parse(c.RedirectURL); err == nil && u.Host != "" {
			policy.formAction = append(policy.formAction, u.Scheme+"://"+u.Host)
		}
	}
	// Allowed sites may show the hosted page in a frame.
	if len(c.AllowedOrigins) > 0 && !preview {
		policy.ancestors = []string{"'self'"}
		for _, o := range c.AllowedOrigins {
			policy.ancestors = append(policy.ancestors, "https://"+o)
		}
	}
	if preview {
		policy = csp{ancestors: []string{"'self'"}}
	}
	h.render(w, r, s, "subscribe", status, v, policy, preview || status != http.StatusOK)
}

func (h *Handler) subscribePage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	c, page, err := h.resolveForm(r.Context(), s, chi.URLParam(r, "form"), false)
	if errors.Is(err, errNoForm) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if c.Status == forms.StatusPaused {
		h.closed(w, r, s, "")
		return
	}
	fl, err := h.subscribeLang(w, r, s, c, "")
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.Forms.RecordView(r.Context(), s.ws, c, r.UserAgent())
	body, err := h.subscribeBody(r.Context(), s, fl, page, nil, forms.Outcome{})
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.renderSubscribe(w, r, s, fl, page, body, http.StatusOK, false)
}

// langOf is the result page of a submission in the language its form was
// shown in, when the pages are Live in it; else the visitor's.
func (h *Handler) langOf(w http.ResponseWriter, r *http.Request, s *site, code string) pageLang {
	if code != "" {
		if pc := h.copyOf(r.Context(), s); pc != nil && pc.Live(code) {
			return pageLang{Code: code, Labels: pc.Labels(code)}
		}
	}
	return h.visitorLang(w, r, s, nil)
}

func (h *Handler) closed(w http.ResponseWriter, r *http.Request, s *site, code string) {
	pl := h.langOf(w, r, s, code)
	l := pl.Labels
	body := messageBody{Heading: l.Text("form.closed.heading", nil), Text: l.Text("form.closed.text", i18n.Args{"publisher": s.set.Branding.PublisherName})}
	if s.set.Pages.Archive.Enabled {
		body.Actions = []action{{Label: l.Text("page.pastIssues", nil), Href: s.links.Archive()}}
	}
	h.messageIn(w, r, s, pl, http.StatusGone, pageTitle(pl, s, "form.closed.title"), body, false)
}

// submissionFrom reads a posted form (the hosted page, the plain HTML
// form, the embed script).
func submissionFrom(r *http.Request) forms.Submission {
	f := r.PostForm
	sub := forms.Submission{
		Email: f.Get("email"), FirstName: f.Get("first_name"), LastName: f.Get("last_name"), Company: f.Get("company"),
		Lists: f["list"], Consent: f.Get("consent") != "", Honeypot: f.Get("website"), Token: f.Get("_t"), Plain: f.Get("_plain") == "1",
		Origin: forms.OriginHost(r.Header.Get("Origin"), r.Header.Get("Referer")), IP: clientIP(r), UserAgent: r.UserAgent(),
		Fields: map[string]string{}, Language: clipLang(f.Get("_lang")), BrowserLanguage: audience.BrowserLanguage(r.Header.Get("Accept-Language")),
	}
	if len(sub.Lists) > 50 {
		sub.Lists = sub.Lists[:50]
	}
	n := 0
	for k, v := range f {
		if strings.HasPrefix(k, "f.") && len(v) > 0 && n < 50 {
			sub.Fields[strings.TrimPrefix(k, "f.")] = v[0]
			n++
		}
	}
	return sub
}

// clipLang keeps a posted language code to a sensible length; the forms
// service checks it is one the form is Live in.
func clipLang(v string) string {
	if len(v) > 35 {
		return ""
	}
	return v
}

func (h *Handler) parseForm(w http.ResponseWriter, r *http.Request, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		return r.ParseMultipartForm(limit) == nil //nolint:gosec // the body is capped by MaxBytesReader above
	}
	return r.ParseForm() == nil
}

func (h *Handler) subscribePost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if !h.parseForm(w, r, 64<<10) {
		http.Error(w, h.visitorLang(nil, r, s, nil).Labels.Text("page.unreadable", nil), http.StatusBadRequest)
		return
	}
	c, page, err := h.resolveForm(r.Context(), s, chi.URLParam(r, "form"), false)
	if errors.Is(err, errNoForm) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	sub := submissionFrom(r)
	// The archive's "Get the next one" box: fill the email in and let the
	// visitor finish the form (lists, consent).
	if r.PostForm.Get("step") == "prefill" {
		prefill := forms.Submission{Email: strings.TrimSpace(sub.Email), Fields: map[string]string{}}
		for _, id := range c.DefaultLists {
			prefill.Lists = append(prefill.Lists, id.String())
		}
		fl, err := h.subscribeLang(w, r, s, c, sub.Language)
		if err != nil {
			h.serverError(w, r, s, err)
			return
		}
		body, err := h.subscribeBody(r.Context(), s, fl, page, &prefill, forms.Outcome{})
		if err != nil {
			h.serverError(w, r, s, err)
			return
		}
		h.renderSubscribe(w, r, s, fl, page, body, http.StatusOK, false)
		return
	}
	out := h.Forms.Submit(r.Context(), s.ws, s.set, c, sub)
	h.subscribeResult(w, r, s, c, page, sub, out)
}

func (h *Handler) subscribeResult(w http.ResponseWriter, r *http.Request, s *site, c forms.Config, page bool, sub forms.Submission, out forms.Outcome) {
	publisher := s.set.Branding.PublisherName
	pl := h.langOf(w, r, s, out.Language)
	l := pl.Labels
	// The address as it was saved (trimmed, lower case), not as it was typed.
	email := out.Email
	if email == "" {
		email = strings.ToLower(strings.TrimSpace(sub.Email))
	}
	archive := []action{}
	if s.set.Pages.Archive.Enabled {
		archive = append(archive, action{Label: l.Text("page.pastIssues", nil), Href: s.links.Archive()})
	}
	switch out.Status {
	case forms.OutcomeConfirm, forms.OutcomeSubscribed:
		if out.RedirectURL != "" {
			http.Redirect(w, r, out.RedirectURL, http.StatusSeeOther)
			return
		}
		if out.Status == forms.OutcomeConfirm {
			h.messageIn(w, r, s, pl, http.StatusOK, pageTitle(pl, s, "form.confirm.heading"), messageBody{Icon: "mail", Heading: l.Text("form.confirm.heading", nil),
				Lead:    labelHTML(l, "form.confirm.text", i18n.Args{"email": email, "publisher": publisher}, ""),
				Actions: archive, Footer: l.Text("form.confirm.footer", nil)}, true)
			return
		}
		h.messageIn(w, r, s, pl, http.StatusOK, pageTitle(pl, s, "form.subscribed.heading"), messageBody{Icon: "tick", Heading: l.Text("form.subscribed.heading", nil),
			Lead: labelHTML(l, "form.subscribed.text", i18n.Args{"email": email, "publisher": publisher}, ""), Actions: archive}, true)
	case forms.OutcomeInvalid, forms.OutcomeError:
		status := http.StatusUnprocessableEntity
		if out.Status == forms.OutcomeError {
			status = http.StatusServiceUnavailable
		}
		fl, err := h.subscribeLang(w, r, s, c, out.Language)
		if err != nil {
			h.serverError(w, r, s, err)
			return
		}
		body, err := h.subscribeBody(r.Context(), s, fl, page, &sub, out)
		if err != nil {
			h.serverError(w, r, s, err)
			return
		}
		h.renderSubscribe(w, r, s, fl, page, body, status, false)
	case forms.OutcomeChallenge:
		h.renderChallenge(w, r, s, pl, out.ChallengeID, "")
	case forms.OutcomeRateLimited:
		w.Header().Set("Retry-After", "60")
		h.messageIn(w, r, s, pl, http.StatusTooManyRequests, pageTitle(pl, s, "form.tooMany.heading"), messageBody{Heading: l.Text("form.tooMany.heading", nil), Text: out.Message,
			Actions: []action{{Label: l.Text("page.backToForm", nil), Href: s.links.Subscribe(slugOf(c, page))}}}, true)
	case forms.OutcomeClosed:
		h.closed(w, r, s, out.Language)
	default:
		reason := out.Message
		if reason == "" {
			reason = l.Text("error.origin", nil)
		}
		h.messageIn(w, r, s, pl, http.StatusForbidden, pageTitle(pl, s, "form.notAllowed.title"), messageBody{Heading: l.Text("form.notAllowed.heading", nil), Text: l.Text("form.notAllowed.text", i18n.Args{"reason": reason}),
			Actions: []action{{Label: l.Text("page.subscribePage", nil), Href: s.links.Subscribe("")}}}, true)
	}
}

func slugOf(c forms.Config, page bool) string {
	if page || c.Builtin {
		return ""
	}
	return c.Slug
}

type challengeBody struct {
	ID, SiteKey, Action, ErrorURL, Alert string
}

func (h *Handler) renderChallenge(w http.ResponseWriter, r *http.Request, s *site, pl pageLang, id, alert string) {
	body := challengeBody{ID: id, SiteKey: s.set.TurnstileSiteKey(), Action: s.links.Challenge(), ErrorURL: s.links.TurnstileError(), Alert: alert}
	v := h.view(s, pageTitle(pl, s, "challenge.title"), body)
	h.setLang(v, r, s, pl, nil)
	for i := range v.Menu {
		// The check itself in another language, whichever form led here.
		v.Menu[i].Href = s.links.Challenge() + "?" + url.Values{"c": {id}, langParam: {v.Menu[i].Code}}.Encode()
	}
	v.Scripts = []string{s.links.Static("challenge.js?v=" + h.version), forms.TurnstileScript}
	h.render(w, r, s, "challenge", http.StatusOK, v, csp{scripts: []string{forms.TurnstileOrigin}, frames: []string{forms.TurnstileOrigin}}, true)
}

func (h *Handler) challengePage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("c")
	if _, err := uuid.Parse(id); err != nil || s.set.TurnstileSiteKey() == "" {
		h.notFound(w, r, s)
		return
	}
	h.renderChallenge(w, r, s, h.visitorLang(w, r, s, nil), id, "")
}

func (h *Handler) challengePost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if !h.parseForm(w, r, 16<<10) {
		http.Error(w, h.visitorLang(nil, r, s, nil).Labels.Text("page.unreadable", nil), http.StatusBadRequest)
		return
	}
	id, answer := r.PostForm.Get("challenge"), r.PostForm.Get("cf-turnstile-response")
	if answer == "" {
		pl := h.langOf(w, r, s, languages.Base(r.PostForm.Get("_lang")))
		h.renderChallenge(w, r, s, pl, id, pl.Labels.Text("challenge.finish", nil))
		return
	}
	var held forms.Config
	var heldPage bool
	formOf := func(ctx context.Context, formID string, builtin bool) (forms.Config, error) {
		if builtin {
			c, _, err := h.resolveForm(ctx, s, "", false)
			held, heldPage = c, true
			return c, err
		}
		fid, err := uuid.Parse(formID)
		if err != nil {
			return forms.Config{}, err
		}
		f, err := h.Q.FormGet(ctx, store.FormGetParams{WorkspaceID: s.ws.ID, ID: fid})
		if err != nil {
			return forms.Config{}, err
		}
		held = forms.ConfigOf(f)
		heldPage = s.set.HostedFormID() != nil && *s.set.HostedFormID() == f.ID
		return held, nil
	}
	out := h.Forms.SolveChallenge(r.Context(), s.ws, s.set, id, answer, r.PostForm.Get("_lang"), clientIP(r), formOf)
	if out.Status == forms.OutcomeInvalid && len(out.Errors) == 0 && held.ID == uuid.Nil && !held.Builtin {
		pl := h.langOf(w, r, s, out.Language)
		h.messageIn(w, r, s, pl, http.StatusGone, pageTitle(pl, s, "challenge.expired.title"), messageBody{Heading: pl.Labels.Text("challenge.expired.heading", nil),
			Text: out.Message, Actions: []action{{Label: pl.Labels.Text("page.backToForm", nil), Href: s.links.Subscribe("")}}}, true)
		return
	}
	sub := forms.Submission{Email: out.Email, FirstName: out.FirstName}
	h.subscribeResult(w, r, s, held, heldPage, sub, out)
}

func (h *Handler) embedForm(ctx context.Context, s *site, publicID string) (forms.Config, error) {
	f, err := h.Q.FormByPublicID(ctx, store.FormByPublicIDParams{WorkspaceID: s.ws.ID, PublicID: publicID})
	if err != nil {
		return forms.Config{}, err
	}
	return forms.ConfigOf(f), nil
}

func (h *Handler) formScript(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "script")
	if !strings.HasSuffix(name, ".js") {
		http.NotFound(w, r)
		return
	}
	s, err := h.site(r)
	if err != nil {
		jsError(w, http.StatusNotFound, "This signup form doesn't exist.")
		return
	}
	c, err := h.embedForm(r.Context(), s, strings.TrimSuffix(name, ".js"))
	if err != nil || c.Status == forms.StatusDraft {
		jsError(w, http.StatusNotFound, "This signup form doesn't exist or isn't published yet.")
		return
	}
	js, err := h.Forms.Script(r.Context(), s.ws, s.set, c, s.links)
	if err != nil {
		jsError(w, http.StatusInternalServerError, "This signup form couldn't load. Reload the page.")
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", "text/javascript; charset=utf-8")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cross-Origin-Resource-Policy", "cross-origin")
	hd.Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(js) //nolint:gosec // generated script; the form's settings are JSON-encoded into it
}

func jsError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "console.warn(%q);\n", "ReputeCast: "+msg)
}

// cors answers the CORS part of an embed request: only the form's allowed
// sites (and the instance's own pages) may read the answer.
func (h *Handler) cors(w http.ResponseWriter, r *http.Request, s *site, c forms.Config) bool {
	w.Header().Add("Vary", "Origin")
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // same-origin or not a browser: no CORS headers needed
	}
	if !forms.OriginAllowed(forms.OriginHost(origin, ""), c.AllowedOrigins, h.Forms.OwnOrigins(s.set)) {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	return true
}

func (h *Handler) formPreflight(w http.ResponseWriter, r *http.Request) {
	s, err := h.site(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	c, err := h.embedForm(r.Context(), s, chi.URLParam(r, "id"))
	if err != nil || !h.cors(w, r, s, c) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	hd := w.Header()
	hd.Set("Access-Control-Allow-Methods", "GET, POST")
	hd.Set("Access-Control-Allow-Headers", "Accept, Content-Type")
	hd.Set("Access-Control-Max-Age", "600")
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// formSession gives an embedded form its timestamp token and counts a view.
func (h *Handler) formSession(w http.ResponseWriter, r *http.Request) {
	s, err := h.site(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"status": "missing"})
		return
	}
	c, err := h.embedForm(r.Context(), s, chi.URLParam(r, "id"))
	if err != nil || c.Status == forms.StatusDraft {
		writeJSON(w, http.StatusNotFound, map[string]string{"status": "missing"})
		return
	}
	if !h.cors(w, r, s, c) {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": forms.OutcomeRejected})
		return
	}
	h.Forms.RecordView(r.Context(), s.ws, c, r.UserAgent())
	writeJSON(w, http.StatusOK, map[string]string{"t": h.Forms.FormToken(formKey(c), h.now())})
}

// formSubmit receives an embedded form's submission and answers in JSON.
func (h *Handler) formSubmit(w http.ResponseWriter, r *http.Request) {
	s, err := h.site(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, forms.Outcome{Status: forms.OutcomeClosed})
		return
	}
	c, err := h.embedForm(r.Context(), s, chi.URLParam(r, "id"))
	if err != nil || c.Status == forms.StatusDraft {
		writeJSON(w, http.StatusNotFound, forms.Outcome{Status: forms.OutcomeClosed, Message: h.visitorLang(nil, r, s, nil).Labels.Text("form.gone", nil)})
		return
	}
	allowed := h.cors(w, r, s, c)
	if !h.parseForm(w, r, 64<<10) {
		writeJSON(w, http.StatusBadRequest, forms.Outcome{Status: forms.OutcomeInvalid, Message: h.visitorLang(nil, r, s, nil).Labels.Text("form.unreadableShort", nil)})
		return
	}
	sub := submissionFrom(r)
	if !allowed {
		sub.Origin = forms.OriginHost(r.Header.Get("Origin"), "")
	}
	out := h.Forms.Submit(r.Context(), s.ws, s.set, c, sub)
	// Plain HTML forms (no script) get the hosted result page instead.
	if !strings.Contains(r.Header.Get("Accept"), "application/json") && r.PostForm.Get("_js") == "" {
		h.subscribeResult(w, r, s, c, false, sub, out)
		return
	}
	status := http.StatusOK
	body := map[string]any{"status": out.Status}
	switch out.Status {
	case forms.OutcomeInvalid:
		status = http.StatusUnprocessableEntity
	case forms.OutcomeRateLimited:
		status = http.StatusTooManyRequests
		w.Header().Set("Retry-After", "60")
	case forms.OutcomeRejected:
		status = http.StatusForbidden
	case forms.OutcomeClosed:
		status = http.StatusGone
	case forms.OutcomeError:
		status = http.StatusServiceUnavailable
	case forms.OutcomeChallenge:
		body["challengeUrl"] = s.links.Challenge() + "?c=" + url.QueryEscape(out.ChallengeID)
	}
	if len(out.Errors) > 0 {
		body["errors"] = out.Errors
	}
	if out.Message != "" {
		body["message"] = out.Message
	}
	if out.RedirectURL != "" {
		body["redirect"] = out.RedirectURL
	}
	writeJSON(w, status, body)
}
