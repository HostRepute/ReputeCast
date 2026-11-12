package importer

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Ghost's export (Settings › Labs › Export your content) is one JSON file:
// {"db":[{"meta":{…},"data":{"posts":[…],"posts_meta":[…],"tags":[…],
// "posts_tags":[…],"users":[…],"posts_authors":[…],"products":[…],
// "posts_products":[…]}}]}; exports from Ghost 1 have "data" at the top.
// A post's html is its rendered body, with __GHOST_URL__ standing for the
// site's address in links and images; status is published, draft,
// scheduled or sent (email only); visibility public, members, paid or
// tiers (posts_products names them). meta_title and meta_description are
// on posts_meta since Ghost 3, on the post before.

// ghostURL stands for the site's address in a Ghost export.
const ghostURL = "__GHOST_URL__"

// ghostID is an id written as a string, or as a number in old exports.
type ghostID string

func (g *ghostID) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*g = ghostID(s)
		return nil
	}
	if string(b) == "null" {
		*g = ""
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*g = ghostID(n.String())
	return nil
}

// ghostFlag is a boolean an old export writes as 0 or 1.
type ghostFlag bool

func (g *ghostFlag) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	*g = ghostFlag(s == "true" || s == "1")
	return nil
}

type ghostExport struct {
	DB   []struct{ Data ghostData } `json:"db"`
	Data *ghostData                 `json:"data"`
}

type ghostData struct {
	Posts         []ghostPost `json:"posts"`
	PostsMeta     []ghostMeta `json:"posts_meta"`
	Tags          []ghostTag  `json:"tags"`
	PostsTags     []ghostRel  `json:"posts_tags"`
	Users         []ghostUser `json:"users"`
	PostsAuthors  []ghostRel  `json:"posts_authors"`
	Products      []ghostTag  `json:"products"`
	PostsProducts []ghostRel  `json:"posts_products"`
}

type ghostPost struct {
	ID              ghostID   `json:"id"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	HTML            *string   `json:"html"`
	FeatureImage    *string   `json:"feature_image"`
	Type            string    `json:"type"`
	Page            ghostFlag `json:"page"`
	Status          string    `json:"status"`
	Visibility      string    `json:"visibility"`
	PublishedAt     *string   `json:"published_at"`
	CustomExcerpt   *string   `json:"custom_excerpt"`
	CanonicalURL    *string   `json:"canonical_url"`
	MetaTitle       *string   `json:"meta_title"`
	MetaDescription *string   `json:"meta_description"`
	AuthorID        ghostID   `json:"author_id"`
}

type ghostMeta struct {
	PostID          ghostID `json:"post_id"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
	FeatureImageAlt *string `json:"feature_image_alt"`
}

type ghostTag struct {
	ID         ghostID `json:"id"`
	Name       string  `json:"name"`
	Visibility string  `json:"visibility"`
	Type       string  `json:"type"`
}

type ghostUser struct {
	ID   ghostID `json:"id"`
	Name string  `json:"name"`
}

// ghostRel joins a post to a tag (tag_id), an author (author_id) or a tier
// (product_id), in sort_order.
type ghostRel struct {
	PostID    ghostID `json:"post_id"`
	TagID     ghostID `json:"tag_id"`
	AuthorID  ghostID `json:"author_id"`
	ProductID ghostID `json:"product_id"`
	SortOrder int     `json:"sort_order"`
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// ghostFile finds the export JSON in a zip: a .json file at its root or in
// its one folder that is shaped as a Ghost export.
func ghostFile(zr *zip.Reader) *zip.File {
	for _, f := range zr.File {
		name := strings.TrimPrefix(f.Name, "./")
		if !strings.EqualFold(path.Ext(name), ".json") || strings.Count(name, "/") > 1 || strings.HasPrefix(path.Base(name), ".") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		ok := ghostShape(json.NewDecoder(rc))
		_ = rc.Close()
		if ok {
			return f
		}
	}
	return nil
}

// ghostShape reads a JSON document's start, as far as it takes to see a
// Ghost export's posts: {"db":[{…,"data":{…,"posts":[ or, in old exports,
// {…,"data":{…,"posts":[. Values before them are skipped, not kept.
func ghostShape(dec *json.Decoder) bool {
	if !delim(dec, '{') {
		return false
	}
	for dec.More() {
		key, ok := keyOf(dec)
		if !ok {
			return false
		}
		switch key {
		case "db":
			if !delim(dec, '[') || !delim(dec, '{') {
				return false
			}
			return hasPosts(dec)
		case "data":
			return postsIn(dec)
		default:
			if !skip(dec) {
				return false
			}
		}
	}
	return false
}

// hasPosts reads an object's keys up to "data", then its posts.
func hasPosts(dec *json.Decoder) bool {
	for dec.More() {
		key, ok := keyOf(dec)
		if !ok {
			return false
		}
		if key == "data" {
			return postsIn(dec)
		}
		if !skip(dec) {
			return false
		}
	}
	return false
}

// postsIn reads a data object up to its "posts" array.
func postsIn(dec *json.Decoder) bool {
	if !delim(dec, '{') {
		return false
	}
	for dec.More() {
		key, ok := keyOf(dec)
		if !ok {
			return false
		}
		if key == "posts" {
			return delim(dec, '[')
		}
		if !skip(dec) {
			return false
		}
	}
	return false
}

func delim(dec *json.Decoder, want json.Delim) bool {
	t, err := dec.Token()
	d, ok := t.(json.Delim)
	return err == nil && ok && d == want
}

func keyOf(dec *json.Decoder) (string, bool) {
	t, err := dec.Token()
	k, ok := t.(string)
	return k, err == nil && ok
}

// skip reads past one value.
func skip(dec *json.Decoder) bool {
	var v json.RawMessage
	return dec.Decode(&v) == nil
}

// ParseGhost reads a Ghost export: the JSON, or a zip holding it. Pages
// aren't posts. Links and images to the site itself become root-relative
// ("/my-post/", "/content/images/…"): on the site's own domain the old
// addresses lead to the imported posts, and a zip's images are copied.
func ParseGhost(data []byte) ([]Post, error) {
	body := trimBOM(data)
	if isZip(data) {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, ErrNotExport
		}
		f := ghostFile(zr)
		if f == nil {
			return nil, ErrNotExport
		}
		if body, err = readEntry(f); err != nil {
			return nil, err
		}
		body = trimBOM(body)
	}
	var ex ghostExport
	if err := json.Unmarshal(body, &ex); err != nil {
		return nil, ErrNotExport
	}
	var d ghostData
	switch {
	case len(ex.DB) > 0:
		d = ex.DB[0].Data
	case ex.Data != nil:
		d = *ex.Data
	default:
		return nil, ErrNotExport
	}
	meta := map[ghostID]ghostMeta{}
	for _, m := range d.PostsMeta {
		meta[m.PostID] = m
	}
	tags := map[ghostID]ghostTag{}
	for _, t := range d.Tags {
		tags[t.ID] = t
	}
	users := map[ghostID]string{}
	for _, u := range d.Users {
		users[u.ID] = strings.TrimSpace(u.Name)
	}
	products := map[ghostID]ghostTag{}
	for _, p := range d.Products {
		products[p.ID] = p
	}
	byPost := func(rels []ghostRel) map[ghostID][]ghostRel {
		out := map[ghostID][]ghostRel{}
		for _, r := range rels {
			out[r.PostID] = append(out[r.PostID], r)
		}
		for _, rs := range out {
			sort.SliceStable(rs, func(i, j int) bool { return rs[i].SortOrder < rs[j].SortOrder })
		}
		return out
	}
	postTags, postAuthors, postProducts := byPost(d.PostsTags), byPost(d.PostsAuthors), byPost(d.PostsProducts)
	out := []Post{}
	for _, gp := range d.Posts {
		if gp.Type == "page" || bool(gp.Page) || gp.ID == "" {
			continue
		}
		if len(out) == MaxPosts {
			return nil, ErrTooManyPosts
		}
		m := meta[gp.ID]
		p := Post{SourceID: string(gp.ID), Title: strings.TrimSpace(gp.Title), Slug: strings.ToLower(strings.TrimSpace(gp.Slug)),
			Draft: gp.Status != "published", PublishedAt: parseTime(str(gp.PublishedAt)), Visibility: Public, TierRefs: []string{}, Tags: []string{},
			Excerpt: str(gp.CustomExcerpt), CoverURL: siteRelative(str(gp.FeatureImage)), CoverAlt: str(m.FeatureImageAlt),
			SEOTitle: firstOf(str(m.MetaTitle), str(gp.MetaTitle)), SEODescription: firstOf(str(m.MetaDescription), str(gp.MetaDescription)),
			CanonicalURL: siteRelative(str(gp.CanonicalURL)), HTML: siteRelative(str(gp.HTML))}
		switch gp.Visibility {
		case "", "public":
		case "members":
			p.Visibility = Members
		default:
			// paid, tiers, or a filter naming tiers.
			p.Visibility = Paid
			if gp.Visibility != "paid" {
				for _, r := range postProducts[gp.ID] {
					if pr, ok := products[r.ProductID]; ok && pr.Type != "free" && pr.Name != "" {
						p.TierRefs = append(p.TierRefs, pr.Name)
					}
				}
			}
		}
		for _, r := range postTags[gp.ID] {
			t, ok := tags[r.TagID]
			// Internal tags (#hash) organise the site; readers never see them.
			if !ok || t.Visibility == "internal" || strings.HasPrefix(t.Name, "#") || strings.TrimSpace(t.Name) == "" {
				continue
			}
			p.Tags = append(p.Tags, strings.TrimSpace(t.Name))
		}
		if rs := postAuthors[gp.ID]; len(rs) > 0 {
			p.Author = users[rs[0].AuthorID]
		} else if gp.AuthorID != "" {
			p.Author = users[gp.AuthorID]
		}
		if p.Title == "" {
			p.Title = p.Slug
		}
		if p.Slug == "" {
			p.Slug = "post-" + strconv.Itoa(len(out)+1)
		} else {
			p.LegacyPaths = []string{LegacyKey(p.Slug, "")}
		}
		p.SourceURL = "/" + p.Slug + "/"
		out = append(out, p)
	}
	return out, nil
}

// siteRelative makes the export's own address root-relative.
func siteRelative(s string) string { return strings.ReplaceAll(s, ghostURL, "") }

func firstOf(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
