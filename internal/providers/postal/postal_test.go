package postal_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/postal"
)

var (
	keyOnce sync.Once
	key     *rsa.PrivateKey
)

const kid = "1f2e3d4c5b6a79881f2e3d4c5b6a79881f2e3d4c5b6a79881f2e3d4c5b6a7988"

func signingKey(t *testing.T) *rsa.PrivateKey {
	keyOnce.Do(func() {
		var err error
		key, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
	})
	return key
}

func jwks(t *testing.T) json.RawMessage {
	k := signingKey(t)
	b, _ := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes()),
	}}})
	return b
}

func fakeAPI(env *contracttest.Env) http.Handler {
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, status string, data any) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "time": 0.01, "flags": map[string]any{}, "data": data})
	}
	guard := func(w http.ResponseWriter, r *http.Request) bool {
		switch env.Mode() {
		case contracttest.ModeOutage:
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, "502 Bad Gateway")
			return true
		case contracttest.ModeAuth:
			write(w, "error", map[string]string{"code": "InvalidServerAPIKey", "message": "The API token provided in X-Server-API-Key was not valid."})
			return true
		}
		if r.Header.Get("X-Server-API-Key") != "postal-key" {
			write(w, "error", map[string]string{"code": "InvalidServerAPIKey"})
			return true
		}
		return false
	}
	n := 12000
	var mu sync.Mutex
	mux.HandleFunc("/api/v1/send/raw", func(w http.ResponseWriter, r *http.Request) {
		if guard(w, r) {
			return
		}
		var in struct {
			MailFrom string   `json:"mail_from"`
			RcptTo   []string `json:"rcpt_to"`
			Data     string   `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if len(in.RcptTo) == 1 && in.RcptTo[0] == contracttest.RejectAddress {
			write(w, "error", map[string]string{"code": "UnauthenticatedFromAddress", "message": "The From address is not authorised to send mail from this server"})
			return
		}
		mu.Lock()
		n++
		id := n
		mu.Unlock()
		msgs := map[string]any{}
		for _, rc := range in.RcptTo {
			msgs[rc] = map[string]any{"id": id, "token": "tok"}
		}
		write(w, "success", map[string]any{"message_id": "abc@mta.acme.io", "messages": msgs})
	})
	mux.HandleFunc("/api/v1/messages/message", func(w http.ResponseWriter, r *http.Request) {
		if guard(w, r) {
			return
		}
		write(w, "error", map[string]any{"code": "MessageNotFound", "message": "No message found matching provided ID", "id": 0})
	})
	return mux
}

func newProvider(t *testing.T, env *contracttest.Env) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(postal.Settings{ServerURL: env.URL, SigningJWKS: jwks(t), Version: "3.3.4"})
	p, err := postal.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{"api_key":"postal-key"}`)})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func webhook(file string) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.TrimSpace(body)
		sum := sha256.Sum256(body)
		sig, err := rsa.SignPKCS1v15(rand.Reader, signingKey(t), crypto.SHA256, sum[:])
		if err != nil {
			t.Fatal(err)
		}
		r, _ := http.NewRequest(http.MethodPost, "https://reputecast.acme.io/hooks/postal/token", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Postal-Signature-256", base64.StdEncoding.EncodeToString(sig))
		r.Header.Set("X-Postal-Signature-KID", kid)
		r.Header.Set("User-Agent", "Postal/3.3.4")
		return r, body
	}
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Adapter: postal.Adapter,
		API:     fakeAPI,
		New:     newProvider,
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, _ []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != len(msgs) {
				t.Fatalf("want one send per message, got %d requests", len(reqs))
			}
			var in struct {
				Data string `json:"data"`
			}
			_ = json.Unmarshal(reqs[0].Body, &in)
			raw, _ := base64.StdEncoding.DecodeString(in.Data)
			for _, want := range []string{"X-Postal-Tag: rc:" + msgs[0].ID.String(), "List-Unsubscribe-Post: List-Unsubscribe=One-Click", "X-ReputeCast-Message: " + msgs[0].ID.String()} {
				if !strings.Contains(string(raw), want) {
					t.Fatalf("raw message lacks %q:\n%s", want, raw)
				}
			}
		},
		Webhooks: []contracttest.Webhook{
			{Name: "MessageSent", Request: webhook("message-sent.json"), Want: []providers.Event{{
				Type: providers.EventDelivered, Recipient: "a.k@gmail.com", ProviderMessageID: "12345",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e50"), OriginalType: "MessageSent", OriginalCode: "250 2.0.0",
			}}},
			{Name: "MessageDeliveryFailed", Request: webhook("delivery-failed.json"), Want: []providers.Event{{
				Type: providers.EventHardBounce, Recipient: "k.n@hotmail.com", ProviderMessageID: "12346",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e51"), OriginalType: "MessageDeliveryFailed · HardFail", OriginalCode: "550 5.4.1", BounceClass: "policy_block",
			}}},
			{Name: "MessageDelayed", Request: webhook("message-delayed.json"), Want: []providers.Event{{
				Type: providers.EventDeferred, Recipient: "p.s@yahoo.com", ProviderMessageID: "12347",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e52"), OriginalType: "MessageDelayed · SoftFail", OriginalCode: "421 4.7.0", BounceClass: "reputation",
			}}},
		},
	})
}

func TestVerify(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	p := newProvider(t, env).(providers.Verifier)
	q, err := p.Verify(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if q.Account != "Postal 3.3.4 · reachable" {
		t.Fatalf("account: %q", q.Account)
	}
	env.SetMode(contracttest.ModeAuth)
	if _, err := p.Verify(context.Background()); err == nil || !strings.Contains(err.Error(), "InvalidServerAPIKey") {
		t.Fatalf("want an auth error, got %v", err)
	}
}

func TestParseJWKSAndPEM(t *testing.T) {
	keys, err := postal.ParseJWKS(jwks(t))
	if err != nil || keys[kid] == nil || keys[kid].N.Cmp(signingKey(t).N) != 0 {
		t.Fatalf("jwks: %v", err)
	}
	if _, err := postal.ParsePublicKey("not a key"); err == nil {
		t.Fatal("garbage must not parse")
	}
}
