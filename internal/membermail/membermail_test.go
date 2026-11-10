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
		"summary":       ReportSummary(l, sampleSummary("weekly")),
		"monthly":       ReportSummary(l, sampleSummary("monthly")),
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

func sampleSummary(freq string) ReportSummaryEmail {
	from := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	return ReportSummaryEmail{Workspace: "Acme <News>", Frequency: freq, From: from, To: from.AddDate(0, 0, 7),
		Campaigns: SummaryCampaigns{Started: 2, Collecting: 1, Sent: 182340, Delivered: 180912, HumanOpens: 61204, Clicks: 9876, Unsubscribes: 214, Bounces: 1428,
			Complaints: 31, ComplaintRate: 0.03, Reported: true},
		Top:      []SummaryCampaign{{Name: "Weekly Digest #142 & more", Sent: 48210, HumanOpens: 16402, Clicks: 2511, Link: "https://x/w/acme/campaigns/1/report"}},
		Audience: &SummaryAudience{Joined: 1240, Departed: 56, Subscribers: 48213}, Health: &SummaryHealth{Score: 92, Label: "Good"},
		Links: SummaryLinks{Campaigns: "https://x/w/acme/campaigns", Deliverability: "https://x/w/acme/deliverability", Preferences: "https://x/w/acme/notifications#summary"}}
}

// TestReportSummary: the summary is an accessible email (tables with
// headers, links named by their target) whose plain-text part carries the
// same numbers, with every value escaped.
func TestReportSummary(t *testing.T) {
	for _, l := range []i18n.Lang{i18n.Default, i18n.For(i18n.Vietnamese), i18n.For(i18n.German)} {
		m := ReportSummary(l, sampleSummary("weekly"))
		if m.Kind != "report_summary" || m.Lang.Locale != l.Locale || !strings.Contains(m.Subject, "Acme <News>") {
			t.Errorf("%s: %+v", l.Locale, m)
		}
		for _, n := range []int64{182340, 61204, 9876, 1428, 48210, 16402, 1240, 56, 48213, 92} {
			if v := l.Region.Int(n); !strings.Contains(m.Text, v) || !strings.Contains(m.HTML, v) {
				t.Errorf("%s: %s is in one part only", l.Locale, v)
			}
		}
		for _, want := range []string{`scope="col"`, `scope="row"`, `text-align:right`, `href="https://x/w/acme/campaigns/1/report"`, `href="https://x/w/acme/notifications#summary"`} {
			if !strings.Contains(m.HTML, want) {
				t.Errorf("%s: the HTML has no %s", l.Locale, want)
			}
		}
		if strings.Contains(m.HTML, "Acme <News>") || strings.Contains(m.HTML, "#142 & more") || !strings.Contains(m.HTML, "Acme &lt;News&gt;") {
			t.Errorf("%s: values aren't escaped", l.Locale)
		}
		// The report link's accessible name starts with what the reader sees.
		if !strings.Contains(m.HTML, `aria-label="`) || !strings.Contains(m.Text, "https://x/w/acme/campaigns/1/report") {
			t.Errorf("%s: report link: %s", l.Locale, m.Text)
		}
		if r, ok := Reason(l, m.Kind); !ok || r != "" {
			t.Errorf("%s: the layout adds a reason the summary already gives: %q", l.Locale, r)
		}
	}
	// Sections the member's role doesn't show are left out.
	d := sampleSummary("monthly")
	d.Audience, d.Health = nil, nil
	if full, part := ReportSummary(i18n.Default, sampleSummary("monthly")), ReportSummary(i18n.Default, d); strings.Count(full.HTML, "<h2") != strings.Count(part.HTML, "<h2")+2 ||
		strings.Contains(part.Text, "48,213") {
		t.Errorf("sections without the audience and health:\n%s", part.Text)
	}
}
