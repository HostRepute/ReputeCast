package neverbounce

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
	contracttest.AddrValid:      "check_valid.json",
	contracttest.AddrInvalid:    "check_invalid.json",
	contracttest.AddrCatchAll:   "check_catchall.json",
	contracttest.AddrRole:       "check_role.json",
	contracttest.AddrDisposable: "check_disposable.json",
	contracttest.AddrUnknown:    "check_unknown.json",
	contracttest.AddrSpamTrap:   "check_trap.json",
}

// fake answers as NeverBounce does: errors come back with HTTP 200 and a
// status word, except real outages.
func fake(env *contracttest.Env) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch env.Mode() {
		case contracttest.ModeOutage:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(file("temp_unavail.json"))
			return
		case contracttest.ModeRateLimited:
			_, _ = w.Write(file("throttle.json"))
			return
		case contracttest.ModeAuth:
			_, _ = w.Write(file("auth_failure.json"))
			return
		}
		switch r.URL.Path {
		case "/v4.2/account/info":
			_, _ = w.Write(file("account_info.json"))
		case "/v4.2/single/check":
			if env.Mode() == contracttest.ModeNoCredits {
				_, _ = w.Write(file("no_credits.json"))
				return
			}
			_, _ = w.Write(file(byAddress[r.URL.Query().Get("email")]))
		default:
			http.NotFound(w, r)
		}
	})
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{Service: "neverbounce", API: fake, SpamTrap: true})
}
