package importer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PostmarkBase is Postmark's API.
const PostmarkBase = "https://api.postmarkapp.com"

// Postmark reads a server's templates with its server API token. Templates
// that use a layout get the layout merged in at {{{ @content }}}.
type Postmark struct {
	c       *client
	layouts map[string]pmTemplate
}

// NewPostmark builds a client; base is PostmarkBase outside tests.
func NewPostmark(token, base string, hc *http.Client) (*Postmark, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrNoKey
	}
	if base == "" {
		base = PostmarkBase
	}
	return &Postmark{c: newClient("Postmark", base, hc, func(r *http.Request) { r.Header.Set("X-Postmark-Server-Token", token) },
		func(status int, _ []byte) string {
			return fmt.Sprintf("Postmark answered %d: this token is for a different server or was revoked.", status)
		})}, nil
}

// ErrNoKey is returned when a source has no API key.
var ErrNoKey = errors.New("paste an API key first")

// Dialect implements TemplateSource.
func (p *Postmark) Dialect() Dialect { return DialectPostmark }

// Connect implements TemplateSource: the server the token belongs to.
func (p *Postmark) Connect(ctx context.Context) (Account, error) {
	var s struct {
		Name string `json:"Name"`
	}
	if err := p.c.get(ctx, "/server", &s); err != nil {
		return Account{}, err
	}
	return Account{Name: s.Name, Kind: "server"}, nil
}

type pmListItem struct {
	TemplateID     int64  `json:"TemplateId"`
	Name           string `json:"Name"`
	Alias          string `json:"Alias"`
	TemplateType   string `json:"TemplateType"`
	LayoutTemplate string `json:"LayoutTemplate"`
	Active         bool   `json:"Active"`
}

type pmTemplate struct {
	pmListItem
	Subject  string `json:"Subject"`
	HTMLBody string `json:"HtmlBody"`
	TextBody string `json:"TextBody"`
}

// List implements TemplateSource: the standard templates, in Postmark's
// order. Layouts aren't templates of their own; they are merged in.
func (p *Postmark) List(ctx context.Context) ([]Ref, error) {
	out := []Ref{}
	for offset := 0; offset < 10_000; offset += 100 {
		var page struct {
			TotalCount int          `json:"TotalCount"`
			Templates  []pmListItem `json:"Templates"`
		}
		if err := p.c.get(ctx, fmt.Sprintf("/templates?Count=100&Offset=%d&TemplateType=Standard", offset), &page); err != nil {
			return nil, err
		}
		for _, t := range page.Templates {
			ref := t.Alias
			if ref == "" {
				ref = fmt.Sprint(t.TemplateID)
			}
			// Inactive templates are listed but not picked.
			out = append(out, Ref{Ref: ref, Alias: t.Alias, Name: t.Name, Suggested: t.Active})
		}
		if len(page.Templates) < 100 || offset+100 >= page.TotalCount {
			break
		}
	}
	return out, nil
}

// Fetch implements TemplateSource.
func (p *Postmark) Fetch(ctx context.Context, ref string) (Source, error) {
	var t pmTemplate
	if err := p.c.get(ctx, "/templates/"+url.PathEscape(ref), &t); err != nil {
		return Source{}, err
	}
	src := Source{Ref: ref, Alias: t.Alias, Name: t.Name, Subject: t.Subject, Format: FormatHTML, Body: t.HTMLBody, Text: t.TextBody, Notes: []Note{}}
	if src.Alias == "" {
		src.Alias = t.Name
	}
	if t.LayoutTemplate != "" {
		layout, err := p.layout(ctx, t.LayoutTemplate)
		switch {
		case err != nil:
			src.Notes = append(src.Notes, Note{Kind: NoteManual, Field: "template", Message: fmt.Sprintf("its layout `%s` couldn't be read, so the email has no header or footer. Add them with saved sections.", t.LayoutTemplate)})
		default:
			src.Body = mergeLayout(layout.HTMLBody, src.Body)
			src.Text = mergeLayout(layout.TextBody, src.Text)
			src.Notes = append(src.Notes, Note{Kind: NoteInfo, Field: "template", Message: fmt.Sprintf("its layout “%s” was merged into the email.", layout.Name)})
		}
	}
	return src, nil
}

func (p *Postmark) layout(ctx context.Context, alias string) (pmTemplate, error) {
	if l, ok := p.layouts[alias]; ok {
		return l, nil
	}
	var l pmTemplate
	if err := p.c.get(ctx, "/templates/"+url.PathEscape(alias), &l); err != nil {
		return pmTemplate{}, err
	}
	if p.layouts == nil {
		p.layouts = map[string]pmTemplate{}
	}
	p.layouts[alias] = l
	return l, nil
}

var contentPlaceholders = []string{"{{{ @content }}}", "{{{@content}}}", "{{{ @content}}}", "{{{@content }}}"}

// mergeLayout puts a template's content into its layout.
func mergeLayout(layout, content string) string {
	if layout == "" {
		return content
	}
	for _, ph := range contentPlaceholders {
		if strings.Contains(layout, ph) {
			return strings.Replace(layout, ph, content, 1)
		}
	}
	return content
}
