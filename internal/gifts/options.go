package gifts

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

func invalid(location, message string) *api.Problem { return api.Invalid(location, message) }

// notFound is a 404 in the reader's language: what is gift or option.
func notFound(ctx context.Context, what string) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, text(ctx, "problem.notFound."+what, nil))
}

// GiftOptionInput is a gift option as the editor sends it.
type GiftOptionInput struct {
	TierID      string `json:"tierId" format:"uuid"`
	Months      int    `json:"months" minimum:"1" maximum:"36" doc:"How long the gift lasts"`
	AmountCents int64  `json:"amountCents" minimum:"1" doc:"The one-time price, in the workspace currency"`
}

// clean checks an option and finds its tier.
func (s *Service) cleanOption(ctx context.Context, ws *store.Workspace, id uuid.UUID, in GiftOptionInput) (uuid.UUID, error) {
	tier, err := uuid.Parse(in.TierID)
	if err != nil {
		return uuid.Nil, invalid("body.tierId", text(ctx, "problem.tier", nil))
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tier, WorkspaceID: ws.ID})
	if err != nil || t.Visibility == "archived" {
		return uuid.Nil, invalid("body.tierId", text(ctx, "problem.tier", nil))
	}
	if in.Months < 1 || in.Months > 36 {
		return uuid.Nil, invalid("body.months", text(ctx, "problem.months", nil))
	}
	if in.AmountCents < 1 || in.AmountCents > 100_000_000 {
		return uuid.Nil, invalid("body.amountCents", text(ctx, "problem.amount", nil))
	}
	taken, err := s.Q.GiftOptionTaken(ctx, store.GiftOptionTakenParams{WorkspaceID: ws.ID, TierID: tier, Months: int32(in.Months), ID: id})
	if err != nil {
		return uuid.Nil, err
	}
	if taken {
		return uuid.Nil, invalid("body.months", text(ctx, "problem.optionTaken", i18n.Args{"months": in.Months}))
	}
	return tier, nil
}

// CreateOption adds a gift option (earn:edit).
func (s *Service) CreateOption(ctx context.Context, ws *store.Workspace, p *authz.Principal, in GiftOptionInput) (GiftOption, error) {
	tier, err := s.cleanOption(ctx, ws, uuid.Nil, in)
	if err != nil {
		return GiftOption{}, err
	}
	cur := s.Earn.PageSettingsOf(ctx, ws.ID).Currency
	id := newID()
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.GiftOptionInsert(ctx, store.GiftOptionInsertParams{ID: id, WorkspaceID: ws.ID, TierID: tier, Months: int32(in.Months), AmountCents: in.AmountCents,
			Currency: cur, CreatedBy: userID(p), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.gift_option_created", TargetType: "earn_gift_option", TargetID: id.String(),
			TargetName: s.optionName(ctx, ws, tier, in.Months), After: map[string]any{"tier": tier.String(), "months": in.Months, "amountCents": in.AmountCents, "currency": cur}})
	})
	if err != nil {
		return GiftOption{}, err
	}
	return s.option(ctx, ws, id)
}

// UpdateOption changes a gift option (earn:edit). Gifts already bought
// keep what they were sold as.
func (s *Service) UpdateOption(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in GiftOptionInput) (GiftOption, error) {
	before, err := s.Q.GiftOptionGet(ctx, store.GiftOptionGetParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return GiftOption{}, notFound(ctx, "option")
	}
	if err != nil {
		return GiftOption{}, err
	}
	tier, err := s.cleanOption(ctx, ws, id, in)
	if err != nil {
		return GiftOption{}, err
	}
	cur := s.Earn.PageSettingsOf(ctx, ws.ID).Currency
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.GiftOptionUpdate(ctx, store.GiftOptionUpdateParams{TierID: tier, Months: int32(in.Months), AmountCents: in.AmountCents, Currency: cur, At: now,
			ID: id, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.gift_option_updated", TargetType: "earn_gift_option", TargetID: id.String(),
			TargetName: s.optionName(ctx, ws, tier, in.Months),
			Before:     map[string]any{"tier": before.TierID.String(), "months": before.Months, "amountCents": before.AmountCents, "currency": before.Currency},
			After:      map[string]any{"tier": tier.String(), "months": in.Months, "amountCents": in.AmountCents, "currency": cur}})
	})
	if err != nil {
		return GiftOption{}, err
	}
	return s.option(ctx, ws, id)
}

// DeleteOption removes a gift option from the gift page (earn:edit).
// Gifts already bought keep working.
func (s *Service) DeleteOption(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	before, err := s.Q.GiftOptionGet(ctx, store.GiftOptionGetParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(ctx, "option")
	}
	if err != nil {
		return err
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.GiftOptionDelete(ctx, store.GiftOptionDeleteParams{At: now, ID: id, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.gift_option_deleted", TargetType: "earn_gift_option", TargetID: id.String(),
			TargetName: s.optionName(ctx, ws, before.TierID, int(before.Months)),
			Before:     map[string]any{"tier": before.TierID.String(), "months": before.Months, "amountCents": before.AmountCents, "currency": before.Currency}})
	})
}

// optionName is "Weekly Digest Insider · 1 year" for the audit log.
func (s *Service) optionName(ctx context.Context, ws *store.Workspace, tier uuid.UUID, months int) string {
	name := ""
	if t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tier, WorkspaceID: ws.ID}); err == nil {
		name = t.Name
	}
	return english("optionName", i18n.Args{"tier": name, "months": months})
}

// option reads one option as the tab lists it.
func (s *Service) option(ctx context.Context, ws *store.Workspace, id uuid.UUID) (GiftOption, error) {
	mode, _ := s.sellMode(ctx, ws)
	all, err := s.options(ctx, ws, mode, s.now().AddDate(0, 0, -30))
	if err != nil {
		return GiftOption{}, err
	}
	for _, o := range all {
		if o.ID == id.String() {
			return o, nil
		}
	}
	return GiftOption{}, notFound(ctx, "option")
}

// GiftPageInput turns the gift page on or off and sets how long gift links
// work.
type GiftPageInput struct {
	On         bool `json:"on"`
	LinkMonths int  `json:"linkMonths" enum:"3,6,12" doc:"How long a gift link works after delivery"`
}

// SetPage changes the gift page (earn:edit). Gifts already bought keep
// working when it's off: their emails go out and their links can be
// redeemed.
func (s *Service) SetPage(ctx context.Context, ws *store.Workspace, p *authz.Principal, in GiftPageInput) (GiftPage, error) {
	if !slices.Contains(LinkMonths, in.LinkMonths) {
		return GiftPage{}, invalid("body.linkMonths", text(ctx, "problem.linkMonths", nil))
	}
	before, err := s.pageOf(ctx, ws)
	if err != nil {
		return GiftPage{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		at, by := now, userID(p)
		if before.On == in.On {
			// The switch didn't move: keep who moved it last.
			if before.ChangedAt != nil {
				at = *before.ChangedAt
			}
			st, err := q.EarnSettings(ctx, ws.ID)
			if err != nil {
				return err
			}
			by = st.GiftPageBy
		}
		if err := q.GiftPageSet(ctx, store.GiftPageSetParams{GiftPageOn: in.On, At: at, By: by, GiftLinkMonths: int32(in.LinkMonths), WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.gift_page_updated", TargetType: "earn_gift_page", TargetID: ws.ID.String(),
			TargetName: before.Host, Before: map[string]any{"on": before.On, "linkMonths": before.LinkMonths}, After: map[string]any{"on": in.On, "linkMonths": in.LinkMonths}})
	})
	if err != nil {
		return GiftPage{}, err
	}
	return s.pageOf(ctx, ws)
}
