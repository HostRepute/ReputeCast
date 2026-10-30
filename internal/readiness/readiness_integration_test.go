package readiness_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mcpserver"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
)

const ws = "/workspaces/acme-newsletter"

func campaignID(t *testing.T, c *apptest.Client, name string) string {
	t.Helper()
	var list struct {
		Items []struct{ ID, Name string }
	}
	c.Do("GET", ws+"/campaigns?pageSize=100&q="+strings.ReplaceAll(name, " ", "%20"), nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.Name == name {
			return it.ID
		}
	}
	t.Fatalf("campaign %q not found", name)
	return ""
}

type reviewRun struct {
	Run struct {
		ID     string
		Checks []readiness.ReadinessCheck
		Groups []readiness.ReadinessGroup
	}
}

type agentBearer struct{ token string }

func (b agentBearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// A run a member reading English starts reads in Vietnamese for a
// Vietnamese reader, in English as it was for English readers and API
// clients, and stays English for MCP agents; runs stored before their texts
// were kept read in English for everyone.
func TestRunsReadInEachReadersLanguage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	id := campaignID(t, en, "Weekly Digest #143")

	var run readiness.ReadinessRun
	readinesstest.Post(t, env, en, ws+"/campaigns/"+id+"/readiness").MustStatus(t, 200).JSON(t, &run)
	if len(run.Checks) == 0 || strings.Contains(run.Checks[0].Title, "Uy tín") {
		t.Fatalf("English run: %+v", run.Checks)
	}

	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	var viReview, enReview reviewRun
	vi.Do("GET", ws+"/campaigns/"+id+"/review", nil).MustStatus(t, 200).JSON(t, &viReview)
	en.Do("GET", ws+"/campaigns/"+id+"/review", nil).MustStatus(t, 200).JSON(t, &enReview)
	if viReview.Run.ID != run.ID || len(viReview.Run.Checks) != len(run.Checks) {
		t.Fatalf("the same run: %s %s", viReview.Run.ID, run.ID)
	}
	byKey := map[string]readiness.ReadinessCheck{}
	for _, c := range viReview.Run.Checks {
		byKey[c.Key] = c
	}
	for i, c := range run.Checks {
		// English readers read exactly what the run stored.
		if e := enReview.Run.Checks[i]; e.Title != c.Title || e.Detail != c.Detail || e.Evidence != c.Evidence || e.Ack != c.Ack {
			t.Errorf("%s in English: %+v, stored %+v", c.Key, e, c)
		}
	}
	if c := byKey["content.plain_text"]; !strings.HasPrefix(c.Title, "Đã tạo phiên bản văn bản thuần") {
		t.Errorf("plain text in Vietnamese: %q", c.Title)
	}
	if c := byKey["audience.role"]; !strings.Contains(c.Title, "địa chỉ chức danh") {
		t.Errorf("role addresses in Vietnamese: %q", c.Title)
	}
	for _, g := range viReview.Run.Groups {
		if g.Key == "reputation" && g.Title != "Uy tín người gửi" {
			t.Errorf("group in Vietnamese: %+v", g)
		}
	}
	if strings.Contains(string(vi.Do("GET", ws+"/campaigns/"+id+"/review", nil).Body), `"i18n"`) {
		t.Errorf("the review carries kept messages")
	}
	// A run's own endpoint reads in the request's language too.
	var viRun readiness.ReadinessRun
	readinesstest.Post(t, env, vi, ws+"/campaigns/"+id+"/readiness").MustStatus(t, 200).JSON(t, &viRun)
	if len(viRun.Groups) == 0 || viRun.Groups[0].Title == run.Groups[0].Title {
		t.Errorf("run in Vietnamese: %+v", viRun.Groups)
	}

	// The approval page reads the request's checks and its providers'
	// status in the reader's language, from the snapshot's English.
	var inbox struct{ Items []struct{ ID, Name string } }
	en.Do("GET", ws+"/approvals?status=pending", nil).MustStatus(t, 200).JSON(t, &inbox)
	var reqID string
	for _, it := range inbox.Items {
		if it.Name == "November Product Update" {
			reqID = it.ID
		}
	}
	if reqID == "" {
		t.Fatalf("no request for November: %+v", inbox.Items)
	}
	type approval struct {
		Readiness struct{ Issues []readiness.ReadinessCheck }
		Route     struct {
			SplitLabel string
			Shares     []struct{ Status, StatusLabel string }
		}
	}
	var viReq, enReq approval
	vi.Do("GET", ws+"/approvals/"+reqID, nil).MustStatus(t, 200).JSON(t, &viReq)
	en.Do("GET", ws+"/approvals/"+reqID, nil).MustStatus(t, 200).JSON(t, &enReq)
	if len(viReq.Readiness.Issues) == 0 || len(viReq.Readiness.Issues) != len(enReq.Readiness.Issues) {
		t.Fatalf("issues: %+v %+v", viReq.Readiness.Issues, enReq.Readiness.Issues)
	}
	for i, c := range enReq.Readiness.Issues {
		if v := viReq.Readiness.Issues[i]; v.Key != c.Key || v.Title == c.Title {
			t.Errorf("issue %s in Vietnamese: %q, in English %q", c.Key, v.Title, c.Title)
		}
	}
	if enReq.Route.SplitLabel != "Broadcasts 90 / 10" || viReq.Route.SplitLabel != "Gửi hàng loạt 90 / 10" {
		t.Errorf("split: %q, in Vietnamese %q", enReq.Route.SplitLabel, viReq.Route.SplitLabel)
	}
	warming := 0
	for i, s := range enReq.Route.Shares {
		if s.Status == "warming_up" {
			warming++
			if v := viReq.Route.Shares[i].StatusLabel; !strings.HasPrefix(s.StatusLabel, "Warming up · day ") || !strings.HasPrefix(v, "Đang khởi động · ngày ") {
				t.Errorf("warming up: %q, in Vietnamese %q", s.StatusLabel, v)
			}
		}
	}
	if warming == 0 {
		t.Errorf("no warming-up provider: %+v", enReq.Route.Shares)
	}

	// MCP agents read the stored run in English, without kept messages.
	var agent struct{ Secret string }
	en.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Reader", "permissions": []string{"campaigns:view"}}).MustStatus(t, 201).JSON(t, &agent)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: env.Server.URL + mcpserver.Path,
		HTTPClient: &http.Client{Transport: agentBearer{agent.Secret}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs.Close() }()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_send_readiness", Arguments: map[string]any{"campaign": "Weekly Digest #143"}})
	if err != nil || res.IsError {
		t.Fatalf("tool: %v", err)
	}
	var out strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out.WriteString(tc.Text)
		}
	}
	if strings.Contains(out.String(), `i18n`) || strings.Contains(out.String(), "Uy tín") || !strings.Contains(out.String(), "Plain-text version generated") {
		t.Errorf("MCP view: %s", out.String())
	}

	// A run stored before its texts were kept reads in English.
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE readiness_runs SET checks = jsonb_set(checks, '{checks}', (SELECT jsonb_agg(e - 'i18n') FROM jsonb_array_elements(checks->'checks') e))
		WHERE campaign_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	vi.Do("GET", ws+"/campaigns/"+id+"/review", nil).MustStatus(t, 200).JSON(t, &viReview)
	for _, c := range viReview.Run.Checks {
		if c.Key == "content.plain_text" && !strings.HasPrefix(c.Title, "Plain-text version generated") {
			t.Errorf("old run in Vietnamese: %q", c.Title)
		}
	}
	var stored json.RawMessage
	if err := env.App.Pool.QueryRow(ctx, `SELECT checks FROM readiness_runs WHERE campaign_id = $1 ORDER BY ran_at DESC LIMIT 1`, id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), `"title": "Đã tạo`) {
		t.Errorf("a Vietnamese request stored Vietnamese: %s", stored)
	}
}
