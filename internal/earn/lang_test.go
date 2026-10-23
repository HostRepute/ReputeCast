package earn_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Earn's server text reads in the request's language: problems, the
// import's words, and what's stored for later readers (the import's
// notification and the complimentary access note it gives), which keeps
// its English for the row and reads in each member's own language.
func TestEarnInVietnamese(t *testing.T) {
	f := newFlow(t)
	f.connect()
	f.createTier()
	vi := f.env.Client()
	vi.SignIn("owner@flow.example", "")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	// Problems.
	if p := vi.Do(http.MethodGet, f.base+"/tiers/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t); p["detail"] != "Không tìm thấy gói." {
		t.Fatalf("not found (vi): %v", p)
	}
	if p := f.c.Do(http.MethodGet, f.base+"/tiers/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t); p["detail"] != "Tier not found." {
		t.Fatalf("not found (en): %v", p)
	}

	// A CSV import, read in Vietnamese: a CSV file in the reader's words.
	csv := "email,plan,paid_through,name\nkim@flow.example,Insider," + time.Now().AddDate(0, 6, 0).Format("2006-01-02") + ",Kim Reader\n"
	var imp struct{ ID, Platform string }
	f.c.DoRaw(http.MethodPost, f.base+"/imports?source=csv&fileName=export.csv", "application/octet-stream", strings.NewReader(csv)).MustStatus(t, 200).JSON(t, &imp)
	vi.Do(http.MethodGet, f.base+"/imports/"+imp.ID, nil).MustStatus(t, 200).JSON(t, &imp)
	if imp.Platform != "tệp CSV" {
		t.Fatalf("platform (vi): %q", imp.Platform)
	}
	f.c.Do(http.MethodGet, f.base+"/imports/"+imp.ID, nil).MustStatus(t, 200).JSON(t, &imp)
	if imp.Platform != "a CSV file" {
		t.Fatalf("platform (en): %q", imp.Platform)
	}
	f.c.Do(http.MethodPost, f.base+"/imports/"+imp.ID+"/match", nil).MustStatus(t, 200)
	f.c.Do(http.MethodPut, f.base+"/imports/"+imp.ID+"/decisions", map[string]any{"noSubscription": "complimentary", "sendWelcome": false}).MustStatus(t, 200)
	f.c.Do(http.MethodPost, f.base+"/imports/"+imp.ID+"/start", nil).MustStatus(t, 200)
	if err := f.svc.RunImport(context.Background(), uuid.MustParse(imp.ID)); err != nil {
		t.Fatal(err)
	}

	// The notification keeps its English in the row, and reads in each
	// member's language.
	var title, body string
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT title, body FROM notifications WHERE workspace_id = $1 AND kind = 'paid_import_finished'`, f.ws.ID).
		Scan(&title, &body); err != nil {
		t.Fatal(err)
	}
	if title != "Paid members imported from a CSV file" || body != "1 row: 0 paid, 0 free, 1 complimentary." {
		t.Fatalf("stored: %q, %q", title, body)
	}
	var list struct {
		Items []struct{ Title, Body string }
	}
	vi.Do(http.MethodGet, "/workspaces/"+f.ws.Slug+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) == 0 || list.Items[0].Title != "Đã nhập hội viên trả phí từ tệp CSV" || list.Items[0].Body != "1 dòng: 0 trả phí, 0 miễn phí, 1 được miễn phí." {
		t.Fatalf("notification (vi): %+v", list.Items)
	}
	f.c.Do(http.MethodGet, "/workspaces/"+f.ws.Slug+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Items[0].Title != "Paid members imported from a CSV file" {
		t.Fatalf("notification (en): %+v", list.Items)
	}

	// The complimentary access note an import gives reads in the reader's
	// language, until a person rewrites it.
	comp := f.members("complimentary")
	if len(comp) != 1 {
		t.Fatalf("complimentary: %+v", comp)
	}
	var m struct {
		GrantNote string
		Source    struct{ Kind, Label string }
	}
	vi.Do(http.MethodGet, f.base+"/members/"+comp[0].ID, nil).MustStatus(t, 200).JSON(t, &m)
	if m.GrantNote != "Đã trả phí đến ngày ghi trong tệp CSV, không có thuê bao trong Stripe" {
		t.Fatalf("grant note (vi): %q", m.GrantNote)
	}
	// So does where the member came from, stored in English.
	if m.Source.Kind != "import" || m.Source.Label != "Nhập từ tệp CSV" {
		t.Fatalf("source (vi): %+v", m.Source)
	}
	f.c.Do(http.MethodGet, f.base+"/members/"+comp[0].ID, nil).MustStatus(t, 200).JSON(t, &m)
	if m.GrantNote != "Paid-through date from the CSV file, with no subscription in Stripe" || m.Source.Label != "Imported from a CSV file" {
		t.Fatalf("grant note and source (en): %q, %+v", m.GrantNote, m.Source)
	}
	if _, err := f.env.App.Pool.Exec(context.Background(), `UPDATE paid_subscriptions SET grant_note = 'Moved from our old shop' WHERE id = $1`, comp[0].ID); err != nil {
		t.Fatal(err)
	}
	vi.Do(http.MethodGet, f.base+"/members/"+comp[0].ID, nil).MustStatus(t, 200).JSON(t, &m)
	if m.GrantNote != "Moved from our old shop" {
		t.Fatalf("a note a person wrote stays as written: %q", m.GrantNote)
	}
}
