package accounts_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
)

// Account & security's problems read in the member's language, and the
// inviter's notification, written while the invitee read English, reads
// in Vietnamese for a Vietnamese reader.
func TestAccountServerTextInVietnamese(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", linh.ID)

	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignIn("linh@acme.io", "")
	if p := vi.Do("PATCH", "/me/profile", map[string]any{"name": " ", "timezone": "UTC"}).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy nhập tên của bạn như nhóm sẽ thấy." {
		t.Errorf("profile name: %v", p)
	}
	if p := env.Client().Do("PATCH", "/me/profile", map[string]any{"name": " ", "timezone": "UTC"}).MustStatus(t, 401).Problem(t); p["detail"] != "Sign in to continue." {
		t.Errorf("English sign-in: %v", p)
	}

	token, _ := invite(t, env, ws, linh.ID, "jo@contractor.dev", "editor", `{}`, 5*24*time.Hour)
	env.Client().Do("POST", "/invitations/"+token+"/accept", map[string]any{"name": "Jo Lindqvist", "password": "a long enough passphrase"}).MustStatus(t, 200)
	var title, summary, body string
	var data []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT title, summary, body, data FROM notifications WHERE user_id = $1 AND kind = 'invite_accepted'`, linh.ID).Scan(&title, &summary, &body, &data); err != nil {
		t.Fatal(err)
	}
	if title != "Invite accepted" || summary != "Jo Lindqvist joined Acme Newsletter as Editor" {
		t.Errorf("English: %q · %q", title, summary)
	}
	vt, vs, vb := notify.Localize(i18n.For(i18n.Vietnamese), title, summary, body, data)
	if vt != "Lời mời đã được chấp nhận" || vs != "Jo Lindqvist đã tham gia Acme Newsletter với vai trò Biên tập viên" ||
		vb != "Jo Lindqvist đã tham gia Acme Newsletter với vai trò Biên tập viên." {
		t.Errorf("Vietnamese: %q · %q · %q", vt, vs, vb)
	}
}

// The public URL check is stored with its lines' messages, and reads in
// each reader's language; a check stored before keeps its English.
func TestURLCheckKeptForLaterReaders(t *testing.T) {
	stored := []byte(`{"checkedAt":"2026-10-22T09:00:00Z","dns":{"status":"bad","detail":"x doesn't resolve.","i18n":{"id":"accounts.check.dnsBad","args":{"host":"reputecast.acme.io"}}},` +
		`"tls":{"status":"warn","detail":"Plain http: passkeys, secure cookies and most providers' webhooks need https."},"reachable":{"status":"ok","detail":"Answered"}}`)
	var c accounts.URLCheck
	if err := json.Unmarshal(stored, &c); err != nil {
		t.Fatal(err)
	}
	v := c.In(i18n.For(i18n.Vietnamese))
	if v.DNS.Detail != "reputecast.acme.io không phân giải được. Hãy thêm bản ghi A hoặc AAAA trỏ về máy chủ này." {
		t.Errorf("kept line: %q", v.DNS.Detail)
	}
	if v.TLS.Detail != "Plain http: passkeys, secure cookies and most providers' webhooks need https." {
		t.Errorf("a line stored before reads in English: %q", v.TLS.Detail)
	}
	out, _ := json.Marshal(v)
	again, _ := json.Marshal(c)
	if json.Valid(out) && string(out) == string(again) {
		t.Error("the API answer still carries the kept messages")
	}
	if en := c.In(i18n.Default); en.DNS.Detail != "reputecast.acme.io doesn't resolve. Add an A or AAAA record pointing at this server." {
		t.Errorf("English: %q", en.DNS.Detail)
	}
}
