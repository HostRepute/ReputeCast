package products

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Delivery: each order gets a personal download link by email through the
// product's delivery template (the Purchases category's product-delivery)
// on the Transactional route. Its download_url is a sensitive variable,
// so the message log never keeps it; the link is signed from its id.

// DeliverArgs sends a link's delivery email.
type DeliverArgs struct {
	LinkID uuid.UUID `json:"link_id"`
}

// Kind implements river.JobArgs.
func (DeliverArgs) Kind() string { return "products.deliver" }

// InsertOpts sends delivery emails from the events queue, retried.
func (DeliverArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 8}
}

// ConfirmArgs sends the double opt-in email of a buyer who ticked the list offer.
type ConfirmArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	ContactID   uuid.UUID `json:"contact_id"`
}

// Kind implements river.JobArgs.
func (ConfirmArgs) Kind() string { return "products.confirm" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ConfirmArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 5}
}

// newLink gives an order a new download link (the older ones stop
// working) and queues its delivery email, in tx.
func (s *Service) newLink(ctx context.Context, tx pgx.Tx, ws *store.Workspace, o store.ProductOrder, reason string, by *uuid.UUID) (uuid.UUID, error) {
	q := store.New(tx)
	p, err := q.ProductAny(ctx, o.ProductID)
	if err != nil {
		return uuid.Nil, err
	}
	l, ok := liveOf(p)
	if !ok {
		l = draftOf(p)
	}
	now := s.now()
	id := newID()
	days, downloads := l.LinkDays, l.LinkDownloads
	if days <= 0 {
		days = 7
	}
	if downloads <= 0 {
		downloads = 5
	}
	if err := q.ProductOrderLinkInsert(ctx, store.ProductOrderLinkInsertParams{ID: id, OrderID: o.ID, Reason: reason, CreatedBy: by, At: now,
		ExpiresAt: now.Add(time.Duration(days) * 24 * time.Hour), DownloadsAllowed: downloads}); err != nil {
		return uuid.Nil, err
	}
	if err := q.ProductOrderLinksSupersede(ctx, store.ProductOrderLinksSupersedeParams{OrderID: o.ID, Keep: id, At: now}); err != nil {
		return uuid.Nil, err
	}
	if s.Jobs != nil && s.Jobs() != nil {
		if _, err := s.Jobs().InsertTx(ctx, tx, DeliverArgs{LinkID: id}, nil); err != nil {
			return uuid.Nil, err
		}
	}
	return id, nil
}

// DownloadToken is a link's token: signed from its id, never stored.
func (s *Service) DownloadToken(wsID, orderID, linkID uuid.UUID) string {
	return s.Links.Sign(linktoken.Claims{Kind: TokenDownload, WorkspaceID: wsID, MessageID: linkID, ContactID: orderID})
}

// DownloadURL is a link's download page.
func (s *Service) DownloadURL(ctx context.Context, ws *store.Workspace, orderID, linkID uuid.UUID) string {
	return s.siteLinks(ctx, ws).base + "/d/" + s.DownloadToken(ws.ID, orderID, linkID)
}

// Deliver sends a link's delivery email. A link superseded or revoked
// meanwhile isn't sent.
func (s *Service) Deliver(ctx context.Context, linkID uuid.UUID) error {
	link, err := s.Q.ProductOrderLinkGet(ctx, linkID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if link.MessageID != nil || link.SupersededAt != nil {
		return nil
	}
	o, err := s.Q.ProductOrderByID(ctx, link.OrderID)
	if err != nil {
		return err
	}
	if o.AccessRevokedAt != nil {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, o.WorkspaceID)
	if err != nil {
		return err
	}
	p, err := s.Q.ProductAny(ctx, o.ProductID)
	if err != nil {
		return err
	}
	l, ok := liveOf(p)
	if !ok {
		l = draftOf(p)
	}
	key := ""
	if id, err := uuid.Parse(l.DeliveryTemplate); err == nil {
		if t, err := s.Q.ProductTemplateExists(ctx, store.ProductTemplateExistsParams{WorkspaceID: ws.ID, ID: id}); err == nil {
			key = t.Key
		}
	}
	if key == "" {
		return s.deliveryFailed(ctx, &ws, o, link, "template_missing", deliveryNoTemplate)
	}
	files, err := s.Q.DownloadFilesOf(ctx, store.DownloadFilesOfParams{OwnerKind: OwnerProduct, OwnerID: o.ProductID})
	if err != nil {
		return err
	}
	fs := make([]store.DownloadFile, len(files))
	for i, f := range files {
		fs[i] = f.DownloadFile
	}
	first := strings.TrimSpace(o.BuyerName)
	if i := strings.IndexByte(first, ' '); i > 0 {
		first = first[:i]
	}
	if o.ContactID != nil {
		if c, err := s.Q.ProductBuyerContact(ctx, store.ProductBuyerContactParams{WorkspaceID: ws.ID, ID: *o.ContactID}); err == nil && c.FirstName != "" {
			first = c.FirstName
		}
	}
	data := map[string]any{
		"product_name":      l.Name,
		"download_url":      s.DownloadURL(ctx, &ws, o.ID, link.ID),
		"expires_on":        dayText(&ws, link.ExpiresAt),
		"downloads_allowed": link.DownloadsAllowed,
		"order_number":      fmt.Sprint(o.Number),
		"amount":            moneyText(o.AmountCents, o.Currency),
		"file":              fileText(summary(fs)),
	}
	links := s.siteLinks(ctx, &ws)
	site := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(links.base, "https://"), "http://"), "/")
	data["site"] = site
	if s.SiteOn != nil && s.SiteOn(ctx, &ws) {
		data["account_url"] = site + "/account"
	}
	if first != "" {
		data["first_name"] = first
	}
	// The expiry date and the files' line are written in the language of
	// the version the buyer gets.
	sum := summary(fs)
	localize := func(code string) map[string]any {
		l := pagelabels.Lang(code)
		if code == "" {
			l = pagelabels.Lang(s.defaultLanguage(ctx, ws.ID))
		}
		return map[string]any{"expires_on": dayTextIn(l, &ws, link.ExpiresAt), "file": fileTextIn(l, sum)}
	}
	in := transactional.TransactionalSend{Template: key, To: transactional.TransactionalAddress{Email: o.BuyerEmail, Name: o.BuyerName}, Data: data,
		Tags: []string{"products"}, Metadata: map[string]string{"source": "products", "order": fmt.Sprint(o.Number)}, Localize: localize}
	status, body, err := s.Transactional().Send(ctx, &ws, in, "products:"+link.ID.String(), "api")
	if err != nil {
		var detail string
		var pr interface{ Error() string }
		if errors.As(err, &pr) {
			detail = pr.Error()
		}
		return s.deliveryFailed(ctx, &ws, o, link, "send", detail)
	}
	var res struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Detail string `json:"detail"`
		Code   string `json:"code"`
	}
	_ = json.Unmarshal(body, &res)
	mid, perr := uuid.Parse(res.ID)
	if status >= 300 || perr != nil {
		reason := res.Detail
		if reason == "" {
			reason = res.Code
		}
		if status >= 500 {
			// Temporary: River tries again.
			return fmt.Errorf("products: delivery answered %d: %s", status, reason)
		}
		return s.deliveryFailed(ctx, &ws, o, link, res.Code, reason)
	}
	return s.Q.ProductOrderLinkDelivery(ctx, store.ProductOrderLinkDeliveryParams{ID: link.ID, MessageID: &mid, Delivery: "sent", DeliveryError: ""})
}

// deliveryNoTemplate is the stored reason of a delivery a product without
// its template couldn't send; deliveryError writes it in the reader's
// language. Other reasons are what Transactional answered, in English.
const deliveryNoTemplate = "the product has no delivery template"

// deliveryError is a stored delivery error as a reader reads it.
func deliveryError(l i18n.Lang, stored string) string {
	if stored == deliveryNoTemplate {
		return word(l, "delivery.noTemplate", nil)
	}
	return stored
}

// deliveryFailed records a delivery email that couldn't go out and tells
// the people who manage orders.
func (s *Service) deliveryFailed(ctx context.Context, ws *store.Workspace, o store.ProductOrder, link store.ProductOrderLink, code, detail string) error {
	if err := s.Q.ProductOrderLinkDelivery(ctx, store.ProductOrderLinkDeliveryParams{ID: link.ID, Delivery: "failed", DeliveryError: clip(detail, 300)}); err != nil {
		return err
	}
	if detail == "" {
		detail = code
	}
	// Kept as messages, so each member reads it in their own language; the
	// missing template's reason too (Transactional's answers stay English).
	reason := "other"
	if detail == deliveryNoTemplate {
		reason = "template_missing"
	}
	args := i18n.Args{"order": orderNo(o.Number), "product": o.ProductName, "email": o.BuyerEmail, "reason": reason, "detail": detail}
	n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindProductDeliveryFailed, Severity: "bad",
		TitleText: msgs.M("notify.deliveryFailedTitle").Keep(nil), BodyText: msgs.M("notify.deliveryFailedBody").Keep(args),
		Link: "/w/" + ws.Slug + "/earn/products?order=" + o.ID.String(), Data: map[string]any{"order_id": o.ID.String(), "code": code}}
	if err := notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.OrdersManage, n); err != nil {
		s.log().WarnContext(ctx, "products: delivery failed notification", "err", err)
	}
	return nil
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// location is the workspace's timezone.
func location(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// dayText is a date as emails and public pages write it ("22 Oct 2026").
func dayText(ws *store.Workspace, t time.Time) string {
	return t.In(location(ws)).Format("2 Jan 2006")
}

// dayTextIn is dayText in a language's region format ("22 thg 10, 2026").
func dayTextIn(l i18n.Lang, ws *store.Workspace, t time.Time) string {
	if standard(l) {
		return dayText(ws, t)
	}
	return l.Region.Date(t.In(location(ws)))
}

// defaultLanguage is the workspace's default content language ("en" when
// it never set one): its emails are written in it.
func (s *Service) defaultLanguage(ctx context.Context, wsID uuid.UUID) string {
	if rows, err := s.Q.LanguagesList(ctx, wsID); err == nil {
		for _, r := range rows {
			if r.IsDefault && r.RemovedAt == nil {
				return r.Code
			}
		}
	}
	return "en"
}

// moneyText is an amount with its currency's symbol ("$29.00").
func moneyText(cents int64, currency string) string {
	sym := map[string]string{"usd": "$", "eur": "€", "gbp": "£", "aud": "A$", "cad": "C$", "jpy": "¥", "inr": "₹", "brl": "R$", "chf": "CHF ", "sek": "SEK "}[strings.ToLower(currency)]
	if sym == "" {
		sym = strings.ToUpper(currency) + " "
	}
	if strings.ToLower(currency) == "jpy" {
		return sym + thousands(cents)
	}
	return sym + thousands(cents/100) + fmt.Sprintf(".%02d", abs(cents%100))
}

// MoneyShort is an amount without cents when they're zero ("$29", "$15.50").
// MoneyText is an amount with its currency ("$29.00").
func MoneyText(cents int64, currency string) string { return moneyText(cents, currency) }

func MoneyShort(cents int64, currency string) string {
	full := moneyText(cents, currency)
	return strings.TrimSuffix(full, ".00")
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func thousands(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// fileText is how the delivery email names the files ("the PDF, 4.2 MB").
func fileText(f ProductFilesSummary) string { return fileTextIn(i18n.For(i18n.English), f) }

// fileTextIn is fileText in a language: the email's.
func fileTextIn(l i18n.Lang, f ProductFilesSummary) string {
	if f.Count == 0 {
		return ""
	}
	return word(l, "delivery.files", i18n.Args{"count": f.Count, "kind": f.Kind, "size": humanSizeIn(l, f.Bytes)})
}

// deliverWorker sends delivery emails.
type deliverWorker struct {
	river.WorkerDefaults[DeliverArgs]
	S *Service
}

func (w *deliverWorker) Work(ctx context.Context, job *river.Job[DeliverArgs]) error {
	return w.S.Deliver(ctx, job.Args.LinkID)
}

// confirmWorker sends a buyer's double opt-in email.
type confirmWorker struct {
	river.WorkerDefaults[ConfirmArgs]
	S *Service
}

func (w *confirmWorker) Work(ctx context.Context, job *river.Job[ConfirmArgs]) error {
	ws, err := w.S.Q.GetWorkspace(ctx, job.Args.WorkspaceID)
	if err != nil {
		return err
	}
	if w.S.Audience == nil || w.S.Audience() == nil {
		return nil
	}
	_, err = w.S.Audience().SendConfirmations(ctx, &ws, []uuid.UUID{job.Args.ContactID})
	return err
}
