package demo

import (
	"fmt"
	"strings"
)

// The demo emails of the composer boards. @@file.png@@ marks an image of
// assets/composer; the seeder swaps in its library URL.

const plexFont = `'IBM Plex Sans', Helvetica, Arial, sans-serif`

// digestHead is shared by the Weekly Digest versions and the saved template.
const digestHead = `  <mj-head>
    <mj-title>%s</mj-title>
    <mj-preview>%s</mj-preview>
    <mj-attributes>
      <mj-all font-family="Georgia, 'Times New Roman', serif"></mj-all>
      <mj-text color="#1F2937" padding="0px"></mj-text>
    </mj-attributes>
  </mj-head>`

const digestHeader = `    <!-- rc:header-logo -->
    <mj-section background-color="#ffffff" padding="20px 24px" border-bottom="1px solid #EEF0F3">
      <mj-column width="40%" vertical-align="middle">
        <mj-image src="@@acme-logo.png@@" alt="Acme" width="96px" height="28px" align="left" padding="0px"></mj-image>
      </mj-column>
      <mj-column width="60%" vertical-align="middle">
        <mj-text align="right" font-family="` + plexFont + `" font-size="12px" color="#5B6472"><a href="https://acme.io/blog" style="color:#5B6472;text-decoration:none">Blog</a> · <a href="https://docs.acme.io" style="color:#5B6472;text-decoration:none">Docs</a> · <a href="https://acme.io/changelog" style="color:#5B6472;text-decoration:none">Changelog</a></mj-text>
      </mj-column>
    </mj-section>`

const digestFooter = `    <!-- rc:footer -->
    <mj-section background-color="#F3F4F6" padding="18px 24px">
      <mj-column>
        <mj-text align="center" font-family="` + plexFont + `" font-size="12px" line-height="1.6" color="#5B6472">Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105<br><a href="{{ unsubscribe_url }}" style="color:#0F766E;text-decoration:none">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#0F766E;text-decoration:none">Email preferences</a> · <a href="{{ view_in_browser_url }}" style="color:#0F766E;text-decoration:none">View in browser</a></mj-text>
      </mj-column>
    </mj-section>`

func button(href, label string) string {
	return `<mj-button href="` + href + `" align="left" background-color="#0F766E" color="#ffffff" font-family="` + plexFont + `" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">` + label + `</mj-button>`
}

func digestCards(padding string) string {
	return `    <!-- rc:article-cards -->
    <mj-section background-color="#ffffff" padding="` + padding + `">
      <mj-column width="50%" padding="0px 8px 0px 0px" css-class="article-card">
        <mj-image src="@@article-card.png@@" alt="Feature flags dashboard" width="268px" height="120px" padding="0px 0px 10px"></mj-image>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Feature flags are generally available</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">Target rollouts by plan, region or percentage, with an audit trail for every change.</mj-text>
        ` + button("https://docs.acme.io/flags", "Read the docs") + `
      </mj-column>
      <mj-column width="50%" padding="0px 0px 0px 8px" css-class="article-card">
        <mj-image src="@@article-card.png@@" alt="Audit log export settings" width="268px" height="120px" padding="0px 0px 10px"></mj-image>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Audit log exports</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">Stream every workspace event to S3 or your SIEM, with 13 months of history.</mj-text>
        ` + button("https://acme.io/changelog", "See the changelog") + `
      </mj-column>
    </mj-section>`
}

// digestMJML is Weekly Digest #142. stage 1 is the copy of the template,
// 9 adds the article cards, 14 is the version on the Composer board.
func digestMJML(stage int, subject, preheader string) string {
	var b strings.Builder
	b.WriteString("<mjml>\n" + fmt.Sprintf(digestHead, xmlText(subject), xmlText(preheader)) + "\n  <mj-body width=\"600px\" background-color=\"#F3F4F6\">\n")
	b.WriteString(digestHeader + "\n")
	headline, intro := "Three launches, one deprecation, and the roadmap for Q4", `Hi {{ first_name | default: "there" }}, here's what shipped this week at Acme, what we're retiring in November, and where the next quarter is headed.`
	if stage < 9 {
		headline, intro = "This week at Acme", `Hi {{ first_name | default: "there" }}, the launches, guides and changelog highlights of the week.`
	}
	b.WriteString(`    <mj-section background-color="#ffffff" padding="20px 24px 8px">
      <mj-column>
        <mj-image src="@@q4-roadmap.png@@" alt="Q4 roadmap illustration" width="552px" height="180px" padding="0px 0px 16px"></mj-image>
        <mj-text font-size="26px" line-height="1.2" font-weight="700" color="#0B1220" padding="0px 0px 10px">` + headline + `</mj-text>
        <mj-text font-size="15px" line-height="1.55">` + intro + `</mj-text>
      </mj-column>
    </mj-section>
`)
	cardPadding := "16px 24px"
	if stage < 14 {
		cardPadding = "12px 24px"
	}
	b.WriteString(digestCards(cardPadding) + "\n")
	if stage >= 9 {
		b.WriteString(`    <mj-section background-color="#ffffff" padding="8px 24px">
      <mj-column>
        <mj-divider border-width="1px" border-color="#E3E6EB" padding="0px"></mj-divider>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="8px 24px 16px">
      <mj-column>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Retiring the v1 API on 30 November</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151">If you still call api.acme.io/v1, the <a href="https://docs.acme.io/migrate-v2" style="color:#0F766E">migration guide</a> covers every endpoint. Most teams finish in an afternoon.</mj-text>
      </mj-column>
    </mj-section>
`)
	}
	b.WriteString(digestFooter + "\n  </mj-body>\n</mjml>\n")
	return b.String()
}

func xmlText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// flashSaleMJML is "Flash sale: 48 hours only" as on the ComposerWarnings
// board: a hero image without alt text, a button to a blocklisted domain,
// four plan cards whose pasted HTML pushes the size near Gmail's clipping
// limit, and (from version 6) no footer.
func flashSaleMJML(subject, preheader string, withFooter bool, cardBloat int) string {
	var b strings.Builder
	b.WriteString("<mjml>\n  <mj-head>\n    <mj-title>" + xmlText(subject) + "</mj-title>\n    <mj-preview>" + xmlText(preheader) + "</mj-preview>\n" +
		`    <mj-attributes>
      <mj-all font-family="Georgia, 'Times New Roman', serif"></mj-all>
      <mj-text color="#1F2937" padding="0px"></mj-text>
    </mj-attributes>
  </mj-head>
  <mj-body width="600px" background-color="#F3F4F6">
    <!-- rc:header-logo -->
    <mj-section background-color="#ffffff" padding="20px 24px" border-bottom="1px solid #EEF0F3">
      <mj-column width="40%" vertical-align="middle">
        <mj-image src="@@acme-logo.png@@" alt="Acme" width="96px" height="28px" align="left" padding="0px"></mj-image>
      </mj-column>
      <mj-column width="60%" vertical-align="middle">
        <mj-text align="right" font-family="` + plexFont + `" font-size="12px" color="#5B6472"><a href="https://acme.io/pricing" style="color:#5B6472;text-decoration:none">Pricing</a> · <a href="https://docs.acme.io" style="color:#5B6472;text-decoration:none">Docs</a></mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="20px 24px 8px">
      <mj-column>
        <mj-image src="@@sale-hero.png@@" width="552px" height="220px" padding="0px 0px 16px"></mj-image>
        <mj-text font-size="28px" line-height="1.15" font-weight="700" color="#0B1220" padding="0px 0px 10px">48 hours. 30% off every annual plan.</mj-text>
        <mj-text font-size="15px" line-height="1.55">Hi {{ first_name | default: "there" }}, until Thursday at midnight every annual plan is 30% off, including upgrades from monthly.</mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="8px 24px 20px">
      <mj-column>
        <mj-button href="https://go.acme-promo.link/sale" align="center" background-color="#B91C1C" color="#ffffff" font-family="` + plexFont + `" font-size="15px" font-weight="700" border-radius="6px" inner-padding="14px 28px" padding="0px">Claim your discount</mj-button>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="8px 24px">
      <mj-column>
        <mj-divider border-width="1px" border-color="#E3E6EB" padding="0px"></mj-divider>
      </mj-column>
    </mj-section>
`)
	plans := [][2][2]string{
		{{"Starter · $9/mo", "Now $75.60 a year."}, {"Team · $29/mo", "Now $243.60 a year."}},
		{{"Business · $79/mo", ""}, {"Enterprise", ""}},
	}
	for row, pair := range plans {
		pad := "8px 24px 16px"
		if row == 1 {
			pad = "0px 24px 16px"
		}
		b.WriteString(`    <!-- rc:product-cards -->
    <mj-section background-color="#ffffff" padding="` + pad + `">
`)
		for col, p := range pair {
			cpad := "0px 8px 0px 0px"
			if col == 1 {
				cpad = "0px 0px 0px 8px"
			}
			b.WriteString(`      <mj-column width="50%" padding="` + cpad + `">
        <mj-image src="@@plan-card.png@@" alt="` + p[0] + ` plan" width="268px" height="110px" padding="0px 0px 8px"></mj-image>
        <mj-text font-size="16px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 4px">` + p[0] + `</mj-text>
`)
			if p[1] != "" || cardBloat > 0 {
				b.WriteString("        <mj-text font-size=\"13px\" line-height=\"1.5\" color=\"#374151\">" + pastedPrice(p[1], cardBloat) + "</mj-text>\n")
			}
			b.WriteString("      </mj-column>\n")
		}
		b.WriteString("    </mj-section>\n")
	}
	if withFooter {
		b.WriteString(`    <!-- rc:footer -->
    <mj-section background-color="#F3F4F6" padding="18px 24px">
      <mj-column>
        <mj-text align="center" font-family="` + plexFont + `" font-size="12px" line-height="1.6" color="#5B6472">Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105<br><a href="{{ unsubscribe_url }}" style="color:#0F766E;text-decoration:none">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#0F766E;text-decoration:none">Email preferences</a></mj-text>
      </mj-column>
    </mj-section>
`)
	}
	b.WriteString("  </mj-body>\n</mjml>\n")
	return b.String()
}

// pastedPrice is a price line as it arrives when copied from a pricing page
// built in a page builder: nested spans and an Outlook table fallback with
// the same inline CSS repeated on every element. It renders as one short
// line; its weight is what pushes the email toward Gmail's clipping limit.
func pastedPrice(text string, bloat int) string {
	const style = `font-family:Georgia,'Times New Roman',serif;font-size:13px;line-height:19.5px;color:#374151;font-weight:400;font-style:normal;font-variant-ligatures:normal;font-variant-caps:normal;letter-spacing:normal;text-align:left;text-indent:0px;text-transform:none;white-space:normal;word-spacing:0px;-webkit-text-stroke-width:0px;text-decoration-thickness:initial;text-decoration-style:initial;text-decoration-color:initial;background-color:transparent;mso-line-height-rule:exactly`
	var b strings.Builder
	open := 0
	for i := 0; i < bloat; i++ {
		b.WriteString("\n" + strings.Repeat("    ", 5+i%6) + `<span style="` + style + `">`)
		open++
	}
	b.WriteString(text)
	for ; open > 0; open-- {
		b.WriteString("</span>")
	}
	return b.String()
}

// octoberMJML is October Product Update (sent), offered as "Copy of …".
func octoberMJML() string {
	return `<mjml>
  <mj-head>
    <mj-title>October at Acme: what shipped</mj-title>
    <mj-preview>Feature flags preview, faster exports and the Postal warm-up.</mj-preview>
    <mj-attributes>
      <mj-all font-family="Georgia, 'Times New Roman', serif"></mj-all>
      <mj-text color="#1F2937" padding="0px"></mj-text>
    </mj-attributes>
  </mj-head>
  <mj-body width="600px" background-color="#F3F4F6">
` + digestHeader + `
    <mj-section background-color="#ffffff" padding="20px 24px 8px">
      <mj-column>
        <mj-image src="@@october-hero.png@@" alt="October at Acme" width="552px" height="200px" padding="0px 0px 16px"></mj-image>
        <mj-text font-size="26px" line-height="1.2" font-weight="700" color="#0B1220" padding="0px 0px 10px">October at Acme: what shipped</mj-text>
        <mj-text font-size="15px" line-height="1.55">Hi {{ first_name | default: "there" }}, five things shipped this month. Here are the ones you'll notice first.</mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="8px 24px">
      <mj-column>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Feature flags, in preview</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">Roll a change out to 5% of accounts, watch the numbers, then go to 100%. <a href="https://docs.acme.io/flags" style="color:#0F766E">Read the docs</a>.</mj-text>
      </mj-column>
    </mj-section>
    <mj-section background-color="#ffffff" padding="8px 24px 16px">
      <mj-column>
        <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Exports are four times faster</mj-text>
        <mj-text font-size="14px" line-height="1.5" color="#374151">CSV and JSON exports now stream, so a million rows arrive in under a minute.</mj-text>
      </mj-column>
    </mj-section>
` + digestFooter + `
  </mj-body>
</mjml>
`
}

// savedSections are the four reusable blocks of "Saved sections · 4".
var savedSections = []struct{ name, description, mjml string }{
	{"Changelog highlights", "Three bullet points with links", `<mj-section background-color="#ffffff" padding="12px 24px">
  <mj-column>
    <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 8px">From the changelog</mj-text>
    <mj-text font-size="14px" line-height="1.6" color="#374151">• <a href="https://acme.io/changelog" style="color:#0F766E">Faster search in the dashboard</a><br>• <a href="https://acme.io/changelog" style="color:#0F766E">Webhooks retry for 72 hours</a><br>• <a href="https://acme.io/changelog" style="color:#0F766E">New regions: Frankfurt and Sydney</a></mj-text>
  </mj-column>
</mj-section>`},
	{"Customer quote", "A pull quote with the customer's name", `<mj-section background-color="#F0FDFA" padding="20px 32px">
  <mj-column>
    <mj-text font-size="18px" line-height="1.5" font-style="italic" color="#0B1220" padding="0px 0px 8px">“We moved 40 services to feature flags in a week and haven't had a bad deploy since.”</mj-text>
    <mj-text font-family="` + plexFont + `" font-size="12px" color="#5B6472">Priya Natarajan, platform lead at Northwind Labs</mj-text>
  </mj-column>
</mj-section>`},
	{"Webinar invite", "Date, time and a register button", `<mj-section background-color="#ffffff" padding="16px 24px">
  <mj-column>
    <mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px">Live: rolling out safely with feature flags</mj-text>
    <mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">Thursday 23 October · 17:00 UTC · 45 minutes with Q&amp;A</mj-text>
    ` + button("https://acme.io/webinars/flags", "Save your seat") + `
  </mj-column>
</mj-section>`},
	{"Docs callout", "A tinted box pointing to the docs", `<mj-section background-color="#ffffff" padding="8px 24px">
  <mj-column background-color="#F3F4F6" padding="14px 16px" border-radius="8px">
    <mj-text font-family="` + plexFont + `" font-size="13px" line-height="1.5" color="#374151">New to the API? The <a href="https://docs.acme.io/quickstart" style="color:#0F766E">quickstart</a> gets you from key to first request in five minutes.</mj-text>
  </mj-column>
</mj-section>`},
}
