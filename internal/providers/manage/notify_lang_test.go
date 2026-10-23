package manage

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The notifications read as they always did in English, and in each
// member's own language from the text notify keeps.
func TestNotificationsReadInEachMembersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	since := time.Date(2026, 10, 12, 14, 2, 0, 0, time.UTC)
	for _, c := range []struct {
		args   i18n.Args
		en, vi string
	}{
		{i18n.Args{"hasError": "yes", "error": "503 ServiceUnavailable", "hasOp": "yes", "op": "SendEmail", "since": since,
			"auto": "Automations and System", "autoCount": 2, "held": "Broadcasts", "heldCount": 1},
			"503 ServiceUnavailable on SendEmail since 14:02. Automations and System failed over automatically. Broadcasts wait for you.",
			"503 ServiceUnavailable ở SendEmail từ 14:02. 2 luồng đã tự động chuyển dự phòng. 1 luồng đang chờ bạn quyết định."},
		{i18n.Args{"hasError": "no", "error": "", "hasOp": "no", "op": "", "since": since, "auto": "", "autoCount": 0, "held": "", "heldCount": 0},
			"API errors since 14:02.", "Lỗi API từ 14:02."},
	} {
		en, text := keep(mNotifyDownBody, c.args)
		if en != c.en || text.In(vi) != c.vi {
			t.Errorf("down: %q / %q", en, text.In(vi))
		}
	}
	title, _ := keep(mNotifyHeldTitle, i18n.Args{"campaign": `"Weekly Digest"`})
	body, text := keep(mNotifyHeldBody, i18n.Args{"name": "Mailgun", "count": int64(12000), "countText": "12000"})
	if title != `"Weekly Digest" is waiting for a decision` || body != "Mailgun is down. Its share (12000 recipients) is held; the rest keeps sending." ||
		text.In(vi) != "Mailgun đang gián đoạn. Phần của nó (12.000 người nhận) đang bị tạm giữ; phần còn lại vẫn tiếp tục gửi." {
		t.Errorf("held: %q / %q / %q", title, body, text.In(vi))
	}
	body, text = keep(mNotifySilentBody, i18n.Args{"since": since})
	if body != "No delivery, bounce or complaint events since 14:02. Bounce protection for this route is blind; its share of Broadcasts is held." ||
		text.In(vi) != "Không có sự kiện gửi thành công, thư trả lại hay khiếu nại nào từ 14:02. Tuyến này đang không được bảo vệ khỏi thư trả lại; phần Gửi hàng loạt của nó đang bị tạm giữ." {
		t.Errorf("silent: %q / %q", body, text.In(vi))
	}
}
