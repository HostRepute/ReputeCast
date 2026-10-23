package mcpserver_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mcpserver"
	"github.com/hostrepute/reputecast/internal/notify"
)

func TestCellsInVietnamese(t *testing.T) {
	cells := mcpserver.CellsIn(i18n.For(i18n.Vietnamese), authz.NewSet(authz.CampaignsPause, authz.ProvidersView))
	if cells[0].Capability != "Đọc báo cáo và liên hệ" || cells[0].Label != "Cho phép · sức khỏe nhà cung cấp" || cells[3].Label != "Chỉ tạm dừng · không bao giờ bắt đầu" || cells[1].Label != "Từ chối" {
		t.Fatalf("cells: %+v", cells)
	}
}

// The agents' activity, recorded while everyone read English, reads in a
// Vietnamese member's language: what a call read, what it found, why it was
// denied and the proposal's notification. What the agent reads stays
// English.
func TestAgentActivityInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var agent struct{ Secret string }
	c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Ops", "permissions": []string{"providers:view", "campaigns:view"}}).MustStatus(t, 201).JSON(t, &agent)
	cs, err := connect(t, env, agent.Secret)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs.Close() }()
	ctx := context.Background()
	if res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_provider_health", Arguments: map[string]any{}}); err != nil || res.IsError {
		t.Fatalf("health: %v %s", err, text(res))
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "pause_campaign", Arguments: map[string]any{"campaign": "October Product Update"}})
	if err != nil || !res.IsError || !strings.Contains(text(res), "needs the campaigns:pause permission") {
		t.Fatalf("denied: %v %s", err, text(res))
	}

	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")
	var actions struct {
		Items []struct{ Tool, Target, Summary, Note, Result string }
	}
	vi.Do("GET", ws+"/mcp-agents/actions", nil).MustStatus(t, 200).JSON(t, &actions)
	// The newest of each tool is this test's; the demo's own stay English.
	var health, denied bool
	seen := map[string]bool{}
	for _, a := range actions.Items {
		if seen[a.Tool] {
			continue
		}
		seen[a.Tool] = true
		switch a.Tool {
		case "get_provider_health":
			health = a.Target == "Mọi nhà cung cấp" && strings.Contains(a.Summary, "ổn định")
			if !health {
				t.Errorf("health: %+v", a)
			}
		case "pause_campaign":
			denied = a.Result == "denied" && a.Note == "cần quyền campaigns:pause · đã ghi đề xuất cho Linh Nguyen"
			if !denied {
				t.Errorf("denied: %+v", a)
			}
		}
	}
	if !health || !denied {
		t.Fatalf("actions: %+v", actions.Items)
	}
	var ov struct {
		Capabilities []string
		Resources    []string
	}
	vi.Do("GET", ws+"/mcp", nil).MustStatus(t, 200).JSON(t, &ov)
	if len(ov.Capabilities) != 5 || ov.Capabilities[3] != "Lên lịch hoặc gửi" || ov.Resources[2] != "nhật ký kiểm tra" {
		t.Errorf("overview: %+v", ov)
	}

	var title, summary, body string
	var data []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT title, summary, body, data FROM notifications WHERE kind = 'agent_proposal' ORDER BY created_at DESC LIMIT 1`).Scan(&title, &summary, &body, &data); err != nil {
		t.Fatal(err)
	}
	if title != "Ops asked to pause October Product Update" || !strings.HasPrefix(body, "That needs campaigns:pause, which the agent doesn't hold.") {
		t.Errorf("English: %q · %q", title, body)
	}
	if vt, _, vb := notify.Localize(i18n.For(i18n.Vietnamese), title, summary, body, data); vt != "Ops đề xuất tạm dừng October Product Update" || !strings.HasPrefix(vb, "Việc này cần quyền campaigns:pause, mà tác tử không có.") {
		t.Errorf("Vietnamese: %q · %q", vt, vb)
	}
}
