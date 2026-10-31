package ses_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
)

// dedicatedIPs is a fake GetDedicatedIps: answer gets the request's
// NextToken and writes the response.
func dedicatedIPs(answer func(w http.ResponseWriter, token string)) func(env *contracttest.Env) http.Handler {
	return func(*contracttest.Env) http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /v2/email/dedicated-ips", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			answer(w, r.URL.Query().Get("NextToken"))
		})
		return mux
	}
}

func awsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"__type": code, "message": msg})
}

func sendingIPs(t *testing.T, env *contracttest.Env) (providers.SendingIPs, error) {
	t.Helper()
	return newProvider(t, env).(providers.IPReporter).SendingIPs(context.Background())
}

// Every page of GetDedicatedIps counts: the paginator follows NextToken,
// and the list comes back each IP once, in address order.
func TestSendingIPsFollowsPages(t *testing.T) {
	env := contracttest.NewEnv(t, dedicatedIPs(func(w http.ResponseWriter, token string) {
		switch token {
		case "":
			_, _ = io.WriteString(w, `{"DedicatedIps":[{"Ip":"203.0.113.40","WarmupStatus":"DONE","WarmupPercentage":100}],"NextToken":"b"}`)
		case "b":
			_, _ = io.WriteString(w, `{"DedicatedIps":[{"Ip":"198.51.100.7","WarmupStatus":"IN_PROGRESS","WarmupPercentage":20}],"NextToken":"c"}`)
		default:
			_, _ = io.WriteString(w, `{"DedicatedIps":[{"Ip":"203.0.113.40","WarmupStatus":"DONE","WarmupPercentage":100}]}`)
		}
	}))
	got, err := sendingIPs(t, env)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Dedicated || !slices.Equal(got.IPs, []string{"198.51.100.7", "203.0.113.40"}) {
		t.Fatalf("SendingIPs = %+v, want the three pages' two IPs", got)
	}
	var tokens []string
	for _, r := range env.Requests() {
		_, q, _ := strings.Cut(r.Path, "?")
		tokens = append(tokens, q)
	}
	if !slices.Equal(tokens, []string{"", "NextToken=b", "NextToken=c"}) {
		t.Fatalf("requests %q, want one per page", tokens)
	}
}

// An account without dedicated IPs sends from SES's shared pool.
func TestSendingIPsSharedPool(t *testing.T) {
	env := contracttest.NewEnv(t, dedicatedIPs(func(w http.ResponseWriter, _ string) {
		_, _ = io.WriteString(w, `{"DedicatedIps":[]}`)
	}))
	got, err := sendingIPs(t, env)
	if err != nil {
		t.Fatal(err)
	}
	if got.Dedicated || got.IPs == nil || len(got.IPs) != 0 {
		t.Fatalf("SendingIPs = %+v, want the shared pool (no IPs, an empty list)", got)
	}
}

// Access from before ReputeCast asked for ses:GetDedicatedIps is refused:
// the error says what to allow, in the reader's language, and never reads
// as "no dedicated IPs".
func TestSendingIPsPermissionMissing(t *testing.T) {
	env := contracttest.NewEnv(t, dedicatedIPs(func(w http.ResponseWriter, _ string) {
		awsError(w, http.StatusForbidden, "AccessDeniedException", "User: arn:aws:iam::123456789012:user/reputecast-ses is not authorized to perform: ses:GetDedicatedIps")
	}))
	got, err := sendingIPs(t, env)
	if err == nil {
		t.Fatalf("SendingIPs = %+v, want an error", got)
	}
	var ce *providers.CallError
	if !errors.As(err, &ce) || !errors.Is(err, providers.ErrUnauthorized) || !strings.Contains(ce.Message, "ses:GetDedicatedIps") || ce.Code != "AccessDeniedException" {
		t.Fatalf("error %v (%+v), want a refused call that names ses:GetDedicatedIps", err, ce)
	}
	contracttest.CheckText(t, "the missing permission", ce.Message, ce.Text)
}

// A page that fails ends the check with an error, not with the IPs of the
// pages before it.
func TestSendingIPsPageFails(t *testing.T) {
	env := contracttest.NewEnv(t, dedicatedIPs(func(w http.ResponseWriter, token string) {
		if token == "" {
			_, _ = io.WriteString(w, `{"DedicatedIps":[{"Ip":"203.0.113.40","WarmupStatus":"DONE","WarmupPercentage":100}],"NextToken":"b"}`)
			return
		}
		awsError(w, http.StatusTooManyRequests, "TooManyRequestsException", "Rate exceeded")
	}))
	got, err := sendingIPs(t, env)
	if err == nil || len(got.IPs) != 0 {
		t.Fatalf("SendingIPs = %+v, %v; want an error and no partial list", got, err)
	}
}
