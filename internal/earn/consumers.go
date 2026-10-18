package earn

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// One-time payments belong to other domains (digital products, design
// expansion phase 7): their events arrive on the same webhook as the paid
// members' and are stored the same way, but a consumer registered here
// takes them first, inside the transaction that applies the event. An event
// a consumer handles never reaches the paid-member logic, so a product's
// refund is never booked on a subscription.

// PaymentEvent is a one-time payment event offered to consumers.
type PaymentEvent struct {
	Workspace    *store.Workspace
	ConnectionID uuid.UUID
	Provider     string
	Mode         payments.Mode
	// EventID is the payments event log's row; Type the provider's event
	// type ("checkout.session.completed").
	EventID uuid.UUID
	Type    string
	Payment payments.Payment
	// Client opens the connection's client (to read more from the provider).
	Client func() (payments.Client, error)
}

// Consumed is what a consumer did with an event, for the event log.
type Consumed struct {
	// Outcome is the event log's key ("order_paid"); Ignored files it as
	// ignored rather than applied.
	Outcome   string
	Ignored   bool
	Detail    map[string]any
	ContactID *uuid.UUID
	// AmountCents is signed: a refund is negative.
	AmountCents *int64
	Currency    string
}

// PaymentConsumer handles an event about its own objects; ok false leaves
// the event to the next consumer, then to Earn.
type PaymentConsumer func(ctx context.Context, tx pgx.Tx, e PaymentEvent) (Consumed, bool, error)

// Consumers belong to one Service: each App (and each test's) has its own,
// with its own database.
var (
	consumersMu sync.RWMutex
	consumers   = map[*Service][]PaymentConsumer{}
)

// RegisterPaymentConsumer adds a consumer of the one-time payment events s
// applies.
func RegisterPaymentConsumer(s *Service, c PaymentConsumer) {
	consumersMu.Lock()
	defer consumersMu.Unlock()
	consumers[s] = append(consumers[s], c)
}

// consume offers an event to the consumers; ok reports that one took it.
func (s *Service) consume(ctx context.Context, tx pgx.Tx, ws *store.Workspace, conn store.PaymentConnection, pe store.PaymentEvent) (outcome, bool, error) {
	consumersMu.RLock()
	list := append([]PaymentConsumer(nil), consumers[s]...)
	consumersMu.RUnlock()
	if len(list) == 0 {
		return outcome{}, false, nil
	}
	v, ok := payments.ParserFor(firstNonEmpty(conn.Provider, DefaultProvider))
	if !ok {
		return outcome{}, false, nil
	}
	pp, ok := v.(payments.PaymentParser)
	if !ok {
		return outcome{}, false, nil
	}
	p, err := pp.ParsePayment(pe.Payload)
	if err != nil || p.Kind == payments.PaymentOther {
		return outcome{}, false, nil
	}
	e := PaymentEvent{Workspace: ws, ConnectionID: conn.ID, Provider: firstNonEmpty(conn.Provider, DefaultProvider), Mode: payments.Mode(pe.Mode),
		EventID: pe.ID, Type: pe.Type, Payment: p, Client: func() (payments.Client, error) { return s.open(conn) }}
	for _, c := range list {
		got, ok, err := c(ctx, tx, e)
		if err != nil {
			return outcome{}, true, err
		}
		if !ok {
			continue
		}
		o := applied(got.Outcome, got.Detail)
		if got.Ignored {
			o.status = "ignored"
		}
		o.contact, o.amount, o.currency = got.ContactID, got.AmountCents, got.Currency
		return o, true, nil
	}
	return outcome{}, false, nil
}
