package publicpages

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"html/template"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/forms"
	pagesvc "github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// The blocks that show live data, drawn on every request: a page is
// stored as the HTML of its blocks (internal/pages), and its custom
// elements (<rc-product>, <rc-form>, <rc-countdown>, <rc-event>,
// <rc-products>, <rc-video>, <rc-posts>, <rc-tiers>) become the public
// markup here, with the product's current price, the form's fields and the
// time left. A countdown that ended reveals or hides the blocks it names.
// Nothing here runs a script: the page works without JavaScript, and
// pages.js only adds the reader's own time zone and loads a video on click.

// esc escapes text for HTML.
func esc(s string) string { return template.HTMLEscapeString(s) }

func hashString(b []byte) string {
	f := fnv.New64a()
	_, _ = f.Write(b)
	return strconv.FormatUint(f.Sum64(), 36)
}

// pageRender draws a page's blocks for one request.
type pageRender struct {
	h   *Handler
	ctx context.Context
	s   *site
	v   *siteView
	// The page: its id, address and where its Buy buttons say they are.
	pageID  uuid.UUID
	pageURL string
	source  string
	// preview: the builder or Preview; drafts show, nothing is live.
	preview  bool
	now      time.Time
	merchant products.Merchant
	products map[uuid.UUID]products.PublicProduct
	// failed is a signup that came back with errors: its block shows them.
	failed *formFailure
	// What the drawing found: video origins for the CSP, the phone's Buy
	// bar, the first event.
	frames []string
	buyBar *buyBarView
	event  *eventView
}

// formFailure is a signup of a block that didn't go through.
type formFailure struct {
	block  string
	values forms.Submission
	out    forms.Outcome
}

// buyBarView is the phone's Buy bar.
type buyBarView struct {
	Name, Sub, Button, Href string
}

// eventView is a page's event, for the registered page and the calendar.
type eventView struct {
	Title, Where, Link, Reminders string
	Start, End                    time.Time
	TZ                            *time.Location
}

// render draws a page's blocks.
func (pr *pageRender) render(src string) (template.HTML, error) {
	nodes, err := html.ParseFragment(strings.NewReader(src), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return "", err
	}
	if pr.merchant == (products.Merchant{}) && pr.h.Products != nil {
		pr.merchant, _ = pr.h.Products.Merchant(pr.ctx, pr.s.ws)
	}
	if err := pr.loadProducts(nodes); err != nil {
		return "", err
	}
	// Countdowns first: what they reveal and hide.
	hidden := map[string]bool{}
	walkNodes(nodes, func(n *html.Node) {
		if n.Data != pagesvc.ElCountdown {
			return
		}
		at, ok := pagesvc.Local(attrOf(n, "at"), attrOf(n, "tz"))
		if !ok {
			return
		}
		ended := !pr.now.Before(at)
		if attrOf(n, "end") != "reveal" {
			return
		}
		// Before the end the revealed blocks wait; after it the hidden ones go.
		ids := attrOf(n, "reveal")
		if ended {
			ids = attrOf(n, "hide")
		}
		for _, id := range pagesvc.BlockRefs(ids) {
			hidden[id] = true
		}
	})
	delete(hidden, "")
	nodes = removeBlocks(nodes, hidden)
	token := make([]byte, 8)
	_, _ = rand.Read(token)
	mark := "rcblk" + hex.EncodeToString(token)
	parts := map[string]template.HTML{}
	i := 0
	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			if c.Type == html.ElementNode && strings.HasPrefix(c.Data, "rc-") {
				out, err := pr.block(c)
				if err != nil {
					pr.h.log().WarnContext(pr.ctx, "page block", "block", c.Data, "err", err)
				}
				key := fmt.Sprintf("%s-%d-", mark, i)
				i++
				parts[key] = out
				n.InsertBefore(&html.Node{Type: html.TextNode, Data: key}, c)
				n.RemoveChild(c)
			} else {
				visit(c)
			}
			c = next
		}
	}
	root := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	visit(root)
	var buf bytes.Buffer
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&buf, c); err != nil {
			return "", err
		}
	}
	out := buf.String()
	for k, v := range parts {
		out = strings.Replace(out, k, string(v), 1)
	}
	return template.HTML(pr.h.mediaHTML(out)), nil //nolint:gosec // the page's sanitized blocks and our own escaped markup
}

func walkNodes(nodes []*html.Node, f func(*html.Node)) {
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			f(n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	for _, n := range nodes {
		visit(n)
	}
}

// removeBlocks drops the blocks whose data-pb-id is in ids.
func removeBlocks(nodes []*html.Node, ids map[string]bool) []*html.Node {
	if len(ids) == 0 {
		return nodes
	}
	out := nodes[:0]
	for _, n := range nodes {
		if n.Type == html.ElementNode && ids[attrOf(n, "data-pb-id")] {
			continue
		}
		var prune func(p *html.Node)
		prune = func(p *html.Node) {
			for c := p.FirstChild; c != nil; {
				next := c.NextSibling
				if c.Type == html.ElementNode && ids[attrOf(c, "data-pb-id")] {
					p.RemoveChild(c)
				} else {
					prune(c)
				}
				c = next
			}
		}
		prune(n)
		out = append(out, n)
	}
	return out
}

func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// loadProducts reads the products the page's blocks show.
func (pr *pageRender) loadProducts(nodes []*html.Node) error {
	if pr.products == nil {
		pr.products = map[uuid.UUID]products.PublicProduct{}
	}
	var ids []uuid.UUID
	walkNodes(nodes, func(n *html.Node) {
		switch n.Data {
		case pagesvc.ElProduct:
			if id, err := uuid.Parse(attrOf(n, "product")); err == nil {
				ids = append(ids, id)
			}
		case pagesvc.ElProducts:
			for _, p := range strings.Split(attrOf(n, "products"), ",") {
				if id, err := uuid.Parse(strings.TrimSpace(p)); err == nil {
					ids = append(ids, id)
				}
			}
		}
	})
	if len(ids) == 0 || pr.h.Products == nil {
		return nil
	}
	m, err := pr.h.Products.PublicProducts(pr.ctx, pr.s.ws, ids, pr.preview)
	if err != nil {
		return err
	}
	for k, v := range m {
		pr.products[k] = v
	}
	return nil
}

// block draws one live block.
func (pr *pageRender) block(n *html.Node) (template.HTML, error) {
	switch n.Data {
	case pagesvc.ElProduct:
		return pr.product(n), nil
	case pagesvc.ElProducts:
		return pr.productGrid(n)
	case pagesvc.ElForm:
		return pr.form(n)
	case pagesvc.ElCountdown:
		return pr.countdown(n), nil
	case pagesvc.ElEvent:
		return pr.eventBlock(n), nil
	case pagesvc.ElVideo:
		return pr.video(n), nil
	case pagesvc.ElPosts:
		return pr.posts(n)
	case pagesvc.ElTiers:
		return pr.tiers(n)
	}
	// Store products come with the stores (phase 7, X20).
	if pr.preview {
		return template.HTML(`<p class="pg-empty">` + esc(pr.v.S("store.empty")) + `</p>`), nil //nolint:gosec // escaped
	}
	return "", nil
}

// money writes a price ("$29", "$15.50").
func money(cents int64, currency string) string { return pagesvc.Money(cents, currency) }

// priceText is a product's price for a button ("$29", "$5+").
func priceText(p products.PublicProduct) string {
	if p.Live.PriceKind == products.PricePWYW {
		return money(p.Live.MinCents, p.Currency) + "+"
	}
	return money(p.Live.PriceCents, p.Currency)
}

// buttonText fills a block's button text: {price} and {product}.
func buttonText(tpl string, p products.PublicProduct, def string) string {
	t := strings.TrimSpace(tpl)
	if t == "" {
		t = def
	}
	t = strings.ReplaceAll(t, "{price}", priceText(p))
	return strings.ReplaceAll(t, "{product}", products.ShortName(p.Live.Name))
}

// fileLine writes a product's files with the pages' words ("PDF, 86
// pages" or "PDF · 86 pages · 4.2 MB").
func (pr *pageRender) fileLine(f products.ProductFilesSummary) string {
	if f.Count == 0 {
		return ""
	}
	if f.Pages > 0 && f.Count == 1 {
		return pr.v.S("dl.filePages", "kind", f.Kind, "count", f.Pages, "size", products.HumanSize(f.Bytes))
	}
	return pr.v.S("dl.file", "kind", f.Kind, "size", products.HumanSize(f.Bytes))
}

// buyAction is where a Buy button posts.
func (pr *pageRender) buyAction(p products.PublicProduct) string {
	return pr.s.path("buy", p.ID.String())
}

// product draws a product block: the price, the Buy button, the list
// offer and the checkout note; the "final" layout adds the cover and the
// name beside it.
func (pr *pageRender) product(n *html.Node) template.HTML {
	v := pr.v
	id, err := uuid.Parse(attrOf(n, "product"))
	if err != nil {
		if pr.preview {
			return template.HTML(`<p class="pg-empty">` + esc(v.S("buy.pick")) + `</p>`) //nolint:gosec // escaped
		}
		return ""
	}
	p, ok := pr.products[id]
	if !ok {
		if pr.preview {
			return template.HTML(`<p class="pg-empty">` + esc(v.S("buy.archived")) + `</p>`) //nolint:gosec // escaped
		}
		return ""
	}
	if !p.Published && !pr.preview {
		return ""
	}
	box := pr.buyBox(n, p)
	if attrOf(n, "layout") != "final" {
		return box
	}
	var b strings.Builder
	b.WriteString(`<div class="finalcta"><div class="fc-l">`)
	b.WriteString(coverImg(p, "md"))
	b.WriteString(`<div class="fc-t">`)
	if e := attrOf(n, "eyebrow"); e != "" {
		b.WriteString(`<span class="eyebrow">` + esc(e) + `</span>`)
	}
	b.WriteString(`<h2 class="sp-h2">` + esc(p.Live.Name) + `</h2>`)
	b.WriteString(`<span class="fc-d">`)
	if f := products.FileTextLong(p.Files); f != "" {
		b.WriteString(esc(f) + `<br>`)
	}
	b.WriteString(esc(v.S("buy.downloadAfter")) + `</span></div></div>`)
	b.WriteString(string(box))
	b.WriteString(`</div>`)
	return template.HTML(b.String()) //nolint:gosec // escaped
}

// coverImg is a product's cover at a size class.
func coverImg(p products.PublicProduct, size string) string {
	if p.Live.CoverURL == "" {
		return `<span class="cov ` + size + ` blank" role="img" aria-label="` + esc(p.Live.Name) + `"></span>`
	}
	alt := p.Live.CoverAlt
	return `<img class="cov ` + size + `" src="` + esc(p.Live.CoverURL) + `" alt="` + esc(alt) + `" loading="lazy" decoding="async">`
}

// buyBox is the price, the Buy button, the offer and the note.
func (pr *pageRender) buyBox(n *html.Node, p products.PublicProduct) template.HTML {
	v := pr.v
	m := pr.merchant
	final := attrOf(n, "layout") == "final"
	elID := strings.TrimSpace(attrOf(n, "id"))
	if elID == "" {
		elID = strings.TrimSpace(attrOf(n, "data-pb-id"))
	}
	var b strings.Builder
	selling := m.Connected && p.Published
	if selling && !pr.preview {
		b.WriteString(`<form class="buybox" method="post" action="` + esc(pr.buyAction(p)) + `"`)
	} else {
		b.WriteString(`<div class="buybox"`)
	}
	if elID != "" {
		b.WriteString(` id="` + esc(elID) + `"`)
	}
	b.WriteString(`>`)
	if selling && !pr.preview {
		b.WriteString(`<input type="hidden" name="_src" value="` + esc(pr.source) + `">`)
		if pr.pageID != uuid.Nil {
			b.WriteString(`<input type="hidden" name="_page" value="` + pr.pageID.String() + `">`)
		}
	}
	showPrice := attrOf(n, "price") != "0"
	pwyw := p.Live.PriceKind == products.PricePWYW
	if showPrice {
		if pwyw {
			b.WriteString(`<div class="bb-price"><span class="amt pw">` + esc(v.S("buy.pwyw")) + `</span></div>`)
			if p.Live.SuggestedCents > 0 {
				b.WriteString(`<p class="bb-sub">` + esc(v.S("buy.pwywFromSuggested", "min", money(p.Live.MinCents, p.Currency), "suggested", money(p.Live.SuggestedCents, p.Currency))) + `</p>`)
			} else {
				b.WriteString(`<p class="bb-sub">` + esc(v.S("buy.pwywFrom", "min", money(p.Live.MinCents, p.Currency))) + `</p>`)
			}
		} else {
			b.WriteString(`<div class="bb-price"><span class="amt">` + esc(money(p.Live.PriceCents, p.Currency)) + `</span><span class="per">` + esc(v.S("buy.oneTime")) + `</span></div>`)
		}
	}
	label := buttonText(attrOf(n, "button"), p, v.S("buy.defaultButton", "price", priceText(p)))
	switch {
	case !m.Connected:
		b.WriteString(`<span class="st-btn lg full" aria-disabled="true">` + esc(v.S("buy.unavailable")) + `</span>`)
	case pr.preview:
		b.WriteString(`<span class="st-btn lg full">` + esc(label) + `</span>`)
	default:
		b.WriteString(`<button class="st-btn lg full" type="submit">` + esc(label) + `</button>`)
	}
	if attrOf(n, "offer") != "0" && p.Live.OfferOn && p.Live.OfferText != "" {
		note := strings.TrimSpace(p.OfferNote + " " + v.S("buy.offerNote"))
		b.WriteString(`<label class="offer"><input type="checkbox" name="offer" value="1"><span><span class="ot">` + esc(p.Live.OfferText) +
			`</span><span class="os">` + esc(note) + `</span></span></label>`)
	}
	merchant := m.Account
	if merchant == "" {
		merchant = pr.s.set.Branding.PublisherName
	}
	var note string
	switch {
	case final && m.Tax:
		note = v.S("buy.noteFinal", "merchant", merchant)
	case final:
		note = v.S("buy.noteFinalNoTax", "merchant", merchant)
	case pwyw && products.FileText(p.Files) != "" && m.Tax:
		note = v.S("buy.notePwyw", "merchant", merchant, "file", products.FileText(p.Files))
	case pwyw && products.FileText(p.Files) != "":
		note = v.S("buy.notePwywNoTax", "merchant", merchant, "file", products.FileText(p.Files))
	case m.Tax:
		note = v.S("buy.noteBox", "merchant", merchant)
	default:
		note = v.S("buy.noteBoxNoTax", "merchant", merchant)
	}
	if !m.Connected {
		if pr.preview {
			note = v.S("buy.unavailableNote")
		} else {
			note = ""
		}
	}
	if note != "" {
		b.WriteString(`<p class="bb-note">` + esc(note) + `</p>`)
	}
	if selling && !pr.preview {
		b.WriteString(`</form>`)
	} else {
		b.WriteString(`</div>`)
	}
	if pr.buyBar == nil && !final {
		href := "#buy"
		if elID != "" {
			href = "#" + elID
		}
		sub := money(p.Live.PriceCents, p.Currency)
		if pwyw {
			sub = v.S("shop.pwywFrom", "min", money(p.Live.MinCents, p.Currency))
		}
		if f := products.FileText(p.Files); f != "" {
			sub = v.S("buy.bar", "price", sub, "file", f)
		}
		pr.buyBar = &buyBarView{Name: p.Live.Name, Sub: sub, Button: label, Href: href}
	}
	return template.HTML(b.String()) //nolint:gosec // escaped
}

// productGrid draws the shop's cards for chosen products, or the shop's.
func (pr *pageRender) productGrid(n *html.Node) (template.HTML, error) {
	if attrOf(n, "source") == "store" {
		return "", nil
	}
	var list []products.PublicProduct
	ids := strings.Split(attrOf(n, "products"), ",")
	for _, raw := range ids {
		if id, err := uuid.Parse(strings.TrimSpace(raw)); err == nil {
			if p, ok := pr.products[id]; ok && (p.Published || pr.preview) {
				list = append(list, p)
			}
		}
	}
	if len(list) == 0 && pr.h.Products != nil {
		all, err := pr.h.Products.ShopProducts(pr.ctx, pr.s.ws)
		if err != nil {
			return "", err
		}
		list = all
	}
	if c, err := strconv.Atoi(attrOf(n, "count")); err == nil && c > 0 && len(list) > c {
		list = list[:c]
	}
	return pr.shopCards(list, attrOf(n, "button")), nil
}

// shopCards are the shop's product cards.
func (pr *pageRender) shopCards(list []products.PublicProduct, button string) template.HTML {
	v := pr.v
	if len(list) == 0 {
		return template.HTML(`<p class="pg-empty">` + esc(v.S("shop.empty")) + `</p>`) //nolint:gosec // escaped
	}
	var b strings.Builder
	b.WriteString(`<div class="shopg">`)
	for _, p := range list {
		href := pr.s.path("products", p.Live.Slug)
		if p.SalesPage != "" {
			href = pr.s.path(p.SalesPage)
		}
		label := strings.TrimSpace(button)
		if label == "" {
			label = v.S("shop.see", "name", seeName(p.Live.Name))
		}
		b.WriteString(`<article class="shopc"><div class="art">` + coverImg(p, "card") + `</div><div class="b">`)
		b.WriteString(`<h2><a href="` + esc(href) + `">` + esc(p.Live.Name) + `</a></h2>`)
		// The short description is the card's line ("Eight chapters on …
		// PDF, 86 pages."): it says what the files are.
		b.WriteString(`<p>` + esc(p.Live.ShortDesc) + `</p><div class="f">`)
		if p.Live.PriceKind == products.PricePWYW {
			b.WriteString(`<span class="pr">` + esc(v.S("shop.pwywFrom", "min", money(p.Live.MinCents, p.Currency))) + `</span>`)
		} else {
			b.WriteString(`<span class="pr">` + esc(money(p.Live.PriceCents, p.Currency)) + `</span>`)
		}
		b.WriteString(`<a class="st-btn full" href="` + esc(href) + `">` + esc(label) + `</a></div></div></article>`)
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String()) //nolint:gosec // escaped
}

// seeName is what a shop card's button names: "the guide", "the kit".
func seeName(name string) string {
	l := strings.ToLower(name)
	for _, w := range []string{"guide", "kit", "planner", "course", "book", "ebook", "template", "templates", "checklist", "workbook", "toolkit"} {
		if strings.Contains(l, w) {
			return "the " + w
		}
	}
	return "it"
}

// form draws a signup form block: the workspace's form, posted to the
// page itself so a failed signup comes back with its errors in place.
func (pr *pageRender) form(n *html.Node) (template.HTML, error) {
	v := pr.v
	id, err := uuid.Parse(attrOf(n, "form"))
	if err != nil {
		if pr.preview {
			return template.HTML(`<p class="pg-empty">` + esc(v.S("form.missing")) + `</p>`), nil //nolint:gosec // escaped
		}
		return "", nil
	}
	f, err := pr.h.Q.FormGet(pr.ctx, store.FormGetParams{WorkspaceID: pr.s.ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	c := forms.ConfigOf(f)
	if c.Status == forms.StatusPaused || c.Status == forms.StatusDraft {
		if pr.preview {
			return template.HTML(`<p class="pg-empty">` + esc(v.S("form.paused")) + `</p>`), nil //nolint:gosec // escaped
		}
		return "", nil
	}
	fl, err := pr.h.formLangIn(pr.ctx, pr.s, c, v.Lang)
	if err != nil {
		return "", err
	}
	block := attrOf(n, "data-pb-id")
	var values *forms.Submission
	out := forms.Outcome{}
	if pr.failed != nil && pr.failed.block == block {
		values, out = &pr.failed.values, pr.failed.out
	}
	body, err := pr.h.subscribeBody(pr.ctx, pr.s, fl, false, values, out)
	if err != nil {
		return "", err
	}
	heading := strings.TrimSpace(attrOf(n, "heading"))
	if heading == "" {
		heading = body.Heading
	}
	var b strings.Builder
	action := pr.pageURL + "/form"
	if pr.preview {
		b.WriteString(`<div class="fcard">`)
	} else {
		b.WriteString(`<form class="fcard" method="post" action="` + esc(action) + `" novalidate>`)
	}
	b.WriteString(`<h2>` + esc(heading) + `</h2>`)
	if body.Alert != "" {
		b.WriteString(`<p class="ferr" role="alert">` + esc(body.Alert) + `</p>`)
	}
	if !pr.preview {
		b.WriteString(`<input type="hidden" name="_t" value="` + esc(body.Token) + `"><input type="hidden" name="_lang" value="` + esc(v.Lang) + `">`)
		b.WriteString(`<input type="hidden" name="_form" value="` + id.String() + `"><input type="hidden" name="_block" value="` + esc(block) + `">`)
		b.WriteString(`<input type="hidden" name="_page" value="` + esc(pr.pageURL) + `">`)
	}
	for _, fv := range body.Fields {
		b.WriteString(`<label class="fl" for="` + esc(block+"-"+fv.ID) + `"><span>` + esc(fv.Label))
		if fv.Optional {
			b.WriteString(` <span class="op">` + esc(v.S("form.optional")) + `</span>`)
		}
		b.WriteString(`</span>`)
		typ := fv.Type
		if typ == "select" {
			typ = "text"
		}
		b.WriteString(`<input class="st-in" id="` + esc(block+"-"+fv.ID) + `" type="` + esc(typ) + `" name="` + esc(fv.Name) + `"`)
		if fv.Placeholder != "" {
			b.WriteString(` placeholder="` + esc(fv.Placeholder) + `"`)
		}
		if fv.Autocomplete != "" {
			b.WriteString(` autocomplete="` + esc(fv.Autocomplete) + `"`)
		}
		if fv.Value != "" {
			b.WriteString(` value="` + esc(fv.Value) + `"`)
		}
		if fv.Required {
			b.WriteString(` required`)
		}
		if fv.Error != "" {
			b.WriteString(` aria-invalid="true"`)
		}
		if pr.preview {
			b.WriteString(` tabindex="-1"`)
		}
		b.WriteString(`></label>`)
		if fv.Error != "" {
			b.WriteString(`<p class="ferr" role="alert">` + esc(fv.Error) + `</p>`)
		}
	}
	for _, l := range body.Lists {
		b.WriteString(`<label class="cons"><input type="checkbox" name="list" value="` + esc(l.ID) + `"`)
		if l.Checked {
			b.WriteString(` checked`)
		}
		b.WriteString(`><span><b>` + esc(l.Name) + `</b>`)
		if l.Text != "" {
			b.WriteString(` · ` + esc(l.Text))
		}
		b.WriteString(`</span></label>`)
	}
	if body.ListError != "" {
		b.WriteString(`<p class="ferr" role="alert">` + esc(body.ListError) + `</p>`)
	}
	if body.Consent.Text != "" {
		b.WriteString(`<label class="cons"><input type="checkbox" name="consent" value="1" required`)
		if body.ConsentChecked {
			b.WriteString(` checked`)
		}
		b.WriteString(`><span>` + esc(body.Consent.Before))
		if body.Consent.Href != "" {
			b.WriteString(`<a href="` + esc(body.Consent.Href) + `" class="st-lnk">` + esc(body.Consent.Link) + `</a>`)
		}
		b.WriteString(esc(body.Consent.After) + `</span></label>`)
		if body.ConsentError != "" {
			b.WriteString(`<p class="ferr" role="alert">` + esc(body.ConsentError) + `</p>`)
		}
	}
	if body.Honeypot && !pr.preview {
		b.WriteString(`<div class="hp" aria-hidden="true"><label>Website <input type="text" name="website" tabindex="-1" autocomplete="off"></label></div>`)
	}
	button := body.Button
	if pr.preview {
		b.WriteString(`<span class="st-btn lg full">` + esc(button) + `</span>`)
	} else {
		b.WriteString(`<button type="submit" class="st-btn lg full">` + esc(button) + `</button>`)
	}
	if hint := strings.TrimSpace(attrOf(n, "hint")); hint != "" {
		b.WriteString(`<p class="fhint">` + esc(hint) + `</p>`)
	}
	if pr.preview {
		b.WriteString(`</div>`)
	} else {
		b.WriteString(`</form>`)
	}
	return template.HTML(b.String()), nil //nolint:gosec // escaped
}

// countdown draws a countdown: the time left in tabular figures, read out
// as "Opens in 5 days, 3 hours". It's drawn on the server, so it works
// without scripts; the page shows the time left when it was loaded.
func (pr *pageRender) countdown(n *html.Node) template.HTML {
	v := pr.v
	at, ok := pagesvc.Local(attrOf(n, "at"), attrOf(n, "tz"))
	if !ok {
		return ""
	}
	left := at.Sub(pr.now)
	if left <= 0 {
		if attrOf(n, "end") == "message" && attrOf(n, "message") != "" {
			return template.HTML(`<p class="cd-s">` + esc(attrOf(n, "message")) + `</p>`) //nolint:gosec // escaped
		}
		return ""
	}
	units := strings.Split(attrOf(n, "units"), ",")
	has := map[string]bool{}
	for _, u := range units {
		has[strings.TrimSpace(u)] = true
	}
	total := int(left / time.Second)
	days, hours, minutes, seconds := total/86400, (total%86400)/3600, (total%3600)/60, total%60
	if !has["days"] {
		hours += days * 24
	}
	label := strings.TrimSpace(attrOf(n, "label"))
	var spoken string
	switch {
	case days > 0:
		spoken = v.S("cd.leftDays", "days", days, "hours", total%86400/3600)
	case hours > 0:
		spoken = v.S("cd.leftHours", "hours", hours, "minutes", minutes)
	default:
		spoken = v.S("cd.leftMinutes", "minutes", max(minutes, 1))
	}
	if label != "" {
		spoken = v.S("cd.left", "label", label, "left", spoken)
	}
	var b strings.Builder
	b.WriteString(`<div class="cdw" role="timer" aria-label="` + esc(spoken) + `"><div class="cd" aria-hidden="true">`)
	unit := func(n int, key string) {
		fmt.Fprintf(&b, `<span class="cdu"><b>%02d</b><span>%s</span></span>`, n, esc(v.S(key, "count", n)))
	}
	if has["days"] {
		unit(days, "cd.days")
	}
	if has["hours"] {
		unit(hours, "cd.hours")
	}
	if has["minutes"] {
		unit(minutes, "cd.minutes")
	}
	if has["seconds"] {
		unit(seconds, "cd.seconds")
	}
	b.WriteString(`</div><p class="cd-s">` + esc(spoken) + `</p></div>`)
	return template.HTML(b.String()) //nolint:gosec // escaped
}

// Icons of the event details.
const (
	icoCal   = `<svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="2.5" y="3.5" width="11" height="10" rx="1.5"></rect><path d="M2.5 6.5h11M5.5 2v3M10.5 2v3"></path></svg>`
	icoClock = `<svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="8" cy="8" r="5.8"></circle><path d="M8 4.8V8l2.2 1.4"></path></svg>`
	icoLink  = `<svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6.6 9.4a2.8 2.8 0 0 0 4 0l2-2a2.8 2.8 0 0 0-4-4l-.6.6M9.4 6.6a2.8 2.8 0 0 0-4 0l-2 2a2.8 2.8 0 0 0 4 4l.6-.6"></path></svg>`
	icoPin   = `<svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 14s4.5-4.2 4.5-7.5a4.5 4.5 0 0 0-9 0C3.5 9.8 8 14 8 14z"></path><circle cx="8" cy="6.5" r="1.6"></circle></svg>`
)

// eventOf reads an event block.
func eventOf(n *html.Node) (*eventView, bool) {
	tz := attrOf(n, "tz")
	start, ok := pagesvc.Local(attrOf(n, "date")+"T"+attrOf(n, "start"), tz)
	if !ok {
		return nil, false
	}
	end := start
	if e := attrOf(n, "end"); e != "" {
		if t, ok := pagesvc.Local(attrOf(n, "date")+"T"+e, tz); ok && t.After(start) {
			end = t
		}
	}
	loc, _ := time.LoadLocation(tz)
	ev := &eventView{Title: attrOf(n, "title"), Link: attrOf(n, "link"), Reminders: attrOf(n, "reminders"), Start: start, End: end, TZ: loc}
	if attrOf(n, "where") == "venue" {
		ev.Where = attrOf(n, "venue")
	}
	return ev, true
}

// timeRange writes "17:00–17:30 UTC" in a time zone.
func timeRange(start, end time.Time, loc *time.Location) string {
	s, e := start.In(loc), end.In(loc)
	zone := s.Format("MST")
	if loc == time.UTC || zone == "UTC" {
		zone = "UTC"
	}
	if end.Equal(start) {
		return s.Format("15:04") + " " + zone
	}
	return s.Format("15:04") + "–" + e.Format("15:04") + " " + zone
}

// longDate writes "29 October 2026" in the page's language.
func (v *siteView) longDate(t time.Time) string {
	if v.labels.Code() == "en" {
		return t.Format("2 January 2006")
	}
	return v.labels.Lang().Region.LongDate(t)
}

// shortDate writes "22 Oct"; fullDate "15 Oct 2026".
func (v *siteView) shortDate(t time.Time) string { return v.labels.DayMonth(t) }

func (v *siteView) fullDate(t time.Time) string {
	if v.labels.Code() == "en" {
		return t.Format("2 Jan 2006")
	}
	return v.labels.Lang().Region.Date(t)
}

// eventBlock draws the event details: the date and the time in the
// event's zone, the reader's own time (pages.js fills it in), where, and
// how the joining link comes. The joining link itself is never on the
// page: it's sent after registering.
func (pr *pageRender) eventBlock(n *html.Node) template.HTML {
	v := pr.v
	ev, ok := eventOf(n)
	if !ok {
		return ""
	}
	if pr.event == nil {
		pr.event = ev
	}
	start := ev.Start.In(ev.TZ)
	var b strings.Builder
	b.WriteString(`<div class="evd">`)
	where := v.S("event.onlinePlain")
	if p := attrOf(n, "platform"); p != "" && attrOf(n, "where") != "venue" {
		where = v.S("event.online", "platform", p)
	}
	icon := icoCal
	if ev.Where != "" {
		where = ev.Where
	}
	b.WriteString(`<div class="evd-r"><span class="ei">` + icon + `</span><span class="et"><b>` + esc(v.longDate(start)) + `</b><span>` + esc(where) + `</span></span></div>`)
	b.WriteString(`<div class="evd-r"><span class="ei">` + icoClock + `</span><span class="et"><b>` + esc(timeRange(ev.Start, ev.End, ev.TZ)) + `</b>`)
	if attrOf(n, "local") != "0" {
		tpl := v.S("event.yourTime", "time", "{time}", "zone", "{zone}")
		b.WriteString(`<span class="lt" hidden data-start="` + ev.Start.UTC().Format(time.RFC3339) + `" data-end="` + ev.End.UTC().Format(time.RFC3339) +
			`" data-tpl="` + esc(tpl) + `"></span>`)
	}
	b.WriteString(`</span></div>`)
	if attrOf(n, "where") != "venue" {
		// The link shows once the reader registers; reminders by email
		// only when a live automation sends them.
		text := v.S("event.joinShown")
		if ev.Reminders != "" && pr.pageID != uuid.Nil && !pr.preview && pr.h.remindersLive(pr.ctx, pr.s, pr.pageID) {
			text = v.S("event.joinText", "reminders", ev.Reminders)
		} else if ev.Reminders != "" && pr.preview {
			text = v.S("event.joinText", "reminders", ev.Reminders)
		}
		b.WriteString(`<div class="evd-r"><span class="ei">` + icoLink + `</span><span class="et"><b>` + esc(v.S("event.join")) + `</b><span>` + esc(text) + `</span></span></div>`)
	} else if ev.Where != "" {
		_ = icoPin
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String()) //nolint:gosec // escaped
}

// video draws a video that loads nothing until it's clicked: a link to
// the video, which pages.js turns into the player.
func (pr *pageRender) video(n *html.Node) template.HTML {
	v := pr.v
	raw := attrOf(n, "url")
	site, id, ok := pagesvc.VideoOf(raw)
	if !ok {
		if pr.preview {
			return template.HTML(`<p class="pg-empty">Video</p>`)
		}
		return ""
	}
	embed := "https://www.youtube-nocookie.com/embed/" + url.PathEscape(id) + "?autoplay=1"
	origin := "https://www.youtube-nocookie.com"
	if site == "Vimeo" {
		embed = "https://player.vimeo.com/video/" + url.PathEscape(id) + "?autoplay=1"
		origin = "https://player.vimeo.com"
	}
	pr.frames = appendOnce(pr.frames, origin)
	title := strings.TrimSpace(attrOf(n, "title"))
	if title == "" {
		title = v.S("video.play", "site", site)
	}
	return template.HTML(`<div class="pg-video"><a href="` + esc(raw) + `" data-embed="` + esc(embed) + `" rel="noopener"><span class="play" aria-hidden="true"></span><span class="vt">` + //nolint:gosec // escaped
		esc(title) + `</span><span class="vs">` + esc(v.S("video.note", "site", site)) + `</span></a></div>`)
}

func appendOnce(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// posts draws the latest posts of a tag (or of the site).
func (pr *pageRender) posts(n *html.Node) (template.HTML, error) {
	v := pr.v
	if pr.h.Website == nil {
		return "", nil
	}
	count, _ := strconv.Atoi(attrOf(n, "count"))
	if count <= 0 {
		count = 3
	}
	q := website.PublicQuery{Limit: count}
	if slug := attrOf(n, "tag"); slug != "" {
		if tag, err := pr.h.Website.TagBySlug(pr.ctx, pr.s.ws, slug); err == nil && tag != nil {
			id := tag.ID
			q.Tag = &id
		}
	}
	posts, _, err := pr.h.Website.PublicPosts(pr.ctx, pr.s.ws, q, v.labels.Lang())
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if h := strings.TrimSpace(attrOf(n, "heading")); h != "" {
		b.WriteString(`<div class="sec-head"><h2 class="sp-h2">` + esc(h) + `</h2></div>`)
	}
	if len(posts) == 0 {
		b.WriteString(`<p class="pg-empty">` + esc(v.S("posts.empty")) + `</p>`)
		return template.HTML(b.String()), nil //nolint:gosec // escaped
	}
	b.WriteString(`<div class="st-list">`)
	loc := location(pr.s.ws)
	for _, p := range posts {
		at := p.PublishedAt.In(loc)
		b.WriteString(`<a href="` + esc(pr.s.path("posts", p.Slug)) + `" class="st-row"><div class="tx"><div class="st-meta"><span><time datetime="` +
			at.UTC().Format(time.RFC3339) + `">` + esc(v.fullDate(at)) + `</time></span></div><h3>` + esc(p.Title) + `</h3>`)
		if p.Excerpt != "" {
			b.WriteString(`<p class="ex">` + esc(p.Excerpt) + `</p>`)
		}
		b.WriteString(`</div></a>`)
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String()), nil //nolint:gosec // escaped
}

// tiers draws the paid tiers with their prices, each opening the upgrade
// page.
func (pr *pageRender) tiers(n *html.Node) (template.HTML, error) {
	v := pr.v
	yearly := attrOf(n, "period") == "year"
	tiers, err := pr.h.publicTiers(pr.ctx, pr.s, v, yearly)
	if err != nil {
		return "", err
	}
	want := map[string]bool{}
	for _, t := range strings.Split(attrOf(n, "tiers"), ",") {
		if t = strings.TrimSpace(t); t != "" {
			want[t] = true
		}
	}
	var b strings.Builder
	b.WriteString(`<div class="shopg">`)
	shown := 0
	for _, t := range tiers {
		if len(want) > 0 && !want[t.ID] {
			continue
		}
		shown++
		price := t.Price
		b.WriteString(`<article class="shopc"><div class="b"><h2>` + esc(t.Name) + `</h2>`)
		if t.Description != "" {
			b.WriteString(`<p>` + esc(t.Description) + `</p>`)
		}
		b.WriteString(`<div class="f"><span class="pr">` + esc(price) + `</span><a class="st-btn full" href="` + esc(t.Href) + `">` + esc(v.S("tiers.choose")) + `</a></div></div></article>`)
	}
	b.WriteString(`</div>`)
	if shown == 0 {
		return template.HTML(`<p class="pg-empty">` + esc(v.S("tiers.none")) + `</p>`), nil //nolint:gosec // escaped
	}
	return template.HTML(b.String()), nil //nolint:gosec // escaped
}

// tierView is a paid tier of a tiers block.
type tierView struct {
	ID, Name, Description, Price, Href string
}

// publicTiers are the paid tiers as the upgrade page sells them.
func (h *Handler) publicTiers(ctx context.Context, s *site, v *siteView, yearly bool) ([]tierView, error) {
	if h.Earn == nil {
		return nil, nil
	}
	period := "month"
	if yearly {
		period = "year"
	}
	up, err := h.Earn.Upgrade(ctx, s.ws, earn.UpgradeRequest{Period: period, Lang: v.labels.Lang()})
	if err != nil || up.Unavailable {
		return nil, err
	}
	var out []tierView
	for _, c := range up.Cards {
		t := tierView{ID: c.ID, Name: c.Name, Price: strings.TrimSpace(c.Amount + " " + c.Unit), Href: s.path("upgrade")}
		if len(c.Benefits) > 0 {
			t.Description = strings.Join(c.Benefits, " · ")
		}
		out = append(out, t)
	}
	return out, nil
}
