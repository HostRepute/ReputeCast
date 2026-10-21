package publicpages

import (
	"context"
	"html"
	"html/template"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/store"
)

// Texts on the public subscribe page (SMS, design expansion phase 8, X21):
// the phone field with its country picker and the SMS consent box, with
// its own text and unchecked by default. The words and the countries come
// from internal/texting through forms.Texts; without it (or in a workspace
// that doesn't text) the page shows neither.

// smsConsentView is the SMS consent box.
type smsConsentView struct {
	HTML    template.HTML
	Checked bool
}

// smsSignup turns the form's phone field into the number with its country
// picker, or drops it when the workspace doesn't text, and adds the SMS
// consent box.
func (h *Handler) smsSignup(ctx context.Context, s *site, fl formLang, b *subscribeBody, values *forms.Submission, out forms.Outcome, privacy string) {
	i := slices.IndexFunc(b.Fields, func(f fieldView) bool { return f.Key == forms.FieldPhone })
	if i < 0 {
		return
	}
	c := fl.form
	var offer forms.TextOffer
	ok := false
	if h.Forms != nil && h.Forms.Texts != nil {
		offer, ok = h.Forms.Texts.Offer(ctx, s.ws, c.ShownLanguage())
	}
	if !ok {
		b.Fields = slices.Delete(b.Fields, i, i+1)
		return
	}
	f := &b.Fields[i]
	f.Type, f.Autocomplete, f.Placeholder = "tel", "tel", offer.Placeholder
	if p, _ := c.Field(forms.FieldPhone); p.Label == "" {
		f.Label = offer.PhoneLabel
	}
	f.Countries, f.CountryLabel, f.Country = offer.Countries, offer.CountryLabel, offer.Default
	f.Value = ""
	if values != nil {
		f.Value = values.Phone
		if values.PhoneCountry != "" {
			f.Country = values.PhoneCountry
		}
	}
	f.Error = out.Errors[forms.FieldPhone]
	if !c.Content.SmsConsent {
		return
	}
	links := []phraseLink{{Words: offer.TermsWords, Href: offer.TermsURL}}
	if privacy != "" {
		links = append(links, phraseLink{Words: fl.Labels.Text("form.privacy", nil), Href: privacy})
	}
	b.Sms = &smsConsentView{HTML: linkPhrases(c.SMSConsentText(offer), links), Checked: values != nil && values.SmsConsent}
}

// TextPrefs is the preference center's texts: a Texts switch per list and
// the number they go to. Nil when texting isn't wired (the app sets it).
type TextPrefs interface {
	// Prefs is the contact's switches and their words; false when the
	// workspace doesn't text or the contact has no number.
	Prefs(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, lang string) (PrefTexts, bool)
	// SavePrefs records the switches of the lists shown, with the request's
	// evidence, in the caller's transaction; it returns the lists whose
	// opt-in confirmation Confirm texts after the commit.
	SavePrefs(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, shown, on []uuid.UUID, source, ip, userAgent string) ([]uuid.UUID, error)
	Confirm(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, lists []uuid.UUID)
}

// PrefTexts is a contact's Texts switches.
type PrefTexts struct {
	// Phone is the number, masked; Label the switches' word ("Texts"); Note
	// the line under the lists; Save the account page's button.
	Phone, Label, Note, Save string
	On                       map[uuid.UUID]bool
	// Locked: the number replied STOP, so the switches stay off.
	Locked bool
}

// textsView is the preference center's Texts switches.
type textsView struct {
	Label, Note, Shown string
	Locked             bool
}

// prefTexts adds the Texts switch to each list the page shows.
func (h *Handler) prefTexts(ctx context.Context, s *site, contactID uuid.UUID, code string, b *preferencesBody) {
	if h.TextPrefs == nil || len(b.Lists) == 0 {
		return
	}
	p, ok := h.TextPrefs.Prefs(ctx, s.ws, contactID, code)
	if !ok {
		return
	}
	shown := make([]string, 0, len(b.Lists))
	for i := range b.Lists {
		id, err := uuid.Parse(b.Lists[i].ID)
		if err != nil {
			continue
		}
		b.Lists[i].Texts = p.On[id]
		shown = append(shown, b.Lists[i].ID)
	}
	b.Texts = &textsView{Label: p.Label, Note: p.Note, Shown: strings.Join(shown, ","), Locked: p.Locked}
}

// saveTexts records the posted Texts switches, from the preference center
// or the account page (source); the lists to confirm come back for after
// the commit.
func (h *Handler) saveTexts(ctx context.Context, tx pgx.Tx, r *http.Request, s *site, contactID uuid.UUID, source string) ([]uuid.UUID, error) {
	f := r.PostForm
	if h.TextPrefs == nil || !f.Has("sms_shown") {
		return nil, nil
	}
	var shown, on []uuid.UUID
	for _, raw := range strings.Split(f.Get("sms_shown"), ",") {
		if id, err := uuid.Parse(strings.TrimSpace(raw)); err == nil && len(shown) < 50 {
			shown = append(shown, id)
		}
	}
	for _, raw := range f["sms_list"] {
		if id, err := uuid.Parse(raw); err == nil && len(on) < 50 {
			on = append(on, id)
		}
	}
	ip := ""
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	return h.TextPrefs.SavePrefs(ctx, tx, s.ws, contactID, shown, on, source, ip, r.UserAgent())
}

// clipTo keeps a posted value to n bytes (the number, its country).
func clipTo(v string, n int) string {
	v = strings.TrimSpace(v)
	if len(v) > n {
		return ""
	}
	return v
}

// phraseLink is words of a sentence and where they link.
type phraseLink struct {
	Words, Href string
}

// linkPhrases escapes text and links the first occurrence of each link's
// words, in any letter case.
func linkPhrases(text string, links []phraseLink) template.HTML {
	type hit struct {
		at, end int
		href    string
	}
	var hits []hit
	for _, l := range links {
		if l.Words == "" || l.Href == "" {
			continue
		}
		if i, j := forms.PhraseIndex(text, l.Words); i >= 0 {
			hits = append(hits, hit{i, j, l.Href})
		}
	}
	slices.SortFunc(hits, func(a, b hit) int { return a.at - b.at })
	var sb strings.Builder
	pos := 0
	for _, x := range hits {
		if x.at < pos {
			continue
		}
		sb.WriteString(html.EscapeString(text[pos:x.at]))
		sb.WriteString(`<a href="` + html.EscapeString(x.href) + `">` + html.EscapeString(text[x.at:x.end]) + `</a>`)
		pos = x.end
	}
	sb.WriteString(html.EscapeString(text[pos:]))
	return template.HTML(sb.String()) //nolint:gosec // every piece is escaped above
}

// accountTexts is the website account page's Texts section: a switch per
// list it shows, the number they go to, Save. False when the member has no
// number or the workspace doesn't text.
func (h *Handler) accountTexts(ctx context.Context, s *site, contactID uuid.UUID, code string, lists []accountListView) (accountSectionView, bool) {
	if h.TextPrefs == nil || len(lists) == 0 {
		return accountSectionView{}, false
	}
	p, ok := h.TextPrefs.Prefs(ctx, s.ws, contactID, code)
	if !ok {
		return accountSectionView{}, false
	}
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString(`<div class="ch"><h2 id="atx-h">` + esc(p.Label) + `</h2><span class="st-k">` + esc(p.Note) + `</span></div>`)
	b.WriteString(`<form method="post" action="` + esc(s.path("account", "texts")) + `" class="st-stack"><div>`)
	shown := make([]string, 0, len(lists))
	for _, l := range lists {
		id, err := uuid.Parse(l.ID)
		if err != nil {
			continue
		}
		shown = append(shown, l.ID)
		attrs := ""
		if p.On[id] {
			attrs += " checked"
		}
		if p.Locked {
			attrs += " disabled"
		}
		b.WriteString(`<div class="st-line"><div class="tx"><b id="tx-` + esc(l.ID) + `">` + esc(l.Name) + `</b></div><input type="checkbox" class="st-sw" role="switch" name="sms_list" value="` +
			esc(l.ID) + `"` + attrs + ` aria-labelledby="atx-h tx-` + esc(l.ID) + `"></div>`)
	}
	b.WriteString(`</div><input type="hidden" name="sms_shown" value="` + esc(strings.Join(shown, ",")) + `">`)
	if !p.Locked {
		b.WriteString(`<div class="st-split"><button class="st-btn" type="submit">` + esc(p.Save) + `</button></div>`)
	}
	b.WriteString(`</form>`)
	return accountSectionView{ID: "texts", Title: p.Label, HTML: template.HTML(b.String())}, true //nolint:gosec // every piece is escaped above
}

// saveAccountTexts records the account page's Texts switches, with the
// account page as their evidence, and texts the opt-in confirmation once.
func (h *Handler) saveAccountTexts(ctx context.Context, r *http.Request, s *site, contactID uuid.UUID) error {
	if h.TextPrefs == nil {
		return nil
	}
	var confirm []uuid.UUID
	err := db.Tx(ctx, h.Pool, func(tx pgx.Tx) error {
		var err error
		confirm, err = h.saveTexts(ctx, tx, r, s, contactID, "account_page")
		return err
	})
	if err == nil && len(confirm) > 0 {
		h.TextPrefs.Confirm(ctx, s.ws, contactID, confirm)
	}
	return err
}
