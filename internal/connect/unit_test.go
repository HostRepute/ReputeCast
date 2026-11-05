package connect

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
)

func TestWebhookEventName(t *testing.T) {
	for _, c := range []struct{ name, body, want string }{
		{"ses via sns", `{"Type":"Notification","Message":"{\"eventType\":\"Send\",\"mail\":{}}"}`, "Send"},
		{"sns handshake", `{"Type":"SubscriptionConfirmation","SubscribeURL":"https://sns"}`, ""},
		{"postmark", `{"RecordType":"Delivery","MessageID":"x"}`, "Delivery"},
		{"mailgun", `{"signature":{"token":"t"},"event-data":{"event":"delivered","id":"x"}}`, "delivered"},
		{"sendgrid", `[{"event":"processed","email":"a@b.c"},{"event":"delivered"}]`, "processed"},
		{"resend", `{"type":"email.sent","data":{"email_id":"x"}}`, "email.sent"},
		{"sparkpost", `[{"msys":{"message_event":{"type":"delivery"}}}]`, "delivery"},
		{"sparkpost track", `[{"msys":{"track_event":{"type":"open"}}}]`, "open"},
		{"not json", `event=delivered`, ""},
		{"no event", `{"hello":"world"}`, ""},
	} {
		if got := webhookEventName([]byte(c.body)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestStreamsText(t *testing.T) {
	en := i18n.Default
	ses := []manage.ProviderStreamRole{{Stream: "broadcast", Role: "split", Percent: 90}, {Stream: "automation", Role: "primary"}, {Stream: "system", Role: "fallback"}}
	if got := streamsText(en, ses); got != "Broadcasts 90% · Automations · System fallback" {
		t.Errorf("ses: %q", got)
	}
	// The streams a provider sends come first, then the ones it backs up.
	postmark := []manage.ProviderStreamRole{{Stream: "automation", Role: "fallback"}, {Stream: "system", Role: "primary"}}
	if got := streamsText(en, postmark); got != "System · Automations fallback" {
		t.Errorf("postmark: %q", got)
	}
	if got := streamsText(en, []manage.ProviderStreamRole{{Stream: "broadcast", Role: "condition"}}); got != "Broadcasts (rules)" {
		t.Errorf("rules: %q", got)
	}
	if got := streamsText(en, nil); got != "Not routed yet" {
		t.Errorf("none: %q", got)
	}
	if got := streamsText(i18n.For(i18n.Vietnamese), ses); got != "Gửi hàng loạt 90% · Tự động hóa · dự phòng Hệ thống" {
		t.Errorf("Vietnamese: %q", got)
	}
}

// Mail server tests show what a server answers, so they don't reach
// internal addresses unless private networks are allowed.
func TestCheckHost(t *testing.T) {
	ctx := context.Background()
	closed := &Service{}
	for _, host := range []string{"127.0.0.1", "10.0.4.18", "localhost", "smtp.internal", "[::1]", "169.254.169.254"} {
		if err := closed.checkHost(ctx, host); !netguard.IsBlocked(err) {
			t.Errorf("%s: err = %v", host, err)
		}
	}
	if err := closed.checkHost(ctx, "203.0.113.25"); err != nil {
		t.Errorf("public address: %v", err)
	}
	open := &Service{AllowPrivateNetworks: true}
	for _, host := range []string{"127.0.0.1", "10.0.4.18", "localhost"} {
		if err := open.checkHost(ctx, host); err != nil {
			t.Errorf("%s with private networks allowed: %v", host, err)
		}
	}
	if err := open.checkHost(ctx, "169.254.169.254"); !netguard.IsBlocked(err) {
		t.Errorf("metadata address with private networks allowed: %v", err)
	}
	e := hostBlocked("host", closed.checkHost(ctx, "localhost"))
	if e.Field != "host" || e.Code != "address_blocked" || !strings.Contains(e.Message, "providers.allow_private_networks") {
		t.Errorf("explanation: %+v", e)
	}
}

// The self-test follows no redirect and dials no private address.
func TestSelfTestGuardsTheAddress(t *testing.T) {
	var inside atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		inside.Add(1)
	}))
	defer internal.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL+"/admin/stop", http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	now := time.Now()

	st := SelfTest(context.Background(), redirect.URL+"/hooks/postal/tok", "tok", now, true)
	if st.OK || st.Code != "HTTP 307" || inside.Load() != 0 {
		t.Fatalf("redirect followed: %+v, internal hit %d times", st, inside.Load())
	}

	u, _ := url.Parse(internal.URL + "/hooks/postal/tok")
	st = selfTestPost(context.Background(), SelfTestState{At: now}, u, u.String(), "tok", now, false)
	if st.OK || !st.Private || st.Code != "ECONNREFUSED" || !strings.Contains(st.Reason, "127.0.0.1 is a private address") || inside.Load() != 0 {
		t.Fatalf("private address dialed: %+v, internal hit %d times", st, inside.Load())
	}

	if st = SelfTest(context.Background(), internal.URL+"/hooks/postal/tok", "tok", now, true); !st.OK {
		t.Fatalf("allowed private receiver: %+v", st)
	}
}

// A new address for the provider needs every saved secret the form leaves
// blank typed again, each one named next to its field; the same address
// (in another case, with a final slash) or another setting keeps them.
func TestSecretsForAddress(t *testing.T) {
	d := providers.Descriptor{Fields: []providers.Field{
		{Key: "server_url", Label: "Server URL", Endpoint: true},
		{Key: "api_key", Label: "API key", Secret: true},
		{Key: "signing_key", Label: "Signing key", Secret: true},
		{Key: "region", Label: "Region", Options: []string{"us", "eu"}},
	}}
	stored := map[string]any{"server_url": "https://mta.acme.io", "region": "us"}
	creds := map[string]any{"api_key": "k1", "signing_key": "s1"}
	en := i18n.Default
	for _, v := range []map[string]string{{}, {"server_url": "HTTPS://mta.acme.io/"}, {"region": "eu"},
		{"server_url": "https://evil.example", "api_key": "k2", "signing_key": "s2"}} {
		if err := secretsForAddress(en, d, stored, creds, v); err != nil {
			t.Errorf("%v: %v", v, err)
		}
	}
	if err := secretsForAddress(en, d, stored, map[string]any{}, map[string]string{"server_url": "https://evil.example"}); err != nil {
		t.Errorf("nothing saved to keep: %v", err)
	}
	var p *api.Problem
	err := secretsForAddress(en, d, stored, creds, map[string]string{"server_url": "https://evil.example", "api_key": "k2"})
	if !errors.As(err, &p) || len(p.Errors) != 1 || p.Errors[0].Location != "body.values.signing_key" ||
		p.Errors[0].Message != "The address changed: enter the signing key again. ReputeCast doesn't send a saved secret to a new address." {
		t.Fatalf("one secret left blank: %v", err)
	}
	err = secretsForAddress(en, d, stored, creds, map[string]string{"server_url": "https://evil.example"})
	if !errors.As(err, &p) || p.Status != 422 || len(p.Errors) != 2 || p.Errors[0].Location != "body.values.api_key" || p.Errors[1].Location != "body.values.signing_key" ||
		!strings.HasPrefix(p.Detail, "The address changed: enter the API key again.") {
		t.Fatalf("both left blank: %v", err)
	}
}
