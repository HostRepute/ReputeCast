package bouncer

import (
	"embed"
	"net/http"
	"testing"

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

// fake answers as Bouncer does: 401 for a bad key, 402 without credits.
func fake(env *contracttest.Env) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("X-Api-Key") != contracttest.Key || env.Mode() == contracttest.ModeAuth {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write(file("error_unauthorized.json"))
			return
		}
		switch env.Mode() {
		case contracttest.ModeOutage:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":500,"message":"Internal Server Error"}`))
			return
		case contracttest.ModeRateLimited:
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"status":429,"message":"Too Many Requests"}`))
			return
		}
		switch r.URL.Path {
		case "/v1.1/credits":
			_, _ = w.Write(file("credits.json"))
		case "/v1.1/email/verify":
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
	contracttest.Run(t, contracttest.Suite{Service: "bouncer", API: fake})
}
