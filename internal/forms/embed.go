package forms

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"html"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
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
	Name    string `json:"name"`
	Checked bool   `json:"checked"`
}

type embedMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type embedConfig struct {
	ID          string                  `json:"id"`
	Kind        string                  `json:"kind"`
	Submit      string                  `json:"submit"`
	Session     string                  `json:"session"`
	CSS         string                  `json:"css"`
	Hosted      string                  `json:"hosted"`
	Accent      string                  `json:"accent"`
	Hover       string                  `json:"hover"`
	OnAccent    string                  `json:"onAccent"`
	Link        string                  `json:"link"`
	Publisher   string                  `json:"publisher"`
	Initial     string                  `json:"initial"`
	Logo        string                  `json:"logo"`
	Heading     string                  `json:"heading"`
	Description string                  `json:"description"`
	Button      string                  `json:"button"`
	Footnote    string                  `json:"footnote"`
	ListsLabel  string                  `json:"listsLabel"`
	Fields      []embedField            `json:"fields"`
	Lists       []embedList             `json:"lists"`
	ListChoice  bool                    `json:"listChoice"`
	Consent     string                  `json:"consent"`
	Privacy     string                  `json:"privacy"`
	Honeypot    bool                    `json:"honeypot"`
	Popup       map[string]int          `json:"popup"`
	Messages    map[string]embedMessage `json:"messages"`
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

// Messages after a submission, for the script and the hosted page.
func doneMessages(publisher string) map[string]embedMessage {
	return map[string]embedMessage{
		OutcomeConfirm:    {Title: "Check your inbox", Text: "We sent you a link to confirm your subscription. It only takes a click."},
		OutcomeSubscribed: {Title: "You're subscribed", Text: "Thanks for joining " + publisher + ". The next issue is on its way to your inbox."},
		"error":           {Title: "", Text: msgSaveFailed},
	}
}

// EmbedScript is the script of a form (/p/{workspace}/forms/{id}.js): the
// renderer with the form's configuration. It holds no personal data.
func EmbedScript(c Config, links PageLinks, b Branding, lists []audience.PublicList, customs []store.CustomField) []byte {
	cfg := embedConfig{
		ID: c.PublicID, Kind: c.Kind, Submit: links.Submit(c.PublicID), Session: links.FormSession(c.PublicID),
		CSS: links.Static("embed.css?v=" + EmbedCSSVersion), Hosted: links.PlainAction(c.Slug),
		Accent: b.Palette.Accent, Hover: b.Palette.Hover, OnAccent: b.Palette.OnAccent, Link: b.Palette.Text,
		Publisher: b.PublisherName, Initial: b.Initial(), Heading: c.Content.Heading, Description: c.Content.Description,
		Button: c.Content.Button, Footnote: c.Content.Footnote, ListsLabel: c.Content.ListsLabel, ListChoice: c.ListChoice,
		Consent: c.ConsentText, Privacy: b.PrivacyURL, Honeypot: c.Spam.Honeypot,
		Popup:    map[string]int{"delay": c.Content.PopupDelay, "scroll": c.Content.PopupScroll, "repeatDays": c.Content.PopupRepeatDays},
		Messages: doneMessages(b.PublisherName), Fields: []embedField{}, Lists: []embedList{},
	}
	if b.Logo != nil {
		cfg.Logo = links.Logo(b.Logo.Hash)
	}
	for _, f := range c.Fields {
		ef := embedField{Key: f.Key, Name: FieldName(f.Key), Label: FieldLabel(f, customs), Type: "text", Required: f.Required || f.Key == FieldEmail, Options: []string{}}
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
		cfg.Fields = append(cfg.Fields, ef)
	}
	byID := map[uuid.UUID]audience.PublicList{}
	for _, l := range lists {
		byID[l.ID] = l
	}
	for _, id := range c.Lists {
		if l, ok := byID[id]; ok {
			checked := false
			for _, d := range c.DefaultLists {
				if d == id {
					checked = true
				}
			}
			cfg.Lists = append(cfg.Lists, embedList{ID: id.String(), Name: l.Name, Checked: checked})
		}
	}
	raw, _ := json.Marshal(cfg) // escapes <, > and & for script contexts
	return []byte(strings.Replace(embedScript, "__RC_FORM__", string(raw), 1))
}

// PlainAction is where the plain HTML form posts: always under /p/ so the
// cross-site post reaches the form whichever site it sits on.
func (l PageLinks) PlainAction(slug string) string {
	return l.origin + "/p/" + l.slug + "/subscribe/" + slug
}

// PlainHTML is the form as plain HTML for sites that can't load scripts.
// It posts to the hosted page, which answers with the result page.
func PlainHTML(c Config, links PageLinks, lists []audience.PublicList, customs []store.CustomField, b Branding) string {
	esc := html.EscapeString
	var w bytes.Buffer
	w.WriteString(`<form action="` + esc(links.PlainAction(c.Slug)) + `" method="post" accept-charset="utf-8">` + "\n")
	w.WriteString(`  <input type="hidden" name="_plain" value="1">` + "\n")
	for _, f := range c.Fields {
		label := esc(FieldLabel(f, customs))
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
		byID[l.ID] = l.Name
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
		text := esc(c.ConsentText)
		if b.PrivacyURL != "" {
			if i := strings.Index(strings.ToLower(text), "privacy policy"); i >= 0 {
				text = text[:i] + `<a href="` + esc(b.PrivacyURL) + `">` + text[i:i+14] + `</a>` + text[i+14:]
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
