package zerobounce

import (
	"embed"
	"encoding/json"
	"net/http"
	"strings"
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
	contracttest.AddrValid:      "validate_valid.json",
	contracttest.AddrInvalid:    "validate_invalid.json",
	contracttest.AddrCatchAll:   "validate_catchall.json",
	contracttest.AddrRole:       "validate_role.json",
	contracttest.AddrDisposable: "validate_disposable.json",
	contracttest.AddrUnknown:    "validate_unknown.json",
	contracttest.AddrSpamTrap:   "validate_trap.json",
}

// fake answers as ZeroBounce does: a bad key and an empty balance share one
// error on validate; getcredits answers -1 for a bad key.
func fake(env *contracttest.Env) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mode := env.Mode()
		switch mode {
		case contracttest.ModeOutage:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`<html><body>503 Service Unavailable</body></html>`))
			return
		case contracttest.ModeRateLimited:
			w.Header().Set("Retry-After", "10")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"Too many requests"}`))
			return
		}
		switch r.URL.Path {
		case "/v2/getcredits":
			switch mode {
			case contracttest.ModeAuth:
				_, _ = w.Write(file("getcredits_invalid_key.json"))
			case contracttest.ModeNoCredits:
				_, _ = w.Write(file("getcredits_empty.json"))
			default:
				_, _ = w.Write(file("getcredits.json"))
			}
		case "/v2/validate":
			if mode == contracttest.ModeAuth || mode == contracttest.ModeNoCredits {
				_, _ = w.Write(file("error_key_or_credits.json"))
				return
			}
			_, _ = w.Write(file(byAddress[r.URL.Query().Get("email")]))
		case "/v2/validatebatch":
			if mode == contracttest.ModeAuth || mode == contracttest.ModeNoCredits {
				_, _ = w.Write(file("validatebatch_error.json"))
				return
			}
			var in struct {
				Key   string `json:"api_key"`
				Batch []struct {
					Email string `json:"email_address"`
				} `json:"email_batch"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			out := struct {
				Batch  []json.RawMessage `json:"email_batch"`
				Errors []any             `json:"errors"`
			}{Errors: []any{}}
			for _, e := range in.Batch {
				out.Batch = append(out.Batch, file(byAddress[e.Email]))
			}
			_ = json.NewEncoder(w).Encode(out)
		default:
			http.NotFound(w, r)
		}
	})
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{Service: "zerobounce", API: fake, SpamTrap: true})
}

func TestMap(t *testing.T) {
	cases := []struct {
		status, sub  string
		result       adapter.Result
		reason, code string
	}{
		{"valid", "", adapter.Valid, "", "valid"},
		{"invalid", "mailbox_not_found", adapter.Invalid, adapter.ReasonInvalidAddress, "invalid / mailbox_not_found"},
		{"spamtrap", "", adapter.Invalid, adapter.ReasonSpamTrap, "spamtrap"},
		{"do_not_mail", "possible_trap", adapter.Invalid, adapter.ReasonSpamTrap, "do_not_mail / possible_trap"},
		{"catch-all", "", adapter.Risky, adapter.ReasonCatchAll, "catch-all"},
		{"do_not_mail", "role_based", adapter.Risky, adapter.ReasonRole, "do_not_mail / role_based"},
		{"do_not_mail", "disposable", adapter.Risky, adapter.ReasonDisposable, "do_not_mail / disposable"},
		{"do_not_mail", "toxic", adapter.Risky, "toxic", "do_not_mail / toxic"},
		{"abuse", "", adapter.Risky, "abuse", "abuse"},
		{"unknown", "greylisted", adapter.Unknown, "", "unknown / greylisted"},
	}
	for _, c := range cases {
		a := Map("x@example.com", c.status, c.sub)
		if a.Result != c.result || a.Reason != c.reason || a.Code != c.code {
			t.Errorf("Map(%s, %s) = %+v", c.status, c.sub, a)
		}
	}
	if got := Map("x@example.com", "catch-all", "").Tag(); got != "verify:catch-all" {
		t.Errorf("catch-all tag = %q", got)
	}
	if got := Map("x@example.com", "abuse", "").Tag(); got != "verify:abuse" {
		t.Errorf("abuse tag = %q", got)
	}
}

func TestRegionEndpoints(t *testing.T) {
	for region, want := range map[string]string{"": "https://api.zerobounce.net", "us": "https://api-us.zerobounce.net", "eu": "https://api-eu.zerobounce.net"} {
		a, _ := New(adapter.Config{APIKey: "k", Region: region})
		if got := a.(*client).base; !strings.EqualFold(got, want) {
			t.Errorf("region %q uses %s, want %s", region, got, want)
		}
	}
}
