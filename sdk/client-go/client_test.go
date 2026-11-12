package reputecast_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	reputecast "github.com/hostrepute/reputecast/sdk/client-go"
)

const lists = `{"items":[{"id":"l1","name":"Weekly Digest","description":"","doubleOptIn":false,"subscribed":1,"pending":0,"createdAt":"2026-01-01T00:00:00Z"}]}`

func TestNewSendsTheKeyAndUserAgent(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(context.Background())
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, lists)
	}))
	defer srv.Close()
	rc, err := reputecast.New(srv.URL+"/", "rc_live_test")
	if err != nil {
		t.Fatal(err)
	}
	res, err := rc.ListListsWithResponse(context.Background(), "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	if res.JSON200 == nil || len(res.JSON200.Items) != 1 || res.JSON200.Items[0].Name != "Weekly Digest" {
		t.Fatalf("lists: %s", res.Body)
	}
	if got.URL.Path != "/api/v1/workspaces/acme-newsletter/lists" || got.Header.Get("Authorization") != "Bearer rc_live_test" ||
		got.Header.Get("User-Agent") != reputecast.UserAgent || !strings.HasPrefix(reputecast.UserAgent, "ReputeCast-SDK-Go/") {
		t.Errorf("request: %s %v", got.URL.Path, got.Header)
	}
}

func TestErrorAnswersAreErrors(t *testing.T) {
	problem := `{"type":"about:blank","status":403,"title":"Forbidden","detail":"You need team:view. Linh Tran can give it to you.",` +
		`"code":"permission_denied","permission":"team:view","helpers":["Linh Tran"],"errors":[{"location":"path.workspace","message":"No."}],"meta":{"reference":"err_7KQ2M4XA"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/lists") {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, "Bad gateway from the proxy")
			return
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, problem)
	}))
	defer srv.Close()
	rc, err := reputecast.New(srv.URL, "rc_live_test", reputecast.WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}

	res, err := rc.ListMembersWithResponse(context.Background(), "acme-newsletter", nil)
	var apiErr *reputecast.Error
	if res != nil || !errors.As(err, &apiErr) {
		t.Fatalf("res %v, err %v", res, err)
	}
	if apiErr.Status != 403 || apiErr.Code != "permission_denied" || apiErr.Permission != "team:view" || len(apiErr.Helpers) != 1 ||
		apiErr.Title != "Forbidden" || apiErr.Detail == "" || len(apiErr.Errors) != 1 || apiErr.Errors[0].Location != "path.workspace" ||
		apiErr.Meta["reference"] != "err_7KQ2M4XA" {
		t.Errorf("error: %+v", apiErr)
	}
	if want := "reputecast: 403 permission_denied: You need team:view. Linh Tran can give it to you."; err.Error() != want {
		t.Errorf("message %q", err.Error())
	}

	_, err = rc.ListListsWithResponse(context.Background(), "acme-newsletter")
	if !errors.As(err, &apiErr) || apiErr.Status != 502 || apiErr.Code != "" || apiErr.Detail != "Bad gateway from the proxy" || apiErr.Title != "Bad Gateway" {
		t.Errorf("text answer: %v", err)
	}
}

func TestAnIdempotencyKeyLetsASendBeRetried(t *testing.T) {
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		if len(keys) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"id":"0193","status":"queued"}`)
	}))
	defer srv.Close()
	rc, err := reputecast.New(srv.URL, "rc_live_test")
	if err != nil {
		t.Fatal(err)
	}
	key := "pwreset-u_42-t_7"
	res, err := rc.SendTransactionalMessageWithResponse(context.Background(), "acme-newsletter",
		&reputecast.SendTransactionalMessageParams{IdempotencyKey: &key},
		reputecast.TransactionalSend{Template: "password-reset", To: reputecast.TransactionalAddress{Email: "sam.rivera@outlook.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.JSON202 == nil || res.JSON202.Id != "0193" || len(keys) != 2 || keys[0] != key || keys[1] != key {
		t.Fatalf("sent %v: %s", keys, res.Body)
	}
}

func TestNewRefusesOptionsThatCantWork(t *testing.T) {
	for _, c := range []struct {
		url, key string
		opts     []reputecast.Option
	}{
		{"https://news.example.com", "", nil},
		{"news.example.com", "k", nil},
		{"ftp://news.example.com", "k", nil},
		{"https://news.example.com", "k", []reputecast.Option{reputecast.WithMaxRetries(-1)}},
	} {
		if _, err := reputecast.New(c.url, c.key, c.opts...); err == nil {
			t.Errorf("New(%q, %q) took it", c.url, c.key)
		}
	}
}
