// Package importer reads the exports of other platforms into posts and
// converts their HTML into the MJML a ReputeCast post holds (M61, G-26):
//
//   - a Substack export, the zip with posts.csv and posts/<post_id>.html;
//   - a Ghost export, the JSON file (or a zip holding it, with the site's
//     content/images when the zip has them);
//   - a WordPress export, the WXR file (WordPress eXtended RSS).
//
// The format is told from the content, never from the file's name. It
// touches no database: internal/website stages the posts, and imports
// them.
package importer

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"html"
	"io"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/i18n"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "importer")

// The platforms an export comes from.
const (
	Substack  = "substack"
	Ghost     = "ghost"
	WordPress = "wordpress"
)

// Name is a platform's name ("Substack"), the same in every language.
func Name(source string) string {
	switch source {
	case Substack:
		return "Substack"
	case Ghost:
		return "Ghost"
	case WordPress:
		return "WordPress"
	}
	return source
}

// Limits of an export.
const (
	// MaxPosts is the most posts one import takes.
	MaxPosts = 10000
	// maxEntry is the largest file read from inside a zip.
	maxEntry = 200 << 20
)

// ErrNotExport is a file that isn't a Substack, Ghost or WordPress export.
var ErrNotExport = errors.New("not a Substack, Ghost or WordPress export")

// ErrTooManyPosts is an export with more than MaxPosts posts.
var ErrTooManyPosts = errors.New("more posts than one import takes")

// Visibility levels, as website posts have them.
const (
	Public  = "public"
	Members = "members"
	Paid    = "paid"
)

// Post is a post as an export has it.
type Post struct {
	// SourceID is the platform's id of the post: Substack's post_id,
	// Ghost's id, WordPress's wp:post_id.
	SourceID    string
	Title       string
	Slug        string
	PublishedAt *time.Time
	// Draft: not published on the old site (a draft, scheduled, pending).
	Draft      bool
	Visibility string
	// TierRefs are the old site's tiers that could read a paid post, by
	// name ("Paid subscribers", a Ghost tier's name); empty: every paid
	// member.
	TierRefs []string
	Tags     []string
	Author   string
	// Excerpt is the platform's own (Substack's subtitle), as text.
	Excerpt  string
	CoverURL string
	CoverAlt string
	// SEO and the canonical address.
	SEOTitle, SEODescription, CanonicalURL string
	// SourceURL is the post's address on the old site: absolute when the
	// export says where the site is, else its path.
	SourceURL string
	// LegacyPaths are the old site's addresses of the post, as LegacyKey
	// makes them ("p/my-post", "my-old-slug", "?p=123").
	LegacyPaths []string
	HTML        string
}

// Problem is something about a post the import couldn't keep as it was:
// an embed shown as a link, an image without alt text.
type Problem struct {
	Code string `json:"code"`
	// Detail is what it is about: the platform of an embed, an image's
	// address, a shortcode's name, the tags left out, a new address.
	Detail string `json:"detail,omitempty"`
}

// Problem codes the converter reports.
const (
	ProblemEmbed     = "embed_link"
	ProblemAlt       = "image_alt"
	ProblemImage     = "image_failed"
	ProblemShortcode = "shortcode"
)

// Detect names the platform an export comes from, by its content.
func Detect(data []byte) (string, error) {
	if isZip(data) {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return "", ErrNotExport
		}
		if _, ok := substackCSV(zr); ok {
			return Substack, nil
		}
		if f := ghostFile(zr); f != nil {
			return Ghost, nil
		}
		return "", ErrNotExport
	}
	body := trimBOM(data)
	switch {
	case looksLikeWXR(body):
		return WordPress, nil
	case looksLikeGhost(body):
		return Ghost, nil
	}
	return "", ErrNotExport
}

// Parse reads the posts of an export from a platform (as Detect named it).
func Parse(source string, data []byte) ([]Post, error) {
	switch source {
	case Substack:
		return ParseSubstack(data)
	case Ghost:
		return ParseGhost(data)
	case WordPress:
		return ParseWXR(data)
	}
	return nil, ErrNotExport
}

// File is a file the export holds at path (a Ghost zip's
// content/images/2024/05/photo.jpg for "/content/images/2024/05/photo.jpg"),
// so an image of a site the export can't name the address of can still be
// copied. ok is false when the export isn't a zip or has no such file.
func File(data []byte, p string) ([]byte, bool) {
	if !isZip(data) || !strings.HasPrefix(p, "/") || strings.Contains(p, "..") {
		return nil, false
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, false
	}
	rel := strings.TrimPrefix(path.Clean(p), "/")
	// Ghost's importer takes images at content/images/ or at images/.
	names := []string{rel}
	if short, ok := strings.CutPrefix(rel, "content/"); ok {
		names = append(names, short)
	}
	for _, f := range zr.File {
		name := strings.TrimPrefix(f.Name, "./")
		for _, want := range names {
			dir, ok := strings.CutSuffix(name, want)
			// At the zip's root, or under the one folder it was zipped in.
			if !ok || dir != "" && (!strings.HasSuffix(dir, "/") || strings.Count(dir, "/") != 1) {
				continue
			}
			b, err := readEntry(f)
			if err != nil {
				return nil, false
			}
			return b, true
		}
	}
	return nil, false
}

// LegacyKey is how an old address is kept and looked up: lower case,
// without the slashes around it; "?p=<id>" (WordPress's own links) keeps
// its query, as "?p=123".
func LegacyKey(p, rawQuery string) string {
	p = strings.ToLower(strings.Trim(p, "/"))
	if p == "" {
		if id := queryP(rawQuery); id != "" {
			return "?p=" + id
		}
	}
	return p
}

// queryP is the p= of a query when it's a number.
func queryP(rawQuery string) string {
	for part := range strings.SplitSeq(rawQuery, "&") {
		k, v, _ := strings.Cut(part, "=")
		if k == "p" && v != "" && strings.Trim(v, "0123456789") == "" {
			return v
		}
	}
	return ""
}

func isZip(data []byte) bool { return bytes.HasPrefix(data, []byte("PK\x03\x04")) }

func trimBOM(data []byte) []byte {
	return bytes.TrimLeft(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), " \t\r\n")
}

// readEntry reads a file of a zip, up to maxEntry.
func readEntry(f *zip.File) ([]byte, error) {
	if f.UncompressedSize64 > maxEntry {
		return nil, ErrNotExport
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(io.LimitReader(rc, maxEntry+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxEntry {
		return nil, ErrNotExport
	}
	return b, nil
}

// unescape reads text an export keeps with HTML entities ("&#x201C;").
func unescape(s string) string { return strings.TrimSpace(html.UnescapeString(s)) }

// textOf is an HTML fragment's text, its spaces collapsed.
func textOf(fragment string) string {
	if !strings.ContainsAny(fragment, "<&") {
		return strings.Join(strings.Fields(fragment), " ")
	}
	nodes, err := xhtml.ParseFragment(strings.NewReader(fragment), bodyContext())
	if err != nil {
		return strings.Join(strings.Fields(fragment), " ")
	}
	var b strings.Builder
	for _, n := range nodes {
		collectText(&b, n)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func collectText(b *strings.Builder, n *xhtml.Node) {
	switch n.Type {
	case xhtml.TextNode:
		b.WriteString(n.Data)
	case xhtml.ElementNode:
		if dropped(n) {
			return
		}
		if blockLevel(n) || n.Data == "br" {
			b.WriteByte(' ')
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(b, c)
	}
}

// Summary is the text an excerpt can be made of when the export has none:
// the post's first paragraph that has text.
func Summary(fragment string) string {
	nodes, err := xhtml.ParseFragment(strings.NewReader(fragment), bodyContext())
	if err != nil {
		return ""
	}
	var found string
	var walk func(*xhtml.Node) bool
	walk = func(n *xhtml.Node) bool {
		if n.Type == xhtml.ElementNode {
			if dropped(n) || widget(n) {
				return false
			}
			if n.Data == "p" {
				var b strings.Builder
				collectText(&b, n)
				if t := strings.Join(strings.Fields(b.String()), " "); utf8.RuneCountInString(t) >= 2 {
					found = t
					return true
				}
				return false
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	for _, n := range nodes {
		if walk(n) {
			break
		}
	}
	return found
}

// parseTime reads the dates exports write: RFC 3339, with or without
// fractions, or "2006-01-02 15:04:05" in UTC.
func parseTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "0000-00-00") {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			u := t.UTC()
			return &u
		}
	}
	return nil
}

// looksLikeGhost reports a JSON document shaped as a Ghost export: its data
// (under db, or at the top in old exports) holds posts.
func looksLikeGhost(body []byte) bool {
	if len(body) == 0 || body[0] != '{' {
		return false
	}
	return ghostShape(json.NewDecoder(bytes.NewReader(body)))
}

// looksLikeWXR reports WordPress's RSS with its export namespace.
func looksLikeWXR(body []byte) bool {
	head := body[:min(len(body), 64<<10)]
	return bytes.HasPrefix(head, []byte("<")) && bytes.Contains(head, []byte("<rss")) && bytes.Contains(head, []byte("wordpress.org/export/"))
}
