package forms

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"html"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed embed/form.js
var embedScript string

//go:embed embed/form.css
var embedCSS []byte

// gzipKB is the gzipped size of b in KB, rounded up.
func gzipKB(b []byte) int {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return (buf.Len() + 1023) / 1024
}

// EmbedCSS is the stylesheet of embedded and popup forms (/p/_/embed.css).
func EmbedCSS() []byte { return embedCSS }

// EmbedCSSVersion changes with the stylesheet (cache busting).
var EmbedCSSVersion = func() string {
	sum := sha256.Sum256(embedCSS)
	return hex.EncodeToString(sum[:6])
}()

type embedField struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []string `json:"options"`
}

type embedList struct {
	ID      string `json:"id"`
	Checked bool   `json:"checked"`
}

type embedMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// embedCopy is the form in one language.
type embedCopy struct {
	Lang        string                  `json:"lang"`
	Dir         string                  `json:"dir"`
	Heading     string                  `json:"heading"`
	Description string                  `json:"description"`
	Button      string                  `json:"button"`
	Footnote    string                  `json:"footnote"`
	ListsLabel  string                  `json:"listsLabel"`
	Fields      []embedField            `json:"fields"`
	Lists       map[string]string       `json:"lists"`
	Consent     string                  `json:"consent"`
	Privacy     string                  `json:"privacy"`
	PrivacyText string                  `json:"privacyText"`
	Messages    map[string]embedMessage `json:"messages"`
	Text        map[string]string       `json:"text"`
}

// embedConfig is what the script renders: the form's settings, and its copy
// in every language it is Live in (Langs), with which one to show (Mode,
// Fixed) and the language to fall back to.
type embedConfig struct {
	ID         string               `json:"id"`
	Kind       string               `json:"kind"`
	Submit     string               `json:"submit"`
	Session    string               `json:"session"`
	CSS        string               `json:"css"`
	Hosted     string               `json:"hosted"`
	Accent     string               `json:"accent"`
	Hover      string               `json:"hover"`
	OnAccent   string               `json:"onAccent"`
	Link       string               `json:"link"`
	Publisher  string               `json:"publisher"`
	Initial    string               `json:"initial"`
	Logo       string               `json:"logo"`
	Lists      []embedList          `json:"lists"`
	ListChoice bool                 `json:"listChoice"`
	Honeypot   bool                 `json:"honeypot"`
	Popup      map[string]int       `json:"popup"`
	Mode       string               `json:"mode"`
	Fixed      string               `json:"fixed"`
	Fallback   string               `json:"fallback"`
	Langs      map[string]embedCopy `json:"langs"`
}

// FieldLabelIn is the label a field shows in a language: the form's own,
// else the built-in field's label in that language, else the custom
// field's name.
func FieldLabelIn(f fieldDoc, customs []store.CustomField, l pagelabels.Set) string {
	if f.Label != "" {
		return f.Label
	}
	switch f.Key {
	case FieldEmail:
		return l.Text("form.email", nil)
	case FieldFirstName:
		return l.Text("form.firstName", nil)
	case FieldLastName:
		return l.Text("form.lastName", nil)
	case FieldCompany:
		return l.Text("form.company", nil)
	}
	return FieldLabel(f, customs)
}

// FieldLabel is the label a field shows.
func FieldLabel(f fieldDoc, customs []store.CustomField) string {
	if f.Label != "" {
		return f.Label
	}
	if l, ok := builtinFields[f.Key]; ok {
		return l
	}
	for _, c := range customs {
		if c.Key == f.Key {
			return c.Label
		}
	}
	return f.Key
}

// FieldName is the name of a field in a form post: built-ins by key,
// custom fields as "f.<key>".
func FieldName(key string) string {
	if _, ok := builtinFields[key]; ok {
		return key
	}
	return "f." + key
}

// doneMessages are what the form says after a signup, in its language:
// the publisher's after-submit message, else the built-in one.
func doneMessages(c Config) map[string]embedMessage {
	l := c.Labels
	confirm := embedMessage{Title: l.Text("form.confirm.heading", nil), Text: l.Text("form.confirm.short", nil)}
	subscribed := embedMessage{Title: l.Text("form.subscribed.heading", nil), Text: l.Text("form.subscribed.short", i18n.Args{"publisher": c.publisher})}
	if t := c.Content.AfterSubmitText; t != "" {
		confirm.Text, subscribed.Text = t, t
	}
	return map[string]embedMessage{
		OutcomeConfirm:    confirm,
		OutcomeSubscribed: subscribed,
		"error":           {Title: "", Text: l.Text("error.saveFailed", nil)},
	}
}

// listName is a list's name in the form's language.
func (c Config) listName(l audience.PublicList) string {
	if n := c.ListNames[l.ID]; n != "" {
		return n
	}
	return l.Name
}

// privacy is the privacy policy link in the form's language.
func (c Config) privacy(b Branding) string {
	if c.PrivacyURL != "" {
		return c.PrivacyURL
	}
	return b.PrivacyURL
}

// embedCopyOf is the form's copy in the language c shows.
func embedCopyOf(c Config, b Branding, lists []audience.PublicList, customs []store.CustomField) embedCopy {
	l := c.Labels
	out := embedCopy{Lang: c.ShownLanguage(), Dir: pagelabels.Dir(c.ShownLanguage()), Heading: c.Content.Heading, Description: c.Content.Description,
		Button: c.Content.Button, Footnote: c.Content.Footnote, ListsLabel: c.Content.ListsLabel, Fields: []embedField{}, Lists: map[string]string{},
		Consent: c.ConsentText, Privacy: c.privacy(b), PrivacyText: l.Text("form.privacy", nil), Messages: doneMessages(c),
		Text: map[string]string{
			"optional": l.Text("form.optional", nil), "choose": l.Text("form.choose", nil), "honeypot": l.Text("form.honeypot", nil),
			"emailMissing": l.Text("error.emailMissing", nil), "emailInvalid": l.Text("error.emailInvalid", nil), "consent": l.Text("error.consent", nil),
			"close": l.Text("form.close", nil),
		}}
	for _, f := range c.Fields {
		ef := embedField{Key: f.Key, Name: FieldName(f.Key), Label: FieldLabelIn(f, customs, l), Type: "text", Required: f.Required || f.Key == FieldEmail, Options: []string{}}
		switch f.Key {
		case FieldEmail:
			ef.Type, ef.Placeholder = "email", c.Content.EmailPlaceholder
		case FieldFirstName:
			ef.Placeholder = c.Content.NamePlaceholder
		}
		for _, cf := range customs {
			if cf.Key == f.Key {
				ef.Type = cf.Type
				_ = json.Unmarshal(cf.Options, &ef.Options)
				if ef.Options == nil {
					ef.Options = []string{}
				}
			}
		}
		out.Fields = append(out.Fields, ef)
	}
	for _, pl := range lists {
		if slices.Contains(c.Lists, pl.ID) {
			out.Lists[pl.ID.String()] = c.listName(pl)
		}
	}
	return out
}

// EmbedScript is the script of a form in the language c shows it in.
func EmbedScript(c Config, links PageLinks, b Branding, lists []audience.PublicList, customs []store.CustomField) []byte {
	if c.publisher == "" {
		c.publisher = b.PublisherName
	}
	cp := embedCopyOf(c, b, lists, customs)
	return embedScriptOf(c, links, b, lists, map[string]embedCopy{cp.Lang: cp}, cp.Lang)
}

// Script is the script of a form (/p/{workspace}/forms/{id}.js) with its
// copy in every language it is Live in. It holds no personal data.
func (s *Service) Script(ctx context.Context, ws *store.Workspace, set *Settings, c Config, links PageLinks) ([]byte, error) {
	fc, err := s.loadFormCopy(ctx, s.Q, ws, set, c)
	if err != nil {
		return nil, err
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	customs, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	return fc.script(links, set.Branding, lists, customs), nil
}

func (fc *FormCopy) script(links PageLinks, b Branding, lists []audience.PublicList, customs []store.CustomField) []byte {
	langs := map[string]embedCopy{}
	for _, code := range fc.LiveLanguages() {
		lc := fc.In(code)
		fc.listNamesIn(&lc, lists)
		langs[code] = embedCopyOf(lc, b, lists, customs)
	}
	return embedScriptOf(fc.Form, links, b, lists, langs, fc.Fallback())
}

// plain is the form for its plain HTML copy: in its own language, or the
// one it always shows when that is Live.
func (fc *FormCopy) plain(lists []audience.PublicList) Config {
	c := fc.In(fc.Pick(""))
	fc.listNamesIn(&c, lists)
	return c
}

func embedScriptOf(c Config, links PageLinks, b Branding, lists []audience.PublicList, langs map[string]embedCopy, fallback string) []byte {
	mode := c.LanguageMode
	if mode == "" {
		mode = LanguagePage
	}
	cfg := embedConfig{
		ID: c.PublicID, Kind: c.Kind, Submit: links.Submit(c.PublicID), Session: links.FormSession(c.PublicID),
		CSS: links.Static("embed.css?v=" + EmbedCSSVersion), Hosted: links.PlainAction(c.Slug),
		Accent: b.Palette.Accent, Hover: b.Palette.Hover, OnAccent: b.Palette.OnAccent, Link: b.Palette.Text,
		Publisher: b.PublisherName, Initial: b.Initial(), ListChoice: c.ListChoice, Honeypot: c.Spam.Honeypot,
		Popup: map[string]int{"delay": c.Content.PopupDelay, "scroll": c.Content.PopupScroll, "repeatDays": c.Content.PopupRepeatDays},
		Lists: []embedList{}, Mode: mode, Fixed: c.FixedLanguage, Fallback: fallback, Langs: langs,
	}
	if b.Logo != nil {
		cfg.Logo = links.Logo(b.Logo.Hash)
	}
	byID := map[uuid.UUID]bool{}
	for _, l := range lists {
		byID[l.ID] = true
	}
	for _, id := range c.Lists {
		if byID[id] {
			cfg.Lists = append(cfg.Lists, embedList{ID: id.String(), Checked: slices.Contains(c.DefaultLists, id)})
		}
	}
	raw, _ := json.Marshal(cfg) // escapes <, > and & for script contexts
	return []byte(strings.Replace(embedScript, "__RC_FORM__", string(raw), 1))
}

// PrivacyPolicyIndex finds "privacy policy" in a consent sentence, in any
// letter case, and returns its byte offset in text, or -1. The phrase is
// ASCII, so the search compares bytes and the offset always points into
// text itself (lowercasing first would shift it for letters whose
// lowercase form has another UTF-8 length, like "İ").
//
// Consent texts in other languages are linked with PhraseIndex and the
// language's words for it (the form.privacy label).
func PrivacyPolicyIndex(text string) int {
	const want = "privacy policy"
	for i := 0; i+len(want) <= len(text); i++ {
		match := true
		for j := 0; j < len(want); j++ {
			c := text[i+j]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			if c != want[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// PlainAction is where the plain HTML form posts: always under /p/ so the
// cross-site post reaches the form whichever site it sits on.
func (l PageLinks) PlainAction(slug string) string {
	return l.origin + "/p/" + l.slug + "/subscribe/" + slug
}

// PlainHTML is the form as plain HTML for sites that can't load scripts,
// in the language c shows it in (it says so, so the signup records the
// consent text it showed). It posts to the hosted page, which answers with
// the result page.
func PlainHTML(c Config, links PageLinks, lists []audience.PublicList, customs []store.CustomField, b Branding) string {
	esc := html.EscapeString
	lang := c.ShownLanguage()
	if lang == "" {
		lang = "en"
	}
	var w bytes.Buffer
	w.WriteString(`<form action="` + esc(links.PlainAction(c.Slug)) + `" method="post" accept-charset="utf-8" lang="` + esc(lang) + `">` + "\n")
	w.WriteString(`  <input type="hidden" name="_plain" value="1">` + "\n")
	w.WriteString(`  <input type="hidden" name="_lang" value="` + esc(lang) + `">` + "\n")
	for _, f := range c.Fields {
		label := esc(FieldLabelIn(f, customs, c.Labels))
		name := esc(FieldName(f.Key))
		req := ""
		if f.Required || f.Key == FieldEmail {
			req = " required"
		}
		switch f.Key {
		case FieldEmail:
			w.WriteString(`  <label>` + label + ` <input type="email" name="email" autocomplete="email"` + req + `></label>` + "\n")
		case FieldFirstName:
			w.WriteString(`  <label>` + label + ` <input type="text" name="first_name" autocomplete="given-name"` + req + `></label>` + "\n")
		case FieldLastName:
			w.WriteString(`  <label>` + label + ` <input type="text" name="last_name" autocomplete="family-name"` + req + `></label>` + "\n")
		case FieldCompany:
			w.WriteString(`  <label>` + label + ` <input type="text" name="company" autocomplete="organization"` + req + `></label>` + "\n")
		default:
			w.WriteString(`  <label>` + label + ` <input type="text" name="` + name + `"` + req + `></label>` + "\n")
		}
	}
	byID := map[uuid.UUID]string{}
	for _, l := range lists {
		byID[l.ID] = c.listName(l)
	}
	if c.ListChoice && len(c.Lists) > 1 {
		w.WriteString(`  <fieldset>` + "\n    <legend>" + esc(c.Content.ListsLabel) + "</legend>\n")
		for _, id := range c.Lists {
			checked := ""
			for _, d := range c.DefaultLists {
				if d == id {
					checked = " checked"
				}
			}
			w.WriteString(`    <label><input type="checkbox" name="list" value="` + id.String() + `"` + checked + `> ` + esc(byID[id]) + `</label>` + "\n")
		}
		w.WriteString("  </fieldset>\n")
	}
	if c.ConsentText != "" {
		raw := c.ConsentText
		text := esc(raw)
		if href := c.privacy(b); href != "" {
			if i, j := PhraseIndex(raw, c.Labels.Text("form.privacy", nil)); i >= 0 {
				text = esc(raw[:i]) + `<a href="` + esc(href) + `">` + esc(raw[i:j]) + `</a>` + esc(raw[j:])
			}
		}
		w.WriteString(`  <label><input type="checkbox" name="consent" value="1" required> ` + text + `</label>` + "\n")
	}
	if c.Spam.Honeypot {
		w.WriteString(`  <div style="position:absolute;left:-10000px" aria-hidden="true"><input type="text" name="website" tabindex="-1" autocomplete="off"></div>` + "\n")
	}
	w.WriteString(`  <button type="submit">` + esc(c.Content.Button) + `</button>` + "\n</form>")
	return w.String()
}
