package renders

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// adapter is one rendering service's API.
type adapter interface {
	// clients lists the email clients the account renders in. It is also
	// how a key is tested.
	clients(ctx context.Context, c conn) ([]RenderClient, error)
	// start sends the email and starts the captures; it returns the
	// service's id for them.
	start(ctx context.Context, c conn, e Email, clients []string) (string, error)
	// results reports each client's capture.
	results(ctx context.Context, c conn, id string, clients []string) ([]clientResult, error)
}

var adapters = map[string]adapter{Litmus: litmus{}, MailgunInspect: mailgunInspect{}}

type clientResult struct {
	client string
	status string // pending, done, failed
	image  string
	thumb  string
}

// conn is one call's connection: the service's base URL and the key.
type conn struct {
	http    *http.Client
	base    string
	service string
	key     string
}

// Why a call failed, worded for people in describe.
type callKind int

const (
	unreachable callKind = iota // network, timeout, blocked address
	refused                     // 401 or 403: the key
	failed                      // any other answer the service gives
)

type callError struct {
	kind   callKind
	status int
	err    error
}

func (e *callError) Error() string {
	switch e.kind {
	case refused:
		return fmt.Sprintf("key refused (HTTP %d)", e.status)
	case failed:
		return fmt.Sprintf("HTTP %d", e.status)
	}
	return e.err.Error()
}

func (e *callError) Unwrap() error { return e.err }

// call sends a JSON request and decodes a JSON answer into out (when set).
func (c conn) call(ctx context.Context, method, path string, body, out any) error {
	if c.http == nil || c.base == "" {
		return &callError{kind: unreachable, err: errors.New("no client")}
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.base, "/")+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ReputeCast")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	switch c.service {
	case MailgunInspect:
		req.SetBasicAuth("api", c.key)
	default:
		req.SetBasicAuth(c.key, "")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return &callError{kind: unreachable, err: err}
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return &callError{kind: unreachable, err: err}
	}
	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return &callError{kind: refused, status: res.StatusCode}
	case res.StatusCode < 200 || res.StatusCode > 299:
		return &callError{kind: failed, status: res.StatusCode}
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return &callError{kind: failed, status: res.StatusCode, err: err}
		}
	}
	return nil
}

// each runs f for every client, four at a time.
func each(clients []string, f func(string) clientResult) []clientResult {
	out := make([]clientResult, len(clients))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, c := range clients {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			out[i] = f(c)
		}()
	}
	wg.Wait()
	return out
}

// ── Litmus ──────────────────────────────────────────────────────────────

// litmus is the Litmus Instant API: POST an email for an email_guid, then
// GET each client's preview (captured on first request; prefetch starts
// them all at once).
type litmus struct{}

func (litmus) clients(ctx context.Context, c conn) ([]RenderClient, error) {
	var codes []string
	if err := c.call(ctx, http.MethodGet, "/v1/clients", nil, &codes); err != nil {
		return nil, err
	}
	out := make([]RenderClient, 0, len(codes))
	for _, code := range codes {
		if validID(code) {
			out = append(out, RenderClient{ID: code, Name: litmusName(code)})
		}
	}
	return out, nil
}

func (litmus) start(ctx context.Context, c conn, e Email, clients []string) (string, error) {
	var res struct {
		GUID string `json:"email_guid"`
	}
	if err := c.call(ctx, http.MethodPost, "/v1/emails", map[string]any{"html_text": e.HTML, "subject": e.Subject}, &res); err != nil {
		return "", err
	}
	if !validID(res.GUID) {
		return "", &callError{kind: failed, status: http.StatusOK, err: errors.New("no email_guid")}
	}
	cfgs := make([]map[string]string, len(clients))
	for i, cl := range clients {
		cfgs[i] = map[string]string{"client": cl}
	}
	// Start every capture now rather than on first view; a refusal here
	// only means they start when first asked for.
	_ = c.call(ctx, http.MethodPost, "/v1/emails/"+url.PathEscape(res.GUID)+"/previews/prefetch", map[string]any{"configurations": cfgs}, nil)
	return res.GUID, nil
}

func (litmus) results(ctx context.Context, c conn, id string, clients []string) ([]clientResult, error) {
	if !validID(id) {
		return nil, &callError{kind: failed, err: errors.New("bad id")}
	}
	var refusedErr error
	var mu sync.Mutex
	out := each(clients, func(cl string) clientResult {
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		var p struct {
			Full     string `json:"full_url"`
			Thumb    string `json:"thumb_url"`
			Thumb450 string `json:"thumb450_url"`
		}
		err := c.call(cctx, http.MethodGet, "/v1/emails/"+url.PathEscape(id)+"/previews/"+url.PathEscape(cl), nil, &p)
		var ce *callError
		switch {
		case err == nil && p.Full != "":
			thumb := p.Thumb450
			if thumb == "" {
				thumb = p.Thumb
			}
			return clientResult{client: cl, status: StatusDone, image: p.Full, thumb: thumb}
		case errors.As(err, &ce) && ce.kind == refused:
			mu.Lock()
			refusedErr = err
			mu.Unlock()
		case errors.As(err, &ce) && ce.kind == failed && ce.status >= 400 && ce.status < 500 && ce.status != http.StatusTooManyRequests && ce.status != http.StatusRequestTimeout:
			// The service won't capture this client for this email.
			return clientResult{client: cl, status: StatusFailed}
		}
		return clientResult{client: cl, status: StatusPending}
	})
	if refusedErr != nil {
		return nil, refusedErr
	}
	return out, nil
}

var (
	litmusCode   = regexp.MustCompile(`^([A-Z]+?)(\d+[A-Z]*)?$`)
	litmusSuffix = regexp.MustCompile(`^(\d+)([A-Z]*)$`)
)

// litmusName reads a Litmus client code as people say it: OL2019 →
// Outlook 2019, GMAILNEW → Gmail, IPHONE13 → iPhone 13. Unknown codes stay.
func litmusName(code string) string {
	names := map[string]string{
		"OL": "Outlook", "OUTLOOK": "Outlook", "OFFICE": "Outlook", "GMAIL": "Gmail", "GMAILNEW": "Gmail", "IPHONE": "iPhone", "IPAD": "iPad",
		"APPLEMAIL": "Apple Mail", "YAHOO": "Yahoo Mail", "AOL": "AOL Mail", "ANDROID": "Android", "SAMSUNG": "Samsung Mail", "THUNDERBIRD": "Thunderbird",
		"OUTLOOKCOM": "Outlook.com", "OFFICE365": "Outlook (Microsoft 365)", "GMAILANDROID": "Gmail for Android", "GMAILIOS": "Gmail for iOS",
	}
	if n, ok := names[code]; ok {
		return n
	}
	if m := litmusCode.FindStringSubmatch(code); m != nil {
		if n, ok := names[m[1]]; ok {
			if s := litmusSuffix.FindStringSubmatch(m[2]); s != nil {
				n += " " + s[1]
				if s[2] != "" {
					n += " " + s[2][:1] + strings.ToLower(s[2][1:])
				}
			}
			return n
		}
	}
	return code
}

// ── Mailgun Inspect ─────────────────────────────────────────────────────

// mailgunInspect is Mailgun Inspect's email previews (formerly Email on
// Acid): create a test with the clients, then read its results keyed by
// client id, each with a status and screenshot URLs.
type mailgunInspect struct{}

// mgClient is a client as Mailgun Inspect lists it.
type mgClient struct {
	ID          string `json:"id"`
	Client      string `json:"client"`
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	OS          string `json:"os"`
}

func (m mgClient) label(id string) string {
	name := m.DisplayName
	if name == "" {
		name = m.Client
	}
	if name == "" {
		name = m.Name
	}
	if name == "" {
		name = id
	}
	if m.OS != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(m.OS)) {
		name += " · " + m.OS
	}
	return name
}

// decodeClients reads a client list given as {"clients": {...}} or
// {"clients": [...]}, or as that map or array alone.
func decodeClients(raw json.RawMessage) []RenderClient {
	var wrapped struct {
		Clients json.RawMessage `json:"clients"`
		Items   json.RawMessage `json:"items"`
	}
	if json.Unmarshal(raw, &wrapped) == nil {
		switch {
		case len(wrapped.Clients) > 0:
			raw = wrapped.Clients
		case len(wrapped.Items) > 0:
			raw = wrapped.Items
		}
	}
	var out []RenderClient
	var byID map[string]mgClient
	if json.Unmarshal(raw, &byID) == nil {
		for id, c := range byID {
			if c.ID != "" {
				id = c.ID
			}
			if validID(id) {
				out = append(out, RenderClient{ID: id, Name: c.label(id)})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out
	}
	var list []mgClient
	if json.Unmarshal(raw, &list) == nil {
		for _, c := range list {
			if validID(c.ID) {
				out = append(out, RenderClient{ID: c.ID, Name: c.label(c.ID)})
			}
		}
	}
	return out
}

func (mailgunInspect) clients(ctx context.Context, c conn) ([]RenderClient, error) {
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodGet, "/v1/preview/clients", nil, &raw); err != nil {
		return nil, err
	}
	return decodeClients(raw), nil
}

func (mailgunInspect) start(ctx context.Context, c conn, e Email, clients []string) (string, error) {
	var res struct {
		ID string `json:"id"`
	}
	if err := c.call(ctx, http.MethodPost, "/v2/preview/tests", map[string]any{"subject": e.Subject, "html": e.HTML, "clients": clients}, &res); err != nil {
		return "", err
	}
	if !validID(res.ID) {
		return "", &callError{kind: failed, status: http.StatusOK, err: errors.New("no test id")}
	}
	return res.ID, nil
}

func (mailgunInspect) results(ctx context.Context, c conn, id string, clients []string) ([]clientResult, error) {
	if !validID(id) {
		return nil, &callError{kind: failed, err: errors.New("bad id")}
	}
	var res map[string]struct {
		Status      string            `json:"status"`
		Screenshots map[string]string `json:"screenshots"`
		Thumbnail   string            `json:"thumbnail"`
		FullThumb   string            `json:"full_thumbnail"`
	}
	if err := c.call(ctx, http.MethodGet, "/v1/preview/tests/"+url.PathEscape(id)+"/results", nil, &res); err != nil {
		return nil, err
	}
	out := make([]clientResult, 0, len(clients))
	for _, cl := range clients {
		r, ok := res[cl]
		if !ok {
			out = append(out, clientResult{client: cl, status: StatusPending})
			continue
		}
		image := r.Screenshots["default"]
		if image == "" {
			for _, v := range r.Screenshots {
				image = v
				break
			}
		}
		thumb := r.Thumbnail
		if thumb == "" {
			thumb = r.FullThumb
		}
		st := StatusPending
		switch strings.ToLower(r.Status) {
		case "complete", "completed", "done", "success":
			st = StatusDone
		case "bounced", "failed", "error", "unavailable":
			st = StatusFailed
		}
		out = append(out, clientResult{client: cl, status: st, image: image, thumb: thumb})
	}
	return out, nil
}
