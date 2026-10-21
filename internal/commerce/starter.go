package commerce

import (
	"context"
	"html"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// StarterEmail writes the Abandoned cart starter's emails in the store's
// voice (automations' Service.StoreEmail): the first with the Checkout
// items block (sample items from the catalog on the canvas, the
// checkout's own items when it's sent), the second with the store's
// newest products.
func (s *Service) StarterEmail(ctx context.Context, wsID, storeID uuid.UUID, key string) (automations.EmailContent, bool) {
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return automations.EmailContent{}, false
	}
	st, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: storeID, WorkspaceID: wsID})
	if err != nil {
		return automations.EmailContent{}, false
	}
	name := html.EscapeString(st.Name)
	var subject, preview, intro, section string
	switch key {
	case "c1":
		cb, err := s.CheckoutBlock(ctx, &ws, storeID)
		if err != nil {
			return automations.EmailContent{}, false
		}
		subject, preview = "You left something behind", "Your cart is saved."
		intro = `Hi {{ first_name | default: "there" }}, your cart at ` + name + ` is saved. Here's what's in it:`
		section = CheckoutSection(storeID, cb.HTML)
	case "c2":
		// The newest of the store's new arrivals (or its first collection),
		// else its three newest products.
		b := CommerceBlock{Store: storeID.String(), Mode: "collection", Count: 3, SoldOut: "hide", Button: "Shop now", Heading: "New in the " + st.Name + " shop"}
		if cols, err := s.Q.CommerceCollections(ctx, st.ID); err == nil && len(cols) > 0 {
			b.Collection = cols[0].ExternalID
			for _, c := range cols {
				if strings.Contains(strings.ToLower(c.Title), "new") && c.Products > 0 {
					b.Collection = c.ExternalID
					break
				}
			}
		} else {
			rows, err := s.Q.CommerceProducts(ctx, store.CommerceProductsParams{StoreID: st.ID, Lim: 3})
			if err != nil || len(rows) == 0 {
				return automations.EmailContent{}, false
			}
			b.Mode = "chosen"
			for _, r := range rows {
				b.Products = append(b.Products, r.ExternalID)
			}
		}
		r, err := s.RenderBlock(ctx, &ws, b)
		if err != nil {
			return automations.EmailContent{}, false
		}
		subject, preview = "Still deciding?", "Your cart is still saved."
		intro = `Still deciding, {{ first_name | default: "there" }}? Your cart is saved for a few more days. And here's what's new in the shop:`
		section = GridSection(r.HTML)
	default:
		return automations.EmailContent{}, false
	}
	src := `<mjml><mj-head><mj-title>` + subject + `</mj-title><mj-preview>` + preview + `</mj-preview><mj-attributes><mj-all font-family="IBM Plex Sans, Arial, sans-serif"></mj-all></mj-attributes></mj-head>
<mj-body background-color="#F3F4F6" width="600px">
  <mj-section background-color="#ffffff" padding="24px 24px 8px"><mj-column>
    <mj-text font-size="13px" font-weight="700" color="#0F766E" padding="0px 0px 12px">` + name + `</mj-text>
    <mj-text font-size="22px" font-weight="700" color="#0B1220" padding="0px 0px 10px">` + subject + `</mj-text>
    <mj-text font-size="15px" line-height="1.6" color="#374151" padding="0px 0px 12px">` + intro + `</mj-text>
  </mj-column></mj-section>
  ` + section + `
  <mj-section background-color="#ffffff" padding="8px 24px 24px"><mj-column>
    <mj-text font-size="12px" line-height="1.5" color="#5B6472">You get this because you agreed to emails from ` + name + ` at checkout. <a href="{{ unsubscribe_url }}" style="color:#5B6472">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#5B6472">Email preferences</a></mj-text>
  </mj-column></mj-section>
</mj-body></mjml>`
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		return automations.EmailContent{}, false
	}
	return automations.EmailContent{Subject: subject, Preheader: preview, MJML: src, HTML: res.HTML, Text: subject + "\n\n" + preview}, true
}

// CheckoutSection is the Checkout items block as the composer keeps it: a
// section named for the layer tree, its drawing in a raw block.
func CheckoutSection(storeID uuid.UUID, drawing string) string {
	return `<mj-section background-color="#ffffff" padding="0px 24px" rc-name="Checkout items" rc-checkout='{"store":"` + storeID.String() + `"}'><mj-column><mj-raw>` +
		drawing + `</mj-raw></mj-column></mj-section>`
}

// GridSection is a product grid or card as the composer keeps it: the
// block's settings on the section (read from its drawing's marker), its
// drawing in a raw block.
func GridSection(drawing string) string {
	cfg, name := "", "Product grid"
	if m := storeItemsRe.FindStringSubmatch(drawing); len(m) > 1 {
		cfg = m[1]
		if strings.Contains(cfg, `"mode":"card"`) {
			name = "Product card"
		}
	}
	return `<mj-section background-color="#ffffff" padding="0px" rc-store='` + strings.ReplaceAll(cfg, "'", "&#39;") + `' rc-name="` + name + `"><mj-column><mj-raw>` +
		drawing + `</mj-raw></mj-column></mj-section>`
}
