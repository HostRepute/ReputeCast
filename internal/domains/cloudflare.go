package domains

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Cloudflare adds a domain's missing records through the Cloudflare API
// ("Add all records via Cloudflare API"). The API token is used for one
// request and never stored.
type Cloudflare struct {
	// BaseURL is the API root (tests point it at a fake server).
	BaseURL string
	HTTP    *http.Client
}

// CloudflareResult is what the API call changed.
type CloudflareResult struct {
	Zone    string   `json:"zone"`
	Created []string `json:"created"`
	Updated []string `json:"updated"`
	Skipped []string `json:"skipped"`
	Failed  []string `json:"failed"`
}

type cfResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result json.RawMessage `json:"result"`
}

type cfRecord struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Proxied  *bool  `json:"proxied,omitempty"`
}

// ErrCloudflareToken means Cloudflare refused the token.
var ErrCloudflareToken = errors.New("Cloudflare rejected the API token")

// cloudflareError is a failed Cloudflare call, in words the domain page
// shows (problem.cloudflare*): Cloudflare couldn't be reached, answered
// with an error, or shows no zone for the domain.
type cloudflareError struct {
	kind    string // unreachable, answered, no_zone
	status  int
	message string // Cloudflare's own words, or the network error
	domain  string
	cause   error
}

func (e *cloudflareError) Error() string {
	switch e.kind {
	case "unreachable":
		return "couldn't reach Cloudflare: " + e.cause.Error()
	case "no_zone":
		return fmt.Sprintf("no Cloudflare zone for %s is visible to this token", e.domain)
	}
	msg := fmt.Sprintf("Cloudflare answered %d", e.status)
	if e.message != "" {
		msg += ": " + e.message
	}
	return msg
}

func (e *cloudflareError) Unwrap() error { return e.cause }

// problem is the error as the domain page shows it, in a reader's language.
func (e *cloudflareError) problem(ctx context.Context) string {
	switch e.kind {
	case "unreachable":
		return say(ctx, "problem.cloudflareUnreachable", i18n.Args{"error": e.cause.Error()})
	case "no_zone":
		return say(ctx, "problem.cloudflareNoZone", i18n.Args{"domain": e.domain})
	}
	return say(ctx, "problem.cloudflareAnswered", i18n.Args{"status": e.status, "hasMessage": yesNo(e.message != ""), "message": e.message})
}

func (c *Cloudflare) base() string {
	if c != nil && c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return "https://api.cloudflare.com/client/v4"
}

func (c *Cloudflare) client() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (c *Cloudflare) do(ctx context.Context, token, method, path string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base()+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return &cloudflareError{kind: "unreachable", cause: err}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var r cfResponse
	_ = json.Unmarshal(raw, &r)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrCloudflareToken
	}
	if resp.StatusCode >= 300 || !r.Success {
		e := &cloudflareError{kind: "answered", status: resp.StatusCode}
		if len(r.Errors) > 0 {
			e.message = r.Errors[0].Message
		}
		return e
	}
	if out != nil {
		return json.Unmarshal(r.Result, out)
	}
	return nil
}

// zone finds the Cloudflare zone of a domain (the domain or a parent).
func (c *Cloudflare) zone(ctx context.Context, token, domain string) (string, string, error) {
	name := domain
	for strings.Contains(name, ".") {
		var zones []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := c.do(ctx, token, http.MethodGet, "/zones?name="+url.QueryEscape(name), nil, &zones); err != nil {
			return "", "", err
		}
		if len(zones) > 0 {
			return zones[0].ID, zones[0].Name, nil
		}
		name = name[strings.IndexByte(name, '.')+1:]
	}
	return "", "", &cloudflareError{kind: "no_zone", domain: domain}
}

// Apply creates or updates every record that isn't verified yet. The
// sending domain's SPF record is updated in place (two SPF records would
// break SPF); an existing DMARC record is left alone.
func (c *Cloudflare) Apply(ctx context.Context, token, domain string, recs []Record) (CloudflareResult, error) {
	out := CloudflareResult{Created: []string{}, Updated: []string{}, Skipped: []string{}, Failed: []string{}}
	zoneID, zoneName, err := c.zone(ctx, token, domain)
	if err != nil {
		return out, err
	}
	out.Zone = zoneName
	for _, r := range recs {
		if r.Status == StatusVerified {
			out.Skipped = append(out.Skipped, r.Name)
			continue
		}
		label := r.Type + " " + r.Name
		var existing []cfRecord
		if err := c.do(ctx, token, http.MethodGet, "/zones/"+zoneID+"/dns_records?type="+url.QueryEscape(r.Type)+"&name="+url.QueryEscape(r.Name), nil, &existing); err != nil {
			out.Failed = append(out.Failed, label+": "+err.Error())
			continue
		}
		want := cfRecord{Type: r.Type, Name: r.Name, Content: r.Value, TTL: 1}
		if r.Type == "MX" {
			if f := strings.Fields(r.Value); len(f) == 2 {
				if pr, err := strconv.Atoi(f[0]); err == nil {
					want.Priority, want.Content = &pr, f[1]
				}
			}
		}
		if r.Type == "CNAME" {
			off := false
			want.Proxied = &off // DKIM and return-path CNAMEs must resolve to the provider
		}
		var match *cfRecord
		for i, e := range existing {
			switch {
			case r.Type == "TXT" && r.Purpose == PurposeSPF && strings.HasPrefix(strings.ToLower(strings.Trim(e.Content, `"`)), "v=spf1"):
				match = &existing[i]
			case r.Type == "TXT" && r.Purpose == PurposeDMARC && strings.HasPrefix(strings.Trim(e.Content, `"`), "v=DMARC1"):
				match = &existing[i]
			case r.Type == "TXT" && r.Purpose == PurposeDKIM && strings.Contains(e.Content, "p="):
				match = &existing[i]
			case r.Type != "TXT":
				match = &existing[i]
			}
			if match != nil {
				break
			}
		}
		switch {
		case match != nil && r.Purpose == PurposeDMARC:
			out.Skipped = append(out.Skipped, r.Name)
		case match != nil && strings.Trim(match.Content, `"`) == want.Content:
			out.Skipped = append(out.Skipped, r.Name)
		case match != nil:
			if err := c.do(ctx, token, http.MethodPatch, "/zones/"+zoneID+"/dns_records/"+match.ID, want, nil); err != nil {
				out.Failed = append(out.Failed, label+": "+err.Error())
				continue
			}
			out.Updated = append(out.Updated, r.Name)
		default:
			if err := c.do(ctx, token, http.MethodPost, "/zones/"+zoneID+"/dns_records", want, nil); err != nil {
				out.Failed = append(out.Failed, label+": "+err.Error())
				continue
			}
			out.Created = append(out.Created, r.Name)
		}
	}
	return out, nil
}
