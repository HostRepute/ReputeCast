package readiness

import "strconv"

// Store blocks (design expansion phase 7, X20): a product card or grid
// naming a product the store removed warns in the Content group. The send
// leaves the product out; the editor offers another one. The composer's
// content check finds it (internal/commerce adds it to the checker).

// issueProductRemoved is internal/commerce's content check.
const issueProductRemoved = "store.product_removed"

// stores adds the removed-product checks.
func (b *builder) stores() {
	if !b.f.Content.Checked {
		return
	}
	for i, is := range issuesOf(b.f.Content.Report, issueProductRemoved) {
		b.add(ReadinessCheck{Key: "content.store_product." + strconv.Itoa(i), Group: GroupContent, Status: Warn, Label: "Store products",
			Title: is.Title, Detail: is.Detail, Short: is.Tag,
			Ack:   "I understand the email names a product the store removed.",
			Fixes: []ReadinessFix{link("Open in editor", b.editor(is.Path, ""))}})
	}
}
