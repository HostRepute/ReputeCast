package api

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Every kind of thing NotFound names has a whole sentence of its own, and
// LocalizeProblem writes it in the reader's language.
func TestNotFoundMessages(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	n := 0
	for _, k := range msgs.Keys() {
		if !strings.HasPrefix(k, "notFound.") {
			continue
		}
		n++
		src := msgs.M(k).Source()
		thing, ok := strings.CutSuffix(src, " not found.")
		if !ok {
			t.Errorf("%s: %q isn't \"<thing> not found.\"", k, src)
			continue
		}
		p := NotFound(thing)
		if p.Detail != src {
			t.Errorf("NotFound(%q) = %q, want %q", thing, p.Detail, src)
		}
		d, ok := localizedDetail(p, vi)
		if !ok || d == src || !strings.HasPrefix(d, "Không tìm thấy ") {
			t.Errorf("%s in Vietnamese: %q, %v", k, d, ok)
		}
	}
	if n < 100 {
		t.Fatalf("only %d not-found messages", n)
	}
	if d, _ := localizedDetail(NotFound("Campaign"), vi); d != "Không tìm thấy chiến dịch." {
		t.Errorf("campaign: %q", d)
	}
	// A thing without a message keeps its English.
	if _, ok := localizedDetail(NotFound("Gizmo"), vi); ok {
		t.Error("an unknown thing was translated")
	}
}

// The problems the middleware writes keep their English, dates in the
// standard format included, and read in Vietnamese with the region's dates.
func TestMiddlewareProblems(t *testing.T) {
	at := time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)
	args := i18n.Args{"date": at, "integration": "Zapier"}
	if got := mIntegrationKeyExpired.Text(i18n.Default, args); got != "This key expired on Oct 22, 2026. Reconnect Zapier in Settings › Integrations to get a new key." {
		t.Errorf("English: %q", got)
	}
	if got := mKeyExpired.Text(i18n.Default, args); got != "This API key expired on Oct 22, 2026. Rotate it in Settings › Keys & MCP." {
		t.Errorf("English: %q", got)
	}
	if got := mIntegrationKeyExpired.Text(i18n.For(i18n.Vietnamese), args); got != "Khóa này đã hết hạn ngày 22 thg 10, 2026. Hãy kết nối lại Zapier trong Cài đặt › Tích hợp để nhận khóa mới." {
		t.Errorf("Vietnamese: %q", got)
	}
}
