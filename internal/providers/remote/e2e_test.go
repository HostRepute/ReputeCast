package remote_test

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/remote"
)

// The template adapter (sdk/adapter-template) run as a real Node process,
// called through the Go runtime and checked with the shared contract suite:
// the protocol, the SDK and the runtime tested together. It needs Node 24
// and `pnpm install` in sdk/ (CI does both); otherwise it is skipped.

const (
	exampleAPIKey     = "test-example-esp-api-key"
	exampleWebhookKey = "test-example-esp-webhook-signing-key"
)

var templateDir = filepath.Join("..", "..", "..", "sdk", "adapter-template")

// node returns a Node 24+ binary (REPUTECAST_TEST_NODE or the one on PATH).
// Without one the test is skipped, unless REPUTECAST_ADAPTER_E2E=required
// (CI), where that is a failure.
func node(t *testing.T) string {
	t.Helper()
	skip := func(format string, a ...any) {
		t.Helper()
		if os.Getenv("REPUTECAST_ADAPTER_E2E") == "required" {
			t.Fatalf(format, a...)
		}
		t.Skipf(format, a...)
	}
	bin := os.Getenv("REPUTECAST_TEST_NODE")
	if bin == "" {
		var err error
		if bin, err = exec.LookPath("node"); err != nil {
			skip("Node isn't installed: the template adapter's end-to-end test needs Node 24")
		}
	}
	out, err := exec.CommandContext(context.Background(), bin, "--version").Output() //nolint:gosec // the test's own Node binary
	if err != nil {
		skip("node --version: %v", err)
	}
	m := regexp.MustCompile(`^v(\d+)\.`).FindStringSubmatch(strings.TrimSpace(string(out)))
	if m == nil {
		skip("unexpected node version %q", out)
	}
	if major, _ := strconv.Atoi(m[1]); major < 24 {
		skip("Node %s is too old for the adapter SDK; it needs 24 (set REPUTECAST_TEST_NODE)", strings.TrimSpace(string(out)))
	}
	if _, err := os.Stat(filepath.Join(templateDir, "node_modules", "@reputecast", "adapter-sdk", "package.json")); err != nil {
		skip("the template's dependencies aren't installed: run `pnpm install` in sdk/")
	}
	return bin
}

// adapterProcess runs the template with the SDK from source and returns the
// address it listens on.
type adapterProcess struct {
	mu   sync.Mutex
	logs strings.Builder
}

func (p *adapterProcess) log() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.logs.String()
}

func startTemplate(t *testing.T, bin, secret, apiURL string) (string, *adapterProcess) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, "--conditions=reputecast-source", "src/main.ts") //nolint:gosec // the test's own Node binary and template
	cmd.Dir = templateDir
	cmd.Env = append(os.Environ(),
		"PORT=0", "HOST=127.0.0.1",
		"REPUTECAST_ADAPTER_SECRET="+secret,
		"EXAMPLE_ESP_API_URL="+apiURL,
		"REPUTECAST_ADAPTER_ALLOW_HOSTS=127.0.0.1",
		"REPUTECAST_ADAPTER_LOG_LEVEL=info",
	)
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = 5 * time.Second
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()
	})
	proc := &adapterProcess{}
	ready := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			proc.mu.Lock()
			proc.logs.WriteString(line + "\n")
			proc.mu.Unlock()
			var l struct{ Msg, URL string }
			if json.Unmarshal([]byte(line), &l) == nil && l.Msg == "adapter listening" && l.URL != "" {
				select {
				case ready <- l.URL:
				default:
				}
			}
		}
	}()
	select {
	case url := <-ready:
		return url, proc
	case <-time.After(30 * time.Second):
		t.Fatalf("the template adapter didn't start:\n%s", proc.log())
	}
	return "", nil
}

// exampleESP is the Go twin of sdk/adapter-template/test/fake-api.ts.
func exampleESP(env *contracttest.Env) http.Handler {
	var n atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply := func(status int, v any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(v)
		}
		fail := func(status int, code, msg string) {
			reply(status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
		}
		switch {
		case env.Mode() == contracttest.ModeOutage:
			fail(http.StatusServiceUnavailable, "service_unavailable", "Example ESP is down for maintenance.")
		case env.Mode() == contracttest.ModeAuth || r.Header.Get("Authorization") != "Bearer "+exampleAPIKey:
			fail(http.StatusUnauthorized, "invalid_api_key", "The API key is not valid.")
		case r.Method == http.MethodPost && r.URL.Path == "/v1/messages":
			var m struct {
				To []struct{ Email string } `json:"to"`
			}
			_ = json.NewDecoder(r.Body).Decode(&m)
			if len(m.To) > 0 && m.To[0].Email == contracttest.RejectAddress {
				fail(http.StatusUnprocessableEntity, "recipient_suppressed", "The recipient is on Example ESP's suppression list.")
				return
			}
			reply(http.StatusAccepted, map[string]string{"id": fmt.Sprintf("msg_%06d", n.Add(1)), "status": "queued"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/account":
			reply(http.StatusOK, map[string]any{"plan": "Growth", "monthly_limit": 50000, "sent_this_month": 1234, "sandbox": false})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/suppressions":
			if r.URL.Query().Get("cursor") == "" {
				reply(http.StatusOK, map[string]any{"data": []map[string]string{{"email": "Bounced@Example.com", "reason": "bounce", "created_at": "2026-10-11T08:00:00Z"}}, "next_cursor": "page-2"})
				return
			}
			reply(http.StatusOK, map[string]any{"data": []map[string]string{{"email": "complainer@example.com", "reason": "complaint", "created_at": "2026-10-12T09:30:00Z"}}, "next_cursor": nil})
		default:
			fail(http.StatusNotFound, "not_found", "No route "+r.Method+" "+r.URL.Path+".")
		}
	})
}

// fixtures are the template's recorded webhooks, signed the way Example ESP does.
func fixtures(t *testing.T) []contracttest.Webhook {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(templateDir, "test", "fixtures", "webhooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name    string
		Payload json.RawMessage
		Want    []struct {
			Type, Recipient, ProviderMessageID, MessageID, OriginalType, OriginalCode, BounceClass string
		}
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	out := []contracttest.Webhook{}
	for _, c := range cases {
		body := []byte(c.Payload)
		want := []providers.Event{}
		for _, w := range c.Want {
			ev := providers.Event{Type: providers.EventType(w.Type), Recipient: w.Recipient, ProviderMessageID: w.ProviderMessageID,
				OriginalType: w.OriginalType, OriginalCode: w.OriginalCode, BounceClass: w.BounceClass}
			if w.MessageID != "" {
				ev.MessageID = contracttest.UUID(t, w.MessageID)
			}
			want = append(want, ev)
		}
		out = append(out, contracttest.Webhook{Name: c.Name, Want: want, Request: func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
			ts := strconv.FormatInt(time.Now().Unix(), 10)
			mac := hmac.New(sha256.New, []byte(exampleWebhookKey))
			mac.Write([]byte(ts + "."))
			mac.Write(body)
			r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/community:example-esp/0123456789abcdef0123", strings.NewReader(string(body)))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Example-Signature", "t="+ts+",v1="+hex.EncodeToString(mac.Sum(nil)))
			return r, body
		}})
	}
	return out
}

func TestTemplateAdapterEndToEnd(t *testing.T) {
	bin := node(t)
	raw, err := os.ReadFile(filepath.Join(templateDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := remote.ParseManifest(raw)
	if err != nil {
		t.Fatalf("the template's manifest doesn't pass ReputeCast's install check: %v", err)
	}
	register(t, m)

	// The process starts once, before the suite creates its fake API, so it
	// calls a proxy that forwards to that fake once it exists.
	var api atomic.Pointer[contracttest.Env]
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env := api.Load()
		if env == nil || env.URL == "" {
			http.Error(w, "the fake API isn't running yet", http.StatusServiceUnavailable)
			return
		}
		target, _ := url.Parse(env.URL)
		httputil.NewSingleHostReverseProxy(target).ServeHTTP(w, r)
	}))
	t.Cleanup(proxy.Close)
	secret := remote.NewSecret()
	addr, proc := startTemplate(t, bin, secret, proxy.URL)
	defer func() {
		if t.Failed() {
			t.Logf("adapter process log:\n%s", proc.log())
		}
	}()
	in, err := remote.NewInstallation(remote.Options{Manifest: m, Endpoint: addr, Secrets: []string{secret}, AllowPrivate: true,
		PublicURL: "https://reputecast.acme.io", Log: quiet})
	if err != nil {
		t.Fatal(err)
	}
	open := func(t *testing.T, _ *contracttest.Env) providers.Provider {
		creds, _ := json.Marshal(map[string]string{"api_key": exampleAPIKey, "webhook_signing_key": exampleWebhookKey})
		p, err := in.Open(providers.Config{Credentials: creds, WebhookSecret: "whsec-e2e"})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	contracttest.Run(t, contracttest.Suite{
		Adapter: m.Key(),
		API: func(env *contracttest.Env) http.Handler {
			api.Store(env)
			return exampleESP(env)
		},
		New: open,
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			posts := 0
			for _, r := range env.Requests() {
				if r.Method != http.MethodPost || r.Path != "/v1/messages" {
					continue
				}
				posts++
				var body struct {
					Metadata map[string]string `json:"metadata"`
					Headers  map[string]string `json:"headers"`
					Tracking struct{ Opens, Clicks bool }
				}
				if err := json.Unmarshal(r.Body, &body); err != nil {
					t.Fatal(err)
				}
				if body.Tracking.Opens || body.Tracking.Clicks || body.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" || body.Headers[remote.MessageHeader] == "" {
					t.Fatalf("message sent without its headers or with provider tracking: %s", r.Body)
				}
				if r.Header.Get("Authorization") != "Bearer "+exampleAPIKey {
					t.Fatal("the API key didn't reach the provider")
				}
			}
			if posts != len(msgs) {
				t.Fatalf("%d messages sent, %d API calls", len(msgs), posts)
			}
			for i, r := range res {
				if !strings.HasPrefix(r.ProviderMessageID, "msg_") || r.MessageID != msgs[i].ID {
					t.Fatalf("result %d: %+v", i, r)
				}
			}
		},
		Webhooks: fixtures(t),
	})

	// The optional operations the template declares, through the runtime.
	t.Run("Optional", func(t *testing.T) {
		env := api.Load()
		env.SetMode(contracttest.ModeOK)
		p := open(t, env)
		ctx := context.Background()
		h, err := in.Health(ctx)
		if err != nil || h.Adapter != "example-esp" || h.Version != m.Version || !strings.HasPrefix(h.SDK, "@reputecast/adapter-sdk ") {
			t.Fatalf("health %+v %v", h, err)
		}
		q, err := p.(providers.QuotaReporter).Quota(ctx)
		if err != nil || q.Kind != "monthly_plan" || q.Max != 50000 || q.Used != 1234 || q.Account != "Growth · 50,000 per month" {
			t.Fatalf("quota %+v %v", q, err)
		}
		if _, err := p.(providers.Verifier).Verify(ctx); err != nil {
			t.Fatal(err)
		}
		list, err := p.(providers.SuppressionSyncer).ListSuppressions(ctx, time.Now().Add(-24*time.Hour))
		if err != nil || len(list) != 2 || list[0].Email != "bounced@example.com" || list[0].Reason != "hard_bounce" || list[1].Reason != "complaint" {
			t.Fatalf("suppressions %+v %v", list, err)
		}
		before := len(env.Requests())
		if err := p.(providers.SuppressionSyncer).AddSuppressions(ctx, []string{"a@example.com"}); err != nil || len(env.Requests()) != before {
			t.Fatalf("a pull-only adapter must not be asked to add suppressions: %v", err)
		}
		if _, ok := p.(providers.DomainManager); ok {
			t.Fatal("the template doesn't declare domain setup")
		}
		env.SetMode(contracttest.ModeAuth)
		if _, err := p.(providers.Verifier).Verify(ctx); !errors.Is(err, providers.ErrUnauthorized) {
			t.Fatalf("rejected credentials: %v", err)
		}
	})
}
