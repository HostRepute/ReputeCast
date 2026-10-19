package revenue

import (
	"context"
	"errors"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// AttributionSettings is Earn › Settings › Attribution.
type AttributionSettings struct {
	// Days is the attribution window: an upgrade or a conversion is credited
	// to the last email clicked within it.
	Days int `json:"days" minimum:"1" maximum:"30"`
	// ClickToken adds rc_click=… to links to Domains.
	ClickToken bool     `json:"clickToken"`
	Domains    []string `json:"domains"`
	// Endpoint is the conversion API's address.
	Endpoint string          `json:"endpoint"`
	Keys     []ConversionKey `json:"keys"`
	// The conversions log's totals in the workspace currency.
	Summary  ConversionSummary `json:"summary"`
	Currency string            `json:"currency"`
	// CanManage: the person may change these (payments:manage).
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
	// CanCreateKey: the person may create API keys.
	CanCreateKey bool `json:"canCreateKey"`
}

// ConversionKey is an API key that can record conversions.
type ConversionKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedBy  string     `json:"createdBy"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	LastUsedIP string     `json:"lastUsedIp,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	// Only: the key holds conversions:create and nothing else.
	Only bool `json:"only"`
}

// ConversionSummary is the conversions log's totals.
type ConversionSummary struct {
	Count         int64      `json:"count"`
	ValueCents    int64      `json:"valueCents"`
	Credited      int64      `json:"credited"`
	CreditedCents int64      `json:"creditedCents"`
	Since         *time.Time `json:"since,omitempty"`
	// Name, TypicalCents and Sender describe a single kind of conversion.
	Name         string `json:"name,omitempty"`
	TypicalCents int64  `json:"typicalCents,omitempty"`
	Sender       string `json:"sender,omitempty"`
}

// AttributionInput changes the attribution settings.
type AttributionInput struct {
	Days       int      `json:"days" doc:"The attribution window, 1 to 30 days."`
	ClickToken bool     `json:"clickToken"`
	Domains    []string `json:"domains" maxItems:"20" doc:"Your own domains (app.acme.io): the click token goes on links to them only."`
}

var hostRe = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9-]{0,61}[a-z0-9]$`)

// cleanDomain is a domain as the settings keep it: lower case, no scheme,
// path or port ("https://App.Acme.io/x" → app.acme.io).
func cleanDomain(raw string) (string, bool) {
	d := strings.ToLower(strings.TrimSpace(raw))
	if strings.Contains(d, "://") {
		if u, err := url.Parse(d); err == nil {
			d = u.Host
		}
	}
	d = strings.TrimPrefix(d, "*.")
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	if h, _, err := net.SplitHostPort(d); err == nil {
		d = h
	}
	d = strings.TrimSuffix(d, ".")
	return d, hostRe.MatchString(d)
}

// Attribution reads the attribution settings, the keys that can record
// conversions and the log's totals.
func (s *Service) Attribution(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AttributionSettings, error) {
	out := AttributionSettings{Days: 7, Domains: []string{}, Keys: []ConversionKey{}, Helpers: []string{}, Currency: "usd",
		Endpoint:  strings.TrimRight(s.URLs.Base(), "/") + "/api/v1/workspaces/" + ws.Slug + "/conversions",
		CanManage: p.Can(authz.PaymentsManage), CanCreateKey: p.Can(authz.APIKeysManage)}
	st, err := s.Q.RevenueAttributionSettings(ctx, ws.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if err == nil {
		out.Days, out.ClickToken = int(st.AttributionDays), st.ClickTokenOn
		if st.ClickTokenDomains != nil {
			out.Domains = st.ClickTokenDomains
		}
		if st.Currency != "" {
			out.Currency = st.Currency
		}
	}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws.ID, p.UserID, 3, authz.PaymentsManage)
	}
	keys, err := s.Q.RevenueConversionKeys(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, k := range keys {
		ck := ConversionKey{ID: k.ID.String(), Name: k.Name, Prefix: k.Prefix, CreatedBy: k.CreatedByName, CreatedAt: k.CreatedAt, LastUsedAt: k.LastUsedAt, ExpiresAt: k.ExpiresAt,
			Only: len(k.Permissions) == 1}
		if k.LastUsedIp != nil {
			ck.LastUsedIP = k.LastUsedIp.String()
		}
		out.Keys = append(out.Keys, ck)
	}
	c, err := s.Q.RevenueConversionCounts(ctx, store.RevenueConversionCountsParams{WorkspaceID: ws.ID, Currency: out.Currency})
	if err != nil {
		return out, err
	}
	out.Summary = ConversionSummary{Count: c.Conversions, ValueCents: c.ValueCents, Credited: c.Credited, CreditedCents: c.CreditedCents}
	if c.Conversions > 0 {
		f := c.FirstAt
		out.Summary.Since = &f
		kinds, err := s.Q.RevenueConversionKinds(ctx, store.RevenueConversionKindsParams{WorkspaceID: ws.ID, FromAt: c.FirstAt.Add(-time.Second), ToAt: s.now().Add(time.Minute)})
		if err != nil {
			return out, err
		}
		if len(kinds) == 1 {
			out.Summary.Name, out.Summary.TypicalCents, out.Summary.Sender = kinds[0].Name, kinds[0].ValueCents, conversionSender(kinds[0].KeyName)
		}
	}
	return out, nil
}

// SetAttribution changes the window and the click token (audited).
func (s *Service) SetAttribution(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AttributionInput) (AttributionSettings, error) {
	if in.Days < 1 || in.Days > 30 {
		return AttributionSettings{}, invalid(ctx, "body.days", "problem.window", nil)
	}
	domains := []string{}
	for i, raw := range in.Domains {
		d, ok := cleanDomain(raw)
		if !ok {
			return AttributionSettings{}, invalidAt(ctx, "body.domains", i, "problem.domain", map[string]any{"domain": strings.TrimSpace(raw)})
		}
		if !slices.Contains(domains, d) {
			domains = append(domains, d)
		}
	}
	if in.ClickToken && len(domains) == 0 {
		return AttributionSettings{}, invalid(ctx, "body.domains", "problem.domainsNeeded", nil)
	}
	before, err := s.Q.RevenueAttributionSettings(ctx, ws.ID)
	if err != nil {
		return AttributionSettings{}, err
	}
	if before.ClickTokenDomains == nil {
		before.ClickTokenDomains = []string{}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RevenueSetAttribution(ctx, store.RevenueSetAttributionParams{WorkspaceID: ws.ID, AttributionDays: int32(in.Days), ClickTokenOn: in.ClickToken,
			ClickTokenDomains: domains, At: s.now(), UpdatedBy: userID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.attribution_updated", TargetType: "earn_settings", TargetID: ws.ID.String(),
			TargetName: ws.Name, Before: map[string]any{"days": before.AttributionDays, "clickToken": before.ClickTokenOn, "domains": before.ClickTokenDomains},
			After: map[string]any{"days": in.Days, "clickToken": in.ClickToken, "domains": domains}})
	})
	if err != nil {
		return AttributionSettings{}, err
	}
	s.settings.Delete(ws.ID)
	return s.Attribution(ctx, ws, p)
}

// cachedSettings is a workspace's click token settings for a minute, so a
// click's redirect seldom reads them.
type cachedSettings struct {
	on      bool
	domains []string
	days    int
	until   time.Time
}

func (s *Service) clickSettings(ctx context.Context, wsID uuid.UUID) cachedSettings {
	if v, ok := s.settings.Load(wsID); ok {
		if c := v.(cachedSettings); s.now().Before(c.until) {
			return c
		}
	}
	c := cachedSettings{days: 7, until: s.now().Add(time.Minute)}
	if st, err := s.Q.RevenueAttributionSettings(ctx, wsID); err == nil {
		c.on, c.domains, c.days = st.ClickTokenOn, st.ClickTokenDomains, int(st.AttributionDays)
	}
	s.settings.Store(wsID, c)
	return c
}

// ClickTokenFor signs the click token of a reader's click on an email.
func (s *Service) ClickTokenFor(wsID, messageID, contactID uuid.UUID, at time.Time) string {
	return s.Links.Sign(linktoken.Claims{Kind: ClickToken, WorkspaceID: wsID, MessageID: messageID, ContactID: contactID, Link: uint32(at.Unix() / 60)}) //nolint:gosec // minutes since 1970 fit in 32 bits until the year 10136
}

// matchesDomain reports whether host is one of the domains or below one.
func matchesDomain(host string, domains []string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, d := range domains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// sameLink compares two links by host and path.
func sameLink(a, b string) bool {
	ua, err1 := url.Parse(strings.TrimSpace(a))
	ub, err2 := url.Parse(strings.TrimSpace(b))
	if err1 != nil || err2 != nil || ua.Host == "" {
		return false
	}
	return strings.EqualFold(ua.Host, ub.Host) && strings.TrimSuffix(ua.Path, "/") == strings.TrimSuffix(ub.Path, "/")
}

// Decorate adds the click token to a click's destination when the token is
// on and the link goes to one of the publisher's domains, never to a
// sponsor's link.
func (s *Service) Decorate(ctx context.Context, c tracking.ClickTarget) string {
	if s.Links == nil {
		return ""
	}
	st := s.clickSettings(ctx, c.WorkspaceID)
	if !st.on || len(st.domains) == 0 {
		return ""
	}
	u, err := url.Parse(c.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !matchesDomain(u.Hostname(), st.domains) || u.Query().Has("rc_click") {
		return ""
	}
	if m, err := s.Q.RevenueMessageOf(ctx, c.MessageID); err == nil && m.CampaignID != nil {
		if links, err := s.Q.RevenueSponsorLinks(ctx, m.CampaignID); err == nil {
			for _, l := range links {
				if l != "" && sameLink(l, c.URL) {
					return ""
				}
			}
		}
	}
	tok := s.ClickTokenFor(c.WorkspaceID, c.MessageID, c.ContactID, c.At)
	if u.RawQuery == "" {
		u.RawQuery = "rc_click=" + tok
	} else {
		u.RawQuery += "&rc_click=" + tok
	}
	return u.String()
}

var upgradePath = regexp.MustCompile(`^(/p/[^/]+)?/upgrade(/[^/]+)?/?$`)

// IsUpgradeLink reports whether a link goes to an upgrade page (the hosted
// page, a hidden tier's own link, or the {{ upgrade_url }} variable).
func IsUpgradeLink(raw string) bool {
	if strings.Contains(raw, "upgrade_url") {
		return true
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	p := u.Path
	if strings.HasSuffix(p, "/upgrade/checkout") || strings.HasSuffix(p, "/upgrade/welcome") {
		return false
	}
	return upgradePath.MatchString(p)
}

// Clicked counts a person's click on an upgrade link of a campaign (the
// report's Revenue section); bots never reach it.
func (s *Service) Clicked(ctx context.Context, q *store.Queries, c tracking.HumanClick) error {
	if c.CampaignID == nil || !IsUpgradeLink(c.URL) {
		return nil
	}
	return q.RevenueCampaignClickBump(ctx, store.RevenueCampaignClickBumpParams{CampaignID: *c.CampaignID, WorkspaceID: c.WorkspaceID})
}
