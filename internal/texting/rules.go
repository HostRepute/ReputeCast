package texting

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// Rules are the workspace's SMS rules, as code reads them.
type Rules struct {
	BrandPrefix     string
	OptOutText      string
	OptOutInSteps   bool
	ConfirmationOn  bool
	Confirmation    string
	RequireYes      bool
	Window          channels.Window
	PushWindow      bool
	WhatsAppWindow  bool
	Keywords        sms.Keywords
	HelpReply       string
	OptOutReply     string
	OptInReply      string
	KeywordsBy      string
	Budget          float64
	BudgetAlertPct  int
	BudgetSetBy     *uuid.UUID
	BudgetSetAt     *time.Time
	PauseFailed     float64
	PauseFiltered   float64
	PauseOptOut     float64
	OtherCountries  string
	OtherRouteID    *uuid.UUID
	RetentionDays   int
	ShortDomain     string
	ShortStatus     string
	ShortTarget     string
	ShortCheckedAt  *time.Time
	ShortVerifiedAt *time.Time
	ShortDetail     string
	UpdatedBy       *uuid.UUID
	UpdatedAt       time.Time
}

// defaultTexts writes a new workspace's replies with its name.
func defaultTexts(brand string) (prefix, help, optOut, optIn, confirmation string) {
	if brand == "" {
		brand = "We"
	}
	prefix = brand + ":"
	help = prefix + " Reply STOP to cancel. Msg & data rates may apply."
	optOut = prefix + " You're unsubscribed and won't get more texts. Reply START to resubscribe."
	optIn = prefix + " You're subscribed again. Reply HELP for help, STOP to cancel."
	confirmation = prefix + " You're signed up for texts. Msg & data rates may apply. Reply HELP for help, STOP to cancel."
	return
}

// brandOf is the short name texts start with: the workspace's first word
// ("Acme Newsletter" → "Acme").
func brandOf(ws *store.Workspace) string {
	f := strings.Fields(ws.Name)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// ensureRules creates the workspace's rules with their defaults.
func (s *Service) ensureRules(ctx context.Context, q *store.Queries, ws *store.Workspace) error {
	prefix, help, optOut, optIn, conf := defaultTexts(brandOf(ws))
	return q.SmsSettingsEnsure(ctx, store.SmsSettingsEnsureParams{WorkspaceID: ws.ID, BrandPrefix: prefix, HelpReply: help, OptOutReply: optOut,
		OptInReply: optIn, ConfirmationText: conf, At: s.now()})
}

// rules reads the workspace's rules (defaults when it has none yet).
func (s *Service) rules(ctx context.Context, q *store.Queries, ws *store.Workspace) (Rules, error) {
	row, err := q.SmsSettings(ctx, ws.ID)
	if notFound(err) {
		prefix, help, optOut, optIn, conf := defaultTexts(brandOf(ws))
		return Rules{BrandPrefix: prefix, OptOutText: "Reply STOP to opt out", OptOutInSteps: true, ConfirmationOn: true, Confirmation: conf,
			Window: channels.DefaultWindow, PushWindow: true, WhatsAppWindow: true, Keywords: sms.DefaultKeywords, HelpReply: help, OptOutReply: optOut,
			OptInReply: optIn, KeywordsBy: "reputecast", BudgetAlertPct: 80, PauseFailed: 5, PauseFiltered: 2, PauseOptOut: 3, OtherCountries: "dont_send",
			RetentionDays: 30, ShortStatus: "none"}, nil
	}
	if err != nil {
		return Rules{}, err
	}
	return Rules{
		BrandPrefix: row.BrandPrefix, OptOutText: row.OptOutText, OptOutInSteps: row.OptOutInSteps, ConfirmationOn: row.ConfirmationOn,
		Confirmation: row.ConfirmationText, RequireYes: row.RequireYes,
		Window:     channels.Window{From: int(row.QuietFrom), To: int(row.QuietTo)}.Clamp(),
		PushWindow: row.PushSameWindow, WhatsAppWindow: row.WhatsappSameWindow,
		Keywords:  sms.Keywords{OptOut: row.OptOutKeywords, OptIn: row.OptInKeywords, Help: row.HelpKeywords},
		HelpReply: row.HelpReply, OptOutReply: row.OptOutReply, OptInReply: row.OptInReply, KeywordsBy: row.KeywordsBy,
		Budget: row.Budget, BudgetAlertPct: int(row.BudgetAlertPct), BudgetSetBy: row.BudgetSetBy, BudgetSetAt: row.BudgetSetAt,
		PauseFailed: row.PauseFailedPct, PauseFiltered: row.PauseFilteredPct, PauseOptOut: row.PauseOptOutPct,
		OtherCountries: row.OtherCountries, OtherRouteID: row.OtherRouteID, RetentionDays: int(row.RetentionDays),
		ShortDomain: row.ShortDomain, ShortStatus: row.ShortDomainStatus, ShortTarget: row.ShortDomainTarget, ShortCheckedAt: row.ShortDomainCheckedAt,
		ShortVerifiedAt: row.ShortDomainVerifiedAt, ShortDetail: row.ShortDomainDetail, UpdatedBy: row.UpdatedBy, UpdatedAt: row.UpdatedAt,
	}, nil
}

// countryName is a country's English name ("GB" → "United Kingdom"); the
// app shows names in the reader's language.
func countryName(code string) string {
	r, err := language.ParseRegion(code)
	if err != nil {
		return code
	}
	if n := display.English.Regions().Name(r); n != "" {
		return n
	}
	return code
}

func validCountry(code string) bool {
	if len(code) != 2 || strings.ToUpper(code) != code {
		return false
	}
	_, err := language.ParseRegion(code)
	return err == nil
}

// round4 keeps prices to a hundredth of a cent; round2 amounts to cents.
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
func round2(v float64) float64 { return math.Floor(v*100+0.5+1e-7) / 100 }

// monthStart is the first moment of t's month in loc.
func monthStart(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, loc)
}

// spentThisMonth is the month's spend on texts.
func (s *Service) spentThisMonth(ctx context.Context, ws *store.Workspace) (float64, error) {
	var spent float64
	loc := location(ws.Timezone)
	// Texts handed to a provider: queued, held and skipped ones cost nothing yet.
	err := s.Pool.QueryRow(ctx, `SELECT coalesce(round(sum(cost), 4), 0)::float8 FROM channel_messages
		WHERE workspace_id = $1 AND channel = 'sms' AND status IN ('sending','sent','delivered','undelivered','failed')
		AND coalesce(sent_at, created_at) >= $2`, ws.ID, monthStart(s.now(), loc)).Scan(&spent)
	return spent, err
}

func location(tz string) *time.Location {
	if l, err := time.LoadLocation(tz); err == nil && tz != "" {
		return l
	}
	return time.UTC
}

func pct(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return math.Round(float64(n)*1000/float64(d)) / 10
}
