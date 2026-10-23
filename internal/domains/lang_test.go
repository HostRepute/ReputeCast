package domains_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Settings › Senders and a domain's records read in the request's language:
// the table's cells, the senders' badges, the records' purposes, notes and
// statuses, the DMARC tip and the problems. DNS words stay as DNS writes
// them; API clients that send no Accept-Language keep getting English.
func TestSendersInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	var ov overviewT
	vi.Do("GET", ws+"/senders", nil).MustStatus(t, 200).JSON(t, &ov)
	news, notify, acme := ov.Domains[0], ov.Domains[1], ov.Domains[2]
	if news.DKIM.Text != "3/3 nhà cung cấp" || news.SPF.Text != "pass · -all" || news.DMARC.Text != "p=quarantine" || news.BIMI.Text != "Chưa đặt" {
		t.Fatalf("news.acme.io: %+v", news)
	}
	if notify.Tracking.Text != "Tắt · không theo dõi" || acme.DKIM.Text != "Đang chờ CNAME 2/3" || acme.ReturnPath.Text != "Mặc định" {
		t.Fatalf("notify.acme.io %+v, acme.io %+v", notify, acme)
	}
	by := map[string]int{}
	for i, s := range ov.Senders {
		by[s.Email] = i
	}
	marcus := ov.Senders[by["marcus@acme.io"]]
	if marcus.Verification[0].Text != "Đang chờ · DKIM acme.io" || marcus.Verification[1].Text != "Chưa thiết lập" {
		t.Fatalf("marcus: %+v", marcus.Verification)
	}

	var detail struct {
		DMARCTip string `json:"dmarcTip"`
		Records  []struct{ Purpose, Label, Note, StatusText string }
	}
	vi.Do("GET", ws+"/domains/"+acme.ID, nil).MustStatus(t, 200).JSON(t, &detail)
	labels := map[string]string{}
	for _, r := range detail.Records {
		labels[r.Purpose+"|"+r.Label] = r.StatusText
	}
	if _, ok := labels["dkim|DKIM 2/3"]; !ok {
		t.Fatalf("records: %+v", detail.Records)
	}
	if !strings.HasPrefix(detail.DMARCTip, "acme.io ") || !strings.Contains(detail.DMARCTip, "`p=none`") {
		t.Fatalf("DMARC tip: %q", detail.DMARCTip)
	}
	for _, r := range detail.Records {
		if r.Purpose == "dmarc" && r.Note != "Gmail và Yahoo bắt buộc với người gửi hàng loạt" {
			t.Fatalf("DMARC note: %+v", r)
		}
		if r.StatusText != "Đã xác minh" && r.StatusText != "Đang chờ" && r.StatusText != "Còn thiếu" && !strings.HasPrefix(r.StatusText, "Đang chờ ·") {
			t.Fatalf("status: %+v", r)
		}
	}

	if pr := vi.Do("POST", ws+"/senders", map[string]any{"name": "X", "email": "not an address"}).MustStatus(t, 422).Problem(t); pr["detail"] != "Hãy nhập một địa chỉ như hello@news.acme.io." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	if pr := vi.Do("POST", ws+"/domains", map[string]any{"domain": "news.acme.io"}).MustStatus(t, 409).Problem(t); pr["detail"] != "news.acme.io đã là tên miền gửi thư của không gian làm việc này." {
		t.Fatalf("Vietnamese conflict: %v", pr)
	}
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	if pr := en.Do("POST", ws+"/domains", map[string]any{"domain": "news.acme.io"}).MustStatus(t, 409).Problem(t); pr["detail"] != "news.acme.io is already a sending domain of this workspace." {
		t.Fatalf("English conflict: %v", pr)
	}
}
