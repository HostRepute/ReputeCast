package ingest_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Sending providers › Events reads in the request's language: its problem,
// the bounce mailbox's name and the source labels of system mail. Campaign
// names and providers' own words stay as they are; the export stays English.
func TestEventLogInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	en := env.Client()
	en.SignInDemo("linh@acme.io")

	const ws = "/workspaces/acme-newsletter"
	type logT struct {
		Items []struct{ Provider, Channel, Source, Diagnostic string }
	}
	var vl, el logT
	vi.Do("GET", ws+"/provider-events?pageSize=50", nil).MustStatus(t, 200).JSON(t, &vl)
	en.Do("GET", ws+"/provider-events?pageSize=50", nil).MustStatus(t, 200).JSON(t, &el)
	if len(vl.Items) == 0 || len(vl.Items) != len(el.Items) {
		t.Fatalf("items: %d, %d", len(vl.Items), len(el.Items))
	}
	seen := map[string]bool{}
	for i, it := range vl.Items {
		e := el.Items[i]
		switch e.Source {
		case "Double opt-in confirmation":
			seen["opt-in"] = it.Source == "Thư xác nhận hai bước"
		case "API · transactional":
			seen["api"] = it.Source == "API · giao dịch"
		case "Weekly Digest #142":
			seen["campaign"] = it.Source == e.Source
		}
		if e.Channel == "bounce_mailbox" {
			seen["mailbox"] = e.Provider == "SMTP · bounce mailbox" && it.Provider == "SMTP · hộp thư nhận thư trả lại"
		}
		if it.Diagnostic != e.Diagnostic {
			t.Fatalf("a provider's words changed: %q, %q", it.Diagnostic, e.Diagnostic)
		}
	}
	for _, k := range []string{"opt-in", "api", "campaign", "mailbox"} {
		if !seen[k] {
			t.Fatalf("%s: %+v", k, vl.Items)
		}
	}
	if pr := vi.Do("GET", ws+"/provider-events?provider=nope", nil).MustStatus(t, 422).Problem(t); pr["detail"] != "Hãy chọn một nhà cung cấp gửi thư." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
}
