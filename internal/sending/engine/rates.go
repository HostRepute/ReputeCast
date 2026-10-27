package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// Rates per mailbox provider during a send (review GS-13): a person who
// can send lowers or raises how fast one mailbox provider gets mail, for
// the rest of the send and on every route (the live monitor's "Adjust
// rates"). 0 is the sending providers' full rate. The send loop reads them
// again within seconds; a canary's are the main send's.

// maxRate is the highest rate a person can set per mailbox provider, in
// messages per second.
const maxRate = 1000

// SetRates changes a running or waiting send's rates per mailbox
// provider, audited with what they were.
func (e *Engine) SetRates(ctx context.Context, campaignID, by uuid.UUID, rates map[string]float64) error {
	now := e.now()
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if c.Status != "sending" && c.Status != "paused" {
			return stateErr("state.notSending", campaignArgs(c.Name, c.Status))
		}
		snd, err := q.SendingGetSendForUpdate(ctx, c.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return stateErr("state.noSend", campaignArgs(c.Name, c.Status))
		}
		if err != nil {
			return err
		}
		before := map[string]float64{}
		_ = json.Unmarshal(snd.Throttles, &before)
		after := maps.Clone(before)
		for _, k := range sortedKeys(rates) {
			if key, args := rateProblem(k, rates[k]); key != "" {
				return stateErr(key, args)
			}
			after[k] = math.Round(rates[k]*10) / 10
		}
		b, _ := json.Marshal(after)
		if err := q.SendingSetThrottles(ctx, store.SendingSetThrottlesParams{CampaignID: c.ID, Throttles: b, At: now}); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		ref, name := e.actorRef(ctx, q, by)
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.rates_changed", TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Actor: e.actorFor(ctx, ref, name), Before: map[string]any{"rates": before}, After: map[string]any{"rates": after},
			Metadata: map[string]any{"summary": ratesSummary(before, after), "status": c.Status}})
	})
}

// rateProblem is why a rate can't be set (a message key and its
// arguments), or "" when it can: an unknown mailbox provider, or a rate
// outside 0.1–1,000 /s that isn't 0.
func rateProblem(mbp string, v float64) (string, i18n.Args) {
	if !slices.Contains(routing.MailboxProviders, mbp) {
		return "state.rateMailbox", i18n.Args{"key": mbp}
	}
	if math.IsNaN(v) || v < 0 || v > maxRate || (v > 0 && v < 0.1) {
		return "state.rateRange", i18n.Args{"mailbox": mbp, "max": int64(maxRate)}
	}
	return "", nil
}

// ratesSummary says what changed ("Outlook 5 /s → 2 /s · Gmail full rate
// → 8 /s").
func ratesSummary(before, after map[string]float64) string {
	var parts []string
	for _, k := range routing.MailboxProviders {
		a, b := before[k], after[k]
		if a == b {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s → %s", mailboxName(k), rateLabel(a), rateLabel(b)))
	}
	if len(parts) == 0 {
		return "No change"
	}
	return strings.Join(parts, " · ")
}

func rateLabel(v float64) string {
	if v <= 0 {
		return "full rate"
	}
	return fmtPerSecond(v)
}
