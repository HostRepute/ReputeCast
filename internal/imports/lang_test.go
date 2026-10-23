package imports_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/imports"
)

// The wizard, the problems, the jobs tray and the live log read in the
// member's language; what the import stores (the log, the source's
// failure, the consent history, the suppressions) stays English and reads
// in each reader's language. English readers and API clients see what they
// always saw.
func TestImportInVietnamese(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	vi := i18n.WithLang(ctx, i18n.For(i18n.Vietnamese))
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	ws := env.CreateWorkspace("Importers", owner.ID)
	c := env.Client()
	c.SignIn("owner@example.com", "")
	base := "/workspaces/" + ws.Slug
	var list struct{ ID string }
	c.Do(http.MethodPost, base+"/lists", map[string]any{"name": "Newsletter"}).MustStatus(t, 201).JSON(t, &list)
	c.Do(http.MethodPost, base+"/contacts", map[string]any{"email": "ana@gmail.com", "lists": []string{list.ID}, "consent": "single"}).MustStatus(t, 201)

	const key = "5f0c9e3a7b2d4e6f8a1b3c5d7e9f0a2b-us21"
	srv := fakeMailchimp(t, key)
	defer srv.Close()
	svc := service(env, map[string]string{"mailchimp": srv.URL})
	imp, err := svc.Create(ctx, &ws, imports.ImportCreateInput{Source: "mailchimp"})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse(imp.ID)

	// The source's failure: the problem in the request's language, and kept
	// for the error state in each reader's.
	_, err = svc.Connect(vi, &ws, id, imports.ImportConnectInput{APIKey: "0000-us21"})
	if err == nil || !strings.Contains(err.Error(), `Mailchimp đã từ chối khóa API: "API Key Invalid" của trung tâm dữ liệu us21.`) {
		t.Fatalf("a wrong key, in Vietnamese: %v", err)
	}
	if got, err := svc.Get(vi, &ws, id); err != nil || got.Error == nil || !strings.HasPrefix(got.Error.Message, "Mailchimp đã từ chối khóa API") {
		t.Fatalf("the kept failure, in Vietnamese: %+v %v", got.Error, err)
	}
	const rejected = `Mailchimp rejected the API key: "API Key Invalid" from the us21 data center. Keys look like xxxxxxxx-us21; the suffix must match the data center the account lives in.`
	if got, err := svc.Get(ctx, &ws, id); err != nil || got.Error == nil || got.Error.Message != rejected {
		t.Fatalf("the kept failure, in English: %+v %v", got.Error, err)
	}

	got, err := svc.Connect(ctx, &ws, id, imports.ImportConnectInput{APIKey: key})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	opts := got.Options
	opts.Lists = []imports.ImportListChoice{{SourceID: "3f9a2c", Target: "new:Acme Customers"}}
	if _, err := svc.Update(ctx, &ws, id, imports.ImportUpdateInput{Options: &opts}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestStage(ctx, &ws, id); err != nil {
		t.Fatalf("stage: %v", err)
	}

	fields, err := svc.Fields(vi, &ws, id)
	if err != nil || len(fields.Fields) == 0 {
		t.Fatalf("fields: %+v %v", fields, err)
	}
	labels := map[string]string{}
	for _, f := range fields.Fields {
		for _, o := range f.Options {
			labels[f.Source+" "+o.Value] = o.Label
		}
	}
	if labels["EMAIL email"] != "email · bắt buộc" || labels["FNAME skip"] != "Bỏ qua" || labels["FNAME first_name"] != "first_name" ||
		labels["FNAME consent_evidence"] != "Bằng chứng đồng ý (ngày, IP)" || labels["FNAME new:fname"] != `Tạo trường mới "fname"` {
		t.Errorf("field targets in Vietnamese: %v", labels)
	}

	consent, err := svc.Consent(vi, &ws, id)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]imports.ImportConsentStatus{}
	for _, s := range consent.Statuses {
		rows[s.Status] = s
	}
	if s := rows["subscribed"]; s.Label != "subscribed" || s.Note != "audience này đã bật xác nhận hai bước từ năm 2024" ||
		s.Options[0].Label != "Đã đăng ký · đã xác nhận, giữ ngày và IP đăng ký làm bằng chứng" || s.Effect != "Nhận Gửi hàng loạt và Tự động hóa ngay" {
		t.Errorf("subscribed, in Vietnamese: %+v", s)
	}
	if s := rows["unsubscribed"]; s.Effect != "Chặn Gửi hàng loạt và Tự động hóa; vẫn nhận thư Hệ thống. Lần nhập không thay đổi được điều này." {
		t.Errorf("unsubscribed, in Vietnamese: %+v", s)
	}
	if s := rows["complained"]; s.Label != `archived, có khiếu nại "abuse"` {
		t.Errorf("complained, in Vietnamese: %+v", s)
	}
	if h := consent.HistoryExample; !strings.HasPrefix(h, "Nhập từ Mailchimp, ") || !strings.HasSuffix(h, ", trạng thái subscribed, bằng chứng OPTIN_TIME 2025-03-02") {
		t.Errorf("history example, in Vietnamese: %q", h)
	}
	en, err := svc.Consent(ctx, &ws, id)
	if err != nil {
		t.Fatal(err)
	}
	if s := en.Statuses[0]; s.Note != "double opt-in was on in this audience since 2024" || s.Options[0].Label != "Subscribed · confirmed, keep opt-in date and IP as evidence" {
		t.Errorf("subscribed, in English: %+v", s)
	}

	p, err := svc.Preview(vi, &ws, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Duplicates) != 1 || p.Duplicates[0].Result != "Gộp · điền tên và múi giờ" || p.Duplicates[0].InReputeCast != "Newsletter · tên trống" || p.Duplicates[0].InSource != `tên "Ana"` {
		t.Errorf("duplicates, in Vietnamese: %+v", p.Duplicates)
	}
	if p, err := svc.Preview(ctx, &ws, id); err != nil || p.Duplicates[0].Result != "Merge · fills first name and timezone" || p.Duplicates[0].InReputeCast != "Newsletter · first name empty" {
		t.Errorf("duplicates, in English: %+v %v", p.Duplicates, err)
	}

	if _, err := svc.Start(ctx, &ws, id); err != nil {
		t.Fatalf("start: %v", err)
	}
	logged := func(ctx context.Context) string {
		got, err := svc.Get(ctx, &ws, id)
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for _, e := range got.Log {
			lines = append(lines, e.Message)
		}
		return strings.Join(lines, "\n")
	}
	vlog, elog := logged(vi), logged(ctx)
	for _, want := range []string{
		`Bắt đầu nhập · 2 hàng · audience "Acme Customers" → danh sách Acme Customers · sự đồng ý: tin tưởng xác nhận hai bước`,
		"Lô 1 · 2 hàng: nhập 0, gộp 1, chặn gửi 1, bỏ qua 0",
		"Đã nhập xong · 1 liên hệ · 1 địa chỉ chặn gửi",
	} {
		if !strings.Contains(vlog, want) {
			t.Errorf("the live log, in Vietnamese, has no %q:\n%s", want, vlog)
		}
	}
	for _, want := range []string{
		`Import started · 2 rows · audience "Acme Customers" → list Acme Customers · consent: trust double opt-in`,
		"Batch 1 · 2 rows: 0 imported, 1 merged, 1 suppressed, 0 skipped",
		"Import finished · 1 contacts · 1 suppressions",
	} {
		if !strings.Contains(elog, want) {
			t.Errorf("the live log, in English, has no %q:\n%s", want, elog)
		}
	}

	// The suppression and the consent history store English and keep their
	// messages for readers of other languages.
	var code, where string
	var texts json.RawMessage
	if err := env.App.Pool.QueryRow(ctx, `SELECT original_code, context, texts FROM suppressions WHERE workspace_id = $1 AND email = 'ben@gmail.com'`, ws.ID).Scan(&code, &where, &texts); err != nil {
		t.Fatal(err)
	}
	vl := i18n.For(i18n.Vietnamese)
	if code != "unsubscribed in Mailchimp" || where != "Migration" ||
		audience.KeptIn(vl, texts, "original_code", code) != "đã hủy đăng ký trong Mailchimp" || audience.KeptIn(vl, texts, "context", where) != "Chuyển từ công cụ khác" {
		t.Errorf("suppression: %q %q %s", code, where, texts)
	}
	var detail, note string
	if err := env.App.Pool.QueryRow(ctx, `SELECT ce.detail, ce.note, ce.texts FROM consent_events ce JOIN contacts c ON c.id = ce.contact_id
		WHERE c.workspace_id = $1 AND c.email = 'ana@gmail.com' AND ce.action = 'imported'`, ws.ID).Scan(&detail, &note, &texts); err != nil {
		t.Fatal(err)
	}
	if detail != `list "Acme Customers" · status subscribed` || note != "evidence OPTIN_TIME 2025-03-02" ||
		audience.KeptIn(vl, texts, "detail", detail) != `danh sách "Acme Customers" · trạng thái subscribed` || audience.KeptIn(vl, texts, "note", note) != "bằng chứng OPTIN_TIME 2025-03-02" {
		t.Errorf("consent history: %q %q %s", detail, note, texts)
	}

	jobs, err := svc.BackgroundJobs(vi, &ws)
	if err != nil || len(jobs.Items) != 1 || jobs.Items[0].Title != "Đã nhập từ Mailchimp" || jobs.Items[0].Detail != "Đã nhập 1 liên hệ · 1 bị chặn gửi" {
		t.Errorf("jobs tray, in Vietnamese: %+v %v", jobs.Items, err)
	}
	if jobs, _ := svc.BackgroundJobs(ctx, &ws); jobs.Items[0].Title != "Imported from Mailchimp" || jobs.Items[0].Detail != "1 contacts imported · 1 suppressed" {
		t.Errorf("jobs tray, in English: %+v", jobs.Items)
	}

	// Over HTTP: a problem in the request's language, and a file's failure
	// kept for each reader.
	vc := env.Client()
	vc.Header = http.Header{"Accept-Language": {"vi"}}
	vc.SignIn("owner@example.com", "")
	var d struct {
		ID    string
		Error *struct{ Code, Message string }
	}
	vc.Do(http.MethodPost, base+"/imports", map[string]any{"source": "csv"}).MustStatus(t, 201).JSON(t, &d)
	if p := vc.Do(http.MethodPost, base+"/imports/"+d.ID+"/stage", nil).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy chọn ít nhất một danh sách để nhập." {
		t.Errorf("stage without lists, in Vietnamese: %v", p["detail"])
	}
	if p := c.Do(http.MethodPost, base+"/imports/"+d.ID+"/stage", nil).MustStatus(t, 422).Problem(t); p["detail"] != "Choose at least one list to import." {
		t.Errorf("stage without lists, in English: %v", p["detail"])
	}
	if _, err := svc.Upload(ctx, &ws, uuid.MustParse(d.ID), "empty.csv", []byte("\n\n")); err != nil {
		t.Fatal(err)
	}
	vc.Do(http.MethodGet, base+"/imports/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Error == nil || d.Error.Code != "file_unreadable" || d.Error.Message != "empty.csv trống." {
		t.Errorf("an empty file, in Vietnamese: %+v", d.Error)
	}
	c.Do(http.MethodGet, base+"/imports/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Error == nil || d.Error.Message != "empty.csv is empty." {
		t.Errorf("an empty file, in English: %+v", d.Error)
	}
}
