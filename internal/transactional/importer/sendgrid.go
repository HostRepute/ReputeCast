package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// SendGridBase is SendGrid's API.
const SendGridBase = "https://api.sendgrid.com"

// SendGrid reads dynamic templates (Handlebars) with an API key that has
// Template Engine read access. The active version of each template is
// imported, with its test data.
type SendGrid struct{ c *client }

// NewSendGrid builds a client; base is SendGridBase outside tests.
func NewSendGrid(key, base string, hc *http.Client) (*SendGrid, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrNoKey
	}
	if base == "" {
		base = SendGridBase
	}
	return &SendGrid{c: newClient("SendGrid", base, hc, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+key) },
		func(status int, _ []byte) string {
			if status == http.StatusForbidden {
				return "SendGrid answered 403: this key can't read templates. Give it Template Engine read access."
			}
			return fmt.Sprintf("SendGrid answered %d: the API key was revoked or is mistyped.", status)
		})}, nil
}

// Dialect implements TemplateSource.
func (s *SendGrid) Dialect() Dialect { return DialectSendGrid }

// Connect implements TemplateSource: the key must read templates; the
// account's name comes from its profile when the key may read it.
func (s *SendGrid) Connect(ctx context.Context) (Account, error) {
	var probe struct {
		Result []json.RawMessage `json:"result"`
	}
	if err := s.c.get(ctx, "/v3/templates?generations=dynamic&page_size=1", &probe); err != nil {
		return Account{}, err
	}
	var profile struct {
		Company   string `json:"company"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	name := "SendGrid account"
	if err := s.c.get(ctx, "/v3/user/profile", &profile); err == nil {
		switch {
		case profile.Company != "":
			name = profile.Company
		case profile.FirstName != "":
			name = strings.TrimSpace(profile.FirstName + " " + profile.LastName)
		}
	}
	return Account{Name: name, Kind: "account"}, nil
}

type sgTemplate struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Versions []sgVersion `json:"versions"`
}

type sgVersion struct {
	ID           string `json:"id"`
	Active       int    `json:"active"`
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	HTMLContent  string `json:"html_content"`
	PlainContent string `json:"plain_content"`
	TestData     string `json:"test_data"`
	Editor       string `json:"editor"`
}

// List implements TemplateSource.
func (s *SendGrid) List(ctx context.Context) ([]Ref, error) {
	out := []Ref{}
	path := "/v3/templates?generations=dynamic&page_size=200"
	for range 50 {
		var page struct {
			Result   []sgTemplate `json:"result"`
			Metadata struct {
				Next string `json:"next"`
			} `json:"_metadata"`
		}
		if err := s.c.get(ctx, path, &page); err != nil {
			return nil, err
		}
		for _, t := range page.Result {
			out = append(out, Ref{Ref: t.ID, Alias: t.Name, Name: t.Name, Suggested: true})
		}
		// The next page is an absolute URL on SendGrid's host: only its
		// page token is used, so a crafted answer can't send us elsewhere.
		token := queryParam(page.Metadata.Next, "page_token")
		if token == "" {
			break
		}
		path = "/v3/templates?generations=dynamic&page_size=200&page_token=" + url.QueryEscape(token)
	}
	return out, nil
}

// queryParam reads one query parameter of a URL a provider answered with.
func queryParam(raw, key string) string {
	u, err := url.Parse(raw)
	if raw == "" || err != nil {
		return ""
	}
	return u.Query().Get(key)
}

// Fetch implements TemplateSource.
func (s *SendGrid) Fetch(ctx context.Context, ref string) (Source, error) {
	var t sgTemplate
	if err := s.c.get(ctx, "/v3/templates/"+url.PathEscape(ref), &t); err != nil {
		return Source{}, err
	}
	src := Source{Ref: ref, Alias: t.Name, Name: t.Name, Format: FormatHTML, Notes: []Note{}}
	var v *sgVersion
	for i := range t.Versions {
		if t.Versions[i].Active == 1 {
			v = &t.Versions[i]
			break
		}
	}
	if v == nil && len(t.Versions) > 0 {
		v = &t.Versions[0]
		src.Notes = append(src.Notes, Note{Kind: NoteInfo, Field: "template", Message: fmt.Sprintf("no version is active in SendGrid, so “%s” was imported.", v.Name)})
	}
	if v == nil {
		return Source{}, &SourceError{Code: CodeNotFound, Message: fmt.Sprintf("“%s” has no versions in SendGrid.", t.Name)}
	}
	src.Subject, src.Body, src.Text = v.Subject, v.HTMLContent, v.PlainContent
	if strings.TrimSpace(v.TestData) != "" {
		var data map[string]any
		if json.Unmarshal([]byte(v.TestData), &data) == nil {
			src.Sample = data
		}
	}
	return src, nil
}
