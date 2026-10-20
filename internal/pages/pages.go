package pages

import (
	"context"
	"embed"
	"encoding/json"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "pages")

// Templates a page starts from.
const (
	TemplateAbout      = "about"
	TemplateLeadMagnet = "lead_magnet"
	TemplateLaunch     = "launch"
	TemplateWebinar    = "webinar"
	TemplateSales      = "sales"
	TemplateBlank      = "blank"
)

// TemplateKeys are the templates in the picker's order.
var TemplateKeys = []string{TemplateAbout, TemplateLeadMagnet, TemplateLaunch, TemplateWebinar, TemplateSales, TemplateBlank}

// Statuses of a page.
const (
	StatusDraft       = "draft"
	StatusScheduled   = "scheduled"
	StatusPublished   = "published"
	StatusUnpublished = "unpublished"
)

// Who may read a page.
const (
	VisibilityPublic  = "public"
	VisibilityMembers = "members"
	VisibilityPaid    = "paid"
)

// Limits of what a page holds.
const (
	MaxHTML    = 2 << 20
	MaxProject = 6 << 20
)

// Selling is the payment connection Buy buttons sell through.
type Selling struct {
	Account string
	Mode    string
	Live    bool
}

// Service holds the pages domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Log   *slog.Logger
	Now   func() time.Time
	// Forms gives the site's addresses (the custom domain once verified)
	// and the branding pages use.
	Forms *forms.Service
	// Website keeps "Show in navigation" header links and says whether the
	// website is on (nil: pages stand alone).
	Website *website.Service
	// Selling is the payment connection Buy buttons need (nil: none).
	Selling func(ctx context.Context, ws *store.Workspace) (Selling, bool, error)
	// Approvals applies the workspace's approval rules to publishing (nil:
	// website:publish decides).
	Approvals Approvals
	// Probe checks a link of a page (the "A link doesn't work" warning):
	// its HTTP status (nil: links aren't checked).
	Probe func(ctx context.Context, url string) (int, error)
	// ReminderFlow makes a webinar's reminder automation (nil: none).
	ReminderFlow func(ctx context.Context, ws *store.Workspace, by *uuid.UUID, r ReminderRequest) (uuid.UUID, error)
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

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func actor(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// lang is the request's language for server words.
func lang(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// word is one of the package's server words.
func word(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(lang(ctx), args)
}

// Slugify turns a name into an address ("Warm-up planner" → "warm-up-planner").
func Slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// Letters outside ASCII don't fit an address; they're dropped.
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		out = strings.Trim(out[:80], "-")
	}
	if out == "" {
		out = "page"
	}
	return out
}

// slugRe is an address's shape: lowercase letters, digits and single hyphens.
var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// reserved are addresses the site and the public pages use themselves.
var reserved = map[string]bool{
	"p": true, "api": true, "auth": true, "hooks": true, "media": true, "w": true, "d": true, "t": true, "u": true,
	"subscribe": true, "confirm": true, "preferences": true, "unsubscribe": true, "view": true, "archive": true, "posts": true, "tag": true,
	"feed": true, "signin": true, "signout": true, "account": true, "sitemap": true, "robots": true, "poll": true, "s": true, "forms": true,
	"upgrade": true, "welcome": true, "billing": true, "gift": true, "gifts": true, "join": true, "leaderboard": true, "recommends": true,
	"products": true, "buy": true, "checkout": true, "thanks": true, "download": true, "downloads": true, "theme": true, "logo": true,
	"sign-in": true, "settings": true, "admin": true, "static": true, "assets": true, "favicon": true,
}

// Reserved reports whether an address belongs to the site itself.
func Reserved(slug string) bool { return reserved[slug] }

// settingsDoc is a version's settings (page_versions.settings).
type settingsDoc struct {
	SEOTitle       string   `json:"seoTitle"`
	SEODescription string   `json:"seoDescription"`
	SocialImage    string   `json:"socialImage"`
	Visibility     string   `json:"visibility"`
	Tiers          []string `json:"tiers"`
	HideHeader     bool     `json:"hideHeader"`
	ShowInNav      bool     `json:"showInNav"`
	NavLabel       string   `json:"navLabel"`
}

func settingsOf(p store.Page) settingsDoc {
	d := settingsDoc{SEOTitle: p.SeoTitle, SEODescription: p.SeoDescription, SocialImage: p.SocialImage, Visibility: p.Visibility,
		HideHeader: p.HideHeader, ShowInNav: p.ShowInNav, NavLabel: p.NavLabel, Tiers: []string{}}
	for _, t := range p.Tiers {
		d.Tiers = append(d.Tiers, t.String())
	}
	return d
}

func (d settingsDoc) json() []byte {
	b, _ := json.Marshal(d)
	return b
}

func parseSettings(raw []byte) settingsDoc {
	var d settingsDoc
	_ = json.Unmarshal(raw, &d)
	if d.Visibility == "" {
		d.Visibility = VisibilityPublic
	}
	if d.Tiers == nil {
		d.Tiers = []string{}
	}
	return d
}

func jsonInto(raw []byte, v any) error { return json.Unmarshal(raw, v) }

// Money is an amount as the pages write it: "$29", "$15.50", "€9".
func Money(cents int64, currency string) string {
	cur := strings.ToLower(currency)
	sym := map[string]string{"usd": "$", "eur": "€", "gbp": "£", "aud": "A$", "cad": "C$", "jpy": "¥", "inr": "₹", "brl": "R$"}[cur]
	if sym == "" {
		sym = strings.ToUpper(currency) + " "
		if currency == "" {
			sym = "$"
		}
	}
	neg := cents < 0
	if neg {
		cents = -cents
	}
	whole, frac := cents/100, cents%100
	if cur == "jpy" {
		whole, frac = cents, 0
	}
	digits := strconv.FormatInt(whole, 10)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := sym + b.String()
	if frac != 0 {
		out += "." + strconv.FormatInt(frac/10, 10) + strconv.FormatInt(frac%10, 10)
	}
	if neg {
		out = "-" + out
	}
	return out
}

// initialsOf are "SR" for Sofia Rossi.
func initialsOf(name string) string {
	f := strings.Fields(name)
	out := ""
	for i, w := range f {
		if i == 0 || i == len(f)-1 {
			for _, r := range w {
				out += strings.ToUpper(string(r))
				break
			}
		}
	}
	return out
}
