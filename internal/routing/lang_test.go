package routing_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Routing reads in the request's language: its issues, the problems of
// saving it and of the route tester. API clients that send no
// Accept-Language keep getting English.
func TestRoutingInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	var c config
	en.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &c)
	ids := map[string]string{}
	for _, p := range c.Providers {
		ids[p.Adapter] = p.ID
	}

	bad := input(c)
	bad[0].Splits[1]["weight"] = 20
	pr := vi.Do("PUT", ws+"/routing", map[string]any{"streams": bad}).MustStatus(t, 422).Problem(t)
	if !strings.HasPrefix(pr["detail"].(string), "Phần chia tỷ trọng của Gửi hàng loạt cộng lại là ") || !strings.HasSuffix(pr["detail"].(string), "; tổng phải là 100%.") {
		t.Fatalf("Vietnamese weights: %v", pr)
	}
	pr = en.Do("PUT", ws+"/routing", map[string]any{"streams": bad}).MustStatus(t, 422).Problem(t)
	if !strings.HasPrefix(pr["detail"].(string), "The Broadcasts split adds up to ") || !strings.HasSuffix(pr["detail"].(string), "; it must add up to 100%.") {
		t.Fatalf("English weights: %v", pr)
	}

	bulk := input(c)
	bulk[0].Fallback = ids["postmark"]
	pr = vi.Do("PUT", ws+"/routing", map[string]any{"streams": bulk}).MustStatus(t, 409).Problem(t)
	if pr["detail"] != "Postmark chỉ cho phép thư hàng loạt trên luồng Broadcast, mà tuyến này dùng luồng giao dịch của nhà cung cấp đó. Hãy chọn nhà cung cấp khác, hoặc giữ nhà cung cấp này và xác nhận các điều khoản." {
		t.Fatalf("Vietnamese bulk terms: %v", pr)
	}
	pr = en.Do("PUT", ws+"/routing", map[string]any{"streams": bulk}).MustStatus(t, 409).Problem(t)
	if pr["detail"] != "Postmark allows bulk mail only on a Broadcast stream, and this route uses its transactional stream. Choose another provider, or keep it and acknowledge the terms." {
		t.Fatalf("English bulk terms: %v", pr)
	}

	if pr := vi.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": "not-an-address"}).MustStatus(t, 422).Problem(t); pr["detail"] != "Hãy nhập một địa chỉ email, ví dụ name@example.com." {
		t.Fatalf("Vietnamese tester: %v", pr)
	}
}
