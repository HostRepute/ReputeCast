package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestShareCampaignReportTool: an agent holding reports:share makes a
// read-only link that shares what it chose and expires when it said; an
// agent without the permission is denied (review AN-14, AN-16).
func TestShareCampaignReportTool(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	agent := func(name string, perms ...string) *mcp.ClientSession {
		t.Helper()
		var a struct{ Secret string }
		c.Do("POST", ws+"/mcp-agents", map[string]any{"name": name, "permissions": perms}).MustStatus(t, 201).JSON(t, &a)
		cs, err := connect(t, env, a.Secret)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		t.Cleanup(func() { _ = cs.Close() })
		return cs
	}
	share := func(cs *mcp.ClientSession) *mcp.CallToolResult {
		t.Helper()
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "share_campaign_report",
			Arguments: map[string]any{"campaign": "October Product Update", "expires_in_days": 7, "include_email": true}})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := share(agent("Sharer", "reports:view", "reports:share", "campaigns:view"))
	if res.IsError {
		t.Fatalf("share: %s", text(res))
	}
	var out struct {
		URL       string     `json:"url"`
		ExpiresAt *time.Time `json:"expires_at"`
		Includes  struct{ Cost, Warmup, Email bool }
	}
	if err := json.Unmarshal([]byte(text(res)), &out); err != nil || !strings.Contains(out.URL, "/r/") || out.ExpiresAt == nil ||
		time.Until(*out.ExpiresAt) < 6*24*time.Hour || time.Until(*out.ExpiresAt) > 7*24*time.Hour+time.Hour || !out.Includes.Email || out.Includes.Cost {
		t.Fatalf("link: %+v %v %s", out, err, text(res))
	}
	// The link opens the report with the email and without the cost.
	var shared struct {
		Includes struct{ Email bool }
		Report   struct {
			ClickMap  struct{ HTML string }
			Providers struct{ Columns []struct{ Cost *float64 } }
		}
	}
	token := out.URL[strings.LastIndex(out.URL, "/")+1:]
	env.Client().Do("GET", "/shared-reports/"+token, nil).MustStatus(t, 200).JSON(t, &shared)
	if !shared.Includes.Email || shared.Report.ClickMap.HTML == "" || len(shared.Report.Providers.Columns) == 0 || shared.Report.Providers.Columns[0].Cost != nil {
		t.Fatalf("shared: %+v", shared)
	}

	// Reading reports isn't sharing them.
	res = share(agent("Reader", "reports:view", "campaigns:view"))
	if !res.IsError || !strings.Contains(text(res), "reports:share") {
		t.Fatalf("an agent without reports:share shared: %s", text(res))
	}
}
