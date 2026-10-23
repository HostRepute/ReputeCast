package apikeys_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Settings › Keys & MCP reads in the app's language: the permission
// picker, why a permission is locked and the problems. The audit entry of a
// key created in English keeps its summary's messages for other readers.
func TestKeysInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")

	var agents struct {
		Permissions []struct{ Key, Group, Label string }
		Locked      []struct{ Key, Reason string }
	}
	vi.Do("GET", ws+"/mcp-agents", nil).MustStatus(t, 200).JSON(t, &agents)
	if p := agents.Permissions[0]; p.Group != "Chiến dịch" || p.Label != "Xem chiến dịch" {
		t.Errorf("catalogue: %+v", p)
	}
	reasons := map[string]string{}
	for _, l := range agents.Locked {
		reasons[l.Key] = l.Reason
	}
	if reasons["ai:use"] != "Tính năng AI dành cho con người. Tác tử dùng mô hình của riêng chúng." ||
		reasons["subscribers:verify"] != "Không dành cho tác tử: quyền này gửi địa chỉ ra ngoài ReputeCast" {
		t.Errorf("locked: %v", reasons)
	}
	if p := vi.Do("POST", ws+"/api-keys", map[string]any{"name": " ", "permissions": []string{"campaigns:view"}}).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy đặt tên cho khóa." {
		t.Errorf("no name: %v", p)
	}

	en := env.Client()
	en.SignInDemo("linh@acme.io")
	var made struct{ Key struct{ ID string } }
	en.Do("POST", ws+"/api-keys", map[string]any{"name": "Reporting", "permissions": []string{"reports:view"}}).MustStatus(t, 201).JSON(t, &made)
	var raw []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT metadata FROM audit_events WHERE action = 'apikey.created' AND target_id = $1`, made.Key.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var meta struct{ Summary string }
	_ = json.Unmarshal(raw, &meta)
	if meta.Summary != "reports:view · all lists · no expiry" {
		t.Errorf("English summary: %q", meta.Summary)
	}
	if got := audit.KeptIn(raw).SummaryIn(i18n.For(i18n.Vietnamese), meta.Summary); got != "reports:view · mọi danh sách · không hết hạn" {
		t.Errorf("Vietnamese summary: %q", got)
	}
}
