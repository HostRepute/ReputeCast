package commerce

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// activity is one entry of a store's activity log: its facts, which the
// app words.
type activity struct {
	Kind         string     `json:"-"`
	Tone         string     `json:"-"`
	OrderID      *uuid.UUID `json:"-"`
	CheckoutID   *uuid.UUID `json:"-"`
	ContactID    *uuid.UUID `json:"-"`
	At           time.Time  `json:"-"`
	Order        string     `json:"order,omitempty"`
	Amount       int64      `json:"amount,omitempty"`
	Currency     string     `json:"currency,omitempty"`
	Customer     string     `json:"customer,omitempty"`
	Accepted     bool       `json:"accepted,omitempty"`
	Contact      bool       `json:"contact,omitempty"`
	Lists        []string   `json:"lists,omitempty"`
	Flows        []string   `json:"flows,omitempty"`
	Matched      string     `json:"matched,omitempty"`
	Credited     string     `json:"credited,omitempty"`
	CreditedKind string     `json:"creditedKind,omitempty"`
	CreditedStep int        `json:"creditedStep,omitempty"`
	Window       int        `json:"window,omitempty"`
	Recovered    bool       `json:"recovered,omitempty"`
	Topic        string     `json:"topic,omitempty"`
	Source       string     `json:"source,omitempty"`
	By           string     `json:"by,omitempty"`
	Missed       int        `json:"missed,omitempty"`
	Products     int        `json:"products,omitempty"`
	Flow         string     `json:"flow,omitempty"`
	Hours        int        `json:"hours,omitempty"`
	Polled       bool       `json:"polled,omitempty"`
	Reason       string     `json:"reason,omitempty"`
	Until        *time.Time `json:"until,omitempty"`
	Message      string     `json:"message,omitempty"`
}

func (s *Service) activity(ctx context.Context, q *store.Queries, st store.Store, a activity) error {
	at := a.At
	if at.IsZero() {
		at = s.now()
	}
	if a.Tone == "" {
		a.Tone = "neutral"
	}
	return q.CommerceActivityInsert(ctx, store.CommerceActivityInsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID, At: at,
		Kind: a.Kind, Tone: a.Tone, Data: mustJSON(a), OrderID: a.OrderID, CheckoutID: a.CheckoutID, ContactID: a.ContactID})
}

func activityItem(r store.StoreActivity) CommerceActivityItem {
	var a activity
	_ = json.Unmarshal(r.Data, &a)
	out := CommerceActivityItem{ID: r.ID.String(), At: r.At, Kind: r.Kind, Tone: r.Tone, Order: a.Order, Amount: a.Amount, Currency: a.Currency,
		Customer: a.Customer, Accepted: a.Accepted, Contact: a.Contact, Lists: a.Lists, Flows: a.Flows, Matched: a.Matched, Credited: a.Credited,
		CreditedKind: a.CreditedKind, CreditedStep: a.CreditedStep, Window: a.Window, Recovered: a.Recovered, Topic: a.Topic, Source: a.Source,
		By: a.By, Missed: a.Missed, Products: a.Products, Flow: a.Flow, Hours: a.Hours, Polled: a.Polled, Reason: a.Reason, Until: a.Until,
		Message: a.Message}
	if r.ContactID != nil {
		out.ContactID = r.ContactID.String()
	}
	if out.Lists == nil {
		out.Lists = []string{}
	}
	if out.Flows == nil {
		out.Flows = []string{}
	}
	return out
}

// Activity is a page of a store's activity log, newest first (30 days).
func (s *Service) Activity(ctx context.Context, ws *store.Workspace, id uuid.UUID, cursor string, kinds []string, limit int) (CommerceActivityList, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceActivityList{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if kinds == nil {
		kinds = []string{}
	}
	params := store.CommerceActivityParams{StoreID: st.ID, Kinds: kinds, Lim: int32(limit + 1)}
	if at, aid, ok := decodeCursor(cursor); ok {
		params.Before, params.BeforeID = &at, &aid
	}
	rows, err := s.Q.CommerceActivity(ctx, params)
	if err != nil {
		return CommerceActivityList{}, err
	}
	out := CommerceActivityList{Items: []CommerceActivityItem{}}
	for i, r := range rows {
		if i == limit {
			out.Next = encodeCursor(rows[i-1].At, rows[i-1].ID)
			break
		}
		out.Items = append(out.Items, activityItem(r))
	}
	return out, nil
}

func encodeCursor(at time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id.String()))
}

func decodeCursor(c string) (time.Time, uuid.UUID, bool) {
	if c == "" {
		return time.Time{}, uuid.Nil, false
	}
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, uuid.Nil, false
	}
	ts, ids, ok := strings.Cut(string(b), "|")
	if !ok {
		return time.Time{}, uuid.Nil, false
	}
	at, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, uuid.Nil, false
	}
	id, err := uuid.Parse(ids)
	if err != nil {
		return time.Time{}, uuid.Nil, false
	}
	return at, id, true
}
