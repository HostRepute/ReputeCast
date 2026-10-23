package website

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// Website statuses.
const (
	StatusOff       = "off"
	StatusTurningOn = "turning_on"
	StatusOn        = "on"
)

// Home layouts.
const (
	LayoutFeatured = "featured"
	LayoutList     = "list"
	LayoutGrid     = "grid"
)

// Home section kinds.
const (
	SectionSubscribe = "subscribe"
	SectionFeatured  = "featured"
	SectionLatest    = "latest"
	SectionTag       = "tag"
	SectionAbout     = "about"
	SectionTiers     = "tiers"
)

// Navigation link kinds.
const (
	NavHome    = "home"
	NavTag     = "tag"
	NavPage    = "page"
	NavArchive = "archive"
	NavURL     = "url"
)

// MaxHeaderLinks is how many links the header holds.
const MaxHeaderLinks = 6

// MaxFeatured is how many posts the home page features.
const MaxFeatured = 3

// MaxTags is how many tags a post has.
const MaxTags = 5

// WebsiteHomeSection is one section of the home page, in its order.
type WebsiteHomeSection struct {
	Kind string `json:"kind" enum:"subscribe,featured,latest,tag,about,tiers"`
	On   bool   `json:"on"`
	// Count is how many posts the latest posts and tag rows show.
	Count int `json:"count,omitempty" minimum:"0" maximum:"24"`
	// Tag is a tag row's tag.
	Tag string `json:"tag,omitempty"`
	// Note is the subscribe band's line under its form ("Free, every
	// Tuesday. Unsubscribe any time with one click.").
	Note string `json:"note,omitempty" maxLength:"200"`
	// Text is the subscribe band's line under every post ("The Weekly
	// Digest, every Tuesday.").
	Text string `json:"text,omitempty" maxLength:"300"`
}

// WebsiteNavLink is a link of the header.
type WebsiteNavLink struct {
	Kind string `json:"kind" enum:"home,tag,page,archive,url"`
	// Ref is a tag's id or a page's slug.
	Ref string `json:"ref,omitempty" maxLength:"120"`
	// Label overrides the tag's or page's name (an outside address needs one).
	Label string `json:"label,omitempty" maxLength:"40"`
	URL   string `json:"url,omitempty" maxLength:"500"`
}

// WebsiteFooter are the footer's switches. Privacy and "Sent with
// ReputeCast" follow the branding.
type WebsiteFooter struct {
	Archive     bool `json:"archive"`
	RSS         bool `json:"rss"`
	Preferences bool `json:"preferences"`
}

// WebsiteSignupList is a list offered on sign-up, whether it's checked,
// and the consent text shown with it (kept with each sign-up).
type WebsiteSignupList struct {
	ListID  string `json:"listId" format:"uuid"`
	Checked bool   `json:"checked"`
	Consent string `json:"consent,omitempty" maxLength:"300"`
}

// WebsitePaywall is the paywall card's text and the tiers it offers.
type WebsitePaywall struct {
	Heading string `json:"heading" maxLength:"120"`
	Text    string `json:"text" maxLength:"400"`
	// Tiers are the tiers to show, in order (empty: every public tier).
	Tiers []string `json:"tiers" maxItems:"10"`
	// Upgrade and Signup are the button labels; Free is the free row.
	Upgrade string `json:"upgrade" maxLength:"40"`
	Signup  string `json:"signup" maxLength:"40"`
	Free    string `json:"free" maxLength:"200"`
}

// config is a workspace's website settings, decoded, with defaults.
type config struct {
	row         store.WebsiteSetting
	Sections    []WebsiteHomeSection
	Header      []WebsiteNavLink
	Footer      WebsiteFooter
	SignupLists []WebsiteSignupList
	Paywall     WebsitePaywall
	// paywallDefault: nobody wrote the paywall card yet; its words are
	// ReputeCast's, in the reader's language.
	paywallDefault bool
}

// paywallIn is the paywall card as a reader of a language reads it: what
// the workspace wrote, else ReputeCast's words in that language.
func (c *config) paywallIn(l i18n.Lang) WebsitePaywall {
	if !c.paywallDefault {
		return c.Paywall
	}
	p := DefaultPaywallIn(l)
	p.Tiers = c.Paywall.Tiers
	return p
}

// On reports whether the website is on (turning on counts as off until it finishes).
func (c *config) On() bool { return c.row.Status == StatusOn }

func defaultSections() []WebsiteHomeSection {
	return []WebsiteHomeSection{
		{Kind: SectionSubscribe, On: true},
		{Kind: SectionFeatured, On: true},
		{Kind: SectionLatest, On: true, Count: 6},
		{Kind: SectionAbout, On: true},
		{Kind: SectionTiers, On: true},
	}
}

// DefaultPaywall is the paywall card before anyone edits it, in English.
func DefaultPaywall() WebsitePaywall { return DefaultPaywallIn(i18n.For(i18n.English)) }

// DefaultPaywallIn is the paywall card before anyone edits it, in a
// language.
func DefaultPaywallIn(l i18n.Lang) WebsitePaywall {
	w := WordsIn(l)
	return WebsitePaywall{Heading: w.Text("paywall.heading", nil), Text: "", Tiers: []string{}, Upgrade: w.Text("paywall.upgrade", nil),
		Signup: w.Text("paywall.signup", nil), Free: w.Text("paywall.free", nil)}
}

func decodeConfig(row store.WebsiteSetting) *config {
	c := &config{row: row, Footer: WebsiteFooter{Archive: true, RSS: true, Preferences: true}}
	if err := json.Unmarshal(row.Sections, &c.Sections); err != nil || len(c.Sections) == 0 {
		c.Sections = defaultSections()
	}
	if err := json.Unmarshal(row.HeaderLinks, &c.Header); err != nil || c.Header == nil {
		c.Header = []WebsiteNavLink{}
	}
	if len(row.Footer) > 2 {
		_ = json.Unmarshal(row.Footer, &c.Footer)
	}
	if err := json.Unmarshal(row.SignupLists, &c.SignupLists); err != nil || c.SignupLists == nil {
		c.SignupLists = []WebsiteSignupList{}
	}
	c.Paywall, c.paywallDefault = DefaultPaywall(), true
	if len(row.Paywall) > 2 {
		_ = json.Unmarshal(row.Paywall, &c.Paywall)
		c.paywallDefault = false
	}
	if c.Paywall.Tiers == nil {
		c.Paywall.Tiers = []string{}
	}
	if c.row.Featured == nil {
		c.row.Featured = []uuid.UUID{}
	}
	return c
}

// load reads (and creates on first use) a workspace's website settings.
func (s *Service) load(ctx context.Context, q *store.Queries, ws *store.Workspace) (*config, error) {
	// A preview with unsaved Home & navigation settings (WithHomeNavDraft).
	if c := draftConfig(ctx, ws.ID); c != nil {
		return c, nil
	}
	row, err := q.WebsiteSettingsGet(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := q.WebsiteSettingsEnsure(ctx, ws.ID); err != nil {
			return nil, err
		}
		row, err = q.WebsiteSettingsGet(ctx, ws.ID)
	}
	if err != nil {
		return nil, err
	}
	return decodeConfig(row), nil
}

// siteTitle is the site's title: its own, else the subscribe page's
// heading, else the publisher's name.
func siteTitle(c *config, set *forms.Settings) string {
	if t := strings.TrimSpace(c.row.Title); t != "" {
		return t
	}
	if t := strings.TrimSpace(set.Pages.Subscribe.Heading); t != "" {
		return t
	}
	return set.Branding.PublisherName
}

// rules are the access settings as the gate reads them, with every paid tier.
func (s *Service) rules(ctx context.Context, q *store.Queries, ws *store.Workspace, c *config) (gate.Rules, []gate.Tier, error) {
	r := gate.Rules{SigninOn: c.row.SigninOn, VisitorsOn: c.row.MeterVisitorsOn, Visitors: int(c.row.MeterVisitors),
		MembersOn: c.row.MeterMembersOn, Members: int(c.row.MeterMembers), Tiers: map[uuid.UUID]gate.Tier{}}
	tiers, err := s.tiers(ctx, q, ws)
	if err != nil {
		return r, nil, err
	}
	for _, t := range tiers {
		r.Tiers[t.ID] = t
	}
	return r, tiers, nil
}

// tiers are the workspace's paid tiers, lowest rank first.
func (s *Service) tiers(ctx context.Context, q *store.Queries, ws *store.Workspace) ([]gate.Tier, error) {
	rows, err := q.WebsiteTiers(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make([]gate.Tier, 0, len(rows))
	for _, r := range rows {
		out = append(out, gate.Tier{ID: r.ID, Name: r.Name, Label: TierLabel(r.Name), Rank: int(r.AccessRank)})
	}
	return out, nil
}

// TierLabel is the quiet label of a tier on posts: its last word unless
// that word is generic ("Weekly Digest Insider" → "Insider"; "Founding
// member" → "Founding member").
func TierLabel(name string) string {
	f := strings.Fields(name)
	if len(f) <= 2 {
		return name
	}
	last := strings.ToLower(f[len(f)-1])
	switch last {
	case "member", "members", "tier", "plan", "membership", "supporter", "subscription":
		return name
	}
	return f[len(f)-1]
}
