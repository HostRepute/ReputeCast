package demo

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/agency"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// The agency sample's content: its three workspace templates and the
// shared library (41 items), with their MJML.

//go:embed assets/agency/*.png
var agencyAssets embed.FS

func agencySeedTemplate(ctx context.Context, n *nwSeed, tpl, ws, by uuid.UUID, at time.Time) (agency.AgencyTemplatePlan, error) {
	return agency.SeedTemplate(ctx, n.Tx, tpl, ws, by, at)
}

func agencyEnableSignoff(raw json.RawMessage) (json.RawMessage, error) {
	return approvals.EnableClientSignoff(raw, true)
}

// nwImage uploads one of the sample's images to a workspace's library and
// returns its URL.
func (n *nwSeed) nwImage(ctx context.Context, wsKey, file string) (string, error) {
	data, err := agencyAssets.ReadFile("assets/agency/" + file)
	if err != nil {
		return "", err
	}
	ctype, w, h, alpha, err := media.Inspect(data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	var id uuid.UUID
	if err := n.Tx.QueryRow(ctx, `INSERT INTO media_assets (id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, sha256, data, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (workspace_id, sha256) DO UPDATE SET hidden_at = NULL RETURNING id`,
		ID(), n.ws[wsKey].ID, file, ctype, len(data), w, h, alpha, sum[:], data, n.people["ravi"].ID, n.at(-30, 11, 0)).Scan(&id); err != nil {
		return "", err
	}
	return media.URL(id, file), nil
}

// MJML building blocks of the sample's emails.

const nwFont = `Helvetica, Arial, sans-serif`

func nwDoc(title string, sections ...string) string {
	return `<mjml><mj-head><mj-title>` + composerXMLText(title) + `</mj-title><mj-attributes><mj-all font-family="` + nwFont + `"></mj-all>` +
		`<mj-text font-size="15px" line-height="1.5" color="#1F2937" padding="0px 0px 10px"></mj-text></mj-attributes></mj-head>` +
		`<mj-body width="600px" background-color="#F3F4F6">` + strings.Join(sections, "\n") + `</mj-body></mjml>`
}

// nwSectionDoc is a saved section as a document, for compiling and thumbs.
func nwSectionDoc(section string) string {
	return `<mjml><mj-body width="600px">` + section + `</mj-body></mjml>`
}

func nwHeaderSec(name, color string) string {
	return `<mj-section background-color="#FFFFFF" padding="16px 24px"><mj-column><mj-text font-size="17px" font-weight="700" color="` + color + `" padding="0px">` +
		composerXMLText(name) + `</mj-text></mj-column></mj-section>`
}

func nwButton(label, href, bg, color string) string {
	return `<mj-button href="` + href + `" background-color="` + bg + `" color="` + color + `" font-size="14px" font-weight="700" border-radius="6px" inner-padding="11px 20px" padding="6px 0px 0px">` +
		composerXMLText(label) + `</mj-button>`
}

// nwHeroSec is a coloured hero: a headline, a line and a button; bgURL
// lays a pattern over the colour.
func nwHeroSec(bg, bgURL, title, line, button, href, btnBg, btnColor string) string {
	pattern := ""
	if bgURL != "" {
		pattern = ` background-url="` + bgURL + `" background-repeat="repeat" background-size="96px 96px"`
	}
	return `<mj-section background-color="` + bg + `"` + pattern + ` padding="40px 32px"><mj-column>` +
		`<mj-text align="center" font-family="Georgia, 'Times New Roman', serif" font-size="30px" line-height="1.2" color="#FFFFFF">` + composerXMLText(title) + `</mj-text>` +
		`<mj-text align="center" font-size="15px" color="#E5E7EB" padding="0px 0px 14px">` + line + `</mj-text>` +
		nwButton(button, href, btnBg, btnColor) + `</mj-column></mj-section>`
}

func nwTextSec(title, body string) string {
	return `<mj-section background-color="#FFFFFF" padding="24px 32px"><mj-column>` +
		`<mj-text font-size="20px" font-weight="700" color="#0B1220">` + composerXMLText(title) + `</mj-text>` +
		`<mj-text>` + body + `</mj-text></mj-column></mj-section>`
}

func nwBannerSec(text, bg, color string) string {
	return `<mj-section background-color="` + bg + `" padding="12px 24px"><mj-column><mj-text align="center" font-size="14px" font-weight="700" color="` + color + `" padding="0px">` +
		composerXMLText(text) + `</mj-text></mj-column></mj-section>`
}

func nwGridSec(images []string, names []string, button, href string) string {
	var b strings.Builder
	b.WriteString(`<mj-section background-color="#FFFFFF" padding="16px 12px">`)
	for i, img := range images {
		b.WriteString(`<mj-column><mj-image src="` + img + `" alt="` + composerXMLText(names[i]) + `" width="168px" padding="8px"></mj-image>` +
			`<mj-text align="center" font-size="14px" font-weight="700" padding="0px 8px 6px">` + composerXMLText(names[i]) + `</mj-text>` +
			`<mj-button href="` + href + `" background-color="#0F766E" color="#FFFFFF" font-size="13px" border-radius="999px" inner-padding="7px 14px" padding="0px 8px 8px">` +
			composerXMLText(button) + `</mj-button></mj-column>`)
	}
	b.WriteString(`</mj-section>`)
	return b.String()
}

func nwTwoColSec(img, alt, title, body string) string {
	return `<mj-section background-color="#FFFFFF" padding="16px 24px"><mj-column width="45%"><mj-image src="` + img + `" alt="` + composerXMLText(alt) + `" padding="0px"></mj-image></mj-column>` +
		`<mj-column width="55%"><mj-text font-size="17px" font-weight="700" padding="0px 0px 6px 16px">` + composerXMLText(title) + `</mj-text>` +
		`<mj-text padding="0px 0px 0px 16px">` + body + `</mj-text></mj-column></mj-section>`
}

func nwFooterSec(address string) string {
	return `<mj-section padding="16px 24px 24px"><mj-column><mj-text align="center" font-size="12px" color="#4B5563" padding="0px">` + composerXMLText(address) +
		`<br>You get this because you subscribed. <a href="{{ unsubscribe_url }}" style="color:#4B5563">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#4B5563">Preferences</a></mj-text></mj-column></mj-section>`
}

func nwCompile(src string) (string, error) {
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		return "", fmt.Errorf("compile sample MJML: %s", res.Errors[0].Message)
	}
	return res.HTML, nil
}

// nwItem is a saved section or campaign template of the sample: its MJML
// (a section, or a whole document), compiled, with its thumbnail.
func nwItem(name, src string, section bool) (agency.TplItem, error) {
	doc := src
	if section {
		doc = nwSectionDoc(src)
	}
	html, err := nwCompile(doc)
	if err != nil {
		return agency.TplItem{}, fmt.Errorf("%s: %w", name, err)
	}
	thumb, _ := json.Marshal(mjml.Thumb(doc))
	return agency.TplItem{Name: name, MJML: src, HTML: html, Thumb: thumb}, nil
}

// nwSectionMJML is a generic saved section of the sample, by its name.
func nwSectionMJML(name string, v int, img string) string {
	switch {
	case strings.HasPrefix(name, "Header"):
		return nwHeaderSec("Your store", "#0B1220")
	case strings.HasPrefix(name, "Footer"):
		return nwFooterSec("Your company · street, city")
	case strings.Contains(name, "Two columns"), strings.Contains(name, "spotlight"), strings.Contains(name, "of the month"):
		return nwTwoColSec(img, name, name, "A picture on one side and a few lines about it on the other.")
	case strings.Contains(name, "banner"), strings.Contains(name, "Banner"), strings.Contains(name, "code"), strings.Contains(name, "Countdown"):
		return nwBannerSec(name+fmt.Sprintf(" · v%d", v), "#FEF3C7", "#92400E")
	case strings.Contains(name, "grid"), strings.Contains(name, "links"):
		return nwGridSec([]string{img, img, img}, []string{"One", "Two", "Three"}, "Shop", "{{ shop_url | default: '#' }}")
	}
	return nwTextSec(name, fmt.Sprintf("Copy for the %s block (version %d). Hi {{ first_name | default: \"there\" }}, here's what's new.", strings.ToLower(name), v))
}

// nwTemplateMJML is a generic campaign template of the sample.
func nwTemplateMJML(name string, v int, img string) string {
	return nwDoc(name,
		nwHeaderSec("Your store", "#0B1220"),
		nwHeroSec("#0F766E", "", name, "Hi {{ first_name | default: \"there\" }}, a short line about this email.", "Shop now", "{{ shop_url | default: '#' }}", "#FFFFFF", "#0F766E"),
		nwTwoColSec(img, name, fmt.Sprintf("%s · section %d", name, v), "Two or three sentences that tell the story of this send."),
		nwFooterSec("Your company · street, city"))
}

// Flows of the sample's automations, by step.
type nwStep struct {
	kind, label string
	delay       int
	unit        string
	email       string // ref
}

func nwFlow(trigger automations.AutomationTrigger, steps ...nwStep) json.RawMessage {
	g := automations.AutomationGraph{Nodes: []automations.AutomationNode{{ID: "start", Type: "trigger", Position: automations.AutomationNodePosition{X: 0, Y: 0},
		Data: automations.AutomationNodeData{Label: "Trigger", Trigger: &trigger}}}, Edges: []automations.AutomationEdge{}}
	prev := "start"
	for i, st := range steps {
		id := fmt.Sprintf("s%d", i+1)
		node := automations.AutomationNode{ID: id, Type: st.kind, Position: automations.AutomationNodePosition{X: 0, Y: float64(140 * (i + 1))},
			Data: automations.AutomationNodeData{Label: st.label}}
		switch st.kind {
		case "delay":
			node.Data.Delay = &automations.AutomationDuration{Amount: st.delay, Unit: st.unit}
		case "email":
			node.Data.Email = &automations.AutomationEmailStep{EmailID: "email:" + st.email}
		}
		g.Nodes = append(g.Nodes, node)
		g.Edges = append(g.Edges, automations.AutomationEdge{ID: prev + "-" + id, Source: prev, Target: id})
		prev = id
	}
	return g.MustJSON()
}

func nwEmail(ref, name, subject, headline string) (agency.TplEmail, error) {
	src := nwDoc(subject, nwHeaderSec("Your store", "#0B1220"),
		nwTextSec(headline, "Hi {{ first_name | default: \"there\" }}, "+composerXMLText(strings.ToLower(subject[:1])+subject[1:])+"."),
		nwFooterSec("Your company · street, city"))
	html, err := nwCompile(src)
	if err != nil {
		return agency.TplEmail{}, err
	}
	return agency.TplEmail{Ref: ref, Name: name, Subject: subject, MJML: src, HTML: html}, nil
}

// nwAutomation is a flow of trigger, emails and waits.
func nwAutomation(name, desc string, trigger automations.AutomationTrigger, steps []nwStep, emails [][3]string) (agency.TplAutomation, error) {
	out := agency.TplAutomation{Name: name, Description: desc, Graph: nwFlow(trigger, steps...), Emails: []agency.TplEmail{}}
	for _, e := range emails {
		em, err := nwEmail(e[0], e[1], e[2], e[1])
		if err != nil {
			return out, err
		}
		out.Emails = append(out.Emails, em)
	}
	return out, nil
}

func joinedList(name string) automations.AutomationTrigger {
	return automations.AutomationTrigger{Kind: "joined_list", ListID: "list:" + name}
}

func apiEvent(event string) automations.AutomationTrigger {
	return automations.AutomationTrigger{Kind: "api_event", Event: event, Reentry: "always"}
}

func nwWait(days int) nwStep {
	return nwStep{kind: "delay", label: fmt.Sprintf("Wait %d days", days), delay: days, unit: "days"}
}

func nwSend(ref, label string) nwStep { return nwStep{kind: "email", label: label, email: ref} }

// The flows the templates and the library share.
func nwWelcome(list string) (agency.TplAutomation, error) {
	return nwAutomation("Welcome series", "Three emails over a week for new subscribers", joinedList(list),
		[]nwStep{nwSend("w1", "Welcome"), nwWait(2), nwSend("w2", "Our story"), nwWait(5), nwSend("w3", "First order offer")},
		[][3]string{{"w1", "Welcome", "Welcome aboard"}, {"w2", "Our story", "How we started"}, {"w3", "First order offer", "Here's 10% off your first order"}})
}

func nwAbandonedCart(name string) (agency.TplAutomation, error) {
	return nwAutomation(name, "Reminds shoppers of what they left in the cart (Shopify's checkout event)", apiEvent("checkout_abandoned"),
		[]nwStep{{kind: "delay", label: "Wait 1 hour", delay: 1, unit: "hours"}, nwSend("c1", "Reminder"), nwWait(1), nwSend("c2", "Last call")},
		[][3]string{{"c1", "Reminder", "You left something in your cart"}, {"c2", "Last call", "Your cart is about to expire"}})
}

func nwPostPurchase() (agency.TplAutomation, error) {
	return nwAutomation("Post-purchase thank-you", "Thanks the customer and asks for a review", apiEvent("order_placed"),
		[]nwStep{nwWait(3), nwSend("p1", "Thank you"), nwWait(10), nwSend("p2", "Review request")},
		[][3]string{{"p1", "Thank you", "Thanks for your order"}, {"p2", "Review request", "How is it going?"}})
}

func nwWinBack() (agency.TplAutomation, error) {
	return nwAutomation("Win-back", "Brings back customers who haven't opened in 90 days", automations.AutomationTrigger{Kind: "tag_added", Tag: "lapsed-90d"},
		[]nwStep{nwSend("b1", "We miss you"), nwWait(7), nwSend("b2", "A gift")},
		[][3]string{{"b1", "We miss you", "It's been a while"}, {"b2", "A gift", "A little something to welcome you back"}})
}

// workspaceTemplates creates the three workspace templates of the Library
// board: Retail starter (Shopify), Hospitality starter and Publisher starter.
func (n *nwSeed) workspaceTemplates(ctx context.Context) error {
	retailSections := []string{"Header · logo", "Hero · product", "Product grid · 3 up", "Product spotlight", "Free shipping banner", "Discount code",
		"Customer reviews", "Two columns · image left", "Category links", "Countdown banner", "Brand story", "Social links", "App download", "Footer · retail"}
	retailTemplates := []string{"Weekly offers", "New arrivals", "Flash sale", "Back in stock", "Gift guide", "Thank-you note"}
	hospitalitySections := []string{"Header · logo", "Hero · seasonal", "Booking button", "Two columns · image left", "Event listing", "Opening hours",
		"Map and directions", "Social links", "Footer · hospitality"}
	publisherSections := []string{"Masthead", "Lead story", "Story list", "Book of the month", "Footer · publisher"}
	items := func(names []string, section bool) ([]agency.TplItem, error) {
		out := []agency.TplItem{}
		for _, name := range names {
			src := nwSectionMJML(name, 1, "/media/starters/hero.png")
			if !section {
				src = nwTemplateMJML(name, 1, "/media/starters/hero.png")
			}
			it, err := nwItem(name, src, section)
			if err != nil {
				return nil, err
			}
			out = append(out, it)
		}
		return out, nil
	}
	flows := func(fns ...func() (agency.TplAutomation, error)) ([]agency.TplAutomation, error) {
		out := []agency.TplAutomation{}
		for _, fn := range fns {
			a, err := fn()
			if err != nil {
				return nil, err
			}
			out = append(out, a)
		}
		return out, nil
	}
	retail := agency.TemplateContent{Lists: []agency.TplList{{Name: "Newsletter", Description: "News and offers", DoubleOptIn: true},
		{Name: "Customers", Description: "People who bought from the store, from Shopify"}, {Name: "VIP", Description: "The best customers"}},
		ClientSignoff: true, NextSteps: []string{"provider", "domain", "invite", "shopify"}, SMSTexts: []agency.TplSMS{}, Forms: []agency.TplForm{}, Roles: []agency.TplRole{}}
	var err error
	if retail.Automations, err = flows(func() (agency.TplAutomation, error) { return nwWelcome("Newsletter") },
		func() (agency.TplAutomation, error) { return nwAbandonedCart("Abandoned cart") }, nwPostPurchase, nwWinBack); err != nil {
		return err
	}
	if retail.Sections, err = items(retailSections, true); err != nil {
		return err
	}
	if retail.Templates, err = items(retailTemplates, false); err != nil {
		return err
	}
	hospitality := agency.TemplateContent{Lists: []agency.TplList{{Name: "Newsletter", Description: "News and offers", DoubleOptIn: true},
		{Name: "Guests", Description: "Everyone who booked a stay or a table"}}, AutomationsNote: "welcome, booking reminder by SMS, win-back",
		NextSteps: []string{"provider", "domain", "invite"}, Templates: []agency.TplItem{}, SMSTexts: []agency.TplSMS{}, Forms: []agency.TplForm{}, Roles: []agency.TplRole{}}
	if hospitality.Automations, err = flows(func() (agency.TplAutomation, error) { return nwWelcome("Newsletter") },
		func() (agency.TplAutomation, error) {
			return nwAutomation("Booking reminder", "A reminder the day before a booking (by text once SMS is set up)", apiEvent("booking_confirmed"),
				[]nwStep{{kind: "delay", label: "Until the day before", delay: 1, unit: "days"}, nwSend("r1", "Reminder")},
				[][3]string{{"r1", "Reminder", "See you tomorrow"}})
		}, nwWinBack); err != nil {
		return err
	}
	if hospitality.Sections, err = items(hospitalitySections, true); err != nil {
		return err
	}
	publisher := agency.TemplateContent{Lists: []agency.TplList{{Name: "Newsletter", Description: "The weekly letter", DoubleOptIn: true},
		{Name: "Book club", Description: "Monthly picks and events"}}, ClientSignoff: true, NextSteps: []string{"provider", "domain", "invite"},
		Templates: []agency.TplItem{}, SMSTexts: []agency.TplSMS{}, Forms: []agency.TplForm{}, Roles: []agency.TplRole{}}
	if publisher.Automations, err = flows(func() (agency.TplAutomation, error) { return nwWelcome("Newsletter") },
		func() (agency.TplAutomation, error) {
			return nwAutomation("New release alert", "Tells the book club about a new release", apiEvent("release_published"),
				[]nwStep{nwSend("n1", "New release")}, [][3]string{{"n1", "New release", "Out today"}})
		}); err != nil {
		return err
	}
	if publisher.Sections, err = items(publisherSections, true); err != nil {
		return err
	}
	for _, t := range []struct {
		name, desc string
		content    agency.TemplateContent
		by         string
		updated    time.Time
	}{
		{nwRetail, "Shops on Shopify: lists, the welcome and cart flows, product sections and client sign-off", retail, "tomas", n.onBoards(date(2026, 10, 2)).Add(4 * time.Hour)},
		{nwHospitality, "Hotels, restaurants and venues", hospitality, "mei", n.onBoards(date(2026, 9, 18)).Add(5 * time.Hour)},
		{nwPublisher, "Publishers and bookshops", publisher, "ella", n.onBoards(date(2026, 9, 4)).Add(6 * time.Hour)},
	} {
		raw, err := json.Marshal(t.content)
		if err != nil {
			return err
		}
		id := ID()
		by := n.people[t.by].ID
		if err := n.Q.AgencyTemplateInsert(ctx, store.AgencyTemplateInsertParams{ID: id, Name: t.name, Description: t.desc, Content: raw, CreatedBy: &by}); err != nil {
			return err
		}
		if _, err := n.Tx.Exec(ctx, `UPDATE agency_templates SET created_at = $2, updated_at = $3 WHERE id = $1`, id, n.onBoards(date(2026, 8, 1)), t.updated); err != nil {
			return err
		}
		n.templates[t.name] = id
	}
	return nil
}

// nwLibVersion is one version of a library item.
type nwLibVersion struct {
	at      time.Time
	by      string
	note    string
	content agency.LibraryContent
	extra   []string // summary lines the content doesn't show ("Snow overlay: 40%")
}

type nwLibCopy struct {
	ws      string
	version int
	at      time.Time
	by      string
}

type nwLibItem struct {
	kind, name string
	versions   []nwLibVersion
	copies     []nwLibCopy
}

// library fills the shared library: 22 saved sections, 9 campaign
// templates, 5 automation templates, 3 SMS texts and 2 transactional
// templates, with the versions and copies of the Library board.
func (n *nwSeed) library(ctx context.Context) error {
	snow, err := n.nwImage(ctx, "northwind", "snow.png")
	if err != nil {
		return err
	}
	products := []string{}
	for _, f := range []string{"product-1.png", "product-2.png", "product-3.png"} {
		u, err := n.nwImage(ctx, "northwind", f)
		if err != nil {
			return err
		}
		products = append(products, u)
	}
	day := func(m time.Month, d, h int) time.Time {
		return n.onBoards(date(2026, m, d)).Add(time.Duration(h-10) * time.Hour)
	}
	// Content that doesn't compile stops the seed (cerr, checked below).
	var cerr error
	sec := func(src string) agency.LibraryContent {
		it, err := nwItem("", src, true)
		if err != nil && cerr == nil {
			cerr = err
		}
		return agency.LibraryContent{MJML: src, HTML: it.HTML, Thumb: it.Thumb}
	}
	tpl := func(src string) agency.LibraryContent {
		it, err := nwItem("", src, false)
		if err != nil && cerr == nil {
			cerr = err
		}
		return agency.LibraryContent{MJML: src, HTML: it.HTML, Thumb: it.Thumb}
	}
	hero := func(title, button, pattern string) string {
		return nwHeroSec("#1E3A8A", pattern, title, `Hi {{ first_name | default: "there" }}, gifts ship free until 15 Dec.`, button, "{{ shop_url }}", "#FDE68A", "#1E3A8A")
	}
	grid := func(button string) string {
		return nwGridSec(products, []string{"Day pack", "Trail tent", "Hiking boots"}, button, "{{ product_feed }}")
	}
	items := []nwLibItem{
		{kind: agency.KindSection, name: "Holiday hero (snow)", versions: []nwLibVersion{
			{at: day(9, 18, 15), by: "ravi", note: "First version", content: sec(hero("Happy holidays", "Shop now", ""))},
			{at: day(10, 2, 11), by: "ravi", note: "Snowfall overlay and a new headline", content: sec(hero("The holidays, wrapped", "Shop now", snow)), extra: []string{"Snow overlay: 40%"}},
			{at: day(10, 12, 16), by: "ravi", note: "Lighter snowfall; the button reads “Shop gifts”", content: sec(hero("The holidays, wrapped", "Shop gifts", snow)), extra: []string{"Snow overlay: 25%"}},
		}, copies: []nwLibCopy{{"contoso", 2, day(10, 3, 9), "mei"}, {"fabrikam", 3, day(10, 13, 9), "mei"}, {"tailspin", 3, day(10, 13, 14), "ella"}, {"alpine", 3, day(10, 14, 10), "ella"}}},
		{kind: agency.KindSection, name: "Product grid · 3 up", versions: n.versions(5, day(9, 30, 14), "ravi", func(v int) agency.LibraryContent {
			return sec(grid([]string{"Shop", "Shop", "View", "View", "Shop now"}[v-1]))
		}), copies: []nwLibCopy{{"contoso", 5, day(10, 1, 9), "mei"}, {"fabrikam", 5, day(10, 1, 10), "mei"}, {"tailspin", 5, day(10, 2, 9), "ella"},
			{"alpine", 5, day(10, 2, 11), "ella"}, {"litware", 5, day(10, 6, 9), "ella"}}},
		{kind: agency.KindAutomation, name: "Abandoned cart (Shopify)", versions: n.versions(2, day(9, 22, 11), "mei", func(v int) agency.LibraryContent {
			a, _ := nwAbandonedCart("Abandoned cart (Shopify)")
			if v == 1 {
				a.Graph = nwFlow(apiEvent("checkout_abandoned"), nwStep{kind: "delay", label: "Wait 4 hours", delay: 4, unit: "hours"}, nwSend("c1", "Reminder"))
				a.Emails = a.Emails[:1]
			}
			return agency.LibraryContent{Automation: &a}
		}), copies: []nwLibCopy{{"contoso", 2, day(9, 23, 9), "mei"}, {"fabrikam", 2, day(9, 23, 10), "mei"}, {"tailspin", 2, day(9, 24, 9), "ella"}}},
		{kind: agency.KindTemplate, name: "Seasonal sale", versions: n.versions(4, day(10, 8, 15), "ravi", func(v int) agency.LibraryContent {
			return tpl(nwDoc("Seasonal sale", nwHeaderSec("Your store", "#0B1220"),
				nwBannerSec([]string{"Up to 30% off", "Up to 40% off", "The seasonal sale is on", "The seasonal sale: up to 40% off"}[v-1], "#FDE68A", "#7C2D12"),
				grid("Shop the sale"), nwFooterSec("Your company · street, city")))
		}), copies: []nwLibCopy{{"contoso", 4, day(10, 9, 9), "mei"}, {"fabrikam", 4, day(10, 9, 10), "mei"}, {"tailspin", 4, day(10, 9, 11), "ella"}, {"alpine", 4, day(10, 10, 9), "ella"}}},
		{kind: agency.KindSMS, name: "Flash sale text", versions: []nwLibVersion{{at: day(10, 1, 12), by: "ella", note: "First version",
			content: agency.LibraryContent{Body: "{{ workspace_name }}: 24-hour flash sale, 25% off everything with FLASH25. {{ shop_url }} Reply STOP to opt out"}}},
			copies: []nwLibCopy{{"fabrikam", 1, day(10, 2, 9), "mei"}, {"alpine", 1, day(10, 3, 9), "ella"}}},
		{kind: agency.KindTransactional, name: "Order confirmation", versions: n.versions(3, day(9, 15, 11), "tomas", func(v int) agency.LibraryContent {
			src := nwDoc("Order confirmation", nwHeaderSec("Your store", "#0B1220"),
				nwTextSec([]string{"Thanks for your order", "Your order is confirmed", "Order {{ order_number }} is confirmed"}[v-1],
					"Hi {{ first_name | default: \"there\" }}, we're getting order {{ order_number }} ready. You'll get another email when it ships."),
				nwFooterSec("Your company · street, city"))
			html, _ := nwCompile(src)
			return agency.LibraryContent{Subject: []string{"Thanks for your order", "Your order is confirmed", "Order {{ order_number }} is confirmed"}[v-1], MJML: src, HTML: html}
		}), copies: []nwLibCopy{{"contoso", 3, day(9, 16, 9), "tomas"}, {"fabrikam", 3, day(9, 16, 10), "tomas"}, {"tailspin", 3, day(9, 17, 9), "tomas"}}},
		{kind: agency.KindTemplate, name: "Monthly roundup", versions: n.versions(2, day(8, 28, 10), "ella", func(v int) agency.LibraryContent {
			return tpl(nwDoc("Monthly roundup", nwHeaderSec("Your store", "#0B1220"),
				nwBannerSec([]string{"This month", "This month at a glance"}[v-1], "#FEF3C7", "#92400E"),
				nwTwoColSec(products[0], "Story", "The month in three stories", "A few lines about each, with a link to read on."), nwFooterSec("Your company · street, city")))
		})},
	}
	// The rest of the library: one version each or a few, used less.
	fillSections := []string{"Free shipping banner", "Customer reviews", "Countdown banner", "Discount code", "Gift card promo", "Store locator", "Social links",
		"Footer · retail", "Header · logo left", "Two columns · image left", "Testimonial quote", "Event listing", "Booking button", "Menu highlights",
		"Author spotlight", "Book of the month", "Reading list", "Shipping deadlines", "Loyalty points", "App download"}
	for i, name := range fillSections {
		vs := 1 + i%3
		at := day(8, 10+i, 11)
		img := products[i%3]
		items = append(items, nwLibItem{kind: agency.KindSection, name: name, versions: n.versions(vs, at, []string{"ravi", "mei", "ella"}[i%3], func(v int) agency.LibraryContent {
			return sec(nwSectionMJML(name, v, img))
		})})
	}
	for i, name := range []string{"New arrivals", "Flash sale", "Back in stock", "Gift guide", "Event invitation", "Newsletter · classic", "Product launch"} {
		img := products[i%3]
		items = append(items, nwLibItem{kind: agency.KindTemplate, name: name, versions: n.versions(1+i%3, day(8, 12+i*2, 10), []string{"ella", "ravi", "mei"}[i%3],
			func(v int) agency.LibraryContent { return tpl(nwTemplateMJML(name, v, img)) })})
	}
	for i, fn := range []func() (agency.TplAutomation, error){func() (agency.TplAutomation, error) { return nwWelcome("Newsletter") }, nwWinBack, nwPostPurchase,
		func() (agency.TplAutomation, error) {
			return nwAutomation("Birthday reward", "A gift on the contact's birthday", automations.AutomationTrigger{Kind: "date_field", Field: "birthday", Yearly: true, At: "09:00"},
				[]nwStep{nwSend("d1", "Birthday gift")}, [][3]string{{"d1", "Birthday gift", "Happy birthday from us"}})
		}} {
		a, err := fn()
		if err != nil {
			return err
		}
		items = append(items, nwLibItem{kind: agency.KindAutomation, name: a.Name, versions: n.versions(1+i%3, day(8, 20+i, 10), "mei",
			func(int) agency.LibraryContent { return agency.LibraryContent{Automation: &a} })})
	}
	items = append(items,
		nwLibItem{kind: agency.KindSMS, name: "Booking reminder text", versions: n.versions(2, day(9, 5, 10), "mei", func(v int) agency.LibraryContent {
			return agency.LibraryContent{Body: []string{"See you tomorrow at {{ booking_time }}. Reply C to cancel.",
				"{{ workspace_name }}: see you tomorrow at {{ booking_time }}. Reply C to cancel, STOP to opt out"}[v-1]}
		})},
		nwLibItem{kind: agency.KindSMS, name: "Order shipped text", versions: n.versions(1, day(9, 8, 10), "tomas", func(int) agency.LibraryContent {
			return agency.LibraryContent{Body: "{{ workspace_name }}: order {{ order_number }} is on its way. Track it: {{ tracking_url }}"}
		})},
		nwLibItem{kind: agency.KindTransactional, name: "Password reset", versions: n.versions(1, day(8, 18, 10), "tomas", func(int) agency.LibraryContent {
			src := nwDoc("Password reset", nwHeaderSec("Your store", "#0B1220"),
				nwTextSec("Reset your password", "Hi {{ first_name | default: \"there\" }}, use the link below within an hour. <a href=\"{{ reset_url }}\">Reset your password</a>"),
				nwFooterSec("Your company · street, city"))
			html, _ := nwCompile(src)
			return agency.LibraryContent{Subject: "Reset your password", MJML: src, HTML: html}
		})},
	)
	if cerr != nil {
		return cerr
	}
	for _, it := range items {
		if err := n.libraryItem(ctx, it); err != nil {
			return fmt.Errorf("%s: %w", it.name, err)
		}
	}
	return nil
}

// versions makes count versions ending at last, a week apart.
func (n *nwSeed) versions(count int, last time.Time, by string, content func(v int) agency.LibraryContent) []nwLibVersion {
	out := []nwLibVersion{}
	notes := []string{"First version", "Tighter spacing", "New button colour", "Copy edits", "Mobile fixes"}
	for v := 1; v <= count; v++ {
		out = append(out, nwLibVersion{at: last.AddDate(0, 0, -7*(count-v)), by: by, note: notes[(v-1)%len(notes)], content: content(v)})
	}
	return out
}

func (n *nwSeed) libraryItem(ctx context.Context, it nwLibItem) error {
	id := ID()
	first, last := it.versions[0], it.versions[len(it.versions)-1]
	_, fields := agency.SeedLibraryVersion(it.kind, last.content)
	by := n.people[first.by].ID
	if err := n.Q.AgencyLibraryInsertItem(ctx, store.AgencyLibraryInsertItemParams{ID: id, Kind: it.kind, Name: it.name, Fields: fields, CreatedBy: &by, At: first.at}); err != nil {
		return err
	}
	src := n.ws["northwind"].ID
	for i, v := range it.versions {
		summary, _ := agency.SeedLibraryVersion(it.kind, v.content)
		summary = append(summary, v.extra...)
		raw, err := json.Marshal(v.content)
		if err != nil {
			return err
		}
		vby := n.people[v.by].ID
		if err := n.Q.AgencyLibraryInsertVersion(ctx, store.AgencyLibraryInsertVersionParams{ItemID: id, Version: int32(i + 1), Content: raw, Summary: summary, Note: v.note,
			SourceWorkspaceID: &src, CreatedBy: &vby, At: v.at}); err != nil {
			return err
		}
	}
	lby := n.people[last.by].ID
	if _, err := n.Tx.Exec(ctx, `UPDATE agency_library_items SET version = $2, updated_by = $3, updated_at = $4 WHERE id = $1`, id, len(it.versions), lby, last.at); err != nil {
		return err
	}
	for _, c := range it.copies {
		cby := n.people[c.by].ID
		switch it.kind {
		case agency.KindSection, agency.KindTemplate, agency.KindAutomation:
			if _, err := agency.SeedLibraryCopy(ctx, n.Tx, id, int32(c.version), n.ws[c.ws].ID, cby, c.at); err != nil {
				return err
			}
		case agency.KindTransactional:
			if err := n.transactionalCopy(ctx, n.Tx, id, it, c); err != nil {
				return err
			}
		default:
			// SMS texts reach a client with its SMS channel (X21); the
			// sample records where they went.
			if err := n.Q.AgencyLibraryInsertCopy(ctx, store.AgencyLibraryInsertCopyParams{ID: ID(), ItemID: id, Version: int32(c.version), WorkspaceID: n.ws[c.ws].ID,
				TargetKind: "sms", TargetID: ID(), CopiedBy: &cby, At: c.at}); err != nil {
				return err
			}
		}
	}
	return nil
}

// transactionalCopy saves a transactional template copied from the
// library in a client, as a draft, and records the copy.
func (n *nwSeed) transactionalCopy(ctx context.Context, tx pgx.Tx, item uuid.UUID, it nwLibItem, c nwLibCopy) error {
	v := it.versions[c.version-1].content
	id := ID()
	by := n.people[c.by].ID
	origin, _ := json.Marshal(map[string]any{"source": "agency_library", "item": item.String(), "version": c.version})
	key := strings.ReplaceAll(strings.ToLower(it.name), " ", "-")
	if _, err := tx.Exec(ctx, `INSERT INTO transactional_templates (id, workspace_id, key, name, description, subject, preheader, editor_mode, mjml, html, plain_text,
		variables, test_data, origin, created_by, created_at, updated_at, draft_saved_at, draft_saved_by)
		VALUES ($1,$2,$3,$4,'',$5,'','visual',$6,$7,'','[]','{}',$8,$9,$10,$10,$10,$9)`,
		id, n.ws[c.ws].ID, key, it.name, v.Subject, v.MJML, v.HTML, origin, by, c.at); err != nil {
		return err
	}
	return n.Q.AgencyLibraryInsertCopy(ctx, store.AgencyLibraryInsertCopyParams{ID: ID(), ItemID: item, Version: int32(c.version), WorkspaceID: n.ws[c.ws].ID,
		TargetKind: "transactional", TargetID: id, CopiedBy: &by, At: c.at})
}
