package demo

import (
	"fmt"
	"html"
	"strings"
)

// reportEmailHTML is the sent email of a demo campaign that has no stored
// content: a 600px newsletter with a header, a hero, articles and a footer.
// The October Product Update is the email the Report board's click map
// draws (shown there at 340px, so the sizes here are the board's × 1.765).
func reportEmailHTML(name string, oct bool) string {
	const ph = "display:flex;align-items:center;justify-content:center;background:#E9ECF0;color:#5B6472;font-family:'IBM Plex Sans',Arial,sans-serif;font-size:19px;font-weight:500;border:2px dashed #C9CED6;box-sizing:border-box"
	const plex = "font-family:'IBM Plex Sans',Arial,sans-serif"
	const btn = "display:inline-block;padding:7px 12px;background:#0F766E;color:#ffffff;" + plex + ";font-size:14px;font-weight:600;border-radius:5px;text-decoration:none"
	const link = "color:inherit;text-decoration:none"
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>` + esc(name) + `</title></head>`)
	b.WriteString(`<body style="margin:0;background:#ffffff"><div style="width:600px;margin:0 auto;background:#ffffff;font-family:Georgia,'Times New Roman',serif;font-size:16px;line-height:1.45;color:#374151">`)
	// Header: logo and navigation.
	b.WriteString(`<div style="display:flex;justify-content:space-between;align-items:center;padding:14px 18px;border-bottom:2px solid #EEF0F3">`)
	b.WriteString(`<a href="https://acme.io" style="` + link + `"><div style="` + ph + `;width:78px;height:25px">Acme</div></a>`)
	b.WriteString(`<span style="` + plex + `;font-size:14px;padding-right:64px"><a href="https://acme.io/blog" style="` + link + `">Blog</a> · <a href="https://docs.acme.io" style="` + link + `">Docs</a> · <a href="https://acme.io/changelog" style="` + link + `">Changelog</a></span>`)
	b.WriteString(`</div>`)
	title, intro := name, "Here's what's new at Acme this week, in five minutes."
	type article struct{ heading, body, label, url string }
	var cards []article
	var closing article
	if oct {
		title, intro = "October Product Update: three launches", "Hi there, here's what shipped in October…"
		cards = []article{
			{"Feature flags GA", "", "Read the docs", "https://docs.acme.io/flags"},
			{"Audit log exports", "", "See changelog", "https://acme.io/changelog/audit-exports"},
		}
		closing = article{"Retiring the v1 API on 30 November", "The migration guide covers every endpoint.", "Read the guide", "https://docs.acme.io/migrate-v2"}
	} else {
		cards = []article{
			{"From the changelog", "", "Read the post", "https://acme.io/blog/" + fileSlug(name)},
			{"Customer story", "", "See how they did it", "https://acme.io/customers/northwind"},
		}
		closing = article{"Office hours on Thursday", "Bring your questions about the API and the dashboard.", "Save a seat", "https://acme.io/office-hours"}
	}
	heroURL, imgURLs := "https://acme.io/blog/"+fileSlug(name), []string{"https://acme.io/features/" + fileSlug(cards[0].heading), "https://acme.io/features/" + fileSlug(cards[1].heading)}
	b.WriteString(`<div style="padding:18px"><a href="` + esc(heroURL) + `" style="` + link + `"><div style="` + ph + `;height:124px;margin-bottom:11px">Hero image</div></a>`)
	b.WriteString(`<div style="font-size:21px;font-weight:700;color:#0B1220">` + esc(title) + `</div><div>` + esc(intro) + `</div></div>`)
	b.WriteString(`<div style="display:flex;gap:14px;padding:0 18px 18px">`)
	for i, c := range cards {
		fmt.Fprintf(&b, `<div style="flex:1"><a href="%s" aria-label="%s image" style="display:block;%s"><div style="%s;height:78px"></div></a>`, esc(imgURLs[i]), esc(c.heading), link, ph)
		fmt.Fprintf(&b, `<div style="font-weight:700;margin:7px 0 4px;color:#0B1220">%s</div>`, esc(c.heading))
		fmt.Fprintf(&b, `<a href="%s" style="%s">%s</a></div>`, esc(c.url), btn, esc(c.label))
	}
	b.WriteString(`</div>`)
	b.WriteString(`<div style="padding:11px 18px 18px"><div style="font-weight:700;color:#0B1220">` + esc(closing.heading) + `</div>`)
	fmt.Fprintf(&b, `<div>%s <a href="%s" style="color:inherit;text-decoration:underline">%s</a></div></div>`, esc(closing.body), esc(closing.url), esc(closing.label))
	// Footer.
	b.WriteString(`<div style="padding:14px;background:#F3F4F6;` + plex + `;font-size:13px;color:#5B6472;text-align:center">`)
	b.WriteString(`Acme Inc · 500 Market St · <a href="{{ unsubscribe_url }}" style="color:inherit;text-decoration:underline">Unsubscribe</a> · `)
	b.WriteString(`<a href="{{ preferences_url }}" style="color:inherit;text-decoration:underline">Email preferences</a> · `)
	b.WriteString(`<a href="{{ view_online_url }}" style="color:inherit;text-decoration:underline">View in browser</a>`)
	b.WriteString(`</div></div></body></html>`)
	return b.String()
}

// fileSlug is "weekly-digest-141".
func fileSlug(name string) string {
	return strings.Trim(strings.ToLower(nonAlnum.ReplaceAllString(name, "-")), "-")
}
