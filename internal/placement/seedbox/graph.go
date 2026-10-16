package seedbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GraphURL is Microsoft Graph's API root.
const GraphURL = "https://graph.microsoft.com/v1.0"

// GraphScopes are the delegated permissions a Microsoft seed is connected
// with: read and delete its mail, and keep access between tests.
var GraphScopes = []string{"offline_access", "openid", "email", "https://graph.microsoft.com/Mail.ReadWrite"}

// graphBox reads an Outlook.com or Microsoft 365 seed through Microsoft
// Graph, where each message says whether it's Focused or Other
// (inferenceClassification).
type graphBox struct {
	cfg   Config
	hc    *http.Client
	base  string
	token string
}

func openGraph(ctx context.Context, cfg Config) (Box, error) {
	tok, err := accessToken(cfg.Token)
	if err != nil {
		return nil, err
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	base := cfg.GraphURL
	if base == "" {
		base = GraphURL
	}
	b := &graphBox{cfg: cfg, hc: hc, base: strings.TrimRight(base, "/"), token: tok}
	// Signing in is reading the inbox's id: proof the grant covers mail.
	var f struct {
		ID string `json:"id"`
	}
	if err := b.get(ctx, "/me/mailFolders/inbox?$select=id,displayName", &f); err != nil {
		return nil, err
	}
	return b, nil
}

type graphError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (b *graphBox) do(ctx context.Context, method, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, b.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Accept", "application/json")
	resp, err := b.hc.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		var ge graphError
		_ = json.Unmarshal(body, &ge)
		return fmt.Errorf("%w: %s", ErrAuthExpired, firstNonEmpty(ge.Error.Message, ge.Error.Code, resp.Status))
	case resp.StatusCode == http.StatusForbidden:
		var ge graphError
		_ = json.Unmarshal(body, &ge)
		return fmt.Errorf("%w: %s", ErrAuthFailed, firstNonEmpty(ge.Error.Message, ge.Error.Code, resp.Status))
	case resp.StatusCode == http.StatusNotFound && method == http.MethodDelete:
		return nil // already gone
	case resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%w: Microsoft Graph answered %s", ErrUnreachable, resp.Status)
	case resp.StatusCode >= 300:
		var ge graphError
		_ = json.Unmarshal(body, &ge)
		return fmt.Errorf("microsoft graph %s %s: %s", method, path, firstNonEmpty(ge.Error.Message, resp.Status))
	}
	if out != nil && len(body) > 0 {
		return json.Unmarshal(body, out)
	}
	return nil
}

func (b *graphBox) get(ctx context.Context, path string, out any) error {
	return b.do(ctx, http.MethodGet, path, out)
}

// Graph's well-known folders where copies land.
var graphFolders = []struct{ id, role, label string }{
	{"inbox", RoleInbox, "Inbox"},
	{"junkemail", RoleSpam, "Junk Email"},
}

func (b *graphBox) Folders(context.Context) ([]Folder, error) {
	return []Folder{
		{Role: RoleInbox, Name: "inbox", Label: "Inbox"},
		{Role: RoleTab, Name: "focused", Label: "Focused"},
		{Role: RoleTab, Name: "other", Label: "Other"},
		{Role: RoleSpam, Name: "junkemail", Label: "Junk Email"},
	}, nil
}

type graphHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type graphMessage struct {
	ID                      string        `json:"id"`
	InternetMessageID       string        `json:"internetMessageId"`
	ReceivedDateTime        time.Time     `json:"receivedDateTime"`
	InferenceClassification string        `json:"inferenceClassification"`
	InternetMessageHeaders  []graphHeader `json:"internetMessageHeaders"`
}

// messages lists a folder's messages received since since, with their
// headers. A seed gets few messages, so a page or two covers a test.
func (b *graphBox) messages(ctx context.Context, folder string, since time.Time) ([]graphMessage, error) {
	q := url.Values{}
	q.Set("$select", "id,internetMessageId,receivedDateTime,inferenceClassification,internetMessageHeaders")
	q.Set("$top", "50")
	if !since.IsZero() {
		q.Set("$filter", "receivedDateTime ge "+since.Add(-time.Hour).UTC().Format(time.RFC3339))
	}
	path := "/me/mailFolders/" + folder + "/messages?" + q.Encode()
	var out []graphMessage
	for page := 0; page < 10 && path != ""; page++ {
		var res struct {
			Value []graphMessage `json:"value"`
			Next  string         `json:"@odata.nextLink"`
		}
		if err := b.get(ctx, path, &res); err != nil {
			return out, err
		}
		out = append(out, res.Value...)
		path = strings.TrimPrefix(res.Next, b.base)
	}
	return out, nil
}

// header turns Graph's header list back into a header block.
func (m graphMessage) header() Header {
	var sb strings.Builder
	for _, h := range m.InternetMessageHeaders {
		sb.WriteString(h.Name)
		sb.WriteString(": ")
		sb.WriteString(strings.ReplaceAll(h.Value, "\r\n", " "))
		sb.WriteString("\r\n")
	}
	if m.InternetMessageID != "" && !strings.Contains(strings.ToLower(sb.String()), "\nmessage-id:") {
		sb.WriteString("Message-ID: " + m.InternetMessageID + "\r\n")
	}
	return ParseHeader([]byte(sb.String()))
}

func (b *graphBox) Find(ctx context.Context, tokens []string, since time.Time) ([]Found, error) {
	want := tokenSet(tokens)
	var out []Found
	for _, f := range graphFolders {
		msgs, err := b.messages(ctx, f.id, since)
		if err != nil {
			return out, err
		}
		for _, m := range msgs {
			h := m.header()
			tok := matchToken(h, want)
			if tok == "" {
				continue
			}
			fd := Found{Token: tok, Role: f.role, Folder: f.label, ReceivedAt: m.ReceivedDateTime, Header: h, Ref: Ref{Folder: f.id, GraphID: m.ID}}
			if f.role == RoleInbox {
				switch strings.ToLower(m.InferenceClassification) {
				case "focused":
					fd.Tab = "Focused"
				case "other":
					fd.Tab = "Other"
				}
			}
			out = append(out, fd)
			delete(want, tok)
		}
	}
	return out, nil
}

func (b *graphBox) Delete(ctx context.Context, tokens []string, since time.Time) (int, error) {
	found, err := b.Find(ctx, tokens, since)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, f := range found {
		if err := b.do(ctx, http.MethodDelete, "/me/messages/"+url.PathEscape(f.Ref.GraphID), nil); err != nil {
			if errors.Is(err, ErrAuthExpired) || errors.Is(err, ErrAuthFailed) {
				return n, err
			}
			continue
		}
		n++
	}
	return n, nil
}

func (b *graphBox) Close() error { return nil }
