package texting

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// An SMS campaign's schedule on its approval keeps its message, so each
// approver reads it in their own language and region format, also after
// the snapshot is stored; the English label is the one requests compare.
func TestApprovalScheduleInTheApproversLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 30, 13, 0, 0, 0, time.UTC)
	minute := int32(9 * 60)
	day := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		c             store.SmsCampaignRow
		english, want string
	}{
		{store.SmsCampaignRow{ScheduleMode: "now"}, "Now, holding whoever is outside their window", "Ngay bây giờ, giữ lại những người đang ngoài khung giờ gửi"},
		{store.SmsCampaignRow{ScheduleMode: "at", SendAt: &at}, "30 Oct 2026 09:00 for everyone",
			vi.Region.Date(at.In(ny)) + " " + vi.Region.Time(at.In(ny), "") + " cho mọi người"},
		{store.SmsCampaignRow{ScheduleMode: "local", LocalDate: &day, LocalMinute: &minute}, "30 Oct 2026 at 09:00 in each recipient's time zone",
			vi.Region.Date(day.Add(9*time.Hour)) + " lúc " + vi.Region.Time(day.Add(9*time.Hour), "") + " theo múi giờ của từng người nhận"},
		{store.SmsCampaignRow{}, "Not scheduled", "Chưa lên lịch"},
	} {
		if got := scheduleLabel(tc.c, ny); got != tc.english {
			t.Errorf("%s: English label %q", tc.c.ScheduleMode, got)
		}
		b, err := json.Marshal(scheduleText(tc.c, ny))
		if err != nil {
			t.Fatal(err)
		}
		var kept i18n.Text
		if err := json.Unmarshal(b, &kept); err != nil {
			t.Fatal(err)
		}
		if got := kept.In(vi); got != tc.want {
			t.Errorf("%s: Vietnamese %q, want %q", tc.c.ScheduleMode, got, tc.want)
		}
	}
}

// A Send SMS step's issue reads in the member's language on the canvas; its
// English stays as it was.
func TestStepIssueInTheMembersLanguage(t *testing.T) {
	n := automations.AutomationNode{Data: automations.AutomationNodeData{SMS: &automations.AutomationSMSStep{Body: "Watch it live: https://bit.ly/acme-live"}}}
	issues := SmsStep{}.Validate(n)
	if len(issues) != 1 || issues[0].Message != "bit.ly is a public link shortener: carriers filter texts with them. Paste the full address; it's shortened on your own domain." {
		t.Fatalf("English: %+v", issues)
	}
	if got := issues[0].Text.In(i18n.For(i18n.Vietnamese)); got != "bit.ly là dịch vụ rút gọn liên kết công khai: nhà mạng lọc tin nhắn có những liên kết này. Hãy dán địa chỉ đầy đủ; liên kết sẽ được rút gọn trên tên miền của bạn." {
		t.Errorf("Vietnamese: %q", got)
	}
}
