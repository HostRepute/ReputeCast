package membermail

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
)

func every(l i18n.Lang) map[string]mailer.Message {
	at := time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)
	return map[string]mailer.Message{
		"invite":        Invitation(l, InvitationEmail{Inviter: "Linh Nguyen", Workspace: "Acme Newsletter", Role: "Editor", Message: "Welcome <aboard>", Link: "https://x/invite/t", Expires: at}),
		"reset":         PasswordReset(l, "Acme ReputeCast", "aisha@acme.io", "https://x/reset-password/t"),
		"reminder":      SignInReminder(l, "Acme ReputeCast", "aisha@acme.io", "Google Workspace SSO", "https://x/sign-in"),
		"admin reset":   AdminPasswordReset(l, "Acme ReputeCast", "Linh Nguyen", "aisha@acme.io", "https://x/reset-password/t"),
		"email change":  EmailChange(l, "Acme ReputeCast", "aisha@acme.io", "https://x/account?verifyEmail=t"),
		"email changed": EmailChanged(l, "Acme ReputeCast", "aisha@acme.io", "aisha@bello.dev"),
		"2fa reset":     TwoFactorReset(l, "Acme ReputeCast", "Linh Nguyen", "aisha@acme.io", "lost phone", at),
		"notification":  Notification(l, NotificationEmail{Workspace: "Acme Newsletter", Items: []NotificationLine{{Title: "Send auto-paused", Body: "Outlook complaints hit 0.18%.", Link: "/w/acme/live"}}, Base: "https://x", Preferences: "https://x/w/acme/notifications#preferences"}),
		"digest":        Notification(l, NotificationEmail{Workspace: "Acme Newsletter", Items: []NotificationLine{{Title: "A", Body: "a"}, {Title: "B", Body: "b", Link: "/w/acme"}}, Base: "https://x", Preferences: "https://x/p"}),
	}
}

func TestEnglishKeepsTodaysEmails(t *testing.T) {
	m := every(i18n.Default)
	if r := m["reset"]; r.Subject != "Reset your Acme ReputeCast password" ||
		r.Text != "Someone asked to reset the password of aisha@acme.io on Acme ReputeCast.\n\nSet a new password: https://x/reset-password/t\n\nThe link works once and expires in 15 minutes. If it wasn't you, ignore this email; your password stays the same.\n" ||
		r.HTML != `<p style="font:14px/1.5 sans-serif">Someone asked to reset the password of <b>aisha@acme.io</b> on Acme ReputeCast.</p><p style="font:14px sans-serif"><a href="https://x/reset-password/t">Set a new password</a></p><p style="font:13px/1.5 sans-serif;color:#5B6472">The link works once and expires in 15 minutes. If it wasn't you, ignore this email; your password stays the same.</p>` {
		t.Errorf("reset: %+v", r)
	}
	if i := m["invite"]; i.Subject != "Linh Nguyen invited you to Acme Newsletter" ||
		i.Text != "Linh Nguyen invited you to Acme Newsletter on ReputeCast as Editor.\n\nWelcome <aboard>\n\nAccept the invitation: https://x/invite/t\n\nThe link works until Oct 22, 2026.\n" ||
		!strings.Contains(i.HTML, "Welcome &lt;aboard&gt;") {
		t.Errorf("invite: %+v", i)
	}
	if n := m["notification"]; n.Subject != "[Acme Newsletter] Send auto-paused" || !strings.Contains(n.Text, "\nOpen: https://x/w/acme/live\n") ||
		!strings.Contains(n.HTML, `because of your <a href="https://x/w/acme/notifications#preferences">notification preferences</a>.`) {
		t.Errorf("notification: %+v", n)
	}
	if d := m["digest"]; d.Subject != "[Acme Newsletter] 2 notifications" || !strings.HasPrefix(d.Text, "• A: a\n• B: b\n  https://x/w/acme\n") {
		t.Errorf("digest: %+v", d)
	}
}

func TestVietnameseAndPseudo(t *testing.T) {
	vi := every(i18n.For(i18n.Vietnamese))
	if i := vi["invite"]; i.Subject != "Linh Nguyen đã mời bạn vào Acme Newsletter" || !strings.Contains(i.Text, "Liên kết có hiệu lực đến 22 thg 10, 2026.") || i.Lang.Locale != i18n.Vietnamese {
		t.Errorf("invite: %+v", i)
	}
	if d := vi["digest"]; d.Subject != "[Acme Newsletter] 2 thông báo" {
		t.Errorf("digest: %q", d.Subject)
	}
	if r, ok := Reason(i18n.For(i18n.Vietnamese), "invite"); !ok || !strings.HasPrefix(r, "Bạn nhận được email này") {
		t.Errorf("reason: %q", r)
	}
	if f := Footer(i18n.For(i18n.Vietnamese), "Acme ReputeCast", "Acme Newsletter", "reputecast.acme.io"); f != "Gửi từ Acme ReputeCast cho Acme Newsletter · reputecast.acme.io" {
		t.Errorf("footer: %q", f)
	}
	i18n.SetPseudo(true)
	defer i18n.SetPseudo(false)
	for name, m := range every(i18n.Lang{Locale: i18n.Pseudo}) {
		if !strings.HasPrefix(m.Subject, "⟦") && name != "notification" {
			t.Errorf("%s: the pseudo-locale brackets the subject: %q", name, m.Subject)
		}
	}
	for name, m := range vi {
		if strings.Contains(m.Text, "{") || strings.Contains(m.HTML, "{") {
			t.Errorf("%s has an unfilled argument: %+v", name, m)
		}
	}
}

func TestEveryMessageIsTranslated(t *testing.T) {
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("membermail messages without Vietnamese: %v", missing)
	}
}
