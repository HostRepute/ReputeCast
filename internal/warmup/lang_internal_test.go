package warmup

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The log's English reads exactly as it did before its messages were kept,
// and the same message reads in Vietnamese with the region's numbers.
func TestLogMessages(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	cases := []struct {
		key     string
		args    i18n.Args
		english string
		vietnam string
	}{
		{"log.slowdown", i18n.Args{"metric": "bounce", "rate": 2.3 / 100, "guard": 2.0 / 100, "guardDigits": digits(2), "mailbox": mailboxArg("yahoo", "Yahoo"), "action": "hold", "day": 7, "cap": int64(4000)},
			"Bounces hit 2.3% (guard 2%), mostly Yahoo. Day 7 cap held at 4,000.",
			"Tỷ lệ trả thư đạt 2,3% (ngưỡng bảo vệ 2%), chủ yếu ở Yahoo. Ngày 7 giữ giới hạn ở mức 4.000."},
		{"log.slowdown", i18n.Args{"metric": "complaint", "rate": 0.12 / 100, "guard": 0.08 / 100, "guardDigits": digits(0.08), "mailbox": mailboxArg("", ""), "action": "pause"},
			"Complaints hit 0.12% (guard 0.08%). The plan is paused until someone resumes it.",
			"Tỷ lệ khiếu nại đạt 0,12% (ngưỡng bảo vệ 0,08%). Kế hoạch tạm dừng cho đến khi có người tiếp tục."},
		{"log.slowdown", i18n.Args{"metric": "bounce", "rate": 3.1 / 100, "guard": 2.5 / 100, "guardDigits": digits(2.5), "mailbox": mailboxArg("other", "Other"), "action": "step_back", "day": 5, "cap": int64(1500)},
			"Bounces hit 3.1% (guard 2.5%), mostly Other. Day 5 steps back to 1,500.",
			"Tỷ lệ trả thư đạt 3,1% (ngưỡng bảo vệ 2,5%), chủ yếu ở các nhà cung cấp hộp thư khác. Ngày 5 lùi về mức 1.500."},
		{"log.dayComplete", i18n.Args{"day": 3, "sent": int64(1200), "bounce": 0.4 / 100, "inside": "yes", "campaign": "Weekly Digest #141"},
			`Day 3 complete: 1,200 sent inside "Weekly Digest #141", bounce 0.4%.`,
			`Ngày 3 xong: đã gửi 1.200 trong "Weekly Digest #141", trả thư 0,4%.`},
		{"log.shareRaised", i18n.Args{"share": 15.0 / 100, "day": 6},
			"Broadcast share raised to 15% on day 6.", "Tỷ trọng Gửi hàng loạt tăng lên 15% vào ngày 6."},
		{"log.started", i18n.Args{"named": "yes", "name": "Linh Nguyen", "days": 21, "order": "engaged"},
			"Plan started by Linh Nguyen: 21 counted days, engaged-first.", "Linh Nguyen đã bắt đầu kế hoạch: 21 ngày được tính, liên hệ tương tác nhiều nhất trước."},
		{"log.settings", i18n.Args{"named": "no", "name": ""}, "Plan settings changed by someone.", "Ai đó đã đổi cài đặt kế hoạch."},
	}
	for _, c := range cases {
		k := keep(c.key, c.args)
		if k.English != c.english {
			t.Errorf("%s English: %q, want %q", c.key, k.English, c.english)
		}
		if got := k.Text.In(vi); got != c.vietnam {
			t.Errorf("%s Vietnamese: %q, want %q", c.key, got, c.vietnam)
		}
	}
	if digits(2) != "0" || digits(0.5) != "1" || digits(0.08) != "2" || digits(0.1) != "1" {
		t.Fatalf("digits: %s %s %s %s", digits(2), digits(0.5), digits(0.08), digits(0.1))
	}
}

// A stored text reads in its English for English readers and rows without
// a message, and in the reader's language otherwise.
func TestTextIn(t *testing.T) {
	k := keep("status.capHeld", i18n.Args{"day": 7})
	texts := textsJSON(map[string]i18n.Text{"status_detail": k.Text})
	if got := textIn(i18n.Default, texts, "status_detail", k.English); got != "Day 7 cap held" {
		t.Fatalf("English: %q", got)
	}
	if got := textIn(i18n.For(i18n.Vietnamese), texts, "status_detail", k.English); got != "Giữ nguyên giới hạn ngày 7" {
		t.Fatalf("Vietnamese: %q", got)
	}
	if got := textIn(i18n.For(i18n.Vietnamese), []byte(`{}`), "status_detail", "Day 6 cap held"); got != "Day 6 cap held" {
		t.Fatalf("a row from before: %q", got)
	}
	if string(textsJSON(map[string]i18n.Text{"status_detail": {}})) != "{}" {
		t.Fatal("an empty message is left out")
	}
}
