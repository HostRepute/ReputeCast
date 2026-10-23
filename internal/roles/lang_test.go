package roles_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Roles & permissions reads in the app's language: the permission
// catalogue, the built-in roles, the impact preview and the problems.
func TestRolesInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")

	type list struct {
		Roles []struct {
			ID, Key, Name, Description string
		}
		Permissions []struct{ Key, Group, Label, Description string }
		Groups      []string
	}
	var l list
	vi.Do("GET", ws+"/roles", nil).MustStatus(t, 200).JSON(t, &l)
	if p := l.Permissions[0]; p.Key != "campaigns:view" || p.Group != "Chiến dịch" || p.Label != "Xem chiến dịch" || p.Description != "Xem chiến dịch, nội dung, người nhận và trạng thái của chúng." {
		t.Errorf("catalogue: %+v", p)
	}
	if l.Groups[0] != "Chiến dịch" {
		t.Errorf("groups: %v", l.Groups)
	}
	editor := ""
	for _, r := range l.Roles {
		switch r.Key {
		case "viewer":
			if r.Name != "Người xem" || r.Description != "Chỉ đọc. Không bao giờ thấy thông tin bí mật" {
				t.Errorf("viewer: %+v", r)
			}
		case "editor":
			editor = r.ID
		case "content-writer":
			if r.Name != "Content Writer" {
				t.Errorf("a custom role keeps its name: %+v", r)
			}
		}
	}

	var imp struct{ Sensitive, Approvals string }
	vi.Do("POST", ws+"/roles/impact", map[string]any{"permissions": []string{"campaigns:edit"}}).MustStatus(t, 200).JSON(t, &imp)
	if imp.Sensitive != "Chưa chọn · không bắt buộc xác thực hai lớp" ||
		imp.Approvals != "Người giữ vai trò này yêu cầu phê duyệt được cho lần gửi vì họ có campaigns:edit mà không có campaigns:send" {
		t.Errorf("impact: %+v", imp)
	}
	if p := vi.Do("DELETE", ws+"/roles/"+editor, nil).MustStatus(t, 409).Problem(t); p["detail"] != "Biên tập viên là vai trò có sẵn nên không xóa được." {
		t.Errorf("built-in delete: %v", p)
	}
	if p := vi.Do("POST", ws+"/roles", map[string]any{"name": " ", "permissions": []string{"campaigns:view"}}).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy đặt tên cho vai trò." {
		t.Errorf("no name: %v", p)
	}

	// API clients get English.
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	en.Do("GET", ws+"/roles", nil).MustStatus(t, 200).JSON(t, &l)
	if p := l.Permissions[0]; p.Group != "Campaigns" || p.Label != "View campaigns" {
		t.Errorf("English catalogue: %+v", p)
	}
	if p := en.Do("DELETE", ws+"/roles/"+editor, nil).MustStatus(t, 409).Problem(t); p["detail"] != "Editor is a built-in role and can't be deleted." {
		t.Errorf("English built-in delete: %v", p)
	}
}
