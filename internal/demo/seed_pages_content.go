package demo

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// The demo pages' blocks (PublicSalesPage, PublicPagesStates, PagesList):
// the same markup the page builder edits, with the boards' copy.

func pgEsc(s string) string { return html.EscapeString(s) }

// pgSection is a top-level block.
func pgSection(kind, id, layer, class, bg, space, inner string) string {
	return fmt.Sprintf(`<section data-pb="%s" data-pb-id="%s" data-pb-name="%s" data-pb-bg="%s" data-pb-space="%s" class="%s">%s</section>`,
		kind, id, pgEsc(layer), bg, space, class, inner)
}

type pgQuote struct{ text, initials, name, role string }

func pgQuotes(qs []pgQuote) string {
	var b strings.Builder
	for _, q := range qs {
		fmt.Fprintf(&b, `<figure class="quote"><blockquote>“%s”</blockquote><figcaption><span class="qa" aria-hidden="true">%s</span><span class="qn"><b>%s</b><span class="qr">%s</span></span></figcaption></figure>`,
			pgEsc(q.text), q.initials, pgEsc(q.name), pgEsc(q.role))
	}
	return b.String()
}

func pgFAQ(openFirst bool, qa [][2]string) string {
	var b strings.Builder
	for i, x := range qa {
		open := ""
		if openFirst && i == 0 {
			open = " open"
		}
		fmt.Fprintf(&b, `<details%s><summary>%s</summary><p>%s</p></details>`, open, pgEsc(x[0]), pgEsc(x[1]))
	}
	return b.String()
}

// fieldGuideHTML is The Deliverability Field Guide's sales page: version
// 8 went live on 1 Oct; the draft (9) has a longer hero line, Hannah
// Weiss's quote and one more question.
func fieldGuideHTML(draft bool, product, cover string) string {
	lead := "Get your newsletter into the inbox. Eight chapters on authentication, warm-up and list hygiene."
	if draft {
		lead = "Get your newsletter into the inbox and keep it there. Eight practical chapters on authentication, warm-up, list hygiene and what to do when a mailbox provider starts pushing back."
	}
	chapters := [][2]string{
		{"Authentication without guesswork", "SPF, DKIM and DMARC: what each one proves, and the order to add them."},
		{"Warming up a domain", "A four-week schedule for a new domain or IP, with the numbers to watch each day."},
		{"Keeping your list clean", "Bounces, complaints and quiet subscribers, and when to let them go."},
		{"Reading the dashboards", "Gmail Postmaster Tools and Microsoft SNDS, screen by screen."},
		{"Content that filters trust", "Links, images and wording that raise flags, and the myths that don't."},
		{"When a provider defers you", "A 24-hour playbook for 421s, rate limits and sudden drops."},
		{"Blocklists", "Finding out you're listed, getting delisted and staying off."},
		{"Monitoring that warns you first", "The five numbers to check every week, and the alerts worth setting."},
	}
	var ch strings.Builder
	for i, c := range chapters {
		fmt.Fprintf(&ch, `<li><span class="n">%02d</span><span class="t"><b>%s</b><span class="d">%s</span></span></li>`, i+1, pgEsc(c[0]), pgEsc(c[1]))
	}
	quotes := []pgQuote{
		{"Chapter 6 got us out of a Gmail deferral in two days. Clear steps, no hand-waving.", "PN", "Priya Natarajan", "Runs a 12,000-reader newsletter"},
		{"I finally understand what my DMARC reports are telling me. The checklists sit next to my keyboard.", "OH", "Omar Haddad", "Founder, Haddad Studio"},
	}
	if draft {
		quotes = append(quotes, pgQuote{"We warmed up a new domain with the four-week schedule and never saw a deferral.", "HW", "Hannah Weiss", "Weekly Digest Insider"})
	}
	faq := [][2]string{
		{"What exactly do I get?", "A PDF of 86 pages, 4.2 MB: eight chapters, each with a one-page checklist. It reads well on a laptop, a tablet or on paper."},
		{"How do I get the file?", "Right after you pay, a page opens with the download, and we email you a link. The link works for 7 days and 5 downloads."},
		{"I lost the link. Can I get a new one?", "Yes. Sign in at news.acme.io/account and choose “Send me a new link”. It always goes to the address you bought with."},
		{"Who takes my payment?", "Acme Media, through Stripe. Your card details go to Stripe, never to Acme or to this site. Stripe emails your receipt, with any tax."},
	}
	if draft {
		faq = append(faq, [2]string{"Do I have to subscribe to anything?", "No. Buying doesn't add you to any newsletter. Tick “Also send me the Weekly Digest” only if you want it."})
	}
	return strings.Join([]string{
		pgSection("hero", "b-hero", "Hero", "sp-hero", "lavender", "normal",
			`<div class="sp-in"><div class="hero-grid"><div class="hero-copy" data-pb-id="b-hero-copy" data-pb-name="Heading, text and checklist"><span class="eyebrow">Field guide · PDF, 86 pages</span><h1>The Deliverability Field Guide</h1>`+
				`<p class="sp-lead">`+pgEsc(lead)+`</p><ul class="ticks"><li>8 chapters and 86 pages, with a checklist after each</li><li>Real DNS records, bounce codes and dashboards, explained</li><li>Written by Sofia Rossi, who runs Acme's sending</li></ul>`+
				`<rc-product data-pb-id="buy" data-pb-name="Product · The Deliverability Field Guide" id="buy" product="`+product+`" button="Buy for {price}" price="1" offer="1" layout="box"></rc-product></div>`+
				`<div class="hero-art"><img data-pb="image" data-pb-id="b-cover" data-pb-name="Image · cover" class="cov xl" src="`+pgEsc(cover)+`" alt="Cover of The Deliverability Field Guide" width="300" height="400"></div></div></div>`),
		pgSection("features", "b-inside", "Features list · What's inside", "sp-sec w", "white", "normal",
			`<div class="sp-in"><div class="sec-head"><span class="eyebrow">What's inside</span><h2 class="sp-h2">Eight chapters, each ending in a one-page checklist</h2></div><ol class="chapters">`+ch.String()+`</ol></div>`),
		pgSection("bio", "b-bio", "Author bio · Sofia Rossi", "sp-sec lav", "lavender", "normal",
			`<div class="sp-in nar"><div class="bio"><span class="av7" aria-hidden="true">SR</span><div class="tx"><span class="eyebrow">About the author</span><h2 class="sp-h2">Sofia Rossi</h2>`+
				`<span class="role">Editor of the Weekly Digest at Acme</span><p>Sofia looks after Acme's newsletter and the mail behind it. She led the Weekly Digest's move to news.acme.io in September, and this guide is the playbook the team wrote along the way: what they checked, in what order, and what they would do differently.</p>`+
				`<a href="/tag/deliverability" class="st-lnk">Read her posts</a></div></div></div>`),
		pgSection("testimonials", "b-quotes", "Testimonials", "sp-sec", "white", "normal",
			`<div class="sp-in"><div class="sec-head"><span class="eyebrow">From readers</span><h2 class="sp-h2">What people did with it</h2></div><div class="quotes">`+pgQuotes(quotes)+`</div></div>`),
		pgSection("faq", "b-faq", "FAQ · Before you buy", "sp-sec lav", "lavender", "normal",
			`<div class="sp-in nar"><div class="sec-head"><span class="eyebrow">Questions</span><h2 class="sp-h2">Before you buy</h2></div><div class="faq">`+pgFAQ(true, faq)+`</div></div>`),
		pgSection("product", "b-final", "Product · second Buy button", "sp-sec lav top0", "lavender", "normal",
			`<div class="sp-in nar"><rc-product data-pb-id="buy-2" data-pb-name="Product · second Buy button" product="`+product+`" button="Buy for {price}" price="1" offer="1" layout="final" eyebrow="Get the guide"></rc-product></div>`),
	}, "\n")
}

// dnsChecklistHTML is the lead magnet: a focused page with the form and
// the checklist's reasons.
func dnsChecklistHTML(form, cover string) string {
	return strings.Join([]string{
		pgSection("hero", "b-hero", "Hero · the free checklist", "lp-band", "lavender", "normal",
			`<div class="sp-in"><div class="lp-hero"><div class="lp-copy" data-pb-id="b-hero-copy" data-pb-name="Heading and text"><span class="eyebrow">Free checklist · PDF, 2 pages</span>`+
				`<h1>Every DNS record your newsletter needs, on two pages</h1><p class="sp-lead sm">SPF, DKIM, DMARC, the return path and BIMI, in the order to add them. Check your domain in 15 minutes.</p></div>`+
				`<img data-pb="image" data-pb-id="b-cover" data-pb-name="Image · cover" class="cov lg" src="`+pgEsc(cover)+`" alt="Cover of the DNS records checklist" width="150" height="200"></div></div>`),
		pgSection("section", "b-body", "The form and the reasons", "sp-sec cmp top0", "white", "compact",
			`<div class="sp-in"><div class="lp-body"><rc-form data-pb-id="form" data-pb-name="Signup form · DNS checklist" form="`+form+`" heading="Get the checklist" hint="We'll email a confirmation link. The checklist is on the page that opens."></rc-form>`+
				`<div class="bens" data-pb-id="b-bens" data-pb-name="Features list · benefits"><div class="ben"><b>Every record, explained</b>What each one proves, and what it should look like.</div><div class="ben"><b>In the right order</b>SPF and DKIM before DMARC, so nothing bounces on the way.</div><div class="ben"><b>The one most people miss</b>A two-minute check of the return path.</div></div>`+
				`<figure class="quote" data-pb-id="b-quote" data-pb-name="Testimonial"><blockquote>“Found two broken DKIM keys in the first ten minutes.”</blockquote><figcaption><span class="qa" aria-hidden="true">KW</span><span class="qn"><b>Kenji Watanabe</b><span class="qr">Editor, Tokyo Dev Weekly</span></span></figcaption></figure>`+
				`<div class="faq" data-pb-id="b-faq" data-pb-name="FAQ">`+pgFAQ(false, [][2]string{
				{"Is it really free?", "Yes. You confirm your email and get the PDF and the Weekly Digest. Unsubscribe any time."},
				{"Which providers does it cover?", "Any provider. The examples use Amazon SES, Postmark and Mailgun."},
			})+`</div></div></div>`),
	}, "\n")
}

// launchKitHTML is the launch page: a countdown to its opening (6 Oct,
// 09:00 New York time in the sample), then the Buy button in place of the
// waitlist.
func launchKitHTML(product, form, cover string, opens time.Time) string {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		ny = time.UTC
	}
	opens = opens.In(ny)
	return strings.Join([]string{
		pgSection("hero", "b-hero", "Hero · the launch", "lp-band", "lavender", "normal",
			`<div class="sp-in"><div class="lp-hero"><div class="lp-copy" data-pb-id="b-hero-copy" data-pb-name="Heading, text and countdown">`+
				`<span class="eyebrow" data-pb-id="eb-before">Opens `+opens.Format("2 January, 15:04")+` New York time</span><span class="eyebrow" data-pb-id="eb-after">Open now</span><h1>Newsletter Launch Kit</h1>`+
				`<p class="sp-lead sm" data-pb-id="lead-before">Twelve templates and a checklist to get your first issues out the door. Pay what you want when it opens.</p>`+
				`<p class="sp-lead sm" data-pb-id="lead-after">Twelve templates and a checklist to get your first issues out the door. Pay what you want.</p>`+
				`<rc-countdown data-pb-id="countdown" data-pb-name="Countdown" at="`+opens.Format("2006-01-02T15:04")+`" tz="America/New_York" units="days,hours,minutes" end="reveal" reveal="eb-after,lead-after,buy" hide="eb-before,lead-before,waitlist" label="Opens"></rc-countdown>`+
				`<rc-product data-pb-id="buy" data-pb-name="Product · Launch Kit Buy button" id="buy" product="`+product+`" button="Buy the Launch Kit" price="1" offer="1" layout="box"></rc-product></div>`+
				`<img data-pb="image" data-pb-id="b-cover" data-pb-name="Image · cover" class="cov lg" src="`+pgEsc(cover)+`" alt="Cover of the Newsletter Launch Kit" width="150" height="200"></div></div>`),
		pgSection("section", "b-body", "The waitlist and what's in the kit", "sp-sec cmp top0", "white", "compact",
			`<div class="sp-in"><div class="lp-body"><rc-form data-pb-id="waitlist" data-pb-name="Signup form · the waitlist" form="`+form+`" heading="Join the waitlist"></rc-form>`+
				`<div class="bens" data-pb-id="b-bens" data-pb-name="Features list"><div class="ben"><b>12 MJML templates</b>Welcome, digest, launch and re-engagement emails, ready to edit.</div><div class="ben"><b>A launch checklist</b>30 steps from domain setup to your third issue.</div><div class="ben"><b>Works anywhere</b>In ReputeCast or any tool that reads MJML 4.</div></div></div></div>`),
	}, "\n")
}

// dmarcLiveHTML is the webinar: the event in each reader's time (it starts
// at start, 17:00 UTC), the host, the agenda and the registration form,
// which writes the date to dmarc_live_at.
func dmarcLiveHTML(form string, start time.Time) string {
	start = start.UTC()
	return strings.Join([]string{
		pgSection("hero", "b-hero", "Hero · the session", "lp-band", "lavender", "normal",
			`<div class="sp-in"><div class="lp-copy solo" data-pb-id="b-hero-copy" data-pb-name="Heading and text"><span class="eyebrow">Live session · 30 minutes</span><h1>Live: Fix your DMARC setup in 30 minutes</h1>`+
				`<p class="sp-lead sm">Marcus Chen takes a real domain from p=none to p=quarantine, live, and answers your questions as he goes.</p></div></div>`),
		pgSection("section", "b-body", "Details, host, agenda and registration", "sp-sec cmp top0", "white", "compact",
			`<div class="sp-in nar"><div class="lp-body"><rc-event data-pb-id="event" data-pb-name="Event details" date="`+start.Format("2006-01-02")+`" start="`+start.Format("15:04")+`" end="`+start.Add(30*time.Minute).Format("15:04")+`" tz="UTC" local="1" where="online" platform="Zoom" link="https://acme.zoom.us/j/88412093371" calendar="1" title="Live: Fix your DMARC setup in 30 minutes" reminders="1 day and 1 hour before" field="dmarc_live_at"></rc-event>`+
				`<div class="host" data-pb-id="b-host" data-pb-name="Speaker · Marcus Chen"><span class="hav" aria-hidden="true">MC</span><span class="hn"><b>Marcus Chen</b><span class="hr">Runs email infrastructure at Acme · your host</span></span></div>`+
				`<div data-pb-id="b-agenda" data-pb-name="Agenda"><h2 class="agenda-h">Agenda</h2><ol class="agenda"><li><span class="m">5 min</span>Reading your DMARC reports</li><li><span class="m">15 min</span>From p=none to p=quarantine, live</li><li><span class="m">10 min</span>Your questions</li></ol></div>`+
				`<rc-form data-pb-id="register" data-pb-name="Signup form · DMARC live" form="`+form+`" heading="Save your seat" hint="You join this session's list only. No newsletter unless you sign up for one."></rc-form></div></div>`),
	}, "\n")
}

// aboutHTML is the About page: who writes the newsletter, and a subscribe band.
func aboutHTML(form string) string {
	return strings.Join([]string{
		pgSection("hero", "b-hero", "Hero", "sp-hero", "lavender", "normal",
			`<div class="sp-in nar"><div class="hero-copy" data-pb-id="b-hero-copy" data-pb-name="Heading and text"><span class="eyebrow">About</span><h1>The Acme newsletter</h1>`+
				`<p class="sp-lead">What shipped, what's next and the occasional deep dive, from the people who build Acme. Every Tuesday since 2021, for 48,000 readers.</p></div></div>`),
		pgSection("text", "b-story", "Text · why we write it", "sp-sec", "white", "normal",
			`<div class="sp-in nar"><div class="prose"><h2>Why we write it</h2><p>We started the Weekly Digest to tell our customers what changed in Acme without making them read the changelog. It grew into the place where our engineers explain how things work, and why.</p>`+
				`<p>Each issue has the week's releases, one guide worth your time and a look at what's coming. Product Updates go out when something big ships; Engineering is for the deep dives.</p></div></div>`),
		pgSection("bio", "b-team", "Author bios · the team", "sp-sec lav", "lavender", "normal",
			`<div class="sp-in nar"><div class="bio"><span class="av7" aria-hidden="true">SR</span><div class="tx"><span class="eyebrow">Editor</span><h2 class="sp-h2">Sofia Rossi</h2><span class="role">Edits the Weekly Digest and runs Acme's sending</span>`+
				`<p>Sofia joined Acme's developer relations team in 2022. She writes the guides, picks the links and answers every reply.</p></div></div></div>`),
		pgSection("signup", "b-subscribe", "Signup form · subscribe band", "sp-sec cmp lav", "lavender", "compact",
			`<div class="sp-in nar"><rc-form data-pb-id="subscribe" data-pb-name="Signup form" form="`+form+`" heading="Get the Weekly Digest" hint="One email a week. Unsubscribe any time."></rc-form></div>`),
	}, "\n")
}

// sponsorHTML is Daniel Park's draft: a blank page he's writing.
func sponsorHTML() string {
	return strings.Join([]string{
		pgSection("text", "b-text", "Text · sponsoring", "sp-sec", "white", "normal",
			`<div class="sp-in nar"><div class="prose"><h1>Sponsor the Weekly Digest</h1><p>48,000 developers and engineering leads read the Weekly Digest every Tuesday. One sponsor per issue, clearly marked, written with our editors.</p>`+
				`<h2>What you get</h2><ul><li>A 60-word slot near the top of one issue</li><li>A link with your own UTM parameters</li><li>The issue's open and click numbers a week later</li></ul></div></div>`),
		pgSection("text", "b-contact", "Text · how to book", "sp-sec lav", "lavender", "normal",
			`<div class="sp-in nar"><div class="prose"><h2>How to book</h2><p>Write to sponsors@acme.io with the week you'd like and a line about your product. We reply within two working days.</p></div></div>`),
	}, "\n")
}
