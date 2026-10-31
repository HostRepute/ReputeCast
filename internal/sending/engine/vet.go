package engine

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// Mail only moves to a provider that can take it (review GS-5): a person's
// move, the guard's automatic move and the failover while an audience is
// built vet the target with state.Vet, as the provider screens do.

// senderVerification is a campaign sender's verification per provider ID,
// nil when the campaign has no sender.
func senderVerification(ctx context.Context, q *store.Queries, ws uuid.UUID, senderID *uuid.UUID) map[string]string {
	if senderID == nil {
		return nil
	}
	s, err := q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws, ID: *senderID})
	if err != nil {
		return nil
	}
	return state.SenderVerification(s.Verification)
}

// vetTarget checks whether a provider can take need more messages of a
// campaign.
func (e *Engine) vetTarget(ctx context.Context, q *store.Queries, sender map[string]string, p store.Provider, need int64, loc *time.Location) state.Vetting {
	return state.Vet(p, state.Load{Broadcast: true, Messages: need, Sender: sender, Warmup: warmup.Today(ctx, q, p.ID, e.now(), loc)})
}

// vetErr is why a person's move was refused, in their language.
func vetErr(c state.VetCheck, provider string) error {
	t := c.Keep(provider)
	return &StateError{Msg: t.In(i18n.Default), text: t}
}
