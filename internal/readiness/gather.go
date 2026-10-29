package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

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
	"github.com/hostrepute/reputecast/internal/sending/engine"
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

// URIBlocklists are the URI blocklists link domains are looked up on.
var URIBlocklists = contentcheck.URIListNames

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
	stageRules    stage = GroupRules
)

// runOrder is the order groups run (and tick in on the review screen).
var runOrder = []string{GroupAuthentication, GroupContent, GroupLanguages, GroupAccessibility, GroupConditional, GroupCompliance, GroupReputation, GroupLinks, GroupRules,
	GroupAudience, GroupRouting}

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
		SenderName: c.SenderName, SenderEmail: c.SenderEmail, SenderDomain: c.SenderDomain, ReplyTo: firstNonEmpty(c.ReplyTo, c.SenderReplyTo), StartAt: sched.StartAt(now), ScheduleMode: sched.Mode,
		HTMLOnly: HTMLOnly(c.Mjml, c.Html)}
	ab := parseABTest(c.AbTest)
	if ab != nil {
		f.Campaign.ABKind = ab.Kind
	}
	compiled := c.UpdatedAt
	if c.AutosavedAt != nil {
		compiled = *c.AutosavedAt
	}
	f.Campaign.CompiledAt = &compiled
	g := &gathered{ws: ws, c: c, byID: map[uuid.UUID]store.Provider{}}

	// Prepare: the audience and the route every group refers to. Company
	// domains are classified first (bounded), as the send will do.
	if s.Mailbox != nil {
		if _, err := s.Mailbox.ClassifyPending(ctx, ws.ID, 5*time.Second); err != nil && s.Log != nil {
			s.Log.WarnContext(ctx, "mailbox classification before Send Readiness", "campaign", c.ID, "err", err)
		}
	}
	def := ParseAudience(c.Audience)
	aud, recipients, err := s.computeAudience(ctx, s.Q, ws.ID, def, sunsetDays(ws.SunsetPolicy), now, true)
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
	// What makes the send unusual enough for a typed confirmation (SR-13):
	// an audience much larger than usual, a first send to a list, from the
	// sending domain or through a provider.
	if names, err := s.names(ctx, s.Q, ws.ID); err == nil {
		refs := []RouteShareRef{}
		for _, sh := range f.Route.Shares {
			if id, err := uuid.Parse(sh.ProviderID); err == nil && sh.Recipients > 0 {
				refs = append(refs, RouteShareRef{ID: id, Short: sh.Short})
			}
		}
		f.Confirm = s.confirmFacts(ctx, s.Q, ws.ID, c.ID, def, names, c.SenderDomain, refs)
	}
	if est := f.Route.Estimate; est != nil && est.Throttles != nil {
		// The throttles the send will use, as the engine reads them (the
		// defaults and the workspace's), not a copy of them.
		f.MailboxRates = est.Throttles
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
			if len(chk.URILists) > 0 {
				f.Content.URIBlocklists = chk.URILists
			}
			// A/B variants B and C are checked like the campaign (SR-7).
			if ab != nil {
				f.Content.Variants = s.variants(ctx, ws, c, ab, false)
			}
		} else if s.Log != nil {
			s.Log.WarnContext(ctx, "readiness content check failed", "campaign", c.ID, "err", err)
		}
	}
	// The size Gmail measures is the HTML as sent: links rewritten for
	// tracking and the open pixel added (SR-7).
	tracking := campaignTracking(c.Tracking)
	f.Content.TrackOpens, f.Content.TrackClicks = tracking.Opens, tracking.Clicks
	f.Content.TrackedBytes = trackedSize(c.Html, tracking, s.URLs, ws.Slug)
	s.trackingFacts(ctx, &f.Content)
	if s.Conditional != nil {
		f.Polls = s.Conditional.PollFacts(ctx, ws, c.ID, c.Mjml)
	}
	if s.Website != nil {
		f.Web = s.Website.WebFacts(ctx, ws, c.ID)
	}
	if s.Sponsor != nil && strings.TrimSpace(c.Mjml) != "" {
		f.Sponsor = s.Sponsor.SponsorFacts(ctx, ws, c.ID, c.Mjml)
	}
	if s.UpgradeOffers != nil && strings.TrimSpace(c.Mjml) != "" {
		f.UpgradeOffers = s.UpgradeOffers.EndedUpgradeOffers(ctx, ws, c.ID, c.Mjml)
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
				f.Content.Versions = append(f.Content.Versions, VersionReport{Language: v.Language, Report: chk.ContentCheckReport,
					TrackedBytes: trackedSize(v.HTML, tracking, s.URLs, ws.Slug)})
			}
		}
	}
	// The personalization tags of every version and variant, and whether
	// they have fallbacks.
	texts := []string{c.Subject, c.Preheader, c.Mjml, c.Html}
	for _, v := range f.Languages.Versions {
		texts = append(texts, v.Subject, v.Preheader, v.MJML, v.HTML)
	}
	for _, v := range ab.others() {
		texts = append(texts, v.Subject, v.Preheader, v.MJML, v.HTML)
	}
	f.Content.Fields = s.personalizationFields(ctx, ws.ID, texts...)
	done(stageLang, f)

	// Grow's blocks: who the Referral block shows to, what the
	// Recommendations block shows.
	if s.Grow != nil {
		byList := map[uuid.UUID]int64{}
		for _, r := range g.recipients {
			for _, l := range r.Lists {
				if id, err := uuid.Parse(l); err == nil {
					byList[id]++
				}
			}
		}
		f.Grow = s.Grow.ReadinessRows(ctx, ws, c.ID, c.Html, byList, int64(len(g.recipients)))
	}

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
	f.Compliance.Gmail = s.postmaster(ctx, ws.ID, firstNonEmpty(f.Auth.Domain, c.SenderDomain), now)
	f.Compliance.PostmasterConnected = f.Compliance.Gmail != nil || s.postmasterConnected(ctx, ws.ID, now)
	done(stageComply, f)

	if f.Reputation, err = s.reputation(ctx, g, f); err != nil {
		return f, fmt.Errorf("reputation: %w", err)
	}
	done(stageReput, f)

	// Links: the content check again, requesting every link (and image),
	// of the campaign, of every language version and of every content
	// variant (SR-7). The prober keeps answers for ten minutes, so a link
	// they share is requested once.
	if f.Content.Checked && s.Campaigns != nil {
		chk, err := s.Campaigns.Checks(ctx, ws, c.ID, campaigns.ComposerChecksRequest{Subject: c.Subject, Preheader: c.Preheader, MJML: c.Mjml, HTML: c.Html, ProbeLinks: true})
		if err == nil {
			f.Content.Report, f.Content.FooterRemovedAt = chk.ContentCheckReport, chk.FooterRemovedAt
			s.librarySizes(ctx, ws.ID, f.Content.Report.ImageList)
		}
		for i, v := range f.Languages.Versions {
			if i >= len(f.Content.Versions) || f.Content.Versions[i].Language != v.Language {
				break
			}
			if chk, err := s.Campaigns.Checks(ctx, ws, c.ID, campaigns.ComposerChecksRequest{Subject: v.Subject, Preheader: v.Preheader, MJML: v.MJML, HTML: v.HTML,
				Language: v.Language, ProbeLinks: true}); err == nil {
				f.Content.Versions[i].Report = chk.ContentCheckReport
			}
		}
		if ab != nil {
			f.Content.Variants = s.variants(ctx, ws, c, ab, true)
		}
	}
	done(stageLinks, f)
	// The workspace's rules (SR-10), on the links as fetched.
	f.Rules = s.rulesFor(ctx, ws.ID, "broadcast")
	f.Content.FooterText = footerText(c.Mjml, f.Content.Report.PlainText)
	done(stageRules, f)
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
	counts, cells, order, ok := planRoute(set, override, g.recipients, g.c.SenderDomain)
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
	// The 7-day rates and the volume behind them (a few messages can't
	// judge a provider, SR-5).
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
			Account: h.Account, Sandbox: h.Sandbox || p.Status == state.Sandbox, Quota: h.Quota, BulkAllowed: caps.BulkAllowed(), Terms: termsOf(p, caps), TermsKind: termsKind(p, caps),
			Tracking: state.TrackingOf(p), IPs: p.Ips, ConnectedAt: p.ConnectedAt, Bounce: u.Bounce7d, Complaint: u.Complaint7d,
			ProviderReported: u.ProviderReported, OneClick: caps.OneClick, Sent7d: u.Sent7d}
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
			sh.Down = downFacts(p, h)
			sh.DownDetail = downDetail(p, h, f.Now, loc)
		}
		if h.Rate != nil && h.Rate.Max > 0 {
			sh.Rate = h.Rate.Max
		}
		// The warm-up plan's day and cap when the send starts, counted as
		// the send loop counts them (warmup.AllowanceFor and DayOn: a paused
		// plan stays on its day, a slowdown holds the cap).
		if a, ok, err := warmup.AllowanceFor(ctx, s.Q, p.ID, f.Campaign.StartAt, loc); err != nil {
			return out, err
		} else if ok {
			sh.WarmupDay, sh.WarmupDays = a.Day, a.Days
			sh.WarmupCap = a.Cap
			sh.WarmupToday = sameDay(f.Campaign.StartAt, f.Now, loc)
			sh.Rate, sh.RateRamp = a.Rate, true
			if p.Status == state.Healthy {
				sh.Status = state.WarmingUp
			}
		}
		out.Shares = append(out.Shares, sh)
	}
	// The fallback must be able to take the send (C11).
	if fb := routeFallback(set, override); fb != nil {
		out.Fallback = s.vetFallback(ctx, g, *fb, f.Audience.Final, f.Now, loc)
	}
	// The estimate: the send played out with the engine's own rules (C4).
	if s.Estimator != nil && len(cells) > 0 {
		est, err := s.Estimator.Estimate(ctx, g.c.ID, engine.EstimateOptions{Start: f.Campaign.StartAt, Recipients: cells})
		switch {
		case err == nil:
			out.Estimate = &est
		case s.Log != nil:
			s.Log.WarnContext(ctx, "readiness estimate", "campaign", g.c.ID, "err", err)
		}
	}
	return out, nil
}

func sameDay(a, b time.Time, loc *time.Location) bool {
	x, y := a.In(loc), b.In(loc)
	return x.Year() == y.Year() && x.YearDay() == y.YearDay()
}

// termsKind is what a provider's terms say about bulk marketing, as a
// key of the terms.* messages: "yes, with production access".
func termsKind(p store.Provider, caps state.Snapshot) string {
	switch caps.Bulk {
	case providers.BulkAllowed:
		if p.Adapter == "ses" {
			return "productionAccess"
		}
		return "yes"
	case providers.BulkYourPolicy:
		return "ownPolicy"
	case providers.BulkBroadcastStreams:
		if caps.BroadcastStream {
			return "broadcastStream"
		}
		return "transactionalStream"
	}
	return "transactionalOnly"
}

// termsOf is the terms in English, as runs store them.
func termsOf(p store.Provider, caps state.Snapshot) string {
	return tx("terms."+termsKind(p, caps), nil).en()
}

// downFacts says why a provider can't take mail, and who: the provider
// for rejected credentials, its server for refused connections.
func downFacts(p store.Provider, h state.Health) *DownFacts {
	d := &DownFacts{Since: h.Down.Since, Who: p.Name}
	if h.Down.Reason == "auth" {
		d.Auth = true
		return d
	}
	var cfg struct {
		ServerURL string `json:"server_url"`
		Host      string `json:"host"`
	}
	_ = json.Unmarshal(p.Config, &cfg)
	if host := strings.TrimPrefix(strings.TrimPrefix(firstNonEmpty(cfg.ServerURL, cfg.Host), "https://"), "http://"); host != "" {
		d.Who = strings.TrimSuffix(host, "/")
	}
	if h.LastError != nil {
		d.Code = h.LastError.Code
	}
	return d
}

// downDetail is the outage in English, as runs store it.
func downDetail(p store.Provider, h state.Health, now time.Time, loc *time.Location) string {
	b := &builder{f: Facts{Now: now}, loc: loc}
	return b.downText(ShareFacts{Down: downFacts(p, h)}).en()
}

// auth reads the stored authentication check of the sending domain. A
// lookup that failed when Deliverability checked the domain (a DNS timeout,
// SERVFAIL) proves nothing about the record: the run looks the domain up
// again, and what still fails is Unknown, never a Block (SR-6).
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
	var checkedAt *time.Time
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
			checkedAt = &t
		case strings.EqualFold(chk.Asset, dnscheck.OrgDomain(domain)) && r.DMARC != nil:
			orgDMARC = r.DMARC
		}
	}
	if auth != nil && s.DNS != nil && LookupFailed(*auth) {
		fresh := Relookup(ctx, s.DNS, domain, *auth)
		auth = &fresh
		now := s.now()
		checkedAt = &now
	}
	if auth == nil {
		return out, nil
	}
	var aligned *float64
	if s.Deliverability != nil {
		if ov, err := s.Deliverability.Overview(ctx, g.ws); err == nil {
			for _, a := range ov.Assets {
				if strings.EqualFold(a.Asset, domain) && a.DMARCAligned != nil {
					v := *a.DMARCAligned
					aligned = &v
				}
			}
		}
	}
	out = AuthFactsOf(domain, *auth, orgDMARC, carrying(f.Route.Shares))
	out.CheckedAt, out.DMARCAligned = checkedAt, aligned
	return out, nil
}

// Relookup checks a sending domain again, with the records its providers
// require (those of the stored check), within 20 seconds.
func Relookup(ctx context.Context, r dnscheck.Resolver, domain string, stored dnscheck.DomainAuth) dnscheck.DomainAuth {
	reqs := make([]dnscheck.Requirement, 0, len(stored.Records))
	for _, rc := range stored.Records {
		reqs = append(reqs, rc.Requirement)
	}
	lctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return dnscheck.CheckDomain(lctx, r, domain, reqs)
}

// LookupFailed reports whether any lookup of a domain's authentication check
// failed (DNS didn't answer), as opposed to finding a record missing or wrong.
func LookupFailed(a dnscheck.DomainAuth) bool {
	if a.SPF.Status == dnscheck.StatusError || a.DMARC.Status == dnscheck.StatusError || a.BIMI.Status == dnscheck.StatusError {
		return true
	}
	for _, k := range a.DKIM {
		if k.Status == dnscheck.StatusError {
			return true
		}
	}
	for _, m := range a.MailFrom {
		if m.Status == dnscheck.StatusError || m.SPF == dnscheck.StatusError {
			return true
		}
	}
	for _, r := range a.Records {
		if r.Status == dnscheck.RecordError {
			return true
		}
	}
	return false
}

// AuthFactsOf turns a domain's authentication check into the facts of the
// providers carrying the send: their SPF, DKIM and MAIL FROM records. A
// record whose lookup failed is unknown, unless another record of the same
// kind is definitely wrong.
func AuthFactsOf(domain string, auth dnscheck.DomainAuth, orgDMARC *dnscheck.DMARC, shares []ShareFacts) AuthFacts {
	out := AuthFacts{Domain: domain, Checked: true, SPF: []ProviderAuthFacts{}, DKIM: []ProviderAuthFacts{}, MailFrom: []ProviderAuthFacts{}}
	out.SPFRecord = auth.SPF.Record
	if auth.SPF.Status == dnscheck.StatusError {
		out.SPFError = dnsTrouble(auth.SPF.Error)
	}
	dmarc := auth.DMARC
	if dmarc.Status != dnscheck.StatusPass && orgDMARC != nil && orgDMARC.Status == dnscheck.StatusPass {
		dmarc = *orgDMARC
	}
	out.DMARCStatus, out.DMARCPolicy, out.DMARCRecord = dmarc.Status, dmarc.Policy, dmarc.Record
	out.DMARCDomain = firstNonEmpty(dmarc.PolicyDomain, dmarc.Domain, domain)
	if dmarc.Status == dnscheck.StatusError {
		out.DMARCError = dnsTrouble(dmarc.Error)
	}
	out.BIMIStatus, out.BIMILogo = auth.BIMI.Status, auth.BIMI.Logo
	if auth.BIMI.Status == dnscheck.StatusError {
		out.BIMIError = dnsTrouble(auth.BIMI.Error)
	}
	if auth.BIMI.Status == dnscheck.StatusInvalid {
		out.BIMIProblem = auth.BIMI.Error
	}
	for _, sh := range shares {
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
		switch {
		case len(spfRecs) > 0:
			spf.OK = allVerified(spfRecs)
			if failed := lookupErrors(spfRecs); !spf.OK && failed != "" && !definitelyWrong(spfRecs) {
				spf.Unknown, spf.Trouble = true, failed
			} else if !spf.OK {
				spf.DetailKind, spf.Value, spf.Name = AuthSPFAdd, spfRecs[0].Value, spfRecs[0].Name
				spf.Detail = tx("auth.spfAdd", A{"value": spf.Value, "name": spf.Name}).en()
			}
		case len(rpRecs) > 0:
			for _, mf := range auth.MailFrom {
				if strings.EqualFold(mf.Domain, rpRecs[0].Name) {
					spf.OK = mf.SPF == dnscheck.StatusPass
					if mf.SPF == dnscheck.StatusError {
						spf.Unknown, spf.Trouble = true, dnsTrouble(mf.Error)
					}
				}
			}
		case auth.SPF.Status == dnscheck.StatusError:
			spf.Unknown, spf.Trouble = true, dnsTrouble(auth.SPF.Error)
		}
		out.SPF = append(out.SPF, spf)
		if len(dkimRecs) > 0 {
			dk := ProviderAuthFacts{Short: sh.Short, OK: allVerified(dkimRecs)}
			trouble, wrong := lookupErrors(dkimRecs), definitelyWrong(dkimRecs)
			for _, k := range auth.DKIM {
				for _, rc := range dkimRecs {
					if sel, _, ok := strings.Cut(strings.ToLower(rc.Name), "._domainkey."); ok && strings.EqualFold(k.Selector, sel) {
						switch k.Status {
						case dnscheck.StatusPass:
						case dnscheck.StatusError:
							dk.OK = false
							if trouble == "" || trouble == TroubleOther {
								trouble = dnsTrouble(k.Error)
							}
						default:
							dk.OK, wrong = false, true
						}
						if k.KeyBits > 0 && (dk.KeyBits == 0 || k.KeyBits < dk.KeyBits) {
							dk.KeyBits = k.KeyBits
						}
					}
				}
			}
			switch {
			case !dk.OK && trouble != "" && !wrong:
				dk.Unknown, dk.Trouble = true, trouble
			case !dk.OK:
				dk.DetailKind = AuthUnverified
				dk.Detail = tx("auth.dkimUnverified", nil).en()
			case sh.Adapter == "ses":
				dk.DetailKind, dk.CNAMEs = AuthCNAMEs, len(dkimRecs)
				dk.Detail = tx("auth.cnames", A{"count": dk.CNAMEs}).en()
			default:
				sel, _, _ := strings.Cut(strings.ToLower(dkimRecs[0].Name), "._domainkey.")
				dk.DetailKind, dk.Selector = AuthSelector, sel
				dk.Detail = tx("auth.selector", A{"selector": sel}).en()
			}
			out.DKIM = append(out.DKIM, dk)
		}
		if len(rpRecs) > 0 {
			mf := ProviderAuthFacts{Short: sh.Short, Domain: strings.ToLower(rpRecs[0].Name), OK: allVerified(rpRecs)}
			trouble, wrong := lookupErrors(rpRecs), definitelyWrong(rpRecs)
			for _, m := range auth.MailFrom {
				if !strings.EqualFold(m.Domain, mf.Domain) {
					continue
				}
				switch {
				case m.Status == dnscheck.StatusError:
					mf.OK = false
					if trouble == "" || trouble == TroubleOther {
						trouble = dnsTrouble(m.Error)
					}
				case !m.Aligned:
					mf.OK, wrong = false, true
				}
			}
			if !mf.OK && trouble != "" && !wrong {
				mf.Unknown, mf.Trouble = true, trouble
			}
			out.MailFrom = append(out.MailFrom, mf)
		}
	}
	return out
}

func allVerified(rs []dnscheck.RecordCheck) bool {
	for _, r := range rs {
		if r.Status != dnscheck.RecordVerified {
			return false
		}
	}
	return true
}

// lookupErrors is why the first record whose lookup failed couldn't be
// read ("" when every lookup answered).
func lookupErrors(rs []dnscheck.RecordCheck) string {
	for _, r := range rs {
		if r.Status == dnscheck.RecordError {
			return TroubleOther
		}
	}
	return ""
}

// definitelyWrong reports whether DNS answered that a record is missing or
// has another value.
func definitelyWrong(rs []dnscheck.RecordCheck) bool {
	for _, r := range rs {
		if r.Status == dnscheck.RecordMissing || r.Status == dnscheck.RecordWrongValue {
			return true
		}
	}
	return false
}

// Kinds of DNS trouble (ProviderAuthFacts.Trouble), so the evidence says
// what happened in each reader's language.
const (
	TroubleTimeout  = "timeout"  // no server answered in time
	TroubleServFail = "servfail" // the server answered SERVFAIL
	TroubleRefused  = "refused"  // the server refused the query
	TroubleOther    = "other"    // no usable answer
)

// dnsTrouble names what went wrong with a lookup from its error.
func dnsTrouble(err string) string {
	e := strings.ToLower(err)
	switch {
	case strings.Contains(e, "timeout") || strings.Contains(e, "deadline") || strings.Contains(e, "timed out"):
		return TroubleTimeout
	case strings.Contains(e, "servfail"):
		return TroubleServFail
	case strings.Contains(e, "refused"):
		return TroubleRefused
	}
	return TroubleOther
}

// reputation reads the blocklist verdict and the lists behind it.
func (s *Service) reputation(ctx context.Context, g *gathered, f Facts) (ReputationFacts, error) {
	out := ReputationFacts{Status: Unknown, Via: "none", Listings: []ListingFacts{}}
	if s.Deliverability == nil {
		out.ReasonText = tx("reputation.unavailable", nil)
		out.Reason = out.ReasonText.en()
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
		if rep.Status != Unknown {
			checked := checkedAssets(ov)
			for _, sh := range carrying(f.Route.Shares) {
				for _, ip := range sh.IPs {
					if !checked[strings.ToLower(ip)] && !slices.Contains(out.UncheckedIPs, ip) {
						out.UncheckedIPs = append(out.UncheckedIPs, ip)
					}
				}
			}
		}
	}
	out.Domains = s.reputationDomains(ctx, g, f, checkedAssets(ov))
	if err == nil {
		out.ReverseDNS = s.reverseDNS(ctx, f, &ov)
	} else {
		out.ReverseDNS = s.reverseDNS(ctx, f, nil)
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

// checkedAssets are the assets Deliverability's blocklist checks cover.
func checkedAssets(ov deliverability.DeliverabilityOverview) map[string]bool {
	out := map[string]bool{}
	for _, src := range ov.Blocklists.Results {
		for _, a := range src.Assets {
			out[strings.ToLower(a)] = true
		}
	}
	return out
}

// reputationDomains are the domains the send's reputation rides on (SR-2):
// the sending domain, its parent (organizational) domain, the return-path
// domains of the providers carrying the send and the tracking domain. Those
// Deliverability's blocklist checks cover come from there; every other one
// is looked up live on the URI blocklists, as link domains are, so a domain
// HostRepute hasn't checked yet (often the parent) is never taken for
// clean (R1.L2, F-3). Without live lookups it is unknown, with that reason.
func (s *Service) reputationDomains(ctx context.Context, g *gathered, f Facts, watched map[string]bool) []DomainAssetFacts {
	sending := strings.ToLower(firstNonEmpty(f.Auth.Domain, g.c.SenderDomain))
	if sending == "" {
		return nil
	}
	out := []DomainAssetFacts{}
	index := map[string]int{}
	add := func(d DomainAssetFacts) {
		if i, ok := index[d.Domain]; ok {
			for _, p := range d.Providers {
				if !slices.Contains(out[i].Providers, p) {
					out[i].Providers = append(out[i].Providers, p)
				}
			}
			return
		}
		index[d.Domain] = len(out)
		out = append(out, d)
	}
	add(DomainAssetFacts{Domain: sending, Role: DomainSending})
	if org := dnscheck.OrgDomain(sending); org != sending {
		add(DomainAssetFacts{Domain: org, Role: DomainParent})
	}
	for _, mf := range f.Auth.MailFrom {
		if d := strings.ToLower(mf.Domain); d != "" {
			add(DomainAssetFacts{Domain: d, Role: DomainReturnPath, Providers: []string{mf.Short}})
		}
	}
	if host := trackingHost(s.URLs.Base()); host != "" {
		add(DomainAssetFacts{Domain: host, Role: DomainTracking})
	}
	live := []string{}
	for i, d := range out {
		if watched[d.Domain] {
			continue
		}
		out[i].Live = true
		if s.Links == nil {
			out[i].Unknown = UnknownUnchecked
			continue
		}
		live = append(live, d.Domain)
	}
	if len(live) == 0 {
		return out
	}
	rep, err := s.Links.Check(ctx, live)
	for i, d := range out {
		if !d.Live || d.Unknown != "" {
			continue
		}
		switch {
		case err != nil:
			out[i].Unknown = contentcheck.TroubleOther
		case len(rep.Listed[d.Domain]) > 0:
			out[i].Lists = rep.Listed[d.Domain]
		case rep.Unknown[d.Domain] != "":
			out[i].Unknown = rep.Unknown[d.Domain]
		}
	}
	return out
}

// trackingHost is the domain of the open pixel and the tracked links: the
// public URL's host, unless it's an address or a local name (development).
func trackingHost(base string) string {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return ""
	}
	h := strings.ToLower(u.Hostname())
	if h == "" || h == "localhost" || strings.HasSuffix(h, ".localhost") || !strings.Contains(h, ".") || net4(h) != "" || strings.Contains(h, ":") {
		return ""
	}
	return h
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
	out := ComplianceFacts{UnsubscribeURL: strings.ReplaceAll(s.URLs.Unsubscribe(ws.Slug, "TOKEN"), "TOKEN", "{token}"),
		Mailto: firstNonEmpty(f.Campaign.ReplyTo, f.Campaign.SenderEmail)}
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
