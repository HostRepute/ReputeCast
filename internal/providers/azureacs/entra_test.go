package azureacs_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/azureacs"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
)

const (
	tenant   = "8a1b2c3d-0000-4000-8000-00000000c0de"
	audience = "api://reputecast-webhook"
)

func entraToken(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	head := enc(map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid})
	body := enc(claims)
	sum := sha256.Sum256([]byte(head + "." + body))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return head + "." + body + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestEntraProtectedDelivery(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var fetches atomic.Int32
	env := contracttest.NewEnv(t, func(*contracttest.Env) http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /"+tenant+"/discovery/v2.0/keys", func(w http.ResponseWriter, _ *http.Request) {
			fetches.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
				"kty": "RSA", "use": "sig", "kid": "key-1",
				"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
			}}})
		})
		return mux
	})
	settings, _ := json.Marshal(azureacs.Settings{Endpoint: "https://acme.communication.azure.com", EventGridAudience: audience, EventGridTenantID: tenant, AuthorityURL: env.URL})
	p, err := azureacs.New(providers.Config{Settings: settings, WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	v := p.(providers.WebhookVerifier)
	valid := func() map[string]any {
		return map[string]any{"iss": "https://sts.windows.net/" + tenant + "/", "aud": audience, "tid": tenant, "roles": []string{azureacs.EventGridRole},
			"appid": "4962773b-9cdb-44cf-a8bf-237846a00ab7", "exp": time.Now().Add(time.Hour).Unix(), "nbf": time.Now().Add(-time.Minute).Unix()}
	}
	check := func(token string) error {
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/azureacs/token", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		return v.VerifyWebhook(context.Background(), r, nil)
	}
	if err := check(entraToken(t, key, "key-1", valid())); err != nil {
		t.Fatalf("a valid Event Grid token: %v", err)
	}
	v2 := valid()
	v2["iss"], v2["aud"] = "https://login.microsoftonline.com/"+tenant+"/v2.0", "reputecast-webhook"
	if err := check(entraToken(t, key, "key-1", v2)); err != nil {
		t.Fatalf("a v2 token with the application ID: %v", err)
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	bad := map[string]func(c map[string]any) (string, string){
		"expired": func(c map[string]any) (string, string) {
			c["exp"] = time.Now().Add(-time.Hour).Unix()
			return "key-1", ""
		},
		"other tenant": func(c map[string]any) (string, string) { c["tid"] = "other"; return "key-1", "" },
		"other app":    func(c map[string]any) (string, string) { c["aud"] = "api://someone-else"; return "key-1", "" },
		"no role":      func(c map[string]any) (string, string) { c["roles"] = []string{"Reader"}; return "key-1", "" },
		"wrong issuer": func(c map[string]any) (string, string) {
			c["iss"] = "https://evil.example/" + tenant + "/"
			return "key-1", ""
		},
		"unknown key": func(map[string]any) (string, string) { return "key-2", "" },
		"forged":      func(map[string]any) (string, string) { return "key-1", "forged" },
		"not yet valid": func(c map[string]any) (string, string) {
			c["nbf"] = time.Now().Add(time.Hour).Unix()
			return "key-1", ""
		},
	}
	for name, mutate := range bad {
		c := valid()
		kid, mode := mutate(c)
		signer := key
		if mode == "forged" {
			signer = other
		}
		if err := check(entraToken(t, signer, kid, c)); !errors.Is(err, providers.ErrBadSignature) {
			t.Fatalf("%s: must be refused, got %v", name, err)
		}
	}
	if err := check("not.a.jwt"); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("garbage: %v", err)
	}
	if n := fetches.Load(); n > 2 {
		t.Fatalf("the tenant's keys are cached: %d fetches", n)
	}
	// Without a token, the shared secret still works.
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/azureacs/token", nil)
	r.Header.Set(azureacs.SecretHeader, secret)
	if err := v.VerifyWebhook(context.Background(), r, nil); err != nil {
		t.Fatalf("secret header: %v", err)
	}
}
