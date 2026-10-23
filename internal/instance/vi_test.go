package instance_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
)

// Instance admin reads in the admin's language: the problems name their
// field in any language; the two-factor reset's notification, written while
// the admin read English, reads in Vietnamese for a Vietnamese reader; and
// the users list's sign-in events read in Vietnamese.
func TestInstanceAdminInVietnamese(t *testing.T) {
	env, admin := setup(t)
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")

	p := vi.Do("PUT", "/admin/instance", map[string]any{"name": " ", "publicUrl": "https://reputecast.acme.io", "defaultTimezone": "UTC"}).MustStatus(t, 422).Problem(t)
	errs, _ := p["errors"].([]any)
	if p["detail"] != "Hãy đặt tên cho hệ thống, như Acme ReputeCast." || len(errs) != 1 || errs[0].(map[string]any)["location"] != "body.name" {
		t.Errorf("instance name: %v", p)
	}
	if p := vi.Do("POST", "/admin/users/"+userID(t, env, "linh@acme.io")+"/deactivate", map[string]any{"reason": "testing"}).MustStatus(t, 409).Problem(t); p["detail"] != "Bạn không thể vô hiệu hóa tài khoản của chính mình. Hãy nhờ một quản trị viên hệ thống khác." {
		t.Errorf("deactivate yourself: %v", p)
	}

	id := userID(t, env, "aisha@acme.io")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE users SET totp_enabled_at = now(), totp_secret_enc = '\x00' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	admin.Do("POST", "/admin/users/"+id+"/two-factor-reset", map[string]any{"reason": "lost her phone", "password": demo.Password}).MustStatus(t, 204)
	var title, summary, body string
	var data []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT title, summary, body, data FROM notifications WHERE user_id = $1 AND kind = 'twofa_reset'`, id).Scan(&title, &summary, &body, &data); err != nil {
		t.Fatal(err)
	}
	if title != "Two-factor was reset" || summary != "Linh Nguyen reset your two-factor" {
		t.Errorf("English: %q · %q", title, summary)
	}
	vt, vs, vb := notify.Localize(i18n.For(i18n.Vietnamese), title, summary, body, data)
	if vt != "Xác thực hai lớp đã được đặt lại" || vs != "Linh Nguyen đã đặt lại xác thực hai lớp của bạn" ||
		!strings.HasPrefix(vb, "Linh Nguyen đã đặt lại xác thực hai lớp cho tài khoản của bạn (lý do: lost her phone). Hãy thiết lập lại ứng dụng xác thực trước ngày ") {
		t.Errorf("Vietnamese: %q · %q · %q", vt, vs, vb)
	}

	var events struct {
		Items []struct{ Event, Result, Detail string }
	}
	vi.Do("GET", "/admin/sign-ins?limit=8", nil).MustStatus(t, 200).JSON(t, &events)
	if len(events.Items) == 0 {
		t.Fatal("no sign-in events")
	}
	for _, e := range events.Items {
		if strings.Contains(e.Event, "Password") || e.Detail == "domain not allowed" || e.Detail == "wrong password" {
			t.Errorf("an event in English: %+v", e)
		}
	}
}
