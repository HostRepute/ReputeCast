package connect

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
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
