package products

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// ProductPrice is a product's price, in the currency's minor units.
type ProductPrice struct {
	Kind           string `json:"kind" enum:"one_time,pwyw"`
	AmountCents    int64  `json:"amountCents" minimum:"0" maximum:"100000000"`
	MinimumCents   int64  `json:"minimumCents" minimum:"0" maximum:"100000000"`
	SuggestedCents int64  `json:"suggestedCents" minimum:"0" maximum:"100000000"`
}

// ProductOffer is the list offer next to the Buy button.
type ProductOffer struct {
	On     bool   `json:"on"`
	ListID string `json:"listId" doc:"The list a buyer who ticks it joins, with its double opt-in"`
	Text   string `json:"text" maxLength:"160" doc:"The exact text, kept as consent"`
}

// ProductDelivery is how buyers get the file.
type ProductDelivery struct {
	TemplateID     string `json:"templateId" doc:"The transactional template of the delivery email"`
	LinkDays       int32  `json:"linkDays" minimum:"1" maximum:"30"`
	Downloads      int32  `json:"downloads" minimum:"1" maximum:"20"`
	RevokeOnRefund bool   `json:"revokeOnRefund" doc:"Revoke access when a payment is refunded or disputed"`
}

// ProductFields are a product's editable fields: the draft, or what's live.
type ProductFields struct {
	Name             string          `json:"name" maxLength:"120"`
	Slug             string          `json:"slug" maxLength:"80"`
	ShortDescription string          `json:"shortDescription" maxLength:"120"`
	LongDescription  string          `json:"longDescription" maxLength:"4000"`
	CoverURL         string          `json:"coverUrl" maxLength:"1000"`
	CoverAlt         string          `json:"coverAlt" maxLength:"300"`
	Price            ProductPrice    `json:"price"`
	SalesPageID      string          `json:"salesPageId" doc:"Empty for the default product page"`
	ShowInShop       bool            `json:"showInShop"`
	Offer            ProductOffer    `json:"offer"`
	Delivery         ProductDelivery `json:"delivery"`
}

func fieldsOf(l Live) ProductFields {
	return ProductFields{Name: l.Name, Slug: l.Slug, ShortDescription: l.ShortDesc, LongDescription: l.LongDesc, CoverURL: l.CoverURL, CoverAlt: l.CoverAlt,
		Price:       ProductPrice{Kind: l.PriceKind, AmountCents: l.PriceCents, MinimumCents: l.MinCents, SuggestedCents: l.SuggestedCents},
		SalesPageID: l.SalesPageID, ShowInShop: l.ShowInShop, Offer: ProductOffer{On: l.OfferOn, ListID: l.OfferListID, Text: l.OfferText},
		Delivery: ProductDelivery{TemplateID: l.DeliveryTemplate, LinkDays: l.LinkDays, Downloads: l.LinkDownloads, RevokeOnRefund: l.RevokeOnRefund}}
}

// ProductFilesSummary is what a product's files add up to ("PDF, 86 pages, 4.2 MB").
type ProductFilesSummary struct {
	Count int    `json:"count"`
	Kind  string `json:"kind" doc:"The first file's kind in capitals (PDF, ZIP, XLSX)"`
	Pages int    `json:"pages" doc:"The first file's pages, for a PDF"`
	Bytes int64  `json:"bytes"`
}

// ProductSalesPage is where a product's Buy buttons lead.
type ProductSalesPage struct {
	PageID      string     `json:"pageId,omitempty" doc:"Empty for the default product page"`
	Name        string     `json:"name"`
	Path        string     `json:"path" example:"/field-guide"`
	Status      string     `json:"status" enum:"draft,scheduled,published,unpublished,default"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}

// ProductListItem is one row of Earn › Products.
type ProductListItem struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	ShortName string              `json:"shortName"`
	CoverURL  string              `json:"coverUrl"`
	CoverAlt  string              `json:"coverAlt"`
	Files     ProductFilesSummary `json:"files"`
	Price     ProductPrice        `json:"price"`
	Currency  string              `json:"currency"`
	Status    string              `json:"status" enum:"draft,published,archived"`
	// StatusAt and StatusBy say when and by whom it was published or
	// archived; a draft names who edited it last.
	StatusAt      *time.Time       `json:"statusAt,omitempty"`
	StatusBy      string           `json:"statusBy"`
	Orders        int64            `json:"orders"`
	GrossCents    int64            `json:"grossCents"`
	Refunds       int64            `json:"refunds"`
	RefundedCents int64            `json:"refundedCents"`
	SalesPage     ProductSalesPage `json:"salesPage"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

// ProductCount is a product's count ("214 Field Guide").
type ProductCount struct {
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Count     int64  `json:"count"`
}

// ProductsStats are the period's numbers (live payments, or test ones in
// test mode).
type ProductsStats struct {
	Orders         int64          `json:"orders"`
	ByProduct      []ProductCount `json:"byProduct"`
	GrossCents     int64          `json:"grossCents"`
	RefundedCents  int64          `json:"refundedCents"`
	RefundedOrders int64          `json:"refundedOrders"`
	OpenDisputes   int64          `json:"openDisputes"`
	NetCents       int64          `json:"netCents"`
}

// ProductSourceRow is where some sales came from: a product's sales page,
// the shop or its default page.
type ProductSourceRow struct {
	ProductID  string `json:"productId"`
	Product    string `json:"product"`
	Source     string `json:"source" enum:"page,shop,product_page"`
	Orders     int64  `json:"orders"`
	GrossCents int64  `json:"grossCents"`
}

// ProductsBuyers are buyers as contacts: the best-selling product's, and
// the list offers ticked per product.
type ProductsBuyers struct {
	Product   string         `json:"product"`
	Contacts  int64          `json:"contacts" doc:"Buyers who were contacts already"`
	Joined    int64          `json:"joined" doc:"Buyers who ticked the list offer and became contacts"`
	None      int64          `json:"none" doc:"Buyers who aren't contacts"`
	Ticked    int64          `json:"ticked"`
	TickedBy  []ProductCount `json:"tickedBy"`
	OfferText string         `json:"offerText"`
}

// ProductsPayments is the payment connection products sell through.
type ProductsPayments struct {
	Connected        bool   `json:"connected"`
	Provider         string `json:"provider"`
	Mode             string `json:"mode" enum:"live,test,"`
	Account          string `json:"account"`
	HasLive          bool   `json:"hasLive"`
	Currency         string `json:"currency"`
	TaxAtCheckout    bool   `json:"taxAtCheckout"`
	PricesIncludeTax bool   `json:"pricesIncludeTax"`
}

// ProductsHelpers name who can help with what a member can't do.
type ProductsHelpers struct {
	Edit    []string `json:"edit"`
	Publish []string `json:"publish"`
	Orders  []string `json:"orders"`
	Export  []string `json:"export"`
}

// ProductsOverview is Earn › Products.
type ProductsOverview struct {
	Period   string             `json:"period" enum:"30d,90d,12m"`
	From     time.Time          `json:"from"`
	Until    time.Time          `json:"until"`
	Stats    ProductsStats      `json:"stats"`
	Products []ProductListItem  `json:"products"`
	Sources  []ProductSourceRow `json:"sources"`
	Buyers   ProductsBuyers     `json:"buyers"`
	Payments ProductsPayments   `json:"payments"`
	Helpers  ProductsHelpers    `json:"helpers"`
}

// periodOf is a period's start.
func periodOf(period string, now time.Time) time.Time {
	switch period {
	case "90d":
		return now.AddDate(0, 0, -90)
	case "12m":
		return now.AddDate(-1, 0, 0)
	}
	return now.AddDate(0, 0, -30)
}

// payments reads the connection products sell through.
func (s *Service) payments(ctx context.Context, ws *store.Workspace) (ProductsPayments, earn.SellingConnection, error) {
	if s.Earn == nil || s.Earn() == nil {
		return ProductsPayments{}, earn.SellingConnection{}, nil
	}
	sel, ok, err := s.Earn().Selling(ctx, ws)
	if err != nil || !ok {
		return ProductsPayments{}, sel, err
	}
	return ProductsPayments{Connected: true, Provider: sel.Provider, Mode: string(sel.Mode), Account: sel.Account, HasLive: sel.HasLive,
		Currency: sel.Currency, TaxAtCheckout: sel.TaxAtCheckout, PricesIncludeTax: sel.PricesIncludeTax}, sel, nil
}

// numbersMode is the mode the numbers count: live, or test while only a
// test key is connected.
func numbersMode(pay ProductsPayments) string {
	if pay.Connected && pay.Mode == string(payments.Test) && !pay.HasLive {
		return string(payments.Test)
	}
	return string(payments.Live)
}

// Overview is Earn › Products for a period.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, period string) (ProductsOverview, error) {
	now := s.now()
	if period == "" {
		period = "30d"
	}
	since := periodOf(period, now)
	out := ProductsOverview{Period: period, From: since, Until: now, Products: []ProductListItem{}, Sources: []ProductSourceRow{},
		Stats: ProductsStats{ByProduct: []ProductCount{}}, Buyers: ProductsBuyers{TickedBy: []ProductCount{}}}
	pay, _, err := s.payments(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Payments = pay
	rows, err := s.Q.ProductsList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	mode := numbersMode(pay)
	stats, err := s.Q.ProductOrderStats(ctx, store.ProductOrderStatsParams{WorkspaceID: ws.ID, Mode: mode, Since: since, Until: now.Add(time.Second)})
	if err != nil {
		return out, err
	}
	byProduct := map[uuid.UUID]store.ProductOrderStatsRow{}
	for _, st := range stats {
		byProduct[st.ProductID] = st
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.Product.ID
	}
	files, err := s.filesByProduct(ctx, ids)
	if err != nil {
		return out, err
	}
	pages, err := s.salesPages(ctx, ws)
	if err != nil {
		return out, err
	}
	names := map[uuid.UUID]string{}
	for _, r := range rows {
		item := s.listItem(ws, r.Product, r.PublishedByName, r.EditedByName, files[r.Product.ID], pages)
		if st, ok := byProduct[r.Product.ID]; ok {
			item.Orders, item.GrossCents, item.Refunds, item.RefundedCents = st.Orders, st.Gross, st.Refunds, st.Refunded
		}
		if item.Currency == "" {
			item.Currency = pay.Currency
		}
		names[r.Product.ID] = item.ShortName
		out.Products = append(out.Products, item)
	}
	// Numbers: every order of the period, products deleted since included.
	for _, st := range stats {
		out.Stats.Orders += st.Orders
		out.Stats.GrossCents += st.Gross
		out.Stats.RefundedCents += st.Refunded
		out.Stats.RefundedOrders += st.Refunds
		out.Stats.OpenDisputes += st.Disputes
		if n, ok := names[st.ProductID]; ok && st.Orders > 0 {
			out.Stats.ByProduct = append(out.Stats.ByProduct, ProductCount{ProductID: st.ProductID.String(), Name: n, Count: st.Orders})
		}
	}
	slices.SortStableFunc(out.Stats.ByProduct, func(a, b ProductCount) int { return int(b.Count - a.Count) })
	out.Stats.NetCents = out.Stats.GrossCents - out.Stats.RefundedCents

	src, err := s.Q.ProductOrderSources(ctx, store.ProductOrderSourcesParams{WorkspaceID: ws.ID, Mode: mode, Since: since, Until: now.Add(time.Second)})
	if err != nil {
		return out, err
	}
	order := map[string]int{SourcePage: 0, SourceShop: 1, SourceProductPage: 2}
	for _, r := range src {
		n, ok := names[r.ProductID]
		if !ok {
			continue
		}
		out.Sources = append(out.Sources, ProductSourceRow{ProductID: r.ProductID.String(), Product: n, Source: r.Source, Orders: r.Orders, GrossCents: r.Gross})
	}
	rank := map[string]int{}
	for i, p := range out.Products {
		rank[p.ID] = i
	}
	slices.SortStableFunc(out.Sources, func(a, b ProductSourceRow) int {
		if rank[a.ProductID] != rank[b.ProductID] {
			return rank[a.ProductID] - rank[b.ProductID]
		}
		return order[a.Source] - order[b.Source]
	})

	buyers, err := s.Q.ProductOrderBuyers(ctx, store.ProductOrderBuyersParams{WorkspaceID: ws.ID, Mode: mode, Since: since, Until: now.Add(time.Second)})
	if err != nil {
		return out, err
	}
	if len(out.Stats.ByProduct) > 0 {
		top, _ := uuid.Parse(out.Stats.ByProduct[0].ProductID)
		out.Buyers.Product = out.Stats.ByProduct[0].Name
		ticked := map[uuid.UUID]int64{}
		for _, b := range buyers {
			ticked[b.ProductID] += b.Ticked
			out.Buyers.Ticked += b.Ticked
			if b.ProductID != top {
				continue
			}
			switch b.Buyer {
			case "contact":
				out.Buyers.Contacts += b.Orders
			case "joined":
				out.Buyers.Joined += b.Orders
			default:
				out.Buyers.None += b.Orders
			}
		}
		for _, pc := range out.Stats.ByProduct {
			id, _ := uuid.Parse(pc.ProductID)
			if n := ticked[id]; n > 0 {
				out.Buyers.TickedBy = append(out.Buyers.TickedBy, ProductCount{ProductID: pc.ProductID, Name: pc.Name, Count: n})
			}
		}
		for _, r := range rows {
			if r.Product.ID == top {
				if l, ok := liveOf(r.Product); ok {
					out.Buyers.OfferText = l.OfferText
				} else {
					out.Buyers.OfferText = r.Product.OfferText
				}
			}
		}
	}
	out.Helpers = s.helpersAll(ctx, ws)
	return out, nil
}

// listItem is a product's row.
func (s *Service) listItem(ws *store.Workspace, p store.Product, publishedBy, editedBy string, files []store.DownloadFile, pages map[uuid.UUID]pageRef) ProductListItem {
	l, live := liveOf(p)
	if !live {
		l = draftOf(p)
	}
	item := ProductListItem{ID: p.ID.String(), Name: l.Name, ShortName: ShortName(l.Name), CoverURL: l.CoverURL, CoverAlt: l.CoverAlt, Files: summary(files),
		Price:    ProductPrice{Kind: l.PriceKind, AmountCents: l.PriceCents, MinimumCents: l.MinCents, SuggestedCents: l.SuggestedCents},
		Currency: p.Currency, Status: p.Status, StatusBy: editedBy, UpdatedAt: p.UpdatedAt}
	switch p.Status {
	case StatusPublished:
		item.StatusAt, item.StatusBy = p.PublishedAt, publishedBy
	case StatusArchived:
		item.StatusAt = p.ArchivedAt
	}
	item.SalesPage = s.salesPage(ws, l, pages)
	return item
}

// summary adds up a product's files: the first one's kind and pages, every
// file's bytes.
func summary(files []store.DownloadFile) ProductFilesSummary {
	out := ProductFilesSummary{Count: len(files)}
	for i, f := range files {
		if i == 0 {
			out.Kind = kindLabel(f.Kind, f.Name)
			if f.Pages != nil {
				out.Pages = int(*f.Pages)
			}
		}
		out.Bytes += f.SizeBytes
	}
	return out
}

func (s *Service) filesByProduct(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]store.DownloadFile, error) {
	out := map[uuid.UUID][]store.DownloadFile{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Q.DownloadFilesOfOwners(ctx, store.DownloadFilesOfOwnersParams{OwnerKind: "product", OwnerIds: ids})
	if err != nil {
		return nil, err
	}
	for _, f := range rows {
		out[f.OwnerID] = append(out[f.OwnerID], f)
	}
	return out, nil
}

// pageRef is a page a product can sell on.
type pageRef struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Status      string
	PublishedAt *time.Time
}

func (s *Service) salesPages(ctx context.Context, ws *store.Workspace) (map[uuid.UUID]pageRef, error) {
	rows, err := s.Q.PagesList(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]pageRef{}
	for _, row := range rows {
		r := row.Page
		slug := r.Slug
		if r.LiveSlug != nil && r.Status == "published" {
			slug = *r.LiveSlug
		}
		out[r.ID] = pageRef{ID: r.ID, Name: r.Name, Slug: slug, Status: r.Status, PublishedAt: r.PublishedAt}
	}
	return out, nil
}

// salesPage is where a product's Buy buttons lead.
func (s *Service) salesPage(ws *store.Workspace, l Live, pages map[uuid.UUID]pageRef) ProductSalesPage {
	if id, err := uuid.Parse(l.SalesPageID); err == nil {
		if pg, ok := pages[id]; ok {
			return ProductSalesPage{PageID: pg.ID.String(), Name: pg.Name, Path: "/" + pg.Slug, Status: pg.Status, PublishedAt: pg.PublishedAt}
		}
	}
	return ProductSalesPage{Name: l.Name, Path: "/products/" + l.Slug, Status: "default"}
}

// helpersAll names who can edit, publish, manage orders and export.
func (s *Service) helpersAll(ctx context.Context, ws *store.Workspace) ProductsHelpers {
	return ProductsHelpers{Edit: s.helpers(ctx, ws, authz.ProductsEdit), Publish: s.helpers(ctx, ws, authz.ProductsPublish),
		Orders: s.helpers(ctx, ws, authz.OrdersManage), Export: s.helpers(ctx, ws, authz.SubscribersExport)}
}

// helpers names up to two Admins or Owners who hold perm, Admins first, as
// "Ask Marcus Chen or Linh Nguyen" reads on the boards.
func (s *Service) helpers(ctx context.Context, ws *store.Workspace, perm authz.Permission) []string {
	out := []string{}
	rows, err := s.Q.ListMembersWithRoles(ctx, ws.ID)
	if err != nil {
		return out
	}
	rank := func(key string) int {
		switch key {
		case authz.RoleAdmin:
			return 0
		case authz.RoleOwner:
			return 1
		}
		return 2
	}
	slices.SortStableFunc(rows, func(a, b store.ListMembersWithRolesRow) int { return rank(a.RoleKey) - rank(b.RoleKey) })
	for _, r := range rows {
		if rank(r.RoleKey) < 2 && authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			out = append(out, r.Name)
			if len(out) == 2 {
				break
			}
		}
	}
	return out
}

// ProductFile is one of a product's files.
type ProductFile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind" doc:"What the content is: pdf, zip, xlsx…, empty when it couldn't be told"`
	KindLabel   string    `json:"kindLabel" doc:"PDF, ZIP, XLSX"`
	ContentType string    `json:"contentType"`
	Pages       int       `json:"pages"`
	SizeBytes   int64     `json:"sizeBytes"`
	UploadedAt  time.Time `json:"uploadedAt"`
	UploadedBy  string    `json:"uploadedBy"`
}

func fileOf(f store.DownloadFile, by string) ProductFile {
	out := ProductFile{ID: f.ID.String(), Name: f.Name, Kind: f.Kind, KindLabel: kindLabel(f.Kind, f.Name), ContentType: f.ContentType, SizeBytes: f.SizeBytes,
		UploadedAt: f.UploadedAt, UploadedBy: by}
	if f.Pages != nil {
		out.Pages = int(*f.Pages)
	}
	return out
}

// ProductCheck is one publish check.
type ProductCheck struct {
	Key string `json:"key" enum:"stripe,file,price,template,route,cover_alt,sales_page"`
	OK  bool   `json:"ok"`
	// What the check found: the account and mode, the file, the price,
	// the template and its version, the route, the sales page.
	Account  string     `json:"account,omitempty"`
	Mode     string     `json:"mode,omitempty"`
	File     string     `json:"file,omitempty"`
	Bytes    int64      `json:"bytes,omitempty"`
	Template string     `json:"template,omitempty"`
	Version  int32      `json:"version,omitempty"`
	Draft    bool       `json:"draft,omitempty" doc:"The template has a draft but no published version"`
	Route    string     `json:"route,omitempty" example:"Postmark, then Amazon SES"`
	Path     string     `json:"path,omitempty"`
	At       *time.Time `json:"at,omitempty"`
	Default  bool       `json:"default,omitempty" doc:"No sales page chosen: the default product page"`
	// TemplateID links the fix ("Open product-delivery and publish it").
	TemplateID string `json:"templateId,omitempty"`
}

// ProductListOption is a list the offer can use.
type ProductListOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DoubleOptIn bool   `json:"doubleOptIn"`
}

// ProductPageOption is a page a product can sell on.
type ProductPageOption struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Status string `json:"status" enum:"draft,scheduled,published,unpublished"`
}

// ProductTemplateOption is a transactional template a product can deliver with.
type ProductTemplateOption struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Version  int32  `json:"version" doc:"The live version; 0 when none is published"`
	Category string `json:"category"`
	Kind     string `json:"kind" doc:"essential or optional"`
}

// ProductNumbers are a published product's numbers since it was first published.
type ProductNumbers struct {
	Since         time.Time `json:"since"`
	Orders        int64     `json:"orders"`
	GrossCents    int64     `json:"grossCents"`
	RefundedCents int64     `json:"refundedCents"`
	Refunds       int64     `json:"refunds"`
	FromPage      int64     `json:"fromPage"`
	FromShop      int64     `json:"fromShop"`
	FromDefault   int64     `json:"fromDefault"`
	Contacts      int64     `json:"contacts"`
	Joined        int64     `json:"joined"`
	None          int64     `json:"none"`
	// OfferList names the list buyers joined at checkout.
	OfferList  string                    `json:"offerList"`
	Automation *ProductAutomationNumbers `json:"automation,omitempty"`
	Test       bool                      `json:"test"`
}

// ProductAutomationNumbers is the "Bought a product" automation's numbers.
type ProductAutomationNumbers struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Entered  int64  `json:"entered" doc:"Contacts still in the flow or through it"`
	Contacts int64  `json:"contacts"`
	Refunded int64  `json:"refunded" doc:"Contacts who left because their payment was refunded"`
}

// ProductDetail is the product editor.
type ProductDetail struct {
	ID          string         `json:"id"`
	Status      string         `json:"status" enum:"draft,published,archived"`
	Draft       ProductFields  `json:"draft"`
	Live        *ProductFields `json:"live,omitempty"`
	Changes     []string       `json:"changes" doc:"The draft's fields that differ from what's live"`
	PublishedAt *time.Time     `json:"publishedAt,omitempty"`
	PublishedBy string         `json:"publishedBy"`
	ArchivedAt  *time.Time     `json:"archivedAt,omitempty"`
	CreatedBy   string         `json:"createdBy"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	UpdatedBy   string         `json:"updatedBy"`
	Files       []ProductFile  `json:"files"`
	FilesBytes  int64          `json:"filesBytes"`
	MaxBytes    int64          `json:"maxBytes"`
	MaxFile     int64          `json:"maxFile"`
	ChunkBytes  int64          `json:"chunkBytes"`
	Checks      []ProductCheck `json:"checks"`
	// Numbers since the product was first published.
	Numbers   *ProductNumbers         `json:"numbers,omitempty"`
	Pages     []ProductPageOption     `json:"pages"`
	Lists     []ProductListOption     `json:"lists"`
	Templates []ProductTemplateOption `json:"templates"`
	Payments  ProductsPayments        `json:"payments"`
	Route     string                  `json:"route"`
	// Addresses: the site's products path ("news.acme.io/products/"),
	// the shop and the default product page.
	SiteHost  string           `json:"siteHost"`
	ShopURL   string           `json:"shopUrl"`
	PageURL   string           `json:"pageUrl"`
	SalesPage ProductSalesPage `json:"salesPage"`
	// Orders counts every live order (the price change dialog).
	Orders  int64           `json:"orders"`
	Helpers ProductsHelpers `json:"helpers"`
}

// Detail is one product for its editor.
func (s *Service) Detail(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ProductDetail, error) {
	p, err := s.Q.ProductGet(ctx, store.ProductGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ProductDetail{}, api.MapError(err, "Product")
	}
	return s.detail(ctx, ws, p)
}

func (s *Service) detail(ctx context.Context, ws *store.Workspace, p store.Product) (ProductDetail, error) {
	draft := draftOf(p)
	out := ProductDetail{ID: p.ID.String(), Status: p.Status, Draft: fieldsOf(draft), Changes: []string{}, PublishedAt: p.PublishedAt, ArchivedAt: p.ArchivedAt,
		UpdatedAt: p.UpdatedAt, Files: []ProductFile{}, MaxBytes: MaxProductBytes, MaxFile: MaxFileBytes, ChunkBytes: ChunkBytes, Pages: []ProductPageOption{},
		Lists: []ProductListOption{}, Templates: []ProductTemplateOption{}}
	if l, ok := liveOf(p); ok {
		f := fieldsOf(l)
		out.Live = &f
		out.Changes = changes(l, draft)
	}
	var names struct{ published, created, updated string }
	_ = s.Pool.QueryRow(ctx, `SELECT coalesce((SELECT name FROM users WHERE id = $1), ''), coalesce((SELECT name FROM users WHERE id = $2), ''),
		coalesce((SELECT name FROM users WHERE id = $3), '')`, p.PublishedBy, p.CreatedBy, p.UpdatedBy).Scan(&names.published, &names.created, &names.updated)
	out.PublishedBy, out.CreatedBy, out.UpdatedBy = names.published, names.created, names.updated

	files, err := s.Q.DownloadFilesOf(ctx, store.DownloadFilesOfParams{OwnerKind: "product", OwnerID: p.ID})
	if err != nil {
		return out, err
	}
	for _, f := range files {
		out.Files = append(out.Files, fileOf(f.DownloadFile, f.UploadedByName))
		out.FilesBytes += f.DownloadFile.SizeBytes
	}
	pay, _, err := s.payments(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Payments = pay
	if out.Payments.Currency == "" {
		out.Payments.Currency = "usd"
	}
	pages, err := s.salesPages(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, pg := range pages {
		out.Pages = append(out.Pages, ProductPageOption{ID: pg.ID.String(), Name: pg.Name, Path: "/" + pg.Slug, Status: pg.Status})
	}
	slices.SortFunc(out.Pages, func(a, b ProductPageOption) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	lists, err := s.Q.ProductListOptions(ctx, ws.ID)
	if err == nil {
		for _, l := range lists {
			out.Lists = append(out.Lists, ProductListOption{ID: l.ID.String(), Name: l.Name, DoubleOptIn: l.DoubleOptIn})
		}
	}
	tpls, err := s.templateOptions(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Templates = tpls
	route, routeOK, err := s.route(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Route = route
	out.Checks = s.checks(ctx, ws, p, draft, files, pay, tpls, route, routeOK, pages)
	out.SalesPage = s.salesPage(ws, draft, pages)
	links := s.siteLinks(ctx, ws)
	out.SiteHost, out.ShopURL, out.PageURL = links.host, links.shop, links.product(draft.Slug)
	if p.FirstPublishedAt != nil {
		n, err := s.numbers(ctx, ws, p, pay)
		if err != nil {
			return out, err
		}
		out.Numbers = &n
		out.Orders = n.Orders
	}
	out.Helpers = s.helpersAll(ctx, ws)
	return out, nil
}

// changes names the draft's fields that differ from what's live.
func changes(live, draft Live) []string {
	out := []string{}
	add := func(changed bool, key string) {
		if changed {
			out = append(out, key)
		}
	}
	add(live.Name != draft.Name, "name")
	add(live.Slug != draft.Slug, "address")
	add(live.ShortDesc != draft.ShortDesc, "shortDescription")
	add(live.LongDesc != draft.LongDesc, "longDescription")
	add(live.CoverURL != draft.CoverURL, "cover")
	add(live.CoverAlt != draft.CoverAlt, "coverAlt")
	add(live.PriceKind != draft.PriceKind || live.PriceCents != draft.PriceCents || live.MinCents != draft.MinCents || live.SuggestedCents != draft.SuggestedCents, "price")
	add(live.SalesPageID != draft.SalesPageID, "salesPage")
	add(live.ShowInShop != draft.ShowInShop, "shop")
	add(live.OfferOn != draft.OfferOn || live.OfferListID != draft.OfferListID || live.OfferText != draft.OfferText, "offer")
	add(live.DeliveryTemplate != draft.DeliveryTemplate || live.LinkDays != draft.LinkDays || live.LinkDownloads != draft.LinkDownloads ||
		live.RevokeOnRefund != draft.RevokeOnRefund, "delivery")
	return out
}

// priceChanged reports whether publishing the draft changes what buyers pay.
func priceChanged(live, draft Live) bool {
	return live.PriceKind != draft.PriceKind || live.PriceCents != draft.PriceCents || live.MinCents != draft.MinCents || live.SuggestedCents != draft.SuggestedCents
}

// templateOptions are the workspace's transactional templates.
func (s *Service) templateOptions(ctx context.Context, ws *store.Workspace) ([]ProductTemplateOption, error) {
	out := []ProductTemplateOption{}
	rows, err := s.Q.ProductDeliveryTemplates(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out = append(out, ProductTemplateOption{ID: r.ID.String(), Key: r.Key, Name: r.Name, Version: r.Version, Category: r.Category, Kind: r.Kind})
	}
	return out, nil
}

// route is the Transactional route in words ("Postmark, then Amazon SES").
func (s *Service) route(ctx context.Context, ws *store.Workspace) (string, bool, error) {
	set, err := routing.Load(ctx, s.Q, ws.ID)
	if err != nil {
		return "", false, err
	}
	def, ok := set.Default(routing.StreamTransactional)
	if !ok || len(def.Splits) == 0 {
		return "", false, nil
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return "", false, err
	}
	name := func(id uuid.UUID) string {
		for _, p := range provs {
			if p.ID == id {
				return p.Name
			}
		}
		return ""
	}
	top := def.Splits[0]
	for _, sp := range def.Splits[1:] {
		if sp.Weight > top.Weight {
			top = sp
		}
	}
	primary := name(top.ProviderID)
	if primary == "" {
		return "", false, nil
	}
	if def.Fallback != nil {
		if fb := name(*def.Fallback); fb != "" {
			return primary + ", then " + fb, true, nil
		}
	}
	return primary, true, nil
}

// checks are a product's publish checks, on its draft.
func (s *Service) checks(ctx context.Context, ws *store.Workspace, p store.Product, d Live, files []store.DownloadFilesOfRow, pay ProductsPayments,
	tpls []ProductTemplateOption, route string, routeOK bool, pages map[uuid.UUID]pageRef) []ProductCheck {
	out := []ProductCheck{
		{Key: "stripe", OK: pay.Connected, Account: pay.Account, Mode: pay.Mode},
	}
	fc := ProductCheck{Key: "file", OK: len(files) > 0}
	if len(files) > 0 {
		fc.File, fc.Bytes = files[0].DownloadFile.Name, files[0].DownloadFile.SizeBytes
	}
	out = append(out, fc)
	priced := d.PriceCents >= MinimumCents
	if d.PriceKind == PricePWYW {
		priced = d.MinCents >= MinimumCents
	}
	out = append(out, ProductCheck{Key: "price", OK: priced})
	tc := ProductCheck{Key: "template"}
	for _, t := range tpls {
		if t.ID == d.DeliveryTemplate {
			tc.Template, tc.Version, tc.TemplateID = t.Key, t.Version, t.ID
			tc.OK = t.Version > 0
			tc.Draft = t.Version == 0
		}
	}
	out = append(out, tc, ProductCheck{Key: "route", OK: routeOK, Route: route})
	out = append(out, ProductCheck{Key: "cover_alt", OK: d.CoverURL == "" || strings.TrimSpace(d.CoverAlt) != ""})
	sp := ProductCheck{Key: "sales_page", OK: true, Default: true}
	if id, err := uuid.Parse(d.SalesPageID); err == nil {
		sp.Default = false
		pg, ok := pages[id]
		sp.OK = ok && pg.Status == "published"
		if ok {
			sp.Path, sp.At = "/"+pg.Slug, pg.PublishedAt
		}
	}
	out = append(out, sp)
	return out
}

func checksPass(cs []ProductCheck) bool {
	for _, c := range cs {
		if !c.OK {
			return false
		}
	}
	return true
}

// numbers are a published product's numbers since it was first published.
func (s *Service) numbers(ctx context.Context, ws *store.Workspace, p store.Product, pay ProductsPayments) (ProductNumbers, error) {
	mode := numbersMode(pay)
	since := *p.FirstPublishedAt
	until := s.now().Add(time.Second)
	out := ProductNumbers{Since: since, Test: mode == string(payments.Test)}
	stats, err := s.Q.ProductOrderStats(ctx, store.ProductOrderStatsParams{WorkspaceID: ws.ID, Mode: mode, Since: since.Add(-24 * time.Hour), Until: until})
	if err != nil {
		return out, err
	}
	for _, st := range stats {
		if st.ProductID == p.ID {
			out.Orders, out.GrossCents, out.RefundedCents, out.Refunds = st.Orders, st.Gross, st.Refunded, st.Refunds
		}
	}
	src, err := s.Q.ProductOrderSources(ctx, store.ProductOrderSourcesParams{WorkspaceID: ws.ID, Mode: mode, Since: since.Add(-24 * time.Hour), Until: until})
	if err != nil {
		return out, err
	}
	for _, r := range src {
		if r.ProductID != p.ID {
			continue
		}
		switch r.Source {
		case SourcePage:
			out.FromPage += r.Orders
		case SourceShop:
			out.FromShop += r.Orders
		default:
			out.FromDefault += r.Orders
		}
	}
	buyers, err := s.Q.ProductOrderBuyers(ctx, store.ProductOrderBuyersParams{WorkspaceID: ws.ID, Mode: mode, Since: since.Add(-24 * time.Hour), Until: until})
	if err != nil {
		return out, err
	}
	for _, b := range buyers {
		if b.ProductID != p.ID {
			continue
		}
		switch b.Buyer {
		case "contact":
			out.Contacts += b.Orders
		case "joined":
			out.Joined += b.Orders
		default:
			out.None += b.Orders
		}
	}
	l, _ := liveOf(p)
	if id, err := uuid.Parse(l.OfferListID); err == nil {
		if lst, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id}); err == nil {
			out.OfferList = lst.Name
		}
	}
	a, err := s.automationNumbers(ctx, ws, p.ID)
	if err != nil {
		return out, err
	}
	out.Automation = a
	return out, nil
}

// automationNumbers are the numbers of a live "Bought a product" automation
// for this product (or any product).
func (s *Service) automationNumbers(ctx context.Context, ws *store.Workspace, productID uuid.UUID) (*ProductAutomationNumbers, error) {
	var a ProductAutomationNumbers
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		SELECT a.id, a.name,
		       (SELECT count(*) FROM automation_runs r WHERE r.automation_id = a.id)::bigint,
		       (SELECT count(*) FROM automation_runs r WHERE r.automation_id = a.id AND r.exit_reason = 'refunded')::bigint
		FROM automations a
		WHERE a.workspace_id = $1 AND a.status = 'live' AND a.deleted_at IS NULL
		  AND a.trigger->>'kind' = 'bought_product' AND (coalesce(a.trigger->>'productId', '') IN ('', $2))
		ORDER BY coalesce(a.trigger->>'productId', '') = $2 DESC, a.created_at
		LIMIT 1`, ws.ID, productID.String()).Scan(&id, &a.Name, &a.Contacts, &a.Refunded)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		// Automations whose schema is older or missing: no numbers.
		s.log().WarnContext(ctx, "products: automation numbers", "err", err)
		return nil, nil
	}
	a.ID = id.String()
	a.Entered = a.Contacts - a.Refunded
	return &a, nil
}

// siteLinks are the public site's product addresses.
type siteLinks struct {
	host string
	base string
	shop string
}

func (l siteLinks) product(slug string) string { return l.base + "/products/" + slug }

func (s *Service) siteLinks(ctx context.Context, ws *store.Workspace) siteLinks {
	base := strings.TrimRight(s.URLs.Base(), "/") + "/p/" + ws.Slug
	if s.Forms != nil {
		if set, err := s.Forms.LoadSettings(ctx, ws); err == nil {
			links := s.Forms.Links(ws, set)
			if links.Custom {
				base = strings.TrimRight(links.Origin(), "/")
			} else {
				base = strings.TrimRight(links.Path(), "/")
			}
		}
	}
	host := strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	return siteLinks{host: host + "/products/", base: base, shop: base + "/products"}
}

// ProductCreate starts a product.
type ProductCreate struct {
	Name string `json:"name,omitempty" maxLength:"120"`
}

// Create starts a draft product.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in ProductCreate) (ProductDetail, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = word(lang(ctx), "product.untitled", nil)
	}
	id := newID()
	now := s.now()
	// The delivery template comes first: a product without one can't publish.
	tpl, terr := s.ensureTemplate(ctx, ws)
	if terr != nil {
		s.log().WarnContext(ctx, "products: delivery template", "err", terr)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		slug, err := freeSlug(ctx, q, ws.ID, id, Slugify(name))
		if err != nil {
			return err
		}
		if err := q.ProductInsert(ctx, store.ProductInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Slug: slug, PriceKind: PriceOneTime,
			ShowInShop: true, DeliveryTemplateID: tpl, Actor: actor(p), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "product.created", TargetType: "product", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "slug": slug, "status": StatusDraft}})
	})
	if err != nil {
		return ProductDetail{}, err
	}
	return s.Detail(ctx, ws, id)
}

// freeSlug is base, or base-2, base-3… when it's taken.
func freeSlug(ctx context.Context, q *store.Queries, wsID, id uuid.UUID, base string) (string, error) {
	slug := base
	for i := 2; i < 1000; i++ {
		taken, err := q.ProductSlugTaken(ctx, store.ProductSlugTakenParams{WorkspaceID: wsID, ID: id, Slug: slug})
		if err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
		suffix := fmt.Sprintf("-%d", i)
		if len(base)+len(suffix) > 80 {
			base = strings.Trim(base[:80-len(suffix)], "-")
		}
		slug = base + suffix
	}
	return "", api.Conflict(word(lang(ctx), "product.slugExhausted", nil))
}

// Save saves a product's draft. A published product's live version
// changes only with Publish changes.
func (s *Service) Save(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ProductFields) (ProductDetail, error) {
	l := lang(ctx)
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(strings.ToLower(in.Slug))
	in.ShortDescription = strings.TrimSpace(in.ShortDescription)
	in.CoverAlt = strings.TrimSpace(in.CoverAlt)
	in.Offer.Text = strings.TrimSpace(in.Offer.Text)
	switch {
	case in.Name == "":
		return ProductDetail{}, api.Invalid("body.name", word(l, "product.nameRequired", nil))
	case !slugRe.MatchString(in.Slug):
		return ProductDetail{}, api.Invalid("body.slug", word(l, "product.slugInvalid", nil))
	case utf8.RuneCountInString(in.ShortDescription) > 120:
		return ProductDetail{}, api.Invalid("body.shortDescription", word(l, "product.shortLong", nil))
	}
	if in.Price.Kind == "" {
		in.Price.Kind = PriceOneTime
	}
	switch in.Price.Kind {
	case PriceOneTime:
		if in.Price.AmountCents > 0 && in.Price.AmountCents < MinimumCents {
			return ProductDetail{}, api.Invalid("body.price.amountCents", word(l, "product.priceMinimum", nil))
		}
	case PricePWYW:
		if in.Price.MinimumCents < MinimumCents {
			return ProductDetail{}, api.Invalid("body.price.minimumCents", word(l, "product.pwywMinimum", nil))
		}
		if in.Price.SuggestedCents > 0 && in.Price.SuggestedCents < in.Price.MinimumCents {
			return ProductDetail{}, api.Invalid("body.price.suggestedCents", word(l, "product.suggestedBelow", nil))
		}
	default:
		return ProductDetail{}, api.Invalid("body.price.kind", word(l, "product.priceKind", nil))
	}
	if in.Delivery.LinkDays == 0 {
		in.Delivery.LinkDays = 7
	}
	if in.Delivery.Downloads == 0 {
		in.Delivery.Downloads = 5
	}
	if in.CoverURL != "" {
		if !strings.HasPrefix(in.CoverURL, "/media/") && !strings.HasPrefix(in.CoverURL, "https://") {
			return ProductDetail{}, api.Invalid("body.coverUrl", word(l, "product.coverURL", nil))
		}
	}
	var salesPage, offerList, template *uuid.UUID
	if in.SalesPageID != "" {
		pid, err := uuid.Parse(in.SalesPageID)
		if err != nil {
			return ProductDetail{}, api.Invalid("body.salesPageId", word(l, "product.pageGone", nil))
		}
		if _, err := s.Q.PageGet(ctx, store.PageGetParams{WorkspaceID: ws.ID, ID: pid}); err != nil {
			return ProductDetail{}, api.Invalid("body.salesPageId", word(l, "product.pageGone", nil))
		}
		salesPage = &pid
	}
	if in.Offer.ListID != "" {
		lid, err := uuid.Parse(in.Offer.ListID)
		if err != nil {
			return ProductDetail{}, api.Invalid("body.offer.listId", word(l, "product.listGone", nil))
		}
		if _, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: lid}); err != nil {
			return ProductDetail{}, api.Invalid("body.offer.listId", word(l, "product.listGone", nil))
		}
		offerList = &lid
	}
	if in.Offer.On && (offerList == nil || in.Offer.Text == "") {
		return ProductDetail{}, api.Invalid("body.offer", word(l, "product.offerIncomplete", nil))
	}
	if in.Delivery.TemplateID != "" {
		tid, err := uuid.Parse(in.Delivery.TemplateID)
		if err != nil {
			return ProductDetail{}, api.Invalid("body.delivery.templateId", word(l, "product.templateGone", nil))
		}
		if _, err := s.Q.ProductTemplateExists(ctx, store.ProductTemplateExistsParams{WorkspaceID: ws.ID, ID: tid}); err != nil {
			return ProductDetail{}, api.Invalid("body.delivery.templateId", word(l, "product.templateGone", nil))
		}
		template = &tid
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ProductLock(ctx, store.ProductLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Product")
		}
		taken, err := q.ProductSlugTaken(ctx, store.ProductSlugTakenParams{WorkspaceID: ws.ID, ID: id, Slug: in.Slug})
		if err != nil {
			return err
		}
		if taken {
			sug, _ := freeSlug(ctx, q, ws.ID, id, in.Slug)
			return api.Err(409, "slug_taken", word(l, "product.slugTaken", map[string]any{"path": "/products/" + in.Slug})).WithMeta("suggestion", sug)
		}
		before := draftOf(cur)
		if err := q.ProductSave(ctx, store.ProductSaveParams{ID: id, Name: in.Name, Slug: in.Slug, ShortDesc: in.ShortDescription, LongDesc: strings.TrimSpace(in.LongDescription),
			CoverUrl: in.CoverURL, CoverAlt: in.CoverAlt, PriceKind: in.Price.Kind, PriceCents: in.Price.AmountCents, MinCents: in.Price.MinimumCents,
			SuggestedCents: in.Price.SuggestedCents, SalesPageID: salesPage, ShowInShop: in.ShowInShop, OfferOn: in.Offer.On, OfferListID: offerList,
			OfferText: in.Offer.Text, LinkDays: in.Delivery.LinkDays, LinkDownloads: in.Delivery.Downloads, DeliveryTemplateID: template,
			RevokeOnRefund: in.Delivery.RevokeOnRefund, At: s.now(), Actor: actor(p)}); err != nil {
			return err
		}
		after, err := q.ProductGet(ctx, store.ProductGetParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		diff := changes(before, draftOf(after))
		if len(diff) == 0 {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "product.updated", TargetType: "product", TargetID: id.String(), TargetName: in.Name,
			Before: before, After: draftOf(after), Metadata: map[string]any{"changed": diff, "draft": cur.Status != StatusDraft}})
	})
	if err != nil {
		return ProductDetail{}, err
	}
	return s.Detail(ctx, ws, id)
}

// ProductPublishInput publishes a product's draft.
type ProductPublishInput struct {
	// ConfirmPrice confirms a published product's new price: existing
	// buyers keep their access, new buyers pay the new price.
	ConfirmPrice bool `json:"confirmPrice,omitempty"`
}

// Publish makes the draft live: the price in Stripe, then the shop, its
// pages and checkout.
func (s *Service) Publish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ProductPublishInput) (ProductDetail, error) {
	l := lang(ctx)
	cur, err := s.Q.ProductGet(ctx, store.ProductGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ProductDetail{}, api.MapError(err, "Product")
	}
	det, err := s.detail(ctx, ws, cur)
	if err != nil {
		return ProductDetail{}, err
	}
	if !checksPass(det.Checks) {
		return ProductDetail{}, api.Err(409, "checks_failed", word(l, "product.checksFailed", nil))
	}
	draft := draftOf(cur)
	live, wasLive := liveOf(cur)
	if wasLive && priceChanged(live, draft) && !in.ConfirmPrice {
		return ProductDetail{}, api.Err(409, "price_change", word(l, "product.priceChangeConfirm", nil)).WithMeta("orders", det.Orders)
	}
	if wasLive && len(changes(live, draft)) == 0 && cur.Status == StatusPublished {
		return det, nil
	}
	sel, ok, err := s.Earn().Selling(ctx, ws)
	if err != nil {
		return ProductDetail{}, err
	}
	if !ok {
		return ProductDetail{}, api.Err(409, "stripe_not_connected", word(l, "product.stripeMissing", nil))
	}
	currency := cur.Currency
	if currency == "" {
		currency = sel.Currency
	}
	ids, err := s.syncStripe(ctx, ws, cur, draft, sel, currency)
	if err != nil {
		return ProductDetail{}, err
	}
	snap, _ := json.Marshal(draft)
	stripeJSON, _ := json.Marshal(ids)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.ProductLock(ctx, store.ProductLockParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return api.MapError(err, "Product")
		}
		if err := q.ProductPublish(ctx, store.ProductPublishParams{ID: id, Live: snap, LiveSlug: &draft.Slug, Currency: currency, Stripe: stripeJSON,
			At: now, Actor: actor(p)}); err != nil {
			return err
		}
		if wasLive && cur.LiveSlug != nil && *cur.LiveSlug != draft.Slug {
			if err := q.ProductRedirectInsert(ctx, store.ProductRedirectInsertParams{WorkspaceID: ws.ID, Slug: *cur.LiveSlug, ProductID: id, At: now}); err != nil {
				return err
			}
		}
		_ = q.ProductRedirectDelete(ctx, store.ProductRedirectDeleteParams{WorkspaceID: ws.ID, Slug: draft.Slug})
		action := "product.published"
		var before any
		if wasLive {
			action, before = "product.changes_published", live
		}
		meta := map[string]any{"mode": string(sel.Mode)}
		if wasLive {
			meta["changed"] = changes(live, draft)
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "product", TargetID: id.String(), TargetName: draft.Name,
			Before: before, After: draft, Metadata: meta}); err != nil {
			return err
		}
		if wasLive && priceChanged(live, draft) {
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "product.price_changed", TargetType: "product", TargetID: id.String(),
				TargetName: draft.Name, Before: map[string]any{"kind": live.PriceKind, "amountCents": live.PriceCents, "minimumCents": live.MinCents, "suggestedCents": live.SuggestedCents},
				After:    map[string]any{"kind": draft.PriceKind, "amountCents": draft.PriceCents, "minimumCents": draft.MinCents, "suggestedCents": draft.SuggestedCents},
				Metadata: map[string]any{"currency": currency, "orders": det.Orders}})
		}
		return nil
	})
	if err != nil {
		return ProductDetail{}, err
	}
	return s.Detail(ctx, ws, id)
}

// syncStripe creates or updates the product in Stripe and gives it a price
// for what the draft costs; a replaced price is archived, never deleted.
func (s *Service) syncStripe(ctx context.Context, ws *store.Workspace, p store.Product, d Live, sel earn.SellingConnection, currency string) (stripeIDs, error) {
	ids := readStripe(p.Stripe)
	client, err := s.Earn().SellingClient(ctx, ws, sel.ID)
	if err != nil {
		return ids, s.stripeProblem(ctx, err)
	}
	m, err := ensurePrice(ctx, client, ids[string(sel.Mode)], p.ID, d, currency, sel.PricesIncludeTax)
	if err != nil {
		return ids, s.stripeProblem(ctx, err)
	}
	ids[string(sel.Mode)] = m
	return ids, nil
}

// ensurePrice makes sure the provider has the product with a price for d.
func ensurePrice(ctx context.Context, client payments.Client, m stripeMode, productID uuid.UUID, d Live, currency string, inclusive bool) (stripeMode, error) {
	one, ok := client.(payments.OneTimePrices)
	if !ok {
		return m, errors.New("products: the payment provider doesn't sell one-time prices")
	}
	prod, err := client.SaveProduct(ctx, payments.Product{ID: m.Product, Name: d.Name, Description: d.ShortDesc, Active: true,
		Metadata: map[string]string{"reputecast_product": productID.String()}})
	if err != nil {
		return m, err
	}
	m.Product = prod.ID
	same := m.Price != "" && m.Kind == d.PriceKind && m.Amount == d.PriceCents && m.Min == d.MinCents && m.Suggested == d.SuggestedCents
	if same {
		return m, nil
	}
	req := payments.OneTimePrice{ProductID: prod.ID, Currency: currency, TaxInclusive: inclusive, Metadata: map[string]string{"reputecast_product": productID.String()}}
	if d.PriceKind == PricePWYW {
		req.Custom = &payments.CustomAmount{MinimumCents: d.MinCents, PresetCents: d.SuggestedCents}
	} else {
		req.AmountCents = d.PriceCents
	}
	price, err := one.CreateOneTimePrice(ctx, req)
	if err != nil {
		return m, err
	}
	if m.Price != "" {
		// Checkouts started before the change finish at the old price.
		_ = client.RetirePrice(ctx, m.Price)
		m.Retired = append(m.Retired, m.Price)
	}
	m.Price, m.Kind, m.Amount, m.Min, m.Suggested = price.ID, d.PriceKind, d.PriceCents, d.MinCents, d.SuggestedCents
	return m, nil
}

func (s *Service) stripeProblem(ctx context.Context, err error) error {
	l := lang(ctx)
	switch {
	case errors.Is(err, payments.ErrKeyRejected):
		return api.Err(409, "stripe_key_rejected", word(l, "product.stripeRejected", nil))
	case errors.Is(err, payments.ErrPermission):
		return api.Err(409, "stripe_permission", word(l, "product.stripePermission", nil))
	}
	s.log().WarnContext(ctx, "products: stripe", "err", err)
	return api.Err(503, "stripe_unavailable", word(l, "product.stripeUnavailable", nil))
}

// Archive stops selling a product: it can't be bought, and its buyers keep
// their downloads.
func (s *Service) Archive(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, archive bool) (ProductDetail, error) {
	l := lang(ctx)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ProductLock(ctx, store.ProductLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Product")
		}
		now := s.now()
		switch {
		case archive && cur.Status == StatusPublished:
			if err := q.ProductArchive(ctx, store.ProductArchiveParams{ID: id, At: now, Actor: actor(p)}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "product.archived", TargetType: "product", TargetID: id.String(), TargetName: cur.Name,
				Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": StatusArchived}})
		case !archive && cur.Status == StatusArchived:
			if err := q.ProductUnarchive(ctx, store.ProductUnarchiveParams{ID: id, At: now, Actor: actor(p)}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "product.unarchived", TargetType: "product", TargetID: id.String(), TargetName: cur.Name,
				Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": StatusPublished}})
		case archive:
			return api.Conflict(word(l, "product.archiveDraft", nil))
		}
		return api.Conflict(word(l, "product.notArchived", nil))
	})
	if err != nil {
		return ProductDetail{}, err
	}
	return s.Detail(ctx, ws, id)
}

// Delete removes a product that was never published; a published one is
// archived instead, so its buyers keep their downloads.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	l := lang(ctx)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ProductLock(ctx, store.ProductLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Product")
		}
		if cur.FirstPublishedAt != nil {
			return api.Conflict(word(l, "product.deletePublished", nil))
		}
		if err := q.ProductSoftDelete(ctx, store.ProductSoftDeleteParams{ID: id, At: s.now(), Actor: actor(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "product.deleted", TargetType: "product", TargetID: id.String(), TargetName: cur.Name,
			Before: draftOf(cur)})
	})
}

// Summary is the Earn overview's "Product sales" card: the last 30 days.
func (s *Service) Summary(ctx context.Context, ws *store.Workspace) (ProductsSummary, error) {
	now := s.now()
	since := periodOf("30d", now)
	out := ProductsSummary{From: since, Until: now}
	pay, _, err := s.payments(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Currency = pay.Currency
	stats, err := s.Q.ProductOrderStats(ctx, store.ProductOrderStatsParams{WorkspaceID: ws.ID, Mode: numbersMode(pay), Since: since, Until: now.Add(time.Second)})
	if err != nil {
		return out, err
	}
	for _, st := range stats {
		out.Orders += st.Orders
		out.GrossCents += st.Gross
		out.RefundedCents += st.Refunded
		out.RefundedOrders += st.Refunds
		if st.Orders > 0 {
			out.Products++
		}
	}
	if out.Currency == "" {
		out.Currency = "usd"
	}
	return out, nil
}
