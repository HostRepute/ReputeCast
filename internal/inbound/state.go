package inbound

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// State is how a workspace receives mail, as the rest of ReputeCast reads
// it: Settings › Senders (reply handling), Send Readiness, Deliverability's
// DMARC card and the sending engine.
type State struct {
	// Route: the workspace has an inbound route.
	Route bool
	// Receiving: the route receives (its first message arrived and nothing
	// says it stopped).
	Receiving bool
	// Inbox: replies stay in ReputeCast's inbox (reply handling "inbox").
	Inbox bool
	// Domain and Local are where the route receives: a mailbox route's
	// local part, "" for a provider's.
	Domain, Local string
	// LastDMARC is when the last DMARC report arrived by email.
	LastDMARC *time.Time
	// Failing: the route stopped receiving; Detail is the sentence
	// Receiving mail shows about it (an MX record that moved, a mailbox
	// that refused the sign-in).
	Failing bool
	Detail  i18n.Text
}

// Box is where the route receives.
func (st State) Box() Box { return Box{Domain: st.Domain, Local: st.Local} }

// StateOf reads how the workspace receives mail now.
func StateOf(ctx context.Context, q *store.Queries, workspaceID uuid.UUID) (State, error) {
	r, err := q.InboundRouteGet(ctx, workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	ws, err := q.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return State{}, err
	}
	st := State{Route: true, Receiving: r.Status == "receiving", Inbox: replyHandling(ws.SendingDefaults) == "inbox", Domain: r.Domain, Local: r.MailboxLocal,
		Failing: r.Status == "failing", Detail: storedDetail(r.StatusDetail)}
	if at, err := q.InboundLastDMARC(ctx, workspaceID); err == nil {
		st.LastDMARC = &at
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return st, err
	}
	return st, nil
}
