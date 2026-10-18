package emailable

import (
	"embed"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/verification/adapter"
	"github.com/hostrepute/reputecast/internal/verification/adapter/contracttest"
)

//go:embed testdata
var recorded embed.FS

func file(name string) []byte {
	b, err := recorded.ReadFile("testdata/" + name)
	if err != nil {
		panic(err)
	}
	return b
}

var byAddress = map[string]string{
	contracttest.AddrValid:      "verify_valid.json",
	contracttest.AddrInvalid:    "verify_invalid.json",
	contracttest.AddrCatchAll:   "verify_catchall.json",
	contracttest.AddrRole:       "verify_role.json",
	contracttest.AddrDisposable: "verify_disposable.json",
	contracttest.AddrUnknown:    "verify_unknown.json",
}

// fake answers as Emailable does: 401 for a bad key, 402 without credits.
func fake(env *contracttest.Env) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch env.Mode() {
		case contracttest.ModeOutage:
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		case contracttest.ModeRateLimited:
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"Rate limit exceeded"}`))
			return
		case contracttest.ModeAuth:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write(file("error_unauthorized.json"))
			return
		}
		switch r.URL.Path {
		case "/v1/account":
			_, _ = w.Write(file("account.json"))
		case "/v1/verify":
			if env.Mode() == contracttest.ModeNoCredits {
				w.WriteHeader(http.StatusPaymentRequired)
				_, _ = w.Write(file("error_payment.json"))
				return
			}
			_, _ = w.Write(file(byAddress[r.URL.Query().Get("email")]))
		default:
			http.NotFound(w, r)
		}
	})
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{Service: "emailable", API: fake})
}

// Emailable answers 249 while it's still checking: no answer yet, retried later.
func TestStillChecking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusTryAgain)
		_, _ = w.Write([]byte(`{"message":"Your request is taking longer than normal. Please send your request again."}`))
	}))
	defer srv.Close()
	a, _ := New(adapter.Config{APIKey: "k", BaseURL: srv.URL})
	_, err := a.Verify(t.Context(), "slow@example.com", 3*time.Second)
	if !errors.Is(err, adapter.ErrUnavailable) {
		t.Fatalf("249 = %v, want ErrUnavailable", err)
	}
}
