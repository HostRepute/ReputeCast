package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// simData is what the demo transport planned for one event.
type simData struct {
	Type       string  `json:"type"`
	Code       string  `json:"code"`
	Diagnostic string  `json:"diagnostic"`
	Class      string  `json:"class"`
	UA         string  `json:"ua"`
	IP         string  `json:"ip"`
	Link       float64 `json:"link"`
}

type engagement struct {
	message uuid.UUID
	kind    string
	at      time.Time
	data    simData
	link    int
}

// PumpSim applies the demo transport's events that are due: delivery
// events through the same normalization as provider webhooks (event log,
// message status, counters, suppression list), opens and clicks through
// the tracking recorder (with its human-vs-machine classification). It
// returns how many it applied and when the next one is due.
func (e *Engine) PumpSim(ctx context.Context, now time.Time) (int, *time.Time, error) {
	var later []engagement
	n := 0
	err := sending.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		later, n = later[:0], 0
		q := store.New(tx)
		rows, err := q.SendingDueSimEvents(ctx, store.SendingDueSimEventsParams{Now: now, Lim: 500})
		if err != nil || len(rows) == 0 {
			return err
		}
		n = len(rows)
		ids := make([]uuid.UUID, 0, len(rows))
		eventIDs := make([]int64, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.MessageID)
			eventIDs = append(eventIDs, r.ID)
		}
		msgs, err := q.SendingSimMessages(ctx, ids)
		if err != nil {
			return err
		}
		byID := map[uuid.UUID]store.SendingSimMessagesRow{}
		for _, m := range msgs {
			byID[m.ID] = m
		}
		// Apply in counter-row order (campaign, sending provider, mailbox
		// provider, variant), the order the send loop locks them in.
		sort.SliceStable(rows, func(i, j int) bool {
			return cellBefore(byID[rows[i].MessageID], byID[rows[j].MessageID])
		})
		provs := map[uuid.UUID]store.Provider{}
		touched := map[uuid.UUID]time.Time{}
		links := map[string]int{}
		for _, r := range rows {
			m, ok := byID[r.MessageID]
			if !ok {
				continue
			}
			var d simData
			_ = json.Unmarshal(r.Data, &d)
			switch r.Kind {
			case "open", "click":
				g := engagement{message: m.ID, kind: r.Kind, at: r.DueAt, data: d}
				if r.Kind == "click" {
					cnt, err := e.linkCount(ctx, q, m, links)
					if err != nil {
						return err
					}
					if cnt == 0 {
						continue
					}
					g.link = min(int(math.Floor(d.Link*d.Link*float64(cnt))), cnt-1)
				}
				later = append(later, g)
				continue
			}
			p, ok := provs[r.ProviderID]
			if !ok {
				if p, err = q.ProvGetAny(ctx, r.ProviderID); err != nil {
					return err
				}
				provs[r.ProviderID] = p
			}
			pmid := ""
			if m.ProviderMessageID != nil {
				pmid = *m.ProviderMessageID
			}
			mid := m.ID
			ev := providers.Event{Type: providers.EventType(r.Kind), ProviderMessageID: pmid, MessageID: &mid, Recipient: m.Email, OccurredAt: r.DueAt,
				OriginalType: d.Type, OriginalCode: d.Code, Diagnostic: d.Diagnostic, BounceClass: d.Class}
			if e.Normalizer != nil {
				if _, err := e.Normalizer.Apply(ctx, tx, p, nil, "webhook", r.DueAt, ev); err != nil {
					return err
				}
			}
			if t, ok := touched[p.ID]; !ok || r.DueAt.After(t) {
				touched[p.ID] = r.DueAt
			}
			if m.Stream == "automation" && e.Automations != nil {
				kind := map[string]string{"delivered": "delivered", "hard_bounce": "bounced", "complaint": "complained"}[r.Kind]
				if kind != "" {
					if err := e.Automations.RecordMessageEvent(ctx, q, automations.MessageEvent{MessageID: m.ID, Type: kind, At: r.DueAt}); err != nil {
						return err
					}
				}
			}
		}
		for id, at := range touched {
			if err := q.ProvTouchLastEvent(ctx, store.ProvTouchLastEventParams{ID: id, At: at}); err != nil {
				return err
			}
		}
		return q.SendingDeleteSimEvents(ctx, eventIDs)
	})
	if err != nil {
		return 0, nil, err
	}
	if e.Recorder != nil && len(later) > 0 {
		evs := make([]sending.Engagement, 0, len(later))
		for _, g := range later {
			evs = append(evs, sending.Engagement{MessageID: g.message, Click: g.kind == "click", Link: g.link, UserAgent: g.data.UA, IP: g.data.IP, Method: "GET", At: g.at})
		}
		if err := e.Recorder.RecordEngagement(ctx, evs); err != nil {
			e.log().WarnContext(ctx, "simulated engagement", "err", err)
		}
	}
	// Clicks count once they settle (review AN-2), on this pump's clock.
	settle := e.settleClicks(ctx, now)
	if n > 0 {
		return n, nil, nil
	}
	next, err := e.Q.SendingNextSimEvent(ctx)
	if errors.Is(err, pgx.ErrNoRows) || next.IsZero() {
		return 0, settle, nil
	}
	if err != nil {
		return 0, nil, err
	}
	return 0, earliest(&next, settle), nil
}

// cellBefore orders messages by the campaign counter row they bump.
func cellBefore(a, b store.SendingSimMessagesRow) bool {
	ac, bc := uuid.Nil, uuid.Nil
	if a.CampaignID != nil {
		ac = *a.CampaignID
	}
	if b.CampaignID != nil {
		bc = *b.CampaignID
	}
	if c := bytes.Compare(ac[:], bc[:]); c != 0 {
		return c < 0
	}
	ap, bp := uuid.Nil, uuid.Nil
	if a.ProviderID != nil {
		ap = *a.ProviderID
	}
	if b.ProviderID != nil {
		bp = *b.ProviderID
	}
	if c := bytes.Compare(ap[:], bp[:]); c != 0 {
		return c < 0
	}
	if a.MailboxProvider != b.MailboxProvider {
		return a.MailboxProvider < b.MailboxProvider
	}
	return a.Variant < b.Variant
}

// linkCount is how many tracked links the message's email has.
func (e *Engine) linkCount(ctx context.Context, q *store.Queries, m store.SendingSimMessagesRow, cache map[string]int) (int, error) {
	key := ""
	switch {
	case m.CampaignID != nil:
		key = "c" + m.CampaignID.String()
	default:
		key = "m" + m.ID.String()
	}
	if n, ok := cache[key]; ok {
		return n, nil
	}
	n := 0
	if m.CampaignID != nil {
		ls, err := q.TrackingCampaignLinks(ctx, *m.CampaignID)
		if err != nil {
			return 0, err
		}
		n = len(ls)
	} else {
		tm, err := q.TrackingMessage(ctx, m.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		var ref contentRef
		_ = json.Unmarshal(tm.ContentRef, &ref)
		if ref.Email != uuid.Nil {
			ls, err := q.TrackingAutomationLinks(ctx, store.TrackingAutomationLinksParams{EmailID: ref.Email, Version: int32(ref.Version), Language: ref.Language})
			if err != nil {
				return 0, err
			}
			n = len(ls)
		}
	}
	cache[key] = n
	return n, nil
}

// contentRef is messages.content_ref for automation emails.
type contentRef struct {
	Email   uuid.UUID `json:"email"`
	Version int       `json:"version"`
	Subject string    `json:"subject,omitempty"`
	// Language is the language version sent ("" is the source).
	Language string `json:"language,omitempty"`
	// Ctx is what the flow's trigger passed to its emails (a store's
	// checkout or order).
	Ctx map[string]string `json:"ctx,omitempty"`
}
