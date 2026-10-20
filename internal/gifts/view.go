package gifts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/revenue"
	"github.com/hostrepute/reputecast/internal/store"
)

// GiftsView is Earn › Gifts: the numbers, the gift options and the gift
// page, the gift emails, and the gifts that need a look.
type GiftsView struct {
	// Mode is the mode gifts are sold in (live once the live key is in);
	// "" without a payment provider.
	Mode      string `json:"mode" enum:"live,test,"`
	Connected bool   `json:"connected"`
	Provider  string `json:"provider"`
	Currency  string `json:"currency"`

	Numbers GiftNumbers  `json:"numbers"`
	Options []GiftOption `json:"options"`
	Page    GiftPage     `json:"page"`
	// Tiers can be given (the option editor's choices).
	Tiers []GiftTier `json:"tiers"`
	// ListNames are the lists a recipient joins (the first option's tier's).
	ListNames []string `json:"listNames"`

	Emails []GiftEmail `json:"emails"`
	// The emails' sender, category and route, as Member emails shows them.
	FromName     string `json:"fromName"`
	FromEmail    string `json:"fromEmail"`
	CategoryName string `json:"categoryName"`
	CategoryKind string `json:"categoryKind"`
	RoutePrimary string `json:"routePrimary"`
	// Ending is set while gift-ending isn't published and a reminder is
	// due: when the first one is, and for whose gift.
	Ending *GiftEndingWarning `json:"ending,omitempty"`

	// NeedsLook: delivered a week ago and not redeemed, or refunded lately.
	NeedsLook []GiftRow `json:"needsLook"`

	CanEdit   bool     `json:"canEdit"`
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// GiftNumbers are the tab's numbers of the last 30 days.
type GiftNumbers struct {
	Sold      int64 `json:"sold"`
	SoldCents int64 `json:"soldCents"`
	// ByMonths splits the sales by length: 7 for a year, 4 for 3 months.
	ByMonths []GiftMonthsCount `json:"byMonths"`
	// Of the gifts sold, those delivered, redeemed and still waiting.
	Delivered int64 `json:"delivered"`
	Redeemed  int64 `json:"redeemed"`
	Waiting   int64 `json:"waiting"`
	// Scheduled gifts, and the next delivery (in its buyer's time zone).
	Scheduled    int64      `json:"scheduled"`
	NextAt       *time.Time `json:"nextAt,omitempty"`
	NextTimezone string     `json:"nextTimezone,omitempty"`
	// Paid members with gift access now (never counted in MRR).
	WithAccess int64 `json:"withAccess"`
}

// GiftMonthsCount is how many gifts of a length sold.
type GiftMonthsCount struct {
	Months int   `json:"months"`
	Count  int64 `json:"count"`
}

// GiftOption is a gift readers can buy: a tier for a period at a price.
type GiftOption struct {
	ID     string `json:"id"`
	TierID string `json:"tierId"`
	Tier   string `json:"tier"`
	// Short is the tier's name without its list's ("Insider").
	Short       string `json:"short"`
	Months      int    `json:"months"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	// Sold in the last 30 days, and for how much.
	Sold      int64 `json:"sold"`
	SoldCents int64 `json:"soldCents"`
}

// GiftPage is the gift page's switch, address and redemption window.
type GiftPage struct {
	On   bool   `json:"on"`
	URL  string `json:"url"`
	Host string `json:"host"`
	// LinkMonths is how long a gift link works after delivery.
	LinkMonths int `json:"linkMonths"`
	// Who last turned it on or off, and when.
	ChangedBy string     `json:"changedBy,omitempty"`
	ChangedAt *time.Time `json:"changedAt,omitempty"`
}

// GiftTier is a tier an option can give, with its monthly price (the
// option editor suggests the same price).
type GiftTier struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Short        string `json:"short"`
	MonthlyCents int64  `json:"monthlyCents"`
	YearlyCents  int64  `json:"yearlyCents"`
}

// GiftEmail is one of the three gift emails and its template's state.
type GiftEmail struct {
	Email        string `json:"email" enum:"gift_purchased,gift_received,gift_ending"`
	TemplateID   string `json:"templateId,omitempty"`
	Template     string `json:"template"`
	Name         string `json:"name"`
	Status       string `json:"status" enum:"live,draft,missing"`
	LiveVersion  int    `json:"liveVersion"`
	DraftVersion int    `json:"draftVersion"`
}

// GiftEndingWarning: gift-ending isn't published, and the first reminder
// is due on Due, for the gift Buyer gave (in person, or to Recipient).
type GiftEndingWarning struct {
	Due        time.Time `json:"due"`
	Buyer      string    `json:"buyer"`
	InPerson   bool      `json:"inPerson"`
	Recipient  string    `json:"recipient"`
	TemplateID string    `json:"templateId,omitempty"`
}

// GiftRow is a gift in the table.
type GiftRow struct {
	ID string `json:"id"`
	// BoughtAt is when it was paid.
	BoughtAt   time.Time `json:"boughtAt"`
	Buyer      string    `json:"buyer"`
	BuyerEmail string    `json:"buyerEmail"`
	InPerson   bool      `json:"inPerson"`
	// Recipient's name and email (none in person).
	RecipientName  string `json:"recipientName"`
	RecipientEmail string `json:"recipientEmail"`
	Tier           string `json:"tier"`
	Short          string `json:"short"`
	Months         int    `json:"months"`
	AmountCents    int64  `json:"amountCents"`
	Currency       string `json:"currency"`
	// Status as the table shows it.
	Status string `json:"status" enum:"scheduled,not_redeemed,redeemed,expired,refunded"`
	// DeliverAt is when a scheduled gift goes out (nil: at once), in the
	// buyer's Timezone; DeliveredAt when it went.
	DeliverAt   *time.Time `json:"deliverAt,omitempty"`
	Timezone    string     `json:"timezone"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	// RemindAt is when "Your gift" goes again if it isn't redeemed;
	// RemindedAt when it went.
	RemindAt   *time.Time `json:"remindAt,omitempty"`
	RemindedAt *time.Time `json:"remindedAt,omitempty"`
	// LinkExpiresAt is when the redeem link stops working.
	LinkExpiresAt *time.Time `json:"linkExpiresAt,omitempty"`
	RedeemedAt    *time.Time `json:"redeemedAt,omitempty"`
	// RedeemedBy is set when someone other than the recipient redeemed
	// (an in-person gift, or a passed-on link).
	RedeemedBy string     `json:"redeemedBy,omitempty"`
	AccessEnds *time.Time `json:"accessEnds,omitempty"`
	RefundedAt *time.Time `json:"refundedAt,omitempty"`
	// EmailStatus is how the last "Your gift" email went: sent, or why not.
	EmailStatus string `json:"emailStatus"`
	// PaymentURL opens the payment in the provider's dashboard.
	PaymentURL string `json:"paymentUrl,omitempty"`
}

// GiftList is a page of the Gifts table.
type GiftList struct {
	Items    []GiftRow  `json:"items"`
	Counts   GiftCounts `json:"counts"`
	Status   string     `json:"status"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
	// Total is how many gifts the filter shows.
	Total int64 `json:"total"`
}

// GiftCounts are the table's filters' counts.
type GiftCounts struct {
	All         int64 `json:"all"`
	Scheduled   int64 `json:"scheduled"`
	NotRedeemed int64 `json:"notRedeemed"`
	Redeemed    int64 `json:"redeemed"`
	Expired     int64 `json:"expired"`
	Refunded    int64 `json:"refunded"`
}

// PageSize is the table's page.
const PageSize = 20

// tableStatus is a gift's status as the table names it.
func tableStatus(status string) string {
	switch status {
	case StatusPaid:
		return "scheduled"
	case StatusDelivered:
		return "not_redeemed"
	}
	return status
}

// dbStatus is the filter's status in the database.
func dbStatus(filter string) (string, bool) {
	switch filter {
	case "", "all":
		return "", true
	case "scheduled":
		return StatusPaid, true
	case "not_redeemed":
		return StatusDelivered, true
	case StatusRedeemed, StatusExpired, StatusRefunded:
		return filter, true
	}
	return "", false
}

// sellMode is the mode gifts are sold and listed in: live once its key is
// in, else test; "" without a provider (the demo lists live gifts then).
func (s *Service) sellMode(ctx context.Context, ws *store.Workspace) (string, bool) {
	mode := s.Earn.ActiveMode(ctx, ws.ID)
	if mode == "" {
		return "live", false
	}
	return mode, true
}

// View builds Earn › Gifts.
func (s *Service) View(ctx context.Context, ws *store.Workspace, p *authz.Principal) (GiftsView, error) {
	now := s.now()
	mode, connected := s.sellMode(ctx, ws)
	ps := s.Earn.PageSettingsOf(ctx, ws.ID)
	out := GiftsView{Connected: connected, Currency: ps.Currency, Options: []GiftOption{}, Tiers: []GiftTier{}, Emails: []GiftEmail{}, NeedsLook: []GiftRow{},
		CanEdit: p.Can(authz.EarnEdit), CanManage: p.Can(authz.EarnManage)}
	if connected {
		out.Mode = mode
	}
	if conns, err := s.Q.RevenueConnections(ctx, ws.ID); err == nil {
		for _, c := range conns {
			if c.Mode == mode || out.Provider == "" {
				out.Provider = providerName(c.Provider)
			}
		}
	}
	if out.Provider == "" {
		out.Provider = "Stripe"
	}
	if !out.CanEdit || !out.CanManage {
		perm := authz.EarnManage
		if !out.CanEdit {
			perm = authz.EarnEdit
		}
		out.Helpers = revenue.Helpers(ctx, s.Q, ws.ID, p.UserID, 2, perm)
	}
	if out.Helpers == nil {
		out.Helpers = []string{}
	}
	since := now.Add(-30 * 24 * time.Hour)
	var err error
	if out.Options, err = s.options(ctx, ws, mode, since); err != nil {
		return out, err
	}
	if out.Tiers, err = s.tiers(ctx, ws); err != nil {
		return out, err
	}
	out.ListNames = []string{}
	if rows, err := s.Q.GiftOptions(ctx, ws.ID); err == nil && len(rows) > 0 {
		if names, err := s.Q.EarnListNames(ctx, rows[0].ListIds); err == nil && names != nil {
			out.ListNames = names
		}
	}
	if out.Page, err = s.pageOf(ctx, ws); err != nil {
		return out, err
	}
	if out.Numbers, err = s.numbers(ctx, ws, mode, since); err != nil {
		return out, err
	}
	if err := s.emails(ctx, ws, p, mode, &out); err != nil {
		return out, err
	}
	looks, err := s.Q.GiftNeedsLook(ctx, store.GiftNeedsLookParams{WorkspaceID: ws.ID, Mode: mode, WeekAgo: now.Add(-RemindAfter), MonthAgo: since})
	if err != nil {
		return out, err
	}
	dash := s.dashboard(ctx, ws, mode)
	for _, r := range looks {
		out.NeedsLook = append(out.NeedsLook, s.row(ctx, gift(r.EarnGift, r.TierName, r.ListIds, nil), dash))
	}
	return out, nil
}

func providerName(p string) string {
	switch p {
	case "", "stripe":
		return "Stripe"
	}
	return strings.ToUpper(p[:1]) + p[1:]
}

// options lists the gift options with what they sold since a time.
func (s *Service) options(ctx context.Context, ws *store.Workspace, mode string, since time.Time) ([]GiftOption, error) {
	rows, err := s.Q.GiftOptions(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	sales, err := s.Q.GiftOptionSales(ctx, store.GiftOptionSalesParams{WorkspaceID: ws.ID, Mode: mode, Since: since})
	if err != nil {
		return nil, err
	}
	sold := map[uuid.UUID]store.GiftOptionSalesRow{}
	for _, r := range sales {
		if r.OptionID != nil {
			sold[*r.OptionID] = r
		}
	}
	out := make([]GiftOption, 0, len(rows))
	for _, r := range rows {
		o := GiftOption{ID: r.ID.String(), TierID: r.TierID.String(), Tier: r.TierName, Short: s.tierShort(ctx, r.TierName, r.ListIds), Months: int(r.Months),
			AmountCents: r.AmountCents, Currency: r.Currency}
		if x, ok := sold[r.ID]; ok {
			o.Sold, o.SoldCents = x.Sold, x.Cents
		}
		out = append(out, o)
	}
	return out, nil
}

// tiers are the tiers an option can give: not archived, with their
// current prices.
func (s *Service) tiers(ctx context.Context, ws *store.Workspace) ([]GiftTier, error) {
	rows, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	prices, err := s.Q.EarnPrices(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := []GiftTier{}
	for _, t := range rows {
		if t.Visibility == "archived" {
			continue
		}
		g := GiftTier{ID: t.ID.String(), Name: t.Name, Short: s.tierShort(ctx, t.Name, t.ListIds)}
		for _, pr := range prices {
			if pr.TierID != t.ID || pr.Status != "current" {
				continue
			}
			switch pr.Period {
			case "month":
				g.MonthlyCents = pr.AmountCents
			case "year":
				g.YearlyCents = pr.AmountCents
			}
		}
		out = append(out, g)
	}
	return out, nil
}

// pageOf reads the gift page's settings.
func (s *Service) pageOf(ctx context.Context, ws *store.Workspace) (GiftPage, error) {
	if err := s.Q.EarnSettingsEnsure(ctx, ws.ID); err != nil {
		return GiftPage{}, err
	}
	st, err := s.Q.EarnSettings(ctx, ws.ID)
	if err != nil {
		return GiftPage{}, err
	}
	out := GiftPage{On: st.GiftPageOn, URL: s.Earn.PageURL(ctx, ws, "gift"), Host: s.Earn.PageHost(ctx, ws, "gift"), LinkMonths: int(st.GiftLinkMonths),
		ChangedAt: st.GiftPageAt}
	if st.GiftPageBy != nil {
		out.ChangedBy, _ = s.Q.GiftPageByName(ctx, ws.ID)
	}
	return out, nil
}

// numbers are the tab's numbers since a time.
func (s *Service) numbers(ctx context.Context, ws *store.Workspace, mode string, since time.Time) (GiftNumbers, error) {
	out := GiftNumbers{ByMonths: []GiftMonthsCount{}}
	by, err := s.Q.GiftSoldByMonths(ctx, store.GiftSoldByMonthsParams{WorkspaceID: ws.ID, Mode: mode, Since: since})
	if err != nil {
		return out, err
	}
	for _, r := range by {
		out.Sold += r.Sold
		out.SoldCents += r.Cents
		out.ByMonths = append(out.ByMonths, GiftMonthsCount{Months: int(r.Months), Count: r.Sold})
	}
	n, err := s.Q.GiftNumbers(ctx, store.GiftNumbersParams{Since: since, WorkspaceID: ws.ID, Mode: mode})
	if err != nil {
		return out, err
	}
	out.Delivered, out.Redeemed, out.Waiting, out.Scheduled = n.Delivered, n.Redeemed, n.Waiting, n.Scheduled
	if next, err := s.Q.GiftNextScheduled(ctx, store.GiftNextScheduledParams{WorkspaceID: ws.ID, Mode: mode}); err == nil {
		out.NextAt, out.NextTimezone = next.DeliverAt, next.Timezone
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if out.WithAccess, err = s.Q.GiftAccessNow(ctx, store.GiftAccessNowParams{WorkspaceID: ws.ID, Mode: mode}); err != nil {
		return out, err
	}
	return out, nil
}

// emails reads the three gift emails' templates and, while gift-ending
// isn't published, when its first reminder is due.
func (s *Service) emails(ctx context.Context, ws *store.Workspace, p *authz.Principal, mode string, out *GiftsView) error {
	me, err := s.Earn.MemberEmails(ctx, ws, p)
	if err != nil {
		return err
	}
	out.FromName, out.FromEmail, out.CategoryName, out.CategoryKind, out.RoutePrimary = me.FromName, me.FromEmail, me.CategoryName, me.CategoryKind, me.RoutePrimary
	var ending *GiftEmail
	for _, e := range me.Emails {
		if !strings.HasPrefix(e.Email, "gift_") {
			continue
		}
		out.Emails = append(out.Emails, GiftEmail{Email: e.Email, TemplateID: e.TemplateID, Template: e.Template, Name: e.Name, Status: e.Status,
			LiveVersion: e.LiveVersion, DraftVersion: e.DraftVersion})
		if e.Email == earn.EmailGiftEnding {
			ending = &out.Emails[len(out.Emails)-1]
		}
	}
	if ending == nil || ending.Status == "live" {
		return nil
	}
	first, err := s.Q.GiftFirstEnding(ctx, store.GiftFirstEndingParams{WorkspaceID: ws.ID, Mode: mode, Now: s.now()})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && first.AccessEnds == nil) {
		return nil
	}
	if err != nil {
		return err
	}
	out.Ending = &GiftEndingWarning{Due: first.AccessEnds.Add(-EndingBefore), Buyer: first.BuyerName, InPerson: first.InPerson,
		Recipient: firstNonEmpty(first.RecipientName, first.RedeemedEmail), TemplateID: ending.TemplateID}
	return nil
}

// List is a page of the Gifts table: a filter (all, scheduled,
// not_redeemed, redeemed, expired, refunded) and a search.
func (s *Service) List(ctx context.Context, ws *store.Workspace, filter, search string, pageNo int) (GiftList, error) {
	status, ok := dbStatus(filter)
	if !ok {
		return GiftList{}, invalid("status", text(ctx, "problem.filter", nil))
	}
	if pageNo < 1 {
		pageNo = 1
	}
	search = strings.TrimSpace(search)
	if len(search) > 100 {
		search = search[:100]
	}
	search = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(search)
	mode, _ := s.sellMode(ctx, ws)
	out := GiftList{Items: []GiftRow{}, Status: firstNonEmpty(filter, "all"), Page: pageNo, PageSize: PageSize}
	c, err := s.Q.GiftCounts(ctx, store.GiftCountsParams{WorkspaceID: ws.ID, Mode: mode, Search: search})
	if err != nil {
		return out, err
	}
	out.Counts = GiftCounts{All: c.AllGifts, Scheduled: c.Scheduled, NotRedeemed: c.NotRedeemed, Redeemed: c.Redeemed, Expired: c.Expired, Refunded: c.Refunded}
	switch status {
	case "":
		out.Total = c.AllGifts
	case StatusPaid:
		out.Total = c.Scheduled
	case StatusDelivered:
		out.Total = c.NotRedeemed
	case StatusRedeemed:
		out.Total = c.Redeemed
	case StatusExpired:
		out.Total = c.Expired
	case StatusRefunded:
		out.Total = c.Refunded
	}
	rows, err := s.Q.GiftList(ctx, store.GiftListParams{WorkspaceID: ws.ID, Mode: mode, Status: status, Search: search, Off: int32((pageNo - 1) * PageSize), Lim: PageSize})
	if err != nil {
		return out, err
	}
	dash := s.dashboard(ctx, ws, mode)
	for _, r := range rows {
		out.Items = append(out.Items, s.row(ctx, gift(r.EarnGift, r.TierName, r.ListIds, r.AccessEnds), dash))
	}
	return out, nil
}

// aGift is a gift with its tier's name and lists.
type aGift struct {
	store.EarnGift
	TierName   string
	ListIDs    []uuid.UUID
	AccessEnds *time.Time
}

func gift(g store.EarnGift, tier string, lists []uuid.UUID, ends *time.Time) aGift {
	return aGift{EarnGift: g, TierName: tier, ListIDs: lists, AccessEnds: ends}
}

// dashboard links payments to the provider's dashboard (nil without a
// client).
func (s *Service) dashboard(ctx context.Context, ws *store.Workspace, mode string) func(string) string {
	client, _, err := s.Earn.ClientFor(ctx, ws.ID, mode)
	if err != nil {
		return func(string) string { return "" }
	}
	return func(pi string) string {
		if pi == "" {
			return ""
		}
		return client.DashboardURL("payments", pi)
	}
}

// row is a gift as the table shows it.
func (s *Service) row(ctx context.Context, g aGift, dash func(string) string) GiftRow {
	r := GiftRow{ID: g.ID.String(), BoughtAt: g.CreatedAt, Buyer: g.BuyerName, BuyerEmail: g.BuyerEmail, InPerson: g.InPerson, RecipientName: g.RecipientName,
		RecipientEmail: g.RecipientEmail, Tier: g.TierName, Short: s.tierShort(ctx, g.TierName, g.ListIDs), Months: int(g.Months), AmountCents: g.AmountCents,
		Currency: g.Currency, Status: tableStatus(g.Status), DeliverAt: g.DeliverAt, Timezone: g.Timezone, DeliveredAt: g.DeliveredAt, RemindedAt: g.RemindedAt,
		LinkExpiresAt: g.LinkExpiresAt, RedeemedAt: g.RedeemedAt, AccessEnds: g.AccessEnds, RefundedAt: g.RefundedAt, EmailStatus: g.EmailStatus,
		PaymentURL: dash(g.PaymentIntent)}
	if g.PaidAt != nil {
		r.BoughtAt = *g.PaidAt
	}
	if r.Buyer == "" {
		r.Buyer = g.BuyerEmail
	}
	if g.Status == StatusDelivered && g.RemindedAt == nil && g.DeliveredAt != nil {
		at := g.DeliveredAt.Add(RemindAfter)
		r.RemindAt = &at
	}
	if g.Status == StatusRedeemed && g.RedeemedEmail != "" && (g.InPerson || !strings.EqualFold(g.RedeemedEmail, g.RecipientEmail)) {
		r.RedeemedBy = g.RedeemedEmail
	}
	return r
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
