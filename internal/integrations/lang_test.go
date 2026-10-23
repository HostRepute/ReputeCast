package integrations_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The hub's own words read in the request's language: a problem, and the
// attention notification, which keeps its messages so every member reads
// it in theirs while its row keeps the English. API clients that don't
// ask get English.
func TestIntegrationsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	en := env.Client()
	en.SignInDemo("linh@acme.io")

	zapier := connectionID(t, en, "zapier")
	if pr := vi.Do("PUT", ws+"/integrations/connections/"+zapier+"/address", map[string]any{"address": "https://x.example"}).MustStatus(t, 422).Problem(t); pr["detail"] != "Zapier không có địa chỉ để đặt." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	if pr := en.Do("PUT", ws+"/integrations/connections/"+zapier+"/address", map[string]any{"address": "https://x.example"}).MustStatus(t, 422).Problem(t); pr["detail"] != "Zapier has no address to set." {
		t.Fatalf("English problem: %v", pr)
	}

	// blog.acme.io goes quiet: the members who manage integrations hear.
	exec(t, env, `UPDATE wordpress_sites SET last_report_at = now() - interval '30 hours' WHERE address = 'blog.acme.io'`)
	if err := hubService(env).CheckHealth(ctx); err != nil {
		t.Fatal(err)
	}
	var stored int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'integration_attention' AND title = 'Integration needs attention'
		AND body LIKE 'WordPress: blog.acme.io hasn''t reported in since%'`).Scan(&stored)
	if stored == 0 {
		t.Fatal("the row doesn't keep its English")
	}
	find := func(c *apptest.Client, prefix string) map[string]any {
		var list map[string]any
		c.Do("GET", ws+"/notifications?limit=100", nil).MustStatus(t, 200).JSON(t, &list)
		for _, it := range list["items"].([]any) {
			n := it.(map[string]any)
			if strings.HasPrefix(n["body"].(string), prefix) {
				return n
			}
		}
		t.Fatalf("no notification starting %q: %v", prefix, list["items"])
		return nil
	}
	if n := find(vi, "WordPress: blog.acme.io không gửi báo cáo từ "); n["title"] != "Tích hợp cần chú ý" || n["categoryLabel"] != "Tích hợp" {
		t.Fatalf("Vietnamese notification: %v", n)
	}
	if n := find(en, "WordPress: blog.acme.io hasn't reported in since "); n["title"] != "Integration needs attention" {
		t.Fatalf("English notification: %v", n)
	}
}
