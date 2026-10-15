package importer

import (
	"errors"
	"html"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// MaxFiles and MaxFileBytes limit one upload.
const (
	MaxFiles     = 50
	MaxFileBytes = 2 << 20
)

// ErrFileType is returned for files that are neither MJML nor HTML.
var ErrFileType = errors.New("only .mjml, .html and .htm files can be imported")

var (
	titleRe   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	hbBlockRe = regexp.MustCompile(`\{\{~?\s*[#/^>]|\{\{\{|\{\{~?\s*else\b`)
	liquidTag = regexp.MustCompile(`\{%-?\s*[a-z]+`)
)

// FromFile reads an uploaded MJML or HTML file. The name becomes the
// template's name and ID; MJML's <mj-title> and <mj-preview>, or HTML's
// <title>, become the subject and preheader. The dialect says how its tags
// are written: Liquid, or Handlebars exported from SendGrid or Mailgun.
func FromFile(name string, data []byte) (Source, Dialect, error) {
	base := path.Base(strings.ReplaceAll(name, `\`, "/"))
	ext := strings.ToLower(path.Ext(base))
	if !utf8.Valid(data) {
		return Source{}, "", errors.New("the file isn't UTF-8 text")
	}
	text := strings.TrimPrefix(string(data), string(rune(0xFEFF)))
	src := Source{Ref: base, Alias: strings.TrimSuffix(base, path.Ext(base)), Notes: []Note{}}
	src.Name = humanize(src.Alias)
	switch {
	case ext == ".mjml" || strings.HasPrefix(strings.TrimSpace(text), "<mjml"):
		src.Format = FormatMJML
		src.Subject, src.Preheader = mjml.HeadMeta(text)
	case ext == ".html" || ext == ".htm":
		src.Format = FormatHTML
		if m := titleRe.FindStringSubmatch(text); m != nil {
			src.Subject = strings.TrimSpace(html.UnescapeString(m[1]))
		}
	default:
		return Source{}, "", ErrFileType
	}
	src.Body = text
	if src.Subject == "" {
		src.Notes = append(src.Notes, Note{Kind: NoteChange, Field: "subject", Message: "the file has no title, so the subject is empty. Write one before publishing."})
	}
	d := DialectFile
	if hbBlockRe.MatchString(text) && !liquidTag.MatchString(text) {
		d = DialectHandlebarsFile
	}
	return src, d, nil
}
