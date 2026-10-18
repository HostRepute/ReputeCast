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

// Group titles and the prefixes used when checks are listed by severity.
var (
	groupTitles = map[string]string{
		GroupReputation: "Sender reputation", GroupAuthentication: "Authentication", GroupContent: "Content", GroupLinks: "Links",
		GroupCompliance: "Compliance", GroupAudience: "Audience quality", GroupRouting: "Routing", GroupLanguages: "Languages",
	}
	groupPrefixes = map[string]string{
		GroupReputation: "Sender reputation", GroupAuthentication: "Authentication", GroupContent: "Content", GroupLinks: "Links",
		GroupCompliance: "Compliance", GroupAudience: "Audience", GroupRouting: "Routing", GroupLanguages: "Languages",
	}
)

// GroupTitle is the heading of a check group ("Audience quality").
func GroupTitle(g string) string { return groupTitles[g] }

// DefaultMailboxRates are the per-mailbox-provider throttles: Outlook, Yahoo
// and iCloud accept slower streams more reliably; Gmail and the rest take
// each provider's full rate. The sending engine adapts them when deferrals rise.
var DefaultMailboxRates = map[string]float64{"gmail": 0, "outlook": 5, "yahoo": 4, "icloud": 3, "other": 0}

// MailboxOrder is the chart order of mailbox providers.
var MailboxOrder = []string{"gmail", "outlook", "yahoo", "icloud", "other"}

// MailboxName is the display name of a mailbox provider key.
func MailboxName(k string) string {
	switch k {
	case "gmail":
		return "Gmail"
	case "outlook":
		return "Outlook"
	case "yahoo":
		return "Yahoo"
	case "icloud":
		return "iCloud"
	}
	return "Other"
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
	checks []ReadinessCheck
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
	b.web()
	b.languages()
	b.accessibility()
	b.links()
	b.conditional()
	b.compliance()
	b.audience()
	b.routing()
	b.everyVersion()
	for i := range b.checks {
		if b.checks[i].Fixes == nil {
			b.checks[i].Fixes = []ReadinessFix{}
		}
		if b.checks[i].FlatTitle == "" {
			b.checks[i].FlatTitle = groupPrefixes[b.checks[i].Group] + ": " + b.checks[i].Title
		}
		if b.checks[i].Evidence == "" {
			b.checks[i].Evidence = sentence(b.checks[i].Title)
			if b.checks[i].Detail != "" && !b.checks[i].Mono {
				b.checks[i].Evidence += " " + sentence(b.checks[i].Detail)
			}
		}
	}
	out := ReadinessResult{Checks: b.checks, Estimates: b.estimates()}
	for _, c := range b.checks {
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
		out.Problem = firstNonEmpty(f.Reputation.Outage, f.Reputation.Reason, "Some checks couldn't run.")
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

func (b *builder) add(c ReadinessCheck) { b.checks = append(b.checks, c) }

func (b *builder) when(t time.Time) string { return Clock(t, b.f.Now, b.loc) }

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

func link(label, href string) ReadinessFix {
	return ReadinessFix{Kind: "link", Label: label, Href: href}
}

func apply(action, label, tone string, params map[string]string) ReadinessFix {
	return ReadinessFix{Kind: "apply", Action: action, Label: label, Tone: tone, Params: params}
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

func (b *builder) reputation() {
	r := b.f.Reputation
	domain := firstNonEmpty(b.f.Auth.Domain, b.f.Campaign.SenderDomain)
	viaName := "HostRepute"
	if r.Via == "builtin" {
		viaName = "the built-in checks"
	}
	checked := ""
	if r.CheckedAt != nil {
		checked = " · checked " + b.when(*r.CheckedAt)
	}
	unknownNote := "not checked"
	if r.Outage != "" {
		unknownNote = "timed out"
	}
	if domain != "" {
		key := "reputation.domain_blocklists"
		var listed []ListingFacts
		for _, l := range r.Listings {
			if l.Type == "domain" && strings.EqualFold(l.Asset, domain) {
				listed = append(listed, l)
			}
		}
		switch {
		case r.Status == Unknown:
			b.add(ReadinessCheck{Key: key, Group: GroupReputation, Status: Unknown,
				Title:     fmt.Sprintf("Domain blocklists for %s couldn't be checked", domain),
				FlatTitle: fmt.Sprintf("Sender reputation · domain blocklists (%d sources via %s)", r.Sources, viaName),
				Detail:    firstNonEmpty(r.Reason, r.Outage), Note: unknownNote, Label: "Domain blocklists",
				Fixes: []ReadinessFix{link("Open deliverability", b.href("/deliverability"))}})
		case len(listed) == 0:
			b.add(ReadinessCheck{Key: key, Group: GroupReputation, Status: Pass,
				Title:  fmt.Sprintf("%s is clean on %d of %d domain blocklists", domain, r.Sources, r.Sources),
				Detail: b.sourcesLine("domain") + checked, Label: "Domain blocklists",
				Short: fmt.Sprintf("%s clean on %d of %d blocklists", domain, r.Sources, r.Sources)})
		default:
			status := Warn
			names := []string{}
			for _, l := range listed {
				if l.Impact == "high" {
					status = Block
				}
				names = append(names, l.Source)
			}
			b.add(ReadinessCheck{Key: key, Group: GroupReputation, Status: status,
				Title:  fmt.Sprintf("%s is listed on %d of %d domain blocklists", domain, len(listed), r.Sources),
				Detail: listingSentence(listed[0]) + " " + impactSentence(listed[0].Impact, "domain"), Label: "Domain blocklists",
				Ack:   fmt.Sprintf("I understand %s is listed on %s.", domain, JoinAnd(names)),
				Fixes: []ReadinessFix{link("Request delisting", b.href("/deliverability"))}})
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
			var listed []ListingFacts
			for _, l := range r.Listings {
				if l.Type == "ip" && l.Asset == ip {
					listed = append(listed, l)
				}
			}
			switch {
			case r.Status == Unknown:
				b.add(ReadinessCheck{Key: key, Group: GroupReputation, Status: Unknown,
					Title:     fmt.Sprintf("IP blocklists for %s couldn't be checked", ip),
					FlatTitle: "Sender reputation · IP blocklists for " + ip,
					Detail:    firstNonEmpty(r.Reason, r.Outage), Note: unknownNote, Label: "IP blocklists"})
			case len(listed) == 0:
				b.add(ReadinessCheck{Key: key, Group: GroupReputation, Status: Pass,
					Title:  fmt.Sprintf("%s IP %s is clean on %d of %d IP blocklists", s.Name, ip, r.Sources, r.Sources),
					Detail: strings.TrimPrefix(b.sourcesLine("ip")+checked, " · "), Label: "IP blocklists",
					Short: fmt.Sprintf("%s clean on %d of %d blocklists", ip, r.Sources, r.Sources)})
			default:
				status := Warn
				for _, l := range listed {
					if l.Impact == "high" {
						status = Block
					}
				}
				detail := listingSentence(listed[0]) + " " + impactSentence(listed[0].Impact, "ip")
				if len(shared) > 0 {
					detail += " " + JoinAnd(shared) + " IPs are a shared pool and are checked at domain level."
				}
				fixes := []ReadinessFix{link("Request delisting", b.href("/deliverability"))}
				if o := b.otherAvailable(s.ProviderID); o != nil {
					fixes = append(fixes, apply(FixRouteOnly, "Send via "+o.Short+" only", "", map[string]string{"providerId": o.ProviderID, "provider": o.Short}))
				}
				b.add(ReadinessCheck{Key: key, Group: GroupReputation, Status: status,
					Title:  fmt.Sprintf("%s IP %s is listed on %d of %d IP blocklists", s.Name, ip, len(listed), r.Sources),
					Detail: detail, Label: "IP blocklists",
					Ack:   fmt.Sprintf("I understand the %s IP is listed on %s, a %s-impact list.", s.Name, shortSource(listed[0].Source), listed[0].Impact),
					Fixes: fixes})
			}
		}
	}
	// Bounce and complaint rates of each provider against its own limits.
	for _, s := range b.availableShares() {
		b.add(b.providerRates(s))
	}
}

// sourcesLine names the first lists of a kind: "Spamhaus DBL, SURBL, URIBL,
// Barracuda and 80 more".
func (b *builder) sourcesLine(kind string) string {
	r := b.f.Reputation
	names := r.DomainSources
	if kind == "ip" {
		names = r.IPSources
	}
	if len(names) == 0 {
		if r.Sources > 0 {
			return fmt.Sprintf("%d sources", r.Sources)
		}
		return ""
	}
	n := min(4, len(names))
	line := strings.Join(names[:n], ", ")
	if rest := r.Sources - n; rest > 0 {
		line += fmt.Sprintf(" and %d more", rest)
	}
	return line
}

var (
	asnRe  = regexp.MustCompile(`\((AS\d+)\)`)
	cidrRe = regexp.MustCompile(`(/\d{1,2})\b`)
)

func listingSentence(l ListingFacts) string {
	d := strings.TrimSpace(l.Detail)
	switch {
	case cidrRe.MatchString(d):
		return l.Source + " lists the whole " + cidrRe.FindStringSubmatch(d)[1] + " of the hosting network."
	case strings.Contains(strings.ToLower(d), "network-range"):
		s := l.Source + " lists the whole hosting network"
		if m := asnRe.FindStringSubmatch(d); m != nil {
			s += " (" + m[1] + ")"
		}
		return s + "."
	case d != "":
		return l.Source + ": " + sentence(d)
	}
	return "Listed on " + l.Source + "."
}

func impactSentence(impact, kind string) string {
	switch impact {
	case "high":
		if kind == "domain" {
			return "Major mailbox providers use it; mail from this domain is likely to be filtered."
		}
		return "Major mailbox providers use it; mail from this IP is likely to be rejected."
	case "medium":
		return "Some mailbox providers use it, so expect deferrals."
	}
	return "Few mailbox providers use it, so impact is low."
}

// shortSource abbreviates "UCEPROTECT Level 3" to "UCEPROTECT L3".
func shortSource(s string) string { return strings.Replace(s, " Level ", " L", 1) }

func limitPhrase(s ShareFacts) string {
	l := s.Limits
	switch {
	case l.Bounce.Review > 0 && l.Bounce.Label != "":
		return l.Bounce.Label + " review thresholds"
	case l.Bounce.Label == "warm-up guard":
		return "the warm-up guard"
	case l.Bounce.Label != "":
		return l.Bounce.Label + "'s limits"
	}
	return "the provider's limits"
}

func statusWord(status string) string {
	switch status {
	case state.Healthy:
		return "Healthy"
	case state.WarmingUp:
		return "Warming up"
	case state.Degraded:
		return "Degraded"
	case state.Down:
		return "Down"
	case state.Sandbox:
		return "Sandbox"
	case state.Paused:
		return "Paused"
	}
	if status == "" {
		return "Healthy"
	}
	return capitalize(strings.ReplaceAll(status, "_", " "))
}

func (b *builder) providerRates(s ShareFacts) ReadinessCheck {
	key := "reputation.provider." + s.ProviderID
	c := ReadinessCheck{Key: key, Group: GroupReputation, Status: Pass, Label: s.Short + " bounce and complaint rates"}
	l := s.Limits
	detail := ""
	if s.WarmupDays > 0 {
		capText := "today's cap"
		if !s.WarmupToday {
			capText = "the cap on the send day"
		}
		detail = fmt.Sprintf("Day %d of %d · %s %s messages, this send uses %s", s.WarmupDay, s.WarmupDays, capText, Count(s.WarmupCap), Count(s.Recipients))
	} else {
		standing := statusWord(s.Status)
		if strings.Contains(strings.ToLower(s.Account), "no enforcement") {
			standing += ", no enforcement"
		}
		window := "trailing 7 days"
		if s.ProviderReported {
			window = "reported by " + s.Name + ", 7 days"
		}
		detail = "Account status: " + standing + " · " + window
	}
	c.Detail = detail
	if s.Bounce == nil && s.Complaint == nil {
		c.Title = "No mail through " + s.Name + " in the last 7 days"
		c.Detail = "Bounce and complaint rates appear after the first sends. " + detail
		c.Short = s.Short + ": no recent mail"
		return c
	}
	bounce, complaint := 0.0, 0.0
	if s.Bounce != nil {
		bounce = *s.Bounce
	}
	if s.Complaint != nil {
		complaint = *s.Complaint
	}
	limits := fmt.Sprintf("(%s and %s)", Pct(l.Bounce.Limit), Pct(l.Complaint.Limit))
	phrase := "inside " + limitPhrase(s) + " " + limits
	switch {
	case (l.Bounce.Pause > 0 && bounce >= l.Bounce.Pause) || (l.Complaint.Pause > 0 && complaint >= l.Complaint.Pause):
		c.Status = Block
		phrase = fmt.Sprintf("at %s pause level (%s and %s)", strings.TrimSuffix(limitPhrase(s), " review thresholds"), Pct(l.Bounce.Pause), Pct(l.Complaint.Pause))
		c.Fixes = []ReadinessFix{link("Open deliverability", b.href("/deliverability"))}
	case bounce >= l.Bounce.Limit || complaint >= l.Complaint.Limit:
		c.Status = Warn
		phrase = "over " + limitPhrase(s) + " " + limits
		c.Fixes = []ReadinessFix{link("Open deliverability", b.href("/deliverability"))}
	case bounce >= 0.8*l.Bounce.Limit || complaint >= 0.8*l.Complaint.Limit:
		c.Status = Warn
		phrase = "close to " + limitPhrase(s) + " " + limits
		c.Fixes = []ReadinessFix{link("Open deliverability", b.href("/deliverability"))}
	}
	c.Title = fmt.Sprintf("%s bounce %s and complaint %s, %s", s.Name, Pct(bounce), Pct(complaint), phrase)
	c.Short = fmt.Sprintf("%s bounce %s, complaints %s", s.Short, Pct(bounce), Pct(complaint))
	if c.Status == Warn {
		c.Ack = fmt.Sprintf("I understand %s's bounce or complaint rate is close to its limit.", s.Name)
	}
	// A warming provider sends at most its day's cap.
	if c.Status == Pass && s.WarmupCap > 0 && s.Recipients > s.WarmupCap {
		c.Status = Warn
		c.Title = fmt.Sprintf("%s can send %s on the send day; this send routes %s to it", s.Name, Count(s.WarmupCap), Count(s.Recipients))
		c.Detail = fmt.Sprintf("The rest waits for the next day's cap, so the send takes longer. Day %d of %d of the warm-up.", s.WarmupDay, s.WarmupDays)
		c.Ack = fmt.Sprintf("I understand part of %s's share waits for the next day's warm-up cap.", s.Name)
		if o := b.otherAvailable(s.ProviderID); o != nil {
			c.Fixes = []ReadinessFix{apply(FixRouteOnly, "Send via "+o.Short+" only", "", map[string]string{"providerId": o.ProviderID, "provider": o.Short})}
		}
	}
	return c
}

func (b *builder) authentication() {
	a := b.f.Auth
	domain := firstNonEmpty(a.Domain, b.f.Campaign.SenderDomain)
	if domain == "" {
		b.add(ReadinessCheck{Key: "auth.sender", Group: GroupAuthentication, Status: Block,
			Title: "The campaign has no sender", Detail: "Pick a From address on a verified sending domain in the composer.",
			Fixes: []ReadinessFix{link("Open in editor", b.editor(nil, ""))}})
		return
	}
	if !a.Checked {
		b.add(ReadinessCheck{Key: "auth.unchecked", Group: GroupAuthentication, Status: Warn,
			Title:  "Authentication for " + domain + " hasn't been checked yet",
			Detail: "Deliverability checks SPF, DKIM, DMARC and MAIL FROM every few hours. Run the checks to verify the records before sending.",
			Ack:    "I understand SPF, DKIM and DMARC haven't been verified for " + domain + ".",
			Fixes:  []ReadinessFix{link("Open deliverability", b.href("/deliverability"))}})
		return
	}
	senders := b.href("/settings/senders")
	if bad := failing(a.SPF); len(bad) == 0 && len(a.SPF) > 0 {
		b.add(ReadinessCheck{Key: "auth.spf", Group: GroupAuthentication, Status: Pass,
			Title: "SPF authorizes " + providersPhrase(shorts(a.SPF), "sending providers"), Detail: a.SPFRecord, Mono: a.SPFRecord != "", Label: "SPF", Short: "SPF"})
	} else if len(a.SPF) == 0 && a.SPFRecord == "" {
		b.add(ReadinessCheck{Key: "auth.spf", Group: GroupAuthentication, Status: Block, Title: "No SPF record on " + domain,
			Detail: "Mailbox providers can't tell which servers may send for the domain. Publish the SPF record shown in Senders & domains.", Label: "SPF",
			Fixes: []ReadinessFix{link("Open senders & domains", senders)}})
	} else {
		b.add(ReadinessCheck{Key: "auth.spf", Group: GroupAuthentication, Status: Block, Title: "SPF doesn't authorize " + JoinAnd(bad),
			Detail: strings.TrimSpace(a.SPFRecord + "  " + firstFailDetail(a.SPF)), Label: "SPF",
			Fixes: []ReadinessFix{link("Open senders & domains", senders)}})
	}
	if bad := failing(a.DKIM); len(bad) == 0 && len(a.DKIM) > 0 {
		bits := 0
		parts := []string{}
		for _, d := range a.DKIM {
			if d.KeyBits > 0 && (bits == 0 || d.KeyBits < bits) {
				bits = d.KeyBits
			}
			parts = append(parts, d.Short+": "+d.Detail)
		}
		title := "DKIM signing on " + providersPhrase(shorts(a.DKIM), "providers")
		status := Pass
		if bits > 0 {
			title += fmt.Sprintf(", %d-bit keys", bits)
			if bits < 1024 {
				status = Warn
				title = fmt.Sprintf("DKIM keys are %d-bit; mailbox providers expect 1024 or more", bits)
			}
		}
		c := ReadinessCheck{Key: "auth.dkim", Group: GroupAuthentication, Status: status, Title: title, Detail: strings.Join(parts, " · "), Label: "DKIM", Short: "DKIM"}
		if status == Warn {
			c.Ack = "I understand the DKIM keys are shorter than recommended."
			c.Fixes = []ReadinessFix{link("Open senders & domains", senders)}
		}
		b.add(c)
	} else {
		b.add(ReadinessCheck{Key: "auth.dkim", Group: GroupAuthentication, Status: Block, Title: "DKIM isn't verified for " + JoinAnd(orAll(bad, shorts(a.DKIM))),
			Detail: firstNonEmpty(firstFailDetail(a.DKIM), "Without DKIM, Gmail and Yahoo reject bulk mail."), Label: "DKIM",
			Fixes: []ReadinessFix{link("Open senders & domains", senders)}})
	}
	switch a.DMARCStatus {
	case "pass":
		title := "DMARC p=" + firstNonEmpty(a.DMARCPolicy, "none")
		status := Pass
		if a.DMARCAligned != nil {
			title += ", " + Pct(math.Round(*a.DMARCAligned*10)/10) + " of mail aligned over the last 7 days"
			if *a.DMARCAligned < 90 {
				status = Warn
				title = fmt.Sprintf("Only %s of mail aligned with DMARC over the last 7 days", Pct(math.Round(*a.DMARCAligned*10)/10))
			}
		}
		c := ReadinessCheck{Key: "auth.dmarc", Group: GroupAuthentication, Status: status, Title: title, Detail: a.DMARCRecord, Mono: a.DMARCRecord != "", Label: "DMARC",
			Short: "DMARC p=" + firstNonEmpty(a.DMARCPolicy, "none")}
		if status == Warn {
			c.Ack = "I understand some mail isn't aligned with DMARC."
			c.Fixes = []ReadinessFix{link("Open deliverability", b.href("/deliverability"))}
		}
		b.add(c)
	case "missing", "":
		b.add(ReadinessCheck{Key: "auth.dmarc", Group: GroupAuthentication, Status: Block, Title: "No DMARC record for " + domain,
			Detail: "Gmail and Yahoo require DMARC from bulk senders. Publish at least v=DMARC1; p=none.", Label: "DMARC",
			Fixes: []ReadinessFix{link("Open senders & domains", senders)}})
	default:
		b.add(ReadinessCheck{Key: "auth.dmarc", Group: GroupAuthentication, Status: Block, Title: "The DMARC record of " + domain + " is invalid",
			Detail: a.DMARCRecord, Mono: a.DMARCRecord != "", Label: "DMARC", Fixes: []ReadinessFix{link("Open senders & domains", senders)}})
	}
	if bad := failing(a.MailFrom); len(bad) == 0 && len(a.MailFrom) > 0 {
		parts := []string{}
		for _, m := range a.MailFrom {
			parts = append(parts, m.Domain+" ("+m.Short+")")
		}
		title := "Custom MAIL FROM domains verified"
		if len(a.MailFrom) == 1 {
			title = "Custom MAIL FROM domain verified"
		}
		b.add(ReadinessCheck{Key: "auth.mail_from", Group: GroupAuthentication, Status: Pass, Title: title, Detail: strings.Join(parts, " · "), Label: "MAIL FROM", Short: "custom MAIL FROM"})
	} else if len(a.MailFrom) > 0 {
		verb := "uses"
		if len(bad) > 1 {
			verb = "use"
		}
		b.add(ReadinessCheck{Key: "auth.mail_from", Group: GroupAuthentication, Status: Warn,
			Title:  fmt.Sprintf("%s %s the default MAIL FROM, so SPF doesn't align with %s", JoinAnd(bad), verb, domain),
			Detail: "DMARC still passes through DKIM. A custom MAIL FROM (return path) on your domain aligns SPF too.", Label: "MAIL FROM",
			Ack: "I understand SPF doesn't align for " + JoinAnd(bad) + ".", Fixes: []ReadinessFix{link("Open senders & domains", senders)}})
	}
	if a.BIMIStatus == "pass" {
		b.add(ReadinessCheck{Key: "auth.bimi", Group: GroupAuthentication, Status: Pass, Title: "BIMI record found", Detail: firstNonEmpty(a.BIMILogo, "Your logo shows in Gmail, Yahoo and Apple Mail."), Label: "BIMI", Short: "BIMI"})
	} else {
		b.add(ReadinessCheck{Key: "auth.bimi", Group: GroupAuthentication, Status: Pass, Title: "BIMI not required",
			Detail: "Optional. No record found; only affects logo display in Gmail, Yahoo and Apple Mail.", Label: "BIMI",
			Fixes: []ReadinessFix{link("Set up BIMI", b.href("/deliverability"))}})
	}
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

func orAll(bad, all []string) []string {
	if len(bad) > 0 {
		return bad
	}
	if len(all) > 0 {
		return all
	}
	return []string{"the sending providers"}
}

func firstFailDetail(list []ProviderAuthFacts) string {
	for _, p := range list {
		if !p.OK && p.Detail != "" {
			return sentence(p.Short + ": " + p.Detail)
		}
	}
	return ""
}

func (b *builder) content() {
	ct := b.f.Content
	if !ct.Checked {
		b.add(ReadinessCheck{Key: "content.unchecked", Group: GroupContent, Status: Unknown, Title: "The content couldn't be checked",
			Detail: "The MJML doesn't compile or the check failed. Open the campaign in the editor and save it again.", Note: "not checked",
			Fixes: []ReadinessFix{link("Open in editor", b.editor(nil, ""))}})
		return
	}
	r := ct.Report
	subject := b.f.Campaign.Subject
	shortened := issuesOf(r, "links.shortener")
	if strings.TrimSpace(subject) == "" {
		b.add(ReadinessCheck{Key: "content.spam", Group: GroupContent, Status: Block, Title: "The subject is empty",
			FlatTitle: "Content: the subject is empty", Detail: "Inbox lists show the subject first; write one before sending.", Label: "Subject",
			Fixes: []ReadinessFix{link("Open in editor", b.editor(nil, ""))}})
	} else if len(r.SubjectHeuristics) > 0 {
		words := make([]string, 0, len(r.SubjectHeuristics))
		for _, h := range r.SubjectHeuristics {
			words = append(words, heuristicWord(h))
		}
		n := len(r.SubjectHeuristics)
		title := fmt.Sprintf("Subject line \"%s\" trips %d spam %s", subject, n, Plural(int64(n), "heuristic", "heuristics"))
		detail := capitalize(JoinAnd(words)) + "."
		c := ReadinessCheck{Key: "content.spam", Group: GroupContent, Status: Warn, Title: title, FlatTitle: "Content: " + lowerFirst(title), Label: "Spam triggers",
			Ack: fmt.Sprintf("I understand the subject trips %d spam %s.", n, Plural(int64(n), "heuristic", "heuristics"))}
		if s := contentcheck.SuggestSubject(subject); s != "" && s != subject {
			detail += fmt.Sprintf(" Suggested: \"%s\"", s)
			c.Fixes = []ReadinessFix{apply(FixUseSubject, "Use suggestion", "", map[string]string{"subject": s})}
		}
		c.Detail = detail
		b.add(c)
	} else {
		links := "no shortened links"
		if len(shortened) > 0 {
			links = fmt.Sprintf("%d shortened %s, see Links", len(shortened), Plural(int64(len(shortened)), "link", "links"))
		}
		b.add(ReadinessCheck{Key: "content.spam", Group: GroupContent, Status: Pass, Title: "No spam triggers in subject, preheader or body",
			Detail: fmt.Sprintf("Subject: \"%s\" · no all-caps, no urgency phrases, %s", subject, links), Label: "Spam triggers"})
	}
	share := int(math.Round(r.ImageShare * 100))
	ratio := ReadinessCheck{Key: "content.image_ratio", Group: GroupContent, Status: Pass, Label: "Image-to-text ratio",
		Title:  fmt.Sprintf("Image-to-text ratio %d%%", share),
		Detail: fmt.Sprintf("%d %s, %s words · recommended under 60%%", r.Images, Plural(int64(r.Images), "image", "images"), Count(int64(r.Words))),
		Short:  fmt.Sprintf("image ratio %d%%", share)}
	if share >= 60 {
		ratio.Status = Warn
		ratio.Title = fmt.Sprintf("Image-to-text ratio %d%%, over the recommended 60%%", share)
		ratio.Detail = fmt.Sprintf("%d images, %s words. Spam filters score image-heavy mail higher, and most of it disappears when images are off.", r.Images, Count(int64(r.Words)))
		ratio.Ack = fmt.Sprintf("I understand the email is %d%% image.", share)
		ratio.Fixes = []ReadinessFix{link("Open in editor", b.editor(nil, "deliverability"))}
	}
	b.add(ratio)
	kb := int(math.Round(float64(r.SizeBytes) / 1024))
	cssKB := int(math.Round(float64(ct.InlineCSSBytes) / 1024))
	trackKB := 0
	if r.Links > 0 {
		trackKB = int(math.Max(1, math.Round(float64(r.Links*200)/1024)))
	}
	size := ReadinessCheck{Key: "content.size", Group: GroupContent, Status: Pass, Label: "HTML size",
		Title:  fmt.Sprintf("HTML size %d KB, under Gmail's 102 KB clipping limit", kb),
		Detail: fmt.Sprintf("Inlined CSS %d KB · tracking links add about %d KB", cssKB, trackKB),
		Short:  fmt.Sprintf("%d KB of 102 KB", kb)}
	if r.SizeBytes >= contentcheck.NearClipBytes {
		size.Status = Warn
		size.Title = fmt.Sprintf("HTML size %d KB, close to Gmail's 102 KB clipping limit", kb)
		if r.SizeBytes > contentcheck.ClipBytes {
			size.Title = fmt.Sprintf("HTML size %d KB, over Gmail's 102 KB clipping limit", kb)
		}
		size.Detail = fmt.Sprintf("Gmail clips the message and hides the footer, including the unsubscribe link. Inlined CSS %d KB · tracking links add about %d KB.", cssKB, trackKB)
		size.Ack = fmt.Sprintf("I understand the email is %d KB, near Gmail's clipping limit.", kb)
		for _, is := range issuesOf(r, "content.size") {
			for _, fx := range is.Fixes {
				if fx.Kind == "minify" {
					size.Fixes = append(size.Fixes, apply(FixMinify, fx.Label, "", nil))
				}
			}
		}
		size.Fixes = append(size.Fixes, link("Open in editor", b.editor(nil, "deliverability")))
	}
	// With conditional content no one gets the whole email: Conditional
	// content checks the size of each version (design expansion, X11).
	if b.conditionalVersions() {
		size = b.conditionalSize(size)
	}
	b.add(size)
	// Alt text is an accessibility check when the Accessibility group runs.
	alt := ReadinessCheck{Key: "content.alt_text", Group: GroupContent, Status: Pass, Label: "Alt text"}
	switch {
	case b.f.Accessibility.Ran:
	case r.Images == 0:
		alt.Title, alt.Short = "No images, so no alt text needed", "no images"
	case r.ImagesMissingAlt == 0:
		alt.Title = fmt.Sprintf("All %d images have alt text", r.Images)
		switch r.Images {
		case 1:
			alt.Title = "The image has alt text"
		case 2:
			alt.Title = "Both images have alt text"
		}
		alt.Short = fmt.Sprintf("%d of %d images have alt text", r.Images, r.Images)
	default:
		alt.Status = Warn
		alt.Title = fmt.Sprintf("%d of %d images have no alt text", r.ImagesMissingAlt, r.Images)
		first := issuesOf(r, "content.alt_text")
		alt.Detail = "Images are off by default in Outlook and many corporate clients; alt text is what shows."
		var path []int
		if len(first) > 0 {
			alt.Detail = sentence(first[0].Title) + " " + alt.Detail
			path = first[0].Path
		}
		alt.Ack = fmt.Sprintf("I understand %d %s no alt text.", r.ImagesMissingAlt, Plural(int64(r.ImagesMissingAlt), "image has", "images have"))
		alt.Fixes = []ReadinessFix{link("Open in editor", b.editor(path, "deliverability"))}
	}
	if !b.f.Accessibility.Ran {
		b.add(alt)
	}
	if strings.TrimSpace(r.PlainText) != "" {
		ptKB := int(math.Max(1, math.Round(float64(len(r.PlainText))/1024)))
		b.add(ReadinessCheck{Key: "content.plain_text", Group: GroupContent, Status: Pass, Title: "Plain-text version generated",
			Detail: fmt.Sprintf("multipart/alternative · %d KB", ptKB), Label: "Plain text", Short: "plain-text generated",
			Fixes: []ReadinessFix{link("Preview", b.href("/campaigns/"+b.f.Campaign.ID+"/edit?view=preview"))}})
	} else {
		b.add(ReadinessCheck{Key: "content.plain_text", Group: GroupContent, Status: Warn, Title: "The plain-text version is empty",
			Detail: "The email has no text, so the text/plain part is blank. Add a sentence that says what the images show.", Label: "Plain text",
			Ack: "I understand the plain-text version is empty.", Fixes: []ReadinessFix{link("Open in editor", b.editor(nil, ""))}})
	}
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

func heuristicWord(h string) string {
	switch h {
	case "urgency":
		return "urgency phrasing"
	}
	return h
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
	respond := ReadinessCheck{Key: "links.respond", Group: GroupLinks, Status: Pass, Label: "Links respond"}
	switch {
	case total == 0:
		respond.Title, respond.Short = "The email has no links", "no links"
	case len(probed) == 0:
		respond.Title = fmt.Sprintf("%d %s found", total, Plural(int64(total), "link", "links"))
		respond.Detail = "Links weren't requested in this run."
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
			n := len(blockIssues)
			respond.Title = fmt.Sprintf("%d of %d links respond; the blocklisted %s counted above", len(considered), total, Plural(int64(n), "one is", "ones are"))
			respond.Short = respond.Title
		} else {
			respond.Title = fmt.Sprintf("All %d links respond", total)
			if total == 1 {
				respond.Title = "The link responds"
			}
			if all200 {
				respond.Title += " (HTTP 200)"
			}
			respond.Short = fmt.Sprintf("all %d links respond", total)
		}
		if slow.Millis > 0 {
			respond.Detail = fmt.Sprintf("Slowest: %s at %d ms", displayURL(slow.URL), slow.Millis)
		}
	default:
		respond.Status = Warn
		verb := "don't respond"
		if len(broken) == 1 {
			verb = "doesn't respond"
		}
		respond.Title = fmt.Sprintf("%d of %d %s %s", len(broken), total, Plural(int64(total), "link", "links"), verb)
		parts := []string{}
		for i, l := range broken {
			if i == 3 {
				parts = append(parts, fmt.Sprintf("and %d more", len(broken)-3))
				break
			}
			what := fmt.Sprintf("returns %d", l.Status)
			if l.Status == 0 {
				what = "doesn't respond"
			}
			parts = append(parts, displayURL(l.URL)+" "+what)
		}
		respond.Detail = strings.Join(parts, " · ")
		respond.Ack = fmt.Sprintf("I understand %d %s %s.", len(broken), Plural(int64(len(broken)), "link", "links"), verb)
		respond.Fixes = []ReadinessFix{link("Open in editor", b.editor(broken[0].Path, "deliverability"))}
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
		line := strings.Join(ct.URIBlocklists, ", ")
		if len(domains) > 0 {
			shown := domains
			more := ""
			if len(domains) > 4 {
				shown = domains[:4]
				more = fmt.Sprintf(" and %d more", len(domains)-4)
			}
			if line != "" {
				line += " · "
			}
			line += strings.Join(shown, ", ") + more
		}
		b.add(ReadinessCheck{Key: "links.blocklists", Group: GroupLinks, Status: Pass, Title: "No link domains on URI blocklists", Detail: line, Label: "URI blocklists",
			Short: "no blocklisted link domains"})
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
			used := "Used by a link."
			if text != "" {
				used = fmt.Sprintf("Used by the \"%s\" %s.", text, kind)
			}
			c := ReadinessCheck{Key: "links.blocklist." + host, Group: GroupLinks, Status: Block,
				Title:  fmt.Sprintf("%s is listed on %s", host, lists),
				Detail: used + " Mail that links to a blocklisted domain is filtered by Gmail, Outlook and most corporate gateways regardless of your own reputation.",
				Label:  "URI blocklists"}
			for _, fx := range is.Fixes {
				if fx.Kind == "replace_link" && fx.Value != "" {
					c.Fixes = append(c.Fixes, apply(FixReplaceLink, fx.Label, "danger", map[string]string{"from": is.URL, "to": fx.Value}))
				}
			}
			c.Fixes = append(c.Fixes, link("Edit link", b.editor(is.Path, "deliverability")))
			b.add(c)
		}
	}
	if short := issuesOf(r, "links.shortener"); len(short) > 0 {
		first := short[0]
		title := fmt.Sprintf("1 link redirects through %s", displayURL(first.URL))
		if len(short) > 1 {
			title = fmt.Sprintf("%d links redirect through URL shorteners", len(short))
		}
		c := ReadinessCheck{Key: "links.shortener", Group: GroupLinks, Status: Warn, Title: title,
			Detail: "Shorteners are often on URI blocklists and hide the destination.", Label: "URL shortener",
			Ack: fmt.Sprintf("I understand %d %s through a URL shortener.", len(short), Plural(int64(len(short)), "link goes", "links go"))}
		final := ""
		for _, l := range r.LinkList {
			if l.URL == first.URL && l.Final != "" {
				final = l.Final
			}
		}
		if final != "" && len(short) == 1 {
			c.Fixes = []ReadinessFix{apply(FixReplaceLink, "Replace with destination URL", "", map[string]string{"from": first.URL, "to": final})}
		}
		c.Fixes = append(c.Fixes, link("Edit link", b.editor(first.Path, "deliverability")))
		b.add(c)
	}
	if empty := issuesOf(r, "links.empty"); len(empty) > 0 {
		b.add(ReadinessCheck{Key: "links.empty", Group: GroupLinks, Status: Warn, Title: empty[0].Title, Detail: empty[0].Detail, Label: "Empty link",
			Ack: "I understand a button has no link.", Fixes: []ReadinessFix{link("Edit link", b.editor(empty[0].Path, "deliverability"))}})
	}
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

func linkTextFor(r contentcheck.ContentCheckReport, href string) (string, string) {
	for _, l := range r.LinkList {
		if l.URL == href {
			kind := "link"
			switch l.Kind {
			case "button":
				kind = "button"
			case "image":
				kind = "image"
			case "nav":
				kind = "menu link"
			case "social":
				kind = "social link"
			}
			return strings.TrimSpace(l.Text), kind
		}
	}
	return "", "link"
}

func (b *builder) compliance() {
	cf := b.f.Compliance
	if cf.HTTPS {
		b.add(ReadinessCheck{Key: "compliance.list_unsubscribe", Group: GroupCompliance, Status: Pass, Title: "One-click unsubscribe headers present",
			Detail: fmt.Sprintf("List-Unsubscribe: <%s> · List-Unsubscribe-Post: List-Unsubscribe=One-Click", cf.UnsubscribeURL), Mono: true,
			Label: "One-click unsubscribe", Short: "one-click unsubscribe headers"})
	} else {
		b.add(ReadinessCheck{Key: "compliance.list_unsubscribe", Group: GroupCompliance, Status: Warn, Title: "One-click unsubscribe needs an HTTPS address",
			Detail: "Unsubscribe links point at " + cf.UnsubscribeURL + ". Gmail and Yahoo only honour one-click unsubscribe over HTTPS; set the instance's public URL to an https:// address.",
			Label:  "One-click unsubscribe", Ack: "I understand one-click unsubscribe won't work until the public URL uses HTTPS."})
	}
	ct := b.f.Content
	footerBlocked := false
	if ct.Checked {
		if f := issuesOf(ct.Report, "compliance.footer"); len(f) > 0 {
			footerBlocked = true
			title := "The email body has no unsubscribe link or postal address"
			switch f[0].Title {
			case "No postal address in the footer":
				title = "The footer has no postal address"
			case "No unsubscribe link in the body":
				title = "The email body has no unsubscribe link"
			}
			detail := "ReputeCast adds List-Unsubscribe headers automatically, but Gmail and Yahoo bulk-sender rules and CAN-SPAM also require a visible link and address in the body."
			if ct.FooterRemovedAt != nil {
				detail = "The footer block was deleted in the editor at " + b.when(*ct.FooterRemovedAt) + ". " + detail
			}
			b.add(ReadinessCheck{Key: "compliance.footer", Group: GroupCompliance, Status: Block, Title: title, FlatTitle: "Compliance: " + lowerFirst(title), Detail: detail,
				Label: "Footer", Fixes: []ReadinessFix{apply(FixInsertFooter, "Insert footer block", "danger", nil), link("Open in editor", b.editor(nil, "deliverability"))}})
		} else {
			b.add(ReadinessCheck{Key: "compliance.footer", Group: GroupCompliance, Status: Pass, Title: "Footer has an unsubscribe link and a postal address",
				Detail: firstNonEmpty(ct.Address, "Found in the footer"), Label: "Footer", Short: "footer address"})
		}
	}
	c := ReadinessCheck{Key: "compliance.bulk_sender", Group: GroupCompliance, Status: Pass, Title: "Bulk-sender requirements met", Label: "Bulk-sender rules", Short: "bulk-sender rules"}
	parts := []string{}
	if b.f.Auth.DMARCStatus == "pass" {
		parts = append(parts, "DMARC aligned")
	} else {
		c.Status = Warn
		c.Title = "Gmail and Yahoo bulk-sender rules need DMARC"
		c.Ack = "I understand the domain doesn't meet Gmail and Yahoo's DMARC rule yet."
	}
	if cf.HTTPS {
		parts = append(parts, "one-click unsubscribe")
	}
	if cf.ComplaintRate != nil {
		cr := *cf.ComplaintRate
		switch {
		case cr >= 0.3:
			c.Status = Block
			c.Title = fmt.Sprintf("Complaint rate %s is over Gmail and Yahoo's 0.3%% limit", Pct(cr))
			c.Fixes = []ReadinessFix{link("Open deliverability", b.href("/deliverability"))}
		case cr >= 0.1 && c.Status == Pass:
			c.Status = Warn
			c.Title = fmt.Sprintf("Complaint rate %s, over Gmail's recommended 0.1%%", Pct(cr))
			c.Ack = "I understand the complaint rate is above Gmail's recommended 0.1%."
		}
		parts = append(parts, fmt.Sprintf("complaint rate %s, under the 0.3%% limit", Pct(cr)))
		if cr >= 0.3 {
			parts[len(parts)-1] = fmt.Sprintf("complaint rate %s", Pct(cr))
		}
	} else {
		parts = append(parts, "complaint rate not measured yet")
	}
	parts = append(parts, "unsubscribes processed within 2 days")
	c.Detail = strings.Join(parts, " · ")
	if footerBlocked && c.Status == Pass {
		// The rules also want a visible unsubscribe link and address in the
		// body: the footer block says they aren't met, and its fix meets them.
		return
	}
	b.add(c)
}

func (b *builder) audience() {
	a := b.f.Audience
	if a.Empty {
		b.add(ReadinessCheck{Key: "audience.empty", Group: GroupAudience, Status: Block, Title: "The campaign has no audience",
			FlatTitle: "Audience: the campaign has no audience", Detail: "Pick the lists or segments that receive it in the composer.", Label: "Audience",
			Fixes: []ReadinessFix{link("Choose an audience", b.editor(nil, ""))}})
		return
	}
	if a.Final == 0 {
		b.add(ReadinessCheck{Key: "audience.empty", Group: GroupAudience, Status: Block, Title: "Nobody would receive this campaign",
			FlatTitle: "Audience: nobody would receive this campaign", Detail: "Every contact of the audience is suppressed, a role address or excluded.", Label: "Audience",
			Fixes: []ReadinessFix{link("Choose an audience", b.editor(nil, ""))}})
	}
	days := a.UnengagedDays
	if days <= 0 {
		days = 180
	}
	share := 0.0
	if a.Final > 0 {
		share = float64(a.Unengaged) / float64(a.Final) * 100
	}
	un := ReadinessCheck{Key: "audience.unengaged", Group: GroupAudience, Status: Pass}
	switch {
	case a.Unengaged == 0 && a.UnengagedExcluded:
		un.Title = fmt.Sprintf("%s unengaged contacts excluded", Count(a.SunsetExcluded))
		un.Detail = fmt.Sprintf("No open or click in %d days; the sunset tier is excluded from this send", days)
		un.Short = "unengaged excluded"
	case a.Unengaged > 0 && a.SunsetOverride:
		// The campaign overrides the sunset policy (SendReadinessBlocked).
		un.Status = Warn
		un.Title = fmt.Sprintf("%s Sunset-tier contacts are included", Count(a.Unengaged))
		un.FlatTitle = fmt.Sprintf("Audience: %s Sunset-tier contacts included", Count(a.Unengaged))
		un.Detail = fmt.Sprintf("The sunset policy would normally exclude them; this campaign overrides it. Excluding them changes the audience to %s.", Count(a.Final-a.Unengaged))
		un.Evidence = fmt.Sprintf("%s of the audience (%s contacts) is in the Sunset tier; this campaign overrides the sunset policy.", Pct1(share), Count(a.Unengaged))
		un.Ack = fmt.Sprintf("I understand %s Sunset-tier contacts are included.", Count(a.Unengaged))
		un.Fixes = []ReadinessFix{apply(FixExcludeSunset, fmt.Sprintf("Exclude %s", Count(a.Unengaged)), "", map[string]string{"count": fmt.Sprint(a.Unengaged)})}
	case a.Unengaged > 0 && (share >= UnengagedWarnShare || a.Unengaged >= UnengagedWarnCount):
		un.Status = Warn
		un.Title = fmt.Sprintf("%s unengaged contacts (no open in %d days) are in this audience", Count(a.Unengaged), days)
		un.FlatTitle = fmt.Sprintf("Audience: %s unengaged contacts included", Count(a.Unengaged))
		un.Detail = fmt.Sprintf("Sending to them raises the chance of spam-trap hits and lowers Gmail engagement signals. Excluding them changes the audience to %s.", Count(a.Final-a.Unengaged))
		un.Evidence = fmt.Sprintf("%s of the audience (%s contacts) hasn't engaged in more than %d days.", Pct1(share), Count(a.Unengaged), days)
		un.Ack = fmt.Sprintf("I understand %s unengaged contacts are included.", Count(a.Unengaged))
		un.Fixes = []ReadinessFix{
			apply(FixExcludeSunset, fmt.Sprintf("Exclude %s unengaged", Count(a.Unengaged)), "primary", map[string]string{"count": fmt.Sprint(a.Unengaged)}),
			link("Review them", b.href("/audience?tier=sunset")),
		}
	case a.Unengaged > 0:
		un.Title = fmt.Sprintf("%s unengaged contacts, %s of the audience", Count(a.Unengaged), Pct1(share))
		un.Detail = fmt.Sprintf("No open in %d days. Under 1%%, and the sunset policy stops Broadcasts to them after 30 days.", days)
		un.Short = fmt.Sprintf("%s unengaged", Count(a.Unengaged))
	default:
		un.Title = "No unengaged contacts in this audience"
		un.Detail = fmt.Sprintf("Everyone opened or clicked in the last %d days", days)
		un.Short = "0 unengaged"
	}
	// No label: "Audience quality" names it in compact tables (ApprovalReview).
	b.add(un)
	if a.RecentlyBounced == 0 {
		b.add(ReadinessCheck{Key: "audience.bounced", Group: GroupAudience, Status: Pass, Title: "No contacts that bounced in the last 30 days",
			Detail: fmt.Sprintf("%s contacts on the shared suppression list are excluded", Count(a.Suppressed)), Label: "Recently bounced"})
	} else {
		b.add(ReadinessCheck{Key: "audience.bounced", Group: GroupAudience, Status: Warn,
			Title:  fmt.Sprintf("%s %s bounced in the last 30 days", Count(a.RecentlyBounced), Plural(a.RecentlyBounced, "contact", "contacts")),
			Detail: "Soft bounces that aren't suppressed yet. Another hard bounce suppresses them; repeated bounces hurt the route's reputation.",
			Label:  "Recently bounced", Ack: fmt.Sprintf("I understand %s recently bounced contacts are included.", Count(a.RecentlyBounced)),
			Fixes: []ReadinessFix{link("Review them", b.href("/audience?status=bounced"))}})
	}
	role := ReadinessCheck{Key: "audience.role", Group: GroupAudience, Status: Pass, Label: "Role addresses",
		Title:  fmt.Sprintf("%s role addresses excluded automatically", Count(a.Role)),
		Detail: "info@, support@, sales@, admin@ and similar"}
	if a.Role == 0 {
		role.Title = "No role addresses in this audience"
	}
	if a.Role == 1 {
		role.Title = "1 role address excluded automatically"
	}
	b.add(role)
	if a.SegmentsSkipped {
		b.add(ReadinessCheck{Key: "audience.segments", Group: GroupAudience, Status: Warn, Title: "Segments in the audience couldn't be counted",
			Detail: "The counts cover the lists only. Open the segment to recount it.", Label: "Segments",
			Ack: "I understand the segment part of the audience wasn't counted."})
	}
}

func streamLabel(s string) string {
	switch s {
	case "automation":
		return "Automations"
	case "system":
		return "System"
	}
	return "Broadcasts"
}

func policyWords(p string) string {
	switch p {
	case "automatic":
		return "automatic"
	case "warm_only":
		return "automatic to a warmed-up provider"
	}
	return "pause and ask"
}

func (b *builder) shareStatusLabel(s ShareFacts) string {
	switch {
	case !s.Available && s.Status == state.Down && s.DownSince != nil:
		return "Down since " + b.when(*s.DownSince)
	case s.Status == state.WarmingUp && s.WarmupDays > 0:
		return fmt.Sprintf("Warming up · day %d of %d", s.WarmupDay, s.WarmupDays)
	}
	return statusWord(s.Status)
}

func (b *builder) routing() {
	rt := b.f.Route
	stream := streamLabel(rt.Stream)
	routes := b.href("/providers/routing")
	if rt.NoRoute || len(rt.Shares) == 0 {
		b.add(ReadinessCheck{Key: "routing.health", Group: GroupRouting, Status: Block, Title: stream + " has no sending provider",
			Detail: "Pick a primary provider for " + stream + " in routing.", Label: "Route", Fixes: []ReadinessFix{link("Set up routing", routes)}})
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
		word := "Down"
		switch d.Status {
		case state.Paused:
			word = "paused"
		case state.Degraded:
			word = "not reporting events"
		}
		pct := 0.0
		var held int64
		for _, s := range down {
			pct += s.Percent
			held += s.Recipients
		}
		detail := ""
		if d.DownDetail != "" {
			detail = sentence(d.DownDetail) + " "
		}
		detail += fmt.Sprintf("%s of this send has no healthy provider.", trimNumber(math.Round(pct), 0)+"%")
		c := ReadinessCheck{Key: "routing.health", Group: GroupRouting, Status: Block, Label: "Route health",
			Title: fmt.Sprintf("%s is %s and %s failover is set to %s", d.Name, word, stream, policyWords(rt.Policy))}
		if o := b.otherAvailable(d.ProviderID); o != nil {
			if o.Quota != nil && o.Quota.Max > 0 {
				detail += fmt.Sprintf(" %s is %s with %s of %s quota free.", o.Short, statusWord(o.Status), Count(o.Quota.Max-o.Quota.Used), Count(o.Quota.Max))
			} else {
				detail += fmt.Sprintf(" %s is %s.", o.Short, statusWord(o.Status))
			}
			c.Fixes = append(c.Fixes, apply(FixRouteOnly, "Send 100% via "+o.Short, "danger", map[string]string{"providerId": o.ProviderID, "provider": o.Short}))
		}
		c.Fixes = append(c.Fixes, link("Check "+d.Short, b.href("/providers")))
		c.Detail = detail
		if rt.Policy != "pause_and_ask" && b.otherAvailable(d.ProviderID) != nil {
			c.Status = Warn
			c.Title = fmt.Sprintf("%s is %s; its %s share fails over to another provider", d.Name, word, trimNumber(math.Round(pct), 0)+"%")
			c.Ack = fmt.Sprintf("I understand %s recipients fail over from %s.", Count(held), d.Name)
		}
		b.add(c)
	} else {
		parts := []string{}
		for _, s := range shares {
			label := statusWord(s.Status)
			if s.Status == state.WarmingUp && s.WarmupDays > 0 {
				label = fmt.Sprintf("Warming up, day %d", s.WarmupDay)
			}
			parts = append(parts, fmt.Sprintf("%s (%s) %s", s.Name, label, trimNumber(math.Round(s.Percent), 0)+"%"))
		}
		c := ReadinessCheck{Key: "routing.health", Group: GroupRouting, Status: Pass, Title: "Every route has a healthy provider", Label: "Route health",
			Detail: strings.Join(parts, " · ") + " · failover: " + policyWords(rt.Policy), Short: "every route healthy",
			Fixes: []ReadinessFix{link("Edit route", routes)}}
		if len(shares) == 1 {
			c.Title = "The route has a healthy provider"
		}
		// Quota on the provider's own 24-hour allowance.
		for _, s := range shares {
			if s.Quota != nil && s.Quota.Kind == "rolling_24h" && s.Quota.Max > 0 && s.Recipients > s.Quota.Max-s.Quota.Used {
				c.Status = Warn
				c.Title = fmt.Sprintf("%s has %s of its %s daily quota left; this send needs %s", s.Name, Count(max(s.Quota.Max-s.Quota.Used, 0)), Count(s.Quota.Max), Count(s.Recipients))
				c.Detail = "Sending pauses at the quota and resumes as it frees up, so the send takes longer."
				c.Ack = fmt.Sprintf("I understand %s's quota stretches the send.", s.Name)
				c.Short = ""
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
		parts := []string{}
		for _, s := range sending {
			parts = append(parts, s.Short+": "+s.Terms)
		}
		title := "Both providers' terms allow bulk marketing"
		short := "both providers' terms allow bulk marketing"
		switch {
		case len(sending) == 1:
			title = sending[0].Short + "'s terms allow bulk marketing"
			short = sending[0].Short + " terms allow bulk marketing"
		case len(sending) > 2:
			title = fmt.Sprintf("All %d providers' terms allow bulk marketing", len(sending))
			short = fmt.Sprintf("all %d providers' terms allow bulk marketing", len(sending))
		}
		b.add(ReadinessCheck{Key: "routing.terms", Group: GroupRouting, Status: Pass, Title: title, Detail: strings.Join(parts, " · "), Label: "Bulk-marketing terms", Short: short})
	} else {
		s := notAllowed[0]
		c := ReadinessCheck{Key: "routing.terms", Group: GroupRouting, Status: Block, Label: "Bulk-marketing terms",
			Title:  s.Name + "'s terms don't allow bulk marketing",
			Detail: sentence(capitalize(s.Terms)) + " Route " + stream + " through a provider whose terms allow them.",
			Fixes:  []ReadinessFix{link("Edit route", routes)}}
		if s.Sandbox {
			c.Title = s.Name + " is in the sandbox"
			c.Detail = "It only delivers to verified addresses. Request production access in Sending providers."
			c.Fixes = []ReadinessFix{link("Open sending providers", b.href("/providers"))}
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
		parts := []string{}
		for _, s := range shares {
			parts = append(parts, trackingNote(s))
		}
		b.add(ReadinessCheck{Key: "routing.tracking", Group: GroupRouting, Status: Pass, Title: "Provider-side open and click tracking is off, so nothing is double-tracked",
			Detail: strings.Join(parts, " · "), Label: "Provider-side tracking", Short: "provider-side tracking is off"})
	} else {
		s := double[0]
		what := "adds its own open pixel and rewrites links"
		switch {
		case s.Tracking.OpenPixel && !s.Tracking.LinkRewrite:
			what = "adds its own open pixel"
		case !s.Tracking.OpenPixel && s.Tracking.LinkRewrite:
			what = "rewrites links"
		}
		b.add(ReadinessCheck{Key: "routing.tracking", Group: GroupRouting, Status: Warn, Title: s.Name + " " + what,
			Detail: firstNonEmpty(s.Tracking.Detail, "Opens and clicks would count twice."), Label: "Provider-side tracking",
			Ack:   "I understand " + s.Name + "'s own tracking double-counts opens and clicks.",
			Fixes: []ReadinessFix{link("Turn it off", b.href("/providers"))}})
	}
}

func trackingNote(s ShareFacts) string {
	scope := strings.TrimSpace(s.Tracking.Scope)
	switch {
	case s.Tracking.Mode == "provider":
		return s.Short + ": its own tracking, ReputeCast's off"
	case s.Adapter == "postal" || s.Adapter == "smtp":
		return s.Short + ": link rewriting disabled"
	case scope != "":
		return s.Short + " " + lowerFirst(scope) + ": no tracking"
	}
	return s.Short + ": no tracking"
}

func (b *builder) estimates() ReadinessEstimates {
	a := b.f.Audience
	out := ReadinessEstimates{Throttling: []ReadinessThrottle{}}
	aud := ReadinessAudience{Rows: []ReadinessAudienceRow{}, Final: a.Final, Lists: a.Lists, Suppressed: a.Suppressed, Role: a.Role, Duplicates: a.Duplicates,
		Unengaged: a.Unengaged, UnengagedExcluded: a.UnengagedExcluded}
	if aud.Lists == nil {
		aud.Lists = []string{}
	}
	if !a.Empty {
		aud.Rows = append(aud.Rows, ReadinessAudienceRow{Label: a.BaseLabel, Count: a.Base, Kind: "base"},
			ReadinessAudienceRow{Label: "Suppression list (shared)", Count: a.Suppressed, Kind: "minus"},
			ReadinessAudienceRow{Label: "Role addresses", Count: a.Role, Kind: "minus"})
		if a.Duplicates > 0 {
			aud.Rows = append(aud.Rows, ReadinessAudienceRow{Label: "Duplicates", Count: a.Duplicates, Kind: "minus"})
		}
		switch {
		case a.SunsetOverride:
			aud.Rows = append(aud.Rows, ReadinessAudienceRow{Label: "Sunset-tier override", Count: a.StoppedIncluded, Kind: "plus"})
		case a.Stopped > 0:
			aud.Rows = append(aud.Rows, ReadinessAudienceRow{Label: "Sunset policy (Broadcasts stopped)", Count: a.Stopped, Kind: "minus"})
		}
		if a.UnengagedExcluded {
			aud.Rows = append(aud.Rows, ReadinessAudienceRow{Label: "Unengaged (sunset tier)", Count: a.SunsetExcluded, Kind: "minus"})
		}
	}
	out.Audience = aud

	rt := b.f.Route
	route := ReadinessRoute{Stream: firstNonEmpty(rt.Stream, "broadcast"), StreamLabel: streamLabel(rt.Stream), Policy: firstNonEmpty(rt.Policy, "pause_and_ask"),
		Override: rt.Override, Shares: []ReadinessRouteShare{}}
	pcts := []string{}
	maxMinutes := 0
	for _, s := range rt.Shares {
		rs := ReadinessRouteShare{ProviderID: s.ProviderID, Name: s.Name, Short: s.Short, Adapter: s.Adapter, Label: s.Label, Color: state.Color(s.Adapter),
			Percent: math.Round(s.Percent*10) / 10, Recipients: s.Recipients, Status: s.Status, StatusLabel: b.shareStatusLabel(s), Available: s.Available,
			WarmupDay: s.WarmupDay, WarmupDays: s.WarmupDays, Rate: s.Rate}
		switch s.Pricing.Model {
		case "self_hosted":
			rs.PriceLabel = "self-hosted"
		case "plan":
			rs.PriceLabel = "plan"
		default:
			rs.PriceLabel = Per1000(s.Pricing.Per1000)
			rs.Cost = math.Round(float64(s.Recipients)*s.Pricing.Per1000/1000*100) / 100
		}
		if s.Rate > 0 {
			rs.RateLabel = Rate(s.Rate)
			if s.RateRamp {
				rs.RateLabel += " ramp"
			}
			rs.Minutes = Minutes(float64(s.Recipients) / s.Rate)
		}
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
	route.SplitLabel = route.StreamLabel
	if len(pcts) > 0 {
		route.SplitLabel += " " + strings.Join(pcts, " / ")
	}
	for _, k := range MailboxOrder {
		n := a.Mailboxes[k]
		rate := b.f.MailboxRates[k]
		out.Throttling = append(out.Throttling, ReadinessThrottle{Key: k, Name: MailboxName(k), Recipients: n, Rate: rate})
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
	return out
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
		grp := ReadinessGroup{Key: g, Title: groupTitles[g], Status: Pass}
		var passes []ReadinessCheck
		for _, c := range b.checks {
			if c.Group != g {
				continue
			}
			grp.Total++
			if c.Status == Pass {
				grp.Passed++
				if c.Short != "" {
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
		grp.Caption = b.caption(g, grp)
		grp.Summary = b.summary(g, passes)
		grp.Running = b.running(g)
		if g == GroupAccessibility {
			grp.Note = b.accessibilityNote()
		}
		if g == GroupConditional {
			grp.Note = b.conditionalNote()
		}
		out = append(out, grp)
	}
	return out
}

func (b *builder) caption(g string, grp ReadinessGroup) string {
	switch g {
	case GroupReputation:
		r := b.f.Reputation
		via := "via HostRepute"
		if r.Via == "builtin" {
			via = "via the built-in checks"
		}
		if r.Status == Unknown {
			return fmt.Sprintf("%s · %d sources · not refreshed", via, r.Sources)
		}
		return fmt.Sprintf("%s · %d sources · %d of %d passed", via, r.Sources, grp.Passed, grp.Total)
	case GroupAuthentication:
		domain := firstNonEmpty(b.f.Auth.Domain, b.f.Campaign.SenderDomain)
		names := shortsOf(b.availableShares())
		if len(names) == 0 {
			return domain
		}
		return domain + " · records required by " + JoinAnd(names)
	case GroupContent:
		s := b.f.Campaign.Name
		if n := len(b.f.Content.Versions); n > 0 && b.f.Campaign.CompiledAt != nil {
			return fmt.Sprintf("%s · %d versions compiled from MJML %s", s, n+1, b.when(*b.f.Campaign.CompiledAt))
		}
		if b.f.Campaign.CompiledAt != nil {
			s += " · compiled from MJML " + b.when(*b.f.Campaign.CompiledAt)
		}
		return s
	case GroupLinks:
		r := b.f.Content.Report
		if len(b.f.Content.Versions) > 0 {
			return fmt.Sprintf("%d %s in each version · checked %s", len(r.LinkList), Plural(int64(len(r.LinkList)), "link", "links"), b.when(r.CheckedAt))
		}
		return fmt.Sprintf("%d %s · checked %s", len(r.LinkList), Plural(int64(len(r.LinkList)), "link", "links"), b.when(r.CheckedAt))
	case GroupLanguages:
		return b.languagesCaption()
	case GroupCompliance:
		return "Gmail and Yahoo bulk-sender rules"
	case GroupAudience:
		a := b.f.Audience
		if a.Empty {
			return "no audience yet"
		}
		return fmt.Sprintf("%s · %s contacts", firstNonEmpty(a.GroupLabel, a.BaseLabel), Count(a.Base))
	case GroupRouting:
		return streamLabel(b.f.Route.Stream) + " stream"
	case GroupAccessibility:
		return b.accessibilityCaption()
	case GroupConditional:
		return b.conditionalCaption()
	}
	return ""
}

// summaryOrder puts a group's passing checks in the order its one-line
// summary names them ("71 KB of 102 KB, image ratio 44%, …"); checks not
// listed keep their place after the listed ones.
var summaryOrder = map[string]int{"content.size": 1, "content.image_ratio": 2, "content.alt_text": 3, "content.plain_text": 4}

// summary is the line a group shows when its passing checks are collapsed.
func (b *builder) summary(g string, passes []ReadinessCheck) string {
	passes = slices.Clone(passes)
	slices.SortStableFunc(passes, func(x, y ReadinessCheck) int {
		rx, ry := summaryOrder[x.Key], summaryOrder[y.Key]
		if rx == 0 {
			rx = len(summaryOrder) + 1
		}
		if ry == 0 {
			ry = len(summaryOrder) + 1
		}
		return rx - ry
	})
	shorts := make([]string, 0, len(passes))
	for _, c := range passes {
		shorts = append(shorts, c.Short)
	}
	switch g {
	case GroupAuthentication:
		parts := []string{}
		for _, s := range shorts {
			if s == "SPF" || s == "DKIM" || s == "custom MAIL FROM" || strings.HasPrefix(s, "DMARC") {
				parts = append(parts, s)
			}
		}
		if len(parts) == 0 {
			return ""
		}
		names := shortsOf(b.availableShares())
		return JoinAnd(parts) + " verified for " + JoinAnd(names)
	case GroupContent, GroupCompliance:
		if g == GroupCompliance && len(shorts) < 3 {
			out := []string{}
			for _, s := range shorts {
				switch s {
				case "one-click unsubscribe headers":
					out = append(out, "List-Unsubscribe and List-Unsubscribe-Post headers present")
				case "bulk-sender rules":
					out = append(out, "bulk-sender rules met")
				default:
					out = append(out, s)
				}
			}
			return strings.Join(out, "; ")
		}
		return strings.Join(shorts, ", ")
	case GroupAudience:
		a := b.f.Audience
		return fmt.Sprintf("%s recently bounced, %s role addresses excluded, %s suppressed", Count(a.RecentlyBounced), Count(a.Role), Count(a.Suppressed))
	}
	return strings.Join(shorts, "; ")
}

func (b *builder) running(g string) string {
	switch g {
	case GroupReputation:
		assets := []string{}
		for _, s := range b.availableShares() {
			assets = append(assets, s.IPs...)
		}
		if d := firstNonEmpty(b.f.Auth.Domain, b.f.Campaign.SenderDomain); d != "" {
			assets = append(assets, d)
		}
		return fmt.Sprintf("checking %s on %d sources", JoinAnd(assets), b.f.Reputation.Sources)
	case GroupAuthentication:
		return "SPF, DKIM, DMARC and MAIL FROM for " + JoinAnd(shortsOf(b.availableShares()))
	case GroupContent:
		return "spam triggers, size, images, alt text and plain text"
	case GroupLinks:
		return fmt.Sprintf("fetching %d URLs and checking URI blocklists", len(b.f.Content.Report.LinkList))
	case GroupCompliance:
		return "one-click unsubscribe headers, footer address, bulk-sender rules"
	case GroupAudience:
		return "unengaged, recently bounced and role addresses"
	case GroupRouting:
		return "provider health, terms and tracking settings"
	case GroupLanguages:
		return "who gets which version, outdated and draft versions, page copy"
	case GroupAccessibility:
		return accessibilityRunning
	case GroupConditional:
		return conditionalRunning
	}
	return ""
}

// RunningText is what a group shows while its checks run, before facts exist.
func RunningText(g string) string {
	switch g {
	case GroupReputation:
		return "domain and IP blocklists, provider bounce and complaint rates"
	case GroupAuthentication:
		return "SPF, DKIM, DMARC and MAIL FROM"
	case GroupContent:
		return "spam triggers, size, images, alt text and plain text"
	case GroupLinks:
		return "fetching links and checking URI blocklists"
	case GroupCompliance:
		return "one-click unsubscribe headers, footer address, bulk-sender rules"
	case GroupAudience:
		return "unengaged, recently bounced and role addresses"
	case GroupRouting:
		return "provider health, terms and tracking settings"
	case GroupLanguages:
		return "who gets which version, outdated and draft versions, page copy"
	case GroupAccessibility:
		return accessibilityRunning
	case GroupConditional:
		return conditionalRunning
	}
	return ""
}
