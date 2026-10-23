package campaigns_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/campaigns"
)

// The composer's own words follow the request's language: a new draft's
// name, a duplicate's, a problem, and the labels campaigns writes on
// versions, which are stored in English and read in each reader's
// language. API clients that don't ask get English.
func TestComposerInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	const ws = "/workspaces/acme-newsletter"

	var draft, dup campaigns.ComposerCampaign
	vi.Do(http.MethodPost, ws+"/campaigns", map[string]any{}).MustStatus(t, 201).JSON(t, &draft)
	if draft.Name != "Chiến dịch chưa đặt tên" {
		t.Fatalf("new draft: %q", draft.Name)
	}
	en.Do(http.MethodPatch, ws+"/campaigns/"+draft.ID, map[string]any{"name": "Spring notes"}).MustStatus(t, 200)
	vi.Do(http.MethodPost, ws+"/campaigns/"+draft.ID+"/duplicate", nil).MustStatus(t, 201).JSON(t, &dup)
	if dup.Name != "Bản sao của Spring notes" {
		t.Fatalf("duplicate: %q", dup.Name)
	}

	label := func(c *apptest.Client) string {
		var v campaigns.ComposerVersionList
		c.Do(http.MethodGet, ws+"/campaigns/"+dup.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &v)
		return v.Items[len(v.Items)-1].Label
	}
	if got := label(en); got != "copied from “Spring notes”" {
		t.Fatalf("English version label: %q", got)
	}
	if got := label(vi); got != "sao chép từ “Spring notes”" {
		t.Fatalf("Vietnamese version label: %q", got)
	}

	unknown := "0192f2c4-7c3a-7b1e-9d4a-3b5c6d7e8f90"
	if pr := vi.Do(http.MethodPatch, ws+"/campaigns/"+draft.ID, map[string]any{"senderId": unknown}).MustStatus(t, 422).Problem(t); pr["detail"] != "Người gửi đó không có trong không gian làm việc này." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	if pr := en.Do(http.MethodPatch, ws+"/campaigns/"+draft.ID, map[string]any{"senderId": unknown}).MustStatus(t, 422).Problem(t); pr["detail"] != "That sender doesn't exist in this workspace." {
		t.Fatalf("English problem: %v", pr)
	}
}
