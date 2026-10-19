package products

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// What the public pages show of products: the shop, product blocks, the
// default product page and the account's purchases.

// PublicProduct is a product as readers see it: its published version (or
// its draft, for the builder's previews).
type PublicProduct struct {
	ID       uuid.UUID
	Status   string
	Live     Live
	Files    ProductFilesSummary
	Currency string
	// SalesPage is the published sales page's address ("" for none: the
	// default product page).
	SalesPage string
	// OfferList and OfferNote are the list offer's list and its line
	// ("One email a week.").
	OfferList string
	OfferNote string
	// Published: readers can buy it.
	Published bool
}

// Merchant is who buyers pay.
type Merchant struct {
	Connected bool
	Account   string
	Test      bool
	Tax       bool
	Currency  string
}

// Merchant reads the payment connection products sell through.
func (s *Service) Merchant(ctx context.Context, ws *store.Workspace) (Merchant, error) {
	if s.Earn == nil || s.Earn() == nil {
		return Merchant{}, nil
	}
	sel, ok, err := s.Earn().Selling(ctx, ws)
	if err != nil || !ok {
		return Merchant{}, err
	}
	return Merchant{Connected: true, Account: sel.Account, Test: sel.Mode == payments.Test, Tax: sel.TaxAtCheckout, Currency: sel.Currency}, nil
}

// PublicProducts reads products by id; draft products only when drafts.
func (s *Service) PublicProducts(ctx context.Context, ws *store.Workspace, ids []uuid.UUID, drafts bool) (map[uuid.UUID]PublicProduct, error) {
	out := map[uuid.UUID]PublicProduct{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Q.ProductsByIDs(ctx, store.ProductsByIDsParams{WorkspaceID: ws.ID, Ids: ids})
	if err != nil {
		return nil, err
	}
	return s.publicOf(ctx, ws, rows, drafts)
}

// ShopProducts are the published products shown in the shop.
func (s *Service) ShopProducts(ctx context.Context, ws *store.Workspace) ([]PublicProduct, error) {
	rows, err := s.Q.ProductsShop(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	m, err := s.publicOf(ctx, ws, rows, false)
	if err != nil {
		return nil, err
	}
	out := make([]PublicProduct, 0, len(rows))
	for _, r := range rows {
		if p, ok := m[r.ID]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// ProductBySlug is the product answering at /products/{slug}; moved is the
// address it moved to.
func (s *Service) ProductBySlug(ctx context.Context, ws *store.Workspace, slug string, drafts bool) (PublicProduct, string, error) {
	p, err := s.Q.ProductByLiveSlug(ctx, store.ProductByLiveSlugParams{WorkspaceID: ws.ID, Slug: &slug})
	if errors.Is(err, pgx.ErrNoRows) {
		if id, rerr := s.Q.ProductRedirectGet(ctx, store.ProductRedirectGetParams{WorkspaceID: ws.ID, Slug: slug}); rerr == nil {
			if moved, merr := s.Q.ProductAny(ctx, id); merr == nil {
				if l, ok := liveOf(moved); ok {
					return PublicProduct{}, l.Slug, nil
				}
			}
		}
		if drafts {
			// The builder's preview of a draft's default page.
			if row, derr := s.draftBySlug(ctx, ws, slug); derr == nil {
				m, err := s.publicOf(ctx, ws, []store.Product{row}, true)
				if err != nil {
					return PublicProduct{}, "", err
				}
				return m[row.ID], "", nil
			}
		}
		return PublicProduct{}, "", ErrNotForSale
	}
	if err != nil {
		return PublicProduct{}, "", err
	}
	m, err := s.publicOf(ctx, ws, []store.Product{p}, drafts)
	if err != nil {
		return PublicProduct{}, "", err
	}
	out, ok := m[p.ID]
	if !ok {
		return PublicProduct{}, "", ErrNotForSale
	}
	return out, "", nil
}

func (s *Service) draftBySlug(ctx context.Context, ws *store.Workspace, slug string) (store.Product, error) {
	var id uuid.UUID
	if err := s.Pool.QueryRow(ctx, `SELECT id FROM products WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NULL`, ws.ID, slug).Scan(&id); err != nil {
		return store.Product{}, err
	}
	return s.Q.ProductAny(ctx, id)
}

func (s *Service) publicOf(ctx context.Context, ws *store.Workspace, rows []store.Product, drafts bool) (map[uuid.UUID]PublicProduct, error) {
	out := map[uuid.UUID]PublicProduct{}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	files, err := s.filesByProduct(ctx, ids)
	if err != nil {
		return nil, err
	}
	sel, _ := s.Merchant(ctx, ws)
	for _, r := range rows {
		live, ok := liveOf(r)
		published := ok && r.Status == StatusPublished
		if !published {
			if !drafts || (r.Status == StatusArchived && !ok) {
				continue
			}
			if !ok || r.Status == StatusDraft {
				live = draftOf(r)
			}
		}
		cur := r.Currency
		if cur == "" {
			cur = sel.Currency
		}
		if cur == "" {
			cur = "usd"
		}
		p := PublicProduct{ID: r.ID, Status: r.Status, Live: live, Files: summary(files[r.ID]), Currency: cur, Published: published}
		if live.SalesPageID != "" {
			if pid, err := uuid.Parse(live.SalesPageID); err == nil {
				var slug string
				if err := s.Pool.QueryRow(ctx, `SELECT live_slug FROM pages WHERE id = $1 AND workspace_id = $2 AND status = 'published' AND deleted_at IS NULL`, pid, ws.ID).Scan(&slug); err == nil {
					p.SalesPage = slug
				}
			}
		}
		if live.OfferOn && live.OfferListID != "" {
			if lid, err := uuid.Parse(live.OfferListID); err == nil {
				if l, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: lid}); err == nil {
					p.OfferList = l.Name
					p.OfferNote = firstSentence(l.Description)
				}
			}
		}
		out[r.ID] = p
	}
	return out, nil
}

// firstSentence is a description's first sentence ("One email a week.").
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, ".!?"); i >= 0 {
		return s[:i+1]
	}
	return s
}

// FileText is how pages name a product's files ("PDF, 86 pages",
// "ZIP, 38 MB").
func FileText(f ProductFilesSummary) string {
	switch {
	case f.Count == 0:
		return ""
	case f.Count > 1:
		return f.Kind + ", " + humanSize(f.Bytes)
	case f.Pages > 0:
		return f.Kind + ", " + plural(f.Pages, "page")
	}
	return f.Kind + ", " + humanSize(f.Bytes)
}

// FileTextLong adds the size ("PDF, 86 pages, 4.2 MB").
func FileTextLong(f ProductFilesSummary) string {
	if f.Count == 1 && f.Pages > 0 {
		return f.Kind + ", " + plural(f.Pages, "page") + ", " + humanSize(f.Bytes)
	}
	return FileText(f)
}

// HumanSize writes a file size ("4.2 MB", "380 KB").
func HumanSize(n int64) string { return humanSize(n) }

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return itoa(n) + " " + word + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// KindLabel names a file's kind ("PDF", "ZIP").
func KindLabel(kind, name string) string { return kindLabel(kind, name) }
