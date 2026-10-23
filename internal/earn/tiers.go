package earn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/text/unicode/norm"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Visibilities of a tier.
const (
	VisibilityPublic   = "public"
	VisibilityHidden   = "hidden"
	VisibilityArchived = "archived"
)

// EarnListRef names a list.
type EarnListRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EarnPrice is one of a tier's prices.
type EarnPrice struct {
	ID          string `json:"id"`
	Period      string `json:"period" enum:"month,year"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	// Status: current takes new paid members; legacy keeps the ones who pay it.
	Status       string `json:"status" enum:"current,legacy"`
	StripeLiveID string `json:"stripeLiveId,omitempty"`
	StripeTestID string `json:"stripeTestId,omitempty"`
	// StripeID is the id in the mode public pages use, and URL its page.
	StripeID  string `json:"stripeId,omitempty"`
	StripeURL string `json:"stripeUrl,omitempty"`
	Origin    string `json:"origin" enum:"tier,mapped"`
	// Members pay this price (paying and trialing), Discounted of them with an offer.
	Members    int64 `json:"members"`
	Paying     int64 `json:"paying"`
	Discounted int64 `json:"discounted"`
	// Offers are the paying members with an offer, by the offer's label
	// ("45 with FALL26"), most first.
	Offers []EarnPriceOffer `json:"offers"`
}

// EarnPriceOffer is how many paying members of a price have an offer.
type EarnPriceOffer struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// EarnSyncStep is one step of saving a tier in the provider.
type EarnSyncStep struct {
	Step   string     `json:"step" enum:"product,month,year"`
	Status string     `json:"status" enum:"done,failed,pending"`
	At     *time.Time `json:"at,omitempty"`
	ID     string     `json:"id,omitempty"`
	// Amount is the price's amount (month and year steps).
	AmountCents int64  `json:"amountCents,omitempty"`
	Error       string `json:"error,omitempty"`
	Tries       int    `json:"tries,omitempty"`
}

// EarnTier is a tier: what readers pay for.
type EarnTier struct {
	ID string `json:"id"`
	// Name is the full name; ShortName how readers say it ("Insider").
	Name        string        `json:"name"`
	ShortName   string        `json:"shortName"`
	Slug        string        `json:"slug"`
	Description string        `json:"description"`
	Benefits    []string      `json:"benefits"`
	TrialDays   int           `json:"trialDays" enum:"0,7,14,30"`
	Lists       []EarnListRef `json:"lists"`
	AccessRank  int           `json:"accessRank"`
	Visibility  string        `json:"visibility" enum:"public,hidden,archived"`
	// Highlight is the label above the card on the upgrade page ("Most
	// popular"), when this tier is the highlighted one.
	Highlight string `json:"highlight,omitempty"`
	// Monthly and Yearly are the current prices.
	Monthly *EarnPrice  `json:"monthly,omitempty"`
	Yearly  *EarnPrice  `json:"yearly,omitempty"`
	Prices  []EarnPrice `json:"prices"`
	// Paying are the paying members (active and past due), by period.
	Paying        int64 `json:"paying"`
	PayingMonthly int64 `json:"payingMonthly"`
	PayingYearly  int64 `json:"payingYearly"`
	Trialing      int64 `json:"trialing"`
	// MRRCents is the tier's monthly recurring revenue (paying members
	// only, after discounts, a yearly price as a twelfth).
	MRRCents int64 `json:"mrrCents"`
	// IncludesBelow names the tiers of lower rank whose paid content it includes.
	Includes []string `json:"includes"`
	// The product in the provider (in the mode public pages use).
	ProductID  string         `json:"productId,omitempty"`
	ProductURL string         `json:"productUrl,omitempty"`
	SyncStatus string         `json:"syncStatus" enum:"draft,synced,failed"`
	SyncSteps  []EarnSyncStep `json:"syncSteps"`
	SyncedAt   *time.Time     `json:"syncedAt,omitempty"`
	// Origin is "import" for a tier whose product was mapped when paid
	// members were imported: from ImportedFrom ("substack") on ImportedAt.
	Origin           string     `json:"origin,omitempty"`
	ImportedFrom     string     `json:"importedFrom,omitempty"`
	ImportedAt       *time.Time `json:"importedAt,omitempty"`
	HiddenLink       string     `json:"hiddenLink"`
	CreatedAt        time.Time  `json:"createdAt"`
	ArchivedAt       *time.Time `json:"archivedAt,omitempty"`
	CanDelete        bool       `json:"canDelete"`
	DiscountedYearly int64      `json:"discountedYearly"`
}

// tierFacts are the prices and paying members of every tier.
type tierFacts struct {
	prices map[uuid.UUID][]store.EarnPrice
	counts map[uuid.UUID]map[string]int64 // price → status → n
	disc   map[uuid.UUID]int64
	offers map[uuid.UUID][]EarnPriceOffer
	lists  map[uuid.UUID]string
	mrr    map[uuid.UUID]int64
	mode   string
}

func (s *Service) loadFacts(ctx context.Context, ws *store.Workspace, mode string) (tierFacts, error) {
	f := tierFacts{prices: map[uuid.UUID][]store.EarnPrice{}, counts: map[uuid.UUID]map[string]int64{}, disc: map[uuid.UUID]int64{}, lists: map[uuid.UUID]string{},
		mrr: map[uuid.UUID]int64{}, offers: map[uuid.UUID][]EarnPriceOffer{}, mode: mode}
	prices, err := s.Q.EarnPrices(ctx, ws.ID)
	if err != nil {
		return f, err
	}
	for _, p := range prices {
		f.prices[p.TierID] = append(f.prices[p.TierID], p)
	}
	if mode == "" {
		mode = "live"
	}
	counts, err := s.Q.EarnPriceCounts(ctx, store.EarnPriceCountsParams{WorkspaceID: ws.ID, Mode: mode})
	if err != nil {
		return f, err
	}
	for _, c := range counts {
		if c.PriceID == nil {
			continue
		}
		if f.counts[*c.PriceID] == nil {
			f.counts[*c.PriceID] = map[string]int64{}
		}
		f.counts[*c.PriceID][c.Status] += c.N
		f.disc[*c.PriceID] += c.Discounted
	}
	lists, err := s.Q.EarnLists(ctx, ws.ID)
	if err != nil {
		return f, err
	}
	for _, l := range lists {
		f.lists[l.ID] = l.Name
	}
	rows, err := s.Pool.Query(ctx, `SELECT tier_id, round(coalesce(sum(CASE WHEN period = 'year' THEN coalesce((discount->>'amountCents')::bigint, amount_cents) / 12.0
		ELSE coalesce((discount->>'amountCents')::bigint, amount_cents) END), 0))::bigint
		FROM paid_subscriptions WHERE workspace_id = $1 AND mode = $2 AND kind = 'stripe' AND ended_at IS NULL AND status IN ('active','past_due') AND tier_id IS NOT NULL
		GROUP BY tier_id`, ws.ID, mode)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var n int64
		if err := rows.Scan(&id, &n); err != nil {
			return f, err
		}
		f.mrr[id] = n
	}
	if err := rows.Err(); err != nil {
		return f, err
	}
	rows.Close()
	offers, err := s.Pool.Query(ctx, `SELECT price_id, discount->>'label', count(*) FROM paid_subscriptions
		WHERE workspace_id = $1 AND mode = $2 AND kind = 'stripe' AND ended_at IS NULL AND status IN ('active','past_due') AND price_id IS NOT NULL
		AND coalesce(discount->>'label', '') <> '' GROUP BY 1, 2 ORDER BY 3 DESC, 2`, ws.ID, mode)
	if err != nil {
		return f, err
	}
	defer offers.Close()
	for offers.Next() {
		var id uuid.UUID
		var o EarnPriceOffer
		if err := offers.Scan(&id, &o.Label, &o.Count); err != nil {
			return f, err
		}
		f.offers[id] = append(f.offers[id], o)
	}
	return f, offers.Err()
}

func (s *Service) tierView(ctx context.Context, ws *store.Workspace, t store.EarnTier, f tierFacts, page UpgradePage, all []store.EarnTier) EarnTier {
	listNames := []string{}
	lists := []EarnListRef{}
	for _, id := range t.ListIds {
		if n, ok := f.lists[id]; ok {
			lists = append(lists, EarnListRef{ID: id.String(), Name: n})
			listNames = append(listNames, n)
		}
	}
	mode := payments.Mode(firstNonEmpty(f.mode, "live"))
	out := EarnTier{ID: t.ID.String(), Name: t.Name, ShortName: shortTier(t.Name, listNames), Slug: t.Slug, Description: t.Description, Benefits: t.Benefits,
		TrialDays: int(t.TrialDays), Lists: lists, AccessRank: int(t.AccessRank), Visibility: t.Visibility, Prices: []EarnPrice{}, Includes: []string{},
		SyncStatus: t.SyncStatus, SyncSteps: []EarnSyncStep{}, SyncedAt: t.SyncedAt, Origin: t.Origin, CreatedAt: t.CreatedAt, ArchivedAt: t.ArchivedAt}
	if out.Benefits == nil {
		out.Benefits = []string{}
	}
	_ = json.Unmarshal(t.SyncSteps, &out.SyncSteps)
	if out.SyncSteps == nil {
		out.SyncSteps = []EarnSyncStep{}
	}
	for i := range out.SyncSteps {
		out.SyncSteps[i].Error = syncErrorIn(lang(ctx), out.SyncSteps[i].Error)
	}
	if page.Highlight == t.ID.String() {
		out.Highlight = page.HighlightLabel
	}
	out.ProductID = t.StripeProductLive
	if mode == payments.Test {
		out.ProductID = t.StripeProductTest
	}
	if out.ProductID != "" {
		out.ProductURL = dashboard(DefaultProvider, mode, "products", out.ProductID)
	}
	out.HiddenLink = s.pageHost(ctx, ws, "upgrade/"+t.Slug)
	for _, p := range f.prices[t.ID] {
		c := f.counts[p.ID]
		ep := EarnPrice{ID: p.ID.String(), Period: p.Period, AmountCents: p.AmountCents, Currency: p.Currency, Status: p.Status, StripeLiveID: p.StripeLiveID,
			StripeTestID: p.StripeTestID, Origin: p.Origin, Paying: c[StatusActive] + c[StatusPastDue], Discounted: f.disc[p.ID], Offers: f.offers[p.ID]}
		if ep.Offers == nil {
			ep.Offers = []EarnPriceOffer{}
		}
		ep.Members = ep.Paying + c[StatusTrialing]
		ep.StripeID = p.StripeLiveID
		if mode == payments.Test {
			ep.StripeID = p.StripeTestID
		}
		if ep.StripeID != "" {
			ep.StripeURL = dashboard(DefaultProvider, mode, "prices", ep.StripeID)
		}
		out.Prices = append(out.Prices, ep)
		out.Paying += ep.Paying
		out.Trialing += c[StatusTrialing]
		if p.Period == "month" {
			out.PayingMonthly += ep.Paying
		} else {
			out.PayingYearly += ep.Paying
			out.DiscountedYearly += ep.Discounted
		}
		if p.Status == "current" {
			cp := ep
			if p.Period == "month" {
				out.Monthly = &cp
			} else {
				out.Yearly = &cp
			}
		}
	}
	out.MRRCents = f.mrr[t.ID]
	for _, o := range all {
		if o.ID != t.ID && o.AccessRank < t.AccessRank && o.DeletedAt == nil {
			names := []string{}
			for _, id := range o.ListIds {
				names = append(names, f.lists[id])
			}
			out.Includes = append(out.Includes, shortTier(o.Name, names))
		}
	}
	return out
}

// EarnUpgradeCard is the upgrade page's card on Plans & offers.
type EarnUpgradeCard struct {
	URL  string `json:"url"`
	Host string `json:"host"`
	// Live: a public tier can be bought (a payment provider is connected).
	Live             bool  `json:"live"`
	TestMode         bool  `json:"testMode"`
	Visits           int64 `json:"visits"`
	CheckoutsStarted int64 `json:"checkoutsStarted"`
	Completed        int64 `json:"completed"`
	Trials           int64 `json:"trials"`
	PaidAtOnce       int64 `json:"paidAtOnce"`
	// TestCheckoutURL opens the page in test mode for a day (team members).
	TestCheckoutURL string `json:"testCheckoutUrl,omitempty"`
}

// EarnFreeCard is the free list's card.
type EarnFreeCard struct {
	List    *EarnListRef `json:"list,omitempty"`
	Line    string       `json:"line"`
	Text    string       `json:"text"`
	Readers int64        `json:"readers"`
}

// EarnPlans is Earn › Plans & offers: the upgrade page, the free card and
// the tiers (offers and gift options come from X17's section).
type EarnPlans struct {
	UpgradePage EarnUpgradeCard `json:"upgradePage"`
	Free        EarnFreeCard    `json:"free"`
	Tiers       []EarnTier      `json:"tiers"`
	Currency    string          `json:"currency"`
	Connected   bool            `json:"connected"`
	CanEdit     bool            `json:"canEdit"`
	Helpers     []string        `json:"helpers"`
	Lists       []EarnListRef   `json:"lists"`
	// Brand colours the tier editor's preview as readers see the card.
	Brand Brand `json:"brand"`
}

// Plans is Plans & offers.
func (s *Service) Plans(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnPlans, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return EarnPlans{}, err
	}
	act, connectedNow := active(conns)
	mode := "live"
	if connectedNow {
		mode = act.Mode
	}
	f, err := s.loadFacts(ctx, ws, mode)
	if err != nil {
		return EarnPlans{}, err
	}
	page, err := s.upgradePage(ctx, ws)
	if err != nil {
		return EarnPlans{}, err
	}
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return EarnPlans{}, err
	}
	out := EarnPlans{Tiers: []EarnTier{}, Connected: connectedNow, CanEdit: p.Can(authz.EarnEdit), Helpers: s.helpers(ctx, ws.ID, authz.EarnEdit, p.UserID), Lists: []EarnListRef{}}
	cs, _ := s.checkoutSettings(ctx, ws, conns)
	out.Currency = cs.Currency
	public := false
	for _, t := range tiers {
		v := s.tierView(ctx, ws, t, f, page, tiers)
		out.Tiers = append(out.Tiers, v)
		if t.Visibility == VisibilityPublic && (v.Monthly != nil || v.Yearly != nil) {
			public = true
		}
	}
	for id, n := range f.lists {
		out.Lists = append(out.Lists, EarnListRef{ID: id.String(), Name: n})
	}
	slices.SortFunc(out.Lists, func(a, b EarnListRef) int { return strings.Compare(a.Name, b.Name) })
	since := s.now().AddDate(0, 0, -30)
	card := EarnUpgradeCard{URL: s.pageURL(ctx, ws, "upgrade"), Host: s.pageHost(ctx, ws, "upgrade"), Live: public && connectedNow, TestMode: connectedNow && act.Mode == "test"}
	card.Visits, _ = s.Q.EarnPageVisits(ctx, store.EarnPageVisitsParams{WorkspaceID: ws.ID, Page: "upgrade", Since: since})
	card.TestCheckoutURL = s.TestCheckoutURL(ctx, ws)
	if c, err := s.Q.EarnCheckoutCounts(ctx, store.EarnCheckoutCountsParams{WorkspaceID: ws.ID, Since: since}); err == nil {
		card.CheckoutsStarted, card.Completed, card.Trials = c.Started, c.Completed, c.Trials
		card.PaidAtOnce = c.Completed - c.Trials
	}
	out.Brand = s.brand(ctx, ws)
	out.UpgradePage = card
	out.Free = EarnFreeCard{Line: page.FreeLine, Text: page.FreeText}
	if id, err := uuid.Parse(page.FreeList); err == nil {
		if n, ok := f.lists[id]; ok {
			out.Free.List = &EarnListRef{ID: id.String(), Name: n}
			_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM list_memberships WHERE list_id = $1 AND status = 'subscribed'`, id).Scan(&out.Free.Readers)
		}
	}
	return out, nil
}

// TierDetail is the tier editor's data.
func (s *Service) TierDetail(ctx context.Context, ws *store.Workspace, id string) (EarnTier, error) {
	tid, err := uuid.Parse(id)
	if err != nil {
		return EarnTier{}, notFound(ctx, "tier")
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tid, WorkspaceID: ws.ID})
	if err != nil {
		return EarnTier{}, mapErr(ctx, err, "tier")
	}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return EarnTier{}, err
	}
	mode := "live"
	if c, ok := active(conns); ok {
		mode = c.Mode
	}
	f, err := s.loadFacts(ctx, ws, mode)
	if err != nil {
		return EarnTier{}, err
	}
	page, err := s.upgradePage(ctx, ws)
	if err != nil {
		return EarnTier{}, err
	}
	all, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return EarnTier{}, err
	}
	out := s.tierView(ctx, ws, t, f, page, all)
	if t.Origin == "import" {
		var at time.Time
		if err := s.Pool.QueryRow(ctx, `SELECT source, created_at FROM earn_imports WHERE workspace_id = $1 AND status = 'done' ORDER BY created_at LIMIT 1`, ws.ID).
			Scan(&out.ImportedFrom, &at); err == nil {
			out.ImportedAt = &at
		}
	}
	n, _ := s.Q.EarnTierSubscriptions(ctx, &t.ID)
	out.CanDelete = n == 0
	return out, nil
}

// EarnTierInput creates or changes a tier.
type EarnTierInput struct {
	Name        string   `json:"name" minLength:"1" maxLength:"80"`
	Slug        string   `json:"slug,omitempty" maxLength:"60" doc:"The hidden tier's link: /upgrade/{slug}. Made from the name when empty."`
	Description string   `json:"description,omitempty" maxLength:"500"`
	Benefits    []string `json:"benefits" maxItems:"12"`
	// Monthly and Yearly are amounts in minor units ($8 = 800); 0 or
	// omitted turns the period off.
	Monthly    int64    `json:"monthly,omitempty" minimum:"0" maximum:"10000000"`
	Yearly     int64    `json:"yearly,omitempty" minimum:"0" maximum:"100000000"`
	TrialDays  int      `json:"trialDays" enum:"0,7,14,30"`
	Lists      []string `json:"lists" maxItems:"20"`
	AccessRank int      `json:"accessRank" minimum:"1" maximum:"20"`
	Visibility string   `json:"visibility" enum:"public,hidden,archived"`
	// Highlight puts the tier forward on the upgrade page with this label
	// ("Most popular"); empty takes the highlight off it.
	Highlight string `json:"highlight,omitempty" maxLength:"30"`
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugFrom(name string) string {
	s := strings.ToLower(norm.NFKD.String(name))
	s = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, s)
	s = strings.Trim(slugRe.ReplaceAllString(s, "-"), "-")
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if s == "" {
		s = "tier"
	}
	return s
}

// tierCurrency is the currency new prices are in: the workspace's, else
// the connected account's default.
func (s *Service) tierCurrency(ctx context.Context, ws *store.Workspace) (string, error) {
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return "", err
	}
	if st.Currency != "" {
		return st.Currency, nil
	}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return "", err
	}
	if c, ok := active(conns); ok && c.Currency != "" {
		return c.Currency, nil
	}
	for _, c := range conns {
		if c.Currency != "" {
			return c.Currency, nil
		}
	}
	return "usd", nil
}

func (s *Service) validateTier(ctx context.Context, ws *store.Workspace, in *EarnTierInput, id uuid.UUID) ([]uuid.UUID, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, invalid(ctx, "body.name", "tier.name", nil)
	}
	if in.Monthly <= 0 && in.Yearly <= 0 {
		return nil, invalid(ctx, "body.monthly", "tier.noPrice", nil)
	}
	if in.Slug = strings.TrimSpace(in.Slug); in.Slug == "" {
		in.Slug = slugFrom(shortTierFromLists(ctx, s, ws, in))
	} else {
		in.Slug = slugFrom(in.Slug)
	}
	if taken, err := s.Q.EarnTierSlugTaken(ctx, store.EarnTierSlugTakenParams{WorkspaceID: ws.ID, Slug: in.Slug, ID: id}); err != nil {
		return nil, err
	} else if taken {
		return nil, invalid(ctx, "body.slug", "tier.slugTaken", i18n.Args{"slug": in.Slug})
	}
	benefits := []string{}
	for _, b := range in.Benefits {
		if b = strings.TrimSpace(b); b != "" {
			benefits = append(benefits, clip(b, 160))
		}
	}
	in.Benefits = benefits
	lists := []uuid.UUID{}
	known, err := s.Q.EarnLists(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, l := range in.Lists {
		lid, err := uuid.Parse(l)
		if err != nil || !slices.ContainsFunc(known, func(k store.EarnListsRow) bool { return k.ID == lid }) {
			return nil, invalid(ctx, "body.lists", "tier.list", nil)
		}
		if !slices.Contains(lists, lid) {
			lists = append(lists, lid)
		}
	}
	if in.AccessRank == 0 {
		in.AccessRank = 1
	}
	if in.Visibility == "" {
		in.Visibility = VisibilityPublic
	}
	return lists, nil
}

// shortTierFromLists makes the slug from the tier's short name ("insider").
func shortTierFromLists(ctx context.Context, s *Service, ws *store.Workspace, in *EarnTierInput) string {
	ids := []uuid.UUID{}
	for _, l := range in.Lists {
		if id, err := uuid.Parse(l); err == nil {
			ids = append(ids, id)
		}
	}
	names, _ := s.Q.EarnListNames(ctx, ids)
	return shortTier(in.Name, names)
}

// EarnTierSaved is a saved tier and, when a provider is connected, how
// saving it there went.
type EarnTierSaved struct {
	Tier EarnTier `json:"tier"`
	// SyncError says what the provider answered when a step failed; the
	// tier is saved either way and Retry creates only what's missing.
	SyncError string `json:"syncError,omitempty"`
}

// CreateTier creates a tier and its product and prices in the provider.
func (s *Service) CreateTier(ctx context.Context, ws *store.Workspace, p *authz.Principal, in EarnTierInput) (EarnTierSaved, error) {
	id := newID()
	lists, err := s.validateTier(ctx, ws, &in, id)
	if err != nil {
		return EarnTierSaved{}, err
	}
	cur, err := s.tierCurrency(ctx, ws)
	if err != nil {
		return EarnTierSaved{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnTierInsert(ctx, store.EarnTierInsertParams{ID: id, WorkspaceID: ws.ID, Name: in.Name, Slug: in.Slug, Description: in.Description,
			Benefits: in.Benefits, TrialDays: int32(in.TrialDays), ListIds: lists, AccessRank: int32(in.AccessRank), Visibility: in.Visibility, CreatedBy: userID(p), At: now}); err != nil {
			return err
		}
		for _, pr := range []struct {
			period string
			amount int64
		}{{"month", in.Monthly}, {"year", in.Yearly}} {
			if pr.amount <= 0 {
				continue
			}
			if err := q.EarnPriceInsert(ctx, store.EarnPriceInsertParams{ID: newID(), WorkspaceID: ws.ID, TierID: id, Period: pr.period, AmountCents: pr.amount,
				Currency: cur, Status: "current", Origin: "tier", At: now}); err != nil {
				return err
			}
		}
		if err := q.EarnSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.EarnSettingsSetCurrency(ctx, store.EarnSettingsSetCurrencyParams{WorkspaceID: ws.ID, Currency: cur}); err != nil {
			return err
		}
		if err := s.setHighlight(ctx, q, ws, id, in.Highlight, true); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.tier_created", TargetType: "earn_tier", TargetID: id.String(), TargetName: in.Name,
			After: tierAudit(in, cur)})
	})
	if err != nil {
		return EarnTierSaved{}, err
	}
	return s.afterTierSave(ctx, ws, id)
}

func tierAudit(in EarnTierInput, cur string) map[string]any {
	m := map[string]any{"name": in.Name, "trial_days": in.TrialDays, "visibility": in.Visibility, "access_rank": in.AccessRank, "benefits": len(in.Benefits),
		"currency": cur, "lists": in.Lists}
	if in.Monthly > 0 {
		m["monthly"] = priceText(in.Monthly, cur, "month")
	}
	if in.Yearly > 0 {
		m["yearly"] = priceText(in.Yearly, cur, "year")
	}
	if in.Highlight != "" {
		m["highlight"] = in.Highlight
	}
	return m
}

// afterTierSave saves the tier in the provider and returns it.
func (s *Service) afterTierSave(ctx context.Context, ws *store.Workspace, id uuid.UUID) (EarnTierSaved, error) {
	out := EarnTierSaved{}
	if msg, err := s.syncTier(ctx, ws, id); err != nil {
		return out, err
	} else {
		out.SyncError = syncErrorIn(lang(ctx), msg)
	}
	t, err := s.TierDetail(ctx, ws, id.String())
	out.Tier = t
	return out, err
}

// UpdateTier changes a tier. A new amount is a new price: the old one
// becomes a legacy price its paid members keep paying.
func (s *Service) UpdateTier(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnTierInput) (EarnTierSaved, error) {
	tid, err := uuid.Parse(id)
	if err != nil {
		return EarnTierSaved{}, notFound(ctx, "tier")
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tid, WorkspaceID: ws.ID})
	if err != nil {
		return EarnTierSaved{}, mapErr(ctx, err, "tier")
	}
	lists, err := s.validateTier(ctx, ws, &in, tid)
	if err != nil {
		return EarnTierSaved{}, err
	}
	cur, err := s.tierCurrency(ctx, ws)
	if err != nil {
		return EarnTierSaved{}, err
	}
	prices, err := s.Q.EarnTierPrices(ctx, tid)
	if err != nil {
		return EarnTierSaved{}, err
	}
	now := s.now()
	before := map[string]any{"name": t.Name, "trial_days": t.TrialDays, "visibility": t.Visibility, "access_rank": t.AccessRank, "benefits": len(t.Benefits)}
	replaced := []map[string]any{}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnTierUpdate(ctx, store.EarnTierUpdateParams{ID: tid, WorkspaceID: ws.ID, Name: in.Name, Slug: in.Slug, Description: in.Description,
			Benefits: in.Benefits, TrialDays: int32(in.TrialDays), ListIds: lists, AccessRank: int32(in.AccessRank), Visibility: in.Visibility, At: now}); err != nil {
			return err
		}
		for _, pr := range []struct {
			period string
			amount int64
		}{{"month", in.Monthly}, {"year", in.Yearly}} {
			var current *store.EarnPrice
			for i := range prices {
				if prices[i].Period == pr.period && prices[i].Status == "current" {
					current = &prices[i]
				}
			}
			switch {
			case current != nil && current.AmountCents == pr.amount:
				continue
			case current != nil:
				before[pr.period] = priceText(current.AmountCents, current.Currency, pr.period)
				if err := q.EarnPriceRetire(ctx, store.EarnPriceRetireParams{ID: current.ID, At: &now}); err != nil {
					return err
				}
			}
			if pr.amount <= 0 {
				if current != nil {
					replaced = append(replaced, map[string]any{"period": pr.period, "old": priceText(current.AmountCents, current.Currency, pr.period), "new": nil})
				}
				continue
			}
			if err := q.EarnPriceInsert(ctx, store.EarnPriceInsertParams{ID: newID(), WorkspaceID: ws.ID, TierID: tid, Period: pr.period, AmountCents: pr.amount,
				Currency: cur, Status: "current", Origin: "tier", At: now}); err != nil {
				return err
			}
			r := map[string]any{"period": pr.period, "new": priceText(pr.amount, cur, pr.period)}
			if current != nil {
				r["old"] = priceText(current.AmountCents, current.Currency, pr.period)
			}
			replaced = append(replaced, r)
		}
		if err := s.setHighlight(ctx, q, ws, tid, in.Highlight, false); err != nil {
			return err
		}
		action := "earn.tier_updated"
		if in.Visibility == VisibilityArchived && t.Visibility != VisibilityArchived {
			action = "earn.tier_archived"
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "earn_tier", TargetID: tid.String(), TargetName: in.Name,
			Before: before, After: tierAudit(in, cur)}); err != nil {
			return err
		}
		for _, r := range replaced {
			act := "earn.price_added"
			if r["old"] != nil {
				act = "earn.price_replaced"
			}
			if r["new"] == nil {
				act = "earn.price_removed"
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: act, TargetType: "earn_tier", TargetID: tid.String(), TargetName: in.Name,
				Before: map[string]any{"price": r["old"]}, After: map[string]any{"price": r["new"]}, Metadata: map[string]any{"period": r["period"]}}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return EarnTierSaved{}, err
	}
	return s.afterTierSave(ctx, ws, tid)
}

// setHighlight records the highlighted tier on the upgrade page: one tier
// at a time, with its label.
func (s *Service) setHighlight(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, label string, created bool) error {
	page, err := s.upgradePageTx(ctx, q, ws)
	if err != nil {
		return err
	}
	label = strings.TrimSpace(label)
	switch {
	case label != "":
		page.Highlight, page.HighlightLabel = id.String(), label
	case page.Highlight == id.String():
		page.Highlight, page.HighlightLabel = "", ""
	case created:
		return nil
	default:
		return nil
	}
	b, _ := json.Marshal(page)
	_, err = q.EarnSettingsSetUpgradePageOnly(ctx, store.EarnSettingsSetUpgradePageOnlyParams{WorkspaceID: ws.ID, UpgradePage: b})
	return err
}

// DeleteTier deletes a tier that never had paid members.
func (s *Service) DeleteTier(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) error {
	tid, err := uuid.Parse(id)
	if err != nil {
		return notFound(ctx, "tier")
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tid, WorkspaceID: ws.ID})
	if err != nil {
		return mapErr(ctx, err, "tier")
	}
	if n, err := s.Q.EarnTierSubscriptions(ctx, &tid); err != nil {
		return err
	} else if n > 0 {
		return problem(ctx, http.StatusConflict, "tier_has_members", "tier.hasMembers", i18n.Args{"count": n})
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnTierDelete(ctx, store.EarnTierDeleteParams{ID: tid, WorkspaceID: ws.ID, At: &[]time.Time{s.now()}[0]}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.tier_deleted", TargetType: "earn_tier", TargetID: tid.String(), TargetName: t.Name,
			Before: map[string]any{"name": t.Name, "visibility": t.Visibility}})
	})
}

// SyncTier retries saving a tier in the provider.
func (s *Service) SyncTier(ctx context.Context, ws *store.Workspace, id string) (EarnTierSaved, error) {
	tid, err := uuid.Parse(id)
	if err != nil {
		return EarnTierSaved{}, notFound(ctx, "tier")
	}
	if _, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tid, WorkspaceID: ws.ID}); err != nil {
		return EarnTierSaved{}, mapErr(ctx, err, "tier")
	}
	return s.afterTierSave(ctx, ws, tid)
}

// syncTier saves a tier's product and current prices in every connected
// mode, and stops legacy prices from taking new paid members. It records
// each step; a step that fails (after the client's retries) stops the
// sync and its message is returned. Retrying creates only what's missing.
func (s *Service) syncTier(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return "", err
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return "", err
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return "", err
	}
	modes := []connRow{}
	for _, c := range conns {
		if c.Status == "connected" && c.KeyRejectedAt == nil {
			modes = append(modes, c)
		}
	}
	if len(modes) == 0 {
		return "", s.Q.EarnTierSync(ctx, store.EarnTierSyncParams{ID: id, SyncStatus: "draft", SyncSteps: []byte(`[]`)})
	}
	var steps []EarnSyncStep
	// The steps shown are the active mode's (live once connected).
	act, _ := active(conns)
	fail := ""
	for _, c := range modes {
		client, err := s.open(c.PaymentConnection)
		if err != nil {
			continue
		}
		mine := c.ID == act.ID
		record := func(step EarnSyncStep) {
			if mine {
				steps = append(steps, step)
			}
		}
		product := t.StripeProductLive
		if c.Mode == "test" {
			product = t.StripeProductTest
		}
		meta := map[string]string{"reputecast_tier": t.ID.String(), "reputecast_workspace": ws.Slug}
		pr, err := client.SaveProduct(ctx, payments.Product{ID: product, Name: t.Name, Description: t.Description, Active: true, Metadata: meta})
		now := s.now()
		if err != nil {
			record(EarnSyncStep{Step: "product", Status: "failed", Error: syncError(err), Tries: 3})
			if fail == "" {
				fail = syncError(err)
			}
			continue
		}
		record(EarnSyncStep{Step: "product", Status: "done", At: &now, ID: pr.ID})
		if pr.ID != product {
			if err := s.Q.EarnTierSetProduct(ctx, store.EarnTierSetProductParams{ID: id, Mode: c.Mode, Product: pr.ID}); err != nil {
				return "", err
			}
		}
		prices, err := s.Q.EarnTierPrices(ctx, id)
		if err != nil {
			return "", err
		}
		for _, p := range prices {
			sid := p.StripeLiveID
			if c.Mode == "test" {
				sid = p.StripeTestID
			}
			if p.Status == "legacy" {
				if sid != "" && p.RetiredAt != nil && p.Origin == "tier" {
					_ = client.RetirePrice(ctx, sid)
				}
				continue
			}
			if sid != "" {
				now := s.now()
				record(EarnSyncStep{Step: p.Period, Status: "done", At: &now, ID: sid, AmountCents: p.AmountCents})
				continue
			}
			np, err := client.CreatePrice(ctx, payments.Price{ProductID: pr.ID, AmountCents: p.AmountCents, Currency: p.Currency, Interval: p.Period,
				TaxInclusive: st.PricesIncludeTax, Metadata: map[string]string{"reputecast_tier": t.ID.String()}})
			if err != nil {
				record(EarnSyncStep{Step: p.Period, Status: "failed", AmountCents: p.AmountCents, Error: syncError(err), Tries: 3})
				if fail == "" {
					fail = syncError(err)
				}
				break
			}
			now := s.now()
			record(EarnSyncStep{Step: p.Period, Status: "done", At: &now, ID: np.ID, AmountCents: p.AmountCents})
			if err := s.Q.EarnPriceSetStripe(ctx, store.EarnPriceSetStripeParams{ID: p.ID, Mode: c.Mode, StripeID: np.ID}); err != nil {
				return "", err
			}
		}
	}
	status := "synced"
	var synced *time.Time
	if fail != "" {
		status = "failed"
	} else {
		now := s.now()
		synced = &now
	}
	b, _ := json.Marshal(steps)
	if steps == nil {
		b = []byte(`[]`)
	}
	if err := s.Q.EarnTierSync(ctx, store.EarnTierSyncParams{ID: id, SyncStatus: status, SyncSteps: b, SyncedAt: synced}); err != nil {
		return "", err
	}
	return fail, nil
}

func syncError(err error) string {
	var ae *payments.APIError
	if errors.As(err, &ae) {
		return fmt.Sprintf("%d %s", ae.Status, http.StatusText(ae.Status))
	}
	if errors.Is(err, payments.ErrUnavailable) {
		return syncNoAnswer
	}
	return err.Error()
}

// syncNoAnswer is the stored error of a provider that didn't answer.
const syncNoAnswer = "no answer"

// syncErrorIn is a stored sync error as a reader reads it: the provider's
// answer as it came ("402 Payment Required"), and no answer in the
// reader's language.
func syncErrorIn(l i18n.Lang, stored string) string {
	if stored == syncNoAnswer {
		return msgs.M("sync.noAnswer").Text(l, nil)
	}
	return stored
}
