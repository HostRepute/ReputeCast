package webhooks

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The delivery log's fixed sentences read in the member's language; the
// endpoint's and the network's own words stay as they are.
func TestDeliveryErrorsInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	demo := "Demo endpoint: ReputeCast never sends demo data. Point it at your own URL to receive events."
	if mDemoEndpoint.Text(i18n.Default, nil) != demo || mInvalidURL.Text(i18n.Default, nil) != "Invalid URL" {
		t.Fatal("the stored English changed")
	}
	if got := deliveryErrorIn(vi, demo); got != "Endpoint mẫu: ReputeCast không bao giờ gửi dữ liệu mẫu. Hãy trỏ endpoint đến URL của bạn để nhận sự kiện." {
		t.Errorf("demo: %q", got)
	}
	if got := deliveryErrorIn(vi, "Invalid URL"); got != "URL không hợp lệ" {
		t.Errorf("invalid URL: %q", got)
	}
	if got := deliveryErrorIn(vi, "HTTP 500 Internal Server Error"); got != "HTTP 500 Internal Server Error" {
		t.Errorf("an endpoint's answer: %q", got)
	}
	if got := deliveryErrorIn(i18n.Default, demo); got != demo {
		t.Errorf("English: %q", got)
	}
}
