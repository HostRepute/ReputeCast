package earn

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Selling is how other domains sell through the workspace's payment
// connection (digital products, design expansion phase 7): the connection
// public pages use (live once connected, else test), its currency and tax
// settings, and a client on it. It needs no principal: public pages and
// jobs call it.

// SellingConnection is a connected payment account, as sellers see it.
type SellingConnection struct {
	ID       uuid.UUID
	Provider string
	Mode     payments.Mode
	// Account is the account's name ("Acme Media").
	Account string
	// Currency is the workspace currency (lowercase), else the account's.
	Currency string
	// Tax: the provider calculates tax (Stripe Tax), the workspace adds it
	// at checkout, and whether prices include it.
	TaxEnabled       bool
	TaxAtCheckout    bool
	PricesIncludeTax bool
	// Live is also connected, when Mode is test: false means "test mode".
	HasLive bool
}

// Selling returns the connection sales use; ok is false when no payment
// provider is connected.
func (s *Service) Selling(ctx context.Context, ws *store.Workspace) (SellingConnection, bool, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return SellingConnection{}, false, err
	}
	c, ok := active(conns)
	if !ok {
		return SellingConnection{}, false, nil
	}
	return s.selling(ctx, ws, conns, c)
}

// SellingIn returns the connected account of one mode (a test order's
// return from checkout).
func (s *Service) SellingIn(ctx context.Context, ws *store.Workspace, mode payments.Mode) (SellingConnection, bool, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return SellingConnection{}, false, err
	}
	c, ok := connected(conns, string(mode))
	if !ok {
		return SellingConnection{}, false, nil
	}
	return s.selling(ctx, ws, conns, c)
}

func (s *Service) selling(ctx context.Context, ws *store.Workspace, conns []connRow, c connRow) (SellingConnection, bool, error) {
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return SellingConnection{}, false, err
	}
	out := SellingConnection{ID: c.ID, Provider: firstNonEmpty(c.Provider, DefaultProvider), Mode: payments.Mode(c.Mode), Account: c.AccountName,
		Currency: firstNonEmpty(st.Currency, c.Currency), TaxEnabled: c.TaxEnabled, TaxAtCheckout: st.TaxAtCheckout && c.TaxEnabled,
		PricesIncludeTax: st.PricesIncludeTax}
	_, out.HasLive = connected(conns, string(payments.Live))
	return out, true, nil
}

// SellingClient opens a client on a selling connection.
func (s *Service) SellingClient(ctx context.Context, ws *store.Workspace, id uuid.UUID) (payments.Client, error) {
	row, err := s.Q.PayConnection(ctx, store.PayConnectionParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	return s.open(row.PaymentConnection)
}

// Dashboard links to an object in a provider's dashboard ("payments" and
// a payment intent's id for an order).
func Dashboard(provider string, mode payments.Mode, kind, id string) string {
	return dashboard(firstNonEmpty(provider, DefaultProvider), mode, kind, id)
}
