package readiness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// Deliverability is what a run reads from the deliverability domain:
// blocklist results, authentication and provider account health.
type Deliverability interface {
	Overview(ctx context.Context, ws *store.Workspace) (deliverability.DeliverabilityOverview, error)
	Reputation(ctx context.Context, ws *store.Workspace) (deliverability.DeliverabilityReputation, error)
}

// URIBlocklists are the URI blocklists link domains are checked on.
var URIBlocklists = []string{"SURBL", "URIBL", "Spamhaus DBL"}

// stage names the step a run is in; groups tick in as their stage finishes.
type stage string

const (
	stagePrepare  stage = "prepare"
	stageAuth     stage = GroupAuthentication
	stageContent  stage = GroupContent
	stageAccess   stage = GroupAccessibility
	stageComply   stage = GroupCompliance
	stageReput    stage = GroupReputation
	stageLinks    stage = GroupLinks
	stageAudience stage = GroupAudience
	stageRouting  stage = GroupRouting
	stageLang     stage = GroupLanguages
	stageCond     stage = GroupConditional
)

// runOrder is the order groups run (and tick in on the review screen).
var runOrder = []string{GroupAuthentication, GroupContent, GroupLanguages, GroupAccessibility, GroupConditional, GroupCompliance, GroupReputation, GroupLinks, GroupAudience, GroupRouting}

// sunsetDays is the number of days without an open or click after which a
// contact is unengaged: the last day before the sunset tier of the
// workspace's sunset policy.
func sunsetDays(raw json.RawMessage) int {
	if d := audience.ParsePolicy(raw).SunsetDays - 1; d > 0 {
		return d
	}
	return 180
}

// Schedule is campaigns.schedule.
type Schedule struct {
	Mode     string `json:"mode"`
	At       string `json:"at,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// ParseSchedule reads campaigns.schedule (mode "now" when unset).
func ParseSchedule(raw json.RawMessage) Schedule {
	var s Schedule
	_ = json.Unmarshal(raw, &s)
	if s.Mode == "" {
		s.Mode = "now"
	}
	return s
}

// StartAt is when a send with this schedule would start.
func (s Schedule) StartAt(now time.Time) time.Time {
	if s.Mode == "now" || s.At == "" {
		return now
	}
	if t, err := time.Parse(time.RFC3339, s.At); err == nil {
		return t
	}
	return now
}

// gathered keeps what the stages share.
type gathered struct {
	ws         *store.Workspace
	c          store.ReadinessCampaignRow
	recipients []Recipient
	provs      []store.Provider
	byID       map[uuid.UUID]store.Provider
}

// gather collects the facts of a run, calling done after each stage.
func (s *Service) gather(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, done func(stage, Facts)) (Facts, error) {
	now := s.now()
	loc := state.Location(ws.Timezone)
	sched := ParseSchedule(c.Schedule)
	f := Facts{Now: now, Location: loc, Workspace: ws.Slug, MailboxRates: DefaultMailboxRates}
	f.Campaign = CampaignFacts{ID: c.ID.String(), Name: c.Name, Subject: c.Subject, Preheader: c.Preheader, Version: int(c.Version),
		SenderName: c.SenderName, SenderEmail: c.SenderEmail, SenderDomain: c.SenderDomain, StartAt: sched.StartAt(now), ScheduleMode: sched.Mode}
	compiled := c.UpdatedAt
	if c.AutosavedAt != nil {
		compiled = *c.AutosavedAt
	}
	f.Campaign.CompiledAt = &compiled
	g := &gathered{ws: ws, c: c, byID: map[uuid.UUID]store.Provider{}}

	// Prepare: the audience and the route every group refers to.
	def := ParseAudience(c.Audience)
	aud, recipients, err := s.computeAudience(ctx, s.Q, ws.ID, def, sunsetDays(ws.SunsetPolicy), now)
	if err != nil {
		return f, fmt.Errorf("audience: %w", err)
	}
	f.Audience, g.recipients = aud, recipients
	if g.provs, err = s.Q.ProvListActive(ctx, ws.ID); err != nil {
		return f, err
	}
	for _, p := range g.provs {
		g.byID[p.ID] = p
	}
	if f.Route, err = s.route(ctx, g, f); err != nil {
		return f, fmt.Errorf("route: %w", err)
	}
	done(stagePrepare, f)

	if f.Auth, err = s.auth(ctx, g, f); err != nil {
		return f, fmt.Errorf("authentication: %w", err)
	}
	done(stageAuth, f)

	// Content (without requesting links, which is the slow part).
	f.Content = ContentFacts{Address: templates.WorkspaceAddress(ws), URIBlocklists: URIBlocklists, OwnDomain: contentcheck.RegistrableDomain(c.SenderDomain)}
	if s.Campaigns != nil && (strings.TrimSpace(c.Mjml) != "" || strings.TrimSpace(c.Html) != "") {
		chk, err := s.Campaigns.Checks(ctx, ws, c.ID, campaigns.ComposerChecksRequest{Subject: c.Subject, Preheader: c.Preheader, MJML: c.Mjml, HTML: c.Html})
		if err == nil {
			f.Content.Report, f.Content.Checked, f.Content.FooterRemovedAt = chk.ContentCheckReport, true, chk.FooterRemovedAt
			f.Content.InlineCSSBytes = inlineCSS(c.Html)
		} else if s.Log != nil {
			s.Log.WarnContext(ctx, "readiness content check failed", "campaign", c.ID, "err", err)
		}
	}
	if s.Conditional != nil {
		f.Polls = s.Conditional.PollFacts(ctx, ws, c.ID, c.Mjml)
	}
	done(stageContent, f)

	// Languages: who gets which version, and the content checks of every
	// version that is sent.
	if s.Languages != nil {
		lf, err := s.Languages.CampaignReadiness(ctx, ws, c.ID, f.Audience.Languages)
		if err != nil {
			return f, fmt.Errorf("languages: %w", err)
		}
		f.Languages = lf
		if f.Content.Checked && s.Campaigns != nil {
			for _, v := range lf.Versions {
				chk, err := s.Campaigns.Checks(ctx, ws, c.ID, campaigns.ComposerChecksRequest{Subject: v.Subject, Preheader: v.Preheader, MJML: v.MJML, HTML: v.HTML, Language: v.Language})
				if err != nil {
					return f, fmt.Errorf("languages: %s content: %w", v.Language, err)
				}
				f.Content.Versions = append(f.Content.Versions, VersionReport{Language: v.Language, Report: chk.ContentCheckReport})
			}
		}
	}
	done(stageLang, f)

	// Accessibility: every language version, on the content as saved.
	if s.Accessibility != nil {
		f.Accessibility = s.Accessibility.ReadinessFacts(ctx, ws, accessibility.CampaignContent{ID: c.ID, Name: c.Name, Status: c.Status, MJML: c.Mjml, HTML: c.Html, Hash: c.ContentHash})
		done(stageAccess, f)
	}

	// Conditional content: every version the audience gets; and the polls
	// and surveys of the content (design expansion, X11).
	if s.Conditional != nil {
		f.Conditional = s.Conditional.ConditionalFacts(ctx, ws, c.ID)
		done(stageCond, f)
	}

	f.Compliance = s.compliance(ws, f)
	done(stageComply, f)

	if f.Reputation, err = s.reputation(ctx, g, f); err != nil {
		return f, fmt.Errorf("reputation: %w", err)
	}
	done(stageReput, f)

	// Links: the content check again, requesting every link.
	if f.Content.Checked && s.Campaigns != nil {
		chk, err := s.Campaigns.Checks(ctx, ws, c.ID, campaigns.ComposerChecksRequest{Subject: c.Subject, Preheader: c.Preheader, MJML: c.Mjml, HTML: c.Html, ProbeLinks: true})
		if err == nil {
			f.Content.Report, f.Content.FooterRemovedAt = chk.ContentCheckReport, chk.FooterRemovedAt
		}
	}
	done(stageLinks, f)
	// Address checks: the verification results of the final recipients.
	if s.Verification != nil && !f.Audience.Empty {
		f.Verification = s.Verification.ReadinessFacts(ctx, ws, c.ID, g.recipients, f.Audience.SuppressedEmails)
	}
	done(stageAudience, f)
	done(stageRouting, f)
	return f, nil
}

var styleAttr = regexp.MustCompile(`(?i)\sstyle="([^"]*)"`)

// inlineCSS adds up the inline style attributes of the HTML.
func inlineCSS(html string) int {
	n := 0
	for _, m := range styleAttr.FindAllStringSubmatch(html, -1) {
		n += len(m[1])
	}
	return n
}

// route plans the send over the stream's rules and describes each provider.
func (s *Service) route(ctx context.Context, g *gathered, f Facts) (RouteFacts, error) {
	out := RouteFacts{Stream: string(providers.StreamBroadcast), Shares: []ShareFacts{}}
	set, err := routing.Load(ctx, s.Q, g.ws.ID)
	if err != nil {
		return out, err
	}
	out.Policy = string(set.Policy(providers.StreamBroadcast))
	override := ParseRouteOverride(g.c.RouteOverride)
	out.Override = override != nil
	counts, order, ok := planRoute(set, override, g.recipients, g.c.SenderDomain)
	if !ok {
		out.NoRoute = true
		return out, nil
	}
	var provs []store.Provider
	for _, id := range order {
		if p, found := g.byID[id]; found {
			provs = append(provs, p)
		}
	}
	// Without recipients, shares are the split's weights.
	weights, totalWeight := map[uuid.UUID]int{}, 0
	splits := []routing.Split{}
	if override != nil {
		splits = override.Splits
	} else if def, ok := set.Default(providers.StreamBroadcast); ok {
		splits = def.Splits
	}
	for _, sp := range splits {
		weights[sp.ProviderID] += max(sp.Weight, 0)
		totalWeight += max(sp.Weight, 0)
	}
	usage, err := manage.UsageOf(ctx, s.Q, g.ws, provs, f.Now)
	if err != nil {
		return out, err
	}
	loc := f.Location
	for _, p := range provs {
		caps := state.Caps(p)
		h := state.HealthOf(p)
		u := usage[p.ID]
		sh := ShareFacts{ProviderID: p.ID.String(), Name: p.Name, Short: state.Short(p.Name, p.Adapter), Adapter: p.Adapter, Label: p.Label,
			Recipients: counts[p.ID], Status: p.Status, Available: state.Available(p), Pricing: state.PricingOf(p), Limits: state.LimitsOf(p),
			Account: h.Account, Sandbox: h.Sandbox || p.Status == state.Sandbox, Quota: h.Quota, BulkAllowed: caps.BulkAllowed(), Terms: termsOf(p, caps),
			Tracking: state.TrackingOf(p), IPs: p.Ips, ConnectedAt: p.ConnectedAt, Bounce: u.Bounce7d, Complaint: u.Complaint7d,
			ProviderReported: state.ReputationOf(h, f.Now) != nil}
		if p.IpMode != "dedicated" {
			sh.IPs = nil
		}
		if f.Audience.Final > 0 {
			sh.Percent = float64(sh.Recipients) / float64(f.Audience.Final) * 100
		} else if totalWeight > 0 {
			sh.Percent = float64(weights[p.ID]) * 100 / float64(totalWeight)
		}
		if h.Down != nil {
			since := h.Down.Since
			sh.DownSince = &since
			sh.DownDetail = downDetail(p, h, f.Now, loc)
		}
		if h.Rate != nil && h.Rate.Max > 0 {
			sh.Rate = h.Rate.Max
		}
		if plan, err := s.Q.WarmupActiveForProvider(ctx, p.ID); err == nil {
			wp := warmup.ParsePlan(plan.Schedule, plan.Progress)
			day := min(max(state.WarmupDay(plan.StartDate, f.Campaign.StartAt, loc), 1), int(plan.Days))
			sh.WarmupDay, sh.WarmupDays = day, int(plan.Days)
			sh.WarmupCap = wp.Cap(day)
			sh.WarmupToday = sameDay(f.Campaign.StartAt, f.Now, loc)
			sh.Rate, sh.RateRamp = wp.Rate(day), true
			if p.Status == state.Healthy {
				sh.Status = state.WarmingUp
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
		out.Shares = append(out.Shares, sh)
	}
	return out, nil
}

func sameDay(a, b time.Time, loc *time.Location) bool {
	x, y := a.In(loc), b.In(loc)
	return x.Year() == y.Year() && x.YearDay() == y.YearDay()
}

func termsOf(p store.Provider, caps state.Snapshot) string {
	switch caps.Bulk {
	case providers.BulkAllowed:
		if p.Adapter == "ses" {
			return "yes, with production access"
		}
		return "yes"
	case providers.BulkYourPolicy:
		return "self-hosted, your own policy"
	case providers.BulkBroadcastStreams:
		if caps.BroadcastStream {
			return "on its Broadcast stream"
		}
		return "only on a Broadcast stream, and this route uses its transactional stream"
	}
	return "transactional mail only"
}

func downDetail(p store.Provider, h state.Health, now time.Time, loc *time.Location) string {
	since := Clock(h.Down.Since, now, loc)
	if h.Down.Reason == "auth" {
		return fmt.Sprintf("%s has rejected the credentials since %s", p.Name, since)
	}
	who := p.Name
	var cfg struct {
		ServerURL string `json:"server_url"`
		Host      string `json:"host"`
	}
	_ = json.Unmarshal(p.Config, &cfg)
	if host := strings.TrimPrefix(strings.TrimPrefix(firstNonEmpty(cfg.ServerURL, cfg.Host), "https://"), "http://"); host != "" {
		who = strings.TrimSuffix(host, "/")
	}
	s := fmt.Sprintf("%s has refused connections since %s", who, since)
	if h.LastError != nil && h.LastError.Code != "" {
		s += " (" + h.LastError.Code + ")"
	}
	return s
}

// auth reads the stored authentication check of the sending domain.
func (s *Service) auth(ctx context.Context, g *gathered, f Facts) (AuthFacts, error) {
	domain := strings.ToLower(g.c.SenderDomain)
	out := AuthFacts{Domain: domain, SPF: []ProviderAuthFacts{}, DKIM: []ProviderAuthFacts{}, MailFrom: []ProviderAuthFacts{}}
	if domain == "" {
		return out, nil
	}
	checks, err := s.Q.ListDeliverabilityAssetChecks(ctx, g.ws.ID)
	if err != nil {
		return out, err
	}
	var auth *dnscheck.DomainAuth
	var orgDMARC *dnscheck.DMARC
	for _, chk := range checks {
		var r struct {
			Auth  *dnscheck.DomainAuth `json:"auth"`
			DMARC *dnscheck.DMARC      `json:"dmarc"`
		}
		if json.Unmarshal(chk.Result, &r) != nil {
			continue
		}
		switch {
		case strings.EqualFold(chk.Asset, domain) && r.Auth != nil:
			auth = r.Auth
			t := chk.CheckedAt
			out.CheckedAt = &t
		case strings.EqualFold(chk.Asset, dnscheck.OrgDomain(domain)) && r.DMARC != nil:
			orgDMARC = r.DMARC
		}
	}
	if auth == nil {
		return out, nil
	}
	out.Checked = true
	out.SPFRecord = auth.SPF.Record
	dmarc := auth.DMARC
	if dmarc.Status != dnscheck.StatusPass && orgDMARC != nil && orgDMARC.Status == dnscheck.StatusPass {
		dmarc = *orgDMARC
	}
	out.DMARCStatus, out.DMARCPolicy, out.DMARCRecord = dmarc.Status, dmarc.Policy, dmarc.Record
	out.BIMIStatus, out.BIMILogo = auth.BIMI.Status, auth.BIMI.Logo
	if s.Deliverability != nil {
		if ov, err := s.Deliverability.Overview(ctx, g.ws); err == nil {
			for _, a := range ov.Assets {
				if strings.EqualFold(a.Asset, domain) && a.DMARCAligned != nil {
					v := *a.DMARCAligned
					out.DMARCAligned = &v
				}
			}
		}
	}
	// Each provider carrying the send: its SPF, DKIM and MAIL FROM records.
	for _, sh := range carrying(f.Route.Shares) {
		var spfRecs, dkimRecs, rpRecs []dnscheck.RecordCheck
		for _, rc := range auth.Records {
			if rc.ProviderID != sh.ProviderID {
				continue
			}
			switch rc.Purpose {
			case "spf":
				spfRecs = append(spfRecs, rc)
			case "dkim":
				dkimRecs = append(dkimRecs, rc)
			case "return_path":
				rpRecs = append(rpRecs, rc)
			}
		}
		// SPF: the provider's SPF records, or the SPF of its MAIL FROM domain,
		// or the domain's own record when it has neither.
		spf := ProviderAuthFacts{Short: sh.Short, OK: auth.SPF.Status == dnscheck.StatusPass}
		if len(spfRecs) > 0 {
			spf.OK = allVerified(spfRecs)
			if !spf.OK {
				spf.Detail = "add " + spfRecs[0].Value + " to the SPF record of " + spfRecs[0].Name
			}
		} else if len(rpRecs) > 0 {
			for _, mf := range auth.MailFrom {
				if strings.EqualFold(mf.Domain, rpRecs[0].Name) {
					spf.OK = mf.SPF == dnscheck.StatusPass
				}
			}
		}
		out.SPF = append(out.SPF, spf)
		if len(dkimRecs) > 0 {
			dk := ProviderAuthFacts{Short: sh.Short, OK: allVerified(dkimRecs)}
			for _, k := range auth.DKIM {
				for _, rc := range dkimRecs {
					if sel, _, ok := strings.Cut(strings.ToLower(rc.Name), "._domainkey."); ok && strings.EqualFold(k.Selector, sel) {
						if k.Status != dnscheck.StatusPass {
							dk.OK = false
						}
						if k.KeyBits > 0 && (dk.KeyBits == 0 || k.KeyBits < dk.KeyBits) {
							dk.KeyBits = k.KeyBits
						}
					}
				}
			}
			switch {
			case !dk.OK:
				dk.Detail = "DKIM records not verified"
			case sh.Adapter == "ses":
				dk.Detail = fmt.Sprintf("%d Easy DKIM CNAMEs verified", len(dkimRecs))
			default:
				sel, _, _ := strings.Cut(strings.ToLower(dkimRecs[0].Name), "._domainkey.")
				dk.Detail = "selector " + sel + " verified"
			}
			out.DKIM = append(out.DKIM, dk)
		}
		if len(rpRecs) > 0 {
			mf := ProviderAuthFacts{Short: sh.Short, Domain: strings.ToLower(rpRecs[0].Name), OK: allVerified(rpRecs)}
			for _, m := range auth.MailFrom {
				if strings.EqualFold(m.Domain, mf.Domain) && !m.Aligned {
					mf.OK = false
				}
			}
			out.MailFrom = append(out.MailFrom, mf)
		}
	}
	return out, nil
}

func allVerified(rs []dnscheck.RecordCheck) bool {
	for _, r := range rs {
		if r.Status != dnscheck.RecordVerified {
			return false
		}
	}
	return true
}

// reputation reads the blocklist verdict and the lists behind it.
func (s *Service) reputation(ctx context.Context, g *gathered, f Facts) (ReputationFacts, error) {
	out := ReputationFacts{Status: Unknown, Via: "none", Listings: []ListingFacts{}}
	if s.Deliverability == nil {
		out.Reason = "Blocklist checks aren't available in this process."
		return out, nil
	}
	rep, err := s.Deliverability.Reputation(ctx, g.ws)
	if err != nil {
		return out, err
	}
	out.Status, out.Via, out.Sources, out.Reason, out.CheckedAt = rep.Status, rep.Via, rep.Sources, rep.Reason, rep.CheckedAt
	ov, err := s.Deliverability.Overview(ctx, g.ws)
	if err == nil {
		if ov.HostRepute.Unreachable != nil && rep.Status == Unknown {
			out.Outage = ov.HostRepute.Unreachable.Message
		}
		for _, src := range ov.Blocklists.Results {
			isDomain := slices.ContainsFunc(src.Assets, func(a string) bool { return !strings.ContainsAny(a, ":") && net4(a) == "" })
			isIP := slices.ContainsFunc(src.Assets, func(a string) bool { return net4(a) != "" })
			if isDomain && !slices.Contains(out.DomainSources, src.Source) {
				out.DomainSources = append(out.DomainSources, src.Source)
			}
			if isIP && !slices.Contains(out.IPSources, src.Source) {
				out.IPSources = append(out.IPSources, src.Source)
			}
		}
	}
	for _, l := range rep.Listings {
		typ := "domain"
		if net4(l.Asset) != "" || strings.Contains(l.Asset, ":") {
			typ = "ip"
		}
		lf := ListingFacts{Asset: l.Asset, Type: typ, Source: l.Source, Impact: l.Impact, DelistURL: l.DelistURL, Since: l.Since}
		for _, src := range ov.Blocklists.Results {
			if src.Source == l.Source && slices.Contains(src.ListedAssets, l.Asset) {
				lf.Detail = src.Detail
			}
		}
		out.Listings = append(out.Listings, lf)
	}
	return out, nil
}

var ipv4 = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)

func net4(s string) string {
	if ipv4.MatchString(s) {
		return s
	}
	return ""
}

// compliance gathers the one-click unsubscribe link and the complaint rate.
func (s *Service) compliance(ws *store.Workspace, f Facts) ComplianceFacts {
	out := ComplianceFacts{UnsubscribeURL: strings.ReplaceAll(s.URLs.Unsubscribe(ws.Slug, "TOKEN"), "TOKEN", "{token}")}
	out.HTTPS = strings.HasPrefix(out.UnsubscribeURL, "https://") || localURL(out.UnsubscribeURL)
	var weighted, total float64
	for _, sh := range f.Route.Shares {
		if sh.Complaint == nil || sh.Recipients == 0 {
			continue
		}
		weighted += *sh.Complaint * float64(sh.Recipients)
		total += float64(sh.Recipients)
	}
	if total > 0 {
		v := weighted / total
		out.ComplaintRate = &v
	}
	return out
}

// localURL reports a development instance (localhost): mailbox providers
// can't reach it, so its plain-HTTP links aren't worth a warning.
func localURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	h := u.Hostname()
	return h == "localhost" || h == "127.0.0.1" || h == "::1" || strings.HasSuffix(h, ".localhost")
}

// RoutingKey is the Broadcasts routing as it stands, part of a run's fingerprint.
func RoutingKey(ctx context.Context, q *store.Queries, wsID uuid.UUID) json.RawMessage {
	rows, err := q.RoutingListRules(ctx, wsID)
	if err != nil {
		return nil
	}
	type rule struct {
		Conditions json.RawMessage `json:"c"`
		Splits     json.RawMessage `json:"s"`
		Enabled    bool            `json:"e"`
	}
	var rules []rule
	for _, r := range rows {
		if r.Stream == "broadcast" {
			rules = append(rules, rule{r.Conditions, r.Splits, r.Enabled})
		}
	}
	b, _ := json.Marshal(rules)
	return b
}
