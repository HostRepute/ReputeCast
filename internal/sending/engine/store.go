package engine

import (
	"context"

	"github.com/google/uuid"
)

// StoreContent is what store blocks need from the sending engine
// (internal/commerce): products read when the email goes out, and the
// trigger's checkout or order in automation emails.
type StoreContent interface {
	// Resolve redraws the store blocks of an email's HTML with current
	// prices and availability. Stable keeps every product in its place (a
	// hidden product stays in the HTML, invisible), so the email's tracked
	// links keep their positions across sends.
	Resolve(ctx context.Context, workspaceID uuid.UUID, html string, stable bool) string
	// Vars are a message's own link variables ("checkout_url").
	Vars(ctx context.Context, msgCtx map[string]string) map[string]string
	// Splice fills a rendered message's Checkout items from its context.
	Splice(ctx context.Context, msgCtx map[string]string, html string) string
}

// resolveStore redraws a campaign's store blocks once, when its send is
// prepared: every A/B variant and language version gets the same products.
func (e *Engine) resolveStore(ctx context.Context, ws uuid.UUID, c *sendContent) {
	if e.Store == nil {
		return
	}
	c.HTML = e.Store.Resolve(ctx, ws, c.HTML, false)
	for i := range c.Variants {
		if c.Variants[i].HTML != "" {
			c.Variants[i].HTML = e.Store.Resolve(ctx, ws, c.Variants[i].HTML, false)
		}
	}
	for i := range c.Languages {
		if c.Languages[i].HTML != "" {
			c.Languages[i].HTML = e.Store.Resolve(ctx, ws, c.Languages[i].HTML, false)
		}
		for j := range c.Languages[i].Variants {
			if c.Languages[i].Variants[j].HTML != "" {
				c.Languages[i].Variants[j].HTML = e.Store.Resolve(ctx, ws, c.Languages[i].Variants[j].HTML, false)
			}
		}
	}
}
