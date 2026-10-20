package vonage_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/contracttest"
	"github.com/hostrepute/reputecast/internal/sms/vonage"
	"github.com/hostrepute/reputecast/internal/sms/vonage/vonagetest"
)

var now = time.Date(2026, 10, 14, 9, 40, 55, 0, time.UTC)

func open(t *testing.T, base string) sms.Provider {
	t.Helper()
	p, err := vonage.New(sms.Config{BaseURL: base, Settings: map[string]string{"api_key": vonagetest.APIKey},
		Credentials: map[string]string{"api_secret": vonagetest.APISecret, "signature_secret": vonagetest.SignatureSecret}, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestContract(t *testing.T) {
	s := vonagetest.SignatureSecret
	contracttest.Run(t, contracttest.Suite{
		Adapter: vonage.Adapter,
		API: func(env *contracttest.Env) http.Handler {
			f := vonagetest.New()
			f.Mode = env.Mode
			return f.Handler()
		},
		New:  func(t *testing.T, env *contracttest.Env) sms.Provider { return open(t, env.URL) },
		From: "ACME",
		Webhooks: []contracttest.Webhook{
			{Name: "delivered", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return vonagetest.Receipt(s, u, "1800000000000001", "+447700900456", "ACME", "delivered", "0", now)
			}, Want: []sms.Event{{Kind: sms.EventDelivered, ProviderMessageID: "1800000000000001", To: "+447700900456", From: "ACME"}}},
			{Name: "absent subscriber", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return vonagetest.Receipt(s, u, "1800000000003307", "+447700903307", "ACME", "failed", "1", now)
			}, Want: []sms.Event{{Kind: sms.EventUndelivered, ProviderMessageID: "1800000000003307", Code: "1", Reason: sms.ReasonUnreachable}}},
			{Name: "inbound Stop", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return vonagetest.Inbound(s, u, "0A0000001234", "+447700902290", "447700900000", "Stop", now)
			}, Want: []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: "0A0000001234", From: "+447700902290", Body: "Stop"}}},
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			b := append([]byte{}, body...)
			for i := range b {
				if b[i] == '0' {
					b[i] = '9'
					break
				}
			}
			return r.Clone(context.Background()), b
		},
	})
}

func TestOldWebhooksAreRefused(t *testing.T) {
	p := open(t, "http://unused.invalid")
	r, body := vonagetest.Receipt(vonagetest.SignatureSecret, "https://reputecast.acme.io/hooks/sms/vonage/x", "1", "+447700900456", "ACME", "delivered", "0", now.Add(-10*time.Minute))
	if err := p.VerifyWebhook(context.Background(), r, body); err == nil {
		t.Fatal("a webhook signed 10 minutes ago is a replay")
	}
}

func TestRegisterWebhooksSetsTheAccountCallbacks(t *testing.T) {
	f := vonagetest.New()
	srv := f.Server()
	defer srv.Close()
	p := open(t, srv.URL).(sms.WebhookRegistrar)
	if _, err := p.RegisterWebhooks(context.Background(), sms.Webhooks{StatusURL: "https://h/s", InboundURL: "https://h/i"}, nil); err != nil {
		t.Fatal(err)
	}
	if dr, mo := f.Callbacks(); dr != "https://h/s" || mo != "https://h/i" {
		t.Fatalf("callbacks %q %q", dr, mo)
	}
}
