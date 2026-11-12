package importer

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"path"
	"strings"
)

// Substack's export (Settings › Exports) is a zip: posts.csv, one row a
// post (post_id, post_date, is_published, email_sent_at, type, audience,
// title, subtitle, podcast_url), and posts/<post_id>.html with each post's
// body, sometimes inside one folder. post_id is "<number>.<slug>"; the
// post's address on Substack is /p/<slug>. The export has neither authors
// nor tags.

// Substack's audiences and the tiers they stand for.
var substackAudience = map[string]struct {
	visibility string
	tier       string
}{
	"everyone":  {Public, ""},
	"only_free": {Members, ""},
	"only_paid": {Paid, "Paid subscribers"},
	"founding":  {Paid, "Founding members"},
}

// substackCSV finds posts.csv, at the zip's root or in its one folder.
func substackCSV(zr *zip.Reader) (*zip.File, bool) {
	for _, f := range zr.File {
		name := strings.TrimPrefix(f.Name, "./")
		if path.Base(name) == "posts.csv" && strings.Count(name, "/") <= 1 {
			return f, true
		}
	}
	return nil, false
}

// ParseSubstack reads a Substack export zip. Threads and pages aren't
// posts; a row without its HTML file has no body to import.
func ParseSubstack(data []byte) ([]Post, error) {
	if !isZip(data) {
		return nil, ErrNotExport
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrNotExport
	}
	csvFile, ok := substackCSV(zr)
	if !ok {
		return nil, ErrNotExport
	}
	dir := path.Dir(strings.TrimPrefix(csvFile.Name, "./"))
	if dir == "." {
		dir = ""
	} else {
		dir += "/"
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[strings.TrimPrefix(f.Name, "./")] = f
	}
	raw, err := readEntry(csvFile)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(bytes.NewReader(trimBOM(raw)))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil || len(rows) == 0 {
		return nil, ErrNotExport
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	if _, ok := col["post_id"]; !ok {
		return nil, ErrNotExport
	}
	cell := func(row []string, name string) string {
		if i, ok := col[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}
	out := []Post{}
	for _, row := range rows[1:] {
		id := cell(row, "post_id")
		if id == "" {
			continue
		}
		switch strings.ToLower(cell(row, "type")) {
		case "thread", "page":
			continue
		}
		f, ok := files[dir+"posts/"+id+".html"]
		if !ok {
			continue
		}
		body, err := readEntry(f)
		if err != nil {
			return nil, err
		}
		if len(out) == MaxPosts {
			return nil, ErrTooManyPosts
		}
		slug := id
		if num, after, found := strings.Cut(id, "."); found && num != "" && strings.Trim(num, "0123456789") == "" {
			slug = after
		}
		p := Post{SourceID: id, Title: unescape(cell(row, "title")), Slug: strings.ToLower(slug), Excerpt: unescape(cell(row, "subtitle")),
			Draft: !strings.EqualFold(cell(row, "is_published"), "true"), Visibility: Public, TierRefs: []string{}, Tags: []string{},
			HTML: string(body)}
		if a, ok := substackAudience[strings.ToLower(cell(row, "audience"))]; ok {
			p.Visibility = a.visibility
			if a.tier != "" {
				p.TierRefs = []string{a.tier}
			}
		}
		p.PublishedAt = parseTime(cell(row, "post_date"))
		if p.PublishedAt == nil {
			p.PublishedAt = parseTime(cell(row, "email_sent_at"))
		}
		if p.Title == "" {
			p.Title = slug
		}
		p.SourceURL = "/p/" + p.Slug
		p.LegacyPaths = []string{LegacyKey("p/"+p.Slug, "")}
		out = append(out, p)
	}
	return out, nil
}
