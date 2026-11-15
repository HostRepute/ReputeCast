package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/go-chi/chi/v5"
)

type rawUpload struct {
	RawBody []byte `contentType:"text/csv"`
}

type jsonUpload struct {
	Body struct {
		Name string `json:"name"`
	}
}

// Huma reads a body with a 5 s deadline unless the operation sets its own;
// a raw upload gets the server's read timeout instead.
func TestRawUploadsGetTheReadTimeout(t *testing.T) {
	_, h := humatest.New(t)
	raw := func(context.Context, *rawUpload) (*struct{}, error) { return nil, nil }
	Post(h, Op{ID: "raw", Path: "/raw", Access: Public}, raw)
	Post(WithReadTimeout(h, 2*time.Minute), Op{ID: "raw-configured", Path: "/raw-configured", Access: Public}, raw)
	Post(h, Op{ID: "json", Path: "/json", Access: Public}, func(context.Context, *jsonUpload) (*struct{}, error) { return nil, nil })
	for path, want := range map[string]time.Duration{
		Version + "/raw":            UploadReadTimeout,
		Version + "/raw-configured": 2 * time.Minute,
		Version + "/json":           5 * time.Second,
	} {
		if got := h.OpenAPI().Paths[path].Post.BodyReadTimeout; got != want {
			t.Errorf("%s reads its body for %s, want %s", path, got, want)
		}
	}
}

// An operation that waits on a slow upstream still answers after the
// server's write timeout when it sets WriteTimeout; others lose the answer.
func TestWriteTimeoutOutlastsTheServers(t *testing.T) {
	router := chi.NewRouter()
	h := humachi.New(router, huma.DefaultConfig("test", "1"))
	slow := func(context.Context, *struct{}) (*Body[string], error) {
		time.Sleep(300 * time.Millisecond)
		return OK("drafted"), nil
	}
	Get(h, Op{ID: "slow", Path: "/slow", Access: Public}, slow)
	Get(h, Op{ID: "slow-extended", Path: "/slow-extended", Access: Public, WriteTimeout: 5 * time.Second}, slow)
	srv := httptest.NewUnstartedServer(router)
	srv.Config.WriteTimeout = 100 * time.Millisecond
	srv.Start()
	defer srv.Close()
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	res, err := client.Get(srv.URL + Version + "/slow-extended")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "drafted") {
		t.Fatalf("extended operation: %d %s", res.StatusCode, body)
	}
	if res, err := client.Get(srv.URL + Version + "/slow"); err == nil {
		_ = res.Body.Close()
		t.Fatal("the server's write timeout no longer applies to other operations")
	}
}
