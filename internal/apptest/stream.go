package apptest

import (
	"context"
	"net/http"
)

// Stream opens a server-sent events stream at path under /api/v1 with the
// client's session. The caller reads the body and closes it; the stream
// stays open until ctx ends.
func (c *Client) Stream(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.env.Server.URL+"/api/v1"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if c.Bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.Bearer)
		req.Header.Del("Sec-Fetch-Site")
	}
	for k, vs := range c.Header {
		req.Header[k] = vs
	}
	hc := *c.http
	hc.Timeout = 0
	return hc.Do(req)
}
