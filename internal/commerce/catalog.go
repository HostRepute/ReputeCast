package commerce

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Catalog is the composer's product picker: each store with products, its
// collections and counts. A disconnected store isn't offered (its blocks
// keep their last prices).
func (s *Service) Catalog(ctx context.Context, ws *store.Workspace) (CommerceCatalog, error) {
	out := CommerceCatalog{Stores: []CommerceCatalogStore{}}
	rows, err := s.Q.CommerceStores(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, st := range rows {
		if st.Status != StatusConnected && st.Status != StatusImporting || !st.SyncProducts {
			continue
		}
		n, err := s.Q.CommerceProductCount(ctx, st.ID)
		if err != nil {
			return out, err
		}
		c := CommerceCatalogStore{ID: st.ID.String(), Name: st.Name, Platform: st.Platform, Currency: st.Currency, Products: n,
			Collections: []CommerceCollection{}, Status: st.Status}
		cols, err := s.Q.CommerceCollections(ctx, st.ID)
		if err != nil {
			return out, err
		}
		for _, col := range cols {
			c.Collections = append(c.Collections, CommerceCollection{ID: col.ExternalID, Title: col.Title, Products: col.Products})
		}
		out.Stores = append(out.Stores, c)
	}
	return out, nil
}

// CommerceProductsInput pages a store's catalog.
type CommerceProductsInput struct {
	Collection string `query:"collection" maxLength:"64"`
	Search     string `query:"search" maxLength:"100"`
	Cursor     int    `query:"cursor" minimum:"0"`
	Limit      int    `query:"limit" minimum:"0" maximum:"100"`
}

// Products is a page of a store's products, newest first, with their
// prices and availability now.
func (s *Service) Products(ctx context.Context, ws *store.Workspace, id uuid.UUID, in CommerceProductsInput) (CommerceProductList, error) {
	st, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return CommerceProductList{}, api.Err(http.StatusNotFound, api.CodeNotFound, mStoreNotFound.Text(i18n.LangFrom(ctx), nil))
	}
	if err != nil {
		return CommerceProductList{}, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 48
	}
	search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.TrimSpace(in.Search))
	rows, err := s.Q.CommerceProducts(ctx, store.CommerceProductsParams{StoreID: st.ID, Collection: in.Collection, Search: search, Lim: int32(limit + 1), Off: int32(in.Cursor)})
	if err != nil {
		return CommerceProductList{}, err
	}
	out := CommerceProductList{Items: []CommerceProduct{}}
	for i, r := range rows {
		if i == limit {
			out.Next = in.Cursor + limit
			break
		}
		out.Items = append(out.Items, productView(r))
	}
	if out.Total, err = s.Q.CommerceProductCount(ctx, st.ID); err != nil {
		return out, err
	}
	return out, nil
}

// CatalogEntry is a store platform's card in Settings › Integrations
// (the Commerce group).
type CatalogEntry struct {
	Platform     string
	Status       string // not_connected, waiting, connected, attention
	ConnectionID string
	Stores       int
	Name         string
	State        string
	Orders       int64
	LastEventAt  *time.Time
}

// CatalogEntries are the Commerce group's two cards. A store disconnected
// by a person shows Not connected; one the platform revoked needs
// attention.
func (s *Service) CatalogEntries(ctx context.Context, ws *store.Workspace) ([]CatalogEntry, error) {
	rows, err := s.Q.CommerceStores(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	since := s.now().Add(-PeriodDays * 24 * time.Hour)
	out := []CatalogEntry{}
	for _, platform := range []string{Shopify, WooCommerce} {
		e := CatalogEntry{Platform: platform, Status: "not_connected"}
		rank := -1
		for _, st := range rows {
			if st.Platform != platform {
				continue
			}
			state := s.syncState(ctx, st).State
			status, r := "", 0
			switch {
			case st.Status == StatusDisconnected && st.RevokedAt != nil && st.DisconnectedBy == nil:
				status, r, state = "attention", 3, "revoked"
			case st.Status == StatusDisconnected:
				continue
			case st.Status == StatusConnecting:
				status, r = "waiting", 0
			case state == "webhook_missing" || state == "rate_limited" || state == "no_plugin":
				status, r = "attention", 2
			default:
				status, r = "connected", 1
			}
			e.Stores++
			if r > rank {
				rank = r
				e.Status, e.ConnectionID, e.Name, e.State, e.LastEventAt = status, st.ID.String(), st.Name, state, st.LastEventAt
				if t, err := s.Q.CommerceStoreTotals(ctx, store.CommerceStoreTotalsParams{StoreID: st.ID, Since: since, Until: s.now()}); err == nil {
					e.Orders = t.Orders
				}
			}
		}
		out = append(out, e)
	}
	return out, nil
}

// AutomationOptions are the store triggers' choices: each store (a
// disconnected one too, so a paused flow still names it), its products and
// collections.
func (s *Service) AutomationOptions(ctx context.Context, ws *store.Workspace) (CommerceAutomationOptions, error) {
	out := CommerceAutomationOptions{Stores: []CommerceTriggerStore{}}
	rows, err := s.Q.CommerceStores(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, st := range rows {
		if st.Status == StatusConnecting {
			continue
		}
		host := st.PublicDomain
		if host == "" {
			host = st.Domain
		}
		t := CommerceTriggerStore{ID: st.ID.String(), Name: st.Name, Address: host, Platform: st.Platform, Status: st.Status, Currency: st.Currency,
			Checkouts: st.SyncCheckouts && (st.Platform == Shopify || st.PluginFound), Products: []CommerceProductRef{}, Collections: []CommerceCollection{}}
		products, err := s.Q.CommerceProducts(ctx, store.CommerceProductsParams{StoreID: st.ID, Lim: 500})
		if err != nil {
			return out, err
		}
		for _, p := range products {
			t.Products = append(t.Products, CommerceProductRef{ID: p.ExternalID, Title: p.Title})
		}
		cols, err := s.Q.CommerceCollections(ctx, st.ID)
		if err != nil {
			return out, err
		}
		for _, c := range cols {
			t.Collections = append(t.Collections, CommerceCollection{ID: c.ExternalID, Title: c.Title, Products: c.Products})
		}
		out.Stores = append(out.Stores, t)
	}
	return out, nil
}
