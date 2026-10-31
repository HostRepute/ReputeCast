package products

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// FailedTemplateKey is the buyer's notice that a delayed payment (a bank
// debit) didn't clear (review 2026-11 EN-7). It is made, as a draft, with
// the delivery template, and sent once it's published.
const FailedTemplateKey = "product-payment-failed"

// PaymentFailedArgs tells a buyer their payment didn't go through. The
// address is the checkout's, so the job carries no personal data.
type PaymentFailedArgs struct {
	CheckoutID uuid.UUID `json:"checkout_id"`
}

// Kind implements river.JobArgs.
func (PaymentFailedArgs) Kind() string { return "products.payment_failed" }

// InsertOpts sends from the events queue, retried, once per checkout.
func (PaymentFailedArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 8, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type paymentFailedWorker struct {
	river.WorkerDefaults[PaymentFailedArgs]
	S *Service
}

func (w *paymentFailedWorker) Work(ctx context.Context, job *river.Job[PaymentFailedArgs]) error {
	return w.S.SendPaymentFailed(ctx, job.Args.CheckoutID)
}

// SendPaymentFailed emails the buyer of a failed checkout, with a link
// back to the product to try again. Without a published template nothing
// is sent: the Transactional log shows nothing, the order list nothing.
func (s *Service) SendPaymentFailed(ctx context.Context, checkoutID uuid.UUID) error {
	co, err := s.Q.ProductCheckoutGet(ctx, checkoutID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if co.Status != "failed" || co.Email == "" || s.Transactional == nil || s.Transactional() == nil {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, co.WorkspaceID)
	if err != nil {
		return err
	}
	// A workspace whose products predate the notice gets its draft now.
	if _, err := s.ensureStarter(ctx, &ws, FailedTemplateKey, "template.failedName"); err != nil {
		return err
	}
	t, err := s.Q.ProductTemplateByKey(ctx, store.ProductTemplateByKeyParams{WorkspaceID: ws.ID, Key: FailedTemplateKey})
	if err != nil {
		return err
	}
	if t.Version == 0 {
		s.log().Info("products: payment failed email not sent: its template isn't published", "workspace", ws.ID, "checkout", co.ID)
		return nil
	}
	p, err := s.Q.ProductAny(ctx, co.ProductID)
	if err != nil {
		return err
	}
	l, ok := liveOf(p)
	if !ok {
		l = draftOf(p)
	}
	links := s.siteLinks(ctx, &ws)
	retry := co.SourceUrl
	if retry == "" || !strings.HasPrefix(retry, "https://") && !strings.HasPrefix(retry, "http://") {
		retry = links.product(l.Slug)
	}
	data := map[string]any{"product_name": l.Name, "retry_url": retry,
		"site": strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(links.base, "https://"), "http://"), "/")}
	in := transactional.TransactionalSend{Template: FailedTemplateKey, To: transactional.TransactionalAddress{Email: co.Email}, Data: data,
		Tags: []string{"products"}, Metadata: map[string]string{"source": "products", "checkout": co.ID.String()}}
	status, body, err := s.Transactional().Send(ctx, &ws, in, "products-failed:"+co.ID.String(), "api")
	if err != nil {
		return err
	}
	if status >= 500 {
		return fmt.Errorf("products: payment failed email answered %d: %s", status, body)
	}
	if status >= 300 {
		var res struct{ Code, Detail string }
		_ = json.Unmarshal(body, &res)
		s.log().Warn("products: payment failed email refused", "workspace", ws.ID, "checkout", co.ID, "code", res.Code, "detail", res.Detail)
	}
	return nil
}
