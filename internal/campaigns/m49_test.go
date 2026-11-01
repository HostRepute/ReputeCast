package campaigns_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Text, push and WhatsApp campaigns are managed like email ones (review
// 2026-11 CH-1): a draft is deleted and restored, a sent one archived and
// unarchived, and bulk tagging takes them all.
func TestChannelCampaignsDeleteArchiveAndTag(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	pick := func(channel, statuses string) string {
		t.Helper()
		var id string
		err := env.App.Pool.QueryRow(ctx, `SELECT c.id::text FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id
			WHERE w.slug = 'acme-newsletter' AND c.channel = $1 AND c.status = ANY(string_to_array($2, ',')) AND c.deleted_at IS NULL
			ORDER BY c.created_at LIMIT 1`, channel, statuses).Scan(&id)
		if err != nil {
			t.Skipf("the demo has no %s campaign in %s: %v", channel, statuses, err)
		}
		return id
	}
	var ids []string
	for _, ch := range []string{"sms", "push"} {
		sent := pick(ch, "sent")
		c.Do(http.MethodPost, ws+"/campaigns/"+sent+"/archive", nil).MustStatus(t, http.StatusNoContent)
		var l listed
		c.Do(http.MethodGet, ws+"/campaigns?status=archived&pageSize=100", nil).MustStatus(t, 200).JSON(t, &l)
		found := false
		for _, it := range l.Items {
			found = found || it.ID == sent
		}
		if !found {
			t.Fatalf("%s: the archived campaign isn't under Archived", ch)
		}
		c.Do(http.MethodPost, ws+"/campaigns/"+sent+"/unarchive", nil).MustStatus(t, http.StatusNoContent)
		ids = append(ids, sent)
	}
	for _, ch := range []string{"sms", "whatsapp"} {
		draft := pick(ch, "draft")
		c.Do(http.MethodDelete, ws+"/campaigns/"+draft, nil).MustStatus(t, http.StatusNoContent)
		c.Do(http.MethodPost, ws+"/campaigns/"+draft+"/restore", nil).MustStatus(t, http.StatusNoContent)
		ids = append(ids, draft)
	}

	var res struct{ Done int }
	c.Do(http.MethodPost, ws+"/campaigns/bulk", map[string]any{"ids": ids, "action": "tag", "tag": "channels"}).MustStatus(t, 200).JSON(t, &res)
	if res.Done != len(ids) {
		t.Fatalf("tagged %d of %d", res.Done, len(ids))
	}
}
