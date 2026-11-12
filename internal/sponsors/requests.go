package sponsors

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Reviewing the sponsor page's requests (Earn › Sponsors › Requests): a
// person approves one (perhaps for another issue, slot or price), which
// holds the issue until the payment's deadline and emails the sponsor the
// link to pay, or declines it with a reason the sponsor reads.

// closedFor is how long declined and expired requests stay on the tab.
const closedFor = 30 * 24 * time.Hour

// maxReason is the longest reason for declining.
const maxReason = 500

// SponsorRequestItem is a request on the Requests tab.
type SponsorRequestItem struct {
	ID string `json:"id"`
	// Status: requested (waiting for a person), awaiting_payment
	// (approved, held until PaymentDueAt), booked (paid), declined,
	// expired or canceled.
	Status  string `json:"status" enum:"requested,awaiting_payment,booked,declined,expired,canceled"`
	Company string `json:"company"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	// Sponsor is the library's sponsor the request is filed under.
	Sponsor     *SponsorRef `json:"sponsor,omitempty"`
	SlotID      string      `json:"slotId"`
	SlotName    string      `json:"slotName"`
	ListName    string      `json:"listName"`
	SlotDeleted bool        `json:"slotDeleted"`
	Issue       int         `json:"issue"`
	IssueOn     time.Time   `json:"issueOn"`
	// IssueState is a request's issue: open, or held by another booking
	// (HeldBy, until HeldUntil when it's a hold or waits for a payment).
	IssueState  string     `json:"issueState" enum:"open,held"`
	HeldBy      string     `json:"heldBy,omitempty"`
	HeldUntil   *time.Time `json:"heldUntil,omitempty"`
	PriceCents  int64      `json:"priceCents"`
	Currency    string     `json:"currency"`
	SubmittedAt time.Time  `json:"submittedAt"`
	// PaymentDueAt is an approved request's deadline; PaidOn and
	// PaidSource its payment once booked.
	PaymentDueAt   *time.Time `json:"paymentDueAt,omitempty"`
	PaidOn         *time.Time `json:"paidOn,omitempty"`
	PaidSource     string     `json:"paidSource,omitempty" enum:"manual,stripe,"`
	DeclinedReason string     `json:"declinedReason,omitempty"`
	ClosedAt       *time.Time `json:"closedAt,omitempty"`
}

// SponsorRequestList is the Requests tab: the requests waiting for a
// person, the approved ones waiting for their payment, and the ones
// declined or expired in the last 30 days.
type SponsorRequestList struct {
	Items []SponsorRequestItem `json:"items"`
	// SelfServe: the sponsor page is on; PageURL is its address.
	SelfServe bool     `json:"selfServe"`
	PageURL   string   `json:"pageUrl"`
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// SponsorRequestIssue is an issue a request can be approved for.
type SponsorRequestIssue struct {
	Number  int       `json:"number"`
	IssueOn time.Time `json:"issueOn"`
	// Taken: another booking holds it (HeldBy, until HeldUntil when it's
	// a hold or waits for its payment).
	Taken     bool       `json:"taken"`
	HeldBy    string     `json:"heldBy,omitempty"`
	HeldUntil *time.Time `json:"heldUntil,omitempty"`
	// PaymentDueAt is the payment's deadline if approved now.
	PaymentDueAt time.Time `json:"paymentDueAt"`
}

// SponsorRequestFile is a file sent with a request.
type SponsorRequestFile struct {
	Kind      string `json:"kind" enum:"logo,image"`
	Name      string `json:"name"`
	SizeBytes int    `json:"sizeBytes"`
	URL       string `json:"url"`
}

// SponsorRequestSlot is a slot a request can be approved for, with its
// issues of the next twelve weeks.
type SponsorRequestSlot struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	ListName     string                `json:"listName"`
	PriceCents   int64                 `json:"priceCents"`
	Currency     string                `json:"currency"`
	PayDays      int                   `json:"payDays"`
	CreativeDays int                   `json:"creativeDays"`
	Issues       []SponsorRequestIssue `json:"issues"`
}

// SponsorRequestDetail is a request's panel: what the sponsor sent, the
// creative as the Sponsor block shows it, and what Approve can choose.
type SponsorRequestDetail struct {
	SponsorRequestItem
	Website string `json:"website"`
	Notes   string `json:"notes"`
	// Creative is the creative as sent; Label, Initials and Logo are what
	// the Sponsor block shows with it; Files the images sent with it.
	Creative SponsorCreative      `json:"creative"`
	Label    string               `json:"label"`
	Initials string               `json:"initials"`
	Logo     string               `json:"logo,omitempty"`
	Files    []SponsorRequestFile `json:"files"`
	// PayDays is the request's slot's time to pay; ApprovedBy and
	// ApprovedAt name the approval.
	PayDays    int        `json:"payDays"`
	ApprovedBy string     `json:"approvedBy,omitempty"`
	ApprovedAt *time.Time `json:"approvedAt,omitempty"`
	// Slots are where Approve can put it (the request's slot first).
	Slots     []SponsorRequestSlot  `json:"slots"`
	Activity  []SponsorBookingEvent `json:"activity"`
	CanManage bool                  `json:"canManage"`
	Helpers   []string              `json:"helpers"`
}

// ApproveInput approves a request, perhaps for another issue, slot or
// price.
type ApproveInput struct {
	SlotID     string `json:"slotId,omitempty" format:"uuid" doc:"The request's slot when absent."`
	Issue      int    `json:"issue,omitempty" minimum:"1" doc:"The request's issue when absent."`
	PriceCents *int64 `json:"priceCents,omitempty" minimum:"1" doc:"The request's price when absent."`
}

// DeclineInput declines a request with the reason the sponsor reads.
type DeclineInput struct {
	Reason string `json:"reason" minLength:"1" maxLength:"500"`
}

// reqRow is a self-serve booking with its slot and sponsor.
type reqRow = store.SponsorRequestsOfRow

func (s *Service) requestItem(ctx context.Context, ws *store.Workspace, r reqRow) SponsorRequestItem {
	d, _ := parseRequest(r.Request)
	it := SponsorRequestItem{ID: r.ID.String(), Status: r.Status, Company: firstNonEmpty(d.Company, r.SponsorName), Name: d.Name, Email: d.Email,
		SlotID: r.SlotID.String(), SlotName: r.SlotName, ListName: r.ListName, SlotDeleted: r.SlotDeleted, Issue: int(r.IssueNumber),
		IssueOn: noonOn(r.IssueOn, ws), IssueState: "open", PriceCents: r.PriceCents, Currency: r.Currency, SubmittedAt: r.CreatedAt, PaymentDueAt: r.PaymentDueAt,
		DeclinedReason: r.DeclinedReason}
	if it.Status == "sent" {
		it.Status = "booked"
	}
	if r.SponsorID != nil {
		it.Sponsor = refOf(*r.SponsorID, r.SponsorName, r.SponsorLogo)
	}
	if r.PaymentStatus == "paid" {
		it.PaidOn, it.PaidSource = noonPtr(r.PaidOn, ws), r.PaidSource
	}
	switch it.Status {
	case "requested":
		// Who holds the issue the request asks for, if anyone.
		if h, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: r.SlotID, IssueNumber: r.IssueNumber}); err == nil {
			it.IssueState = "held"
			if h.SponsorID != nil {
				if sp, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: *h.SponsorID, WorkspaceID: ws.ID}); err == nil {
					it.HeldBy = sp.Name
				}
			}
			switch h.Status {
			case "awaiting_payment":
				it.HeldUntil = h.PaymentDueAt
			case "hold":
				it.HeldUntil = noonPtr(h.HoldUntil, ws)
			}
		}
	case "declined", "expired", "canceled", "booked":
		at := r.UpdatedAt
		it.ClosedAt = &at
	}
	return it
}

// Requests is the Requests tab.
func (s *Service) Requests(ctx context.Context, ws *store.Workspace, p *authz.Principal) (SponsorRequestList, error) {
	out := SponsorRequestList{Items: []SponsorRequestItem{}, PageURL: s.pageURL(ctx, ws, "sponsor"), CanManage: p.Can(authz.SponsorsManage), Helpers: []string{}}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws.ID, authz.SponsorsManage, p.UserID, 3)
	}
	st, err := s.settings(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.SelfServe = st.on
	open, err := s.Q.SponsorRequestsOf(ctx, store.SponsorRequestsOfParams{WorkspaceID: ws.ID, Statuses: []string{"requested", "awaiting_payment"}, Since: time.Unix(0, 0)})
	if err != nil {
		return out, err
	}
	// Decided in the last 30 days: booked (paid), declined, expired or
	// canceled.
	closed, err := s.Q.SponsorRequestsOf(ctx, store.SponsorRequestsOfParams{WorkspaceID: ws.ID,
		Statuses: []string{"booked", "sent", "declined", "expired", "canceled"}, Since: s.now().Add(-closedFor)})
	if err != nil {
		return out, err
	}
	for _, r := range append(open, closed...) {
		out.Items = append(out.Items, s.requestItem(ctx, ws, r))
	}
	return out, nil
}

// requestRow reads one self-serve booking of the workspace.
func (s *Service) requestRow(ctx context.Context, ws *store.Workspace, id uuid.UUID) (reqRow, error) {
	r, err := s.Q.SponsorRequestGet(ctx, store.SponsorRequestGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return reqRow{}, mapErr(ctx, err, "request")
	}
	return reqRow(r), nil
}

// RequestDetail is a request's panel.
func (s *Service) RequestDetail(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SponsorRequestDetail, error) {
	r, err := s.requestRow(ctx, ws, id)
	if err != nil {
		return SponsorRequestDetail{}, err
	}
	d, _ := parseRequest(r.Request)
	it := s.requestItem(ctx, ws, r)
	cr := parseCreative(r.Creative)
	out := SponsorRequestDetail{SponsorRequestItem: it, Website: d.Website, Notes: d.Notes, Creative: cr, Initials: initials(it.Company),
		Label: text(ctx, "block.labelBy", map[string]any{"sponsor": firstNonEmpty(r.SponsorName, it.Company)}), Logo: cr.Logo,
		Files: s.requestFiles(ctx, ws, cr), Slots: []SponsorRequestSlot{}, Activity: []SponsorBookingEvent{}, CanManage: p.Can(authz.SponsorsManage),
		Helpers: []string{}}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws.ID, authz.SponsorsManage, p.UserID, 3)
	}
	if sl, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: r.SlotID, WorkspaceID: ws.ID}); err == nil {
		out.PayDays = int(sl.PayDays)
	}
	if r.Status == "requested" {
		if out.Slots, err = s.approveSlots(ctx, ws, r); err != nil {
			return out, err
		}
	}
	events, err := s.Q.SponsorBookingEvents(ctx, r.ID)
	if err != nil {
		return out, err
	}
	for _, e := range events {
		var data map[string]any
		_ = json.Unmarshal(e.Data, &data)
		if data == nil {
			data = map[string]any{}
		}
		out.Activity = append(out.Activity, SponsorBookingEvent{Kind: e.Kind, At: e.OccurredAt, Actor: e.ActorName, Data: data})
		if e.Kind == "approved" && out.ApprovedAt == nil {
			at := e.OccurredAt
			out.ApprovedBy, out.ApprovedAt = e.ActorName, &at
		}
	}
	return out, nil
}

// requestFiles are the images a request sent, by their media library
// rows (hidden there).
func (s *Service) requestFiles(ctx context.Context, ws *store.Workspace, cr SponsorCreative) []SponsorRequestFile {
	out := []SponsorRequestFile{}
	for _, f := range []struct{ kind, url string }{{"logo", cr.Logo}, {"image", cr.Image}} {
		rest, ok := strings.CutPrefix(f.url, "/media/")
		if !ok {
			continue
		}
		id, err := uuid.Parse(strings.SplitN(rest, "/", 2)[0])
		if err != nil {
			continue
		}
		if a, err := s.Q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: id}); err == nil {
			out = append(out, SponsorRequestFile{Kind: f.kind, Name: a.FileName, SizeBytes: int(a.SizeBytes), URL: f.url})
		}
	}
	return out
}

// approveSlots are the slots a request can be approved for, its own first,
// with the issues of the next twelve weeks whose payment could still
// arrive before their creative is due.
func (s *Service) approveSlots(ctx context.Context, ws *store.Workspace, r reqRow) ([]SponsorRequestSlot, error) {
	slots, err := s.Q.SponsorSlots(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	today := s.today(ws)
	last := today.AddDate(0, 0, openWeeks*7)
	ids := make([]uuid.UUID, 0, len(slots))
	for _, sl := range slots {
		ids = append(ids, sl.ID)
	}
	held, err := s.Q.SponsorIssuesHeld(ctx, store.SponsorIssuesHeldParams{SlotIds: ids, FromOn: today, ToOn: last.AddDate(0, 0, 366)})
	if err != nil {
		return nil, err
	}
	type holder struct {
		name  string
		until *time.Time
	}
	taken := map[uuid.UUID]map[int]holder{}
	for _, h := range held {
		if taken[h.SlotID] == nil {
			taken[h.SlotID] = map[int]holder{}
		}
		x := holder{name: h.SponsorName}
		switch h.Status {
		case "awaiting_payment":
			x.until = h.PaymentDueAt
		case "hold":
			x.until = noonPtr(h.HoldUntil, ws)
		}
		taken[h.SlotID][int(h.IssueNumber)] = x
	}
	now := s.now()
	out := []SponsorRequestSlot{}
	for _, sl := range slots {
		row := slotRow(sl)
		it := SponsorRequestSlot{ID: sl.ID.String(), Name: sl.Name, ListName: sl.ListName, PriceCents: sl.PriceCents, Currency: sl.Currency, PayDays: int(sl.PayDays),
			CreativeDays: int(sl.CreativeDays), Issues: []SponsorRequestIssue{}}
		for n := max(NextNumber(row, today), 1); ; n++ {
			on := IssueOn(row, n)
			if on.After(last) && (sl.ID != r.SlotID || n > int(r.IssueNumber)) {
				break
			}
			due := payDue(now, sl.PayDays, on, sl.CreativeDays, ws)
			if !due.After(now) {
				continue
			}
			h, isHeld := taken[sl.ID][n]
			it.Issues = append(it.Issues, SponsorRequestIssue{Number: n, IssueOn: noonOn(on, ws), Taken: isHeld, HeldBy: h.name, HeldUntil: h.until, PaymentDueAt: due})
		}
		if sl.ID == r.SlotID {
			out = append([]SponsorRequestSlot{it}, out...)
		} else {
			out = append(out, it)
		}
	}
	return out, nil
}

// Approve approves a request: the issue is held for the sponsor until the
// payment's deadline (pay_days from now, at the latest when the creative
// is due), and they're emailed the link to pay.
func (s *Service) Approve(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ApproveInput) (SponsorRequestDetail, error) {
	r, err := s.requestRow(ctx, ws, id)
	if err != nil {
		return SponsorRequestDetail{}, err
	}
	if r.Status != "requested" {
		return SponsorRequestDetail{}, problem(ctx, 409, api.CodeConflict, "problem.notRequested", nil)
	}
	slotID := r.SlotID
	if in.SlotID != "" {
		if slotID, err = uuid.Parse(in.SlotID); err != nil {
			return SponsorRequestDetail{}, invalid(ctx, "body.slotId", "problem.pickSlot", nil)
		}
	}
	slot, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: slotID, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return SponsorRequestDetail{}, invalid(ctx, "body.slotId", "problem.pickSlot", nil)
	}
	if err != nil {
		return SponsorRequestDetail{}, err
	}
	issue := int(r.IssueNumber)
	if in.Issue > 0 {
		issue = in.Issue
	}
	price := r.PriceCents
	currency := r.Currency
	if slotID != r.SlotID {
		price, currency = slot.PriceCents, slot.Currency
	}
	if in.PriceCents != nil {
		price = *in.PriceCents
	}
	if price <= 0 {
		return SponsorRequestDetail{}, invalid(ctx, "body.priceCents", "problem.pricePositive", nil)
	}
	row := store.SponsorSlot{ID: slot.ID, FirstNumber: slot.FirstNumber, FirstOn: slot.FirstOn, EveryDays: slot.EveryDays}
	on := IssueOn(row, issue)
	now := s.now()
	if on.Before(s.today(ws)) {
		return SponsorRequestDetail{}, problem(ctx, 409, api.CodeConflict, "problem.issuePast", map[string]any{"issue": issue})
	}
	due := payDue(now, slot.PayDays, on, slot.CreativeDays, ws)
	if !due.After(now) {
		return SponsorRequestDetail{}, problem(ctx, 409, api.CodeConflict, "problem.issueTooSoon", map[string]any{"issue": issue})
	}
	if _, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: slot.ID, IssueNumber: int32(issue)}); err == nil {
		return SponsorRequestDetail{}, issueTaken(ctx, issue)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return SponsorRequestDetail{}, err
	}
	cr := parseCreative(r.Creative)
	if cr.UTM.Campaign == "" || cr.UTM.Campaign == defaultUTM(ws, r.ListName, int(r.IssueNumber)).Campaign {
		cr.UTM = defaultUTM(ws, slot.ListName, issue)
	}
	crRaw, _ := json.Marshal(cr)
	creativeDue := on.AddDate(0, 0, -int(slot.CreativeDays))
	dueDay := localDay(due, ws)
	token, hash := authn.NewToken()
	sealed, err := s.sealToken(id, token)
	if err != nil {
		return SponsorRequestDetail{}, err
	}
	before := map[string]any{"status": r.Status, "slot": r.SlotName, "issue": r.IssueNumber, "priceCents": r.PriceCents}
	after := map[string]any{"status": "awaiting_payment", "slot": slot.Name, "issue": issue, "priceCents": price, "paymentDueAt": due.UTC().Format(time.RFC3339)}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.SponsorBookingApprove(ctx, store.SponsorBookingApproveParams{ID: id, WorkspaceID: ws.ID, SlotID: slot.ID, IssueNumber: int32(issue), IssueOn: on,
			PriceCents: price, Currency: currency, CampaignID: s.campaignOn(ctx, ws, slot.ListID, on), Creative: crRaw, CreativeDue: &creativeDue, PaymentDue: &dueDay,
			PaymentDueAt: due, StatusTokenHash: hash, At: now})
		if isUnique(err) {
			return issueTaken(ctx, issue)
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return problem(ctx, 409, api.CodeConflict, "problem.notRequested", nil)
		}
		if err := s.event(ctx, q, id, p, "approved", map[string]any{"issue": issue, "slot": slot.Name, "priceCents": price, "paymentDueAt": due.UTC().Format(time.RFC3339)}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsor_booking.approved", TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, issue, slot.Name), Before: before, After: after}); err != nil {
			return err
		}
		return s.queueMail(ctx, tx, MailArgs{WorkspaceID: ws.ID, BookingID: id, Email: MailApproved, Token: sealed})
	})
	if err != nil {
		return SponsorRequestDetail{}, err
	}
	return s.RequestDetail(ctx, ws, p, id)
}

func issueTaken(ctx context.Context, issue int) error {
	return problem(ctx, 409, api.CodeConflict, "problem.requestIssueTaken", map[string]any{"issue": issue})
}

// Decline declines a request; the sponsor is emailed the reason.
func (s *Service) Decline(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in DeclineInput) (SponsorRequestDetail, error) {
	reason := strings.TrimSpace(strings.ReplaceAll(in.Reason, "\r\n", "\n"))
	if reason == "" {
		return SponsorRequestDetail{}, invalid(ctx, "body.reason", "problem.reasonRequired", nil)
	}
	if utf8.RuneCountInString(reason) > maxReason {
		return SponsorRequestDetail{}, invalid(ctx, "body.reason", "problem.reasonLength", map[string]any{"max": maxReason})
	}
	r, err := s.requestRow(ctx, ws, id)
	if err != nil {
		return SponsorRequestDetail{}, err
	}
	if r.Status != "requested" {
		return SponsorRequestDetail{}, problem(ctx, 409, api.CodeConflict, "problem.notRequested", nil)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.SponsorBookingDecline(ctx, store.SponsorBookingDeclineParams{ID: id, WorkspaceID: ws.ID, Reason: reason, At: s.now()})
		if err != nil {
			return err
		}
		if n == 0 {
			return problem(ctx, 409, api.CodeConflict, "problem.notRequested", nil)
		}
		if err := s.event(ctx, q, id, p, "declined", map[string]any{"reason": reason}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsor_booking.declined", TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(r.ListName, int(r.IssueNumber), r.SlotName), Reason: reason, Before: map[string]any{"status": r.Status},
			After: map[string]any{"status": "declined"}}); err != nil {
			return err
		}
		return s.queueMail(ctx, tx, MailArgs{WorkspaceID: ws.ID, BookingID: id, Email: MailDeclined})
	})
	if err != nil {
		return SponsorRequestDetail{}, err
	}
	return s.RequestDetail(ctx, ws, p, id)
}
