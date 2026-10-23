package imports

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/netguard"
)

// A self-hosted server's address is limited to scheme, host and path, and
// may not point at an internal network (SSRF).
func TestServerAddress(t *testing.T) {
	public := &Service{}
	private := &Service{Guard: netguard.Policy{AllowPrivate: true}}
	cases := []struct {
		name, in string
		svc      *Service
		want     string // the cleaned address, or "" when refused
		problem  string // a word of the refusal
	}{
		{name: "trailing slash", in: "https://lists.example.com/", svc: public, want: "https://lists.example.com"},
		{name: "path kept", in: "https://example.com/listmonk/api/", svc: public, want: "https://example.com/listmonk/api"},
		{name: "port kept", in: "http://lists.example.com:9000", svc: public, want: "http://lists.example.com:9000"},
		{name: "no scheme", in: "lists.example.com", svc: public, problem: "full address"},
		{name: "other scheme", in: "ftp://lists.example.com", svc: public, problem: "full address"},
		{name: "fragment swallows the API path", in: "http://169.254.169.254/latest/meta-data/#/api", svc: public, problem: "no credentials, query or fragment"},
		{name: "query swallows the API path", in: "http://10.0.0.5:8080/admin?x=/api", svc: public, problem: "no credentials, query or fragment"},
		{name: "credentials", in: "http://user:pass@lists.example.com", svc: public, problem: "no credentials, query or fragment"},
		{name: "metadata address", in: "http://169.254.169.254/latest/meta-data/", svc: public, problem: "private or internal network"},
		{name: "private address", in: "http://10.0.0.5:8080/admin", svc: public, problem: "private or internal network"},
		{name: "loopback", in: "http://127.0.0.1:9000", svc: public, problem: "private or internal network"},
		{name: "localhost", in: "http://localhost:9000", svc: public, problem: "private or internal network"},
		{name: "IPv6 loopback", in: "http://[::1]:9000", svc: public, problem: "private or internal network"},
		{name: "private allowed", in: "http://localhost:9000/", svc: private, want: "http://localhost:9000"},
		{name: "metadata never allowed", in: "http://169.254.169.254/", svc: private, problem: "private or internal network"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, problem := c.svc.serverAddress(context.Background(), c.in)
			if c.want != "" {
				if problem != "" || got != c.want {
					t.Fatalf("got %q, %q; want %q", got, problem, c.want)
				}
				return
			}
			if problem == "" || !strings.Contains(problem, c.problem) {
				t.Fatalf("got %q, %q; want a refusal mentioning %q", got, problem, c.problem)
			}
		})
	}
}

// The data center in a Mailchimp key becomes part of the API host: only a
// short code passes.
func TestMailchimpDataCenter(t *testing.T) {
	for _, bad := range []string{"x-10.0.0.5:8443/#", "x-evil.example.com/?", "x-US21", "x-us", "x-"} {
		_, err := NewMailchimp(bad, "", nil)
		var se *SourceError
		if !errors.As(err, &se) || se.Code != CodeAuthFailed {
			t.Fatalf("%q: got %v; want the key-format refusal", bad, err)
		}
	}
	m, err := NewMailchimp("5f0c9e3a7b2d4e6f8a1b3c5d7e9f0a2b-us21", "", nil)
	if err != nil || m.DC != "us21" || m.c.base != "https://us21.api.mailchimp.com/3.0" {
		t.Fatalf("got %+v, %v", m, err)
	}
}

// The guarded client refuses an internal address when it connects, without
// retrying, and reports it in a sentence.
func TestGuardedClientRefusesInternalAddress(t *testing.T) {
	lm, err := NewListmonk("http://127.0.0.1:1", "user", "token", netguard.Policy{}.Client(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = lm.Connect(context.Background())
	var se *SourceError
	if !errors.As(err, &se) || se.Code != CodeUnavailable || !strings.Contains(se.Message, "Blocked: 127.0.0.1") {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("a blocked address was retried")
	}
}

// Redirects are not followed: the person enters the address the server
// answers at.
func TestRedirectIsNotFollowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://lists.example.com"+r.URL.Path, http.StatusMovedPermanently)
	}))
	defer srv.Close()
	lm, err := NewListmonk(srv.URL, "user", "token", netguard.Policy{AllowPrivate: true}.Client(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	_, err = lm.Connect(context.Background())
	var se *SourceError
	if !errors.As(err, &se) || se.Status != http.StatusMovedPermanently || !strings.Contains(se.Message, "redirects to another address") {
		t.Fatalf("got %v", err)
	}
}
