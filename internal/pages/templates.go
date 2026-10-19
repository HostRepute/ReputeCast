package pages

import (
	"fmt"
	"html"
	"strings"
)

// The six templates of "New page", as the HTML of their blocks: every
// block is a top-level <section data-pb="…"> with an id and a layer name,
// its markup the public pages' own (static/pages.css), so the page builder
// edits exactly what readers get. Blocks that show live data are custom
// elements (blocks.go) the site draws.

// TemplateOptions fill a template in: the page's name, the publisher, the
// workspace's signup form, product and timezone when it has them.
type TemplateOptions struct {
	Name      string
	Publisher string
	// FormID is the form signup blocks start with ("" for none yet).
	FormID string
	// ProductID is the product a product block starts with.
	ProductID string
	// TZ is the workspace's time zone (countdowns and events).
	TZ string
	// Today is the date a countdown or an event starts from ("2026-10-15").
	Date string
}

// TemplateHTML is a template's blocks.
func TemplateHTML(template string, o TemplateOptions) string {
	if o.TZ == "" {
		o.TZ = "UTC"
	}
	if o.Date == "" {
		o.Date = "2026-11-02"
	}
	e := html.EscapeString
	name, pub := e(o.Name), e(o.Publisher)
	switch template {
	case TemplateAbout:
		return join(
			section("hero", "Hero", "sp-hero", "lavender", "",
				`<div class="sp-in nar"><div class="hero-copy" data-pb-name="Heading and text"><span class="eyebrow">About</span><h1>`+name+`</h1>`+
					`<p class="sp-lead">Say who writes `+pub+`, who it's for and what readers get from it, in two sentences.</p></div></div>`),
			section("text", "Text · our story", "sp-sec", "white", "",
				`<div class="sp-in nar"><div class="prose"><h2>Why we write it</h2><p>Tell the story behind the newsletter: when it started, what it covers and what a reader can expect each week.</p>`+
					`<p>Keep it short and specific. Readers decide in a few lines whether it's for them.</p></div></div>`),
			section("bio", "Author bio", "sp-sec lav", "lavender", "",
				`<div class="sp-in nar"><div class="bio"><span class="av7" aria-hidden="true">`+e(initials(o.Publisher))+`</span><div class="tx">`+
					`<span class="eyebrow">Who writes it</span><h2 class="sp-h2">Your name</h2><span class="role">Editor of `+pub+`</span>`+
					`<p>A few lines about the person behind the newsletter, and what they work on.</p></div></div></div>`),
			formSection("signup", "Signup form · subscribe band", o.FormID, "Get "+o.Publisher+" by email", "One email a week. Unsubscribe any time."),
		)
	case TemplateLeadMagnet:
		return join(
			section("hero", "Hero · the free file", "lp-band", "lavender", "",
				`<div class="sp-in"><div class="lp-hero"><div class="lp-copy" data-pb-name="Heading and text"><span class="eyebrow">Free checklist · PDF</span><h1>`+name+`</h1>`+
					`<p class="sp-lead sm">Say what the file helps with and how long it takes, in one or two sentences.</p></div>`+
					`<span class="cov lg blank" role="img" aria-label="Cover of `+name+`"></span></div></div>`),
			section("section", "The form and the reasons", "sp-sec cmp", "white", "",
				`<div class="sp-in"><div class="lp-body"><rc-form data-pb-id="form" data-pb-name="Signup form" form="`+e(o.FormID)+`" heading="Get the checklist" hint="We'll email a confirmation link. The file is on the page that opens."></rc-form>`+
					`<div class="bens" data-pb-name="Features list · benefits"><div class="ben"><b>The first benefit</b>What the reader gets from it.</div><div class="ben"><b>The second benefit</b>Why it saves them time.</div><div class="ben"><b>The third benefit</b>What they would otherwise miss.</div></div></div></div>`),
			quotesSection("Testimonials", 1),
			faqSection("FAQ", "Questions", "", [][2]string{{"Is it really free?", "Yes. You confirm your email and get the file and the newsletter. Unsubscribe any time."}, {"What's in it?", "Say what the file covers, page by page."}}),
		)
	case TemplateLaunch:
		return join(
			section("hero", "Hero · the launch", "lp-band", "lavender", "",
				`<div class="sp-in"><div class="lp-hero"><div class="lp-copy" data-pb-name="Heading, text and countdown"><span class="eyebrow">Opens soon</span><h1>`+name+`</h1>`+
					`<p class="sp-lead sm">Say what launches, when, and why it's worth waiting for.</p>`+
					`<rc-countdown data-pb-id="countdown" data-pb-name="Countdown" at="`+o.Date+`T09:00" tz="`+e(o.TZ)+`" units="days,hours,minutes" end="reveal" reveal="buy" hide="waitlist" label="Opens"></rc-countdown></div>`+
					`<span class="cov lg blank" role="img" aria-label="Cover of `+name+`"></span></div></div>`),
			section("section", "The waitlist, then the Buy button", "sp-sec cmp", "white", "",
				`<div class="sp-in"><div class="lp-body"><div class="lp-cols">`+
					`<rc-form data-pb-id="waitlist" data-pb-name="Signup form · the waitlist" form="`+e(o.FormID)+`" heading="Join the waitlist" hint="We'll email you the moment it opens."></rc-form>`+
					`<rc-product data-pb-id="buy" data-pb-name="Product · revealed when the countdown ends" id="buy" product="`+e(o.ProductID)+`" button="Buy {product}" price="1" offer="1" layout="box"></rc-product>`+
					`</div><div class="bens" data-pb-name="Features list"><div class="ben"><b>What's in it</b>The main thing the buyer gets.</div><div class="ben"><b>Who it's for</b>The reader it helps most.</div><div class="ben"><b>How it works</b>What happens after they buy.</div></div></div></div>`),
		)
	case TemplateWebinar:
		return join(
			section("hero", "Hero · the event", "lp-band", "lavender", "",
				`<div class="sp-in"><div class="lp-copy solo" data-pb-name="Heading and text"><span class="eyebrow">Live session</span><h1>`+name+`</h1>`+
					`<p class="sp-lead sm">Say who runs it, what they'll show and what attendees take away.</p></div></div>`),
			section("section", "Details, agenda and registration", "sp-sec cmp", "white", "",
				`<div class="sp-in nar"><div class="lp-body"><rc-event data-pb-id="event" data-pb-name="Event details" date="`+o.Date+`" start="17:00" end="17:30" tz="`+e(o.TZ)+`" local="1" where="online" platform="Zoom" calendar="1" title="`+name+`" reminders="1 day and 1 hour before"></rc-event>`+
					`<div class="host" data-pb-name="Speaker"><span class="hav" aria-hidden="true">`+e(initials(o.Publisher))+`</span><span class="hn"><b>Your name</b><span class="hr">What you do, and why you're running this session</span></span></div>`+
					`<div data-pb-name="Agenda"><h2 class="agenda-h">Agenda</h2><ol class="agenda"><li><span class="m">5 min</span>The first part</li><li><span class="m">15 min</span>The main part</li><li><span class="m">10 min</span>Your questions</li></ol></div>`+
					`<rc-form data-pb-id="register" data-pb-name="Signup form · registration" form="`+e(o.FormID)+`" heading="Save your seat" hint="We email the joining link right after you register."></rc-form></div></div>`),
		)
	case TemplateSales:
		return join(
			section("hero", "Hero", "sp-hero", "lavender", "",
				`<div class="sp-in"><div class="hero-grid"><div class="hero-copy" data-pb-name="Heading, text and checklist"><span class="eyebrow">Guide · PDF</span><h1>`+name+`</h1>`+
					`<p class="sp-lead">Say in two sentences what the reader gets, and the problem it solves.</p>`+
					`<ul class="ticks"><li>The first thing inside</li><li>The second thing inside</li><li>Who wrote it, and why</li></ul>`+
					`<rc-product data-pb-id="buy" data-pb-name="Product" id="buy" product="`+e(o.ProductID)+`" button="Buy for {price}" price="1" offer="1" layout="box"></rc-product></div>`+
					`<div class="hero-art"><span class="cov xl blank" role="img" aria-label="Cover of `+name+`"></span></div></div></div>`),
			featuresSection("features", "Features list · What's inside", "What's inside", "Say what each part covers", []string{"The first chapter", "The second chapter", "The third chapter", "The fourth chapter"}),
			section("bio", "Author bio", "sp-sec lav", "lavender", "",
				`<div class="sp-in nar"><div class="bio"><span class="av7" aria-hidden="true">`+e(initials(o.Publisher))+`</span><div class="tx"><span class="eyebrow">About the author</span>`+
					`<h2 class="sp-h2">Your name</h2><span class="role">What you do</span><p>Why you wrote it, and what you've learned doing the work.</p></div></div></div>`),
			quotesSection("Testimonials", 3),
			faqSection("FAQ · Before you buy", "Before you buy", "lav", [][2]string{
				{"What exactly do I get?", "Say what the file is, its size and how it reads."},
				{"How do I get the file?", "Right after you pay, a page opens with the download, and we email you a link."},
				{"Who takes my payment?", "Your payment goes through Stripe. Stripe emails your receipt, with any tax."},
			}),
			section("product", "Product · second Buy button", "sp-sec lav top0", "lavender", "",
				`<div class="sp-in nar"><rc-product data-pb-id="buy-2" data-pb-name="Product · second Buy button" product="`+e(o.ProductID)+`" button="Buy for {price}" price="1" offer="1" layout="final" eyebrow="Get it now"></rc-product></div>`),
		)
	}
	// Blank: one empty section on the theme.
	return section("text", "Text", "sp-sec", "white", "",
		`<div class="sp-in nar"><div class="prose"><h1>`+name+`</h1><p>Start writing, or add blocks from the left.</p></div></div>`)
}

// section is a top-level block.
func section(kind, layer, class, bg, space, inner string) string {
	id := kind
	return fmt.Sprintf(`<section data-pb="%s" data-pb-id="%s" data-pb-name="%s" data-pb-bg="%s" data-pb-space="%s" class="%s">%s</section>`,
		kind, idFor(id), html.EscapeString(layer), bg, nz(space, "normal"), class, inner)
}

func nz(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// idFor names a template's block (UniqueIDs keeps a page's ids unique).
func idFor(kind string) string { return "b-" + kind }

func join(parts ...string) string { return strings.Join(parts, "\n") }

func formSection(kind, layer, form, heading, hint string) string {
	e := html.EscapeString
	return section(kind, layer, "sp-sec cmp lav", "lavender", "compact",
		`<div class="sp-in nar"><rc-form data-pb-id="subscribe" data-pb-name="Signup form" form="`+e(form)+`" heading="`+e(heading)+`" hint="`+e(hint)+`"></rc-form></div>`)
}

func featuresSection(kind, layer, eyebrow, heading string, items []string) string {
	var b strings.Builder
	for i, it := range items {
		fmt.Fprintf(&b, `<li><span class="n">%02d</span><span class="t"><b>%s</b><span class="d">One line on what it covers.</span></span></li>`, i+1, html.EscapeString(it))
	}
	return section(kind, layer, "sp-sec w", "white", "",
		`<div class="sp-in"><div class="sec-head"><span class="eyebrow">`+html.EscapeString(eyebrow)+`</span><h2 class="sp-h2">`+html.EscapeString(heading)+`</h2></div><ol class="chapters">`+b.String()+`</ol></div>`)
}

func quotesSection(layer string, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(`<figure class="quote"><blockquote>“What a reader said about it, in their words.”</blockquote><figcaption><span class="qa" aria-hidden="true">AB</span><span class="qn"><b>A reader</b><span class="qr">Who they are</span></span></figcaption></figure>`)
	}
	return section("testimonials", layer, "sp-sec", "white", "",
		`<div class="sp-in"><div class="sec-head"><span class="eyebrow">From readers</span><h2 class="sp-h2">What people say</h2></div><div class="quotes">`+b.String()+`</div></div>`)
}

func faqSection(layer, heading, bg string, qa [][2]string) string {
	class, bgName := "sp-sec", "white"
	if bg == "lav" {
		class, bgName = "sp-sec lav", "lavender"
	}
	var b strings.Builder
	for i, x := range qa {
		open := ""
		if i == 0 {
			open = " open"
		}
		b.WriteString(`<details` + open + `><summary>` + html.EscapeString(x[0]) + `</summary><p>` + html.EscapeString(x[1]) + `</p></details>`)
	}
	return section("faq", layer, class, bgName, "",
		`<div class="sp-in nar"><div class="sec-head"><span class="eyebrow">Questions</span><h2 class="sp-h2">`+html.EscapeString(heading)+`</h2></div><div class="faq">`+b.String()+`</div></div>`)
}

func initials(name string) string {
	f := strings.Fields(name)
	out := ""
	for _, w := range f {
		for _, r := range w {
			out += strings.ToUpper(string(r))
			break
		}
		if len(out) >= 2 {
			break
		}
	}
	if out == "" {
		return "A"
	}
	return out
}
