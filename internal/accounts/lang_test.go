package accounts_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

// The invitation page reads in the page's language: what the role can do,
// its scope, why two-factor is required and the closed-invitation problems
// come from the server in the language the app asks for.
func TestInvitationInVietnamese(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", linh.ID)
	beta := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, 'Beta Program')`, beta, ws.ID); err != nil {
		t.Fatal(err)
	}
	token, _ := invite(t, env, ws, linh.ID, "jo@contractor.dev", "editor", `{"lists":["`+beta.String()+`"]}`, 5*24*time.Hour)

	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	var v map[string]any
	vi.Do("GET", "/invitations/"+token, nil).MustStatus(t, 200).JSON(t, &v)
	want := map[string]string{
		"roleName":    "Biên tập viên",
		"scope":       "Chỉ danh sách Beta Program",
		"description": "Bạn sẽ tạo, sửa và gửi được chiến dịch cho danh sách Beta Program. Lần gửi trên 10.000 người nhận cần người thứ hai phê duyệt.",
		"inviterName": "Linh Nguyen",
	}
	for k, w := range want {
		if v[k] != w {
			t.Errorf("%s = %q, want %q", k, v[k], w)
		}
	}
	if r := v["twoFactor"].(map[string]any)["reason"]; r != "Vai trò của bạn có thể gửi chiến dịch, nên bắt buộc xác thực hai lớp." {
		t.Errorf("two-factor reason: %q", r)
	}

	// API clients get English unless they ask.
	var en map[string]any
	env.Client().Do("GET", "/invitations/"+token, nil).MustStatus(t, 200).JSON(t, &en)
	if en["roleName"] != "Editor" || en["scope"] != "Beta Program list only" ||
		en["description"] != "You'll be able to create, edit and send campaigns for the Beta Program list. Sends over 10,000 recipients need a second person's approval." {
		t.Fatalf("English: %v", en)
	}

	// An invitation whose inviter's account is gone names an admin of the workspace.
	r, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: ws.ID, Key: "viewer"})
	if err != nil {
		t.Fatal(err)
	}
	orphan, hash := authn.NewToken()
	now := time.Now()
	if _, err := env.App.Q.CreateInvitation(ctx, store.CreateInvitationParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: "kim@example.com", RoleID: r.ID, Scope: json.RawMessage(`{}`), TokenHash: hash,
		LastSentAt: &now, ExpiresAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	vi.Do("GET", "/invitations/"+orphan, nil).MustStatus(t, 200).JSON(t, &v)
	if v["inviterName"] != "Một quản trị viên của Acme Newsletter" || v["inviterUnknown"] != true || v["roleName"] != "Người xem" ||
		v["description"] != "Bạn sẽ xem được chiến dịch, báo cáo và cài đặt. Bạn không thể thay đổi chúng, và thông tin bí mật không bao giờ được hiển thị." {
		t.Fatalf("no inviter: %v", v)
	}
	p := vi.Do("POST", "/invitations/"+orphan+"/accept", map[string]any{"name": "Kim", "password": "a long enough passphrase"}).MustStatus(t, 410).Problem(t)
	if p["code"] != "invite_expired" || p["detail"] != "Lời mời này đã hết hạn. Hãy nhờ một quản trị viên của Acme Newsletter gửi lời mời mới." {
		t.Fatalf("expired, in Vietnamese: %v", p)
	}
	var req map[string]any
	vi.Do("POST", "/invitations/"+orphan+"/request", nil).MustStatus(t, 200).JSON(t, &req)
	if req["inviter"] != "một quản trị viên của Acme Newsletter" {
		t.Fatalf("request: %v", req)
	}
}

// The sign-in pages show the server's problems in the page's language.
func TestSignInProblemsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io", Name: "Aisha Bello"})
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	p := vi.Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "nope"}).MustStatus(t, 401).Problem(t)
	detail, _ := p["detail"].(string)
	if p["code"] != "invalid_credentials" || !strings.HasPrefix(detail, "Email hoặc mật khẩu không đúng. Còn ") || !strings.HasSuffix(detail, " lần thử trước khi bị khóa 15 phút.") {
		t.Fatalf("wrong password, in Vietnamese: %v", p)
	}
	if p := vi.Do("GET", "/invitations/not-a-real-token-0000", nil).MustStatus(t, 404).Problem(t); p["detail"] != "Liên kết mời này không hợp lệ. Hãy kiểm tra bạn đã sao chép đủ liên kết, hoặc xin lời mời mới." {
		t.Fatalf("unknown invitation, in Vietnamese: %v", p)
	}
	// English stays as it was.
	p = env.Client().Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "nope"}).MustStatus(t, 401).Problem(t)
	detail, _ = p["detail"].(string)
	if !strings.HasPrefix(detail, "The email or password is incorrect. ") || !strings.HasSuffix(detail, " left before a 15-minute lock.") {
		t.Fatalf("wrong password, in English: %v", p)
	}
}
