package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// keyServer is an OpenAI-compatible server that answers Test connection and
// the model list, and keeps the keys it was sent.
type keyServer struct {
	srv  *httptest.Server
	mu   sync.Mutex
	keys []string
}

func newKeyServer(t *testing.T) *keyServer {
	k := &keyServer{}
	k.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k.mu.Lock()
		k.keys = append(k.keys, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		k.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gemma3:27b","object":"model"}]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "chatcmpl-1", "object": "chat.completion", "model": "gemma3:27b",
			"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": "Ready."}, "finish_reason": "stop"}},
			"usage":   map[string]any{"prompt_tokens": 9, "completion_tokens": 2, "total_tokens": 11}})
	}))
	t.Cleanup(k.srv.Close)
	return k
}

func (k *keyServer) got() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	return slices.Clone(k.keys)
}

// The saved key stands in for an empty one only at the address it was saved
// with: Test connection, the model list and saving all refuse to send it to
// another server, next to the key field.
func TestSavedKeyStaysWithItsAddress(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	mine, theirs := newKeyServer(t), newKeyServer(t)
	c.Do(http.MethodDelete, ws+"/ai/connection", nil).MustStatus(t, 200)
	connect := func(baseURL, key string) *apptest.Response {
		return c.Do(http.MethodPut, ws+"/ai/connection", map[string]any{"provider": "compatible", "label": "Office model", "baseUrl": baseURL, "apiKey": key,
			"model": "gemma3:27b", "capKind": "usd", "capUsd": 10, "inputPrice": 2, "outputPrice": 10})
	}
	connect(mine.srv.URL+"/v1", "sk-office-1").MustStatus(t, 200)
	if !slices.Contains(mine.got(), "sk-office-1") {
		t.Fatalf("connect: keys %v", mine.got())
	}

	const again = "The address changed: paste the API key again. ReputeCast doesn't send a saved key to a new address."
	other := theirs.srv.URL + "/v1"
	for name, r := range map[string]*apptest.Response{
		"test":    c.Do(http.MethodPost, ws+"/ai/test", map[string]any{"provider": "compatible", "baseUrl": other, "model": "gemma3:27b"}),
		"models":  c.Do(http.MethodPost, ws+"/ai/models", map[string]any{"provider": "compatible", "baseUrl": other}),
		"connect": connect(other, ""),
	} {
		r.MustStatus(t, 422)
		var p struct {
			Errors []struct{ Location, Message string }
		}
		r.JSON(t, &p)
		if len(p.Errors) != 1 || p.Errors[0].Location != "body.apiKey" || p.Errors[0].Message != again {
			t.Errorf("%s: %s", name, r.Body)
		}
	}
	if len(theirs.got()) != 0 {
		t.Fatalf("the saved key reached another server: %v", theirs.got())
	}
	var base string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT base_url FROM ai_connections c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = 'acme-newsletter'`).Scan(&base); err != nil || base != mine.srv.URL+"/v1" {
		t.Fatalf("saved address %q (%v)", base, err)
	}

	// The saved address (with a final slash) keeps the saved key.
	var test struct{ OK bool }
	c.Do(http.MethodPost, ws+"/ai/test", map[string]any{"provider": "compatible", "baseUrl": mine.srv.URL + "/v1/", "model": "gemma3:27b"}).MustStatus(t, 200).JSON(t, &test)
	if keys := mine.got(); !test.OK || keys[len(keys)-1] != "sk-office-1" {
		t.Fatalf("same address: %+v, keys %v", test, keys)
	}
	// A key pasted for the new address goes there, and only it does.
	connect(other, "sk-other-2").MustStatus(t, 200)
	if keys := theirs.got(); len(keys) == 0 || slices.Contains(keys, "sk-office-1") || !slices.Contains(keys, "sk-other-2") {
		t.Fatalf("new address: keys %v", keys)
	}
}
