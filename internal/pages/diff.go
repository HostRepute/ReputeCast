package pages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/store"
)

// The publish dialog's "Since version 8, published 1 Oct by Sofia Rossi":
// what changed between the live version and the draft, block by block,
// line by line, and the settings that changed.

// PageDiffLine is a line of a block: removed, added or unchanged.
type PageDiffLine struct {
	Op   string `json:"op" enum:"-,+, "`
	Text string `json:"text"`
}

// PageDiffBlock is a block that changed.
type PageDiffBlock struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Kind  string         `json:"kind" enum:"changed,added,removed,moved"`
	Lines []PageDiffLine `json:"lines"`
}

// PageDiffSetting is a setting that changed.
type PageDiffSetting struct {
	Key    string `json:"key"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// PageDiff is what publishing would change.
type PageDiff struct {
	// First: the page was never published; Since is the live version.
	First       bool              `json:"first"`
	Since       int32             `json:"since"`
	PublishedAt *time.Time        `json:"publishedAt,omitempty"`
	PublishedBy string            `json:"publishedBy,omitempty"`
	Version     int32             `json:"version"`
	Blocks      []PageDiffBlock   `json:"blocks"`
	Settings    []PageDiffSetting `json:"settings"`
}

// Diff compares a page's draft with its live version.
func (s *Service) Diff(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PageDiff, error) {
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return PageDiff{}, err
	}
	out := PageDiff{Version: pg.Version, Blocks: []PageDiffBlock{}, Settings: []PageDiffSetting{}}
	if pg.LiveVersion == nil {
		out.First = true
		return out, nil
	}
	v, err := s.Q.PageVersionGet(ctx, store.PageVersionGetParams{PageID: pg.ID, Version: *pg.LiveVersion})
	if errors.Is(err, pgx.ErrNoRows) {
		out.First = true
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Since, out.PublishedAt, out.PublishedBy = v.PageVersion.Version, v.PageVersion.PublishedAt, v.PublishedByName
	out.Blocks = DiffBlocks(v.PageVersion.Html, pg.Html)
	before, after := parseSettings(v.PageVersion.Settings), settingsOf(pg)
	add := func(key, a, b string) {
		if a != b {
			out.Settings = append(out.Settings, PageDiffSetting{Key: key, Before: a, After: b})
		}
	}
	add("name", v.PageVersion.Name, pg.Name)
	add("slug", "/"+v.PageVersion.Slug, "/"+pg.Slug)
	add("seoTitle", before.SEOTitle, after.SEOTitle)
	add("seoDescription", before.SEODescription, after.SEODescription)
	add("socialImage", before.SocialImage, after.SocialImage)
	add("visibility", before.Visibility, after.Visibility)
	add("tiers", strings.Join(before.Tiers, ","), strings.Join(after.Tiers, ","))
	add("hideHeader", fmt.Sprint(before.HideHeader), fmt.Sprint(after.HideHeader))
	add("showInNav", fmt.Sprint(before.ShowInNav), fmt.Sprint(after.ShowInNav))
	add("navLabel", before.NavLabel, after.NavLabel)
	return out, nil
}

// blockOf is a top-level block's id, name and lines.
type blockOf struct {
	id, name string
	lines    []string
}

func blocksOf(src string) []blockOf {
	var out []blockOf
	for i, n := range parse(src) {
		if n.Type != html.ElementNode {
			continue
		}
		b := blockOf{id: attr(n, "data-pb-id"), name: attr(n, "data-pb-name")}
		if b.id == "" {
			b.id = fmt.Sprintf("#%d", i)
		}
		if b.name == "" {
			b.name = n.Data
		}
		b.lines = linesOf(n)
		out = append(out, b)
	}
	return out
}

// textBlocks are the elements whose text is one line of a diff.
var textBlocks = map[string]bool{"h1": true, "h2": true, "h3": true, "h4": true, "p": true, "li": true, "summary": true, "blockquote": true,
	"figcaption": true, "dt": true, "dd": true}

// linesOf are a block's lines: its texts, then what its live blocks say.
func linesOf(root *html.Node) []string {
	var out []string
	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if attr(n, "aria-hidden") == "true" {
				// Initials and icons: nothing a reader reads.
				return
			}
			if textBlocks[n.Data] {
				if t := lineText(n); t != "" {
					out = append(out, t)
				}
				return
			}
			if spec, ok := blockSpecs[n.Data]; ok {
				var parts []string
				for _, a := range n.Attr {
					if _, known := spec[a.Key]; known && a.Val != "" {
						parts = append(parts, a.Key+": "+a.Val)
					}
				}
				out = append(out, blockLabel(n.Data)+" · "+strings.Join(parts, " · "))
				return
			}
			if n.Data == "img" {
				out = append(out, "Image · "+fileName(attr(n, "src"))+" · alt: "+attr(n, "alt"))
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(root)
	return out
}

// lineText is an element's text as one diff line, without what's hidden
// from readers; an attribution's parts read "Hannah Weiss · Weekly Digest
// Insider".
func lineText(n *html.Node) string {
	var parts []string
	var walk func(*html.Node)
	walk = func(c *html.Node) {
		if c.Type == html.ElementNode && attr(c, "aria-hidden") == "true" {
			return
		}
		if c.Type == html.TextNode {
			if t := strings.Join(strings.Fields(c.Data), " "); t != "" {
				parts = append(parts, t)
			}
			return
		}
		for k := c.FirstChild; k != nil; k = k.NextSibling {
			walk(k)
		}
	}
	walk(n)
	sep := " "
	if n.Data == "figcaption" {
		sep = " · "
	}
	return strings.Join(parts, sep)
}

func blockLabel(el string) string {
	switch el {
	case ElForm:
		return "Signup form"
	case ElCountdown:
		return "Countdown"
	case ElEvent:
		return "Event details"
	case ElProduct:
		return "Product"
	case ElProducts:
		return "Product grid"
	case ElStoreProduct:
		return "Store product"
	case ElTiers:
		return "Pricing tiers"
	case ElPosts:
		return "Recent posts"
	case ElVideo:
		return "Video"
	}
	return el
}

// DiffBlocks compares two versions of a page's blocks.
func DiffBlocks(before, after string) []PageDiffBlock {
	a, b := blocksOf(before), blocksOf(after)
	byID := map[string]blockOf{}
	for _, x := range a {
		byID[x.id] = x
	}
	inAfter := map[string]bool{}
	out := []PageDiffBlock{}
	for _, x := range b {
		inAfter[x.id] = true
		old, ok := byID[x.id]
		if !ok {
			d := PageDiffBlock{ID: x.id, Name: x.name, Kind: "added", Lines: []PageDiffLine{}}
			for _, l := range x.lines {
				d.Lines = append(d.Lines, PageDiffLine{Op: "+", Text: l})
			}
			out = append(out, d)
			continue
		}
		if lines := diffLines(old.lines, x.lines); len(lines) > 0 {
			out = append(out, PageDiffBlock{ID: x.id, Name: x.name, Kind: "changed", Lines: lines})
		}
	}
	for _, x := range a {
		if !inAfter[x.id] {
			d := PageDiffBlock{ID: x.id, Name: x.name, Kind: "removed", Lines: []PageDiffLine{}}
			for _, l := range x.lines {
				d.Lines = append(d.Lines, PageDiffLine{Op: "-", Text: l})
			}
			out = append(out, d)
		}
	}
	return out
}

// diffLines is a line diff by longest common subsequence: removed and
// added lines, in order, without the unchanged ones.
func diffLines(a, b []string) []PageDiffLine {
	n, m := len(a), len(b)
	if n > 400 || m > 400 {
		a, b = a[:min(n, 400)], b[:min(m, 400)]
		n, m = len(a), len(b)
	}
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []PageDiffLine
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, PageDiffLine{Op: "-", Text: a[i]})
			i++
		default:
			out = append(out, PageDiffLine{Op: "+", Text: b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, PageDiffLine{Op: "-", Text: a[i]})
	}
	for ; j < m; j++ {
		out = append(out, PageDiffLine{Op: "+", Text: b[j]})
	}
	return out
}
