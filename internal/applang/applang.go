// Package applang holds what a member reads ReputeCast in: their app
// language and region format (Account & security › Language & region), and
// the instance defaults for the sign-in page, members who haven't chosen
// and invitations (connect-and-reach brief §3.7, §5.9). The languages and
// formats themselves are internal/i18n's.
package applang

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service saves members' language choices.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
}

// What the API stores when a member hasn't chosen: the language follows the
// instance default, the region follows the language, the clock and the
// week follow the region.
const (
	FollowDefault = "default"
	Auto          = "auto"
)

// MemberLanguage is what a member chose in Language & region.
type MemberLanguage struct {
	Language     string `json:"language" enum:"default,en,vi,en-XA" doc:"default follows the instance default"`
	RegionFormat string `json:"regionFormat" enum:"auto,standard,en-US,vi-VN,de-DE,en-GB,es-ES,fr-FR,pt-BR" doc:"auto follows the app language"`
	HourCycle    string `json:"hourCycle" enum:"auto,h23,h12" doc:"auto follows the region format"`
	WeekStart    string `json:"weekStart" enum:"auto,monday,sunday" doc:"auto follows the region format"`
}

// LanguageOption is an app language, named in itself.
type LanguageOption struct {
	Code string `json:"code" example:"vi"`
	Name string `json:"name" example:"Tiếng Việt"`
}

// Choices returns what a member chose.
func Choices(u store.User) MemberLanguage {
	out := MemberLanguage{Language: FollowDefault, RegionFormat: Auto, HourCycle: Auto, WeekStart: Auto}
	// A language that can't be picked any more (the pseudo-locale on a server
	// that turned it off) reads as the instance default.
	if u.Language != nil {
		if l, ok := i18n.ParseLocale(*u.Language); ok {
			out.Language = string(l)
		}
	}
	if u.RegionFormat != nil && i18n.Region(*u.RegionFormat).Valid() {
		out.RegionFormat = *u.RegionFormat
	}
	if u.HourCycle != nil {
		out.HourCycle = *u.HourCycle
	}
	if u.WeekStart != nil {
		out.WeekStart = *u.WeekStart
	}
	return out
}

// Languages returns the app languages the server can answer in, in picker
// order (the pseudo-locale last, when it's on).
func Languages() []LanguageOption {
	out := []LanguageOption{}
	for _, l := range i18n.Locales() {
		out = append(out, LanguageOption{Code: string(l), Name: l.Name()})
	}
	return out
}

// LanguageCodes is Languages without the names.
func LanguageCodes() []string {
	out := []string{}
	for _, l := range i18n.Locales() {
		out = append(out, string(l))
	}
	return out
}

func optional(v string, follow string) *string {
	if v == "" || v == follow {
		return nil
	}
	return &v
}

func deref(p *string, follow string) string {
	if p == nil {
		return follow
	}
	return *p
}

// Save stores a member's choices. Saving is audited as account.language_update.
func (s *Service) Save(ctx context.Context, in MemberLanguage) (MemberLanguage, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return MemberLanguage{}, api.Err(401, api.CodeUnauthenticated, "Sign in to continue.")
	}
	if in.Language != FollowDefault {
		l, ok := i18n.ParseLocale(in.Language)
		if !ok || string(l) != in.Language {
			if in.Language == string(i18n.Pseudo) {
				return MemberLanguage{}, api.Invalid("body.language", "The pseudo-locale is only available in development, or when the server runs with i18n.pseudo_locale on.")
			}
			return MemberLanguage{}, api.Invalid("body.language", "ReputeCast isn't translated into that language. Pick English or Tiếng Việt.")
		}
	}
	if in.RegionFormat != Auto && !i18n.Region(in.RegionFormat).Valid() {
		return MemberLanguage{}, api.Invalid("body.regionFormat", "Pick one of the region formats in the list.")
	}
	before := Choices(*u)
	var saved store.User
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		saved, err = q.AppLangSetUser(ctx, store.AppLangSetUserParams{
			ID: u.ID, Language: optional(in.Language, FollowDefault), RegionFormat: optional(in.RegionFormat, Auto),
			HourCycle: optional(in.HourCycle, Auto), WeekStart: optional(in.WeekStart, Auto),
		})
		if err != nil {
			return err
		}
		after := Choices(saved)
		if after == before {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.language_update", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Name,
			Before: map[string]any{"language": before.Language, "region_format": before.RegionFormat, "hour_cycle": before.HourCycle, "week_start": before.WeekStart},
			After:  map[string]any{"language": after.Language, "region_format": after.RegionFormat, "hour_cycle": after.HourCycle, "week_start": after.WeekStart}})
	})
	if err != nil {
		return MemberLanguage{}, err
	}
	return Choices(saved), nil
}

// ErrUnknownLanguage is an instance default or invitation language
// ReputeCast isn't translated into.
var ErrUnknownLanguage = errors.New("applang: unknown language")

// CheckDefault validates an instance default: a real app language (never
// the pseudo-locale) and a region format.
func CheckDefault(language, region string) error {
	if l, ok := i18n.ParseLocale(language); !ok || l == i18n.Pseudo || string(l) != language {
		return ErrUnknownLanguage
	}
	if !i18n.Region(region).Valid() {
		return ErrUnknownLanguage
	}
	return nil
}

// ForUser returns how a member reads their email: their choices, else the
// instance defaults. Unknown members read the instance default.
func ForUser(ctx context.Context, q *store.Queries, u store.User) i18n.Lang {
	def, err := q.AppLangInstanceDefaults(ctx)
	if err != nil {
		return i18n.Default
	}
	return i18n.Resolve(deref(u.Language, ""), deref(u.RegionFormat, ""), def.DefaultLanguage, def.DefaultRegionFormat)
}

// ForEmail returns how the person with this email reads, or the instance
// default when nobody has it (an invitee).
func ForEmail(ctx context.Context, q *store.Queries, email string) i18n.Lang {
	if r, err := q.AppLangForEmail(ctx, email); err == nil {
		return i18n.Resolve(r.Language, r.RegionFormat, r.DefaultLanguage, r.DefaultRegionFormat)
	}
	return InstanceDefault(ctx, q)
}

// ForUserID is ForUser by ID.
func ForUserID(ctx context.Context, q *store.Queries, id uuid.UUID) i18n.Lang {
	r, err := q.AppLangForUser(ctx, id)
	if err != nil {
		return InstanceDefault(ctx, q)
	}
	return i18n.Resolve(r.Language, r.RegionFormat, r.DefaultLanguage, r.DefaultRegionFormat)
}

// InstanceDefault is how people without an account read: the instance's
// default language and region format.
func InstanceDefault(ctx context.Context, q *store.Queries) i18n.Lang {
	def, err := q.AppLangInstanceDefaults(ctx)
	if err != nil {
		return i18n.Default
	}
	return i18n.Resolve("", "", def.DefaultLanguage, def.DefaultRegionFormat)
}

// ForInvitation is how an invitation reads: its own language, or the
// instance default. A language that isn't the instance's uses its own
// region format.
func ForInvitation(ctx context.Context, q *store.Queries, language string) i18n.Lang {
	def, err := q.AppLangInstanceDefaults(ctx)
	if err != nil {
		def = store.AppLangInstanceDefaultsRow{DefaultLanguage: "en", DefaultRegionFormat: string(i18n.Standard)}
	}
	if language == "" || language == def.DefaultLanguage {
		return i18n.Resolve("", "", def.DefaultLanguage, def.DefaultRegionFormat)
	}
	return i18n.Resolve(language, "", def.DefaultLanguage, def.DefaultRegionFormat)
}
