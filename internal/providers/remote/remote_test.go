package remote_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/remote"
	"github.com/hostrepute/reputecast/internal/providers/remote/remotetest"
)

func validManifest() map[string]any {
	var m map[string]any
	_ = json.Unmarshal(remotetest.ManifestJSON(remotetest.Manifest()), &m)
	return m
}

func TestManifestValidation(t *testing.T) {
	raw, _ := json.Marshal(validManifest())
	m, err := remote.ParseManifest(raw)
	if err != nil {
		t.Fatalf("valid manifest: %v", err)
	}
	if m.Key() != "community:fake-esp" || strings.Join(m.Labels(), " · ") != "webhooks · suppression sync · batch 50 · domain setup via API · quota & rate discovery" {
		t.Fatalf("key %s labels %v", m.Key(), m.Labels())
	}
	c := m.ProviderCapabilities()
	if c.Bulk != providers.BulkAllowed || c.Webhooks != "Registered" || c.DomainAPI != "full" || !c.SuppressionSync || c.BatchSize != 50 || !c.QuotaDiscovery || !c.RateDiscovery || c.Pricing != "$0.50 per 1,000" {
		t.Fatalf("capabilities %+v", c)
	}
	d := m.Descriptor(func(providers.Config) (providers.Provider, error) { return nil, nil })
	if d.Kind != "community" || d.Events != "Webhooks" || d.DomainSetup != "API" || d.SuppressionMode != "Two-way" || len(d.Fields) != 3 || !d.Fields[0].Secret || len(d.Terms) != 1 || len(d.Terms[0].Streams) != 3 {
		t.Fatalf("descriptor %+v", d)
	}

	cases := []struct {
		name   string
		change func(m map[string]any)
		want   string
	}{
		{"kind", func(m map[string]any) { m["kind"] = "http" }, "kind is \"http\""},
		{"newer protocol", func(m map[string]any) { m["protocol"] = 2 }, "update ReputeCast"},
		{"id", func(m map[string]any) { m["id"] = "Fake ESP" }, "id must be"},
		{"pricing", func(m map[string]any) { delete(m, "pricing") }, "pricing"},
		{"unknown capability", func(m map[string]any) { m["capabilities"].(map[string]any)["supressionSync"] = "pull" }, "supressionSync is not a capability"},
		{"batch", func(m map[string]any) { m["capabilities"].(map[string]any)["batch"] = 5000 }, "batch"},
		{"suppression value", func(m map[string]any) { m["capabilities"].(map[string]any)["suppressionSync"] = true }, "suppressionSync"},
		{"register without webhooks", func(m map[string]any) { m["capabilities"].(map[string]any)["webhooks"] = false }, "registerWebhooks needs"},
		{"bulk", func(m map[string]any) { m["terms"] = map[string]any{"bulkMarketing": "broadcast_streams_only"} }, "bulkMarketing"},
		{"secret field", func(m map[string]any) {
			m["config"] = []any{map[string]any{"key": "server_token", "label": "Server token"}}
		}, "looks like a secret"},
		{"duplicate field", func(m map[string]any) {
			m["config"] = []any{map[string]any{"key": "region", "label": "Region"}, map[string]any{"key": "region", "label": "Region"}}
		}, "declared twice"},
		{"network host", func(m map[string]any) {
			m["permissions"] = map[string]any{"network": []any{"https://api.example.com/v1"}}
		}, "not a host name"},
		{"network setting", func(m map[string]any) { m["permissions"] = map[string]any{"network": []any{"settings:server_url"}} }, "doesn't declare"},
		{"wrong type", func(m map[string]any) { m["protocol"] = "1" }, "protocol has the wrong type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validManifest()
			tc.change(m)
			raw, _ := json.Marshal(m)
			if _, err := remote.ParseManifest(raw); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
	m2 := validManifest()
	m2["capabilities"] = map[string]any{"webhooks": true, "suppressionSync": "pull", "quotaDiscovery": true, "dedicatedIps": false}
	m2["permissions"] = map[string]any{"network": []any{"api.fake-esp.test", "*.fake-esp.test:443", "settings:region"}}
	raw, _ = json.Marshal(m2)
	if m, err := remote.ParseManifest(raw); err != nil || m.Capabilities.Batch != 1 || m.Capabilities.QuotaDiscovery != "quota" || !m.Capabilities.Pull() || m.Capabilities.Push() {
		t.Fatalf("minimal capabilities: %+v %v", m.Capabilities, err)
	}
}

func TestSignatures(t *testing.T) {
	now := time.Unix(1_760_346_062, 0)
	body := []byte(`{"protocol":1}`)
	h := remote.SignatureHeader([]string{"rcas_new", "rcas_old"}, now.Unix(), body)
	ts := "1760346062"
	for _, secrets := range [][]string{{"rcas_new"}, {"rcas_old"}, {"rcas_other", "rcas_old"}} {
		if err := remote.Verify(secrets, ts, h, body, now); err != nil {
			t.Fatalf("%v: %v", secrets, err)
		}
	}
	if err := remote.Verify([]string{"rcas_other"}, ts, h, body, now); !errors.Is(err, remote.ErrBadSignature) {
		t.Fatalf("wrong secret: %v", err)
	}
	if err := remote.Verify([]string{"rcas_new"}, ts, h, []byte(`{"protocol":2}`), now); !errors.Is(err, remote.ErrBadSignature) {
		t.Fatalf("tampered body: %v", err)
	}
	if err := remote.Verify([]string{"rcas_new"}, ts, h, body, now.Add(6*time.Minute)); !errors.Is(err, remote.ErrStaleTimestamp) {
		t.Fatalf("stale: %v", err)
	}
	if err := remote.Verify([]string{"rcas_new"}, "", h, body, now); !errors.Is(err, remote.ErrNoSignature) {
		t.Fatalf("unsigned: %v", err)
	}
	if s := remote.NewSecret(); !strings.HasPrefix(s, "rcas_") || len(s) < 45 || s == remote.NewSecret() {
		t.Fatalf("secret %q", s)
	}
}

func TestCheckEndpoint(t *testing.T) {
	cases := []struct {
		url          string
		allowPrivate bool
		ok           bool
		want         string
	}{
		{"https://adapters.example.com:4600/", false, true, "https://adapters.example.com:4600"},
		{"https://adapters.example.com/esp", false, true, "https://adapters.example.com/esp"},
		{"http://adapters.example.com", false, false, "use https://"},
		{"http://127.0.0.1:4600", false, false, "allow_private_networks"},
		{"http://127.0.0.1:4600", true, true, "http://127.0.0.1:4600"},
		{"http://example-esp:4600", true, true, "http://example-esp:4600"},
		{"http://8.8.8.8:4600", true, false, "use https:// for a public address"},
		{"http://169.254.169.254/", true, false, "metadata"},
		{"https://user:pass@adapters.example.com", false, false, "credentials"},
		{"https://adapters.example.com/?x=1", false, false, "query"},
		{"ftp://adapters.example.com", false, false, "full URL"},
		{"localhost:4600", true, false, "full URL"},
	}
	for _, c := range cases {
		got, err := remote.CheckEndpoint(c.url, c.allowPrivate)
		if c.ok != (err == nil) || c.ok && got != c.want || !c.ok && !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s (private %v): %q, %v; want %q", c.url, c.allowPrivate, got, err, c.want)
		}
	}
}

// fixture is a fake adapter and an installation pointed at it.
type fixture struct {
	srv     *httptest.Server
	adapter *remotetest.Adapter
	calls   atomic.Int64
	clock   *clock
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{adapter: remotetest.New(testSecret), clock: &clock{t: time.Now()}}
	f.adapter.Now = f.clock.now
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		f.adapter.ServeHTTP(w, r)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fixture) open(t *testing.T, m remote.Manifest, secrets ...string) providers.Provider {
	t.Helper()
	return f.provider(t, f.install(t, m, secrets...))
}

func (f *fixture) install(t *testing.T, m remote.Manifest, secrets ...string) *remote.Installation {
	t.Helper()
	if len(secrets) == 0 {
		secrets = []string{testSecret}
	}
	in, err := remote.NewInstallation(remote.Options{Manifest: m, Endpoint: f.srv.URL, Secrets: secrets, AllowPrivate: true, PublicURL: "https://reputecast.acme.io", Now: f.clock.now, Log: quiet})
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func (f *fixture) provider(t *testing.T, in *remote.Installation) providers.Provider {
	t.Helper()
	creds, _ := json.Marshal(map[string]string{"api_key": "fake-api-key", "webhook_signing_key": signingKey})
	p, err := in.Open(providers.Config{Credentials: creds})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func callErr(t *testing.T, err error, code string) *providers.CallError {
	t.Helper()
	var ce *providers.CallError
	if !errors.As(err, &ce) || ce.Code != code {
		t.Fatalf("want a CallError %q, got %v", code, err)
	}
	return ce
}

func TestAnswersOutsideTheProtocolAreRefused(t *testing.T) {
	msg := []providers.Message{contracttest.Message(0)}
	cases := []struct {
		name  string
		setup func(a *remotetest.Adapter)
	}{
		{"unsigned", func(a *remotetest.Adapter) { a.Unsigned = true }},
		{"another secret", func(a *remotetest.Adapter) { a.AnswerSecret = "rcas_someone-else" }},
		{"another request", func(a *remotetest.Adapter) {
			a.Rewrite = func(_ string, b []byte) []byte {
				return bytes.Replace(b, []byte(`"idempotencyKey":"send-`), []byte(`"idempotencyKey":"xend-`), 1)
			}
		}},
		{"not JSON", func(a *remotetest.Adapter) {
			a.Rewrite = func(string, []byte) []byte { return []byte("<html>oops</html>") }
		}},
		{"missing result", func(a *remotetest.Adapter) {
			a.Rewrite = func(_ string, b []byte) []byte {
				var m map[string]any
				_ = json.Unmarshal(b, &m)
				delete(m, "result")
				out, _ := json.Marshal(m)
				return out
			}
		}},
		{"result for another message", func(a *remotetest.Adapter) {
			a.Rewrite = func(_ string, b []byte) []byte {
				return bytes.Replace(b, []byte(`"messageId":"`+msg[0].ID.String()), []byte(`"messageId":"0199d5a8-0000-7000-8000-000000000000`), 1)
			}
		}},
		{"too many results", func(a *remotetest.Adapter) {
			a.Rewrite = func(_ string, b []byte) []byte {
				return bytes.Replace(b, []byte(`"results":[`), []byte(`"results":[{"messageId":"x","providerMessageId":"y"},`), 1)
			}
		}},
		{"old protocol", func(a *remotetest.Adapter) {
			a.Rewrite = func(_ string, b []byte) []byte {
				return bytes.Replace(b, []byte(`"protocol":1`), []byte(`"protocol":0`), 1)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.setup(f.adapter)
			p := f.open(t, remotetest.Manifest())
			_, err := p.Send(context.Background(), msg)
			ce := callErr(t, err, "adapter_invalid_response")
			if !ce.Temporary || !errors.Is(err, providers.ErrUnavailable) {
				t.Fatalf("an invalid answer is an adapter outage: %+v", ce)
			}
		})
	}
}

func TestContentTypeAndSizeLimits(t *testing.T) {
	sign := func(w http.ResponseWriter, body []byte) {
		ts := time.Now().Unix()
		w.Header().Set(remote.HeaderTimestamp, strconv.FormatInt(ts, 10))
		w.Header().Set(remote.HeaderSignature, "v1="+remote.Sign(testSecret, ts, body))
	}
	for name, h := range map[string]http.HandlerFunc{
		"html": func(w http.ResponseWriter, r *http.Request) {
			body := []byte(`{"protocol":1}`)
			sign(w, body)
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write(body)
		},
		"too large": func(w http.ResponseWriter, r *http.Request) {
			body := []byte(`{"protocol":1,"pad":"` + strings.Repeat("x", 70<<10) + `"}`)
			sign(w, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			in, err := remote.NewInstallation(remote.Options{Manifest: remotetest.Manifest(), Endpoint: srv.URL, Secrets: []string{testSecret}, AllowPrivate: true, Log: quiet})
			if err != nil {
				t.Fatal(err)
			}
			_, err = in.Health(context.Background())
			_ = callErr(t, err, "adapter_invalid_response")
		})
	}
}

func TestProviderErrorsKeepTheirMeaning(t *testing.T) {
	f := newFixture(t)
	mode := ""
	f.adapter.Mode = func() string { return mode }
	p := f.open(t, remotetest.Manifest())
	q := p.(providers.QuotaReporter)
	got, err := q.Quota(context.Background())
	if err != nil || got.Kind != "monthly_plan" || got.Max != 50000 || got.Used != 1234 || got.MaxRate != 14 || got.Account == "" {
		t.Fatalf("quota %+v %v", got, err)
	}
	mode = contracttest.ModeAuth
	_, err = q.Quota(context.Background())
	if ce := callErr(t, err, "invalid_api_key"); !ce.Auth || ce.Status != 401 || !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("auth: %+v", ce)
	}
	mode = contracttest.ModeOutage
	_, err = p.(providers.Verifier).Verify(context.Background())
	if ce := callErr(t, err, "service_unavailable"); !ce.Temporary || !errors.Is(err, providers.ErrUnavailable) || ce.Op != "verify" {
		t.Fatalf("outage: %+v", ce)
	}
	// Provider failures don't open the adapter's circuit breaker.
	for range remote.BreakerThreshold + 1 {
		_, _ = q.Quota(context.Background())
	}
	mode = ""
	if _, err := q.Quota(context.Background()); err != nil {
		t.Fatalf("provider outages must not trip the adapter's breaker: %v", err)
	}
}

func TestTimeoutsAndCircuitBreaker(t *testing.T) {
	f := newFixture(t)
	in := f.install(t, remotetest.Manifest())
	p := f.provider(t, in)
	f.adapter.SetDelay(2 * time.Second)
	for i := range remote.BreakerThreshold {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_, err := p.(providers.Verifier).Verify(ctx)
		cancel()
		if ce := callErr(t, err, "adapter_unreachable"); !strings.Contains(ce.Message, "timed out") || !ce.Temporary {
			t.Fatalf("call %d: %+v", i, ce)
		}
	}
	if state, until, last := in.BreakerState(); state != remote.BreakerOpen || until.IsZero() || last == "" {
		t.Fatalf("breaker %s %v %q", state, until, last)
	}
	before := f.calls.Load()
	start := time.Now()
	_, err := p.Send(context.Background(), []providers.Message{contracttest.Message(0)})
	if ce := callErr(t, err, "adapter_unavailable"); !errors.Is(err, providers.ErrUnavailable) || time.Since(start) > time.Second || f.calls.Load() != before {
		t.Fatalf("an open breaker must fail at once without calling the adapter: %+v", ce)
	}
	// After the pause, one probe goes through; its success closes the breaker.
	f.adapter.SetDelay(0)
	f.clock.add(remote.BreakerCooldown + time.Second)
	if state, _, _ := in.BreakerState(); state != remote.BreakerHalfOpen {
		t.Fatalf("breaker %s after the cool-down", state)
	}
	if _, err := p.(providers.Verifier).Verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	if state, _, _ := in.BreakerState(); state != remote.BreakerClosed {
		t.Fatalf("breaker %s after a good probe", state)
	}
	// A check by an admin probes even while the breaker is open.
	f.adapter.SetDelay(2 * time.Second)
	for range remote.BreakerThreshold {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_, _ = p.(providers.Verifier).Verify(ctx)
		cancel()
	}
	f.adapter.SetDelay(0)
	if h, err := in.Health(context.Background()); err != nil || h.Adapter != "fake-esp" || h.Version != "1.2.3" {
		t.Fatalf("health %+v %v", h, err)
	}
	if state, _, _ := in.BreakerState(); state != remote.BreakerClosed {
		t.Fatalf("breaker %s after a good health check", state)
	}
}

func TestUnreachableAdapter(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	in, err := remote.NewInstallation(remote.Options{Manifest: remotetest.Manifest(), Endpoint: url, Secrets: []string{testSecret}, AllowPrivate: true, Log: quiet})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := in.Open(providers.Config{})
	body, h := remotetest.WebhookBody(signingKey, remotetest.Event{Event: "delivered", Recipient: "a@example.com", Timestamp: "2026-10-12T13:41:02Z"})
	r := httptest.NewRequest(http.MethodPost, "/hooks/community:fake-esp/0123456789abcdef0123", bytes.NewReader(body))
	r.Header = h
	err = p.(providers.WebhookVerifier).VerifyWebhook(context.Background(), r, body)
	if !errors.Is(err, providers.ErrUnavailable) || errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("an unreachable adapter is not a bad signature: %v", err)
	}
	if _, err := p.ParseWebhook(providers.WithSignatureVerified(context.Background()), r, body); !errors.Is(err, providers.ErrUnavailable) {
		t.Fatalf("parse: %v", err)
	}
	_ = callErr(t, func() error {
		_, err := p.Send(context.Background(), []providers.Message{contracttest.Message(0)})
		return err
	}(), "adapter_unreachable")

	none, _ := remote.NewInstallation(remote.Options{Manifest: remotetest.Manifest(), Secrets: []string{testSecret}, Log: quiet})
	p2, _ := none.Open(providers.Config{})
	_, err = p2.(providers.Verifier).Verify(context.Background())
	if ce := callErr(t, err, "adapter_not_configured"); !strings.Contains(ce.Message, "reputecast adapter set-url fake-esp") {
		t.Fatalf("not configured: %+v", ce)
	}
}

func TestSendIsIdempotentAndSplit(t *testing.T) {
	f := newFixture(t)
	p := f.open(t, remotetest.Manifest())
	msgs := []providers.Message{contracttest.Message(0), contracttest.Message(1)}
	for range 2 {
		res, err := p.Send(context.Background(), msgs)
		if err != nil || len(res) != 2 || res[0].ProviderMessageID == "" {
			t.Fatalf("send %+v %v", res, err)
		}
	}
	for id, n := range f.adapter.Sent() {
		if n != 1 {
			t.Fatalf("message %s was handed to the provider %d times: a retried batch must reuse its idempotency key", id, n)
		}
	}

	// A batch larger than one request goes out in several calls; when a
	// later call fails, the accepted messages stand and the rest are retried.
	big := make([]providers.Message, 3)
	for i := range big {
		big[i] = contracttest.Message(i)
		big[i].HTML = strings.Repeat("x", 9<<20)
	}
	sends := 0
	f.adapter.Rewrite = func(op string, b []byte) []byte {
		if op != remote.OpSend {
			return b
		}
		sends++
		if sends == 1 {
			return b
		}
		var r map[string]any
		_ = json.Unmarshal(b, &r)
		delete(r, "result")
		r["error"] = map[string]any{"code": "provider_unavailable", "message": "Fake ESP answered 503.", "status": 503}
		out, _ := json.Marshal(r)
		return out
	}
	res, err := p.Send(context.Background(), big)
	if err != nil || len(res) != 3 {
		t.Fatalf("split send: %v %+v", err, len(res))
	}
	if res[0].Err != nil || res[1].Err != nil || res[2].Err == nil || !res[2].Err.Temporary || sends != 2 {
		t.Fatalf("results %+v %+v %+v after %d calls", res[0].Err, res[1].Err, res[2].Err, sends)
	}
	// When nothing was accepted, the whole call fails.
	sends = 1
	_, err = p.Send(context.Background(), big[2:])
	_ = callErr(t, err, "")
}

func TestHandshakeAndVerificationShareOneCall(t *testing.T) {
	f := newFixture(t)
	p := f.open(t, remotetest.Manifest())
	body := []byte(`{"type":"handshake","challenge":"c-42"}`)
	r := httptest.NewRequest(http.MethodPost, "/hooks/community:fake-esp/0123456789abcdef0123", bytes.NewReader(body))
	r.Header.Set(remotetest.SignatureHeader, remotetest.SignWebhook(signingKey, body))
	handled, resp, err := p.(providers.WebhookHandshaker).Handshake(context.Background(), r, body)
	if err != nil || !handled || string(resp) != `{"challenge":"c-42"}` {
		t.Fatalf("handshake %v %s %v", handled, resp, err)
	}
	ev, h := remotetest.WebhookBody(signingKey, remotetest.Event{Event: "delivered", Recipient: "a@example.com", Timestamp: "2026-10-12T13:41:02Z"})
	r2 := httptest.NewRequest(http.MethodPost, "/hooks/community:fake-esp/0123456789abcdef0123", bytes.NewReader(ev))
	r2.Header = h
	if handled, _, err := p.(providers.WebhookHandshaker).Handshake(context.Background(), r2, ev); handled || err != nil {
		t.Fatalf("an event is not a handshake: %v %v", handled, err)
	}
	if err := p.(providers.WebhookVerifier).VerifyWebhook(context.Background(), r2, ev); err != nil {
		t.Fatal(err)
	}
	verifies, url := 0, ""
	for _, c := range f.adapter.Calls() {
		if c.Operation == remote.OpWebhookVerify {
			verifies++
			var in remote.WebhookVerifyInput
			_ = json.Unmarshal(c.Input, &in)
			url = in.Request.URL
		}
	}
	if verifies != 2 || url != "https://reputecast.acme.io/hooks/community:fake-esp/0123456789abcdef0123" {
		t.Fatalf("%d verify calls for two requests (handshake then verify share one); forwarded URL %q", verifies, url)
	}
}

func TestSecretRotation(t *testing.T) {
	f := newFixture(t)
	// The process still has the previous secret: calls signed with both work,
	// and so do its answers signed with the previous one.
	f.adapter.Secrets = []string{"rcas_previous"}
	p := f.open(t, remotetest.Manifest(), "rcas_current", "rcas_previous")
	if _, err := p.(providers.Verifier).Verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Once the previous secret expires, the process must have the new one.
	p2 := f.open(t, remotetest.Manifest(), "rcas_current")
	_, err := p2.(providers.Verifier).Verify(context.Background())
	if ce := callErr(t, err, "adapter_unauthenticated"); !strings.Contains(ce.Message, "reputecast adapter secret fake-esp") {
		t.Fatalf("%+v", ce)
	}
}

func TestOptionalOperations(t *testing.T) {
	f := newFixture(t)
	p := f.open(t, remotetest.Manifest())
	list, err := p.(providers.SuppressionSyncer).ListSuppressions(context.Background(), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(list) != 2 || list[0].Email != "bounced@example.com" || list[0].Reason != "hard_bounce" || list[0].CreatedAt.IsZero() || list[1].Reason != "complaint" {
		t.Fatalf("suppressions %+v %v", list, err)
	}
	emails := make([]string, 1200)
	for i := range emails {
		emails[i] = "a" + strconv.Itoa(i) + "@example.com"
	}
	if err := p.(providers.SuppressionSyncer).AddSuppressions(context.Background(), emails); err != nil || len(f.adapter.Pushed()) != 1200 {
		t.Fatalf("push %v %d", err, len(f.adapter.Pushed()))
	}
	adds := 0
	for _, c := range f.adapter.Calls() {
		if c.Operation == remote.OpSuppressionsAdd {
			adds++
		}
	}
	if adds != 3 {
		t.Fatalf("%d suppressions/add calls for 1,200 addresses, want 3", adds)
	}
	recs, err := p.(providers.DomainManager).AddDomain(context.Background(), "News.Acme.io")
	if err != nil || len(recs) != 2 || recs[0].Name != "fk1._domainkey.news.acme.io" || recs[0].Status != "pending" {
		t.Fatalf("domain %+v %v", recs, err)
	}
	if recs, err = p.(providers.DomainManager).DomainStatus(context.Background(), "news.acme.io"); err != nil || recs[1].Status != "verified" {
		t.Fatalf("status %+v %v", recs, err)
	}
	if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), webhookPath); err != nil || f.adapter.Webhook() != webhookPath {
		t.Fatalf("register %v %q", err, f.adapter.Webhook())
	}

	// A pull-only adapter is never asked to add, a push-only one never to list.
	pull := remotetest.Manifest()
	pull.Capabilities.SuppressionSync = "pull"
	before := len(f.adapter.Calls())
	if err := f.open(t, pull).(providers.SuppressionSyncer).AddSuppressions(context.Background(), emails); err != nil || len(f.adapter.Calls()) != before {
		t.Fatal("pull-only adapters must not receive pushes")
	}
	push := remotetest.Manifest()
	push.Capabilities.SuppressionSync = "push"
	if l, err := f.open(t, push).(providers.SuppressionSyncer).ListSuppressions(context.Background(), time.Time{}); err != nil || len(l) != 0 || len(f.adapter.Calls()) != before {
		t.Fatal("push-only adapters must not be listed")
	}
}

func TestHealthChecksTheInstalledAdapter(t *testing.T) {
	f := newFixture(t)
	// A process answering as another adapter (wrong address, wrong image).
	f.adapter.Rewrite = func(op string, b []byte) []byte {
		if op == remote.OpHealth {
			return bytes.Replace(b, []byte(`"adapter":"fake-esp"`), []byte(`"adapter":"other-esp"`), 1)
		}
		return b
	}
	in := f.install(t, remotetest.Manifest())
	_, err := in.Health(context.Background())
	if ce := callErr(t, err, "adapter_invalid_response"); !strings.Contains(ce.Message, `"other-esp" adapter, not "fake-esp"`) {
		t.Fatalf("%+v", ce)
	}
}

func TestInvalidEventsAreRefused(t *testing.T) {
	for name, rewrite := range map[string]func([]byte) []byte{
		"type": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"type":"delivered"`), []byte(`"type":"arrived"`), 1)
		},
		"time": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"occurredAt":"2026-10-12T13:41:02Z"`), []byte(`"occurredAt":"yesterday"`), 1)
		},
		"raw": func(b []byte) []byte { return bytes.Replace(b, []byte(`"raw":{`), []byte(`"raw":null,"x":{`), 1) },
		"message": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"recipient":"a@example.com"`), []byte(`"recipient":"a@example.com","messageId":"msg-1"`), 1)
		},
		"no email": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"recipient":"a@example.com"`), []byte(`"recipient":""`), 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.adapter.Rewrite = func(op string, b []byte) []byte {
				if op == remote.OpWebhookParse {
					return rewrite(b)
				}
				return b
			}
			p := f.open(t, remotetest.Manifest())
			body, h := remotetest.WebhookBody(signingKey, remotetest.Event{Event: "delivered", Recipient: "a@example.com", Timestamp: "2026-10-12T13:41:02Z"})
			r := httptest.NewRequest(http.MethodPost, "/hooks/community:fake-esp/x", bytes.NewReader(body))
			r.Header = h
			_, err := p.ParseWebhook(context.Background(), r, body)
			if !errors.Is(err, providers.ErrUnavailable) {
				t.Fatalf("an invalid event must be refused (and retried later): %v", err)
			}
		})
	}
}

func TestMessagesOnTheWire(t *testing.T) {
	f := newFixture(t)
	p := f.open(t, remotetest.Manifest())
	m := contracttest.Message(0)
	m.ReplyTo = &mail.Address{Name: "Help", Address: "help@acme.io"}
	m.Stream = providers.StreamSystem
	if _, err := p.Send(context.Background(), []providers.Message{m}); err != nil {
		t.Fatal(err)
	}
	var in remote.SendInput
	_ = json.Unmarshal(f.adapter.Calls()[0].Input, &in)
	w := in.Messages[0]
	if w.Stream != "system" || w.ReplyTo == nil || w.ReplyTo.Address != "help@acme.io" || w.From.Name != "Acme Newsletter" || w.Subject != m.Subject || w.Tags["campaign"] != "contract" {
		t.Fatalf("message %+v", w)
	}
	if _, err := p.Send(context.Background(), make([]providers.Message, 51)); err == nil {
		t.Fatal("a batch larger than declared must be refused")
	}
}
