package telnyx_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/contracttest"
	"github.com/hostrepute/reputecast/internal/sms/telnyx"
	"github.com/hostrepute/reputecast/internal/sms/telnyx/telnyxtest"
)

var now = time.Date(2026, 10, 14, 15, 0, 4, 0, time.UTC)

func open(t *testing.T, base string) sms.Provider {
	t.Helper()
	p, err := telnyx.New(sms.Config{BaseURL: base, Settings: map[string]string{"public_key": telnyxtest.PublicKey}, Credentials: map[string]string{"api_key": telnyxtest.APIKey},
		Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Adapter: telnyx.Adapter,
		API: func(env *contracttest.Env) http.Handler {
			f := telnyxtest.New()
			f.Mode = env.Mode
			return f.Handler()
		},
		New:  func(t *testing.T, env *contracttest.Env) sms.Provider { return open(t, env.URL) },
		From: "+18885550190",
		Webhooks: []contracttest.Webhook{
			{Name: "delivered", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return telnyxtest.Receipt(u, "msg-1", "+13125550123", "+18885550190", "delivered", "", now)
			}, Want: []sms.Event{{Kind: sms.EventDelivered, ProviderMessageID: "msg-1", To: "+13125550123", From: "+18885550190"}}},
			{Name: "blocked as spam", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return telnyxtest.Receipt(u, "msg-2", "+13125550124", "+18885550190", "delivery_failed", "40003", now)
			}, Want: []sms.Event{{Kind: sms.EventUndelivered, ProviderMessageID: "msg-2", Code: "40003", Reason: sms.ReasonFiltered}}},
			{Name: "received", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return telnyxtest.Inbound(u, "msg-in", "+13125550125", "+18885550190", "START", now)
			}, Want: []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: "msg-in", From: "+13125550125", To: "+18885550190", Body: "START"}}},
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			b := append([]byte{}, body...)
			b[len(b)-2] = ' '
			return r.Clone(context.Background()), b
		},
	})
}

func TestReplayIsRefused(t *testing.T) {
	p := open(t, "http://unused.invalid")
	r, body := telnyxtest.Receipt("https://h/hooks", "m", "+13125550123", "+18885550190", "delivered", "", now.Add(-time.Hour))
	if err := p.VerifyWebhook(context.Background(), r, body); err == nil {
		t.Fatal("a webhook signed an hour ago is a replay")
	}
}
