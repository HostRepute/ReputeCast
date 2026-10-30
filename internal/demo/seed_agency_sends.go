package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// The agency sample's sending: each client's own providers, domains and
// senders, contacts, campaigns, the two approvals waiting for a client's
// own approver, and the week's two incidents.

type nwProvider struct {
	ws, short string
	p         demoProvider
}

// sending connects each client's own providers (sample data: marked demo,
// they never reach a real service), routes every stream to them and adds
// the clients' domains and senders.
func (n *nwSeed) sending(ctx context.Context) error {
	checked := n.Ago(2 * time.Minute)
	ses := func(region, configSet string, quota int64) demoProvider {
		return demoProvider{adapter: "ses", name: "Amazon SES", label: region, status: state.Healthy, detail: "Production · no enforcement", ipMode: "shared",
			config: map[string]any{"demo": true, "region": region, "configuration_set": configSet, "access": "role", "role_arn": "arn:aws:iam::210987654321:role/reputecast-ses"},
			caps:   capsOf("ses"),
			limits: state.Limits{Bounce: state.Limit{Limit: 5, Review: 5, Pause: 10, Label: "SES"}, Complaint: state.Limit{Limit: 0.1, Review: 0.1, Pause: 0.5, Label: "SES"}},
			health: state.Health{Quota: &state.Quota{Kind: "rolling_24h", Max: quota}, Rate: &state.Rate{Max: 50}, Account: "Production · no enforcement",
				LastSuccessAt: &checked, LastSuccessOp: "GetAccount"},
			pricing:   state.Pricing{Model: "metered", Per1000: 0.10, Source: "AWS pricing"},
			tracking:  state.Tracking{Scope: "Configuration set " + configSet, Mode: "reputecast", CheckedAt: &checked, Detail: "Open and click tracking off in the configuration set."},
			connected: 300 * 24 * time.Hour, lastEvent: 3 * time.Minute}
	}
	postmark := func(server string, monthly float64, label string) demoProvider {
		caps := capsOf("postmark")
		caps.Pricing = fmt.Sprintf("plan, $%.0f per month", monthly)
		return demoProvider{adapter: "postmark", name: "Postmark", label: label, status: state.Healthy, detail: "Approved", ipMode: "shared",
			config: map[string]any{"demo": true, "server": server, "message_stream": "broadcast"}, caps: caps,
			limits:  state.Limits{Bounce: state.Limit{Limit: 10, Label: "Postmark"}, Complaint: state.Limit{Limit: 0.1, Label: "Postmark"}},
			health:  state.Health{Quota: &state.Quota{Kind: "monthly_plan", Max: 50000}, Account: "Approved", LastSuccessAt: &checked, LastSuccessOp: "GET /server"},
			pricing: state.Pricing{Model: "plan", Monthly: monthly}, tracking: state.Tracking{Scope: label, Mode: "reputecast", CheckedAt: &checked},
			connected: 200 * 24 * time.Hour, lastEvent: 9 * time.Minute}
	}
	mailgun := demoProvider{adapter: "mailgun", name: "Mailgun", label: "mg.litwarebooks.example", status: state.Healthy, detail: "Flex plan", ipMode: "shared",
		config: map[string]any{"demo": true, "domain": "mg.litwarebooks.example", "region": "us"}, caps: capsOf("mailgun"),
		limits: state.Limits{Bounce: state.Limit{Limit: 5, Label: "Mailgun"}, Complaint: state.Limit{Limit: 0.1, Label: "Mailgun"}},
		health: state.Health{Account: "Flex plan", LastSuccessAt: &checked, LastSuccessOp: "GET /v3/domains"},
		// Mailgun's API doesn't publish prices: the plan is entered in the workspace.
		pricing:  state.Pricing{Model: "plan", Monthly: 35, Source: "workspace"},
		tracking: state.Tracking{Scope: "mg.litwarebooks.example", Mode: "reputecast", CheckedAt: &checked}, connected: 180 * 24 * time.Hour, lastEvent: 5 * time.Minute}
	provs := []nwProvider{
		{"contoso", "ses", ses("us-west-2", "contoso-broadcasts", 200000)},
		{"fabrikam", "ses", ses("us-east-1", "fabrikam-broadcasts", 600000)},
		{"tailspin", "ses", ses("us-east-1", "tailspin", 100000)},
		{"alpine", "ses", ses("us-east-2", "alpine", 50000)},
		{"coho", "postmark", postmark("Coho Winery", 15, "Broadcasts stream")},
		{"litware", "mailgun", mailgun},
		{"litware", "postmark", postmark("Litware Books", 0, "Automations fallback")},
		{"northwind", "ses", ses("us-east-1", "northwind-notes", 50000)},
	}
	nadia := n.people["nadia"]
	for _, p := range provs {
		id := ID()
		if err := insertProvider(ctx, n.Seeder, id, n.ws[p.ws].ID, p.p, nadia.ID, checked); err != nil {
			return err
		}
		n.providers[p.ws+"/"+p.short] = id
	}
	// Routing: one provider per stream; Litware's automations fail over
	// to Postmark, which also carries its system mail.
	for _, c := range nwClients {
		main := ""
		for _, p := range provs {
			if p.ws == c.key && main == "" {
				main = p.short
			}
		}
		if main == "" {
			continue
		}
		mainID := n.providers[c.key+"/"+main]
		for _, stream := range []string{"broadcast", "automation", "system"} {
			target, fallback := mainID, (*uuid.UUID)(nil)
			if c.key == "litware" {
				pm := n.providers["litware/postmark"]
				switch stream {
				case "automation":
					fallback = &pm
				case "system":
					target = pm
				}
			}
			splits := `[{"provider_id":"` + target.String() + `","weight":100}]`
			if _, err := n.Tx.Exec(ctx, `INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id, updated_by, updated_at)
				VALUES ($1,$2,$3,100,'','{}',$4,$5,$6,$7)`, ID(), n.ws[c.key].ID, stream, json.RawMessage(splits), fallback, nadia.ID, n.onBoards(c.since)); err != nil {
				return err
			}
		}
	}
	// Domains and default senders.
	for _, d := range []struct{ ws, domain, name, local string }{
		{"contoso", "mail.contoso.example", "Contoso Coffee", "hello"},
		{"fabrikam", "mail.fabrikam.example", "Fabrikam Outdoors", "news"},
		{"tailspin", "mail.tailspintoys.example", "Tailspin Toys", "hello"},
		{"alpine", "mail.alpineskihouse.example", "Alpine Ski House", "news"},
		{"coho", "mail.cohowinery.example", "Coho Winery", "cellar"},
		{"litware", "mail.litwarebooks.example", "Litware Books", "letters"},
		{"northwind", "mail.northwind.studio", "Northwind Studio", "notes"},
		{"wwi", "mail.wideworldimporters.example", "", ""},
	} {
		ws := n.ws[d.ws]
		status := "verified"
		if d.name == "" {
			status = "pending"
		}
		domainID := ID()
		if _, err := n.Tx.Exec(ctx, `INSERT INTO sending_domains (id, workspace_id, domain, status, dns_records, auth_summary, last_checked_at, created_at, created_by)
			VALUES ($1,$2,$3,$4,'[]',$5,$6,$7,$8)`, domainID, ws.ID, d.domain, status,
			json.RawMessage(map[bool]string{true: `{"spf":"pass","dkim":"pass","dmarc":"quarantine","bimi":"missing","return_path":"aligned"}`, false: `{}`}[status == "verified"]),
			n.Ago(10*time.Minute), ws.CreatedAt.Add(time.Hour), nadia.ID); err != nil {
			return err
		}
		if d.name == "" {
			continue
		}
		ver := map[string]string{}
		for key, id := range n.providers {
			if strings.HasPrefix(key, d.ws+"/") {
				ver[id.String()] = "verified"
			}
		}
		vb, _ := json.Marshal(ver)
		senderID := ID()
		email := d.local + "@" + d.domain
		if _, err := n.Tx.Exec(ctx, `INSERT INTO senders (id, workspace_id, domain_id, name, email, verification, created_at, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			senderID, ws.ID, domainID, d.name, email, vb, ws.CreatedAt.Add(2*time.Hour), nadia.ID); err != nil {
			return err
		}
		n.senders[d.ws] = senderID
		if _, err := n.Tx.Exec(ctx, `UPDATE workspaces SET default_sender_id = $2, default_automation_sender_id = $2, default_system_sender_id = $2,
			branding = jsonb_set(coalesce(branding, '{}'), '{footer_address}', to_jsonb($3::text)) WHERE id = $1`, ws.ID, senderID, nwAddresses[d.ws]); err != nil {
			return err
		}
	}
	return nil
}

var nwAddresses = map[string]string{
	"contoso": "Contoso Coffee · 1200 Pike St · Seattle, WA 98101", "fabrikam": "Fabrikam Outdoors · 400 N Michigan Ave · Chicago, IL 60611",
	"tailspin": "Tailspin Toys · 77 Water St · New York, NY 10005", "alpine": "Alpine Ski House · 1 Summit Rd · Breckenridge, CO 80424",
	"coho": "Coho Winery · 3200 Vineyard Ln · Napa, CA 94558", "litware": "Litware Books · 15 Beacon St · Boston, MA 02108",
	"northwind": "Northwind Studio · 222 W Merchandise Mart Plaza · Chicago, IL 60654",
}

// The contacts of each client: who's on which list, by position.
type nwListPlan struct {
	name, desc string
	from, to   int // positions [from, to)
	every      int // only every n-th position (0: all)
	offset     int
}

type nwAudience struct {
	ws         string
	total      int
	lists      []nwListPlan
	suppressed [2]int // positions [from, to) on the suppression list
	role       [2]int // positions [from, to) that are role addresses
	domain     string // the client's own customers' company domain for role addresses
}

var nwAudiences = []nwAudience{
	// Holiday blend pre-order: Coffee Club and Newsletter, 60,226 people,
	// 1,212 suppressed and 74 role addresses left out: 58,940.
	{ws: "contoso", total: 64210, lists: []nwListPlan{
		{name: "Newsletter", from: 0, to: 60226, every: 3, offset: 0}, {name: "Newsletter", from: 0, to: 60226, every: 3, offset: 1},
		{name: "Coffee Club", desc: "Monthly roast subscribers", from: 0, to: 60226, every: 3, offset: 0}, {name: "Coffee Club", from: 0, to: 60226, every: 3, offset: 2},
		{name: "Customers", from: 30000, to: 64210}, {name: "VIP", from: 62000, to: 63500}},
		suppressed: [2]int{1000, 1000 + 1212}, role: [2]int{0, 74}, domain: "contoso"},
	{ws: "fabrikam", total: 182406, lists: []nwListPlan{{name: "Newsletter", from: 0, to: 150000}, {name: "Customers", from: 100000, to: 182406},
		{name: "VIP", from: 180000, to: 182406}}, suppressed: [2]int{150000, 151800}, role: [2]int{151800, 151900}, domain: "fabrikam"},
	{ws: "tailspin", total: 41862, lists: []nwListPlan{{name: "Newsletter", from: 0, to: 41862}, {name: "Customers", from: 20000, to: 41862},
		{name: "VIP", from: 40862, to: 41862}}, suppressed: [2]int{41000, 41300}, domain: "tailspin"},
	{ws: "alpine", total: 23115, lists: []nwListPlan{{name: "Newsletter", from: 0, to: 20000}, {name: "Guests", from: 10000, to: 23115}},
		suppressed: [2]int{19700, 19900}, domain: "alpine"},
	// Wine club: November box goes to the 3,412 members of the Wine club.
	{ws: "coho", total: 12940, lists: []nwListPlan{{name: "Newsletter", from: 0, to: 10000}, {name: "Guests", from: 5000, to: 12940},
		{name: "Wine club", desc: "Club members, shipped four times a year", from: 9000, to: 12412}}, suppressed: [2]int{8800, 8900}, domain: "coho"},
	{ws: "litware", total: 8320, lists: []nwListPlan{{name: "Newsletter", from: 0, to: 8320}, {name: "Book club", from: 0, to: 2500}},
		suppressed: [2]int{8200, 8240}, domain: "litware"},
	{ws: "northwind", total: 3410, lists: []nwListPlan{{name: "Studio notes", desc: "The studio's monthly letter", from: 0, to: 3410}},
		suppressed: [2]int{3390, 3410}, domain: "northwind"},
}

var (
	nwFirst = []string{"anna", "ben", "carla", "david", "elena", "felix", "grace", "hugo", "ines", "jamal", "kara", "liam", "maya", "noah", "olga", "pablo",
		"quinn", "rosa", "sam", "tara", "umar", "vera", "will", "xena", "yusuf", "zoe"}
	nwLast = []string{"adams", "baker", "chen", "diaz", "evans", "fox", "garcia", "hill", "ito", "jones", "kim", "lopez", "moore", "nguyen", "ortiz", "patel",
		"reed", "silva", "taylor", "wong"}
	nwMailboxes = []struct{ key, domain string }{{"gmail", "gmail.com"}, {"outlook", "outlook.com"}, {"yahoo", "yahoo.com"}, {"icloud", "icloud.com"}, {"other", "comcast.net"}}
	nwShares    = []int{46, 21, 9, 8, 16}
)

// contacts loads each client's contacts by COPY, with their lists and the
// suppression list. j.doe@example.com is in two clients (the contact
// search of the boards).
func (n *nwSeed) contacts(ctx context.Context) error {
	// Lists only the client has (the template brought the others).
	for _, a := range nwAudiences {
		if n.lists[a.ws] == nil {
			n.lists[a.ws] = map[string]uuid.UUID{}
		}
		for _, l := range a.lists {
			if _, ok := n.lists[a.ws][l.name]; ok {
				continue
			}
			id := ID()
			if _, err := n.Q.CreateList(ctx, store.CreateListParams{ID: id, WorkspaceID: n.ws[a.ws].ID, Name: l.name, Description: l.desc, DoubleOptIn: true}); err != nil {
				return err
			}
			n.lists[a.ws][l.name] = id
		}
	}
	for _, a := range nwAudiences {
		if err := n.audience(ctx, a); err != nil {
			return fmt.Errorf("%s: %w", a.ws, err)
		}
	}
	return nil
}

func (n *nwSeed) audience(ctx context.Context, a nwAudience) error {
	if d := n.scale; d > 1 {
		// A smaller sample: every position divided, the same lists and rules.
		a.total /= d
		lists := make([]nwListPlan, len(a.lists))
		for i, l := range a.lists {
			l.from, l.to = l.from/d, l.to/d
			lists[i] = l
		}
		a.lists = lists
		a.suppressed = [2]int{a.suppressed[0] / d, a.suppressed[1] / d}
		a.role = [2]int{a.role[0] / d, a.role[1] / d}
	}
	ws := n.ws[a.ws]
	start := ws.CreatedAt
	span := n.Now.Sub(start)
	tz := ws.Timezone
	var contacts, members, suppressions [][]any
	mbIdx := make([]int, 0, 100)
	for i, s := range nwShares {
		for range s {
			mbIdx = append(mbIdx, i)
		}
	}
	for i := 0; i < a.total; i++ {
		id := ID()
		first, last := nwFirst[i%len(nwFirst)], nwLast[(i/len(nwFirst))%len(nwLast)]
		mb := nwMailboxes[mbIdx[(i*37)%len(mbIdx)]]
		email := fmt.Sprintf("%s.%s%d@%s", first, last, i, mb.domain)
		role := i >= a.role[0] && i < a.role[1]
		if role {
			email = fmt.Sprintf("orders@%s-shop%d.example", a.domain, i)
		}
		created := start.Add(time.Duration(float64(span) * float64(i) / float64(a.total+1)))
		if created.After(n.Now) {
			created = n.Now.Add(-time.Hour)
		}
		jdoe := i == 102 && (a.ws == "contoso" || a.ws == "fabrikam")
		if jdoe {
			email, first, last = "j.doe@example.com", "jordan", "doe"
			if a.ws == "contoso" {
				created = n.onBoards(date(2025, 2, 2))
			}
		}
		tier := "active"
		switch i % 10 {
		case 7:
			tier = "cooling"
		case 9:
			tier = "at_risk"
		}
		contacts = append(contacts, []any{id, ws.ID, email, strings.ToUpper(first[:1]) + first[1:], strings.ToUpper(last[:1]) + last[1:], "active", tier, "import",
			tz, "US", mb.key, "double", role, created, created, "en"})
		for _, l := range a.lists {
			if i < l.from || i >= l.to || (l.every > 0 && (i-l.from)%l.every != l.offset) {
				continue
			}
			list := n.lists[a.ws][l.name]
			status, unsub := "subscribed", (*time.Time)(nil)
			if jdoe && a.ws == "fabrikam" && l.name == "Newsletter" {
				status = "unsubscribed"
				t := n.onBoards(date(2026, 9, 3))
				unsub = &t
			}
			members = append(members, []any{list, id, status, "import", created, unsub, created})
		}
		if i >= a.suppressed[0] && i < a.suppressed[1] {
			reason := []string{"hard_bounce", "unsubscribe", "complaint", "unsubscribe"}[i%4]
			suppressions = append(suppressions, []any{ID(), ws.ID, email, reason, "reputecast", created.Add(24 * time.Hour)})
		}
	}
	if _, err := n.Tx.CopyFrom(ctx, pgx.Identifier{"contacts"}, []string{"id", "workspace_id", "email", "first_name", "last_name", "status", "tier", "source",
		"timezone", "country", "mailbox_provider", "consent", "role_address", "created_at", "updated_at", "language"}, pgx.CopyFromRows(contacts)); err != nil {
		return err
	}
	if _, err := n.Tx.CopyFrom(ctx, pgx.Identifier{"list_memberships"}, []string{"list_id", "contact_id", "status", "source", "subscribed_at", "unsubscribed_at", "created_at"},
		pgx.CopyFromRows(members)); err != nil {
		return err
	}
	_, err := n.Tx.CopyFrom(ctx, pgx.Identifier{"suppressions"}, []string{"id", "workspace_id", "email", "reason", "source", "created_at"}, pgx.CopyFromRows(suppressions))
	return err
}

// nwCampaign is a campaign of the sample.
type nwCampaign struct {
	ws, name, status, subject, preheader string
	lists                                []string
	count                                int64
	at                                   time.Time // the send (scheduled, approval) or when it went out (sent)
	tz                                   string
	by                                   string
	src                                  string
	humanOpens                           float64 // sent: % of delivered
}

func (n *nwSeed) campaigns(ctx context.Context) error {
	blend, err := n.nwImage(ctx, "contoso", "holiday-blend.png")
	if err != nil {
		return err
	}
	la := mustLoc("America/Los_Angeles")
	inLA := func(days, hour, minute int) time.Time {
		d := n.at(days, 12, 0).In(la)
		return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, la)
	}
	list := []nwCampaign{
		{ws: "contoso", name: "Holiday blend pre-order", status: "approval", subject: "The Holiday blend is back: pre-order now", preheader: "Roasted in small batches, shipping 18 Nov",
			lists: []string{"Coffee Club", "Newsletter"}, count: 58940, at: inLA(19, 8, 0), tz: "America/Los_Angeles", by: "mei", src: nwHolidayBlendMJML(blend)},
		{ws: "contoso", name: "October roast club", status: "sent", subject: "October's roast: Ethiopia Guji", lists: []string{"Coffee Club", "Newsletter"}, count: 57812,
			at: inLA(-9, 8, 0), by: "mei", humanOpens: 41.8},
		{ws: "contoso", name: "Cold brew is back", status: "sent", subject: "Cold brew season is back", lists: []string{"Coffee Club", "Newsletter"}, count: 57390,
			at: inLA(-23, 8, 0), by: "mei", humanOpens: 39.6},
		{ws: "fabrikam", name: "Weekend gear sale, by email and text", status: "scheduled", subject: "This weekend: 25% off packs and tents", lists: []string{"Newsletter", "Customers"},
			count: 141220, at: n.ahead(1, 9, 0), tz: "America/Chicago", by: "mei"},
		{ws: "fabrikam", name: "Fall clearance", status: "sent", subject: "Fall clearance: up to 50% off", lists: []string{"Newsletter", "Customers"}, count: 138904,
			at: n.at(-2, 7, 0), by: "mei", humanOpens: 33.4},
		{ws: "fabrikam", name: "Holiday gift guide", status: "draft", subject: "Gifts for everyone who loves the outdoors", lists: []string{"Newsletter"}, by: "ravi",
			at: n.Ago(26 * time.Hour)},
		{ws: "tailspin", name: "Holiday catalog", status: "draft", subject: "The holiday catalog is here", lists: []string{"Newsletter"}, by: "ella", at: n.Ago(30 * time.Hour)},
		{ws: "tailspin", name: "Back to school picks", status: "sent", subject: "Back to school, sorted", lists: []string{"Newsletter"}, count: 40211, at: n.at(-44, 9, 0), by: "ella",
			humanOpens: 37.2},
		{ws: "alpine", name: "Season pass early bird", status: "scheduled", subject: "Season passes: early-bird prices end Sunday", lists: []string{"Newsletter"}, count: 19804,
			at: n.ahead(0, 17, 0), tz: "America/Chicago", by: "ella"},
		{ws: "coho", name: "Wine club: November box", status: "approval", subject: "Your November box: three reds for the season", lists: []string{"Wine club"}, count: 3412,
			at: n.ahead(4, 10, 0), tz: "America/Chicago", by: "mei"},
		{ws: "litware", name: "Fall reading list", status: "sent", subject: "Twelve books for the fall", lists: []string{"Newsletter"}, count: 8104, at: n.at(-7, 8, 0), by: "ella",
			humanOpens: 48.9},
		{ws: "northwind", name: "Studio notes #18", status: "scheduled", subject: "Studio notes #18: what we shipped this month", lists: []string{"Studio notes"}, count: 3390,
			at: n.ahead(7, 9, 0), tz: "America/Chicago", by: "nadia"},
		{ws: "northwind", name: "Studio notes #17", status: "sent", subject: "Studio notes #17", lists: []string{"Studio notes"}, count: 3388, at: n.at(-26, 9, 0), by: "nadia",
			humanOpens: 52.3},
	}
	for _, c := range list {
		if err := n.campaign(ctx, c); err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
	}
	// Tailspin's paused countdown flow (search results) and Coho's saved
	// section (Mei's search for "holiday").
	tailspin := n.ws["tailspin"]
	ella := n.people["ella"]
	auto := ID()
	g := nwFlow(apiEvent("holiday_countdown"), nwSend("h1", "Countdown"))
	if _, err := n.Tx.Exec(ctx, `INSERT INTO automations (id, workspace_id, name, description, status, trigger, live_version, draft_version, created_by, created_at, updated_at,
		paused_at, paused_by, pause_reason, updated_by) VALUES ($1,$2,'Holiday countdown','Twelve days of deals before the holidays','paused',$3,1,1,$4,$5,$6,$6,$4,'Waiting for the holiday catalog',$4)`,
		auto, tailspin.ID, json.RawMessage(`{"kind":"api_event","event":"holiday_countdown"}`), ella.ID, n.at(-60, 10, 0), n.at(-5, 11, 0)); err != nil {
		return err
	}
	if _, err := n.Tx.Exec(ctx, `INSERT INTO automation_versions (automation_id, version, graph, created_by, created_at, published_by, published_at, summary) VALUES ($1,1,$2,$3,$4,$3,$4,'')`,
		auto, g, ella.ID, n.at(-60, 10, 0)); err != nil {
		return err
	}
	section := nwBannerSec("Order by 18 Dec for delivery before the holidays", "#9D174D", "#FFFFFF")
	it, err := nwItem("Holiday shipping banner", section, true)
	if err != nil {
		return err
	}
	_, err = n.Tx.Exec(ctx, `INSERT INTO templates (id, workspace_id, kind, name, description, mjml, html, thumb, size_bytes, created_by, updated_by, created_at, updated_at)
		VALUES ($1,$2,'section','Holiday shipping banner','',$3,'',$4,0,$5,$5,$6,$6)`, ID(), n.ws["coho"].ID, section, it.Thumb, n.people["mei"].ID, n.at(-12, 15, 0))
	return err
}

func mustLoc(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return l
}

// campaign writes one campaign with its content, route and counters.
func (n *nwSeed) campaign(ctx context.Context, c nwCampaign) error {
	ws := n.ws[c.ws]
	by := n.people[c.by]
	ids := []string{}
	for _, l := range c.lists {
		ids = append(ids, n.lists[c.ws][l].String())
	}
	audience, _ := json.Marshal(map[string]any{"lists": ids})
	src := c.src
	if src == "" {
		src = nwDoc(c.subject, nwHeaderSec(ws.Name, "#0B1220"),
			nwTextSec(c.subject, "Hi {{ first_name | default: \"there\" }}, "+composerXMLText(c.preheader+" A few lines about this email.")), nwFooterSec(nwAddresses[c.ws]))
	}
	html, err := nwCompile(src)
	if err != nil {
		return err
	}
	id := ID()
	created := c.at.Add(-5 * 24 * time.Hour)
	if c.status == "draft" {
		created = c.at.Add(-3 * 24 * time.Hour)
	}
	if created.After(n.Now) {
		created = n.Now.Add(-48 * time.Hour)
	}
	var count *int64
	if c.count > 0 {
		count = &c.count
	}
	schedule := json.RawMessage(`{}`)
	var scheduledAt, started, completed *time.Time
	switch c.status {
	case "scheduled", "approval":
		schedule, _ = json.Marshal(map[string]any{"mode": "at", "at": c.at.UTC().Format(time.RFC3339), "timezone": c.tz})
		if c.status == "scheduled" {
			t := c.at
			scheduledAt = &t
		}
	case "sent":
		schedule = json.RawMessage(`{"mode":"at"}`)
		t, done := c.at, c.at.Add(time.Duration(float64(c.count)/40/60)*time.Minute+10*time.Minute)
		scheduledAt, started, completed = &t, &t, &done
	}
	hash := campaigns.ContentHash(c.subject, c.preheader, src, html, nil)
	updated := created.Add(2 * time.Hour)
	if completed != nil {
		updated = *completed
	}
	if _, err := n.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, audience, audience_count, mjml, html, plain_text, version,
		schedule, created_by, updated_by, created_at, updated_at, scheduled_at, started_by, started_at, completed_at, content_hash, autosaved_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,3,$13,$14,$14,$15,$16,$17,$18,$19,$20,$21,$16)`,
		id, ws.ID, c.name, c.status, c.subject, c.preheader, n.senders[c.ws], audience, count, src, html, contentcheck.PlainText(html), schedule, by.ID, created, updated,
		scheduledAt, map[bool]*uuid.UUID{true: &by.ID, false: nil}[started != nil], started, completed, hash); err != nil {
		return err
	}
	if _, err := n.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, plain_text, saved_by, saved_at, label, source, size_bytes)
		VALUES ($1,3,$2,$3,$4,$5,$6,$7,$8,'','autosave',$9)`, id, c.subject, c.preheader, src, html, contentcheck.PlainText(html), by.ID, updated, len(html)); err != nil {
		return err
	}
	n.Campaigns[c.ws+"/"+c.name] = id
	provider := n.mainProvider(c.ws)
	if provider == uuid.Nil || c.count == 0 {
		return nil
	}
	cost := int64(0)
	if strings.HasSuffix(n.providerKey(provider), "/ses") {
		cost = c.count * 100
	}
	status := "pending"
	if c.status == "sent" {
		status = "done"
	}
	if _, err := n.Tx.Exec(ctx, `INSERT INTO campaign_routes (campaign_id, provider_id, planned, status, cost_micros) VALUES ($1,$2,$3,$4,$5)`, id, provider, c.count, status, cost); err != nil {
		return err
	}
	if c.status != "sent" {
		return nil
	}
	weights := make([]float64, len(MailboxShare))
	for i, mb := range MailboxShare {
		weights[i] = mb.Share
	}
	for i, sent := range Split(c.count, weights) {
		delivered := int64(math.Round(float64(sent) * 0.991))
		human := int64(math.Round(float64(delivered) * c.humanOpens / 100))
		// Gmail and iCloud never report complaints (review F-2).
		complained := delivered / 4000
		if !mailbox.ReportsComplaints(MailboxShare[i].Key) {
			complained = 0
		}
		if _, err := n.Tx.Exec(ctx, `INSERT INTO campaign_counters (campaign_id, provider_id, mailbox_provider, sent, delivered, hard_bounced, soft_bounced, complained,
			opened_human, opened_machine, clicked, unsubscribed) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			id, provider, MailboxShare[i].Key, sent, delivered, (sent-delivered)*2/3, (sent-delivered)-(sent-delivered)*2/3, complained, human, delivered/9,
			delivered*6/100, delivered/500); err != nil {
			return err
		}
	}
	return nil
}

// mainProvider is the provider of a workspace's Broadcasts.
func (n *nwSeed) mainProvider(ws string) uuid.UUID {
	for _, short := range []string{"ses", "postmark", "mailgun"} {
		if id, ok := n.providers[ws+"/"+short]; ok && (ws != "litware" || short != "postmark") {
			return id
		}
	}
	return uuid.Nil
}

func (n *nwSeed) providerKey(id uuid.UUID) string {
	for k, v := range n.providers {
		if v == id {
			return k
		}
	}
	return ""
}

// nwHolidayBlendMJML is Contoso Coffee's "Holiday blend pre-order" as the
// AgencyClientApproval board shows it.
func nwHolidayBlendMJML(img string) string {
	return `<mjml><mj-head><mj-title>The Holiday blend is back: pre-order now</mj-title><mj-preview>Roasted in small batches, shipping 18 Nov</mj-preview>
<mj-attributes><mj-all font-family="Georgia, 'Times New Roman', serif"></mj-all><mj-text color="#1F2937" font-size="15px" line-height="1.55" padding="0px 0px 12px"></mj-text></mj-attributes></mj-head>
<mj-body width="600px" background-color="#F5F1EA">
<mj-section background-color="#FFFFFF" padding="16px 24px"><mj-group><mj-column width="12%" vertical-align="middle"><mj-text padding="0px" font-family="Helvetica, Arial, sans-serif" font-size="12px" font-weight="700" color="#FFFFFF" container-background-color="#92400E" align="center" line-height="28px">CC</mj-text></mj-column>
<mj-column width="88%" vertical-align="middle"><mj-text padding="0px 0px 0px 10px" font-family="Helvetica, Arial, sans-serif" font-size="15px" font-weight="700" color="#1F2937">Contoso Coffee</mj-text></mj-column></mj-group></mj-section>
<mj-section background-color="#FFFFFF" padding="0px"><mj-column><mj-image src="` + img + `" alt="A bag of the Holiday blend 2026 next to a cup" height="200px" padding="0px"></mj-image></mj-column></mj-section>
<mj-section background-color="#FFFFFF" padding="24px 32px 28px"><mj-column>
<mj-text font-size="26px" font-weight="700" color="#111827" line-height="1.2">The Holiday blend is back</mj-text>
<mj-text>Dark chocolate, orange peel and a little cinnamon, roasted in small batches in Seattle. Pre-order now and we ship on 18 Nov, before anyone else gets a bag.</mj-text>
<mj-text><strong>$18.50</strong> a 12 oz bag · free shipping on two or more</mj-text>
<mj-button href="https://contoso.example/holiday-blend" background-color="#92400E" color="#FFFFFF" font-family="Helvetica, Arial, sans-serif" font-size="14px" font-weight="700" border-radius="4px" inner-padding="11px 18px" align="left" padding="4px 0px 0px">Pre-order the Holiday blend</mj-button>
</mj-column></mj-section>
<mj-section padding="14px 32px 24px"><mj-column><mj-text font-family="Helvetica, Arial, sans-serif" font-size="12px" color="#4B5563" padding="0px">Contoso Coffee · 1200 Pike St · Seattle, WA 98101<br>You get this because you're in the Coffee Club. <a href="{{ unsubscribe_url }}" style="color:#4B5563">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#4B5563">Preferences</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`
}

// approvals asks for the two sends waiting for a client's own approver:
// Coho Winery's Wine club box (Daniela Ortiz) and Contoso Coffee's Holiday
// blend (Jane Morales, with Ravi's and Mei's comments).
func (n *nwSeed) approvals(ctx context.Context) error {
	mei, ravi := n.people["mei"], n.people["ravi"]
	for _, a := range []struct {
		ws, name, note string
		ago            time.Duration
	}{
		{"coho", "Wine club: November box", "Three reds this time; the tasting notes are Daniela's from last week.", 19 * time.Hour},
		{"contoso", "Holiday blend pre-order", "Prices are final. The photo is the new bag.", 17*time.Hour + 40*time.Minute},
	} {
		reqID, err := n.approval(ctx, a.ws, a.name, a.note, n.Ago(a.ago))
		if err != nil {
			return fmt.Errorf("%s: %w", a.name, err)
		}
		if a.ws != "contoso" {
			continue
		}
		for _, cm := range []struct {
			by   store.User
			body string
			at   time.Duration
		}{
			{ravi, "The bag photo is from Tuesday's shoot, so it's the 2026 design.", a.ago - 28*time.Minute},
			{mei, "Thanks Ravi. Jane, the send is set for 3 Nov at 08:00 your time, like last year.", a.ago - 31*time.Minute},
		} {
			if _, err := n.Tx.Exec(ctx, `INSERT INTO approval_comments (id, request_id, author_id, body, created_at) VALUES ($1,$2,$3,$4,$5)`,
				ID(), reqID, cm.by.ID, cm.body, n.Ago(cm.at)); err != nil {
				return err
			}
		}
	}
	return nil
}

// approval stores a complete Send Readiness run and Mei's request under
// client sign-off.
func (n *nwSeed) approval(ctx context.Context, wsKey, name, note string, requested time.Time) (uuid.UUID, error) {
	ws := n.ws[wsKey]
	if err := n.Tx.QueryRow(ctx, `SELECT approval_policy FROM workspaces WHERE id = $1`, ws.ID).Scan(&ws.ApprovalPolicy); err != nil {
		return uuid.Nil, err
	}
	id := n.Campaigns[wsKey+"/"+name]
	mei := n.people["mei"]
	var subject, preheader, src, html, hash string
	var audience, schedule json.RawMessage
	var version int
	var senderID uuid.UUID
	if err := n.Tx.QueryRow(ctx, `SELECT subject, preheader, mjml, html, content_hash, audience, schedule, version, sender_id FROM campaigns WHERE id = $1`, id).
		Scan(&subject, &preheader, &src, &html, &hash, &audience, &schedule, &version, &senderID); err != nil {
		return uuid.Nil, err
	}
	var senderName, senderEmail, domain string
	if err := n.Tx.QueryRow(ctx, `SELECT s.name, s.email, d.domain FROM senders s JOIN sending_domains d ON d.id = s.domain_id WHERE s.id = $1`, senderID).
		Scan(&senderName, &senderEmail, &domain); err != nil {
		return uuid.Nil, err
	}
	rs := &readinessSeed{s: n.Seeder, ws: ws, images: map[string]string{}}
	aud, err := (&readiness.Service{Q: n.Q, Now: func() time.Time { return requested }}).Audience(ctx, &ws, audience)
	if err != nil {
		return uuid.Nil, err
	}
	ranAt := requested.Add(-6 * time.Minute)
	checker := &campaigns.Service{Q: n.Q, Checker: &contentcheck.Checker{Blocklist: contentcheck.StoreBlocklist{Q: n.Q}, Prober: ReadinessProberOverlay(nil), Now: func() time.Time { return ranAt }}}
	chk, err := checker.Checks(ctx, &ws, id, campaigns.ComposerChecksRequest{Subject: subject, Preheader: preheader, MJML: src, HTML: html, ProbeLinks: true})
	if err != nil {
		return uuid.Nil, err
	}
	sched := readiness.ParseSchedule(schedule)
	start, _ := time.Parse(time.RFC3339, sched.At)
	loc := state.Location(ws.Timezone)
	provider := n.mainProvider(wsKey)
	share := n.share(wsKey, provider, aud.Final)
	f := readiness.Facts{Now: ranAt, Location: loc, Workspace: ws.Slug, MailboxRates: readiness.DefaultMailboxRates,
		Campaign: readiness.CampaignFacts{ID: id.String(), Name: name, Subject: subject, Preheader: preheader, Version: version, CompiledAt: &ranAt, SenderName: senderName,
			SenderEmail: senderEmail, SenderDomain: domain, StartAt: start, ScheduleMode: "at"},
		Audience: aud,
		Route:    readiness.RouteFacts{Stream: "broadcast", Policy: "pause_and_ask", Shares: []readiness.ShareFacts{share}},
		Reputation: readiness.ReputationFacts{Status: readiness.Pass, Via: "builtin", Sources: 15, CheckedAt: &ranAt, Listings: []readiness.ListingFacts{},
			DomainSources: []string{"Spamhaus DBL", "SURBL", "URIBL"}, IPSources: []string{}},
		Auth: readiness.AuthFacts{Domain: domain, Checked: true, CheckedAt: &ranAt, SPFRecord: "v=spf1 include:" + nwSPFInclude(share.Adapter) + " -all",
			SPF: []readiness.ProviderAuthFacts{{Short: share.Short, OK: true}}, DKIM: []readiness.ProviderAuthFacts{{Short: share.Short, OK: true, Detail: "DKIM verified", KeyBits: 2048}},
			MailFrom:    []readiness.ProviderAuthFacts{{Short: share.Short, OK: true, Domain: "bounce." + domain}},
			DMARCStatus: "pass", DMARCPolicy: "quarantine", DMARCRecord: "v=DMARC1; p=quarantine; adkim=s", DMARCAligned: f64(100), BIMIStatus: "missing"},
		Content: readiness.ContentFacts{Checked: true, Report: chk.ContentCheckReport, Address: nwAddresses[wsKey], InlineCSSBytes: 6 * 1024,
			URIBlocklists: readiness.URIBlocklists, OwnDomain: strings.TrimPrefix(domain, "mail.")},
		Compliance: readiness.ComplianceFacts{UnsubscribeURL: n.Cfg.HTTP.PublicURL + "/u/{token}", HTTPS: true, ComplaintRate: f64(0.02)},
	}
	res, runID, err := rs.storeRun(ctx, id, f, ranAt, &mei.ID, readiness.TriggerApproval)
	if err != nil {
		return uuid.Nil, err
	}
	labels := aud.Lists
	snap := approvals.Snapshot{Version: version, ContentHash: hash, Subject: subject, Preheader: preheader, SenderID: senderID.String(),
		From: senderName + " <" + senderEmail + ">", Links: approvals.Links(src, html), Sections: approvals.Sections(src), Audience: audience,
		AudienceLabels: labels, AudienceCount: aud.Final, AudienceBaseline: aud.Final, Suppressed: aud.Suppressed, Role: aud.Role, Schedule: schedule,
		Thresholds: readiness.ParseThresholds(json.RawMessage(readinessThresholds), nil), AuthorRole: "Admin",
		Rules: []approvals.ApprovalRuleMatch{{Key: approvals.RuleClientSignoff, Label: "Client sign-off: one approval from a Client approver", Short: "client sign-off", Badge: "Client sign-off"}}}
	rs.snapshotRun(&snap, res, runID, ranAt)
	if err := rs.routeKey(ctx, &snap); err != nil {
		return uuid.Nil, err
	}
	reqID := ID()
	if err := rs.insertRequest(ctx, reqID, id, mei.ID, requested, note, "pending", snap, &runID, nil, nil); err != nil {
		return uuid.Nil, err
	}
	// The client's approvers hear about it.
	for _, m := range n.clientApprovers(wsKey) {
		if _, err := n.Tx.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, created_at, email_status)
			VALUES ($1,$2,$3,'approval_requested','warn','Approval requested',$4,$5,$6,$7,'sent')`, ID(), m.ID, ws.ID,
			fmt.Sprintf("Mei Lin asked you to approve “%s” · %s recipients", name, groupThousands(aud.Final)),
			fmt.Sprintf("Mei Lin from Northwind Studio asked for approval: “%s” · %s recipients · rule: client sign-off", name, groupThousands(aud.Final)),
			"/w/"+ws.Slug+"/approvals/"+reqID.String(), requested); err != nil {
			return uuid.Nil, err
		}
	}
	return reqID, nil
}

func (n *nwSeed) clientApprovers(wsKey string) []store.User {
	out := []store.User{}
	for _, c := range nwClients {
		if c.key != wsKey {
			continue
		}
		for who, role := range c.people {
			if role == "client_approver" {
				out = append(out, n.people[who])
			}
		}
	}
	return out
}

func nwSPFInclude(adapter string) string {
	switch adapter {
	case "postmark":
		return "spf.mtasv.net"
	case "mailgun":
		return "mailgun.org"
	}
	return "amazonses.com"
}

// share is a provider's share of a route, as Send Readiness sees it.
func (n *nwSeed) share(wsKey string, id uuid.UUID, recipients int64) readiness.ShareFacts {
	key := n.providerKey(id)
	out := readiness.ShareFacts{ProviderID: id.String(), Recipients: recipients, Percent: 100, Status: state.Healthy, Available: true, Rate: 50,
		Bounce: f64(0.4), Complaint: f64(0.02), BulkAllowed: true, Account: "Production · no enforcement", ConnectedAt: n.Ago(300 * 24 * time.Hour),
		Tracking: state.Tracking{Mode: "reputecast"}}
	switch {
	case strings.HasSuffix(key, "/postmark"):
		out.Name, out.Short, out.Adapter, out.Label = "Postmark", "Postmark", "postmark", "Broadcasts stream"
		out.Pricing, out.Account, out.Terms = state.Pricing{Model: "plan", Monthly: 15}, "Approved", "yes, on the Broadcasts stream"
		out.Limits = state.Limits{Bounce: state.Limit{Limit: 10, Label: "Postmark"}, Complaint: state.Limit{Limit: 0.1, Label: "Postmark"}}
	default:
		region := map[string]string{"contoso": "us-west-2", "fabrikam": "us-east-1", "alpine": "us-east-2"}[wsKey]
		if region == "" {
			region = "us-east-1"
		}
		out.Name, out.Short, out.Adapter, out.Label = "Amazon SES", "SES", "ses", region
		out.Pricing, out.Terms, out.TermsKind = state.Pricing{Model: "metered", Per1000: 0.10}, "yes, with production access", "productionAccess"
		out.Limits = state.Limits{Bounce: state.Limit{Limit: 5, Review: 5, Pause: 10, Label: "SES"}, Complaint: state.Limit{Limit: 0.1, Review: 0.1, Pause: 0.5, Label: "SES"}}
		out.Quota = &state.Quota{Kind: "rolling_24h", Max: 200000}
	}
	return out
}

// groupThousands writes 58940 as 58,940.
func groupThousands(v int64) string {
	s := fmt.Sprint(v)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// incidents are the week's two: Fabrikam's "Fall clearance" auto-paused on
// Yahoo deferrals (Mei resumed it at a lower rate), and Litware's Mailgun
// outage, when automations failed over to Postmark on their own.
func (n *nwSeed) incidents(ctx context.Context) error {
	mei := n.people["mei"]
	fab, lit := n.ws["fabrikam"], n.ws["litware"]
	camp := n.Campaigns["fabrikam/Fall clearance"]
	ses, mg, pm := n.providers["fabrikam/ses"], n.providers["litware/mailgun"], n.providers["litware/postmark"]
	tripped, resumed := n.at(-2, 7, 42), n.at(-2, 8, 30)
	if _, err := n.Tx.Exec(ctx, `INSERT INTO incidents (id, workspace_id, occurred_at, kind, severity, title, detail, campaign_id, provider_id, resolved_at, source)
		VALUES ($1,$2,$3,'auto_pause','warn','“Fall clearance” auto-paused','Yahoo deferrals reached 4.1% (limit 3%)',$4,$5,$6,'reputecast')`,
		ID(), fab.ID, tripped, camp, ses, resumed); err != nil {
		return err
	}
	if _, err := n.Tx.Exec(ctx, `INSERT INTO campaign_incidents (id, campaign_id, kind, scope, provider_id, mailbox_provider, metric, value, threshold, held, tripped_at,
		resolved_at, resolution, resolved_by, detail) VALUES ($1,$2,'threshold','mailbox_provider',$3,'yahoo','deferral',4.1,3,9120,$4,$5,'resumed_with_changes',$6,$7)`,
		ID(), camp, ses, tripped, resumed, mei.ID, json.RawMessage(`{"changes":["Yahoo at 20 messages a second"]}`)); err != nil {
		return err
	}
	down, up := n.at(-3, 14, 10), n.at(-3, 14, 52)
	if _, err := n.Tx.Exec(ctx, `INSERT INTO incidents (id, workspace_id, occurred_at, kind, severity, title, detail, provider_id, resolved_at, source)
		VALUES ($1,$2,$3,'provider_down','bad','Mailgun was down for 42 minutes','Mailgun''s API answered 503 from 14:10; it recovered at 14:52.',$4,$5,'reputecast')`,
		ID(), lit.ID, down, mg, up); err != nil {
		return err
	}
	_, err := n.Tx.Exec(ctx, `INSERT INTO stream_failovers (id, workspace_id, stream, provider_id, fallback_provider_id, mode, started_at, ended_at, end_reason, detail)
		VALUES ($1,$2,'automation',$3,$4,'automatic',$5,$6,'recovered','{}')`, ID(), lit.ID, mg, pm, down.Add(time.Minute), up)
	return err
}
