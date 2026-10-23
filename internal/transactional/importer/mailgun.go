package importer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Mailgun's API, per region.
const (
	MailgunUS = "https://api.mailgun.net"
	MailgunEU = "https://api.eu.mailgun.net"
)

// Mailgun reads the templates of a domain (or of every domain on the
// account) with a private API key. The active version of each template is
// imported; its Subject header, when the version has one, is the subject.
type Mailgun struct {
	c      *client
	domain string
}

// NewMailgun builds a client. region is "us" or "eu"; base overrides it
// in tests. An empty domain reads every domain's templates.
func NewMailgun(key, region, domain, base string, hc *http.Client) (*Mailgun, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrNoKey
	}
	if base == "" {
		base = MailgunUS
		if strings.EqualFold(region, "eu") {
			base = MailgunEU
		}
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	if strings.ContainsAny(domain, "/?#@ ") {
		return nil, sourceErr(0, CodeBadResponse, "mailgun.domainOnly", nil)
	}
	return &Mailgun{domain: domain, c: newClient("Mailgun", base, hc, func(r *http.Request) { r.SetBasicAuth("api", key) },
		func(status int, _ []byte) i18n.Text {
			return say("auth.mailgun", i18n.Args{"code": strconv.Itoa(status)})
		})}, nil
}

// Dialect implements TemplateSource.
func (m *Mailgun) Dialect() Dialect { return DialectMailgun }

func (m *Mailgun) domains(ctx context.Context) ([]string, error) {
	if m.domain != "" {
		return []string{m.domain}, nil
	}
	var res struct {
		Items []struct {
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"items"`
	}
	if err := m.c.get(ctx, "/v4/domains?limit=1000", &res); err != nil {
		return nil, err
	}
	out := []string{}
	for _, d := range res.Items {
		if d.State != "disabled" {
			out = append(out, d.Name)
		}
	}
	return out, nil
}

// Connect implements TemplateSource: the domains whose templates are read.
func (m *Mailgun) Connect(ctx context.Context) (Account, error) {
	ds, err := m.domains(ctx)
	if err != nil {
		return Account{}, err
	}
	if m.domain != "" {
		// Check the key can read this domain's templates.
		var probe struct{}
		if err := m.c.get(ctx, "/v3/"+url.PathEscape(m.domain)+"/templates?limit=1", &probe); err != nil {
			return Account{}, err
		}
	}
	if len(ds) == 0 {
		return Account{}, sourceErr(0, CodeNotFound, "mailgun.noDomains", nil)
	}
	return Account{Name: strings.Join(ds, ", "), Kind: "domains"}, nil
}

// List implements TemplateSource. Refs are "domain/name".
func (m *Mailgun) List(ctx context.Context) ([]Ref, error) {
	ds, err := m.domains(ctx)
	if err != nil {
		return nil, err
	}
	out := []Ref{}
	for _, d := range ds {
		path := "/v3/" + url.PathEscape(d) + "/templates?limit=100"
		for range 100 {
			var page struct {
				Items []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				} `json:"items"`
				Paging struct {
					Next string `json:"next"`
				} `json:"paging"`
			}
			if err := m.c.get(ctx, path, &page); err != nil {
				return nil, err
			}
			for _, t := range page.Items {
				name := t.Name
				if t.Description != "" {
					name = t.Description
				}
				out = append(out, Ref{Ref: d + "/" + t.Name, Alias: t.Name, Name: name, Suggested: true})
			}
			pivot := queryParam(page.Paging.Next, "p")
			if len(page.Items) < 100 || pivot == "" {
				break
			}
			path = "/v3/" + url.PathEscape(d) + "/templates?limit=100&page=next&p=" + url.QueryEscape(pivot)
		}
	}
	return out, nil
}

// Fetch implements TemplateSource.
func (m *Mailgun) Fetch(ctx context.Context, ref string) (Source, error) {
	domain, name, ok := strings.Cut(ref, "/")
	if !ok || domain == "" || name == "" {
		return Source{}, sourceErr(0, CodeNotFound, "mailgun.noTemplate", nil)
	}
	var res struct {
		Template struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Version     struct {
				Tag      string            `json:"tag"`
				Template string            `json:"template"`
				Engine   string            `json:"engine"`
				Headers  map[string]string `json:"headers"`
			} `json:"version"`
		} `json:"template"`
	}
	if err := m.c.get(ctx, "/v3/"+url.PathEscape(domain)+"/templates/"+url.PathEscape(name)+"?active=yes", &res); err != nil {
		return Source{}, err
	}
	t := res.Template
	display := t.Name
	if t.Description != "" {
		display = t.Description
	}
	src := Source{Ref: ref, Alias: t.Name, Name: display, Format: FormatHTML, Body: t.Version.Template, Notes: []Note{}}
	for k, v := range t.Version.Headers {
		if strings.EqualFold(k, "subject") {
			src.Subject = v
		}
	}
	if src.Subject == "" {
		src.Notes = append(src.Notes, noteOf(NoteChange, "subject", 0, "", say("note.mailgunSubject", nil)))
	}
	if e := strings.ToLower(t.Version.Engine); e != "" && e != "handlebars" {
		src.Notes = append(src.Notes, noteOf(NoteManual, "template", 0, "", say("note.mailgunEngine", i18n.Args{"engine": t.Version.Engine})))
	}
	return src, nil
}
