package products_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/payments"
)

// Products' server text reads in the request's language: problems, and the
// dispute notification, which keeps its English in the row and reads in
// each member's own language.
func TestProductsInVietnamese(t *testing.T) {
	s := newShop(t)
	vi := s.env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	if p := vi.Do("GET", ws+"/products/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t); p["detail"] != "Không tìm thấy sản phẩm." {
		t.Fatalf("not found (vi): %v", p)
	}
	if p := s.owner.Do("GET", ws+"/products/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t); p["detail"] != "Product not found." {
		t.Fatalf("not found (en): %v", p)
	}

	// A dispute: the row keeps the English, each member reads their own.
	s.subscribeOrders()
	o, paid := s.buy(s.fieldGuide, buyer{Email: "dee.dispute@example.com"})
	due := time.Date(2030, 3, 14, 15, 0, 0, 0, time.UTC)
	_, ev, err := s.stripe.Dispute(payments.Live, paid.Charge, "fraudulent", due.Unix())
	if err != nil {
		t.Fatal(err)
	}
	if st, out := s.hook(ev); st != "applied" || out != "order_disputed" {
		t.Fatalf("dispute: %s %s", st, out)
	}
	var stored string
	s.scan(`SELECT body FROM notifications WHERE workspace_id = $1 AND kind = 'order_disputed' AND data->>'order_id' = $2 ORDER BY created_at LIMIT 1`,
		[]any{s.ws.ID, o.ID.String()}, &stored)
	if !strings.HasPrefix(stored, "dee.dispute@example.com disputed order") || !strings.Contains(stored, "Respond in Stripe by") ||
		!strings.HasSuffix(stored, "Access to the download was revoked, as the product says.") {
		t.Fatalf("stored: %q", stored)
	}
	var list struct {
		Items []struct{ Title, Body string }
	}
	vi.Do("GET", ws+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	found := false
	for _, it := range list.Items {
		if it.Title == "Tranh chấp được mở" {
			found = true
			if !strings.HasPrefix(it.Body, "dee.dispute@example.com đã mở tranh chấp cho đơn hàng") || !strings.Contains(it.Body, "thg 3, 2030") ||
				!strings.HasSuffix(it.Body, "Quyền tải xuống đã bị thu hồi, theo cài đặt của sản phẩm.") {
				t.Errorf("notification (vi): %q", it.Body)
			}
		}
	}
	if !found {
		t.Fatalf("no dispute notification in Vietnamese: %+v", list.Items)
	}
}
