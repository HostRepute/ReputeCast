package website

import (
	"context"
	"html"
	"strings"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// linkEmail is a sign-in, sign-up or email-change email's facts.
type linkEmail struct {
	purpose   string
	to        string
	first     string
	link      string
	code      string
	minutes   int
	postTitle string
	lists     []string
}

// linkEmail composes the email of a link: plain text and a small HTML
// version in the publisher's colours, no tracking, on the System stream.
func (s *Service) linkEmail(ctx context.Context, ws *store.Workspace, c *config, set *forms.Settings, l i18n.Lang, e linkEmail) mailer.Message {
	w := WordsIn(l)
	site := siteTitle(c, set)
	host := hostOnly(strings.TrimSuffix(s.sitePath(ctx, ws), "/"))
	args := i18n.Args{"site": site, "host": host, "email": e.to, "minutes": e.minutes, "code": CodeText(e.code), "title": e.postTitle,
		"lists": strings.Join(e.lists, " and ")}
	greeting := w.Text("mail.hiThere", nil)
	if f := strings.TrimSpace(e.first); f != "" {
		greeting = w.Text("mail.hiName", i18n.Args{"name": f})
	}
	var subject, heading, lead, button, after, foot string
	var code []string
	switch e.purpose {
	case PurposeSignup:
		subject = w.Text("mail.signup.subject", args)
		heading = subject
		lead = w.Text("mail.signup.leadNoPost", args)
		if e.postTitle != "" {
			lead = w.Text("mail.signup.lead", args)
		}
		button = w.Text("mail.signup.button", nil)
		after = w.Text("mail.signup.ignore", nil)
		if len(e.lists) > 0 {
			after = w.Text("mail.signup.lists", args) + " " + after
		}
		foot = w.Text("mail.signup.footer", args)
	case PurposeEmail:
		subject = w.Text("mail.email.subject", args)
		heading = subject
		lead = w.Text("mail.email.lead", args)
		button = w.Text("mail.email.button", nil)
		after = w.Text("mail.email.ignore", nil)
		foot = w.Text("mail.footer", args)
	default:
		subject = w.Text("mail.signin.subject", args)
		heading = w.Text("mail.signin.heading", args)
		lead = greeting + " " + w.Text("mail.signin.lead", args)
		button = w.Text("mail.signin.button", nil)
		code = []string{w.Text("mail.signin.codeIntro", nil), CodeText(e.code)}
		after = w.Text("mail.signin.ignore", args)
		foot = w.Text("mail.footer", args)
	}
	why := w.Text("mail.why."+e.purpose, args)
	var text strings.Builder
	text.WriteString(heading + "\n\n" + lead + "\n\n" + e.link + "\n\n")
	if len(code) == 2 {
		text.WriteString(code[0] + " " + code[1] + "\n\n")
	}
	text.WriteString(after + "\n\n" + foot + "\n" + why + "\n")
	esc := html.EscapeString
	b := set.Branding
	accent, on, bg := b.Palette.Accent, b.Palette.OnAccent, b.Palette.Background
	var h strings.Builder
	h.WriteString(`<!doctype html><html lang="` + esc(string(l.Locale)) + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + esc(subject) + `</title></head>`)
	h.WriteString(`<body style="margin:0;padding:0;background:` + esc(bg) + `;font-family:'IBM Plex Sans',Arial,sans-serif;color:#1F2937">`)
	h.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + esc(bg) + `"><tr><td align="center" style="padding:32px 16px">`)
	h.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border:1px solid #E3E6EB;border-radius:14px"><tr><td style="padding:28px">`)
	h.WriteString(`<p style="margin:0 0 16px;font-size:13px;font-weight:700;color:#5B6472">` + esc(site) + `</p>`)
	h.WriteString(`<h1 style="margin:0 0 12px;font-size:20px;line-height:1.3;color:#0B1220">` + esc(heading) + `</h1>`)
	h.WriteString(`<p style="margin:0 0 20px;font-size:15px;line-height:1.5">` + esc(lead) + `</p>`)
	h.WriteString(`<p style="margin:0 0 20px"><a href="` + esc(e.link) + `" style="display:inline-block;padding:12px 20px;border-radius:9px;background:` + esc(accent) + `;color:` + esc(on) + `;font-weight:700;font-size:15px;text-decoration:none">` + esc(button) + `</a></p>`)
	if len(code) == 2 {
		h.WriteString(`<p style="margin:0 0 8px;font-size:14px;line-height:1.5">` + esc(code[0]) + `</p>`)
		h.WriteString(`<p style="margin:0 0 20px;font-size:26px;font-weight:700;letter-spacing:4px;font-family:'IBM Plex Mono',Menlo,monospace;color:#0B1220" aria-label="` +
			esc(strings.Join(strings.Split(strings.ReplaceAll(e.code, " ", ""), ""), " ")) + `">` + esc(code[1]) + `</p>`)
	}
	h.WriteString(`<p style="margin:0 0 20px;font-size:13px;line-height:1.5;color:#5B6472">` + esc(after) + `</p>`)
	h.WriteString(`<p style="margin:0;font-size:12px;line-height:1.5;color:#8A93A3">` + esc(foot) + `<br>` + esc(why) + `</p>`)
	h.WriteString(`</td></tr></table></td></tr></table></body></html>`)
	return mailer.Message{Subject: subject, Text: text.String(), HTML: h.String(), Kind: "website_" + e.purpose, Lang: l}
}

// commentEmail is a reply email's facts.
type commentEmail struct {
	// name wrote the reply (the publication's name for a team reply).
	name, post string
	// comment is the member's own comment, reply the reply's text.
	comment, reply string
	link, mute     string
	// more replies wait than the one the email quotes.
	more int
}

// quoteOf is a comment's text for an email: its first n characters.
func quoteOf(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return strings.TrimRight(string(r[:n]), " .,;:") + "…"
}

// commentReplyEmail composes a reply email: "{name} replied to your comment
// on {post}", the member's comment and the reply, Read the reply and Stop
// these emails, in the publisher's colours, on the System stream.
func (s *Service) commentReplyEmail(ctx context.Context, ws *store.Workspace, c *config, set *forms.Settings, l i18n.Lang, e commentEmail) mailer.Message {
	w := WordsIn(l)
	site := siteTitle(c, set)
	host := hostOnly(strings.TrimSuffix(s.sitePath(ctx, ws), "/"))
	subject := w.Text("mail.commentReply.subject", i18n.Args{"name": e.name, "post": e.post})
	yours, theirs := quoteOf(e.comment, 200), quoteOf(e.reply, 1500)
	button, mute := w.Text("mail.commentReply.button", nil), w.Text("mail.commentReply.mute", nil)
	why := w.Text("mail.commentReply.why", i18n.Args{"host": host})
	more := ""
	if e.more > 0 {
		more = w.Text("mail.commentReply.more", i18n.Args{"count": e.more})
	}
	var text strings.Builder
	text.WriteString(subject + "\n\n" + w.Text("mail.commentReply.yours", nil) + "\n" + yours + "\n\n" + w.Text("mail.commentReply.theirs", nil) + "\n" + theirs + "\n\n")
	if more != "" {
		text.WriteString(more + "\n\n")
	}
	text.WriteString(button + ": " + e.link + "\n\n" + why + "\n")
	if e.mute != "" {
		text.WriteString(mute + ": " + e.mute + "\n")
	}
	esc := html.EscapeString
	para := func(s string) string { return strings.ReplaceAll(esc(s), "\n", "<br>") }
	b := set.Branding
	accent, on, bg := b.Palette.Accent, b.Palette.OnAccent, b.Palette.Background
	quote := `margin:0 0 16px;padding:10px 16px;border-left:3px solid #DDD6FE;background:#F5F3FF;border-radius:0 8px 8px 0;font-size:15px;line-height:1.55;`
	var h strings.Builder
	h.WriteString(`<!doctype html><html lang="` + esc(string(l.Locale)) + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + esc(subject) + `</title></head>`)
	h.WriteString(`<body style="margin:0;padding:0;background:` + esc(bg) + `;font-family:'IBM Plex Sans',Arial,sans-serif;color:#1F2937">`)
	h.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + esc(bg) + `"><tr><td align="center" style="padding:32px 16px">`)
	h.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;background:#ffffff;border:1px solid #E3E6EB;border-radius:14px"><tr><td style="padding:28px">`)
	h.WriteString(`<p style="margin:0 0 16px;font-size:13px;font-weight:700;color:#5B6472">` + esc(site) + `</p>`)
	h.WriteString(`<h1 style="margin:0 0 16px;font-size:21px;line-height:1.3;color:#0B1220">` + esc(subject) + `</h1>`)
	h.WriteString(`<p style="margin:0 0 6px;font-size:13px;color:#5B6472">` + esc(w.Text("mail.commentReply.yours", nil)) + `</p>`)
	h.WriteString(`<p style="` + quote + `color:#4B5563">` + para(yours) + `</p>`)
	h.WriteString(`<p style="margin:0 0 6px;font-size:13px;color:#5B6472">` + esc(w.Text("mail.commentReply.theirs", nil)) + `</p>`)
	h.WriteString(`<p style="` + quote + `color:#1F2937">` + para(theirs) + `</p>`)
	if more != "" {
		h.WriteString(`<p style="margin:0 0 16px;font-size:14px;line-height:1.5;color:#4B5563">` + esc(more) + `</p>`)
	}
	h.WriteString(`<p style="margin:0 0 20px"><a href="` + esc(e.link) + `" style="display:inline-block;padding:12px 20px;border-radius:9px;background:` + esc(accent) + `;color:` + esc(on) + `;font-weight:700;font-size:15px;text-decoration:none">` + esc(button) + `</a></p>`)
	h.WriteString(`<p style="margin:0;font-size:12px;line-height:1.6;color:#5B6472">` + esc(why))
	if e.mute != "" {
		h.WriteString(` <a href="` + esc(e.mute) + `" style="color:` + esc(accent) + `;text-decoration:underline">` + esc(mute) + `</a>`)
	}
	h.WriteString(`</p></td></tr></table></td></tr></table></body></html>`)
	return mailer.Message{Subject: subject, Text: text.String(), HTML: h.String(), Kind: "website_comment_reply", Lang: l}
}
