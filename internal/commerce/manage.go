package commerce

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
)

// CommerceSettingsInput changes a store's settings; fields left out keep
// their value.
type CommerceSettingsInput struct {
	// ListID is the list customers who accepted marketing join ("" leaves
	// it as it is).
	ListID          *string `json:"listId,omitempty" maxLength:"36"`
	WriteBack       *bool   `json:"writeBack,omitempty"`
	TagLinks        *bool   `json:"tagLinks,omitempty"`
	AttributionDays *int    `json:"attributionDays,omitempty" minimum:"1" maximum:"30"`
	Products        *bool   `json:"products,omitempty"`
	Customers       *bool   `json:"customers,omitempty"`
	Orders          *bool   `json:"orders,omitempty"`
	Checkouts       *bool   `json:"checkouts,omitempty"`
}

// UpdateSettings changes what syncs, the list, consent write-back and
// attribution. Each kind of change is audited with its before and after.
func (s *Service) UpdateSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in CommerceSettingsInput) (CommerceStorePage, error) {
	l := i18n.LangFrom(ctx)
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceStorePage{}, err
	}
	listID := st.ListID
	if in.ListID != nil && *in.ListID != "" {
		lid, err := uuid.Parse(*in.ListID)
		if err != nil || !slices.ContainsFunc(s.listOptions(ctx, ws.ID), func(o CommerceListOption) bool { return o.ID == lid.String() }) {
			return CommerceStorePage{}, api.Invalid("body.listId", mListUnknown.Text(l, nil))
		}
		listID = &lid
	}
	writeBack := pick(in.WriteBack, st.WriteBack) && st.Platform == Shopify && slices.Contains(st.Scopes, shopify.OptionalScope)
	tag := pick(in.TagLinks, st.TagLinks)
	days := st.AttributionDays
	if in.AttributionDays != nil {
		days = int32(min(max(*in.AttributionDays, 1), 30))
	}
	params := store.CommerceStoreSetSettingsParams{ID: st.ID, ListID: listID, WriteBack: writeBack, TagLinks: tag, AttributionDays: days,
		SyncProducts: pick(in.Products, st.SyncProducts), SyncCustomers: pick(in.Customers, st.SyncCustomers), SyncOrders: pick(in.Orders, st.SyncOrders),
		SyncCheckouts: pick(in.Checkouts, st.SyncCheckouts), By: userID(p), At: s.now()}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.CommerceStoreSetSettings(ctx, params); err != nil {
			return err
		}
		log := func(action string, before, after map[string]any) error {
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "store", TargetID: st.ID.String(), TargetName: st.Name,
				Before: before, After: after})
		}
		if !sameID(listID, st.ListID) {
			if err := log("store.list_updated", map[string]any{"list": idString(st.ListID)}, map[string]any{"list": idString(listID)}); err != nil {
				return err
			}
		}
		if writeBack != st.WriteBack {
			if err := log("store.writeback_updated", map[string]any{"writeBack": st.WriteBack}, map[string]any{"writeBack": writeBack}); err != nil {
				return err
			}
		}
		if tag != st.TagLinks || days != st.AttributionDays {
			if err := log("store.attribution_updated", map[string]any{"tagLinks": st.TagLinks, "days": st.AttributionDays},
				map[string]any{"tagLinks": tag, "days": days}); err != nil {
				return err
			}
		}
		if params.SyncProducts != st.SyncProducts || params.SyncCustomers != st.SyncCustomers || params.SyncOrders != st.SyncOrders || params.SyncCheckouts != st.SyncCheckouts {
			if err := log("store.sync_updated",
				map[string]any{"products": st.SyncProducts, "customers": st.SyncCustomers, "orders": st.SyncOrders, "checkouts": st.SyncCheckouts},
				map[string]any{"products": params.SyncProducts, "customers": params.SyncCustomers, "orders": params.SyncOrders, "checkouts": params.SyncCheckouts}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return CommerceStorePage{}, err
	}
	s.forget(ws.ID)
	return s.Page(ctx, ws, p, st.ID)
}

func pick(v *bool, old bool) bool {
	if v == nil {
		return old
	}
	return *v
}

func sameID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func idString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// CommerceQueued says a sync was queued.
type CommerceQueued struct {
	Queued bool `json:"queued"`
}

// SyncNow queues a reconciliation ("Sync now").
func (s *Service) SyncNow(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (CommerceQueued, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceQueued{}, err
	}
	if st.Status != StatusConnected {
		return CommerceQueued{}, api.Conflict(mStoreNotConnected.Text(i18n.LangFrom(ctx), i18n.Args{"name": st.Name}))
	}
	by := ""
	if u := userID(p); u != nil {
		by = u.String()
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store.sync_started", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name}); err != nil {
			return err
		}
		if jc := s.jobs(); jc != nil {
			if _, err := jc.InsertTx(ctx, tx, ReconcileArgs{StoreID: st.ID.String(), By: by}, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return CommerceQueued{}, err
	}
	return CommerceQueued{Queued: true}, nil
}

// RegisterAgain registers a store's missing webhooks again ("Register
// again", "Register all again"): the polling stops, and the log says what
// it covered.
func (s *Service) RegisterAgain(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, all bool) (CommerceStorePage, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceStorePage{}, err
	}
	if st.Status != StatusConnected && st.Status != StatusImporting {
		return CommerceStorePage{}, api.Conflict(mStoreNotConnected.Text(i18n.LangFrom(ctx), i18n.Args{"name": st.Name}))
	}
	before, err := s.Q.CommerceWebhooks(ctx, st.ID)
	if err != nil {
		return CommerceStorePage{}, err
	}
	by := userID(p)
	if err := s.registerWebhooks(ctx, st, by, !all); err != nil {
		return CommerceStorePage{}, s.problem(ctx, st.Platform, err)
	}
	after, err := s.Q.CommerceWebhooks(ctx, st.ID)
	if err != nil {
		return CommerceStorePage{}, err
	}
	var back []string
	for _, b := range before {
		if b.Status != "missing" {
			continue
		}
		for _, a := range after {
			if a.Topic == b.Topic && a.Status == "registered" {
				back = append(back, a.Topic)
			}
		}
	}
	name := s.userName(ctx, by)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, t := range back {
			if err := s.activity(ctx, q, st, activity{Kind: "webhook_registered", Tone: "ok", Topic: t, By: name, Polled: true}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store.webhooks_registered", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, After: map[string]any{"topics": back, "all": all}})
	})
	if err != nil {
		return CommerceStorePage{}, err
	}
	return s.Page(ctx, ws, p, st.ID)
}

// CommerceDisconnectInput disconnects a store; DeleteData also deletes its
// products and orders, confirmed by typing the store's name.
type CommerceDisconnectInput struct {
	DeleteData bool   `json:"deleteData"`
	Confirm    string `json:"confirm,omitempty" maxLength:"200"`
}

// Disconnect removes ReputeCast's webhooks from the store, pauses the
// automations its triggers start (they say why), and keeps its products
// and orders unless DeleteData. Contacts the store added stay, with their
// consent.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in CommerceDisconnectInput) (CommerceStoreSummary, error) {
	l := i18n.LangFrom(ctx)
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceStoreSummary{}, err
	}
	if in.DeleteData && strings.TrimSpace(in.Confirm) != st.Name {
		return CommerceStoreSummary{}, api.Invalid("body.confirm", mDisconnectConfirm.Text(l, i18n.Args{"name": st.Name}))
	}
	// Best effort: a store that refused us has no webhooks of ours left.
	if st.RevokedAt == nil && len(st.CredentialsSealed) > 0 {
		if c, err := s.client(st); err == nil {
			if rows, err := s.Q.CommerceWebhooks(ctx, st.ID); err == nil {
				for _, r := range rows {
					if r.ExternalID != "" {
						_ = c.Unsubscribe(ctx, r.ExternalID)
					}
				}
			}
		}
	}
	by := userID(p)
	now := s.now()
	var flows []string
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if flows, err = s.pauseFlows(ctx, q, st); err != nil {
			return err
		}
		if err := q.CommerceStoreDisconnected(ctx, store.CommerceStoreDisconnectedParams{ID: st.ID, At: now, By: by}); err != nil {
			return err
		}
		if err := q.CommerceWebhooksClear(ctx, st.ID); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store.disconnected", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, After: map[string]any{"deleteData": in.DeleteData, "paused": flows}}); err != nil {
			return err
		}
		if err := s.activity(ctx, q, st, activity{Kind: "disconnected", Tone: "bad", By: s.userName(ctx, by), Flows: flows}); err != nil {
			return err
		}
		if !in.DeleteData {
			return nil
		}
		for _, del := range []func(context.Context, uuid.UUID) error{q.CommerceDeleteStoreCheckouts, q.CommerceDeleteStoreOrders, q.CommerceDeleteStoreData,
			q.CommerceDeleteStoreCustomers, q.CommerceDeleteStoreCollections} {
			if err := del(ctx, st.ID); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "store.data_deleted", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, Reason: "typed confirmation"})
	})
	if err != nil {
		return CommerceStoreSummary{}, err
	}
	s.forget(ws.ID)
	st, err = s.Q.CommerceStoreByID(ctx, st.ID)
	if err != nil {
		return CommerceStoreSummary{}, err
	}
	return s.summary(ctx, st), nil
}
