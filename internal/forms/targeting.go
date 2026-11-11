package forms

import (
	"context"
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Targeting (M60 G-24): where and when a popup or embedded form shows. The
// embed script checks the page, where the visitor came from, the device
// and whether this browser has seen a form before; the session endpoint
// checks the schedule (the server's clock) and the country (GeoIP, the
// address never stored). A form that isn't shown counts no view. Hosted
// forms take the schedule only: outside it the hosted page is closed.
// Rules are plain comparisons, never regular expressions, so nothing a
// publisher types runs on the public path.

// GeoIP finds the country of an address (internal/geoip; nil: off).
type GeoIP interface {
	Lookup(ip netip.Addr) (country, city string)
	Loaded() bool
}

// targetingDoc is a form's targeting (contentDoc.Targeting).
type targetingDoc struct {
	Pages     pageRulesDoc `json:"pages"`
	Sources   sourcesDoc   `json:"sources"`
	Devices   []string     `json:"devices,omitempty"`
	Countries countriesDoc `json:"countries"`
	Visitors  string       `json:"visitors,omitempty"`
	Schedule  scheduleDoc  `json:"schedule"`
}

type pageRulesDoc struct {
	Mode  string        `json:"mode,omitempty"`
	Rules []pageRuleDoc `json:"rules,omitempty"`
}

type pageRuleDoc struct {
	Op    string `json:"op"`
	Value string `json:"value"`
}

type sourcesDoc struct {
	UTMSource   []string `json:"utmSource,omitempty"`
	UTMMedium   []string `json:"utmMedium,omitempty"`
	UTMCampaign []string `json:"utmCampaign,omitempty"`
	Referrer    string   `json:"referrer,omitempty"`
}

type countriesDoc struct {
	Mode  string   `json:"mode,omitempty"`
	Codes []string `json:"codes,omitempty"`
}

type scheduleDoc struct {
	StartsAt *time.Time `json:"startsAt,omitempty"`
	EndsAt   *time.Time `json:"endsAt,omitempty"`
}

// SignupFormTargeting is where and when a form shows. Empty rules show it
// everywhere, to everyone; hosted forms keep the schedule only.
type SignupFormTargeting struct {
	Pages     SignupFormPageRules `json:"pages"`
	Sources   SignupFormSources   `json:"sources"`
	Devices   []string            `json:"devices" maxItems:"3" enum:"desktop,tablet,mobile" doc:"None: every device"`
	Countries SignupFormCountries `json:"countries"`
	Visitors  string              `json:"visitors" enum:"all,new,returning" doc:"Returning: this browser has seen one of the workspace's forms before today"`
	Schedule  SignupFormSchedule  `json:"schedule"`
}

// SignupFormPageRules picks the pages a form shows on, by the path and
// query after the domain.
type SignupFormPageRules struct {
	Mode  string               `json:"mode" enum:"all,only,except"`
	Rules []SignupFormPageRule `json:"rules" maxItems:"20"`
}

// SignupFormPageRule matches a page's path and query.
type SignupFormPageRule struct {
	Op    string `json:"op" enum:"contains,equals,starts_with"`
	Value string `json:"value" minLength:"1" maxLength:"200" example:"/pricing"`
}

// SignupFormSources picks visitors by where they came from: any of the
// listed UTM values (each list empty: any), and the kind of referrer.
type SignupFormSources struct {
	UTMSource   []string `json:"utmSource" maxItems:"10"`
	UTMMedium   []string `json:"utmMedium" maxItems:"10"`
	UTMCampaign []string `json:"utmCampaign" maxItems:"10"`
	Referrer    string   `json:"referrer" enum:"any,search,social,other_site,direct"`
}

// SignupFormCountries picks visitors by country (ISO 3166 alpha-2), found
// from their address with the server's GeoIP database.
type SignupFormCountries struct {
	Mode  string   `json:"mode" enum:"all,only,except"`
	Codes []string `json:"codes" maxItems:"50"`
}

// SignupFormSchedule is when a form shows; either end may be open.
type SignupFormSchedule struct {
	StartsAt *time.Time `json:"startsAt,omitempty"`
	EndsAt   *time.Time `json:"endsAt,omitempty"`
}

var (
	countryPattern = regexp.MustCompile(`^[A-Z]{2}$`)
	utmPattern     = regexp.MustCompile(`^[\p{L}\p{N} ._~+-]{1,100}$`)
)

// targetingOut is a form's targeting for the API (always set).
func targetingOut(t *targetingDoc) SignupFormTargeting {
	out := SignupFormTargeting{
		Pages:     SignupFormPageRules{Mode: "all", Rules: []SignupFormPageRule{}},
		Sources:   SignupFormSources{UTMSource: []string{}, UTMMedium: []string{}, UTMCampaign: []string{}, Referrer: "any"},
		Devices:   []string{},
		Countries: SignupFormCountries{Mode: "all", Codes: []string{}},
		Visitors:  "all",
	}
	if t == nil {
		return out
	}
	if t.Pages.Mode != "" {
		out.Pages.Mode = t.Pages.Mode
	}
	for _, r := range t.Pages.Rules {
		out.Pages.Rules = append(out.Pages.Rules, SignupFormPageRule{Op: r.Op, Value: r.Value})
	}
	out.Sources.UTMSource = append(out.Sources.UTMSource, t.Sources.UTMSource...)
	out.Sources.UTMMedium = append(out.Sources.UTMMedium, t.Sources.UTMMedium...)
	out.Sources.UTMCampaign = append(out.Sources.UTMCampaign, t.Sources.UTMCampaign...)
	if t.Sources.Referrer != "" {
		out.Sources.Referrer = t.Sources.Referrer
	}
	out.Devices = append(out.Devices, t.Devices...)
	if t.Countries.Mode != "" {
		out.Countries.Mode = t.Countries.Mode
	}
	out.Countries.Codes = append(out.Countries.Codes, t.Countries.Codes...)
	if t.Visitors != "" {
		out.Visitors = t.Visitors
	}
	out.Schedule = SignupFormSchedule{StartsAt: t.Schedule.StartsAt, EndsAt: t.Schedule.EndsAt}
	return out
}

func cleanList(ctx context.Context, in []string, loc string) ([]string, error) {
	var out []string
	for i, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if !utmPattern.MatchString(v) {
			return nil, api.Invalid(fmt.Sprintf("%s[%d]", loc, i), say(ctx, mTargetUTM, nil))
		}
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out, nil
}

// parseTargeting checks a form's targeting; nil means no rules. Hosted
// forms keep only the schedule.
func parseTargeting(ctx context.Context, in *SignupFormTargeting, kind string) (*targetingDoc, error) {
	if in == nil {
		return nil, nil
	}
	const loc = "body.content.targeting"
	t := &targetingDoc{}
	if s := in.Schedule; s.StartsAt != nil || s.EndsAt != nil {
		if s.StartsAt != nil && s.EndsAt != nil && !s.EndsAt.After(*s.StartsAt) {
			return nil, api.Invalid(loc+".schedule.endsAt", say(ctx, mTargetEnds, nil))
		}
		t.Schedule = scheduleDoc{StartsAt: utcPtr(s.StartsAt), EndsAt: utcPtr(s.EndsAt)}
	}
	if kind != KindHosted {
		if in.Pages.Mode == "only" || in.Pages.Mode == "except" {
			for i, r := range in.Pages.Rules {
				v := strings.TrimSpace(r.Value)
				if v == "" {
					return nil, api.Invalid(fmt.Sprintf("%s.pages.rules[%d].value", loc, i), say(ctx, mTargetRule, nil))
				}
				t.Pages.Rules = append(t.Pages.Rules, pageRuleDoc{Op: r.Op, Value: v})
			}
			if len(t.Pages.Rules) == 0 {
				return nil, api.Invalid(loc+".pages.rules", say(ctx, mTargetRule, nil))
			}
			t.Pages.Mode = in.Pages.Mode
		}
		var err error
		if t.Sources.UTMSource, err = cleanList(ctx, in.Sources.UTMSource, loc+".sources.utmSource"); err != nil {
			return nil, err
		}
		if t.Sources.UTMMedium, err = cleanList(ctx, in.Sources.UTMMedium, loc+".sources.utmMedium"); err != nil {
			return nil, err
		}
		if t.Sources.UTMCampaign, err = cleanList(ctx, in.Sources.UTMCampaign, loc+".sources.utmCampaign"); err != nil {
			return nil, err
		}
		if in.Sources.Referrer != "any" {
			t.Sources.Referrer = in.Sources.Referrer
		}
		for _, d := range []string{"desktop", "tablet", "mobile"} {
			if slices.Contains(in.Devices, d) {
				t.Devices = append(t.Devices, d)
			}
		}
		if len(t.Devices) == 3 {
			t.Devices = nil // every device
		}
		if in.Countries.Mode == "only" || in.Countries.Mode == "except" {
			for i, c := range in.Countries.Codes {
				c = strings.ToUpper(strings.TrimSpace(c))
				if !countryPattern.MatchString(c) {
					return nil, api.Invalid(fmt.Sprintf("%s.countries.codes[%d]", loc, i), say(ctx, mTargetCountry, i18n.Args{"code": c}))
				}
				if !slices.Contains(t.Countries.Codes, c) {
					t.Countries.Codes = append(t.Countries.Codes, c)
				}
			}
			if len(t.Countries.Codes) == 0 {
				return nil, api.Invalid(loc+".countries.codes", say(ctx, mTargetCountries, nil))
			}
			t.Countries.Mode = in.Countries.Mode
		}
		if in.Visitors == "new" || in.Visitors == "returning" {
			t.Visitors = in.Visitors
		}
	}
	if t.empty() {
		return nil, nil
	}
	return t, nil
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC().Truncate(time.Minute)
	return &u
}

func (t *targetingDoc) empty() bool {
	return t == nil || (t.Pages.Mode == "" && len(t.Sources.UTMSource) == 0 && len(t.Sources.UTMMedium) == 0 && len(t.Sources.UTMCampaign) == 0 &&
		t.Sources.Referrer == "" && len(t.Devices) == 0 && t.Countries.Mode == "" && t.Visitors == "" && t.Schedule.StartsAt == nil && t.Schedule.EndsAt == nil)
}

// Open reports whether a form's schedule lets it show at now.
func (c Config) Open(now time.Time) bool {
	t := c.Content.Targeting
	if t == nil {
		return true
	}
	if t.Schedule.StartsAt != nil && now.Before(*t.Schedule.StartsAt) {
		return false
	}
	if t.Schedule.EndsAt != nil && !now.Before(*t.Schedule.EndsAt) {
		return false
	}
	return true
}

// CountryShown reports whether a form shows to a visitor from country (""
// when unknown: an "only" rule then hides it, an "except" rule doesn't).
func (c Config) CountryShown(country string) bool {
	t := c.Content.Targeting
	if t == nil || t.Countries.Mode == "" {
		return true
	}
	in := country != "" && slices.Contains(t.Countries.Codes, country)
	if t.Countries.Mode == "only" {
		return in
	}
	return !in
}

// serverRules reports whether the session decides if the form shows (the
// schedule or countries), so the script waits for its answer.
func (c Config) serverRules() bool {
	t := c.Content.Targeting
	return t != nil && (t.Schedule.StartsAt != nil || t.Schedule.EndsAt != nil || t.Countries.Mode != "")
}

// GeoIPLoaded reports whether country rules can work on this server.
func (s *Service) GeoIPLoaded() bool { return s.GeoIP != nil && s.GeoIP.Loaded() }

func (s *Service) country(ip *netip.Addr) string {
	if s.GeoIP == nil || ip == nil {
		return ""
	}
	c, _ := s.GeoIP.Lookup(*ip)
	return c
}
