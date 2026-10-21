package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/twilio/twiliotest"
	"github.com/hostrepute/reputecast/internal/sms/vonage/vonagetest"
	"github.com/hostrepute/reputecast/internal/texting"
)

// The registrants of Acme Summit 2026 (brief §4): 1,240 people, 902 of them
// already contacts and 338 new. 1,032 gave a phone number and 891 ticked SMS
// consent; 10 replied STOP to the opt-in confirmation and 7 numbers were
// suppressed when it failed (5 landlines, 2 invalid), which leaves 874 who
// can receive texts: US 611, CA 48, GB 118, DE 52, NL 25, IE 20. Ireland's 20
// haven't had their confirmation yet: ACME is pending there.

// smsPerson is one registrant.
type smsPerson struct {
	id          uuid.UUID
	first, last string
	email       string
	isNew       bool
	// tz is the contact's time zone field ("" when it's assumed from the
	// number); zone is the one texts use.
	tz, zone string
	phone    string
	country  string
	// state: receive, stop (STOP to the confirmation), landline, invalid,
	// phone (a number, no SMS consent) or none (no number).
	state      string
	registered time.Time
	ip         string
	company    string
	// The campaign: how the text did, a click, a reply.
	outcome  string // delivered, unreachable, absent, filtered, not_in_service
	click    bool
	machine  int
	reply    string
	keyword  string // STOP, Stop, HELP: a keyword reply
	replyAt  time.Duration
	region   string // the zone group: de_nl, gb, atlantic, east, central, mountain, pacific, ie
	sentAt   time.Time
	position int
}

// smsBucket is a group of people with numbers alike.
type smsBucket struct {
	country, region, tz string
	codes               []string
	n                   int
	assumed             int // how many have no time zone on their contact
}

var smsReceive = []smsBucket{
	{"DE", "de_nl", "Europe/Berlin", []string{"de"}, 52, 0},
	{"NL", "de_nl", "Europe/Amsterdam", []string{"nl"}, 25, 0},
	{"GB", "gb", "Europe/London", []string{"gb"}, 118, 0},
	{"CA", "atlantic", "America/Halifax", []string{"902"}, 3, 0},
	{"CA", "east", "America/Toronto", []string{"416", "647", "514"}, 23, 0},
	{"CA", "east", "America/Toronto", []string{"709"}, 2, 2},
	{"CA", "central", "America/Winnipeg", []string{"204"}, 4, 0},
	{"CA", "mountain", "America/Edmonton", []string{"780", "403"}, 6, 0},
	{"CA", "pacific", "America/Vancouver", []string{"604"}, 10, 0},
	{"US", "east", "America/New_York", []string{"212", "646", "917", "718", "617", "404", "305", "202", "215", "919", "704", "860", "973", "201"}, 243, 3},
	{"US", "central", "America/Chicago", []string{"312", "773", "630", "708", "847", "512", "214", "713", "612", "314", "414", "615"}, 220, 5},
	{"US", "mountain", "America/Denver", []string{"720", "303", "801", "505"}, 43, 0},
	{"US", "pacific", "America/Los_Angeles", []string{"415", "213", "310", "206", "503", "408", "619"}, 105, 2},
	{"IE", "ie", "Europe/Dublin", []string{"ie"}, 20, 0},
}

// smsNumbers hands out fictional numbers: 555-01xx in North America, the
// UK's drama range, and look-alike mobile ranges elsewhere.
type smsNumbers struct {
	used map[string]bool
	next map[string]int
}

func (g *smsNumbers) take(code string) string {
	for {
		i := g.next[code]
		g.next[code] = i + 1
		var n string
		switch code {
		case "gb":
			n = fmt.Sprintf("+447700%06d", 900100+i)
		case "de":
			n = fmt.Sprintf("+49151234%05d", 50000+i)
		case "nl":
			n = fmt.Sprintf("+316123%05d", 40000+i)
		case "ie":
			n = fmt.Sprintf("+35385123%04d", 1000+i)
		case "fr":
			n = fmt.Sprintf("+3361234%04d", 1000+i)
		default:
			if i > 99 {
				panic("demo: out of 555-01xx numbers for " + code)
			}
			n = fmt.Sprintf("+1%s55501%02d", code, i)
		}
		if !g.used[n] {
			g.used[n] = true
			return n
		}
	}
}

// smsNamed are the people the boards name.
var smsNamed = []struct {
	key, first, last, email, phone, tz, country, region string
	existing                                            bool
}{
	{"sam", "Sam", "Rivera", "sam.rivera@outlook.com", "+13125550123", "America/Chicago", "US", "central", true},
	{"lena", "Lena", "Fischer", "lena.fischer@hotmail.com", "+4915123452207", "Europe/Berlin", "DE", "de_nl", true},
	{"dana", "Dana", "Whitfield", "dana.whitfield@gmail.com", "+17735550148", "America/Chicago", "US", "central", false},
	{"alex", "Alex", "Morgan", "alex.morgan@northwind-labs.com", "+13125550192", "America/Chicago", "US", "central", false},
	{"olivia", "Olivia", "Hughes", "olivia.hughes@outlook.com", "+447700903381", "Europe/London", "GB", "gb", false},
	{"kim", "Kim", "Tran", "kim.tran@gmail.com", "+16465550175", "America/New_York", "US", "east", false},
	{"jordan", "Jordan", "Ellis", "jordan.ellis@yahoo.com", "+17205550133", "America/Denver", "US", "mountain", false},
}

type smsExisting struct {
	id          uuid.UUID
	first, last string
	email       string
	tz          string
}

func smsRegistrants(ctx context.Context, s *Seeder, st *smsState) error {
	ws := s.WS()
	rng := audRand(21, 2026)
	chicago := smsLoc("America/Chicago")
	nums := &smsNumbers{used: map[string]bool{"+13125550161": true, "+447700900456": true, "+18885550142": true, "+13125550187": true}, next: map[string]int{}}
	for _, n := range smsNamed {
		nums.used[n.phone] = true
	}
	for _, n := range []string{"+16305550121", "+14155550177", "+17085550161", "+12195550144", "+447700902290", "+447700903307"} {
		nums.used[n] = true
	}
	// Existing contacts, by time zone, to register with numbers that fit.
	taken := map[uuid.UUID]bool{}
	byEmail := map[string]smsExisting{}
	rows, err := s.Tx.Query(ctx, `SELECT id, first_name, last_name, email, timezone FROM contacts WHERE workspace_id = $1 AND email = ANY($2)`,
		ws.ID, []string{"sam.rivera@outlook.com", "lena.fischer@hotmail.com"})
	if err != nil {
		return err
	}
	for rows.Next() {
		var e smsExisting
		if err := rows.Scan(&e.id, &e.first, &e.last, &e.email, &e.tz); err != nil {
			rows.Close()
			return err
		}
		byEmail[e.email] = e
		taken[e.id] = true
	}
	rows.Close()
	pools := map[string][]smsExisting{}
	pool := func(tz string) ([]smsExisting, error) {
		if p, ok := pools[tz]; ok {
			return p, nil
		}
		rows, err := s.Tx.Query(ctx, `SELECT id, first_name, last_name, email, timezone FROM contacts WHERE workspace_id = $1 AND status = 'active'
			AND erased_at IS NULL AND timezone = $2 AND consent <> 'pending' ORDER BY id LIMIT 600`, ws.ID, tz)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []smsExisting{}
		for rows.Next() {
			var e smsExisting
			if err := rows.Scan(&e.id, &e.first, &e.last, &e.email, &e.tz); err != nil {
				return nil, err
			}
			if !taken[e.id] {
				out = append(out, e)
			}
		}
		pools[tz] = out
		return out, rows.Err()
	}
	existingLeft := 902 - 2 // Sam and Lena
	pickExisting := func(tz string) (*smsExisting, error) {
		if existingLeft <= 0 {
			return nil, nil
		}
		p, err := pool(tz)
		if err != nil {
			return nil, err
		}
		for len(p) > 0 {
			e := p[0]
			p = p[1:]
			pools[tz] = p
			if !taken[e.id] {
				taken[e.id] = true
				existingLeft--
				return &e, nil
			}
		}
		return nil, nil
	}
	emails := map[string]bool{}
	companies := []string{"Northwind Labs", "Lattice Labs", "Brightpath", "Kestrel", "Meridian Analytics", "Bluefin", "Oak & Iron", "Pixel Harbor", "Quillworks",
		"Redwood Capital", "Saltmarsh Studio", "Tidewater", "Umbra Systems", "Vantage Point", "Willowtree", "Zenith Works", "Basalt Engineering", "Cinder"}
	newPerson := func(tz string) smsPerson {
		for {
			first, last := audFirstNames[rng.IntN(len(audFirstNames))], audLastNames[rng.IntN(len(audLastNames))]
			email := strings.ToLower(audASCII(first)+"."+audASCII(last)) + fmt.Sprintf("%d@", rng.IntN(90)+10) + []string{"gmail.com", "outlook.com", "yahoo.com", "icloud.com", "proton.me"}[rng.IntN(5)]
			if emails[email] {
				continue
			}
			emails[email] = true
			return smsPerson{id: ID(), first: first, last: last, email: email, isNew: true, tz: tz, company: companies[rng.IntN(len(companies))]}
		}
	}
	regStart, regEnd := s.smsDay(2, 9, 0, chicago), s.smsDay(13, 20, 0, chicago)
	regAt := func() time.Time {
		return regStart.Add(time.Duration(rng.Int64N(int64(regEnd.Sub(regStart))))).Truncate(time.Second)
	}
	ip := func() string {
		return fmt.Sprintf("%s.%d", []string{"203.0.113", "198.51.100", "192.0.2"}[rng.IntN(3)], 2+rng.IntN(250))
	}
	people := []smsPerson{}
	add := func(p smsPerson) {
		if p.registered.IsZero() {
			p.registered = regAt()
		}
		if p.ip == "" {
			p.ip = ip()
		}
		if p.zone == "" {
			p.zone = p.tz
		}
		people = append(people, p)
	}
	// The named people first.
	namedAt := map[string]int{}
	for _, n := range smsNamed {
		p := smsPerson{first: n.first, last: n.last, email: n.email, tz: n.tz, zone: n.tz, phone: n.phone, country: n.country, region: n.region, state: "receive"}
		if n.existing {
			e, ok := byEmail[n.email]
			if !ok {
				return fmt.Errorf("demo contact %s is missing", n.email)
			}
			p.id, p.first, p.last = e.id, e.first, e.last
			if e.tz != "" {
				p.tz, p.zone = e.tz, e.tz
			}
		} else {
			p.id, p.isNew = ID(), true
			p.company = companies[rng.IntN(len(companies))]
			emails[n.email] = true
		}
		if n.key == "sam" {
			p.registered, p.ip = s.smsDay(2, 14, 6, chicago), "203.0.113.77"
		}
		add(p)
		namedAt[n.key] = len(people) - 1
	}
	// Then everyone who can receive texts, bucket by bucket.
	fixed := map[string][]string{"central": {"+16305550121", "+14155550177"}, "gb": {"+447700903307"}}
	for _, b := range smsReceive {
		n := b.n
		if b.assumed == 0 {
			for _, nm := range smsNamed {
				if nm.country == b.country && nm.region == b.region && nm.tz == b.tz {
					n-- // a named person of this bucket
				}
			}
		} else {
			for _, nm := range smsNamed {
				if nm.country == b.country && nm.region == b.region && nm.tz == b.tz && b.n > b.assumed {
					n--
				}
			}
		}
		for i := 0; i < n; i++ {
			var p smsPerson
			assumed := i < b.assumed
			newOnly := assumed || b.tz == "America/Halifax" || b.tz == "America/Winnipeg" || b.tz == "America/Edmonton" || b.tz == "America/Vancouver" || b.country == "IE"
			if !newOnly {
				e, err := pickExisting(b.tz)
				if err != nil {
					return err
				}
				if e != nil {
					p = smsPerson{id: e.id, first: e.first, last: e.last, email: e.email, tz: e.tz}
				}
			}
			if p.id == uuid.Nil {
				tz := b.tz
				if assumed {
					tz = ""
				}
				p = newPerson(tz)
			}
			p.country, p.region, p.state, p.zone = b.country, b.region, "receive", b.tz
			if list := fixed[b.region]; len(list) > 0 && b.country != "CA" && !assumed {
				p.phone = list[0]
				fixed[b.region] = list[1:]
			} else {
				p.phone = nums.take(b.codes[i%len(b.codes)])
			}
			add(p)
		}
	}
	// STOP to the confirmation (10), failed confirmations (7), numbers
	// without SMS consent (141) and registrants without a number (208).
	others := []struct {
		state, country, region, tz, code string
		n                                int
		phone                            string
	}{
		{"stop", "US", "east", "America/New_York", "212", 2, ""}, {"stop", "US", "central", "America/Chicago", "773", 2, ""},
		{"stop", "US", "pacific", "America/Los_Angeles", "310", 1, ""}, {"stop", "CA", "east", "America/Toronto", "416", 1, ""},
		{"stop", "GB", "gb", "Europe/London", "gb", 2, ""}, {"stop", "GB", "gb", "Europe/London", "gb", 1, "+447700902290"},
		{"stop", "DE", "de_nl", "Europe/Berlin", "de", 1, ""},
		{"landline", "US", "central", "America/Chicago", "708", 1, "+17085550161"}, {"landline", "US", "central", "America/Chicago", "708", 1, ""},
		{"landline", "US", "east", "America/New_York", "212", 1, ""}, {"landline", "US", "pacific", "America/Los_Angeles", "213", 1, ""},
		{"landline", "US", "mountain", "America/Denver", "303", 1, ""},
		{"invalid", "US", "central", "America/Chicago", "219", 1, "+12195550144"}, {"invalid", "US", "east", "America/New_York", "617", 1, ""},
		{"phone", "US", "east", "America/New_York", "718", 40, ""}, {"phone", "US", "central", "America/Chicago", "847", 35, ""},
		{"phone", "US", "pacific", "America/Los_Angeles", "408", 18, ""}, {"phone", "US", "mountain", "America/Denver", "801", 7, ""},
		{"phone", "CA", "east", "America/Toronto", "647", 10, ""}, {"phone", "GB", "gb", "Europe/London", "gb", 15, ""},
		{"phone", "DE", "de_nl", "Europe/Berlin", "de", 8, ""}, {"phone", "NL", "de_nl", "Europe/Amsterdam", "nl", 5, ""},
		{"phone", "FR", "", "Europe/Paris", "fr", 3, ""},
		{"none", "", "", "", "", 208, ""},
	}
	newLeft := 338
	for _, p := range people {
		if p.isNew {
			newLeft--
		}
	}
	for _, o := range others {
		for i := 0; i < o.n; i++ {
			var p smsPerson
			// The rest of the 338 new contacts: everyone without a number,
			// and the last of the numbers without consent.
			wantNew := o.state == "none" || (o.state == "phone" && newLeft > 208)
			if !wantNew && o.tz != "" {
				e, err := pickExisting(o.tz)
				if err != nil {
					return err
				}
				if e != nil {
					p = smsPerson{id: e.id, first: e.first, last: e.last, email: e.email, tz: e.tz}
				}
			}
			if p.id == uuid.Nil {
				tz := o.tz
				if tz == "" {
					tz = []string{"America/New_York", "America/Chicago", "Europe/London", "America/Los_Angeles"}[rng.IntN(4)]
				}
				p = newPerson(tz)
				newLeft--
			}
			p.state, p.country, p.region, p.zone = o.state, o.country, o.region, o.tz
			if o.state != "none" {
				if o.phone != "" && i == 0 {
					p.phone = o.phone
				} else {
					p.phone = nums.take(o.code)
				}
			}
			if p.phone == "+447700902290" {
				p.registered = s.smsDay(2, 17, 58, smsLoc("Europe/London"))
			}
			if p.phone == "+17085550161" {
				p.registered = s.smsDay(3, 11, 14, chicago)
			}
			if p.phone == "+12195550144" {
				p.registered = s.smsDay(4, 16, 39, chicago)
			}
			add(p)
		}
	}
	if len(people) != 1240 {
		return fmt.Errorf("demo: %d registrants, want 1,240", len(people))
	}
	nNew := 0
	for _, p := range people {
		if p.isNew {
			nNew++
		}
	}
	if nNew != 338 {
		return fmt.Errorf("demo: %d new registrants, want 338", nNew)
	}
	named := map[string]*smsPerson{}
	for k, i := range namedAt {
		named[k] = &people[i]
	}
	smsCampaignOutcomes(people, named, rng)
	return smsWrite(ctx, s, st, people, named, rng)
}

// smsCampaignOutcomes decides how "Summit: 8 days to go" did for each
// recipient: 840 delivered, 14 failed, 232 people clicked (41 link
// previews apart), 6 opted out, 2 asked for help, 11 replied.
func smsCampaignOutcomes(people []smsPerson, named map[string]*smsPerson, rng *rand.Rand) {
	byRegion := map[string][]*smsPerson{}
	for i := range people {
		p := &people[i]
		if p.state == "receive" && p.country != "IE" {
			key := p.region + "/" + p.country
			byRegion[key] = append(byRegion[key], p)
			p.outcome = "delivered"
		}
	}
	setFail := func(key string, outcomes ...string) {
		list := byRegion[key]
		j := 0
		for _, o := range outcomes {
			for j < len(list) && (list[j].outcome != "delivered" || isNamed(list[j], named)) {
				j++
			}
			if j < len(list) {
				list[j].outcome = o
				j++
			}
		}
	}
	// The fixed numbers first: the filtered Chicago number, the number no
	// longer in service, the UK number whose phone was off.
	for i := range people {
		switch people[i].phone {
		case "+16305550121":
			people[i].outcome = "filtered"
		case "+14155550177":
			people[i].outcome = "not_in_service"
		case "+447700903307":
			people[i].outcome = "absent"
		}
	}
	setFail("de_nl/DE", "absent")
	setFail("gb/GB", "absent")
	setFail("east/CA", "unreachable")
	setFail("east/US", "unreachable", "unreachable", "filtered", "filtered", "not_in_service")
	setFail("central/US", "unreachable", "unreachable")
	setFail("mountain/US", "unreachable")
	// Clicks by people, per country.
	clicks := map[string]int{"US": 158, "CA": 13, "GB": 34, "DE": 17, "NL": 10}
	alex := named["alex"]
	alex.click = true
	clicks["US"]--
	delivered := []*smsPerson{}
	for i := range people {
		if people[i].outcome == "delivered" {
			delivered = append(delivered, &people[i])
		}
	}
	rng.Shuffle(len(delivered), func(i, j int) { delivered[i], delivered[j] = delivered[j], delivered[i] })
	for _, p := range delivered {
		if clicks[p.country] > 0 && !p.click && !isNamed(p, named) {
			p.click = true
			clicks[p.country]--
		}
	}
	for i := 0; i < 41; i++ {
		delivered[(i*7)%len(delivered)].machine++
	}
	// Replies.
	reply := func(p *smsPerson, body, keyword string, at time.Duration) {
		p.reply, p.keyword, p.replyAt = body, keyword, at
	}
	reply(named["dana"], "STOP", "STOP", 61*time.Second)
	reply(named["alex"], "Is there parking?", "", 125*time.Second)
	reply(named["olivia"], "HELP", "HELP", 4*time.Minute+10*time.Second)
	reply(named["kim"], "Will the talks be recorded?", "", 6*time.Minute+5*time.Second)
	reply(named["lena"], "Stop", "STOP", 9*time.Minute+20*time.Second)
	reply(named["jordan"], "Can I bring a colleague?", "", 15*time.Minute+2*time.Second)
	more := []struct {
		country, body, keyword string
	}{
		{"US", "STOP", "STOP"}, {"US", "stop", "STOP"}, {"US", "STOP", "STOP"}, {"GB", "STOP", "STOP"}, {"US", "HELP", "HELP"},
		{"US", "Is the livestream free?", ""}, {"US", "Thanks! See you there", ""}, {"GB", "Will slides be shared after?", ""},
		{"US", "What time does registration open?", ""}, {"CA", "Is there a hotel discount?", ""}, {"US", "Can I change my ticket name?", ""},
		{"DE", "Gibt es eine Aufzeichnung?", ""}, {"US", "👍", ""},
	}
	for _, m := range more {
		for _, p := range delivered {
			if p.country == m.country && p.reply == "" && !isNamed(p, named) {
				reply(p, m.body, m.keyword, time.Duration(2+rng.IntN(40))*time.Minute+time.Duration(rng.IntN(60))*time.Second)
				break
			}
		}
	}
}

func isNamed(p *smsPerson, named map[string]*smsPerson) bool {
	for _, n := range named {
		if n == p {
			return true
		}
	}
	return false
}

// smsGroupStart is when a zone group's texts start: 10:00 local on the 14th.
func smsGroupStart(s *Seeder, region, zone string) time.Time {
	return s.smsDay(14, 10, 0, smsLoc(zone))
}

func smsPrice(country string) float64 {
	return map[string]float64{"US": 0.011, "CA": 0.011, "GB": 0.045, "DE": 0.088, "NL": 0.096, "IE": 0.076}[country]
}

const smsIPhoneUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1"

// smsWrite writes the registrants and everything that happened to them.
func smsWrite(ctx context.Context, s *Seeder, st *smsState, people []smsPerson, named map[string]*smsPerson, rng *rand.Rand) error {
	ws := s.WS()
	chicago := smsLoc("America/Chicago")
	summitDay := st.summitDate.Format("2006-01-02")
	consentText := "Text me reminders about Acme Summit 2026. Up to 4 messages. Msg & data rates may apply. Reply HELP for help, STOP to cancel. Terms: news.acme.io/sms-terms"
	// Contacts: the new ones, then the existing ones' numbers.
	var newRows, members, emailConsent [][]any
	var upID []uuid.UUID
	var upPhone, upCountry, upType, upSource []string
	var upAt []time.Time
	for _, p := range people {
		lineType, lineSource := "unknown", ""
		country := ""
		if p.phone != "" {
			country = p.country
			lineType, lineSource = "mobile", "number_plan"
			if strings.HasPrefix(p.phone, "+1") {
				lineType = "unknown"
				// The boards' people: Twilio's lookup said mobile.
				for _, n := range named {
					if n.id == p.id {
						lineType, lineSource = "mobile", "provider"
					}
				}
			}
			if p.state == "landline" {
				lineType, lineSource = "landline", "provider"
			}
		}
		var phoneAt any
		if p.phone != "" {
			phoneAt = p.registered
		}
		if p.isNew {
			attrs, _ := json.Marshal(map[string]any{"summit_date": summitDay})
			ctry := country
			if ctry == "" {
				ctry = "US"
			}
			newRows = append(newRows, []any{p.id, ws.ID, p.email, p.first, p.last, "form:summit", p.tz, ctry, attrs, p.registered, p.registered, "single", p.company,
				"en", p.phone, country, lineType, lineSource, phoneAt})
		} else if p.phone != "" {
			upID = append(upID, p.id)
			upPhone, upCountry, upType, upSource = append(upPhone, p.phone), append(upCountry, country), append(upType, lineType), append(upSource, lineSource)
			upAt = append(upAt, p.registered)
		} else {
			upID = append(upID, p.id)
			upPhone, upCountry, upType, upSource = append(upPhone, ""), append(upCountry, ""), append(upType, "unknown"), append(upSource, "")
			upAt = append(upAt, p.registered)
		}
		members = append(members, []any{st.summit, p.id, "subscribed", "form:summit", p.registered, p.registered})
		ipAddr := netip.MustParseAddr(p.ip)
		emailConsent = append(emailConsent, []any{ID(), p.id, st.summit, "subscribed", "form:summit", ipAddr, "news.acme.io/summit · registration page", p.registered, "1"})
	}
	if err := smsCopy(ctx, s, "contacts", []string{"id", "workspace_id", "email", "first_name", "last_name", "source", "timezone", "country", "attributes",
		"created_at", "updated_at", "consent", "company", "language", "phone", "phone_country", "phone_line_type", "phone_line_source", "phone_set_at"}, newRows); err != nil {
		return err
	}
	// Every demo contact has a score (X13): the new registrants start at 0,
	// with no events yet.
	newIDs := make([]uuid.UUID, 0, len(newRows))
	for _, r := range newRows {
		newIDs = append(newIDs, r[0].(uuid.UUID))
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO contact_scores (contact_id, workspace_id, score, events_part, fields_part, version, computed_at)
		SELECT c.id, c.workspace_id, 0, 0, 0, (SELECT coalesce(max(version), 1) FROM contact_scores WHERE workspace_id = $1), $2
		FROM contacts c WHERE c.workspace_id = $1 AND c.id = ANY($3::uuid[]) ON CONFLICT (contact_id) DO NOTHING`, ws.ID, s.Now, newIDs); err != nil {
		return fmt.Errorf("new registrants' scores: %w", err)
	}
	// The new registrants are active, so the stored count of the segment of
	// active contacts takes them in, as a recount would. The audience's
	// segments are saved by a deferred step, so this one runs after it.
	added := int64(len(newRows))
	s.After(func(ctx context.Context, s *Seeder) error {
		if _, err := s.Tx.Exec(ctx, `UPDATE segments SET cached_count = cached_count + $2,
				tier_counts = jsonb_set(tier_counts, '{active}', to_jsonb(coalesce((tier_counts->>'active')::bigint, 0) + $2))
			WHERE workspace_id = $1 AND name = 'Active in last 30 days'`, s.WS().ID, added); err != nil {
			return fmt.Errorf("active segment's count: %w", err)
		}
		return nil
	})
	if _, err := s.Tx.Exec(ctx, `UPDATE contacts c SET phone = CASE WHEN v.phone = '' THEN c.phone ELSE v.phone END,
			phone_country = CASE WHEN v.phone = '' THEN c.phone_country ELSE v.country END,
			phone_line_type = CASE WHEN v.phone = '' THEN c.phone_line_type ELSE v.lt END,
			phone_line_source = CASE WHEN v.phone = '' THEN c.phone_line_source ELSE v.ls END,
			phone_set_at = CASE WHEN v.phone = '' THEN c.phone_set_at ELSE v.at END,
			attributes = c.attributes || jsonb_build_object('summit_date', $7::text)
		FROM unnest($1::uuid[], $2::text[], $3::text[], $4::text[], $5::text[], $6::timestamptz[]) AS v(id, phone, country, lt, ls, at)
		WHERE c.id = v.id`, upID, upPhone, upCountry, upType, upSource, upAt, summitDay); err != nil {
		return fmt.Errorf("contacts' numbers: %w", err)
	}
	if err := smsCopy(ctx, s, "list_memberships", []string{"list_id", "contact_id", "status", "source", "subscribed_at", "created_at"}, members); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "consent_events", []string{"id", "contact_id", "list_id", "action", "source", "ip", "detail", "occurred_at", "consent_version"}, emailConsent); err != nil {
		return err
	}

	// SMS consent with its evidence, the confirmations and what came of them.
	twilioSender, vonageSender := "+18885550142", "ACME"
	var consents, consentEvents, msgs, events, raws, inbound, sups, links, clicks [][]any
	msgSeq := 0
	twilioSID := func() string {
		msgSeq++
		return fmt.Sprintf("SM%08x%024x", msgSeq*2654435761%0xffffffff, uint64(msgSeq)*0x9e3779b97f4a7c15)
	}
	vonageID := func() string {
		msgSeq++
		return fmt.Sprintf("1800000%09d", msgSeq)
	}
	raw := func(provider uuid.UUID, kind string, at time.Time, contentType string, body []byte, summary string) uuid.UUID {
		id := ID()
		raws = append(raws, []any{id, ws.ID, provider, kind, at, contentType, json.RawMessage(`{}`), body, true, summary, at.Add(time.Second), "", int32(1)})
		return id
	}
	event := func(msg uuid.UUID, kind, code string, detail map[string]any, rawID *uuid.UUID, machine bool, at time.Time) {
		if detail == nil {
			detail = map[string]any{}
		}
		d, _ := json.Marshal(detail)
		events = append(events, []any{ID(), ws.ID, msg, kind, code, d, rawID, machine, at})
	}
	twilioReceipt := func(sid, to, status, code string, at time.Time) (uuid.UUID, string) {
		v := url.Values{"MessageSid": {sid}, "SmsSid": {sid}, "MessageStatus": {status}, "SmsStatus": {status}, "To": {to}, "From": {twilioSender},
			"AccountSid": {twiliotest.AccountSID}, "ApiVersion": {"2010-04-01"}}
		summary := "MessageStatus=" + status
		if code != "" {
			v.Set("ErrorCode", code)
			summary += " · ErrorCode=" + code
		}
		return raw(st.twilio, "receipt", at, "application/x-www-form-urlencoded", []byte(v.Encode()), summary), summary
	}
	vonageReceipt := func(id, to, status, errCode string, price float64, at time.Time) uuid.UUID {
		b, _ := json.Marshal(map[string]string{"msisdn": strings.TrimPrefix(to, "+"), "to": vonageSender, "network-code": "23410", "messageId": id,
			"price": fmt.Sprintf("%.8f", price), "status": status, "scts": at.UTC().Format("0601021504"), "err-code": errCode, "api-key": vonagetest.APIKey,
			"message-timestamp": at.UTC().Format("2006-01-02 15:04:05")})
		summary := "status=" + status
		if errCode != "0" {
			summary += " · err-code=" + errCode
		}
		return raw(st.vonage, "receipt", at, "application/json", b, summary)
	}
	type sent struct {
		id       uuid.UUID
		provider uuid.UUID
		sender   uuid.UUID
	}
	route := func(country string) (sent, string) {
		if country == "US" || country == "CA" {
			return sent{provider: st.twilio, sender: st.tollFree}, twilioSender
		}
		return sent{provider: st.vonage, sender: st.acme}, vonageSender
	}
	marcus := s.Users["marcus@acme.io"]
	gbExtraCharged := false
	optOutReply := st.optOut
	replyMessage := func(p *smsPerson, body string, at time.Time) uuid.UUID {
		r, from := route(p.country)
		id := ID()
		cnt := sms.CountText(body)
		pmid := twilioSID()
		if r.provider == st.vonage {
			pmid = vonageID()
		}
		msgs = append(msgs, []any{id, ws.ID, "sms", "reply", nil, nil, "", nil, &p.id, nil, p.phone, p.country, p.zone, "contact", &r.provider, &r.sender, from,
			body, int32(cnt.Segments), string(cnt.Encoding), nil, 0.0, "delivered", "", "", pmid, nil, nil, int32(1), at, &at, ptrTime(at.Add(3 * time.Second)), nil, at.Add(3 * time.Second)})
		event(id, "sent", "", nil, nil, false, at)
		event(id, "delivered", "", nil, nil, false, at.Add(3*time.Second))
		return id
	}
	inboundRow := func(p *smsPerson, body, kind, keyword string, msg *uuid.UUID, campaign *uuid.UUID, lists []string, replied bool, replyID *uuid.UUID, at time.Time, context string) {
		r, from := route(p.country)
		var rawID uuid.UUID
		if r.provider == st.twilio {
			v := url.Values{"MessageSid": {twilioSID()}, "From": {p.phone}, "To": {from}, "Body": {body}, "NumMedia": {"0"}, "AccountSid": {twiliotest.AccountSID}}
			rawID = raw(r.provider, "inbound", at, "application/x-www-form-urlencoded", []byte(v.Encode()), "Inbound · Body=“"+body+"”")
		} else {
			b, _ := json.Marshal(map[string]string{"msisdn": strings.TrimPrefix(p.phone, "+"), "to": "447700900000", "messageId": vonageID(), "text": body, "type": "text",
				"api-key": vonagetest.APIKey, "message-timestamp": at.UTC().Format("2006-01-02 15:04:05")})
			rawID = raw(r.provider, "inbound", at, "application/json", b, "Inbound · text=“"+body+"”")
		}
		eff, _ := json.Marshal(map[string]any{"lists": lists, "contacts": 1, "replied": replied, "suppressed": kind == "opt_out"})
		inbound = append(inbound, []any{ID(), ws.ID, &r.provider, p.phone, from, body, kind, keyword, msg, campaign, nil, []uuid.UUID{p.id}, eff, replyID, &rawID, at})
		if msg != nil {
			evKind := map[string]string{"opt_out": "opted_out", "help": "help", "reply": "replied"}[kind]
			event(*msg, evKind, "", map[string]any{"keyword": keyword}, &rawID, false, at)
		}
		_ = context
	}
	consentPage := "news.acme.io/summit"
	confirmations := 0
	for i := range people {
		p := &people[i]
		if p.state == "none" || p.state == "phone" {
			continue
		}
		status := "subscribed"
		var stopped *time.Time
		if p.state == "stop" {
			status = "opted_out"
		}
		ipAddr := netip.MustParseAddr(p.ip)
		consentEvents = append(consentEvents, []any{ID(), ws.ID, p.id, &st.summit, "sms", "subscribed", p.phone, "form:summit", consentPage, consentText, "1", "",
			&ipAddr, "", nil, nil, "", p.registered})
		if p.country == "IE" {
			// Held until ACME is approved in Ireland.
			r, from := route(p.country)
			id := ID()
			cnt := sms.CountText(st.confirmation)
			msgs = append(msgs, []any{id, ws.ID, "sms", "confirmation", nil, nil, "", nil, &p.id, &st.summit, p.phone, p.country, p.zone, "contact", &r.provider, &r.sender,
				from, st.confirmation, int32(cnt.Segments), string(cnt.Encoding), nil, 0.0, "held", "sender_pending", "", "", nil, nil, int32(0), p.registered.Add(30 * time.Second),
				nil, nil, nil, p.registered.Add(30 * time.Second)})
			consentEvents = append(consentEvents, []any{ID(), ws.ID, p.id, &st.summit, "sms", "confirmation_sent", p.phone, "reputecast", "Opt-in confirmation", "", "", "",
				nil, "", nil, nil, "held", p.registered.Add(30 * time.Second)})
			consents = append(consents, []any{ws.ID, st.summit, p.id, "sms", status, "form:summit", "", stopped, p.registered})
			continue
		}
		// The confirmation, within a minute of registering (inside the window,
		// or at 09:00 local the next morning).
		loc := smsLoc(p.zone)
		at := p.registered.Add(40 * time.Second)
		lt := at.In(loc)
		if lt.Hour() < 9 || lt.Hour() >= 20 {
			next := time.Date(lt.Year(), lt.Month(), lt.Day(), 9, 0, 12, 0, loc)
			if lt.Hour() >= 20 {
				next = next.AddDate(0, 0, 1)
			}
			at = next
		}
		r, from := route(p.country)
		id := ID()
		r.id = id
		price := smsPrice(p.country)
		cost := price
		if p.country == "GB" && !gbExtraCharged {
			// One UK network charged a little more than the list price.
			cost, gbExtraCharged = 0.048, true
		}
		pmid := twilioSID()
		if r.provider == st.vonage {
			pmid = vonageID()
		}
		msgStatus, reason, code := "delivered", "", ""
		final := at.Add(time.Duration(3+rng.IntN(6)) * time.Second)
		var delivered, failed *time.Time
		switch p.state {
		case "landline":
			msgStatus, reason, code = "undelivered", "landline", "30006"
			failed = &final
		case "invalid":
			msgStatus, reason, code = "failed", "invalid", "21211"
			failed = &final
		default:
			delivered = &final
		}
		cnt := sms.CountText(st.confirmation)
		msgs = append(msgs, []any{id, ws.ID, "sms", "confirmation", nil, nil, "", nil, &p.id, &st.summit, p.phone, p.country, p.zone, "contact", &r.provider, &r.sender,
			from, st.confirmation, int32(cnt.Segments), string(cnt.Encoding), &price, cost, msgStatus, reason, code, pmid, &at, nil, int32(1), at, &at, delivered, failed, final})
		confirmations++
		event(id, "sent", "", nil, nil, false, at)
		var rawID uuid.UUID
		if r.provider == st.twilio {
			rawID, _ = twilioReceipt(pmid, p.phone, map[string]string{"delivered": "delivered", "undelivered": "undelivered", "failed": "failed"}[msgStatus], code, final)
		} else {
			rawID = vonageReceipt(pmid, p.phone, "delivered", "0", price, final)
		}
		evKind := msgStatus
		detail := map[string]any{}
		if reason != "" {
			detail = map[string]any{"reason": reason, "suppressed": true}
		}
		event(id, evKind, code, detail, &rawID, false, final)
		consentEvents = append(consentEvents, []any{ID(), ws.ID, p.id, &st.summit, "sms", "confirmation_sent", p.phone, "reputecast", "Opt-in confirmation", "", "", "",
			nil, "", nil, nil, msgStatus, at})
		switch p.state {
		case "landline", "invalid":
			consentEvents = append(consentEvents, []any{ID(), ws.ID, p.id, &st.summit, "sms", "confirmation_failed", p.phone, "reputecast", "Opt-in confirmation", "", "", "",
				nil, "", nil, nil, reason, final})
			sups = append(sups, []any{ID(), ws.ID, "sms", p.phone, reason, "", "Twilio " + code, "Twilio", &st.twilio, nil, nil, "Opt-in confirmation", nil, final})
		case "stop":
			stopAt := final.Add(time.Duration(2+rng.IntN(20)) * time.Minute)
			word := "STOP"
			if p.phone == "+447700902290" {
				stopAt, word = s.smsDay(2, 18, 22, smsLoc("Europe/London")), "Stop"
			}
			stopped = &stopAt
			status = "opted_out"
			replyID := replyMessage(p, optOutReply, stopAt.Add(2*time.Second))
			inboundRow(p, word, "opt_out", "STOP", &id, nil, []string{"Acme Summit 2026"}, true, &replyID, stopAt, "Opt-in confirmation")
			consentEvents = append(consentEvents, []any{ID(), ws.ID, p.id, &st.summit, "sms", "opted_out", p.phone, "keyword", "Text message", "", "", "STOP",
				nil, "", nil, nil, "Opted out by text", stopAt})
			provName := "Twilio"
			if r.provider == st.vonage {
				provName = "Vonage"
			}
			sups = append(sups, []any{ID(), ws.ID, "sms", p.phone, "opted_out", "STOP", word, provName, &r.provider, nil, nil, "Opt-in confirmation", nil, stopAt})
		}
		consents = append(consents, []any{ws.ID, st.summit, p.id, "sms", status, "form:summit", map[bool]string{true: "STOP", false: ""}[stopped != nil], stopped,
			map[bool]time.Time{true: derefTime(stopped), false: p.registered}[stopped != nil]})
	}
	if confirmations != 871 {
		return fmt.Errorf("demo: %d confirmations, want 871", confirmations)
	}

	// The campaign: 10:00 local in each zone, Twilio at 3 a second and
	// Vonage at 30.
	campaign := st.campaign
	fullBody := func(code string) string {
		return "Acme: " + strings.Replace(st.body, "acme.io/summit/agenda", "go.acme.io/"+code, 1) + " Reply STOP to opt out"
	}
	groups := map[string][]*smsPerson{}
	order := []string{}
	for i := range people {
		p := &people[i]
		if p.state != "receive" {
			continue
		}
		if p.country == "IE" {
			continue
		}
		key := p.region
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], p)
	}
	for _, key := range order {
		list := groups[key]
		// The Twilio group goes in the order people registered.
		for j, p := range list {
			p.position = j
		}
		for _, p := range list {
			start := smsGroupStart(s, key, p.zone)
			rate := 3.0
			if p.country != "US" && p.country != "CA" {
				rate = 30
			}
			p.sentAt = start.Add(time.Duration(float64(p.position) / rate * float64(time.Second)))
		}
	}
	alex := named["alex"]
	alex.sentAt = smsGroupStart(s, "central", alex.zone).Add(38 * time.Second)
	target := "https://acme.io/summit/agenda?utm_campaign=summit-8-days&utm_medium=sms&utm_source=sms"
	clickUA := []string{smsIPhoneUA, "Mozilla/5.0 (Linux; Android 16; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36"}
	machineUA := []string{"facebookexternalhit/1.1 Facebot Twitterbot/1.0", "WhatsApp/2.25.20 A", "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)"}
	twilioSentCallbacks := 0
	for _, key := range order {
		for _, p := range groups[key] {
			r, from := route(p.country)
			id := ID()
			code := texting.LinkCode(campaign, p.id, 0)
			body := fullBody(code)
			cnt := sms.CountText(body)
			price := smsPrice(p.country)
			pmid := twilioSID()
			if r.provider == st.vonage {
				pmid = vonageID()
			}
			sendAfter := smsGroupStart(s, key, p.zone)
			expires := s.smsDay(14, 20, 0, smsLoc(p.zone))
			final := p.sentAt.Add(time.Duration(3+rng.IntN(25)) * time.Second)
			if p == alex {
				final = p.sentAt.Add(3 * time.Second)
			}
			status, reason, pcode := "delivered", "", ""
			var delivered, failed *time.Time
			switch p.outcome {
			case "unreachable":
				status, reason, pcode = "undelivered", "unreachable", "30003"
			case "absent":
				status, reason, pcode = "undelivered", "unreachable", "absent subscriber"
			case "filtered":
				status, reason, pcode = "undelivered", "filtered", "30007"
			case "not_in_service":
				status, reason, pcode = "undelivered", "not_in_service", "30005"
			}
			if status == "delivered" {
				delivered = &final
			} else {
				failed = &final
			}
			zoneSource := "contact"
			if p.tz == "" {
				zoneSource = "area_code"
				if strings.HasPrefix(p.phone, "+1709") {
					zoneSource = "country"
				}
			}
			msgs = append(msgs, []any{id, ws.ID, "sms", "campaign", &campaign, nil, "", nil, &p.id, &st.summit, p.phone, p.country, p.zone, zoneSource, &r.provider, &r.sender,
				from, body, int32(cnt.Segments), string(cnt.Encoding), &price, price * float64(cnt.Segments), status, reason, pcode, pmid, &sendAfter, &expires, int32(1),
				s.smsDay(13, 16, 20, chicago), &p.sentAt, delivered, failed, final})
			links = append(links, []any{code, ws.ID, &id, &campaign, nil, &p.id, "acme.io/summit/agenda", target, s.smsDay(13, 16, 20, chicago),
				s.smsDay(13, 16, 20, chicago).AddDate(1, 0, 0)})
			var sentRaw *uuid.UUID
			if r.provider == st.twilio && twilioSentCallbacks < 652 {
				v := url.Values{"MessageSid": {pmid}, "MessageStatus": {"sent"}, "To": {p.phone}, "From": {from}, "AccountSid": {twiliotest.AccountSID}}
				rid := raw(r.provider, "receipt", p.sentAt.Add(time.Second), "application/x-www-form-urlencoded", []byte(v.Encode()), "MessageStatus=sent")
				sentRaw = &rid
				twilioSentCallbacks++
			}
			event(id, "sent", "", nil, sentRaw, false, p.sentAt)
			var rawID uuid.UUID
			detail := map[string]any{}
			if r.provider == st.twilio {
				rawID, _ = twilioReceipt(pmid, p.phone, status, map[bool]string{true: pcode, false: ""}[status != "delivered"], final)
			} else {
				vs, ec := "delivered", "0"
				if status != "delivered" {
					vs, ec = "failed", "1"
				}
				rawID = vonageReceipt(pmid, p.phone, vs, ec, price, final)
			}
			if reason != "" {
				detail = map[string]any{"reason": reason, "temporary": reason == "unreachable", "suppressed": reason == "not_in_service"}
				if reason == "filtered" {
					detail["filtered"], detail["filteredLimit"] = 1, 3
				}
			}
			event(id, status, map[bool]string{true: pcode, false: ""}[status != "delivered"], detail, &rawID, false, final)
			if reason == "not_in_service" {
				sups = append(sups, []any{ID(), ws.ID, "sms", p.phone, "not_in_service", "", "Twilio 30005", "Twilio", &st.twilio, &campaign, nil, "", nil, final})
			}
			// Clicks: people, and link previews counted apart.
			if p.click {
				at := final.Add(time.Duration(30+rng.IntN(3000)) * time.Second)
				if p == alex {
					at = smsGroupStart(s, "central", p.zone).Add(72 * time.Second)
				}
				clicks = append(clicks, []any{ID(), ws.ID, code, &campaign, &id, &p.id, false, clickUA[rng.IntN(len(clickUA))], at})
				event(id, "clicked", "", map[string]any{"code": code, "url": "acme.io/summit/agenda"}, nil, false, at)
			}
			for m := 0; m < p.machine; m++ {
				at := final.Add(time.Duration(1+rng.IntN(20)) * time.Second)
				clicks = append(clicks, []any{ID(), ws.ID, code, &campaign, &id, &p.id, true, machineUA[rng.IntN(len(machineUA))], at})
				event(id, "clicked", "", map[string]any{"code": code, "url": "acme.io/summit/agenda"}, nil, true, at)
			}
			// Replies.
			if p.reply != "" {
				at := smsGroupStart(s, key, p.zone).Add(p.replyAt)
				switch p.keyword {
				case "STOP":
					replyID := replyMessage(p, optOutReply, at.Add(2*time.Second))
					inboundRow(p, p.reply, "opt_out", "STOP", &id, &campaign, []string{"Acme Summit 2026"}, true, &replyID, at, "")
					for j := range consents {
						if consents[j][2] == p.id {
							stop := at
							consents[j][4], consents[j][6], consents[j][7], consents[j][8] = "opted_out", "STOP", &stop, at
						}
					}
					consentEvents = append(consentEvents, []any{ID(), ws.ID, p.id, &st.summit, "sms", "opted_out", p.phone, "keyword", "Text message", "", "", "STOP",
						nil, "", nil, nil, "Opted out by text", at})
					provName := "Twilio"
					if r.provider == st.vonage {
						provName = "Vonage"
					}
					sups = append(sups, []any{ID(), ws.ID, "sms", p.phone, "opted_out", "STOP", p.reply, provName, &r.provider, &campaign, nil, "", nil, at})
				case "HELP":
					replyID := replyMessage(p, st.help, at.Add(2*time.Second))
					inboundRow(p, p.reply, "help", "HELP", &id, &campaign, nil, true, &replyID, at, "")
				default:
					inboundRow(p, p.reply, "reply", "", &id, &campaign, nil, false, nil, at, "")
				}
			}
		}
	}
	// Ireland: skipped, ACME is pending there.
	for i := range people {
		p := &people[i]
		if p.state != "receive" || p.country != "IE" {
			continue
		}
		r, from := route(p.country)
		msgs = append(msgs, []any{ID(), ws.ID, "sms", "campaign", &campaign, nil, "", nil, &p.id, &st.summit, p.phone, p.country, p.zone, "contact", &r.provider, &r.sender,
			from, nil, int32(1), "gsm7", nil, 0.0, "skipped", "sender_pending", "", "", nil, nil, int32(0), s.smsDay(13, 16, 20, chicago), nil, nil, nil,
			s.smsDay(13, 16, 20, chicago)})
	}
	// Six test texts, all to Marcus: the wizard's and five of the campaign.
	testAt := []time.Time{s.smsDay(1, 10, 31, chicago), s.smsDay(13, 15, 41, chicago), s.smsDay(13, 16, 4, chicago), s.smsDay(13, 16, 6, chicago),
		s.smsDay(13, 16, 15, chicago), s.smsDay(14, 9, 31, chicago)}
	for i, at := range testAt {
		id := ID()
		var camp *uuid.UUID
		body := "Acme: This is a test from ReputeCast. Reply STOP to opt out"
		if i > 0 {
			camp = &campaign
			body = fullBody(texting.LinkCode(id, uuid.Nil, 0))
		}
		cnt := sms.CountText(body)
		price := 0.011
		pmid := twilioSID()
		final := at.Add(3100 * time.Millisecond)
		msgs = append(msgs, []any{id, ws.ID, "sms", "test", camp, nil, "", nil, nil, nil, "+13125550161", "US", "America/Chicago", "area_code", &st.twilio, &st.tollFree,
			twilioSender, body, int32(cnt.Segments), string(cnt.Encoding), &price, price, "delivered", "", "", pmid, nil, nil, int32(1), at, &at, &final, nil, final})
		event(id, "sent", "", nil, nil, false, at)
		rawID, _ := twilioReceipt(pmid, "+13125550161", "delivered", "", final)
		event(id, "delivered", "", nil, &rawID, false, final)
		_ = marcus
	}
	if err := smsCopy(ctx, s, "channel_consents", []string{"workspace_id", "list_id", "contact_id", "channel", "status", "source", "keyword", "stopped_at", "changed_at"}, consents); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "channel_consent_events", []string{"id", "workspace_id", "contact_id", "list_id", "channel", "action", "phone", "source", "page",
		"consent_text", "consent_version", "keyword", "ip", "user_agent", "actor_id", "import_id", "note", "occurred_at"}, consentEvents); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "sms_raw_events", []string{"id", "workspace_id", "provider_id", "kind", "received_at", "content_type", "headers", "body", "signature_ok",
		"summary", "normalized_at", "normalize_error", "events"}, raws); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "channel_messages", []string{"id", "workspace_id", "channel", "kind", "campaign_id", "automation_id", "automation_step", "run_id",
		"contact_id", "list_id", "recipient", "country", "time_zone", "zone_source", "provider_id", "sender_id", "sender", "body", "units", "encoding", "price", "cost",
		"status", "reason", "provider_code", "provider_message_id", "send_after", "expires_at", "attempts", "created_at", "sent_at", "delivered_at", "failed_at",
		"updated_at"}, msgs); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "channel_message_events", []string{"id", "workspace_id", "message_id", "kind", "provider_code", "detail", "raw_event_id", "machine",
		"occurred_at"}, events); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "sms_links", []string{"code", "workspace_id", "message_id", "campaign_id", "automation_id", "contact_id", "original", "target", "created_at",
		"expires_at"}, links); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "sms_link_clicks", []string{"id", "workspace_id", "code", "campaign_id", "message_id", "contact_id", "machine", "user_agent", "occurred_at"},
		clicks); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "sms_inbound", []string{"id", "workspace_id", "provider_id", "phone", "sender", "body", "kind", "keyword", "message_id", "campaign_id",
		"automation_id", "contact_ids", "effect", "reply_message_id", "raw_event_id", "received_at"}, inbound); err != nil {
		return err
	}
	return smsCopy(ctx, s, "channel_suppressions", []string{"id", "workspace_id", "channel", "phone", "reason", "keyword", "original_code", "source", "source_provider_id",
		"campaign_id", "automation_id", "context", "created_by", "created_at"}, sups)
}

func ptrTime(t time.Time) *time.Time { return &t }

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
