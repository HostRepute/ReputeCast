package publicpages

import (
	"context"
	"strings"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/products"
)

// The confirmation page's download (design expansion phase 7, X19): a
// signup form's "After confirmation: offer a download". The reader who
// confirmed gets a link of their own, working for the form's days.

// confirmDownload is the download card under the confirmation.
type confirmDownload struct {
	Name, Meta, Button, Href, Note string
}

// confirmDownloadFor is the download the form the contact signed up with
// offers, if any.
func (h *Handler) confirmDownloadFor(ctx context.Context, s *site, pc audience.PublicContact, code string) *confirmDownload {
	if h.Products == nil {
		return nil
	}
	last, err := h.Q.FormLastSubmissionForContact(ctx, pc.ID)
	if err != nil || last.FormID == nil {
		return nil
	}
	offer, ok, err := h.Products.LeadMagnetFor(ctx, s.ws, strings.TrimSuffix(s.path(), "/"), *last.FormID, pc.ID)
	if err != nil {
		h.log().WarnContext(ctx, "lead magnet link", "err", err)
		return nil
	}
	if !ok || offer.URL == "" {
		return nil
	}
	l := i18n.For(i18n.English)
	if loc, ok := i18n.ParseLocale(code); ok {
		l = i18n.For(loc)
	}
	text := func(key string, args i18n.Args) string { return sellMsgs.M(key).Text(l, args) }
	size := products.HumanSize(offer.Size)
	meta := []string{offer.Kind}
	if offer.Pages > 0 {
		meta = append(meta, text("confirm.download.pages", i18n.Args{"count": offer.Pages}))
	}
	meta = append(meta, size)
	return &confirmDownload{Name: offer.Name, Meta: strings.Join(meta, " · "), Href: offer.URL,
		Button: text("confirm.download.button", i18n.Args{"kind": offer.Kind, "size": size}),
		Note:   text("confirm.download.note", i18n.Args{"days": offer.Days})}
}
