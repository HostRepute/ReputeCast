package sponsors

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// SlotInput creates or changes a slot.
type SlotInput struct {
	ListID     string `json:"listId" format:"uuid"`
	Name       string `json:"name" minLength:"1" maxLength:"60"`
	Placement  string `json:"placement,omitempty" maxLength:"120" doc:"Where it sits in the issue (\"After the first story\")."`
	PriceCents int64  `json:"priceCents" minimum:"0" doc:"The default price of a booking, in minor units."`
	Currency   string `json:"currency,omitempty" maxLength:"3" doc:"ISO 4217; the workspace's currency when absent."`
	// The issues: numbered from FirstNumber on FirstOn, every EveryDays.
	FirstNumber  int    `json:"firstNumber" minimum:"1" maximum:"100000"`
	FirstOn      string `json:"firstOn" format:"date" doc:"The date of issue FirstNumber, in the workspace's time zone."`
	EveryDays    int    `json:"everyDays,omitempty" minimum:"1" maximum:"31" doc:"7 when absent (weekly)."`
	CreativeDays *int   `json:"creativeDays,omitempty" minimum:"0" maximum:"30" doc:"The creative is due this many days before the issue (4 when absent)."`
	Selling      *bool  `json:"selling,omitempty"`
}

func slotItem(r store.SponsorSlotsRow, ws *store.Workspace, bookings int) SponsorSlotItem {
	it := SponsorSlotItem{ID: r.ID.String(), Name: r.Name, ListID: r.ListID.String(), ListName: r.ListName, Placement: r.Placement, PriceCents: r.PriceCents,
		Currency: r.Currency, FirstNumber: int(r.FirstNumber), FirstOn: noonOn(r.FirstOn, ws), EveryDays: int(r.EveryDays), CreativeDays: int(r.CreativeDays),
		Selling: r.Selling, Bookings: bookings}
	if r.EveryDays == 7 {
		it.Weekday = civil(r.FirstOn).Weekday().String()
	}
	return it
}

// slotOf turns a single-slot row into the list row's shape.
func slotOf(r store.SponsorSlotGetRow) store.SponsorSlotsRow {
	return store.SponsorSlotsRow(r)
}

func (s *Service) cleanSlot(ctx context.Context, ws *store.Workspace, in SlotInput) (store.SponsorSlotInsertParams, error) {
	out := store.SponsorSlotInsertParams{Name: strings.TrimSpace(in.Name), Placement: strings.TrimSpace(in.Placement), PriceCents: in.PriceCents,
		FirstNumber: int32(in.FirstNumber), EveryDays: int32(in.EveryDays), CreativeDays: 4, Selling: true}
	if out.Name == "" {
		return out, invalid(ctx, "body.name", "problem.slotName", nil)
	}
	lid, err := uuid.Parse(in.ListID)
	if err != nil {
		return out, invalid(ctx, "body.listId", "problem.slotList", nil)
	}
	var ok bool
	_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM lists WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL)`, lid, ws.ID).Scan(&ok)
	if !ok {
		return out, invalid(ctx, "body.listId", "problem.slotList", nil)
	}
	out.ListID = lid
	d, err := time.Parse("2006-01-02", in.FirstOn)
	if err != nil {
		return out, invalid(ctx, "body.firstOn", "problem.slotDate", nil)
	}
	out.FirstOn = d
	if out.EveryDays == 0 {
		out.EveryDays = 7
	}
	if in.CreativeDays != nil {
		out.CreativeDays = int32(*in.CreativeDays)
	}
	if in.Selling != nil {
		out.Selling = *in.Selling
	}
	out.Currency = strings.ToLower(strings.TrimSpace(in.Currency))
	if out.Currency == "" {
		out.Currency = s.currency(ctx, ws)
	}
	return out, nil
}

// currency is the workspace's currency (Earn's, else US dollars).
func (s *Service) currency(ctx context.Context, ws *store.Workspace) string {
	var c string
	_ = s.Pool.QueryRow(ctx, `SELECT currency FROM earn_settings WHERE workspace_id = $1`, ws.ID).Scan(&c)
	if c == "" {
		return "usd"
	}
	return c
}

// CreateSlot adds a slot.
func (s *Service) CreateSlot(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SlotInput) (SponsorSlotItem, error) {
	row, err := s.cleanSlot(ctx, ws, in)
	if err != nil {
		return SponsorSlotItem{}, err
	}
	row.ID, row.WorkspaceID, row.CreatedBy, row.At = newID(), ws.ID, userID(p), s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorSlotInsert(ctx, row); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.slot_created", TargetType: "sponsor_slot", TargetID: row.ID.String(),
			TargetName: row.Name, After: map[string]any{"list": row.ListID.String(), "placement": row.Placement, "priceCents": row.PriceCents,
				"firstNumber": row.FirstNumber, "firstOn": row.FirstOn.Format("2006-01-02"), "everyDays": row.EveryDays, "creativeDays": row.CreativeDays}})
	})
	if err != nil {
		return SponsorSlotItem{}, err
	}
	return s.slotItemByID(ctx, ws, row.ID)
}

// UpdateSlot changes a slot; booked issues keep their dates and prices.
func (s *Service) UpdateSlot(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SlotInput) (SponsorSlotItem, error) {
	old, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return SponsorSlotItem{}, api.MapError(err, "Slot")
	}
	row, err := s.cleanSlot(ctx, ws, in)
	if err != nil {
		return SponsorSlotItem{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorSlotUpdate(ctx, store.SponsorSlotUpdateParams{ID: id, WorkspaceID: ws.ID, ListID: row.ListID, Name: row.Name, Placement: row.Placement,
			PriceCents: row.PriceCents, Currency: row.Currency, FirstNumber: row.FirstNumber, FirstOn: row.FirstOn, EveryDays: row.EveryDays,
			CreativeDays: row.CreativeDays, Selling: row.Selling, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.slot_updated", TargetType: "sponsor_slot", TargetID: id.String(),
			TargetName: row.Name,
			Before: map[string]any{"name": old.Name, "placement": old.Placement, "priceCents": old.PriceCents, "firstNumber": old.FirstNumber,
				"firstOn": old.FirstOn.Format("2006-01-02"), "everyDays": old.EveryDays, "creativeDays": old.CreativeDays, "selling": old.Selling},
			After: map[string]any{"name": row.Name, "placement": row.Placement, "priceCents": row.PriceCents, "firstNumber": row.FirstNumber,
				"firstOn": row.FirstOn.Format("2006-01-02"), "everyDays": row.EveryDays, "creativeDays": row.CreativeDays, "selling": row.Selling}})
	})
	if err != nil {
		return SponsorSlotItem{}, err
	}
	return s.slotItemByID(ctx, ws, id)
}

// DeleteSlot removes a slot with no upcoming bookings.
func (s *Service) DeleteSlot(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	old, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Slot")
	}
	var upcoming int
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM sponsor_bookings WHERE slot_id = $1 AND status IN ('hold','booked') AND issue_on >= $2`, id, s.today(ws)).Scan(&upcoming)
	if upcoming > 0 {
		return problem(ctx, 409, api.CodeConflict, "problem.slotHasBookings", map[string]any{"name": old.Name, "count": upcoming})
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorSlotDelete(ctx, store.SponsorSlotDeleteParams{ID: id, WorkspaceID: ws.ID, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.slot_deleted", TargetType: "sponsor_slot", TargetID: id.String(),
			TargetName: old.Name, Before: map[string]any{"list": old.ListName, "priceCents": old.PriceCents}})
	})
}

func (s *Service) slotItemByID(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SponsorSlotItem, error) {
	r, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return SponsorSlotItem{}, api.MapError(err, "Slot")
	}
	var n int
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM sponsor_bookings WHERE slot_id = $1 AND status IN ('booked','sent')`, id).Scan(&n)
	return slotItem(slotOf(r), ws, n), nil
}
