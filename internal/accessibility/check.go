package accessibility

import (
	"errors"
	"fmt"
	"html"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// Version is one language version of an email.
type Version struct {
	// Lang is its BCP 47 code ("en", "es").
	Lang string
	// MJML is the source; HTML the compiled email. With MJML the checks
	// compile it themselves, to point every issue at its element.
	MJML string
	HTML string
	// Source marks the version the others are translated from.
	Source bool
}

// PageTitle is what the link check read from a linked page.
type PageTitle struct {
	Title string
	// Lang is the page's language (<html lang>), "" when it doesn't say.
	Lang string
}

// Options are the workspace's settings and what the link check knows.
type Options struct {
	// Policy is "warn" or "block"; MinTextPx the smallest text size.
	Policy    string
	MinTextPx int
	// Titles are linked pages' titles by URL.
	Titles map[string]PageTitle
	Now    func() time.Time
}

// ErrCompile means an MJML version doesn't compile, so it can't be checked.
var ErrCompile = errors.New("the email doesn't compile")

type finding struct {
	key   string
	issue AccessibilityIssue
	ver   AccessibilityIssueVersion
	order int
}

type versionCheck struct {
	lang     string
	mode     string
	findings []finding
	counts   AccessibilityCounts
}

// Check runs every accessibility check on the compiled HTML of each
// language version and lists each issue once, with its languages.
func Check(versions []Version, opts Options) (AccessibilityReport, error) {
	start := time.Now()
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	if opts.MinTextPx < MinMinTextPx || opts.MinTextPx > MaxMinTextPx {
		opts.MinTextPx = DefaultMinTextPx
	}
	if opts.Policy != PolicyBlock {
		opts.Policy = PolicyWarn
	}
	r := AccessibilityReport{Issues: []AccessibilityIssue{}, Languages: []AccessibilityLanguage{}, Policy: opts.Policy, MinTextPx: opts.MinTextPx,
		Counts: AccessibilityCounts{Languages: []string{}, Dir: DirLTR}, Mode: "mjml"}
	// The source first: it names the issues.
	vs := append([]Version(nil), versions...)
	sort.SliceStable(vs, func(i, j int) bool { return vs[i].Source && !vs[j].Source })
	var checks []versionCheck
	for i, v := range vs {
		if v.Lang == "" {
			v.Lang = "en"
		}
		vc, err := checkVersion(v, opts)
		if err != nil {
			return r, err
		}
		checks = append(checks, vc)
		r.Languages = append(r.Languages, AccessibilityLanguage{Code: v.Lang, Name: LanguageName(v.Lang), Source: v.Source || i == 0})
		r.Counts.Languages = append(r.Counts.Languages, v.Lang)
		if i == 0 {
			lang := r.Counts.Languages
			r.Counts = vc.counts
			r.Counts.Languages = lang
			r.Mode = vc.mode
		}
	}
	r.Issues = merge(checks, r.Languages)
	for _, is := range r.Issues {
		if is.Severity == Serious {
			r.Serious++
		} else {
			r.Moderate++
		}
	}
	r.CheckedAt = now()
	r.DurationMs = int(time.Since(start) / time.Millisecond)
	return r, nil
}

func checkVersion(v Version, opts Options) (versionCheck, error) {
	vc := versionCheck{lang: v.Lang, mode: "html"}
	var d *doc
	src := v.HTML
	if strings.TrimSpace(v.MJML) != "" {
		parsed, err := parseDoc(v.MJML)
		if err != nil {
			return vc, fmt.Errorf("%w: %w", ErrCompile, err)
		}
		if parsed.body != nil {
			res := mjml.Compile(annotate(parsed))
			if len(res.Errors) > 0 {
				return vc, fmt.Errorf("%w: %s", ErrCompile, res.Errors[0].Message)
			}
			d, src, vc.mode = parsed, res.HTML, "mjml"
		}
	}
	finalized, add := Finalize(src, v.Lang)
	vc.counts = AccessibilityCounts{Languages: []string{v.Lang}, Dir: add.Dir, LayoutTables: add.LayoutTables, RoleAdded: add.RoleAdded, DataTables: add.DataTables}
	if strings.TrimSpace(src) == "" {
		return vc, nil
	}
	w := walkHTML(finalized)
	c := &checker{d: d, w: w, lang: v.Lang, opts: opts, vc: &vc}
	c.images()
	c.links()
	c.contrast()
	c.sizes()
	c.headings()
	return vc, nil
}

type checker struct {
	d     *doc
	w     *walker
	lang  string
	opts  Options
	vc    *versionCheck
	order int
}

func (c *checker) add(key string, is AccessibilityIssue, ver AccessibilityIssueVersion) {
	if is.Path == nil {
		is.Path = []int{}
	}
	if is.Paths == nil {
		is.Paths = [][]int{}
		if len(is.Path) > 0 {
			is.Paths = [][]int{is.Path}
		}
	}
	if is.Fix.Suggestions == nil {
		is.Fix.Suggestions = []AccessibilityColorSuggestion{}
	}
	if is.Fix.Headings == nil {
		is.Fix.Headings = []AccessibilityHeadingChange{}
	}
	if is.Fix.Links == nil {
		is.Fix.Links = []AccessibilityLinkTarget{}
	}
	if is.Element.Count == 0 {
		is.Element.Count = 1
	}
	is.ID = key
	ver.Lang = c.lang
	c.order++
	c.vc.findings = append(c.vc.findings, finding{key: key, issue: is, ver: ver, order: c.order})
}

// pathOf reads a path key; nil for elements outside MJML.
func pathOf(k string) []int {
	p, ok := parsePathKey(k)
	if !ok {
		return nil
	}
	return p
}

// sectionLabel is the label of the element's section, when the section
// carries a ReputeCast marker ("hero", "footer").
func (c *checker) sectionLabel(p []int) string {
	sec, label := c.d.section(p)
	if sec == nil || sec.Marker == "" {
		return ""
	}
	return label
}

// location is the element's path as the panel shows it.
func (c *checker) location(p []int, inner string) []string {
	out := []string{"mjml", "mj-body"}
	if c.d == nil || len(p) == 0 {
		if inner != "" {
			out = append(out, inner)
		}
		return out
	}
	sec, _ := c.d.section(p)
	if sec == nil {
		return out
	}
	part := sec.Tag
	if l := c.sectionLabel(p); l != "" {
		part += " · " + l
	}
	out = append(out, part)
	if len(p) > 1 {
		if n := c.d.node(p); n != nil && n.Tag != "mj-column" && n.Tag != "mj-group" && inner == "" {
			out = append(out, n.Tag)
		}
	}
	if inner != "" {
		out = append(out, inner)
	}
	return out
}

func (c *checker) nodeAt(k string) *mjml.Node {
	if c.d == nil {
		return nil
	}
	return c.d.nodes[k]
}

// imageArgs say which image an issue names (name.image, alt.title…): the
// hero image, the logo, an image in a section, or an image.
func (c *checker) imageArgs(p []int) i18n.Args {
	label := c.sectionLabel(p)
	if label == "" && c.d != nil && len(p) > 0 {
		_, label = c.d.section(p)
	}
	switch label {
	case "hero":
		return i18n.Args{"image": "hero"}
	case "header":
		return i18n.Args{"image": "logo"}
	case "", "image", "text":
		return i18n.Args{"image": "other"}
	}
	return i18n.Args{"image": "section", "section": label}
}

// named sets what an issue is about from its message.
func named(is AccessibilityIssue, name i18n.Text) AccessibilityIssue {
	is.Texts.Name, is.Name = name, titleIn(i18n.Default, name)
	return is
}

func (c *checker) images() {
	for _, im := range c.w.images {
		c.vc.counts.Images++
		n := c.nodeAt(im.path)
		missing := !im.hasAlt
		if n != nil && n.Tag == "mj-image" {
			_, has := n.Get("alt")
			missing = !has
		}
		if n != nil && n.Tag == "mj-social-element" {
			alt, has := n.Get("alt")
			missing = !has || strings.TrimSpace(alt) == ""
			if strings.TrimSpace(stripTags(n.Inner)) != "" {
				missing = false // its text link names it
			}
		}
		if im.link != nil {
			// A linked image's alt text is its link's text: the link check
			// judges it, so it is one issue, not two.
			if im.link.name() != "" || socialPair(c.w, im.link) {
				c.vc.counts.ImagesOK++
			}
			continue
		}
		if !missing {
			c.vc.counts.ImagesOK++
			continue
		}
		p := pathOf(im.path)
		image := c.imageArgs(p)
		file := path.Base(strings.SplitN(im.src, "?", 2)[0])
		is := worded(AccessibilityIssue{
			Kind: KindAltText, Severity: Serious,
			Path: p, Location: c.location(p, ""), Element: AccessibilityElement{Tag: tagOr(n, "img"), Role: "image", Label: file, Section: c.sectionLabel(p), Index: im.index, Src: im.src},
			Fix: AccessibilityFixSpec{Kind: KindAltText, PerLanguage: true, Decorative: true},
		}, mAltTitle.Keep(image), mAltTag.Keep(nil), mAltDetail.Keep(nil))
		c.add(KindAltText+":"+im.path+":"+fmt.Sprint(im.index), named(is, mNameImage.Keep(image)), AccessibilityIssueVersion{})
	}
}

func tagOr(n *mjml.Node, def string) string {
	if n != nil {
		return n.Tag
	}
	return def
}

// socialPair reports an icon link next to a text link to the same address
// in one social element: the text names both.
func socialPair(w *walker, l *linkInfo) bool {
	for _, o := range w.links {
		if o != l && o.path == l.path && o.href == l.href && strings.TrimSpace(collapse(o.text.String())) != "" {
			return true
		}
	}
	return false
}

func (c *checker) links() {
	groups := map[string]*linkGroup{}
	var order []string
	for _, l := range c.w.links {
		c.vc.counts.Links++
		n := c.nodeAt(l.path)
		p := pathOf(l.path)
		name := l.name()
		inner := ""
		role := "link"
		tag := "a"
		switch {
		case n != nil && n.Tag == "mj-button":
			role, tag = "button", "mj-button"
			// gomjml labels an empty button "Button"; MJML leaves it empty.
			if strings.TrimSpace(html.UnescapeString(stripTags(n.Inner))) == "" && !strings.Contains(strings.ToLower(n.Inner), "<img") {
				name = ""
			}
		case n != nil && (n.Tag == "mj-image" || n.Tag == "mj-social-element" || n.Tag == "mj-navbar-link"):
			tag = n.Tag
		case n != nil:
			inner = "a"
		}
		if name == "" {
			if socialPair(c.w, l) {
				c.vc.counts.LinksOK++
				continue
			}
			key := KindLinkNoText + ":" + l.path + ":" + fmt.Sprint(l.index)
			if len(l.imgs) > 0 {
				im := l.imgs[0]
				image := c.imageArgs(p)
				is := worded(AccessibilityIssue{Kind: KindLinkNoText, Severity: Serious,
					Path: p, Location: c.location(p, inner), Element: AccessibilityElement{Tag: tag, Role: "linked_image", Label: path.Base(strings.SplitN(im.src, "?", 2)[0]),
						Section: c.sectionLabel(p), Index: l.index, Href: l.href, Src: im.src},
					Fix: AccessibilityFixSpec{Kind: KindAltText, PerLanguage: true}}, mLinkedAltTitle.Keep(image), mNoTextLinkTag.Keep(nil), mLinkedAltDetail.Keep(nil))
				c.add(key, named(is, mNameLinkedImage.Keep(image)), AccessibilityIssueVersion{})
				continue
			}
			what := i18n.Args{"what": role}
			if s := c.sectionLabel(p); s != "" {
				what = i18n.Args{"what": role + "In", "section": s}
			}
			tagText := mNoTextLinkTag
			if role == "button" {
				tagText = mNoTextButtonTag
			}
			is := worded(AccessibilityIssue{Kind: KindLinkNoText, Severity: Serious,
				Path: p, Location: c.location(p, inner), Element: AccessibilityElement{Tag: tag, Role: role, Section: c.sectionLabel(p), Index: l.index, Href: l.href},
				Fix: AccessibilityFixSpec{Kind: KindLinkText, PerLanguage: true}}, mNoTextTitle.Keep(what), tagText.Keep(nil), mNoTextDetail.Keep(nil))
			c.add(key, named(is, mNameNoText.Keep(what)), AccessibilityIssueVersion{})
			continue
		}
		vague, bare := SaysNothing(name, c.lang), BareURL(name, l.href)
		if !vague && !bare {
			c.vc.counts.LinksOK++
			continue
		}
		// Links with the same words are one issue: "two “Read more” links".
		gk := "vague:" + normalizeText(name)
		if bare {
			gk = "bare:" + strings.ToLower(name)
		}
		g := groups[gk]
		if g == nil {
			g = &linkGroup{bare: bare && !vague, name: name, inner: inner}
			groups[gk] = g
			order = append(order, gk)
		}
		g.links = append(g.links, l)
		g.els = append(g.els, AccessibilityElement{Tag: tag, Role: role, Label: name, Section: c.sectionLabel(p), Index: l.index, Href: l.href})
	}
	for _, gk := range order {
		c.linkTextIssue(groups[gk])
	}
}

// linkGroup is the links of a version that share words that say nothing.
type linkGroup struct {
	bare  bool
	name  string
	inner string
	links []*linkInfo
	els   []AccessibilityElement
}

func (c *checker) linkTextIssue(g *linkGroup) {
	first := g.links[0]
	p := pathOf(first.path)
	el := g.els[0]
	el.Count = len(g.links)
	targets := []AccessibilityLinkTarget{}
	paths := [][]int{}
	var keys []string
	sameHref := true
	for _, l := range g.links {
		lp := nonNilPath(pathOf(l.path))
		targets = append(targets, AccessibilityLinkTarget{Path: lp, Index: l.index, Href: l.href})
		paths = append(paths, lp)
		keys = append(keys, l.path+"."+fmt.Sprint(l.index))
		if l.href != first.href {
			sameHref = false
		}
	}
	ver := AccessibilityIssueVersion{Text: g.name}
	if t, ok := c.opts.Titles[first.href]; ok && sameHref && t.Title != "" && (t.Lang == "" || BaseLanguage(t.Lang) == BaseLanguage(c.lang)) {
		ver.Suggestion = t.Title
	}
	is := AccessibilityIssue{Kind: KindLinkText, Severity: Moderate, Path: p, Paths: paths, Location: c.location(p, g.inner), Element: el,
		Fix: AccessibilityFixSpec{Kind: KindLinkText, PerLanguage: true, Links: targets}}
	words := i18n.Args{"count": len(g.links), "text": g.name}
	if g.bare {
		is = worded(is, mBareTitle.Keep(words), mBareTag.Keep(nil))
	} else {
		is = worded(is, mVagueTitle.Keep(words), mVagueTag.Keep(i18n.Args{"text": g.name}))
	}
	c.add(KindLinkText+":"+strings.Join(keys, ","), is, ver)
}

type contrastGroup struct {
	path   string
	fg, bg Color
	need   float64
	large  bool
	ratio  float64
	text   []string
	button bool
}

func (c *checker) contrast() {
	groups := map[string]*contrastGroup{}
	var order []string
	overImage := map[string]bool{}
	// Backgrounds of buttons, for "your other buttons".
	buttonBG := map[string]map[string]Color{}
	for _, run := range c.w.runs {
		n := c.nodeAt(run.path)
		isButton := n != nil && n.Tag == "mj-button"
		if isButton && !run.overImage {
			if buttonBG[run.path] == nil {
				buttonBG[run.path] = map[string]Color{}
			}
			buttonBG[run.path][run.bg.Hex()] = run.bg
		}
		if run.overImage {
			overImage[run.path] = true
			continue
		}
		ratio := Ratio(run.fg, run.bg)
		large := Large(run.size, run.bold)
		need := Needs(large)
		if passes(ratio, need) {
			continue
		}
		k := run.path + "|" + run.fg.Hex() + "|" + run.bg.Hex()
		g := groups[k]
		if g == nil {
			g = &contrastGroup{path: run.path, fg: run.fg, bg: run.bg, ratio: ratio, large: true, button: isButton}
			groups[k] = g
			order = append(order, k)
		}
		if need > g.need {
			g.need = need
		}
		if !large {
			g.large = false
		}
		g.text = append(g.text, run.text)
	}
	c.vc.counts.OverImage = len(overImage)
	for _, k := range order {
		g := groups[k]
		p := pathOf(g.path)
		n := c.nodeAt(g.path)
		label := truncate(collapse(strings.Join(g.text, " ")), 40)
		spec := AccessibilityFixSpec{Kind: "color", OneClick: true, Ratio: RoundDown(g.ratio), Needs: g.need, Large: g.large}
		var from, against Color
		role := "text"
		if g.button {
			spec.Target, from, against, role = "background", g.bg, g.fg, "button"
		} else {
			spec.Target, from, against = "text", g.fg, g.bg
		}
		spec.From, spec.Against = from.Hex(), against.Hex()
		spec.Suggestions = c.suggest(g, from, against, buttonBG)
		// A button names its text's color, other text its background's
		// ("white text on #5EBFB0", "#9CA3AF text on white").
		colors := i18n.Args{"label": label, "fg": g.fg.Hex(), "bg": g.bg.Hex(), "fgName": "", "bgName": "", "ratio": g.ratio, "need": g.need}
		if g.button {
			colors["fgName"] = colorWord(g.fg)
		} else {
			colors["bgName"] = colorWord(g.bg)
		}
		is := worded(AccessibilityIssue{Kind: KindContrast, Severity: Serious, Path: p, Location: c.location(p, ""),
			Element: AccessibilityElement{Tag: tagOr(n, "span"), Role: role, Label: label, Section: c.sectionLabel(p)}, Fix: spec},
			mContrastTitle.Keep(colors), mContrastTag.Keep(i18n.Args{"ratio": g.ratio}), mContrastDetail.Keep(nil))
		c.add(KindContrast+":"+g.path+":"+g.fg.Hex()+"/"+g.bg.Hex(), is, AccessibilityIssueVersion{})
	}
}

// colorWord is ColorName's word for white and black, for contrast.title;
// "" for the other colors, which read as their hex value.
func colorWord(c Color) string {
	if n := ColorName(c); n != c.Hex() {
		return n
	}
	return ""
}

// suggest offers the colors that fix a contrast issue: the email's other
// buttons' color, the nearest shade that passes AA and one that passes AAA.
func (c *checker) suggest(g *contrastGroup, from, against Color, buttonBG map[string]map[string]Color) []AccessibilityColorSuggestion {
	out := []AccessibilityColorSuggestion{}
	seen := map[string]bool{}
	addS := func(col Color, reason string) {
		if seen[col.Hex()] {
			return
		}
		seen[col.Hex()] = true
		out = append(out, AccessibilityColorSuggestion{Color: col.Hex(), Ratio: RoundDown(Ratio(col, against)), Reason: reason})
	}
	if g.button {
		freq := map[string]int{}
		cols := map[string]Color{}
		for p, bgs := range buttonBG {
			if p == g.path {
				continue
			}
			for h, col := range bgs {
				if passes(Ratio(col, against), g.need) {
					freq[h]++
					cols[h] = col
				}
			}
		}
		best := ""
		for h, n := range freq {
			if best == "" || n > freq[best] || (n == freq[best] && h < best) {
				best = h
			}
		}
		if best != "" {
			addS(cols[best], "other_buttons")
		}
	}
	if col, _, ok := Nearest(from, against, g.need); ok {
		addS(col, "nearest_aa")
	}
	if col, _, ok := Nearest(from, against, NeedsAAA(g.large)); ok {
		addS(col, "aaa")
	}
	return out
}

func (c *checker) sizes() {
	min := float64(c.opts.MinTextPx)
	smallest := map[string]float64{}
	var order []string
	for _, run := range c.w.runs {
		if run.size >= min-0.01 {
			continue
		}
		if _, ok := smallest[run.path]; !ok {
			order = append(order, run.path)
			smallest[run.path] = run.size
		}
		if run.size < smallest[run.path] {
			smallest[run.path] = run.size
		}
	}
	for _, k := range order {
		p := pathOf(k)
		n := c.nodeAt(k)
		size := roundPx(smallest[k])
		label := c.sectionLabel(p)
		args := i18n.Args{"px": size, "min": c.opts.MinTextPx}
		title, subject := mSizeTitle, mSizeSubject
		if label != "" {
			args["section"] = label
			title, subject = mSizeTitleIn, mSizeSubjectIn
		}
		is := worded(AccessibilityIssue{Kind: KindTextSize, Severity: Moderate,
			Path: p, Location: c.location(p, ""), Element: AccessibilityElement{Tag: tagOr(n, "span"), Role: "text", Section: label},
			Fix: AccessibilityFixSpec{Kind: KindTextSize, OneClick: true, FromPx: size, ToPx: c.opts.MinTextPx}}, title.Keep(args), mSizeTag.Keep(i18n.Args{"px": size}))
		is.Texts.Subject = subject.Keep(args)
		is.Fix.Subject = titleIn(i18n.Default, is.Texts.Subject)
		c.add(KindTextSize+":"+k, is, AccessibilityIssueVersion{})
	}
}

// headings finds skipped levels. Fixing a skip moves the headings under it
// up as well, so a whole run of card titles is one issue with one fix.
func (c *checker) headings() {
	type flagged struct {
		h        *headingInfo
		from, to int
		prev     int
	}
	var runs [][]flagged
	prevFixed, shiftFrom, shiftBy := 0, 0, 0
	for _, h := range c.w.headings {
		eff := h.level
		if shiftBy > 0 && h.level >= shiftFrom {
			eff = h.level - shiftBy
		} else {
			shiftBy = 0
		}
		before := prevFixed
		switch {
		case prevFixed > 0 && eff > prevFixed+1:
			to := prevFixed + 1
			shiftFrom, shiftBy = h.level, h.level-to
			eff = to
			runs = append(runs, []flagged{{h, h.level, to, before}})
		case eff != h.level:
			f := flagged{h, h.level, eff, before}
			last := len(runs) - 1
			if last >= 0 && runs[last][0].from == f.from && runs[last][0].to == f.to {
				runs[last] = append(runs[last], f)
			} else {
				runs = append(runs, []flagged{f})
			}
		}
		prevFixed = eff
	}
	for _, run := range runs {
		first := run[0]
		changes := []AccessibilityHeadingChange{}
		paths := [][]int{}
		cards := true
		texts := []string{}
		var keyParts []string
		for _, f := range run {
			p := pathOf(f.h.path)
			changes = append(changes, AccessibilityHeadingChange{Path: nonNilPath(p), Index: f.h.index, Text: collapse(f.h.text.String()), From: f.from, To: f.to})
			paths = append(paths, nonNilPath(p))
			texts = append(texts, collapse(f.h.text.String()))
			if !strings.Contains(c.sectionLabel(p)+c.layoutLabel(p), "card") {
				cards = false
			}
			keyParts = append(keyParts, f.h.path+"."+fmt.Sprint(f.h.index))
		}
		n := len(run)
		cardsArg := "no"
		if cards {
			cardsArg = "yes"
		}
		levels := i18n.Args{"count": n, "text": texts[0], "from": first.from, "prev": first.prev, "to": first.to, "cards": cardsArg}
		subject := mHeadingSubject.Keep(i18n.Args{"cards": cardsArg})
		if n == 1 {
			subject = mHeadingSubject1.Keep(i18n.Args{"text": texts[0]})
		}
		p := pathOf(first.h.path)
		inner := fmt.Sprintf("h%d", first.from)
		if n > 1 {
			inner += fmt.Sprintf(" × %d", n)
		}
		is := worded(AccessibilityIssue{Kind: KindHeading, Severity: Moderate, Path: p, Paths: paths, Location: c.location(p, inner),
			Element: AccessibilityElement{Tag: fmt.Sprintf("h%d", first.from), Role: "heading", Label: texts[0], Section: c.sectionLabel(p), Index: first.h.index, Count: n},
			Fix:     AccessibilityFixSpec{Kind: KindHeading, OneClick: true, Headings: changes}},
			mHeadingTitle.Keep(levels), mHeadingTag.Keep(i18n.Args{"from": first.from, "prev": first.prev}), mHeadingDetail.Keep(i18n.Args{"count": n, "to": first.to}))
		is.Texts.Subject = subject
		is.Fix.Subject = titleIn(i18n.Default, subject)
		c.add(KindHeading+":"+strings.Join(keyParts, ","), is, AccessibilityIssueVersion{})
	}
}

// layoutLabel is the section's look-based label ("2 columns"), for
// recognizing card rows without a marker.
func (c *checker) layoutLabel(p []int) string {
	if c.d == nil {
		return ""
	}
	sec, label := c.d.section(p)
	if sec == nil {
		return ""
	}
	if strings.Contains(sec.Marker, "card") {
		return "cards"
	}
	return label
}

func nonNilPath(p []int) []int {
	if p == nil {
		return []int{}
	}
	return p
}

// merge lists each issue once, with every language it's in.
func merge(checks []versionCheck, langs []AccessibilityLanguage) []AccessibilityIssue {
	byKey := map[string]*AccessibilityIssue{}
	firstOrder := map[string]int{}
	var keys []string
	for vi, vc := range checks {
		for _, f := range vc.findings {
			is, ok := byKey[f.key]
			if !ok {
				cp := f.issue
				cp.Languages = []string{}
				cp.Versions = []AccessibilityIssueVersion{}
				is = &cp
				byKey[f.key] = is
				keys = append(keys, f.key)
				firstOrder[f.key] = vi*100000 + f.order
			}
			if !contains(is.Languages, vc.lang) {
				is.Languages = append(is.Languages, vc.lang)
				is.Versions = append(is.Versions, f.ver)
			}
		}
	}
	out := make([]AccessibilityIssue, 0, len(keys))
	for _, k := range keys {
		is := byKey[k]
		if is.Kind == KindLinkText {
			is.Texts.Detail = linkTextDetail(*is, langs)
			is.Detail, _ = sentencesIn(i18n.Default, is.Texts.Detail)
		}
		if is.Kind == KindContrast && len(langs) < 2 {
			is.Detail, is.Texts.Detail = "", nil // one version: nothing is shared
		}
		is.Texts.Short = shortText(*is)
		is.Short = ShortName(*is)
		out = append(out, *is)
	}
	rank := map[string]int{KindAltText: 0, KindLinkNoText: 1, KindContrast: 2, KindLinkText: 3, KindHeading: 4, KindTextSize: 5}
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := out[i].Severity == Serious, out[j].Severity == Serious
		if si != sj {
			return si
		}
		if rank[out[i].Kind] != rank[out[j].Kind] {
			return rank[out[i].Kind] < rank[out[j].Kind]
		}
		return firstOrder[out[i].ID] < firstOrder[out[j].ID]
	})
	return out
}

// linkTextDetail names the link's text in the other languages and who gets
// the page's title as a suggestion: one message a sentence.
func linkTextDetail(is AccessibilityIssue, langs []AccessibilityLanguage) []i18n.Text {
	var others []string
	var suggested, field []string
	src := ""
	for i, v := range is.Versions {
		if i == 0 {
			src = v.Text
		} else if v.Text != "" && v.Text != src {
			others = append(others, quote(v.Text))
		}
		if v.Suggestion != "" {
			suggested = append(suggested, v.Lang)
		} else {
			field = append(field, v.Lang)
		}
	}
	var out []i18n.Text
	if len(others) > 0 {
		out = append(out, mLinkTextAlso.Keep(i18n.Args{"textList": others}))
	}
	switch {
	case len(suggested) > 0 && len(field) > 0:
		out = append(out, mLinkTextSplit.Keep(i18n.Args{"suggestedLangs": suggested, "fieldLangs": field, "count": len(field)}))
	case len(suggested) == 0 && len(langs) > 0:
		out = append(out, mLinkTextWrite.Keep(nil))
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return b.String()
}
