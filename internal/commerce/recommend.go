package commerce

import (
	"context"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/store"
)

// Store blocks whose products depend on the reader (M62, G-31; spec
// publish-sell §3.6a):
//
//   - Recently viewed: the store's products the reader viewed or added to
//     their cart, newest first, without what they bought.
//   - Bought together: products often in an order with what the reader
//     bought or viewed (store_product_pairs, rebuilt nightly).
//
// When an email goes out, such a block becomes Liquid over
// rc_store.<key>, its count of places each reading one product, with the
// block's best sellers (the same for everyone) for a reader without
// history. BlockRecipients fills rc_store for a page of readers at once:
// one query per block and page. Every place is drawn, so the email's links
// keep their numbers; an empty one shows nothing. Each reader's list is
// read as of their message's queueing, so the click redirect, which fills
// the link again, finds the product the email showed in its place.
//
// Viewed products (browse, cart and stock flows) is the same, over
// rc_viewed_<n>: the products the automation's trigger named (the run's
// context, Vars), read when the email goes out.

// RootStore is the per-recipient variables' root.
const RootStore = "rc_store"

// recipientKeyRe finds the per-recipient blocks a template reads: their
// mode (rv, bt), store and count.
var recipientKeyRe = regexp.MustCompile(`\brc_store\.(rv|bt)_([0-9a-f]{32})_([1-6])\b`)

// recipientKey names one block's products under rc_store.
func recipientKey(mode string, storeID uuid.UUID, n int) string {
	p := "rv"
	if mode == ModeBoughtTogether {
		p = "bt"
	}
	return p + "_" + hex.EncodeToString(storeID[:]) + "_" + strconv.Itoa(n)
}

// recipientHTML opens a per-recipient block's drawing: an {% if %} on the
// reader having products, then a grid of its places. The caller closes it
// with the drawing for everyone else.
func recipientHTML(st store.Store, b CommerceBlock, button string, words blockWords) string {
	n := blockCount(b)
	list := RootStore + "." + recipientKey(b.Mode, st.ID, n)
	var sb strings.Builder
	sb.WriteString(`{% if ` + list + `[0] %}<div style="padding:22px 28px 22px 28px;font-family:` + sans + `">`)
	sb.WriteString(headingHTML(b))
	sb.WriteString(`<div style="font-size:0;line-height:0;text-align:left">`)
	for i := range n {
		sb.WriteString(liquidCellHTML(list+"["+strconv.Itoa(i)+"]", b, button, words, i%3 == 2))
	}
	sb.WriteString(`</div></div>`)
	return sb.String()
}

// liquidCellHTML is gridCellHTML for a product read per reader: item is
// the Liquid expression of one place ({id, url, title, image, price,
// compare, sold}); a place the reader has no product for draws nothing.
func liquidCellHTML(item string, b CommerceBlock, button string, words blockWords, last bool) string {
	margin := "0 14px 18px 0"
	if last {
		margin = "0 0 18px 0"
	}
	v := func(field string) string { return `{{ ` + item + `.` + field + ` | escape }}` }
	link := `{{ ` + item + `.url }}`
	var sb strings.Builder
	sb.WriteString(`{% if ` + item + ` %}<div class="rc-product" data-rc-product="` + v("id") + `" style="display:inline-block;vertical-align:top;width:100%;max-width:162px;margin:` +
		margin + `;font-size:13px;line-height:1.3;text-align:left;">`)
	sb.WriteString(`<a href="` + link + `" style="text-decoration:none">{% if ` + item + `.image %}<img src="` + v("image") + `" alt="` + v("title") +
		`" width="162" style="display:block;width:100%;max-width:162px;height:auto;border:0;border-radius:8px;background:#F3F4F6">{% else %}` +
		`<div style="width:100%;max-width:162px;height:113px;box-sizing:border-box;border:1px solid #E3E6EB;border-radius:8px;background:#F3F4F6" role="img" aria-label="` +
		v("title") + `"></div>{% endif %}</a>`)
	sb.WriteString(`<p style="margin:10px 0 0 0;font-family:` + sans + `;font-size:13px;font-weight:600;line-height:1.3;color:#0B1220">` + v("title") + `</p>`)
	sb.WriteString(`<p style="margin:6px 0 0 0;font-family:` + sans + `;font-size:13px;line-height:1.3;color:#374151">{% if ` + item + `.sold %}` + soldPill(words) + `{% else %}` + v("price"))
	if b.CompareAt {
		sb.WriteString(`{% if ` + item + `.compare %} <s style="color:#6B7280">` + v("compare") + `</s>{% endif %}`)
	}
	sb.WriteString(`{% endif %}</p>`)
	sb.WriteString(`{% unless ` + item + `.sold %}<a href="` + link + `" style="display:inline-block;margin-top:10px;padding:6px 10px;border-radius:5px;background:#4338CA;color:#ffffff;font-family:` +
		sans + `;font-size:12px;font-weight:600;line-height:1.4;text-decoration:none">` + html.EscapeString(button) + `</a>{% endunless %}`)
	sb.WriteString(`</div>{% endif %}`)
	return sb.String()
}

// soldPill is the "Sold out" label in place of a price.
func soldPill(words blockWords) string {
	return `<span style="display:inline-block;padding:1px 7px;border-radius:999px;background:#EEF0F3;color:#4B5563;font-size:12px;font-weight:600">` +
		html.EscapeString(words.soldOut) + `</span>`
}

// linkURL is an http(s) address safe to print as it is in an attribute
// (quotes, angle brackets, braces and spaces escaped), "" otherwise. Links
// keep no HTML escaping: the click redirect reads them back as addresses.
func linkURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return urlEscaper.Replace(u.String())
}

var urlEscaper = strings.NewReplacer(`"`, "%22", "'", "%27", "<", "%3C", ">", "%3E", " ", "%20", "{", "%7B", "}", "%7D")

// recipientItem is one place of a per-recipient block.
func recipientItem(p store.StoreProduct) map[string]any {
	it := map[string]any{"id": p.ExternalID, "title": p.Title, "url": linkURL(p.Url), "price": Money(p.PriceCents, p.Currency, false)}
	if img := linkURL(p.ImageUrl); img != "" {
		it["image"] = img
	}
	if p.CompareAtCents > p.PriceCents {
		it["compare"] = Money(p.CompareAtCents, p.Currency, false)
	}
	// Liquid takes "" and false as set apart from nil only for false: a
	// product for sale has no "sold" at all.
	if !p.Available {
		it["sold"] = true
	}
	return it
}

// BlockRecipients gives each reader their per-recipient store blocks'
// products (liquid.RecipientData), in sends, previews, test emails, view
// online and the click redirect.
type BlockRecipients struct{ s *Service }

// Recipients is the service's liquid.RecipientData.
func (s *Service) Recipients() BlockRecipients { return BlockRecipients{s} }

// Roots implements liquid.RecipientData.
func (BlockRecipients) Roots() []string { return []string{RootStore} }

// Vars implements liquid.RecipientData: for each per-recipient block the
// template reads, one query for the whole page of readers.
func (r BlockRecipients) Vars(ctx context.Context, wsID uuid.UUID, recipients []liquid.Recipient, _ []string) (map[uuid.UUID]map[string]any, error) {
	out := map[uuid.UUID]map[string]any{}
	if len(recipients) == 0 {
		return out, nil
	}
	contacts := make([]uuid.UUID, len(recipients))
	messages := make([]uuid.UUID, len(recipients))
	for i, rc := range recipients {
		contacts[i], messages[i] = rc.ContactID, rc.MessageID
	}
	now := r.s.now()
	lists := map[uuid.UUID]map[string]any{}
	add := func(key string, contact uuid.UUID, p store.StoreProduct) {
		l := lists[contact]
		if l == nil {
			l = map[string]any{}
			lists[contact] = l
		}
		items, _ := l[key].([]any)
		l[key] = append(items, recipientItem(p))
	}
	seen := map[string]bool{}
	for _, m := range recipientKeyRe.FindAllStringSubmatch(liquid.SourceFrom(ctx), -1) {
		key := m[1] + "_" + m[2] + "_" + m[3]
		if seen[key] {
			continue
		}
		seen[key] = true
		storeID, err := uuid.Parse(m[2])
		if err != nil {
			continue
		}
		n, _ := strconv.Atoi(m[3])
		if m[1] == "bt" {
			rows, err := r.s.Q.CommerceBoughtTogetherFor(ctx, store.CommerceBoughtTogetherForParams{Now: now, Contacts: contacts, Messages: messages, StoreID: storeID, WorkspaceID: wsID, Lim: int32(n)})
			if err != nil {
				return out, fmt.Errorf("bought together: %w", err)
			}
			for _, row := range rows {
				add(key, row.ContactID, row.StoreProduct)
			}
			continue
		}
		rows, err := r.s.Q.CommerceRecentlyViewedFor(ctx, store.CommerceRecentlyViewedForParams{Now: now, Contacts: contacts, Messages: messages, StoreID: storeID, WorkspaceID: wsID, Lim: int32(n)})
		if err != nil {
			return out, fmt.Errorf("recently viewed: %w", err)
		}
		for _, row := range rows {
			add(key, row.ContactID, row.StoreProduct)
		}
	}
	for c, l := range lists {
		out[c] = map[string]any{RootStore: l}
	}
	return out, nil
}

// Viewed products ─────────────────────────────────────────────────────

// viewedVar names a place of the Viewed products block: rc_viewed_0 is
// its address, rc_viewed_0_title, _image, _price and _sold the rest.
func viewedVar(i int, field string) string {
	if field == "" {
		return "rc_viewed_" + strconv.Itoa(i)
	}
	return "rc_viewed_" + strconv.Itoa(i) + "_" + field
}

// viewedHTML draws the Viewed products block, in the Checkout items
// block's style: on the canvas with the store's newest products, in an
// email as Liquid over rc_viewed_<n> (viewedVars), a row per place.
func (s *Service) viewedHTML(ctx context.Context, st store.Store, b CommerceBlock, mode int, out *CommerceBlockRender, words blockWords) (string, error) {
	n := blockCount(b)
	var rows strings.Builder
	if mode == drawCanvas {
		sample, err := s.Q.CommerceProducts(ctx, store.CommerceProductsParams{StoreID: st.ID, Lim: int32(n)})
		if err != nil {
			return "", err
		}
		for i, r := range sample {
			p := productView(r)
			out.Products = append(out.Products, p)
			img := greyThumb
			if p.ImageURL != "" {
				img = `<img src="` + html.EscapeString(p.ImageURL) + `" alt="" width="48" height="48" style="` + thumbStyle + `">`
			}
			price := html.EscapeString(Money(p.Price, p.Currency, false))
			if !p.Available {
				price = soldPill(words)
			}
			rows.WriteString(viewedRow(i == 0, html.EscapeString(p.URL), img, html.EscapeString(p.Title), price))
		}
		return viewedTable(b, rows.String()), nil
	}
	for i := range n {
		v := func(field string) string { return `{{ ` + viewedVar(i, field) + ` | escape }}` }
		img := `{% if ` + viewedVar(i, "image") + ` %}<img src="` + v("image") + `" alt="" width="48" height="48" style="` + thumbStyle + `">{% else %}` + greyThumb + `{% endif %}`
		price := `{% if ` + viewedVar(i, "sold") + ` %}` + soldPill(words) + `{% else %}` + v("price") + `{% endif %}`
		rows.WriteString(`{% if ` + viewedVar(i, "") + ` %}` + viewedRow(i == 0, `{{ `+viewedVar(i, "")+` }}`, img, v("title"), price) + `{% endif %}`)
	}
	return `{% if ` + viewedVar(0, "") + ` %}` + viewedTable(b, rows.String()) + `{% endif %}`, nil
}

const (
	thumbStyle = "display:block;width:48px;height:48px;border-radius:8px;border:1px solid #E3E6EB"
	greyThumb  = `<div style="width:48px;height:48px;border-radius:8px;background:#F3F4F6;border:1px solid #E3E6EB"></div>`
)

// viewedTable is the Viewed products block around its rows.
func viewedTable(b CommerceBlock, rows string) string {
	return `<div style="padding:22px 28px 22px 28px;font-family:` + sans + `">` + headingHTML(b) +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border-collapse:separate;border:1px solid #E3E6EB;border-radius:8px;font-family:` +
		sans + `">` + rows + `</table></div>`
}

// viewedRow is one product of the Viewed products block: its picture and
// name (both linking to it) and its price; the parts are HTML already.
func viewedRow(first bool, link, img, title, price string) string {
	top := "border-top:1px solid #EEF0F3;"
	if first {
		top = ""
	}
	return `<tr><td width="60" style="padding:10px 0 10px 12px;vertical-align:middle;` + top + `"><a href="` + link + `" style="text-decoration:none">` + img + `</a></td>` +
		`<td style="padding:10px 12px;vertical-align:middle;font-size:13.5px;color:#0B1220;` + top + `"><a href="` + link +
		`" style="color:#0B1220;font-weight:600;text-decoration:none">` + title + `</a></td>` +
		`<td align="right" style="padding:10px 12px;vertical-align:middle;font-size:13.5px;font-weight:700;color:#0B1220;white-space:nowrap;` + top + `">` + price + `</td></tr>`
}

// viewedVars adds a message's Viewed products to its variables: the
// products its run's trigger named (msgCtx "products", the store's ids in
// order), as they are now. Removed products and other stores' are left out.
func (s *Service) viewedVars(ctx context.Context, msgCtx map[string]string, out map[string]string) {
	storeID, err := uuid.Parse(msgCtx["store"])
	if err != nil || msgCtx["products"] == "" {
		return
	}
	var ids []uuid.UUID
	for _, raw := range strings.Split(msgCtx["products"], ",") {
		if id, err := uuid.Parse(strings.TrimSpace(raw)); err == nil && len(ids) < 6 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	rows, err := s.Q.CommerceProductsByIDs(ctx, ids)
	if err != nil {
		return
	}
	soldOut := ""
	i := 0
	for _, id := range ids {
		for _, p := range rows {
			if p.ID != id || p.StoreID != storeID || p.RemovedAt != nil {
				continue
			}
			link := linkURL(p.Url)
			if link == "" {
				continue
			}
			out[viewedVar(i, "")] = link
			out[viewedVar(i, "title")] = p.Title
			out[viewedVar(i, "price")] = Money(p.PriceCents, p.Currency, false)
			if img := linkURL(p.ImageUrl); img != "" {
				out[viewedVar(i, "image")] = img
			}
			if !p.Available {
				if soldOut == "" {
					soldOut = mSoldOut.Text(s.emailLang(ctx, p.WorkspaceID), nil)
				}
				out[viewedVar(i, "sold")] = soldOut
			}
			i++
		}
	}
}
