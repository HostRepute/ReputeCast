package mcpclient

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type recordingTransport struct{ sent []*http.Request }

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.sent = append(r.sent, req)
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
}

// The credential goes only to the configured origin: the same scheme, host
// and port (the default port spelled out or not, the host in any case). Plain
// http for an https server, another port or another host isn't sent.
func TestAuthTransportOrigin(t *testing.T) {
	u, _ := url.Parse("https://mcp.example.com/mcp")
	base := &recordingTransport{}
	rt := &authTransport{base: base, serverOrigin: canonicalOrigin(u), header: "Authorization", value: "Bearer member-token"}
	for _, c := range []struct {
		url  string
		sent bool
	}{
		{"https://mcp.example.com/mcp", true},
		{"https://MCP.example.com:443/messages?sessionid=1", true},
		{"http://mcp.example.com/messages", false},
		{"http://mcp.example.com:443/messages", false},
		{"https://mcp.example.com:8443/messages", false},
		{"https://evil.example.com/messages", false},
		{"https://mcp.example.com.evil.example/messages", false},
	} {
		before := len(base.sent)
		req, _ := http.NewRequest(http.MethodPost, c.url, strings.NewReader("{}"))
		res, err := rt.RoundTrip(req)
		if res != nil {
			_ = res.Body.Close()
		}
		if sent := len(base.sent) > before; sent != c.sent || (err == nil) != c.sent || (!c.sent && !errors.Is(err, errElsewhere)) {
			t.Fatalf("%s: sent %v, err %v", c.url, sent, err)
		}
		if c.sent && base.sent[len(base.sent)-1].Header.Get("Authorization") != "Bearer member-token" {
			t.Fatalf("%s went without the credential", c.url)
		}
	}
	if rt.refused() == "" {
		t.Fatal("a refused request isn't remembered")
	}
}
