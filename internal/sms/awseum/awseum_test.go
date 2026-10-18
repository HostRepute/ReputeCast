package awseum_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/awseum"
	"github.com/hostrepute/reputecast/internal/sms/awseum/awseumtest"
	"github.com/hostrepute/reputecast/internal/sms/contracttest"
)

var now = time.Date(2026, 10, 14, 15, 0, 3, 0, time.UTC)

func open(t *testing.T, f *awseumtest.Fake, settings map[string]string) sms.Provider {
	t.Helper()
	s := map[string]string{"region": "us-east-1", "access": "keys", "access_key_id": awseumtest.AccessKeyID}
	for k, v := range settings {
		s[k] = v
	}
	p, err := awseum.New(sms.Config{BaseURL: "http://aws.fake", HTTP: f.Client(), Settings: s, Credentials: map[string]string{"secret_access_key": awseumtest.SecretAccessKey},
		Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestContract(t *testing.T) {
	var fake *awseumtest.Fake
	msgID := uuid.MustParse("019a0e31-0000-7000-8000-000000000001")
	contracttest.Run(t, contracttest.Suite{
		Adapter: awseum.Adapter,
		API: func(env *contracttest.Env) http.Handler {
			fake = awseumtest.New()
			fake.Mode = env.Mode
			return fake.Handler()
		},
		New:  func(t *testing.T, _ *contracttest.Env) sms.Provider { return open(t, fake, nil) },
		From: "+18885550160",
		Webhooks: []contracttest.Webhook{
			{Name: "delivered", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return awseumtest.Event(u, "TEXT_DELIVERED", "m-1", "+13125550123", "+18885550160", msgID.String(), now)
			}, Want: []sms.Event{{Kind: sms.EventDelivered, ProviderMessageID: "m-1", To: "+13125550123", From: "+18885550160"}}},
			{Name: "carrier blocked", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return awseumtest.Event(u, "TEXT_CARRIER_BLOCKED", "m-2", "+13125550124", "+18885550160", msgID.String(), now)
			}, Want: []sms.Event{{Kind: sms.EventUndelivered, ProviderMessageID: "m-2", Code: "TEXT_CARRIER_BLOCKED", Reason: sms.ReasonFiltered}}},
			{Name: "two-way STOP", Request: func(_ *testing.T, _ *contracttest.Env, u string) (*http.Request, []byte) {
				return awseumtest.Inbound(u, "in-1", "+13125550125", "+18885550160", "STOP", now)
			}, Want: []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: "in-1", From: "+13125550125", To: "+18885550160", Body: "STOP"}}},
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			b := append([]byte{}, body...)
			for i := range b {
				if b[i] == 'D' {
					b[i] = 'E'
					break
				}
			}
			return r.Clone(context.Background()), b
		},
	})
}

func TestEventsCarryReputeCastsMessageID(t *testing.T) {
	f := awseumtest.New()
	p := open(t, f, nil)
	id := uuid.Must(uuid.NewV7())
	r, body := awseumtest.Event("https://h/hooks/sms/aws_eum/x", "TEXT_DELIVERED", "m-9", "+13125550123", "+18885550160", id.String(), now)
	if err := p.VerifyWebhook(context.Background(), r, body); err != nil {
		t.Fatal(err)
	}
	evs, err := p.ParseWebhook(context.Background(), "", body)
	if err != nil || len(evs) != 1 || evs[0].MessageID == nil || *evs[0].MessageID != id {
		t.Fatalf("events %v %+v", err, evs)
	}
}

func TestSubscriptionIsConfirmedAndOtherTopicsRefused(t *testing.T) {
	f := awseumtest.New()
	p := open(t, f, map[string]string{"topic_arn": awseumtest.TopicARN})
	r, body := awseumtest.SubscriptionConfirmation("https://h/hooks/sms/aws_eum/x", now)
	handled, err := p.(sms.WebhookHandshaker).Handshake(context.Background(), r, body)
	if !handled || err != nil || !f.Confirmed {
		t.Fatalf("handshake %v %v confirmed %v", handled, err, f.Confirmed)
	}
	other := open(t, f, map[string]string{"topic_arn": "arn:aws:sns:us-east-1:999999999999:someone-else"})
	r2, body2 := awseumtest.Event("https://h/x", "TEXT_DELIVERED", "m", "+1", "+1", "", now)
	if err := other.VerifyWebhook(context.Background(), r2, body2); err == nil {
		t.Fatal("a notification from another topic is refused")
	}
}

func TestRegisterWebhooksCreatesTheTopicAndTwoWay(t *testing.T) {
	f := awseumtest.New()
	p := open(t, f, nil).(sms.WebhookRegistrar)
	setup, err := p.RegisterWebhooks(context.Background(), sms.Webhooks{StatusURL: "https://h/hooks/sms/aws_eum/x", InboundURL: "https://h/hooks/sms/aws_eum/x"},
		[]sms.Sender{{Kind: "toll_free", Value: "+18885550160", ProviderRef: "phone-0160"}})
	if err != nil {
		t.Fatal(err)
	}
	if setup.Settings["topic_arn"] != awseumtest.TopicARN || setup.Settings["configuration_set"] == "" || f.TwoWay["phone-0160"] != awseumtest.TopicARN {
		t.Fatalf("setup %+v two-way %+v", setup, f.TwoWay)
	}
}
