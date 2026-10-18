package kickbox

import (
	"embed"
	"net/http"
	"testing"

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

// fake answers as Kickbox does: 403 for both a bad key and an empty
// balance, the balance in X-Kickbox-Balance on every verify answer.
func fake(env *contracttest.Env) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch env.Mode() {
		case contracttest.ModeOutage:
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`<html>502 Bad Gateway</html>`))
			return
		case contracttest.ModeRateLimited:
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"success":false,"message":"Too many requests"}`))
			return
		case contracttest.ModeAuth:
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write(file("error_key.json"))
			return
		}
		switch r.URL.Path {
		case "/v2/balance":
			_, _ = w.Write(file("balance.json"))
		case "/v2/verify":
			if env.Mode() == contracttest.ModeNoCredits {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write(file("error_balance.json"))
				return
			}
			w.Header().Set("X-Kickbox-Balance", "36655")
			w.Header().Set("X-Kickbox-Response-Time", "312")
			_, _ = w.Write(file(byAddress[r.URL.Query().Get("email")]))
		default:
			http.NotFound(w, r)
		}
	})
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{Service: "kickbox", API: fake})
}

func TestMap(t *testing.T) {
	if a := Map("x@example.com", "undeliverable", "rejected_email", false, false, false); a.Result != adapter.Invalid || a.Code != "undeliverable / rejected_email" {
		t.Errorf("undeliverable: %+v", a)
	}
	if a := Map("x@example.com", "risky", "low_quality", false, false, false); a.Result != adapter.Risky || a.Reason != "low_quality" {
		t.Errorf("risky: %+v", a)
	}
	if a := Map("x@example.com", "deliverable", "accepted_email", true, false, false); a.Reason != adapter.ReasonRole || a.Tag() != "verify:role" {
		t.Errorf("role: %+v", a)
	}
}
