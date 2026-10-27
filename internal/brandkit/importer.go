package brandkit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
)

// BrandKitImportInput asks for a website's logo, colours and fonts.
type BrandKitImportInput struct {
	Website string `json:"website" minLength:"3" maxLength:"200" doc:"acme.io or https://acme.io"`
}

// BrandImportColor is a colour found on the site.
type BrandImportColor struct {
	Hex string `json:"hex"`
	// Used is false when the kit doesn't use it (Notes say why).
	Used bool `json:"used"`
}

// BrandKitImport is what a website shows, as a kit to check: nothing is
// saved. Logos found are added to the media library so the kit can use them.
type BrandKitImport struct {
	Website   string             `json:"website"`
	FoundAt   time.Time          `json:"foundAt"`
	LogoLight *BrandLogo         `json:"logoLight,omitempty"`
	LogoDark  *BrandLogo         `json:"logoDark,omitempty"`
	LogoAlt   string             `json:"logoAlt"`
	Style     BrandStyle         `json:"style"`
	Colors    []BrandImportColor `json:"colors"`
	Fonts     []string           `json:"fonts"`
	// Notes say what the import decided and why ("#6366F1 isn't used for
	// links…"), in the reader's language.
	Notes []string `json:"notes"`
}

const (
	pageLimit  = 2 << 20
	cssLimit   = 512 << 10
	stylesheet = 3
)

// normalize turns "acme.io" into https://acme.io/.
func normalize(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("bad url")
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u, nil
}

// fetch GETs a URL through the address guard, following up to three
// redirects (each checked again).
func (s *Service) fetch(ctx context.Context, u *url.URL, limit int64, accept string) ([]byte, *url.URL, string, error) {
	client := s.Guard.Client(10 * time.Second)
	for range 4 {
		if _, err := s.Guard.ValidateURL(u.String()); err != nil {
			return nil, u, "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, u, "", err
		}
		req.Header.Set("User-Agent", "ReputeCast brand kit import")
		req.Header.Set("Accept", accept)
		res, err := client.Do(req)
		if err != nil {
			return nil, u, "", err
		}
		if res.StatusCode >= 300 && res.StatusCode < 400 {
			loc := res.Header.Get("Location")
			_ = res.Body.Close()
			next, err := u.Parse(loc)
			if loc == "" || err != nil {
				return nil, u, "", fmt.Errorf("HTTP %d", res.StatusCode)
			}
			u = next
			continue
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, limit))
		_ = res.Body.Close()
		if err != nil {
			return nil, u, "", err
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return nil, u, "", fmt.Errorf("HTTP %d", res.StatusCode)
		}
		return body, u, res.Header.Get("Content-Type"), nil
	}
	return nil, u, "", errors.New("too many redirects")
}

// page is what the HTML of a site's home page holds.
type page struct {
	themeColor string
	styles     []string // inline <style> and style="" text
	sheets     []*url.URL
	fontLinks  []*url.URL
	logos      []logoRef
	svgLogo    bool
	title      string
}

type logoRef struct {
	src  *url.URL
	alt  string
	dark bool
}

var logoWord = regexp.MustCompile(`(?i)logo|brand`)
var darkWord = regexp.MustCompile(`(?i)white|inverse|reversed|negative|dark|on-dark|ondark`)

func attrs(t html.Token) map[string]string {
	m := map[string]string{}
	for _, a := range t.Attr {
		m[strings.ToLower(a.Key)] = a.Val
	}
	return m
}

func parsePage(base *url.URL, body []byte) page {
	var p page
	z := html.NewTokenizer(bytes.NewReader(body))
	inStyle, inTitle, inSVGLogo := false, false, 0
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return p
		}
		t := z.Token()
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			a := attrs(t)
			if st := a["style"]; st != "" {
				p.styles = append(p.styles, "x{"+st+"}")
			}
			switch t.Data {
			case "meta":
				if strings.EqualFold(a["name"], "theme-color") {
					p.themeColor = a["content"]
				}
			case "link":
				rel := strings.ToLower(a["rel"])
				href, err := base.Parse(a["href"])
				if a["href"] == "" || err != nil {
					continue
				}
				switch {
				case strings.Contains(href.Host, "fonts.googleapis.com") || strings.Contains(href.Host, "fonts.bunny.net"):
					p.fontLinks = append(p.fontLinks, href)
				case strings.Contains(rel, "stylesheet") && href.Host == base.Host && len(p.sheets) < stylesheet:
					p.sheets = append(p.sheets, href)
				}
			case "style":
				inStyle = tt == html.StartTagToken
			case "title":
				inTitle = tt == html.StartTagToken
			case "img":
				hint := a["class"] + " " + a["id"] + " " + a["alt"] + " " + a["src"]
				if !logoWord.MatchString(hint) || a["src"] == "" || strings.HasPrefix(a["src"], "data:") {
					continue
				}
				src, err := base.Parse(a["src"])
				if err != nil {
					continue
				}
				p.logos = append(p.logos, logoRef{src: src, alt: a["alt"], dark: darkWord.MatchString(a["class"] + " " + a["src"])})
			case "svg":
				if tt == html.StartTagToken && logoWord.MatchString(a["class"]+" "+a["id"]+" "+a["aria-label"]) {
					inSVGLogo++
					p.svgLogo = true
				}
			}
		case html.TextToken:
			if inStyle {
				p.styles = append(p.styles, t.Data)
			}
			if inTitle && p.title == "" {
				p.title = strings.TrimSpace(t.Data)
			}
		case html.EndTagToken:
			switch t.Data {
			case "style":
				inStyle = false
			case "title":
				inTitle = false
			}
		}
	}
}

var (
	hexInCSS   = regexp.MustCompile(`#([0-9A-Fa-f]{6}|[0-9A-Fa-f]{3})\b`)
	rgbInCSS   = regexp.MustCompile(`rgba?\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(?:,\s*(?:1|1\.0+)\s*)?\)`)
	cssRule    = regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`)
	fontFamily = regexp.MustCompile(`(?i)font-family\s*:\s*([^;}]+)`)
	brandVar   = regexp.MustCompile(`(?i)--[a-z0-9-]*(primary|brand|accent)[a-z0-9-]*\s*:\s*(#[0-9A-Fa-f]{3,6}\b|rgba?\([^)]*\))`)
)

func hex6(raw string) string {
	raw = strings.TrimPrefix(strings.TrimSpace(raw), "#")
	if len(raw) == 3 {
		raw = string([]byte{raw[0], raw[0], raw[1], raw[1], raw[2], raw[2]})
	}
	if len(raw) != 6 {
		return ""
	}
	if _, err := strconv.ParseUint(raw, 16, 32); err != nil {
		return ""
	}
	return "#" + strings.ToUpper(raw)
}

func rgbHex(m []string) string {
	var v [3]int
	for i := range 3 {
		n, err := strconv.Atoi(m[i+1])
		if err != nil || n > 255 {
			return ""
		}
		v[i] = n
	}
	return fmt.Sprintf("#%02X%02X%02X", v[0], v[1], v[2])
}

// hsl of #RRGGBB: saturation and lightness, 0 to 1.
func satLight(hex string) (float64, float64) {
	n, _ := strconv.ParseUint(hex[1:], 16, 32)
	r, g, b := float64(n>>16&255)/255, float64(n>>8&255)/255, float64(n&255)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l := (mx + mn) / 2
	if mx == mn {
		return 0, l
	}
	d := mx - mn
	if l > 0.5 {
		return d / (2 - mx - mn), l
	}
	return d / (mx + mn), l
}

// colorCount ranks the colours of the CSS: brand variables first, then by
// how often they're used.
func colorCount(css string) []string {
	count := map[string]int{}
	for _, m := range brandVar.FindAllStringSubmatch(css, -1) {
		h := hex6(m[2])
		if sub := rgbInCSS.FindStringSubmatch(m[2]); sub != nil {
			h = rgbHex(sub)
		}
		if h != "" {
			count[h] += 1000
		}
	}
	for _, m := range hexInCSS.FindAllStringSubmatch(css, -1) {
		if h := hex6(m[1]); h != "" {
			count[h]++
		}
	}
	for _, m := range rgbInCSS.FindAllStringSubmatch(css, -1) {
		if h := rgbHex(m); h != "" {
			count[h]++
		}
	}
	out := make([]string, 0, len(count))
	for h := range count {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		if count[out[i]] != count[out[j]] {
			return count[out[i]] > count[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

// googleFamilies reads the families of a Google Fonts (or Bunny Fonts)
// link. The query is split by hand: css2 links hold semicolons
// (wght@400;700), which url.ParseQuery refuses.
func googleFamilies(u *url.URL) []string {
	var out []string
	for _, pair := range strings.Split(u.RawQuery, "&") {
		raw, ok := strings.CutPrefix(pair, "family=")
		if !ok {
			continue
		}
		f, err := url.QueryUnescape(raw)
		if err != nil {
			continue
		}
		for _, part := range strings.Split(f, "|") {
			name := strings.TrimSpace(strings.SplitN(part, ":", 2)[0])
			name = strings.Join(strings.Fields(strings.ReplaceAll(name, "+", " ")), " ")
			if name != "" && fontName.MatchString(name) {
				out = append(out, name)
			}
		}
	}
	return out
}

var fontName = regexp.MustCompile(`^[A-Za-z0-9 ]{1,60}$`)
var serifName = regexp.MustCompile(`(?i)serif|playfair|merriweather|lora|georgia|garamond|baskerville|times|crimson|libre caslon|fraunces|spectral`)

// fontFor is a kit font for a family: its stylesheet when it's a Google or
// Bunny font, and the matching email-safe fallback.
func fontFor(name string, link *url.URL) BrandFont {
	f := BrandFont{Name: name, Fallback: "Arial, Helvetica, sans-serif"}
	if serifName.MatchString(name) && !strings.Contains(strings.ToLower(name), "sans") {
		f.Fallback = "Georgia, 'Times New Roman', serif"
	}
	if link != nil {
		u := *link
		u.RawQuery = "family=" + strings.ReplaceAll(name, " ", "+") + ":wght@400;700&display=swap"
		f.URL = u.String()
	}
	return f
}

// Import reads a website's logo, colours and fonts into a kit to check.
func (s *Service) Import(ctx context.Context, ws *store.Workspace, p *authz.Principal, in BrandKitImportInput) (BrandKitImport, error) {
	l := i18n.LangFrom(ctx)
	site, err := normalize(in.Website)
	if err != nil {
		return BrandKitImport{}, api.Invalid("body.website", mBadWebsite.Text(l, nil))
	}
	body, final, _, err := s.fetch(ctx, site, pageLimit, "text/html")
	if err != nil {
		return BrandKitImport{}, api.Err(http.StatusBadGateway, "website_unreachable", mUnreachable.Text(l, i18n.Args{"site": site.Host, "reason": netguard.Describe(ctx, err)}))
	}
	pg := parsePage(final, body)
	css := strings.Join(pg.styles, "\n")
	for _, sheet := range pg.sheets {
		if b, _, _, err := s.fetch(ctx, sheet, cssLimit, "text/css"); err == nil {
			css += "\n" + string(b)
		}
	}
	host := strings.TrimPrefix(final.Host, "www.")
	out := BrandKitImport{Website: host, FoundAt: s.now(), Style: DefaultStyle(), Colors: []BrandImportColor{}, Fonts: []string{}, Notes: []string{}}

	// Colours: the theme colour and brand variables first.
	ranked := colorCount(css)
	if h := hex6(pg.themeColor); h != "" {
		ranked = append([]string{h}, removeHex(ranked, h)...)
	}
	var chromatic, darks, tints []string
	for _, h := range ranked {
		sat, light := satLight(h)
		switch {
		case sat >= 0.25 && light > 0.15 && light < 0.85:
			chromatic = append(chromatic, h)
		case light < 0.3 && Contrast(h, "#FFFFFF") >= 7:
			darks = append(darks, h)
		case light > 0.93 && h != "#FFFFFF":
			tints = append(tints, h)
		}
	}
	used := map[string]bool{}
	if len(chromatic) > 0 {
		out.Style.Colors.Button = chromatic[0]
		for _, h := range chromatic {
			if Contrast(h, "#FFFFFF") >= 4.5 {
				out.Style.Colors.Button = h
				break
			}
		}
		used[out.Style.Colors.Button] = true
		out.Style.Colors.Link = out.Style.Colors.Button
		for i, h := range chromatic {
			c := Contrast(h, "#FFFFFF")
			if c >= 4.5 {
				out.Style.Colors.Link = h
				used[h] = true
				break
			}
			if i == 0 {
				out.Notes = append(out.Notes, mLinkContrast.Text(l, i18n.Args{"color": h, "ratio": l.Region.Decimal(round2(c), 2)}))
			}
		}
	}
	if len(darks) > 0 {
		out.Style.Colors.Text = darks[0]
		used[darks[0]] = true
	}
	if len(tints) > 0 {
		out.Style.Colors.Background = tints[0]
		used[tints[0]] = true
	}
	for _, h := range append(append(append([]string{}, chromatic...), darks...), tints...) {
		if len(out.Colors) >= 6 {
			break
		}
		out.Colors = append(out.Colors, BrandImportColor{Hex: h, Used: used[h]})
	}

	// Fonts: the ones the site loads, the heading's first.
	var families []string
	links := map[string]*url.URL{}
	for _, link := range pg.fontLinks {
		for _, f := range googleFamilies(link) {
			if _, ok := links[f]; !ok {
				families = append(families, f)
				links[f] = link
			}
		}
	}
	heading, bodyFont := "", ""
	for _, m := range cssRule.FindAllStringSubmatch(css, -1) {
		ff := fontFamily.FindStringSubmatch(m[2])
		if ff == nil {
			continue
		}
		first := strings.Trim(strings.TrimSpace(strings.SplitN(ff[1], ",", 2)[0]), `'"`)
		if !fontName.MatchString(first) {
			continue
		}
		sel := strings.ToLower(m[1])
		switch {
		case heading == "" && regexp.MustCompile(`\bh[1-3]\b|heading|title|display`).MatchString(sel):
			heading = first
		case bodyFont == "" && regexp.MustCompile(`\bbody\b|\bhtml\b|\bp\b|:root`).MatchString(sel):
			bodyFont = first
		}
	}
	if heading == "" || bodyFont == "" {
		// Without CSS saying which is which: a serif face heads, a sans one reads.
		for _, f := range families {
			isSerif := serifName.MatchString(f) && !strings.Contains(strings.ToLower(f), "sans")
			if heading == "" && isSerif {
				heading = f
			} else if bodyFont == "" && !isSerif {
				bodyFont = f
			}
		}
	}
	if bodyFont == "" && len(families) > 0 {
		bodyFont = families[0]
	}
	if heading == "" {
		heading = bodyFont
	}
	if heading != "" {
		out.Style.Heading = fontFor(heading, links[heading])
		out.Fonts = append(out.Fonts, heading)
	}
	if bodyFont != "" {
		out.Style.Body = fontFor(bodyFont, links[bodyFont])
		if bodyFont != heading {
			out.Fonts = append(out.Fonts, bodyFont)
		}
	}

	// Logos: an image named for the logo, a dark-mode one when there is.
	out.LogoAlt = pg.title
	if i := strings.IndexAny(out.LogoAlt, "|–—-·:"); i > 0 {
		out.LogoAlt = strings.TrimSpace(out.LogoAlt[:i])
	}
	for _, ref := range pg.logos {
		if (ref.dark && out.LogoDark != nil) || (!ref.dark && out.LogoLight != nil) {
			continue
		}
		logo, why := s.storeLogo(ctx, ws, p, ref)
		if logo == nil {
			if why != "" {
				out.Notes = append(out.Notes, why)
			}
			continue
		}
		if ref.alt != "" && out.LogoAlt == "" {
			out.LogoAlt = ref.alt
		}
		if ref.dark {
			out.LogoDark = logo
		} else {
			out.LogoLight = logo
		}
	}
	if out.LogoLight == nil && out.LogoDark == nil && pg.svgLogo {
		out.Notes = append(out.Notes, mSVGLogo.Text(l, i18n.Args{"site": host}))
	}
	return out, nil
}

func removeHex(list []string, h string) []string {
	out := list[:0:0]
	for _, x := range list {
		if x != h {
			out = append(out, x)
		}
	}
	return out
}

// storeLogo adds a logo found on the site to the media library. why says
// what was wrong with it otherwise.
func (s *Service) storeLogo(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref logoRef) (*BrandLogo, string) {
	l := i18n.LangFrom(ctx)
	if s.Media == nil {
		return nil, ""
	}
	if strings.HasSuffix(strings.ToLower(ref.src.Path), ".svg") {
		return nil, mSVGLogo.Text(l, i18n.Args{"site": strings.TrimPrefix(ref.src.Host, "www.")})
	}
	data, _, _, err := s.fetch(ctx, ref.src, media.MaxUploadBytes, "image/png, image/jpeg, image/gif")
	if err != nil {
		return nil, ""
	}
	name := path.Base(ref.src.Path)
	a, err := s.Media.Upload(ctx, ws, actor(p), name, data)
	if err != nil {
		var pr *api.Problem
		if errors.As(err, &pr) {
			return nil, pr.Detail
		}
		return nil, ""
	}
	return &BrandLogo{MediaID: a.ID, URL: a.URL, FileName: a.FileName, Width: a.Width, Height: a.Height}, ""
}
