package azureacs

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
)

// tokenEntry caches one application's access token, shared by every
// adapter instance (the factory builds one per request).
type tokenEntry struct {
	mu      sync.Mutex
	token   string
	expires time.Time
}

var (
	tokensMu sync.Mutex
	tokens   = map[string]*tokenEntry{}
)

// token returns an Entra ID access token for the Communication Services
// data plane (client credentials), reused until two minutes before it
// expires.
func (p *Provider) token(ctx context.Context) (string, error) {
	authority := strings.TrimRight(cmp.Or(p.s.AuthorityURL, DefaultAuthority), "/")
	sum := sha256.Sum256([]byte(authority + "\x00" + p.s.TenantID + "\x00" + p.s.ClientID + "\x00" + p.c.ClientSecret))
	id := hex.EncodeToString(sum[:])
	tokensMu.Lock()
	e, ok := tokens[id]
	if !ok {
		e = &tokenEntry{}
		tokens[id] = e
	}
	tokensMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.token != "" && time.Until(e.expires) > 2*time.Minute {
		return e.token, nil
	}
	form := url.Values{"client_id": {p.s.ClientID}, "client_secret": {p.c.ClientSecret}, "grant_type": {"client_credentials"},
		"scope": {"https://communication.azure.com/.default"}}
	c := apiclient.New(authority, p.http)
	c.Decode = func(_ int, body []byte) (string, string) {
		var e struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error != "" {
			return e.Error, strings.SplitN(e.Description, "\r\n", 2)[0]
		}
		return "", strings.TrimSpace(string(body))
	}
	res, err := c.Exec(ctx, apiclient.Request{Op: "POST /oauth2/v2.0/token", Method: http.MethodPost, Path: "/" + url.PathEscape(p.s.TenantID) + "/oauth2/v2.0/token",
		ContentType: "application/x-www-form-urlencoded", Body: []byte(form.Encode())})
	if err != nil {
		var ce *providers.CallError
		if asCallError(err, &ce) && ce.Status == http.StatusBadRequest {
			// invalid_client, unauthorized_client: the application's
			// credentials are wrong.
			ce.Auth = true
		}
		return "", err
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(res.Body, &out); err != nil || out.AccessToken == "" {
		return "", &providers.CallError{Op: "POST /oauth2/v2.0/token", Status: res.Status, Temporary: true, Message: "Entra ID returned no access token"}
	}
	e.token, e.expires = out.AccessToken, time.Now().Add(time.Duration(out.ExpiresIn)*time.Second)
	return e.token, nil
}
