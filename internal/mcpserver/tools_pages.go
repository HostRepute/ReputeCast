package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
)

// Landing and sales pages and digital products (design expansion phase 7,
// X19): agents list and read pages and products. They never publish a
// page, change a price or see who bought: a person does that in
// ReputeCast.

// PagesReader lists and reads pages (internal/pages).
type PagesReader interface {
	Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal, period string) (pages.PagesOverview, error)
	Detail(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (pages.PageDetail, error)
}

// ProductsReader lists and reads products (internal/products).
type ProductsReader interface {
	Overview(ctx context.Context, ws *store.Workspace, period string) (products.ProductsOverview, error)
	Detail(ctx context.Context, ws *store.Workspace, id uuid.UUID) (products.ProductDetail, error)
}

// Design expansion (append-only): the pages' and products' tools.
func init() {
	Tools = append(Tools,
		MCPToolInfo{"list_pages", "The workspace's landing and sales pages with their status, address, visibility and 30-day views and results.", string(authz.WebsiteView), false},
		MCPToolInfo{"get_page", "One page: its settings, status, publish checks and numbers. No visitor data.", string(authz.WebsiteView), false},
		MCPToolInfo{"list_products", "Digital products with their price, status and 30-day orders and gross: counts only, never buyers.", string(authz.ProductsView), false},
		MCPToolInfo{"get_product", "One product: its description, files, price, delivery settings and publish checks. No buyer data.", string(authz.ProductsView), false},
	)
}

type pageRef struct {
	Page string `json:"page" jsonschema:"the page's id or its address (the slug, like field-guide)"`
}

type productRef struct {
	Product string `json:"product" jsonschema:"the product's id or its address (the slug, like deliverability-field-guide)"`
}

// addPagesTools registers the pages' and products' tools.
func (s *Service) addPagesTools(srv *mcp.Server) {
	noPages := errors.New("pages aren't available on this server")
	noProducts := errors.New("products aren't available on this server")

	tool(s, srv, "list_pages", authz.WebsiteView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "page.list_read", targetType: "page", targetID: c.Workspace.ID.String(), targetName: c.Workspace.Name}
		if s.Pages == nil {
			return nil, e, noPages
		}
		ws := c.Workspace
		ov, err := s.Pages.Overview(ctx, &ws, c.Principal, "30d")
		if err != nil {
			return nil, e, err
		}
		out := make([]map[string]any, 0, len(ov.Pages))
		for _, p := range ov.Pages {
			m := map[string]any{"id": p.ID, "name": p.Name, "address": p.Path, "url": p.URL, "template": p.Template, "status": p.Status,
				"unpublished_changes": p.Changes, "visibility": p.Visibility, "views_30d": p.Views,
				"result_30d": map[string]any{"kind": p.Result.Kind, "count": p.Result.Count, "amount_cents": p.Result.AmountCents, "currency": p.Result.Currency,
					"conversion_percent": p.Result.Conversion}}
			if p.PublishedAt != nil {
				m["published_at"] = p.PublishedAt
			}
			if p.PublishAt != nil {
				m["publish_at"] = p.PublishAt
			}
			out = append(out, m)
		}
		e.summary = fmt.Sprintf("%d pages", len(out))
		return map[string]any{"pages": out, "shop_on": ov.Shop.On, "note": "Numbers are the last 30 days, counted on the server without cookies. Agents never publish pages."}, e, nil
	}, nil)

	tool(s, srv, "get_page", authz.WebsiteView, func(ctx context.Context, c *agentCall, in pageRef) (any, entry, error) {
		e := entry{action: "page.read", targetType: "page", targetName: in.Page}
		if s.Pages == nil {
			return nil, e, noPages
		}
		ws := c.Workspace
		id, err := s.findPage(ctx, &ws, c.Principal, in.Page)
		if err != nil {
			return nil, e, err
		}
		p, err := s.Pages.Detail(ctx, &ws, c.Principal, id)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = p.ID, p.Name
		checks := make([]map[string]any, 0, len(p.Checks))
		for _, ch := range p.Checks {
			checks = append(checks, map[string]any{"check": ch.Key, "status": ch.Status})
		}
		out := map[string]any{"id": p.ID, "name": p.Name, "address": p.Path, "url": p.URL, "template": p.Template, "status": p.Status,
			"unpublished_changes": p.Changes, "visibility": p.Settings.Visibility, "seo_title": p.Settings.SEOTitle, "seo_description": p.Settings.SEODescription,
			"in_navigation": p.Settings.ShowInNav, "focused": p.Settings.HideHeader, "blocks": p.Counts.Blocks, "images": p.Counts.Images, "links": p.Counts.Links,
			"versions": p.Versions, "checks": checks, "can_publish": false, "note": "Agents never publish. A person with website:publish publishes pages in ReputeCast."}
		if p.PublishAt != nil {
			out["publish_at"] = p.PublishAt
		}
		return out, e, nil
	}, nil)

	tool(s, srv, "list_products", authz.ProductsView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "product.list_read", targetType: "product", targetID: c.Workspace.ID.String(), targetName: c.Workspace.Name}
		if s.Products == nil {
			return nil, e, noProducts
		}
		ws := c.Workspace
		ov, err := s.Products.Overview(ctx, &ws, "30d")
		if err != nil {
			return nil, e, err
		}
		out := make([]map[string]any, 0, len(ov.Products))
		for _, p := range ov.Products {
			out = append(out, map[string]any{"id": p.ID, "name": p.Name, "status": p.Status, "price": priceOf(p.Price, p.Currency), "currency": p.Currency,
				"files": products.FileText(p.Files), "orders_30d": p.Orders, "gross_cents_30d": p.GrossCents, "refunds_30d": p.Refunds,
				"refunded_cents_30d": p.RefundedCents})
		}
		e.summary = fmt.Sprintf("%d products", len(out))
		return map[string]any{"products": out, "note": "Live orders of the last 30 days, before Stripe's fees; test orders are left out. Buyers are never listed."}, e, nil
	}, nil)

	tool(s, srv, "get_product", authz.ProductsView, func(ctx context.Context, c *agentCall, in productRef) (any, entry, error) {
		e := entry{action: "product.read", targetType: "product", targetName: in.Product}
		if s.Products == nil {
			return nil, e, noProducts
		}
		ws := c.Workspace
		id, err := s.findProduct(ctx, &ws, in.Product)
		if err != nil {
			return nil, e, err
		}
		p, err := s.Products.Detail(ctx, &ws, id)
		if err != nil {
			return nil, e, err
		}
		f := p.Draft
		if p.Live != nil {
			f = *p.Live
		}
		e.targetID, e.targetName = p.ID, f.Name
		files := make([]map[string]any, 0, len(p.Files))
		for _, fl := range p.Files {
			files = append(files, map[string]any{"name": fl.Name, "kind": fl.Kind, "bytes": fl.SizeBytes, "pages": fl.Pages})
		}
		checks := make([]map[string]any, 0, len(p.Checks))
		for _, ch := range p.Checks {
			checks = append(checks, map[string]any{"check": ch.Key, "ok": ch.OK})
		}
		return map[string]any{"id": p.ID, "name": f.Name, "slug": f.Slug, "status": p.Status, "short_description": f.ShortDescription,
			"long_description": f.LongDescription, "price": priceOf(f.Price, p.Payments.Currency), "in_shop": f.ShowInShop, "list_offer": f.Offer.On,
			"link_days": f.Delivery.LinkDays, "downloads_per_link": f.Delivery.Downloads, "files": files, "unpublished_changes": p.Changes, "checks": checks,
			"note": "Agents never publish products or change prices. Buyers and orders aren't available to agents."}, e, nil
	}, nil)
}

// priceOf says a price in words for the agent ("$29.00", "pay what you
// want, from $5.00, suggested $15.00").
func priceOf(p products.ProductPrice, currency string) string {
	if currency == "" {
		currency = "usd"
	}
	if p.Kind == "pwyw" {
		out := "pay what you want, from " + products.MoneyText(p.MinimumCents, currency)
		if p.SuggestedCents > 0 {
			out += ", suggested " + products.MoneyText(p.SuggestedCents, currency)
		}
		return out
	}
	return products.MoneyText(p.AmountCents, currency)
}

// findPage turns a page's id or address into its id.
func (s *Service) findPage(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (uuid.UUID, error) {
	ref = strings.Trim(strings.TrimSpace(ref), "/")
	if id, err := uuid.Parse(ref); err == nil {
		return id, nil
	}
	ov, err := s.Pages.Overview(ctx, ws, p, "30d")
	if err != nil {
		return uuid.Nil, err
	}
	for _, pg := range ov.Pages {
		if pg.Slug == ref {
			return uuid.Parse(pg.ID)
		}
	}
	return uuid.Nil, fmt.Errorf("no page at /%s", ref)
}

// findProduct turns a product's id or address into its id.
func (s *Service) findProduct(ctx context.Context, ws *store.Workspace, ref string) (uuid.UUID, error) {
	ref = strings.Trim(strings.TrimSpace(ref), "/")
	if id, err := uuid.Parse(ref); err == nil {
		return id, nil
	}
	rows, err := s.Q.ProductsList(ctx, ws.ID)
	if err != nil {
		return uuid.Nil, err
	}
	for _, r := range rows {
		if r.Product.Slug == ref {
			return r.Product.ID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("no product at /products/%s", ref)
}
