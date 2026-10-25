package counters

import (
	"bytes"
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

type hourKey struct {
	campaign uuid.UUID
	hour     int
}

type hourDelta struct{ human, privacy, machine, clicked int64 }

type linkKey struct {
	campaign uuid.UUID
	position int
}

type linkDelta struct{ clicks, unique, bot int64 }

// Ledger collects counter changes during a transaction and applies them
// in one lock order (Flush). The zero value is ready to use.
type Ledger struct {
	cells map[Cell]*Delta
	hours map[hourKey]*hourDelta
	links map[linkKey]*linkDelta
}

// Move records a message moving from one state to another: the counters
// of its cell and, when the campaign has started, its hourly engagement.
func (l *Ledger) Move(c Cell, started *time.Time, before, after State) {
	if c.Campaign == uuid.Nil || c.Provider == uuid.Nil {
		return
	}
	d := Diff(before, after)
	if !d.Zero() {
		if l.cells == nil {
			l.cells = map[Cell]*Delta{}
		}
		cur := l.cells[c]
		if cur == nil {
			cur = &Delta{}
			l.cells[c] = cur
		}
		cur.add(d)
	}
	if started == nil {
		return
	}
	hb, ha := before.HoursOf(*started), after.HoursOf(*started)
	l.hour(c.Campaign, hb.Human, func(h *hourDelta) { h.human-- })
	l.hour(c.Campaign, ha.Human, func(h *hourDelta) { h.human++ })
	l.hour(c.Campaign, hb.Privacy, func(h *hourDelta) { h.privacy-- })
	l.hour(c.Campaign, ha.Privacy, func(h *hourDelta) { h.privacy++ })
	l.hour(c.Campaign, hb.Machine, func(h *hourDelta) { h.machine-- })
	l.hour(c.Campaign, ha.Machine, func(h *hourDelta) { h.machine++ })
	l.hour(c.Campaign, hb.Clicked, func(h *hourDelta) { h.clicked-- })
	l.hour(c.Campaign, ha.Clicked, func(h *hourDelta) { h.clicked++ })
}

func (l *Ledger) hour(campaign uuid.UUID, h *int, fn func(*hourDelta)) {
	if h == nil {
		return
	}
	if l.hours == nil {
		l.hours = map[hourKey]*hourDelta{}
	}
	k := hourKey{campaign, *h}
	cur := l.hours[k]
	if cur == nil {
		cur = &hourDelta{}
		l.hours[k] = cur
	}
	fn(cur)
}

// Click records a click on a campaign's link: a person's (unique is their
// first click on that link) or a bot's.
func (l *Ledger) Click(campaign uuid.UUID, position int, bot, unique bool) {
	if campaign == uuid.Nil {
		return
	}
	if l.links == nil {
		l.links = map[linkKey]*linkDelta{}
	}
	k := linkKey{campaign, position}
	cur := l.links[k]
	if cur == nil {
		cur = &linkDelta{}
		l.links[k] = cur
	}
	switch {
	case bot:
		cur.bot++
	default:
		cur.clicks++
		if unique {
			cur.unique++
		}
	}
}

// Empty reports whether the ledger has nothing to apply.
func (l *Ledger) Empty() bool {
	return len(l.cells) == 0 && len(l.hours) == 0 && len(l.links) == 0
}

func uuidLess(a, b uuid.UUID) bool { return bytes.Compare(a[:], b[:]) < 0 }

// Flush applies what the ledger collected: counter cells, then hours, then
// links, each in key order, and empties it.
func (l *Ledger) Flush(ctx context.Context, q *store.Queries) error {
	if len(l.cells) > 0 {
		keys := make([]Cell, 0, len(l.cells))
		for k, d := range l.cells {
			if !d.Zero() {
				keys = append(keys, k)
			}
		}
		sort.Slice(keys, func(i, j int) bool {
			a, b := keys[i], keys[j]
			if a.Campaign != b.Campaign {
				return uuidLess(a.Campaign, b.Campaign)
			}
			if a.Provider != b.Provider {
				return uuidLess(a.Provider, b.Provider)
			}
			if a.Mailbox != b.Mailbox {
				return a.Mailbox < b.Mailbox
			}
			return a.Variant < b.Variant
		})
		if len(keys) > 0 {
			var p store.CountersBumpParams
			for _, k := range keys {
				d := l.cells[k]
				p.CampaignIds = append(p.CampaignIds, k.Campaign)
				p.ProviderIds = append(p.ProviderIds, k.Provider)
				p.Mailboxes = append(p.Mailboxes, k.Mailbox)
				p.Variants = append(p.Variants, k.Variant)
				p.Sent = append(p.Sent, d.Sent)
				p.Delivered = append(p.Delivered, d.Delivered)
				p.Deferred = append(p.Deferred, d.Deferred)
				p.Hard = append(p.Hard, d.HardBounced)
				p.Soft = append(p.Soft, d.SoftBounced)
				p.Complained = append(p.Complained, d.Complained)
				p.Rejected = append(p.Rejected, d.Rejected)
				p.Human = append(p.Human, d.OpenedHuman)
				p.Privacy = append(p.Privacy, d.OpenedPrivacy)
				p.Machine = append(p.Machine, d.OpenedMachine)
				p.Clicked = append(p.Clicked, d.Clicked)
				p.Unsubscribed = append(p.Unsubscribed, d.Unsubscribed)
			}
			if err := q.CountersBump(ctx, p); err != nil {
				return err
			}
		}
	}
	if len(l.hours) > 0 {
		keys := make([]hourKey, 0, len(l.hours))
		for k, d := range l.hours {
			if *d != (hourDelta{}) {
				keys = append(keys, k)
			}
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].campaign != keys[j].campaign {
				return uuidLess(keys[i].campaign, keys[j].campaign)
			}
			return keys[i].hour < keys[j].hour
		})
		if len(keys) > 0 {
			var p store.CountersBumpHourlyParams
			for _, k := range keys {
				d := l.hours[k]
				p.CampaignIds = append(p.CampaignIds, k.campaign)
				p.Hours = append(p.Hours, int32(k.hour))
				p.Human = append(p.Human, d.human)
				p.Privacy = append(p.Privacy, d.privacy)
				p.Machine = append(p.Machine, d.machine)
				p.Clicked = append(p.Clicked, d.clicked)
			}
			if err := q.CountersBumpHourly(ctx, p); err != nil {
				return err
			}
		}
	}
	if len(l.links) > 0 {
		keys := make([]linkKey, 0, len(l.links))
		for k := range l.links {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].campaign != keys[j].campaign {
				return uuidLess(keys[i].campaign, keys[j].campaign)
			}
			return keys[i].position < keys[j].position
		})
		var p store.CountersBumpLinksParams
		for _, k := range keys {
			d := l.links[k]
			p.CampaignIds = append(p.CampaignIds, k.campaign)
			p.Positions = append(p.Positions, int32(k.position))
			p.Clicks = append(p.Clicks, d.clicks)
			p.UniqueClicks = append(p.UniqueClicks, d.unique)
			p.BotClicks = append(p.BotClicks, d.bot)
		}
		if err := q.CountersBumpLinks(ctx, p); err != nil {
			return err
		}
	}
	l.cells, l.hours, l.links = nil, nil, nil
	return nil
}

// Moved is a message's move as CountersMoveDelivery and
// CountersMoveEngagement return it.
type Moved struct {
	Cell    Cell
	Started *time.Time
	Before  State
	After   State
}

// FromDelivery reads a CountersMoveDelivery row.
func FromDelivery(r store.CountersMoveDeliveryRow) Moved {
	m := Moved{Started: r.CampaignStartedAt}
	m.Cell = cellOf(r.CampaignID, r.ProviderID, r.MailboxProvider, r.Variant)
	m.Before = State{Status: r.OldStatus, SentAt: r.OldSentAt, DeliveredAt: r.OldDeliveredAt, DeferredAt: r.OldDeferredAt, SoftBouncedAt: r.OldSoftBouncedAt,
		OpenedAt: r.OldOpenedAt, HumanOpenedAt: r.OldHumanOpenedAt, PrivacyOpenedAt: r.OldPrivacyOpenedAt, ClickedAt: r.OldClickedAt, UnsubscribedAt: r.OldUnsubscribedAt}
	m.After = State{Status: r.NewStatus, SentAt: r.NewSentAt, DeliveredAt: r.NewDeliveredAt, DeferredAt: r.NewDeferredAt, SoftBouncedAt: r.NewSoftBouncedAt,
		OpenedAt: r.NewOpenedAt, HumanOpenedAt: r.NewHumanOpenedAt, PrivacyOpenedAt: r.NewPrivacyOpenedAt, ClickedAt: r.NewClickedAt, UnsubscribedAt: r.NewUnsubscribedAt}
	return m
}

// FromEngagement reads a CountersMoveEngagement row.
func FromEngagement(r store.CountersMoveEngagementRow) Moved {
	return FromDelivery(store.CountersMoveDeliveryRow(r))
}

func cellOf(campaign, provider *uuid.UUID, mailbox, variant string) Cell {
	c := Cell{Mailbox: mailbox, Variant: variant}
	if campaign != nil {
		c.Campaign = *campaign
	}
	if provider != nil {
		c.Provider = *provider
	}
	return c
}

// Add records a move.
func (l *Ledger) Add(m Moved) { l.Move(m.Cell, m.Started, m.Before, m.After) }
