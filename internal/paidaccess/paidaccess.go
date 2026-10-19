// Package paidaccess says who has paid access (Earn, design expansion X16)
// to the domains that choose recipients and content: paid-only campaigns
// ("Who on this list"), paid-only blocks (Visibility · Paid access), paid
// tokens and the contacts table. It reads Earn's tables the way segments
// read scores; internal/earn writes them.
//
// A contact has paid access while a live subscription, complimentary grant
// or gift hasn't ended and is trialing, active, complimentary or a gift, or
// past due while Stripe retries (unless the access rule paused it). A
// higher tier includes a lower tier's paid content: "Insider and above"
// means a tier ranked at least Insider's. Test paid members never count.
package paidaccess

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// Who on a campaign's lists receives it.
const (
	Everyone = ""
	Paid     = "paid" // paid members only
	Free     = "free" // free subscribers only (no paid access at all)
)

// The reader's personal Earn links: internal/earn fills them in per reader
// at send time, like the unsubscribe link, and never through click
// tracking; the composer's checks and preview treat them as such.
const (
	UpgradeURL = "upgrade_url"
	ManageURL  = "manage_subscription_url"
)

// IsReaderLink reports whether a Liquid variable is one of the reader's
// personal Earn links.
func IsReaderLink(key string) bool { return key == UpgradeURL || key == ManageURL }

// Audience is the paid part of campaigns.audience: who on the lists gets
// the campaign and, for paid members, the lowest tier (empty: any).
type Audience struct {
	Who  string
	Tier string
}

// Parse reads the paid part of a stored audience (both key styles).
func Parse(raw json.RawMessage) Audience {
	var in struct {
		Paid      string `json:"paid"`
		PaidTier  string `json:"paid_tier"`
		PaidTier2 string `json:"paidTier"`
	}
	_ = json.Unmarshal(raw, &in)
	a := Audience{Who: in.Paid, Tier: strings.TrimSpace(in.PaidTier)}
	if a.Tier == "" {
		a.Tier = strings.TrimSpace(in.PaidTier2)
	}
	switch a.Who {
	case Paid:
	case Free:
		a.Tier = ""
	default:
		return Audience{}
	}
	return a
}

// Set reports whether the audience keeps only paid members or only free
// subscribers.
func (a Audience) Set() bool { return a.Who != Everyone }

// noRank is a rank no tier has: a deleted tier's paid members are nobody.
const noRank = 1 << 20

// MinRank is the lowest access rank a paid audience or condition keeps: 1
// for any paid tier, the tier's rank for "tier and above", and a rank no
// tier has when the tier was deleted.
func MinRank(ctx context.Context, q *store.Queries, wsID uuid.UUID, tier string) (int, error) {
	if tier == "" {
		return 1, nil
	}
	id, ok := tierID(tier)
	if !ok {
		// Not a tier id: nobody has that tier.
		return noRank, nil
	}
	r, err := q.PaidAccessTierRank(ctx, store.PaidAccessTierRankParams{ID: id, WorkspaceID: wsID})
	if errors.Is(err, pgx.ErrNoRows) {
		return noRank, nil
	}
	if err != nil {
		return 0, err
	}
	return int(r), nil
}

func tierID(s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(s)
	return id, err == nil
}

// Member is a contact's paid access as an audience reads it.
type Member struct {
	Status string // trialing, active, past_due, complimentary, gift
	Kind   string // stripe, complimentary, gift
	Rank   int
}

// Keeper decides who a paid audience keeps, read once per count or send.
// A nil Keeper keeps everyone.
type Keeper struct {
	who string
	// members are the contacts with paid access: at the tier and above for
	// paid members, at any tier for free subscribers (who are the others).
	members map[uuid.UUID]Member
}

// Load reads who an audience keeps.
func Load(ctx context.Context, q *store.Queries, wsID uuid.UUID, a Audience) (*Keeper, error) {
	if !a.Set() {
		return nil, nil
	}
	rank := 1
	if a.Who == Paid {
		var err error
		if rank, err = MinRank(ctx, q, wsID, a.Tier); err != nil {
			return nil, err
		}
	}
	rows, err := q.PaidAccessRows(ctx, store.PaidAccessRowsParams{WorkspaceID: wsID, MinRank: int32(min(rank, noRank))})
	if err != nil {
		return nil, err
	}
	k := &Keeper{who: a.Who, members: make(map[uuid.UUID]Member, len(rows))}
	for _, r := range rows {
		k.members[r.ContactID] = Member{Status: r.Status, Kind: r.Kind, Rank: int(r.TierRank)}
	}
	return k, nil
}

// Keeps reports whether the audience keeps a contact.
func (k *Keeper) Keeps(id uuid.UUID) bool {
	if k == nil {
		return true
	}
	_, paid := k.members[id]
	if k.who == Free {
		return !paid
	}
	return paid
}

// Member is a kept paid member's access (false for free subscribers).
func (k *Keeper) Member(id uuid.UUID) (Member, bool) {
	if k == nil || k.who != Paid {
		return Member{}, false
	}
	m, ok := k.members[id]
	return m, ok
}

// Who is the audience's choice (Paid or Free).
func (k *Keeper) Who() string {
	if k == nil {
		return Everyone
	}
	return k.who
}

// ListCounts are the recipients of lists split by paid access.
type ListCounts struct {
	Everyone int64
	Paid     int64
	Free     int64
}

// CountLists counts the recipients of lists (minus excluded lists and
// suppressions) as the composer does: everyone, paid members at the tier
// and above, and free subscribers.
func CountLists(ctx context.Context, q *store.Queries, wsID uuid.UUID, lists, exclude []uuid.UUID, tier string) (ListCounts, error) {
	rank, err := MinRank(ctx, q, wsID, tier)
	if err != nil {
		return ListCounts{}, err
	}
	if lists == nil {
		lists = []uuid.UUID{}
	}
	if exclude == nil {
		exclude = []uuid.UUID{}
	}
	r, err := q.PaidAccessListCounts(ctx, store.PaidAccessListCountsParams{WorkspaceID: wsID, Lists: lists, ExcludeLists: exclude, MinRank: int32(rank)})
	if err != nil {
		return ListCounts{}, err
	}
	return ListCounts{Everyone: r.Everyone, Paid: r.Paid, Free: r.Free}, nil
}

// Tier is a tier in access order, for the pickers.
type Tier struct {
	ID   uuid.UUID
	Name string
	// Short is how readers name it: the name without its list's
	// ("Weekly Digest Insider" on Weekly Digest is "Insider").
	Short string
	Rank  int
	// Top is the highest tier: "Founding member" rather than "Founding
	// member and above".
	Top bool
}

// Tiers are the workspace's tiers (archived ones left out), lowest first.
func Tiers(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]Tier, error) {
	rows, err := q.PaidAccessTiers(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make([]Tier, 0, len(rows))
	top := 0
	for _, r := range rows {
		out = append(out, Tier{ID: r.ID, Name: r.Name, Short: ShortName(r.Name, r.ListNames), Rank: int(r.AccessRank)})
		top = max(top, int(r.AccessRank))
	}
	for i := range out {
		out[i].Top = out[i].Rank == top
	}
	return out, nil
}

// ShortName is how readers name a tier: its name without the list it
// belongs to ("Weekly Digest Insider" on Weekly Digest is "Insider").
func ShortName(name string, lists []string) string {
	for _, l := range lists {
		l = strings.TrimSpace(l)
		if l != "" && strings.HasPrefix(name, l+" ") && len(name) > len(l)+1 {
			return strings.TrimSpace(name[len(l)+1:])
		}
	}
	return name
}

// Fact is a contact's paid access, as personalization and conditions read
// it: the highest tier's subscription, grant or gift.
type Fact struct {
	Status      string // trialing, active, past_due, complimentary, gift
	Kind        string // stripe, complimentary, gift
	TierID      uuid.UUID
	Tier        string
	Rank        int
	Period      string // month, year, or empty for grants and gifts
	AmountCents int64
	Currency    string
	Discount    json.RawMessage
	StartedAt   time.Time
	// RenewsAt is the next renewal while it renews; EndsAt is when access
	// ends (a scheduled cancellation, a trial without a card, a grant's or
	// a gift's end).
	RenewsAt *time.Time
	EndsAt   *time.Time
	TrialEnd *time.Time
}

// Facts reads some contacts' paid access; contacts without any are left
// out.
func Facts(ctx context.Context, q *store.Queries, wsID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]Fact, error) {
	out := map[uuid.UUID]Fact{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.PaidAccessFacts(ctx, store.PaidAccessFactsParams{WorkspaceID: wsID, ContactIds: ids})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		f := Fact{Status: r.Status, Kind: r.Kind, Tier: r.TierName, Rank: int(r.TierRank), Period: r.Period, AmountCents: r.AmountCents, Currency: r.Currency,
			Discount: r.Discount, StartedAt: r.StartedAt, TrialEnd: r.TrialEnd}
		if r.TierID != nil {
			f.TierID = *r.TierID
		}
		switch {
		case r.Kind == "complimentary" || r.Kind == "gift":
			f.EndsAt = r.GrantUntil
			if f.EndsAt == nil && r.Kind == "gift" {
				f.EndsAt = r.CurrentPeriodEnd
			}
		case r.CancelAt != nil:
			f.EndsAt = r.CancelAt
		default:
			f.RenewsAt = r.CurrentPeriodEnd
			if r.Status == "trialing" && r.TrialEnd != nil {
				f.RenewsAt = r.TrialEnd
			}
		}
		out[r.ContactID] = f
	}
	return out, nil
}
