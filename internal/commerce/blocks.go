package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// Store blocks in emails. The composer keeps a block's settings on its
// section (rc-store) and its drawing in an mj-raw between two markers:
//
//	<!--rc:store-items {"store":"…","mode":"collection",…}--> … <!--/rc:store-items-->
//
// The markers survive both MJML compilers, so the compiled HTML says where
// each block is: when a campaign's send is prepared, and when an
// automation email goes out, the drawing is replaced by one with the
// products' prices and availability at that moment.
//
// The Checkout items block of automation emails sits between
// <!--rc:checkout-items--> and <!--/rc:checkout-items-->; each message gets
// the items of the checkout or order that started its run. Its "Return to
// checkout" button links to {{ checkout_url }}.

const (
	itemsOpen     = "<!--rc:store-items "
	itemsClose    = "<!--/rc:store-items-->"
	checkoutOpen  = "<!--rc:checkout-items-->"
	checkoutClose = "<!--/rc:checkout-items-->"
)

var storeItemsRe = regexp.MustCompile(`(?s)<!--rc:store-items (\{.*?\})-->(.*?)<!--/rc:store-items-->`)

// CommerceBlockRenderInput draws a block.
type CommerceBlockRenderInput struct {
	Block CommerceBlock `json:"block"`
}

// RenderBlock draws a store block with its products now, for the
// composer's canvas: removed products are drawn, and flagged.
func (s *Service) RenderBlock(ctx context.Context, ws *store.Workspace, b CommerceBlock) (CommerceBlockRender, error) {
	l := i18n.LangFrom(ctx)
	id, err := uuid.Parse(b.Store)
	if err != nil {
		return CommerceBlockRender{}, api.Invalid("body.block.store", mBlockStore.Text(l, nil))
	}
	st, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return CommerceBlockRender{}, api.Invalid("body.block.store", mBlockStore.Text(l, nil))
	}
	if err != nil {
		return CommerceBlockRender{}, err
	}
	switch b.Mode {
	case "card", "chosen":
		if len(b.Products) == 0 {
			return CommerceBlockRender{}, api.Invalid("body.block.products", mBlockProducts.Text(l, nil))
		}
	case "collection":
		if b.Collection == "" {
			return CommerceBlockRender{}, api.Invalid("body.block.collection", mBlockCollection.Text(l, nil))
		}
	}
	return s.draw(ctx, st, b, drawCanvas)
}

// draw modes: the composer's canvas (removed products flagged, nothing
// hidden), a campaign's send (sold-out and removed products left out) and
// an automation email (left in place, invisible: links keep their places).
const (
	drawCanvas = iota
	drawSend
	drawStable
)

func (s *Service) draw(ctx context.Context, st store.Store, b CommerceBlock, mode int) (CommerceBlockRender, error) {
	out := CommerceBlockRender{StoreName: st.Name, Platform: st.Platform, Products: []CommerceProduct{}, Removed: []CommerceRemovedProduct{}, SoldOut: []string{},
		Disconnected: st.Status == StatusDisconnected}
	// The block's words go into the email: in the workspace's default
	// language, the language its emails are written in. The canvas's flag
	// over a removed product is for the member.
	el := s.emailLang(ctx, st.WorkspaceID)
	flagText := mRemovedFlag.Text(i18n.LangFrom(ctx), nil)
	var rows []store.StoreProduct
	var err error
	switch b.Mode {
	case "collection":
		n := b.Count
		if n <= 0 {
			n = 3
		}
		rows, err = s.Q.CommerceProductsNewest(ctx, store.CommerceProductsNewestParams{StoreID: st.ID, Collection: b.Collection, Lim: int32(n + 6)})
		if err != nil {
			return out, err
		}
		if b.SoldOut != "label" {
			kept := rows[:0]
			for _, r := range rows {
				if r.Available || mode == drawStable {
					kept = append(kept, r)
				}
			}
			rows = kept
		}
		if len(rows) > n {
			rows = rows[:n]
		}
	default:
		ids := b.Products
		if b.Mode == "card" && len(ids) > 1 {
			ids = ids[:1]
		}
		found, err := s.Q.CommerceProductsByExternal(ctx, store.CommerceProductsByExternalParams{StoreID: st.ID, Ids: ids})
		if err != nil {
			return out, err
		}
		for _, id := range ids {
			if i := slices.IndexFunc(found, func(r store.StoreProduct) bool { return r.ExternalID == id }); i >= 0 {
				rows = append(rows, found[i])
			}
		}
	}
	type shown struct {
		p      CommerceProduct
		hidden bool
		sold   bool
		// flag marks a removed product on the composer's canvas.
		flag bool
	}
	var items []shown
	for _, r := range rows {
		p := productView(r)
		if p.Removed {
			out.Removed = append(out.Removed, CommerceRemovedProduct{ID: p.ID, Title: p.Title, RemovedAt: p.RemovedAt, Platform: st.Platform})
			if mode == drawSend {
				continue
			}
			items = append(items, shown{p: p, hidden: mode == drawStable, flag: mode == drawCanvas})
			continue
		}
		if !p.Available {
			out.SoldOut = append(out.SoldOut, p.Title)
			if b.SoldOut != "label" {
				if mode == drawSend {
					continue
				}
				items = append(items, shown{p: p, hidden: mode == drawStable, sold: mode == drawCanvas})
				continue
			}
			items = append(items, shown{p: p, sold: true})
			continue
		}
		items = append(items, shown{p: p})
	}
	for _, it := range items {
		out.Products = append(out.Products, it.p)
	}
	button := b.Button
	if strings.TrimSpace(button) == "" {
		button = mShopNow.Text(el, nil)
	}
	words := blockWords{soldOut: mSoldOut.Text(el, nil), removed: flagText}
	var sb strings.Builder
	cfg, _ := json.Marshal(b)
	sb.WriteString(itemsOpen)
	sb.Write(cfg)
	sb.WriteString("-->")
	visible := 0
	for _, it := range items {
		if !it.hidden {
			visible++
		}
	}
	if visible > 0 || mode == drawCanvas {
		if b.Mode == "card" && len(items) > 0 {
			sb.WriteString(cardHTML(items[0].p, b, button, words, items[0].sold, items[0].hidden, items[0].flag))
		} else {
			sb.WriteString(`<div style="padding:22px 28px 22px 28px;font-family:'IBM Plex Sans',Helvetica,Arial,sans-serif">`)
			if h := strings.TrimSpace(b.Heading); h != "" {
				sb.WriteString(`<h2 style="margin:0 0 14px 0;font-family:Georgia,'Times New Roman',serif;font-size:20px;line-height:1.3;color:#0B1220">`)
				sb.WriteString(html.EscapeString(h))
				sb.WriteString(`</h2>`)
			}
			sb.WriteString(`<div style="font-size:0;line-height:0;text-align:left">`)
			for i, it := range items {
				sb.WriteString(gridCellHTML(it.p, b, button, words, it.sold, it.hidden, it.flag, i%3 == 2))
			}
			sb.WriteString(`</div></div>`)
		}
	}
	sb.WriteString(itemsClose)
	out.HTML = sb.String()
	return out, nil
}

func productView(r store.StoreProduct) CommerceProduct {
	p := CommerceProduct{ID: r.ExternalID, Title: r.Title, URL: r.Url, ImageURL: r.ImageUrl, ImageAlt: r.ImageAlt, Price: r.PriceCents, CompareAt: r.CompareAtCents,
		Currency: r.Currency, Available: r.Available, Removed: r.RemovedAt != nil, RemovedAt: r.RemovedAt}
	if r.Inventory != nil {
		n := int(*r.Inventory)
		p.Inventory = &n
	}
	if p.ImageAlt == "" {
		p.ImageAlt = r.Title
	}
	return p
}

const (
	hideStyle = "display:none;max-height:0;overflow:hidden;mso-hide:all;"
	sans      = "'IBM Plex Sans',Helvetica,Arial,sans-serif"
)

// blockWords are a store block's own words: in the email's language, and
// the canvas's flag in the member's.
type blockWords struct{ soldOut, removed string }

// priceHTML is "$58", or "$58 $64" with the compare-at price struck.
func priceHTML(p CommerceProduct, b CommerceBlock, words blockWords, sold bool) string {
	if sold {
		return `<span style="display:inline-block;padding:1px 7px;border-radius:999px;background:#EEF0F3;color:#4B5563;font-size:12px;font-weight:600">` +
			html.EscapeString(words.soldOut) + `</span>`
	}
	out := html.EscapeString(Money(p.Price, p.Currency, false))
	if b.CompareAt && p.CompareAt > p.Price {
		out += ` <s style="color:#6B7280">` + html.EscapeString(Money(p.CompareAt, p.Currency, false)) + `</s>`
	}
	return out
}

func imageHTML(p CommerceProduct, width int) string {
	if p.ImageURL == "" {
		// A product without a picture keeps its place: the boards' grey tile.
		return fmt.Sprintf(`<div style="width:100%%;max-width:%dpx;height:%dpx;box-sizing:border-box;border:1px solid #E3E6EB;border-radius:8px;background:#F3F4F6" role="img" aria-label="%s"></div>`,
			width, width*7/10, html.EscapeString(p.ImageAlt))
	}
	return fmt.Sprintf(`<img src="%s" alt="%s" width="%d" style="display:block;width:100%%;max-width:%dpx;height:auto;border:0;border-radius:8px;background:#F3F4F6">`,
		html.EscapeString(p.ImageURL), html.EscapeString(p.ImageAlt), width, width)
}

// removedFlag is the canvas's chip over a product the store removed.
func removedFlag(text string) string {
	return `<span style="position:absolute;top:-30px;left:-8px;display:inline-flex;align-items:center;gap:5px;height:22px;padding:0 8px;border-radius:6px;` +
		`background:#B91C1C;color:#ffffff;font-family:` + sans + `;font-size:11px;font-weight:600;line-height:22px;white-space:nowrap">` + html.EscapeString(text) + `</span>`
}

func gridCellHTML(p CommerceProduct, b CommerceBlock, button string, words blockWords, sold, hidden, flag, last bool) string {
	margin := "0 14px 18px 0"
	if last {
		margin = "0 0 18px 0"
	}
	style := "display:inline-block;vertical-align:top;width:100%;max-width:162px;margin:" + margin + ";font-size:13px;line-height:1.3;text-align:left;"
	if hidden {
		style += hideStyle
	}
	if flag {
		style += "position:relative;outline:2px solid #DC2626;outline-offset:6px;border-radius:8px;"
	}
	var sb strings.Builder
	sb.WriteString(`<div class="rc-product" data-rc-product="` + html.EscapeString(p.ID) + `" style="` + style + `">`)
	if flag {
		sb.WriteString(removedFlag(words.removed))
	}
	link := html.EscapeString(p.URL)
	if link != "" {
		sb.WriteString(`<a href="` + link + `" style="text-decoration:none">` + imageHTML(p, 162) + `</a>`)
	} else {
		sb.WriteString(imageHTML(p, 162))
	}
	sb.WriteString(`<p style="margin:10px 0 0 0;font-family:` + sans + `;font-size:13px;font-weight:600;line-height:1.3;color:#0B1220">` + html.EscapeString(p.Title) + `</p>`)
	sb.WriteString(`<p style="margin:6px 0 0 0;font-family:` + sans + `;font-size:13px;line-height:1.3;color:#374151">` + priceHTML(p, b, words, sold) + `</p>`)
	if !sold && link != "" {
		sb.WriteString(`<a href="` + link + `" style="display:inline-block;margin-top:10px;padding:6px 10px;border-radius:5px;background:#4338CA;color:#ffffff;font-family:` + sans +
			`;font-size:12px;font-weight:600;line-height:1.4;text-decoration:none">` + html.EscapeString(button) + `</a>`)
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

// cardHTML is the Product card's drawing: the image beside the name, the
// price and the button.
func cardHTML(p CommerceProduct, b CommerceBlock, button string, words blockWords, sold, hidden, flag bool) string {
	style := "padding:16px 24px;font-family:" + sans + ";"
	if hidden {
		style += hideStyle
	}
	if flag {
		style += "position:relative;outline:2px solid #DC2626;outline-offset:-6px;border-radius:8px;"
	}
	link := html.EscapeString(p.URL)
	var sb strings.Builder
	sb.WriteString(`<div class="rc-product" data-rc-product="` + html.EscapeString(p.ID) + `" style="` + style + `">`)
	if flag {
		sb.WriteString(strings.Replace(removedFlag(words.removed), "top:-30px;left:-8px", "top:-11px;left:16px", 1))
	}
	sb.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border-collapse:collapse"><tr>`)
	sb.WriteString(`<td width="40%" style="vertical-align:top;padding:0 16px 0 0">`)
	if link != "" {
		sb.WriteString(`<a href="` + link + `">` + imageHTML(p, 168) + `</a>`)
	} else {
		sb.WriteString(imageHTML(p, 168))
	}
	sb.WriteString(`</td><td style="vertical-align:top">`)
	sb.WriteString(`<p style="margin:0;font-size:17px;font-weight:700;line-height:1.3;color:#0B1220">` + html.EscapeString(p.Title) + `</p>`)
	sb.WriteString(`<p style="margin:8px 0 0 0;font-size:14px;line-height:1.4;color:#374151">` + priceHTML(p, b, words, sold) + `</p>`)
	if !sold && link != "" {
		sb.WriteString(`<a href="` + link + `" style="display:inline-block;margin-top:14px;padding:9px 16px;border-radius:6px;background:#0F766E;color:#ffffff;font-size:14px;font-weight:600;text-decoration:none">` +
			html.EscapeString(button) + `</a>`)
	}
	sb.WriteString(`</td></tr></table></div>`)
	return sb.String()
}

// Money writes an amount the way emails show prices: "$58", or "$58.00"
// with the cents always shown.
func Money(v int64, currency string, cents bool) string {
	amt := stores.FormatAmount(v, currency)
	if !cents && strings.HasSuffix(amt, ".00") {
		amt = strings.TrimSuffix(amt, ".00")
	}
	switch strings.ToUpper(currency) {
	case "USD", "":
		return "$" + amt
	case "EUR":
		return "€" + amt
	case "GBP":
		return "£" + amt
	case "CAD":
		return "CA$" + amt
	case "AUD":
		return "A$" + amt
	case "JPY":
		return "¥" + amt
	}
	return amt + " " + strings.ToUpper(currency)
}

// Resolve implements the sending engine's StoreContent: every store block
// of an email is drawn again with what the products cost and whether
// they're available now. A block whose store is disconnected keeps its
// last drawing.
func (s *Service) Resolve(ctx context.Context, ws uuid.UUID, h string, stable bool) string {
	if !strings.Contains(h, itemsOpen) {
		return h
	}
	mode := drawSend
	if stable {
		mode = drawStable
	}
	return storeItemsRe.ReplaceAllStringFunc(h, func(region string) string {
		m := storeItemsRe.FindStringSubmatch(region)
		var b CommerceBlock
		if len(m) < 2 || json.Unmarshal([]byte(m[1]), &b) != nil {
			return region
		}
		id, err := uuid.Parse(b.Store)
		if err != nil {
			return region
		}
		st, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: id, WorkspaceID: ws})
		if err != nil || st.Status == StatusDisconnected {
			return region
		}
		r, err := s.draw(ctx, st, b, mode)
		if err != nil {
			return region
		}
		return r.HTML
	})
}

// Vars implements StoreContent: a message's checkout_url, the store's
// recovery link of the checkout that started its run (an order's store
// front page).
func (s *Service) Vars(ctx context.Context, msgCtx map[string]string) map[string]string {
	if id, err := uuid.Parse(msgCtx["checkout"]); err == nil {
		if ch, err := s.Q.CommerceCheckoutByID(ctx, id); err == nil && ch.RecoveryUrl != "" {
			return map[string]string{"checkout_url": ch.RecoveryUrl}
		}
	}
	if id, err := uuid.Parse(msgCtx["store"]); err == nil {
		if st, err := s.Q.CommerceStoreByID(ctx, id); err == nil {
			host := st.PublicDomain
			if host == "" {
				host = st.Domain
			}
			return map[string]string{"checkout_url": "https://" + host + "/"}
		}
	}
	return nil
}

// Splice implements StoreContent: the Checkout items block gets the items
// of the checkout or order that started the message's run.
func (s *Service) Splice(ctx context.Context, msgCtx map[string]string, h string) string {
	start := strings.Index(h, checkoutOpen)
	if start < 0 {
		return h
	}
	end := strings.Index(h[start:], checkoutClose)
	if end < 0 {
		return h
	}
	end += start + len(checkoutClose)
	var items []orderItem
	var total int64
	currency := ""
	var wsID uuid.UUID
	switch {
	case msgCtx["checkout"] != "":
		id, err := uuid.Parse(msgCtx["checkout"])
		if err != nil {
			return h
		}
		ch, err := s.Q.CommerceCheckoutByID(ctx, id)
		if err != nil {
			return h
		}
		_ = json.Unmarshal(ch.Items, &items)
		total, currency, wsID = ch.TotalCents, ch.Currency, ch.WorkspaceID
	case msgCtx["order"] != "":
		id, err := uuid.Parse(msgCtx["order"])
		if err != nil {
			return h
		}
		o, err := s.Q.CommerceOrderByID(ctx, id)
		if err != nil {
			return h
		}
		_ = json.Unmarshal(o.Items, &items)
		total, currency, wsID = o.TotalCents, o.Currency, o.WorkspaceID
	default:
		return h
	}
	return h[:start] + CheckoutItemsHTMLIn(s.emailLang(ctx, wsID), items, total, currency) + h[end:]
}

// CheckoutItemsHTML draws checkout items in English: each line with its
// variant and quantity and price, then the total.
func CheckoutItemsHTML(items []orderItem, total int64, currency string) string {
	return CheckoutItemsHTMLIn(i18n.Default, items, total, currency)
}

// CheckoutItemsHTMLIn draws checkout items with the total's label in a
// language: the email's.
func CheckoutItemsHTMLIn(l i18n.Lang, items []orderItem, total int64, currency string) string {
	var sb strings.Builder
	sb.WriteString(checkoutOpen)
	sb.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border-collapse:separate;border:1px solid #E3E6EB;border-radius:8px;font-family:` + sans + `">`)
	for i, it := range items {
		top := "border-top:1px solid #EEF0F3;"
		if i == 0 {
			top = ""
		}
		detail := fmt.Sprint(it.Quantity)
		if it.Variant != "" {
			detail = it.Variant + " · " + detail
		}
		img := `<div style="width:48px;height:48px;border-radius:8px;background:#F3F4F6;border:1px solid #E3E6EB"></div>`
		if it.ImageURL != "" {
			img = `<img src="` + html.EscapeString(it.ImageURL) + `" alt="" width="48" height="48" style="display:block;width:48px;height:48px;border-radius:8px;border:1px solid #E3E6EB">`
		}
		sb.WriteString(`<tr><td width="60" style="padding:10px 0 10px 12px;vertical-align:middle;` + top + `">` + img + `</td>`)
		sb.WriteString(`<td style="padding:10px 12px;vertical-align:middle;font-size:13.5px;color:#0B1220;` + top + `"><b style="font-weight:600">` + html.EscapeString(it.Title) +
			`</b><br><span style="font-size:12px;color:#5B6472">` + html.EscapeString(detail) + `</span></td>`)
		sb.WriteString(`<td align="right" style="padding:10px 12px;vertical-align:middle;font-size:13.5px;font-weight:700;color:#0B1220;white-space:nowrap;` + top + `">` +
			html.EscapeString(Money(it.Price*int64(max(it.Quantity, 1)), currency, true)) + `</td></tr>`)
	}
	sb.WriteString(`<tr><td colspan="2" style="padding:10px 12px;background:#F9FAFB;border-top:1px solid #E3E6EB;font-size:13.5px;font-weight:700;color:#0B1220">` +
		html.EscapeString(mTotal.Text(l, nil)) + `</td>`)
	sb.WriteString(`<td align="right" style="padding:10px 12px;background:#F9FAFB;border-top:1px solid #E3E6EB;font-size:13.5px;font-weight:700;color:#0B1220;white-space:nowrap">` +
		html.EscapeString(Money(total, currency, true)) + `</td></tr></table>`)
	sb.WriteString(checkoutClose)
	return sb.String()
}

// CheckoutBlockHTML is the Checkout items block as an automation email
// carries it, in English: sample items between the markers, and the
// button.
func CheckoutBlockHTML(sample []orderItem, total int64, currency, button string) string {
	return CheckoutBlockHTMLIn(i18n.Default, sample, total, currency, button)
}

// CheckoutBlockHTMLIn is CheckoutBlockHTML with its words in a language:
// the email's.
func CheckoutBlockHTMLIn(l i18n.Lang, sample []orderItem, total int64, currency, button string) string {
	if button == "" {
		button = mReturnToCheckout.Text(l, nil)
	}
	return CheckoutItemsHTMLIn(l, sample, total, currency) +
		`<div style="padding:16px 0 0 0"><a href="{{ checkout_url }}" style="display:inline-block;padding:11px 18px;border-radius:6px;background:#4338CA;color:#ffffff;font-family:` +
		sans + `;font-size:14px;font-weight:600;text-decoration:none">` + html.EscapeString(button) + `</a></div>`
}

// CommerceCheckoutBlock is the Checkout items block's HTML, with sample
// items from the store, for an automation email.
type CommerceCheckoutBlock struct {
	HTML string `json:"html"`
}

// CheckoutBlock draws the Checkout items block with sample items: a store's
// newest products.
func (s *Service) CheckoutBlock(ctx context.Context, ws *store.Workspace, storeID uuid.UUID) (CommerceCheckoutBlock, error) {
	st, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: storeID, WorkspaceID: ws.ID})
	if err != nil {
		return CommerceCheckoutBlock{}, api.Err(http.StatusNotFound, api.CodeNotFound, mStoreNotFound.Text(i18n.LangFrom(ctx), nil))
	}
	rows, err := s.Q.CommerceProducts(ctx, store.CommerceProductsParams{StoreID: st.ID, Lim: 2})
	if err != nil {
		return CommerceCheckoutBlock{}, err
	}
	var items []orderItem
	var total int64
	for _, r := range rows {
		items = append(items, orderItem{ProductID: r.ExternalID, Title: r.Title, Quantity: 1, Price: r.PriceCents, ImageURL: r.ImageUrl})
		total += r.PriceCents
	}
	return CommerceCheckoutBlock{HTML: CheckoutBlockHTMLIn(s.emailLang(ctx, ws.ID), items, total, st.Currency, "")}, nil
}
