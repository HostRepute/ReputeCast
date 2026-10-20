package forms

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// brandingDoc is the public-page part of workspaces.branding. Other keys
// (the postal address of the email footer) belong to other screens and are
// never touched here.
type brandingDoc struct {
	PublisherName    string    `json:"publisherName,omitempty"`
	Accent           string    `json:"accent,omitempty"`
	Background       string    `json:"background,omitempty"`
	AccentHover      string    `json:"accentHover,omitempty"`
	AccentSoft       string    `json:"accentSoft,omitempty"`
	AccentText       string    `json:"accentText,omitempty"`
	OnAccent         string    `json:"onAccent,omitempty"`
	Font             string    `json:"font,omitempty"`
	CustomFontFamily string    `json:"customFontFamily,omitempty"`
	CustomFontURL    string    `json:"customFontUrl,omitempty"`
	PoweredBy        *bool     `json:"poweredBy,omitempty"`
	PrivacyURL       string    `json:"privacyUrl,omitempty"`
	Logo             *logoMeta `json:"logo,omitempty"`
}

type logoMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Hash string `json:"hash"`
}

// Branding is how a workspace's public pages look, with defaults filled.
type Branding struct {
	PublisherName    string
	Palette          Palette
	Font             string
	CustomFontFamily string
	CustomFontURL    string
	PoweredBy        bool
	PrivacyURL       string
	Logo             *logoMeta
}

func brandingOf(ws *store.Workspace, raw json.RawMessage) Branding {
	var d brandingDoc
	_ = json.Unmarshal(raw, &d)
	b := Branding{PublisherName: d.PublisherName, Font: d.Font, CustomFontFamily: d.CustomFontFamily, CustomFontURL: d.CustomFontURL,
		PoweredBy: d.PoweredBy == nil || *d.PoweredBy, PrivacyURL: d.PrivacyURL, Logo: d.Logo}
	if b.PublisherName == "" {
		b.PublisherName = ws.Name
	}
	if b.Font == "" {
		b.Font = "plex"
	}
	accent, bg := d.Accent, d.Background
	if accent == "" {
		accent = defaultBrand
	}
	if bg == "" {
		bg = defaultTint
	}
	p, err := DerivePalette(accent, bg)
	if err != nil {
		p, _ = DerivePalette(defaultBrand, defaultTint)
	}
	// Stored derived colours win: they are what was saved (and what the
	// design's sample publisher uses).
	if d.AccentHover != "" && d.Accent != "" {
		p.Hover, p.Soft, p.Text, p.OnAccent = d.AccentHover, nz(d.AccentSoft, p.Soft), nz(d.AccentText, p.Text), nz(d.OnAccent, p.OnAccent)
	}
	b.Palette = p
	return b
}

func nz(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// Initial is the letter on the logo tile when there is no logo.
func (b Branding) Initial() string {
	for _, r := range strings.TrimSpace(b.PublisherName) {
		return strings.ToUpper(string(r))
	}
	return "N"
}

// FontStack is the CSS font-family of the pages.
func (b Branding) FontStack() string {
	switch b.Font {
	case "system":
		return "system-ui,-apple-system,'Segoe UI',Roboto,sans-serif"
	case "custom":
		if b.CustomFontFamily != "" {
			return cssString(b.CustomFontFamily) + ",'IBM Plex Sans',system-ui,sans-serif"
		}
	}
	return "'IBM Plex Sans',system-ui,sans-serif"
}

// cssString quotes a font family name for CSS.
func cssString(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '\'' || r == '"' || r == '\\' || r == ';' || r == '{' || r == '}' || r == '<' || r == '>' || r < 0x20 {
			return -1
		}
		return r
	}, s)
	return "'" + s + "'"
}

// FontFace is the @font-face rule of a custom web font ("" otherwise).
func (b Branding) FontFace() string {
	if b.FontOrigin() == "" || b.CustomFontFamily == "" {
		return ""
	}
	u := strings.NewReplacer(`"`, "%22", `\`, "%5C", "\n", "", "\r", "", "(", "%28", ")", "%29").Replace(b.CustomFontURL)
	format := "woff2"
	if strings.HasSuffix(strings.ToLower(u), ".woff") {
		format = "woff"
	}
	return "@font-face{font-family:" + cssString(b.CustomFontFamily) + ";src:url(\"" + u + "\") format('" + format + "');font-display:swap}\n"
}

// FontOrigin is the origin a custom web font loads from ("" for the built-in fonts).
func (b Branding) FontOrigin() string {
	if b.Font != "custom" || b.CustomFontURL == "" {
		return ""
	}
	u, err := url.Parse(b.CustomFontURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	return "https://" + u.Host
}

// PublicBrandingLogo is the uploaded logo.
type PublicBrandingLogo struct {
	Name string `json:"name" example:"acme-mark.svg"`
	Type string `json:"type" example:"image/svg+xml"`
	URL  string `json:"url"`
}

// PublicBranding is the Branding card.
type PublicBranding struct {
	PublisherName    string              `json:"publisherName"`
	Accent           string              `json:"accent"`
	Background       string              `json:"background"`
	AccentHover      string              `json:"accentHover"`
	AccentSoft       string              `json:"accentSoft"`
	AccentText       string              `json:"accentText"`
	OnAccent         string              `json:"onAccent"`
	Font             string              `json:"font" enum:"plex,system,custom"`
	CustomFontFamily string              `json:"customFontFamily"`
	CustomFontURL    string              `json:"customFontUrl"`
	PoweredBy        bool                `json:"poweredBy" doc:"Show \"Sent with ReputeCast\" in the footer"`
	PrivacyURL       string              `json:"privacyUrl"`
	Logo             *PublicBrandingLogo `json:"logo,omitempty"`
	Initial          string              `json:"initial" doc:"The letter on the logo tile when there is no logo"`
	// Notes say what was adjusted to keep text readable.
	Notes []string `json:"notes"`
}

func (b Branding) public() PublicBranding {
	out := PublicBranding{PublisherName: b.PublisherName, Accent: b.Palette.Accent, Background: b.Palette.Background, AccentHover: b.Palette.Hover,
		AccentSoft: b.Palette.Soft, AccentText: b.Palette.Text, OnAccent: b.Palette.OnAccent, Font: b.Font, CustomFontFamily: b.CustomFontFamily,
		CustomFontURL: b.CustomFontURL, PoweredBy: b.PoweredBy, PrivacyURL: b.PrivacyURL, Initial: b.Initial(), Notes: b.Palette.Notes}
	if out.Notes == nil {
		out.Notes = []string{}
	}
	return out
}

// brandingOut is the branding for the app, the logo served by the instance
// (the custom domain may not resolve where the app runs).
func (s *Service) brandingOut(ws *store.Workspace, set *Settings) PublicBranding {
	out := set.Branding.public()
	if l := set.Branding.Logo; l != nil {
		out.Logo = &PublicBrandingLogo{Name: l.Name, Type: l.Type, URL: LinksOn(s.URLs.Base(), ws.Slug, false).Logo(l.Hash)}
	}
	return out
}

// SubscribePage are the hosted subscribe page's options.
type SubscribePage struct {
	Heading     string `json:"heading" maxLength:"120"`
	Description string `json:"description" maxLength:"600"`
	Footnote    string `json:"footnote" maxLength:"300"`
	// ListText is the line under each list (by list id); {subscribers} is
	// replaced by the list's size.
	ListText map[string]string `json:"listText"`
	// Index lets search engines list the page.
	Index bool `json:"index"`
}

// ConfirmPage are the options of the page after the confirmation click.
type ConfirmPage struct {
	// ListText says what each list sends and when ("Tuesdays at 09:00 your
	// time."); the next scheduled issue is added after it.
	ListText map[string]string `json:"listText"`
	// OfferText names what a list the contact isn't on brings ("release
	// news"): "Add it if you'd like release news too."
	OfferText map[string]string `json:"offerText"`
	// InboxTip asks to add the sender to the contacts.
	InboxTip bool `json:"inboxTip"`
}

// PreferencesPage are the preference center's options.
type PreferencesPage struct {
	// Lists offered (by id); empty offers every public list.
	Lists     []string          `json:"lists"`
	ListText  map[string]string `json:"listText"`
	Frequency bool              `json:"frequency"`
	Pause     bool              `json:"pause"`
	FirstName bool              `json:"firstName"`
}

// UnsubscribePage are the unsubscribe page's options.
type UnsubscribePage struct {
	Fewer       bool   `json:"fewer" doc:"Offer the monthly best-of instead"`
	FewerTitle  string `json:"fewerTitle" maxLength:"120"`
	FewerText   string `json:"fewerText" maxLength:"400"`
	FewerButton string `json:"fewerButton" maxLength:"60"`
	Reasons     bool   `json:"reasons" doc:"Ask why, optionally"`
}

// ArchivePage are the public archive's options.
type ArchivePage struct {
	Enabled bool `json:"enabled"`
	// Lists whose sent campaigns are published (by id); empty uses every public list.
	Lists        []string `json:"lists"`
	SubscribeBox bool     `json:"subscribeBox"`
	Index        bool     `json:"index"`
}

// Pages are every public page's options.
type Pages struct {
	Subscribe   SubscribePage   `json:"subscribe"`
	Confirm     ConfirmPage     `json:"confirm"`
	Preferences PreferencesPage `json:"preferences"`
	Unsubscribe UnsubscribePage `json:"unsubscribe"`
	Archive     ArchivePage     `json:"archive"`
	// PreviewEmail is the subscriber the app previews the pages for; empty
	// uses whoever confirmed a subscription last.
	PreviewEmail string `json:"previewEmail"`
}

func defaultPages() Pages {
	return Pages{
		Subscribe:   SubscribePage{ListText: map[string]string{}, Index: true, Footnote: "We'll send a confirmation link first. No spam, ever. You can change lists or pause emails from any message."},
		Confirm:     ConfirmPage{ListText: map[string]string{}, OfferText: map[string]string{}, InboxTip: true},
		Preferences: PreferencesPage{Lists: []string{}, ListText: map[string]string{}, Frequency: true, Pause: true, FirstName: true},
		Unsubscribe: UnsubscribePage{Fewer: true, FewerTitle: "Too many emails? Get fewer instead.",
			FewerText:   "Switch to one monthly best-of and nothing else. Most people who pick this stay subscribed.",
			FewerButton: "Send me the monthly best-of only", Reasons: true},
		Archive: ArchivePage{Enabled: true, Lists: []string{}, SubscribeBox: true, Index: true},
	}
}

func pagesOf(raw json.RawMessage) Pages {
	p := defaultPages()
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	if p.Subscribe.ListText == nil {
		p.Subscribe.ListText = map[string]string{}
	}
	if p.Confirm.ListText == nil {
		p.Confirm.ListText = map[string]string{}
	}
	if p.Confirm.OfferText == nil {
		p.Confirm.OfferText = map[string]string{}
	}
	if p.Preferences.ListText == nil {
		p.Preferences.ListText = map[string]string{}
	}
	if p.Preferences.Lists == nil {
		p.Preferences.Lists = []string{}
	}
	if p.Archive.Lists == nil {
		p.Archive.Lists = []string{}
	}
	return p
}

// Settings are a workspace's public page settings.
type Settings struct {
	Workspace *store.Workspace
	row       store.PublicPageSetting
	Branding  Branding
	Pages     Pages
}

type pageSettings = Settings

// CopyLanguage is the language the pages are written in.
func (s *Settings) CopyLanguage() string {
	if s.row.CopyLanguage == "" {
		return "en"
	}
	return s.row.CopyLanguage
}

// HostedFormID is the form behind /subscribe, if any.
func (s *Settings) HostedFormID() *uuid.UUID { return s.row.HostedFormID }

// DomainState is the custom domain as set, verified or not, with its
// check's status and finding (polls warn while answer links can't use it).
func (s *Settings) DomainState() (domain, status, detail string) {
	if s.row.CustomDomain == nil {
		return "", "", ""
	}
	return *s.row.CustomDomain, s.row.DomainStatus, s.row.DomainDetail
}

// CustomDomain is the verified custom domain, or "".
func (s *Settings) CustomDomain() string {
	if s.row.CustomDomain != nil && s.row.DomainStatus == "verified" {
		return *s.row.CustomDomain
	}
	return ""
}

// DomainDates are when the custom domain's DNS was last checked and when it
// was verified (the publication website, design expansion phase 7).
func (s *Settings) DomainDates() (checkedAt, verifiedAt *time.Time) {
	return s.row.DomainCheckedAt, s.row.DomainVerifiedAt
}

// Path is an address on the pages' host: "/posts/x" on a custom domain,
// "/p/<workspace>/posts/x" on the instance (the publication website).
func (l PageLinks) Path(parts ...string) string { return l.page(parts...) }

// TurnstileSiteKey is the public site key when Turnstile can be used.
func (s *Settings) TurnstileSiteKey() string {
	if s.row.TurnstileSiteKey != "" && len(s.row.TurnstileSecretEnc) > 0 && s.row.TurnstileStatus == "ok" {
		return s.row.TurnstileSiteKey
	}
	return ""
}

// LogoVersion changes when the logo changes (cache busting).
func (s *Settings) LogoVersion() string {
	if s.Branding.Logo == nil {
		return ""
	}
	return s.Branding.Logo.Hash
}

// UpdatedAt is when the settings last changed (theme cache busting).
func (s *Settings) UpdatedAt() time.Time { return s.row.UpdatedAt }

// LoadSettings reads (and creates on first use) a workspace's settings.
func (s *Service) LoadSettings(ctx context.Context, ws *store.Workspace) (*Settings, error) {
	return s.settings(ctx, ws)
}

func (s *Service) settings(ctx context.Context, ws *store.Workspace) (*Settings, error) {
	row, err := s.Q.PublicSettingsGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := s.Q.PublicSettingsEnsure(ctx, ws.ID); err != nil {
			return nil, err
		}
		row, err = s.Q.PublicSettingsGet(ctx, ws.ID)
	}
	if err != nil {
		return nil, err
	}
	raw, err := s.Q.FormsWorkspaceBranding(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	return &Settings{Workspace: ws, row: row, Branding: brandingOf(ws, raw), Pages: pagesOf(row.Pages)}, nil
}

// PageLinks builds the links of a workspace's public pages, either on its
// verified custom domain (news.acme.io/subscribe) or on the instance
// (reputecast.acme.io/p/acme-newsletter/subscribe). Form scripts and
// submissions always use /p/{workspace}/forms/… on whichever host.
type PageLinks struct {
	origin string
	prefix string
	slug   string
	Custom bool
}

// LinksOn returns links on an origin: the custom domain's ("https://news.acme.io")
// or the instance's.
func LinksOn(origin, slug string, custom bool) PageLinks {
	l := PageLinks{origin: strings.TrimRight(origin, "/"), slug: slug, Custom: custom}
	if !custom {
		l.prefix = "/p/" + url.PathEscape(slug)
	}
	return l
}

// Links are the preferred (canonical) links of a workspace: its verified
// custom domain when it has one.
func (s *Service) Links(ws *store.Workspace, set *Settings) PageLinks {
	if d := set.CustomDomain(); d != "" {
		return LinksOn(customScheme(s.URLs)+d, ws.Slug, true)
	}
	return LinksOn(s.URLs.Base(), ws.Slug, false)
}

// customScheme is https, unless the instance itself runs on plain HTTP
// (development).
func customScheme(b publicurl.Builder) string {
	if strings.HasPrefix(b.Base(), "http://") {
		return "http://"
	}
	return "https://"
}

func (l PageLinks) page(parts ...string) string {
	var b strings.Builder
	b.WriteString(l.origin)
	b.WriteString(l.prefix)
	for _, p := range parts {
		b.WriteByte('/')
		b.WriteString(url.PathEscape(p))
	}
	return b.String()
}

// Origin is "https://news.acme.io".
func (l PageLinks) Origin() string { return l.origin }

// Home is the root of the pages ("/" on a custom domain, the subscribe page otherwise).
func (l PageLinks) Home() string { return l.page("subscribe") }

// Subscribe is the hosted page of a form ("" for the workspace's page).
func (l PageLinks) Subscribe(form string) string {
	if form == "" {
		return l.page("subscribe")
	}
	return l.page("subscribe", form)
}

// Challenge receives Turnstile answers.
func (l PageLinks) Challenge() string { return l.page("subscribe", "challenge") }

// Poll is a poll's answer page, and Survey a survey's page (internal/polls).
func (l PageLinks) Poll(token string) string  { return l.page("poll", token) }
func (l PageLinks) Survey(slug string) string { return l.page("s", slug) }

// Confirm, Preferences, Unsubscribe and View are the token pages.
func (l PageLinks) Confirm(token string) string     { return l.page("confirm", token) }
func (l PageLinks) Preferences(token string) string { return l.page("preferences", token) }

// PreferencesRequest asks for a preference link by email.
func (l PageLinks) PreferencesRequest() string      { return l.page("preferences") }
func (l PageLinks) Unsubscribe(token string) string { return l.page("unsubscribe", token) }
func (l PageLinks) View(token string) string        { return l.page("view", token) }
func (l PageLinks) Archive() string                 { return l.page("archive") }
func (l PageLinks) Issue(slug string) string        { return l.page("archive", slug) }
func (l PageLinks) Feed() string                    { return l.page("archive.rss") }
func (l PageLinks) Sitemap() string                 { return l.page("sitemap.xml") }

// SmsTerms is the text message terms page (SMS, X21).
func (l PageLinks) SmsTerms() string { return l.page("sms-terms") }
func (l PageLinks) forms(rest string) string {
	return l.origin + "/p/" + url.PathEscape(l.slug) + "/forms/" + rest
}
func (l PageLinks) Script(publicID string) string { return l.forms(url.PathEscape(publicID) + ".js") }
func (l PageLinks) Submit(publicID string) string {
	return l.forms(url.PathEscape(publicID) + "/submit")
}
func (l PageLinks) FormSession(publicID string) string {
	return l.forms(url.PathEscape(publicID) + "/session")
}

// Theme is the workspace's colours and font as a stylesheet.
func (l PageLinks) Theme(version string) string {
	return l.origin + "/p/" + url.PathEscape(l.slug) + "/theme.css?v=" + url.QueryEscape(version)
}

// Logo is the uploaded logo.
func (l PageLinks) Logo(version string) string {
	return l.origin + "/p/" + url.PathEscape(l.slug) + "/logo?v=" + url.QueryEscape(version)
}

// Static is a file of the shared public assets (stylesheet, fonts, scripts).
func (l PageLinks) Static(file string) string { return l.origin + "/p/_/" + file }

// TurnstileError receives errors of the Turnstile widget.
func (l PageLinks) TurnstileError() string {
	return l.origin + "/p/" + url.PathEscape(l.slug) + "/turnstile-error"
}

// OwnOrigins are the hosts that may always submit a form: the instance and
// the verified custom domain.
func (s *Service) OwnOrigins(set *Settings) []string {
	out := []string{}
	if u, err := url.Parse(s.URLs.Base()); err == nil && u.Host != "" {
		out = append(out, strings.ToLower(u.Host))
	}
	if d := set.CustomDomain(); d != "" {
		out = append(out, d)
	}
	return out
}

// PublicTurnstile is the Turnstile set-up (the secret is never returned).
type PublicTurnstile struct {
	SiteKey   string     `json:"siteKey"`
	HasSecret bool       `json:"hasSecret"`
	Status    string     `json:"status" enum:"off,ok,error"`
	Error     string     `json:"error" doc:"Why Cloudflare rejected the keys, with its error code"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
}

func (s *Settings) turnstile() PublicTurnstile {
	return PublicTurnstile{SiteKey: s.row.TurnstileSiteKey, HasSecret: len(s.row.TurnstileSecretEnc) > 0, Status: s.row.TurnstileStatus,
		Error: s.row.TurnstileError, CheckedAt: s.row.TurnstileCheckedAt}
}

// PublicCustomDomain is the custom domain and its DNS check.
type PublicCustomDomain struct {
	Domain     string     `json:"domain" example:"news.acme.io"`
	Status     string     `json:"status" enum:"none,pending,verified,failed"`
	Detail     string     `json:"detail"`
	CheckedAt  *time.Time `json:"checkedAt,omitempty"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
	// Target is the host the domain's CNAME record points at.
	Target string `json:"target" example:"reputecast.acme.io"`
	// TXTName and TXTValue are the alternative proof for domains that can't use a CNAME.
	TXTName  string `json:"txtName"`
	TXTValue string `json:"txtValue"`
}

func (s *Settings) domain(urls publicurl.Builder) PublicCustomDomain {
	out := PublicCustomDomain{Status: s.row.DomainStatus, Detail: s.row.DomainDetail, CheckedAt: s.row.DomainCheckedAt, VerifiedAt: s.row.DomainVerifiedAt, Target: instanceHost(urls)}
	if s.row.CustomDomain != nil {
		out.Domain = *s.row.CustomDomain
		out.TXTName = "_reputecast." + out.Domain
	}
	return out
}

func instanceHost(urls publicurl.Builder) string {
	if u, err := url.Parse(urls.Base()); err == nil {
		return strings.ToLower(u.Hostname())
	}
	return ""
}

// PublicPageList is a list as the page settings show it.
type PublicPageList struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
	Subscribed  int64  `json:"subscribed"`
}

// PublicPageURLs are where the pages live.
type PublicPageURLs struct {
	Subscribe   string `json:"subscribe"`
	Preferences string `json:"preferences" doc:"Asks for a preference link by email"`
	Archive     string `json:"archive"`
	Feed        string `json:"feed"`
	Sitemap     string `json:"sitemap"`
}

// PublicPagesSettings feeds the Subscribe page, Preference center,
// Unsubscribe page, Archive and Branding tabs.
type PublicPagesSettings struct {
	Branding PublicBranding     `json:"branding"`
	Domain   PublicCustomDomain `json:"domain"`
	Pages    Pages              `json:"pages"`
	// HostedFormID is the form behind the subscribe page; empty uses the built-in form.
	HostedFormID string              `json:"hostedFormId"`
	Forms        []SignupFormSummary `json:"forms"`
	Lists        []PublicPageList    `json:"lists"`
	URLs         PublicPageURLs      `json:"urls"`
	Turnstile    PublicTurnstile     `json:"turnstile"`
	// SenderEmail is the default sender, for the inbox tip on the confirmation page.
	SenderEmail string    `json:"senderEmail"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PublicPages returns every public page setting.
func (s *Service) PublicPages(ctx context.Context, ws *store.Workspace) (PublicPagesSettings, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return PublicPagesSettings{}, err
	}
	links := s.Links(ws, set)
	dom := set.domain(s.URLs)
	if dom.Domain != "" {
		dom.TXTValue = s.domainToken(ws, dom.Domain)
	}
	out := PublicPagesSettings{Branding: set.Branding.public(), Domain: dom, Pages: set.Pages, Forms: []SignupFormSummary{}, Lists: []PublicPageList{},
		URLs:      PublicPageURLs{Subscribe: links.Subscribe(""), Preferences: links.PreferencesRequest(), Archive: links.Archive(), Feed: links.Feed(), Sitemap: links.Sitemap()},
		Turnstile: set.turnstile(), UpdatedAt: set.row.UpdatedAt}
	out.Branding = s.brandingOut(ws, set)
	if set.row.HostedFormID != nil {
		out.HostedFormID = set.row.HostedFormID.String()
	}
	forms, err := s.Q.FormsList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, f := range forms {
		out.Forms = append(out.Forms, s.summary(f, set, links))
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, PublicPageList{ID: l.ID.String(), Name: l.Name, Description: l.Description, Public: l.Public, Subscribed: l.Subscribed})
	}
	if sender, err := s.Q.FormsDefaultSender(ctx, ws.ID); err == nil {
		out.SenderEmail = sender.Email
	}
	return out, nil
}

// PublicBrandingInput saves the Branding card.
type PublicBrandingInput struct {
	PublisherName    string `json:"publisherName" minLength:"1" maxLength:"80"`
	Accent           string `json:"accent" pattern:"^#[0-9a-fA-F]{6}$" example:"#4338CA"`
	Background       string `json:"background" pattern:"^#[0-9a-fA-F]{6}$" example:"#F5F3FF"`
	Font             string `json:"font" enum:"plex,system,custom"`
	CustomFontFamily string `json:"customFontFamily,omitempty" maxLength:"80"`
	CustomFontURL    string `json:"customFontUrl,omitempty" maxLength:"500" doc:"https URL of a .woff2 or .woff file"`
	PoweredBy        bool   `json:"poweredBy"`
	PrivacyURL       string `json:"privacyUrl,omitempty" maxLength:"500"`
}

// SaveBranding stores the branding and the colours derived from it.
func (s *Service) SaveBranding(ctx context.Context, ws *store.Workspace, in PublicBrandingInput) (PublicPagesSettings, error) {
	name := strings.TrimSpace(in.PublisherName)
	if name == "" {
		return PublicPagesSettings{}, api.Invalid("body.publisherName", "Name the publisher: it heads every public page.")
	}
	pal, err := DerivePalette(in.Accent, in.Background)
	if err != nil {
		loc := "body.accent"
		if strings.Contains(err.Error(), "background") {
			loc = "body.background"
		}
		return PublicPagesSettings{}, api.Invalid(loc, capitalizeFirst(err.Error())+".")
	}
	if in.Font == "custom" {
		u, err := url.Parse(strings.TrimSpace(in.CustomFontURL))
		if strings.TrimSpace(in.CustomFontFamily) == "" {
			return PublicPagesSettings{}, api.Invalid("body.customFontFamily", "Name the font family, as in the font's CSS.")
		}
		fontFile := err == nil && (strings.HasSuffix(u.Path, ".woff2") || strings.HasSuffix(u.Path, ".woff"))
		if !fontFile || u.Scheme != "https" || u.Host == "" {
			return PublicPagesSettings{}, api.Invalid("body.customFontUrl", "Link the font file itself: an https address ending in .woff2 or .woff.")
		}
	}
	privacy := strings.TrimSpace(in.PrivacyURL)
	if privacy != "" {
		u, err := url.Parse(privacy)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return PublicPagesSettings{}, api.Invalid("body.privacyUrl", "Use a full address like https://acme.io/privacy.")
		}
	}
	powered := in.PoweredBy
	patch := brandingDoc{PublisherName: name, Accent: pal.Accent, Background: pal.Background, AccentHover: pal.Hover, AccentSoft: pal.Soft, AccentText: pal.Text,
		OnAccent: pal.OnAccent, Font: in.Font, CustomFontFamily: strings.TrimSpace(in.CustomFontFamily), CustomFontURL: strings.TrimSpace(in.CustomFontURL),
		PoweredBy: &powered, PrivacyURL: privacy}
	if in.Font != "custom" {
		patch.CustomFontFamily, patch.CustomFontURL = "", ""
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.FormsWorkspaceBranding(ctx, ws.ID)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(patch)
		// Empty optional keys are removed rather than stored empty.
		var drop []string
		for _, k := range []string{"customFontFamily", "customFontUrl", "privacyUrl"} {
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			if _, ok := m[k]; !ok {
				drop = append(drop, k)
			}
		}
		if err := q.FormsRemoveBrandingKeys(ctx, store.FormsRemoveBrandingKeysParams{ID: ws.ID, Keys: drop}); err != nil {
			return err
		}
		after, err := q.FormsUpdateBranding(ctx, store.FormsUpdateBrandingParams{ID: ws.ID, Patch: raw})
		if err != nil {
			return err
		}
		if err := s.touchSettings(ctx, q, ws); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.branding", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: publicBrandingKeys(before), After: publicBrandingKeys(after)})
	})
	if err != nil {
		return PublicPagesSettings{}, err
	}
	return s.PublicPages(ctx, ws)
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// publicBrandingKeys is the public-page part of a branding document, for the audit log.
func publicBrandingKeys(raw json.RawMessage) map[string]any {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	out := map[string]any{}
	for _, k := range []string{"publisherName", "accent", "background", "font", "customFontFamily", "customFontUrl", "poweredBy", "privacyUrl", "logo"} {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}

// touchSettings moves the settings' updated_at, which versions the theme stylesheet.
func (s *Service) touchSettings(ctx context.Context, q *store.Queries, ws *store.Workspace) error {
	if err := q.PublicSettingsEnsure(ctx, ws.ID); err != nil {
		return err
	}
	return q.PublicSettingsTouch(ctx, store.PublicSettingsTouchParams{WorkspaceID: ws.ID, UpdatedBy: actor(ctx), At: s.now()})
}

// PublicPageInput saves one page's options; only the page named in the path is read.
type PublicPageInput struct {
	Subscribe   *SubscribePage   `json:"subscribe,omitempty"`
	Confirm     *ConfirmPage     `json:"confirm,omitempty"`
	Preferences *PreferencesPage `json:"preferences,omitempty"`
	Unsubscribe *UnsubscribePage `json:"unsubscribe,omitempty"`
	Archive     *ArchivePage     `json:"archive,omitempty"`
	// HostedFormID picks the form behind the subscribe page ("" for the built-in form).
	HostedFormID *string `json:"hostedFormId,omitempty"`
	// PreviewEmail changes whose pages the app previews ("" for the latest confirmation).
	PreviewEmail *string `json:"previewEmail,omitempty" maxLength:"320"`
}

// SavePage stores the options of one or more pages.
func (s *Service) SavePage(ctx context.Context, ws *store.Workspace, in PublicPageInput) (PublicPagesSettings, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PublicSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		row, err := q.PublicSettingsGet(ctx, ws.ID)
		if err != nil {
			return err
		}
		before := pagesOf(row.Pages)
		lists, err := s.Audience.PublicLists(ctx, q, ws)
		if err != nil {
			return err
		}
		after, changed, err := applyPages(before, in, lists)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(after)
		if err := q.PublicSettingsSetPages(ctx, store.PublicSettingsSetPagesParams{WorkspaceID: ws.ID, Pages: raw, UpdatedBy: actor(ctx), At: s.now()}); err != nil {
			return err
		}
		// New or changed text leaves the other languages' copy to catch up.
		if err := s.syncPageCopy(ctx, q, ws); err != nil {
			return err
		}
		var hostedBefore, hostedAfter string
		if row.HostedFormID != nil {
			hostedBefore = row.HostedFormID.String()
		}
		hostedAfter = hostedBefore
		if in.HostedFormID != nil {
			var id *uuid.UUID
			if *in.HostedFormID != "" {
				f, err := s.form(ctx, q, ws, *in.HostedFormID)
				if err != nil {
					return api.Invalid("body.hostedFormId", "Pick a form of this workspace.")
				}
				id = &f.ID
				hostedAfter = f.ID.String()
			} else {
				hostedAfter = ""
			}
			if err := q.PublicSettingsSetHostedForm(ctx, store.PublicSettingsSetHostedFormParams{WorkspaceID: ws.ID, HostedFormID: id, UpdatedBy: actor(ctx), At: s.now()}); err != nil {
				return err
			}
			changed = append(changed, "hostedForm")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.update", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"pages": before, "hostedFormId": hostedBefore}, After: map[string]any{"pages": after, "hostedFormId": hostedAfter},
			Metadata: map[string]any{"changed": changed}})
	})
	if err != nil {
		return PublicPagesSettings{}, err
	}
	return s.PublicPages(ctx, ws)
}

// applyPages lays the pages named in in over before, cleaned as they are
// saved: trimmed text, lists of the workspace only.
func applyPages(before Pages, in PublicPageInput, lists []audience.PublicList) (Pages, []string, error) {
	after := before
	known := map[string]bool{}
	for _, l := range lists {
		known[l.ID.String()] = true
	}
	cleanText := func(m map[string]string, loc string) (map[string]string, error) {
		out := map[string]string{}
		for k, v := range m {
			if !known[k] {
				continue
			}
			v = strings.TrimSpace(v)
			if len(v) > 300 {
				return nil, api.Invalid(loc, "Keep each list's line under 300 characters.")
			}
			if v != "" {
				out[k] = v
			}
		}
		return out, nil
	}
	cleanLists := func(ids []string, loc string) ([]string, error) {
		out := []string{}
		for _, id := range ids {
			if !known[id] {
				return nil, api.Invalid(loc, "Pick lists from this workspace.")
			}
			if !slices.Contains(out, id) {
				out = append(out, id)
			}
		}
		return out, nil
	}
	var err error
	changed := []string{}
	if in.Subscribe != nil {
		p := *in.Subscribe
		p.Heading, p.Description, p.Footnote = strings.TrimSpace(p.Heading), strings.TrimSpace(p.Description), strings.TrimSpace(p.Footnote)
		if p.ListText, err = cleanText(p.ListText, "body.subscribe.listText"); err != nil {
			return before, nil, err
		}
		after.Subscribe = p
		changed = append(changed, "subscribe")
	}
	if in.Confirm != nil {
		p := *in.Confirm
		if p.ListText, err = cleanText(p.ListText, "body.confirm.listText"); err != nil {
			return before, nil, err
		}
		if p.OfferText, err = cleanText(p.OfferText, "body.confirm.offerText"); err != nil {
			return before, nil, err
		}
		after.Confirm = p
		changed = append(changed, "confirm")
	}
	if in.Preferences != nil {
		p := *in.Preferences
		if p.Lists, err = cleanLists(p.Lists, "body.preferences.lists"); err != nil {
			return before, nil, err
		}
		if p.ListText, err = cleanText(p.ListText, "body.preferences.listText"); err != nil {
			return before, nil, err
		}
		after.Preferences = p
		changed = append(changed, "preferences")
	}
	if in.Unsubscribe != nil {
		p := *in.Unsubscribe
		p.FewerTitle, p.FewerText, p.FewerButton = strings.TrimSpace(p.FewerTitle), strings.TrimSpace(p.FewerText), strings.TrimSpace(p.FewerButton)
		if p.Fewer && (p.FewerTitle == "" || p.FewerButton == "") {
			return before, nil, api.Invalid("body.unsubscribe.fewerButton", "Give the offer a heading and a button, or turn it off.")
		}
		after.Unsubscribe = p
		changed = append(changed, "unsubscribe")
	}
	if in.Archive != nil {
		p := *in.Archive
		if p.Lists, err = cleanLists(p.Lists, "body.archive.lists"); err != nil {
			return before, nil, err
		}
		after.Archive = p
		changed = append(changed, "archive")
	}
	if in.PreviewEmail != nil {
		e := strings.ToLower(strings.TrimSpace(*in.PreviewEmail))
		if e != "" && (!strings.Contains(e, "@") || strings.ContainsAny(e, " <>\"")) {
			return before, nil, api.Invalid("body.previewEmail", "Enter a subscriber's email address, or leave it empty.")
		}
		after.PreviewEmail = e
		changed = append(changed, "preview")
	}
	return after, changed, nil
}

// WithDraft is a copy of the settings with unsaved page options laid over
// them, for previews. Nothing is stored.
func (s *Service) WithDraft(ctx context.Context, set *Settings, in PublicPageInput) (*Settings, error) {
	lists, err := s.Audience.PublicLists(ctx, s.Q, set.Workspace)
	if err != nil {
		return nil, err
	}
	pages, _, err := applyPages(set.Pages, in, lists)
	if err != nil {
		return nil, err
	}
	out := *set
	out.Pages = pages
	if in.HostedFormID != nil {
		out.row.HostedFormID = nil
		if id, err := uuid.Parse(*in.HostedFormID); err == nil {
			out.row.HostedFormID = &id
		}
	}
	return &out, nil
}

// UnsubscribeStats are the Unsubscribe page tab's numbers.
type UnsubscribeStats struct {
	Days int `json:"days"`
	// Unsubscribed counts contacts who left from the public pages: the
	// unsubscribe page, one-click, or unticking lists in the preference center.
	Unsubscribed int64 `json:"unsubscribed"`
	OneClick     int64 `json:"oneClick" doc:"Of them, with their mail app's one-click unsubscribe"`
	ChoseFewer   int64 `json:"choseFewer" doc:"Took the monthly best-of instead of leaving"`
	Answered     int64 `json:"answered" doc:"Said why they left"`
}

// UnsubscribeNumbers counts the last 30 days of the unsubscribe page.
func (s *Service) UnsubscribeNumbers(ctx context.Context, ws *store.Workspace) (UnsubscribeStats, error) {
	row, err := s.Q.PublicUnsubscribeStats(ctx, store.PublicUnsubscribeStatsParams{WorkspaceID: ws.ID, Since: s.now().AddDate(0, 0, -30)})
	if err != nil {
		return UnsubscribeStats{}, err
	}
	return UnsubscribeStats{Days: 30, Unsubscribed: row.Unsubscribed, OneClick: row.OneClick, ChoseFewer: row.ChoseFewer, Answered: row.Answered}, nil
}

// svgEventHandler finds event attributes (onload, onmouseover, onbegin…)
// in any spacing.
var svgEventHandler = regexp.MustCompile(`(?i)[\s"'/]on[a-z]+\s*=`)

// MaxLogoBytes bounds an uploaded logo.
const MaxLogoBytes = 256 << 10

// sniffLogo checks the bytes of an uploaded logo and returns its type, or
// what's wrong with the file in words for the person uploading it.
func sniffLogo(data []byte) (ctype, problem string) {
	if len(data) == 0 {
		return "", "The file is empty."
	}
	if len(data) > MaxLogoBytes {
		return "", fmt.Sprintf("The logo is %d KB; the limit is 256 KB. An SVG or a 200 px PNG is plenty.", len(data)>>10)
	}
	head := data[:min(len(data), 512)]
	switch ct := http.DetectContentType(data); ct {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return ct, ""
	}
	lower := bytes.ToLower(head)
	if bytes.Contains(lower, []byte("<svg")) {
		all := bytes.ToLower(data)
		bad := svgEventHandler.Match(all)
		for _, s := range [][]byte{[]byte("<script"), []byte("javascript:"), []byte("<foreignobject"), []byte("<iframe"), []byte("<!entity"), []byte("<embed"), []byte("<object")} {
			bad = bad || bytes.Contains(all, s)
		}
		if bad {
			return "", "This SVG contains scripts or embedded content. Export it again as a plain SVG or a PNG."
		}
		return "image/svg+xml", ""
	}
	return "", "That file isn't an image the pages can show. Upload an SVG, PNG, JPEG, GIF or WebP."
}

// UploadLogo stores the publisher's logo.
func (s *Service) UploadLogo(ctx context.Context, ws *store.Workspace, name string, data []byte) (PublicPagesSettings, error) {
	ctype, problem := sniffLogo(data)
	if problem != "" {
		return PublicPagesSettings{}, api.Invalid("body.file", problem)
	}
	sum := sha256.Sum256(data)
	meta := logoMeta{Name: safeFileName(name, ctype), Type: ctype, Hash: hex.EncodeToString(sum[:8])}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PublicSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.PublicSettingsSetLogo(ctx, store.PublicSettingsSetLogoParams{WorkspaceID: ws.ID, Logo: data, LogoType: ctype, UpdatedBy: actor(ctx), At: s.now()}); err != nil {
			return err
		}
		raw, _ := json.Marshal(map[string]any{"logo": meta})
		if _, err := q.FormsUpdateBranding(ctx, store.FormsUpdateBrandingParams{ID: ws.ID, Patch: raw}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.logo", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			After: map[string]any{"name": meta.Name, "type": ctype, "bytes": len(data)}})
	})
	if err != nil {
		return PublicPagesSettings{}, err
	}
	return s.PublicPages(ctx, ws)
}

// RemoveLogo goes back to the letter tile.
func (s *Service) RemoveLogo(ctx context.Context, ws *store.Workspace) (PublicPagesSettings, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PublicSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.PublicSettingsSetLogo(ctx, store.PublicSettingsSetLogoParams{WorkspaceID: ws.ID, Logo: nil, LogoType: "", UpdatedBy: actor(ctx), At: s.now()}); err != nil {
			return err
		}
		if err := q.FormsRemoveBrandingKeys(ctx, store.FormsRemoveBrandingKeysParams{ID: ws.ID, Keys: []string{"logo"}}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.logo_remove", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name})
	})
	if err != nil {
		return PublicPagesSettings{}, err
	}
	return s.PublicPages(ctx, ws)
}

// Logo returns the logo bytes for the public pages.
func (s *Service) Logo(ctx context.Context, ws *store.Workspace) ([]byte, string, time.Time, error) {
	row, err := s.Q.PublicSettingsLogo(ctx, ws.ID)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return row.Logo, row.LogoType, row.UpdatedAt, nil
}

func safeFileName(name, ctype string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	if i := strings.LastIndex(name, "."); i > 0 {
		name = name[:i]
	}
	ext := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp", "image/svg+xml": ".svg"}[ctype]
	base := Slugify(name, 60)
	if base == "" {
		base = "logo"
	}
	return base + ext
}

// systemProvider names the provider of the System stream's default route,
// or "" when System mail has nowhere to go.
func (s *Service) systemProvider(ctx context.Context, ws *store.Workspace) string {
	set, err := routing.Load(ctx, s.Q, ws.ID)
	if err != nil {
		return ""
	}
	rule, ok := set.Default(providers.StreamSystem)
	if !ok || len(rule.Splits) == 0 {
		return ""
	}
	best := rule.Splits[0]
	for _, sp := range rule.Splits[1:] {
		if sp.Weight > best.Weight {
			best = sp
		}
	}
	p, err := s.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: ws.ID, ID: best.ProviderID})
	if err != nil {
		return ""
	}
	return p.Name
}
