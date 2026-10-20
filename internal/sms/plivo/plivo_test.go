package plivo_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/contracttest"
	"github.com/hostrepute/reputecast/internal/sms/plivo"
	"github.com/hostrepute/reputecast/internal/sms/plivo/plivotest"
)

func open(t *testing.T, base string) sms.Provider {
	t.Helper()
	p, err := plivo.New(sms.Config{BaseURL: base, Settings: map[string]string{"auth_id": plivotest.AuthID}, Credentials: map[string]string{"auth_token": plivotest.AuthToken}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestContract(t *testing.T) {
	tok := plivotest.AuthToken
	contracttest.Run(t, contracttest.Suite{
		Adapter: plivo.Adapter,
		API: func(env *contracttest.Env) http.Handler {
			f := plivotest.New()
			f.Mode = env.Mode
			return f.Handler()
		},
		New:  func(t *testing.T, env *contracttest.Env) sms.Provider { return open(t, env.URL) },
		From: "+18885550170",
		Webhooks: []contracttest.Webhook{
			{Name: "delivered", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return plivotest.Receipt(tok, u, "db3ce55a-0001", "+13125550123", "+18885550170", "delivered", "000")
			}, Want: []sms.Event{{Kind: sms.EventDelivered, ProviderMessageID: "db3ce55a-0001", To: "+13125550123", From: "+18885550170"}}},
			{Name: "spam detected", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return plivotest.Receipt(tok, u, "db3ce55a-0030", "+13125550124", "+18885550170", "undelivered", "30")
			}, Want: []sms.Event{{Kind: sms.EventUndelivered, ProviderMessageID: "db3ce55a-0030", Code: "30", Reason: sms.ReasonFiltered}}},
			{Name: "inbound HELP", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return plivotest.Inbound(tok, u, "in-0001", "+13125550125", "+18885550170", "HELP")
			}, Want: []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: "in-0001", From: "+13125550125", Body: "HELP"}}},
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			return r.Clone(context.Background()), append(append([]byte{}, body...), "&Status=failed"...)
		},
	})
}
