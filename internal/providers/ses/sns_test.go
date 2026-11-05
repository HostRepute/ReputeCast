package ses_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/ses"
)

// Signing certificates come only from a regional SNS endpoint: other
// amazonaws.com hosts of the same shape are S3 buckets anyone can own.
func TestCertURLsAreRegionalSNSEndpoints(t *testing.T) {
	for _, u := range []string{
		"https://sns.us-east-1.amazonaws.com/SimpleNotificationService-9c6465fa7f48f5cacd23014631ec1136.pem",
		"https://sns.eu-central-2.amazonaws.com/x.pem",
		"https://sns.ap-southeast-7.amazonaws.com/x.pem",
		"https://sns.il-central-1.amazonaws.com/x.pem",
		"https://sns.mx-central-1.amazonaws.com/x.pem",
		"https://sns.ca-west-1.amazonaws.com/x.pem",
		"https://sns.us-gov-west-1.amazonaws.com/x.pem",
		"https://sns.us-isob-east-1.amazonaws.com/x.pem",
		"https://sns.cn-north-1.amazonaws.com.cn/x.pem",
		"https://sns.cn-northwest-1.amazonaws.com.cn/x.pem",
	} {
		if !ses.ValidCertURL(u) {
			t.Errorf("%s is an SNS endpoint", u)
		}
	}
	for _, u := range []string{
		"https://sns.s3.amazonaws.com/x.pem",                   // the S3 bucket "sns"
		"https://sns.s3-external-1.amazonaws.com/x.pem",        // the same bucket, another S3 endpoint
		"https://sns.s3-us-west-2.amazonaws.com/x.pem",         // S3's older regional endpoints
		"https://sns.s3-website-us-east-1.amazonaws.com/x.pem", // a bucket's website
		"https://sns.s3.us-east-1.amazonaws.com/x.pem",
		"https://sns.us-east-1.s3.amazonaws.com/x.pem", // the bucket "sns.us-east-1"
		"https://sns.amazonaws.com/x.pem",
		"https://sns.us-east.amazonaws.com/x.pem",
		"https://evil.sns.us-east-1.amazonaws.com/x.pem",
		"https://sns.us-east-1.amazonaws.com.evil.io/x.pem",
		"https://evil.example/cert.pem",
		"http://sns.us-east-1.amazonaws.com/x.pem",
		"https://sns.us-east-1.amazonaws.com:8443/x.pem",
		"https://user@sns.us-east-1.amazonaws.com/x.pem",
		"https://sns.us-east-1.amazonaws.com/x.pem?Action=Echo",
		"https://sns.us-east-1.amazonaws.com/x.txt",
	} {
		if ses.ValidCertURL(u) {
			t.Errorf("%s must not be accepted as a signing certificate URL", u)
		}
	}
}

// providerWithTopics is the contract's provider with other saved topics.
func providerWithTopics(t *testing.T, env *contracttest.Env, confirmed *bool, topics []string) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(ses.Settings{Region: "us-east-1", ConfigurationSet: "rc-broadcasts", Access: "keys", Endpoint: env.URL, TopicARNs: topics})
	p, err := ses.New(providers.Config{
		Settings:    settings,
		Credentials: json.RawMessage(`{"access_key_id":"AKIDEXAMPLE","secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`),
		HTTPClient:  &http.Client{Transport: transport{t: t, confirmed: confirmed}, Timeout: 10 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Without a saved topic, nothing SNS signs is trusted: neither events nor a
// subscription confirmation, whoever's topic it comes from.
func TestNoSavedTopicTrustsNothing(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	ctx := context.Background()
	confirmed := false
	p := providerWithTopics(t, env, &confirmed, nil)
	r, body := notification("bounce.json", "2")(t, env)
	if err := p.(providers.WebhookVerifier).VerifyWebhook(ctx, r, body); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("an event without a saved topic: %v", err)
	}
	if _, err := p.ParseWebhook(ctx, r, body); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("parsed without a saved topic: %v", err)
	}
	conf := envelope(t, "SubscriptionConfirmation", "You have chosen to subscribe to the topic.", "2")
	handled, _, err := p.(providers.WebhookHandshaker).Handshake(ctx, r, conf)
	if !handled || !errors.Is(err, providers.ErrBadSignature) || confirmed {
		t.Fatalf("confirmation without a saved topic: handled %v, confirmed %v, %v", handled, confirmed, err)
	}
	// Another topic than the saved ones is refused the same way.
	p = providerWithTopics(t, env, &confirmed, []string{"arn:aws:sns:us-east-1:123456789012:reputecast-events-other"})
	if handled, _, err := p.(providers.WebhookHandshaker).Handshake(ctx, r, conf); !handled || !errors.Is(err, providers.ErrBadSignature) || confirmed {
		t.Fatalf("confirmation of another topic: handled %v, confirmed %v, %v", handled, confirmed, err)
	}
}

// A subscription link is visited only on a regional SNS endpoint, even in a
// confirmation SNS really signed.
func TestSubscribeURLOnSNSOnly(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	confirmed := false
	p := newProviderWith(t, env, &confirmed).(providers.WebhookHandshaker)
	key, _ := signer(t)
	e := ses.Envelope{Type: "SubscriptionConfirmation", MessageID: "m-1", Token: "tok", TopicArn: topicARN, Message: "You have chosen to subscribe to the topic.",
		SubscribeURL: "https://sns.s3.amazonaws.com/?Action=ConfirmSubscription&Token=tok", Timestamp: "2026-10-12T13:41:02.200Z", SignatureVersion: "2", SigningCertURL: certURL}
	sum := sha256.Sum256([]byte(e.StringToSign()))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	e.Signature = base64.StdEncoding.EncodeToString(sig)
	body, _ := json.Marshal(e)
	r, _ := http.NewRequest(http.MethodPost, "https://reputecast.acme.io/hooks/ses/token", bytes.NewReader(body))
	if handled, _, err := p.Handshake(context.Background(), r, body); !handled || !errors.Is(err, providers.ErrBadSignature) || confirmed {
		t.Fatalf("a subscription link on S3: handled %v, confirmed %v, %v", handled, confirmed, err)
	}
}

// SNS asks the webhook to confirm a subscription as soon as it is made and
// doesn't retry a refusal: provisioning saves the topic before it
// subscribes the webhook, so that confirmation is accepted.
func TestTopicSavedBeforeSubscribing(t *testing.T) {
	ctx := context.Background()
	const hook = "https://reputecast.acme.io/hooks/ses/token"
	topic := "arn:aws:sns:us-east-1:123456789012:reputecast-events"
	aws := newFakeAWS()
	p := queueProvider(t, aws, map[string]any{"events": "", "queue_url": ""}).(providers.Provisioner)
	var saved map[string]any
	var before []string
	in := providers.ProvisionInput{WebhookURL: hook, Save: func(_ context.Context, s map[string]any) error {
		saved, before = s, append([]string(nil), aws.ops...)
		return nil
	}}
	res, err := p.Provision(ctx, "sns_topic", in)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(before) != "[sns:CreateTopic]" || fmt.Sprint(saved["topic_arns"]) != "["+topic+"]" || saved["topic_arn"] != topic {
		t.Fatalf("saved %v after %v", saved, before)
	}
	if aws.sub.Get("TopicArn") != topic || aws.sub.Get("Protocol") != "https" || aws.sub.Get("Endpoint") != hook {
		t.Fatalf("Subscribe %v", aws.sub)
	}
	if fmt.Sprint(res.Settings["topic_arns"]) != "["+topic+"]" || res.Settings["subscription_arn"] != "arn:aws:sns:us-east-1:123456789012:reputecast-events:3f1c" {
		t.Fatalf("result %+v", res.Settings)
	}

	// A topic saved already is subscribed again without a second save.
	aws = newFakeAWS()
	p = queueProvider(t, aws, map[string]any{"events": "", "queue_url": "", "topic_arns": []string{topic}}).(providers.Provisioner)
	in.Save = func(context.Context, map[string]any) error { t.Fatal("saved twice"); return nil }
	if _, err := p.Provision(ctx, "sns_topic", in); err != nil || aws.sub == nil {
		t.Fatalf("subscribed again: %v, %v", aws.sub, err)
	}

	// No subscription when the topic can't be saved first.
	for _, save := range []func(context.Context, map[string]any) error{nil, func(context.Context, map[string]any) error { return errors.New("database gone") }} {
		aws = newFakeAWS()
		p = queueProvider(t, aws, map[string]any{"events": "", "queue_url": ""}).(providers.Provisioner)
		if _, err := p.Provision(ctx, "sns_topic", providers.ProvisionInput{WebhookURL: hook, Save: save}); err == nil || aws.sub != nil {
			t.Fatalf("subscribed without saving the topic: %v, %v", aws.sub, err)
		}
	}
}
