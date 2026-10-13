package remote_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/remote"
	"github.com/hostrepute/reputecast/internal/providers/remote/remotetest"
)

const (
	testSecret  = "rcas_test-secret-for-the-contract-suite"
	signingKey  = "fake-webhook-signing-key"
	webhookPath = "https://reputecast.acme.io/hooks/community:fake-esp/0123456789abcdef0123"
)

// register puts the fake adapter in the provider registry for the
// contract suite's registration checks.
func register(t *testing.T, m remote.Manifest) {
	t.Helper()
	owner := &struct{ name string }{t.Name()}
	d := m.Descriptor(func(providers.Config) (providers.Provider, error) { return nil, nil })
	if err := providers.SetRuntime(owner, []providers.Descriptor{d}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = providers.SetRuntime(owner, nil) })
}

// open builds a provider for a connection made through the fake adapter at url.
func open(t *testing.T, m remote.Manifest, url string, secrets ...string) providers.Provider {
	t.Helper()
	if len(secrets) == 0 {
		secrets = []string{testSecret}
	}
	in, err := remote.NewInstallation(remote.Options{Manifest: m, Endpoint: url, Secrets: secrets, AllowPrivate: true, PublicURL: "https://reputecast.acme.io"})
	if err != nil {
		t.Fatal(err)
	}
	settings, _ := json.Marshal(map[string]any{"region": "eu", "internal_note": "never sent"})
	creds, _ := json.Marshal(map[string]any{"api_key": "fake-api-key", "webhook_signing_key": signingKey, "undeclared": "never sent"})
	p, err := in.Open(providers.Config{Settings: settings, Credentials: creds, WebhookSecret: "whsec-fake"})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func fakeAdapter(env *contracttest.Env) http.Handler {
	a := remotetest.New(testSecret)
	a.Mode = env.Mode
	return a
}

func webhook(events ...remotetest.Event) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
		body, h := remotetest.WebhookBody(signingKey, events...)
		r, _ := http.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(body))
		r.Header = h
		return r, body
	}
}

func TestContract(t *testing.T) {
	m := remotetest.Manifest()
	register(t, m)
	contracttest.Run(t, contracttest.Suite{
		Adapter: m.Key(),
		API:     fakeAdapter,
		New:     func(t *testing.T, env *contracttest.Env) providers.Provider { return open(t, m, env.URL) },
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, _ []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || reqs[0].Path != "/v1/send" {
				t.Fatalf("want one POST /v1/send, got %+v", reqs)
			}
			h := reqs[0].Header
			if err := remote.Verify([]string{testSecret}, h.Get(remote.HeaderTimestamp), h.Get(remote.HeaderSignature), reqs[0].Body, time.Now()); err != nil {
				t.Fatalf("the request is not signed with the installation's secret: %v", err)
			}
			if err := remote.Verify([]string{"rcas_another"}, h.Get(remote.HeaderTimestamp), h.Get(remote.HeaderSignature), reqs[0].Body, time.Now()); err == nil {
				t.Fatal("the signature must depend on the secret")
			}
			var env1 struct {
				Operation      string             `json:"operation"`
				Adapter        string             `json:"adapter"`
				IdempotencyKey string             `json:"idempotencyKey"`
				Connection     *remote.Connection `json:"connection"`
				Input          remote.SendInput   `json:"input"`
			}
			if err := json.Unmarshal(reqs[0].Body, &env1); err != nil {
				t.Fatal(err)
			}
			if env1.Operation != "send" || env1.Adapter != "fake-esp" || h.Get(remote.HeaderIdempotency) != env1.IdempotencyKey || env1.IdempotencyKey == "" {
				t.Fatalf("envelope %+v, headers %v", env1, h)
			}
			c := env1.Connection
			if c.Credentials["api_key"] != "fake-api-key" || c.Credentials["undeclared"] != nil || c.Settings["region"] != "eu" || c.Settings["internal_note"] != nil || c.WebhookSecret != "whsec-fake" {
				t.Fatalf("only declared connection values may be sent: %+v", c)
			}
			for i, m := range env1.Input.Messages {
				if m.ID != msgs[i].ID.String() || m.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" || m.Headers[remote.MessageHeader] != msgs[i].ID.String() {
					t.Fatalf("message %d lost its id or headers: %+v", i, m)
				}
			}
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			r.Header.Set(remotetest.SignatureHeader, remotetest.SignWebhook("another-key", body))
			return r, body
		},
		Webhooks: []contracttest.Webhook{
			{Name: "Delivered", Request: webhook(remotetest.Event{Event: "delivered", MessageID: "fake-1", Recipient: "D.F@outlook.com", Timestamp: "2026-10-12T13:41:02Z",
				SMTPCode: "250 2.6.0", RCMessage: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"}),
				Want: []providers.Event{{Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "fake-1",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "delivered", OriginalCode: "250 2.6.0"}}},
			{Name: "HardBounce", Request: webhook(remotetest.Event{Event: "bounced", BounceType: "hard", MessageID: "fake-2", Recipient: "john@example.com", Timestamp: "2026-10-12T13:41:02Z",
				SMTPCode: "550 5.1.1", Description: "The email account that you tried to reach does not exist.", RCMessage: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"}),
				Want: []providers.Event{{Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "fake-2",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "bounced · hard", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown"}}},
			{Name: "Batch", Request: webhook(
				remotetest.Event{Event: "bounced", BounceType: "soft", MessageID: "fake-3", Recipient: "b.h@yahoo.com", Timestamp: "2026-10-12T13:41:02Z", SMTPCode: "452 4.2.2", Description: "Mailbox full"},
				remotetest.Event{Event: "complained", MessageID: "fake-4", Recipient: "m.g@aol.com", Timestamp: "2026-10-12T13:42:00Z"},
				remotetest.Event{Event: "opened", MessageID: "fake-5", Recipient: "x@example.com", Timestamp: "2026-10-12T13:43:00Z"}),
				Want: []providers.Event{
					{Type: providers.EventSoftBounce, Recipient: "b.h@yahoo.com", ProviderMessageID: "fake-3", OriginalType: "bounced · soft", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full"},
					{Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "fake-4", OriginalType: "complained"},
				}},
		},
	})
}

// Every capability combination exposes exactly the declared optional
// interfaces: core code decides what to do from them.
func TestMethodSetFollowsTheManifest(t *testing.T) {
	for mask := range 16 {
		m := remotetest.Manifest()
		c := remote.Capabilities{Batch: 1, Webhooks: true}
		if mask&1 != 0 {
			c.QuotaDiscovery = "quota"
		}
		if mask&2 != 0 {
			c.SuppressionSync = "pull"
		}
		if mask&4 != 0 {
			c.DomainSetup = "dkim_only"
		}
		if mask&8 != 0 {
			c.RegisterWebhooks = true
		}
		m.Capabilities = c
		p := open(t, m, "http://127.0.0.1:1")
		_, q := p.(providers.QuotaReporter)
		_, s := p.(providers.SuppressionSyncer)
		_, d := p.(providers.DomainManager)
		_, r := p.(providers.WebhookRegistrar)
		if q != (mask&1 != 0) || s != (mask&2 != 0) || d != (mask&4 != 0) || r != (mask&8 != 0) {
			t.Fatalf("mask %d: quota %v suppressions %v domains %v register %v", mask, q, s, d, r)
		}
		for _, ok := range []bool{is[providers.Verifier](p), is[providers.WebhookVerifier](p), is[providers.WebhookHandshaker](p)} {
			if !ok {
				t.Fatalf("mask %d: a community adapter always verifies credentials and webhooks", mask)
			}
		}
		if p.Adapter() != "community:fake-esp" || p.Capabilities().BatchSize != 1 {
			t.Fatalf("mask %d: %s %+v", mask, p.Adapter(), p.Capabilities())
		}
	}
}

func is[T any](p providers.Provider) bool {
	_, ok := p.(T)
	return ok
}
