package smtp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/smtp"
)

const secret = "whsec-test-smtp"

// Mailpit from docker compose; override with MAILPIT_SMTP / MAILPIT_API.
func mailpit() (smtpAddr, api string) {
	smtpAddr, api = os.Getenv("MAILPIT_SMTP"), os.Getenv("MAILPIT_API")
	if smtpAddr == "" {
		smtpAddr = "localhost:1025"
	}
	if api == "" {
		api = "http://localhost:8025"
	}
	return smtpAddr, api
}

func mailpitReady(t *testing.T) (bool, string) {
	addr, _ := mailpit()
	c, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return false, "Mailpit is not running on " + addr + " (docker compose up -d)"
	}
	_ = c.Close()
	return true, ""
}

func newProvider(t *testing.T, _ *contracttest.Env) providers.Provider {
	t.Helper()
	addr, _ := mailpit()
	host, port, _ := net.SplitHostPort(addr)
	settings, _ := json.Marshal(map[string]any{"host": host, "port": atoi(port), "tls": "none", "return_path": "bounces@news.acme.io"})
	p, err := smtp.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{}`), WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

func report(file string) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
		body, err := os.ReadFile("dsn/testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		r, _ := http.NewRequest(http.MethodPost, "https://reputecast.acme.io/hooks/smtp/token", bytes.NewReader(body))
		r.Header.Set("Content-Type", "message/rfc822")
		r.Header.Set(smtp.SignatureHeader, smtp.Sign(secret, body))
		return r, body
	}
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Adapter:          smtp.Adapter,
		New:              newProvider,
		SendReady:        mailpitReady,
		SkipFailureModes: true,
		Delivered: func(t *testing.T, _ *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			_, api := mailpit()
			for i, m := range msgs {
				headers := mailpitHeaders(t, api, m.Subject)
				if got := first(headers, "X-Reputecast-Message"); got != m.ID.String() {
					t.Fatalf("message %d: X-ReputeCast-Message %q, want %s", i, got, m.ID)
				}
				if got := first(headers, "List-Unsubscribe-Post"); got != "List-Unsubscribe=One-Click" {
					t.Fatalf("message %d: List-Unsubscribe-Post %q", i, got)
				}
				if got := first(headers, "Return-Path"); !strings.Contains(got, "bounces@news.acme.io") {
					t.Fatalf("message %d: Return-Path %q, want the bounce address", i, got)
				}
				if res[i].ProviderMessageID != first(headers, "Message-Id") {
					t.Fatalf("message %d: provider id %q, Message-ID %q", i, res[i].ProviderMessageID, first(headers, "Message-Id"))
				}
			}
		},
		Webhooks: []contracttest.Webhook{
			{Name: "DSN", Request: report("postfix-hard.eml"), Want: []providers.Event{{
				Type: providers.EventHardBounce, Recipient: "t.lee@example.org", ProviderMessageID: "<0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e4f@news.acme.io>",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e4f"), OriginalType: "DSN · 5.1.1", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown",
			}}},
			{Name: "ARF", Request: report("arf.eml"), Want: []providers.Event{{
				Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "<0199d5a8-2b6e-7c41-9a0e-000000000001@news.acme.io>",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-000000000001"), OriginalType: "ARF · abuse",
			}}},
			{Name: "DelayedHeuristic", Request: report("exim-delayed.eml"), Want: []providers.Event{{
				Type: providers.EventDeferred, Recipient: "b.h@yahoo.com", ProviderMessageID: "<0199d5a8-2b6e-7c41-9a0e-000000000002@news.acme.io>",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-000000000002"), OriginalType: "DSN · 4.2.2", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full",
			}}},
		},
	})
}

// mailpitHeaders finds the message with subject in Mailpit and returns its headers.
func mailpitHeaders(t *testing.T, api, subject string) map[string][]string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		q := url.Values{"query": {`subject:"` + subject + `"`}}
		resp, err := http.Get(api + "/api/v1/search?" + q.Encode())
		if err != nil {
			t.Fatal(err)
		}
		var res struct {
			Messages []struct{ ID string }
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		_ = resp.Body.Close()
		if len(res.Messages) > 0 {
			hr, err := http.Get(api + "/api/v1/message/" + res.Messages[0].ID + "/headers")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = hr.Body.Close() }()
			h := map[string][]string{}
			_ = json.NewDecoder(hr.Body).Decode(&h)
			return h
		}
		if time.Now().After(deadline) {
			t.Fatalf("message %q never reached Mailpit", subject)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func first(h map[string][]string, k string) string {
	for name, v := range h {
		if strings.EqualFold(name, k) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func TestDialFailures(t *testing.T) {
	// Nothing listens on this port: the call fails as an outage.
	settings, _ := json.Marshal(map[string]any{"host": "127.0.0.1", "port": 1, "tls": "none"})
	p, err := smtp.New(providers.Config{Settings: settings})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = p.Send(ctx, []providers.Message{contracttest.Message(0)})
	if !errors.Is(err, providers.ErrUnavailable) {
		t.Fatalf("connection refused must be an outage: %v", err)
	}
}

func TestNotAReport(t *testing.T) {
	p := newProvider(t, nil)
	body := []byte("From: someone@example.org\r\nTo: bounces@news.acme.io\r\nSubject: Thanks!\r\n\r\nGreat newsletter.\r\n")
	evs, err := p.ParseWebhook(providers.WithSignatureVerified(context.Background()), nil, body)
	if err != nil || len(evs) != 0 {
		t.Fatalf("a human reply is not a bounce: %v %v", evs, err)
	}
}
