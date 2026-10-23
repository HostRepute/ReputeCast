package deliverability_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/store"
)

// Deliverability reads in the request's language: the providers' account
// health, the timeline's links and delisting states, the DMARC sources'
// words and the problems. Names, DNS results and the providers' own words
// stay as they are; the export stays English.
func TestDeliverabilityInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	var o overview
	vi.Do("GET", "/workspaces/acme-newsletter/deliverability", nil).MustStatus(t, 200).JSON(t, &o)
	ses, pm, postal := o.Providers[0], o.Providers[1], o.Providers[2]
	if ses.Badge != "Ổn định · no enforcement" || ses.Bounce.Label != "Tỷ lệ trả thư" || ses.Bounce.Thresholds != "xem xét 5%, tạm dừng 10%" ||
		ses.Note.Value != "Chỉ có chỉ số trả thư và khiếu nại; không có VDM" {
		t.Fatalf("SES: %+v", ses)
	}
	if pm.Complaint.Label != "Tỷ lệ khiếu nại spam" || pm.Bounce.Thresholds != "giới hạn 10%" || pm.Note.Value != "Transactional · thư Hệ thống và Giao dịch" {
		t.Fatalf("Postmark: %+v", pm)
	}
	if postal.Name != "Postal (tự lưu trữ)" || postal.Badge != "Đang khởi động" || postal.Bounce.Thresholds != "ngưỡng bảo vệ khởi động 2%" ||
		postal.Complaint.Thresholds != "ngưỡng bảo vệ 0,1%" || postal.Note.Value != "Do ReputeCast đặt; Postal không có giới hạn tài khoản" {
		t.Fatalf("Postal: %+v", postal)
	}
	unknown := false
	for _, s := range o.DMARC.Sources {
		unknown = unknown || (o.DMARC.Unknown != nil && s.Label == "Không rõ · "+o.DMARC.Unknown.IP)
	}
	if o.Postmaster.Peak.Note != "lần gửi đầu tiên sau khi chuyển sang ReputeCast" || !unknown || o.DMARC.Sources[0].Label != "Amazon SES" {
		t.Fatalf("Postmaster %+v, DMARC %+v", o.Postmaster.Peak, o.DMARC.Sources)
	}
	items := o.Incidents.Items
	if items[0].Links[0].Label != "Kế hoạch khởi động" || items[3].Links[0].Label != "Báo cáo" || !strings.Contains(items[1].Detail, " Đã yêu cầu gỡ khỏi danh sách ngày ") {
		t.Fatalf("timeline: %+v", items)
	}

	if st, body := rawPostIn(t, env, "vi", "/workspaces/acme-newsletter/deliverability/dmarc/reports", []byte(" ")); st != 422 ||
		!strings.Contains(body, `"detail":"Hãy chọn một báo cáo tổng hợp DMARC: tệp .xml`) {
		t.Fatalf("Vietnamese problem: %d %s", st, body)
	}
	if st, body := rawPostIn(t, env, "vi", "/workspaces/acme-newsletter/deliverability/dmarc/reports", []byte("<feedback></feedback>")); st != 422 ||
		!strings.Contains(body, `"detail":"Đây không phải báo cáo tổng hợp DMARC: không có report_metadata hay bản ghi nào."`) {
		t.Fatalf("Vietnamese report problem: %d %s", st, body)
	}
	if st, body := rawPostIn(t, env, "en", "/workspaces/acme-newsletter/deliverability/dmarc/reports", []byte("<feedback></feedback>")); st != 422 ||
		!strings.Contains(body, `"detail":"This isn't a DMARC aggregate report: no report_metadata or records."`) {
		t.Fatalf("English report problem: %d %s", st, body)
	}

	// A provider's dedicated-IP offer reads in the adapter's words, kept with
	// its capability snapshot.
	if o.Pools[0].DedicatedIPOption != "không bắt buộc, 24,95 US$ mỗi IP mỗi tháng" {
		t.Fatalf("dedicated IPs in Vietnamese: %q", o.Pools[0].DedicatedIPOption)
	}

	// The CSV export stays English.
	r := vi.Do("GET", "/workspaces/acme-newsletter/deliverability/export", nil).MustStatus(t, 200)
	if !strings.Contains(string(r.Body), "review 5%, pause 10%") {
		t.Fatalf("export: %s", r.Body)
	}

	// Send Readiness reads the reputation reason in English; the API, in the
	// reader's language.
	_, _ = env.App.Pool.Exec(context.Background(), `UPDATE hostrepute_connections SET status = 'revoked', token_rejected_at = now()`)
	var rep struct{ Status, Reason string }
	vi.Do("GET", "/workspaces/acme-newsletter/deliverability/reputation", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Status != "unknown" || !strings.HasPrefix(rep.Reason, "HostRepute đã từ chối token vào ") || !strings.HasSuffix(rep.Reason, "; hãy kết nối lại để tiếp tục kiểm tra danh sách chặn.") {
		t.Fatalf("Vietnamese reason: %+v", rep)
	}
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	en.Do("GET", "/workspaces/acme-newsletter/deliverability/reputation", nil).MustStatus(t, 200).JSON(t, &rep)
	if !regexp.MustCompile(`^HostRepute rejected the token on [A-Z][a-z]{2} \d{1,2}; reconnect to resume blocklist checks\.$`).MatchString(rep.Reason) {
		t.Fatalf("English reason: %+v", rep)
	}
}

// rawPostIn uploads a raw body in a language, for the demo's Workspace Admin.
func rawPostIn(t *testing.T, env *apptest.Env, lang, path string, body []byte) (int, string) {
	t.Helper()
	ctx := context.Background()
	u, err := env.App.Q.GetUserByEmail(ctx, "linh@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	token, hash := authn.NewToken()
	if _, err := env.App.Q.CreateSession(ctx, store.CreateSessionParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, TokenHash: hash, AuthMethod: "test", MfaVerified: true, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1"+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept-Language", lang)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: env.App.Cfg.Auth.CookieName, Value: token})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// A listing the built-in check finds keeps its English in the row and its
// messages beside it: the timeline and the notification read in each
// reader's language.
func TestBuiltinListingInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	en.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)

	zone := dnscheck.NewZone()
	zone.Add("25.113.0.203.zen.spamhaus.org", "A", "127.0.0.3")
	s := &deliverability.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DNS: zone,
		HostRepute: &hostrepute.Service{}}
	var wsID uuid.UUID
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID)
	if err := s.RunChecks(ctx, deliverability.CheckArgs{WorkspaceID: wsID, Blocklists: true}); err != nil {
		t.Fatal(err)
	}
	var title, detail string
	_ = env.App.Pool.QueryRow(ctx, `SELECT title, detail FROM incidents WHERE source = 'builtin'`).Scan(&title, &detail)
	if title != "203.0.113.25 listed on Spamhaus ZEN" || detail != "Found by the built-in daily check (high impact)." {
		t.Fatalf("stored: %q, %q", title, detail)
	}
	var o overview
	vi.Do("GET", "/workspaces/acme-newsletter/deliverability", nil).MustStatus(t, 200).JSON(t, &o)
	found := false
	for _, it := range o.Incidents.Items {
		if it.Title == "203.0.113.25 bị liệt kê trên Spamhaus ZEN" && it.Detail == "Phát hiện qua lần kiểm tra hằng ngày có sẵn (mức ảnh hưởng cao)." {
			found = true
		}
	}
	if !found {
		t.Fatalf("timeline in Vietnamese: %+v", o.Incidents.Items)
	}
	var list struct {
		Items []struct{ Title, Body string }
	}
	vi.Do("GET", "/workspaces/acme-newsletter/notifications?limit=100", nil).MustStatus(t, 200).JSON(t, &list)
	found = false
	for _, n := range list.Items {
		if n.Title == "203.0.113.25 bị liệt kê trên Spamhaus ZEN" && strings.HasPrefix(n.Body, "Phát hiện qua lần kiểm tra hằng ngày có sẵn") {
			found = true
		}
	}
	if !found {
		t.Fatalf("notification in Vietnamese: %+v", list.Items)
	}

	// Why a run fell short is stored in English and reads in the reader's
	// language.
	var runID uuid.UUID
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM deliverability_check_runs WHERE trigger = 'fallback'`).Scan(&runID)
	_, _ = env.App.Pool.Exec(ctx, `UPDATE deliverability_check_runs SET status = 'done', error = '2 of 16 lookups didn''t answer; they show as unknown.' WHERE id = $1`, runID)
	var run struct{ Error string }
	vi.Do("GET", "/workspaces/acme-newsletter/deliverability/checks/"+runID.String(), nil).MustStatus(t, 200).JSON(t, &run)
	if run.Error != "2 trên 16 lượt tra cứu không có phản hồi; các kết quả đó hiển thị là không rõ." {
		t.Fatalf("Vietnamese run: %+v", run)
	}
	en.Do("GET", "/workspaces/acme-newsletter/deliverability/checks/"+runID.String(), nil).MustStatus(t, 200).JSON(t, &run)
	if run.Error != "2 of 16 lookups didn't answer; they show as unknown." {
		t.Fatalf("English run: %+v", run)
	}
	_, _ = env.App.Pool.Exec(ctx, `UPDATE deliverability_check_runs SET status = 'failed', error = 'HostRepute rejected the token. Reconnect to resume.' WHERE id = $1`, runID)
	vi.Do("GET", "/workspaces/acme-newsletter/deliverability/checks/"+runID.String(), nil).MustStatus(t, 200).JSON(t, &run)
	if run.Error != "HostRepute đã từ chối token. Hãy kết nối lại để tiếp tục." {
		t.Fatalf("Vietnamese failed run: %+v", run)
	}
}
