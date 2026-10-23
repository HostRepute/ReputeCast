package transactional_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Transactional's editor reads in the member's language: its problems, the
// checks, a preview's problem and the publish dialog. The categories
// ReputeCast creates are the workspace's from then on, written once in its
// default language. The send API keeps its English.
func TestTransactionalInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	// The workspace writes in Vietnamese first.
	for _, q := range []string{
		`UPDATE workspace_languages SET is_default = false WHERE workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`,
		`UPDATE workspace_languages SET is_default = true WHERE code = 'vi' AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`,
	} {
		if _, err := env.App.Pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	vi := env.Client()
	vi.SignInDemo("sofia@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	en := env.Client()
	en.SignInDemo("sofia@acme.io")

	// The first template creates the Essential category, in the
	// workspace's language, whoever creates it.
	var tpl ttemplate
	en.Do("POST", base+"/templates", map[string]any{"name": "Password reset", "starter": "password-reset"}).MustStatus(t, 201).JSON(t, &tpl)
	if tpl.Category == nil || tpl.Category.Name != "Tài khoản" || tpl.Category.Kind != "essential" {
		t.Fatalf("category: %+v", tpl.Category)
	}

	// Problems.
	if p := vi.Do("GET", base+"/templates/nope", nil).MustStatus(t, 404).Problem(t); p["detail"] != "Không tìm thấy mẫu." {
		t.Fatalf("not found (vi): %v", p)
	}
	if p := en.Do("GET", base+"/templates/nope", nil).MustStatus(t, 404).Problem(t); p["detail"] != "Template not found." {
		t.Fatalf("not found (en): %v", p)
	}
	bad := map[string]any{"baseRevision": tpl.DraftRevision, "variables": []map[string]any{{"name": "to", "type": "text", "required": false, "sensitive": false}}}
	if p := vi.Do("PUT", base+"/templates/password-reset/draft", bad).MustStatus(t, 422).Problem(t); p["detail"] != "to do ReputeCast điền; hãy chọn tên khác." {
		t.Fatalf("variable (vi): %v", p)
	}
	bad["variables"] = []map[string]any{{"name": "count", "type": "number", "sample": "abc", "required": false, "sensitive": false}}
	if p := vi.Do("PUT", base+"/templates/password-reset/draft", bad).MustStatus(t, 422).Problem(t); p["detail"] != `Giá trị mẫu của count không phải là số hợp lệ: "abc" không phải là số.` {
		t.Fatalf("sample (vi): %v", p)
	}
	if p := en.Do("PUT", base+"/templates/password-reset/draft", bad).MustStatus(t, 422).Problem(t); p["detail"] != `The sample of count isn't a valid number: "abc" is not a number.` {
		t.Fatalf("sample (en): %v", p)
	}
	if p := vi.Do("POST", base+"/templates/password-reset/preview", map[string]any{"version": "nope"}).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy chọn bản nháp, bản đang chạy hoặc một số phiên bản." {
		t.Fatalf("preview (vi): %v", p)
	}

	// The checks: an undeclared variable in the subject, and what passes.
	type check struct {
		ID, Title, Detail string
		Fixes             []struct{ Label string }
	}
	var checks struct{ Checks []check }
	find := func(id string) check {
		t.Helper()
		for _, c := range checks.Checks {
			if c.ID == id {
				return c
			}
		}
		t.Fatalf("no %s in %+v", id, checks.Checks)
		return check{}
	}
	unsaved := map[string]any{"subject": "Hi {{ nickname }}"}
	vi.Do("POST", base+"/templates/password-reset/checks", unsaved).MustStatus(t, 200).JSON(t, &checks)
	if c := find("variables.undeclared"); c.Title != "nickname được dùng nhưng chưa khai báo" ||
		c.Detail != "nickname được dùng trong tiêu đề nhưng chưa khai báo. Lệnh gọi sẽ không gửi biến này, nên nó không hiện gì." ||
		len(c.Fixes) != 2 || c.Fixes[0].Label != "Khai báo là văn bản không bắt buộc" {
		t.Fatalf("undeclared (vi): %+v", c)
	}
	if c := find("liquid.ok"); c.Title != "Liquid kết xuất được" {
		t.Fatalf("passing (vi): %+v", c)
	}
	en.Do("POST", base+"/templates/password-reset/checks", unsaved).MustStatus(t, 200).JSON(t, &checks)
	if c := find("variables.undeclared"); c.Title != "nickname is used but not declared" ||
		c.Detail != "nickname is used in the subject but not declared. Calls won't send it, so it prints nothing." {
		t.Fatalf("undeclared (en): %+v", c)
	}

	// A preview that misses required variables says so in the reader's words.
	var prev struct {
		OK      bool
		Problem string
	}
	vi.Do("POST", base+"/templates/password-reset/preview", map[string]any{"data": map[string]any{"first_name": "Sam"}}).MustStatus(t, 200).JSON(t, &prev)
	if prev.OK || prev.Problem != "reset_url và expires_in là bắt buộc nhưng đang thiếu. Một lệnh gọi thật có dạng như vậy sẽ thất bại với missing_variables." {
		t.Fatalf("preview problem (vi): %+v", prev)
	}

	// The publish dialog's first version.
	var pv struct {
		Diff struct {
			ContentSummary string
			Variables      []struct{ Name, Summary string }
		}
	}
	vi.Do("POST", base+"/templates/password-reset/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.Diff.ContentSummary != "Phiên bản đầu tiên" || len(pv.Diff.Variables) == 0 || !strings.Contains(pv.Diff.Variables[0].Summary, "không bắt buộc") {
		t.Fatalf("publish dialog (vi): %+v", pv.Diff)
	}
	en.Do("POST", base+"/templates/password-reset/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.Diff.ContentSummary != "First version" || !strings.Contains(pv.Diff.Variables[0].Summary, "optional") {
		t.Fatalf("publish dialog (en): %+v", pv.Diff)
	}
}
