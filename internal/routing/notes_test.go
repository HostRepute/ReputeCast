package routing_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/routing"
)

// Route notes and delivery errors are stored in English as they always
// were, and read in a reader's language; anything else reads as stored.
func TestNotesAndErrorsIn(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	notes := []struct {
		key   string
		args  i18n.Args
		en, v string
	}{
		{"campaignRoute", i18n.Args{"provider": "SES"}, "Campaign route: SES only", "Tuyến của chiến dịch: chỉ SES"},
		{"byRule", i18n.Args{"provider": "Postmark", "rule": "Outlook: warm IPs"}, "Postmark by rule: Outlook: warm IPs", "Postmark theo quy tắc: Outlook: warm IPs"},
		{"split", i18n.Args{"provider": "SES", "split": "90/10"}, "SES share of the 90/10 split", "Phần của SES trong phân bổ 90/10"},
		{"splitWarmup", i18n.Args{"provider": "Postal", "split": "90/10", "day": 9}, "Postal share of the 90/10 split, warm-up day 9",
			"Phần của Postal trong phân bổ 90/10, khởi động ngày 9"},
		{"instead", i18n.Args{"alt": "SES", "provider": "Postal", "why": "cap", "tier": "active", "day": 9}, "SES instead of Postal: over today's warm-up cap",
			"SES thay cho Postal: vượt giới hạn khởi động hôm nay"},
		{"instead", i18n.Args{"alt": "SES", "provider": "Postal", "why": "tier", "tier": "at_risk", "day": 9}, "SES instead of Postal: not sending to at risk contacts on warm-up day 9",
			"SES thay cho Postal: không gửi cho liên hệ nhóm Có nguy cơ vào ngày khởi động 9"},
		{"instead", i18n.Args{"alt": "SES", "provider": "Postal", "why": "paused", "tier": "active", "day": 9}, "SES instead of Postal: warm-up paused",
			"SES thay cho Postal: khởi động đang tạm dừng"},
		{"failover", i18n.Args{"provider": "Postal", "fallback": "SES"}, "Postal was down; Broadcasts failed over to SES",
			"Postal gián đoạn; luồng Gửi hàng loạt đã chuyển dự phòng sang SES"},
	}
	for _, n := range notes {
		en := routing.RouteNote(n.key, n.args)
		if en != n.en {
			t.Errorf("%s: %q, want %q", n.key, en, n.en)
		}
		if got := routing.NoteIn(vi, en); got != n.v {
			t.Errorf("%s in Vietnamese: %q, want %q", n.key, got, n.v)
		}
		if got := routing.NoteIn(i18n.Default, en); got != en {
			t.Errorf("%s in English: %q", n.key, got)
		}
	}
	errs := []struct {
		key   string
		args  i18n.Args
		en, v string
	}{
		{"down", i18n.Args{"provider": "Postmark"}, "Postmark is down", "Postmark đang gián đoạn"},
		{"downNoOther", i18n.Args{"provider": "Postmark"}, "Postmark was down and no other provider in the route could take the message.",
			"Postmark gián đoạn và không nhà cung cấp nào khác trong tuyến nhận được thư này."},
		{"providersRemoved", nil, "The route's providers were removed.", "Các nhà cung cấp của tuyến đã bị gỡ."},
		{"contentGone", nil, "The message's content is no longer available, so it can't be sent.", "Nội dung của thư không còn nữa, nên không thể gửi."},
		{"unreadable", i18n.Args{"error": "cipher: message authentication failed"}, "The message couldn't be opened: cipher: message authentication failed",
			"Không mở được thư: cipher: message authentication failed"},
		{"allFailed", i18n.Args{"error": "550 5.7.1 rejected"}, "Every provider in the route failed: 550 5.7.1 rejected", "Mọi nhà cung cấp trong tuyến đều thất bại: 550 5.7.1 rejected"},
	}
	for _, e := range errs {
		en := routing.DeliveryError(e.key, e.args)
		if en != e.en {
			t.Errorf("%s: %q, want %q", e.key, en, e.en)
		}
		if got := routing.DeliveryErrorIn(vi, en); got != e.v {
			t.Errorf("%s in Vietnamese: %q, want %q", e.key, got, e.v)
		}
	}
	if got := routing.DeliveryErrorIn(vi, "421 4.7.0 try again later"); got != "421 4.7.0 try again later" {
		t.Errorf("a provider's own words: %q", got)
	}
	if got := routing.NoteIn(vi, "something else"); got != "something else" {
		t.Errorf("an unknown note: %q", got)
	}
}
