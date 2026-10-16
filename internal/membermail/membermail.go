// Package membermail writes the emails ReputeCast sends to team members:
// invitations, password resets, email changes, two-factor resets and
// notifications, in the member's app language. internal/systemmail wraps
// them in the branded layout and sends them through the System stream.
package membermail

import (
	"embed"
	"html"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
)

// Emails to members (invitations, password resets, email changes,
// two-factor resets, notifications) are written in the member's app
// language: their own choice, the instance default when they haven't
// chosen, or the invitation's language for someone without an account
// (internal/applang decides). The texts are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "membermail")

var (
	mReasonDefault   = msgs.M("reason.default")
	mFooterSentBy    = msgs.M("footer.sentBy")
	mFooterSentHost  = msgs.M("footer.sentByHost")
	mFooterSentFor   = msgs.M("footer.sentFor")
	mFooterSentForHo = msgs.M("footer.sentForHost")
	mLinkLine        = msgs.M("linkLine")

	mInviteSubject = msgs.M("invite.subject")
	mInviteLead    = msgs.M("invite.lead")
	mInviteAction  = msgs.M("invite.action")
	mInviteExpires = msgs.M("invite.expires")

	mResetSubject = msgs.M("reset.subject")
	mResetLead    = msgs.M("reset.lead")
	mResetAction  = msgs.M("reset.action")
	mResetNote    = msgs.M("reset.note")

	mResetSSOSubject = msgs.M("resetSso.subject")
	mResetSSOLead    = msgs.M("resetSso.lead")
	mResetSSOAction  = msgs.M("resetSso.action")
	mResetSSONote    = msgs.M("resetSso.note")

	mAdminResetSubject = msgs.M("adminReset.subject")
	mAdminResetLead    = msgs.M("adminReset.lead")
	mAdminResetAction  = msgs.M("adminReset.action")
	mAdminResetNote    = msgs.M("adminReset.note")

	mEmailChangeSubject = msgs.M("emailChange.subject")
	mEmailChangeLead    = msgs.M("emailChange.lead")
	mEmailChangeAction  = msgs.M("emailChange.action")
	mEmailChangeNote    = msgs.M("emailChange.note")

	mEmailChangedSubject = msgs.M("emailChanged.subject")
	mEmailChangedLead    = msgs.M("emailChanged.lead")

	mTwofaSubject = msgs.M("twofaReset.subject")
	mTwofaLead    = msgs.M("twofaReset.lead")
	mTwofaReason  = msgs.M("twofaReset.reason")
	mTwofaNote    = msgs.M("twofaReset.note")

	mNotifySubject       = msgs.M("notify.subject")
	mNotifyDigestSubject = msgs.M("notify.digestSubject")
	mNotifyOpen          = msgs.M("notify.open")
	mNotifyOpenShort     = msgs.M("notify.openShort")
	mNotifyOpenText      = msgs.M("notify.openText")
	mNotifyWhy           = msgs.M("notify.why")
	mNotifyWhyText       = msgs.M("notify.whyText")
)

// Reason says why a message of a member kind was sent, in the reader's
// language. ok is false for kinds it doesn't know (subscribers' mail).
func Reason(l i18n.Lang, kind string) (reason string, ok bool) {
	if kind == "notification" {
		// Notifications already end with the preferences link.
		return "", true
	}
	if m, err := msgs.Lookup("reason." + kind); err == nil {
		return m.Text(l, nil), true
	}
	return "", false
}

// DefaultReason is "This is an automated message from Acme ReputeCast."
func DefaultReason(l i18n.Lang, brand string) string {
	return mReasonDefault.Text(l, i18n.Args{"brand": brand})
}

// A block of a member email.
type block struct {
	kind  string // lead, plain, link, note
	msg   i18n.Msg
	args  i18n.Args
	text  string // plain text instead of a message (the inviter's own words)
	label i18n.Msg
	url   string
}

// compose writes a member email's text and HTML from the same messages.
func compose(l i18n.Lang, kind string, subject string, blocks ...block) mailer.Message {
	var text []string
	var body strings.Builder
	for _, b := range blocks {
		switch b.kind {
		case "link":
			label := b.label.Text(l, nil)
			text = append(text, mLinkLine.Text(l, i18n.Args{"label": label, "url": b.url}))
			body.WriteString(`<p style="font:14px sans-serif"><a href="` + html.EscapeString(b.url) + `">` + b.label.HTML(l, nil, nil) + `</a></p>`)
		default:
			var t, h string
			if b.text != "" {
				t, h = b.text, html.EscapeString(b.text)
			} else {
				t, h = b.msg.Text(l, b.args), b.msg.HTML(l, b.args, nil)
			}
			text = append(text, t)
			style := "font:14px/1.5 sans-serif"
			if b.kind == "note" {
				style = "font:13px/1.5 sans-serif;color:#5B6472"
			}
			body.WriteString(`<p style="` + style + `">` + h + `</p>`)
		}
	}
	return mailer.Message{Kind: kind, Subject: subject, Text: strings.Join(text, "\n\n") + "\n", HTML: body.String(), Lang: l}
}

// InvitationEmail is an invitation to join a workspace, in the
// invitation's language.
type InvitationEmail struct {
	Inviter, Workspace, Role string
	// Message is the inviter's own words, kept as written.
	Message string
	Link    string
	Expires time.Time
}

// Invitation writes an invitation email.
func Invitation(l i18n.Lang, d InvitationEmail) mailer.Message {
	args := i18n.Args{"inviter": d.Inviter, "workspace": d.Workspace, "role": d.Role}
	blocks := []block{{kind: "lead", msg: mInviteLead, args: args}}
	if strings.TrimSpace(d.Message) != "" {
		blocks = append(blocks, block{kind: "plain", text: strings.TrimSpace(d.Message)})
	}
	blocks = append(blocks,
		block{kind: "link", label: mInviteAction, url: d.Link},
		block{kind: "note", msg: mInviteExpires, args: i18n.Args{"date": l.Region.Date(d.Expires)}})
	return compose(l, "invite", mInviteSubject.Text(l, args), blocks...)
}

// PasswordReset is the link someone asked for to reset their password.
func PasswordReset(l i18n.Lang, instance, email, link string) mailer.Message {
	args := i18n.Args{"instance": instance, "email": email}
	return compose(l, "password_reset", mResetSubject.Text(l, args),
		block{kind: "lead", msg: mResetLead, args: args},
		block{kind: "link", label: mResetAction, url: link},
		block{kind: "note", msg: mResetNote})
}

// SignInReminder tells someone without a password how they sign in.
func SignInReminder(l i18n.Lang, instance, email, provider, link string) mailer.Message {
	args := i18n.Args{"instance": instance, "email": email, "provider": provider}
	return compose(l, "password_reset_sso", mResetSSOSubject.Text(l, args),
		block{kind: "lead", msg: mResetSSOLead, args: args},
		block{kind: "link", label: mResetSSOAction, url: link},
		block{kind: "note", msg: mResetSSONote})
}

// AdminPasswordReset is a reset link an instance admin sent.
func AdminPasswordReset(l i18n.Lang, instance, admin, email, link string) mailer.Message {
	args := i18n.Args{"instance": instance, "admin": admin, "email": email}
	return compose(l, "password_reset", mAdminResetSubject.Text(l, args),
		block{kind: "lead", msg: mAdminResetLead, args: args},
		block{kind: "link", label: mAdminResetAction, url: link},
		block{kind: "note", msg: mAdminResetNote})
}

// EmailChange asks to confirm a new sign-in address.
func EmailChange(l i18n.Lang, instance, oldEmail, link string) mailer.Message {
	args := i18n.Args{"instance": instance, "email": oldEmail}
	return compose(l, "email_change", mEmailChangeSubject.Text(l, args),
		block{kind: "lead", msg: mEmailChangeLead, args: args},
		block{kind: "link", label: mEmailChangeAction, url: link},
		block{kind: "note", msg: mEmailChangeNote})
}

// EmailChanged tells the previous address that the sign-in email changed.
func EmailChanged(l i18n.Lang, instance, oldEmail, newEmail string) mailer.Message {
	args := i18n.Args{"instance": instance, "old": oldEmail, "new": newEmail}
	m := compose(l, "email_changed", mEmailChangedSubject.Text(l, args), block{kind: "lead", msg: mEmailChangedLead, args: args})
	// A short notice: plain text, converted to HTML by the layout.
	m.HTML = ""
	return m
}

// TwoFactorReset tells a member an instance admin reset their two-factor.
func TwoFactorReset(l i18n.Lang, instance, admin, email, reason string, before time.Time) mailer.Message {
	args := i18n.Args{"instance": instance, "admin": admin, "email": email}
	return compose(l, "twofa_reset", mTwofaSubject.Text(l, args),
		block{kind: "lead", msg: mTwofaLead, args: args},
		block{kind: "plain", msg: mTwofaReason, args: i18n.Args{"reason": reason}},
		block{kind: "plain", msg: mTwofaNote, args: i18n.Args{"date": l.Region.Date(before)}})
}

// NotificationLine is one notification in an email.
type NotificationLine struct {
	Title, Body string
	// Link is an app path, or empty.
	Link string
}

// NotificationEmail is one notification, or a digest of several from one workspace.
type NotificationEmail struct {
	Workspace string
	Items     []NotificationLine
	// Base is the instance's public URL; Preferences the notification preferences page.
	Base, Preferences string
}

// Notification writes a notification email: one notification in full, or a digest.
func Notification(l i18n.Lang, d NotificationEmail) mailer.Message {
	var subject string
	if len(d.Items) == 1 {
		subject = d.Items[0].Title
	} else {
		subject = mNotifyDigestSubject.Text(l, i18n.Args{"count": len(d.Items)})
	}
	if d.Workspace != "" {
		subject = mNotifySubject.Text(l, i18n.Args{"workspace": d.Workspace, "title": subject})
	}
	var text, body strings.Builder
	if len(d.Items) == 1 {
		n := d.Items[0]
		text.WriteString(n.Title + "\n\n" + n.Body + "\n")
		body.WriteString(`<p style="font:600 16px sans-serif;margin:0 0 8px">` + html.EscapeString(n.Title) + `</p><p style="font:14px/1.5 sans-serif;margin:0 0 16px">` + html.EscapeString(n.Body) + `</p>`)
		if n.Link != "" {
			text.WriteString("\n" + mNotifyOpenText.Text(l, i18n.Args{"url": d.Base + n.Link}) + "\n")
			body.WriteString(`<p style="font:14px sans-serif"><a href="` + html.EscapeString(d.Base+n.Link) + `">` + mNotifyOpen.HTML(l, nil, nil) + `</a></p>`)
		}
	} else {
		for _, n := range d.Items {
			text.WriteString("• " + n.Title + ": " + n.Body + "\n")
			body.WriteString(`<p style="font:14px/1.5 sans-serif;margin:0 0 12px"><b>` + html.EscapeString(n.Title) + `</b><br>` + html.EscapeString(n.Body))
			if n.Link != "" {
				text.WriteString("  " + d.Base + n.Link + "\n")
				body.WriteString(`<br><a href="` + html.EscapeString(d.Base+n.Link) + `">` + mNotifyOpenShort.HTML(l, nil, nil) + `</a>`)
			}
			body.WriteString("</p>")
		}
	}
	text.WriteString("\n—\n" + mNotifyWhyText.Text(l, i18n.Args{"url": d.Preferences}) + "\n")
	body.WriteString(`<p style="font:12px sans-serif;color:#5B6472;margin-top:24px">` + mNotifyWhy.HTML(l, nil, i18n.Tags{
		"link": func(inner string) string {
			return `<a href="` + html.EscapeString(d.Preferences) + `">` + inner + `</a>`
		},
	}) + `</p>`)
	return mailer.Message{Kind: "notification", Subject: subject, Text: text.String(), HTML: body.String(), Lang: l}
}

// Footer is "Sent by Acme ReputeCast for Acme Newsletter · reputecast.acme.io".
func Footer(l i18n.Lang, instance, workspace, host string) string {
	args := i18n.Args{"instance": instance, "workspace": workspace, "host": host}
	switch {
	case workspace == "" && host == "":
		return mFooterSentBy.Text(l, args)
	case workspace == "":
		return mFooterSentHost.Text(l, args)
	case host == "":
		return mFooterSentFor.Text(l, args)
	}
	return mFooterSentForHo.Text(l, args)
}
