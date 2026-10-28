package engine_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

var vi = i18n.For(i18n.Vietnamese)

// checkAutoPauseLanguages: an auto-pause's analysis, notification and
// timeline entry keep the English they always had, and read in Vietnamese
// for a Vietnamese reader.
func checkAutoPauseLanguages(t *testing.T, f *sendtest.Fixture, snap engine.LiveSnapshot) {
	t.Helper()
	d := snap.Incident.Detail
	if d.Options[0].Title != "Resume as is" || d.Options[1].Title != "Resume with changes (recommended)" || d.Options[2].Title != "Cancel the Outlook share" ||
		d.Options[2].Action != "Cancel the Outlook share" || d.Options[1].Action != "Resume with changes" ||
		d.Note != "Microsoft's SmartScreen uses complaint rate directly; staying under 0.1% protects placement for every later send." ||
		!regexp.MustCompile(`^Lower the Outlook rate to \d+ /s for the rest of this send\.$`).MatchString(d.Recommendations[len(d.Recommendations)-1]) {
		t.Fatalf("English options %+v, note %q, recommendations %q", d.Options, d.Note, d.Recommendations)
	}
	vd := snap.In(vi).Incident.Detail
	if vd.Options[0].Title != "Tiếp tục như cũ" || vd.Options[1].Title != "Tiếp tục với thay đổi (khuyên dùng)" || vd.Options[2].Title != "Hủy phần của Outlook" ||
		vd.Note != "SmartScreen của Microsoft dùng trực tiếp tỷ lệ khiếu nại; giữ dưới 0,1% giúp bảo vệ vị trí thư cho mọi lần gửi sau." ||
		!strings.HasPrefix(vd.Recommendations[len(vd.Recommendations)-1], "Giảm tốc độ Outlook xuống ") {
		t.Fatalf("Vietnamese options %+v, note %q, recommendations %q", vd.Options, vd.Note, vd.Recommendations)
	}
	for i, c := range vd.Causes {
		if c.Body == d.Causes[i].Body {
			t.Fatalf("cause %d stayed English: %q", i, c.Body)
		}
	}
	for i, r := range vd.Recommendations {
		if r == d.Recommendations[i] {
			t.Fatalf("recommendation %d stayed English: %q", i, r)
		}
	}
	if snap.Incident.Detail.Options[0].Title != "Resume as is" {
		t.Fatal("In changed the snapshot it was given")
	}

	var title, summary, body string
	var data []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT title, summary, body, data FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1 LIMIT 1`,
		f.Campaign.String()).Scan(&title, &summary, &body, &data); err != nil {
		t.Fatal(err)
	}
	if title != `"Auto-pause" auto-paused · Outlook` ||
		!regexp.MustCompile(`^Outlook complaint rate reached \d+\.\d\d% \(limit 0\.10%\)\. [\d,]+ held, everything else still sending\.$`).MatchString(body) {
		t.Fatalf("English notification %q: %q", title, body)
	}
	vt, _, vb := notify.Localize(vi, title, summary, body, data)
	if vt != "“Auto-pause” tự tạm dừng · Outlook" ||
		!regexp.MustCompile(`^Tỷ lệ khiếu nại ở Outlook đạt \d+,\d\d% \(giới hạn 0,10%\)\. Đã tạm giữ [\d.]+ người nhận, phần còn lại vẫn đang gửi\.$`).MatchString(vb) {
		t.Fatalf("Vietnamese notification %q: %q", vt, vb)
	}

	var incident string
	var texts []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT title, texts FROM incidents WHERE kind = 'auto_pause' AND campaign_id = $1`, f.Campaign).Scan(&incident, &texts); err != nil {
		t.Fatal(err)
	}
	var kept map[string]i18n.Text
	_ = json.Unmarshal(texts, &kept)
	if !regexp.MustCompile(`^"Auto-pause" auto-paused: Outlook complaint rate reached \d+\.\d\d% \(limit 0\.10%\)\. [\d,]+ held\.$`).MatchString(incident) ||
		!strings.HasPrefix(kept["title"].In(vi), "“Auto-pause” tự tạm dừng: Tỷ lệ khiếu nại ở Outlook đạt ") {
		t.Fatalf("timeline %q, kept %q", incident, kept["title"].In(vi))
	}
}

// checkOutageLanguages: a held share's analysis and the decision request.
func checkOutageLanguages(t *testing.T, f *sendtest.Fixture, snap engine.LiveSnapshot) {
	t.Helper()
	d := snap.Incident.Detail
	if d.Options[0].Title != "Retry Postal" || d.Options[0].Action != "Retry Postal" || d.Options[1].Title != "Move the remaining Postal share to SES (recommended)" ||
		d.Move.RiskText != "Low · SES is warm" || d.Causes[0].Lead == "" {
		t.Fatalf("English outage %+v", d)
	}
	vd := snap.In(vi).Incident.Detail
	if vd.ErrorKind != "lỗi kết nối" || vd.Options[0].Title != "Thử lại Postal" || vd.Options[1].Title != "Chuyển phần còn lại của Postal sang SES (khuyên dùng)" ||
		vd.Options[2].Action != "Hủy phần của Postal" || vd.Move.RiskText != "Thấp · SES đã khởi động xong" || vd.Causes[0].Lead == d.Causes[0].Lead {
		t.Fatalf("Vietnamese outage %+v", vd)
	}
	for i, l := range d.Log {
		if l.Text == "health check · still failing" && vd.Log[i].Text != "kiểm tra tình trạng · vẫn lỗi" {
			t.Fatalf("health check line %q", vd.Log[i].Text)
		}
	}

	var title, summary, body string
	var data []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT title, summary, body, data FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1 AND data ? 'providerId' LIMIT 1`,
		f.Campaign.String()).Scan(&title, &summary, &body, &data); err != nil {
		t.Fatal(err)
	}
	// Until someone decides, the held share tests its provider (review GS-3).
	if title != `"Outage" is waiting for a decision` ||
		!regexp.MustCompile(`returned connection errors for \d+ minutes?\. Its share \([\d,]+ recipients\) is held; the rest keeps sending\. Until someone decides, ReputeCast tests Postal with one message a minute and sends the share again after 3 go through in a row\.$`).MatchString(body) {
		t.Fatalf("English decision %q: %q", title, body)
	}
	vt, _, vb := notify.Localize(vi, title, summary, body, data)
	if vt != "“Outage” đang chờ bạn quyết định" || !strings.Contains(vb, " trả về lỗi kết nối trong ") ||
		!strings.HasSuffix(vb, "người nhận) đang bị tạm giữ; phần còn lại vẫn tiếp tục gửi. Cho đến khi có người quyết định, ReputeCast kiểm tra Postal bằng một thư mỗi phút và gửi tiếp phần này sau 3 lần thành công liên tiếp.") {
		t.Fatalf("Vietnamese decision %q: %q", vt, vb)
	}
}
