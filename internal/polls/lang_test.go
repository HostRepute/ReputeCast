package polls_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/polls"
)

// What members read about polls and surveys follows the request's
// language: a problem, the report's breakdown labels and a new survey's
// name. API clients that don't ask get English.
func TestPollsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	en := env.Client()
	en.SignInDemo("linh@acme.io")

	// The poll report's breakdown by engagement tier.
	campaign := one[string](t, env, `SELECT id::text FROM campaigns WHERE name = 'Weekly Digest #141'`)
	tiers := func(c *apptest.Client) map[string]string {
		var list polls.PollResultsList
		c.Do(http.MethodGet, ws+"/campaigns/"+campaign+"/polls", nil).MustStatus(t, 200).JSON(t, &list)
		out := map[string]string{}
		for _, r := range list.Items[0].Breakdown.Tiers {
			out[r.Key] = r.Label
		}
		return out
	}
	if got := tiers(vi); got["active"] != "Tích cực" || got["at_risk"] != "Có nguy cơ" {
		t.Fatalf("Vietnamese tiers: %v", got)
	}
	if got := tiers(en); got["active"] != "Active" || got["at_risk"] != "At risk" {
		t.Fatalf("English tiers: %v", got)
	}

	// A new survey's name, and a problem.
	var s polls.Survey
	vi.Do(http.MethodPost, ws+"/surveys", map[string]any{}).MustStatus(t, 201).JSON(t, &s)
	if s.Name != "Khảo sát chưa đặt tên" {
		t.Fatalf("new survey: %q", s.Name)
	}
	var list polls.SurveyList
	vi.Do(http.MethodGet, ws+"/surveys", nil).MustStatus(t, 200).JSON(t, &list)
	taken := ""
	for _, x := range list.Items {
		if x.ID != s.ID {
			taken = x.Slug
		}
	}
	if pr := vi.Do(http.MethodPatch, ws+"/surveys/"+s.ID, map[string]any{"slug": taken}).MustStatus(t, 422).Problem(t); pr["detail"] != "Một khảo sát khác của không gian làm việc này đã dùng địa chỉ đó. Hãy chọn địa chỉ khác." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	if pr := en.Do(http.MethodPatch, ws+"/surveys/"+s.ID, map[string]any{"slug": taken}).MustStatus(t, 422).Problem(t); pr["detail"] != "Another survey of this workspace has that address. Pick another." {
		t.Fatalf("English problem: %v", pr)
	}
}
