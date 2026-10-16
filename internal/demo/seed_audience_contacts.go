package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
)

// List bits of a generated contact.
const (
	audInDigest = 1 << iota
	audInUpdates
	audInBeta
)

var audListBits = []int{audInDigest, audInUpdates, audInBeta}

// audEvent is the engagement event a generated contact needs for the saved
// segments: an open of a campaign or a click.
type audEvent struct {
	kind     string // open, click
	at       time.Time
	campaign string // campaign name, "" when none fits
}

// audContact is one generated contact before it is written.
type audContact struct {
	id                         uuid.UUID
	email, first, last         string
	mailbox, detail            string
	tier, consent, source      string
	company, country, tz, lang string
	created                    time.Time
	lastOpen, lastHuman        *time.Time
	lastClick, lastSent        *time.Time
	machine                    bool
	activity                   time.Time
	tags                       []string
	attrs                      map[string]any
	lists                      int // audInDigest | audInUpdates | audInBeta
	pending                    bool
	subscribed                 map[int]time.Time // list bit → subscribed_at
	memberSource               map[int]string    // list bit → membership source
	sunsetAt, stopped          *time.Time
	importID                   *uuid.UUID
	board                      string
	ev                         *audEvent
}

func (c *audContact) imported() bool {
	return strings.HasPrefix(c.source, "import:") && c.source != "import:csv"
}

// audSeed carries what the late audience seeding needs (events,
// suppressions, segments) from the contact generation.
type audSeed struct {
	contacts       []*audContact
	byBoard        map[string]*audContact
	departed       []audDeparture
	tiersComputed  time.Time
	brevoImport    uuid.UUID
	mailchimpOld   uuid.UUID
	mailchimpDraft uuid.UUID
	emails         map[string]bool
	sends          []audSend
	supps          []audSupp
}

// audDeparture is a contact who left a list in the last 30 days.
type audDeparture struct {
	c    *audContact
	list int
	at   time.Time
}

// audSend is a Broadcast of the sample data and when it went out (the
// campaigns themselves are seeded in seed_campaigns.go).
type audSend struct {
	name  string
	at    time.Time
	lists int
}

func audSends(s *Seeder) []audSend {
	all := audInDigest | audInUpdates | audInBeta
	out := []audSend{
		{"October Product Update", s.designDay(3, 10, 0), all},
		{"Weekly Digest #141", s.designDay(5, 9, 0), audInDigest},
		{"Feature flags preview", s.designDay(10, 15, 0), audInBeta},
		{"Weekly Digest #140", s.designDay(12, 9, 0), audInDigest},
		{"Weekly Digest #139", s.designDay(19, 9, 0), audInDigest},
	}
	// Sent from Brevo before the migration.
	for i := 0; i < 39; i++ {
		out = append(out, audSend{fmt.Sprintf("Weekly Digest #%d", 138-i), s.designDay(26+7*i, 9, 0), audInDigest})
	}
	return out
}

func (a *audSeed) send(name string) audSend {
	for _, x := range a.sends {
		if x.name == name {
			return x
		}
	}
	return audSend{}
}

// audCohort is a group of contacts with the same lists and tier.
type audCohort struct {
	lists int
	tier  string
	n     int
}

// The audCohorts behind the sample numbers: list sizes 31,988 / 14,110 /
// 2,115, tiers 29,412 / 9,884 / 5,302 / 3,615, plus 2,118 contacts from a
// Mailchimp import still waiting for consent (Weekly Digest, pending).
var audCohorts = []audCohort{
	{audInDigest, "active", 19537}, {audInDigest, "cooling", 6048}, {audInDigest, "at_risk", 2920}, {audInDigest, "sunset", 1883},
	{audInUpdates, "active", 7180}, {audInUpdates, "cooling", 2822}, {audInUpdates, "at_risk", 1547}, {audInUpdates, "sunset", 1043},
	{audInBeta, "active", 1012}, {audInBeta, "cooling", 182}, {audInBeta, "at_risk", 78}, {audInBeta, "sunset", 25},
	{audInDigest | audInUpdates, "active", 600}, {audInDigest | audInUpdates, "cooling", 210}, {audInDigest | audInUpdates, "at_risk", 115}, {audInDigest | audInUpdates, "sunset", 75},
	{audInDigest | audInBeta, "active", 234}, {audInDigest | audInBeta, "cooling", 42}, {audInDigest | audInBeta, "at_risk", 18}, {audInDigest | audInBeta, "sunset", 6},
	{audInUpdates | audInBeta, "active", 170}, {audInUpdates | audInBeta, "cooling", 31}, {audInUpdates | audInBeta, "at_risk", 13}, {audInUpdates | audInBeta, "sunset", 4},
	{audInDigest | audInUpdates | audInBeta, "active", 234}, {audInDigest | audInUpdates | audInBeta, "cooling", 41}, {audInDigest | audInUpdates | audInBeta, "at_risk", 18}, {audInDigest | audInUpdates | audInBeta, "sunset", 7},
}

// audMailchimpPending is the tier mix of the consent-pending Mailchimp cohort.
var audMailchimpPending = []audCohort{{audInDigest, "active", 445}, {audInDigest, "cooling", 508}, {audInDigest, "at_risk", 593}, {audInDigest, "sunset", 572}}

// audBoardContact is a contact shown by name on the Contacts board.
type audBoardContact struct {
	key, first, last, email, mailbox, detail, tier, consent, source, company string
	lists                                                                    int
	pending                                                                  bool
	joined                                                                   int // days before the design's today
	lastOpen                                                                 int // days before the design's today
	openHour                                                                 int
	machine                                                                  bool
	plan                                                                     string
}

var audBoardContacts = []audBoardContact{
	{key: "sam", first: "Sam", last: "Rivera", email: "sam.rivera@outlook.com", mailbox: "outlook", tier: "active", consent: "double", source: "import:brevo", company: "Northwind Labs", lists: audInDigest | audInUpdates, joined: 24, lastOpen: 5, openHour: 9, plan: "team"},
	{key: "priya", first: "Priya", last: "Natarajan", email: "priya.n@gmail.com", mailbox: "gmail", tier: "active", consent: "double", source: "form:docs", company: "Lattice Labs", lists: audInDigest | audInBeta, joined: 589, lastOpen: 3, openHour: 11, plan: "starter"},
	{key: "diego", first: "Diego", last: "Alvarez", email: "d.alvarez@yahoo.com", mailbox: "yahoo", tier: "cooling", consent: "double", source: "form:blog", lists: audInDigest, joined: 700, lastOpen: 59, openHour: 8, plan: "business"},
	{key: "hannah", first: "Hannah", last: "Berg", email: "hannah@bergconsulting.de", mailbox: "other", detail: "Google Workspace", tier: "active", consent: "double", source: "api", company: "Berg Consulting", lists: audInUpdates, joined: 112, lastOpen: 3, openHour: 14},
	{key: "kenji", first: "Kenji", last: "Watanabe", email: "kenji.w@icloud.com", mailbox: "icloud", tier: "active", consent: "double", source: "form:footer", lists: audInDigest, joined: 280, lastOpen: 5, openHour: 9, machine: true},
	{key: "lena", first: "Lena", last: "Fischer", email: "lena.fischer@hotmail.com", mailbox: "outlook", tier: "at_risk", consent: "pending", source: "import:mailchimp", lists: audInDigest, pending: true, joined: 24, lastOpen: 131, openHour: 18},
	{key: "omar", first: "Omar", last: "Haddad", email: "omar.haddad@gmail.com", mailbox: "gmail", tier: "active", consent: "double", source: "form:pricing", lists: audInDigest | audInUpdates | audInBeta, joined: 377, lastOpen: 3, openHour: 12},
	{key: "grace", first: "Grace", last: "Okafor", email: "g.okafor@outlook.com", mailbox: "outlook", tier: "sunset", consent: "single", source: "import:csv", lists: audInDigest, joined: 971, lastOpen: 224, openHour: 7},
	{key: "tomasz", first: "Tomasz", last: "Nowak", email: "t.nowak@gmail.com", mailbox: "gmail", tier: "cooling", consent: "double", source: "form:popup", lists: audInUpdates, joined: 65, lastOpen: 44, openHour: 20},
	{key: "aisha", first: "Aisha", last: "Rahman", email: "aisha@rahmanlabs.io", mailbox: "other", detail: "Fastmail", tier: "active", consent: "double", source: "api", lists: audInBeta, joined: 23, lastOpen: 10, openHour: 16},
}

// Global mailbox mix: Gmail 46%, Outlook 21%, Yahoo 9%, iCloud 8%, Other 16% of 48,213.
var audMailboxTotals = map[string]int{"gmail": 22178, "outlook": 10125, "yahoo": 4339, "icloud": 3857, "other": 7714}

// The consent-pending Mailchimp cohort leans Outlook (it drives the Outlook complaints).
var audPendingMailboxes = map[string]int{"outlook": 900, "gmail": 700, "yahoo": 250, "icloud": 120, "other": 148}

var audMailboxOrder = []string{"gmail", "outlook", "yahoo", "icloud", "other"}

type audGen struct {
	s      *Seeder
	rng    *rand.Rand
	emails map[string]bool
	now    time.Time
}

// audRand is a seeded generator, so the sample data is the same on every
// run. It never produces anything secret.
func audRand(a, b uint64) *rand.Rand {
	return rand.New(rand.NewPCG(a, b)) //nolint:gosec // deterministic sample data
}

func (g *audGen) daysAgo(min, max float64) time.Time {
	d := min + g.rng.Float64()*(max-min)
	return g.now.Add(-time.Duration(d * 24 * float64(time.Hour))).Truncate(time.Second)
}

func (g *audGen) minutes(min, max int) time.Duration {
	return time.Duration(min+g.rng.IntN(max-min+1)) * time.Minute
}

func (g *audGen) name() (string, string) {
	return audFirstNames[g.rng.IntN(len(audFirstNames))], audLastNames[g.rng.IntN(len(audLastNames))]
}

func audASCII(s string) string {
	r := strings.NewReplacer("ä", "a", "ö", "o", "ü", "u", "é", "e", "è", "e", "á", "a", "í", "i", "ó", "o", "ú", "u", "ñ", "n", "ç", "c", "ã", "a")
	return strings.ToLower(r.Replace(s))
}

// email makes a unique, plausible address for a name at a domain.
func (g *audGen) email(first, last, domain string) string {
	f, l := audASCII(first), audASCII(last)
	var local string
	switch g.rng.IntN(7) {
	case 0:
		local = f + "." + l
	case 1:
		local = f + l[:1]
	case 2:
		local = f[:1] + "." + l
	case 3:
		local = f + "_" + l
	case 4:
		local = f + l
	case 5:
		local = f + "." + l + fmt.Sprint(g.rng.IntN(90)+10)
	default:
		local = l + "." + f
	}
	e := local + "@" + domain
	for i := 2; g.emails[e] || emailaddr.IsRole(e) || emailaddr.TrapLike(e); i++ {
		e = fmt.Sprintf("%s%d@%s", local, i, domain)
	}
	g.emails[e] = true
	return e
}

func (g *audGen) domain(mailbox string) (string, string) {
	switch mailbox {
	case "gmail":
		return "gmail.com", ""
	case "outlook":
		return audOutlookDomains[g.rng.IntN(len(audOutlookDomains))], ""
	case "yahoo":
		return audYahooDomains[g.rng.IntN(len(audYahooDomains))], ""
	case "icloud":
		return audICloudDomains[g.rng.IntN(len(audICloudDomains))], ""
	}
	c := audCompanyDomains[g.rng.IntN(len(audCompanyDomains))]
	return c.domain, c.hosting
}

// engagement sets the last open (and sometimes a click) inside the window
// of the contact's tier.
func (g *audGen) engagement(c *audContact) {
	var lo, hi float64
	switch c.tier {
	case "active":
		lo, hi = 0.5, 29.5
	case "cooling":
		lo, hi = 31.5, 89.5
	case "at_risk":
		lo, hi = 91.5, 172.5
	default:
		lo, hi = 182, 700
	}
	if c.tier == "sunset" && !c.imported() && g.rng.IntN(10) < 4 {
		// Never opened: they joined long enough ago to be Sunset.
		c.created = g.daysAgo(190, 900)
		return
	}
	t := g.daysAgo(lo, hi)
	c.lastHuman = &t
	open := t
	c.lastOpen = &open
	if c.mailbox == "icloud" && c.tier != "sunset" && g.rng.IntN(10) < 7 {
		// Apple Mail Privacy Protection fetched the last email.
		m := g.daysAgo(0.2, 5)
		c.lastOpen = &m
		c.machine = true
	}
	if g.rng.IntN(4) == 0 {
		cl := t.Add(g.minutes(1, 90))
		c.lastClick = &cl
	}
	// Imported contacts keep their engagement history from before they
	// joined ReputeCast; everyone else joined before they last opened.
	if !c.imported() && c.created.After(t) {
		c.created = t.Add(-time.Duration(1+g.rng.IntN(400)) * 24 * time.Hour)
	}
}

// audGenerateContacts builds the 48,213 contacts of Acme Newsletter.
func audGenerateContacts(s *Seeder, a *audSeed) {
	g := &audGen{s: s, rng: audRand(48213, 2026), emails: a.emails, now: s.Now}
	brevoAt := s.designDay(24, 16, 40)
	mcAt := s.designDay(24, 14, 5)
	a.sends = audSends(s)

	type key struct {
		lists   int
		tier    string
		pending bool
	}
	need := map[key]int{}
	for _, co := range audCohorts {
		need[key{co.lists, co.tier, false}] += co.n
	}
	for _, co := range audMailchimpPending {
		need[key{co.lists, co.tier, true}] += co.n
	}
	mailboxes := map[string]int{}
	pendingBoxes := map[string]int{}
	for k, v := range audMailboxTotals {
		mailboxes[k] = v - audPendingMailboxes[k]
		pendingBoxes[k] = audPendingMailboxes[k]
	}

	// Board contacts first, so their slots come out of the cohorts.
	for _, b := range audBoardContacts {
		c := &audContact{id: ID(), email: b.email, first: b.first, last: b.last, mailbox: b.mailbox, detail: b.detail, tier: b.tier, consent: b.consent,
			source: b.source, company: b.company, lists: b.lists, pending: b.pending, board: b.key, attrs: map[string]any{}, tags: []string{},
			subscribed: map[int]time.Time{}, memberSource: map[int]string{}, lang: "en", country: "US", tz: "America/Chicago"}
		c.created = s.designDay(b.joined, 10, 0)
		switch b.key {
		case "sam":
			c.created = brevoAt
		case "lena":
			c.created = mcAt
		}
		lo := s.designDay(b.lastOpen, b.openHour, 41)
		human := lo
		c.lastHuman, c.lastOpen = &human, &lo
		if b.machine {
			h := s.designDay(17, 8, 12)
			c.lastHuman = &h
			c.machine = true
		}
		g.emails[c.email] = true
		need[key{c.lists, c.tier, c.pending}]--
		if c.pending {
			pendingBoxes[c.mailbox]--
		} else {
			mailboxes[c.mailbox]--
		}
		if b.plan != "" {
			c.attrs["plan"] = b.plan
			c.attrs["mrr"] = map[string]float64{"starter": 9, "team": 29, "business": 79}[b.plan]
		}
		a.byBoard[b.key] = c
		a.contacts = append(a.contacts, c)
	}
	sam := a.byBoard["sam"]
	sam.tags = []string{"customer", "team-plan", "feature-flags-beta"}
	sam.attrs["signup_source"] = "brevo-migration"
	sam.importID = &a.brevoImport
	click := s.designDay(3, 10, 22)
	sam.lastClick = &click
	priya := a.byBoard["priya"]
	priya.attrs["signup_source"] = "docs"
	priya.tz, priya.country = "Europe/Berlin", "DE"
	priya.tags = []string{"beta"}
	a.byBoard["diego"].attrs["signup_source"] = "webinar"
	a.byBoard["hannah"].country, a.byBoard["hannah"].tz, a.byBoard["hannah"].lang = "DE", "Europe/Berlin", "de"
	lena := a.byBoard["lena"]
	lena.importID = &a.mailchimpOld
	lena.country, lena.tz, lena.lang = "DE", "Europe/Berlin", "de"
	a.byBoard["kenji"].country, a.byBoard["kenji"].tz = "JP", "Asia/Tokyo"
	a.byBoard["omar"].country, a.byBoard["omar"].tz = "US", "America/New_York"
	a.byBoard["grace"].country, a.byBoard["grace"].tz = "NG", "Africa/Lagos"
	a.byBoard["tomasz"].country, a.byBoard["tomasz"].tz, a.byBoard["tomasz"].lang = "PL", "Europe/Warsaw", "pl"
	a.byBoard["aisha"].country, a.byBoard["aisha"].tz = "GB", "Europe/London"
	for _, c := range a.contacts {
		if !c.pending && !c.imported() && c.created.Before(brevoAt) {
			c.importID = &a.brevoImport // everyone came over from Brevo on 18 Sep
		}
	}

	// Mailbox pools, shuffled.
	pool := func(m map[string]int) []string {
		var out []string
		for _, k := range audMailboxOrder {
			for i := 0; i < m[k]; i++ {
				out = append(out, k)
			}
		}
		g.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
		return out
	}
	boxes := pool(mailboxes)
	pboxes := pool(pendingBoxes)
	sources := []string{"form:docs", "form:blog", "form:footer", "form:pricing", "form:popup", "api", "import:csv"}
	make1 := func(lists int, tier string, pending bool) *audContact {
		var mb string
		if pending {
			mb, pboxes = pboxes[0], pboxes[1:]
		} else {
			mb, boxes = boxes[0], boxes[1:]
		}
		first, last := g.name()
		domain, hosting := g.domain(mb)
		cc := audCountries[g.rng.IntN(len(audCountries))]
		c := &audContact{id: ID(), first: first, last: last, mailbox: mb, tier: tier, lists: lists, pending: pending,
			attrs: map[string]any{}, tags: []string{}, subscribed: map[int]time.Time{}, memberSource: map[int]string{},
			country: cc.code, tz: cc.tz, lang: "en"}
		if mb == "other" {
			c.detail = hosting
		}
		c.email = g.email(first, last, domain)
		switch {
		case pending:
			c.consent, c.source, c.created, c.importID = "pending", "import:mailchimp", mcAt, &a.mailchimpOld
		case tier == "active" && g.rng.IntN(100) < 14:
			// Brevo had no signup date or source for them.
			c.consent, c.source, c.created, c.importID = "double", "import:brevo", brevoAt, &a.brevoImport
		default:
			c.consent = "double"
			if g.rng.IntN(100) < 14 {
				c.consent = "single"
			}
			c.source = sources[g.rng.IntN(len(sources))]
			if tier == "active" && g.rng.IntN(100) < 4 {
				// Joined after the migration.
				c.created = g.daysAgo(0.5, 23.5)
				if g.rng.IntN(3) == 0 {
					c.source = "api"
				}
			} else {
				c.created = g.daysAgo(40, 1100)
				c.importID = &a.brevoImport
			}
		}
		switch c.country {
		case "DE":
			c.lang = "de"
		case "FR":
			c.lang = "fr"
		case "ES", "MX":
			c.lang = "es"
		case "BR":
			c.lang = "pt"
		}
		g.engagement(c)
		return c
	}
	for _, list := range []struct {
		cohorts []audCohort
		pending bool
	}{{audCohorts, false}, {audMailchimpPending, true}} {
		for _, co := range list.cohorts {
			k := key{co.lists, co.tier, list.pending}
			for i := 0; i < need[k]; i++ {
				a.contacts = append(a.contacts, make1(co.lists, co.tier, list.pending))
			}
			need[k] = 0
		}
	}
	audShapeSegments(g, a)
	audShapeSunset(g, a)
	audShapeMemberships(g, a)
	audShapeSent(g, a)
	audShapeActivity(g, a)
}

func audIsGenerated(c *audContact) bool { return c.board == "" }

// after returns a time between min and max after t, and never later than
// a few minutes ago.
func (g *audGen) after(t time.Time, min, max time.Duration) time.Time {
	d := min + time.Duration(g.rng.Int64N(int64(max-min)))
	out := t.Add(d).Truncate(time.Second)
	if limit := g.now.Add(-10 * time.Minute); out.After(limit) {
		out = limit.Add(-time.Duration(g.rng.IntN(3600)) * time.Second)
	}
	return out
}

// audShapeSegments sets up the contacts behind the saved segments' counts.
func audShapeSegments(g *audGen, a *audSeed) {
	sam := a.byBoard["sam"]
	var outlookActive, outlookCooling, betaActive, gmailActive, gmailCooling []*audContact
	for _, c := range a.contacts {
		if !audIsGenerated(c) || c.pending {
			continue
		}
		outlookDigest := c.mailbox == "outlook" && c.lists&audInDigest != 0
		switch {
		case outlookDigest && c.tier == "active":
			outlookActive = append(outlookActive, c)
		case outlookDigest && c.tier == "cooling":
			outlookCooling = append(outlookCooling, c)
		}
		if c.lists&audInBeta != 0 && c.tier == "active" && !outlookDigest {
			betaActive = append(betaActive, c)
		}
		if c.mailbox == "gmail" && c.tier == "active" && c.lists&audInBeta == 0 {
			gmailActive = append(gmailActive, c)
		}
		if c.mailbox == "gmail" && c.tier == "cooling" {
			gmailCooling = append(gmailCooling, c)
		}
	}
	recent := []audSend{a.send("October Product Update"), a.send("Weekly Digest #141"), a.send("Weekly Digest #140"), a.send("Weekly Digest #139")}

	// Engaged Outlook users on paid plans: 1,821 Active (Sam included) and
	// 297 Cooling who opened this morning, after the 03:00 tier run.
	s1Active := append([]*audContact{sam}, outlookActive[:1820]...)
	s1Cooling := outlookCooling[:297]
	business := 0
	for i, c := range append(append([]*audContact{}, s1Active...), s1Cooling...) {
		if c == sam {
			continue
		}
		plan := "team"
		if i%7 < 2 && business < 600 {
			plan = "business"
			business++
		}
		c.attrs["plan"] = plan
		c.attrs["mrr"] = map[string]float64{"team": 29, "business": 79}[plan]
		c.tags = append(c.tags, "customer")
	}
	for _, c := range s1Active[1:] {
		send := recent[g.rng.IntN(len(recent))]
		t := g.after(send.at, 10*time.Minute, 36*time.Hour)
		c.ev = &audEvent{kind: "open", at: t, campaign: send.name}
		c.lastHuman, c.lastOpen, c.machine = &t, &t, false
	}
	from := a.tiersComputed.Add(time.Minute)
	to := g.now.Add(-3 * time.Minute)
	if to.Before(from) {
		to = from
	}
	for i, c := range s1Cooling {
		t := from.Add(time.Duration(float64(to.Sub(from)) * float64(i+1) / float64(len(s1Cooling)+1))).Truncate(time.Second)
		c.ev = &audEvent{kind: "open", at: t, campaign: "Weekly Digest #141"}
		c.lastHuman, c.lastOpen, c.machine = &t, &t, false
	}

	// Beta Program, opened a beta email: 1,402 with Aisha.
	ffp := a.send("Feature flags preview")
	for _, c := range append(betaActive[:1401:1401], a.byBoard["aisha"]) {
		t := g.after(ffp.at, 10*time.Minute, 48*time.Hour)
		if c.board == "aisha" {
			t = *c.lastHuman
		}
		c.ev = &audEvent{kind: "open", at: t, campaign: ffp.name}
		if c.lastHuman == nil || c.lastHuman.Before(t) {
			c.lastHuman, c.lastOpen, c.machine = &t, &t, false
		}
	}

	// Gmail only, clickers: 4,916 with a click in 90 days.
	for _, c := range gmailActive[:3500] {
		send := recent[0]
		if c.lists&audInDigest != 0 {
			send = recent[g.rng.IntN(len(recent))]
		}
		t := g.after(send.at, 10*time.Minute, 36*time.Hour)
		c.ev = &audEvent{kind: "click", at: t, campaign: send.name}
		c.lastClick = &t
		o := t.Add(-g.minutes(1, 20))
		if c.lastHuman == nil || c.lastHuman.Before(o) {
			c.lastHuman, c.lastOpen, c.machine = &o, &o, false
		}
	}
	var older []audSend
	for _, x := range a.sends {
		if age := g.now.Sub(x.at); age > 32*24*time.Hour && age < 84*24*time.Hour {
			older = append(older, x)
		}
	}
	for _, c := range gmailCooling[:1416] {
		var t time.Time
		campaign := ""
		if c.lists&audInDigest != 0 && len(older) > 0 {
			send := older[g.rng.IntN(len(older))]
			t = g.after(send.at, 10*time.Minute, 24*time.Hour)
			campaign = send.name
		} else {
			t = g.daysAgo(32, 88)
		}
		c.ev = &audEvent{kind: "click", at: t, campaign: campaign}
		c.lastClick = &t
		o := t.Add(-g.minutes(1, 20))
		if c.lastHuman == nil || c.lastHuman.Before(o) {
			c.lastHuman, c.lastOpen, c.machine = &o, &o, false
		}
	}

	// Other customers, never Outlook on the Weekly Digest (that would put
	// them in the first segment): 486 more on the business plan, so
	// "mrr over $50" is 1,087 with Diego; starter and team plans for others.
	others, paid := 0, 0
	for _, c := range a.contacts {
		if !audIsGenerated(c) || c.pending || c.attrs["plan"] != nil || (c.mailbox == "outlook" && c.lists&audInDigest != 0) {
			continue
		}
		switch {
		case others < 486 && c.lists&audInUpdates != 0:
			c.attrs["plan"], c.attrs["mrr"] = "business", float64(79)
			c.company = audCompanyName(g, c)
			c.tags = append(c.tags, "customer")
			others++
		case paid < 5200 && g.rng.IntN(4) == 0:
			plan, mrr := "starter", float64(9)
			if g.rng.IntN(3) == 0 {
				plan, mrr = "team", 29
				c.tags = append(c.tags, "customer")
			} else if g.rng.IntN(5) == 0 {
				c.tags = append(c.tags, "trial")
			}
			c.attrs["plan"], c.attrs["mrr"] = plan, mrr
			paid++
		}
		if g.rng.IntN(5) == 0 {
			c.attrs["signup_source"] = []string{"pricing-page", "docs", "webinar", "blog", "brevo-migration"}[g.rng.IntN(5)]
		}
		if c.company == "" && c.mailbox == "other" && g.rng.IntN(3) > 0 {
			c.company = audCompanyName(g, c)
		}
	}
	for _, c := range a.contacts {
		if c.lists&audInBeta != 0 && audIsGenerated(c) && g.rng.IntN(2) == 0 {
			c.tags = append(c.tags, "beta")
		}
	}
}

func audCompanyName(g *audGen, c *audContact) string {
	if c.mailbox == "other" {
		d := emailaddr.Domain(c.email)
		if i := strings.Index(d, "."); i > 0 {
			d = d[:i]
		}
		parts := strings.Split(d, "-")
		for i, p := range parts {
			if p != "" {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
		return strings.Join(parts, " ")
	}
	return []string{"Kestrel Analytics", "Bluefin Software", "Oak & Iron", "Pixel Harbor", "Tidewater Labs", "Meridian Health"}[g.rng.IntN(6)]
}

// audNextWeekday returns the next given weekday at hour:min in the
// workspace timezone, strictly after now.
func audNextWeekday(now time.Time, day time.Weekday, hour, min int) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	n := now.In(loc)
	t := time.Date(n.Year(), n.Month(), n.Day(), hour, min, 0, 0, loc)
	for t.Weekday() != day || !t.After(now) {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

// audBaseline is the date tiers count from: the latest human open or click,
// or the day the contact joined.
func audBaseline(c *audContact) time.Time {
	var e time.Time
	for _, t := range []*time.Time{c.lastHuman, c.lastClick} {
		if t != nil && t.After(e) {
			e = *t
		}
	}
	if e.IsZero() {
		return c.created
	}
	return e
}

// audShapeSunset sets the sunset policy's bookkeeping and its "this week":
// 1,204 contacts enter Sunset on Wednesday and 893 stop receiving
// Broadcasts on Friday.
func audShapeSunset(g *audGen, a *audSeed) {
	s := g.s
	week := g.now.Add(7*24*time.Hour - time.Minute)
	// "This week" is read a while after the seed's now (the seed takes a
	// minute or more under load): what is due today stays ahead by half an
	// hour, so it is still counted then.
	const lead = 30 * time.Minute
	wed := audNextWeekday(g.now.Add(lead), time.Wednesday, 10, 0)
	fri := audNextWeekday(g.now, time.Friday, 3, 0)
	enabled := s.designDay(41, 3, 0) // 1 Sep, the policy's first run
	entering, stopping := 0, 0
	for _, c := range a.contacts {
		if c.pending {
			continue
		}
		switch {
		case c.tier == "at_risk" && audIsGenerated(c) && entering < 1204 && c.lastClick == nil:
			cross := wed.Add(g.minutes(0, 59))
			if cross.After(week) {
				cross = week
			}
			t := cross.AddDate(0, 0, -181)
			c.lastHuman, c.lastOpen, c.machine = &t, &t, false
			if !c.imported() && c.created.After(t) {
				c.created = t.AddDate(0, 0, -30-g.rng.IntN(300))
			}
			entering++
		case c.tier == "sunset":
			if audIsGenerated(c) && stopping < 893 {
				stop := fri.Add(-g.minutes(0, 59))
				if !stop.After(g.now.Add(lead)) {
					stop = g.now.Add(lead + g.minutes(1, 5))
				}
				base := stop.AddDate(0, 0, -30-181)
				if c.lastHuman != nil {
					c.lastHuman, c.lastOpen, c.machine = &base, &base, false
					c.lastClick = nil
				} else {
					c.created = base
				}
				stopping++
				// Entered Sunset 30 days before the stop.
				sunset := stop.AddDate(0, 0, -30)
				c.sunsetAt = &sunset
				if !sunset.Before(enabled) {
					c.tags = append(c.tags, "sunset")
				}
				continue
			}
			if audIsGenerated(c) {
				// Keep everyone else out of this week's Friday.
				if b := audBaseline(c); b.After(g.now.AddDate(0, 0, -213)) && b.Before(g.now.AddDate(0, 0, -202)) {
					t := g.now.AddDate(0, 0, -215-g.rng.IntN(30))
					if c.lastHuman != nil {
						c.lastHuman, c.lastOpen, c.machine = &t, &t, false
						c.lastClick = nil
					} else {
						c.created = t
					}
				}
			}
			b := audBaseline(c)
			// The nightly run that saw them cross 181 days.
			sunset := time.Date(b.Year(), b.Month(), b.Day(), 3, 0, 0, 0, b.Location()).AddDate(0, 0, 182)
			if sunset.After(a.tiersComputed) {
				sunset = a.tiersComputed
			}
			c.sunsetAt = &sunset
			if stop := sunset.AddDate(0, 0, 30); !stop.After(g.now) {
				if stop.Before(enabled) {
					stop = enabled
				}
				c.stopped = &stop
			}
			if !sunset.Before(enabled) {
				c.tags = append(c.tags, "sunset")
			}
		}
	}
}

// audShapeMemberships sets subscription dates so 30-day net growth is +812
// (Weekly Digest), +391 (Product Updates) and +81 (Beta Program).
func audShapeMemberships(g *audGen, a *audSeed) {
	s := g.s
	recent := g.now.AddDate(0, 0, -30)
	joined := map[int]int{audInDigest: 932, audInUpdates: 455, audInBeta: 90}
	left := map[int]int{audInDigest: 120, audInUpdates: 64, audInBeta: 9}
	for _, c := range a.contacts {
		for _, bit := range audListBits {
			if c.lists&bit == 0 {
				continue
			}
			at := c.created
			if c.source == "import:brevo" && audIsGenerated(c) {
				// Brevo kept the original list dates.
				at = g.daysAgo(60, 900)
			}
			c.subscribed[bit] = at
			c.memberSource[bit] = c.source
		}
	}
	// Sam added Product Updates in the preference center on 3 Oct.
	sam := a.byBoard["sam"]
	sam.subscribed[audInUpdates] = s.designDay(9, 9, 52)
	sam.memberSource[audInUpdates] = "preference_center"
	got := map[int]int{}
	for _, c := range a.contacts {
		for _, bit := range audListBits {
			if c.lists&bit != 0 && !c.pending && c.subscribed[bit].After(recent) {
				got[bit]++
			}
		}
	}
	for _, bit := range audListBits {
		for _, c := range a.contacts {
			if got[bit] == joined[bit] {
				break
			}
			if c.lists&bit == 0 || c.pending || !audIsGenerated(c) {
				continue
			}
			isRecent := c.subscribed[bit].After(recent)
			switch {
			case got[bit] < joined[bit] && !isRecent:
				at := g.daysAgo(0.5, 29)
				if at.Before(c.created) {
					continue
				}
				c.subscribed[bit] = at
				got[bit]++
			case got[bit] > joined[bit] && isRecent && c.created.Before(recent):
				c.subscribed[bit] = g.daysAgo(31, 300)
				got[bit]--
			}
		}
	}
	// Departures: people who left one list in the last 30 days but kept another.
	for _, bit := range audListBits {
		n := 0
		for i, c := range a.contacts {
			if n >= left[bit] {
				break
			}
			if c.lists&bit != 0 || c.pending || !audIsGenerated(c) || c.tier != "active" || i%11 != 0 || c.created.After(recent) {
				continue
			}
			a.departed = append(a.departed, audDeparture{c: c, list: bit, at: g.daysAgo(0.5, 29)})
			n++
		}
	}
}

// audShapeSent sets when each contact was last emailed.
func audShapeSent(g *audGen, a *audSeed) {
	opu := a.send("October Product Update").at
	for _, c := range a.contacts {
		switch {
		case c.pending:
			continue
		case c.stopped != nil && c.stopped.Before(opu):
			t := c.stopped.Add(-6 * 24 * time.Hour)
			c.lastSent = &t
		case c.created.After(opu):
			if t := c.created.Add(2 * time.Minute); t.Before(g.now) {
				c.lastSent = &t // the welcome email
			}
		default:
			t := opu.Add(g.minutes(0, 40))
			c.lastSent = &t
		}
	}
}

// audShapeActivity sets the default sort: the board's contacts first.
func audShapeActivity(g *audGen, a *audSeed) {
	for _, c := range a.contacts {
		t := c.created
		for _, x := range []*time.Time{c.lastOpen, c.lastClick, c.lastSent} {
			if x != nil && x.After(t) {
				t = *x
			}
		}
		if limit := g.now.Add(-20 * time.Minute); t.After(limit) {
			t = limit
		}
		c.activity = t
	}
	order := []string{"sam", "priya", "diego", "hannah", "kenji", "lena", "omar", "grace", "tomasz", "aisha"}
	for i, k := range order {
		a.byBoard[k].activity = g.now.Add(-time.Duration(i+1) * time.Minute)
	}
}

// audWriteContacts copies the contacts and their list memberships.
func audWriteContacts(ctx context.Context, s *Seeder, a *audSeed, lists map[int]uuid.UUID) error {
	ws := s.WS()
	rows := make([][]any, 0, len(a.contacts))
	var members [][]any
	for _, c := range a.contacts {
		attrs, _ := json.Marshal(c.attrs)
		evidence := []byte(`{}`)
		if c.consent == "double" {
			evidence, _ = json.Marshal(map[string]string{"at": c.created.UTC().Format(time.RFC3339), "source": "double opt-in confirmation"})
		}
		rows = append(rows, []any{c.id, ws.ID, c.email, c.first, c.last, "active", c.tier, c.source, c.tz, c.country, c.mailbox, c.tags, attrs,
			c.lastOpen, c.lastClick, c.lastSent, c.created, c.created, c.company, c.lang, c.detail, c.consent, evidence, c.importID,
			emailaddr.IsRole(c.email), c.lastHuman, c.machine, c.activity, c.sunsetAt, c.stopped, c.sunsetAt})
		for _, bit := range audListBits {
			if c.lists&bit == 0 {
				continue
			}
			var sub any = c.subscribed[bit]
			status := "subscribed"
			if c.pending {
				status, sub = "pending", nil
			}
			members = append(members, []any{lists[bit], c.id, status, c.memberSource[bit], sub, nil, c.subscribed[bit]})
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"contacts"}, []string{"id", "workspace_id", "email", "first_name", "last_name", "status", "tier", "source",
		"timezone", "country", "mailbox_provider", "tags", "attributes", "last_open_at", "last_click_at", "last_sent_at", "created_at", "updated_at",
		"company", "language", "mailbox_detail", "consent", "consent_evidence", "import_id", "role_address", "last_human_open_at",
		"last_open_machine", "last_activity_at", "sunset_at", "broadcasts_stopped_at", "tier_changed_at"}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("contacts: %w", err)
	}
	for _, d := range a.departed {
		members = append(members, []any{lists[d.list], d.c.id, "unsubscribed", d.c.source, d.c.created, d.at, d.c.created})
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"list_memberships"}, []string{"list_id", "contact_id", "status", "source", "subscribed_at", "unsubscribed_at", "created_at"},
		pgx.CopyFromRows(members)); err != nil {
		return fmt.Errorf("memberships: %w", err)
	}
	return nil
}

// audSeedEngineeringBlog gives the second workspace its 6,420 subscribers.
func audSeedEngineeringBlog(ctx context.Context, s *Seeder) error {
	eng := s.Workspaces["acme-engineering-blog"]
	list := s.Lists["Engineering Blog"]
	g := &audGen{s: s, rng: audRand(6420, 7), emails: map[string]bool{}, now: s.Now}
	tiers := []struct {
		tier string
		n    int
	}{{"active", 3980}, {"cooling", 1290}, {"at_risk", 705}, {"sunset", 445}}
	var mbs []string
	for _, k := range audMailboxOrder {
		for i := 0; i < map[string]int{"gmail": 3210, "outlook": 1220, "yahoo": 320, "icloud": 450, "other": 1220}[k]; i++ {
			mbs = append(mbs, k)
		}
	}
	g.rng.Shuffle(len(mbs), func(i, j int) { mbs[i], mbs[j] = mbs[j], mbs[i] })
	var rows, members [][]any
	i := 0
	for _, t := range tiers {
		for k := 0; k < t.n; k++ {
			first, last := g.name()
			mb := mbs[i]
			i++
			domain, hosting := g.domain(mb)
			c := &audContact{tier: t.tier, mailbox: mb, source: "form:blog", created: g.daysAgo(30, 900)}
			g.engagement(c)
			cc := audCountries[g.rng.IntN(len(audCountries))]
			id := ID()
			rows = append(rows, []any{id, eng.ID, g.email(first, last, domain), first, last, t.tier, mb, hosting, "double", "form:blog",
				c.lastOpen, c.lastHuman, c.lastClick, c.created, c.created, c.machine, audBaseline(c), cc.code, cc.tz, "en"})
			members = append(members, []any{list, id, "subscribed", "form:blog", c.created, c.created})
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"contacts"}, []string{"id", "workspace_id", "email", "first_name", "last_name", "tier", "mailbox_provider",
		"mailbox_detail", "consent", "source", "last_open_at", "last_human_open_at", "last_click_at", "created_at", "updated_at", "last_open_machine",
		"last_activity_at", "country", "timezone", "language"}, pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"list_memberships"}, []string{"list_id", "contact_id", "status", "source", "subscribed_at", "created_at"}, pgx.CopyFromRows(members))
	return err
}
