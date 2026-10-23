package smtprelay_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
)

// Instance admin › SMTP relay reads in the request's language: the problems
// of saving it, and the status checks, which keep their English and their
// messages for later readers. The instance admins' notification keeps its
// messages too. The relay's SMTP replies stay English.
func TestSMTPRelayInVietnamese(t *testing.T) {
	e := newRelayEnv(t)
	vi := e.env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	same := map[string]any{"enabled": true, "hostname": "localhost", "starttlsPort": e.p587, "tlsPort": e.p587, "certSource": "acme"}
	if pr := vi.Do("PUT", "/admin/smtp-relay", same).MustStatus(t, 422).Problem(t); pr["detail"] != "Cổng STARTTLS và cổng TLS phải khác nhau." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	if pr := e.owner.Do("PUT", "/admin/smtp-relay", same).MustStatus(t, 422).Problem(t); pr["detail"] != "The STARTTLS and TLS ports must differ." {
		t.Fatalf("English problem: %v", pr)
	}

	// The relay stops answering; the check runs from an English request.
	e.stopRelay()
	type checks struct {
		Status struct {
			Checks []struct {
				Key, Detail string
				OK          bool
			}
		}
	}
	var en, read checks
	e.owner.Do("POST", "/admin/smtp-relay/check", nil).MustStatus(t, 200).JSON(t, &en)
	port := strconv.Itoa(e.p587)
	var english string
	for _, c := range en.Status.Checks {
		if c.Key == "starttls" {
			english = c.Detail
		}
	}
	if !strings.HasPrefix(english, "Port "+port+" isn't reachable at localhost: ") {
		t.Fatalf("English check: %+v", en.Status.Checks)
	}
	vi.Do("GET", "/admin/smtp-relay", nil).MustStatus(t, 200).JSON(t, &read)
	for _, c := range read.Status.Checks {
		switch c.Key {
		case "starttls":
			if !strings.HasPrefix(c.Detail, "Không truy cập được cổng "+port+" tại localhost: ") {
				t.Fatalf("Vietnamese check: %q", c.Detail)
			}
		case "dns":
			if !strings.HasPrefix(c.Detail, "DNS · localhost trỏ về ") {
				t.Fatalf("Vietnamese DNS check: %q", c.Detail)
			}
		}
	}

	// The notification keeps its English in the row and its messages beside it.
	var title, body string
	var data []byte
	if err := e.env.App.Pool.QueryRow(context.Background(), `SELECT title, body, data FROM notifications WHERE kind = 'smtp_relay_unreachable' LIMIT 1`).Scan(&title, &body, &data); err != nil {
		t.Fatal(err)
	}
	if title != "SMTP relay unreachable" || !strings.HasPrefix(body, english+". Apps that send by SMTP can't connect") {
		t.Fatalf("stored: %q, %q", title, body)
	}
	vt, _, vb := notify.Localize(i18n.For(i18n.Vietnamese), title, "", body, data)
	if vt != "Không truy cập được SMTP relay" || !strings.HasPrefix(vb, "Không truy cập được cổng "+port+" tại localhost: ") || !strings.HasSuffix(vb, "HTTP API vẫn hoạt động.") {
		t.Fatalf("Vietnamese notification: %q, %q", vt, vb)
	}
}
