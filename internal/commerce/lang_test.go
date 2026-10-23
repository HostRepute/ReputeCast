package commerce_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// A store's server text reads in each reader's language: the notification
// a missing webhook sends keeps its English in the row and reads in each
// member's language, the composer's check in the member's, and a block's
// words in the email's (the workspace's default language).
func TestStoresInVietnamese(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	ctx := context.Background()
	vi := s.env.Client()
	vi.SignIn("marcus@acme.io", "")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	s.fake.DeleteWebhook("checkouts/update")
	s.clk.add(time.Minute)
	if err := s.svc.Health(ctx); err != nil {
		t.Fatal(err)
	}
	if body := one[string](t, s, `SELECT body FROM notifications WHERE kind = 'store_problem' ORDER BY created_at DESC LIMIT 1`); !strings.HasPrefix(body, "It was deleted in Shopify on Oct 15, 2026 at") {
		t.Fatalf("stored: %q", body)
	}
	var list struct {
		Items []struct{ Title, Body string }
	}
	vi.Do("GET", s.path+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) == 0 || list.Items[0].Title != "Thiếu webhook checkouts/update trong Acme Merch" ||
		!strings.HasPrefix(list.Items[0].Body, "Webhook này đã bị xóa trên Shopify lúc ") || !strings.Contains(list.Items[0].Body, "15 thg 10, 2026") {
		t.Fatalf("notification (vi): %+v", list.Items)
	}

	// The composer's check, in the member's language.
	block := commerce.CommerceBlock{Store: s.id, Mode: "chosen", Products: []string{"8841203", "8841215"}, SoldOut: "label"}
	var r commerce.CommerceBlockRender
	s.c.Do("POST", s.path+"/commerce/blocks/render", map[string]any{"block": block}).MustStatus(t, 200).JSON(t, &r)
	s.deliver(s.fake.Deliver("products/delete", map[string]any{"id": 8841215}))
	issues := s.svc.RemovedProducts(i18n.WithLang(ctx, i18n.For(i18n.Vietnamese)), contentcheck.Input{WorkspaceID: s.ws.ID, HTML: r.HTML})
	if len(issues) != 1 || issues[0].Title != "Feature Flag sticker pack không còn trong Acme Merch" || issues[0].Tag != "Không còn trong cửa hàng" {
		t.Fatalf("check (vi): %+v", issues)
	}

	// A block's words follow the workspace's default language: English
	// here, Vietnamese once the workspace writes in it.
	if !strings.Contains(r.HTML, "Shop now") {
		t.Fatalf("English block: %s", r.HTML)
	}
	s.exec(`INSERT INTO workspace_languages (workspace_id, code, is_default, is_fallback) VALUES ($1, 'vi', false, false)`, s.ws.ID)
	s.exec(`UPDATE workspace_languages SET is_default = false WHERE workspace_id = $1`, s.ws.ID)
	s.exec(`UPDATE workspace_languages SET is_default = true WHERE workspace_id = $1 AND code = 'vi'`, s.ws.ID)
	vi.Do("POST", s.path+"/commerce/blocks/render", map[string]any{"block": block}).MustStatus(t, 200).JSON(t, &r)
	if !strings.Contains(r.HTML, "Mua ngay") || strings.Contains(r.HTML, "Shop now") {
		t.Fatalf("Vietnamese block: %s", r.HTML)
	}
}
