// Package forms implements signup forms and the settings of the public
// pages (spec 5.12, 7.12): embedded, popup and hosted forms, the embed
// script, spam protection, the submission pipeline the public pages call,
// double opt-in emails on the System stream, and the pages' branding,
// custom domain and Turnstile keys.
//
// Contacts, lists, consent and suppressions are written through the
// audience package; the pages themselves are rendered by internal/publicpages.
package forms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service holds the forms domain dependencies.
type Service struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Audit    *audit.Logger
	Log      *slog.Logger
	Audience *audience.Service
	Box      *secrets.Box
	// Mail sends double opt-in confirmations on the System stream (App.Mail,
	// read at call time: the system mailer is wired after the services).
	Mail   func() mailer.Sender
	Tokens *linktoken.Signer
	URLs   publicurl.Builder
	// DNS verifies custom domains.
	DNS dnscheck.Resolver
	// Turnstile verifies challenge tokens with Cloudflare.
	Turnstile *Turnstile
	// Addresses checks signups with the workspace's verification service
	// (nil: forms keep ReputeCast's own checks only).
	Addresses AddressChecker
	Now       func() time.Time
	// Hooks are told about signups and confirmations (Grow's referrals),
	// read at call time.
	Hooks func() []SignupHook

	limitsOnce sync.Once
	limits     *limiter
	hostsMu    sync.Mutex
	hosts      map[string]hostEntry
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func (s *Service) mail() mailer.Sender {
	if s.Mail == nil {
		return mailer.Unavailable{}
	}
	if m := s.Mail(); m != nil {
		return m
	}
	return mailer.Unavailable{}
}

// Form kinds and statuses.
const (
	KindEmbedded = "embedded"
	KindPopup    = "popup"
	KindHosted   = "hosted"

	StatusDraft  = "draft"
	StatusLive   = "live"
	StatusPaused = "paused"
)

// Built-in fields every form can show; custom fields come from the audience.
const (
	FieldEmail     = "email"
	FieldFirstName = "first_name"
	FieldLastName  = "last_name"
	FieldCompany   = "company"
)

var builtinFields = map[string]string{FieldEmail: "Email", FieldFirstName: "First name", FieldLastName: "Last name", FieldCompany: "Company"}

// fieldDoc is one stored field (forms.fields).
type fieldDoc struct {
	Key      string `json:"key"`
	Label    string `json:"label,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// contentDoc is the stored copy and look of a form (forms.content).
type contentDoc struct {
	Heading          string `json:"heading"`
	Description      string `json:"description"`
	Button           string `json:"button"`
	Footnote         string `json:"footnote"`
	ListsLabel       string `json:"listsLabel"`
	EmailPlaceholder string `json:"emailPlaceholder"`
	NamePlaceholder  string `json:"namePlaceholder"`
	// AfterSubmitText is what the form says after a signup; empty uses the
	// built-in message ("We sent you a link to confirm…").
	AfterSubmitText string `json:"afterSubmitText,omitempty"`
	// Popups open after PopupDelay seconds or PopupScroll percent of the
	// page, whichever comes first, and stay closed for PopupRepeatDays.
	PopupDelay      int `json:"popupDelay"`
	PopupScroll     int `json:"popupScroll"`
	PopupRepeatDays int `json:"popupRepeatDays"`
}

// spamDoc is the stored spam protection (forms.spam_protection).
type spamDoc struct {
	Honeypot        bool `json:"honeypot"`
	MinSeconds      int  `json:"min_seconds"`
	RateLimitPerIP  int  `json:"rate_limit_per_ip"`
	BlockDisposable bool `json:"block_disposable"`
	Turnstile       bool `json:"turnstile"`
}

// Config is a form as the submission pipeline and the pages read it.
type Config struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	Name           string
	Kind           string
	Slug           string
	PublicID       string
	Status         string
	Fields         []fieldDoc
	Lists          []uuid.UUID
	DefaultLists   []uuid.UUID
	ListChoice     bool
	DoubleOptIn    bool
	ConsentText    string
	ConsentVersion int
	Content        contentDoc
	Spam           spamDoc
	AllowedOrigins []string
	AfterSubmit    string
	RedirectURL    string
	// Builtin is the hosted page's form when the workspace has none.
	Builtin bool
	// CopyLanguage is the language the form is written in; LanguageMode
	// says which language it shows (page, browser or fixed, with
	// FixedLanguage).
	CopyLanguage  string
	LanguageMode  string
	FixedLanguage string
	// Language is the language this Config shows the form in (FormCopy.In);
	// empty is its copy language. Labels are that language's built-in
	// labels, PrivacyURL its privacy policy link (empty: the branding's) and
	// ListNames the lists' names in it.
	Language   string
	Labels     pagelabels.Set
	PrivacyURL string
	ListNames  map[uuid.UUID]string
	publisher  string
}

// HasField reports whether the form shows a field.
func (c Config) HasField(key string) bool {
	return slices.ContainsFunc(c.Fields, func(f fieldDoc) bool { return f.Key == key })
}

func (c Config) Field(key string) (fieldDoc, bool) {
	for _, f := range c.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return fieldDoc{}, false
}

// Source is the consent and contact source of the form's signups.
func (c Config) Source() string {
	if c.Builtin {
		return audience.SourceSubscribePage
	}
	return "form:" + c.Slug
}

// ConsentVersionLabel is the version stored with consent events ("v3").
func (c Config) ConsentVersionLabel() string {
	if c.ConsentText == "" {
		return ""
	}
	return fmt.Sprintf("v%d", c.ConsentVersion)
}

func defaultSpam() spamDoc {
	return spamDoc{Honeypot: true, MinSeconds: 3, RateLimitPerIP: 5, BlockDisposable: true}
}

// ConfigOf reads a stored form.
func ConfigOf(f store.Form) Config {
	c := Config{
		ID: f.ID, WorkspaceID: f.WorkspaceID, Name: f.Name, Kind: f.Kind, Slug: f.Slug, PublicID: f.PublicID, Status: f.Status,
		Lists: nonNilIDs(f.ListIds), DefaultLists: nonNilIDs(f.DefaultListIds), ListChoice: f.ListChoice, DoubleOptIn: f.DoubleOptIn,
		ConsentText: f.ConsentText, ConsentVersion: int(f.ConsentVersion), AllowedOrigins: nonNilStrings(f.AllowedOrigins),
		AfterSubmit: f.AfterSubmit, RedirectURL: f.RedirectUrl, Spam: defaultSpam(),
		CopyLanguage: f.CopyLanguage, LanguageMode: f.LanguageMode, FixedLanguage: f.FixedLanguage,
	}
	c.Labels = pagelabels.For(c.CopyLanguage, nil)
	_ = json.Unmarshal(f.Fields, &c.Fields)
	if !c.HasField(FieldEmail) {
		c.Fields = append([]fieldDoc{{Key: FieldEmail, Required: true}}, c.Fields...)
	}
	_ = json.Unmarshal(f.Content, &c.Content)
	if len(f.SpamProtection) > 0 {
		_ = json.Unmarshal(f.SpamProtection, &c.Spam)
	}
	c.Content = withContentDefaults(c.Content, c.DoubleOptIn)
	if c.Spam.MinSeconds <= 0 {
		c.Spam.MinSeconds = 3
	}
	if c.Spam.RateLimitPerIP <= 0 {
		c.Spam.RateLimitPerIP = 5
	}
	return c
}

func withContentDefaults(c contentDoc, double bool) contentDoc {
	if c.Button == "" {
		c.Button = "Subscribe"
	}
	if c.ListsLabel == "" {
		c.ListsLabel = "What would you like?"
	}
	if c.EmailPlaceholder == "" {
		c.EmailPlaceholder = "you@company.com"
	}
	if c.NamePlaceholder == "" {
		c.NamePlaceholder = "Sam"
	}
	if c.PopupDelay <= 0 && c.PopupScroll <= 0 {
		c.PopupDelay = 8
	}
	if c.PopupRepeatDays <= 0 {
		c.PopupRepeatDays = 14
	}
	return c
}

// BuiltinForm is the hosted page's form for a workspace that hasn't made
// one: email and first name, every public list, double opt-in.
func BuiltinForm(ws *store.Workspace, publisher string, lists []audience.PublicList) Config {
	c := Config{
		WorkspaceID: ws.ID, Name: "Subscribe page", Kind: KindHosted, Slug: "subscribe", Status: StatusLive, Builtin: true,
		Fields:     []fieldDoc{{Key: FieldEmail, Required: true}, {Key: FieldFirstName}},
		ListChoice: true, DoubleOptIn: true, Spam: defaultSpam(), AfterSubmit: "message", ConsentVersion: 1,
		ConsentText: DefaultConsentText(publisher), AllowedOrigins: []string{},
		CopyLanguage: "en", LanguageMode: LanguageBrowser, Labels: pagelabels.English(), publisher: publisher,
	}
	for _, l := range lists {
		if l.Public {
			c.Lists = append(c.Lists, l.ID)
		}
	}
	if len(c.Lists) > 0 {
		c.DefaultLists = []uuid.UUID{c.Lists[0]}
	}
	c.Content = withContentDefaults(contentDoc{Heading: "Subscribe to " + publisher}, true)
	return c
}

// DefaultConsentText is the consent sentence of a new form.
func DefaultConsentText(publisher string) string {
	return "I agree to receive emails from " + publisher + ". Unsubscribe any time with one click. See the privacy policy."
}

func nonNilIDs(xs []uuid.UUID) []uuid.UUID {
	if xs == nil {
		return []uuid.UUID{}
	}
	return xs
}

func nonNilStrings(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

type SignupFormField struct {
	Key      string `json:"key" pattern:"^[a-z][a-z0-9_]{0,62}$" doc:"email, first_name, last_name, company or a custom field key"`
	Label    string `json:"label" maxLength:"80" doc:"Shown above the input; empty uses the field's name"`
	Required bool   `json:"required"`
}

// SignupFormContent is the copy of a form.
type SignupFormContent struct {
	Heading          string `json:"heading" maxLength:"120"`
	Description      string `json:"description" maxLength:"500"`
	Button           string `json:"button" maxLength:"40"`
	Footnote         string `json:"footnote" maxLength:"300"`
	ListsLabel       string `json:"listsLabel" maxLength:"80"`
	EmailPlaceholder string `json:"emailPlaceholder" maxLength:"80"`
	NamePlaceholder  string `json:"namePlaceholder" maxLength:"80"`
	PopupDelay       int    `json:"popupDelay" minimum:"0" maximum:"600" doc:"Seconds before a popup opens"`
	PopupScroll      int    `json:"popupScroll" minimum:"0" maximum:"100" doc:"Percent of the page scrolled before a popup opens; 0 is off"`
	PopupRepeatDays  int    `json:"popupRepeatDays" minimum:"0" maximum:"365" doc:"Days a closed popup stays closed"`
	// AfterSubmitText is what the form says after a signup; empty uses the built-in message.
	AfterSubmitText string `json:"afterSubmitText,omitempty" maxLength:"300"`
}

// SignupFormSpam is a form's spam protection.
type SignupFormSpam struct {
	// Honeypot adds an invisible field and the timing check.
	Honeypot   bool `json:"honeypot"`
	MinSeconds int  `json:"minSeconds" minimum:"0" maximum:"60" doc:"Submissions faster than this look automated"`
	// Turnstile asks Cloudflare Turnstile when a submission looks automated.
	Turnstile bool `json:"turnstile"`
	// BlockDisposable rejects disposable domains and role addresses.
	BlockDisposable bool `json:"blockDisposable"`
	RateLimitPerIP  int  `json:"rateLimitPerIp" minimum:"1" maximum:"120" doc:"Submissions per minute from one address"`
}

// SignupFormStats are the builder's "Last 30 days".
type SignupFormStats struct {
	Days        int   `json:"days"`
	Views       int64 `json:"views"`
	Submissions int64 `json:"submissions"`
	// SubmissionRate is submissions per view, in percent.
	SubmissionRate float64 `json:"submissionRate"`
	Confirmed      int64   `json:"confirmed"`
	// ConfirmedRate is confirmed submissions per submission, in percent.
	ConfirmedRate float64 `json:"confirmedRate"`
	Blocked       int64   `json:"blocked" doc:"Submissions blocked as spam"`
	Pending       int64   `json:"pending" doc:"Double opt-in confirmations not clicked yet"`
}

// SignupFormBlockedOrigin is a site whose submissions were rejected today.
type SignupFormBlockedOrigin struct {
	Origin string    `json:"origin" example:"staging.acme.io"`
	Count  int64     `json:"count"`
	LastAt time.Time `json:"lastAt"`
}

// SignupFormURLs are where a form lives.
type SignupFormURLs struct {
	// Hosted is the form's hosted page (the workspace's /subscribe for the hosted page form).
	Hosted string `json:"hosted"`
	// Script is the embed script; Submit receives submissions.
	Script string `json:"script"`
	Submit string `json:"submit"`
	// ScriptKB is what the embed script weighs on the wire (gzip), rounded up.
	ScriptKB int `json:"scriptKb"`
}

// SignupFormSummary is a form in the selector.
type SignupFormSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind" enum:"embedded,popup,hosted"`
	Status   string `json:"status" enum:"draft,live,paused"`
	PublicID string `json:"publicId" example:"frm_8c2e1a"`
	Slug     string `json:"slug"`
	// HostedPage is true for the form behind the workspace's /subscribe page.
	HostedPage bool   `json:"hostedPage"`
	HostedURL  string `json:"hostedUrl"`
}

// SignupForm is one form with everything the builder shows.
type SignupForm struct {
	SignupFormSummary
	Fields         []SignupFormField `json:"fields"`
	ListIDs        []string          `json:"listIds" doc:"Lists offered on the form"`
	DefaultListIDs []string          `json:"defaultListIds" doc:"Offered lists that start checked"`
	ListChoice     bool              `json:"listChoice" doc:"Visitors pick lists; off joins every offered list"`
	DoubleOptIn    bool              `json:"doubleOptIn"`
	ConsentText    string            `json:"consentText"`
	ConsentVersion int               `json:"consentVersion"`
	Spam           SignupFormSpam    `json:"spam"`
	AfterSubmit    string            `json:"afterSubmit" enum:"message,redirect"`
	RedirectURL    string            `json:"redirectUrl"`
	AllowedOrigins []string          `json:"allowedOrigins"`
	Content        SignupFormContent `json:"content"`
	URLs           SignupFormURLs    `json:"urls"`
	// PlainHTML is the form as plain HTML, for sites that can't load scripts.
	PlainHTML      string                    `json:"plainHtml"`
	Stats          SignupFormStats           `json:"stats"`
	BlockedOrigins []SignupFormBlockedOrigin `json:"blockedOrigins"`
	// Waiting counts the signups of the last 30 days whose confirmation
	// email hasn't gone out (nothing could send System mail).
	Waiting     int64      `json:"waiting"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// SignupFormInput saves a form ("Save & publish").
type SignupFormInput struct {
	Name           string            `json:"name" minLength:"1" maxLength:"80"`
	Kind           string            `json:"kind" enum:"embedded,popup,hosted"`
	Fields         []SignupFormField `json:"fields" maxItems:"20"`
	ListIDs        []string          `json:"listIds" maxItems:"50"`
	DefaultListIDs []string          `json:"defaultListIds" maxItems:"50"`
	ListChoice     bool              `json:"listChoice"`
	DoubleOptIn    bool              `json:"doubleOptIn"`
	ConsentText    string            `json:"consentText" maxLength:"600"`
	Spam           SignupFormSpam    `json:"spam"`
	AfterSubmit    string            `json:"afterSubmit" enum:"message,redirect"`
	RedirectURL    string            `json:"redirectUrl" maxLength:"2000"`
	AllowedOrigins []string          `json:"allowedOrigins" maxItems:"50"`
	Content        SignupFormContent `json:"content"`
	// Publish makes the form live (the builder's Save & publish); false keeps a draft a draft.
	Publish bool `json:"publish"`
}

type SignupFormCreateInput struct {
	Name string `json:"name" minLength:"1" maxLength:"80"`
	Kind string `json:"kind" enum:"embedded,popup,hosted"`
}

// SignupFormList is a list a form can offer.
type SignupFormList struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
	Subscribed  int64  `json:"subscribed"`
}

// SignupFormCustomField is a custom contact field a form can ask for.
type SignupFormCustomField struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Type    string   `json:"type" enum:"text,number,date,boolean,select"`
	Options []string `json:"options"`
}

// SignupFormsOverview feeds the Forms & public pages screens: the forms,
// the lists and fields they can use, and what the preview needs.
type SignupFormsOverview struct {
	Forms        []SignupFormSummary     `json:"forms"`
	Lists        []SignupFormList        `json:"lists"`
	CustomFields []SignupFormCustomField `json:"customFields"`
	// HostedURL is the workspace's subscribe page; HostedFormID the form behind it (empty: the built-in form).
	HostedURL    string `json:"hostedUrl"`
	HostedFormID string `json:"hostedFormId"`
	// SystemProvider names the System stream's provider ("Postmark"); empty
	// when none is routed. SystemMail says whether System mail can go out
	// now: without it confirmations wait and their links are sent by hand.
	SystemProvider string             `json:"systemProvider"`
	SystemMail     bool               `json:"systemMail"`
	Turnstile      PublicTurnstile    `json:"turnstile"`
	Branding       PublicBranding     `json:"branding"`
	Domain         PublicCustomDomain `json:"domain"`
	// Origins that may always submit: the instance and the custom domain.
	OwnOrigins []string `json:"ownOrigins"`
}

var errFormNotFound = api.NotFound("Form")

func (s *Service) form(ctx context.Context, q *store.Queries, ws *store.Workspace, id string) (store.Form, error) {
	fid, err := uuid.Parse(id)
	if err != nil {
		return store.Form{}, errFormNotFound
	}
	f, err := q.FormGet(ctx, store.FormGetParams{WorkspaceID: ws.ID, ID: fid})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Form{}, errFormNotFound
	}
	return f, err
}

// Overview lists the forms and what the builder needs around them.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace) (SignupFormsOverview, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return SignupFormsOverview{}, err
	}
	forms, err := s.Q.FormsList(ctx, ws.ID)
	if err != nil {
		return SignupFormsOverview{}, err
	}
	links := s.Links(ws, set)
	out := SignupFormsOverview{Forms: []SignupFormSummary{}, Lists: []SignupFormList{}, CustomFields: []SignupFormCustomField{}, HostedURL: links.Subscribe(""), OwnOrigins: s.OwnOrigins(set)}
	if set.row.HostedFormID != nil {
		out.HostedFormID = set.row.HostedFormID.String()
	}
	for _, f := range forms {
		out.Forms = append(out.Forms, s.summary(f, set, links))
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, SignupFormList{ID: l.ID.String(), Name: l.Name, Description: l.Description, Public: l.Public, Subscribed: l.Subscribed})
	}
	fields, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, f := range fields {
		var opts []string
		_ = json.Unmarshal(f.Options, &opts)
		if opts == nil {
			opts = []string{}
		}
		out.CustomFields = append(out.CustomFields, SignupFormCustomField{Key: f.Key, Label: f.Label, Type: f.Type, Options: opts})
	}
	out.SystemProvider = s.systemProvider(ctx, ws)
	out.SystemMail = s.SystemMailAvailable(ctx, ws)
	out.Turnstile = set.turnstile()
	out.Branding = s.brandingOut(ws, set)
	out.Domain = set.domain(s.URLs)
	return out, nil
}

func (s *Service) summary(f store.Form, set *pageSettings, links PageLinks) SignupFormSummary {
	hosted := set.row.HostedFormID != nil && *set.row.HostedFormID == f.ID
	u := links.Subscribe(f.Slug)
	if hosted {
		u = links.Subscribe("")
	}
	return SignupFormSummary{ID: f.ID.String(), Name: f.Name, Kind: f.Kind, Status: f.Status, PublicID: f.PublicID, Slug: f.Slug, HostedPage: hosted, HostedURL: u}
}

// Get returns one form with its numbers.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, id string) (SignupForm, error) {
	f, err := s.form(ctx, s.Q, ws, id)
	if err != nil {
		return SignupForm{}, err
	}
	return s.detail(ctx, ws, f)
}

func (s *Service) detail(ctx context.Context, ws *store.Workspace, f store.Form) (SignupForm, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return SignupForm{}, err
	}
	links := s.Links(ws, set)
	c := ConfigOf(f)
	out := SignupForm{
		SignupFormSummary: s.summary(f, set, links),
		Fields:            []SignupFormField{}, ListIDs: idStrings(c.Lists), DefaultListIDs: idStrings(c.DefaultLists), ListChoice: c.ListChoice,
		DoubleOptIn: c.DoubleOptIn, ConsentText: c.ConsentText, ConsentVersion: c.ConsentVersion,
		Spam:        SignupFormSpam{Honeypot: c.Spam.Honeypot, MinSeconds: c.Spam.MinSeconds, Turnstile: c.Spam.Turnstile, BlockDisposable: c.Spam.BlockDisposable, RateLimitPerIP: c.Spam.RateLimitPerIP},
		AfterSubmit: c.AfterSubmit, RedirectURL: c.RedirectURL, AllowedOrigins: c.AllowedOrigins,
		Content: SignupFormContent{Heading: c.Content.Heading, Description: c.Content.Description, Button: c.Content.Button, Footnote: c.Content.Footnote,
			ListsLabel: c.Content.ListsLabel, EmailPlaceholder: c.Content.EmailPlaceholder, NamePlaceholder: c.Content.NamePlaceholder,
			PopupDelay: c.Content.PopupDelay, PopupScroll: c.Content.PopupScroll, PopupRepeatDays: c.Content.PopupRepeatDays, AfterSubmitText: c.Content.AfterSubmitText},
		BlockedOrigins: []SignupFormBlockedOrigin{}, PublishedAt: f.PublishedAt, UpdatedAt: f.UpdatedAt,
	}
	for _, fd := range c.Fields {
		out.Fields = append(out.Fields, SignupFormField{Key: fd.Key, Label: fd.Label, Required: fd.Required || fd.Key == FieldEmail})
	}
	out.URLs = SignupFormURLs{Hosted: out.HostedURL, Script: links.Script(f.PublicID), Submit: links.Submit(f.PublicID)}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	customs, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	fc, err := s.loadFormCopy(ctx, s.Q, ws, set, c)
	if err != nil {
		return out, err
	}
	out.PlainHTML = PlainHTML(fc.plain(lists), links, lists, customs, set.Branding)
	out.URLs.ScriptKB = gzipKB(fc.script(links, set.Branding, lists, customs))
	loc := location(ws)
	today := s.now().In(loc)
	since := dayOf(today.AddDate(0, 0, -29), loc)
	rows, err := s.Q.FormStatsSince(ctx, store.FormStatsSinceParams{FormIds: []uuid.UUID{f.ID}, Since: since})
	if err != nil {
		return out, err
	}
	out.Stats = SignupFormStats{Days: 30}
	for _, r := range rows {
		out.Stats.Views, out.Stats.Submissions, out.Stats.Confirmed, out.Stats.Blocked = r.Views, r.Submissions, r.Confirmed, r.Blocked
	}
	out.Stats.SubmissionRate = ratio(out.Stats.Submissions, out.Stats.Views)
	out.Stats.ConfirmedRate = ratio(out.Stats.Confirmed, out.Stats.Submissions)
	if f.DoubleOptIn {
		out.Stats.Pending = max(out.Stats.Submissions-out.Stats.Confirmed, 0)
	}
	blocked, err := s.Q.FormBlockedOrigins(ctx, store.FormBlockedOriginsParams{FormID: f.ID, Day: dayOf(today, loc)})
	if err != nil {
		return out, err
	}
	for _, b := range blocked {
		out.BlockedOrigins = append(out.BlockedOrigins, SignupFormBlockedOrigin{Origin: b.Origin, Count: b.Count, LastAt: b.LastAt})
	}
	if f.DoubleOptIn {
		if out.Waiting, err = s.Q.FormWaitingCount(ctx, store.FormWaitingCountParams{FormID: &f.ID, Since: s.now().Add(-waitingWindow)}); err != nil {
			return out, err
		}
	}
	return out, nil
}

func ratio(n, d int64) float64 {
	if d <= 0 {
		return 0
	}
	return float64(int64(float64(n)/float64(d)*1000+0.5)) / 10
}

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

// location is the workspace's timezone (days of the form numbers).
func location(ws *store.Workspace) *time.Location {
	if loc, err := time.LoadLocation(ws.Timezone); err == nil {
		return loc
	}
	return time.UTC
}

// dayOf is the calendar day of t in loc, as stored in date columns.
func dayOf(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

var (
	slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)
	keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
)

// Slugify turns a name into a URL segment ("Newsletter signup" → "newsletter-signup").
func Slugify(name string, max int) string {
	s := strings.Trim(slugUnsafe.ReplaceAllString(strings.ToLower(asciiFold(name)), "-"), "-")
	if len(s) > max {
		s = strings.TrimRight(s[:max], "-")
	}
	return s
}

// asciiFold drops accents from common Latin letters so slugs stay readable.
func asciiFold(s string) string {
	r := strings.NewReplacer("à", "a", "á", "a", "â", "a", "ä", "a", "ã", "a", "å", "a", "ç", "c", "è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i", "ñ", "n", "ò", "o", "ó", "o", "ô", "o", "ö", "o", "õ", "o", "ø", "o", "ù", "u", "ú", "u",
		"û", "u", "ü", "u", "ý", "y", "ÿ", "y", "ß", "ss", "æ", "ae", "œ", "oe", "’", "", "'", "")
	return r.Replace(strings.ToLower(s))
}

func (s *Service) uniqueSlug(ctx context.Context, q *store.Queries, ws *store.Workspace, name string, except uuid.UUID) (string, error) {
	base := Slugify(name, 50)
	if base == "" {
		base = "form"
	}
	slug := base
	for i := 2; ; i++ {
		taken, err := q.FormSlugTaken(ctx, store.FormSlugTakenParams{WorkspaceID: ws.ID, Slug: slug, ID: except})
		if err != nil {
			return "", err
		}
		if !taken && slug != "subscribe" && slug != "challenge" {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func (s *Service) newPublicID(ctx context.Context, q *store.Queries, ws *store.Workspace) (string, error) {
	for range 8 {
		b := make([]byte, 3)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		id := "frm_" + hex.EncodeToString(b)
		taken, err := q.FormPublicIDTaken(ctx, store.FormPublicIDTakenParams{WorkspaceID: ws.ID, PublicID: id})
		if err != nil {
			return "", err
		}
		if !taken {
			return id, nil
		}
	}
	return "", errors.New("forms: no free public id")
}

func actor(ctx context.Context) *uuid.UUID {
	if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
		id := p.UserID
		return &id
	}
	return nil
}

// Create adds a draft form with sensible defaults: email and first name,
// the first public list, double opt-in and every spam check.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, in SignupFormCreateInput) (SignupForm, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return SignupForm{}, api.Invalid("body.name", "Name the form.")
	}
	kind := in.Kind
	if kind == "" {
		kind = KindEmbedded
	}
	set, err := s.settings(ctx, ws)
	if err != nil {
		return SignupForm{}, err
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return SignupForm{}, err
	}
	var offered []uuid.UUID
	for _, l := range lists {
		if l.Public {
			offered = append(offered, l.ID)
			break
		}
	}
	if offered == nil && len(lists) > 0 {
		offered = []uuid.UUID{lists[0].ID}
	}
	publisher := set.Branding.PublisherName
	fields, _ := json.Marshal([]fieldDoc{{Key: FieldEmail, Required: true}, {Key: FieldFirstName}})
	content, _ := json.Marshal(withContentDefaults(contentDoc{Heading: "The " + publisher + " newsletter", Footnote: "We'll send a confirmation link first. No spam, ever."}, true))
	spam, _ := json.Marshal(defaultSpam())
	var f store.Form
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		slug, err := s.uniqueSlug(ctx, q, ws, name, uuid.Nil)
		if err != nil {
			return err
		}
		pid, err := s.newPublicID(ctx, q, ws)
		if err != nil {
			return err
		}
		f, err = q.FormInsert(ctx, store.FormInsertParams{
			ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Name: name, Kind: kind, Slug: slug, PublicID: pid,
			ListIds: nonNilIDs(offered), DefaultListIds: nonNilIDs(offered), ListChoice: false, DoubleOptIn: true,
			Fields: fields, Content: content, SpamProtection: spam, ConsentText: DefaultConsentText(publisher), ConsentVersion: 1,
			AllowedOrigins: []string{}, AfterSubmit: "message", RedirectUrl: "", Status: StatusDraft, CreatedBy: actor(ctx), At: s.now(),
		})
		if err != nil {
			return err
		}
		// A new form's text is English, its copy language. A hosted form
		// follows the visitor's browser, the others the page they're on.
		if kind == KindHosted {
			if f, err = q.FormSetLanguage(ctx, store.FormSetLanguageParams{WorkspaceID: ws.ID, ID: f.ID, LanguageMode: LanguageBrowser, By: actor(ctx), At: s.now()}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.create", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
			After: map[string]any{"name": f.Name, "kind": f.Kind, "slug": f.Slug, "publicId": f.PublicID, "languageMode": f.LanguageMode}})
	})
	if err != nil {
		return SignupForm{}, err
	}
	return s.detail(ctx, ws, f)
}

// formSnapshot is what the audit log keeps of a form.
func formSnapshot(f store.Form) map[string]any {
	return map[string]any{
		"name": f.Name, "kind": f.Kind, "status": f.Status, "lists": f.ListIds, "defaultLists": f.DefaultListIds, "listChoice": f.ListChoice,
		"doubleOptIn": f.DoubleOptIn, "fields": f.Fields, "content": f.Content, "spamProtection": f.SpamProtection,
		"consentText": f.ConsentText, "consentVersion": f.ConsentVersion, "allowedOrigins": f.AllowedOrigins,
		"afterSubmit": f.AfterSubmit, "redirectUrl": f.RedirectUrl,
	}
}

// Update saves the builder ("Save & publish").
func (s *Service) Update(ctx context.Context, ws *store.Workspace, id string, in SignupFormInput) (SignupForm, error) {
	var after store.Form
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		p, err := s.validate(ctx, q, ws, before, in)
		if err != nil {
			return err
		}
		after, err = q.FormUpdate(ctx, p)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.update", TargetType: "form", TargetID: after.ID.String(), TargetName: after.Name,
			Before: formSnapshot(before), After: formSnapshot(after)})
	})
	if err != nil {
		return SignupForm{}, err
	}
	s.forgetHosts()
	return s.detail(ctx, ws, after)
}

func (s *Service) validate(ctx context.Context, q *store.Queries, ws *store.Workspace, before store.Form, in SignupFormInput) (store.FormUpdateParams, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return store.FormUpdateParams{}, api.Invalid("body.name", "Name the form.")
	}
	// Fields: email first and always required; built-ins or custom fields of the workspace.
	customs, err := q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return store.FormUpdateParams{}, err
	}
	customKeys := map[string]bool{}
	for _, c := range customs {
		customKeys[c.Key] = true
	}
	fields := []fieldDoc{{Key: FieldEmail, Required: true, Label: labelOf(in.Fields, FieldEmail)}}
	seen := map[string]bool{FieldEmail: true}
	for i, f := range in.Fields {
		if f.Key == FieldEmail || seen[f.Key] {
			continue
		}
		if _, ok := builtinFields[f.Key]; !ok && !customKeys[f.Key] {
			return store.FormUpdateParams{}, api.Invalid(fmt.Sprintf("body.fields[%d].key", i), fmt.Sprintf("%s isn't a contact field. Add it under Audience › Fields first.", f.Key))
		}
		if !keyPattern.MatchString(f.Key) {
			return store.FormUpdateParams{}, api.Invalid(fmt.Sprintf("body.fields[%d].key", i), "That isn't a field key.")
		}
		seen[f.Key] = true
		fields = append(fields, fieldDoc{Key: f.Key, Label: strings.TrimSpace(f.Label), Required: f.Required})
	}
	// Lists: the workspace's; defaults must be offered.
	lists, err := parseLists(ctx, q, ws, in.ListIDs, "body.listIds")
	if err != nil {
		return store.FormUpdateParams{}, err
	}
	if len(lists) == 0 {
		return store.FormUpdateParams{}, api.Invalid("body.listIds", "Offer at least one list: signups need somewhere to go.")
	}
	defaults, err := parseLists(ctx, q, ws, in.DefaultListIDs, "body.defaultListIds")
	if err != nil {
		return store.FormUpdateParams{}, err
	}
	defaults = slices.DeleteFunc(defaults, func(d uuid.UUID) bool { return !slices.Contains(lists, d) })
	// Consent text: a new version whenever the words change.
	consent := strings.TrimSpace(in.ConsentText)
	version := before.ConsentVersion
	if consent != before.ConsentText && before.ConsentText != "" {
		version++
	}
	if version < 1 {
		version = 1
	}
	redirect := strings.TrimSpace(in.RedirectURL)
	if in.AfterSubmit == "redirect" {
		u, err := url.Parse(redirect)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
			return store.FormUpdateParams{}, api.Invalid("body.redirectUrl", "Use a full address like https://acme.io/thanks.")
		}
	} else {
		redirect = ""
	}
	origins, err := normalizeOrigins(in.AllowedOrigins)
	if err != nil {
		return store.FormUpdateParams{}, err
	}
	spam := spamDoc{Honeypot: in.Spam.Honeypot, MinSeconds: in.Spam.MinSeconds, Turnstile: in.Spam.Turnstile, BlockDisposable: in.Spam.BlockDisposable, RateLimitPerIP: in.Spam.RateLimitPerIP}
	if spam.MinSeconds <= 0 {
		spam.MinSeconds = 3
	}
	if spam.RateLimitPerIP <= 0 {
		spam.RateLimitPerIP = 5
	}
	c := in.Content
	content := contentDoc{
		Heading: strings.TrimSpace(c.Heading), Description: strings.TrimSpace(c.Description), Button: strings.TrimSpace(c.Button), Footnote: strings.TrimSpace(c.Footnote),
		ListsLabel: strings.TrimSpace(c.ListsLabel), EmailPlaceholder: strings.TrimSpace(c.EmailPlaceholder), NamePlaceholder: strings.TrimSpace(c.NamePlaceholder),
		PopupDelay: c.PopupDelay, PopupScroll: c.PopupScroll, PopupRepeatDays: c.PopupRepeatDays, AfterSubmitText: strings.TrimSpace(c.AfterSubmitText),
	}
	fieldsJSON, _ := json.Marshal(fields)
	contentJSON, _ := json.Marshal(content)
	spamJSON, _ := json.Marshal(spam)
	status := before.Status
	if in.Publish {
		status = StatusLive
	}
	return store.FormUpdateParams{
		WorkspaceID: ws.ID, ID: before.ID, Name: name, Kind: in.Kind, ListIds: lists, DefaultListIds: nonNilIDs(defaults), ListChoice: in.ListChoice,
		DoubleOptIn: in.DoubleOptIn, Fields: fieldsJSON, Content: contentJSON, SpamProtection: spamJSON, ConsentText: consent, ConsentVersion: version,
		AllowedOrigins: origins, AfterSubmit: in.AfterSubmit, RedirectUrl: redirect, Status: status, UpdatedBy: actor(ctx), At: s.now(),
	}, nil
}

func labelOf(fields []SignupFormField, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return strings.TrimSpace(f.Label)
		}
	}
	return ""
}

func parseLists(ctx context.Context, q *store.Queries, ws *store.Workspace, ids []string, loc string) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, api.Invalid(loc, "Pick lists from this workspace.")
		}
		if slices.Contains(out, id) {
			continue
		}
		if _, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, api.Invalid(loc, "Pick lists from this workspace.")
			}
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

var hostPattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9-]{0,61}[a-z0-9]$|^localhost$`)

// NormalizeOrigin turns "https://Docs.Acme.io/path" or "docs.acme.io" into
// the host an origin is matched on ("docs.acme.io", with ":port" when set).
func NormalizeOrigin(raw string) (string, bool) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return "", false
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" || u.User != nil {
		return "", false
	}
	host := strings.TrimSuffix(u.Hostname(), ".")
	if !hostPattern.MatchString(host) {
		return "", false
	}
	if p := u.Port(); p != "" {
		return host + ":" + p, true
	}
	return host, true
}

func normalizeOrigins(in []string) ([]string, error) {
	out := []string{}
	for i, raw := range in {
		h, ok := NormalizeOrigin(raw)
		if !ok {
			return nil, api.Invalid(fmt.Sprintf("body.allowedOrigins[%d]", i), fmt.Sprintf("%q isn't a domain. Use a site's address, like docs.acme.io.", raw))
		}
		if !slices.Contains(out, h) {
			out = append(out, h)
		}
	}
	return out, nil
}

// SignupFormRenameInput renames a form (its slug follows).
type SignupFormRenameInput struct {
	Name string `json:"name" minLength:"1" maxLength:"80"`
}

// Rename changes a form's name and the slug of its hosted page.
func (s *Service) Rename(ctx context.Context, ws *store.Workspace, id string, in SignupFormRenameInput) (SignupForm, error) {
	var after store.Form
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return api.Invalid("body.name", "Name the form.")
		}
		slug := before.Slug
		if name != before.Name {
			if slug, err = s.uniqueSlug(ctx, q, ws, name, before.ID); err != nil {
				return err
			}
		}
		after, err = q.FormRename(ctx, store.FormRenameParams{WorkspaceID: ws.ID, ID: before.ID, Name: name, Slug: slug, UpdatedBy: actor(ctx), At: s.now()})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.rename", TargetType: "form", TargetID: after.ID.String(), TargetName: after.Name,
			Before: map[string]any{"name": before.Name, "slug": before.Slug}, After: map[string]any{"name": after.Name, "slug": after.Slug}})
	})
	if err != nil {
		return SignupForm{}, err
	}
	return s.detail(ctx, ws, after)
}

// SignupFormStatusInput pauses or resumes a form.
type SignupFormStatusInput struct {
	Status string `json:"status" enum:"live,paused"`
}

// SetStatus pauses a live form (submissions are refused) or makes it live.
func (s *Service) SetStatus(ctx context.Context, ws *store.Workspace, id string, in SignupFormStatusInput) (SignupForm, error) {
	var after store.Form
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		after, err = q.FormSetStatus(ctx, store.FormSetStatusParams{WorkspaceID: ws.ID, ID: before.ID, Status: in.Status, UpdatedBy: actor(ctx), At: s.now()})
		if err != nil {
			return err
		}
		action := "form.resume"
		if in.Status == StatusPaused {
			action = "form.pause"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "form", TargetID: after.ID.String(), TargetName: after.Name,
			Before: map[string]any{"status": before.Status}, After: map[string]any{"status": after.Status}})
	})
	if err != nil {
		return SignupForm{}, err
	}
	return s.detail(ctx, ws, after)
}

// Delete removes a form. Contacts it brought in stay; automations started
// by it show that their trigger form is gone.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, id string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if _, err := q.FormDelete(ctx, store.FormDeleteParams{WorkspaceID: ws.ID, ID: before.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.delete", TargetType: "form", TargetID: before.ID.String(), TargetName: before.Name,
			Before: formSnapshot(before)})
	})
}

// SignupFormOriginInput names a site.
type SignupFormOriginInput struct {
	Origin string `json:"origin" minLength:"1" maxLength:"255" example:"staging.acme.io"`
}

// AllowOrigin adds a site to a form's allowed domains (the "Allow
// staging.acme.io" action of the blocked-origin alert).
func (s *Service) AllowOrigin(ctx context.Context, ws *store.Workspace, id string, in SignupFormOriginInput) (SignupForm, error) {
	host, ok := NormalizeOrigin(in.Origin)
	if !ok {
		return SignupForm{}, api.Invalid("body.origin", "That isn't a domain. Use a site's address, like docs.acme.io.")
	}
	var after store.Form
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		after, err = q.FormAllowOrigin(ctx, store.FormAllowOriginParams{WorkspaceID: ws.ID, ID: before.ID, Origin: host, UpdatedBy: actor(ctx), At: s.now()})
		if errors.Is(err, pgx.ErrNoRows) {
			after = before // already allowed
		} else if err != nil {
			return err
		}
		if _, err := q.FormDismissOrigin(ctx, store.FormDismissOriginParams{FormID: before.ID, Origin: host, At: ptrTime(s.now())}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.allow_origin", TargetType: "form", TargetID: before.ID.String(), TargetName: before.Name,
			Before: map[string]any{"allowedOrigins": before.AllowedOrigins}, After: map[string]any{"allowedOrigins": after.AllowedOrigins}})
	})
	if err != nil {
		return SignupForm{}, err
	}
	return s.detail(ctx, ws, after)
}

// KeepBlocked dismisses a blocked-origin alert ("Keep blocked"): the site
// stays rejected and the alert stays away until it submits again tomorrow.
func (s *Service) KeepBlocked(ctx context.Context, ws *store.Workspace, id string, in SignupFormOriginInput) (SignupForm, error) {
	host, ok := NormalizeOrigin(in.Origin)
	if !ok {
		return SignupForm{}, api.Invalid("body.origin", "That isn't a domain.")
	}
	var f store.Form
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		f, err = s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		n, err := q.FormDismissOrigin(ctx, store.FormDismissOriginParams{FormID: f.ID, Origin: host, At: ptrTime(s.now())})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.keep_blocked", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
			Metadata: map[string]any{"origin": host, "alerts": n}})
	})
	if err != nil {
		return SignupForm{}, err
	}
	return s.detail(ctx, ws, f)
}
