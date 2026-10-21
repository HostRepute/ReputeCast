package publicpages

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// The account page's Purchases section (PublicAccount): each product the
// member bought, with its download link's state and "Send me a new link",
// which goes only to the address on the order.

func init() {
	website.RegisterAccountSection(purchasesSection)
}

// purchasesSection draws the section for a member who bought something.
func purchasesSection(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, l i18n.Lang) (website.AccountSection, bool, error) {
	q := website.AccountQueries(ctx)
	if q == nil {
		return website.AccountSection{}, false, nil
	}
	cid := contactID
	rows, err := q.ProductOrdersOfContact(ctx, store.ProductOrdersOfContactParams{WorkspaceID: ws.ID, ContactID: &cid, Email: ""})
	if err != nil || len(rows) == 0 {
		return website.AccountSection{}, false, err
	}
	words := website.WordsIn(l)
	v := &siteView{words: words}
	loc := location(ws)
	now := time.Now()
	var b strings.Builder
	b.WriteString(`<div>`)
	for _, r := range rows {
		o := r.ProductOrder
		if o.Mode == "test" {
			continue
		}
		var file string
		if fs, err := q.DownloadFilesOf(ctx, store.DownloadFilesOfParams{OwnerKind: products.OwnerProduct, OwnerID: o.ProductID}); err == nil && len(fs) > 0 {
			f := fs[0].DownloadFile
			file = products.KindLabel(f.Kind, f.Name)
			if f.Pages != nil && len(fs) == 1 {
				file = v.S("dl.filePages", "kind", file, "count", int(*f.Pages), "size", products.HumanSize(f.SizeBytes))
			}
		}
		amount := products.MoneyShort(o.AmountCents, o.Currency)
		if !strings.Contains(amount, ".") {
			amount += ".00"
		}
		bought := v.S("account.bought", "file", file, "date", v.shortDate(o.PaidAt.In(loc)), "amount", amount, "order", o.Number)
		if file == "" {
			bought = strings.TrimPrefix(bought, " · ")
		}
		state, canNew := "", false
		links, _ := q.ProductOrderLatestLinks(ctx, []uuid.UUID{o.ID})
		switch {
		case o.AccessRevokedAt != nil && o.AccessRevokeReason == "refund" && o.RefundedAt != nil:
			state = v.S("account.refunded", "date", v.shortDate(o.RefundedAt.In(loc)))
		case o.AccessRevokedAt != nil:
			state = v.S("account.revoked")
		case len(links) > 0:
			lk := links[0].ProductOrderLink
			canNew = true
			switch {
			case lk.DownloadsUsed >= lk.DownloadsAllowed:
				state = v.S("account.usedUp", "total", lk.DownloadsAllowed)
			case !now.Before(lk.ExpiresAt):
				state = v.S("account.expired", "date", v.shortDate(lk.ExpiresAt.In(loc)), "used", lk.DownloadsUsed, "total", lk.DownloadsAllowed)
			default:
				state = v.S("account.works", "date", v.shortDate(lk.ExpiresAt.In(loc)), "left", lk.DownloadsAllowed-lk.DownloadsUsed, "total", lk.DownloadsAllowed)
			}
		}
		b.WriteString(`<div class="st-line top"><div class="pg">`)
		if r.CoverUrl != "" {
			b.WriteString(`<img src="` + esc(coverSrc(r.CoverUrl)) + `" alt="` + esc(r.CoverAlt) + `" width="56" height="75" loading="lazy">`)
		}
		b.WriteString(`<div class="tx"><b>` + esc(r.CurrentProductName) + `</b><span class="st-k">` + esc(bought) + `</span>`)
		if state != "" {
			b.WriteString(`<span class="st-k">` + esc(state) + `</span>`)
		}
		b.WriteString(`</div></div>`)
		if canNew {
			b.WriteString(`<form method="post" action="account/purchases/` + o.ID.String() + `"><button class="st-btn2 sm" type="submit">` + esc(v.S("account.newLink")) + `</button></form>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div><span class="st-k">` + esc(v.S("account.note")) + `</span>`)
	return website.AccountSection{ID: "purchases", Title: v.S("account.purchases"), HTML: b.String()}, true, nil
}

// coverSrc is a cover at an address the site's pages can load.
func coverSrc(src string) string { return src }

// accountNewLink sends a new link for one of the member's orders.
func (h *Handler) accountNewLink(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Products == nil || h.Website == nil {
		h.siteNotFound(w, r, s)
		return
	}
	sess := h.session(r, s)
	if sess == nil {
		http.Redirect(w, r, h.signinHref(s, s.path("account")), http.StatusSeeOther) //nolint:gosec // G710: the site's own sign-in page
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "order"))
	if err != nil {
		h.siteNotFound(w, r, s)
		return
	}
	if err := h.Products.AccountNewLink(r.Context(), s.ws, sess.ContactID, sess.Email, id); err != nil {
		h.log().WarnContext(r.Context(), "account new link", "err", err)
	} else {
		h.flash(w, r, s, "purchase_link")
	}
	http.Redirect(w, r, s.path("account")+"#purchases", http.StatusSeeOther) //nolint:gosec // G710: the site's own account page
}
