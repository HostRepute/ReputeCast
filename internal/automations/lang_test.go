package automations

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

var vi = i18n.For(i18n.Vietnamese)

// The canvas's names, the diff, the problems and the alerts read in
// Vietnamese with the words the web uses (web/src/features/automations).
func TestTextsInVietnamese(t *testing.T) {
	g, refs := welcomeV4(), welcomeRefs()
	titles := map[string]string{
		"n2":  "Chờ 10 phút",
		"n4":  "Đã mở step 1 trong vòng 2 ngày?",
		"n7":  "50 / 50 ở step 3",
		"n10": "Đã nhấp một liên kết docs",
		"n13": "Rời quy trình · vẫn ở trong danh sách",
	}
	for id, want := range titles {
		if got := say(vi, nodeTitle(g, *g.Node(id), refs)); got != want {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
	}
	if got := say(vi, triggerLabel(g.Node("n1").Data.Trigger, refs)); got != "Tham gia danh sách · Weekly Digest" {
		t.Errorf("trigger label %q", got)
	}

	items := Diff(welcomeV3(), welcomeV4(), DiffContext{
		FromRefs: refs, ToVersion: 4, Lang: vi,
		Failures: map[string]WebhookFailure{"w1": {URL: "https://hooks.slack.com/services/T000/B000/XXXX", Status: 404, Since: sep29, Failures: 1284}},
		Check:    func(string, int) string { return "" },
	})
	want := []struct{ title, detail string }{
		{"Chia A/B sau “Step 2”, 50 / 50", "2 bước email mới: 3A “Set up your first flag” và 3B “A 5-minute walkthrough”. Cả hai email đều đạt kiểm tra nội dung của Độ sẵn sàng gửi."},
		{"Trì hoãn trước “Step 2”: 1 ngày → 2 ngày", "Những liên hệ đang chờ giữ nguyên bộ đếm giờ ban đầu, trừ khi bạn chuyển họ sang v4 bên dưới."},
		{"Webhook “notify-slack” sau trình kích hoạt", "Endpoint https://hooks.slack.com/… trả về 404 từ 29 thg 9. 1.284 lệnh gọi thất bại có trong nhật ký webhook."},
		{"Trình kích hoạt, step 1, rẽ nhánh, nhánh “Không”, mục tiêu và thoát", ""},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items: %+v", len(items), items)
	}
	for i, w := range want {
		if items[i].Title != w.title || items[i].Detail != w.detail {
			t.Errorf("item %d:\n got %q | %q\nwant %q | %q", i, items[i].Title, items[i].Detail, w.title, w.detail)
		}
	}

	broken := welcomeV4()
	broken.Edges = slices.DeleteFunc(broken.Edges, func(e AutomationEdge) bool { return e.Source == "n4" && e.Data.Path == PathNo })
	var problems []string
	for _, is := range validate(vi, broken, refs) {
		problems = append(problems, is.Message)
	}
	if !slices.Contains(problems, "Rẽ nhánh “Đã mở step 1 trong vòng 2 ngày?” chưa có nhánh “Không”. Hãy nối nhánh này với một bước hoặc một lối thoát.") {
		t.Errorf("problems %q", problems)
	}

	ny, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 10, 12, 11, 0, 0, 0, ny)
	down := time.Date(2026, 10, 12, 9, 2, 0, 0, ny)
	route := AutomationRoute{
		Primary:  []AutomationRouteProvider{{Short: "SES", Status: "down", DownSince: &down, DownReason: "outage"}},
		Fallback: &AutomationRouteProvider{Short: "Postmark", Status: "down", DownSince: &down, DownReason: "auth", LastErrorStatus: 401},
	}
	if got := say(vi, heldDetailText(route, 23, ny, now)); got != "SES ngừng hoạt động từ 09:02 và Postmark (dự phòng) trả về 401: khóa API hoặc token của nhà cung cấp bị từ chối. 23 email đang chờ và sẽ được gửi ngay khi có nhà cung cấp ổn định; thời gian trì hoãn trong quy trình vẫn tiếp tục tính." {
		t.Errorf("held detail %q", got)
	}
	if got := say(vi, finishEstimateText(welcomeV3(), nil)); got != "ngay lập tức" {
		t.Errorf("estimate %q", got)
	}
}

// A version's summary is stored in English with its text, and reads in each
// reader's language; one stored before (no text) reads in English.
func TestVersionSummaryKeepsItsText(t *testing.T) {
	items := Diff(welcomeV3(), welcomeV4(), DiffContext{FromRefs: welcomeRefs(), ToVersion: 4})
	summary := versionSummary(items, false)
	english := say(i18n.Default, summary)
	if !strings.HasPrefix(english, `added A/B split after "Step 2", 50 / 50; delay before "Step 2": 1 day → 2 days; `) {
		t.Fatalf("English summary %q", english)
	}
	kept := keptText(keepText(summary))
	if got := keptSummary(vi, english, kept); !strings.HasPrefix(got, "thêm chia A/B sau “Step 2”, 50 / 50; trì hoãn trước “Step 2”: 1 ngày → 2 ngày; ") ||
		len([]rune(got)) > 160 {
		t.Errorf("Vietnamese summary %q", got)
	}
	if got := keptSummary(i18n.Default, english, kept); got != english {
		t.Errorf("English reader %q", got)
	}
	if got := keptSummary(vi, "first published version", i18n.Text{}); got != "first published version" {
		t.Errorf("old row %q", got)
	}
	created := keptText(keepText(txt(mSummaryTemplate, "template", txt(msg("template.welcome.name")))))
	if got := keptSummary(vi, "created from the Welcome series template", created); got != "tạo từ mẫu Chuỗi chào mừng" {
		t.Errorf("creation summary %q", got)
	}
	if !fromCreation("created from the Welcome series template", created) || !fromCreation("created from a blank canvas", i18n.Text{}) ||
		fromCreation("duplicated from Welcome", txt(mSummaryDuplicated, "name", "Welcome")) {
		t.Error("fromCreation")
	}
}

// Every starter writes its cards, flow and emails in each language.
func TestStartersInEachLanguage(t *testing.T) {
	for _, l := range []i18n.Lang{i18n.Default, vi} {
		for _, tpl := range templates {
			card := tpl.card(l)
			if card.Name == "" || card.Summary == "" || card.Outline == "" {
				t.Errorf("%s card %+v", tpl.key, card)
			}
			g := tpl.graph("ref", l)
			if len(g.Nodes) != tpl.steps {
				t.Errorf("%s: %d steps, want %d", tpl.key, len(g.Nodes), tpl.steps)
			}
			for _, def := range tpl.emails {
				e := tpl.email(l, def)
				c := starterEmail(l, e)
				if e.name == "" || c.Subject == "" || !strings.Contains(c.HTML, "{{ unsubscribe_url }}") || len(e.body) != def.paras {
					t.Errorf("%s/%s: %+v", tpl.key, def.key, e)
				}
			}
		}
	}
	c := starterEmail(vi, templateEmail{subject: "x", heading: "x"})
	if !strings.Contains(c.HTML, `Bạn nhận được email này vì đã đăng ký {{ workspace_name }}. <a href="{{ unsubscribe_url }}">Hủy đăng ký</a> · {{ footer_address }}`) ||
		!strings.Contains(c.Text, "Hủy đăng ký: {{ unsubscribe_url }}") {
		t.Errorf("footer %s", c.HTML)
	}
	en := DemoEmail("x", "", "x", "")
	if !strings.Contains(en.HTML, `You get this because you subscribed to {{ workspace_name }}. <a href="{{ unsubscribe_url }}">Unsubscribe</a> · {{ footer_address }}`) {
		t.Errorf("English footer %s", en.HTML)
	}
}
