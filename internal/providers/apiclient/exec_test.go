package apiclient_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
)

func TestExec(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/send":
			b, _ := io.ReadAll(r.Body)
			if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || string(b) != "a=1" ||
				r.Header.Get("Authorization") != "Bearer k" || r.Header.Get("X-Call") != "yes" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("X-Message-Id", "m-1")
			w.WriteHeader(http.StatusAccepted)
		case "/auth":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"message":"bad key"}`)
		case "/down":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/slow":
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"page":"`+r.URL.Query().Get("page")+`"}`)
		}
	}))
	defer srv.Close()
	c := apiclient.New(srv.URL, srv.Client())
	c.Header.Set("Authorization", "Bearer k")
	ctx := context.Background()

	res, err := c.Exec(ctx, apiclient.Request{Op: "POST /send", Method: http.MethodPost, Path: "/send", ContentType: "application/x-www-form-urlencoded",
		Body: []byte("a=1"), Header: http.Header{"X-Call": {"yes"}}})
	if err != nil || res.Status != http.StatusAccepted || res.Header.Get("X-Message-Id") != "m-1" {
		t.Fatalf("send: %+v, %v", res, err)
	}
	if _, err := c.Exec(ctx, apiclient.Request{Op: "GET /auth", Method: http.MethodGet, Path: "/auth"}); !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("401 must be ErrUnauthorized: %v", err)
	}
	if _, err := c.Exec(ctx, apiclient.Request{Op: "GET /down", Method: http.MethodGet, Path: "/down"}); !errors.Is(err, providers.ErrUnavailable) {
		t.Fatalf("503 must be ErrUnavailable: %v", err)
	}
	var ce *providers.CallError
	if _, err := c.Exec(ctx, apiclient.Request{Op: "GET /slow", Method: http.MethodGet, Path: "/slow"}); !errors.As(err, &ce) || !ce.Throttled {
		t.Fatalf("429 must be throttled: %v", err)
	}
	res, err = c.Exec(ctx, apiclient.Request{Op: "GET /list", Method: http.MethodGet, Path: srv.URL + "/list?page=2"})
	if err != nil || !strings.Contains(string(res.Body), `"2"`) {
		t.Fatalf("same-origin link: %v", err)
	}
	if _, err := c.Exec(ctx, apiclient.Request{Op: "GET /list", Method: http.MethodGet, Path: "https://elsewhere.example/list"}); err == nil {
		t.Fatal("a link to another host must be refused")
	}
}
