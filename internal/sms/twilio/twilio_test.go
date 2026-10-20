package twilio_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/contracttest"
	"github.com/hostrepute/reputecast/internal/sms/twilio"
	"github.com/hostrepute/reputecast/internal/sms/twilio/twiliotest"
)

const secret = "0f3c9a1d7e2b4c6a8d0e1f2a3b4c5d6e"

func open(t *testing.T, base string) sms.Provider {
	t.Helper()
	p, err := twilio.New(sms.Config{BaseURL: base, Settings: map[string]string{"account_sid": twiliotest.AccountSID, "api_key_sid": twiliotest.KeySID, "mode": "live"},
		Credentials: map[string]string{"api_key_secret": twiliotest.KeySecret}, WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestContract(t *testing.T) {
	var fake *twiliotest.Fake
	contracttest.Run(t, contracttest.Suite{
		Adapter: twilio.Adapter,
		API: func(env *contracttest.Env) http.Handler {
			fake = twiliotest.New()
			fake.Mode = env.Mode
			return fake.Handler()
		},
		New:  func(t *testing.T, env *contracttest.Env) sms.Provider { return open(t, env.URL) },
		From: "+18885550142",
		Webhooks: []contracttest.Webhook{
			{Name: "delivered", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return twiliotest.Receipt(secret, u, "SM9a41c207", "+13125550123", "+18885550142", "delivered", "")
			}, Want: []sms.Event{{Kind: sms.EventDelivered, ProviderMessageID: "SM9a41c207", To: "+13125550123", From: "+18885550142"}}},
			{Name: "filtered", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return twiliotest.Receipt(secret, u, "SM30007", "+16305550121", "+18885550142", "undelivered", "30007")
			}, Want: []sms.Event{{Kind: sms.EventUndelivered, ProviderMessageID: "SM30007", Code: "30007", Reason: sms.ReasonFiltered}}},
			{Name: "not in service", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return twiliotest.Receipt(secret, u, "SM30005", "+14155550177", "+18885550142", "failed", "30005")
			}, Want: []sms.Event{{Kind: sms.EventFailed, ProviderMessageID: "SM30005", Code: "30005", Reason: sms.ReasonNotInService}}},
			{Name: "inbound STOP", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return twiliotest.Inbound(secret, u, "SMin0148", "+17735550148", "+18885550142", "STOP", "STOP")
			}, Want: []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: "SMin0148", From: "+17735550148", To: "+18885550142", Body: "STOP", OptOut: "STOP"}}},
			{Name: "inbound reply", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return twiliotest.Inbound(secret, u, "SMin0192", "+13125550192", "+18885550142", "Is there parking?", "")
			}, Want: []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: "SMin0192", Body: "Is there parking?"}}},
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			return r.Clone(context.Background()), append(append([]byte{}, body...), "&ErrorCode=30003"...)
		},
	})
	_ = fake
}

// The signature covers the address Twilio called: the same body at another
// address fails.
func TestSignatureCoversTheURL(t *testing.T) {
	p := open(t, "http://unused.invalid")
	r, body := twiliotest.Receipt(secret, "https://reputecast.acme.io/hooks/sms/twilio/abc", "SM1", "+13125550123", "+18885550142", "delivered", "")
	if err := p.VerifyWebhook(context.Background(), r, body); err != nil {
		t.Fatal(err)
	}
	r.URL, _ = url.Parse("https://elsewhere.example/hooks/sms/twilio/abc")
	if err := p.VerifyWebhook(context.Background(), r, body); err == nil {
		t.Fatal("a replay at another address must fail")
	}
	// The account's default signature (auth token, no settings header) can't
	// be checked with an API key and is refused.
	r2, body2 := twiliotest.Receipt(secret, "https://reputecast.acme.io/hooks/sms/twilio/abc", "SM1", "+13125550123", "+18885550142", "delivered", "")
	r2.Header.Del(twilio.SettingsHeader)
	if err := p.VerifyWebhook(context.Background(), r2, body2); err == nil {
		t.Fatal("a webhook without the shared key's settings header is refused")
	}
}

func TestRegisterWebhooksSetsNumbersAndReturnsTheSharedKey(t *testing.T) {
	fake := twiliotest.New()
	srv := fake.Server()
	defer srv.Close()
	p := open(t, srv.URL).(sms.WebhookRegistrar)
	setup, err := p.RegisterWebhooks(context.Background(), sms.Webhooks{StatusURL: "https://reputecast.acme.io/hooks/sms/twilio/tok", InboundURL: "https://reputecast.acme.io/hooks/sms/twilio/tok"},
		[]sms.Sender{{Kind: "toll_free", Value: "+18885550142"}, {Kind: "sender_id", Value: "ACME"}})
	if err != nil {
		t.Fatal(err)
	}
	if setup.Secret == "" || setup.Secret != fake.SharedKeySecret {
		t.Fatalf("the shared key's secret comes back once: %q", setup.Secret)
	}
	if fake.Numbers[0].SmsURL != "https://reputecast.acme.io/hooks/sms/twilio/tok" {
		t.Fatalf("inbound webhook on the number: %q", fake.Numbers[0].SmsURL)
	}
}

func TestRegistrationsAndSenders(t *testing.T) {
	fake := twiliotest.New()
	srv := fake.Server()
	defer srv.Close()
	p := open(t, srv.URL)
	ss, err := p.(sms.SenderLister).Senders(context.Background())
	if err != nil || len(ss) != 3 || ss[0].Kind != "toll_free" || ss[0].Throughput != 3 || ss[1].Kind != "10dlc" {
		t.Fatalf("senders %v %+v", err, ss)
	}
	regs, err := p.(sms.RegistrationReader).Registrations(context.Background(), ss[:2])
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range regs {
		got[r.Sender+" "+r.Kind] = r.Status
	}
	if got["+18885550142 toll_free"] != "verified" || got["+13125550187 brand"] != "approved" || got["+13125550187 campaign"] != "pending" {
		t.Fatalf("registrations %+v", got)
	}
	fake.CampaignStatus, fake.CampaignReason = "FAILED", "Opt-in proof URL could not be verified"
	regs, _ = p.(sms.RegistrationReader).Registrations(context.Background(), ss[1:2])
	for _, r := range regs {
		if r.Kind == "campaign" && (r.Status != "rejected" || r.Reason == "") {
			t.Fatalf("a rejected campaign carries the carrier's reason: %+v", r)
		}
	}
}
