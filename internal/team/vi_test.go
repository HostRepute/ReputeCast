package team_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// The member page and the team's problems read in the app's language, and
// a change made in English keeps its audit summary's messages for readers
// of other languages.
func TestMemberPageInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")
	var l memberList
	vi.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)
	daniel, linh, aisha := byEmail(t, l, "daniel@acme.io"), byEmail(t, l, "linh@acme.io"), byEmail(t, l, "aisha@acme.io")

	var detail struct {
		Groups      []string
		Permissions []struct {
			Key, Group, Label, Source, Scope string
			Granted                          bool
		}
		KeysNote string
	}
	vi.Do("GET", ws+"/members/"+daniel.ID, nil).MustStatus(t, 200).JSON(t, &detail)
	if detail.Groups[0] != "Chiến dịch" || detail.KeysNote != "Không có. Vai trò này không thể tạo khóa API." {
		t.Errorf("detail: %v · %q", detail.Groups, detail.KeysNote)
	}
	for _, p := range detail.Permissions {
		switch p.Key {
		case "campaigns:view":
			if p.Group != "Chiến dịch" || p.Label != "Xem chiến dịch" || p.Source != "Vai trò · Content Writer" || p.Scope != "mọi danh sách" {
				t.Errorf("campaigns:view: %+v", p)
			}
		case "campaigns:send":
			if p.Source != "không được cấp" {
				t.Errorf("campaigns:send: %+v", p)
			}
		}
	}
	if p := vi.Do("POST", ws+"/members/"+linh.ID+"/deactivate", map[string]any{}).MustStatus(t, 409).Problem(t); p["detail"] != "Bạn không thể tự vô hiệu hóa chính mình. Hãy nhờ một thành viên khác có quyền quản lý nhóm, hoặc rời không gian làm việc trong Tài khoản và bảo mật." {
		t.Errorf("deactivate yourself: %v", p)
	}

	// Signed out in English: the audit log reads the summary in Vietnamese.
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	en.Do("POST", ws+"/members/"+aisha.ID+"/sign-out", nil).MustStatus(t, 200)
	var raw []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT metadata FROM audit_events WHERE action = 'member.signed_out' ORDER BY occurred_at DESC LIMIT 1`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var meta struct{ Summary string }
	_ = json.Unmarshal(raw, &meta)
	kept := audit.KeptIn(raw)
	if meta.Summary == "" || kept.SummaryIn(i18n.Default, "") != meta.Summary || kept.SummaryIn(i18n.For(i18n.Vietnamese), "") == meta.Summary {
		t.Errorf("kept summary: %q, %+v", meta.Summary, kept)
	}
}
