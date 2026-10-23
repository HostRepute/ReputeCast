package readiness

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/providers/state"
)

// groupTitle is the heading of a check group ("Audience quality").
func groupTitle(g string) Text { return tx("group."+g+".title", nil) }

// groupPrefix names a group before a check's title when checks are listed
// by severity ("Links: go.acme-promo.link is listed…").
func groupPrefix(g string) Text { return tx("group."+g+".prefix", nil) }

// GroupTitle is the heading of a check group ("Audience quality").
func GroupTitle(g string) string { return groupTitle(g).en() }

// DefaultMailboxRates are the per-mailbox-provider throttles: Outlook, Yahoo
// and iCloud accept slower streams more reliably; Gmail and the rest take
// each provider's full rate. The sending engine adapts them when deferrals rise.
var DefaultMailboxRates = map[string]float64{"gmail": 0, "outlook": 5, "yahoo": 4, "icloud": 3, "other": 0}

// MailboxOrder is the chart order of mailbox providers.
var MailboxOrder = []string{"gmail", "outlook", "yahoo", "icloud", "other"}

// MailboxName is the display name of a mailbox provider key.
func MailboxName(k string) string { return mailboxName(k).en() }

// mailboxName names a mailbox provider; "Other" is a word, the rest names.
func mailboxName(k string) Text {
	switch k {
	case "gmail":
		return RawText("Gmail")
	case "outlook":
		return RawText("Outlook")
	case "yahoo":
		return RawText("Yahoo")
	case "icloud":
		return RawText("iCloud")
	}
	return tx("mailbox.other", nil)
}

// UnengagedWarnShare and UnengagedWarnCount decide when unengaged contacts
// are worth a warning: 1% of the audience or 500 contacts.
const (
	UnengagedWarnShare = 1.0
	UnengagedWarnCount = 500
)

type builder struct {
	f      Facts
	loc    *time.Location
	checks []check
}

// Assemble builds the checklist and the estimates from what a run gathered.
func Assemble(f Facts) ReadinessResult {
	if f.Location == nil {
		f.Location = time.UTC
	}
	if f.MailboxRates == nil {
		f.MailboxRates = DefaultMailboxRates
	}
	b := &builder{f: f, loc: f.Location}
	b.reputation()
	b.authentication()
	b.content()
	b.polls()
	b.sponsor()
	b.upgradeOffers()
	b.web()
	b.languages()
	b.accessibility()
	b.links()
	b.conditional()
	b.compliance()
	b.audience()
	b.paidAudience()
	b.verification()
	b.routing()
	b.everyVersion()
	b.stores()
	for i := range b.checks {
		c := &b.checks[i]
		if c.FlatTitle.Empty() {
			c.FlatTitle = tx("check.flat", A{"group": groupPrefix(c.Group), "title": c.Title})
		}
		if c.Evidence.Empty() {
			c.Evidence = c.Title
			if !c.Detail.Empty() && !c.Mono {
				c.Evidence = sentences(c.Title, c.Detail)
			}
		}
	}
	out := ReadinessResult{Checks: make([]ReadinessCheck, 0, len(b.checks)), Estimates: b.estimates()}
	for _, c := range b.checks {
		out.Checks = append(out.Checks, c.result())
		out.Counts.Total++
		switch c.Status {
		case Pass:
			out.Counts.Passed++
		case Warn:
			out.Counts.Warnings++
		case Block:
			out.Counts.Blocks++
		case Unknown:
			out.Counts.Unknown++
		}
	}
	switch {
	case out.Counts.Blocks > 0:
		out.Verdict = VerdictBlocked
	case out.Counts.Warnings > 0:
		out.Verdict = VerdictWarnings
	default:
		out.Verdict = VerdictReady
	}
	if out.Counts.Unknown > 0 {
		out.Incomplete = true
		switch {
		case strings.TrimSpace(f.Reputation.Outage) != "":
			out.problem = RawText(f.Reputation.Outage)
		case strings.TrimSpace(f.Reputation.Reason) != "":
			out.problem = unknownReason(f.Reputation)
		default:
			out.problem = tx("problem.someChecks", nil)
		}
		out.Problem = out.problem.en()
	}
	out.Groups = b.groups()
	return out
}

func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, ".") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "\"") || strings.HasSuffix(s, "”") {
		return s
	}
	return s + "."
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (b *builder) add(c check) { b.checks = append(b.checks, c) }

// when is a time of the run in the workspace zone: "09:40" today, "Mon 12
// Oct, 09:40" on another day.
func (b *builder) when(t time.Time) map[string]any { return clockArg(t, b.f.Now, b.loc) }

// day is a day in the workspace zone: "Mon 12 Oct".
func (b *builder) day(t time.Time) map[string]any { return ArgDay(t.In(b.loc)) }

// href is an app path inside the workspace.
func (b *builder) href(path string) string { return "/w/" + b.f.Workspace + path }

// editor deep-links into the composer, selecting an element when path is set.
func (b *builder) editor(path []int, panel string) string {
	u := b.href("/campaigns/" + b.f.Campaign.ID + "/edit")
	q := url.Values{}
	if len(path) > 0 {
		parts := make([]string, len(path))
		for i, p := range path {
			parts[i] = fmt.Sprint(p)
		}
		q.Set("select", strings.Join(parts, "-"))
	}
	if panel != "" {
		q.Set("panel", panel)
	}
	if len(q) > 0 {
		return u + "?" + q.Encode()
	}
	return u
}

// shares returns the route's providers that carry part of this send.
func (b *builder) shares() []ShareFacts {
	return carrying(b.f.Route.Shares)
}

// carrying keeps the providers that carry part of the send; all of them
// when nobody is routed yet (an empty audience).
func carrying(shares []ShareFacts) []ShareFacts {
	out := []ShareFacts{}
	for _, s := range shares {
		if s.Recipients > 0 {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return shares
	}
	return out
}

func (b *builder) availableShares() []ShareFacts {
	out := []ShareFacts{}
	for _, s := range b.shares() {
		if s.Available {
			out = append(out, s)
		}
	}
	return out
}

func shortsOf(shares []ShareFacts) []string {
	out := make([]string, 0, len(shares))
	for _, s := range shares {
		out = append(out, s.Short)
	}
	return out
}

// otherAvailable is the largest available share other than id.
func (b *builder) otherAvailable(id string) *ShareFacts {
	var best *ShareFacts
	for _, s := range b.f.Route.Shares {
		if s.ProviderID == id || !s.Available {
			continue
		}
		if best == nil || s.Recipients > best.Recipients {
			c := s
			best = &c
		}
	}
	return best
}

// routeOnlyFix moves the whole send to another provider.
func routeOnlyFix(o *ShareFacts, label Text, tone string) fix {
	return apply(FixRouteOnly, label, tone, map[string]string{"providerId": o.ProviderID, "provider": o.Short})
}

func (b *builder) reputation() {
	r := b.f.Reputation
	domain := firstNonEmpty(b.f.Auth.Domain, b.f.Campaign.SenderDomain)
	via := "hostrepute"
	if r.Via == "builtin" {
		via = "builtin"
	}
	var checked any
	if r.CheckedAt != nil {
		checked = b.when(*r.CheckedAt)
	}
	unknownNote := tx("note.notChecked", nil)
	if r.Outage != "" {
		unknownNote = tx("note.timedOut", nil)
	}
	deliverability := link(tx("fix.openDeliverability", nil), b.href("/deliverability"))
	if domain != "" {
		key := "reputation.domain_blocklists"
		label := tx("label.domainBlocklists", nil)
		var listed []ListingFacts
		for _, l := range r.Listings {
			if l.Type == "domain" && strings.EqualFold(l.Asset, domain) {
				listed = append(listed, l)
			}
		}
		switch {
		case r.Status == Unknown:
			b.add(check{Key: key, Group: GroupReputation, Status: Unknown,
				Title:     tx("reputation.domainUnknown", A{"domain": domain}),
				FlatTitle: tx("reputation.domainUnknownFlat", A{"sources": r.Sources, "via": via}),
				Detail:    unknownReason(r), Note: unknownNote, Label: label,
				Fixes: []fix{deliverability}})
		case len(listed) == 0:
			b.add(check{Key: key, Group: GroupReputation, Status: Pass,
				Title:  tx("reputation.domainClean", A{"domain": domain, "sources": r.Sources}),
				Detail: b.sourcesDetail("domain", checked), Label: label,
				Short: tx("reputation.domainCleanShort", A{"domain": domain, "sources": r.Sources})})
		default:
			status := Warn
			names := []string{}
			for _, l := range listed {
				if l.Impact == "high" {
					status = Block
				}
				names = append(names, l.Source)
			}
			b.add(check{Key: key, Group: GroupReputation, Status: status,
				Title:  tx("reputation.domainListed", A{"domain": domain, "count": len(listed), "sources": r.Sources}),
				Detail: sentences(listingText(listed[0]), impactText(listed[0].Impact, "domain")), Label: label,
				Ack:   tx("reputation.domainListedAck", A{"domain": domain, "lists": ArgAnd(names)}),
				Fixes: []fix{link(tx("fix.requestDelisting", nil), b.href("/deliverability"))}})
		}
	}
	// Dedicated IPs of the providers carrying the send.
	shared := []string{}
	for _, s := range b.availableShares() {
		if len(s.IPs) == 0 {
			shared = append(shared, s.Short)
		}
	}
	for _, s := range b.availableShares() {
		for _, ip := range s.IPs {
			key := "reputation.ip_blocklists." + ip
			label := tx("label.ipBlocklists", nil)
			var listed []ListingFacts
			for _, l := range r.Listings {
				if l.Type == "ip" && l.Asset == ip {
					listed = append(listed, l)
				}
			}
			switch {
			case r.Status == Unknown:
				b.add(check{Key: key, Group: GroupReputation, Status: Unknown,
					Title:     tx("reputation.ipUnknown", A{"ip": ip}),
					FlatTitle: tx("reputation.ipUnknownFlat", A{"ip": ip}),
					Detail:    unknownReason(r), Note: unknownNote, Label: label})
			case len(listed) == 0:
				b.add(check{Key: key, Group: GroupReputation, Status: Pass,
					Title:  tx("reputation.ipClean", A{"provider": s.Name, "ip": ip, "sources": r.Sources}),
					Detail: b.sourcesDetail("ip", checked), Label: label,
					Short: tx("reputation.ipCleanShort", A{"ip": ip, "sources": r.Sources})})
			default:
				status := Warn
				for _, l := range listed {
					if l.Impact == "high" {
						status = Block
					}
				}
				detail := sentences(listingText(listed[0]), impactText(listed[0].Impact, "ip"))
				if len(shared) > 0 {
					detail = sentences(detail, tx("reputation.sharedPool", A{"providers": ArgAnd(shared)}))
				}
				fixes := []fix{link(tx("fix.requestDelisting", nil), b.href("/deliverability"))}
				if o := b.otherAvailable(s.ProviderID); o != nil {
					fixes = append(fixes, routeOnlyFix(o, tx("fix.sendViaOnly", A{"provider": o.Short}), ""))
				}
				b.add(check{Key: key, Group: GroupReputation, Status: status,
					Title:  tx("reputation.ipListed", A{"provider": s.Name, "ip": ip, "count": len(listed), "sources": r.Sources}),
					Detail: detail, Label: label,
					Ack:   tx("reputation.ipListedAck", A{"provider": s.Name, "list": shortSource(listed[0].Source), "impact": listed[0].Impact}),
					Fixes: fixes})
			}
		}
	}
	// Bounce and complaint rates of each provider against its own limits.
	for _, s := range b.availableShares() {
		b.add(b.providerRates(s))
	}
}

// unknownReason says why the blocklists couldn't be checked: the reason
// Deliverability gave, or HostRepute's error.
func unknownReason(r ReputationFacts) Text {
	if strings.TrimSpace(r.Reason) != "" {
		if !r.ReasonText.Empty() {
			return r.ReasonText
		}
		return RawText(r.Reason)
	}
	return RawText(r.Outage)
}

// sourcesDetail is the lists a check looked at and when: "Spamhaus DBL,
// SURBL, URIBL, Barracuda and 80 more · checked 09:40".
func (b *builder) sourcesDetail(kind string, checked any) Text {
	line := b.sourcesLine(kind)
	switch {
	case line.Empty() && checked == nil:
		return nil
	case line.Empty():
		return tx("reputation.checkedAt", A{"when": checked})
	case checked == nil:
		return line
	}
	return tx("reputation.sourcesChecked", A{"sources": line, "when": checked})
}

// sourcesLine names the first lists of a kind: "Spamhaus DBL, SURBL, URIBL,
// Barracuda and 80 more".
func (b *builder) sourcesLine(kind string) Text {
	r := b.f.Reputation
	names := r.DomainSources
	if kind == "ip" {
		names = r.IPSources
	}
	if len(names) == 0 {
		if r.Sources > 0 {
			return tx("reputation.sourcesCount", A{"count": r.Sources})
		}
		return nil
	}
	n := min(4, len(names))
	if rest := r.Sources - n; rest > 0 {
		return tx("reputation.sourcesMore", A{"names": argJoin(", ", names[:n]), "more": rest})
	}
	return tx("reputation.sources", A{"names": argJoin(", ", names[:n])})
}

var (
	asnRe  = regexp.MustCompile(`\((AS\d+)\)`)
	cidrRe = regexp.MustCompile(`(/\d{1,2})\b`)
)

// listingText says what a blocklist lists.
func listingText(l ListingFacts) Text {
	d := strings.TrimSpace(l.Detail)
	switch {
	case cidrRe.MatchString(d):
		return tx("listing.range", A{"source": l.Source, "range": cidrRe.FindStringSubmatch(d)[1]})
	case strings.Contains(strings.ToLower(d), "network-range"):
		if m := asnRe.FindStringSubmatch(d); m != nil {
			return tx("listing.networkAsn", A{"source": l.Source, "asn": m[1]})
		}
		return tx("listing.network", A{"source": l.Source})
	case d != "":
		// The list's own words.
		return tx("listing.detail", A{"source": l.Source, "detail": sentence(d)})
	}
	return tx("listing.on", A{"source": l.Source})
}

func impactText(impact, kind string) Text {
	switch impact {
	case "high":
		if kind == "domain" {
			return tx("impact.highDomain", nil)
		}
		return tx("impact.highIp", nil)
	case "medium":
		return tx("impact.medium", nil)
	}
	return tx("impact.low", nil)
}

// shortSource abbreviates "UCEPROTECT Level 3" to "UCEPROTECT L3".
func shortSource(s string) string { return strings.Replace(s, " Level ", " L", 1) }

// limitsText names whose limits a provider's rates are measured against:
// "SES review thresholds", "the warm-up guard", "Postmark's limits".
func limitsText(s ShareFacts) Text {
	l := s.Limits
	switch {
	case l.Bounce.Review > 0 && l.Bounce.Label != "":
		return tx("limits.review", A{"name": l.Bounce.Label})
	case l.Bounce.Label == "warm-up guard":
		return tx("limits.guard", nil)
	case l.Bounce.Label != "":
		return tx("limits.named", A{"name": l.Bounce.Label})
	}
	return tx("limits.provider", nil)
}

// statusText is a provider status: "Healthy", "Warming up".
func statusText(status string) Text {
	switch status {
	case state.Healthy, "":
		return tx("status.healthy", nil)
	case state.WarmingUp:
		return tx("status.warmingUp", nil)
	case state.Degraded:
		return tx("status.degraded", nil)
	case state.Down:
		return tx("status.down", nil)
	case state.Sandbox:
		return tx("status.sandbox", nil)
	case state.Paused:
		return tx("status.paused", nil)
	}
	return RawText(capitalize(strings.ReplaceAll(status, "_", " ")))
}

func (b *builder) providerRates(s ShareFacts) check {
	key := "reputation.provider." + s.ProviderID
	c := check{Key: key, Group: GroupReputation, Status: Pass, Label: tx("rates.label", A{"provider": s.Short})}
	l := s.Limits
	var detail Text
	if s.WarmupDays > 0 {
		detail = tx("rates.warmup", A{"day": s.WarmupDay, "days": s.WarmupDays, "today": yes(s.WarmupToday), "cap": s.WarmupCap, "count": s.Recipients})
	} else {
		window := tx("rates.windowTrailing", nil)
		if s.ProviderReported {
			window = tx("rates.windowReported", A{"provider": s.Name})
		}
		detail = tx("rates.account", A{"status": statusText(s.Status), "unenforced": yes(strings.Contains(strings.ToLower(s.Account), "no enforcement")), "window": window})
	}
	c.Detail = detail
	deliverability := link(tx("fix.openDeliverability", nil), b.href("/deliverability"))
	if s.Bounce == nil && s.Complaint == nil {
		c.Title = tx("rates.noMail", A{"provider": s.Name})
		c.Detail = tx("rates.noMailDetail", A{"detail": detail})
		c.Short = tx("rates.noMailShort", A{"provider": s.Short})
		return c
	}
	bounce, complaint := 0.0, 0.0
	if s.Bounce != nil {
		bounce = *s.Bounce
	}
	if s.Complaint != nil {
		complaint = *s.Complaint
	}
	limits := A{"limits": limitsText(s), "bounce": ArgPercent(l.Bounce.Limit), "complaint": ArgPercent(l.Complaint.Limit)}
	phrase := tx("rates.inside", limits)
	switch {
	case (l.Bounce.Pause > 0 && bounce >= l.Bounce.Pause) || (l.Complaint.Pause > 0 && complaint >= l.Complaint.Pause):
		c.Status = Block
		var whose any = limitsText(s)
		if l.Bounce.Review > 0 && l.Bounce.Label != "" {
			whose = l.Bounce.Label
		}
		phrase = tx("rates.pause", A{"limits": whose, "bounce": ArgPercent(l.Bounce.Pause), "complaint": ArgPercent(l.Complaint.Pause)})
		c.Fixes = []fix{deliverability}
	case bounce >= l.Bounce.Limit || complaint >= l.Complaint.Limit:
		c.Status = Warn
		phrase = tx("rates.over", limits)
		c.Fixes = []fix{deliverability}
	case bounce >= 0.8*l.Bounce.Limit || complaint >= 0.8*l.Complaint.Limit:
		c.Status = Warn
		phrase = tx("rates.close", limits)
		c.Fixes = []fix{deliverability}
	}
	c.Title = tx("rates.title", A{"provider": s.Name, "bounce": ArgPercent(bounce), "complaint": ArgPercent(complaint), "phrase": phrase})
	c.Short = tx("rates.short", A{"provider": s.Short, "bounce": ArgPercent(bounce), "complaint": ArgPercent(complaint)})
	if c.Status == Warn {
		c.Ack = tx("rates.ack", A{"provider": s.Name})
	}
	// A warming provider sends at most its day's cap.
	if c.Status == Pass && s.WarmupCap > 0 && s.Recipients > s.WarmupCap {
		c.Status = Warn
		c.Title = tx("rates.cap", A{"provider": s.Name, "cap": s.WarmupCap, "count": s.Recipients})
		c.Detail = tx("rates.capDetail", A{"day": s.WarmupDay, "days": s.WarmupDays})
		c.Ack = tx("rates.capAck", A{"provider": s.Name})
		if o := b.otherAvailable(s.ProviderID); o != nil {
			c.Fixes = []fix{routeOnlyFix(o, tx("fix.sendViaOnly", A{"provider": o.Short}), "")}
		}
	}
	return c
}

// yes is a select argument: "yes" or "no".
func yes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func (b *builder) authentication() {
	a := b.f.Auth
	domain := firstNonEmpty(a.Domain, b.f.Campaign.SenderDomain)
	if domain == "" {
		b.add(check{Key: "auth.sender", Group: GroupAuthentication, Status: Block,
			Title: tx("auth.noSender", nil), Detail: tx("auth.noSenderDetail", nil),
			Fixes: []fix{link(tx("fix.openInEditor", nil), b.editor(nil, ""))}})
		return
	}
	if !a.Checked {
		b.add(check{Key: "auth.unchecked", Group: GroupAuthentication, Status: Warn,
			Title:  tx("auth.unchecked", A{"domain": domain}),
			Detail: tx("auth.uncheckedDetail", nil),
			Ack:    tx("auth.uncheckedAck", A{"domain": domain}),
			Fixes:  []fix{link(tx("fix.openDeliverability", nil), b.href("/deliverability"))}})
		return
	}
	senders := link(tx("fix.openSenders", nil), b.href("/settings/senders"))
	if bad := failing(a.SPF); len(bad) == 0 && len(a.SPF) > 0 {
		names := shorts(a.SPF)
		b.add(check{Key: "auth.spf", Group: GroupAuthentication, Status: Pass,
			Title: tx("auth.spfPass", A{"count": len(names), "provider": first(names)}), Detail: RawText(a.SPFRecord), Mono: a.SPFRecord != "", Label: RawText("SPF"), Short: RawText("SPF")})
	} else if len(a.SPF) == 0 && a.SPFRecord == "" {
		b.add(check{Key: "auth.spf", Group: GroupAuthentication, Status: Block, Title: tx("auth.spfMissing", A{"domain": domain}),
			Detail: tx("auth.spfMissingDetail", nil), Label: RawText("SPF"), Fixes: []fix{senders}})
	} else {
		var detail Text
		fail := firstFailText(a.SPF)
		switch {
		case a.SPFRecord != "" && !fail.Empty():
			detail = tx("auth.spfFailDetail", A{"record": a.SPFRecord, "fix": fail})
		case a.SPFRecord != "":
			detail = RawText(a.SPFRecord)
		default:
			detail = fail
		}
		b.add(check{Key: "auth.spf", Group: GroupAuthentication, Status: Block, Title: tx("auth.spfFails", A{"providers": ArgAnd(bad)}),
			Detail: detail, Label: RawText("SPF"), Fixes: []fix{senders}})
	}
	if bad := failing(a.DKIM); len(bad) == 0 && len(a.DKIM) > 0 {
		bits := 0
		parts := []Text{}
		for _, d := range a.DKIM {
			if d.KeyBits > 0 && (bits == 0 || d.KeyBits < bits) {
				bits = d.KeyBits
			}
			parts = append(parts, authDetail(d))
		}
		names := shorts(a.DKIM)
		title := tx("auth.dkimPass", A{"count": len(names), "provider": first(names), "keys": yes(bits > 0), "bits": fmt.Sprint(bits)})
		status := Pass
		if bits > 0 && bits < 1024 {
			status = Warn
			title = tx("auth.dkimWeak", A{"bits": fmt.Sprint(bits)})
		}
		c := check{Key: "auth.dkim", Group: GroupAuthentication, Status: status, Title: title, Detail: joined(" · ", parts), Label: RawText("DKIM"), Short: RawText("DKIM")}
		if status == Warn {
			c.Ack = tx("auth.dkimWeakAck", nil)
			c.Fixes = []fix{senders}
		}
		b.add(c)
	} else {
		var providers any = ArgAnd(bad)
		switch {
		case len(bad) > 0:
		case len(a.DKIM) > 0:
			providers = ArgAnd(shorts(a.DKIM))
		default:
			providers = tx("auth.theSendingProviders", nil)
		}
		detail := firstFailText(a.DKIM)
		if detail.Empty() {
			detail = tx("auth.dkimMissingDetail", nil)
		}
		b.add(check{Key: "auth.dkim", Group: GroupAuthentication, Status: Block, Title: tx("auth.dkimFails", A{"providers": providers}),
			Detail: detail, Label: RawText("DKIM"), Fixes: []fix{senders}})
	}
	switch a.DMARCStatus {
	case "pass":
		policy := firstNonEmpty(a.DMARCPolicy, "none")
		title := tx("auth.dmarcPass", A{"policy": policy, "measured": "no"})
		status := Pass
		if a.DMARCAligned != nil {
			aligned := ArgPercent(math.Round(*a.DMARCAligned*10) / 10)
			title = tx("auth.dmarcPass", A{"policy": policy, "measured": "yes", "aligned": aligned})
			if *a.DMARCAligned < 90 {
				status = Warn
				title = tx("auth.dmarcLow", A{"aligned": aligned})
			}
		}
		c := check{Key: "auth.dmarc", Group: GroupAuthentication, Status: status, Title: title, Detail: RawText(a.DMARCRecord), Mono: a.DMARCRecord != "", Label: RawText("DMARC"),
			Short: RawText("DMARC p=" + policy)}
		if status == Warn {
			c.Ack = tx("auth.dmarcLowAck", nil)
			c.Fixes = []fix{link(tx("fix.openDeliverability", nil), b.href("/deliverability"))}
		}
		b.add(c)
	case "missing", "":
		b.add(check{Key: "auth.dmarc", Group: GroupAuthentication, Status: Block, Title: tx("auth.dmarcMissing", A{"domain": domain}),
			Detail: tx("auth.dmarcMissingDetail", nil), Label: RawText("DMARC"), Fixes: []fix{senders}})
	default:
		b.add(check{Key: "auth.dmarc", Group: GroupAuthentication, Status: Block, Title: tx("auth.dmarcInvalid", A{"domain": domain}),
			Detail: RawText(a.DMARCRecord), Mono: a.DMARCRecord != "", Label: RawText("DMARC"), Fixes: []fix{senders}})
	}
	if bad := failing(a.MailFrom); len(bad) == 0 && len(a.MailFrom) > 0 {
		parts := []string{}
		for _, m := range a.MailFrom {
			parts = append(parts, m.Domain+" ("+m.Short+")")
		}
		b.add(check{Key: "auth.mail_from", Group: GroupAuthentication, Status: Pass, Title: tx("auth.mailFromPass", A{"count": len(a.MailFrom)}),
			Detail: RawText(strings.Join(parts, " · ")), Label: RawText("MAIL FROM"), Short: tx("auth.mailFromShort", nil)})
	} else if len(a.MailFrom) > 0 {
		b.add(check{Key: "auth.mail_from", Group: GroupAuthentication, Status: Warn,
			Title:  tx("auth.mailFromDefault", A{"providers": ArgAnd(bad), "count": len(bad), "domain": domain}),
			Detail: tx("auth.mailFromDefaultDetail", nil), Label: RawText("MAIL FROM"),
			Ack: tx("auth.mailFromDefaultAck", A{"providers": ArgAnd(bad)}), Fixes: []fix{senders}})
	}
	if a.BIMIStatus == "pass" {
		detail := RawText(a.BIMILogo)
		if detail.Empty() {
			detail = tx("auth.bimiShows", nil)
		}
		b.add(check{Key: "auth.bimi", Group: GroupAuthentication, Status: Pass, Title: tx("auth.bimiFound", nil), Detail: detail, Label: RawText("BIMI"), Short: RawText("BIMI")})
	} else {
		b.add(check{Key: "auth.bimi", Group: GroupAuthentication, Status: Pass, Title: tx("auth.bimiNotRequired", nil),
			Detail: tx("auth.bimiOptional", nil), Label: RawText("BIMI"),
			Fixes: []fix{link(tx("fix.setUpBimi", nil), b.href("/deliverability"))}})
	}
}

func first(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func failing(list []ProviderAuthFacts) []string {
	out := []string{}
	for _, p := range list {
		if !p.OK {
			out = append(out, p.Short)
		}
	}
	return out
}

func shorts(list []ProviderAuthFacts) []string {
	out := []string{}
	for _, p := range list {
		out = append(out, p.Short)
	}
	return out
}

// authDetail is what a provider's record says: "SES: 3 Easy DKIM CNAMEs
// verified".
func authDetail(p ProviderAuthFacts) Text {
	var d any = p.Detail
	switch p.DetailKind {
	case AuthCNAMEs:
		d = tx("auth.cnames", A{"count": p.CNAMEs})
	case AuthSelector:
		d = tx("auth.selector", A{"selector": p.Selector})
	case AuthUnverified:
		d = tx("auth.dkimUnverified", nil)
	case AuthSPFAdd:
		d = tx("auth.spfAdd", A{"value": p.Value, "name": p.Name})
	}
	return tx("auth.detail", A{"provider": p.Short, "detail": d})
}

// firstFailText is what's wrong with the first failing provider, as a
// sentence.
func firstFailText(list []ProviderAuthFacts) Text {
	for _, p := range list {
		if !p.OK && (p.Detail != "" || p.DetailKind != "") {
			return asSentence(authDetail(p))
		}
	}
	return nil
}

// joined is phrases joined with a separator: "SES: 3 Easy DKIM CNAMEs
// verified · Postal: selector postal-2026 verified".
func joined(sep string, parts []Text) Text {
	if len(parts) == 0 {
		return nil
	}
	return tx("text", A{"text": argJoin(sep, parts)})
}

// asSentence ends a text with a full stop when it has none.
func asSentence(t Text) Text {
	if t.Empty() {
		return nil
	}
	return tx("text", A{"text": argSentence(t)})
}

func (b *builder) content() {
	ct := b.f.Content
	if !ct.Checked {
		b.add(check{Key: "content.unchecked", Group: GroupContent, Status: Unknown, Title: tx("content.unchecked", nil),
			Detail: tx("content.uncheckedDetail", nil), Note: tx("note.notChecked", nil),
			Fixes: []fix{link(tx("fix.openInEditor", nil), b.editor(nil, ""))}})
		return
	}
	r := ct.Report
	subject := b.f.Campaign.Subject
	shortened := issuesOf(r, "links.shortener")
	spamLabel := tx("label.spamTriggers", nil)
	if strings.TrimSpace(subject) == "" {
		b.add(check{Key: "content.spam", Group: GroupContent, Status: Block, Title: tx("content.subjectEmpty", nil),
			FlatTitle: tx("content.subjectEmptyFlat", nil), Detail: tx("content.subjectEmptyDetail", nil), Label: tx("label.subject", nil),
			Fixes: []fix{link(tx("fix.openInEditor", nil), b.editor(nil, ""))}})
	} else if len(r.SubjectHeuristics) > 0 {
		n := len(r.SubjectHeuristics)
		args := A{"subject": subject, "count": n}
		c := check{Key: "content.spam", Group: GroupContent, Status: Warn, Title: tx("content.subjectSpam", args), FlatTitle: tx("content.subjectSpamFlat", args), Label: spamLabel,
			Ack: tx("content.subjectSpamAck", A{"count": n})}
		c.Detail = heuristicsText(r.SubjectHeuristics)
		if s := contentcheck.SuggestSubject(subject); s != "" && s != subject {
			c.Detail = sentences(c.Detail, tx("content.suggested", A{"subject": s}))
			c.Fixes = []fix{apply(FixUseSubject, tx("fix.useSuggestion", nil), "", map[string]string{"subject": s})}
		}
		b.add(c)
	} else {
		b.add(check{Key: "content.spam", Group: GroupContent, Status: Pass, Title: tx("content.noSpam", nil),
			Detail: tx("content.noSpamDetail", A{"subject": subject, "shortened": len(shortened)}), Label: spamLabel})
	}
	share := int(math.Round(r.ImageShare * 100))
	ratio := check{Key: "content.image_ratio", Group: GroupContent, Status: Pass, Label: tx("label.imageRatio", nil),
		Title:  tx("content.imageRatio", A{"share": ArgPercent(float64(share))}),
		Detail: tx("content.imageRatioDetail", A{"images": r.Images, "words": r.Words, "limit": ArgPercent(60)}),
		Short:  tx("content.imageRatioShort", A{"share": ArgPercent(float64(share))})}
	if share >= 60 {
		ratio.Status = Warn
		ratio.Title = tx("content.imageRatioHigh", A{"share": ArgPercent(float64(share)), "limit": ArgPercent(60)})
		ratio.Detail = tx("content.imageRatioHighDetail", A{"images": r.Images, "words": r.Words})
		ratio.Ack = tx("content.imageRatioAck", A{"share": ArgPercent(float64(share))})
		ratio.Fixes = []fix{link(tx("fix.openInEditor", nil), b.editor(nil, "deliverability"))}
	}
	b.add(ratio)
	kb := int(math.Round(float64(r.SizeBytes) / 1024))
	cssKB := int(math.Round(float64(ct.InlineCSSBytes) / 1024))
	trackKB := 0
	if r.Links > 0 {
		trackKB = int(math.Max(1, math.Round(float64(r.Links*200)/1024)))
	}
	sizeArgs := A{"kb": kb, "clip": 102, "css": cssKB, "tracking": trackKB}
	size := check{Key: "content.size", Group: GroupContent, Status: Pass, Label: tx("label.htmlSize", nil),
		Title:  tx("content.size", sizeArgs),
		Detail: tx("content.sizeDetail", sizeArgs),
		Short:  tx("content.sizeShort", sizeArgs)}
	if r.SizeBytes >= contentcheck.NearClipBytes {
		size.Status = Warn
		size.Title = tx("content.sizeNear", sizeArgs)
		if r.SizeBytes > contentcheck.ClipBytes {
			size.Title = tx("content.sizeOver", sizeArgs)
		}
		size.Detail = tx("content.sizeWarnDetail", sizeArgs)
		size.Ack = tx("content.sizeAck", sizeArgs)
		for _, is := range issuesOf(r, "content.size") {
			for _, fx := range is.Fixes {
				if fx.Kind == "minify" {
					size.Fixes = append(size.Fixes, apply(FixMinify, minifyLabel(fx.Label), "", nil))
				}
			}
		}
		size.Fixes = append(size.Fixes, link(tx("fix.openInEditor", nil), b.editor(nil, "deliverability")))
	}
	// With conditional content no one gets the whole email: Conditional
	// content checks the size of each version (design expansion, X11).
	if b.conditionalVersions() {
		size = b.conditionalSize(size)
	}
	b.add(size)
	// Alt text is an accessibility check when the Accessibility group runs.
	alt := check{Key: "content.alt_text", Group: GroupContent, Status: Pass, Label: tx("label.altText", nil)}
	switch {
	case b.f.Accessibility.Ran:
	case r.Images == 0:
		alt.Title, alt.Short = tx("content.altNoImages", nil), tx("content.altNoImagesShort", nil)
	case r.ImagesMissingAlt == 0:
		alt.Title = tx("content.altAll", A{"count": r.Images})
		alt.Short = tx("content.altAllShort", A{"count": r.Images})
	default:
		alt.Status = Warn
		alt.Title = tx("content.altMissing", A{"missing": r.ImagesMissingAlt, "images": r.Images})
		first := issuesOf(r, "content.alt_text")
		alt.Detail = tx("content.altDetail", nil)
		var path []int
		if len(first) > 0 {
			alt.Detail = sentences(contentAltTitle(r, first[0]), alt.Detail)
			path = first[0].Path
		}
		alt.Ack = tx("content.altAck", A{"count": r.ImagesMissingAlt})
		alt.Fixes = []fix{link(tx("fix.openInEditor", nil), b.editor(path, "deliverability"))}
	}
	if !b.f.Accessibility.Ran {
		b.add(alt)
	}
	if strings.TrimSpace(r.PlainText) != "" {
		ptKB := int(math.Max(1, math.Round(float64(len(r.PlainText))/1024)))
		b.add(check{Key: "content.plain_text", Group: GroupContent, Status: Pass, Title: tx("content.plainText", nil),
			Detail: tx("content.plainTextDetail", A{"kb": ptKB}), Label: tx("label.plainText", nil), Short: tx("content.plainTextShort", nil),
			Fixes: []fix{link(tx("fix.preview", nil), b.href("/campaigns/"+b.f.Campaign.ID+"/edit?view=preview"))}})
	} else {
		b.add(check{Key: "content.plain_text", Group: GroupContent, Status: Warn, Title: tx("content.plainTextEmpty", nil),
			Detail: tx("content.plainTextEmptyDetail", nil), Label: tx("label.plainText", nil),
			Ack: tx("content.plainTextEmptyAck", nil), Fixes: []fix{link(tx("fix.openInEditor", nil), b.editor(nil, ""))}})
	}
	b.grow()
}

// heuristicsText names the spam heuristics a subject trips: "All caps and
// urgency phrasing."
func heuristicsText(hs []string) Text {
	words := make([]any, 0, len(hs))
	for _, h := range hs {
		words = append(words, heuristicWord(h))
	}
	return tx("content.heuristics", A{"list": argCapital(ArgAnd(words))})
}

// heuristicWord names a subject heuristic (contentcheck.SubjectHeuristics).
func heuristicWord(h string) any {
	switch h {
	case "all caps":
		return tx("heuristic.allCaps", nil)
	case "repeated punctuation":
		return tx("heuristic.punctuation", nil)
	case "urgency":
		return tx("heuristic.urgency", nil)
	case "money words":
		return tx("heuristic.money", nil)
	case "click-bait":
		return tx("heuristic.clickbait", nil)
	}
	return h
}

var savesKB = regexp.MustCompile(`saves about (\d+) KB`)

// minifyLabel is the composer's minify fix ("Minify inline CSS · saves
// about 9 KB") in the reader's language.
func minifyLabel(label string) Text {
	if m := savesKB.FindStringSubmatch(label); m != nil {
		var kb int
		_, _ = fmt.Sscan(m[1], &kb)
		if t := tx("fix.minifySaves", A{"kb": kb}); t.en() == label {
			return t
		}
	}
	if label == tx("fix.minify", nil).en() {
		return tx("fix.minify", nil)
	}
	return RawText(label)
}

// contentAltTitle names the first image without alt text as the content
// check does ("Hero image has no alt text").
func contentAltTitle(r contentcheck.ContentCheckReport, is contentcheck.ContentCheckIssue) Text {
	var t Text
	switch {
	case is.Section >= 0 && is.Section < len(r.Sections):
		switch l := r.Sections[is.Section].Label; l {
		case "hero":
			t = tx("contentAlt.hero", nil)
		case "header":
			t = tx("contentAlt.logo", nil)
		default:
			t = tx("contentAlt.section", A{"section": argSection(l)})
		}
	default:
		t = tx("contentAlt.image", nil)
	}
	return verified(t, is.Title)
}

// verified keeps a text another package wrote in English when the
// message meant to say it in each reader's language doesn't say the same.
func verified(t Text, english string) Text {
	if t.en() == english {
		return t
	}
	return RawText(english)
}

func issuesOf(r contentcheck.ContentCheckReport, id string) []contentcheck.ContentCheckIssue {
	out := []contentcheck.ContentCheckIssue{}
	for _, is := range r.Issues {
		if is.ID == id {
			out = append(out, is)
		}
	}
	return out
}

func (b *builder) links() {
	ct := b.f.Content
	if !ct.Checked {
		return
	}
	r := ct.Report
	blockIssues := issuesOf(r, "links.blocklist")
	blocked := map[string]bool{}
	for _, is := range blockIssues {
		if h := hostOf(is.URL); h != "" {
			blocked[h] = true
		}
	}
	total := len(r.LinkList)
	var considered, probed, broken []contentcheck.ContentCheckLink
	for _, l := range r.LinkList {
		if blocked[l.Host] {
			continue
		}
		considered = append(considered, l)
		if l.Probed {
			probed = append(probed, l)
			if l.Broken {
				broken = append(broken, l)
			}
		}
	}
	respond := check{Key: "links.respond", Group: GroupLinks, Status: Pass, Label: tx("label.linksRespond", nil)}
	switch {
	case total == 0:
		respond.Title, respond.Short = tx("links.none", nil), tx("links.noneShort", nil)
	case len(probed) == 0:
		respond.Title = tx("links.found", A{"count": total})
		respond.Detail = tx("links.notRequested", nil)
		respond.Short = respond.Title
	case len(broken) == 0:
		all200 := true
		slow := probed[0]
		for _, l := range probed {
			if l.Status != 200 {
				all200 = false
			}
			if l.Millis > slow.Millis {
				slow = l
			}
		}
		if len(blocked) > 0 {
			respond.Title = tx("links.respondExceptBlocked", A{"count": len(considered), "total": total, "blocked": len(blockIssues)})
			respond.Short = respond.Title
		} else {
			respond.Title = tx("links.allRespond", A{"count": total, "http200": yes(all200)})
			respond.Short = tx("links.allRespondShort", A{"count": total})
			respond.every = total > 1
		}
		if slow.Millis > 0 {
			respond.Detail = tx("links.slowest", A{"url": displayURL(slow.URL), "ms": slow.Millis})
		}
	default:
		respond.Status = Warn
		respond.Title = tx("links.broken", A{"count": len(broken), "total": total})
		parts := []Text{}
		for i, l := range broken {
			if i == 3 {
				parts = append(parts, tx("links.brokenMore", A{"count": len(broken) - 3}))
				break
			}
			if l.Status == 0 {
				parts = append(parts, tx("links.noResponse", A{"url": displayURL(l.URL)}))
			} else {
				parts = append(parts, tx("links.returns", A{"url": displayURL(l.URL), "status": fmt.Sprint(l.Status)}))
			}
		}
		respond.Detail = joined(" · ", parts)
		respond.Ack = tx("links.brokenAck", A{"count": len(broken)})
		respond.Fixes = []fix{link(tx("fix.openInEditor", nil), b.editor(broken[0].Path, "deliverability"))}
	}
	b.add(respond)
	if len(blockIssues) == 0 {
		domains := []string{}
		seen := map[string]bool{}
		for _, l := range r.LinkList {
			d := contentcheck.RegistrableDomain(l.Host)
			if l.Host != d && strings.Count(l.Host, ".") == 2 && !strings.HasPrefix(l.Host, "www.") {
				d = l.Host
			}
			d = strings.TrimPrefix(d, "www.")
			if !seen[d] {
				seen[d] = true
				domains = append(domains, d)
			}
		}
		slices.Sort(domains)
		var detail Text
		if len(domains) > 0 {
			shown := domains
			if len(domains) > 4 {
				shown = domains[:4]
				detail = tx("links.domainsMore", A{"lists": argJoin(", ", ct.URIBlocklists), "hasLists": yes(len(ct.URIBlocklists) > 0), "domains": argJoin(", ", shown), "more": len(domains) - 4})
			} else {
				detail = tx("links.domains", A{"lists": argJoin(", ", ct.URIBlocklists), "hasLists": yes(len(ct.URIBlocklists) > 0), "domains": argJoin(", ", shown)})
			}
		} else if len(ct.URIBlocklists) > 0 {
			detail = RawText(strings.Join(ct.URIBlocklists, ", "))
		}
		b.add(check{Key: "links.blocklists", Group: GroupLinks, Status: Pass, Title: tx("links.notListed", nil), Detail: detail, Label: tx("label.uriBlocklists", nil),
			Short: tx("links.notListedShort", nil)})
	} else {
		done := map[string]bool{}
		for _, is := range blockIssues {
			host := hostOf(is.URL)
			if done[host] {
				continue
			}
			done[host] = true
			lists := strings.TrimPrefix(is.Tag, "Link domain on ")
			text, kind := linkTextFor(r, is.URL)
			used := tx("links.usedByLink", nil)
			if text != "" {
				used = tx("links.usedBy", A{"text": text, "kind": kind})
			}
			title := tx("links.listed", A{"host": host, "lists": lists})
			if named := tx("links.listed", A{"host": host, "lists": ArgAnd(listNames(lists))}); named.en() == title.en() {
				title = named
			}
			c := check{Key: "links.blocklist." + host, Group: GroupLinks, Status: Block,
				Title:  title,
				Detail: sentences(used, tx("links.listedDetail", nil)),
				Label:  tx("label.uriBlocklists", nil)}
			for _, fx := range is.Fixes {
				if fx.Kind == "replace_link" && fx.Value != "" {
					c.Fixes = append(c.Fixes, apply(FixReplaceLink, replaceLabel(fx.Label, fx.Value), "danger", map[string]string{"from": is.URL, "to": fx.Value}))
				}
			}
			c.Fixes = append(c.Fixes, link(tx("fix.editLink", nil), b.editor(is.Path, "deliverability")))
			b.add(c)
		}
	}
	if short := issuesOf(r, "links.shortener"); len(short) > 0 {
		first := short[0]
		title := tx("links.shortenerOne", A{"url": displayURL(first.URL)})
		if len(short) > 1 {
			title = tx("links.shortenerMany", A{"count": len(short)})
		}
		c := check{Key: "links.shortener", Group: GroupLinks, Status: Warn, Title: title,
			Detail: tx("links.shortenerDetail", nil), Label: tx("label.urlShortener", nil),
			Ack: tx("links.shortenerAck", A{"count": len(short)})}
		final := ""
		for _, l := range r.LinkList {
			if l.URL == first.URL && l.Final != "" {
				final = l.Final
			}
		}
		if final != "" && len(short) == 1 {
			c.Fixes = []fix{apply(FixReplaceLink, tx("fix.replaceWithDestination", nil), "", map[string]string{"from": first.URL, "to": final})}
		}
		c.Fixes = append(c.Fixes, link(tx("fix.editLink", nil), b.editor(first.Path, "deliverability")))
		b.add(c)
	}
	if empty := issuesOf(r, "links.empty"); len(empty) > 0 {
		b.add(check{Key: "links.empty", Group: GroupLinks, Status: Warn, Title: emptyLinkTitle(r, empty[0]), Detail: emptyLinkDetail(empty[0]), Label: tx("label.emptyLink", nil),
			Ack: tx("links.emptyAck", nil), Fixes: []fix{link(tx("fix.editLink", nil), b.editor(empty[0].Path, "deliverability"))}})
	}
}

// replaceLabel is the content check's replace fix ("Replace with
// acme.io/sale") in the reader's language.
func replaceLabel(label, to string) Text {
	return verified(tx("fix.replaceWith", A{"target": strings.TrimPrefix(to, "https://")}), label)
}

// emptyLinkTitle is the content check's empty button ("Button “Read the
// docs” has no link") in the reader's language.
func emptyLinkTitle(r contentcheck.ContentCheckReport, is contentcheck.ContentCheckIssue) Text {
	for _, l := range r.LinkList {
		if slices.Equal(l.Path, is.Path) && l.Text != "" {
			return verified(tx("links.emptyButton", A{"text": truncateText(l.Text, 40)}), is.Title)
		}
	}
	text := strings.TrimSuffix(strings.TrimPrefix(is.Title, "Button “"), "” has no link")
	return verified(tx("links.emptyButton", A{"text": text}), is.Title)
}

func emptyLinkDetail(is contentcheck.ContentCheckIssue) Text {
	return verified(tx("links.emptyDetail", nil), is.Detail)
}

// truncateText shortens text to n characters with an ellipsis, as the
// content check does.
func truncateText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// listNames reads the content check's list of blocklists ("SURBL and
// Spamhaus DBL") back into names.
func listNames(s string) []string {
	head, last := "", s
	if i := strings.LastIndex(s, " and "); i >= 0 {
		head, last = s[:i], s[i+len(" and "):]
	}
	out := []string{}
	if head != "" {
		out = append(out, strings.Split(head, ", ")...)
	}
	return append(out, last)
}

func hostOf(raw string) string {
	h := strings.TrimSpace(raw)
	if !strings.Contains(h, "://") {
		h = "https://" + h
	}
	u, err := url.Parse(h)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// displayURL drops the scheme and query: "docs.acme.io/changelog".
func displayURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	}
	return strings.TrimSuffix(u.Host+u.EscapedPath(), "/")
}

// linkTextFor is a link's text and what it is: link, button, image, menu
// link or social link.
func linkTextFor(r contentcheck.ContentCheckReport, href string) (string, string) {
	for _, l := range r.LinkList {
		if l.URL == href {
			kind := "link"
			switch l.Kind {
			case "button", "image":
				kind = l.Kind
			case "nav":
				kind = "menu"
			case "social":
				kind = "social"
			}
			return strings.TrimSpace(l.Text), kind
		}
	}
	return "", "link"
}

func (b *builder) compliance() {
	cf := b.f.Compliance
	if cf.HTTPS {
		b.add(check{Key: "compliance.list_unsubscribe", Group: GroupCompliance, Status: Pass, Title: tx("compliance.oneClick", nil),
			Detail: RawText(fmt.Sprintf("List-Unsubscribe: <%s> · List-Unsubscribe-Post: List-Unsubscribe=One-Click", cf.UnsubscribeURL)), Mono: true,
			Label: tx("label.oneClick", nil), Short: tx("compliance.oneClickShort", nil)})
	} else {
		b.add(check{Key: "compliance.list_unsubscribe", Group: GroupCompliance, Status: Warn, Title: tx("compliance.oneClickHttps", nil),
			Detail: tx("compliance.oneClickHttpsDetail", A{"url": cf.UnsubscribeURL}),
			Label:  tx("label.oneClick", nil), Ack: tx("compliance.oneClickHttpsAck", nil)})
	}
	ct := b.f.Content
	footerBlocked := false
	if ct.Checked {
		if f := issuesOf(ct.Report, "compliance.footer"); len(f) > 0 {
			footerBlocked = true
			missing := "both"
			switch f[0].Title {
			case "No postal address in the footer":
				missing = "address"
			case "No unsubscribe link in the body":
				missing = "unsubscribe"
			}
			detail := tx("compliance.footerDetail", nil)
			if ct.FooterRemovedAt != nil {
				detail = tx("compliance.footerRemoved", A{"when": b.when(*ct.FooterRemovedAt)})
			}
			b.add(check{Key: "compliance.footer", Group: GroupCompliance, Status: Block, Title: tx("compliance.footer", A{"missing": missing}),
				FlatTitle: tx("compliance.footerFlat", A{"missing": missing}), Detail: detail, Label: tx("label.footer", nil),
				Fixes: []fix{apply(FixInsertFooter, tx("fix.insertFooter", nil), "danger", nil), link(tx("fix.openInEditor", nil), b.editor(nil, "deliverability"))}})
		} else {
			detail := RawText(ct.Address)
			if detail.Empty() {
				detail = tx("compliance.footerFound", nil)
			}
			b.add(check{Key: "compliance.footer", Group: GroupCompliance, Status: Pass, Title: tx("compliance.footerPass", nil),
				Detail: detail, Label: tx("label.footer", nil), Short: tx("compliance.footerShort", nil)})
		}
	}
	c := check{Key: "compliance.bulk_sender", Group: GroupCompliance, Status: Pass, Title: tx("compliance.bulk", nil), Label: tx("label.bulkRules", nil),
		Short: tx("compliance.bulkShort", nil)}
	parts := []Text{}
	if b.f.Auth.DMARCStatus == "pass" {
		parts = append(parts, tx("compliance.partDmarc", nil))
	} else {
		c.Status = Warn
		c.Title = tx("compliance.bulkDmarc", nil)
		c.Ack = tx("compliance.bulkDmarcAck", nil)
	}
	if cf.HTTPS {
		parts = append(parts, tx("compliance.partOneClick", nil))
	}
	if cf.ComplaintRate != nil {
		cr := *cf.ComplaintRate
		rate := ArgPercent(cr)
		switch {
		case cr >= 0.3:
			c.Status = Block
			c.Title = tx("compliance.complaintsOver", A{"rate": rate, "limit": ArgPercent(0.3)})
			c.Fixes = []fix{link(tx("fix.openDeliverability", nil), b.href("/deliverability"))}
		case cr >= 0.1 && c.Status == Pass:
			c.Status = Warn
			c.Title = tx("compliance.complaintsHigh", A{"rate": rate, "limit": ArgPercent(0.1)})
			c.Ack = tx("compliance.complaintsHighAck", A{"limit": ArgPercent(0.1)})
		}
		if cr >= 0.3 {
			parts = append(parts, tx("compliance.partComplaints", A{"rate": rate}))
		} else {
			parts = append(parts, tx("compliance.partComplaintsUnder", A{"rate": rate, "limit": ArgPercent(0.3)}))
		}
	} else {
		parts = append(parts, tx("compliance.partNotMeasured", nil))
	}
	parts = append(parts, tx("compliance.partUnsubscribes", A{"days": 2}))
	c.Detail = joined(" · ", parts)
	if footerBlocked && c.Status == Pass {
		// The rules also want a visible unsubscribe link and address in the
		// body: the footer block says they aren't met, and its fix meets them.
		return
	}
	b.add(c)
}

func (b *builder) audience() {
	a := b.f.Audience
	choose := link(tx("fix.chooseAudience", nil), b.editor(nil, ""))
	if a.Empty {
		b.add(check{Key: "audience.empty", Group: GroupAudience, Status: Block, Title: tx("audience.none", nil),
			FlatTitle: tx("audience.noneFlat", nil), Detail: tx("audience.noneDetail", nil), Label: tx("label.audience", nil),
			Fixes: []fix{choose}})
		return
	}
	if a.Final == 0 {
		b.add(check{Key: "audience.empty", Group: GroupAudience, Status: Block, Title: tx("audience.nobody", nil),
			FlatTitle: tx("audience.nobodyFlat", nil), Detail: tx("audience.nobodyDetail", nil), Label: tx("label.audience", nil),
			Fixes: []fix{choose}})
	}
	days := a.UnengagedDays
	if days <= 0 {
		days = 180
	}
	share := 0.0
	if a.Final > 0 {
		share = float64(a.Unengaged) / float64(a.Final) * 100
	}
	un := check{Key: "audience.unengaged", Group: GroupAudience, Status: Pass}
	switch {
	case a.Unengaged == 0 && a.UnengagedExcluded:
		un.Title = tx("audience.unengagedExcluded", A{"count": a.SunsetExcluded})
		un.Detail = tx("audience.unengagedExcludedDetail", A{"days": days})
		un.Short = tx("audience.unengagedExcludedShort", nil)
	case a.Unengaged > 0 && a.SunsetOverride:
		// The campaign overrides the sunset policy (SendReadinessBlocked).
		un.Status = Warn
		un.Title = tx("audience.sunsetIncluded", A{"count": a.Unengaged})
		un.FlatTitle = tx("audience.sunsetIncludedFlat", A{"count": a.Unengaged})
		un.Detail = tx("audience.sunsetIncludedDetail", A{"rest": a.Final - a.Unengaged})
		un.Evidence = tx("audience.sunsetIncludedEvidence", A{"share": ArgPercent1(share), "count": a.Unengaged})
		un.Ack = tx("audience.sunsetIncludedAck", A{"count": a.Unengaged})
		un.Fixes = []fix{apply(FixExcludeSunset, tx("fix.exclude", A{"count": a.Unengaged}), "", map[string]string{"count": fmt.Sprint(a.Unengaged)})}
	case a.Unengaged > 0 && (share >= UnengagedWarnShare || a.Unengaged >= UnengagedWarnCount):
		un.Status = Warn
		un.Title = tx("audience.unengaged", A{"count": a.Unengaged, "days": days})
		un.FlatTitle = tx("audience.unengagedFlat", A{"count": a.Unengaged})
		un.Detail = tx("audience.unengagedDetail", A{"rest": a.Final - a.Unengaged})
		un.Evidence = tx("audience.unengagedEvidence", A{"share": ArgPercent1(share), "count": a.Unengaged, "days": days})
		un.Ack = tx("audience.unengagedAck", A{"count": a.Unengaged})
		un.Fixes = []fix{
			apply(FixExcludeSunset, tx("fix.excludeUnengaged", A{"count": a.Unengaged}), "primary", map[string]string{"count": fmt.Sprint(a.Unengaged)}),
			link(tx("fix.reviewThem", nil), b.href("/audience?tier=sunset")),
		}
	case a.Unengaged > 0:
		un.Title = tx("audience.unengagedFew", A{"count": a.Unengaged, "share": ArgPercent1(share)})
		un.Detail = tx("audience.unengagedFewDetail", A{"days": days, "limit": ArgPercent(1), "after": 30})
		un.Short = tx("audience.unengagedFewShort", A{"count": a.Unengaged})
	default:
		un.Title = tx("audience.engaged", nil)
		un.Detail = tx("audience.engagedDetail", A{"days": days})
		un.Short = tx("audience.engagedShort", nil)
	}
	// No label: "Audience quality" names it in compact tables (ApprovalReview).
	b.add(un)
	if a.RecentlyBounced == 0 {
		b.add(check{Key: "audience.bounced", Group: GroupAudience, Status: Pass, Title: tx("audience.notBounced", nil),
			Detail: tx("audience.notBouncedDetail", A{"count": a.Suppressed}), Label: tx("label.recentlyBounced", nil)})
	} else {
		b.add(check{Key: "audience.bounced", Group: GroupAudience, Status: Warn,
			Title:  tx("audience.bounced", A{"count": a.RecentlyBounced}),
			Detail: tx("audience.bouncedDetail", nil),
			Label:  tx("label.recentlyBounced", nil), Ack: tx("audience.bouncedAck", A{"count": a.RecentlyBounced}),
			Fixes: []fix{link(tx("fix.reviewThem", nil), b.href("/audience?status=bounced"))}})
	}
	b.add(check{Key: "audience.role", Group: GroupAudience, Status: Pass, Label: tx("label.roleAddresses", nil),
		Title: tx("audience.role", A{"count": a.Role}), Detail: tx("audience.roleDetail", nil)})
	if a.SegmentsSkipped {
		b.add(check{Key: "audience.segments", Group: GroupAudience, Status: Warn, Title: tx("audience.segmentsSkipped", nil),
			Detail: tx("audience.segmentsSkippedDetail", nil), Label: tx("label.segments", nil),
			Ack: tx("audience.segmentsSkippedAck", nil)})
	}
}

// streamText names a sending stream: "Broadcasts".
func streamText(s string) Text {
	switch s {
	case "automation":
		return tx("stream.automations", nil)
	case "system":
		return tx("stream.system", nil)
	}
	return tx("stream.broadcasts", nil)
}

// policyText says what happens when a provider fails.
func policyText(p string) Text {
	switch p {
	case "automatic":
		return tx("policy.automatic", nil)
	case "warm_only":
		return tx("policy.warmOnly", nil)
	}
	return tx("policy.pauseAndAsk", nil)
}

func (b *builder) shareStatusText(s ShareFacts) Text {
	switch {
	case !s.Available && s.Status == state.Down && s.DownSince != nil:
		return tx("share.downSince", A{"when": b.when(*s.DownSince)})
	case s.Status == state.WarmingUp && s.WarmupDays > 0:
		return tx("share.warmingUp", A{"day": s.WarmupDay, "days": s.WarmupDays})
	}
	return statusText(s.Status)
}

// downText says why a provider can't take mail: "mta.acme.io has refused
// connections since 08:52 (ECONNREFUSED)".
func (b *builder) downText(s ShareFacts) Text {
	switch {
	case s.Down != nil && s.Down.Auth:
		return tx("share.rejectsCredentials", A{"provider": s.Down.Who, "when": b.when(s.Down.Since)})
	case s.Down != nil && s.Down.Code != "":
		return tx("share.refusesCode", A{"host": s.Down.Who, "when": b.when(s.Down.Since), "code": s.Down.Code})
	case s.Down != nil:
		return tx("share.refuses", A{"host": s.Down.Who, "when": b.when(s.Down.Since)})
	}
	return RawText(sentence(s.DownDetail))
}

func (b *builder) routing() {
	rt := b.f.Route
	stream := streamText(rt.Stream)
	routes := b.href("/providers/routing")
	if rt.NoRoute || len(rt.Shares) == 0 {
		b.add(check{Key: "routing.health", Group: GroupRouting, Status: Block, Title: tx("routing.noProvider", A{"stream": stream}),
			Detail: tx("routing.noProviderDetail", A{"stream": stream}), Label: tx("label.route", nil), Fixes: []fix{link(tx("fix.setUpRouting", nil), routes)}})
		return
	}
	shares := b.shares()
	var down []ShareFacts
	for _, s := range shares {
		if !s.Available && s.Recipients > 0 {
			down = append(down, s)
		}
	}
	if len(down) > 0 {
		d := down[0]
		word := "down"
		switch d.Status {
		case state.Paused:
			word = "paused"
		case state.Degraded:
			word = "silent"
		}
		pct := 0.0
		var held int64
		for _, s := range down {
			pct += s.Percent
			held += s.Recipients
		}
		parts := []Text{}
		if d.DownDetail != "" || d.Down != nil {
			parts = append(parts, b.downText(d))
		}
		parts = append(parts, tx("routing.heldShare", A{"share": ArgPercent(math.Round(pct))}))
		c := check{Key: "routing.health", Group: GroupRouting, Status: Block, Label: tx("label.routeHealth", nil),
			Title: tx("routing.down", A{"provider": d.Name, "state": word, "stream": stream, "policy": policyText(rt.Policy)})}
		if o := b.otherAvailable(d.ProviderID); o != nil {
			if o.Quota != nil && o.Quota.Max > 0 {
				parts = append(parts, tx("routing.otherQuota", A{"provider": o.Short, "status": firstNonEmpty(o.Status, state.Healthy), "statusName": statusText(o.Status),
					"free": o.Quota.Max - o.Quota.Used, "max": o.Quota.Max}))
			} else {
				parts = append(parts, tx("routing.other", A{"provider": o.Short, "status": firstNonEmpty(o.Status, state.Healthy), "statusName": statusText(o.Status)}))
			}
			c.Fixes = append(c.Fixes, routeOnlyFix(o, tx("fix.sendAllVia", A{"share": ArgPercent(100), "provider": o.Short}), "danger"))
		}
		c.Fixes = append(c.Fixes, link(tx("fix.check", A{"provider": d.Short}), b.href("/providers")))
		c.Detail = sentences(parts...)
		if rt.Policy != "pause_and_ask" && b.otherAvailable(d.ProviderID) != nil {
			c.Status = Warn
			c.Title = tx("routing.failsOver", A{"provider": d.Name, "state": word, "share": ArgPercent(math.Round(pct))})
			c.Ack = tx("routing.failsOverAck", A{"count": held, "provider": d.Name})
		}
		b.add(c)
	} else {
		parts := []Text{}
		for _, s := range shares {
			status := statusText(s.Status)
			if s.Status == state.WarmingUp && s.WarmupDays > 0 {
				status = tx("share.warmingUpDay", A{"day": s.WarmupDay})
			}
			parts = append(parts, tx("routing.healthyShare", A{"provider": s.Name, "status": status, "share": ArgPercent(math.Round(s.Percent))}))
		}
		c := check{Key: "routing.health", Group: GroupRouting, Status: Pass, Title: tx("routing.healthy", A{"count": len(shares)}), Label: tx("label.routeHealth", nil),
			Detail: tx("routing.healthyDetail", A{"shares": argJoin(" · ", parts), "policy": policyText(rt.Policy)}), Short: tx("routing.healthyShort", nil),
			Fixes: []fix{link(tx("fix.editRoute", nil), routes)}}
		// Quota on the provider's own 24-hour allowance.
		for _, s := range shares {
			if s.Quota != nil && s.Quota.Kind == "rolling_24h" && s.Quota.Max > 0 && s.Recipients > s.Quota.Max-s.Quota.Used {
				c.Status = Warn
				c.Title = tx("routing.quota", A{"provider": s.Name, "left": max(s.Quota.Max-s.Quota.Used, 0), "max": s.Quota.Max, "count": s.Recipients})
				c.Detail = tx("routing.quotaDetail", nil)
				c.Ack = tx("routing.quotaAck", A{"provider": s.Name})
				c.Short = nil
			}
		}
		b.add(c)
	}
	// Terms of the providers that can send: one that is down sends nothing,
	// and the health check above already blocks on it.
	sending := b.availableShares()
	if len(sending) == 0 {
		sending = shares
	}
	var notAllowed []ShareFacts
	for _, s := range sending {
		if !s.BulkAllowed || s.Sandbox {
			notAllowed = append(notAllowed, s)
		}
	}
	if len(notAllowed) == 0 {
		parts := []Text{}
		for _, s := range sending {
			parts = append(parts, tx("routing.termsOf", A{"provider": s.Short, "terms": termsText(s)}))
		}
		args := A{"count": len(sending), "provider": sending[0].Short}
		b.add(check{Key: "routing.terms", Group: GroupRouting, Status: Pass, Title: tx("routing.terms", args), Detail: joined(" · ", parts), Label: tx("label.bulkTerms", nil),
			Short: tx("routing.termsShort", args)})
	} else {
		s := notAllowed[0]
		c := check{Key: "routing.terms", Group: GroupRouting, Status: Block, Label: tx("label.bulkTerms", nil),
			Title:  tx("routing.termsNo", A{"provider": s.Name}),
			Detail: sentences(tx("routing.termsNoWhy", A{"terms": argCapital(termsText(s))}), tx("routing.termsNoFix", A{"stream": stream})),
			Fixes:  []fix{link(tx("fix.editRoute", nil), routes)}}
		if s.Sandbox {
			c.Title = tx("routing.sandbox", A{"provider": s.Name})
			c.Detail = tx("routing.sandboxDetail", nil)
			c.Fixes = []fix{link(tx("fix.openProviders", nil), b.href("/providers"))}
		}
		b.add(c)
	}
	var double []ShareFacts
	for _, s := range shares {
		if s.Tracking.Mode != "provider" && (s.Tracking.OpenPixel || s.Tracking.LinkRewrite) {
			double = append(double, s)
		}
	}
	if len(double) == 0 {
		parts := []Text{}
		for _, s := range shares {
			parts = append(parts, trackingNote(s))
		}
		b.add(check{Key: "routing.tracking", Group: GroupRouting, Status: Pass, Title: tx("routing.trackingOff", nil),
			Detail: joined(" · ", parts), Label: tx("label.providerTracking", nil), Short: tx("routing.trackingOffShort", nil)})
	} else {
		s := double[0]
		what := "both"
		switch {
		case s.Tracking.OpenPixel && !s.Tracking.LinkRewrite:
			what = "pixel"
		case !s.Tracking.OpenPixel && s.Tracking.LinkRewrite:
			what = "links"
		}
		detail := RawText(s.Tracking.Detail)
		if detail.Empty() {
			detail = tx("routing.trackingTwice", nil)
		}
		b.add(check{Key: "routing.tracking", Group: GroupRouting, Status: Warn, Title: tx("routing.trackingOn", A{"provider": s.Name, "what": what}),
			Detail: detail, Label: tx("label.providerTracking", nil),
			Ack:   tx("routing.trackingOnAck", A{"provider": s.Name}),
			Fixes: []fix{link(tx("fix.turnOff", nil), b.href("/providers"))}})
	}
}

// termsText is what a provider's terms say about bulk marketing ("yes, with
// production access").
func termsText(s ShareFacts) Text {
	if s.TermsKind != "" {
		return tx("terms."+s.TermsKind, nil)
	}
	return RawText(s.Terms)
}

func trackingNote(s ShareFacts) Text {
	scope := strings.TrimSpace(s.Tracking.Scope)
	switch {
	case s.Tracking.Mode == "provider":
		return tx("tracking.provider", A{"provider": s.Short})
	case s.Adapter == "postal" || s.Adapter == "smtp":
		return tx("tracking.noRewriting", A{"provider": s.Short})
	case scope != "":
		return tx("tracking.scope", A{"provider": s.Short, "scope": lowerFirst(scope)})
	}
	return tx("tracking.none", A{"provider": s.Short})
}

func (b *builder) estimates() ReadinessEstimates {
	a := b.f.Audience
	out := ReadinessEstimates{Throttling: []ReadinessThrottle{}}
	t := &estimateTexts{}
	aud := ReadinessAudience{Rows: []ReadinessAudienceRow{}, Final: a.Final, Lists: a.Lists, Suppressed: a.Suppressed, Role: a.Role, Duplicates: a.Duplicates,
		Unengaged: a.Unengaged, UnengagedExcluded: a.UnengagedExcluded}
	if aud.Lists == nil {
		aud.Lists = []string{}
	}
	addRow := func(label Text, count int64, kind string) {
		aud.Rows = append(aud.Rows, ReadinessAudienceRow{Label: label.en(), Count: count, Kind: kind})
		t.Rows = append(t.Rows, label)
	}
	if !a.Empty {
		addRow(b.baseLabel(), a.Base, "base")
		addRow(tx("estimate.suppressed", nil), a.Suppressed, "minus")
		addRow(tx("estimate.role", nil), a.Role, "minus")
		if a.Duplicates > 0 {
			addRow(tx("estimate.duplicates", nil), a.Duplicates, "minus")
		}
		switch {
		case a.SunsetOverride:
			addRow(tx("estimate.sunsetOverride", nil), a.StoppedIncluded, "plus")
		case a.Stopped > 0:
			addRow(tx("estimate.sunsetStopped", nil), a.Stopped, "minus")
		}
		if a.UnengagedExcluded {
			addRow(tx("estimate.unengaged", nil), a.SunsetExcluded, "minus")
		}
	}
	out.Audience = aud

	rt := b.f.Route
	t.Stream = streamText(rt.Stream)
	route := ReadinessRoute{Stream: firstNonEmpty(rt.Stream, "broadcast"), StreamLabel: t.Stream.en(), Policy: firstNonEmpty(rt.Policy, "pause_and_ask"),
		Override: rt.Override, Shares: []ReadinessRouteShare{}}
	pcts := []string{}
	maxMinutes := 0
	for _, s := range rt.Shares {
		st := shareTexts{Status: b.shareStatusText(s)}
		rs := ReadinessRouteShare{ProviderID: s.ProviderID, Name: s.Name, Short: s.Short, Adapter: s.Adapter, Label: s.Label, Color: state.Color(s.Adapter),
			Percent: math.Round(s.Percent*10) / 10, Recipients: s.Recipients, Status: s.Status, Available: s.Available,
			WarmupDay: s.WarmupDay, WarmupDays: s.WarmupDays, Rate: s.Rate}
		rs.Pricing = pricingModel(s.Pricing.Model)
		switch rs.Pricing {
		case "self_hosted":
			st.Price = tx("estimate.selfHosted", nil)
		case "plan":
			st.Price = tx("estimate.plan", nil)
		default:
			st.Price = tx("estimate.per1000", A{"price": ArgMoney(s.Pricing.Per1000), "count": 1000})
			rs.Cost = math.Round(float64(s.Recipients)*s.Pricing.Per1000/1000*100) / 100
		}
		if s.Rate > 0 {
			st.Rate = tx("estimate.rate", A{"rate": math.Round(s.Rate*10) / 10, "ramp": yes(s.RateRamp)})
			rs.Minutes = Minutes(float64(s.Recipients) / s.Rate)
		}
		rs.StatusLabel, rs.PriceLabel, rs.RateLabel = st.Status.en(), st.Price.en(), st.Rate.en()
		t.Shares = append(t.Shares, st)
		if s.Available {
			route.TotalCost += rs.Cost
			maxMinutes = max(maxMinutes, rs.Minutes)
		} else if s.Recipients > 0 {
			route.Down++
		}
		route.Shares = append(route.Shares, rs)
		pcts = append(pcts, trimNumber(math.Round(s.Percent), 0))
	}
	route.TotalCost = math.Round(route.TotalCost*100) / 100
	t.Split = t.Stream
	if len(pcts) > 0 {
		t.Split = tx("estimate.split", A{"stream": t.Stream, "split": strings.Join(pcts, " / ")})
	}
	route.SplitLabel = t.Split.en()
	for _, k := range MailboxOrder {
		n := a.Mailboxes[k]
		rate := b.f.MailboxRates[k]
		name := mailboxName(k)
		out.Throttling = append(out.Throttling, ReadinessThrottle{Key: k, Name: name.en(), Recipients: n, Rate: rate})
		t.Throttles = append(t.Throttles, name)
		if rate > 0 && n > 0 {
			maxMinutes = max(maxMinutes, Minutes(float64(n)/rate))
		}
	}
	route.TotalMinutes = maxMinutes
	// Moving a held share to the largest available provider.
	for _, s := range rt.Shares {
		if s.Available || s.Recipients == 0 {
			continue
		}
		if o := b.otherAvailable(s.ProviderID); o != nil {
			m := &ReadinessHeldMove{From: s.Short, To: o.Short, ToID: o.ProviderID}
			if o.Pricing.Model == "metered" || o.Pricing.Model == "" {
				m.AddCost = math.Round(float64(s.Recipients)*o.Pricing.Per1000/1000*100) / 100
			}
			if o.Rate > 0 {
				m.AddMinutes = Minutes(float64(s.Recipients) / o.Rate)
			}
			route.Move = m
		}
		break
	}
	out.Route = route
	out.texts = t
	return out
}

// pricingModel is how a provider is paid for: metered, plan or self_hosted.
func pricingModel(m string) string {
	switch m {
	case "plan", "self_hosted":
		return m
	}
	return "metered"
}

// baseLabel names the audience: "Weekly Digest list", "All subscribers".
func (b *builder) baseLabel() Text {
	a := b.f.Audience
	switch a.BaseKind {
	case BaseAll:
		return tx("audience.all", nil)
	case BaseList:
		return tx("audience.baseList", A{"name": a.BaseName})
	case BaseSegment:
		return tx("audience.baseSegment", A{"name": a.BaseName})
	case BaseNone:
		return tx("audience.base", nil)
	}
	return RawText(a.BaseLabel)
}

// groupLabel is the caption of the audience-quality group: "Weekly Digest",
// "All subscribers".
func (b *builder) groupLabel() Text {
	a := b.f.Audience
	switch a.BaseKind {
	case BaseAll:
		return tx("audience.all", nil)
	case BaseNone:
		return tx("audience.base", nil)
	case BaseList, BaseSegment:
		return RawText(a.BaseName)
	}
	return RawText(firstNonEmpty(a.GroupLabel, a.BaseLabel))
}

func rank(status string) int {
	switch status {
	case Block:
		return 0
	case Unknown:
		return 1
	case Warn:
		return 2
	}
	return 3
}

func (b *builder) groups() []ReadinessGroup {
	out := []ReadinessGroup{}
	for _, g := range Groups {
		grp := ReadinessGroup{Key: g, Status: Pass}
		var passes []check
		for _, c := range b.checks {
			if c.Group != g {
				continue
			}
			grp.Total++
			if c.Status == Pass {
				grp.Passed++
				if !c.Short.Empty() {
					passes = append(passes, c)
				}
			}
			if rank(c.Status) < rank(grp.Status) {
				grp.Status = c.Status
			}
		}
		if grp.Total == 0 {
			continue
		}
		t := &groupTexts{Title: groupTitle(g), Caption: b.caption(g, grp), Summary: b.summary(g, passes), Running: b.running(g)}
		switch g {
		case GroupAccessibility:
			t.Note = b.accessibilityNote()
		case GroupConditional:
			t.Note = b.conditionalNote()
		}
		grp.Title, grp.Caption, grp.Summary, grp.Running, grp.Note = t.Title.en(), t.Caption.en(), t.Summary.en(), t.Running.en(), t.Note.en()
		grp.texts = t
		out = append(out, grp)
	}
	return out
}

func (b *builder) caption(g string, grp ReadinessGroup) Text {
	switch g {
	case GroupReputation:
		r := b.f.Reputation
		via := "hostrepute"
		if r.Via == "builtin" {
			via = "builtin"
		}
		if r.Status == Unknown {
			return tx("caption.reputationUnknown", A{"via": via, "sources": r.Sources})
		}
		return tx("caption.reputation", A{"via": via, "sources": r.Sources, "passed": grp.Passed, "total": grp.Total})
	case GroupAuthentication:
		domain := firstNonEmpty(b.f.Auth.Domain, b.f.Campaign.SenderDomain)
		names := shortsOf(b.availableShares())
		if len(names) == 0 {
			return RawText(domain)
		}
		return tx("caption.authentication", A{"domain": domain, "providers": ArgAnd(names)})
	case GroupContent:
		name := b.f.Campaign.Name
		if n := len(b.f.Content.Versions); n > 0 && b.f.Campaign.CompiledAt != nil {
			return tx("caption.contentVersions", A{"name": name, "count": n + 1, "when": b.when(*b.f.Campaign.CompiledAt)})
		}
		if b.f.Campaign.CompiledAt != nil {
			return tx("caption.content", A{"name": name, "when": b.when(*b.f.Campaign.CompiledAt)})
		}
		return RawText(name)
	case GroupLinks:
		r := b.f.Content.Report
		return tx("caption.links", A{"count": len(r.LinkList), "every": yes(len(b.f.Content.Versions) > 0), "when": b.when(r.CheckedAt)})
	case GroupLanguages:
		return b.languagesCaption()
	case GroupCompliance:
		return tx("caption.compliance", nil)
	case GroupAudience:
		a := b.f.Audience
		if a.Empty {
			return tx("caption.audienceEmpty", nil)
		}
		return tx("caption.audience", A{"name": b.groupLabel(), "count": a.Base})
	case GroupRouting:
		return tx("caption.routing", A{"stream": streamText(b.f.Route.Stream)})
	case GroupAccessibility:
		return b.accessibilityCaption()
	case GroupConditional:
		return b.conditionalCaption()
	}
	return nil
}

// summaryOrder puts a group's passing checks in the order its one-line
// summary names them ("71 KB of 102 KB, image ratio 44%, …"); checks not
// listed keep their place after the listed ones.
var summaryOrder = map[string]int{"content.size": 1, "content.image_ratio": 2, "content.alt_text": 3, "content.plain_text": 4}

// summary is the line a group shows when its passing checks are collapsed.
func (b *builder) summary(g string, passes []check) Text {
	passes = slices.Clone(passes)
	slices.SortStableFunc(passes, func(x, y check) int {
		rx, ry := summaryOrder[x.Key], summaryOrder[y.Key]
		if rx == 0 {
			rx = len(summaryOrder) + 1
		}
		if ry == 0 {
			ry = len(summaryOrder) + 1
		}
		return rx - ry
	})
	shorts := make([]Text, 0, len(passes))
	for _, c := range passes {
		shorts = append(shorts, c.Short)
	}
	switch g {
	case GroupAuthentication:
		parts := []Text{}
		for _, c := range passes {
			switch c.Key {
			case "auth.spf", "auth.dkim", "auth.dmarc", "auth.mail_from":
				parts = append(parts, c.Short)
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return tx("summary.authentication", A{"records": ArgAnd(parts), "providers": ArgAnd(shortsOf(b.availableShares()))})
	case GroupContent, GroupCompliance:
		if g == GroupCompliance && len(shorts) < 3 {
			out := []Text{}
			for _, c := range passes {
				switch c.Key {
				case "compliance.list_unsubscribe":
					out = append(out, tx("summary.headers", nil))
				case "compliance.bulk_sender":
					out = append(out, tx("summary.bulkMet", nil))
				default:
					out = append(out, c.Short)
				}
			}
			return joined("; ", out)
		}
		return joined(", ", shorts)
	case GroupAudience:
		a := b.f.Audience
		return tx("summary.audience", A{"bounced": a.RecentlyBounced, "role": a.Role, "suppressed": a.Suppressed})
	}
	return joined("; ", shorts)
}

func (b *builder) running(g string) Text {
	switch g {
	case GroupReputation:
		assets := []string{}
		for _, s := range b.availableShares() {
			assets = append(assets, s.IPs...)
		}
		if d := firstNonEmpty(b.f.Auth.Domain, b.f.Campaign.SenderDomain); d != "" {
			assets = append(assets, d)
		}
		return tx("running.reputationAssets", A{"assets": ArgAnd(assets), "sources": b.f.Reputation.Sources})
	case GroupAuthentication:
		return tx("running.authenticationFor", A{"providers": ArgAnd(shortsOf(b.availableShares()))})
	case GroupLinks:
		return tx("running.linksCount", A{"count": len(b.f.Content.Report.LinkList)})
	}
	return runningText(g)
}

// RunningText is what a group shows while its checks run, before facts exist.
func RunningText(g string) string { return runningText(g).en() }

func runningText(g string) Text {
	switch g {
	case GroupReputation, GroupAuthentication, GroupContent, GroupLinks, GroupCompliance, GroupAudience, GroupRouting, GroupLanguages, GroupAccessibility, GroupConditional:
		return tx("running."+g, nil)
	}
	return nil
}
