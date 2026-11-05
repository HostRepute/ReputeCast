package awseum_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/awseum"
	"github.com/hostrepute/reputecast/internal/sms/awseum/awseumtest"
)

const hook = "https://h/hooks/sms/aws_eum/x"

// Signing certificates come only from a regional SNS endpoint: other
// amazonaws.com hosts of the same shape are S3 buckets anyone can own.
func TestCertURLsAreRegionalSNSEndpoints(t *testing.T) {
	for _, u := range []string{
		awseumtest.CertURL,
		"https://sns.eu-south-2.amazonaws.com/x.pem",
		"https://sns.ap-southeast-7.amazonaws.com/x.pem",
		"https://sns.me-central-1.amazonaws.com/x.pem",
		"https://sns.us-gov-east-1.amazonaws.com/x.pem",
		"https://sns.cn-north-1.amazonaws.com.cn/x.pem",
	} {
		if !awseum.ValidCertURL(u) {
			t.Errorf("%s is an SNS endpoint", u)
		}
	}
	for _, u := range []string{
		"https://sns.s3.amazonaws.com/x.pem",
		"https://sns.s3-external-1.amazonaws.com/x.pem",
		"https://sns.s3-us-west-2.amazonaws.com/x.pem",
		"https://sns.s3.us-east-1.amazonaws.com/x.pem",
		"https://sns.us-east-1.s3.amazonaws.com/x.pem",
		"https://sns.amazonaws.com/x.pem",
		"https://sns.us-east-1.amazonaws.com.evil.io/x.pem",
		"http://sns.us-east-1.amazonaws.com/x.pem",
		"https://sns.us-east-1.amazonaws.com:8443/x.pem",
		"https://user@sns.us-east-1.amazonaws.com/x.pem",
		"https://sns.us-east-1.amazonaws.com/x.pem?Action=Echo",
	} {
		if awseum.ValidCertURL(u) {
			t.Errorf("%s must not be accepted as a signing certificate URL", u)
		}
	}
}

// openPinned opens the sample account as the webhook does: settings as
// stored, and a way to keep the topic.
func openPinned(t *testing.T, f *awseumtest.Fake, settings map[string]string, pin func(context.Context, string, string) (string, error)) sms.Provider {
	t.Helper()
	s := map[string]string{"region": "us-east-1", "access": "keys", "access_key_id": awseumtest.AccessKeyID}
	for k, v := range settings {
		s[k] = v
	}
	p, err := awseum.New(sms.Config{BaseURL: "http://aws.fake", HTTP: f.Client(), Settings: s, Credentials: map[string]string{"secret_access_key": awseumtest.SecretAccessKey},
		Pin: pin})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Webhooks set up by hand in AWS have no topic kept: nothing is trusted
// until the first subscription confirmation SNS signed keeps its topic,
// and from then on every other topic is refused (trust on first use).
func TestFirstConfirmationKeepsItsTopic(t *testing.T) {
	ctx := context.Background()
	f := awseumtest.New()
	kept, pins := "", 0
	pin := func(_ context.Context, key, value string) (string, error) {
		pins++
		if key != "topic_arn" {
			t.Fatalf("kept %q", key)
		}
		if kept == "" {
			kept = value
		}
		return kept, nil
	}
	p := openPinned(t, f, nil, pin)
	r, body := awseumtest.Event(hook, "TEXT_DELIVERED", "m-1", "+13125550123", "+18885550160", "", now)
	if err := p.VerifyWebhook(ctx, r, body); !errors.Is(err, sms.ErrBadSignature) {
		t.Fatalf("an event before any topic is kept: %v", err)
	}
	// A confirmation that isn't SNS's keeps nothing.
	r, body = awseumtest.SubscriptionConfirmation(hook, now)
	forged := []byte(strings.Replace(string(body), "reputecast-sms", "reputecast-smz", 1))
	if handled, err := p.(sms.WebhookHandshaker).Handshake(ctx, r, forged); !handled || !errors.Is(err, sms.ErrBadSignature) || pins != 0 || f.Confirmed {
		t.Fatalf("forged confirmation: handled %v, %v, kept %d, confirmed %v", handled, err, pins, f.Confirmed)
	}
	handled, err := p.(sms.WebhookHandshaker).Handshake(ctx, r, body)
	if !handled || err != nil || !f.Confirmed || kept != awseumtest.TopicARN {
		t.Fatalf("first confirmation: handled %v, %v, confirmed %v, kept %q", handled, err, f.Confirmed, kept)
	}
	// Another topic's confirmation, SNS-signed too, is refused: the topic
	// kept first wins even for an adapter opened before it was kept.
	f.Confirmed = false
	r, body = awseumtest.SubscriptionConfirmationOf(hook, "arn:aws:sns:us-east-1:999999999999:someone-else", now)
	if handled, err := p.(sms.WebhookHandshaker).Handshake(ctx, r, body); !handled || !errors.Is(err, sms.ErrBadSignature) || f.Confirmed {
		t.Fatalf("another topic: handled %v, %v, confirmed %v", handled, err, f.Confirmed)
	}
	if kept != awseumtest.TopicARN {
		t.Fatalf("the kept topic changed: %q", kept)
	}
	p = openPinned(t, f, map[string]string{"topic_arn": kept}, pin)
	r, body = awseumtest.Event(hook, "TEXT_DELIVERED", "m-1", "+13125550123", "+18885550160", "", now)
	if err := p.VerifyWebhook(ctx, r, body); err != nil {
		t.Fatalf("an event from the kept topic: %v", err)
	}
	r, body = awseumtest.SubscriptionConfirmationOf(hook, "arn:aws:sns:us-east-1:999999999999:someone-else", now)
	if handled, err := p.(sms.WebhookHandshaker).Handshake(ctx, r, body); !handled || !errors.Is(err, sms.ErrBadSignature) || f.Confirmed || pins != 2 {
		t.Fatalf("another topic once one is kept: handled %v, %v, confirmed %v, kept %d", handled, err, f.Confirmed, pins)
	}

	// Where nothing can be kept, nothing new is trusted.
	r, body = awseumtest.SubscriptionConfirmation(hook, now)
	if handled, err := open(t, f, nil).(sms.WebhookHandshaker).Handshake(ctx, r, body); !handled || !errors.Is(err, sms.ErrBadSignature) || f.Confirmed {
		t.Fatalf("no way to keep the topic: handled %v, %v, confirmed %v", handled, err, f.Confirmed)
	}
	// A topic that couldn't be kept is no signature failure: the webhook
	// answers 502 and SNS tries again.
	p = openPinned(t, f, nil, func(context.Context, string, string) (string, error) { return "", errors.New("database gone") })
	if handled, err := p.(sms.WebhookHandshaker).Handshake(ctx, r, body); !handled || err == nil || errors.Is(err, sms.ErrBadSignature) || f.Confirmed {
		t.Fatalf("keeping failed: handled %v, %v, confirmed %v", handled, err, f.Confirmed)
	}
}

// Registering the webhooks keeps the topic with the provider before the
// webhook subscribes to it: SNS asks for the confirmation at once and
// doesn't retry a refusal.
func TestRegisterWebhooksKeepsTheTopicFirst(t *testing.T) {
	ctx := context.Background()
	senders := []sms.Sender{{Kind: "toll_free", Value: "+18885550160", ProviderRef: "phone-0160"}}
	subscribed := func(f *awseumtest.Fake) int {
		for i, r := range f.Requests() {
			if strings.HasPrefix(r.Path, "/sns") && strings.Contains(r.Body, "Action=Subscribe") {
				return i
			}
		}
		return -1
	}
	f := awseumtest.New()
	var kept map[string]string
	before := -1
	save := func(_ context.Context, s map[string]string) error {
		kept, before = s, len(f.Requests())
		return nil
	}
	setup, err := open(t, f, nil).(sms.WebhookRegistrar).RegisterWebhooks(ctx, sms.Webhooks{StatusURL: hook, InboundURL: hook, Save: save}, senders)
	if err != nil {
		t.Fatal(err)
	}
	if kept["topic_arn"] != awseumtest.TopicARN || setup.Settings["topic_arn"] != awseumtest.TopicARN {
		t.Fatalf("kept %v, settings %v", kept, setup.Settings)
	}
	if at := subscribed(f); before < 1 || at < before {
		t.Fatalf("kept after %d requests, subscribed at %d", before, at)
	}

	f = awseumtest.New()
	save = func(context.Context, map[string]string) error { return errors.New("database gone") }
	if _, err := open(t, f, nil).(sms.WebhookRegistrar).RegisterWebhooks(ctx, sms.Webhooks{StatusURL: hook, InboundURL: hook, Save: save}, senders); err == nil || subscribed(f) >= 0 {
		t.Fatalf("subscribed without keeping the topic: %v", err)
	}
}
