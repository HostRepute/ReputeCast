package connect

import (
	"testing"

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
	ses := []manage.ProviderStreamRole{{Stream: "broadcast", Role: "split", Percent: 90}, {Stream: "automation", Role: "primary"}, {Stream: "system", Role: "fallback"}}
	if got := streamsText(ses); got != "Broadcasts 90% · Automations · System fallback" {
		t.Errorf("ses: %q", got)
	}
	// The streams a provider sends come first, then the ones it backs up.
	postmark := []manage.ProviderStreamRole{{Stream: "automation", Role: "fallback"}, {Stream: "system", Role: "primary"}}
	if got := streamsText(postmark); got != "System · Automations fallback" {
		t.Errorf("postmark: %q", got)
	}
	if got := streamsText([]manage.ProviderStreamRole{{Stream: "broadcast", Role: "condition"}}); got != "Broadcasts (rules)" {
		t.Errorf("rules: %q", got)
	}
	if got := streamsText(nil); got != "Not routed yet" {
		t.Errorf("none: %q", got)
	}
}
