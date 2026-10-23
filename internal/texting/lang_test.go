package texting

import (
	"fmt"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/sms"
)

// What ReputeCast writes about texts reads in English exactly as it did
// before it had messages, and in Vietnamese for a Vietnamese reader, the
// numbers in the reader's region format.
func TestMessagesKeepTheirEnglish(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	pause := func(metric string, value, limit float64, held, events, base int64) i18n.Args {
		a := pauseArgs(metric, value, limit)
		a["held"], a["events"], a["base"] = held, events, base
		return a
	}
	for _, c := range []struct {
		got, want string
	}{
		{mNotifyPausedTitle.Text(en, i18n.Args{"name": "Summit: parking"}), `"Summit: parking" auto-paused`},
		{mNotifyPausedSummary.Text(en, pause(MetricFiltered, 2.4, 2, 1204, 3, 125)), "Carriers filtered too many texts: 2.4% (limit 2%). 1,204 held."},
		{mNotifyPausedSummary.Text(en, pause(MetricFailed, 6, 2.5, 0, 31, 500)), "Too many texts failed: 6.0% (limit 2.5%). 0 held."},
		{mNotifyPausedSummary.Text(en, pause(MetricOptOuts, 3.2, 3, 12, 31, 960)), "Too many people opted out: 3.2% (limit 3%). 12 held."},
		{mAuditPausedSummary.Text(en, pause(MetricOptOuts, 3.2, 3, 1204, 31, 960)), "opt_outs 3.2% over limit 3% · 31 of 960 · 1,204 held"},
		{mAuditPausedReason.Text(en, i18n.Args{"metric": MetricFiltered}), "Carriers filtered too many texts"},
		{mNotifyBudgetUsedBody.Text(en, i18n.Args{"month": 10, "budget": en.Region.Money(100, 2)}),
			"October's SMS budget is used up ($100.00). Campaigns can't start and SMS steps wait until someone raises it."},
		{mNotifyBudgetAlertTitle.Text(en, i18n.Args{"level": 0.8}), "SMS budget at 80%"},
		{mNotifyBudgetAlertBody.Text(en, i18n.Args{"spent": en.Region.Money(80.4, 2), "budget": en.Region.Money(100, 2)}), "$80.40 of $100.00 spent on texts this month."},
		{mNotifySilentBody.Text(en, i18n.Args{"provider": "Twilio", "hours": 2}),
			"No receipts from Twilio for 2 hours while texts were sent. Check the status callback in Twilio."},
		{mNotifyRejectedBody.Text(en, i18n.Args{"provider": "Twilio", "what": "Acme reminders", "sender": "+1 888 555 0142", "reason": "Opt-in isn't described"}),
			"Twilio rejected Acme reminders for +1 888 555 0142: Opt-in isn't described."},
		{mCheckQuietSome.Text(en, i18n.Args{"window": "09:00–20:00", "skips": 1, "steps": 3}),
			"Quiet hours: texts go out 09:00–20:00 in each contact’s time zone; outside it, 1 of 3 SMS steps skip the text"},
		{mCheckPending.Text(en, i18n.Args{"sender": "ACME", "countries": countriesIn(en, []string{"IE"}), "count": 20}),
			"ACME isn’t approved yet in Ireland: 20 contacts there are skipped until it is"},
		{mCheckNoRoute.Text(en, i18n.Args{"countries": countriesIn(en, []string{"IE", "DE", "ES"}), "count": 1}),
			"No route covers Ireland, Germany and Spain: 1 contact there is skipped"},
		{mCheckRejected.Text(en, i18n.Args{"sender": "ACME", "countries": countriesIn(en, []string{"DE"}), "count": 1204}),
			"ACME’s registration was rejected in Germany: 1,204 contacts there are skipped"},
		{mBlocked.Text(en, i18n.Args{"count": 1}), "Scheduling stays off until every block is fixed (1 block)."},
		{mBlocked.Text(en, i18n.Args{"count": 3}), "Scheduling stays off until every block is fixed (3 blocks)."},
		{mThreshold.Text(en, i18n.Args{"min": 0.001, "max": 1}), "A threshold is a rate from 0.1% to 100%."},
		{mTypeCount.Text(en, i18n.Args{"count": "1204"}), "Type 1204 to confirm the send."},
		{mNoProviderSender.Text(en, i18n.Args{"provider": "Vonage", "country": countryNameIn(en, "GB")}), "None of Vonage's senders can text United Kingdom yet."},
		{mLinkExpired.Text(en, i18n.Args{"months": 12}), "This link has expired. Short links in texts work for 12 months."},
		{mLabelCopyOf.Text(en, i18n.Args{"name": "Summit: 8 days to go"}), "Copy of Summit: 8 days to go"},

		{mNotifyPausedSummary.Text(vi, pause(MetricFiltered, 2.4, 2, 1204, 3, 125)), "Nhà mạng lọc quá nhiều tin nhắn: 2,4% (giới hạn 2%). Đã tạm giữ 1.204 tin nhắn."},
		{mNotifyBudgetUsedBody.Text(vi, i18n.Args{"month": 10, "budget": vi.Region.Money(100, 2)}),
			"Ngân sách SMS tháng 10 đã dùng hết (100,00\u00a0US$). Chiến dịch không bắt đầu được và các bước SMS phải chờ đến khi có người nâng ngân sách."},
		{mCheckPending.Text(vi, i18n.Args{"sender": "ACME", "countries": countriesIn(vi, []string{"IE", "DE"}), "count": 1204}),
			"ACME chưa được phê duyệt ở Ireland và Đức: 1.204 liên hệ ở đó bị bỏ qua cho đến khi được phê duyệt"},
		{mThreshold.Text(vi, i18n.Args{"min": 0.001, "max": 1}), "Ngưỡng là một tỷ lệ từ 0,1% đến 100%."},
		{mLinkExpired.Text(vi, i18n.Args{"months": 12}), "Liên kết này đã hết hạn. Liên kết rút gọn trong tin nhắn dùng được trong 12 tháng."},
	} {
		if c.got != c.want {
			t.Errorf("got  %q\nwant %q", c.got, c.want)
		}
	}
}

// Text ReputeCast stored in English (a provider's answer, the short-link
// domain's check, a suppression's context and code, an email
// subscription's source, a consent change's page) reads in the reader's
// language; what providers and people wrote stays as it came.
func TestStoredEnglishInVietnamese(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	if got := answerOf(fmt.Errorf("call: %w", sms.ErrUnavailable)); got != "no answer" {
		t.Fatalf("answer: %q", got)
	}
	if got := answerOf(sms.ErrCredentialsRejected); got != "401 Unauthorized · invalid credentials" {
		t.Fatalf("answer: %q", got)
	}
	if got := providerListCode("Twilio"); got != "Twilio opt-out list" {
		t.Fatalf("code: %q", got)
	}
	if got := shortDetail(mDomainOther, i18n.Args{"domain": "go.acme.io", "cname": "x.example.net", "target": "reputecast.acme.io"}); got != "go.acme.io is a CNAME for x.example.net, not reputecast.acme.io." {
		t.Fatalf("detail: %q", got)
	}
	for _, c := range []struct {
		got, want string
	}{
		{answerIn(vi, "no answer"), "không có phản hồi"},
		{answerIn(vi, "401 Unauthorized · invalid credentials"), "401 Unauthorized · thông tin xác thực không hợp lệ"},
		{answerIn(vi, "503 Service Unavailable"), "503 Service Unavailable"},
		{answerIn(en, "no answer"), "no answer"},
		{shortDetailIn(vi, "CNAME points at reputecast.acme.io."), "CNAME trỏ đến reputecast.acme.io."},
		{shortDetailIn(vi, "go.acme.io is a CNAME for x.example.net, not reputecast.acme.io."), "go.acme.io là CNAME đến x.example.net, không phải reputecast.acme.io."},
		{shortDetailIn(vi, "DNS didn't answer for go.acme.io."), "DNS không phản hồi cho go.acme.io."},
		{shortDetailIn(vi, "No CNAME found. DNS can take up to an hour."), "Không tìm thấy CNAME. DNS có thể mất đến một giờ."},
		{shortDetailIn(en, "CNAME points at reputecast.acme.io."), "CNAME points at reputecast.acme.io."},
		{shortDetailIn(vi, "Something a resolver said."), "Something a resolver said."},
		{suppressionCodeIn(vi, "provider_list", "Twilio opt-out list"), "Danh sách hủy nhận của Twilio"},
		{suppressionCodeIn(vi, "provider_list", "STOP"), "STOP"},
		{suppressionCodeIn(vi, "not_in_service", "Twilio 30005"), "Twilio 30005"},
		{contextIn(vi, contextOf("confirmation")), "Tin xác nhận đăng ký"},
		{contextIn(vi, contextOf("test")), "Tin nhắn thử"},
		{contextIn(en, contextOf("confirmation")), "Opt-in confirmation"},
		{contextIn(vi, ""), ""},
		{emailSourceLabel(en, "preference_center"), "preference center"},
		{emailSourceLabel(en, "one_click"), "List-Unsubscribe one-click"},
		{emailSourceLabel(en, "manual"), "added by the team"},
		{emailSourceLabel(en, "api"), "the API"},
		{emailSourceLabel(vi, "one_click"), "List-Unsubscribe một lần nhấp"},
		{emailSourceLabel(vi, "repermission"), "email xin phép lại"},
		{emailSourceLabel(vi, "team"), "do nhóm thêm vào"},
		{emailSourceLabel(en, "import:brevo"), segments.SourceLabel("import:brevo")},
		{emailSourceLabel(vi, "import:brevo"), segments.SourceLabelIn(vi, "import:brevo")},
		{evidenceIn(vi, evPreferenceCenter.Source()), "Trung tâm tùy chọn"},
		{evidenceIn(vi, "Account page"), "Trang tài khoản"},
		{countryNameIn(vi, "DE"), "Đức"},
		{countryNameIn(en, "DE"), "Germany"},
	} {
		if c.got != c.want {
			t.Errorf("got  %q\nwant %q", c.got, c.want)
		}
	}
}
