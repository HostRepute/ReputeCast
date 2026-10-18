package pages

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/html"
)

// Blocks that show live data are custom elements in the page's HTML: the
// page builder edits their attributes in its Block tab, and the site draws
// them on every request (render.go) with the product's current price, the
// form's fields, the time left before a countdown ends.
const (
	ElForm         = "rc-form"          // a signup form: one of the workspace's forms
	ElCountdown    = "rc-countdown"     // to a date and time in a time zone
	ElEvent        = "rc-event"         // a webinar's or an event's details
	ElProduct      = "rc-product"       // one digital product with its Buy button
	ElProducts     = "rc-products"      // a product grid: digital or store products
	ElStoreProduct = "rc-store-product" // one product of a connected store (phase 7, X20)
	ElTiers        = "rc-tiers"         // the paid tiers (phase 6)
	ElPosts        = "rc-posts"         // recent posts of a tag
	ElVideo        = "rc-video"         // YouTube or Vimeo, loaded only when the reader clicks
)

// blockSpec is a custom element's attributes and how each one is checked.
type blockSpec map[string]func(string) (string, bool)

func (b blockSpec) clean(key, val string) (string, bool) {
	f, ok := b[key]
	if !ok {
		return "", false
	}
	return f(val)
}

var (
	dateRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	clockRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
	localRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T([01]\d|2[0-3]):[0-5]\d$`)
)

func anyUUID(v string) (string, bool) {
	if v == "" {
		return "", true
	}
	id, err := uuid.Parse(v)
	return id.String(), err == nil
}

func uuidList(v string) (string, bool) {
	var out []string
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			return "", false
		}
		out = append(out, id.String())
	}
	if len(out) > 12 {
		out = out[:12]
	}
	return strings.Join(out, ","), true
}

func text(n int) func(string) (string, bool) {
	return func(v string) (string, bool) { return clip(strings.Join(strings.Fields(v), " "), n), true }
}

func oneOf(vals ...string) func(string) (string, bool) {
	return func(v string) (string, bool) {
		for _, x := range vals {
			if v == x {
				return v, true
			}
		}
		return "", false
	}
}

func flag(v string) (string, bool) { return v, v == "1" || v == "0" }

func small(lo, hi int) func(string) (string, bool) {
	return func(v string) (string, bool) {
		n, err := strconv.Atoi(v)
		return strconv.Itoa(n), err == nil && n >= lo && n <= hi
	}
}

func timezone(v string) (string, bool) {
	if v == "" || len(v) > 64 {
		return "", false
	}
	_, err := time.LoadLocation(v)
	return v, err == nil
}

func httpsURL(v string) (string, bool) {
	if v == "" {
		return "", true
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || len(v) > 1000 {
		return "", false
	}
	return v, true
}

func blockRef(v string) (string, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	return v, v == "" || idRe.MatchString(v)
}

var blockSpecs = map[string]blockSpec{
	ElForm: {"form": anyUUID, "heading": text(120), "hint": text(300), "bg": oneOf("white", "lavender", "indigo", "ink")},
	ElCountdown: {"at": func(v string) (string, bool) { return v, localRe.MatchString(v) }, "tz": timezone,
		"units": func(v string) (string, bool) {
			for _, u := range strings.Split(v, ",") {
				if u != "days" && u != "hours" && u != "minutes" && u != "seconds" {
					return "", false
				}
			}
			return v, v != ""
		},
		"end": oneOf("hide", "message", "reveal"), "message": text(200), "reveal": blockRef, "hide": blockRef, "label": text(80)},
	ElEvent: {"date": func(v string) (string, bool) { return v, dateRe.MatchString(v) }, "start": func(v string) (string, bool) { return v, clockRe.MatchString(v) },
		"end": func(v string) (string, bool) { return v, v == "" || clockRe.MatchString(v) }, "tz": timezone, "local": flag,
		"where": oneOf("online", "venue"), "platform": text(40), "venue": text(200), "link": httpsURL, "calendar": flag, "title": text(160),
		"reminders": text(160)},
	ElProduct: {"product": anyUUID, "button": text(60), "price": flag, "offer": flag, "layout": oneOf("box", "final"), "eyebrow": text(60),
		"bg": oneOf("white", "lavender", "indigo", "ink")},
	ElProducts:     {"source": oneOf("digital", "store"), "products": uuidList, "count": small(1, 12), "button": text(40)},
	ElStoreProduct: {"store": text(80), "product": text(120), "button": text(60)},
	ElTiers:        {"tiers": uuidList, "period": oneOf("month", "year")},
	ElPosts:        {"tag": func(v string) (string, bool) { return v, v == "" || idRe.MatchString(v) }, "count": small(1, 6), "heading": text(120)},
	ElVideo:        {"url": videoURL, "title": text(160)},
}

// videoURL keeps YouTube and Vimeo addresses.
func videoURL(v string) (string, bool) {
	if v == "" {
		return "", true
	}
	_, _, ok := VideoOf(v)
	return v, ok
}

var (
	youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{6,20}$`)
	vimeoID   = regexp.MustCompile(`^\d{4,12}$`)
)

// VideoOf reads a YouTube or Vimeo address: the site ("YouTube", "Vimeo")
// and the video's id.
func VideoOf(raw string) (site, id string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", "", false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	switch host {
	case "youtube.com", "m.youtube.com", "youtube-nocookie.com":
		id = u.Query().Get("v")
		if strings.HasPrefix(u.Path, "/embed/") || strings.HasPrefix(u.Path, "/shorts/") || strings.HasPrefix(u.Path, "/live/") {
			id = strings.Trim(u.Path[strings.Index(u.Path[1:], "/")+1:], "/")
		}
		return "YouTube", id, youtubeID.MatchString(id)
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
		return "YouTube", id, youtubeID.MatchString(id)
	case "vimeo.com", "player.vimeo.com":
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		id = parts[len(parts)-1]
		return "Vimeo", id, vimeoID.MatchString(id)
	}
	return "", "", false
}

// Usage is what a page's blocks refer to and contain: the publish checks,
// the status bar's counts and the forms' "Used on pages".
type Usage struct {
	Blocks   int
	Products []uuid.UUID
	// BuyButtons counts the product blocks and grids (they need Stripe).
	BuyButtons int
	Forms      []uuid.UUID
	Countdowns []Moment
	Events     []Moment
	Images     int
	// ImagesNoAlt are the images without alt text, by their address.
	ImagesNoAlt []string
	Links       []Link
	// FirstHeading is the first block's top heading level (1 for an h1), 0
	// without a heading.
	FirstHeading int
	Stores       int
	Tiers        int
	Videos       int
}

// Moment is a countdown's end or an event's start, in its time zone.
type Moment struct {
	At    time.Time
	Label string
}

// Link is a link of the page.
type Link struct {
	Text string
	Href string
}

// Analyze reads a page's HTML for its checks and counts.
func Analyze(src string) Usage {
	nodes := parse(src)
	var u Usage
	seenProduct := map[uuid.UUID]bool{}
	seenForm := map[uuid.UUID]bool{}
	addProduct := func(v string) {
		if id, err := uuid.Parse(v); err == nil && !seenProduct[id] {
			seenProduct[id] = true
			u.Products = append(u.Products, id)
		}
	}
	for _, n := range nodes {
		if n.Type == html.ElementNode && n.Data == "section" {
			u.Blocks++
		}
	}
	first := true
	for _, n := range nodes {
		if n.Type != html.ElementNode {
			continue
		}
		if first {
			first = false
			walk([]*html.Node{n}, func(e *html.Node) bool {
				if u.FirstHeading == 0 {
					switch e.Data {
					case "h1":
						u.FirstHeading = 1
					case "h2":
						u.FirstHeading = 2
					case "h3":
						u.FirstHeading = 3
					}
				}
				return true
			})
		}
	}
	walk(nodes, func(n *html.Node) bool {
		switch n.Data {
		case ElProduct:
			u.BuyButtons++
			addProduct(attr(n, "product"))
		case ElProducts:
			if attr(n, "source") != "store" {
				u.BuyButtons++
				for _, p := range strings.Split(attr(n, "products"), ",") {
					addProduct(p)
				}
			} else {
				u.Stores++
			}
		case ElStoreProduct:
			u.Stores++
		case ElTiers:
			u.Tiers++
		case ElVideo:
			u.Videos++
		case ElForm:
			if id, err := uuid.Parse(attr(n, "form")); err == nil && !seenForm[id] {
				seenForm[id] = true
				u.Forms = append(u.Forms, id)
			}
		case ElCountdown:
			if at, ok := Local(attr(n, "at"), attr(n, "tz")); ok {
				u.Countdowns = append(u.Countdowns, Moment{At: at, Label: attr(n, "label")})
			}
		case ElEvent:
			if at, ok := Local(attr(n, "date")+"T"+attr(n, "start"), attr(n, "tz")); ok {
				u.Events = append(u.Events, Moment{At: at, Label: attr(n, "title")})
			}
		case "img":
			u.Images++
			if strings.TrimSpace(attr(n, "alt")) == "" {
				u.ImagesNoAlt = append(u.ImagesNoAlt, attr(n, "src"))
			}
		case "a":
			u.Links = append(u.Links, Link{Text: textOf(n), Href: attr(n, "href")})
		}
		return true
	})
	return u
}

// Local reads "2026-10-06T09:00" in a time zone.
func Local(at, tz string) (time.Time, bool) {
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", at, loc)
	return t, err == nil
}

// ProductsIn lists the digital products a page's HTML shows (product blocks
// and grids), for a product's "used on" and the composer's picker.
func ProductsIn(src string) []uuid.UUID { return Analyze(src).Products }

// FormsIn lists the signup forms a page's HTML uses (the form builder's
// "Used on pages").
func FormsIn(src string) []uuid.UUID { return Analyze(src).Forms }
