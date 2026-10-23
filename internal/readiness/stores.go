package readiness

import "strconv"

// Store blocks (design expansion phase 7, X20): a product card or grid
// naming a product the store removed warns in the Content group. The send
// leaves the product out; the editor offers another one. The composer's
// content check finds it (internal/commerce adds it to the checker).

// issueProductRemoved is internal/commerce's content check.
const issueProductRemoved = "store.product_removed"

// stores adds the removed-product checks. Their title, detail and short
// label are the store's content check's own (English).
func (b *builder) stores() {
	if !b.f.Content.Checked {
		return
	}
	for i, is := range issuesOf(b.f.Content.Report, issueProductRemoved) {
		b.add(check{Key: "content.store_product." + strconv.Itoa(i), Group: GroupContent, Status: Warn, Label: tx("label.storeProducts", nil),
			Title: RawText(is.Title), Detail: RawText(is.Detail), Short: RawText(is.Tag),
			Ack:   tx("stores.ack", nil),
			Fixes: []fix{link(tx("fix.openInEditor", nil), b.editor(is.Path, ""))}})
	}
}
